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
            // Get pending and failed items for upload
            val itemsToUpload = container.mediaItemDao.getPendingAndFailedItems(
                listOf(UploadStatus.PENDING, UploadStatus.FAILED)
            )
            
            if (itemsToUpload.isEmpty()) {
                Log.d("UploadWorker", "No items to upload")
                return Result.success()
            }
            
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
                    Log.d("UploadWorker", "Upload complete: $successCount succeeded, $failureCount failed")
                }

                override suspend fun onUploadError(error: Throwable) {
                    Log.e("UploadWorker", "Upload error from callback: ${error.message}", error)
                }
            }
            
            // Use the enhanced UploadManager for uploads with progress tracking
            var uploadedCount = 0
            
            for (chunk in itemsToUpload.chunked(10)) {
                try {
                    val result = container.uploadManager.uploadMedia(chunk, progressCallback)
                    
                    if (result.isSuccess) {
                        val uploadResult = result.getOrNull()
                        uploadedCount += uploadResult?.successCount ?: chunk.size
                        
                        // Mark successfully uploaded items
                        uploadResult?.let { res ->
                            var successItems = 0
                            for (item in chunk) {
                                if (successItems < res.successCount) {
                                    container.mediaItemDao.updateStatus(item.id, UploadStatus.UPLOADED)
                                    successItems++
                                } else {
                                    // Mark remaining as failed
                                    container.mediaItemDao.updateStatus(
                                        item.id, 
                                        UploadStatus.FAILED, 
                                        "Upload batch partially failed"
                                    )
                                }
                            }
                        }
                    } else {
                        val error = result.exceptionOrNull()
                        Log.e("UploadWorker", "Batch upload failed: ${error?.message}", error)
                        
                        // Mark all items as failed for retry
                        chunk.forEach { item ->
                            container.mediaItemDao.updateStatus(
                                item.id, 
                                UploadStatus.FAILED, 
                                error?.message ?: "Upload failed"
                            )
                        }
                    }
                } catch (e: Exception) {
                    Log.e("UploadWorker", "Batch upload exception", e)
                    
                    // Mark all items as failed for retry
                    chunk.forEach { item ->
                        container.mediaItemDao.updateStatus(
                            item.id, 
                            UploadStatus.FAILED, 
                            e.message ?: "Upload failed"
                        )
                    }
                }
            }
            
            Log.d("UploadWorker", "Uploaded $uploadedCount items")
            Result.success()
        } catch (e: Exception) {
            Log.e("UploadWorker", "Upload worker failed", e)
            Result.retry()
        }
    }
}
