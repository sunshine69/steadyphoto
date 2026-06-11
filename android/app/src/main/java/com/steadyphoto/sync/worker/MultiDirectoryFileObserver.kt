package com.steadyphoto.sync.worker

import android.os.FileObserver as JavaFileObserver
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.io.File

class MultiDirectoryFileObserver(
    private val onNewFile: (file: String) -> Unit = {}
) : CoroutineScope by CoroutineScope(SupervisorJob() + Dispatchers.IO) {

    companion object {
        private const val TAG = "MultiDirFileObserver"
        
        // Directories to watch - common places where media files appear
        val OBSERVED_DIRS = listOf(
            "/storage/emulated/0/Download/",      // Browser downloads, email attachments
            "/storage/emulated/0/Pictures/",       // App pictures
            "/storage/emulated/0/DCIM/Camera/",    // Camera photos (backup from camera app)
            "/storage/emulated/0/Music/",          // Music files
            "/storage/emulated/0/Android/data/"    // Some apps save here
        )
        
        // Track which directories we're observing to avoid duplicate watches
        private val observedDirs = mutableSetOf<String>()
    }

    private var observers: List<JavaFileObserver> = emptyList()

    /**
     * Start watching all configured directories.
     */
    fun startWatching() {
        if (observers.isNotEmpty()) return
        
        observers = OBSERVED_DIRS.mapNotNull { dir ->
            try {
                // Check directory exists and is accessible
                val fileDir = File(dir)
                if (!fileDir.exists() || !fileDir.isDirectory) {
                    Log.d(TAG, "Directory doesn't exist or isn't a directory: $dir")
                    return@mapNotNull null
                }
                
                if (observedDirs.contains(dir)) {
                    Log.d(TAG, "Already watching: $dir")
                    return@mapNotNull null
                }
                
                val observer = object : JavaFileObserver(dir) {
                    override fun onEvent(event: Int, path: String?) {
                        if (path == null) return
                        
                        // Only process create and modify events
                        if ((event and FileObserverConstants.EVENTS_MASK) != 0 && event != FileObserverConstants.CLOSE_WRITE) {
                            val eventType = when (event and 0xFF) {
                                FileObserverConstants.CREATE -> "CREATE"
                                FileObserverConstants.MODIFY -> "MODIFY"  // Changed from MODIFIED to MODIFY
                                else -> "UNKNOWN($event)"
                            }
                            
                            // Ignore directories, only care about files
                            val fullPath = "$dir$path"
                            if (!File(fullPath).isFile) return
                            
                            Log.d(TAG, "New file detected: $fullPath ($eventType)")
                            
                            // Debounce - wait 3 seconds to let the download complete
                            CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
                                Thread.sleep(3_000L)
                                onNewFile(fullPath)
                            }
                        }
                    }
                }
                
                observer.startWatching()
                observedDirs.add(dir)
                Log.d(TAG, "Started watching: $dir")
                observer
                
            } catch (e: Exception) {
                Log.w(TAG, "Failed to watch directory $dir", e)
                null
            }
        }
        
        Log.i(TAG, "Watching ${observers.size} directories for new files")
    }

    /**
     * Stop watching all directories.
     */
    fun stopWatching() {
        observers.forEach { observer ->
            try {
                observer.stopWatching()
            } catch (e: Exception) {
                Log.w(TAG, "Failed to stop watching", e)
            }
        }
        
        OBSERVED_DIRS.forEach { dir -> observedDirs.remove(dir) }
        observers = emptyList()
        Log.d(TAG, "Stopped watching all directories")
    }

    /**
     * Check if currently watching any directory.
     */
    fun isCurrentlyWatching(): Boolean = observers.isNotEmpty()
}
