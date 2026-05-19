package com.hjtpx.captcha.sdk;

public interface CaptchaListener {
    void onCaptchaLoaded(String sessionId);
    void onCaptchaVerified(boolean success);
    void onCaptchaError(String error);
    void onCaptchaExpired();
}
