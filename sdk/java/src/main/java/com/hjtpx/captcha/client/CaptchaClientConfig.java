package com.hjtpx.captcha.client;

import com.hjtpx.captcha.pool.ConnectionPoolConfig;
import com.hjtpx.captcha.retry.RetryConfig;

public class CaptchaClientConfig {
    private String baseUrl;
    private String apiKey;
    private String secretKey;
    private ConnectionPoolConfig connectionPoolConfig;
    private RetryConfig retryConfig;

    public CaptchaClientConfig() {
        this.connectionPoolConfig = new ConnectionPoolConfig();
        this.retryConfig = new RetryConfig();
    }

    public CaptchaClientConfig(String baseUrl) {
        this();
        this.baseUrl = baseUrl;
    }

    public CaptchaClientConfig(String baseUrl, String apiKey) {
        this(baseUrl);
        this.apiKey = apiKey;
    }

    public CaptchaClientConfig(String baseUrl, String apiKey, String secretKey) {
        this(baseUrl, apiKey);
        this.secretKey = secretKey;
    }

    public String getBaseUrl() {
        return baseUrl;
    }

    public void setBaseUrl(String baseUrl) {
        this.baseUrl = baseUrl;
    }

    public String getApiKey() {
        return apiKey;
    }

    public void setApiKey(String apiKey) {
        this.apiKey = apiKey;
    }

    public String getSecretKey() {
        return secretKey;
    }

    public void setSecretKey(String secretKey) {
        this.secretKey = secretKey;
    }

    public ConnectionPoolConfig getConnectionPoolConfig() {
        return connectionPoolConfig;
    }

    public void setConnectionPoolConfig(ConnectionPoolConfig connectionPoolConfig) {
        this.connectionPoolConfig = connectionPoolConfig;
    }

    public RetryConfig getRetryConfig() {
        return retryConfig;
    }

    public void setRetryConfig(RetryConfig retryConfig) {
        this.retryConfig = retryConfig;
    }
}
