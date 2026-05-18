package middleware

import (
	"bytes"
	"context"
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha3"
	"crypto/subtle"
	"crypto/ed25519"
	"encoding/base64"
	"encoding/binary"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"hash/crc32"
	"io"
	"math"
	"net/http"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type SecurityV2Config struct {
	SecretKey            string
	PrivateKey           []byte
	PublicKey            []byte
	Algorithm            string
	TimestampTolerance   time.Duration
	NonceCacheTTL        time.Duration
	SignatureHeader      string
	TimestampHeader      string
	NonceHeader          string
	VersionHeader        string
	ExcludePaths         []string
	EnableHMACSHA3       bool
	EnableEd25519        bool
	ReplayWindowSecs     int64
	ReplayMaxAge         time.Duration
	EnableTimestampCheck bool
	EnableNonceCheck     bool
	EnableVersionRouting bool
	DefaultAPIVersion    string
	SupportedVersions   []string
	TokenBucketRate      float64
	TokenBucketBurst     int
	TokenBucketRefillMs  int64
	EnableDistributed    bool
	RedisKeyPrefix       string
	DebugMode            bool
}

type EnhancedSecurityV2 struct {
	config           SecurityV2Config
	nonceCache       *nonceCacheV2
	tokenBuckets     map[string]*tokenBucket
	tokenBucketsMu   sync.RWMutex
	versionRouter    *versionRouter
	distributedLock  sync.Mutex
}

type nonceCacheV2 struct {
	records map[string]*nonceRecordV2
	mu      sync.RWMutex
	limit   int
}

type nonceRecordV2 struct {
	timestamp time.Time
	hash      string
	count     int
}

type tokenBucket struct {
	tokens     float64
	lastRefill time.Time
	rate       float64
	burst      int
	mu         sync.Mutex
}

type versionRouter struct {
	versionMap map[string]bool
	defaultV   string
	mu         sync.RWMutex
}

type Request struct {
	Method      string
	Path        string
	Query       string
	Body        []byte
	Headers     map[string]string
	Timestamp   int64
	Nonce       string
	Version     string
}

type SignatureResult struct {
	Valid       bool
	Algorithm   string
	Signature   string
	Timestamp   int64
	Nonce       string
	Version     string
	ErrorCode   string
	ErrorReason string
	ElapsedTime time.Duration
}

type ReplayCheckResult struct {
	Allowed         bool
	IsReplay        bool
	TimestampValid  bool
	NonceValid      bool
	RemainingWindow int64
	TTL             time.Duration
}

type RateLimitResult struct {
	Allowed     bool
	Remaining   int
	Limit       int
	ResetAt     time.Time
	RetryAfter  time.Duration
	IsDistributed bool
}

type VersionRouteResult struct {
	Version    string
	Compatible bool
	Handler    string
	DeprecationNotice string
}

var defaultSecurityV2Config = SecurityV2Config{
	SecretKey:            "enhanced-security-v2-secret-key-change-in-production",
	Algorithm:            "HMAC-SHA3-256",
	TimestampTolerance:   5 * time.Minute,
	NonceCacheTTL:        24 * time.Hour,
	SignatureHeader:      "X-Signature-V2",
	TimestampHeader:      "X-Timestamp-V2",
	NonceHeader:          "X-Nonce-V2",
	VersionHeader:        "X-API-Version",
	ExcludePaths:         []string{"/health", "/api/health", "/metrics", "/api/metrics"},
	EnableHMACSHA3:       true,
	EnableEd25519:        false,
	ReplayWindowSecs:     300,
	ReplayMaxAge:         5 * time.Minute,
	EnableTimestampCheck: true,
	EnableNonceCheck:     true,
	EnableVersionRouting: true,
	DefaultAPIVersion:    "v1",
	SupportedVersions:   []string{"v1", "v2", "v3"},
	TokenBucketRate:      100.0,
	TokenBucketBurst:     200,
	TokenBucketRefillMs:  100,
	EnableDistributed:    false,
	RedisKeyPrefix:       "security_v2:",
	DebugMode:            false,
}

var globalSecurityV2 *EnhancedSecurityV2
var globalOnce sync.Once

func NewEnhancedSecurityV2(config ...SecurityV2Config) *EnhancedSecurityV2 {
	globalOnce.Do(func() {
		cfg := defaultSecurityV2Config
		if len(config) > 0 {
			cfg = config[0]
		}

		globalSecurityV2 = &EnhancedSecurityV2{
			config:     cfg,
			nonceCache: newNonceCacheV2(100000),
			tokenBuckets: make(map[string]*tokenBucket),
			versionRouter: newVersionRouter(cfg.SupportedVersions, cfg.DefaultAPIVersion),
		}

		go globalSecurityV2.cleanupLoop()
	})

	return globalSecurityV2
}

func GetEnhancedSecurityV2() *EnhancedSecurityV2 {
	return NewEnhancedSecurityV2()
}

func newNonceCacheV2(limit int) *nonceCacheV2 {
	return &nonceCacheV2{
		records: make(map[string]*nonceRecordV2),
		limit:   limit,
	}
}

func newVersionRouter(versions []string, defaultV string) *versionRouter {
	vr := &versionRouter{
		versionMap: make(map[string]bool),
		defaultV:   defaultV,
	}
	for _, v := range versions {
		vr.versionMap[v] = true
	}
	return vr
}

func (n *nonceCacheV2) get(nonce string) (bool, *nonceRecordV2) {
	n.mu.RLock()
	defer n.mu.RUnlock()
	rec, exists := n.records[nonce]
	return exists, rec
}

func (n *nonceCacheV2) set(nonce string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if len(n.records) >= n.limit {
		n.shrinkToLimit()
	}
	n.records[nonce] = &nonceRecordV2{
		timestamp: time.Now(),
		hash:      hashNonceV2(nonce),
		count:     1,
	}
}

func (n *nonceCacheV2) increment(nonce string) {
	n.mu.Lock()
	defer n.mu.Unlock()
	if rec, exists := n.records[nonce]; exists {
		rec.count++
	}
}

func (n *nonceCacheV2) cleanup() {
	n.mu.Lock()
	defer n.mu.Unlock()
	now := time.Now()
	for nonce, rec := range n.records {
		if now.Sub(rec.timestamp) > 24*time.Hour {
			delete(n.records, nonce)
		}
	}
}

func (n *nonceCacheV2) shrinkToLimit() {
	count := 0
	halfLimit := n.limit / 2
	for nonce := range n.records {
		if count >= halfLimit {
			delete(n.records, nonce)
		}
		count++
	}
}

func (s *EnhancedSecurityV2) cleanupLoop() {
	ticker := time.NewTicker(15 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		s.nonceCache.cleanup()
		s.cleanupTokenBuckets()
	}
}

func (s *EnhancedSecurityV2) cleanupTokenBuckets() {
	s.tokenBucketsMu.Lock()
	defer s.tokenBucketsMu.Unlock()
	now := time.Now()
	for key, tb := range s.tokenBuckets {
		tb.mu.Lock()
		if now.Sub(tb.lastRefill) > 10*time.Minute {
			delete(s.tokenBuckets, key)
		}
		tb.mu.Unlock()
	}
}

func hashNonceV2(nonce string) string {
	h := sha3.New256()
	h.Write([]byte(nonce))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *EnhancedSecurityV2) EnhancedSignature(ctx context.Context, req *Request) (string, error) {
	if s.config.Algorithm == "Ed25519" && len(s.config.PrivateKey) == ed25519.PrivateKeySize {
		return s.signEd25519(req)
	}
	return s.signHMAC(req)
}

func (s *EnhancedSecurityV2) signHMAC(req *Request) (string, error) {
	var h hash.Hash
	switch s.config.Algorithm {
	case "HMAC-SHA3-256":
		h = sha3.New256()
	case "HMAC-SHA3-512":
		h = sha3.New512()
	default:
		h = sha3.New256()
	}

	stringToSign := s.buildStringToSign(req)

	if s.config.EnableHMACSHA3 && s.config.Algorithm == "HMAC-SHA3-256" {
		mac := hmac.New(func() hash.Hash { return sha3.New256() }, []byte(s.config.SecretKey))
		mac.Write([]byte(stringToSign))
		return hex.EncodeToString(mac.Sum(nil)), nil
	}

	mac := hmac.New(func() hash.Hash { return sha3.New256() }, []byte(s.config.SecretKey))
	mac.Write([]byte(stringToSign))
	return hex.EncodeToString(mac.Sum(nil)), nil
}

func (s *EnhancedSecurityV2) signEd25519(req *Request) (string, error) {
	stringToSign := s.buildStringToSign(req)

	signature, err := ed25519.Sign(s.config.PrivateKey, []byte(stringToSign))
	if err != nil {
		return "", fmt.Errorf("Ed25519 signing failed: %w", err)
	}

	return base64.StdEncoding.EncodeToString(signature), nil
}

func (s *EnhancedSecurityV2) buildStringToSign(req *Request) string {
	var parts []string

	parts = append(parts, strings.ToUpper(req.Method))
	parts = append(parts, req.Path)

	if req.Query != "" {
		parts = append(parts, s.sortQueryString(req.Query))
	}

	if req.Version != "" {
		parts = append(parts, req.Version)
	}

	parts = append(parts, strconv.FormatInt(req.Timestamp, 10))

	if req.Nonce != "" {
		parts = append(parts, req.Nonce)
	}

	if len(req.Body) > 0 {
		parts = append(parts, s.hashBody(req.Body))
	}

	return strings.Join(parts, "\n")
}

func (s *EnhancedSecurityV2) sortQueryString(query string) string {
	if query == "" {
		return ""
	}

	values, err := parseQuery(query)
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
		vals := values[key]
		for _, val := range vals {
			resultParts = append(resultParts, key+"="+val)
		}
	}

	return strings.Join(resultParts, "&")
}

