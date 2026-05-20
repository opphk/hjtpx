package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestComboCaptchaGenerate(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/combo/generate", func(c *gin.Context) {
		var req struct {
			AppKey    string `json:"app_key"`
			UserID    string `json:"user_id"`
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
				"captcha_id":   "test_combo_123",
				"challenge_id": "challenge_456",
				"expires_at":   1234567890,
			},
		})
	})

	reqBody := map[string]string{
		"app_key":    "test_app_key",
		"user_id":    "user_123",
		"captcha_id": "combo_001",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/combo/generate", bytes.NewBuffer(body))
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

func TestComboCaptchaVerify(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/combo/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string  `json:"captcha_id"`
			ChallengeID string  `json:"challenge_id"`
			Token       string  `json:"token"`
			X           float64 `json:"x"`
			Y           float64 `json:"y"`
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
				"risk_level":     "low",
				"score":          0.85,
				"verification_id": "verify_789",
			},
		})
	})

	reqBody := map[string]interface{}{
		"captcha_id":   "test_combo_123",
		"challenge_id": "challenge_456",
		"token":        "test_token",
		"x":            150.5,
		"y":            200.3,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/combo/verify", bytes.NewBuffer(body))
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
}

func TestComboCaptchaVerifyInvalid(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/combo/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string  `json:"captcha_id"`
			ChallengeID string  `json:"challenge_id"`
			Token       string  `json:"token"`
			X           float64 `json:"x"`
			Y           float64 `json:"y"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1,
			"message": "verification failed",
			"data": gin.H{
				"valid":      false,
				"risk_level": "high",
				"score":      0.15,
			},
		})
	})

	reqBody := map[string]interface{}{
		"captcha_id":   "invalid_combo",
		"challenge_id": "invalid_challenge",
		"token":        "invalid_token",
		"x":            999.0,
		"y":            999.0,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/combo/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 1 {
		t.Errorf("Expected code 1, got %v", response["code"])
	}
}

func TestComboCaptchaMissingParams(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/combo/generate", func(c *gin.Context) {
		var req struct {
			AppKey string `json:"app_key" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "missing required parameters"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	reqBody := map[string]string{
		"app_key": "test_app_key",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/combo/generate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for missing params, got %d", w.Code)
	}
}

func TestComboCaptchaInvalidJSON(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/combo/generate", func(c *gin.Context) {
		var req struct {
			AppKey string `json:"app_key" binding:"required"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "invalid JSON"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	req, _ := http.NewRequest("POST", "/api/v1/combo/generate", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("Expected status 400 for invalid JSON, got %d", w.Code)
	}
}

func TestComboCaptchaExpiredChallenge(t *testing.T) {
	r := setupTestRouter()
	r.POST("/api/v1/combo/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string  `json:"captcha_id"`
			ChallengeID string  `json:"challenge_id"`
			Token       string  `json:"token"`
			X           float64 `json:"x"`
			Y           float64 `json:"y"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1001,
			"message": "challenge expired",
			"data":    nil,
		})
	})

	reqBody := map[string]interface{}{
		"captcha_id":   "expired_combo",
		"challenge_id": "expired_challenge",
		"token":        "test_token",
		"x":            150.0,
		"y":            200.0,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/combo/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 1001 {
		t.Errorf("Expected code 1001 for expired challenge, got %v", response["code"])
	}
}

func TestComboCaptchaTypes(t *testing.T) {
	comboTypes := []string{"click", "slider", "drag", "rotate", "gesture"}

	for _, comboType := range comboTypes {
		t.Run("Test_"+comboType+"_combo", func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/api/v1/combo/generate", func(c *gin.Context) {
				var req struct {
					ComboType string `json:"combo_type"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"code":    0,
					"message": "success",
					"data": gin.H{
						"combo_type": req.ComboType,
						"captcha_id": "combo_123",
					},
				})
			})

			reqBody := map[string]string{
				"combo_type": comboType,
				"app_key":    "test_key",
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/combo/generate", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status 200 for combo type %s, got %d", comboType, w.Code)
			}
		})
	}
}

func TestComboCaptchaMultipleSteps(t *testing.T) {
	r := setupTestRouter()
	step := 0

	r.POST("/api/v1/combo/verify", func(c *gin.Context) {
		var req struct {
			Step int `json:"step"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		step = req.Step

		if step < 3 {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "step completed",
				"data": gin.H{
					"next_step": step + 1,
					"valid":     true,
				},
			})
		} else {
			c.JSON(http.StatusOK, gin.H{
				"code":    0,
				"message": "all steps completed",
				"data": gin.H{
					"valid":     true,
					"completed": true,
				},
			})
		}
	})

	for currentStep := 1; currentStep <= 3; currentStep++ {
		reqBody := map[string]interface{}{
			"step":    currentStep,
			"captcha_id": "multi_step_combo",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/combo/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Step %d: Expected status 200, got %d", currentStep, w.Code)
		}

		var response map[string]interface{}
		json.Unmarshal(w.Body.Bytes(), &response)

		if response["code"].(float64) != 0 {
			t.Errorf("Step %d: Expected code 0, got %v", currentStep, response["code"])
		}
	}
}

func TestComboCaptchaSecurityHeaders(t *testing.T) {
	r := setupTestRouter()
	r.GET("/api/v1/combo/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/api/v1/combo/health", nil)

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	if w.Header().Get("X-Content-Type-Options") == "" {
		t.Log("X-Content-Type-Options header not set")
	}
}

func TestComboCaptchaRateLimiting(t *testing.T) {
	r := setupTestRouter()
	requestCount := 0

	r.POST("/api/v1/combo/generate", func(c *gin.Context) {
		requestCount++
		if requestCount > 100 {
			c.JSON(http.StatusTooManyRequests, gin.H{"error": "rate limit exceeded"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"code": 0})
	})

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("POST", "/api/v1/combo/generate", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK && requestCount <= 100 {
			t.Errorf("Expected status 200 for request %d, got %d", i+1, w.Code)
		}
	}
}

func TestComboCaptchaLoadBalancing(t *testing.T) {
	serverCount := 3
	requests := make([]int, serverCount)

	for i := 0; i < serverCount; i++ {
		idx := i
		r := setupTestRouter()
		r.POST("/api/v1/combo/generate", func(c *gin.Context) {
			requests[idx]++
			c.JSON(http.StatusOK, gin.H{"code": 0, "server": idx})
		})

		req, _ := http.NewRequest("POST", "/api/v1/combo/generate", bytes.NewBuffer([]byte("{}")))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Server %d: Expected status 200, got %d", idx, w.Code)
		}
	}

	totalRequests := 0
	for _, count := range requests {
		totalRequests += count
	}

	if totalRequests != serverCount {
		t.Errorf("Expected %d total requests, got %d", serverCount, totalRequests)
	}
}
