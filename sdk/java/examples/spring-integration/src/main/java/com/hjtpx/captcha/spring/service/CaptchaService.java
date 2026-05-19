package com.hjtpx.captcha.spring.service;

import com.hjtpx.captcha.client.CaptchaClient;
import com.hjtpx.captcha.exception.CaptchaException;
import com.hjtpx.captcha.model.*;
import org.springframework.stereotype.Service;

import java.util.List;

@Service
public class CaptchaService {

    private final CaptchaClient captchaClient;

    public CaptchaService(CaptchaClient captchaClient) {
        this.captchaClient = captchaClient;
    }

    public SliderCaptchaResponse getSliderCaptcha(Integer width, Integer height, Integer tolerance) {
        return captchaClient.getSliderCaptcha(width, height, tolerance);
    }

    public VerifyCaptchaResponse verifySliderCaptcha(
            String sessionId,
            int x,
            Integer y,
            List<TrajectoryPoint> trajectory) {
        return captchaClient.verifySliderCaptcha(sessionId, x, y, trajectory);
    }

    public ClickCaptchaResponse getClickCaptcha(String mode, Boolean shuffle, Integer points) {
        return captchaClient.getClickCaptcha(mode, shuffle, points);
    }

    public VerifyCaptchaResponse verifyClickCaptcha(
            String sessionId,
            List<List<Integer>> points,
            List<Integer> clickSequence) {
        return captchaClient.verifyClickCaptcha(sessionId, points, clickSequence);
    }

    public RotationCaptchaResponse getRotationCaptcha() {
        return captchaClient.getRotationCaptcha();
    }

    public VerifyCaptchaResponse verifyRotationCaptcha(String challengeId, int angle) {
        return captchaClient.verifyRotationCaptcha(challengeId, angle);
    }

    public GestureCaptchaResponse getGestureCaptcha() {
        return captchaClient.getGestureCaptcha();
    }

    public VerifyCaptchaResponse verifyGestureCaptcha(String sessionId, List<Integer> pattern) {
        return captchaClient.verifyGestureCaptcha(sessionId, pattern);
    }

    public JigsawCaptchaResponse getJigsawCaptcha(Integer width, Integer height, Integer gridSize) {
        return captchaClient.getJigsawCaptcha(width, height, gridSize);
    }

    public VerifyCaptchaResponse verifyJigsawCaptcha(String sessionId, List<JigsawPiece> pieces) {
        return captchaClient.verifyJigsawCaptcha(sessionId, pieces);
    }

    public VoiceCaptchaResponse getVoiceCaptcha(String language) {
        return captchaClient.getVoiceCaptcha(language);
    }

    public VerifyCaptchaResponse verifyVoiceCaptcha(String sessionId, String answer) {
        return captchaClient.verifyVoiceCaptcha(sessionId, answer);
    }

    public LoginResponse login(String username, String password, String captchaToken) {
        return captchaClient.login(username, password, captchaToken);
    }

    public void logout() {
        captchaClient.logout();
    }
}
