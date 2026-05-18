package tools

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/elliptic"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha256"
	"crypto/x509"
	"encoding/base64"
	"encoding/pem"
	"fmt"
	"io"
	mrand "math/rand"
	"strings"
	"sync"
	"time"
)

type KeyManager struct {
	currentKey       []byte
	keyHistory       []KeyRecord
	keyRotationTime  time.Duration
	lastRotation     time.Time
	mu               sync.RWMutex
	rsaPrivateKey    *rsa.PrivateKey
	rsaPublicKey     *rsa.PublicKey
}

type KeyRecord struct {
	Key       []byte    `json:"key"`
	CreatedAt time.Time `json:"created_at"`
	ExpiresAt time.Time `json:"expires_at"`
	Version   int       `json:"version"`
}

type KeyAlgorithm string

const (
	KeyAlgorithmAES256 KeyAlgorithm = "aes-256-gcm"
	KeyAlgorithmRSA2048 KeyAlgorithm = "rsa-2048"
	KeyAlgorithmChaCha20 KeyAlgorithm = "chacha20-poly1305"
)

func NewKeyManager(rotationInterval time.Duration) *KeyManager {
	km := &KeyManager{
		keyRotationTime: rotationInterval,
		lastRotation:    time.Now(),
		keyHistory:      make([]KeyRecord, 0),
	}

	key, err := km.generateKey(32)
	if err != nil {
		key = []byte("hjtpx-default-key-32-bytes-xx")
	}

	km.currentKey = key
	km.addKeyRecord(key)

	if err := km.generateRSAKeyPair(); err != nil {
		fmt.Printf("Warning: Failed to generate RSA key pair: %v\n", err)
	}

	return km
}

func (km *KeyManager) generateKey(length int) ([]byte, error) {
	key := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return nil, fmt.Errorf("failed to generate key: %w", err)
	}
	return key, nil
}

func (km *KeyManager) addKeyRecord(key []byte) {
	km.mu.Lock()
	defer km.mu.Unlock()

	record := KeyRecord{
		Key:       make([]byte, len(key)),
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(km.keyRotationTime),
		Version:   len(km.keyHistory) + 1,
	}
	copy(record.Key, key)

	km.keyHistory = append(km.keyHistory, record)

	if len(km.keyHistory) > 10 {
		km.keyHistory = km.keyHistory[len(km.keyHistory)-10:]
	}
}

func (km *KeyManager) generateRSAKeyPair() error {
	var err error
	km.rsaPrivateKey, err = rsa.GenerateKey(rand.Reader, 2048)
	if err != nil {
		return fmt.Errorf("failed to generate RSA key pair: %w", err)
	}
	km.rsaPublicKey = &km.rsaPrivateKey.PublicKey
	return nil
}

func (km *KeyManager) GetCurrentKey() []byte {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if time.Since(km.lastRotation) > km.keyRotationTime {
		go km.rotateKey()
	}

	key := make([]byte, len(km.currentKey))
	copy(key, km.currentKey)
	return key
}

func (km *KeyManager) RotateKey() error {
	km.mu.Lock()
	defer km.mu.Unlock()

	return km.rotateKey()
}

func (km *KeyManager) rotateKey() error {
	newKey, err := km.generateKey(32)
	if err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	km.currentKey = newKey
	km.lastRotation = time.Now()
	km.addKeyRecord(newKey)

	return nil
}

func (km *KeyManager) GetKeyVersion() int {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if len(km.keyHistory) == 0 {
		return 0
	}
	return km.keyHistory[len(km.keyHistory)-1].Version
}

func (km *KeyManager) GetKeyHistory() []KeyRecord {
	km.mu.RLock()
	defer km.mu.RUnlock()

	history := make([]KeyRecord, len(km.keyHistory))
	for i, record := range km.keyHistory {
		history[i] = KeyRecord{
			CreatedAt: record.CreatedAt,
			ExpiresAt: record.ExpiresAt,
			Version:   record.Version,
			Key:       make([]byte, len(record.Key)),
		}
		copy(history[i].Key, record.Key)
	}

	return history
}

