package captcha

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"
)

func BenchmarkCaptchaClient_SingleRequest(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"benchmark","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GenerateImageCaptcha(nil)
	}

	b.StopTimer()
	client.Close()
}

func BenchmarkCaptchaClient_ParallelRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"benchmark","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	defer client.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.GenerateImageCaptcha(nil)
		}
	})
}

func BenchmarkCaptchaClient_WithConnectionPool(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"benchmark","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:      server.URL,
		MaxIdleConns: 100,
		MaxOpenConns: 200,
	})
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GenerateImageCaptcha(nil)
	}
}

func BenchmarkCaptchaClient_ParallelWithPool(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"benchmark","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:      server.URL,
		MaxIdleConns: 100,
		MaxOpenConns: 200,
	})
	defer client.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.GenerateImageCaptcha(nil)
		}
	})
}

func BenchmarkCaptchaClient_VerifyRequests(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.VerifyImageCaptcha("test-id", "answer")
	}
}

func BenchmarkCaptchaClient_SliderCaptcha(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"slider-benchmark",
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
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GenerateSliderCaptcha()
	}
}

func BenchmarkCaptchaClient_ClickCaptcha(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data: json.RawMessage(`{
				"challenge_id":"click-benchmark",
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
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.GenerateClickCaptcha()
	}
}

func BenchmarkCaptchaClient_VerifySlider(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"success":true,"score":0.90,"risk_level":"low"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.VerifySliderCaptcha("test-id", "120")
	}
}

func BenchmarkCaptchaClient_VerifyClick(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
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
	defer client.Close()

	clicks := []ClickData{
		{X: 100, Y: 100, Duration: 500},
		{X: 150, Y: 150, Duration: 300},
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.VerifyClickCaptcha("test-id", clicks)
	}
}

func BenchmarkCaptchaClient_HighConcurrency(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		time.Sleep(1 * time.Millisecond)
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"benchmark","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{
		BaseURL:      server.URL,
		MaxIdleConns: 100,
		MaxOpenConns: 500,
	})
	defer client.Close()

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, _ = client.GenerateImageCaptcha(nil)
		}
	})
}

func BenchmarkClientCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		client := NewCaptchaClient("app-id", "app-secret", nil)
		client.Close()
	}
}

func BenchmarkConfigDefaults(b *testing.B) {
	for i := 0; i < b.N; i++ {
		cfg := &Config{}
		cfg.setDefaults()
		_ = cfg
	}
}

func BenchmarkPoolStats(b *testing.B) {
	client := NewCaptchaClient("app-id", "app-secret", nil)

	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    json.RawMessage(`{"challenge_id":"benchmark","image":"data:image/png;base64,test"}`),
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client = NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})

	for i := 0; i < 100; i++ {
		_, _ = client.GenerateImageCaptcha(nil)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.GetStats()
	}

	client.Close()
}

func BenchmarkExtractBase64Image(b *testing.B) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	dataURI := "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = client.ExtractBase64Image(dataURI)
	}
}

func BenchmarkConcurrentClientCreation(b *testing.B) {
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			client := NewCaptchaClient("app-id", "app-secret", nil)
			client.Close()
		}
	})
}

func BenchmarkSDKErrorCreation(b *testing.B) {
	for i := 0; i < b.N; i++ {
		err := NewSDKError(500, "internal error")
		_ = err
	}
}

func BenchmarkIsSDKError(b *testing.B) {
	err := NewSDKError(400, "bad request")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = IsSDKError(err)
	}
}

func BenchmarkGetSDKErrorCode(b *testing.B) {
	err := NewSDKError(404, "not found")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = GetSDKErrorCode(err)
	}
}

func BenchmarkJSONMarshal(b *testing.B) {
	req := &VerifyImageCaptchaRequest{
		ChallengeID: "test-challenge-id",
		Answer:     "test-answer",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = json.Marshal(req)
	}
}

func BenchmarkJSONUnmarshal(b *testing.B) {
	data := []byte(`{"challenge_id":"test","image":"data:image/png;base64,test"}`)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		var resp ImageCaptchaResponse
		_ = json.Unmarshal(data, &resp)
	}
}

func BenchmarkBuildURL(b *testing.B) {
	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: "http://test.com"})

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.buildURL("/api/v1/test")
	}
}

func BenchmarkClientClose(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		client := NewCaptchaClient("app-id", "app-secret", nil)
		_ = client.Close()
	}
}

func BenchmarkSetPoolConfig(b *testing.B) {
	client := NewCaptchaClient("app-id", "app-secret", nil)
	cfg := &Config{
		MaxIdleConns: 50,
		MaxOpenConns: 500,
		MaxRetries:   5,
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = client.SetPoolConfig(cfg)
	}

	client.Close()
}

func BenchmarkSuccessRateCalculation(b *testing.B) {
	var total int64 = 1000
	var success int64 = 950

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_ = calculateSuccessRate(total, success)
	}
}

func BenchmarkMultipleCaptchaTypes(b *testing.B) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		path := r.URL.Path
		var data json.RawMessage

		switch path {
		case ImageCaptchaPath:
			data = json.RawMessage(`{"challenge_id":"image","image":"data:image/png;base64,test"}`)
		case SliderCaptchaPath:
			data = json.RawMessage(`{"challenge_id":"slider","background_image":"","slider_image":"","slider_width":50,"slider_height":50}`)
		case ClickCaptchaPath:
			data = json.RawMessage(`{"challenge_id":"click","background_image":"","target_position":[100,100],"target_index":3,"icon_positions":[[50,50],[100,100],[150,150],[200,200]]}`)
		default:
			data = json.RawMessage(`{"challenge_id":"default"}`)
		}

		resp := SDKResponse{
			Code:    0,
			Message: "success",
			Data:    data,
		}
		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(resp)
	}))
	defer server.Close()

	client := NewCaptchaClient("app-id", "app-secret", &Config{BaseURL: server.URL})
	defer client.Close()

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		switch i % 3 {
		case 0:
			_, _ = client.GenerateImageCaptcha(nil)
		case 1:
			_, _ = client.GenerateSliderCaptcha()
		case 2:
			_, _ = client.GenerateClickCaptcha()
		}
	}
}
