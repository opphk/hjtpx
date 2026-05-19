package crypto

import (
	"testing"
	"time"
)

func TestChaCha20Poly1305EncryptDecrypt(t *testing.T) {
	key, err := GenerateRandomKey(KeySize256)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	plaintext := []byte("Hello, ChaCha20-Poly1305!")
	ciphertext, err := ChaCha20Poly1305Encrypt(plaintext, key)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := ChaCha20Poly1305Decrypt(ciphertext, key)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text mismatch: got %s, want %s", decrypted, plaintext)
	}
}

func TestChaCha20Poly1305WithAAD(t *testing.T) {
	key, err := GenerateRandomKey(KeySize256)
	if err != nil {
		t.Fatalf("Failed to generate key: %v", err)
	}

	plaintext := []byte("Secret message with AAD")
	aad := []byte("Additional authenticated data")

	ciphertext, err := ChaCha20Poly1305EncryptWithAAD(plaintext, key, aad)
	if err != nil {
		t.Fatalf("Encryption with AAD failed: %v", err)
	}

	decrypted, err := ChaCha20Poly1305DecryptWithAAD(ciphertext, key, aad)
	if err != nil {
		t.Fatalf("Decryption with AAD failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Decrypted text mismatch with AAD")
	}
}

func TestArgon2Hash(t *testing.T) {
	password := []byte("test-password-123")
	salt, err := GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	hash, err := Argon2Hash(password, salt, nil)
	if err != nil {
		t.Fatalf("Argon2 hash failed: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}
}

func TestArgon2HashString(t *testing.T) {
	password := "test-password-123"
	hash, err := Argon2HashString(password, nil)
	if err != nil {
		t.Fatalf("Argon2 hash string failed: %v", err)
	}

	if len(hash) == 0 {
		t.Error("Hash string is empty")
	}
}

func TestScryptDeriveKey(t *testing.T) {
	password := []byte("test-password")
	salt, err := GenerateRandomBytes(16)
	if err != nil {
		t.Fatalf("Failed to generate salt: %v", err)
	}

	key, err := ScryptDeriveKey(password, salt, nil)
	if err != nil {
		t.Fatalf("Scrypt derivation failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}

func TestBlake2bHash(t *testing.T) {
	data := []byte("test data for blake2b")
	
	hash, err := Blake2bHash(data, nil)
	if err != nil {
		t.Fatalf("BLAKE2b hash failed: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}
}

func TestBlake2bHashWithKey(t *testing.T) {
	data := []byte("test data with key")
	key, _ := GenerateRandomBytes(32)

	hash, err := Blake2bHash(data, key)
	if err != nil {
		t.Fatalf("BLAKE2b hash with key failed: %v", err)
	}

	if len(hash) != 32 {
		t.Errorf("Expected hash length 32, got %d", len(hash))
	}
}

func TestBlake2b512Hash(t *testing.T) {
	data := []byte("test data for blake2b-512")
	
	hash, err := Blake2b512Hash(data, nil)
	if err != nil {
		t.Fatalf("BLAKE2b-512 hash failed: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
}

func TestAESWrapUnwrap(t *testing.T) {
	wrappingKey, err := GenerateRandomKey(KeySize256)
	if err != nil {
		t.Fatalf("Failed to generate wrapping key: %v", err)
	}

	keyToWrap, err := GenerateRandomKey(KeySize256)
	if err != nil {
		t.Fatalf("Failed to generate key to wrap: %v", err)
	}

	wrapped, err := AESWrapKey(wrappingKey, keyToWrap)
	if err != nil {
		t.Fatalf("Key wrapping failed: %v", err)
	}

	unwrapped, err := AESUnwrapKey(wrappingKey, wrapped)
	if err != nil {
		t.Fatalf("Key unwrapping failed: %v", err)
	}

	if !ConstantTimeCompareBytes(keyToWrap, unwrapped) {
		t.Error("Unwrapped key does not match original")
	}
}

func TestHKDF(t *testing.T) {
	ikm := []byte("input key material")
	salt, _ := GenerateRandomBytes(32)
	info := []byte("hkdf test")

	key, err := HKDF(ikm, &HKDFParams{
		Hash:      AlgoSHA256,
		Salt:      salt,
		Info:      info,
		KeyLength: 32,
	})
	if err != nil {
		t.Fatalf("HKDF failed: %v", err)
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}

func TestEncryptThenMAC(t *testing.T) {
	encryptionKey, _ := GenerateRandomKey(KeySize256)
	macKey, _ := GenerateRandomKey(KeySize256)
	plaintext := []byte("test encrypt then mac")

	ciphertext, mac, err := EncryptThenMAC(plaintext, encryptionKey, macKey, "AES-GCM")
	if err != nil {
		t.Fatalf("Encrypt-then-MAC failed: %v", err)
	}

	decrypted, err := VerifyThenDecrypt(ciphertext, mac, encryptionKey, macKey, "AES-GCM")
	if err != nil {
		t.Fatalf("Verify-then-decrypt failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Decrypted text mismatch")
	}
}

func TestEncryptWithMultipleKeys(t *testing.T) {
	key1, _ := GenerateRandomKey(KeySize256)
	key2, _ := GenerateRandomKey(KeySize256)
	plaintext := []byte("multi-layer encryption test")

	ciphertext, err := EncryptWithMultipleKeys(plaintext, [][]byte{key1, key2}, []string{"AES-GCM", "ChaCha20-Poly1305"})
	if err != nil {
		t.Fatalf("Multi-key encryption failed: %v", err)
	}

	decrypted, err := DecryptWithMultipleKeys(ciphertext, [][]byte{key1, key2}, []string{"AES-GCM", "ChaCha20-Poly1305"})
	if err != nil {
		t.Fatalf("Multi-key decryption failed: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Multi-layer decryption failed")
	}
}

func TestKeyRotationManager(t *testing.T) {
	manager, err := NewKeyRotationManager(&KeyRotationConfig{
		RotationInterval:   1 * time.Minute,
		MaxKeyAge:          10 * time.Minute,
		AutoRotation:       false,
		BackupBeforeRotate: false,
	})
	if err != nil {
		t.Fatalf("Failed to create key rotation manager: %v", err)
	}

	key, err := manager.GetCurrentKey()
	if err != nil {
		t.Fatalf("Failed to get current key: %v", err)
	}

	if key.Metadata.Status != KeyStatusActive {
		t.Error("Expected active key status")
	}

	newKey, err := manager.RotateKey()
	if err != nil {
		t.Fatalf("Key rotation failed: %v", err)
	}

	if newKey.Metadata.Version != 2 {
		t.Errorf("Expected version 2, got %d", newKey.Metadata.Version)
	}

	stats := manager.GetStats()
	if stats.TotalKeyVersions != 2 {
		t.Errorf("Expected 2 key versions, got %d", stats.TotalKeyVersions)
	}
}

func TestKeyRotationManagerEncryption(t *testing.T) {
	manager, err := NewKeyRotationManager(&KeyRotationConfig{
		AutoRotation: false,
		MaxKeyAge:    100 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create key rotation manager: %v", err)
	}

	plaintext := "test encryption with rotation manager"
	ciphertext, err := manager.EncryptStringWithCurrentKey(plaintext)
	if err != nil {
		t.Fatalf("Encryption failed: %v", err)
	}

	decrypted, err := manager.DecryptStringWithKey(1, ciphertext)
	if err != nil {
		t.Fatalf("Decryption failed: %v", err)
	}

	if decrypted != plaintext {
		t.Error("Decrypted text mismatch")
	}
}

func TestKeyRotationWithAlgorithm(t *testing.T) {
	manager, err := NewKeyRotationManager(&KeyRotationConfig{
		AutoRotation: false,
		Algorithm:    AlgorithmChaCha20Poly1305,
		MaxKeyAge:    100 * time.Hour,
	})
	if err != nil {
		t.Fatalf("Failed to create key rotation manager: %v", err)
	}

	key, err := manager.GetCurrentKey()
	if err != nil {
		t.Fatalf("Failed to get current key: %v", err)
	}

	if key.Metadata.Algorithm != AlgorithmChaCha20Poly1305 {
		t.Errorf("Expected ChaCha20-Poly1305 algorithm")
	}
}

func TestArgon2Params(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("somesaltvalue")
	
	params := &Argon2Params{
		Memory:      32768,
		Iterations:  2,
		Parallelism: 2,
		SaltLength:  16,
		KeyLength:   64,
	}

	hash, err := Argon2Hash(password, salt, params)
	if err != nil {
		t.Fatalf("Argon2 with custom params failed: %v", err)
	}

	if len(hash) != 64 {
		t.Errorf("Expected hash length 64, got %d", len(hash))
	}
}

func TestScryptParams(t *testing.T) {
	password := []byte("test-password")
	salt := []byte("somesaltvalue")
	
	params := &ScryptParams{
		N:       8192,
		R:       8,
		P:       1,
		KeyLen:  64,
		SaltLen: 16,
	}

	key, err := ScryptDeriveKey(password, salt, params)
	if err != nil {
		t.Fatalf("Scrypt with custom params failed: %v", err)
	}

	if len(key) != 64 {
		t.Errorf("Expected key length 64, got %d", len(key))
	}
}