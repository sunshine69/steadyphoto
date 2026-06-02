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
import com.steadyphoto.sync.ui.screens.auth.AuthViewModel
import com.steadyphoto.sync.ui.screens.main.MainViewModel
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
    val uploadManager: com.steadyphoto.sync.data.repository.UploadManager
) {
    val mediaItemDao: MediaItemDao
        get() = database.mediaItemDao()
}

val appModule = module {
    // ApiClient is now the single point of API access - components use apiClient.apiService 
    // directly, which ensures they always get a fresh instance when base URL changes.
    single<ApiClient> { com.steadyphoto.sync.data.remote.api.ApiClient }
    
    single<SyncDatabase> { SyncDatabase.getDatabase(androidContext()) }
    single<MediaItemDao> { get<SyncDatabase>().mediaItemDao() }
    single<PermissionHelper> { PermissionHelper(androidContext()) }
    single<NetworkConnectivityMonitor> { NetworkConnectivityMonitor(androidContext()) }
    
    // UploadManager now takes ApiClient instead of ApiService - it will call 
    // apiClient.apiService for each API request, ensuring the latest URL is always used.
    single {
        com.steadyphoto.sync.data.repository.UploadManager(
            context = androidContext(),
            apiClient = get(),
            mediaItemDao = get(),
            networkMonitor = get()
        )
    }
    
    single<SyncRepository> { 
        SyncRepositoryImpl(
            context = androidContext(),
            mediaItemDao = get(),
            uploadManager = get(),
            apiClient = get()
        )
    }
    
    // ViewModel declarations for Koin - AuthViewModel no longer takes ApiService as a parameter
    factory { AuthViewModel() }
    factory { MainViewModel(get<AppContainer>()) }
}

val appContainerModule = module {
    single<AppContainer> {
        AppContainer(
            apiClient = get(),
            database = get(),
            repository = get(),
            permissionHelper = get(),
            uploadManager = get()
        )
    }
}
