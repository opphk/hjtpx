package service

import (
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/crypto"
)

var (
	ErrKeyNotFound       = errors.New("key not found")
	ErrKeyExpired        = errors.New("key expired")
	ErrKeyRotationFailed = errors.New("key rotation failed")
	ErrInvalidKeyID      = errors.New("invalid key ID")
)

type PQKeyType string

const (
	PQKeyTypeKyber     PQKeyType = "kyber"
	PQKeyTypeDilithium PQKeyType = "dilithium"
)

type KeyStatus string

const (
	KeyStatusActive   KeyStatus = "active"
	KeyStatusInactive KeyStatus = "inactive"
	KeyStatusExpired  KeyStatus = "expired"
	KeyStatusRevoked  KeyStatus = "revoked"
)

type StoredKey struct {
	ID          string      `json:"id"`
	Type        PQKeyType   `json:"type"`
	Algorithm   crypto.PQAlgorithm `json:"algorithm"`
	PublicKey   string      `json:"public_key"`
	PrivateKey  string      `json:"private_key,omitempty"`
	Status      KeyStatus   `json:"status"`
	CreatedAt   time.Time   `json:"created_at"`
	ExpiresAt   time.Time   `json:"expires_at"`
	RotatedAt   *time.Time  `json:"rotated_at,omitempty"`
	Version     int         `json:"version"`
	Metadata    map[string]string `json:"metadata,omitempty"`
}

type KeyRotationPolicy struct {
	AutoRotate     bool          `json:"auto_rotate"`
	RotationPeriod time.Duration `json:"rotation_period"`
	RetentionCount int           `json:"retention_count"`
}

type KeyManagementSystem struct {
	mu            sync.RWMutex
	keys          map[string]*StoredKey
	activeKeyIDs  map[PQKeyType]string
	policies      map[PQKeyType]*KeyRotationPolicy
	pqCrypto      *crypto.PostQuantumCryptoV2
	nextVersion   int
}

func NewKeyManagementSystem() *KeyManagementSystem {
	return &KeyManagementSystem{
		keys:         make(map[string]*StoredKey),
		activeKeyIDs: make(map[PQKeyType]string),
		policies: map[PQKeyType]*KeyRotationPolicy{
			PQKeyTypeKyber: {
				AutoRotate:     true,
				RotationPeriod: 30 * 24 * time.Hour,
				RetentionCount: 5,
			},
			PQKeyTypeDilithium: {
				AutoRotate:     true,
				RotationPeriod: 30 * 24 * time.Hour,
				RetentionCount: 5,
			},
		},
		pqCrypto:    crypto.NewPostQuantumCryptoV2(),
		nextVersion: 1,
	}
}

func (kms *KeyManagementSystem) generateKeyID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 16)
	crypto.GenerateRandomBytes(len(randomBytes))
	hash := sha256.Sum256(append(randomBytes, byte(timestamp)))
	return base64.URLEncoding.EncodeToString(hash[:])
}

func (kms *KeyManagementSystem) GenerateKyberKey(algorithm crypto.PQAlgorithm, metadata map[string]string) (*StoredKey, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	keyPair, err := kms.pqCrypto.GenerateKyberKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	pubKeyStr, err := kms.pqCrypto.SerializeKyberPublicKey(keyPair)
	if err != nil {
		return nil, err
	}

	privKeyStr, err := kms.pqCrypto.SerializeKyberPrivateKey(keyPair)
	if err != nil {
		return nil, err
	}

	keyID := kms.generateKeyID()
	now := time.Now()
	expiresAt := now.Add(90 * 24 * time.Hour)

	storedKey := &StoredKey{
		ID:         keyID,
		Type:       PQKeyTypeKyber,
		Algorithm:  algorithm,
		PublicKey:  pubKeyStr,
		PrivateKey: privKeyStr,
		Status:     KeyStatusActive,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		Version:    kms.nextVersion,
		Metadata:   metadata,
	}

	kms.nextVersion++
	kms.keys[keyID] = storedKey
	kms.activeKeyIDs[PQKeyTypeKyber] = keyID

	return storedKey, nil
}

func (kms *KeyManagementSystem) GenerateDilithiumKey(algorithm crypto.PQAlgorithm, metadata map[string]string) (*StoredKey, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	keyPair, err := kms.pqCrypto.GenerateDilithiumKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	pubKeyStr, err := kms.pqCrypto.SerializeDilithiumPublicKey(keyPair)
	if err != nil {
		return nil, err
	}

	privKeyStr, err := kms.pqCrypto.SerializeDilithiumPrivateKey(keyPair)
	if err != nil {
		return nil, err
	}

	keyID := kms.generateKeyID()
	now := time.Now()
	expiresAt := now.Add(90 * 24 * time.Hour)

	storedKey := &StoredKey{
		ID:         keyID,
		Type:       PQKeyTypeDilithium,
		Algorithm:  algorithm,
		PublicKey:  pubKeyStr,
		PrivateKey: privKeyStr,
		Status:     KeyStatusActive,
		CreatedAt:  now,
		ExpiresAt:  expiresAt,
		Version:    kms.nextVersion,
		Metadata:   metadata,
	}

	kms.nextVersion++
	kms.keys[keyID] = storedKey
	kms.activeKeyIDs[PQKeyTypeDilithium] = keyID

	return storedKey, nil
}

