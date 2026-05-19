package com.hjtpx.captcha.spring.controller;

import com.hjtpx.captcha.model.*;
import com.hjtpx.captcha.spring.service.CaptchaService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.List;
import java.util.Map;

@RestController
@RequestMapping("/api/captcha")
public class CaptchaController {

    private final CaptchaService captchaService;

    public CaptchaController(CaptchaService captchaService) {
        this.captchaService = captchaService;
    }

    @GetMapping("/slider")
    public ResponseEntity<Map<String, Object>> getSliderCaptcha(
            @RequestParam(required = false, defaultValue = "320") Integer width,
            @RequestParam(required = false, defaultValue = "160") Integer height,
            @RequestParam(required = false, defaultValue = "8") Integer tolerance) {

        try {
            SliderCaptchaResponse captcha = captchaService.getSliderCaptcha(width, height, tolerance);

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("session_id", captcha.getSessionId());
            response.put("image_url", captcha.getImageUrl());
            response.put("puzzle_url", captcha.getPuzzleUrl());
            response.put("secret_y", captcha.getSecretY());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @PostMapping("/slider/verify")
    public ResponseEntity<Map<String, Object>> verifySliderCaptcha(
            @RequestBody Map<String, Object> request) {

        try {
            String sessionId = (String) request.get("session_id");
            Number xNum = (Number) request.get("x");
            Number yNum = (Number) request.get("y");
            @SuppressWarnings("unchecked")
            List<TrajectoryPoint> trajectory = (List<TrajectoryPoint>) request.get("trajectory");

            if (sessionId == null || xNum == null) {
                return badRequest("Missing required parameters");
            }

            int x = xNum.intValue();
            Integer y = yNum != null ? yNum.intValue() : null;

            VerifyCaptchaResponse result = captchaService.verifySliderCaptcha(
                    sessionId, x, y, trajectory);

            Map<String, Object> response = new HashMap<>();
            response.put("success", result.isSuccess());
            response.put("message", result.getMessage());
            response.put("risk_score", result.getScore());
            response.put("captcha_pass", result.isSuccess());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @GetMapping("/click")
    public ResponseEntity<Map<String, Object>> getClickCaptcha(
            @RequestParam(required = false, defaultValue = "number") String mode,
            @RequestParam(required = false, defaultValue = "3") Integer points,
            @RequestParam(required = false, defaultValue = "true") Boolean shuffle) {

        try {
            ClickCaptchaResponse captcha = captchaService.getClickCaptcha(mode, shuffle, points);

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("session_id", captcha.getSessionId());
            response.put("image_url", captcha.getImageUrl());
            response.put("hint", captcha.getHint());
            response.put("hint_order", captcha.getHintOrder());
            response.put("max_points", captcha.getMaxPoints());
            response.put("mode", captcha.getMode());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @PostMapping("/click/verify")
    public ResponseEntity<Map<String, Object>> verifyClickCaptcha(
            @RequestBody Map<String, Object> request) {

        try {
            String sessionId = (String) request.get("session_id");
            @SuppressWarnings("unchecked")
            List<List<Integer>> points = (List<List<Integer>>) request.get("points");
            @SuppressWarnings("unchecked")
            List<Integer> clickSequence = (List<Integer>) request.get("click_sequence");

            if (sessionId == null || points == null) {
                return badRequest("Missing required parameters");
            }

            VerifyCaptchaResponse result = captchaService.verifyClickCaptcha(
                    sessionId, points, clickSequence);

            Map<String, Object> response = new HashMap<>();
            response.put("success", result.isSuccess());
            response.put("message", result.getMessage());
            response.put("risk_score", result.getScore());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @GetMapping("/rotation")
    public ResponseEntity<Map<String, Object>> getRotationCaptcha() {

        try {
            RotationCaptchaResponse captcha = captchaService.getRotationCaptcha();

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("challenge_id", captcha.getChallengeId());
            response.put("image", captcha.getImage());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @PostMapping("/rotation/verify")
    public ResponseEntity<Map<String, Object>> verifyRotationCaptcha(
            @RequestBody Map<String, Object> request) {

        try {
            String challengeId = (String) request.get("challenge_id");
            Number angleNum = (Number) request.get("angle");

            if (challengeId == null || angleNum == null) {
                return badRequest("Missing required parameters");
            }

            int angle = angleNum.intValue();
            VerifyCaptchaResponse result = captchaService.verifyRotationCaptcha(
                    challengeId, angle);

            Map<String, Object> response = new HashMap<>();
            response.put("success", result.isSuccess());
            response.put("message", result.getMessage());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @GetMapping("/gesture")
    public ResponseEntity<Map<String, Object>> getGestureCaptcha() {

        try {
            GestureCaptchaResponse captcha = captchaService.getGestureCaptcha();

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("session_id", captcha.getSessionId());
            response.put("pattern", captcha.getPattern());
            response.put("grid_size", captcha.getGridSize());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @PostMapping("/gesture/verify")
    public ResponseEntity<Map<String, Object>> verifyGestureCaptcha(
            @RequestBody Map<String, Object> request) {

        try {
            String sessionId = (String) request.get("session_id");
            @SuppressWarnings("unchecked")
            List<Integer> pattern = (List<Integer>) request.get("pattern");

            if (sessionId == null || pattern == null) {
                return badRequest("Missing required parameters");
            }

            VerifyCaptchaResponse result = captchaService.verifyGestureCaptcha(
                    sessionId, pattern);

            Map<String, Object> response = new HashMap<>();
            response.put("success", result.isSuccess());
            response.put("message", result.getMessage());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @GetMapping("/jigsaw")
    public ResponseEntity<Map<String, Object>> getJigsawCaptcha(
            @RequestParam(required = false, defaultValue = "300") Integer width,
            @RequestParam(required = false, defaultValue = "300") Integer height,
            @RequestParam(required = false, defaultValue = "3") Integer gridSize) {

        try {
            JigsawCaptchaResponse captcha = captchaService.getJigsawCaptcha(
                    width, height, gridSize);

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("session_id", captcha.getSessionId());
            response.put("image_url", captcha.getImageUrl());
            response.put("grid_size", captcha.getGridSize());
            response.put("piece_count", captcha.getPieces() != null ? captcha.getPieces().size() : 0);

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @PostMapping("/jigsaw/verify")
    public ResponseEntity<Map<String, Object>> verifyJigsawCaptcha(
            @RequestBody Map<String, Object> request) {

        try {
            String sessionId = (String) request.get("session_id");
            @SuppressWarnings("unchecked")
            List<JigsawPiece> pieces = (List<JigsawPiece>) request.get("pieces");

            if (sessionId == null || pieces == null) {
                return badRequest("Missing required parameters");
            }

            VerifyCaptchaResponse result = captchaService.verifyJigsawCaptcha(
                    sessionId, pieces);

            Map<String, Object> response = new HashMap<>();
            response.put("success", result.isSuccess());
            response.put("message", result.getMessage());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    @PostMapping("/verify")
    public ResponseEntity<Map<String, Object>> genericVerify(
            @RequestBody VerifyCaptchaRequest request) {

        try {
            VerifyCaptchaResponse result = captchaService.verifySliderCaptcha(
                    request.getSessionId(),
                    request.getX(),
                    request.getY(),
                    request.getTrajectory());

            Map<String, Object> response = new HashMap<>();
            response.put("success", result.isSuccess());
            response.put("message", result.getMessage());
            response.put("risk_score", result.getScore());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            return handleError(e);
        }
    }

    private ResponseEntity<Map<String, Object>> handleError(Exception e) {
        Map<String, Object> response = new HashMap<>();
        response.put("success", false);
        response.put("error", e.getMessage());

        return ResponseEntity.status(500).body(response);
    }

    private ResponseEntity<Map<String, Object>> badRequest(String message) {
        Map<String, Object> response = new HashMap<>();
        response.put("success", false);
        response.put("error", message);

        return ResponseEntity.badRequest().body(response);
    }
}
