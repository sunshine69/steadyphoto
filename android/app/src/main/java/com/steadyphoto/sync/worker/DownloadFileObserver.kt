package com.steadyphoto.sync.worker

import android.os.FileObserver as AndroidFileObserver
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import java.io.File

@Suppress("DEPRECATION")
class DownloadFileObserver(
@Suppress("DEPRECATION")
    private val onNewFile: (String) -> Unit
) : AndroidFileObserver("/storage/emulated/0/Download/", FileObserverConstants.CREATE or FileObserverConstants.MODIFY) {

    // Track whether we're currently watching (isAlive() from Java FileObserver is not accessible in Kotlin)
    private var _isWatching = false
    
    companion object {
        private const val TAG = "DownloadFileObserver"
        
        // Events we care about - only create and modify events for files
        
        // Track which directories we're observing to avoid duplicate watches
        private val observedDirs = mutableSetOf<String>()
    }

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
            if (!File(path).isFile) return
            
            Log.d(TAG, "New file detected in Downloads: $path ($eventType)")
            
            // Debounce - don't trigger for every tiny write event
            CoroutineScope(SupervisorJob() + Dispatchers.IO).launch {
                Thread.sleep(3_000L) // Wait 3 seconds to let the download complete
                onNewFile(path)
            }
        }
    }

    /**
     * Start watching to ensure we don't create duplicate observers.
     */
    override fun startWatching() {
        val dirPath = "/storage/emulated/0/Download/"
        if (observedDirs.contains(dirPath)) {
            Log.d(TAG, "Already watching Downloads directory")
            return
        }
        
        super.startWatching()
        observedDirs.add(dirPath)
        _isWatching = true
        Log.d(TAG, "Started observing Downloads directory: $dirPath")
    }

    /**
     * Stop watching to clean up our tracking set.
     */
    override fun stopWatching() {
        val dirPath = "/storage/emulated/0/Download/"
        observedDirs.remove(dirPath)
        
        super.stopWatching()
        _isWatching = false
        Log.d(TAG, "Stopped observing Downloads directory")
    }

    /**
     * Check if currently watching. Useful for debugging and testing.
     */
    fun isCurrentlyWatching(): Boolean = _isWatching

}
