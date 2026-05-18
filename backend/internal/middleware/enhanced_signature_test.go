package middleware

import (
	"testing"
	"time"
)

func TestNewEnhancedSignature(t *testing.T) {
	sig := NewEnhancedSignature()
	if sig == nil {
		t.Fatal("EnhancedSignature should be created")
	}
	if sig.config.Algorithm != AlgorithmHMACSHA512 {
		t.Error("Default algorithm should be HMAC-SHA512")
	}
}

func TestNewEnhancedSignatureWithConfig(t *testing.T) {
	config := SignatureConfig{
		SecretKey:  []byte("custom-key-1234567890"),
		Algorithm: AlgorithmHMACSHA256,
	}
	sig := NewEnhancedSignature(config)
	if sig.config.Algorithm != AlgorithmHMACSHA256 {
		t.Error("Algorithm should be HMAC-SHA256")
	}
}

func TestGenerateSignature(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey: []byte("test-key-1234567890"),
	})

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-123",
		Body:      []byte(`{"data":"test"}`),
	}

	signature, err := sig.GenerateSignature(req)
	if err != nil {
		t.Fatalf("GenerateSignature failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestGenerateSignatureWithNonce(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:  []byte("test-key-1234567890"),
		EnableNonce: true,
	})

	req := SignatureRequest{
		Method:    "GET",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
	}

	signature, err := sig.GenerateSignature(req)
	if err != nil {
		t.Fatalf("GenerateSignature failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestVerifySignature(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping test in short mode")
	}
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:           []byte("test-key-1234567890"),
		TimestampTolerance:  5 * time.Minute,
		EnableReplayCheck:   false,
	})

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-123",
		Body:      []byte(`{"data":"test"}`),
	}

	sigCopy := *sig
	signature, _ := sigCopy.GenerateSignature(req)

	err := sigCopy.VerifySignature(req, signature)
	if err != nil {
		t.Errorf("VerifySignature failed: %v", err)
	}
}

func TestVerifySignatureInvalid(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:           []byte("test-key-1234567890"),
		TimestampTolerance:  5 * time.Minute,
		EnableReplayCheck:   false,
	})

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-123",
		Body:      []byte(`{"data":"test"}`),
	}

	err := sig.VerifySignature(req, "invalid-signature")
	if err == nil {
		t.Error("Should fail verification with invalid signature")
	}
}

func TestVerifySignatureExpiredTimestamp(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:           []byte("test-key-1234567890"),
		TimestampTolerance:  1 * time.Second,
		EnableReplayCheck:   false,
	})

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Add(-10 * time.Second).Unix(),
		Nonce:     "test-nonce-123",
	}

	signature, _ := sig.GenerateSignature(req)

	err := sig.VerifySignature(req, signature)
	if err == nil {
		t.Error("Should fail verification with expired timestamp")
	}
}

func TestReplayProtection(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:           []byte("test-key-1234567890"),
		TimestampTolerance:  5 * time.Minute,
		EnableReplayCheck:   true,
		EnableNonce:        true,
	})

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "unique-nonce-123",
	}

	signature, _ := sig.GenerateSignature(req)

	err := sig.VerifySignature(req, signature)
	if err != nil {
		t.Errorf("First verification should succeed: %v", err)
	}

	err = sig.VerifySignature(req, signature)
	if err == nil {
		t.Error("Second verification should fail (replay attack)")
	}
}

func TestGenerateNonce(t *testing.T) {
	nonce, err := GenerateNonce(32)
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	if len(nonce) < 16 {
		t.Error("Nonce should be at least 16 characters")
	}

	nonce2, _ := GenerateNonce(32)
	if nonce == nonce2 {
		t.Error("Nonces should be unique")
	}
}

func TestGenerateSecureNonce(t *testing.T) {
	nonce, err := GenerateSecureNonce()
	if err != nil {
		t.Fatalf("GenerateSecureNonce failed: %v", err)
	}

	if len(nonce) < 32 {
		t.Error("Secure nonce should be at least 32 hex characters")
	}
}

func TestGenerateTimestampNonce(t *testing.T) {
	timestamp, nonce, err := GenerateTimestampNonce()
	if err != nil {
		t.Fatalf("GenerateTimestampNonce failed: %v", err)
	}

	if timestamp == 0 {
		t.Error("Timestamp should not be zero")
	}

	if nonce == "" {
		t.Error("Nonce should not be empty")
	}
}

