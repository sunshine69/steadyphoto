package com.steadyphoto.sync.data.repository

import com.steadyphoto.sync.data.local.entity.UploadStatus

/**
 * Represents upload progress for a single media item.
 */
data class UploadProgress(
    val itemId: Long,
    val fileName: String,
    val bytesUploaded: Long,
    val totalBytes: Long,
    val status: UploadStatus,
    val errorMessage: String? = null,
    val retryCount: Int = 0,
    val chunkIndex: Int? = null, // For chunked uploads
    val totalChunks: Int? = null, // For chunked uploads
    val serverId: String? = null // Server-assigned media ID (set after upload/duplicate detection)
)

/**
 * Represents the overall upload session progress.
 */
data class UploadSessionProgress(
    val totalItems: Int,
    val uploadedItems: Int,
    val failedItems: Int,
    val currentUpload: UploadProgress?,
    val isComplete: Boolean = false,
    val errorMessage: String? = null
) {
    val progressPercentage: Float
        get() = if (totalItems == 0) 0f else uploadedItems.toFloat() / totalItems * 100f
}

/**
 * Callback interface for upload progress updates.
 */
interface UploadProgressCallback {
    suspend fun onProgressUpdated(progress: UploadSessionProgress)
    suspend fun onUploadComplete(successCount: Int, failureCount: Int)
    suspend fun onUploadError(error: Throwable)
}
