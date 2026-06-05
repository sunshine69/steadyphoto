package com.steadyphoto.sync.data.repository

/**
 * Result of scanning MediaStore for media files.
 */
data class MediaScanResult(
    val totalScanned: Int,      // Total number of rows examined in MediaStore
    val newItemsInserted: Int,   // Number of NEW items added to the local DB
    val duplicatesSkipped: Int   // Number of duplicates skipped (already in DB)
)
