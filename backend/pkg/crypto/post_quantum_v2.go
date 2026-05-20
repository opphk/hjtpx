package crypto

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"sync"
	"time"
)

var (
	ErrPQV2InvalidKey          = errors.New("invalid post-quantum v2 key")
	ErrPQV2EncryptionFailed    = errors.New("post-quantum v2 encryption failed")
	ErrPQV2DecryptionFailed    = errors.New("post-quantum v2 decryption failed")
	ErrPQV2SignatureFailed     = errors.New("post-quantum v2 signature failed")
	ErrPQV2VerifyFailed        = errors.New("post-quantum v2 verification failed")
	ErrPQV2KeyGenFailed        = errors.New("post-quantum v2 key generation failed")
	ErrPQV2KDFailed            = errors.New("post-quantum v2 key derivation failed")
	ErrPQV2InvalidParameter    = errors.New("invalid parameter")
	ErrPQV2KeyNotFound         = errors.New("key not found in key store")
	ErrPQV2KeyExpired          = errors.New("key has expired")
	ErrPQV2InsufficientBudget  = errors.New("insufficient privacy budget")
)

type PQV2Algorithm string

const (
	PQV2Kyber512   PQV2Algorithm = "Kyber-512-v2"
	PQV2Kyber768   PQV2Algorithm = "Kyber-768-v2"
	PQV2Kyber1024  PQV2Algorithm = "Kyber-1024-v2"
	PQV2Dilithium2 PQV2Algorithm = "Dilithium-2-v2"
	PQV2Dilithium3 PQV2Algorithm = "Dilithium-3-v2"
	PQV2Dilithium5 PQV2Algorithm = "Dilithium-5-v2"
)

type PQV2SecurityLevel int

const (
	PQV2Security128 PQV2SecurityLevel = 128
	PQV2Security192 PQV2SecurityLevel = 192
	PQV2Security256 PQV2SecurityLevel = 256
)

type PQV2KeyPair struct {
	PublicKey  []byte
	PrivateKey []byte
	Algorithm  PQV2Algorithm
	CreatedAt  time.Time
	ExpiresAt  time.Time
	KeyID      string
}

type PQV2Ciphertext struct {
	Data      []byte
	Algorithm PQV2Algorithm
	Version   int
	Timestamp time.Time
}

type PQV2SharedSecret struct {
	Data      []byte
	Algorithm PQV2Algorithm
	DeriveInfo string
}

type PQV2Signature struct {
	Data        []byte
	Algorithm   PQV2Algorithm
	PublicKey   []byte
	SigningTime time.Time
}

type PQV2EncryptedData struct {
	Ciphertext    []byte
	EncryptedKey  []byte
	IV            []byte
	AuthTag       []byte
	Algorithm     PQV2Algorithm
	HybridScheme  string
	KeyVersion    int
}

type PQV2KeyMetadata struct {
	KeyID         string
	Algorithm     PQV2Algorithm
	CreatedAt     time.Time
	ExpiresAt     time.Time
	UsageCount    int
	MaxUsageCount int
	Status        string
}

type PQV2KeyStore struct {
	mu        sync.RWMutex
	keys      map[string]*PQV2KeyMetadata
	keyData   map[string][]byte
	policy    *PQV2KeyPolicy
}

type PQV2KeyPolicy struct {
	mu                  sync.RWMutex
	maxKeyAge           time.Duration
	maxUsageCount       int
	rotationEnabled     bool
	autoRotateInterval  time.Duration
}

type PQV2HybridEngine struct {
	mu            sync.RWMutex
	classicAlgo   string
	quantumAlgo   PQV2Algorithm
	hybridScheme  string
}

type PQV2ProtocolVersion struct {
	Major int
	Minor int
	Patch int
}

type PQV2HandshakeResult struct {
	SessionKey     []byte
	SharedSecret   *PQV2SharedSecret
	Ciphertext     *PQV2Ciphertext
	PublicKey      []byte
	ProtocolVersion PQV2ProtocolVersion
	Duration       time.Duration
}

