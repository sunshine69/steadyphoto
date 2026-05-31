package com.steadyphoto.sync.data.local

import android.content.Context
import androidx.room.Database
import androidx.room.Room
import androidx.room.RoomDatabase
import com.steadyphoto.sync.data.local.dao.MediaItemDao
import com.steadyphoto.sync.data.local.entity.MediaItemEntity

@Database(
    entities = [MediaItemEntity::class],
    version = 1,
    exportSchema = true
)
abstract class SyncDatabase : RoomDatabase() {

    abstract fun mediaItemDao(): MediaItemDao

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
