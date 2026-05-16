package com.hjtpx.sdk;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VerifyCaptchaResponse {
    private boolean success;
    private double score;
    private String message;

    @JsonProperty("risk_level")
    private String riskLevel;

    public VerifyCaptchaResponse() {}

    public boolean isSuccess() {
        return success;
    }

    public void setSuccess(boolean success) {
        this.success = success;
    }

    public double getScore() {
        return score;
    }

    public void setScore(double score) {
        this.score = score;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }

    public String getRiskLevel() {
        return riskLevel;
    }

    public void setRiskLevel(String riskLevel) {
        this.riskLevel = riskLevel;
    }

    @Override
    public String toString() {
        return "VerifyCaptchaResponse{" +
                "success=" + success +
                ", score=" + score +
                ", message='" + message + '\'' +
                ", riskLevel='" + riskLevel + '\'' +
                '}';
    }
}
