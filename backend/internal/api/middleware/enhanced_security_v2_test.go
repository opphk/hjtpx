package middleware

import (
	"bytes"
	"context"
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

func TestSecurityV2ConfigDefaults(t *testing.T) {
	config := defaultSecurityV2Config

	if config.SecretKey == "" {
		t.Error("SecretKey should not be empty")
	}
	if config.Algorithm != "HMAC-SHA3-256" {
		t.Errorf("Default algorithm should be HMAC-SHA3-256, got %s", config.Algorithm)
	}
	if config.TimestampTolerance != 5*time.Minute {
		t.Errorf("Default timestamp tolerance should be 5 minutes, got %v", config.TimestampTolerance)
	}
	if config.ReplayWindowSecs != 300 {
		t.Errorf("Default replay window should be 300, got %d", config.ReplayWindowSecs)
	}
	if config.TokenBucketRate != 100.0 {
		t.Errorf("Default token bucket rate should be 100.0, got %f", config.TokenBucketRate)
	}
	if config.TokenBucketBurst != 200 {
		t.Errorf("Default token bucket burst should be 200, got %d", config.TokenBucketBurst)
	}
}

func TestNewSecurityV2Config(t *testing.T) {
	secretKey := "test-secret-key-v2"
	config := NewSecurityV2Config(secretKey)

	if config.SecretKey != secretKey {
		t.Error("SecretKey should match")
	}
	if config.Algorithm != "HMAC-SHA3-256" {
		t.Error("Algorithm should be HMAC-SHA3-256")
	}
	if config.EnableHMACSHA3 != true {
		t.Error("HMAC-SHA3 should be enabled")
	}
	if len(config.SupportedVersions) != 3 {
		t.Errorf("Should support 3 versions, got %d", len(config.SupportedVersions))
	}
}

func TestNewEnhancedSecurityV2(t *testing.T) {
	security := NewEnhancedSecurityV2()

	if security == nil {
		t.Fatal("EnhancedSecurityV2 should not be nil")
	}

	if security.config.Algorithm != "HMAC-SHA3-256" {
		t.Error("Algorithm should be HMAC-SHA3-256")
	}
}

func TestGetEnhancedSecurityV2(t *testing.T) {
	security1 := GetEnhancedSecurityV2()
	security2 := GetEnhancedSecurityV2()

	if security1 == nil || security2 == nil {
		t.Fatal("EnhancedSecurityV2 should not be nil")
	}

	if security1 != security2 {
		t.Error("GetEnhancedSecurityV2 should return the same instance")
	}
}

func TestEnhancedSignatureHMACSHA3(t *testing.T) {
	security := NewEnhancedSecurityV2()

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Query:     "foo=bar",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-12345678",
		Version:   "v2",
	}

	signature, err := security.EnhancedSignature(context.Background(), req)
	if err != nil {
		t.Fatalf("EnhancedSignature failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	signature2, err := security.EnhancedSignature(context.Background(), req)
	if err != nil {
		t.Fatalf("EnhancedSignature second call failed: %v", err)
	}

	if signature != signature2 {
		t.Error("Same inputs should produce same signature")
	}
}

func TestEnhancedSignatureDifferentInputs(t *testing.T) {
	security := NewEnhancedSecurityV2()

	timestamp := time.Now().Unix()
	nonce := "test-nonce-12345678"

	req1 := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test1"}`),
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   "v2",
	}

	req2 := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test2"}`),
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   "v2",
	}

	sig1, _ := security.EnhancedSignature(context.Background(), req1)
	sig2, _ := security.EnhancedSignature(context.Background(), req2)

	if sig1 == sig2 {
		t.Error("Different inputs should produce different signatures")
	}
}

func TestEnhancedReplayProtectionValid(t *testing.T) {
	security := NewEnhancedSecurityV2()

	nonce := "valid-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	timestamp := time.Now().Unix()

	result := security.EnhancedReplayProtection(context.Background(), nonce, timestamp)

	if !result {
		t.Error("Valid request should pass replay protection")
	}
}

