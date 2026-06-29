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

        // Max consecutive failed upload attempts before giving up and stopping the service.
        private const val MAX_CONSECUTIVE_UPLOAD_FAILURES = 3

        fun newIntent(context: Context): Intent {
            return Intent(context, SyncService::class.java)
        }
    }

    // Track consecutive upload failures to prevent infinite retries
    private var consecutiveUploadFailures = 0
    
    // Flag to track whether Stop was explicitly requested. Prevents START_STICKY from 
    // restarting the service after a manual stop. Android may restart killed services,
    // and without this flag it would start running again even though the user stopped it.
    private var stopRequested = false

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
        
        mediaContentObserver = MediaContentObserver(Handler(android.os.Looper.getMainLooper()), serviceScope) {
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
            else -> {
                // If Android restarts this service after it was killed (e.g., during Doze mode),
                // check if the user had explicitly requested a stop. Don't auto-restart if so.
                if (!stopRequested) {
                    startSyncJob()
                } else {
                    Log.d("SyncService", "Not restarting service — user had stopped it")
                    stopRequested = false  // Reset flag for next time the user starts sync
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
                while (true) {
                    try {
                        ensureActive()
                    } catch (_: CancellationException) {
                        break
                    }
                    
                    // Perform a single scan to check for new items, then upload if needed.
                    // If nothing needs doing, do NOT loop again — exit the service early.
                    val hasWork = performSyncAndCheckForMore()
                    
                    if (!hasWork) {
                        Log.d("SyncService", "No more work to do — stopping sync job")
                        break  // Exit the loop; service will stop itself via onDestroy after cleanup
                    }
                    
                    // Only delay between actual work cycles. If there was work, wait and check again.
                    val syncControl = container.settingsRepository.syncControlFlow.first()
                    delay(syncControl.fallbackSyncIntervalMinutes * 60_000L)
                }
            }
        }
    }

    private fun stopSyncJob() {
        // Mark that the user explicitly requested a stop. This prevents START_STICKY from
        // restarting the service after it's been killed by Android (e.g., during Doze mode).
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
            // Step 1: Scan for new media files - using incremental scanning if possible
            val scanResult = container.repository.scanNewMedia(forceFullScan = false)
            
            when (scanResult) {
                is com.steadyphoto.sync.data.repository.ScanResult.Success -> {
                    hasWork = true  // Items were found
                    
                    updateNotification(
                        "Scanning...", 
                        "${scanResult.totalScanned} items scanned, ${scanResult.newItemsInserted} new items found"
                    )
                    
                    // Step 2: Upload pending/failed items using the upload manager
                    val pendingAndFailed = container.mediaItemDao.getPendingAndFailedItems(
                        listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, 
                               com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
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
                                    consecutiveUploadFailures = 0  // Reset failure counter on success
                                    val uploadResult = result.getOrNull()
                                    updateNotification(
                                        "Upload Complete", 
                                        "${uploadResult?.successCount ?: 0} succeeded, ${uploadResult?.failureCount ?: pendingAndFailed.size} failed"
                                    )
                                }
                                else -> {
                                    consecutiveUploadFailures++
                                    val error = result.exceptionOrNull()?.message ?: "Unknown error"
                                    updateNotification("Upload Failed", error)
                                    
                                    // Check if we've exceeded the max consecutive failure limit
                                    if (consecutiveUploadFailures >= MAX_CONSECUTIVE_UPLOAD_FAILURES) {
                                        Log.w("SyncService", 
                                            "Max upload failures reached ($MAX_CONSECUTIVE_UPLOAD_FAILURES). Stopping service to prevent battery drain.")
                                        updateNotification(
                                            "Upload Failed", 
                                            "Too many failed uploads. Please check network and try again later."
                                        )
                                        return false  // Stop the service — no more retries
                                    }
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
                        listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING, 
                               com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
                    )
                    
                    if (pendingAndFailed.isNotEmpty()) {
                        hasWork = true  // Pending uploads exist — check again later
                        
                        updateNotification("Uploading...", "${pendingAndFailed.size} items to upload")
                        
                        val result = container.uploadManager.uploadMedia(pendingAndFailed)
                        
                        when {
                            result.isSuccess -> {
                                consecutiveUploadFailures = 0  // Reset failure counter on success
                                val uploadResult = result.getOrNull()
                                updateNotification(
                                    "Upload Complete", 
                                    "${uploadResult?.successCount ?: 0} succeeded, ${uploadResult?.failureCount ?: pendingAndFailed.size} failed"
                                )
                            }
                            else -> {
                                consecutiveUploadFailures++
                                val error = result.exceptionOrNull()?.message ?: "Unknown error"
                                updateNotification("Upload Failed", error)
                                
                                // Check if we've exceeded the max consecutive failure limit
                                if (consecutiveUploadFailures >= MAX_CONSECUTIVE_UPLOAD_FAILURES) {
                                    Log.w("SyncService", 
                                        "Max upload failures reached ($MAX_CONSECUTIVE_UPLOAD_FAILURES). Stopping service to prevent battery drain.")
                                    updateNotification(
                                        "Upload Failed", 
                                        "Too many failed uploads. Please check network and try again later."
                                    )
                                    return false  // Stop the service — no more retries
                                }
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
            
            // If an unexpected exception occurs, reset failure counter and stop
            consecutiveUploadFailures = 0
        }
        
        return hasWork
    }

    /**
     * Performs the full synchronization cycle.
     * This is a convenience wrapper for FileObserver/ContentObserver callbacks.
     */
    private suspend fun performSync() {
        performSyncAndCheckForMore()  // Ignore return value — observers keep service alive
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
