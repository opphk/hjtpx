package com.captchax.sdk;

import com.google.gson.Gson;
import com.google.gson.GsonBuilder;
import okhttp3.*;

import java.io.IOException;
import java.util.*;
import java.util.concurrent.TimeUnit;

public class CaptchaXClient {
    private static final MediaType JSON = MediaType.parse("application/json; charset=utf-8");

    private final String baseUrl;
    private String appId;
    private int timeout;
    private int retryTimes;
    private ApiVersion apiVersion;
    private final OkHttpClient httpClient;
    private final Gson gson;
    private final Map<String, String> defaultHeaders;

    public CaptchaXClient(CaptchaConfig config) {
        if (config.getBaseUrl() == null || config.getBaseUrl().isEmpty()) {
            throw new IllegalArgumentException("baseUrl is required");
        }
        this.baseUrl = config.getBaseUrl().replaceAll("/+$", "");
        this.appId = config.getAppId();
        this.timeout = config.getTimeout();
        this.retryTimes = config.getRetryTimes();
        this.apiVersion = config.getApiVersion();

        this.httpClient = new OkHttpClient.Builder()
                .connectTimeout(this.timeout, TimeUnit.MILLISECONDS)
                .readTimeout(this.timeout, TimeUnit.MILLISECONDS)
                .writeTimeout(this.timeout, TimeUnit.MILLISECONDS)
                .retryOnConnectionFailure(true)
                .build();

        this.gson = new GsonBuilder().create();
        this.defaultHeaders = new HashMap<>();
        this.defaultHeaders.put("Content-Type", "application/json");
        this.defaultHeaders.put("Accept", "application/json");

        if (this.appId != null) {
            this.defaultHeaders.put("X-App-ID", this.appId);
        }
    }

    public CaptchaXClient(String baseUrl) {
        this(new CaptchaConfig(baseUrl));
    }

    public CaptchaXClient(String baseUrl, String appId) {
        this(new CaptchaConfig(baseUrl, appId));
    }

    public void setAppId(String appId) {
        this.appId = appId;
        this.defaultHeaders.put("X-App-ID", appId);
    }

    public void setApiVersion(ApiVersion apiVersion) {
        this.apiVersion = apiVersion;
    }

    public ApiVersion getApiVersion() {
        return this.apiVersion;
    }

    private String getApiPrefix() {
        return "/api/" + apiVersion.getValue();
    }

    private String buildUrl(String endpoint) {
        return baseUrl + endpoint;
    }

    private String post(String endpoint, Map<String, Object> body) throws CaptchaXException {
        return post(endpoint, body, null);
    }

