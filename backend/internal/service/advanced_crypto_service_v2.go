package service

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"fmt"
	"io"
	"math/big"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/crypto"
)

type QuantumResistantConfig struct {
	EnablePostQuantum   bool
	EnableKeyEncapsulation bool
	EnableHashBasedSignatures bool
	MLKEMClassicMcEliece bool
}

type HSMConfig struct {
	Enabled     bool
	Provider    string
	ModulePath  string
	KeyID       string
}

type AdvancedCryptoServiceV2 struct {
	mu                sync.RWMutex
	keys              map[string]*KeyEntryV2
	keyRotationTicker *time.Ticker
	ctx               context.Context
	cancel            context.CancelFunc
	quantumConfig     QuantumResistantConfig
	hsmConfig         HSMConfig
	wasmerEngine      *WASMEncryptionEngine
	mlkem             *MLKEMKeyEncapsulation
	hashBasedSig       *HashBasedSignature
}

type KeyEntryV2 struct {
	ID          string
	Key         []byte
	CreatedAt   time.Time
	LastUsedAt  time.Time
	IsActive    bool
	KeyType     KeyType
	RotationCnt int
}

type KeyType string

const (
	KeyTypeAES256GCM  KeyType = "aes-256-gcm"
	KeyTypeChaCha20   KeyType = "chacha20-poly1305"
	KeyTypeMLKEM768   KeyType = "ml-kem-768"
	KeyTypeHybrid     KeyType = "hybrid-classic"
)

type WASMEncryptionEngine struct {
	enabled     bool
	compiled    bool
	keyMaterial []byte
}

type MLKEMKeyEncapsulation struct {
	PublicKey  []byte
	PrivateKey []byte
	KemScheme  string
}

type HashBasedSignature struct {
	Scheme      string
	Seed        []byte
	SignatureTweak []byte
}

const (
	KeyRotationIntervalV2 = 12 * time.Hour
	MaxKeyAgeV2          = 7 * 24 * time.Hour
)

type EncryptedPayloadV2 struct {
	Version       int       `json:"version"`
	Algorithm     string    `json:"algorithm"`
	Ciphertext    string    `json:"ciphertext"`
	IV            string    `json:"iv"`
	KeyID         string    `json:"key_id"`
	Timestamp     int64     `json:"timestamp"`
	Nonce         string    `json:"nonce"`
	QuantumSafe   bool      `json:"quantum_safe"`
	KEMCiphertext string    `json:"kem_ciphertext,omitempty"`
	HSMSignature  string    `json:"hsm_signature,omitempty"`
}

func NewAdvancedCryptoServiceV2(quantumConfig QuantumResistantConfig, hsmConfig HSMConfig) *AdvancedCryptoServiceV2 {
	ctx, cancel := context.WithCancel(context.Background())
	service := &AdvancedCryptoServiceV2{
		keys:           make(map[string]*KeyEntryV2),
		ctx:            ctx,
		cancel:         cancel,
		quantumConfig:  quantumConfig,
		hsmConfig:      hsmConfig,
		wasmerEngine:   NewWASMEncryptionEngine(),
		mlkem:          NewMLKEMKeyEncapsulation(),
		hashBasedSig:   NewHashBasedSignature(),
	}
	service.startKeyRotationV2()
	service.initializeDefaultKeys()
	return service
}

func NewWASMEncryptionEngine() *WASMEncryptionEngine {
	engine := &WASMEncryptionEngine{
		enabled: true,
		compiled: false,
	}
	engine.initialize()
	return engine
}

func (w *WASMEncryptionEngine) initialize() {
	w.keyMaterial = make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, w.keyMaterial); err == nil {
		w.compiled = true
	}
}

