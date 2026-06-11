package com.steadyphoto.sync.ui.screens.settings

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.automirrored.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import org.koin.androidx.compose.koinViewModel
import androidx.compose.ui.unit.dp
import com.steadyphoto.sync.data.remote.api.ApiClient

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SettingsScreen(
    onNavigateUp: () -> Unit,
    modifier: Modifier = Modifier
) {
    val viewModel: SettingsViewModel = koinViewModel()
    
    // Collect settings from ViewModel - these will persist across screen recreations
    var wifiOnlyEnabled by remember { mutableStateOf(true) }
    var autoSyncEnabled by remember { mutableStateOf(true) }
    var apiUrl by remember { mutableStateOf(ApiClient.getBaseUrl()) }
    var isValidUrl by remember { mutableStateOf(true) }

    // Collect network settings flow - this triggers emission and updates state
    LaunchedEffect(Unit) {
        viewModel.networkSettingsFlow.collect { settings ->
            wifiOnlyEnabled = settings.wifiOnlyEnabled
        }
    }

    // Collect sync control flow separately  
    LaunchedEffect(viewModel.syncControlFlow) {
        viewModel.syncControlFlow.collect { enabled ->
            autoSyncEnabled = enabled
        }
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("Settings") },
                navigationIcon = {
                    IconButton(onClick = onNavigateUp) {
                       Icon(Icons.AutoMirrored.Filled.ArrowBack, contentDescription = "Back")
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
            // API URL setting
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
                                viewModel.updateApiUrl(apiUrl.trim())
                            }
                        },
                        enabled = isValidUrl,
                        modifier = Modifier.fillMaxWidth()
                    ) {
                        Text("Save")
                    }
                }
            }

            // Network Settings Section - SIMPLIFIED: WiFi Only toggle
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Network", style = MaterialTheme.typography.titleMedium)
                    Spacer(modifier = Modifier.height(8.dp))

                    // Simple two-option network setting: WiFi only vs Any Network
                    Row(
                        modifier = Modifier.fillMaxWidth(),
                        horizontalArrangement = Arrangement.SpaceBetween,
                        verticalAlignment = Alignment.CenterVertically
                    ) {
                        Column(modifier = Modifier.weight(1f)) {
                            Text("Upload on", style = MaterialTheme.typography.bodyLarge)
                            Text(if (wifiOnlyEnabled) "WiFi only" else "Any network", 
                                 style = MaterialTheme.typography.bodyMedium,
                                 color = if (wifiOnlyEnabled) MaterialTheme.colorScheme.primary else MaterialTheme.colorScheme.onSurfaceVariant)
                        }
                        
                        Switch(
                            checked = wifiOnlyEnabled,
                            onCheckedChange = { enabled ->
                                viewModel.setWifiOnly(enabled)
                            }
                        )
                    }
                }
            }

            // Auto-sync toggle - persisted to DataStore
            Card(modifier = Modifier.fillMaxWidth()) {
                Column(modifier = Modifier.padding(16.dp)) {
                    Text("Sync Control", style = MaterialTheme.typography.titleMedium)
                    Spacer(modifier = Modifier.height(8.dp))

                    SwitchPreferenceRow(
                        title = "Auto-sync",
                        description = "Automatically scan and upload new media in the background",
                        checked = autoSyncEnabled,
                        onCheckedChange = { enabled ->
                            viewModel.setAutoSyncEnabled(enabled)
                        }
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

/**
 * A reusable preference row component with a title, description, and switch.
 */
@Composable
private fun SwitchPreferenceRow(
    title: String,
    description: String,
    checked: Boolean,
    onCheckedChange: (Boolean) -> Unit
) {
    Row(
        modifier = Modifier.fillMaxWidth(),
        horizontalArrangement = Arrangement.SpaceBetween,
        verticalAlignment = Alignment.CenterVertically
    ) {
        Column(modifier = Modifier.weight(1f)) {
            Text(title, style = MaterialTheme.typography.bodyLarge)
            Text(description, style = MaterialTheme.typography.bodySmall)
        }
        
        Switch(
            checked = checked,
            onCheckedChange = onCheckedChange
        )
    }
}


