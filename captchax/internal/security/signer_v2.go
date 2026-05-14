package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type SigningAlgorithm string

const (
	AlgorithmHMACSHA256 SigningAlgorithm = "HMAC-SHA256"
	AlgorithmHMACSHA384 SigningAlgorithm = "HMAC-SHA384"
	AlgorithmHMACSHA512 SigningAlgorithm = "HMAC-SHA512"
)

type SignerConfigV2 struct {
	SecretKey            []byte
	TimestampTolerance   time.Duration
	SignatureHeader      string
	TimestampHeader      string
	NonceHeader          string
	Algorithm            SigningAlgorithm
	AppIDHeader          string
	SignaturePrefix      string
	EnableNonceCheck     bool
	EnableTimestampCheck bool
}

var defaultSignerConfigV2 = &SignerConfigV2{
	TimestampTolerance:   5 * time.Minute,
	SignatureHeader:       "X-Signature",
	TimestampHeader:       "X-Timestamp",
	NonceHeader:           "X-Nonce",
	Algorithm:             AlgorithmHMACSHA256,
	AppIDHeader:           "X-App-ID",
	SignaturePrefix:       "sha256=",
	EnableNonceCheck:      true,
	EnableTimestampCheck:  true,
}

type SignerV2 struct {
	config    *SignerConfigV2
	secretKey []byte
}

type SignatureParamsV2 struct {
	Method      string
	Path        string
	QueryParams url.Values
	Headers     map[string]string
	Body        []byte
	Timestamp   int64
	Nonce       string
	AppID       string
}

func NewSignerV2(secretKey []byte) *SignerV2 {
	return NewSignerV2WithConfig(secretKey, nil)
}

func NewSignerV2WithConfig(secretKey []byte, config *SignerConfigV2) *SignerV2 {
	cfg := defaultSignerConfigV2
	if config != nil {
		cfg = config
	}
	if cfg.TimestampTolerance == 0 {
		cfg.TimestampTolerance = defaultSignerConfigV2.TimestampTolerance
	}
	if cfg.SignatureHeader == "" {
		cfg.SignatureHeader = defaultSignerConfigV2.SignatureHeader
	}
	if cfg.TimestampHeader == "" {
		cfg.TimestampHeader = defaultSignerConfigV2.TimestampHeader
	}
	if cfg.NonceHeader == "" {
		cfg.NonceHeader = defaultSignerConfigV2.NonceHeader
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = defaultSignerConfigV2.Algorithm
	}
	return &SignerV2{
		config:    cfg,
		secretKey: secretKey,
	}
}

func (s *SignerV2) GenerateKey(length int) ([]byte, error) {
	key := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("signer: failed to generate random key: %w", err)
	}
	return key, nil
}

func (s *SignerV2) Sign(data []byte) (string, error) {
	if len(s.secretKey) == 0 {
		return "", errors.New("signer: secret key is empty")
	}

	var mac func() []byte
	switch s.config.Algorithm {
	case AlgorithmHMACSHA256:
		h := hmac.New(sha256.New, s.secretKey)
		h.Write(data)
		mac = func() []byte { return h.Sum(nil) }
	case AlgorithmHMACSHA384:
		h := hmac.New(sha512.New384, s.secretKey)
		h.Write(data)
		mac = func() []byte { return h.Sum(nil) }
	case AlgorithmHMACSHA512:
		h := hmac.New(sha512.New, s.secretKey)
		h.Write(data)
		mac = func() []byte { return h.Sum(nil) }
	default:
		h := hmac.New(sha256.New, s.secretKey)
		h.Write(data)
		mac = func() []byte { return h.Sum(nil) }
	}

	return hex.EncodeToString(mac()), nil
}

func (s *SignerV2) SignString(data string) (string, error) {
	return s.Sign([]byte(data))
}

func (s *SignerV2) Verify(signature string, data []byte) (bool, error) {
	expectedSig, err := s.Sign(data)
	if err != nil {
		return false, err
	}

	sigToCompare := signature
	if strings.HasPrefix(sigToCompare, s.config.SignaturePrefix) {
		sigToCompare = strings.TrimPrefix(sigToCompare, s.config.SignaturePrefix)
	}

	return subtle.ConstantTimeCompare([]byte(sigToCompare), []byte(expectedSig)) == 1, nil
}

func (s *SignerV2) SignRequest(params *SignatureParamsV2) (string, string, error) {
	if params.Timestamp == 0 {
		params.Timestamp = time.Now().Unix()
	}
	if params.Nonce == "" {
		nonce, err := GenerateNonceV2(32)
		if err != nil {
			return "", "", fmt.Errorf("signer: failed to generate nonce: %w", err)
		}
		params.Nonce = nonce
	}

	signatureData, err := s.BuildSignatureData(params)
	if err != nil {
		return "", "", fmt.Errorf("signer: failed to build signature data: %w", err)
	}

	signature, err := s.Sign([]byte(signatureData))
	if err != nil {
		return "", "", fmt.Errorf("signer: failed to sign: %w", err)
	}

	return signature, params.Nonce, nil
}

