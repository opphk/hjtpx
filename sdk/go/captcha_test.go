package captcha

import (
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCaptchaClient(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	assert.NotNil(t, client)
	assert.Equal(t, "app-id", client.appID)
	assert.Equal(t, "app-secret", client.appSecret)
	assert.Equal(t, DefaultAPIEndpoint, client.baseURL)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
}

func TestNewCaptchaClientWithConfig(t *testing.T) {
	cfg := &Config{
		MaxIdleConns:     20,
		MaxOpenConns:    200,
		ConnMaxLifetime: 60 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
		HTTPTimeout:     60 * time.Second,
		DialTimeout:     20 * time.Second,
		ReadTimeout:     30 * time.Second,
		WriteTimeout:    30 * time.Second,
		MaxRetries:      5,
		RetryDelay:      200 * time.Millisecond,
		BaseURL:         "http://custom-endpoint.com",
		DebugMode:       true,
	}

	client := NewCaptchaClient("app-id", "app-secret", cfg)

	assert.Equal(t, 20, client.transport.MaxIdleConns)
	assert.Equal(t, 200, client.transport.MaxConnsPerHost)
	assert.Equal(t, 60*time.Second, client.httpClient.Timeout)
	assert.Equal(t, 5, client.config.MaxRetries)
	assert.Equal(t, 200*time.Millisecond, client.config.RetryDelay)
	assert.Equal(t, "http://custom-endpoint.com", client.baseURL)
	assert.True(t, client.config.DebugMode)
}

func TestCaptchaClient_Close(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	err := client.Close()
	assert.NoError(t, err)

	err = client.Close()
	assert.NoError(t, err)
}

func TestCaptchaClient_SetPoolConfig(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)

	newCfg := &Config{
		MaxIdleConns:   50,
		MaxOpenConns:   500,
		HTTPTimeout:    60 * time.Second,
		MaxRetries:     10,
		RetryDelay:     500 * time.Millisecond,
	}

	err := client.SetPoolConfig(newCfg)
	assert.NoError(t, err)

	assert.Equal(t, 50, client.transport.MaxIdleConns)
	assert.Equal(t, 500, client.transport.MaxConnsPerHost)
	assert.Equal(t, 60*time.Second, client.httpClient.Timeout)
	assert.Equal(t, 10, client.config.MaxRetries)
	assert.Equal(t, 500*time.Millisecond, client.config.RetryDelay)
}

func TestCaptchaClient_SetPoolConfig_ClosedClient(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	client.Close()

	err := client.SetPoolConfig(&Config{MaxIdleConns: 100})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "client is closed")
}

func TestCaptchaClient_GenerateSliderCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, SliderCaptchaPath, r.URL.Path)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"slider-test",
				"background_image":"data:image/png;base64,abc",
				"slider_image":"data:image/png;base64,xyz",
				"slider_width":50,
				"slider_height":50
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	resp, err := client.GenerateSliderCaptcha()

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "slider-test", resp.ChallengeID)
	assert.Equal(t, 50, resp.SliderWidth)
	assert.Equal(t, 50, resp.SliderHeight)
}

func TestCaptchaClient_VerifySliderCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, VerifyCaptchaPath, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req VerifyCaptchaRequest
		json.Unmarshal(body, &req)

		assert.Equal(t, "slider-id", req.ChallengeID)
		assert.Equal(t, "slide", req.Action)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true,"score":0.95,"risk_level":"low"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	resp, err := client.VerifySliderCaptcha("slider-id", "120")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 0.95, resp.Score)
}

func TestCaptchaClient_VerifySliderCaptcha_InvalidParams(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)

	_, err := client.VerifySliderCaptcha("", "120")
	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 400, GetSDKErrorCode(err))

	_, err = client.VerifySliderCaptcha("slider-id", "")
	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 400, GetSDKErrorCode(err))
}

func TestCaptchaClient_GenerateClickCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, ClickCaptchaPath, r.URL.Path)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"click-test",
				"background_image":"data:image/png;base64,abc",
				"target_position":[100,100],
				"target_index":3,
				"icon_positions":[[50,50],[100,100],[150,150],[200,200]]
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	resp, err := client.GenerateClickCaptcha()

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "click-test", resp.ChallengeID)
	assert.Equal(t, 3, resp.TargetIndex)
	assert.Len(t, resp.IconPositions, 4)
}

