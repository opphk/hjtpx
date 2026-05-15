package com.captchax.example

import android.app.Activity
import androidx.compose.foundation.layout.Column
import androidx.compose.foundation.layout.fillMaxWidth
import androidx.compose.foundation.layout.padding
import androidx.compose.material3.Button
import androidx.compose.material3.MaterialTheme
import androidx.compose.material3.OutlinedButton
import androidx.compose.material3.Text
import androidx.compose.runtime.Composable
import androidx.compose.runtime.DisposableEffect
import androidx.compose.runtime.getValue
import androidx.compose.runtime.mutableStateOf
import androidx.compose.runtime.remember
import androidx.compose.runtime.setValue
import androidx.compose.ui.Modifier
import androidx.compose.ui.platform.LocalContext
import androidx.compose.ui.unit.dp
import com.captchax.sdk.CaptchaError
import com.captchax.sdk.CaptchaType
import com.captchax.sdk.CaptchaView
import com.captchax.sdk.CaptchaViewListener
import com.captchax.sdk.CaptchaX

@Composable
fun CaptchaButton(
    scene: String,
    onSuccess: (String) -> Unit,
    onError: (CaptchaError) -> Unit,
    modifier: Modifier = Modifier
) {
    val context = LocalContext.current as Activity
    var showCaptcha by remember { mutableStateOf(false) }
    var captchaView by remember { mutableStateOf<CaptchaView?>(null) }
    
    Button(
        onClick = { showCaptcha = true },
        modifier = modifier.fillMaxWidth()
    ) {
        Text("验证")
    }
    
    if (showCaptcha) {
        CaptchaDialog(
            onDismiss = { showCaptcha = false },
            onSuccess = { token ->
                showCaptcha = false
                onSuccess(token)
            },
            onError = { error ->
                showCaptcha = false
                onError(error)
            }
        )
    }
}

@Composable
fun CaptchaDialog(
    onDismiss: () -> Unit,
    onSuccess: (String) -> Unit,
    onError: (CaptchaError) -> Unit
) {
    val context = LocalContext.current
    var captchaView by remember { mutableStateOf<CaptchaView?>(null) }
    var captchaType by remember { mutableStateOf(CaptchaType.SLIDER) }
    
    DisposableEffect(Unit) {
        onDispose {
            captchaView?.destroy()
        }
    }
    
    Column(modifier = Modifier.padding(16.dp)) {
        Text(
            text = "请完成验证",
            style = MaterialTheme.typography.headlineSmall,
            modifier = Modifier.padding(bottom = 16.dp)
        )
        
        captchaView = CaptchaView(context).apply {
            this.listener = object : CaptchaViewListener {
                override fun onSuccess(token: String) {
                    onSuccess(token)
                }
                
                override fun onError(error: CaptchaError) {
                    onError(error)
                }
                
                override fun onClose() {
                    onDismiss()
                }
                
                override fun onReady() {
                }
                
                override fun onLoading() {
                }
                
                override fun onLoaded() {
                }
            }
            load(captchaType)
        }
        
        OutlinedButton(
            onClick = {
                captchaView?.destroy()
                onDismiss()
            },
            modifier = Modifier
                .fillMaxWidth()
                .padding(top = 16.dp)
        ) {
            Text("取消")
        }
    }
}

@Composable
fun CaptchaTypeSelector(
    onTypeSelected: (CaptchaType) -> Unit,
    modifier: Modifier = Modifier
) {
    Column(modifier = modifier.padding(16.dp)) {
        Text(
            text = "选择验证码类型",
            style = MaterialTheme.typography.titleMedium,
            modifier = Modifier.padding(bottom = 8.dp)
        )
        
        CaptchaType.entries.forEach { type ->
            OutlinedButton(
                onClick = { onTypeSelected(type) },
                modifier = Modifier
                    .fillMaxWidth()
                    .padding(vertical = 4.dp)
            ) {
                Text(type.name)
            }
        }
    }
}
