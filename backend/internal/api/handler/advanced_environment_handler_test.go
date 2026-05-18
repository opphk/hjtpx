package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestAdvancedEnvironmentHandler_DetectEnvironment(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	tests := []struct {
		name       string
		request    DetectEnvironmentRequest
		wantStatus int
		wantSuccess bool
	}{
		{
			name: "Valid Request",
			request: DetectEnvironmentRequest{
				DetectionID:   "test_123",
				RiskScore:     30.0,
				RiskLevel:     "low",
				AllDetections: []string{"normal_detection"},
				Fingerprint:   "fp_test",
				IPAddress:     "192.168.1.1",
				UserAgent:     "Mozilla/5.0",
			},
			wantStatus: http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "High Risk Request",
			request: DetectEnvironmentRequest{
				DetectionID:   "test_456",
				RiskScore:     85.0,
				RiskLevel:     "high",
				AllDetections: []string{"vm_detected", "headless"},
				Fingerprint:   "fp_high_risk",
				IPAddress:     "10.0.0.1",
				UserAgent:     "HeadlessChrome",
			},
			wantStatus: http.StatusOK,
			wantSuccess: true,
		},
		{
			name: "Request with Client Results",
			request: DetectEnvironmentRequest{
				DetectionID:   "test_789",
				RiskScore:     50.0,
				RiskLevel:     "medium",
				AllDetections: []string{"some_detection"},
				ClientResults: map[string]interface{}{
					"webgl_anomaly": map[string]interface{}{
						"score":      20.0,
						"detections": []interface{}{"low_extension_count"},
					},
					"vm_cpu": map[string]interface{}{
						"score":      35.0,
						"detections": []interface{}{"typical_vm_cpu:4"},
					},
				},
				Fingerprint: "fp_with_results",
				IPAddress:   "172.16.0.1",
				UserAgent:   "Mozilla/5.0",
			},
			wantStatus: http.StatusOK,
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			
			router.POST("/detect", handler.DetectEnvironment)
			
			body, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest(http.MethodPost, "/detect", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			
			success, ok := response["success"].(bool)
			if !ok {
				t.Fatal("response missing 'success' field")
			}
			
			if success != tt.wantSuccess {
				t.Errorf("success = %v, want %v", success, tt.wantSuccess)
			}
		})
	}
}

func TestAdvancedEnvironmentHandler_DetectEnvironment_InvalidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	tests := []struct {
		name       string
		body       string
		wantStatus int
	}{
		{
			name:       "Empty Body",
			body:       "",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "Invalid JSON",
			body:       "{invalid}",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.POST("/detect", handler.DetectEnvironment)
			
			req, _ := http.NewRequest(http.MethodPost, "/detect", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAdvancedEnvironmentHandler_CheckTorNetwork(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	tests := []struct {
		name       string
		queryIP    string
		wantStatus int
		wantSuccess bool
	}{
		{
			name:       "Check Tor with Private IP",
			queryIP:    "192.168.1.1",
			wantStatus: http.StatusOK,
			wantSuccess: true,
		},
		{
			name:       "Check Tor without IP (uses default)",
			queryIP:    "",
			wantStatus: http.StatusOK,
			wantSuccess: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/tor-check", handler.CheckTorNetwork)
			
			url := "/tor-check"
			if tt.queryIP != "" {
				url += "?ip=" + tt.queryIP
			}
			
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
			
			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("failed to unmarshal response: %v", err)
			}
			
			success, ok := response["success"].(bool)
			if !ok {
				t.Fatal("response missing 'success' field")
			}
			
			if success != tt.wantSuccess {
				t.Errorf("success = %v, want %v", success, tt.wantSuccess)
			}
		})
	}
}

func TestAdvancedEnvironmentHandler_GetEnvironmentStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	router := gin.New()
	router.GET("/stats", handler.GetEnvironmentStats)
	
	req, _ := http.NewRequest(http.MethodGet, "/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)
	
	if w.Code != http.StatusOK {
		t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
	}
	
	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}
	
	success, ok := response["success"].(bool)
	if !ok || !success {
		t.Error("expected success = true")
	}
	
	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatal("response missing 'data' field")
	}
	
	if data["total_detections"] == nil {
		t.Error("data missing 'total_detections' field")
	}
}

