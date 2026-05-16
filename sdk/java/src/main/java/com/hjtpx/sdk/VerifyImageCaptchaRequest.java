package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VerifyImageCaptchaRequest {
    @JsonProperty("challenge_id")
    private String challengeId;

    private String answer;

    public VerifyImageCaptchaRequest() {}

    public VerifyImageCaptchaRequest(String challengeId, String answer) {
        this.challengeId = challengeId;
        this.answer = answer;
    }

    public String getChallengeId() {
        return challengeId;
    }

    public void setChallengeId(String challengeId) {
        this.challengeId = challengeId;
    }

    public String getAnswer() {
        return answer;
    }

    public void setAnswer(String answer) {
        this.answer = answer;
    }
}
