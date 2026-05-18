package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

public class JigsawPiece {
    @JsonProperty("index")
    private int index;

    @JsonProperty("original_x")
    private int originalX;

    @JsonProperty("original_y")
    private int originalY;

    @JsonProperty("current_x")
    private int currentX;

    @JsonProperty("current_y")
    private int currentY;

    @JsonProperty("width")
    private int width;

    @JsonProperty("height")
    private int height;

    @JsonProperty("rotation")
    private int rotation;

    public JigsawPiece() {
    }

    public JigsawPiece(int index, int originalX, int originalY, int currentX, int currentY, int width, int height) {
        this.index = index;
        this.originalX = originalX;
        this.originalY = originalY;
        this.currentX = currentX;
        this.currentY = currentY;
        this.width = width;
        this.height = height;
        this.rotation = 0;
    }

    public int getIndex() {
        return index;
    }

    public void setIndex(int index) {
        this.index = index;
    }

    public int getOriginalX() {
        return originalX;
    }

    public void setOriginalX(int originalX) {
        this.originalX = originalX;
    }

    public int getOriginalY() {
        return originalY;
    }

    public void setOriginalY(int originalY) {
        this.originalY = originalY;
    }

    public int getCurrentX() {
        return currentX;
    }

    public void setCurrentX(int currentX) {
        this.currentX = currentX;
    }

    public int getCurrentY() {
        return currentY;
    }

    public void setCurrentY(int currentY) {
        this.currentY = currentY;
    }

    public int getWidth() {
        return width;
    }

    public void setWidth(int width) {
        this.width = width;
    }

    public int getHeight() {
        return height;
    }

    public void setHeight(int height) {
        this.height = height;
    }

    public int getRotation() {
        return rotation;
    }

    public void setRotation(int rotation) {
        this.rotation = rotation;
    }
}