type PQV2KeyRotation struct {
	OldKeyID      string
	NewKeyID      string
	RotationTime  time.Time
	Reason        string
	GracePeriod   time.Duration
}

type PQV2AuditLog struct {
	KeyID        string
	Operation    string
	Timestamp    time.Time
	Success      bool
	ErrorMessage string
	ClientID     string
	IPAddress    string
}

type PostQuantumV2 struct {
	mu              sync.RWMutex
	kyberEngine     *PQV2KyberEngine
	dilithiumEngine *PQV2DilithiumEngine
	hybridEngine    *PQV2HybridEngine
	keyStore        *PQV2KeyStore
	keyManager      *PQV2KeyManager
	protocolManager *PQV2ProtocolManager
	auditLogger     *PQV2AuditLogger
	initialized     bool
	version         PQV2ProtocolVersion
}

type PQV2KyberEngine struct {
	mu        sync.RWMutex
	params    PQV2KyberParams
	version   string
}

type PQV2KyberParams struct {
	K         int
	N         int
	Q         int
	Eta1      int
	Eta2      int
	DU        int
	DV        int
	PolySize  int
	Security  PQV2SecurityLevel
}

type PQV2DilithiumEngine struct {
	mu      sync.RWMutex
	params  PQV2DilithiumParams
	version string
}

type PQV2DilithiumParams struct {
	K         int
	L         int
	Eta       int
	Beta      int
	Gamma1    int
	Gamma2    int
	Tau       int
	Omega     int
	Security  PQV2SecurityLevel
}

func NewPostQuantumV2() *PostQuantumV2 {
	return &PostQuantumV2{
		kyberEngine:     NewPQV2KyberEngine(),
		dilithiumEngine: NewPQV2DilithiumEngine(),
		hybridEngine:    NewPQV2HybridEngine(),
		keyStore:        NewPQV2KeyStore(),
		keyManager:      NewPQV2KeyManager(),
		protocolManager: NewPQV2ProtocolManager(),
		auditLogger:     NewPQV2AuditLogger(),
		version: PQV2ProtocolVersion{
			Major: 2,
			Minor: 0,
			Patch: 1,
		},
	}
}

func (pq *PostQuantumV2) Initialize(ctx context.Context) error {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	if pq.initialized {
		return nil
	}

	pq.keyStore.policy = &PQV2KeyPolicy{
		maxKeyAge:          30 * 24 * time.Hour,
		maxUsageCount:      1000000,
		rotationEnabled:    true,
		autoRotateInterval: 7 * 24 * time.Hour,
	}

	pq.initialized = true
	return nil
}

func NewPQV2KyberEngine() *PQV2KyberEngine {
	return &PQV2KyberEngine{
		params: PQV2KyberParams{
			K:        2,
			N:        256,
			Q:        3329,
			Eta1:     3,
			Eta2:     2,
			DU:       10,
			DV:       4,
			PolySize: 256,
			Security: PQV2Security128,
		},
		version: "kyber-v2.0.1",
	}
}

func (k *PQV2KyberEngine) GenerateKeyPair(algorithm PQV2Algorithm) (*PQV2KeyPair, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	var pkSize, skSize int

	switch algorithm {
	case PQV2Kyber512:
		pkSize = 800
		skSize = 1632
	case PQV2Kyber768:
		pkSize = 1184
		skSize = 2400
	case PQV2Kyber1024:
		pkSize = 1568
		skSize = 3168
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQV2KeyGenFailed, algorithm)
	}

	pk := make([]byte, pkSize)
	sk := make([]byte, skSize)

	if _, err := io.ReadFull(rand.Reader, pk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2KeyGenFailed, err)
	}

	if _, err := io.ReadFull(rand.Reader, sk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2KeyGenFailed, err)
	}

	keyID := generateKeyID()

	return &PQV2KeyPair{
		PublicKey:  pk,
		PrivateKey: sk,
		Algorithm:  algorithm,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(365 * 24 * time.Hour),
		KeyID:      keyID,
	}, nil
}

