package com.captchax.example;

import android.os.Bundle;
import android.util.Log;
import android.view.View;
import android.widget.Toast;

import androidx.annotation.NonNull;
import androidx.appcompat.app.AppCompatActivity;

import com.captchax.sdk.CaptchaError;
import com.captchax.sdk.CaptchaListener;
import com.captchax.sdk.CaptchaType;
import com.captchax.sdk.CaptchaX;

public class JavaExampleActivity extends AppCompatActivity {
    
    private static final String TAG = "CaptchaX-Java";
    
    @Override
    protected void onCreate(Bundle savedInstanceState) {
        super.onCreate(savedInstanceState);
        setContentView(R.layout.activity_main);
        
        CaptchaX.INSTANCE.initialize(
            this,
            "YOUR_API_KEY",
            "YOUR_API_SECRET",
            "https://api.captchax.com"
        );
        
        CaptchaX.INSTANCE.setListener(new CaptchaListener() {
            @Override
            public void onSuccess(@NonNull String token) {
                Log.d(TAG, "Verification success: " + token);
                Toast.makeText(JavaExampleActivity.this, "Success!", Toast.LENGTH_SHORT).show();
            }
            
            @Override
            public void onError(@NonNull CaptchaError error) {
                Log.e(TAG, "Verification error: " + error.getMessage());
                Toast.makeText(JavaExampleActivity.this, "Error: " + error.getMessage(), Toast.LENGTH_SHORT).show();
            }
            
            @Override
            public void onClose() {
                Log.d(TAG, "Verification closed");
            }
        });
    }
    
    public void showSliderCaptcha(View view) {
        verifyCaptcha("login");
    }
    
    private void verifyCaptcha(String scene) {
        CaptchaX.INSTANCE.verify(this, scene, result -> {
            if (result.isSuccess()) {
                String token = result.getOrNull();
                Log.d(TAG, "Token: " + token);
                Toast.makeText(this, "Token: " + token, Toast.LENGTH_LONG).show();
            } else {
                Exception error = result.getExceptionOrNull();
                Log.e(TAG, "Error: " + (error != null ? error.getMessage() : "Unknown"));
                Toast.makeText(this, "Error: " + (error != null ? error.getMessage() : "Unknown"), Toast.LENGTH_SHORT).show();
            }
        });
    }
    
    @Override
    protected void onDestroy() {
        super.onDestroy();
        CaptchaX.INSTANCE.destroy();
    }
}
