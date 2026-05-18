package middleware

import (
	"bytes"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/base64"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"hash"
	"io"
	"math"
	"math/big"
	"net/url"
	"os"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"golang.org/x/crypto/ed25519"
	"golang.org/x/crypto/chacha20poly1305"
)

type EnhancedSignatureConfig struct {
	SecretKey                  string
	Algorithm                  string
	TimestampTolerance         time.Duration
	RequireTimestamp           bool
	RequireNonce               bool
	NonceCacheTTL              time.Duration
	SignatureHeader            string
	TimestampHeader            string
	NonceHeader                string
	ExcludePaths               []string
	EnableHMAC_SHA512          bool
	EnableDoubleSignature      bool
	EnableSequenceCheck        bool
	MaxSequenceGap             int64
	EnableReplayCache          bool
	ReplayCacheTTL             time.Duration
	MinNonceLength             int
	MaxNonceLength             int
	EnableRateLimitPerIP       bool
	RateLimitPerIPLimit        int
	RateLimitPerIPWindow       time.Duration
	EnableIntegrityCheck       bool
	BodyIntegrityHeader        string
	AdditionalHeaders          []string
	SignatureVersion           string
	DebugMode                  bool
	EnableEd25519              bool
	Ed25519PublicKeyPath       string
	Ed25519PrivateKeyPath      string
	EnableAESGCM               bool
	EnableChaCha20             bool
	EncryptionKeyPath          string
	EnableKeyRotation          bool
	KeyRotationInterval        time.Duration
	EnablePerfectForwardSecrecy bool
	EnableCertificatePinning   bool
	AllowedCertFingerprints    []string
	EnableMutualTLS            bool
}

type EnhancedSignatureResult struct {
	Valid               bool
	Reason              string
	Timestamp           int64
	Nonce               string
	Signature           string
	Sequence            int64
	ElapsedTime         time.Duration
	ErrorCode           string
	ClientIP            string
	RequestPath         string
	ReplayDetected      bool
	IntegrityValid      bool
	SignatureAlgorithm  string
	EncryptionEnabled   bool
}

type nonceRecord struct {
	timestamp   time.Time
	hashedNonce string
	count       int
}

type enhancedNonceCache struct {
	records map[string]*nonceRecord
	mu      sync.RWMutex
	limit   int
}

type enhancedSignatureState struct {
	sequenceCounters       map[string]int64
	ipRequestCounts        map[string]*ipRequestCounter
	ed25519PublicKeys      map[string]ed25519.PublicKey
	activeEncryptionKeys   map[int][]byte
	currentKeyVersion      int
	mu                     sync.RWMutex
}

type ipRequestCounter struct {
	count     int
	resetTime time.Time
}

type signatureValidator struct {
	config             EnhancedSignatureConfig
	nonceCache         *enhancedNonceCache
	state              *enhancedSignatureState
	ed25519PublicKey   ed25519.PublicKey
	ed25519PrivateKey  ed25519.PrivateKey
	encryptionKey      []byte
}

var defaultEnhancedSignatureConfig = EnhancedSignatureConfig{
	SecretKey:                  "enhanced-secret-key-change-in-production",
	Algorithm:                  "SHA256",
	TimestampTolerance:         5 * time.Minute,
	RequireTimestamp:           true,
	RequireNonce:               true,
	NonceCacheTTL:              24 * time.Hour,
	SignatureHeader:            "X-Signature",
	TimestampHeader:            "X-Timestamp",
	NonceHeader:                "X-Nonce",
	ExcludePaths:               []string{"/health", "/api/health", "/metrics", "/api/metrics", "/swagger/*", "/docs/*"},
	EnableHMAC_SHA512:          false,
	EnableDoubleSignature:      false,
	EnableSequenceCheck:        false,
	MaxSequenceGap:             10,
	EnableReplayCache:          true,
	ReplayCacheTTL:             24 * time.Hour,
	MinNonceLength:             8,
	MaxNonceLength:             64,
	EnableRateLimitPerIP:       false,
	RateLimitPerIPLimit:        100,
	RateLimitPerIPWindow:       time.Minute,
	EnableIntegrityCheck:       true,
	BodyIntegrityHeader:        "X-Body-Integrity",
	AdditionalHeaders:          []string{"X-Request-ID", "X-Forwarded-For"},
	SignatureVersion:           "3.0",
	DebugMode:                  false,
	EnableEd25519:              false,
	EnableAESGCM:               false,
	EnableChaCha20:             false,
	EnableKeyRotation:          false,
	KeyRotationInterval:        24 * time.Hour,
	EnablePerfectForwardSecrecy: false,
	EnableCertificatePinning:   false,
	EnableMutualTLS:            false,
}

