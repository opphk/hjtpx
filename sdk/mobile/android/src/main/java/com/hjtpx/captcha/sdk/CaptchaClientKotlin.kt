package com.hjtpx.captcha.sdk

import android.content.Context
import android.graphics.Bitmap
import android.graphics.BitmapFactory
import android.os.Handler
import android.os.Looper
import android.util.Log
import com.bumptech.glide.Glide
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.withContext
import java.io.IOException
import java.net.URL
import java.util.concurrent.TimeUnit

class CaptchaClient(
    private val context: Context,
    private val baseURL: String,
    private val appId: String,
    private val appSecret: String
) {
    private val okHttpClient = okhttp3.OkHttpClient.Builder()
        .connectTimeout(10, TimeUnit.SECONDS)
        .readTimeout(30, TimeUnit.SECONDS)
        .writeTimeout(30, TimeUnit.SECONDS)
        .retryOnConnectionFailure(true)
        .build()

    private val mainHandler = Handler(Looper.getMainLooper())

    interface CaptchaCallback<T> {
        fun onSuccess(result: T)
        fun onError(error: String)
    }

    data class SliderCaptchaResult(
        val sessionId: String,
        val imageUrl: String,
        val puzzleUrl: String,
        val secretY: Int,
        val imageWidth: Int,
        val imageHeight: Int
    )

    data class ClickCaptchaResult(
        val sessionId: String,
        val imageUrl: String,
        val hint: String,
        val hintOrder: List<Int>,
        val maxPoints: Int,
        val mode: String
    )

    data class VerifyResult(
        val success: Boolean,
        val message: String,
        val remainingAttempts: Int,
        val riskScore: Double
    )

    data class TrajectoryPoint(
        val x: Int,
        val y: Int,
        val timestamp: Long
    )

    fun generateSliderCaptcha(
        width: Int = 320,
        height: Int = 160,
        tolerance: Int = 8,
        callback: CaptchaCallback<SliderCaptchaResult>
    ) {
        Thread {
            try {
                val url = "$baseURL/api/v1/captcha/slider?width=$width&height=$height&tolerance=$tolerance"
                val request = okhttp3.Request.Builder()
                    .url(url)
                    .get()
                    .addHeader("X-API-Key", appId)
                    .build()

                val response = okHttpClient.newCall(request).execute()
                val body = response.body?.string()

                if (response.isSuccessful && body != null) {
                    val json = JSONObject(body)
                    if (json.getInt("code") == 0) {
                        val data = json.getJSONObject("data")
                        val result = SliderCaptchaResult(
                            sessionId = data.getString("session_id"),
                            imageUrl = data.getString("image_url"),
                            puzzleUrl = data.getString("puzzle_url"),
                            secretY = data.optInt("secret_y", 0),
                            imageWidth = data.optInt("image_width", width),
                            imageHeight = data.optInt("image_height", height)
                        )
                        mainHandler.post { callback.onSuccess(result) }
                    } else {
                        mainHandler.post { callback.onError(json.getString("message")) }
                    }
                } else {
                    mainHandler.post { callback.onError("Request failed: ${response.code}") }
                }
            } catch (e: Exception) {
                mainHandler.post { callback.onError(e.message ?: "Unknown error") }
            }
        }.start()
    }

    fun verifySliderCaptcha(
        sessionId: String,
        x: Int,
        y: Int? = null,
        trajectory: List<TrajectoryPoint>? = null,
        callback: CaptchaCallback<VerifyResult>
    ) {
        Thread {
            try {
                val url = "$baseURL/api/v1/captcha/verify"
                val jsonBody = JSONObject().apply {
                    put("session_id", sessionId)
                    put("type", "slider")
                    put("x", x)
                    y?.let { put("y", it) }
                    trajectory?.let { points ->
                        put("trajectory", JSONArray().apply {
                            points.forEach { point ->
                                put(JSONObject().apply {
                                    put("x", point.x)
                                    put("y", point.y)
                                    put("t", point.timestamp)
                                })
                            }
                        })
                    }
                }

                val requestBody = okhttp3.MediaType.parse("application/json")
                    ?.let { okhttp3.RequestBody.create(it, jsonBody.toString()) }

                val request = okhttp3.Request.Builder()
                    .url(url)
                    .post(requestBody!!)
                    .addHeader("Content-Type", "application/json")
                    .addHeader("X-API-Key", appId)
                    .build()

                val response = okHttpClient.newCall(request).execute()
                val body = response.body?.string()

                if (response.isSuccessful && body != null) {
                    val json = JSONObject(body)
                    if (json.getInt("code") == 0) {
                        val data = json.getJSONObject("data")
                        val result = VerifyResult(
                            success = data.getBoolean("success"),
                            message = data.getString("message"),
                            remainingAttempts = data.optInt("remaining_attempts", 0),
                            riskScore = data.optDouble("risk_score", 0.0)
                        )
                        mainHandler.post { callback.onSuccess(result) }
                    } else {
                        mainHandler.post { callback.onError(json.getString("message")) }
                    }
                } else {
                    mainHandler.post { callback.onError("Request failed: ${response.code}") }
                }
            } catch (e: Exception) {
                mainHandler.post { callback.onError(e.message ?: "Unknown error") }
            }
        }.start()
    }

    fun generateClickCaptcha(
        mode: String = "number",
        maxPoints: Int = 3,
        allowShuffle: Boolean = true,
        callback: CaptchaCallback<ClickCaptchaResult>
    ) {
        Thread {
            try {
                val url = "$baseURL/api/v1/captcha/click?mode=$mode&points=$maxPoints&shuffle=$allowShuffle"
                val request = okhttp3.Request.Builder()
                    .url(url)
                    .get()
                    .addHeader("X-API-Key", appId)
                    .build()

                val response = okHttpClient.newCall(request).execute()
                val body = response.body?.string()

                if (response.isSuccessful && body != null) {
                    val json = JSONObject(body)
                    if (json.getInt("code") == 0) {
                        val data = json.getJSONObject("data")
                        val result = ClickCaptchaResult(
                            sessionId = data.getString("session_id"),
                            imageUrl = data.getString("image_url"),
                            hint = data.getString("hint"),
                            hintOrder = data.getJSONArray("hint_order").let { arr ->
                                (0 until arr.length()).map { arr.getInt(it) }
                            },
                            maxPoints = data.getInt("max_points"),
                            mode = data.getString("mode")
                        )
                        mainHandler.post { callback.onSuccess(result) }
                    } else {
                        mainHandler.post { callback.onError(json.getString("message")) }
                    }
                } else {
                    mainHandler.post { callback.onError("Request failed: ${response.code}") }
                }
            } catch (e: Exception) {
                mainHandler.post { callback.onError(e.message ?: "Unknown error") }
            }
        }.start()
    }

    fun verifyClickCaptcha(
        sessionId: String,
        points: List<List<Int>>,
        clickSequence: List<Int>? = null,
        callback: CaptchaCallback<VerifyResult>
    ) {
        Thread {
            try {
                val url = "$baseURL/api/v1/captcha/verify"
                val jsonBody = JSONObject().apply {
                    put("session_id", sessionId)
                    put("type", "click")
                    put("points", JSONArray().apply {
                        points.forEach { point ->
                            put(JSONArray().apply {
                                point.forEach { put(it) }
                            })
                        }
                    })
                    clickSequence?.let {
                        put("click_sequence", JSONArray().apply {
                            it.forEach { put(it) }
                        })
                    }
                }

                val requestBody = okhttp3.MediaType.parse("application/json")
                    ?.let { okhttp3.RequestBody.create(it, jsonBody.toString()) }

                val request = okhttp3.Request.Builder()
                    .url(url)
                    .post(requestBody!!)
                    .addHeader("Content-Type", "application/json")
                    .addHeader("X-API-Key", appId)
                    .build()

                val response = okHttpClient.newCall(request).execute()
                val body = response.body?.string()

                if (response.isSuccessful && body != null) {
                    val json = JSONObject(body)
                    if (json.getInt("code") == 0) {
                        val data = json.getJSONObject("data")
                        val result = VerifyResult(
                            success = data.getBoolean("success"),
                            message = data.getString("message"),
                            remainingAttempts = data.optInt("remaining_attempts", 0),
                            riskScore = data.optDouble("risk_score", 0.0)
                        )
                        mainHandler.post { callback.onSuccess(result) }
                    } else {
                        mainHandler.post { callback.onError(json.getString("message")) }
                    }
                } else {
                    mainHandler.post { callback.onError("Request failed: ${response.code}") }
                }
            } catch (e: Exception) {
                mainHandler.post { callback.onError(e.message ?: "Unknown error") }
            }
        }.start()
    }
}
