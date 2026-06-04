package com.steadyphoto.sync.worker

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.os.Build
import android.os.IBinder
import androidx.core.app.NotificationCompat
import com.steadyphoto.sync.R
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.di.AppContainer
import com.steadyphoto.sync.ui.MainActivity
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.delay
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

/**
 * Foreground service that handles background media scanning and synchronization.
 * Runs continuously while the app is in use or when auto-sync is enabled.
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

        fun newIntent(context: Context): Intent {
            return Intent(context, SyncService::class.java)
        }
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())
    }

    override fun onStartCommand(intent: Intent?, flags: Int, startId: Int): Int {
        when (intent?.action) {
            ACTION_START_SYNC -> startSyncJob()
            ACTION_STOP_SYNC -> stopSyncJob()
            else -> startSyncJob()
        }
        return START_STICKY
    }

    private fun startSyncJob() {
        if (syncJob == null || !syncJob!!.isActive) {
            syncJob = serviceScope.launch {
                while (true) {
                try {
                    ensureActive()
                } catch (_: CancellationException) {
                    break
                }
                    performSync()
                    delay(60_000L) // Check every minute
                }
            }
        }
    }

    private fun stopSyncJob() {
        syncJob?.cancel()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    /**
     * Performs the full synchronization cycle:
     * 1. Scan device for new/changed media files
     * 2. Upload pending items to server
     */
    private suspend fun performSync() {
        try {
            // Step 1: Scan for new media files (uses MediaScannerWorker logic internally)
            val scannedItems = container.repository.scanNewMedia()
            
            when (scannedItems) {
                is com.steadyphoto.sync.data.repository.ScanResult.Success -> {
                    updateNotification(
                        "Scanning...", 
                        "${scannedItems.totalScanned} items scanned, ${scannedItems.newItemsInserted} new items found"
                    )
                    
                    // Step 2: Upload pending/failed items
                    val pendingAndFailed = container.mediaItemDao.getPendingAndFailedItems(
                        listOf(UploadStatus.PENDING, UploadStatus.FAILED)
                    )
                    
                    if (pendingAndFailed.isNotEmpty()) {
                        updateNotification("Uploading...", "${pendingAndFailed.size} items to upload")
                        
                        // Use UploadManager for consistent upload handling with progress tracking
                        val result = container.uploadManager.uploadMedia(pendingAndFailed)
                        
                        when {
                            result.isSuccess -> {
                                val uploadResult = result.getOrNull()
                                updateNotification(
                                    "Upload Complete", 
                                    "${uploadResult?.successCount ?: 0} succeeded, ${uploadResult?.failureCount ?: pendingAndFailed.size} failed"
                                )
                            }
                            else -> {
                                val error = result.exceptionOrNull()?.message ?: "Unknown error"
                                updateNotification("Upload Failed", error)
                            }
                        }
                    } else {
                        updateNotification("Syncing", "No items to upload")
                    }
                }
                
                is com.steadyphoto.sync.data.repository.ScanResult.NoNewItems -> {
                    // No new items found, check for pending uploads anyway
                    val pendingAndFailed = container.mediaItemDao.getPendingAndFailedItems(
                        listOf(UploadStatus.PENDING, UploadStatus.FAILED)
                    )
                    
                    if (pendingAndFailed.isNotEmpty()) {
                        updateNotification("Uploading...", "${pendingAndFailed.size} items to upload")
                        
                        val result = container.uploadManager.uploadMedia(pendingAndFailed)
                        
                        when {
                            result.isSuccess -> {
                                val uploadResult = result.getOrNull()
                                updateNotification(
                                    "Upload Complete", 
                                    "${uploadResult?.successCount ?: 0} succeeded, ${uploadResult?.failureCount ?: pendingAndFailed.size} failed"
                                )
                            }
                            else -> {
                                val error = result.exceptionOrNull()?.message ?: "Unknown error"
                                updateNotification("Upload Failed", error)
                            }
                        }
                    } else {
                        updateNotification("Syncing", "No items to sync")
                    }
                }
                
                is com.steadyphoto.sync.data.repository.ScanResult.PermissionDenied -> {
                    updateNotification("Scan Error", "Permission denied for media scan")
                }
                
                is com.steadyphoto.sync.data.repository.ScanResult.Error -> {
                    updateNotification("Scan Error", scannedItems.message)
                }
            }
        } catch (e: Exception) {
            // Log the error but don't crash the service
            android.util.Log.e("SyncService", "Error during sync cycle", e)
            updateNotification("Sync Error", e.message ?: "Unknown error")
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

    /**
     * Updates the notification content to reflect current sync status.
     */
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
        serviceScope.coroutineContext[Job]?.cancel()
    }
}
