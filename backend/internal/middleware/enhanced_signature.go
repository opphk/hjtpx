package middleware

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha512"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"hash"
	"io"
	"math"
	"strconv"
	"strings"
	"sync"
	"time"

	apperrors "github.com/hjtpx/hjtpx/internal/pkg/errors"
)

type SignatureAlgorithm string

const (
	AlgorithmHMACSHA256 SignatureAlgorithm = "HMAC-SHA256"
	AlgorithmHMACSHA384 SignatureAlgorithm = "HMAC-SHA384"
	AlgorithmHMACSHA512 SignatureAlgorithm = "HMAC-SHA512"
)

type SignatureConfig struct {
	SecretKey            []byte
	Algorithm            SignatureAlgorithm
	TimestampTolerance   time.Duration
	EnableNonce          bool
	EnableReplayCheck    bool
	MaxNonceCacheSize    int
	SignatureHeader      string
	TimestampHeader      string
	NonceHeader          string
	RequireHTTPS         bool
	AllowedOrigins       []string
	CustomHeaders        []string
}

var defaultSignatureConfig = SignatureConfig{
	Algorithm:          AlgorithmHMACSHA512,
	TimestampTolerance: 5 * time.Minute,
	EnableNonce:        true,
	EnableReplayCheck:  true,
	MaxNonceCacheSize:  10000,
	SignatureHeader:    "X-Signature",
	TimestampHeader:    "X-Timestamp",
	NonceHeader:        "X-Nonce",
	RequireHTTPS:       false,
}

type EnhancedSignature struct {
	config      SignatureConfig
	nonceCache  map[string]time.Time
	mu          sync.RWMutex
	stats       SignatureStats
}

type SignatureStats struct {
	TotalRequests    int64
	ValidSignatures  int64
	InvalidSignatures int64
	ReplayAttacks    int64
	ExpiredRequests  int64
	NonceCollisions  int64
}

type SignatureRequest struct {
	Method      string
	Path        string
	QueryParams map[string]string
	Headers     map[string]string
	Body        []byte
	Timestamp   int64
	Nonce       string
}

func NewEnhancedSignature(config ...SignatureConfig) *EnhancedSignature {
	cfg := defaultSignatureConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	if len(cfg.SecretKey) == 0 {
		cfg.SecretKey = []byte("hjtpx-signature-key-2024")
	}

	return &EnhancedSignature{
		config:     cfg,
		nonceCache: make(map[string]time.Time),
	}
}

func (s *EnhancedSignature) GenerateSignature(req SignatureRequest) (string, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.config.EnableNonce && req.Nonce == "" {
		nonce, err := GenerateNonce(32)
		if err != nil {
			return "", fmt.Errorf("failed to generate nonce: %w", err)
		}
		req.Nonce = nonce
	}

	if req.Timestamp == 0 {
		req.Timestamp = time.Now().Unix()
	}

	signatureData := s.buildSignatureData(req)
	signature := s.computeHMAC(signatureData)

	return signature, nil
}

func (s *EnhancedSignature) buildSignatureData(req SignatureRequest) string {
	var parts []string

	parts = append(parts, strings.ToUpper(req.Method))
	parts = append(parts, req.Path)

	if req.Timestamp > 0 {
		parts = append(parts, strconv.FormatInt(req.Timestamp, 10))
	}

	if req.Nonce != "" {
		parts = append(parts, req.Nonce)
	}

	if len(s.config.CustomHeaders) > 0 {
		for _, header := range s.config.CustomHeaders {
			if value, ok := req.Headers[header]; ok {
				parts = append(parts, value)
			}
		}
	}

	if len(req.QueryParams) > 0 {
		sortedParams := s.sortQueryParams(req.QueryParams)
		parts = append(parts, sortedParams)
	}

	if len(req.Body) > 0 {
		bodyHash := s.hashBody(req.Body)
		parts = append(parts, bodyHash)
	}

	return strings.Join(parts, "|")
}

func (s *EnhancedSignature) computeHMAC(data string) string {
	var h hash.Hash

	switch s.config.Algorithm {
	case AlgorithmHMACSHA256:
		h = hmac.New(sha512.New384, s.config.SecretKey)
	case AlgorithmHMACSHA384:
		h = hmac.New(sha512.New384, s.config.SecretKey)
	case AlgorithmHMACSHA512:
		h = hmac.New(sha512.New512_256, s.config.SecretKey)
	default:
		h = hmac.New(sha512.New512_256, s.config.SecretKey)
	}

	h.Write([]byte(data))
	return base64.StdEncoding.EncodeToString(h.Sum(nil))
}