var globalEnhancedNonceCache = &enhancedNonceCache{
	records: make(map[string]*nonceRecord),
	limit:   100000,
}

var globalEnhancedSignatureState = &enhancedSignatureState{
	sequenceCounters:     make(map[string]int64),
	ipRequestCounts:      make(map[string]*ipRequestCounter),
	ed25519PublicKeys:   make(map[string]ed25519.PublicKey),
	activeEncryptionKeys: make(map[int][]byte),
	currentKeyVersion:    1,
}

func init() {
	go globalEnhancedNonceCache.cleanupLoop()
	go globalEnhancedSignatureState.cleanupLoop()
	if defaultEnhancedSignatureConfig.EnableKeyRotation {
		go globalEnhancedSignatureState.keyRotationLoop()
	}
}

func (n *enhancedNonceCache) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		n.cleanup()
	}
}

func (n *enhancedNonceCache) cleanup() {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now()
	for nonce, record := range n.records {
		if now.Sub(record.timestamp) > 24*time.Hour {
			delete(n.records, nonce)
		}
	}
	if len(n.records) > n.limit {
		n.shrinkToLimit()
	}
}

func (n *enhancedNonceCache) shrinkToLimit() {
	count := 0
	limit := n.limit / 2
	for nonce := range n.records {
		if count >= limit {
			delete(n.records, nonce)
		}
		count++
	}
}

func (n *enhancedNonceCache) isUsed(nonce string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	hashedNonce := hashNonce(nonce)
	_, exists := n.records[hashedNonce]
	return exists
}

func (n *enhancedNonceCache) markUsed(nonce string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	hashedNonce := hashNonce(nonce)
	n.records[hashedNonce] = &nonceRecord{
		timestamp:   time.Now(),
		hashedNonce: hashedNonce,
		count:       1,
	}
}

func (n *enhancedNonceCache) incrementCount(nonce string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	hashedNonce := hashNonce(nonce)
	if record, exists := n.records[hashedNonce]; exists {
		record.count++
	}
}

func hashNonce(nonce string) string {
	h := sha256.New()
	h.Write([]byte(nonce))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *enhancedSignatureState) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *enhancedSignatureState) keyRotationLoop() {
	ticker := time.NewTicker(defaultEnhancedSignatureConfig.KeyRotationInterval)
	defer ticker.Stop()
	for range ticker.C {
		s.rotateEncryptionKey()
	}
}

func (s *enhancedSignatureState) rotateEncryptionKey() {
	s.mu.Lock()
	defer s.mu.Unlock()

	oldKey := make([]byte, 32)
	if len(s.activeEncryptionKeys[s.currentKeyVersion]) == 32 {
		copy(oldKey, s.activeEncryptionKeys[s.currentKeyVersion])
	}

	newKey := make([]byte, 32)
	io.ReadFull(rand.Reader, newKey)

	s.activeEncryptionKeys[s.currentKeyVersion] = oldKey
	s.currentKeyVersion++
	s.activeEncryptionKeys[s.currentKeyVersion] = newKey

	if len(s.activeEncryptionKeys) > 5 {
		oldestVersion := s.currentKeyVersion - 5
		delete(s.activeEncryptionKeys, oldestVersion)
	}
}

func (s *enhancedSignatureState) cleanup() {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()

	for ip, counter := range s.ipRequestCounts {
		if now.After(counter.resetTime) {
			delete(s.ipRequestCounts, ip)
		}
	}

	for key := range s.sequenceCounters {
		if strings.HasPrefix(key, "cleanup_") {
			delete(s.sequenceCounters, key)
		}
	}
}

func (s *enhancedSignatureState) getNextSequence(clientID string) int64 {
	s.mu.Lock()
	defer s.mu.Unlock()
	seq := s.sequenceCounters[clientID]
	s.sequenceCounters[clientID] = seq + 1
	return seq
}

func (s *enhancedSignatureState) validateSequence(clientID string, seq int64) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	expectedSeq := s.sequenceCounters[clientID]
	return seq == expectedSeq
}

func (s *enhancedSignatureState) incrementIPRequest(ip string, window time.Duration) bool {
	s.mu.Lock()
	defer s.mu.Unlock()
	now := time.Now()

	counter, exists := s.ipRequestCounts[ip]
	if !exists || now.After(counter.resetTime) {
		s.ipRequestCounts[ip] = &ipRequestCounter{
			count:     1,
			resetTime: now.Add(window),
		}
		return true
	}

	counter.count++
	return counter.count <= 100
}

