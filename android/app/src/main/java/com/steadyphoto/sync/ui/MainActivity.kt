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
import com.steadyphoto.sync.ui.screens.auth.LoginScreen
import com.steadyphoto.sync.ui.screens.main.HomeScreen
import com.steadyphoto.sync.ui.screens.setup.SetupScreen
import com.steadyphoto.sync.ui.theme.SteadyPhotoTheme

class MainActivity : ComponentActivity() {
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        
        // Initialize ApiClient with context for SharedPreferences access
        ApiClient.init(this)
        
        // Log the current API endpoint for debugging purposes
        ApiClient.logApiEndpoint()
        
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
                            HomeScreen(
                                onNavigateToSettings = { /* TODO: Navigate to settings */ },
                                onLogout = { 
                                    ApiClient.clearAuthToken()
                                    isLoggedIn = false
                                },
                                modifier = Modifier.fillMaxSize()
                            )
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
