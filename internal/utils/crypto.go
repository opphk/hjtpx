package utils

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/md5"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"fmt"
	"io"

	"golang.org/x/crypto/pbkdf2"
)

const (
	KeySize   = 32
	SaltSize  = 32
	NonceSize = 12
	Iterations = 100000
)

type CryptoUtil struct {
	masterKey []byte
}

func NewCryptoUtil(masterKey string) *CryptoUtil {
	hash := sha256.Sum256([]byte(masterKey))
	return &CryptoUtil{
		masterKey: hash[:],
	}
}

func MD5(data string) string {
	hash := md5.Sum([]byte(data))
	return hex.EncodeToString(hash[:])
}

func MD5Bytes(data []byte) string {
	hash := md5.Sum(data)
	return hex.EncodeToString(hash[:])
}

func SHA256(data string) string {
	hash := sha256.Sum256([]byte(data))
	return hex.EncodeToString(hash[:])
}

func SHA256Bytes(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func Base64Encode(data []byte) string {
	return base64.StdEncoding.EncodeToString(data)
}

func Base64Decode(data string) ([]byte, error) {
	return base64.StdEncoding.DecodeString(data)
}

func Base64URLEncode(data []byte) string {
	return base64.URLEncoding.EncodeToString(data)
}

func Base64URLDecode(data string) ([]byte, error) {
	return base64.URLEncoding.DecodeString(data)
}

func (c *CryptoUtil) Encrypt(plaintext []byte) ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("failed to generate salt: %w", err)
	}

	key := pbkdf2.Key(c.masterKey, salt, Iterations, KeySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, NonceSize)
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nil, nonce, plaintext, nil)

	ciphertext = append(salt, nonce...)
	ciphertext = append(ciphertext, gcm.Seal(nil, nonce, plaintext, nil)...)

	return ciphertext, nil
}

func (c *CryptoUtil) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < SaltSize+NonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	salt := ciphertext[:SaltSize]
	nonce := ciphertext[SaltSize : SaltSize+NonceSize]
	actualCiphertext := ciphertext[SaltSize+NonceSize:]

	key := pbkdf2.Key(c.masterKey, salt, Iterations, KeySize, sha256.New)

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, actualCiphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (c *CryptoUtil) EncryptString(plaintext string) (string, error) {
	ciphertext, err := c.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return Base64Encode(ciphertext), nil
}

func (c *CryptoUtil) DecryptString(ciphertext string) (string, error) {
	data, err := Base64Decode(ciphertext)
	if err != nil {
		return "", err
	}
	plaintext, err := c.Decrypt(data)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func GenerateRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

func GenerateRandomString(n int) (string, error) {
	const letters = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	bytes := make([]byte, n)
	if _, err := rand.Read(bytes); err != nil {
		return "", fmt.Errorf("failed to generate random string: %w", err)
	}
	for i, b := range bytes {
		bytes[i] = letters[int(b)%len(letters)]
	}
	return string(bytes), nil
}

func GenerateNonce() (string, error) {
	bytes, err := GenerateRandomBytes(32)
	if err != nil {
		return "", err
	}
	return Base64URLEncode(bytes), nil
}

func GenerateSecureToken() string {
	bytes, _ := GenerateRandomBytes(32)
	return hex.EncodeToString(bytes)
}

func HMACSHA256(message, key []byte) []byte {
	h := sha256.New()
	h.Write(key)
	h.Write(message)
	return h.Sum(nil)
}

func HMACSHA256String(message, key string) string {
	return hex.EncodeToString(HMACSHA256([]byte(message), []byte(key)))
}

func SHA256HMAC(message, secret string) string {
	key := []byte(secret)
	h := sha256.New()
	h.Write(key)
	h.Write([]byte(message))
	return hex.EncodeToString(h.Sum(nil))
}
