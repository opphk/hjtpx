package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class SliderCaptchaResponse {
    @JsonProperty("challenge_id")
    private String challengeId;

    @JsonProperty("background_image")
    private String backgroundImage;

    @JsonProperty("slider_image")
    private String sliderImage;

    @JsonProperty("slider_width")
    private int sliderWidth;

    @JsonProperty("slider_height")
    private int sliderHeight;

    public SliderCaptchaResponse() {}

    public String getChallengeId() {
        return challengeId;
    }

    public void setChallengeId(String challengeId) {
        this.challengeId = challengeId;
    }

    public String getBackgroundImage() {
        return backgroundImage;
    }

    public void setBackgroundImage(String backgroundImage) {
        this.backgroundImage = backgroundImage;
    }

    public String getSliderImage() {
        return sliderImage;
    }

    public void setSliderImage(String sliderImage) {
        this.sliderImage = sliderImage;
    }

    public int getSliderWidth() {
        return sliderWidth;
    }

    public void setSliderWidth(int sliderWidth) {
        this.sliderWidth = sliderWidth;
    }

    public int getSliderHeight() {
        return sliderHeight;
    }

    public void setSliderHeight(int sliderHeight) {
        this.sliderHeight = sliderHeight;
    }

    @Override
    public String toString() {
        return "SliderCaptchaResponse{" +
                "challengeId='" + challengeId + '\'' +
                ", sliderWidth=" + sliderWidth +
                ", sliderHeight=" + sliderHeight +
                '}';
    }
}