func (s *SignerV2) BuildSignatureData(params *SignatureParamsV2) (string, error) {
	var parts []string

	parts = append(parts, strings.ToUpper(params.Method))
	parts = append(parts, params.Path)

	if params.QueryParams != nil {
		sortedKeys := make([]string, 0, len(params.QueryParams))
		for k := range params.QueryParams {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		var queryParts []string
		for _, k := range sortedKeys {
			values := params.QueryParams[k]
			for _, v := range values {
				queryParts = append(queryParts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
			}
		}
		parts = append(parts, strings.Join(queryParts, "&"))
	} else {
		parts = append(parts, "")
	}

	if params.AppID != "" {
		parts = append(parts, params.AppID)
	}

	parts = append(parts, strconv.FormatInt(params.Timestamp, 10))

	if params.Nonce != "" {
		parts = append(parts, params.Nonce)
	}

	if len(params.Body) > 0 {
		h := sha256.New()
		h.Write(params.Body)
		parts = append(parts, hex.EncodeToString(h.Sum(nil)))
	}

	return strings.Join(parts, "\n"), nil
}

func GenerateNonceV2(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789-_"
	nonce := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randNum, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("signer: failed to generate random number: %w", err)
		}
		nonce[i] = charset[randNum.Int64()]
	}

	return string(nonce), nil
}

type SignatureValidatorV2 struct {
	config      *SignerConfigV2
	nonceCache  *NonceCache
	keyManager  *KeyManager
}

type NonceCache struct {
	mu      sync.RWMutex
	entries map[string]time.Time
	maxAge  time.Duration
}

func NewNonceCache(maxAge time.Duration) *NonceCache {
	nc := &NonceCache{
		entries: make(map[string]time.Time),
		maxAge:  maxAge,
	}
	go nc.cleanup()
	return nc
}

func (nc *NonceCache) IsUsed(nonce string) bool {
	nc.mu.RLock()
	defer nc.mu.RUnlock()
	_, exists := nc.entries[nonce]
	return exists
}

func (nc *NonceCache) MarkUsed(nonce string) {
	nc.mu.Lock()
	defer nc.mu.Unlock()
	nc.entries[nonce] = time.Now()
}

func (nc *NonceCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	for range ticker.C {
		nc.mu.Lock()
		now := time.Now()
		for nonce, t := range nc.entries {
			if now.Sub(t) > nc.maxAge {
				delete(nc.entries, nonce)
			}
		}
		nc.mu.Unlock()
	}
}

type KeyManager struct {
	mu            sync.RWMutex
	currentKey    []byte
	previousKey   []byte
	keyVersion    int
	keyMetadata   map[int]*KeyMetadata
	rotationTime  time.Duration
	lastRotation  time.Time
}

type KeyMetadata struct {
	Version     int
	Key         []byte
	CreatedAt   time.Time
	ExpiresAt   time.Time
	IsActive    bool
}

func NewKeyManager(initialKey []byte, rotationTime time.Duration) *KeyManager {
	km := &KeyManager{
		currentKey:   initialKey,
		keyVersion:   1,
		keyMetadata:  make(map[int]*KeyMetadata),
		rotationTime: rotationTime,
		lastRotation:  time.Now(),
	}
	km.keyMetadata[1] = &KeyMetadata{
		Version:   1,
		Key:       initialKey,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(rotationTime * 30),
		IsActive:  true,
	}
	if rotationTime > 0 {
		go km.autoRotate()
	}
	return km
}

func (km *KeyManager) GetCurrentKey() []byte {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.currentKey
}

func (km *KeyManager) GetKeyByVersion(version int) ([]byte, bool) {
	km.mu.RLock()
	defer km.mu.RUnlock()
	meta, exists := km.keyMetadata[version]
	if !exists || !meta.IsActive {
		return nil, false
	}
	return meta.Key, true
}

func (km *KeyManager) GetCurrentVersion() int {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return km.keyVersion
}

func (km *KeyManager) Rotate(newKey []byte) error {
	km.mu.Lock()
	defer km.mu.Unlock()

	if km.keyMetadata[km.keyVersion] != nil {
		km.keyMetadata[km.keyVersion].IsActive = false
	}

	km.previousKey = km.currentKey
	km.keyVersion++

	km.keyMetadata[km.keyVersion] = &KeyMetadata{
		Version:   km.keyVersion,
		Key:       newKey,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(km.rotationTime * 30),
		IsActive:  true,
	}
	km.currentKey = newKey
	km.lastRotation = time.Now()

	return nil
}

func (km *KeyManager) VerifyWithAnyKey(signature string, data []byte, algorithms []SigningAlgorithm) (bool, error) {
	km.mu.RLock()
	defer km.mu.RUnlock()

	var keysToCheck [][]byte
	if km.currentKey != nil {
		keysToCheck = append(keysToCheck, km.currentKey)
	}
	if km.previousKey != nil {
		keysToCheck = append(keysToCheck, km.previousKey)
	}

	if len(keysToCheck) == 0 {
		return false, errors.New("signer: no keys available")
	}

	if len(algorithms) == 0 {
		algorithms = []SigningAlgorithm{AlgorithmHMACSHA256}
	}

	for _, key := range keysToCheck {
		for _, algo := range algorithms {
			signer := &SignerV2{
				config: &SignerConfigV2{
					Algorithm:      algo,
					SignaturePrefix: "sha256=",
				},
				secretKey: key,
			}
			valid, err := signer.Verify(signature, data)
			if err == nil && valid {
				return true, nil
			}
		}
	}

	return false, nil
}

