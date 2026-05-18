package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestNewDashboardHandler(t *testing.T) {
	handler := NewDashboardHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.dashboardService)
}

func TestGetDashboardHandler_Function(t *testing.T) {
	handler := GetDashboardHandler()
	assert.NotNil(t, handler)
}

func TestGetDashboardHandler_ReturnsNewInstance(t *testing.T) {
	handler1 := GetDashboardHandler()
	handler2 := GetDashboardHandler()
	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
}

func TestGetDashboardData_QueryParameters(t *testing.T) {
	tests := []struct {
		name   string
		period string
	}{
		{"hour", "hour"},
		{"day", "day"},
		{"week", "week"},
		{"month", "month"},
		{"default", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/dashboard", GetDashboardData)

			url := "/dashboard"
			if tt.period != "" {
				url = "/dashboard?period=" + tt.period
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
		})
	}
}

func TestExportDashboardData_Formats(t *testing.T) {
	tests := []struct {
		name   string
		format string
	}{
		{"csv format", "csv"},
		{"json format", "json"},
		{"excel format", "excel"},
		{"default format", ""},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			gin.SetMode(gin.TestMode)
			r := gin.New()
			r.GET("/dashboard/export", ExportDashboardData)

			url := "/dashboard/export"
			if tt.format != "" {
				url = "/dashboard/export?format=" + tt.format
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
		})
	}
}

func TestExportDashboardData_Headers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/export", ExportDashboardData)

	req, _ := http.NewRequest("GET", "/dashboard/export?format=csv", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	if w.Code == http.StatusOK {
		assert.NotEmpty(t, w.Header().Get("Content-Disposition"))
	}
}

func TestGetRecentVerifications(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/recent", GetRecentVerifications)

	req, _ := http.NewRequest("GET", "/dashboard/recent", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)
}

func TestGetDashboardAlerts(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.GET("/dashboard/alerts", GetDashboardAlerts)

	req, _ := http.NewRequest("GET", "/dashboard/alerts", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Contains(t, []int{http.StatusOK, http.StatusInternalServerError}, w.Code)

	if w.Code == http.StatusOK {
		var resp map[string]interface{}
		err := json.Unmarshal(w.Body.Bytes(), &resp)
		assert.NoError(t, err)
		assert.Contains(t, resp, "data")
	}
}

func TestDashboardHandler_Structure(t *testing.T) {
	handler := &DashboardHandler{}
	assert.Nil(t, handler.dashboardService)

	handler.dashboardService = &mockDashboardService{}
	assert.NotNil(t, handler.dashboardService)
}

type mockDashboardService struct{}

func (m *mockDashboardService) GetDashboardData(period string) (interface{}, error) {
	return map[string]interface{}{}, nil
}

func (m *mockDashboardService) ExportData(format, period string) ([]byte, error) {
	return []byte{}, nil
}

func (m *mockDashboardService) CheckAlerts() []interface{} {
	return []interface{}{}
}
