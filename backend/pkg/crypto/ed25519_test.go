package crypto

import (
	"encoding/base64"
	"fmt"
	"testing"
	"time"
)

func TestEd25519GenerateKeyPair(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key pair: %v", err)
	}

	if len(privateKey) != ed25519.PrivateKeySize {
		t.Errorf("Expected private key size %d, got %d", ed25519.PrivateKeySize, len(privateKey))
	}

	if len(publicKey) != ed25519.PublicKeySize {
		t.Errorf("Expected public key size %d, got %d", ed25519.PublicKeySize, len(publicKey))
	}
}

func TestEd25519SignAndVerify(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key pair: %v", err)
	}

	message := []byte("test message for signing")
	signature := ed25519.Sign(privateKey, message)

	if !ed25519.Verify(publicKey, message, signature) {
		t.Error("Signature verification failed")
	}

	wrongMessage := []byte("wrong message")
	if ed25519.Verify(publicKey, wrongMessage, signature) {
		t.Error("Signature should not verify for wrong message")
	}
}

func TestEd25519SignString(t *testing.T) {
	privateKey, _, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Ed25519 key pair: %v", err)
	}

	message := "test message"
	sig, err := RSASignString(message, nil)
	if err != nil && err != ErrPrivateKeyInvalid {
		t.Errorf("Unexpected error: %v", err)
	}
}

func TestEd25519Manager(t *testing.T) {
	manager, err := NewEd25519Manager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create Ed25519 manager: %v", err)
	}

	message := []byte("test message for manager")
	signature, keyID, err := manager.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if keyID == "" {
		t.Error("Key ID should not be empty")
	}

	valid, returnedKeyID := manager.Verify(message, signature)
	if !valid {
		t.Error("Signature should be valid")
	}

	if returnedKeyID != keyID {
		t.Errorf("Key ID mismatch: expected %s, got %s", keyID, returnedKeyID)
	}

	invalidMessage := []byte("invalid message")
	valid, _ = manager.Verify(invalidMessage, signature)
	if valid {
		t.Error("Signature should not be valid for different message")
	}
}

func TestEd25519ManagerRotation(t *testing.T) {
	manager, err := NewEd25519Manager(1 * time.Second)
	if err != nil {
		t.Fatalf("Failed to create Ed25519 manager: %v", err)
	}

	initialKeyID := manager.GetKeyID()
	if initialKeyID == "" {
		t.Error("Initial key ID should not be empty")
	}

	time.Sleep(2 * time.Second)

	err = manager.Rotate()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	newKeyID := manager.GetKeyID()
	if newKeyID == initialKeyID {
		t.Error("Key ID should have changed after rotation")
	}

	message := []byte("test message")
	oldSignature, _, _ := manager.Sign(message)
	valid, _ := manager.Verify(message, oldSignature)
	if !valid {
		t.Error("Should still verify with old key after rotation")
	}
}

func TestEd25519ManagerGetAllPublicKeys(t *testing.T) {
	manager, err := NewEd25519Manager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create Ed25519 manager: %v", err)
	}

	initialCount := len(manager.GetAllPublicKeys())

	err = manager.Rotate()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	newCount := len(manager.GetAllPublicKeys())
	if newCount != initialCount+1 {
		t.Errorf("Expected %d keys after rotation, got %d", initialCount+1, newCount)
	}
}

func TestEd25519ManagerSignString(t *testing.T) {
	manager, err := NewEd25519Manager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create Ed25519 manager: %v", err)
	}

	message := "test message string"
	signature, keyID, err := manager.SignString(message)
	if err != nil {
		t.Fatalf("Failed to sign string: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	valid, returnedKeyID := manager.VerifyString(message, signature)
	if !valid {
		t.Error("String signature should be valid")
	}

	if returnedKeyID != keyID {
		t.Errorf("Key ID mismatch: expected %s, got %s", keyID, returnedKeyID)
	}
}

func TestEd25519ManagerGetKeyInfo(t *testing.T) {
	manager, err := NewEd25519Manager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create Ed25519 manager: %v", err)
	}

	info := manager.GetKeyInfo()
	if info == nil {
		t.Error("Key info should not be nil")
	}

	if _, ok := info["current_key"]; !ok {
		t.Error("Key info should contain current_key")
	}

	if info["total_keys"] == nil || info["total_keys"].(int) < 1 {
		t.Error("Key info should contain at least one key")
	}
}