func (km *KeyManager) DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(salt) == 0 {
		salt = make([]byte, 16)
		io.ReadFull(rand.Reader, salt)
	}

	km.mu.RLock()
	currentKey := make([]byte, len(km.currentKey))
	copy(currentKey, km.currentKey)
	km.mu.RUnlock()

	derived := sha256.New()
	derived.Write(currentKey)
	derived.Write([]byte(password))
	derived.Write(salt)

	return derived.Sum(nil), nil
}

func (km *KeyManager) EncryptWithCurrentKey(plaintext []byte) ([]byte, error) {
	key := km.GetCurrentKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	io.ReadFull(rand.Reader, nonce)

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (km *KeyManager) DecryptWithCurrentKey(ciphertext []byte) ([]byte, error) {
	key := km.GetCurrentKey()

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (km *KeyManager) EncryptString(plaintext string) (string, error) {
	encrypted, err := km.EncryptWithCurrentKey([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(encrypted), nil
}

func (km *KeyManager) DecryptString(ciphertext string) (string, error) {
	data, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	decrypted, err := km.DecryptWithCurrentKey(data)
	if err != nil {
		return "", err
	}

	return string(decrypted), nil
}

func (km *KeyManager) EncryptWithRSA(plaintext []byte) ([]byte, error) {
	km.mu.RLock()
	publicKey := km.rsaPublicKey
	km.mu.RUnlock()

	if publicKey == nil {
		return nil, fmt.Errorf("RSA public key not initialized")
	}

	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, plaintext, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA encryption failed: %w", err)
	}

	return ciphertext, nil
}

func (km *KeyManager) DecryptWithRSA(ciphertext []byte) ([]byte, error) {
	km.mu.RLock()
	privateKey := km.rsaPrivateKey
	km.mu.RUnlock()

	if privateKey == nil {
		return nil, fmt.Errorf("RSA private key not initialized")
	}

	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("RSA decryption failed: %w", err)
	}

	return plaintext, nil
}

func (km *KeyManager) GetRSAPublicKeyPEM() (string, error) {
	km.mu.RLock()
	pubKey := km.rsaPublicKey
	km.mu.RUnlock()

	if pubKey == nil {
		return "", fmt.Errorf("RSA public key not initialized")
	}

	pubKeyBytes, err := x509.MarshalPKIXPublicKey(pubKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}

	pubKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: pubKeyBytes,
	})

	return string(pubKeyPEM), nil
}

func (km *KeyManager) GetRSAPrivateKeyPEM() (string, error) {
	km.mu.RLock()
	privKey := km.rsaPrivateKey
	km.mu.RUnlock()

	if privKey == nil {
		return "", fmt.Errorf("RSA private key not initialized")
	}

	privKeyBytes := x509.MarshalPKCS1PrivateKey(privKey)
	privKeyPEM := pem.EncodeToMemory(&pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: privKeyBytes,
	})

	return string(privKeyPEM), nil
}

func (km *KeyManager) SetRSAKeys(privateKeyPEM, publicKeyPEM string) error {
	privKey, err := parsePrivateKeyPEM(privateKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse private key: %w", err)
	}

	pubKey, err := parsePublicKeyPEM(publicKeyPEM)
	if err != nil {
		return fmt.Errorf("failed to parse public key: %w", err)
	}

	km.mu.Lock()
	defer km.mu.Unlock()

	km.rsaPrivateKey = privKey
	km.rsaPublicKey = pubKey

	return nil
}

func parsePrivateKeyPEM(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	priv, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	return priv, nil
}

func parsePublicKeyPEM(pemData string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, fmt.Errorf("failed to decode PEM block")
	}

	pub, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, err
	}

	rsaPub, ok := pub.(*rsa.PublicKey)
	if !ok {
		return nil, fmt.Errorf("not an RSA public key")
	}

	return rsaPub, nil
}

func (km *KeyManager) GenerateSymmetricKey(algorithm KeyAlgorithm) ([]byte, error) {
	var keySize int

	switch algorithm {
	case KeyAlgorithmAES256:
		keySize = 32
	case KeyAlgorithmRSA2048:
		keySize = 256
	case KeyAlgorithmChaCha20:
		keySize = 32
	default:
		keySize = 32
	}

	return km.generateKey(keySize)
}

