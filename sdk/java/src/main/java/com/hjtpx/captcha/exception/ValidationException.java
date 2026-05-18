package com.hjtpx.captcha.exception;

public class ValidationException extends CaptchaException {
    public ValidationException(String message) {
        super(message, "VALIDATION_ERROR", false);
    }

    public ValidationException(String message, Throwable cause) {
        super(message, "VALIDATION_ERROR", false, cause);
    }
}