func TestCaptchaClient_VerifyClickCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, VerifyCaptchaPath, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req VerifyCaptchaRequest
		json.Unmarshal(body, &req)

		assert.Equal(t, "click-id", req.ChallengeID)
		assert.Equal(t, "click", req.Action)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true,"score":0.90}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	clicks := []ClickData{
		{X: 100, Y: 100, Duration: 500},
		{X: 150, Y: 150, Duration: 300},
	}

	resp, err := client.VerifyClickCaptcha("click-id", clicks)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestCaptchaClient_VerifyClickCaptcha_InvalidParams(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)

	_, err := client.VerifyClickCaptcha("", []ClickData{{X: 100, Y: 100}})
	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 400, GetSDKErrorCode(err))

	_, err = client.VerifyClickCaptcha("click-id", []ClickData{})
	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 400, GetSDKErrorCode(err))
}

func TestCaptchaClient_GenerateImageCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, ImageCaptchaPath, r.URL.Path)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"image-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	resp, err := client.GenerateImageCaptcha(nil)

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "image-test", resp.ChallengeID)
}

func TestCaptchaClient_VerifyImageCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, ImageVerifyPath, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req VerifyImageCaptchaRequest
		json.Unmarshal(body, &req)

		assert.Equal(t, "image-id", req.ChallengeID)
		assert.Equal(t, "test1234", req.Answer)

		success := req.Answer == "test1234"
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(fmt.Sprintf(`{"success":%v}`, success)),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	resp, err := client.VerifyImageCaptcha("image-id", "test1234")

	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestCaptchaClient_VerifyImageCaptcha_InvalidParams(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)

	_, err := client.VerifyImageCaptcha("", "answer")
	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 400, GetSDKErrorCode(err))

	_, err = client.VerifyImageCaptcha("image-id", "")
	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 400, GetSDKErrorCode(err))
}

func TestSDKError_Error(t *testing.T) {
	err := &SDKError{Code: 400, Message: "bad request"}
	assert.Equal(t, "SDKError(code=400, message=bad request)", err.Error())

	wrappedErr := fmt.Errorf("wrapped: %w", err)
	assert.Contains(t, wrappedErr.Error(), "wrapped")
}

func TestSDKError_Unwrap(t *testing.T) {
	originalErr := errors.New("original error")
	sdkErr := &SDKError{Code: 500, Message: "server error", Err: originalErr}

	unwrapped := sdkErr.Unwrap()
	assert.Equal(t, originalErr, unwrapped)
}

func TestSDKError_UnwrapNil(t *testing.T) {
	sdkErr := &SDKError{Code: 500, Message: "server error"}
	assert.Nil(t, sdkErr.Unwrap())
}

func TestIsSDKError(t *testing.T) {
	sdkErr := &SDKError{Code: 400, Message: "test"}
	assert.True(t, IsSDKError(sdkErr))

	regularErr := errors.New("regular error")
	assert.False(t, IsSDKError(regularErr))

	wrappedErr := fmt.Errorf("wrapped: %w", sdkErr)
	assert.True(t, IsSDKError(wrappedErr))
}

func TestGetSDKErrorCode(t *testing.T) {
	sdkErr := &SDKError{Code: 404, Message: "not found"}
	assert.Equal(t, 404, GetSDKErrorCode(sdkErr))

	regularErr := errors.New("regular error")
	assert.Equal(t, 0, GetSDKErrorCode(regularErr))

	wrappedErr := fmt.Errorf("wrapped: %w", sdkErr)
	assert.Equal(t, 404, GetSDKErrorCode(wrappedErr))
}

func TestNewSDKError(t *testing.T) {
	err := NewSDKError(500, "internal error")
	assert.Equal(t, 500, err.Code)
	assert.Equal(t, "internal error", err.Message)
	assert.Nil(t, err.Err)
}

func TestWrapSDKError(t *testing.T) {
	originalErr := errors.New("original")
	wrapped := wrapSDKError(400, "bad request", originalErr)

	assert.Equal(t, 400, wrapped.Code)
	assert.Equal(t, "bad request", wrapped.Message)
	assert.Equal(t, originalErr, wrapped.Err)
}

