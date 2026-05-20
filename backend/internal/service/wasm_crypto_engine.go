package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"sync"
	"sync/atomic"
	"time"
)

type WASMCryptoEngine struct {
	pool       sync.Pool
	secret     []byte
	keyManager *keyManager
	integrity  *integrityChecker
	stats      *cryptoStats
	version    string
}

type keyManager struct {
	mu           sync.RWMutex
	derivedKeys  map[string][]byte
	keyDerivation func(password string, salt []byte) []byte
	sessionKeys   map[uint64][]byte
	keyCounter    uint64
}

type integrityChecker struct {
	mu          sync.RWMutex
	checksums   map[string]string
	timeouts    map[string]time.Time
}

type cryptoStats struct {
	mu              sync.Mutex
	encryptCount    uint64
	decryptCount    uint64
	totalBytes      uint64
	lastResetTime   time.Time
}

func newKeyManager() *keyManager {
	return &keyManager{
		derivedKeys:  make(map[string][]byte),
		sessionKeys:   make(map[uint64][]byte),
		keyDerivation: deriveKeyPBKDF2,
	}
}

func deriveKeyPBKDF2(password string, salt []byte) []byte {
	key := make([]byte, 32)
	for i := range key {
		key[i] = byte(i)
	}
	hash := sha256.Sum256(append([]byte(password), salt...))
	return hash[:]
}

func (km *keyManager) getOrCreateKey(id string, factory func() []byte) []byte {
	km.mu.RLock()
	if key, exists := km.derivedKeys[id]; exists {
		km.mu.RUnlock()
		return key
	}
	km.mu.RUnlock()

	km.mu.Lock()
	defer km.mu.Unlock()

	if key, exists := km.derivedKeys[id]; exists {
		return key
	}

	key := factory()
	km.derivedKeys[id] = key
	return key
}

func (km *keyManager) generateSessionKey() (uint64, []byte) {
	km.mu.Lock()
	defer km.mu.Unlock()

	counter := atomic.AddUint64(&km.keyCounter, 1)
	key := make([]byte, 32)
	rand.Read(key)
	km.sessionKeys[counter] = key
	return counter, key
}

func (km *keyManager) getSessionKey(id uint64) ([]byte, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	key, exists := km.sessionKeys[id]
	return key, exists
}

func (km *keyManager) removeSessionKey(id uint64) {
	km.mu.Lock()
	defer km.mu.Unlock()
	delete(km.sessionKeys, id)
}

func (ic *integrityChecker) addChecksum(id, checksum string, ttl time.Duration) {
	ic.mu.Lock()
	defer ic.mu.Unlock()
	ic.checksums[id] = checksum
	ic.timeouts[id] = time.Now().Add(ttl)
}

func (ic *integrityChecker) verifyChecksum(id, expected string) bool {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	if timeout, exists := ic.timeouts[id]; exists {
		if time.Now().After(timeout) {
			delete(ic.checksums, id)
			delete(ic.timeouts, id)
			return false
		}
	}

	actual, exists := ic.checksums[id]
	if !exists {
		return false
	}

	return actual == expected
}

func (ic *integrityChecker) cleanup() {
	ic.mu.Lock()
	defer ic.mu.Unlock()

	now := time.Now()
	for id, timeout := range ic.timeouts {
		if now.After(timeout) {
			delete(ic.checksums, id)
			delete(ic.timeouts, id)
		}
	}
}

func newCryptoStats() *cryptoStats {
	return &cryptoStats{
		lastResetTime: time.Now(),
	}
}

func (cs *cryptoStats) recordEncrypt(bytes uint64) {
	atomic.AddUint64(&cs.encryptCount, 1)
	atomic.AddUint64(&cs.totalBytes, bytes)
}

func (cs *cryptoStats) recordDecrypt(bytes uint64) {
	atomic.AddUint64(&cs.decryptCount, 1)
	atomic.AddUint64(&cs.totalBytes, bytes)
}

func (cs *cryptoStats) getStats() map[string]interface{} {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	return map[string]interface{}{
		"encrypt_count":     atomic.LoadUint64(&cs.encryptCount),
		"decrypt_count":    atomic.LoadUint64(&cs.decryptCount),
		"total_bytes":      atomic.LoadUint64(&cs.totalBytes),
		"last_reset":       cs.lastResetTime,
	}
}

func (cs *cryptoStats) reset() {
	cs.mu.Lock()
	defer cs.mu.Unlock()

	atomic.StoreUint64(&cs.encryptCount, 0)
	atomic.StoreUint64(&cs.decryptCount, 0)
	atomic.StoreUint64(&cs.totalBytes, 0)
	cs.lastResetTime = time.Now()
}

