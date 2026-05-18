package config

import (
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"
)

type EncryptedConfig struct {
	EncryptedData string    `json:"encrypted_data"`
	IV            string    `json:"iv"`
	Timestamp     time.Time `json:"timestamp"`
	Version       int       `json:"version"`
}

type ConfigEncryption struct {
	key    []byte
	mu     sync.RWMutex
	enabled bool
}

var (
	encryptionInstance *ConfigEncryption
	encryptionOnce     sync.Once
)

func GetConfigEncryption() *ConfigEncryption {
	encryptionOnce.Do(func() {
		instance := &ConfigEncryption{
			enabled: GetEnvManager().GetBool("CONFIG_ENCRYPTION_ENABLED", false),
		}

		keyStr := GetEnvManager().Get("CONFIG_ENCRYPTION_KEY", "")
		if keyStr != "" {
			instance.key = []byte(keyStr)
		} else if instance.enabled {
			instance.key = generateRandomKey(32)
		}

		encryptionInstance = instance
	})

	return encryptionInstance
}

func (e *ConfigEncryption) Enable(key string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if len(key) != 16 && len(key) != 24 && len(key) != 32 {
		return errors.New("encryption key must be 16, 24, or 32 bytes")
	}

	e.key = []byte(key)
	e.enabled = true
	return nil
}

func (e *ConfigEncryption) Disable() {
	e.mu.Lock()
	defer e.mu.Unlock()
	e.enabled = false
	e.key = nil
}

func (e *ConfigEncryption) IsEnabled() bool {
	e.mu.RLock()
	defer e.mu.RUnlock()
	return e.enabled
}

func (e *ConfigEncryption) Encrypt(data interface{}) (*EncryptedConfig, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled || e.key == nil {
		return nil, errors.New("encryption is not enabled")
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal data: %w", err)
	}

	block, err := aes.NewCipher(e.key)
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

	ciphertext := gcm.Seal(nonce, nonce, jsonData, nil)

	return &EncryptedConfig{
		EncryptedData: base64.StdEncoding.EncodeToString(ciphertext),
		IV:            base64.StdEncoding.EncodeToString(nonce),
		Timestamp:     time.Now(),
		Version:       1,
	}, nil
}

func (e *ConfigEncryption) Decrypt(encrypted *EncryptedConfig, target interface{}) error {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.enabled || e.key == nil {
		return errors.New("decryption is not enabled")
	}

	ciphertext, err := base64.StdEncoding.DecodeString(encrypted.EncryptedData)
	if err != nil {
		return fmt.Errorf("failed to decode ciphertext: %w", err)
	}

	block, err := aes.NewCipher(e.key)
	if err != nil {
		return fmt.Errorf("failed to create cipher: %w", err)
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return fmt.Errorf("failed to create GCM: %w", err)
	}

	nonce, err := base64.StdEncoding.DecodeString(encrypted.IV)
	if err != nil {
		return fmt.Errorf("failed to decode nonce: %w", err)
	}

	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return fmt.Errorf("failed to decrypt: %w", err)
	}

	if err := json.Unmarshal(plaintext, target); err != nil {
		return fmt.Errorf("failed to unmarshal data: %w", err)
	}

	return nil
}

func (e *ConfigEncryption) EncryptFile(path string, data interface{}) error {
	encrypted, err := e.Encrypt(data)
	if err != nil {
		return err
	}

	file, err := os.OpenFile(path, os.O_CREATE|os.O_WRONLY|os.O_TRUNC, 0600)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	encoder := json.NewEncoder(file)
	encoder.SetIndent("", "  ")
	return encoder.Encode(encrypted)
}

func (e *ConfigEncryption) DecryptFile(path string, target interface{}) error {
	file, err := os.Open(path)
	if err != nil {
		return fmt.Errorf("failed to open file: %w", err)
	}
	defer file.Close()

	var encrypted EncryptedConfig
	if err := json.NewDecoder(file).Decode(&encrypted); err != nil {
		return fmt.Errorf("failed to decode encrypted config: %w", err)
	}

	return e.Decrypt(&encrypted, target)
}

