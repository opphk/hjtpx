package middleware

import (
	"bytes"
	"encoding/hex"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestEnhancedSignatureGeneration(t *testing.T) {
	secretKey := "test-secret-key-12345"
	method := "POST"
	path := "/api/test"
	query := "key=value"
	timestamp := time.Now().Unix()
	nonce := "test-nonce-12345"
	body := []byte(`{"test": "data"}`)

	signature := GenerateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, body)

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	if len(signature) != 64 {
		t.Errorf("Expected SHA256 signature length 64, got %d", len(signature))
	}
}

func TestEnhancedSignatureVerification(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey := "test-secret-key-12345"

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:          secretKey,
		RequireTimestamp:   true,
		RequireNonce:       true,
		TimestampTolerance: 5 * time.Minute,
	}))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	method := "POST"
	path := "/test"
	query := "key=value"
	timestamp := time.Now().Unix()
	nonce := "test-nonce-" + fmt.Sprintf("%d", timestamp)
	body := []byte(`{"test": "data"}`)

	signature := GenerateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, body)

	req := httptest.NewRequest(method, path+"?"+query, nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK, got %d", w.Code)
	}
}

func TestEnhancedSignatureMissingSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:        "test-secret",
		RequireTimestamp: true,
		RequireNonce:     true,
	}))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("POST", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401, got %d", w.Code)
	}
}

func TestEnhancedSignatureInvalidTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey := "test-secret"

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:          secretKey,
		RequireTimestamp:   true,
		RequireNonce:       true,
		TimestampTolerance: 5 * time.Minute,
	}))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	oldTimestamp := time.Now().Add(-10 * time.Minute).Unix()
	nonce := "test-nonce"
	signature := GenerateEnhancedSignature(secretKey, "POST", "/test", "", oldTimestamp, nonce, nil)

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(oldTimestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Expected status 401 for expired timestamp, got %d", w.Code)
	}
}

func TestEnhancedSignatureReplayDetection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey := "test-secret"

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:          secretKey,
		RequireTimestamp:   true,
		RequireNonce:       true,
		TimestampTolerance: 5 * time.Minute,
	}))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	timestamp := time.Now().Unix()
	nonce := "unique-nonce-replay-test"
	signature := GenerateEnhancedSignature(secretKey, "POST", "/test", "", timestamp, nonce, nil)

	req1 := httptest.NewRequest("POST", "/test", nil)
	req1.Header.Set("X-Signature", signature)
	req1.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req1.Header.Set("X-Nonce", nonce)

	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("First request should succeed, got %d", w1.Code)
	}

	req2 := httptest.NewRequest("POST", "/test", nil)
	req2.Header.Set("X-Signature", signature)
	req2.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req2.Header.Set("X-Nonce", nonce)

	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("Replay request should be blocked, got %d", w2.Code)
	}
}

func TestEnhancedSignatureNonceFormat(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		nonce         string
		shouldSucceed bool
	}{
		{
			name:          "Valid alphanumeric nonce",
			nonce:         "ValidNonce123-_",
			shouldSucceed: true,
		},
		{
			name:          "Empty nonce",
			nonce:         "",
			shouldSucceed: false,
		},
		{
			name:          "Nonce too short",
			nonce:         "short",
			shouldSucceed: false,
		},
		{
			name:          "Nonce with invalid characters",
			nonce:         "invalid@nonce#",
			shouldSucceed: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			secretKey := "test-secret"

			router := gin.New()
			router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
				SecretKey:          secretKey,
				RequireTimestamp:   true,
				RequireNonce:       true,
				TimestampTolerance: 5 * time.Minute,
				MinNonceLength:     8,
				MaxNonceLength:     64,
			}))

			router.POST("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			timestamp := time.Now().Unix()
			var signature string
			if tc.nonce != "" {
				signature = GenerateEnhancedSignature(secretKey, "POST", "/test", "", timestamp, tc.nonce, nil)
			}

			req := httptest.NewRequest("POST", "/test", nil)
			req.Header.Set("X-Signature", signature)
			req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
			req.Header.Set("X-Nonce", tc.nonce)

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.shouldSucceed && w.Code != http.StatusOK {
				t.Errorf("Expected success for nonce: %s, got %d", tc.name, w.Code)
			}
			if !tc.shouldSucceed && w.Code == http.StatusOK {
				t.Errorf("Expected failure for nonce: %s", tc.name)
			}
		})
	}
}

