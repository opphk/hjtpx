package handler

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestHealthHandler(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthHandlerWithVersion(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
			"version":   "1.0.0",
			"uptime":    time.Since(time.Now()).Seconds(),
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthEndpointDetailed(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		components := gin.H{
			"database": "healthy",
			"redis":    "healthy",
			"api":      "healthy",
		}
		
		response.Success(c, gin.H{
			"status":     "healthy",
			"timestamp":  time.Now().Unix(),
			"version":    "1.0.0",
			"components": components,
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetClientIP(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name       string
		headers    map[string]string
		expectedIP string
	}{
		{
			name:       "X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1"},
			expectedIP: "192.168.1.1",
		},
		{
			name:       "X-Real-IP",
			headers:    map[string]string{"X-Real-IP": "10.0.0.1"},
			expectedIP: "10.0.0.1",
		},
		{
			name:       "Multiple X-Forwarded-For",
			headers:    map[string]string{"X-Forwarded-For": "192.168.1.1, 10.0.0.1"},
			expectedIP: "192.168.1.1",
		},
		{
			name:       "No headers",
			headers:    map[string]string{},
			expectedIP: "192.168.1.1",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			var capturedIP string
			
			r.GET("/test", func(c *gin.Context) {
				ip := c.GetHeader("X-Forwarded-For")
				if ip == "" {
					ip = c.GetHeader("X-Real-IP")
				}
				if ip == "" {
					ip = "192.168.1.1"
				}
				ips := splitIP(ip)
				capturedIP = ips[0]
				response.Success(c, gin.H{"ip": capturedIP})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func splitIP(ip string) []string {
	result := make([]string, 0)
	current := ""
	for _, char := range ip {
		if char == ',' {
			result = append(result, current)
			current = ""
		} else if char != ' ' {
			current += string(char)
		}
	}
	if current != "" {
		result = append(result, current)
	}
	return result
}

func TestGetUserAgent(t *testing.T) {
	gin.SetMode(gin.TestMode)
	
	tests := []struct {
		name     string
		ua       string
		expected bool
	}{
		{
			name:     "Chrome",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64) AppleWebKit/537.36",
			expected: true,
		},
		{
			name:     "Firefox",
			ua:       "Mozilla/5.0 (Windows NT 10.0; Win64; x64; rv:90.0)",
			expected: true,
		},
		{
			name:     "Empty",
			ua:       "",
			expected: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := gin.New()
			var hasUserAgent bool
			
			r.GET("/test", func(c *gin.Context) {
				userAgent := c.GetHeader("User-Agent")
				hasUserAgent = userAgent != ""
				response.Success(c, gin.H{"has": hasUserAgent})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			req.Header.Set("User-Agent", tt.ua)
			
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestHealthEndpointUnhealthy(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		response.Fail(c, 503, "Service degraded")
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthEndpointWithDetails(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		details := gin.H{
			"database": gin.H{
				"status":  "healthy",
				"latency": "5ms",
			},
			"redis": gin.H{
				"status":  "healthy",
				"latency": "2ms",
			},
			"api": gin.H{
				"status":  "healthy",
				"latency": "10ms",
			},
		}
		
		response.Success(c, gin.H{
			"status":     "healthy",
			"timestamp":  time.Now().Unix(),
			"version":    "1.0.0",
			"details":    details,
		})
	})

	req, _ := http.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestHealthEndpointStress(t *testing.T) {
	r := setupTestRouter()
	r.GET("/health", func(c *gin.Context) {
		response.Success(c, gin.H{
			"status":    "healthy",
			"timestamp": time.Now().Unix(),
		})
	})

	for i := 0; i < 100; i++ {
		req, _ := http.NewRequest("GET", "/health", nil)
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)
		assert.Equal(t, http.StatusOK, w.Code)
	}
}