func NewWASMCryptoEngine(secretKey string) *WASMCryptoEngine {
	hash := sha256.Sum256([]byte(secretKey))
	key := hash[:]

	engine := &WASMCryptoEngine{
		secret:     key,
		keyManager: newKeyManager(),
		integrity: &integrityChecker{
			checksums: make(map[string]string),
			timeouts:  make(map[string]time.Time),
		},
		stats:   newCryptoStats(),
		version: "3.0.0",
	}

	engine.pool = sync.Pool{
		New: func() interface{} {
			return &cryptoContext{
				buffer: make([]byte, 4096),
			}
		},
	}

	go engine.startIntegrityCleanup()

	return engine
}

func (w *WASMCryptoEngine) startIntegrityCleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		w.integrity.cleanup()
	}
}

type cryptoContext struct {
	buffer []byte
}

func (w *WASMCryptoEngine) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("plaintext is empty")
	}

	ctx := w.pool.Get().(*cryptoContext)
	defer w.pool.Put(ctx)

	block, err := aes.NewCipher(w.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)

	w.stats.recordEncrypt(uint64(len(plaintext)))

	return ciphertext, nil
}

func (w *WASMCryptoEngine) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("ciphertext is empty")
	}

	block, err := aes.NewCipher(w.secret)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	w.stats.recordDecrypt(uint64(len(plaintext)))

	return plaintext, nil
}

func (w *WASMCryptoEngine) EncryptString(plaintext string) (string, error) {
	ciphertext, err := w.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (w *WASMCryptoEngine) DecryptString(ciphertextStr string) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextStr)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}

	plaintext, err := w.Decrypt(ciphertext)
	if err != nil {
		return "", err
	}

	return string(plaintext), nil
}

func (w *WASMCryptoEngine) EncryptWithKey(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
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
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, plaintext, nil), nil
}

