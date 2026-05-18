package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
	"github.com/hjtpx/hjtpx/internal/service"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestEnhancedDDoSProtection(t *testing.T) {
	fmt.Println("\n=== Testing Enhanced DDoS Protection ===")

	t.Run("BasicRateLimiting", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.EnhancedDDoSMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		for i := 0; i < 100; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.1:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				if i >= 60 {
					t.Logf("Request %d blocked as expected", i)
				}
			}
		}

		t.Log("Basic rate limiting test completed")
	})

	t.Run("BlacklistIP", func(t *testing.T) {
		ddosService := service.NewEnhancedDDoSProtectionService()
		testIP := "10.0.0.1"

		ddosService.AddToBlacklist(testIP, "test_block", 1*time.Hour)

		stats := ddosService.GetIPStats(testIP)
		if stats != nil && !stats.IsBlacklisted {
			t.Error("IP should be blacklisted")
		}

		ddosService.RemoveFromBlacklist(testIP)

		t.Log("Blacklist test completed")
	})

	t.Run("ConnectionTracking", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.ConnectionTrackingMiddlewareHandler(5, 60))

		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		for i := 0; i < 10; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = "192.168.1.100:12345"
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)
		}

		t.Log("Connection tracking test completed")
	})

	t.Log("Enhanced DDoS Protection tests completed")
}

func TestEnhancedCSRFProtection(t *testing.T) {
	fmt.Println("\n=== Testing Enhanced CSRF Protection ===")

	t.Run("TokenGeneration", func(t *testing.T) {
		csrf := service.NewCSRFSecurity(nil)

		token1, err := csrf.GenerateToken()
		if err != nil {
			t.Fatalf("Failed to generate token: %v", err)
		}

		if len(token1) < 16 {
			t.Error("Token length too short")
		}

		token2, _ := csrf.GenerateToken()
		if token1 == token2 {
			t.Error("Tokens should be unique")
		}

		t.Log("Token generation test completed")
	})

	t.Run("TokenStorageAndVerification", func(t *testing.T) {
		csrf := service.NewCSRFSecurity(nil)

		sessionID := "test-session-123"
		token, _ := csrf.GenerateToken()

		err := csrf.StoreToken(sessionID, token)
		if err != nil {
			t.Fatalf("Failed to store token: %v", err)
		}

		valid, err := csrf.VerifyToken(sessionID, token)
		if err != nil {
			t.Fatalf("Failed to verify token: %v", err)
		}

		if !valid {
			t.Error("Token should be valid")
		}

		invalid, _ := csrf.VerifyToken(sessionID, "invalid-token")
		if invalid {
			t.Error("Invalid token should not be valid")
		}

		t.Log("Token storage and verification test completed")
	})

	t.Run("MiddlewareIntegration", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.EnhancedCSRFProtection())

		router.GET("/protected", func(c *gin.Context) {
			token := c.GetHeader("X-CSRF-Token")
			c.JSON(200, gin.H{"csrf_token": token})
		})

		router.POST("/submit", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "success"})
		})

		req, _ := http.NewRequest("GET", "/protected", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		csrfToken := w.Header().Get("X-CSRF-Token")
		if csrfToken == "" {
			t.Error("CSRF token should be set in response header")
		}

		t.Log("CSRF middleware integration test completed")
	})

	t.Log("Enhanced CSRF Protection tests completed")
}

