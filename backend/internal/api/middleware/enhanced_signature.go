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
	"encoding/binary"
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
	"github.com/hjtpx/hjtpx/pkg/crypto"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"golang.org/x/crypto/ed25519"
)

type EnhancedSignatureConfig struct {
	SecretKey             string
	Algorithm             string
	TimestampTolerance    time.Duration
	RequireTimestamp      bool
	RequireNonce          bool
	NonceCacheTTL         time.Duration
	SignatureHeader       string
	TimestampHeader       string
	NonceHeader           string
	ExcludePaths          []string
	EnableHMAC_SHA512     bool
	EnableDoubleSignature bool
	EnableSequenceCheck   bool
	MaxSequenceGap        int64
	EnableReplayCache     bool
	ReplayCacheTTL        time.Duration
	MinNonceLength        int
	MaxNonceLength        int
	EnableRateLimitPerIP  bool
	RateLimitPerIPLimit   int
	RateLimitPerIPWindow  time.Duration
	EnableIntegrityCheck  bool
	BodyIntegrityHeader   string
	AdditionalHeaders     []string
	SignatureVersion      string
	DebugMode             bool
}

type EnhancedSignatureResult struct {
	Valid          bool
	Reason         string
	Timestamp      int64
	Nonce          string
	Signature      string
	Sequence       int64
	ElapsedTime    time.Duration
	ErrorCode      string
	ClientIP       string
	RequestPath    string
	ReplayDetected bool
	IntegrityValid bool
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
	sequenceCounters map[string]int64
	ipRequestCounts  map[string]*ipRequestCounter
	mu               sync.RWMutex
}

type ipRequestCounter struct {
	count     int
	resetTime time.Time
}

type signatureValidator struct {
	config     EnhancedSignatureConfig
	nonceCache *enhancedNonceCache
	state      *enhancedSignatureState
}

var defaultEnhancedSignatureConfig = EnhancedSignatureConfig{
	SecretKey:             "enhanced-secret-key-change-in-production",
	Algorithm:             "SHA256",
	TimestampTolerance:    5 * time.Minute,
	RequireTimestamp:      true,
	RequireNonce:          true,
	NonceCacheTTL:         24 * time.Hour,
	SignatureHeader:       "X-Signature",
	TimestampHeader:       "X-Timestamp",
	NonceHeader:           "X-Nonce",
	ExcludePaths:          []string{"/health", "/api/health", "/metrics", "/api/metrics", "/swagger/*", "/docs/*"},
	EnableHMAC_SHA512:     false,
	EnableDoubleSignature: false,
	EnableSequenceCheck:   false,
	MaxSequenceGap:        10,
	EnableReplayCache:     true,
	ReplayCacheTTL:        24 * time.Hour,
	MinNonceLength:        8,
	MaxNonceLength:        64,
	EnableRateLimitPerIP:  false,
	RateLimitPerIPLimit:   100,
	RateLimitPerIPWindow:  time.Minute,
	EnableIntegrityCheck:  true,
	BodyIntegrityHeader:   "X-Body-Integrity",
	AdditionalHeaders:     []string{"X-Request-ID", "X-Forwarded-For"},
	SignatureVersion:      "2.0",
	DebugMode:             false,
}

var globalEnhancedNonceCache = &enhancedNonceCache{
	records: make(map[string]*nonceRecord),
	limit:   100000,
}

var globalEnhancedSignatureState = &enhancedSignatureState{
	sequenceCounters: make(map[string]int64),
	ipRequestCounts:  make(map[string]*ipRequestCounter),
}

func init() {
	go globalEnhancedNonceCache.cleanupLoop()
	go globalEnhancedSignatureState.cleanupLoop()
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

			if cfg.DebugMode {
				logEnhancedSignatureFailure(c, &result, signature, expectedSignature, startTime)
			}

			c.AbortWithStatusJSON(401, gin.H{
				"error":   "invalid_signature",
				"message": "Signature verification failed",
			})
			return
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
		HMAC_SHA512      bool `json:"hmac_sha512"`
		DoubleSignature  bool `json:"double_signature"`
		SequenceCheck    bool `json:"sequence_check"`
		ReplayProtection bool `json:"replay_protection"`
		IntegrityCheck   bool `json:"integrity_check"`
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
	return info
}

