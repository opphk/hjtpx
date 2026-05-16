package com.hjtpx.sdk;

import org.junit.jupiter.api.Test;

import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;

class ModelTest {
    @Test
    void testImageCaptchaRequest() {
        ImageCaptchaRequest request = new ImageCaptchaRequest();
        request.setType(CaptchaType.NUMBER);
        request.setCount(4);
        request.setNoiseMode(2);
        request.setLineMode(1);

        assertEquals(CaptchaType.NUMBER, request.getType());
        assertEquals(4, request.getCount());
        assertEquals(2, request.getNoiseMode());
        assertEquals(1, request.getLineMode());
    }

    @Test
    void testImageCaptchaRequestConstructor() {
        ImageCaptchaRequest request = new ImageCaptchaRequest(CaptchaType.MIXED, 5);
        assertEquals(CaptchaType.MIXED, request.getType());
        assertEquals(5, request.getCount());
    }

    @Test
    void testSliderCaptchaRequest() {
        SliderCaptchaRequest request = new SliderCaptchaRequest(360, 220);
        assertEquals(360, request.getWidth());
        assertEquals(220, request.getHeight());
    }

    @Test
    void testClickCaptchaRequest() {
        ClickCaptchaRequest request = new ClickCaptchaRequest(360, 220, 4);
        assertEquals(360, request.getWidth());
        assertEquals(220, request.getHeight());
        assertEquals(4, request.getIconCount());
    }

    @Test
    void testClickData() {
        ClickData click = new ClickData(100, 200, 500);
        assertEquals(100, click.getX());
        assertEquals(200, click.getY());
        assertEquals(500, click.getDuration());
    }

    @Test
    void testClickDataDefaultConstructor() {
        ClickData click = new ClickData();
        click.setX(150);
        click.setY(250);
        click.setDuration(300);
        assertEquals(150, click.getX());
        assertEquals(250, click.getY());
        assertEquals(300, click.getDuration());
    }

    @Test
    void testVerifyImageCaptchaRequest() {
        VerifyImageCaptchaRequest request = new VerifyImageCaptchaRequest("test-id", "1234");
        assertEquals("test-id", request.getChallengeId());
        assertEquals("1234", request.getAnswer());
    }

    @Test
    void testVerifyCaptchaRequest() {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest("test-id", "slide", null);
        assertEquals("test-id", request.getChallengeId());
        assertEquals("slide", request.getAction());
    }

    @Test
    void testVerifyCaptchaResponse() {
        VerifyCaptchaResponse response = new VerifyCaptchaResponse();
        response.setSuccess(true);
        response.setScore(15.5);
        response.setRiskLevel("low");

        assertTrue(response.isSuccess());
        assertEquals(15.5, response.getScore());
        assertEquals("low", response.getRiskLevel());
    }

    @Test
    void testVerifyImageCaptchaResponse() {
        VerifyImageCaptchaResponse response = new VerifyImageCaptchaResponse(true);
        assertTrue(response.isSuccess());

        response.setSuccess(false);
        assertFalse(response.isSuccess());
    }

    @Test
    void testSliderCaptchaResponse() {
        SliderCaptchaResponse response = new SliderCaptchaResponse();
        response.setChallengeId("slider-123");
        response.setBackgroundImage("data:image/png;base64,abc");
        response.setSliderImage("data:image/png;base64,xyz");
        response.setSliderWidth(50);
        response.setSliderHeight(50);

        assertEquals("slider-123", response.getChallengeId());
        assertEquals(50, response.getSliderWidth());
        assertEquals(50, response.getSliderHeight());
    }

    @Test
    void testClickCaptchaResponse() {
        ClickCaptchaResponse response = new ClickCaptchaResponse();
        response.setChallengeId("click-123");
        response.setTargetIndex(2);
        response.setTargetPosition(Arrays.asList(100, 120));
        response.setIconPositions(Arrays.asList(
                Arrays.asList(50, 60),
                Arrays.asList(100, 120),
                Arrays.asList(150, 180)
        ));

        assertEquals("click-123", response.getChallengeId());
        assertEquals(2, response.getTargetIndex());
        assertEquals(2, response.getTargetPosition().size());
        assertEquals(3, response.getIconPositions().size());
    }

    @Test
    void testSDKResponse() {
        SDKResponse<String> response = new SDKResponse<>(0, "success", "data");
        assertEquals(0, response.getCode());
        assertEquals("success", response.getMessage());
        assertEquals("data", response.getData());
        assertTrue(response.isSuccess());
    }

    @Test
    void testSDKResponseError() {
        SDKResponse<String> response = new SDKResponse<>(1001, "error", null);
        assertFalse(response.isSuccess());
    }

    @Test
    void testPoolStats() {
        PoolStats stats = new PoolStats();
        stats.setTotalRequests(100);
        stats.setSuccessfulRequests(95);
        stats.setFailedRequests(5);
        stats.setSuccessRate(95.0);

        assertEquals(100, stats.getTotalRequests());
        assertEquals(95, stats.getSuccessfulRequests());
        assertEquals(5, stats.getFailedRequests());
        assertEquals(95.0, stats.getSuccessRate());
    }

    @Test
    void testCaptchaType() {
        assertEquals("number", CaptchaType.NUMBER.getValue());
        assertEquals("letter", CaptchaType.LETTER.getValue());
        assertEquals("mixed", CaptchaType.MIXED.getValue());

        assertEquals(CaptchaType.NUMBER, CaptchaType.valueOf("NUMBER"));
    }

    @Test
    void testConfig() {
        Config config = new Config();
        config.setBaseUrl("http://test.com");
        config.setAppId("app-id");
        config.setAppSecret("app-secret");
        config.setTimeout(60000);
        config.setMaxRetries(5);
        config.setDebugMode(true);

        assertEquals("http://test.com", config.getBaseUrl());
        assertEquals("app-id", config.getAppId());
        assertEquals("app-secret", config.getAppSecret());
        assertEquals(60000, config.getTimeout());
        assertEquals(5, config.getMaxRetries());
        assertTrue(config.isDebugMode());
    }

    @Test
    void testConfigBuilder() {
        Config config = Config.builder()
                .baseUrl("http://test.com")
                .appId("app-id")
                .appSecret("app-secret")
                .timeout(30000)
                .maxRetries(3)
                .debugMode(true)
                .build();

        assertEquals("http://test.com", config.getBaseUrl());
        assertEquals("app-id", config.getAppId());
        assertEquals("app-secret", config.getAppSecret());
        assertEquals(30000, config.getTimeout());
        assertEquals(3, config.getMaxRetries());
        assertTrue(config.isDebugMode());
    }
}
