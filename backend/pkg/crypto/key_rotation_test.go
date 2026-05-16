package crypto

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"
)

func TestDynamicKeyManager(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	key, keyID := manager.GetCurrentKey()
	if key == nil {
		t.Error("Current key should not be nil")
	}

	if keyID == "" {
		t.Error("Key ID should not be empty")
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}

func TestDynamicKeyManagerRotation(t *testing.T) {
	manager, err := NewDynamicKeyManager(1*time.Second, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	initialKeyID := manager.GetCurrentKey()

	time.Sleep(1100 * time.Millisecond)

	err = manager.Rotate()
	if err != nil {
		t.Fatalf("Failed to rotate: %v", err)
	}

	newKeyID, _ := manager.GetCurrentKey()
	if newKeyID == nil {
		t.Error("New key should not be nil")
	}

	if fmt.Sprintf("%x", initialKeyID) == fmt.Sprintf("%x", newKeyID) {
		t.Error("Key should have changed after rotation")
	}
}

func TestDynamicKeyManagerGetKeyByID(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	initialKeyID := manager.GetKeyInfo()
	if initialKeyID == nil {
		t.Error("Key info should not be nil")
	}

	_, keyID := manager.GetCurrentKey()
	key, exists := manager.GetKeyByID(keyID)
	if !exists {
		t.Error("Key should exist")
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}

	_, exists = manager.GetKeyByID("nonexistent")
	if exists {
		t.Error("Non-existent key should not be found")
	}
}

func TestDynamicKeyManagerMaxKeys(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 3)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	initialKeyCount := len(manager.GetAllKeys())

	for i := 0; i < 5; i++ {
		manager.Rotate()
		time.Sleep(10 * time.Millisecond)
	}

	finalKeyCount := len(manager.GetAllKeys())
	if finalKeyCount > 3 {
		t.Errorf("Key count %d exceeds max keys %d", finalKeyCount, 3)
	}

	if finalKeyCount <= initialKeyCount {
		t.Error("Key count should have increased")
	}
}

func TestDynamicKeyManagerEncryption(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	plaintext := []byte("sensitive data for encryption")
	ciphertext, err := manager.EncryptWithCurrentKey(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	if len(ciphertext) <= len(plaintext) {
		t.Error("Ciphertext should be longer than plaintext (includes nonce)")
	}

	keyInfo := manager.GetKeyInfo()
	keyID := keyInfo.KeyID

	decrypted, err := manager.DecryptWithKey(string(ciphertext), keyID)
	if err != nil {
		t.Fatalf("Failed to decrypt: %v", err)
	}

	if string(decrypted) != string(plaintext) {
		t.Error("Decrypted data does not match original plaintext")
	}
}

func TestDynamicKeyManagerRotationStatus(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	status := manager.GetRotationStatus()

	if status["current_key_id"] == nil || status["current_key_id"] == "" {
		t.Error("Status should contain current_key_id")
	}

	if status["total_keys"] == nil || status["total_keys"].(int) < 1 {
		t.Error("Status should contain at least one key")
	}

	if status["rotation_enabled"] != true {
		t.Error("Rotation should be enabled")
	}

	if status["interval"] == nil {
		t.Error("Status should contain rotation interval")
	}
}

func TestDynamicKeyManagerEnableRotation(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	manager.EnableRotation(false)

	manager.mu.RLock()
	enabled := manager.rotationEnabled
	manager.mu.RUnlock()

	if enabled != false {
		t.Error("Rotation should be disabled")
	}

	manager.EnableRotation(true)

	manager.mu.RLock()
	enabled = manager.rotationEnabled
	manager.mu.RUnlock()

	if enabled != true {
		t.Error("Rotation should be enabled")
	}
}

func TestDynamicKeyManagerSetRotationInterval(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	manager.SetRotationInterval(10 * time.Minute)

	manager.mu.RLock()
	interval := manager.rotationInterval
	manager.mu.RUnlock()

	if interval != 10*time.Minute {
		t.Errorf("Expected interval 10m, got %v", interval)
	}
}

func TestDynamicKeyManagerVersionHistory(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	for i := 0; i < 3; i++ {
		manager.Rotate()
		time.Sleep(10 * time.Millisecond)
	}

	versions := manager.GetVersionHistory()
	if len(versions) < 3 {
		t.Errorf("Expected at least 3 versions, got %d", len(versions))
	}

	for _, v := range versions {
		if v.KeyID == "" {
			t.Error("Version should have a key ID")
		}
	}
}

func TestDynamicKeyManagerMetrics(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	metrics := manager.GetMetrics()

	if metrics.TotalRotations < 0 {
		t.Error("Total rotations should be non-negative")
	}

	if metrics.ActiveKeysCount < 1 {
		t.Error("Should have at least one active key")
	}

	if metrics.RotationFrequency == 0 {
		t.Error("Rotation frequency should be set")
	}
}

func TestDynamicKeyManagerExportKeysJSON(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	jsonStr, err := manager.ExportKeysJSON()
	if err != nil {
		t.Fatalf("Failed to export keys: %v", err)
	}

	if jsonStr == "" {
		t.Error("Exported JSON should not be empty")
	}

	var exportedKeys []map[string]interface{}
	err = json.Unmarshal([]byte(jsonStr), &exportedKeys)
	if err != nil {
		t.Fatalf("Failed to parse exported JSON: %v", err)
	}

	if len(exportedKeys) < 1 {
		t.Error("Should have exported at least one key")
	}
}

func TestDynamicKeyManagerBackup(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	backup, err := manager.CreateBackup()
	if err != nil {
		t.Fatalf("Failed to create backup: %v", err)
	}

	if backup.KeyID == "" {
		t.Error("Backup should have a key ID")
	}

	if len(backup.KeyData) != 32 {
		t.Errorf("Expected backup key length 32, got %d", len(backup.KeyData))
	}

	if backup.Checksum == "" {
		t.Error("Backup should have a checksum")
	}
}

func TestDynamicKeyManagerRestoreBackup(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	backup, _ := manager.CreateBackup()
	originalKeyID := backup.KeyID

	manager.Rotate()
	time.Sleep(10 * time.Millisecond)

	err = manager.RestoreFromBackup(backup)
	if err != nil {
		t.Fatalf("Failed to restore backup: %v", err)
	}

	key, exists := manager.GetKeyByID(originalKeyID)
	if !exists {
		t.Error("Restored key should exist")
	}

	if len(key) != 32 {
		t.Errorf("Expected key length 32, got %d", len(key))
	}
}

func TestDynamicKeyManagerValidateKey(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	keyInfo := manager.GetKeyInfo()
	keyID := keyInfo.KeyID

	if !manager.ValidateKey(keyID) {
		t.Error("Current key should be valid")
	}

	if manager.ValidateKey("nonexistent") {
		t.Error("Non-existent key should not be valid")
	}
}

func TestDynamicKeyManagerMarkKeyUsed(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	keyInfo := manager.GetKeyInfo()
	keyID := keyInfo.KeyID

	manager.MarkKeyUsed(keyID)

	key, exists := manager.GetKeyByID(keyID)
	if !exists {
		t.Error("Key should exist after marking as used")
	}
}

func TestDynamicKeyManagerPolicy(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	policy := manager.GetPolicy()

	if policy.Interval != 5*time.Minute {
		t.Errorf("Expected interval 5m, got %v", policy.Interval)
	}

	if policy.MaxKeys != 5 {
		t.Errorf("Expected max keys 5, got %d", policy.MaxKeys)
	}

	if policy.NotifyBefore == 0 {
		t.Error("NotifyBefore should be set")
	}
}

func TestDynamicKeyManagerUpdatePolicy(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	policy := KeyRotationPolicy{
		Interval: 10 * time.Minute,
		MaxKeys:  10,
	}

	manager.UpdatePolicy(policy)

	updatedPolicy := manager.GetPolicy()
	if updatedPolicy.Interval != 10*time.Minute {
		t.Errorf("Expected interval 10m, got %v", updatedPolicy.Interval)
	}

	if updatedPolicy.MaxKeys != 10 {
		t.Errorf("Expected max keys 10, got %d", updatedPolicy.MaxKeys)
	}
}

func TestDynamicKeyManagerAuditLog(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	log := manager.CreateAuditLog("test_action", "Test details", true)

	if log.Action != "test_action" {
		t.Error("Audit log action mismatch")
	}

	if log.Details != "Test details" {
		t.Error("Audit log details mismatch")
	}

	if log.Success != true {
		t.Error("Audit log success should be true")
	}

	if log.Timestamp.IsZero() {
		t.Error("Audit log timestamp should be set")
	}
}

func TestDynamicKeyManagerCallback(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	eventReceived := false

	manager.RegisterCallback(func(event KeyRotationEvent) {
		eventReceived = true
	})

	manager.Rotate()

	time.Sleep(100 * time.Millisecond)

	if !eventReceived {
		t.Error("Callback should have received event")
	}
}

func TestDynamicKeyManagerGetAllKeys(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	initialCount := len(manager.GetAllKeys())

	manager.Rotate()
	time.Sleep(10 * time.Millisecond)
	manager.Rotate()
	time.Sleep(10 * time.Millisecond)

	finalCount := len(manager.GetAllKeys())

	if finalCount <= initialCount {
		t.Error("Key count should increase after rotations")
	}
}

func TestDynamicKeyManagerConcurrentAccess(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 10)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	done := make(chan bool)

	for i := 0; i < 10; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				_, keyID := manager.GetCurrentKey()
				if keyID == "" {
					t.Error("Key ID should not be empty")
				}
			}
			done <- true
		}()
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestDynamicKeyManagerEncryptionWithRotation(t *testing.T) {
	manager, err := NewDynamicKeyManager(5*time.Minute, 5)
	if err != nil {
		t.Fatalf("Failed to create key manager: %v", err)
	}

	plaintext := []byte("test data for encryption")
	ciphertext, err := manager.EncryptWithCurrentKey(plaintext)
	if err != nil {
		t.Fatalf("Failed to encrypt: %v", err)
	}

	manager.Rotate()

	keyInfo := manager.GetKeyInfo()
	_, exists := manager.GetKeyByID(keyInfo.KeyID)

	if !exists {
		t.Error("Current key should exist after rotation")
	}
}

func BenchmarkDynamicKeyManagerSign(b *testing.B) {
	manager, _ := NewDynamicKeyManager(5*time.Minute, 5)
	message := []byte("benchmark test data")

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.EncryptWithCurrentKey(message)
	}
}

func BenchmarkDynamicKeyManagerRotation(b *testing.B) {
	manager, _ := NewDynamicKeyManager(5*time.Minute, 5)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		manager.Rotate()
	}
}
