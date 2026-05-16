package crypto

import (
	"crypto/ed25519"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"
	"strconv"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidEd25519Key     = fmt.Errorf("invalid ed25519 key")
	ErrInvalidEd25519Signature = fmt.Errorf("invalid ed25519 signature")
	ErrEd25519SignFailed     = fmt.Errorf("ed25519 signature generation failed")
)

type Ed25519KeyPair struct {
	PrivateKey ed25519.PrivateKey
	PublicKey  ed25519.PublicKey
	KeyID      string
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Version    int
}

func (k *Ed25519KeyPair) IsExpired() bool {
	return time.Now().After(k.ExpiresAt)
}

func (k *Ed25519KeyPair) IsValid() bool {
	if k.PrivateKey == nil || k.PublicKey == nil {
		return false
	}
	if k.IsExpired() {
		return false
	}
	return len(k.PrivateKey) == ed25519.PrivateKeySize && len(k.PublicKey) == ed25519.PublicKeySize
}

func (k *Ed25519KeyPair) ToPEM() (privatePEM, publicPEM string, err error) {
	privateBytes, err := encodeEd25519PrivateKey(k.PrivateKey)
	if err != nil {
		return "", "", err
	}
	
	publicBytes, err := encodeEd25519PublicKey(k.PublicKey)
	if err != nil {
		return "", "", err
	}
	
	return string(privateBytes), string(publicBytes), nil
}

func encodeEd25519PrivateKey(key ed25519.PrivateKey) ([]byte, error) {
	return []byte(fmt.Sprintf("-----BEGIN PRIVATE KEY-----\n%s\n-----END PRIVATE KEY-----",
		base64.StdEncoding.EncodeToString(key))), nil
}

func encodeEd25519PublicKey(key ed25519.PublicKey) ([]byte, error) {
	return []byte(fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----",
		base64.StdEncoding.EncodeToString(key))), nil
}

type Ed25519Manager struct {
	currentKey  *Ed25519KeyPair
	previousKey *Ed25519KeyPair
	keys        map[string]*Ed25519KeyPair
	mu          sync.RWMutex
	keyLifetime time.Duration
	version     int
}

func NewEd25519Manager(keyLifetime time.Duration) (*Ed25519Manager, error) {
	m := &Ed25519Manager{
		keys:        make(map[string]*Ed25519KeyPair),
		keyLifetime: keyLifetime,
		version:     1,
	}
	
	if err := m.generateNewKey(); err != nil {
		return nil, err
	}
	
	go m.autoRotate()
	
	return m, nil
}

func (m *Ed25519Manager) generateNewKey() error {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return fmt.Errorf("failed to generate ed25519 key: %w", err)
	}
	
	keyID := generateKeyID()
	now := time.Now()
	
	newKey := &Ed25519KeyPair{
		PrivateKey: privateKey,
		PublicKey:  publicKey,
		KeyID:      keyID,
		CreatedAt:  now,
		ExpiresAt:  now.Add(m.keyLifetime),
		Version:    m.version,
	}
	
	m.mu.Lock()
	if m.currentKey != nil {
		m.previousKey = m.currentKey
		m.keys[m.previousKey.KeyID] = m.previousKey
	}
	m.currentKey = newKey
	m.version++
	m.mu.Unlock()
	
	return nil
}

func (m *Ed25519Manager) GetCurrentKey() *Ed25519KeyPair {
	m.mu.RLock()
	defer m.mu.RUnlock()
	return m.currentKey
}

func (m *Ed25519Manager) GetPublicKey() ed25519.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentKey == nil {
		return nil
	}
	return m.currentKey.PublicKey
}

func (m *Ed25519Manager) GetKeyID() string {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.currentKey == nil {
		return ""
	}
	return m.currentKey.KeyID
}

func (m *Ed25519Manager) GetAllPublicKeys() []ed25519.PublicKey {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	keys := make([]ed25519.PublicKey, 0, len(m.keys)+1)
	
	if m.currentKey != nil && m.currentKey.PublicKey != nil {
		keys = append(keys, m.currentKey.PublicKey)
	}
	
	for _, k := range m.keys {
		if k.PublicKey != nil {
			keys = append(keys, k.PublicKey)
		}
	}
	
	return keys
}

func (m *Ed25519Manager) Sign(message []byte) (signature []byte, keyID string, err error) {
	m.mu.RLock()
	currentKey := m.currentKey
	m.mu.RUnlock()
	
	if currentKey == nil || !currentKey.IsValid() {
		return nil, "", ErrInvalidEd25519Key
	}
	
	sig := ed25519.Sign(currentKey.PrivateKey, message)
	return sig, currentKey.KeyID, nil
}

func (m *Ed25519Manager) Verify(message, signature []byte) (bool, string) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.currentKey != nil && ed25519.Verify(m.currentKey.PublicKey, message, signature) {
		return true, m.currentKey.KeyID
	}
	
	for keyID, k := range m.keys {
		if k.PublicKey != nil && ed25519.Verify(k.PublicKey, message, signature) {
			return true, keyID
		}
	}
	
	return false, ""
}

