package service

import (
	"context"
	"testing"
	"time"
)

func TestAdvancedCryptoServiceV2_GenerateKey(t *testing.T) {
	config := QuantumResistantConfig{
		EnablePostQuantum:          true,
		EnableKeyEncapsulation:     true,
		EnableHashBasedSignatures: true,
	}

	hsmConfig := HSMConfig{
		Enabled:  false,
		Provider: "software",
	}

	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()

	t.Run("Generate AES256 Key", func(t *testing.T) {
		response, err := service.GenerateKeyV2(ctx, KeyTypeAES256GCM)
		if err != nil {
			t.Fatalf("Failed to generate AES key: %v", err)
		}

		if response.KeyID == "" {
			t.Error("Generated key ID is empty")
		}

		if response.KeyType != "aes-256-gcm" {
			t.Errorf("Expected key type 'aes-256-gcm', got '%s'", response.KeyType)
		}

		if !response.Success {
			t.Error("Key generation should succeed")
		}
	})

	t.Run("Generate ChaCha20 Key", func(t *testing.T) {
		response, err := service.GenerateKeyV2(ctx, KeyTypeChaCha20)
		if err != nil {
			t.Fatalf("Failed to generate ChaCha20 key: %v", err)
		}

		if response.KeyType != "chacha20-poly1305" {
			t.Errorf("Expected key type 'chacha20-poly1305', got '%s'", response.KeyType)
		}
	})

	t.Run("Generate Hybrid Key", func(t *testing.T) {
		response, err := service.GenerateKeyV2(ctx, KeyTypeHybrid)
		if err != nil {
			t.Fatalf("Failed to generate hybrid key: %v", err)
		}

		if response.KeyType != "hybrid-classic" {
			t.Errorf("Expected key type 'hybrid-classic', got '%s'", response.KeyType)
		}
	})
}

func TestAdvancedCryptoServiceV2_EncryptDecrypt(t *testing.T) {
	config := QuantumResistantConfig{
		EnablePostQuantum:          false,
		EnableKeyEncapsulation:     false,
		EnableHashBasedSignatures: false,
	}

	hsmConfig := HSMConfig{
		Enabled:  false,
		Provider: "software",
	}

	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	keyResponse, _ := service.GenerateKeyV2(ctx, KeyTypeAES256GCM)

	testCases := []struct {
		name      string
		plaintext string
	}{
		{"Simple Text", "Hello, World!"},
		{"Chinese Text", "你好，世界！"},
		{"Empty String", ""},
		{"Special Characters", "!@#$%^&*()_+-=[]{}|;':\",./<>?"},
		{"Long Text", "Lorem ipsum dolor sit amet, consectetur adipiscing elit. " +
			"Sed do eiusmod tempor incididunt ut labore et dolore magna aliqua."},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			encrypted, err := service.EncryptV2(ctx, tc.plaintext, keyResponse.KeyID)
			if err != nil {
				t.Fatalf("Failed to encrypt: %v", err)
			}

			if encrypted.Version != 3 {
				t.Errorf("Expected version 3, got %d", encrypted.Version)
			}

			if encrypted.Algorithm != "AES-256-GCM" {
				t.Errorf("Expected algorithm 'AES-256-GCM', got '%s'", encrypted.Algorithm)
			}

			decrypted, err := service.DecryptV2(ctx, encrypted)
			if err != nil {
				t.Fatalf("Failed to decrypt: %v", err)
			}

			if decrypted != tc.plaintext {
				t.Errorf("Expected '%s', got '%s'", tc.plaintext, decrypted)
			}
		})
	}
}