func TestEnhancedSignatureBodyIntegrity(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey := "test-secret"

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:          secretKey,
		RequireTimestamp:   true,
		RequireNonce:       true,
		TimestampTolerance: 5 * time.Minute,
		EnableIntegrityCheck: true,
	}))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	timestamp := time.Now().Unix()
	nonce := "test-nonce-body"
	body := []byte(`{"sensitive": "data"}`)
	signature := GenerateEnhancedSignature(secretKey, "POST", "/test", "", timestamp, nonce, body)

	integrity := computeBodyIntegrity(body)

	req := httptest.NewRequest("POST", "/test", bytesFromString(string(body)))
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Body-Integrity", integrity)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK with valid body integrity, got %d", w.Code)
	}
}

func TestGenerateSecureNonce(t *testing.T) {
	nonce, err := GenerateSecureNonce(16)
	if err != nil {
		t.Errorf("Unexpected error: %v", err)
	}

	if len(nonce) != 16 {
		t.Errorf("Expected nonce length 16, got %d", len(nonce))
	}

	nonce2, _ := GenerateSecureNonce(32)
	if nonce == nonce2 {
		t.Error("Expected different nonces")
	}
}

func TestGenerateTimestampWithMillis(t *testing.T) {
	ts1 := GenerateTimestampWithMillis()
	ts2 := GenerateTimestampWithMillis()

	if ts1 == 0 {
		t.Error("Expected non-zero timestamp")
	}

	if ts1 > ts2 {
		t.Error("Expected increasing timestamps")
	}
}

func TestVerifyTimestampMillis(t *testing.T) {
	tolerance := 5 * time.Minute

	now := time.Now().UnixMilli()

	err := VerifyTimestampMillis(now, tolerance)
	if err != nil {
		t.Errorf("Expected no error for current timestamp: %v", err)
	}

	oldTimestamp := now - int64(10*time.Minute/time.Millisecond)
	err = VerifyTimestampMillis(oldTimestamp, tolerance)
	if err == nil {
		t.Error("Expected error for old timestamp")
	}
}

func TestEnhancedSignatureInfo(t *testing.T) {
	info := GetEnhancedSignatureInfo()

	if info.Algorithm != "SHA256" {
		t.Errorf("Expected SHA256 algorithm, got %s", info.Algorithm)
	}

	if !info.NonceRequired {
		t.Error("Expected nonce to be required")
	}

	if info.Version != "2.0" {
		t.Errorf("Expected version 2.0, got %s", info.Version)
	}

	if !info.Features.ReplayProtection {
		t.Error("Expected replay protection to be enabled")
	}
}

func TestSecureCompare(t *testing.T) {
	testCases := []struct {
		a        string
		b        string
		expected bool
	}{
		{
			a:        "same",
			b:        "same",
			expected: true,
		},
		{
			a:        "different",
			b:        "different",
			expected: true,
		},
		{
			a:        "a",
			b:        "b",
			expected: false,
		},
		{
			a:        "longer string",
			b:        "longer string",
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.a+"_vs_"+tc.b, func(t *testing.T) {
			result := secureCompareEnhanced(tc.a, tc.b)
			if result != tc.expected {
				t.Errorf("Expected %v for '%s' vs '%s'", tc.expected, tc.a, tc.b)
			}
		})
	}
}

