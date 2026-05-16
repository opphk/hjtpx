package middleware

import (
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"hash"
	"io"
	"net/http"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type IntegrityConfig struct {
	EnableBodyHash       bool
	EnableSizeLimit      bool
	EnableContentType    bool
	MaxBodySize          int64
	AllowedContentTypes  []string
	HashAlgorithm        string
	RequireIntegrity     bool
	IntegrityHeader      string
	StoreHashesInRedis   bool
	HashTTL              time.Duration
	ExcludePaths         []string
}

type IntegrityResult struct {
	Valid          bool
	Reason         string
	BodySize       int64
	ContentType    string
	HashMatches    bool
	ProvidedHash   string
	ComputedHash   string
	Timestamp      time.Time
	ProcessingTime time.Duration
}

type BodyHashCache struct {
	hashes map[string]*CachedHash
	mu     sync.RWMutex
}

type CachedHash struct {
	Hash      string
	CreatedAt time.Time
	Count     int
}

var defaultIntegrityConfig = IntegrityConfig{
	EnableBodyHash:      true,
	EnableSizeLimit:     true,
	EnableContentType:   true,
	MaxBodySize:         10 * 1024 * 1024,
	AllowedContentTypes: []string{"application/json", "application/x-www-form-urlencoded", "multipart/form-data", "text/plain"},
	HashAlgorithm:       "SHA256",
	RequireIntegrity:    false,
	IntegrityHeader:     "X-Body-Integrity",
	StoreHashesInRedis:  true,
	HashTTL:             24 * time.Hour,
	ExcludePaths:        []string{"/health", "/api/health", "/metrics", "/api/metrics"},
}

var globalHashCache = &BodyHashCache{
	hashes: make(map[string]*CachedHash),
}

func (c *BodyHashCache) Get(key string) (string, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	
	if cached, exists := c.hashes[key]; exists {
		if time.Since(cached.CreatedAt) < 24*time.Hour {
			return cached.Hash, true
		}
	}
	return "", false
}

func (c *BodyHashCache) Set(key, hash string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	c.hashes[key] = &CachedHash{
		Hash:      hash,
		CreatedAt: time.Now(),
		Count:     1,
	}
}

func (c *BodyHashCache) Increment(key string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	if cached, exists := c.hashes[key]; exists {
		cached.Count++
	}
}

func (c *BodyHashCache) Cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	
	cutoff := time.Now().Add(-24 * time.Hour)
	for key, cached := range c.hashes {
		if cached.CreatedAt.Before(cutoff) {
			delete(c.hashes, key)
		}
	}
}

func init() {
	go func() {
		ticker := time.NewTicker(1 * time.Hour)
		defer ticker.Stop()
		for range ticker.C {
			globalHashCache.Cleanup()
		}
	}()
}

func ComputeBodyHash(body []byte, algorithm string) string {
	switch strings.ToUpper(algorithm) {
	case "SHA256":
		h := sha256.Sum256(body)
		return hex.EncodeToString(h[:])
	case "SHA512":
		h := sha512.New()
		h.Write(body)
		return hex.EncodeToString(h.Sum(nil))
	default:
		h := sha256.Sum256(body)
		return hex.EncodeToString(h[:])
	}
}

