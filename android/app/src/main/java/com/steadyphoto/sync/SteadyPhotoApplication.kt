package com.steadyphoto.sync

import android.app.Application
import androidx.work.ExistingPeriodicWorkPolicy
import androidx.work.PeriodicWorkRequestBuilder
import androidx.work.WorkManager
import com.steadyphoto.sync.data.remote.api.ApiClient
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
        
        // Initialize ApiClient BEFORE Koin startup to ensure BASE_URL is loaded from SharedPreferences
        // before any Retrofit instance is created by dependency injection.
        // This prevents the "failed to connect to localhost/127.0.0.1:8080" error when users 
        // configure a custom API URL in SetupScreen.
        ApiClient.init(this)
        
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
        
        // Initialize Koin for dependency injection (after ApiClient is initialized)
        startKoin {
            androidLogger()
            androidContext(this@SteadyPhotoApplication)
            modules(appModule, appContainerModule)
        }
    }
}