func NewEnhancedSignatureConfig(secretKey string) EnhancedSignatureConfig {
	return EnhancedSignatureConfig{
		SecretKey:             secretKey,
		Algorithm:             "SHA256",
		TimestampTolerance:    5 * time.Minute,
		RequireTimestamp:      true,
		RequireNonce:          true,
		NonceCacheTTL:         24 * time.Hour,
		SignatureHeader:       "X-Signature",
		TimestampHeader:       "X-Timestamp",
		NonceHeader:           "X-Nonce",
		ExcludePaths:          []string{},
		EnableHMAC_SHA512:     false,
		EnableDoubleSignature: false,
		EnableSequenceCheck:   false,
		MaxSequenceGap:        10,
		EnableReplayCache:     true,
		ReplayCacheTTL:        24 * time.Hour,
		MinNonceLength:        8,
		MaxNonceLength:        64,
		EnableRateLimitPerIP:  false,
		RateLimitPerIPLimit:   100,
		RateLimitPerIPWindow:  time.Minute,
		EnableIntegrityCheck:  true,
		BodyIntegrityHeader:   "X-Body-Integrity",
		AdditionalHeaders:     []string{"X-Request-ID", "X-Forwarded-For"},
		SignatureVersion:      "2.0",
		DebugMode:             false,
	}
}

const EnhancedSignatureVersion = "2.0"

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

type Ed25519Config struct {
	Enabled          bool
	PublicKeyPath    string
	PrivateKeyPath   string
	SignatureTTL     time.Duration
	RequireSignature bool
	PublicKey        ed25519.PublicKey
	PrivateKey       ed25519.PrivateKey
}

func (e *Ed25519Config) Load() error {
	if e.PublicKeyPath != "" {
		data, err := os.ReadFile(e.PublicKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read public key file: %w", err)
		}
		publicKey, err := crypto.ParseEd25519PublicKeyFromPEM(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse public key: %w", err)
		}
		e.PublicKey = publicKey
	}

	if e.PrivateKeyPath != "" {
		data, err := os.ReadFile(e.PrivateKeyPath)
		if err != nil {
			return fmt.Errorf("failed to read private key file: %w", err)
		}
		privateKey, err := crypto.ParseEd25519PrivateKeyFromPEM(string(data))
		if err != nil {
			return fmt.Errorf("failed to parse private key: %w", err)
		}
		e.PrivateKey = privateKey
	}

	return nil
}

func GenerateEd25519KeyPair() ([]byte, []byte, error) {
	privateKey, publicKey, err := crypto.GenerateEd25519KeyPair()
	if err != nil {
		return nil, nil, err
	}
	return privateKey, publicKey, nil
}

func SignEd25519(message, privateKey []byte) ([]byte, error) {
	if len(privateKey) != ed25519.PrivateKeySize {
		return nil, fmt.Errorf("invalid private key size")
	}
	return crypto.SignEd25519(message, ed25519.PrivateKey(privateKey))
}

func VerifyEd25519(message, signature, publicKey []byte) (bool, error) {
	if len(publicKey) != ed25519.PublicKeySize {
		return false, fmt.Errorf("invalid public key size")
	}
	return crypto.VerifyEd25519(message, signature, publicKey)
}

func SignEd25519String(message string, privateKey ed25519.PrivateKey) (string, error) {
	return crypto.SignEd25519String(message, privateKey)
}

func VerifyEd25519String(message, signatureBase64 string, publicKey ed25519.PublicKey) (bool, error) {
	return crypto.VerifyEd25519String(message, signatureBase64, publicKey)
}

type RequestEncryptionConfig struct {
	Enabled                     bool
	EncryptionKey               []byte
	Algorithm                   string
	EnablePayloadEncryption     bool
	EnableResponseEncryption    bool
	KeyRotationInterval         time.Duration
	CurrentKeyVersion           int
	KeyHistory                  [][]byte
	EnablePerfectForwardSecrecy bool
}

