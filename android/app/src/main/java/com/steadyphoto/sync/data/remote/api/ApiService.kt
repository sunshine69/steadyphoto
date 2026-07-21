package com.steadyphoto.sync.data.remote.api

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
import retrofit2.Call
import retrofit2.http.HTTP
import retrofit2.http.Body
import retrofit2.http.Field
import retrofit2.http.FormUrlEncoded
import retrofit2.http.GET
import retrofit2.http.Header
import retrofit2.http.Multipart
import retrofit2.http.POST
import retrofit2.http.Part
import retrofit2.http.Query
import java.io.File

interface ApiService {
    
    /**
     * Login endpoint - authenticates user and returns JWT token.
     */
    @POST("/api/v1/auth/login")
    suspend fun login(
        @Header("Authorization") authHeader: String? = null,
        @Body body: com.steadyphoto.sync.data.remote.dto.LoginRequest
    ): com.steadyphoto.sync.data.remote.dto.LoginResponse

    /**
     * Refresh endpoint - validates the refresh token and returns a new access token.
     */
    @POST("/api/v1/auth/refresh")
    suspend fun refresh(
        @Body body: com.steadyphoto.sync.data.remote.dto.RefreshRequest
    ): com.steadyphoto.sync.data.remote.dto.RefreshResponse

    /**
     * Synchronous version of refresh for use in OkHttp interceptor.
     */
    @POST("/api/v1/auth/refresh")
    fun refreshSync(
        @Body body: com.steadyphoto.sync.data.remote.dto.RefreshRequest
    ): Call<com.steadyphoto.sync.data.remote.dto.RefreshResponse>

    /**
     * Upload media files to the server.
     * Uses multipart/form-data as expected by the Go backend.
     */
    @Multipart
    @POST("/api/v1/media/upload")
    suspend fun uploadMedia(
        @Part files: List<MultipartBody.Part>,
        @Part("albumId") albumId: RequestBody? = null
    ): com.steadyphoto.sync.data.remote.dto.UploadResponse

    /**
     * Upload a single file with progress tracking support.
     */
    @Multipart
    @POST("/api/v1/media/upload/single")
    suspend fun uploadSingleFile(
        @Part file: MultipartBody.Part,
        @Part("fileName") fileName: RequestBody,
        @Part("mimeType") mimeType: RequestBody,
        @Part("fileSize") fileSize: RequestBody,
        @Part("uploadId") uploadId: RequestBody? = null,
        @Part("fileCreatedAt") fileCreatedAt: RequestBody? = null
    ): com.steadyphoto.sync.data.remote.dto.UploadResponse

    /**
     * Upload a file chunk for resumable uploads.
     */
    @Multipart
    @POST("/api/v1/media/upload/chunk")
    suspend fun uploadChunk(
        @Part chunk: MultipartBody.Part,
        @Part("uploadId") uploadId: RequestBody,
        @Part("chunkIndex") chunkIndex: RequestBody,
        @Part("totalChunks") totalChunks: RequestBody,
        @Part("fileName") fileName: RequestBody? = null
    ): com.steadyphoto.sync.data.remote.dto.ChunkUploadResponse

    /**
     * Get sync status / list of uploaded media on server.
     */
    @GET("/api/v1/media")
    suspend fun getSyncStatus(
        @Query("limit") limit: Int = 50
    ): com.steadyphoto.sync.data.remote.dto.SyncStatusResponse

    /**
     * Delete a media item from the server.
     */
    @HTTP(method = "DELETE", path = "/api/v1/media/delete", hasBody = true)
    suspend fun deleteMedia(
        @Part("mediaId") mediaId: RequestBody
    ): com.steadyphoto.sync.data.remote.dto.DeleteResponse

    /**
     * Get upload session status for resumable uploads.
     */
    @GET("/api/v1/media/upload/status")
    suspend fun getUploadStatus(
        @Query("uploadId") uploadId: String
    ): com.steadyphoto.sync.data.remote.dto.UploadSessionResponse

    /**
     * Abort an in-progress upload session.
     */
    @Multipart
    @POST("/api/v1/media/upload/abort")
    suspend fun abortUpload(
        @Part("uploadId") uploadId: RequestBody
    ): com.steadyphoto.sync.data.remote.dto.AbortResponse

    companion object {
        const val CHUNK_SIZE = 5 * 1024 * 1024 // 5MB chunks
    }
}
