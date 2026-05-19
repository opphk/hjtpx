package service

import (
	"context"
	"testing"
)

func TestQuantumSafeCryptoSystem(t *testing.T) {
	system := NewQuantumSafeCryptoSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	if !system.initialized {
		t.Error("System should be initialized")
	}
}

func TestKyberKeyEncapsulation(t *testing.T) {
	kyber := NewKyberKeyEncapsulation()
	ctx := context.Background()

	if err := kyber.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize Kyber: %v", err)
	}

	publicKey, privateKey, err := kyber.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if publicKey == nil || privateKey == nil {
		t.Fatal("Key pair should not be nil")
	}

	ciphertext, sharedSecret, err := kyber.Encapsulate(publicKey)
	if err != nil {
		t.Fatalf("Failed to encapsulate: %v", err)
	}

	if ciphertext == nil || sharedSecret == nil {
		t.Fatal("Ciphertext and shared secret should not be nil")
	}

	decryptedSecret, err := kyber.Decapsulate(privateKey, ciphertext)
	if err != nil {
		t.Fatalf("Failed to decapsulate: %v", err)
	}

	if decryptedSecret == nil {
		t.Error("Decrypted secret should not be nil")
	}
}

func TestDilithiumSignature(t *testing.T) {
	dilithium := NewDilithiumSignature()
	ctx := context.Background()

	if err := dilithium.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize Dilithium: %v", err)
	}

	publicKey, privateKey, err := dilithium.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if publicKey == nil || privateKey == nil {
		t.Fatal("Key pair should not be nil")
	}

	message := []byte("test message")
	signature, err := dilithium.Sign(privateKey, message)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if signature == nil {
		t.Fatal("Signature should not be nil")
	}

	valid, err := dilithium.Verify(publicKey, message, signature.Signature)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}

	if !valid {
		t.Error("Signature should be valid")
	}
}

func TestMcElieceCrypto(t *testing.T) {
	mceliece := NewMcElieceCrypto()
	ctx := context.Background()

	if err := mceliece.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize McEliece: %v", err)
	}

	publicKey, privateKey, err := mceliece.GenerateKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate key pair: %v", err)
	}

	if publicKey == nil || privateKey == nil {
		t.Fatal("Key pair should not be nil")
	}

	message := []byte("test message")
	ciphertext, err := mceliece.Encrypt(publicKey, message)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if ciphertext == nil {
		t.Fatal("Ciphertext should not be nil")
	}

	decrypted, err := mceliece.Decrypt(privateKey, ciphertext)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decrypted == nil {
		t.Error("Decrypted message should not be nil")
	}
}

func TestHybridCryptoEngine(t *testing.T) {
	engine := NewHybridCryptoEngine()
	ctx := context.Background()

	if err := engine.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize engine: %v", err)
	}

	plaintext := []byte("test message")
	quantumKey := make([]byte, 32)
	for i := range quantumKey {
		quantumKey[i] = byte(i)
	}

	result, err := engine.Encrypt(plaintext, quantumKey, "kyber")
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if result == nil {
		t.Fatal("Encryption result should not be nil")
	}

	if result.Ciphertext == nil {
		t.Error("Ciphertext should not be nil")
	}

	if !result.QuantumResistant {
		t.Error("Should be quantum resistant")
	}
}

func TestQuantumKeyDistribution(t *testing.T) {
	qkd := NewQuantumKeyDistribution()
	ctx := context.Background()

	if err := qkd.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize QKD: %v", err)
	}

	channel, err := qkd.SetupChannel("alice", "bob", 1000)
	if err != nil {
		t.Fatalf("Failed to setup channel: %v", err)
	}

	if channel == nil {
		t.Fatal("Channel should not be nil")
	}

	result, err := qkd.RunBB84Protocol(channel.ID)
	if err != nil {
		t.Fatalf("Failed to run BB84 protocol: %v", err)
	}

	if result == nil {
		t.Fatal("BB84 result should not be nil")
	}

	if result.FinalKey == nil {
		t.Error("Final key should not be nil")
	}

	if result.SecurityLevel < 0 || result.SecurityLevel > 1 {
		t.Errorf("Security level should be between 0 and 1, got %f", result.SecurityLevel)
	}
}

func TestQuantumEncryptionDecryption(t *testing.T) {
	system := NewQuantumSafeCryptoSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	plaintext := "test message"

	encryptResult, err := system.EncryptQuantumSafe(ctx, plaintext, "kyber")
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if encryptResult == nil {
		t.Fatal("Encryption result should not be nil")
	}

	decryptResult, err := system.DecryptQuantumSafe(ctx, encryptResult.Result.Ciphertext, encryptResult.Result.IV, "kyber")
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if decryptResult == nil {
		t.Fatal("Decryption result should not be nil")
	}
}

func TestQuantumSigning(t *testing.T) {
	system := NewQuantumSafeCryptoSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	message := "test message"

	result, err := system.SignQuantumSafe(ctx, message, "dilithium")
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}

	if result == nil {
		t.Fatal("Signing result should not be nil")
	}

	if result.Signature == nil {
		t.Error("Signature should not be nil")
	}

	if !result.Valid {
		t.Error("Signature should be valid")
	}
}

func TestHybridSignature(t *testing.T) {
	system := NewQuantumSafeCryptoSystem()
	ctx := context.Background()

	if err := system.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize system: %v", err)
	}

	message := "test message"

	result, err := system.GenerateHybridSignature(ctx, message)
	if err != nil {
		t.Fatalf("Failed to generate hybrid signature: %v", err)
	}

	if result == nil {
		t.Fatal("Signature result should not be nil")
	}

	if result.Signature == nil {
		t.Error("Signature should not be nil")
	}

	if !result.QuantumSafe {
		t.Error("Signature should be quantum safe")
	}
}
