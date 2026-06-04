package com.steadyphoto.sync.data.remote.dto

import com.google.gson.annotations.SerializedName

/**
 * Response from the server after uploading media files.
 */
data class UploadResponse(
    val uploaded: List<UploadedMedia> = emptyList(),
    
    @SerializedName("skipped_duplicates")
    val skippedDuplicates: List<SkippedDuplicateItem> = emptyList(),
    
    val errors: List<UploadError> = emptyList()
)

/**
 * Skipped duplicate item from the server response.
 */
data class SkippedDuplicateItem(
    @SerializedName("filename")
    val filename: String,
    
    val id: String
)

/**
 * Individual upload error from the server response.
 */
data class UploadError(
    @SerializedName("filename")
    val filename: String? = null,
    
    val message: String
)

/**
 * Media item returned by the server after successful upload.
 */
data class UploadedMedia(
    val id: String,
    @SerializedName("filename")
    val filename: String,
    val mediaType: String,
    val path: String,
    val size: Long,
    
    @SerializedName("captured_at")
    val capturedAt: String? = null
)

/**
 * Response from GET /api/v1/media (sync status).
 */
data class SyncStatusResponse(
    val items: List<UploadedMedia> = emptyList(),
    val total: Int = 0
)

/**
 * Response from DELETE /api/v1/media/delete.
 */
data class DeleteResponse(
    val success: Boolean,
    
    @SerializedName("message")
    val message: String? = null
)
