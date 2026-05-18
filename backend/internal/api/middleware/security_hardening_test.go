package middleware

import (
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/gin-gonic/gin"
	"github.com/stretchr/testify/assert"
)

func TestSSRFProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		url            string
		expectedStatus int
		shouldBlock    bool
	}{
		{
			name:        "Localhost HTTP",
			url:         "?url=http://localhost/admin",
			shouldBlock: true,
		},
		{
			name:        "127.0.0.1 HTTP",
			url:         "?url=http://127.0.0.1:8080/admin",
			shouldBlock: true,
		},
		{
			name:        "Private IP 192.168",
			url:         "?url=http://192.168.1.1/admin",
			shouldBlock: true,
		},
		{
			name:        "Private IP 10.x",
			url:         "?url=http://10.0.0.1/admin",
			shouldBlock: true,
		},
		{
			name:        "Private IP 172.x",
			url:         "?url=http://172.16.0.1/admin",
			shouldBlock: true,
		},
		{
			name:        "Cloud Metadata AWS",
			url:         "?url=http://169.254.169.254/latest/meta-data/",
			shouldBlock: true,
		},
		{
			name:        "IPv6 Localhost",
			url:         "?url=http://[::1]/admin",
			shouldBlock: true,
		},
		{
			name:        "File Protocol",
			url:         "?url=file:///etc/passwd",
			shouldBlock: true,
		},
		{
			name:        "Gopher Protocol",
			url:         "?url=gopher://localhost:70/",
			shouldBlock: true,
		},
		{
			name:        "Dict Protocol",
			url:         "?url=dict://localhost:11211/stats",
			shouldBlock: true,
		},
		{
			name:        "LDAP Protocol",
			url:         "?url=ldap://localhost:389/",
			shouldBlock: true,
		},
		{
			name:        "Public URL",
			url:         "?url=https://example.com/api",
			shouldBlock: false,
		},
		{
			name:        "Normal GitHub API",
			url:         "?url=https://api.github.com/users",
			shouldBlock: false,
		},
		{
			name:        "No URL Parameter",
			url:         "/test",
			shouldBlock: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(SSRFProtectionMiddleware())
			router.GET("/fetch", func(c *gin.Context) {
				c.JSON(http.StatusOK, gin.H{"status": "ok"})
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/fetch"+tc.url, nil)
			router.ServeHTTP(w, req)

			if tc.shouldBlock {
				assert.Equal(t, http.StatusForbidden, w.Code, "Test case: %s", tc.name)
			} else {
				assert.Equal(t, http.StatusOK, w.Code, "Test case: %s", tc.name)
			}
		})
	}
}

func TestSSRFProtectionWithAllowedDomains(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SSRFProtectionMiddleware(SSRFProtectionConfig{
		Enabled:        true,
		AllowedDomains: []string{"example.com", "trusted-site.com"},
	}))
	router.GET("/fetch", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	testCases := []struct {
		name        string
		url         string
		shouldBlock bool
	}{
		{
			name:        "Allowed Domain",
			url:         "?url=https://example.com/api",
			shouldBlock: false,
		},
		{
			name:        "Subdomain Allowed",
			url:         "?url=https://api.example.com/data",
			shouldBlock: false,
		},
		{
			name:        "Not Allowed Domain",
			url:         "?url=http://untrusted.com/api",
			shouldBlock: true,
		},
		{
			name:        "Private IP Despite Allowed Domain Config",
			url:         "?url=http://127.0.0.1",
			shouldBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/fetch"+tc.url, nil)
			router.ServeHTTP(w, req)

			if tc.shouldBlock {
				assert.Equal(t, http.StatusForbidden, w.Code, "Test case: %s", tc.name)
			} else {
				assert.Equal(t, http.StatusOK, w.Code, "Test case: %s", tc.name)
			}
		})
	}
}

func TestSSRFProtectionDisabled(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(SSRFProtectionMiddleware(SSRFProtectionConfig{
		Enabled: false,
	}))
	router.GET("/fetch", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/fetch?url=http://localhost", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
}

func TestIsSSRFAttack(t *testing.T) {
	testCases := []struct {
		name        string
		url         string
		shouldBlock bool
	}{
		{
			name:        "Empty URL",
			url:         "",
			shouldBlock: false,
		},
		{
			name:        "Localhost",
			url:         "http://localhost",
			shouldBlock: true,
		},
		{
			name:        "127.0.0.1",
			url:         "http://127.0.0.1",
			shouldBlock: true,
		},
		{
			name:        "Private Network",
			url:         "http://192.168.1.1",
			shouldBlock: true,
		},
		{
			name:        "Public URL",
			url:         "https://example.com",
			shouldBlock: false,
		},
		{
			name:        "File Protocol",
			url:         "file:///etc/passwd",
			shouldBlock: true,
		},
		{
			name:        "Gopher Protocol",
			url:         "gopher://localhost",
			shouldBlock: true,
		},
		{
			name:        "Cloud Metadata",
			url:         "http://169.254.169.254",
			shouldBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isSSRFAttack(tc.url, defaultSSRFConfig)
			assert.Equal(t, tc.shouldBlock, result, "Test case: %s", tc.name)
		})
	}
}

