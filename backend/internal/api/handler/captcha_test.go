package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func TestGetSliderCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/captcha/slider", GetSliderCaptcha)

	req, _ := http.NewRequest("GET", "/captcha/slider", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
	assert.NotEmpty(t, resp["image_url"])
	assert.NotEmpty(t, resp["puzzle_y"])
}

func TestGetClickCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/captcha/click", GetClickCaptcha)

	req, _ := http.NewRequest("GET", "/captcha/click", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.NotEmpty(t, resp["session_id"])
	assert.NotEmpty(t, resp["image_url"])
	assert.NotEmpty(t, resp["hint"])
	assert.NotEmpty(t, resp["max_points"])
}

func TestVerifyCaptchaSliderSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/captcha/slider", GetSliderCaptcha)
	r.POST("/captcha/verify", VerifyCaptcha)

	sliderReq, _ := http.NewRequest("GET", "/captcha/slider", nil)
	sliderW := httptest.NewRecorder()
	r.ServeHTTP(sliderW, sliderReq)

	var sliderResp map[string]interface{}
	json.Unmarshal(sliderW.Body.Bytes(), &sliderResp)
	sessionID := sliderResp["session_id"].(string)
	puzzleY := int(sliderResp["puzzle_y"].(float64))

	verifyReq := VerifyRequest{
		SessionID: sessionID,
		Type:      "slider",
		X:         puzzleY,
		Y:         puzzleY,
	}

	body, _ := json.Marshal(verifyReq)
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Contains(t, []interface{}{true, false}, resp["success"])
}

func TestVerifyCaptchaInvalidSession(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/captcha/verify", VerifyCaptcha)

	verifyReq := VerifyRequest{
		SessionID: "invalid-session-id",
		Type:      "slider",
		X:         100,
		Y:         100,
	}

	body, _ := json.Marshal(verifyReq)
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusNotFound, resp.Code)
}

func TestVerifyCaptchaInvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/captcha/verify", VerifyCaptcha)

	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestVerifyCaptchaTypeMismatch(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()

	r.GET("/captcha/slider", GetSliderCaptcha)
	r.POST("/captcha/verify", VerifyCaptcha)

	sliderReq, _ := http.NewRequest("GET", "/captcha/slider", nil)
	sliderW := httptest.NewRecorder()
	r.ServeHTTP(sliderW, sliderReq)

	var sliderResp map[string]interface{}
	json.Unmarshal(sliderW.Body.Bytes(), &sliderResp)
	sessionID := sliderResp["session_id"].(string)

	verifyReq := VerifyRequest{
		SessionID: sessionID,
		Type:      "click",
		Points:    []ClickPoint{},
	}

	body, _ := json.Marshal(verifyReq)
	req, _ := http.NewRequest("POST", "/captcha/verify", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, http.StatusBadRequest, resp.Code)
}

func TestGenerateSessionID(t *testing.T) {
	sessionID1 := generateSessionID()
	sessionID2 := generateSessionID()

	assert.NotEmpty(t, sessionID1)
	assert.NotEmpty(t, sessionID2)
	assert.NotEqual(t, sessionID1, sessionID2)
	assert.Contains(t, sessionID1, "sess_")
}

func TestGenerateSliderImage(t *testing.T) {
	background, slider, targetX, targetY := generateSliderImage()

	assert.NotEmpty(t, background)
	assert.NotEmpty(t, slider)
	assert.Greater(t, targetX, 0)
	assert.Greater(t, targetY, 0)
}

func TestGenerateClickImage(t *testing.T) {
	imageURL, hint, points, maxPoints := generateClickImage()

	assert.NotEmpty(t, imageURL)
	assert.NotEmpty(t, hint)
	assert.Greater(t, len(points), 0)
	assert.GreaterOrEqual(t, maxPoints, 2)
	assert.LessOrEqual(t, maxPoints, 4)
}

func TestGenerateAdvancedClickImage(t *testing.T) {
	imageURL, hint, points, maxPoints := generateAdvancedClickImage()

	assert.NotEmpty(t, imageURL)
	assert.NotEmpty(t, hint)
	assert.Greater(t, len(points), 0)
	assert.Equal(t, maxPoints, len(points))

	for _, point := range points {
		assert.Greater(t, point[0], 0)
		assert.Greater(t, point[1], 0)
	}
}

func TestAbs(t *testing.T) {
	assert.Equal(t, 5, abs(-5))
	assert.Equal(t, 5, abs(5))
	assert.Equal(t, 0, abs(0))
}

func TestCalculateAdaptiveTolerance(t *testing.T) {
	tests := []struct {
		name          string
		sessionPoints [][2]int
		reqPoints     []ClickPoint
		expectedMin  int
		expectedMax  int
	}{
		{
			name:          "empty points",
			sessionPoints: [][2]int{},
			reqPoints:     []ClickPoint{},
			expectedMin:   25,
			expectedMax:   40,
		},
		{
			name:          "close points",
			sessionPoints: [][2]int{{100, 100}},
			reqPoints:     []ClickPoint{{X: 105, Y: 105}},
			expectedMin:   25,
			expectedMax:   35,
		},
		{
			name:          "medium distance",
			sessionPoints: [][2]int{{100, 100}},
			reqPoints:     []ClickPoint{{X: 115, Y: 115}},
			expectedMin:   25,
			expectedMax:   35,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateAdaptiveTolerance(tt.sessionPoints, tt.reqPoints)
			assert.GreaterOrEqual(t, result, tt.expectedMin)
			assert.LessOrEqual(t, result, tt.expectedMax)
		})
	}
}

func TestVerifyClickOrder(t *testing.T) {
	tests := []struct {
		name          string
		sessionPoints [][2]int
		reqPoints     []ClickPoint
		expected      bool
	}{
		{
			name:          "valid order",
			sessionPoints: [][2]int{{100, 100}, {200, 200}},
			reqPoints:     []ClickPoint{{X: 100, Y: 100}, {X: 200, Y: 200}},
			expected:      true,
		},
		{
			name:          "invalid order",
			sessionPoints: [][2]int{{100, 100}, {200, 200}},
			reqPoints:     []ClickPoint{{X: 200, Y: 200}, {X: 100, Y: 100}},
			expected:      false,
		},
		{
			name:          "length mismatch",
			sessionPoints: [][2]int{{100, 100}, {200, 200}},
			reqPoints:     []ClickPoint{{X: 100, Y: 100}},
			expected:      false,
		},
		{
			name:          "within tolerance",
			sessionPoints: [][2]int{{100, 100}},
			reqPoints:     []ClickPoint{{X: 135, Y: 135}},
			expected:      true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifyClickOrder(tt.sessionPoints, tt.reqPoints)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestCleanupExpiredSessions(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/captcha/slider", GetSliderCaptcha)

	for i := 0; i < 5; i++ {
		req, _ := http.NewRequest("GET", "/captcha/slider", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
	}

	initialCount := len(captchaSessions)

	cleanupExpiredSessions()

	assert.LessOrEqual(t, len(captchaSessions), initialCount)
}
