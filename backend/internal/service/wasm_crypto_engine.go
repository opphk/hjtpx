package service

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"errors"
	"fmt"
	"io"
	"sync"
)

type WASMCryptoEngine struct {
	pool   sync.Pool
	secret []byte
}

func NewWASMCryptoEngine(secretKey string) *WASMCryptoEngine {
	hash := sha256.Sum256([]byte(secretKey))
	key := hash[:]

	engine := &WASMCryptoEngine{
		secret: key,
	}

	engine.pool = sync.Pool{
		New: func() interface{} {
			return &cryptoContext{
				buffer: make([]byte, 4096),
			}
		},
	}

	return engine
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
	data := append([]byte(password), salt...)
	hash := sha256.Sum256(data)
	return hash[:], nil
}

func (w *WASMCryptoEngine) GenerateKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, err
	}
	return key, nil
}

func (w *WASMCryptoEngine) GenerateNonce(size int) ([]byte, error) {
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

func (w *WASMCryptoEngine) GetPerformanceMetrics() map[string]interface{} {
	return map[string]interface{}{
		"algorithm":       "AES-256-GCM",
		"implementation":  "WASM-simulated",
		"pool_size":       "4096 bytes",
		"supports_batch":  true,
		"supports_stream": true,
	}
}
