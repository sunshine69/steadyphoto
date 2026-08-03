package com.steadyphoto.sync.worker

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.IBinder
import android.os.Build
import android.util.Log
import androidx.core.app.NotificationCompat
import com.steadyphoto.sync.R
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.di.AppContainer
import com.steadyphoto.sync.ui.MainActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

/**
 * Foreground service that handles background media scanning and synchronization.
 * 
 * Performs ONE sync cycle (scan + upload) and then stops itself.
 * Called via intent from MainActivity or BroadcastReceivers.
 * Battery optimization:
 * - UploadManager limits concurrency to maxConcurrentUploads (default 3)
 * - Max consecutive upload failures before giving up
 */
class SyncService : Service(), KoinComponent {

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var syncJob: Job? = null
    
    // Inject dependencies via Koin
    private val container: AppContainer by inject()

    companion object {
        const val CHANNEL_ID = "sync_service_channel"
        const val NOTIFICATION_ID = 1
        const val ACTION_START_SYNC = "com.steadyphoto.sync.ACTION_START_SYNC"
        const val ACTION_STOP_SYNC = "com.steadyphoto.sync.ACTION_STOP_SYNC"

        // Max consecutive failed upload attempts before giving up and stopping the service.
        private const val MAX_CONSECUTIVE_UPLOAD_FAILURES = 3

        fun newIntent(context: Context): Intent {
            return Intent(context, SyncService::class.java)
        }
    }

    // Track consecutive upload failures to prevent infinite retries
    private var consecutiveUploadFailures = 0
    
