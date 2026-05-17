package sdk

import (
	"encoding/json"
	"io"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewCaptchaClient(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	assert.NotNil(t, client)
	assert.NotNil(t, client.Client)
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
	assert.NotNil(t, client)
	assert.Equal(t, "http://custom-endpoint.com", client.baseURL)
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
	slider, err := client.GenerateSliderCaptcha()

	assert.NoError(t, err)
	assert.NotNil(t, slider)
	assert.Equal(t, "slider-test", slider.ChallengeID)
	assert.Equal(t, 50, slider.SliderWidth)
	assert.Equal(t, 50, slider.SliderHeight)
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
	click, err := client.GenerateClickCaptcha()

	assert.NoError(t, err)
	assert.NotNil(t, click)
	assert.Equal(t, "click-test", click.ChallengeID)
	assert.Equal(t, 3, click.TargetIndex)
}

func TestCaptchaClient_GenerateImageCaptchaWithOptions(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "GET", r.Method)
		assert.Equal(t, ImageCaptchaPath, r.URL.Path)
		assert.Contains(t, r.URL.RawQuery, "type=mixed")
		assert.Contains(t, r.URL.RawQuery, "count=6")

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
	image, err := client.GenerateImageCaptchaWithOptions(CaptchaTypeMixed, 6)

	assert.NoError(t, err)
	assert.NotNil(t, image)
	assert.Equal(t, "image-test", image.ChallengeID)
}

func TestCaptchaClient_VerifyImageCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req VerifyImageCaptchaRequest
		json.Unmarshal(body, &req)
		assert.Equal(t, "test-challenge", req.ChallengeID)
		assert.Equal(t, "test1234", req.Answer)

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	result, err := client.VerifyImageCaptcha("test-challenge", "test1234")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestCaptchaClient_VerifySliderCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		assert.Equal(t, "test-slider", req["challenge_id"])

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true,"score":0.95}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	result, err := client.VerifySliderCaptcha("test-slider", "150")

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
	assert.Equal(t, 0.95, result.Score)
}

func TestCaptchaClient_VerifyClickCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		body, _ := io.ReadAll(r.Body)
		var req map[string]interface{}
		json.Unmarshal(body, &req)
		assert.Equal(t, "test-click", req["challenge_id"])

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true,"score":0.85}`),
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
	result, err := client.VerifyClickCaptcha("test-click", clicks)

	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.True(t, result.Success)
}

func TestCaptchaClient_GetStats(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	stats := client.GetStats()
	assert.NotNil(t, stats)
}

func TestCaptchaClient_Close(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	err := client.Close()
	assert.NoError(t, err)
}

func TestCaptchaClient_SetPoolConfig(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	
	cfg := &Config{
		MaxIdleConns: 50,
		MaxOpenConns: 500,
		MaxRetries:   5,
	}
	client.SetPoolConfig(cfg)
}

func TestCalculateSuccessRate(t *testing.T) {
	tests := []struct {
		name     string
		total    int64
		success  int64
		expected float64
	}{
		{"Zero total", 0, 0, 0.0},
		{"50% success", 100, 50, 50.0},
		{"100% success", 100, 100, 100.0},
		{"0% success", 100, 0, 0.0},
		{"75% success", 1000, 750, 75.0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := calculateSuccessRate(tt.total, tt.success)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestVerifyImageCaptchaRequest(t *testing.T) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	
	_, err := client.Client.VerifyImageCaptcha(nil)
	assert.Error(t, err)
	
	_, err = client.Client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{})
	assert.Error(t, err)
	
	_, err = client.Client.VerifyImageCaptcha(&VerifyImageCaptchaRequest{ChallengeID: "test"})
	assert.Error(t, err)
}

func TestGenerateImageCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"test","image":"data:image/png;base64,abc"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	image, err := client.Client.GenerateImageCaptcha(nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, image)
}

func TestGetSliderCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"slider",
				"background_image":"",
				"slider_image":"",
				"slider_width":50,
				"slider_height":50
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	slider, err := client.Client.GetSliderCaptcha(nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, slider)
}

func TestGetClickCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"click",
				"background_image":"",
				"target_position":[1,2],
				"target_index":0,
				"icon_positions":[[1,2],[3,4]]
			}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	click, err := client.Client.GetClickCaptcha(nil)
	
	assert.NoError(t, err)
	assert.NotNil(t, click)
}

func TestVerifyCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true,"score":0.9}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	
	_, err := client.Client.VerifyCaptcha(nil)
	assert.Error(t, err)
	
	_, err = client.Client.VerifyCaptcha(&VerifyCaptchaRequest{})
	assert.Error(t, err)
}

func TestVerifySliderCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	
	_, err := client.Client.VerifySliderCaptcha("", "100")
	assert.Error(t, err)
	
	_, err = client.Client.VerifySliderCaptcha("test", "")
	assert.Error(t, err)
}

func TestVerifyClickCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	
	_, err := client.Client.VerifyClickCaptcha("", nil)
	assert.Error(t, err)
	
	_, err = client.Client.VerifyClickCaptcha("test", nil)
	assert.Error(t, err)
}

func TestGetGestureCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"gesture","pattern":"1→3→5","grid_size":3}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	gesture, err := client.Client.GetGestureCaptcha()
	
	assert.NoError(t, err)
	assert.NotNil(t, gesture)
	assert.Equal(t, "gesture", gesture.ChallengeID)
}

func TestVerifyGestureCaptcha(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	
	_, err := client.Client.VerifyGestureCaptcha(nil)
	assert.Error(t, err)
	
	_, err = client.Client.VerifyGestureCaptcha(&VerifyGestureRequest{})
	assert.Error(t, err)
	
	_, err = client.Client.VerifyGestureCaptcha(&VerifyGestureRequest{ChallengeID: "test"})
	assert.Error(t, err)
}

func TestParseSliderAnswer(t *testing.T) {
	result := parseSliderAnswer("150")
	assert.Equal(t, 150, result)
	
	result = parseSliderAnswer("not-a-number")
	assert.Equal(t, 0, result)
}
