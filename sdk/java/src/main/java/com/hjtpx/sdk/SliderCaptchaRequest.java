package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SliderCaptchaRequest {
    private int width = 360;
    private int height = 220;

    public SliderCaptchaRequest() {}

    public SliderCaptchaRequest(int width, int height) {
        this.width = width;
        this.height = height;
    }

    public int getWidth() {
        return width;
    }

    public void setWidth(int width) {
        this.width = width;
    }

    public int getHeight() {
        return height;
    }

    public void setHeight(int height) {
        this.height = height;
    }
}
