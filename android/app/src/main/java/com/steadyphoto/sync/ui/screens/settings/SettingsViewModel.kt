package com.steadyphoto.sync.ui.screens.settings

import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import kotlinx.coroutines.flow.SharingStarted
import kotlinx.coroutines.flow.SharedFlow
import kotlinx.coroutines.flow.map
import kotlinx.coroutines.flow.shareIn
import kotlinx.coroutines.launch
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.data.settings.NetworkSettings
import com.steadyphoto.sync.data.settings.PreferredNetworkType

class SettingsViewModel(
    private val settingsRepository: com.steadyphoto.sync.data.settings.SettingsRepository,
    private val syncManager: com.steadyphoto.sync.worker.SyncManager
) : ViewModel() {

    val networkSettingsFlow: SharedFlow<NetworkSettings> = 
        settingsRepository.networkSettingsFlow.map { it }.shareIn(
            viewModelScope,
            SharingStarted.Lazily,
            1
        )

    val syncControlFlow: SharedFlow<Boolean> = 
        settingsRepository.syncControlFlow.map { it.autoSyncEnabled }.shareIn(
            viewModelScope,
            SharingStarted.Lazily,
            1
        )

    val autoStartAtBootFlow: SharedFlow<Boolean> = 
        settingsRepository.autoStartAtBootFlow.shareIn(
            viewModelScope,
            SharingStarted.Lazily,
            1
        )

    fun updateApiUrl(url: String) {
        ApiClient.updateBaseUrl(url.trim())
    }

    fun setWifiOnly(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setWifiOnly(enabled)
            if (enabled) {
                settingsRepository.setPreferredNetworkType(PreferredNetworkType.WIFI_ONLY)
            } else {
                settingsRepository.setPreferredNetworkType(PreferredNetworkType.ANY_NETWORK)
            }
            syncManager.rescheduleWithCurrentSettings()
        }
    }

    fun setSyncOnMeteredConnection(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setSyncOnMeteredConnection(enabled)
            syncManager.rescheduleWithCurrentSettings()
        }
    }

    fun setAutoSyncEnabled(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setAutoSyncEnabled(enabled)
            syncManager.rescheduleWithCurrentSettings()
        }
    }

    fun setAutoStartAtBoot(enabled: Boolean) {
        viewModelScope.launch {
            settingsRepository.setAutoStartAtBoot(enabled)
            syncManager.rescheduleWithCurrentSettings()
        }
    }
}
