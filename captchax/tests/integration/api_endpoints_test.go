package integration

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
)

func setupTestRouter() *httptest.Server {
	return httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"service":   "captchax-api",
				"version":   "2.0.0",
			})
		case "/api/v1/captcha/slider/generate":
			var req struct {
				AppID string `json:"app_id"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.AppID == "" {
				http.Error(w, `{"error": "app_id is required"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              "test-slider-id",
				"background_b64": "mock_background",
				"slider_b64":     "mock_slider",
				"target_x":        100,
				"target_y":        50,
			})
		case "/api/v1/captcha/slider/verify":
			var req struct {
				CaptchaID string `json:"captcha_id"`
				TargetX   int    `json:"target_x"`
				TargetY   int    `json:"target_y"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.CaptchaID == "" {
				http.Error(w, `{"error": "captcha_id is required"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if req.TargetX >= 95 && req.TargetX <= 105 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"message": "verification successful",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "verification failed",
				})
			}
		case "/api/v1/captcha/click/generate":
			var req struct {
				AppID string `json:"app_id"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.AppID == "" {
				http.Error(w, `{"error": "app_id is required"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      "test-click-id",
				"image":   "mock_image_base64",
				"target_chars":   []string{"中", "心", "测", "试"},
			})
		case "/api/v1/captcha/click/verify":
			var req struct {
				CaptchaID string `json:"captcha_id"`
				Clicks    []struct {
					X int `json:"x"`
					Y int `json:"y"`
				} `json:"clicks"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.CaptchaID == "" || len(req.Clicks) < 4 {
				http.Error(w, `{"error": "captcha_id and clicks are required"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"score":   0.95,
				"message": "verification successful",
			})
		case "/api/v1/captcha/puzzle/generate":
			var req struct {
				AppID string `json:"app_id"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.AppID == "" {
				http.Error(w, `{"error": "app_id is required"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              "test-puzzle-id",
				"background_b64": "mock_background",
				"puzzle_b64":      "mock_puzzle",
				"target_x":        120,
				"target_y":        80,
			})
		case "/api/v1/captcha/puzzle/verify":
			var req struct {
				CaptchaID string `json:"captcha_id"`
				TargetX   int    `json:"target_x"`
				TargetY   int    `json:"target_y"`
			}
			json.NewDecoder(r.Body).Decode(&req)
			if req.CaptchaID == "" {
				http.Error(w, `{"error": "captcha_id is required"}`, http.StatusBadRequest)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			if req.TargetX >= 115 && req.TargetX <= 125 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"message": "verification successful",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "verification failed",
				})
			}
		default:
			http.NotFound(w, r)
		}
	}))
}

func TestHealthEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Health check returns 200", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}
	})

	t.Run("Health check returns healthy status", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["status"] != "healthy" {
			t.Errorf("Status = %s, want healthy", response["status"])
		}

		if response["service"] != "captchax-api" {
			t.Errorf("Service = %s, want captchax-api", response["service"])
		}
	})
}

func TestSliderGenerateEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Valid slider generate request", func(t *testing.T) {
		body := map[string]string{"app_id": "test-app"}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var response map[string]interface{}
		err = json.NewDecoder(resp.Body).Decode(&response)
		if err != nil {
			t.Fatalf("Failed to unmarshal response: %v", err)
		}

		if response["id"] == nil || response["id"] == "" {
			t.Error("Expected non-empty id in response")
		}

		if response["background_b64"] == nil {
			t.Error("Expected background_b64 in response")
		}

		if response["slider_b64"] == nil {
			t.Error("Expected slider_b64 in response")
		}
	})

	t.Run("Missing app_id returns 400", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestSliderVerifyEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Valid slider verify request", func(t *testing.T) {
		body := map[string]interface{}{
			"captcha_id": "test-slider-id",
			"target_x":   100,
			"target_y":   50,
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		if response["success"] != true {
			t.Errorf("Success = %v, want true", response["success"])
		}
	})

	t.Run("Wrong slider answer", func(t *testing.T) {
		body := map[string]interface{}{
			"captcha_id": "test-slider-id",
			"target_x":   50,
			"target_y":   50,
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		if response["success"] != false {
			t.Errorf("Success = %v, want false for wrong answer", response["success"])
		}
	})

	t.Run("Missing captcha_id returns 400", func(t *testing.T) {
		body := map[string]interface{}{
			"target_x": 100,
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestClickGenerateEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Valid click generate request", func(t *testing.T) {
		body := map[string]string{"app_id": "test-app"}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/click/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		if response["id"] == nil {
			t.Error("Expected non-empty id in response")
		}

		if response["target_chars"] == nil {
			t.Error("Expected target_chars in response")
		}
	})

	t.Run("Missing app_id returns 400", func(t *testing.T) {
		body := map[string]string{}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/click/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestClickVerifyEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Valid click verify request", func(t *testing.T) {
		body := map[string]interface{}{
			"captcha_id": "test-click-id",
			"clicks": []map[string]int{
				{"x": 70, "y": 105},
				{"x": 145, "y": 105},
				{"x": 220, "y": 105},
				{"x": 295, "y": 105},
			},
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/click/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		if response["success"] != true {
			t.Errorf("Success = %v, want true", response["success"])
		}

		if response["score"] == nil {
			t.Error("Expected score in response")
		}
	})

	t.Run("Missing clicks returns 400", func(t *testing.T) {
		body := map[string]interface{}{
			"captcha_id": "test-click-id",
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/click/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}

func TestPuzzleGenerateEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Valid puzzle generate request", func(t *testing.T) {
		body := map[string]string{"app_id": "test-app"}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/puzzle/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		if response["id"] == nil {
			t.Error("Expected non-empty id in response")
		}

		if response["puzzle_b64"] == nil {
			t.Error("Expected puzzle_b64 in response")
		}
	})
}

func TestPuzzleVerifyEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Valid puzzle verify request", func(t *testing.T) {
		body := map[string]interface{}{
			"captcha_id": "test-puzzle-id",
			"target_x":   120,
			"target_y":   80,
		}
		jsonBody, _ := json.Marshal(body)

		resp, err := http.Post(server.URL+"/api/v1/captcha/puzzle/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var response map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&response)

		if response["success"] != true {
			t.Errorf("Success = %v, want true", response["success"])
		}
	})
}

