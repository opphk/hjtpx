package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"fmt"
	"math"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type SignatureConfig struct {
	SecretKey        string
	Algorithm        string
	TimestampTolerance time.Duration
	RequireTimestamp bool
	RequireNonce     bool
	NonceCacheTTL    time.Duration
	SignatureHeader  string
	TimestampHeader  string
	NonceHeader      string
	ExcludePaths     []string
}

type SignatureResult struct {
	Valid       bool
	Reason      string
	Timestamp   int64
	Nonce       string
	Signature   string
	ElapsedTime time.Duration
}

var defaultSignatureConfig = SignatureConfig{
	SecretKey:          "default-secret-key-change-in-production",
	Algorithm:          "SHA256",
	TimestampTolerance: 5 * time.Minute,
	RequireTimestamp:   true,
	RequireNonce:       true,
	NonceCacheTTL:      24 * time.Hour,
	SignatureHeader:    "X-Signature",
	TimestampHeader:   "X-Timestamp",
	NonceHeader:       "X-Nonce",
	ExcludePaths:      []string{"/health", "/api/health", "/metrics", "/api/metrics"},
}

type nonceCache struct {
	used map[string]time.Time
	mu   sync.RWMutex
}

var globalNonceCache = &nonceCache{
	used: make(map[string]time.Time),
}

func init() {
	go globalNonceCache.cleanup()
}

func (n *nonceCache) cleanup() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()
	for range ticker.C {
		n.mu.Lock()
		now := time.Now()
		for nonce, timestamp := range n.used {
			if now.Sub(timestamp) > 24*time.Hour {
				delete(n.used, nonce)
			}
		}
		n.mu.Unlock()
	}
}

func (n *nonceCache) isUsed(nonce string) bool {
	n.mu.RLock()
	defer n.mu.RUnlock()
	_, exists := n.used[nonce]
	return exists
}

func (n *nonceCache) markUsed(nonce string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	n.used[nonce] = time.Now()
}

func calculateSignature(secretKey, method, path, query string, timestamp int64, nonce, bodyHash string) string {
	stringToSign := buildStringToSign(method, path, query, timestamp, nonce, bodyHash)
	return computeHMAC(secretKey, stringToSign)
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

func computeHMAC(key, data string) string {
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
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
		return fmt.Errorf("timestamp out of tolerance: diff=%v seconds", diff)
	}

	return nil
}

func verifyNonce(nonce string, ttl time.Duration) error {
	if nonce == "" {
		return fmt.Errorf("nonce is empty")
	}

	if len(nonce) < 8 || len(nonce) > 64 {
		return fmt.Errorf("nonce length invalid: must be between 8 and 64 characters")
	}

	if globalNonceCache.isUsed(nonce) {
		return fmt.Errorf("nonce already used: potential replay attack")
	}

	if redis.Client != nil {
		ctx := context.Background()
		key := fmt.Sprintf("signature:nonce:%s", nonce)
		exists, err := redis.Client.Exists(ctx, key).Result()
		if err == nil && exists > 0 {
			return fmt.Errorf("nonce already used in redis: potential replay attack")
		}
		err = redis.Client.Set(ctx, key, "1", ttl).Err()
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

		signature := c.GetHeader(cfg.SignatureHeader)
		if signature == "" {
			c.AbortWithStatusJSON(401, gin.H{
				"error":   "missing_signature",
				"message": "X-Signature header is required",
			})
			return
		}

		var timestamp int64
		var timestampErr error

		if cfg.RequireTimestamp {
			timestampStr := c.GetHeader(cfg.TimestampHeader)
			if timestampStr == "" {
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "missing_timestamp",
					"message": "X-Timestamp header is required",
				})
				return
			}

			timestamp, timestampErr = strconv.ParseInt(timestampStr, 10, 64)
			if timestampErr != nil {
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "invalid_timestamp",
					"message": "X-Timestamp must be a valid Unix timestamp",
				})
				return
			}

			if err := verifyTimestamp(timestamp, cfg.TimestampTolerance); err != nil {
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
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "missing_nonce",
					"message": "X-Nonce header is required",
				})
				return
			}

			if err := verifyNonce(nonce, cfg.NonceCacheTTL); err != nil {
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
			body, _ = c.GetRawData()
			c.Request.Body = createBodyReaderForSignature(body)
		}

		bodyHash := hashBody(body)

		expectedSignature := calculateSignature(
			cfg.SecretKey,
			method,
			path,
			query,
			timestamp,
			nonce,
			bodyHash,
		)

		signatureValid := secureCompare(signature, expectedSignature)

		elapsed := time.Since(startTime)

		if !signatureValid {
			logSignatureFailure(c, signature, expectedSignature, timestamp, nonce, elapsed)
			c.AbortWithStatusJSON(401, gin.H{
				"error":   "invalid_signature",
				"message": "Signature verification failed",
			})
			return
		}

		c.Set("signature_verified", true)
		c.Set("signature_timestamp", timestamp)
		c.Set("signature_nonce", nonce)

		c.Next()
	}
}

