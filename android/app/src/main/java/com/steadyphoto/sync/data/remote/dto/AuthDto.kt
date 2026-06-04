package com.steadyphoto.sync.data.remote.dto

import com.google.gson.annotations.SerializedName

/**
 * Response from login endpoint - uses Gson @SerializedName for snake_case mapping.
 */
data class LoginResponse(
    @SerializedName("access_token")
    val accessToken: String,
    
    @SerializedName("refresh_token")
    val refreshToken: String,
    
    @SerializedName("user_id")
    val userId: String,
    
    val role: String,
    
    val status: String
)

/**
 * User data DTO - uses Gson @SerializedName for snake_case mapping.
 */
data class UserDto(
    val id: String,
    val email: String,
    val role: String
)

/**
 * Response for chunk upload operations.
 */
data class ChunkUploadResponse(
    val success: Boolean,
    
    @SerializedName("upload_id")
    val uploadId: String?,
    
    @SerializedName("chunk_index")
    val chunkIndex: Int,
    
    @SerializedName("total_chunks")
    val totalChunks: Int,
    
    @SerializedName("bytes_received")
    val bytesReceived: Long,
    
    val message: String? = null
)

/**
 * Response for upload session status.
 */
data class UploadSessionResponse(
    @SerializedName("upload_id")
    val uploadId: String,
    
    @SerializedName("file_name")
    val fileName: String,
    
    @SerializedName("file_size")
    val fileSize: Long,
    
    @SerializedName("uploaded_chunks")
    val uploadedChunks: List<Int> = emptyList(),
    
    @SerializedName("total_chunks")
    val totalChunks: Int,
    
    @SerializedName("is_complete")
    val isComplete: Boolean,
    
    @SerializedName("created_at")
    val createdAt: String? = null
)

/**
 * Response for aborting an upload session.
 */
data class AbortResponse(
    val success: Boolean,
    
    @SerializedName("message")
    val message: String? = null
)
