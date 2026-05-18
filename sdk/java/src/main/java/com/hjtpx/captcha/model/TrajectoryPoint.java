package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class TrajectoryPoint {
    @JsonProperty("x")
    private int x;

    @JsonProperty("y")
    private int y;

    @JsonProperty("t")
    private long timestamp;

    public TrajectoryPoint() {
    }

    public TrajectoryPoint(int x, int y, long timestamp) {
        this.x = x;
        this.y = y;
        this.timestamp = timestamp;
    }

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
}
