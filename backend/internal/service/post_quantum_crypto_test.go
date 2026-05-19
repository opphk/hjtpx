package service

import (
	"context"
	"testing"

	"github.com/hjtpx/hjtpx/pkg/crypto"
)

func TestNewPostQuantumCryptoService(t *testing.T) {
	service := NewPostQuantumCryptoService()
	
	if service == nil {
		t.Fatal("Service should not be nil")
	}
	
	if service.kyberLevel != crypto.Kyber768 {
		t.Errorf("Expected Kyber768, got %v", service.kyberLevel)
	}
	
	if service.dilithiumLevel != crypto.Dilithium3 {
		t.Errorf("Expected Dilithium3, got %v", service.dilithiumLevel)
	}
	
	if service.qkdSimulator == nil {
		t.Error("QKD simulator should not be nil")
	}
}

func TestSetSecurityLevels(t *testing.T) {
	service := NewPostQuantumCryptoService()
	
	service.SetSecurityLevels(crypto.Kyber1024, crypto.Dilithium5)
	
	if service.kyberLevel != crypto.Kyber1024 {
		t.Errorf("Expected Kyber1024, got %v", service.kyberLevel)
	}
	
	if service.dilithiumLevel != crypto.Dilithium5 {
		t.Errorf("Expected Dilithium5, got %v", service.dilithiumLevel)
	}
}

func TestGenerateKyberKeyPair(t *testing.T) {
	service := NewPostQuantumCryptoService()
	
	keyPair, err := service.GenerateKyberKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Kyber key pair: %v", err)
	}
	
	if keyPair == nil {
		t.Fatal("Key pair should not be nil")
	}
	
	if keyPair.PublicKey == nil {
		t.Error("Public key should not be nil")
	}
	
	if keyPair.PrivateKey == nil {
		t.Error("Private key should not be nil")
	}
	
	if keyPair.ID == "" {
		t.Error("Key pair ID should not be empty")
	}
	
	if !keyPair.IsActive {
		t.Error("Key pair should be active")
	}
}

func TestGenerateDilithiumKeyPair(t *testing.T) {
	service := NewPostQuantumCryptoService()
	
	keyPair, err := service.GenerateDilithiumKeyPair()
	if err != nil {
		t.Fatalf("Failed to generate Dilithium key pair: %v", err)
	}
	
	if keyPair == nil {
		t.Fatal("Key pair should not be nil")
	}
	
	if keyPair.PublicKey == nil {
		t.Error("Public key should not be nil")
	}
	
	if keyPair.PrivateKey == nil {
		t.Error("Private key should not be nil")
	}
	
	if keyPair.ID == "" {
		t.Error("Key pair ID should not be empty")
	}
}

func TestEncryptDecrypt(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	plaintext := "This is a test message for post-quantum encryption"
	
	req := &PQCEncryptionRequest{
		Plaintext: plaintext,
		UseHybrid: true,
	}
	
	encryptResp, err := service.Encrypt(ctx, req)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	
	if !encryptResp.Success {
		t.Error("Encryption should succeed")
	}
	
	if encryptResp.HybridResult == nil {
		t.Fatal("Hybrid result should not be nil")
	}
	
	if !encryptResp.HybridResult.QuantumResistant {
		t.Error("Should be quantum resistant")
	}
	
	decryptReq := &PQCDecryptionRequest{
		HybridResult: encryptResp.HybridResult,
		KeyPairID:    encryptResp.KeyPairID,
	}
	
	decryptResp, err := service.Decrypt(ctx, decryptReq)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}
	
	if !decryptResp.Success {
		t.Error("Decryption should succeed")
	}
}

func TestSignVerify(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	message := "This is a test message for post-quantum signing"
	
	signReq := &PQCSigningRequest{
		Message: message,
	}
	
	signResp, err := service.Sign(ctx, signReq)
	if err != nil {
		t.Fatalf("Failed to sign: %v", err)
	}
	
	if !signResp.Success {
		t.Error("Signing should succeed")
	}
	
	if signResp.Signature == nil {
		t.Fatal("Signature should not be nil")
	}
	
	if signResp.PublicKey == nil {
		t.Fatal("Public key should not be nil")
	}
	
	verifyReq := &PQCVerificationRequest{
		Message:   message,
		Signature: signResp.Signature,
		PublicKey: signResp.PublicKey,
	}
	
	verifyResp, err := service.Verify(ctx, verifyReq)
	if err != nil {
		t.Fatalf("Failed to verify: %v", err)
	}
	
	if !verifyResp.Success {
		t.Error("Verification should succeed")
	}
	
	if !verifyResp.Valid {
		t.Error("Signature should be valid")
	}
}