func (s *enhancedSignatureState) getIPRequestCount(ip string) int {
	s.mu.RLock()
	defer s.mu.RUnlock()
	if counter, exists := s.ipRequestCounts[ip]; exists {
		return counter.count
	}
	return 0
}

func calculateEnhancedSignature(secretKey, method, path, query string, timestamp int64, nonce, bodyHash string, additionalData ...string) string {
	stringToSign := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData...)
	return computeEnhancedHMAC(secretKey, stringToSign, false)
}

func calculateDoubleSignature(secretKey string, params ...string) string {
	stringToSign := strings.Join(params, "|")
	return computeEnhancedHMAC(secretKey, stringToSign, true)
}

func buildEnhancedStringToSign(method, path, query string, timestamp int64, nonce, bodyHash string, additionalData ...string) string {
	var parts []string
	parts = append(parts, strings.ToUpper(method))
	parts = append(parts, path)

	if query != "" {
		sortedQuery := sortQueryStringEnhanced(query)
		parts = append(parts, sortedQuery)
	}

	parts = append(parts, strconv.FormatInt(timestamp, 10))

	if nonce != "" {
		parts = append(parts, nonce)
	}

	if bodyHash != "" {
		parts = append(parts, bodyHash)
	}

	for _, data := range additionalData {
		if data != "" {
			parts = append(parts, data)
		}
	}

	return strings.Join(parts, "\n")
}

func sortQueryStringEnhanced(query string) string {
	if query == "" {
		return ""
	}

	values, err := url.ParseQuery(query)
	if err != nil {
		return query
	}

	var keys []string
	for key := range values {
		keys = append(keys, key)
	}
	sort.Strings(keys)

	var resultParts []string
	for _, key := range keys {
		valuesList := values[key]
		for _, value := range valuesList {
			resultParts = append(resultParts, key+"="+value)
		}
	}

	return strings.Join(resultParts, "&")
}