func (kms *KeyManagementSystem) GetKey(keyID string) (*StoredKey, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	key, exists := kms.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	if key.Status == KeyStatusExpired || time.Now().After(key.ExpiresAt) {
		return nil, ErrKeyExpired
	}

	return key, nil
}

func (kms *KeyManagementSystem) GetActiveKey(keyType PQKeyType) (*StoredKey, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	keyID, exists := kms.activeKeyIDs[keyType]
	if !exists {
		return nil, ErrKeyNotFound
	}

	key, exists := kms.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	if key.Status == KeyStatusExpired || time.Now().After(key.ExpiresAt) {
		return nil, ErrKeyExpired
	}

	return key, nil
}

func (kms *KeyManagementSystem) RotateKey(keyID string) (*StoredKey, error) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	oldKey, exists := kms.keys[keyID]
	if !exists {
		return nil, ErrKeyNotFound
	}

	oldKey.Status = KeyStatusInactive
	now := time.Now()
	oldKey.RotatedAt = &now

	var newKey *StoredKey

	switch oldKey.Type {
	case PQKeyTypeKyber:
		keyPair, err := kms.pqCrypto.GenerateKyberKeyPair(oldKey.Algorithm)
		if err != nil {
			return nil, err
		}

		pubKeyStr, err := kms.pqCrypto.SerializeKyberPublicKey(keyPair)
		if err != nil {
			return nil, err
		}

		privKeyStr, err := kms.pqCrypto.SerializeKyberPrivateKey(keyPair)
		if err != nil {
			return nil, err
		}

		newKeyID := kms.generateKeyID()
		expiresAt := now.Add(90 * 24 * time.Hour)

		newKey = &StoredKey{
			ID:         newKeyID,
			Type:       PQKeyTypeKyber,
			Algorithm:  oldKey.Algorithm,
			PublicKey:  pubKeyStr,
			PrivateKey: privKeyStr,
			Status:     KeyStatusActive,
			CreatedAt:  now,
			ExpiresAt:  expiresAt,
			Version:    kms.nextVersion,
			Metadata:   oldKey.Metadata,
		}

		kms.activeKeyIDs[PQKeyTypeKyber] = newKeyID

	case PQKeyTypeDilithium:
		keyPair, err := kms.pqCrypto.GenerateDilithiumKeyPair(oldKey.Algorithm)
		if err != nil {
			return nil, err
		}

		pubKeyStr, err := kms.pqCrypto.SerializeDilithiumPublicKey(keyPair)
		if err != nil {
			return nil, err
		}

		privKeyStr, err := kms.pqCrypto.SerializeDilithiumPrivateKey(keyPair)
		if err != nil {
			return nil, err
		}

		newKeyID := kms.generateKeyID()
		expiresAt := now.Add(90 * 24 * time.Hour)

		newKey = &StoredKey{
			ID:         newKeyID,
			Type:       PQKeyTypeDilithium,
			Algorithm:  oldKey.Algorithm,
			PublicKey:  pubKeyStr,
			PrivateKey: privKeyStr,
			Status:     KeyStatusActive,
			CreatedAt:  now,
			ExpiresAt:  expiresAt,
			Version:    kms.nextVersion,
			Metadata:   oldKey.Metadata,
		}

		kms.activeKeyIDs[PQKeyTypeDilithium] = newKeyID

	default:
		return nil, fmt.Errorf("%w: unknown key type", ErrKeyRotationFailed)
	}

	kms.nextVersion++
	kms.keys[newKey.ID] = newKey

	kms.cleanupOldKeys(oldKey.Type)

	return newKey, nil
}

func (kms *KeyManagementSystem) cleanupOldKeys(keyType PQKeyType) {
	policy := kms.policies[keyType]
	if policy == nil {
		return
	}

	var keysToDelete []string
	activeKeyID := kms.activeKeyIDs[keyType]

	keyCount := 0
	for _, key := range kms.keys {
		if key.Type == keyType {
			keyCount++
		}
	}

	for _, key := range kms.keys {
		if key.Type == keyType && key.ID != activeKeyID && key.Status != KeyStatusActive && keyCount > policy.RetentionCount {
			keysToDelete = append(keysToDelete, key.ID)
			keyCount--
		}
	}

	for _, keyID := range keysToDelete {
		delete(kms.keys, keyID)
	}
}

func (kms *KeyManagementSystem) RevokeKey(keyID string) error {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	key, exists := kms.keys[keyID]
	if !exists {
		return ErrKeyNotFound
	}

	key.Status = KeyStatusRevoked
	return nil
}

