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
class SyncManager(private val context: Context) : KoinComponent {

    // We use inject() here because this class is managed by Koin as a singleton via appModule
    private val container: AppContainer by inject()
    
    // Direct access to SettingsRepository for synchronous reads when needed
    private val settingsRepository: SettingsRepository 
        get() = container.settingsRepository

    companion object {
        private const val TAG = "SyncManager"
        const val PERIODIC_SCAN_TAG = "periodic_scan_work"
        const val IMMEDIATE_SYNC_TAG = "immediate_sync_work"
    }

    /**
     * Schedules the periodic maintenance scan.
     */
    fun schedulePeriodicSync() {
        val workManager = WorkManager.getInstance(context)

        runBlocking {
            val (networkSettings, syncControlSettings) = getAllSettings()
            
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
     */
    fun rescheduleWithCurrentSettings() {
        cancelAllSyncWork()
        schedulePeriodicSync()
    }

    /**
     * Cancels all scheduled sync work.
     */
    fun cancelAllSyncWork() {
        val workManager = WorkManager.getInstance(context)
        workManager.cancelAllWorkByTag(PERIODIC_SCAN_TAG)
        workManager.cancelAllWorkByTag(IMMEDIATE_SYNC_TAG)
    }

    suspend fun isSyncInProgress(): Boolean {
        val workManager = WorkManager.getInstance(context)
        val info = workManager.getWorkInfosByTag(IMMEDIATE_SYNC_TAG).get()
        return info.any { it.state == WorkInfo.State.RUNNING || it.state == WorkInfo.State.ENQUEUED }
    }

    private suspend fun getNetworkConstraintsFromSettings(networkSettings: com.steadyphoto.sync.data.settings.NetworkSettings): Constraints {
        val builder = Constraints.Builder()
        when (networkSettings.wifiOnlyEnabled) {
            true -> builder.setRequiredNetworkType(NetworkType.UNMETERED)
            false -> {
                if (!networkSettings.syncOnMeteredConnection) {
                    builder.setRequiredNetworkType(NetworkType.UNMETERED)
                } else {
                    builder.setRequiredNetworkType(NetworkType.CONNECTED)
                }
            }
        }
        return builder.build()
    }

    private suspend fun getAllSettings(): Pair<NetworkSettings, com.steadyphoto.sync.data.settings.SyncControlSettings> {
        return try {
            val networkSettings = settingsRepository.networkSettingsFlow.first()
            val syncControlSettings = settingsRepository.syncControlFlow.first()
            Pair(networkSettings, syncControlSettings)
        } catch (e: Exception) {
            Pair(NetworkSettings.default(), com.steadyphoto.sync.data.settings.SyncControlSettings.default())
        }
    }
}
