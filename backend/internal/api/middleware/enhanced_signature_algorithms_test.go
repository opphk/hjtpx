package middleware

import (
	"bytes"
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"strconv"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
)

func TestSignatureAlgorithmConstants(t *testing.T) {
	algorithms := []SignatureAlgorithm{
		AlgorithmHMACSHA256,
		AlgorithmHMACSHA512,
		AlgorithmBlake2b256,
		AlgorithmBlake2b512,
	}

	expectedValues := []string{
		"HMAC-SHA256",
		"HMAC-SHA512",
		"BLAKE2B-256",
		"BLAKE2B-512",
	}

	for i, algo := range algorithms {
		if string(algo) != expectedValues[i] {
			t.Errorf("Algorithm %d expected %s, got %s", i, expectedValues[i], algo)
		}

		if !algo.IsValid() {
			t.Errorf("Algorithm %s should be valid", algo)
		}
	}

	invalidAlgo := SignatureAlgorithm("INVALID")
	if invalidAlgo.IsValid() {
		t.Error("Invalid algorithm should not be valid")
	}
}

func TestSignatureAlgorithmOutputLength(t *testing.T) {
	tests := []struct {
		algorithm SignatureAlgorithm
		expected  int
	}{
		{AlgorithmHMACSHA256, 32},
		{AlgorithmBlake2b256, 32},
		{AlgorithmHMACSHA512, 64},
		{AlgorithmBlake2b512, 64},
	}

	for _, test := range tests {
		length := test.algorithm.OutputLength()
		if length != test.expected {
			t.Errorf("Algorithm %s expected output length %d, got %d", test.algorithm, test.expected, length)
		}
	}
}

func TestKeyRotationManagerCreation(t *testing.T) {
	initialKey := []byte("test-initial-key-1234567890123456")
	rotationPeriod := 1 * time.Hour
	maxHistory := 5

	mgr := NewKeyRotationManager(initialKey, rotationPeriod, maxHistory)

	if mgr == nil {
		t.Fatal("KeyRotationManager should not be nil")
	}

	if !bytes.Equal(mgr.GetCurrentKey(), initialKey) {
		t.Error("Current key should match initial key")
	}

	if mgr.GetKeyVersion() != 1 {
		t.Errorf("Initial key version should be 1, got %d", mgr.GetKeyVersion())
	}

	history := mgr.GetKeyHistory()
	if len(history) != 1 {
		t.Errorf("Initial history should have 1 entry, got %d", len(history))
	}
}

func TestKeyRotationManagerRotate(t *testing.T) {
	initialKey := []byte("test-initial-key-1234567890123456")
	rotationPeriod := 1 * time.Hour
	maxHistory := 5

	mgr := NewKeyRotationManager(initialKey, rotationPeriod, maxHistory)

	err := mgr.RotateKey()
	if err != nil {
		t.Fatalf("Key rotation failed: %v", err)
	}

	if mgr.GetKeyVersion() != 2 {
		t.Errorf("Key version should be 2 after rotation, got %d", mgr.GetKeyVersion())
	}

	if bytes.Equal(mgr.GetCurrentKey(), initialKey) {
		t.Error("Current key should be different after rotation")
	}

	oldKey, found := mgr.GetHistoricalKey(1)
	if !found {
		t.Error("Should be able to retrieve old key")
	}
	if !bytes.Equal(oldKey, initialKey) {
		t.Error("Retrieved old key should match initial key")
	}

	history := mgr.GetKeyHistory()
	if len(history) != 2 {
		t.Errorf("History should have 2 entries after rotation, got %d", len(history))
	}
}

func TestKeyRotationManagerValidateKey(t *testing.T) {
	initialKey := []byte("test-initial-key-1234567890123456")
	rotationPeriod := 1 * time.Hour
	maxHistory := 5

	mgr := NewKeyRotationManager(initialKey, rotationPeriod, maxHistory)

	if !mgr.ValidateKey(initialKey) {
		t.Error("Initial key should be valid")
	}

	if mgr.ValidateKey([]byte("wrong-key")) {
		t.Error("Wrong key should not be valid")
	}

	err := mgr.RotateKey()
	if err != nil {
		t.Fatalf("Key rotation failed: %v", err)
	}

	if !mgr.ValidateKey(initialKey) {
		t.Error("Old key should still be valid after rotation")
	}

	newKey := mgr.GetCurrentKey()
	if !mgr.ValidateKey(newKey) {
		t.Error("New key should be valid")
	}
}

