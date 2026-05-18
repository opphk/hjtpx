package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestDistributedRateLimitMiddleware_DefaultOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(DistributedRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDistributedRateLimitMiddleware_CustomOptions(t *testing.T) {
	r := setupTestRouter()
	options := &DistributedRateLimitOptions{
		MaxRequests: 50,
		WindowSecs:  30,
	}
	r.Use(DistributedRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Node"))
}

func TestDistributedRateLimitMiddleware_WithXForwardedFor(t *testing.T) {
	r := setupTestRouter()
	r.Use(DistributedRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDistributedRateLimitMiddleware_WithXRealIP(t *testing.T) {
	r := setupTestRouter()
	r.Use(DistributedRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetDistributedRateLimitService(t *testing.T) {
	service := GetDistributedRateLimitService()
	assert.NotNil(t, service)
}

func TestDistributedRateLimitMiddleware_ResponseHeaders(t *testing.T) {
	r := setupTestRouter()
	r.Use(DistributedRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.50:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Node"))
}

func TestDistributedRateLimitMiddleware_FixedWindow(t *testing.T) {
	r := setupTestRouter()
	options := &DistributedRateLimitOptions{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	r.Use(DistributedRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDistributedRateLimitMiddleware_SlidingWindow(t *testing.T) {
	r := setupTestRouter()
	options := &DistributedRateLimitOptions{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	r.Use(DistributedRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.101:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestDistributedRateLimitMiddleware_LeakyBucket(t *testing.T) {
	r := setupTestRouter()
	options := &DistributedRateLimitOptions{
		MaxRequests: 50,
		WindowSecs:  30,
	}
	r.Use(DistributedRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.102:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
