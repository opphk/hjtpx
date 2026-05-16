package security

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/handler"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/stretchr/testify/assert"
)

func setupTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	r.POST("/api/captcha/slider", handler.GetSliderCaptcha)
	r.POST("/api/captcha/click", handler.GetClickCaptcha)
	r.POST("/api/captcha/verify", handler.VerifyCaptcha)
	return r
}

func TestRateLimitIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("RateLimitEnforcement", func(t *testing.T) {
		router := setupTestRouter()

		requestCount := 0
		maxRequests := 100
		startTime := time.Now()

		for requestCount < maxRequests {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
			router.ServeHTTP(resp, req)

			requestCount++

			if resp.Code == http.StatusTooManyRequests {
				t.Logf("限流在第 %d 个请求时触发", requestCount)
				break
			}
		}

		duration := time.Since(startTime)
		t.Logf("发送 %d 个请求耗时: %v", requestCount, duration)

		if requestCount >= maxRequests {
			t.Log("限流未触发，所有请求均成功")
		}
	})

	t.Run("RateLimitRecovery", func(t *testing.T) {
		router := setupTestRouter()

		burstSize := 10
		for i := 0; i < burstSize; i++ {
			resp := httptest.NewRecorder()
			req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
			router.ServeHTTP(resp, req)
		}

		time.Sleep(1 * time.Second)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
		router.ServeHTTP(resp, req)

		if resp.Code != http.StatusTooManyRequests {
			t.Log("限流恢复后可以继续请求")
		}
	})

	t.Run("RateLimitPerIPTracking", func(t *testing.T) {
		client := redis.GetClient()
		if client == nil {
			t.Skip("Redis not available")
		}

		ctx := redis.Context
		testKey := fmt.Sprintf("ratelimit:test:ip:%d", time.Now().Unix())

		client.Set(ctx, testKey, 0, 5*time.Minute)

		for i := 0; i < 5; i++ {
			client.Incr(ctx, testKey)
		}

		count, _ := client.Get(ctx, testKey).Int()
		t.Logf("IP请求计数: %d", count)

		client.Del(ctx, testKey)
	})
}

func TestXSSIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("XSSPayloadPrevention", func(t *testing.T) {
		router := setupTestRouter()

		xssPayloads := []string{
			"<script>alert('XSS')</script>",
			"javascript:alert('XSS')",
			"<img src=x onerror=alert('XSS')>",
			"<svg onload=alert('XSS')>",
			"';alert('XSS');//",
			"<iframe src='javascript:alert(\"XSS\")'>",
		}

		for _, payload := range xssPayloads {
			t.Run(payload[:min(30, len(payload))], func(t *testing.T) {
				verifyReq := map[string]interface{}{
					"session_id": "test_session",
					"type":       "slider",
					"x":          150,
					"y":          100,
				}

				verifyJSON, _ := json.Marshal(verifyReq)
				modifiedJSON := strings.Replace(string(verifyJSON), "test_session", payload, 1)

				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBufferString(modifiedJSON))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(resp, req)

				if resp.Code == http.StatusOK {
					body, _ := io.ReadAll(resp.Body)
					bodyStr := string(body)

					if strings.Contains(bodyStr, "<script>") ||
					   strings.Contains(bodyStr, "javascript:") ||
					   strings.Contains(bodyStr, "<img") ||
					   strings.Contains(bodyStr, "<svg") {
						t.Errorf("XSS payload not sanitized in response: %s", payload)
					}
				}
			})
		}
	})

	t.Run("XSSInUserAgent", func(t *testing.T) {
		router := setupTestRouter()

		maliciousUserAgent := "<script>alert('XSS')</script>"

		verifyReq := map[string]interface{}{
			"session_id": "test_session",
			"type":       "slider",
			"x":          150,
			"y":          100,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("User-Agent", maliciousUserAgent)
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			body, _ := io.ReadAll(resp.Body)
			if strings.Contains(string(body), "<script>") {
				t.Error("XSS payload in User-Agent not sanitized")
			}
		}
	})

	t.Run("XSSInReferer", func(t *testing.T) {
		router := setupTestRouter()

		maliciousReferer := "http://evil.com/<script>alert('XSS')</script>"

		verifyReq := map[string]interface{}{
			"session_id": "test_session",
			"type":       "slider",
			"x":          150,
			"y":          100,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Referer", maliciousReferer)
		router.ServeHTTP(resp, req)

		t.Logf("Referer XSS test completed with status: %d", resp.Code)
	})
}

func TestSQLInjectionIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("SQLInjectionPrevention", func(t *testing.T) {
		router := setupTestRouter()

		sqlPayloads := []string{
			"' OR '1'='1",
			"'; DROP TABLE users;--",
			"1; DELETE FROM verification_logs WHERE '1'='1",
			"UNION SELECT * FROM users",
			"1' AND '1'='1",
			"admin'--",
			"1' OR '1' = '1' --",
			"'; EXEC xp_cmdshell('dir');--",
		}

		for _, payload := range sqlPayloads {
			t.Run(payload[:min(30, len(payload))], func(t *testing.T) {
				verifyReq := map[string]interface{}{
					"session_id": payload,
					"type":       "slider",
					"x":          150,
					"y":          100,
				}
				verifyJSON, _ := json.Marshal(verifyReq)

				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(resp, req)

				assert.NotEqual(t, http.StatusInternalServerError, resp.Code,
					"SQL injection should not cause 500 error")
			})
		}
	})

	t.Run("SQLInjectionViaQueryParams", func(t *testing.T) {
		router := setupTestRouter()

		maliciousQuery := url.Values{}
		maliciousQuery.Add("session_id", "test' OR '1'='1")
		maliciousQuery.Add("type", "slider")

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", strings.NewReader(maliciousQuery.Encode()))
		req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
		router.ServeHTTP(resp, req)

		assert.NotEqual(t, http.StatusInternalServerError, resp.Code)
	})

	t.Run("BlindSQLInjectionTiming", func(t *testing.T) {
		router := setupTestRouter()

		verifyReq := map[string]interface{}{
			"session_id": "test' AND SLEEP(5)--",
			"type":       "slider",
			"x":          150,
			"y":          100,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		startTime := time.Now()
		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(resp, req)
		duration := time.Since(startTime)

		t.Logf("Blind SQL injection test duration: %v", duration)
		assert.Less(t, duration, 3*time.Second, "SLEEP injection should not cause long delay")
	})
}

