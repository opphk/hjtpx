package service

import (
	"encoding/base64"
	"testing"
	"time"
)

func TestWASMCryptoEngineCreation(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")
	if engine == nil {
		t.Fatal("NewWASMCryptoEngine returned nil")
	}
	if engine.keyManager == nil {
		t.Fatal("keyManager is nil")
	}
	if engine.integrity == nil {
		t.Fatal("integrity is nil")
	}
	if engine.stats == nil {
		t.Fatal("stats is nil")
	}
}

func TestWASMCryptoEngineEncryptDecrypt(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintext := []byte("Hello, World!")
	ciphertext, err := engine.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	if len(ciphertext) <= len(plaintext) {
		t.Error("Ciphertext should be longer than plaintext")
	}

	decrypted, err := engine.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestWASMCryptoEngineEncryptString(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintext := "Hello, World!"
	ciphertext, err := engine.EncryptString(plaintext)
	if err != nil {
		t.Fatalf("EncryptString failed: %v", err)
	}

	decrypted, err := engine.DecryptString(ciphertext)
	if err != nil {
		t.Fatalf("DecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Decrypted string doesn't match: got %s, want %s", decrypted, plaintext)
	}
}

func TestWASMCryptoEngineEmptyPlaintext(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.Encrypt([]byte{})
	if err == nil {
		t.Error("Expected error for empty plaintext")
	}
}

func TestWASMCryptoEngineEmptyCiphertext(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.Decrypt([]byte{})
	if err == nil {
		t.Error("Expected error for empty ciphertext")
	}
}

func TestWASMCryptoEngineEncryptWithKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}

	plaintext := []byte("Secret message")
	ciphertext, err := engine.EncryptWithKey(plaintext, key)
	if err != nil {
		t.Fatalf("EncryptWithKey failed: %v", err)
	}

	decrypted, err := engine.DecryptWithKey(ciphertext, key)
	if err != nil {
		t.Fatalf("DecryptWithKey failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestWASMCryptoEngineEncryptWithInvalidKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.EncryptWithKey([]byte("test"), []byte("short"))
	if err == nil {
		t.Error("Expected error for invalid key length")
	}
}

func TestWASMCryptoEngineHash(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	data := []byte("Hello, World!")
	hash := engine.Hash(data)

	if hash == "" {
		t.Error("Hash should not be empty")
	}

	hash2 := engine.Hash(data)
	if hash != hash2 {
		t.Error("Same data should produce same hash")
	}

	hash3 := engine.Hash([]byte("Different"))
	if hash == hash3 {
		t.Error("Different data should produce different hash")
	}
}

func TestWASMCryptoEngineDeriveKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	password := "user-password"
	salt := make([]byte, 16)
	for i := range salt {
		salt[i] = byte(i)
	}

	key, err := engine.DeriveKey(password, salt)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	key2, _ := engine.DeriveKey(password, salt)
	if string(key) != string(key2) {
		t.Error("Same password and salt should produce same key")
	}
}

func TestWASMCryptoEngineDeriveKeyWithShortSalt(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.DeriveKey("password", []byte("short"))
	if err == nil {
		t.Error("Expected error for short salt")
	}
}

func TestWASMCryptoEngineGenerateKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	key, err := engine.GenerateKey()
	if err != nil {
		t.Fatalf("GenerateKey failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	key2, _ := engine.GenerateKey()
	if string(key) == string(key2) {
		t.Error("Two generated keys should be different")
	}
}

func TestWASMCryptoEngineGenerateSessionKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	sessionKey, err := engine.GenerateSessionKey()
	if err != nil {
		t.Fatalf("GenerateSessionKey failed: %v", err)
	}

	if sessionKey == "" {
		t.Error("Session key should not be empty")
	}
}

func TestWASMCryptoEngineEncryptWithSessionKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintext := []byte("Session message")

	encrypted1, err := engine.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	decrypted, err := engine.Decrypt(encrypted1)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestWASMCryptoEngineRevokeSessionKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.RevokeSessionKey("invalid-format")
	if err == nil {
		t.Error("Expected error for invalid session key format")
	}
}

func TestWASMCryptoEngineGenerateNonce(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	nonce, err := engine.GenerateNonce(16)
	if err != nil {
		t.Fatalf("GenerateNonce failed: %v", err)
	}

	if len(nonce) != 16 {
		t.Errorf("Expected nonce length 16, got %d", len(nonce))
	}

	nonce2, _ := engine.GenerateNonce(16)
	if string(nonce) == string(nonce2) {
		t.Error("Two generated nonces should be different")
	}
}

func TestWASMCryptoEngineGenerateNonceInvalidSize(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.GenerateNonce(8)
	if err == nil {
		t.Error("Expected error for nonce size < 12")
	}

	_, err = engine.GenerateNonce(300)
	if err == nil {
		t.Error("Expected error for nonce size > 256")
	}
}