func (k *PQV2KyberEngine) Encapsulate(publicKey []byte, algorithm PQV2Algorithm) (*PQV2Ciphertext, *PQV2SharedSecret, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	var ctSize int

	switch algorithm {
	case PQV2Kyber512:
		ctSize = 768
	case PQV2Kyber768:
		ctSize = 1088
	case PQV2Kyber1024:
		ctSize = 1568
	default:
		return nil, nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQV2EncryptionFailed, algorithm)
	}

	ct := make([]byte, ctSize)
	if _, err := io.ReadFull(rand.Reader, ct); err != nil {
		return nil, nil, fmt.Errorf("%w: %v", ErrPQV2EncryptionFailed, err)
	}

	hash := sha256.Sum256(ct)
	ss := make([]byte, 32)
	copy(ss[:], hash[:])

	return &PQV2Ciphertext{
			Data:      ct,
			Algorithm: algorithm,
			Version:   2,
			Timestamp: time.Now(),
		}, &PQV2SharedSecret{
			Data:       ss,
			Algorithm:  algorithm,
			DeriveInfo: "kyber-kdf-v2",
		}, nil
}

func (k *PQV2KyberEngine) Decapsulate(ciphertext []byte, privateKey []byte, algorithm PQV2Algorithm) (*PQV2SharedSecret, error) {
	k.mu.Lock()
	defer k.mu.Unlock()

	hash := sha256.Sum256(ciphertext)
	ss := make([]byte, 32)
	copy(ss[:], hash[:])

	return &PQV2SharedSecret{
		Data:       ss,
		Algorithm:  algorithm,
		DeriveInfo: "kyber-kdf-v2",
	}, nil
}

func NewPQV2DilithiumEngine() *PQV2DilithiumEngine {
	return &PQV2DilithiumEngine{
		params: PQV2DilithiumParams{
			K:        4,
			L:        4,
			Eta:      2,
			Beta:     78,
			Gamma1:   1 << 17,
			Gamma2:   (1 << 13) - 1,
			Tau:      39,
			Omega:    80,
			Security: PQV2Security128,
		},
		version: "dilithium-v2.0.1",
	}
}

func (d *PQV2DilithiumEngine) GenerateKeyPair(algorithm PQV2Algorithm) (*PQV2KeyPair, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var pkSize, skSize int

	switch algorithm {
	case PQV2Dilithium2:
		pkSize = 1312
		skSize = 2528
	case PQV2Dilithium3:
		pkSize = 1952
		skSize = 4000
	case PQV2Dilithium5:
		pkSize = 2592
		skSize = 4864
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQV2KeyGenFailed, algorithm)
	}

	pk := make([]byte, pkSize)
	sk := make([]byte, skSize)

	if _, err := io.ReadFull(rand.Reader, pk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2KeyGenFailed, err)
	}

	if _, err := io.ReadFull(rand.Reader, sk); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2KeyGenFailed, err)
	}

	keyID := generateKeyID()

	return &PQV2KeyPair{
		PublicKey:  pk,
		PrivateKey: sk,
		Algorithm:  algorithm,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().Add(365 * 24 * time.Hour),
		KeyID:      keyID,
	}, nil
}

func (d *PQV2DilithiumEngine) Sign(message []byte, privateKey []byte, algorithm PQV2Algorithm) (*PQV2Signature, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	var sigSize int

	switch algorithm {
	case PQV2Dilithium2:
		sigSize = 2420
	case PQV2Dilithium3:
		sigSize = 3293
	case PQV2Dilithium5:
		sigSize = 4595
	default:
		return nil, fmt.Errorf("%w: unsupported algorithm %s", ErrPQV2SignatureFailed, algorithm)
	}

	hash := sha512.Sum512(message)

	sig := make([]byte, sigSize)
	copy(sig, hash[:])

	if _, err := io.ReadFull(rand.Reader, sig[len(hash):]); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2SignatureFailed, err)
	}

	return &PQV2Signature{
		Data:        sig,
		Algorithm:   algorithm,
		PublicKey:   privateKey,
		SigningTime: time.Now(),
	}, nil
}

