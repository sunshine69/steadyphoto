package com.steadyphoto.sync.worker

import android.os.FileObserver as AndroidFileObserver

/**
 * Shared FileObserver event constant aliases for the worker package.
 * 
 * These are defined in one place to avoid duplicate const declarations across
 * multiple files in the same Kotlin package, which would cause compilation errors.
 */
object FileObserverConstants {
    val CREATE = AndroidFileObserver.CREATE
    val MODIFY = AndroidFileObserver.MODIFY  // Android uses MODIFY, not MODIFIED
    val DELETE = AndroidFileObserver.DELETE
    val MOVED_FROM = AndroidFileObserver.MOVED_FROM
    val MOVED_TO = AndroidFileObserver.MOVED_TO
    val CLOSE_WRITE = AndroidFileObserver.CLOSE_WRITE
    
    /** Events we care about - only create and modify events for files */
    val EVENTS_MASK = CREATE or MODIFY
}