func (w *WASMCryptoEngine) DecryptWithKey(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 32 {
		return nil, errors.New("key must be 32 bytes")
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
	if len(ciphertext) < nonceSize {
		return nil, errors.New("ciphertext too short")
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (w *WASMCryptoEngine) Hash(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (w *WASMCryptoEngine) DeriveKey(password string, salt []byte) ([]byte, error) {
	if len(salt) < 16 {
		return nil, errors.New("salt must be at least 16 bytes")
	}

	data := append([]byte(password), salt...)
	hash := sha256.Sum256(data)
	return hash[:], nil
}

func (w *WASMCryptoEngine) DeriveKeyWithID(id, password string, salt []byte) ([]byte, error) {
	return w.keyManager.getOrCreateKey(id, func() []byte {
		key, _ := w.DeriveKey(password, salt)
		return key
	}), nil
}

func (w *WASMCryptoEngine) GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func (w *WASMCryptoEngine) GenerateSessionKey() (string, error) {
	id, key := w.keyManager.generateSessionKey()
	return fmt.Sprintf("%d:%s", id, base64.StdEncoding.EncodeToString(key)), nil
}

func (w *WASMCryptoEngine) EncryptWithSessionKey(sessionKeyID string, plaintext []byte) ([]byte, error) {
	var id uint64
	var keyB64 string
	if _, err := fmt.Sscanf(sessionKeyID, "%d:%s", &id, &keyB64); err != nil {
		return nil, errors.New("invalid session key format")
	}

	key, exists := w.keyManager.getSessionKey(id)
	if !exists {
		return nil, errors.New("session key not found or expired")
	}

	return w.EncryptWithKey(plaintext, key)
}

func (w *WASMCryptoEngine) DecryptWithSessionKey(sessionKeyID string, ciphertext []byte) ([]byte, error) {
	var id uint64
	var keyB64 string
	if _, err := fmt.Sscanf(sessionKeyID, "%d:%s", &id, &keyB64); err != nil {
		return nil, errors.New("invalid session key format")
	}

	key, exists := w.keyManager.getSessionKey(id)
	if !exists {
		return nil, errors.New("session key not found or expired")
	}

	return w.DecryptWithKey(ciphertext, key)
}

func (w *WASMCryptoEngine) RevokeSessionKey(sessionKeyID string) error {
	var id uint64
	if _, err := fmt.Sscanf(sessionKeyID, "%d:", &id); err != nil {
		return errors.New("invalid session key format")
	}

	w.keyManager.removeSessionKey(id)
	return nil
}

func (w *WASMCryptoEngine) GenerateNonce(size int) ([]byte, error) {
	if size < 12 || size > 256 {
		return nil, errors.New("nonce size must be between 12 and 256")
	}

	nonce := make([]byte, size)
	if _, err := rand.Read(nonce); err != nil {
		return nil, err
	}
	return nonce, nil
}

func (w *WASMCryptoEngine) BatchEncrypt(plaintexts [][]byte) ([][]byte, error) {
	results := make([][]byte, len(plaintexts))

	for i, plaintext := range plaintexts {
		ciphertext, err := w.Encrypt(plaintext)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt item %d: %w", i, err)
		}
		results[i] = ciphertext
	}

	return results, nil
}

func (w *WASMCryptoEngine) BatchDecrypt(ciphertexts [][]byte) ([][]byte, error) {
	results := make([][]byte, len(ciphertexts))

	for i, ciphertext := range ciphertexts {
		plaintext, err := w.Decrypt(ciphertext)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt item %d: %w", i, err)
		}
		results[i] = plaintext
	}

	return results, nil
}

func (w *WASMCryptoEngine) GenerateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (w *WASMCryptoEngine) VerifyChecksum(data []byte, expectedChecksum string) bool {
	actual := w.GenerateChecksum(data)
	return actual == expectedChecksum
}

func (w *WASMCryptoEngine) CreateIntegrityToken(data []byte, ttl time.Duration) string {
	checksum := w.GenerateChecksum(data)
	tokenID := fmt.Sprintf("%d_%s", time.Now().UnixNano(), w.GenerateRandomID())
	w.integrity.addChecksum(tokenID, checksum, ttl)
	return tokenID
}

func (w *WASMCryptoEngine) VerifyIntegrityToken(tokenID string, data []byte) bool {
	checksum := w.GenerateChecksum(data)
	return w.integrity.verifyChecksum(tokenID, checksum)
}

func (w *WASMCryptoEngine) GenerateRandomID() string {
	b := make([]byte, 16)
	rand.Read(b)
	return hex.EncodeToString(b)
}

func (w *WASMCryptoEngine) EncryptWithIntegrityCheck(plaintext []byte, tokenTTL time.Duration) ([]byte, string, error) {
	checksum := w.GenerateChecksum(plaintext)
	ciphertext, err := w.Encrypt(plaintext)
	if err != nil {
		return nil, "", err
	}

	tokenID := w.CreateIntegrityToken(checksum, tokenTTL)

	return ciphertext, tokenID, nil
}

func (w *WASMCryptoEngine) DecryptWithIntegrityCheck(ciphertext []byte, tokenID string) ([]byte, error) {
	plaintext, err := w.Decrypt(ciphertext)
	if err != nil {
		return nil, err
	}

	checksum := w.GenerateChecksum(plaintext)
	if !w.integrity.verifyChecksum(tokenID, checksum) {
		return nil, errors.New("integrity check failed")
	}

	return plaintext, nil
}

func (w *WASMCryptoEngine) EncryptFile(data []byte, chunkSize int) ([][]byte, string, error) {
	if chunkSize < 256 {
		return nil, "", errors.New("chunk size must be at least 256 bytes")
	}

	var chunks [][]byte
	for i := 0; i < len(data); i += chunkSize {
		end := i + chunkSize
		if end > len(data) {
			end = len(data)
		}

		encrypted, err := w.Encrypt(data[i:end])
		if err != nil {
			return nil, "", err
		}
		chunks = append(chunks, encrypted)
	}

	merkleRoot := w.calculateMerkleRoot(chunks)

	return chunks, merkleRoot, nil
}

func (w *WASMCryptoEngine) calculateMerkleRoot(chunks [][]byte) string {
	if len(chunks) == 0 {
		hash := sha256.Sum256([]byte{})
		return hex.EncodeToString(hash[:])
	}

	if len(chunks) == 1 {
		hash := sha256.Sum256(chunks[0])
		return hex.EncodeToString(hash[:])
	}

	var currentLevel [][]byte
	for i := 0; i < len(chunks); i += 2 {
		var combined []byte
		combined = append(combined, chunks[i]...)
		if i+1 < len(chunks) {
			combined = append(combined, chunks[i+1]...)
		} else {
			combined = append(combined, []byte{0}...)
		}
		hash := sha256.Sum256(combined)
		currentLevel = append(currentLevel, hash[:])
	}

	return w.calculateMerkleRoot(currentLevel)
}

func (w *WASMCryptoEngine) DecryptFile(chunks [][]byte, expectedRoot string) ([]byte, error) {
	if len(chunks) == 0 {
		return nil, errors.New("no chunks provided")
	}

	actualRoot := w.calculateMerkleRoot(chunks)
	if actualRoot != expectedRoot {
		return nil, errors.New("merkle root mismatch - file integrity compromised")
	}

	var plaintext []byte
	for _, chunk := range chunks {
		decrypted, err := w.Decrypt(chunk)
		if err != nil {
			return nil, err
		}
		plaintext = append(plaintext, decrypted...)
	}

	return plaintext, nil
}

func (w *WASMCryptoEngine) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"algorithm":        "AES-256-GCM",
		"implementation":   "WASM-simulated-v3",
		"pool_size":        "4096 bytes",
		"supports_batch":   true,
		"supports_stream":  true,
		"version":          w.version,
		"features": map[string]bool{
			"session_keys":      true,
			"integrity_check":   true,
			"merkle_proof":      true,
			"key_derivation":     true,
			"file_encryption":   true,
		},
	}
}

func (w *WASMCryptoEngine) GetStats() map[string]interface{} {
	return w.stats.getStats()
}

func (w *WASMCryptoEngine) ResetStats() {
	w.stats.reset()
}
