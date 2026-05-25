package com.steadyphoto.sync.data.local;

import androidx.annotation.NonNull;
import androidx.room.DatabaseConfiguration;
import androidx.room.InvalidationTracker;
import androidx.room.RoomDatabase;
import androidx.room.RoomOpenHelper;
import androidx.room.migration.AutoMigrationSpec;
import androidx.room.migration.Migration;
import androidx.room.util.DBUtil;
import androidx.room.util.TableInfo;
import androidx.sqlite.db.SupportSQLiteDatabase;
import androidx.sqlite.db.SupportSQLiteOpenHelper;
import com.steadyphoto.sync.data.local.dao.MediaItemDao;
import com.steadyphoto.sync.data.local.dao.MediaItemDao_Impl;
import java.lang.Class;
import java.lang.Override;
import java.lang.String;
import java.lang.SuppressWarnings;
import java.util.ArrayList;
import java.util.Arrays;
import java.util.HashMap;
import java.util.HashSet;
import java.util.List;
import java.util.Map;
import java.util.Set;
import javax.annotation.processing.Generated;

@Generated("androidx.room.RoomProcessor")
@SuppressWarnings({"unchecked", "deprecation"})
public final class SyncDatabase_Impl extends SyncDatabase {
  private volatile MediaItemDao _mediaItemDao;

