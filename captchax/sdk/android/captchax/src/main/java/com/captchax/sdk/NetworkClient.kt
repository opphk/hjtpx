package com.captchax.sdk

import com.captchax.sdk.util.Logger
import okhttp3.Call
import okhttp3.Callback
import okhttp3.HttpUrl.Companion.toHttpUrl
import okhttp3.MediaType.Companion.toMediaType
import okhttp3.OkHttpClient
import okhttp3.Request
import okhttp3.RequestBody.Companion.toRequestBody
import okhttp3.Response
import okhttp3.ResponseBody
import org.json.JSONObject
import java.io.IOException
import java.security.MessageDigest
import java.util.concurrent.TimeUnit
import javax.crypto.Mac
import javax.crypto.spec.SecretKeySpec
import kotlin.coroutines.resume
import kotlin.coroutines.resumeWithException
import kotlin.coroutines.suspendCoroutine

class NetworkClient(private val config: CaptchaConfig) {
    
    private val client: OkHttpClient by lazy {
        OkHttpClient.Builder()
            .connectTimeout(config.timeout, TimeUnit.MILLISECONDS)
            .readTimeout(config.timeout, TimeUnit.MILLISECONDS)
            .writeTimeout(config.timeout, TimeUnit.MILLISECONDS)
            .build()
    }
    
    data class NetworkResponse(
        val isSuccess: Boolean,
        val data: Map<String, Any>?,
        val errorCode: String?,
        val errorMessage: String?
    )
    
    data class UploadResponse(
        val isSuccess: Boolean,
        val url: String?,
        val errorCode: String?,
        val errorMessage: String?
    )
    
    suspend fun request(
        endpoint: String,
        method: okhttp3.HttpMethod,
        params: Map<String, Any>?
    ): NetworkResponse = suspendCoroutine { continuation ->
        try {
            val url = "${config.serverUrl}$endpoint".toHttpUrl()
            val body = params?.let { createRequestBody(it) }
            
            val requestBuilder = Request.Builder()
                .url(url)
                .apply {
                    when (method) {
                        okhttp3.HttpMethod.GET -> get()
                        okhttp3.HttpMethod.POST -> post(body ?: "".toRequestBody())
                        okhttp3.HttpMethod.PUT -> put(body ?: "".toRequestBody())
                        okhttp3.HttpMethod.DELETE -> delete(body ?: "".toRequestBody())
                        else -> get()
                    }
                }
            
            signRequest(requestBuilder, endpoint, params ?: emptyMap())
            
            client.newCall(requestBuilder.build()).enqueue(object : Callback {
                override fun onFailure(call: Call, e: IOException) {
                    Logger.e("NetworkClient", "Request failed: ${e.message}")
                    continuation.resume(
                        NetworkResponse(
                            isSuccess = false,
                            data = null,
                            errorCode = "NETWORK_ERROR",
                            errorMessage = e.message
                        )
                    )
                }
                
                override fun onResponse(call: Call, response: Response) {
                    try {
                        val responseBody = response.body?.string()
                        val jsonResponse = responseBody?.let { parseJson(it) }
                        
                        if (response.isSuccessful && jsonResponse != null) {
                            val data = jsonResponse.optJSONObject("data")?.let { jsonToMap(it) }
                                ?: jsonResponse.optJSONArray("data")?.let { array ->
                                    mapOf("items" to (0 until array.length()).map { array.get(it) })
                                }
                            
                            continuation.resume(
                                NetworkResponse(
                                    isSuccess = true,
                                    data = data,
                                    errorCode = null,
                                    errorMessage = null
                                )
                            )
                        } else {
                            continuation.resume(
                                NetworkResponse(
                                    isSuccess = false,
                                    data = null,
                                    errorCode = jsonResponse?.optString("code", "SERVER_ERROR"),
                                    errorMessage = jsonResponse?.optString("message", "Unknown error")
                                )
                            )
                        }
                    } catch (e: Exception) {
                        Logger.e("NetworkClient", "Parse error: ${e.message}")
                        continuation.resume(
                            NetworkResponse(
                                isSuccess = false,
                                data = null,
                                errorCode = "PARSE_ERROR",
                                errorMessage = e.message
                            )
                        )
                    }
                }
            })
        } catch (e: Exception) {
            Logger.e("NetworkClient", "Request error: ${e.message}")
            continuation.resume(
                NetworkResponse(
                    isSuccess = false,
                    data = null,
                    errorCode = "REQUEST_ERROR",
                    errorMessage = e.message
                )
            )
        }
    }
    
