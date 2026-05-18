package com.hjtpx.captcha;

import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;
import org.junit.jupiter.api.BeforeEach;
import org.junit.jupiter.api.Test;
import org.junit.jupiter.api.extension.ExtendWith;
import org.mockito.Mock;
import org.mockito.junit.jupiter.MockitoExtension;

import java.util.Arrays;
import java.util.List;

import static org.junit.jupiter.api.Assertions.*;
import static org.mockito.ArgumentMatchers.any;
import static org.mockito.Mockito.*;

@ExtendWith(MockitoExtension.class)
public class CaptchaClientTest {

    private CaptchaClient client;

    @BeforeEach
    public void setUp() {
        client = new CaptchaClient("http://localhost:8080");
    }

    @Test
    public void testClientInitialization() {
        assertNotNull(client);
        assertEquals("http://localhost:8080", client.getConfig().getBaseUrl());
    }

    @Test
    public void testTrajectoryPoint() {
        TrajectoryPoint point = new TrajectoryPoint(100, 200, 1234567890L);
        assertEquals(100, point.getX());
        assertEquals(200, point.getY());
        assertEquals(1234567890L, point.getTimestamp());

        point.setX(150);
        point.setY(250);
        point.setTimestamp(9876543210L);
        assertEquals(150, point.getX());
        assertEquals(250, point.getY());
        assertEquals(9876543210L, point.getTimestamp());
    }

    @Test
    public void testJigsawPiece() {
        JigsawPiece piece = new JigsawPiece(0, 0, 0, 100, 100, 100, 100);
        assertEquals(0, piece.getIndex());
        assertEquals(0, piece.getOriginalX());
        assertEquals(0, piece.getOriginalY());
        assertEquals(100, piece.getCurrentX());
        assertEquals(100, piece.getCurrentY());
        assertEquals(100, piece.getWidth());
        assertEquals(100, piece.getHeight());
        assertEquals(0, piece.getRotation());

        piece.setRotation(90);
        assertEquals(90, piece.getRotation());
    }

    @Test
    public void testVerifyCaptchaRequest() {
        VerifyCaptchaRequest request = new VerifyCaptchaRequest();
        request.setSessionId("test-session-id");
        request.setType("slider");
        request.setX(150);
        request.setY(200);

        TrajectoryPoint point1 = new TrajectoryPoint(0, 200, 1234567890L);
        TrajectoryPoint point2 = new TrajectoryPoint(150, 200, 1234567891L);
        request.setTrajectory(Arrays.asList(point1, point2));

        assertEquals("test-session-id", request.getSessionId());
        assertEquals("slider", request.getType());
        assertEquals(150, request.getX());
        assertEquals(200, request.getY());
        assertNotNull(request.getTrajectory());
        assertEquals(2, request.getTrajectory().size());
    }

    @Test
    public void testApiResponse() {
        ApiResponse<String> response = new ApiResponse<>();
        response.setCode(0);
        response.setMessage("success");
        response.setData("test-data");

        assertTrue(response.isSuccess());
        assertEquals(0, response.getCode());
        assertEquals("success", response.getMessage());
        assertEquals("test-data", response.getData());

        response.setCode(1);
        assertFalse(response.isSuccess());
    }

    @Test
    public void testSliderCaptchaResponse() {
        SliderCaptchaResponse response = new SliderCaptchaResponse();
        response.setSessionId("session-123");
        response.setImageUrl("http://example.com/image.jpg");
        response.setPuzzleUrl("http://example.com/puzzle.jpg");
        response.setHintUrl("http://example.com/hint.jpg");
        response.setShape(1);
        response.setSecretY(100);
        response.setImageWidth(320);
        response.setImageHeight(160);
        response.setTolerance(5);

        assertEquals("session-123", response.getSessionId());
        assertEquals("http://example.com/image.jpg", response.getImageUrl());
        assertEquals("http://example.com/puzzle.jpg", response.getPuzzleUrl());
        assertEquals("http://example.com/hint.jpg", response.getHintUrl());
        assertEquals(1, response.getShape());
        assertEquals(100, response.getSecretY());
        assertEquals(320, response.getImageWidth());
        assertEquals(160, response.getImageHeight());
        assertEquals(5, response.getTolerance());
    }

