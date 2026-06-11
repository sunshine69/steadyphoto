package com.steadyphoto.sync.worker

import android.os.FileObserver as AndroidFileObserver
import android.util.Log
import java.io.File

class DirectoryFileObserver(
    private val path: String,
    private val onDirectoryChanged: (file: File) -> Unit = {}
) : AndroidFileObserver(path, FileObserverConstants.CREATE or FileObserverConstants.MODIFY or FileObserverConstants.MOVED_FROM or FileObserverConstants.MOVED_TO) {

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
            FileObserverConstants.CREATE, FileObserverConstants.MODIFY -> {
                Log.d(TAG, "New media file detected: ${file.absolutePath}")
                onDirectoryChanged(file)
            }
            FileObserverConstants.MOVED_TO -> {
                // File was moved/renamed to this location - could be a copy operation
                Log.d(TAG, "File moved here: ${file.absolutePath}")
                onDirectoryChanged(file)
            }
        }
    }

    private fun getEventTypeString(event: Int): String {
        return when (event) {
            FileObserverConstants.CREATE -> "CREATE"
            FileObserverConstants.DELETE -> "DELETE"
            FileObserverConstants.MODIFY -> "MODIFY"  // Changed from MODIFIED to MODIFY
            FileObserverConstants.MOVED_FROM -> "MOVED_FROM"
            FileObserverConstants.MOVED_TO -> "MOVED_TO"
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
