package com.steadyphoto.sync.data.remote.dto

import kotlinx.serialization.Serializable

/**
 * Request body for DELETE /api/v1/media/delete endpoint.
 */
@Serializable
data class DeleteRequest(
    val media_ids: List<String> = emptyList()
)
