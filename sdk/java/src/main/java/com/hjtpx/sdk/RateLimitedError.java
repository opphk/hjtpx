package com.hjtpx.sdk;

public class RateLimitedError extends SDKErrorWithRetry {
    public RateLimitedError(Integer retryAfter) {
        super(429, "Rate limited", retryAfter);
    }

    public RateLimitedError(String message, Integer retryAfter) {
        super(429, message, retryAfter);
    }

    public RateLimitedError(String message, Integer retryAfter, Exception cause) {
        super(429, message, retryAfter, cause);
    }
}
