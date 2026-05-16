package com.hjtpx.sdk;

import com.fasterxml.jackson.core.JsonProcessingException;
import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.DeserializationFeature;
import com.fasterxml.jackson.databind.JsonNode;
import com.fasterxml.jackson.databind.ObjectMapper;
import okhttp3.*;

import java.io.IOException;
import java.net.URLEncoder;
import java.nio.charset.StandardCharsets;
import java.time.Duration;
import java.time.Instant;
import java.util.*;
import java.util.concurrent.TimeUnit;

public class CaptchaClient implements AutoCloseable {
    private static final String IMAGE_CAPTCHA_PATH = "/api/v1/captcha/image";
    private static final String IMAGE_VERIFY_PATH = "/api/v1/captcha/image/verify";
    private static final String SLIDER_CAPTCHA_PATH = "/api/v1/captcha/slider";
    private static final String CLICK_CAPTCHA_PATH = "/api/v1/captcha/click";
    private static final String VERIFY_PATH = "/api/v1/captcha/verify";

    private final Config config;
    private final OkHttpClient httpClient;
    private final ObjectMapper objectMapper;
    private final Stats stats;

    private static class Stats {
        long totalRequests = 0;
        long failedRequests = 0;
        long successfulRequests = 0;
        long retriedRequests = 0;
        String lastError = null;
        String lastErrorTime = null;
    }

    public CaptchaClient() {
        this(new Config());
    }

    public CaptchaClient(Config config) {
        this.config = config != null ? config : new Config();
        this.httpClient = createHttpClient();
        this.objectMapper = createObjectMapper();
        this.stats = new Stats();
    }

    private OkHttpClient createHttpClient() {
        return new OkHttpClient.Builder()
                .connectTimeout(Duration.ofMillis(config.getTimeout()))
                .readTimeout(Duration.ofMillis(config.getTimeout()))
                .writeTimeout(Duration.ofMillis(config.getTimeout()))
                .retryOnConnectionFailure(false)
                .build();
    }

    private ObjectMapper createObjectMapper() {
        ObjectMapper mapper = new ObjectMapper();
        mapper.configure(DeserializationFeature.FAIL_ON_UNKNOWN_PROPERTIES, false);
        return mapper;
    }

    private String buildUrl(String path, Map<String, String> params) {
        StringBuilder url = new StringBuilder();
        url.append(config.getBaseUrl().replaceAll("/$", "")).append(path);

        if (params != null && !params.isEmpty()) {
            StringBuilder queryParams = new StringBuilder();
            for (Map.Entry<String, String> entry : params.entrySet()) {
                if (queryParams.length() > 0) {
                    queryParams.append("&");
                }
                try {
                    queryParams.append(URLEncoder.encode(entry.getKey(), StandardCharsets.UTF_8.name()))
                            .append("=")
                            .append(URLEncoder.encode(entry.getValue(), StandardCharsets.UTF_8.name()));
                } catch (Exception e) {
                    queryParams.append(entry.getKey()).append("=").append(entry.getValue());
                }
            }
            url.append("?").append(queryParams);
        }
        return url.toString();
    }

    private Request.Builder createRequestBuilder(String url, String method) {
        Request.Builder builder = new Request.Builder()
                .url(url)
                .method(method, null)
                .header("Content-Type", "application/json")
                .header("Accept", "application/json");

        if (config.getAppId() != null && !config.getAppId().isEmpty()) {
            builder.header("X-App-ID", config.getAppId());
        }
        if (config.getAppSecret() != null && !config.getAppSecret().isEmpty()) {
            builder.header("X-App-Secret", config.getAppSecret());
        }

        return builder;
    }

    private Request.Builder createRequestBuilder(String url, String method, String jsonBody) {
        RequestBody body = RequestBody.create(jsonBody, MediaType.parse("application/json; charset=utf-8"));
        return createRequestBuilder(url, method).post(body);
    }

