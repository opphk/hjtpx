package security

import (
	"crypto/hmac"
	"crypto/rand"
	"crypto/sha256"
	"crypto/subtle"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"math/big"
	"net/url"
	"sort"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type SignerConfig struct {
	SecretKey           []byte
	TimestampTolerance  time.Duration
	SignatureHeader     string
	TimestampHeader     string
	NonceHeader         string
	Algorithm           string
	AppIDHeader         string
	SignaturePrefix     string
}

var defaultSignerConfig = &SignerConfig{
	TimestampTolerance: 5 * time.Minute,
	SignatureHeader:     "X-Signature",
	TimestampHeader:     "X-Timestamp",
	NonceHeader:         "X-Nonce",
	Algorithm:           "HMAC-SHA256",
	AppIDHeader:         "X-App-ID",
	SignaturePrefix:     "sha256=",
}

type SignatureParams struct {
	Method      string
	Path        string
	QueryParams url.Values
	Headers     map[string]string
	Body        []byte
	Timestamp   int64
	Nonce       string
	AppID       string
}

type SignatureValidator struct {
	config      *SignerConfig
	nonceCache  map[string]time.Time
	nonceMaxAge time.Duration
}

func NewSigner(secretKey []byte) *Signer {
	return &Signer{
		config:    defaultSignerConfig,
		secretKey: secretKey,
	}
}

func NewSignerWithConfig(config *SignerConfig) *Signer {
	cfg := *config
	if cfg.TimestampTolerance == 0 {
		cfg.TimestampTolerance = defaultSignerConfig.TimestampTolerance
	}
	if cfg.SignatureHeader == "" {
		cfg.SignatureHeader = defaultSignerConfig.SignatureHeader
	}
	if cfg.TimestampHeader == "" {
		cfg.TimestampHeader = defaultSignerConfig.TimestampHeader
	}
	if cfg.NonceHeader == "" {
		cfg.NonceHeader = defaultSignerConfig.NonceHeader
	}
	if cfg.Algorithm == "" {
		cfg.Algorithm = defaultSignerConfig.Algorithm
	}
	return &Signer{
		config:    &cfg,
		secretKey: cfg.SecretKey,
	}
}

type Signer struct {
	config    *SignerConfig
	secretKey []byte
}

func (s *Signer) GenerateKey(length int) ([]byte, error) {
	key := make([]byte, length)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("signer: failed to generate random key: %w", err)
	}
	return key, nil
}

func (s *Signer) SignString(data string) (string, error) {
	return s.Sign([]byte(data))
}

func (s *Signer) Sign(data []byte) (string, error) {
	if len(s.secretKey) == 0 {
		return "", errors.New("signer: secret key is empty")
	}

	switch s.config.Algorithm {
	case "HMAC-SHA256":
		return s.signHMACSHA256(data)
	case "HMAC-SHA512":
		return s.signHMACSHA512(data)
	default:
		return s.signHMACSHA256(data)
	}
}

func (s *Signer) signHMACSHA256(data []byte) (string, error) {
	h := hmac.New(sha256.New, s.secretKey)
	h.Write(data)
	signature := h.Sum(nil)
	return hex.EncodeToString(signature), nil
}

func (s *Signer) signHMACSHA512(data []byte) (string, error) {
	h := hmac.New(sha256.New, s.secretKey)
	h.Write(data)
	return hex.EncodeToString(h.Sum(nil)), nil
}

func (s *Signer) Verify(signature string, data []byte) (bool, error) {
	expectedSig, err := s.Sign(data)
	if err != nil {
		return false, err
	}

	sigToCompare := signature
	if strings.HasPrefix(sigToCompare, s.config.SignaturePrefix) {
		sigToCompare = strings.TrimPrefix(sigToCompare, s.config.SignaturePrefix)
	}

	return subtle.ConstantTimeCompare([]byte(sigToCompare), []byte(expectedSig)) == 1, nil
}

func (s *Signer) SignRequest(params *SignatureParams) (string, string, error) {
	if params.Timestamp == 0 {
		params.Timestamp = time.Now().Unix()
	}
	if params.Nonce == "" {
		nonce, err := GenerateNonce(16)
		if err != nil {
			return "", "", fmt.Errorf("signer: failed to generate nonce: %w", err)
		}
		params.Nonce = nonce
	}

	signatureData, err := s.buildSignatureData(params)
	if err != nil {
		return "", "", fmt.Errorf("signer: failed to build signature data: %w", err)
	}

	signature, err := s.Sign([]byte(signatureData))
	if err != nil {
		return "", "", fmt.Errorf("signer: failed to sign: %w", err)
	}

	return signature, params.Nonce, nil
}

