package com.steadyphoto.sync.data.local.dao;

import android.database.Cursor;
import android.os.CancellationSignal;
import androidx.annotation.NonNull;
import androidx.annotation.Nullable;
import androidx.room.CoroutinesRoom;
import androidx.room.EntityDeletionOrUpdateAdapter;
import androidx.room.EntityInsertionAdapter;
import androidx.room.RoomDatabase;
import androidx.room.RoomSQLiteQuery;
import androidx.room.SharedSQLiteStatement;
import androidx.room.util.CursorUtil;
import androidx.room.util.DBUtil;
import androidx.room.util.StringUtil;
import androidx.sqlite.db.SupportSQLiteStatement;
import com.steadyphoto.sync.data.local.entity.MediaItemEntity;
import com.steadyphoto.sync.data.local.entity.UploadStatus;
import java.lang.Class;
import java.lang.Double;
import java.lang.Exception;
import java.lang.IllegalArgumentException;
import java.lang.Integer;
import java.lang.Long;
import java.lang.Object;
import java.lang.Override;
import java.lang.String;
import java.lang.StringBuilder;
import java.lang.SuppressWarnings;
import java.util.ArrayList;
import java.util.Collections;
import java.util.List;
import java.util.concurrent.Callable;
import javax.annotation.processing.Generated;
import kotlin.Unit;
import kotlin.coroutines.Continuation;
import kotlinx.coroutines.flow.Flow;

@Generated("androidx.room.RoomProcessor")
@SuppressWarnings({"unchecked", "deprecation"})
public final class MediaItemDao_Impl implements MediaItemDao {
  private final RoomDatabase __db;

  private final EntityInsertionAdapter<MediaItemEntity> __insertionAdapterOfMediaItemEntity;

  private final EntityDeletionOrUpdateAdapter<MediaItemEntity> __updateAdapterOfMediaItemEntity;

  private final SharedSQLiteStatement __preparedStmtOfUpdateStatus;

  private final SharedSQLiteStatement __preparedStmtOfCleanupUploadedItems;

