package com.steadyphoto.sync.data.repository

import android.content.Context
import android.util.Log
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.data.local.entity.UploadStatus
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.withTimeout
import okhttp3.MediaType.Companion.toMediaType

/**
 * Implementation of SyncRepository that handles media synchronization.
 */
class SyncRepositoryImpl(
    private val context: Context,
    private val apiClient: ApiClient,
    private val mediaItemDao: MediaItemDao,
) : SyncRepository {

    companion object {
        private const val TAG = "SyncRepository"
    }

    override fun getPendingItems(): Flow<List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>> {
        return mediaItemDao.getItemsByStatus(UploadStatus.PENDING)
    }

    override fun getUploadedItems(): Flow<List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>> {
        return mediaItemDao.getItemsByStatus(UploadStatus.UPLOADED)
    }

    override suspend fun scanNewMedia(forceFullScan: Boolean): ScanResult {
        val scanner = MediaScanner(context.contentResolver, mediaItemDao)
        
        // Clean up deleted files first
        scanner.cleanupDeletedFiles()
        
        // Perform the actual scan
        val scanResult = scanner.scanForNewMedia(forceFullScan)

        // Count all items that need processing (Pending, Failed, or stuck Uploading)
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(UploadStatus.PENDING, UploadStatus.FAILED, UploadStatus.UPLOADING)
        )

        Log.d(TAG, "Scan complete: ${scanResult.totalScanned} scanned, ${scanResult.newItemsInserted} inserted, ${pendingItems.size} pending/failed/stuck")

        if (scanResult.totalScanned == 0 && scanResult.newItemsInserted == 0 && pendingItems.isEmpty()) {
            Log.d(TAG, "No files found in MediaStore and no pending items")
            return ScanResult.NoNewItems
        }

        // If we found something new OR we have existing items to process, return Success to trigger upload
        if (scanResult.newItemsInserted > 0 || pendingItems.isNotEmpty()) {
            return ScanResult.Success(
                totalScanned = scanResult.totalScanned,
                newItemsInserted = scanResult.newItemsInserted,
                duplicatesSkipped = scanResult.duplicatesSkipped
            )
        }

        return ScanResult.NoNewItems
    }

    override suspend fun uploadMedia(items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>): Result<UploadResult> {
        // ... (rest of the file seems fine as it just delegates to ApiService or UploadManager if used)
        // Wait, the previous version used ApiService directly here. 
        // But UploadWorker uses UploadManager.
        // SyncRepositoryImpl.uploadMedia is used by MainViewModel.
        
        // Let's check if we should delegate to UploadManager here too for consistency.
        // Actually, looking at the previous turn's read_file for SyncRepositoryImpl.kt,
        // it had a full implementation using Retrofit.
        
        // I'll keep the full implementation but make sure it uses the same status list.
        
        if (items.isEmpty()) {
            Log.w(TAG, "No items to upload")
            return Result.success(UploadResult(successCount = 0, failureCount = 0))
        }

        val token = apiClient.getAuthToken()
        if (token == null) {
            Log.w(TAG, "No auth token for upload")
            markItemsFailed(items, "No authentication")
            return Result.failure(Exception("No authentication"))
        }

        try {
            val apiService = apiClient.apiService

            withTimeout(30_000) { // Increased timeout
                Log.d(TAG, "Uploading ${items.size} items to server")
                
                var successCount = 0
                var failureCount = 0
                
                // For simplicity and reliability, we process items one by one here 
                // (MainViewModel uses this for immediate UI feedback)
                for (item in items) {
                    try {
                        mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADING)
                        
                        val uri = android.net.Uri.parse(item.uri)
                        val inputStream = context.contentResolver.openInputStream(uri) 
                            ?: throw Exception("Cannot open input stream")
                        
                        val fileContent = inputStream.use { it.readBytes() }
                        val filePart = okhttp3.MultipartBody.Part.createFormData(
                            "file",
                            item.fileName,
                            okhttp3.RequestBody.create(item.mimeType.toMediaType(), fileContent)
                        )

                        val response = apiService.uploadSingleFile(
                            file = filePart,
                            fileName = okhttp3.RequestBody.create("text/plain".toMediaType(), item.fileName),
                            mimeType = okhttp3.RequestBody.create("text/plain".toMediaType(), item.mimeType),
                            fileSize = okhttp3.RequestBody.create("text/plain".toMediaType(), item.fileSize.toString())
                        )

                        mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
                        successCount++
                    } catch (e: Exception) {
                        Log.w(TAG, "Failed to upload ${item.fileName}: ${e.message}")
                        mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, e.message)
                        failureCount++
                    }
                }

                return@withTimeout UploadResult(successCount = successCount, failureCount = failureCount)
            }
        } catch (e: Exception) {
            Log.e(TAG, "Upload session failed", e)
            return Result.failure(e)
        }
        return Result.success(UploadResult(0, items.size))
    }

    private suspend fun markItemsFailed(items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>, reason: String) {
        for (item in items) {
            mediaItemDao.updateStatus(item.id, UploadStatus.FAILED, reason)
        }
    }

    override suspend fun markAsUploaded(item: com.steadyphoto.sync.data.local.entity.MediaItemEntity) {
        mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
    }

    override suspend fun deleteMedia(itemId: String): Result<Unit> {
        val itemIdLong = itemId.toLongOrNull() ?: return Result.failure(Exception("Invalid ID"))
        mediaItemDao.deleteByMediaIds(listOf(itemIdLong))
        return Result.success(Unit)
    }

    override suspend fun getSyncStatus(limit: Int): Result<List<com.steadyphoto.sync.data.remote.dto.UploadedMedia>> {
        return Result.success(emptyList()) // Placeholder
    }
}