    @Test
    public void testClickCaptchaResponse() {
        ClickCaptchaResponse response = new ClickCaptchaResponse();
        response.setSessionId("session-123");
        response.setImageUrl("http://example.com/image.jpg");
        response.setHint("1,2,3");
        response.setHintOrder(Arrays.asList(1, 2, 3));
        response.setMaxPoints(3);
        response.setMode("number");
        response.setAllowShuffle(true);
        response.setPoints(Arrays.asList(Arrays.asList(10, 20), Arrays.asList(30, 40)));

        assertEquals("session-123", response.getSessionId());
        assertEquals("http://example.com/image.jpg", response.getImageUrl());
        assertEquals("1,2,3", response.getHint());
        assertEquals(Arrays.asList(1, 2, 3), response.getHintOrder());
        assertEquals(3, response.getMaxPoints());
        assertEquals("number", response.getMode());
        assertTrue(response.isAllowShuffle());
        assertNotNull(response.getPoints());
        assertEquals(2, response.getPoints().size());
    }

    @Test
    public void testVerifyCaptchaResponse() {
        VerifyCaptchaResponse response = new VerifyCaptchaResponse();
        response.setSuccess(true);
        response.setMessage("Verification successful");
        response.setRemainingAttempts(3);
        response.setRiskScore(0.1);
        response.setCaptchaPass(true);

        VerifyCaptchaResponse.TrajectoryResult trajectoryResult = new VerifyCaptchaResponse.TrajectoryResult();
        trajectoryResult.setScore(0.9);
        trajectoryResult.setPassed(true);
        trajectoryResult.setReasons(Arrays.asList("smooth", "human-like"));
        response.setTrajectoryResult(trajectoryResult);

        assertTrue(response.isSuccess());
        assertEquals("Verification successful", response.getMessage());
        assertEquals(3, response.getRemainingAttempts());
        assertEquals(0.1, response.getRiskScore());
        assertTrue(response.getCaptchaPass());
        assertNotNull(response.getTrajectoryResult());
        assertEquals(0.9, response.getTrajectoryResult().getScore());
        assertTrue(response.getTrajectoryResult().isPassed());
        assertEquals(2, response.getTrajectoryResult().getReasons().size());
    }

    @Test
    public void testLoginRequestAndResponse() {
        LoginRequest loginRequest = new LoginRequest();
        loginRequest.setUsername("testuser");
        loginRequest.setPassword("testpass");
        loginRequest.setCaptchaToken("token-123");

        assertEquals("testuser", loginRequest.getUsername());
        assertEquals("testpass", loginRequest.getPassword());
        assertEquals("token-123", loginRequest.getCaptchaToken());

        LoginResponse loginResponse = new LoginResponse();
        loginResponse.setAccessToken("access-token");
        loginResponse.setRefreshToken("refresh-token");
        loginResponse.setExpiresIn(3600);

        LoginResponse.UserInfo userInfo = new LoginResponse.UserInfo();
        userInfo.setId(1);
        userInfo.setUsername("testuser");
        userInfo.setEmail("test@example.com");
        loginResponse.setUser(userInfo);

        assertEquals("access-token", loginResponse.getAccessToken());
        assertEquals("refresh-token", loginResponse.getRefreshToken());
        assertEquals(3600, loginResponse.getExpiresIn());
        assertNotNull(loginResponse.getUser());
        assertEquals(1, loginResponse.getUser().getId());
        assertEquals("testuser", loginResponse.getUser().getUsername());
        assertEquals("test@example.com", loginResponse.getUser().getEmail());
    }

    @Test
    public void testClientClose() throws Exception {
        CaptchaClient closeableClient = new CaptchaClient("http://localhost:8080");
        closeableClient.close();
    }
}