func TestEnhancedReplayProtectionExpiredTimestamp(t *testing.T) {
	security := NewEnhancedSecurityV2()

	nonce := "expired-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	timestamp := time.Now().Add(-10 * time.Minute).Unix()

	result := security.EnhancedReplayProtection(context.Background(), nonce, timestamp)

	if result {
		t.Error("Expired timestamp should fail replay protection")
	}
}

func TestEnhancedReplayProtectionNonceReuse(t *testing.T) {
	security := NewEnhancedSecurityV2()

	nonce := "reuse-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	timestamp := time.Now().Unix()

	firstResult := security.EnhancedReplayProtection(context.Background(), nonce, timestamp)
	if !firstResult {
		t.Error("First use of nonce should succeed")
	}

	secondResult := security.EnhancedReplayProtection(context.Background(), nonce, timestamp)
	if secondResult {
		t.Error("Reuse of nonce should fail replay protection")
	}
}

func TestTokenBucketRateLimitAllowed(t *testing.T) {
	security := NewEnhancedSecurityV2()

	key := "test-rate-limit-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	allowed, err := security.TokenBucketRateLimit(context.Background(), key, 100.0, 10)
	if err != nil {
		t.Fatalf("TokenBucketRateLimit failed: %v", err)
	}

	if !allowed {
		t.Error("First request should be allowed")
	}
}

func TestTokenBucketRateLimitExceeded(t *testing.T) {
	security := NewEnhancedSecurityV2()

	key := "test-rate-limit-exceed-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	for i := 0; i < 10; i++ {
		allowed, err := security.TokenBucketRateLimit(context.Background(), key, 100.0, 10)
		if err != nil {
			t.Fatalf("TokenBucketRateLimit failed: %v", err)
		}
		if !allowed {
			t.Errorf("Request %d should be allowed", i+1)
		}
	}

	allowed, err := security.TokenBucketRateLimit(context.Background(), key, 100.0, 10)
	if err != nil {
		t.Fatalf("TokenBucketRateLimit failed: %v", err)
	}

	if allowed {
		t.Error("Request exceeding burst should be rejected")
	}
}

func TestTokenBucketRefillOverTime(t *testing.T) {
	security := NewEnhancedSecurityV2()

	key := "test-refill-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	for i := 0; i < 5; i++ {
		security.TokenBucketRateLimit(context.Background(), key, 1.0, 5)
	}

	allowed, _ := security.TokenBucketRateLimit(context.Background(), key, 1.0, 5)
	if allowed {
		t.Error("Bucket should be exhausted")
	}

	time.Sleep(200 * time.Millisecond)

	allowed, _ = security.TokenBucketRateLimit(context.Background(), key, 1.0, 5)
	if !allowed {
		t.Error("Bucket should have refilled after time")
	}
}

func TestVersionRoutingBasic(t *testing.T) {
	security := NewEnhancedSecurityV2()

	version, err := security.VersionRouting(context.Background(), "/api/v2/test")
	if err != nil {
		t.Fatalf("VersionRouting failed: %v", err)
	}

	if version != "v2" {
		t.Errorf("Expected version v2, got %s", version)
	}
}

func TestVersionRoutingDefault(t *testing.T) {
	security := NewEnhancedSecurityV2()

	version, err := security.VersionRouting(context.Background(), "/api/test")
	if err != nil {
		t.Fatalf("VersionRouting failed: %v", err)
	}

	if version != "v1" {
		t.Errorf("Expected default version v1, got %s", version)
	}
}

func TestVersionRoutingPathExtraction(t *testing.T) {
	security := NewEnhancedSecurityV2()

	testCases := []struct {
		path     string
		expected string
	}{
		{"/v1/resource", "v1"},
		{"/v2/resource", "v2"},
		{"/v3/resource", "v3"},
		{"/api/test", "v1"},
		{"/resource", "v1"},
	}

	for _, tc := range testCases {
		version, _ := security.VersionRouting(context.Background(), tc.path)
		if version != tc.expected {
			t.Errorf("Path %s: expected %s, got %s", tc.path, tc.expected, version)
		}
	}
}