    // Flag to track whether Stop was explicitly requested. Prevents START_STICKY from 
    // restarting the service after a manual stop.
    private var stopRequested = false

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START_SYNC -> startSyncJob()
            ACTION_STOP_SYNC -> stopSyncJob()
            else -> {
                if (!stopRequested) {
                    startSyncJob()
                } else {
                    Log.d("SyncService", "Not restarting service — user had stopped it")
                    stopRequested = false
                    stopForeground(STOP_FOREGROUND_REMOVE)
                    stopSelf()
                }
            }
        }
        return START_STICKY
    }

    private fun startSyncJob() {
        if (syncJob == null || !syncJob!!.isActive) {
            syncJob = serviceScope.launch {
                try {
                    performSyncAndCheckForMore()
                } catch (e: Exception) {
                    Log.e("SyncService", "Sync job failed", e)
                } finally {
                    Log.d("SyncService", "Sync job finished - stopping service")
                    stopForeground(STOP_FOREGROUND_REMOVE)
                    stopSelf()
                }
            }
        }
    }

    private fun stopSyncJob() {
        stopRequested = true
        syncJob?.cancel()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    /**
     * Performs the full synchronization cycle and returns whether there's more work.
     * 
     * Returns true if:
     * - New items were scanned/uploaded, OR
     * - Pending uploads were found (so we should check again later)
     * 
     * Returns false if nothing to do — service can safely stop itself.
     */
    private suspend fun performSyncAndCheckForMore(): Boolean {
        var hasWork = false
        
        try {
            // Step 1: Scan for new media files
            val scanResult = container.repository.scanNewMedia(forceFullScan = false)
            
            when (scanResult) {
                is com.steadyphoto.sync.data.repository.ScanResult.Success -> {
                    hasWork = true
                    
                    updateNotification(
                        "Scanning...", 
                        "${scanResult.totalScanned} items scanned, ${scanResult.newItemsInserted} new items found"
                    )
                    
                    // Step 2: Upload pending/failed items
                    val pendingAndFailed = container.mediaItemDao.getPendingAndFailedItems(
                        listOf(UploadStatus.PENDING, UploadStatus.FAILED)
                    )
                    
                    if (pendingAndFailed.isNotEmpty()) {
                        uploadPendingItems(pendingAndFailed)
                    } else {
                        updateNotification("Syncing", "No items to upload")
                    }
                }

                is com.steadyphoto.sync.data.repository.ScanResult.NoNewItems -> {
                    val pendingAndFailed = container.mediaItemDao.getPendingAndFailedItems(
                        listOf(UploadStatus.PENDING, UploadStatus.FAILED)
                    )
                    
                    if (pendingAndFailed.isNotEmpty()) {
                        hasWork = true
                        uploadPendingItems(pendingAndFailed)
                    } else {
                        updateNotification("Syncing", "No items to sync")
                    }
                }
                
                is com.steadyphoto.sync.data.repository.ScanResult.PermissionDenied -> {
                    updateNotification("Scan Error", "Permission denied for media scan")
                }
                
                is com.steadyphoto.sync.data.repository.ScanResult.Error -> {
                    updateNotification("Scan Error", scanResult.message)
                }
            }
        } catch (e: Exception) {
            Log.e("SyncService", "Error during sync cycle", e)
            updateNotification("Sync Error", e.message ?: "Unknown error")
            consecutiveUploadFailures = 0
        }
        
        return hasWork
    }

    private suspend fun uploadPendingItems(pendingAndFailed: List<com.steadyphoto.sync.data.local.entity.MediaItemEntity>) {
        updateNotification("Uploading...", "${pendingAndFailed.size} items to upload")
        
        // Check network settings before uploading
        val networkSettings = container.settingsRepository.networkSettingsFlow.first()
        if (!container.uploadManager.networkMonitor.isNetworkAcceptableForUpload(networkSettings)) {
            val currentType = container.uploadManager.networkMonitor.getCurrentNetworkType()?.name ?: "unknown"
            Log.w("SyncService", "Skipping upload - network type ($currentType) doesn't meet preferences")
            updateNotification("Waiting for Network", "Uploading on $currentType not allowed in settings. Waiting...")
            return
        }
        
        val result = container.uploadManager.uploadMedia(pendingAndFailed)
        
        when {
            result.isSuccess -> {
                consecutiveUploadFailures = 0
                updateNotification(
                    "Upload Complete", 
                    "${pendingAndFailed.size} items uploaded"
                )
            }
            else -> {
                consecutiveUploadFailures++
                val error = result.exceptionOrNull()?.message ?: "Unknown error"
                updateNotification("Upload Failed", error)
                
                if (consecutiveUploadFailures >= MAX_CONSECUTIVE_UPLOAD_FAILURES) {
                    Log.w("SyncService", 
                        "Max upload failures reached ($MAX_CONSECUTIVE_UPLOAD_FAILURES). Stopping service.")
                    updateNotification(
                        "Upload Failed", 
                        "Too many failed uploads. Please check network and try again later."
                    )
                }
            }
        }
    }

    private fun createNotificationChannel() {
        if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
            val channel = NotificationChannel(
                CHANNEL_ID,
                "Sync Service",
                NotificationManager.IMPORTANCE_LOW
            ).apply {
                description = "Background media synchronization"
            }

            val notificationManager = getSystemService(Context.NOTIFICATION_SERVICE) as NotificationManager
            notificationManager.createNotificationChannel(channel)
        }
    }

    private fun createNotification(): Notification {
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        return NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle("SteadyPhoto Sync")
            .setContentText("Syncing your photos...")
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()
    }

    private fun updateNotification(title: String, text: String) {
        val pendingIntent = PendingIntent.getActivity(
            this,
            0,
            Intent(this, MainActivity::class.java),
            PendingIntent.FLAG_IMMUTABLE or PendingIntent.FLAG_UPDATE_CURRENT
        )

        val notification = NotificationCompat.Builder(this, CHANNEL_ID)
            .setContentTitle(title)
            .setContentText(text)
            .setSmallIcon(R.drawable.ic_launcher_foreground)
            .setContentIntent(pendingIntent)
            .setOngoing(true)
            .build()

        startForeground(NOTIFICATION_ID, notification)
    }

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onDestroy() {
        super.onDestroy()
        syncJob?.cancel()
    }
}
