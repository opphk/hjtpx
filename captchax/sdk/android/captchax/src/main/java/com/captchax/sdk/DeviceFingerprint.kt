package com.captchax.sdk

import android.annotation.SuppressLint
import android.content.Context
import android.os.Build
import java.security.MessageDigest
import java.util.UUID

object DeviceFingerprint {
    
    private var cachedFingerprint: String? = null
    private var cachedDeviceInfo: Map<String, Any>? = null
    
    @SuppressLint("HardwareIds")
    fun generate(): String {
        cachedFingerprint?.let { return it }
        
        val deviceInfo = collect()
        val fingerprintBuilder = StringBuilder()
        
        fingerprintBuilder.append(deviceInfo["androidId"] ?: "")
        fingerprintBuilder.append(deviceInfo["model"] ?: "")
        fingerprintBuilder.append(deviceInfo["manufacturer"] ?: "")
        fingerprintBuilder.append(deviceInfo["sdkInt"] ?: "")
        fingerprintBuilder.append(deviceInfo["board"] ?: "")
        fingerprintBuilder.append(deviceInfo["hardware"] ?: "")
        
        val fingerprint = sha256(fingerprintBuilder.toString())
        cachedFingerprint = fingerprint
        
        return fingerprint
    }
    
    fun collect(): Map<String, Any> {
        cachedDeviceInfo?.let { return it }
        
        val info = mutableMapOf<String, Any>()
        
        info["androidId"] = getAndroidId()
        info["model"] = getDeviceModel()
        info["manufacturer"] = Build.MANUFACTURER
        info["brand"] = Build.BRAND
        info["device"] = Build.DEVICE
        info["product"] = Build.PRODUCT
        info["sdkInt"] = Build.VERSION.SDK_INT
        info["release"] = Build.VERSION.RELEASE
        info["board"] = Build.BOARD
        info["hardware"] = Build.HARDWARE
        info["fingerprint"] = Build.FINGERPRINT
        info["screenWidth"] = getScreenWidth()
        info["screenHeight"] = getScreenHeight()
        info["screenDensity"] = getScreenDensity()
        info["timezone"] = java.util.TimeZone.getDefault().id
        info["locale"] = java.util.Locale.getDefault().toString()
        info["hasPlayServices"] = hasPlayServices()
        
        cachedDeviceInfo = info
        
        return info
    }
    
    @SuppressLint("HardwareIds")
    private fun getAndroidId(): String {
        return try {
            android.provider.Settings.Secure.getString(
                android.content.ContextWrapper::class.java.classLoader
                    ?.let { Class.forName("android.app.AppGlobals").getMethod("getInitialApplication") }
                    ?.invoke(null) as? android.content.Context
                    ?: return "unknown",
                android.provider.Settings.Secure.ANDROID_ID
            ) ?: UUID.randomUUID().toString()
        } catch (e: Exception) {
            UUID.randomUUID().toString()
        }
    }
    
    fun getDeviceModel(): String {
        return "${Build.MANUFACTURER}_${Build.MODEL}".replace(" ", "_")
    }
    
    fun getManufacturer(): String {
        return Build.MANUFACTURER
    }
    
    fun getOsVersion(): String {
        return Build.VERSION.RELEASE
    }
    
    fun getSdkVersion(): Int {
        return Build.VERSION.SDK_INT
    }
    
    private fun getScreenWidth(): Int {
        return try {
            val resource = android.content.res.Resources.getSystem()
            val config = resource.configuration
            config.screenWidthDp
        } catch (e: Exception) {
            0
        }
    }
    
    private fun getScreenHeight(): Int {
        return try {
            val resource = android.content.res.Resources.getSystem()
            val config = resource.configuration
            config.screenHeightDp
        } catch (e: Exception) {
            0
        }
    }
    
    private fun getScreenDensity(): Float {
        return try {
            android.content.res.Resources.getSystem().displayMetrics.density
        } catch (e: Exception) {
            1.0f
        }
    }
    
    private fun hasPlayServices(): Boolean {
        return try {
            Class.forName("com.google.android.gms.common.GoogleApiAvailability")
            true
        } catch (e: ClassNotFoundException) {
            false
        }
    }
    
    fun isEmulator(): Boolean {
        return (Build.FINGERPRINT.startsWith("generic")
                || Build.FINGERPRINT.startsWith("unknown")
                || Build.MODEL.contains("google_sdk")
                || Build.MODEL.contains("Emulator")
                || Build.MODEL.contains("Android SDK built for x86")
                || Build.MANUFACTURER.contains("Genymotion")
                || (Build.BRAND.startsWith("generic") && Build.DEVICE.startsWith("generic"))
                || "google_sdk".equals(Build.PRODUCT))
    }
    
    fun isRooted(): Boolean {
        val paths = arrayOf(
            "/system/app/Superuser.apk",
            "/sbin/su",
            "/system/bin/su",
            "/system/xbin/su",
            "/data/local/xbin/su",
            "/data/local/bin/su",
            "/system/sd/xbin/su",
            "/system/bin/failsafe/su",
            "/data/local/su",
            "/su/bin/su"
        )
        
        return paths.any { path ->
            try {
                java.io.File(path).exists()
            } catch (e: Exception) {
                false
            }
        }
    }
    
    private fun sha256(input: String): String {
        val bytes = MessageDigest.getInstance("SHA-256").digest(input.toByteArray())
        return bytes.joinToString("") { "%02x".format(it) }
    }
    
    fun clearCache() {
        cachedFingerprint = null
        cachedDeviceInfo = null
    }
}