func TestEnhancedSignatureDoubleSignature(t *testing.T) {
	gin.SetMode(gin.TestMode)

	secretKey := "test-secret"

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:            secretKey,
		RequireTimestamp:     true,
		RequireNonce:         true,
		TimestampTolerance:   5 * time.Minute,
		EnableDoubleSignature: true,
	}))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	timestamp := time.Now().Unix()
	nonce := "test-nonce-double"
	method := "POST"
	path := "/test"

	primarySig := GenerateEnhancedSignature(secretKey, method, path, "", timestamp, nonce, nil)
	secondarySig := calculateDoubleSignature(secretKey, method, path, strconv.FormatInt(timestamp, 10), nonce)

	req := httptest.NewRequest(method, path, nil)
	req.Header.Set("X-Signature", primarySig)
	req.Header.Set("X-Signature-Secondary", secondarySig)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Expected status OK with valid double signature, got %d", w.Code)
	}
}

func TestRequestEncryption(t *testing.T) {
	config := defaultRequestEncryptionConfig
	config.Enabled = true
	config.EnablePayloadEncryption = true
	config.EncryptionKey = []byte("12345678901234567890123456789012")

	plaintext := []byte(`{"sensitive": "data"}`)

	encrypted, err := EncryptRequestBody(plaintext, config)
	if err != nil {
		t.Errorf("Encryption failed: %v", err)
	}

	if encrypted.Version != 1 {
		t.Errorf("Expected version 1, got %d", encrypted.Version)
	}

	if encrypted.EncryptedData == "" {
		t.Error("Expected encrypted data")
	}

	decrypted, err := DecryptRequestBody(encrypted, config)
	if err != nil {
		t.Errorf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Expected decrypted data to match original")
	}
}

func TestRequestEncryptionKeyRotation(t *testing.T) {
	config := defaultRequestEncryptionConfig
	config.Enabled = true
	config.EnablePayloadEncryption = true
	config.EncryptionKey = []byte("12345678901234567890123456789012")

	plaintext := []byte(`{"data": "test"}`)

	encrypted, err := EncryptRequestBody(plaintext, config)
	if err != nil {
		t.Errorf("Initial encryption failed: %v", err)
	}

	oldVersion := encrypted.KeyVersion

	err = RotateEncryptionKey(&config)
	if err != nil {
		t.Errorf("Key rotation failed: %v", err)
	}

	if encrypted.KeyVersion >= config.CurrentKeyVersion {
		t.Error("Expected key version to be incremented after rotation")
	}

	decrypted, err := DecryptRequestBody(encrypted, config)
	if err != nil {
		t.Errorf("Decryption with old key failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Expected decrypted data to match original after key rotation")
	}

	if oldVersion >= config.CurrentKeyVersion {
		t.Error("Expected key version to increment")
	}
}

func TestDualSignature(t *testing.T) {
	config := DoubleSignatureConfig{
		Enabled:            true,
		PrimaryAlgorithm:   "SHA256",
		SecondaryAlgorithm: "SHA512",
		PrimaryKey:         []byte("primary-key-12345"),
		SecondaryKey:       []byte("secondary-key-12345"),
		VerifyOrder:        "any",
		RequireBothValid:   false,
	}

	message := []byte("test message")

	primarySig, secondarySig, err := GenerateDualSignature(message, config)
	if err != nil {
		t.Errorf("Signature generation failed: %v", err)
	}

	if primarySig == "" || secondarySig == "" {
		t.Error("Expected non-empty signatures")
	}

	valid1, valid2, err := VerifyDualSignature(message, primarySig, secondarySig, config)
	if err != nil {
		t.Errorf("Signature verification failed: %v", err)
	}

	if !valid1 || !valid2 {
		t.Error("Expected valid signatures")
	}

	valid1, valid2, err = VerifyDualSignature(message, primarySig, "invalid", config)
	if err != nil {
		t.Errorf("Expected no error for invalid secondary: %v", err)
	}
	if valid2 {
		t.Error("Expected invalid secondary signature")
	}
}

func TestAntiReplayBloomFilter(t *testing.T) {
	filter := NewBloomFilter(1000, 7)

	item1 := "unique-item-1"
	item2 := "unique-item-2"

	if filter.Contains(item1) {
		t.Error("New item should not be in filter")
	}

	filter.Add(item1)

	if !filter.Contains(item1) {
		t.Error("Added item should be in filter")
	}

	if filter.Contains(item2) {
		t.Error("Non-added item should not be in filter")
	}
}