var defaultRequestEncryptionConfig = RequestEncryptionConfig{
	Enabled:                     false,
	Algorithm:                   "AES-256-GCM",
	EnablePayloadEncryption:     false,
	EnableResponseEncryption:    false,
	KeyRotationInterval:         24 * time.Hour,
	CurrentKeyVersion:           1,
	KeyHistory:                  make([][]byte, 0),
	EnablePerfectForwardSecrecy: false,
}

type EncryptedRequest struct {
	Version       int    `json:"v"`
	KeyVersion    int    `json:"kv"`
	EncryptedData string `json:"d"`
	IV            string `json:"iv"`
	AuthTag       string `json:"tag"`
	Timestamp     int64  `json:"t"`
	Signature     string `json:"s"`
}

func EncryptRequestBody(body []byte, config RequestEncryptionConfig) (*EncryptedRequest, error) {
	if !config.Enabled || !config.EnablePayloadEncryption {
		return nil, fmt.Errorf("request encryption not enabled")
	}

	if len(config.EncryptionKey) == 0 {
		return nil, fmt.Errorf("encryption key not set")
	}

	block, err := aes.NewCipher(config.EncryptionKey)
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

	ciphertext := gcm.Seal(nil, nonce, body, nil)

	authTagSize := gcm.Overhead()
	authTag := ciphertext[len(ciphertext)-authTagSize:]
	encryptedData := ciphertext[:len(ciphertext)-authTagSize]

	signature := sha256.Sum256(append(body, nonce...))

	return &EncryptedRequest{
		Version:       1,
		KeyVersion:    config.CurrentKeyVersion,
		EncryptedData: base64.StdEncoding.EncodeToString(encryptedData),
		IV:            base64.StdEncoding.EncodeToString(nonce),
		AuthTag:       base64.StdEncoding.EncodeToString(authTag),
		Timestamp:     time.Now().Unix(),
		Signature:     hex.EncodeToString(signature[:]),
	}, nil
}

func DecryptRequestBody(encrypted *EncryptedRequest, config RequestEncryptionConfig) ([]byte, error) {
	if !config.Enabled || !config.EnablePayloadEncryption {
		return nil, fmt.Errorf("request encryption not enabled")
	}

	var key []byte
	if encrypted.KeyVersion < config.CurrentKeyVersion {
		if encrypted.KeyVersion-1 < len(config.KeyHistory) {
			key = config.KeyHistory[encrypted.KeyVersion-1]
		} else {
			return nil, fmt.Errorf("key version not found in history")
		}
	} else {
		key = config.EncryptionKey
	}

	nonce, err := base64.StdEncoding.DecodeString(encrypted.IV)
	if err != nil {
		return nil, err
	}

	encryptedData, err := base64.StdEncoding.DecodeString(encrypted.EncryptedData)
	if err != nil {
		return nil, err
	}

	authTag, err := base64.StdEncoding.DecodeString(encrypted.AuthTag)
	if err != nil {
		return nil, err
	}

	ciphertext := append(encryptedData, authTag...)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	signature := sha256.Sum256(append(plaintext, nonce...))
	if hex.EncodeToString(signature[:]) != encrypted.Signature {
		return nil, fmt.Errorf("signature verification failed")
	}

	return plaintext, nil
}

func RotateEncryptionKey(config *RequestEncryptionConfig) error {
	if len(config.KeyHistory) >= 10 {
		config.KeyHistory = config.KeyHistory[1:]
	}

	config.KeyHistory = append(config.KeyHistory, config.EncryptionKey)

	newKey := make([]byte, len(config.EncryptionKey))
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return err
	}

	config.EncryptionKey = newKey
	config.CurrentKeyVersion++

	return nil
}

func EnhancedRequestEncryption() gin.HandlerFunc {
	config := defaultRequestEncryptionConfig
	config.Enabled = true

	return func(c *gin.Context) {
		if !config.EnablePayloadEncryption {
			c.Next()
			return
		}

		if c.Request.Body == nil {
			c.Next()
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.Next()
			return
		}

		if c.GetHeader("X-Encrypted") == "true" {
			encrypted := &EncryptedRequest{}
			if err := json.Unmarshal(body, encrypted); err == nil {
				decrypted, err := DecryptRequestBody(encrypted, config)
				if err == nil {
					c.Request.Body = io.NopCloser(bytes.NewBuffer(decrypted))
					c.Set("decrypted_body", decrypted)
				}
			}
		} else {
			encrypted, err := EncryptRequestBody(body, config)
			if err == nil {
				c.Set("encrypted_request", encrypted)
				c.Header("X-Encrypted", "true")
			}
		}

		c.Next()
	}
}

