package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SecurityHeadersMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
	assert.NotEmpty(t, w.Header().Get("Strict-Transport-Security"))
	assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, w.Header().Get("X-XSS-Protection"))
	assert.NotEmpty(t, w.Header().Get("Referrer-Policy"))
}

func TestInputValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(InputValidationMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test?param=value", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusOK, w.Code)

	w = httptest.NewRecorder()
	req, _ = http.NewRequest("GET", "/test?param=<script>alert(1)</script>", nil)
	router.ServeHTTP(w, req)
	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestFingerprintMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(FingerprintMiddleware())

	var capturedFingerprint string
	router.GET("/test", func(c *gin.Context) {
		fp, _ := ExtractFingerprintFromContext(c)
		if fp != nil {
			capturedFingerprint = fp.FingerprintID
		}
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("User-Agent", "test-agent")
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, capturedFingerprint)
}

func TestSmartRateLimitMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(SmartRateLimitMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	for i := 0; i < 5; i++ {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.1:12345"
		router.ServeHTTP(w, req)
		if i < 5 {
			assert.Equal(t, http.StatusOK, w.Code)
		} else {
			assert.Equal(t, http.StatusTooManyRequests, w.Code)
		}
	}
}

func TestComprehensiveSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(ComprehensiveSecurityMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "test")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
}

func TestOWASPTop10Middleware(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(OWASPTop10SecurityMiddleware())

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "secure")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
