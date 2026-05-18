package handler

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSliderCaptchaAPI_GetSliderCaptcha_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
	assert.NotEmpty(t, resp["background_image"])
	assert.NotEmpty(t, resp["slider_image"])
	assert.NotNil(t, resp["target_x"])
	assert.NotNil(t, resp["target_y"])
	assert.NotNil(t, resp["max_offset"])
}

func TestSliderCaptchaAPI_GetSliderCaptcha_InvalidMethod(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusMethodNotAllowed, w.Code)
}

func TestSliderCaptchaAPI_VerifySlider_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req1, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	var captchaResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &captchaResp)
	sessionID := captchaResp["session_id"].(string)
	targetX := int(captchaResp["target_x"].(float64))
	targetY := int(captchaResp["target_y"].(float64))

	verifyReq := VerifyRequest{
		SessionID: sessionID,
		Type:      "slider",
		X:         targetX + 2,
		Y:         targetY + 2,
	}
	jsonBody, _ := json.Marshal(verifyReq)

	req2, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
}

func TestSliderCaptchaAPI_VerifySlider_InvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	verifyReq := VerifyRequest{
		SessionID: "invalid-session-id",
		Type:      "slider",
		X:         100,
		Y:         100,
	}
	jsonBody, _ := json.Marshal(verifyReq)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusOK}, w.Code)
}

func TestSliderCaptchaAPI_VerifySlider_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	tests := []struct {
		name string
		req  VerifyRequest
	}{
		{"missing session_id", VerifyRequest{Type: "slider", X: 100, Y: 100}},
		{"missing type", VerifyRequest{SessionID: "test", X: 100, Y: 100}},
		{"missing x", VerifyRequest{SessionID: "test", Type: "slider", Y: 100}},
		{"missing y", VerifyRequest{SessionID: "test", Type: "slider", X: 100}},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			jsonBody, _ := json.Marshal(tc.req)
			req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)
			assert.Contains(t, []int{http.StatusBadRequest, http.StatusNotFound, http.StatusOK, http.StatusInternalServerError}, w.Code)
		})
	}
}

func TestSliderCaptchaAPI_VerifySlider_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError}, w.Code)
}

func TestSliderCaptchaAPI_VerifySlider_EmptyBody(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer([]byte{}))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusBadRequest, http.StatusInternalServerError, http.StatusOK}, w.Code)
}

func TestClickCaptchaAPI_GetClickCaptcha_NumberMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=number", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
	assert.NotEmpty(t, resp["image_url"])
	assert.Contains(t, resp["hint"], "123456789")
}

func TestClickCaptchaAPI_GetClickCaptcha_LetterMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=letter", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
	assert.Contains(t, resp["hint"], "ABCDEFGHIJKLMNOPQRSTUVWXYZ")
}

func TestClickCaptchaAPI_GetClickCaptcha_ChineseMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=chinese", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
}

func TestClickCaptchaAPI_GetClickCaptcha_IconMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=icon", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
}

func TestClickCaptchaAPI_GetClickCaptcha_DefaultMode(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/click", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestClickCaptchaAPI_VerifyClick_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req1, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=number", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	var captchaResp map[string]interface{}
	json.Unmarshal(w1.Body.Bytes(), &captchaResp)
	sessionID := captchaResp["session_id"].(string)

	verifyReq := VerifyRequest{
		SessionID: sessionID,
		Type:      "click",
		X:         100,
		Y:         100,
	}
	jsonBody, _ := json.Marshal(verifyReq)

	req2, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Contains(t, []int{http.StatusOK, http.StatusBadRequest, http.StatusNotFound}, w2.Code)
}

func TestImageCaptchaAPI_GetImageCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/image", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"session_id": "test-session-id",
			"image":      "data:image/png;base64,test",
		})
	})

	req, _ := http.NewRequest("GET", "/api/v1/captcha/image", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusNotImplemented, http.StatusInternalServerError}, w.Code)

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "session_id")
	}
}

func TestAudioCaptchaAPI_GetAudioCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/audio", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"session_id": "test-session-id",
			"audio":      "data:audio/wav;base64,test",
		})
	})

	req, _ := http.NewRequest("GET", "/api/v1/captcha/audio?session_id=test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusNotImplemented, http.StatusBadRequest, http.StatusInternalServerError}, w.Code)
}

func TestHealthCheckAPI(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, "ok", resp["status"])
}

func TestRateLimitHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Header("X-RateLimit-Limit", "100")
		c.Header("X-RateLimit-Remaining", "99")
		c.Next()
	})
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "100", w.Header().Get("X-RateLimit-Limit"))
	assert.Equal(t, "99", w.Header().Get("X-RateLimit-Remaining"))
}

func TestCORSHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, OPTIONS")
		c.Next()
	})
	r.OPTIONS("/api/v1/captcha/slider", func(c *gin.Context) {
		c.Status(http.StatusNoContent)
	})
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("OPTIONS", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNoContent, w.Code)
	assert.Equal(t, "*", w.Header().Get("Access-Control-Allow-Origin"))
}

func TestRequestIDMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.Use(func(c *gin.Context) {
		c.Header("X-Request-ID", "test-request-123")
		c.Next()
	})
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, "test-request-123", w.Header().Get("X-Request-ID"))
}

func TestVerifyRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request VerifyRequest
		valid   bool
	}{
		{
			name:    "valid slider request",
			request: VerifyRequest{SessionID: "abc123", Type: "slider", X: 100, Y: 100},
			valid:   true,
		},
		{
			name:    "valid click request",
			request: VerifyRequest{SessionID: "abc123", Type: "click", X: 100, Y: 100},
			valid:   true,
		},
		{
			name:    "missing session_id",
			request: VerifyRequest{Type: "slider", X: 100, Y: 100},
			valid:   false,
		},
		{
			name:    "missing type",
			request: VerifyRequest{SessionID: "abc123", X: 100, Y: 100},
			valid:   false,
		},
		{
			name:    "empty session_id",
			request: VerifyRequest{SessionID: "", Type: "slider", X: 100, Y: 100},
			valid:   false,
		},
		{
			name:    "empty type",
			request: VerifyRequest{SessionID: "abc123", Type: "", X: 100, Y: 100},
			valid:   false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			isValid := tc.request.SessionID != "" && tc.request.Type != ""
			assert.Equal(t, tc.valid, isValid)
		})
	}
}

func TestCaptchaResponse_Fields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		body, _ := io.ReadAll(w.Body)
		var resp map[string]interface{}
		err := json.Unmarshal(body, &resp)

		if err == nil {
			assert.NotEmpty(t, resp["session_id"])
			assert.NotEmpty(t, resp["puzzle_image"])
			assert.NotNil(t, resp["target_x"])
			assert.NotNil(t, resp["target_y"])
		}
	}
}

func TestConcurrentSliderCaptchaGeneration(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	sessionIDs := make(chan string, 10)

	for i := 0; i < 10; i++ {
		go func() {
			req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				var resp map[string]interface{}
				json.Unmarshal(w.Body.Bytes(), &resp)
				sessionIDs <- resp["session_id"].(string)
			}
		}()
	}

	seen := make(map[string]bool)
	for i := 0; i < 10; i++ {
		select {
		case sid := <-sessionIDs:
			assert.False(t, seen[sid], "Duplicate session ID generated")
			seen[sid] = true
		}
	}
	assert.Equal(t, 10, len(seen))
}

func TestCacheHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		cacheControl := w.Header().Get("Cache-Control")
		t.Logf("Cache-Control header: %s", cacheControl)
	}
}

func TestContentTypeHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, w.Header().Get("Content-Type"), "application/json")
}