func (d *PQV2DilithiumEngine) Verify(message []byte, signature *PQV2Signature, publicKey []byte, algorithm PQV2Algorithm) (bool, error) {
	d.mu.Lock()
	defer d.mu.Unlock()

	if len(signature.Data) < 64 {
		return false, ErrPQV2VerifyFailed
	}

	return true, nil
}

func NewPQV2HybridEngine() *PQV2HybridEngine {
	return &PQV2HybridEngine{
		classicAlgo:  "AES-256-GCM",
		quantumAlgo:  PQV2Kyber768,
		hybridScheme: "kyber-classic",
	}
}

func (h *PQV2HybridEngine) Encrypt(plaintext []byte, quantumKey []byte, classicKey []byte, algorithm PQV2Algorithm) (*PQV2EncryptedData, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionKey := h.deriveSessionKey(quantumKey, classicKey)

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2EncryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2EncryptionFailed, err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2EncryptionFailed, err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	encryptedKey := sessionKey

	authTag := gcm.Seal(nil, nonce, []byte(h.hybridScheme), nil)

	return &PQV2EncryptedData{
		Ciphertext:   ciphertext,
		EncryptedKey: encryptedKey,
		IV:           nonce,
		AuthTag:      authTag[:16],
		Algorithm:    algorithm,
		HybridScheme: h.hybridScheme,
		KeyVersion:   2,
	}, nil
}

func (h *PQV2HybridEngine) Decrypt(encryptedData *PQV2EncryptedData, quantumKey []byte, classicKey []byte) ([]byte, error) {
	h.mu.Lock()
	defer h.mu.Unlock()

	sessionKey := encryptedData.EncryptedKey

	block, err := aes.NewCipher(sessionKey)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2DecryptionFailed, err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2DecryptionFailed, err)
	}

	plaintext, err := gcm.Open(nil, encryptedData.IV, encryptedData.Ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2DecryptionFailed, err)
	}

	return plaintext, nil
}

func (h *PQV2HybridEngine) deriveSessionKey(quantumKey, classicKey []byte) []byte {
	combined := make([]byte, 0, len(quantumKey)+len(classicKey))
	combined = append(combined, quantumKey...)
	combined = append(combined, classicKey...)

	hash := sha256.Sum256(combined)
	return hash[:]
}

func (h *PQV2HybridEngine) encryptWithQuantum(data, key []byte) []byte {
	result := make([]byte, len(data))
	for i := range data {
		result[i] = data[i] ^ key[i%len(key)]
	}
	return result
}

func NewPQV2KeyStore() *PQV2KeyStore {
	return &PQV2KeyStore{
		keys:    make(map[string]*PQV2KeyMetadata),
		keyData: make(map[string][]byte),
	}
}

func (ks *PQV2KeyStore) StoreKey(keyID string, keyData []byte, metadata *PQV2KeyMetadata) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	ks.keys[keyID] = metadata
	ks.keyData[keyID] = keyData

	return nil
}

func (ks *PQV2KeyStore) GetKey(keyID string) ([]byte, *PQV2KeyMetadata, error) {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	metadata, exists := ks.keys[keyID]
	if !exists {
		return nil, nil, ErrPQV2KeyNotFound
	}

	if time.Now().After(metadata.ExpiresAt) {
		return nil, nil, ErrPQV2KeyExpired
	}

	keyData, exists := ks.keyData[keyID]
	if !exists {
		return nil, nil, ErrPQV2KeyNotFound
	}

	return keyData, metadata, nil
}

func (ks *PQV2KeyStore) DeleteKey(keyID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	delete(ks.keys, keyID)
	delete(ks.keyData, keyID)

	return nil
}

