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
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"golang.org/x/crypto/blake2b"
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
	AlgorithmHeader       string
	ExcludePaths          []string
	EnableHMAC_SHA512     bool
	EnableBlake2b         bool
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
	EnableKeyRotation     bool
	KeyRotationInterval   time.Duration
	MaxKeyHistory         int
	EnableAuditLog        bool
	AuditLogPath          string
	EnablePerformanceLog  bool
	CacheSignatures       bool
	SignatureCacheSize    int
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

type SignatureAlgorithm string

const (
	AlgorithmHMACSHA256  SignatureAlgorithm = "HMAC-SHA256"
	AlgorithmHMACSHA512  SignatureAlgorithm = "HMAC-SHA512"
	AlgorithmBlake2b256  SignatureAlgorithm = "BLAKE2B-256"
	AlgorithmBlake2b512  SignatureAlgorithm = "BLAKE2B-512"
)

func (a SignatureAlgorithm) IsValid() bool {
	switch a {
	case AlgorithmHMACSHA256, AlgorithmHMACSHA512, AlgorithmBlake2b256, AlgorithmBlake2b512:
		return true
	default:
		return false
	}
}

func (a SignatureAlgorithm) GetHashFunc() func() hash.Hash {
	switch a {
	case AlgorithmHMACSHA256, AlgorithmBlake2b256:
		return sha256.New
	case AlgorithmHMACSHA512, AlgorithmBlake2b512:
		return sha512.New
	default:
		return sha256.New
	}
}

func (a SignatureAlgorithm) OutputLength() int {
	switch a {
	case AlgorithmBlake2b256, AlgorithmHMACSHA256:
		return 32
	case AlgorithmBlake2b512, AlgorithmHMACSHA512:
		return 64
	default:
		return 32
	}
}

type KeyRotationManager struct {
	mu              sync.RWMutex
	currentKey      []byte
	keyVersion      int
	keyHistory      []KeyVersion
	rotationPeriod  time.Duration
	maxHistory      int
	lastRotation    time.Time
	rotating        atomic.Bool
	rotationCount   int64
	onRotation      func(oldKey, newKey []byte)
}

type KeyVersion struct {
	Version   int
	Key       []byte
	CreatedAt time.Time
	ExpiresAt time.Time
	Active    bool
}

func NewKeyRotationManager(initialKey []byte, rotationPeriod time.Duration, maxHistory int) *KeyRotationManager {
	mgr := &KeyRotationManager{
		currentKey:     initialKey,
		keyVersion:    1,
		keyHistory:    make([]KeyVersion, 0),
		rotationPeriod: rotationPeriod,
		maxHistory:    maxHistory,
		lastRotation:  time.Now(),
	}

	mgr.keyHistory = append(mgr.keyHistory, KeyVersion{
		Version:   1,
		Key:       initialKey,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(rotationPeriod * time.Duration(maxHistory)),
		Active:    true,
	})

	return mgr
}

func (k *KeyRotationManager) GetCurrentKey() []byte {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.currentKey
}

func (k *KeyRotationManager) GetKeyVersion() int {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return k.keyVersion
}

func (k *KeyRotationManager) ShouldRotate() bool {
	k.mu.RLock()
	defer k.mu.RUnlock()
	return time.Since(k.lastRotation) >= k.rotationPeriod
}

func (k *KeyRotationManager) RotateKey() error {
	if k.rotating.Load() {
		return fmt.Errorf("rotation already in progress")
	}

	k.rotating.Store(true)
	defer k.rotating.Store(false)

	k.mu.Lock()
	defer k.mu.Unlock()

	oldKey := make([]byte, len(k.currentKey))
	copy(oldKey, k.currentKey)

	newKey := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, newKey); err != nil {
		return fmt.Errorf("failed to generate new key: %w", err)
	}

	for i := range k.keyHistory {
		k.keyHistory[i].Active = false
	}

	k.keyVersion++
	newVersion := KeyVersion{
		Version:   k.keyVersion,
		Key:       newKey,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(k.rotationPeriod * time.Duration(k.maxHistory)),
		Active:    true,
	}

	k.keyHistory = append(k.keyHistory, newVersion)

	if len(k.keyHistory) > k.maxHistory {
		k.keyHistory = k.keyHistory[len(k.keyHistory)-k.maxHistory:]
	}

	k.currentKey = newKey
	k.lastRotation = time.Now()

	if k.onRotation != nil {
		go k.onRotation(oldKey, newKey)
	}

	return nil
}

