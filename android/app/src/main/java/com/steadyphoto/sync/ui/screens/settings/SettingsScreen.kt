package com.steadyphoto.sync.ui.screens.settings

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.steadyphoto.sync.data.remote.api.ApiClient

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    onNavigateUp: () -> Unit,
    modifier: Modifier = Modifier
) {
    var autoSyncEnabled by remember { mutableStateOf(true) }
    var wifiOnly by remember { mutableStateOf(true) }
    
    // Load the current API URL from ApiClient
    var apiUrl by remember { mutableStateOf(ApiClient.getBaseUrl()) }
    var isValidUrl by remember { mutableStateOf(true) }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateUp) {
                        Icon(Icons.Default.ArrowBack, contentDescription = "Back")
                    }
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // API URL setting - NEW: Added this section for configuring the API endpoint
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("API Server", style = MaterialTheme.typography.titleMedium)
                    Spacer(modifier = Modifier.height(8.dp))
                    
                    OutlinedTextField(
                        value = apiUrl,
                        onValueChange = {
                            apiUrl = it
                            isValidUrl = it.isNotEmpty() && (it.startsWith("http://") || it.startsWith("https://"))
                        },
                        label = { Text("API URL") },
                        placeholder = { Text("e.g., https://api.steadyphoto.com/") },
                        singleLine = true,
                        isError = !isValidUrl,
                        supportingText = if (!isValidUrl) {
                            { Text("URL must start with http:// or https://") }
                        } else null,
                        modifier = Modifier.fillMaxWidth()
                    )
                    
                    Spacer(modifier = Modifier.height(8.dp))
                    
                    Button(
                        onClick = {
                            if (isValidUrl) {
                                ApiClient.updateBaseUrl(apiUrl.trim())
                                // Note: The app may need to restart for changes to take effect
                                // A toast or snackbar could be shown here to inform the user
                            }
                        },
                        enabled = isValidUrl,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Save")
                    }
                }
            }

            // Auto-sync toggle
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Auto Sync", style = MaterialTheme.typography.titleMedium)
                    Switch(
                        checked = autoSyncEnabled,
                        onCheckedChange = { autoSyncEnabled = it }
                    )
                }
            }

            // WiFi only toggle
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("WiFi Only", style = MaterialTheme.typography.titleMedium)
                    Switch(
                        checked = wifiOnly,
                        onCheckedChange = { wifiOnly = it }
                    )
                }
            }

            // Storage info (placeholder)
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Storage", style = MaterialTheme.typography.titleMedium)
                    Spacer(modifier = Modifier.height(8.dp))
                    Text("Local storage: 0 MB used")
                    Text("Synced items: 0")
                }
            }

            // Account section (placeholder)
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Account", style = MaterialTheme.typography.titleMedium)
                    Spacer(modifier = Modifier.height(8.dp))
                    Button(onClick = { /* TODO: Logout */ }) {
                        Text("Logout")
                    }
                }
            }
        }
    }
}