    suspend fun uploadImage(imageBytes: ByteArray): UploadResponse = suspendCoroutine { continuation ->
        val requestBody = okhttp3.MultipartBody.Builder()
            .setType(okhttp3.MultipartBody.FORM)
            .addFormDataPart(
                "image",
                "captcha.png",
                imageBytes.toRequestBody("image/png".toMediaType())
            )
            .build()
        
        val request = Request.Builder()
            .url("${config.serverUrl}/api/v1/captcha/upload")
            .post(requestBody)
            .build()
        
        client.newCall(request).enqueue(object : Callback {
            override fun onFailure(call: Call, e: IOException) {
                continuation.resume(
                    UploadResponse(
                        isSuccess = false,
                        url = null,
                        errorCode = "UPLOAD_ERROR",
                        errorMessage = e.message
                    )
                )
            }
            
            override fun onResponse(call: Call, response: Response) {
                try {
                    val responseBody = response.body?.string()
                    val jsonResponse = responseBody?.let { parseJson(it) }
                    
                    if (response.isSuccessful && jsonResponse != null) {
                        continuation.resume(
                            UploadResponse(
                                isSuccess = true,
                                url = jsonResponse.optString("url"),
                                errorCode = null,
                                errorMessage = null
                            )
                        )
                    } else {
                        continuation.resume(
                            UploadResponse(
                                isSuccess = false,
                                url = null,
                                errorCode = jsonResponse?.optString("code", "UPLOAD_ERROR"),
                                errorMessage = jsonResponse?.optString("message", "Upload failed")
                            )
                        )
                    }
                } catch (e: Exception) {
                    continuation.resume(
                        UploadResponse(
                            isSuccess = false,
                            url = null,
                            errorCode = "PARSE_ERROR",
                            errorMessage = e.message
                        )
                    )
                }
            }
        })
    }
    
    private fun signRequest(requestBuilder: Request.Builder, endpoint: String, params: Map<String, Any>) {
        val timestamp = System.currentTimeMillis().toString()
        val nonce = java.util.UUID.randomUUID().toString()
        
        val signString = buildString {
            append(config.apiKey)
            append(timestamp)
            append(nonce)
            params.entries.sortedBy { it.key }.forEach { (key, value) ->
                append(key).append(value)
            }
        }
        
        val signature = hmacSha256(signString, config.apiSecret)
        
        requestBuilder
            .addHeader("X-API-Key", config.apiKey)
            .addHeader("X-Timestamp", timestamp)
            .addHeader("X-Nonce", nonce)
            .addHeader("X-Signature", signature)
            .addHeader("Content-Type", "application/json")
            .addHeader("Accept", "application/json")
    }
    
    private fun hmacSha256(data: String, key: String): String {
        val algorithm = "HmacSHA256"
        val secretKey = SecretKeySpec(key.toByteArray(), algorithm)
        val mac = Mac.getInstance(algorithm)
        mac.init(secretKey)
        val hash = mac.doFinal(data.toByteArray())
        return hash.joinToString("") { "%02x".format(it) }
    }
    
    private fun createRequestBody(params: Map<String, Any>): okhttp3.RequestBody {
        val json = JSONObject(params).toString()
        return json.toRequestBody("application/json".toMediaType())
    }
    
    private fun parseJson(jsonString: String): JSONObject? {
        return try {
            JSONObject(jsonString)
        } catch (e: Exception) {
            Logger.e("NetworkClient", "JSON parse error: ${e.message}")
            null
        }
    }
    
    private fun jsonToMap(jsonObject: JSONObject): Map<String, Any> {
        val map = mutableMapOf<String, Any>()
        jsonObject.keys().forEach { key ->
            val value = jsonObject.get(key)
            map[key] = when (value) {
                is JSONObject -> jsonToMap(value)
                is org.json.JSONArray -> (0 until value.length()).map { value.get(it) }
                else -> value
            }
        }
        return map
    }
    
    private fun ByteArray.toRequestBody(contentType: okhttp3.MediaType): okhttp3.RequestBody {
        return this.toRequestBody(contentType)
    }
}
