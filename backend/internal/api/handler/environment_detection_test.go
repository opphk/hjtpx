package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func setupRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestEnvironmentDetectionHandler_DetectEnvironment(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/environment", handler.DetectEnvironment)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		checkSuccess   bool
	}{
		{
			name: "Valid request with basic data",
			body: map[string]interface{}{
				"fingerprint":       "test_fingerprint_001",
				"canvas_hash":       "canvas123",
				"webgl_hash":        "webgl456",
				"user_agent":        "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
				"screen_resolution": "1920x1080",
				"risk_score":        25.0,
			},
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name: "Request with VPN indicators",
			body: map[string]interface{}{
				"fingerprint":     "test_fingerprint_002",
				"canvas_hash":     "canvas789",
				"risk_score":      45.0,
				"connection_type": "vpn",
			},
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name: "Request with automation indicators",
			body: map[string]interface{}{
				"fingerprint": "test_fingerprint_003",
				"canvas_hash": "canvas999",
				"user_agent":  "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/91.0.4472.0 Safari/537.36",
				"risk_score":  85.0,
			},
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name: "Missing fingerprint",
			body: map[string]interface{}{
				"canvas_hash": "canvas123",
				"risk_score":  25.0,
			},
			expectedStatus: http.StatusBadRequest,
			checkSuccess:   false,
		},
		{
			name: "Invalid risk score (negative)",
			body: map[string]interface{}{
				"fingerprint": "test_fp",
				"risk_score":  -10.0,
			},
			expectedStatus: http.StatusBadRequest,
			checkSuccess:   false,
		},
		{
			name: "Invalid risk score (over 100)",
			body: map[string]interface{}{
				"fingerprint": "test_fp",
				"risk_score":  150.0,
			},
			expectedStatus: http.StatusBadRequest,
			checkSuccess:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest("POST", "/api/v1/detect/environment", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("User-Agent", "Test User Agent")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.checkSuccess {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if success, ok := response["success"].(bool); ok && !success {
					t.Error("Expected success=true in response")
				}
			}
		})
	}
}

func TestEnvironmentDetectionHandler_DetectEnvironment_ProxyHeaders(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/environment", handler.DetectEnvironment)

	testCases := []struct {
		name        string
		headers     map[string]string
		expectVPN   bool
		expectProxy bool
	}{
		{
			name: "No proxy headers",
			headers: map[string]string{
				"X-Forwarded-For": "",
				"X-Real-IP":       "",
			},
			expectVPN:   false,
			expectProxy: false,
		},
		{
			name: "With X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "192.0.2.1",
			},
			expectVPN:   false,
			expectProxy: true,
		},
		{
			name: "With multiple proxy headers",
			headers: map[string]string{
				"X-Forwarded-For": "192.0.2.1, 192.0.2.2",
				"X-Real-IP":       "192.0.2.1",
				"Via":             "1.1 proxy.example.com",
			},
			expectVPN:   false,
			expectProxy: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			body := map[string]interface{}{
				"fingerprint": "proxy_test_fp",
				"canvas_hash": "canvas_hash",
				"risk_score":  30.0,
			}
			bodyBytes, _ := json.Marshal(body)

			req, _ := http.NewRequest("POST", "/api/v1/detect/environment", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			for key, value := range tc.headers {
				req.Header.Set(key, value)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
			}

			var response map[string]interface{}
			if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
				t.Fatalf("Failed to parse response: %v", err)
			}

			data, ok := response["data"].(map[string]interface{})
			if !ok {
				t.Fatal("Expected data field in response")
			}

			if tc.expectProxy {
				if isVPN, ok := data["is_proxy"].(bool); !ok || !isVPN {
					t.Logf("Proxy detection: got %v", data["is_proxy"])
				}
			}
		})
	}
}

