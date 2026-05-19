package com.hjtpx.captcha

import android.content.Context
import okhttp3.*
import org.json.JSONObject
import java.io.IOException
import java.util.concurrent.TimeUnit

class HjtpxClient private constructor(context: Context) {
    private val context: Context = context.applicationContext
    private var apiKey: String = ""
    private var apiSecret: String = ""
    private var serverUrl: String = "https://your-domain.com"
    private var timeout: Long = 30
    private var language: String = "en-US"
    
    private val client: OkHttpClient = OkHttpClient.Builder()
        .connectTimeout(timeout, TimeUnit.SECONDS)
        .readTimeout(timeout, TimeUnit.SECONDS)
        .writeTimeout(timeout, TimeUnit.SECONDS)
        .build()
    
    companion object {
        @Volatile
        private var instance: HjtpxClient? = null
        
        fun getInstance(context: Context): HjtpxClient {
            return instance ?: synchronized(this) {
                instance ?: HjtpxClient(context).also { instance = it }
            }
        }
    }
    
    fun configure(apiKey: String, apiSecret: String, serverUrl: String) {
        this.apiKey = apiKey
        this.apiSecret = apiSecret
        this.serverUrl = serverUrl
    }
    
    fun setLanguage(language: String) {
        this.language = language
    }
    
    fun setTimeout(timeout: Long) {
        this.timeout = timeout
    }
    
    fun getCaptcha(
        type: CaptchaType,
        appId: String,
        callback: CaptchaCallback
    ) {
        val url = "$serverUrl/api/v1/captcha/get"
        
        val json = JSONObject().apply {
            put("captcha_type", type.value)
            put("app_id", appId)
            put("language", language)
            put("timestamp", System.currentTimeMillis())
        }
        
        val body = RequestBody.create(
            MediaType.parse("application/json; charset=utf-8"),
            json.toString()
        )
        
        val request = Request.Builder()
            .url(url)
            .post(body)
            .addHeader("Content-Type", "application/json")
            .addHeader("X-API-Key", apiKey)
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e)
            }
            
            @Throws(IOException::class)
            override fun onResponse(call: Call, response: Response) {
                if (!response.isSuccessful) {
                    callback.onError(Exception("Unexpected code $response"))
                    return
                }
                
                response.body()?.let { body ->
                    val jsonResponse = JSONObject(body.string())
                    val captchaResponse = CaptchaResponse(
                        captchaId = jsonResponse.optString("captcha_id", ""),
                        captchaType = jsonResponse.optString("captcha_type", ""),
                        code = jsonResponse.optInt("code", -1),
                        message = jsonResponse.optString("message", ""),
                        data = parseCaptchaData(jsonResponse.optJSONObject("data"))
                    )
                    callback.onSuccess(captchaResponse)
                } ?: callback.onError(Exception("No data received"))
            }
        })
    }
    
    fun verifyCaptcha(
        captchaId: String,
        token: String,
        appId: String,
        callback: VerifyCallback
    ) {
        val url = "$serverUrl/api/v1/captcha/verify"
        
        val json = JSONObject().apply {
            put("captcha_id", captchaId)
            put("token", token)
            put("app_id", appId)
            put("timestamp", System.currentTimeMillis())
        }
        
        val body = RequestBody.create(
            MediaType.parse("application/json; charset=utf-8"),
            json.toString()
        )
        
        val request = Request.Builder()
            .url(url)
            .post(body)
            .addHeader("Content-Type", "application/json")
            .addHeader("X-API-Key", apiKey)
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                callback.onError(e)
            }
            
            @Throws(IOException::class)
            override fun onResponse(call: Call, response: Response) {
                if (!response.isSuccessful) {
                    callback.onError(Exception("Unexpected code $response"))
                    return
                }
                
                response.body()?.let { body ->
                    val jsonResponse = JSONObject(body.string())
                    val verifyResponse = VerifyResponse(
                        success = jsonResponse.optBoolean("success", false),
                        captchaId = jsonResponse.optString("captcha_id", ""),
                        score = if (jsonResponse.has("score")) jsonResponse.getDouble("score") else null,
                        message = jsonResponse.optString("message", ""),
                        verifyId = jsonResponse.optString("verify_id", "")
                    )
                    callback.onSuccess(verifyResponse)
                } ?: callback.onError(Exception("No data received"))
            }
        })
    }
    
    fun reportResult(captchaId: String, result: Boolean, appId: String) {
        val url = "$serverUrl/api/v1/captcha/report"
        
        val json = JSONObject().apply {
            put("captcha_id", captchaId)
            put("result", result)
            put("app_id", appId)
            put("timestamp", System.currentTimeMillis())
        }
        
        val body = RequestBody.create(
            MediaType.parse("application/json; charset=utf-8"),
            json.toString()
        )
        
        val request = Request.Builder()
            .url(url)
            .post(body)
            .addHeader("Content-Type", "application/json")
            .addHeader("X-API-Key", apiKey)
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                e.printStackTrace()
            }
            
            @Throws(IOException::class)
            override fun onResponse(call: Call, response: Response) {
                // Async report, no need to handle response
            }
        })
    }
    
    private fun parseCaptchaData(json: JSONObject?): CaptchaData? {
        if (json == null) return null
        
        return CaptchaData(
            backgroundImage = json.optString("background_image", null),
            sliderImage = json.optString("slider_image", null),
            targetPosition = if (json.has("target_position")) json.getInt("target_position") else null,
            hintText = json.optString("hint_text", null)
        )
    }
}

enum class CaptchaType(val value: String) {
    SLIDER("slider"),
    CLICK("click"),
    ROTATE("rotate"),
    VOICE("voice"),
    GESTURE("gesture")
}

data class CaptchaResponse(
    val captchaId: String,
    val captchaType: String,
    val code: Int,
    val message: String,
    val data: CaptchaData?
)

data class CaptchaData(
    val backgroundImage: String?,
    val sliderImage: String?,
    val targetPosition: Int?,
    val hintText: String?
)

data class VerifyResponse(
    val success: Boolean,
    val captchaId: String,
    val score: Double?,
    val message: String,
    val verifyId: String?
)

interface CaptchaCallback {
    fun onSuccess(response: CaptchaResponse)
    fun onError(error: Exception)
}

interface VerifyCallback {
    fun onSuccess(response: VerifyResponse)
    fun onError(error: Exception)
}
