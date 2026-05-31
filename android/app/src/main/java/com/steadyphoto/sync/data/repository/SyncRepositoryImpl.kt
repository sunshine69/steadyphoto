package com.steadyphoto.sync.data.repository

import android.content.ContentResolver
import android.content.Context
import android.provider.MediaStore
import androidx.work.WorkManager
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.di.AppContainer
import kotlinx.coroutines.flow.Flow
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody.Companion.toRequestBody
import java.io.File

class SyncRepositoryImpl(
    private val context: Context,
    private val mediaItemDao: MediaItemDao,
    private val container: AppContainer
) : SyncRepository {

    override fun getPendingItems(): Flow<List<MediaItemEntity>> {
        return mediaItemDao.getItemsByStatus(UploadStatus.PENDING)
    }

    override fun getUploadedItems(): Flow<List<MediaItemEntity>> {
        return mediaItemDao.getItemsByStatus(UploadStatus.UPLOADED)
    }

    override suspend fun scanNewMedia(): ScanResult {
        try {
            // Trigger the async media scanner worker in background
            WorkManager.getInstance(context).enqueueUniqueWork(
                "media_scan",
                androidx.work.ExistingWorkPolicy.REPLACE,
                androidx.work.OneTimeWorkRequest.Builder(com.steadyphoto.sync.worker.MediaScannerWorker::class.java).build()
            )
            
            // For synchronous scanning (fallback), we can also do a direct scan here
            val scannedItems = performDirectScan(context.contentResolver)
            
            if (scannedItems.isEmpty()) {
                return ScanResult.NoNewItems
            }
            
            var insertedCount = 0
            var duplicatesSkipped = 0
            
            for (item in scannedItems) {
                val existing = mediaItemDao.getByHash(item.hash)
                if (existing == null) {
                    mediaItemDao.insert(item)
                    insertedCount++
                } else {
                    duplicatesSkipped++
                }
            }
            
            return ScanResult.Success(
                totalScanned = scannedItems.size,
                newItemsInserted = insertedCount,
                duplicatesSkipped = duplicatesSkipped
            )
        } catch (e: Exception) {
            return ScanResult.Error(message = "Scan failed: ${e.message}", cause = e)
        }
    }

    /**
     * Performs a direct synchronous scan of media files.
     */
    private fun performDirectScan(contentResolver: ContentResolver): List<MediaItemEntity> {
        val items = mutableListOf<MediaItemEntity>()
        
        // Query images
        items.addAll(scanImages(contentResolver))
        
        // Then query videos
        items.addAll(scanVideos(contentResolver))
        
        return items
    }

    private fun scanImages(contentResolver: ContentResolver): List<MediaItemEntity> {
        val items = mutableListOf<MediaItemEntity>()
        
        val projection = arrayOf(
            MediaStore.Images.Media._ID,
            MediaStore.Images.Media.DISPLAY_NAME,
            MediaStore.Images.Media.MIME_TYPE,
            MediaStore.Images.Media.SIZE,
            MediaStore.Images.Media.DATE_ADDED,
            MediaStore.Images.Media.DATA
        )
        
        val selection = "${MediaStore.Images.Media.SIZE} > 10240 AND (" +
                "${MediaStore.Images.Media.MIME_TYPE} = ? OR " +
                "${MediaStore.Images.Media.MIME_TYPE} = ? OR " +
                "${MediaStore.Images.Media.MIME_TYPE} = ? OR " +
                "${MediaStore.Images.Media.MIME_TYPE} = ?" +
                ")"
        val selectionArgs = arrayOf(
            "image/jpeg",
            "image/png",
            "image/heic",
            "image/webp"
        )
        
        val sortOrder = "${MediaStore.Images.Media.DATE_ADDED} DESC"
        
        contentResolver.query(
            MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
            projection,
            selection,
            selectionArgs,
            sortOrder
        )?.use { cursor ->
            val idIndex = cursor.getColumnIndexOrThrow(MediaStore.Images.Media._ID)
            val nameIndex = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.DISPLAY_NAME)
            val mimeIndex = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.MIME_TYPE)
            val sizeIndex = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.SIZE)
            val dateAddedIndex = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.DATE_ADDED)
            val pathIndex = cursor.getColumnIndexOrThrow(MediaStore.Images.Media.DATA)
            
            while (cursor.moveToNext()) {
                try {
                    val id = cursor.getLong(idIndex)
                    val fileName = cursor.getString(nameIndex)
                    val mimeType = cursor.getString(mimeIndex)
                    val fileSize = cursor.getLong(sizeIndex)
                    val dateAdded = cursor.getLong(dateAddedIndex)
                    val localPath = cursor.getString(pathIndex)
                    
                    val uri = "${MediaStore.Images.Media.EXTERNAL_CONTENT_URI}/$id"
                    
                    // Compute SHA256 hash using Gomobile bindings
                    val hash = if (localPath != null && localPath.isNotEmpty()) {
                        com.steadyphoto.sync.util.MediaUtils.computeHash(localPath)
                            ?: "hash_failed_${fileName}_${dateAdded}"
                    } else {
                        "no_path_${fileName}_${dateAdded}"
                    }
                    
                    items.add(
                        MediaItemEntity(
                            uri = uri,
                            localPath = if (localPath.isNotEmpty()) localPath else null,
                            fileName = fileName,
                            hash = hash,
                            mimeType = mimeType,
                            fileSize = fileSize,
                            captureTime = dateAdded * 1000
                        )
                    )
                } catch (e: Exception) {
                    // Skip problematic items
                }
            }
        }
        
        return items
    }

    private fun scanVideos(contentResolver: ContentResolver): List<MediaItemEntity> {
        val items = mutableListOf<MediaItemEntity>()
        
        val projection = arrayOf(
            MediaStore.Video.Media._ID,
            MediaStore.Video.Media.DISPLAY_NAME,
            MediaStore.Video.Media.MIME_TYPE,
            MediaStore.Video.Media.SIZE,
            MediaStore.Video.Media.DATE_ADDED,
            MediaStore.Video.Media.DATA
        )
        
        val selection = "${MediaStore.Video.Media.SIZE} > 10240 AND (" +
                "${MediaStore.Video.Media.MIME_TYPE} = ? OR " +
                "${MediaStore.Video.Media.MIME_TYPE} = ?" +
                ")"
        val selectionArgs = arrayOf(
            "video/mp4",
            "video/quicktime"
        )
        
        val sortOrder = "${MediaStore.Video.Media.DATE_ADDED} DESC"
        
        contentResolver.query(
            MediaStore.Video.Media.EXTERNAL_CONTENT_URI,
            projection,
            selection,
            selectionArgs,
            sortOrder
        )?.use { cursor ->
            val idIndex = cursor.getColumnIndexOrThrow(MediaStore.Video.Media._ID)
            val nameIndex = cursor.getColumnIndexOrThrow(MediaStore.Video.Media.DISPLAY_NAME)
            val mimeIndex = cursor.getColumnIndexOrThrow(MediaStore.Video.Media.MIME_TYPE)
            val sizeIndex = cursor.getColumnIndexOrThrow(MediaStore.Video.Media.SIZE)
            val dateAddedIndex = cursor.getColumnIndexOrThrow(MediaStore.Video.Media.DATE_ADDED)
            val pathIndex = cursor.getColumnIndexOrThrow(MediaStore.Video.Media.DATA)
            
            while (cursor.moveToNext()) {
                try {
                    val id = cursor.getLong(idIndex)
                    val fileName = cursor.getString(nameIndex)
                    val mimeType = cursor.getString(mimeIndex)
                    val fileSize = cursor.getLong(sizeIndex)
                    val dateAdded = cursor.getLong(dateAddedIndex)
                    val localPath = cursor.getString(pathIndex)
                    
                    val uri = "${MediaStore.Video.Media.EXTERNAL_CONTENT_URI}/$id"
                    
                    // Compute SHA256 hash using Gomobile bindings
                    val hash = if (localPath != null && localPath.isNotEmpty()) {
                        com.steadyphoto.sync.util.MediaUtils.computeHash(localPath)
                            ?: "hash_failed_${fileName}_${dateAdded}"
                    } else {
                        "no_path_${fileName}_${dateAdded}"
                    }
                    
                    items.add(
                        MediaItemEntity(
                            uri = uri,
                            localPath = if (localPath.isNotEmpty()) localPath else null,
                            fileName = fileName,
                            hash = hash,
                            mimeType = mimeType,
                            fileSize = fileSize,
                            captureTime = dateAdded * 1000
                        )
                    )
                } catch (e: Exception) {
                    // Skip problematic items
                }
            }
        }
        
        return items
    }

    override suspend fun uploadMedia(items: List<MediaItemEntity>): Result<Unit> {
        return try {
            // Delegate to UploadManager for consistent upload handling with progress tracking
            val result = container.uploadManager.uploadMedia(items)
            
            if (result.isSuccess) {
                val uploadResult = result.getOrNull()
                // Mark successfully uploaded items in the database
                uploadResult?.let { res ->
                    var successCount = 0
                    for (item in items) {
                        if (successCount < res.successCount) {
                            mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
                            successCount++
                        } else {
                            // Mark remaining as failed
                            mediaItemDao.updateStatus(
                                item.id, 
                                UploadStatus.FAILED, 
                                "Upload batch partially failed"
                            )
                        }
                    }
                }
            } else {
                // Mark all items as failed for retry
                val error = result.exceptionOrNull()
                items.forEach { item ->
                    mediaItemDao.updateStatus(
                        item.id, 
                        UploadStatus.FAILED, 
                        error?.message ?: "Upload failed"
                    )
                }
            }
            
            result.map { Unit }
        } catch (e: Exception) {
            // Mark all items as failed for retry
            items.forEach { item ->
                mediaItemDao.updateStatus(
                    item.id, 
                    UploadStatus.FAILED, 
                    e.message ?: "Upload error"
                )
            }
            Result.failure(e)
        }
    }

    override suspend fun markAsUploaded(item: MediaItemEntity) {
        mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
    }

    override suspend fun deleteMedia(itemId: String): Result<Unit> {
        return try {
            val token = getAuthToken() ?: return Result.failure(Exception("No auth token"))
            
            container.apiService.deleteMedia(
                authHeader = "Bearer $token",
                mediaId = itemId
            )
            
            // Update local status after successful deletion
            mediaItemDao.updateStatus(itemId.toLongOrNull() ?: 0L, UploadStatus.DELETED)
            Result.success(Unit)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    override suspend fun getSyncStatus(limit: Int): Result<List<com.steadyphoto.sync.data.remote.dto.UploadedMedia>> {
        return try {
            val token = getAuthToken() ?: return Result.failure(Exception("No auth token"))
            
            val response = container.apiService.getSyncStatus(
                authHeader = "Bearer $token",
                limit = limit
            )
            Result.success(response.items)
        } catch (e: Exception) {
            Result.failure(e)
        }
    }

    private fun getAuthToken(): String? {
        val prefs = context.getSharedPreferences("app_prefs", Context.MODE_PRIVATE)
        return prefs.getString("auth_token", null)
    }
}
