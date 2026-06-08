package com.steadyphoto.sync.data.repository

import android.content.Context
import android.net.ConnectivityManager
import android.net.Network
import android.net.NetworkCapabilities
import android.net.NetworkRequest
import kotlinx.coroutines.channels.awaitClose
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.callbackFlow
import com.steadyphoto.sync.data.settings.NetworkSettings

/**
 * Monitors network connectivity status and provides real-time updates.
 * Enhanced to track granular network type information (WiFi, Cellular, etc.)
 * and check if the current network matches user preferences.
 */
class NetworkConnectivityMonitor(private val context: Context) {
    
    private val connectivityManager = 
        context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    
    /**
     * Returns a Flow that emits the current detailed network state.
     * Includes both availability and network type information.
     */
    fun observeDetailedNetworkState(): Flow<DetailedNetworkState> = callbackFlow {
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                val detailedState = getCurrentDetailedState()
                if (detailedState.isSuccess) {
                    trySend(detailedState.getOrThrow()).isSuccess
                } else {
                    trySend(DetailedNetworkState.Unavailable).isSuccess
                }
            }
            
            override fun onLost(network: Network) {
                trySend(DetailedNetworkState.Unavailable).isSuccess
            }
            
            override fun onLosing(network: Network, maxMsToLive: Int) {
                // Network is about to be lost - could emit a "losing" state if needed
            }
            
            override fun onCapabilitiesChanged(
                network: Network,
                networkCapabilities: NetworkCapabilities
            ) {
                val detailedState = getCurrentDetailedState()
                if (detailedState.isSuccess) {
                    trySend(detailedState.getOrThrow()).isSuccess
                } else {
                    trySend(DetailedNetworkState.Unavailable).isSuccess
                }
            }
            
            override fun onUnavailable() {
                trySend(DetailedNetworkState.Unavailable).isSuccess
            }
        }
        
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
            
        connectivityManager.registerNetworkCallback(request, callback)
        
        awaitClose {
            try {
                connectivityManager.unregisterNetworkCallback(callback)
            } catch (e: IllegalArgumentException) {
                // Callback may already be unregistered
            }
        }
    }
    
    /**
     * Returns a Flow that emits the current network availability status.
     * Kept for backward compatibility with existing code.
     */
    fun observeNetworkStatus(): Flow<NetworkAvailability> = callbackFlow {
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                trySend(NetworkAvailability.Available).isSuccess
            }
            
            override fun onLost(network: Network) {
                trySend(NetworkAvailability.Unavailable).isSuccess
            }
            
            override fun onLosing(network: Network, maxMsToLive: Int) {
                // Network is about to be lost
            }
            
            override fun onCapabilitiesChanged(
                network: Network,
                networkCapabilities: NetworkCapabilities
            ) {
                val hasInternet = networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                        networkCapabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
                
                trySend(if (hasInternet) NetworkAvailability.Available else NetworkAvailability.Unavailable).isSuccess
            }
        }
        
        val request = NetworkRequest.Builder()
            .addCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET)
            .build()
            
        connectivityManager.registerNetworkCallback(request, callback)
        
        awaitClose {
            try {
                connectivityManager.unregisterNetworkCallback(callback)
            } catch (e: IllegalArgumentException) {
                // Callback may already be unregistered
            }
        }
    }
    
    /**
     * Checks current network status synchronously.
     */
    fun isCurrentlyConnected(): Boolean {
        val activeNetwork = connectivityManager.activeNetwork ?: return false
        val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork) ?: return false
        
        return capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
    }
    
    /**
     * Gets the type of the currently active network.
     * Returns null if no network is connected or type cannot be determined.
     */
    fun getCurrentNetworkType(): NetworkType? {
        val activeNetwork = connectivityManager.activeNetwork ?: return null
        val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork) ?: return null
        
        return when {
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> NetworkType.WIFI
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> NetworkType.CELLULAR
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> NetworkType.ETHERNET
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_BLUETOOTH) -> NetworkType.BLUETOOTH
            else -> null
        }
    }
    
    /**
     * Checks if the current network meets the user's preferences.
     * Returns true if:
     * - No settings are configured (default behavior)
     * - Network is available AND matches the preferred type
     */
    fun isNetworkAcceptableForUpload(settings: NetworkSettings): Boolean {
        // First check if we have any internet connection at all
        if (!isCurrentlyConnected()) {
            return false
        }
        
        val currentType = getCurrentNetworkType() ?: return false
        
        return when {
            // If WiFi-only is enabled, only accept WiFi or Ethernet (unmetered)
            settings.wifiOnlyEnabled -> {
                currentType == NetworkType.WIFI || currentType == NetworkType.ETHERNET
            }
            
            // If sync on metered is disabled, reject cellular
            !settings.syncOnMeteredConnection && currentType == NetworkType.CELLULAR -> false
            
            else -> true
        }
    }
    
    /**
     * Helper to get the current detailed network state.
     */
    private fun getCurrentDetailedState(): Result<DetailedNetworkState> {
        val activeNetwork = connectivityManager.activeNetwork ?: return Result.failure(Exception("No active network"))
        val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork) ?: return Result.failure(Exception("No network capabilities"))
        
        val hasInternet = capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_INTERNET) &&
                capabilities.hasCapability(NetworkCapabilities.NET_CAPABILITY_VALIDATED)
        
        if (!hasInternet) {
            return Result.success(DetailedNetworkState.Unavailable)
        }
        
        val currentType = when {
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> NetworkType.WIFI
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> NetworkType.CELLULAR
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> NetworkType.ETHERNET
            else -> NetworkType.UNKNOWN
        }
        
        return Result.success(DetailedNetworkState.Available(currentType))
    }
}

/**
 * Represents the type of network connection currently active.
 */
enum class NetworkType {
    WIFI,
    CELLULAR,
    ETHERNET,
    BLUETOOTH,
    UNKNOWN
}

/**
 * Detailed network state including availability and type information.
 */
sealed class DetailedNetworkState {
    data class Available(val networkType: NetworkType) : DetailedNetworkState()
    object Unavailable : DetailedNetworkState()
    
    val isConnected: Boolean = this is Available
    
    /**
     * Checks if the current network meets basic upload requirements (any connection).
     */
    fun canUpload(): Boolean = isConnected
}

/**
 * Represents the current network availability status.
 * Kept for backward compatibility with existing code.
 */
sealed class NetworkAvailability {
    object Available : NetworkAvailability()
    object Unavailable : NetworkAvailability()
}
