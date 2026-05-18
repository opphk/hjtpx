package com.hjtpx.captcha.client;

import com.fasterxml.jackson.core.type.TypeReference;
import com.fasterxml.jackson.databind.ObjectMapper;
import com.hjtpx.captcha.exception.ApiException;
import com.hjtpx.captcha.exception.CaptchaException;
import com.hjtpx.captcha.exception.NetworkException;
import com.hjtpx.captcha.model.*;
import com.hjtpx.captcha.pool.ConnectionPoolManager;
import com.hjtpx.captcha.retry.RetryManager;
import com.hjtpx.captcha.signer.HmacSigner;
import org.apache.http.HttpEntity;
import org.apache.http.HttpStatus;
import org.apache.http.client.methods.*;
import org.apache.http.client.utils.URIBuilder;
import org.apache.http.entity.ContentType;
import org.apache.http.entity.StringEntity;
import org.apache.http.impl.client.CloseableHttpClient;
import org.apache.http.util.EntityUtils;
import org.slf4j.Logger;
import org.slf4j.LoggerFactory;

import java.io.IOException;
import java.net.URI;
import java.net.URISyntaxException;
import java.util.List;
import java.util.Map;

public class CaptchaClient implements AutoCloseable {
    private static final Logger logger = LoggerFactory.getLogger(CaptchaClient.class);
    private static final ObjectMapper objectMapper = new ObjectMapper();

    private final CaptchaClientConfig config;
    private final ConnectionPoolManager connectionPoolManager;
    private final RetryManager retryManager;
    private final HmacSigner signer;
    private String accessToken;

    public CaptchaClient(String baseUrl) {
        this(new CaptchaClientConfig(baseUrl));
    }

    public CaptchaClient(String baseUrl, String apiKey) {
        this(new CaptchaClientConfig(baseUrl, apiKey));
    }

    public CaptchaClient(CaptchaClientConfig config) {
        this.config = config;
        this.connectionPoolManager = new ConnectionPoolManager(config.getConnectionPoolConfig());
        this.retryManager = new RetryManager(config.getRetryConfig());
        this.signer = config.getSecretKey() != null ? new HmacSigner(config.getSecretKey()) : null;
    }

    public SliderCaptchaResponse getSliderCaptcha() {
        return getSliderCaptcha(null, null, null);
    }

    public SliderCaptchaResponse getSliderCaptcha(Integer width, Integer height, Integer tolerance) {
        try {
            URIBuilder builder = new URIBuilder(config.getBaseUrl() + "/api/v1/captcha/slider");
            if (width != null) builder.addParameter("width", width.toString());
            if (height != null) builder.addParameter("height", height.toString());
            if (tolerance != null) builder.addParameter("tolerance", tolerance.toString());

            return executeGet(builder.build(), SliderCaptchaResponse.class);
        } catch (URISyntaxException e) {
            throw new CaptchaException("Invalid URI", e);
        }
    }

    public ClickCaptchaResponse getClickCaptcha() {
        return getClickCaptcha(null, null, null);
    }

    public ClickCaptchaResponse getClickCaptcha(String mode, Boolean shuffle, Integer points) {
        try {
            URIBuilder builder = new URIBuilder(config.getBaseUrl() + "/api/v1/captcha/click");
            if (mode != null) builder.addParameter("mode", mode);
            if (shuffle != null) builder.addParameter("shuffle", shuffle.toString());
            if (points != null) builder.addParameter("points", points.toString());

            return executeGet(builder.build(), ClickCaptchaResponse.class);
        } catch (URISyntaxException e) {
            throw new CaptchaException("Invalid URI", e);
        }
    }

    public RotationCaptchaResponse getRotationCaptcha() {
        return executeGet(config.getBaseUrl() + "/api/v1/captcha/rotation", RotationCaptchaResponse.class);
    }

    public GestureCaptchaResponse getGestureCaptcha() {
        return executeGet(config.getBaseUrl() + "/api/v1/captcha/gesture", GestureCaptchaResponse.class);
    }

