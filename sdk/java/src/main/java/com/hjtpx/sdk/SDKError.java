package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SDKError extends Exception {
    private final int code;
    private final String message;
    private final Exception cause;

    public SDKError(int code, String message) {
        this(code, message, null);
    }

    public SDKError(int code, String message, Exception cause) {
        super(buildMessage(code, message, cause));
        this.code = code;
        this.message = message;
        this.cause = cause;
    }

    private static String buildMessage(int code, String message, Exception cause) {
        StringBuilder sb = new StringBuilder();
        sb.append("SDKError(code=").append(code).append(", message=").append(message).append(")");
        if (cause != null) {
            sb.append(": ").append(cause.getMessage());
        }
        return sb.toString();
    }

    public int getCode() {
        return code;
    }

    public String getMessage() {
        return message;
    }

    public Exception getCauseException() {
        return cause;
    }

    public boolean isRateLimited() {
        return code == 429;
    }

    public boolean isUnauthorized() {
        return code == 401;
    }

    public boolean isServerError() {
        return code >= 500;
    }

    public boolean isInvalidParams() {
        return code == 400;
    }

    public Integer getRetryAfter() {
        return null;
    }

    @Override
    public String toString() {
        return "SDKError{" +
                "code=" + code +
                ", message='" + message + '\'' +
                '}';
    }
}
