package com.hjtpx.sdk;

public class ServerError extends SDKError {
    public ServerError(int statusCode) {
        super(statusCode, "Server error: " + statusCode);
    }

    public ServerError(int statusCode, String message) {
        super(statusCode, message);
    }

    public ServerError(int statusCode, String message, Exception cause) {
        super(statusCode, message, cause);
    }
}
