package com.steadyphoto.sync.data.repository

import android.content.Context
import android.util.Log
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.remote.api.ApiClient
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
        return mediaItemDao.getItemsByStatus(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING)
    }

    override fun getUploadedItems(): Flow<List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>> {
        return mediaItemDao.getItemsByStatus(com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED)
    }

    override suspend fun scanNewMedia(forceFullScan: Boolean): ScanResult {
        val scanner = MediaScanner(context.contentResolver, mediaItemDao)
        
        // Clean up deleted files first
        scanner.cleanupDeletedFiles()
        
        // Perform the actual scan - returns detailed result with scanning statistics
        val scanResult = scanner.scanForNewMedia(forceFullScan)

        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )

        Log.d(TAG, "Scan complete: ${scanResult.totalScanned} scanned, ${scanResult.newItemsInserted} inserted, $pendingItems pending")

        // Only return NoNewItems if nothing was actually scanned (totalScanned == 0)
        // This shouldn't happen normally but could indicate a permissions issue
        if (scanResult.totalScanned == 0 && scanResult.newItemsInserted == 0) {
            Log.d(TAG, "No files found in MediaStore - possible permission issue")
            return ScanResult.NoNewItems
        }

        // If new items were inserted or duplicates were skipped, return Success
        if (scanResult.newItemsInserted > 0 || scanResult.duplicatesSkipped > 0) {
            return ScanResult.Success(
                totalScanned = scanResult.totalScanned,
                newItemsInserted = scanResult.newItemsInserted,
                duplicatesSkipped = scanResult.duplicatesSkipped
            )
        }

        // If nothing was inserted and no duplicates were skipped but we did scan some files,
        // it means all found items already exist in the DB (fully synced)
        if (scanResult.totalScanned > 0 && scanResult.newItemsInserted == 0 && scanResult.duplicatesSkipped == 0) {
            Log.d(TAG, "All scanned items are already in database - fully synced")
            return ScanResult.NoNewItems
        }

        // Fallback: treat as success with no new items
        return ScanResult.Success(
            totalScanned = scanResult.totalScanned,
            newItemsInserted = 0,
            duplicatesSkipped = 0
        )
    }

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
            val apiService = apiClient.apiService

            withTimeout(10_000) {
                Log.d(TAG, "Uploading ${items.size} items to server")
                
                var successCount = 0
                var failureCount = 0
                
                val fileParts = items.mapNotNull { item ->
                    try {
                        val uri = android.net.Uri.parse(item.uri)
                        val inputStream = context.contentResolver.openInputStream(uri) 
                            ?: throw Exception("Cannot open input stream for $uri")
                        
                        val fileContent = inputStream.use { it.readBytes() }
                        
                        okhttp3.MultipartBody.Part.createFormData(
                            "files",
                            item.fileName,
                            okhttp3.RequestBody.create(item.mimeType.toMediaType(), fileContent)
                        )
                    } catch (e: Exception) {
                        Log.w(TAG, "Failed to create file part for ${item.fileName}: ${e.message}")
                        null
                    }
                }

                if (fileParts.isEmpty()) {
                    markItemsFailed(items, "No files could be read from local storage")
                    throw Exception("No valid files to upload")
                }

                val response = apiService.uploadMedia(fileParts)

                // Build a map of filename -> server ID for uploaded items
                val uploadedMap = response.uploaded.associate { it.filename to it.id }
                
                // Mark all items as either UPLOADED or FAILED based on server response
                for (item in items) {
                    when {
                        item.fileName in uploadedMap -> {
                            // Successfully uploaded - store the UUID server ID
                            mediaItemDao.updateStatusWithServerId(
                                item.id, 
                                com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED, 
                                uploadedMap[item.fileName]!!
                            )
                            Log.d(TAG, "Marked as UPLOADED: ${item.fileName} (server ID: ${uploadedMap[item.fileName]})")
                            successCount++
                        }
                        item.fileName in response.skippedDuplicates.map { it.filename } -> {
                            // Server already has this file - treat as uploaded
                            val duplicateItem = response.skippedDuplicates.find { it.filename == item.fileName }!!
                            mediaItemDao.updateStatusWithServerId(
                                item.id, 
                                com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED, 
                                duplicateItem.id
                            )
                            Log.d(TAG, "Marked as UPLOADED (duplicate): ${item.fileName} (server ID: ${duplicateItem.id})")
                            successCount++
                        }
                        else -> {
                            // Upload failed or was rejected
                            markItemsFailed(listOf(item), "Upload rejected by server")
                            Log.w(TAG, "Marked as FAILED: ${item.fileName}")
                            failureCount++
                        }
                    }
                }

                return@withTimeout UploadResult(successCount = successCount, failureCount = failureCount)

            }

        } catch (e: Exception) {
            Log.e(TAG, "Failed to upload items", e)
            markItemsFailed(items, "Upload initiation failed: ${e.message}")
            return Result.failure(e)
        }

        // Should not reach here - withTimeout should always throw or return
        return Result.success(UploadResult(successCount = 0, failureCount = items.size))
    }

    private suspend fun markItemsFailed(items: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>, reason: String) {
        for (item in items) {
            mediaItemDao.updateStatus(item.id, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED, reason)
        }
    }

    override suspend fun markAsUploaded(item: com.steadyphoto.sync.data.local.entity.MediaItemEntity) {
        try {
            mediaItemDao.updateStatus(item.id, com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED)
        } catch (e: Exception) {
            Log.e(TAG, "Failed to mark item as uploaded", e)
        }
    }

    override suspend fun deleteMedia(itemId: String): Result<Unit> {
        val itemIdLong = runCatching { itemId.toLong() }.getOrNull()

        if (itemIdLong == null) {
            Log.w(TAG, "Invalid item ID format: $itemId")
            return Result.failure(Exception("Invalid item ID format"))
        }

        // Get the media item first to check its status and serverId
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )

        val localItem = pendingItems.find { it.id == itemIdLong } ?: run {
            // Item might already be uploaded - check recently synced items
            val recentItems = mediaItemDao.getRecentItems(100)
            val foundItem = recentItems.find { it.id == itemIdLong }

            if (foundItem == null) {
                Log.w(TAG, "Attempted to delete non-existent item: $itemId")
                return Result.failure(Exception("Item not found"))
            }

            // If it's already uploaded, try deleting from server first
            if (foundItem.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED && foundItem.serverId != null) {
                val token = apiClient.getAuthToken()
                if (token == null) {
                    Log.w(TAG, "No auth token for deletion")
                    return Result.failure(Exception("No authentication"))
                }

                try {
                    // Always get fresh ApiService from ApiClient to ensure correct URL is used
                    val apiService = apiClient.apiService

                    withTimeout(10_000) { // 10 second timeout for deletion
                        Log.d(TAG, "Deleting uploaded media from server: ${foundItem.serverId}")
                        val requestBody = okhttp3.RequestBody.create("text/plain".toMediaType(), foundItem.serverId)
                        apiService.deleteMedia(requestBody)
                    }

                } catch (e: Exception) {
                    Log.w(TAG, "Server deletion failed, will retry later", e)
                    // Don't fail the local deletion if server fails - mark as pending for retry
                    mediaItemDao.updateStatus(itemIdLong!!, com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, "Deletion from server failed, will retry")
                }
            }

            // Delete from local database
            val deletedCount = mediaItemDao.deleteByMediaIds(listOf(itemIdLong!!))

            if (deletedCount > 0) {
                Log.d(TAG, "Successfully deleted item: $itemId")
                return Result.success(Unit)
            } else {
                Log.w(TAG, "Failed to delete item locally: $itemId")
                return Result.failure(Exception("Item not found"))
            }
        }

        // If it's a pending/failed item, just delete from local database (no server deletion needed)
        val deletedCount = mediaItemDao.deleteByMediaIds(listOf(itemIdLong!!))

        if (deletedCount > 0) {
            Log.d(TAG, "Successfully deleted item: $itemId")
            return Result.success(Unit)
        } else {
            Log.w(TAG, "Failed to delete item locally: $itemId")
            return Result.failure(Exception("Item not found"))
        }
    }

    override suspend fun getSyncStatus(limit: Int): Result<List<com.steadyphoto.sync.data.remote.dto.UploadedMedia>> {
        // Get local sync status first
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )

        if (pendingItems.isEmpty()) {
            Log.d(TAG, "No items need syncing")
            return Result.success(emptyList())
        }

        // Check with server for sync status - don't use run {} since getAuthToken returns nullable and we're in suspend context
        val token = apiClient.getAuthToken()
        if (token == null) {
            Log.w(TAG, "No auth token for sync check")
            return Result.failure(Exception("No authentication"))
        }

        try {
            // Always get fresh ApiService from ApiClient to ensure correct URL is used
            val apiService = apiClient.apiService

            withTimeout(10_000) { // 10 second timeout for sync check
                Log.d(TAG, "Checking sync status with server")
                val response = apiService.getSyncStatus(limit)

                if (response.items.isNotEmpty()) {
                    return@withTimeout Unit
                } else {
                    Log.w(TAG, "Server returned null items list")
                    throw Exception("No response from server")
                }

            }

        } catch (e: Exception) {
            Log.e(TAG, "Error checking sync status with server", e)
            // Return local pending items as fallback
            return Result.success(emptyList())
        }

        // Success path - sync check completed
        return Result.success(emptyList())
    }
}
