package com.captchax.sdk

import android.app.Activity
import android.content.Context
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

object CaptchaX {
    private var _config: CaptchaConfig? = null
    private val networkClient: NetworkClient by lazy { NetworkClient(_config!!) }
    private val imageCache: ImageCache by lazy { ImageCache() }
    
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    
    val config: CaptchaConfig
        get() = _config ?: throw IllegalStateException("CaptchaX not initialized. Call initialize() first.")

    var listener: CaptchaListener? = null

    @JvmOverloads
    fun initialize(
        context: Context,
        apiKey: String,
        apiSecret: String,
        serverUrl: String = "https://api.captchax.com"
    ) {
        _config = CaptchaConfig(
            apiKey = apiKey,
            apiSecret = apiSecret,
            serverUrl = serverUrl
        )
    }

    fun initialize(context: Context, config: CaptchaConfig) {
        _config = config
    }

    fun verify(activity: Activity, scene: String, callback: (Result<String>) -> Unit) {
        verify(scene, object : CaptchaVerifyCallback {
            override fun onSuccess(token: String) {
                callback(Result.success(token))
                listener?.onSuccess(token)
            }

            override fun onError(error: CaptchaError) {
                callback(Result.failure(error))
                listener?.onError(error)
            }
        })
    }

    fun verify(scene: String, callback: CaptchaVerifyCallback) {
        scope.launch {
            try {
                val response = withContext(Dispatchers.IO) {
                    networkClient.request(
                        endpoint = "/api/v1/captcha/request",
                        method = okhttp3.HttpMethod.POST,
                        params = mapOf(
                            "scene" to scene,
                            "fingerprint" to DeviceFingerprint.generate(),
                            "sdk" to "android",
                            "version" to BuildConfig.VERSION_NAME
                        )
                    )
                }

                if (response.isSuccess) {
                    callback.onSuccess(response.data?.get("token") as? String ?: "")
                } else {
                    callback.onError(
                        CaptchaError.fromCode(
                            response.errorCode ?: "UNKNOWN_ERROR",
                            response.errorMessage ?: "Unknown error"
                        )
                    )
                }
            } catch (e: Exception) {
                callback.onError(CaptchaError.UnknownError(e.message ?: "Unknown error"))
            }
        }
    }

    fun preload(scene: String) {
        if (_config?.preloadEnabled != true) return
        
        scope.launch {
            withContext(Dispatchers.IO) {
                try {
                    networkClient.request(
                        endpoint = "/api/v1/captcha/preload",
                        method = okhttp3.HttpMethod.POST,
                        params = mapOf(
                            "scene" to scene,
                            "fingerprint" to DeviceFingerprint.generate()
                        )
                    )
                } catch (e: Exception) {
                    Logger.e("CaptchaX", "Preload failed: ${e.message}")
                }
            }
        }
    }

    fun getCaptchaView(activity: Activity, type: CaptchaType, listener: CaptchaViewListener): CaptchaView {
        return CaptchaView(activity).apply {
            this.listener = listener
            load(type)
        }
    }

    fun destroy() {
        imageCache.clear()
        listener = null
    }

    internal fun getImageCache(): ImageCache = imageCache
    
    internal fun getNetworkClient(): NetworkClient = networkClient
}

object BuildConfig {
    const val VERSION_NAME = "1.0.0"
    const val VERSION_CODE = 1
}
