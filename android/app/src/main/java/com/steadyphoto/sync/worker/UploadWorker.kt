package com.steadyphoto.sync.worker

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.Data
import androidx.work.WorkerParameters
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.data.repository.NetworkConnectivityMonitor
import com.steadyphoto.sync.data.repository.UploadProgressCallback
import com.steadyphoto.sync.data.repository.UploadSessionProgress
import com.steadyphoto.sync.di.AppContainer
import kotlinx.coroutines.flow.first
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

/**
 * Worker responsible for uploading pending media items.
 * Enhanced to respect network preferences (WiFi-only, metered connection settings).
 */
class UploadWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params), KoinComponent {

    private val container: AppContainer by inject()

    companion object {
        private const val TAG = "UploadWorker"
        
        // Result keys for work data
        const val RESULT_SKIPPED_NO_NETWORK = "skipped_no_network"
        const val RESULT_SKIPPED_WRONG_TYPE = "skipped_wrong_network_type"
    }

    override suspend fun doWork(): Result {
        return try {
            // Get current network settings
            val networkSettings = container.settingsRepository.networkSettingsFlow.first()
            
            Log.d(TAG, "UploadWorker started with settings: wifiOnly=${networkSettings.wifiOnlyEnabled}, metered=${networkSettings.syncOnMeteredConnection}")

            // Check if we have any internet connection at all
            if (!container.uploadManager.networkMonitor.isCurrentlyConnected()) {
                Log.w(TAG, "No network connection available - skipping upload")
                val data = Data.Builder()
                    .putBoolean(RESULT_SKIPPED_NO_NETWORK, true)
                    .build()
                return Result.success(data)
            }

            // Check if current network meets preferences (WiFi-only, metered settings)
            if (!container.uploadManager.networkMonitor.isNetworkAcceptableForUpload(networkSettings)) {
                val currentType = container.uploadManager.networkMonitor.getCurrentNetworkType()?.name ?: "unknown"
                Log.w(TAG, "Current network type ($currentType) doesn't meet preferences - skipping upload")
                
                val data = Data.Builder()
                    .putBoolean(RESULT_SKIPPED_WRONG_TYPE, true)
                    .putString("network_type", currentType)
                    .build()
                return Result.success(data)
            }

            // Get items for upload. Include UPLOADING status in case a previous worker crashed
            // and left items in that state.
            val itemsToUpload = container.mediaItemDao.getPendingAndFailedItems(
                listOf(UploadStatus.PENDING, UploadStatus.FAILED, UploadStatus.UPLOADING)
            )
            
            if (itemsToUpload.isEmpty()) {
                Log.d(TAG, "No items to upload")
                return Result.success()
            }
            
            Log.d(TAG, "Found ${itemsToUpload.size} items to upload on acceptable network")

            // Create a progress callback that updates WorkManager's progress data
            val progressCallback = object : UploadProgressCallback {
                override suspend fun onProgressUpdated(progress: UploadSessionProgress) {
                    val progressData = Data.Builder()
                        .putInt("totalItems", progress.totalItems)
                        .putInt("uploadedItems", progress.uploadedItems)
                        .putInt("failedItems", progress.failedItems)
                        .putBoolean("isComplete", progress.isComplete)
                        .build()
                    setProgress(progressData)
                }

                override suspend fun onUploadComplete(successCount: Int, failureCount: Int) {
                    Log.d(TAG, "Upload session complete: $successCount succeeded, $failureCount failed")
                }

                override suspend fun onUploadError(error: Throwable) {
                    Log.e(TAG, "Upload session error: ${error.message}", error)
                }
            }
            
            // Use the UploadManager for uploads. 
            // Note: UploadManager ALREADY updates the database status for each item (PENDING -> UPLOADING -> UPLOADED/FAILED).
            // We don't need to manually update statuses here anymore.
            val result = container.uploadManager.uploadMedia(itemsToUpload, progressCallback)
            
            if (result.isSuccess) {
                Log.d(TAG, "All items processed successfully")
                Result.success()
            } else {
                Log.w(TAG, "Some items failed to upload, will retry via WorkManager scheduler")
                // We return success because we've processed the items and marked failures in the DB.
                // The periodic scanner or manual sync will pick them up again.
                Result.success()
            }
        } catch (e: Exception) {
            Log.e(TAG, "Upload worker failed", e)
            Result.retry()
        }
    }
}
