package service

import (
	"context"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrKeyNotFound          = errors.New("key not found")
	ErrKeyExpired          = errors.New("key expired")
	ErrKeyRevoked          = errors.New("key revoked")
	ErrInvalidKeyType      = errors.New("invalid key type")
	ErrKeyInUse            = errors.New("key is in use")
	ErrKeyGenerationFailed = errors.New("key generation failed")
	ErrKeyRotationFailed   = errors.New("key rotation failed")
	ErrQuotaExceeded       = errors.New("key quota exceeded")
)

type KeyType string

const (
	KeyTypeAES256      KeyType = "aes-256-gcm"
	KeyTypeChaCha20    KeyType = "chacha20-poly1305"
	KeyTypeKyber512    KeyType = "kyber-512"
	KeyTypeKyber768    KeyType = "kyber-768"
	KeyTypeDilithium2  KeyType = "dilithium-2"
	KeyTypeDilithium3  KeyType = "dilithium-3"
)

type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusExpired  KeyStatus = "expired"
	KeyStatusRevoked  KeyStatus = "revoked"
	KeyStatusPending  KeyStatus = "pending"
)

type KeyMetadata struct {
	KeyID        string            `json:"key_id"`
	KeyType      KeyType           `json:"key_type"`
	Status       KeyStatus         `json:"status"`
	CreatedAt    time.Time         `json:"created_at"`
	ExpiresAt    *time.Time        `json:"expires_at,omitempty"`
	RotatedAt    *time.Time        `json:"rotated_at,omitempty"`
	RevokedAt    *time.Time        `json:"revoked_at,omitempty"`
	RevokedBy    string            `json:"revoked_by,omitempty"`
	UseCount     int64             `json:"use_count"`
	LastUsedAt   *time.Time        `json:"last_used_at,omitempty"`
	Labels       map[string]string `json:"labels,omitempty"`
	Version      int               `json:"version"`
	Algorithm    string            `json:"algorithm"`
	Purpose      string            `json:"purpose"`
	OwnerID      string            `json:"owner_id"`
}

type KeyMaterial struct {
	KeyID        string    `json:"key_id"`
	PublicKey    []byte    `json:"public_key,omitempty"`
	PrivateKey   []byte    `json:"private_key,omitempty"`
	EncryptedKey []byte    `json:"encrypted_key,omitempty"`
	IV           []byte    `json:"iv,omitempty"`
	WrappedKey   []byte    `json:"wrapped_key,omitempty"`
	CreatedAt    time.Time `json:"created_at"`
}

type KeyRotationPolicy struct {
	PolicyID         string        `json:"policy_id"`
	KeyType          KeyType       `json:"key_type"`
	RotationInterval time.Duration `json:"rotation_interval"`
	MaxKeyVersions   int           `json:"max_key_versions"`
	AutoRotate       bool          `json:"auto_rotate"`
	WarningThreshold time.Duration `json:"warning_threshold"`
}

type KeyUsageLog struct {
	KeyID        string    `json:"key_id"`
	Operation    string    `json:"operation"`
	UserID       string    `json:"user_id"`
	IPAddress    string    `json:"ip_address"`
	Success      bool      `json:"success"`
	Timestamp    time.Time `json:"timestamp"`
	Duration     time.Duration `json:"duration"`
}

type KeyAuditLog struct {
	KeyID        string    `json:"key_id"`
	Action       string    `json:"action"`
	ActorID      string    `json:"actor_id"`
	Details      string    `json:"details"`
	Timestamp    time.Time `json:"timestamp"`
	IPAddress    string    `json:"ip_address"`
	Result       string    `json:"result"`
}

type KeyGenerationRequest struct {
	KeyType    KeyType           `json:"key_type"`
	Purpose    string            `json:"purpose"`
	OwnerID    string            `json:"owner_id"`
	Labels     map[string]string `json:"labels,omitempty"`
	ExpiresAt  *time.Time        `json:"expires_at,omitempty"`
	Algorithm  string            `json:"algorithm,omitempty"`
}

