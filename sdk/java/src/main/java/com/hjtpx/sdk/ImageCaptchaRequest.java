package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ImageCaptchaRequest {
    private CaptchaType type = CaptchaType.MIXED;
    private int count = 4;
    private String customSet;
    private int noiseMode = 0;
    private int lineMode = 0;

    public ImageCaptchaRequest() {}

    public ImageCaptchaRequest(CaptchaType type, int count) {
        this.type = type;
        this.count = count;
    }

    public CaptchaType getType() {
        return type;
    }

    public void setType(CaptchaType type) {
        this.type = type;
    }

    public int getCount() {
        return count;
    }

    public void setCount(int count) {
        this.count = count;
    }

    public String getCustomSet() {
        return customSet;
    }

    public void setCustomSet(String customSet) {
        this.customSet = customSet;
    }

    public int getNoiseMode() {
        return noiseMode;
    }

    public void setNoiseMode(int noiseMode) {
        this.noiseMode = noiseMode;
    }

    public int getLineMode() {
        return lineMode;
    }

    public void setLineMode(int lineMode) {
        this.lineMode = lineMode;
    }
}