func TestAntiReplaySlidingWindow(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := EnhancedAntiReplayConfig{
		WindowSize:           1 * time.Minute,
		MaxRequestsPerWindow: 5,
		EnableSlidingWindow:  true,
		EnableBloomFilter:    true,
	}

	router := gin.New()
	router.Use(EnhancedAntiReplayV2(config))

	router.POST("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	for i := 0; i < 5; i++ {
		req := httptest.NewRequest("POST", "/test", nil)
		req.Header.Set("X-Nonce", fmt.Sprintf("sliding-test-%d", i))
		req.Header.Set("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))

		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code != http.StatusOK {
			t.Errorf("Request %d should succeed, got %d", i+1, w.Code)
		}
	}

	req := httptest.NewRequest("POST", "/test", nil)
	req.Header.Set("X-Nonce", "sliding-test-exceed")
	req.Header.Set("X-Timestamp", strconv.FormatInt(time.Now().Unix(), 10))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusTooManyRequests {
		t.Errorf("Expected rate limit, got %d", w.Code)
	}
}

func TestNonceValidator(t *testing.T) {
	validator := &NonceValidator{
		bloomFilter:      NewBloomFilter(1000, 7),
		redisEnabled:     false,
		nonceCacheTTL:    24 * time.Hour,
		strictValidation: true,
		MaxNonceAge:      15 * time.Minute,
	}

	validNonce := "valid-nonce-12345"
	timestamp := time.Now().Unix()

	err := validator.ValidateNonce(validNonce, timestamp)
	if err != nil {
		t.Errorf("Expected valid nonce to pass: %v", err)
	}

	err = validator.ValidateNonce(validNonce, timestamp)
	if err == nil {
		t.Error("Expected replay detection")
	}

	shortNonce := "short"
	err = validator.ValidateNonce(shortNonce, timestamp)
	if err == nil {
		t.Error("Expected short nonce to fail")
	}

	invalidCharsNonce := "invalid@nonce#"
	err = validator.ValidateNonce(invalidCharsNonce, timestamp)
	if err == nil {
		t.Error("Expected invalid characters to fail")
	}

	oldTimestamp := time.Now().Add(-20 * time.Minute).Unix()
	oldNonce := "old-nonce"
	err = validator.ValidateNonce(oldNonce, oldTimestamp)
	if err == nil {
		t.Error("Expected old nonce to fail")
	}
}

func TestSlidingWindowCounter(t *testing.T) {
	counter := &slidingWindowCounter{
		requests: make([]time.Time, 0),
		window:   1 * time.Second,
	}

	for i := 0; i < 10; i++ {
		counter.AddRequest()
	}

	if counter.Count() != 10 {
		t.Errorf("Expected count 10, got %d", counter.Count())
	}

	time.Sleep(1100 * time.Millisecond)

	if counter.Count() > 0 {
		t.Errorf("Expected count 0 after window expiry, got %d", counter.Count())
	}
}

func TestBuildEnhancedStringToSign(t *testing.T) {
	stringToSign := buildEnhancedStringToSign(
		"POST",
		"/api/test",
		"key=value&other=123",
		1234567890,
		"test-nonce",
		"body-hash",
		"additional",
	)

	if stringToSign == "" {
		t.Error("Expected non-empty string to sign")
	}

	expected := "POST\n/api/test\nkey=value&other=123\n1234567890\ntest-nonce\nbody-hash\nadditional"
	if stringToSign != expected {
		t.Errorf("String to sign mismatch:\nExpected: %s\nGot: %s", expected, stringToSign)
	}
}

func TestComputeEnhancedHMAC(t *testing.T) {
	key := "test-key"
	data := "test-data"

	sha256Result := computeEnhancedHMAC(key, data, false)
	sha512Result := computeEnhancedHMAC(key, data, true)

	if sha256Result == "" || sha512Result == "" {
		t.Error("Expected non-empty HMAC results")
	}

	if sha256Result == sha512Result {
		t.Error("Expected different HMAC results for SHA256 vs SHA512")
	}

	if len(sha256Result) != 64 {
		t.Errorf("Expected SHA256 length 64, got %d", len(sha256Result))
	}

	if len(sha512Result) != 128 {
		t.Errorf("Expected SHA512 length 128, got %d", len(sha512Result))
	}
}

func TestHashNonce(t *testing.T) {
	nonce := "test-nonce-123"

	hash1 := hashNonce(nonce)
	hash2 := hashNonce(nonce)

	if hash1 != hash2 {
		t.Error("Expected same nonce to produce same hash")
	}

	differentNonce := "different-nonce"
	hash3 := hashNonce(differentNonce)

	if hash1 == hash3 {
		t.Error("Expected different nonces to produce different hashes")
	}
}

func bytesFromString(s string) *bytes.Reader {
	return bytes.NewReader([]byte(s))
}

func TestCheckReplayFunction(t *testing.T) {
	nonce := fmt.Sprintf("replay-test-%d", time.Now().UnixNano())

	firstCheck := CheckReplay(nonce)
	if firstCheck {
		t.Error("First check should return false")
	}

	secondCheck := CheckReplay(nonce)
	if !secondCheck {
		t.Error("Second check should return true (replay detected)")
	}
}

func TestSignatureExcludePaths(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(EnhancedSignatureVerification(EnhancedSignatureConfig{
		SecretKey:        "test-secret",
		RequireTimestamp: true,
		RequireNonce:     true,
		ExcludePaths:     []string{"/health", "/metrics"},
	}))

	router.GET("/health", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	router.GET("/api/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req1 := httptest.NewRequest("GET", "/health", nil)
	w1 := httptest.NewRecorder()
	router.ServeHTTP(w1, req1)

	if w1.Code != http.StatusOK {
		t.Errorf("Health endpoint should be excluded, got %d", w1.Code)
	}

	req2 := httptest.NewRequest("GET", "/api/test", nil)
	w2 := httptest.NewRecorder()
	router.ServeHTTP(w2, req2)

	if w2.Code != http.StatusUnauthorized {
		t.Errorf("API endpoint should require signature, got %d", w2.Code)
	}
}

func TestEd25519KeyGeneration(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Errorf("Key generation failed: %v", err)
	}

	if len(privateKey) == 0 || len(publicKey) == 0 {
		t.Error("Expected non-empty keys")
	}
}

func TestEd25519SignAndVerify(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Errorf("Key generation failed: %v", err)
	}

	message := []byte("test message to sign")

	signature, err := SignEd25519(message, privateKey)
	if err != nil {
		t.Errorf("Signing failed: %v", err)
	}

	if len(signature) != 64 {
		t.Errorf("Expected signature length 64, got %d", len(signature))
	}

	valid, err := VerifyEd25519(message, signature, publicKey)
	if err != nil {
		t.Errorf("Verification failed: %v", err)
	}

	if !valid {
		t.Error("Expected valid signature")
	}

	invalidMessage := []byte("different message")
	valid, _ = VerifyEd25519(invalidMessage, signature, publicKey)
	if valid {
		t.Error("Expected invalid signature for different message")
	}
}

func TestEd25519StringSignAndVerify(t *testing.T) {
	_, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Errorf("Key generation failed: %v", err)
	}

	privateKey := make([]byte, 64)
	for i := range privateKey {
		privateKey[i] = byte(i)
	}

	message := "test message"

	signature, err := SignEd25519String(message, privateKey)
	if err != nil {
		t.Errorf("String signing failed: %v", err)
	}

	if signature == "" {
		t.Error("Expected non-empty signature")
	}

	decodedSig, err := hex.DecodeString(signature)
	if err != nil {
		t.Errorf("Signature hex decode failed: %v", err)
	}

	valid, err := VerifyEd25519([]byte(message), decodedSig, publicKey)
	if err != nil {
		t.Errorf("String verification failed: %v", err)
	}

	if !valid {
		t.Error("Expected valid string signature")
	}
}