func (s *EnhancedSignature) hashBody(body []byte) string {
	h := sha512.New512_256()
	h.Write(body)
	return hex.EncodeToString(h.Sum(nil))
}

func (s *EnhancedSignature) sortQueryParams(params map[string]string) string {
	if len(params) == 0 {
		return ""
	}

	var keys []string
	for k := range params {
		keys = append(keys, k)
	}

	sortedParams := make([]string, 0, len(params))
	for _, k := range keys {
		sortedParams = append(sortedParams, fmt.Sprintf("%s=%s", k, params[k]))
	}

	return strings.Join(sortedParams, "&")
}

func (s *EnhancedSignature) VerifySignature(req SignatureRequest, signature string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.stats.TotalRequests++

	if err := s.verifyTimestamp(req.Timestamp); err != nil {
		s.stats.ExpiredRequests++
		return err
	}

	if s.config.EnableReplayCheck {
		if err := s.checkReplay(req.Nonce, req.Timestamp); err != nil {
			return err
		}
	}

	expectedSignature, err := s.GenerateSignature(req)
	if err != nil {
		s.stats.InvalidSignatures++
		return fmt.Errorf("failed to generate signature: %w", err)
	}

	if !hmac.Equal([]byte(signature), []byte(expectedSignature)) {
		s.stats.InvalidSignatures++
		return errors.New("signature verification failed")
	}

	s.stats.ValidSignatures++

	return nil
}

func (s *EnhancedSignature) verifyTimestamp(timestamp int64) error {
	if timestamp == 0 {
		return apperrors.New(apperrors.CodeInvalidParams, "missing timestamp")
	}

	requestTime := time.Unix(timestamp, 0)
	now := time.Now()

	diff := now.Sub(requestTime)
	if diff < 0 {
		diff = -diff
	}

	if diff > s.config.TimestampTolerance {
		return fmt.Errorf("timestamp outside tolerance window: %v", diff)
	}

	return nil
}

func (s *EnhancedSignature) checkReplay(nonce string, timestamp int64) error {
	if nonce == "" {
		return apperrors.New(apperrors.CodeInvalidParams, "missing nonce")
	}

	key := s.buildNonceKey(nonce, timestamp)

	if _, exists := s.nonceCache[key]; exists {
		s.stats.ReplayAttacks++
		return apperrors.New(apperrors.CodeSecurityRisk, "replay attack detected: nonce already used")
	}

	s.nonceCache[key] = time.Now()

	if len(s.nonceCache) > s.config.MaxNonceCacheSize {
		s.cleanExpiredNonces()
	}

	return nil
}

func (s *EnhancedSignature) buildNonceKey(nonce string, timestamp int64) string {
	return fmt.Sprintf("%s:%d", nonce, timestamp)
}