type DoubleSignatureConfig struct {
	Enabled            bool
	PrimaryAlgorithm   string
	SecondaryAlgorithm string
	PrimaryKey         []byte
	SecondaryKey       []byte
	VerifyOrder        string
	RequireBothValid   bool
}

func (d *DoubleSignatureConfig) Validate() error {
	if !d.Enabled {
		return nil
	}

	if len(d.PrimaryKey) == 0 {
		return fmt.Errorf("primary key required for double signature")
	}

	if len(d.SecondaryKey) == 0 {
		return fmt.Errorf("secondary key required for double signature")
	}

	if d.RequireBothValid && d.PrimaryAlgorithm == d.SecondaryAlgorithm {
		return fmt.Errorf("algorithms must be different when both signatures required")
	}

	return nil
}

func GenerateDualSignature(message []byte, config DoubleSignatureConfig) (string, string, error) {
	if err := config.Validate(); err != nil {
		return "", "", err
	}

	primarySig := hmac.New(sha256.New, config.PrimaryKey)
	primarySig.Write(message)
	primarySignature := hex.EncodeToString(primarySig.Sum(nil))

	secondarySig := hmac.New(sha512.New, config.SecondaryKey)
	secondarySig.Write(message)
	secondarySignature := hex.EncodeToString(secondarySig.Sum(nil))

	return primarySignature, secondarySignature, nil
}

func VerifyDualSignature(message []byte, primarySig, secondarySig string, config DoubleSignatureConfig) (bool, bool, error) {
	if err := config.Validate(); err != nil {
		return false, false, err
	}

	primaryValid := false
	if config.PrimaryAlgorithm == "SHA256" {
		expectedPrimary := hmac.New(sha256.New, config.PrimaryKey)
		expectedPrimary.Write(message)
		expectedPrimaryHex := hex.EncodeToString(expectedPrimary.Sum(nil))
		primaryValid = primarySig == expectedPrimaryHex
	} else if config.PrimaryAlgorithm == "SHA512" {
		expectedPrimary := hmac.New(sha512.New, config.PrimaryKey)
		expectedPrimary.Write(message)
		expectedPrimaryHex := hex.EncodeToString(expectedPrimary.Sum(nil))
		primaryValid = primarySig == expectedPrimaryHex
	}

	secondaryValid := false
	if config.SecondaryAlgorithm == "SHA256" {
		expectedSecondary := hmac.New(sha256.New, config.SecondaryKey)
		expectedSecondary.Write(message)
		expectedSecondaryHex := hex.EncodeToString(expectedSecondary.Sum(nil))
		secondaryValid = secondarySig == expectedSecondaryHex
	} else if config.SecondaryAlgorithm == "SHA512" {
		expectedSecondary := hmac.New(sha512.New, config.SecondaryKey)
		expectedSecondary.Write(message)
		expectedSecondaryHex := hex.EncodeToString(expectedSecondary.Sum(nil))
		secondaryValid = secondarySig == expectedSecondaryHex
	}

	return primaryValid, secondaryValid, nil
}

type AntiReplayConfig struct {
	WindowSize           time.Duration
	MaxRequestsPerWindow int
	EnableSlidingWindow  bool
	EnableBloomFilter    bool
	BloomFilterSize      int
	BloomFilterHashCount int
	CacheBackend         string
}

type BloomFilter struct {
	bitArray  []bool
	size      int
	hashCount int
}

func NewBloomFilter(size, hashCount int) *BloomFilter {
	return &BloomFilter{
		bitArray:  make([]bool, size),
		size:      size,
		hashCount: hashCount,
	}
}

func (b *BloomFilter) Add(item string) {
	for i := 0; i < b.hashCount; i++ {
		hash := sha256.Sum256(append([]byte(item), byte(i)))
		index := binary.BigEndian.Uint64(hash[:]) % uint64(b.size)
		b.bitArray[index] = true
	}
}

