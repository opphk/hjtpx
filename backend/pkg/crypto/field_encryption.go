package crypto

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

var (
	ErrFieldNotFound     = errors.New("field not found in data")
	ErrKeyNotFound       = errors.New("key not found")
	ErrKeyRotationFailed = errors.New("key rotation failed")
)

type FieldEncryption struct {
	FieldName  string `json:"field_name"`
	Ciphertext string `json:"ciphertext"`
	IV         string `json:"iv"`
	Algorithm  string `json:"algorithm"`
	Version    int    `json:"version"`
}

type KeyInfo struct {
	KeyID        string    `json:"key_id"`
	Key          []byte    `json:"key"`
	Version      int       `json:"version"`
	CreatedAt    time.Time `json:"created_at"`
	RotatedAt    time.Time `json:"rotated_at"`
	Status       string    `json:"status"`
	Algorithm    string    `json:"algorithm"`
	Rotations    int       `json:"rotations"`
}

type FieldLevelEncryption struct {
	masterKey       []byte
	keyVersions     map[int]*KeyInfo
	currentVersion  int
	mu              sync.RWMutex
	keyStoragePath  string
	auditEnabled    bool
}

type EncryptionAudit struct {
	Operation     string    `json:"operation"`
	FieldName     string    `json:"field_name"`
	KeyID         string    `json:"key_id"`
	Timestamp     time.Time `json:"timestamp"`
	Success       bool      `json:"success"`
	ErrorMessage  string    `json:"error_message,omitempty"`
	IPAddress     string    `json:"ip_address,omitempty"`
	UserID        uint      `json:"user_id,omitempty"`
}

var (
	fieldEncryptionInstance *FieldLevelEncryption
	fieldEncryptionOnce    sync.Once
)

func NewFieldLevelEncryption(masterKey []byte, storagePath string) (*FieldLevelEncryption, error) {
	fe := &FieldLevelEncryption{
		masterKey:      deriveKey(masterKey),
		keyVersions:    make(map[int]*KeyInfo),
		currentVersion: 1,
		keyStoragePath: storagePath,
		auditEnabled:   true,
	}

	if err := fe.initializeKey(); err != nil {
		return nil, err
	}

	return fe, nil
}

func GetFieldLevelEncryption() *FieldLevelEncryption {
	return fieldEncryptionInstance
}

func InitFieldLevelEncryption(masterKey []byte, storagePath string) error {
	var err error
	fieldEncryptionOnce.Do(func() {
		fieldEncryptionInstance, err = NewFieldLevelEncryption(masterKey, storagePath)
	})
	return err
}

func (f *FieldLevelEncryption) initializeKey() error {
	keyInfo := &KeyInfo{
		KeyID:     generateKeyID(),
		Key:       f.masterKey,
		Version:   f.currentVersion,
		CreatedAt: time.Now(),
		RotatedAt: time.Now(),
		Status:    "active",
		Algorithm: "AES-256-GCM",
		Rotations: 0,
	}

	f.keyVersions[f.currentVersion] = keyInfo

	if f.keyStoragePath != "" {
		if err := f.saveKeysToStorage(); err != nil {
			return fmt.Errorf("failed to save keys to storage: %w", err)
		}
	}

	return nil
}

func (f *FieldLevelEncryption) EncryptField(data map[string]interface{}, fieldName string) (*FieldEncryption, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	value, exists := data[fieldName]
	if !exists {
		return nil, ErrFieldNotFound
	}

	valueStr, ok := value.(string)
	if !ok {
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal field value: %w", err)
		}
		valueStr = string(valueBytes)
	}

	keyInfo := f.keyVersions[f.currentVersion]
	ciphertext, err := f.encryptWithKey([]byte(valueStr), keyInfo.Key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	fieldEncryption := &FieldEncryption{
		FieldName:  fieldName,
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		IV:         base64.StdEncoding.EncodeToString([]byte{}),
		Algorithm:  "AES-256-GCM",
		Version:    f.currentVersion,
	}

	f.logEncryptionAudit("encrypt", fieldName, keyInfo.KeyID, true, "")

	return fieldEncryption, nil
}

