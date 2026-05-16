package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestCORS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	tests := []struct {
		name           string
		method         string
		origin         string
		expectedOrigin string
		expectedStatus int
	}{
		{
			name:           "OPTIONS request",
			method:         "OPTIONS",
			origin:         "https://example.com",
			expectedOrigin: "https://example.com",
			expectedStatus: http.StatusNoContent,
		},
		{
			name:           "GET request with origin",
			method:         "GET",
			origin:         "https://example.com",
			expectedOrigin: "https://example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "GET request without origin",
			method:         "GET",
			origin:         "",
			expectedOrigin: "*",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.Default()
			r.Use(CORS())
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "ok"})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest(tt.method, "/test", nil)
			if tt.origin != "" {
				req.Header.Set("Origin", tt.origin)
			}
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			assert.Equal(t, tt.expectedOrigin, w.Header().Get("Access-Control-Allow-Origin"))
			assert.Equal(t, "true", w.Header().Get("Access-Control-Allow-Credentials"))
			assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Methods"))
			assert.NotEmpty(t, w.Header().Get("Access-Control-Allow-Headers"))
			assert.NotEmpty(t, w.Header().Get("Access-Control-Max-Age"))
		})
	}
}
