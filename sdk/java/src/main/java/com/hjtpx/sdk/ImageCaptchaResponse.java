package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class ImageCaptchaResponse {
    @JsonProperty("challenge_id")
    private String challengeId;

    private String image;

    public ImageCaptchaResponse() {}

    public ImageCaptchaResponse(String challengeId, String image) {
        this.challengeId = challengeId;
        this.image = image;
    }

    public String getChallengeId() {
        return challengeId;
    }

    public void setChallengeId(String challengeId) {
        this.challengeId = challengeId;
    }

    public String getImage() {
        return image;
    }

    public void setImage(String image) {
        this.image = image;
    }

    @Override
    public String toString() {
        return "ImageCaptchaResponse{" +
                "challengeId='" + challengeId + '\'' +
                ", image='" + (image != null && image.length() > 50 ? image.substring(0, 50) + "..." : image) + '\'' +
                '}';
    }
}
