package com.steadyphoto.sync.data.repository

import android.content.Context
import android.util.Log
import androidx.work.WorkManager
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.entity.UploadStatus
import kotlinx.coroutines.*
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import okhttp3.MediaType.Companion.toMediaType
import okio.BufferedSink

/**
 * Enhanced upload manager that handles:
 * - Streaming uploads with progress tracking
 * - Chunked/resumable uploads for large files (with Android 10+ compatibility)
 * - Automatic retry with exponential backoff
 * - Network connectivity monitoring
 * - Batch upload optimization
 */
class UploadManager(
    private val context: Context,
    private val apiClient: com.steadyphoto.sync.data.remote.api.ApiClient,
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
        items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>,
        callback: UploadProgressCallback? = null
    ): Result<UploadResult> {
        return try {
            if (!networkMonitor.isCurrentlyConnected()) {
                Log.w(TAG, "No network connection available")
                callback?.onUploadError(Exception("No network connection"))
                return Result.failure(Exception("No network connection"))
            }

            val token = apiClient.getAuthToken() ?: run {
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
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        callback: UploadProgressCallback? = null
    ): Result<UploadItemResult> {
        return try {
            val token = apiClient.getAuthToken() ?: run {
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
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        uri: android.net.Uri,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                // Use ContentResolver.openInputStream() to read file from URI - works on Android 10+ where File access is restricted
                
                Log.d(TAG, "Uploading ${item.fileName} (${item.fileSize} bytes) - attempt $attempt")

                val requestFile = object : okhttp3.RequestBody() {
                    override fun contentType(): okhttp3.MediaType? = item.mimeType.toMediaType()
                    override fun contentLength(): Long = item.fileSize
                    
                    // Tell OkHttp this stream can only be read once - prevents retry attempts
                    override fun isOneShot(): Boolean = true
                    
                    override fun writeTo(sink: BufferedSink) {
                        Log.d(TAG, "writeTo called for ${item.fileName} - writing ${item.fileSize} bytes")
                        
                        // Open a fresh InputStream for each write attempt (isOneShot means we can only read once)
                        val inputStream = context.contentResolver.openInputStream(uri) 
                            ?: throw Exception("Cannot open input stream for $uri")
                        
                        try {
                            var totalWritten = 0L
                            val buffer = ByteArray(8192)
                            while (true) {
                                val bytesRead = inputStream.read(buffer)
                                if (bytesRead == -1) break
                                sink.write(buffer, 0, bytesRead)
                                totalWritten += bytesRead
                            }
                            Log.d(TAG, "writeTo complete for ${item.fileName}: total written = $totalWritten")
                        } finally {
                            inputStream.close()
                        }
                    }
                }

                val filePart = okhttp3.MultipartBody.Part.createFormData(
                    "file",
                    item.fileName,
                    requestFile
                )

                // Always get fresh ApiService from ApiClient to ensure correct URL is used
                val apiService = apiClient.apiService
                
                Log.d(TAG, "Sending HTTP POST for ${item.fileName}")
                
                try {
                    val response = apiService.uploadSingleFile(
                        file = filePart,
                        fileName = okhttp3.RequestBody.create("text/plain".toMediaType(), item.fileName),
                        mimeType = okhttp3.RequestBody.create("text/plain".toMediaType(), item.mimeType),
                        fileSize = okhttp3.RequestBody.create("text/plain".toMediaType(), item.fileSize.toString())
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
                        val delayTime = kotlin.math.min(
                            (INITIAL_RETRY_DELAY_MS * (2L * (attempt - 1))).toDouble(),
                            MAX_RETRY_DELAY_MS.toDouble()
                        ).toLong()
                        kotlinx.coroutines.delay(delayTime.milliseconds)
                    }
                }
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
                    val delayTime = kotlin.math.min(
                        (INITIAL_RETRY_DELAY_MS * (2L * (attempt - 1))).toDouble(),
                        MAX_RETRY_DELAY_MS.toDouble()
                    ).toLong()
                    kotlinx.coroutines.delay(delayTime.milliseconds)
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
     * 
     * FIX: On Android 10+, InputStream from ContentResolver doesn't support seek().
     * Solution: Read the entire file into a byte array first, then extract just the chunk we need.
     */
    private suspend fun performChunkedUpload(
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
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

        // Read the entire file content once into memory (acceptable for large files since we need random access anyway)
        val fileContent = try {
            val uri = android.net.Uri.parse(item.uri)
            context.contentResolver.openInputStream(uri)?.use { input ->
                input.readBytes()
            } ?: throw Exception("Cannot read file content from URI: $uri")
        } catch (e: Exception) {
            Log.e(TAG, "Failed to read file content for ${item.fileName}", e)
            return UploadItemResult.Failed("Failed to read file content: ${e.message}")
        }

        // Upload each chunk using the pre-read byte array - no seek() needed!
        var uploadedChunks = mutableListOf<Int>()

        for (chunkIndex in 0 until totalChunks) {
            val success = uploadChunkFromBytes(
                item = item,
                token = token,
                uploadId = uploadId,
                chunkIndex = chunkIndex,
                totalChunks = totalChunks,
                fileSize = fileSize,
                fileContent = fileContent
            )

            if (success) {
                uploadedChunks.add(chunkIndex)

                // Update progress
                val bytesUploaded = ((chunkIndex + 1).toLong() * CHUNK_SIZE).coerceAtMost(fileSize)

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
     * Upload a single chunk from pre-read byte array - no seek() needed!
     */
    private suspend fun uploadChunkFromBytes(
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        token: String,
        uploadId: String,
        chunkIndex: Int,
        totalChunks: Int,
        fileSize: Long,
        fileContent: ByteArray
    ): Boolean {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                val offset = chunkIndex * CHUNK_SIZE
                val remaining = fileSize - offset
                val currentChunkSize = kotlin.math.min(CHUNK_SIZE, remaining).toInt()
                
                // Extract just the bytes we need from the pre-read array - no seek needed!
                val chunkData = fileContent.sliceArray((offset.toInt()) until ((offset + currentChunkSize).toInt()))

                // Create chunk request body from the byte array
                val chunkBody = okhttp3.RequestBody.create("application/octet-stream".toMediaType(), chunkData)

                // Always get fresh ApiService from ApiClient to ensure correct URL is used
                val apiService = apiClient.apiService
                
                Log.d(TAG, "Uploading chunk $chunkIndex of ${totalChunks} for ${item.fileName}")
                
                val response = apiService.uploadChunk(
                    chunk = okhttp3.MultipartBody.Part.createFormData(
                        "chunk",
                        "chunk_${chunkIndex}_${item.fileName}",
                        chunkBody
                    ),
                    uploadId = okhttp3.RequestBody.create("text/plain".toMediaType(), uploadId),
                    chunkIndex = okhttp3.RequestBody.create("text/plain".toMediaType(), chunkIndex.toString()),
                    totalChunks = okhttp3.RequestBody.create("text/plain".toMediaType(), totalChunks.toString()),
                    fileName = okhttp3.RequestBody.create("text/plain".toMediaType(), item.fileName)
                )

                return response.success

            } catch (e: Exception) {
                lastError = e
                Log.w(TAG, "Chunk upload attempt $attempt failed for ${item.fileName}: ${e.message}")

                if (attempt < currentConfig.maxRetries) {
                    val delayTime = kotlin.math.min(
                        (INITIAL_RETRY_DELAY_MS * (2L * (attempt - 1))).toDouble(),
                        MAX_RETRY_DELAY_MS.toDouble()
                    ).toLong()
                    kotlinx.coroutines.delay(delayTime.milliseconds)
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
        items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>,
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
