package com.steadyphoto.sync.data.remote.dto

import kotlinx.serialization.Serializable

/**
 * Request body for POST /api/v1/auth/refresh endpoint - sends refresh token to get a new access token.
 */
@Serializable
data class RefreshRequest(
    val refresh_token: String
)