func parseQuery(query string) (map[string][]string, error) {
	result := make(map[string][]string)
	pairs := strings.Split(query, "&")
	for _, pair := range pairs {
		kv := strings.SplitN(pair, "=", 2)
		if len(kv) == 2 {
			result[kv[0]] = append(result[kv[0]], kv[1])
		}
	}
	return result, nil
}

func (s *EnhancedSecurityV2) hashBody(body []byte) string {
	if len(body) == 0 {
		return ""
	}
	h := sha3.New256()
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *EnhancedSecurityV2) EnhancedReplayProtection(ctx context.Context, nonce string, timestamp int64) bool {
	if s.config.EnableTimestampCheck {
		now := time.Now().Unix()
		diff := math.Abs(float64(now - timestamp))
		if diff > s.config.ReplayWindowSecs {
			return false
		}
	}

	if s.config.EnableNonceCheck {
		if s.config.EnableDistributed && redis.Client != nil {
			redisKey := fmt.Sprintf("%sreplay:%s:%d", s.config.RedisKeyPrefix, nonce, timestamp)
			exists, err := redis.Client.Exists(ctx, redisKey).Result()
			if err == nil && exists > 0 {
				return false
			}
			err = redis.Client.Set(ctx, redisKey, "1", s.config.NonceCacheTTL).Err()
			if err != nil {
				fmt.Printf("[EnhancedSecurityV2] Warning: Redis nonce check failed: %v\n", err)
			}
		}

		exists, _ := s.nonceCache.get(nonce)
		if exists {
			return false
		}
		s.nonceCache.set(nonce)
	}

	return true
}

