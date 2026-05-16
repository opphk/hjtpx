package crypto

import (
	"encoding/base64"
	"strings"
	"testing"
	"time"
)

func TestGenerateRandomKey(t *testing.T) {
	key, err := GenerateRandomKey(KeySize256)
	if err != nil {
		t.Fatalf("GenerateRandomKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected key length 32, got %d", len(key))
	}

	key2, err := GenerateRandomKey(KeySize128)
	if err != nil {
		t.Fatalf("GenerateRandomKey failed: %v", err)
	}
	if len(key2) != 16 {
		t.Fatalf("expected key length 16, got %d", len(key2))
	}

	key3, err := GenerateRandomKey(KeySize192)
	if err != nil {
		t.Fatalf("GenerateRandomKey failed: %v", err)
	}
	if len(key3) != 24 {
		t.Fatalf("expected key length 24, got %d", len(key3))
	}
}

func TestGenerateRandomString(t *testing.T) {
	s, err := GenerateRandomString(20)
	if err != nil {
		t.Fatalf("GenerateRandomString failed: %v", err)
	}
	if len(s) != 20 {
		t.Fatalf("expected length 20, got %d", len(s))
	}

	s2, err := GenerateRandomString(10)
	if err != nil {
		t.Fatalf("GenerateRandomString failed: %v", err)
	}
	if len(s2) != 10 {
		t.Fatalf("expected length 10, got %d", len(s2))
	}
	if s == s2 {
		t.Fatal("two random strings should not be equal")
	}
}

func TestGenerateSalt(t *testing.T) {
	salt, err := GenerateSalt(16)
	if err != nil {
		t.Fatalf("GenerateSalt failed: %v", err)
	}
	if len(salt) != 16 {
		t.Fatalf("expected salt length 16, got %d", len(salt))
	}

	defaultSalt, err := GenerateSalt(0)
	if err != nil {
		t.Fatalf("GenerateSalt with default failed: %v", err)
	}
	if len(defaultSalt) != 32 {
		t.Fatalf("expected default salt length 32, got %d", len(defaultSalt))
	}
}

func TestAESEncryptDecrypt(t *testing.T) {
	key, _ := GenerateRandomKey(KeySize256)
	plaintext := []byte("Hello, World! This is a test message.")

	ciphertext, err := AESEncrypt(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncrypt failed: %v", err)
	}

	decrypted, err := AESDecrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("AESDecrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %s, expected %s", decrypted, plaintext)
	}
}

func TestAESEncryptDecryptString(t *testing.T) {
	key, _ := GenerateRandomKey(KeySize256)
	plaintext := "Sensitive data that needs encryption"

	encrypted, err := AESEncryptString(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncryptString failed: %v", err)
	}

	decrypted, err := AESDecryptString(encrypted, key)
	if err != nil {
		t.Fatalf("AESDecryptString failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted text mismatch: got %s, expected %s", decrypted, plaintext)
	}
}

func TestAESEncryptWithAAD(t *testing.T) {
	key, _ := GenerateRandomKey(KeySize256)
	plaintext := []byte("Data with AAD")
	aad := []byte("additional authenticated data")

	ciphertext, err := AESEncryptWithAAD(plaintext, key, aad)
	if err != nil {
		t.Fatalf("AESEncryptWithAAD failed: %v", err)
	}

	decrypted, err := AESDecryptWithAAD(ciphertext, key, aad)
	if err != nil {
		t.Fatalf("AESDecryptWithAAD failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %s, expected %s", decrypted, plaintext)
	}
}

func TestAESEncryptWithAAD_WrongAAD(t *testing.T) {
	key, _ := GenerateRandomKey(KeySize256)
	plaintext := []byte("Data with AAD")

	ciphertext, err := AESEncryptWithAAD(plaintext, key, []byte("correct aad"))
	if err != nil {
		t.Fatalf("AESEncryptWithAAD failed: %v", err)
	}

	_, err = AESDecryptWithAAD(ciphertext, key, []byte("wrong aad"))
	if err == nil {
		t.Fatal("expected error when decrypting with wrong AAD")
	}
}

