package com.steadyphoto.sync.data.local.dao

import androidx.room.*
import com.steadyphoto.sync.data.local.entity.SyncStateEntity

@Dao
interface SyncStateDao {

    @Insert(onConflict = OnConflictStrategy.REPLACE)
    suspend fun insertOrUpdate(syncState: SyncStateEntity)

    @Query("SELECT * FROM sync_state WHERE key = 'last_sync_timestamp' LIMIT 1")
    suspend fun getLastSyncTimestamp(): SyncStateEntity?

    @Query("DELETE FROM sync_state WHERE key = 'last_sync_timestamp'")
    suspend fun clearLastSyncTimestamp()
}