func computeEnhancedHMAC(key, data string, useSHA512 bool) string {
	var h func() hash.Hash
	if useSHA512 {
		h = sha512.New
	} else {
		h = sha256.New
	}

	mac := hmac.New(h, []byte(key))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func hashBodyEnhanced(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	h := sha256.New()
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func computeBodyIntegrity(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	h := sha512.New384()
	h.Write(body)
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func verifyBodyIntegrity(body []byte, expectedIntegrity string) bool {
	if expectedIntegrity == "" {
		return true
	}
	actualIntegrity := computeBodyIntegrity(body)
	return subtle.ConstantTimeCompare([]byte(actualIntegrity), []byte(expectedIntegrity)) == 1
}

func verifyEnhancedTimestamp(timestamp int64, tolerance time.Duration) error {
	now := time.Now().Unix()
	diff := math.Abs(float64(now - timestamp))

	if diff > tolerance.Seconds() {
		return fmt.Errorf("timestamp out of tolerance: diff=%.2f seconds", diff)
	}

	if diff > tolerance.Seconds()*0.8 {
		return fmt.Errorf("timestamp approaching tolerance limit: diff=%.2f seconds", diff)
	}

	return nil
}

func verifyEnhancedNonce(nonce string, config EnhancedSignatureConfig) error {
	if nonce == "" {
		return fmt.Errorf("nonce is empty")
	}

	if len(nonce) < config.MinNonceLength || len(nonce) > config.MaxNonceLength {
		return fmt.Errorf("nonce length invalid: must be between %d and %d characters", config.MinNonceLength, config.MaxNonceLength)
	}

	if !isValidNonceFormat(nonce) {
		return fmt.Errorf("nonce format invalid: must be alphanumeric with optional dashes and underscores")
	}

	if globalEnhancedNonceCache.isUsed(nonce) {
		return fmt.Errorf("nonce already used: potential replay attack")
	}

	if config.EnableReplayCache && redis.Client != nil {
		ctx := context.Background()
		key := fmt.Sprintf("enhanced_signature:nonce:%s", hashNonce(nonce))
		exists, err := redis.Client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return fmt.Errorf("nonce already used in cache: potential replay attack")
		}
		err = redis.Client.Set(ctx, key, "1", config.NonceCacheTTL).Err()
		if err != nil {
			fmt.Printf("[EnhancedSignature] Warning: failed to store nonce in redis: %v\n", err)
		}
	}

	globalEnhancedNonceCache.markUsed(nonce)

	return nil
}

func isValidNonceFormat(nonce string) bool {
	if len(nonce) == 0 {
		return false
	}
	for _, c := range nonce {
		if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
			return false
		}
	}
	return true
}

func EnhancedSignatureVerification(config ...EnhancedSignatureConfig) gin.HandlerFunc {
	cfg := defaultEnhancedSignatureConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	validator := &signatureValidator{
		config:     cfg,
		nonceCache: globalEnhancedNonceCache,
		state:      globalEnhancedSignatureState,
	}

	if cfg.EnableEd25519 {
		validator.loadEd25519Keys()
	}

	if cfg.EnableAESGCM || cfg.EnableChaCha20 {
		validator.loadEncryptionKey()
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		isExcluded := false
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				isExcluded = true
				break
			}
		}
		if isExcluded {
			c.Next()
			return
		}

		startTime := time.Now()
		clientIP := c.ClientIP()
		result := EnhancedSignatureResult{
			ClientIP:    clientIP,
			RequestPath: path,
		}

		if cfg.EnableRateLimitPerIP {
			if !validator.state.incrementIPRequest(clientIP, cfg.RateLimitPerIPWindow) {
				result.Valid = false
				result.Reason = "rate limit exceeded"
				result.ErrorCode = "RATE_LIMIT_EXCEEDED"
				c.AbortWithStatusJSON(429, gin.H{
					"error":       "rate_limit_exceeded",
					"message":     "Too many requests from this IP",
					"retry_after": cfg.RateLimitPerIPWindow.Seconds(),
				})
				return
			}
		}

		signature := c.GetHeader(cfg.SignatureHeader)
		if signature == "" {
			result.Valid = false
			result.Reason = "missing signature"
			result.ErrorCode = "MISSING_SIGNATURE"
			c.AbortWithStatusJSON(401, gin.H{
				"error":   "missing_signature",
				"message": "X-Signature header is required",
			})
			return
		}
		result.Signature = signature

		var timestamp int64
		if cfg.RequireTimestamp {
			timestampStr := c.GetHeader(cfg.TimestampHeader)
			if timestampStr == "" {
				result.Valid = false
				result.Reason = "missing timestamp"
				result.ErrorCode = "MISSING_TIMESTAMP"
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "missing_timestamp",
					"message": "X-Timestamp header is required",
				})
				return
			}

			var err error
			timestamp, err = strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				result.Valid = false
				result.Reason = "invalid timestamp format"
				result.ErrorCode = "INVALID_TIMESTAMP"
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "invalid_timestamp",
					"message": "X-Timestamp must be a valid Unix timestamp",
				})
				return
			}
			result.Timestamp = timestamp

			if err := verifyEnhancedTimestamp(timestamp, cfg.TimestampTolerance); err != nil {
				result.Valid = false
				result.Reason = err.Error()
				result.ErrorCode = "TIMESTAMP_EXPIRED"
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "timestamp_expired",
					"message": err.Error(),
				})
				return
			}
		}

		nonce := c.GetHeader(cfg.NonceHeader)
		if cfg.RequireNonce {
			if nonce == "" {
				result.Valid = false
				result.Reason = "missing nonce"
				result.ErrorCode = "MISSING_NONCE"
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "missing_nonce",
					"message": "X-Nonce header is required",
				})
				return
			}
			result.Nonce = nonce

			if err := verifyEnhancedNonce(nonce, cfg); err != nil {
				result.Valid = false
				result.Reason = err.Error()
				result.ErrorCode = "NONCE_INVALID"
				result.ReplayDetected = true
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "nonce_invalid",
					"message": err.Error(),
				})
				return
			}
		}

		method := c.Request.Method
		query := c.Request.URL.RawQuery

		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		if cfg.EnableAESGCM || cfg.EnableChaCha20 {
			if encryptedBody := c.GetHeader("X-Encrypted-Body"); encryptedBody != "" {
				var err error
				body, err = validator.decryptRequestBody(encryptedBody)
				if err != nil {
					result.Valid = false
					result.Reason = "failed to decrypt body: " + err.Error()
					result.ErrorCode = "DECRYPTION_FAILED"
					c.AbortWithStatusJSON(401, gin.H{
						"error":   "decryption_failed",
						"message": "Failed to decrypt request body",
					})
					return
				}
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
				c.Set("decrypted_body", body)
			}
		}

		bodyHash := hashBodyEnhanced(body)

		if cfg.EnableIntegrityCheck {
			bodyIntegrity := c.GetHeader(cfg.BodyIntegrityHeader)
			if bodyIntegrity != "" {
				if !verifyBodyIntegrity(body, bodyIntegrity) {
					result.Valid = false
					result.Reason = "body integrity check failed"
					result.ErrorCode = "INTEGRITY_CHECK_FAILED"
					c.AbortWithStatusJSON(401, gin.H{
						"error":   "integrity_check_failed",
						"message": "Body integrity verification failed",
					})
					return
				}
				result.IntegrityValid = true
			}
		}

		var additionalData []string
		for _, header := range cfg.AdditionalHeaders {
			if value := c.GetHeader(header); value != "" {
				additionalData = append(additionalData, value)
			}
		}

		if cfg.EnableEd25519 {
			if !validator.verifyEd25519Signature(signature, method, path, query, timestamp, nonce, bodyHash, additionalData...) {
				result.Valid = false
				result.Reason = "Ed25519 signature verification failed"
				result.ErrorCode = "ED25519_SIGNATURE_INVALID"
				result.SignatureAlgorithm = "Ed25519"
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "invalid_signature",
					"message": "Ed25519 signature verification failed",
				})
				return
			}
			result.SignatureAlgorithm = "Ed25519"
		} else {
			expectedSignature := calculateEnhancedSignature(
				cfg.SecretKey,
				method,
				path,
				query,
				timestamp,
				nonce,
				bodyHash,
				additionalData...,
			)

			if !secureCompareEnhanced(signature, expectedSignature) {
				result.Valid = false
				result.Reason = "signature mismatch"
				result.ErrorCode = "SIGNATURE_MISMATCH"
				result.SignatureAlgorithm = "HMAC-" + cfg.Algorithm

				if cfg.DebugMode {
					logEnhancedSignatureFailure(c, &result, signature, expectedSignature, startTime)
				}

				c.AbortWithStatusJSON(401, gin.H{
					"error":   "invalid_signature",
					"message": "Signature verification failed",
				})
				return
			}
			result.SignatureAlgorithm = "HMAC-" + cfg.Algorithm
		}

		if cfg.EnableDoubleSignature {
			secondarySig := c.GetHeader("X-Signature-Secondary")
			if secondarySig != "" {
				secondaryExpected := calculateDoubleSignature(
					cfg.SecretKey,
					method,
					path,
					strconv.FormatInt(timestamp, 10),
					nonce,
				)
				if !secureCompareEnhanced(secondarySig, secondaryExpected) {
					result.Valid = false
					result.Reason = "secondary signature mismatch"
					result.ErrorCode = "SECONDARY_SIGNATURE_MISMATCH"
					c.AbortWithStatusJSON(401, gin.H{
						"error":   "invalid_signature",
						"message": "Secondary signature verification failed",
					})
					return
				}
			}
		}

		result.Valid = true
		result.Reason = "signature valid"
		result.ElapsedTime = time.Since(startTime)

		c.Set("enhanced_signature_verified", true)
		c.Set("enhanced_signature_timestamp", timestamp)
		c.Set("enhanced_signature_nonce", nonce)
		c.Set("enhanced_signature_result", &result)

		c.Next()
	}
}