func TestGenerateHMACSignature(t *testing.T) {
	data := []byte("test data")
	key := []byte("test-key-1234567890")

	signature, err := GenerateHMACSignature(data, key, AlgorithmHMACSHA512)
	if err != nil {
		t.Fatalf("GenerateHMACSignature failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestGenerateHMACSignatureEmptyKey(t *testing.T) {
	data := []byte("test data")

	_, err := GenerateHMACSignature(data, nil, AlgorithmHMACSHA512)
	if err == nil {
		t.Error("Should fail with empty key")
	}
}

func TestVerifyHMACSignature(t *testing.T) {
	data := []byte("test data")
	key := []byte("test-key-1234567890")

	signature, _ := GenerateHMACSignature(data, key, AlgorithmHMACSHA512)

	valid, err := VerifyHMACSignature(data, []byte(signature), key, AlgorithmHMACSHA512)
	if err != nil {
		t.Fatalf("VerifyHMACSignature failed: %v", err)
	}

	if !valid {
		t.Error("Valid signature should verify")
	}
}

func TestVerifyHMACSignatureInvalid(t *testing.T) {
	data := []byte("test data")
	key := []byte("test-key-1234567890")

	signature, _ := GenerateHMACSignature(data, key, AlgorithmHMACSHA512)

	valid, _ := VerifyHMACSignature([]byte("modified data"), []byte(signature), key, AlgorithmHMACSHA512)
	if valid {
		t.Error("Modified data should not verify")
	}
}

func TestGenerateSignatureWithExpiry(t *testing.T) {
	data := []byte("test data")
	key := []byte("test-key-1234567890")
	expiry := 5 * time.Minute

	signature, expiryTimestamp, err := GenerateSignatureWithExpiry(data, key, expiry)
	if err != nil {
		t.Fatalf("GenerateSignatureWithExpiry failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	if expiryTimestamp == 0 {
		t.Error("Expiry timestamp should not be zero")
	}
}

func TestVerifySignatureWithExpiry(t *testing.T) {
	data := []byte("test data")
	key := []byte("test-key-1234567890")
	expiry := 5 * time.Minute

	signature, expiryTimestamp, _ := GenerateSignatureWithExpiry(data, key, expiry)

	valid, err := VerifySignatureWithExpiry(data, signature, key, expiryTimestamp)
	if err != nil {
		t.Fatalf("VerifySignatureWithExpiry failed: %v", err)
	}

	if !valid {
		t.Error("Valid signature should verify")
	}
}

func TestVerifySignatureWithExpiryExpired(t *testing.T) {
	data := []byte("test data")
	key := []byte("test-key-1234567890")
	expiry := 1 * time.Millisecond

	signature, expiryTimestamp, _ := GenerateSignatureWithExpiry(data, key, expiry)

	time.Sleep(10 * time.Millisecond)

	_, err := VerifySignatureWithExpiry(data, signature, key, expiryTimestamp)
	if err == nil {
		t.Error("Should fail with expired signature")
	}
}

func TestReplayProtectionStruct(t *testing.T) {
	rp := NewReplayProtection(1000, 5*time.Minute)

	nonce := "test-nonce-123"
	timestamp := time.Now().Unix()

	ok, err := rp.Check(nonce, timestamp)
	if err != nil {
		t.Fatalf("Check failed: %v", err)
	}

	if !ok {
		t.Error("First check should succeed")
	}
}

func TestReplayProtectionStructReplay(t *testing.T) {
	rp := NewReplayProtection(1000, 5*time.Minute)

	nonce := "test-nonce-123"
	timestamp := time.Now().Unix()

	rp.Check(nonce, timestamp)

	ok, err := rp.Check(nonce, timestamp)
	if err == nil {
		t.Error("Second check should fail (replay)")
	}

	if ok {
		t.Error("Second check should return false")
	}
}

func TestReplayProtectionStructClear(t *testing.T) {
	rp := NewReplayProtection(1000, 5*time.Minute)

	nonce := "test-nonce-123"
	timestamp := time.Now().Unix()

	rp.Check(nonce, timestamp)

	rp.Clear()

	ok, err := rp.Check(nonce, timestamp)
	if err != nil {
		t.Fatalf("Check after clear failed: %v", err)
	}

	if !ok {
		t.Error("Check should succeed after clear")
	}
}

func TestReplayProtectionStructSize(t *testing.T) {
	rp := NewReplayProtection(1000, 5*time.Minute)

	rp.Check("nonce1", time.Now().Unix())
	rp.Check("nonce2", time.Now().Unix())

	if rp.Size() != 2 {
		t.Error("Size should be 2")
	}
}

func TestSignatureBuilder(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey: []byte("test-key-1234567890"),
	})

	builder := NewSignatureBuilder(sig)
	builder.SetMethod("POST").
		SetPath("/api/v1/test").
		SetBody([]byte(`{"data":"test"}`))

	signature, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestSignatureBuilderWithTimestamp(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey: []byte("test-key-1234567890"),
	})

	timestamp := time.Now().Unix()

	builder := NewSignatureBuilder(sig)
	builder.SetMethod("GET").
		SetPath("/api/v1/test").
		SetTimestamp(timestamp)

	signature, req, err := builder.Sign()
	if err != nil {
		t.Fatalf("Sign failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	if req.Timestamp != timestamp {
		t.Error("Timestamp should match")
	}
}

func TestSignatureValidator(t *testing.T) {
	validator := NewSignatureValidator()

	sig := validator.signature
	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-123",
	}

	signature, _ := sig.GenerateSignature(req)

	err := validator.Validate(req, signature)
	if err != nil {
		t.Errorf("Validation failed: %v", err)
	}
}