func TestVersionRouterIsSupported(t *testing.T) {
	vr := newVersionRouter([]string{"v1", "v2", "v3"}, "v1")

	if !vr.isVersionSupported("v1") {
		t.Error("v1 should be supported")
	}
	if !vr.isVersionSupported("v2") {
		t.Error("v2 should be supported")
	}
	if vr.isVersionSupported("v4") {
		t.Error("v4 should not be supported")
	}
}

func TestCheckReplay(t *testing.T) {
	security := NewEnhancedSecurityV2()

	nonce := "check-replay-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	timestamp := time.Now().Unix()

	result := security.CheckReplay(context.Background(), nonce, timestamp)

	if !result.Allowed {
		t.Error("CheckReplay should allow valid request")
	}
	if !result.TimestampValid {
		t.Error("Timestamp should be valid")
	}
	if result.IsReplay {
		t.Error("Should not be detected as replay")
	}
}

func TestCheckReplayExpired(t *testing.T) {
	security := NewEnhancedSecurityV2()

	nonce := "expired-replay-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	timestamp := time.Now().Add(-10 * time.Minute).Unix()

	result := security.CheckReplay(context.Background(), nonce, timestamp)

	if result.Allowed {
		t.Error("CheckReplay should not allow expired timestamp")
	}
	if result.TimestampValid {
		t.Error("Timestamp should be invalid")
	}
}

func TestCheckReplayNonceReuse(t *testing.T) {
	security := NewEnhancedSecurityV2()

	nonce := "reuse-check-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	timestamp := time.Now().Unix()

	security.CheckReplay(context.Background(), nonce, timestamp)

	result := security.CheckReplay(context.Background(), nonce, timestamp)

	if result.Allowed {
		t.Error("CheckReplay should not allow nonce reuse")
	}
	if !result.IsReplay {
		t.Error("Should be detected as replay")
	}
}

func TestVerifySignature(t *testing.T) {
	security := NewEnhancedSecurityV2()

	timestamp := time.Now().Unix()
	nonce := "verify-sig-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   "v2",
	}

	signature, _ := security.EnhancedSignature(context.Background(), req)

	result := security.VerifySignature(req, signature)

	if !result.Valid {
		t.Errorf("VerifySignature failed: %s - %s", result.ErrorCode, result.ErrorReason)
	}
	if result.Algorithm != "HMAC-SHA3-256" {
		t.Errorf("Algorithm should be HMAC-SHA3-256, got %s", result.Algorithm)
	}
}

func TestVerifySignatureInvalid(t *testing.T) {
	security := NewEnhancedSecurityV2()

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: time.Now().Unix(),
		Nonce:     "invalid-sig-nonce",
		Version:   "v2",
	}

	result := security.VerifySignature(req, "invalid-signature")

	if result.Valid {
		t.Error("Invalid signature should not be valid")
	}
	if result.ErrorCode != "SIGNATURE_MISMATCH" {
		t.Errorf("Error code should be SIGNATURE_MISMATCH, got %s", result.ErrorCode)
	}
}

func TestCheckRateLimit(t *testing.T) {
	security := NewEnhancedSecurityV2()

	key := "check-rate-limit-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	result := security.CheckRateLimit(context.Background(), key, 100.0, 10)

	if !result.Allowed {
		t.Error("CheckRateLimit should allow request")
	}
	if result.Remaining != 9 {
		t.Errorf("Remaining should be 9, got %d", result.Remaining)
	}
	if result.Limit != 10 {
		t.Errorf("Limit should be 10, got %d", result.Limit)
	}
}

func TestRouteVersion(t *testing.T) {
	security := NewEnhancedSecurityV2()

	result := security.RouteVersion(context.Background(), "v2")

	if result.Version != "v2" {
		t.Errorf("Version should be v2, got %s", result.Version)
	}
	if !result.Compatible {
		t.Error("v2 should be compatible")
	}
}

func TestRouteVersionDeprecated(t *testing.T) {
	security := NewEnhancedSecurityV2()

	result := security.RouteVersion(context.Background(), "v1")

	if result.Version != "v1" {
		t.Errorf("Version should be v1, got %s", result.Version)
	}
	if result.DeprecationNotice == "" {
		t.Error("v1 should have deprecation notice")
	}
}

