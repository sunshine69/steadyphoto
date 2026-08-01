package com.steadyphoto.sync.data.repository

import android.content.Context
import android.util.Log
import androidx.work.WorkManager
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.data.settings.SettingsRepository
import kotlinx.coroutines.*
import kotlinx.coroutines.flow.first
import kotlin.time.Duration.Companion.milliseconds
import kotlin.time.Duration.Companion.seconds
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.asStateFlow
import kotlinx.coroutines.flow.update
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
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
    val networkMonitor: NetworkConnectivityMonitor, // Public for access by workers
    private val settingsRepository: com.steadyphoto.sync.data.settings.SettingsRepository = SettingsRepository(context)
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

            // Check current network settings against user preferences (WiFi-only, metered)
            val settings = settingsRepository.networkSettingsFlow.first()
            if (!networkMonitor.isNetworkAcceptableForUpload(settings)) {
                val currentType = networkMonitor.getCurrentNetworkType()?.name ?: "unknown"
                Log.w(TAG, "Current network ($currentType) doesn't meet preferences - skipping upload")
                callback?.onUploadError(Exception("Network type not acceptable: $currentType"))
                return Result.failure(Exception("Network type not acceptable for current settings"))
            }

            val token = apiClient.getAuthToken() ?: run {
                val error = Exception("No auth token available")
                callback?.onUploadError(error)
                return Result.failure(error)
            }

            // Filter out items that don't have a valid URI
            val validItems = items.filter { 
                it.uri.isNotEmpty() && !it.hash.startsWith("hash_failed_") && !it.hash.startsWith("no_path_")
            }

            if (validItems.isEmpty()) {
                Log.w(TAG, "No valid items to upload")
                return Result.success(UploadResult(0, 0))
            }

            // Initialize progress tracking
            _uploadProgress.value = UploadSessionProgress(
                totalItems = validItems.size,
                uploadedItems = 0,
                failedItems = 0,
                currentUpload = null
            )

            callback?.onProgressUpdated(_uploadProgress.value!!)

            var successCount = 0
            var failureCount = 0

            // Upload items in batches
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
                    val serverId = result.mediaId
                    if (serverId != null && serverId.isNotEmpty() && !serverId.startsWith("uploaded_")) {
                        mediaItemDao.updateStatusWithServerId(item.id, UploadStatus.UPLOADED, serverId)
                    } else {
                        mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
                    }
                }
                is UploadItemResult.Failed -> {
                    mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, result.message)
                }
            }

            Result.success(result)

        } catch (e: Exception) {
            Log.e(TAG, "Single item upload failed for ${item.fileName}", e)
            mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, "Upload error: ${e.message}")
            Result.failure(e)
        }
    }

    private suspend fun performSingleFileUpload(
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        uri: android.net.Uri,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                Log.d(TAG, "Uploading ${item.fileName} (${item.fileSize} bytes) - attempt $attempt")

                val requestFile = object : okhttp3.RequestBody() {
                    override fun contentType(): okhttp3.MediaType? = item.mimeType.toMediaType()
                    override fun contentLength(): Long = item.fileSize
                    override fun isOneShot(): Boolean = true
                    
                    override fun writeTo(sink: BufferedSink) {
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
                        } finally {
                            inputStream.close()
                        }
                    }
                }

                val filePart = okhttp3.MultipartBody.Part.createFormData("file", item.fileName, requestFile)
                val apiService = apiClient.apiService
                
                // Convert milliseconds to seconds for the Go server (time.Unix expects seconds)
                val fileCreatedAtSeconds = item.fileCreatedAt?.let { (it / 1000).toString() }?.toRequestBody("text/plain".toMediaType())
                val captureTimeSeconds = item.captureTime?.let { (it / 1000).toString() }?.toRequestBody("text/plain".toMediaType())
                
                val response = apiService.uploadSingleFile(
                    file = filePart,
                    fileName = item.fileName.toRequestBody("text/plain".toMediaType()),
                    mimeType = item.mimeType.toRequestBody("text/plain".toMediaType()),
                    fileSize = item.fileSize.toString().toRequestBody("text/plain".toMediaType()),
                    fileCreatedAt = fileCreatedAtSeconds,
                    captureTime = captureTimeSeconds
                )

                // Check if the server reported this as a duplicate
                if (response.skippedDuplicates.isNotEmpty()) {
                    val skipped = response.skippedDuplicates[0]
                    Log.d(TAG, "Server reported '${item.fileName}' as duplicate (ID=${skipped.id})")
                    
                    // Update local status with server ID
                    mediaItemDao.updateStatusWithServerId(item.id, UploadStatus.SKIPPED_DUPLICATE, skipped.id)

                    _uploadProgress.update { current ->
                        current?.copy(
                            currentUpload = UploadProgress(
                                itemId = item.id,
                                fileName = item.fileName,
                                bytesUploaded = item.fileSize,
                                totalBytes = item.fileSize,
                                status = UploadStatus.SKIPPED_DUPLICATE,
                                serverId = skipped.id
                            )
                        )
                    }

                    callback?.onProgressUpdated(_uploadProgress.value!!)

                    // Compare timestamps and update if they differ
                    val needsTimestampUpdate = shouldUpdateTimestamps(item, skipped)
                    if (needsTimestampUpdate) {
                        try {
                            apiService.updateMediaTimestamps(skipped.id, 
                                com.steadyphoto.sync.data.remote.dto.UpdateTimestampsRequest(
                                    capturedAt = formatTimestampForApi(item.captureTime),
                                    fileCreatedAt = formatTimestampForApi(item.fileCreatedAt)
                                )
                            )
                            Log.d(TAG, "Updated timestamps for duplicate '${item.fileName}' (ID=${skipped.id})")
                        } catch (e: Exception) {
                            Log.w(TAG, "Failed to update timestamps for duplicate '${item.fileName}': ${e.message}")
                        }
                    } else {
                        Log.d(TAG, "Timestamps already match for duplicate '${item.fileName}' (ID=${skipped.id})")
                    }

                    return UploadItemResult.Success(mediaId = skipped.id)
                }

                _uploadProgress.update { current ->
                    current?.copy(
                        currentUpload = UploadProgress(
                            itemId = item.id,
                            fileName = item.fileName,
                            bytesUploaded = item.fileSize,
                            totalBytes = item.fileSize,
                            status = UploadStatus.UPLOADED,
                            serverId = response.uploaded.firstOrNull()?.id
                        )
                    )
                }

                return UploadItemResult.Success(mediaId = response.uploaded.firstOrNull()?.id)

            } catch (e: Exception) {
                lastError = e
                Log.w(TAG, "Upload attempt $attempt failed for ${item.fileName}: ${e.message}")
                if (attempt < currentConfig.maxRetries) {
                    delay(INITIAL_RETRY_DELAY_MS * attempt)
                }
            }
        }

        return UploadItemResult.Failed(message = lastError?.message ?: "Unknown error")
    }

    /**
     * Perform chunked upload for large files.
     * FIXED: No longer reads entire file into memory. Reads chunks sequentially from InputStream.
     */
    /**
     * Compare client timestamps with server timestamps from duplicate response.
     * Returns true if timestamps differ and need updating.
     * Client timestamps are Long (milliseconds since epoch), server timestamps are RFC3339 strings.
     */
    private fun shouldUpdateTimestamps(item: com.steadyphoto.sync.data.local.entity.MediaItemEntity, skipped: com.steadyphoto.sync.data.remote.dto.SkippedDuplicateItem): Boolean {
        // If client has timestamp info that server lacks, we need to update
        if (item.captureTime != null && skipped.capturedAt == null) {
            return true
        }
        if (item.fileCreatedAt != null && skipped.fileCreatedAt == null) {
            return true
        }
        
        // If both have timestamp info but they differ, we need to update
        // Convert client milliseconds to RFC3339 for comparison
        if (item.captureTime != null && skipped.capturedAt != null) {
            try {
                val clientTimestamp = java.time.Instant.ofEpochMilli(item.captureTime!!)
                val clientRfc3339 = java.time.format.DateTimeFormatter.ISO_INSTANT.format(clientTimestamp)
                if (clientRfc3339 != skipped.capturedAt) {
                    return true
                }
            } catch (e: Exception) {
                Log.w(TAG, "Failed to convert capturedAt for comparison: ${e.message}")
            }
        }
        if (item.fileCreatedAt != null && skipped.fileCreatedAt != null) {
            try {
                val clientTimestamp = java.time.Instant.ofEpochMilli(item.fileCreatedAt!!)
                val clientRfc3339 = java.time.format.DateTimeFormatter.ISO_INSTANT.format(clientTimestamp)
                if (clientRfc3339 != skipped.fileCreatedAt) {
                    return true
                }
            } catch (e: Exception) {
                Log.w(TAG, "Failed to convert fileCreatedAt for comparison: ${e.message}")
            }
        }
        
        return false
    }
    
    /**
     * Convert client timestamp (Long, milliseconds since epoch) to RFC3339 string for API request.
     */
    private fun formatTimestampForApi(timestampMillis: Long?): String? {
        return timestampMillis?.let { millis ->
            try {
                // Convert milliseconds to seconds (epoch seconds) for the Go server
                (millis / 1000).toString()
            } catch (e: Exception) {
                Log.w(TAG, "Failed to format timestamp for API: ${e.message}")
                null
            }
        }
    }
    private suspend fun performChunkedUpload(
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        token: String,
        callback: UploadProgressCallback? = null
    ): UploadItemResult {
        val fileSize = item.fileSize
        val uploadId = "${item.id}_${System.currentTimeMillis()}"
        val totalChunks = ((fileSize + CHUNK_SIZE - 1) / CHUNK_SIZE).toInt()

        Log.d(TAG, "Starting memory-safe chunked upload for ${item.fileName}: $totalChunks chunks")

        for (chunkIndex in 0 until totalChunks) {
            val success = uploadChunkSequentially(item, uploadId, chunkIndex, totalChunks)
            if (!success) {
                return UploadItemResult.Failed(message = "Failed to upload chunk $chunkIndex")
            }

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
        }

        // All chunks uploaded, now complete the upload (assembles file, checks duplicates)
        val completeResult = completeUpload(item, uploadId, token)
        
        if (completeResult != null) {
            // Check if the server reported this as a duplicate
            if (completeResult.skippedDuplicates.isNotEmpty()) {
                val skipped = completeResult.skippedDuplicates[0]
                Log.d(TAG, "Server reported '${item.fileName}' as duplicate during chunked upload (ID=${skipped.id})")
                mediaItemDao.updateStatusWithServerId(item.id, UploadStatus.SKIPPED_DUPLICATE, skipped.id)

                _uploadProgress.update { current ->
                    current?.copy(
                        currentUpload = UploadProgress(
                            itemId = item.id,
                            fileName = item.fileName,
                            bytesUploaded = item.fileSize,
                            totalBytes = item.fileSize,
                            status = UploadStatus.SKIPPED_DUPLICATE,
                            serverId = skipped.id
                        )
                    )
                }

                callback?.onProgressUpdated(_uploadProgress.value!!)

                // Compare timestamps and update if they differ
                val needsTimestampUpdate = shouldUpdateTimestamps(item, skipped)
                if (needsTimestampUpdate) {
                    try {
                        val apiService = apiClient.apiService
                        apiService.updateMediaTimestamps(skipped.id, 
                            com.steadyphoto.sync.data.remote.dto.UpdateTimestampsRequest(
                                capturedAt = formatTimestampForApi(item.captureTime),
                                fileCreatedAt = formatTimestampForApi(item.fileCreatedAt)
                            )
                        )
                        Log.d(TAG, "Updated timestamps for duplicate during chunked upload '${item.fileName}' (ID=${skipped.id})")
                    } catch (e: Exception) {
                        Log.w(TAG, "Failed to update timestamps for duplicate during chunked upload '${item.fileName}': ${e.message}")
                    }
                } else {
                    Log.d(TAG, "Timestamps already match for duplicate during chunked upload '${item.fileName}' (ID=${skipped.id})")
                }

                return UploadItemResult.Success(mediaId = skipped.id)
            }

            // Successfully uploaded
            _uploadProgress.update { current ->
                current?.copy(
                    currentUpload = UploadProgress(
                        itemId = item.id,
                        fileName = item.fileName,
                        bytesUploaded = item.fileSize,
                        totalBytes = item.fileSize,
                        status = UploadStatus.UPLOADED,
                        serverId = completeResult.uploaded.firstOrNull()?.id
                    )
                )
            }

            callback?.onProgressUpdated(_uploadProgress.value!!)

            return UploadItemResult.Success(mediaId = completeResult.uploaded.firstOrNull()?.id)
        }

        // Complete upload returned null (error), abort the session
        try {
            apiClient.apiService.abortUpload(
                uploadId = uploadId.toRequestBody("text/plain".toMediaType())
            )
        } catch (e: Exception) {
            Log.w(TAG, "Failed to abort upload session for '${item.fileName}': ${e.message}")
        }
        
        return UploadItemResult.Failed(message = "Failed to complete upload")
    }

    /**
     * Complete a resumable upload session by calling the /complete endpoint.
     */
    private suspend fun completeUpload(
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        uploadId: String,
        token: String
    ): com.steadyphoto.sync.data.remote.dto.CompleteUploadResponse? {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                Log.d(TAG, "Completing upload session $uploadId for ${item.fileName} - attempt $attempt")
                
                val apiService = apiClient.apiService
                val response = apiService.completeUpload(
                    uploadId = uploadId.toRequestBody("text/plain".toMediaType())
                )

                if (response.success) {
                    Log.d(TAG, "Upload session $uploadId completed successfully")
                    return response
                }

                lastError = Exception("Server returned failure for complete request")

            } catch (e: Exception) {
                lastError = e
                Log.w(TAG, "Complete upload attempt $attempt failed: ${e.message}")
                if (attempt < currentConfig.maxRetries) {
                    delay(INITIAL_RETRY_DELAY_MS * attempt)
                }
            }
        }

        Log.e(TAG, "Complete upload failed after all retries: ${lastError?.message}")
        return null
    }

    /**
     * Upload a single chunk by reading only that part of the file from an InputStream.
     */
    private suspend fun uploadChunkSequentially(
        item: com.steadyphoto.sync.data.local.entity.MediaItemEntity,
        uploadId: String,
        chunkIndex: Int,
        totalChunks: Int
    ): Boolean {
        var lastError: Exception? = null

        for (attempt in 1..currentConfig.maxRetries) {
            try {
                val uri = android.net.Uri.parse(item.uri)
                val offset = chunkIndex * CHUNK_SIZE
                val remaining = item.fileSize - offset
                val currentChunkSize = kotlin.math.min(CHUNK_SIZE, remaining).toInt()
                
                // Read only the required chunk from the stream
                val chunkData = ByteArray(currentChunkSize)
                context.contentResolver.openInputStream(uri)?.use { input ->
                    // Skip to offset
                    var skipped = 0L
                    while (skipped < offset) {
                        val skipResult = input.skip(offset - skipped)
                        if (skipResult <= 0) break
                        skipped += skipResult
                    }
                    
                    // If skip didn't work (common on some streams), read manually
                    if (skipped < offset) {
                        val skipBuffer = ByteArray(8192)
                        while (skipped < offset) {
                            val toRead = kotlin.math.min(8192L, offset - skipped).toInt()
                            val read = input.read(skipBuffer, 0, toRead)
                            if (read == -1) break
                            skipped += read
                        }
                    }
                    
                    // Now read the actual chunk data
                    var bytesRead = 0
                    while (bytesRead < currentChunkSize) {
                        val read = input.read(chunkData, bytesRead, currentChunkSize - bytesRead)
                        if (read == -1) break
                        bytesRead += read
                    }
                } ?: throw Exception("Could not open input stream")

                val chunkBody = chunkData.toRequestBody("application/octet-stream".toMediaType())
                val apiService = apiClient.apiService
                
                val response = apiService.uploadChunk(
                    chunk = okhttp3.MultipartBody.Part.createFormData("chunk", "chunk_$chunkIndex", chunkBody),
                    uploadId = uploadId.toRequestBody("text/plain".toMediaType()),
                    chunkIndex = chunkIndex.toString().toRequestBody("text/plain".toMediaType()),
                    totalChunks = totalChunks.toString().toRequestBody("text/plain".toMediaType()),
                    fileName = item.fileName.toRequestBody("text/plain".toMediaType())
                )

                if (response.success) return true
                else lastError = Exception("Server returned failure for chunk $chunkIndex")

            } catch (e: Exception) {
                lastError = e
                Log.w(TAG, "Chunk $chunkIndex attempt $attempt failed: ${e.message}")
                if (attempt < currentConfig.maxRetries) delay(INITIAL_RETRY_DELAY_MS * attempt)
            }
        }
        return false
    }

    private suspend fun uploadBatch(
        items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>,
        token: String,
        callback: UploadProgressCallback? = null
    ): BatchResult {
        var successCount = 0
        var failureCount = 0

        // Limit concurrency to prevent battery/CPU drain from too many simultaneous uploads.
        // The maxConcurrentUploads config is respected here (default is 3).
        val concurrency = currentConfig.maxConcurrentUploads.coerceAtLeast(1)
        
        coroutineScope {
            // Process items in batches of maxConcurrentUploads
            for (chunk in items.chunked(concurrency)) {
                val results = chunk.map { item ->
                    async(Dispatchers.IO) {
                        val result = uploadSingleItem(item, callback)
                        synchronized(this@UploadManager) {
                            if (result.isSuccess) successCount++ else failureCount++
                        }
                    }
                }
                results.awaitAll()
            }
        }
        return BatchResult(successCount, failureCount)
    }

    fun scheduleBackgroundUpload() {
        WorkManager.getInstance(context).enqueueUniqueWork(
            "upload_work",
            androidx.work.ExistingWorkPolicy.REPLACE, // Use REPLACE to ensure fresh start
            androidx.work.OneTimeWorkRequest.Builder(com.steadyphoto.sync.worker.UploadWorker::class.java).build()
        )
    }

    fun cancelUploads() {
        WorkManager.getInstance(context).cancelAllWork()
    }

    val applicationContext: android.content.Context
        get() = context
}

data class BatchResult(val success: Int, val failure: Int)

sealed class UploadItemResult {
    data class Success(val mediaId: String?, val retryCount: Int = 0) : UploadItemResult()
    data class Failed(val message: String, val retryCount: Int = 0) : UploadItemResult()
}

data class UploadResult(val successCount: Int, val failureCount: Int) {
    val isSuccess: Boolean = failureCount == 0
}