func (kms *KeyManagementSystem) ListKeys(keyType PQKeyType, status KeyStatus) ([]*StoredKey, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	var result []*StoredKey
	for _, key := range kms.keys {
		if keyType != "" && key.Type != keyType {
			continue
		}
		if status != "" && key.Status != status {
			continue
		}
		result = append(result, key)
	}

	return result, nil
}

func (kms *KeyManagementSystem) SetRotationPolicy(keyType PQKeyType, policy *KeyRotationPolicy) {
	kms.mu.Lock()
	defer kms.mu.Unlock()

	kms.policies[keyType] = policy
}

func (kms *KeyManagementSystem) GetRotationPolicy(keyType PQKeyType) (*KeyRotationPolicy, error) {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	policy, exists := kms.policies[keyType]
	if !exists {
		return nil, errors.New("policy not found")
	}

	return policy, nil
}

func (kms *KeyManagementSystem) EncryptWithActiveKyberKey(plaintext []byte) ([]byte, string, error) {
	key, err := kms.GetActiveKey(PQKeyTypeKyber)
	if err != nil {
		return nil, "", err
	}

	pubKey, err := kms.pqCrypto.DeserializeKyberPublicKey(key.PublicKey)
	if err != nil {
		return nil, "", err
	}

	ciphertext, err := kms.pqCrypto.EncryptWithKyber(plaintext, pubKey.PublicKey, key.Algorithm)
	if err != nil {
		return nil, "", err
	}

	return ciphertext, key.ID, nil
}

func (kms *KeyManagementSystem) DecryptWithKey(ciphertext []byte, keyID string) ([]byte, error) {
	key, err := kms.GetKey(keyID)
	if err != nil {
		return nil, err
	}

	privKey, err := kms.pqCrypto.DeserializeKyberPrivateKey(key.PrivateKey)
	if err != nil {
		return nil, err
	}

	plaintext, err := kms.pqCrypto.DecryptWithKyber(ciphertext, privKey.PrivateKey, key.Algorithm)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (kms *KeyManagementSystem) SignWithActiveDilithiumKey(message []byte) ([]byte, string, error) {
	key, err := kms.GetActiveKey(PQKeyTypeDilithium)
	if err != nil {
		return nil, "", err
	}

	privKey, err := kms.pqCrypto.DeserializeDilithiumPrivateKey(key.PrivateKey)
	if err != nil {
		return nil, "", err
	}

	signature, err := kms.pqCrypto.DilithiumSign(message, privKey.PrivateKey, key.Algorithm)
	if err != nil {
		return nil, "", err
	}

	return signature.Data, key.ID, nil
}

func (kms *KeyManagementSystem) VerifyWithKey(message []byte, signature []byte, keyID string) (bool, error) {
	key, err := kms.GetKey(keyID)
	if err != nil {
		return false, err
	}

	pubKey, err := kms.pqCrypto.DeserializeDilithiumPublicKey(key.PublicKey)
	if err != nil {
		return false, err
	}

	valid, err := kms.pqCrypto.DilithiumVerify(message, signature, pubKey.PublicKey, key.Algorithm)
	if err != nil {
		return false, err
	}

	return valid, nil
}

func (kms *KeyManagementSystem) ExportKey(keyID string) (string, error) {
	key, err := kms.GetKey(keyID)
	if err != nil {
		return "", err
	}

	data, err := json.Marshal(key)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func (kms *KeyManagementSystem) ImportKey(exportedKey string) (*StoredKey, error) {
	data, err := base64.StdEncoding.DecodeString(exportedKey)
	if err != nil {
		return nil, err
	}

	var key StoredKey
	if err := json.Unmarshal(data, &key); err != nil {
		return nil, err
	}

	kms.mu.Lock()
	defer kms.mu.Unlock()

	kms.keys[key.ID] = &key

	if key.Status == KeyStatusActive {
		kms.activeKeyIDs[key.Type] = key.ID
	}

	if key.Version >= kms.nextVersion {
		kms.nextVersion = key.Version + 1
	}

	return &key, nil
}

func (kms *KeyManagementSystem) CheckExpiredKeys() []*StoredKey {
	kms.mu.RLock()
	defer kms.mu.RUnlock()

	var expiredKeys []*StoredKey
	now := time.Now()

	for _, key := range kms.keys {
		if key.Status == KeyStatusActive && now.After(key.ExpiresAt) {
			expiredKeys = append(expiredKeys, key)
		}
	}

	return expiredKeys
}

func (kms *KeyManagementSystem) AutoRotateExpiredKeys() ([]*StoredKey, error) {
	expiredKeys := kms.CheckExpiredKeys()
	var rotatedKeys []*StoredKey

	for _, key := range expiredKeys {
		policy := kms.policies[key.Type]
		if policy != nil && policy.AutoRotate {
			newKey, err := kms.RotateKey(key.ID)
			if err != nil {
				return nil, err
			}
			rotatedKeys = append(rotatedKeys, newKey)
		}
	}

	return rotatedKeys, nil
}
