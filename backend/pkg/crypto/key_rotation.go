package crypto

import (
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrInvalidKeyVersion  = errors.New("invalid key version")
	ErrKeyExpired         = errors.New("key has expired")
	ErrKeyRevoked         = errors.New("key has been revoked")
	ErrNoActiveKey        = errors.New("no active encryption key available")
	ErrKeyStorageFailed   = errors.New("key storage operation failed")
)

type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusRetired  KeyStatus = "retired"
	KeyStatusRevoked  KeyStatus = "revoked"
	KeyStatusArchived KeyStatus = "archived"
)

type KeyAlgorithm string

const (
	AlgorithmAES256GCM        KeyAlgorithm = "AES-256-GCM"
	AlgorithmAES192GCM        KeyAlgorithm = "AES-192-GCM"
	AlgorithmAES128GCM        KeyAlgorithm = "AES-128-GCM"
	AlgorithmChaCha20Poly1305 KeyAlgorithm = "ChaCha20-Poly1305"
)

type KeyMetadata struct {
	KeyID        string        `json:"key_id"`
	Version      int           `json:"version"`
	Algorithm    KeyAlgorithm  `json:"algorithm"`
	Status       KeyStatus     `json:"status"`
	CreatedAt    time.Time     `json:"created_at"`
	ActivatedAt  time.Time     `json:"activated_at,omitempty"`
	RetiredAt    time.Time     `json:"retired_at,omitempty"`
	ExpiresAt    time.Time     `json:"expires_at,omitempty"`
	Rotations    int           `json:"rotations"`
	Description  string        `json:"description,omitempty"`
	KeyType      string        `json:"key_type"`
}

type EncryptionKey struct {
	Metadata *KeyMetadata `json:"metadata"`
	KeyData  []byte       `json:"key_data"`
}

type KeyRotationConfig struct {
	RotationInterval   time.Duration `json:"rotation_interval"`
	MaxKeyAge          time.Duration `json:"max_key_age"`
	KeyHistoryLimit    int           `json:"key_history_limit"`
	AutoRotation       bool          `json:"auto_rotation"`
	BackupBeforeRotate bool          `json:"backup_before_rotate"`
	StoragePath        string        `json:"storage_path"`
	KeyLength          int           `json:"key_length"`
	Algorithm          KeyAlgorithm  `json:"algorithm"`
}

type KeyRotationManager struct {
	config            *KeyRotationConfig
	keys              map[int]*EncryptionKey
	currentVersion    int
	rotationTimer     *time.Ticker
	mu                sync.RWMutex
	rotationCallbacks []func(*KeyRotationEvent)
	shutdownChan      chan struct{}
}

type KeyRotationEvent struct {
	EventType    string        `json:"event_type"`
	Timestamp    time.Time     `json:"timestamp"`
	OldKeyID     string        `json:"old_key_id,omitempty"`
	NewKeyID     string        `json:"new_key_id,omitempty"`
	NewVersion   int           `json:"new_version"`
	Error        error         `json:"error,omitempty"`
}

type KeyRotationStats struct {
	TotalRotations        int           `json:"total_rotations"`
	SuccessfulRotations   int           `json:"successful_rotations"`
	FailedRotations       int           `json:"failed_rotations"`
	CurrentKeyAge         time.Duration `json:"current_key_age"`
	LastRotationTime      time.Time     `json:"last_rotation_time"`
	NextRotationTime      time.Time     `json:"next_rotation_time"`
	ActiveKeyVersion      int           `json:"active_key_version"`
	TotalKeyVersions      int           `json:"total_key_versions"`
}

var defaultKeyRotationConfig = KeyRotationConfig{
	RotationInterval:   24 * time.Hour,
	MaxKeyAge:          7 * 24 * time.Hour,
	KeyHistoryLimit:    10,
	AutoRotation:       true,
	BackupBeforeRotate: true,
	KeyLength:          32,
	Algorithm:          AlgorithmAES256GCM,
}

