package security

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/hex"
	"errors"
	"fmt"
	"io"
	"strconv"
	"strings"

	"golang.org/x/crypto/bcrypt"
	"golang.org/x/crypto/pbkdf2"
)

const (
	AESKeySize256      = 32
	AESKeySize192      = 24
	AESKeySize128      = 16
	PBKDF2Iterations   = 100000
	SaltSize           = 32
	NonceSize          = 12
	MinPasswordLength  = 8
	MaxPasswordLength  = 128
)

var (
	ErrInvalidKeySize      = errors.New("encrypt: invalid key size")
	ErrInvalidCiphertext   = errors.New("encrypt: invalid ciphertext")
	ErrInvalidPassword     = errors.New("encrypt: invalid password")
	ErrPasswordTooWeak     = errors.New("encrypt: password too weak")
	ErrInvalidSalt         = errors.New("encrypt: invalid salt")
	ErrInvalidNonce        = errors.New("encrypt: invalid nonce")
	ErrDecryptionFailed    = errors.New("encrypt: decryption failed")
	ErrInvalidHash         = errors.New("encrypt: invalid hash format")
	ErrBcryptCost          = errors.New("encrypt: bcrypt cost must be between 4 and 31")
)

type Encryptor struct {
	key         []byte
	blockCipher cipher.Block
	gcm         cipher.AEAD
	version     int
}

type EncryptConfig struct {
	KeySize     int
	Algorithm   string
	KDFSalt     []byte
	Iterations  int
}

var defaultEncryptConfig = &EncryptConfig{
	KeySize:    AESKeySize256,
	Algorithm:  "AES-GCM",
	Iterations: PBKDF2Iterations,
}

func NewEncryptor(password string) (*Encryptor, error) {
	return NewEncryptorWithConfig(password, defaultEncryptConfig)
}

func NewEncryptorWithConfig(password string, config *EncryptConfig) (*Encryptor, error) {
	if len(password) < MinPasswordLength {
		return nil, fmt.Errorf("%w: minimum length is %d", ErrInvalidPassword, MinPasswordLength)
	}

	if config == nil {
		config = defaultEncryptConfig
	}

	keySize := config.KeySize
	if keySize == 0 {
		keySize = AESKeySize256
	}

	var key []byte
	if len(config.KDFSalt) > 0 {
		key = pbkdf2.Key([]byte(password), config.KDFSalt, config.Iterations, keySize, sha256.New)
	} else {
		salt := make([]byte, SaltSize)
		if _, err := io.ReadFull(rand.Reader, salt); err != nil {
			return nil, fmt.Errorf("encrypt: failed to generate salt: %w", err)
		}
		key = pbkdf2.Key([]byte(password), salt, config.Iterations, keySize, sha256.New)
	}

	return NewEncryptorWithKey(key)
}

func NewEncryptorWithKey(key []byte) (*Encryptor, error) {
	if len(key) != AESKeySize256 && len(key) != AESKeySize192 && len(key) != AESKeySize128 {
		return nil, fmt.Errorf("%w: expected 16, 24, or 32 bytes, got %d", ErrInvalidKeySize, len(key))
	}

	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("encrypt: failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("encrypt: failed to create GCM: %w", err)
	}

	return &Encryptor{
		key:         key,
		blockCipher: block,
		gcm:         gcm,
		version:     1,
	}, nil
}

func GenerateAESKey() ([]byte, error) {
	key := make([]byte, AESKeySize256)
	if _, err := io.ReadFull(rand.Reader, key); err != nil {
		return nil, fmt.Errorf("encrypt: failed to generate key: %w", err)
	}
	return key, nil
}

func (e *Encryptor) Encrypt(plaintext []byte) ([]byte, error) {
	if len(plaintext) == 0 {
		return nil, errors.New("encrypt: plaintext cannot be empty")
	}

	nonce := make([]byte, e.gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, fmt.Errorf("encrypt: failed to generate nonce: %w", err)
	}

	ciphertext := e.gcm.Seal(nonce, nonce, plaintext, []byte("captchax-v1"))

	return ciphertext, nil
}

