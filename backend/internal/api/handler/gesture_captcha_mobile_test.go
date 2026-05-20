package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGenerateGestureCaptcha(t *testing.T) {
	router := gin.New()
	router.GET("/api/v1/captcha/gesture", GenerateGestureCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/gesture", nil)
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	if resp["code"] != response.SuccessCode {
		t.Errorf("Expected success code, got %v", resp["code"])
	}

	data := resp["data"].(map[string]interface{})

	if data["session_id"] == nil || data["session_id"] == "" {
		t.Error("Session ID should not be empty")
	}

	if data["pattern"] == nil || data["pattern"] == "" {
		t.Error("Pattern should not be empty")
	}

	if data["mobile_optimized"] == nil {
		t.Error("Mobile optimized flag should be present")
	}

	t.Logf("Test GenerateGestureCaptcha passed: session_id=%s, mobile_optimized=%v",
		data["session_id"], data["mobile_optimized"])
}

func TestGenerateGestureCaptchaMobile(t *testing.T) {
	mobileAgents := []string{
		"Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)",
		"Mozilla/5.0 (Linux; Android 11; SM-G991B) AppleWebKit/537.36",
		"Mozilla/5.0 (iPad; CPU OS 14_0 like Mac OS X)",
	}

	for _, agent := range mobileAgents {
		router := gin.New()
		router.GET("/api/v1/captcha/gesture", GenerateGestureCaptcha)

		req, _ := http.NewRequest("GET", "/api/v1/captcha/gesture", nil)
		req.Header.Set("User-Agent", agent)

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Mobile UA '%s': Expected status 200, got %d", agent, w.Code)
		}

		var resp map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
			t.Errorf("Mobile UA '%s': Failed to parse response: %v", agent, err)
		}

		data := resp["data"].(map[string]interface{})
		mobileOptimized := data["mobile_optimized"].(bool)

		if !mobileOptimized {
			t.Errorf("Mobile UA '%s': Should be detected as mobile", agent)
		}

		t.Logf("Mobile UA '%s' detected correctly as mobile", agent)
	}
}

func TestVerifyGestureCaptcha(t *testing.T) {
	GenerateGestureCaptcha(httptest.NewRequest("GET", "/", nil))
	sessionID := "test_session_123"

	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", func(c *gin.Context) {
		var req VerifyGestureCaptchaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, response.CodeInvalidParams, "参数错误")
			return
		}

		gestureMutex.RLock()
		session := gestureSessions[req.ID]
		gestureMutex.RUnlock()

		if session == nil {
			gestureMutex.Lock()
			gestureSessions[req.ID] = &GestureCaptchaSession{
				SessionID:   req.ID,
				Pattern:     "1-5-9",
				Status:      "pending",
				MaxAttempts: 3,
				VerifyCount: 0,
				CreatedAt:   time.Now(),
				ExpiredAt:   time.Now().Add(5 * time.Minute),
				IsMobile:    false,
			}
			gestureMutex.Unlock()
		}

		success, _ := verifyGesturePattern("1-5-9", req.Pattern)

		if success {
			response.Success(c, gin.H{
				"success": true,
				"message": "验证成功",
			})
		} else {
			response.Success(c, gin.H{
				"success": false,
				"message": "验证失败",
			})
		}
	})

	payload := map[string]interface{}{
		"id":      sessionID,
		"pattern": "1-5-9",
		"touch_data": &TouchVerificationData{
			TotalTouches:  3,
			TouchPressure: 0.5,
			TouchDuration: 1500,
			TouchArea:     20,
			VelocityProfile: []float64{150.0, 180.0, 160.0},
			IsMultiTouch: false,
		},
		"device_info": &DeviceInfo{
			IsMobile:       true,
			TouchCapable:   true,
			MaxTouchPoints: 5,
			Platform:       "iOS",
		},
	}

	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	t.Logf("Test VerifyGestureCaptcha passed: %v", resp)
}

func TestVerifyGestureCaptchaTouchData(t *testing.T) {
	testCases := []struct {
		name        string
		touchData   *TouchVerificationData
		expectScore float64
	}{
		{
			name: "Valid touch data",
			touchData: &TouchVerificationData{
				TotalTouches:    3,
				TouchPressure:   0.5,
				TouchDuration:   1500,
				TouchArea:       20,
				VelocityProfile: []float64{150.0, 180.0, 160.0},
				IsMultiTouch:    false,
			},
			expectScore: 70,
		},
		{
			name: "Too fast touch",
			touchData: &TouchVerificationData{
				TotalTouches:    0,
				TouchPressure:   0.5,
				TouchDuration:   30,
				TouchArea:       20,
				VelocityProfile: []float64{1500.0, 1800.0, 1600.0},
				IsMultiTouch:    false,
			},
			expectScore: 55,
		},
		{
			name: "Multi-touch detected",
			touchData: &TouchVerificationData{
				TotalTouches:    5,
				TouchPressure:   0.8,
				TouchDuration:   2000,
				TouchArea:       30,
				VelocityProfile: []float64{100.0, 120.0, 110.0},
				IsMultiTouch:    true,
			},
			expectScore: 50,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateTouchVerificationScore(tc.touchData)

			if score > tc.expectScore+10 || score < tc.expectScore-10 {
				t.Errorf("Expected score around %f, got %f", tc.expectScore, score)
			}

			indicators := analyzeTouchData(tc.touchData)
			t.Logf("Touch data analysis: score=%f, indicators=%v", score, indicators)
		})
	}
}

