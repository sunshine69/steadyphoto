package com.steadyphoto.sync.worker

import android.os.FileObserver as AndroidFileObserver
import android.util.Log
import java.io.File

// Alias FileObserver constants to avoid fully qualified names in Kotlin
private const val CREATE = AndroidFileObserver.CREATE
private const val MODIFY = AndroidFileObserver.MODIFY  // Android uses MODIFY, not MODIFIED
private const val DELETE = AndroidFileObserver.DELETE
private const val MOVED_FROM = AndroidFileObserver.MOVED_FROM
private const val MOVED_TO = AndroidFileObserver.MOVED_TO

/**
 * FileObserver that monitors directories for real-time filesystem changes.
 * 
 * Unlike ContentObserver (which depends on MediaStore), FileObserver detects
 * file creation/modification at the filesystem level - within milliseconds,
 * not seconds or minutes later. This is critical because:
 * 1. Files downloaded via USB/MTP are written directly to disk before MediaStore indexes them
 * 2. Other apps creating files may not immediately update MediaStore
 * 3. Even when ContentObserver fires, it's after the media scanner has already indexed the file
 * 
 * FileObserver catches files as soon as they're written to disk, allowing us to
 * start syncing immediately rather than waiting for MediaStore events.
 */
class DirectoryFileObserver(
    private val path: String,
    private val onDirectoryChanged: (file: File) -> Unit = {}
) : AndroidFileObserver(path, CREATE or MODIFY or MOVED_FROM or MOVED_TO) {

    companion object {
        private const val TAG = "DirectoryFileObserver"
        
        // Key directories to monitor for new media files
        val MONITORED_DIRS = listOf(
            "/storage/emulated/0/DCIM",           // Camera photos/videos (Android 13+ scoped)
            "/storage/emulated/0/Pictures/Camera", // Camera directory on some devices  
            "/storage/emulated/0/Download",       // Downloads folder
            "/storage/emulated/0/Movies",         // Video downloads
        )

        // Filter to only watch image and video files (not thumbnails, temp files, etc.)
        private val MEDIA_EXTENSIONS = setOf(
            "jpg", "jpeg", "png", "heic", "heif",  // Images
            "mp4", "mov", "mkv", "avi", "webm"      // Videos
        )

        private fun isMediaFile(file: File): Boolean {
            return MEDIA_EXTENSIONS.contains(file.extension.lowercase())
        }
    }

    override fun onEvent(event: Int, path: String?) {
        if (path == null) return
        
        Log.d(TAG, "File event detected in $path: ${getEventTypeString(event)}")
        
        val file = File(this.path, path)
        
        // Only process files that exist and are media files
        if (!file.exists()) {
            Log.d(TAG, "File no longer exists: $path - likely a transient rename event")
            return
        }
        
        if (!isMediaFile(file)) {
            Log.d(TAG, "Not a media file (extension: ${file.extension}), ignoring: $path")
            return
        }
        
        // Only trigger sync for actual file creation/modification events
        // Ignore MOVED_FROM as it's just the source side of a rename
        when (event) {
            CREATE, MODIFY -> {
                Log.d(TAG, "New media file detected: ${file.absolutePath}")
                onDirectoryChanged(file)
            }
            MOVED_TO -> {
                // File was moved/renamed to this location - could be a copy operation
                Log.d(TAG, "File moved here: ${file.absolutePath}")
                onDirectoryChanged(file)
            }
        }
    }

    private fun getEventTypeString(event: Int): String {
        return when (event) {
            CREATE -> "CREATE"
            DELETE -> "DELETE"
            MODIFY -> "MODIFY"  // Changed from MODIFIED to MODIFY
            MOVED_FROM -> "MOVED_FROM"
            MOVED_TO -> "MOVED_TO"
            else -> "UNKNOWN($event)"
        }
    }

    /**
     * Start observing the directory. Call this after construction.
     */
    override fun startWatching() {
        Log.d(TAG, "Starting FileObserver for: $path")
        super.startWatching()
    }

    /**
     * Stop observing the directory. Call when service is destroyed.
     */
    override fun stopWatching() {
        Log.d(TAG, "Stopping FileObserver for: $path")
        super.stopWatching()
    }
}
