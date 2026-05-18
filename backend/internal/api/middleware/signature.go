package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type SignatureConfig struct {
	SecretKey            string
	Algorithm            string
	TimestampTolerance   time.Duration
	RequireTimestamp     bool
	RequireNonce         bool
	NonceCacheTTL        time.Duration
	SignatureHeader      string
	TimestampHeader      string
	NonceHeader          string
	ExcludePaths         []string
	EnableHMAC_SHA512    bool
	MinNonceLength       int
	MaxNonceLength       int
	EnableReplayCache    bool
	ReplayCacheTTL       time.Duration
	EnableIntegrityCheck bool
	BodyIntegrityHeader  string
}

type SignatureResult struct {
	Valid          bool
	Reason         string
	Timestamp      int64
	Nonce          string
	Signature      string
	ElapsedTime    time.Duration
	ErrorCode      string
	ClientIP       string
	RequestPath    string
	ReplayDetected bool
	IntegrityValid bool
}

type nonceCache struct {
	records map[string]*nonceRecord
	mu      sync.RWMutex
	limit   int
}

type signatureState struct {
	sequenceCounters map[string]int64
	ipRequestCounts  map[string]*ipRequestCounter
	mu               sync.RWMutex
}

var defaultSignatureConfig = SignatureConfig{
	SecretKey:            "enhanced-signature-key-change-in-production",
	Algorithm:            "SHA256",
	TimestampTolerance:   5 * time.Minute,
	RequireTimestamp:     true,
	RequireNonce:         true,
	NonceCacheTTL:        24 * time.Hour,
	SignatureHeader:      "X-Signature",
	TimestampHeader:      "X-Timestamp",
	NonceHeader:          "X-Nonce",
	ExcludePaths:         []string{"/health", "/api/health", "/metrics", "/api/metrics", "/swagger/*", "/docs/*"},
	EnableHMAC_SHA512:    false,
	MinNonceLength:       8,
	MaxNonceLength:       64,
	EnableReplayCache:    true,
	ReplayCacheTTL:       24 * time.Hour,
	EnableIntegrityCheck: true,
	BodyIntegrityHeader:  "X-Body-Integrity",
}

var globalSignatureState = &signatureState{
	sequenceCounters: make(map[string]int64),
	ipRequestCounts:  make(map[string]*ipRequestCounter),
}

var globalNonceCache = &nonceCache{
	records: make(map[string]*nonceRecord),
	limit:   100000,
}

func init() {
	go globalNonceCache.cleanupLoop()
	go globalSignatureState.cleanupLoop()
}

func (n *nonceCache) cleanupLoop() {
	ticker := time.NewTicker(30 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		n.cleanup()
	}
}

func (n *nonceCache) cleanup() {
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

func (n *nonceCache) shrinkToLimit() {
	count := 0
	limit := n.limit / 2
	for nonce := range n.records {
		if count >= limit {
			delete(n.records, nonce)
		}
		count++
	}
}

func (n *nonceCache) isUsed(nonce string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	hashedNonce := hashNonceGlobal(nonce)
	_, exists := n.records[hashedNonce]
	return exists
}

func (n *nonceCache) markUsed(nonce string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	hashedNonce := hashNonceGlobal(nonce)
	n.records[hashedNonce] = &nonceRecord{
		timestamp:   time.Now(),
		hashedNonce: hashedNonce,
		count:       1,
	}
}

func hashNonceGlobal(nonce string) string {
	h := sha256.New()
	h.Write([]byte(nonce))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *signatureState) cleanupLoop() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.cleanup()
	}
}

