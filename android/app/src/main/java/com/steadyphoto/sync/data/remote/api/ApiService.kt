package com.steadyphoto.sync.data.remote.api

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.MultipartBody
import okhttp3.RequestBody
import okhttp3.RequestBody.Companion.asRequestBody
import okhttp3.RequestBody.Companion.toRequestBody
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
    @FormUrlEncoded
    @POST("/api/auth/login")
    suspend fun login(
        @Field("email") email: String,
        @Field("password") password: String
    ): com.steadyphoto.sync.data.remote.dto.LoginResponse

    /**
     * Upload media files to the server.
     * Uses multipart/form-data as expected by the Go backend.
     */
    @Multipart
    @POST("/api/v1/media/upload")
    suspend fun uploadMedia(
        @Header("Authorization") authHeader: String,
        @Part files: List<MultipartBody.Part>,
        @Part("albumId") albumId: RequestBody? = null
    ): com.steadyphoto.sync.data.remote.dto.UploadResponse

    /**
     * Upload a single file with progress tracking support.
     */
    @Multipart
    @POST("/api/v1/media/upload/single")
    suspend fun uploadSingleFile(
        @Header("Authorization") authHeader: String,
        @Part file: MultipartBody.Part,
        @Part("fileName") fileName: RequestBody,
        @Part("mimeType") mimeType: RequestBody,
        @Part("fileSize") fileSize: RequestBody,
        @Part("uploadId") uploadId: RequestBody? = null
    ): com.steadyphoto.sync.data.remote.dto.UploadResponse

    /**
     * Upload a file chunk for resumable uploads.
     */
    @Multipart
    @POST("/api/v1/media/upload/chunk")
    suspend fun uploadChunk(
        @Header("Authorization") authHeader: String,
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
        @Header("Authorization") authHeader: String,
        @Query("limit") limit: Int = 50
    ): com.steadyphoto.sync.data.remote.dto.SyncStatusResponse

    /**
     * Delete a media item from the server.
     */
    @FormUrlEncoded
    @POST("/api/v1/media/delete")
    suspend fun deleteMedia(
        @Header("Authorization") authHeader: String,
        @Field("mediaId") mediaId: String
    ): com.steadyphoto.sync.data.remote.dto.DeleteResponse

    /**
     * Get upload session status for resumable uploads.
     */
    @GET("/api/v1/media/upload/status")
    suspend fun getUploadStatus(
        @Header("Authorization") authHeader: String,
        @Query("uploadId") uploadId: String
    ): com.steadyphoto.sync.data.remote.dto.UploadSessionResponse

    /**
     * Abort an in-progress upload session.
     */
    @FormUrlEncoded
    @POST("/api/v1/media/upload/abort")
    suspend fun abortUpload(
        @Header("Authorization") authHeader: String,
        @Field("uploadId") uploadId: String
    ): com.steadyphoto.sync.data.remote.dto.AbortResponse

    companion object {
        const val CHUNK_SIZE = 5 * 1024 * 1024 // 5MB chunks
    }
}
