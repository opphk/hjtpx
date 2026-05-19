package com.hjtpx.captcha.sdk;

import android.content.Context;
import android.os.Handler;
import android.os.Looper;
import android.util.Log;

import org.json.JSONObject;

import java.io.IOException;
import java.util.concurrent.TimeUnit;

import okhttp3.Call;
import okhttp3.Callback;
import okhttp3.MediaType;
import okhttp3.OkHttpClient;
import okhttp3.Request;
import okhttp3.RequestBody;
import okhttp3.Response;

public class CaptchaClient {
    private static final String TAG = "HjtpxCaptcha";
    private static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");

    private final String baseUrl;
    private final String appId;
    private final String appSecret;
    private final OkHttpClient client;
    private final Handler mainHandler;

    private CaptchaListener listener;

    public CaptchaClient(Context context, String baseUrl, String appId, String appSecret) {
        this.baseUrl = baseUrl;
        this.appId = appId;
        this.appSecret = appSecret;
        this.mainHandler = new Handler(Looper.getMainLooper());

        this.client = new OkHttpClient.Builder()
                .connectTimeout(30, TimeUnit.SECONDS)
                .readTimeout(30, TimeUnit.SECONDS)
                .writeTimeout(30, TimeUnit.SECONDS)
                .retryOnConnectionFailure(true)
                .build();
    }

    public void setListener(CaptchaListener listener) {
        this.listener = listener;
    }

    public void generateSliderCaptcha(int width, int height, CaptchaCallback callback) {
        try {
            JSONObject requestBody = new JSONObject();
            requestBody.put("app_id", appId);
            requestBody.put("captcha_type", "slider");
            requestBody.put("width", width);
            requestBody.put("height", height);

            String url = baseUrl + "/api/captcha/slider";
            RequestBody body = RequestBody.create(requestBody.toString(), JSON);
            Request request = new Request.Builder()
                    .url(url)
                    .post(body)
                    .addHeader("Content-Type", "application/json")
                    .build();

            client.newCall(request).enqueue(new Callback() {
                @Override
                public void onFailure(Call call, IOException e) {
                    mainHandler.post(() -> callback.onError(e.getMessage()));
                }

                @Override
                public void onResponse(Call call, Response response) throws IOException {
                    try {
                        String responseBody = response.body().string();
                        JSONObject jsonResponse = new JSONObject(responseBody);

                        SliderCaptchaResult result = new SliderCaptchaResult();
                        result.sessionId = jsonResponse.getString("session_id");
                        result.backgroundImage = baseUrl + jsonResponse.getString("background_image");
                        result.sliderImage = baseUrl + jsonResponse.getString("slider_image");

                        mainHandler.post(() -> callback.onSuccess(result));
                    } catch (Exception e) {
                        mainHandler.post(() -> callback.onError("Parse error: " + e.getMessage()));
                    }
                }
            });
        } catch (Exception e) {
            callback.onError(e.getMessage());
        }
    }

    public void verifySliderCaptcha(String sessionId, float x, CaptchaCallback callback) {
        try {
            JSONObject requestBody = new JSONObject();
            requestBody.put("session_id", sessionId);
            requestBody.put("x", x);
            requestBody.put("app_id", appId);

            String url = baseUrl + "/api/captcha/verify/slider";
            RequestBody body = RequestBody.create(requestBody.toString(), JSON);
            Request request = new Request.Builder()
                    .url(url)
                    .post(body)
                    .addHeader("Content-Type", "application/json")
                    .build();

            client.newCall(request).enqueue(new Callback() {
                @Override
                public void onFailure(Call call, IOException e) {
                    mainHandler.post(() -> callback.onError(e.getMessage()));
                }

                @Override
                public void onResponse(Call call, Response response) throws IOException {
                    try {
                        String responseBody = response.body().string();
                        JSONObject jsonResponse = new JSONObject(responseBody);

                        VerifyResult result = new VerifyResult();
                        result.success = jsonResponse.getBoolean("success");
                        result.score = jsonResponse.optDouble("score", 0.0);
                        result.message = jsonResponse.optString("message", "");

                        mainHandler.post(() -> callback.onSuccess(result));
                    } catch (Exception e) {
                        mainHandler.post(() -> callback.onError("Parse error: " + e.getMessage()));
                    }
                }
            });
        } catch (Exception e) {
            callback.onError(e.getMessage());
        }
    }

