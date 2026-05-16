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

func TestGetAdvancedDetectionScriptNoCache(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/advanced/script", GetAdvancedDetectionScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/advanced/script", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
	assert.Equal(t, "0", w.Header().Get("Expires"))
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
			"webgl": "Google Inc.|ANGLE (Intel)",
			"fonts": "Arial,Helvetica",
			"webdriver": "no_wd"
		},
		"timestamp": 1700000000000
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
	assert.NotNil(t, resp["risk_score"])
	assert.NotNil(t, resp["confidence"])
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

func TestSubmitAdvancedDetectionInvalidSession(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)

	body := `{"session_id": "", "data": {}}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestSubmitAdvancedDetectionWithUserAgent(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)

	body := `{
		"session_id": "test_session_ua",
		"data": {
			"user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36",
			"webdriver": "true"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HeadlessChrome/120.0")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	riskScore := resp["risk_score"].(float64)
	assert.Greater(t, riskScore, 0.0, "automation detection should increase risk score")
}

func TestSubmitAdvancedDetectionCloudIP(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)

	body := `{
		"session_id": "test_cloud",
		"data": {
			"ip_info": {
				"ip": "54.123.45.67",
				"isp": "Amazon.com Inc.",
				"organization": "AWS EC2"
			}
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

func TestGetAdvancedDetectionResult(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)
	r.GET("/api/v1/detect/advanced/result", GetAdvancedDetectionResult)

	body := `{
		"session_id": "result_test_session",
		"data": {
			"user_agent": "Mozilla/5.0 Chrome/120.0",
			"webgl": "Google Inc.|ANGLE (Intel)"
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

	var resp2 map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.NoError(t, err)
	assert.Equal(t, true, resp2["success"])
}

func TestGetAdvancedDetectionResultNotFound(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/advanced/result", GetAdvancedDetectionResult)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/advanced/result?session_id=nonexistent", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetAdvancedDetectionResultMissingSessionID(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/advanced/result", GetAdvancedDetectionResult)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/advanced/result", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
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

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "chrome", data["browser"])
	assert.Equal(t, "blink", data["engine"])
	assert.Equal(t, "Windows", data["os"])
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
	assert.Equal(t, "gecko", data["engine"])
}

func TestAnalyzeBrowserEngineSafari(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/browser-engine", AnalyzeBrowserEngine)

	body := `{"user_agent": "Mozilla/5.0 (Macintosh; Intel Mac OS X 10_15_7) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Safari/605.1.15"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/browser-engine", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "safari", data["browser"])
	assert.Equal(t, "webkit", data["engine"])
}

func TestAnalyzeBrowserEngineEdge(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/browser-engine", AnalyzeBrowserEngine)

	body := `{"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36 Edg/120.0.0.0"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/browser-engine", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "edge", data["browser"])
	assert.Equal(t, "blink", data["engine"])
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

func TestAnalyzeBrowserEngineMobile(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/browser-engine", AnalyzeBrowserEngine)

	body := `{"user_agent": "Mozilla/5.0 (iPhone; CPU iPhone OS 17_2 like Mac OS X) AppleWebKit/605.1.15 (KHTML, like Gecko) Version/17.2 Mobile/15E148 Safari/604.1"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/browser-engine", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["mobile"])
}

func TestDetectVMEnvironment(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/vm", DetectVMEnvironment)

	body := `{
		"webgl_renderer": "SwiftShader for Chrome",
		"screen_size": "1920x1080",
		"cpu_cores": 2,
		"device_memory": 0.5
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/vm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_vm"])
}

func TestDetectVMEnvironmentVMware(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/vm", DetectVMEnvironment)

	body := `{"webgl_renderer": "VMware Virtual Platform Graphics Adapter"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/vm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_vm"])
}

func TestDetectVMEnvironmentNormal(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/vm", DetectVMEnvironment)

	body := `{
		"webgl_renderer": "NVIDIA GeForce RTX 3080",
		"screen_size": "1920x1080",
		"cpu_cores": 8,
		"device_memory": 32
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/vm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["is_vm"])
}

func TestDetectVMEnvironmentZeroScreen(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/vm", DetectVMEnvironment)

	body := `{"screen_size": "0x0"}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/vm", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_vm"])
}

func TestDetectCloudEnvironment(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{
		"ip_address": "54.123.45.67",
		"isp": "Amazon.com Inc.",
		"organization": "Amazon Data Services"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_cloud"])
	assert.Equal(t, "aws", data["provider"])
}

func TestDetectCloudEnvironmentGCP(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{
		"ip_address": "34.123.45.67",
		"isp": "Google LLC",
		"organization": "Google Cloud Platform"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_cloud"])
	assert.Equal(t, "gcp", data["provider"])
}

func TestDetectCloudEnvironmentAzure(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{
		"ip_address": "13.65.89.123",
		"isp": "Microsoft Corporation",
		"organization": "Microsoft Azure"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_cloud"])
	assert.Equal(t, "azure", data["provider"])
}

func TestDetectCloudEnvironmentDatacenter(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{
		"ip_address": "192.168.1.100",
		"organization": "Some Datacenter Inc."
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_datacenter"])
}

func TestDetectCloudEnvironmentNormal(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/cloud", DetectCloudEnvironment)

	body := `{
		"ip_address": "203.0.113.42",
		"isp": "ISP Name",
		"organization": "Home User"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/cloud", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["is_cloud"])
}

func TestDetectContainerEnvironment(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/container", DetectContainerEnvironment)

	body := `{
		"cpu_cores": 1,
		"device_memory": 0.25,
		"platform": "Linux"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/container", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_container"])
}

func TestDetectContainerEnvironmentDocker(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/container", DetectContainerEnvironment)

	body := `{
		"cpu_cores": 2,
		"device_memory": 1,
		"platform": "docker",
		"user_agent": "Mozilla/5.0 (X11; Linux x86_64) Docker Container"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/container", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_container"])
}

func TestDetectContainerEnvironmentNormal(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/container", DetectContainerEnvironment)

	body := `{
		"cpu_cores": 8,
		"device_memory": 16,
		"platform": "Win32"
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/container", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["is_container"])
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
		"permissions": {"notifications": "denied", "geolocation": "denied"}
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

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_headless"])
	assert.Greater(t, data["confidence"].(float64), 0.3)
}

func TestDetectHeadlessBrowserPuppeteer(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/headless", DetectHeadlessBrowser)

	body := `{
		"user_agent": "Mozilla/5.0 (X11; Linux x86_64) AppleWebKit/537.36 (KHTML, like Gecko) HeadlessChrome/120.0.0.0 Safari/537.36",
		"navigator_webdriver": true
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/headless", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_headless"])
}

func TestDetectHeadlessBrowserNormal(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/headless", DetectHeadlessBrowser)

	body := `{
		"user_agent": "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36 (KHTML, like Gecko) Chrome/120.0.0.0 Safari/537.36",
		"navigator_webdriver": false,
		"chrome_runtime": true,
		"plugins_count": 5,
		"languages": ["en-US", "en"],
		"permissions": {}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/headless", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, false, data["is_headless"])
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

	data := resp["data"].(map[string]interface{})
	assert.NotNil(t, data["md5_hash"])
	assert.NotNil(t, data["length"])
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
		"extensions": ["WEBGL_debug_renderer_info", "EXT_texture_filter_anisotropic"],
		"max_texture_size": 16384
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/webgl", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, "Google Inc.", data["vendor"])
	assert.Equal(t, false, data["is_software"])
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

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)

	data := resp["data"].(map[string]interface{})
	assert.Equal(t, true, data["is_software"])
	assert.Equal(t, "SwiftShader", data["software_name"])
}

func TestBatchDetection(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/submit", SubmitAdvancedDetection)
	r.POST("/api/v1/detect/advanced/batch", BatchDetection)

	body1 := `{
		"session_id": "batch_test_1",
		"data": {"user_agent": "Mozilla/5.0 Chrome/120.0"}
	}`

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

	var resp2 map[string]interface{}
	err := json.Unmarshal(w2.Body.Bytes(), &resp2)
	assert.NoError(t, err)
	assert.Equal(t, true, resp2["success"])
	assert.Equal(t, float64(1), resp2["count"])
}

func TestBatchDetectionEmpty(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/batch", BatchDetection)

	body := `{"session_ids": ["nonexistent1", "nonexistent2"]}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, float64(0), resp["count"])
}

func TestBatchDetectionMissingSessionIDs(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/advanced/batch", BatchDetection)

	body := `{}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/advanced/batch", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestRiskLevelCritical(t *testing.T) {
	result := &AdvancedDetectionResponse{RiskScore: 85}
	assert.Equal(t, "critical", getRiskLevel(result.RiskScore))
}

func TestRiskLevelHigh(t *testing.T) {
	result := &AdvancedDetectionResponse{RiskScore: 65}
	assert.Equal(t, "high", getRiskLevel(result.RiskScore))
}

func TestRiskLevelMedium(t *testing.T) {
	result := &AdvancedDetectionResponse{RiskScore: 45}
	assert.Equal(t, "medium", getRiskLevel(result.RiskScore))
}

func TestRiskLevelLow(t *testing.T) {
	result := &AdvancedDetectionResponse{RiskScore: 25}
	assert.Equal(t, "low", getRiskLevel(result.RiskScore))
}

func TestRiskLevelMinimal(t *testing.T) {
	result := &AdvancedDetectionResponse{RiskScore: 10}
	assert.Equal(t, "minimal", getRiskLevel(result.RiskScore))
}
