package captcha

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func TestClientCreation(t *testing.T) {
	client := NewClient("http://localhost:8080")
	if client == nil {
		t.Fatal("Client should not be nil")
	}
	if client.baseURL != "http://localhost:8080" {
		t.Errorf("Expected baseURL to be 'http://localhost:8080', got '%s'", client.baseURL)
	}
	if client.httpClient == nil {
		t.Error("HTTP client should not be nil")
	}
	if client.httpClient.Timeout != 30*time.Second {
		t.Errorf("Expected default timeout to be 30s, got %v", client.httpClient.Timeout)
	}
}

func TestClientWithOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 60 * time.Second}
	apiKey := "test-api-key"

	client := NewClient(
		"http://localhost:8080",
		WithHTTPClient(customClient),
		WithAPIKey(apiKey),
		WithTimeout(45*time.Second),
	)

	if client.httpClient != customClient {
		t.Error("Custom HTTP client should be set")
	}
	if client.apiKey != apiKey {
		t.Errorf("Expected API key to be '%s', got '%s'", apiKey, client.apiKey)
	}
	if client.httpClient.Timeout != 45*time.Second {
		t.Errorf("Expected timeout to be 45s, got %v", client.httpClient.Timeout)
	}
}

func TestGetSliderCaptchaSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/captcha/slider" {
			t.Errorf("Expected path '/api/v1/captcha/slider', got '%s'", r.URL.Path)
		}
		if r.URL.Query().Get("width") != "320" {
			t.Errorf("Expected width '320', got '%s'", r.URL.Query().Get("width"))
		}

		response := struct {
			Code    int                    `json:"code"`
			Message string                 `json:"message"`
			Data    SliderCaptchaResponse  `json:"data"`
		}{
			Code:    0,
			Message: "success",
			Data: SliderCaptchaResponse{
				SessionID:    "test-session-123",
				ImageURL:     "http://example.com/image.png",
				PuzzleURL:    "http://example.com/puzzle.png",
				SecretY:      80,
				ImageWidth:   320,
				ImageHeight:  160,
			},
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	captcha, err := client.GetSliderCaptcha(320, 160, 8)

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if captcha.SessionID != "test-session-123" {
		t.Errorf("Expected session ID 'test-session-123', got '%s'", captcha.SessionID)
	}
	if captcha.SecretY != 80 {
		t.Errorf("Expected secret Y '80', got '%d'", captcha.SecretY)
	}
}

func TestGetSliderCaptchaWithParams(t *testing.T) {
	tests := []struct {
		name       string
		width      int
		height     int
		tolerance  int
		wantWidth  string
		wantHeight string
		wantTol    string
	}{
		{"small", 200, 100, 5, "200", "100", "5"},
		{"medium", 320, 160, 8, "320", "160", "8"},
		{"large", 640, 320, 10, "640", "320", "10"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if r.URL.Query().Get("width") != tt.wantWidth {
					t.Errorf("Width: expected '%s', got '%s'", tt.wantWidth, r.URL.Query().Get("width"))
				}
				if r.URL.Query().Get("height") != tt.wantHeight {
					t.Errorf("Height: expected '%s', got '%s'", tt.wantHeight, r.URL.Query().Get("height"))
				}
				if r.URL.Query().Get("tolerance") != tt.wantTol {
					t.Errorf("Tolerance: expected '%s', got '%s'", tt.wantTol, r.URL.Query().Get("tolerance"))
				}

				response := struct {
					Code    int                   `json:"code"`
					Message string                `json:"message"`
					Data    SliderCaptchaResponse `json:"data"`
				}{
					Code:    0,
					Message: "success",
					Data:    SliderCaptchaResponse{SessionID: "test-session"},
				}
				json.NewEncoder(w).Encode(response)
			}))
			defer server.Close()

			client := NewClient(server.URL)
			_, err := client.GetSliderCaptcha(tt.width, tt.height, tt.tolerance)
			if err != nil {
				t.Fatalf("Unexpected error: %v", err)
			}
		})
	}
}

func TestVerifyCaptchaSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != http.MethodPost {
			t.Errorf("Expected POST method, got '%s'", r.Method)
		}
		if r.URL.Path != "/api/v1/captcha/verify" {
			t.Errorf("Expected path '/api/v1/captcha/verify', got '%s'", r.URL.Path)
		}

		var req VerifyCaptchaRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.SessionID != "test-session" {
			t.Errorf("Expected session ID 'test-session', got '%s'", req.SessionID)
		}
		if req.X != 150 {
			t.Errorf("Expected X '150', got '%d'", req.X)
		}

		response := struct {
			Code    int                      `json:"code"`
			Message string                   `json:"message"`
			Data    VerifyCaptchaResponse    `json:"data"`
		}{
			Code:    0,
			Message: "success",
			Data: VerifyCaptchaResponse{
				Success:     true,
				Message:     "Verification successful",
				Remaining:   3,
				TrajectoryResult: &TrajectoryResult{
					Score:   0.95,
					Passed:  true,
					Reasons: []string{"smooth trajectory"},
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	result, err := client.VerifyCaptcha(&VerifyCaptchaRequest{
		SessionID: "test-session",
		X:         150,
		Y:         80,
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Expected verification to be successful")
	}
	if result.TrajectoryResult == nil {
		t.Error("Expected trajectory result to be present")
	}
	if result.TrajectoryResult.Score != 0.95 {
		t.Errorf("Expected trajectory score '0.95', got '%f'", result.TrajectoryResult.Score)
	}
}

func TestVerifyCaptchaWithTrajectory(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req VerifyCaptchaRequest
		json.NewDecoder(r.Body).Decode(&req)

		if len(req.Trajectory) != 3 {
			t.Errorf("Expected 3 trajectory points, got %d", len(req.Trajectory))
		}

		for i, point := range req.Trajectory {
			if point.X == 0 || point.Y == 0 || point.T == 0 {
				t.Errorf("Trajectory point %d has invalid values", i)
			}
		}

		response := struct {
			Code    int                   `json:"code"`
			Message string                `json:"message"`
			Data    VerifyCaptchaResponse `json:"data"`
		}{
			Code:    0,
			Message: "success",
			Data:    VerifyCaptchaResponse{Success: true},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	now := time.Now().UnixMilli()

	result, err := client.VerifyCaptcha(&VerifyCaptchaRequest{
		SessionID: "test-session",
		X:         150,
		Trajectory: []TrajectoryPoint{
			{X: 0, Y: 80, T: now - 1000},
			{X: 75, Y: 82, T: now - 500},
			{X: 150, Y: 80, T: now},
		},
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if !result.Success {
		t.Error("Expected verification to be successful")
	}
}

func TestLoginSuccess(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/auth/login" {
			t.Errorf("Expected path '/api/v1/auth/login', got '%s'", r.URL.Path)
		}

		var req LoginRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.Username != "testuser" {
			t.Errorf("Expected username 'testuser', got '%s'", req.Username)
		}
		if req.Password != "password123" {
			t.Errorf("Expected password 'password123', got '%s'", req.Password)
		}

		response := struct {
			Code    int            `json:"code"`
			Message string         `json:"message"`
			Data    LoginResponse  `json:"data"`
		}{
			Code:    0,
			Message: "success",
			Data: LoginResponse{
				AccessToken:  "access-token-123",
				RefreshToken: "refresh-token-456",
				ExpiresIn:    3600,
				User: struct {
					ID       uint   `json:"id"`
					Username string `json:"username"`
					Email    string `json:"email"`
				}{
					ID:       1,
					Username: "testuser",
					Email:    "test@example.com",
				},
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	auth := client.Auth()

	result, err := auth.Login(&LoginRequest{
		Username: "testuser",
		Password: "password123",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if result.AccessToken != "access-token-123" {
		t.Errorf("Expected access token 'access-token-123', got '%s'", result.AccessToken)
	}
	if result.User.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", result.User.Username)
	}
	if result.ExpiresIn != 3600 {
		t.Errorf("Expected expires in 3600, got %d", result.ExpiresIn)
	}
}

func TestLoginWithCaptchaToken(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req LoginRequest
		json.NewDecoder(r.Body).Decode(&req)

		if req.CaptchaToken == "" {
			t.Error("Expected captcha token to be present")
		}

		response := struct {
			Code    int           `json:"code"`
			Message string        `json:"message"`
			Data    LoginResponse `json:"data"`
		}{
			Code:    0,
			Message: "success",
			Data: LoginResponse{
				AccessToken: "access-token",
				ExpiresIn:   3600,
			},
		}
		json.NewEncoder(w).Encode(response)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	auth := client.Auth()

	_, err := auth.Login(&LoginRequest{
		Username:     "testuser",
		Password:     "password123",
		CaptchaToken: "captcha-token-xyz",
	})

	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
}

func TestGetDetectionScript(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/v1/detect/script" {
			t.Errorf("Expected path '/api/v1/detect/script', got '%s'", r.URL.Path)
		}

		expectedCallback := "myCallback"
		if r.URL.Query().Get("callback") != expectedCallback {
			t.Errorf("Expected callback '%s', got '%s'", expectedCallback, r.URL.Query().Get("callback"))
		}

		w.Write([]byte(`console.log("detection script");`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	env := client.Env()

	script, err := env.GetDetectionScript(expectedCallback)
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if script == "" {
		t.Error("Expected script to be non-empty")
	}
}

func TestGetDetectionScriptWithoutCallback(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Query().Get("callback") != "" {
			t.Error("Expected no callback parameter")
		}
		w.Write([]byte(`console.log("detection script");`))
	}))
	defer server.Close()

	client := NewClient(server.URL)
	env := client.Env()

	script, err := env.GetDetectionScript("")
	if err != nil {
		t.Fatalf("Unexpected error: %v", err)
	}
	if script == "" {
		t.Error("Expected script to be non-empty")
	}
}

func TestAPIErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		response := struct {
			Code    int    `json:"code"`
			Message string `json:"message"`
		}{
			Code:    400,
			Message: "Invalid parameters",
		}
		json.NewEncoder(w).Encode(response)
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(server.URL)
	_, err := client.GetSliderCaptcha(320, 160, 8)

	if err == nil {
		t.Fatal("Expected error for API error response")
	}
}

func TestNetworkErrorHandling(t *testing.T) {
	client := NewClient("http://localhost:99999")
	_, err := client.GetSliderCaptcha(320, 160, 8)

	if err == nil {
		t.Fatal("Expected error for network failure")
	}
}

func TestTrajectoryPoint(t *testing.T) {
	point := TrajectoryPoint{
		X: 100,
		Y: 200,
		T: 1234567890,
	}

	if point.X != 100 {
		t.Errorf("Expected X '100', got '%d'", point.X)
	}
	if point.Y != 200 {
		t.Errorf("Expected Y '200', got '%d'", point.Y)
	}
	if point.T != 1234567890 {
		t.Errorf("Expected T '1234567890', got '%d'", point.T)
	}
}

func TestSliderCaptchaResponse(t *testing.T) {
	response := SliderCaptchaResponse{
		SessionID:    "session-123",
		ImageURL:     "http://example.com/image.png",
		PuzzleURL:    "http://example.com/puzzle.png",
		HintURL:      "http://example.com/hint.png",
		Shape:        1,
		SecretY:      80,
		ImageWidth:   320,
		ImageHeight:  160,
	}

	if response.SessionID != "session-123" {
		t.Errorf("Expected session ID 'session-123', got '%s'", response.SessionID)
	}
	if response.SecretY != 80 {
		t.Errorf("Expected secret Y '80', got '%d'", response.SecretY)
	}
	if response.ImageWidth != 320 {
		t.Errorf("Expected image width '320', got '%d'", response.ImageWidth)
	}
}

func TestVerifyCaptchaRequest(t *testing.T) {
	now := time.Now().UnixMilli()
	req := VerifyCaptchaRequest{
		SessionID: "session-123",
		X:         150,
		Y:         80,
		Trajectory: []TrajectoryPoint{
			{X: 0, Y: 80, T: now - 1000},
			{X: 150, Y: 80, T: now},
		},
	}

	if req.SessionID != "session-123" {
		t.Errorf("Expected session ID 'session-123', got '%s'", req.SessionID)
	}
	if req.X != 150 {
		t.Errorf("Expected X '150', got '%d'", req.X)
	}
	if len(req.Trajectory) != 2 {
		t.Errorf("Expected 2 trajectory points, got %d", len(req.Trajectory))
	}
}

func TestTrajectoryResult(t *testing.T) {
	result := TrajectoryResult{
		Score:   0.95,
		Passed:  true,
		Reasons: []string{"smooth trajectory", "reasonable speed"},
	}

	if result.Score != 0.95 {
		t.Errorf("Expected score '0.95', got '%f'", result.Score)
	}
	if !result.Passed {
		t.Error("Expected passed to be true")
	}
	if len(result.Reasons) != 2 {
		t.Errorf("Expected 2 reasons, got %d", len(result.Reasons))
	}
}

func TestLoginRequest(t *testing.T) {
	req := LoginRequest{
		Username:     "testuser",
		Password:     "password123",
		CaptchaToken: "token-xyz",
	}

	if req.Username != "testuser" {
		t.Errorf("Expected username 'testuser', got '%s'", req.Username)
	}
	if req.Password != "password123" {
		t.Errorf("Expected password 'password123', got '%s'", req.Password)
	}
	if req.CaptchaToken != "token-xyz" {
		t.Errorf("Expected captcha token 'token-xyz', got '%s'", req.CaptchaToken)
	}
}

func TestWithAPIKeyOption(t *testing.T) {
	apiKey := "test-api-key-123"
	client := NewClient("http://localhost:8080", WithAPIKey(apiKey))

	if client.apiKey != apiKey {
		t.Errorf("Expected API key '%s', got '%s'", apiKey, client.apiKey)
	}
}

func TestWithTimeoutOption(t *testing.T) {
	timeout := 60 * time.Second
	client := NewClient("http://localhost:8080", WithTimeout(timeout))

	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestWithHTTPClientOption(t *testing.T) {
	customClient := &http.Client{Timeout: 45 * time.Second}
	client := NewClient("http://localhost:8080", WithHTTPClient(customClient))

	if client.httpClient != customClient {
		t.Error("Custom HTTP client should be set")
	}
}

func TestMultipleOptions(t *testing.T) {
	customClient := &http.Client{Timeout: 45 * time.Second}
	apiKey := "test-api-key"
	timeout := 60 * time.Second

	client := NewClient(
		"http://localhost:8080",
		WithHTTPClient(customClient),
		WithAPIKey(apiKey),
		WithTimeout(timeout),
	)

	if client.httpClient != customClient {
		t.Error("Custom HTTP client should be set")
	}
	if client.apiKey != apiKey {
		t.Errorf("Expected API key '%s', got '%s'", apiKey, client.apiKey)
	}
	if client.httpClient.Timeout != timeout {
		t.Errorf("Expected timeout %v, got %v", timeout, client.httpClient.Timeout)
	}
}

func TestClientOptions(t *testing.T) {
	options := []Option{
		WithAPIKey("key1"),
		WithTimeout(30 * time.Second),
	}

	if len(options) != 2 {
		t.Errorf("Expected 2 options, got %d", len(options))
	}
}
