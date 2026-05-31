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
import com.steadyphoto.sync.ui.MainActivity
import kotlinx.coroutines.*

/**
 * Foreground service that handles background media scanning and synchronization.
 * Runs continuously while the app is in use or when auto-sync is enabled.
 */
class SyncService : Service() {

    private val serviceScope = CoroutineScope(SupervisorJob() + Dispatchers.Default)
    private var syncJob: Job? = null

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
                // TODO: Implement actual sync logic
                // This will scan media, upload to server, and update database
                while (isActive) {
                    performSync()
                    delay(60_000) // Check every minute
                }
            }
        }
    }

    private fun stopSyncJob() {
        syncJob?.cancel()
        stopForeground(STOP_FOREGROUND_REMOVE)
        stopSelf()
    }

    private suspend fun performSync() {
        // TODO: Implement actual synchronization logic
        // 1. Scan device for new media files
        // 2. Check which files need to be uploaded (based on hash comparison)
        // 3. Upload new/modified files to server
        // 4. Download any missing files from server
        // 5. Update local database with sync status
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

    override fun onBind(intent: Intent?): IBinder? = null

    override fun onDestroy() {
        super.onDestroy()
        syncJob?.cancel()
        serviceScope.cancel()
    }
}
