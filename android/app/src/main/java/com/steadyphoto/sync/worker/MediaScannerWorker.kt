package com.steadyphoto.sync.worker

import android.content.Context
import android.media.MediaScannerConnection
import android.os.Environment
import android.util.Log
import androidx.work.CoroutineWorker
import androidx.work.WorkerParameters
import com.steadyphoto.sync.data.local.entity.UploadStatus
import com.steadyphoto.sync.di.AppContainer
import org.koin.core.component.KoinComponent
import org.koin.core.component.inject
import java.io.File
import kotlin.coroutines.resume
import kotlin.coroutines.suspendCoroutine

class MediaScannerWorker(
    context: Context,
    params: WorkerParameters
) : CoroutineWorker(context, params), KoinComponent {

    private val container: AppContainer by inject()
    // Inject SyncManager via Koin since we are a KoinComponent
    private val syncManager: SyncManager by inject()

    override suspend fun doWork(): Result {
        return try {
            Log.d("MediaScannerWorker", "Background media scan started")

            // 1. Force a MediaStore scan of the common camera directories.
            scanCameraDirectories()

            // 2. Perform the database scan
            val scanResult = container.repository.scanNewMedia(forceFullScan = true)

            when (scanResult) {
                is com.steadyphoto.sync.data.repository.ScanResult.Success -> {
                    Log.d("MediaScannerWorker", "Scan success: ${scanResult.newItemsInserted} new items")
                }
                else -> Log.d("MediaScannerWorker", "No new items found via MediaStore")
            }

            // 3. Check for any pending items (including stuck ones)
            val pendingItems = container.mediaItemDao.getPendingAndFailedItems(
                listOf(UploadStatus.PENDING, UploadStatus.FAILED, UploadStatus.UPLOADING)
            )
            
            if (pendingItems.isNotEmpty()) {
                Log.d("MediaScannerWorker", "Found ${pendingItems.size} items to sync, triggering uploader")
                syncManager.triggerUpload()
            }

            Result.success()
        } catch (e: Exception) {
            Log.e("MediaScannerWorker", "Scan phase failed", e)
            Result.retry()
        }
    }

    private suspend fun scanCameraDirectories() {
        val paths = listOf(
            File(Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DCIM), "Camera"),
            Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_PICTURES),
            Environment.getExternalStoragePublicDirectory(Environment.DIRECTORY_DCIM)
        ).filter { it.exists() }.map { it.absolutePath }.toTypedArray()

        if (paths.isEmpty()) return

        Log.d("MediaScannerWorker", "Requesting MediaStore scan for: ${paths.joinToString()}")
        
        suspendCoroutine { continuation ->
            MediaScannerConnection.scanFile(applicationContext, paths, null) { path, uri ->
                Log.v("MediaScannerWorker", "Scanned $path -> $uri")
            }
            continuation.resume(Unit)
        }
    }
}
