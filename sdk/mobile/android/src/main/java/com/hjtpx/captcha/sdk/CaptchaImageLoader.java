package com.hjtpx.captcha.sdk;

import android.content.Context;
import android.graphics.Bitmap;
import android.graphics.BitmapFactory;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;

import java.io.IOException;
import java.io.InputStream;
import java.net.HttpURLConnection;
import java.net.URL;
import java.util.concurrent.ExecutorService;
import java.util.concurrent.Executors;

public class CaptchaImageLoader {
    private static final String TAG = "ImageLoader";
    private final ExecutorService executorService;
    private final Handler mainHandler;
    private final CaptchaClient client;

    public CaptchaImageLoader(CaptchaClient client) {
        this.client = client;
        this.executorService = Executors.newFixedThreadPool(3);
        this.mainHandler = new Handler(Looper.getMainLooper());
    }

    public void loadImage(String imageUrl, ImageCallback callback) {
        executorService.execute(() -> {
            try {
                URL url = new URL(imageUrl);
                HttpURLConnection connection = (HttpURLConnection) url.openConnection();
                connection.setDoInput(true);
                connection.setConnectTimeout(10000);
                connection.setReadTimeout(10000);
                connection.setRequestProperty("User-Agent", "HjtpxCaptcha-Android/1.0");

                InputStream inputStream = connection.getInputStream();
                Bitmap bitmap = BitmapFactory.decodeStream(inputStream);
                inputStream.close();
                connection.disconnect();

                mainHandler.post(() -> callback.onSuccess(bitmap));
            } catch (IOException e) {
                Log.e(TAG, "Failed to load image: " + e.getMessage());
                mainHandler.post(() -> callback.onError(e.getMessage()));
            }
        });
    }

    public void preloadImage(String imageUrl) {
        executorService.execute(() -> {
            try {
                URL url = new URL(imageUrl);
                HttpURLConnection connection = (HttpURLConnection) url.openConnection();
                connection.setDoInput(true);
                connection.setConnectTimeout(5000);
                connection.setReadTimeout(5000);
                connection.setRequestProperty("User-Agent", "HjtpxCaptcha-Android/1.0");

                InputStream inputStream = connection.getInputStream();
                BitmapFactory.decodeStream(inputStream);
                inputStream.close();
                connection.disconnect();
            } catch (IOException e) {
                Log.w(TAG, "Preload failed for: " + imageUrl);
            }
        });
    }

    public void destroy() {
        executorService.shutdown();
    }

    public interface ImageCallback {
        void onSuccess(Bitmap bitmap);
        void onError(String error);
    }
}