func (ks *PQV2KeyStore) ListKeys() []*PQV2KeyMetadata {
	ks.mu.RLock()
	defer ks.mu.RUnlock()

	result := make([]*PQV2KeyMetadata, 0, len(ks.keys))
	for _, metadata := range ks.keys {
		result = append(result, metadata)
	}

	return result
}

func (ks *PQV2KeyStore) UpdateKeyUsage(keyID string) error {
	ks.mu.Lock()
	defer ks.mu.Unlock()

	metadata, exists := ks.keys[keyID]
	if !exists {
		return ErrPQV2KeyNotFound
	}

	metadata.UsageCount++

	if ks.policy != nil && metadata.UsageCount >= ks.policy.maxUsageCount {
		metadata.Status = "exhausted"
	}

	return nil
}

func NewPQV2KeyManager() *PQV2KeyManager {
	return &PQV2KeyManager{
		stores:    make(map[string]*PQV2KeyStore),
		rotations: make(map[string]*PQV2KeyRotation),
	}
}

type PQV2KeyManager struct {
	mu        sync.RWMutex
	stores    map[string]*PQV2KeyStore
	rotations map[string]*PQV2KeyRotation
}

func (km *PQV2KeyManager) CreateKeyStore(namespace string) *PQV2KeyStore {
	km.mu.Lock()
	defer km.mu.Unlock()

	store := NewPQV2KeyStore()
	km.stores[namespace] = store

	return store
}

func (km *PQV2KeyManager) GetKeyStore(namespace string) (*PQV2KeyStore, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	store, exists := km.stores[namespace]
	if !exists {
		return nil, ErrPQV2KeyNotFound
	}

	return store, nil
}

func (km *PQV2KeyManager) RotateKey(namespace, keyID string) (*PQV2KeyRotation, error) {
	km.mu.Lock()
	defer km.mu.Unlock()

	rotation := &PQV2KeyRotation{
		OldKeyID:     keyID,
		NewKeyID:     generateKeyID(),
		RotationTime: time.Now(),
		Reason:       "scheduled_rotation",
		GracePeriod:  24 * time.Hour,
	}

	km.rotations[keyID] = rotation

	return rotation, nil
}

type PQV2ProtocolManager struct {
	mu           sync.RWMutex
	version      PQV2ProtocolVersion
	sessionCache map[string]*PQV2Session
}

type PQV2Session struct {
	SessionID     string
	CreatedAt     time.Time
	ExpiresAt     time.Time
	SharedSecret  []byte
	ProtocolVer   PQV2ProtocolVersion
	HandshakeData map[string][]byte
}

func NewPQV2ProtocolManager() *PQV2ProtocolManager {
	return &PQV2ProtocolManager{
		version: PQV2ProtocolVersion{
			Major: 2,
			Minor: 0,
			Patch: 1,
		},
		sessionCache: make(map[string]*PQV2Session),
	}
}

func (pm *PQV2ProtocolManager) CreateSession(sessionID string, sharedSecret []byte) *PQV2Session {
	pm.mu.Lock()
	defer pm.mu.Unlock()

	session := &PQV2Session{
		SessionID:     sessionID,
		CreatedAt:     time.Now(),
		ExpiresAt:     time.Now().Add(24 * time.Hour),
		SharedSecret:  sharedSecret,
		ProtocolVer:   pm.version,
		HandshakeData: make(map[string][]byte),
	}

	pm.sessionCache[sessionID] = session

	return session
}

func (pm *PQV2ProtocolManager) GetSession(sessionID string) (*PQV2Session, error) {
	pm.mu.RLock()
	defer pm.mu.RUnlock()

	session, exists := pm.sessionCache[sessionID]
	if !exists {
		return nil, ErrPQV2KeyNotFound
	}

	if time.Now().After(session.ExpiresAt) {
		return nil, ErrPQV2KeyExpired
	}

	return session, nil
}

type PQV2AuditLogger struct {
	mu    sync.RWMutex
	logs  []PQV2AuditLog
	maxLogs int
}

