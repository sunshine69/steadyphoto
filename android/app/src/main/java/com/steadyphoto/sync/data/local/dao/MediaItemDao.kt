package com.steadyphoto.sync.data.local.dao

import androidx.room.*
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.data.local.entity.UploadStatus
import kotlinx.coroutines.flow.Flow

@Dao
interface MediaItemDao {

    @Query("SELECT * FROM media_items WHERE uploadStatus = :status ORDER BY createdAt ASC")
    fun getItemsByStatus(status: UploadStatus): Flow<List<MediaItemEntity>>

    @Query("SELECT * FROM media_items WHERE uploadStatus IN (:statuses) ORDER BY createdAt ASC")
    suspend fun getPendingAndFailedItems(statuses: List<UploadStatus>): List<MediaItemEntity>

    @Query("SELECT * FROM media_items WHERE hash = :hash LIMIT 1")
    suspend fun getByHash(hash: String): MediaItemEntity?

    @Query("SELECT * FROM media_items WHERE uri = :uri LIMIT 1")
    suspend fun getByUri(uri: String): MediaItemEntity?

    @Insert(onConflict = OnConflictStrategy.IGNORE)
    suspend fun insert(item: MediaItemEntity): Long

    @Update
    suspend fun update(item: MediaItemEntity)

    @Query("UPDATE media_items SET uploadStatus = :status, errorMessage = :errorMessage WHERE id = :id")
    suspend fun updateStatus(id: Long, status: UploadStatus, errorMessage: String? = null)

    @Query("UPDATE media_items SET uploadStatus = :status, serverId = :serverId WHERE id = :id")
    suspend fun updateStatusWithServerId(id: Long, status: UploadStatus, serverId: String?)

    @Query("DELETE FROM media_items WHERE uploadStatus = 'UPLOADED' AND serverId IS NOT NULL")
    suspend fun cleanupUploadedItems()

    @Query("SELECT COUNT(*) FROM media_items WHERE uploadStatus = 'PENDING'")
    fun getPendingCount(): Flow<Int>

    @Query("SELECT COUNT(*) FROM media_items WHERE uploadStatus = 'UPLOADED' AND serverId IS NOT NULL")
    fun getUploadedCount(): Flow<Int>

    @Query("SELECT * FROM media_items ORDER BY createdAt DESC LIMIT :limit")
    suspend fun getRecentItems(limit: Int): List<MediaItemEntity>

    @Query("SELECT id FROM media_items")
    suspend fun getAllStoredMediaIds(): List<Long>

    @Query("DELETE FROM media_items WHERE id IN (:idsToDelete)")
    suspend fun deleteByMediaIds(idsToDelete: List<Long>): Int

    @Query("DELETE FROM media_items")
    suspend fun deleteAllMedia(): Int

    /**
     * Get the maximum fileCreatedAt (MediaStore DATE_ADDED) across all items.
     * Used for incremental scanning to skip already-processed files.
     */
    @Query("SELECT MAX(fileCreatedAt) FROM media_items")
    suspend fun getMaxFileCreatedAt(): Long?
}
