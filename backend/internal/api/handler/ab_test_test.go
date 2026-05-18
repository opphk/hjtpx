package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestNewABTestHandler(t *testing.T) {
	handler := NewABTestHandler()
	assert.NotNil(t, handler)
	assert.NotNil(t, handler.abTestService)
}

func TestGetABTestHandler(t *testing.T) {
	handler1 := GetABTestHandler()
	handler2 := GetABTestHandler()
	assert.NotNil(t, handler1)
	assert.NotNil(t, handler2)
}

func TestCreateABTestRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request CreateABTestRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: CreateABTestRequest{
				Name:          "Test A/B",
				Description:   "Test description",
				ApplicationID: 1,
				Variants: []CreateVariantRequest{
					{Name: "Control", IsControl: true, TrafficPercent: 50},
					{Name: "Variant", IsControl: false, TrafficPercent: 50},
				},
			},
			wantErr: false,
		},
		{
			name: "missing name",
			request: CreateABTestRequest{
				ApplicationID: 1,
				Variants: []CreateVariantRequest{
					{Name: "Control", TrafficPercent: 50},
					{Name: "Variant", TrafficPercent: 50},
				},
			},
			wantErr: true,
		},
		{
			name: "missing application_id",
			request: CreateABTestRequest{
				Name: "Test A/B",
				Variants: []CreateVariantRequest{
					{Name: "Control", TrafficPercent: 50},
					{Name: "Variant", TrafficPercent: 50},
				},
			},
			wantErr: true,
		},
		{
			name: "not enough variants",
			request: CreateABTestRequest{
				Name:          "Test A/B",
				ApplicationID: 1,
				Variants:      []CreateVariantRequest{},
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			request: CreateABTestRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req CreateABTestRequest
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

func TestUpdateABTestRequest_Validation(t *testing.T) {
	name := "Updated Name"
	desc := "Updated Description"
	config := map[string]interface{}{"key": "value"}

	tests := []struct {
		name    string
		request UpdateABTestRequest
		wantErr bool
	}{
		{
			name: "valid request with all fields",
			request: UpdateABTestRequest{
				Name:        &name,
				Description: &desc,
				Config:      &config,
			},
			wantErr: false,
		},
		{
			name:    "empty request is valid",
			request: UpdateABTestRequest{},
			wantErr: false,
		},
		{
			name: "valid request with partial fields",
			request: UpdateABTestRequest{
				Description: &desc,
			},
			wantErr: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.PUT("/test", func(c *gin.Context) {
				var req UpdateABTestRequest
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

func TestListABTestsQuery_Defaults(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ListABTestsQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":       query.Page,
			"page_size":  query.PageSize,
			"keyword":    query.Keyword,
			"status":     query.Status,
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

func TestListABTestsQuery_WithParams(t *testing.T) {
	r := setupTestRouter()
	r.GET("/test", func(c *gin.Context) {
		var query ListABTestsQuery
		if err := c.ShouldBindQuery(&query); err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
			return
		}
		c.JSON(http.StatusOK, gin.H{
			"page":           query.Page,
			"page_size":      query.PageSize,
			"keyword":        query.Keyword,
			"application_id": query.ApplicationID,
			"status":         query.Status,
			"sort_field":     query.SortField,
			"sort_order":     query.SortOrder,
		})
	})

	req, _ := http.NewRequest("GET", "/test?page=2&page_size=20&keyword=test&application_id=1&status=running&sort_field=name&sort_order=asc", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var response map[string]interface{}
	json.Unmarshal(w.Body.Bytes(), &response)

	assert.Equal(t, float64(2), response["page"])
	assert.Equal(t, float64(20), response["page_size"])
	assert.Equal(t, "test", response["keyword"])
	assert.Equal(t, float64(1), response["application_id"])
	assert.Equal(t, "running", response["status"])
	assert.Equal(t, "name", response["sort_field"])
	assert.Equal(t, "asc", response["sort_order"])
}

func TestAssignVariantRequest_Validation(t *testing.T) {
	userID := uint(123)
	tests := []struct {
		name    string
		request service.AssignVariantRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: service.AssignVariantRequest{
				TestID:    1,
				SessionID: "session-123",
				UserID:    &userID,
				DeviceID:  "device-456",
			},
			wantErr: false,
		},
		{
			name: "missing test_id",
			request: service.AssignVariantRequest{
				SessionID: "session-123",
			},
			wantErr: true,
		},
		{
			name: "missing session_id",
			request: service.AssignVariantRequest{
				TestID: 1,
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			request: service.AssignVariantRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req service.AssignVariantRequest
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

func TestTrackEventRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request service.TrackEventRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: service.TrackEventRequest{
				TestID:       1,
				VariantID:    1,
				SessionID:    "session-123",
				EventName:    "conversion",
				EventType:    "purchase",
				IsConversion: true,
				Value:        99.99,
			},
			wantErr: false,
		},
		{
			name: "missing test_id",
			request: service.TrackEventRequest{
				VariantID: 1,
				SessionID: "session-123",
				EventName: "conversion",
			},
			wantErr: true,
		},
		{
			name: "missing variant_id",
			request: service.TrackEventRequest{
				TestID:    1,
				SessionID: "session-123",
				EventName: "conversion",
			},
			wantErr: true,
		},
		{
			name: "missing event_name",
			request: service.TrackEventRequest{
				TestID:    1,
				VariantID: 1,
				SessionID: "session-123",
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			request: service.TrackEventRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req service.TrackEventRequest
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

func TestCreateVariantRequest_JSON(t *testing.T) {
	config := map[string]interface{}{"color": "blue"}
	request := CreateVariantRequest{
		Name:           "Test Variant",
		IsControl:      true,
		TrafficPercent: 50,
		Config:         config,
		Description:    "Test variant description",
	}

	data, err := json.Marshal(request)
	assert.NoError(t, err)

	var unmarshaled CreateVariantRequest
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, request.Name, unmarshaled.Name)
	assert.Equal(t, request.IsControl, unmarshaled.IsControl)
	assert.Equal(t, request.TrafficPercent, unmarshaled.TrafficPercent)
	assert.Equal(t, request.Description, unmarshaled.Description)
}
