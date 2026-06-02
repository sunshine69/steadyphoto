package com.steadyphoto.sync.util

import android.content.ContentResolver
import android.content.Context
import android.net.Uri
import mobile.MediaProcessor
import mobile.FileMetadata
import java.io.InputStream
import java.security.MessageDigest

object MediaUtils {
    
    private val processor = MediaProcessor()
    
    /**
     * Computes SHA256 hash of a file using Gomobile bindings.
     */
    fun computeHash(filePath: String): String? {
        return try {
            processor.mediaHasher(filePath)
        } catch (e: Exception) {
            android.util.Log.e("MediaUtils", "Exception computing hash for $filePath", e)
            null
        }
    }
    
    /**
     * Computes SHA256 hash of a file using ContentResolver openInputStream.
     * This works on Android 10+ where localPath is not accessible due to scoped storage.
     */
    fun computeHashFromUri(context: Context, uri: Uri): String? {
        return try {
            context.contentResolver.openInputStream(uri)?.use { inputStream ->
                hashStream(inputStream)
            }
        } catch (e: Exception) {
            android.util.Log.e("MediaUtils", "Exception computing hash for URI $uri", e)
            null
        }
    }
    
    /**
     * Helper function to compute SHA256 hash from an InputStream.
     */
    private fun hashStream(inputStream: InputStream): String {
        val md = MessageDigest.getInstance("SHA-256")
        inputStream.use { stream ->
            val buffer = ByteArray(8192)
            var bytesRead: Int
            while (stream.read(buffer).also { bytesRead = it } != -1) {
                md.update(buffer, 0, bytesRead)
            }
        }
        return md.digest().joinToString("") { "%02x".format(it) }
    }
    
    /**
     * Gets file metadata using Gomobile bindings.
     */
    fun getFileMetadata(filePath: String): FileMetadata? {
        return try {
            processor.getFileMetadata(filePath)
        } catch (e: Exception) {
            android.util.Log.e("MediaUtils", "Exception getting metadata for $filePath", e)
            null
        }
    }
}