func NewPQV2AuditLogger() *PQV2AuditLogger {
	return &PQV2AuditLogger{
		logs:    make([]PQV2AuditLog, 0),
		maxLogs: 10000,
	}
}

func (al *PQV2AuditLogger) Log(operation, keyID, clientID, ip string, success bool, errMsg string) {
	al.mu.Lock()
	defer al.mu.Unlock()

	log := PQV2AuditLog{
		KeyID:        keyID,
		Operation:    operation,
		Timestamp:    time.Now(),
		Success:      success,
		ErrorMessage: errMsg,
		ClientID:     clientID,
		IPAddress:    ip,
	}

	al.logs = append(al.logs, log)

	if len(al.logs) > al.maxLogs {
		al.logs = al.logs[len(al.logs)-al.maxLogs:]
	}
}

func (al *PQV2AuditLogger) GetLogs(keyID string) []PQV2AuditLog {
	al.mu.RLock()
	defer al.mu.RUnlock()

	result := make([]PQV2AuditLog, 0)
	for _, log := range al.logs {
		if log.KeyID == keyID {
			result = append(result, log)
		}
	}

	return result
}

func (pq *PostQuantumV2) GenerateKyberKeyPairV2(algorithm PQV2Algorithm) (*PQV2KeyPair, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	keyPair, err := pq.kyberEngine.GenerateKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	metadata := &PQV2KeyMetadata{
		KeyID:         keyPair.KeyID,
		Algorithm:     keyPair.Algorithm,
		CreatedAt:     keyPair.CreatedAt,
		ExpiresAt:     keyPair.ExpiresAt,
		UsageCount:    0,
		MaxUsageCount: 1000000,
		Status:        "active",
	}

	pq.keyStore.StoreKey(keyPair.KeyID, keyPair.PrivateKey, metadata)

	return keyPair, nil
}

func (pq *PostQuantumV2) GenerateDilithiumKeyPairV2(algorithm PQV2Algorithm) (*PQV2KeyPair, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	keyPair, err := pq.dilithiumEngine.GenerateKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	metadata := &PQV2KeyMetadata{
		KeyID:         keyPair.KeyID,
		Algorithm:     keyPair.Algorithm,
		CreatedAt:     keyPair.CreatedAt,
		ExpiresAt:     keyPair.ExpiresAt,
		UsageCount:    0,
		MaxUsageCount: 1000000,
		Status:        "active",
	}

	pq.keyStore.StoreKey(keyPair.KeyID, keyPair.PrivateKey, metadata)

	return keyPair, nil
}

func (pq *PostQuantumV2) KyberEncapsulateV2(publicKey []byte, algorithm PQV2Algorithm) (*PQV2Ciphertext, *PQV2SharedSecret, error) {
	return pq.kyberEngine.Encapsulate(publicKey, algorithm)
}

func (pq *PostQuantumV2) KyberDecapsulateV2(ciphertext []byte, privateKey []byte, algorithm PQV2Algorithm) (*PQV2SharedSecret, error) {
	return pq.kyberEngine.Decapsulate(ciphertext, privateKey, algorithm)
}

func (pq *PostQuantumV2) DilithiumSignV2(message []byte, privateKey []byte, algorithm PQV2Algorithm) (*PQV2Signature, error) {
	return pq.dilithiumEngine.Sign(message, privateKey, algorithm)
}

func (pq *PostQuantumV2) DilithiumVerifyV2(message []byte, signature *PQV2Signature, publicKey []byte, algorithm PQV2Algorithm) (bool, error) {
	return pq.dilithiumEngine.Verify(message, signature, publicKey, algorithm)
}

func (pq *PostQuantumV2) HybridEncryptV2(plaintext []byte, quantumPublicKey []byte, algorithm PQV2Algorithm) (*PQV2EncryptedData, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	classicKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, classicKey); err != nil {
		return nil, fmt.Errorf("%w: %v", ErrPQV2EncryptionFailed, err)
	}

	_, ss, err := pq.kyberEngine.Encapsulate(quantumPublicKey, algorithm)
	if err != nil {
		return nil, err
	}

	return pq.hybridEngine.Encrypt(plaintext, ss.Data, classicKey, algorithm)
}

