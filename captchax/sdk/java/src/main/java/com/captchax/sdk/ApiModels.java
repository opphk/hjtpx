package com.captchax.sdk;

import com.google.gson.annotations.SerializedName;
import java.util.List;
import java.util.Map;

public class ApiResponse<T> {
    @SerializedName("code")
    private int code;

    @SerializedName("message")
    private String message;

    @SerializedName("data")
    private T data;

    public int getCode() {
        return code;
    }

    public String getMessage() {
        return message;
    }

    public T getData() {
        return data;
    }

    public boolean isSuccess() {
        return code == 200;
    }
}

class CaptchaResponse<T> extends ApiResponse<T> {
}

class SliderCaptchaResult {
    @SerializedName("id")
    private String id;

    @SerializedName("background_b64")
    private String backgroundB64;

    @SerializedName("slider_b64")
    private String sliderB64;

    @SerializedName("target_x")
    private int targetX;

    @SerializedName("target_y")
    private int targetY;

    public String getId() {
        return id;
    }

    public String getBackgroundB64() {
        return backgroundB64;
    }

    public String getSliderB64() {
        return sliderB64;
    }

    public int getTargetX() {
        return targetX;
    }

    public int getTargetY() {
        return targetY;
    }
}

class SliderVerifyResult {
    @SerializedName("success")
    private boolean success;

    @SerializedName("message")
    private String message;

    public boolean isSuccess() {
        return success;
    }

    public String getMessage() {
        return message;
    }
}

class CharPosition {
    @SerializedName("char")
    private String character;

    @SerializedName("x")
    private int x;

    @SerializedName("y")
    private int y;

    public CharPosition() {}

    public CharPosition(String character, int x, int y) {
        this.character = character;
        this.x = x;
        this.y = y;
    }

    public String getCharacter() {
        return character;
    }

    public int getX() {
        return x;
    }

    public int getY() {
        return y;
    }
}

class ClickCaptchaResult {
    @SerializedName("id")
    private String id;

    @SerializedName("image")
    private String image;

    @SerializedName("target_chars")
    private List<String> targetChars;

    @SerializedName("char_positions")
    private List<CharPosition> charPositions;

    public String getId() {
        return id;
    }

    public String getImage() {
        return image;
    }

    public List<String> getTargetChars() {
        return targetChars;
    }

    public List<CharPosition> getCharPositions() {
        return charPositions;
    }
}

class ClickVerifyResult {
    @SerializedName("success")
    private boolean success;

    @SerializedName("score")
    private double score;

    @SerializedName("message")
    private String message;

    public boolean isSuccess() {
        return success;
    }

    public double getScore() {
        return score;
    }

    public String getMessage() {
        return message;
    }
}

class PuzzleCaptchaResult {
    @SerializedName("id")
    private String id;

    @SerializedName("background_b64")
    private String backgroundB64;

    @SerializedName("puzzle_b64")
    private String puzzleB64;

    @SerializedName("target_x")
    private int targetX;

    @SerializedName("target_y")
    private int targetY;

    public String getId() {
        return id;
    }

    public String getBackgroundB64() {
        return backgroundB64;
    }

    public String getPuzzleB64() {
        return puzzleB64;
    }

    public int getTargetX() {
        return targetX;
    }

    public int getTargetY() {
        return targetY;
    }
}

class PuzzleVerifyResult {
    @SerializedName("success")
    private boolean success;

    @SerializedName("message")
    private String message;

    public boolean isSuccess() {
        return success;
    }

    public String getMessage() {
        return message;
    }
}

class Scenario {
    @SerializedName("id")
    private String id;

    @SerializedName("name")
    private String name;

    @SerializedName("description")
    private String description;

    @SerializedName("difficulty")
    private String difficulty;

    @SerializedName("config")
    private Map<String, Object> config;

    @SerializedName("created_at")
    private String createdAt;

    @SerializedName("updated_at")
    private String updatedAt;

    public String getId() {
        return id;
    }

    public String getName() {
        return name;
    }

    public String getDescription() {
        return description;
    }

    public String getDifficulty() {
        return difficulty;
    }

    public Map<String, Object> getConfig() {
        return config;
    }

    public String getCreatedAt() {
        return createdAt;
    }

    public String getUpdatedAt() {
        return updatedAt;
    }
}

class Webhook {
    @SerializedName("id")
    private String id;

    @SerializedName("app_id")
    private String appId;

    @SerializedName("url")
    private String url;

    @SerializedName("secret")
    private String secret;

    @SerializedName("events")
    private List<String> events;

    @SerializedName("headers")
    private Map<String, String> headers;

    @SerializedName("enabled")
    private boolean enabled;

    @SerializedName("created_at")
    private String createdAt;

    @SerializedName("updated_at")
    private String updatedAt;

    public String getId() {
        return id;
    }

    public String getAppId() {
        return appId;
    }

    public String getUrl() {
        return url;
    }

    public String getSecret() {
        return secret;
    }

    public List<String> getEvents() {
        return events;
    }

    public Map<String, String> getHeaders() {
        return headers;
    }

    public boolean isEnabled() {
        return enabled;
    }

    public String getCreatedAt() {
        return createdAt;
    }

    public String getUpdatedAt() {
        return updatedAt;
    }
}

class BatchVerifyItem {
    @SerializedName("captcha_id")
    private String captchaId;

    @SerializedName("type")
    private String type;

    @SerializedName("target_x")
    private int targetX;

    @SerializedName("target_y")
    private Integer targetY;

    @SerializedName("clicks")
    private List<CharPosition> clicks;

    public BatchVerifyItem(String captchaId, String type, int targetX) {
        this.captchaId = captchaId;
        this.type = type;
        this.targetX = targetX;
    }

    public BatchVerifyItem withTargetY(int targetY) {
        this.targetY = targetY;
        return this;
    }

    public BatchVerifyItem withClicks(List<CharPosition> clicks) {
        this.clicks = clicks;
        return this;
    }
}

class BatchVerifyResult {
    @SerializedName("captcha_id")
    private String captchaId;

    @SerializedName("success")
    private boolean success;

    @SerializedName("message")
    private String message;

    @SerializedName("score")
    private Double score;

    public String getCaptchaId() {
        return captchaId;
    }

    public boolean isSuccess() {
        return success;
    }

    public String getMessage() {
        return message;
    }

    public Double getScore() {
        return score;
    }
}

class BatchVerifySummary {
    @SerializedName("total")
    private int total;

    @SerializedName("success")
    private int successCount;

    @SerializedName("failed")
    private int failed;

    @SerializedName("skipped")
    private int skipped;

    public int getTotal() {
        return total;
    }

    public int getSuccessCount() {
        return successCount;
    }

    public int getFailed() {
        return failed;
    }

    public int getSkipped() {
        return skipped;
    }
}

class BatchVerifyResponse {
    @SerializedName("results")
    private List<BatchVerifyResult> results;

    @SerializedName("summary")
    private BatchVerifySummary summary;

    public List<BatchVerifyResult> getResults() {
        return results;
    }

    public BatchVerifySummary getSummary() {
        return summary;
    }
}

class HealthStatus {
    @SerializedName("status")
    private String status;

    @SerializedName("service")
    private String service;

    @SerializedName("timestamp")
    private String timestamp;

    @SerializedName("version")
    private String version;

    public String getStatus() {
        return status;
    }

    public String getService() {
        return service;
    }

    public String getTimestamp() {
        return timestamp;
    }

    public String getVersion() {
        return version;
    }
}