func TestAESEncrypt_InvalidKeyLength(t *testing.T) {
	_, err := AESEncrypt([]byte("data"), []byte("short"))
	if err != ErrInvalidKeyLength {
		t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestAESDecrypt_EmptyCiphertext(t *testing.T) {
	key, _ := GenerateRandomKey(KeySize256)
	_, err := AESDecrypt([]byte{}, key)
	if err != ErrCiphertextTooShort {
		t.Fatalf("expected ErrCiphertextTooShort, got %v", err)
	}
}

func TestAESEncryptWithCBC(t *testing.T) {
	key, _ := GenerateRandomKey(KeySize256)
	plaintext := []byte("CBC mode test data")

	ciphertext, err := AESEncryptWithCBC(plaintext, key)
	if err != nil {
		t.Fatalf("AESEncryptWithCBC failed: %v", err)
	}

	decrypted, err := AESDecryptWithCBC(ciphertext, key)
	if err != nil {
		t.Fatalf("AESDecryptWithCBC failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %s, expected %s", decrypted, plaintext)
	}
}

func TestHashSHA256(t *testing.T) {
	data := []byte("test data")
	hash := HashSHA256(data)
	if len(hash) != 64 {
		t.Fatalf("expected SHA256 hash length 64, got %d", len(hash))
	}

	hash2 := HashSHA256(data)
	if hash != hash2 {
		t.Fatal("SHA256 hash should be deterministic")
	}
}

func TestHashSHA512(t *testing.T) {
	data := []byte("test data")
	hash := HashSHA512(data)
	if len(hash) != 128 {
		t.Fatalf("expected SHA512 hash length 128, got %d", len(hash))
	}
}

func TestHashBytes(t *testing.T) {
	data := []byte("test data")
	h1 := HashBytes(data, AlgoSHA256)
	h2 := HashBytes(data, AlgoSHA512)
	if h1 == h2 {
		t.Fatal("different algorithms should produce different hashes")
	}
	if HashBytes(data, "invalid") != h1 {
		t.Fatal("default should be SHA256")
	}
}

func TestComputeHMAC(t *testing.T) {
	key := []byte("secret-key")
	data := []byte("message")

	mac, err := ComputeHMAC(key, data, AlgoSHA256)
	if err != nil {
		t.Fatalf("ComputeHMAC failed: %v", err)
	}
	if len(mac) == 0 {
		t.Fatal("HMAC should not be empty")
	}

	mac2, err := ComputeHMAC(key, data, AlgoSHA256)
	if err != nil {
		t.Fatalf("ComputeHMAC failed: %v", err)
	}
	if string(mac) != string(mac2) {
		t.Fatal("HMAC should be deterministic")
	}

	mac512, err := ComputeHMAC(key, data, AlgoSHA512)
	if err != nil {
		t.Fatalf("ComputeHMAC SHA512 failed: %v", err)
	}
	if string(mac) == string(mac512) {
		t.Fatal("different algorithms should produce different HMACs")
	}
}

func TestVerifyHMAC(t *testing.T) {
	key := []byte("secret-key")
	data := []byte("message")

	mac, _ := ComputeHMAC(key, data, AlgoSHA256)
	if !VerifyHMAC(key, data, mac, AlgoSHA256) {
		t.Fatal("VerifyHMAC should return true for valid MAC")
	}

	if VerifyHMAC(key, []byte("wrong-message"), mac, AlgoSHA256) {
		t.Fatal("VerifyHMAC should return false for wrong message")
	}

	if VerifyHMAC([]byte("wrong-key"), data, mac, AlgoSHA256) {
		t.Fatal("VerifyHMAC should return false for wrong key")
	}
}

func TestHMACString(t *testing.T) {
	key := []byte("secret")
	data := []byte("data")
	result := HMACString(key, data, AlgoSHA256)
	if len(result) != 64 {
		t.Fatalf("expected hex HMAC length 64, got %d", len(result))
	}
}

func TestHashPassword(t *testing.T) {
	password := "mySecurePassword123!"

	hash, err := HashPassword(password)
	if err != nil {
		t.Fatalf("HashPassword failed: %v", err)
	}
	if hash == "" {
		t.Fatal("hash should not be empty")
	}

	if !VerifyPassword(password, hash) {
		t.Fatal("VerifyPassword should return true for correct password")
	}

	if VerifyPassword("wrongPassword", hash) {
		t.Fatal("VerifyPassword should return false for wrong password")
	}
}

func TestHashPasswordWithCost(t *testing.T) {
	password := "testPassword123"

	hash, err := HashPasswordWithCost(password, 4)
	if err != nil {
		t.Fatalf("HashPasswordWithCost failed: %v", err)
	}

	if !VerifyPasswordWithCost(password, hash, 4) {
		t.Fatal("VerifyPasswordWithCost should return true for correct cost")
	}

	if VerifyPasswordWithCost("wrong", hash, 4) {
		t.Fatal("VerifyPasswordWithCost should return false for wrong password")
	}
}

func TestPBKDF2Hash(t *testing.T) {
	password := []byte("password")
	salt := []byte("somesalt")

	hash, err := PBKDF2Hash(password, salt, 10000, 32)
	if err != nil {
		t.Fatalf("PBKDF2Hash failed: %v", err)
	}
	if len(hash) != 32 {
		t.Fatalf("expected hash length 32, got %d", len(hash))
	}

	hash2, err := PBKDF2Hash(password, salt, 10000, 32)
	if err != nil {
		t.Fatalf("PBKDF2Hash failed: %v", err)
	}
	if string(hash) != string(hash2) {
		t.Fatal("PBKDF2 hash should be deterministic")
	}
}

func TestGenerateRSAKeyPair(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair failed: %v", err)
	}
	if priv == nil {
		t.Fatal("private key should not be nil")
	}
	if pub == nil {
		t.Fatal("public key should not be nil")
	}
}

func TestRSAEncryptDecrypt(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair failed: %v", err)
	}

	plaintext := []byte("RSA encryption test")

	ciphertext, err := RSAEncrypt(plaintext, pub)
	if err != nil {
		t.Fatalf("RSAEncrypt failed: %v", err)
	}

	decrypted, err := RSADecrypt(ciphertext, priv)
	if err != nil {
		t.Fatalf("RSADecrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Fatalf("decrypted text mismatch: got %s, expected %s", decrypted, plaintext)
	}
}

func TestRSASignVerify(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair failed: %v", err)
	}

	message := []byte("message to sign")

	signature, err := RSASign(message, priv)
	if err != nil {
		t.Fatalf("RSASign failed: %v", err)
	}

	err = RSAVerify(message, signature, pub)
	if err != nil {
		t.Fatalf("RSAVerify failed: %v", err)
	}

	err = RSAVerify([]byte("tampered message"), signature, pub)
	if err == nil {
		t.Fatal("RSAVerify should fail for tampered message")
	}
}

