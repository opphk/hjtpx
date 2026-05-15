package com.captchax.example

import android.os.Bundle
import android.util.Log
import android.view.View
import android.widget.Toast
import androidx.appcompat.app.AppCompatActivity
import com.captchax.sdk.CaptchaError
import com.captchax.sdk.CaptchaListener
import com.captchax.sdk.CaptchaType
import com.captchax.sdk.CaptchaView
import com.captchax.sdk.CaptchaViewListener
import com.captchax.sdk.CaptchaX
import com.captchax.sdk.util.Logger

class MainActivity : AppCompatActivity() {
    
    private var captchaView: CaptchaView? = null
    private val tag = "CaptchaX-Demo"
    
    override fun onCreate(savedInstanceState: Bundle?) {
        super.onCreate(savedInstanceState)
        setContentView(R.layout.activity_main)
        
        CaptchaX.initialize(
            context = this,
            apiKey = "YOUR_API_KEY",
            apiSecret = "YOUR_API_SECRET",
            serverUrl = "https://api.captchax.com"
        )
        
        CaptchaX.listener = object : CaptchaListener {
            override fun onSuccess(token: String) {
                Log.d(tag, "Verification success: $token")
                Toast.makeText(this@MainActivity, "Verification successful!", Toast.LENGTH_SHORT).show()
            }
            
            override fun onError(error: CaptchaError) {
                Log.e(tag, "Verification error: ${error.message}")
                Toast.makeText(this@MainActivity, "Error: ${error.message}", Toast.LENGTH_SHORT).show()
            }
            
            override fun onClose() {
                Log.d(tag, "Verification closed")
            }
        }
        
        Logger.enableDebug(true)
    }
    
    fun showSliderCaptcha(view: View) {
        showCaptcha(CaptchaType.SLIDER)
    }
    
    fun showClickCaptcha(view: View) {
        showCaptcha(CaptchaType.CLICK)
    }
    
    fun showPuzzleCaptcha(view: View) {
        showCaptcha(CaptchaType.PUZZLE)
    }
    
    fun showRotateCaptcha(view: View) {
        showCaptcha(CaptchaType.ROTATE)
    }
    
    fun showTextCaptcha(view: View) {
        showCaptcha(CaptchaType.TEXT)
    }
    
    fun showIconCaptcha(view: View) {
        showCaptcha(CaptchaType.ICON)
    }
    
    private fun showCaptcha(type: CaptchaType) {
        CaptchaX.verify(this, "login") { result ->
            result.onSuccess { token ->
                Log.d(tag, "Token: $token")
                Toast.makeText(this, "Token: $token", Toast.LENGTH_LONG).show()
            }.onFailure { error ->
                Log.e(tag, "Error: ${error.message}")
                Toast.makeText(this, "Error: ${error.message}", Toast.LENGTH_SHORT).show()
            }
        }
    }
    
    fun preloadCaptcha(view: View) {
        CaptchaX.preload("login")
        Toast.makeText(this, "Preloading captcha...", Toast.LENGTH_SHORT).show()
    }
    
    override fun onDestroy() {
        super.onDestroy()
        CaptchaX.destroy()
    }
}