func TestSignatureValidatorStrictMode(t *testing.T) {
	validator := NewSignatureValidator()
	validator.SetStrictMode(true)

	sig := validator.signature
	req := SignatureRequest{
		Method: "POST",
		Path:   "/api/v1/test",
	}

	signature, _ := sig.GenerateSignature(req)

	err := validator.Validate(req, signature)
	if err == nil {
		t.Error("Strict mode should require timestamp")
	}
}

func TestSignatureValidatorAllowExpired(t *testing.T) {
	validator := NewSignatureValidator()
	validator.SetAllowExpired(true)

	sig := validator.signature
	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Add(-10 * time.Minute).Unix(),
		Nonce:     "test-nonce-123",
	}

	signature, _ := sig.GenerateSignature(req)

	err := validator.Validate(req, signature)
	if err != nil {
		t.Errorf("Allow expired should pass: %v", err)
	}
}

func TestSignatureStats(t *testing.T) {
	sig := NewEnhancedSignature()

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-123",
	}

	signature, _ := sig.GenerateSignature(req)
	sig.VerifySignature(req, signature)

	stats := sig.GetStats()

	if stats.TotalRequests == 0 {
		t.Error("TotalRequests should be incremented")
	}

	if stats.ValidSignatures == 0 {
		t.Error("ValidSignatures should be incremented")
	}
}

func TestSignatureStatsInvalid(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:          []byte("test-key-1234567890"),
		EnableReplayCheck: false,
	})

	req := SignatureRequest{
		Method:    "POST",
		Path:      "/api/v1/test",
		Timestamp: time.Now().Unix(),
		Nonce:     "test-nonce-123",
	}

	sig.VerifySignature(req, "invalid-signature")

	stats := sig.GetStats()

	if stats.InvalidSignatures == 0 {
		t.Error("InvalidSignatures should be incremented")
	}
}

func TestSignatureStatsReset(t *testing.T) {
	sig := NewEnhancedSignature()

	sig.GetStats()

	sig.ResetStats()

	stats := sig.GetStats()

	if stats.TotalRequests != 0 {
		t.Error("Stats should be reset")
	}
}