    private <T> SDKResponse<T> executeRequest(Request request, TypeReference<SDKResponse<T>> typeRef) throws SDKError {
        stats.totalRequests++;
        SDKError lastError = null;

        for (int attempt = 0; attempt <= config.getMaxRetries(); attempt++) {
            if (attempt > 0) {
                stats.retriedRequests++;
                try {
                    long delay = config.getRetryDelay() * attempt;
                    Thread.sleep(delay);
                } catch (InterruptedException e) {
                    Thread.currentThread().interrupt();
                    throw new SDKError(500, "Request interrupted");
                }
            }

            try (Response response = httpClient.newCall(request).execute()) {
                int statusCode = response.code();
                String body = response.body() != null ? response.body().string() : "";

                if (statusCode == 429) {
                    Integer retryAfter = null;
                    String retryAfterHeader = response.header("Retry-After");
                    if (retryAfterHeader != null) {
                        try {
                            retryAfter = Integer.parseInt(retryAfterHeader);
                        } catch (NumberFormatException ignored) {}
                    }
                    throw new RateLimitedError("Rate limited", retryAfter);
                }

                if (statusCode == 401) {
                    throw new UnauthorizedError();
                }

                if (statusCode >= 500) {
                    if (attempt < config.getMaxRetries()) {
                        continue;
                    }
                    throw new ServerError(statusCode);
                }

                if (statusCode != 200) {
                    throw new SDKError(statusCode, "HTTP error: " + statusCode);
                }

                SDKResponse<T> sdkResponse = objectMapper.readValue(body, typeRef);
                if (sdkResponse.getCode() != 0) {
                    throw new SDKError(sdkResponse.getCode(), sdkResponse.getMessage());
                }

                stats.successfulRequests++;
                return sdkResponse;

            } catch (SDKError e) {
                lastError = e;
                if (e.isRateLimited() || e.isServerError() || e.isUnauthorized()) {
                    if (e.isRateLimited() && ((RateLimitedError) e).getRetryAfter() != null) {
                        try {
                            Thread.sleep(((RateLimitedError) e).getRetryAfter() * 1000L);
                        } catch (InterruptedException ie) {
                            Thread.currentThread().interrupt();
                        }
                    }
                    if (attempt < config.getMaxRetries() && !(e instanceof UnauthorizedError)) {
                        continue;
                    }
                }
                throw e;
            } catch (IOException e) {
                lastError = new NetworkError("Network error: " + e.getMessage(), e);
                if (attempt < config.getMaxRetries()) {
                    continue;
                }
            }
        }

        stats.failedRequests++;
        stats.lastError = lastError != null ? lastError.getMessage() : "Unknown error";
        stats.lastErrorTime = Instant.now().toString();
        if (lastError != null) {
            throw lastError;
        }
        throw new SDKError(500, "Unknown error");
    }

    public ImageCaptchaResponse generateImageCaptcha() throws SDKError {
        return generateImageCaptcha(null);
    }

    public ImageCaptchaResponse generateImageCaptcha(ImageCaptchaRequest request) throws SDKError {
        Map<String, String> params = new HashMap<>();
        if (request != null) {
            if (request.getType() != null) {
                params.put("type", request.getType().getValue());
            }
            if (request.getCount() > 0) {
                params.put("count", String.valueOf(request.getCount()));
            }
            if (request.getCustomSet() != null && !request.getCustomSet().isEmpty()) {
                params.put("custom_set", request.getCustomSet());
            }
            if (request.getNoiseMode() > 0) {
                params.put("noise_mode", String.valueOf(request.getNoiseMode()));
            }
            if (request.getLineMode() > 0) {
                params.put("line_mode", String.valueOf(request.getLineMode()));
            }
        }

        String url = buildUrl(IMAGE_CAPTCHA_PATH, params.isEmpty() ? null : params);
        Request requestObj = createRequestBuilder(url, "GET").build();

        if (config.isDebugMode()) {
            System.out.println("[DEBUG] GET " + url);
        }

        SDKResponse<ImageCaptchaResponse> response = executeRequest(
                requestObj,
                new TypeReference<SDKResponse<ImageCaptchaResponse>>() {}
        );

        return response.getData();
    }