  public MediaItemDao_Impl(@NonNull final RoomDatabase __db) {
    this.__db = __db;
    this.__insertionAdapterOfMediaItemEntity = new EntityInsertionAdapter<MediaItemEntity>(__db) {
      @Override
      @NonNull
      protected String createQuery() {
        return "INSERT OR IGNORE INTO `media_items` (`id`,`uri`,`localPath`,`fileName`,`hash`,`mimeType`,`fileSize`,`captureTime`,`cameraModel`,`gpsLatitude`,`gpsLongitude`,`uploadStatus`,`serverId`,`errorMessage`,`createdAt`,`lastAttemptAt`,`retryCount`) VALUES (nullif(?, 0),?,?,?,?,?,?,?,?,?,?,?,?,?,?,?,?)";
      }

      @Override
      protected void bind(@NonNull final SupportSQLiteStatement statement,
          @NonNull final MediaItemEntity entity) {
        statement.bindLong(1, entity.getId());
        statement.bindString(2, entity.getUri());
        if (entity.getLocalPath() == null) {
          statement.bindNull(3);
        } else {
          statement.bindString(3, entity.getLocalPath());
        }
        statement.bindString(4, entity.getFileName());
        statement.bindString(5, entity.getHash());
        statement.bindString(6, entity.getMimeType());
        statement.bindLong(7, entity.getFileSize());
        if (entity.getCaptureTime() == null) {
          statement.bindNull(8);
        } else {
          statement.bindLong(8, entity.getCaptureTime());
        }
        if (entity.getCameraModel() == null) {
          statement.bindNull(9);
        } else {
          statement.bindString(9, entity.getCameraModel());
        }
        if (entity.getGpsLatitude() == null) {
          statement.bindNull(10);
        } else {
          statement.bindDouble(10, entity.getGpsLatitude());
        }
        if (entity.getGpsLongitude() == null) {
          statement.bindNull(11);
        } else {
          statement.bindDouble(11, entity.getGpsLongitude());
        }
        statement.bindString(12, __UploadStatus_enumToString(entity.getUploadStatus()));
        if (entity.getServerId() == null) {
          statement.bindNull(13);
        } else {
          statement.bindString(13, entity.getServerId());
        }
        if (entity.getErrorMessage() == null) {
          statement.bindNull(14);
        } else {
          statement.bindString(14, entity.getErrorMessage());
        }
        statement.bindLong(15, entity.getCreatedAt());
        if (entity.getLastAttemptAt() == null) {
          statement.bindNull(16);
        } else {
          statement.bindLong(16, entity.getLastAttemptAt());
        }
        statement.bindLong(17, entity.getRetryCount());
      }
    };
    this.__updateAdapterOfMediaItemEntity = new EntityDeletionOrUpdateAdapter<MediaItemEntity>(__db) {
      @Override
      @NonNull
      protected String createQuery() {
        return "UPDATE OR ABORT `media_items` SET `id` = ?,`uri` = ?,`localPath` = ?,`fileName` = ?,`hash` = ?,`mimeType` = ?,`fileSize` = ?,`captureTime` = ?,`cameraModel` = ?,`gpsLatitude` = ?,`gpsLongitude` = ?,`uploadStatus` = ?,`serverId` = ?,`errorMessage` = ?,`createdAt` = ?,`lastAttemptAt` = ?,`retryCount` = ? WHERE `id` = ?";
      }

      @Override
      protected void bind(@NonNull final SupportSQLiteStatement statement,
          @NonNull final MediaItemEntity entity) {
        statement.bindLong(1, entity.getId());
        statement.bindString(2, entity.getUri());
        if (entity.getLocalPath() == null) {
          statement.bindNull(3);
        } else {
          statement.bindString(3, entity.getLocalPath());
        }
        statement.bindString(4, entity.getFileName());
        statement.bindString(5, entity.getHash());
        statement.bindString(6, entity.getMimeType());
        statement.bindLong(7, entity.getFileSize());
        if (entity.getCaptureTime() == null) {
          statement.bindNull(8);
        } else {
          statement.bindLong(8, entity.getCaptureTime());
        }
        if (entity.getCameraModel() == null) {
          statement.bindNull(9);
        } else {
          statement.bindString(9, entity.getCameraModel());
        }
        if (entity.getGpsLatitude() == null) {
          statement.bindNull(10);
        } else {
          statement.bindDouble(10, entity.getGpsLatitude());
        }
        if (entity.getGpsLongitude() == null) {
          statement.bindNull(11);
        } else {
          statement.bindDouble(11, entity.getGpsLongitude());
        }
        statement.bindString(12, __UploadStatus_enumToString(entity.getUploadStatus()));
        if (entity.getServerId() == null) {
          statement.bindNull(13);
        } else {
          statement.bindString(13, entity.getServerId());
        }
        if (entity.getErrorMessage() == null) {
          statement.bindNull(14);
        } else {
          statement.bindString(14, entity.getErrorMessage());
        }
        statement.bindLong(15, entity.getCreatedAt());
        if (entity.getLastAttemptAt() == null) {
          statement.bindNull(16);
        } else {
          statement.bindLong(16, entity.getLastAttemptAt());
        }
        statement.bindLong(17, entity.getRetryCount());
        statement.bindLong(18, entity.getId());
      }
    };
    this.__preparedStmtOfUpdateStatus = new SharedSQLiteStatement(__db) {
      @Override
      @NonNull
      public String createQuery() {
        final String _query = "UPDATE media_items SET uploadStatus = ?, errorMessage = ? WHERE id = ?";
        return _query;
      }
    };
    this.__preparedStmtOfCleanupUploadedItems = new SharedSQLiteStatement(__db) {
      @Override
      @NonNull
      public String createQuery() {
        final String _query = "DELETE FROM media_items WHERE uploadStatus = 'UPLOADED' AND serverId IS NOT NULL";
        return _query;
      }
    };
  }

  @Override
  public Object insert(final MediaItemEntity item, final Continuation<? super Long> $completion) {
    return CoroutinesRoom.execute(__db, true, new Callable<Long>() {
      @Override
      @NonNull
      public Long call() throws Exception {
        __db.beginTransaction();
        try {
          final Long _result = __insertionAdapterOfMediaItemEntity.insertAndReturnId(item);
          __db.setTransactionSuccessful();
          return _result;
        } finally {
          __db.endTransaction();
        }
      }
    }, $completion);
  }

  @Override
  public Object update(final MediaItemEntity item, final Continuation<? super Unit> $completion) {
    return CoroutinesRoom.execute(__db, true, new Callable<Unit>() {
      @Override
      @NonNull
      public Unit call() throws Exception {
        __db.beginTransaction();
        try {
          __updateAdapterOfMediaItemEntity.handle(item);
          __db.setTransactionSuccessful();
          return Unit.INSTANCE;
        } finally {
          __db.endTransaction();
        }
      }
    }, $completion);
  }

