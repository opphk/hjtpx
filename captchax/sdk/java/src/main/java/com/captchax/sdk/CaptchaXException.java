package com.captchax.sdk;

public class CaptchaXException extends Exception {
    private final int code;
    private final int statusCode;
    private final Object details;

    public CaptchaXException(String message) {
        super(message);
        this.code = 500;
        this.statusCode = 500;
        this.details = null;
    }

    public CaptchaXException(String message, int code, int statusCode) {
        super(message);
        this.code = code;
        this.statusCode = statusCode;
        this.details = null;
    }

    public CaptchaXException(String message, int code, int statusCode, Object details) {
        super(message);
        this.code = code;
        this.statusCode = statusCode;
        this.details = details;
    }

    public CaptchaXException(String message, Throwable cause) {
        super(message, cause);
        this.code = 500;
        this.statusCode = 500;
        this.details = null;
    }

    public int getCode() {
        return code;
    }

    public int getStatusCode() {
        return statusCode;
    }

    public Object getDetails() {
        return details;
    }

    @Override
    public String toString() {
        return "CaptchaXException{" +
                "code=" + code +
                ", statusCode=" + statusCode +
                ", message='" + getMessage() + '\'' +
                '}';
    }
}
