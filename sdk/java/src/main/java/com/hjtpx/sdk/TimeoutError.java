package com.hjtpx.sdk;

public class TimeoutError extends SDKError {
    public TimeoutError(String message) {
        super(408, message);
    }

    public TimeoutError(String message, Exception cause) {
        super(408, message, cause);
    }
}