    public VerifyImageCaptchaResponse verifyImageCaptcha(String challengeId, String answer) throws SDKError {
        if (challengeId == null || challengeId.isEmpty()) {
            throw new InvalidParamsError("challenge_id is required");
        }
        if (answer == null || answer.isEmpty()) {
            throw new InvalidParamsError("answer is required");
        }

        String url = buildUrl(IMAGE_VERIFY_PATH, null);
        VerifyImageCaptchaRequest request = new VerifyImageCaptchaRequest(challengeId, answer);

        try {
            String jsonBody = objectMapper.writeValueAsString(request);
            Request requestObj = createRequestBuilder(url, "POST", jsonBody).build();

            if (config.isDebugMode()) {
                System.out.println("[DEBUG] POST " + url);
                System.out.println("[DEBUG] Body: " + jsonBody);
            }

            SDKResponse<VerifyImageCaptchaResponse> response = executeRequest(
                    requestObj,
                    new TypeReference<SDKResponse<VerifyImageCaptchaResponse>>() {}
            );

            return response.getData();
        } catch (JsonProcessingException e) {
            throw new SDKError(500, "Failed to serialize request", e);
        }
    }

    public SliderCaptchaResponse generateSliderCaptcha() throws SDKError {
        return generateSliderCaptcha(null);
    }

    public SliderCaptchaResponse generateSliderCaptcha(SliderCaptchaRequest request) throws SDKError {
        Map<String, String> params = new HashMap<>();
        if (request != null) {
            if (request.getWidth() > 0) {
                params.put("width", String.valueOf(request.getWidth()));
            }
            if (request.getHeight() > 0) {
                params.put("height", String.valueOf(request.getHeight()));
            }
        }

        String url = buildUrl(SLIDER_CAPTCHA_PATH, params.isEmpty() ? null : params);
        Request requestObj = createRequestBuilder(url, "GET").build();

        if (config.isDebugMode()) {
            System.out.println("[DEBUG] GET " + url);
        }

        SDKResponse<SliderCaptchaResponse> response = executeRequest(
                requestObj,
                new TypeReference<SDKResponse<SliderCaptchaResponse>>() {}
        );

        return response.getData();
    }

    public VerifyCaptchaResponse verifySliderCaptcha(String challengeId, String offset) throws SDKError {
        if (challengeId == null || challengeId.isEmpty()) {
            throw new InvalidParamsError("challenge_id is required");
        }
        if (offset == null || offset.isEmpty()) {
            throw new InvalidParamsError("offset is required");
        }

        String url = buildUrl(VERIFY_PATH, null);
        Map<String, Object> data = new HashMap<>();
        data.put("offset", offset);

        VerifyCaptchaRequest request = new VerifyCaptchaRequest(challengeId, "slide", data);

        try {
            String jsonBody = objectMapper.writeValueAsString(request);
            Request requestObj = createRequestBuilder(url, "POST", jsonBody).build();

            if (config.isDebugMode()) {
                System.out.println("[DEBUG] POST " + url);
                System.out.println("[DEBUG] Body: " + jsonBody);
            }

            SDKResponse<VerifyCaptchaResponse> response = executeRequest(
                    requestObj,
                    new TypeReference<SDKResponse<VerifyCaptchaResponse>>() {}
            );

            return response.getData();
        } catch (JsonProcessingException e) {
            throw new SDKError(500, "Failed to serialize request", e);
        }
    }

    public ClickCaptchaResponse generateClickCaptcha() throws SDKError {
        return generateClickCaptcha(null);
    }

    public ClickCaptchaResponse generateClickCaptcha(ClickCaptchaRequest request) throws SDKError {
        Map<String, String> params = new HashMap<>();
        if (request != null) {
            if (request.getWidth() > 0) {
                params.put("width", String.valueOf(request.getWidth()));
            }
            if (request.getHeight() > 0) {
                params.put("height", String.valueOf(request.getHeight()));
            }
            if (request.getIconCount() > 0) {
                params.put("icon_count", String.valueOf(request.getIconCount()));
            }
        }

        String url = buildUrl(CLICK_CAPTCHA_PATH, params.isEmpty() ? null : params);
        Request requestObj = createRequestBuilder(url, "GET").build();

        if (config.isDebugMode()) {
            System.out.println("[DEBUG] GET " + url);
        }

        SDKResponse<ClickCaptchaResponse> response = executeRequest(
                requestObj,
                new TypeReference<SDKResponse<ClickCaptchaResponse>>() {}
        );

        return response.getData();
    }