func TestAdvancedCryptoServiceV2_ChaCha20Encryption(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}

	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	keyResponse, _ := service.GenerateKeyV2(ctx, KeyTypeChaCha20)

	plaintext := "ChaCha20 Test Message"
	encrypted, err := service.EncryptV2(ctx, plaintext, keyResponse.KeyID)
	if err != nil {
		t.Fatalf("Failed to encrypt with ChaCha20: %v", err)
	}

	if encrypted.Algorithm != "ChaCha20-Poly1305" {
		t.Errorf("Expected algorithm 'ChaCha20-Poly1305', got '%s'", encrypted.Algorithm)
	}

	decrypted, err := service.DecryptV2(ctx, encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted != plaintext {
		t.Errorf("Expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestAdvancedCryptoServiceV2_QuantumResistantHash(t *testing.T) {
	config := QuantumResistantConfig{
		EnablePostQuantum: true,
	}

	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	testData := []byte("Test data for quantum resistant hashing")

	hash1 := service.GenerateQuantumResistantHashV2(testData)
	hash2 := service.GenerateQuantumResistantHashV2(testData)

	if hash1 != hash2 {
		t.Error("Same data should produce same hash")
	}

	hash3 := service.GenerateQuantumResistantHashV2([]byte("Different data"))
	if hash1 == hash3 {
		t.Error("Different data should produce different hash")
	}

	if len(hash1) == 0 {
		t.Error("Hash should not be empty")
	}
}

func TestAdvancedCryptoServiceV2_GetActiveKeys(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()

	initialKeys, _ := service.GetActiveKeysV2(ctx)
	initialCount := len(initialKeys)

	_, _ = service.GenerateKeyV2(ctx, KeyTypeAES256GCM)
	_, _ = service.GenerateKeyV2(ctx, KeyTypeChaCha20)

	activeKeys, err := service.GetActiveKeysV2(ctx)
	if err != nil {
		t.Fatalf("Failed to get active keys: %v", err)
	}

	if len(activeKeys) < initialCount+2 {
		t.Errorf("Expected at least %d active keys, got %d", initialCount+2, len(activeKeys))
	}
}

func TestAdvancedCryptoServiceV2_RevokeKey(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	keyResponse, _ := service.GenerateKeyV2(ctx, KeyTypeAES256GCM)

	err := service.RevokeKey(ctx, keyResponse.KeyID)
	if err != nil {
		t.Fatalf("Failed to revoke key: %v", err)
	}

	_, err = service.EncryptV2(ctx, "test", keyResponse.KeyID)
	if err == nil {
		t.Error("Should not be able to encrypt with revoked key")
	}
}

func TestAdvancedCryptoServiceV2_GetKeyStatistics(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	_, _ = service.GenerateKeyV2(ctx, KeyTypeAES256GCM)
	_, _ = service.GenerateKeyV2(ctx, KeyTypeChaCha20)

	stats := service.GetKeyStatistics(ctx)

	if stats.TotalKeys < 2 {
		t.Errorf("Expected at least 2 total keys, got %d", stats.TotalKeys)
	}

	if stats.ActiveKeys < 2 {
		t.Errorf("Expected at least 2 active keys, got %d", stats.ActiveKeys)
	}

	if stats.KeyTypes == nil {
		t.Error("Key types map should not be nil")
	}
}

func TestAdvancedCryptoServiceV2_HSMEncryption(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{
		Enabled:  true,
		Provider: "software",
	}

	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	keyResponse, _ := service.GenerateKeyV2(ctx, KeyTypeAES256GCM)

	payload, err := service.EncryptWithHSM(ctx, "HSM Test", keyResponse.KeyID)
	if err != nil {
		t.Fatalf("Failed to encrypt with HSM: %v", err)
	}

	if payload.HSMSignature == "" {
		t.Error("HSM signature should not be empty")
	}

	if !service.VerifyHSMSignature(payload) {
		t.Error("HSM signature verification failed")
	}
}

func TestAdvancedCryptoServiceV2_KeyCeremony(t *testing.T) {
	config := QuantumResistantConfig{
		EnableKeyEncapsulation: true,
	}
	hsmConfig := HSMConfig{
		Enabled: true,
	}

	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	result, err := service.PerformKeyCeremony(ctx)
	if err != nil {
		t.Fatalf("Key ceremony failed: %v", err)
	}

	if !result.Success {
		t.Error("Key ceremony should succeed")
	}

	if len(result.Steps) == 0 {
		t.Error("Key ceremony should have steps")
	}

	if result.MasterKeyHash == "" {
		t.Error("Master key hash should not be empty")
	}
}

func TestAdvancedCryptoServiceV2_WASMEncryption(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	plaintext := []byte("WASM encryption test")

	encrypted, err := service.GenerateWASMEncryptedData(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt with WASM: %v", err)
	}

	if len(encrypted) == 0 {
		t.Error("Encrypted data should not be empty")
	}

	decrypted, err := service.GenerateWASMDecryptedData(encrypted)
	if err != nil {
		t.Fatalf("Failed to decrypt with WASM: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Errorf("Expected '%s', got '%s'", string(plaintext), string(decrypted))
	}
}

func TestAdvancedCryptoServiceV2_MLKEMKeyEncapsulation(t *testing.T) {
	mlkem := NewMLKEMKeyEncapsulation()

	err := mlkem.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate ML-KEM key pair: %v", err)
	}

	if len(mlkem.PublicKey) == 0 {
		t.Error("Public key should not be empty")
	}

	if len(mlkem.PrivateKey) == 0 {
		t.Error("Private key should not be empty")
	}

	sharedSecret, ephemeralPub, err := mlkem.Encapsulate(mlkem.PublicKey)
	if err != nil {
		t.Fatalf("Failed to encapsulate: %v", err)
	}

	if len(sharedSecret) == 0 {
		t.Error("Shared secret should not be empty")
	}

	if len(ephemeralPub) == 0 {
		t.Error("Ephemeral public key should not be empty")
	}

	decryptedSecret, err := mlkem.Decapsulate(ephemeralPub)
	if err != nil {
		t.Fatalf("Failed to decapsulate: %v", err)
	}

	if string(sharedSecret) != string(decryptedSecret) {
		t.Error("Shared secrets should match")
	}
}

func TestAdvancedCryptoServiceV2_HashBasedSignature(t *testing.T) {
	hbs := NewHashBasedSignature()

	message := []byte("Test message for hash-based signature")
	signature, err := hbs.Sign(message)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if len(signature) == 0 {
		t.Error("Signature should not be empty")
	}

	if !hbs.Verify(message, signature) {
		t.Error("Signature verification should succeed")
	}

	wrongMessage := []byte("Wrong message")
	if hbs.Verify(wrongMessage, signature) {
		t.Error("Signature verification should fail for wrong message")
	}
}

func TestAdvancedCryptoServiceV2_InvalidKey(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()

	_, err := service.EncryptV2(ctx, "test", "invalid-key-id")
	if err == nil {
		t.Error("Should fail with invalid key ID")
	}
}

func TestAdvancedCryptoServiceV2_ConcurrentEncryption(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	ctx := context.Background()
	keyResponse, _ := service.GenerateKeyV2(ctx, KeyTypeAES256GCM)

	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func(id int) {
			plaintext := "Concurrent test"
			encrypted, err := service.EncryptV2(ctx, plaintext, keyResponse.KeyID)
			if err != nil {
				t.Errorf("Concurrent encryption %d failed: %v", id, err)
			}

			_, err = service.DecryptV2(ctx, encrypted)
			if err != nil {
				t.Errorf("Concurrent decryption %d failed: %v", id, err)
			}

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestAdvancedCryptoServiceV2_KeyRotation(t *testing.T) {
	config := QuantumResistantConfig{}
	hsmConfig := HSMConfig{}
	service := NewAdvancedCryptoServiceV2(config, hsmConfig)
	defer service.Close()

	time.Sleep(100 * time.Millisecond)

	ctx := context.Background()
	activeKeys1, _ := service.GetActiveKeysV2(ctx)
	activeCount1 := len(activeKeys1)

	service.rotateKeysV2()

	time.Sleep(100 * time.Millisecond)

	activeKeys2, _ := service.GetActiveKeysV2(ctx)
	activeCount2 := len(activeKeys2)

	if activeCount2 < activeCount1 {
		t.Errorf("Expected at least %d active keys after rotation, got %d", activeCount1, activeCount2)
	}
}