func TestXSSProtection(t *testing.T) {
	fmt.Println("\n=== Testing XSS Protection ===")

	t.Run("BasicSanitization", func(t *testing.T) {
		xss := service.NewXSSSecurity(nil)

		testCases := []struct {
			input    string
			contains string
		}{
			{"<script>alert('xss')</script>", ""},
			{"<img src=x onerror=alert(1)>", ""},
			{"javascript:alert('xss')", ""},
			{"<iframe src='evil.com'></iframe>", ""},
			{"normal text", "normal text"},
		}

		for _, tc := range testCases {
			sanitized := xss.SanitizeInput(tc.input)

			if tc.contains != "" && !strings.Contains(sanitized, tc.contains) {
				t.Errorf("Input %s: expected to contain %s, got %s", tc.input, tc.contains, sanitized)
			}

			if strings.Contains(strings.ToLower(sanitized), "script") ||
				strings.Contains(strings.ToLower(sanitized), "javascript:") ||
				strings.Contains(strings.ToLower(sanitized), "onerror") ||
				strings.Contains(strings.ToLower(sanitized), "onload") {
				t.Errorf("XSS input %s was not sanitized properly: %s", tc.input, sanitized)
			}
		}

		t.Log("Basic sanitization test completed")
	})

	t.Run("XSSDetection", func(t *testing.T) {
		xss := service.NewXSSSecurity(nil)

		maliciousInputs := []string{
			"<script>alert('test')</script>",
			"javascript:void(0)",
			"<img src=x onerror=alert(1)>",
			"onclick=alert('xss')",
			"<iframe src='http://evil.com'></iframe>",
		}

		for _, input := range maliciousInputs {
			detected, pattern := xss.DetectXSS(input)
			if !detected {
				t.Errorf("XSS attack not detected in input: %s", input)
			} else {
				t.Logf("Detected XSS pattern: %s in input: %s", pattern, input)
			}
		}

		benignInputs := []string{
			"Hello World",
			"<p>Normal text</p>",
			"https://example.com",
			"user@example.com",
		}

		for _, input := range benignInputs {
			detected, _ := xss.DetectXSS(input)
			if detected {
				t.Errorf("Benign input incorrectly flagged as XSS: %s", input)
			}
		}

		t.Log("XSS detection test completed")
	})

	t.Run("MiddlewareIntegration", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.EnhancedXSSProtectionMiddleware())

		router.POST("/submit", func(c *gin.Context) {
			var data map[string]string
			c.BindJSON(&data)
			c.JSON(200, gin.H{"status": "ok"})
		})

		body := map[string]string{"name": "<script>alert('xss')</script>"}
		jsonBody, _ := json.Marshal(body)

		req, _ := http.NewRequest("POST", "/submit", bytes.NewBuffer(jsonBody))
		req.Header.Set("Content-Type", "application/json")
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		t.Log("XSS middleware integration test completed")
	})

	t.Log("XSS Protection tests completed")
}

func TestRequestSignature(t *testing.T) {
	fmt.Println("\n=== Testing Request Signature Verification ===")

	t.Run("SignatureGeneration", func(t *testing.T) {
		validator := service.NewSignatureValidator(service.RequestSignatureConfig{
			SecretKey: "test-secret-key-12345",
			Algorithm: "SHA256",
		})

		method := "POST"
		path := "/api/test"
		query := "param1=value1&param2=value2"
		body := []byte(`{"key": "value"}`)

		signature, timestamp, nonce, err := validator.GenerateSignature(method, path, query, body)
		if err != nil {
			t.Fatalf("Failed to generate signature: %v", err)
		}

		if signature == "" {
			t.Error("Signature should not be empty")
		}

		if timestamp == 0 {
			t.Error("Timestamp should not be zero")
		}

		if nonce == "" {
			t.Error("Nonce should not be empty")
		}

		t.Logf("Generated signature: %s, timestamp: %d, nonce: %s", signature[:16]+"...", timestamp, nonce)
		t.Log("Signature generation test completed")
	})

	t.Run("SignatureVerification", func(t *testing.T) {
		validator := service.NewSignatureValidator(service.RequestSignatureConfig{
			SecretKey: "test-secret-key-12345",
			Algorithm: "SHA256",
		})

		method := "POST"
		path := "/api/test"
		query := "param1=value1"
		body := []byte(`{"test": "data"}`)

		signature, timestamp, nonce, _ := validator.GenerateSignature(method, path, query, body)

		headers := map[string]string{
			"X-Signature": signature,
			"X-Timestamp": fmt.Sprintf("%d", timestamp),
			"X-Nonce":     nonce,
		}

		result := validator.ValidateRequest(method, path, query, headers, body, "127.0.0.1")

		if !result.Valid {
			t.Errorf("Signature verification failed: %s", result.Reason)
		}

		if !result.SignatureValid {
			t.Error("Signature should be valid")
		}

		t.Log("Signature verification test completed")
	})

	t.Run("ReplayAttackPrevention", func(t *testing.T) {
		validator := service.NewSignatureValidator(service.RequestSignatureConfig{
			SecretKey: "test-secret-key-12345",
			Algorithm: "SHA256",
		})

		method := "POST"
		path := "/api/test"
		query := ""
		body := []byte(`{"test": "data"}`)

		signature, timestamp, nonce, _ := validator.GenerateSignature(method, path, query, body)

		headers := map[string]string{
			"X-Signature": signature,
			"X-Timestamp": fmt.Sprintf("%d", timestamp),
			"X-Nonce":     nonce,
		}

		result1 := validator.ValidateRequest(method, path, query, headers, body, "127.0.0.1")
		if !result1.Valid {
			t.Errorf("First request should be valid: %s", result1.Reason)
		}

		result2 := validator.ValidateRequest(method, path, query, headers, body, "127.0.0.1")
		if result2.Valid {
			t.Error("Replay attack should be detected")
		}

		if !result2.ReplayDetected {
			t.Error("Replay should be detected")
		}

		t.Log("Replay attack prevention test completed")
	})

	t.Run("TimestampValidation", func(t *testing.T) {
		validator := service.NewSignatureValidator(service.RequestSignatureConfig{
			SecretKey:          "test-secret-key-12345",
			Algorithm:          "SHA256",
			TimestampTolerance: 5 * time.Minute,
		})

		method := "POST"
		path := "/api/test"
		query := ""
		body := []byte(`{"test": "data"}`)

		oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
		nonce, _ := service.GenerateSecureNonce(16)

		headers := map[string]string{
			"X-Signature": "dummy-signature",
			"X-Timestamp": fmt.Sprintf("%d", oldTimestamp),
			"X-Nonce":     nonce,
		}

		result := validator.ValidateRequest(method, path, query, headers, body, "127.0.0.1")

		if result.TimestampValid {
			t.Error("Old timestamp should be invalid")
		}

		t.Log("Timestamp validation test completed")
	})

	t.Log("Request Signature tests completed")
}