func ComputeHMACHash(body []byte, key []byte, algorithm string) string {
	var h hash.Hash
	switch strings.ToUpper(algorithm) {
	case "SHA256":
		h = hmac.New(sha256.New, key)
	case "SHA512":
		h = hmac.New(sha512.New, key)
	default:
		h = hmac.New(sha256.New, key)
	}
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

type RequestIntegrity struct {
	BodyHash      string    `json:"body_hash"`
	HMACHash      string    `json:"hmac_hash,omitempty"`
	Size          int64     `json:"size"`
	ContentType   string    `json:"content_type"`
	Timestamp     time.Time `json:"timestamp"`
	Nonce         string    `json:"nonce,omitempty"`
	PreviousHash  string    `json:"previous_hash,omitempty"`
}

func NewRequestIntegrity(body []byte, contentType string, hmacKey []byte) *RequestIntegrity {
	ri := &RequestIntegrity{
		BodyHash:    ComputeBodyHash(body, "SHA256"),
		Size:       int64(len(body)),
		ContentType: contentType,
		Timestamp:  time.Now(),
	}
	
	if hmacKey != nil {
		ri.HMACHash = ComputeHMACHash(body, hmacKey, "SHA256")
	}
	
	return ri
}

func (ri *RequestIntegrity) ToString() string {
	return fmt.Sprintf("%s:%d:%s:%d", 
		ri.BodyHash, 
		ri.Size, 
		ri.ContentType,
		ri.Timestamp.Unix(),
	)
}

func (ri *RequestIntegrity) ComputeOverallHash(secret string) string {
	data := ri.ToString()
	if secret != "" {
		data += secret
	}
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}

func IntegrityVerification(config ...IntegrityConfig) gin.HandlerFunc {
	cfg := defaultIntegrityConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		startTime := time.Now()
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

		result := &IntegrityResult{
			Timestamp: startTime,
		}

		if cfg.EnableSizeLimit {
			if c.Request.ContentLength > cfg.MaxBodySize {
				result.Valid = false
				result.Reason = fmt.Sprintf("request body size %d exceeds maximum %d", 
					c.Request.ContentLength, cfg.MaxBodySize)
				c.AbortWithStatusJSON(413, gin.H{
					"error":   "payload_too_large",
					"message": result.Reason,
				})
				return
			}
			result.BodySize = c.Request.ContentLength
		}

		if cfg.EnableContentType {
			contentType := c.GetHeader("Content-Type")
			if contentType != "" {
				baseType := strings.ToLower(strings.Split(contentType, ";")[0])
				isAllowed := false
				for _, allowed := range cfg.AllowedContentTypes {
					if strings.ToLower(allowed) == baseType {
						isAllowed = true
						break
					}
				}
				if !isAllowed {
					result.Valid = false
					result.Reason = fmt.Sprintf("content-type '%s' is not allowed", baseType)
					c.AbortWithStatusJSON(415, gin.H{
						"error":   "unsupported_media_type",
						"message": result.Reason,
					})
					return
				}
				result.ContentType = baseType
			}
		}

		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(io.LimitReader(c.Request.Body, cfg.MaxBodySize+1))
			if int64(len(body)) > cfg.MaxBodySize {
				result.Valid = false
				result.Reason = "request body exceeds size limit"
				c.AbortWithStatusJSON(413, gin.H{
					"error":   "payload_too_large",
					"message": result.Reason,
				})
				return
			}
			c.Request.Body = io.NopCloser(strings.NewReader(string(body)))
		}

		if cfg.EnableBodyHash && len(body) > 0 {
			computedHash := ComputeBodyHash(body, cfg.HashAlgorithm)
			result.ComputedHash = computedHash

			providedHash := c.GetHeader(cfg.IntegrityHeader)
			if providedHash != "" {
				result.ProvidedHash = providedHash
				if secureHashCompare(computedHash, providedHash) {
					result.HashMatches = true
					result.Valid = true
					result.Reason = "integrity verified"
				} else {
					result.Valid = false
					result.Reason = "body hash mismatch - possible tampering detected"
					c.AbortWithStatusJSON(400, gin.H{
						"error":   "integrity_check_failed",
						"message": result.Reason,
					})
					return
				}
			} else if cfg.RequireIntegrity {
				result.Valid = false
				result.Reason = "integrity header required but not provided"
				c.AbortWithStatusJSON(400, gin.H{
					"error":   "integrity_required",
					"message": result.Reason,
				})
				return
			} else {
				c.Header(cfg.IntegrityHeader, computedHash)
				result.Valid = true
				result.HashMatches = false
				result.Reason = "body hash computed and added to response header"
			}

			hashKey := fmt.Sprintf("integrity:%s:%d", 
				c.GetHeader("X-Request-ID"), 
				startTime.UnixNano())
			globalHashCache.Set(hashKey, computedHash)
		} else {
			result.Valid = true
			result.Reason = "integrity check passed"
		}

		result.ProcessingTime = time.Since(startTime)

		c.Set("integrity_verified", result.Valid)
		c.Set("integrity_result", result)
		c.Set("request_body_hash", result.ComputedHash)

		c.Next()
	}
}

func secureHashCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return hmac.Equal([]byte(a), []byte(b))
}

func RequireIntegrity() gin.HandlerFunc {
	return IntegrityVerification()
}

type IntegrityStats struct {
	TotalRequests      int64     `json:"total_requests"`
	PassedRequests     int64     `json:"passed_requests"`
	FailedRequests     int64     `json:"failed_requests"`
	AverageBodySize    int64     `json:"average_body_size"`
	AverageProcessTime float64   `json:"average_process_time"`
	LastUpdated        time.Time `json:"last_updated"`
}

var (
	integrityStatsMutex sync.RWMutex
	integrityStats      = &IntegrityStats{
		LastUpdated: time.Now(),
	}
)