  @Override
  @NonNull
  protected SupportSQLiteOpenHelper createOpenHelper(@NonNull final DatabaseConfiguration config) {
    final SupportSQLiteOpenHelper.Callback _openCallback = new RoomOpenHelper(config, new RoomOpenHelper.Delegate(1) {
      @Override
      public void createAllTables(@NonNull final SupportSQLiteDatabase db) {
        db.execSQL("CREATE TABLE IF NOT EXISTS `media_items` (`id` INTEGER PRIMARY KEY AUTOINCREMENT NOT NULL, `uri` TEXT NOT NULL, `localPath` TEXT, `fileName` TEXT NOT NULL, `hash` TEXT NOT NULL, `mimeType` TEXT NOT NULL, `fileSize` INTEGER NOT NULL, `captureTime` INTEGER, `cameraModel` TEXT, `gpsLatitude` REAL, `gpsLongitude` REAL, `uploadStatus` TEXT NOT NULL, `serverId` TEXT, `errorMessage` TEXT, `createdAt` INTEGER NOT NULL, `lastAttemptAt` INTEGER, `retryCount` INTEGER NOT NULL)");
        db.execSQL("CREATE UNIQUE INDEX IF NOT EXISTS `index_media_items_hash` ON `media_items` (`hash`)");
        db.execSQL("CREATE TABLE IF NOT EXISTS room_master_table (id INTEGER PRIMARY KEY,identity_hash TEXT)");
        db.execSQL("INSERT OR REPLACE INTO room_master_table (id,identity_hash) VALUES(42, 'b6ff09c57598040c361f9fab1d90978e')");
      }

      @Override
      public void dropAllTables(@NonNull final SupportSQLiteDatabase db) {
        db.execSQL("DROP TABLE IF EXISTS `media_items`");
        final List<? extends RoomDatabase.Callback> _callbacks = mCallbacks;
        if (_callbacks != null) {
          for (RoomDatabase.Callback _callback : _callbacks) {
            _callback.onDestructiveMigration(db);
          }
        }
      }

      @Override
      public void onCreate(@NonNull final SupportSQLiteDatabase db) {
        final List<? extends RoomDatabase.Callback> _callbacks = mCallbacks;
        if (_callbacks != null) {
          for (RoomDatabase.Callback _callback : _callbacks) {
            _callback.onCreate(db);
          }
        }
      }

      @Override
      public void onOpen(@NonNull final SupportSQLiteDatabase db) {
        mDatabase = db;
        internalInitInvalidationTracker(db);
        final List<? extends RoomDatabase.Callback> _callbacks = mCallbacks;
        if (_callbacks != null) {
          for (RoomDatabase.Callback _callback : _callbacks) {
            _callback.onOpen(db);
          }
        }
      }

      @Override
      public void onPreMigrate(@NonNull final SupportSQLiteDatabase db) {
        DBUtil.dropFtsSyncTriggers(db);
      }

      @Override
      public void onPostMigrate(@NonNull final SupportSQLiteDatabase db) {
      }

      @Override
      @NonNull
      public RoomOpenHelper.ValidationResult onValidateSchema(
          @NonNull final SupportSQLiteDatabase db) {
        final HashMap<String, TableInfo.Column> _columnsMediaItems = new HashMap<String, TableInfo.Column>(17);
        _columnsMediaItems.put("id", new TableInfo.Column("id", "INTEGER", true, 1, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("uri", new TableInfo.Column("uri", "TEXT", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("localPath", new TableInfo.Column("localPath", "TEXT", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("fileName", new TableInfo.Column("fileName", "TEXT", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("hash", new TableInfo.Column("hash", "TEXT", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("mimeType", new TableInfo.Column("mimeType", "TEXT", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("fileSize", new TableInfo.Column("fileSize", "INTEGER", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("captureTime", new TableInfo.Column("captureTime", "INTEGER", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("cameraModel", new TableInfo.Column("cameraModel", "TEXT", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("gpsLatitude", new TableInfo.Column("gpsLatitude", "REAL", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("gpsLongitude", new TableInfo.Column("gpsLongitude", "REAL", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("uploadStatus", new TableInfo.Column("uploadStatus", "TEXT", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("serverId", new TableInfo.Column("serverId", "TEXT", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("errorMessage", new TableInfo.Column("errorMessage", "TEXT", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("createdAt", new TableInfo.Column("createdAt", "INTEGER", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("lastAttemptAt", new TableInfo.Column("lastAttemptAt", "INTEGER", false, 0, null, TableInfo.CREATED_FROM_ENTITY));
        _columnsMediaItems.put("retryCount", new TableInfo.Column("retryCount", "INTEGER", true, 0, null, TableInfo.CREATED_FROM_ENTITY));
        final HashSet<TableInfo.ForeignKey> _foreignKeysMediaItems = new HashSet<TableInfo.ForeignKey>(0);
        final HashSet<TableInfo.Index> _indicesMediaItems = new HashSet<TableInfo.Index>(1);
        _indicesMediaItems.add(new TableInfo.Index("index_media_items_hash", true, Arrays.asList("hash"), Arrays.asList("ASC")));
        final TableInfo _infoMediaItems = new TableInfo("media_items", _columnsMediaItems, _foreignKeysMediaItems, _indicesMediaItems);
        final TableInfo _existingMediaItems = TableInfo.read(db, "media_items");
        if (!_infoMediaItems.equals(_existingMediaItems)) {
          return new RoomOpenHelper.ValidationResult(false, "media_items(com.steadyphoto.sync.data.local.entity.MediaItemEntity).\n"
                  + " Expected:\n" + _infoMediaItems + "\n"
                  + " Found:\n" + _existingMediaItems);
        }
        return new RoomOpenHelper.ValidationResult(true, null);
      }
    }, "b6ff09c57598040c361f9fab1d90978e", "a41327d53f3ba2c4c2a385d3ef8f0f23");
    final SupportSQLiteOpenHelper.Configuration _sqliteConfig = SupportSQLiteOpenHelper.Configuration.builder(config.context).name(config.name).callback(_openCallback).build();
    final SupportSQLiteOpenHelper _helper = config.sqliteOpenHelperFactory.create(_sqliteConfig);
    return _helper;
  }

  @Override
  @NonNull
  protected InvalidationTracker createInvalidationTracker() {
    final HashMap<String, String> _shadowTablesMap = new HashMap<String, String>(0);
    final HashMap<String, Set<String>> _viewTables = new HashMap<String, Set<String>>(0);
    return new InvalidationTracker(this, _shadowTablesMap, _viewTables, "media_items");
  }

  @Override
  public void clearAllTables() {
    super.assertNotMainThread();
    final SupportSQLiteDatabase _db = super.getOpenHelper().getWritableDatabase();
    try {
      super.beginTransaction();
      _db.execSQL("DELETE FROM `media_items`");
      super.setTransactionSuccessful();
    } finally {
      super.endTransaction();
      _db.query("PRAGMA wal_checkpoint(FULL)").close();
      if (!_db.inTransaction()) {
        _db.execSQL("VACUUM");
      }
    }
  }

  @Override
  @NonNull
  protected Map<Class<?>, List<Class<?>>> getRequiredTypeConverters() {
    final HashMap<Class<?>, List<Class<?>>> _typeConvertersMap = new HashMap<Class<?>, List<Class<?>>>();
    _typeConvertersMap.put(MediaItemDao.class, MediaItemDao_Impl.getRequiredConverters());
    return _typeConvertersMap;
  }

  @Override
  @NonNull
  public Set<Class<? extends AutoMigrationSpec>> getRequiredAutoMigrationSpecs() {
    final HashSet<Class<? extends AutoMigrationSpec>> _autoMigrationSpecsSet = new HashSet<Class<? extends AutoMigrationSpec>>();
    return _autoMigrationSpecsSet;
  }

  @Override
  @NonNull
  public List<Migration> getAutoMigrations(
      @NonNull final Map<Class<? extends AutoMigrationSpec>, AutoMigrationSpec> autoMigrationSpecs) {
    final List<Migration> _autoMigrations = new ArrayList<Migration>();
    return _autoMigrations;
  }

  @Override
  public MediaItemDao mediaItemDao() {
    if (_mediaItemDao != null) {
      return _mediaItemDao;
    } else {
      synchronized(this) {
        if(_mediaItemDao == null) {
          _mediaItemDao = new MediaItemDao_Impl(this);
        }
        return _mediaItemDao;
      }
    }
  }
}
