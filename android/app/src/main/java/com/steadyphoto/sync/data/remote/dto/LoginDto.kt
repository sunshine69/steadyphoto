package com.steadyphoto.sync.data.remote.dto

import kotlinx.serialization.Serializable

/**
 * Request body for POST /api/v1/auth/login endpoint.
 */
@Serializable
data class LoginRequest(
    val email: String,
    val password: String
)
