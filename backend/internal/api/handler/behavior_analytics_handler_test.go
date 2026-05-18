package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestNewBehaviorAnalyticsHandler(t *testing.T) {
	handler := NewBehaviorAnalyticsHandler()
	if handler == nil {
		t.Error("NewBehaviorAnalyticsHandler 返回了 nil")
	}
}

func TestGetBehaviorAnalytics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/analytics", GetBehaviorAnalytics)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/analytics", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetUserBehaviorPatterns(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/patterns", GetUserBehaviorPatterns)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/patterns", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestAnalyzeBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/behavior/analyze", AnalyzeBehavior)

	reqBody := map[string]interface{}{
		"user_id":    "user-123",
		"session_id": "session-456",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/behavior/analyze", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetBehaviorScore(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/score/:user_id", GetBehaviorScore)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/score/user-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetAnomalyDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/anomalies", GetAnomalyDetection)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/anomalies", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestTrackBehavior(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/behavior/track", TrackBehavior)

	reqBody := map[string]interface{}{
		"user_id": "user-123",
		"action":  "click",
		"target":  "button-submit",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/behavior/track", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetHeatmapData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/heatmap", GetHeatmapData)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/heatmap", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetUserJourney(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/journey/:user_id", GetUserJourney)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/journey/user-123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetEngagementMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/engagement", GetEngagementMetrics)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/engagement", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetRetentionMetrics(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/retention", GetRetentionMetrics)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/retention", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetConversionFunnel(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/funnel", GetConversionFunnel)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/funnel", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestGetCohortAnalysis(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/behavior/cohort", GetCohortAnalysis)

	req, _ := http.NewRequest("GET", "/api/v1/behavior/cohort", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportBehaviorData(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/behavior/export", ExportBehaviorData)

	reqBody := map[string]interface{}{
		"format": "csv",
		"period": "monthly",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/behavior/export", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}