func (km *KeyManager) autoRotate() {
	ticker := time.NewTicker(km.rotationTime)
	for range ticker.C {
		newKey := make([]byte, 32)
		if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
			continue
		}
		km.Rotate(newKey)
	}
}

func (km *KeyManager) NeedsRotation() bool {
	km.mu.RLock()
	defer km.mu.RUnlock()
	return time.Since(km.lastRotation) >= km.rotationTime
}

func NewSignatureValidatorV2(config *SignerConfigV2, keyManager *KeyManager) *SignatureValidatorV2 {
	if config == nil {
		config = defaultSignerConfigV2
	}
	return &SignatureValidatorV2{
		config:     config,
		nonceCache: NewNonceCache(10 * time.Minute),
		keyManager: keyManager,
	}
}

func (v *SignatureValidatorV2) ValidateSignature(c *gin.Context) (bool, error) {
	signature := c.GetHeader(v.config.SignatureHeader)
	if signature == "" {
		signature = c.GetHeader("X-Signature")
	}
	if signature == "" {
		return false, errors.New("signature: missing signature header")
	}

	timestamp := c.GetHeader(v.config.TimestampHeader)
	if timestamp == "" {
		timestamp = c.GetHeader("X-Timestamp")
	}
	if timestamp == "" && v.config.EnableTimestampCheck {
		return false, errors.New("signature: missing timestamp header")
	}

	if timestamp != "" {
		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			return false, errors.New("signature: invalid timestamp format")
		}

		now := time.Now().Unix()
		toleranceSeconds := int64(v.config.TimestampTolerance.Seconds())
		if ts < now-toleranceSeconds || ts > now+toleranceSeconds {
			return false, fmt.Errorf("signature: timestamp expired or too far in future (tolerance: %v)", v.config.TimestampTolerance)
		}
	}

	nonce := c.GetHeader(v.config.NonceHeader)
	if nonce == "" {
		nonce = c.GetHeader("X-Nonce")
	}
	if nonce != "" && v.config.EnableNonceCheck {
		if v.nonceCache.IsUsed(nonce) {
			return false, errors.New("signature: nonce already used (replay attack detected)")
		}
		v.nonceCache.MarkUsed(nonce)
	}

	appID := c.GetHeader(v.config.AppIDHeader)
	if appID == "" {
		appID = c.GetHeader("X-App-ID")
	}

	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
	}

	params := &SignatureParamsV2{
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Body:      body,
		Timestamp: 0,
		Nonce:     nonce,
		AppID:     appID,
	}
	params.QueryParams = c.Request.URL.Query()

	if timestamp != "" {
		ts, _ := strconv.ParseInt(timestamp, 10, 64)
		params.Timestamp = ts
	}

	signatureData, err := (&SignerV2{config: v.config}).BuildSignatureData(params)
	if err != nil {
		return false, fmt.Errorf("signature: failed to build signature data: %w", err)
	}

	if v.keyManager != nil {
		return v.keyManager.VerifyWithAnyKey(signature, []byte(signatureData), []SigningAlgorithm{v.config.Algorithm})
	}

	signer := &SignerV2{
		config:    v.config,
		secretKey: v.config.SecretKey,
	}
	return signer.Verify(signature, []byte(signatureData))
}

func SignatureMiddlewareV2(keyManager *KeyManager) gin.HandlerFunc {
	return SignatureMiddlewareV2WithConfig(keyManager, nil)
}

func SignatureMiddlewareV2WithConfig(keyManager *KeyManager, config *SignerConfigV2) gin.HandlerFunc {
	cfg := defaultSignerConfigV2
	if config != nil {
		cfg = config
	}
	validator := NewSignatureValidatorV2(cfg, keyManager)

	return func(c *gin.Context) {
		signature := c.GetHeader(cfg.SignatureHeader)
		timestamp := c.GetHeader(cfg.TimestampHeader)

		if signature != "" && timestamp != "" {
			valid, err := validator.ValidateSignature(c)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "unauthorized",
					"message": err.Error(),
				})
				return
			}
			if !valid {
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "unauthorized",
					"message": "invalid signature",
				})
				return
			}
		}

		c.Next()
	}
}

type SignedRequestV2 struct {
	AppID     string            `json:"app_id"`
	Timestamp int64             `json:"timestamp"`
	Nonce     string            `json:"nonce"`
	Signature string            `json:"signature"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Query     map[string]string `json:"query,omitempty"`
	Body      json.RawMessage   `json:"body,omitempty"`
}

func ParseSignedRequestV2(c *gin.Context) (*SignedRequestV2, error) {
	var req SignedRequestV2
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, fmt.Errorf("failed to parse signed request: %w", err)
	}
	return &req, nil
}
