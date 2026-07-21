package com.steadyphoto.sync.data.repository

import android.content.ContentResolver
import android.media.MediaScannerConnection
import android.os.Build
import android.provider.MediaStore
import android.util.Log
import java.io.File
import java.security.MessageDigest

/**
 * Scans the device's MediaStore for new images and videos that haven't been synced yet.
 */
class MediaScanner(
    private val contentResolver: ContentResolver,
    private val mediaItemDao: com.steadyphoto.sync.data.local.dao.MediaItemDao,
) {

    companion object {
        private const val TAG = "MediaScanner"
        
        private val IMAGE_MIME_TYPES = arrayOf("image/jpeg", "image/png", "image/heic", "image/webp")
        private val VIDEO_MIME_TYPES = arrayOf("video/mp4", "video/quicktime", "video/x-ms-wmv")
        
        // Removed leading dots to match substringAfterLast result
        private val IMAGE_EXTENSIONS = setOf("jpg", "jpeg", "png", "heic", "heif", "webp", "gif")
        private val VIDEO_EXTENSIONS = setOf("mp4", "mov", "wmv", "avi", "mkv", "webm")
    }

    /**
     * Scan the device for new media files and add them to the database if not already present.
     */
    suspend fun scanForNewMedia(forceFullScan: Boolean = false, lastSyncTimestamp: Long = 0L): MediaScanResult {
        var totalScanned = 0
        var newItemsInserted = 0

        // 1. ALWAYS perform the reliable, type-specific scans first using timestamp filtering if provided.
        Log.d(TAG, "Running standard MediaStore scan (Images & Video). Last sync: $lastSyncTimestamp")
        
        val imageResult = scanMediaType(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, IMAGE_MIME_TYPES, lastSyncTimestamp)
        totalScanned += imageResult.totalScanned
        newItemsInserted += imageResult.newItemsInserted
        
        val videoResult = scanMediaType(MediaStore.Video.Media.EXTERNAL_CONTENT_URI, VIDEO_MIME_TYPES, lastSyncTimestamp)
        totalScanned += videoResult.totalScanned
        newItemsInserted += videoResult.newItemsInserted

        // 2. Perform the broad "Files" scan if requested (fallback mechanism).
        if (forceFullScan) {
            Log.d(TAG, "Running broad Files provider scan as fallback")
            
            val unifiedCursor = contentResolver.query(
                MediaStore.Files.getContentUri("external"),
                projectionForScan(),
                "${MediaStore.MediaColumns.SIZE} > 1024", 
                null,
                null
            )
            
            unifiedCursor?.use { cursor ->
                val result = scanAllRows(cursor)
                totalScanned += result.totalScanned
                newItemsInserted += result.newItemsInserted
            } ?: run {
                Log.e(TAG, "Failed to query MediaStore for unified files")
            }
        }

        Log.d(TAG, "Scan complete: $totalScanned scanned, $newItemsInserted inserted")
        
        // 3. Count total pending/failed items in DB to decide if uploader should run
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, 
                   com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED,
                   com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADING)
        )
        
        val duplicatesSkipped = maxOf(0, pendingItems.size - newItemsInserted)

        return MediaScanResult(totalScanned, newItemsInserted, duplicatesSkipped)
    }

    private fun projectionForScan() = arrayOf(
            MediaStore.MediaColumns._ID, 
            MediaStore.MediaColumns.DISPLAY_NAME, 
            MediaStore.MediaColumns.MIME_TYPE, 
            MediaStore.MediaColumns.SIZE
    )

    private suspend fun scanAllRows(cursor: android.database.Cursor): MediaScanResult {
        var totalScanned = 0
        var newItemsInserted = 0
        
        val idColumnName = MediaStore.MediaColumns._ID

        cursor.use {
            while (it.moveToNext()) {
                try {
                    totalScanned++
                    
                    val id = it.getLong(it.getColumnIndexOrThrow(idColumnName))
                    val fileName = it.getString(it.getColumnIndexOrThrow(MediaStore.MediaColumns.DISPLAY_NAME)) ?: continue
                    val mimeType = it.getString(it.getColumnIndexOrThrow(MediaStore.MediaColumns.MIME_TYPE))
                    val fileSize = it.getLong(it.getColumnIndexOrThrow(MediaStore.MediaColumns.SIZE))

                    val ext = fileName.substringAfterLast('.', "").lowercase()
                    val isImage = (mimeType?.startsWith("image/") == true) || IMAGE_EXTENSIONS.contains(ext)
                    val isVideo = (mimeType?.startsWith("video/") == true) || VIDEO_EXTENSIONS.contains(ext)

                    if (!isImage && !isVideo) continue 

                    val contentUri = MediaStore.Files.getContentUri("external").buildUpon()
                        .appendPath(id.toString())
                        .build()

                    if (isAlreadyInDb(contentUri.toString())) continue

                    val hash = computeHashFromContentUri(contentUri)
                    if (isAlreadyInDbByHash(hash)) continue

                    val entity = com.steadyphoto.sync.data.local.entity.MediaItemEntity(
                        uri = contentUri.toString(),
                        localPath = null, 
                        fileName = fileName,
                        hash = hash,
                        mimeType = mimeType ?: if (isImage) "image/jpeg" else "video/mp4",
                        fileSize = fileSize,
                        uploadStatus = com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                    )

                    val insertedId = mediaItemDao.insert(entity)
                    if (insertedId != -1L) {
                        newItemsInserted++
                        Log.d(TAG, "Added new file via Files provider: $fileName")
                    }

                } catch (e: Exception) {
                    Log.w(TAG, "Error processing media item in broad scan", e)
                }
            }
        }

        return MediaScanResult(totalScanned, newItemsInserted, 0)
    }

    private suspend fun isAlreadyInDb(uri: String): Boolean {
        val existing = mediaItemDao.getByUri(uri)
        return existing != null && (existing.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED || 
                                   existing.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING ||
                                   existing.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADING)
    }

    private suspend fun isAlreadyInDbByHash(hash: String): Boolean {
        if (hash.isEmpty()) return false
        val existing = mediaItemDao.getByHash(hash)
        return existing != null && (existing.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED ||
                                   existing.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING ||
                                   existing.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADING)
    }

    private suspend fun scanMediaType(uri: android.net.Uri, mimeTypes: Array<String>, lastSyncTimestamp: Long = 0L): MediaScanResult {
        var totalScanned = 0
        var newItemsInserted = 0
        
        val mimeTypeColumn = MediaStore.MediaColumns.MIME_TYPE
        val idColumnName = MediaStore.MediaColumns._ID
        val dateAddedColumn = MediaStore.MediaColumns.DATE_ADDED

        val mimeTypeSelection = "$mimeTypeColumn IN (${mimeTypes.joinToString(",") { "'$it'" }})"
        // Filter by timestamp if provided to enable incremental scanning (Immich-style)
        val timeSelection = if (lastSyncTimestamp > 0L) " AND $dateAddedColumn > $lastSyncTimestamp" else ""
        val finalSelection = "$mimeTypeSelection AND ${MediaStore.MediaColumns.SIZE} > 0$timeSelection"
        
        contentResolver.query(
            uri,
            arrayOf(idColumnName, MediaStore.MediaColumns.DISPLAY_NAME, mimeTypeColumn, MediaStore.MediaColumns.SIZE, MediaStore.MediaColumns.DATE_ADDED),
            finalSelection,
            null,
            null
        )?.use { cursor ->
            while (cursor.moveToNext()) {
                try {
                    totalScanned++
                    val id = cursor.getLong(cursor.getColumnIndexOrThrow(idColumnName))
                    val fileName = cursor.getString(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.DISPLAY_NAME)) ?: continue
                    val mimeType = cursor.getString(cursor.getColumnIndexOrThrow(mimeTypeColumn)) ?: continue
                    val fileSize = cursor.getLong(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.SIZE))
                    val dateAdded = cursor.getLong(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.DATE_ADDED))

                    val contentUri = uri.buildUpon().appendPath(id.toString()).build()

                    if (isAlreadyInDb(contentUri.toString())) continue

                    val hash = computeHashFromContentUri(contentUri)
                    if (isAlreadyInDbByHash(hash)) continue

                    val entity = com.steadyphoto.sync.data.local.entity.MediaItemEntity(
                        uri = contentUri.toString(),
                        localPath = null,
                        fileName = fileName,
                        hash = hash,
                        mimeType = mimeType,
                        fileSize = fileSize,
                        uploadStatus = com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                        fileCreatedAt = dateAdded,
                    )

                    val insertedId = mediaItemDao.insert(entity)
                    if (insertedId != -1L) {
                        newItemsInserted++
                        Log.d(TAG, "Added new file via type scan: $fileName (dateAdded=$dateAdded)")
                    }
                } catch (e: Exception) {
                    Log.w(TAG, "Error processing media item", e)
                }
            }
        }
        return MediaScanResult(totalScanned, newItemsInserted, 0)
    }

    private fun computeHashFromContentUri(contentUri: android.net.Uri): String {
        try {
            val digest = MessageDigest.getInstance("SHA-256")
            var totalBytesRead = 0L
            val buffer = ByteArray(8192)

            contentResolver.openInputStream(contentUri)?.use { input ->
                var read: Int
                while (input.read(buffer).also { read = it } > 0 && totalBytesRead < 1_048_576) { 
                    digest.update(buffer, 0, read)
                    totalBytesRead += read
                }
            }

            return digest.digest().joinToString("") { "%02x".format(it) }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to compute hash for $contentUri", e)
            return "hash_failed_${contentUri.lastPathSegment}_${System.currentTimeMillis()}"
        }
    }

    suspend fun cleanupDeletedFiles() {
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, 
                   com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED,
                   com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADING)
        )

        var deletedCount = 0
        for (item in pendingItems) {
            try {
                val uri = android.net.Uri.parse(item.uri)
                contentResolver.openInputStream(uri)?.use { } ?: run {
                    mediaItemDao.updateStatus(item.id, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED, "File not accessible")
                    deletedCount++
                }
            } catch (e: Exception) {
                mediaItemDao.updateStatus(item.id, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED, "File not accessible")
                deletedCount++
            }
        }
    }
}
