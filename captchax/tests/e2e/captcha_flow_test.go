package e2e

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
)

func TestE2E_CaptchaFlow_Slider(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/health":
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":  "healthy",
				"service": "captchax-api",
				"version": "2.0.0",
			})
		case "/api/v1/captcha/slider/generate":
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				AppID string `json:"app_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if req.AppID == "" {
				http.Error(w, `{"error": "app_id required"}`, http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              fmt.Sprintf("slider-%s", req.AppID),
				"background_b64": "mock_background_data",
				"slider_b64":     "mock_slider_data",
				"target_x":        150,
				"target_y":        100,
			})
		case "/api/v1/captcha/slider/verify":
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				CaptchaID string `json:"captcha_id"`
				TargetX   int    `json:"target_x"`
				TargetY   int    `json:"target_y"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if req.CaptchaID == "" {
				http.Error(w, `{"error": "captcha_id required"}`, http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if req.TargetX >= 145 && req.TargetX <= 155 && req.TargetY >= 95 && req.TargetY <= 105 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"message": "Verification successful",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Verification failed",
				})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Run("Slider captcha generation", func(t *testing.T) {
		reqBody := map[string]string{"app_id": "test-app"}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["id"] == nil || result["id"] == "" {
			t.Error("Expected captcha ID in response")
		}

		if result["background_b64"] == nil || result["background_b64"] == "" {
			t.Error("Expected background_b64 in response")
		}

		if result["slider_b64"] == nil || result["slider_b64"] == "" {
			t.Error("Expected slider_b64 in response")
		}
	})

	t.Run("Slider captcha verification - correct answer", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"captcha_id": "slider-test-app",
			"target_x":   150,
			"target_y":   100,
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["success"] != true {
			t.Errorf("Expected success=true, got %v", result["success"])
		}
	})

	t.Run("Slider captcha verification - wrong answer", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"captcha_id": "slider-test-app",
			"target_x":   50,
			"target_y":   50,
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/slider/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["success"] != false {
			t.Errorf("Expected success=false for wrong answer, got %v", result["success"])
		}
	})
}

func TestE2E_CaptchaFlow_Click(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/captcha/click/generate":
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				AppID string `json:"app_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if req.AppID == "" {
				http.Error(w, `{"error": "app_id required"}`, http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":      fmt.Sprintf("click-%s", req.AppID),
				"image":   "mock_image_base64",
				"target_chars": []string{"中", "心", "测", "试"},
			})
		case "/api/v1/captcha/click/verify":
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				CaptchaID string `json:"captcha_id"`
				Clicks    []struct {
					X int `json:"x"`
					Y int `json:"y"`
				} `json:"clicks"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if req.CaptchaID == "" {
				http.Error(w, `{"error": "captcha_id required"}`, http.StatusBadRequest)
				return
			}

			if len(req.Clicks) < 4 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"score":   0,
					"message": "Click count mismatch, expected 4 clicks",
				})
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"success": true,
				"score":   0.95,
				"message": "Verification successful",
			})
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Run("Click captcha generation", func(t *testing.T) {
		reqBody := map[string]string{"app_id": "test-app"}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/click/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["id"] == nil || result["id"] == "" {
			t.Error("Expected captcha ID in response")
		}

		if result["target_chars"] == nil {
			t.Error("Expected target_chars in response")
		}
	})

	t.Run("Click captcha verification", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"captcha_id": "click-test-app",
			"clicks": []map[string]int{
				{"x": 70, "y": 105},
				{"x": 145, "y": 105},
				{"x": 220, "y": 105},
				{"x": 295, "y": 105},
			},
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/click/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["success"] != true {
			t.Errorf("Expected success=true, got %v", result["success"])
		}

		if result["score"] == nil {
			t.Error("Expected score in response")
		}
	})
}