func (e *Encryptor) EncryptString(plaintext string) (string, error) {
	ciphertext, err := e.Encrypt([]byte(plaintext))
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (e *Encryptor) Decrypt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) == 0 {
		return nil, errors.New("encrypt: ciphertext cannot be empty")
	}

	nonceSize := e.gcm.NonceSize()
	if len(ciphertext) < nonceSize {
		return nil, fmt.Errorf("%w: ciphertext too short", ErrInvalidCiphertext)
	}

	nonce, ciphertext := ciphertext[:nonceSize], ciphertext[nonceSize:]
	plaintext, err := e.gcm.Open(nil, nonce, ciphertext, []byte("captchax-v1"))
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return plaintext, nil
}

func (e *Encryptor) DecryptString(ciphertext string) (string, error) {
	decoded, err := base64.StdEncoding.DecodeString(ciphertext)
	if err != nil {
		return "", fmt.Errorf("encrypt: failed to decode base64: %w", err)
	}
	plaintext, err := e.Decrypt(decoded)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (e *Encryptor) EncryptWithSalt(plaintext, salt []byte) ([]byte, error) {
	if len(salt) < 16 {
		return nil, fmt.Errorf("%w: salt must be at least 16 bytes", ErrInvalidSalt)
	}

	key := pbkdf2.Key(plaintext, salt, PBKDF2Iterations, AESKeySize256, sha256.New)
	tempEncryptor, err := NewEncryptorWithKey(key)
	if err != nil {
		return nil, err
	}

	ciphertext, err := tempEncryptor.Encrypt(plaintext)
	if err != nil {
		return nil, err
	}

	prefix := []byte(fmt.Sprintf("v%d$%s$", e.version, hex.EncodeToString(salt[:16])))
	result := make([]byte, 0, len(prefix)+len(ciphertext))
	result = append(result, prefix...)
	result = append(result, ciphertext...)

	return result, nil
}

func (e *Encryptor) DecryptWithSalt(ciphertext []byte) ([]byte, error) {
	if len(ciphertext) < 20 {
		return nil, ErrInvalidCiphertext
	}

	parts := strings.SplitN(string(ciphertext), "$", 3)
	if len(parts) != 3 {
		return nil, ErrInvalidCiphertext
	}

	versionStr := strings.TrimPrefix(parts[0], "v")
	version, err := strconv.Atoi(versionStr)
	if err != nil {
		return nil, ErrInvalidCiphertext
	}

	saltHex := parts[1]
	salt, err := hex.DecodeString(saltHex)
	if err != nil || len(salt) < 16 {
		return nil, ErrInvalidSalt
	}

	key := pbkdf2.Key(e.key, salt, PBKDF2Iterations, AESKeySize256, sha256.New)
	tempEncryptor, err := NewEncryptorWithKey(key)
	if err != nil {
		return nil, err
	}

	plaintext, err := tempEncryptor.Decrypt([]byte(parts[2]))
	if err != nil {
		return nil, fmt.Errorf("%w: version %d", err, version)
	}

	return plaintext, nil
}

type PasswordHasher struct {
	cost     int
	minCost  int
	maxCost  int
}

var defaultPasswordHasher = &PasswordHasher{
	cost:     bcrypt.DefaultCost,
	minCost:  bcrypt.MinCost,
	maxCost:  bcrypt.MaxCost,
}

func NewPasswordHasher() *PasswordHasher {
	return &PasswordHasher{
		cost:    bcrypt.DefaultCost,
		minCost: bcrypt.MinCost,
		maxCost: bcrypt.MaxCost,
	}
}

func NewPasswordHasherWithCost(cost int) (*PasswordHasher, error) {
	if cost < bcrypt.MinCost || cost > bcrypt.MaxCost {
		return nil, ErrBcryptCost
	}
	return &PasswordHasher{
		cost:    cost,
		minCost: bcrypt.MinCost,
		maxCost: bcrypt.MaxCost,
	}, nil
}

func (p *PasswordHasher) Hash(password string) (string, error) {
	if err := p.ValidatePassword(password); err != nil {
		return "", err
	}

	hash, err := bcrypt.GenerateFromPassword([]byte(password), p.cost)
	if err != nil {
		return "", fmt.Errorf("encrypt: failed to hash password: %w", err)
	}

	return string(hash), nil
}

func (p *PasswordHasher) HashWithSalt(password, salt string) (string, error) {
	if err := p.ValidatePassword(password); err != nil {
		return "", err
	}

	if len(salt) < 16 {
		return "", ErrInvalidSalt
	}

	saltedPassword := password + salt
	hash, err := bcrypt.GenerateFromPassword([]byte(saltedPassword), p.cost)
	if err != nil {
		return "", fmt.Errorf("encrypt: failed to hash password with salt: %w", err)
	}

	return fmt.Sprintf("bcrypt$%s$%s", salt, string(hash)), nil
}

func (p *PasswordHasher) Verify(password, hash string) error {
	if strings.HasPrefix(hash, "bcrypt$") {
		parts := strings.SplitN(hash, "$", 3)
		if len(parts) != 3 {
			return ErrInvalidHash
		}
		salt := parts[1]
		hashBytes := []byte(parts[2])
		saltedPassword := password + salt
		return bcrypt.CompareHashAndPassword(hashBytes, []byte(saltedPassword))
	}

	hashBytes, err := hex.DecodeString(hash)
	if err != nil && hash != "" {
		return bcrypt.CompareHashAndPassword([]byte(hash), []byte(password))
	}

	return bcrypt.CompareHashAndPassword(hashBytes, []byte(password))
}

func (p *PasswordHasher) ValidatePassword(password string) error {
	if len(password) < MinPasswordLength {
		return fmt.Errorf("%w: minimum length is %d characters", ErrInvalidPassword, MinPasswordLength)
	}
	if len(password) > MaxPasswordLength {
		return fmt.Errorf("%w: maximum length is %d characters", ErrInvalidPassword, MaxPasswordLength)
	}
	return nil
}

func (p *PasswordHasher) Cost() int {
	return p.cost
}

func (p *PasswordHasher) SetCost(cost int) error {
	if cost < p.minCost || cost > p.maxCost {
		return ErrBcryptCost
	}
	p.cost = cost
	return nil
}

func HashPassword(password string) (string, error) {
	return defaultPasswordHasher.Hash(password)
}

func VerifyPassword(password, hash string) error {
	return defaultPasswordHasher.Verify(password, hash)
}

type EncryptedConfig struct {
	Encryptor *Encryptor
}

func NewEncryptedConfig(key []byte) (*EncryptedConfig, error) {
	encryptor, err := NewEncryptorWithKey(key)
	if err != nil {
		return nil, err
	}
	return &EncryptedConfig{Encryptor: encryptor}, nil
}

func (ec *EncryptedConfig) EncryptValue(value string) (string, error) {
	return ec.Encryptor.EncryptString(value)
}

func (ec *EncryptedConfig) DecryptValue(encrypted string) (string, error) {
	return ec.Encryptor.DecryptString(encrypted)
}

func (ec *EncryptedConfig) EncryptMap(m map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m {
		encrypted, err := ec.Encryptor.EncryptString(v)
		if err != nil {
			return nil, fmt.Errorf("encrypt: failed to encrypt key %s: %w", k, err)
		}
		result[k] = encrypted
	}
	return result, nil
}

func (ec *EncryptedConfig) DecryptMap(m map[string]string) (map[string]string, error) {
	result := make(map[string]string)
	for k, v := range m {
		decrypted, err := ec.Encryptor.DecryptString(v)
		if err != nil {
			return nil, fmt.Errorf("encrypt: failed to decrypt key %s: %w", k, err)
		}
		result[k] = decrypted
	}
	return result, nil
}

func DeriveKey(password, salt []byte) ([]byte, error) {
	if len(salt) < 16 {
		return nil, ErrInvalidSalt
	}
	return pbkdf2.Key(password, salt, PBKDF2Iterations, AESKeySize256, sha256.New), nil
}

func GenerateSalt() ([]byte, error) {
	salt := make([]byte, SaltSize)
	if _, err := io.ReadFull(rand.Reader, salt); err != nil {
		return nil, fmt.Errorf("encrypt: failed to generate salt: %w", err)
	}
	return salt, nil
}

type SecureString struct {
	data []byte
}

func NewSecureString(s string) *SecureString {
	return &SecureString{data: []byte(s)}
}

func (s *SecureString) String() string {
	return string(s.data)
}

func (s *SecureString) Bytes() []byte {
	return s.data
}

func (s *SecureString) Wipe() {
	for i := range s.data {
		s.data[i] = 0
	}
	s.data = nil
}

func (s *SecureString) Equal(other *SecureString) bool {
	if s == nil || other == nil {
		return s == other
	}
	if len(s.data) != len(other.data) {
		return false
	}
	var diff byte
	for i := 0; i < len(s.data); i++ {
		diff |= s.data[i] ^ other.data[i]
	}
	return diff == 0
}
