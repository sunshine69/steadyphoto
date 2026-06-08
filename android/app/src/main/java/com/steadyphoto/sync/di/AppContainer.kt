package com.steadyphoto.sync.di

import android.content.Context
import androidx.activity.ComponentActivity
import androidx.core.content.ContextCompat
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.SyncDatabase
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.data.repository.NetworkConnectivityMonitor
import com.steadyphoto.sync.data.repository.SyncRepository
import com.steadyphoto.sync.data.repository.SyncRepositoryImpl
import com.steadyphoto.sync.data.settings.SettingsRepository
import com.steadyphoto.sync.ui.screens.auth.AuthViewModel
import com.steadyphoto.sync.ui.screens.main.MainViewModel
import com.steadyphoto.sync.ui.screens.settings.SettingsViewModel
import com.steadyphoto.sync.worker.SyncManager
import org.koin.android.ext.koin.androidContext
import org.koin.dsl.module

/**
 * Permission helper that can be injected and used with the application context.
 */
class PermissionHelper(private val context: Context) {
    
    fun hasMediaPermissions(): Boolean {
        return com.steadyphoto.sync.util.PermissionHelper.hasMediaPermissions(context)
    }
    
    fun getPermissionsToRequest(): Array<String> {
        return com.steadyphoto.sync.util.PermissionHelper.getMediaPermissionsToRequest(context)
    }
    
    fun shouldShowRationale(activity: ComponentActivity): Boolean {
        return com.steadyphoto.sync.util.PermissionHelper.shouldShowPermissionRationale(context, activity)
    }
    
    /**
     * Check if permission was permanently denied.
     */
    fun isPermanentlyDenied(activity: ComponentActivity): Boolean {
        return com.steadyphoto.sync.util.PermissionHelper.isPermissionPermanentlyDenied(context, activity)
    }
}

// App Container for dependency access outside Koin scope - includes ApiClient so components 
// can get the latest ApiService when API URL changes (e.g., after SetupScreen configuration).
class AppContainer(
    val apiClient: ApiClient,
    val database: SyncDatabase,
    val repository: SyncRepository,
    val permissionHelper: PermissionHelper,
    val uploadManager: com.steadyphoto.sync.data.repository.UploadManager,
    val settingsRepository: SettingsRepository,
    val syncManager: SyncManager
) {
    val mediaItemDao: MediaItemDao
        get() = database.mediaItemDao()
}

// We provide a single module that contains everything to avoid "unresolved reference" errors 
// in the Application class when trying to split modules incorrectly.
val appModule = module {
    // API Client singleton (it's an object, so we use it directly)
    single<ApiClient> { com.steadyphoto.sync.data.remote.api.ApiClient }
    
    // Database & DAOs
    single<SyncDatabase> { SyncDatabase.getDatabase(androidContext()) }
    single<MediaItemDao> { get<SyncDatabase>().mediaItemDao() }
    
    // Utilities
    single<PermissionHelper> { PermissionHelper(androidContext()) }
    single<NetworkConnectivityMonitor> { NetworkConnectivityMonitor(androidContext()) }
    
    // Repositories & Managers
    single<SettingsRepository> { SettingsRepository(androidContext()) }
    single<SyncManager> { SyncManager(androidContext()) }
    
    single<com.steadyphoto.sync.data.repository.UploadManager> {
        com.steadyphoto.sync.data.repository.UploadManager(
            context = androidContext(),
            apiClient = get(),
            mediaItemDao = get(),
            networkMonitor = get(),
            settingsRepository = get()
        )
    }
    
    single<SyncRepository> { 
        SyncRepositoryImpl(
            context = androidContext(),
            apiClient = get(),
            mediaItemDao = get()
        )
    }

    // App Container - Providing the container itself as a singleton so ViewModels can inject it.
    single<AppContainer> {
        AppContainer(
            apiClient = get(),
            database = get(),
            repository = get(),
            permissionHelper = get(),
            uploadManager = get(),
            settingsRepository = get(),
            syncManager = get()
        )
    }

    // ViewModels - using factory to provide new instances per screen request.
    factory { AuthViewModel() }
    factory { MainViewModel(get<AppContainer>()) }
    factory { SettingsViewModel(get(), get()) }
}