func generateRandomKey(length int) []byte {
	key := make([]byte, length)
	rand.Read(key)
	return key
}

type ConfigBackup struct {
	mu          sync.RWMutex
	backupDir   string
	maxBackups  int
	enabled     bool
	compression bool
}

var (
	backupInstance *ConfigBackup
	backupOnce     sync.Once
)

func GetConfigBackup() *ConfigBackup {
	backupOnce.Do(func() {
		env := GetEnvManager()
		backupDir := env.Get("CONFIG_BACKUP_DIR", "./config_backups")

		backupInstance = &ConfigBackup{
			backupDir:   backupDir,
			maxBackups:  env.GetInt("CONFIG_BACKUP_MAX_COUNT", 10),
			enabled:     env.GetBool("CONFIG_BACKUP_ENABLED", true),
			compression: env.GetBool("CONFIG_BACKUP_COMPRESSION", true),
		}

		os.MkdirAll(backupDir, 0755)
	})

	return backupInstance
}

func (b *ConfigBackup) Backup(configPath string) error {
	b.mu.Lock()
	defer b.mu.Unlock()

	if !b.enabled {
		return nil
	}

	data, err := os.ReadFile(configPath)
	if err != nil {
		return fmt.Errorf("failed to read config: %w", err)
	}

	filename := fmt.Sprintf("config_%s.backup", time.Now().Format("20060102150405"))
	if b.compression {
		filename += ".gz"
	}

	backupPath := filepath.Join(b.backupDir, filename)

	if b.compression {
		compressed, err := compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress: %w", err)
		}
		data = compressed
	}

	if err := os.WriteFile(backupPath, data, 0600); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}

	return b.cleanOldBackups(filepath.Base(configPath))
}

func (b *ConfigBackup) cleanOldBackups(configName string) error {
	pattern := filepath.Join(b.backupDir, fmt.Sprintf("%s_*.backup*", configName))
	matches, err := filepath.Glob(pattern)
	if err != nil {
		return err
	}

	if len(matches) <= b.maxBackups {
		return nil
	}

	var backupFiles []struct {
		path    string
	ModTime  time.Time
	}
	for _, match := range matches {
		info, err := os.Stat(match)
		if err != nil {
			continue
		}
		backupFiles = append(backupFiles, struct {
			path    string
			ModTime time.Time
		}{match, info.ModTime()})
	}

	for i := 0; i < len(backupFiles)-b.maxBackups; i++ {
		for j := i + 1; j < len(backupFiles); j++ {
			if backupFiles[j].ModTime.Before(backupFiles[i].ModTime) {
				backupFiles[i], backupFiles[j] = backupFiles[j], backupFiles[i]
			}
		}
	}

	for i := 0; i < len(backupFiles)-b.maxBackups; i++ {
		os.Remove(backupFiles[i].path)
	}

	return nil
}

func (b *ConfigBackup) ListBackups(configName string) ([]string, error) {
	b.mu.RLock()
	defer b.mu.RUnlock()

	pattern := filepath.Join(b.backupDir, fmt.Sprintf("%s_*.backup*", configName))
	return filepath.Glob(pattern)
}

func (b *ConfigBackup) Restore(backupPath string, targetPath string) error {
	b.mu.RLock()
	defer b.mu.RUnlock()

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return fmt.Errorf("failed to read backup: %w", err)
	}

	if b.compression && strings.HasSuffix(backupPath, ".gz") {
		data, err = decompressData(data)
		if err != nil {
			return fmt.Errorf("failed to decompress: %w", err)
		}
	}

	return os.WriteFile(targetPath, data, 0644)
}

func compressData(data []byte) ([]byte, error) {
	return data, nil
}

func decompressData(data []byte) ([]byte, error) {
	return data, nil
}
