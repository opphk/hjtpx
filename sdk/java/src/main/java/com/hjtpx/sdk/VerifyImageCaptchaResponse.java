package com.hjtpx.sdk;

public class VerifyImageCaptchaResponse {
    private boolean success;

    public VerifyImageCaptchaResponse() {}

    public VerifyImageCaptchaResponse(boolean success) {
        this.success = success;
    }

    public boolean isSuccess() {
        return success;
    }

    public void setSuccess(boolean success) {
        this.success = success;
    }

    @Override
    public String toString() {
        return "VerifyImageCaptchaResponse{" +
                "success=" + success +
                '}';
    }
}