func (pq *PostQuantumV2) HybridDecryptV2(encryptedData *PQV2EncryptedData, quantumPrivateKey []byte, algorithm PQV2Algorithm) ([]byte, error) {
	pq.mu.Lock()
	defer pq.mu.Unlock()

	classicKey := make([]byte, 32)
	copy(classicKey, encryptedData.EncryptedKey)

	quantumKey := make([]byte, 32)
	copy(quantumKey, encryptedData.EncryptedKey)

	return pq.hybridEngine.Decrypt(encryptedData, quantumKey, classicKey)
}

func (pq *PostQuantumV2) PerformHandshakeV2(clientPublicKey []byte, algorithm PQV2Algorithm) (*PQV2HandshakeResult, error) {
	start := time.Now()

	serverKP, err := pq.kyberEngine.GenerateKeyPair(algorithm)
	if err != nil {
		return nil, err
	}

	ciphertext, sharedSecret, err := pq.kyberEngine.Encapsulate(clientPublicKey, algorithm)
	if err != nil {
		return nil, err
	}

	sessionKey := sharedSecret.Data

	return &PQV2HandshakeResult{
		SessionKey:     sessionKey,
		SharedSecret:   sharedSecret,
		Ciphertext:     ciphertext,
		PublicKey:      serverKP.PublicKey,
		ProtocolVersion: pq.version,
		Duration:       time.Since(start),
	}, nil
}

func (pq *PostQuantumV2) SignDataV2(message []byte, privateKey []byte, algorithm PQV2Algorithm) (*PQV2Signature, error) {
	return pq.dilithiumEngine.Sign(message, privateKey, algorithm)
}

func (pq *PostQuantumV2) VerifySignatureV2(message []byte, signature *PQV2Signature, publicKey []byte, algorithm PQV2Algorithm) (bool, error) {
	return pq.dilithiumEngine.Verify(message, signature, publicKey, algorithm)
}

func (pq *PostQuantumV2) DeriveKeyV2(secret []byte, purpose string, length int) ([]byte, error) {
	info := purpose + ":" + pq.version.String()

	h := hmac.New(sha256.New, secret)
	h.Write([]byte(info))

	result := make([]byte, length)
	copy(result, h.Sum(nil))

	return result, nil
}

func (pq *PostQuantumV2) SerializeKeyPairV2(keyPair *PQV2KeyPair) (string, error) {
	data, err := json.Marshal(map[string]interface{}{
		"public_key": base64.StdEncoding.EncodeToString(keyPair.PublicKey),
		"private_key": base64.StdEncoding.EncodeToString(keyPair.PrivateKey),
		"algorithm":  keyPair.Algorithm,
		"created_at": keyPair.CreatedAt,
		"expires_at": keyPair.ExpiresAt,
		"key_id":     keyPair.KeyID,
	})
	if err != nil {
		return "", fmt.Errorf("failed to serialize key pair: %w", err)
	}
	return string(data), nil
}

func (pq *PostQuantumV2) DeserializeKeyPairV2(data string) (*PQV2KeyPair, error) {
	var parsed struct {
		PublicKey  string `json:"public_key"`
		PrivateKey string `json:"private_key"`
		Algorithm  string `json:"algorithm"`
		CreatedAt  string `json:"created_at"`
		ExpiresAt  string `json:"expires_at"`
		KeyID      string `json:"key_id"`
	}
	if err := json.Unmarshal([]byte(data), &parsed); err != nil {
		return nil, fmt.Errorf("failed to deserialize key pair: %w", err)
	}

	pkBytes, err := base64.StdEncoding.DecodeString(parsed.PublicKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode public key: %w", err)
	}

	skBytes, err := base64.StdEncoding.DecodeString(parsed.PrivateKey)
	if err != nil {
		return nil, fmt.Errorf("failed to decode private key: %w", err)
	}

	createdAt, _ := time.Parse(time.RFC3339, parsed.CreatedAt)
	expiresAt, _ := time.Parse(time.RFC3339, parsed.ExpiresAt)

	return &PQV2KeyPair{
		PublicKey:  pkBytes,
		PrivateKey: skBytes,
		Algorithm:  PQV2Algorithm(parsed.Algorithm),
		CreatedAt:  createdAt,
		ExpiresAt:  expiresAt,
		KeyID:      parsed.KeyID,
	}, nil
}

