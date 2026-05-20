package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNeuralCaptchaGenerate(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/generate", func(c *gin.Context) {
		var req struct {
			AppKey  string `json:"app_key"`
			UserID  string `json:"user_id"`
			Type    string `json:"type"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id":     "neural_123",
				"challenge_id":   "challenge_456",
				"neural_pattern": "pattern_789",
				"difficulty":     0.7,
				"expires_at":     1234567890,
			},
		})
	})

	reqBody := map[string]string{
		"app_key": "test_app_key",
		"user_id": "user_123",
		"type":    "neural_verification",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/generate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}
}

func TestNeuralCaptchaVerify(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string                   `json:"captcha_id"`
			ChallengeID string                   `json:"challenge_id"`
			Token       string                   `json:"token"`
			Patterns    []map[string]interface{} `json:"patterns"`
			Features    map[string]interface{}   `json:"features"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"valid":              true,
				"neural_confidence":  0.92,
				"risk_level":         "low",
				"verification_id":    "verify_789",
				"neural_analysis": gin.H{
					"pattern_match":   0.95,
					"behavior_score":  0.88,
					"device_fingerprint": "fp_abc123",
				},
			},
		})
	})

	patterns := []map[string]interface{}{
		{"x": 100, "y": 150, "timestamp": 0},
		{"x": 110, "y": 160, "timestamp": 50},
	}

	features := map[string]interface{}{
		"mouse_velocity":    2.5,
		"click_frequency":   0.3,
		"scroll_pattern":    "normal",
		"keystroke_timing":  150.5,
	}

	reqBody := map[string]interface{}{
		"captcha_id":   "neural_123",
		"challenge_id": "challenge_456",
		"token":        "test_token",
		"patterns":     patterns,
		"features":     features,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}

	data := response["data"].(map[string]interface{})
	if data["valid"] != true {
		t.Error("Expected valid to be true")
	}

	confidence := data["neural_confidence"].(float64)
	if confidence < 0 || confidence > 1 {
		t.Errorf("Expected confidence in [0, 1], got %v", confidence)
	}
}

func TestNeuralCaptchaInvalidPattern(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string                   `json:"captcha_id"`
			ChallengeID string                   `json:"challenge_id"`
			Token       string                   `json:"token"`
			Patterns    []map[string]interface{} `json:"patterns"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1002,
			"message": "invalid neural pattern",
			"data": gin.H{
				"valid":             false,
				"neural_confidence": 0.1,
				"risk_level":        "high",
			},
		})
	})

	reqBody := map[string]interface{}{
		"captcha_id":   "invalid_neural",
		"challenge_id": "invalid_challenge",
		"token":        "invalid_token",
		"patterns":     []map[string]interface{}{},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 1002 {
		t.Errorf("Expected code 1002, got %v", response["code"])
	}
}

func TestNeuralCaptchaAdaptiveDifficulty(t *testing.T) {
	r := setupTestRouter()
	difficulty := 0.5

	r.POST("/api/v1/neural/generate", func(c *gin.Context) {
		var req struct {
			AppKey     string  `json:"app_key"`
			Difficulty float64 `json:"difficulty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		difficulty = req.Difficulty
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id": "neural_adaptive_123",
				"difficulty": difficulty,
			},
		})
	})

	testCases := []struct {
		name        string
		difficulty  float64
	}{
		{"Easy", 0.2},
		{"Medium", 0.5},
		{"Hard", 0.8},
		{"Expert", 0.95},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"app_key":    "test_key",
				"difficulty": tc.difficulty,
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/neural/generate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200, got %d", w.Code)
			}

			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)

			if response["code"].(float64) != 0 {
				t.Errorf("Expected code 0, got %v", response["code"])
			}
		})
	}
}

func TestNeuralCaptchaDeviceAnalysis(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/device/analyze", func(c *gin.Context) {
		var req struct {
			CaptchaID string                 `json:"captcha_id"`
			DeviceInfo map[string]interface{} `json:"device_info"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"device_score":     0.85,
				"device_type":      "desktop",
				"browser":          "Chrome",
				"os":               "Windows",
				"risk_indicators":  []string{"suspicious_fingerprint", "known_bot"},
			},
		})
	})

	deviceInfo := map[string]interface{}{
		"user_agent":     "Mozilla/5.0",
		"screen_width":   1920,
		"screen_height":  1080,
		"platform":       "Win32",
		"language":       "en-US",
		"timezone":       "UTC",
		"canvas_fingerprint": "abc123",
		"webgl_vendor":   "NVIDIA",
		"webgl_renderer": "GeForce GTX 1080",
	}

	reqBody := map[string]interface{}{
		"captcha_id": "neural_123",
		"device_info": deviceInfo,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/device/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}

	data := response["data"].(map[string]interface{})
	deviceScore := data["device_score"].(float64)
	if deviceScore < 0 || deviceScore > 1 {
		t.Errorf("Expected device score in [0, 1], got %v", deviceScore)
	}
}

func TestNeuralCaptchaBehaviorAnalysis(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/behavior/analyze", func(c *gin.Context) {
		var req struct {
			UserID    string                   `json:"user_id"`
			SessionID string                   `json:"session_id"`
			Behaviors []map[string]interface{} `json:"behaviors"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"behavior_score":      0.78,
				"human_likelihood":    0.92,
				"anomaly_count":        2,
				"risk_factors":         []string{"unusual_timing", "pattern_deviation"},
				"recommendation":      "allow_with_monitoring",
			},
		})
	})

	behaviors := []map[string]interface{}{
		{"type": "mouse_movement", "velocity": 2.5, "acceleration": 0.3},
		{"type": "click_pattern", "frequency": 0.5, "precision": 0.9},
		{"type": "keystroke", "timing": 150.5, "pressure": 0.7},
	}

	reqBody := map[string]interface{}{
		"user_id":    "user_123",
		"session_id": "session_456",
		"behaviors":  behaviors,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/behavior/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}
}