func TestQKDSetupAndExecute(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	setupReq := &PQCQKDSetupRequest{
		NodeAID:     "alice-test",
		NodeBID:     "bob-test",
		PhotonCount: 1000,
	}
	
	setupResp, err := service.SetupQKDChannel(ctx, setupReq)
	if err != nil {
		t.Fatalf("Failed to setup QKD channel: %v", err)
	}
	
	if !setupResp.Success {
		t.Error("QKD setup should succeed")
	}
	
	if setupResp.Channel == nil {
		t.Fatal("Channel should not be nil")
	}
	
	if setupResp.Channel.ID == "" {
		t.Error("Channel ID should not be empty")
	}
	
	executeReq := &PQCQKDExecuteRequest{
		ChannelID: setupResp.Channel.ID,
	}
	
	executeResp, err := service.ExecuteQKDProtocol(ctx, executeReq)
	if err != nil {
		t.Fatalf("Failed to execute QKD protocol: %v", err)
	}
	
	if !executeResp.Success {
		t.Error("QKD execution should succeed")
	}
	
	if executeResp.Result == nil {
		t.Fatal("QKD result should not be nil")
	}
	
	if executeResp.Result.FinalKey == nil {
		t.Error("Final key should not be nil")
	}
	
	if executeResp.Result.SecurityLevel < 0 || executeResp.Result.SecurityLevel > 1 {
		t.Errorf("Security level should be between 0 and 1, got %f", executeResp.Result.SecurityLevel)
	}
}

func TestGetQKDChannel(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	setupReq := &PQCQKDSetupRequest{
		NodeAID:     "alice-get",
		NodeBID:     "bob-get",
		PhotonCount: 500,
	}
	
	setupResp, err := service.SetupQKDChannel(ctx, setupReq)
	if err != nil {
		t.Fatalf("Failed to setup QKD channel: %v", err)
	}
	
	channel, err := service.GetQKDChannel(ctx, setupResp.Channel.ID)
	if err != nil {
		t.Fatalf("Failed to get QKD channel: %v", err)
	}
	
	if channel == nil {
		t.Fatal("Channel should not be nil")
	}
	
	if channel.ID != setupResp.Channel.ID {
		t.Errorf("Channel ID mismatch: expected %s, got %s", setupResp.Channel.ID, channel.ID)
	}
}

func TestSerializeDeserializeHybridResult(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	req := &PQCEncryptionRequest{
		Plaintext: "Test serialization",
		UseHybrid: true,
	}
	
	encryptResp, err := service.Encrypt(ctx, req)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}
	
	serialized, err := SerializeHybridResult(encryptResp.HybridResult)
	if err != nil {
		t.Fatalf("Failed to serialize: %v", err)
	}
	
	if serialized == "" {
		t.Error("Serialized data should not be empty")
	}
	
	deserialized, err := DeserializeHybridResult(serialized)
	if err != nil {
		t.Fatalf("Failed to deserialize: %v", err)
	}
	
	if deserialized == nil {
		t.Fatal("Deserialized result should not be nil")
	}
	
	if deserialized.HybridScheme != encryptResp.HybridResult.HybridScheme {
		t.Error("Hybrid scheme mismatch")
	}
}

func TestMultipleEncryptions(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	messages := []string{
		"Message 1",
		"Message 2 with special characters: !@#$%^&*()",
		"Message 3 with longer text to test various sizes",
	}
	
	for _, msg := range messages {
		req := &PQCEncryptionRequest{
			Plaintext: msg,
			UseHybrid: true,
		}
		
		resp, err := service.Encrypt(ctx, req)
		if err != nil {
			t.Fatalf("Failed to encrypt message '%s': %v", msg, err)
		}
		
		if !resp.Success {
			t.Errorf("Encryption should succeed for message: %s", msg)
		}
	}
}

func TestMultipleSignatures(t *testing.T) {
	service := NewPostQuantumCryptoService()
	ctx := context.Background()
	
	messages := []string{
		"Important document",
		"Financial transaction",
		"Legal contract",
	}
	
	for _, msg := range messages {
		signReq := &PQCSigningRequest{
			Message: msg,
		}
		
		signResp, err := service.Sign(ctx, signReq)
		if err != nil {
			t.Fatalf("Failed to sign message '%s': %v", msg, err)
		}
		
		if !signResp.Success {
			t.Errorf("Signing should succeed for message: %s", msg)
		}
		
		verifyReq := &PQCVerificationRequest{
			Message:   msg,
			Signature: signResp.Signature,
			PublicKey: signResp.PublicKey,
		}
		
		verifyResp, err := service.Verify(ctx, verifyReq)
		if err != nil {
			t.Fatalf("Failed to verify message '%s': %v", msg, err)
		}
		
		if !verifyResp.Valid {
			t.Errorf("Signature should be valid for message: %s", msg)
		}
	}
}