func (v *signatureValidator) verifyEd25519Signature(signature string, method, path, query string, timestamp int64, nonce, bodyHash string, additionalData ...string) bool {
	if len(v.ed25519PublicKey) == 0 {
		return false
	}

	signatureBytes, err := base64.StdEncoding.DecodeString(signature)
	if err != nil {
		return false
	}

	message := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData...)
	return ed25519.Verify(v.ed25519PublicKey, []byte(message), signatureBytes)
}

func (v *signatureValidator) loadEd25519Keys() {
	if v.config.Ed25519PublicKeyPath != "" {
		data, err := os.ReadFile(v.config.Ed25519PublicKeyPath)
		if err == nil {
			v.ed25519PublicKey = ed25519.PublicKey(data)
		}
	}

	if v.config.Ed25519PrivateKeyPath != "" {
		data, err := os.ReadFile(v.config.Ed25519PrivateKeyPath)
		if err == nil {
			v.ed25519PrivateKey = ed25519.PrivateKey(data)
		}
	}
}

func (v *signatureValidator) loadEncryptionKey() {
	if v.config.EncryptionKeyPath != "" {
		data, err := os.ReadFile(v.config.EncryptionKeyPath)
		if err == nil && len(data) == 32 {
			v.encryptionKey = data
		}
	}

	if len(v.encryptionKey) == 0 {
		v.encryptionKey = make([]byte, 32)
		io.ReadFull(rand.Reader, v.encryptionKey)
	}
}

