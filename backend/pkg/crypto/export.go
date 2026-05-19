package crypto

import (
	"encoding/base64"
	"time"
)

type EncryptionModule struct {
	keyRotationManager *KeyRotationManager
}

func NewEncryptionModule(config *KeyRotationConfig) (*EncryptionModule, error) {
	manager, err := NewKeyRotationManager(config)
	if err != nil {
		return nil, err
	}

	return &EncryptionModule{
		keyRotationManager: manager,
	}, nil
}

func (m *EncryptionModule) Encrypt(plaintext string) (string, error) {
	return m.keyRotationManager.EncryptStringWithCurrentKey(plaintext)
}

func (m *EncryptionModule) Decrypt(ciphertext string, version int) (string, error) {
	return m.keyRotationManager.DecryptStringWithKey(version, ciphertext)
}

func (m *EncryptionModule) RotateKey() (*EncryptionKey, error) {
	return m.keyRotationManager.RotateKey()
}

func (m *EncryptionModule) RotateKeyWithAlgorithm(algorithm KeyAlgorithm) (*EncryptionKey, error) {
	return m.keyRotationManager.RotateKeyWithAlgorithm(algorithm)
}

func (m *EncryptionModule) GetCurrentKey() (*EncryptionKey, error) {
	return m.keyRotationManager.GetCurrentKey()
}

func (m *EncryptionModule) GetKeyByVersion(version int) (*EncryptionKey, error) {
	return m.keyRotationManager.GetKeyByVersion(version)
}

func (m *EncryptionModule) GetStats() KeyRotationStats {
	return m.keyRotationManager.GetStats()
}

func (m *EncryptionModule) StartAutoRotation(interval time.Duration) {
	m.keyRotationManager.SetRotationInterval(interval)
}

func (m *EncryptionModule) StopAutoRotation() {
	m.keyRotationManager.StopAutoRotation()
}

func (m *EncryptionModule) RegisterRotationCallback(callback func(*KeyRotationEvent)) {
	m.keyRotationManager.RegisterRotationCallback(callback)
}

func (m *EncryptionModule) ChaCha20Encrypt(plaintext string, key []byte) (string, error) {
	ciphertext, err := ChaCha20Poly1305Encrypt([]byte(plaintext), key)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(ciphertext), nil
}

func (m *EncryptionModule) ChaCha20Decrypt(ciphertextBase64 string, key []byte) (string, error) {
	ciphertext, err := base64.StdEncoding.DecodeString(ciphertextBase64)
	if err != nil {
		return "", err
	}
	plaintext, err := ChaCha20Poly1305Decrypt(ciphertext, key)
	if err != nil {
		return "", err
	}
	return string(plaintext), nil
}

func (m *EncryptionModule) Argon2Hash(password string) (string, error) {
	return Argon2HashString(password, nil)
}

func (m *EncryptionModule) Blake2bHash(data string) (string, error) {
	return Blake2bHashString(data, nil)
}

func (m *EncryptionModule) GenerateKeyPair() (string, string, error) {
	return GenerateSecureKeyPair()
}

var (
	DefaultEncryptionModule *EncryptionModule
)

func InitDefaultEncryptionModule(storagePath string) error {
	var err error
	DefaultEncryptionModule, err = NewEncryptionModule(&KeyRotationConfig{
		RotationInterval:   24 * time.Hour,
		MaxKeyAge:          7 * 24 * time.Hour,
		AutoRotation:       true,
		BackupBeforeRotate: true,
		StoragePath:        storagePath,
		KeyLength:          32,
		Algorithm:          AlgorithmAES256GCM,
	})
	return err
}