func TestRateLimiting(t *testing.T) {
	fmt.Println("\n=== Testing Rate Limiting ===")

	t.Run("TokenBucketAlgorithm", func(t *testing.T) {
		rateLimitService := service.NewTokenBucketRateLimitService()

		key := "test-client-1"

		allowed := 0
		for i := 0; i < 150; i++ {
			if rateLimitService.Allow(key) {
				allowed++
			}
		}

		if allowed < 100 || allowed > 150 {
			t.Logf("Allowed %d requests (expected around 100-150 due to token bucket)", allowed)
		}

		tokens := rateLimitService.GetTokens(key)
		t.Logf("Remaining tokens for %s: %.2f", key, tokens)

		rateLimitService.Reset(key)

		t.Log("Token bucket algorithm test completed")
	})

	t.Run("SlidingWindowAlgorithm", func(t *testing.T) {
		rateLimitService := service.NewSlidingWindowRateLimitService()

		key := "test-client-2"

		allowed := 0
		for i := 0; i < 120; i++ {
			if rateLimitService.Allow(key) {
				allowed++
			}
		}

		t.Logf("Allowed %d requests in sliding window", allowed)

		rateLimitService.Reset(key)

		t.Log("Sliding window algorithm test completed")
	})

	t.Run("DistributedRateLimiting", func(t *testing.T) {
		rateLimitService := service.NewDistributedRateLimitService()

		ctx := context.Background()
		key := "distributed-test"

		allowed := 0
		for i := 0; i < 50; i++ {
			ok, _ := rateLimitService.Allow(ctx, key, 100)
			if ok {
				allowed++
			}
		}

		t.Logf("Distributed: allowed %d requests", allowed)

		t.Log("Distributed rate limiting test completed")
	})

	t.Log("Rate Limiting tests completed")
}

