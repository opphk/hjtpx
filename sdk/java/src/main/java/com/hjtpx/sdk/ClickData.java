package com.hjtpx.sdk;

public class ClickData {
    private int x;
    private int y;
    private long duration;

    public ClickData() {}

    public ClickData(int x, int y, long duration) {
        this.x = x;
        this.y = y;
        this.duration = duration;
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

    public long getDuration() {
        return duration;
    }

    public void setDuration(long duration) {
        this.duration = duration;
    }

    @Override
    public String toString() {
        return "ClickData{" +
                "x=" + x +
                ", y=" + y +
                ", duration=" + duration +
                '}';
    }
}
