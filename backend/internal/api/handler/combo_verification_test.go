package handler

import (
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

func TestCreateComboVerificationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/combo-verification/flow", CreateComboVerificationFlow)

	reqBody := `{
		"user_id": "test_user_123",
		"client_ip": "192.168.1.1",
		"user_agent": "test-agent",
		"fingerprint": "test-fingerprint",
		"risk_score": 50,
		"strategy": "any"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/combo-verification/flow", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var response service.CreateFlowResponse
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.FlowID == "" {
		t.Error("Expected FlowID to be set")
	}

	if len(response.Steps) == 0 {
		t.Error("Expected at least one step")
	}
}

func TestGetComboVerificationFlow(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/combo-verification/flow", CreateComboVerificationFlow)
	r.GET("/api/v1/combo-verification/flow/:flow_id", GetComboVerificationFlow)

	// First create a flow
	reqBody := `{
		"user_id": "test_user_456",
		"client_ip": "192.168.1.2",
		"user_agent": "test-agent",
		"fingerprint": "test-fingerprint",
		"risk_score": 30,
		"strategy": "any"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/combo-verification/flow", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Failed to create flow: %d", w.Code)
	}

	var createResp service.CreateFlowResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	if err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}

	// Then get the flow
	req = httptest.NewRequest(http.MethodGet, "/api/v1/combo-verification/flow/"+createResp.FlowID, nil)
	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestSelectSmartCaptcha(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/smart-captcha/select", SelectSmartCaptcha)

	reqBody := `{
		"user_id": "test_user_789",
		"risk_score": 75,
		"device_type": "desktop",
		"platform": "web",
		"network_quality": 0.8,
		"accessibility_required": false,
		"previous_attempts": 0,
		"time_constraints": false,
		"geo_location": "CN"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/smart-captcha/select", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Body.String())
	}

	var response service.CaptchaSelectionResult
	err := json.Unmarshal(w.Body.Bytes(), &response)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if response.CaptchaType == "" {
		t.Error("Expected CaptchaType to be set")
	}

	if response.Name == "" {
		t.Error("Expected Name to be set")
	}
}

func TestGetUserDifficulty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/v1/dynamic-difficulty/user/:user_id", GetUserDifficulty)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/dynamic-difficulty/user/test_user_1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}
}

func TestUpdateBehaviorAndAdjustDifficulty(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/dynamic-difficulty/behavior", UpdateBehaviorAndAdjustDifficulty)

	reqBody := `{
		"user_id": "test_user_behavior",
		"action_type": "verification",
		"duration": 2000000000,
		"accuracy": 0.95,
		"error_rate": 0
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/dynamic-difficulty/behavior", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d: %s", w.Code, w.Body.String())
	}
}

func TestGetAllCaptchaCapabilities(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.GET("/api/v1/smart-captcha/capabilities", GetAllCaptchaCapabilities)

	req := httptest.NewRequest(http.MethodGet, "/api/v1/smart-captcha/capabilities", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d", w.Code)
	}

	var capabilities []*service.CaptchaCapability
	err := json.Unmarshal(w.Body.Bytes(), &capabilities)
	if err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if len(capabilities) == 0 {
		t.Error("Expected at least one capability")
	}
}

func TestComboVerificationService_CreateFlow(t *testing.T) {
	service := service.NewComboVerificationService(
		service.NewEnhancedAdaptiveDifficultyService(),
	)

	req := &service.CreateFlowRequest{
		UserID:      "test_user",
		ClientIP:    "192.168.1.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
		RiskScore:   65,
		Strategy:    "majority",
	}

	ctx := context.Background()
	resp, err := service.CreateFlow(ctx, req)
	if err != nil {
		t.Fatalf("Failed to create flow: %v", err)
	}

	if resp.FlowID == "" {
		t.Error("Expected FlowID to be set")
	}

	if resp.Strategy != "majority" {
		t.Errorf("Expected strategy 'majority', got '%s'", resp.Strategy)
	}
}

func TestSmartCaptchaSelector_SelectCaptcha(t *testing.T) {
	selector := service.NewSmartCaptchaSelector()

	ctx := &service.SelectionContext{
		UserID:             "test_user",
		RiskScore:          85,
		DeviceType:         "desktop",
		Platform:           "web",
		NetworkQuality:     0.9,
		AccessibilityRequired: false,
		PreviousAttempts:   0,
		TimeConstraints:    false,
		GeoLocation:        "US",
	}

	result, err := selector.SelectCaptcha(context.Background(), ctx)
	if err != nil {
		t.Fatalf("Failed to select captcha: %v", err)
	}

	if result.CaptchaType == "" {
		t.Error("Expected CaptchaType to be set")
	}
}

func TestEnhancedDynamicDifficultyService_UpdateBehavior(t *testing.T) {
	service := service.NewEnhancedDynamicDifficultyService()

	behavior := &service.RealTimeBehavior{
		UserID:        "test_user_diff",
		ActionType:    "verification",
		Duration:      3000000000,
		Accuracy:      0.85,
		Deviation:     0.1,
		Velocity:      5.0,
		ErrorRate:     0.1,
		ConsecutiveSuccess: 2,
	}

	adjustment, err := service.UpdateBehavior(context.Background(), behavior)
	if err != nil {
		t.Fatalf("Failed to update behavior: %v", err)
	}

	_ = adjustment
}

func TestVerifyComboVerificationStep(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.Default()
	r.POST("/api/v1/combo-verification/flow", CreateComboVerificationFlow)
	r.POST("/api/v1/combo-verification/verify/step", VerifyComboVerificationStep)

	// First create a flow
	reqBody := `{
		"user_id": "test_user_verify",
		"client_ip": "192.168.1.3",
		"user_agent": "test-agent",
		"fingerprint": "test-fingerprint",
		"risk_score": 20,
		"strategy": "any"
	}`

	req := httptest.NewRequest(http.MethodPost, "/api/v1/combo-verification/flow", strings.NewReader(reqBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	var createResp service.CreateFlowResponse
	err := json.Unmarshal(w.Body.Bytes(), &createResp)
	if err != nil {
		t.Fatalf("Failed to parse create response: %v", err)
	}

	if len(createResp.Steps) == 0 {
		t.Skip("No steps created, skipping verification test")
	}

	// Then verify a step
	verifyBody := `{
		"flow_id": "` + createResp.FlowID + `",
		"step_id": "` + createResp.Steps[0].StepID + `",
		"captcha_type": "slider",
		"answer": {"x": 150}
	}`

	req = httptest.NewRequest(http.MethodPost, "/api/v1/combo-verification/verify/step", strings.NewReader(verifyBody))
	req.Header.Set("Content-Type", "application/json")

	w = httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status code 200, got %d: %s", w.Code, w.Body.String())
	}
}