    private String post(String endpoint, Map<String, Object> body, String deduplicationId) throws CaptchaXException {
        String url = buildUrl(endpoint);
        Exception lastException = null;

        for (int attempt = 0; attempt <= retryTimes; attempt++) {
            try {
                RequestBody requestBody = RequestBody.create(gson.toJson(body), JSON);

                Headers.Builder headersBuilder = new Headers.Builder();
                for (Map.Entry<String, String> entry : defaultHeaders.entrySet()) {
                    headersBuilder.add(entry.getKey(), entry.getValue());
                }
                if (deduplicationId != null) {
                    headersBuilder.add("X-Deduplication-ID", deduplicationId);
                }

                Request request = new Request.Builder()
                        .url(url)
                        .post(requestBody)
                        .headers(headersBuilder.build())
                        .build();

                try (Response response = httpClient.newCall(request).execute()) {
                    String responseBody = response.body() != null ? response.body().string() : "";

                    if (!response.isSuccessful()) {
                        throw new CaptchaXException(
                                "HTTP error: " + response.code(),
                                response.code(),
                                response.code()
                        );
                    }

                    return responseBody;
                }
            } catch (CaptchaXException e) {
                throw e;
            } catch (IOException e) {
                lastException = e;
                if (attempt < retryTimes) {
                    try {
                        Thread.sleep((long) Math.pow(2, attempt) * 1000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw new CaptchaXException("Request interrupted", ie);
                    }
                }
            }
        }

        throw new CaptchaXException("Request failed after " + (retryTimes + 1) + " attempts: " + lastException.getMessage());
    }

    private String get(String endpoint) throws CaptchaXException {
        String url = buildUrl(endpoint);
        Exception lastException = null;

        for (int attempt = 0; attempt <= retryTimes; attempt++) {
            try {
                Headers.Builder headersBuilder = new Headers.Builder();
                for (Map.Entry<String, String> entry : defaultHeaders.entrySet()) {
                    headersBuilder.add(entry.getKey(), entry.getValue());
                }

                Request request = new Request.Builder()
                        .url(url)
                        .get()
                        .headers(headersBuilder.build())
                        .build();

                try (Response response = httpClient.newCall(request).execute()) {
                    String responseBody = response.body() != null ? response.body().string() : "";

                    if (!response.isSuccessful()) {
                        throw new CaptchaXException(
                                "HTTP error: " + response.code(),
                                response.code(),
                                response.code()
                        );
                    }

                    return responseBody;
                }
            } catch (CaptchaXException e) {
                throw e;
            } catch (IOException e) {
                lastException = e;
                if (attempt < retryTimes) {
                    try {
                        Thread.sleep((long) Math.pow(2, attempt) * 1000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw new CaptchaXException("Request interrupted", ie);
                    }
                }
            }
        }

        throw new CaptchaXException("Request failed after " + (retryTimes + 1) + " attempts: " + lastException.getMessage());
    }

    private String put(String endpoint, Map<String, Object> body) throws CaptchaXException {
        String url = buildUrl(endpoint);
        Exception lastException = null;

        for (int attempt = 0; attempt <= retryTimes; attempt++) {
            try {
                RequestBody requestBody = RequestBody.create(gson.toJson(body), JSON);

                Headers.Builder headersBuilder = new Headers.Builder();
                for (Map.Entry<String, String> entry : defaultHeaders.entrySet()) {
                    headersBuilder.add(entry.getKey(), entry.getValue());
                }

                Request request = new Request.Builder()
                        .url(url)
                        .put(requestBody)
                        .headers(headersBuilder.build())
                        .build();

                try (Response response = httpClient.newCall(request).execute()) {
                    String responseBody = response.body() != null ? response.body().string() : "";

                    if (!response.isSuccessful()) {
                        throw new CaptchaXException(
                                "HTTP error: " + response.code(),
                                response.code(),
                                response.code()
                        );
                    }

                    return responseBody;
                }
            } catch (CaptchaXException e) {
                throw e;
            } catch (IOException e) {
                lastException = e;
                if (attempt < retryTimes) {
                    try {
                        Thread.sleep((long) Math.pow(2, attempt) * 1000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw new CaptchaXException("Request interrupted", ie);
                    }
                }
            }
        }

        throw new CaptchaXException("Request failed after " + (retryTimes + 1) + " attempts: " + lastException.getMessage());
    }

    private String delete(String endpoint) throws CaptchaXException {
        String url = buildUrl(endpoint);
        Exception lastException = null;

        for (int attempt = 0; attempt <= retryTimes; attempt++) {
            try {
                Headers.Builder headersBuilder = new Headers.Builder();
                for (Map.Entry<String, String> entry : defaultHeaders.entrySet()) {
                    headersBuilder.add(entry.getKey(), entry.getValue());
                }

                Request request = new Request.Builder()
                        .url(url)
                        .delete()
                        .headers(headersBuilder.build())
                        .build();

                try (Response response = httpClient.newCall(request).execute()) {
                    String responseBody = response.body() != null ? response.body().string() : "";

                    if (!response.isSuccessful()) {
                        throw new CaptchaXException(
                                "HTTP error: " + response.code(),
                                response.code(),
                                response.code()
                        );
                    }

                    return responseBody;
                }
            } catch (CaptchaXException e) {
                throw e;
            } catch (IOException e) {
                lastException = e;
                if (attempt < retryTimes) {
                    try {
                        Thread.sleep((long) Math.pow(2, attempt) * 1000);
                    } catch (InterruptedException ie) {
                        Thread.currentThread().interrupt();
                        throw new CaptchaXException("Request interrupted", ie);
                    }
                }
            }
        }

        throw new CaptchaXException("Request failed after " + (retryTimes + 1) + " attempts: " + lastException.getMessage());
    }

    private <T> T parseResponse(String responseBody, Class<T> clazz) throws CaptchaXException {
        ApiResponse<?> response = gson.fromJson(responseBody, ApiResponse.class);
        if (response == null || !response.isSuccess()) {
            String message = response != null ? response.getMessage() : "Unknown error";
            int code = response != null ? response.getCode() : 500;
            throw new CaptchaXException(message, code, 500);
        }
        String dataJson = gson.toJson(response.getData());
        return gson.fromJson(dataJson, clazz);
    }

    public HealthStatus healthCheck() throws CaptchaXException {
        String response = get("/health");
        return parseResponse(response, HealthStatus.class);
    }

    public SliderCaptchaResult generateSliderCaptcha() throws CaptchaXException {
        return generateSliderCaptcha(null, null, null, null);
    }

    public SliderCaptchaResult generateSliderCaptcha(Integer width, Integer height) throws CaptchaXException {
        return generateSliderCaptcha(width, height, null, null);
    }

    public SliderCaptchaResult generateSliderCaptcha(Integer width, Integer height, String clientInfo, String scenarioId) throws CaptchaXException {
        if (appId == null) {
            throw new CaptchaXException("appId is required for captcha generation");
        }

        Map<String, Object> body = new HashMap<>();
        body.put("app_id", appId);
        if (width != null) body.put("width", width);
        if (height != null) body.put("height", height);
        if (clientInfo != null) body.put("client_info", clientInfo);
        if (scenarioId != null) body.put("scenario_id", scenarioId);

        String response = post(getApiPrefix() + "/captcha/slider", body);
        return parseResponse(response, SliderCaptchaResult.class);
    }

    public SliderVerifyResult verifySliderCaptcha(String captchaId, int targetX) throws CaptchaXException {
        return verifySliderCaptcha(captchaId, targetX, 0);
    }

    public SliderVerifyResult verifySliderCaptcha(String captchaId, int targetX, Integer targetY) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        body.put("captcha_id", captchaId);
        body.put("target_x", targetX);
        if (targetY != null) body.put("target_y", targetY);

        String response = post(getApiPrefix() + "/captcha/slider/verify", body);
        return parseResponse(response, SliderVerifyResult.class);
    }

    public ClickCaptchaResult generateClickCaptcha() throws CaptchaXException {
        return generateClickCaptcha(null, null, null);
    }

    public ClickCaptchaResult generateClickCaptcha(Integer charCount) throws CaptchaXException {
        return generateClickCaptcha(charCount, null, null);
    }

    public ClickCaptchaResult generateClickCaptcha(Integer charCount, String clientInfo, String scenarioId) throws CaptchaXException {
        if (appId == null) {
            throw new CaptchaXException("appId is required for captcha generation");
        }

        Map<String, Object> body = new HashMap<>();
        body.put("app_id", appId);
        if (charCount != null) body.put("char_count", charCount);
        if (clientInfo != null) body.put("client_info", clientInfo);
        if (scenarioId != null) body.put("scenario_id", scenarioId);

        String response = post(getApiPrefix() + "/captcha/click", body);
        return parseResponse(response, ClickCaptchaResult.class);
    }

    public ClickVerifyResult verifyClickCaptcha(String captchaId, List<CharPosition> clicks) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        body.put("captcha_id", captchaId);
        List<Map<String, Object>> clicksData = new ArrayList<>();
        for (CharPosition click : clicks) {
            Map<String, Object> clickData = new HashMap<>();
            clickData.put("char", click.getCharacter());
            clickData.put("x", click.getX());
            clickData.put("y", click.getY());
            clicksData.add(clickData);
        }
        body.put("clicks", clicksData);

        String response = post(getApiPrefix() + "/captcha/click/verify", body);
        return parseResponse(response, ClickVerifyResult.class);
    }

    public PuzzleCaptchaResult generatePuzzleCaptcha() throws CaptchaXException {
        return generatePuzzleCaptcha(null, null, null, null);
    }

    public PuzzleCaptchaResult generatePuzzleCaptcha(Integer width, Integer height) throws CaptchaXException {
        return generatePuzzleCaptcha(width, height, null, null);
    }

    public PuzzleCaptchaResult generatePuzzleCaptcha(Integer width, Integer height, String clientInfo, String scenarioId) throws CaptchaXException {
        if (appId == null) {
            throw new CaptchaXException("appId is required for captcha generation");
        }

        Map<String, Object> body = new HashMap<>();
        body.put("app_id", appId);
        if (width != null) body.put("width", width);
        if (height != null) body.put("height", height);
        if (clientInfo != null) body.put("client_info", clientInfo);
        if (scenarioId != null) body.put("scenario_id", scenarioId);

        String response = post(getApiPrefix() + "/captcha/puzzle", body);
        return parseResponse(response, PuzzleCaptchaResult.class);
    }

    public PuzzleVerifyResult verifyPuzzleCaptcha(String captchaId, int targetX) throws CaptchaXException {
        return verifyPuzzleCaptcha(captchaId, targetX, 0);
    }

    public PuzzleVerifyResult verifyPuzzleCaptcha(String captchaId, int targetX, Integer targetY) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        body.put("captcha_id", captchaId);
        body.put("target_x", targetX);
        if (targetY != null) body.put("target_y", targetY);

        String response = post(getApiPrefix() + "/captcha/puzzle/verify", body);
        return parseResponse(response, PuzzleVerifyResult.class);
    }

