package middleware

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func init() {
	gin.SetMode(gin.TestMode)
}

func TestEnhancedSignatureConfigDefaults(t *testing.T) {
	config := defaultEnhancedSignatureConfig

	if config.SecretKey == "" {
		t.Error("SecretKey should not be empty")
	}
	if config.Algorithm != "SHA256" {
		t.Error("Default algorithm should be SHA256")
	}
	if config.TimestampTolerance != 5*time.Minute {
		t.Error("Default timestamp tolerance should be 5 minutes")
	}
	if !config.RequireTimestamp {
		t.Error("Timestamp should be required by default")
	}
	if !config.RequireNonce {
		t.Error("Nonce should be required by default")
	}
}

func TestNewEnhancedSignatureConfig(t *testing.T) {
	secretKey := "test-secret-key"
	config := NewEnhancedSignatureConfig(secretKey)

	if config.SecretKey != secretKey {
		t.Error("SecretKey should match")
	}
	if config.Algorithm != "SHA256" {
		t.Error("Algorithm should be SHA256")
	}
	if config.RequireNonce != true {
		t.Error("Nonce should be required")
	}
}

func TestEnhancedSignatureExcludedPaths(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EnhancedSignatureVerification())
	router.GET("/health", func(c *gin.Context) {
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/health", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Excluded path should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureMissingHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.Use(EnhancedSignatureVerification())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Missing signature should return 401, got %d", w.Code)
	}
}

func TestEnhancedSignatureValid(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-12345")

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.POST("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "test-nonce-123456789"

	signature := GenerateEnhancedSignature(
		config.SecretKey,
		"POST",
		"/api/test",
		"",
		timestamp,
		nonce,
		body,
	)

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid signature should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureInvalidSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-12345")

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.POST("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "test-nonce-123456789"

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", "invalid-signature")
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Invalid signature should return 401, got %d", w.Code)
	}
}

func TestEnhancedSignatureExpiredTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-12345")
	config.TimestampTolerance = 5 * time.Minute

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.GET("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	timestamp := time.Now().Add(-10 * time.Minute).Unix()
	nonce := "test-nonce-123456789"

	signature := GenerateEnhancedSignature(
		config.SecretKey,
		"GET",
		"/api/test",
		"",
		timestamp,
		nonce,
		nil,
	)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expired timestamp should return 401, got %d", w.Code)
	}
}

func TestEnhancedSignatureReplayAttack(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-12345")

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.POST("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "unique-nonce-replay-test"

	signature := GenerateEnhancedSignature(
		config.SecretKey,
		"POST",
		"/api/test",
		"",
		timestamp,
		nonce,
		body,
	)

	req1 := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req1.Header.Set("Content-Type", "application/json")
	req1.Header.Set("X-Signature", signature)
	req1.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req1.Header.Set("X-Nonce", nonce)

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request should succeed, got %d", w1.Code)
	}

	req2 := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Signature", signature)
	req2.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req2.Header.Set("X-Nonce", nonce)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Replay request should fail with 401, got %d", w2.Code)
	}
}

func TestEnhancedNonceCache(t *testing.T) {
	cache := &enhancedNonceCache{
		records: make(map[string]*nonceRecord),
		limit:   100,
	}

	nonce := "test-nonce-12345"

	if cache.isUsed(nonce) {
		t.Error("New nonce should not be marked as used")
	}

	cache.markUsed(nonce)

	if !cache.isUsed(nonce) {
		t.Error("Nonce should be marked as used after markUsed")
	}
}

func TestEnhancedNonceCacheCleanup(t *testing.T) {
	cache := &enhancedNonceCache{
		records: make(map[string]*nonceRecord),
		limit:   100,
	}

	for i := 0; i < 10; i++ {
		nonce := "test-nonce-" + strconv.Itoa(i)
		nonceRecord := &nonceRecord{
			timestamp:  time.Now().Add(-25 * time.Hour),
			hashedNonce: hashNonce(nonce),
			count:       1,
		}
		cache.records[hashNonce(nonce)] = nonceRecord
	}

	cache.cleanup()

	for _, record := range cache.records {
		if time.Since(record.timestamp) > 24*time.Hour {
			t.Error("Old nonces should be cleaned up")
		}
	}
}

func TestGenerateEnhancedNonce(t *testing.T) {
	nonce, err := GenerateEnhancedNonce(16)
	if err != nil {
		t.Fatalf("GenerateEnhancedNonce failed: %v", err)
	}

	if len(nonce) < 8 {
		t.Error("Nonce length should be at least 8")
	}

	nonce2, _ := GenerateEnhancedNonce(16)
	if nonce == nonce2 {
		t.Error("Generated nonces should be unique")
	}
}

