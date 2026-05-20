package database

import (
	"archive/zip"
	"bytes"
	"compress/gzip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"fmt"
	"io"
	"os"
	"path/filepath"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type BackupConfig struct {
	Directory       string
	RetentionDays  int
	CompressEnabled bool
	EncryptEnabled  bool
	EncryptionKey   string
	RemoteEnabled  bool
	RemotePath     string
}

type BackupInfo struct {
	ID         string
	Filename   string
	Path       string
	Size       int64
	Compressed bool
	Encrypted  bool
	CreatedAt  time.Time
	ExpiresAt  time.Time
	Checksum   string
	Status     string
}

type BackupService struct {
	db      *gorm.DB
	config  *BackupConfig
	backups []BackupInfo
}

func NewBackupService(db *gorm.DB, cfg *config.BackupConfig) *BackupService {
	return &BackupService{
		db:     db,
		config: &BackupConfig{
			Directory:       cfg.BackupDir,
			RetentionDays:   cfg.RetentionDays,
			CompressEnabled: cfg.CompressionEnabled,
			EncryptEnabled:  cfg.EncryptionEnabled,
			EncryptionKey:   cfg.EncryptionKey,
			RemoteEnabled:   cfg.RemoteBackupEnabled,
			RemotePath:      cfg.RemoteBackupPath,
		},
		backups: make([]BackupInfo, 0),
	}
}

func (s *BackupService) CreateBackup(ctx context.Context) (*BackupInfo, error) {
	backupID := fmt.Sprintf("backup_%s", time.Now().Format("20060102_150405"))
	filename := fmt.Sprintf("%s.sql", backupID)
	filepath := filepath.Join(s.config.Directory, filename)

	if err := os.MkdirAll(s.config.Directory, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}

	var buf bytes.Buffer
	tables := []string{
		"users", "admins", "applications", "verifications",
		"verification_logs", "behavior_data", "blacklists",
		"ab_tests", "ab_test_variants", "ab_test_events",
	}

	for _, table := range tables {
		if err := s.backupTable(&buf, table); err != nil {
			return nil, fmt.Errorf("failed to backup table %s: %w", table, err)
		}
	}

	data := buf.Bytes()

	if s.config.CompressEnabled {
		var compressed bytes.Buffer
		writer := gzip.NewWriter(&compressed)
		if _, err := writer.Write(data); err != nil {
			return nil, fmt.Errorf("failed to compress backup: %w", err)
		}
		if err := writer.Close(); err != nil {
			return nil, fmt.Errorf("failed to close compressor: %w", err)
		}
		data = compressed.Bytes()
		filename += ".gz"
	}

	if s.config.EncryptEnabled && s.config.EncryptionKey != "" {
		encrypted, err := s.encryptData(data, s.config.EncryptionKey)
		if err != nil {
			return nil, fmt.Errorf("failed to encrypt backup: %w", err)
		}
		data = encrypted
		filename += ".enc"
	}

	if err := os.WriteFile(filepath, data, 0644); err != nil {
		return nil, fmt.Errorf("failed to write backup file: %w", err)
	}

	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	checksum := s.calculateChecksum(data)

	info := &BackupInfo{
		ID:         backupID,
		Filename:   filename,
		Path:       filepath,
		Size:       stat.Size(),
		Compressed: s.config.CompressEnabled,
		Encrypted:  s.config.EncryptEnabled,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, s.config.RetentionDays),
		Checksum:   checksum,
		Status:     "completed",
	}

	s.backups = append(s.backups, *info)

	if err := s.cleanupOldBackups(ctx); err != nil {
		return nil, fmt.Errorf("failed to cleanup old backups: %w", err)
	}

	return info, nil
}

func (s *BackupService) backupTable(buf *bytes.Buffer, tableName string) error {
	var count int64
	if err := s.db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", tableName)).Scan(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	copyStatement := fmt.Sprintf("COPY %s TO STDOUT WITH CSV HEADER;\n", tableName)
	buf.WriteString(copyStatement)

	var rows *gorm.DB
	if err := s.db.Raw(fmt.Sprintf("SELECT * FROM %s", tableName)).Scan(&rows).Error; err != nil {
		return err
	}

	return nil
}

