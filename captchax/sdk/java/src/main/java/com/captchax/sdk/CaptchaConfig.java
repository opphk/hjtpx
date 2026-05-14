package com.captchax.sdk;

import com.google.gson.annotations.SerializedName;

import java.util.List;
import java.util.Map;

public class CaptchaConfig {
    private String baseUrl;
    private String appId;
    private int timeout;
    private int retryTimes;
    private ApiVersion apiVersion;

    public CaptchaConfig(String baseUrl) {
        this.baseUrl = baseUrl;
        this.timeout = 10000;
        this.retryTimes = 3;
        this.apiVersion = ApiVersion.V1;
    }

    public CaptchaConfig(String baseUrl, String appId) {
        this(baseUrl);
        this.appId = appId;
    }

    public CaptchaConfig(Builder builder) {
        this.baseUrl = builder.baseUrl;
        this.appId = builder.appId;
        this.timeout = builder.timeout;
        this.retryTimes = builder.retryTimes;
        this.apiVersion = builder.apiVersion;
    }

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

    public int getTimeout() {
        return timeout;
    }

    public void setTimeout(int timeout) {
        this.timeout = timeout;
    }

    public int getRetryTimes() {
        return retryTimes;
    }

    public void setRetryTimes(int retryTimes) {
        this.retryTimes = retryTimes;
    }

    public ApiVersion getApiVersion() {
        return apiVersion;
    }

    public void setApiVersion(ApiVersion apiVersion) {
        this.apiVersion = apiVersion;
    }

    public static class Builder {
        private String baseUrl;
        private String appId;
        private int timeout = 10000;
        private int retryTimes = 3;
        private ApiVersion apiVersion = ApiVersion.V1;

        public Builder baseUrl(String baseUrl) {
            this.baseUrl = baseUrl;
            return this;
        }

        public Builder appId(String appId) {
            this.appId = appId;
            return this;
        }

        public Builder timeout(int timeout) {
            this.timeout = timeout;
            return this;
        }

        public Builder retryTimes(int retryTimes) {
            this.retryTimes = retryTimes;
            return this;
        }

        public Builder apiVersion(ApiVersion apiVersion) {
            this.apiVersion = apiVersion;
            return this;
        }

        public CaptchaConfig build() {
            return new CaptchaConfig(this);
        }
    }
}
