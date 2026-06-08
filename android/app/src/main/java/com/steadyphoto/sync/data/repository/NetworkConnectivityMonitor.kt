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
 */
class NetworkConnectivityMonitor(private val context: Context) {
    
    private val connectivityManager = 
        context.getSystemService(Context.CONNECTIVITY_SERVICE) as ConnectivityManager
    
    /**
     * Returns a Flow that emits the current detailed network state.
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
            
            override fun onLosing(network: Network, maxMsToLive: Int) {}
            
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
            } catch (e: IllegalArgumentException) {}
        }
    }
    
    /**
     * Returns a Flow that emits the current network availability status.
     */
    fun observeNetworkStatus(): Flow<NetworkAvailability> = callbackFlow {
        val callback = object : ConnectivityManager.NetworkCallback() {
            override fun onAvailable(network: Network) {
                trySend(NetworkAvailability.Available).isSuccess
            }
            
            override fun onLost(network: Network) {
                trySend(NetworkAvailability.Unavailable).isSuccess
            }
            
            override fun onLosing(network: Network, maxMsToLive: Int) {}
            
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
            } catch (e: IllegalArgumentException) {}
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
     */
    fun getCurrentNetworkType(): NetworkType? {
        val activeNetwork = connectivityManager.activeNetwork ?: return null
        val capabilities = connectivityManager.getNetworkCapabilities(activeNetwork) ?: return null
        
        return when {
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_WIFI) -> NetworkType.WIFI
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_CELLULAR) -> NetworkType.CELLULAR
            capabilities.hasTransport(NetworkCapabilities.TRANSPORT_ETHERNET) -> NetworkType.ETHERNET
            else -> null
        }
    }
    
    /**
     * Checks if the current network meets the user's preferences.
     * Returns true if:
     * - No internet connection at all = false (can't upload without any connection)
     * - WiFi-only enabled + on WiFi/Ethernet = true
     * - WiFi-only disabled + on any network = true
     */
    fun isNetworkAcceptableForUpload(settings: NetworkSettings): Boolean {
        // First check if we have any internet connection at all
        if (!isCurrentlyConnected()) {
            return false
        }
        
        val currentType = getCurrentNetworkType() ?: return false
        
        // If WiFi-only is enabled, only accept WiFi or Ethernet (unmetered networks)
        // If WiFi-only is disabled, accept any network including cellular
        return when {
            settings.wifiOnlyEnabled -> {
                currentType == NetworkType.WIFI || currentType == NetworkType.ETHERNET
            }
            
            else -> true  // Any network is acceptable when wifiOnly is disabled
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
    UNKNOWN
}

/**
 * Detailed network state including availability and type information.
 */
sealed class DetailedNetworkState {
    data class Available(val networkType: NetworkType) : DetailedNetworkState()
    object Unavailable : DetailedNetworkState()
    
    val isConnected: Boolean = this is Available
    
    fun canUpload(): Boolean = isConnected
}

/**
 * Represents the current network availability status.
 */
sealed class NetworkAvailability {
    object Available : NetworkAvailability()
    object Unavailable : NetworkAvailability()
}