func TestParseEd25519PrivateKey(t *testing.T) {
	privateKey, _, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	pemData := ExportEd25519PrivateKey(privateKey)
	parsedKey, err := ParseEd25519PrivateKey(pemData)
	if err != nil {
		t.Fatalf("Failed to parse private key: %v", err)
	}

	if len(parsedKey) != len(privateKey) {
		t.Error("Parsed key length mismatch")
	}

	message := []byte("test message")
	sig1 := ed25519.Sign(privateKey, message)
	sig2 := ed25519.Sign(parsedKey, message)

	for i := range sig1 {
		if sig1[i] != sig2[i] {
			t.Error("Signature mismatch between original and parsed key")
			break
		}
	}
}

func TestParseEd25519PublicKey(t *testing.T) {
	_, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	pemData := ExportEd25519PublicKey(publicKey)
	parsedKey, err := ParseEd25519PublicKey(pemData)
	if err != nil {
		t.Fatalf("Failed to parse public key: %v", err)
	}

	if len(parsedKey) != len(publicKey) {
		t.Error("Parsed key length mismatch")
	}
}

func TestParseEd25519PrivateKeyInvalid(t *testing.T) {
	_, err := ParseEd25519PrivateKey("invalid pem data")
	if err == nil {
		t.Error("Should fail for invalid PEM data")
	}
}

func TestParseEd25519PublicKeyInvalid(t *testing.T) {
	_, err := ParseEd25519PublicKey("invalid pem data")
	if err == nil {
		t.Error("Should fail for invalid PEM data")
	}
}

func TestEd25519ManagerConcurrent(t *testing.T) {
	manager, err := NewEd25519Manager(5 * time.Minute)
	if err != nil {
		t.Fatalf("Failed to create Ed25519 manager: %v", err)
	}

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func() {
			message := []byte(fmt.Sprintf("test message %d", i))
			signature, keyID, err := manager.Sign(message)
			if err != nil {
				t.Errorf("Sign error: %v", err)
				done <- false
				return
			}
			valid, _ := manager.Verify(message, signature)
			if !valid {
				t.Error("Verification failed")
				done <- false
				return
			}
			if keyID == "" {
				t.Error("Key ID should not be empty")
				done <- false
				return
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		if !<-done {
			t.Fail()
		}
	}
}

func TestSignatureAlgorithmSupport(t *testing.T) {
	privateKey, _, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("test message for algorithm")

	testCases := []struct {
		name      string
		algorithm SignatureAlgorithm
	}{
		{"HMAC-SHA256", AlgorithmHMACSHA256},
		{"HMAC-SHA512", AlgorithmHMACSHA512},
		{"Ed25519", AlgorithmEd25519},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := SignatureRequest{
				Algorithm:  tc.algorithm,
				Message:   message,
				SecretKey: "test-secret-key",
				PrivateKey: privateKey,
			}

			resp, err := Sign(req)
			if err != nil {
				t.Fatalf("Failed to sign with %s: %v", tc.name, err)
			}

			if resp.Signature == "" {
				t.Error("Signature should not be empty")
			}

			if resp.Algorithm != string(tc.algorithm) {
				t.Errorf("Algorithm mismatch: expected %s, got %s", tc.algorithm, resp.Algorithm)
			}
		})
	}
}

func TestVerifySignature(t *testing.T) {
	message := []byte("test message for verification")
	secretKey := "test-secret-key"

	testCases := []struct {
		name      string
		algorithm SignatureAlgorithm
	}{
		{"HMAC-SHA256", AlgorithmHMACSHA256},
		{"HMAC-SHA512", AlgorithmHMACSHA512},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			req := SignatureRequest{
				Algorithm:  tc.algorithm,
				Message:   message,
				SecretKey: secretKey,
			}

			sigResp, err := Sign(req)
			if err != nil {
				t.Fatalf("Failed to sign: %v", err)
			}

			verifyReq := VerifyRequest{
				Algorithm: tc.algorithm,
				Message:   message,
				Signature: sigResp.Signature,
				SecretKey: secretKey,
			}

			verifyResp := Verify(verifyReq)
			if !verifyResp.Valid {
				t.Error("Signature should be valid")
			}
		})
	}
}