    public BatchVerifyResponse batchVerify(List<BatchVerifyItem> items) throws CaptchaXException {
        return batchVerify(items, null);
    }

    public BatchVerifyResponse batchVerify(List<BatchVerifyItem> items, String deduplicationId) throws CaptchaXException {
        List<Map<String, Object>> itemsData = new ArrayList<>();
        for (BatchVerifyItem item : items) {
            Map<String, Object> itemData = new HashMap<>();
            itemData.put("captcha_id", item.captchaId);
            itemData.put("type", item.type);
            itemData.put("target_x", item.targetX);
            if (item.targetY != null) itemData.put("target_y", item.targetY);
            if (item.clicks != null) {
                List<Map<String, Object>> clicksData = new ArrayList<>();
                for (CharPosition click : item.clicks) {
                    Map<String, Object> clickData = new HashMap<>();
                    clickData.put("char", click.getCharacter());
                    clickData.put("x", click.getX());
                    clickData.put("y", click.getY());
                    clicksData.add(clickData);
                }
                itemData.put("clicks", clicksData);
            }
            itemsData.add(itemData);
        }

        Map<String, Object> body = new HashMap<>();
        body.put("items", itemsData);

        String response = post(getApiPrefix() + "/captcha/batch/verify", body, deduplicationId);
        return parseResponse(response, BatchVerifyResponse.class);
    }

