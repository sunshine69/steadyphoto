package com.steadyphoto.sync.worker

import android.content.ContentResolver
import android.media.MediaScannerConnection
import android.util.Log
import java.io.File

/**
 * Utility class that uses MediaScannerConnection to force-index new media files.
 * 
 * When a file is downloaded by another app (Chrome, email), Android's built-in
 * media scanner may take 30+ seconds or never index it at all, especially for
 * non-standard locations like /Download/ with custom MIME types.
 * 
 * By calling MediaScannerConnection.scanFile() immediately after detecting a new file,
 * we force the system to register it in MediaStore RIGHT AWAY, which then causes
 * ContentObserver to fire reliably. This is critical for catching downloaded files
 * that would otherwise be missed by ContentObserver.
 */
object ForceMediaIndexer {

    private const val TAG = "ForceMediaIndexer"
    
    // Track recently indexed files to avoid duplicate indexing attempts
    private val lastIndexedFiles = mutableSetOf<String>()
    private var lastScanTime = 0L
    private val debounceDelayMs = 5_000L // Only scan every 5 seconds max
    
    // File extensions that should be force-indexed as media (even if MIME type is wrong)
    private val MEDIA_EXTENSIONS = setOf(
        ".jpg", ".jpeg", ".png", ".heic", ".heif",  // Images
        ".mp4", ".mov", ".wmv"                       // Videos
    )

    /**
     * Force-index a single file by calling MediaScannerConnection.scanFile().
     * This will register the file in MediaStore immediately, causing ContentObserver to fire.
     */
    fun forceIndexFile(context: android.content.Context, filePath: String) {
        val now = System.currentTimeMillis()
        
        // Debounce - don't try to index more than once every 5 seconds
        if (now - lastScanTime < debounceDelayMs) return
        
        // Check if we've already indexed this file recently
        if (lastIndexedFiles.contains(filePath)) {
            Log.d(TAG, "Already indexed: $filePath")
            return
        }
        
        val file = File(filePath)
        if (!file.exists()) {
            Log.d(TAG, "File doesn't exist yet, skipping: $filePath")
            return
        }
        
        // Only index media files (by extension) - skip non-media files like PDFs, etc.
        val ext = file.extension.lowercase()
        if (!MEDIA_EXTENSIONS.any { ext.endsWith(it) }) {
            Log.d(TAG, "Not a media file, skipping: $filePath")
            return
        }
        
        lastScanTime = now
        lastIndexedFiles.add(filePath)
        
        // Keep the set from growing too large - remove entries older than 1 hour
        if (lastIndexedFiles.size > 100) {
            lastIndexedFiles.clear()
        }
        
        Log.d(TAG, "Force-indexing file: $filePath")
        
        val uri = android.net.Uri.parse("file://$filePath")
        MediaScannerConnection.scanFile(
            context,
            arrayOf(filePath),
            null // Let system determine MIME type from extension
        ) { path, scannedUri ->
            if (scannedUri != null) {
                Log.d(TAG, "Successfully force-indexed file: $filePath ($scannedUri)")
            } else {
                Log.w(TAG, "Failed to force-index file: $filePath")
            }
        }
    }

    /**
     * Force-index all files in a directory that haven't been indexed yet.
     * Useful when we detect a directory-level event (like MEDIA_SCANNER_FINISHED).
     */
    fun forceIndexDirectory(context: android.content.Context, dirPath: String) {
        val now = System.currentTimeMillis()
        
        // Debounce - don't scan directories more than once every 5 seconds
        if (now - lastScanTime < debounceDelayMs) return
        
        Log.d(TAG, "Force-indexing directory: $dirPath")
        
        val dir = File(dirPath)
        if (!dir.exists() || !dir.isDirectory) {
            Log.w(TAG, "Directory doesn't exist or isn't a directory: $dirPath")
            return
        }
        
        lastScanTime = now
        
        // Get all files in the directory that are media files and haven't been indexed yet
        val mediaFiles = dir.listFiles { _, name ->
            MEDIA_EXTENSIONS.any { ext -> name.lowercase().endsWith(ext) }
        } ?: emptyArray()
        
        if (mediaFiles.isEmpty()) return
        
        Log.d(TAG, "Found ${mediaFiles.size} media files in $dirPath to index")
        
        // Filter out already-indexed files
        val filesToIndex = mediaFiles.filter { file ->
            !lastIndexedFiles.contains(file.absolutePath)
        }
        
        if (filesToIndex.isEmpty()) return
        
        Log.d(TAG, "Indexing ${filesToIndex.size} new files: $dirPath")
        
        // Group by extension to batch similar MIME types together for efficiency
        val groupedByExt = filesToIndex.groupBy { it.extension.lowercase() }
        
        for ((ext, files) in groupedByExt) {
            val mimeTypes = when (ext) {
                ".jpg", ".jpeg" -> "image/jpeg"
                ".png" -> "image/png"
                ".heic", ".heif" -> "image/heic"
                ".mp4" -> "video/mp4"
                ".mov" -> "video/quicktime"
                ".wmv" -> "video/x-ms-wmv"
                else -> null
            }
            
            val paths = files.map { it.absolutePath }.toTypedArray()
            
            MediaScannerConnection.scanFile(
                context,
                paths,
                arrayOf(mimeTypes) // Match each path with its MIME type
            ) { path, scannedUri ->
                Log.d(TAG, "Batch indexed ${paths.size} files in $dirPath")
            }
        }
    }

    /**
     * Clear the last-indexed-files cache. Useful for testing or when we need to re-scan everything.
     */
    fun clearIndexedCache() {
        lastIndexedFiles.clear()
        Log.d(TAG, "Cleared indexed files cache")
    }
}