func (k *KeyRotationManager) GetHistoricalKey(version int) ([]byte, bool) {
	k.mu.RLock()
	defer k.mu.RUnlock()

	for _, kv := range k.keyHistory {
		if kv.Version == version {
			return kv.Key, true
		}
	}
	return nil, false
}

func (k *KeyRotationManager) GetKeyHistory() []KeyVersion {
	k.mu.RLock()
	defer k.mu.RUnlock()

	history := make([]KeyVersion, len(k.keyHistory))
	copy(history, k.keyHistory)
	return history
}

func (k *KeyRotationManager) ValidateKey(key []byte) bool {
	k.mu.RLock()
	defer k.mu.RUnlock()

	if subtle.ConstantTimeCompare(k.currentKey, key) == 1 {
		return true
	}

	for _, kv := range k.keyHistory {
		if kv.Active && subtle.ConstantTimeCompare(kv.Key, key) == 1 {
			return true
		}
	}

	return false
}

type SignatureAuditLogger struct {
	mu          sync.Mutex
	logs        []SignatureAuditEntry
	maxLogs     int
	enableFile  bool
	logFile     *os.File
	enableJSON  bool
}

type SignatureAuditEntry struct {
	Timestamp      time.Time     `json:"timestamp"`
	RequestPath    string        `json:"request_path"`
	ClientIP       string        `json:"client_ip"`
	Algorithm      string        `json:"algorithm"`
	Signature      string        `json:"signature"`
	Valid          bool          `json:"valid"`
	Reason         string        `json:"reason"`
	ErrorCode      string        `json:"error_code,omitempty"`
	Duration       time.Duration `json:"duration"`
	UserAgent      string        `json:"user_agent,omitempty"`
	KeyVersion     int           `json:"key_version,omitempty"`
	ReplayDetected bool          `json:"replay_detected,omitempty"`
}

func NewSignatureAuditLogger(maxLogs int, logPath string) (*SignatureAuditLogger, error) {
	logger := &SignatureAuditLogger{
		logs:      make([]SignatureAuditEntry, 0, maxLogs),
		maxLogs:   maxLogs,
		enableFile: logPath != "",
		enableJSON: true,
	}

	if logger.enableFile {
		var err error
		logger.logFile, err = os.OpenFile(logPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
		if err != nil {
			return nil, fmt.Errorf("failed to open audit log file: %w", err)
		}
	}

	return logger, nil
}

func (l *SignatureAuditLogger) Log(entry SignatureAuditEntry) {
	l.mu.Lock()
	defer l.mu.Unlock()

	entry.Timestamp = time.Now()
	l.logs = append(l.logs, entry)

	if len(l.logs) > l.maxLogs {
		l.logs = l.logs[len(l.logs)-l.maxLogs:]
	}

	if l.enableFile && l.logFile != nil {
		data, _ := json.Marshal(entry)
		l.logFile.Write(data)
		l.logFile.Write([]byte("\n"))
	}
}

func (l *SignatureAuditLogger) GetLogs(limit int) []SignatureAuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	if limit <= 0 || limit > len(l.logs) {
		limit = len(l.logs)
	}

	logs := make([]SignatureAuditEntry, limit)
	copy(logs, l.logs[len(l.logs)-limit:])
	return logs
}

func (l *SignatureAuditLogger) GetFailedLogs(limit int) []SignatureAuditEntry {
	l.mu.Lock()
	defer l.mu.Unlock()

	failedLogs := make([]SignatureAuditEntry, 0)
	for i := len(l.logs) - 1; i >= 0 && len(failedLogs) < limit; i-- {
		if !l.logs[i].Valid {
			failedLogs = append(failedLogs, l.logs[i])
		}
	}

	return failedLogs
}

func (l *SignatureAuditLogger) Close() error {
	if l.logFile != nil {
		return l.logFile.Close()
	}
	return nil
}

type signatureCache struct {
	mu      sync.RWMutex
	cache   map[string]*cacheEntry
	maxSize int
	hits    int64
	misses  int64
}

type cacheEntry struct {
	signature []byte
	expiresAt time.Time
}

func newSignatureCache(maxSize int) *signatureCache {
	return &signatureCache{
		cache:   make(map[string]*cacheEntry),
		maxSize: maxSize,
	}
}

func (c *signatureCache) get(key string) ([]byte, bool) {
	c.mu.RLock()
	entry, exists := c.cache[key]
	c.mu.RUnlock()

	if !exists {
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	if time.Now().After(entry.expiresAt) {
		c.mu.Lock()
		delete(c.cache, key)
		c.mu.Unlock()
		atomic.AddInt64(&c.misses, 1)
		return nil, false
	}

	atomic.AddInt64(&c.hits, 1)
	return entry.signature, true
}

func (c *signatureCache) set(key string, signature []byte, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		c.evictOldest()
	}

	c.cache[key] = &cacheEntry{
		signature: signature,
		expiresAt: time.Now().Add(ttl),
	}
}

