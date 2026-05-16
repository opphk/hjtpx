package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;
import java.util.List;

public class ClickCaptchaResponse {
    @JsonProperty("challenge_id")
    private String challengeId;

    @JsonProperty("background_image")
    private String backgroundImage;

    @JsonProperty("target_position")
    private List<Integer> targetPosition;

    @JsonProperty("target_index")
    private int targetIndex;

    @JsonProperty("icon_positions")
    private List<List<Integer>> iconPositions;

    public ClickCaptchaResponse() {}

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

    public List<Integer> getTargetPosition() {
        return targetPosition;
    }

    public void setTargetPosition(List<Integer> targetPosition) {
        this.targetPosition = targetPosition;
    }

    public int getTargetIndex() {
        return targetIndex;
    }

    public void setTargetIndex(int targetIndex) {
        this.targetIndex = targetIndex;
    }

    public List<List<Integer>> getIconPositions() {
        return iconPositions;
    }

    public void setIconPositions(List<List<Integer>> iconPositions) {
        this.iconPositions = iconPositions;
    }

    @Override
    public String toString() {
        return "ClickCaptchaResponse{" +
                "challengeId='" + challengeId + '\'' +
                ", targetIndex=" + targetIndex +
                ", iconPositions=" + iconPositions +
                '}';
    }
}
