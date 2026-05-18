package com.hjtpx.captcha.examples;

import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.model.*;

import java.util.Arrays;
import java.util.List;

public class UsageExample {

    public static void main(String[] args) {
        String baseUrl = "http://localhost:8080";
        String apiKey = "your-api-key";
        String secretKey = "your-secret-key";

        try (CaptchaClient client = new CaptchaClient(baseUrl, apiKey)) {
            sliderCaptchaExample(client);
            clickCaptchaExample(client);
            rotationCaptchaExample(client);
            gestureCaptchaExample(client);
            jigsawCaptchaExample(client);
            voiceCaptchaExample(client);
            connectCaptchaExample(client);
            threeDCaptchaExample(client);
            loginExample(client);
            detectionExample(client);
        } catch (Exception e) {
            e.printStackTrace();
        }
    }

    private static void sliderCaptchaExample(CaptchaClient client) {
        System.out.println("=== Slider Captcha Example ===");
        try {
            SliderCaptchaResponse captcha = client.getSliderCaptcha(320, 160, 5);
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Image URL: " + captcha.getImageUrl());
            System.out.println("Puzzle URL: " + captcha.getPuzzleUrl());
            System.out.println("Secret Y: " + captcha.getSecretY());

            TrajectoryPoint point1 = new TrajectoryPoint(0, captcha.getSecretY(), System.currentTimeMillis() - 1000);
            TrajectoryPoint point2 = new TrajectoryPoint(50, captcha.getSecretY() + 5, System.currentTimeMillis() - 800);
            TrajectoryPoint point3 = new TrajectoryPoint(100, captcha.getSecretY() - 3, System.currentTimeMillis() - 500);
            TrajectoryPoint point4 = new TrajectoryPoint(150, captcha.getSecretY() + 2, System.currentTimeMillis() - 200);
            TrajectoryPoint point5 = new TrajectoryPoint(180, captcha.getSecretY(), System.currentTimeMillis());
            List<TrajectoryPoint> trajectory = Arrays.asList(point1, point2, point3, point4, point5);

            VerifyCaptchaResponse verifyResponse = client.verifySliderCaptcha(
                captcha.getSessionId(),
                180,
                captcha.getSecretY(),
                trajectory
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
            if (verifyResponse.getTrajectoryResult() != null) {
                System.out.println("Trajectory score: " + verifyResponse.getTrajectoryResult().getScore());
            }
        } catch (Exception e) {
            System.err.println("Slider captcha error: " + e.getMessage());
        }
    }

    private static void clickCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== Click Captcha Example ===");
        try {
            ClickCaptchaResponse captcha = client.getClickCaptcha("number", true, 3);
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Image URL: " + captcha.getImageUrl());
            System.out.println("Hint: " + captcha.getHint());
            System.out.println("Hint order: " + captcha.getHintOrder());

            List<List<Integer>> points = Arrays.asList(
                Arrays.asList(50, 50),
                Arrays.asList(150, 100),
                Arrays.asList(250, 80)
            );

            VerifyCaptchaResponse verifyResponse = client.verifyClickCaptcha(
                captcha.getSessionId(),
                points,
                captcha.getHintOrder()
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("Click captcha error: " + e.getMessage());
        }
    }

    private static void rotationCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== Rotation Captcha Example ===");
        try {
            RotationCaptchaResponse captcha = client.getRotationCaptcha();
            System.out.println("Challenge ID: " + captcha.getChallengeId());
            System.out.println("Image URL: " + captcha.getImageUrl());

            int correctAngle = 180;
            VerifyCaptchaResponse verifyResponse = client.verifyRotationCaptcha(
                captcha.getChallengeId(),
                correctAngle
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("Rotation captcha error: " + e.getMessage());
        }
    }

    private static void gestureCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== Gesture Captcha Example ===");
        try {
            GestureCaptchaResponse captcha = client.getGestureCaptcha();
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Pattern: " + captcha.getPattern());
            System.out.println("Grid size: " + captcha.getGridSize());

            List<Integer> pattern = Arrays.asList(1, 2, 3, 5, 7, 8, 9);
            VerifyCaptchaResponse verifyResponse = client.verifyGestureCaptcha(
                captcha.getSessionId(),
                pattern
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("Gesture captcha error: " + e.getMessage());
        }
    }

    private static void jigsawCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== Jigsaw Captcha Example ===");
        try {
            JigsawCaptchaResponse captcha = client.getJigsawCaptcha(300, 300, 3);
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Image URL: " + captcha.getImageUrl());
            System.out.println("Grid size: " + captcha.getGridSize());
            System.out.println("Number of pieces: " + captcha.getPieces().size());

            List<JigsawPiece> pieces = captcha.getPieces();
            for (JigsawPiece piece : pieces) {
                piece.setCurrentX(piece.getOriginalX());
                piece.setCurrentY(piece.getOriginalY());
            }

            VerifyCaptchaResponse verifyResponse = client.verifyJigsawCaptcha(
                captcha.getSessionId(),
                pieces
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("Jigsaw captcha error: " + e.getMessage());
        }
    }

    private static void voiceCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== Voice Captcha Example ===");
        try {
            VoiceCaptchaResponse captcha = client.getVoiceCaptcha("zh-CN");
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Audio URL: " + captcha.getAudioUrl());
            System.out.println("Text: " + captcha.getText());

            VerifyCaptchaResponse verifyResponse = client.verifyVoiceCaptcha(
                captcha.getSessionId(),
                captcha.getText()
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("Voice captcha error: " + e.getMessage());
        }
    }

    private static void connectCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== Connect Captcha Example ===");
        try {
            ConnectCaptchaResponse captcha = client.getConnectCaptcha();
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Image URL: " + captcha.getImageUrl());

            List<List<Integer>> connections = Arrays.asList(
                Arrays.asList(0, 3),
                Arrays.asList(1, 4),
                Arrays.asList(2, 5)
            );

            VerifyCaptchaResponse verifyResponse = client.verifyConnectCaptcha(
                captcha.getSessionId(),
                connections
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("Connect captcha error: " + e.getMessage());
        }
    }

    private static void threeDCaptchaExample(CaptchaClient client) {
        System.out.println("\n=== 3D Captcha Example ===");
        try {
            ThreeDCaptchaResponse captcha = client.getThreeDCaptcha();
            System.out.println("Session ID: " + captcha.getSessionId());
            System.out.println("Scene URL: " + captcha.getSceneUrl());
            System.out.println("Target ID: " + captcha.getTargetId());
            System.out.println("Hint: " + captcha.getHint());

            List<Double> targetPosition = Arrays.asList(100.5, 150.0, 50.0);
            VerifyCaptchaResponse verifyResponse = client.verifyThreeDCaptcha(
                captcha.getSessionId(),
                targetPosition
            );

            System.out.println("Verification success: " + verifyResponse.isSuccess());
            System.out.println("Message: " + verifyResponse.getMessage());
        } catch (Exception e) {
            System.err.println("3D captcha error: " + e.getMessage());
        }
    }

    private static void loginExample(CaptchaClient client) {
        System.out.println("\n=== Login Example ===");
        try {
            LoginResponse loginResponse = client.login("username", "password");
            System.out.println("Login successful");
            System.out.println("Access token: " + loginResponse.getAccessToken());
            System.out.println("Refresh token: " + loginResponse.getRefreshToken());
            System.out.println("Expires in: " + loginResponse.getExpiresIn() + " seconds");
            System.out.println("User: " + loginResponse.getUser().getUsername());

            client.logout();
            System.out.println("Logged out successfully");
        } catch (Exception e) {
            System.err.println("Login error: " + e.getMessage());
        }
    }

    private static void detectionExample(CaptchaClient client) {
        System.out.println("\n=== Detection Example ===");
        try {
            String script = client.getDetectionScript();
            System.out.println("Detection script length: " + script.length() + " characters");

            System.out.println("\n=== Done ===");
        } catch (Exception e) {
            System.err.println("Detection error: " + e.getMessage());
        }
    }
}
