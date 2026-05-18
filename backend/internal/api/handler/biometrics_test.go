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

func TestRegisterBiometricProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - register with user_id",
			requestBody: map[string]interface{}{
				"user_id": "user123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - missing user_id",
			requestBody: map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/biometrics/register", RegisterBiometricProfile)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/biometrics/register", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestVerifyBiometrics(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		requestBody    map[string]interface{}
		expectedStatus int
	}{
		{
			name: "success - verify with user_id",
			requestBody: map[string]interface{}{
				"user_id": "user123",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "error - missing user_id",
			requestBody: map[string]interface{}{},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.POST("/api/v1/biometrics/verify", VerifyBiometrics)

			body, _ := json.Marshal(tt.requestBody)
			req, _ := http.NewRequest("POST", "/api/v1/biometrics/verify", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")

			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestGetBiometricProfile(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		userID         string
		expectedStatus int
	}{
		{
			name:           "success - get profile with user_id",
			userID:         "user123",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "error - missing user_id",
			userID:         "",
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			r.GET("/api/v1/biometrics/profile", GetBiometricProfile)

			url := "/api/v1/biometrics/profile"
			if tt.userID != "" {
				url += "?user_id=" + tt.userID
			}

			req, _ := http.NewRequest("GET", url, nil)
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestBiometricsHandlerStructures(t *testing.T) {
	t.Run("RegisterBiometricProfileRequest marshaling", func(t *testing.T) {
		req := RegisterBiometricProfileRequest{
			UserID: "user123",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var unmarshaled RegisterBiometricProfileRequest
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, req.UserID, unmarshaled.UserID)
	})

	t.Run("VerifyBiometricsRequest marshaling", func(t *testing.T) {
		req := VerifyBiometricsRequest{
			UserID: "user123",
		}

		data, err := json.Marshal(req)
		assert.NoError(t, err)
		assert.NotEmpty(t, data)

		var unmarshaled VerifyBiometricsRequest
		err = json.Unmarshal(data, &unmarshaled)
		assert.NoError(t, err)
		assert.Equal(t, req.UserID, unmarshaled.UserID)
	})

	t.Run("BiometricsHandler instantiation", func(t *testing.T) {
		handler := NewBiometricsHandler()
		assert.NotNil(t, handler)
		assert.NotNil(t, handler.biometricsService)
	})

	t.Run("GetBiometricsHandler singleton", func(t *testing.T) {
		handler1 := GetBiometricsHandler()
		handler2 := GetBiometricsHandler()
		assert.NotNil(t, handler1)
		assert.NotNil(t, handler2)
		assert.Equal(t, handler1, handler2)
	})
}

func TestBiometricsEndpoints(t *testing.T) {
	gin.SetMode(gin.TestMode)

	t.Run("concurrent registration requests", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/v1/biometrics/register", RegisterBiometricProfile)

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				body, _ := json.Marshal(map[string]interface{}{
					"user_id": "user" + string(rune('0'+index)),
				})
				req, _ := http.NewRequest("POST", "/api/v1/biometrics/register", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code == http.StatusOK || w.Code == http.StatusBadRequest {
					done <- true
				} else {
					done <- false
				}
			}(i)
		}

		successCount := 0
		for i := 0; i < 10; i++ {
			if <-done {
				successCount++
			}
		}

		assert.GreaterOrEqual(t, successCount, 8)
	})

	t.Run("concurrent verification requests", func(t *testing.T) {
		r := gin.New()
		r.POST("/api/v1/biometrics/verify", VerifyBiometrics)

		done := make(chan bool, 10)

		for i := 0; i < 10; i++ {
			go func(index int) {
				body, _ := json.Marshal(map[string]interface{}{
					"user_id": "user" + string(rune('0'+index)),
				})
				req, _ := http.NewRequest("POST", "/api/v1/biometrics/verify", bytes.NewBuffer(body))
				req.Header.Set("Content-Type", "application/json")

				w := httptest.NewRecorder()
				r.ServeHTTP(w, req)

				if w.Code == http.StatusOK || w.Code == http.StatusBadRequest {
					done <- true
				} else {
					done <- false
				}
			}(i)
		}

		successCount := 0
		for i := 0; i < 10; i++ {
			if <-done {
				successCount++
			}
		}

		assert.GreaterOrEqual(t, successCount, 8)
	})
}
