package com.steadyphoto.sync.data.local.entity

import androidx.room.Entity
import androidx.room.Index
import androidx.room.PrimaryKey

/**
 * Entity representing a media file on the device that may be uploaded to the server.
 */
@Entity(
    tableName = "media_items",
    indices = [Index(value = ["hash"], unique = true)]
)
data class MediaItemEntity(
    @PrimaryKey(autoGenerate = true)
    val id: Long = 0,

    /** MediaStore URI for accessing the file */
    val uri: String,

    /** Absolute path to the file on the device (fallback/backup) */
    val localPath: String? = null,

    /** Filename of the media item */
    val fileName: String,

    /** SHA256 hash for deduplication (computed by Gomobile) */
    val hash: String,

    /** MIME type (image/jpeg, video/mp4, etc.) */
    val mimeType: String,

    /** File size in bytes */
    val fileSize: Long,

    /** Capture time from EXIF data (if available) */
    val captureTime: Long? = null,

    /** Camera model from EXIF (if available) */
    val cameraModel: String? = null,

    /** GPS latitude from EXIF (if available) */
    val gpsLatitude: Double? = null,

    /** GPS longitude from EXIF (if available) */
    val gpsLongitude: Double? = null,

    /** Upload status */
    val uploadStatus: UploadStatus = UploadStatus.PENDING,

    /** Server-assigned ID after successful upload */
    val serverId: String? = null,

    /** Error message if upload failed */
    val errorMessage: String? = null,

    /** Timestamp when the item was added to local DB */
    val createdAt: Long = System.currentTimeMillis(),

    /** Timestamp of last upload attempt */
    val lastAttemptAt: Long? = null,

    /** Number of upload retry attempts */
    val retryCount: Int = 0
)

enum class UploadStatus {
    PENDING,           // Not yet uploaded
    UPLOADING,         // Currently being uploaded
    UPLOADED,          // Successfully uploaded to server
    FAILED,            // Upload failed (will be retried)
    SKIPPED_DUPLICATE, // Server already has this file
    CANCELLED,         // User cancelled the upload
    DELETED            // Deleted from local and remote
}
