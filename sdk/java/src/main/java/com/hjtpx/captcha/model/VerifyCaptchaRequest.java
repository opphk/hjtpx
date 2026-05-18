package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

public class VerifyCaptchaRequest {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("type")
    private String type;

    @JsonProperty("x")
    private Integer x;

    @JsonProperty("y")
    private Integer y;

    @JsonProperty("trajectory")
    private List<TrajectoryPoint> trajectory;

    @JsonProperty("points")
    private List<List<Integer>> points;

    @JsonProperty("click_sequence")
    private List<Integer> clickSequence;

    @JsonProperty("angle")
    private Integer angle;

    @JsonProperty("pattern")
    private List<Integer> pattern;

    @JsonProperty("pieces")
    private List<JigsawPiece> pieces;

    @JsonProperty("answer")
    private String answer;

    @JsonProperty("connections")
    private List<List<Integer>> connections;

    @JsonProperty("target_position")
    private List<Double> targetPosition;

    @JsonProperty("behavior_data")
    private List<BehaviorDataPoint> behaviorData;

    public static class BehaviorDataPoint {
        @JsonProperty("x")
        private int x;

        @JsonProperty("y")
        private int y;

        @JsonProperty("timestamp")
        private long timestamp;

        @JsonProperty("event")
        private String event;

        public int getX() {
            return x;
        }

        public void setX(int x) {
            this.x = x;
        }

        public int getY() {
            return y;
        }

        public void setY(int y) {
            this.y = y;
        }

        public long getTimestamp() {
            return timestamp;
        }

        public void setTimestamp(long timestamp) {
            this.timestamp = timestamp;
        }

        public String getEvent() {
            return event;
        }

        public void setEvent(String event) {
            this.event = event;
        }
    }

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getType() {
        return type;
    }

    public void setType(String type) {
        this.type = type;
    }

    public Integer getX() {
        return x;
    }

    public void setX(Integer x) {
        this.x = x;
    }

    public Integer getY() {
        return y;
    }

    public void setY(Integer y) {
        this.y = y;
    }

    public List<TrajectoryPoint> getTrajectory() {
        return trajectory;
    }

    public void setTrajectory(List<TrajectoryPoint> trajectory) {
        this.trajectory = trajectory;
    }

    public List<List<Integer>> getPoints() {
        return points;
    }

    public void setPoints(List<List<Integer>> points) {
        this.points = points;
    }

    public List<Integer> getClickSequence() {
        return clickSequence;
    }

    public void setClickSequence(List<Integer> clickSequence) {
        this.clickSequence = clickSequence;
    }

    public Integer getAngle() {
        return angle;
    }

    public void setAngle(Integer angle) {
        this.angle = angle;
    }

    public List<Integer> getPattern() {
        return pattern;
    }

    public void setPattern(List<Integer> pattern) {
        this.pattern = pattern;
    }

    public List<JigsawPiece> getPieces() {
        return pieces;
    }

    public void setPieces(List<JigsawPiece> pieces) {
        this.pieces = pieces;
    }

    public String getAnswer() {
        return answer;
    }

    public void setAnswer(String answer) {
        this.answer = answer;
    }

    public List<List<Integer>> getConnections() {
        return connections;
    }

    public void setConnections(List<List<Integer>> connections) {
        this.connections = connections;
    }

    public List<Double> getTargetPosition() {
        return targetPosition;
    }

    public void setTargetPosition(List<Double> targetPosition) {
        this.targetPosition = targetPosition;
    }

    public List<BehaviorDataPoint> getBehaviorData() {
        return behaviorData;
    }

    public void setBehaviorData(List<BehaviorDataPoint> behaviorData) {
        this.behaviorData = behaviorData;
    }
}