func TestGenerateSecureNonce(t *testing.T) {
	nonce, err := GenerateSecureNonce(16)
	if err != nil {
		t.Fatalf("GenerateSecureNonce failed: %v", err)
	}

	if len(nonce) != 16 {
		t.Errorf("Nonce length should be 16, got %d", len(nonce))
	}

	for _, c := range nonce {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			t.Error("Nonce should only contain alphanumeric, dash, and underscore")
		}
	}
}

func TestGenerateSecureNonceLengthValidation(t *testing.T) {
	nonce, _ := GenerateSecureNonce(4)
	if len(nonce) != 16 {
		t.Errorf("Nonce should default to 16 when less, got %d", len(nonce))
	}

	nonce, _ = GenerateSecureNonce(100)
	if len(nonce) != 64 {
		t.Errorf("Nonce should be capped at 64, got %d", len(nonce))
	}
}

func TestHashNonce(t *testing.T) {
	nonce := "test-nonce-12345"
	hash1 := hashNonce(nonce)
	hash2 := hashNonce(nonce)

	if hash1 != hash2 {
		t.Error("Same nonce should produce same hash")
	}

	hash3 := hashNonce("different-nonce")
	if hash1 == hash3 {
		t.Error("Different nonces should produce different hashes")
	}
}

func TestIsValidNonceFormat(t *testing.T) {
	validNonces := []string{
		"abc123",
		"ABC-123_xyz",
		"a1-b2_c3",
		"123456789012345678901234567890",
	}

	for _, nonce := range validNonces {
		if !isValidNonceFormat(nonce) {
			t.Errorf("Nonce '%s' should be valid", nonce)
		}
	}

	invalidNonces := []string{
		"",
		"abc@123",
		"abc 123",
		"abc!123",
		"abc#123",
	}

	for _, nonce := range invalidNonces {
		if isValidNonceFormat(nonce) {
			t.Errorf("Nonce '%s' should be invalid", nonce)
		}
	}
}

func TestCalculateEnhancedSignature(t *testing.T) {
	secretKey := "test-secret"
	method := "POST"
	path := "/api/test"
	query := "foo=bar&baz=qux"
	timestamp := int64(1234567890)
	nonce := "test-nonce"
	bodyHash := "abc123"

	sig := calculateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, bodyHash)

	if sig == "" {
		t.Error("Signature should not be empty")
	}

	sig2 := calculateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, bodyHash)
	if sig != sig2 {
		t.Error("Same inputs should produce same signature")
	}
}

func TestCalculateDoubleSignature(t *testing.T) {
	secretKey := "test-secret"

	sig := calculateDoubleSignature(secretKey, "param1", "param2", "param3")

	if sig == "" {
		t.Error("Double signature should not be empty")
	}
}

func TestBuildEnhancedStringToSign(t *testing.T) {
	method := "POST"
	path := "/api/test"
	query := "foo=bar"
	timestamp := int64(1234567890)
	nonce := "test-nonce"
	bodyHash := "abc123"

	stringToSign := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash)

	if stringToSign == "" {
		t.Error("String to sign should not be empty")
	}

	if !contains(stringToSign, "POST") {
		t.Error("String to sign should contain method")
	}
	if !contains(stringToSign, "/api/test") {
		t.Error("String to sign should contain path")
	}
	if !contains(stringToSign, "foo=bar") {
		t.Error("String to sign should contain sorted query")
	}
	if !contains(stringToSign, "1234567890") {
		t.Error("String to sign should contain timestamp")
	}
}

func TestSortQueryStringEnhanced(t *testing.T) {
	query := "z=3&a=1&m=2"
	sorted := sortQueryStringEnhanced(query)

	expected := "a=1&m=2&z=3"
	if sorted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sorted)
	}
}

func TestHashBodyEnhanced(t *testing.T) {
	body := []byte(`{"test": "data"}`)

	hash1 := hashBodyEnhanced(body)
	hash2 := hashBodyEnhanced(body)

	if hash1 != hash2 {
		t.Error("Same body should produce same hash")
	}

	hash3 := hashBodyEnhanced([]byte(`{"different": "data"}`))
	if hash1 == hash3 {
		t.Error("Different bodies should produce different hashes")
	}

	emptyHash := hashBodyEnhanced(nil)
	if emptyHash != "" {
		t.Error("Empty body should produce empty hash")
	}
}

func TestComputeBodyIntegrity(t *testing.T) {
	body := []byte(`{"test": "data"}`)

	integrity := computeBodyIntegrity(body)

	if integrity == "" {
		t.Error("Body integrity should not be empty")
	}
}

