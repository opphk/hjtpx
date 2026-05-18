package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VoiceCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("audio_url")
    private String audioUrl;

    @JsonProperty("text")
    private String text;

    @JsonProperty("language")
    private String language;

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getAudioUrl() {
        return audioUrl;
    }

    public void setAudioUrl(String audioUrl) {
        this.audioUrl = audioUrl;
    }

    public String getText() {
        return text;
    }

    public void setText(String text) {
        this.text = text;
    }

    public String getLanguage() {
        return language;
    }

    public void setLanguage(String language) {
        this.language = language;
    }
}
