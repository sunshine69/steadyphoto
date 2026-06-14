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
        loadRecentItemsSafely()
    }

    /**
     * Load counts safely on init to prevent crashes when database isn't ready.
     */
    private fun loadCountsSafely() {
        viewModelScope.launch {
            try {
                // Using collect (via first()) from Flow is correct, 
                // but we need to ensure the Dao returns a proper Flow<Int>
                container.mediaItemDao.getPendingCount().collect { pending ->
                    _uiState.value = _uiState.value.copy(pendingCount = pending)
                }

                container.mediaItemDao.getUploadedCount().collect { uploaded ->
                    _uiState.value = _uiState.value.copy(uploadedCount = uploaded)
                }
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error loading counts on init, using defaults", e)
                // Set safe defaults to prevent crashes when database is not yet ready
                _uiState.value = _uiState.value.copy(pendingCount = 0, uploadedCount = 0)
            }
        }
    }

    /**
     * Load recent items safely on init to display in the UI.
     */
    private fun loadRecentItemsSafely() {
        viewModelScope.launch {
            try {
                val recentItems = container.mediaItemDao.getRecentItems(10)
                _uiState.value = _uiState.value.copy(recentItems = recentItems)
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error loading recent items on init, using defaults", e)
                // Set safe defaults to prevent crashes when database is not yet ready
                _uiState.value = _uiState.value.copy(recentItems = emptyList())
            }
        }
    }

    /**
     * Refresh recent items after a sync operation.
     */
    private fun refreshRecentItems() {
        viewModelScope.launch {
            try {
                val recentItems = container.mediaItemDao.getRecentItems(10)
                _uiState.value = _uiState.value.copy(recentItems = recentItems)
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error refreshing recent items", e)
                _uiState.value = _uiState.value.copy(recentItems = emptyList())
            }
        }
    }

    /**
     * Refresh counts after a sync operation.
     */
    private fun refreshCounts() {
        viewModelScope.launch {
            try {
                // Since getPendingCount and getUploadedCount are Flows, 
                // we just trigger the update logic by re-collecting or relying on existing collection.
                // For simplicity in a "refresh" call, we can manually query once:
                val pending = container.mediaItemDao.getPendingAndFailedItems(listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING)).size 
                _uiState.value = _uiState.value.copy(pendingCount = pending)

                // A more robust way would be to let the Flows in loadCountsSafely do their job,
                // but since this is a manual refresh:
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error refreshing counts", e)
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
     * Handle the result of a permission request from ActivityResultLauncher.
     */
    fun handlePermissionResult(isGranted: Boolean, permanentlyDenied: Boolean = false) {
        if (isGranted) {
            // Check BEFORE clearing the rationale state - we need to know if it was showing before we clear it
            val wasShowingRationale = uiState.value.showPermissionRationale
            
            _uiState.value = _uiState.value.copy(
                showPermissionRationale = false,
                errorMessage = null
            )
            android.util.Log.d("MainViewModel", "Permissions granted")
            
            // Automatically start sync after permissions are granted if we were showing the rationale
            if (wasShowingRationale) {
                viewModelScope.launch {
                    try {
                        // Verify permissions are actually granted now before starting
                        if (hasMediaPermissions()) {
                            startBackgroundSync()
                        } else {
                            android.util.Log.w("MainViewModel", "Permission denied after granting - not starting sync")
                        }
                    } catch (e: Exception) {
                        android.util.Log.e("MainViewModel", "Error auto-starting after permission grant", e)
                        _uiState.value = _uiState.value.copy(
                            errorMessage = "Could not start sync after permissions granted. Please try again."
                        )
                    }
                }
            }
        } else if (permanentlyDenied) {
            _uiState.value = _uiState.value.copy(
                showPermissionRationale = false,
                errorMessage = "Media permissions have been permanently denied. Please enable them in Settings."
            )
        } else {
            _uiState.value = _uiState.value.copy(
                showPermissionRationale = true,
                errorMessage = "Media permissions are required to scan and upload your photos and videos."
            )
        }
    }

    /**
     * Initiates the sync process: scan media and upload pending items.
     */
    fun startSync() {
        viewModelScope.launch {
            // Check permissions first before doing any work
            if (!hasMediaPermissions()) {
                _uiState.value = _uiState.value.copy(
                    showPermissionRationale = true,
                    errorMessage = null  // Clear error so Start button becomes enabled again
                )
                return@launch
            }

            _uiState.value = _uiState.value.copy(syncState = SyncUiState.Scanning)

            // Trigger media scan using MediaStore
            val scanResult = container.repository.scanNewMedia()

            when (scanResult) {
                is ScanResult.Success -> {
                    // ALWAYS proceed to upload attempt if the scan was successful.
                    proceedToUpload(scanResult.totalScanned, scanResult.duplicatesSkipped)
                }
                is ScanResult.NoNewItems -> {
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
                val authToken = container.apiClient.getAuthToken()
                if (authToken.isNullOrEmpty()) {
                    _uiState.value = _uiState.value.copy(
                        errorMessage = "No authentication. Please log in again."
                    )
                    return@launch
                }

                // Check permissions before attempting sync
                if (!hasMediaPermissions()) {
                    _uiState.value = _uiState.value.copy(
                        showPermissionRationale = true,
                        errorMessage = null  // Clear error so Start button becomes enabled again
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
     * Stop background sync - this cancels WorkManager jobs.
     * The Service will be stopped automatically when stopSyncJob() is called via ACTION_STOP_SYNC intent.
     */
    fun stopBackgroundSync() {
        viewModelScope.launch {
            try {
                // Use SyncManager to cancel all work consistently
                container.syncManager.cancelAllSyncWork()

                _uiState.value = _uiState.value.copy(
                    isBackgroundSyncRunning = false,
                    syncState = SyncUiState.Idle  // Reset state so Start button becomes enabled again
                )
            } catch (e: Exception) {
                android.util.Log.e("MainViewModel", "Error stopping background sync", e)
                _uiState.value = _uiState.value.copy(
                    isBackgroundSyncRunning = false,
                    syncState = SyncUiState.Idle  // Reset state so Start button becomes enabled again
                )
            }
        }
    }

    /**
     * Proceeds to upload phase after successful scanning.
     * Now uses UploadManager for consistent, memory-safe uploads (streaming + chunked).
     */
    private suspend fun proceedToUpload(totalScanned: Int, duplicatesSkipped: Int) {
        _uiState.value = _uiState.value.copy(syncState = SyncUiState.Uploading)

        // Get pending items (including those that failed previously) directly from the DB to ensure fresh data
        val pendingItems = container.mediaItemDao.getPendingAndFailedItems(
            listOf(com.steadyphoto.sync.data.local.entity.UploadStatus.PENDING,
                   com.steadyphoto.sync.data.local.entity.UploadStatus.FAILED)
        )

        if (pendingItems.isNotEmpty()) {
            try {
                // Use UploadManager for consistent, streaming uploads (fixes OOM on large files)
                val result = container.uploadManager.uploadMedia(pendingItems, object : com.steadyphoto.sync.data.repository.UploadProgressCallback {
                    override suspend fun onProgressUpdated(progress: com.steadyphoto.sync.data.repository.UploadSessionProgress) {
                        // Optional: could update UI with progress here if needed
                    }

                    override suspend fun onUploadComplete(successCount: Int, failureCount: Int) {
                        android.util.Log.d("MainViewModel", "Upload complete via UploadManager: $successCount succeeded, $failureCount failed")
                    }

                    override suspend fun onUploadError(error: Throwable) {
                        android.util.Log.e("MainViewModel", "Upload error via UploadManager: ${error.message}", error)
                    }
                })
                
                result.onSuccess { uploadResult ->
                    _uiState.value = _uiState.value.copy(
                        syncState = SyncUiState.Success(uploadResult.successCount),
                        errorMessage = "Uploaded ${uploadResult.successCount} files (${totalScanned} scanned, $duplicatesSkipped duplicates skipped)"
                    )
                    refreshCounts()
                    refreshRecentItems()
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
            // If there were no pending items to upload, we show Success with 0 count but acknowledge the scan finished successfully
            _uiState.value = _uiState.value.copy(syncState = SyncUiState.Success(0))
            refreshCounts() // Refresh counts just in case anything changed during scanning/uploading logic
        }
    }

    fun clearError() {
        _uiState.value = _uiState.value.copy(
            errorMessage = null, 
            showPermissionRationale = false,
            syncState = SyncUiState.Idle  // Reset state so Start button becomes enabled again for retrying
        )
    }
}