func TestEnvironmentDetectionHandler_GetFingerprintAnalysis(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/environment", handler.DetectEnvironment)
	router.GET("/api/v1/detect/fingerprint-analysis", handler.GetFingerprintAnalysis)

	fingerprint := "analysis_test_fp"
	body := map[string]interface{}{
		"fingerprint": fingerprint,
		"canvas_hash": "canvas_test_hash",
		"webgl_hash":  "webgl_test_hash",
		"user_agent":  "Mozilla/5.0",
		"risk_score":  20.0,
	}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/detect/environment", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Fatalf("Setup failed: expected status %d, got %d", http.StatusOK, w.Code)
	}

	var createResponse map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &createResponse)

	data, ok := createResponse["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	actualFingerprint, ok := data["fingerprint_id"].(string)
	if !ok || actualFingerprint == "" {
		t.Fatal("Expected fingerprint_id in response")
	}

	getReq, _ := http.NewRequest("GET", "/api/v1/detect/fingerprint-analysis?fingerprint="+actualFingerprint, nil)
	getW := httptest.NewRecorder()
	router.ServeHTTP(getW, getReq)

	if getW.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, getW.Code)
	}

	var analysisResponse map[string]interface{}
	if err := json.Unmarshal(getW.Body.Bytes(), &analysisResponse); err != nil {
		t.Fatalf("Failed to parse analysis response: %v", err)
	}

	if success, ok := analysisResponse["success"].(bool); !ok || !success {
		t.Error("Expected success=true in analysis response")
	}
}

func TestEnvironmentDetectionHandler_GetFingerprintAnalysis_NotFound(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.GET("/api/v1/detect/fingerprint-analysis", handler.GetFingerprintAnalysis)

	req, _ := http.NewRequest("GET", "/api/v1/detect/fingerprint-analysis?fingerprint=nonexistent_fp", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusNotFound {
		t.Errorf("Expected status %d, got %d", http.StatusNotFound, w.Code)
	}

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	if success, ok := response["success"].(bool); !ok || success {
		t.Error("Expected success=false for non-existent fingerprint")
	}
}

func TestEnvironmentDetectionHandler_CheckProxy(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/proxy-check", handler.CheckProxy)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		checkProxy     bool
	}{
		{
			name:           "Valid proxy check - direct IP",
			body:           map[string]interface{}{"ip_address": "203.0.113.1"},
			expectedStatus: http.StatusOK,
			checkProxy:     false,
		},
		{
			name:           "Valid proxy check - datacenter IP",
			body:           map[string]interface{}{"ip_address": "52.94.236.1"},
			expectedStatus: http.StatusOK,
			checkProxy:     true,
		},
		{
			name:           "Missing IP address",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			checkProxy:     false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest("POST", "/api/v1/detect/proxy-check", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if success, ok := response["success"].(bool); !ok || !success {
					t.Error("Expected success=true in response")
				}
			}
		})
	}
}

func TestEnvironmentDetectionHandler_GetDetectionStats(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.GET("/api/v1/detect/stats", handler.GetDetectionStats)

	req, _ := http.NewRequest("GET", "/api/v1/detect/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true in response")
	}

	data, ok := response["data"].(map[string]interface{})
	if !ok {
		t.Fatal("Expected data field in response")
	}

	expectedFields := []string{"total_fingerprints", "bot_fingerprints", "vpn_fingerprints", "avg_anomaly_score"}
	for _, field := range expectedFields {
		if _, exists := data[field]; !exists {
			t.Errorf("Expected field '%s' in stats response", field)
		}
	}
}

func TestEnvironmentDetectionHandler_GetClusters(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/environment", handler.DetectEnvironment)
	router.GET("/api/v1/detect/clusters", handler.GetClusters)

	for i := 0; i < 5; i++ {
		body := map[string]interface{}{
			"fingerprint": "cluster_test_fp_" + string(rune('0'+i)),
			"canvas_hash": "cluster_hash",
			"webgl_hash":  "webgl_hash",
			"user_agent":  "Mozilla/5.0",
			"risk_score":  30.0,
		}
		bodyBytes, _ := json.Marshal(body)
		req, _ := http.NewRequest("POST", "/api/v1/detect/environment", bytes.NewBuffer(bodyBytes))
		req.Header.Set("Content-Type", "application/json")

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)
	}

	req, _ := http.NewRequest("GET", "/api/v1/detect/clusters", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true in response")
	}
}

