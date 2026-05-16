package com.hjtpx.sdk;

public class NetworkError extends SDKError {
    public NetworkError(String message) {
        super(0, message);
    }

    public NetworkError(String message, Exception cause) {
        super(0, message, cause);
    }
}