func (b *BloomFilter) Contains(item string) bool {
	for i := 0; i < b.hashCount; i++ {
		hash := sha256.Sum256(append([]byte(item), byte(i)))
		index := binary.BigEndian.Uint64(hash[:]) % uint64(b.size)
		if !b.bitArray[index] {
			return false
		}
	}
	return true
}

func (b *BloomFilter) FalsePositiveRate() float64 {
	k := float64(b.hashCount)
	m := float64(b.size)
	n := float64(b.countItems())
	return math.Pow(1-math.Exp(-k*n/m), k)
}

func (b *BloomFilter) countItems() int {
	count := 0
	for _, v := range b.bitArray {
		if v {
			count++
		}
	}
	return count
}

var globalBloomFilter = NewBloomFilter(1000000, 7)

func CheckReplay(nonce string) bool {
	if globalBloomFilter.Contains(nonce) {
		return true
	}
	globalBloomFilter.Add(nonce)
	return false
}

func EnhancedAntiReplay(config AntiReplayConfig) gin.HandlerFunc {
	requestCounts := make(map[string][]time.Time)
	mu := sync.Mutex{}

	return func(c *gin.Context) {
		if !config.EnableSlidingWindow && !config.EnableBloomFilter {
			c.Next()
			return
		}

		nonce := c.GetHeader("X-Nonce")
		if nonce == "" {
			c.Next()
			return
		}

		if config.EnableBloomFilter {
			if CheckReplay(nonce) {
				c.AbortWithStatusJSON(429, gin.H{
					"error":   "replay_detected",
					"message": "Nonce already used",
				})
				return
			}
		}

		if config.EnableSlidingWindow {
			mu.Lock()
			clientIP := c.ClientIP()
			now := time.Now()
			windowStart := now.Add(-config.WindowSize)

			times := requestCounts[clientIP]
			validTimes := make([]time.Time, 0)
			for _, t := range times {
				if t.After(windowStart) {
					validTimes = append(validTimes, t)
				}
			}

			if len(validTimes) >= config.MaxRequestsPerWindow {
				mu.Unlock()
				c.AbortWithStatusJSON(429, gin.H{
					"error":   "rate_limit_exceeded",
					"message": fmt.Sprintf("Maximum %d requests per %v", config.MaxRequestsPerWindow, config.WindowSize),
				})
				return
			}

			validTimes = append(validTimes, now)
			requestCounts[clientIP] = validTimes
			mu.Unlock()
		}

		c.Next()
	}
}

// EnhancedAntiReplayConfig 增强的防重放攻击配置
type EnhancedAntiReplayConfig struct {
	WindowSize                time.Duration
	MaxRequestsPerWindow      int
	EnableSlidingWindow       bool
	EnableBloomFilter         bool
	EnableRedisCache          bool
	BloomFilterSize           int
	BloomFilterHashCount      int
	NonceCacheTTL             time.Duration
	TimestampTolerance        time.Duration
	StrictNonceValidation     bool
	EnableClientIDTracking    bool
	MaxNonceAge               time.Duration
}

var defaultEnhancedAntiReplayConfig = EnhancedAntiReplayConfig{
	WindowSize:                1 * time.Minute,
	MaxRequestsPerWindow:      100,
	EnableSlidingWindow:       true,
	EnableBloomFilter:         true,
	EnableRedisCache:          false,
	BloomFilterSize:           1000000,
	BloomFilterHashCount:      7,
	NonceCacheTTL:             24 * time.Hour,
	TimestampTolerance:        5 * time.Minute,
	StrictNonceValidation:     true,
	EnableClientIDTracking:    false,
	MaxNonceAge:               15 * time.Minute,
}

// NonceValidator nonce验证器
type NonceValidator struct {
	bloomFilter      *BloomFilter
	redisEnabled     bool
	nonceCacheTTL    time.Duration
	strictValidation bool
	MaxNonceAge      time.Duration
	mu               sync.RWMutex
}

var globalNonceValidator = &NonceValidator{
	bloomFilter:      NewBloomFilter(1000000, 7),
	redisEnabled:     false,
	nonceCacheTTL:    24 * time.Hour,
	strictValidation: true,
	MaxNonceAge:      15 * time.Minute,
}

