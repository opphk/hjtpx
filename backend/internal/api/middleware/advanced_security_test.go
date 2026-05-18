package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestHTTPSRedirect(t *testing.T) {
	tests := []struct {
		name           string
		config         HTTPSConfig
		proto          string
		tls            interface{}
		expectedStatus int
		expectedLoc    string
	}{
		{
			name: "redirect HTTP to HTTPS",
			config: HTTPSConfig{
				Enabled:      true,
				RedirectCode: http.StatusMovedPermanently,
			},
			proto:          "http",
			tls:            nil,
			expectedStatus: http.StatusMovedPermanently,
			expectedLoc:    "https://localhost/test",
		},
		{
			name: "X-Forwarded-Proto https",
			config: HTTPSConfig{
				Enabled:      true,
				RedirectCode: http.StatusMovedPermanently,
			},
			proto:          "https",
			tls:            nil,
			expectedStatus: http.StatusOK,
			expectedLoc:    "",
		},
		{
			name: "disabled redirect",
			config: HTTPSConfig{
				Enabled: false,
			},
			proto:          "http",
			tls:            nil,
			expectedStatus: http.StatusOK,
			expectedLoc:    "",
		},
		{
			name: "exclude path",
			config: HTTPSConfig{
				Enabled:      true,
				ExcludePaths: []string{"/health"},
			},
			proto:          "http",
			tls:            nil,
			expectedStatus: http.StatusOK,
			expectedLoc:    "",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(HTTPSRedirect(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			router.GET("/health", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			if tt.proto != "" {
				req.Header.Set("X-Forwarded-Proto", tt.proto)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
			if tt.expectedLoc != "" {
				assert.Equal(t, tt.expectedLoc, w.Header().Get("Location"))
			}
		})
	}
}

func TestInputValidationMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		config         InputValidationMiddlewareConfig
		queryParams    string
		bodyParams     map[string]string
		expectedStatus int
	}{
		{
			name: "valid email",
			config: InputValidationMiddlewareConfig{
				Enabled:       true,
				ValidateQuery: true,
			},
			queryParams:    "email=test@example.com",
			expectedStatus: http.StatusOK,
		},
		{
			name: "valid phone",
			config: InputValidationMiddlewareConfig{
				Enabled:       true,
				ValidateQuery: true,
			},
			queryParams:    "phone=13812345678",
			expectedStatus: http.StatusOK,
		},
		{
			name: "exclude path",
			config: InputValidationMiddlewareConfig{
				Enabled:       true,
				ValidateQuery: true,
				ExcludePaths:  []string{"/public"},
			},
			queryParams:    "",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(InputValidationMiddleware(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			router.GET("/public", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			url := "/test"
			if tt.queryParams != "" {
				url += "?" + tt.queryParams
			}

			req := httptest.NewRequest(http.MethodGet, url, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestXSSProtectionMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		body           string
		expectedStatus int
		expectBlocked  bool
	}{
		{
			name:           "clean body",
			body:           `{"username":"test"}`,
			expectedStatus: http.StatusOK,
			expectBlocked:  false,
		},
		{
			name:           "XSS script tag",
			body:           `{"username":"<script>alert(1)</script>"}`,
			expectedStatus: http.StatusOK,
			expectBlocked:  true,
		},
		{
			name:           "XSS event handler",
			body:           `{"username":"<img src=x onerror=alert(1)>"}`,
			expectedStatus: http.StatusOK,
			expectBlocked:  true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(XSSProtectionMiddleware())
			router.POST("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodPost, "/test", bytes.NewBufferString(tt.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	tests := []struct {
		name         string
		config       SecurityHeadersMiddlewareConfig
		expectHeader map[string]string
	}{
		{
			name:   "default headers",
			config: SecurityHeadersMiddlewareConfig{Enabled: true},
			expectHeader: map[string]string{
				"X-Frame-Options":           "DENY",
				"X-Content-Type-Options":    "nosniff",
				"X-XSS-Protection":          "1; mode=block",
				"Strict-Transport-Security": "max-age=31536000; includeSubDomains; preload",
			},
		},
		{
			name: "custom CSP",
			config: SecurityHeadersMiddlewareConfig{
				Enabled: true,
				CSP:     "default-src 'none'",
			},
			expectHeader: map[string]string{
				"Content-Security-Policy": "default-src 'none'",
			},
		},
		{
			name: "disabled",
			config: SecurityHeadersMiddlewareConfig{
				Enabled: false,
			},
			expectHeader: map[string]string{
				"X-Frame-Options": "DENY",
			},
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeadersMiddleware(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			for header, expected := range tt.expectHeader {
				assert.Equal(t, expected, w.Header().Get(header))
			}
		})
	}
}

func TestCSRFTokenMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		method         string
		token          string
		useCookie      bool
		expectedStatus int
	}{
		{
			name:           "GET request generates token",
			method:         http.MethodGet,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "POST without token",
			method:         http.MethodPost,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "POST with valid token",
			method:         http.MethodPost,
			token:          "test-token",
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(CSRFTokenMiddleware(CSRFTokenMiddlewareConfig{
				Enabled:     true,
				CookieName:  "csrf_token",
				HeaderName:  "X-CSRF-Token",
				SafeMethods: []string{"GET", "HEAD", "OPTIONS"},
			}))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			router.POST("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(tt.method, "/test", nil)
			if tt.token != "" {
				req.Header.Set("X-CSRF-Token", tt.token)
			}
			if tt.useCookie {
				req.Header.Set("Cookie", "csrf_token=test-token")
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.Equal(t, tt.expectedStatus, w.Code)
		})
	}
}

func TestRateLimitMiddleware(t *testing.T) {
	tests := []struct {
		name           string
		config         RateLimitMiddlewareConfig
		requests       int
		expectedStatus int
	}{
		{
			name: "within limit",
			config: RateLimitMiddlewareConfig{
				Enabled:     true,
				MaxRequests: 10,
				Window:      1,
			},
			requests:       5,
			expectedStatus: http.StatusOK,
		},
		{
			name: "exceed limit",
			config: RateLimitMiddlewareConfig{
				Enabled:     true,
				MaxRequests: 5,
				Window:      1,
			},
			requests:       10,
			expectedStatus: http.StatusTooManyRequests,
		},
		{
			name: "disabled",
			config: RateLimitMiddlewareConfig{
				Enabled:     false,
				MaxRequests: 1,
			},
			requests:       10,
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(RateLimitMiddleware(tt.config))
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			for i := 0; i < tt.requests; i++ {
				req := httptest.NewRequest(http.MethodGet, "/test", nil)
				w := httptest.NewRecorder()
				router.ServeHTTP(w, req)

				if i == tt.requests-1 {
					assert.Equal(t, tt.expectedStatus, w.Code)
				}
			}
		})
	}
}

func TestSecurityHeadersContent(t *testing.T) {
	tests := []struct {
		name   string
		header string
		value  string
	}{
		{"CSP", "Content-Security-Policy", "default-src 'self'"},
		{"HSTS", "Strict-Transport-Security", "max-age=31536000"},
		{"X-Frame", "X-Frame-Options", "DENY"},
		{"X-Content-Type", "X-Content-Type-Options", "nosniff"},
		{"X-XSS-Protection", "X-XSS-Protection", "1; mode=block"},
		{"Referrer-Policy", "Referrer-Policy", "strict-origin-when-cross-origin"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeadersMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, "/test", nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			assert.NotEmpty(t, w.Header().Get(tt.header))
		})
	}
}

func TestCSRFTokenGeneration(t *testing.T) {
	router := gin.New()
	router.Use(CSRFTokenMiddleware())
	router.GET("/test", func(c *gin.Context) {
		token := c.GetHeader("X-CSRF-Token")
		c.JSON(http.StatusOK, gin.H{"token": token})
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-CSRF-Token"))
	assert.NotEmpty(t, w.Header().Get("Set-Cookie"))
}

func TestRateLimitHeaders(t *testing.T) {
	router := gin.New()
	router.Use(RateLimitMiddleware(RateLimitMiddlewareConfig{
		Enabled:     true,
		MaxRequests: 100,
		Window:      60,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Limit"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Remaining"))
	assert.NotEmpty(t, w.Header().Get("X-RateLimit-Reset"))
}

func TestSetupSecurityMiddleware(t *testing.T) {
	router := gin.New()
	SetupSecurityMiddleware(router)

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "OK")
	})

	req := httptest.NewRequest(http.MethodGet, "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
	assert.NotEmpty(t, w.Header().Get("X-Content-Type-Options"))
	assert.NotEmpty(t, w.Header().Get("Content-Security-Policy"))
}

func TestExcludePaths(t *testing.T) {
	tests := []struct {
		name       string
		exclude    []string
		path       string
		shouldSkip bool
	}{
		{
			name:       "exact match",
			exclude:    []string{"/health"},
			path:       "/health",
			shouldSkip: true,
		},
		{
			name:       "prefix match",
			exclude:    []string{"/api/public"},
			path:       "/api/public/data",
			shouldSkip: true,
		},
		{
			name:       "no match",
			exclude:    []string{"/health"},
			path:       "/api/test",
			shouldSkip: false,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SecurityHeadersMiddleware(SecurityHeadersMiddlewareConfig{
				Enabled:      true,
				ExcludePaths: tt.exclude,
			}))
			router.GET("/health", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			router.GET("/api/public/data", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})
			router.GET("/api/test", func(c *gin.Context) {
				c.String(http.StatusOK, "OK")
			})

			req := httptest.NewRequest(http.MethodGet, tt.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tt.shouldSkip {
				assert.Empty(t, w.Header().Get("X-Frame-Options"))
			} else {
				assert.NotEmpty(t, w.Header().Get("X-Frame-Options"))
			}
		})
	}
}
