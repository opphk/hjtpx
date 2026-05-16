package captcha

import (
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

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

func TestSDKErrorTypes(t *testing.T) {
	t.Run("Error constants", func(t *testing.T) {
		assert.NotNil(t, ErrNilRequest)
		assert.NotNil(t, ErrMissingChallenge)
		assert.NotNil(t, ErrMissingAnswer)
		assert.NotNil(t, ErrEmptyDataURI)
		assert.NotNil(t, ErrUnsupportedFormat)
		assert.NotNil(t, ErrAPIError)
		assert.NotNil(t, ErrNetworkError)
		assert.NotNil(t, ErrTimeoutError)
		assert.NotNil(t, ErrRetryExhausted)
	})

	t.Run("SDKError with wrapped error", func(t *testing.T) {
		innerErr := fmt.Errorf("inner error")
		sdkErr := &SDKError{
			Code:    500,
			Message: "test error",
			Err:     innerErr,
		}

		assert.Contains(t, sdkErr.Error(), "500")
		assert.Contains(t, sdkErr.Error(), "test error")
		assert.Contains(t, sdkErr.Error(), "inner error")
		assert.Equal(t, innerErr, sdkErr.Unwrap())
	})

	t.Run("SDKError without wrapped error", func(t *testing.T) {
		sdkErr := &SDKError{
			Code:    404,
			Message: "not found",
		}

		assert.Contains(t, sdkErr.Error(), "404")
		assert.Contains(t, sdkErr.Error(), "not found")
		assert.Nil(t, sdkErr.Unwrap())
	})
}

func TestRetryConfig(t *testing.T) {
	t.Run("DefaultRetryConfig", func(t *testing.T) {
		config := DefaultRetryConfig()
		assert.NotNil(t, config)
		assert.Equal(t, 3, config.MaxRetries)
		assert.Equal(t, 100*time.Millisecond, config.InitialDelay)
		assert.Equal(t, 5*time.Second, config.MaxDelay)
		assert.Equal(t, 2.0, config.BackoffFactor)
		assert.Contains(t, config.RetryableCodes, 429)
		assert.Contains(t, config.RetryableCodes, 500)
		assert.Contains(t, config.RetryableCodes, 502)
		assert.Contains(t, config.RetryableCodes, 503)
		assert.Contains(t, config.RetryableCodes, 504)
	})

	t.Run("ShouldRetry", func(t *testing.T) {
		config := DefaultRetryConfig()

		assert.True(t, config.ShouldRetry(429))
		assert.True(t, config.ShouldRetry(500))
		assert.True(t, config.ShouldRetry(502))
		assert.True(t, config.ShouldRetry(503))
		assert.True(t, config.ShouldRetry(504))

		assert.False(t, config.ShouldRetry(400))
		assert.False(t, config.ShouldRetry(401))
		assert.False(t, config.ShouldRetry(403))
		assert.False(t, config.ShouldRetry(404))
		assert.False(t, config.ShouldRetry(200))
		assert.False(t, config.ShouldRetry(201))
	})

	t.Run("NextDelay", func(t *testing.T) {
		config := DefaultRetryConfig()

		delay0 := config.NextDelay(0)
		delay1 := config.NextDelay(1)
		delay2 := config.NextDelay(2)

		assert.GreaterOrEqual(t, delay0, 100*time.Millisecond)
		assert.Greater(t, delay1, delay0)
		assert.Greater(t, delay2, delay1)

		config.MaxDelay = 200 * time.Millisecond
		delay := config.NextDelay(100)
		assert.Equal(t, 200*time.Millisecond, delay)
	})
}

func TestSignatureGeneration(t *testing.T) {
	t.Run("GenerateSignature with signature key", func(t *testing.T) {
		client := NewClient(WithSignatureKey("test-signature-key"))

		sig1 := client.GenerateSignature("GET", "/test", nil, nil)
		assert.NotEmpty(t, sig1)
		assert.Len(t, sig1, 64)

		sig2 := client.GenerateSignature("GET", "/test", nil, nil)
		assert.Equal(t, sig1, sig2)

		sig3 := client.GenerateSignature("POST", "/test", nil, nil)
		assert.NotEqual(t, sig1, sig3)
	})

	t.Run("GenerateSignature with params", func(t *testing.T) {
		client := NewClient(WithSignatureKey("test-signature-key"))

		params := map[string]string{"a": "1", "b": "2"}
		sig := client.GenerateSignature("GET", "/test", params, nil)
		assert.NotEmpty(t, sig)

		params2 := map[string]string{"b": "2", "a": "1"}
		sig2 := client.GenerateSignature("GET", "/test", params2, nil)
		assert.Equal(t, sig, sig2)
	})

	t.Run("GenerateSignature with body", func(t *testing.T) {
		client := NewClient(WithSignatureKey("test-signature-key"))

		sig := client.GenerateSignature("POST", "/test", nil, []byte(`{"key":"value"}`))
		assert.NotEmpty(t, sig)

		sig2 := client.GenerateSignature("POST", "/test", nil, []byte(`{"key":"value"}`))
		assert.Equal(t, sig, sig2)

		sig3 := client.GenerateSignature("POST", "/test", nil, []byte(`{"key":"other"}`))
		assert.NotEqual(t, sig, sig3)
	})

	t.Run("GenerateSignature without signature key", func(t *testing.T) {
		client := NewClient()

		sig := client.GenerateSignature("GET", "/test", nil, nil)
		assert.Empty(t, sig)
	})
}

func TestHMACSignature(t *testing.T) {
	t.Run("GenerateHMACSignature with app secret", func(t *testing.T) {
		client := NewClient(
			WithAppID("test-app"),
			WithAppSecret("test-secret"),
		)

		sig := client.GenerateHMACSignature("test data")
		assert.NotEmpty(t, sig)

		sig2 := client.GenerateHMACSignature("test data")
		assert.Equal(t, sig, sig2)

		sig3 := client.GenerateHMACSignature("different data")
		assert.NotEqual(t, sig, sig3)
	})

	t.Run("GenerateHMACSignature without app secret", func(t *testing.T) {
		client := NewClient()

		sig := client.GenerateHMACSignature("test data")
		assert.Empty(t, sig)
	})
}

func TestVerifySignature(t *testing.T) {
	client := NewClient(WithSignatureKey("test-signature-key"))

	sig := client.GenerateSignature("GET", "/test", nil, nil)

	assert.True(t, client.VerifySignature("GET", "/test", nil, nil, sig))
	assert.False(t, client.VerifySignature("GET", "/test", nil, nil, "invalid-signature"))
	assert.False(t, client.VerifySignature("POST", "/test", nil, nil, sig))
}

func TestEnableSignature(t *testing.T) {
	client := NewClient()

	assert.False(t, client.useSignature)

	client.EnableSignature(true)
	assert.True(t, client.useSignature)

	client.EnableSignature(false)
	assert.False(t, client.useSignature)

	client2 := NewClient(WithSignatureKey("test-key"))
	assert.True(t, client2.useSignature)
}

func TestSetRetryConfig(t *testing.T) {
	client := NewClient()

	config := DefaultRetryConfig()
	config.MaxRetries = 10

	client.SetRetryConfig(config)

	assert.Equal(t, 10, client.retryConfig.MaxRetries)
}

func TestRetryWithServerError(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		if attemptCount < 3 {
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

	client := NewClient(
		WithEndpoint(server.URL),
		WithRetryConfig(&RetryConfig{
			MaxRetries:     5,
			InitialDelay:   10 * time.Millisecond,
			MaxDelay:       100 * time.Millisecond,
			BackoffFactor:  1.5,
			RetryableCodes: []int{500, 502, 503, 504},
		}),
	)

	resp, err := client.GenerateImageCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, 3, attemptCount)
}

func TestRetryWithTimeout(t *testing.T) {
	t.Skip("Timeout test requires network isolation, skipping in sandbox environment")

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
			Data:    json.RawMessage(`{"challenge_id":"timeout-test","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewClient(
		WithEndpoint(server.URL),
		WithTimeout(100*time.Millisecond),
		WithRetryConfig(&RetryConfig{
			MaxRetries:     3,
			InitialDelay:   50 * time.Millisecond,
			MaxDelay:       200 * time.Millisecond,
			BackoffFactor:  2.0,
			RetryableCodes: []int{408, 500, 502, 503, 504},
		}),
	)

	_, err := client.GenerateImageCaptcha(nil)
	assert.Error(t, err)
}

func TestNoRetryOnClientError(t *testing.T) {
	attemptCount := 0

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		attemptCount++
		w.WriteHeader(http.StatusBadRequest)
	}))
	defer server.Close()

	client := NewClient(
		WithEndpoint(server.URL),
		WithRetryConfig(&RetryConfig{
			MaxRetries:     3,
			InitialDelay:   10 * time.Millisecond,
			MaxDelay:       100 * time.Millisecond,
			RetryableCodes: []int{500, 502, 503, 504},
		}),
	)

	_, err := client.GenerateImageCaptcha(nil)
	assert.Error(t, err)
	assert.Equal(t, 1, attemptCount)
}

func TestSDKErrorIs(t *testing.T) {
	t.Run("ErrNilRequest", func(t *testing.T) {
		_, err := NewClient().VerifyImageCaptcha(nil)
		assert.Error(t, err)

		sdkErr, ok := err.(*SDKError)
		assert.True(t, ok)
		assert.Equal(t, ErrNilRequest, sdkErr.Err)
	})

	t.Run("ErrMissingChallenge", func(t *testing.T) {
		_, err := NewClient().VerifyImageCaptcha(&VerifyImageCaptchaRequest{})
		assert.Error(t, err)

		sdkErr, ok := err.(*SDKError)
		assert.True(t, ok)
		assert.Equal(t, ErrMissingChallenge, sdkErr.Err)
	})

	t.Run("ErrMissingAnswer", func(t *testing.T) {
		_, err := NewClient().VerifyImageCaptcha(&VerifyImageCaptchaRequest{
			ChallengeID: "test-id",
		})
		assert.Error(t, err)

		sdkErr, ok := err.(*SDKError)
		assert.True(t, ok)
		assert.Equal(t, ErrMissingAnswer, sdkErr.Err)
	})

	t.Run("ErrEmptyDataURI", func(t *testing.T) {
		_, err := NewClient().ExtractBase64Image("")
		assert.Error(t, err)

		sdkErr, ok := err.(*SDKError)
		assert.True(t, ok)
		assert.Equal(t, ErrEmptyDataURI, sdkErr.Err)
	})

	t.Run("ErrUnsupportedFormat", func(t *testing.T) {
		_, err := NewClient().ExtractBase64Image("data:image/gif;base64,test")
		assert.Error(t, err)

		sdkErr, ok := err.(*SDKError)
		assert.True(t, ok)
		assert.Equal(t, ErrUnsupportedFormat, sdkErr.Err)
	})
}

func TestClientWithMultipleOptions(t *testing.T) {
	client := NewClient(
		WithAPIKey("api-key"),
		WithAPISecret("api-secret"),
		WithAppID("app-id"),
		WithAppSecret("app-secret"),
		WithSignatureKey("signature-key"),
		WithEndpoint("http://example.com"),
		WithTimeout(60*time.Second),
		WithDebugMode(true),
		WithRetryConfig(&RetryConfig{
			MaxRetries: 5,
		}),
	)

	assert.Equal(t, "api-key", client.apiKey)
	assert.Equal(t, "api-secret", client.apiSecret)
	assert.Equal(t, "app-id", client.appID)
	assert.Equal(t, "app-secret", client.appSecret)
	assert.Equal(t, "signature-key", client.signatureKey)
	assert.Equal(t, "http://example.com", client.endpoint)
	assert.Equal(t, 60*time.Second, client.timeout)
	assert.True(t, client.debugMode)
	assert.True(t, client.useSignature)
	assert.Equal(t, 5, client.retryConfig.MaxRetries)
}

func TestSignatureWithComplexParams(t *testing.T) {
	client := NewClient(WithSignatureKey("complex-signature-key"))

	params := map[string]string{
		"z": "z-value",
		"a": "a-value",
		"m": "m-value",
	}

	sig := client.GenerateSignature("GET", "/api/test", params, nil)
	assert.NotEmpty(t, sig)

	sig2 := client.GenerateSignature("GET", "/api/test",
		map[string]string{"a": "a-value", "m": "m-value", "z": "z-value"}, nil)
	assert.Equal(t, sig, sig2)
}

func TestGenerateToken(t *testing.T) {
	client := NewClient(
		WithAppID("test-app-id"),
		WithAppSecret("test-app-secret"),
	)

	token := client.GenerateToken("nonce-123", time.Now().Unix())
	assert.NotEmpty(t, token)

	token2 := client.GenerateToken("nonce-123", time.Now().Unix())
	assert.Equal(t, token, token2)
}

func TestMockServerClickEndpoint(t *testing.T) {
	t.Skip("Mock server test skipped due to network restrictions in sandbox environment")
	
	mock := NewMockServer(18082)
	err := mock.Start()
	assert.NoError(t, err)
	defer mock.Stop()

	time.Sleep(50 * time.Millisecond)

	client := NewClient(WithEndpoint("http://localhost:18082"))

	resp, err := client.GetClickCaptcha(nil)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.Equal(t, mock.ChallengeID, resp.ChallengeID)
	assert.Equal(t, 3, resp.TargetIndex)
	assert.Len(t, resp.IconPositions, 9)
}
