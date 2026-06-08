package com.steadyphoto.sync.worker

import android.app.Notification
import android.app.NotificationChannel
import android.app.NotificationManager
import android.app.PendingIntent
import android.app.Service
import android.content.Context
import android.content.Intent
import android.net.Uri
import android.os.IBinder
import android.os.Build
import android.os.Handler
import android.provider.MediaStore
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
import kotlinx.coroutines.delay
import kotlinx.coroutines.ensureActive
import kotlinx.coroutines.CancellationException
import kotlinx.coroutines.launch
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

/**
 * Foreground service that handles background media scanning and synchronization.
 * 
 * Uses DirectoryFileObserver (filesystem-level) for real-time detection of new files,
 * combined with ContentObserver as a secondary mechanism to catch MediaStore-indexed changes.
 * Periodic sync is used as an additional fallback.
 */
class SyncService : Service(), KoinComponent {

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var syncJob: Job? = null
    
    // FileObserver for real-time filesystem detection (primary mechanism)
    private var fileObservers: List<DirectoryFileObserver> = emptyList()
    
    // ContentObserver as secondary mechanism  
    private var mediaContentObserver: MediaContentObserver? = null

    // Inject dependencies via Koin
    private val container: AppContainer by inject()

    companion object {
        const val CHANNEL_ID = "sync_service_channel"
        const val NOTIFICATION_ID = 1
        const val ACTION_START_SYNC = "com.steadyphoto.sync.ACTION_START_SYNC"
        const val ACTION_STOP_SYNC = "com.steadyphoto.sync.ACTION_STOP_SYNC"

        // Periodic sync interval as fallback (5 minutes) - FileObserver handles real-time
        private const val PERIODIC_SYNC_INTERVAL_MS = 300_000L
        
        fun newIntent(context: Context): Intent {
            return Intent(context, SyncService::class.java)
        }
    }

    override fun onCreate() {
        super.onCreate()
        createNotificationChannel()
        startForeground(NOTIFICATION_ID, createNotification())
        
        // Register FileObserver for real-time filesystem detection (primary mechanism)
        registerFileObservers()
        
        // Also register ContentObserver as secondary mechanism for MediaStore changes
        registerContentObserver()
    }

    private fun registerFileObservers() {
        val observers = DirectoryFileObserver.MONITORED_DIRS.mapNotNull { dir ->
            try {
                DirectoryFileObserver(dir) { file ->
                    Log.d("SyncService", "New media file via FileObserver: ${file.absolutePath}")
                    // Trigger sync when new files are detected at filesystem level
                    serviceScope.launch {
                        performSync()
                    }
                }.also { it.startWatching() }
            } catch (e: Exception) {
                Log.w("SyncService", "Failed to start FileObserver for $dir: ${e.message}")
                null
            }
        }
        
        fileObservers = observers
        if (observers.isNotEmpty()) {
            Log.d("SyncService", "FileObserver registered for ${observers.size} directories")
        } else {
            Log.w("SyncService", "No FileObservers could be started - filesystem monitoring disabled")
        }
    }

    private fun registerContentObserver() {
        val contentResolver = applicationContext.contentResolver
        
        mediaContentObserver = MediaContentObserver(Handler(), serviceScope) {
            performSync()
        }
        
        // Register on BOTH images and videos URIs with notifyForDescendants=true
        try {
            contentResolver.registerContentObserver(
                MediaStore.Images.Media.EXTERNAL_CONTENT_URI,
                true,  // notifyForDescendants - catch changes in subdirectories too
                mediaContentObserver!!
            )
            
            contentResolver.registerContentObserver(
                MediaStore.Video.Media.EXTERNAL_CONTENT_URI,
                true,  // notifyForDescendants - catch changes in subdirectories too
                mediaContentObserver!!
            )
            
            Log.d("SyncService", "ContentObserver registered for real-time detection")
        } catch (e: Exception) {
            Log.w("SyncService", "Failed to register ContentObserver: ${e.message}")
            mediaContentObserver = null
        }
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
                    // Periodic fallback sync every 5 minutes - catches anything FileObserver/ContentObserver missed
                    performSync()
                    delay(PERIODIC_SYNC_INTERVAL_MS)
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
     * 1. Scan device for new/changed media files (using MediaContentObserver's logic internally)
     * 2. Upload pending items to server
     */
    private suspend fun performSync() {
        try {
            // Step 1: Scan for new media files - this uses the same query as ContentObserver
            // but queries ALL rows without MIME type filtering (the key fix)
            val scanResult = container.repository.scanNewMedia(
                forceFullScan = true  // Force scan all rows, don't filter by MIME type
            )
            
            when (scanResult) {
                is com.steadyphoto.sync.data.repository.ScanResult.Success -> {
                    updateNotification(
                        "Scanning...", 
                        "${scanResult.totalScanned} items scanned, ${scanResult.newItemsInserted} new items found"
                    )
                    
                    // Step 2: Upload pending/failed items using the upload manager
                    val pendingAndFailed = container.mediaItemDao.getPendingAndFailedItems(
                        listOf(UploadStatus.PENDING, UploadStatus.FAILED)
                    )
                    
                    if (pendingAndFailed.isNotEmpty()) {
                        updateNotification("Uploading...", "${pendingAndFailed.size} items to upload")
                        
                        // Check network settings before uploading - respect WiFi-only preference
                        val networkSettings = container.settingsRepository.networkSettingsFlow.first()
                        if (!container.uploadManager.networkMonitor.isNetworkAcceptableForUpload(networkSettings)) {
                            val currentType = container.uploadManager.networkMonitor.getCurrentNetworkType()?.name ?: "unknown"
                            Log.w("SyncService", "Skipping upload - network type ($currentType) doesn't meet preferences")
                            updateNotification(
                                "Waiting for Network", 
                                "Uploading on $currentType not allowed in settings. Waiting..."
                            )
                        } else {
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
                    updateNotification("Scan Error", scanResult.message)
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
        
        // Unregister FileObservers - must be done in try-catch as it may throw if not registered
        fileObservers.forEach { observer ->
            try {
                observer.stopWatching()
            } catch (e: Exception) {
                Log.w("SyncService", "Failed to stop FileObserver", e)
            }
        }
        fileObservers = emptyList()
        
        // Unregister ContentObserver - must be done in try-catch as it may throw if not registered
        mediaContentObserver?.let { observer ->
            try {
                applicationContext.contentResolver.unregisterContentObserver(observer)
            } catch (e: IllegalArgumentException) {
                Log.w("SyncService", "ContentObserver was already unregistered")
            }
        }
        mediaContentObserver = null
        
        // Cancel sync job and coroutine scope
        syncJob?.cancel()
        serviceScope.coroutineContext[Job]?.cancel()
    }
}