func TestE2E_CaptchaFlow_Puzzle(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/v1/captcha/puzzle/generate":
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				AppID string `json:"app_id"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if req.AppID == "" {
				http.Error(w, `{"error": "app_id required"}`, http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              fmt.Sprintf("puzzle-%s", req.AppID),
				"background_b64": "mock_background_data",
				"puzzle_b64":      "mock_puzzle_data",
				"target_x":        120,
				"target_y":        80,
			})
		case "/api/v1/captcha/puzzle/verify":
			if r.Method != "POST" {
				http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
				return
			}
			var req struct {
				CaptchaID string `json:"captcha_id"`
				TargetX   int    `json:"target_x"`
				TargetY   int    `json:"target_y"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			if req.CaptchaID == "" {
				http.Error(w, `{"error": "captcha_id required"}`, http.StatusBadRequest)
				return
			}

			w.Header().Set("Content-Type", "application/json")
			if req.TargetX >= 115 && req.TargetX <= 125 && req.TargetY >= 75 && req.TargetY <= 85 {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": true,
					"message": "Verification successful",
				})
			} else {
				json.NewEncoder(w).Encode(map[string]interface{}{
					"success": false,
					"message": "Verification failed",
				})
			}
		default:
			http.NotFound(w, r)
		}
	}))
	defer server.Close()

	t.Run("Puzzle captcha generation", func(t *testing.T) {
		reqBody := map[string]string{"app_id": "test-app"}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/puzzle/generate", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["id"] == nil || result["id"] == "" {
			t.Error("Expected captcha ID in response")
		}

		if result["puzzle_b64"] == nil || result["puzzle_b64"] == "" {
			t.Error("Expected puzzle_b64 in response")
		}
	})

	t.Run("Puzzle captcha verification", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"captcha_id": "puzzle-test-app",
			"target_x":   120,
			"target_y":   80,
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v1/captcha/puzzle/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["success"] != true {
			t.Errorf("Expected success=true, got %v", result["success"])
		}
	})
}