func TestSDKError_Is(t *testing.T) {
	originalErr := errors.New("network error")
	sdkErr := &SDKError{Code: 500, Message: "server error", Err: originalErr}

	assert.True(t, sdkErr.Is(originalErr))
	assert.False(t, sdkErr.Is(errors.New("different error")))
}

func TestConfigSetDefaults(t *testing.T) {
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

func TestCaptchaClient_AutoRetry(t *testing.T) {
	attemptCount := 0
	failCount := 2
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		t.Logf("Attempt %d received", attemptCount)
		if attemptCount <= failCount {
			w.WriteHeader(http.StatusInternalServerError)
			return
		}

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"retry-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:    server.URL,
		MaxRetries: 5,
		RetryDelay: 10 * time.Millisecond,
	})
	defer client.Close()

	resp, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)

	assert.GreaterOrEqual(t, attemptCount, 2, "should have at least 2 attempts")

	stats := client.GetStats()
	assert.GreaterOrEqual(t, int(stats.TotalRequests), 1)
	assert.GreaterOrEqual(t, int(stats.SuccessfulRequests), 1)
}

func TestCaptchaClient_RetryOnTimeout(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 2 {
			time.Sleep(200 * time.Millisecond)
			return
		}

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"timeout-retry","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:     server.URL,
		HTTPTimeout: 100 * time.Millisecond,
		MaxRetries:  3,
		RetryDelay:  10 * time.Millisecond,
	})

	resp, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
}

func TestCaptchaClient_RateLimitRetry(t *testing.T) {
	attemptCount := 0
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount == 1 {
			w.WriteHeader(http.StatusTooManyRequests)
			w.Header().Set("Retry-After", "0")
			return
		}

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"rate-limit-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:    server.URL,
		MaxRetries: 5,
		RetryDelay: 10 * time.Millisecond,
	})

	resp, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 2, attemptCount)
}

func TestCaptchaClient_Unauthorized(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	_, err := client.GenerateImageCaptcha(nil)

	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 401, GetSDKErrorCode(err))
}

func TestCaptchaClient_ConnectionPool(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", &Config{
		MaxIdleConns: 10,
		MaxOpenConns: 50,
	})

	assert.Equal(t, 10, client.transport.MaxIdleConns)
	assert.Equal(t, 50, client.transport.MaxConnsPerHost)

	err := client.SetPoolConfig(&Config{MaxIdleConns: 20, MaxOpenConns: 100})
	assert.NoError(t, err)
	assert.Equal(t, 20, client.transport.MaxIdleConns)
	assert.Equal(t, 100, client.transport.MaxConnsPerHost)
}

func TestCaptchaClient_GetStats(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"stats-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})

	for i := 0; i < 5; i++ {
		_, err := client.GenerateImageCaptcha(nil)
		assert.NoError(t, err)
	}

	stats := client.GetStats()
	assert.Equal(t, int64(5), stats.TotalRequests)
	assert.Equal(t, int64(5), stats.SuccessfulRequests)
	assert.Equal(t, int64(0), stats.FailedRequests)
	assert.True(t, stats.SuccessRate > 0)
}

func TestCaptchaClient_GetStats_AfterErrors(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:    server.URL,
		MaxRetries: 0,
	})

	_, _ = client.GenerateImageCaptcha(nil)

	stats := client.GetStats()
	assert.Equal(t, int64(1), stats.TotalRequests)
	assert.Equal(t, int64(0), stats.SuccessfulRequests)
	assert.Equal(t, int64(1), stats.FailedRequests)
	assert.Equal(t, float64(0), stats.SuccessRate)
	assert.NotNil(t, stats.LastError)
}

func TestCalculateSuccessRate(t *testing.T) {
	assert.Equal(t, float64(0), calculateSuccessRate(0, 0))
	assert.Equal(t, float64(100), calculateSuccessRate(100, 100))
	assert.Equal(t, float64(50), calculateSuccessRate(100, 50))
	assert.Equal(t, float64(75), calculateSuccessRate(40, 30))
}