    public VerifyCaptchaResponse verifyClickCaptcha(String challengeId, List<ClickData> clicks) throws SDKError {
        if (challengeId == null || challengeId.isEmpty()) {
            throw new InvalidParamsError("challenge_id is required");
        }
        if (clicks == null || clicks.isEmpty()) {
            throw new InvalidParamsError("clicks is required");
        }

        String url = buildUrl(VERIFY_PATH, null);
        Map<String, Object> data = new HashMap<>();
        data.put("clicks", clicks);

        VerifyCaptchaRequest request = new VerifyCaptchaRequest(challengeId, "click", data);

        try {
            String jsonBody = objectMapper.writeValueAsString(request);
            Request requestObj = createRequestBuilder(url, "POST", jsonBody).build();

            if (config.isDebugMode()) {
                System.out.println("[DEBUG] POST " + url);
                System.out.println("[DEBUG] Body: " + jsonBody);
            }

            SDKResponse<VerifyCaptchaResponse> response = executeRequest(
                    requestObj,
                    new TypeReference<SDKResponse<VerifyCaptchaResponse>>() {}
            );

            return response.getData();
        } catch (JsonProcessingException e) {
            throw new SDKError(500, "Failed to serialize request", e);
        }
    }

    public byte[] extractBase64Image(String dataUri) throws SDKError {
        if (dataUri == null || dataUri.isEmpty()) {
            throw new InvalidParamsError("data_uri is required");
        }

        try {
            String prefix;
            if (dataUri.startsWith("data:image/png;base64,")) {
                prefix = "data:image/png;base64,";
            } else if (dataUri.startsWith("data:image/jpeg;base64,")) {
                prefix = "data:image/jpeg;base64,";
            } else {
                throw new InvalidParamsError("Unsupported image format");
            }

            String base64Data = dataUri.substring(prefix.length());
            return Base64.getDecoder().decode(base64Data);
        } catch (IllegalArgumentException e) {
            throw new SDKError(500, "Failed to decode base64 image", e);
        }
    }

    public PoolStats getStats() {
        PoolStats poolStats = new PoolStats();
        poolStats.setActiveConnections(0);
        poolStats.setIdleConnections(config.getMaxIdleConns());
        poolStats.setTotalRequests(stats.totalRequests);
        poolStats.setFailedRequests(stats.failedRequests);
        poolStats.setSuccessfulRequests(stats.successfulRequests);
        poolStats.setRetriedRequests(stats.retriedRequests);

        if (stats.totalRequests > 0) {
            poolStats.setSuccessRate((double) stats.successfulRequests / stats.totalRequests * 100);
        } else {
            poolStats.setSuccessRate(0);
        }

        poolStats.setLastError(stats.lastError);
        poolStats.setLastErrorTime(stats.lastErrorTime);
        return poolStats;
    }

    public void setDebugMode(boolean enabled) {
        config.setDebugMode(enabled);
    }

    public void setTimeout(int timeout) {
        config.setTimeout(timeout);
    }

    public void setMaxRetries(int maxRetries) {
        config.setMaxRetries(maxRetries);
    }

    @Override
    public void close() {
        httpClient.dispatcher().executorService().shutdown();
        httpClient.connectionPool().evictAll();
    }

    public Config getConfig() {
        return config;
    }

    public static CaptchaClient create(String baseUrl) {
        Config config = new Config();
        config.setBaseUrl(baseUrl);
        return new CaptchaClient(config);
    }

    public static CaptchaClient create(String baseUrl, String appId, String appSecret) {
        Config config = new Config();
        config.setBaseUrl(baseUrl);
        config.setAppId(appId);
        config.setAppSecret(appSecret);
        return new CaptchaClient(config);
    }
}
