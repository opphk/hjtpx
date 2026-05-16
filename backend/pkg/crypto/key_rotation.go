package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

type KeyRotationEventType string

const (
	KeyRotated           KeyRotationEventType = "rotated"
	KeyRotationScheduled KeyRotationEventType = "scheduled"
	KeyRotationFailed    KeyRotationEventType = "failed"
)

type KeyRotationEvent struct {
	Type          KeyRotationEventType `json:"type"`
	Timestamp     time.Time            `json:"timestamp"`
	KeyID         string               `json:"key_id"`
	PreviousKeyID string               `json:"previous_key_id,omitempty"`
	Error         string               `json:"error,omitempty"`
}

type KeyRotationCallback func(event KeyRotationEvent)

type DynamicKeyManager struct {
	keys             map[string]*DynamicKey
	currentKeyID     string
	mu               sync.RWMutex
	rotationInterval time.Duration
	maxKeys          int
	callbacks        []KeyRotationCallback
	lastRotation     time.Time
	rotationEnabled  bool
}

type DynamicKey struct {
	KeyID        string    `json:"key_id"`
	Key          []byte    `json:"-"` // 敏感数据不序列化
	KeyBase64    string    `json:"key_base64,omitempty"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	LastUsedAt   time.Time `json:"last_used_at"`
	RotationCount int      `json:"rotation_count"`
	Metadata     map[string]string `json:"metadata,omitempty"`
}

func NewDynamicKeyManager(rotationInterval time.Duration, maxKeys int) (*DynamicKeyManager, error) {
	m := &DynamicKeyManager{
		keys:             make(map[string]*DynamicKey),
		rotationInterval: rotationInterval,
		maxKeys:          maxKeys,
		lastRotation:     time.Now(),
		rotationEnabled:  true,
	}
	
	if err := m.generateNewKey(); err != nil {
		return nil, err
	}
	
	go m.autoRotation()
	
	return m, nil
}

func (m *DynamicKeyManager) generateNewKey() error {
	keyBytes := make([]byte, 32)
	if _, err := rand.Read(keyBytes); err != nil {
		return fmt.Errorf("failed to generate random key: %w", err)
	}
	
	keyID := generateKeyID()
	now := time.Now()
	
	dynamicKey := &DynamicKey{
		KeyID:        keyID,
		Key:          keyBytes,
		KeyBase64:    base64.StdEncoding.EncodeToString(keyBytes),
		Version:      len(m.keys) + 1,
		CreatedAt:    now,
		ExpiresAt:    now.Add(m.rotationInterval),
		LastUsedAt:   now,
		RotationCount: 0,
		Metadata:     make(map[string]string),
	}
	
	m.mu.Lock()
	if m.currentKeyID != "" && m.keys[m.currentKeyID] != nil {
		m.keys[m.currentKeyID].RotationCount++
	}
	
	m.currentKeyID = keyID
	m.keys[keyID] = dynamicKey
	m.lastRotation = now
	
	if len(m.keys) > m.maxKeys {
		m.cleanupOldKeys()
	}
	m.mu.Unlock()
	
	m.emitEvent(KeyRotationEvent{
		Type:          KeyRotated,
		Timestamp:     now,
		KeyID:         keyID,
		PreviousKeyID: m.getPreviousKeyID(),
	})
	
	return nil
}

func (m *DynamicKeyManager) getPreviousKeyID() string {
	var previousID string
	m.mu.RLock()
	for id := range m.keys {
		if id != m.currentKeyID {
			previousID = id
		}
	}
	m.mu.RUnlock()
	return previousID
}

func (m *DynamicKeyManager) cleanupOldKeys() {
	if len(m.keys) <= m.maxKeys {
		return
	}
	
	oldestID := m.currentKeyID
	var oldestTime time.Time
	
	for id, key := range m.keys {
		if id != m.currentKeyID {
			if oldestTime.IsZero() || key.CreatedAt.Before(oldestTime) {
				oldestTime = key.CreatedAt
				oldestID = id
			}
		}
	}
	
	delete(m.keys, oldestID)
}

func (m *DynamicKeyManager) GetCurrentKey() ([]byte, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if key, exists := m.keys[m.currentKeyID]; exists {
		return key.Key, key.KeyID
	}
	return nil, ""
}

func (m *DynamicKeyManager) GetKeyInfo() *DynamicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if key, exists := m.keys[m.currentKeyID]; exists {
		return key
	}
	return nil
}

func (m *DynamicKeyManager) GetAllKeys() []*DynamicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	keys := make([]*DynamicKey, 0, len(m.keys))
	for _, k := range m.keys {
		keys = append(keys, k)
	}
	return keys
}

func (m *DynamicKeyManager) GetKeyByID(keyID string) ([]byte, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if key, exists := m.keys[keyID]; exists {
		return key.Key, true
	}
	return nil, false
}

func (m *DynamicKeyManager) Rotate() error {
	return m.generateNewKey()
}

func (m *DynamicKeyManager) autoRotation() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		m.mu.RLock()
		enabled := m.rotationEnabled
		m.mu.RUnlock()
		
		if !enabled {
			continue
		}
		
		m.mu.RLock()
		lastRot := m.lastRotation
		interval := m.rotationInterval
		m.mu.RUnlock()
		
		if time.Since(lastRot) >= interval {
			m.emitEvent(KeyRotationEvent{
				Type:      KeyRotationScheduled,
				Timestamp: time.Now(),
			})
			
			if err := m.Rotate(); err != nil {
				m.emitEvent(KeyRotationEvent{
					Type:      KeyRotationFailed,
					Timestamp: time.Now(),
					Error:     err.Error(),
				})
			}
		}
	}
}

func (m *DynamicKeyManager) RegisterCallback(callback KeyRotationCallback) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

func (m *DynamicKeyManager) emitEvent(event KeyRotationEvent) {
	m.mu.RLock()
	callbacks := m.callbacks
	m.mu.RUnlock()
	
	for _, cb := range callbacks {
		go cb(event)
	}
}

func (m *DynamicKeyManager) EnableRotation(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rotationEnabled = enabled
}

func (m *DynamicKeyManager) SetRotationInterval(interval time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.rotationInterval = interval
}

func (m *DynamicKeyManager) GetRotationStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	status := map[string]interface{}{
		"current_key_id":   m.currentKeyID,
		"total_keys":       len(m.keys),
		"rotation_enabled": m.rotationEnabled,
		"last_rotation":    m.lastRotation,
		"next_rotation":    m.lastRotation.Add(m.rotationInterval),
		"interval":         m.rotationInterval.String(),
	}
	
	if key, exists := m.keys[m.currentKeyID]; exists {
		status["current_key_expires"] = key.ExpiresAt
	}
	
	return status
}

func (m *DynamicKeyManager) EncryptWithCurrentKey(plaintext []byte) ([]byte, error) {
	key, keyID := m.GetCurrentKey()
	if key == nil {
		return nil, fmt.Errorf("no active key available")
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	
	ciphertext := gcm.Seal(nonce, nonce, plaintext, []byte(keyID))
	
	return ciphertext, nil
}

func (m *DynamicKeyManager) DecryptWithKey(ciphertext, keyID string) ([]byte, error) {
	key, exists := m.GetKeyByID(keyID)
	if !exists {
		return nil, fmt.Errorf("key not found: %s", keyID)
	}
	
	ciphertextBytes, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return nil, err
	}
	
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}
	
	nonceSize := gcm.NonceSize()
	if len(ciphertextBytes) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}
	
	nonce, ciphertextBytes := ciphertextBytes[:nonceSize], ciphertextBytes[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertextBytes, []byte(keyID))
	if err != nil {
		return nil, err
	}
	
	return plaintext, nil
}

func (m *DynamicKeyManager) ExportKeysJSON() (string, error) {
	keys := m.GetAllKeys()
	type exportKey struct {
		KeyID         string            `json:"key_id"`
		Version       int               `json:"version"`
		CreatedAt     time.Time         `json:"created_at"`
		ExpiresAt     time.Time         `json:"expires_at"`
		LastUsedAt    time.Time         `json:"last_used_at"`
		RotationCount int               `json:"rotation_count"`
		Metadata      map[string]string `json:"metadata"`
	}
	
	exportData := make([]exportKey, len(keys))
	for i, k := range keys {
		exportData[i] = exportKey{
			KeyID:         k.KeyID,
			Version:       k.Version,
			CreatedAt:     k.CreatedAt,
			ExpiresAt:     k.ExpiresAt,
			LastUsedAt:    k.LastUsedAt,
			RotationCount: k.RotationCount,
			Metadata:      k.Metadata,
		}
	}
	
	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", err
	}
	
	return string(data), nil
}

type KeyVersion struct {
	KeyID        string    `json:"key_id"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	ExpiresAt    time.Time `json:"expires_at"`
	IsActive     bool      `json:"is_active"`
	CanDecrypt   bool      `json:"can_decrypt"`
}

