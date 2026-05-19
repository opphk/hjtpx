package service

import (
	"bytes"
	"testing"
)

func TestNewWASMCryptoEngineV3(t *testing.T) {
	engine := NewWASMCryptoEngineV3()
	if engine == nil {
		t.Fatal("Expected engine to be non-nil")
	}
	if engine.algorithmType != "AES-256-GCM" {
		t.Errorf("Expected AES-256-GCM, got %s", engine.algorithmType)
	}
}

func TestWASMCryptoEncryptDecrypt(t *testing.T) {
	engine := NewWASMCryptoEngineV3()
	plaintext := []byte("Hello, World! This is a test message for encryption.")

	// 测试加密
	ciphertext, err := engine.Encrypt(plaintext)
	if err != nil {
		t.Fatalf("Encrypt failed: %v", err)
	}
	if len(ciphertext) == 0 {
		t.Fatal("Ciphertext is empty")
	}

	// 测试解密
	decrypted, err := engine.Decrypt(ciphertext)
	if err != nil {
		t.Fatalf("Decrypt failed: %v", err)
	}

	if !bytes.Equal(plaintext, decrypted) {
		t.Errorf("Decrypted text doesn't match. Expected '%s', got '%s'", plaintext, decrypted)
	}
}

func TestEmptyInput(t *testing.T) {
	engine := NewWASMCryptoEngineV3()
	
	_, err := engine.Encrypt(nil)
	if err == nil {
		t.Error("Expected error for nil input")
	}
	
	_, err = engine.Encrypt([]byte{})
	if err == nil {
		t.Error("Expected error for empty input")
	}
}

func TestWASMCryptoPerformanceMetrics(t *testing.T) {
	engine := NewWASMCryptoEngineV3()
	
	// 先做一些操作
	plaintext := []byte("Test data")
	for i := 0; i < 10; i++ {
		ciphertext, _ := engine.Encrypt(plaintext)
		engine.Decrypt(ciphertext)
	}
	
	metrics := engine.GetPerformanceMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be non-nil")
	}
	
	if metrics.TotalOperations == 0 {
		t.Error("Expected total operations to be greater than 0")
	}
}

func TestWASMCryptoResetStats(t *testing.T) {
	engine := NewWASMCryptoEngineV3()
	
	// 做一些操作
	plaintext := []byte("Test")
	ciphertext, _ := engine.Encrypt(plaintext)
	engine.Decrypt(ciphertext)
	
	// 重置统计
	engine.ResetStats()
	
	metrics := engine.GetPerformanceMetrics()
	if metrics.TotalOperations != 0 {
		t.Error("Expected total operations to be 0 after reset")
	}
}

func TestStreamEncryptDecrypt(t *testing.T) {
	engine := NewWASMCryptoEngineV3()
	
	// 创建大量数据
	plaintext := make([]byte, 1024*1024) // 1MB
	for i := range plaintext {
		plaintext[i] = byte(i % 256)
	}
	
	// 测试流式加密
	ciphertext, err := engine.StreamEncrypt(plaintext)
	if err != nil {
		t.Fatalf("StreamEncrypt failed: %v", err)
	}
	
	// 测试流式解密
	decrypted, err := engine.StreamDecrypt(ciphertext)
	if err != nil {
		t.Fatalf("StreamDecrypt failed: %v", err)
	}
	
	if !bytes.Equal(plaintext, decrypted) {
		t.Error("Decrypted stream doesn't match")
	}
}
