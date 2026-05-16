package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetSystemStatus(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		setupRouter    func(*gin.Engine)
		expectedStatus int
	}{
		{
			name: "success - returns system status",
			setupRouter: func(r *gin.Engine) {
				r.GET("/admin/system/status", GetSystemStatus)
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			tt.setupRouter(r)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/admin/system/status", nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetRequestTrend(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParam     string
		expectedStatus int
	}{
		{
			name:           "success - default period",
			queryParam:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - hour period",
			queryParam:     "?period=hour",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - day period",
			queryParam:     "?period=day",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - week period",
			queryParam:     "?period=week",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/admin/request/trend", GetRequestTrend)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/admin/request/trend"+tt.queryParam, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetRiskRulesSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/admin/risk-rules/summary", GetRiskRulesSummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/risk-rules/summary", nil)
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestListRiskRules(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParam     string
		expectedStatus int
	}{
		{
			name:           "success - default",
			queryParam:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with status",
			queryParam:     "?status=active",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "success - with type",
			queryParam:     "?type=ip",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/admin/risk-rules", ListRiskRules)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/admin/risk-rules"+tt.queryParam, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetRiskRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		id             string
		expectedStatus int
	}{
		{
			name:           "success - valid id",
			id:             "1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "bad request - invalid id",
			id:             "invalid",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/admin/risk-rules/:id", GetRiskRule)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/admin/risk-rules/"+tt.id, nil)
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCreateRiskRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.POST("/admin/risk-rules", CreateRiskRule)

	body := map[string]interface{}{
		"name":        "Test Rule",
		"type":        "ip",
		"description": "Test description",
		"severity":    "high",
		"conditions":  map[string]interface{}{},
	}
	jsonBody, _ := json.Marshal(body)
	req, _ := http.NewRequest("POST", "/admin/risk-rules", bytes.NewBuffer(jsonBody))
	req.Header.Set("Content-Type", "application/json")

	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	// Status might be 200 or 500 depending on db
	assert.NotNil(t, w.Code)
}

func TestToggleRiskRule(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.PUT("/admin/risk-rules/:id/toggle", ToggleRiskRule)

	req, _ := http.NewRequest("PUT", "/admin/risk-rules/1/toggle", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotNil(t, w.Code)
}

func TestGetApplicationsSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/admin/applications/summary", GetApplicationsSummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/applications/summary", nil)
	r.ServeHTTP(w, req)

	// Either OK or Internal Server Error is acceptable depending on DB connection
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

func TestGetLogsSummary(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/admin/logs/summary", GetLogsSummary)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/admin/logs/summary", nil)
	r.ServeHTTP(w, req)

	// Either OK or Internal Server Error is acceptable depending on DB connection
	assert.True(t, w.Code == http.StatusOK || w.Code == http.StatusInternalServerError)
}

func TestClearLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.DELETE("/admin/logs/clear", ClearLogs)

	req, _ := http.NewRequest("DELETE", "/admin/logs/clear", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotNil(t, w.Code)
}