func (s *EnhancedSecurityV2) TokenBucketRateLimit(ctx context.Context, key string, rate float64, burst int) (bool, error) {
	if rate <= 0 {
		rate = s.config.TokenBucketRate
	}
	if burst <= 0 {
		burst = s.config.TokenBucketBurst
	}

	if s.config.EnableDistributed && redis.Client != nil {
		return s.tokenBucketDistributed(ctx, key, rate, burst)
	}

	return s.tokenBucketLocal(key, rate, burst)
}

func (s *EnhancedSecurityV2) tokenBucketLocal(key string, rate float64, burst int) (bool, error) {
	s.tokenBucketsMu.Lock()
	tb, exists := s.tokenBuckets[key]
	if !exists {
		tb = &tokenBucket{
			tokens:     float64(burst),
			lastRefill: time.Now(),
			rate:       rate,
			burst:      burst,
		}
		s.tokenBuckets[key] = tb
	}
	s.tokenBucketsMu.Unlock()

	tb.mu.Lock()
	defer tb.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Milliseconds()
	tb.lastRefill = now

	tokensToAdd := (float64(elapsed) / float64(s.config.TokenBucketRefillMs)) * tb.rate
	tb.tokens = math.Min(float64(tb.burst), tb.tokens+tokensToAdd)

	if tb.tokens >= 1 {
		tb.tokens--
		return true, nil
	}

	return false, nil
}