func TestExportParseRSAKeys(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair(2048)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair failed: %v", err)
	}

	privPEM, err := ExportRSAPrivateKeyToPEM(priv)
	if err != nil {
		t.Fatalf("ExportRSAPrivateKeyToPEM failed: %v", err)
	}

	pubPEM, err := ExportRSAPublicKeyToPEM(pub)
	if err != nil {
		t.Fatalf("ExportRSAPublicKeyToPEM failed: %v", err)
	}

	parsedPriv, err := ParseRSAPrivateKeyFromPEM(privPEM)
	if err != nil {
		t.Fatalf("ParseRSAPrivateKeyFromPEM failed: %v", err)
	}

	parsedPub, err := ParseRSAPublicKeyFromPEM(pubPEM)
	if err != nil {
		t.Fatalf("ParseRSAPublicKeyFromPEM failed: %v", err)
	}

	if parsedPriv == nil || parsedPub == nil {
		t.Fatal("parsed keys should not be nil")
	}
}

func TestGenerateECDSAKeyPair(t *testing.T) {
	priv, pub, err := GenerateECDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPair failed: %v", err)
	}
	if priv == nil {
		t.Fatal("private key should not be nil")
	}
	if pub == nil {
		t.Fatal("public key should not be nil")
	}
}

func TestECDSASignVerify(t *testing.T) {
	priv, pub, err := GenerateECDSAKeyPair()
	if err != nil {
		t.Fatalf("GenerateECDSAKeyPair failed: %v", err)
	}

	message := []byte("ECDSA test message")

	signature, err := ECDSASign(message, priv)
	if err != nil {
		t.Fatalf("ECDSASign failed: %v", err)
	}

	if !ECDSAVerify(message, signature, pub) {
		t.Fatal("ECDSAVerify should return true for valid signature")
	}

	if ECDSAVerify([]byte("wrong message"), signature, pub) {
		t.Fatal("ECDSAVerify should return false for wrong message")
	}
}

