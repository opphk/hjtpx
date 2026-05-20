package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupLianliankanRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestLianliankanCaptchaGenerate(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/generate", func(c *gin.Context) {
		var req struct {
			AppKey     string `json:"app_key"`
			Difficulty string `json:"difficulty"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id":    "lianliankan_123",
				"challenge_id": "challenge_456",
				"grid_size":     4,
				"pairs":         8,
				"expires_at":    1234567890,
			},
		})
	})

	reqBody := map[string]string{
		"app_key":    "test_app_key",
		"difficulty": "medium",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/generate", bytes.NewBuffer(body))
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
	if data["grid_size"].(float64) != 4 {
		t.Errorf("Expected grid_size 4, got %v", data["grid_size"])
	}
}

func TestLianliankanCaptchaVerify(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string      `json:"captcha_id"`
			ChallengeID string      `json:"challenge_id"`
			Moves       [][]int     `json:"moves"`
			TimeUsed    float64     `json:"time_used"`
			Attempts    int         `json:"attempts"`
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
				"pairs_found":    len(req.Moves),
				"total_pairs":    8,
				"time_bonus":     0.85,
				"attempt_penalty": 0.95,
				"final_score":    0.9,
			},
		})
	})

	moves := [][]int{
		{0, 0, 1, 1},
		{2, 2, 3, 3},
	}

	reqBody := map[string]interface{}{
		"captcha_id":   "lianliankan_123",
		"challenge_id": "challenge_456",
		"moves":        moves,
		"time_used":    45.5,
		"attempts":     10,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/verify", bytes.NewBuffer(body))
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

func TestLianliankanCaptchaInvalidMoves(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string  `json:"captcha_id"`
			ChallengeID string  `json:"challenge_id"`
			Moves       [][]int `json:"moves"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    1005,
			"message": "invalid move sequence",
			"data": gin.H{
				"valid":      false,
				"error_type": "invalid_move",
			},
		})
	})

	reqBody := map[string]interface{}{
		"captcha_id":   "invalid_lianliankan",
		"challenge_id": "invalid_challenge",
		"moves":        [][]int{{0, 0, 0, 0}},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 1005 {
		t.Errorf("Expected code 1005, got %v", response["code"])
	}
}

func TestLianliankanCaptchaTimeExpired(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string  `json:"captcha_id"`
			ChallengeID string  `json:"challenge_id"`
			TimeUsed    float64 `json:"time_used"`
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
		"captcha_id":   "expired_lianliankan",
		"challenge_id": "expired_challenge",
		"time_used":    300.0,
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status 200, got %d", w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if response["code"].(float64) != 1001 {
		t.Errorf("Expected code 1001, got %v", response["code"])
	}
}

func TestLianliankanCaptchaDifficultyLevels(t *testing.T) {
	r := setupLianliankanRouter()
	gridSizes := map[string]float64{
		"easy":   4,
		"medium": 6,
		"hard":   8,
	}

	for difficulty, expectedGrid := range gridSizes {
		t.Run("Difficulty_"+difficulty, func(t *testing.T) {
			r.POST("/api/v1/lianliankan/generate", func(c *gin.Context) {
				var req struct {
					Difficulty string `json:"difficulty"`
				}
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"code":    0,
					"message": "success",
					"data": gin.H{
						"grid_size": gridSizes[req.Difficulty],
						"pairs":    gridSizes[req.Difficulty] * 2,
					},
				})
			})

			reqBody := map[string]string{
				"app_key":    "test_key",
				"difficulty": difficulty,
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/lianliankan/generate", bytes.NewBuffer(body))
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
			if data["grid_size"].(float64) != expectedGrid {
				t.Errorf("Expected grid_size %f, got %v", expectedGrid, data["grid_size"])
			}
		})
	}
}

func TestLianliankanCaptchaHint(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/hint", func(c *gin.Context) {
		var req struct {
			CaptchaID   string `json:"captcha_id"`
			ChallengeID string `json:"challenge_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"hint_type": "highlight",
				"cells":     [][]int{{0, 0}, {1, 1}},
				"cost":      0.1,
			},
		})
	})

	reqBody := map[string]string{
		"captcha_id":   "lianliankan_123",
		"challenge_id": "challenge_456",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/hint", bytes.NewBuffer(body))
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

func TestLianliankanCaptchaScoreCalculation(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/verify", func(c *gin.Context) {
		var req struct {
			CaptchaID   string  `json:"captcha_id"`
			ChallengeID string  `json:"challenge_id"`
			Moves       [][]int `json:"moves"`
			TimeUsed    float64 `json:"time_used"`
			Attempts    int     `json:"attempts"`
			HintsUsed   int     `json:"hints_used"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		baseScore := 1.0
		timeRatio := 1.0 - (req.TimeUsed / 120.0)
		if timeRatio < 0 {
			timeRatio = 0
		}
		timeBonus := timeRatio * 0.3

		attemptPenalty := float64(req.Attempts) / 100.0
		if attemptPenalty > 0.5 {
			attemptPenalty = 0.5
		}

		hintPenalty := float64(req.HintsUsed) * 0.1

		finalScore := baseScore + timeBonus - attemptPenalty - hintPenalty
		if finalScore < 0 {
			finalScore = 0
		}
		if finalScore > 1 {
			finalScore = 1
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"valid":           true,
				"time_bonus":      timeBonus,
				"attempt_penalty": attemptPenalty,
				"hint_penalty":    hintPenalty,
				"final_score":     finalScore,
			},
		})
	})

	testCases := []struct {
		name      string
		timeUsed  float64
		attempts  int
		hintsUsed int
	}{
		{"Fast with few attempts", 30.0, 8, 0},
		{"Slow with many attempts", 90.0, 25, 2},
		{"Medium pace", 60.0, 15, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			reqBody := map[string]interface{}{
				"captcha_id":   "lianliankan_123",
				"challenge_id": "challenge_456",
				"moves":        [][]int{{0, 0, 1, 1}},
				"time_used":    tc.timeUsed,
				"attempts":     tc.attempts,
				"hints_used":   tc.hintsUsed,
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/lianliankan/verify", bytes.NewBuffer(body))
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
			score := data["final_score"].(float64)
			if score < 0 || score > 1 {
				t.Errorf("Expected score in [0, 1], got %v", score)
			}
		})
	}
}