func TestWASMCryptoEngineBatchEncrypt(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintexts := [][]byte{
		[]byte("Message 1"),
		[]byte("Message 2"),
		[]byte("Message 3"),
	}

	ciphertexts, err := engine.BatchEncrypt(plaintexts)
	if err != nil {
		t.Fatalf("BatchEncrypt failed: %v", err)
	}

	if len(ciphertexts) != len(plaintexts) {
		t.Errorf("Expected %d ciphertexts, got %d", len(plaintexts), len(ciphertexts))
	}
}

func TestWASMCryptoEngineBatchDecrypt(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintexts := [][]byte{
		[]byte("Message 1"),
		[]byte("Message 2"),
		[]byte("Message 3"),
	}

	ciphertexts, _ := engine.BatchEncrypt(plaintexts)
	decrypted, err := engine.BatchDecrypt(ciphertexts)
	if err != nil {
		t.Fatalf("BatchDecrypt failed: %v", err)
	}

	if len(decrypted) != len(plaintexts) {
		t.Errorf("Expected %d decrypted texts, got %d", len(plaintexts), len(decrypted))
	}

	for i, p := range plaintexts {
		if string(decrypted[i]) != string(p) {
			t.Errorf("Decrypted text %d doesn't match: got %s, want %s", i, string(decrypted[i]), string(p))
		}
	}
}

func TestWASMCryptoEngineGenerateChecksum(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	data := []byte("Test data")
	checksum := engine.GenerateChecksum(data)

	if checksum == "" {
		t.Error("Checksum should not be empty")
	}

	if len(checksum) != 64 {
		t.Errorf("Expected checksum length 64 (hex encoded), got %d", len(checksum))
	}

	checksum2 := engine.GenerateChecksum(data)
	if checksum != checksum2 {
		t.Error("Same data should produce same checksum")
	}
}

func TestWASMCryptoEngineVerifyChecksum(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	data := []byte("Test data")
	checksum := engine.GenerateChecksum(data)

	if !engine.VerifyChecksum(data, checksum) {
		t.Error("VerifyChecksum should return true for correct checksum")
	}

	if engine.VerifyChecksum([]byte("Different"), checksum) {
		t.Error("VerifyChecksum should return false for incorrect checksum")
	}
}

func TestWASMCryptoEngineCreateVerifyIntegrityToken(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	data := []byte("Test data")
	tokenID := engine.CreateIntegrityToken(data, time.Hour)

	if tokenID == "" {
		t.Error("Token ID should not be empty")
	}

	if !engine.VerifyIntegrityToken(tokenID, data) {
		t.Error("VerifyIntegrityToken should return true for valid token and data")
	}

	if engine.VerifyIntegrityToken(tokenID, []byte("Different")) {
		t.Error("VerifyIntegrityToken should return false for different data")
	}
}

func TestWASMCryptoEngineEncryptWithIntegrityCheck(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintext := []byte("Important data")
	ciphertext, tokenID, err := engine.EncryptWithIntegrityCheck(plaintext, time.Hour)
	if err != nil {
		t.Fatalf("EncryptWithIntegrityCheck failed: %v", err)
	}

	decrypted, err := engine.DecryptWithIntegrityCheck(ciphertext, tokenID)
	if err != nil {
		t.Fatalf("DecryptWithIntegrityCheck failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got %s, want %s", string(decrypted), string(plaintext))
	}
}

func TestWASMCryptoEngineDecryptWithIntegrityCheckFailure(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintext := []byte("Important data")
	ciphertext, tokenID, err := engine.EncryptWithIntegrityCheck(plaintext, time.Hour)
	if err != nil {
		t.Fatalf("EncryptWithIntegrityCheck failed: %v", err)
	}

	decrypted, err := engine.DecryptWithIntegrityCheck(ciphertext, "invalid-token")
	if err == nil {
		t.Error("Expected error for invalid token")
		_ = decrypted
	}
}

func TestWASMCryptoEngineEncryptDecryptFile(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	data := []byte("This is a large file content that needs to be encrypted in chunks")
	ciphertexts, merkleRoot, err := engine.EncryptFile(data, 256)
	if err != nil {
		t.Fatalf("EncryptFile failed: %v", err)
	}

	if len(ciphertexts) == 0 {
		t.Error("Should have at least one ciphertext chunk")
	}

	if merkleRoot == "" {
		t.Error("Merkle root should not be empty")
	}

	decrypted, err := engine.DecryptFile(ciphertexts, merkleRoot)
	if err != nil {
		t.Fatalf("DecryptFile failed: %v", err)
	}

	if string(decrypted) != string(data) {
		t.Errorf("Decrypted data doesn't match: got %s, want %s", string(decrypted), string(data))
	}
}