func (f *FieldLevelEncryption) EncryptFields(data map[string]interface{}, fieldNames []string) (map[string]*FieldEncryption, error) {
	results := make(map[string]*FieldEncryption)

	for _, fieldName := range fieldNames {
		encryption, err := f.EncryptField(data, fieldName)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt field %s: %w", fieldName, err)
		}
		results[fieldName] = encryption
	}

	return results, nil
}

func (f *FieldLevelEncryption) DecryptField(encryptedData *FieldEncryption) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keyInfo, exists := f.keyVersions[encryptedData.Version]
	if !exists {
		return "", ErrKeyNotFound
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encryptedData.Ciphertext)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	plaintext, err := f.decryptWithKey(ciphertext, keyInfo.Key)
	if err != nil {
		f.logEncryptionAudit("decrypt", encryptedData.FieldName, keyInfo.KeyID, false, err.Error())
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	f.logEncryptionAudit("decrypt", encryptedData.FieldName, keyInfo.KeyID, true, "")

	return string(plaintext), nil
}

func (f *FieldLevelEncryption) DecryptFields(encryptedData map[string]*FieldEncryption) (map[string]string, error) {
	results := make(map[string]string)

	for fieldName, enc := range encryptedData {
		plaintext, err := f.DecryptField(enc)
		if err != nil {
			return nil, fmt.Errorf("failed to decrypt field %s: %w", fieldName, err)
		}
		results[fieldName] = plaintext
	}

	return results, nil
}

func (f *FieldLevelEncryption) encryptWithKey(plaintext, key []byte) ([]byte, error) {
	block, err := aes.NewCipher(key)
	if err != nil {
		return nil, fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := rand.Read(nonce); err != nil {
		return nil, fmt.Errorf("failed to generate nonce: %w", err)
	}

	ciphertext := gcm.Seal(nonce, nonce, plaintext, nil)
	return ciphertext, nil
}

func (f *FieldLevelEncryption) decryptWithKey(ciphertext, key []byte) ([]byte, error) {
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
		return nil, fmt.Errorf("failed to decrypt: %w", err)
	}

	return plaintext, nil
}

func (f *FieldLevelEncryption) RotateKey() error {
	f.mu.Lock()
	defer f.mu.Unlock()

	f.currentVersion++

	newKey := make([]byte, len(f.masterKey))
	copy(newKey, f.masterKey)
	for i := range newKey {
		newKey[i] ^= byte(time.Now().UnixNano() & 0xFF)
	}

	keyInfo := &KeyInfo{
		KeyID:     generateKeyID(),
		Key:       newKey,
		Version:   f.currentVersion,
		CreatedAt: time.Now(),
		RotatedAt: time.Now(),
		Status:    "active",
		Algorithm: "AES-256-GCM",
		Rotations: f.keyVersions[f.currentVersion-1].Rotations + 1,
	}

	f.keyVersions[f.currentVersion] = keyInfo
	f.masterKey = newKey

	if f.keyVersions[f.currentVersion-1] != nil {
		f.keyVersions[f.currentVersion-1].Status = "retired"
	}

	if f.keyStoragePath != "" {
		if err := f.saveKeysToStorage(); err != nil {
			return fmt.Errorf("%w: %v", ErrKeyRotationFailed, err)
		}
	}

	return nil
}

func (f *FieldLevelEncryption) ReEncryptWithNewKey(data *FieldEncryption) (*FieldEncryption, error) {
	plaintext, err := f.DecryptField(data)
	if err != nil {
		return nil, err
	}

	f.mu.RLock()
	newKeyInfo := f.keyVersions[f.currentVersion]
	f.mu.RUnlock()

	ciphertext, err := f.encryptWithKey([]byte(plaintext), newKeyInfo.Key)
	if err != nil {
		return nil, fmt.Errorf("failed to re-encrypt: %w", err)
	}

	return &FieldEncryption{
		FieldName:  data.FieldName,
		Ciphertext: base64.StdEncoding.EncodeToString(ciphertext),
		IV:         base64.StdEncoding.EncodeToString([]byte{}),
		Algorithm:  "AES-256-GCM",
		Version:    f.currentVersion,
	}, nil
}