func TestConstantTimeCompare(t *testing.T) {
	if !ConstantTimeCompare("hello", "hello") {
		t.Fatal("ConstantTimeCompare should return true for equal strings")
	}
	if ConstantTimeCompare("hello", "world") {
		t.Fatal("ConstantTimeCompare should return false for different strings")
	}
}

func TestConstantTimeCompareBytes(t *testing.T) {
	if !ConstantTimeCompareBytes([]byte("hello"), []byte("hello")) {
		t.Fatal("ConstantTimeCompareBytes should return true for equal bytes")
	}
	if ConstantTimeCompareBytes([]byte("hello"), []byte("world")) {
		t.Fatal("ConstantTimeCompareBytes should return false for different bytes")
	}
}

func TestDeriveKey(t *testing.T) {
	key, err := DeriveKey([]byte("password"), []byte("salt"), 32, 10000)
	if err != nil {
		t.Fatalf("DeriveKey failed: %v", err)
	}
	if len(key) != 32 {
		t.Fatalf("expected key length 32, got %d", len(key))
	}
}

func TestGenerateSecureToken(t *testing.T) {
	token, err := GenerateSecureToken(32)
	if err != nil {
		t.Fatalf("GenerateSecureToken failed: %v", err)
	}
	if token == "" {
		t.Fatal("token should not be empty")
	}

	decoded, err := base64.URLEncoding.DecodeString(token)
	if err != nil {
		t.Fatalf("token should be valid base64 URL encoding: %v", err)
	}
	if len(decoded) != 32 {
		t.Fatalf("expected decoded token length 32, got %d", len(decoded))
	}
}

func TestMaskSensitiveData(t *testing.T) {
	tests := []struct {
		input        string
		visibleStart int
		visibleEnd   int
		expected     string
	}{
		{"1234567890", 2, 2, "12******90"},
		{"abc", 1, 1, "a*c"},
		{"short", 10, 10, "*****"},
		{"test@email.com", 0, 0, "**************"},
		{"hello", 0, 3, "**llo"},
		{"hello", 2, 0, "he***"},
	}

	for _, tt := range tests {
		result := MaskSensitiveData(tt.input, tt.visibleStart, tt.visibleEnd)
		if result != tt.expected {
			t.Errorf("MaskSensitiveData(%q, %d, %d) = %q, want %q",
				tt.input, tt.visibleStart, tt.visibleEnd, result, tt.expected)
		}
	}
}

func TestEncryptSensitive(t *testing.T) {
	plaintext := "sensitive-info"

	result, err := EncryptSensitive(plaintext)
	if err != nil {
		t.Fatalf("EncryptSensitive failed: %v", err)
	}

	if result.Ciphertext == plaintext {
		t.Fatal("ciphertext should not equal plaintext")
	}
	if result.Algorithm != "AES-256-GCM" {
		t.Fatalf("expected algorithm AES-256-GCM, got %s", result.Algorithm)
	}
	if result.Timestamp == 0 {
		t.Fatal("timestamp should not be zero")
	}
}

