package com.hjtpx.captcha.exception;

public class NetworkException extends CaptchaException {
    public NetworkException(String message) {
        super(message, "NETWORK_ERROR", true);
    }

    public NetworkException(String message, Throwable cause) {
        super(message, "NETWORK_ERROR", true, cause);
    }
}
