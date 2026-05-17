package middleware

import (
	"bytes"
	"context"
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"hjtpx/internal/config"
	"hjtpx/internal/database"
	"hjtpx/internal/utils"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
)

type SignatureConfig struct {
	SecretKey          string
	TimestampTolerance time.Duration
	NonceExpiration    time.Duration
	Enabled            bool
}

type SignatureVerifier struct {
	config     SignatureConfig
	redisClient *redis.Client
}

type SignatureRequest struct {
	Signature string `json:"signature"`
	Nonce     string `json:"nonce"`
	Timestamp int64  `json:"timestamp"`
	AppID     string `json:"app_id"`
	Fingerprint string `json:"fingerprint"`
}

type SignatureResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
}

func NewSignatureVerifier(cfg config.SignatureConfig, redisClient *redis.Client) *SignatureVerifier {
	return &SignatureVerifier{
		config: SignatureConfig{
			SecretKey:          cfg.SecretKey,
			TimestampTolerance: time.Duration(cfg.TimestampToleranceSeconds) * time.Second,
			NonceExpiration:    time.Duration(cfg.NonceExpirationSeconds) * time.Second,
			Enabled:            cfg.Enabled,
		},
		redisClient: redisClient,
	}
}

func SignatureVerification(sv *SignatureVerifier) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !sv.config.Enabled {
			c.Next()
			return
		}

		if sv.shouldSkipSignature(c.Request.URL.Path) {
			c.Next()
			return
		}

		signatureReq, err := sv.extractSignatureHeaders(c)
		if err != nil {
			utils.Warn("Signature verification failed - missing headers: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, SignatureResponse{
				Code:    401,
				Message: "Missing signature headers",
			})
			return
		}

		if err := sv.validateTimestamp(signatureReq.Timestamp); err != nil {
			utils.Warn("Signature verification failed - timestamp: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, SignatureResponse{
				Code:    401,
				Message: "Request timestamp expired or invalid",
			})
			return
		}

		if err := sv.validateNonce(signatureReq.Nonce); err != nil {
			utils.Warn("Signature verification failed - nonce: %v", err)
			c.AbortWithStatusJSON(http.StatusUnauthorized, SignatureResponse{
				Code:    401,
				Message: "Invalid or replayed nonce",
			})
			return
		}

		body, err := io.ReadAll(c.Request.Body)
		if err != nil {
			utils.Error("Failed to read request body: %v", err)
			c.AbortWithStatusJSON(http.StatusBadRequest, SignatureResponse{
				Code:    400,
				Message: "Failed to read request body",
			})
			return
		}
		c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

		expectedSignature := sv.generateSignature(c.Request.Method, c.Request.URL.Path, string(body), signatureReq.Timestamp, signatureReq.AppID)

		if !sv.verifySignature(signatureReq.Signature, expectedSignature) {
			utils.Warn("Signature verification failed - signature mismatch for path: %s", c.Request.URL.Path)
			c.AbortWithStatusJSON(http.StatusUnauthorized, SignatureResponse{
				Code:    401,
				Message: "Invalid signature",
			})
			return
		}

		if err := sv.markNonceUsed(signatureReq.Nonce); err != nil {
			utils.Error("Failed to mark nonce as used: %v", err)
		}

		c.Set("app_id", signatureReq.AppID)
		c.Set("fingerprint", signatureReq.Fingerprint)

		c.Next()
	}
}

func (sv *SignatureVerifier) extractSignatureHeaders(c *gin.Context) (*SignatureRequest, error) {
	signature := c.GetHeader("X-Signature")
	nonce := c.GetHeader("X-Nonce")
	timestampStr := c.GetHeader("X-Timestamp")
	appID := c.GetHeader("X-App-ID")
	fingerprint := c.GetHeader("X-Fingerprint")

	if signature == "" || nonce == "" || timestampStr == "" {
		return nil, fmt.Errorf("missing required headers")
	}

	timestamp, err := strconv.ParseInt(timestampStr, 10, 64)
	if err != nil {
		return nil, fmt.Errorf("invalid timestamp format")
	}

	return &SignatureRequest{
		Signature:   signature,
		Nonce:       nonce,
		Timestamp:   timestamp,
		AppID:       appID,
		Fingerprint: fingerprint,
	}, nil
}

