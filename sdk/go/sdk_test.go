package sdk

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
	assert.Equal(t, DefaultAPIEndpoint, client.baseURL)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	assert.False(t, client.debugMode)
}

func TestNewClientWithOptions(t *testing.T) {
	client := NewClient(
		WithAPIKey("test-api-key"),
		WithAPISecret("test-api-secret"),
		WithEndpoint("http://example.com"),
		WithTimeout(60*time.Second),
		WithDebugMode(true),
	)

	assert.NotNil(t, client)
	assert.Equal(t, "test-api-key", client.apiKey)
	assert.Equal(t, "test-api-secret", client.apiSecret)
	assert.Equal(t, "http://example.com", client.baseURL)
	assert.Equal(t, 60*time.Second, client.httpClient.Timeout)
	assert.True(t, client.debugMode)
}

func TestClientSetters(t *testing.T) {
	client := NewClient()

	client.SetEndpoint("http://new-endpoint.com")
	assert.Equal(t, "http://new-endpoint.com", client.baseURL)

	client.SetDebugMode(true)
	assert.True(t, client.debugMode)

	customClient := &http.Client{Timeout: 10 * time.Second}
	client.SetHTTPClient(customClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestBuildURL(t *testing.T) {
	client := NewClient(WithEndpoint("http://test.com"))

	assert.Equal(t, "http://test.com"+ImageCaptchaPath, client.buildURL(ImageCaptchaPath))
	assert.Equal(t, "http://test.com/custom/path", client.buildURL("/custom/path"))
	assert.Equal(t, "http://other.com/path", client.buildURL("http://other.com/path"))
}

func TestDebug(t *testing.T) {
	client := NewClient(WithDebugMode(true))
	assert.NotPanics(t, func() {
		client.debug("Test message: %s", "value")
	})
}

func TestVerifyImageCaptchaErrors(t *testing.T) {
	client := NewClient()

	_, err := client.VerifyImageCaptcha(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")

	_, err = client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenge_id is required")

	_, err = client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{ChallengeID: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "answer is required")
}

func TestDoRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	_, err := client.doRequest("GET", "/test", nil)
	assert.Error(t, err)
}

func TestDoRequestAPIError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    400,
			Message: "Bad request",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	_, err := client.doRequest("GET", "/test", nil)
	assert.Error(t, err)
}

func TestDoRequestJSONError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json{"))
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	_, err := client.doRequest("GET", "/test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to unmarshal response")
}

func TestExtractBase64Image(t *testing.T) {
	client := NewClient()

	tests := []struct {
		name      string
		dataURI   string
		expectErr bool
	}{
		{
			name:      "空字符串",
			dataURI:   "",
			expectErr: true,
		},
		{
			name:      "无效格式",
			dataURI:   "invalid-data",
			expectErr: true,
		},
		{
			name:      "PNG格式",
			dataURI:   "data:image/png;base64,SGVsbG8gV29ybGQ=",
			expectErr: false,
		},
		{
			name:      "JPEG格式",
			dataURI:   "data:image/jpeg;base64,SGVsbG8gV29ybGQ=",
			expectErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := client.ExtractBase64Image(tt.dataURI)
			if tt.expectErr {
				assert.Error(t, err)
			} else {
				assert.NoError(t, err)
			}
		})
	}
}

func TestDecodeResponseFunctions(t *testing.T) {
	t.Run("DecodeImageCaptchaResponse", func(t *testing.T) {
		data := json.RawMessage(`{"challenge_id":"test","image":"data:image/png;base64,test"}`)
		resp, err := DecodeImageCaptchaResponse(data)
		assert.NoError(t, err)
		assert.Equal(t, "test", resp.ChallengeID)
	})

	t.Run("DecodeVerifyResponse", func(t *testing.T) {
		data := json.RawMessage(`{"success":true}`)
		resp, err := DecodeVerifyResponse(data)
		assert.NoError(t, err)
		assert.True(t, resp.Success)
	})

	t.Run("DecodeSliderResponse", func(t *testing.T) {
		data := json.RawMessage(`{"challenge_id":"slider","background_image":"","slider_image":"","slider_width":50,"slider_height":50}`)
		resp, err := DecodeSliderResponse(data)
		assert.NoError(t, err)
		assert.Equal(t, "slider", resp.ChallengeID)
	})

	t.Run("DecodeClickResponse", func(t *testing.T) {
		data := json.RawMessage(`{"challenge_id":"click","background_image":"","target_position":[1,2]}`)
		resp, err := DecodeClickResponse(data)
		assert.NoError(t, err)
		assert.Equal(t, "click", resp.ChallengeID)
	})
}

