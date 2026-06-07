package com.steadyphoto.sync

import android.app.Application
import android.content.Intent
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.di.appContainerModule
import com.steadyphoto.sync.di.appModule
import com.steadyphoto.sync.worker.SyncManager
import com.steadyphoto.sync.worker.SyncService
import org.koin.android.ext.koin.androidContext
import org.koin.android.ext.koin.androidLogger
import org.koin.core.context.startKoin

class SteadyPhotoApplication : Application() {

    override fun onCreate() {
        super.onCreate()
        
        // 1. Initialize ApiClient BEFORE Koin startup
        ApiClient.init(this)
        
        // 2. Initialize Koin for dependency injection
        startKoin {
            androidLogger()
            androidContext(this@SteadyPhotoApplication)
            modules(appModule, appContainerModule)
        }

        // 3. Start the foreground SyncService for real-time monitoring.
        // This ensures FileObserver and ContentObserver are active as long as the app is "running".
        try {
            val intent = Intent(this, SyncService::class.java).apply {
                action = SyncService.ACTION_START_SYNC
            }
            startForegroundService(intent)
        } catch (e: Exception) {
            android.util.Log.e("SteadyPhoto", "Failed to start SyncService in onCreate", e)
        }

        // 4. Use SyncManager to orchestrate fallback background work via WorkManager.
        SyncManager.getInstance(this).schedulePeriodicSync()
    }
}
