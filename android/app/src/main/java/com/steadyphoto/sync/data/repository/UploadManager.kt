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

            // Filter out items that don't have a valid URI (on Android 10+, localPath may be null but we can still upload via URI)
            val validItems = items.filter { 
                it.uri.isNotEmpty() && !it.hash.startsWith("hash_failed_") && !it.hash.startsWith("no_path_")
            }

            if (validItems.isEmpty()) {
                Log.w(TAG, "No valid items to upload - possible scoped storage issue on Android 10+")
                return Result.failure(Exception("No valid files to upload. This may be due to scoped storage restrictions on Android 10+."))
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

            // On Android 10+, use URI-based access via ContentResolver instead of File objects
            val uri = android.net.Uri.parse(item.uri)
            
            // Determine upload strategy based on file size and config
            val result = if (currentConfig.enableChunkedUpload && item.fileSize > CHUNK_SIZE) {
                performChunkedUpload(item, token, callback)
            } else {
                performSingleFileUpload(item, uri, token, callback)
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
     * Perform a single file upload with retry logic using URI-based access.
     */
    private suspend fun performSingleFileUpload(
        item: MediaItemEntity,
        uri: android.net.Uri,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                // Use ContentResolver.openInputStream() to read file from URI - works on Android 10+ where File access is restricted
                val inputStream = context.contentResolver.openInputStream(uri) 
                    ?: throw Exception("Cannot open input stream for $uri")

                // Create multipart part with progress tracking via counting sink
                val requestFile = okhttp3.RequestBody.create(item.mimeType.toMediaType(), inputStream.readBytes())

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
     * Perform chunked upload for large files using URI-based access.
     */
    private suspend fun performChunkedUpload(
        item: MediaItemEntity,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        // Use the file size from the entity instead of File.length() - works on Android 10+
        val fileSize = item.fileSize
        if (fileSize <= 0) {
            return UploadItemResult.Failed("Invalid file size")
        }

        // Generate unique upload ID
        val uploadId = "${item.id}_${System.currentTimeMillis()}"
        val totalChunks = ((fileSize + CHUNK_SIZE - 1) / CHUNK_SIZE).toInt()

        Log.d(TAG, "Starting chunked upload for ${item.fileName}: $totalChunks chunks")

        // Upload each chunk with retry logic using URI-based access
        var uploadedChunks = mutableListOf<Int>()

        for (chunkIndex in 0 until totalChunks) {
            val success = uploadChunkWithRetry(
                item = item,
                token = token,
                uploadId = uploadId,
                chunkIndex = chunkIndex,
                totalChunks = totalChunks,
                fileSize = fileSize
            )

            if (success) {
                uploadedChunks.add(chunkIndex)

                // Update progress
                val bytesUploaded = min(
                    ((chunkIndex + 1).toLong() * CHUNK_SIZE),
                    fileSize
                )

                _uploadProgress.update { current ->
                    current?.copy(
                        currentUpload = UploadProgress(
                            itemId = item.id,
                            fileName = item.fileName,
                            bytesUploaded = bytesUploaded,
                            totalBytes = fileSize,
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
                    bytesUploaded = fileSize,
                    totalBytes = fileSize,
                    status = UploadStatus.UPLOADED
                )
            )
        }

        return UploadItemResult.Success(mediaId = uploadId)
    }

    /**
     * Upload a single chunk with retry logic using URI-based access.
     */
    private suspend fun uploadChunkWithRetry(
        item: MediaItemEntity,
        token: String,
        uploadId: String,
        chunkIndex: Int,
        totalChunks: Int,
        fileSize: Long
    ): Boolean {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                val uri = android.net.Uri.parse(item.uri)
                val offset = chunkIndex * CHUNK_SIZE
                val remaining = fileSize - offset
                val currentChunkSize = min(CHUNK_SIZE, remaining).toInt()

                // Use ContentResolver.openInputStream() to read the URI and skip to the right position for the chunk
                // Note: We can't use seek on InputStream, so we'll read from the beginning up to the end of this chunk
                val inputStream = context.contentResolver.openInputStream(uri) 
                    ?: throw Exception("Cannot open input stream for $uri")

                // Create a byte array for just this chunk by reading and skipping appropriately
                var lastErrorForChunk: Exception? = null
                var chunkData: ByteArray? = null
                
                try {
                    inputStream.use { input ->
                        if (offset > 0) {
                            // skip() doesn't guarantee it will skip all requested bytes, so we loop until offset is reached
                            var skipped = 0L
                            while (skipped < offset) {
                                val n = input.skip(offset - skipped)
                                if (n == 0L && !input.markSupported()) {
                                    // mark not supported and couldn't skip any more bytes - read remaining to discard
                                    val discardBuffer = ByteArray(min(8192, (offset - skipped).toInt()))
                                    while (skipped < offset) {
                                        val bytesReadDiscard = input.read(discardBuffer, 0, min(discardBuffer.size, (offset - skipped).toInt()))
                                        if (bytesReadDiscard <= 0) break
                                        skipped += bytesReadDiscard
                                    }
                                } else if (n > 0L) {
                                    skipped += n
                                } else if (!input.markSupported()) {
                                    // mark not supported and skip returned 0 - need to read to discard
                                    val discardBuffer = ByteArray(min(8192, (offset - skipped).toInt()))
                                    while (skipped < offset) {
                                        val bytesReadDiscard = input.read(discardBuffer, 0, min(discardBuffer.size, (offset - skipped).toInt()))
                                        if (bytesReadDiscard <= 0) break
                                        skipped += bytesReadDiscard
                                    }
                                }
                            }
                        }
                        
                        val buffer = ByteArray(currentChunkSize)
                        val bytesRead = input.read(buffer, 0, currentChunkSize)
                        
                        if (bytesRead < 0) {
                            throw Exception("Failed to read chunk data")
                        }
                        
                        // Trim the buffer if we didn't read all bytes (for last chunk)
                        chunkData = if (bytesRead < currentChunkSize) {
                            buffer.copyOf(bytesRead)
                        } else {
                            buffer
                        }
                    }
                } catch (e: Exception) {
                    lastErrorForChunk = e
                    throw e
                }

                if (chunkData == null || lastErrorForChunk != null) {
                    Log.e(TAG, "Failed to read chunk data for ${item.fileName}")
                    return false
                }

                // Create chunk request body from the byte array
                val chunkBody = okhttp3.RequestBody.create("application/octet-stream".toMediaType(), chunkData!!)

                // Always get fresh ApiService from ApiClient to ensure correct URL is used
                val apiService = apiClient.apiService
                
                val response = apiService.uploadChunk(
                    chunk = MultipartBody.Part.createFormData(
                        "chunk",
                        "chunk_${chunkIndex}_${item.fileName}",
                        chunkBody
                    ),
                    uploadId = uploadId.toRequestBody("text/plain".toMediaType()),
                    chunkIndex = chunkIndex.toString().toRequestBody("text/plain".toMediaType()),
                    totalChunks = totalChunks.toString().toRequestBody("text/plain".toMediaType()),
                    fileName = item.fileName.toRequestBody("text/plain".toMediaType())
                )

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