    public void generateClickCaptcha(int count, CaptchaCallback callback) {
        try {
            JSONObject requestBody = new JSONObject();
            requestBody.put("app_id", appId);
            requestBody.put("captcha_type", "click");
            requestBody.put("count", count);

            String url = baseUrl + "/api/captcha/click";
            RequestBody body = RequestBody.create(requestBody.toString(), JSON);
            Request request = new Request.Builder()
                    .url(url)
                    .post(body)
                    .addHeader("Content-Type", "application/json")
                    .build();

            client.newCall(request).enqueue(new Callback() {
                @Override
                public void onFailure(Call call, IOException e) {
                    mainHandler.post(() -> callback.onError(e.getMessage()));
                }

                @Override
                public void onResponse(Call call, Response response) throws IOException {
                    try {
                        String responseBody = response.body().string();
                        JSONObject jsonResponse = new JSONObject(responseBody);

                        ClickCaptchaResult result = new ClickCaptchaResult();
                        result.sessionId = jsonResponse.getString("session_id");
                        result.backgroundImage = baseUrl + jsonResponse.getString("background_image");
                        result.targetCount = jsonResponse.getInt("target_count");

                        mainHandler.post(() -> callback.onSuccess(result));
                    } catch (Exception e) {
                        mainHandler.post(() -> callback.onError("Parse error: " + e.getMessage()));
                    }
                }
            });
        } catch (Exception e) {
            callback.onError(e.getMessage());
        }
    }

    public void verifyClickCaptcha(String sessionId, int[] xCoords, int[] yCoords, CaptchaCallback callback) {
        try {
            JSONObject requestBody = new JSONObject();
            requestBody.put("session_id", sessionId);
            requestBody.put("app_id", appId);

            org.json.JSONArray xArray = new org.json.JSONArray();
            for (int x : xCoords) xArray.put(x);
            requestBody.put("x_coords", xArray);

            org.json.JSONArray yArray = new org.json.JSONArray();
            for (int y : yCoords) yArray.put(y);
            requestBody.put("y_coords", yArray);

            String url = baseUrl + "/api/captcha/verify/click";
            RequestBody body = RequestBody.create(requestBody.toString(), JSON);
            Request request = new Request.Builder()
                    .url(url)
                    .post(body)
                    .addHeader("Content-Type", "application/json")
                    .build();

            client.newCall(request).enqueue(new Callback() {
                @Override
                public void onFailure(Call call, IOException e) {
                    mainHandler.post(() -> callback.onError(e.getMessage()));
                }

                @Override
                public void onResponse(Call call, Response response) throws IOException {
                    try {
                        String responseBody = response.body().string();
                        JSONObject jsonResponse = new JSONObject(responseBody);

                        VerifyResult result = new VerifyResult();
                        result.success = jsonResponse.getBoolean("success");
                        result.score = jsonResponse.optDouble("score", 0.0);
                        result.message = jsonResponse.optString("message", "");

                        mainHandler.post(() -> callback.onSuccess(result));
                    } catch (Exception e) {
                        mainHandler.post(() -> callback.onError("Parse error: " + e.getMessage()));
                    }
                }
            });
        } catch (Exception e) {
            callback.onError(e.getMessage());
        }
    }

    public void destroy() {
        client.dispatcher().cancelAll();
    }

    public interface CaptchaCallback<T> {
        void onSuccess(T result);
        void onError(String error);
    }

    public static class SliderCaptchaResult {
        public String sessionId;
        public String backgroundImage;
        public String sliderImage;
    }

    public static class ClickCaptchaResult {
        public String sessionId;
        public String backgroundImage;
        public int targetCount;
    }

    public static class VerifyResult {
        public boolean success;
        public double score;
        public String message;
    }
}