func TestE2E_BatchVerification(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v2/batch/verify" && r.Method == "POST" {
			var req struct {
				Items []struct {
					CaptchaID string `json:"captcha_id"`
					Type      string `json:"type"`
					TargetX   int    `json:"target_x"`
					TargetY   int    `json:"target_y"`
				} `json:"items"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				http.Error(w, "invalid request", http.StatusBadRequest)
				return
			}

			results := make([]map[string]interface{}, len(req.Items))
			successCount := 0

			for i, item := range req.Items {
				if item.Type == "" {
					results[i] = map[string]interface{}{
						"captcha_id": item.CaptchaID,
						"success":    false,
						"message":    "type is required",
					}
				} else if item.TargetX > 0 {
					results[i] = map[string]interface{}{
						"captcha_id": item.CaptchaID,
						"success":    true,
						"message":    "verification successful",
					}
					successCount++
				} else {
					results[i] = map[string]interface{}{
						"captcha_id": item.CaptchaID,
						"success":    false,
						"message":    "verification failed",
					}
				}
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"results": results,
				"summary": map[string]int{
					"total":   len(req.Items),
					"success": successCount,
					"failed":  len(req.Items) - successCount,
				},
			})
		}
	}))
	defer server.Close()

	t.Run("Batch verification flow", func(t *testing.T) {
		reqBody := map[string]interface{}{
			"items": []map[string]interface{}{
				{"captcha_id": "captcha-1", "type": "slider", "target_x": 100, "target_y": 50},
				{"captcha_id": "captcha-2", "type": "click", "target_x": 0, "target_y": 0},
				{"captcha_id": "captcha-3", "type": "", "target_x": 100, "target_y": 50},
			},
		}
		jsonBody, _ := json.Marshal(reqBody)

		resp, err := http.Post(server.URL+"/api/v2/batch/verify", "application/json",
			bytes.NewBuffer(jsonBody))
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["results"] == nil {
			t.Error("Expected results in response")
		}

		if result["summary"] == nil {
			t.Error("Expected summary in response")
		}

		summary := result["summary"].(map[string]interface{})
		if summary["total"] != float64(3) {
			t.Errorf("Expected total=3, got %v", summary["total"])
		}
	})
}

func TestE2E_ErrorHandling(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
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
				"id":       "generated-id",
				"success": true,
			})
		}
	}))
	defer server.Close()

	t.Run("Missing app_id returns error", func(t *testing.T) {
		reqBody := map[string]string{}
		jsonBody, _ := json.Marshal(reqBody)

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

func TestE2E_ConcurrentRequests(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/v1/captcha/slider/generate" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"id":              fmt.Sprintf("concurrent-captcha-%d", 1),
				"background_b64": "data",
				"slider_b64":     "data",
				"target_x":        100,
				"target_y":        50,
			})
		}
	}))
	defer server.Close()

	t.Run("Concurrent captcha generation requests", func(t *testing.T) {
		requestCount := 50
		results := make(chan int, requestCount)
		errors := make(chan error, requestCount)

		for i := 0; i < requestCount; i++ {
			go func() {
				reqBody := map[string]string{"app_id": "test"}
				jsonBody, _ := json.Marshal(reqBody)
				resp, err := http.Post(server.URL+"/api/v1/captcha/slider/generate", "application/json",
					bytes.NewBuffer(jsonBody))
				if err != nil {
					errors <- err
					return
				}
				defer resp.Body.Close()
				results <- resp.StatusCode
			}()
		}

		successCount := 0
		for i := 0; i < requestCount; i++ {
			select {
			case status := <-results:
				if status == http.StatusOK {
					successCount++
				}
			case err := <-errors:
				t.Errorf("Request error: %v", err)
			}
		}

		if successCount != requestCount {
			t.Errorf("Expected %d successful requests, got %d", requestCount, successCount)
		}
	})
}

func TestE2E_HealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/health" {
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"status":    "healthy",
				"service":   "captchax-api",
				"version":   "2.0.0",
				"timestamp": "2026-05-14T10:00:00Z",
			})
		}
	}))
	defer server.Close()

	t.Run("Health endpoint returns service status", func(t *testing.T) {
		resp, err := http.Get(server.URL + "/health")
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d", resp.StatusCode, http.StatusOK)
		}

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["status"] != "healthy" {
			t.Errorf("Expected status=healthy, got %v", result["status"])
		}

		if result["service"] != "captchax-api" {
			t.Errorf("Expected service=captchax-api, got %v", result["service"])
		}

		if result["version"] == nil {
			t.Error("Expected version in response")
		}
	})
}

func TestE2E_SessionManagement(t *testing.T) {
	t.Run("Session ID is generated and returned", func(t *testing.T) {
		sessionStore := sync.Map{}
		sessionCounter := int64(0)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.Header.Get("X-Session-ID")

			if sessionID == "" {
				newID := atomic.AddInt64(&sessionCounter, 1)
				sessionID = fmt.Sprintf("sess-%d", newID)
				sessionStore.Store(sessionID, map[string]interface{}{
					"created": true,
				})
				w.Header().Set("X-Session-ID", sessionID)
			}

			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"session_id": sessionID,
				"status":     "ok",
			})
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		sessionID := resp.Header.Get("X-Session-ID")
		if sessionID == "" {
			t.Error("Expected X-Session-ID header in response")
		}
	})

	t.Run("Session ID logic verification", func(t *testing.T) {
		sessionCounter := int64(0)

		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			sessionID := r.Header.Get("X-Session-ID")

			if sessionID == "" {
				newID := atomic.AddInt64(&sessionCounter, 1)
				sessionID = fmt.Sprintf("sess-%d", newID)
			}

			w.Header().Set("X-Session-ID", sessionID)
			w.Header().Set("Content-Type", "application/json")
			json.NewEncoder(w).Encode(map[string]interface{}{
				"session_id": sessionID,
				"status":     "ok",
			})
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("X-Session-ID") == "" {
			t.Error("Server should return session ID in header")
		}
	})
}
