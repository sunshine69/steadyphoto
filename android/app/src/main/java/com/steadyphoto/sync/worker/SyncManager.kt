package com.steadyphoto.sync.worker

import android.content.Context
import android.util.Log
import androidx.work.*
import com.steadyphoto.sync.data.settings.NetworkSettings
import com.steadyphoto.sync.data.settings.SettingsRepository
import com.steadyphoto.sync.di.AppContainer
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.runBlocking
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject
import java.util.concurrent.TimeUnit

/**
 * SyncManager is a helper class to coordinate background work using WorkManager.
 * Enhanced to respect network settings from SettingsRepository.
 */
class SyncManager internal constructor(private val context: Context) : KoinComponent {

    private val container: AppContainer by inject()
    
    // Direct access to SettingsRepository for synchronous reads when needed
    private val settingsRepository: SettingsRepository 
        get() = container.settingsRepository

    companion object {
        private const val TAG = "SyncManager"
        const val PERIODIC_SCAN_TAG = "periodic_scan_work"
        const val IMMEDIATE_SYNC_TAG = "immediate_sync_work"

        @Volatile
        private var INSTANCE: SyncManager? = null

        fun getInstance(context: Context): SyncManager {
            return INSTANCE ?: synchronized(this) {
                INSTANCE ?: SyncManager(context.applicationContext).also { INSTANCE = it }
            }
        }
    }

    /**
     * Schedules the periodic maintenance scan.
     * Uses KEEP to prevent resetting the 1-hour timer every time the app starts.
     * Only schedules if auto-sync is enabled in settings.
     */
    fun schedulePeriodicSync() {
        val workManager = WorkManager.getInstance(context)

        runBlocking {
            val (networkSettings, syncControlSettings) = getAllSettings()
            
            // Don't schedule periodic sync if auto-sync is disabled
            if (!syncControlSettings.autoSyncEnabled) {
                Log.d(TAG, "Auto-sync is disabled - not scheduling periodic scan")
                return@runBlocking
            }

            val constraints = getNetworkConstraintsFromSettings(networkSettings)
            
            val periodicRequest = PeriodicWorkRequestBuilder<MediaScannerWorker>(1, TimeUnit.HOURS)
                .addTag(PERIODIC_SCAN_TAG)
                .setConstraints(constraints)
                .build()

            workManager.enqueueUniquePeriodicWork(
                "SteadyPhoto_PeriodicScan",
                ExistingPeriodicWorkPolicy.KEEP, 
                periodicRequest
            )
            Log.d(TAG, "Scheduled periodic scan every 1 hour (KEEP policy).")
        }
    }

    /**
     * Triggers an immediate sync/scan.
     */
    fun triggerImmediateSync() {
        Log.d(TAG, "Triggering immediate sync...")
        val workManager = WorkManager.getInstance(context)

        runBlocking {
            val (networkSettings, _) = getAllSettings()
            val scanConstraints = getNetworkConstraintsFromSettings(networkSettings)

            val scanRequest = OneTimeWorkRequestBuilder<MediaScannerWorker>()
                .addTag(IMMEDIATE_SYNC_TAG)
                .setExpedited(OutOfQuotaPolicy.RUN_AS_NON_EXPEDITED_WORK_REQUEST) 
                .build()

            val uploadRequest = OneTimeWorkRequestBuilder<UploadWorker>()
                .addTag(IMMEDIATE_SYNC_TAG)
                .setConstraints(scanConstraints)
                .build()

            workManager.beginUniqueWork(
                "SteadyPhoto_ImmediateSync_${System.currentTimeMillis()}", 
                ExistingWorkPolicy.REPLACE, 
                scanRequest
            )
            .then(uploadRequest)
            .enqueue()
        }
    }