type KeyGenerationResponse struct {
	KeyID      string        `json:"key_id"`
	Metadata   *KeyMetadata  `json:"metadata"`
	Material   *KeyMaterial  `json:"material,omitempty"`
	PublicKey  []byte        `json:"public_key,omitempty"`
	PrivateKey []byte        `json:"private_key,omitempty"`
}

type KeyRotationRequest struct {
	KeyID         string `json:"key_id"`
	RotationType  string `json:"rotation_type"`
	Reason        string `json:"reason,omitempty"`
	ActorID       string `json:"actor_id"`
}

type KeyRotationResponse struct {
	Success         bool           `json:"success"`
	OldKeyID        string         `json:"old_key_id,omitempty"`
	NewKeyID        string         `json:"new_key_id,omitempty"`
	OldKeyStatus    KeyStatus      `json:"old_key_status,omitempty"`
	KeysRetired     int            `json:"keys_retired,omitempty"`
	RotationDetails string         `json:"rotation_details,omitempty"`
}

type KeyRevocationRequest struct {
	KeyID      string `json:"key_id"`
	Reason     string `json:"reason"`
	RevokedBy  string `json:"revoked_by"`
}

type KeyRevocationResponse struct {
	Success    bool       `json:"success"`
	KeyID      string     `json:"key_id"`
	Status     KeyStatus  `json:"status"`
	RevokedAt  time.Time  `json:"revoked_at"`
}

type KeyRetrievalRequest struct {
	KeyID        string    `json:"key_id"`
	DecryptKey   bool      `json:"decrypt_key"`
	AuthToken    string    `json:"auth_token,omitempty"`
}

type KeyRetrievalResponse struct {
	KeyID      string        `json:"key_id"`
	Metadata   *KeyMetadata  `json:"metadata"`
	PublicKey  []byte        `json:"public_key,omitempty"`
	PrivateKey []byte        `json:"private_key,omitempty"`
}

type KeyListRequest struct {
	OwnerID    string     `json:"owner_id"`
	KeyType    KeyType    `json:"key_type,omitempty"`
	Status     KeyStatus  `json:"status,omitempty"`
	Page       int        `json:"page"`
	PageSize   int        `json:"page_size"`
	SortBy     string     `json:"sort_by"`
	SortOrder  string     `json:"sort_order"`
}

type KeyListResponse struct {
	Keys       []*KeyMetadata `json:"keys"`
	TotalCount int            `json:"total_count"`
	Page       int            `json:"page"`
	PageSize   int            `json:"page_size"`
}

type KeyManagementSystem struct {
	mu               sync.RWMutex
	keys             map[string]*KeyMetadata
	keyMaterials     map[string]*KeyMaterial
	rotationPolicies map[string]*KeyRotationPolicy
	usageLogs        map[string][]*KeyUsageLog
	auditLogs        map[string][]*KeyAuditLog
	maxKeysPerOwner  int
	rotationScheduler *RotationScheduler
}

type RotationScheduler struct {
	mu      sync.RWMutex
	jobs    map[string]*RotationJob
	enabled bool
}

type RotationJob struct {
	JobID      string
	KeyID      string
	Schedule   time.Time
	Status     string
	Result     string
	LastRun    *time.Time
	NextRun    *time.Time
}

func NewKeyManagementSystem() *KeyManagementSystem {
	return &KeyManagementSystem{
		keys:             make(map[string]*KeyMetadata),
		keyMaterials:     make(map[string]*KeyMaterial),
		rotationPolicies: make(map[string]*KeyRotationPolicy),
		usageLogs:        make(map[string][]*KeyUsageLog),
		auditLogs:        make(map[string][]*KeyAuditLog),
		maxKeysPerOwner:  100,
		rotationScheduler: &RotationScheduler{
			jobs:    make(map[string]*RotationJob),
			enabled: true,
		},
	}
}

