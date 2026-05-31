package com.steadyphoto.sync

import android.app.Application
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.steadyphoto.sync.di.appContainerModule
import com.steadyphoto.sync.di.appModule
import com.steadyphoto.sync.worker.MediaScannerWorker
import com.steadyphoto.sync.worker.UploadWorker
import org.koin.android.ext.koin.androidContext
import org.koin.android.ext.koin.androidLogger
import org.koin.core.context.startKoin
import java.util.concurrent.TimeUnit

class SteadyPhotoApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        
        // Initialize Koin for dependency injection
        startKoin {
            androidLogger()
            androidContext(this@SteadyPhotoApplication)
            modules(appModule, appContainerModule)
        }
        
        // Schedule periodic media scanning (every 6 hours)
        val scanWorkRequest = PeriodicWorkRequestBuilder<MediaScannerWorker>(
            6, TimeUnit.HOURS
        ).build()
        
        WorkManager.getInstance(this).enqueueUniquePeriodicWork(
            "media_scan",
            ExistingPeriodicWorkPolicy.KEEP,
            scanWorkRequest
        )
        
        // Schedule periodic upload check (every 15 minutes)
        val uploadWorkRequest = PeriodicWorkRequestBuilder<UploadWorker>(
            15, TimeUnit.MINUTES
        ).build()
        
        WorkManager.getInstance(this).enqueueUniquePeriodicWork(
            "media_upload",
            ExistingPeriodicWorkPolicy.KEEP,
            uploadWorkRequest
        )
    }
}