func (s *signatureState) cleanup() {
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

func calculateSignature(secretKey, method, path, query string, timestamp int64, nonce, bodyHash string) string {
	stringToSign := buildStringToSign(method, path, query, timestamp, nonce, bodyHash)
	return computeHMAC(secretKey, stringToSign, false)
}

func buildStringToSign(method, path, query string, timestamp int64, nonce, bodyHash string) string {
	var parts []string
	parts = append(parts, strings.ToUpper(method))
	parts = append(parts, path)

	if query != "" {
		sortedQuery := sortQueryString(query)
		parts = append(parts, sortedQuery)
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

func sortQueryString(query string) string {
	if query == "" {
		return ""
	}

	parts := strings.Split(query, "&")
	var params []string
	for _, part := range parts {
		if idx := strings.IndexByte(part, '='); idx > 0 {
			params = append(params, part[:idx])
		} else {
			params = append(params, part)
		}
	}

	for i := 0; i < len(params)-1; i++ {
		for j := i + 1; j < len(params); j++ {
			if params[i] > params[j] {
				params[i], params[j] = params[j], params[i]
			}
		}
	}

	return strings.Join(params, "&")
}

func computeHMAC(key, data string, useSHA512 bool) string {
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

func hashBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	h := sha256.New()
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func verifyTimestamp(timestamp int64, tolerance time.Duration) error {
	now := time.Now().Unix()
	diff := math.Abs(float64(now - timestamp))

	if diff > tolerance.Seconds() {
		return fmt.Errorf("timestamp out of tolerance: diff=%.2f seconds", diff)
	}

	return nil
}

func verifyNonce(nonce string, config SignatureConfig) error {
	if nonce == "" {
		return fmt.Errorf("nonce is empty")
	}

	if len(nonce) < config.MinNonceLength || len(nonce) > config.MaxNonceLength {
		return fmt.Errorf("nonce length invalid: must be between %d and %d characters", config.MinNonceLength, config.MaxNonceLength)
	}

	if !isValidNonceFormat(nonce) {
		return fmt.Errorf("nonce format invalid")
	}

	if globalNonceCache.isUsed(nonce) {
		return fmt.Errorf("nonce already used: potential replay attack")
	}

	if config.EnableReplayCache && redis.Client != nil {
		ctx := context.Background()
		key := fmt.Sprintf("signature:nonce:%s", hashNonceGlobal(nonce))
		exists, err := redis.Client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return fmt.Errorf("nonce already used in cache: potential replay attack")
		}
		err = redis.Client.Set(ctx, key, "1", config.NonceCacheTTL).Err()
		if err != nil {
			fmt.Printf("[Signature] Warning: failed to store nonce in redis: %v\n", err)
		}
	}

	globalNonceCache.markUsed(nonce)

	return nil
}

func SignatureVerification(config ...SignatureConfig) gin.HandlerFunc {
	cfg := defaultSignatureConfig
	if len(config) > 0 {
		cfg = config[0]
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
		result := SignatureResult{
			ClientIP:    clientIP,
			RequestPath: path,
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

			if err := verifyTimestamp(timestamp, cfg.TimestampTolerance); err != nil {
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

			if err := verifyNonce(nonce, cfg); err != nil {
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

		bodyHash := hashBody(body)

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

		expectedSignature := calculateSignature(
			cfg.SecretKey,
			method,
			path,
			query,
			timestamp,
			nonce,
			bodyHash,
		)

		if !secureCompare(signature, expectedSignature) {
			result.Valid = false
			result.Reason = "signature mismatch"
			result.ErrorCode = "SIGNATURE_MISMATCH"

			c.AbortWithStatusJSON(401, gin.H{
				"error":   "invalid_signature",
				"message": "Signature verification failed",
			})
			return
		}

		result.Valid = true
		result.Reason = "signature valid"
		result.ElapsedTime = time.Since(startTime)

		c.Set("signature_verified", true)
		c.Set("signature_timestamp", timestamp)
		c.Set("signature_nonce", nonce)
		c.Set("signature_result", &result)

		c.Next()
	}
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func GenerateSignature(secretKey, method, path, query string, timestamp int64, nonce string, body []byte) string {
	bodyHash := hashBody(body)
	return calculateSignature(secretKey, method, path, query, timestamp, nonce, bodyHash)
}

func GenerateNonceSecure(length int) (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789-_"

	if length < 8 {
		length = 16
	}
	if length > 64 {
		length = 64
	}

	bytes := make([]byte, length)
	for i := range bytes {
		idx := int(fastRandom(uint64(i))) % len(chars)
		bytes[i] = chars[idx]
	}

	return string(bytes), nil
}

func fastRandom(seed uint64) uint64 {
	seed ^= seed << 13
	seed ^= seed >> 7
	seed ^= seed << 17
	return seed
}

func GetSignatureInfo() map[string]interface{} {
	cfg := defaultSignatureConfig
	return map[string]interface{}{
		"algorithm":         cfg.Algorithm,
		"timestamp":         time.Now().Unix(),
		"nonce_required":    cfg.RequireNonce,
		"tolerance":         cfg.TimestampTolerance.String(),
		"version":           "2.0",
		"hmac_sha512":       cfg.EnableHMAC_SHA512,
		"replay_protection": cfg.EnableReplayCache,
		"integrity_check":   cfg.EnableIntegrityCheck,
	}
}

func RequireSignature() gin.HandlerFunc {
	return SignatureVerification()
}
