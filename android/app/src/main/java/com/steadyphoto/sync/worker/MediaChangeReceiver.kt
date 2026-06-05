package com.steadyphoto.sync.worker

import android.content.BroadcastReceiver
import android.content.Context
import android.content.Intent
import android.util.Log

/**
 * BroadcastReceiver that listens for media-related system events.
 * 
 * Triggers when:
 * - Android's MediaScanner starts/finishes scanning new files (common after USB/MTP downloads)  
 * - Device boots up (to catch any media added while device was off)
 * - External storage is mounted/unmounted (USB connection changes)
 */
class MediaChangeReceiver : BroadcastReceiver() {

    companion object {
        private const val TAG = "MediaChangeReceiver"
        
        // Intent actions for media scanner events - use the standard Android string constants
        const val ACTION_MEDIA_SCANNER_STARTED = "android.intent.action.MEDIA_SCANNER_STARTED"
        const val ACTION_MEDIA_SCANNER_FINISHED = "android.intent.action.MEDIA_SCANNER_FINISHED"
    }

    override fun onReceive(context: Context, intent: Intent) {
        when (intent.action) {
            ACTION_MEDIA_SCANNER_STARTED -> {
                Log.d(TAG, "Media scanner started - files being scanned")
                // Start scanning immediately when system starts scanning
                startSyncServiceIfNeeded(context)
            }
            ACTION_MEDIA_SCANNER_FINISHED -> {
                Log.d(TAG, "Media scanner finished - new files available")
                // Trigger sync after system completes scanning - this is the key event for USB downloads!
                startSyncServiceIfNeeded(context)
            }
            Intent.ACTION_EXTERNAL_APPLICATIONS_AVAILABLE -> {
                val changed = intent.getStringArrayExtra(
                    Intent.EXTRA_CHANGED_PACKAGE_LIST
                )
                if (changed != null && changed.isNotEmpty()) {
                    Log.d(TAG, "External apps available - packages: ${changed.joinToString()}")
                    startSyncServiceIfNeeded(context)
                }
            }
            Intent.ACTION_EXTERNAL_APPLICATIONS_UNAVAILABLE -> {
                Log.d(TAG, "External apps unavailable - storage may have been unmounted")
                // Don't trigger sync on unmount since files might be corrupted mid-transfer
            }
            Intent.ACTION_BOOT_COMPLETED -> {
                Log.d(TAG, "Device booted up - checking for media changes")
                startSyncServiceIfNeeded(context)
            }
            else -> {
                // Check for other media-related intents
                if (intent.action?.startsWith("android.intent.action.") == true && 
                    intent.action?.contains("MEDIA") == true) {
                    Log.d(TAG, "Media action detected: ${intent.action}")
                    startSyncServiceIfNeeded(context)
                }
            }
        }
    }

    private fun startSyncServiceIfNeeded(context: Context) {
        // Start the foreground service if it's not already running
        val intent = SyncService.newIntent(context).apply {
            action = SyncService.ACTION_START_SYNC
        }
        
        try {
            context.startForegroundService(intent)
            Log.d(TAG, "SyncService started due to media change event")
        } catch (e: SecurityException) {
            // Permission denied - likely doze mode or battery optimization blocking it
            Log.w(TAG, "Failed to start SyncService - permission denied", e)
        } catch (e: Exception) {
            // Service might already be running or other issue
            Log.w(TAG, "Failed to start SyncService", e)
        }
    }
}
