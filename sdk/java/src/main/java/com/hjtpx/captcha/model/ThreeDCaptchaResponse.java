package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ThreeDCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("scene_url")
    private String sceneUrl;

    @JsonProperty("target_id")
    private String targetId;

    @JsonProperty("hint")
    private String hint;

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getSceneUrl() {
        return sceneUrl;
    }

    public void setSceneUrl(String sceneUrl) {
        this.sceneUrl = sceneUrl;
    }

    public String getTargetId() {
        return targetId;
    }

    public void setTargetId(String targetId) {
        this.targetId = targetId;
    }

    public String getHint() {
        return hint;
    }

    public void setHint(String hint) {
        this.hint = hint;
    }
}