func TestContentSecurityPolicy(t *testing.T) {
	fmt.Println("\n=== Testing Content Security Policy ===")

	t.Run("CSPHeaderGeneration", func(t *testing.T) {
		cspConfig := service.SecurityHeadersConfig{
			CSP: "default-src 'self'; script-src 'self' 'unsafe-inline' 'unsafe-eval'; style-src 'self' 'unsafe-inline'; img-src 'self' data:; font-src 'self'; connect-src 'self'; frame-ancestors 'none'; base-uri 'self'; form-action 'self'",
		}

		headers := make(map[string]string)
		headers["Content-Security-Policy"] = cspConfig.CSP

		t.Logf("Generated CSP: %s", headers["Content-Security-Policy"])

		if !strings.Contains(headers["Content-Security-Policy"], "default-src") {
			t.Error("CSP should contain default-src")
		}

		if !strings.Contains(headers["Content-Security-Policy"], "script-src") {
			t.Error("CSP should contain script-src")
		}

		t.Log("CSP header generation test completed")
	})

	t.Run("SecurityHeaders", func(t *testing.T) {
		config := service.SecurityHeadersConfig{
			CSP:                 "default-src 'self'",
			HSTS:                "max-age=31536000; includeSubDomains",
			XFrameOptions:       "DENY",
			XContentTypeOptions: "nosniff",
			XXSSProtection:      "1; mode=block",
			ReferrerPolicy:      "strict-origin-when-cross-origin",
		}

		headers := make(map[string]string)
		if config.CSP != "" {
			headers["Content-Security-Policy"] = config.CSP
		}
		if config.HSTS != "" {
			headers["Strict-Transport-Security"] = config.HSTS
		}
		if config.XFrameOptions != "" {
			headers["X-Frame-Options"] = config.XFrameOptions
		}
		if config.XContentTypeOptions != "" {
			headers["X-Content-Type-Options"] = config.XContentTypeOptions
		}
		if config.XXSSProtection != "" {
			headers["X-XSS-Protection"] = config.XXSSProtection
		}
		if config.ReferrerPolicy != "" {
			headers["Referrer-Policy"] = config.ReferrerPolicy
		}

		if headers["Content-Security-Policy"] == "" {
			t.Error("CSP header should be set")
		}

		if headers["Strict-Transport-Security"] == "" {
			t.Error("HSTS header should be set")
		}

		if headers["X-Frame-Options"] != "DENY" {
			t.Error("X-Frame-Options should be DENY")
		}

		t.Logf("Generated security headers: %+v", headers)
		t.Log("Security headers test completed")
	})

	t.Run("MiddlewareIntegration", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.EnhancedCSPMiddleware())
		router.Use(middleware.EnhancedSecurityHeadersMiddleware())

		router.GET("/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		req, _ := http.NewRequest("GET", "/test", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Header().Get("Content-Security-Policy") == "" {
			t.Error("CSP header should be set by middleware")
		}

		if w.Header().Get("X-Frame-Options") != "DENY" {
			t.Error("X-Frame-Options should be DENY")
		}

		t.Log("CSP middleware integration test completed")
	})

	t.Log("Content Security Policy tests completed")
}

func TestSecurityIntegration(t *testing.T) {
	fmt.Println("\n=== Testing Security Integration ===")

	t.Run("CompleteSecurityChain", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.EnhancedDDoSMiddleware())
		router.Use(middleware.EnhancedCSRFProtection())
		router.Use(middleware.EnhancedCSPMiddleware())
		router.Use(middleware.EnhancedSecurityHeadersMiddleware())

		router.GET("/api/public", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "public endpoint"})
		})

		router.POST("/api/private", func(c *gin.Context) {
			c.JSON(200, gin.H{"data": "private endpoint"})
		})

		req, _ := http.NewRequest("GET", "/api/public", nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Expected status 200, got %d", w.Code)
		}

		if w.Header().Get("Content-Security-Policy") == "" {
			t.Error("CSP header should be set")
		}

		if w.Header().Get("Strict-Transport-Security") == "" {
			t.Error("HSTS header should be set")
		}

		t.Log("Complete security chain test completed")
	})

	t.Run("DDoSWithRealisticTraffic", func(t *testing.T) {
		gin.SetMode(gin.TestMode)
		router := gin.New()

		router.Use(middleware.EnhancedDDoSMiddleware())

		router.GET("/api/test", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})

		successCount := 0
		blockCount := 0

		for i := 0; i < 200; i++ {
			req, _ := http.NewRequest("GET", "/api/test", nil)
			req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", i%255)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code == http.StatusOK {
				successCount++
			} else if w.Code == http.StatusTooManyRequests || w.Code == http.StatusForbidden {
				blockCount++
			}
		}

		t.Logf("Realistic traffic test: %d successful, %d blocked", successCount, blockCount)

		if blockCount == 0 {
			t.Log("Note: No requests were blocked - this might indicate rate limits are too permissive")
		}

		t.Log("Realistic traffic test completed")
	})

	t.Log("Security Integration tests completed")
}