func (s *EnhancedSignature) cleanExpiredNonces() {
	cutoff := time.Now().Add(-s.config.TimestampTolerance * 2)

	var toDelete []string
	for key, timestamp := range s.nonceCache {
		if timestamp.Before(cutoff) {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(s.nonceCache, key)
	}
}

func (s *EnhancedSignature) GetStats() SignatureStats {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.stats
}

func (s *EnhancedSignature) ResetStats() {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.stats = SignatureStats{}
}

type SignatureBuilder struct {
	signature    *EnhancedSignature
	request       SignatureRequest
	signatureData string
}

func NewSignatureBuilder(sig *EnhancedSignature) *SignatureBuilder {
	return &SignatureBuilder{
		signature: sig,
		request: SignatureRequest{
			QueryParams: make(map[string]string),
			Headers:     make(map[string]string),
		},
	}
}

func (b *SignatureBuilder) SetMethod(method string) *SignatureBuilder {
	b.request.Method = strings.ToUpper(method)
	return b
}

func (b *SignatureBuilder) SetPath(path string) *SignatureBuilder {
	b.request.Path = path
	return b
}

func (b *SignatureBuilder) SetQueryParam(key, value string) *SignatureBuilder {
	b.request.QueryParams[key] = value
	return b
}

func (b *SignatureBuilder) SetHeader(key, value string) *SignatureBuilder {
	b.request.Headers[strings.ToLower(key)] = value
	return b
}

func (b *SignatureBuilder) SetBody(body []byte) *SignatureBuilder {
	b.request.Body = body
	return b
}

func (b *SignatureBuilder) SetTimestamp(timestamp int64) *SignatureBuilder {
	b.request.Timestamp = timestamp
	return b
}

func (b *SignatureBuilder) SetNonce(nonce string) *SignatureBuilder {
	b.request.Nonce = nonce
	return b
}

func (b *SignatureBuilder) AddCustomHeaders(headers []string) *SignatureBuilder {
	b.signature.config.CustomHeaders = headers
	return b
}

func (b *SignatureBuilder) Build() (string, error) {
	return b.signature.GenerateSignature(b.request)
}

func (b *SignatureBuilder) Sign() (string, SignatureRequest, error) {
	sig, err := b.signature.GenerateSignature(b.request)
	if err != nil {
		return "", SignatureRequest{}, err
	}
	return sig, b.request, nil
}

func GenerateNonce(length int) (string, error) {
	if length < 16 {
		length = 32
	}

	bytes := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	return base64.URLEncoding.EncodeToString(bytes)[:length], nil
}

func GenerateSecureNonce() (string, error) {
	bytes := make([]byte, 32)
	if _, err := io.ReadFull(rand.Reader, bytes); err != nil {
		return "", fmt.Errorf("failed to generate secure nonce: %w", err)
	}

	hash := sha512.Sum512_256(bytes)
	return hex.EncodeToString(hash[:]), nil
}

func GenerateTimestampNonce() (timestamp int64, nonce string, err error) {
	timestamp = time.Now().UnixNano()

	nonceBytes := make([]byte, 16)
	if _, err := io.ReadFull(rand.Reader, nonceBytes); err != nil {
		return 0, "", fmt.Errorf("failed to generate nonce: %w", err)
	}

	nonce = fmt.Sprintf("%d-%s", timestamp, hex.EncodeToString(nonceBytes))
	return timestamp, nonce, nil
}

func GenerateHMACSignature(data, key []byte, algorithm SignatureAlgorithm) (string, error) {
	if len(key) == 0 {
		return "", apperrors.New(apperrors.CodeInvalidParams, "key cannot be empty")
	}

	var h hash.Hash
	switch algorithm {
	case AlgorithmHMACSHA256:
		h = hmac.New(sha512.New384, key)
	case AlgorithmHMACSHA384:
		h = hmac.New(sha512.New384, key)
	case AlgorithmHMACSHA512:
		h = hmac.New(sha512.New512_256, key)
	default:
		h = hmac.New(sha512.New512_256, key)
	}

	h.Write(data)
	return base64.StdEncoding.EncodeToString(h.Sum(nil)), nil
}

func VerifyHMACSignature(data, signature, key []byte, algorithm SignatureAlgorithm) (bool, error) {
	expected, err := GenerateHMACSignature(data, key, algorithm)
	if err != nil {
		return false, err
	}

	return hmac.Equal([]byte(expected), signature), nil
}

func GenerateSignatureWithExpiry(data []byte, key []byte, expiry time.Duration) (string, int64, error) {
	expiryTimestamp := time.Now().Add(expiry).Unix()

	combinedData := fmt.Sprintf("%s|%d", string(data), expiryTimestamp)

	h := hmac.New(sha512.New512_256, key)
	h.Write([]byte(combinedData))
	signature := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return signature, expiryTimestamp, nil
}

func VerifySignatureWithExpiry(data []byte, signature string, key []byte, expiryTimestamp int64) (bool, error) {
	if time.Now().Unix() > expiryTimestamp {
		return false, apperrors.New(apperrors.CodeTokenExpired, "signature expired")
	}

	combinedData := fmt.Sprintf("%s|%d", string(data), expiryTimestamp)

	h := hmac.New(sha512.New512_256, key)
	h.Write([]byte(combinedData))
	expected := base64.StdEncoding.EncodeToString(h.Sum(nil))

	return hmac.Equal([]byte(expected), []byte(signature)), nil
}

type ReplayProtection struct {
	cache       map[string]time.Time
	mu          sync.RWMutex
	maxSize     int
	tolerance   time.Duration
	cleanupTick time.Duration
}

func NewReplayProtection(maxSize int, tolerance time.Duration) *ReplayProtection {
	rp := &ReplayProtection{
		cache:       make(map[string]time.Time),
		maxSize:     maxSize,
		tolerance:   tolerance,
		cleanupTick: time.Minute,
	}

	go rp.startCleanup()
	return rp
}

func (rp *ReplayProtection) startCleanup() {
	ticker := time.NewTicker(rp.cleanupTick)
	defer ticker.Stop()

	for range ticker.C {
		rp.cleanup()
	}
}

func (rp *ReplayProtection) cleanup() {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	cutoff := time.Now().Add(-rp.tolerance * 2)
	var toDelete []string

	for key, timestamp := range rp.cache {
		if timestamp.Before(cutoff) {
			toDelete = append(toDelete, key)
		}
	}

	for _, key := range toDelete {
		delete(rp.cache, key)
	}
}

func (rp *ReplayProtection) Check(nonce string, timestamp int64) (bool, error) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	key := fmt.Sprintf("%s:%d", nonce, timestamp)

	if _, exists := rp.cache[key]; exists {
		return false, apperrors.New(apperrors.CodeSecurityRisk, "replay detected")
	}

	now := time.Now()
	requestTime := time.Unix(timestamp, 0)
	if math.Abs(float64(now.Sub(requestTime))) > float64(rp.tolerance) {
		return false, apperrors.New(apperrors.CodeTokenExpired, "timestamp out of tolerance")
	}

	rp.cache[key] = now

	if len(rp.cache) > rp.maxSize {
		rp.cleanup()
	}

	return true, nil
}

