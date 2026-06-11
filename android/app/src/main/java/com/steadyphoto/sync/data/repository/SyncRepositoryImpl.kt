package com.steadyphoto.sync.data.repository

import android.content.Context
import android.util.Log
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.data.local.entity.UploadStatus
import kotlinx.coroutines.flow.Flow

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

    /**
     * ⚠️ DEPRECATED — DO NOT USE. This method loads entire files into memory via `it.readBytes()`,
     * which causes OOM crashes on large videos. Use `UploadManager.uploadMedia()` instead, which
     * does chunked streaming uploads with retry logic.
     * 
     * MainViewModel and UploadWorker have already been migrated to use UploadManager directly.
     * This method is kept only for backward compatibility — it delegates to UploadManager internally
     * so that even accidental callers won't get OOM crashes, but the behavior (one-by-one upload
     * with no chunking) differs from what UploadManager provides.
     */
    @Deprecated(
        message = "Use UploadManager.uploadMedia() instead — this method loads entire files into memory and causes OOM crashes on large videos",
        level = DeprecationLevel.WARNING
    )
    override suspend fun uploadMedia(items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>): Result<UploadResult> {
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
            // ⚠️ WARNING: This delegates to UploadManager.uploadMedia() internally, but note that
            // this method is deprecated and should not be used for new code. Always use 
            // UploadManager directly via AppContainer.uploadManager.uploadMedia().
            val uploadResult = com.steadyphoto.sync.data.repository.UploadManager(
                context = context,
                apiClient = apiClient,
                mediaItemDao = mediaItemDao,
                networkMonitor = NetworkConnectivityMonitor(context),
                settingsRepository = com.steadyphoto.sync.data.settings.SettingsRepository(context)
            ).uploadMedia(items, object : UploadProgressCallback {
                override suspend fun onProgressUpdated(progress: UploadSessionProgress) {}
                override suspend fun onUploadComplete(successCount: Int, failureCount: Int) {}
                override suspend fun onUploadError(error: Throwable) {}
            })

            return uploadResult
        } catch (e: Exception) {
            Log.e(TAG, "Upload session failed", e)
            markItemsFailed(items, e.message ?: "Unknown error")
            return Result.failure(e)
        }
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
