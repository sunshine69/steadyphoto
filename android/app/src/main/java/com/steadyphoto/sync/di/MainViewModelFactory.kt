package com.steadyphoto.sync.di

import androidx.lifecycle.ViewModel
import androidx.lifecycle.ViewModelProvider
import com.steadyphoto.sync.ui.screens.main.MainViewModel

/**
 * Custom ViewModelFactory that provides AppContainer to ViewModels.
 */
class MainViewModelFactory(
    private val container: AppContainer
) : ViewModelProvider.Factory {

    @Suppress("UNCHECKED_CAST")
    override fun <T : ViewModel> create(modelClass: Class<T>): T {
        if (modelClass.isAssignableFrom(MainViewModel::class.java)) {
            return MainViewModel(container) as T
        }
        throw IllegalArgumentException("Unknown ViewModel class")
    }
}
