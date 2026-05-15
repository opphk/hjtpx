package com.captchax.sdk

sealed class CaptchaError(
    message: String,
    val code: String? = null,
    val details: Map<String, Any>? = null
) : Exception(message) {
    class NetworkError(message: String, details: Map<String, Any>? = null) 
        : CaptchaError(message, "NETWORK_ERROR", details)
    
    class ServerError(message: String, code: String? = null, details: Map<String, Any>? = null) 
        : CaptchaError(message, code ?: "SERVER_ERROR", details)
    
    class ValidationError(message: String, details: Map<String, Any>? = null) 
        : CaptchaError(message, "VALIDATION_ERROR", details)
    
    class TimeoutError(message: String, details: Map<String, Any>? = null) 
        : CaptchaError(message, "TIMEOUT_ERROR", details)
    
    class CancelledError(message: String = "Verification cancelled") 
        : CaptchaError(message, "CANCELLED")
    
    class UnknownError(message: String, details: Map<String, Any>? = null) 
        : CaptchaError(message, "UNKNOWN_ERROR", details)
    
    companion object {
        fun fromCode(code: String, message: String, details: Map<String, Any>? = null): CaptchaError {
            return when (code) {
                "NETWORK_ERROR" -> NetworkError(message, details)
                "TIMEOUT" -> TimeoutError(message, details)
                "INVALID_PARAMS" -> ValidationError(message, details)
                "SERVER_ERROR" -> ServerError(message, code, details)
                else -> UnknownError(message, details)
            }
        }
    }
}

interface CaptchaListener {
    fun onSuccess(token: String)
    fun onError(error: CaptchaError)
    fun onClose()
}

interface CaptchaViewListener {
    fun onSuccess(token: String)
    fun onError(error: CaptchaError)
    fun onClose()
    fun onReady()
    fun onLoading()
    fun onLoaded()
}

interface CaptchaVerifyCallback {
    fun onSuccess(token: String)
    fun onError(error: CaptchaError)
}

@JvmSuppressWildcards
interface CaptchaResultCallback {
    fun onResult(result: Result<String>)
}
