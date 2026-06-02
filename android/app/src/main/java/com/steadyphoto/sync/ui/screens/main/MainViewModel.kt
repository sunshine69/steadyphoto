package com.steadyphoto.sync.ui.screens.main

import android.Manifest
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import androidx.work.*
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.data.repository.ScanResult
import com.steadyphoto.sync.di.AppContainer
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.flow.first
import kotlinx.coroutines.launch
import java.util.concurrent.TimeUnit

sealed class SyncUiState {
    object Idle : SyncUiState()
    object Scanning : SyncUiState()
    object Uploading : SyncUiState()
    data class Success(val count: Int) : SyncUiState()
    data class Error(val message: String) : SyncUiState()
}

data class MainUiState(
    val pendingCount: Int = 0,
    val uploadedCount: Int = 0,
    val syncState: SyncUiState = SyncUiState.Idle,
    val recentItems: List<MediaItemEntity> = emptyList(),
    val errorMessage: String? = null,
    val showPermissionRationale: Boolean = false,
    val isBackgroundSyncRunning: Boolean = false
)

class MainViewModel(
    private val container: AppContainer
) : ViewModel() {

    private val _uiState = MutableStateFlow(MainUiState())
    val uiState: StateFlow<MainUiState> = _uiState

    init {
        loadCountsSafely()
    }

    /**
     * Load counts safely on init to prevent crashes when database isn't ready.
     */
    private fun loadCountsSafely() {
        viewModelScope.launch {
            try {
                val pending = container.mediaItemDao.getPendingCount().first()
                _uiState.value = _uiState.value.copy(pendingCount = pending)

                val uploaded = container.mediaItemDao.getUploadedCount().first()
                _uiState.value = _uiState.value.copy(uploadedCount = uploaded)
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error loading counts on init, using defaults", e)
                // Set safe defaults to prevent crashes
                _uiState.value = _uiState.value.copy(pendingCount = 0, uploadedCount = 0)
            }
        }
    }

    /**
     * Checks if the required media permissions are granted.
     */
    fun hasMediaPermissions(): Boolean {
        return try {
            container.permissionHelper.hasMediaPermissions()
        } catch (e: Exception) {
            android.util.Log.e("MainViewModel", "Error checking permissions", e)
            false
        }
    }

    /**
     * Requests media permissions from the user.
     */
    fun requestMediaPermissions(activity: androidx.activity.ComponentActivity, onPermissionsResult: ((Boolean) -> Unit)? = null) {
        viewModelScope.launch {
            try {
                val permissions = container.permissionHelper.getPermissionsToRequest()

                if (permissions.isEmpty()) {
                    // All permissions already granted
                    onPermissionsResult?.invoke(true)
                    return@launch
                }

                // Request permissions through the activity
                if (container.permissionHelper.hasMediaPermissions()) {
                    onPermissionsResult?.invoke(true)
                } else {
                    val permanentlyDenied = container.permissionHelper.isPermanentlyDenied(activity)
                    _uiState.value = _uiState.value.copy(
                        showPermissionRationale = !permanentlyDenied,
                        errorMessage = if (permanentlyDenied) {
                            "Media permissions have been permanently denied. Please enable them in Settings."
                        } else null
                    )
                    onPermissionsResult?.invoke(false)
                }
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error requesting permissions", e)
                _uiState.value = _uiState.value.copy(
                    errorMessage = "Could not check permissions. Please try again."
                )
                onPermissionsResult?.invoke(false)
            }
        }
    }

    /**
     * Initiates the sync process: scan media and upload pending items.
     */
    fun startSync() {
        viewModelScope.launch {
            // Check permissions first
            if (!hasMediaPermissions()) {
                _uiState.value = _uiState.value.copy(
                    errorMessage = "Storage permission is required to scan your photos and videos."
                )
                return@launch
            }

            _uiState.value = _uiState.value.copy(syncState = SyncUiState.Scanning)

            // Trigger media scan using MediaStore
            val scanResult = container.repository.scanNewMedia()

            when (scanResult) {
                is ScanResult.Success -> {
                    if (scanResult.newItemsInserted > 0 ||
                        _uiState.value.pendingCount == 0) {
                        // Proceed to upload
                        proceedToUpload(scanResult.totalScanned, scanResult.duplicatesSkipped)
                    } else {
                        _uiState.value = _uiState.value.copy(
                            syncState = SyncUiState.Success(0),
                            errorMessage = "No new items found"
                        )
                    }
                }
                is ScanResult.NoNewItems -> {
                    // No new items, nothing to upload
                    _uiState.value = _uiState.value.copy(syncState = SyncUiState.Idle)
                }
                is ScanResult.PermissionDenied -> {
                    _uiState.value = _uiState.value.copy(
                        syncState = SyncUiState.Error("Permission denied"),
                        errorMessage = "Storage permission is required to scan your photos and videos."
                    )
                }
                is ScanResult.Error -> {
                    _uiState.value = _uiState.value.copy(
                        syncState = SyncUiState.Error(scanResult.message),
                        errorMessage = scanResult.message
                    )
                }
            }
        }
    }

    /**
     * Start background sync - this starts the periodic WorkManager jobs.
     */
    fun startBackgroundSync() {
        viewModelScope.launch {
            try {
                // Check if we have a valid auth token first
                val authToken = container.uploadManager.getAuthToken()
                if (authToken.isNullOrEmpty()) {
                    _uiState.value = _uiState.value.copy(
                        errorMessage = "No authentication. Please log in again."
                    )
                    return@launch
                }

                // Start immediate upload attempt first
                startSync()

                // Set state to indicate background sync is active
                _uiState.value = _uiState.value.copy(isBackgroundSyncRunning = true)

            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error starting background sync", e)
                _uiState.value = _uiState.value.copy(
                    errorMessage = "Could not start background sync. Please try again."
                )
            }
        }
    }

    /**
     * Stop background sync - this cancels the periodic WorkManager jobs.
     */
    fun stopBackgroundSync() {
        viewModelScope.launch {
            try {
                // Cancel all periodic work for media scanning and uploading
                WorkManager.getInstance(container.uploadManager.applicationContext)
                    .cancelAllWorkByTag("media_scan")

                WorkManager.getInstance(container.uploadManager.applicationContext)
                    .cancelAllWorkByTag("media_upload")

                _uiState.value = _uiState.value.copy(
                    isBackgroundSyncRunning = false,
                    errorMessage = "Background sync stopped"
                )
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error stopping background sync", e)
                // Still update the state even if cancel fails
                _uiState.value = _uiState.value.copy(
                    isBackgroundSyncRunning = false,
                    errorMessage = "Could not stop background sync. Please try again."
                )
            }
        }
    }

    /**
     * Proceeds to upload phase after successful scanning.
     */
    private suspend fun proceedToUpload(totalScanned: Int, duplicatesSkipped: Int) {
        _uiState.value = _uiState.value.copy(syncState = SyncUiState.Uploading)

        // Get pending items and upload
        val pendingItems = container.mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                   com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )

        if (pendingItems.isNotEmpty()) {
            try {
                val result = container.repository.uploadMedia(pendingItems)
                result.onSuccess {
                    _uiState.value = _uiState.value.copy(
                        syncState = SyncUiState.Success(pendingItems.size),
                        pendingCount = 0,
                        errorMessage = "Uploaded ${pendingItems.size} files (${totalScanned} scanned, $duplicatesSkipped duplicates skipped)"
                    )
                }.onFailure { exception ->
                    _uiState.value = _uiState.value.copy(
                        syncState = SyncUiState.Error(exception.message ?: "Upload failed"),
                        errorMessage = exception.message
                    )
                }
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error during upload", e)
                _uiState.value = _uiState.value.copy(
                    syncState = SyncUiState.Error(e.message ?: "Upload failed"),
                    errorMessage = e.message
                )
            }
        } else {
            _uiState.value = _uiState.value.copy(syncState = SyncUiState.Success(0))
        }
    }

    fun clearError() {
        _uiState.value = _uiState.value.copy(errorMessage = null, showPermissionRationale = false)
    }
}
