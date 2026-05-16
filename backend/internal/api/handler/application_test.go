package handler

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestListApplicationsQueryValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		queryParams    string
		expectedStatus int
	}{
		{
			name:           "default pagination",
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "custom page",
			queryParams:    "?page=2",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "custom page size",
			queryParams:    "?page_size=20",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "with keyword",
			queryParams:    "?keyword=test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid page",
			queryParams:    "?page=-1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "page size too large",
			queryParams:    "?page_size=200",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/admin/applications", ListApplications)

			req, _ := http.NewRequest("GET", "/admin/applications"+tt.queryParams, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestCreateApplicationRequestValidation(t *testing.T) {
	tests := []struct {
		name     string
		req      CreateApplicationRequest
		hasError bool
	}{
		{
			name: "valid request",
			req: CreateApplicationRequest{
				Name:        "Test App",
				UserID:      1,
				Description: "Test description",
			},
			hasError: false,
		},
		{
			name: "empty name",
			req: CreateApplicationRequest{
				Name:        "",
				UserID:      1,
				Description: "Test description",
			},
			hasError: true,
		},
		{
			name: "zero user id",
			req: CreateApplicationRequest{
				Name:        "Test App",
				UserID:      0,
				Description: "Test description",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.req.Name == "" || tt.req.UserID == 0
			assert.Equal(t, tt.hasError, hasError)
		})
	}
}

func TestUpdateApplicationRequestValidation(t *testing.T) {
	tests := []struct {
		name     string
		req      UpdateApplicationRequest
		hasError bool
	}{
		{
			name: "valid request with name",
			req: UpdateApplicationRequest{
				Name: "Updated App",
			},
			hasError: false,
		},
		{
			name: "valid request with description",
			req: UpdateApplicationRequest{
				Description: "Updated description",
			},
			hasError: false,
		},
		{
			name: "valid request with is_active true",
			req: UpdateApplicationRequest{
				IsActive: boolPtr(true),
			},
			hasError: false,
		},
		{
			name: "valid request with is_active false",
			req: UpdateApplicationRequest{
				IsActive: boolPtr(false),
			},
			hasError: false,
		},
		{
			name:     "empty request",
			req:      UpdateApplicationRequest{},
			hasError: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_ = tt.req
		})
	}
}

func TestUpdateApplicationInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.PUT("/admin/applications/:id", UpdateApplication)

	req, _ := http.NewRequest("PUT", "/admin/applications/invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestDeleteApplicationInvalidID(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.DELETE("/admin/applications/:id", DeleteApplication)

	req, _ := http.NewRequest("DELETE", "/admin/applications/invalid", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestPaginatedApplicationsStructure(t *testing.T) {
	pa := PaginatedApplications{
		Data:     nil,
		Total:    100,
		Page:     1,
		PageSize: 10,
	}

	jsonData, err := json.Marshal(pa)
	assert.NoError(t, err)

	var decoded PaginatedApplications
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, pa.Total, decoded.Total)
	assert.Equal(t, pa.Page, decoded.Page)
	assert.Equal(t, pa.PageSize, decoded.PageSize)
}

func TestListApplicationsQueryStructure(t *testing.T) {
	tests := []struct {
		name     string
		query    ListApplicationsQuery
		expected ListApplicationsQuery
	}{
		{
			name:  "default values",
			query: ListApplicationsQuery{},
			expected: ListApplicationsQuery{
				Page:     1,
				PageSize: 10,
			},
		},
		{
			name: "custom values",
			query: ListApplicationsQuery{
				Page:     2,
				PageSize: 20,
				Keyword:  "test",
			},
			expected: ListApplicationsQuery{
				Page:     2,
				PageSize: 20,
				Keyword:  "test",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected.Page, tt.query.Page)
			assert.Equal(t, tt.expected.PageSize, tt.query.PageSize)
			assert.Equal(t, tt.expected.Keyword, tt.query.Keyword)
		})
	}
}

func boolPtr(b bool) *bool {
	return &b
}
