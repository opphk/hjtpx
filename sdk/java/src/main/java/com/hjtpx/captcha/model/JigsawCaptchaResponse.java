package com.hjtpx.captcha.model;

import com.fasterxml.jackson.annotation.JsonProperty;

import java.util.List;

public class JigsawCaptchaResponse {
    @JsonProperty("session_id")
    private String sessionId;

    @JsonProperty("image_url")
    private String imageUrl;

    @JsonProperty("pieces")
    private List<JigsawPiece> pieces;

    @JsonProperty("piece_images")
    private List<String> pieceImages;

    @JsonProperty("grid_size")
    private int gridSize;

    @JsonProperty("piece_width")
    private int pieceWidth;

    @JsonProperty("piece_height")
    private int pieceHeight;

    @JsonProperty("image_width")
    private int imageWidth;

    @JsonProperty("image_height")
    private int imageHeight;

    public String getSessionId() {
        return sessionId;
    }

    public void setSessionId(String sessionId) {
        this.sessionId = sessionId;
    }

    public String getImageUrl() {
        return imageUrl;
    }

    public void setImageUrl(String imageUrl) {
        this.imageUrl = imageUrl;
    }

    public List<JigsawPiece> getPieces() {
        return pieces;
    }

    public void setPieces(List<JigsawPiece> pieces) {
        this.pieces = pieces;
    }

    public List<String> getPieceImages() {
        return pieceImages;
    }

    public void setPieceImages(List<String> pieceImages) {
        this.pieceImages = pieceImages;
    }

    public int getGridSize() {
        return gridSize;
    }

    public void setGridSize(int gridSize) {
        this.gridSize = gridSize;
    }

    public int getPieceWidth() {
        return pieceWidth;
    }

    public void setPieceWidth(int pieceWidth) {
        this.pieceWidth = pieceWidth;
    }

    public int getPieceHeight() {
        return pieceHeight;
    }

    public void setPieceHeight(int pieceHeight) {
        this.pieceHeight = pieceHeight;
    }

    public int getImageWidth() {
        return imageWidth;
    }

    public void setImageWidth(int imageWidth) {
        this.imageWidth = imageWidth;
    }

    public int getImageHeight() {
        return imageHeight;
    }

    public void setImageHeight(int imageHeight) {
        this.imageHeight = imageHeight;
    }
}
