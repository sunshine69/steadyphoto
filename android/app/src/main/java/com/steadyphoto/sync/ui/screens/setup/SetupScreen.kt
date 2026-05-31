package com.steadyphoto.sync.ui.screens.setup

import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.ArrowBack
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import com.steadyphoto.sync.data.remote.api.ApiClient

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun SetupScreen(
    onSetupComplete: () -> Unit,
    modifier: Modifier = Modifier
) {
    var apiUrl by remember { mutableStateOf(ApiClient.getBaseUrl()) }
    var isValidUrl by remember { mutableStateOf(true) }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("Setup") },
                navigationIcon = {
                    // No back button - this is the first screen
                }
            )
        }
    ) { padding ->
        Column(
            modifier = Modifier
                .fillMaxSize()
                .padding(padding)
                .padding(16.dp),
            verticalArrangement = Arrangement.Center,
            horizontalAlignment = Alignment.CenterHorizontally
        ) {
            Text(
                text = "SteadyPhoto Sync",
                style = MaterialTheme.typography.headlineMedium,
                modifier = Modifier.padding(bottom = 8.dp)
            )

            Text(
                text = "Enter your API server URL to get started",
                style = MaterialTheme.typography.bodyLarge,
                color = MaterialTheme.colorScheme.onSurfaceVariant,
                modifier = Modifier.padding(bottom = 32.dp)
            )

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

            Spacer(modifier = Modifier.height(24.dp))

            Button(
                onClick = {
                    if (isValidUrl) {
                        ApiClient.updateBaseUrl(apiUrl.trim())
                        onSetupComplete()
                    }
                },
                enabled = isValidUrl,
                modifier = Modifier.fillMaxWidth()
            ) {
                Text("Continue")
            }

            Spacer(modifier = Modifier.height(16.dp))

            // Example URLs for convenience
            Column(
                modifier = Modifier.fillMaxWidth(),
                verticalArrangement = Arrangement.spacedBy(8.dp)
            ) {
                Text(
                    text = "Example URLs:",
                    style = MaterialTheme.typography.bodySmall,
                    color = MaterialTheme.colorScheme.onSurfaceVariant
                )
                
                OutlinedButton(
                    onClick = { apiUrl = "http://10.0.2.2:8080/" },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("Local (Android Emulator)")
                }

                OutlinedButton(
                    onClick = { apiUrl = "https://api.steadyphoto.com/" },
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Text("Production API")
                }
            }
        }
    }
}
