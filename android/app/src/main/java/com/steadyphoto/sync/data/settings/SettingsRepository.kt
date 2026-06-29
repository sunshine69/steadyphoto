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
 */
class SettingsRepository(private val context: Context) {

    companion object {
        // Network-related keys - simplified to just two options
        private val WIFI_ONLY_KEY = booleanPreferencesKey("wifi_only_enabled")
        
        // Sync control key
        private val AUTO_SYNC_ENABLED_KEY = booleanPreferencesKey("auto_sync_enabled")
        private val FALLBACK_SYNC_INTERVAL_KEY = intPreferencesKey("fallback_sync_interval_minutes")
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
                wifiOnlyEnabled = preferences[WIFI_ONLY_KEY] ?: true, // Default to WiFi-only for battery savings
                syncOnMeteredConnection = false  // Not persisted - derived from wifiOnlyEnabled
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
                autoSyncEnabled = preferences[AUTO_SYNC_ENABLED_KEY] ?: true,
                fallbackSyncIntervalMinutes = preferences[FALLBACK_SYNC_INTERVAL_KEY] ?: 30
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
     * Set the preferred network type for sync operations.
     * No-op since we use wifiOnlyEnabled boolean instead of enum.
     */
    suspend fun setPreferredNetworkType(networkType: PreferredNetworkType) {
        // This is a no-op since we use wifiOnlyEnabled boolean instead
        val enabled = when (networkType) {
            PreferredNetworkType.WIFI_ONLY -> true
            PreferredNetworkType.ANY_NETWORK -> false
        }
        context.settingsDataStore.edit { preferences ->
            preferences[WIFI_ONLY_KEY] = enabled
        }
    }

    /**
     * Set whether to allow syncing on metered (cellular) connections.
     * No-op since we use wifiOnlyEnabled boolean instead.
     */
    suspend fun setSyncOnMeteredConnection(enabled: Boolean) {
        // This is a no-op since we use wifiOnlyEnabled boolean instead
        val enabledWifiOnly = !enabled
        context.settingsDataStore.edit { preferences ->
            preferences[WIFI_ONLY_KEY] = enabledWifiOnly
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

    /**
     * Set the fallback sync interval in minutes.
     */
    suspend fun setFallbackSyncInterval(minutes: Int) {
        context.settingsDataStore.edit { preferences ->
            preferences[FALLBACK_SYNC_INTERVAL_KEY] = minutes
        }
    }
}

/**
 * Represents network-related settings for the sync client.
 * Simplified to just two options: WiFi only or any network.
 */
data class NetworkSettings(
    val wifiOnlyEnabled: Boolean,
    /** Whether to allow syncing on metered (cellular) connections. Derived from wifiOnlyEnabled. */
    val syncOnMeteredConnection: Boolean = false
) {
    companion object {
        /** Default network settings - WiFi-only by default for battery savings. */
        fun default(): NetworkSettings = NetworkSettings(
            wifiOnlyEnabled = true,
            syncOnMeteredConnection = false
        )
    }
}

/**
 * Represents the type of network preferred for sync operations.
 */
enum class PreferredNetworkType(val value: String) {
    WIFI_ONLY("wifi_only"),
    ANY_NETWORK("any_network");

    companion object {
        fun fromValue(value: String?): PreferredNetworkType {
            return values().find { it.value == value } ?: WIFI_ONLY
        }
    }
}

/**
 * Represents sync control settings.
 */
data class SyncControlSettings(
    val autoSyncEnabled: Boolean,
    val fallbackSyncIntervalMinutes: Int
) {
    companion object {
        fun default(): SyncControlSettings = SyncControlSettings(
            autoSyncEnabled = true,
            fallbackSyncIntervalMinutes = 30
        )
    }
}
