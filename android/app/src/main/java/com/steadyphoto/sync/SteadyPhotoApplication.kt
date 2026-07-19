package com.steadyphoto.sync

import android.app.Application
import android.content.Intent
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.di.appModule
import com.steadyphoto.sync.worker.SyncManager
import com.steadyphoto.sync.worker.SyncService
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import kotlinx.coroutines.SupervisorJob
import org.koin.android.ext.koin.androidContext
import org.koin.android.ext.koin.androidLogger
import org.koin.core.context.startKoin
import org.koin.android.ext.android.inject

class SteadyPhotoApplication : Application() {

    // Use lazy injection to ensure Koin is started before we access it
    private val syncManager: SyncManager by inject()
    private val settingsRepository: com.steadyphoto.sync.data.settings.SettingsRepository by inject()

    // CoroutineScope for background initialization tasks (e.g. auto-start at boot)
    private val appScope = CoroutineScope(SupervisorJob() + Dispatchers.Main)

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

        // Launch auto-start logic in a coroutine (not runBlocking) to avoid blocking
        // the main thread during boot, which can cause ANR on some devices.
        appScope.launch {
            val autoStartAtBoot = settingsRepository.autoStartAtBootFlow.first()

            if (autoStartAtBoot) {
                // 3. Start the foreground SyncService for real-time monitoring.
                try {
                    val intent = Intent(this@SteadyPhotoApplication, SyncService::class.java).apply {
                        action = SyncService.ACTION_START_SYNC
                    }
                    startForegroundService(intent)
                } catch (e: Exception) {
                    android.util.Log.e("SteadyPhoto", "Failed to start SyncService in onCreate", e)
                }

                // 4. Use SyncManager to orchestrate fallback background work via WorkManager.
                syncManager.schedulePeriodicSync()
            } else {
                android.util.Log.d("SteadyPhoto", "Auto-start at boot is disabled - skipping auto-start.")
            }
        }
    }
}
