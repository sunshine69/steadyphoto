package com.steadyphoto.sync.ui.screens.auth

import android.util.Log
import androidx.lifecycle.ViewModel
import androidx.lifecycle.viewModelScope
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.data.remote.dto.LoginRequest
import kotlinx.coroutines.flow.MutableStateFlow
import kotlinx.coroutines.flow.StateFlow
import kotlinx.coroutines.launch

sealed class AuthUiState {
    object Idle : AuthUiState()
    object Loading : AuthUiState()
    data class Success(val token: String) : AuthUiState()
    data class Error(val message: String) : AuthUiState()
}

class AuthViewModel : ViewModel() {

    private val _uiState = MutableStateFlow<AuthUiState>(AuthUiState.Idle)
    val uiState: StateFlow<AuthUiState> = _uiState

    fun login(email: String, password: String) {
        viewModelScope.launch {
            _uiState.value = AuthUiState.Loading
            try {
                // Always get the latest ApiService from ApiClient (handles URL changes)
                val apiService = ApiClient.apiService
                
                // Pass LoginRequest body - JSON format expected by Go backend
                val response = apiService.login(
                    authHeader = null,
                    body = LoginRequest(
                        email = email.trim().lowercase(),
                        password = password
                    )
                )
                
                // Store access token for future requests (Go returns "access_token")
                ApiClient.storeAuthToken(response.accessToken)
                // Also store refresh token so the client can auto-refresh when access token expires
                ApiClient.storeRefreshToken(token = response.refreshToken)
                
                _uiState.value = AuthUiState.Success(response.accessToken)
            } catch (e: Exception) {
                Log.e("AuthViewModel", "Login failed", e)
                _uiState.value = AuthUiState.Error(e.message ?: "Login failed")
            }
        }
    }

    fun resetState() {
        _uiState.value = AuthUiState.Idle
    }
}
