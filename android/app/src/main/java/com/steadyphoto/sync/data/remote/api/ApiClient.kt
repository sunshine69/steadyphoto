package com.steadyphoto.sync.data.remote.api

import android.content.Context
import okhttp3.Interceptor
import okhttp3.OkHttpClient
import okhttp3.logging.HttpLoggingInterceptor
import retrofit2.Retrofit
import retrofit2.converter.gson.GsonConverterFactory
import java.util.concurrent.TimeUnit

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

    private val loggingInterceptor = HttpLoggingInterceptor().apply {
        level = HttpLoggingInterceptor.Level.BODY // Always log in debug builds
    }

    private val authInterceptor = Interceptor { chain ->
        val originalRequest = chain.request()
        val token = getAuthToken() ?: ""
        
        val requestBuilder = originalRequest.newBuilder()
            .header("Authorization", "Bearer $token")
        
        chain.proceed(requestBuilder.build())
    }

    private val okHttpClient = OkHttpClient.Builder()
        .addInterceptor(loggingInterceptor)
        .addInterceptor(authInterceptor)
        .connectTimeout(30, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .build()

    val apiService: ApiService by lazy {
        Retrofit.Builder()
            .baseUrl(BASE_URL)
            .client(okHttpClient)
            .addConverterFactory(GsonConverterFactory.create())
            .build()
            .create(ApiService::class.java)
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
            .apply()
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
     * Checks if a user is currently logged in.
     */
    fun isLoggedIn(): Boolean = getAuthToken() != null

    /**
     * Updates the base URL (useful for testing with different servers).
     * Also saves it to SharedPreferences so it persists across app restarts.
     */
    fun updateBaseUrl(newUrl: String) {
        BASE_URL = newUrl.trim()
        saveBaseUrl(BASE_URL)
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