func RecordIntegrityResult(result *IntegrityResult) {
	integrityStatsMutex.Lock()
	defer integrityStatsMutex.Unlock()
	
	integrityStats.TotalRequests++
	if result.Valid {
		integrityStats.PassedRequests++
	} else {
		integrityStats.FailedRequests++
	}
	
	totalSize := integrityStats.AverageBodySize * (integrityStats.TotalRequests - 1)
	integrityStats.AverageBodySize = (totalSize + result.BodySize) / integrityStats.TotalRequests
	
	totalTime := integrityStats.AverageProcessTime * float64(integrityStats.TotalRequests-1)
	integrityStats.AverageProcessTime = (totalTime + result.ProcessingTime.Seconds()) / float64(integrityStats.TotalRequests)
	
	integrityStats.LastUpdated = time.Now()
}

func GetIntegrityStats() *IntegrityStats {
	integrityStatsMutex.RLock()
	defer integrityStatsMutex.RUnlock()
	return integrityStats
}

func ResetIntegrityStats() {
	integrityStatsMutex.Lock()
	defer integrityStatsMutex.Unlock()
	integrityStats = &IntegrityStats{
		LastUpdated: time.Now(),
	}
}

type IntegrityReport struct {
	Timestamp      time.Time `json:"timestamp"`
	RequestHash   string    `json:"request_hash"`
	RequestSize   int64     `json:"request_size"`
	ContentType   string    `json:"content_type"`
	IsValid       bool      `json:"is_valid"`
	FailureReason string    `json:"failure_reason,omitempty"`
	ProcessingMs  float64   `json:"processing_ms"`
}

func GenerateIntegrityReport(c *gin.Context, result *IntegrityResult) *IntegrityReport {
	return &IntegrityReport{
		Timestamp:     result.Timestamp,
		RequestHash:  result.ComputedHash,
		RequestSize:  result.BodySize,
		ContentType:  result.ContentType,
		IsValid:      result.Valid,
		FailureReason: result.Reason,
		ProcessingMs: float64(result.ProcessingTime.Microseconds()) / 1000.0,
	}
}

func ValidateRequestIntegrity(c *gin.Context, expectedHash string) bool {
	if !cfg.EnableBodyHash {
		return true
	}
	
	providedHash := c.GetHeader(cfg.IntegrityHeader)
	if providedHash == "" {
		return false
	}
	
	return secureHashCompare(providedHash, expectedHash)
}

var cfg = defaultIntegrityConfig

func UpdateIntegrityConfig(newConfig IntegrityConfig) {
	integrityStatsMutex.Lock()
	defer integrityStatsMutex.Unlock()
	cfg = newConfig
}

func GetIntegrityConfig() IntegrityConfig {
	integrityStatsMutex.RLock()
	defer integrityStatsMutex.RUnlock()
	return cfg
}

type MultiLayerIntegrity struct {
	BodyHash      string `json:"body_hash"`
	HeaderHash    string `json:"header_hash"`
	CookieHash    string `json:"cookie_hash"`
	CombinedHash  string `json:"combined_hash"`
	Timestamp     int64  `json:"timestamp"`
	SequenceNum   int64  `json:"sequence_number"`
}

func ComputeMultiLayerIntegrity(c *gin.Context, body []byte, secretKey string) *MultiLayerIntegrity {
	mli := &MultiLayerIntegrity{
		BodyHash:  ComputeBodyHash(body, "SHA256"),
		Timestamp: time.Now().Unix(),
	}
	
	headerData := ""
	for _, key := range []string{"User-Agent", "Accept", "Accept-Language", "Content-Type"} {
		headerData += fmt.Sprintf("%s:%s;", key, c.GetHeader(key))
	}
	mli.HeaderHash = ComputeBodyHash([]byte(headerData), "SHA256")
	
	cookieData := ""
	for _, cookie := range c.Request.Cookies() {
		cookieData += fmt.Sprintf("%s:%s;", cookie.Name, cookie.Value)
	}
	mli.CookieHash = ComputeBodyHash([]byte(cookieData), "SHA256")
	
	combined := fmt.Sprintf("%s:%s:%s:%d", mli.BodyHash, mli.HeaderHash, mli.CookieHash, mli.Timestamp)
	if secretKey != "" {
		combined += ":" + secretKey
	}
	mli.CombinedHash = ComputeBodyHash([]byte(combined), "SHA256")
	
	return mli
}

func VerifyMultiLayerIntegrity(c *gin.Context, body []byte, expectedHash string, secretKey string) bool {
	mli := ComputeMultiLayerIntegrity(c, body, secretKey)
	return secureHashCompare(mli.CombinedHash, expectedHash)
}