func (sv *SignatureVerifier) validateTimestamp(timestamp int64) error {
	requestTime := time.Unix(timestamp, 0)
	now := time.Now()

	if now.Sub(requestTime) > sv.config.TimestampTolerance {
		return fmt.Errorf("timestamp too old")
	}

	if requestTime.Sub(now) > sv.config.TimestampTolerance {
		return fmt.Errorf("timestamp too far in future")
	}

	return nil
}

func (sv *SignatureVerifier) validateNonce(nonce string) error {
	ctx := context.Background()
	key := fmt.Sprintf("nonce:%s", nonce)

	exists, err := sv.redisClient.Exists(ctx, key).Result()
	if err != nil {
		return fmt.Errorf("redis error: %w", err)
	}

	if exists > 0 {
		return fmt.Errorf("nonce already used")
	}

	return nil
}

func (sv *SignatureVerifier) markNonceUsed(nonce string) error {
	ctx := context.Background()
	key := fmt.Sprintf("nonce:%s", nonce)

	return sv.redisClient.Set(ctx, key, "1", sv.config.NonceExpiration).Err()
}

func (sv *SignatureVerifier) generateSignature(method, path, body string, timestamp int64, appID string) string {
	message := fmt.Sprintf("%s:%s:%s:%d:%s", method, path, body, timestamp, sv.config.SecretKey)
	if appID != "" {
		message = fmt.Sprintf("%s:%s:%s:%d:%s:%s", method, path, body, timestamp, sv.config.SecretKey, appID)
	}

	h := hmac.New(sha256.New, []byte(sv.config.SecretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func (sv *SignatureVerifier) verifySignature(signature, expectedSignature string) bool {
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func (sv *SignatureVerifier) shouldSkipSignature(path string) bool {
	skipPaths := []string{
		"/",
		"/demo",
		"/admin",
		"/health",
		"/static/",
		"/api/v1/admin/create",
		"/api/v1/admin/login",
		"/api/v1/user/login",
		"/api/v1/user/register",
	}

	for _, skipPath := range skipPaths {
		if strings.HasPrefix(path, skipPath) {
			return true
		}
	}
	return false
}

func GenerateSignature(secretKey, method, path, body string, timestamp int64, appID string) string {
	var message string
	if appID != "" {
		message = fmt.Sprintf("%s:%s:%s:%d:%s:%s", method, path, body, timestamp, secretKey, appID)
	} else {
		message = fmt.Sprintf("%s:%s:%s:%d:%s", method, path, body, timestamp, secretKey)
	}

	h := hmac.New(sha256.New, []byte(secretKey))
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}

func GenerateNonce() (string, error) {
	return utils.GenerateNonce()
}

func ValidateSignatureBody(body []byte, signature, method, path, secretKey string, timestamp int64, appID string) bool {
	expectedSignature := GenerateSignature(secretKey, method, path, string(body), timestamp, appID)
	return hmac.Equal([]byte(signature), []byte(expectedSignature))
}

func InitSignatureMiddleware(cfg config.SignatureConfig, redisClient *redis.Client) *SignatureVerifier {
	return NewSignatureVerifier(cfg, redisClient)
}

func CreateSignatureResponse(success bool, message string, data interface{}) SignatureResponse {
	code := 0
	if !success {
		code = 1
	}
	return SignatureResponse{
		Code:    code,
		Message: message,
		Data:    data,
	}
}

func (sv *SignatureVerifier) GetConfig() SignatureConfig {
	return sv.config
}

func CreateSignatureData(method, path string, body interface{}, appID, secretKey string) (map[string]interface{}, error) {
	var bodyStr string
	switch v := body.(type) {
	case string:
		bodyStr = v
	case []byte:
		bodyStr = string(v)
	default:
		bodyBytes, err := json.Marshal(body)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal body: %w", err)
		}
		bodyStr = string(bodyBytes)
	}

	timestamp := time.Now().Unix()
	nonce, err := GenerateNonce()
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	signature := GenerateSignature(secretKey, method, path, bodyStr, timestamp, appID)

	return map[string]interface{}{
		"signature":  signature,
		"nonce":      nonce,
		"timestamp":  timestamp,
		"app_id":     appID,
	}, nil
}

var _ = database.GetRedisCache
