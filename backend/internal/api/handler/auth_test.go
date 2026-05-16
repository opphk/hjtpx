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

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestLoginRequest_Validation(t *testing.T) {
	tests := []struct {
		name    string
		request LoginRequest
		wantErr bool
	}{
		{
			name: "valid request",
			request: LoginRequest{
				Username: "admin",
				Password: "password123",
			},
			wantErr: false,
		},
		{
			name: "missing username",
			request: LoginRequest{
				Password: "password123",
			},
			wantErr: true,
		},
		{
			name: "missing password",
			request: LoginRequest{
				Username: "admin",
			},
			wantErr: true,
		},
		{
			name:    "empty request",
			request: LoginRequest{},
			wantErr: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/test", func(c *gin.Context) {
				var req LoginRequest
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

func TestLogout_BearerToken(t *testing.T) {
	tests := []struct {
		name           string
		authHeader     string
		expectedStatus int
	}{
		{
			name:           "valid bearer token",
			authHeader:     "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9.test",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "empty auth header",
			authHeader:     "",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "invalid format",
			authHeader:     "InvalidToken",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.POST("/logout", Logout)

			req, _ := http.NewRequest("POST", "/logout", nil)
			if tt.authHeader != "" {
				req.Header.Set("Authorization", tt.authHeader)
			}

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestLoginResponse_JSON(t *testing.T) {
	response := LoginResponse{
		Token: "test-token",
		User: AdminInfo{
			ID:           1,
			Username:     "admin",
			IsSuperAdmin: true,
		},
	}

	data, err := json.Marshal(response)
	assert.NoError(t, err)

	var unmarshaled LoginResponse
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, response.Token, unmarshaled.Token)
	assert.Equal(t, response.User.ID, unmarshaled.User.ID)
	assert.Equal(t, response.User.Username, unmarshaled.User.Username)
	assert.Equal(t, response.User.IsSuperAdmin, unmarshaled.User.IsSuperAdmin)
}

func TestAdminInfo_JSON(t *testing.T) {
	info := AdminInfo{
		ID:           1,
		Username:     "testadmin",
		IsSuperAdmin: false,
	}

	data, err := json.Marshal(info)
	assert.NoError(t, err)

	var unmarshaled AdminInfo
	err = json.Unmarshal(data, &unmarshaled)
	assert.NoError(t, err)

	assert.Equal(t, info.ID, unmarshaled.ID)
	assert.Equal(t, info.Username, unmarshaled.Username)
	assert.Equal(t, info.IsSuperAdmin, unmarshaled.IsSuperAdmin)
}