func createBodyReaderForSignature(data []byte) *readCloser {
	return &readCloser{data: data, position: 0}
}

type readCloser struct {
	data     []byte
	position int
}

func (rc *readCloser) Read(p []byte) (n int, err error) {
	if rc.position >= len(rc.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(p, rc.data[rc.position:])
	rc.position += n
	return n, nil
}

func (rc *readCloser) Close() error {
	return nil
}

func secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func logSignatureFailure(c *gin.Context, provided, expected string, timestamp int64, nonce string, elapsed time.Duration) {
	clientIP := c.ClientIP()
	method := c.Request.Method
	path := c.Request.URL.Path
	userAgent := c.GetHeader("User-Agent")

	sigPreview := provided
	if len(sigPreview) > 16 {
		sigPreview = sigPreview[:16] + "..."
	}

	expectedPreview := expected
	if len(expectedPreview) > 16 {
		expectedPreview = expectedPreview[:16] + "..."
	}

	fmt.Printf("[SIGNATURE_FAILED] %s | %s %s | IP: %s | UA: %s | Timestamp: %d | Nonce: %s | Provided: %s | Expected: %s | Elapsed: %v\n",
		method,
		path,
		c.Request.URL.RawQuery,
		clientIP,
		userAgent,
		timestamp,
		nonce,
		sigPreview,
		expectedPreview,
		elapsed,
	)
}

func GenerateSignature(secretKey, method, path, query string, timestamp int64, nonce string, body []byte) string {
	bodyHash := hashBody(body)
	return calculateSignature(secretKey, method, path, query, timestamp, nonce, bodyHash)
}

func ValidateSignature(c *gin.Context, secretKey string) SignatureResult {
	startTime := time.Now()

	result := SignatureResult{
		Valid:       false,
		ElapsedTime: 0,
	}

	signature := c.GetHeader(defaultSignatureConfig.SignatureHeader)
	result.Signature = signature

	timestampStr := c.GetHeader(defaultSignatureConfig.TimestampHeader)
	if timestampStr != "" {
		timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
		if err == nil {
			result.Timestamp = timestamp
		}
	}

	nonce := c.GetHeader(defaultSignatureConfig.NonceHeader)
	result.Nonce = nonce

	method := c.Request.Method
	path := c.Request.URL.Path
	query := c.Request.URL.RawQuery

	var body []byte
	if c.Request.Body != nil {
		body, _ = c.GetRawData()
	}

	bodyHash := hashBody(body)

	expectedSignature := calculateSignature(
		secretKey,
		method,
		path,
		query,
		result.Timestamp,
		nonce,
		bodyHash,
	)

	if secureCompare(signature, expectedSignature) {
		result.Valid = true
		result.Reason = "signature valid"
	} else {
		result.Reason = "signature mismatch"
	}

	result.ElapsedTime = time.Since(startTime)

	return result
}

func RequireSignature() gin.HandlerFunc {
	return SignatureVerification()
}

type SignatureInfo struct {
	Algorithm      string `json:"algorithm"`
	Timestamp     int64  `json:"timestamp"`
	NonceRequired bool   `json:"nonce_required"`
	Tolerance     string `json:"tolerance"`
}

func GetSignatureInfo() SignatureInfo {
	return SignatureInfo{
		Algorithm:      defaultSignatureConfig.Algorithm,
		Timestamp:      time.Now().Unix(),
		NonceRequired:  defaultSignatureConfig.RequireNonce,
		Tolerance:      defaultSignatureConfig.TimestampTolerance.String(),
	}
}

func NewSignatureConfig(secretKey string) SignatureConfig {
	return SignatureConfig{
		SecretKey:          secretKey,
		Algorithm:          "SHA256",
		TimestampTolerance: 5 * time.Minute,
		RequireTimestamp:   true,
		RequireNonce:       true,
		NonceCacheTTL:      24 * time.Hour,
		SignatureHeader:    "X-Signature",
		TimestampHeader:   "X-Timestamp",
		NonceHeader:       "X-Nonce",
		ExcludePaths:      []string{},
	}
}