func TestCaptchaClient_DebugMode(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", &Config{DebugMode: true})
	assert.NotPanics(t, func() {
		client.debug("Test message: %s", "value")
	})
}

func TestCaptchaClient_BuildURL(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: "http://test.com"})

	url := client.buildURL("/api/v1/test")
	assert.Equal(t, "http://test.com/api/v1/test", url)
}

func TestCaptchaClient_ExtractBase64Image(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)

	tests := []struct {
		name      string
		dataURI   string
		expectErr bool
	}{
		{
			name:      "empty string",
			dataURI:   "",
			expectErr: true,
		},
		{
			name:      "invalid format",
			dataURI:   "invalid-data",
			expectErr: true,
		},
		{
			name:      "PNG format",
			dataURI:   "data:image/png;base64,SGVsbG8gV29ybGQ=",
			expectErr: false,
		},
		{
			name:      "JPEG format",
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

func TestCaptchaClient_Headers(t *testing.T) {
	var receivedAppID, receivedAppSecret string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAppID = r.Header.Get("X-App-ID")
		receivedAppSecret = r.Header.Get("X-App-Secret")

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"header-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("my-app-id", "my-app-secret", &Config{BaseURL: server.URL})
	_, err := client.GenerateImageCaptcha(nil)

	assert.NoError(t, err)
	assert.Equal(t, "my-app-id", receivedAppID)
	assert.Equal(t, "my-app-secret", receivedAppSecret)
}

func TestCaptchaClient_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    500,
			Message: "Internal server error",
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	_, err := client.GenerateImageCaptcha(nil)

	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
	assert.Equal(t, 500, GetSDKErrorCode(err))
}

func TestCaptchaClient_InvalidJSONResponse(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Content-Type", "application/json")
		w.Write([]byte("invalid json{"))
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	_, err := client.GenerateImageCaptcha(nil)

	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
}

func TestCaptchaClient_NetworkError(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: "http://localhost:99999"})
	_, err := client.GenerateImageCaptcha(nil)

	assert.Error(t, err)
	assert.True(t, IsSDKError(err))
}

func TestNewClient(t *testing.T) {
	client := NewClient()
	assert.NotNil(t, client)
	assert.Equal(t, DefaultAPIEndpoint, client.endpoint)
	assert.Equal(t, 30*time.Second, client.timeout)
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
	assert.Equal(t, "http://example.com", client.endpoint)
	assert.Equal(t, 60*time.Second, client.timeout)
	assert.True(t, client.debugMode)
}

func TestClientSetters(t *testing.T) {
	client := NewClient()

	client.SetEndpoint("http://new-endpoint.com")
	assert.Equal(t, "http://new-endpoint.com", client.endpoint)

	client.SetDebugMode(true)
	assert.True(t, client.debugMode)

	customClient := &http.Client{Timeout: 10 * time.Second}
	client.SetHTTPClient(customClient)
	assert.Equal(t, 10*time.Second, client.httpClient.Timeout)
}

func TestBuildURL(t *testing.T) {
	client := NewClient(WithEndpoint("http://test.com"))

	tests := []struct {
		path     string
		expected string
	}{
		{ImageCaptchaPath, "http://test.com" + ImageCaptchaPath},
		{ImageVerifyPath, "http://test.com" + ImageVerifyPath},
		{"/custom/path", "http://test.com/custom/path"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			result := client.buildURL(tt.path)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestDebug(t *testing.T) {
	client := NewClient(WithDebugMode(true))
	assert.NotPanics(t, func() {
		client.debug("Test message: %s", "value")
	})
}

func TestGenerateImageCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, ImageCaptchaPath, r.URL.Path)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"test-id","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	resp, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "test-id", resp.ChallengeID)
	assert.Contains(t, resp.Image, "data:image/png;base64,")
}

func TestGenerateImageCaptchaWithParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, ImageCaptchaPath, r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "type=mixed")
		assert.Contains(t, r.URL.RawQuery, "count=6")

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"param-test-id","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	req := &ImageCaptchaRequest{
		Type:  CaptchaTypeMixed,
		Count: 6,
	}
	resp, err := client.GenerateImageCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "param-test-id", resp.ChallengeID)
}

func TestVerifyImageCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, ImageVerifyPath, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req VerifyImageCaptchaRequest
		json.Unmarshal(body, &req)
		assert.Equal(t, "test-challenge", req.ChallengeID)
		assert.Equal(t, "test1234", req.Answer)

		success := req.Answer == "test1234"
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(fmt.Sprintf(`{"success":%v}`, success)),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	req := &VerifyImageCaptchaRequest{
		ChallengeID: "test-challenge",
		Answer:     "test1234",
	}
	resp, err := client.VerifyImageCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
}

func TestVerifyImageCaptchaFailure(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":false}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	req := &VerifyImageCaptchaRequest{
		ChallengeID: "test-challenge",
		Answer:     "wrong-answer",
	}
	resp, err := client.VerifyImageCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.False(t, resp.Success)
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

func TestGetSliderCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, SliderCaptchaPath, r.URL.Path)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"slider-test",
				"background_image":"data:image/png;base64,abc",
				"slider_image":"data:image/png;base64,xyz",
				"slider_width":50,
				"slider_height":50
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	resp, err := client.GetSliderCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "slider-test", resp.ChallengeID)
	assert.Equal(t, 50, resp.SliderWidth)
	assert.Equal(t, 50, resp.SliderHeight)
}

func TestGetClickCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, ClickCaptchaPath, r.URL.Path)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"click-test",
				"background_image":"data:image/png;base64,abc",
				"target_position":[100,100],
				"target_index":3,
				"icon_positions":[[50,50],[100,100],[150,150],[200,200]]
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	resp, err := client.GetClickCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, "click-test", resp.ChallengeID)
	assert.Equal(t, 3, resp.TargetIndex)
	assert.Len(t, resp.IconPositions, 4)
}

func TestVerifyCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "POST", r.Method)
		assert.Equal(t, VerifyCaptchaPath, r.URL.Path)

		body, _ := io.ReadAll(r.Body)
		var req VerifyCaptchaRequest
		json.Unmarshal(body, &req)
		assert.Equal(t, "verify-challenge", req.ChallengeID)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"success":true,
				"score":0.85,
				"message":"Verification passed",
				"risk_level":"low"
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	req := &VerifyCaptchaRequest{
		ChallengeID: "verify-challenge",
		Action:     "click",
		Data:       map[string]interface{}{"x": 100, "y": 200},
	}
	resp, err := client.VerifyCaptcha(req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.True(t, resp.Success)
	assert.Equal(t, 0.85, resp.Score)
	assert.Equal(t, "low", resp.RiskLevel)
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

func TestDoRequestError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	_, err := client.doRequest("GET", "/test", nil)
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unexpected status code")
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
	assert.Contains(t, err.Error(), "API error")
	assert.Contains(t, err.Error(), "400")
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

func TestMockServer(t *testing.T) {
	t.Skip("Mock server test skipped due to network restrictions in sandbox environment")

	mock := NewMockServer(18080)
	err := mock.Start()
	if err != nil {
		t.Skipf("Cannot start mock server: %v", err)
	}
	defer mock.Stop()

	time.Sleep(100 * time.Millisecond)

	client := NewClient(
		WithEndpoint("http://localhost:18080"),
		WithDebugMode(true),
	)

	resp, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, mock.ChallengeID, resp.ChallengeID)

	mock.SetCorrectAnswer("test-answer")

	verifyResp, err := client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{
		ChallengeID: "test-id",
		Answer:     "wrong",
	})
	assert.NoError(t, err)
	assert.False(t, verifyResp.Success)

	verifyResp2, err := client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{
		ChallengeID: "test-id",
		Answer:     "test-answer",
	})
	assert.NoError(t, err)
	assert.True(t, verifyResp2.Success)
	assert.Equal(t, 1, mock.VerifyCalls)
}

func TestClientWithHeaders(t *testing.T) {
	var receivedAPIKey, receivedAPISecret string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		receivedAPIKey = r.Header.Get("X-API-Key")
		receivedAPISecret = r.Header.Get("X-API-Secret")

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"header-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithEndpoint(server.URL),
		WithAPIKey("my-api-key"),
		WithAPISecret("my-api-secret"),
	)

	_, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.Equal(t, "my-api-key", receivedAPIKey)
	assert.Equal(t, "my-api-secret", receivedAPISecret)
}