func TestSDKError(t *testing.T) {
	t.Run("Error", func(t *testing.T) {
		err := &SDKError{Code: 400, Message: "bad request"}
		assert.Equal(t, "SDKError(code=400, message=bad request)", err.Error())
	})

	t.Run("Unwrap", func(t *testing.T) {
		originalErr := assert.AnError
		sdkErr := &SDKError{Code: 500, Message: "server error", Err: originalErr}
		assert.Equal(t, originalErr, sdkErr.Unwrap())
	})

	t.Run("Is", func(t *testing.T) {
		originalErr := assert.AnError
		sdkErr := &SDKError{Code: 500, Message: "server error", Err: originalErr}
		assert.True(t, sdkErr.Is(originalErr))
	})
}

func TestIsSDKError(t *testing.T) {
	sdkErr := &SDKError{Code: 400, Message: "test"}
	assert.True(t, IsSDKError(sdkErr))

	regularErr := assert.AnError
	assert.False(t, IsSDKError(regularErr))
}

func TestGetSDKErrorCode(t *testing.T) {
	sdkErr := &SDKError{Code: 404, Message: "not found"}
	assert.Equal(t, 404, GetSDKErrorCode(sdkErr))

	regularErr := assert.AnError
	assert.Equal(t, 0, GetSDKErrorCode(regularErr))
}

func TestAuthClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"access_token":"test-token",
				"refresh_token":"refresh-token",
				"expires_in":900,
				"user":{"id":1,"username":"testuser","email":"test@example.com"}
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	authClient := client.Auth()

	resp, err := authClient.Login(&LoginRequest{
		Username: "testuser",
		Password: "password",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-token", resp.AccessToken)
	assert.Equal(t, "refresh-token", resp.RefreshToken)
}

func TestAuthClientRegister(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"user_id":1,
				"username":"newuser",
				"email":"new@example.com",
				"verification_link":"/api/v1/auth/verify-email?token=xxx"
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	authClient := client.Auth()

	resp, err := authClient.Register(&RegisterRequest{
		Username: "newuser",
		Email:    "new@example.com",
		Password: "password123",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "newuser", resp.Username)
}

func TestDetectClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"success":true,
				"risk_score":15.5,
				"anomalies":5
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	detectClient := client.Detect()

	resp, err := detectClient.Submit(&DetectionSubmitRequest{
		DetectionID: "test-id",
		RiskScore:   10.0,
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 15.5, resp.RiskScore)
}

func TestEnvironmentCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"is_bot":false,
				"risk_level":"low",
				"risk_score":10.5,
				"detected_flags":[],
				"fingerprint":"test-fp",
				"is_unique":true
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	detectClient := client.Detect()

	resp, err := detectClient.Check(&EnvironmentCheckRequest{
		Fingerprint: "test-fingerprint",
	})

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.IsBot)
	assert.Equal(t, "low", resp.RiskLevel)
}

func TestNetworkError(t *testing.T) {
	client := NewClient(WithEndpoint("http://localhost:99999"))

	_, err := client.GenerateImageCaptcha(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "failed to send request")
}

func TestTimeout(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(200 * time.Millisecond)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	client := NewClient(
		WithEndpoint(server.URL),
		WithTimeout(50*time.Millisecond),
	)

	_, err := client.GenerateImageCaptcha(nil)
	assert.Error(t, err)
}