func (f *FieldLevelEncryption) saveKeysToStorage() error {
	if f.keyStoragePath == "" {
		return nil
	}

	if err := os.MkdirAll(f.keyStoragePath, 0600); err != nil {
		return fmt.Errorf("failed to create key storage directory: %w", err)
	}

	keyFile := filepath.Join(f.keyStoragePath, "key_metadata.json")
	data, err := json.MarshalIndent(f.keyVersions, "", "  ")
	if err != nil {
		return fmt.Errorf("failed to marshal key metadata: %w", err)
	}

	if err := os.WriteFile(keyFile, data, 0600); err != nil {
		return fmt.Errorf("failed to write key metadata: %w", err)
	}

	return nil
}

func (f *FieldLevelEncryption) LoadKeysFromStorage() error {
	if f.keyStoragePath == "" {
		return nil
	}

	keyFile := filepath.Join(f.keyStoragePath, "key_metadata.json")
	data, err := os.ReadFile(keyFile)
	if err != nil {
		if os.IsNotExist(err) {
			return nil
		}
		return fmt.Errorf("failed to read key metadata: %w", err)
	}

	var keyVersions map[int]*KeyInfo
	if err := json.Unmarshal(data, &keyVersions); err != nil {
		return fmt.Errorf("failed to unmarshal key metadata: %w", err)
	}

	f.mu.Lock()
	f.keyVersions = keyVersions
	for version := range keyVersions {
		if version > f.currentVersion {
			f.currentVersion = version
		}
	}
	f.mu.Unlock()

	return nil
}

func (f *FieldLevelEncryption) logEncryptionAudit(operation, fieldName, keyID string, success bool, errorMsg string) {
	if !f.auditEnabled {
		return
	}

	audit := EncryptionAudit{
		Operation:    operation,
		FieldName:    fieldName,
		KeyID:        keyID,
		Timestamp:    time.Now(),
		Success:      success,
		ErrorMessage: errorMsg,
	}

	auditJSON, _ := json.Marshal(audit)
	fmt.Printf("[EncryptionAudit] %s\n", string(auditJSON))
}

func (f *FieldLevelEncryption) GetKeyVersion() int {
	f.mu.RLock()
	defer f.mu.RUnlock()
	return f.currentVersion
}

func (f *FieldLevelEncryption) GetKeyInfo(version int) (*KeyInfo, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keyInfo, exists := f.keyVersions[version]
	if !exists {
		return nil, ErrKeyNotFound
	}

	return keyInfo, nil
}

func (f *FieldLevelEncryption) ListKeyVersions() []int {
	f.mu.RLock()
	defer f.mu.RUnlock()

	versions := make([]int, 0, len(f.keyVersions))
	for v := range f.keyVersions {
		versions = append(versions, v)
	}

	return versions
}

func (f *FieldLevelEncryption) SetMasterKey(key []byte) {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.masterKey = deriveKey(key)
}

func deriveKey(input []byte) []byte {
	hash := sha256.Sum256(input)
	return hash[:]
}

func generateKeyID() string {
	timestamp := time.Now().UnixNano()
	randomBytes := make([]byte, 8)
	rand.Read(randomBytes)
	return fmt.Sprintf("%d-%x", timestamp, randomBytes)
}

func (f *FieldLevelEncryption) BatchEncrypt(data []map[string]interface{}, fields []string) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(data))

	for _, record := range data {
		encryptedRecord := make(map[string]interface{})

		for key, value := range record {
			shouldEncrypt := false
			for _, field := range fields {
				if key == field {
					shouldEncrypt = true
					break
				}
			}

			if shouldEncrypt {
				encryption, err := f.EncryptField(record, key)
				if err != nil {
					return nil, err
				}
				encryptedRecord[key] = encryption
			} else {
				encryptedRecord[key] = value
			}
		}

		results = append(results, encryptedRecord)
	}

	return results, nil
}

