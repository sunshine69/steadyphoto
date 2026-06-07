package com.steadyphoto.sync.worker

import android.content.Context
import android.util.Log
import androidx.work.*
import com.steadyphoto.sync.di.AppContainer
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject
import java.util.concurrent.TimeUnit

/**
 * SyncManager is a helper class to coordinate background work using WorkManager.
 */
class SyncManager private constructor(private val context: Context) : KoinComponent {

    private val container: AppContainer by inject()

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
     */
    fun schedulePeriodicSync() {
        val workManager = WorkManager.getInstance(context)

        val periodicRequest = PeriodicWorkRequestBuilder<MediaScannerWorker>(1, TimeUnit.HOURS)
            .addTag(PERIODIC_SCAN_TAG)
            .setConstraints(
                Constraints.Builder()
                    .setRequiredNetworkType(NetworkType.CONNECTED)
                    .build()
            )
            .build()

        workManager.enqueueUniquePeriodicWork(
            "SteadyPhoto_PeriodicScan",
            ExistingPeriodicWorkPolicy.KEEP, 
            periodicRequest
        )
        Log.d(TAG, "Scheduled periodic scan every 1 hour (KEEP policy).")
    }

    /**
     * Triggers an immediate sync/scan.
     */
    fun triggerImmediateSync() {
        Log.d(TAG, "Triggering immediate sync...")
        val workManager = WorkManager.getInstance(context)

        val scanRequest = OneTimeWorkRequestBuilder<MediaScannerWorker>()
            .addTag(IMMEDIATE_SYNC_TAG)
            .setExpedited(OutOfQuotaPolicy.RUN_AS_NON_EXPEDITED_WORK_REQUEST) 
            .build()

        val uploadRequest = OneTimeWorkRequestBuilder<UploadWorker>()
            .addTag(IMMEDIATE_SYNC_TAG)
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
            .build()

        workManager.beginUniqueWork(
            "SteadyPhoto_ImmediateSync_${System.currentTimeMillis()}", 
            ExistingWorkPolicy.REPLACE, 
            scanRequest
        )
        .then(uploadRequest)
        .enqueue()
    }

    /**
     * Triggers only the upload phase.
     */
    fun triggerUpload() {
        Log.d(TAG, "Triggering background upload worker...")
        val workManager = WorkManager.getInstance(context)

        val uploadRequest = OneTimeWorkRequestBuilder<UploadWorker>()
            .addTag(IMMEDIATE_SYNC_TAG)
            .setConstraints(Constraints.Builder().setRequiredNetworkType(NetworkType.CONNECTED).build())
            .build()

        workManager.enqueueUniqueWork(
            "SteadyPhoto_BackgroundUpload",
            ExistingWorkPolicy.REPLACE, 
            uploadRequest
        )
    }

    fun cancelAllSyncWork() {
        Log.d(TAG, "Cancelling all scheduled sync work.")
        val workManager = WorkManager.getInstance(context)
        workManager.cancelAllWorkByTag(PERIODIC_SCAN_TAG)
        workManager.cancelAllWorkByTag(IMMEDIATE_SYNC_TAG)
    }

    suspend fun isSyncInProgress(): Boolean {
        val workManager = WorkManager.getInstance(context)
        val info = workManager.getWorkInfosByTag(IMMEDIATE_SYNC_TAG).get()
        return info.any { it.state == WorkInfo.State.RUNNING || it.state == WorkInfo.State.ENQUEUED }
    }
}