func TestVerifySignatureWithEd25519(t *testing.T) {
	privateKey, publicKey, err := GenerateEd25519KeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	message := []byte("test message for Ed25519 verification")

	req := SignatureRequest{
		Algorithm:  AlgorithmEd25519,
		Message:   message,
		PrivateKey: privateKey,
	}

	sigResp, err := Sign(req)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	decodedSig, err := base64.StdEncoding.DecodeString(sigResp.Signature)
	if err != nil {
		t.Fatalf("Failed to decode signature: %v", err)
	}

	if !ed25519.Verify(publicKey, message, decodedSig) {
		t.Error("Ed25519 signature should be valid")
	}
}

func TestSecureCompareEd25519Keys(t *testing.T) {
	_, publicKey1, _ := GenerateEd25519KeyPair()
	_, publicKey2, _ := GenerateEd25519KeyPair()

	if SecureCompareEd25519Keys(publicKey1, publicKey1) != true {
		t.Error("Same keys should be equal")
	}

	if SecureCompareEd25519Keys(publicKey1, publicKey2) != false {
		t.Error("Different keys should not be equal")
	}
}

func TestEd25519KeyPairMethods(t *testing.T) {
	privateKey, publicKey, _ := GenerateEd25519KeyPair()
	now := time.Now()

	keyPair := &Ed25519KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      "test-key-id",
		CreatedAt:  now,
		ExpiresAt:  now.Add(1 * time.Hour),
		Version:    1,
	}

	if !keyPair.IsValid() {
		t.Error("Key pair should be valid")
	}

	keyPair.ExpiresAt = now.Add(-1 * time.Hour)
	if keyPair.IsExpired() != true {
		t.Error("Key pair should be expired")
	}

	keyPair.ExpiresAt = now.Add(1 * time.Hour)
	if keyPair.IsExpired() != false {
		t.Error("Key pair should not be expired")
	}

	if keyPair.ShouldRotate() != false {
		t.Error("Key should not need rotation yet")
	}
}

func TestEd25519KeyPairInvalid(t *testing.T) {
	keyPair := &Ed25519KeyPair{
		PrivateKey: nil,
		PublicKey:  nil,
	}

	if keyPair.IsValid() {
		t.Error("Key pair with nil keys should not be valid")
	}

	keyPair.PrivateKey = ed25519.PrivateKey([]byte("short"))
	if keyPair.IsValid() {
		t.Error("Key pair with wrong key size should not be valid")
	}
}

func TestGenerateSignatureWithTimestamp(t *testing.T) {
	secretKey := "test-secret-key"
	method := "POST"
	path := "/api/test"
	query := "param1=value1"
	timestamp := time.Now().Unix()
	nonce := "test-nonce"
	bodyHash := "abc123"

	signature, err := GenerateSignatureWithTimestamp(
		secretKey, method, path, query, timestamp, nonce, bodyHash,
		AlgorithmHMACSHA256,
	)

	if err != nil {
		t.Fatalf("Failed to generate signature: %v", err)
	}

	if signature == "" {
		t.Error("Signature should not be empty")
	}

	signature2, _ := GenerateSignatureWithTimestamp(
		secretKey, method, path, query, timestamp, nonce, bodyHash,
		AlgorithmHMACSHA256,
	)

	if signature != signature2 {
		t.Error("Same inputs should produce same signature")
	}

	differentSig, _ := GenerateSignatureWithTimestamp(
		secretKey, method, path, query, timestamp, nonce, "different",
		AlgorithmHMACSHA256,
	)

	if signature == differentSig {
		t.Error("Different inputs should produce different signature")
	}
}

func BenchmarkEd25519Sign(b *testing.B) {
	privateKey, _, _ := GenerateEd25519KeyPair()
	message := []byte("benchmark test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ed25519.Sign(privateKey, message)
	}
}

func BenchmarkEd25519Verify(b *testing.B) {
	privateKey, publicKey, _ := GenerateEd25519KeyPair()
	message := []byte("benchmark test message")
	signature := ed25519.Sign(privateKey, message)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ed25519.Verify(publicKey, message, signature)
	}
}

func BenchmarkEd25519ManagerSign(b *testing.B) {
	manager, _ := NewEd25519Manager(5 * time.Minute)
	message := []byte("benchmark test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Sign(message)
	}
}

func BenchmarkHMACSign(b *testing.B) {
	key := []byte("benchmark-secret-key")
	message := []byte("benchmark test message")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		ComputeHMAC(key, message, AlgoSHA256)
	}
}