func TestParseQueryParams(t *testing.T) {
	params := map[string]interface{}{
		"string": "value",
		"int":    123,
		"bool":   true,
		"empty":  "",
		"zero":   0,
	}

	result := ParseQueryParams(params)
	assert.Contains(t, result, "string=value")
	assert.Contains(t, result, "int=123")
	assert.Contains(t, result, "bool=true")
	assert.NotContains(t, result, "empty=")
	assert.NotContains(t, result, "zero=")
}

func TestConfigDefaults(t *testing.T) {
	cfg := &Config{}
	cfg.setDefaults()

	assert.Equal(t, 10, cfg.MaxIdleConns)
	assert.Equal(t, 100, cfg.MaxOpenConns)
	assert.Equal(t, 30*time.Minute, cfg.ConnMaxLifetime)
	assert.Equal(t, 5*time.Minute, cfg.ConnMaxIdleTime)
	assert.Equal(t, 30*time.Second, cfg.HTTPTimeout)
	assert.Equal(t, 10*time.Second, cfg.DialTimeout)
	assert.Equal(t, 15*time.Second, cfg.ReadTimeout)
	assert.Equal(t, 15*time.Second, cfg.WriteTimeout)
	assert.Equal(t, 3, cfg.MaxRetries)
	assert.Equal(t, 100*time.Millisecond, cfg.RetryDelay)
	assert.Equal(t, DefaultAPIEndpoint, cfg.BaseURL)
}

func TestAdminClient(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"totalUsers":100,
				"totalApps":10,
				"totalRequests":10000,
				"totalErrors":500
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	adminClient := client.Admin("test-token")

	stats, err := adminClient.GetDashboardStats()
	assert.NoError(t, err)
	assert.NotNil(t, stats)
	assert.Equal(t, int64(100), stats.TotalUsers)
	assert.Equal(t, int64(10), stats.TotalApps)
}

func TestTokenManager(t *testing.T) {
	client := NewClient()
	tm := NewTokenManager(client)

	tm.SetTokens("access-token", "refresh-token", 900)

	assert.Equal(t, "access-token", tm.GetAccessToken())
	assert.False(t, tm.IsTokenExpired())
}

func TestNewSDKError(t *testing.T) {
	err := NewSDKError(500, "internal error")
	assert.Equal(t, 500, err.Code)
	assert.Equal(t, "internal error", err.Message)
	assert.Nil(t, err.Err)
}

func TestWrapSDKError(t *testing.T) {
	originalErr := assert.AnError
	wrapped := wrapSDKError(400, "bad request", originalErr)

	assert.Equal(t, 400, wrapped.Code)
	assert.Equal(t, "bad request", wrapped.Message)
	assert.Equal(t, originalErr, wrapped.Err)
}

func TestVerifyCaptchaErrors(t *testing.T) {
	client := NewClient()

	_, err := client.VerifyCaptcha(nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "request cannot be nil")

	_, err = client.VerifyCaptcha(&VerifyCaptchaRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenge_id is required")
}

func TestVerifySliderCaptchaErrors(t *testing.T) {
	client := NewClient()

	_, err := client.VerifySliderCaptcha("", "100")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenge_id is required")

	_, err = client.VerifySliderCaptcha("test", "")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "answer is required")
}

func TestVerifyClickCaptchaErrors(t *testing.T) {
	client := NewClient()

	_, err := client.VerifyClickCaptcha("", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenge_id is required")

	_, err = client.VerifyClickCaptcha("test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "clicks is required")
}

func TestGetGestureCaptchaResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"gesture-test",
				"pattern":"1→3→5",
				"grid_size":3
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	gesture, err := client.GetGestureCaptcha()

	assert.NoError(t, err)
	assert.NotNil(t, gesture)
	assert.Equal(t, "gesture-test", gesture.ChallengeID)
	assert.Equal(t, 3, gesture.GridSize)
}

func TestVerifyGestureCaptchaErrors(t *testing.T) {
	client := NewClient()

	_, err := client.VerifyGestureCaptcha(nil)
	assert.Error(t, err)

	_, err = client.VerifyGestureCaptcha(&VerifyGestureRequest{})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "challenge_id is required")

	_, err = client.VerifyGestureCaptcha(&VerifyGestureRequest{ChallengeID: "test"})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "pattern is required")
}