    public JigsawCaptchaResponse getJigsawCaptcha() {
        return getJigsawCaptcha(null, null, null);
    }

    public JigsawCaptchaResponse getJigsawCaptcha(Integer width, Integer height, Integer gridSize) {
        try {
            URIBuilder builder = new URIBuilder(config.getBaseUrl() + "/api/v1/captcha/jigsaw");
            if (width != null) builder.addParameter("width", width.toString());
            if (height != null) builder.addParameter("height", height.toString());
            if (gridSize != null) builder.addParameter("grid_size", gridSize.toString());

            return executeGet(builder.build(), JigsawCaptchaResponse.class);
        } catch (URISyntaxException e) {
            throw new CaptchaException("Invalid URI", e);
        }
    }

    public VoiceCaptchaResponse getVoiceCaptcha() {
        return getVoiceCaptcha(null);
    }

    public VoiceCaptchaResponse getVoiceCaptcha(String language) {
        try {
            URIBuilder builder = new URIBuilder(config.getBaseUrl() + "/api/v1/captcha/voice");
            if (language != null) builder.addParameter("language", language);

            return executeGet(builder.build(), VoiceCaptchaResponse.class);
        } catch (URISyntaxException e) {
            throw new CaptchaException("Invalid URI", e);
        }
    }

    public ConnectCaptchaResponse getConnectCaptcha() {
        return executeGet(config.getBaseUrl() + "/api/v1/captcha/connect", ConnectCaptchaResponse.class);
    }

    public ThreeDCaptchaResponse getThreeDCaptcha() {
        return executeGet(config.getBaseUrl() + "/api/v1/captcha/3d", ThreeDCaptchaResponse.class);
    }

    public VerifyCaptchaResponse verifyCaptcha(VerifyCaptchaRequest request) {
        return executePost(config.getBaseUrl() + "/api/v1/captcha/verify", request, VerifyCaptchaResponse.class);
    }

    public VerifyCaptchaResponse verifySliderCaptcha(String sessionId, int x) {
        return verifySliderCaptcha(sessionId, x, null, null);
    }