func (v PQV2ProtocolVersion) String() string {
	return fmt.Sprintf("v%d.%d.%d", v.Major, v.Minor, v.Patch)
}


type PQV2EncryptionRequest struct {
	Plaintext     []byte
	PublicKey     []byte
	Algorithm     PQV2Algorithm
	HybridScheme  string
}

type PQV2EncryptionResponse struct {
	Success      bool
	EncryptedData *PQV2EncryptedData
	ErrorMessage string
}

type PQV2DecryptionRequest struct {
	EncryptedData *PQV2EncryptedData
	PrivateKey    []byte
	Algorithm     PQV2Algorithm
}

type PQV2DecryptionResponse struct {
	Success     bool
	Plaintext   []byte
	ErrorMessage string
}

func (pq *PostQuantumV2) EncryptV2(ctx context.Context, req *PQV2EncryptionRequest) (*PQV2EncryptionResponse, error) {
	encryptedData, err := pq.HybridEncryptV2(req.Plaintext, req.PublicKey, req.Algorithm)
	if err != nil {
		return &PQV2EncryptionResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	return &PQV2EncryptionResponse{
		Success:       true,
		EncryptedData: encryptedData,
	}, nil
}

func (pq *PostQuantumV2) DecryptV2(ctx context.Context, req *PQV2DecryptionRequest) (*PQV2DecryptionResponse, error) {
	plaintext, err := pq.HybridDecryptV2(req.EncryptedData, req.PrivateKey, req.Algorithm)
	if err != nil {
		return &PQV2DecryptionResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	return &PQV2DecryptionResponse{
		Success:   true,
		Plaintext: plaintext,
	}, nil
}

type PQV2KeyGenerationRequest struct {
	Algorithm PQV2Algorithm
	KeyType   string
	Namespace string
}

type PQV2KeyGenerationResponse struct {
	Success     bool
	KeyPair     *PQV2KeyPair
	KeyStore    string
	ErrorMessage string
}

func (pq *PostQuantumV2) GenerateKeyV2(ctx context.Context, req *PQV2KeyGenerationRequest) (*PQV2KeyGenerationResponse, error) {
	var keyPair *PQV2KeyPair
	var err error

	switch req.KeyType {
	case "signing":
		keyPair, err = pq.GenerateDilithiumKeyPairV2(req.Algorithm)
	default:
		keyPair, err = pq.GenerateKyberKeyPairV2(req.Algorithm)
	}

	if err != nil {
		return &PQV2KeyGenerationResponse{
			Success:      false,
			ErrorMessage: err.Error(),
		}, err
	}

	return &PQV2KeyGenerationResponse{
		Success:  true,
		KeyPair: keyPair,
		KeyStore: req.Namespace,
	}, nil
}

func (pq *PostQuantumV2) GetKeyInfoV2(ctx context.Context, keyID string) (*PQV2KeyMetadata, error) {
	_, metadata, err := pq.keyStore.GetKey(keyID)
	if err != nil {
		return nil, err
	}
	return metadata, nil
}

func (pq *PostQuantumV2) RotateKeyV2(ctx context.Context, namespace, keyID string) (*PQV2KeyRotation, error) {
	return pq.keyManager.RotateKey(namespace, keyID)
}

func (pq *PostQuantumV2) GetSecurityLevelV2() PQV2SecurityLevel {
	return PQV2Security192
}

func (pq *PostQuantumV2) GetVersionV2() PQV2ProtocolVersion {
	return pq.version
}
