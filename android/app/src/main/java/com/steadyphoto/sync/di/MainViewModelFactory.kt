package com.steadyphoto.sync.di

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import com.steadyphoto.sync.ui.screens.main.MainViewModel
import com.steadyphoto.sync.ui.screens.settings.SettingsViewModel

/**
 * Custom ViewModelFactory that provides AppContainer to ViewModels.
 */
class MainViewModelFactory(
    private val container: AppContainer
) : ViewModelProvider.Factory {

    override fun <T : ViewModel> create(modelClass: Class<T>): T {
        if (modelClass.isAssignableFrom(MainViewModel::class.java)) {
            return MainViewModel(container) as T
        }
        if (modelClass.isAssignableFrom(SettingsViewModel::class.java)) {
            return SettingsViewModel(
                container.settingsRepository,
                container.syncManager
            ) as T
        }
        throw IllegalArgumentException("Unknown ViewModel class")
    }
}
