package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewAdvancedAnalyticsHandler(t *testing.T) {
	handler := NewAdvancedAnalyticsHandler()
	if handler == nil {
		t.Error("NewAdvancedAnalyticsHandler 返回了 nil")
	}
}

func TestGetAnalyticsOverview(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/overview", GetAnalyticsOverview)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/overview", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetCaptchaAnalytics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/captcha", GetCaptchaAnalytics)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/captcha", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetUserBehaviorAnalytics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/user-behavior", GetUserBehaviorAnalytics)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/user-behavior", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetSecurityAnalytics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/security", GetSecurityAnalytics)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/security", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetPerformanceAnalytics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/performance", GetPerformanceAnalytics)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/performance", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetAnalyticsReport_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/analytics/report", GetAnalyticsReport)

	reqBody := map[string]interface{}{
		"start_date": "2024-01-01",
		"end_date":   "2024-12-31",
		"type":      "comprehensive",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/analytics/report", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetAnalyticsReport_MissingFields(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/analytics/report", GetAnalyticsReport)

	reqBody := map[string]interface{}{
		"start_date": "2024-01-01",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/analytics/report", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestGetTrendAnalysis_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/trends", GetTrendAnalysis)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/trends", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetRealtimeMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/analytics/realtime", GetRealtimeMetrics)

	req, _ := http.NewRequest("GET", "/api/v1/analytics/realtime", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportAnalytics_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/analytics/export", ExportAnalytics)

	reqBody := map[string]interface{}{
		"format": "json",
		"type":   "captcha",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/analytics/export", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetCustomReport_Success(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/analytics/custom", GetCustomReport)

	reqBody := map[string]interface{}{
		"metrics": []string{"captcha_attempts", "success_rate"},
		"group_by": "day",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/analytics/custom", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}
