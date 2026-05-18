package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SliderCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("image_url")
    private String imageUrl;

    @JsonProperty("puzzle_url")
    private String puzzleUrl;

    @JsonProperty("hint_url")
    private String hintUrl;

    @JsonProperty("shape")
    private int shape;

    @JsonProperty("secret_y")
    private int secretY;

    @JsonProperty("image_width")
    private int imageWidth;

    @JsonProperty("image_height")
    private int imageHeight;

    @JsonProperty("tolerance")
    private int tolerance;

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getImageUrl() {
        return imageUrl;
    }

    public void setImageUrl(String imageUrl) {
        this.imageUrl = imageUrl;
    }

    public String getPuzzleUrl() {
        return puzzleUrl;
    }

    public void setPuzzleUrl(String puzzleUrl) {
        this.puzzleUrl = puzzleUrl;
    }

    public String getHintUrl() {
        return hintUrl;
    }

    public void setHintUrl(String hintUrl) {
        this.hintUrl = hintUrl;
    }

    public int getShape() {
        return shape;
    }

    public void setShape(int shape) {
        this.shape = shape;
    }

    public int getSecretY() {
        return secretY;
    }

    public void setSecretY(int secretY) {
        this.secretY = secretY;
    }

    public int getImageWidth() {
        return imageWidth;
    }

    public void setImageWidth(int imageWidth) {
        this.imageWidth = imageWidth;
    }

    public int getImageHeight() {
        return imageHeight;
    }

    public void setImageHeight(int imageHeight) {
        this.imageHeight = imageHeight;
    }

    public int getTolerance() {
        return tolerance;
    }

    public void setTolerance(int tolerance) {
        this.tolerance = tolerance;
    }
}
