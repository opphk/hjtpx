package service

import (
	"testing"
	"time"
)

// TestGenerateSpatioTemporalCaptcha 测试生成时空验证码
func TestGenerateSpatioTemporalCaptcha(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	req := &SpatioTemporalCaptchaRequest{
		UserID:      "test_user_123",
		PatternType: TimePatternDaily,
		Difficulty:  "medium",
		ClientIP:    "192.168.1.1",
		UserAgent:   "Mozilla/5.0 (Windows NT 10.0; Win64; x64)",
	}

	resp, err := service.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if resp == nil {
		t.Fatal("Response is nil")
	}

	if resp.SessionID == "" {
		t.Error("SessionID should not be empty")
	}

	if resp.TargetPattern == nil {
		t.Error("TargetPattern should not be nil")
	}

	if len(resp.ChallengePoints) != 4 {
		t.Errorf("Expected 4 challenge points, got %d", len(resp.ChallengePoints))
	}

	if len(resp.Options) != 4 {
		t.Errorf("Expected 4 options, got %d", len(resp.Options))
	}

	if resp.ExpiresIn <= 0 {
		t.Error("ExpiresIn should be positive")
	}

	t.Logf("Successfully generated spatio-temporal captcha: session=%s", resp.SessionID)
}

// TestVerifySpatioTemporalCaptcha 测试验证时空验证码
func TestVerifySpatioTemporalCaptcha(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	// 首先生成验证码
	req := &SpatioTemporalCaptchaRequest{
		UserID:      "test_user_456",
		PatternType: TimePatternWeekly,
		Difficulty:  "easy",
		ClientIP:    "10.0.0.1",
		UserAgent:   "Test Agent",
	}

	resp, err := service.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 找到正确的选项
	correctOption := ""
	for _, opt := range resp.Options {
		if opt.IsCorrect {
			correctOption = opt.OptionID
			break
		}
	}

	if correctOption == "" {
		t.Fatal("No correct option found")
	}

	// 使用正确的选项进行验证
	verifyReq := &SpatioTemporalVerifyRequest{
		SessionID:     resp.SessionID,
		SelectedOption: correctOption,
		UserLocation: &SpatioTemporalPoint{
			Timestamp:  time.Now().Unix(),
			Latitude:   resp.TargetPattern.Centroid[0],
			Longitude:  resp.TargetPattern.Centroid[1],
			Accuracy:   LocationAccuracyCity,
			Confidence: 0.9,
		},
		ResponseTime: 5000,
	}

	verifyResp, err := service.Verify(verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if verifyResp == nil {
		t.Fatal("Verify response is nil")
	}

	t.Logf("Verification result: success=%v, score=%.2f, message=%s",
		verifyResp.Success, verifyResp.Score, verifyResp.Message)
}

// TestVerifyWithWrongOption 测试使用错误选项验证
func TestVerifyWithWrongOption(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	req := &SpatioTemporalCaptchaRequest{
		UserID:      "test_user_789",
		PatternType: TimePatternMonthly,
		Difficulty:  "hard",
		ClientIP:    "172.16.0.1",
	}

	resp, err := service.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 找到错误的选项
	wrongOption := ""
	for _, opt := range resp.Options {
		if !opt.IsCorrect {
			wrongOption = opt.OptionID
			break
		}
	}

	if wrongOption == "" {
		t.Fatal("No wrong option found")
	}

	verifyReq := &SpatioTemporalVerifyRequest{
		SessionID:     resp.SessionID,
		SelectedOption: wrongOption,
		UserLocation: &SpatioTemporalPoint{
			Timestamp: time.Now().Unix(),
			Latitude:  0.0,
			Longitude: 0.0,
			Accuracy:  LocationAccuracyCountry,
		},
		ResponseTime: 1000,
	}

	verifyResp, err := service.Verify(verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if verifyResp.Success {
		t.Error("Verification should fail with wrong option")
	}
}

// TestGetSpatioTemporalSession 测试获取时空验证码会话
func TestGetSpatioTemporalSession(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	req := &SpatioTemporalCaptchaRequest{
		UserID:      "test_session_user",
		PatternType: TimePatternCustom,
		Difficulty:  "medium",
	}

	resp, err := service.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 测试获取存在的会话
	session, exists := service.GetSession(resp.SessionID)
	if !exists {
		t.Fatal("Session should exist")
	}

	if session == nil {
		t.Fatal("Session should not be nil")
	}

	if session.UserID != req.UserID {
		t.Errorf("Expected user ID %s, got %s", req.UserID, session.UserID)
	}

	// 测试获取不存在的会话
	_, exists = service.GetSession("non_existent_session")
	if exists {
		t.Error("Session should not exist")
	}
}

// TestMaxAttempts 测试最大尝试次数
func TestMaxAttempts(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	req := &SpatioTemporalCaptchaRequest{
		UserID: "test_attempts_user",
	}

	resp, err := service.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 获取正确选项
	correctOption := ""
	for _, opt := range resp.Options {
		if opt.IsCorrect {
			correctOption = opt.OptionID
			break
		}
	}

	// 超过最大尝试次数
	session, _ := service.GetSession(resp.SessionID)
	session.VerifyCount = session.MaxAttempts

	verifyReq := &SpatioTemporalVerifyRequest{
		SessionID:      resp.SessionID,
		SelectedOption: correctOption,
		UserLocation: &SpatioTemporalPoint{
			Latitude:  resp.TargetPattern.Centroid[0],
			Longitude: resp.TargetPattern.Centroid[1],
		},
		ResponseTime: 2000,
	}

	verifyResp, _ := service.Verify(verifyReq)
	if verifyResp.Success {
		t.Error("Verification should fail when max attempts reached")
	}
}

// TestCalculateCentroid 测试计算质心
func TestCalculateCentroid(t *testing.T) {
	points := []SpatioTemporalPoint{
		{Latitude: 10.0, Longitude: 20.0},
		{Latitude: 30.0, Longitude: 40.0},
		{Latitude: 50.0, Longitude: 60.0},
	}

	centroid := calculateCentroid(points)
	expectedLat := 30.0
	expectedLng := 40.0

	if centroid[0] != expectedLat || centroid[1] != expectedLng {
		t.Errorf("Expected centroid (%.1f, %.1f), got (%.1f, %.1f)",
			expectedLat, expectedLng, centroid[0], centroid[1])
	}
}

// TestHaversineDistance 测试距离计算
func TestHaversineDistance(t *testing.T) {
	// 北京到上海的距离大约是 1068 公里
	beijingLat := 39.9042
	beijingLng := 116.4074
	shanghaiLat := 31.2304
	shanghaiLng := 121.4737

	distance := haversineDistance(beijingLat, beijingLng, shanghaiLat, shanghaiLng)

	// 允许 10% 的误差
	if distance < 960 || distance > 1175 {
		t.Errorf("Expected distance ~1068 km, got %.2f km", distance)
	}

	t.Logf("Distance between Beijing and Shanghai: %.2f km", distance)
}

// TestDifficultyLevels 测试不同难度级别
func TestDifficultyLevels(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	difficulties := []string{"easy", "medium", "hard", "expert"}

	for _, diff := range difficulties {
		req := &SpatioTemporalCaptchaRequest{
			UserID:     "test_difficulty_user",
			Difficulty: diff,
		}

		resp, err := service.Generate(req)
		if err != nil {
			t.Fatalf("Generate failed for difficulty %s: %v", diff, err)
		}

		if resp.TargetPattern == nil {
			t.Errorf("TargetPattern is nil for difficulty %s", diff)
		}

		t.Logf("Generated captcha for difficulty: %s, point count: %d",
			diff, len(resp.TargetPattern.Points))
	}
}

// TestSessionExpiry 测试会话过期
func TestSessionExpiry(t *testing.T) {
	service := NewSpatioTemporalCaptchaService()

	req := &SpatioTemporalCaptchaRequest{
		UserID: "test_expiry_user",
	}

	resp, err := service.Generate(req)
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	// 手动设置过期时间
	session, _ := service.GetSession(resp.SessionID)
	session.ExpiredAt = time.Now().Add(-1 * time.Hour)

	correctOption := ""
	for _, opt := range resp.Options {
		if opt.IsCorrect {
			correctOption = opt.OptionID
			break
		}
	}

	verifyReq := &SpatioTemporalVerifyRequest{
		SessionID:      resp.SessionID,
		SelectedOption: correctOption,
	}

	verifyResp, _ := service.Verify(verifyReq)
	if verifyResp.Success {
		t.Error("Verification should fail for expired session")
	}
}