    /**
     * Triggers only the upload phase.
     */
    fun triggerUpload() {
        Log.d(TAG, "Triggering background upload worker...")
        val workManager = WorkManager.getInstance(context)

        runBlocking {
            val (networkSettings, _) = getAllSettings()
            val uploadConstraints = getNetworkConstraintsFromSettings(networkSettings)

            val uploadRequest = OneTimeWorkRequestBuilder<UploadWorker>()
                .addTag(IMMEDIATE_SYNC_TAG)
                .setConstraints(uploadConstraints)
                .build()

            workManager.enqueueUniqueWork(
                "SteadyPhoto_BackgroundUpload",
                ExistingWorkPolicy.REPLACE, 
                uploadRequest
            )
        }
    }

    /**
     * Re-schedules all sync work with updated network constraints.
     * Call this when settings change to ensure workers use new preferences.
     */
    fun rescheduleWithCurrentSettings() {
        Log.d(TAG, "Re-scheduling sync work with current network settings")
        
        // Cancel existing periodic and immediate work
        cancelAllSyncWork()
        
        // Re-schedule with updated constraints
        schedulePeriodicSync()
    }

    /**
     * Cancels all scheduled sync work.
     */
    fun cancelAllSyncWork() {
        Log.d(TAG, "Cancelling all scheduled sync work.")
        val workManager = WorkManager.getInstance(context)
        workManager.cancelAllWorkByTag(PERIODIC_SCAN_TAG)
        workManager.cancelAllWorkByTag(IMMEDIATE_SYNC_TAG)
    }

    /**
     * Checks if any sync work is currently in progress.
     */
    suspend fun isSyncInProgress(): Boolean {
        val workManager = WorkManager.getInstance(context)
        val info = workManager.getWorkInfosByTag(IMMEDIATE_SYNC_TAG).get()
        return info.any { it.state == WorkInfo.State.RUNNING || it.state == WorkInfo.State.ENQUEUED }
    }

    /**
     * Converts network settings to WorkManager Constraints.
     * Uses NetworkType.UNMETERED for WiFi-only mode, CONNECTED for any network.
     */
    private suspend fun getNetworkConstraintsFromSettings(networkSettings: com.steadyphoto.sync.data.settings.NetworkSettings): Constraints {
        val builder = Constraints.Builder()
        
        when (networkSettings.wifiOnlyEnabled) {
            true -> {
                // WiFi-only: only allow on unmetered connections (WiFi, Ethernet)
                Log.d(TAG, "Using UNMETERED network constraint (WiFi-only mode)")
                builder.setRequiredNetworkType(NetworkType.UNMETERED)
            }
            false -> {
                // Any network: allow all connected networks including cellular
                Log.d(TAG, "Using CONNECTED network constraint (any network mode)")
                if (!networkSettings.syncOnMeteredConnection) {
                    // Don't sync on metered connections - use REQUIRE_UNMETERED or UNMETERED
                    builder.setRequiredNetworkType(NetworkType.UNMETERED)
                } else {
                    builder.setRequiredNetworkType(NetworkType.CONNECTED)
                }
            }
        }
        
        return builder.build()
    }

    /**
     * Gets all current settings from SettingsRepository (both network and sync control).
     * Uses first{} to get a synchronous value from the Flow.
     */
    private suspend fun getAllSettings(): Pair<NetworkSettings, com.steadyphoto.sync.data.settings.SyncControlSettings> {
        return try {
            val networkSettings = settingsRepository.networkSettingsFlow.first()
            val syncControlSettings = settingsRepository.syncControlFlow.first()
            Pair(networkSettings, syncControlSettings)
        } catch (e: Exception) {
            Log.w(TAG, "Failed to read settings, using defaults", e)
            Pair(NetworkSettings.default(), com.steadyphoto.sync.data.settings.SyncControlSettings.default())
        }
    }

    /**
     * Gets the current network settings from SettingsRepository.
     * Uses first{} to get a synchronous value from the Flow.
     */
    private suspend fun getCurrentSettings(): NetworkSettings {
        return getAllSettings().first
    }
}
