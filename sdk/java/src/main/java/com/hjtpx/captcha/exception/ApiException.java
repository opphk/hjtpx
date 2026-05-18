package com.hjtpx.captcha.exception;

public class ApiException extends CaptchaException {
    public ApiException(String message, String code) {
        super(message, code, isRetryableCode(code));
    }

    public ApiException(String message, String code, Throwable cause) {
        super(message, code, isRetryableCode(code), cause);
    }

    private static boolean isRetryableCode(String code) {
        if (code == null) return false;
        try {
            int statusCode = Integer.parseInt(code);
            return statusCode == 429 || (statusCode >= 500 && statusCode < 600);
        } catch (NumberFormatException e) {
            return false;
        }
    }
}