func NewKeyRotationManager(config *KeyRotationConfig) (*KeyRotationManager, error) {
	if config == nil {
		config = &defaultKeyRotationConfig
	}

	manager := &KeyRotationManager{
		config:         config,
		keys:           make(map[int]*EncryptionKey),
		currentVersion: 0,
		shutdownChan:   make(chan struct{}),
	}

	if err := manager.loadKeysFromStorage(); err != nil {
		return nil, fmt.Errorf("failed to load keys from storage: %w", err)
	}

	if err := manager.ensureActiveKey(); err != nil {
		return nil, fmt.Errorf("failed to ensure active key: %w", err)
	}

	if config.AutoRotation {
		manager.startAutoRotation()
	}

	return manager, nil
}

func (m *KeyRotationManager) ensureActiveKey() error {
	m.mu.Lock()
	if m.currentVersion > 0 && m.keys[m.currentVersion] != nil {
		if m.keys[m.currentVersion].Metadata.Status == KeyStatusActive {
			m.mu.Unlock()
			return nil
		}
	}
	m.mu.Unlock()

	return m.generateNewKey()
}

func (m *KeyRotationManager) generateNewKey() error {
	m.mu.Lock()

	keyLength := m.config.KeyLength
	if keyLength == 0 {
		keyLength = 32
	}

	keyData, err := GenerateRandomKey(AESKeySize(keyLength))
	if err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to generate key: %w", err)
	}

	m.currentVersion++

	expiresAt := time.Now().Add(m.config.MaxKeyAge)

	metadata := &KeyMetadata{
		KeyID:       generateUUID(),
		Version:     m.currentVersion,
		Algorithm:   m.config.Algorithm,
		Status:      KeyStatusActive,
		CreatedAt:   time.Now(),
		ActivatedAt: time.Now(),
		ExpiresAt:   expiresAt,
		Rotations:   m.currentVersion - 1,
		KeyType:     "data",
	}

	if m.currentVersion > 1 && m.keys[m.currentVersion-1] != nil {
		metadata.Rotations = m.keys[m.currentVersion-1].Metadata.Rotations + 1
	}

	encryptionKey := &EncryptionKey{
		Metadata: metadata,
		KeyData:  keyData,
	}

	m.keys[m.currentVersion] = encryptionKey

	if m.currentVersion > 1 {
		if err := m.retirePreviousKey(); err != nil {
			m.mu.Unlock()
			return fmt.Errorf("failed to retire previous key: %w", err)
		}
	}

	if err := m.saveKeysToStorage(); err != nil {
		m.mu.Unlock()
		return fmt.Errorf("failed to save keys: %w", err)
	}

	newKeyID := metadata.KeyID
	newVersion := metadata.Version

	m.mu.Unlock()

	m.notify(KeyRotationEvent{
		EventType:  "key_created",
		Timestamp:  time.Now(),
		NewKeyID:   newKeyID,
		NewVersion: newVersion,
	})

	return nil
}

func (m *KeyRotationManager) retirePreviousKey() error {
	prevKey := m.keys[m.currentVersion-1]
	if prevKey != nil && prevKey.Metadata.Status == KeyStatusActive {
		prevKey.Metadata.Status = KeyStatusRetired
		prevKey.Metadata.RetiredAt = time.Now()
	}
	return nil
}

func (m *KeyRotationManager) RotateKey() (*EncryptionKey, error) {
	m.mu.Lock()
	oldKeyID := ""
	if m.keys[m.currentVersion] != nil {
		oldKeyID = m.keys[m.currentVersion].Metadata.KeyID
	}

	needBackup := m.config.BackupBeforeRotate
	m.mu.Unlock()

	if needBackup {
		if err := m.backupKeys(); err != nil {
			return nil, fmt.Errorf("backup failed: %w", err)
		}
	}

	err := m.generateNewKey()
	if err != nil {
		m.notify(KeyRotationEvent{
			EventType: "rotation_failed",
			Timestamp: time.Now(),
			OldKeyID:  oldKeyID,
			Error:     err,
		})
		return nil, err
	}

	m.mu.RLock()
	newKey := m.keys[m.currentVersion]
	newKeyID := newKey.Metadata.KeyID
	newVersion := newKey.Metadata.Version
	m.mu.RUnlock()

	m.notify(KeyRotationEvent{
		EventType:  "rotation_completed",
		Timestamp:  time.Now(),
		OldKeyID:   oldKeyID,
		NewKeyID:   newKeyID,
		NewVersion: newVersion,
	})

	return newKey, nil
}

