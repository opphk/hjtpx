package response

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSuccess(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name string
		data interface{}
	}{
		{
			name: "success with data",
			data: map[string]string{"key": "value"},
		},
		{
			name: "success with nil data",
			data: nil,
		},
		{
			name: "success with string",
			data: "test data",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Success(c, tt.data)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name       string
		httpStatus int
		message    string
	}{
		{
			name:       "400 bad request",
			httpStatus: http.StatusBadRequest,
			message:    "bad request",
		},
		{
			name:       "500 internal server error",
			httpStatus: http.StatusInternalServerError,
			message:    "internal error",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			Error(c, tt.httpStatus, tt.message)

			assert.Equal(t, tt.httpStatus, w.Code)
		})
	}
}

func TestBadRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	BadRequest(c, "bad request")

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestUnauthorized(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Unauthorized(c)

	assert.Equal(t, http.StatusUnauthorized, w.Code)
}

func TestForbidden(t *testing.T) {
	gin.SetMode(gin.TestMode)

	w := httptest.NewRecorder()
	c, _ := gin.CreateTestContext(w)

	Forbidden(c)

	assert.Equal(t, http.StatusForbidden, w.Code)
}

func TestNotFound(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "not found with message",
			message: "not found",
		},
		{
			name:    "not found with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			NotFound(c, tt.message)

			assert.Equal(t, http.StatusNotFound, w.Code)
		})
	}
}

func TestInternalServerError(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "internal server error with message",
			message: "error",
		},
		{
			name:    "internal server error with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			InternalServerError(c, tt.message)

			assert.Equal(t, http.StatusInternalServerError, w.Code)
		})
	}
}

func TestTooManyRequests(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name    string
		message string
	}{
		{
			name:    "too many requests with message",
			message: "too many",
		},
		{
			name:    "too many requests with empty message",
			message: "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)

			TooManyRequests(c, tt.message)

			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		})
	}
}