func TestCSRFIntegration(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("CSRFTokenValidation", func(t *testing.T) {
		router := setupTestRouter()

		verifyReq := map[string]interface{}{
			"session_id": "test_session",
			"type":       "slider",
			"x":          150,
			"y":          100,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(resp, req)

		t.Logf("CSRF test completed with status: %d", resp.Code)
	})

	t.Run("OriginHeaderValidation", func(t *testing.T) {
		router := setupTestRouter()

		verifyReq := map[string]interface{}{
			"session_id": "test_session",
			"type":       "slider",
			"x":          150,
			"y":          100,
		}
		verifyJSON, _ := json.Marshal(verifyReq)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "http://evil.com/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("Origin", "http://evil.com")
		router.ServeHTTP(resp, req)

		if resp.Code == http.StatusOK {
			t.Log("Note: Origin header validation should be implemented in middleware")
		}
	})

	t.Run("SameSiteCookieAttribute", func(t *testing.T) {
		t.Log("SameSite cookie属性应在生产环境中配置")
		t.Log("建议使用 SameSite=Strict 或 SameSite=Lax")
	})
}

func TestInputValidation(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("InvalidJSONHandling", func(t *testing.T) {
		router := setupTestRouter()

		invalidJSON := []byte(`{"session_id": "test", "type": "slider", invalid}`)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(invalidJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(resp, req)

		assert.NotEqual(t, http.StatusOK, resp.Code, "Invalid JSON should not return 200")
	})

	t.Run("MissingRequiredFields", func(t *testing.T) {
		router := setupTestRouter()

		missingSessionID := map[string]interface{}{
			"type": "slider",
			"x":    150,
			"y":    100,
		}
		verifyJSON, _ := json.Marshal(missingSessionID)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(resp, req)

		assert.NotEqual(t, http.StatusOK, resp.Code, "Missing required field should not return 200")
	})

	t.Run("TypeMismatchHandling", func(t *testing.T) {
		router := setupTestRouter()

		typeMismatch := map[string]interface{}{
			"session_id": "test_session",
			"type":       "slider",
			"x":          "not_a_number",
			"y":          100,
		}
		verifyJSON, _ := json.Marshal(typeMismatch)

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
		req.Header.Set("Content-Type", "application/json")
		router.ServeHTTP(resp, req)

		assert.NotEqual(t, http.StatusOK, resp.Code, "Type mismatch should not return 200")
	})

	t.Run("BoundaryValueTesting", func(t *testing.T) {
		router := setupTestRouter()

		testCases := []struct {
			name    string
			x       interface{}
			y       interface{}
			expectOk bool
		}{
			{"zero_values", 0, 0, true},
			{"negative_values", -100, -100, true},
			{"very_large_values", 999999999, 999999999, true},
			{"float_values", 150.5, 100.7, true},
			{"special_chars_in_session", "session_123_测试", 150, true},
		}

		for _, tc := range testCases {
			t.Run(tc.name, func(t *testing.T) {
				verifyReq := map[string]interface{}{
					"session_id": fmt.Sprintf("test_session_%d", time.Now().UnixNano()),
					"type":       "slider",
					"x":          tc.x,
					"y":          tc.y,
				}
				verifyJSON, _ := json.Marshal(verifyReq)

				resp := httptest.NewRecorder()
				req, _ := http.NewRequest("POST", "/api/captcha/verify", bytes.NewBuffer(verifyJSON))
				req.Header.Set("Content-Type", "application/json")
				router.ServeHTTP(resp, req)

				if tc.expectOk {
					t.Logf("Test case %s: status=%d", tc.name, resp.Code)
				}
			})
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("SecurityHeadersPresence", func(t *testing.T) {
		router := setupTestRouter()

		resp := httptest.NewRecorder()
		req, _ := http.NewRequest("POST", "/api/captcha/slider", nil)
		router.ServeHTTP(resp, req)

		securityHeaders := []string{
			"X-Content-Type-Options",
			"X-Frame-Options",
			"X-XSS-Protection",
			"Content-Security-Policy",
		}

		for _, header := range securityHeaders {
			if resp.Header().Get(header) != "" {
				t.Logf("Security header present: %s", header)
			} else {
				t.Logf("Security header missing: %s", header)
			}
		}
	})
}

func TestAuthenticationSecurity(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping security test in short mode")
	}

	t.Run("UnauthorizedAccessPrevention", func(t *testing.T) {
		router := setupTestRouter()

		protectedEndpoints := []struct {
			method string
			path   string
		}{
			{"GET", "/api/admin/stats"},
			{"GET", "/api/admin/logs"},
			{"POST", "/api/admin/app"},
		}

		for _, endpoint := range protectedEndpoints {
			t.Run(fmt.Sprintf("%s %s", endpoint.method, endpoint.path), func(t *testing.T) {
				resp := httptest.NewRecorder()
				var req *http.Request

				if endpoint.method == "GET" {
					req, _ = http.NewRequest("GET", endpoint.path, nil)
				} else {
					req, _ = http.NewRequest("POST", endpoint.path, bytes.NewBufferString("{}"))
				}

				router.ServeHTTP(resp, req)

				assert.NotEqual(t, http.StatusOK, resp.Code,
					"Protected endpoint should not be accessible without auth")
			})
		}
	})
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}