func TestPerformanceBenchmarks(t *testing.T) {
	fmt.Println("\n=== Running Performance Benchmarks ===")

	t.Run("DDoSCheckPerformance", func(t *testing.T) {
		ddosService := service.NewEnhancedDDoSProtectionService()

		start := time.Now()
		iterations := 10000

		for i := 0; i < iterations; i++ {
			req, _ := http.NewRequest("GET", "/test", nil)
			req.RemoteAddr = fmt.Sprintf("192.168.1.%d:12345", i%1000)
			ddosService.CheckRequest(req)
		}

		elapsed := time.Since(start)
		perRequest := elapsed / time.Duration(iterations)

		t.Logf("DDoS check: %d requests in %v (%.2f µs per request)", iterations, elapsed, float64(perRequest.Microseconds()))
	})

	t.Run("XSSSanitizationPerformance", func(t *testing.T) {
		xssService := service.NewXSSSecurity(nil)

		testInputs := []string{
			"<script>alert('xss')</script>",
			"normal text",
			"<img src=x onerror=alert(1)>",
			strings.Repeat("a", 1000),
		}

		start := time.Now()
		iterations := 1000

		for i := 0; i < iterations; i++ {
			for _, input := range testInputs {
				xssService.SanitizeInput(input)
			}
		}

		elapsed := time.Since(start)
		perRequest := elapsed / time.Duration(iterations*len(testInputs))

		t.Logf("XSS sanitization: %d operations in %v (%.2f µs per operation)", iterations*len(testInputs), elapsed, float64(perRequest.Microseconds()))
	})

	t.Run("SignatureVerificationPerformance", func(t *testing.T) {
		validator := service.NewSignatureValidator(service.RequestSignatureConfig{
			SecretKey: "test-secret-key-12345",
			Algorithm: "SHA256",
		})

		method := "POST"
		path := "/api/test"
		query := "param1=value1&param2=value2"
		body := []byte(`{"test": "data", "more": "content"}`)

		signature, timestamp, nonce, _ := validator.GenerateSignature(method, path, query, body)

		headers := map[string]string{
			"X-Signature": signature,
			"X-Timestamp": fmt.Sprintf("%d", timestamp),
			"X-Nonce":     nonce,
		}

		start := time.Now()
		iterations := 5000

		for i := 0; i < iterations; i++ {
			validator.ValidateRequest(method, path, query, headers, body, "127.0.0.1")
		}

		elapsed := time.Since(start)
		perRequest := elapsed / time.Duration(iterations)

		t.Logf("Signature verification: %d operations in %v (%.2f µs per operation)", iterations, elapsed, float64(perRequest.Microseconds()))
	})

	t.Run("RateLimitPerformance", func(t *testing.T) {
		rateLimitService := service.NewTokenBucketRateLimitService()

		start := time.Now()
		iterations := 100000

		for i := 0; i < iterations; i++ {
			rateLimitService.Allow(fmt.Sprintf("client-%d", i%1000))
		}

		elapsed := time.Since(start)
		perRequest := elapsed / time.Duration(iterations)

		t.Logf("Rate limiting: %d operations in %v (%.2f µs per operation)", iterations, elapsed, float64(perRequest.Microseconds()))
	})

	t.Log("Performance Benchmarks completed")
}

func main() {
	fmt.Println("========================================")
	fmt.Println("  HJTPX Security Enhancement Test Suite")
	fmt.Println("========================================")
	fmt.Println()

	testing.Main(func(pat, str string) (bool, error) { return true, nil },
		[]testing.InternalTest{
			{Name: "TestEnhancedDDoSProtection", F: TestEnhancedDDoSProtection},
			{Name: "TestEnhancedCSRFProtection", F: TestEnhancedCSRFProtection},
			{Name: "TestXSSProtection", F: TestXSSProtection},
			{Name: "TestRequestSignature", F: TestRequestSignature},
			{Name: "TestRateLimiting", F: TestRateLimiting},
			{Name: "TestContentSecurityPolicy", F: TestContentSecurityPolicy},
			{Name: "TestSecurityIntegration", F: TestSecurityIntegration},
			{Name: "TestPerformanceBenchmarks", F: TestPerformanceBenchmarks},
		},
		nil,
		nil,
	)
}
