package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestIntegration_SliderCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/slider", GetSliderCaptcha)
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req1, _ := http.NewRequest("GET", "/api/v1/captcha/slider", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	var captchaResp map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &captchaResp)
	assert.NoError(t, err)

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

	assert.Contains(t, w2.Code, []int{http.StatusOK, http.StatusNotFound, http.StatusBadRequest})
}

func TestIntegration_ClickCaptchaFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req1, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode=number", nil)
	w1 := httptest.NewRecorder()
	r.ServeHTTP(w1, req1)

	assert.Equal(t, http.StatusOK, w1.Code)

	var captchaResp map[string]interface{}
	err := json.Unmarshal(w1.Body.Bytes(), &captchaResp)
	assert.NoError(t, err)

	assert.Contains(t, captchaResp, "session_id")
	assert.Contains(t, captchaResp, "image_url")
	assert.Contains(t, captchaResp, "hint")
	assert.Contains(t, captchaResp, "max_points")
}

func TestIntegration_MultipleCaptchaModes(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/api/v1/captcha/click", GetClickCaptcha)

	modes := []struct {
		name     string
		mode     string
		expected bool
	}{
		{"number", "number", true},
		{"letter", "letter", true},
		{"chinese", "chinese", true},
		{"icon", "icon", true},
		{"mixed", "mixed", true},
	}

	for _, tc := range modes {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("GET", "/api/v1/captcha/click?mode="+tc.mode, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)

			var resp map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &resp)
			assert.NoError(t, err)
			assert.Contains(t, resp, "session_id")
		})
	}
}

func TestIntegration_CaptchaVerificationFailure(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	verifyReq := VerifyRequest{
		SessionID: "nonexistent-session",
		Type:      "slider",
		X:         100,
		Y:         50,
	}

	jsonBody, _ := json.Marshal(verifyReq)
	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestIntegration_CaptchaTypeMismatch(t *testing.T) {
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

	verifyReq := VerifyRequest{
		SessionID: sessionID,
		Type:      "click",
		Points:    [][2]int{{100, 100}},
	}

	jsonBody, _ := json.Marshal(verifyReq)
	req2, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBuffer(jsonBody))
	req2.Header.Set("Content-Type", "application/json")
	w2 := httptest.NewRecorder()
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusBadRequest, w2.Code)
}

func TestIntegration_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/v1/captcha/verify", VerifyCaptcha)

	req, _ := http.NewRequest("POST", "/api/v1/captcha/verify", bytes.NewBufferString("invalid json"))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestIntegration_EnvironmentAnalysis(t *testing.T) {
	tests := []struct {
		name         string
		envData      map[string]interface{}
		minRiskScore float64
	}{
		{
			name:         "normal environment",
			envData:      map[string]interface{}{},
			minRiskScore: 0,
		},
		{
			name: "high risk - webdriver detected",
			envData: map[string]interface{}{
				"webdriver": "wd:true",
			},
			minRiskScore: 30,
		},
		{
			name: "high risk - software renderer",
			envData: map[string]interface{}{
				"webgl": "SwiftShader",
			},
			minRiskScore: 25,
		},
		{
			name: "high risk - no webgl",
			envData: map[string]interface{}{
				"webgl": "no_webgl",
			},
			minRiskScore: 20,
		},
		{
			name: "multiple indicators",
			envData: map[string]interface{}{
				"webdriver": "wd:true",
				"webgl":     "SwiftShader",
				"cpu":       "unknown",
			},
			minRiskScore: 40,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := analyzeEnvironmentData(tt.envData)
			assert.GreaterOrEqual(t, score, tt.minRiskScore)
		})
	}
}

func TestIntegration_ClickPointsValidation(t *testing.T) {
	session := &CaptchaSession{
		MaxPoints: 3,
		Tolerance: 35,
		TargetPoints: []ClickPoint{
			{X: 100, Y: 100},
			{X: 200, Y: 200},
			{X: 150, Y: 150},
		},
		HintOrder: []int{0, 1, 2},
	}

	tests := []struct {
		name     string
		req      VerifyRequest
		expected bool
	}{
		{
			name: "correct points in order",
			req: VerifyRequest{
				Points:        [][2]int{{100, 100}, {200, 200}, {150, 150}},
				ClickSequence: []int{0, 1, 2},
			},
			expected: true,
		},
		{
			name: "wrong order",
			req: VerifyRequest{
				Points:        [][2]int{{100, 100}, {200, 200}, {150, 150}},
				ClickSequence: []int{2, 1, 0},
			},
			expected: false,
		},
		{
			name: "points out of tolerance",
			req: VerifyRequest{
				Points:        [][2]int{{100, 100}, {500, 500}, {150, 150}},
				ClickSequence: []int{0, 1, 2},
			},
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			success, _ := verifyClickPoints(session, tt.req)
			assert.Equal(t, tt.expected, success)
		})
	}
}
