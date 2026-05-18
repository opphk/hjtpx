package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class GestureCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("pattern")
    private String pattern;

    @JsonProperty("grid_size")
    private int gridSize;

    @JsonProperty("hint")
    private String hint;

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getPattern() {
        return pattern;
    }

    public void setPattern(String pattern) {
        this.pattern = pattern;
    }

    public int getGridSize() {
        return gridSize;
    }

    public void setGridSize(int gridSize) {
        this.gridSize = gridSize;
    }

    public String getHint() {
        return hint;
    }

    public void setHint(String hint) {
        this.hint = hint;
    }
}
