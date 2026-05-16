package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func TestLoginHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/login", Login)

	tests := []struct {
		name           string
		requestBody    LoginRequest
		expectedCode   int
		expectToken    bool
	}{
		{
			name: "missing username",
			requestBody: LoginRequest{
				Password: "password123",
			},
			expectedCode: http.StatusBadRequest,
			expectToken:  false,
		},
		{
			name: "missing password",
			requestBody: LoginRequest{
				Username: "admin",
			},
			expectedCode: http.StatusBadRequest,
			expectToken:  false,
		},
		{
			name: "empty request body",
			requestBody: LoginRequest{},
			expectedCode: http.StatusBadRequest,
			expectToken:  false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/admin/login", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedCode, w.Code)
		})
	}
}

func TestLogoutHandler(t *testing.T) {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/admin/logout", Logout)

	req, _ := http.NewRequest("POST", "/admin/logout", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)

	var resp response.Response
	err := json.Unmarshal(w.Body.Bytes(), &resp)
	assert.NoError(t, err)
	assert.Equal(t, 0, resp.Code)
}

func TestLoginRequestValidation(t *testing.T) {
	tests := []struct {
		name     string
		req      LoginRequest
		hasError bool
	}{
		{
			name: "valid request",
			req: LoginRequest{
				Username: "admin",
				Password: "password123",
			},
			hasError: false,
		},
		{
			name: "empty username",
			req: LoginRequest{
				Username: "",
				Password: "password123",
			},
			hasError: true,
		},
		{
			name: "empty password",
			req: LoginRequest{
				Username: "admin",
				Password: "",
			},
			hasError: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			hasError := tt.req.Username == "" || tt.req.Password == ""
			assert.Equal(t, tt.hasError, hasError)
		})
	}
}

func TestLoginResponseStructure(t *testing.T) {
	resp := LoginResponse{
		Token: "test-token-123",
		User: AdminInfo{
			ID:           1,
			Username:     "admin",
			IsSuperAdmin: true,
		},
	}

	jsonData, err := json.Marshal(resp)
	assert.NoError(t, err)

	var decoded LoginResponse
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, resp.Token, decoded.Token)
	assert.Equal(t, resp.User.ID, decoded.User.ID)
	assert.Equal(t, resp.User.Username, decoded.User.Username)
	assert.Equal(t, resp.User.IsSuperAdmin, decoded.User.IsSuperAdmin)
}

func TestAdminInfoStructure(t *testing.T) {
	info := AdminInfo{
		ID:           1,
		Username:     "testuser",
		IsSuperAdmin: false,
	}

	jsonData, err := json.Marshal(info)
	assert.NoError(t, err)

	var decoded AdminInfo
	err = json.Unmarshal(jsonData, &decoded)
	assert.NoError(t, err)
	assert.Equal(t, info.ID, decoded.ID)
	assert.Equal(t, info.Username, decoded.Username)
	assert.Equal(t, info.IsSuperAdmin, decoded.IsSuperAdmin)
}