func TestRouteVersionUnsupported(t *testing.T) {
	security := NewEnhancedSecurityV2()

	result := security.RouteVersion(context.Background(), "v99")

	if result.Compatible {
		t.Error("v99 should not be compatible")
	}
	if result.DeprecationNotice == "" {
		t.Error("Unsupported version should have deprecation notice")
	}
}

func TestEnhancedSecurityV2MiddlewareExcludedPath(t *testing.T) {
	security := NewEnhancedSecurityV2()

	router := gin.New()
	router.Use(EnhancedSecurityV2Middleware())
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

func TestEnhancedSecurityV2MiddlewareMissingSignature(t *testing.T) {
	security := NewEnhancedSecurityV2()

	router := gin.New()
	router.Use(EnhancedSecurityV2Middleware())
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

func TestEnhancedSecurityV2MiddlewareValidRequest(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewSecurityV2Config("test-secret-key-v2-12345")

	router := gin.New()
	router.Use(EnhancedSecurityV2Middleware(config))
	router.POST("/api/v2/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	body := []byte(`{"test": "data"}`)
	timestamp := time.Now().Unix()
	nonce := "middleware-test-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)
	version := "v2"

	security := NewEnhancedSecurityV2(config)
	req := CreateRequestForSignature("POST", "/api/v2/test", "", body, timestamp, nonce, version)
	signature, _ := security.EnhancedSignature(context.Background(), req)

	req2 := httptest.NewRequest("POST", "/api/v2/test", bytes.NewReader(body))
	req2.Header.Set("Content-Type", "application/json")
	req2.Header.Set("X-Signature-V2", signature)
	req2.Header.Set("X-Timestamp-V2", strconv.FormatInt(timestamp, 10))
	req2.Header.Set("X-Nonce-V2", nonce)
	req2.Header.Set("X-API-Version", version)

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Errorf("Valid request should return 200, got %d", w.Code)
	}
}

func TestGenerateSecurityV2Nonce(t *testing.T) {
	nonce, err := GenerateSecurityV2Nonce(32)
	if err != nil {
		t.Fatalf("GenerateSecurityV2Nonce failed: %v", err)
	}

	if len(nonce) < 16 {
		t.Error("Nonce length should be at least 16")
	}

	nonce2, _ := GenerateSecurityV2Nonce(32)
	if nonce == nonce2 {
		t.Error("Generated nonces should be unique")
	}
}

func TestGenerateSecurityV2NonceLengthValidation(t *testing.T) {
	nonce, _ := GenerateSecurityV2Nonce(8)
	if len(nonce) != 32 {
		t.Errorf("Nonce should default to 32 when less, got %d", len(nonce))
	}

	nonce, _ = GenerateSecurityV2Nonce(100)
	if len(nonce) != 64 {
		t.Errorf("Nonce should be capped at 64, got %d", len(nonce))
	}
}

func TestGenerateSecurityV2Timestamp(t *testing.T) {
	timestamp := GenerateSecurityV2Timestamp()

	if timestamp <= 0 {
		t.Error("Timestamp should be positive")
	}

	if timestamp != time.Now().Unix() {
		t.Error("Timestamp should match current time")
	}
}

func TestGenerateSecurityV2TimestampMillis(t *testing.T) {
	timestamp := GenerateSecurityV2TimestampMillis()

	if timestamp <= 0 {
		t.Error("Timestamp in millis should be positive")
	}

	nowMillis := time.Now().UnixMilli()
	if timestamp < nowMillis-1000 || timestamp > nowMillis+1000 {
		t.Error("Timestamp should be close to current time in milliseconds")
	}
}

func TestGenerateEd25519KeyPairV2(t *testing.T) {
	publicKey, privateKey, err := GenerateEd25519KeyPairV2()
	if err != nil {
		t.Fatalf("GenerateEd25519KeyPairV2 failed: %v", err)
	}

	if len(publicKey) != 32 {
		t.Errorf("Public key should be 32 bytes, got %d", len(publicKey))
	}

	if len(privateKey) != 64 {
		t.Errorf("Private key should be 64 bytes, got %d", len(privateKey))
	}
}

func TestVerifyTimestampRange(t *testing.T) {
	tolerance := 5 * time.Minute

	err := VerifyTimestampRange(time.Now().Unix(), tolerance)
	if err != nil {
		t.Error("Current timestamp should be within range")
	}

	err = VerifyTimestampRange(time.Now().Add(-3*time.Minute).Unix(), tolerance)
	if err != nil {
		t.Error("3 minutes ago should be within range")
	}

	err = VerifyTimestampRange(time.Now().Add(-10*time.Minute).Unix(), tolerance)
	if err == nil {
		t.Error("10 minutes ago should be out of range")
	}
}

func TestGenerateNonceWithChecksum(t *testing.T) {
	nonce, err := GenerateNonceWithChecksum(32)
	if err != nil {
		t.Fatalf("GenerateNonceWithChecksum failed: %v", err)
	}

	if len(nonce) != 32 {
		t.Errorf("Nonce should be 32 characters, got %d", len(nonce))
	}

	if !VerifyNonceWithChecksum(nonce) {
		t.Error("Generated nonce should pass verification")
	}
}

func TestVerifyNonceWithChecksumInvalid(t *testing.T) {
	if VerifyNonceWithChecksum("short") {
		t.Error("Short nonce should fail verification")
	}

	invalidNonce := "abcdefghijklmnopqrstuvw12" + "xxxx"
	if VerifyNonceWithChecksum(invalidNonce) {
		t.Error("Tampered nonce should fail verification")
	}
}

func TestDistributedTokenBucket(t *testing.T) {
	dtb := NewDistributedTokenBucket("test-key", 100.0, 10)

	if dtb.Key != "test-key" {
		t.Error("Key should match")
	}
	if dtb.Rate != 100.0 {
		t.Error("Rate should match")
	}
	if dtb.Burst != 10 {
		t.Error("Burst should match")
	}
	if dtb.RedisKey == "" {
		t.Error("RedisKey should not be empty")
	}
}

func TestNonceCacheV2(t *testing.T) {
	cache := newNonceCacheV2(100)

	nonce := "cache-test-nonce-12345"

	exists, rec := cache.get(nonce)
	if exists {
		t.Error("New nonce should not exist in cache")
	}

	cache.set(nonce)

	exists, rec = cache.get(nonce)
	if !exists {
		t.Error("Nonce should exist in cache after set")
	}
	if rec == nil {
		t.Error("Record should not be nil")
	}
	if rec.count != 1 {
		t.Errorf("Count should be 1, got %d", rec.count)
	}

	cache.increment(nonce)
	cache.increment(nonce)

	exists, rec = cache.get(nonce)
	if rec.count != 3 {
		t.Errorf("Count should be 3 after increments, got %d", rec.count)
	}
}

func TestNonceCacheV2Shrink(t *testing.T) {
	cache := newNonceCacheV2(5)

	for i := 0; i < 10; i++ {
		cache.set(fmt.Sprintf("nonce-%d", i))
	}

	if len(cache.records) > 5 {
		t.Errorf("Cache should shrink to limit, got %d records", len(cache.records))
	}
}

func TestNonceCacheV2Cleanup(t *testing.T) {
	cache := newNonceCacheV2(100)

	for i := 0; i < 5; i++ {
		cache.set(fmt.Sprintf("old-nonce-%d", i))
	}

	cache.cleanup()

	if len(cache.records) > 0 {
		t.Errorf("Cache should be cleaned up, got %d records", len(cache.records))
	}
}

func TestSecureCompareV2(t *testing.T) {
	if !secureCompareV2("test", "test") {
		t.Error("Equal strings should compare as equal")
	}

	if secureCompareV2("test", "different") {
		t.Error("Different strings should not compare as equal")
	}

	if secureCompareV2("test", "tes") {
		t.Error("Different length strings should not compare as equal")
	}
}

func TestCreateRequestForSignature(t *testing.T) {
	req := CreateRequestForSignature(
		"POST",
		"/api/v2/test",
		"foo=bar",
		[]byte(`{"data": "test"}`),
		time.Now().Unix(),
		"test-nonce",
		"v2",
	)

	if req.Method != "POST" {
		t.Error("Method should be POST")
	}
	if req.Path != "/api/v2/test" {
		t.Error("Path should match")
	}
	if req.Query != "foo=bar" {
		t.Error("Query should match")
	}
	if string(req.Body) != `{"data": "test"}` {
		t.Error("Body should match")
	}
	if req.Version != "v2" {
		t.Error("Version should be v2")
	}
}

func TestValidateRequest(t *testing.T) {
	security := NewEnhancedSecurityV2()

	timestamp := time.Now().Unix()
	nonce := "validate-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   "v2",
	}

	signature, _ := security.EnhancedSignature(context.Background(), req)

	valid, reason := security.ValidateRequest(req, signature)
	if !valid {
		t.Errorf("ValidateRequest should succeed: %s", reason)
	}
}

func TestValidateRequestNil(t *testing.T) {
	security := NewEnhancedSecurityV2()

	valid, reason := security.ValidateRequest(nil, "signature")
	if valid {
		t.Error("Nil request should not be valid")
	}
	if reason != "nil request" {
		t.Errorf("Reason should be 'nil request', got %s", reason)
	}
}

func TestValidateRequestMissingMethod(t *testing.T) {
	security := NewEnhancedSecurityV2()

	req := &Request{
		Path:      "/api/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "missing-method-nonce",
	}

	valid, reason := security.ValidateRequest(req, "signature")
	if valid {
		t.Error("Missing method should not be valid")
	}
	if reason != "missing method" {
		t.Errorf("Reason should be 'missing method', got %s", reason)
	}
}

func TestSignatureContext(t *testing.T) {
	ctx := NewSignatureContext()

	if ctx.Security == nil {
		t.Error("Security should not be nil")
	}

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: time.Now().Unix(),
		Nonce:     "sigctx-nonce-12345",
		Version:   "v2",
	}

	signature, err := ctx.SignRequest(req)
	if err != nil {
		t.Fatalf("SignRequest failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	result := ctx.VerifyRequest(req, signature)
	if !result.Valid {
		t.Error("VerifyRequest should succeed")
	}

	elapsed := ctx.GetElapsedTime()
	if elapsed < 0 {
		t.Error("Elapsed time should be non-negative")
	}
}

func TestSignatureContextVerifyInvalid(t *testing.T) {
	ctx := NewSignatureContext()

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: time.Now().Unix(),
		Nonce:     "invalid-verify-nonce",
		Version:   "v2",
	}

	result := ctx.VerifyRequest(req, "invalid-signature")
	if result.Valid {
		t.Error("Invalid signature should not be valid")
	}
}

func TestVersionRouterDefault(t *testing.T) {
	vr := newVersionRouter([]string{"v1", "v2"}, "v2")

	defaultV := vr.getVersion()
	if defaultV != "v2" {
		t.Errorf("Default version should be v2, got %s", defaultV)
	}
}

func TestTokenBucket(t *testing.T) {
	tb := &tokenBucket{
		tokens:     10,
		lastRefill: time.Now(),
		rate:       100.0,
		burst:      10,
	}

	if tb.tokens != 10 {
		t.Error("Initial tokens should be 10")
	}

	tb.mu.Lock()
	tb.tokens--
	tb.mu.Unlock()

	if tb.tokens != 9 {
		t.Errorf("Tokens should be 9 after decrement, got %f", tb.tokens)
	}
}

func TestVersionRouterMultipleVersions(t *testing.T) {
	vr := newVersionRouter([]string{"v1", "v2", "v3"}, "v1")

	versions := []string{"v1", "v2", "v3", "v4"}
	expected := []bool{true, true, true, false}

	for i, v := range versions {
		if vr.isVersionSupported(v) != expected[i] {
			t.Errorf("Version %s support should be %v", v, expected[i])
		}
	}
}

func TestSecurityMetrics(t *testing.T) {
	security := NewEnhancedSecurityV2()

	IncrementSignatureVerifications()
	IncrementReplayDetections()
	IncrementRateLimitRejections()
	IncrementVersionRoutings()

	metrics := security.GetMetrics()

	if metrics.SignatureVerifications != 1 {
		t.Errorf("SignatureVerifications should be 1, got %d", metrics.SignatureVerifications)
	}
	if metrics.ReplayDetections != 1 {
		t.Errorf("ReplayDetections should be 1, got %d", metrics.ReplayDetections)
	}
	if metrics.RateLimitRejections != 1 {
		t.Errorf("RateLimitRejections should be 1, got %d", metrics.RateLimitRejections)
	}
	if metrics.VersionRoutings != 1 {
		t.Errorf("VersionRoutings should be 1, got %d", metrics.VersionRoutings)
	}
}

func TestBuildStringToSign(t *testing.T) {
	security := NewEnhancedSecurityV2()

	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Query:     "foo=bar&baz=qux",
		Body:      []byte(`{"data": "test"}`),
		Timestamp: 1234567890,
		Nonce:     "test-nonce",
		Version:   "v2",
	}

	stringToSign := security.buildStringToSign(req)

	if stringToSign == "" {
		t.Error("String to sign should not be empty")
	}

	if !containsV2(stringToSign, "POST") {
		t.Error("String to sign should contain method")
	}
	if !containsV2(stringToSign, "/api/v2/test") {
		t.Error("String to sign should contain path")
	}
	if !containsV2(stringToSign, "foo=bar&baz=qux") {
		t.Error("String to sign should contain sorted query")
	}
}

