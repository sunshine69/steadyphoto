package com.steadyphoto.sync.data.repository

import com.steadyphoto.sync.data.local.entity.MediaItemEntity

/**
 * Sealed class representing the result of a media scan operation.
 */
sealed class ScanResult {
    /**
     * Scan completed successfully with new items found.
     */
    data class Success(
        val totalScanned: Int,
        val newItemsInserted: Int,
        val duplicatesSkipped: Int
    ) : ScanResult()

    /**
     * Scan completed but no new items were found.
     */
    object NoNewItems : ScanResult()

    /**
     * Scan failed due to missing permissions.
     */
    object PermissionDenied : ScanResult()

    /**
     * Scan failed due to an error.
     */
    data class Error(
        val message: String,
        val cause: Throwable? = null
    ) : ScanResult()
}