  @Override
  public Object updateStatus(final long id, final UploadStatus status, final String errorMessage,
      final Continuation<? super Unit> $completion) {
    return CoroutinesRoom.execute(__db, true, new Callable<Unit>() {
      @Override
      @NonNull
      public Unit call() throws Exception {
        final SupportSQLiteStatement _stmt = __preparedStmtOfUpdateStatus.acquire();
        int _argIndex = 1;
        _stmt.bindString(_argIndex, __UploadStatus_enumToString(status));
        _argIndex = 2;
        if (errorMessage == null) {
          _stmt.bindNull(_argIndex);
        } else {
          _stmt.bindString(_argIndex, errorMessage);
        }
        _argIndex = 3;
        _stmt.bindLong(_argIndex, id);
        try {
          __db.beginTransaction();
          try {
            _stmt.executeUpdateDelete();
            __db.setTransactionSuccessful();
            return Unit.INSTANCE;
          } finally {
            __db.endTransaction();
          }
        } finally {
          __preparedStmtOfUpdateStatus.release(_stmt);
        }
      }
    }, $completion);
  }

  @Override
  public Object cleanupUploadedItems(final Continuation<? super Unit> $completion) {
    return CoroutinesRoom.execute(__db, true, new Callable<Unit>() {
      @Override
      @NonNull
      public Unit call() throws Exception {
        final SupportSQLiteStatement _stmt = __preparedStmtOfCleanupUploadedItems.acquire();
        try {
          __db.beginTransaction();
          try {
            _stmt.executeUpdateDelete();
            __db.setTransactionSuccessful();
            return Unit.INSTANCE;
          } finally {
            __db.endTransaction();
          }
        } finally {
          __preparedStmtOfCleanupUploadedItems.release(_stmt);
        }
      }
    }, $completion);
  }