func (v *signatureValidator) encryptRequestBody(body []byte) (string, error) {
	if v.config.EnableChaCha20 {
		return v.encryptChaCha20(body)
	}
	return v.encryptAESGCM(body)
}

func (v *signatureValidator) decryptRequestBody(encryptedBody string) ([]byte, error) {
	if v.config.EnableChaCha20 {
		return v.decryptChaCha20(encryptedBody)
	}
	return v.decryptAESGCM(encryptedBody)
}

func (v *signatureValidator) encryptAESGCM(body []byte) (string, error) {
	if len(v.encryptionKey) == 0 {
		return "", fmt.Errorf("encryption key not set")
	}

	block, err := aes.NewCipher(v.encryptionKey)
	if err != nil {
		return "", err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := gcm.Seal(nil, nonce, body, nil)

	encrypted := struct {
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
		Version    int    `json:"v"`
	}{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Version:    1,
	}

	data, err := json.Marshal(encrypted)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func (v *signatureValidator) decryptAESGCM(encryptedBody string) ([]byte, error) {
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedBody)
	if err != nil {
		return nil, err
	}

	var encrypted struct {
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
		Version    int    `json:"v"`
	}

	if err := json.Unmarshal(encryptedData, &encrypted); err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, err
	}

	block, err := aes.NewCipher(v.encryptionKey)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (v *signatureValidator) encryptChaCha20(body []byte) (string, error) {
	if len(v.encryptionKey) == 0 {
		return "", fmt.Errorf("encryption key not set")
	}

	aead, err := chacha20poly1305.New(v.encryptionKey)
	if err != nil {
		return "", err
	}

	nonce := make([]byte, chacha20poly1305.NonceSizeX)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return "", err
	}

	ciphertext := aead.Seal(nil, nonce, body, nil)

	encrypted := struct {
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
		Version    int    `json:"v"`
	}{
		Nonce:      base64.StdEncoding.EncodeToString(nonce),
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		Version:    2,
	}

	data, err := json.Marshal(encrypted)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(data), nil
}

func (v *signatureValidator) decryptChaCha20(encryptedBody string) ([]byte, error) {
	encryptedData, err := base64.StdEncoding.DecodeString(encryptedBody)
	if err != nil {
		return nil, err
	}

	var encrypted struct {
		Nonce      string `json:"nonce"`
		Ciphertext string `json:"ciphertext"`
		Version    int    `json:"v"`
	}

	if err := json.Unmarshal(encryptedData, &encrypted); err != nil {
		return nil, err
	}

	nonce, err := base64.StdEncoding.DecodeString(encrypted.Nonce)
	if err != nil {
		return nil, err
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Ciphertext)
	if err != nil {
		return nil, err
	}

	aead, err := chacha20poly1305.New(v.encryptionKey)
	if err != nil {
		return nil, err
	}

	return aead.Open(nil, nonce, ciphertext, nil)
}

type nopCloserReader struct {
	r io.Reader
}

func (n *nopCloserReader) Read(p []byte) (int, error) {
	return n.r.Read(p)
}

func secureCompareEnhanced(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func logEnhancedSignatureFailure(c *gin.Context, result *EnhancedSignatureResult, provided, expected string, startTime time.Time) {
	method := c.Request.Method
	path := c.Request.URL.Path
	userAgent := c.GetHeader("User-Agent")
	elapsed := time.Since(startTime)

	sigPreview := provided
	if len(sigPreview) > 16 {
		sigPreview = sigPreview[:16] + "..."
	}

	expectedPreview := expected
	if len(expectedPreview) > 16 {
		expectedPreview = expectedPreview[:16] + "..."
	}

	fmt.Printf("[ENHANCED_SIGNATURE_FAILED] %s | %s %s | IP: %s | UA: %s | Timestamp: %d | Nonce: %s | Provided: %s | Expected: %s | Elapsed: %v\n",
		method,
		path,
		c.Request.URL.RawQuery,
		result.ClientIP,
		userAgent,
		result.Timestamp,
		result.Nonce,
		sigPreview,
		expectedPreview,
		elapsed,
	)
}

func GenerateEnhancedSignature(secretKey, method, path, query string, timestamp int64, nonce string, body []byte, additionalData ...string) string {
	bodyHash := hashBodyEnhanced(body)
	return calculateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, bodyHash, additionalData...)
}