    public Map<String, Object> listScenarios() throws CaptchaXException {
        String response = get(getApiPrefix() + "/captcha/scenarios");
        return parseResponseAsMap(response);
    }

    public Scenario createScenario(String name, String description, String difficulty, Map<String, Object> config) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        body.put("name", name);
        if (description != null) body.put("description", description);
        if (difficulty != null) body.put("difficulty", difficulty);
        if (config != null) body.put("config", config);

        String response = post(getApiPrefix() + "/captcha/scenarios", body);
        return parseResponse(response, Scenario.class);
    }

    public Scenario getScenario(String scenarioId) throws CaptchaXException {
        String response = get(getApiPrefix() + "/captcha/scenarios/" + scenarioId);
        return parseResponse(response, Scenario.class);
    }

    public Scenario updateScenario(String scenarioId, String name, String description, String difficulty, Map<String, Object> config) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        if (name != null) body.put("name", name);
        if (description != null) body.put("description", description);
        if (difficulty != null) body.put("difficulty", difficulty);
        if (config != null) body.put("config", config);

        String response = put(getApiPrefix() + "/captcha/scenarios/" + scenarioId, body);
        return parseResponse(response, Scenario.class);
    }

    public Map<String, Object> deleteScenario(String scenarioId) throws CaptchaXException {
        String response = delete(getApiPrefix() + "/captcha/scenarios/" + scenarioId);
        return parseResponseAsMap(response);
    }

    public Webhook registerWebhook(String appId, String url, List<String> events, String secret, Map<String, String> headers) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        body.put("app_id", appId);
        body.put("url", url);
        body.put("events", events);
        if (secret != null) body.put("secret", secret);
        if (headers != null) body.put("headers", headers);

        String response = post(getApiPrefix() + "/captcha/webhook/register", body);
        return parseResponse(response, Webhook.class);
    }

    public Map<String, Object> listWebhooks(String appId) throws CaptchaXException {
        String endpoint = getApiPrefix() + "/captcha/webhook";
        if (appId != null) {
            endpoint += "?app_id=" + appId;
        }
        String response = get(endpoint);
        return parseResponseAsMap(response);
    }

    public Webhook updateWebhook(String webhookId, String url, String secret, List<String> events, Boolean enabled, Map<String, String> headers) throws CaptchaXException {
        Map<String, Object> body = new HashMap<>();
        if (url != null) body.put("url", url);
        if (secret != null) body.put("secret", secret);
        if (events != null) body.put("events", events);
        if (enabled != null) body.put("enabled", enabled);
        if (headers != null) body.put("headers", headers);

        String response = put(getApiPrefix() + "/captcha/webhook/" + webhookId, body);
        return parseResponse(response, Webhook.class);
    }

    public Map<String, Object> unregisterWebhook(String webhookId) throws CaptchaXException {
        String response = delete(getApiPrefix() + "/captcha/webhook/" + webhookId);
        return parseResponseAsMap(response);
    }

    public String createClientInfo(Map<String, Object> extra) {
        Map<String, Object> info = new HashMap<>();
        info.put("platform", System.getProperty("os.name"));
        info.put("timestamp", System.currentTimeMillis());
        if (extra != null) {
            info.putAll(extra);
        }
        return gson.toJson(info);
    }

    @SuppressWarnings("unchecked")
    private Map<String, Object> parseResponseAsMap(String responseBody) throws CaptchaXException {
        return parseResponse(responseBody, Map.class);
    }
}