func (km *KeyManager) GenerateECDHKey() ([]byte, []byte, error) {
	privateKey, x, y, err := elliptic.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDH private key: %w", err)
	}
	
	publicKey := elliptic.Marshal(elliptic.P256(), x, y)
	return privateKey, publicKey, nil
}

func (km *KeyManager) DeriveECDHKey(privateKey, peerPublicKey []byte) ([]byte, error) {
	if len(privateKey) == 0 || len(peerPublicKey) == 0 {
		return nil, fmt.Errorf("invalid key parameters")
	}
	
	x, y := elliptic.Unmarshal(elliptic.P256(), peerPublicKey)
	if x == nil || y == nil {
		return nil, fmt.Errorf("failed to unmarshal peer public key")
	}
	
	sharedX, _ := elliptic.P256().ScalarMult(x, y, privateKey)
	if sharedX == nil {
		return nil, fmt.Errorf("failed to compute shared key")
	}
	
	return sharedX.Bytes(), nil
}

func (km *KeyManager) CreateKeyBundle() map[string]interface{} {
	km.mu.RLock()
	defer km.mu.RUnlock()

	bundle := make(map[string]interface{})
	bundle["version"] = km.GetKeyVersion()
	bundle["rotation_interval"] = km.keyRotationTime.String()
	bundle["last_rotation"] = km.lastRotation.Unix()
	bundle["current_key_hash"] = fmt.Sprintf("%x", sha256.Sum256(km.currentKey))

	if km.rsaPublicKey != nil {
		bundle["has_rsa_keys"] = true
	} else {
		bundle["has_rsa_keys"] = false
	}

	bundle["key_history_count"] = len(km.keyHistory)

	return bundle
}

func (km *KeyManager) ValidateKey(key []byte) bool {
	km.mu.RLock()
	defer km.mu.RUnlock()

	if len(key) != len(km.currentKey) {
		return false
	}

	for i := range key {
		if key[i] != km.currentKey[i] {
			return false
		}
	}

	return true
}

func (km *KeyManager) GetKeyInfo() map[string]interface{} {
	km.mu.RLock()
	defer km.mu.RUnlock()

	info := make(map[string]interface{})
	info["key_length"] = len(km.currentKey)
	info["key_version"] = km.GetKeyVersion()
	info["last_rotation"] = km.lastRotation
	info["next_rotation"] = km.lastRotation.Add(km.keyRotationTime)
	info["time_until_rotation"] = time.Until(km.lastRotation.Add(km.keyRotationTime))
	info["history_size"] = len(km.keyHistory)

	return info
}

func (km *KeyManager) SetRotationInterval(interval time.Duration) {
	km.mu.Lock()
	defer km.mu.Unlock()
	km.keyRotationTime = interval
}

type KeyRotationScheduler struct {
	keyManager    *KeyManager
	stopChan      chan struct{}
	isRunning     bool
	mu            sync.Mutex
}

func NewKeyRotationScheduler(km *KeyManager) *KeyRotationScheduler {
	return &KeyRotationScheduler{
		keyManager: km,
		stopChan:   make(chan struct{}),
		isRunning:  false,
	}
}

func (krs *KeyRotationScheduler) Start(interval time.Duration) {
	krs.mu.Lock()
	defer krs.mu.Unlock()

	if krs.isRunning {
		return
	}

	krs.isRunning = true
	krs.stopChan = make(chan struct{})

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-ticker.C:
				krs.keyManager.RotateKey()
			case <-krs.stopChan:
				return
			}
		}
	}()
}

func (krs *KeyRotationScheduler) Stop() {
	krs.mu.Lock()
	defer krs.mu.Unlock()

	if krs.isRunning {
		close(krs.stopChan)
		krs.isRunning = false
	}
}

func GenerateRandomPassword(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789!@#$%^&*"
	
	b := make([]byte, length)
	for i := range b {
		b[i] = charset[mrand.Intn(len(charset))]
	}
	
	return string(b)
}

func GenerateSalt(length int) ([]byte, error) {
	return generateRandomBytes(length)
}

func generateRandomBytes(length int) ([]byte, error) {
	b := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, b)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func MaskKey(key []byte, visibleChars int) string {
	if len(key) <= visibleChars {
		return strings.Repeat("*", len(key))
	}
	return string(key[:visibleChars]) + strings.Repeat("*", len(key)-visibleChars)
}
