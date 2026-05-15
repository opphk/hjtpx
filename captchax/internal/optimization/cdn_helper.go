package optimization

import (
	"crypto/hmac"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"time"
)

type CDNHelper struct {
	cdnBaseURL string
	signingKey string
}

func NewCDNHelper(cdnBaseURL, signingKey string) *CDNHelper {
	return &CDNHelper{
		cdnBaseURL: cdnBaseURL,
		signingKey: signingKey,
	}
}

func (c *CDNHelper) GenerateSignedURL(path string, expiresIn time.Duration) string {
	deadline := time.Now().Add(expiresIn).Unix()
	signature := generateSignature(path, deadline, c.signingKey)

	return fmt.Sprintf("%s/%s?expires=%d&signature=%s",
		c.cdnBaseURL, path, deadline, signature)
}

func generateSignature(path string, deadline int64, key string) string {
	data := fmt.Sprintf("%s:%d", path, deadline)
	h := hmac.New(sha256.New, []byte(key))
	h.Write([]byte(data))
	return hex.EncodeToString(h.Sum(nil))
}

func (c *CDNHelper) ValidateSignature(path string, deadline int64, signature string) bool {
	expected := generateSignature(path, deadline, c.signingKey)
	return hmac.Equal([]byte(expected), []byte(signature))
}