func (c *signatureCache) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, entry := range c.cache {
		if oldestTime.IsZero() || entry.expiresAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = entry.expiresAt
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
	}
}

func (c *signatureCache) clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*cacheEntry)
}

func (c *signatureCache) stats() (hits, misses int64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, len(c.cache)
}

type signatureValidator struct {
	config       EnhancedSignatureConfig
	nonceCache   *enhancedNonceCache
	state        *enhancedSignatureState
	keyManager   *KeyRotationManager
	auditLogger  *SignatureAuditLogger
	signatureCache *signatureCache
	performanceStats PerformanceStats
}

type PerformanceStats struct {
	totalValidations    int64
	totalValid          int64
	totalInvalid        int64
	totalDurationNanos  int64
	algorithmUsage       map[string]int64
	mu                  sync.RWMutex
}

func NewPerformanceStats() *PerformanceStats {
	return &PerformanceStats{
		algorithmUsage: make(map[string]int64),
	}
}

func (p *PerformanceStats) RecordValidation(algorithm string, valid bool, duration time.Duration) {
	atomic.AddInt64(&p.totalValidations, 1)

	p.mu.Lock()
	defer p.mu.Unlock()

	if valid {
		atomic.AddInt64(&p.totalValid, 1)
	} else {
		atomic.AddInt64(&p.totalInvalid, 1)
	}

	atomic.AddInt64(&p.totalDurationNanos, duration.Nanoseconds())
	p.algorithmUsage[algorithm]++
}

func (p *PerformanceStats) GetStats() map[string]interface{} {
	p.mu.RLock()
	defer p.mu.RUnlock()

	totalDuration := atomic.LoadInt64(&p.totalDurationNanos)
	totalValidations := atomic.LoadInt64(&p.totalValidations)

	return map[string]interface{}{
		"total_validations": atomic.LoadInt64(&p.totalValidations),
		"total_valid":       atomic.LoadInt64(&p.totalValid),
		"total_invalid":     atomic.LoadInt64(&p.totalInvalid),
		"average_duration":  time.Duration(totalDuration) / time.Duration(totalValidations),
		"algorithm_usage":   p.algorithmUsage,
	}
}

var defaultEnhancedSignatureConfig = EnhancedSignatureConfig{
	SecretKey:             "enhanced-secret-key-change-in-production",
	Algorithm:             "HMAC-SHA256",
	TimestampTolerance:    5 * time.Minute,
	RequireTimestamp:      true,
	RequireNonce:          true,
	NonceCacheTTL:         24 * time.Hour,
	SignatureHeader:       "X-Signature",
	TimestampHeader:       "X-Timestamp",
	NonceHeader:           "X-Nonce",
	AlgorithmHeader:       "X-Signature-Algorithm",
	ExcludePaths:          []string{"/health", "/api/health", "/metrics", "/api/metrics", "/swagger/*", "/docs/*"},
	EnableHMAC_SHA512:     false,
	EnableBlake2b:         true,
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
	SignatureVersion:      "3.0",
	DebugMode:             false,
	EnableKeyRotation:     false,
	KeyRotationInterval:   24 * time.Hour,
	MaxKeyHistory:         10,
	EnableAuditLog:        true,
	AuditLogPath:          "",
	EnablePerformanceLog:  true,
	CacheSignatures:       true,
	SignatureCacheSize:    10000,
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

var globalKeyRotationManager *KeyRotationManager
var globalAuditLogger *SignatureAuditLogger
var globalSignatureCache *signatureCache
var globalPerformanceStats *PerformanceStats

func init() {
	initialKey := []byte(defaultEnhancedSignatureConfig.SecretKey)
	globalKeyRotationManager = NewKeyRotationManager(initialKey, defaultEnhancedSignatureConfig.KeyRotationInterval, defaultEnhancedSignatureConfig.MaxKeyHistory)
	globalSignatureCache = newSignatureCache(defaultEnhancedSignatureConfig.SignatureCacheSize)
	globalPerformanceStats = NewPerformanceStats()

	if defaultEnhancedSignatureConfig.EnableAuditLog {
		var err error
		globalAuditLogger, err = NewSignatureAuditLogger(10000, defaultEnhancedSignatureConfig.AuditLogPath)
		if err != nil {
			fmt.Printf("[EnhancedSignature] Warning: failed to initialize audit logger: %v\n", err)
		}
	}

	if defaultEnhancedSignatureConfig.EnableKeyRotation {
		go startKeyRotationLoop()
	}
}

func startKeyRotationLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		if globalKeyRotationManager.ShouldRotate() {
			if err := globalKeyRotationManager.RotateKey(); err != nil {
				fmt.Printf("[EnhancedSignature] Key rotation failed: %v\n", err)
			} else {
				fmt.Printf("[EnhancedSignature] Key rotated successfully, new version: %d\n", globalKeyRotationManager.GetKeyVersion())
				if globalSignatureCache != nil {
					globalSignatureCache.clear()
					fmt.Printf("[EnhancedSignature] Signature cache cleared after key rotation\n")
				}
			}
		}
	}
}

