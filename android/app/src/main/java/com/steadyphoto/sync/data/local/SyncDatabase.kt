package com.steadyphoto.sync.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.dao.SyncStateDao
import com.steadyphoto.sync.data.local.entity.MediaItemEntity
import com.steadyphoto.sync.data.local.entity.SyncStateEntity

@Database(
    entities = [MediaItemEntity::class, SyncStateEntity::class],
    version = 2,
    exportSchema = false
)
abstract class SyncDatabase : RoomDatabase() {

    abstract fun mediaItemDao(): MediaItemDao
    abstract fun syncStateDao(): SyncStateDao

    companion object {
        @Volatile
        private var INSTANCE: SyncDatabase? = null

        fun getDatabase(context: Context): SyncDatabase {
            return INSTANCE ?: synchronized(this) {
                val instance = Room.databaseBuilder(
                    context.applicationContext,
                    SyncDatabase::class.java,
                    "sync_database"
                ).build()
                INSTANCE = instance
                instance
            }
        }
    }
}
