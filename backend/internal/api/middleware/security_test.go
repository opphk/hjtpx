package middleware

import (
	"bytes"
	"fmt"
	"io"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func setupSecurityTestRouter() *gin.Engine {
	gin.SetMode(gin.TestMode)
	r := gin.New()
	return r
}

func TestSignatureVerification_MissingSignature(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(SignatureVerification())
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSignatureVerification_ExcludedPath(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(SignatureVerification())
	r.GET("/health", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/health", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for excluded path, got %d", w.Code)
	}
}

func TestSignatureVerification_ValidSignature(t *testing.T) {
	r := setupSecurityTestRouter()
	cfg := defaultSignatureConfig
	cfg.RequireTimestamp = true
	cfg.RequireNonce = true

	r.Use(SignatureVerification(cfg))
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	body := []byte(`{"key":"value"}`)
	timestamp := fmt.Sprintf("%d", time.Now().Unix())
	nonce := "test-nonce-12345"

	signature := GenerateSignature(
		cfg.SecretKey,
		"GET",
		"/api/test",
		"",
		time.Now().Unix(),
		nonce,
		body,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", bytes.NewReader(body))
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", timestamp)
	req.Header.Set("X-Nonce", nonce)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestSignatureVerification_InvalidSignature(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(SignatureVerification())
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Signature", "invalid-signature")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()))
	req.Header.Set("X-Nonce", "test-nonce-12345")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401, got %d", w.Code)
	}
}

func TestSignatureVerification_ExpiredTimestamp(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(SignatureVerification())
	r.GET("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Signature", "some-signature")
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", time.Now().Unix()-600))
	req.Header.Set("X-Nonce", "test-nonce-12345")
	r.ServeHTTP(w, req)

	if w.Code != 401 {
		t.Fatalf("expected 401 for expired timestamp, got %d", w.Code)
	}
}

func TestComputeHMAC(t *testing.T) {
	signature := computeHMAC("secret", "test-data")
	if len(signature) != 64 {
		t.Fatalf("expected HMAC length 64, got %d", len(signature))
	}

	sig2 := computeHMAC("secret", "test-data")
	if signature != sig2 {
		t.Fatal("HMAC should be deterministic")
	}

	sig3 := computeHMAC("different-key", "test-data")
	if signature == sig3 {
		t.Fatal("different key should produce different HMAC")
	}
}

func TestHashBody(t *testing.T) {
	if hashBody(nil) != "" {
		t.Fatal("hashBody(nil) should return empty string")
	}
	if hashBody([]byte{}) != "" {
		t.Fatal("hashBody(empty) should return empty string")
	}

	h := hashBody([]byte("test body"))
	if len(h) != 64 {
		t.Fatalf("expected SHA256 hex length 64, got %d", len(h))
	}

	h2 := hashBody([]byte("test body"))
	if h != h2 {
		t.Fatal("hashBody should be deterministic")
	}
}

func TestSortQueryString(t *testing.T) {
	result := sortQueryString("b=2&a=1&c=3")
	expected := "a=1&b=2&c=3"
	if result != expected {
		t.Fatalf("expected %s, got %s", expected, result)
	}

	if sortQueryString("") != "" {
		t.Fatal("empty query should return empty string")
	}
}

func TestBuildStringToSign(t *testing.T) {
	result := buildStringToSign("GET", "/api/test", "a=1", 1234567890, "nonce123", "bodyhash")
	if !strings.Contains(result, "GET") {
		t.Fatal("should contain method")
	}
	if !strings.Contains(result, "/api/test") {
		t.Fatal("should contain path")
	}
	if !strings.Contains(result, "1234567890") {
		t.Fatal("should contain timestamp")
	}
}

func TestSecureCompare(t *testing.T) {
	if !secureCompare("hello", "hello") {
		t.Fatal("secureCompare should return true for equal strings")
	}
	if secureCompare("hello", "world") {
		t.Fatal("secureCompare should return false for different strings")
	}
	if secureCompare("abc", "abcd") {
		t.Fatal("secureCompare should return false for different lengths")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce(16)
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}
	if len(nonce) == 0 {
		t.Fatal("nonce should not be empty")
	}

	nonce2, err := GenerateNonce(8)
	if err != nil {
		t.Fatalf("GenerateNonce with min length failed: %v", err)
	}
	if len(nonce2) == 0 {
		t.Fatal("nonce should not be empty")
	}
}

func TestGenerateTimestamp(t *testing.T) {
	ts := GenerateTimestamp()
	if ts <= 0 {
		t.Fatal("timestamp should be positive")
	}
	diff := time.Now().Unix() - ts
	if diff > 2 {
		t.Fatalf("timestamp should be recent, diff: %d", diff)
	}
}

func TestBuildSignatureInput(t *testing.T) {
	sig, err := BuildSignatureInput("secret", "POST", "/api/data", "", time.Now().Unix(), "", []byte(`{"key":"value"}`))
	if err != nil {
		t.Fatalf("BuildSignatureInput failed: %v", err)
	}
	if len(sig) != 64 {
		t.Fatalf("expected signature length 64, got %d", len(sig))
	}
}

func TestGetSignatureInfo(t *testing.T) {
	info := GetSignatureInfo()
	if info.Algorithm != "SHA256" {
		t.Fatalf("expected SHA256, got %s", info.Algorithm)
	}
	if !info.NonceRequired {
		t.Fatal("nonce should be required by default")
	}
}

func TestCSRFProtection_SafeMethod(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(CSRFProtection())
	r.GET("/api/test", func(c *gin.Context) {
		token := GetCSRFToken(c)
		if token == "" {
			t.Fatal("CSRF token should be set for GET requests")
		}
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200 for GET, got %d", w.Code)
	}
}

func TestCSRFProtection_UnsafeMethodMissingToken(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(CSRFProtection())
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403 for POST without CSRF token, got %d", w.Code)
	}
}

func TestCSRFProtection_UnsafeMethodWithToken(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(CSRFProtection())
	r.POST("/api/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	var csrfToken string
	r2 := setupSecurityTestRouter()
	r2.Use(CSRFProtection())
	r2.GET("/api/token", func(c *gin.Context) {
		csrfToken = GetCSRFToken(c)
		if csrfToken != "" {
			c.Header("X-CSRF-Token", csrfToken)
		}
		c.JSON(200, gin.H{"token": csrfToken})
	})

	tokenW := httptest.NewRecorder()
	tokenReq, _ := http.NewRequest("GET", "/api/token", nil)
	r2.ServeHTTP(tokenW, tokenReq)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "/api/test", nil)
	req.Header.Set("X-CSRF-Token", csrfToken)
	r.ServeHTTP(w, req)

	if w.Code != 403 {
		t.Fatalf("expected 403 for POST without matching session, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestGenerateCSRFToken(t *testing.T) {
	token := generateToken(32)
	if token == "" {
		t.Fatal("CSRF token should not be empty")
	}
	if len(token) < 32 {
		t.Fatalf("expected token length >= 32, got %d", len(token))
	}

	token2 := generateToken(32)
	if token == token2 {
		t.Fatal("two consecutive tokens should not be equal")
	}
}

func TestHashToken(t *testing.T) {
	token := "test-csrf-token"
	hash := hashToken(token)
	if len(hash) != 64 {
		t.Fatalf("expected hex hash length 64, got %d", len(hash))
	}

	hash2 := hashToken(token)
	if hash != hash2 {
		t.Fatal("hash should be deterministic")
	}
}

func TestCSRFMemoryStore(t *testing.T) {
	store := NewCSRFMemoryStore()

	err := store.Store("token1", "session1")
	if err != nil {
		t.Fatalf("Store failed: %v", err)
	}

	valid, err := store.Verify("token1", "session1")
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}
	if !valid {
		t.Fatal("token should be valid")
	}

	valid, _ = store.Verify("wrong-token", "session1")
	if valid {
		t.Fatal("wrong token should not be valid")
	}

	store.Delete("session1")
	valid, _ = store.Verify("token1", "session1")
	if valid {
		t.Fatal("token should be invalid after deletion")
	}
}

func TestIsSafeMethod(t *testing.T) {
	safeMethods := []string{"GET", "HEAD", "OPTIONS"}
	if !isSafeMethod("GET", safeMethods) {
		t.Fatal("GET should be safe")
	}
	if isSafeMethod("POST", safeMethods) {
		t.Fatal("POST should not be safe")
	}
	if isSafeMethod("PUT", safeMethods) {
		t.Fatal("PUT should not be safe")
	}
}

func TestGenerateSessionID(t *testing.T) {
	r := setupSecurityTestRouter()
	r.GET("/test", func(c *gin.Context) {
		sessionID := generateSessionID(c)
		if sessionID == "" {
			t.Fatal("session ID should not be empty")
		}
		c.JSON(200, gin.H{"session_id": sessionID})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestXSSFilter_SanitizeHTML(t *testing.T) {
	cfg := defaultXSSConfig

	tests := []struct {
		input    string
		expected string
	}{
		{"<script>alert('xss')</script>", ""},
		{"<p>safe</p>", "<p>safe</p>"},
		{"<img src=x onerror=alert(1)>", "<img src=x>"},
		{"<a href=\"javascript:alert(1)\">link</a>", "<a href=\"alert(1)\">link</a>"},
		{"normal text", "normal text"},
		{"<iframe src='http://evil.com'></iframe>", ""},
	}

	for _, tt := range tests {
		result := sanitizeHTML(tt.input, cfg)
		if result.Value != tt.expected {
			t.Errorf("sanitizeHTML(%q) = %q, want %q", tt.input, result.Value, tt.expected)
		}
	}
}

func TestXSSFilter_DetectXSS(t *testing.T) {
	if !isXSSAttempt("<script>alert(1)</script>") {
		t.Fatal("should detect script tag as XSS")
	}
	if isXSSAttempt("normal text") {
		t.Fatal("should not detect normal text as XSS")
	}
	if !isXSSAttempt("<img onerror=alert(1) src=x>") {
		t.Fatal("should detect event handler as XSS")
	}
	if !isXSSAttempt("<a href=\"javascript:alert(1)\">link</a>") {
		t.Fatal("should detect javascript: URL as XSS")
	}
}

func TestSanitizeString(t *testing.T) {
	input := "<script>alert('xss')</script>"
	result := SanitizeString(input)
	if strings.Contains(result, "<script>") {
		t.Fatal("sanitized string should not contain script tags")
	}
}

func TestSanitizeJSONResponse(t *testing.T) {
	data := map[string]interface{}{
		"name": "<script>alert(1)</script>",
		"nested": map[string]interface{}{
			"desc": "<img src=x onerror=alert(1)>",
		},
		"count": 42,
	}

	result := SanitizeJSONResponse(data)
	resultMap, ok := result.(map[string]interface{})
	if !ok {
		t.Fatal("result should be a map")
	}

	if strings.Contains(resultMap["name"].(string), "<script>") {
		t.Fatal("name should be sanitized")
	}
	if resultMap["count"].(int) != 42 {
		t.Fatal("count should remain unchanged")
	}
}

func TestAddSecurityHeaders(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(AddSecurityHeaders())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	headers := w.Header()
	expectedHeaders := []string{
		"X-Frame-Options",
		"X-XSS-Protection",
		"X-Content-Type-Options",
		"Referrer-Policy",
		"Strict-Transport-Security",
		"Content-Security-Policy",
		"Permissions-Policy",
		"Cross-Origin-Resource-Policy",
		"Cross-Origin-Opener-Policy",
	}

	for _, h := range expectedHeaders {
		if headers.Get(h) == "" {
			t.Errorf("expected header %s to be set", h)
		}
	}
}

func TestEscapeHTML(t *testing.T) {
	input := "<script>alert('xss')</script>"
	result := EscapeHTML(input)
	if result != "&lt;script&gt;alert(&#39;xss&#39;)&lt;/script&gt;" {
		t.Fatalf("unexpected escaped result: %s", result)
	}
}

func TestUnescapeHTML(t *testing.T) {
	input := "&lt;script&gt;alert(1)&lt;/script&gt;"
	result := UnescapeHTML(input)
	if result != "<script>alert(1)</script>" {
		t.Fatalf("unexpected unescaped result: %s", result)
	}
}

func TestSQLInjectionCheck(t *testing.T) {
	tests := []struct {
		input      string
		detected   bool
		severity   int
	}{
		{"SELECT * FROM users", true, 0},
		{"1' OR '1'='1", true, 0},
		{"normal input", false, 0},
		{"DROP TABLE users", true, 0},
		{"admin@example.com", false, 0},
		{"'; DROP TABLE users; --", true, 0},
		{"<script>alert(1)</script>", false, 0},
		{"UNION SELECT * FROM passwords", true, 0},
	}

	for _, tt := range tests {
		result := checkSQLInjection(tt.input, &defaultSQLInjectionConfig)
		if result.Detected != tt.detected {
			t.Errorf("checkSQLInjection(%q): detected=%v, want %v", tt.input, result.Detected, tt.detected)
		}
	}
}

func TestCheckSQLInjection(t *testing.T) {
	detected, patterns, severity := CheckSQLInjection("SELECT * FROM users WHERE id = 1")
	if !detected {
		t.Fatal("should detect SQL injection in SELECT query")
	}
	if len(patterns) == 0 {
		t.Fatal("should return at least one pattern")
	}
	if severity <= 0 {
		t.Fatal("severity should be > 0")
	}

	detected, _, _ = CheckSQLInjection("normal user input")
	if detected {
		t.Fatal("should not detect SQL injection in normal input")
	}
}

func TestSQLSanitize(t *testing.T) {
	input := "SELECT * FROM users; DROP TABLE passwords; --"
	result := SQLSanitize(input)
	if strings.Contains(result, "SELECT") {
		t.Fatal("sanitized result should not contain SELECT")
	}
	if strings.Contains(result, "DROP") {
		t.Fatal("sanitized result should not contain DROP")
	}
}

func TestSQLQueryValidator(t *testing.T) {
	validator := NewSQLQueryValidator()
	validator.AddAllowedTable("users")
	validator.AddAllowedColumn("users", "name")

	err := validator.ValidateQuery("users", "name", "John")
	if err != nil {
		t.Fatalf("valid query should not error: %v", err)
	}

	err = validator.ValidateQuery("unknown_table", "name", "value")
	if err == nil {
		t.Fatal("should error for unknown table")
	}

	err = validator.ValidateQuery("users", "name", "SELECT * FROM passwords")
	if err == nil {
		t.Fatal("should detect SQL injection in value")
	}
}

func TestClamp(t *testing.T) {
	if clamp(5, 10) != 5 {
		t.Fatalf("clamp(5, 10) should return 5")
	}
	if clamp(15, 10) != 10 {
		t.Fatalf("clamp(15, 10) should return 10")
	}
	if clamp(10, 10) != 10 {
		t.Fatalf("clamp(10, 10) should return 10")
	}
}

func TestReadCloser(t *testing.T) {
	data := []byte("test data")
	rc := createBodyReaderForSignature(data)

	read, err := io.ReadAll(rc)
	if err != nil {
		t.Fatalf("ReadAll failed: %v", err)
	}
	if string(read) != string(data) {
		t.Fatalf("read data mismatch: got %s, expected %s", read, data)
	}

	if rc.Close() != nil {
		t.Fatal("Close should return nil")
	}
}

func TestGetXSSReport(t *testing.T) {
	r := setupSecurityTestRouter()
	r.GET("/test", func(c *gin.Context) {
		report := GetXSSReport(c, "param1", "<script>alert(1)</script>")
		if report.Blocked != true {
			t.Fatal("XSS report should indicate blocked")
		}
		if report.Field != "param1" {
			t.Fatalf("expected field 'param1', got %s", report.Field)
		}
		c.JSON(200, gin.H{"report": report})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestNewSignatureConfig(t *testing.T) {
	cfg := NewSignatureConfig("my-secret")
	if cfg.SecretKey != "my-secret" {
		t.Fatalf("expected secret 'my-secret', got %s", cfg.SecretKey)
	}
	if cfg.Algorithm != "SHA256" {
		t.Fatalf("expected SHA256, got %s", cfg.Algorithm)
	}
	if cfg.TimestampTolerance != 5*time.Minute {
		t.Fatalf("expected 5min tolerance, got %v", cfg.TimestampTolerance)
	}
}

func TestVerifyTimestamp(t *testing.T) {
	now := time.Now().Unix()
	err := verifyTimestamp(now, 5*time.Minute)
	if err != nil {
		t.Fatalf("current timestamp should be valid: %v", err)
	}

	err = verifyTimestamp(now-600, 5*time.Minute)
	if err == nil {
		t.Fatal("old timestamp should be invalid")
	}
}

func TestXSSMaxLength(t *testing.T) {
	cfg := XSSConfig{
		MaxLength: 10,
	}
	longInput := "A very long input that exceeds the max length"
	result := sanitizeHTML(longInput, cfg)
	if len(result.Value) > 10 {
		t.Fatalf("result should be truncated to 10 chars, got %d", len(result.Value))
	}
}

func TestSQLInjectionMaxQueryLength(t *testing.T) {
	cfg := SQLInjectionConfig{
		MaxQueryLength: 10,
	}
	longInput := "SELECT * FROM users WHERE id = 1 OR 1=1 -- very long query"
	result := checkSQLInjection(longInput, &cfg)
	if result.Detected {
		t.Fatal("truncated query should not detect patterns")
	}
}

func TestCSRFRedisStore_WithoutRedis(t *testing.T) {
	store := NewCSRFRedisStore(time.Hour)

	err := store.Store("token", "session")
	if err == nil {
		t.Fatal("should error when redis client is nil")
	}

	_, err = store.Verify("token", "session")
	if err == nil {
		t.Fatal("should error when redis client is nil")
	}

	err = store.Delete("session")
	if err != nil {
		t.Fatalf("Delete should not error when redis is nil: %v", err)
	}
}

func TestSanitizeHTMLWithAllowList(t *testing.T) {
	cfg := defaultXSSConfig

	input := "<p>Hello <b>world</b></p>"
	result := sanitizeHTMLWithAllowList(input, cfg)
	if !strings.Contains(result, "<p>") {
		t.Fatal("should allow <p> tag")
	}

	input2 := "<script>alert(1)</script><p>safe</p>"
	result2 := sanitizeHTMLWithAllowList(input2, cfg)
	if strings.Contains(result2, "<script>") {
		t.Fatal("should remove <script> tag")
	}
	if !strings.Contains(result2, "<p>") || !strings.Contains(result2, "safe") {
		t.Fatal("should keep safe content")
	}
}

func TestSignatureConfigDefaults(t *testing.T) {
	cfg := defaultSignatureConfig
	if cfg.SignatureHeader != "X-Signature" {
		t.Fatalf("expected X-Signature, got %s", cfg.SignatureHeader)
	}
	if !cfg.RequireTimestamp {
		t.Fatal("RequireTimestamp should be true by default")
	}
	if !cfg.RequireNonce {
		t.Fatal("RequireNonce should be true by default")
	}
}

func TestCSRFDefaultConfig(t *testing.T) {
	cfg := defaultCSRFConfig
	if cfg.TokenLength != 32 {
		t.Fatalf("expected TokenLength 32, got %d", cfg.TokenLength)
	}
	if cfg.TokenExpiration != time.Hour {
		t.Fatalf("expected TokenExpiration 1h, got %v", cfg.TokenExpiration)
	}
}

func TestXSSDefaultConfig(t *testing.T) {
	cfg := defaultXSSConfig
	if !cfg.EnableLog {
		t.Fatal("EnableLog should be true by default")
	}
	if cfg.MaxLength != 10000 {
		t.Fatalf("expected MaxLength 10000, got %d", cfg.MaxLength)
	}
}

func TestSecurityHeadersMiddleware(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(SecurityHeadersMiddleware())
	r.GET("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/test", nil)
	r.ServeHTTP(w, req)

	if w.Header().Get("X-Frame-Options") != "SAMEORIGIN" {
		t.Fatal("X-Frame-Options should be SAMEORIGIN")
	}
}

func TestSignatureVerification_WithQueryString(t *testing.T) {
	r := setupSecurityTestRouter()
	cfg := defaultSignatureConfig
	r.Use(SignatureVerification(cfg))
	r.GET("/api/search", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	timestamp := time.Now().Unix()
	nonce := "unique-nonce-for-test"
	signature := GenerateSignature(
		cfg.SecretKey,
		"GET",
		"/api/search",
		"q=test&page=1",
		timestamp,
		nonce,
		nil,
	)

	w := httptest.NewRecorder()
	req, _ := http.NewRequest("GET", "/api/search?q=test&page=1", nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", timestamp))
	req.Header.Set("X-Nonce", nonce)
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d. Body: %s", w.Code, w.Body.String())
	}
}

func TestSignatureVersion(t *testing.T) {
	if SignatureVersion != "1" {
		t.Fatalf("expected SignatureVersion 1, got %s", SignatureVersion)
	}
}

func TestXSSFilterWithBlockAttributes(t *testing.T) {
	cfg := XSSConfig{
		BlockAttributes: true,
		MaxLength:       10000,
	}

	input := `<div data-custom="value">content</div>`
	result := sanitizeHTML(input, cfg)
	if strings.Contains(result.Value, "data-custom") {
		t.Fatal("should remove data-* attributes when BlockAttributes is true")
	}
}

func TestXSSFilterExpressionPattern(t *testing.T) {
	cfg := defaultXSSConfig

	input := `<div style="expression(alert(1))">content</div>`
	result := sanitizeHTML(input, cfg)
	if strings.Contains(result.Value, "expression") {
		t.Fatal("should remove expression patterns")
	}
}

func TestXSSFilterXMLPattern(t *testing.T) {
	cfg := defaultXSSConfig

	input := `<?xml version="1.0"?><root>content</root>`
	result := sanitizeHTML(input, cfg)
	if strings.Contains(result.Value, "<?xml") {
		t.Fatal("should remove XML processing instructions")
	}
}

func TestSanitizeRequestBody_NonJSON(t *testing.T) {
	r := setupSecurityTestRouter()
	r.Use(func(c *gin.Context) {
		sanitizeRequestBody(c, defaultXSSConfig)
		c.Next()
	})
	r.POST("/test", func(c *gin.Context) {
		c.JSON(200, gin.H{"status": "ok"})
	})

	w := httptest.NewRecorder()
	body := "name=test&value=<script>alert(1)</script>"
	req, _ := http.NewRequest("POST", "/test", strings.NewReader(body))
	req.Header.Set("Content-Type", "application/x-www-form-urlencoded")
	r.ServeHTTP(w, req)

	if w.Code != 200 {
		t.Fatalf("expected 200, got %d", w.Code)
	}
}

func TestCSRFMemoryStore_ExpiredToken(t *testing.T) {
	store := &CSRFMemoryStore{
		tokens: make(map[string]map[string]time.Time),
	}

	token := generateToken(32)
	hashedToken := hashToken(token)
	store.tokens["session1"] = map[string]time.Time{
		hashedToken: time.Now().Add(-2 * time.Hour),
	}

	valid, err := store.Verify(token, "session1")
	if err != nil {
		t.Fatalf("Verify should not error: %v", err)
	}
	if valid {
		t.Fatal("expired token should not be valid")
	}
}

func TestReadCloserMultipleReads(t *testing.T) {
	data := []byte("test data for read closer")
	rc := createBodyReaderForSignature(data)

	buf := make([]byte, 4)
	n, err := rc.Read(buf)
	if err != nil {
		t.Fatalf("first read failed: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}

	n, err = rc.Read(buf)
	if err != nil {
		t.Fatalf("second read failed: %v", err)
	}
	if n != 4 {
		t.Fatalf("expected 4 bytes, got %d", n)
	}
}

func TestCSRFMemoryStore_DeleteNonExistent(t *testing.T) {
	store := NewCSRFMemoryStore()
	err := store.Delete("non-existent-session")
	if err != nil {
		t.Fatalf("deleting non-existent session should not error: %v", err)
	}
}

func TestVerifyNonce_LengthValidation(t *testing.T) {
	err := verifyNonce("short", time.Hour)
	if err == nil {
		t.Fatal("nonce shorter than 8 chars should error")
	}

	longNonce := strings.Repeat("a", 65)
	err = verifyNonce(longNonce, time.Hour)
	if err == nil {
		t.Fatal("nonce longer than 64 chars should error")
	}
}

func TestIsSafeMethod_CustomMethods(t *testing.T) {
	methods := []string{"GET", "POST"}
	if !isSafeMethod("GET", methods) {
		t.Fatal("GET should be in custom safe methods")
	}
	if !isSafeMethod("POST", methods) {
		t.Fatal("POST should be in custom safe methods")
	}
	if isSafeMethod("DELETE", methods) {
		t.Fatal("DELETE should not be in custom safe methods")
	}
}

func TestNonceCache(t *testing.T) {
	cache := &nonceCache{
		used: make(map[string]time.Time),
	}

	if cache.isUsed("test-nonce") {
		t.Fatal("nonce should not be used initially")
	}

	cache.markUsed("test-nonce")
	if !cache.isUsed("test-nonce") {
		t.Fatal("nonce should be used after marking")
	}
}