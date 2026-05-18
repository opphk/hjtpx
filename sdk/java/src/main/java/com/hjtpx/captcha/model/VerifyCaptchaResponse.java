package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class VerifyCaptchaResponse {
    @JsonProperty("success")
    private boolean success;

    @JsonProperty("message")
    private String message;

    @JsonProperty("remaining_attempts")
    private Integer remainingAttempts;

    @JsonProperty("risk_score")
    private Double riskScore;

    @JsonProperty("captcha_pass")
    private Boolean captchaPass;

    @JsonProperty("fail_reason")
    private String failReason;

    @JsonProperty("trajectory_result")
    private TrajectoryResult trajectoryResult;

    public static class TrajectoryResult {
        @JsonProperty("score")
        private double score;

        @JsonProperty("passed")
        private boolean passed;

        @JsonProperty("reasons")
        private java.util.List<String> reasons;

        public double getScore() {
            return score;
        }

        public void setScore(double score) {
            this.score = score;
        }

        public boolean isPassed() {
            return passed;
        }

        public void setPassed(boolean passed) {
            this.passed = passed;
        }

        public java.util.List<String> getReasons() {
            return reasons;
        }

        public void setReasons(java.util.List<String> reasons) {
            this.reasons = reasons;
        }
    }

    public boolean isSuccess() {
        return success;
    }

    public void setSuccess(boolean success) {
        this.success = success;
    }

    public String getMessage() {
        return message;
    }

    public void setMessage(String message) {
        this.message = message;
    }

    public Integer getRemainingAttempts() {
        return remainingAttempts;
    }

    public void setRemainingAttempts(Integer remainingAttempts) {
        this.remainingAttempts = remainingAttempts;
    }

    public Double getRiskScore() {
        return riskScore;
    }

    public void setRiskScore(Double riskScore) {
        this.riskScore = riskScore;
    }

    public Boolean getCaptchaPass() {
        return captchaPass;
    }

    public void setCaptchaPass(Boolean captchaPass) {
        this.captchaPass = captchaPass;
    }

    public String getFailReason() {
        return failReason;
    }

    public void setFailReason(String failReason) {
        this.failReason = failReason;
    }

    public TrajectoryResult getTrajectoryResult() {
        return trajectoryResult;
    }

    public void setTrajectoryResult(TrajectoryResult trajectoryResult) {
        this.trajectoryResult = trajectoryResult;
    }
}