func TestDecryptSensitive(t *testing.T) {
	plaintext := "sensitive-data"

	result, err := EncryptSensitive(plaintext)
	if err != nil {
		t.Fatalf("EncryptSensitive failed: %v", err)
	}

	decrypted, err := DecryptSensitive(result.Ciphertext, result.Key)
	if err != nil {
		t.Fatalf("DecryptSensitive failed: %v", err)
	}

	if decrypted != plaintext {
		t.Fatalf("decrypted text mismatch: got %s, expected %s", decrypted, plaintext)
	}
}

func TestGenerateAPIKey(t *testing.T) {
	apiKey, err := GenerateAPIKey("sk")
	if err != nil {
		t.Fatalf("GenerateAPIKey failed: %v", err)
	}

	if !strings.HasPrefix(apiKey, "sk_") {
		t.Fatalf("expected prefix 'sk_', got %s", apiKey)
	}

	if !ValidateAPIKey(apiKey, "sk") {
		t.Fatal("ValidateAPIKey should return true for valid key")
	}

	if ValidateAPIKey(apiKey, "pk") {
		t.Fatal("ValidateAPIKey should return false for wrong prefix")
	}
}

func TestNewEncryptedData(t *testing.T) {
	data := NewEncryptedData("cipher", "iv", "auth")
	if data.Algorithm != "AES-256-GCM" {
		t.Fatalf("expected AES-256-GCM, got %s", data.Algorithm)
	}
	if data.Version != 1 {
		t.Fatalf("expected version 1, got %d", data.Version)
	}

	json := data.ToJSON()
	if !strings.Contains(json, `"a":"AES-256-GCM"`) {
		t.Fatal("JSON should contain algorithm field")
	}
}

func TestGetBcryptCost(t *testing.T) {
	hash, err := HashPasswordWithCost("password", 10)
	if err != nil {
		t.Fatalf("HashPasswordWithCost failed: %v", err)
	}

	cost, err := GetBcryptCost(hash)
	if err != nil {
		t.Fatalf("GetBcryptCost failed: %v", err)
	}
	if cost != 10 {
		t.Fatalf("expected cost 10, got %d", cost)
	}
}

func TestHMACWithDifferentAlgorithms(t *testing.T) {
	key := []byte("key")
	data := []byte("data")

	mac256, _ := ComputeHMAC(key, data, AlgoSHA256)
	mac512, _ := ComputeHMAC(key, data, AlgoSHA512)
	mac1, _ := ComputeHMAC(key, data, AlgoSHA1)

	if len(mac256) == 0 || len(mac512) == 0 || len(mac1) == 0 {
		t.Fatal("HMACs should not be empty")
	}

	if string(mac256) == string(mac512) {
		t.Fatal("SHA256 and SHA512 HMAC should differ")
	}
}

func TestAESKeySizeValidation(t *testing.T) {
	invalidKey := []byte("wrong-length-key!!")
	_, err := AESEncrypt([]byte("data"), invalidKey)
	if err != ErrInvalidKeyLength {
		t.Fatalf("expected ErrInvalidKeyLength, got %v", err)
	}
}

func TestRSAKeySizeBounds(t *testing.T) {
	priv, pub, err := GenerateRSAKeyPair(1024)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair with 1024 should default to 2048: %v", err)
	}
	if priv == nil || pub == nil {
		t.Fatal("keys should not be nil")
	}

	_, _, err = GenerateRSAKeyPair(8192)
	if err != nil {
		t.Fatalf("GenerateRSAKeyPair with 8192 should cap to 4096: %v", err)
	}
}

func TestSaltManager_NewSaltManager(t *testing.T) {
	sm := NewSaltManager(16)
	if sm == nil {
		t.Fatal("SaltManager should not be nil")
	}

	currentSalt := sm.GetCurrentSalt()
	if currentSalt == "" {
		t.Fatal("initial salt should not be empty")
	}

	if len(currentSalt) != 16 {
		t.Fatalf("expected salt length 16, got %d", len(currentSalt))
	}
}

func TestSaltManager_GetCurrentSalt(t *testing.T) {
	sm := NewSaltManager(16)

	salt1 := sm.GetCurrentSalt()
	salt2 := sm.GetCurrentSalt()

	if salt1 != salt2 {
		t.Fatal("GetCurrentSalt should return the same salt")
	}
}

