package com.captchax.sdk

import android.annotation.SuppressLint
import android.content.Context
import android.graphics.Bitmap
import android.graphics.Color
import android.graphics.drawable.ColorDrawable
import android.util.AttributeSet
import android.view.LayoutInflater
import android.view.MotionEvent
import android.view.View
import android.view.ViewGroup
import android.widget.FrameLayout
import android.widget.ImageView
import android.widget.ProgressBar
import android.widget.TextView
import kotlinx.coroutines.CoroutineScope
import kotlinx.coroutines.Dispatchers
import kotlinx.coroutines.Job
import kotlinx.coroutines.SupervisorJob
import kotlinx.coroutines.cancel
import kotlinx.coroutines.launch
import kotlinx.coroutines.withContext

class CaptchaView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0
) : FrameLayout(context, attrs, defStyleAttr) {

    var listener: CaptchaViewListener? = null
    
    private var currentType: CaptchaType = CaptchaType.SLIDER
    private var currentToken: String? = null
    private val scope = CoroutineScope(SupervisorJob() + Dispatchers.Main)
    private var loadJob: Job? = null
    
    private val progressBar: ProgressBar
    private val imageView: ImageView
    private val sliderBar: View
    private val sliderThumb: View
    private val instructionText: TextView
    
    private var sliderStartX = 0f
    private var sliderMaxWidth = 0
    
    init {
        LayoutInflater.from(context).inflate(R.layout.captcha_view, this, true)
        
        progressBar = findViewById(R.id.progressBar)
        imageView = findViewById(R.id.captchaImage)
        sliderBar = findViewById(R.id.sliderBar)
        sliderThumb = findViewById(R.id.sliderThumb)
        instructionText = findViewById(R.id.instructionText)
        
        setupSliderInteraction()
    }
    
    private fun setupSliderInteraction() {
        sliderThumb.setOnTouchListener { _, event ->
            when (event.action) {
                MotionEvent.ACTION_DOWN -> {
                    sliderStartX = event.rawX - sliderThumb.x
                    true
                }
                MotionEvent.ACTION_MOVE -> {
                    val newX = event.rawX - sliderStartX
                    val maxX = sliderBar.width - sliderThumb.width
                    sliderThumb.x = newX.coerceIn(0f, maxX.toFloat())
                    true
                }
                MotionEvent.ACTION_UP -> {
                    verifySlider()
                    true
                }
                else -> false
            }
        }
        
        imageView.setOnClickListener { view ->
            if (currentType == CaptchaType.CLICK || currentType == CaptchaType.ICON) {
                handleClick(view, it)
            }
        }
    }
    
    private var clickPoints = mutableListOf<Pair<Float, Float>>()
    
    private fun handleClick(view: View, event: MotionEvent) {
        val location = IntArray(2)
        view.getLocationOnScreen(location)
        
        val x = event.x
        val y = event.y
        
        clickPoints.add(Pair(x, y))
        
        if (clickPoints.size >= 4) {
            verifyClicks()
        }
    }
    
    fun load(type: CaptchaType) {
        currentType = type
        listener?.onLoading()
        showLoading()
        
        loadJob?.cancel()
        loadJob = scope.launch {
            try {
                val cachedBitmap = CaptchaX.getImageCache().get(type.name)
                if (cachedBitmap != null) {
                    showImage(cachedBitmap)
                    listener?.onLoaded()
                    listener?.onReady()
                    return@launch
                }
                
                val bitmap = withContext(Dispatchers.IO) {
                    loadFromNetwork(type)
                }
                
                if (bitmap != null) {
                    CaptchaX.getImageCache().put(type.name, bitmap)
                    showImage(bitmap)
                    listener?.onLoaded()
                    listener?.onReady()
                } else {
                    listener?.onError(CaptchaError.ServerError("Failed to load captcha"))
                }
            } catch (e: Exception) {
                listener?.onError(CaptchaError.UnknownError(e.message ?: "Unknown error"))
            }
        }
    }
    
    private suspend fun loadFromNetwork(type: CaptchaType): Bitmap? {
        return try {
            val response = CaptchaX.getNetworkClient().request(
                endpoint = "/api/v1/captcha/${type.name.lowercase()}",
                method = okhttp3.HttpMethod.POST,
                params = mapOf(
                    "fingerprint" to DeviceFingerprint.generate(),
                    "width" to 300,
                    "height" to 200
                )
            )
            
            if (response.isSuccess && response.data != null) {
                currentToken = response.data["token"] as? String
                val imageUrl = response.data["image"] as? String
                if (imageUrl != null) {
                    loadImageFromUrl(imageUrl)
                } else null
            } else null
        } catch (e: Exception) {
            Logger.e("CaptchaView", "Load failed: ${e.message}")
            null
        }
    }
    
    private suspend fun loadImageFromUrl(url: String): Bitmap? {
        return try {
            val client = okhttp3.OkHttpClient()
            val request = okhttp3.Request.Builder().url(url).build()
            val response = client.newCall(request).execute()
            
            if (response.isSuccessful) {
                val bytes = response.body?.bytes()
                if (bytes != null) {
                    val bitmap = android.graphics.BitmapFactory.decodeByteArray(bytes, 0, bytes.size)
                    bitmap
                } else null
            } else null
        } catch (e: Exception) {
            Logger.e("CaptchaView", "Image load failed: ${e.message}")
            null
        }
    }
    
    private fun showLoading() {
        progressBar.visibility = View.VISIBLE
        imageView.visibility = View.INVISIBLE
    }
    
    private fun showImage(bitmap: Bitmap) {
        progressBar.visibility = View.GONE
        imageView.visibility = View.VISIBLE
        imageView.setImageBitmap(bitmap)
        
        updateInstruction()
    }
    
    private fun updateInstruction() {
        instructionText.text = when (currentType) {
            CaptchaType.SLIDER -> "拖动滑块完成拼图"
            CaptchaType.CLICK -> "请依次点击：${instructionText.text}"
            CaptchaType.ROTATE -> "旋转图片至正确角度"
            CaptchaType.PUZZLE -> "拖动滑块填充拼图"
            CaptchaType.TEXT -> "输入图中文字"
            CaptchaType.ICON -> "依次点击对应图标"
        }
    }
    
    private fun verifySlider() {
        val token = currentToken ?: return
        val distance = (sliderThumb.x / (sliderBar.width - sliderThumb.width) * 100).toInt()
        
        scope.launch {
            try {
                val response = withContext(Dispatchers.IO) {
                    CaptchaX.getNetworkClient().request(
                        endpoint = "/api/v1/captcha/slider/verify",
                        method = okhttp3.HttpMethod.POST,
                        params = mapOf(
                            "token" to token,
                            "distance" to distance,
                            "fingerprint" to DeviceFingerprint.generate()
                        )
                    )
                }
                
                if (response.isSuccess) {
                    val resultToken = response.data?.get("token") as? String ?: token
                    listener?.onSuccess(resultToken)
                } else {
                    reset()
                    listener?.onError(CaptchaError.ValidationError("验证失败"))
                }
            } catch (e: Exception) {
                listener?.onError(CaptchaError.UnknownError(e.message ?: "Verification failed"))
            }
        }
    }
    
    private fun verifyClicks() {
        val token = currentToken ?: return
        
        scope.launch {
            try {
                val response = withContext(Dispatchers.IO) {
                    CaptchaX.getNetworkClient().request(
                        endpoint = "/api/v1/captcha/click/verify",
                        method = okhttp3.HttpMethod.POST,
                        params = mapOf(
                            "token" to token,
                            "points" to clickPoints.map { mapOf("x" to it.first, "y" to it.second) },
                            "fingerprint" to DeviceFingerprint.generate()
                        )
                    )
                }
                
                if (response.isSuccess) {
                    val resultToken = response.data?.get("token") as? String ?: token
                    listener?.onSuccess(resultToken)
                } else {
                    clickPoints.clear()
                    reset()
                    listener?.onError(CaptchaError.ValidationError("验证失败"))
                }
            } catch (e: Exception) {
                listener?.onError(CaptchaError.UnknownError(e.message ?: "Verification failed"))
            }
        }
    }
    
    fun reset() {
        sliderThumb.x = 0f
        clickPoints.clear()
        currentToken = null
        
        when (currentType) {
            CaptchaType.SLIDER, CaptchaType.PUZZLE -> sliderThumb.visibility = View.VISIBLE
            CaptchaType.CLICK, CaptchaType.ICON -> clickPoints.clear()
            else -> {}
        }
        
        load(currentType)
    }
    
    fun destroy() {
        loadJob?.cancel()
        scope.cancel()
        listener = null
    }
}