func TestIsPrivateIP(t *testing.T) {
	testCases := []struct {
		name     string
		ip       string
		expected bool
	}{
		{"10.0.0.1", "10.0.0.1", true},
		{"172.16.0.1", "172.16.0.1", true},
		{"192.168.1.1", "192.168.1.1", true},
		{"127.0.0.1", "127.0.0.1", true},
		{"169.254.0.1", "169.254.0.1", true},
		{"8.8.8.8", "8.8.8.8", false},
		{"1.1.1.1", "1.1.1.1", false},
		{"Public IP", "203.0.113.1", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := isPrivateIP(tc.ip)
			assert.Equal(t, tc.expected, result, "Test case: %s", tc.name)
		})
	}
}

func TestEnhancedXSSProtectionMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name        string
		body        string
		shouldSanitize bool
	}{
		{
			name:        "Script Tag",
			body:        "<script>alert('XSS')</script>",
			shouldSanitize: true,
		},
		{
			name:        "IMG with onerror",
			body:        `<img src=x onerror="alert(1)">`,
			shouldSanitize: true,
		},
		{
			name:        "Normal Content",
			body:        "Hello World",
			shouldSanitize: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(EnhancedXSSProtectionMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test", nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, http.StatusOK, w.Code)
		})
	}
}

func TestEnhancedCSPMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedCSPMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	cspHeader := w.Header().Get("Content-Security-Policy")
	assert.NotEmpty(t, cspHeader)
	assert.Contains(t, cspHeader, "default-src 'self'")
}

func TestEnhancedInputValidationMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		query          string
		expectedStatus int
	}{
		{
			name:           "Normal Query",
			query:          "?input=hello",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "XSS Attack",
			query:          "?input=<script>alert(1)</script>",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "SQL Injection",
			query:          "?input=' OR '1'='1",
			expectedStatus: http.StatusOK,
		},
		{
			name:           "Normal Number",
			query:          "?input=12345",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(EnhancedInputValidationMiddleware())
			router.GET("/test", func(c *gin.Context) {
				c.String(http.StatusOK, "ok")
			})

			w := httptest.NewRecorder()
			req, _ := http.NewRequest("GET", "/test"+tc.query, nil)
			router.ServeHTTP(w, req)

			assert.Equal(t, tc.expectedStatus, w.Code, "Test case: %s", tc.name)
		})
	}
}

func TestEnhancedInputValidationMaxBodySize(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedInputValidationMiddleware(EnhancedInputValidationMiddlewareConfig{
		MaxBodySize: 10,
	}))
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	_ = "This is a very long body that exceeds the limit"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/test", nil)
	req.Body = nil
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusRequestEntityTooLarge, w.Code)
}

func TestEnhancedInputValidationMaxQueryParams(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedInputValidationMiddleware(EnhancedInputValidationMiddlewareConfig{
		MaxQueryParams: 5,
	}))
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	query := "?a=1&b=2&c=3&d=4&e=5&f=6&g=7"
	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test"+query, nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusBadRequest, w.Code)
}

func TestEnhancedSecurityHeadersMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedSecurityHeadersMiddleware())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
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
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestEnhancedCSRFProtection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedCSRFProtection())
	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})
	router.POST("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	t.Run("Safe Method Should Set Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("GET", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusOK, w.Code)
		assert.NotEmpty(t, w.Header().Get("X-CSRF-Token"))
	})

	t.Run("Unsafe Method Without Token", func(t *testing.T) {
		w := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/test", nil)
		router.ServeHTTP(w, req)

		assert.Equal(t, http.StatusForbidden, w.Code)
	})
}

func TestSetupEnhancedSecurityMiddleware(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	SetupEnhancedSecurityMiddleware(router)

	router.GET("/test", func(c *gin.Context) {
		c.String(http.StatusOK, "ok")
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	router.ServeHTTP(w, req)

	assert.Equal(t, http.StatusOK, w.Code)
	assert.NotEmpty(t, w.Header().Get("X-Request-ID"))
}

func TestSanitizeInputFunction(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Script Tag", "<script>alert(1)</script>", ""},
		{"IMG with onerror", `<img src=x onerror=alert(1)>`, ""},
		{"SVG with onload", `<svg onload=alert(1)>`, ""},
		{"Normal Text", "Hello World", "Hello World"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeInput(tc.input)
			if tc.expected == "" {
				assert.NotContains(t, result, "script", "Test case: %s", tc.name)
				assert.NotContains(t, result, "onerror", "Test case: %s", tc.name)
			} else {
				assert.Contains(t, result, tc.expected, "Test case: %s", tc.name)
			}
		})
	}
}

func TestDetectXSSFunction(t *testing.T) {
	testCases := []struct {
		name         string
		input        string
		shouldDetect bool
	}{
		{"Script Tag", "<script>alert(1)</script>", true},
		{"JavaScript Protocol", "javascript:alert(1)", true},
		{"Normal Text", "Hello World", false},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			detected, _ := DetectXSS(tc.input)
			assert.Equal(t, tc.shouldDetect, detected, "Test case: %s", tc.name)
		})
	}
}

func TestSanitizeHTMLFunction(t *testing.T) {
	testCases := []struct {
		name     string
		input    string
		expected string
	}{
		{"Script Tag", "<script>alert(1)</script>", ""},
		{"IMG with onerror", `<img src=x onerror=alert(1)>`, ""},
		{"Iframe", "<iframe src='evil.com'></iframe>", ""},
		{"Normal Text", "Hello World", "Hello World"},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := SanitizeHTML(tc.input)
			assert.NotContains(t, result, "<script", "Test case: %s", tc.name)
			assert.NotContains(t, result, "onerror", "Test case: %s", tc.name)
			assert.NotContains(t, result, "<iframe", "Test case: %s", tc.name)
		})
	}
}
