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

func TestGetDetectionScript(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/script", GetDetectionScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/script", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "application/javascript", w.Header().Get("Content-Type"))
	assert.Contains(t, w.Body.String(), "function")
	assert.Contains(t, w.Body.String(), "XMLHttpRequest")
	assert.Contains(t, w.Body.String(), "api/v1/detect/submit")
}

func TestGetDetectionScriptWithCallback(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/script", GetDetectionScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/script?callback=myCallback", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Contains(t, w.Body.String(), "myCallback")
}

func TestGetDetectionScriptNoCache(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/script", GetDetectionScript)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/detect/script", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, "no-cache, no-store, must-revalidate", w.Header().Get("Cache-Control"))
	assert.Equal(t, "no-cache", w.Header().Get("Pragma"))
	assert.Equal(t, "0", w.Header().Get("Expires"))
}

func TestGetDetectionScriptRandomization(t *testing.T) {
	r := gin.New()
	r.GET("/api/v1/detect/script", GetDetectionScript)

	results := make([]string, 3)
	for i := 0; i < 3; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/api/v1/detect/script", nil)
		r.ServeHTTP(w, req)
		results[i] = w.Body.String()
	}

	atLeastOneDifferent := false
	for i := 1; i < len(results); i++ {
		if results[i] != results[0] {
			atLeastOneDifferent = true
			break
		}
	}
	assert.True(t, atLeastOneDifferent, "detection scripts should be randomized")
}

func TestSubmitDetectionData(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{
		"detection_id": "test_det_123",
		"risk_score": 25.5,
		"chain": ["webgl", "canvas", "audio"],
		"fingerprint": "scrn:1920x1080|lang:en-US|tz:America/New_York",
		"session_id": "sess_test_123",
		"timestamp": 1700000000000,
		"details": {
			"webgl": "vendor|renderer",
			"canvas": "data:image/png;base64,abc123",
			"webdriver": "no_wd"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
	assert.NotNil(t, resp["risk_score"])
}

func TestSubmitDetectionDataInvalidRiskScore(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	testCases := []struct {
		name  string
		body  string
		code  int
	}{
		{
			name: "negative score",
			body: `{"detection_id":"test1","risk_score":-1}`,
			code: http.StatusBadRequest,
		},
		{
			name: "score over 100",
			body: `{"detection_id":"test2","risk_score":101}`,
			code: http.StatusBadRequest,
		},
		{
			name: "NaN score",
			body: `{"detection_id":"test3","risk_score":"NaN"}`,
			code: http.StatusBadRequest,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)
			assert.Equal(t, tc.code, w.Code)
		})
	}
}

func TestSubmitDetectionDataMissingFields(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(`{}`))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, false, resp["success"])
}

func TestSubmitDetectionDataTimestampAnomaly(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{
		"detection_id": "test_ts_anomaly",
		"risk_score": 10,
		"timestamp": 1
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	riskScore := resp["risk_score"].(float64)
	assert.Greater(t, riskScore, 10.0, "timestamp anomaly should increase risk score")
}

func TestSubmitDetectionDataWithProxyHeaders(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{
		"detection_id": "test_proxy",
		"risk_score": 10,
		"chain": ["webgl"],
		"timestamp": 1700000000000
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 10.0.0.2, 10.0.0.3")
	req.Header.Set("Via", "1.1 proxy.example.com")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	riskScore := resp["risk_score"].(float64)
	assert.Greater(t, riskScore, 10.0, "proxy headers should increase risk score")
}

func TestSubmitDetectionDataWithAutomationIndicators(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{
		"detection_id": "test_auto",
		"risk_score": 10,
		"chain": ["webgl", "canvas", "webdriver", "selenium"],
		"details": {
			"webdriver": "wd:true",
			"selenium": "window.selenium_present",
			"puppeteer": "pw_cdc"
		},
		"timestamp": 1700000000000
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "Mozilla/5.0 Headless Chrome")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	riskScore := resp["risk_score"].(float64)
	assert.Greater(t, riskScore, 30.0, "automation indicators should significantly increase risk score")
}

func TestGenerateDetectionMethods(t *testing.T) {
	methods := generateDetectionMethods()
	assert.Greater(t, len(methods), 20, "should have at least 20 detection methods")

	methodNames := make(map[string]bool)
	for _, m := range methods {
		assert.NotEmpty(t, m.Name)
		assert.NotEmpty(t, m.Code)
		assert.NotEmpty(t, m.ReturnVar)
		assert.False(t, methodNames[m.Name], "duplicate method name: "+m.Name)
		methodNames[m.Name] = true
	}
}

func TestSubmitDetectionDataFullDetails(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{
		"detection_id": "test_full",
		"risk_score": 15,
		"chain": ["webgl", "canvas", "audio", "fonts", "webdriver", "selenium", "puppeteer", "playwright", "webrtc_ip"],
		"fingerprint": "scrn:1920x1080|lang:en|tz:UTC|cpu:8|mem:8|plat:Win32",
		"session_id": "sess_full_test",
		"timestamp": 1700000000000,
		"details": {
			"webgl": "Google Inc.|ANGLE (Intel)",
			"webgl2": "Google Inc.|ANGLE (Intel)",
			"canvas": "base64data",
			"audio": "0.123:0.456",
			"fonts": "Arial,Helvetica",
			"math": "0.866|0.123|2|0.523|0.463|0.707|2.718",
			"platform": "Win32||Google Inc.|20030107|Gecko",
			"languages": "en-US|en-US,en",
			"webdriver": "wd:true",
			"selenium": "window.selenium_present",
			"puppeteer": "pw_cdc",
			"playwright": "no_playwright",
			"memory": "8",
			"cpu": "8",
			"screen": "1920x1080x24",
			"connection": "4g|10|50"
		}
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("User-Agent", "HeadlessChrome")
	req.Header.Set("X-Forwarded-For", "203.0.113.1")
	req.Header.Set("Via", "1.1 squid-proxy")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])

	riskScore := resp["risk_score"].(float64)
	assert.Greater(t, riskScore, 50.0, "multiple risk factors should result in high risk score")
	assert.LessOrEqual(t, riskScore, 100.0)
}

func TestSubmitDetectionDataChainMissingEssential(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{
		"detection_id": "test_missing_chain",
		"risk_score": 5,
		"chain": ["screen", "timezone", "platform"],
		"timestamp": 1700000000000
	}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	riskScore := resp["risk_score"].(float64)
	assert.Greater(t, riskScore, 5.0, "missing essential checks should increase risk score")
}

func TestAnalyzeNetworkHeaders(t *testing.T) {
	r := gin.New()
	r.POST("/api/v1/detect/submit", SubmitDetectionData)

	body := `{"detection_id":"test_headers","risk_score":10,"timestamp":1700000000000}`

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/v1/detect/submit", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Forwarded-For", "1.2.3.4")
	req.Header.Set("X-Real-IP", "5.6.7.8")
	req.Header.Set("CF-Connecting-IP", "9.10.11.12")
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	var resp map[string]interface{}
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, true, resp["success"])
}

func TestDetectionMethodsContainNewMethods(t *testing.T) {
	methods := generateDetectionMethods()

	methodNames := make([]string, len(methods))
	for i, m := range methods {
		methodNames[i] = m.Name
	}

	newMethods := []string{"webgl2", "webrtc_ip", "selenium", "puppeteer", "playwright", "chrome_runtime", "window_size", "iframe", "notification", "media_devices", "gpu", "speech"}
	for _, nm := range newMethods {
		assert.Contains(t, methodNames, nm, "new detection method should exist: "+nm)
	}
}