func (m *Ed25519Manager) SignString(message string) (string, string, error) {
	sig, keyID, err := m.Sign([]byte(message))
	if err != nil {
		return "", "", err
	}
	return base64.StdEncoding.EncodeToString(sig), keyID, nil
}

func (m *Ed25519Manager) VerifyString(message, signatureBase64 string) (bool, string) {
	sig, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false, ""
	}
	return m.Verify([]byte(message), sig)
}

func (m *Ed25519Manager) Rotate() error {
	return m.generateNewKey()
}

func (m *Ed25519Manager) autoRotate() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	
	for range ticker.C {
		m.mu.RLock()
		shouldRotate := m.currentKey == nil || m.currentKey.ShouldRotate()
		m.mu.RUnlock()
		
		if shouldRotate {
			if err := m.Rotate(); err != nil {
				fmt.Printf("[Ed25519] Auto rotation failed: %v\n", err)
			}
		}
	}
}

func (m *Ed25519Manager) ShouldRotate() bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	if m.currentKey == nil {
		return true
	}
	
	remaining := m.currentKey.ExpiresAt.Sub(time.Now())
	return remaining < m.keyLifetime/4
}

func (k *Ed25519KeyPair) ShouldRotate() bool {
	remaining := k.ExpiresAt.Sub(time.Now())
	return remaining < 0 || remaining < time.Minute*30
}

func (m *Ed25519Manager) GetKeyInfo() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()
	
	info := make(map[string]interface{})
	
	if m.currentKey != nil {
		info["current_key"] = map[string]interface{}{
			"key_id":     m.currentKey.KeyID,
			"created_at": m.currentKey.CreatedAt,
			"expires_at": m.currentKey.ExpiresAt,
			"version":    m.currentKey.Version,
		}
	}
	
	if m.previousKey != nil {
		info["previous_key"] = map[string]interface{}{
			"key_id":     m.previousKey.KeyID,
			"created_at": m.previousKey.CreatedAt,
			"expires_at": m.previousKey.ExpiresAt,
			"version":    m.previousKey.Version,
		}
	}
	
	info["total_keys"] = len(m.keys) + 1
	info["key_lifetime"] = m.keyLifetime.String()
	
	return info
}

func (m *Ed25519Manager) CleanupOldKeys(maxAge time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for keyID, k := range m.keys {
		if k.CreatedAt.Before(cutoff) {
			delete(m.keys, keyID)
		}
	}
}

func generateKeyID() string {
	bytes := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return fmt.Sprintf("key-%d", time.Now().UnixNano())
	}
	return hex.EncodeToString(bytes)
}

func SecureCompareEd25519Keys(a, b ed25519.PublicKey) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare(a, b) == 1
}

type SignatureAlgorithm string

const (
	AlgorithmHMACSHA256  SignatureAlgorithm = "HMAC-SHA256"
	AlgorithmHMACSHA512  SignatureAlgorithm = "HMAC-SHA512"
	AlgorithmEd25519     SignatureAlgorithm = "Ed25519"
	AlgorithmRSA2048     SignatureAlgorithm = "RSA-2048"
	AlgorithmECDSA256    SignatureAlgorithm = "ECDSA-P256"
)

type SignatureRequest struct {
	Algorithm   SignatureAlgorithm
	Message     []byte
	PrivateKey  interface{}
	SecretKey   string
}

type SignatureResponse struct {
	Signature    string `json:"signature"`
	KeyID        string `json:"key_id,omitempty"`
	Algorithm    string `json:"algorithm"`
	Timestamp    int64  `json:"timestamp"`
}

type VerifyRequest struct {
	Algorithm   SignatureAlgorithm
	Message     []byte
	Signature   string
	PublicKey   interface{}
	SecretKey   string
	KeyID       string
}

type VerifyResponse struct {
	Valid      bool   `json:"valid"`
	KeyID      string `json:"key_id,omitempty"`
	Algorithm  string `json:"algorithm"`
	Error      string `json:"error,omitempty"`
}

func Sign(request SignatureRequest) (*SignatureResponse, error) {
	resp := &SignatureResponse{
		Algorithm: string(request.Algorithm),
		Timestamp: time.Now().Unix(),
	}
	
	switch request.Algorithm {
	case AlgorithmHMACSHA256:
		mac := hmac.New(sha256.New, []byte(request.SecretKey))
		mac.Write(request.Message)
		resp.Signature = hex.EncodeToString(mac.Sum(nil))
		
	case AlgorithmHMACSHA512:
		mac := hmac.New(sha512.New, []byte(request.SecretKey))
		mac.Write(request.Message)
		resp.Signature = hex.EncodeToString(mac.Sum(nil))
		
	case AlgorithmEd25519:
		if privateKey, ok := request.PrivateKey.(ed25519.PrivateKey); ok {
			sig := ed25519.Sign(privateKey, request.Message)
			resp.Signature = base64.StdEncoding.EncodeToString(sig)
		} else {
			return nil, ErrInvalidEd25519Key
		}
		
	default:
		return nil, fmt.Errorf("unsupported algorithm: %s", request.Algorithm)
	}
	
	return resp, nil
}