  @Override
  public Flow<List<MediaItemEntity>> getItemsByStatus(final UploadStatus status) {
    final String _sql = "SELECT * FROM media_items WHERE uploadStatus = ? ORDER BY createdAt ASC";
    final RoomSQLiteQuery _statement = RoomSQLiteQuery.acquire(_sql, 1);
    int _argIndex = 1;
    _statement.bindString(_argIndex, __UploadStatus_enumToString(status));
    return CoroutinesRoom.createFlow(__db, false, new String[] {"media_items"}, new Callable<List<MediaItemEntity>>() {
      @Override
      @NonNull
      public List<MediaItemEntity> call() throws Exception {
        final Cursor _cursor = DBUtil.query(__db, _statement, false, null);
        try {
          final int _cursorIndexOfId = CursorUtil.getColumnIndexOrThrow(_cursor, "id");
          final int _cursorIndexOfUri = CursorUtil.getColumnIndexOrThrow(_cursor, "uri");
          final int _cursorIndexOfLocalPath = CursorUtil.getColumnIndexOrThrow(_cursor, "localPath");
          final int _cursorIndexOfFileName = CursorUtil.getColumnIndexOrThrow(_cursor, "fileName");
          final int _cursorIndexOfHash = CursorUtil.getColumnIndexOrThrow(_cursor, "hash");
          final int _cursorIndexOfMimeType = CursorUtil.getColumnIndexOrThrow(_cursor, "mimeType");
          final int _cursorIndexOfFileSize = CursorUtil.getColumnIndexOrThrow(_cursor, "fileSize");
          final int _cursorIndexOfCaptureTime = CursorUtil.getColumnIndexOrThrow(_cursor, "captureTime");
          final int _cursorIndexOfCameraModel = CursorUtil.getColumnIndexOrThrow(_cursor, "cameraModel");
          final int _cursorIndexOfGpsLatitude = CursorUtil.getColumnIndexOrThrow(_cursor, "gpsLatitude");
          final int _cursorIndexOfGpsLongitude = CursorUtil.getColumnIndexOrThrow(_cursor, "gpsLongitude");
          final int _cursorIndexOfUploadStatus = CursorUtil.getColumnIndexOrThrow(_cursor, "uploadStatus");
          final int _cursorIndexOfServerId = CursorUtil.getColumnIndexOrThrow(_cursor, "serverId");
          final int _cursorIndexOfErrorMessage = CursorUtil.getColumnIndexOrThrow(_cursor, "errorMessage");
          final int _cursorIndexOfCreatedAt = CursorUtil.getColumnIndexOrThrow(_cursor, "createdAt");
          final int _cursorIndexOfLastAttemptAt = CursorUtil.getColumnIndexOrThrow(_cursor, "lastAttemptAt");
          final int _cursorIndexOfRetryCount = CursorUtil.getColumnIndexOrThrow(_cursor, "retryCount");
          final List<MediaItemEntity> _result = new ArrayList<MediaItemEntity>(_cursor.getCount());
          while (_cursor.moveToNext()) {
            final MediaItemEntity _item;
            final long _tmpId;
            _tmpId = _cursor.getLong(_cursorIndexOfId);
            final String _tmpUri;
            _tmpUri = _cursor.getString(_cursorIndexOfUri);
            final String _tmpLocalPath;
            if (_cursor.isNull(_cursorIndexOfLocalPath)) {
              _tmpLocalPath = null;
            } else {
              _tmpLocalPath = _cursor.getString(_cursorIndexOfLocalPath);
            }
            final String _tmpFileName;
            _tmpFileName = _cursor.getString(_cursorIndexOfFileName);
            final String _tmpHash;
            _tmpHash = _cursor.getString(_cursorIndexOfHash);
            final String _tmpMimeType;
            _tmpMimeType = _cursor.getString(_cursorIndexOfMimeType);
            final long _tmpFileSize;
            _tmpFileSize = _cursor.getLong(_cursorIndexOfFileSize);
            final Long _tmpCaptureTime;
            if (_cursor.isNull(_cursorIndexOfCaptureTime)) {
              _tmpCaptureTime = null;
            } else {
              _tmpCaptureTime = _cursor.getLong(_cursorIndexOfCaptureTime);
            }
            final String _tmpCameraModel;
            if (_cursor.isNull(_cursorIndexOfCameraModel)) {
              _tmpCameraModel = null;
            } else {
              _tmpCameraModel = _cursor.getString(_cursorIndexOfCameraModel);
            }
            final Double _tmpGpsLatitude;
            if (_cursor.isNull(_cursorIndexOfGpsLatitude)) {
              _tmpGpsLatitude = null;
            } else {
              _tmpGpsLatitude = _cursor.getDouble(_cursorIndexOfGpsLatitude);
            }
            final Double _tmpGpsLongitude;
            if (_cursor.isNull(_cursorIndexOfGpsLongitude)) {
              _tmpGpsLongitude = null;
            } else {
              _tmpGpsLongitude = _cursor.getDouble(_cursorIndexOfGpsLongitude);
            }
            final UploadStatus _tmpUploadStatus;
            _tmpUploadStatus = __UploadStatus_stringToEnum(_cursor.getString(_cursorIndexOfUploadStatus));
            final String _tmpServerId;
            if (_cursor.isNull(_cursorIndexOfServerId)) {
              _tmpServerId = null;
            } else {
              _tmpServerId = _cursor.getString(_cursorIndexOfServerId);
            }
            final String _tmpErrorMessage;
            if (_cursor.isNull(_cursorIndexOfErrorMessage)) {
              _tmpErrorMessage = null;
            } else {
              _tmpErrorMessage = _cursor.getString(_cursorIndexOfErrorMessage);
            }
            final long _tmpCreatedAt;
            _tmpCreatedAt = _cursor.getLong(_cursorIndexOfCreatedAt);
            final Long _tmpLastAttemptAt;
            if (_cursor.isNull(_cursorIndexOfLastAttemptAt)) {
              _tmpLastAttemptAt = null;
            } else {
              _tmpLastAttemptAt = _cursor.getLong(_cursorIndexOfLastAttemptAt);
            }
            final int _tmpRetryCount;
            _tmpRetryCount = _cursor.getInt(_cursorIndexOfRetryCount);
            _item = new MediaItemEntity(_tmpId,_tmpUri,_tmpLocalPath,_tmpFileName,_tmpHash,_tmpMimeType,_tmpFileSize,_tmpCaptureTime,_tmpCameraModel,_tmpGpsLatitude,_tmpGpsLongitude,_tmpUploadStatus,_tmpServerId,_tmpErrorMessage,_tmpCreatedAt,_tmpLastAttemptAt,_tmpRetryCount);
            _result.add(_item);
          }
          return _result;
        } finally {
          _cursor.close();
        }
      }

      @Override
      protected void finalize() {
        _statement.release();
      }
    });
  }