func TestComputeSignatureWithAlgorithm(t *testing.T) {
	key := "test-key"
	data := "test-data"

	signatures := make(map[SignatureAlgorithm]string)

	for _, algo := range []SignatureAlgorithm{
		AlgorithmHMACSHA256,
		AlgorithmHMACSHA512,
		AlgorithmBlake2b256,
		AlgorithmBlake2b512,
	} {
		sig := computeSignatureWithAlgorithm(key, data, algo)
		if sig == "" {
			t.Errorf("Signature for %s should not be empty", algo)
		}
		signatures[algo] = sig
	}

	if signatures[AlgorithmHMACSHA256] == signatures[AlgorithmHMACSHA512] {
		t.Error("SHA256 and SHA512 signatures should be different")
	}

	if signatures[AlgorithmBlake2b256] == signatures[AlgorithmBlake2b512] {
		t.Error("BLAKE2B-256 and BLAKE2B-512 signatures should be different")
	}

	for algo1, sig1 := range signatures {
		for algo2, sig2 := range signatures {
			if algo1 != algo2 && sig1 == sig2 {
				t.Errorf("Different algorithms %s and %s produced same signature", algo1, algo2)
			}
		}
	}
}

func TestGenerateSignatureWithAlgorithm(t *testing.T) {
	secretKey := "test-secret"
	method := "POST"
	path := "/api/test"
	query := "foo=bar"
	timestamp := time.Now().Unix()
	nonce := "test-nonce-12345678"
	body := []byte(`{"test": "data"}`)

	for _, algo := range []SignatureAlgorithm{
		AlgorithmHMACSHA256,
		AlgorithmHMACSHA512,
		AlgorithmBlake2b256,
		AlgorithmBlake2b512,
	} {
		sig := GenerateSignatureWithAlgorithm(secretKey, method, path, query, timestamp, nonce, body, algo)

		if sig == "" {
			t.Errorf("Signature for %s should not be empty", algo)
		}

		sig2 := GenerateSignatureWithAlgorithm(secretKey, method, path, query, timestamp, nonce, body, algo)
		if sig != sig2 {
			t.Errorf("Same inputs should produce same signature for %s", algo)
		}

		if !VerifySignatureWithAlgorithm(secretKey, method, path, query, timestamp, nonce, body, sig, algo) {
			t.Errorf("Signature verification should pass for %s", algo)
		}

		wrongSig := sig[:len(sig)-1] + "x"
		if VerifySignatureWithAlgorithm(secretKey, method, path, query, timestamp, nonce, body, wrongSig, algo) {
			t.Errorf("Modified signature should fail verification for %s", algo)
		}
	}
}

func TestGenerateSignatureWithKeyManager(t *testing.T) {
	initialKey := []byte("test-initial-key-1234567890123456")
	rotationPeriod := 1 * time.Hour
	maxHistory := 5

	keyManager := NewKeyRotationManager(initialKey, rotationPeriod, maxHistory)

	method := "POST"
	path := "/api/test"
	query := "foo=bar"
	timestamp := time.Now().Unix()
	nonce := "test-nonce-12345678"
	body := []byte(`{"test": "data"}`)

	sig, version := GenerateSignatureWithKeyManager(keyManager, method, path, query, timestamp, nonce, body, AlgorithmHMACSHA256)

	if sig == "" {
		t.Error("Signature should not be empty")
	}

	if version != 1 {
		t.Errorf("Initial version should be 1, got %d", version)
	}

	if !VerifySignatureWithKeyManager(keyManager, method, path, query, timestamp, nonce, body, sig, AlgorithmHMACSHA256, version) {
		t.Error("Signature verification should pass with key manager")
	}

	err := keyManager.RotateKey()
	if err != nil {
		t.Fatalf("Key rotation failed: %v", err)
	}

	if !VerifySignatureWithKeyManager(keyManager, method, path, query, timestamp, nonce, body, sig, AlgorithmHMACSHA256, 1) {
		t.Error("Old signature should still verify with historical key")
	}

	newSig, newVersion := GenerateSignatureWithKeyManager(keyManager, method, path, query, timestamp, nonce, body, AlgorithmHMACSHA256)

	if newSig == sig {
		t.Error("New signature should be different after key rotation")
	}

	if newVersion != 2 {
		t.Errorf("New version should be 2, got %d", newVersion)
	}
}