func GenerateEd25519Signature(privateKey ed25519.PrivateKey, method, path, query string, timestamp int64, nonce string, body []byte, additionalData ...string) string {
	bodyHash := hashBodyEnhanced(body)
	message := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData...)
	signature := ed25519.Sign(privateKey, []byte(message))
	return base64.StdEncoding.EncodeToString(signature)
}

func GenerateEnhancedNonce(length int) (string, error) {
	if length < 8 {
		length = 16
	}
	if length > 64 {
		length = 64
	}

	bytes := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func GenerateSecureNonce(length int) (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	if length < 8 {
		length = 16
	}
	if length > 64 {
		length = 64
	}

	result := make([]byte, length)
	for i := range result {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(chars))))
		if err != nil {
			return "", fmt.Errorf("failed to generate nonce: %w", err)
		}
		result[i] = chars[idx.Int64()]
	}

	return string(result), nil
}

func ValidateEnhancedSignature(c *gin.Context, secretKey string) EnhancedSignatureResult {
	startTime := time.Now()
	cfg := defaultEnhancedSignatureConfig
	cfg.SecretKey = secretKey

	result := EnhancedSignatureResult{
		Valid:       false,
		ElapsedTime: 0,
	}

	signature := c.GetHeader(cfg.SignatureHeader)
	result.Signature = signature

	timestampStr := c.GetHeader(cfg.TimestampHeader)
	if timestampStr != "" {
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err == nil {
			result.Timestamp = timestamp
		}
	}

	nonce := c.GetHeader(cfg.NonceHeader)
	result.Nonce = nonce

	method := c.Request.Method
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery

	var body []byte
	if c.Request.Body != nil {
		body, _ = io.ReadAll(c.Request.Body)
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	}

	bodyHash := hashBodyEnhanced(body)

	expectedSignature := calculateEnhancedSignature(
		secretKey,
		method,
		path,
		query,
		result.Timestamp,
		nonce,
		bodyHash,
	)

	if secureCompareEnhanced(signature, expectedSignature) {
		result.Valid = true
		result.Reason = "signature valid"
	} else {
		result.Reason = "signature mismatch"
		result.ErrorCode = "SIGNATURE_MISMATCH"
	}

	result.ElapsedTime = time.Since(startTime)

	return result
}

func RequireEnhancedSignature() gin.HandlerFunc {
	return EnhancedSignatureVerification()
}

type EnhancedSignatureInfo struct {
	Algorithm     string `json:"algorithm"`
	Timestamp     int64  `json:"timestamp"`
	NonceRequired bool   `json:"nonce_required"`
	Tolerance     string `json:"tolerance"`
	Version       string `json:"version"`
	Features      struct {
		HMAC_SHA512        bool `json:"hmac_sha512"`
		DoubleSignature    bool `json:"double_signature"`
		SequenceCheck      bool `json:"sequence_check"`
		ReplayProtection   bool `json:"replay_protection"`
		IntegrityCheck     bool `json:"integrity_check"`
		Ed25519            bool `json:"ed25519"`
		AESGCM             bool `json:"aes_gcm"`
		ChaCha20           bool `json:"chacha20"`
		KeyRotation        bool `json:"key_rotation"`
		PerfectForwardSecrecy bool `json:"perfect_forward_secrecy"`
	} `json:"features"`
}

func GetEnhancedSignatureInfo() EnhancedSignatureInfo {
	cfg := defaultEnhancedSignatureConfig
	info := EnhancedSignatureInfo{
		Algorithm:     cfg.Algorithm,
		Timestamp:     time.Now().Unix(),
		NonceRequired: cfg.RequireNonce,
		Tolerance:     cfg.TimestampTolerance.String(),
		Version:       cfg.SignatureVersion,
	}
	info.Features.HMAC_SHA512 = cfg.EnableHMAC_SHA512
	info.Features.DoubleSignature = cfg.EnableDoubleSignature
	info.Features.SequenceCheck = cfg.EnableSequenceCheck
	info.Features.ReplayProtection = cfg.EnableReplayCache
	info.Features.IntegrityCheck = cfg.EnableIntegrityCheck
	info.Features.Ed25519 = cfg.EnableEd25519
	info.Features.AESGCM = cfg.EnableAESGCM
	info.Features.ChaCha20 = cfg.EnableChaCha20
	info.Features.KeyRotation = cfg.EnableKeyRotation
	info.Features.PerfectForwardSecrecy = cfg.EnablePerfectForwardSecrecy
	return info
}