func TestSortQueryString(t *testing.T) {
	security := NewEnhancedSecurityV2()

	query := "z=3&a=1&m=2"
	sorted := security.sortQueryString(query)

	expected := "a=1&m=2&z=3"
	if sorted != expected {
		t.Errorf("Expected '%s', got '%s'", expected, sorted)
	}
}

func TestHashBody(t *testing.T) {
	security := NewEnhancedSecurityV2()

	body := []byte(`{"test": "data"}`)

	hash1 := security.hashBody(body)
	hash2 := security.hashBody(body)

	if hash1 != hash2 {
		t.Error("Same body should produce same hash")
	}

	hash3 := security.hashBody([]byte(`{"different": "data"}`))
	if hash1 == hash3 {
		t.Error("Different bodies should produce different hashes")
	}

	emptyHash := security.hashBody(nil)
	if emptyHash != "" {
		t.Error("Empty body should produce empty hash")
	}
}

func TestGetHandlerForVersion(t *testing.T) {
	security := NewEnhancedSecurityV2()

	handlers := map[string]string{
		"v1": "legacyHandler",
		"v2": "standardHandler",
		"v3": "advancedHandler",
	}

	for version, expected := range handlers {
		handler := security.getHandlerForVersion(version)
		if handler != expected {
			t.Errorf("Version %s: expected %s, got %s", version, expected, handler)
		}
	}
}

