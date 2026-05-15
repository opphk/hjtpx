package api

import (
	"bytes"
	"context"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"captchax/internal/service"

	"github.com/gin-gonic/gin"
)

type MockCaptchaService struct {
	GenerateSliderResult  *service.SliderCaptchaResult
	GenerateSliderErr     error
	VerifySliderResult    *service.SliderVerifyResult
	VerifySliderErr       error
	GenerateClickResult   *service.ClickCaptchaResult
	GenerateClickErr      error
	VerifyClickResult     *service.ClickVerifyResult
	VerifyClickErr        error
	GeneratePuzzleResult  *service.PuzzleCaptchaResult
	GeneratePuzzleErr     error
	VerifyPuzzleResult    *service.PuzzleVerifyResult
	VerifyPuzzleErr       error
	GenerateRotateResult  *service.RotateCaptchaResult
	GenerateRotateErr     error
	VerifyRotateResult    *service.RotateVerifyResult
	VerifyRotateErr       error
	GenerateTextResult    *service.TextCaptchaResult
	GenerateTextErr       error
	VerifyTextResult      *service.TextVerifyResult
	VerifyTextErr         error
	GenerateIconResult    *service.IconCaptchaResult
	GenerateIconErr       error
	VerifyIconResult      *service.IconVerifyResult
	VerifyIconErr         error
	GenerateAudioResult   *service.AudioCaptchaResult
	GenerateAudioErr      error
	VerifyAudioResult     *service.AudioVerifyResult
	VerifyAudioErr        error
	GenerateBehaviorResult *service.BehaviorCaptchaResult
	GenerateBehaviorErr    error
	VerifyBehaviorResult   *service.BehaviorVerifyResult
	VerifyBehaviorErr      error
}

func (m *MockCaptchaService) GenerateSliderCaptcha(ctx context.Context, appID, clientInfo string) (*service.SliderCaptchaResult, error) {
	return m.GenerateSliderResult, m.GenerateSliderErr
}

func (m *MockCaptchaService) VerifySliderCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*service.SliderVerifyResult, error) {
	return m.VerifySliderResult, m.VerifySliderErr
}

func (m *MockCaptchaService) GenerateClickCaptcha(ctx context.Context, appID string, charCount int, clientInfo string) (*service.ClickCaptchaResult, error) {
	return m.GenerateClickResult, m.GenerateClickErr
}

func (m *MockCaptchaService) VerifyClickCaptcha(ctx context.Context, captchaID string, clicks []service.CharPositionDTO) (*service.ClickVerifyResult, error) {
	return m.VerifyClickResult, m.VerifyClickErr
}

func (m *MockCaptchaService) GeneratePuzzleCaptcha(ctx context.Context, appID, clientInfo string) (*service.PuzzleCaptchaResult, error) {
	return m.GeneratePuzzleResult, m.GeneratePuzzleErr
}

func (m *MockCaptchaService) VerifyPuzzleCaptcha(ctx context.Context, captchaID string, targetX, targetY int) (*service.PuzzleVerifyResult, error) {
	return m.VerifyPuzzleResult, m.VerifyPuzzleErr
}

func (m *MockCaptchaService) GenerateRotateCaptcha(ctx context.Context, appID, clientInfo string) (*service.RotateCaptchaResult, error) {
	return m.GenerateRotateResult, m.GenerateRotateErr
}

func (m *MockCaptchaService) VerifyRotateCaptcha(ctx context.Context, captchaID string, angle int) (*service.RotateVerifyResult, error) {
	return m.VerifyRotateResult, m.VerifyRotateErr
}

func (m *MockCaptchaService) GenerateTextCaptcha(ctx context.Context, appID, clientInfo string) (*service.TextCaptchaResult, error) {
	return m.GenerateTextResult, m.GenerateTextErr
}

func (m *MockCaptchaService) VerifyTextCaptcha(ctx context.Context, captchaID string, code string) (*service.TextVerifyResult, error) {
	return m.VerifyTextResult, m.VerifyTextErr
}

func (m *MockCaptchaService) GenerateIconCaptcha(ctx context.Context, appID, clientInfo string) (*service.IconCaptchaResult, error) {
	return m.GenerateIconResult, m.GenerateIconErr
}

func (m *MockCaptchaService) VerifyIconCaptcha(ctx context.Context, captchaID string, iconIDs []string) (*service.IconVerifyResult, error) {
	return m.VerifyIconResult, m.VerifyIconErr
}

func (m *MockCaptchaService) GenerateAudioCaptcha(ctx context.Context, appID, clientInfo string) (*service.AudioCaptchaResult, error) {
	return m.GenerateAudioResult, m.GenerateAudioErr
}

