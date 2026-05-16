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

func TestCreateBlacklistRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateBlacklistRequest
		wantErr bool
	}{
		{
			name: "valid request with all fields",
			request: CreateBlacklistRequest{
				Type:       "ip",
				Target:     "192.168.1.100",
				Reason:     "malicious activity",
				Action:     "block",
				Expiration: "2025-12-31",
			},
			wantErr: false,
		},
		{
			name: "valid request with minimal fields",
			request: CreateBlacklistRequest{
				Type:   "ip",
				Target: "10.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			request: CreateBlacklistRequest{
				Target: "192.168.1.100",
			},
			wantErr: true,
		},
		{
			name: "missing target",
			request: CreateBlacklistRequest{
				Type: "ip",
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			request: CreateBlacklistRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req CreateBlacklistRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			body, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

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

func TestUpdateBlacklistRequest_Validation(t *testing.T) {
	reason := "updated reason"
	action := "captcha"

	tests := []struct {
		name    string
		request UpdateBlacklistRequest
		wantErr bool
	}{
		{
			name: "valid request with all fields",
			request: UpdateBlacklistRequest{
				Reason: &reason,
				Action: &action,
			},
			wantErr: false,
		},
		{
			name:    "empty request is valid",
			request: UpdateBlacklistRequest{},
			wantErr: false,
		},
		{
			name: "valid request with partial fields",
			request: UpdateBlacklistRequest{
				Reason: &reason,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.PUT("/test", func(c *gin.Context) {
				var req UpdateBlacklistRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			body, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("PUT", "/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

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

func TestListBlacklistQuery_Defaults(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ListBlacklistQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":    query.Page,
			"size":    query.Size,
			"type":    query.Type,
			"status":  query.Status,
			"keyword": query.Keyword,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(20), response["size"])
}

func TestListBlacklistQuery_WithParams(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ListBlacklistQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":       query.Page,
			"size":       query.Size,
			"type":       query.Type,
			"status":     query.Status,
			"keyword":    query.Keyword,
			"start_date": query.StartDate,
			"end_date":   query.EndDate,
		})
	})

	req, _ := http.NewRequest("GET", "/test?page=2&size=50&type=ip&status=active&keyword=test&start_date=2025-01-01&end_date=2025-12-31", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(2), response["page"])
	assert.Equal(t, float64(50), response["size"])
	assert.Equal(t, "ip", response["type"])
	assert.Equal(t, "active", response["status"])
	assert.Equal(t, "test", response["keyword"])
}

func TestImportBlacklistRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request ImportBlacklistRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: ImportBlacklistRequest{
				Type:    "ip",
				Targets: []string{"192.168.1.1", "192.168.1.2"},
				Reason:  "bulk import",
			},
			wantErr: false,
		},
		{
			name: "missing type",
			request: ImportBlacklistRequest{
				Targets: []string{"192.168.1.1"},
			},
			wantErr: true,
		},
		{
			name: "missing targets",
			request: ImportBlacklistRequest{
				Type: "ip",
			},
			wantErr: true,
		},
		{
			name: "empty targets list",
			request: ImportBlacklistRequest{
				Type:    "ip",
				Targets: []string{},
			},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req ImportBlacklistRequest
				if err := c.ShouldBindJSON(&req); err != nil {
					c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
					return
				}
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			body, _ := json.Marshal(tt.request)
			req, _ := http.NewRequest("POST", "/test", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

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

func TestBlacklistHandler_GetBlacklistHandler(t *testing.T) {
	handler1 := GetBlacklistHandler()
	handler2 := GetBlacklistHandler()

	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
	assert.Equal(t, handler1, handler2)
}