func TestAdvancedEnvironmentHandler_GetCachedDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	tests := []struct {
		name       string
		detectionID string
		wantStatus int
	}{
		{
			name:        "Non-existent ID",
			detectionID: "nonexistent_id_12345",
			wantStatus: http.StatusNotFound,
		},
		{
			name:        "Empty ID",
			detectionID: "",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/detection/:id", handler.GetCachedDetection)
			
			url := "/detection/" + tt.detectionID
			req, _ := http.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != tt.wantStatus {
				t.Errorf("status = %d, want %d", w.Code, tt.wantStatus)
			}
		})
	}
}

func TestAdvancedEnvironmentHandler_ExtractProxyHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	tests := []struct {
		name        string
		headers     map[string]string
		wantHeaders int
	}{
		{
			name:        "No Proxy Headers",
			headers:     map[string]string{},
			wantHeaders: 0,
		},
		{
			name: "With X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1, 10.0.0.1",
			},
			wantHeaders: 1,
		},
		{
			name: "With Multiple Headers",
			headers: map[string]string{
				"X-Forwarded-For": "192.168.1.1",
				"X-Real-IP":       "10.0.0.1",
				"Via":             "1.1 proxy.example.com",
			},
			wantHeaders: 3,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.GET("/test", func(c *gin.Context) {
				headers := handler.extractProxyHeaders(c)
				if len(headers) != tt.wantHeaders {
					t.Errorf("headers count = %d, want %d", len(headers), tt.wantHeaders)
				}
				c.JSON(http.StatusOK, gin.H{"count": len(headers)})
			})
			
			req, _ := http.NewRequest(http.MethodGet, "/test", nil)
			for k, v := range tt.headers {
				req.Header.Set(k, v)
			}
			
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
			
			if w.Code != http.StatusOK {
				t.Errorf("status = %d, want %d", w.Code, http.StatusOK)
			}
		})
	}
}

func TestConvertSummary(t *testing.T) {
	tests := []struct {
		name    string
		summary *DetectionSummary
		wantNil bool
	}{
		{
			name:    "Nil Summary",
			summary: nil,
			wantNil: true,
		},
		{
			name: "Valid Summary",
			summary: &DetectionSummary{
				TotalChecks:      10,
				HighRiskChecks:   3,
				MediumRiskChecks: 2,
				LowRiskChecks:    5,
				Categories: map[string]CategoryResult{
					"webgl": {
						Score:      25.0,
						Detections: []string{"low_extension_count"},
					},
				},
			},
			wantNil: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := convertSummary(tt.summary)
			
			if tt.wantNil && result != nil {
				t.Error("expected nil, got non-nil")
			}
			
			if !tt.wantNil && result == nil {
				t.Error("expected non-nil, got nil")
			}
			
			if result != nil && tt.summary != nil {
				if result.TotalChecks != tt.summary.TotalChecks {
					t.Errorf("TotalChecks = %d, want %d", result.TotalChecks, tt.summary.TotalChecks)
				}
			}
		})
	}
}

func TestConvertToResponse(t *testing.T) {
	handler := NewAdvancedEnvironmentHandler()
	
	response := handler.convertToResponse(nil)
	if response != nil {
		t.Error("convertToResponse with nil should return nil")
	}
}

func BenchmarkAdvancedEnvironmentHandler_DetectEnvironment(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	router := gin.New()
	router.POST("/detect", handler.DetectEnvironment)
	
	request := DetectEnvironmentRequest{
		DetectionID:   "bench_123",
		RiskScore:     30.0,
		RiskLevel:     "low",
		AllDetections: []string{"normal_detection"},
		Fingerprint:   "fp_bench",
		IPAddress:     "192.168.1.1",
		UserAgent:    "Mozilla/5.0",
	}
	
	body, _ := json.Marshal(request)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodPost, "/detect", bytes.NewBuffer(body))
		req.Header.Set("Content-Type", "application/json")
		
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}

func BenchmarkAdvancedEnvironmentHandler_CheckTorNetwork(b *testing.B) {
	gin.SetMode(gin.TestMode)
	
	handler := NewAdvancedEnvironmentHandler()
	
	router := gin.New()
	router.GET("/tor-check", handler.CheckTorNetwork)
	
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		req, _ := http.NewRequest(http.MethodGet, "/tor-check?ip=192.168.1.1", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}
}