func TestEnhancedSecurityV2MiddlewareInvalidTimestamp(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewSecurityV2Config("test-secret-key-v2-12345")

	router := gin.New()
	router.Use(EnhancedSecurityV2Middleware(config))
	router.GET("/api/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	req := httptest.NewRequest("GET", "/api/test", nil)
	req.Header.Set("X-Signature-V2", "some-signature")
	req.Header.Set("X-Timestamp-V2", "invalid-timestamp")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Code != http.StatusUnauthorized {
		t.Errorf("Invalid timestamp should return 401, got %d", w.Code)
	}
}

func TestEnhancedSecurityV2MiddlewareVersionRouting(t *testing.T) {
	gin.SetMode(gin.TestMode)

	config := NewSecurityV2Config("test-secret-key-v2-12345")

	router := gin.New()
	router.Use(EnhancedSecurityV2Middleware(config))
	router.GET("/api/v2/test", func(c *gin.Context) {
		c.String(200, "OK")
	})

	timestamp := time.Now().Unix()
	nonce := "version-test-nonce-" + strconv.FormatInt(time.Now().UnixNano(), 10)

	security := NewEnhancedSecurityV2(config)
	req := CreateRequestForSignature("GET", "/api/v2/test", "", nil, timestamp, nonce, "v2")
	signature, _ := security.EnhancedSignature(context.Background(), req)

	req2 := httptest.NewRequest("GET", "/api/v2/test", nil)
	req2.Header.Set("X-Signature-V2", signature)
	req2.Header.Set("X-Timestamp-V2", strconv.FormatInt(timestamp, 10))
	req2.Header.Set("X-Nonce-V2", nonce)
	req2.Header.Set("X-API-Version", "v2")

	w := httptest.NewRecorder()
	router.ServeHTTP(w, req2)

	if w.Code != http.StatusOK {
		t.Errorf("Valid request should return 200, got %d", w.Code)
	}

	deprecationHeader := w.Header().Get("X-API-Deprecation")
	if deprecationHeader != "" {
		t.Error("v2 should not have deprecation notice")
	}
}

func containsV2(s, substr string) bool {
	return len(s) >= len(substr) && containsHelperV2(s, substr)
}

func containsHelperV2(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func TestRateLimitResultFields(t *testing.T) {
	result := &RateLimitResult{
		Allowed:     true,
		Remaining:   99,
		Limit:       100,
		ResetAt:     time.Now().Add(1 * time.Second),
		RetryAfter:  0,
		IsDistributed: false,
	}

	if !result.Allowed {
		t.Error("Result should be allowed")
	}
	if result.Remaining != 99 {
		t.Errorf("Remaining should be 99, got %d", result.Remaining)
	}
}

func TestReplayCheckResultFields(t *testing.T) {
	result := &ReplayCheckResult{
		Allowed:         true,
		IsReplay:        false,
		TimestampValid:  true,
		NonceValid:      true,
		RemainingWindow: 300,
		TTL:             24 * time.Hour,
	}

	if !result.Allowed {
		t.Error("Result should be allowed")
	}
	if result.IsReplay {
		t.Error("Should not be replay")
	}
	if !result.TimestampValid {
		t.Error("Timestamp should be valid")
	}
}

func TestVersionRouteResultFields(t *testing.T) {
	result := &VersionRouteResult{
		Version:           "v2",
		Compatible:        true,
		Handler:           "standardHandler",
		DeprecationNotice: "",
	}

	if result.Version != "v2" {
		t.Errorf("Version should be v2, got %s", result.Version)
	}
	if result.Handler != "standardHandler" {
		t.Errorf("Handler should be standardHandler, got %s", result.Handler)
	}
}

func TestSignatureResultFields(t *testing.T) {
	result := &SignatureResult{
		Valid:        true,
		Algorithm:    "HMAC-SHA3-256",
		Signature:    "abc123",
		Timestamp:    time.Now().Unix(),
		Nonce:        "test-nonce",
		Version:      "v2",
		ErrorCode:    "",
		ErrorReason:  "",
		ElapsedTime:  time.Millisecond * 100,
	}

	if !result.Valid {
		t.Error("Result should be valid")
	}
	if result.Algorithm != "HMAC-SHA3-256" {
		t.Errorf("Algorithm should be HMAC-SHA3-256, got %s", result.Algorithm)
	}
}

func TestRequestFields(t *testing.T) {
	req := &Request{
		Method:    "POST",
		Path:      "/api/v2/test",
		Query:     "foo=bar",
		Body:      []byte(`{"data": "test"}`),
		Headers:   map[string]string{"X-Custom": "header"},
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce",
		Version:   "v2",
	}

	if req.Method != "POST" {
		t.Error("Method should be POST")
	}
	if req.Headers["X-Custom"] != "header" {
		t.Error("Headers should contain X-Custom")
	}
}

func TestSecurityV2Version(t *testing.T) {
	if SecurityV2Version != "2.0" {
		t.Errorf("SecurityV2Version should be 2.0, got %s", SecurityV2Version)
	}
}