func TestSignatureAuditLogger(t *testing.T) {
	logger, err := NewSignatureAuditLogger(100, "")
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	entry := SignatureAuditEntry{
		RequestPath: "/api/test",
		ClientIP:    "127.0.0.1",
		Algorithm:   "HMAC-SHA256",
		Signature:   "test-signature",
		Valid:       true,
		Reason:      "success",
		Duration:    100 * time.Millisecond,
	}

	logger.Log(entry)

	logs := logger.GetLogs(10)
	if len(logs) != 1 {
		t.Errorf("Should have 1 log entry, got %d", len(logs))
	}

	if !logs[0].Valid {
		t.Error("Log entry should be valid")
	}
}

func TestSignatureAuditLoggerFailedLogs(t *testing.T) {
	logger, err := NewSignatureAuditLogger(100, "")
	if err != nil {
		t.Fatalf("Failed to create audit logger: %v", err)
	}
	defer logger.Close()

	for i := 0; i < 5; i++ {
		logger.Log(SignatureAuditEntry{
			RequestPath: "/api/test",
			ClientIP:    "127.0.0.1",
			Algorithm:   "HMAC-SHA256",
			Valid:       true,
			Duration:    100 * time.Millisecond,
		})
	}

	for i := 0; i < 3; i++ {
		logger.Log(SignatureAuditEntry{
			RequestPath: "/api/test",
			ClientIP:    "127.0.0.1",
			Algorithm:   "HMAC-SHA256",
			Valid:       false,
			Reason:      "invalid signature",
			Duration:    100 * time.Millisecond,
		})
	}

	failedLogs := logger.GetFailedLogs(10)
	if len(failedLogs) != 3 {
		t.Errorf("Should have 3 failed logs, got %d", len(failedLogs))
	}
}

func TestSignatureCache(t *testing.T) {
	cache := newSignatureCache(100)

	key := "test-signature-key"
	signature := []byte("test-signature")
	ttl := 1 * time.Minute

	if _, exists := cache.get(key); exists {
		t.Error("Cache should not contain key before set")
	}

	cache.set(key, signature, ttl)

	if sig, exists := cache.get(key); !exists {
		t.Error("Cache should contain key after set")
	} else if !bytes.Equal(sig, signature) {
		t.Error("Cached signature should match original")
	}

	hits, misses, size := cache.stats()
	if hits != 1 {
		t.Errorf("Should have 1 hit, got %d", hits)
	}
	if misses != 0 {
		t.Errorf("Should have 0 misses, got %d", misses)
	}
	if size != 1 {
		t.Errorf("Cache size should be 1, got %d", size)
	}
}

func TestSignatureCacheEviction(t *testing.T) {
	cache := newSignatureCache(5)

	ttl := 1 * time.Hour

	for i := 0; i < 10; i++ {
		key := fmt.Sprintf("key-%d", i)
		sig := []byte(fmt.Sprintf("signature-%d", i))
		cache.set(key, sig, ttl)
	}

	hits, misses, size := cache.stats()
	if size != 5 {
		t.Errorf("Cache size should be 5 after eviction, got %d", size)
	}
}

func TestSignatureCacheExpiration(t *testing.T) {
	cache := newSignatureCache(100)

	key := "test-key"
	signature := []byte("test-signature")

	cache.set(key, signature, 1*time.Millisecond)

	time.Sleep(10 * time.Millisecond)

	if _, exists := cache.get(key); exists {
		t.Error("Expired entry should not be in cache")
	}
}

func TestPerformanceStats(t *testing.T) {
	stats := NewPerformanceStats()

	stats.RecordValidation("HMAC-SHA256", true, 100*time.Millisecond)
	stats.RecordValidation("HMAC-SHA256", true, 200*time.Millisecond)
	stats.RecordValidation("HMAC-SHA512", false, 50*time.Millisecond)

	result := stats.GetStats()

	if result["total_validations"].(int64) != 3 {
		t.Errorf("Should have 3 total validations, got %v", result["total_validations"])
	}

	if result["total_valid"].(int64) != 2 {
		t.Errorf("Should have 2 valid, got %v", result["total_valid"])
	}

	if result["total_invalid"].(int64) != 1 {
		t.Errorf("Should have 1 invalid, got %v", result["total_invalid"])
	}
}

