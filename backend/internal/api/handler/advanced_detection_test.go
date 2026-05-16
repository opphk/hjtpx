package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestGetAdvancedDetectionScript(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/advanced/script", GetAdvancedDetectionScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/advanced/script", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/javascript", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "fetch")
	assert.Contains(t, w.Body.String(), "/api/v1/detect/advanced/submit")
}

func TestSubmitAdvancedDetection(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)

	body := `{
		"session_id": "test_session_123",
		"fingerprint": "test_fp",
		"data": {
			"user_agent": "Mozilla/5.0 Chrome/120.0",
			"canvas": "test_canvas_data",
			"webgl": "Google Inc.|ANGLE (Intel)"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestSubmitAdvancedDetectionMissingData(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)

	body := `{"session_id": "test_session"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestGetAdvancedDetectionResult(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)
	r.GET("/api/v1/detect/advanced/result", GetAdvancedDetectionResult)

	body := `{
		"session_id": "result_test_session",
		"data": {
			"user_agent": "Mozilla/5.0 Chrome/120.0"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	sessionID := "result_test_session"

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("GET", "/api/v1/detect/advanced/result?session_id="+sessionID, nil)
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestAnalyzeBrowserEngine(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/browser-engine", AnalyzeBrowserEngine)

	body := `{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/browser-engine", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestAnalyzeBrowserEngineFirefox(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/browser-engine", AnalyzeBrowserEngine)

	body := `{"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10.15; rv:120.0) Gecko/20100101 Firefox/120.0"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/browser-engine", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "firefox", data["browser"])
}

func TestAnalyzeBrowserEngineMissingUA(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/browser-engine", AnalyzeBrowserEngine)

	body := `{}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/browser-engine", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDetectVMEnvironment(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/vm", DetectVMEnvironment)

	body := `{"webgl_renderer": "SwiftShader for Chrome", "screen_size": "1920x1080"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/vm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestDetectVMEnvironmentNormal(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/vm", DetectVMEnvironment)

	body := `{"webgl_renderer": "NVIDIA GeForce RTX 3080"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/vm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDetectCloudEnvironment(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{"ip_address": "54.123.45.67", "isp": "Amazon.com Inc."}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDetectCloudEnvironmentGCP(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{"ip_address": "34.123.45.67", "isp": "Google LLC"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDetectContainerEnvironment(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/container", DetectContainerEnvironment)

	body := `{"cpu_cores": 1, "device_memory": 0.25}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/container", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDetectHeadlessBrowser(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/headless", DetectHeadlessBrowser)

	body := `{
		"user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36",
		"navigator_webdriver": true,
		"chrome_runtime": false,
		"plugins_count": 0,
		"languages": [],
		"permissions": {"notifications": "denied"}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/headless", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestEnhancedCanvasFingerprint(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/canvas", EnhancedCanvasFingerprint)

	body := `{"canvas_data": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAAAAEAAAABCAYAAAAfFcSJAAAADUlEQVR42mNk+M9QDwADhgGAWjR9awAAAABJRU5ErkJggg=="}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/canvas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestEnhancedCanvasFingerprintEmpty(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/canvas", EnhancedCanvasFingerprint)

	body := `{"canvas_data": ""}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/canvas", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnhancedWebGLFingerprint(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/webgl", EnhancedWebGLFingerprint)

	body := `{
		"vendor": "Google Inc.",
		"renderer": "ANGLE (Intel)",
		"extensions": ["WEBGL_debug_renderer_info"],
		"max_texture_size": 16384
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/webgl", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestEnhancedWebGLFingerprintSoftware(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/webgl", EnhancedWebGLFingerprint)

	body := `{
		"vendor": "Google Inc.",
		"renderer": "SwiftShader for Chrome",
		"extensions": [],
		"max_texture_size": 1024
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/webgl", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestBatchDetection(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)
	r.POST("/api/v1/detect/advanced/batch", BatchDetection)

	body1 := `{"session_id": "batch_test_1", "data": {"user_agent": "Mozilla/5.0 Chrome/120.0"}}`

	w1 := httptest.NewRecorder()
	req1, _ := http.NewRequest("POST", "/api/v1/detect/advanced/submit", strings.NewReader(body1))
	req1.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w1, req1)

	body2 := `{"session_ids": ["batch_test_1"]}`

	w2 := httptest.NewRecorder()
	req2, _ := http.NewRequest("POST", "/api/v1/detect/advanced/batch", strings.NewReader(body2))
	req2.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w2, req2)

	assert.Equal(t, http.StatusOK, w2.Code)
}

func TestBatchDetectionEmpty(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/batch", BatchDetection)

	body := `{"session_ids": ["nonexistent"]}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRiskLevelCritical(t *testing.T) {
	assert.Equal(t, "critical", getRiskLevel(85))
}

func TestRiskLevelHigh(t *testing.T) {
	assert.Equal(t, "high", getRiskLevel(65))
}

func TestRiskLevelMedium(t *testing.T) {
	assert.Equal(t, "medium", getRiskLevel(45))
}

func TestRiskLevelLow(t *testing.T) {
	assert.Equal(t, "low", getRiskLevel(25))
}

func TestRiskLevelMinimal(t *testing.T) {
	assert.Equal(t, "minimal", getRiskLevel(10))
}