func TestVerifyBodyIntegrity(t *testing.T) {
	body := []byte(`{"test": "data"}`)
	integrity := computeBodyIntegrity(body)

	if !verifyBodyIntegrity(body, integrity) {
		t.Error("Body should verify against its integrity")
	}

	if verifyBodyIntegrity([]byte(`{"modified": "data"}`), integrity) {
		t.Error("Modified body should not verify")
	}

	if !verifyBodyIntegrity(body, "") {
		t.Error("Empty integrity should pass verification")
	}
}

func TestVerifyEnhancedTimestamp(t *testing.T) {
	tolerance := 5 * time.Minute

	err := verifyEnhancedTimestamp(time.Now().Unix(), tolerance)
	if err != nil {
		t.Error("Current timestamp should be valid")
	}

	err = verifyEnhancedTimestamp(time.Now().Add(-10*time.Minute).Unix(), tolerance)
	if err == nil {
		t.Error("Expired timestamp should fail")
	}
}

func TestVerifyEnhancedNonce(t *testing.T) {
	config := NewEnhancedSignatureConfig("test-secret")

	nonce := "valid-nonce-12345678"
	err := verifyEnhancedNonce(nonce, config)
	if err != nil {
		t.Errorf("Valid nonce should pass: %v", err)
	}

	shortNonce := "short"
	err = verifyEnhancedNonce(shortNonce, config)
	if err == nil {
		t.Error("Short nonce should fail")
	}

	longNonce := "this-nonce-is-way-too-long-and-should-fail-because-it-exceeds-maximum-length"
	err = verifyEnhancedNonce(longNonce, config)
	if err == nil {
		t.Error("Long nonce should fail")
	}
}

func TestSecureCompareEnhanced(t *testing.T) {
	if !secureCompareEnhanced("test", "test") {
		t.Error("Equal strings should compare as equal")
	}

	if secureCompareEnhanced("test", "different") {
		t.Error("Different strings should not compare as equal")
	}

	if secureCompareEnhanced("test", "tes") {
		t.Error("Different length strings should not compare as equal")
	}
}

func TestGenerateEnhancedSignature(t *testing.T) {
	secretKey := "test-secret"
	method := "POST"
	path := "/api/test"
	query := ""
	timestamp := time.Now().Unix()
	nonce := "test-nonce-12345"
	body := []byte(`{"test": "data"}`)

	signature := GenerateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, body)

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	signature2 := GenerateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, body)
	if signature != signature2 {
		t.Error("Same inputs should produce same signature")
	}
}

func TestValidateEnhancedSignature(t *testing.T) {
	secretKey := "test-secret-key-12345"

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "validation-test-nonce"

	signature := GenerateEnhancedSignature(
		secretKey,
		"POST",
		"/api/test",
		"",
		timestamp,
		nonce,
		body,
	)

	gin.SetMode(gin.TestMode)
	router := gin.New()
	router.POST("/api/test", func(c *gin.Context) {
		result := ValidateEnhancedSignature(c, secretKey)
		if !result.Valid {
			c.String(401, "Invalid")
		}
		c.String(200, "Valid")
	})

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid signature validation should return 200, got %d", w.Code)
	}
}

func TestGetEnhancedSignatureInfo(t *testing.T) {
	info := GetEnhancedSignatureInfo()

	if info.Algorithm != "SHA256" {
		t.Error("Algorithm should be SHA256")
	}
	if info.Version != "2.0" {
		t.Error("Version should be 2.0")
	}
	if !info.NonceRequired {
		t.Error("Nonce should be required")
	}
	if !info.Features.ReplayProtection {
		t.Error("Replay protection should be enabled")
	}
}

func TestBuildEnhancedSignatureInput(t *testing.T) {
	secretKey := "test-secret"
	method := "POST"
	path := "/api/test"
	query := ""
	timestamp := time.Now().Unix()
	body := []byte(`{"test": "data"}`)

	signature, err := BuildEnhancedSignatureInput(secretKey, method, path, query, timestamp, "", body)
	if err != nil {
		t.Fatalf("BuildEnhancedSignatureInput failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestGenerateTimestampWithMillis(t *testing.T) {
	timestamp := GenerateTimestampWithMillis()

	if timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}

	expectedLength := len(strconv.FormatInt(time.Now().UnixMilli(), 10))
	actualLength := len(strconv.FormatInt(timestamp, 10))

	if actualLength < expectedLength-1 || actualLength > expectedLength+1 {
		t.Error("Timestamp should be in milliseconds")
	}
}

func TestVerifyTimestampMillis(t *testing.T) {
	tolerance := 1 * time.Second

	err := VerifyTimestampMillis(time.Now().UnixMilli(), tolerance)
	if err != nil {
		t.Error("Current timestamp should be valid")
	}

	err = VerifyTimestampMillis(time.Now().Add(-5*time.Second).UnixMilli(), tolerance)
	if err == nil {
		t.Error("Expired timestamp should fail")
	}
}

func TestGenerateRequestID(t *testing.T) {
	id1 := GenerateRequestID()
	id2 := GenerateRequestID()

	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}

	if len(id1) < 10 {
		t.Error("Request ID should be reasonably long")
	}
}

