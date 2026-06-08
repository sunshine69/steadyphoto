package com.steadyphoto.sync.data.remote.api

import android.content.Context
import okhttp3.Interceptor
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.ResponseBody.Companion.toResponseBody


import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.Response
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Call
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.io.IOException
import java.util.concurrent.TimeUnit

// Callback interface for auth failure events (used by UI to show login screen)
var onAuthFailure: (() -> Unit)? = null

object ApiClient {
    // SharedPreferences key for storing the base URL
    private const val PREFS_NAME = "app_prefs"
    private const val KEY_BASE_URL = "base_url"
    
    // Default base URL (used if not configured)
    private const val DEFAULT_BASE_URL = "http://localhost:8080/"
    
    var BASE_URL: String = DEFAULT_BASE_URL
    
    private lateinit var context: Context

    fun init(context: Context) {
        this.context = context.applicationContext
        // Load saved base URL from SharedPreferences, or use default if not set
        loadBaseUrl()
    }

    /**
     * Loads the base URL from SharedPreferences. If no value is found, uses the default.
     */
    private fun loadBaseUrl() {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        BASE_URL = prefs.getString(KEY_BASE_URL, DEFAULT_BASE_URL) ?: DEFAULT_BASE_URL
    }

    /**
     * Saves the base URL to SharedPreferences.
     */
    private fun saveBaseUrl(url: String) {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        prefs.edit().putString(KEY_BASE_URL, url).apply()
    }

    // Changed from Level.BODY to Level.HEADERS to prevent binary data being logged in Logcat during uploads/downloads
    private val loggingInterceptor = HttpLoggingInterceptor().apply {
        level = HttpLoggingInterceptor.Level.HEADERS 
    }

    /**
     * Auth interceptor that adds Bearer token to every request.
     * If a 401 is received, it attempts auto-refresh via the refresh endpoint.
     * If refresh succeeds, it retries the original request with the new token.
     * If refresh fails or there's no refresh token stored, it clears auth state
     * and calls onAuthFailure callback (if set) so UI can show login screen.
     */
    private val authInterceptor = Interceptor { chain ->
        val originalRequest = chain.request()
        
        // Step 1: Try with existing access token first
        var response = attemptWithToken(originalRequest, chain)
        
        // If we got a 401 and have a refresh token, try to refresh
        if (response.code == 401 && getRefreshToken() != null) {
            val refreshToken = getRefreshToken()!!
            
            // Attempt refresh synchronously using Retrofit Call
            var refreshedToken: String? = null
            try {
                val call = apiServiceSync.refreshSync(
                    com.steadyphoto.sync.data.remote.dto.RefreshRequest(refresh_token = refreshToken)
                )
                val retrofitResponse = call.execute()
                
                if (retrofitResponse.isSuccessful && retrofitResponse.body() != null) {
                    refreshedToken = retrofitResponse.body()!!.accessToken
                    
                    // Store the new access token for future requests
                    storeAuthToken(refreshedToken)



                    
                    // Retry original request with new token
                    response = chain.proceed(
                        originalRequest.newBuilder()
                            .header("Authorization", "Bearer $refreshedToken")
                            .build()
                    )
                } else {
                    android.util.Log.e("ApiClient", "Refresh failed: ${retrofitResponse.code()} - ${retrofitResponse.message()}")
                }
            } catch (e: Exception) {
                android.util.Log.e("ApiClient", "Token refresh exception: ${e.message}")
            }
            
            // If refresh didn't work, clear auth and notify UI
            if (refreshedToken == null) {
                clearAuthTokenAndRefresh()
                
                // Dispatch callback to main thread (OkHttp runs on its own threads)
                android.os.Handler(android.os.Looper.getMainLooper()).post {
                    onAuthFailure?.invoke()
                }
                
                return@Interceptor originalResponse(401, "Unauthorized")
            }
        } else if (response.code == 401 && getRefreshToken() == null) {
            // No refresh token available - likely first login or full logout
            
            // Dispatch callback to main thread (OkHttp runs on its own threads)
            android.os.Handler(android.os.Looper.getMainLooper()).post {
                onAuthFailure?.invoke()
            }
            
            clearAuthTokenAndRefresh()
            
            return@Interceptor originalResponse(401, "Unauthorized")
        }
        
        response
    }

    /**
     * Attempts to make the request with the current access token.
     */
    private fun attemptWithToken(request: Request, chain: Interceptor.Chain): Response {
        val token = getAuthToken() ?: ""
        val newRequest = request.newBuilder()
            .header("Authorization", "Bearer $token")
            .build()
        
        return chain.proceed(newRequest)
    }