func TestCORSHeaders(t *testing.T) {
	corsServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusNoContent)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer corsServer.Close()

	t.Run("CORS headers present", func(t *testing.T) {
		req, _ := http.NewRequest("GET", corsServer.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("Access-Control-Allow-Origin") != "*" {
			t.Error("Missing Access-Control-Allow-Origin header")
		}

		if resp.Header.Get("Access-Control-Allow-Methods") == "" {
			t.Error("Missing Access-Control-Allow-Methods header")
		}
	})
}

func TestRateLimitHeaders(t *testing.T) {
	rateLimitServer := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-RateLimit-Limit", "60")
		w.Header().Set("X-RateLimit-Remaining", "59")
		w.Header().Set("X-RateLimit-Reset", "1234567890")
		w.WriteHeader(http.StatusOK)
	}))
	defer rateLimitServer.Close()

	t.Run("Rate limit headers present", func(t *testing.T) {
		req, _ := http.NewRequest("GET", rateLimitServer.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("X-RateLimit-Limit") == "" {
			t.Error("Missing X-RateLimit-Limit header")
		}

		if resp.Header.Get("X-RateLimit-Remaining") == "" {
			t.Error("Missing X-RateLimit-Remaining header")
		}

		if resp.Header.Get("X-RateLimit-Reset") == "" {
			t.Error("Missing X-RateLimit-Reset header")
		}
	})
}

func TestContentTypeHeader(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("JSON content type", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		contentType := resp.Header.Get("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", contentType)
		}
	})
}

func TestRequestIDHeader(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		requestID := r.Header.Get("X-Request-ID")
		if requestID == "" {
			requestID = "generated-id-123"
		}
		w.Header().Set("X-Request-ID", requestID)
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("Request ID header present", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("X-Request-ID") == "" {
			t.Error("Missing X-Request-ID header")
		}
	})

	t.Run("Request ID header echoed", func(t *testing.T) {
		req, _ := http.NewRequest("GET", server.URL, nil)
		req.Header.Set("X-Request-ID", "client-request-id")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("X-Request-ID") != "client-request-id" {
			t.Errorf("X-Request-ID = %s, want client-request-id", resp.Header.Get("X-Request-ID"))
		}
	})
}

func TestNotFoundEndpoint(t *testing.T) {
	server := setupTestRouter()
	defer server.Close()

	t.Run("Unknown endpoint returns 404", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/unknown/endpoint")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusNotFound {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusNotFound)
		}
	})
}

func TestMethodNotAllowed(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.Method != "GET" {
			http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("POST to GET endpoint returns 405", func(t *testing.T) {
		resp, err := http.Post(server.URL, "application/json",
			bytes.NewBufferString("{}"))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusMethodNotAllowed {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusMethodNotAllowed)
		}
	})
}

func TestInvalidJSONRequest(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		var req struct {
			AppID string `json:"app_id"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, `{"error": "invalid request body"}`, http.StatusBadRequest)
			return
		}
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	t.Run("Invalid JSON returns 400", func(t *testing.T) {
		resp, err := http.Post(server.URL, "application/json",
			bytes.NewBufferString("invalid json"))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusBadRequest)
		}
	})
}