func (m *MockCaptchaService) VerifyAudioCaptcha(ctx context.Context, captchaID string, code string) (*service.AudioVerifyResult, error) {
	return m.VerifyAudioResult, m.VerifyAudioErr
}

func (m *MockCaptchaService) GenerateBehaviorCaptcha(ctx context.Context, challengeType string) (*service.BehaviorCaptchaResult, error) {
	return m.GenerateBehaviorResult, m.GenerateBehaviorErr
}

func (m *MockCaptchaService) VerifyBehaviorCaptcha(ctx context.Context, req *service.BehaviorVerifyRequest) (*service.BehaviorVerifyResult, error) {
	return m.VerifyBehaviorResult, m.VerifyBehaviorErr
}

func (m *MockCaptchaService) LogVerification(ctx context.Context, appID, captchaType, captchaID string, success bool, message, ip string) {
}

func setupRouter(handler *Handler) *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/health", handler.HealthCheck)
	r.POST("/api/v1/captcha/slider/generate", handler.getSliderCaptcha)
	r.POST("/api/v1/captcha/slider/verify", handler.verifySliderCaptcha)
	r.POST("/api/v1/captcha/click/generate", handler.getClickCaptcha)
	r.POST("/api/v1/captcha/click/verify", handler.verifyClickCaptcha)
	return r
}