  @Override
  public Object getPendingAndFailedItems(final List<? extends UploadStatus> statuses,
      final Continuation<? super List<MediaItemEntity>> $completion) {
    final StringBuilder _stringBuilder = StringUtil.newStringBuilder();
    _stringBuilder.append("SELECT * FROM media_items WHERE uploadStatus IN (");
    final int _inputSize = statuses.size();
    StringUtil.appendPlaceholders(_stringBuilder, _inputSize);
    _stringBuilder.append(") ORDER BY createdAt ASC");
    final String _sql = _stringBuilder.toString();
    final int _argCount = 0 + _inputSize;
    final RoomSQLiteQuery _statement = RoomSQLiteQuery.acquire(_sql, _argCount);
    int _argIndex = 1;
    for (UploadStatus _item : statuses) {
      _statement.bindString(_argIndex, __UploadStatus_enumToString(_item));
      _argIndex++;
    }
    final CancellationSignal _cancellationSignal = DBUtil.createCancellationSignal();
    return CoroutinesRoom.execute(__db, false, _cancellationSignal, new Callable<List<MediaItemEntity>>() {
      @Override
      @NonNull
      public List<MediaItemEntity> call() throws Exception {
        final Cursor _cursor = DBUtil.query(__db, _statement, false, null);
        try {
          final int _cursorIndexOfId = CursorUtil.getColumnIndexOrThrow(_cursor, "id");
          final int _cursorIndexOfUri = CursorUtil.getColumnIndexOrThrow(_cursor, "uri");
          final int _cursorIndexOfLocalPath = CursorUtil.getColumnIndexOrThrow(_cursor, "localPath");
          final int _cursorIndexOfFileName = CursorUtil.getColumnIndexOrThrow(_cursor, "fileName");
          final int _cursorIndexOfHash = CursorUtil.getColumnIndexOrThrow(_cursor, "hash");
          final int _cursorIndexOfMimeType = CursorUtil.getColumnIndexOrThrow(_cursor, "mimeType");
          final int _cursorIndexOfFileSize = CursorUtil.getColumnIndexOrThrow(_cursor, "fileSize");
          final int _cursorIndexOfCaptureTime = CursorUtil.getColumnIndexOrThrow(_cursor, "captureTime");
          final int _cursorIndexOfCameraModel = CursorUtil.getColumnIndexOrThrow(_cursor, "cameraModel");
          final int _cursorIndexOfGpsLatitude = CursorUtil.getColumnIndexOrThrow(_cursor, "gpsLatitude");
          final int _cursorIndexOfGpsLongitude = CursorUtil.getColumnIndexOrThrow(_cursor, "gpsLongitude");
          final int _cursorIndexOfUploadStatus = CursorUtil.getColumnIndexOrThrow(_cursor, "uploadStatus");
          final int _cursorIndexOfServerId = CursorUtil.getColumnIndexOrThrow(_cursor, "serverId");
          final int _cursorIndexOfErrorMessage = CursorUtil.getColumnIndexOrThrow(_cursor, "errorMessage");
          final int _cursorIndexOfCreatedAt = CursorUtil.getColumnIndexOrThrow(_cursor, "createdAt");
          final int _cursorIndexOfLastAttemptAt = CursorUtil.getColumnIndexOrThrow(_cursor, "lastAttemptAt");
          final int _cursorIndexOfRetryCount = CursorUtil.getColumnIndexOrThrow(_cursor, "retryCount");
          final List<MediaItemEntity> _result = new ArrayList<MediaItemEntity>(_cursor.getCount());
          while (_cursor.moveToNext()) {
            final MediaItemEntity _item_1;
            final long _tmpId;
            _tmpId = _cursor.getLong(_cursorIndexOfId);
            final String _tmpUri;
            _tmpUri = _cursor.getString(_cursorIndexOfUri);
            final String _tmpLocalPath;
            if (_cursor.isNull(_cursorIndexOfLocalPath)) {
              _tmpLocalPath = null;
            } else {
              _tmpLocalPath = _cursor.getString(_cursorIndexOfLocalPath);
            }
            final String _tmpFileName;
            _tmpFileName = _cursor.getString(_cursorIndexOfFileName);
            final String _tmpHash;
            _tmpHash = _cursor.getString(_cursorIndexOfHash);
            final String _tmpMimeType;
            _tmpMimeType = _cursor.getString(_cursorIndexOfMimeType);
            final long _tmpFileSize;
            _tmpFileSize = _cursor.getLong(_cursorIndexOfFileSize);
            final Long _tmpCaptureTime;
            if (_cursor.isNull(_cursorIndexOfCaptureTime)) {
              _tmpCaptureTime = null;
            } else {
              _tmpCaptureTime = _cursor.getLong(_cursorIndexOfCaptureTime);
            }
            final String _tmpCameraModel;
            if (_cursor.isNull(_cursorIndexOfCameraModel)) {
              _tmpCameraModel = null;
            } else {
              _tmpCameraModel = _cursor.getString(_cursorIndexOfCameraModel);
            }
            final Double _tmpGpsLatitude;
            if (_cursor.isNull(_cursorIndexOfGpsLatitude)) {
              _tmpGpsLatitude = null;
            } else {
              _tmpGpsLatitude = _cursor.getDouble(_cursorIndexOfGpsLatitude);
            }
            final Double _tmpGpsLongitude;
            if (_cursor.isNull(_cursorIndexOfGpsLongitude)) {
              _tmpGpsLongitude = null;
            } else {
              _tmpGpsLongitude = _cursor.getDouble(_cursorIndexOfGpsLongitude);
            }
            final UploadStatus _tmpUploadStatus;
            _tmpUploadStatus = __UploadStatus_stringToEnum(_cursor.getString(_cursorIndexOfUploadStatus));
            final String _tmpServerId;
            if (_cursor.isNull(_cursorIndexOfServerId)) {
              _tmpServerId = null;
            } else {
              _tmpServerId = _cursor.getString(_cursorIndexOfServerId);
            }
            final String _tmpErrorMessage;
            if (_cursor.isNull(_cursorIndexOfErrorMessage)) {
              _tmpErrorMessage = null;
            } else {
              _tmpErrorMessage = _cursor.getString(_cursorIndexOfErrorMessage);
            }
            final long _tmpCreatedAt;
            _tmpCreatedAt = _cursor.getLong(_cursorIndexOfCreatedAt);
            final Long _tmpLastAttemptAt;
            if (_cursor.isNull(_cursorIndexOfLastAttemptAt)) {
              _tmpLastAttemptAt = null;
            } else {
              _tmpLastAttemptAt = _cursor.getLong(_cursorIndexOfLastAttemptAt);
            }
            final int _tmpRetryCount;
            _tmpRetryCount = _cursor.getInt(_cursorIndexOfRetryCount);
            _item_1 = new MediaItemEntity(_tmpId,_tmpUri,_tmpLocalPath,_tmpFileName,_tmpHash,_tmpMimeType,_tmpFileSize,_tmpCaptureTime,_tmpCameraModel,_tmpGpsLatitude,_tmpGpsLongitude,_tmpUploadStatus,_tmpServerId,_tmpErrorMessage,_tmpCreatedAt,_tmpLastAttemptAt,_tmpRetryCount);
            _result.add(_item_1);
          }
          return _result;
        } finally {
          _cursor.close();
          _statement.release();
        }
      }
    }, $completion);
  }

