package com.steadyphoto.sync

import android.app.Application
import android.content.Intent
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.di.appModule
import com.steadyphoto.sync.worker.SyncManager
import com.steadyphoto.sync.worker.SyncService
import org.koin.android.ext.koin.androidContext
import org.koin.android.ext.koin.androidLogger
import org.koin.core.context.startKoin
import org.koin.android.ext.android.inject

class SteadyPhotoApplication : Application() {

    // Use lazy injection to ensure Koin is started before we access it
    private val syncManager: SyncManager by inject()

    override fun onCreate() {
        super.onCreate()
        
        // 1. Initialize ApiClient BEFORE Koin startup
        ApiClient.init(this)
        
        // 2. Initialize Koin for dependency injection using the consolidated module
        startKoin {
            androidLogger()
            androidContext(this@SteadyPhotoApplication)
            modules(appModule)
        }

        // 3. Start the foreground SyncService for real-time monitoring.
        try {
            val intent = Intent(this, SyncService::class.java).apply {
                action = SyncService.ACTION_START_SYNC
            }
            startForegroundService(intent)
        } catch (e: Exception) {
            android.util.Log.e("SteadyPhoto", "Failed to start SyncService in onCreate", e)
        }

        // 4. Use SyncManager to orchestrate fallback background work via WorkManager.
        syncManager.schedulePeriodicSync()
    }
}