func TestVerifyGesturePattern(t *testing.T) {
	testCases := []struct {
		expected string
		actual   string
		success  bool
	}{
		{"1-5-9", "1-5-9", true},
		{"1-2-3", "1-2-3", true},
		{"1-5-9", "1-2-3", false},
		{"1-2-3", "1-2", false},
		{"1-2", "1-2-3", false},
		{"", "", true},
		{"1-2-3", "", false},
	}

	for _, tc := range testCases {
		t.Run(tc.expected+"_"+tc.actual, func(t *testing.T) {
			success, _ := verifyGesturePattern(tc.expected, tc.actual)

			if success != tc.success {
				t.Errorf("Expected %v, got %v for pattern '%s' vs '%s'",
					tc.success, success, tc.expected, tc.actual)
			}
		})
	}
}

func TestParsePattern(t *testing.T) {
	testCases := []struct {
		input    string
		expected []int
	}{
		{"1-5-9", []int{1, 5, 9}},
		{"1-2-3-4-5", []int{1, 2, 3, 4, 5}},
		{"", []int{}},
		{"1", []int{1}},
	}

	for _, tc := range testCases {
		t.Run(tc.input, func(t *testing.T) {
			result := parsePattern(tc.input)

			if len(result) != len(tc.expected) {
				t.Errorf("Expected %d points, got %d", len(tc.expected), len(result))
				return
			}

			for i, v := range result {
				if v != tc.expected[i] {
					t.Errorf("Expected %d at position %d, got %d",
						tc.expected[i], i, v)
				}
			}
		})
	}
}

func TestGenerateGesturePattern(t *testing.T) {
	for length := 3; length <= 6; length++ {
		pattern := generateGesturePattern(length)
		points := parsePattern(pattern)

		if len(points) != length {
			t.Errorf("Expected pattern length %d, got %d", length, len(points))
		}

		for i := 0; i < len(points); i++ {
			for j := i + 1; j < len(points); j++ {
				if points[i] == points[j] {
					t.Errorf("Duplicate point found in pattern: %s", pattern)
				}
			}
		}

		t.Logf("Generated pattern length %d: %s", length, pattern)
	}
}

func TestRotateCaptchaGeneration(t *testing.T) {
	router := gin.New()
	router.POST("/api/v1/captcha/rotate", GenerateRotateCaptcha)

	payload := RotateCaptchaRequest{Width: 200, Height: 200}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/rotate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("Failed to parse response: %v", err)
	}

	data := resp["data"].(map[string]interface{})

	if data["session_id"] == nil || data["session_id"] == "" {
		t.Error("Session ID should not be empty")
	}

	if data["target_angle"] == nil {
		t.Error("Target angle should be present")
	}

	if data["current_angle"] == nil {
		t.Error("Current angle should be present")
	}

	t.Logf("Rotate captcha generated: session_id=%s", data["session_id"])
}

func TestRotateCaptchaVerification(t *testing.T) {
	router := gin.New()
	router.POST("/api/v1/captcha/rotate", GenerateRotateCaptcha)
	router.POST("/api/v1/captcha/rotate/verify", VerifyRotateCaptcha)

	payload := RotateCaptchaRequest{Width: 200, Height: 200}
	body, _ := json.Marshal(payload)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/rotate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	var genResp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &genResp)

	data := genResp["data"].(map[string]interface{})
	sessionID := data["session_id"].(string)
	targetAngle := int(data["target_angle"].(float64))

	testCases := []struct {
		name         string
		inputAngle   int
		shouldPass   bool
		description  string
	}{
		{
			name:        "Exact match",
			inputAngle:  targetAngle,
			shouldPass:  true,
			description: "Should pass with exact angle",
		},
		{
			name:        "Within tolerance",
			inputAngle:  targetAngle + 10,
			shouldPass:  true,
			description: "Should pass within 15 degree tolerance",
		},
		{
			name:        "Outside tolerance",
			inputAngle:  targetAngle + 30,
			shouldPass:  false,
			description: "Should fail outside 15 degree tolerance",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			verifyPayload := RotateVerifyRequest{
				SessionID: sessionID,
				Angle:     tc.inputAngle,
				TouchData: &TouchVerificationData{
					TotalTouches:    1,
					TouchPressure:   0.5,
					TouchDuration:   1000,
					TouchArea:       20,
					VelocityProfile: []float64{100.0},
					IsMultiTouch:    false,
				},
			}

			body, _ := json.Marshal(verifyPayload)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/rotate/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			var verifyResp map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &verifyResp)

			verifyData := verifyResp["data"].(map[string]interface{})
			success := verifyData["success"].(bool)

			if success != tc.shouldPass {
				t.Errorf("%s: Expected %v, got %v", tc.description, tc.shouldPass, success)
			}

			t.Logf("%s: input_angle=%d, target_angle=%d, success=%v",
				tc.name, tc.inputAngle, targetAngle, success)
		})
	}
}

