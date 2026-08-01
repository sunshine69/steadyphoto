package com.steadyphoto.sync.ui.screens.main

import androidx.activity.compose.rememberLauncherForActivityResult
import androidx.activity.result.contract.ActivityResultContracts
import androidx.compose.foundation.layout.*
import androidx.compose.material.icons.Icons
import androidx.compose.material.icons.filled.CloudUpload
import androidx.compose.material.icons.filled.MoreVert
import androidx.compose.material.icons.automirrored.filled.Logout
import androidx.compose.material.icons.filled.Refresh
import androidx.compose.material.icons.filled.StopCircle
import androidx.compose.foundation.lazy.LazyColumn
import androidx.compose.foundation.lazy.items
import androidx.compose.material3.*
import androidx.compose.runtime.*
import androidx.activity.ComponentActivity
import androidx.compose.ui.Alignment
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import org.koin.androidx.compose.koinViewModel

@OptIn(ExperimentalMaterial3Api::class)
@Composable
fun HomeScreen(
    onNavigateToSettings: () -> Unit,
    onLogout: () -> Unit = {},
    modifier: Modifier = Modifier,
    viewModel: MainViewModel = koinViewModel()
) {
    val uiState by viewModel.uiState.collectAsState()
    val context = LocalContext.current as ComponentActivity

    // Create an ActivityResultLauncher for requesting permissions
    val permissionLauncher = rememberLauncherForActivityResult(
        contract = ActivityResultContracts.RequestMultiplePermissions(),
        onResult = { permissions ->
            // Check if all requested permissions are granted
            val allGranted = permissions.values.all { it }
            
            if (allGranted) {
                // Permissions granted - clear error state and allow sync to proceed
                viewModel.handlePermissionResult(isGranted = true, permanentlyDenied = false)
            } else {
                // Some permissions denied - check if any were permanently denied
                val permanentlyDenied = permissions.entries.any { (permissionName, isGranted) ->
                    !isGranted && context.shouldShowRequestPermissionRationale(permissionName)
                }
                
                viewModel.handlePermissionResult(isGranted = false, permanentlyDenied = permanentlyDenied)
            }
        }
    )

    // Function to request permissions - this will trigger the launcher
    fun requestPermissions() {
        val permissionsToRequest = context.let { ctx ->
            com.steadyphoto.sync.util.PermissionHelper.getMediaPermissionsToRequest(ctx)
        }
        
        permissionLauncher.launch(permissionsToRequest)
    }

    Scaffold(
        modifier = modifier.fillMaxSize(),
        topBar = {
            TopAppBar(
                title = { Text("SteadyPhoto Sync") },
                actions = {
                    // Logout button
                    IconButton(onClick = onLogout) {
                        Icon(Icons.AutoMirrored.Filled.Logout, contentDescription = "Logout")
                    }
                    IconButton(onClick = onNavigateToSettings) {
                        Icon(Icons.Default.MoreVert, contentDescription = "Settings")
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
            verticalArrangement = Arrangement.spacedBy(24.dp)
        ) {
            // Stats cards - with try-catch to prevent crashes
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

            // Status indicator showing current state
            Surface(
                modifier = Modifier.fillMaxWidth(),
                color = when (val state = uiState.syncState) {
                    is SyncUiState.Idle -> MaterialTheme.colorScheme.surfaceVariant
                    is SyncUiState.Scanning -> MaterialTheme.colorScheme.primaryContainer
                    is SyncUiState.Uploading -> MaterialTheme.colorScheme.tertiaryContainer
                    is SyncUiState.Success -> MaterialTheme.colorScheme.secondaryContainer
                    is SyncUiState.Error -> MaterialTheme.colorScheme.errorContainer
                },
                shape = MaterialTheme.shapes.medium
            ) {
                Row(
                    modifier = Modifier.padding(12.dp),
                    horizontalArrangement = Arrangement.spacedBy(8.dp),
                    verticalAlignment = Alignment.CenterVertically
                ) {
                    when (val state = uiState.syncState) {
                        is SyncUiState.Idle -> Text("Ready - Tap Start to begin")
                        is SyncUiState.Scanning -> Text("Scanning media...")
                        is SyncUiState.Uploading -> {
                            Column {
                                Text("Uploading...")
                                if (uiState.currentUploadItem != null) {
                                    Column(modifier = Modifier.padding(top = 8.dp)) {
                                        Text(
                                            text = uiState.currentUploadItem!!,
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onTertiaryContainer,
                                            maxLines = 1
                                        )
                                        LinearProgressIndicator(
                                            progress = uiState.currentUploadProgress / 100f,
                                            modifier = Modifier.fillMaxWidth().padding(top = 4.dp)
                                        )
                                        Text(
                                            text = "${String.format("%.0f", uiState.currentUploadProgress)}%",
                                            style = MaterialTheme.typography.bodySmall,
                                            color = MaterialTheme.colorScheme.onTertiaryContainer,
                                            modifier = Modifier.padding(top = 2.dp)
                                        )
                                    }
                                }
                            }
                        }
                        is SyncUiState.Success -> Text("Successfully synced ${state.count} items!")
                        is SyncUiState.Error -> {
                            Column {
                                Text(state.message, color = MaterialTheme.colorScheme.onErrorContainer)
                                Button(onClick = viewModel::clearError, modifier = Modifier.padding(top = 8.dp)) {
                                    Text("Dismiss")
                                }
                            }
                        }
                    }
                }
            }

            // Control buttons - Start and Stop for background sync
            Row(
                modifier = Modifier.fillMaxWidth(),
                horizontalArrangement = Arrangement.SpaceEvenly,
                verticalAlignment = Alignment.CenterVertically
            ) {
                Button(
                    onClick = { 
                        viewModel.startBackgroundSync()
                        
                        // Send START intent to the foreground service so it starts its real-time detection loop
                        val syncIntent = com.steadyphoto.sync.worker.SyncService.newIntent(context).apply {
                            action = com.steadyphoto.sync.worker.SyncService.ACTION_START_SYNC
                        }
                        try {
                            context.startForegroundService(syncIntent)
                        } catch (e: Exception) {
                            android.util.Log.w("HomeScreen", "Failed to start SyncService", e)
                        }
                    },
                    // Enabled if NOT currently syncing (scanning or uploading) and no error present
                    enabled = uiState.syncState is SyncUiState.Idle && 
                             uiState.errorMessage == null,
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(Icons.Default.CloudUpload, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Start")
                }

                Button(
                    onClick = { 
                        viewModel.stopBackgroundSync()
                        
                        // Send STOP intent to the foreground service so it stops its real-time detection loop and destroys itself
                        val syncIntent = com.steadyphoto.sync.worker.SyncService.newIntent(context).apply {
                            action = com.steadyphoto.sync.worker.SyncService.ACTION_STOP_SYNC
                        }
                        try {
                            context.startForegroundService(syncIntent)
                        } catch (e: Exception) {
                            android.util.Log.w("HomeScreen", "Failed to stop SyncService", e)
                        }
                    },
                    // Enabled if currently syncing or uploading OR background sync is running
                    enabled = uiState.syncState is SyncUiState.Uploading || 
                             uiState.syncState is SyncUiState.Scanning ||
                             uiState.isBackgroundSyncRunning,
                    colors = ButtonDefaults.buttonColors(
                        containerColor = MaterialTheme.colorScheme.errorContainer,
                        contentColor = MaterialTheme.colorScheme.onErrorContainer
                    ),
                    modifier = Modifier.weight(1f)
                ) {
                    Icon(Icons.Default.StopCircle, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Stop")
                }
            }

            // Quick sync button (one-time upload) - shown when idle and no errors
            if (uiState.syncState is SyncUiState.Idle && uiState.errorMessage == null) {
                OutlinedButton(
                    onClick = { viewModel.startSync() },
                    modifier = Modifier.fillMaxWidth(),
                    colors = ButtonDefaults.outlinedButtonColors(
                        contentColor = MaterialTheme.colorScheme.primary
                    )
                ) {
                    Icon(Icons.Default.Refresh, contentDescription = null)
                    Spacer(modifier = Modifier.width(8.dp))
                    Text("Sync Now")
                }
            }

            // Permission error message if shown
            if (uiState.showPermissionRationale) {
                Card(
                    colors = CardDefaults.cardColors(
                        containerColor = MaterialTheme.colorScheme.errorContainer
                    ),
                    modifier = Modifier.fillMaxWidth()
                ) {
                    Column(modifier = Modifier.padding(12.dp)) {
                        Text("⚠️ Permissions Required", style = MaterialTheme.typography.titleMedium)
                        Spacer(modifier = Modifier.height(4.dp))
                        Text(
                            "Please grant storage permissions to scan your photos and videos.",
                            style = MaterialTheme.typography.bodySmall,
                            color = MaterialTheme.colorScheme.onErrorContainer
                        )
                        Button(onClick = { requestPermissions() }) {
                            Text("Grant Permissions")
                        }
                    }
                }
            }

            // Recent items list (placeholder) - with try-catch to prevent crashes
            if (uiState.recentItems.isNotEmpty()) {
                Text("Recent Media", style = MaterialTheme.typography.titleMedium)
                LazyColumn(
                    modifier = Modifier.fillMaxWidth(),
                    verticalArrangement = Arrangement.spacedBy(8.dp),
                    contentPadding = PaddingValues(vertical = 8.dp)
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

            // Spacer at bottom for scrolling room
            Spacer(modifier = Modifier.height(8.dp))
        }
    }
}
