package middleware

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/stretchr/testify/assert"
)

func TestIPRateLimitMiddleware_DefaultOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(IPRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRateLimitMiddleware_CustomOptions(t *testing.T) {
	r := setupTestRouter()
	options := &RateLimitOptions{
		MaxRequests: 50,
		WindowSecs:  30,
	}
	r.Use(IPRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "50", w.Header().Get("X-RateLimit-Limit"))
}

func TestIPRateLimitMiddleware_WithXForwardedFor(t *testing.T) {
	r := setupTestRouter()
	r.Use(IPRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIPRateLimitMiddleware_WithXRealIP(t *testing.T) {
	r := setupTestRouter()
	r.Use(IPRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Real-IP", "10.0.0.2")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserRateLimitMiddleware_NoUserID(t *testing.T) {
	r := setupTestRouter()
	r.Use(UserRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestUserRateLimitMiddleware_WithUserID(t *testing.T) {
	r := setupTestRouter()
	options := &RateLimitOptions{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	r.Use(UserRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", uint(123))
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAppRateLimitMiddleware_NoAppID(t *testing.T) {
	r := setupTestRouter()
	r.Use(AppRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAppRateLimitMiddleware_WithValidAppID(t *testing.T) {
	r := setupTestRouter()
	options := &RateLimitOptions{
		MaxRequests: 500,
		WindowSecs:  60,
	}
	r.Use(AppRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-App-ID", "12345")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.Equal(t, "500", w.Header().Get("X-RateLimit-Limit"))
}

func TestAppRateLimitMiddleware_InvalidAppID(t *testing.T) {
	r := setupTestRouter()
	r.Use(AppRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-App-ID", "invalid")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCombinedRateLimitMiddleware(t *testing.T) {
	r := setupTestRouter()
	ipOptions := &RateLimitOptions{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	userOptions := &RateLimitOptions{
		MaxRequests: 200,
		WindowSecs:  60,
	}
	appOptions := &RateLimitOptions{
		MaxRequests: 500,
		WindowSecs:  60,
	}
	r.Use(CombinedRateLimitMiddleware(ipOptions, userOptions, appOptions))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestCombinedRateLimitMiddleware_NilOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(CombinedRateLimitMiddleware(nil, nil, nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestRateLimitOptions_Defaults(t *testing.T) {
	options := &RateLimitOptions{
		MaxRequests: 100,
		WindowSecs:  60,
	}

	assert.Equal(t, 100, options.MaxRequests)
	assert.Equal(t, 60, options.WindowSecs)
}

func TestGetRateLimitService(t *testing.T) {
	service := GetRateLimitService()
	assert.NotNil(t, service)
}

func TestRateLimitResponseHeaders(t *testing.T) {
	r := setupTestRouter()
	options := &RateLimitOptions{
		MaxRequests: 100,
		WindowSecs:  60,
	}
	r.Use(IPRateLimitMiddleware(options))
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

	limit, err := strconv.Atoi(w.Header().Get("X-RateLimit-Limit"))
	if err == nil {
		assert.Greater(t, limit, 0)
	}

	remaining, err := strconv.Atoi(w.Header().Get("X-RateLimit-Remaining"))
	if err == nil {
		assert.GreaterOrEqual(t, remaining, 0)
	}
}

func TestRateLimitServiceResponse(t *testing.T) {
	tests := []struct {
		name           string
		headers        map[string]string
		expectedStatus int
	}{
		{
			name:           "no headers",
			headers:        map[string]string{},
			expectedStatus: http.StatusOK,
		},
		{
			name: "with X-Forwarded-For",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "with X-Real-IP",
			headers: map[string]string{
				"X-Real-IP": "10.0.0.2",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "with both headers",
			headers: map[string]string{
				"X-Forwarded-For": "10.0.0.1",
				"X-Real-IP":       "10.0.0.2",
			},
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			r := setupTestRouter()
			r.Use(IPRateLimitMiddleware(nil))
			r.GET("/test", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"message": "success"})
			})

			req, _ := http.NewRequest("GET", "/test", nil)
			for key, value := range tt.headers {
				req.Header.Set(key, value)
			}
			w := httptest.NewRecorder()
			r.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRateLimitResponseFormat(t *testing.T) {
	r := setupTestRouter()
	options := &RateLimitOptions{
		MaxRequests: 1,
		WindowSecs:  1,
	}
	r.Use(IPRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	for i := 0; i < 2; i++ {
		req, _ := http.NewRequest("GET", "/test", nil)
		req.RemoteAddr = "192.168.1.200:12345"
		w := httptest.NewRecorder()
		r.ServeHTTP(w, req)

		if i == 1 && w.Code == http.StatusTooManyRequests {
			var response map[string]interface{}
			json.Unmarshal(w.Body.Bytes(), &response)
			assert.Equal(t, float64(429), response["code"])
			assert.NotEmpty(t, response["message"])
		}
	}
}

// ==================== Token Bucket Rate Limit Middleware Tests ====================

func TestTokenBucketRateLimitMiddleware_DefaultOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(TokenBucketRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestTokenBucketRateLimitMiddleware_CustomOptions(t *testing.T) {
	r := setupTestRouter()
	options := &TokenBucketOptions{
		Rate:          100,
		Capacity:      1000,
		InitialTokens: 1000,
	}
	r.Use(TokenBucketRateLimitMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-TokenBucket-Limit"))
}

func TestTokenBucketRateLimitMiddleware_WithXForwardedFor(t *testing.T) {
	r := setupTestRouter()
	r.Use(TokenBucketRateLimitMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Forwarded-For", "10.0.0.1, 192.168.1.1")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetTokenBucketRateLimitService(t *testing.T) {
	service := GetTokenBucketRateLimitService()
	assert.NotNil(t, service)
}

// ==================== Quota Middleware Tests ====================

func TestQuotaMiddleware_DefaultOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(QuotaMiddleware(nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.1:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQuotaMiddleware_CustomOptions(t *testing.T) {
	r := setupTestRouter()
	options := &QuotaOptions{
		Type:      service.QuotaTypeDaily,
		Limit:     10000,
		HardLimit: true,
	}
	r.Use(QuotaMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.2:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Quota-Limit"))
}

func TestQuotaMiddleware_WithUserID(t *testing.T) {
	r := setupTestRouter()
	options := &QuotaOptions{
		Type:      service.QuotaTypeDaily,
		Limit:     5000,
		HardLimit: true,
	}
	r.Use(QuotaMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.Set("user_id", uint(123))
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestQuotaMiddleware_WithAppID(t *testing.T) {
	r := setupTestRouter()
	options := &QuotaOptions{
		Type:      service.QuotaTypeHourly,
		Limit:     1000,
		HardLimit: true,
	}
	r.Use(QuotaMiddleware(options))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.Header.Set("X-App-ID", "12345")
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestGetQuotaManagementService(t *testing.T) {
	service := GetQuotaManagementService()
	assert.NotNil(t, service)
}

// ==================== Advanced Combined Middleware Tests ====================

func TestAdvancedCombinedMiddleware(t *testing.T) {
	r := setupTestRouter()
	tbOptions := &TokenBucketOptions{
		Rate:          10,
		Capacity:      100,
		InitialTokens: 100,
	}
	quotaOptions := &QuotaOptions{
		Type:      service.QuotaTypeDaily,
		Limit:     10000,
		HardLimit: true,
	}
	r.Use(AdvancedCombinedMiddleware(tbOptions, quotaOptions))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	req.RemoteAddr = "192.168.1.100:12345"
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestAdvancedCombinedMiddleware_NilOptions(t *testing.T) {
	r := setupTestRouter()
	r.Use(AdvancedCombinedMiddleware(nil, nil))
	r.GET("/test", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"message": "success"})
	})

	req, _ := http.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	r.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}
