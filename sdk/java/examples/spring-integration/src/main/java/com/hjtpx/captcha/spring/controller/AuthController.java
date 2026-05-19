package com.hjtpx.captcha.spring.controller;

import com.hjtpx.captcha.model.LoginResponse;
import com.hjtpx.captcha.spring.service.CaptchaService;
import org.springframework.http.ResponseEntity;
import org.springframework.web.bind.annotation.*;

import java.util.HashMap;
import java.util.Map;

@RestController
@RequestMapping("/api/auth")
public class AuthController {

    private final CaptchaService captchaService;

    public AuthController(CaptchaService captchaService) {
        this.captchaService = captchaService;
    }

    @PostMapping("/login")
    public ResponseEntity<Map<String, Object>> login(@RequestBody Map<String, String> request) {
        try {
            String username = request.get("username");
            String password = request.get("password");
            String captchaToken = request.get("captcha_token");

            if (username == null || password == null) {
                return badRequest("Missing credentials");
            }

            LoginResponse result = captchaService.login(username, password, captchaToken);

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("access_token", result.getAccessToken());
            response.put("refresh_token", result.getRefreshToken());
            response.put("expires_in", result.getExpiresIn());

            return ResponseEntity.ok(response);

        } catch (Exception e) {
            Map<String, Object> response = new HashMap<>();
            response.put("success", false);
            response.put("error", e.getMessage());
            return ResponseEntity.status(401).body(response);
        }
    }

    @PostMapping("/logout")
    public ResponseEntity<Map<String, Object>> logout() {
        try {
            captchaService.logout();

            Map<String, Object> response = new HashMap<>();
            response.put("success", true);
            response.put("message", "Logged out successfully");

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