  @Override
  public Object getByHash(final String hash,
      final Continuation<? super MediaItemEntity> $completion) {
    final String _sql = "SELECT * FROM media_items WHERE hash = ? LIMIT 1";
    final RoomSQLiteQuery _statement = RoomSQLiteQuery.acquire(_sql, 1);
    int _argIndex = 1;
    _statement.bindString(_argIndex, hash);
    final CancellationSignal _cancellationSignal = DBUtil.createCancellationSignal();
    return CoroutinesRoom.execute(__db, false, _cancellationSignal, new Callable<MediaItemEntity>() {
      @Override
      @Nullable
      public MediaItemEntity call() throws Exception {
        final Cursor _cursor = DBUtil.query(__db, _statement, false, null);
        try {
          final int _cursorIndexOfId = CursorUtil.getColumnIndexOrThrow(_cursor, "id");
          final int _cursorIndexOfUri = CursorUtil.getColumnIndexOrThrow(_cursor, "uri");
          final int _cursorIndexOfLocalPath = CursorUtil.getColumnIndexOrThrow(_cursor, "localPath");
          final int _cursorIndexOfFileName = CursorUtil.getColumnIndexOrThrow(_cursor, "fileName");
          final int _cursorIndexOfHash = CursorUtil.getColumnIndexOrThrow(_cursor, "hash");
          final int _cursorIndexOfMimeType = CursorUtil.getColumnIndexOrThrow(_cursor, "mimeType");
          final int _cursorIndexOfFileSize = CursorUtil.getColumnIndexOrThrow(_cursor, "fileSize");
          final int _cursorIndexOfCaptureTime = CursorUtil.getColumnIndexOrThrow(_cursor, "captureTime");
          final int _cursorIndexOfCameraModel = CursorUtil.getColumnIndexOrThrow(_cursor, "cameraModel");
          final int _cursorIndexOfGpsLatitude = CursorUtil.getColumnIndexOrThrow(_cursor, "gpsLatitude");
          final int _cursorIndexOfGpsLongitude = CursorUtil.getColumnIndexOrThrow(_cursor, "gpsLongitude");
          final int _cursorIndexOfUploadStatus = CursorUtil.getColumnIndexOrThrow(_cursor, "uploadStatus");
          final int _cursorIndexOfServerId = CursorUtil.getColumnIndexOrThrow(_cursor, "serverId");
          final int _cursorIndexOfErrorMessage = CursorUtil.getColumnIndexOrThrow(_cursor, "errorMessage");
          final int _cursorIndexOfCreatedAt = CursorUtil.getColumnIndexOrThrow(_cursor, "createdAt");
          final int _cursorIndexOfLastAttemptAt = CursorUtil.getColumnIndexOrThrow(_cursor, "lastAttemptAt");
          final int _cursorIndexOfRetryCount = CursorUtil.getColumnIndexOrThrow(_cursor, "retryCount");
          final MediaItemEntity _result;
          if (_cursor.moveToFirst()) {
            final long _tmpId;
            _tmpId = _cursor.getLong(_cursorIndexOfId);
            final String _tmpUri;
            _tmpUri = _cursor.getString(_cursorIndexOfUri);
            final String _tmpLocalPath;
            if (_cursor.isNull(_cursorIndexOfLocalPath)) {
              _tmpLocalPath = null;
            } else {
              _tmpLocalPath = _cursor.getString(_cursorIndexOfLocalPath);
            }
            final String _tmpFileName;
            _tmpFileName = _cursor.getString(_cursorIndexOfFileName);
            final String _tmpHash;
            _tmpHash = _cursor.getString(_cursorIndexOfHash);
            final String _tmpMimeType;
            _tmpMimeType = _cursor.getString(_cursorIndexOfMimeType);
            final long _tmpFileSize;
            _tmpFileSize = _cursor.getLong(_cursorIndexOfFileSize);
            final Long _tmpCaptureTime;
            if (_cursor.isNull(_cursorIndexOfCaptureTime)) {
              _tmpCaptureTime = null;
            } else {
              _tmpCaptureTime = _cursor.getLong(_cursorIndexOfCaptureTime);
            }
            final String _tmpCameraModel;
            if (_cursor.isNull(_cursorIndexOfCameraModel)) {
              _tmpCameraModel = null;
            } else {
              _tmpCameraModel = _cursor.getString(_cursorIndexOfCameraModel);
            }
            final Double _tmpGpsLatitude;
            if (_cursor.isNull(_cursorIndexOfGpsLatitude)) {
              _tmpGpsLatitude = null;
            } else {
              _tmpGpsLatitude = _cursor.getDouble(_cursorIndexOfGpsLatitude);
            }
            final Double _tmpGpsLongitude;
            if (_cursor.isNull(_cursorIndexOfGpsLongitude)) {
              _tmpGpsLongitude = null;
            } else {
              _tmpGpsLongitude = _cursor.getDouble(_cursorIndexOfGpsLongitude);
            }
            final UploadStatus _tmpUploadStatus;
            _tmpUploadStatus = __UploadStatus_stringToEnum(_cursor.getString(_cursorIndexOfUploadStatus));
            final String _tmpServerId;
            if (_cursor.isNull(_cursorIndexOfServerId)) {
              _tmpServerId = null;
            } else {
              _tmpServerId = _cursor.getString(_cursorIndexOfServerId);
            }
            final String _tmpErrorMessage;
            if (_cursor.isNull(_cursorIndexOfErrorMessage)) {
              _tmpErrorMessage = null;
            } else {
              _tmpErrorMessage = _cursor.getString(_cursorIndexOfErrorMessage);
            }
            final long _tmpCreatedAt;
            _tmpCreatedAt = _cursor.getLong(_cursorIndexOfCreatedAt);
            final Long _tmpLastAttemptAt;
            if (_cursor.isNull(_cursorIndexOfLastAttemptAt)) {
              _tmpLastAttemptAt = null;
            } else {
              _tmpLastAttemptAt = _cursor.getLong(_cursorIndexOfLastAttemptAt);
            }
            final int _tmpRetryCount;
            _tmpRetryCount = _cursor.getInt(_cursorIndexOfRetryCount);
            _result = new MediaItemEntity(_tmpId,_tmpUri,_tmpLocalPath,_tmpFileName,_tmpHash,_tmpMimeType,_tmpFileSize,_tmpCaptureTime,_tmpCameraModel,_tmpGpsLatitude,_tmpGpsLongitude,_tmpUploadStatus,_tmpServerId,_tmpErrorMessage,_tmpCreatedAt,_tmpLastAttemptAt,_tmpRetryCount);
          } else {
            _result = null;
          }
          return _result;
        } finally {
          _cursor.close();
          _statement.release();
        }
      }
    }, $completion);
  }

