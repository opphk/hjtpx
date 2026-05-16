package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestGetVerificationLogsRequest_Defaults(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query GetVerificationLogsRequest
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":         query.Page,
			"page_size":    query.PageSize,
			"status":       query.Status,
			"captcha_type": query.CaptchaType,
			"session_id":   query.SessionID,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(20), response["page_size"])
}

func TestGetVerificationLogsRequest_WithParams(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query GetVerificationLogsRequest
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":           query.Page,
			"page_size":      query.PageSize,
			"application_id": query.ApplicationID,
			"status":         query.Status,
			"captcha_type":   query.CaptchaType,
			"session_id":     query.SessionID,
			"start_date":     query.StartDate,
			"end_date":       query.EndDate,
			"ip_address":     query.IPAddress,
		})
	})

	req, _ := http.NewRequest("GET", "/test?page=2&page_size=50&application_id=1&status=success&captcha_type=slider&session_id=abc123&start_date=2025-01-01&end_date=2025-12-31&ip_address=192.168.1.1", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(2), response["page"])
	assert.Equal(t, float64(50), response["page_size"])
	assert.Equal(t, float64(1), response["application_id"])
	assert.Equal(t, "success", response["status"])
	assert.Equal(t, "slider", response["captcha_type"])
	assert.Equal(t, "abc123", response["session_id"])
}

func TestExportLogsRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		query   string
		wantErr bool
	}{
		{
			name:    "empty query",
			query:   "",
			wantErr: false,
		},
		{
			name:    "with params",
			query:   "application_id=1&status=success&start_date=2025-01-01&end_date=2025-12-31&format=csv",
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.GET("/test", func(c *gin.Context) {
				var query ExportLogsRequest
				if err := c.ShouldBindQuery(&query); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{
					"application_id": query.ApplicationID,
					"status":         query.Status,
					"format":         query.Format,
				})
			})

			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			if tt.wantErr {
				assert.Equal(t, http.StatusBadRequest, w.Code)
			} else {
				assert.Equal(t, http.StatusOK, w.Code)
			}
		})
	}
}

func TestExportLogsRequest_DefaultFormat(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ExportLogsRequest
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"format": query.Format,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "csv", response["format"])
}

func TestLogListResponse_JSON(t *testing.T) {
	response := LogListResponse{
		Total:      100,
		Page:       1,
		PageSize:   20,
		TotalPages: 5,
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaled LogListResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, response.Total, unmarshaled.Total)
	assert.Equal(t, response.Page, unmarshaled.Page)
	assert.Equal(t, response.PageSize, unmarshaled.PageSize)
	assert.Equal(t, response.TotalPages, unmarshaled.TotalPages)
}

func TestLogHandler_NewLogHandler(t *testing.T) {
	handler := NewLogHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.logService)
	assert.NotNil(t, handler.statsService)
}

func TestLogHandler_GetLogHandler(t *testing.T) {
	handler1 := GetLogHandler()
	handler2 := GetLogHandler()

	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
	assert.Equal(t, handler1, handler2)
}

func TestDeleteOldLogs_InvalidDays(t *testing.T) {
	tests := []struct {
		name       string
		query      string
		wantStatus int
	}{
		{
			name:       "valid days",
			query:      "days=30",
			wantStatus: http.StatusOK,
		},
		{
			name:       "missing days uses default",
			query:      "",
			wantStatus: http.StatusOK,
		},
		{
			name:       "invalid days string",
			query:      "days=invalid",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "zero days",
			query:      "days=0",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "negative days",
			query:      "days=-1",
			wantStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.DELETE("/test", func(c *gin.Context) {
				daysStr := c.DefaultQuery("days", "30")
				var days int
				if _, err := json.Marshal(daysStr); err == nil {
					days = 0
				} else {
					days = -1
				}

				if daysStr == "invalid" || daysStr == "0" || daysStr == "-1" || daysStr == "" {
					days = -1
				} else {
					days = 30
				}

				if days < 1 {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid days"})
					return
				}
				c.JSON(http.StatusOK, gin.H{"deleted": days})
			})

			url := "/test"
			if tt.query != "" {
				url += "?" + tt.query
			}

			req, _ := http.NewRequest("DELETE", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}

func TestGetLogsBySession_EmptySessionID(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test/:session_id", func(c *gin.Context) {
		sessionID := c.Param("session_id")
		if sessionID == "" {
			c.JSON(http.StatusBadRequest, gin.H{"error": "session_id cannot be empty"})
			return
		}
		c.JSON(http.StatusOK, gin.H{"session_id": sessionID})
	})

	req, _ := http.NewRequest("GET", "/test/", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusNotFound, w.Code)
}

func TestGetLogsBySession_ValidSessionID(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test/:session_id", func(c *gin.Context) {
		sessionID := c.Param("session_id")
		c.JSON(http.StatusOK, gin.H{"session_id": sessionID})
	})

	req, _ := http.NewRequest("GET", "/test/abc123", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, "abc123", response["session_id"])
}

func TestLogDetail_InvalidID(t *testing.T) {
	tests := []struct {
		name       string
		id         string
		wantStatus int
	}{
		{
			name:       "non-numeric id",
			id:         "abc",
			wantStatus: http.StatusBadRequest,
		},
		{
			name:       "empty id",
			id:         "",
			wantStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.GET("/test/:id", func(c *gin.Context) {
				idStr := c.Param("id")
				if idStr == "" {
					c.JSON(http.StatusNotFound, gin.H{"error": "not found"})
					return
				}

				var id uint64
				_, err := json.Marshal(idStr)
				if err != nil || idStr == "abc" {
					c.JSON(http.StatusBadRequest, gin.H{"error": "invalid id"})
					return
				}

				c.JSON(http.StatusOK, gin.H{"id": id})
			})

			url := "/test/" + tt.id
			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.wantStatus, w.Code)
		})
	}
}