func (m *DynamicKeyManager) GetVersionHistory() []KeyVersion {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	versions := make([]KeyVersion, 0, len(m.keys))
	for id, key := range m.keys {
		versions = append(versions, KeyVersion{
			KeyID:     id,
			Version:   key.Version,
			CreatedAt: key.CreatedAt,
			ExpiresAt: key.ExpiresAt,
			IsActive:  id == m.currentKeyID,
			CanDecrypt: true,
		})
	}
	
	return versions
}

type KeyRotationPolicy struct {
	Interval       time.Duration `json:"interval"`
	MaxKeys        int           `json:"max_keys"`
	NotifyBefore   time.Duration `json:"notify_before"`
	AutoPurge     bool          `json:"auto_purge"`
	PurgeAfter    time.Duration `json:"purge_after"`
	EnableMetrics bool          `json:"enable_metrics"`
}

func (m *DynamicKeyManager) GetPolicy() KeyRotationPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	return KeyRotationPolicy{
		Interval:       m.rotationInterval,
		MaxKeys:        m.maxKeys,
		NotifyBefore:   m.rotationInterval / 4,
		AutoPurge:     true,
		PurgeAfter:    m.rotationInterval * 2,
		EnableMetrics: true,
	}
}

func (m *DynamicKeyManager) UpdatePolicy(policy KeyRotationPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.rotationInterval = policy.Interval
	m.maxKeys = policy.MaxKeys
}

