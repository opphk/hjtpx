package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VerifyCaptchaRequest {
    @JsonProperty("challenge_id")
    private String challengeId;

    private String action;

    private Object data;

    public VerifyCaptchaRequest() {}

    public VerifyCaptchaRequest(String challengeId, String action, Object data) {
        this.challengeId = challengeId;
        this.action = action;
        this.data = data;
    }

    public String getChallengeId() {
        return challengeId;
    }

    public void setChallengeId(String challengeId) {
        this.challengeId = challengeId;
    }

    public String getAction() {
        return action;
    }

    public void setAction(String action) {
        this.action = action;
    }

    public Object getData() {
        return data;
    }

    public void setData(Object data) {
        this.data = data;
    }
}
