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
        
        // MIME types we consider as valid media - used for initial filtering on non-force-scan
        private val IMAGE_MIME_TYPES = arrayOf("image/jpeg", "image/png", "image/heic")
        private val VIDEO_MIME_TYPES = arrayOf("video/mp4", "video/quicktime", "video/x-ms-wmv")
        
        // File extensions we consider as valid media - used when MIME type is missing/wrong (e.g., downloaded files)
        private val IMAGE_EXTENSIONS = setOf(".jpg", ".jpeg", ".png", ".heic", ".heif", ".webp", ".gif")
        private val VIDEO_EXTENSIONS = setOf(".mp4", ".mov", ".wmv", ".avi", ".mkv")
    }

    /**
     * Scan the device for new media files and add them to the database if not already present.
     * 
     * @param forceFullScan If true, scan ALL rows from MediaStore without MIME type filtering.
     *                      This is critical for catching downloaded files that may have wrong/missing MIME types.
     */
    suspend fun scanForNewMedia(forceFullScan: Boolean = false): MediaScanResult {
        var totalScanned = 0
        var newItemsInserted = 0

        if (forceFullScan) {
            // UNIFIED FULL SCAN: Query EVERYTHING in external storage via the Files provider.
            // This is the "nuclear option" to catch files that Android hasn't correctly categorized yet.
            Log.d(TAG, "Running unified full MediaStore scan for ALL media types")
            
            val projection = arrayOf(
                MediaStore.MediaColumns._ID, 
                MediaStore.MediaColumns.DISPLAY_NAME, 
                MediaStore.MediaColumns.MIME_TYPE, 
                MediaStore.MediaColumns.SIZE
            )

            // We query the generic Files provider which includes Images AND Videos + other files.
            val unifiedCursor = contentResolver.query(
                MediaStore.Files.getContentUri("external"),
                projection,
                "${MediaStore.MediaColumns.SIZE} > 1024", // Skip tiny system/metadata files (<1KB)
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

        } else {
            // Normal scan - filter by MIME type only (faster but may miss downloaded files with wrong MIME types)
            val imageResult = scanMediaType(MediaStore.Images.Media.EXTERNAL_CONTENT_URI, IMAGE_MIME_TYPES)
            totalScanned += imageResult.totalScanned
            newItemsInserted += imageResult.newItemsInserted
            
            val videoResult = scanMediaType(MediaStore.Video.Media.EXTERNAL_CONTENT_URI, VIDEO_MIME_TYPES)
            totalScanned += videoResult.totalScanned
            newItemsInserted += videoResult.newItemsInserted

            // Secondary check for files that might have correct extensions but were missed by MIME filtering
            Log.d(TAG, "Running secondary extension-based scan for downloaded media")
            val fileCursor = contentResolver.query(
                MediaStore.Files.getContentUri("external"),
                projectionForScan(), 
                "${MediaStore.MediaColumns.SIZE} > 10240", // Only files > 10KB to avoid tiny system files
                null,
                null
            )

            fileCursor?.use { cursor ->
                val result = scanAllRows(cursor)
                totalScanned += result.totalScanned
                newItemsInserted += result.newItemsInserted
            } ?: run {
                Log.e(TAG, "Failed to query MediaStore for external files during normal scan")
            }
        }

        Log.d(TAG, "Scan complete: $totalScanned scanned, $newItemsInserted inserted")
        
        // Calculate how many items were already in the DB (including newly inserted ones) 
        // to help with reporting/logic elsewhere if needed.
        val pendingItems = mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )
        
        // duplicatesSkipped logic: if we scanned everything and found nothing new to insert, 
        // it means all files were already in the DB or aren't media.
        val duplicatesSkipped = maxOf(0, pendingItems.size - (newItemsInserted)) // simplified for this context

        return MediaScanResult(totalScanned, newItemsInserted, duplicatesSkipped)
    }

    private fun projectionForScan() = arrayOf(
            MediaStore.MediaColumns._ID, 
            MediaStore.MediaColumns.DISPLAY_NAME, 
            MediaStore.MediaColumns.MIME_TYPE, 
            MediaStore.MediaColumns.SIZE
    )

    /**
     * Scan ALL rows from a cursor without MIME type filtering.
     */
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

                    // IDENTIFICATION LOGIC: 
                    // Is this actually a media file? Check MIME type OR extension.
                    val isImage = (mimeType?.startsWith("image/") == true) || isImageByExtension(fileName)
                    val isVideo = (mimeType?.startsWith("video/") == true) || isVideoByExtension(fileName)

                    if (!isImage && !isVideo) {
                        // It's a file, but not an image or video. Skip it.
                        continue 
                    }

                    // Construct URI for this media item using the generic MediaStore pattern.
                    // Note: Using Files provider works for all types identified above.
                    val contentUri = MediaStore.Files.getContentUri("external").buildUpon()
                        .appendPath(id.toString())
                        .build()

                    // Check if already in database by hash or URI to avoid duplicates
                    val existingByHash = mediaItemDao.getByHash(contentUri.toString())
                    if (existingByHash != null && existingByHash.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED) {
                        continue
                    }

                    // Compute hash for deduplication using Content URI (first 1MB only)
                    val hash = computeHashFromContentUri(contentUri)
                    
                    // Check if file with same hash exists in DB (even different path - could be copy)
                    val existingByHashValue = mediaItemDao.getByHash(hash)
                    if (existingByHashValue != null && existingByHashValue.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED) {
                        continue
                    }

                    // Add to database with PENDING status
                    val entity = com.steadyphoto.sync.data.local.entity.MediaItemEntity(
                        uri = contentUri.toString(),
                        localPath = null, 
                        fileName = fileName,
                        hash = hash,
                        mimeType = mimeType ?: if (isImage) "image/jpeg" else "video/mp4", // Fallback MIME types
                        fileSize = fileSize,
                        uploadStatus = com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                    )

                    mediaItemDao.insert(entity)
                    newItemsInserted++
                    Log.d(TAG, "Added new file to sync queue: $fileName ($fileSize bytes)")

                } catch (e: Exception) {
                    Log.w(TAG, "Error processing media item", e)
                }
            }
        }

        return MediaScanResult(totalScanned, newItemsInserted, 0)
    }

    private fun isImageByExtension(fileName: String): Boolean = IMAGE_EXTENSIONS.contains(fileName.substringAfterLast('.', "").lowercase())
    private fun isVideoByExtension(fileName: String): Boolean = VIDEO_EXTENSIONS.contains(fileName.substringAfterLast('.', "").lowercase())

    /**
     * Scan a specific media type (images or videos).
     */
    private suspend fun scanMediaType(uri: android.net.Uri, mimeTypes: Array<String>): MediaScanResult {
        var totalScanned = 0
        var newItemsInserted = 0
        
        val mimeTypeColumn = MediaStore.MediaColumns.MIME_TYPE
        // Images and Videos have different _ID columns in some Android versions/providers, but standardizing on MediaColumns._ID is safer when using Files provider context
        val idColumnName = if (uri == MediaStore.Images.Media.EXTERNAL_CONTENT_URI) {
            MediaStore.Images.ImageColumns._ID 
        } else {
            MediaStore.Video.VideoColumns._ID
        }
        
        // Safety: fallback to standard _ID if the specific one fails
        val finalIdColumn = try { idColumnName } catch (e: Exception) { MediaStore.MediaColumns._ID }

        val mimeTypeSelection = "$mimeTypeColumn IN (${mimeTypes.joinToString(",") { "'$it'" }})"
        
        contentResolver.query(
            uri,
            arrayOf(finalIdColumn, MediaStore.MediaColumns.DISPLAY_NAME, mimeTypeColumn, MediaStore.MediaColumns.SIZE),
            mimeTypeSelection + " AND ${MediaStore.MediaColumns.SIZE} > 0",
            null,
            null
        )?.use { cursor ->
            while (cursor.moveToNext()) {
                try {
                    totalScanned++
                    val id = cursor.getLong(cursor.getColumnIndexOrThrow(finalIdColumn))
                    val fileName = cursor.getString(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.DISPLAY_NAME)) ?: continue
                    val mimeType = cursor.getString(cursor.getColumnIndexOrThrow(mimeTypeColumn)) ?: continue
                    val fileSize = cursor.getLong(cursor.getColumnIndexOrThrow(MediaStore.MediaColumns.SIZE))

                    val contentUri = uri.buildUpon().appendPath(id.toString()).build()

                    val existingByHash = mediaItemDao.getByHash(contentUri.toString())
                    if (existingByHash != null && existingByHash.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED) {
                        continue
                    }

                    val hash = computeHashFromContentUri(contentUri)
                    val existingByHashValue = mediaItemDao.getByHash(hash)
                    if (existingByHashValue != null && existingByHashValue.uploadStatus == com.steadyphoto.sync.data.local.entity.UploadStatus.UPLOADED) {
                        continue
                    }

                    val entity = com.steadyphoto.sync.data.local.entity.MediaItemEntity(
                        uri = contentUri.toString(),
                        localPath = null,
                        fileName = fileName,
                        hash = hash,
                        mimeType = mimeType,
                        fileSize = fileSize,
                        uploadStatus = com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                    )

                    mediaItemDao.insert(entity)
                    newItemsInserted++
                } catch (e: Exception) {
                    Log.w(TAG, "Error processing media item", e)
                }
            }
        } ?: run {
            Log.e(TAG, "Failed to query MediaStore for $uri")
        }

        return MediaScanResult(totalScanned, newItemsInserted, 0)
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
            return contentUri.toString() // Fallback to URI string if hashing fails
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

        if (deletedCount > 0) {
            Log.d(TAG, "Cleaned up $deletedCount deleted files from database")
        }
    }
}