func (rp *ReplayProtection) Clear() {
	rp.mu.Lock()
	defer rp.mu.Unlock()
	rp.cache = make(map[string]time.Time)
}

func (rp *ReplayProtection) Size() int {
	rp.mu.RLock()
	defer rp.mu.RUnlock()
	return len(rp.cache)
}

type SignatureValidator struct {
	signature   *EnhancedSignature
	strictMode  bool
	allowExpired bool
}

func NewSignatureValidator(config ...SignatureConfig) *SignatureValidator {
	sig := NewEnhancedSignature(config...)
	return &SignatureValidator{
		signature:   sig,
		strictMode:  false,
		allowExpired: false,
	}
}

func (v *SignatureValidator) SetStrictMode(enabled bool) *SignatureValidator {
	v.strictMode = enabled
	return v
}

func (v *SignatureValidator) SetAllowExpired(allowed bool) *SignatureValidator {
	v.allowExpired = allowed
	return v
}

func (v *SignatureValidator) Validate(req SignatureRequest, signature string) error {
	if v.strictMode {
		if req.Timestamp == 0 {
			return apperrors.New(apperrors.CodeMissingParams, "strict mode: timestamp required")
		}
		if req.Nonce == "" && v.signature.config.EnableNonce {
			return apperrors.New(apperrors.CodeMissingParams, "strict mode: nonce required")
		}
	}

	if !v.allowExpired {
		return v.signature.VerifySignature(req, signature)
	}

	if err := v.signature.verifyTimestamp(req.Timestamp); err != nil {
		if v.strictMode {
			return err
		}
	}

	return v.signature.VerifySignature(req, signature)
}

type SignatureError struct {
	Code    string
	Message string
	Details map[string]interface{}
}

func (e *SignatureError) Error() string {
	return fmt.Sprintf("[%s] %s", e.Code, e.Message)
}

func NewSignatureError(code, message string) *SignatureError {
	return &SignatureError{
		Code:    code,
		Message: message,
		Details: make(map[string]interface{}),
	}
}

func (e *SignatureError) WithDetail(key string, value interface{}) *SignatureError {
	e.Details[key] = value
	return e
}

const (
	ErrCodeMissingSignature = "MISSING_SIGNATURE"
	ErrCodeInvalidSignature = "INVALID_SIGNATURE"
	ErrCodeExpiredTimestamp = "EXPIRED_TIMESTAMP"
	ErrCodeReplayDetected   = "REPLAY_DETECTED"
	ErrCodeInvalidNonce     = "INVALID_NONCE"
	ErrCodeMissingNonce     = "MISSING_NONCE"
	ErrCodeMissingTimestamp = "MISSING_TIMESTAMP"
	ErrCodeInvalidAlgorithm = "INVALID_ALGORITHM"
)

func CreateSignatureError(code, message string, details map[string]interface{}) error {
	return &SignatureError{
		Code:    code,
		Message: message,
		Details: details,
	}
}

func CreateMissingSignatureError() error {
	return CreateSignatureError(ErrCodeMissingSignature, "signature header is missing", nil)
}

func CreateInvalidSignatureError() error {
	return CreateSignatureError(ErrCodeInvalidSignature, "signature verification failed", nil)
}

func CreateExpiredTimestampError(timestamp int64) error {
	return CreateSignatureError(ErrCodeExpiredTimestamp, "timestamp has expired", map[string]interface{}{
		"timestamp": timestamp,
	})
}

func CreateReplayDetectedError(nonce string) error {
	return CreateSignatureError(ErrCodeReplayDetected, "replay attack detected", map[string]interface{}{
		"nonce": nonce,
	})
}

func CreateMissingNonceError() error {
	return CreateSignatureError(ErrCodeMissingNonce, "nonce header is missing", nil)
}

func CreateMissingTimestampError() error {
	return CreateSignatureError(ErrCodeMissingTimestamp, "timestamp header is missing", nil)
}