func TestEnhancedSignatureWithBlake2b(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-1234567890123456")
	config.EnableBlake2b = true

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.POST("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "blake2b-test-nonce-123456"

	signature := GenerateSignatureWithAlgorithm(
		config.SecretKey,
		"POST",
		"/api/test",
		"",
		timestamp,
		nonce,
		body,
		AlgorithmBlake2b256,
	)

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature-Algorithm", string(AlgorithmBlake2b256))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid BLAKE2B-256 signature should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureWithBlake2b512(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-1234567890123456")
	config.EnableBlake2b = true

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.POST("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "blake2b512-test-nonce-123456"

	signature := GenerateSignatureWithAlgorithm(
		config.SecretKey,
		"POST",
		"/api/test",
		"",
		timestamp,
		nonce,
		body,
		AlgorithmBlake2b512,
	)

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature-Algorithm", string(AlgorithmBlake2b512))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid BLAKE2B-512 signature should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureWithSHA512(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewEnhancedSignatureConfig("test-secret-key-1234567890123456")
	config.EnableHMAC_SHA512 = true

	router := gin.New()
	router.Use(EnhancedSignatureVerification(config))
	router.POST("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "sha512-test-nonce-12345678"

	signature := GenerateSignatureWithAlgorithm(
		config.SecretKey,
		"POST",
		"/api/test",
		"",
		timestamp,
		nonce,
		body,
		AlgorithmHMACSHA512,
	)

	req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Signature", signature)
	req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
	req.Header.Set("X-Nonce", nonce)
	req.Header.Set("X-Signature-Algorithm", string(AlgorithmHMACSHA512))

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusOK {
		t.Errorf("Valid HMAC-SHA512 signature should return 200, got %d", w.Code)
	}
}

func TestEnhancedSignatureAlgorithmSelection(t *testing.T) {
	algorithms := []SignatureAlgorithm{
		AlgorithmHMACSHA256,
		AlgorithmHMACSHA512,
		AlgorithmBlake2b256,
		AlgorithmBlake2b512,
	}

	for _, algo := range algorithms {
		algo := algo
		t.Run(string(algo), func(t *testing.T) {
			t.Parallel()

			gin.SetMode(gin.TestMode)

			config := NewEnhancedSignatureConfig("test-secret-key-1234567890123456")

			router := gin.New()
			router.Use(EnhancedSignatureVerification(config))
			router.POST("/api/test", func(c *gin.Context) {
				c.String(200, "OK")
			})

			body := []byte(`{"test": "data"}`)
			timestamp := time.Now().Unix()
			nonce := "test-nonce-12345678"

			signature := GenerateSignatureWithAlgorithm(
				config.SecretKey,
				"POST",
				"/api/test",
				"",
				timestamp,
				nonce,
				body,
				algo,
			)

			req := httptest.NewRequest("POST", "/api/test", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			req.Header.Set("X-Signature", signature)
			req.Header.Set("X-Timestamp", strconv.FormatInt(timestamp, 10))
			req.Header.Set("X-Nonce", nonce)
			req.Header.Set("X-Signature-Algorithm", string(algo))

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if w.Code != http.StatusOK {
				t.Errorf("Valid signature for %s should return 200, got %d", algo, w.Code)
			}
		})
	}
}

func TestGetEnhancedSignatureInfo(t *testing.T) {
	info := GetEnhancedSignatureInfo()

	if info.Version != "3.0" {
		t.Errorf("Version should be 3.0, got %s", info.Version)
	}

	if info.Algorithm != "HMAC-SHA256" {
		t.Errorf("Default algorithm should be HMAC-SHA256, got %s", info.Algorithm)
	}
}

func TestComputeBlake2bFunctions(t *testing.T) {
	key := []byte("test-key")
	data := []byte("test-data")

	hash256, err := computeBlake2b256(key, data)
	if err != nil {
		t.Fatalf("computeBlake2b256 failed: %v", err)
	}
	if len(hash256) != 32 {
		t.Errorf("Blake2b-256 should produce 32 bytes, got %d", len(hash256))
	}

	hash512, err := computeBlake2b512(key, data)
	if err != nil {
		t.Fatalf("computeBlake2b512 failed: %v", err)
	}
	if len(hash512) != 64 {
		t.Errorf("Blake2b-512 should produce 64 bytes, got %d", len(hash512))
	}

	if bytes.Equal(hash256, hash512) {
		t.Error("Blake2b-256 and Blake2b-512 should produce different hashes")
	}
}

func TestKeyRotationWithMultipleRotations(t *testing.T) {
	initialKey := []byte("test-initial-key-1234567890123456")
	rotationPeriod := 1 * time.Hour
	maxHistory := 3

	mgr := NewKeyRotationManager(initialKey, rotationPeriod, maxHistory)

	for i := 2; i <= 5; i++ {
		err := mgr.RotateKey()
		if err != nil {
			t.Fatalf("Rotation %d failed: %v", i, err)
		}

		if mgr.GetKeyVersion() != i {
			t.Errorf("Version should be %d after rotation %d, got %d", i, i, mgr.GetKeyVersion())
		}
	}

	history := mgr.GetKeyHistory()
	if len(history) > maxHistory {
		t.Errorf("History should not exceed maxHistory %d, got %d", maxHistory, len(history))
	}

	for i := 1; i <= 5; i++ {
		_, found := mgr.GetHistoricalKey(i)
		if i > 5-maxHistory && !found {
			t.Errorf("Recent key version %d should still be accessible", i)
		}
	}
}

func TestSignatureAlgorithmJSON(t *testing.T) {
	algo := AlgorithmHMACSHA256

	data, err := json.Marshal(algo)
	if err != nil {
		t.Fatalf("Failed to marshal algorithm: %v", err)
	}

	var decoded SignatureAlgorithm
	err = json.Unmarshal(data, &decoded)
	if err != nil {
		t.Fatalf("Failed to unmarshal algorithm: %v", err)
	}

	if decoded != algo {
		t.Errorf("Decoded algorithm %s should match original %s", decoded, algo)
	}
}

func TestNewEnhancedSignatureConfigHasNewFields(t *testing.T) {
	secretKey := "test-secret"
	config := NewEnhancedSignatureConfig(secretKey)

	if config.EnableBlake2b != false {
		t.Error("EnableBlake2b should default to false")
	}

	if config.EnableKeyRotation != false {
		t.Error("EnableKeyRotation should default to false")
	}

	if config.EnableAuditLog != true {
		t.Error("EnableAuditLog should default to true")
	}

	if config.EnablePerformanceLog != true {
		t.Error("EnablePerformanceLog should default to true")
	}

	if config.CacheSignatures != true {
		t.Error("CacheSignatures should default to true")
	}

	if config.SignatureCacheSize != 10000 {
		t.Errorf("SignatureCacheSize should default to 10000, got %d", config.SignatureCacheSize)
	}

	if config.SignatureVersion != "3.0" {
		t.Errorf("SignatureVersion should be 3.0, got %s", config.SignatureVersion)
	}
}

func TestGlobalComponentsInitialization(t *testing.T) {
	if globalKeyRotationManager == nil {
		t.Error("Global key rotation manager should be initialized")
	}

	if globalSignatureCache == nil {
		t.Error("Global signature cache should be initialized")
	}

	if globalPerformanceStats == nil {
		t.Error("Global performance stats should be initialized")
	}
}

func TestGetGlobalComponents(t *testing.T) {
	mgr := GetKeyRotationManager()
	if mgr == nil {
		t.Error("GetKeyRotationManager should return non-nil")
	}

	cache := GetSignatureCache()
	if cache == nil {
		t.Error("GetSignatureCache should return non-nil")
	}

	stats := GetPerformanceStats()
	if stats == nil {
		t.Error("GetPerformanceStats should return non-nil")
	}

	logger := GetAuditLogger()
	if logger == nil {
		t.Error("GetAuditLogger should return non-nil")
	}
}

func TestTriggerKeyRotation(t *testing.T) {
	err := TriggerKeyRotation()
	if err != nil {
		t.Errorf("TriggerKeyRotation should not error: %v", err)
	}

	initialVersion := globalKeyRotationManager.GetKeyVersion()

	err = TriggerKeyRotation()
	if err != nil {
		t.Errorf("Second TriggerKeyRotation should not error: %v", err)
	}

	if globalKeyRotationManager.GetKeyVersion() != initialVersion+1 {
		t.Errorf("Key version should increase after rotation")
	}
}