func (s *Signer) buildSignatureData(params *SignatureParams) (string, error) {
	var parts []string

	parts = append(parts, strings.ToUpper(params.Method))
	parts = append(parts, params.Path)

	if params.QueryParams != nil {
		sortedKeys := make([]string, 0, len(params.QueryParams))
		for k := range params.QueryParams {
			sortedKeys = append(sortedKeys, k)
		}
		sort.Strings(sortedKeys)

		var queryParts []string
		for _, k := range sortedKeys {
			values := params.QueryParams[k]
			for _, v := range values {
				queryParts = append(queryParts, fmt.Sprintf("%s=%s", url.QueryEscape(k), url.QueryEscape(v)))
			}
		}
		parts = append(parts, strings.Join(queryParts, "&"))
	} else {
		parts = append(parts, "")
	}

	if params.AppID != "" {
		parts = append(parts, params.AppID)
	}

	parts = append(parts, strconv.FormatInt(params.Timestamp, 10))

	if params.Nonce != "" {
		parts = append(parts, params.Nonce)
	}

	if len(params.Body) > 0 {
		bodyHash := sha256.Sum256(params.Body)
		parts = append(parts, hex.EncodeToString(bodyHash[:]))
	}

	return strings.Join(parts, "\n"), nil
}

func (s *Signer) VerifyRequest(params *SignatureParams) (bool, error) {
	if params.Timestamp == 0 {
		headerTS := params.Headers[s.config.TimestampHeader]
		if headerTS == "" {
			headerTS = params.Headers["X-Timestamp"]
		}
		ts, err := strconv.ParseInt(headerTS, 10, 64)
		if err != nil {
			return false, errors.New("signer: invalid timestamp")
		}
		params.Timestamp = ts
	}

	now := time.Now().Unix()
	toleranceSeconds := int64(s.config.TimestampTolerance.Seconds())
	if params.Timestamp < now-toleranceSeconds || params.Timestamp > now+toleranceSeconds {
		return false, fmt.Errorf("signer: timestamp expired or too far in future (tolerance: %v)", s.config.TimestampTolerance)
	}

	if params.Nonce == "" {
		params.Nonce = params.Headers[s.config.NonceHeader]
		if params.Nonce == "" {
			params.Nonce = params.Headers["X-Nonce"]
		}
	}
	if params.Nonce != "" {
		if len(params.Nonce) < 8 || len(params.Nonce) > 128 {
			return false, errors.New("signer: invalid nonce length")
		}
	}

	return true, nil
}

func GenerateNonce(length int) (string, error) {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	nonce := make([]byte, length)
	charsetLen := big.NewInt(int64(len(charset)))

	for i := 0; i < length; i++ {
		randNum, err := rand.Int(rand.Reader, charsetLen)
		if err != nil {
			return "", fmt.Errorf("signer: failed to generate random number: %w", err)
		}
		nonce[i] = charset[randNum.Int64()]
	}

	return string(nonce), nil
}

func GenerateAppSecret() ([]byte, error) {
	return NewSigner(nil).GenerateKey(32)
}

func NewSignatureValidator(config *SignerConfig) *SignatureValidator {
	return &SignatureValidator{
		config:      config,
		nonceCache:  make(map[string]time.Time),
		nonceMaxAge: 10 * time.Minute,
	}
}

