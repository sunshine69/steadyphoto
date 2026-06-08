package com.steadyphoto.sync.data.settings

import android.content.Context
import androidx.datastore.core.DataStore
import androidx.datastore.preferences.core.*
import androidx.datastore.preferences.preferencesDataStore
import kotlinx.coroutines.flow.Flow
import kotlinx.coroutines.flow.catch
import kotlinx.coroutines.flow.map
import java.io.IOException

// Extension property to create the DataStore instance
private val Context.settingsDataStore: DataStore<androidx.datastore.preferences.core.Preferences> by preferencesDataStore(name = "app_settings")

/**
 * Repository for managing application settings using AndroidX DataStore Preferences.
 * Provides type-safe access to persisted preferences with Flow-based reactive updates.
 */
class SettingsRepository(private val context: Context) {

    companion object {
        // Network-related keys
        private val WIFI_ONLY_KEY = booleanPreferencesKey("wifi_only_enabled")
        private val PREFERRED_NETWORK_TYPE_KEY = stringPreferencesKey("preferred_network_type")
        private val SYNC_ON_METERED_KEY = booleanPreferencesKey("sync_on_metered_connection")
        
        // Sync control keys
        private val AUTO_SYNC_ENABLED_KEY = booleanPreferencesKey("auto_sync_enabled")
    }

    /**
     * Flow of the current network settings.
     */
    val networkSettingsFlow: Flow<NetworkSettings> = context.settingsDataStore.data
        .catch { exception ->
            // Handle errors gracefully - return default values if DataStore has issues
            if (exception is IOException) {
                androidx.datastore.preferences.core.emptyPreferences()
            } else {
                throw exception
            }
        }
        .map { preferences ->
            NetworkSettings(
                wifiOnlyEnabled = preferences[WIFI_ONLY_KEY] ?: true,
                preferredNetworkType = when (preferences[PREFERRED_NETWORK_TYPE_KEY]) {
                    "wifi_only" -> PreferredNetworkType.WIFI_ONLY
                    "no_cellular" -> PreferredNetworkType.NO_CELLULAR
                    else -> PreferredNetworkType.ANY_NETWORK // Default
                },
                syncOnMeteredConnection = preferences[SYNC_ON_METERED_KEY] ?: false
            )
        }

    /**
     * Flow of the current sync control settings.
     */
    val syncControlFlow: Flow<SyncControlSettings> = context.settingsDataStore.data
        .catch { exception ->
            if (exception is IOException) {
                androidx.datastore.preferences.core.emptyPreferences()
            } else {
                throw exception
            }
        }
        .map { preferences ->
            SyncControlSettings(
                autoSyncEnabled = preferences[AUTO_SYNC_ENABLED_KEY] ?: true
            )
        }

    /**
     * Set whether uploads should only happen on WiFi.
     */
    suspend fun setWifiOnly(enabled: Boolean) {
        context.settingsDataStore.edit { preferences ->
            preferences[WIFI_ONLY_KEY] = enabled
        }
    }

    /**
     * Set the preferred network type for uploads.
     */
    suspend fun setPreferredNetworkType(type: PreferredNetworkType) {
        context.settingsDataStore.edit { preferences ->
            preferences[PREFERRED_NETWORK_TYPE_KEY] = type.name.lowercase()
        }
    }

    /**
     * Set whether to sync on metered (cellular) connections.
     */
    suspend fun setSyncOnMeteredConnection(enabled: Boolean) {
        context.settingsDataStore.edit { preferences ->
            preferences[SYNC_ON_METERED_KEY] = enabled
        }
    }

    /**
     * Set whether auto-sync is enabled.
     */
    suspend fun setAutoSyncEnabled(enabled: Boolean) {
        context.settingsDataStore.edit { preferences ->
            preferences[AUTO_SYNC_ENABLED_KEY] = enabled
        }
    }
}

/**
 * Represents network-related settings for the sync client.
 */
data class NetworkSettings(
    val wifiOnlyEnabled: Boolean,
    val preferredNetworkType: PreferredNetworkType,
    val syncOnMeteredConnection: Boolean
) {
    companion object {
        /** Default network settings with sensible defaults. */
        fun default(): NetworkSettings = NetworkSettings(
            wifiOnlyEnabled = true, // Default to WiFi-only for battery savings
            preferredNetworkType = PreferredNetworkType.WIFI_ONLY,
            syncOnMeteredConnection = false // Don't use cellular by default
        )
    }
}

/**
 * Represents the user's choice of network type preference.
 */
enum class PreferredNetworkType {
    /** Only upload when on WiFi (unmetered) connection */
    WIFI_ONLY,
    
    /** Upload on any available network including cellular */
    ANY_NETWORK,
    
    /** Upload on WiFi and other non-cellular networks, but not metered connections */
    NO_CELLULAR
}

/**
 * Represents sync control settings.
 */
data class SyncControlSettings(
    val autoSyncEnabled: Boolean
) {
    companion object {
        fun default(): SyncControlSettings = SyncControlSettings(autoSyncEnabled = true)
    }
}
