package com.captchax.sdk.util

import android.util.Log

object Logger {
    
    private const val TAG_PREFIX = "CaptchaX"
    
    var isDebugEnabled = true
    var isLogEnabled = true
    
    fun d(tag: String, message: String) {
        if (isLogEnabled && isDebugEnabled) {
            Log.d("$TAG_PREFIX/$tag", message)
        }
    }
    
    fun i(tag: String, message: String) {
        if (isLogEnabled) {
            Log.i("$TAG_PREFIX/$tag", message)
        }
    }
    
    fun w(tag: String, message: String) {
        if (isLogEnabled) {
            Log.w("$TAG_PREFIX/$tag", message)
        }
    }
    
    fun e(tag: String, message: String, throwable: Throwable? = null) {
        if (isLogEnabled) {
            if (throwable != null) {
                Log.e("$TAG_PREFIX/$tag", message, throwable)
            } else {
                Log.e("$TAG_PREFIX/$tag", message)
            }
        }
    }
    
    fun v(tag: String, message: String) {
        if (isLogEnabled && isDebugEnabled) {
            Log.v("$TAG_PREFIX/$tag", message)
        }
    }
    
    fun wtf(tag: String, message: String, throwable: Throwable? = null) {
        if (isLogEnabled) {
            Log.wtf("$TAG_PREFIX/$tag", message, throwable)
        }
    }
    
    fun log(level: LogLevel, tag: String, message: String) {
        if (!isLogEnabled) return
        
        when (level) {
            LogLevel.VERBOSE -> v(tag, message)
            LogLevel.DEBUG -> d(tag, message)
            LogLevel.INFO -> i(tag, message)
            LogLevel.WARN -> w(tag, message)
            LogLevel.ERROR -> e(tag, message)
            LogLevel.ASSERT -> wtf(tag, message)
        }
    }
    
    enum class LogLevel {
        VERBOSE, DEBUG, INFO, WARN, ERROR, ASSERT
    }
    
    companion object {
        fun enableDebug(enabled: Boolean = true) {
            isDebugEnabled = enabled
        }
        
        fun disableLogging() {
            isLogEnabled = false
        }
        
        fun enableLogging() {
            isLogEnabled = true
        }
    }
}
