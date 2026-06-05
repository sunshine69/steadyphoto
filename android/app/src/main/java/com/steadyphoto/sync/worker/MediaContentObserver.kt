package com.steadyphoto.sync.worker

import android.content.ContentResolver
import android.database.ContentObserver
import android.net.Uri
import android.os.Handler
import android.os.Looper
import android.provider.MediaStore
import android.util.Log
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.launch

/**
 * ContentObserver that monitors MediaStore for real-time changes.
 * 
 * When new media files are added, removed, or modified on the device,
 * this observer detects them immediately (within milliseconds) and triggers
 * a sync operation instead of waiting for the periodic polling loop.
 */
class MediaContentObserver(
    private val handler: Handler = Handler(Looper.getMainLooper()),
    private val coroutineScope: CoroutineScope,
    private val onMediaChange: suspend () -> Unit
) : ContentObserver(handler) {

    companion object {
        private const val TAG = "MediaContentObserver"
        
        // URIs to observe - images and videos
        private val OBSERVED_URIS = listOf(
            MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
            MediaStore.Video.Media.EXTERNAL_CONTENT_URI
        )
    }

    private var isObserving = false
    
    // Track last change time to debounce rapid-fire events  
    private var lastChangeTime = 0L
    private val debounceDelayMs = 2_000L // Wait 2 seconds after changes before triggering sync

    override fun onChange(selfChange: Boolean, uri: Uri?) {
        super.onChange(selfChange, uri)
        
        if (uri == null) return
        
        Log.d(TAG, "Media change detected: $uri")
        
        val now = System.currentTimeMillis()
        
        // Debounce - only trigger if enough time has passed since last change
        if (now - lastChangeTime < debounceDelayMs) {
            Log.d(TAG, "Ignoring rapid-fire event (debouncing)")
            return
        }
        
        lastChangeTime = now
        
        // Trigger sync after debounce period using the provided coroutine scope (non-blocking)
        coroutineScope.launch(Dispatchers.Default) {
            delay(debounceDelayMs)
            onMediaChange()
        }
    }

    /**
     * Register this observer with the ContentResolver.
     */
    fun register(contentResolver: ContentResolver, notifyForDescendants: Boolean = true) {
        if (isObserving) return
        
        OBSERVED_URIS.forEach { uri ->
            contentResolver.registerContentObserver(
                uri,
                notifyForDescendants, // Catch changes in subdirectories too
                this
            )
        }
        
        isObserving = true
        Log.d(TAG, "Registered ContentObserver for ${OBSERVED_URIS.size} URIs")
    }

    /**
     * Unregister this observer from the ContentResolver.
     */
    fun unregister(contentResolver: ContentResolver) {
        if (!isObserving) return
        
        OBSERVED_URIS.forEach { uri ->
            try {
                contentResolver.unregisterContentObserver(this)
            } catch (e: IllegalArgumentException) {
                // Already unregistered, ignore
                Log.w(TAG, "ContentObserver already unregistered for $uri")
            }
        }
        
        isObserving = false
        Log.d(TAG, "Unregistered ContentObserver")
    }

    /**
     * Check if currently observing. Useful for debugging and testing.
     */
    fun isCurrentlyObserving(): Boolean = isObserving
}