func TestWASMCryptoEngineEncryptDecryptFileInvalidChunkSize(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, _, err := engine.EncryptFile([]byte("data"), 128)
	if err == nil {
		t.Error("Expected error for chunk size < 256")
	}
}

func TestWASMCryptoEngineDecryptFileInvalidRoot(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	ciphertexts := [][]byte{[]byte("some data")}

	_, err := engine.DecryptFile(ciphertexts, "invalid-root")
	if err == nil {
		t.Error("Expected error for invalid merkle root")
	}
}

func TestWASMCryptoEngineDecryptFileEmptyChunks(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_, err := engine.DecryptFile([][]byte{}, "any-root")
	if err == nil {
		t.Error("Expected error for empty chunks")
	}
}

func TestWASMCryptoEngineGetPerformanceMetrics(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	metrics := engine.GetPerformanceMetrics()
	if metrics == nil {
		t.Fatal("GetPerformanceMetrics returned nil")
	}

	if metrics["algorithm"] != "AES-256-GCM" {
		t.Errorf("Expected algorithm AES-256-GCM, got %v", metrics["algorithm"])
	}

	if metrics["version"] != "3.0.0" {
		t.Errorf("Expected version 3.0.0, got %v", metrics["version"])
	}

	features, ok := metrics["features"].(map[string]bool)
	if !ok {
		t.Fatal("features should be a map")
	}

	if !features["session_keys"] {
		t.Error("session_keys feature should be enabled")
	}
	if !features["integrity_check"] {
		t.Error("integrity_check feature should be enabled")
	}
}

func TestWASMCryptoEngineGetStats(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_ = engine.Encrypt([]byte("test"))

	stats := engine.GetStats()
	if stats == nil {
		t.Fatal("GetStats returned nil")
	}

	if stats["encrypt_count"] == nil {
		t.Error("encrypt_count should be present")
	}
}

func TestWASMCryptoEngineResetStats(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	_ = engine.Encrypt([]byte("test"))

	engine.ResetStats()

	stats := engine.GetStats()
	if stats["encrypt_count"].(uint64) != 0 {
		t.Error("encrypt_count should be 0 after reset")
	}
}

func TestWASMCryptoEngineGenerateRandomID(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	id1 := engine.GenerateRandomID()
	id2 := engine.GenerateRandomID()

	if id1 == "" || id2 == "" {
		t.Error("IDs should not be empty")
	}

	if id1 == id2 {
		t.Error("Two generated IDs should be different")
	}

	if len(id1) != 32 {
		t.Errorf("Expected ID length 32 (hex encoded), got %d", len(id1))
	}
}

func TestKeyManagerGetOrCreateKey(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	key1, _ := engine.DeriveKeyWithID("user1", "password", []byte("salt-salt-salt-"))
	key2, _ := engine.DeriveKeyWithID("user1", "password", []byte("salt-salt-salt-"))

	if string(key1) != string(key2) {
		t.Error("Same ID should return same cached key")
	}

	key3, _ := engine.DeriveKeyWithID("user2", "password", []byte("salt-salt-salt-"))
	if string(key1) == string(key3) {
		t.Error("Different IDs should return different keys")
	}
}

func BenchmarkWASMCryptoEngineEncrypt(b *testing.B) {
	engine := NewWASMCryptoEngine("test-secret-key")
	plaintext := make([]byte, 1024)
	for i := range plaintext {
		plaintext[i] = byte(i)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Encrypt(plaintext)
	}
}

func BenchmarkWASMCryptoEngineDecrypt(b *testing.B) {
	engine := NewWASMCryptoEngine("test-secret-key")
	plaintext := make([]byte, 1024)
	ciphertext, _ := engine.Encrypt(plaintext)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.Decrypt(ciphertext)
	}
}

func BenchmarkWASMCryptoEngineBatchEncrypt(b *testing.B) {
	engine := NewWASMCryptoEngine("test-secret-key")
	plaintexts := make([][]byte, 100)
	for i := range plaintexts {
		plaintexts[i] = make([]byte, 64)
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = engine.BatchEncrypt(plaintexts)
	}
}

func TestWASMCryptoEngineB64Encoding(t *testing.T) {
	engine := NewWASMCryptoEngine("test-secret-key")

	plaintext := []byte("Hello, World!")
	ciphertext, err := engine.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}

	b64 := base64.StdEncoding.EncodeToString(ciphertext)

	b64Decoded, err := base64.StdEncoding.DecodeString(b64)
	if err != nil {
		t.Fatalf("Base64 decode failed: %v", err)
	}

	decrypted, err := engine.Decrypt(b64Decoded)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text doesn't match: got %s, want %s", string(decrypted), string(plaintext))
	}
}