  @Override
  public Flow<Integer> getPendingCount() {
    final String _sql = "SELECT COUNT(*) FROM media_items WHERE uploadStatus = 'PENDING'";
    final RoomSQLiteQuery _statement = RoomSQLiteQuery.acquire(_sql, 0);
    return CoroutinesRoom.createFlow(__db, false, new String[] {"media_items"}, new Callable<Integer>() {
      @Override
      @NonNull
      public Integer call() throws Exception {
        final Cursor _cursor = DBUtil.query(__db, _statement, false, null);
        try {
          final Integer _result;
          if (_cursor.moveToFirst()) {
            final int _tmp;
            _tmp = _cursor.getInt(0);
            _result = _tmp;
          } else {
            _result = 0;
          }
          return _result;
        } finally {
          _cursor.close();
        }
      }

      @Override
      protected void finalize() {
        _statement.release();
      }
    });
  }

  @Override
  public Flow<Integer> getUploadedCount() {
    final String _sql = "SELECT COUNT(*) FROM media_items WHERE uploadStatus = 'UPLOADED' AND serverId IS NOT NULL";
    final RoomSQLiteQuery _statement = RoomSQLiteQuery.acquire(_sql, 0);
    return CoroutinesRoom.createFlow(__db, false, new String[] {"media_items"}, new Callable<Integer>() {
      @Override
      @NonNull
      public Integer call() throws Exception {
        final Cursor _cursor = DBUtil.query(__db, _statement, false, null);
        try {
          final Integer _result;
          if (_cursor.moveToFirst()) {
            final int _tmp;
            _tmp = _cursor.getInt(0);
            _result = _tmp;
          } else {
            _result = 0;
          }
          return _result;
        } finally {
          _cursor.close();
        }
      }

      @Override
      protected void finalize() {
        _statement.release();
      }
    });
  }