func (s *BackupService) encryptData(data []byte, key string) ([]byte, error) {
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		keyBytes = append(keyBytes, make([]byte, 32-len(keyBytes))...)
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (s *BackupService) decryptData(data []byte, key string) ([]byte, error) {
	keyBytes := []byte(key)
	if len(keyBytes) < 32 {
		keyBytes = append(keyBytes, make([]byte, 32-len(keyBytes))...)
	} else if len(keyBytes) > 32 {
		keyBytes = keyBytes[:32]
	}

	block, err := aes.NewCipher(keyBytes)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func (s *BackupService) calculateChecksum(data []byte) string {
	hash := sha256.Sum256(data)
	return base64.StdEncoding.EncodeToString(hash[:])
}

func (s *BackupService) restoreBackup(ctx context.Context, backupID string) error {
	backup, err := s.getBackupByID(backupID)
	if err != nil {
		return fmt.Errorf("backup not found: %w", err)
	}

	data, err := os.ReadFile(backup.Path)
	if err != nil {
		return fmt.Errorf("failed to read backup file: %w", err)
	}

	if backup.Encrypted {
		data, err = s.decryptData(data, s.config.EncryptionKey)
		if err != nil {
			return fmt.Errorf("failed to decrypt backup: %w", err)
		}
	}

	if backup.Compressed {
		reader, err := gzip.NewReader(bytes.NewReader(data))
		if err != nil {
			return fmt.Errorf("failed to create gzip reader: %w", err)
		}
		defer reader.Close()

		data, err = io.ReadAll(reader)
		if err != nil {
			return fmt.Errorf("failed to decompress backup: %w", err)
		}
	}

	return s.executeRestore(ctx, data)
}

func (s *BackupService) executeRestore(ctx context.Context, data []byte) error {
	statements := strings.Split(string(data), ";")
	for _, stmt := range statements {
		stmt = strings.TrimSpace(stmt)
		if stmt == "" {
			continue
		}
		if err := s.db.Exec(stmt).Error; err != nil {
			return fmt.Errorf("failed to execute statement: %w", err)
		}
	}
	return nil
}

func (s *BackupService) cleanupOldBackups(ctx context.Context) error {
	files, err := os.ReadDir(s.config.Directory)
	if err != nil {
		return fmt.Errorf("failed to read backup directory: %w", err)
	}

	cutoff := time.Now().AddDate(0, 0, -s.config.RetentionDays)

	for _, file := range files {
		if file.IsDir() {
			continue
		}

		info, err := file.Info()
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			filepath := filepath.Join(s.config.Directory, file.Name())
			if err := os.Remove(filepath); err != nil {
				return fmt.Errorf("failed to remove old backup: %w", err)
			}
		}
	}

	return nil
}

func (s *BackupService) getBackupByID(backupID string) (*BackupInfo, error) {
	for _, backup := range s.backups {
		if backup.ID == backupID {
			return &backup, nil
		}
	}
	return nil, fmt.Errorf("backup not found: %s", backupID)
}

func (s *BackupService) ListBackups() []BackupInfo {
	return s.backups
}

func (s *BackupService) GetBackupInfo(backupID string) (*BackupInfo, error) {
	return s.getBackupByID(backupID)
}

func (s *BackupService) DeleteBackup(ctx context.Context, backupID string) error {
	backup, err := s.getBackupByID(backupID)
	if err != nil {
		return err
	}

	if err := os.Remove(backup.Path); err != nil {
		return fmt.Errorf("failed to delete backup file: %w", err)
	}

	for i, b := range s.backups {
		if b.ID == backupID {
			s.backups = append(s.backups[:i], s.backups[i+1:]...)
			break
		}
	}

	return nil
}

func (s *BackupService) CreateIncrementalBackup(ctx context.Context, since time.Time) (*BackupInfo, error) {
	backupID := fmt.Sprintf("incremental_%s", time.Now().Format("20060102_150405"))
	filename := fmt.Sprintf("%s.sql.gz", backupID)
	filepath := filepath.Join(s.config.Directory, filename)

	var buf bytes.Buffer
	tables := []string{"verification_logs", "behavior_data"}

	for _, table := range tables {
		query := fmt.Sprintf("SELECT * FROM %s WHERE created_at > ?", table)
		var rows []map[string]interface{}
		if err := s.db.Raw(query, since).Scan(&rows).Error; err != nil {
			continue
		}
	}

	data := buf.Bytes()

	var compressed bytes.Buffer
	writer := gzip.NewWriter(&compressed)
	if _, err := writer.Write(data); err != nil {
		return nil, fmt.Errorf("failed to compress incremental backup: %w", err)
	}
	if err := writer.Close(); err != nil {
		return nil, fmt.Errorf("failed to close compressor: %w", err)
	}

	if err := os.WriteFile(filepath, compressed.Bytes(), 0644); err != nil {
		return nil, fmt.Errorf("failed to write incremental backup: %w", err)
	}

	stat, err := os.Stat(filepath)
	if err != nil {
		return nil, fmt.Errorf("failed to stat backup file: %w", err)
	}

	info := &BackupInfo{
		ID:         backupID,
		Filename:   filename,
		Path:       filepath,
		Size:       stat.Size(),
		Compressed: true,
		Encrypted:  false,
		CreatedAt:  time.Now(),
		ExpiresAt:  time.Now().AddDate(0, 0, s.config.RetentionDays),
		Status:     "completed",
	}

	return info, nil
}

func (s *BackupService) ExportToZip(ctx context.Context, backupIDs []string) (string, error) {
	zipFilename := fmt.Sprintf("backup_export_%s.zip", time.Now().Format("20060102_150405"))
	zipPath := filepath.Join(s.config.Directory, zipFilename)

	zipFile, err := os.Create(zipPath)
	if err != nil {
		return "", fmt.Errorf("failed to create zip file: %w", err)
	}
	defer zipFile.Close()

	zipWriter := zip.NewWriter(zipFile)
	defer zipWriter.Close()

	for _, backupID := range backupIDs {
		backup, err := s.getBackupByID(backupID)
		if err != nil {
			continue
		}

		data, err := os.ReadFile(backup.Path)
		if err != nil {
			continue
		}

		f, err := zipWriter.Create(backup.Filename)
		if err != nil {
			continue
		}

		if _, err := f.Write(data); err != nil {
			continue
		}
	}

	return zipPath, nil
}
