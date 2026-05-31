package com.steadyphoto.sync.util

import android.Manifest
import android.content.Context
import android.content.pm.PackageManager
import android.os.Build
import androidx.core.app.ActivityCompat
import androidx.core.content.ContextCompat

/**
 * Utility class for handling Android runtime permissions.
 */
object PermissionHelper {

    /**
     * Check if all required media permissions are granted.
     */
    fun hasMediaPermissions(context: Context): Boolean {
        return when {
            // Android 13+ (API 33+) uses granular permissions
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU -> {
                ContextCompat.checkSelfPermission(context, Manifest.permission.READ_MEDIA_IMAGES) == PackageManager.PERMISSION_GRANTED &&
                        ContextCompat.checkSelfPermission(context, Manifest.permission.READ_MEDIA_VIDEO) == PackageManager.PERMISSION_GRANTED
            }
            // Android 12 and below uses READ_EXTERNAL_STORAGE
            else -> {
                ContextCompat.checkSelfPermission(context, Manifest.permission.READ_EXTERNAL_STORAGE) == PackageManager.PERMISSION_GRANTED
            }
        }
    }

    /**
     * Get the list of permissions that need to be requested.
     */
    fun getMediaPermissionsToRequest(context: Context): Array<String> {
        return when {
            // Android 13+ (API 33+) uses granular permissions
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU -> {
                val permissions = mutableListOf<String>()
                if (ContextCompat.checkSelfPermission(context, Manifest.permission.READ_MEDIA_IMAGES) != PackageManager.PERMISSION_GRANTED) {
                    permissions.add(Manifest.permission.READ_MEDIA_IMAGES)
                }
                if (ContextCompat.checkSelfPermission(context, Manifest.permission.READ_MEDIA_VIDEO) != PackageManager.PERMISSION_GRANTED) {
                    permissions.add(Manifest.permission.READ_MEDIA_VIDEO)
                }
                permissions.toTypedArray()
            }
            // Android 12 and below uses READ_EXTERNAL_STORAGE
            else -> {
                if (ContextCompat.checkSelfPermission(context, Manifest.permission.READ_EXTERNAL_STORAGE) != PackageManager.PERMISSION_GRANTED) {
                    arrayOf(Manifest.permission.READ_EXTERNAL_STORAGE)
                } else {
                    emptyArray()
                }
            }
        }
    }

    /**
     * Check if the app has been denied a permission before (for rationale).
     */
    fun shouldShowPermissionRationale(context: Context, activity: android.app.Activity): Boolean {
        return when {
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU -> {
                ActivityCompat.shouldShowRequestPermissionRationale(activity, Manifest.permission.READ_MEDIA_IMAGES) ||
                        ActivityCompat.shouldShowRequestPermissionRationale(activity, Manifest.permission.READ_MEDIA_VIDEO)
            }
            else -> {
                ActivityCompat.shouldShowRequestPermissionRationale(activity, Manifest.permission.READ_EXTERNAL_STORAGE)
            }
        }
    }

    /**
     * Check if the user has permanently denied a permission.
     */
    fun isPermissionPermanentlyDenied(context: Context, activity: android.app.Activity): Boolean {
        return when {
            Build.VERSION.SDK_INT >= Build.VERSION_CODES.TIRAMISU -> {
                !ActivityCompat.shouldShowRequestPermissionRationale(activity, Manifest.permission.READ_MEDIA_IMAGES) &&
                        !ActivityCompat.shouldShowRequestPermissionRationale(activity, Manifest.permission.READ_MEDIA_VIDEO) &&
                        !hasMediaPermissions(context)
            }
            else -> {
                !ActivityCompat.shouldShowRequestPermissionRationale(activity, Manifest.permission.READ_EXTERNAL_STORAGE) &&
                        !hasMediaPermissions(context)
            }
        }
    }

    /**
     * Get human-readable permission rationale message.
     */
    fun getPermissionRationaleMessage(context: Context): String {
        return context.getString(com.steadyphoto.sync.R.string.permission_media_rationale)
    }

    /**
     * Get human-readable permanently denied message.
     */
    fun getPermanentlyDeniedMessage(context: Context): String {
        return context.getString(com.steadyphoto.sync.R.string.permission_media_permanently_denied)
    }
}