func (s *EnhancedSecurityV2) tokenBucketDistributed(ctx context.Context, key string, rate float64, burst int) (bool, error) {
	redisKey := fmt.Sprintf("%stoken_bucket:%s", s.config.RedisKeyPrefix, key)

	script := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local refill_ms = tonumber(ARGV[3])

		local bucket = redis.call('HMGET', key, 'tokens', 'last_refill')
		local tokens = tonumber(bucket[1])
		local last_refill = tonumber(bucket[2])

		if tokens == nil then
			tokens = burst
			last_refill = redis.call('TIME')[1] * 1000 + redis.call('TIME')[2] / 1000
		end

		local now = redis.call('TIME')[1] * 1000 + redis.call('TIME')[2] / 1000
		local elapsed = now - last_refill
		local tokens_to_add = (elapsed / refill_ms) * rate
		tokens = math.min(burst, tokens + tokens_to_add)

		if tokens >= 1 then
			tokens = tokens - 1
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, 60)
			return {1, tokens}
		else
			redis.call('HMSET', key, 'tokens', tokens, 'last_refill', now)
			redis.call('EXPIRE', key, 60)
			return {0, tokens}
		end
	`)

	result, err := script.Run(ctx, redis.Client, []string{redisKey}, rate, burst, s.config.TokenBucketRefillMs).Result()
	if err != nil {
		return s.tokenBucketLocal(key, rate, burst)
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return false, errors.New("unexpected script result")
	}

	allowed := arr[0].(int64) == 1
	remaining := int(arr[1].(float64))

	return allowed, nil
}

func (s *EnhancedSecurityV2) VersionRouting(ctx context.Context, path string) (string, error) {
	if !s.config.EnableVersionRouting {
		return s.config.DefaultAPIVersion, nil
	}

	pathParts := strings.Split(strings.Trim(path, "/"), "/")
	if len(pathParts) < 1 {
		return s.config.DefaultAPIVersion, nil
	}

	firstPart := pathParts[0]
	if strings.HasPrefix(firstPart, "v") {
		versionNum := strings.TrimPrefix(firstPart, "v")
		if _, err := strconv.Atoi(versionNum); err == nil {
			return firstPart, nil
		}
	}

	version := s.versionRouter.getVersion()
	return version, nil
}

func (v *versionRouter) getVersion() string {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.defaultV
}

func (v *versionRouter) isVersionSupported(version string) bool {
	v.mu.RLock()
	defer v.mu.RUnlock()
	return v.versionMap[version]
}

func (s *EnhancedSecurityV2) CheckReplay(ctx context.Context, nonce string, timestamp int64) *ReplayCheckResult {
	result := &ReplayCheckResult{
		Allowed:         true,
		TTL:             s.config.NonceCacheTTL,
		RemainingWindow: s.config.ReplayWindowSecs,
	}

	now := time.Now().Unix()
	timeDiff := now - timestamp

	if timeDiff < 0 {
		timeDiff = -timeDiff
	}

	if timeDiff > s.config.ReplayWindowSecs {
		result.Allowed = false
		result.TimestampValid = false
		result.RemainingWindow = 0
		return result
	}
	result.TimestampValid = true
	result.RemainingWindow = s.config.ReplayWindowSecs - timeDiff

	if s.nonceCache != nil {
		exists, _ := s.nonceCache.get(nonce)
		if exists {
			result.Allowed = false
			result.IsReplay = true
			result.NonceValid = false
			return result
		}
		result.NonceValid = true
	}

	return result
}

func (s *EnhancedSecurityV2) VerifySignature(req *Request, providedSignature string) *SignatureResult {
	startTime := time.Now()

	result := &SignatureResult{
		Algorithm: s.config.Algorithm,
		Signature: providedSignature,
	}

	expectedSignature, err := s.EnhancedSignature(context.Background(), req)
	if err != nil {
		result.ErrorCode = "SIGNATURE_ERROR"
		result.ErrorReason = err.Error()
		result.ElapsedTime = time.Since(startTime)
		return result
	}

	if !secureCompareV2(providedSignature, expectedSignature) {
		result.Valid = false
		result.ErrorCode = "SIGNATURE_MISMATCH"
		result.ErrorReason = "Signature verification failed"
		result.ElapsedTime = time.Since(startTime)
		return result
	}

	result.Valid = true
	result.Timestamp = req.Timestamp
	result.Nonce = req.Nonce
	result.Version = req.Version
	result.ElapsedTime = time.Since(startTime)

	return result
}

func secureCompareV2(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return subtle.ConstantTimeCompare([]byte(a), []byte(b)) == 1
}

func (s *EnhancedSecurityV2) CheckRateLimit(ctx context.Context, key string, rate float64, burst int) *RateLimitResult {
	result := &RateLimitResult{
		Limit:         burst,
		IsDistributed: s.config.EnableDistributed,
	}

	allowed, err := s.TokenBucketRateLimit(ctx, key, rate, burst)
	if err != nil {
		result.Allowed = true
		return result
	}

	result.Allowed = allowed
	if allowed {
		result.Remaining = burst - 1
		result.ResetAt = time.Now().Add(time.Duration(int64(burst/rate)*s.config.TokenBucketRefillMs) * time.Millisecond)
	} else {
		result.Remaining = 0
		result.RetryAfter = time.Duration(s.config.TokenBucketRefillMs) * time.Millisecond
		result.ResetAt = time.Now().Add(result.RetryAfter)
	}

	return result
}

func (s *EnhancedSecurityV2) RouteVersion(ctx context.Context, version string) *VersionRouteResult {
	result := &VersionRouteResult{
		Version:    version,
		Compatible: true,
	}

	if version == "" {
		version = s.config.DefaultAPIVersion
		result.Version = version
	}

	if !s.versionRouter.isVersionSupported(version) {
		result.Compatible = false
		result.DeprecationNotice = fmt.Sprintf("API version '%s' is not supported. Supported versions: %s",
			version, strings.Join(s.config.SupportedVersions, ", "))
		return result
	}

	if version == "v1" {
		result.DeprecationNotice = "API v1 is deprecated. Please upgrade to v2 or v3."
	}

	result.Handler = s.getHandlerForVersion(version)

	return result
}

func (s *EnhancedSecurityV2) getHandlerForVersion(version string) string {
	switch version {
	case "v1":
		return "legacyHandler"
	case "v2":
		return "standardHandler"
	case "v3":
		return "advancedHandler"
	default:
		return "unknownHandler"
	}
}

func EnhancedSecurityV2Middleware(config ...SecurityV2Config) gin.HandlerFunc {
	security := NewEnhancedSecurityV2(config...)

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		for _, excluded := range security.config.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				c.Next()
				return
			}
		}

		signature := c.GetHeader(security.config.SignatureHeader)
		timestampStr := c.GetHeader(security.config.TimestampHeader)
		nonce := c.GetHeader(security.config.NonceHeader)
		apiVersion := c.GetHeader(security.config.VersionHeader)

		if signature == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "missing_signature",
				"message": "X-Signature-V2 header is required",
			})
			return
		}

		var timestamp int64
		if timestampStr != "" {
			var err error
			timestamp, err = strconv.ParseInt(timestampStr, 10, 64)
			if err != nil {
				c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
					"error":   "invalid_timestamp",
					"message": "Invalid timestamp format",
				})
				return
			}
		}

		replayResult := security.CheckReplay(c.Request.Context(), nonce, timestamp)
		if !replayResult.Allowed {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   "replay_detected",
				"message": "Request replay detected or timestamp expired",
			})
			return
		}

		rateLimitKey := fmt.Sprintf("%s:%s:%s", c.ClientIP(), path, nonce)
		rateResult := security.CheckRateLimit(c.Request.Context(), rateLimitKey, 0, 0)
		if !rateResult.Allowed {
			c.Header("Retry-After", strconv.FormatInt(int64(rateResult.RetryAfter.Seconds()), 10))
			c.AbortWithStatusJSON(http.StatusTooManyRequests, gin.H{
				"error":      "rate_limit_exceeded",
				"message":    "Too many requests",
				"retry_after": rateResult.RetryAfter.Seconds(),
			})
			return
		}

		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
		}

		req := &Request{
			Method:    c.Request.Method,
			Path:      path,
			Query:     c.Request.URL.RawQuery,
			Body:      body,
			Timestamp: timestamp,
			Nonce:     nonce,
			Version:   apiVersion,
		}

		sigResult := security.VerifySignature(req, signature)
		if !sigResult.Valid {
			c.AbortWithStatusJSON(http.StatusUnauthorized, gin.H{
				"error":   sigResult.ErrorCode,
				"message": sigResult.ErrorReason,
			})
			return
		}

		versionResult := security.RouteVersion(c.Request.Context(), apiVersion)
		if !versionResult.Compatible {
			c.AbortWithStatusJSON(http.StatusBadRequest, gin.H{
				"error":   "unsupported_api_version",
				"message": versionResult.DeprecationNotice,
			})
			return
		}

		c.Set("security_v2_verified", true)
		c.Set("security_v2_result", sigResult)
		c.Set("security_v2_version", versionResult.Version)
		c.Set("security_v2_rate_limit", rateResult)

		if versionResult.DeprecationNotice != "" {
			c.Header("X-API-Deprecation", versionResult.DeprecationNotice)
		}

		c.Next()
	}
}

func GenerateSecurityV2Nonce(length int) (string, error) {
	if length < 16 {
		length = 32
	}
	if length > 64 {
		length = 64
	}

	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func GenerateSecurityV2Timestamp() int64 {
	return time.Now().Unix()
}

func GenerateSecurityV2TimestampMillis() int64 {
	return time.Now().UnixMilli()
}

func GenerateEd25519KeyPairV2() ([]byte, []byte, error) {
	publicKey, privateKey, err := ed25519.GenerateKey(rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate Ed25519 key pair: %w", err)
	}

	return publicKey, privateKey, nil
}

type DistributedTokenBucket struct {
	Key       string
	Rate      float64
	Burst     int
	RefillMs  int64
	RedisKey  string
}

func NewDistributedTokenBucket(key string, rate float64, burst int) *DistributedTokenBucket {
	return &DistributedTokenBucket{
		Key:      key,
		Rate:     rate,
		Burst:    burst,
		RefillMs: 100,
		RedisKey: fmt.Sprintf("security_v2:dtb:%s", key),
	}
}

func (dtb *DistributedTokenBucket) Allow(ctx context.Context) (bool, float64, error) {
	if redis.Client == nil {
		return false, 0, errors.New("redis client not available")
	}

	script := redis.NewScript(`
		local key = KEYS[1]
		local rate = tonumber(ARGV[1])
		local burst = tonumber(ARGV[2])
		local refill_ms = tonumber(ARGV[3])

		local data = redis.call('GET', key)
		local tokens = burst
		local last_time = 0

		if data then
			local parts = {}
			for part in string.gmatch(data, "[^:]+") do
				table.insert(parts, part)
			end
			if #parts >= 2 then
				tokens = tonumber(parts[1])
				last_time = tonumber(parts[2])
			end
		end

		local now = redis.call('TIME')[1] * 1000 + redis.call('TIME')[2] / 1000
		local elapsed = now - last_time
		local tokens_to_add = (elapsed / refill_ms) * rate
		tokens = math.min(burst, tokens + tokens_to_add)

		local allowed = 0
		if tokens >= 1 then
			tokens = tokens - 1
			allowed = 1
		end

		redis.call('SET', key, string.format("%.2f:%d", tokens, now), 'EX', 120)

		return {allowed, tokens}
	`)

	result, err := script.Run(ctx, redis.Client, []string{dtb.RedisKey}, dtb.Rate, dtb.Burst, dtb.RefillMs).Result()
	if err != nil {
		return false, 0, err
	}

	arr, ok := result.([]interface{})
	if !ok || len(arr) < 2 {
		return false, 0, errors.New("unexpected result format")
	}

	allowed := arr[0].(int64) == 1
	remaining := arr[1].(float64)

	return allowed, remaining, nil
}

func VerifyTimestampRange(timestamp int64, tolerance time.Duration) error {
	now := time.Now().Unix()
	diff := math.Abs(float64(now - timestamp))

	if diff > tolerance.Seconds() {
		return fmt.Errorf("timestamp out of tolerance: diff=%.2f seconds, tolerance=%.2f seconds", diff, tolerance.Seconds())
	}

	return nil
}

func GenerateNonceWithChecksum(length int) (string, error) {
	if length < 24 {
		length = 32
	}

	nonceBytes := make([]byte, length-4)
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return "", err
	}

	nonce := base64.URLEncoding.EncodeToString(nonceBytes)[:length-4]

	checksum := crc32.ChecksumIEEE([]byte(nonce))
	checksumBytes := make([]byte, 4)
	binary.BigEndian.PutUint32(checksumBytes, checksum)

	return nonce + base64.URLEncoding.EncodeToString(checksumBytes), nil
}

func VerifyNonceWithChecksum(nonce string) bool {
	if len(nonce) < 8 {
		return false
	}

	dataLen := len(nonce) - 4
	data := nonce[:dataLen]
	checksumStr := nonce[dataLen:]

	checksumBytes, err := base64.URLEncoding.DecodeString(checksumStr)
	if err != nil || len(checksumBytes) != 4 {
		return false
	}

	storedChecksum := binary.BigEndian.Uint32(checksumBytes)
	computedChecksum := crc32.ChecksumIEEE([]byte(data))

	return storedChecksum == computedChecksum
}

type SecurityMetrics struct {
	SignatureVerifications int64
	ReplayDetections       int64
	RateLimitRejections    int64
	VersionRoutings        int64
	TotalLatencyMs         int64
}

var globalSecurityMetrics = &SecurityMetrics{}
var metricsMu sync.RWMutex

func (s *EnhancedSecurityV2) GetMetrics() SecurityMetrics {
	metricsMu.RLock()
	defer metricsMu.RUnlock()
	return *globalSecurityMetrics
}

func IncrementSignatureVerifications() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	globalSecurityMetrics.SignatureVerifications++
}

func IncrementReplayDetections() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	globalSecurityMetrics.ReplayDetections++
}

func IncrementRateLimitRejections() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	globalSecurityMetrics.RateLimitRejections++
}

func IncrementVersionRoutings() {
	metricsMu.Lock()
	defer metricsMu.Unlock()
	globalSecurityMetrics.VersionRoutings++
}

func NewSecurityV2Config(secretKey string) SecurityV2Config {
	return SecurityV2Config{
		SecretKey:            secretKey,
		Algorithm:            "HMAC-SHA3-256",
		TimestampTolerance:   5 * time.Minute,
		NonceCacheTTL:        24 * time.Hour,
		SignatureHeader:      "X-Signature-V2",
		TimestampHeader:      "X-Timestamp-V2",
		NonceHeader:          "X-Nonce-V2",
		VersionHeader:        "X-API-Version",
		ExcludePaths:         []string{},
		EnableHMACSHA3:       true,
		EnableEd25519:        false,
		ReplayWindowSecs:     300,
		ReplayMaxAge:         5 * time.Minute,
		EnableTimestampCheck: true,
		EnableNonceCheck:     true,
		EnableVersionRouting: true,
		DefaultAPIVersion:    "v1",
		SupportedVersions:   []string{"v1", "v2", "v3"},
		TokenBucketRate:      100.0,
		TokenBucketBurst:     200,
		TokenBucketRefillMs:  100,
		EnableDistributed:    false,
		RedisKeyPrefix:       "security_v2:",
		DebugMode:            false,
	}
}

func CreateRequestForSignature(method, path, query string, body []byte, timestamp int64, nonce, version string) *Request {
	return &Request{
		Method:    method,
		Path:      path,
		Query:     query,
		Body:      body,
		Headers:   make(map[string]string),
		Timestamp: timestamp,
		Nonce:     nonce,
		Version:   version,
	}
}

func (s *EnhancedSecurityV2) ValidateRequest(req *Request, signature string) (bool, string) {
	if req == nil {
		return false, "nil request"
	}

	if req.Method == "" {
		return false, "missing method"
	}

	if req.Path == "" {
		return false, "missing path"
	}

	if !s.EnhancedReplayProtection(context.Background(), req.Nonce, req.Timestamp) {
		return false, "replay protection failed"
	}

	result := s.VerifySignature(req, signature)
	if !result.Valid {
		return false, result.ErrorReason
	}

	return true, "valid"
}

type SignatureContext struct {
	Security    *EnhancedSecurityV2
	Request     *Request
	Signature   string
	StartTime   time.Time
}

func NewSignatureContext(config ...SecurityV2Config) *SignatureContext {
	return &SignatureContext{
		Security:  NewEnhancedSecurityV2(config...),
		StartTime: time.Now(),
	}
}

func (sc *SignatureContext) SignRequest(req *Request) (string, error) {
	signature, err := sc.Security.EnhancedSignature(context.Background(), req)
	if err != nil {
		return "", err
	}
	sc.Signature = signature
	return signature, nil
}

func (sc *SignatureContext) VerifyRequest(req *Request, signature string) *SignatureResult {
	return sc.Security.VerifySignature(req, signature)
}

func (sc *SignatureContext) GetElapsedTime() time.Duration {
	return time.Since(sc.StartTime)
}

const SecurityV2Version = "2.0"