func TestSaltManager_RotateSalt(t *testing.T) {
	sm := NewSaltManager(16)

	oldSalt := sm.GetCurrentSalt()

	err := sm.RotateSalt()
	if err != nil {
		t.Fatalf("RotateSalt failed: %v", err)
	}

	newSalt := sm.GetCurrentSalt()
	if oldSalt == newSalt {
		t.Fatal("salt should have changed after rotation")
	}

	if len(newSalt) != 16 {
		t.Fatalf("expected new salt length 16, got %d", len(newSalt))
	}
}

func TestSaltManager_ValidateWithHistoricalSalts(t *testing.T) {
	sm := NewSaltManager(16)

	initialSalt := sm.GetCurrentSalt()
	if !sm.ValidateWithHistoricalSalts(initialSalt) {
		t.Fatal("initial salt should be valid")
	}

	err := sm.RotateSalt()
	if err != nil {
		t.Fatalf("RotateSalt failed: %v", err)
	}

	newSalt := sm.GetCurrentSalt()
	if !sm.ValidateWithHistoricalSalts(newSalt) {
		t.Fatal("new salt should be valid after rotation")
	}

	if !sm.ValidateWithHistoricalSalts(initialSalt) {
		t.Fatal("old salt should still be valid in historical salts")
	}

	if sm.ValidateWithHistoricalSalts("invalid-salt") {
		t.Fatal("invalid salt should not be valid")
	}
}

func TestSaltManager_MultipleRotations(t *testing.T) {
	sm := NewSaltManager(16)

	salts := make(map[string]bool)
	initialSalt := sm.GetCurrentSalt()
	salts[initialSalt] = true

	for i := 0; i < 5; i++ {
		err := sm.RotateSalt()
		if err != nil {
			t.Fatalf("RotateSalt iteration %d failed: %v", i, err)
		}

		currentSalt := sm.GetCurrentSalt()
		if salts[currentSalt] {
			t.Fatalf("salt %d is not unique", i+1)
		}
		salts[currentSalt] = true
	}

	historicalCount := sm.GetHistoricalSaltsCount()
	if historicalCount < 6 {
		t.Fatalf("expected at least 6 historical salts, got %d", historicalCount)
	}
}

func TestSaltManager_ShouldRotate(t *testing.T) {
	sm := NewSaltManager(16)

	sm.SetRotationPeriod(1 * time.Second)

	if sm.ShouldRotate() {
		t.Fatal("should not need rotation immediately after creation")
	}

	time.Sleep(1100 * time.Millisecond)

	if !sm.ShouldRotate() {
		t.Fatal("should need rotation after rotation period")
	}
}

func TestSaltManager_GetSetRotationPeriod(t *testing.T) {
	sm := NewSaltManager(16)

	period := sm.GetRotationPeriod()
	if period != 5*time.Minute {
		t.Fatalf("expected default rotation period 5m, got %v", period)
	}

	sm.SetRotationPeriod(10 * time.Minute)
	period = sm.GetRotationPeriod()
	if period != 10*time.Minute {
		t.Fatalf("expected new rotation period 10m, got %v", period)
	}
}

func TestSaltManager_HistoricalSaltsLimit(t *testing.T) {
	sm := NewSaltManager(16)

	for i := 0; i < 15; i++ {
		sm.RotateSalt()
	}

	count := sm.GetHistoricalSaltsCount()
	if count != 10 {
		t.Fatalf("expected 10 historical salts (limit), got %d", count)
	}
}

func TestSaltManager_ConcurrentAccess(t *testing.T) {
	sm := NewSaltManager(16)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_ = sm.GetCurrentSalt()
				sm.ValidateWithHistoricalSalts("test-salt")
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestSaltManager_ConcurrentRotation(t *testing.T) {
	sm := NewSaltManager(16)

	done := make(chan bool, 5)

	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				sm.RotateSalt()
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}

	count := sm.GetHistoricalSaltsCount()
	if count < 1 {
		t.Fatal("should have at least one historical salt after rotations")
	}
}