type KeyMetrics struct {
	TotalRotations    int           `json:"total_rotations"`
	LastRotationTime time.Time     `json:"last_rotation_time"`
	ActiveKeysCount   int           `json:"active_keys_count"`
	AvgKeyLifetime    time.Duration `json:"avg_key_lifetime"`
	RotationFrequency time.Duration `json:"rotation_frequency"`
}

func (m *DynamicKeyManager) GetMetrics() KeyMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	var totalRotations int
	var oldestTime time.Time
	
	for _, key := range m.keys {
		totalRotations += key.RotationCount
		if oldestTime.IsZero() || key.CreatedAt.Before(oldestTime) {
			oldestTime = key.CreatedAt
		}
	}
	
	avgLifetime := time.Duration(0)
	if len(m.keys) > 1 && !oldestTime.IsZero() {
		avgLifetime = time.Since(oldestTime) / time.Duration(len(m.keys))
	}
	
	return KeyMetrics{
		TotalRotations:    totalRotations,
		LastRotationTime:  m.lastRotation,
		ActiveKeysCount:   len(m.keys),
		AvgKeyLifetime:    avgLifetime,
		RotationFrequency: m.rotationInterval,
	}
}

type KeyAuditLog struct {
	Timestamp   time.Time `json:"timestamp"`
	Action      string    `json:"action"`
	KeyID       string    `json:"key_id"`
	Version     int       `json:"version"`
	Details     string    `json:"details"`
	Success     bool      `json:"success"`
}

