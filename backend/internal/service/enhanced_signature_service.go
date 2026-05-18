package service

import (
	"crypto"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RequestSignatureConfig struct {
	SecretKey               string
	Algorithm               string
	TimestampTolerance      time.Duration
	RequireTimestamp        bool
	RequireNonce            bool
	NonceCacheTTL           time.Duration
	EnableBodyHash          bool
	EnableQuerySorting      bool
	EnableDoubleSignature   bool
	EnableSequenceNumber    bool
	SignatureHeader         string
	TimestampHeader         string
	NonceHeader             string
	SequenceHeader          string
	ExcludePaths            []string
	MinNonceLength          int
	MaxNonceLength          int
	EnableRateLimit         bool
	RateLimitPerIPLimit     int
	RateLimitWindow         time.Duration
	EnableReplayCache       bool
	ReplayCacheTTL          time.Duration
	MaxClockSkew           time.Duration
	DebugMode               bool
}

type SignatureVerificationResult struct {
	Valid               bool
	Reason              string
	Timestamp           int64
	Nonce               string
	Signature           string
	Sequence            int64
	ClientIP            string
	RequestPath         string
	ElapsedTime         time.Duration
	ErrorCode           string
	ReplayDetected      bool
	SignatureValid      bool
	TimestampValid      bool
	NonceValid          bool
	SequenceValid       bool
	IntegrityValid      bool
}

type NonceCacheEntry struct {
	Used       bool
	UsedAt     time.Time
	Signature  string
}

type SignatureValidator struct {
	config      RequestSignatureConfig
	nonceCache  map[string]*NonceCacheEntry
	nonceMutex  sync.RWMutex
	seqNumbers  map[string]int64
	seqMutex    sync.RWMutex
	rateLimits  map[string]*RateLimitEntry
	rateMutex   sync.RWMutex
}

type RateLimitEntry struct {
	Count    int
	ResetAt  time.Time
}

var defaultSignatureConfig = RequestSignatureConfig{
	Algorithm:              "SHA256",
	TimestampTolerance:     5 * time.Minute,
	RequireTimestamp:       true,
	RequireNonce:           true,
	NonceCacheTTL:         24 * time.Hour,
	EnableBodyHash:        true,
	EnableQuerySorting:    true,
	EnableDoubleSignature: false,
	EnableSequenceNumber:  false,
	SignatureHeader:       "X-Signature",
	TimestampHeader:       "X-Timestamp",
	NonceHeader:           "X-Nonce",
	SequenceHeader:        "X-Sequence",
	ExcludePaths:          []string{"/health", "/metrics", "/api/health", "/docs", "/swagger"},
	MinNonceLength:        16,
	MaxNonceLength:        64,
	EnableRateLimit:       true,
	RateLimitPerIPLimit:   100,
	RateLimitWindow:       1 * time.Minute,
	EnableReplayCache:     true,
	ReplayCacheTTL:        24 * time.Hour,
	MaxClockSkew:         30 * time.Second,
	DebugMode:             false,
}

func NewSignatureValidator(configs ...RequestSignatureConfig) *SignatureValidator {
	cfg := defaultSignatureConfig
	if len(configs) > 0 {
		cfg = configs[0]
	}

	return &SignatureValidator{
		config:     cfg,
		nonceCache: make(map[string]*NonceCacheEntry),
		seqNumbers: make(map[string]int64),
		rateLimits: make(map[string]*RateLimitEntry),
	}
}

func (v *SignatureValidator) ValidateRequest(method, path, query string, headers map[string]string, body []byte, clientIP string) *SignatureVerificationResult {
	startTime := time.Now()
	result := &SignatureVerificationResult{
		Valid:    false,
		ClientIP: clientIP,
		RequestPath: path,
	}

	for _, excluded := range v.config.ExcludePaths {
		if path == excluded || strings.HasPrefix(path, excluded+"/") {
			result.Valid = true
			result.Reason = "excluded_path"
			result.ElapsedTime = time.Since(startTime)
			return result
		}
	}

	if v.config.EnableRateLimit {
		if !v.checkRateLimit(clientIP) {
			result.Reason = "rate_limit_exceeded"
			result.ErrorCode = "RATE_LIMIT_EXCEEDED"
			result.ElapsedTime = time.Since(startTime)
			return result
		}
	}

	signature := headers[v.config.SignatureHeader]
	if signature == "" {
		result.Reason = "missing_signature"
		result.ErrorCode = "MISSING_SIGNATURE"
		return result
	}
	result.Signature = signature

	var timestamp int64
	if v.config.RequireTimestamp {
		timestampStr := headers[v.config.TimestampHeader]
		if timestampStr == "" {
			result.Reason = "missing_timestamp"
			result.ErrorCode = "MISSING_TIMESTAMP"
			return result
		}

		var err error
		timestamp, err = strconv.ParseInt(timestampStr, 10, 64)
		if err != nil {
			result.Reason = "invalid_timestamp_format"
			result.ErrorCode = "INVALID_TIMESTAMP"
			return result
		}
		result.Timestamp = timestamp

		if !v.validateTimestamp(timestamp) {
			result.Reason = "timestamp_out_of_tolerance"
			result.ErrorCode = "TIMESTAMP_EXPIRED"
			result.TimestampValid = false
			return result
		}
		result.TimestampValid = true
	}

	var nonce string
	if v.config.RequireNonce {
		nonce = headers[v.config.NonceHeader]
		if nonce == "" {
			result.Reason = "missing_nonce"
			result.ErrorCode = "MISSING_NONCE"
			return result
		}

		if !v.validateNonce(nonce) {
			result.Reason = "invalid_or_reused_nonce"
			result.ErrorCode = "NONCE_INVALID"
			result.ReplayDetected = true
			return result
		}
		result.Nonce = nonce
		result.NonceValid = true
	}

	var sequence int64
	if v.config.EnableSequenceNumber {
		seqStr := headers[v.config.SequenceHeader]
		if seqStr != "" {
			var err error
			sequence, err = strconv.ParseInt(seqStr, 10, 64)
			if err == nil {
				result.Sequence = sequence
				if !v.validateSequence(clientIP, sequence) {
					result.Reason = "invalid_sequence_number"
					result.ErrorCode = "SEQUENCE_INVALID"
					result.SequenceValid = false
					return result
				}
				result.SequenceValid = true
			}
		}
	}

	expectedSignature := v.calculateSignature(method, path, query, timestamp, nonce, body)

	if !v.secureCompare(signature, expectedSignature) {
		result.Reason = "signature_mismatch"
		result.ErrorCode = "SIGNATURE_MISMATCH"
		result.SignatureValid = false

		if v.config.DebugMode {
			fmt.Printf("[SIGNATURE_DEBUG] Client: %s | Path: %s | Expected: %s... | Got: %s...\n",
				clientIP, path, expectedSignature[:16], signature[:16])
		}

		return result
	}
	result.SignatureValid = true

	result.Valid = true
	result.Reason = "signature_valid"
	result.ElapsedTime = time.Since(startTime)

	return result
}

func (v *SignatureValidator) calculateSignature(method, path, query string, timestamp int64, nonce string, body []byte) string {
	var parts []string

	parts = append(parts, strings.ToUpper(method))
	parts = append(parts, path)

	if query != "" && v.config.EnableQuerySorting {
		parts = append(parts, v.sortQueryString(query))
	} else if query != "" {
		parts = append(parts, query)
	}

	if timestamp > 0 {
		parts = append(parts, strconv.FormatInt(timestamp, 10))
	}

	if nonce != "" {
		parts = append(parts, nonce)
	}

	if v.config.EnableBodyHash && len(body) > 0 {
		bodyHash := v.hashBody(body)
		parts = append(parts, bodyHash)
	}

	stringToSign := strings.Join(parts, "\n")

	return v.computeHMAC(stringToSign)
}

func (v *SignatureValidator) sortQueryString(query string) string {
	if query == "" {
		return ""
	}

	values, err := url.ParseQuery(query)
	if err != nil {
		return query
	}

	var keys []string
	for k := range values {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var resultParts []string
	for _, k := range keys {
		valuesList := values[k]
		sort.Strings(valuesList)
		for _, val := range valuesList {
			resultParts = append(resultParts, fmt.Sprintf("%s=%s", k, val))
		}
	}

	return strings.Join(resultParts, "&")
}

func (v *SignatureValidator) hashBody(body []byte) string {
	var h hash.Hash
	switch v.config.Algorithm {
	case "SHA256":
		h = sha256.New()
	case "SHA512":
		h = sha512.New()
	case "SHA1":
		h = sha1.New()
	default:
		h = sha256.New()
	}
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func (v *SignatureValidator) computeHMAC(data string) string {
	var h func() hash.Hash
	switch v.config.Algorithm {
	case "SHA256":
		h = sha256.New
	case "SHA512":
		h = sha512.New
	case "SHA1":
		h = sha1.New
	default:
		h = sha256.New
	}

	mac := hmac.New(h, []byte(v.config.SecretKey))
	mac.Write([]byte(data))
	return hex.EncodeToString(mac.Sum(nil))
}

func (v *SignatureValidator) validateTimestamp(timestamp int64) bool {
	now := time.Now().Unix()
	diff := now - timestamp

	if diff < 0 {
		diff = -diff
	}

	if float64(diff) > v.config.TimestampTolerance.Seconds() {
		return false
	}

	if float64(diff) > v.config.MaxClockSkew.Seconds() {
		return false
	}

	return true
}

func (v *SignatureValidator) validateNonce(nonce string) bool {
	if len(nonce) < v.config.MinNonceLength || len(nonce) > v.config.MaxNonceLength {
		return false
	}

	v.nonceMutex.Lock()
	defer v.nonceMutex.Unlock()

	if entry, exists := v.nonceCache[nonce]; exists {
		if time.Since(entry.UsedAt) < v.config.NonceCacheTTL {
			return false
		}
	}

	v.nonceCache[nonce] = &NonceCacheEntry{
		Used:   true,
		UsedAt: time.Now(),
	}

	if len(v.nonceCache) > 100000 {
		v.cleanupNonceCache()
	}

	return true
}

func (v *SignatureValidator) validateSequence(clientID string, sequence int64) bool {
	v.seqMutex.Lock()
	defer v.seqMutex.Unlock()

	expectedSeq, exists := v.seqNumbers[clientID]
	if !exists {
		v.seqNumbers[clientID] = sequence
		return true
	}

	if sequence <= expectedSeq {
		return false
	}

	v.seqNumbers[clientID] = sequence
	return true
}

func (v *SignatureValidator) checkRateLimit(clientID string) bool {
	v.rateMutex.Lock()
	defer v.rateMutex.Unlock()

	now := time.Now()
	entry, exists := v.rateLimits[clientID]

	if !exists || now.After(entry.ResetAt) {
		v.rateLimits[clientID] = &RateLimitEntry{
			Count:   1,
			ResetAt: now.Add(v.config.RateLimitWindow),
		}
		return true
	}

	entry.Count++
	if entry.Count > v.config.RateLimitPerIPLimit {
		return false
	}

	return true
}

func (v *SignatureValidator) cleanupNonceCache() {
	now := time.Now()
	cutoff := now.Add(-v.config.NonceCacheTTL)

	for nonce, entry := range v.nonceCache {
		if entry.UsedAt.Before(cutoff) {
			delete(v.nonceCache, nonce)
		}
	}
}

func (v *SignatureValidator) secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (v *SignatureValidator) GenerateSignature(method, path, query string, body []byte) (string, int64, string, error) {
	timestamp := time.Now().Unix()

	nonce, err := GenerateSignatureNonce(16)
	if err != nil {
		return "", 0, "", err
	}

	signature := v.calculateSignature(method, path, query, timestamp, nonce, body)

	return signature, timestamp, nonce, nil
}

func GenerateSignatureNonce(length int) (string, error) {
	if length < 8 {
		length = 16
	}
	if length > 64 {
		length = 64
	}

	bytes := make([]byte, length)
	if _, err := rand.Read(bytes); err != nil {
		return "", err
	}

	return base64.URLEncoding.EncodeToString(bytes), nil
}

func (v *SignatureValidator) GetStats() map[string]interface{} {
	v.nonceMutex.RLock()
	nonceCount := len(v.nonceCache)
	v.nonceMutex.RUnlock()

	v.seqMutex.RLock()
	seqCount := len(v.seqNumbers)
	v.seqMutex.RUnlock()

	v.rateMutex.RLock()
	rateLimitCount := len(v.rateLimits)
	v.rateMutex.RUnlock()

	return map[string]interface{}{
		"nonce_cache_size":    nonceCount,
		"sequence_numbers":     seqCount,
		"rate_limits":          rateLimitCount,
		"config":               v.config,
	}
}

func (v *SignatureValidator) ClearCache() {
	v.nonceMutex.Lock()
	v.nonceCache = make(map[string]*NonceCacheEntry)
	v.nonceMutex.Unlock()

	v.seqMutex.Lock()
	v.seqNumbers = make(map[string]int64)
	v.seqMutex.Unlock()

	v.rateMutex.Lock()
	v.rateLimits = make(map[string]*RateLimitEntry)
	v.rateMutex.Unlock()
}

type RSASignatureValidator struct {
	config      RequestSignatureConfig
	privateKey  *rsa.PrivateKey
	publicKey   *rsa.PublicKey
	nonceCache  map[string]*NonceCacheEntry
	nonceMutex  sync.RWMutex
}

func NewRSASignatureValidator(privateKeyPEM, publicKeyPEM string) (*RSASignatureValidator, error) {
	var privateKey *rsa.PrivateKey
	var publicKey *rsa.PublicKey
	var err error

	if privateKeyPEM != "" {
		block, _ := pem.Decode([]byte(privateKeyPEM))
		if block != nil {
			privateKey, err = x509.ParsePKCS1PrivateKey(block.Bytes)
			if err != nil {
				return nil, fmt.Errorf("failed to parse private key: %w", err)
			}
		}
	}

	if publicKeyPEM != "" {
		block, _ := pem.Decode([]byte(publicKeyPEM))
		if block != nil {
			publicKeyInterface, err := x509.ParsePKIXPublicKey(block.Bytes)
			if err != nil {
				pubKey, err := x509.ParsePKCS1PublicKey(block.Bytes)
				if err != nil {
					return nil, fmt.Errorf("failed to parse public key: %w", err)
				}
				publicKey = pubKey
			} else {
				var ok bool
				publicKey, ok = publicKeyInterface.(*rsa.PublicKey)
				if !ok {
					return nil, errors.New("not an RSA public key")
				}
			}
		}
	}

	return &RSASignatureValidator{
		config:     defaultSignatureConfig,
		privateKey: privateKey,
		publicKey:  publicKey,
		nonceCache: make(map[string]*NonceCacheEntry),
	}, nil
}

func (v *RSASignatureValidator) Sign(message []byte) (string, error) {
	if v.privateKey == nil {
		return "", errors.New("private key not available")
	}

	var hashType crypto.Hash
	switch v.config.Algorithm {
	case "SHA256":
		hashType = crypto.SHA256
	case "SHA512":
		hashType = crypto.SHA512
	default:
		hashType = crypto.SHA256
	}

	hash := sha256.New()
	hash.Write([]byte(message))
	hashed := hash.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, v.privateKey, hashType, hashed)
	if err != nil {
		return "", err
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func (v *RSASignatureValidator) Verify(message, signatureBase64 string) error {
	if v.publicKey == nil {
		return errors.New("public key not available")
	}

	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return err
	}

	var hashType crypto.Hash
	switch v.config.Algorithm {
	case "SHA256":
		hashType = crypto.SHA256
	case "SHA512":
		hashType = crypto.SHA512
	default:
		hashType = crypto.SHA256
	}

	hash := sha256.New()
	hash.Write([]byte(message))
	hashed := hash.Sum(nil)

	return rsa.VerifyPKCS1v15(v.publicKey, hashType, hashed, signature)
}

func (v *RSASignatureValidator) ValidateNonce(nonce string) bool {
	v.nonceMutex.Lock()
	defer v.nonceMutex.Unlock()

	if _, exists := v.nonceCache[nonce]; exists {
		return false
	}

	v.nonceCache[nonce] = &NonceCacheEntry{
		Used:   true,
		UsedAt: time.Now(),
	}

	return true
}

type DoubleSignatureValidator struct {
	primaryValidator   *SignatureValidator
	secondaryValidator *SignatureValidator
	primaryAlgorithm   string
	secondaryAlgorithm string
}

func NewDoubleSignatureValidator(primaryKey, secondaryKey string, primaryAlg, secondaryAlg string) *DoubleSignatureValidator {
	return &DoubleSignatureValidator{
		primaryValidator: NewSignatureValidator(RequestSignatureConfig{
			SecretKey: primaryKey,
			Algorithm: primaryAlg,
		}),
		secondaryValidator: NewSignatureValidator(RequestSignatureConfig{
			SecretKey: secondaryKey,
			Algorithm: secondaryAlg,
		}),
		primaryAlgorithm:   primaryAlg,
		secondaryAlgorithm: secondaryAlg,
	}
}

func (v *DoubleSignatureValidator) ValidateRequest(method, path, query string, headers map[string]string, body []byte, clientIP string) *SignatureVerificationResult {
	primaryResult := v.primaryValidator.ValidateRequest(method, path, query, headers, body, clientIP)

	if !primaryResult.Valid {
		return primaryResult
	}

	secondaryHeaders := map[string]string{
		v.secondaryValidator.config.SignatureHeader: headers["X-Signature-Secondary"],
	}

	secondaryResult := v.secondaryValidator.ValidateRequest(method, path, query, secondaryHeaders, body, clientIP)

	if !secondaryResult.Valid {
		primaryResult.Valid = false
		primaryResult.Reason = "secondary_signature_invalid"
		primaryResult.ErrorCode = "SECONDARY_SIGNATURE_INVALID"
	}

	return primaryResult
}

func (v *DoubleSignatureValidator) GenerateDualSignature(method, path, query string, body []byte) (string, string, int64, error) {
	sig1, timestamp, _, err := v.primaryValidator.GenerateSignature(method, path, query, body)
	if err != nil {
		return "", "", 0, err
	}

	nonce2, _ := GenerateSignatureNonce(16)
	sig2 := v.secondaryValidator.calculateSignature(method, path, query, timestamp, nonce2, body)

	return sig1, sig2, timestamp, nil
}