func NewEnhancedSignatureConfig(secretKey string) EnhancedSignatureConfig {
	return EnhancedSignatureConfig{
		SecretKey:                  secretKey,
		Algorithm:                  "SHA256",
		TimestampTolerance:         5 * time.Minute,
		RequireTimestamp:           true,
		RequireNonce:               true,
		NonceCacheTTL:              24 * time.Hour,
		SignatureHeader:            "X-Signature",
		TimestampHeader:            "X-Timestamp",
		NonceHeader:                "X-Nonce",
		ExcludePaths:               []string{},
		EnableHMAC_SHA512:          false,
		EnableDoubleSignature:      false,
		EnableSequenceCheck:        false,
		MaxSequenceGap:             10,
		EnableReplayCache:          true,
		ReplayCacheTTL:             24 * time.Hour,
		MinNonceLength:             8,
		MaxNonceLength:             64,
		EnableRateLimitPerIP:       false,
		RateLimitPerIPLimit:        100,
		RateLimitPerIPWindow:       time.Minute,
		EnableIntegrityCheck:       true,
		BodyIntegrityHeader:        "X-Body-Integrity",
		AdditionalHeaders:          []string{"X-Request-ID", "X-Forwarded-For"},
		SignatureVersion:           "3.0",
		DebugMode:                  false,
		EnableEd25519:              false,
		EnableAESGCM:               false,
		EnableChaCha20:             false,
		EnableKeyRotation:          false,
		KeyRotationInterval:        24 * time.Hour,
		EnablePerfectForwardSecrecy: false,
		EnableCertificatePinning:   false,
		EnableMutualTLS:            false,
	}
}

const EnhancedSignatureVersion = "3.0"

func BuildEnhancedSignatureInput(secretKey, method, path, query string, timestamp int64, nonce string, body []byte) (string, error) {
	if nonce == "" {
		var err error
		nonce, err = GenerateSecureNonce(16)
		if err != nil {
			return "", err
		}
	}
	bodyHash := hashBodyEnhanced(body)
	return calculateEnhancedSignature(secretKey, method, path, query, timestamp, nonce, bodyHash), nil
}

func GenerateTimestampWithMillis() int64 {
	return time.Now().UnixMilli()
}

func VerifyTimestampMillis(timestamp int64, tolerance time.Duration) error {
	now := time.Now().UnixMilli()
	diff := math.Abs(float64(now - timestamp))
	toleranceMillis := float64(tolerance.Milliseconds())

	if diff > toleranceMillis {
		return fmt.Errorf("timestamp out of tolerance: diff=%.2f ms", diff)
	}

	return nil
}

func GenerateRequestID() string {
	bytes := make([]byte, 16)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return fmt.Sprintf("req_%d_%d", time.Now().UnixNano(), time.Now().UnixMilli())
	}
	return fmt.Sprintf("req_%s", hex.EncodeToString(bytes))
}

func ExtractRequestID(c *gin.Context) string {
	requestID := c.GetHeader("X-Request-ID")
	if requestID == "" {
		requestID = GenerateRequestID()
		c.Header("X-Request-ID", requestID)
	}
	return requestID
}

func CreateSignatureMiddlewareChain() []gin.HandlerFunc {
	return []gin.HandlerFunc{
		func(c *gin.Context) {
			c.Set("request_id", ExtractRequestID(c))
			c.Next()
		},
		EnhancedSignatureVerification(),
	}
}

func GenerateEd25519KeyPair() (ed25519.PrivateKey, ed25519.PublicKey, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return privateKey, publicKey, nil
}

func SaveEd25519KeyPair(privateKey ed25519.PrivateKey, publicKey ed25519.PublicKey, privatePath, publicPath string) error {
	if err := os.WriteFile(privatePath, privateKey, 0600); err != nil {
		return err
	}
	return os.WriteFile(publicPath, publicKey, 0644)
}

func LoadEd25519KeyPair(privatePath, publicPath string) (ed25519.PrivateKey, ed25519.PublicKey, error) {
	privateData, err := os.ReadFile(privatePath)
	if err != nil {
		return nil, nil, err
	}

	publicData, err := os.ReadFile(publicPath)
	if err != nil {
		return nil, nil, err
	}

	return ed25519.PrivateKey(privateData), ed25519.PublicKey(publicData), nil
}

func GenerateEncryptionKey() ([]byte, error) {
	key := make([]byte, 32)
	_, err := io.ReadFull(rand.Reader, key)
	if err != nil {
		return nil, err
	}
	return key, nil
}

func GenerateEphemeralKeyPair() ([]byte, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, err
	}
	return []byte(privateKey), []byte(publicKey), nil
}