func TestRequestBodySerialization(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		assert.True(t, json.Valid(body))
		assert.Contains(t, string(body), "challenge_id")

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))

	_, err := client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{
		ChallengeID: "test-id",
		Answer:     "test-answer",
	})
	assert.NoError(t, err)
}

func TestSliderRequestParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "width=300")
		assert.Contains(t, r.URL.RawQuery, "height=200")

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"slider","background_image":"","slider_image":"","slider_width":50,"slider_height":50}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	_, err := client.GetSliderCaptcha(&SliderCaptchaRequest{
		Width:  300,
		Height: 200,
	})
	assert.NoError(t, err)
}

func TestClickRequestParams(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Contains(t, r.URL.RawQuery, "width=400")
		assert.Contains(t, r.URL.RawQuery, "height=300")
		assert.Contains(t, r.URL.RawQuery, "icon_count=9")

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"click","background_image":"","target_position":[1,2],"icon_positions":[]}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))
	_, err := client.GetClickCaptcha(&ClickCaptchaRequest{
		Width:     400,
		Height:    300,
		IconCount: 9,
	})
	assert.NoError(t, err)
}

func TestFullCaptchaWorkflow(t *testing.T) {
	var challengeID, verifiedAnswer string

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path

		if path == ImageCaptchaPath && r.Method == "GET" {
			challengeID = "workflow-test-challenge"
			resp := SDKResponse{
				Code:    0,
				Message: "success",
				Data:    json.RawMessage(fmt.Sprintf(`{"challenge_id":"%s","image":"data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}`, challengeID)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		if path == ImageVerifyPath && r.Method == "POST" {
			body, _ := io.ReadAll(r.Body)
			var req VerifyImageCaptchaRequest
			json.Unmarshal(body, &req)
			verifiedAnswer = req.Answer

			success := req.Answer == "workflow"
			resp := SDKResponse{
				Code:    0,
				Message: "success",
				Data:    json.RawMessage(fmt.Sprintf(`{"success":%v}`, success)),
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(resp)
			return
		}

		http.NotFound(w, r)
	}))
	defer server.Close()

	client := NewClient(WithEndpoint(server.URL))

	captchaResp, err := client.GenerateImageCaptcha(&ImageCaptchaRequest{
		Type:  CaptchaTypeMixed,
		Count: 4,
	})
	assert.NoError(t, err)
	assert.NotNil(t, captchaResp)
	assert.Equal(t, challengeID, captchaResp.ChallengeID)
	assert.True(t, strings.HasPrefix(captchaResp.Image, "data:image/png;base64,"))

	verifyResp, err := client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{
		ChallengeID: challengeID,
		Answer:     "workflow",
	})
	assert.NoError(t, err)
	assert.True(t, verifyResp.Success)
	assert.Equal(t, "workflow", verifiedAnswer)

	incorrectVerify, err := client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{
		ChallengeID: challengeID,
		Answer:     "wrong",
	})
	assert.NoError(t, err)
	assert.False(t, incorrectVerify.Success)
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

func TestClientCopy(t *testing.T) {
	client1 := NewClient(
		WithAPIKey("key1"),
		WithAPISecret("secret1"),
		WithEndpoint("http://endpoint1.com"),
	)

	client2 := NewClient(
		WithAPIKey("key2"),
	)

	assert.Equal(t, "key1", client1.apiKey)
	assert.Equal(t, "secret1", client1.apiSecret)
	assert.Equal(t, "http://endpoint1.com", client1.endpoint)

	assert.Equal(t, "key2", client2.apiKey)
	assert.Equal(t, "http://localhost:8080", client2.endpoint)
}

func TestDefaultValues(t *testing.T) {
	client := NewClient()

	assert.Equal(t, DefaultAPIEndpoint, client.endpoint)
	assert.NotNil(t, client.httpClient)
	assert.Equal(t, 30*time.Second, client.httpClient.Timeout)
	assert.Equal(t, 30*time.Second, client.timeout)
	assert.False(t, client.debugMode)
}