func TestSignatureBuilderQueryParams(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey: []byte("test-key-1234567890"),
	})

	builder := NewSignatureBuilder(sig)
	builder.SetMethod("GET").
		SetPath("/api/v1/test").
		SetQueryParam("key1", "value1").
		SetQueryParam("key2", "value2")

	signature, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestSignatureBuilderHeaders(t *testing.T) {
	sig := NewEnhancedSignature(SignatureConfig{
		SecretKey:       []byte("test-key-1234567890"),
		CustomHeaders:   []string{"X-Custom-Header"},
	})

	builder := NewSignatureBuilder(sig)
	builder.SetMethod("POST").
		SetPath("/api/v1/test").
		SetHeader("X-Custom-Header", "custom-value")

	signature, err := builder.Build()
	if err != nil {
		t.Fatalf("Build failed: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}
}

func TestSignatureError(t *testing.T) {
	err := NewSignatureError("TEST_CODE", "Test message")

	if err.Code != "TEST_CODE" {
		t.Error("Error code should match")
	}

	if err.Message != "Test message" {
		t.Error("Error message should match")
	}

	if err.Error() != "[TEST_CODE] Test message" {
		t.Error("Error string should match")
	}
}

func TestSignatureErrorWithDetail(t *testing.T) {
	err := NewSignatureError("TEST_CODE", "Test message")
	err.WithDetail("key", "value")

	if err.Details["key"] != "value" {
		t.Error("Detail should be set")
	}
}

func TestCreateMissingSignatureError(t *testing.T) {
	err := CreateMissingSignatureError()
	if err.Error() != "[MISSING_SIGNATURE] signature header is missing" {
		t.Error("Error message should match")
	}
}

func TestCreateInvalidSignatureError(t *testing.T) {
	err := CreateInvalidSignatureError()
	if err.Error() != "[INVALID_SIGNATURE] signature verification failed" {
		t.Error("Error message should match")
	}
}

func TestCreateExpiredTimestampError(t *testing.T) {
	err := CreateExpiredTimestampError(1234567890)
	if err == nil {
		t.Error("Error should be created")
	}
}

func TestCreateReplayDetectedError(t *testing.T) {
	err := CreateReplayDetectedError("test-nonce")
	if err == nil {
		t.Error("Error should be created")
	}
}

func TestCreateMissingNonceError(t *testing.T) {
	err := CreateMissingNonceError()
	if err.Error() != "[MISSING_NONCE] nonce header is missing" {
		t.Error("Error message should match")
	}
}

func TestCreateMissingTimestampError(t *testing.T) {
	err := CreateMissingTimestampError()
	if err.Error() != "[MISSING_TIMESTAMP] timestamp header is missing" {
		t.Error("Error message should match")
	}
}

func TestVerifyTimestampMissing(t *testing.T) {
	sig := NewEnhancedSignature()

	err := sig.verifyTimestamp(0)
	if err == nil {
		t.Error("Should fail with missing timestamp")
	}
}

func TestCheckReplayMissingNonce(t *testing.T) {
	sig := NewEnhancedSignature()

	err := sig.checkReplay("", time.Now().Unix())
	if err == nil {
		t.Error("Should fail with missing nonce")
	}
}

func TestBuildSignatureData(t *testing.T) {
	sig := NewEnhancedSignature()

	req := SignatureRequest{
		Method:      "POST",
		Path:        "/api/v1/test",
		Timestamp:   1234567890,
		Nonce:       "test-nonce",
		QueryParams: map[string]string{"key": "value"},
		Headers:     map[string]string{"X-Custom": "value"},
		Body:        []byte(`{"data":"test"}`),
	}

	data := sig.buildSignatureData(req)

	if data == "" {
		t.Error("Signature data should not be empty")
	}

	if !contains(data, "POST") {
		t.Error("Should contain method")
	}

	if !contains(data, "/api/v1/test") {
		t.Error("Should contain path")
	}

	if !contains(data, "test-nonce") {
		t.Error("Should contain nonce")
	}
}

func TestSortQueryParams(t *testing.T) {
	sig := NewEnhancedSignature()

	params := map[string]string{
		"z": "value1",
		"a": "value2",
		"m": "value3",
	}

	sorted := sig.sortQueryParams(params)

	if !contains(sorted, "a=value2") {
		t.Error("Should contain sorted key 'a'")
	}

	if !contains(sorted, "m=value3") {
		t.Error("Should contain sorted key 'm'")
	}

	if !contains(sorted, "z=value1") {
		t.Error("Should contain sorted key 'z'")
	}
}

func TestHashBody(t *testing.T) {
	sig := NewEnhancedSignature()

	body := []byte(`{"data":"test"}`)
	hash := sig.hashBody(body)

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	hash2 := sig.hashBody(body)
	if hash != hash2 {
		t.Error("Same body should produce same hash")
	}

	hash3 := sig.hashBody([]byte(`{"data":"different"}`))
	if hash == hash3 {
		t.Error("Different body should produce different hash")
	}
}

func TestBuildNonceKey(t *testing.T) {
	sig := NewEnhancedSignature()

	nonce := "test-nonce"
	timestamp := int64(1234567890)

	key := sig.buildNonceKey(nonce, timestamp)

	expected := "test-nonce:1234567890"
	if key != expected {
		t.Errorf("Expected %s, got %s", expected, key)
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