func (m *KeyRotationManager) RotateKeyWithAlgorithm(algorithm KeyAlgorithm) (*EncryptionKey, error) {
	m.mu.Lock()
	originalAlgorithm := m.config.Algorithm
	m.config.Algorithm = algorithm
	m.mu.Unlock()

	key, err := m.RotateKey()

	m.mu.Lock()
	m.config.Algorithm = originalAlgorithm
	m.mu.Unlock()

	return key, err
}

func (m *KeyRotationManager) GetCurrentKey() (*EncryptionKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.currentVersion == 0 {
		return nil, ErrNoActiveKey
	}

	key := m.keys[m.currentVersion]
	if key == nil {
		return nil, ErrNoActiveKey
	}

	if key.Metadata.Status != KeyStatusActive {
		return nil, ErrKeyRevoked
	}

	if !key.Metadata.ExpiresAt.IsZero() && time.Now().After(key.Metadata.ExpiresAt) {
		return nil, ErrKeyExpired
	}

	return key, nil
}

func (m *KeyRotationManager) GetKeyByVersion(version int) (*EncryptionKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	key, exists := m.keys[version]
	if !exists {
		return nil, ErrKeyNotFound
	}

	return key, nil
}

func (m *KeyRotationManager) GetKeyByID(keyID string) (*EncryptionKey, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, key := range m.keys {
		if key.Metadata.KeyID == keyID {
			return key, nil
		}
	}

	return nil, ErrKeyNotFound
}

func (m *KeyRotationManager) GetAllKeys() []*EncryptionKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]*EncryptionKey, 0, len(m.keys))
	for _, key := range m.keys {
		keys = append(keys, key)
	}

	return keys
}

func (m *KeyRotationManager) GetActiveKeys() []*EncryptionKey {
	m.mu.RLock()
	defer m.mu.RUnlock()

	keys := make([]*EncryptionKey, 0)
	for _, key := range m.keys {
		if key.Metadata.Status == KeyStatusActive {
			keys = append(keys, key)
		}
	}

	return keys
}

func (m *KeyRotationManager) RevokeKey(version int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, exists := m.keys[version]
	if !exists {
		return ErrKeyNotFound
	}

	if version == m.currentVersion {
		return errors.New("cannot revoke current active key")
	}

	key.Metadata.Status = KeyStatusRevoked
	key.Metadata.RetiredAt = time.Now()

	return m.saveKeysToStorage()
}

func (m *KeyRotationManager) ArchiveKey(version int) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	key, exists := m.keys[version]
	if !exists {
		return ErrKeyNotFound
	}

	if version == m.currentVersion {
		return errors.New("cannot archive current active key")
	}

	key.Metadata.Status = KeyStatusArchived

	return m.saveKeysToStorage()
}

func (m *KeyRotationManager) PurgeOldKeys(maxAge time.Duration) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	now := time.Now()
	purgedCount := 0

	for version, key := range m.keys {
		if version == m.currentVersion {
			continue
		}

		if key.Metadata.Status == KeyStatusRevoked ||
			key.Metadata.Status == KeyStatusArchived ||
			now.Sub(key.Metadata.CreatedAt) > maxAge {
			delete(m.keys, version)
			purgedCount++
		}
	}

	if purgedCount > 0 {
		if err := m.saveKeysToStorage(); err != nil {
			return fmt.Errorf("failed to save after purge: %w", err)
		}
	}

	return nil
}

