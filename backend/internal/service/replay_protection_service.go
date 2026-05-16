package service

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"math"
	"net/http"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/crypto"
)

type ReplayProtectionConfig struct {
	TimestampTolerance time.Duration
	NonceCacheTTL     time.Duration
	EnableBodyHash    bool
	RequireSignature  bool
	SecretKey         string
	MaxClockSkew      time.Duration
}

type NonceEntry struct {
	Timestamp time.Time
	Used      bool
}

type ReplayProtectionService struct {
	nonceCache   map[string]*NonceEntry
	cacheMutex   sync.RWMutex
	config       ReplayProtectionConfig
	usedNonces   map[string]time.Time
	nonceMutex   sync.RWMutex
}

var defaultReplayConfig = ReplayProtectionConfig{
	TimestampTolerance: 5 * time.Minute,
	NonceCacheTTL:     24 * time.Hour,
	EnableBodyHash:    true,
	RequireSignature:  true,
	MaxClockSkew:      30 * time.Second,
}

func NewReplayProtectionService(config ...ReplayProtectionConfig) *ReplayProtectionService {
	cfg := defaultReplayConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	service := &ReplayProtectionService{
		nonceCache: make(map[string]*NonceEntry),
		usedNonces: make(map[string]time.Time),
		config:     cfg,
	}

	go service.cleanupRoutine()

	return service
}

func (s *ReplayProtectionService) cleanupRoutine() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.cleanupOldNonces()
	}
}

func (s *ReplayProtectionService) cleanupOldNonces() {
	s.nonceMutex.Lock()
	defer s.nonceMutex.Unlock()

	cutoff := time.Now().Add(-s.config.NonceCacheTTL)
	for nonce, timestamp := range s.usedNonces {
		if timestamp.Before(cutoff) {
			delete(s.usedNonces, nonce)
		}
	}

	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	for nonce, entry := range s.nonceCache {
		if entry.Timestamp.Before(cutoff) {
			delete(s.nonceCache, nonce)
		}
	}
}

func (s *ReplayProtectionService) GenerateNonce() (string, error) {
	randomBytes, err := crypto.GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return hex.EncodeToString(randomBytes), nil
}

func (s *ReplayProtectionService) CalculateSignature(secretKey, method, path, query string, timestamp int64, nonce string, body []byte) string {
	stringToSign := s.buildStringToSign(method, path, query, timestamp, nonce, body)
	return s.computeHMAC(secretKey, stringToSign)
}

func (s *ReplayProtectionService) buildStringToSign(method, path, query string, timestamp int64, nonce string, body []byte) string {
	var parts []string

	parts = append(parts, strings.ToUpper(method))
	parts = append(parts, path)

	if query != "" {
		sortedQuery := s.sortQueryString(query)
		parts = append(parts, sortedQuery)
	}

	parts = append(parts, strconv.FormatInt(timestamp, 10))

	if nonce != "" {
		parts = append(parts, nonce)
	}

	if s.config.EnableBodyHash && len(body) > 0 {
		bodyHash := crypto.HashSHA256(body)
		parts = append(parts, bodyHash)
	}

	return strings.Join(parts, "\n")
}

func (s *ReplayProtectionService) sortQueryString(query string) string {
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
		v := values[k]
		for _, val := range v {
			resultParts = append(resultParts, fmt.Sprintf("%s=%s", k, val))
		}
	}

	return strings.Join(resultParts, "&")
}

