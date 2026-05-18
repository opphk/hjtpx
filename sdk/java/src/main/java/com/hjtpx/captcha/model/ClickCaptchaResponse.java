package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

public class ClickCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("image_url")
    private String imageUrl;

    @JsonProperty("hint")
    private String hint;

    @JsonProperty("hint_order")
    private List<Integer> hintOrder;

    @JsonProperty("max_points")
    private int maxPoints;

    @JsonProperty("mode")
    private String mode;

    @JsonProperty("allow_shuffle")
    private boolean allowShuffle;

    @JsonProperty("points")
    private List<List<Integer>> points;

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

    public String getHint() {
        return hint;
    }

    public void setHint(String hint) {
        this.hint = hint;
    }

    public List<Integer> getHintOrder() {
        return hintOrder;
    }

    public void setHintOrder(List<Integer> hintOrder) {
        this.hintOrder = hintOrder;
    }

    public int getMaxPoints() {
        return maxPoints;
    }

    public void setMaxPoints(int maxPoints) {
        this.maxPoints = maxPoints;
    }

    public String getMode() {
        return mode;
    }

    public void setMode(String mode) {
        this.mode = mode;
    }

    public boolean isAllowShuffle() {
        return allowShuffle;
    }

    public void setAllowShuffle(boolean allowShuffle) {
        this.allowShuffle = allowShuffle;
    }

    public List<List<Integer>> getPoints() {
        return points;
    }

    public void setPoints(List<List<Integer>> points) {
        this.points = points;
    }
}
