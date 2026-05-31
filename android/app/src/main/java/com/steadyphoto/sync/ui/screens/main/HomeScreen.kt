package com.steadyphoto.sync.ui.screens.main

import androidx.compose.foundation.layout.*
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudUpload
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.Settings
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.unit.dp
import org.koin.androidx.compose.koinViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onNavigateToSettings: () -> Unit,
    modifier: Modifier = Modifier,
    viewModel: MainViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("SteadyPhoto Sync") },
                actions = {
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(Icons.Default.Settings, contentDescription = "Settings")
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
            horizontalAlignment = Alignment.CenterHorizontally,
            verticalArrangement = Arrangement.spacedBy(16.dp)
        ) {
            // Stats cards
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly
            ) {
                Card(modifier = Modifier.weight(1f)) {
                    Column(
                        modifier = Modifier.padding(16.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(text = "${uiState.pendingCount}", style = MaterialTheme.typography.headlineMedium)
                        Text(text = "Pending")
                    }
                }
                
                Spacer(modifier = Modifier.width(8.dp))
                
                Card(modifier = Modifier.weight(1f)) {
                    Column(
                        modifier = Modifier.padding(16.dp),
                        horizontalAlignment = Alignment.CenterHorizontally
                    ) {
                        Text(text = "${uiState.uploadedCount}", style = MaterialTheme.typography.headlineMedium)
                        Text(text = "Uploaded")
                    }
                }
            }

            // Sync button
            Button(
                onClick = { viewModel.startSync() },
                enabled = uiState.syncState !is SyncUiState.Uploading && 
                         uiState.syncState !is SyncUiState.Scanning,
                modifier = Modifier.fillMaxWidth()
            ) {
                Icon(Icons.Default.Refresh, contentDescription = null)
                Spacer(modifier = Modifier.width(8.dp))
                Text("Sync Now")
            }

            // Current status
            when (val state = uiState.syncState) {
                is SyncUiState.Scanning -> {
                    CircularProgressIndicator()
                    Text("Scanning media...")
                }
                is SyncUiState.Uploading -> {
                    CircularProgressIndicator()
                    Text("Uploading...")
                }
                is SyncUiState.Success -> {
                    Text("Successfully synced ${state.count} items!")
                }
                is SyncUiState.Error -> {
                    Text(state.message, color = MaterialTheme.colorScheme.error)
                    Button(onClick = viewModel::clearError) {
                        Text("Dismiss")
                    }
                }
                else -> {}
            }

            // Recent items list (placeholder)
            if (uiState.recentItems.isNotEmpty()) {
                Text("Recent Media", style = MaterialTheme.typography.titleMedium)
                LazyColumn(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(8.dp)
                ) {
                    items(uiState.recentItems.size) { index ->
                        val item = uiState.recentItems[index]
                        Card(modifier = Modifier.fillMaxWidth()) {
                            Text(
                                text = "${item.fileName} - ${item.uploadStatus}",
                                modifier = Modifier.padding(16.dp)
                            )
                        }
                    }
                }
            }
        }
    }
}