func (s *ReplayProtectionService) computeHMAC(secretKey, data string) string {
	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (s *ReplayProtectionService) VerifyTimestamp(timestamp int64) error {
	now := time.Now().Unix()
	diff := math.Abs(float64(now - timestamp))

	if diff > s.config.TimestampTolerance.Seconds() {
		return fmt.Errorf("timestamp out of tolerance: %v seconds", diff)
	}

	if math.Abs(float64(now-timestamp)) > s.config.MaxClockSkew.Seconds() {
		return fmt.Errorf("clock skew too large: %v seconds", diff)
	}

	return nil
}

func (s *ReplayProtectionService) VerifyNonce(nonce string) error {
	if nonce == "" {
		return fmt.Errorf("nonce is required")
	}

	if len(nonce) < 16 || len(nonce) > 128 {
		return fmt.Errorf("invalid nonce length: %d", len(nonce))
	}

	s.nonceMutex.RLock()
	if _, exists := s.usedNonces[nonce]; exists {
		s.nonceMutex.RUnlock()
		return fmt.Errorf("nonce already used: potential replay attack")
	}
	s.nonceMutex.RUnlock()

	s.nonceMutex.Lock()
	s.usedNonces[nonce] = time.Now()
	s.nonceMutex.Unlock()

	return nil
}

type ReplayVerificationResult struct {
	Valid     bool
	Reason    string
	Timestamp int64
	Nonce     string
	Signature string
	Elapsed   time.Duration
}

func (s *ReplayProtectionService) VerifyRequest(r *http.Request, secretKey string) *ReplayVerificationResult {
	startTime := time.Now()
	result := &ReplayVerificationResult{
		Valid: false,
	}

	signature := r.Header.Get("X-Signature")
	if signature == "" {
		result.Reason = "missing X-Signature header"
		return result
	}
	result.Signature = signature

	timestampStr := r.Header.Get("X-Timestamp")
	if timestampStr == "" {
		result.Reason = "missing X-Timestamp header"
		return result
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		result.Reason = "invalid X-Timestamp format"
		return result
	}
	result.Timestamp = timestamp

	if err := s.VerifyTimestamp(timestamp); err != nil {
		result.Reason = err.Error()
		return result
	}

	nonce := r.Header.Get("X-Nonce")
	if nonce == "" {
		result.Reason = "missing X-Nonce header"
		return result
	}
	result.Nonce = nonce

	if err := s.VerifyNonce(nonce); err != nil {
		result.Reason = err.Error()
		return result
	}

	var body []byte
	if r.Body != nil {
		body, _ = readRequestBody(r)
	}

	expectedSignature := s.CalculateSignature(
		secretKey,
		r.Method,
		r.URL.Path,
		r.URL.RawQuery,
		timestamp,
		nonce,
		body,
	)

	if !s.secureCompare(signature, expectedSignature) {
		result.Reason = "signature mismatch"
		return result
	}

	result.Valid = true
	result.Elapsed = time.Since(startTime)
	return result
}

func (s *ReplayProtectionService) secureCompare(a, b string) bool {
	if len(a) != len(b) {
		return false
	}
	return hmac.Equal([]byte(a), []byte(b))
}

func readRequestBody(r *http.Request) ([]byte, error) {
	if r.Body == nil {
		return nil, nil
	}

	body, err := io.ReadAll(r.Body)
	if err != nil {
		return nil, err
	}

	r.Body = io.NopCloser(strings.NewReader(string(body)))
	return body, nil
}

type RequestSignature struct {
	Method    string
	Path      string
	Query     string
	Timestamp int64
	Nonce     string
	Body      []byte
	Signature string
}

func (s *ReplayProtectionService) CreateSignedRequest(method, path, query string, body []byte, secretKey string) (*RequestSignature, error) {
	nonce, err := s.GenerateNonce()
	if err != nil {
		return nil, err
	}

	timestamp := time.Now().Unix()
	signature := s.CalculateSignature(secretKey, method, path, query, timestamp, nonce, body)

	return &RequestSignature{
		Method:    method,
		Path:      path,
		Query:     query,
		Timestamp: timestamp,
		Nonce:     nonce,
		Body:      body,
		Signature: signature,
	}, nil
}

func (s *ReplayProtectionService) AddHeadersToRequest(r *http.Request, sig *RequestSignature) {
	r.Header.Set("X-Signature", sig.Signature)
	r.Header.Set("X-Timestamp", strconv.FormatInt(sig.Timestamp, 10))
	r.Header.Set("X-Nonce", sig.Nonce)
}

func (s *ReplayProtectionService) GetNonceStats() map[string]interface{} {
	s.nonceMutex.RLock()
	defer s.nonceMutex.RUnlock()

	return map[string]interface{}{
		"total_nonces": len(s.usedNonces),
		"config": map[string]interface{}{
			"timestamp_tolerance": s.config.TimestampTolerance.String(),
			"nonce_cache_ttl":     s.config.NonceCacheTTL.String(),
			"enable_body_hash":    s.config.EnableBodyHash,
			"max_clock_skew":      s.config.MaxClockSkew.String(),
		},
	}
}

func (s *ReplayProtectionService) ClearNonceCache() {
	s.nonceMutex.Lock()
	defer s.nonceMutex.Unlock()
	s.usedNonces = make(map[string]time.Time)
}

func (s *ReplayProtectionService) UpdateConfig(config ReplayProtectionConfig) {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.config = config
}

func (s *ReplayProtectionService) GetConfig() ReplayProtectionConfig {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return s.config
}