func TestExtractRequestID(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var extractedID string
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		extractedID = ExtractRequestID(c)
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedID == "" {
		t.Error("Request ID should be extracted or generated")
	}
}

func TestExtractRequestIDWithHeader(t *testing.T) {
	gin.SetMode(gin.TestMode)

	var extractedID string
	router := gin.New()
	router.GET("/test", func(c *gin.Context) {
		extractedID = ExtractRequestID(c)
		c.String(200, "OK")
	})

	customID := "custom-request-id-12345"
	req := httptest.NewRequest("GET", "/test", nil)
	req.Header.Set("X-Request-ID", customID)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if extractedID != customID {
		t.Errorf("Should extract custom request ID, got %s", extractedID)
	}
}

func TestCreateSignatureMiddlewareChain(t *testing.T) {
	chain := CreateSignatureMiddlewareChain()

	if len(chain) == 0 {
		t.Error("Middleware chain should not be empty")
	}
}

func TestRequireEnhancedSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(RequireEnhancedSignature())
	router.GET("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	timestamp := time.Now().Unix()
	nonce := "require-test-nonce-12345"
	config := defaultEnhancedSignatureConfig

	signature := GenerateEnhancedSignature(
		config.SecretKey,
		"GET",
		"/api/test",
		"",
		timestamp,
		nonce,
		nil,
	)

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid signature should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureResult(t *testing.T) {
	result := EnhancedSignatureResult{
		Valid:       true,
		Reason:      "test passed",
		Timestamp:   time.Now().Unix(),
		Nonce:       "test-nonce",
		Signature:   "test-signature",
		Sequence:    1,
		ElapsedTime: time.Millisecond * 100,
		ClientIP:    "127.0.0.1",
		RequestPath: "/api/test",
	}

	if !result.Valid {
		t.Error("Result should be valid")
	}
	if result.ElapsedTime == 0 {
		t.Error("Elapsed time should be set")
	}
}

func TestEnhancedSignatureWithQueryString(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-12345")

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.GET("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	timestamp := time.Now().Unix()
	nonce := "query-test-nonce-12345"
	query := "foo=bar&baz=qux"

	signature := GenerateEnhancedSignature(
		config.SecretKey,
		"GET",
		"/api/test",
		query,
		timestamp,
		nonce,
		nil,
	)

	req := httptest.NewRequest("GET", "/api/test?"+query, nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid signature with query should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureRateLimitPerIP(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-12345")
	config.EnableRateLimitPerIP = true
	config.RateLimitPerIPLimit = 3
	config.RateLimitPerIPWindow = time.Minute

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.GET("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	timestamp := time.Now().Unix()
	nonce := "rate-limit-test-nonce"

	for i := 0; i < 3; i++ {
		signature := GenerateEnhancedSignature(
			config.SecretKey,
			"GET",
			"/api/test",
			"",
			timestamp,
			nonce+strconv.Itoa(i),
			nil,
		)

		req := httptest.NewRequest("GET", "/api/test", nil)
		req.Header.Set("X-Signature", signature)
		req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
		req.Header.Set("X-Nonce", nonce+strconv.Itoa(i))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got %d", i+1, w.Code)
		}
	}
}

func TestEnhancedSignatureSequenceState(t *testing.T) {
	state := &enhancedSignatureState{
		sequenceCounters: make(map[string]int64),
		ipRequestCounts:  make(map[string]*ipRequestCounter),
	}

	clientID := "test-client"

	seq1 := state.getNextSequence(clientID)
	if seq1 != 0 {
		t.Errorf("First sequence should be 0, got %d", seq1)
	}

	seq2 := state.getNextSequence(clientID)
	if seq2 != 1 {
		t.Errorf("Second sequence should be 1, got %d", seq2)
	}

	if !state.validateSequence(clientID, 2) {
		t.Error("Sequence 2 should be valid")
	}

	if state.validateSequence(clientID, 5) {
		t.Error("Sequence 5 should not be valid")
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
