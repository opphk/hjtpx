package com.hjtpx.sdk;

import java.util.Arrays;
import java.util.List;
import java.util.Scanner;
import java.nio.file.Files;
import java.nio.file.Paths;

public class Example {
    public static void main(String[] args) {
        System.out.println("=".repeat(50));
        System.out.println("hjtpx Java SDK Examples");
        System.out.println("=".repeat(50));

        try (CaptchaClient client = new CaptchaClient()) {
            exampleSliderCaptcha(client);
            exampleClickCaptcha(client);
            exampleImageCaptcha(client);
            exampleStatistics(client);
            exampleErrorHandling(client);

            System.out.println("\n" + "=".repeat(50));
            System.out.println("All examples completed successfully!");
            System.out.println("=".repeat(50));
        } catch (SDKError e) {
            System.err.println("SDK Error: " + e.getMessage());
        } catch (Exception e) {
            System.err.println("Error: " + e.getMessage());
            e.printStackTrace();
        }
    }

    private static void exampleSliderCaptcha(CaptchaClient client) throws SDKError {
        System.out.println("\n=== Slider Captcha Example ===");

        SliderCaptchaRequest request = new SliderCaptchaRequest(360, 220);
        SliderCaptchaResponse slider = client.generateSliderCaptcha(request);

        System.out.println("✓ Challenge ID: " + slider.getChallengeId());
        System.out.println("  Slider Size: " + slider.getSliderWidth() + "x" + slider.getSliderHeight());

        VerifyCaptchaResponse result = client.verifySliderCaptcha(slider.getChallengeId(), "120");
        System.out.println("✓ Verification success: " + result.isSuccess());
        System.out.println("  Score: " + result.getScore());
        System.out.println("  Risk Level: " + result.getRiskLevel());
    }

    private static void exampleClickCaptcha(CaptchaClient client) throws SDKError {
        System.out.println("\n=== Click Captcha Example ===");

        ClickCaptchaRequest request = new ClickCaptchaRequest(360, 220, 4);
        ClickCaptchaResponse click = client.generateClickCaptcha(request);

        System.out.println("✓ Challenge ID: " + click.getChallengeId());
        System.out.println("  Target Index: " + click.getTargetIndex());
        System.out.println("  Icon Positions: " + click.getIconPositions());

        List<ClickData> clicks = Arrays.asList(
            new ClickData(
                click.getTargetPosition().get(0),
                click.getTargetPosition().get(1),
                500
            )
        );

        VerifyCaptchaResponse result = client.verifyClickCaptcha(click.getChallengeId(), clicks);
        System.out.println("✓ Verification success: " + result.isSuccess());
    }

    private static void exampleImageCaptcha(CaptchaClient client) throws SDKError {
        System.out.println("\n=== Image Captcha Example ===");

        ImageCaptchaRequest request = new ImageCaptchaRequest();
        request.setType(CaptchaType.MIXED);
        request.setCount(4);
        request.setNoiseMode(2);
        request.setLineMode(1);

        ImageCaptchaResponse captcha = client.generateImageCaptcha(request);
        System.out.println("✓ Challenge ID: " + captcha.getChallengeId());

        byte[] imageData = client.extractBase64Image(captcha.getImage());
        Files.write(Paths.get("captcha.png"), imageData);
        System.out.println("✓ Image saved to captcha.png");

        VerifyImageCaptchaResponse result = client.verifyImageCaptcha(captcha.getChallengeId(), "1234");
        System.out.println("✓ Verification success: " + result.isSuccess());
    }

    private static void exampleStatistics(CaptchaClient client) {
        System.out.println("\n=== Statistics Example ===");

        PoolStats stats = client.getStats();
        System.out.println("📊 Current Statistics:");
        System.out.println("  Total Requests: " + stats.getTotalRequests());
        System.out.println("  Successful: " + stats.getSuccessfulRequests());
        System.out.println("  Failed: " + stats.getFailedRequests());
        System.out.println("  Retried: " + stats.getRetriedRequests());
        System.out.println("  Success Rate: " + String.format("%.2f", stats.getSuccessRate()) + "%");
    }

    private static void exampleErrorHandling(CaptchaClient client) {
        System.out.println("\n=== Error Handling Example ===");

        try {
            client.verifyImageCaptcha("", "1234");
        } catch (SDKError e) {
            System.out.println("✓ Caught expected error: " + e.getMessage());
            System.out.println("  Error Code: " + e.getCode());
            System.out.println("  Is Invalid Params: " + e.isInvalidParams());
        }

        try {
            client.verifySliderCaptcha("test-id", "");
        } catch (SDKError e) {
            System.out.println("✓ Caught expected error: " + e.getMessage());
        }
    }
}
