package com.hjtpx.captcha

import android.annotation.SuppressLint
import android.content.Context
import android.graphics.Color
import android.os.Build
import android.util.AttributeSet
import android.view.View
import android.webkit.*
import android.widget.FrameLayout
import android.widget.ProgressBar
import com.hjtpx.captcha.R

class CaptchaView @JvmOverloads constructor(
    context: Context,
    attrs: AttributeSet? = null,
    defStyleAttr: Int = 0
) : FrameLayout(context, attrs, defStyleAttr) {
    
    private var webView: WebView? = null
    private var progressBar: ProgressBar? = null
    private var captchaId: String? = null
    private var appId: String? = null
    private var captchaType: CaptchaType = CaptchaType.SLIDER
    private var serverUrl: String = "https://your-domain.com"
    private var language: String = "en-US"
    
    private var delegate: CaptchaViewDelegate? = null
    
    init {
        setupUI()
    }
    
    private fun setupUI() {
        setBackgroundColor(Color.WHITE)
        
        progressBar = ProgressBar(context).apply {
            isIndeterminate = true
            layoutParams = LayoutParams(
                LayoutParams.WRAP_CONTENT,
                LayoutParams.WRAP_CONTENT,
                android.view.Gravity.CENTER
            )
        }
        addView(progressBar)
        
        setupWebView()
    }
    
    @SuppressLint("SetJavaScriptEnabled")
    private fun setupWebView() {
        webView = WebView(context).apply {
            settings.apply {
                javaScriptEnabled = true
                domStorageEnabled = true
                allowFileAccess = true
                loadWithOverviewMode = true
                useWideViewPort = true
                builtInZoomControls = false
                displayZoomControls = false
                cacheMode = WebSettings.LOAD_DEFAULT
                mixedContentMode = WebSettings.MIXED_CONTENT_ALWAYS_ALLOW
            }
            
            if (Build.VERSION.SDK_INT >= Build.VERSION_CODES.O) {
                webViewClient = CaptchaWebViewClient()
                webChromeClient = CaptchaChromeClient()
            }
            
            layoutParams = LayoutParams(
                LayoutParams.MATCH_PARENT,
                LayoutParams.MATCH_PARENT
            )
            
            visibility = View.GONE
        }
        addView(webView)
    }
    
    fun setDelegate(delegate: CaptchaViewDelegate) {
        this.delegate = delegate
    }
    
    fun setCaptchaType(type: CaptchaType) {
        this.captchaType = type
    }
    
    fun setAppId(appId: String) {
        this.appId = appId
    }
    
    fun setServerUrl(url: String) {
        this.serverUrl = url
    }
    
    fun setLanguage(language: String) {
        this.language = language
    }
    
    fun loadCaptcha() {
        val url = buildCaptchaUrl()
        webView?.loadUrl(url)
    }
    
    private fun buildCaptchaUrl(): String {
        val encodedAppId = appId?.replace(" ", "%20") ?: ""
        val encodedType = captchaType.value.replace(" ", "%20")
        val encodedLang = language.replace(" ", "%20")
        
        return "$serverUrl/captcha?app_id=$encodedAppId&type=$encodedType&lang=$encodedLang"
    }
    
    fun setCaptchaId(captchaId: String) {
        this.captchaId = captchaId
    }
    
    fun getCaptchaId(): String? = captchaId
    
    private inner class CaptchaWebViewClient : WebViewClient() {
        override fun onPageFinished(view: WebView?, url: String?) {
            super.onPageFinished(view, url)
            progressBar?.visibility = View.GONE
            webView?.visibility = View.VISIBLE
        }
        
        override fun onReceivedError(
            view: WebView?,
            request: WebResourceRequest?,
            error: WebResourceError?
        ) {
            super.onReceivedError(view, request, error)
            delegate?.onCaptchaError(this@CaptchaView, error?.description?.toString() ?: "Unknown error")
        }
        
        override fun shouldOverrideUrlLoading(
            view: WebView?,
            request: WebResourceRequest?
        ): Boolean {
            request?.url?.let { url ->
                if (url.scheme == "hjtpx") {
                    handleCustomUrl(url)
                    return true
                }
            }
            return super.shouldOverrideUrlLoading(view, request)
        }
    }
    
    private inner class CaptchaChromeClient : WebChromeClient() {
        override fun onProgressChanged(view: WebView?, newProgress: Int) {
            super.onProgressChanged(view, newProgress)
            if (newProgress < 100) {
                progressBar?.visibility = View.VISIBLE
            }
        }
    }
    
    private fun handleCustomUrl(url: android.net.Uri) {
        when (url.host) {
            "verify" -> {
                val verifyId = url.getQueryParameter("verify_id")
                if (verifyId != null) {
                    delegate?.onCaptchaVerified(this@CaptchaView, verifyId)
                }
            }
            "close" -> {
                delegate?.onCaptchaClose(this@CaptchaView)
            }
            "error" -> {
                val message = url.getQueryParameter("message") ?: "Unknown error"
                delegate?.onCaptchaError(this@CaptchaView, message)
            }
        }
    }
    
    fun destroy() {
        webView?.apply {
            stopLoading()
            clearHistory()
            clearCache(true)
            loadUrl("about:blank")
            removeAllViews()
            destroy()
        }
        webView = null
    }
}

interface CaptchaViewDelegate {
    fun onCaptchaVerified(view: CaptchaView, verifyId: String)
    fun onCaptchaError(view: CaptchaView, error: String)
    fun onCaptchaClose(view: CaptchaView)
}