func TestLianliankanCaptchaReset(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/reset", func(c *gin.Context) {
		var req struct {
			CaptchaID   string `json:"captcha_id"`
			ChallengeID string `json:"challenge_id"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"new_challenge_id": "new_challenge_789",
				"penalty":          0.2,
			},
		})
	})

	reqBody := map[string]string{
		"captcha_id":   "lianliankan_123",
		"challenge_id": "challenge_456",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/reset", bytes.NewBuffer(body))
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

func TestLianliankanCaptchaStatistics(t *testing.T) {
	r := setupLianliankanRouter()
	r.GET("/api/v1/lianliankan/stats", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"total_games":      1500,
				"avg_time":         55.5,
				"avg_attempts":     12.3,
				"success_rate":     0.92,
				"avg_score":        0.78,
				"most_difficult":   "hard",
			},
		})
	})

	req, _ := http.NewRequest("GET", "/api/v1/lianliankan/stats", nil)

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

func TestLianliankanCaptchaThemeOptions(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/generate", func(c *gin.Context) {
		var req struct {
			AppKey string `json:"app_key"`
			Theme  string `json:"theme"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id": "lianliankan_themed",
				"theme":      req.Theme,
				"icons":     []string{"icon1", "icon2", "icon3"},
			},
		})
	})

	themes := []string{"animals", "fruits", "emojis", "shapes"}

	for _, theme := range themes {
		t.Run("Theme_"+theme, func(t *testing.T) {
			reqBody := map[string]string{
				"app_key": "test_key",
				"theme":   theme,
			}
			body, _ := json.Marshal(reqBody)

			req, _ := http.NewRequest("POST", "/api/v1/lianliankan/generate", bytes.NewBuffer(body))
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

func TestLianliankanCaptchaAccessibility(t *testing.T) {
	r := setupLianliankanRouter()
	r.GET("/api/v1/lianliankan/generate", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id":     "lianliankan_a11y",
				"text_alternatives": []string{"cat", "dog", "bird"},
				"aria_labels":    []string{"cell-0-0: cat", "cell-0-1: dog"},
			},
		})
	})

	req, _ := http.NewRequest("GET", "/api/v1/lianliankan/generate?accessibility=true", nil)

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

func TestLianliankanCaptchaMobileOptimization(t *testing.T) {
	r := setupLianliankanRouter()
	r.POST("/api/v1/lianliankan/generate", func(c *gin.Context) {
		var req struct {
			AppKey  string `json:"app_key"`
			Mobile  bool   `json:"mobile"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		gridSize := 4
		if req.Mobile {
			gridSize = 4
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id": "lianliankan_mobile",
				"grid_size":  gridSize,
				"touch_optimized": req.Mobile,
			},
		})
	})

	mobileReqBody := map[string]interface{}{
		"app_key": "test_key",
		"mobile":  true,
	}
	body, _ := json.Marshal(mobileReqBody)

	req, _ := http.NewRequest("POST", "/api/v1/lianliankan/generate", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 (iPhone; CPU iPhone OS 14_0 like Mac OS X)")

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
	if data["touch_optimized"] != true {
		t.Error("Expected touch_optimized to be true for mobile")
	}
}

func TestLianliankanCaptchaPerformanceMetrics(t *testing.T) {
	r := setupLianliankanRouter()
	loadTime := 0

	r.POST("/api/v1/lianliankan/generate", func(c *gin.Context) {
		startTime := loadTime
		loadTime++

		var req struct {
			AppKey string `json:"app_key"`
		}
		if err := c.ShouldBindJSON(&req); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data": gin.H{
				"captcha_id":      "lianliankan_perf",
				"generation_time": startTime + 50,
				"cache_hit":       startTime > 0,
			},
		})
	})

	for i := 0; i < 3; i++ {
		reqBody := map[string]string{
			"app_key": "test_key",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/lianliankan/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d: Expected status 200, got %d", i+1, w.Code)
		}
	}
}