func (v *SignatureValidator) ValidateSignature(c *gin.Context, secretKey []byte) (bool, error) {
	signature := c.GetHeader(v.config.SignatureHeader)
	if signature == "" {
		signature = c.GetHeader("X-Signature")
	}
	if signature == "" {
		return false, errors.New("signature: missing signature header")
	}

	timestamp := c.GetHeader(v.config.TimestampHeader)
	if timestamp == "" {
		timestamp = c.GetHeader("X-Timestamp")
	}
	if timestamp == "" {
		return false, errors.New("signature: missing timestamp header")
	}

	ts, err := strconv.ParseInt(timestamp, 10, 64)
	if err != nil {
		return false, errors.New("signature: invalid timestamp format")
	}

	now := time.Now().Unix()
	toleranceSeconds := int64(v.config.TimestampTolerance.Seconds())
	if ts < now-toleranceSeconds || ts > now+toleranceSeconds {
		return false, fmt.Errorf("signature: timestamp expired (tolerance: %v)", v.config.TimestampTolerance)
	}

	nonce := c.GetHeader(v.config.NonceHeader)
	if nonce == "" {
		nonce = c.GetHeader("X-Nonce")
	}
	if nonce != "" {
		if v.isNonceUsed(nonce) {
			return false, errors.New("signature: nonce already used (replay attack detected)")
		}
		v.markNonceUsed(nonce)
	}

	var body []byte
	if c.Request.Body != nil {
		body, err = io.ReadAll(c.Request.Body)
		if err != nil {
			return false, fmt.Errorf("signature: failed to read body: %w", err)
		}
	}

	params := &SignatureParams{
		Method:    c.Request.Method,
		Path:      c.Request.URL.Path,
		Body:      body,
		Timestamp: ts,
		Nonce:     nonce,
	}
	params.QueryParams = c.Request.URL.Query()

	signer := NewSignerWithConfig(v.config)
	signer.secretKey = secretKey

	signatureData, err := signer.buildSignatureData(params)
	if err != nil {
		return false, fmt.Errorf("signature: failed to build signature data: %w", err)
	}

	valid, err := signer.Verify(signature, []byte(signatureData))
	if err != nil {
		return false, fmt.Errorf("signature: verification failed: %w", err)
	}

	return valid, nil
}

func (v *SignatureValidator) isNonceUsed(nonce string) bool {
	if _, exists := v.nonceCache[nonce]; exists {
		return true
	}
	return false
}

func (v *SignatureValidator) markNonceUsed(nonce string) {
	v.nonceCache[nonce] = time.Now()
	v.cleanupOldNonces()
}

func (v *SignatureValidator) cleanupOldNonces() {
	now := time.Now()
	for nonce, timestamp := range v.nonceCache {
		if now.Sub(timestamp) > v.nonceMaxAge {
			delete(v.nonceCache, nonce)
		}
	}
}

func SignatureMiddleware(secretKey []byte) gin.HandlerFunc {
	config := *defaultSignerConfig
	config.SecretKey = secretKey
	validator := NewSignatureValidator(&config)

	return func(c *gin.Context) {
		signature := c.GetHeader(config.SignatureHeader)
		timestamp := c.GetHeader(config.TimestampHeader)

		if signature != "" && timestamp != "" {
			valid, err := validator.ValidateSignature(c, secretKey)
			if err != nil {
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "unauthorized",
					"message": err.Error(),
				})
				return
			}
			if !valid {
				c.AbortWithStatusJSON(401, gin.H{
					"error":   "unauthorized",
					"message": "invalid signature",
				})
				return
			}
		}

		c.Next()
	}
}

func SignerMiddleware() gin.HandlerFunc {
	secretKey, err := NewSigner(nil).GenerateKey(32)
	if err != nil {
		panic("failed to generate secret key: " + err.Error())
	}

	return func(c *gin.Context) {
		signature := c.GetHeader("X-Signature")
		timestamp := c.GetHeader("X-Timestamp")

		if signature == "" || timestamp == "" {
			c.Next()
			return
		}

		ts, err := strconv.ParseInt(timestamp, 10, 64)
		if err != nil {
			c.AbortWithStatusJSON(401, gin.H{
				"error": "invalid timestamp",
			})
			return
		}

		now := time.Now().Unix()
		toleranceSeconds := int64(5 * time.Minute / time.Second)
		if ts < now-toleranceSeconds || ts > now+toleranceSeconds {
			c.AbortWithStatusJSON(401, gin.H{
				"error": "timestamp expired",
			})
			return
		}

		var body []byte
		if c.Request.Body != nil {
			body, _ = io.ReadAll(c.Request.Body)
		}

		signer := NewSigner(secretKey)
		signatureData := fmt.Sprintf("%s:%s:%s:%d", c.Request.Method, c.Request.URL.Path, string(body), ts)
		expectedSig, _ := signer.SignString(signatureData)

		if signature != expectedSig {
			c.AbortWithStatusJSON(401, gin.H{
				"error": "invalid signature",
			})
			return
		}

		c.Next()
	}
}

type SignedRequest struct {
	AppID     string            `json:"app_id"`
	Timestamp int64             `json:"timestamp"`
	Nonce     string            `json:"nonce"`
	Signature string            `json:"signature"`
	Method    string            `json:"method"`
	Path      string            `json:"path"`
	Query     map[string]string `json:"query,omitempty"`
	Body      json.RawMessage   `json:"body,omitempty"`
}

func ParseSignedRequest(c *gin.Context) (*SignedRequest, error) {
	var req SignedRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		return nil, fmt.Errorf("failed to parse signed request: %w", err)
	}
	return &req, nil
}
