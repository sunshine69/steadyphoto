package com.steadyphoto.sync.util

import android.content.Context
import mobile.MediaProcessor
import mobile.FileMetadata

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
