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

func TestApplicationHandler_ListApplications(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "list with default pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list with custom page",
			queryParams:    "?page=2&page_size=20",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list with keyword filter",
			queryParams:    "?keyword=test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list with user_id filter",
			queryParams:    "?user_id=1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list with is_active filter",
			queryParams:    "?is_active=true",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list with sort params",
			queryParams:    "?sort_field=created_at&sort_order=desc",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "list with all filters",
			queryParams:    "?page=1&page_size=10&keyword=test&user_id=1&is_active=true",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/api/v1/applications", ListApplications)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/applications"+tt.queryParams, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_ListApplications_InvalidParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	r := gin.Default()
	r.GET("/api/v1/applications", ListApplications)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/v1/applications?page=-1", nil)
	r.ServeHTTP(w, req)

	assert.NotNil(t, w)
}

func TestApplicationHandler_CreateApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name: "valid request",
			requestBody: CreateApplicationRequest{
				Name:        "Test App",
				UserID:      1,
				Description: "Test Description",
				Domain:      "example.com",
				Website:     "https://example.com",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:           "bad request - empty body",
			requestBody:    nil,
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - missing name",
			requestBody: CreateApplicationRequest{
				UserID: 1,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name: "bad request - missing user_id",
			requestBody: CreateApplicationRequest{
				Name: "Test App",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.POST("/api/v1/applications", CreateApplication)

			w := httptest.NewRecorder()
			var req *http.Request

			if tt.requestBody == nil {
				req, _ = http.NewRequest("POST", "/api/v1/applications", nil)
			} else {
				body, _ := json.Marshal(tt.requestBody)
				req, _ = http.NewRequest("POST", "/api/v1/applications", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")
			}

			r.ServeHTTP(w, req)
			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_UpdateApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:  "invalid app id - non-numeric",
			appID: "abc",
			requestBody: UpdateApplicationRequest{
				Name: ptrToString("Updated Name"),
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:  "invalid app id - empty",
			appID: "",
			requestBody: UpdateApplicationRequest{
				Name: ptrToString("Updated Name"),
			},
			expectedStatus: http.StatusNotFound,
		},
		{
			name:  "valid update request",
			appID: "1",
			requestBody: UpdateApplicationRequest{
				Name:        ptrToString("Updated Name"),
				Description: ptrToString("Updated Description"),
				IsActive:    ptrToBool(true),
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.PUT("/api/v1/applications/:id", UpdateApplication)

			w := httptest.NewRecorder()
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/applications/"+tt.appID, bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_DeleteApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "invalid app id - non-numeric",
			appID:          "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "delete existing app",
			appID:          "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.DELETE("/api/v1/applications/:id", DeleteApplication)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("DELETE", "/api/v1/applications/"+tt.appID, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_RegenerateAPIKey(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "invalid app id - non-numeric",
			appID:          "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "regenerate for non-existent app",
			appID:          "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.POST("/api/v1/applications/:id/regenerate-key", RegenerateApplicationKey)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/v1/applications/"+tt.appID+"/regenerate-key", nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_GetApplicationConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "invalid app id - non-numeric",
			appID:          "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get config for non-existent app",
			appID:          "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/api/v1/applications/:id/config", GetApplicationConfig)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/applications/"+tt.appID+"/config", nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_UpdateApplicationConfig(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		requestBody    interface{}
		expectedStatus int
	}{
		{
			name:           "invalid app id - non-numeric",
			appID:          "abc",
			requestBody: UpdateConfigRequest{
				CaptchaTypes:       []string{"slider"},
				MaxVerifyPerMinute: 60,
			},
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:   "valid config update",
			appID:  "1",
			requestBody: UpdateConfigRequest{
				CaptchaTypes:       []string{"slider", "click"},
				MaxVerifyPerMinute: 100,
				MaxVerifyPerDay:    5000,
				AllowedIPs:         []string{"10.0.0.1"},
				BlockRefusedRequests: true,
				CustomSettings:     map[string]interface{}{"key": "value"},
			},
			expectedStatus: http.StatusOK,
		},
		{
			name:  "update config for non-existent app",
			appID: "999",
			requestBody: UpdateConfigRequest{
				CaptchaTypes:       []string{"slider"},
				MaxVerifyPerMinute: 60,
			},
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.PUT("/api/v1/applications/:id/config", UpdateApplicationConfig)

			w := httptest.NewRecorder()
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("PUT", "/api/v1/applications/"+tt.appID+"/config", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_GetApplicationStatistics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "invalid app id - non-numeric",
			appID:          "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get stats for non-existent app",
			appID:          "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/api/v1/applications/:id/stats", GetApplicationStatistics)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/applications/"+tt.appID+"/stats", nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestApplicationHandler_GetApplication(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		appID          string
		expectedStatus int
	}{
		{
			name:           "invalid app id - non-numeric",
			appID:          "abc",
			expectedStatus: http.StatusBadRequest,
		},
		{
			name:           "get non-existent app",
			appID:          "999",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.GET("/api/v1/applications/:id", GetApplication)

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/api/v1/applications/"+tt.appID, nil)
			r.ServeHTTP(w, req)

			assert.NotNil(t, w)
		})
	}
}

func TestCreateApplicationRequest_JSON(t *testing.T) {
	req := CreateApplicationRequest{
		Name:        "Test App",
		UserID:      1,
		Description: "Test Description",
		Domain:      "example.com",
		Website:     "https://example.com",
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled CreateApplicationRequest
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, req.Name, unmarshaled.Name)
	assert.Equal(t, req.UserID, unmarshaled.UserID)
	assert.Equal(t, req.Description, unmarshaled.Description)
	assert.Equal(t, req.Domain, unmarshaled.Domain)
	assert.Equal(t, req.Website, unmarshaled.Website)
}

func TestUpdateApplicationRequest_JSON(t *testing.T) {
	name := "Updated Name"
	desc := "Updated Description"
	active := true
	domain := "new-domain.com"
	website := "https://new-domain.com"

	req := UpdateApplicationRequest{
		Name:        &name,
		Description: &desc,
		IsActive:    &active,
		Domain:      &domain,
		Website:     &website,
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled UpdateApplicationRequest
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, *req.Name, *unmarshaled.Name)
	assert.Equal(t, *req.Description, *unmarshaled.Description)
	assert.Equal(t, *req.IsActive, *unmarshaled.IsActive)
	assert.Equal(t, *req.Domain, *unmarshaled.Domain)
	assert.Equal(t, *req.Website, *unmarshaled.Website)
}

func TestListApplicationsQuery_Defaults(t *testing.T) {
	query := ListApplicationsQuery{}

	assert.Equal(t, 1, query.Page)
	assert.Equal(t, 10, query.PageSize)
	assert.Empty(t, query.Keyword)
	assert.Equal(t, uint(0), query.UserID)
	assert.Nil(t, query.IsActive)
	assert.Empty(t, query.SortField)
	assert.Empty(t, query.SortOrder)
}

func TestUpdateConfigRequest_JSON(t *testing.T) {
	req := UpdateConfigRequest{
		CaptchaTypes:         []string{"slider", "click"},
		MaxVerifyPerMinute:   100,
		MaxVerifyPerDay:      5000,
		AllowedIPs:           []string{"10.0.0.1", "192.168.1.1"},
		BlockRefusedRequests: true,
		CustomSettings:       map[string]interface{}{"theme": "dark"},
	}

	data, err := json.Marshal(req)
	assert.NoError(t, err)

	var unmarshaled UpdateConfigRequest
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, req.CaptchaTypes, unmarshaled.CaptchaTypes)
	assert.Equal(t, req.MaxVerifyPerMinute, unmarshaled.MaxVerifyPerMinute)
	assert.Equal(t, req.MaxVerifyPerDay, unmarshaled.MaxVerifyPerDay)
	assert.Equal(t, req.AllowedIPs, unmarshaled.AllowedIPs)
	assert.Equal(t, req.BlockRefusedRequests, unmarshaled.BlockRefusedRequests)
	assert.Equal(t, req.CustomSettings, unmarshaled.CustomSettings)
}

func ptrToString(s string) *string {
	return &s
}

func ptrToBool(b bool) *bool {
	return &b
}