func (kms *KeyManagementSystem) GenerateKey(ctx context.Context, req *KeyGenerationRequest) (*KeyGenerationResponse, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	if kms.countKeysByOwner(req.OwnerID) >= kms.maxKeysPerOwner {
		return nil, ErrQuotaExceeded
	}

	keyID := generateKeyID()
	now := time.Now()

	metadata := &KeyMetadata{
		KeyID:     keyID,
		KeyType:   req.KeyType,
		Status:    KeyStatusActive,
		CreatedAt: now,
		ExpiresAt: req.ExpiresAt,
		UseCount:  0,
		Labels:    req.Labels,
		Version:   1,
		Algorithm: req.Algorithm,
		Purpose:   req.Purpose,
		OwnerID:   req.OwnerID,
	}

	material := &KeyMaterial{
		KeyID:     keyID,
		CreatedAt: now,
	}

	switch req.KeyType {
	case KeyTypeAES256:
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		material.EncryptedKey = key
		metadata.Algorithm = "AES-256-GCM"

	case KeyTypeChaCha20:
		key := make([]byte, 32)
		if _, err := rand.Read(key); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		material.EncryptedKey = key
		metadata.Algorithm = "ChaCha20-Poly1305"

	case KeyTypeKyber512:
		pk := make([]byte, 800)
		sk := make([]byte, 1632)
		if _, err := rand.Read(pk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		if _, err := rand.Read(sk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		material.PublicKey = pk
		material.PrivateKey = sk
		metadata.Algorithm = "Kyber-512"

	case KeyTypeKyber768:
		pk := make([]byte, 1184)
		sk := make([]byte, 2400)
		if _, err := rand.Read(pk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		if _, err := rand.Read(sk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		material.PublicKey = pk
		material.PrivateKey = sk
		metadata.Algorithm = "Kyber-768"

	case KeyTypeDilithium2:
		pk := make([]byte, 1312)
		sk := make([]byte, 2528)
		if _, err := rand.Read(pk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		if _, err := rand.Read(sk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		material.PublicKey = pk
		material.PrivateKey = sk
		metadata.Algorithm = "Dilithium-2"

	case KeyTypeDilithium3:
		pk := make([]byte, 1952)
		sk := make([]byte, 4000)
		if _, err := rand.Read(pk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		if _, err := rand.Read(sk); err != nil {
			return nil, fmt.Errorf("%w: %v", ErrKeyGenerationFailed, err)
		}
		material.PublicKey = pk
		material.PrivateKey = sk
		metadata.Algorithm = "Dilithium-3"

	default:
		return nil, ErrInvalidKeyType
	}

	kms.keys[keyID] = metadata
	kms.keyMaterials[keyID] = material

	kms.logAudit(keyID, "key_generated", req.OwnerID, "Key generated successfully", "127.0.0.1", "success")

	return &KeyGenerationResponse{
		KeyID:      keyID,
		Metadata:   metadata,
		Material:   material,
		PublicKey:  material.PublicKey,
		PrivateKey: material.PrivateKey,
	}, nil
}

func (kms *KeyManagementSystem) GetKey(ctx context.Context, keyID string) (*KeyRetrievalResponse, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	metadata, exists := kms.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	if metadata.Status == KeyStatusExpired {
		return nil, ErrKeyExpired
	}

	if metadata.Status == KeyStatusRevoked {
		return nil, ErrKeyRevoked
	}

	material, exists := kms.keyMaterials[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	now := time.Now()
	metadata.UseCount++
	metadata.LastUsedAt = &now

	return &KeyRetrievalResponse{
		KeyID:     keyID,
		Metadata:  metadata,
		PublicKey: material.PublicKey,
		PrivateKey: material.PrivateKey,
	}, nil
}

func (kms *KeyManagementSystem) ListKeys(ctx context.Context, req *KeyListRequest) (*KeyListResponse, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	keys := make([]*KeyMetadata, 0)
	for _, key := range kms.keys {
		if key.OwnerID != req.OwnerID {
			continue
		}
		if req.KeyType != "" && key.KeyType != req.KeyType {
			continue
		}
		if req.Status != "" && key.Status != req.Status {
			continue
		}
		keys = append(keys, key)
	}

	totalCount := len(keys)

	start := (req.Page - 1) * req.PageSize
	end := start + req.PageSize
	if start > len(keys) {
		start = len(keys)
	}
	if end > len(keys) {
		end = len(keys)
	}

	return &KeyListResponse{
		Keys:       keys[start:end],
		TotalCount: totalCount,
		Page:       req.Page,
		PageSize:   req.PageSize,
	}, nil
}

func (kms *KeyManagementSystem) RotateKey(ctx context.Context, req *KeyRotationRequest) (*KeyRotationResponse, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	oldKey, exists := kms.keys[req.KeyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	now := time.Now()
	rotatedAt := now
	oldKey.RotatedAt = &rotatedAt

	oldKeyID := req.KeyID

	newMetadata := &KeyMetadata{
		KeyID:     generateKeyID(),
		KeyType:   oldKey.KeyType,
		Status:    KeyStatusActive,
		CreatedAt: now,
		ExpiresAt: oldKey.ExpiresAt,
		UseCount:  0,
		Labels:    oldKey.Labels,
		Version:   oldKey.Version + 1,
		Algorithm: oldKey.Algorithm,
		Purpose:   oldKey.Purpose,
		OwnerID:   oldKey.OwnerID,
	}

	var newMaterial *KeyMaterial
	switch oldKey.KeyType {
	case KeyTypeAES256:
		key := make([]byte, 32)
		rand.Read(key)
		newMaterial = &KeyMaterial{KeyID: newMetadata.KeyID, EncryptedKey: key, CreatedAt: now}
	case KeyTypeChaCha20:
		key := make([]byte, 32)
		rand.Read(key)
		newMaterial = &KeyMaterial{KeyID: newMetadata.KeyID, EncryptedKey: key, CreatedAt: now}
	case KeyTypeKyber512:
		pk := make([]byte, 800)
		sk := make([]byte, 1632)
		rand.Read(pk)
		rand.Read(sk)
		newMaterial = &KeyMaterial{KeyID: newMetadata.KeyID, PublicKey: pk, PrivateKey: sk, CreatedAt: now}
	case KeyTypeKyber768:
		pk := make([]byte, 1184)
		sk := make([]byte, 2400)
		rand.Read(pk)
		rand.Read(sk)
		newMaterial = &KeyMaterial{KeyID: newMetadata.KeyID, PublicKey: pk, PrivateKey: sk, CreatedAt: now}
	case KeyTypeDilithium2:
		pk := make([]byte, 1312)
		sk := make([]byte, 2528)
		rand.Read(pk)
		rand.Read(sk)
		newMaterial = &KeyMaterial{KeyID: newMetadata.KeyID, PublicKey: pk, PrivateKey: sk, CreatedAt: now}
	case KeyTypeDilithium3:
		pk := make([]byte, 1952)
		sk := make([]byte, 4000)
		rand.Read(pk)
		rand.Read(sk)
		newMaterial = &KeyMaterial{KeyID: newMetadata.KeyID, PublicKey: pk, PrivateKey: sk, CreatedAt: now}
	}

	kms.keys[newMetadata.KeyID] = newMetadata
	kms.keyMaterials[newMetadata.KeyID] = newMaterial

	keysRetired := 0
	if policy, exists := kms.rotationPolicies[string(oldKey.KeyType)]; exists {
		oldVersion := newMetadata.Version - policy.MaxKeyVersions
		for keyID, key := range kms.keys {
			if key.OwnerID == oldKey.OwnerID && key.KeyType == oldKey.KeyType && key.Version < oldVersion {
				key.Status = KeyStatusRevoked
				revokedAt := now
				key.RevokedAt = &revokedAt
				keysRetired++
			}
		}
		_ = policy
	}

	kms.logAudit(oldKeyID, "key_rotated", req.ActorID, req.Reason, "127.0.0.1", "success")
	kms.logAudit(newMetadata.KeyID, "key_created", req.ActorID, "Rotation new key", "127.0.0.1", "success")

	return &KeyRotationResponse{
		Success:         true,
		OldKeyID:        oldKeyID,
		NewKeyID:        newMetadata.KeyID,
		OldKeyStatus:    oldKey.Status,
		KeysRetired:     keysRetired,
		RotationDetails: fmt.Sprintf("Key rotated from %s to %s", oldKeyID, newMetadata.KeyID),
	}, nil
}

func (kms *KeyManagementSystem) RevokeKey(ctx context.Context, req *KeyRevocationRequest) (*KeyRevocationResponse, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	key, exists := kms.keys[req.KeyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	now := time.Now()
	key.Status = KeyStatusRevoked
	key.RevokedAt = &now
	key.RevokedBy = req.RevokedBy

	kms.logAudit(req.KeyID, "key_revoked", req.RevokedBy, req.Reason, "127.0.0.1", "success")

	return &KeyRevocationResponse{
		Success:   true,
		KeyID:     req.KeyID,
		Status:    KeyStatusRevoked,
		RevokedAt: now,
	}, nil
}

func (kms *KeyManagementSystem) SetRotationPolicy(ctx context.Context, policy *KeyRotationPolicy) error {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	policy.PolicyID = generatePolicyID()
	kms.rotationPolicies[policy.PolicyID] = policy

	return nil
}

func (kms *KeyManagementSystem) GetRotationPolicy(ctx context.Context, keyType KeyType) (*KeyRotationPolicy, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	for _, policy := range kms.rotationPolicies {
		if policy.KeyType == keyType {
			return policy, nil
		}
	}

	return nil, nil
}

func (kms *KeyManagementSystem) GetKeyUsageLogs(ctx context.Context, keyID string, limit int) ([]*KeyUsageLog, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	logs, exists := kms.usageLogs[keyID]
	if !exists {
		return []*KeyUsageLog{}, nil
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}

	return logs, nil
}

func (kms *KeyManagementSystem) GetKeyAuditLogs(ctx context.Context, keyID string, limit int) ([]*KeyAuditLog, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	logs, exists := kms.auditLogs[keyID]
	if !exists {
		return []*KeyAuditLog{}, nil
	}

	if limit > 0 && len(logs) > limit {
		return logs[len(logs)-limit:], nil
	}

	return logs, nil
}

func (kms *KeyManagementSystem) ValidateKey(ctx context.Context, keyID string) (bool, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	key, exists := kms.keys[keyID]
	if !exists {
		return false, ErrKeyNotFound
	}

	if key.Status == KeyStatusExpired {
		return false, ErrKeyExpired
	}

	if key.Status == KeyStatusRevoked {
		return false, ErrKeyRevoked
	}

	if key.ExpiresAt != nil && time.Now().After(*key.ExpiresAt) {
		return false, ErrKeyExpired
	}

	return true, nil
}

func (kms *KeyManagementSystem) GetKeyCount(ctx context.Context, ownerID string) int {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	count := 0
	for _, key := range kms.keys {
		if key.OwnerID == ownerID {
			count++
		}
	}
	return count
}

func (kms *KeyManagementSystem) ExportKeys(ctx context.Context, ownerID string, format string) ([]byte, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	keys := make([]*KeyMetadata, 0)
	for _, key := range kms.keys {
		if key.OwnerID == ownerID {
			keys = append(keys, key)
		}
	}

	switch format {
	case "json":
		return json.Marshal(keys)
	case "csv":
		csv := "key_id,key_type,status,created_at,expires_at,version,algorithm,purpose\n"
		for _, key := range keys {
			expiresAt := ""
			if key.ExpiresAt != nil {
				expiresAt = key.ExpiresAt.Format(time.RFC3339)
			}
			csv += fmt.Sprintf("%s,%s,%s,%s,%s,%d,%s,%s\n",
				key.KeyID, key.KeyType, key.Status,
				key.CreatedAt.Format(time.RFC3339), expiresAt,
				key.Version, key.Algorithm, key.Purpose)
		}
		return []byte(csv), nil
	default:
		return nil, errors.New("unsupported format")
	}
}

func (kms *KeyManagementSystem) countKeysByOwner(ownerID string) int {
	count := 0
	for _, key := range kms.keys {
		if key.OwnerID == ownerID {
			count++
		}
	}
	return count
}

func (kms *KeyManagementSystem) logUsage(keyID, operation, userID, ipAddress string, success bool, duration time.Duration) {
	log := &KeyUsageLog{
		KeyID:     keyID,
		Operation: operation,
		UserID:    userID,
		IPAddress: ipAddress,
		Success:   success,
		Timestamp: time.Now(),
		Duration:  duration,
	}

	kms.usageLogs[keyID] = append(kms.usageLogs[keyID], log)

	if len(kms.usageLogs[keyID]) > 10000 {
		kms.usageLogs[keyID] = kms.usageLogs[keyID][len(kms.usageLogs[keyID])-5000:]
	}
}

func (kms *KeyManagementSystem) logAudit(keyID, action, actorID, details, ipAddress, result string) {
	log := &KeyAuditLog{
		KeyID:     keyID,
		Action:    action,
		ActorID:   actorID,
		Details:   details,
		Timestamp: time.Now(),
		IPAddress: ipAddress,
		Result:    result,
	}

	kms.auditLogs[keyID] = append(kms.auditLogs[keyID], log)

	if len(kms.auditLogs[keyID]) > 10000 {
		kms.auditLogs[keyID] = kms.auditLogs[keyID][len(kms.auditLogs[keyID])-5000:]
	}
}

func generateKeyID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

func generatePolicyID() string {
	b := make([]byte, 8)
	rand.Read(b)
	return base64.URLEncoding.EncodeToString(b)
}

type KeyHealthCheckRequest struct {
	KeyID string `json:"key_id"`
}

type KeyHealthCheckResponse struct {
	KeyID        string    `json:"key_id"`
	HealthStatus string    `json:"health_status"`
	Details      string    `json:"details"`
	CheckedAt    time.Time `json:"checked_at"`
}

func (kms *KeyManagementSystem) CheckKeyHealth(ctx context.Context, req *KeyHealthCheckRequest) (*KeyHealthCheckResponse, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	key, exists := kms.keys[req.KeyID]
	if !exists {
		return &KeyHealthCheckResponse{
			KeyID:        req.KeyID,
			HealthStatus: "unknown",
			Details:      "Key not found",
			CheckedAt:    time.Now(),
		}, nil
	}

	status := "healthy"
	details := "Key is operational"

	if key.Status == KeyStatusRevoked {
		status = "revoked"
		details = "Key has been revoked"
	} else if key.Status == KeyStatusExpired {
		status = "expired"
		details = "Key has expired"
	} else if key.ExpiresAt != nil {
		timeUntilExpiry := time.Until(*key.ExpiresAt)
		if timeUntilExpiry < 24*time.Hour {
			status = "warning"
			details = fmt.Sprintf("Key expires in %v", timeUntilExpiry)
		}
	}

	return &KeyHealthCheckResponse{
		KeyID:        req.KeyID,
		HealthStatus: status,
		Details:      details,
		CheckedAt:    time.Now(),
	}, nil
}