func TestEnvironmentDetectionHandler_BatchDetectProxy(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/proxy/batch", handler.BatchDetectProxy)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		checkSuccess   bool
	}{
		{
			name: "Valid batch request",
			body: map[string]interface{}{
				"ips": []string{"203.0.113.1", "52.94.236.1", "8.8.8.8"},
			},
			expectedStatus: http.StatusOK,
			checkSuccess:   true,
		},
		{
			name:           "Empty IP list",
			body:           map[string]interface{}{"ips": []string{}},
			expectedStatus: http.StatusBadRequest,
			checkSuccess:   false,
		},
		{
			name:           "Too many IPs",
			body:           map[string]interface{}{"ips": make([]string, 150)},
			expectedStatus: http.StatusBadRequest,
			checkSuccess:   false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest("POST", "/api/v1/detect/proxy/batch", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.checkSuccess {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if success, ok := response["success"].(bool); !ok || !success {
					t.Error("Expected success=true in response")
				}
			}
		})
	}
}

func TestEnvironmentDetectionHandler_ValidateHeaders(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/validate-headers", handler.ValidateHeaders)

	testCases := []struct {
		name           string
		body           map[string]interface{}
		expectedStatus int
		expectFlagged  bool
	}{
		{
			name: "Clean headers",
			body: map[string]interface{}{
				"headers": map[string]string{
					"Content-Type": "application/json",
					"Accept":       "application/json",
				},
			},
			expectedStatus: http.StatusOK,
			expectFlagged:  false,
		},
		{
			name: "Headers with proxy keyword",
			body: map[string]interface{}{
				"headers": map[string]string{
					"Via": "1.1 proxy.example.com",
				},
			},
			expectedStatus: http.StatusOK,
			expectFlagged:  true,
		},
		{
			name: "Headers with VPN keyword",
			body: map[string]interface{}{
				"headers": map[string]string{
					"X-VPN-Connection": "enabled",
				},
			},
			expectedStatus: http.StatusOK,
			expectFlagged:  true,
		},
		{
			name:           "Missing headers",
			body:           map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
			expectFlagged:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			bodyBytes, _ := json.Marshal(tc.body)
			req, _ := http.NewRequest("POST", "/api/v1/detect/validate-headers", bytes.NewBuffer(bodyBytes))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}

			if tc.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
					t.Fatalf("Failed to parse response: %v", err)
				}

				if isFlagged, ok := response["is_flagged"].(bool); ok {
					if tc.expectFlagged && !isFlagged {
						t.Error("Expected headers to be flagged")
					}
				}
			}
		})
	}
}

func TestEnvironmentDetectionHandler_GetVPNPatterns(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.GET("/api/v1/detect/vpn-patterns", handler.GetVPNPatterns)

	req, _ := http.NewRequest("GET", "/api/v1/detect/vpn-patterns", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, w.Code)
	}

	var response map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &response); err != nil {
		t.Fatalf("Failed to parse response: %v", err)
	}

	if success, ok := response["success"].(bool); !ok || !success {
		t.Error("Expected success=true in response")
	}

	data, ok := response["data"].([]interface{})
	if !ok {
		t.Fatal("Expected data to be an array")
	}

	if len(data) == 0 {
		t.Error("Expected at least one VPN pattern")
	}
}

func TestCalculateCombinedRiskScore(t *testing.T) {
	testCases := []struct {
		name        string
		clientScore float64
		minExpected float64
		maxExpected float64
	}{
		{"Low client score", 20, 0, 30},
		{"Medium client score", 50, 30, 70},
		{"High client score", 80, 60, 100},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			score := calculateCombinedRiskScore(tc.clientScore, nil, nil, nil)

			if score < tc.minExpected {
				t.Errorf("Expected score >= %.2f, got %.2f", tc.minExpected, score)
			}
			if score > tc.maxExpected {
				t.Errorf("Expected score <= %.2f, got %.2f", tc.maxExpected, score)
			}
		})
	}
}