    /**
     * Returns a basic 401 response for when we want to return a controlled failure.
     */
    private fun originalResponse(code: Int, message: String): Response {
        val jsonMediaType = "application/json".toMediaType()
        return Response.Builder()
            .code(code)
            .message(message)
            .protocol(okhttp3.Protocol.HTTP_1_1)
            .request(Request.Builder().url("https://placeholder").build())
            .body(
                ("{\"error\":\"\$message\"}").toResponseBody(jsonMediaType)

            )
            .addHeader("Content-Type", "application/json")
            .build()
    }

    /**
     * Creates a synchronous Retrofit client for use in the OkHttp interceptor.
     * This is needed because suspend functions can't be called from an interceptor context.
     */
    private val syncRetrofitClient: Retrofit = Retrofit.Builder()
        .baseUrl(BASE_URL)
        .addConverterFactory(GsonConverterFactory.create())
        .build()

    /**
     * Synchronous API service for use in OkHttp interceptor (e.g., token refresh).
     */
    private val apiServiceSync: ApiService by lazy {
        syncRetrofitClient.create(ApiService::class.java)
    }

    // Mutable API service that can be recreated when base URL changes
    private var _apiService: ApiService? = null
    private var currentBaseUrlForApiService: String = DEFAULT_BASE_URL
    
    val apiService: ApiService get() {
        if (_apiService == null || currentBaseUrlForApiService != BASE_URL) {
            // Create a new Retrofit instance with the current base URL
            _apiService = Retrofit.Builder()
                .baseUrl(BASE_URL)
                .client(okHttpClient)
                .addConverterFactory(GsonConverterFactory.create())
                .build()
                .create(ApiService::class.java)
            currentBaseUrlForApiService = BASE_URL
        }
        return _apiService!!
    }

    /**
     * OkHttpClient instance with auth and logging interceptors.
     */
    private val okHttpClient: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .addInterceptor(loggingInterceptor)
            .addInterceptor(authInterceptor)
            .connectTimeout(30, TimeUnit.SECONDS)
            .readTimeout(30, TimeUnit.SECONDS)
            .writeTimeout(30, TimeUnit.SECONDS)
            .build()
    }

    /**
     * Retrieves the stored auth token from SharedPreferences.
     */
    fun getAuthToken(): String? {
        return context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .getString("auth_token", null)
    }

    /**
     * Stores an auth token in SharedPreferences.
     */
    fun storeAuthToken(token: String) {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putString("auth_token", token)
            .commit()  // Use commit() to ensure synchronous write before HomeScreen is shown
    }

    /**
     * Clears the stored auth token (logout).
     */
    fun clearAuthToken() {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .remove("auth_token")
            .apply()
    }

    /**
     * Retrieves the stored refresh token from SharedPreferences.
     */
    fun getRefreshToken(): String? {
        return context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .getString("refresh_token", null)
    }

    /**
     * Stores a refresh token in SharedPreferences.
     */
    fun storeRefreshToken(token: String) {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .putString("refresh_token", token)
            .apply()
    }

    /**
     * Clears the stored refresh token (logout).
     */
    fun clearRefreshToken() {
        context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
            .edit()
            .remove("refresh_token")
            .apply()
    }

    /**
     * Clears both auth token and refresh token. Used for logout flow.
     */
    fun clearAuthTokenAndRefresh() {
        clearAuthToken()
        clearRefreshToken()
    }

    /**
     * Checks if a user is currently logged in.
     */
    fun isLoggedIn(): Boolean = getAuthToken() != null

    /**
     * Updates the base URL (useful for testing with different servers).
     * Also saves it to SharedPreferences so it persists across app restarts.
     * This will also recreate the API service if needed.
     */
    fun updateBaseUrl(newUrl: String) {
        BASE_URL = newUrl.trim()
        saveBaseUrl(BASE_URL)
        // Invalidate the cached API service so it gets recreated with the new URL
        _apiService = null
        currentBaseUrlForApiService = DEFAULT_BASE_URL
    }

    /**
     * Returns the current base URL (for use in SetupScreen).
     */
    fun getBaseUrl(): String = BASE_URL

    /**
     * Checks if the user has configured a custom API URL.
     * If false, they haven't completed initial setup and should see the Setup screen.
     */
    fun isSetupComplete(): Boolean {
        val prefs = context.getSharedPreferences(PREFS_NAME, Context.MODE_PRIVATE)
        // A setup is complete if there's a stored base URL that differs from the default
        return prefs.getString(KEY_BASE_URL, DEFAULT_BASE_URL) != DEFAULT_BASE_URL
    }

    /**
     * Logs the current API endpoint for debugging purposes.
     */
    fun logApiEndpoint() {
        android.util.Log.d("ApiClient", "API Endpoint: $BASE_URL")
    }
}
