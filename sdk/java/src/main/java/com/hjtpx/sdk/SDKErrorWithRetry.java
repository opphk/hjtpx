package com.hjtpx.sdk;

public class SDKErrorWithRetry extends SDKError {
    private final Integer retryAfter;

    public SDKErrorWithRetry(int code, String message, Integer retryAfter) {
        this(code, message, retryAfter, null);
    }

    public SDKErrorWithRetry(int code, String message, Integer retryAfter, Exception cause) {
        super(code, message, cause);
        this.retryAfter = retryAfter;
    }

    @Override
    public Integer getRetryAfter() {
        return retryAfter;
    }

    @Override
    public String toString() {
        return "SDKErrorWithRetry{" +
                "code=" + getCode() +
                ", message='" + getMessage() + '\'' +
                ", retryAfter=" + retryAfter +
                '}';
    }
}
