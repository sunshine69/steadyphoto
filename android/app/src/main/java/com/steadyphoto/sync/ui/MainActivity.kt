package com.steadyphoto.sync.ui

import android.os.Bundle
import androidx.activity.ComponentActivity
import androidx.activity.compose.setContent
import androidx.compose.foundation.layout.fillMaxSize
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.Surface
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import com.steadyphoto.sync.data.remote.api.ApiClient
import com.steadyphoto.sync.data.remote.api.onAuthFailure
import com.steadyphoto.sync.ui.screens.auth.LoginScreen
import com.steadyphoto.sync.ui.screens.main.HomeScreen
import com.steadyphoto.sync.ui.screens.settings.SettingsScreen
import com.steadyphoto.sync.ui.screens.setup.SetupScreen
import com.steadyphoto.sync.ui.theme.SteadyPhotoTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        // Initialize ApiClient with context for SharedPreferences access
        ApiClient.init(this)
        
        // Log the current API endpoint for debugging purposes
        ApiClient.logApiEndpoint()
        
        // Set up auth failure callback - when refresh fails or no token exists,
        // this will be called to show login screen and clear auth state.
        onAuthFailure = {
            android.util.Log.d("MainActivity", "Auth failure - navigating to login")
            // Clear both access and refresh tokens
            ApiClient.clearAuthTokenAndRefresh()
            // The callback runs on an OkHttp interceptor thread (not main UI thread),
            // so we need to post the state update back to the main thread.
            // We'll use a LaunchedEffect that watches isLoggedIn to trigger navigation.
        }
        
        setContent {
            SteadyPhotoTheme {
                Surface(
                    modifier = Modifier.fillMaxSize(),
                    color = MaterialTheme.colorScheme.background
                ) {
                    var isLoggedIn by remember { mutableStateOf(false) }

                    // Track whether we should show the setup screen.
                    // Default to true so first-time users always see it.
                    var showSetupScreen by remember { mutableStateOf(true) }

                    // Navigation state: "home" or "settings"
                    var currentRoute by remember { mutableStateOf("home") }

                    LaunchedEffect(Unit) {
                        val isSetupDone = ApiClient.isSetupComplete()
                        showSetupScreen = !isSetupDone
                        
                        // Check if user was previously logged in and restore session
                        isLoggedIn = ApiClient.isLoggedIn()
                        
                        // If setup wasn't done yet, always show Setup first (regardless of login status)
                        // because API URL needs to be configured before login
                        if (!isSetupDone) {
                            isLoggedIn = false
                            showSetupScreen = true
                        } else if (isLoggedIn) {
                            // User is logged in and setup is done - go directly to Home
                            showSetupScreen = false
                        }
                    }

                    // When auth failure callback clears tokens, this will detect it and switch to login screen.
                    LaunchedEffect(isLoggedIn) {
                        if (!isLoggedIn && !showSetupScreen) {
                            android.util.Log.d("MainActivity", "Auth state changed - showing LoginScreen")
                        }
                    }

                    when {
                        showSetupScreen -> {
                            // Show Setup screen - this is the first time the app is being used or API URL not configured
                            SetupScreen(
                                onSetupComplete = { 
                                    // After setup, update state to show LoginScreen/HomeScreen based on login status
                                    showSetupScreen = false
                                },
                                modifier = Modifier.fillMaxSize()
                            )
                        }
                        isLoggedIn -> {
                            when (currentRoute) {
                                "settings" -> SettingsScreen(
                                    onNavigateUp = { currentRoute = "home" },
                                    modifier = Modifier.fillMaxSize()
                                )
                                else -> HomeScreen(
                                    onNavigateToSettings = { currentRoute = "settings" },
                                    onLogout = { 
                                        ApiClient.clearAuthTokenAndRefresh()
                                        isLoggedIn = false
                                    },
                                    modifier = Modifier.fillMaxSize()
                                )
                            }
                        }
                        else -> {
                            LoginScreen(
                                onLoginSuccess = { isLoggedIn = true },
                                modifier = Modifier.fillMaxSize()
                            )
                        }
                    }
                }
            }
        }
    }
}
