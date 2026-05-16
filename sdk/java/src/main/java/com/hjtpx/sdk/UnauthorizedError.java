package com.hjtpx.sdk;

public class UnauthorizedError extends SDKError {
    public UnauthorizedError() {
        super(401, "Unauthorized");
    }

    public UnauthorizedError(String message) {
        super(401, message);
    }
}