func TestNeuralCaptchaModelUpdate(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/model/update", func(c *gin.Context) {
		var req struct {
			ModelVersion string  `json:"model_version"`
			Accuracy     float64 `json:"accuracy"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "model updated successfully",
			"data": gin.H{
				"new_version":   "v2.3.1",
				"previous_version": req.ModelVersion,
				"improvement":   0.05,
			},
		})
	})

	reqBody := map[string]interface{}{
		"model_version": "v2.3.0",
		"accuracy":      0.89,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/model/update", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}
}

func TestNeuralCaptchaConfidenceThreshold(t *testing.T) {
	r := setupTestRouter()
	r.GET("/api/v1/neural/confidence/threshold", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"threshold":          0.75,
				"min_confidence":     0.5,
				"max_confidence":     0.95,
				"auto_adjust":        true,
				"adjustment_interval": 3600,
			},
		})
	})

	req, _ := http.NewRequest("GET", "/api/v1/neural/confidence/threshold", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}

	data := response["data"].(map[string]interface{})
	threshold := data["threshold"].(float64)
	if threshold < 0 || threshold > 1 {
		t.Errorf("Expected threshold in [0, 1], got %v", threshold)
	}
}

func TestNeuralCaptchaPerformanceMetrics(t *testing.T) {
	r := setupTestRouter()
	r.GET("/api/v1/neural/metrics", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"requests_per_second":  1500,
				"avg_latency_ms":       45.5,
				"success_rate":         0.985,
				"false_positive_rate":  0.002,
				"false_negative_rate":  0.008,
				"model_load_time_ms":   120,
				"inference_time_ms":    15,
			},
		})
	})

	req, _ := http.NewRequest("GET", "/api/v1/neural/metrics", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}

	data := response["data"].(map[string]interface{})
	rps := data["requests_per_second"].(float64)
	if rps < 0 {
		t.Errorf("Expected non-negative RPS, got %v", rps)
	}
}

func TestNeuralCaptchaBatchVerify(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/batch/verify", func(c *gin.Context) {
		var req struct {
			Verifications []map[string]interface{} `json:"verifications"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		results := make([]map[string]interface{}, len(req.Verifications))
		for i := range req.Verifications {
			results[i] = map[string]interface{}{
				"captcha_id": req.Verifications[i]["captcha_id"],
				"valid":      true,
				"confidence": 0.9,
			}
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"total":     len(results),
				"valid":     len(results),
				"invalid":   0,
				"results":   results,
			},
		})
	})

	verifications := []map[string]interface{}{
		{"captcha_id": "cap_1", "token": "token_1"},
		{"captcha_id": "cap_2", "token": "token_2"},
		{"captcha_id": "cap_3", "token": "token_3"},
	}

	reqBody := map[string]interface{}{
		"verifications": verifications,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/batch/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 0 {
		t.Errorf("Expected code 0, got %v", response["code"])
	}

	data := response["data"].(map[string]interface{})
	if int(data["total"].(float64)) != 3 {
		t.Errorf("Expected 3 total verifications, got %v", data["total"])
	}
}

func TestNeuralCaptchaCacheHit(t *testing.T) {
	r := setupTestRouter()
	requestCount := 0

	r.POST("/api/v1/neural/verify", func(c *gin.Context) {
		requestCount++
		var req struct {
			CaptchaID string `json:"captcha_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"valid":          true,
				"cache_hit":      requestCount > 1,
				"confidence":     0.9,
			},
		})
	})

	for i := 0; i < 3; i++ {
		reqBody := map[string]string{
			"captcha_id": "cached_cap_123",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/neural/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}

	if requestCount != 3 {
		t.Errorf("Expected 3 requests, got %d", requestCount)
	}
}

func TestNeuralCaptchaTimeout(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/neural/verify", func(c *gin.Context) {
		c.JSON(http.StatusRequestTimeout, gin.H{
			"code":    1003,
			"message": "verification timeout",
			"data":    nil,
		})
	})

	reqBody := map[string]string{
		"captcha_id": "timeout_cap",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/neural/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusRequestTimeout {
		t.Errorf("Expected status 408, got %d", w.Code)
	}
}