func TestHealthEndpoint(t *testing.T) {
	handler := &Handler{}

	router := gin.New()
	router.GET("/health", handler.HealthCheck)

	t.Run("Health check returns 200", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("Health check returns healthy status", func(t *testing.T) {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		if status, ok := data["status"].(string); !ok || status != "healthy" {
			t.Errorf("Status = %v, want 'healthy'", data["status"])
		}

		if service, ok := data["service"].(string); !ok || service != "captchax-api" {
			t.Errorf("Service = %v, want 'captchax-api'", data["service"])
		}
	})
}

func TestSliderEndpoint(t *testing.T) {
	t.Run("Generate slider captcha - success", func(t *testing.T) {
		mockService := &MockCaptchaService{
			GenerateSliderResult: &service.SliderCaptchaResult{
				ID:            "test-slider-123",
				BackgroundB64: "base64background",
				SliderB64:     "base64slider",
				TargetX:       150,
				TargetY:       100,
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		reqBody := SliderGenerateRequest{
			AppID:      "test-app",
			Width:      300,
			Height:     200,
			ClientInfo: "test-client",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if code, ok := response["code"].(float64); !ok || code != 0 {
			t.Errorf("Response code = %v, want 0", response["code"])
		}
	})

	t.Run("Generate slider captcha - invalid request body", func(t *testing.T) {
		mockService := &MockCaptchaService{}
		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/generate", bytes.NewBuffer([]byte("invalid json")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Verify slider captcha - success", func(t *testing.T) {
		mockService := &MockCaptchaService{
			VerifySliderResult: &service.SliderVerifyResult{
				Success: true,
				Message: "verification successful",
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		reqBody := SliderVerifyRequest{
			CaptchaID: "test-slider-123",
			TargetX:  150,
			TargetY:  100,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		if success, ok := data["success"].(bool); !ok || !success {
			t.Errorf("Success = %v, want true", data["success"])
		}
	})

	t.Run("Verify slider captcha - invalid request body", func(t *testing.T) {
		mockService := &MockCaptchaService{}
		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/verify", bytes.NewBuffer([]byte("invalid")))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusBadRequest)
		}
	})

	t.Run("Verify slider captcha - verification failed", func(t *testing.T) {
		mockService := &MockCaptchaService{
			VerifySliderResult: &service.SliderVerifyResult{
				Success: false,
				Message: "verification failed",
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		reqBody := SliderVerifyRequest{
			CaptchaID: "test-slider-123",
			TargetX:  200,
			TargetY:  200,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/slider/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		if success, ok := data["success"].(bool); !ok || success {
			t.Errorf("Success = %v, want false", data["success"])
		}
	})
}

func TestClickEndpoint(t *testing.T) {
	t.Run("Generate click captcha - success", func(t *testing.T) {
		mockService := &MockCaptchaService{
			GenerateClickResult: &service.ClickCaptchaResult{
				ID:            "test-click-123",
				Image:         "base64image",
				TargetChars:   []string{"中", "心", "测", "试"},
				CharPositions: []service.CharPositionDTO{
					{Char: "中", X: 50, Y: 80, Width: 40, Height: 50},
					{Char: "心", X: 125, Y: 80, Width: 40, Height: 50},
					{Char: "测", X: 200, Y: 80, Width: 40, Height: 50},
					{Char: "试", X: 275, Y: 80, Width: 40, Height: 50},
				},
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		reqBody := ClickGenerateRequest{
			AppID:      "test-app",
			CharCount:  4,
			ClientInfo: "test-client",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/click/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if code, ok := response["code"].(float64); !ok || code != 0 {
			t.Errorf("Response code = %v, want 0", response["code"])
		}
	})

	t.Run("Verify click captcha - success", func(t *testing.T) {
		mockService := &MockCaptchaService{
			VerifyClickResult: &service.ClickVerifyResult{
				Success: true,
				Score:   1.0,
				Message: "verification passed",
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := setupRouter(handler)

		reqBody := ClickVerifyRequest{
			CaptchaID: "test-click-123",
			Clicks: []service.CharPositionDTO{
				{X: 70, Y: 105},
				{X: 145, Y: 105},
				{X: 220, Y: 105},
				{X: 295, Y: 105},
			},
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/click/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}

		var response map[string]interface{}
		if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		data, ok := response["data"].(map[string]interface{})
		if !ok {
			t.Fatal("Response data is not a map")
		}

		if success, ok := data["success"].(bool); !ok || !success {
			t.Errorf("Success = %v, want true", data["success"])
		}

		if score, ok := data["score"].(float64); !ok || score < 0.8 {
			t.Errorf("Score = %v, want >= 0.8", data["score"])
		}
	})
}

func TestPuzzleEndpoint(t *testing.T) {
	t.Run("Generate puzzle captcha - success", func(t *testing.T) {
		mockService := &MockCaptchaService{
			GeneratePuzzleResult: &service.PuzzleCaptchaResult{
				ID:            "test-puzzle-123",
				BackgroundB64: "base64background",
				PuzzleB64:     "base64puzzle",
				TargetX:       150,
				TargetY:       100,
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := gin.New()
		router.POST("/api/v1/captcha/puzzle/generate", handler.getPuzzleCaptcha)

		reqBody := PuzzleGenerateRequest{
			AppID:      "test-app",
			Width:      300,
			Height:     200,
			ClientInfo: "test-client",
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/puzzle/generate", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}
	})

	t.Run("Verify puzzle captcha - success", func(t *testing.T) {
		mockService := &MockCaptchaService{
			VerifyPuzzleResult: &service.PuzzleVerifyResult{
				Success: true,
				Message: "verification successful",
			},
		}

		handler := &Handler{
			captchaService: mockService,
		}

		router := gin.New()
		router.POST("/api/v1/captcha/puzzle/verify", handler.verifyPuzzleCaptcha)

		reqBody := PuzzleVerifyRequest{
			CaptchaID: "test-puzzle-123",
			TargetX:   150,
			TargetY:   100,
		}
		body, _ := json.Marshal(reqBody)

		req, _ := http.NewRequest("POST", "/api/v1/captcha/puzzle/verify", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()

		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Status code = %d, want %d", w.Code, http.StatusOK)
		}
	})
}

func TestRequestBinding(t *testing.T) {
	t.Run("Slider generate request JSON binding", func(t *testing.T) {
		jsonStr := `{"app_id":"test","width":300,"height":200,"client_info":"info"}`
		var req SliderGenerateRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if req.AppID != "test" {
			t.Errorf("AppID = %s, want 'test'", req.AppID)
		}
		if req.Width != 300 {
			t.Errorf("Width = %d, want 300", req.Width)
		}
		if req.Height != 200 {
			t.Errorf("Height = %d, want 200", req.Height)
		}
		if req.ClientInfo != "info" {
			t.Errorf("ClientInfo = %s, want 'info'", req.ClientInfo)
		}
	})

	t.Run("Slider verify request JSON binding", func(t *testing.T) {
		jsonStr := `{"captcha_id":"abc123","target_x":150,"target_y":100}`
		var req SliderVerifyRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if req.CaptchaID != "abc123" {
			t.Errorf("CaptchaID = %s, want 'abc123'", req.CaptchaID)
		}
		if req.TargetX != 150 {
			t.Errorf("TargetX = %d, want 150", req.TargetX)
		}
		if req.TargetY != 100 {
			t.Errorf("TargetY = %d, want 100", req.TargetY)
		}
	})

	t.Run("Click verify request JSON binding", func(t *testing.T) {
		jsonStr := `{"captcha_id":"click123","clicks":[{"x":70,"y":105},{"x":145,"y":105}]}`
		var req ClickVerifyRequest
		if err := json.Unmarshal([]byte(jsonStr), &req); err != nil {
			t.Fatalf("Failed to unmarshal: %v", err)
		}

		if req.CaptchaID != "click123" {
			t.Errorf("CaptchaID = %s, want 'click123'", req.CaptchaID)
		}
		if len(req.Clicks) != 2 {
			t.Errorf("Clicks length = %d, want 2", len(req.Clicks))
		}
	})
}