func GetKeyRotationManager() *KeyRotationManager {
	return globalKeyRotationManager
}

func GetAuditLogger() *SignatureAuditLogger {
	return globalAuditLogger
}

func GetSignatureCache() *signatureCache {
	return globalSignatureCache
}

func GetPerformanceStats() *PerformanceStats {
	return globalPerformanceStats
}

func TriggerKeyRotation() error {
	if globalKeyRotationManager == nil {
		return fmt.Errorf("key rotation manager not initialized")
	}
	return globalKeyRotationManager.RotateKey()
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
	_, exists := n.records[nonce]
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

func computeSignatureWithAlgorithm(key, data string, algorithm SignatureAlgorithm) string {
	switch algorithm {
	case AlgorithmHMACSHA256:
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))

	case AlgorithmHMACSHA512:
		mac := hmac.New(sha512.New, []byte(key))
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))

	case AlgorithmBlake2b256:
		hash, err := blake2b.New256([]byte(key))
		if err != nil {
			return ""
		}
		hash.Write([]byte(data))
		return hex.EncodeToString(hash.Sum(nil))

	case AlgorithmBlake2b512:
		hash, err := blake2b.New512([]byte(key))
		if err != nil {
			return ""
		}
		hash.Write([]byte(data))
		return hex.EncodeToString(hash.Sum(nil))

	default:
		mac := hmac.New(sha256.New, []byte(key))
		mac.Write([]byte(data))
		return hex.EncodeToString(mac.Sum(nil))
	}
}

func computeBlake2b256(key, data []byte) ([]byte, error) {
	hash, err := blake2b.New256(key)
	if err != nil {
		return nil, err
	}
	hash.Write(data)
	return hash.Sum(nil), nil
}

func computeBlake2b512(key, data []byte) ([]byte, error) {
	hash, err := blake2b.New512(key)
	if err != nil {
		return nil, err
	}
	hash.Write(data)
	return hash.Sum(nil), nil
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

func GenerateSignatureWithAlgorithm(secretKey, method, path, query string, timestamp int64, nonce string, body []byte, algorithm SignatureAlgorithm, additionalData ...string) string {
	bodyHash := hashBodyEnhanced(body)
	stringToSign := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData...)
	return computeSignatureWithAlgorithm(secretKey, stringToSign, algorithm)
}

func GenerateSignatureWithKeyManager(keyManager *KeyRotationManager, method, path, query string, timestamp int64, nonce string, body []byte, algorithm SignatureAlgorithm, additionalData ...string) (string, int) {
	bodyHash := hashBodyEnhanced(body)
	stringToSign := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData...)
	key := keyManager.GetCurrentKey()
	signature := computeSignatureWithAlgorithm(string(key), stringToSign, algorithm)
	return signature, keyManager.GetKeyVersion()
}

func VerifySignatureWithAlgorithm(secretKey, method, path, query string, timestamp int64, nonce string, body []byte, providedSignature string, algorithm SignatureAlgorithm, additionalData ...string) bool {
	expectedSignature := GenerateSignatureWithAlgorithm(secretKey, method, path, query, timestamp, nonce, body, algorithm, additionalData...)
	return secureCompareEnhanced(providedSignature, expectedSignature)
}

