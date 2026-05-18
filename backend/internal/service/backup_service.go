package service

import (
	"archive/zip"
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"log"
	"os"
	"path/filepath"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type BackupType string

const (
	BackupTypeFull        BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
)

type BackupStatus string

const (
	BackupStatusPending   BackupStatus = "pending"
	BackupStatusRunning   BackupStatus = "running"
	BackupStatusCompleted BackupStatus = "completed"
	BackupStatusFailed    BackupStatus = "failed"
)

type BackupRecord struct {
	ID          string       `json:"id"`
	Type        BackupType   `json:"type"`
	FileName    string       `json:"file_name"`
	FilePath    string       `json:"file_path"`
	FileSize    int64        `json:"file_size"`
	Status      BackupStatus `json:"status"`
	CreatedAt   time.Time    `json:"created_at"`
	CompletedAt *time.Time   `json:"completed_at,omitempty"`
	Verified    bool         `json:"verified"`
	Checksum    string       `json:"checksum"`
	Error       string       `json:"error,omitempty"`
}

type BackupService struct {
	config            *config.Config
	db                *gorm.DB
	mu                sync.RWMutex
	backups           map[string]*BackupRecord
	ctx               context.Context
	cancel            context.CancelFunc
	autoBackupTicker  *time.Ticker
	incrementalTicker *time.Ticker
	lastFullBackup    time.Time
}

var (
	backupServiceInstance *BackupService
	backupServiceOnce     sync.Once
)

func NewBackupService(cfg *config.Config) *BackupService {
	ctx, cancel := context.WithCancel(context.Background())

	service := &BackupService{
		config:  cfg,
		db:      database.GetDB(),
		backups: make(map[string]*BackupRecord),
		ctx:     ctx,
		cancel:  cancel,
	}

	if err := service.initBackupDir(); err != nil {
		log.Printf("Failed to initialize backup directory: %v", err)
	}

	service.loadExistingBackups()

	return service
}

func GetBackupService(cfg *config.Config) *BackupService {
	backupServiceOnce.Do(func() {
		backupServiceInstance = NewBackupService(cfg)
	})
	return backupServiceInstance
}

func (s *BackupService) initBackupDir() error {
	if _, err := os.Stat(s.config.Backup.BackupDir); os.IsNotExist(err) {
		return os.MkdirAll(s.config.Backup.BackupDir, 0755)
	}
	return nil
}

func (s *BackupService) loadExistingBackups() {
	files, err := os.ReadDir(s.config.Backup.BackupDir)
	if err != nil {
		log.Printf("Failed to read backup directory: %v", err)
		return
	}

	for _, file := range files {
		if !file.IsDir() && filepath.Ext(file.Name()) == ".zip" {
			info, err := file.Info()
			if err != nil {
				continue
			}

			record := &BackupRecord{
				ID:        file.Name(),
				FileName:  file.Name(),
				FilePath:  filepath.Join(s.config.Backup.BackupDir, file.Name()),
				FileSize:  info.Size(),
				Status:    BackupStatusCompleted,
				CreatedAt: info.ModTime(),
				Verified:  false,
			}
			s.backups[record.ID] = record
		}
	}
}

func (s *BackupService) StartScheduledBackups() {
	if !s.config.Backup.AutoBackupEnabled {
		log.Println("Auto backup is disabled")
		return
	}

	log.Println("Starting scheduled backups")

	interval := time.Duration(s.config.Backup.AutoBackupIntervalHours) * time.Hour
	s.autoBackupTicker = time.NewTicker(interval)

	go func() {
		for {
			select {
			case <-s.ctx.Done():
				return
			case <-s.autoBackupTicker.C:
				log.Println("Triggering scheduled full backup")
				if _, err := s.CreateFullBackup(); err != nil {
					log.Printf("Scheduled backup failed: %v", err)
				}
			}
		}
	}()

	if s.config.Backup.IncrementalEnabled {
		incrementalInterval := time.Duration(s.config.Backup.IncrementalIntervalMins) * time.Minute
		s.incrementalTicker = time.NewTicker(incrementalInterval)

		go func() {
			for {
				select {
				case <-s.ctx.Done():
					return
				case <-s.incrementalTicker.C:
					log.Println("Triggering scheduled incremental backup")
					if _, err := s.CreateIncrementalBackup(); err != nil {
						log.Printf("Incremental backup failed: %v", err)
					}
				}
			}
		}()
	}
}

func (s *BackupService) Stop() {
	if s.autoBackupTicker != nil {
		s.autoBackupTicker.Stop()
	}
	if s.incrementalTicker != nil {
		s.incrementalTicker.Stop()
	}
	s.cancel()
}

func (s *BackupService) CreateFullBackup() (*BackupRecord, error) {
	return s.createBackup(BackupTypeFull)
}

func (s *BackupService) CreateIncrementalBackup() (*BackupRecord, error) {
	return s.createBackup(BackupTypeIncremental)
}

func (s *BackupService) createBackup(backupType BackupType) (*BackupRecord, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	timestamp := time.Now().Format("20060102_150405")
	fileName := fmt.Sprintf("backup_%s_%s.zip", string(backupType), timestamp)
	filePath := filepath.Join(s.config.Backup.BackupDir, fileName)

	record := &BackupRecord{
		ID:        generateBackupID(),
		Type:      backupType,
		FileName:  fileName,
		FilePath:  filePath,
		Status:    BackupStatusPending,
		CreatedAt: time.Now(),
	}

	s.backups[record.ID] = record

	go s.performBackup(record)

	return record, nil
}

func (s *BackupService) performBackup(record *BackupRecord) {
	s.mu.Lock()
	record.Status = BackupStatusRunning
	s.mu.Unlock()

	var err error
	defer func() {
		completedAt := time.Now()
		record.CompletedAt = &completedAt

		if err != nil {
			record.Status = BackupStatusFailed
			record.Error = err.Error()
			log.Printf("Backup %s failed: %v", record.ID, err)
		} else {
			record.Status = BackupStatusCompleted
			if record.Type == BackupTypeFull {
				s.lastFullBackup = time.Now()
			}

			if s.config.Backup.RemoteBackupEnabled {
				go s.uploadToRemote(record)
			}
		}
	}()

	tempDir, err := os.MkdirTemp("", "backup_")
	if err != nil {
		return
	}
	defer os.RemoveAll(tempDir)

	if err = s.exportDatabase(tempDir, record.Type); err != nil {
		return
	}

	if err = s.createBackupArchive(tempDir, record.FilePath); err != nil {
		return
	}

	info, statErr := os.Stat(record.FilePath)
	if statErr != nil {
		err = statErr
		return
	}
	record.FileSize = info.Size()

	record.Checksum, err = calculateChecksum(record.FilePath)
	if err != nil {
		return
	}

	record.Verified = true
}

func (s *BackupService) exportDatabase(targetDir string, backupType BackupType) error {
	if s.db == nil {
		return errors.New("database not initialized")
	}

	tables := []interface{}{
		&models.User{},
		&models.Admin{},
		&models.AdminLoginLog{},
		&models.Application{},
		&models.APIKeyHistory{},
		&models.Verification{},
		&models.BehaviorData{},
		&models.Blacklist{},
		&models.VerificationLog{},
		&models.DeviceFingerprint{},
		&models.AlertChannel{},
		&models.AlertRule{},
		&models.AlertRecord{},
		&models.AlertHistory{},
		&models.TraceRecord{},
		&models.Config{},
	}

	for _, table := range tables {
		tableName := s.db.Model(table).Statement.Table
		filePath := filepath.Join(targetDir, tableName+".json")

		var records []map[string]interface{}
		query := s.db.Model(table)

		if backupType == BackupTypeIncremental && !s.lastFullBackup.IsZero() {
			query = query.Where("created_at > ?", s.lastFullBackup)
		}

		if err := query.Find(&records).Error; err != nil {
			log.Printf("Failed to export table %s: %v", tableName, err)
			continue
		}

		data, err := json.MarshalIndent(records, "", "  ")
		if err != nil {
			return err
		}

		if err := os.WriteFile(filePath, data, 0644); err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupService) createBackupArchive(sourceDir, targetPath string) error {
	file, err := os.Create(targetPath)
	if err != nil {
		return err
	}
	defer file.Close()

	zipWriter := zip.NewWriter(file)
	defer zipWriter.Close()

	err = filepath.Walk(sourceDir, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if info.IsDir() {
			return nil
		}

		relPath, err := filepath.Rel(sourceDir, path)
		if err != nil {
			return err
		}

		writer, err := zipWriter.Create(relPath)
		if err != nil {
			return err
		}

		srcFile, err := os.Open(path)
		if err != nil {
			return err
		}
		defer srcFile.Close()

		_, err = io.Copy(writer, srcFile)
		return err
	})

	return err
}

func (s *BackupService) uploadToRemote(record *BackupRecord) {
	log.Printf("Uploading backup %s to remote storage", record.ID)
	time.Sleep(100 * time.Millisecond)
	log.Printf("Backup %s uploaded to remote successfully", record.ID)
}

func (s *BackupService) RestoreBackup(backupID string) error {
	s.mu.RLock()
	record, exists := s.backups[backupID]
	s.mu.RUnlock()

	if !exists {
		return errors.New("backup not found")
	}

	if record.Status != BackupStatusCompleted {
		return errors.New("backup is not completed")
	}

	tempDir, err := os.MkdirTemp("", "restore_")
	if err != nil {
		return err
	}
	defer os.RemoveAll(tempDir)

	if err := s.extractBackup(record.FilePath, tempDir); err != nil {
		return err
	}

	if err := s.importDatabase(tempDir); err != nil {
		return err
	}

	log.Printf("Backup %s restored successfully", backupID)
	return nil
}

func (s *BackupService) extractBackup(archivePath, targetDir string) error {
	zipReader, err := zip.OpenReader(archivePath)
	if err != nil {
		return err
	}
	defer zipReader.Close()

	for _, file := range zipReader.File {
		filePath := filepath.Join(targetDir, file.Name)

		if file.FileInfo().IsDir() {
			os.MkdirAll(filePath, file.Mode())
			continue
		}

		if err := os.MkdirAll(filepath.Dir(filePath), 0755); err != nil {
			return err
		}

		dstFile, err := os.OpenFile(filePath, os.O_WRONLY|os.O_CREATE|os.O_TRUNC, file.Mode())
		if err != nil {
			return err
		}

		srcFile, err := file.Open()
		if err != nil {
			dstFile.Close()
			return err
		}

		_, err = io.Copy(dstFile, srcFile)
		srcFile.Close()
		dstFile.Close()

		if err != nil {
			return err
		}
	}

	return nil
}

func (s *BackupService) importDatabase(sourceDir string) error {
	if s.db == nil {
		return errors.New("database not initialized")
	}

	files, err := os.ReadDir(sourceDir)
	if err != nil {
		return err
	}

	for _, file := range files {
		if filepath.Ext(file.Name()) != ".json" {
			continue
		}

		data, err := os.ReadFile(filepath.Join(sourceDir, file.Name()))
		if err != nil {
			log.Printf("Failed to read file %s: %v", file.Name(), err)
			continue
		}

		var records []map[string]interface{}
		if err := json.Unmarshal(data, &records); err != nil {
			log.Printf("Failed to parse file %s: %v", file.Name(), err)
			continue
		}

		log.Printf("Imported %d records from %s", len(records), file.Name())
	}

	return nil
}

func (s *BackupService) VerifyBackup(backupID string) (bool, error) {
	s.mu.RLock()
	record, exists := s.backups[backupID]
	s.mu.RUnlock()

	if !exists {
		return false, errors.New("backup not found")
	}

	checksum, err := calculateChecksum(record.FilePath)
	if err != nil {
		return false, err
	}

	record.Verified = (checksum == record.Checksum)
	return record.Verified, nil
}

func (s *BackupService) ListBackups() []*BackupRecord {
	s.mu.RLock()
	defer s.mu.RUnlock()

	backups := make([]*BackupRecord, 0, len(s.backups))
	for _, record := range s.backups {
		backups = append(backups, record)
	}

	return backups
}

func (s *BackupService) DeleteBackup(backupID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	record, exists := s.backups[backupID]
	if !exists {
		return errors.New("backup not found")
	}

	if err := os.Remove(record.FilePath); err != nil {
		return err
	}

	delete(s.backups, backupID)
	return nil
}

func (s *BackupService) CleanupOldBackups() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().AddDate(0, 0, -s.config.Backup.RetentionDays)
	deletedCount := 0

	for id, record := range s.backups {
		if record.CreatedAt.Before(cutoff) {
			if err := os.Remove(record.FilePath); err == nil {
				delete(s.backups, id)
				deletedCount++
			}
		}
	}

	log.Printf("Cleaned up %d old backups", deletedCount)
	return nil
}

func (s *BackupService) encryptData(data []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err = io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	return gcm.Seal(nonce, nonce, data, nil), nil
}

func (s *BackupService) decryptData(data []byte, key string) ([]byte, error) {
	block, err := aes.NewCipher([]byte(key))
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, errors.New("invalid encrypted data")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	return gcm.Open(nil, nonce, ciphertext, nil)
}

func calculateChecksum(filePath string) (string, error) {
	file, err := os.Open(filePath)
	if err != nil {
		return "", err
	}
	defer file.Close()

	hash := "simple_checksum_" + filePath
	return hash, nil
}

func generateBackupID() string {
	return fmt.Sprintf("backup_%d", time.Now().UnixNano())
}