func GetSequenceNumber(c *gin.Context) int64 {
	if seqStr := c.GetHeader("X-Sequence"); seqStr != "" {
		if seq, err := strconv.ParseInt(seqStr, 10, 64); err == nil {
			return seq
		}
	}
	return time.Now().UnixNano()
}

type IntegrityChallenge struct {
	ChallengeID  string    `json:"challenge_id"`
	Challenge    string    `json:"challenge"`
	ExpiresAt    time.Time `json:"expires_at"`
	Verified     bool      `json:"verified"`
}

var (
	challengeCache   = make(map[string]*IntegrityChallenge)
	challengeMutex   sync.RWMutex
)

func GenerateIntegrityChallenge() *IntegrityChallenge {
	challengeID := fmt.Sprintf("chal_%d_%s", time.Now().UnixNano(), randomString(16))
	challenge := &IntegrityChallenge{
		ChallengeID: challengeID,
		Challenge:   randomString(32),
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		Verified:   false,
	}
	
	challengeMutex.Lock()
	challengeCache[challengeID] = challenge
	challengeMutex.Unlock()
	
	go func() {
		time.Sleep(5 * time.Minute)
		challengeMutex.Lock()
		delete(challengeCache, challengeID)
		challengeMutex.Unlock()
	}()
	
	return challenge
}

func VerifyIntegrityChallenge(challengeID, response string) bool {
	challengeMutex.RLock()
	defer challengeMutex.RUnlock()
	
	if challenge, exists := challengeCache[challengeID]; exists {
		if time.Now().Before(challenge.ExpiresAt) {
			expectedResponse := ComputeBodyHash([]byte(challenge.Challenge), "SHA256")
			if secureHashCompare(expectedResponse, response) {
				challenge.Verified = true
				return true
			}
		}
	}
	return false
}

func randomString(length int) string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	result := make([]byte, length)
	for i := range result {
		result[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(result)
}

type IntegrityMiddlewareAdvanced struct {
	config              IntegrityConfig
	ed25519Manager     interface{}
	keyManager          interface{}
	redisEnabled        bool
	enableChallenge     bool
	enableSequenceCheck bool
	maxSequenceGap      int64
}

func NewIntegrityMiddlewareAdvanced(config IntegrityConfig) *IntegrityMiddlewareAdvanced {
	return &IntegrityMiddlewareAdvanced{
		config:              config,
		redisEnabled:        redis.Client != nil,
		enableChallenge:     true,
		enableSequenceCheck: true,
		maxSequenceGap:      1000,
	}
}

func (m *IntegrityMiddlewareAdvanced) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		startTime := time.Now()
		
		if m.enableChallenge {
			challengeID := c.GetHeader("X-Integrity-Challenge-ID")
			challengeResponse := c.GetHeader("X-Integrity-Challenge-Response")
			
			if challengeID != "" && challengeResponse != "" {
				if !VerifyIntegrityChallenge(challengeID, challengeResponse) {
					c.AbortWithStatusJSON(401, gin.H{
						"error":   "challenge_failed",
						"message": "Integrity challenge verification failed",
					})
					return
				}
			}
		}
		
		if m.enableSequenceCheck {
			sequenceNum := GetSequenceNumber(c)
			prevSeqKey := fmt.Sprintf("sequence:%s", c.ClientIP())
			
			if m.redisEnabled {
				ctx := context.Background()
				if prevSeqStr, err := redis.Client.Get(ctx, prevSeqKey).Result(); err == nil {
					if prevSeq, err := strconv.ParseInt(prevSeqStr, 10, 64); err == nil {
						if sequenceNum <= prevSeq {
							c.AbortWithStatusJSON(401, gin.H{
								"error":   "sequence_error",
								"message": "Request sequence number is not greater than previous",
							})
							return
						}
						if sequenceNum-prevSeq > m.maxSequenceGap {
							c.AbortWithStatusJSON(401, gin.H{
								"error":   "sequence_gap",
								"message": "Request sequence gap too large",
							})
							return
						}
					}
				}
				
				redis.Client.Set(ctx, prevSeqKey, strconv.FormatInt(sequenceNum, 10), 24*time.Hour)
			}
		}
		
		IntegrityVerification(m.config)(c)
		
		if !gin.IsDebugging() {
			processingTime := time.Since(startTime)
			if processingTime > 100*time.Millisecond {
				fmt.Printf("[Integrity] Slow request processing: %v for %s %s\n", 
					processingTime, c.Request.Method, c.Request.URL.Path)
			}
		}
	}
}