func (v *NonceValidator) ValidateNonce(nonce string, timestamp int64) error {
	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	if v.strictValidation {
		if len(nonce) < 8 || len(nonce) > 128 {
			return fmt.Errorf("nonce must be between 8 and 128 characters")
		}

		for _, c := range nonce {
			if !((c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c >= '0' && c <= '9') || c == '-' || c == '_') {
				return fmt.Errorf("nonce contains invalid characters")
			}
		}
	}

	if v.bloomFilter.Contains(nonce) {
		return fmt.Errorf("nonce already used (potential replay attack)")
	}

	if v.redisEnabled && redis.Client != nil {
		ctx := context.Background()
		key := fmt.Sprintf("anti_replay:nonce:%s", nonce)
		exists, err := redis.Client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return fmt.Errorf("nonce already used in cache (potential replay attack)")
		}
		err = redis.Client.Set(ctx, key, "1", v.nonceCacheTTL).Err()
		if err != nil {
			fmt.Printf("[AntiReplay] Warning: failed to store nonce in redis: %v\n", err)
		}
	}

	v.mu.Lock()
	v.bloomFilter.Add(nonce)
	v.mu.Unlock()

	if timestamp > 0 {
		now := time.Now().Unix()
		if now-timestamp > int64(v.MaxNonceAge.Seconds()) {
			return fmt.Errorf("nonce expired (too old)")
		}
	}

	return nil
}

func EnhancedAntiReplayV2(config EnhancedAntiReplayConfig) gin.HandlerFunc {
	requestCounts := make(map[string]*slidingWindowCounter)
	mu := sync.Mutex{}

	validator := &NonceValidator{
		bloomFilter:      NewBloomFilter(config.BloomFilterSize, config.BloomFilterHashCount),
		redisEnabled:     config.EnableRedisCache,
		nonceCacheTTL:    config.NonceCacheTTL,
		strictValidation: config.StrictNonceValidation,
		MaxNonceAge:      config.MaxNonceAge,
	}

	return func(c *gin.Context) {
		nonce := c.GetHeader("X-Nonce")
		timestampStr := c.GetHeader("X-Timestamp")

		var timestamp int64
		if timestampStr != "" {
			var err error
			timestamp, err = strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				c.AbortWithStatusJSON(400, gin.H{
					"error":   "invalid_timestamp",
					"message": "X-Timestamp must be a valid Unix timestamp",
				})
				return
			}
		}

		if err := validator.ValidateNonce(nonce, timestamp); err != nil {
			c.AbortWithStatusJSON(429, gin.H{
				"error":   "replay_detected",
				"message": err.Error(),
			})
			return
		}

		if config.EnableSlidingWindow {
			mu.Lock()
			clientIP := c.ClientIP()

			counter, exists := requestCounts[clientIP]
			if !exists {
				counter = &slidingWindowCounter{
					requests: make([]time.Time, 0),
					window:   config.WindowSize,
				}
				requestCounts[clientIP] = counter
			}

			if counter.Count() >= config.MaxRequestsPerWindow {
				mu.Unlock()
				c.AbortWithStatusJSON(429, gin.H{
					"error":   "rate_limit_exceeded",
					"message": fmt.Sprintf("Maximum %d requests per %v", config.MaxRequestsPerWindow, config.WindowSize),
				})
				return
			}

			counter.AddRequest()
			mu.Unlock()
		}

		c.Next()
	}
}

type slidingWindowCounter struct {
	requests []time.Time
	window   time.Duration
}

func (c *slidingWindowCounter) AddRequest() {
	c.requests = append(c.requests, time.Now())
}

func (c *slidingWindowCounter) Count() int {
	now := time.Now()
	windowStart := now.Add(-c.window)

	count := 0
	for _, t := range c.requests {
		if t.After(windowStart) {
			count++
		}
	}
	return count
}

func (c *slidingWindowCounter) Cleanup() {
	now := time.Now()
	windowStart := now.Add(-c.window)

	validRequests := make([]time.Time, 0)
	for _, t := range c.requests {
		if t.After(windowStart) {
			validRequests = append(validRequests, t)
		}
	}
	c.requests = validRequests
}