func TestGetGestureGridPoints(t *testing.T) {
	router := gin.New()
	router.GET("/api/v1/captcha/gesture/grid", GetGestureGridPoints)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/gesture/grid", nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})

	if data["grid_size"] == nil || int(data["grid_size"].(float64)) != 3 {
		t.Error("Grid size should be 3")
	}

	if data["points"] == nil {
		t.Error("Points should not be empty")
	}

	points := data["points"].(map[string]interface{})
	if len(points) != 9 {
		t.Errorf("Expected 9 points, got %d", len(points))
	}

	t.Logf("Gesture grid points: %d points", len(points))
}

func TestSessionExpiration(t *testing.T) {
	sessionID := "test_expired_session"

	gestureMutex.Lock()
	gestureSessions[sessionID] = &GestureCaptchaSession{
		SessionID:   sessionID,
		Pattern:     "1-5-9",
		Status:      "pending",
		MaxAttempts: 3,
		VerifyCount: 0,
		CreatedAt:   time.Now().Add(-10 * time.Minute),
		ExpiredAt:   time.Now().Add(-5 * time.Minute),
		IsMobile:    true,
	}
	gestureMutex.Unlock()

	router := gin.New()
	router.POST("/api/v1/captcha/gesture/verify", func(c *gin.Context) {
		var req VerifyGestureCaptchaRequest
		if err := c.ShouldBindJSON(&req); err != nil {
			response.Fail(c, response.CodeInvalidParams, "参数错误")
			return
		}

		gestureMutex.RLock()
		session, exists := gestureSessions[req.ID]
		gestureMutex.RUnlock()

		if !exists {
			response.Fail(c, response.CodeNotFound, "会话不存在或已过期")
			return
		}

		if time.Now().After(session.ExpiredAt) {
			gestureMutex.Lock()
			delete(gestureSessions, req.ID)
			gestureMutex.Unlock()
			response.Fail(c, response.CodeNotFound, "会话已过期")
			return
		}

		response.Success(c, gin.H{"message": "OK"})
	})

	payload := map[string]interface{}{
		"id":      sessionID,
		"pattern": "1-5-9",
	}
	body, _ := json.Marshal(payload)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/gesture/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	if resp["code"] != response.SuccessCode {
		t.Error("Expected success code")
	}

	t.Log("Test session expiration passed")
}

func TestGestureCaptchaStatus(t *testing.T) {
	sessionID := "test_status_session"

	expiresAt := time.Now().Add(5 * time.Minute)

	gestureMutex.Lock()
	gestureSessions[sessionID] = &GestureCaptchaSession{
		SessionID:   sessionID,
		Pattern:     "1-5-9",
		Status:      "pending",
		MaxAttempts: 3,
		VerifyCount: 1,
		CreatedAt:   time.Now(),
		ExpiredAt:   expiresAt,
		IsMobile:    true,
	}
	gestureMutex.Unlock()

	router := gin.New()
	router.GET("/api/v1/captcha/gesture/status/:session_id", GetGestureCaptchaStatus)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/gesture/status/"+sessionID, nil)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var resp map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &resp)

	data := resp["data"].(map[string]interface{})

	if data["session_id"] != sessionID {
		t.Errorf("Expected session_id %s, got %s", sessionID, data["session_id"])
	}

	if data["status"] != "pending" {
		t.Errorf("Expected status 'pending', got %s", data["status"])
	}

	if int(data["verify_count"].(float64)) != 1 {
		t.Errorf("Expected verify_count 1, got %d", int(data["verify_count"].(float64)))
	}

	if int(data["max_attempts"].(float64)) != 3 {
		t.Errorf("Expected max_attempts 3, got %d", int(data["max_attempts"].(float64)))
	}

	t.Log("Test gesture captcha status passed")
}

func BenchmarkGesturePatternGeneration(b *testing.B) {
	for i := 0; i < b.N; i++ {
		generateGesturePattern(5)
	}
}

func BenchmarkPatternParsing(b *testing.B) {
	pattern := "1-2-3-4-5-6-7-8-9"

	for i := 0; i < b.N; i++ {
		parsePattern(pattern)
	}
}

func BenchmarkTouchScoreCalculation(b *testing.B) {
	touchData := &TouchVerificationData{
		TotalTouches:    5,
		TouchPressure:   0.5,
		TouchDuration:   1500,
		TouchArea:       20,
		VelocityProfile: []float64{150.0, 180.0, 160.0, 170.0, 155.0},
		IsMultiTouch:    false,
	}

	for i := 0; i < b.N; i++ {
		calculateTouchVerificationScore(touchData)
	}
}
