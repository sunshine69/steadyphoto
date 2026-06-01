package com.steadyphoto.sync.data.remote.dto

import kotlinx.serialization.Serializable

@Serializable
data class LoginResponse(
    val access_token: String,
    val refresh_token: String,
    val user_id: String,
    val role: String,
    val status: String
)

@Serializable
data class UserDto(
    val id: String,
    val email: String,
    val role: String
)

/**
 * Response from the server after uploading media files.
 */
@Serializable
data class UploadResponse(
    val uploaded: List<UploadedMedia> = emptyList(),
    val skipped_duplicates: List<String> = emptyList(),
    val errors: List<UploadError> = emptyList()
)

/**
 * Individual upload error from the server response.
 */
@Serializable
data class UploadError(
    val filename: String? = null,
    val message: String
)

/**
 * Media item returned by the server after successful upload.
 */
@Serializable
data class UploadedMedia(
    val id: String,
    val filename: String,
    val mediaType: String,
    val path: String,
    val size: Long,
    val captured_at: String? = null
)

/**
 * Response from GET /api/v1/media (sync status).
 */
@Serializable
data class SyncStatusResponse(
    val items: List<UploadedMedia> = emptyList(),
    val total: Int = 0
)

/**
 * Response from DELETE /api/v1/media/delete.
 */
@Serializable
data class DeleteResponse(
    val success: Boolean,
    val message: String? = null
)

/**
 * Response for chunk upload operations.
 */
@Serializable
data class ChunkUploadResponse(
    val success: Boolean,
    val uploadId: String?,
    val chunkIndex: Int,
    val totalChunks: Int,
    val bytesReceived: Long,
    val message: String? = null
)

/**
 * Response for upload session status.
 */
@Serializable
data class UploadSessionResponse(
    val uploadId: String,
    val fileName: String,
    val fileSize: Long,
    val uploadedChunks: List<Int> = emptyList(),
    val totalChunks: Int,
    val isComplete: Boolean,
    val createdAt: String? = null
)

/**
 * Response for aborting an upload session.
 */
@Serializable
data class AbortResponse(
    val success: Boolean,
    val message: String? = null
)
