package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class RotationCaptchaResponse {
    @JsonProperty("challenge_id")
    private String challengeId;

    @JsonProperty("image_url")
    private String imageUrl;

    public String getChallengeId() {
        return challengeId;
    }

    public void setChallengeId(String challengeId) {
        this.challengeId = challengeId;
    }

    public String getImageUrl() {
        return imageUrl;
    }

    public void setImageUrl(String imageUrl) {
        this.imageUrl = imageUrl;
    }
}