func (m *KeyRotationManager) GetStats() KeyRotationStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := KeyRotationStats{
		TotalKeyVersions: len(m.keys),
		ActiveKeyVersion: m.currentVersion,
	}

	if m.currentVersion > 0 && m.keys[m.currentVersion] != nil {
		currentKey := m.keys[m.currentVersion]
		stats.CurrentKeyAge = time.Since(currentKey.Metadata.CreatedAt)

		for _, key := range m.keys {
			if key.Metadata.Rotations > stats.TotalRotations {
				stats.TotalRotations = key.Metadata.Rotations
			}
		}

		if currentKey.Metadata.ActivatedAt.IsZero() {
			stats.NextRotationTime = time.Now()
		} else {
			stats.NextRotationTime = currentKey.Metadata.ActivatedAt.Add(m.config.RotationInterval)
		}
	}

	return stats
}

func (m *KeyRotationManager) startAutoRotation() {
	m.mu.Lock()
	if m.rotationTimer != nil {
		m.rotationTimer.Stop()
	}
	m.rotationTimer = time.NewTicker(m.config.RotationInterval)
	m.mu.Unlock()

	go func() {
		for {
			select {
			case <-m.rotationTimer.C:
				_, err := m.RotateKey()
				if err != nil {
					fmt.Printf("Auto key rotation failed: %v\n", err)
				}
			case <-m.shutdownChan:
				return
			}
		}
	}()

	fmt.Printf("Auto key rotation started, interval: %v\n", m.config.RotationInterval)
}

func (m *KeyRotationManager) StopAutoRotation() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.rotationTimer != nil {
		m.rotationTimer.Stop()
		m.rotationTimer = nil
	}
	close(m.shutdownChan)
}

func (m *KeyRotationManager) SetRotationInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.config.RotationInterval = interval

	if m.rotationTimer != nil {
		m.rotationTimer.Stop()
		m.rotationTimer = time.NewTicker(interval)
	}
}

func (m *KeyRotationManager) RegisterRotationCallback(callback func(*KeyRotationEvent)) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.rotationCallbacks = append(m.rotationCallbacks, callback)
}

func (m *KeyRotationManager) notify(event KeyRotationEvent) {
	m.mu.RLock()
	callbacks := make([]func(*KeyRotationEvent), len(m.rotationCallbacks))
	copy(callbacks, m.rotationCallbacks)
	m.mu.RUnlock()

	for _, callback := range callbacks {
		go callback(&event)
	}
}

func (m *KeyRotationManager) saveKeysToStorage() error {
	if m.config.StoragePath == "" {
		return nil
	}

	if err := os.MkdirAll(m.config.StoragePath, 0700); err != nil {
		return fmt.Errorf("failed to create storage directory: %w", err)
	}

	data := make(map[int]keyStorageEntry)
	for version, key := range m.keys {
		data[version] = keyStorageEntry{
			Metadata: key.Metadata,
			KeyData:  base64.StdEncoding.EncodeToString(key.KeyData),
		}
	}

	jsonData, err := json.MarshalIndent(data, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal keys: %w", err)
	}

	filePath := filepath.Join(m.config.StoragePath, "keys.json")
	if err := os.WriteFile(filePath, jsonData, 0600); err != nil {
		return fmt.Errorf("failed to write keys: %w", err)
	}

	return nil
}

func (m *KeyRotationManager) loadKeysFromStorage() error {
	if m.config.StoragePath == "" {
		return nil
	}

	filePath := filepath.Join(m.config.StoragePath, "keys.json")
	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read keys: %w", err)
	}

	var storedData map[int]keyStorageEntry
	if err := json.Unmarshal(data, &storedData); err != nil {
		return fmt.Errorf("failed to unmarshal keys: %w", err)
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for version, entry := range storedData {
		keyData, err := base64.StdEncoding.DecodeString(entry.KeyData)
		if err != nil {
			return fmt.Errorf("failed to decode key data for version %d: %w", version, err)
		}

		m.keys[version] = &EncryptionKey{
			Metadata: entry.Metadata,
			KeyData:  keyData,
		}

		if version > m.currentVersion {
			m.currentVersion = version
		}
	}

	return nil
}