func TestGenerateRecommendations(t *testing.T) {
	testCases := []struct {
		name       string
		riskScore  float64
		minRecs    int
		expectHigh bool
	}{
		{"High risk", 85, 1, true},
		{"Medium risk", 55, 1, false},
		{"Low risk", 25, 1, false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			recs := generateRecommendations(tc.riskScore, nil, nil)

			if len(recs) < tc.minRecs {
				t.Errorf("Expected at least %d recommendations, got %d", tc.minRecs, len(recs))
			}
		})
	}
}

func TestEnvironmentDetectionHandler_CleanupOldData(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.DELETE("/api/v1/detect/fingerprints/cleanup", handler.CleanupOldData)

	testCases := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{"Default cleanup", "", http.StatusOK},
		{"Custom 24h", "?max_age=24h", http.StatusOK},
		{"Custom 7d", "?max_age=7d", http.StatusOK},
		{"Invalid format", "?max_age=invalid", http.StatusBadRequest},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req, _ := http.NewRequest("DELETE", "/api/v1/detect/fingerprints/cleanup"+tc.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != tc.expectedStatus {
				t.Errorf("Expected status %d, got %d", tc.expectedStatus, w.Code)
			}
		})
	}
}

func TestEnvironmentDetectionHandler_ExportImportFingerprintData(t *testing.T) {
	router := setupRouter()
	handler := NewEnvironmentDetectionHandler()

	router.POST("/api/v1/detect/environment", handler.DetectEnvironment)
	router.GET("/api/v1/detect/export", handler.ExportFingerprintData)

	body := map[string]interface{}{
		"fingerprint": "export_test_fp",
		"canvas_hash": "canvas_hash",
		"risk_score":  30.0,
	}
	bodyBytes, _ := json.Marshal(body)

	req, _ := http.NewRequest("POST", "/api/v1/detect/environment", bytes.NewBuffer(bodyBytes))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	exportReq, _ := http.NewRequest("GET", "/api/v1/detect/export", nil)
	exportW := httptest.NewRecorder()
	router.ServeHTTP(exportW, exportReq)

	if exportW.Code != http.StatusOK {
		t.Errorf("Expected status %d, got %d", http.StatusOK, exportW.Code)
	}

	contentType := exportW.Header().Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("Expected Content-Type 'application/json', got '%s'", contentType)
	}

	if len(exportW.Body.Bytes()) == 0 {
		t.Error("Expected non-empty export data")
	}
}

func TestParseDuration(t *testing.T) {
	testCases := []struct {
		name        string
		input       string
		expectError bool
		expectHours int
	}{
		{"Hours format", "24h", false, 24},
		{"Days format", "7d", false, 168},
		{"Minutes format", "60m", false, 1},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			duration, err := parseDuration(tc.input)

			if tc.expectError && err == nil {
				t.Error("Expected error but got none")
			}
			if !tc.expectError && err != nil {
				t.Errorf("Unexpected error: %v", err)
			}
			if !tc.expectError {
				hours := int(duration.Hours())
				if hours != tc.expectHours {
					t.Errorf("Expected %d hours, got %d", tc.expectHours, hours)
				}
			}
		})
	}
}

func TestNewEnvironmentDetectionHandler(t *testing.T) {
	handler := NewEnvironmentDetectionHandler()

	if handler == nil {
		t.Fatal("Expected handler to be created")
	}

	if handler.fingerprintAnalyzer == nil {
		t.Error("Expected fingerprintAnalyzer to be initialized")
	}

	if handler.proxyDetector == nil {
		t.Error("Expected proxyDetector to be initialized")
	}
}
