package com.steadyphoto.sync.data.repository

import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import kotlinx.coroutines.flow.Flow

interface SyncRepository {
    
    // Media item operations
    fun getPendingItems(): Flow<List<MediaItemEntity>>
    fun getUploadedItems(): Flow<List<MediaItemEntity>>
    suspend fun scanNewMedia(): ScanResult
    suspend fun uploadMedia(items: List<MediaItemEntity>): Result<Unit>
    suspend fun markAsUploaded(item: MediaItemEntity)
    suspend fun deleteMedia(itemId: String): Result<Unit>
    suspend fun getSyncStatus(limit: Int = 50): Result<List<com.steadyphoto.sync.data.remote.dto.UploadedMedia>>
}