func Verify(request VerifyRequest) *VerifyResponse {
	resp := &VerifyResponse{
		Algorithm: string(request.Algorithm),
	}
	
	switch request.Algorithm {
	case AlgorithmHMACSHA256:
		mac := hmac.New(sha256.New, []byte(request.SecretKey))
		mac.Write(request.Message)
		expected := mac.Sum(nil)
		actual, err := hex.DecodeString(request.Signature)
		if err != nil {
			resp.Error = "invalid signature format"
			return resp
		}
		resp.Valid = hmac.Equal(expected, actual)
		
	case AlgorithmHMACSHA512:
		mac := hmac.New(sha512.New, []byte(request.SecretKey))
		mac.Write(request.Message)
		expected := mac.Sum(nil)
		actual, err := hex.DecodeString(request.Signature)
		if err != nil {
			resp.Error = "invalid signature format"
			return resp
		}
		resp.Valid = hmac.Equal(expected, actual)
		
	case AlgorithmEd25519:
		sig, err := base64.StdEncoding.DecodeString(request.Signature)
		if err != nil {
			resp.Error = "invalid signature format"
			return resp
		}
		if publicKey, ok := request.PublicKey.(ed25519.PublicKey); ok {
			resp.Valid = ed25519.Verify(publicKey, request.Message, sig)
		} else {
			resp.Error = "invalid public key"
			return resp
		}
		
	default:
		resp.Error = fmt.Sprintf("unsupported algorithm: %s", request.Algorithm)
		return resp
	}
	
	return resp
}

func ParseEd25519PrivateKey(pemData string) (ed25519.PrivateKey, error) {
	pemStr := strings.TrimSpace(pemData)
	pemStr = strings.ReplaceAll(pemStr, "-----BEGIN PRIVATE KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "-----END PRIVATE KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "\n", "")
	
	decoded, err := base64.StdEncoding.DecodeString(pemStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PEM: %w", err)
	}
	
	if len(decoded) != ed25519.PrivateKeySize {
		return nil, ErrInvalidEd25519Key
	}
	
	return ed25519.PrivateKey(decoded), nil
}

func ParseEd25519PublicKey(pemData string) (ed25519.PublicKey, error) {
	pemStr := strings.TrimSpace(pemData)
	pemStr = strings.ReplaceAll(pemStr, "-----BEGIN PUBLIC KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "-----END PUBLIC KEY-----", "")
	pemStr = strings.ReplaceAll(pemStr, "\n", "")
	
	decoded, err := base64.StdEncoding.DecodeString(pemStr)
	if err != nil {
		return nil, fmt.Errorf("failed to decode PEM: %w", err)
	}
	
	if len(decoded) != ed25519.PublicKeySize {
		return nil, ErrInvalidEd25519Key
	}
	
	return ed25519.PublicKey(decoded), nil
}

func GenerateEd25519KeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, publicKey, nil
}

func ExportEd25519PrivateKey(privateKey ed25519.PrivateKey) string {
	return fmt.Sprintf("-----BEGIN PRIVATE KEY-----\n%s\n-----END PRIVATE KEY-----",
		base64.StdEncoding.EncodeToString(privateKey))
}

func ExportEd25519PublicKey(publicKey ed25519.PublicKey) string {
	return fmt.Sprintf("-----BEGIN PUBLIC KEY-----\n%s\n-----END PUBLIC KEY-----",
		base64.StdEncoding.EncodeToString(publicKey))
}

func GenerateSignatureWithTimestamp(secretKey string, method, path, query string, timestamp int64, nonce, bodyHash string, algorithm SignatureAlgorithm) (string, error) {
	stringToSign := buildStringToSignV2(method, path, query, timestamp, nonce, bodyHash)
	
	req := SignatureRequest{
		Algorithm: algorithm,
		Message:   []byte(stringToSign),
		SecretKey: secretKey,
	}
	
	resp, err := Sign(req)
	if err != nil {
		return "", err
	}
	
	return resp.Signature, nil
}

func buildStringToSignV2(method, path, query string, timestamp int64, nonce, bodyHash string) string {
	var parts []string
	parts = append(parts, strings.ToUpper(method))
	parts = append(parts, path)
	
	if query != "" {
		parts = append(parts, query)
	}
	
	parts = append(parts, strconv.FormatInt(timestamp, 10))
	
	if nonce != "" {
		parts = append(parts, nonce)
	}
	
	if bodyHash != "" {
		parts = append(parts, bodyHash)
	}
	
	return strings.Join(parts, "\n")
}
