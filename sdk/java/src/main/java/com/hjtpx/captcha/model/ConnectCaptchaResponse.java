package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

public class ConnectCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("image_url")
    private String imageUrl;

    @JsonProperty("pairs")
    private List<PairItem> pairs;

    public static class PairItem {
        @JsonProperty("left")
        private int left;

        @JsonProperty("right")
        private int right;

        public int getLeft() {
            return left;
        }

        public void setLeft(int left) {
            this.left = left;
        }

        public int getRight() {
            return right;
        }

        public void setRight(int right) {
            this.right = right;
        }
    }

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

    public List<PairItem> getPairs() {
        return pairs;
    }

    public void setPairs(List<PairItem> pairs) {
        this.pairs = pairs;
    }
}
