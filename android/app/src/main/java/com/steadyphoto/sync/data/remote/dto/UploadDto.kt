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
    
    val id: String,
    
    @SerializedName("captured_at")
    val capturedAt: String? = null,
    
    @SerializedName("file_created_at")
    val fileCreatedAt: String? = null
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
    val capturedAt: String? = null,
    
    @SerializedName("file_created_at")
    val fileCreatedAt: String? = null
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

/**
 * Request to update media timestamps.
 */
data class UpdateTimestampsRequest(
    @SerializedName("capturedAt")
    val capturedAt: String? = null,
    
    @SerializedName("fileCreatedAt")
    val fileCreatedAt: String? = null
)

/**
 * Response from timestamps update request.
 */
data class TimestampsUpdateResponse(
    val status: String
)

/**
 * Request to update media tags.
 */
data class UpdateTagsRequest(
    val tags: String
)

/**
 * Response from tags update request.
 */
data class TagsUpdateResponse(
    val status: String
)

/**
 * Request to complete a resumable upload session.
 */
data class CompleteUploadRequest(
    @SerializedName("uploadId")
    val uploadId: String
)

/**
 * Response from the /complete endpoint.
 * This response includes both successful uploads and duplicate detection.
 */
data class CompleteUploadResponse(
    val success: Boolean,
    val uploaded: List<UploadedMedia> = emptyList(),
    @SerializedName("skipped_duplicates")
    val skippedDuplicates: List<SkippedDuplicateItem> = emptyList(),
    val errors: List<UploadError> = emptyList()
)
