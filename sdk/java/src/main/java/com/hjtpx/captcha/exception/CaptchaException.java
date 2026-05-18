package com.hjtpx.captcha.exception;

public class CaptchaException extends RuntimeException {
    private final String code;
    private final boolean retryable;

    public CaptchaException(String message) {
        super(message);
        this.code = null;
        this.retryable = false;
    }

    public CaptchaException(String message, String code) {
        super(message);
        this.code = code;
        this.retryable = false;
    }

    public CaptchaException(String message, String code, boolean retryable) {
        super(message);
        this.code = code;
        this.retryable = retryable;
    }

    public CaptchaException(String message, Throwable cause) {
        super(message, cause);
        this.code = null;
        this.retryable = false;
    }

    public CaptchaException(String message, String code, Throwable cause) {
        super(message, cause);
        this.code = code;
        this.retryable = false;
    }

    public CaptchaException(String message, String code, boolean retryable, Throwable cause) {
        super(message, cause);
        this.code = code;
        this.retryable = retryable;
    }

    public String getCode() {
        return code;
    }

    public boolean isRetryable() {
        return retryable;
    }
}
