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

func TestCreateApplicationRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateApplicationRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: CreateApplicationRequest{
				Name:        "Test App",
				UserID:      1,
				Description: "Test description",
				Domain:      "test.com",
				Website:     "https://test.com",
			},
			wantErr: false,
		},
		{
			name: "missing name",
			request: CreateApplicationRequest{
				UserID: 1,
			},
			wantErr: true,
		},
		{
			name: "missing user_id",
			request: CreateApplicationRequest{
				Name: "Test App",
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			request: CreateApplicationRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req CreateApplicationRequest
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

func TestUpdateApplicationRequest_Validation(t *testing.T) {
	name := "Updated Name"
	isActive := true

	tests := []struct {
		name    string
		request UpdateApplicationRequest
		wantErr bool
	}{
		{
			name: "valid request with all fields",
			request: UpdateApplicationRequest{
				Name:        &name,
				IsActive:    &isActive,
				Description: nil,
			},
			wantErr: false,
		},
		{
			name:    "empty request is valid",
			request: UpdateApplicationRequest{},
			wantErr: false,
		},
		{
			name: "valid request with partial fields",
			request: UpdateApplicationRequest{
				IsActive: &isActive,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.PUT("/test", func(c *gin.Context) {
				var req UpdateApplicationRequest
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

func TestListApplicationsQuery_Defaults(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ListApplicationsQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":       query.Page,
			"page_size":  query.PageSize,
			"keyword":    query.Keyword,
			"sort_field": query.SortField,
			"sort_order": query.SortOrder,
		})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(1), response["page"])
	assert.Equal(t, float64(10), response["page_size"])
}

func TestListApplicationsQuery_WithParams(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ListApplicationsQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":       query.Page,
			"page_size":  query.PageSize,
			"keyword":    query.Keyword,
			"user_id":    query.UserID,
			"is_active":  query.IsActive,
			"sort_field": query.SortField,
			"sort_order": query.SortOrder,
		})
	})

	req, _ := http.NewRequest("GET", "/test?page=2&page_size=50&keyword=test&user_id=1&is_active=true&sort_field=name&sort_order=asc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(2), response["page"])
	assert.Equal(t, float64(50), response["page_size"])
	assert.Equal(t, "test", response["keyword"])
	assert.Equal(t, float64(1), response["user_id"])
	assert.Equal(t, "name", response["sort_field"])
	assert.Equal(t, "asc", response["sort_order"])
}

func TestUpdateConfigRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request UpdateConfigRequest
		wantErr bool
	}{
		{
			name: "valid request with all fields",
			request: UpdateConfigRequest{
				CaptchaTypes:         []string{"slider", "click"},
				MaxVerifyPerMinute:   10,
				MaxVerifyPerDay:      100,
				AllowedIPs:           []string{"192.168.1.1"},
				BlockRefusedRequests: true,
				CustomSettings:       map[string]interface{}{"key": "value"},
			},
			wantErr: false,
		},
		{
			name: "valid request with minimal fields",
			request: UpdateConfigRequest{
				CaptchaTypes: []string{},
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.PUT("/test", func(c *gin.Context) {
				var req UpdateConfigRequest
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

func TestApplicationHandler_NewApplicationHandler(t *testing.T) {
	handler := NewApplicationHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.applicationService)
}

func TestApplicationHandler_GetApplicationHandler(t *testing.T) {
	handler1 := GetApplicationHandler()
	handler2 := GetApplicationHandler()

	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
	assert.Equal(t, handler1, handler2)
}