func (m *KeyRotationManager) backupKeys() error {
	if m.config.StoragePath == "" {
		return nil
	}

	filePath := filepath.Join(m.config.StoragePath, "keys.json")
	backupPath := fmt.Sprintf("%s.backup.%d", filePath, time.Now().Unix())

	data, err := os.ReadFile(filePath)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read keys for backup: %w", err)
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return nil
}

type keyStorageEntry struct {
	Metadata *KeyMetadata `json:"metadata"`
	KeyData  string       `json:"key_data"`
}

func generateUUID() string {
	b := make([]byte, 16)
	rand.Read(b)
	b[6] = (b[6] & 0x0F) | 0x40
	b[8] = (b[8] & 0x3F) | 0x80
	return fmt.Sprintf("%x-%x-%x-%x-%x", b[0:4], b[4:6], b[6:8], b[8:10], b[10:16])
}

func (m *KeyRotationManager) EncryptWithCurrentKey(plaintext []byte) ([]byte, error) {
	key, err := m.GetCurrentKey()
	if err != nil {
		return nil, err
	}

	switch key.Metadata.Algorithm {
	case AlgorithmAES256GCM, AlgorithmAES192GCM, AlgorithmAES128GCM:
		return AESEncrypt(plaintext, key.KeyData)
	case AlgorithmChaCha20Poly1305:
		return ChaCha20Poly1305Encrypt(plaintext, key.KeyData)
	default:
		return AESEncrypt(plaintext, key.KeyData)
	}
}

func (m *KeyRotationManager) DecryptWithKey(version int, ciphertext []byte) ([]byte, error) {
	key, err := m.GetKeyByVersion(version)
	if err != nil {
		return nil, err
	}

	if key.Metadata.Status == KeyStatusRevoked {
		return nil, ErrKeyRevoked
	}

	switch key.Metadata.Algorithm {
	case AlgorithmAES256GCM, AlgorithmAES192GCM, AlgorithmAES128GCM:
		return AESDecrypt(ciphertext, key.KeyData)
	case AlgorithmChaCha20Poly1305:
		return ChaCha20Poly1305Decrypt(ciphertext, key.KeyData)
	default:
		return AESDecrypt(ciphertext, key.KeyData)
	}
}

func (m *KeyRotationManager) EncryptStringWithCurrentKey(plaintext string) (string, error) {
	key, err := m.GetCurrentKey()
	if err != nil {
		return "", err
	}

	switch key.Metadata.Algorithm {
	case AlgorithmAES256GCM, AlgorithmAES192GCM, AlgorithmAES128GCM:
		return AESEncryptString(plaintext, key.KeyData)
	case AlgorithmChaCha20Poly1305:
		ciphertext, err := ChaCha20Poly1305Encrypt([]byte(plaintext), key.KeyData)
		if err != nil {
			return "", err
		}
		return base64.StdEncoding.EncodeToString(ciphertext), nil
	default:
		return AESEncryptString(plaintext, key.KeyData)
	}
}

func (m *KeyRotationManager) DecryptStringWithKey(version int, ciphertextBase64 string) (string, error) {
	key, err := m.GetKeyByVersion(version)
	if err != nil {
		return "", err
	}

	if key.Metadata.Status == KeyStatusRevoked {
		return "", ErrKeyRevoked
	}

	switch key.Metadata.Algorithm {
	case AlgorithmAES256GCM, AlgorithmAES192GCM, AlgorithmAES128GCM:
		return AESDecryptString(ciphertextBase64, key.KeyData)
	case AlgorithmChaCha20Poly1305:
		ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
		if err != nil {
			return "", err
		}
		plaintext, err := ChaCha20Poly1305Decrypt(ciphertext, key.KeyData)
		if err != nil {
			return "", err
		}
		return string(plaintext), nil
	default:
		return AESDecryptString(ciphertextBase64, key.KeyData)
	}
}