func (m *DynamicKeyManager) CreateAuditLog(action, details string, success bool) KeyAuditLog {
	m.mu.RLock()
	currentKey := m.keys[m.currentKeyID]
	m.mu.RUnlock()
	
	return KeyAuditLog{
		Timestamp: time.Now(),
		Action:    action,
		KeyID:     m.currentKeyID,
		Version:   currentKey.Version,
		Details:   details,
		Success:   success,
	}
}

type KeyBackup struct {
	KeyID      string    `json:"key_id"`
	KeyData    []byte    `json:"key_data"`
	Version    int       `json:"version"`
	CreatedAt  time.Time `json:"created_at"`
	Checksum   string    `json:"checksum"`
}

func (m *DynamicKeyManager) CreateBackup() (*KeyBackup, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if key, exists := m.keys[m.currentKeyID]; exists {
		checksum := HashSHA256(key.Key)
		return &KeyBackup{
			KeyID:     key.KeyID,
			KeyData:   key.Key,
			Version:   key.Version,
			CreatedAt: time.Now(),
			Checksum:  checksum,
		}, nil
	}
	
	return nil, fmt.Errorf("no current key available")
}

func (m *DynamicKeyManager) RestoreFromBackup(backup *KeyBackup) error {
	if backup == nil || len(backup.KeyData) != 32 {
		return fmt.Errorf("invalid backup data")
	}
	
	checksum := HashSHA256(backup.KeyData)
	if checksum != backup.Checksum {
		return fmt.Errorf("checksum mismatch")
	}
	
	m.mu.Lock()
	defer m.mu.Unlock()
	
	m.keys[backup.KeyID] = &DynamicKey{
		KeyID:       backup.KeyID,
		Key:         backup.KeyData,
		KeyBase64:   base64.StdEncoding.EncodeToString(backup.KeyData),
		Version:     backup.Version,
		CreatedAt:   time.Now(),
		ExpiresAt:   time.Now().Add(m.rotationInterval),
		LastUsedAt:  time.Now(),
		RotationCount: 0,
		Metadata:   make(map[string]string),
	}
	
	return nil
}

func (m *DynamicKeyManager) ValidateKey(keyID string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if key, exists := m.keys[keyID]; exists {
		return time.Now().Before(key.ExpiresAt)
	}
	return false
}

func (m *DynamicKeyManager) MarkKeyUsed(keyID string) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	if key, exists := m.keys[keyID]; exists {
		key.LastUsedAt = time.Now()
	}
}

type RedisKeyStore interface {
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Get(ctx context.Context, key string) (string, error)
	Delete(ctx context.Context, keys ...string) error
}

type CachedKeyManager struct {
	*DynamicKeyManager
	redisClient RedisKeyStore
	cachePrefix string
}

func NewCachedKeyManager(redisClient RedisKeyStore, rotationInterval time.Duration, maxKeys int) (*CachedKeyManager, error) {
	base, err := NewDynamicKeyManager(rotationInterval, maxKeys)
	if err != nil {
		return nil, err
	}
	
	return &CachedKeyManager{
		DynamicKeyManager: base,
		redisClient:      redisClient,
		cachePrefix:      "key_rotation:",
	}, nil
}

func (c *CachedKeyManager) CacheCurrentKey(ctx context.Context) error {
	key, keyID := c.GetCurrentKey()
	if key == nil {
		return fmt.Errorf("no current key")
	}
	
	cacheKey := c.cachePrefix + "current"
	return c.redisClient.Set(ctx, cacheKey, keyID, c.rotationInterval)
}

func (c *CachedKeyManager) GetCachedKeyID(ctx context.Context) (string, error) {
	cacheKey := c.cachePrefix + "current"
	return c.redisClient.Get(ctx, cacheKey)
}

func (c *CachedKeyManager) InvalidateCache(ctx context.Context) error {
	return c.redisClient.Delete(ctx, c.cachePrefix+"current")
}
