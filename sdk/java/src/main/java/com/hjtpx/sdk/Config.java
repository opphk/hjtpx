package com.hjtpx.sdk;

public class Config {
    public static final String DEFAULT_API_ENDPOINT = "http://localhost:8080";

    private String baseUrl = DEFAULT_API_ENDPOINT;
    private String appId = "";
    private String appSecret = "";
    private int timeout = 30000;
    private int maxRetries = 3;
    private long retryDelay = 100;
    private int maxIdleConns = 10;
    private int maxOpenConns = 100;
    private boolean debugMode = false;

    public Config() {}

    public String getBaseUrl() {
        return baseUrl;
    }

    public void setBaseUrl(String baseUrl) {
        this.baseUrl = baseUrl;
    }

    public String getAppId() {
        return appId;
    }

    public void setAppId(String appId) {
        this.appId = appId;
    }

    public String getAppSecret() {
        return appSecret;
    }

    public void setAppSecret(String appSecret) {
        this.appSecret = appSecret;
    }

    public int getTimeout() {
        return timeout;
    }

    public void setTimeout(int timeout) {
        this.timeout = timeout;
    }

    public int getMaxRetries() {
        return maxRetries;
    }

    public void setMaxRetries(int maxRetries) {
        this.maxRetries = maxRetries;
    }

    public long getRetryDelay() {
        return retryDelay;
    }

    public void setRetryDelay(long retryDelay) {
        this.retryDelay = retryDelay;
    }

    public int getMaxIdleConns() {
        return maxIdleConns;
    }

    public void setMaxIdleConns(int maxIdleConns) {
        this.maxIdleConns = maxIdleConns;
    }

    public int getMaxOpenConns() {
        return maxOpenConns;
    }

    public void setMaxOpenConns(int maxOpenConns) {
        this.maxOpenConns = maxOpenConns;
    }

    public boolean isDebugMode() {
        return debugMode;
    }

    public void setDebugMode(boolean debugMode) {
        this.debugMode = debugMode;
    }

    public static Builder builder() {
        return new Builder();
    }

    public static class Builder {
        private final Config config = new Config();

        public Builder baseUrl(String baseUrl) {
            config.setBaseUrl(baseUrl);
            return this;
        }

        public Builder appId(String appId) {
            config.setAppId(appId);
            return this;
        }

        public Builder appSecret(String appSecret) {
            config.setAppSecret(appSecret);
            return this;
        }

        public Builder timeout(int timeout) {
            config.setTimeout(timeout);
            return this;
        }

        public Builder maxRetries(int maxRetries) {
            config.setMaxRetries(maxRetries);
            return this;
        }

        public Builder retryDelay(long retryDelay) {
            config.setRetryDelay(retryDelay);
            return this;
        }

        public Builder maxIdleConns(int maxIdleConns) {
            config.setMaxIdleConns(maxIdleConns);
            return this;
        }

        public Builder maxOpenConns(int maxOpenConns) {
            config.setMaxOpenConns(maxOpenConns);
            return this;
        }

        public Builder debugMode(boolean debugMode) {
            config.setDebugMode(debugMode);
            return this;
        }

        public Config build() {
            return config;
        }
    }
}
