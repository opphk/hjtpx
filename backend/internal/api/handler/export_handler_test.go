package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestExportHandler_ExportLogs(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/logs", ExportLogsHandler)

	reqBody := map[string]interface{}{
		"format": "csv",
		"filters": map[string]interface{}{
			"start_date": "2024-01-01",
			"end_date":   "2024-01-31",
		},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}

	var resp map[string]interface{}
	if err := json.Unmarshal(w.Body.Bytes(), &resp); err != nil {
		t.Errorf("解析响应失败: %v", err)
	}

	if resp["code"].(float64) != 0 {
		t.Errorf("期望响应码 0, 实际得到 %v", resp["code"])
	}
}

func TestExportHandler_ExportLogs_InvalidFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/logs", ExportLogsHandler)

	reqBody := map[string]interface{}{
		"format": "invalid",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ExportLogs_MissingFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/logs", ExportLogsHandler)

	reqBody := map[string]interface{}{}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/logs", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("期望状态码 400, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ExportStats(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/export/stats", ExportStatsHandler)

	req, _ := http.NewRequest("GET", "/api/v1/export/stats", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ExportStats_ByFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/export/stats/:format", ExportStatsHandler)

	testFormats := []string{"csv", "json", "excel"}
	for _, format := range testFormats {
		req, _ := http.NewRequest("GET", "/api/v1/export/stats/"+format, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("格式 %s 期望状态码 200, 实际得到 %d", format, w.Code)
		}
	}
}

func TestExportHandler_ExportUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/users", ExportUsersHandler)

	reqBody := map[string]interface{}{
		"format":    "csv",
		"user_ids":  []string{"user-1", "user-2"},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ExportUsers_EmptyUsers(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/users", ExportUsersHandler)

	reqBody := map[string]interface{}{
		"format":   "csv",
		"user_ids": []string{},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/users", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("空用户列表应该返回 400, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ScheduleExport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/schedule", ScheduleExportHandler)

	reqBody := map[string]interface{}{
		"type":      "daily",
		"format":    "csv",
		"recipients": []string{"admin@example.com"},
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/schedule", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ScheduleExport_InvalidType(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/schedule", ScheduleExportHandler)

	reqBody := map[string]interface{}{
		"type": "invalid",
		"format": "csv",
	}
	body, _ := json.Marshal(reqBody)

	req, _ := http.NewRequest("POST", "/api/v1/export/schedule", bytes.NewBuffer(body))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效类型应该返回 400, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_ListScheduledExports(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.GET("/api/v1/export/scheduled", ListScheduledExportsHandler)

	req, _ := http.NewRequest("GET", "/api/v1/export/scheduled", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_DeleteScheduledExport(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.DELETE("/api/v1/export/scheduled/:id", DeleteScheduledExportHandler)

	req, _ := http.NewRequest("DELETE", "/api/v1/export/scheduled/123", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("期望状态码 200, 实际得到 %d", w.Code)
	}
}

func TestExportHandler_InvalidJSON(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/v1/export/logs", ExportLogsHandler)

	req, _ := http.NewRequest("POST", "/api/v1/export/logs", bytes.NewBuffer([]byte("invalid json")))
	req.Header.Set("Content-Type", "application/json")
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusBadRequest {
		t.Errorf("无效JSON应该返回 400, 实际得到 %d", w.Code)
	}
}