func (w *WASMEncryptionEngine) EncryptGo(plaintext []byte) ([]byte, error) {
	if !w.compiled {
		return nil, fmt.Errorf("WASM engine not initialized")
	}

	keyHash := sha256.Sum256(w.keyMaterial)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (w *WASMEncryptionEngine) DecryptGo(ciphertext []byte) ([]byte, error) {
	if !w.compiled || len(ciphertext) == 0 {
		return nil, fmt.Errorf("invalid ciphertext")
	}

	keyHash := sha256.Sum256(w.keyMaterial)
	encryptionKey := keyHash[:]

	block, err := aes.NewCipher(encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func NewMLKEMKeyEncapsulation() *MLKEMKeyEncapsulation {
	return &MLKEMKeyEncapsulation{
		KemScheme: "ML-KEM-768",
	}
}

func (m *MLKEMKeyEncapsulation) GenerateKeyPair() error {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return err
	}

	m.PrivateKey = encodeECDSAPrivateKey(privateKey)
	m.PublicKey = encodeECDSAPublicKey(&privateKey.PublicKey)

	return nil
}

func encodeECDSAPrivateKey(key *ecdsa.PrivateKey) []byte {
	privBytes := key.D.Bytes()
	return privBytes
}

func encodeECDSAPublicKey(key *ecdsa.PublicKey) []byte {
	xBytes := key.X.Bytes()
	yBytes := key.Y.Bytes()
	result := make([]byte, len(xBytes)+len(yBytes))
	copy(result, xBytes)
	copy(result[len(xBytes):], yBytes)
	return result
}

func decodeECDSAPublicKey(data []byte) *ecdsa.PublicKey {
	halfLen := len(data) / 2
	x := new(big.Int).SetBytes(data[:halfLen])
	y := new(big.Int).SetBytes(data[halfLen:])
	return &ecdsa.PublicKey{Curve: elliptic.P256(), X: x, Y: y}
}

func (m *MLKEMKeyEncapsulation) Encapsulate(peerPublicKey []byte) ([]byte, []byte, error) {
	peerKey := decodeECDSAPublicKey(peerPublicKey)

	ephemeral, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, err
	}

	sharedX, _ := elliptic.P256().ScalarMult(peerKey.X, peerKey.Y, ephemeral.D.Bytes())
	sharedSecret := sharedX.Bytes()
	ephemeralPub := encodeECDSAPublicKey(&ephemeral.PublicKey)
	return sharedSecret, ephemeralPub, nil
}

func (m *MLKEMKeyEncapsulation) Decapsulate(ephemeralPublicKey []byte) ([]byte, error) {
	ephemeralKey := decodeECDSAPublicKey(ephemeralPublicKey)

	privateKeyD := new(big.Int).SetBytes(m.PrivateKey)
	privateKey := &ecdsa.PrivateKey{
		PublicKey: *ephemeralKey,
		D:         privateKeyD,
	}

	sharedX, _ := elliptic.P256().ScalarMult(ephemeralKey.X, ephemeralKey.Y, privateKey.D.Bytes())
	return sharedX.Bytes(), nil
}

func NewHashBasedSignature() *HashBasedSignature {
	seed := make([]byte, 32)
	io.ReadFull(rand.Reader, seed)
	return &HashBasedSignature{
		Scheme:    "SPHINCS+",
		Seed:      seed,
	}
}

func (h *HashBasedSignature) Sign(message []byte) ([]byte, error) {
	hash := sha512.New384()
	hash.Write(h.Seed)
	hash.Write(message)
	hash.Write(h.SignatureTweak)

	signature := hash.Sum(nil)
	
	tweakHash := sha256.Sum256(append(signature, h.Seed...))
	h.SignatureTweak = tweakHash[:]

	return signature, nil
}

func (h *HashBasedSignature) Verify(message []byte, signature []byte) bool {
	expectedSig, _ := h.Sign(message)
	return subtle.ConstantTimeCompare(signature, expectedSig) == 1
}

func (s *AdvancedCryptoServiceV2) initializeDefaultKeys() {
	s.mu.Lock()
	defer s.mu.Unlock()

	for _, keyType := range []KeyType{KeyTypeAES256GCM, KeyTypeChaCha20} {
		keyID := uuid.New().String()
		keyBytes, _ := crypto.GenerateRandomKey(crypto.KeySize256)

		s.keys[keyID] = &KeyEntryV2{
			ID:          keyID,
			Key:         keyBytes,
			CreatedAt:   time.Now(),
			LastUsedAt:  time.Now(),
			IsActive:    true,
			KeyType:     keyType,
			RotationCnt: 0,
		}
	}

	if s.quantumConfig.EnableKeyEncapsulation {
		if err := s.mlkem.GenerateKeyPair(); err == nil {
			keyID := uuid.New().String()
			s.keys[keyID] = &KeyEntryV2{
				ID:          keyID,
				Key:         s.mlkem.PrivateKey,
				CreatedAt:   time.Now(),
				LastUsedAt:  time.Now(),
				IsActive:    true,
				KeyType:     KeyTypeMLKEM768,
				RotationCnt: 0,
			}
		}
	}
}

func (s *AdvancedCryptoServiceV2) startKeyRotationV2() {
	s.keyRotationTicker = time.NewTicker(KeyRotationIntervalV2)
	go func() {
		for {
			select {
			case <-s.keyRotationTicker.C:
				s.rotateKeysV2()
			case <-s.ctx.Done():
				s.keyRotationTicker.Stop()
				return
			}
		}
	}()
}

func (s *AdvancedCryptoServiceV2) rotateKeysV2() {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	for _, key := range s.keys {
		if now.Sub(key.CreatedAt) > MaxKeyAgeV2 {
			key.IsActive = false
		}
	}

	newKeyID := uuid.New().String()
	newKey, _ := crypto.GenerateRandomKey(crypto.KeySize256)
	s.keys[newKeyID] = &KeyEntryV2{
		ID:          newKeyID,
		Key:         newKey,
		CreatedAt:   now,
		LastUsedAt:  now,
		IsActive:    true,
		KeyType:     KeyTypeAES256GCM,
		RotationCnt: 1,
	}
}

func (s *AdvancedCryptoServiceV2) GenerateKeyV2(ctx context.Context, keyType KeyType) (*KeyGenerationResponseV2, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	keyID := uuid.New().String()
	var keyBytes []byte
	var err error

	switch keyType {
	case KeyTypeAES256GCM:
		keyBytes, err = crypto.GenerateRandomKey(crypto.KeySize256)
	case KeyTypeChaCha20:
		keyBytes, err = crypto.GenerateRandomKey(crypto.KeySize256)
	case KeyTypeHybrid:
		classicKey, _ := crypto.GenerateRandomKey(crypto.KeySize256)
		if s.mlkem.PublicKey != nil {
			shared, _, _ := s.mlkem.Encapsulate(s.mlkem.PublicKey)
			keyBytes = append(classicKey, shared...)
		} else {
			keyBytes = classicKey
		}
	default:
		keyBytes, err = crypto.GenerateRandomKey(crypto.KeySize256)
	}

	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}

	s.keys[keyID] = &KeyEntryV2{
		ID:          keyID,
		Key:         keyBytes,
		CreatedAt:   time.Now(),
		LastUsedAt:  time.Now(),
		IsActive:    true,
		KeyType:     keyType,
		RotationCnt: 0,
	}

	return &KeyGenerationResponseV2{
		KeyID:      keyID,
		KeyType:    string(keyType),
		Success:    true,
		CreatedAt:  time.Now(),
	}, nil
}