func VerifySignatureWithKeyManager(keyManager *KeyRotationManager, method, path, query string, timestamp int64, nonce string, body []byte, providedSignature string, algorithm SignatureAlgorithm, keyVersion int, additionalData ...string) bool {
	bodyHash := hashBodyEnhanced(body)
	stringToSign := buildEnhancedStringToSign(method, path, query, timestamp, nonce, bodyHash, additionalData...)

	var key []byte
	if keyVersion == keyManager.GetKeyVersion() {
		key = keyManager.GetCurrentKey()
	} else {
		var found bool
		key, found = keyManager.GetHistoricalKey(keyVersion)
		if !found {
			return false
		}
	}

	expectedSignature := computeSignatureWithAlgorithm(string(key), stringToSign, algorithm)
	return secureCompareEnhanced(providedSignature, expectedSignature)
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
	SupportedAlgorithms []string `json:"supported_algorithms"`
	Features      struct {
		HMAC_SHA512      bool `json:"hmac_sha512"`
		Blake2b          bool `json:"blake2b"`
		DoubleSignature  bool `json:"double_signature"`
		SequenceCheck    bool `json:"sequence_check"`
		ReplayProtection bool `json:"replay_protection"`
		IntegrityCheck   bool `json:"integrity_check"`
		KeyRotation      bool `json:"key_rotation"`
		AuditLog         bool `json:"audit_log"`
		PerformanceLog   bool `json:"performance_log"`
		SignatureCache   bool `json:"signature_cache"`
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
		SupportedAlgorithms: []string{
			string(AlgorithmHMACSHA256),
			string(AlgorithmHMACSHA512),
			string(AlgorithmBlake2b256),
			string(AlgorithmBlake2b512),
		},
	}
	info.Features.HMAC_SHA512 = cfg.EnableHMAC_SHA512
	info.Features.Blake2b = cfg.EnableBlake2b
	info.Features.DoubleSignature = cfg.EnableDoubleSignature
	info.Features.SequenceCheck = cfg.EnableSequenceCheck
	info.Features.ReplayProtection = cfg.EnableReplayCache
	info.Features.IntegrityCheck = cfg.EnableIntegrityCheck
	info.Features.KeyRotation = cfg.EnableKeyRotation
	info.Features.AuditLog = cfg.EnableAuditLog
	info.Features.PerformanceLog = cfg.EnablePerformanceLog
	info.Features.SignatureCache = cfg.CacheSignatures
	return info
}

func NewEnhancedSignatureConfig(secretKey string) EnhancedSignatureConfig {
	return EnhancedSignatureConfig{
		SecretKey:             secretKey,
		Algorithm:             "HMAC-SHA256",
		TimestampTolerance:    5 * time.Minute,
		RequireTimestamp:      true,
		RequireNonce:          true,
		NonceCacheTTL:         24 * time.Hour,
		SignatureHeader:       "X-Signature",
		TimestampHeader:       "X-Timestamp",
		NonceHeader:           "X-Nonce",
		AlgorithmHeader:       "X-Signature-Algorithm",
		ExcludePaths:          []string{},
		EnableHMAC_SHA512:     false,
		EnableBlake2b:         true,
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
		SignatureVersion:      "3.0",
		DebugMode:             false,
		EnableKeyRotation:     false,
		KeyRotationInterval:   24 * time.Hour,
		MaxKeyHistory:         10,
		EnableAuditLog:        true,
		AuditLogPath:          "",
		EnablePerformanceLog:  true,
		CacheSignatures:       true,
		SignatureCacheSize:    10000,
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
}

func (e *Ed25519Config) Load() error {
	return nil
}

func GenerateEd25519KeyPair() ([]byte, []byte, error) {
	return nil, nil, fmt.Errorf("Ed25519 not supported in standard Go crypto library, use golang.org/x/crypto/ed25519")
}

func SignEd25519(message, privateKey []byte) ([]byte, error) {
	return nil, fmt.Errorf("Ed25519 not supported, use golang.org/x/crypto/ed25519")
}

func VerifyEd25519(message, signature, publicKey []byte) (bool, error) {
	return false, fmt.Errorf("Ed25519 not supported, use golang.org/x/crypto/ed25519")
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

	nonceSize := gcm.NonceSize()
	encryptedData := ciphertext[nonceSize:]
	authTag := ciphertext[:nonceSize]

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

	ciphertext := append(authTag, encryptedData...)

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
		primaryValid = hmac.Equal([]byte(primarySig), expectedPrimary.Sum(nil))
	} else if config.PrimaryAlgorithm == "SHA512" {
		expectedPrimary := hmac.New(sha512.New, config.PrimaryKey)
		expectedPrimary.Write(message)
		primaryValid = hmac.Equal([]byte(primarySig), expectedPrimary.Sum(nil))
	}

	secondaryValid := false
	if config.SecondaryAlgorithm == "SHA256" {
		expectedSecondary := hmac.New(sha256.New, config.SecondaryKey)
		expectedSecondary.Write(message)
		secondaryValid = hmac.Equal([]byte(secondarySig), expectedSecondary.Sum(nil))
	} else if config.SecondaryAlgorithm == "SHA512" {
		expectedSecondary := hmac.New(sha512.New, config.SecondaryKey)
		expectedSecondary.Write(message)
		secondaryValid = hmac.Equal([]byte(secondarySig), expectedSecondary.Sum(nil))
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
