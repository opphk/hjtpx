package com.hjtpx.captcha.exception;

public class AuthenticationException extends CaptchaException {
    public AuthenticationException(String message) {
        super(message, "AUTHENTICATION_ERROR", false);
    }

    public AuthenticationException(String message, Throwable cause) {
        super(message, "AUTHENTICATION_ERROR", false, cause);
    }
}
