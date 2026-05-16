package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ClickCaptchaRequest {
    private int width = 360;
    private int height = 220;
    private int iconCount = 4;

    public ClickCaptchaRequest() {}

    public ClickCaptchaRequest(int width, int height, int iconCount) {
        this.width = width;
        this.height = height;
        this.iconCount = iconCount;
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

    public int getIconCount() {
        return iconCount;
    }

    public void setIconCount(int iconCount) {
        this.iconCount = iconCount;
    }
}
