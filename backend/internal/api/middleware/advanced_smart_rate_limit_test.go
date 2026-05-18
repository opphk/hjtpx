package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestAdaptiveRateLimitMiddleware_DefaultOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(AdaptiveRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdaptiveRateLimitMiddleware_CustomOptions(t *testing.T) {
	r := setupTestRouter()
	config := service.AdaptiveRateLimitConfig{
		BaseLimit:      100,
		PeakLimit:      200,
		OffPeakLimit:   500,
		OffPeakStart:   0,
		OffPeakEnd:     6,
		EnableDynamic:  true,
		CooldownPeriod: 60 * time.Second,
	}
	r.Use(AdaptiveRateLimitMiddleware(&config))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-LoadLevel"))
}

func TestAdaptiveRateLimitMiddleware_WithXForwardedFor(t *testing.T) {
	r := setupTestRouter()
	r.Use(AdaptiveRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdaptiveRateLimitMiddleware_WithXRealIP(t *testing.T) {
	r := setupTestRouter()
	r.Use(AdaptiveRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetAdaptiveRateLimitService(t *testing.T) {
	service := GetAdaptiveRateLimitService()
	assert.NotNil(t, service)
}

func TestAdaptiveRateLimitMiddleware_ResponseHeaders(t *testing.T) {
	r := setupTestRouter()
	r.Use(AdaptiveRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Rate"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-LoadLevel"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-LoadFactor"))
}
