package com.steadyphoto.sync.data.remote.interceptor

import okhttp3.MediaType.Companion.toMediaType
import okhttp3.RequestBody
import okio.*
import java.io.File
import java.util.concurrent.atomic.AtomicLong

/**
 * A RequestBody wrapper that tracks upload progress by counting bytes written.
 */
class ProgressRequestBody(
    private val file: File,
    private val contentType: String = "application/octet-stream",
    private val listener: ProgressListener
) : RequestBody() {

    override fun contentType() = contentType.toMediaType()

    override fun contentLength() = file.length()

    override fun writeTo(sink: BufferedSink) {
        val totalBytes = file.length()
        val bytesWritten = AtomicLong(0L)
        
        // Write directly to the sink and track progress
        file.inputStream().use { inputStream ->
            val buffer = ByteArray(DEFAULT_BUFFER_SIZE)
            var bytesRead: Int
            while (inputStream.read(buffer).also { bytesRead = it } != -1) {
                if (bytesRead > 0) {
                    sink.write(buffer, 0, bytesRead)
                    bytesWritten.addAndGet(bytesRead.toLong())
                    
                    // Report progress (avoid too frequent callbacks)
                    val currentBytes = bytesWritten.get()
                    if (currentBytes % (1024L * 100L) == 0L || currentBytes >= totalBytes) {
                        listener.onProgress(currentBytes, totalBytes)
                    }
                }
            }
        }
    }

    interface ProgressListener {
        fun onProgress(bytesWritten: Long, totalBytes: Long)
    }

    companion object {
        const val DEFAULT_BUFFER_SIZE = 2048 * 8 // 16KB buffer
    }
}

/**
 * A streaming file upload helper that handles large files efficiently.
 */
class StreamingUploadHelper {
    
    data class ChunkInfo(
        val index: Int,
        val offset: Long,
        val size: Long,
        val totalChunks: Int,
        val file: File
    )

    /**
     * Splits a file into chunks for resumable uploads.
     */
    fun splitFileIntoChunks(file: File, chunkSize: Long = 5 * 1024 * 1024): List<ChunkInfo> {
        val chunks = mutableListOf<ChunkInfo>()
        var offset = 0L
        var index = 0
        val totalSize = file.length()
        val calculatedTotalChunks = ((totalSize + chunkSize - 1) / chunkSize).toInt()
        
        while (offset < totalSize) {
            val remaining = totalSize - offset
            val currentChunkSize = if (chunkSize < remaining) chunkSize else remaining
            
            chunks.add(ChunkInfo(
                index = index,
                offset = offset,
                size = currentChunkSize,
                totalChunks = calculatedTotalChunks,
                file = file
            ))
            
            offset += currentChunkSize
            index++
        }
        
        return chunks
    }
    
    /**
     * Creates a RequestBody for a specific chunk of a file.
     */
    fun createChunkRequestBody(
        chunkInfo: ChunkInfo,
        listener: ProgressRequestBody.ProgressListener? = null
    ): RequestBody {
        val bytesWrittenInChunk = AtomicLong(0L)

        return object : RequestBody() {
            override fun contentType() = "application/octet-stream".toMediaType()
            
            override fun contentLength() = chunkInfo.size
            
            override fun writeTo(sink: BufferedSink) {
                val buffer = ByteArray(ProgressRequestBody.DEFAULT_BUFFER_SIZE)
                
                // Write directly to the sink and track progress if listener is provided
                chunkInfo.file.inputStream().use { inputStream ->
                    inputStream.skip(chunkInfo.offset)
                    var totalWritten = 0L
                    while (totalWritten < chunkInfo.size) {
                        val remaining = chunkInfo.size - totalWritten
                        // Read at most 'remaining' or buffer size
                        val toRead = if (remaining < buffer.size) remaining.toInt() else buffer.size
                        val bytesRead = inputStream.read(buffer, 0, toRead)
                        if (bytesRead == -1) break
                        
                        sink.write(buffer, 0, bytesRead)
                        totalWritten += bytesRead.toLong()
                        
                        // Report progress if listener is provided
                        if (listener != null) {
                            bytesWrittenInChunk.addAndGet(bytesRead.toLong())
                            val currentBytes = bytesWrittenInChunk.get()
                            listener.onProgress(currentBytes, chunkInfo.size)
                        }
                    }
                }
            }
        }
    }

    companion object {
        private const val DEFAULT_BUFFER_SIZE = 2048 * 8
    }
}
