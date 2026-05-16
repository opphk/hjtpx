package crypto

import (
	"crypto"
	"crypto/aes"
	"crypto/cipher"
	"crypto/ecdsa"
	"crypto/elliptic"
	"crypto/hmac"
	"crypto/rand"
	"crypto/rsa"
	"crypto/sha1"
	"crypto/sha256"
	"crypto/sha512"
	"crypto/x509"
	"encoding/base64"
	"encoding/hex"
	"encoding/pem"
	"errors"
	"fmt"
	"hash"
	"io"
	mrand "math/rand"
	"math/big"
	"strings"
	"sync"
	"time"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

var (
	ErrInvalidKeyLength   = errors.New("invalid key length for AES")
	ErrCiphertextTooShort = errors.New("ciphertext too short")
	ErrInvalidCiphertext  = errors.New("invalid ciphertext")
	ErrPublicKeyInvalid   = errors.New("invalid public key")
	ErrPrivateKeyInvalid  = errors.New("invalid private key")
	ErrEncryptionFailed   = errors.New("encryption failed")
	ErrDecryptionFailed  = errors.New("decryption failed")
	ErrSignatureFailed    = errors.New("signature generation failed")
	ErrVerificationFailed = errors.New("signature verification failed")
	ErrInvalidPassword   = errors.New("invalid password")
)

type AESKeySize int

const (
	KeySize128 AESKeySize = 16
	KeySize192 AESKeySize = 24
	KeySize256 AESKeySize = 32
)

type HashAlgorithmType string

const (
	AlgoSHA256 HashAlgorithmType = "sha256"
	AlgoSHA512 HashAlgorithmType = "sha512"
	AlgoSHA1   HashAlgorithmType = "sha1"
)

type CryptoConfig struct {
	AESKeySize       AESKeySize
	HashAlgorithm    HashAlgorithmType
	PBKDF2Iterations int
	SaltLength       int
}

var defaultCryptoConfig = CryptoConfig{
	AESKeySize:        KeySize256,
	HashAlgorithm:     AlgoSHA256,
	PBKDF2Iterations: 100000,
	SaltLength:        32,
}

var (
	randMutex  sync.Mutex
	randSource mrand.Source
)

func init() {
	seed := time.Now().UnixNano()
	randSource = mrand.NewSource(seed)
}

func secureRandomBytes(n int) ([]byte, error) {
	b := make([]byte, n)
	randMutex.Lock()
	defer randMutex.Unlock()
	_, err := rand.Read(b)
	if err != nil {
		return nil, fmt.Errorf("failed to generate random bytes: %w", err)
	}
	return b, nil
}

func GenerateRandomKey(keySize AESKeySize) ([]byte, error) {
	return secureRandomBytes(int(keySize))
}

func GenerateRandomString(length int) (string, error) {
	const chars = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789"
	bytes, err := secureRandomBytes(length)
	if err != nil {
		return "", err
	}
	for i := range bytes {
		bytes[i] = chars[int(bytes[i])%len(chars)]
	}
	return string(bytes), nil
}

func GenerateSalt(length int) ([]byte, error) {
	if length <= 0 {
		length = defaultCryptoConfig.SaltLength
	}
	return secureRandomBytes(length)
}

func AESEncrypt(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := secureRandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func AESEncryptWithAAD(plaintext, key, additionalData []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := secureRandomBytes(gcm.NonceSize())
	if err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, additionalData)
	return ciphertext, nil
}

func AESDecryptWithAAD(ciphertext, key, additionalData []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	if len(ciphertext) == 0 {
		return nil, ErrCiphertextTooShort
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, additionalData)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func AESDecrypt(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	if len(ciphertext) == 0 {
		return nil, ErrCiphertextTooShort
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonceSize := gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, ErrCiphertextTooShort
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func AESEncryptString(plaintext string, key []byte) (string, error) {
	ciphertext, err := AESEncrypt([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func AESDecryptString(ciphertextBase64 string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	plaintext, err := AESDecrypt(ciphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func AESEncryptWithCBC(plaintext []byte, key []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	blockSize := block.BlockSize()
	plaintext = pkcs7Pad(plaintext, blockSize)

	ciphertext := make([]byte, len(plaintext))
	iv, err := secureRandomBytes(blockSize)
	if err != nil {
		return nil, fmt.Errorf("failed to generate IV: %w", err)
	}

	mode := cipher.NewCBCEncrypter(block, iv)
	mode.CryptBlocks(ciphertext, plaintext)

	return append(iv, ciphertext...), nil
}

func AESDecryptWithCBC(ciphertext []byte, key []byte) ([]byte, error) {
	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return nil, ErrInvalidKeyLength
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	blockSize := block.BlockSize()
	if len(ciphertext) < blockSize {
		return nil, ErrCiphertextTooShort
	}

	iv := ciphertext[:blockSize]
	ciphertext = ciphertext[blockSize:]

	if len(ciphertext)%blockSize != 0 {
		return nil, ErrInvalidCiphertext
	}

	plaintext := make([]byte, len(ciphertext))
	mode := cipher.NewCBCDecrypter(block, iv)
	mode.CryptBlocks(plaintext, ciphertext)

	return pkcs7Unpad(plaintext, blockSize)
}

func pkcs7Pad(data []byte, blockSize int) []byte {
	padding := blockSize - len(data)%blockSize
	padBytes := make([]byte, padding)
	for i := range padBytes {
		padBytes[i] = byte(padding)
	}
	return append(data, padBytes...)
}

func pkcs7Unpad(data []byte, blockSize int) ([]byte, error) {
	if len(data) == 0 {
		return nil, errors.New("data is empty")
	}
	if len(data)%blockSize != 0 {
		return nil, errors.New("data length is not a multiple of block size")
	}
	padding := int(data[len(data)-1])
	if padding == 0 || padding > blockSize {
		return nil, errors.New("invalid padding")
	}
	for i := len(data) - padding; i < len(data); i++ {
		if data[i] != byte(padding) {
			return nil, errors.New("invalid padding bytes")
		}
	}
	return data[:len(data)-padding], nil
}

func HashSHA256(data []byte) string {
	h := sha256.Sum256(data)
	return hex.EncodeToString(h[:])
}

func HashSHA512(data []byte) string {
	h := sha512.Sum512(data)
	return hex.EncodeToString(h[:])
}

func HashSHA1(data []byte) string {
	h := sha1.Sum(data)
	return hex.EncodeToString(h[:])
}

func HashBytes(data []byte, algorithm HashAlgorithmType) string {
	switch algorithm {
	case AlgoSHA256:
		return HashSHA256(data)
	case AlgoSHA512:
		return HashSHA512(data)
	case AlgoSHA1:
		return HashSHA1(data)
	default:
		return HashSHA256(data)
	}
}

func HashString(data string, algorithm HashAlgorithmType) string {
	return HashBytes([]byte(data), algorithm)
}

func ComputeHMAC(key, data []byte, algorithm HashAlgorithmType) ([]byte, error) {
	var h func() hash.Hash
	switch algorithm {
	case AlgoSHA256:
		h = sha256.New
	case AlgoSHA512:
		h = sha512.New
	default:
		h = sha256.New
	}

	mac := hmac.New(h, key)
	mac.Write(data)
	return mac.Sum(nil), nil
}

func HMACString(key []byte, data []byte, algorithm HashAlgorithmType) string {
	result, err := ComputeHMAC(key, data, algorithm)
	if err != nil {
		return ""
	}
	return hex.EncodeToString(result)
}

func VerifyHMAC(key, data, expectedMAC []byte, algorithm HashAlgorithmType) bool {
	mac, err := ComputeHMAC(key, data, algorithm)
	if err != nil {
		return false
	}
	return hmac.Equal(mac, expectedMAC)
}

func PBKDF2Hash(password, salt []byte, iterations int, keyLength int) ([]byte, error) {
	if iterations <= 0 {
		iterations = defaultCryptoConfig.PBKDF2Iterations
	}
	if keyLength <= 0 {
		keyLength = 32
	}
	return pbkdf2.Key(password, salt, iterations, keyLength, sha256.New), nil
}

func PBKDF2HashString(password, salt string, iterations int, keyLength int) (string, error) {
	saltBytes := []byte(salt)
	if salt == "" {
		var err error
		saltBytes, err = GenerateSalt(16)
		if err != nil {
			return "", err
		}
	}
	hash, err := PBKDF2Hash([]byte(password), saltBytes, iterations, keyLength)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(hash), nil
}

func HashPassword(password string) (string, error) {
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

func HashPasswordWithCost(password string, cost int) (string, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		cost = bcrypt.DefaultCost
	}
	bytes, err := bcrypt.GenerateFromPassword([]byte(password), cost)
	if err != nil {
		return "", fmt.Errorf("failed to hash password: %w", err)
	}
	return string(bytes), nil
}

func VerifyPassword(password, hash string) bool {
	err := bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	return err == nil
}

func VerifyPasswordWithCost(password, hash string, expectedCost int) bool {
	if !VerifyPassword(password, hash) {
		return false
	}
	cost, err := bcrypt.Cost([]byte(hash))
	if err != nil {
		return false
	}
	return cost == expectedCost
}

func GetBcryptCost(hashStr string) (int, error) {
	return bcrypt.Cost([]byte(hashStr))
}

func GenerateRSAKeyPair(bits int) (*rsa.PrivateKey, *rsa.PublicKey, error) {
	if bits < 2048 {
		bits = 2048
	}
	if bits > 4096 {
		bits = 4096
	}

	privateKey, err := rsa.GenerateKey(rand.Reader, bits)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate RSA key: %w", err)
	}

	return privateKey, &privateKey.PublicKey, nil
}

func RSAEncrypt(plaintext []byte, publicKey *rsa.PublicKey) ([]byte, error) {
	if publicKey == nil {
		return nil, ErrPublicKeyInvalid
	}

	hash := sha256.New()
	ciphertext, err := rsa.EncryptOAEP(hash, rand.Reader, publicKey, plaintext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	return ciphertext, nil
}

func RSADecrypt(ciphertext []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, ErrPrivateKeyInvalid
	}

	hash := sha256.New()
	plaintext, err := rsa.DecryptOAEP(hash, rand.Reader, privateKey, ciphertext, nil)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func RSAEncryptString(plaintext string, publicKey *rsa.PublicKey) (string, error) {
	ciphertext, err := RSAEncrypt([]byte(plaintext), publicKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func RSADecryptString(ciphertextBase64 string, privateKey *rsa.PrivateKey) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode base64: %w", err)
	}
	plaintext, err := RSADecrypt(ciphertext, privateKey)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func RSASign(message []byte, privateKey *rsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, ErrPrivateKeyInvalid
	}

	hash := sha256.New()
	hash.Write(message)
	hashed := hash.Sum(nil)

	signature, err := rsa.SignPKCS1v15(rand.Reader, privateKey, crypto.SHA256, hashed)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSignatureFailed, err)
	}

	return signature, nil
}

func RSAVerify(message, signature []byte, publicKey *rsa.PublicKey) error {
	if publicKey == nil {
		return ErrPublicKeyInvalid
	}

	hash := sha256.New()
	hash.Write(message)
	hashed := hash.Sum(nil)

	err := rsa.VerifyPKCS1v15(publicKey, crypto.SHA256, hashed, signature)
	if err != nil {
		return fmt.Errorf("%w: %v", ErrVerificationFailed, err)
	}

	return nil
}

func RSASignString(message string, privateKey *rsa.PrivateKey) (string, error) {
	signature, err := RSASign([]byte(message), privateKey)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(signature), nil
}

func RSAVerifyString(message, signatureBase64 string, publicKey *rsa.PublicKey) bool {
	signature, err := base64.StdEncoding.DecodeString(signatureBase64)
	if err != nil {
		return false
	}
	return RSAVerify([]byte(message), signature, publicKey) == nil
}

func ExportRSAPrivateKeyToPEM(privateKey *rsa.PrivateKey) (string, error) {
	derBytes := x509.MarshalPKCS1PrivateKey(privateKey)
	block := &pem.Block{
		Type:  "RSA PRIVATE KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

func ExportRSAPublicKeyToPEM(publicKey *rsa.PublicKey) (string, error) {
	derBytes, err := x509.MarshalPKIXPublicKey(publicKey)
	if err != nil {
		return "", fmt.Errorf("failed to marshal public key: %w", err)
	}
	block := &pem.Block{
		Type:  "RSA PUBLIC KEY",
		Bytes: derBytes,
	}
	return string(pem.EncodeToMemory(block)), nil
}

func ParseRSAPrivateKeyFromPEM(pemData string) (*rsa.PrivateKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	privateKey, err := x509.ParsePKCS1PrivateKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse private key: %w", err)
	}
	return privateKey, nil
}

func ParseRSAPublicKeyFromPEM(pemData string) (*rsa.PublicKey, error) {
	block, _ := pem.Decode([]byte(pemData))
	if block == nil {
		return nil, errors.New("failed to parse PEM block")
	}
	publicKey, err := x509.ParsePKIXPublicKey(block.Bytes)
	if err != nil {
		return nil, fmt.Errorf("failed to parse public key: %w", err)
	}
	rsaKey, ok := publicKey.(*rsa.PublicKey)
	if !ok {
		return nil, ErrPublicKeyInvalid
	}
	return rsaKey, nil
}

func GenerateECDSAKeyPair() (*ecdsa.PrivateKey, *ecdsa.PublicKey, error) {
	privateKey, err := ecdsa.GenerateKey(elliptic.P256(), rand.Reader)
	if err != nil {
		return nil, nil, fmt.Errorf("failed to generate ECDSA key: %w", err)
	}
	return privateKey, &privateKey.PublicKey, nil
}

func ECDSASign(message []byte, privateKey *ecdsa.PrivateKey) ([]byte, error) {
	if privateKey == nil {
		return nil, ErrPrivateKeyInvalid
	}

	hash := sha256.Sum256(message)
	r, s, err := ecdsa.Sign(rand.Reader, privateKey, hash[:])
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrSignatureFailed, err)
	}

	signature := append(r.Bytes(), s.Bytes()...)
	return signature, nil
}

func ECDSAVerify(message, signature []byte, publicKey *ecdsa.PublicKey) bool {
	if publicKey == nil || len(signature) != 64 {
		return false
	}

	hash := sha256.Sum256(message)

	r := new(big.Int).SetBytes(signature[:32])
	s := new(big.Int).SetBytes(signature[32:])

	return ecdsa.Verify(publicKey, hash[:], r, s)
}

func ConstantTimeCompare(a, b string) bool {
	return hmac.Equal([]byte(a), []byte(b))
}

func ConstantTimeCompareBytes(a, b []byte) bool {
	return hmac.Equal(a, b)
}

func DeriveKey(password, salt []byte, keyLength int, iterations int) ([]byte, error) {
	return PBKDF2Hash(password, salt, iterations, keyLength)
}

func GenerateSecureToken(length int) (string, error) {
	bytes := make([]byte, length)
	_, err := io.ReadFull(rand.Reader, bytes)
	if err != nil {
		return "", fmt.Errorf("failed to generate secure token: %w", err)
	}
	return base64.URLEncoding.EncodeToString(bytes), nil
}

func MaskSensitiveData(data string, visibleStart int, visibleEnd int) string {
	if len(data) <= visibleStart+visibleEnd {
		return strings.Repeat("*", len(data))
	}

	if visibleStart < 0 {
		visibleStart = 0
	}
	if visibleEnd < 0 {
		visibleEnd = 0
	}

	start := data[:visibleStart]
	middle := strings.Repeat("*", len(data)-visibleStart-visibleEnd)
	end := data[len(data)-visibleEnd:]

	return start + middle + end
}

type EncryptionResult struct {
	Ciphertext string `json:"ciphertext"`
	Key        string `json:"key"`
	Algorithm  string `json:"algorithm"`
	Timestamp  int64  `json:"timestamp"`
}

func EncryptSensitive(plaintext string) (*EncryptionResult, error) {
	key, err := GenerateRandomKey(KeySize256)
	if err != nil {
		return nil, err
	}

	ciphertext, err := AESEncryptString(plaintext, key)
	if err != nil {
		return nil, err
	}

	return &EncryptionResult{
		Ciphertext: ciphertext,
		Key:        base64.StdEncoding.EncodeToString(key),
		Algorithm:  "AES-256-GCM",
		Timestamp:  time.Now().Unix(),
	}, nil
}

func DecryptSensitive(ciphertext, keyBase64 string) (string, error) {
	key, err := base64.StdEncoding.DecodeString(keyBase64)
	if err != nil {
		return "", fmt.Errorf("failed to decode key: %w", err)
	}

	return AESDecryptString(ciphertext, key)
}

func GenerateAPIKey(prefix string) (string, error) {
	bytes, err := secureRandomBytes(32)
	if err != nil {
		return "", err
	}
	return prefix + "_" + base64.URLEncoding.EncodeToString(bytes), nil
}

func ValidateAPIKey(apiKey, prefix string) bool {
	if len(apiKey) < len(prefix)+2 {
		return false
	}
	return strings.HasPrefix(apiKey, prefix+"_")
}

type EncryptedData struct {
	Ciphertext string `json:"c"`
	IV         string `json:"i"`
	AuthTag    string `json:"t"`
	Version    int    `json:"v"`
	Algorithm  string `json:"a"`
	CreatedAt  int64  `json:"ct"`
	EncryptedAt int64 `json:"et"`
}

func NewEncryptedData(ciphertext, iv, authTag string) *EncryptedData {
	return &EncryptedData{
		Ciphertext:  ciphertext,
		IV:          iv,
		AuthTag:     authTag,
		Version:     1,
		Algorithm:   "AES-256-GCM",
		CreatedAt:   time.Now().Unix(),
		EncryptedAt: time.Now().Unix(),
	}
}

func (e *EncryptedData) ToJSON() string {
	return fmt.Sprintf(`{"c":"%s","i":"%s","t":"%s","v":%d,"a":"%s","ct":%d,"et":%d}`,
		e.Ciphertext, e.IV, e.AuthTag, e.Version, e.Algorithm, e.CreatedAt, e.EncryptedAt)
}
