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
 * Note: The Go server uses camelCase for all fields in the response.
 */
data class ChunkUploadResponse(
    val success: Boolean,
    
    @SerializedName("uploadId")
    val uploadId: String?,
    
    @SerializedName("chunkIndex")
    val chunkIndex: Int,
    
    @SerializedName("totalChunks")
    val totalChunks: Int,
    
    @SerializedName("bytesReceived")
    val bytesReceived: Long,
    
    val message: String? = null
)

/**
 * Response for upload session status.
 * Note: The Go server uses camelCase for all fields in the response.
 */
data class UploadSessionResponse(
    @SerializedName("uploadId")
    val uploadId: String,
    
    @SerializedName("fileName")
    val fileName: String,
    
    @SerializedName("fileSize")
    val fileSize: Long,
    
    @SerializedName("uploadedChunks")
    val uploadedChunks: List<Int> = emptyList(),
    
    @SerializedName("totalChunks")
    val totalChunks: Int,
    
    @SerializedName("isComplete")
    val isComplete: Boolean,
    
    @SerializedName("createdAt")
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