    public VerifyCaptchaResponse verifySliderCaptcha(String sessionId, int x, Integer y, List<TrajectoryPoint> trajectory) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("slider");
        request.setX(x);
        request.setY(y);
        request.setTrajectory(trajectory);
        return verifyCaptcha(request);
    }

    public VerifyCaptchaResponse verifyClickCaptcha(String sessionId, List<List<Integer>> points) {
        return verifyClickCaptcha(sessionId, points, null);
    }

    public VerifyCaptchaResponse verifyClickCaptcha(String sessionId, List<List<Integer>> points, List<Integer> clickSequence) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("click");
        request.setPoints(points);
        request.setClickSequence(clickSequence);
        return verifyCaptcha(request);
    }

    public VerifyCaptchaResponse verifyRotationCaptcha(String challengeId, int angle) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(challengeId);
        request.setType("rotation");
        request.setAngle(angle);
        return executePost(config.getBaseUrl() + "/api/v1/captcha/rotation/verify", request, VerifyCaptchaResponse.class);
    }

    public VerifyCaptchaResponse verifyGestureCaptcha(String sessionId, List<Integer> pattern) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("gesture");
        request.setPattern(pattern);
        return executePost(config.getBaseUrl() + "/api/v1/captcha/gesture/verify", request, VerifyCaptchaResponse.class);
    }

    public VerifyCaptchaResponse verifyJigsawCaptcha(String sessionId, List<JigsawPiece> pieces) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("jigsaw");
        request.setPieces(pieces);
        return executePost(config.getBaseUrl() + "/api/v1/captcha/jigsaw/verify", request, VerifyCaptchaResponse.class);
    }

    public VerifyCaptchaResponse verifyVoiceCaptcha(String sessionId, String answer) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("voice");
        request.setAnswer(answer);
        return verifyCaptcha(request);
    }

    public VerifyCaptchaResponse verifyConnectCaptcha(String sessionId, List<List<Integer>> connections) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("connect");
        request.setConnections(connections);
        return verifyCaptcha(request);
    }

    public VerifyCaptchaResponse verifyThreeDCaptcha(String sessionId, List<Double> targetPosition) {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId(sessionId);
        request.setType("3d");
        request.setTargetPosition(targetPosition);
        return verifyCaptcha(request);
    }

    public LoginResponse login(String username, String password) {
        return login(username, password, null);
    }

    public LoginResponse login(String username, String password, String captchaToken) {
        LoginRequest request = new LoginRequest();
        request.setUsername(username);
        request.setPassword(password);
        request.setCaptchaToken(captchaToken);

        LoginResponse response = executePost(config.getBaseUrl() + "/api/v1/auth/login", request, LoginResponse.class);
        this.accessToken = response.getAccessToken();
        return response;
    }

    public void logout() {
        executePost(config.getBaseUrl() + "/api/v1/auth/logout", null, Void.class);
        this.accessToken = null;
    }

    public String getDetectionScript() {
        return getDetectionScript(null);
    }

    public String getDetectionScript(String callback) {
        try {
            URIBuilder builder = new URIBuilder(config.getBaseUrl() + "/api/v1/detect/script");
            if (callback != null) builder.addParameter("callback", callback);

            HttpGet request = new HttpGet(builder.build());
            addHeaders(request);

            return retryManager.execute(() -> {
                try (CloseableHttpClient httpClient = connectionPoolManager.getHttpClient();
                     CloseableHttpResponse response = httpClient.execute(request)) {
                    int statusCode = response.getStatusLine().getStatusCode();
                    HttpEntity entity = response.getEntity();
                    String content = entity != null ? EntityUtils.toString(entity, "UTF-8") : "";
                    EntityUtils.consume(entity);

                    if (statusCode != HttpStatus.SC_OK) {
                        throw new ApiException("Failed to get detection script: " + content, String.valueOf(statusCode));
                    }

                    return content;
                } catch (IOException e) {
                    throw new NetworkException("Network error", e);
                }
            });
        } catch (Exception e) {
            if (e instanceof CaptchaException) {
                throw (CaptchaException) e;
            }
            throw new CaptchaException("Failed to get detection script", e);
        }
    }

    public Map<String, Object> submitDetection(Map<String, Object> data) {
        TypeReference<ApiResponse<Map<String, Object>>> typeRef = new TypeReference<ApiResponse<Map<String, Object>>>() {};
        ApiResponse<Map<String, Object>> apiResponse = executePost(config.getBaseUrl() + "/api/v1/detect/submit", data, typeRef);
        return apiResponse.getData();
    }

    public Map<String, Object> checkEnvironment(Map<String, Object> data) {
        TypeReference<ApiResponse<Map<String, Object>>> typeRef = new TypeReference<ApiResponse<Map<String, Object>>>() {};
        ApiResponse<Map<String, Object>> apiResponse = executePost(config.getBaseUrl() + "/api/v1/detect/check", data, typeRef);
        return apiResponse.getData();
    }

    private <T> T executeGet(String url, Class<T> responseType) {
        return executeGet(URI.create(url), responseType);
    }

    private <T> T executeGet(URI uri, Class<T> responseType) {
        HttpGet request = new HttpGet(uri);
        return executeRequest(request, responseType);
    }

    private <T> T executePost(String url, Object body, Class<T> responseType) {
        HttpPost request = new HttpPost(url);
        if (body != null) {
            try {
                String json = objectMapper.writeValueAsString(body);
                request.setEntity(new StringEntity(json, ContentType.APPLICATION_JSON));
            } catch (Exception e) {
                throw new CaptchaException("Failed to serialize request body", e);
            }
        }
        return executeRequest(request, responseType);
    }

    private <T> T executePost(String url, Object body, TypeReference<ApiResponse<T>> typeRef) {
        HttpPost request = new HttpPost(url);
        if (body != null) {
            try {
                String json = objectMapper.writeValueAsString(body);
                request.setEntity(new StringEntity(json, ContentType.APPLICATION_JSON));
            } catch (Exception e) {
                throw new CaptchaException("Failed to serialize request body", e);
            }
        }
        return executeRequestWithTypeRef(request, typeRef);
    }

    private <T> T executeRequest(HttpUriRequest request, Class<T> responseType) {
        addHeaders(request);

        try {
            return retryManager.execute(() -> {
                try (CloseableHttpClient httpClient = connectionPoolManager.getHttpClient();
                     CloseableHttpResponse response = httpClient.execute(request)) {
                    int statusCode = response.getStatusLine().getStatusCode();
                    HttpEntity entity = response.getEntity();
                    String content = entity != null ? EntityUtils.toString(entity, "UTF-8") : "";
                    EntityUtils.consume(entity);

                    if (statusCode != HttpStatus.SC_OK) {
                        throw new ApiException("API request failed: " + content, String.valueOf(statusCode));
                    }

                    if (responseType == Void.class) {
                        return null;
                    }

                    TypeReference<ApiResponse<T>> typeRef = new TypeReference<ApiResponse<T>>() {};
                    ApiResponse<T> apiResponse = objectMapper.readValue(content, typeRef);

                    if (!apiResponse.isSuccess()) {
                        throw new ApiException(apiResponse.getMessage(), String.valueOf(apiResponse.getCode()));
                    }

                    return apiResponse.getData();
                } catch (IOException e) {
                    throw new NetworkException("Network error", e);
                }
            });
        } catch (Exception e) {
            if (e instanceof CaptchaException) {
                throw (CaptchaException) e;
            }
            throw new CaptchaException("Request failed", e);
        }
    }

    private <T> T executeRequestWithTypeRef(HttpUriRequest request, TypeReference<ApiResponse<T>> typeRef) {
        addHeaders(request);

        try {
            return retryManager.execute(() -> {
                try (CloseableHttpClient httpClient = connectionPoolManager.getHttpClient();
                     CloseableHttpResponse response = httpClient.execute(request)) {
                    int statusCode = response.getStatusLine().getStatusCode();
                    HttpEntity entity = response.getEntity();
                    String content = entity != null ? EntityUtils.toString(entity, "UTF-8") : "";
                    EntityUtils.consume(entity);

                    if (statusCode != HttpStatus.SC_OK) {
                        throw new ApiException("API request failed: " + content, String.valueOf(statusCode));
                    }

                    ApiResponse<T> apiResponse = objectMapper.readValue(content, typeRef);

                    if (!apiResponse.isSuccess()) {
                        throw new ApiException(apiResponse.getMessage(), String.valueOf(apiResponse.getCode()));
                    }

                    return apiResponse.getData();
                } catch (IOException e) {
                    throw new NetworkException("Network error", e);
                }
            });
        } catch (Exception e) {
            if (e instanceof CaptchaException) {
                throw (CaptchaException) e;
            }
            throw new CaptchaException("Request failed", e);
        }
    }

    private void addHeaders(HttpUriRequest request) {
        request.setHeader("Content-Type", "application/json");
        request.setHeader("User-Agent", "HJTPX-Captcha-Java-SDK/1.0.0");

        if (config.getApiKey() != null) {
            request.setHeader("X-API-Key", config.getApiKey());
        }

        if (accessToken != null) {
            request.setHeader("Authorization", "Bearer " + accessToken);
        }

        if (signer != null) {
            long timestamp = System.currentTimeMillis();
            String dataToSign = timestamp + ":" + request.getURI().getPath();
            String signature = signer.sign(dataToSign);
            request.setHeader("X-Timestamp", String.valueOf(timestamp));
            request.setHeader("X-Signature", signature);
        }
    }

    public String getAccessToken() {
        return accessToken;
    }

    public void setAccessToken(String accessToken) {
        this.accessToken = accessToken;
    }

    public CaptchaClientConfig getConfig() {
        return config;
    }

    @Override
    public void close() throws Exception {
        connectionPoolManager.close();
    }
}
