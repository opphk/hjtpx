package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/crypto"
)

const (
	KeyRotationInterval = 24 * time.Hour
	MaxKeyAge           = 7 * 24 * time.Hour
)

type AdvancedCryptoService struct {
	mu                sync.RWMutex
	keys              map[string]*KeyEntry
	keyRotationTicker *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
}

type KeyEntry struct {
	ID         string
	Key        []byte
	CreatedAt  time.Time
	LastUsedAt time.Time
	IsActive   bool
}

type EncryptedPayload struct {
	Version     int       `json:"version"`
	Algorithm   string    `json:"algorithm"`
	Ciphertext  string    `json:"ciphertext"`
	IV          string    `json:"iv"`
	KeyID       string    `json:"key_id"`
	Timestamp   int64     `json:"timestamp"`
	Nonce       string    `json:"nonce"`
}

type KeyGenerationRequest struct {
	Algorithm string `json:"algorithm"`
	Size      int    `json:"size"`
}

type KeyGenerationResponse struct {
	KeyID     string `json:"key_id"`
	PublicKey string `json:"public_key,omitempty"`
	Success   bool   `json:"success"`
}

func NewAdvancedCryptoService() *AdvancedCryptoService {
	ctx, cancel := context.WithCancel(context.Background())
	service := &AdvancedCryptoService{
		keys:    make(map[string]*KeyEntry),
		ctx:     ctx,
		cancel:  cancel,
	}
	service.startKeyRotation()
	return service
}

func (s *AdvancedCryptoService) startKeyRotation() {
	s.keyRotationTicker = time.NewTicker(KeyRotationInterval)
	go func() {
		for {
			select {
			case <-s.keyRotationTicker.C:
				s.rotateKeys()
			case <-s.ctx.Done():
				s.keyRotationTicker.Stop()
				return
			}
		}
	}()
}

func (s *AdvancedCryptoService) rotateKeys() {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	now := time.Now()
	for _, key := range s.keys {
		if now.Sub(key.CreatedAt) > MaxKeyAge {
			key.IsActive = false
		}
	}
}

func (s *AdvancedCryptoService) GenerateKey(ctx context.Context) (*KeyGenerationResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	keyID := uuid.New().String()
	keyBytes, err := crypto.GenerateRandomKey(crypto.KeySize256)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	
	s.keys[keyID] = &KeyEntry{
		ID:         keyID,
		Key:        keyBytes,
		CreatedAt:  time.Now(),
		LastUsedAt: time.Now(),
		IsActive:   true,
	}
	
	return &KeyGenerationResponse{
		KeyID:   keyID,
		Success: true,
	}, nil
}

func (s *AdvancedCryptoService) Encrypt(ctx context.Context, plaintext string, keyID string) (*EncryptedPayload, error) {
	s.mu.RLock()
	key, exists := s.keys[keyID]
	if !exists || !key.IsActive {
		s.mu.RUnlock()
		return nil, fmt.Errorf("invalid key ID")
	}
	key.LastUsedAt = time.Now()
	s.mu.RUnlock()
	
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}
	
	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}
	
	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	
	nonceStr := base64.StdEncoding.EncodeToString(nonce)
	ciphertextStr := base64.StdEncoding.EncodeToString(ciphertext)
	
	return &EncryptedPayload{
		Version:    2,
		Algorithm: "AES-256-GCM",
		Ciphertext: ciphertextStr,
		IV:         nonceStr,
		KeyID:      keyID,
		Timestamp:  time.Now().Unix(),
		Nonce:      nonceStr,
	}, nil
}

func (s *AdvancedCryptoService) Decrypt(ctx context.Context, payload *EncryptedPayload) (string, error) {
	s.mu.RLock()
	key, exists := s.keys[payload.KeyID]
	if !exists {
		s.mu.RUnlock()
		return "", fmt.Errorf("invalid key ID")
	}
	s.mu.RUnlock()
	
	nonce, err := base64.StdEncoding.DecodeString(payload.IV)
	if err != nil {
		return "", fmt.Errorf("failed to decode nonce: %w", err)
	}
	
	ciphertext, err := base64.StdEncoding.DecodeString(payload.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}
	
	block, err := aes.NewCipher(key.Key)
	if err != nil {
		return "", fmt.Errorf("failed to create cipher: %w", err)
	}
	
	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", fmt.Errorf("failed to create GCM: %w", err)
	}
	
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}
	
	return string(plaintext), nil
}

func (s *AdvancedCryptoService) GenerateQuantumResistantHash(data []byte) string {
	hash := sha256.New()
	hash.Write(data)
	hashValue := hash.Sum(nil)
	return base64.StdEncoding.EncodeToString(hashValue)
}

func (s *AdvancedCryptoService) GetActiveKeys(ctx context.Context) ([]string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var activeKeys []string
	for id, key := range s.keys {
		if key.IsActive {
			activeKeys = append(activeKeys, id)
		}
	}
	
	return activeKeys, nil
}

func (s *AdvancedCryptoService) Close() {
	s.cancel()
}