func (f *FieldLevelEncryption) BatchDecrypt(data []map[string]interface{}) ([]map[string]interface{}, error) {
	results := make([]map[string]interface{}, 0, len(data))

	for _, record := range data {
		decryptedRecord := make(map[string]interface{})

		for key, value := range record {
			if enc, ok := value.(*FieldEncryption); ok {
				decrypted, err := f.DecryptField(enc)
				if err != nil {
					return nil, err
				}
				decryptedRecord[key] = decrypted
			} else {
				decryptedRecord[key] = value
			}
		}

		results = append(results, decryptedRecord)
	}

	return results, nil
}

type EncryptedFieldValue struct {
	Value    string `json:"value"`
	KeyID    string `json:"key_id"`
	Version  int    `json:"version"`
	Algo     string `json:"algorithm"`
}

func (f *FieldLevelEncryption) EncryptFieldToStruct(data map[string]interface{}, fieldName string) (*EncryptedFieldValue, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	value, exists := data[fieldName]
	if !exists {
		return nil, ErrFieldNotFound
	}

	valueStr, ok := value.(string)
	if !ok {
		valueBytes, err := json.Marshal(value)
		if err != nil {
			return nil, fmt.Errorf("failed to marshal field value: %w", err)
		}
		valueStr = string(valueBytes)
	}

	keyInfo := f.keyVersions[f.currentVersion]
	ciphertext, err := f.encryptWithKey([]byte(valueStr), keyInfo.Key)
	if err != nil {
		return nil, fmt.Errorf("%w: %v", ErrEncryptionFailed, err)
	}

	return &EncryptedFieldValue{
		Value:   base64.StdEncoding.EncodeToString(ciphertext),
		KeyID:   keyInfo.KeyID,
		Version: f.currentVersion,
		Algo:    "AES-256-GCM",
	}, nil
}

func (f *FieldLevelEncryption) DecryptFieldFromStruct(encrypted *EncryptedFieldValue) (string, error) {
	f.mu.RLock()
	defer f.mu.RUnlock()

	keyInfo, exists := f.keyVersions[encrypted.Version]
	if !exists {
		return "", ErrKeyNotFound
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.Value)
	if err != nil {
		return "", fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	plaintext, err := f.decryptWithKey(ciphertext, keyInfo.Key)
	if err != nil {
		return "", fmt.Errorf("%w: %v", ErrDecryptionFailed, err)
	}

	return string(plaintext), nil
}

func (f *FieldLevelEncryption) ValidateKeyStrength(key []byte) error {
	if len(key) < 16 {
		return errors.New("key length must be at least 16 bytes")
	}

	if len(key) < 32 {
		fmt.Printf("[Security Warning] Key length is less than 32 bytes, consider using a stronger key\n")
	}

	uniqueBytes := make(map[byte]bool)
	for _, b := range key {
		uniqueBytes[b] = true
	}

	if len(uniqueBytes) < 10 {
		return errors.New("key lacks sufficient entropy")
	}

	return nil
}

func (f *FieldLevelEncryption) GenerateSecureFieldKey() ([]byte, error) {
	key := make([]byte, 32)
	if _, err := rand.Read(key); err != nil {
		return nil, fmt.Errorf("failed to generate secure key: %w", err)
	}

	return key, nil
}

func (f *FieldLevelEncryption) GetEncryptionStats() map[string]interface{} {
	f.mu.RLock()
	defer f.mu.RUnlock()

	stats := map[string]interface{}{
		"current_version": f.currentVersion,
		"total_versions": len(f.keyVersions),
		"active_keys":    0,
		"retired_keys":   0,
	}

	for _, keyInfo := range f.keyVersions {
		if keyInfo.Status == "active" {
			stats["active_keys"] = stats["active_keys"].(int) + 1
		} else {
			stats["retired_keys"] = stats["retired_keys"].(int) + 1
		}
	}

	return stats
}

func (f *FieldLevelEncryption) DisableAudit() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auditEnabled = false
}

func (f *FieldLevelEncryption) EnableAudit() {
	f.mu.Lock()
	defer f.mu.Unlock()
	f.auditEnabled = true
}
