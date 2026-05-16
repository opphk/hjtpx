package com.hjtpx.sdk;

public class InvalidParamsError extends SDKError {
    public InvalidParamsError(String message) {
        super(400, message);
    }

    public InvalidParamsError(String message, Exception cause) {
        super(400, message, cause);
    }
}
