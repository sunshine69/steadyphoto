package com.steadyphoto.sync.worker

import android.content.Context
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.Data
import androidx.work.WorkerParameters
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.data.repository.UploadProgressCallback
import com.steadyphoto.sync.data.repository.UploadSessionProgress
import com.steadyphoto.sync.di.AppContainer

import org.koin.core.component.KoinComponent
import org.koin.core.component.inject

class UploadWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params), KoinComponent {

    private val container: AppContainer by inject()

    override suspend fun doWork(): Result {
        return try {
            // Get items for upload. Include UPLOADING status in case a previous worker crashed
            // and left items in that state.
            val itemsToUpload = container.mediaItemDao.getPendingAndFailedItems(
                listOf(UploadStatus.PENDING, UploadStatus.FAILED, UploadStatus.UPLOADING)
            )
            
            if (itemsToUpload.isEmpty()) {
                Log.d("UploadWorker", "No items to upload")
                return Result.success()
            }
            
            Log.d("UploadWorker", "Found ${itemsToUpload.size} items to upload")

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
                    Log.d("UploadWorker", "Upload session complete: $successCount succeeded, $failureCount failed")
                }

                override suspend fun onUploadError(error: Throwable) {
                    Log.e("UploadWorker", "Upload session error: ${error.message}", error)
                }
            }
            
            // Use the UploadManager for uploads. 
            // Note: UploadManager ALREADY updates the database status for each item (PENDING -> UPLOADING -> UPLOADED/FAILED).
            // We don't need to manually update statuses here anymore.
            val result = container.uploadManager.uploadMedia(itemsToUpload, progressCallback)
            
            if (result.isSuccess) {
                Log.d("UploadWorker", "All items processed successfully")
                Result.success()
            } else {
                Log.w("UploadWorker", "Some items failed to upload, will retry via WorkManager scheduler")
                // We return success because we've processed the items and marked failures in the DB.
                // The periodic scanner or manual sync will pick them up again.
                Result.success()
            }
        } catch (e: Exception) {
            Log.e("UploadWorker", "Upload worker failed", e)
            Result.retry()
        }
    }
}
