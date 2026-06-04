package com.steadyphoto.sync.data.repository

import android.content.ContentResolver
import android.provider.MediaStore
import android.util.Log
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
        // Non-const arrays since Array<String> can't be const in Kotlin
        private val IMAGE_MIME_TYPES = arrayOf("image/jpeg", "image/png", "image/heic")
        private val VIDEO_MIME_TYPES = arrayOf("video/mp4", "video/quicktime", "video/x-ms-wmv")
    }

    /**
     * Scan the device for new media files and add them to the database if not already present.
     */
    suspend fun scanForNewMedia(): Int {
        var addedCount = 0

        // Scan images
        addedCount += scanMediaType(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, IMAGE_MIME_TYPES)

        // Scan videos
        addedCount += scanMediaType(MediaStore.Video.Media.EXTERNAL_CONTENT_URI, VIDEO_MIME_TYPES)

        Log.d(TAG, "Scan complete: $addedCount items need uploading")
        return addedCount
    }

    /**
     * Scan a specific media type (images or videos).
     */
    private suspend fun scanMediaType(uri: android.net.Uri, mimeTypes: Array<String>): Int {
        var count = 0
        
        // Build selection for MIME types - need to handle each URI differently
        val mimeTypeColumn = MediaStore.MediaColumns.MIME_TYPE
        
        // For images, use the correct ID column name
        val idColumn = if (uri == MediaStore.Images.Media.EXTERNAL_CONTENT_URI) {
            MediaStore.Images.ImageColumns._ID
        } else {
            MediaStore.Video.VideoColumns._ID
        }
        
        // Build MIME type selection: "mime_type IN ('image/jpeg', 'image/png')" OR "mime_type IN (...)"
        val mimeTypeSelection = "$mimeTypeColumn IN (${mimeTypes.joinToString(",") { "'$it'" }})"
        
        // Query with the correct ID column name for each URI type
        contentResolver.query(
            uri,
            arrayOf(idColumn, MediaStore.MediaColumns.DISPLAY_NAME, mimeTypeColumn, MediaStore.MediaColumns.SIZE),
            mimeTypeSelection + " AND ${MediaStore.MediaColumns.SIZE} > 0",
            null,
            null
        )?.use { cursor ->
            while (cursor.moveToNext()) {
                try {
                    val id = cursor.getLong(cursor.getColumnIndexOrThrow(idColumn))
                    val fileName = cursor.getString(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.DISPLAY_NAME)) ?: continue
                    val mimeType = cursor.getString(cursor.getColumnIndexOrThrow(mimeTypeColumn)) ?: continue
                    val fileSize = cursor.getLong(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.SIZE))

                    // Build Content URI - this works on Android 10+ (scoped storage)
                    val contentUri = uri.buildUpon().appendPath(id.toString()).build()

                    // Check if already in database (by hash or URI)
                    val existingByHash = mediaItemDao.getByHash(contentUri.toString())
                    if (existingByHash != null && existingByHash.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED) {
                        Log.d(TAG, "Skipping already uploaded file: $fileName")
                        continue
                    }

                    // Compute hash for deduplication using Content URI
                    val hash = computeHashFromContentUri(contentUri)
                    
                    // Check if file with same hash exists in DB (even different path - could be copy)
                    val existingByHashValue = mediaItemDao.getByHash(hash)
                    if (existingByHashValue != null && existingByHashValue.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED) {
                        Log.d(TAG, "Skipping duplicate file by hash: $fileName")
                        continue
                    }

                    // Add to database with PENDING status - use Content URI as primary access method
                    val entity = com.steadyphoto.sync.data.local.entity.MediaItemEntity(
                        uri = contentUri.toString(),
                        localPath = null, // DATA column returns null on Android 10+
                        fileName = fileName,
                        hash = hash,
                        mimeType = mimeType,
                        fileSize = fileSize,
                        uploadStatus = com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                    )

                    mediaItemDao.insert(entity)
                    count++
                    Log.d(TAG, "Added new file to sync queue: $fileName ($fileSize bytes)")

                } catch (e: Exception) {
                    Log.w(TAG, "Error processing media item", e)
                }
            }
        } ?: run {
            Log.e(TAG, "Failed to query MediaStore for $uri")
        }

        return count
    }

    /**
     * Compute SHA-256 hash of file content using Content URI.
     */
    private fun computeHashFromContentUri(contentUri: android.net.Uri): String {
        try {
            val digest = MessageDigest.getInstance("SHA-256")
            var bytesRead = 0L
            val buffer = ByteArray(8192)

            contentResolver.openInputStream(contentUri)?.use { input ->
                while (input.read(buffer).also { bytesRead += it } > 0 && bytesRead < 1_048_576) { // Read first 1MB only for performance
                    digest.update(buffer, 0, bytesRead.toInt())
                }
            }

            return digest.digest().joinToString("") { "%02x".format(it) }
        } catch (e: Exception) {
            Log.w(TAG, "Failed to compute hash for $contentUri", e)
            // Return URI as fallback - less accurate but prevents duplicates in same location
            return contentUri.toString()
        }
    }

    /**
     * Remove files that no longer exist on disk from the database.
     */
    suspend fun cleanupDeletedFiles() {
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )

        var deletedCount = 0
        for (item in pendingItems) {
            // Try to access the file via Content URI - if it fails, mark as failed
            try {
                val uri = android.net.Uri.parse(item.uri)
                contentResolver.openInputStream(uri)?.use { } ?: run {
                    mediaItemDao.updateStatus(item.id, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED, "File not accessible")
                    deletedCount++
                    Log.d(TAG, "Removed inaccessible file from sync queue: ${item.fileName}")
                }
            } catch (e: Exception) {
                mediaItemDao.updateStatus(item.id, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED, "File not accessible")
                deletedCount++
                Log.d(TAG, "Removed inaccessible file from sync queue: ${item.fileName}")
            }
        }

        if (deletedCount > 0) {
            Log.d(TAG, "Cleaned up $deletedCount deleted files from database")
        }
    }

    /**
     * Get all pending media items that need to be uploaded.
     */
    suspend fun getPendingMediaItems(): List<com.steadyphoto.sync.data.local.entity.MediaItemEntity> {
        return mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )
    }

    /**
     * Get count of pending items.
     */
    suspend fun getPendingCount(): Int {
        return mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        ).size
    }

    /**
     * Get count of uploaded items.
     */
    suspend fun getUploadedCount(): Int {
        return mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED)
        ).size
    }
}
