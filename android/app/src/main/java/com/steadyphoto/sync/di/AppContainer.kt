package com.steadyphoto.sync.di

import android.content.Context
import androidx.activity.ComponentActivity
import androidx.core.content.ContextCompat
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.SyncDatabase
import com.steadyphoto.sync.data.remote.api.ApiService
import com.steadyphoto.sync.data.repository.NetworkConnectivityMonitor
import com.steadyphoto.sync.data.repository.SyncRepository
import com.steadyphoto.sync.data.repository.SyncRepositoryImpl
import com.steadyphoto.sync.data.repository.UploadManager
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

// App Container for dependency access outside Koin scope
class AppContainer(
    val apiService: ApiService,
    val database: SyncDatabase,
    val repository: SyncRepository,
    val permissionHelper: PermissionHelper,
    val uploadManager: UploadManager
) {
    val mediaItemDao: MediaItemDao
        get() = database.mediaItemDao()
}

val appModule = module {
    single<ApiService> { com.steadyphoto.sync.data.remote.api.ApiClient.apiService }
    single<SyncDatabase> { SyncDatabase.getDatabase(androidContext()) }
    single<MediaItemDao> { get<SyncDatabase>().mediaItemDao() }
    single<PermissionHelper> { PermissionHelper(androidContext()) }
    single<NetworkConnectivityMonitor> { NetworkConnectivityMonitor(androidContext()) }
    
    single<UploadManager> {
        UploadManager(
            context = androidContext(),
            apiService = get(),
            mediaItemDao = get(),
            networkMonitor = get()
        )
    }
    
    single<SyncRepository> { 
        SyncRepositoryImpl(
            context = androidContext(),
            mediaItemDao = get(),
            container = get()
        )
    }
    
    // ViewModel declarations for Koin
    factory { AuthViewModel(get()) }
    factory { MainViewModel(get<AppContainer>()) }
}

val appContainerModule = module {
    single<AppContainer> {
        AppContainer(
            apiService = get(),
            database = get(),
            repository = get(),
            permissionHelper = get(),
            uploadManager = get()
        )
    }
}