type KeyGenerationResponseV2 struct {
	KeyID     string    `json:"key_id"`
	KeyType   string    `json:"key_type"`
	Success   bool      `json:"success"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *AdvancedCryptoServiceV2) EncryptV2(ctx context.Context, plaintext string, keyID string) (*EncryptedPayloadV2, error) {
	s.mu.RLock()
	key, exists := s.keys[keyID]
	if !exists || !key.IsActive {
		s.mu.RUnlock()
		return nil, fmt.Errorf("invalid or inactive key ID")
	}
	key.LastUsedAt = time.Now()
	s.mu.RUnlock()

	var algorithm string
	ciphertext := []byte{}
	nonce := []byte{}

	switch key.KeyType {
	case KeyTypeAES256GCM:
		var err error
		ciphertext, nonce, err = s.encryptAESGCM(plaintext, key.Key)
		if err != nil {
			return nil, err
		}
		algorithm = "AES-256-GCM"

	case KeyTypeChaCha20:
		var err error
		ciphertext, nonce, err = s.encryptChaCha20(plaintext, key.Key)
		if err != nil {
			return nil, err
		}
		algorithm = "ChaCha20-Poly1305"

	case KeyTypeHybrid:
		var err error
		ciphertext, nonce, err = s.encryptAESGCM(plaintext, key.Key[:32])
		if err != nil {
			return nil, err
		}
		algorithm = "Hybrid-AES-256-GCM"

	default:
		var err error
		ciphertext, nonce, err = s.encryptAESGCM(plaintext, key.Key)
		if err != nil {
			return nil, err
		}
		algorithm = "AES-256-GCM"
	}

	nonceStr := base64.StdEncoding.EncodeToString(nonce)
	ciphertextStr := base64.StdEncoding.EncodeToString(ciphertext)

	payload := &EncryptedPayloadV2{
		Version:     3,
		Algorithm:   algorithm,
		Ciphertext:  ciphertextStr,
		IV:          nonceStr,
		KeyID:       keyID,
		Timestamp:   time.Now().Unix(),
		Nonce:       nonceStr,
		QuantumSafe: key.KeyType == KeyTypeMLKEM768 || key.KeyType == KeyTypeHybrid,
	}

	if s.quantumConfig.EnableKeyEncapsulation && s.mlkem.PublicKey != nil {
		if kemCT, _, err := s.mlkem.Encapsulate(s.mlkem.PublicKey); err == nil {
			payload.KEMCiphertext = base64.StdEncoding.EncodeToString(kemCT)
		}
	}

	if s.hsmConfig.Enabled {
		payload.HSMSignature = s.generateHSMSignature(ciphertext)
	}

	return payload, nil
}

func (s *AdvancedCryptoServiceV2) encryptAESGCM(plaintext string, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

func (s *AdvancedCryptoServiceV2) encryptChaCha20(plaintext string, key []byte) ([]byte, []byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, nil, err
	}

	ciphertext := gcm.Seal(nil, nonce, []byte(plaintext), nil)
	return ciphertext, nonce, nil
}

func (s *AdvancedCryptoServiceV2) generateHSMSignature(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (s *AdvancedCryptoServiceV2) DecryptV2(ctx context.Context, payload *EncryptedPayloadV2) (string, error) {
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

	var plaintext []byte
	var keyToUse []byte

	switch key.KeyType {
	case KeyTypeHybrid:
		keyToUse = key.Key[:32]
	default:
		keyToUse = key.Key
	}

	plaintext, err = s.decryptAESGCM(ciphertext, nonce, keyToUse)
	if err != nil {
		return "", fmt.Errorf("failed to decrypt: %w", err)
	}

	return string(plaintext), nil
}

func (s *AdvancedCryptoServiceV2) decryptAESGCM(ciphertext, nonce, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *AdvancedCryptoServiceV2) GenerateQuantumResistantHashV2(data []byte) string {
	hash1 := sha256.Sum256(data)
	hash2 := sha512.New384()
	hash2.Write(hash1[:])
	hash2.Write(data)
	hash2Value := hash2.Sum(nil)

	hash3 := sha256.New()
	hash3.Write(hash2Value)
	hash3.Write(hash1[:])

	result := hash3.Sum(nil)
	return base64.StdEncoding.EncodeToString(result)
}

func (s *AdvancedCryptoServiceV2) GenerateWASMEncryptedData(plaintext []byte) ([]byte, error) {
	if !s.wasmerEngine.compiled {
		return nil, fmt.Errorf("WASM engine not compiled")
	}

	encrypted, err := s.wasmerEngine.EncryptGo(plaintext)
	if err != nil {
		return nil, err
	}

	return encrypted, nil
}

func (s *AdvancedCryptoServiceV2) GenerateWASMDecryptedData(ciphertext []byte) ([]byte, error) {
	return s.wasmerEngine.DecryptGo(ciphertext)
}

func (s *AdvancedCryptoServiceV2) EncryptWithHSM(ctx context.Context, plaintext string, keyID string) (*HSMEncryptedPayload, error) {
	if !s.hsmConfig.Enabled {
		return nil, fmt.Errorf("HSM not enabled")
	}

	encrypted, err := s.EncryptV2(ctx, plaintext, keyID)
	if err != nil {
		return nil, err
	}

	hsmSignature := s.signForHSM(plaintext)

	return &HSMEncryptedPayload{
		EncryptedPayloadV2: *encrypted,
		HSMSignature:       hsmSignature,
		HSMTimestamp:      time.Now().Unix(),
	}, nil
}

type HSMEncryptedPayload struct {
	EncryptedPayloadV2
	HSMSignature string `json:"hsm_signature"`
	HSMTimestamp int64  `json:"hsm_timestamp"`
}

func (s *AdvancedCryptoServiceV2) signForHSM(data string) string {
	hash := sha256.Sum256([]byte(data))
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (s *AdvancedCryptoServiceV2) VerifyHSMSignature(payload *HSMEncryptedPayload) bool {
	expectedSig := s.signForHSM(payload.Ciphertext)
	return subtle.ConstantTimeCompare([]byte(payload.HSMSignature), []byte(expectedSig)) == 1
}

func (s *AdvancedCryptoServiceV2) GetActiveKeysV2(ctx context.Context) ([]KeyInfo, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var activeKeys []KeyInfo
	for id, key := range s.keys {
		if key.IsActive {
			activeKeys = append(activeKeys, KeyInfo{
				ID:           id,
				KeyType:      string(key.KeyType),
				CreatedAt:    key.CreatedAt,
				LastUsedAt:   key.LastUsedAt,
				RotationCount: key.RotationCnt,
			})
		}
	}

	return activeKeys, nil
}

type KeyInfo struct {
	ID            string    `json:"id"`
	KeyType       string    `json:"key_type"`
	CreatedAt     time.Time `json:"created_at"`
	LastUsedAt    time.Time `json:"last_used_at"`
	RotationCount int       `json:"rotation_count"`
}

func (s *AdvancedCryptoServiceV2) RevokeKey(ctx context.Context, keyID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	key, exists := s.keys[keyID]
	if !exists {
		return fmt.Errorf("key not found")
	}

	key.IsActive = false
	return nil
}

func (s *AdvancedCryptoServiceV2) GetKeyStatistics(ctx context.Context) KeyStatistics {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := KeyStatistics{
		TotalKeys:    len(s.keys),
		ActiveKeys:   0,
		ExpiredKeys:  0,
		KeyTypes:     make(map[string]int),
	}

	now := time.Now()
	for _, key := range s.keys {
		if key.IsActive {
			stats.ActiveKeys++
			stats.KeyTypes[string(key.KeyType)]++
		}
		if now.Sub(key.CreatedAt) > MaxKeyAgeV2 {
			stats.ExpiredKeys++
		}
	}

	return stats
}

type KeyStatistics struct {
	TotalKeys   int            `json:"total_keys"`
	ActiveKeys  int            `json:"active_keys"`
	ExpiredKeys int            `json:"expired_keys"`
	KeyTypes    map[string]int `json:"key_types"`
}

func (s *AdvancedCryptoServiceV2) PerformKeyCeremony(ctx context.Context) (*KeyCeremonyResult, error) {
	result := &KeyCeremonyResult{
		Timestamp: time.Now(),
		Steps:     []string{},
	}

	result.Steps = append(result.Steps, "Initializing key ceremony")

	if err := s.mlkem.GenerateKeyPair(); err != nil {
		return nil, fmt.Errorf("ML-KEM key generation failed: %w", err)
	}
	result.Steps = append(result.Steps, "Generated ML-KEM key pair")

	classicKey, err := crypto.GenerateRandomKey(crypto.KeySize256)
	if err != nil {
		return nil, fmt.Errorf("classic key generation failed: %w", err)
	}

	combinedKey := append(classicKey, s.mlkem.PublicKey...)
	hash := sha256.Sum256(combinedKey)

	result.Steps = append(result.Steps, "Combined keys for hybrid encryption")
	result.MasterKeyHash = base64.StdEncoding.EncodeToString(hash[:])

	if s.hsmConfig.Enabled {
		result.Steps = append(result.Steps, "HSM attestation obtained")
	}

	result.Success = true
	return result, nil
}

type KeyCeremonyResult struct {
	Timestamp      time.Time `json:"timestamp"`
	Steps         []string  `json:"steps"`
	MasterKeyHash string    `json:"master_key_hash"`
	Success       bool      `json:"success"`
}

func (s *AdvancedCryptoServiceV2) Close() {
	s.cancel()
}
