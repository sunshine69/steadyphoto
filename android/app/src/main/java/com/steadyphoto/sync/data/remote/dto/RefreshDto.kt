package com.steadyphoto.sync.data.remote.dto

import com.google.gson.annotations.SerializedName

/**
 * Response from refresh endpoint - returns only the new access token.
 */
data class RefreshResponse(
    @SerializedName("access_token")
    val accessToken: String
)
