package com.steadyphoto.sync.ui.screens.main

import android.Manifest
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.data.repository.ScanResult
import com.steadyphoto.sync.di.AppContainer
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

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
    val showPermissionRationale: Boolean = false
)

class MainViewModel(
    private val container: AppContainer
) : ViewModel() {

    private val _uiState = MutableStateFlow(MainUiState())
    val uiState: StateFlow<MainUiState> = _uiState

    init {
        loadCounts()
    }

    private fun loadCounts() {
        viewModelScope.launch {
            container.mediaItemDao.getPendingCount().collect { pending ->
                container.mediaItemDao.getUploadedCount().collect { uploaded ->
                    _uiState.value = _uiState.value.copy(
                        pendingCount = pending,
                        uploadedCount = uploaded
                    )
                }
            }
        }
    }

    /**
     * Checks if the required media permissions are granted.
     * Returns true if all necessary permissions are available.
     */
    fun hasMediaPermissions(): Boolean {
        return container.permissionHelper.hasMediaPermissions()
    }

    /**
     * Requests media permissions from the user.
     * This should be called when startSync is invoked but permissions are missing.
     */
    fun requestMediaPermissions(activity: androidx.activity.ComponentActivity, onPermissionsResult: (Boolean) -> Unit) {
        viewModelScope.launch {
            val permissions = container.permissionHelper.getPermissionsToRequest()
            
            if (permissions.isEmpty()) {
                // All permissions already granted
                onPermissionsResult(true)
                return@launch
            }
            
            // Request permissions through the activity
            // Note: In a real implementation, you would use ActivityResultContracts.RequestMultiplePermissions
            // For now, we'll check and report status
            if (container.permissionHelper.hasMediaPermissions()) {
                onPermissionsResult(true)
            } else {
                val permanentlyDenied = container.permissionHelper.isPermanentlyDenied(activity)
                _uiState.value = _uiState.value.copy(
                    showPermissionRationale = !permanentlyDenied,
                    errorMessage = if (permanentlyDenied) {
                        "Media permissions have been permanently denied. Please enable them in Settings."
                    } else null
                )
                onPermissionsResult(false)
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
        } else {
            _uiState.value = _uiState.value.copy(syncState = SyncUiState.Success(0))
        }
    }

    fun clearError() {
        _uiState.value = _uiState.value.copy(errorMessage = null, showPermissionRationale = false)
    }
}
