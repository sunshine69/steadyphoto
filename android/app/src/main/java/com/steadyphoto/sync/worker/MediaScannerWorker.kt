package com.steadyphoto.sync.worker

import android.content.ContentResolver
import android.content.Context
import android.database.Cursor
import android.provider.MediaStore
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.di.AppContainer
import org.koin.androidx.compose.koinViewModel
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

class MediaScannerWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params), KoinComponent {

    private val container: AppContainer by inject()

    override suspend fun doWork(): Result {
        return try {
            // 1. Scan device storage for media files (Standard Sync)
            val newItems = scanMediaFiles(applicationContext.contentResolver)
            
            if (newItems.isNotEmpty()) {
                var insertedCount = 0
                for (item in newItems) {
                    val existing = container.mediaItemDao.getByHash(item.hash)
                    if (existing == null) {
                        container.mediaItemDao.insert(item)
                        insertedCount++
                    }
                }
                Log.d("MediaScannerWorker", "Scanned ${newItems.size} files, inserted $insertedCount new items")
            } else {
                Log.d("MediaScannerWorker", "No new media files found during scan phase")
            }

            // 2. Reconciliation (Cleanup Phase) - Remove ghost entries of deleted files
            reconcileDeletedFiles()

            Result.success()
        } catch (e: Exception) {
            Log.e("MediaScannerWorker", "Scan or reconciliation failed", e)
            Result.retry()
        }
    }

private suspend fun reconcileDeletedFiles() {
    val contentResolver = applicationContext.contentResolver
    try {
        // Get all IDs currently in our local Room database
        val storedIds = container.mediaItemDao.getAllStoredMediaIds().toSet()
        if (storedIds.isEmpty()) return

        // Query MediaStore for ALL valid image/video IDs currently on the system
        val validSystemIds = mutableSetOf<Long>()
        val projection = arrayOf(MediaStore.MediaColumns._ID)

        // Scan Images
        contentResolver.query(
            MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
            projection, null, null, null
        )?.use { cursor ->
            val idColumn = cursor.getColumnIndexOrThrow(MediaStore.MediaColumns._ID)
            while (cursor.moveToNext()) validSystemIds.add(cursor.getLong(idColumn))
        }

        // Scan Videos
        contentResolver.query(
            MediaStore.Video.Media.EXTERNAL_CONTENT_URI,
            projection, null, null, null
        )?.use { cursor ->
            val idColumn = cursor.getColumnIndexOrThrow(MediaStore.MediaColumns._ID)
            while (cursor.moveToNext()) validSystemIds.add(cursor.getLong(idColumn))
        }

        // Identify "Ghosts" (In DB but not in MediaStore)
        val idsToDelete = storedIds.filter { it !in validSystemIds }

        if (idsToDelete.isNotEmpty()) {
            container.mediaItemDao.deleteByMediaIds(idsToDelete)
            Log.i("MediaScannerWorker", "Reconciliation: Removed ${idsToDelete.size} ghost entries from database.")
        } else {
            Log.d("MediaScannerWorker", "Reconciliation: No deleted files found to clean up.")
        }
    } catch (e: Exception) {
        Log.e("MediaScannerWorker", "Error during reconciliation phase", e)
    }
}

    private fun scanMediaFiles(contentResolver: ContentResolver): List<MediaItemEntity> {
        val items = mutableListOf<MediaItemEntity>()
        
        // Query images first
        items.addAll(scanImages(contentResolver))
        
        // Then query videos
        items.addAll(scanVideos(contentResolver))
        
        Log.d("MediaScannerWorker", "Found ${items.size} total media files")
        return items
    }

    private fun scanImages(contentResolver: ContentResolver): List<MediaItemEntity> {
        val items = mutableListOf<MediaItemEntity>()
        
        // Define the columns we want to retrieve
        val projection = arrayOf(
            MediaStore.Images.Media._ID,
            MediaStore.Images.Media.DISPLAY_NAME,
            MediaStore.Images.Media.MIME_TYPE,
            MediaStore.Images.Media.SIZE,
            MediaStore.Images.Media.DATE_ADDED,
            MediaStore.Images.Media.DATA // Absolute path (deprecated but still useful)
        )
        
        // Filter for common image types and minimum size (>10KB to skip thumbnails)
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
        
        // Sort by date added descending (newest first)
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
                    
                    // Construct URI for this media item
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
                            captureTime = dateAdded * 1000 // Convert seconds to milliseconds
                        )
                    )
                } catch (e: Exception) {
                    Log.w("MediaScannerWorker", "Error processing image cursor row", e)
                }
            }
        } ?: run {
            Log.e("MediaScannerWorker", "Failed to query images from MediaStore")
        }
        
        return items
    }

    private fun scanVideos(contentResolver: ContentResolver): List<MediaItemEntity> {
        val items = mutableListOf<MediaItemEntity>()
        
        // Define the columns we want to retrieve
        val projection = arrayOf(
            MediaStore.Video.Media._ID,
            MediaStore.Video.Media.DISPLAY_NAME,
            MediaStore.Video.Media.MIME_TYPE,
            MediaStore.Video.Media.SIZE,
            MediaStore.Video.Media.DATE_ADDED,
            MediaStore.Video.Media.DATA // Absolute path (deprecated but still useful)
        )
        
        // Filter for common video types and minimum size (>10KB to skip thumbnails)
        val selection = "${MediaStore.Video.Media.SIZE} > 10240 AND (" +
                "${MediaStore.Video.Media.MIME_TYPE} = ? OR " +
                "${MediaStore.Video.Media.MIME_TYPE} = ?" +
                ")"
        val selectionArgs = arrayOf(
            "video/mp4",
            "video/quicktime" // .mov files from iPhone
        )
        
        // Sort by date added descending (newest first)
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
                    
                    // Construct URI for this media item
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
                            captureTime = dateAdded * 1000 // Convert seconds to milliseconds
                        )
                    )
                } catch (e: Exception) {
                    Log.w("MediaScannerWorker", "Error processing video cursor row", e)
                }
            }
        } ?: run {
            Log.e("MediaScannerWorker", "Failed to query videos from MediaStore")
        }
        
        return items
    }
}
