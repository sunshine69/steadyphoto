package com.steadyphoto.sync.data.local.entity

import androidx.room.Entity
import androidx.room.PrimaryKey

/**
 * Entity storing sync state - specifically the timestamp of the last successful scan.
 * Used for incremental scanning to avoid re-processing already-seen files.
 */
@Entity(tableName = "sync_state")
data class SyncStateEntity(
    @PrimaryKey
    val key: String = "last_sync_timestamp",
    
    /** Timestamp in seconds (MediaStore DATE_ADDED format) of last successful scan */
    val value: String
)