  @NonNull
  public static List<Class<?>> getRequiredConverters() {
    return Collections.emptyList();
  }

  private String __UploadStatus_enumToString(@NonNull final UploadStatus _value) {
    switch (_value) {
      case PENDING: return "PENDING";
      case UPLOADING: return "UPLOADING";
      case UPLOADED: return "UPLOADED";
      case FAILED: return "FAILED";
      case SKIPPED_DUPLICATE: return "SKIPPED_DUPLICATE";
      case CANCELLED: return "CANCELLED";
      case DELETED: return "DELETED";
      default: throw new IllegalArgumentException("Can't convert enum to string, unknown enum value: " + _value);
    }
  }

  private UploadStatus __UploadStatus_stringToEnum(@NonNull final String _value) {
    switch (_value) {
      case "PENDING": return UploadStatus.PENDING;
      case "UPLOADING": return UploadStatus.UPLOADING;
      case "UPLOADED": return UploadStatus.UPLOADED;
      case "FAILED": return UploadStatus.FAILED;
      case "SKIPPED_DUPLICATE": return UploadStatus.SKIPPED_DUPLICATE;
      case "CANCELLED": return UploadStatus.CANCELLED;
      case "DELETED": return UploadStatus.DELETED;
      default: throw new IllegalArgumentException("Can't convert value to enum, unknown value: " + _value);
    }
  }
}
