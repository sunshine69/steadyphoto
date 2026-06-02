package com.steadyphoto.sync.data.repository

import android.content.Context
import android.util.Log
import androidx.work.WorkManager
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.data.remote.api.ApiClient
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import okio.*
import java.io.File
import kotlin.math.min

/**
 * Enhanced upload manager that handles:
 * - Streaming uploads with progress tracking
 * - Chunked/resumable uploads for large files
 * - Automatic retry with exponential backoff
 * - Network connectivity monitoring
 * - Batch upload optimization
 */
class UploadManager(
    private val context: Context,
    private val apiClient: ApiClient,  // Changed from ApiService to ApiClient
    private val mediaItemDao: MediaItemDao,
    private val networkMonitor: NetworkConnectivityMonitor
) {

    companion object {
        private const val TAG = "UploadManager"
        private const val MAX_RETRIES = 3
        private const val INITIAL_RETRY_DELAY_MS = 1000L // 1 second
        private const val MAX_RETRY_DELAY_MS = 60000L // 1 minute
        private const val CHUNK_SIZE = 5 * 1024 * 1024L // 5MB chunks

        // Upload configuration
        data class UploadConfig(
            val batchSize: Int = 10,
            val maxConcurrentUploads: Int = 3,
            val enableChunkedUpload: Boolean = true,
            val chunkSize: Long = CHUNK_SIZE,
            val enableRetry: Boolean = true,
            val maxRetries: Int = MAX_RETRIES
        )
    }

    private val _uploadProgress = MutableStateFlow<UploadSessionProgress?>(null)
    val uploadProgress: StateFlow<UploadSessionProgress?> = _uploadProgress.asStateFlow()

    private var currentConfig = UploadConfig()

    /**
     * Configure the upload manager with custom settings.
     */
    fun configure(config: UploadConfig) {
        currentConfig = config
        Log.d(TAG, "UploadManager configured: $config")
    }

    /**
     * Upload a list of media items with progress tracking.
     */
    suspend fun uploadMedia(
        items: List<MediaItemEntity>,
        callback: UploadProgressCallback? = null
    ): Result<UploadResult> {
        return try {
            if (!networkMonitor.isCurrentlyConnected()) {
                Log.w(TAG, "No network connection available")
                callback?.onUploadError(Exception("No network connection"))
                return Result.failure(Exception("No network connection"))
            }

            val token = getAuthToken() ?: run {
                val error = Exception("No auth token available")
                callback?.onUploadError(error)
                return Result.failure(error)
            }

            // Filter out items that don't have a valid local path
            val validItems = items.filter { it.localPath != null && File(it.localPath!!).exists() }

            if (validItems.isEmpty()) {
                Log.w(TAG, "No valid items to upload")
                return Result.failure(Exception("No valid files to upload"))
            }

            // Initialize progress tracking
            _uploadProgress.value = UploadSessionProgress(
                totalItems = validItems.size,
                uploadedItems = 0,
                failedItems = 0,
                currentUpload = null
            )

            callback?.onProgressUpdated(_uploadProgress.value!!)

            // Upload items in batches
            var successCount = 0
            var failureCount = 0

            for (chunk in validItems.chunked(currentConfig.batchSize)) {
                val batchResult = uploadBatch(chunk, token, callback)
                successCount += batchResult.success
                failureCount += batchResult.failure

                // Update progress after each batch
                _uploadProgress.update { current ->
                    current?.copy(
                        uploadedItems = successCount,
                        failedItems = failureCount,
                        currentUpload = null
                    )
                }

                callback?.onProgressUpdated(_uploadProgress.value!!)
            }

            // Mark upload as complete
            _uploadProgress.update { current ->
                current?.copy(
                    uploadedItems = successCount,
                    failedItems = failureCount,
                    currentUpload = null,
                    isComplete = true
                )
            }

            callback?.onUploadComplete(successCount, failureCount)

            Log.d(TAG, "Upload complete: $successCount succeeded, $failureCount failed")
            Result.success(UploadResult(successCount, failureCount))

        } catch (e: Exception) {
            Log.e(TAG, "Upload failed", e)
            _uploadProgress.update { current ->
                current?.copy(
                    isComplete = true,
                    errorMessage = e.message
                )
            }
            callback?.onUploadError(e)
            Result.failure(e)
        }
    }

    /**
     * Upload a single item with full progress tracking and retry logic.
     */
    suspend fun uploadSingleItem(
        item: MediaItemEntity,
        callback: UploadProgressCallback? = null
    ): Result<UploadItemResult> {
        return try {
            val token = getAuthToken() ?: run {
                val error = Exception("No auth token")
                mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, "No auth token")
                return Result.failure(error)
            }

            if (!networkMonitor.isCurrentlyConnected()) {
                mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, "No network connection")
                return Result.failure(Exception("No network connection"))
            }

            // Update status to uploading
            mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADING)

            val file = File(item.localPath ?: "")
            if (!file.exists()) {
                mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, "File not found")
                return Result.failure(Exception("File not found: ${item.localPath}"))
            }

            // Determine upload strategy based on file size and config
            val result = if (currentConfig.enableChunkedUpload && file.length() > CHUNK_SIZE) {
                performChunkedUpload(item, token, callback)
            } else {
                performSingleFileUpload(item, token, callback)
            }

            // Update local status based on result
            when (result) {
                is UploadItemResult.Success -> {
                    mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
                }
                is UploadItemResult.Failed -> {
                    if (currentConfig.enableRetry && result.retryCount < currentConfig.maxRetries) {
                        mediaItemDao.updateStatus(
                            item.id,
                            UploadStatus.FAILED,
                            "Upload failed: ${result.message}"
                        )
                    } else {
                        mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, result.message)
                    }
                }
            }

            Result.success(result)

        } catch (e: Exception) {
            Log.e(TAG, "Single item upload failed for ${item.fileName}", e)
            mediaItemDao.updateStatus(
                item.id,
                UploadStatus.FAILED,
                "Upload error: ${e.message}"
            )
            Result.failure(e)
        }
    }

    /**
     * Perform a single file upload with retry logic.
     */
    private suspend fun performSingleFileUpload(
        item: MediaItemEntity,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                val file = File(item.localPath ?: "")

                // Create multipart part with progress tracking via counting sink
                val requestFile = file.asRequestBody(item.mimeType.toMediaType())

                val filePart = MultipartBody.Part.createFormData(
                    "file",
                    item.fileName,
                    object : okhttp3.RequestBody() {
                        override fun contentType() = item.mimeType.toMediaType()
                        override fun contentLength() = requestFile.contentLength()

                        override fun writeTo(sink: BufferedSink) {
                            val totalBytes = requestFile.contentLength()
                            var bytesWritten = 0L

                            // Create a counting sink that wraps the underlying network sink (matching ProgressRequestBody pattern)
                            val countingSink = object : ForwardingSink(sink) {
                                @Throws(java.io.IOException::class)
                                override fun write(source: Buffer, byteCount: Long) {
                                    super.write(source, byteCount)
                                    bytesWritten += byteCount

                                    // Report progress every 100KB or at completion
                                    if (bytesWritten % (1024 * 100L) == 0L || bytesWritten >= totalBytes) {
                                        val progress = UploadProgress(
                                            itemId = item.id,
                                            fileName = item.fileName,
                                            bytesUploaded = bytesWritten,
                                            totalBytes = totalBytes,
                                            status = UploadStatus.UPLOADING,
                                            retryCount = attempt - 1
                                        )

                                        _uploadProgress.update { current ->
                                            current?.copy(currentUpload = progress)
                                        }
                                    }
                                }
                            }

                            // Write the file through our counting sink (matching ProgressRequestBody pattern with .buffer())
                            requestFile.writeTo(countingSink.buffer())
                        }
                    }
                )

                // Always get fresh ApiService from ApiClient to ensure correct URL is used
                val apiService = apiClient.apiService
                
                val response = apiService.uploadSingleFile(
                    authHeader = "Bearer $token",
                    file = filePart,
                    fileName = item.fileName.toRequestBody("text/plain".toMediaType()),
                    mimeType = item.mimeType.toRequestBody("text/plain".toMediaType()),
                    fileSize = item.fileSize.toString().toRequestBody("text/plain".toMediaType())
                )

                // Update progress to complete
                _uploadProgress.update { current ->
                    current?.copy(
                        currentUpload = UploadProgress(
                            itemId = item.id,
                            fileName = item.fileName,
                            bytesUploaded = item.fileSize,
                            totalBytes = item.fileSize,
                            status = UploadStatus.UPLOADED
                        )
                    )
                }

                return UploadItemResult.Success(
                    mediaId = response.uploaded.firstOrNull()?.id ?: "",
                    retryCount = attempt - 1
                )

            } catch (e: Exception) {
                lastError = e
                Log.w(TAG, "Upload attempt $attempt failed for ${item.fileName}: ${e.message}")

                // Update progress with error
                _uploadProgress.update { current ->
                    current?.copy(
                        currentUpload = UploadProgress(
                            itemId = item.id,
                            fileName = item.fileName,
                            bytesUploaded = 0L,
                            totalBytes = item.fileSize,
                            status = UploadStatus.FAILED,
                            errorMessage = e.message,
                            retryCount = attempt - 1
                        )
                    )
                }

                if (attempt < currentConfig.maxRetries) {
                    // Exponential backoff
                    val delayTime = min(
                        INITIAL_RETRY_DELAY_MS * (2L * (attempt - 1)),
                        MAX_RETRY_DELAY_MS
                    )
                    kotlinx.coroutines.delay(delayTime)
                }
            }
        }

        return UploadItemResult.Failed(
            message = lastError?.message ?: "Unknown error",
            retryCount = currentConfig.maxRetries
        )
    }

    /**
     * Perform chunked upload for large files.
     */
    private suspend fun performChunkedUpload(
        item: MediaItemEntity,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        val file = File(item.localPath ?: "")
        if (!file.exists()) {
            return UploadItemResult.Failed("File not found")
        }

        // Generate unique upload ID
        val uploadId = "${item.id}_${System.currentTimeMillis()}"
        val totalChunks = ((file.length() + CHUNK_SIZE - 1) / CHUNK_SIZE).toInt()

        Log.d(TAG, "Starting chunked upload for ${item.fileName}: $totalChunks chunks")

        // Upload each chunk with retry logic
        var uploadedChunks = mutableListOf<Int>()

        for (chunkIndex in 0 until totalChunks) {
            val success = uploadChunkWithRetry(
                item = item,
                token = token,
                uploadId = uploadId,
                chunkIndex = chunkIndex,
                totalChunks = totalChunks,
                file = file
            )

            if (success) {
                uploadedChunks.add(chunkIndex)

                // Update progress
                val bytesUploaded = min(
                    ((chunkIndex + 1).toLong() * CHUNK_SIZE),
                    file.length()
                )

                _uploadProgress.update { current ->
                    current?.copy(
                        currentUpload = UploadProgress(
                            itemId = item.id,
                            fileName = item.fileName,
                            bytesUploaded = bytesUploaded,
                            totalBytes = file.length(),
                            status = UploadStatus.UPLOADING,
                            chunkIndex = chunkIndex,
                            totalChunks = totalChunks
                        )
                    )
                }
            } else {
                return UploadItemResult.Failed(
                    message = "Failed to upload chunk $chunkIndex",
                    retryCount = currentConfig.maxRetries
                )
            }
        }

        // All chunks uploaded successfully
        Log.d(TAG, "All chunks uploaded for ${item.fileName}")

        _uploadProgress.update { current ->
            current?.copy(
                currentUpload = UploadProgress(
                    itemId = item.id,
                    fileName = item.fileName,
                    bytesUploaded = file.length(),
                    totalBytes = file.length(),
                    status = UploadStatus.UPLOADED
                )
            )
        }

        return UploadItemResult.Success(mediaId = uploadId)
    }

    /**
     * Upload a single chunk with retry logic.
     */
    private suspend fun uploadChunkWithRetry(
        item: MediaItemEntity,
        token: String,
        uploadId: String,
        chunkIndex: Int,
        totalChunks: Int,
        file: File
    ): Boolean {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                val offset = chunkIndex * CHUNK_SIZE
                val remaining = file.length() - offset
                val currentChunkSize = min(CHUNK_SIZE, remaining).toInt()

                // Create chunk request body
                val chunkFile = createTempChunk(file, offset, currentChunkSize)

                if (chunkFile == null) {
                    Log.e(TAG, "Failed to create temp chunk for ${item.fileName}")
                    return false
                }

                // Always get fresh ApiService from ApiClient to ensure correct URL is used
                val apiService = apiClient.apiService
                
                val response = apiService.uploadChunk(
                    authHeader = "Bearer $token",
                    chunk = MultipartBody.Part.createFormData(
                        "chunk",
                        "chunk_${chunkIndex}_${item.fileName}",
                        chunkFile.asRequestBody("application/octet-stream".toMediaType())
                    ),
                    uploadId = uploadId.toRequestBody("text/plain".toMediaType()),
                    chunkIndex = chunkIndex.toString().toRequestBody("text/plain".toMediaType()),
                    totalChunks = totalChunks.toString().toRequestBody("text/plain".toMediaType()),
                    fileName = item.fileName.toRequestBody("text/plain".toMediaType())
                )

                // Clean up temp file
                chunkFile.delete()

                return response.success

            } catch (e: Exception) {
                lastError = e
                Log.w(TAG, "Chunk upload attempt $attempt failed for ${item.fileName}: ${e.message}")

                if (attempt < currentConfig.maxRetries) {
                    val delayTime = min(
                        INITIAL_RETRY_DELAY_MS * (2L * (attempt - 1)),
                        MAX_RETRY_DELAY_MS
                    )
                    kotlinx.coroutines.delay(delayTime)
                }
            }
        }

        Log.e(TAG, "Chunk upload failed after ${currentConfig.maxRetries} attempts: ${lastError?.message}")
        return false
    }

    /**
     * Create a temporary file containing just the chunk data.
     */
    private fun createTempChunk(file: File, offset: Long, size: Int): File? {
        return try {
            val tempFile = File.createTempFile("chunk_", "_${file.name}", context.cacheDir)

            file.inputStream().use { input ->
                tempFile.outputStream().use { output ->
                    input.skip(offset)

                    val buffer = ByteArray(min(size, 8192))
                    var bytesRead: Int = -1
                    var totalBytesWritten = 0L

                    while (totalBytesWritten < size && input.read(buffer).also { bytesRead = it } != -1) {
                        val toWrite = min(bytesRead.toLong(), size - totalBytesWritten).toInt()
                        output.write(buffer, 0, toWrite)
                        totalBytesWritten += toWrite
                    }

                    output.flush()
                }
            }

            tempFile
        } catch (e: Exception) {
            Log.e(TAG, "Failed to create temp chunk file", e)
            null
        }
    }

    /**
     * Upload a batch of items using concurrent uploads.
     */
    private suspend fun uploadBatch(
        items: List<MediaItemEntity>,
        token: String,
        callback: UploadProgressCallback? = null
    ): BatchResult {
        return try {
            var successCount = 0
            var failureCount = 0

            // Upload items concurrently with limited parallelism
            coroutineScope {
                items.map { item ->
                    async(Dispatchers.IO) {
                        val result = uploadSingleItem(item, callback)
                        if (result.isSuccess) {
                            synchronized(this@UploadManager) { 
                                successCount++
                            }
                        } else {
                            synchronized(this@UploadManager) {
                                failureCount++
                            }
                        }
                    }
                }.awaitAll()
            }

            BatchResult(successCount, failureCount)

        } catch (e: Exception) {
            Log.e(TAG, "Batch upload failed", e)
            callback?.onUploadError(e)
            BatchResult(0, items.size)
        }
    }

    /**
     * Trigger the background upload worker.
     */
    fun scheduleBackgroundUpload() {
        androidx.work.WorkManager.getInstance(context).enqueueUniqueWork(
            "upload_work",
            androidx.work.ExistingWorkPolicy.KEEP,
            androidx.work.OneTimeWorkRequest.Builder(com.steadyphoto.sync.worker.UploadWorker::class.java).build()
        )
        Log.d(TAG, "Background upload scheduled")
    }

    /**
     * Cancel all pending uploads.
     */
    fun cancelUploads() {
        WorkManager.getInstance(context).cancelAllWork()
        _uploadProgress.update { current ->
            current?.copy(
                totalItems = 0,
                uploadedItems = 0,
                failedItems = 0,
                currentUpload = null,
                isComplete = true,
                errorMessage = "Upload cancelled by user"
            )
        }
        Log.d(TAG, "All uploads cancelled")
    }

    /**
     * Get the auth token from shared preferences.
     */
    fun getAuthToken(): String? {
        val prefs = context.getSharedPreferences("app_prefs", Context.MODE_PRIVATE)
        return prefs.getString("auth_token", null)
    }

    /**
     * Get the application context for WorkManager operations.
     */
    val applicationContext: android.content.Context
        get() = context
}

/**
 * Result of a batch upload operation.
 */
data class BatchResult(
    val success: Int,
    val failure: Int
)

/**
 * Result of an individual item upload.
 */
sealed class UploadItemResult {
    data class Success(val mediaId: String, val retryCount: Int = 0) : UploadItemResult()
    data class Failed(val message: String, val retryCount: Int = 0) : UploadItemResult()
}

/**
 * Result of the overall upload session.
 */
data class UploadResult(
    val successCount: Int,
    val failureCount: Int
) {
    val isSuccess: Boolean = failureCount == 0
}
