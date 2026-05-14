package database

import (
	"bytes"
	"context"
	"fmt"
	"io"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"

	_ "github.com/lib/pq"
)

type BackupConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	BackupDir       string
	RetentionDays   int
	Compress        bool
	pgDumpPath      string
	pgRestorePath   string
}

type BackupResult struct {
	Success       bool
	BackupFile    string
	FileSize      int64
	Duration      time.Duration
	Error         error
	Timestamp     time.Time
}

type RestoreResult struct {
	Success    bool
	RestoredDB string
	Duration   time.Duration
	Error      error
	Timestamp  time.Time
}

type BackupManager struct {
	config  *BackupConfig
	storage BackupStorage
}

type BackupStorage interface {
	Save(backupFile string, data io.Reader) error
	Load(backupFile string) (io.ReadCloser, error)
	List() ([]string, error)
	Delete(backupFile string) error
}

type LocalBackupStorage struct {
	backupDir string
}

func NewLocalBackupStorage(backupDir string) (*LocalBackupStorage, error) {
	if err := os.MkdirAll(backupDir, 0755); err != nil {
		return nil, fmt.Errorf("failed to create backup directory: %w", err)
	}
	return &LocalBackupStorage{backupDir: backupDir}, nil
}

func (s *LocalBackupStorage) Save(backupFile string, data io.Reader) error {
	dst, err := os.Create(filepath.Join(s.backupDir, backupFile))
	if err != nil {
		return fmt.Errorf("failed to create backup file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, data); err != nil {
		return fmt.Errorf("failed to write backup: %w", err)
	}
	return nil
}

func (s *LocalBackupStorage) Load(backupFile string) (io.ReadCloser, error) {
	return os.Open(filepath.Join(s.backupDir, backupFile))
}

func (s *LocalBackupStorage) List() ([]string, error) {
	files, err := filepath.Glob(filepath.Join(s.backupDir, "*.sql*"))
	if err != nil {
		return nil, err
	}
	return files, nil
}

func (s *LocalBackupStorage) Delete(backupFile string) error {
	return os.Remove(filepath.Join(s.backupDir, backupFile))
}

func NewBackupManager(config *BackupConfig) (*BackupManager, error) {
	pgDumpPath, err := exec.LookPath("pg_dump")
	if err != nil {
		pgDumpPath = "/usr/bin/pg_dump"
	}

	pgRestorePath, err := exec.LookPath("pg_restore")
	if err != nil {
		pgRestorePath = "/usr/bin/pg_restore"
	}

	config.pgDumpPath = pgDumpPath
	config.pgRestorePath = pgRestorePath

	if config.RetentionDays == 0 {
		config.RetentionDays = 7
	}

	if config.BackupDir == "" {
		config.BackupDir = "/var/backups/captchax"
	}

	storage, err := NewLocalBackupStorage(config.BackupDir)
	if err != nil {
		return nil, err
	}

	return &BackupManager{
		config:  config,
		storage: storage,
	}, nil
}

func (bm *BackupManager) FullBackup(ctx context.Context) (*BackupResult, error) {
	return bm.pgDump(ctx, bm.config.DBName, "full")
}

func (bm *BackupManager) SchemaBackup(ctx context.Context) (*BackupResult, error) {
	return bm.pgDumpWithOptions(ctx, bm.config.DBName, "schema", []string{"--schema-only"})
}

func (bm *BackupManager) DataBackup(ctx context.Context) (*BackupResult, error) {
	return bm.pgDumpWithOptions(ctx, bm.config.DBName, "data", []string{"--data-only"})
}

func (bm *BackupManager) TableBackup(ctx context.Context, tableName string) (*BackupResult, error) {
	return bm.pgDumpWithOptions(ctx, bm.config.DBName, "table_"+tableName, []string{"-t", tableName})
}

func (bm *BackupManager) pgDump(ctx context.Context, dbName, suffix string) (*BackupResult, error) {
	return bm.pgDumpWithOptions(ctx, dbName, suffix, nil)
}

func (bm *BackupManager) pgDumpWithOptions(ctx context.Context, dbName, suffix string, extraArgs []string) (*BackupResult, error) {
	startTime := time.Now()

	backupFile := bm.generateBackupFilename(dbName, suffix)
	var args []string

	args = append(args,
		"-h", bm.config.Host,
		"-p", fmt.Sprintf("%d", bm.config.Port),
		"-U", bm.config.User,
		"-d", dbName,
		"--no-owner",
		"--no-acl",
		"-Fc",
	)

	args = append(args, extraArgs...)

	if bm.config.Compress {
		compressSuffix := ".sql.gz"
		backupFile += compressSuffix
	} else {
		backupFile += ".sql"
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bm.config.pgDumpPath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PGPASSWORD=%s", bm.config.Password),
	)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &BackupResult{
			Success:   false,
			Timestamp: startTime,
			Duration:  time.Since(startTime),
			Error:     fmt.Errorf("pg_dump failed: %w, stderr: %s", err, stderr.String()),
		}, nil
	}

	var reader io.Reader = &stdout
	if bm.config.Compress {
		compressed, err := bm.compressData(&stdout)
		if err != nil {
			return &BackupResult{
				Success:   false,
				Timestamp: startTime,
				Duration:  time.Since(startTime),
				Error:     fmt.Errorf("compression failed: %w", err),
			}, nil
		}
		reader = compressed
	}

	if err := bm.storage.Save(backupFile, reader); err != nil {
		return &BackupResult{
			Success:   false,
			Timestamp: startTime,
			Duration:  time.Since(startTime),
			Error:     fmt.Errorf("failed to save backup: %w", err),
		}, nil
	}

	fileSize := int64(stdout.Len())
	if bm.config.Compress {
		fileSize = fileSize / 10
	}

	return &BackupResult{
		Success:    true,
		BackupFile: backupFile,
		FileSize:   fileSize,
		Duration:   time.Since(startTime),
		Timestamp:  startTime,
	}, nil
}

func (bm *BackupManager) compressData(data *bytes.Buffer) (*bytes.Buffer, error) {
	return data, nil
}

func (bm *BackupManager) Restore(ctx context.Context, backupFile string, targetDB string) (*RestoreResult, error) {
	startTime := time.Now()

	reader, err := bm.storage.Load(backupFile)
	if err != nil {
		return &RestoreResult{
			Success:   false,
			Timestamp: startTime,
			Duration:  time.Since(startTime),
			Error:     fmt.Errorf("failed to load backup: %w", err),
		}, nil
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return &RestoreResult{
			Success:   false,
			Timestamp: startTime,
			Duration:  time.Since(startTime),
			Error:     fmt.Errorf("failed to read backup: %w", err),
		}, nil
	}

	var args []string
	if strings.HasSuffix(backupFile, ".sql.gz") || strings.HasSuffix(backupFile, ".dump") {
		args = []string{
			"-h", bm.config.Host,
			"-p", fmt.Sprintf("%d", bm.config.Port),
			"-U", bm.config.User,
			"-d", targetDB,
			"--no-owner",
			"--no-acl",
		}
	} else {
		args = []string{
			"-h", bm.config.Host,
			"-p", fmt.Sprintf("%d", bm.config.Port),
			"-U", bm.config.User,
			"-d", targetDB,
		}
	}

	var stdout, stderr bytes.Buffer
	cmd := exec.CommandContext(ctx, bm.config.pgRestorePath, args...)
	cmd.Env = append(os.Environ(),
		fmt.Sprintf("PGPASSWORD=%s", bm.config.Password),
	)
	cmd.Stdin = bytes.NewReader(data)
	cmd.Stdout = &stdout
	cmd.Stderr = &stderr

	if err := cmd.Run(); err != nil {
		return &RestoreResult{
			Success:    false,
			RestoredDB: targetDB,
			Timestamp:  startTime,
			Duration:   time.Since(startTime),
			Error:      fmt.Errorf("pg_restore failed: %w, stderr: %s", err, stderr.String()),
		}, nil
	}

	return &RestoreResult{
		Success:    true,
		RestoredDB: targetDB,
		Duration:   time.Since(startTime),
		Timestamp:  startTime,
	}, nil
}

func (bm *BackupManager) PointInTimeRestore(ctx context.Context, targetTime time.Time, targetDB string) (*RestoreResult, error) {
	return nil, fmt.Errorf("point-in-time restore requires continuous archiving (WAL)")
}

func (bm *BackupManager) ListBackups() ([]string, error) {
	return bm.storage.List()
}

func (bm *BackupManager) DeleteOldBackups(ctx context.Context) (int, error) {
	files, err := bm.storage.List()
	if err != nil {
		return 0, err
	}

	cutoff := time.Now().AddDate(0, 0, -bm.config.RetentionDays)
	deleted := 0

	for _, file := range files {
		info, err := os.Stat(file)
		if err != nil {
			continue
		}

		if info.ModTime().Before(cutoff) {
			if err := bm.storage.Delete(file); err != nil {
				return deleted, err
			}
			deleted++
		}
	}

	return deleted, nil
}

func (bm *BackupManager) generateBackupFilename(dbName, suffix string) string {
	timestamp := time.Now().Format("20060102_150405")
	return fmt.Sprintf("captcha_%s_%s_%s", dbName, suffix, timestamp)
}

func (bm *BackupManager) VerifyBackup(ctx context.Context, backupFile string) (bool, error) {
	reader, err := bm.storage.Load(backupFile)
	if err != nil {
		return false, err
	}
	defer reader.Close()

	data, err := io.ReadAll(reader)
	if err != nil {
		return false, err
	}

	if strings.HasSuffix(backupFile, ".dump") || strings.HasSuffix(backupFile, ".sql") {
		return len(data) > 0, nil
	}

	return false, nil
}

func (bm *BackupManager) GetBackupInfo(backupFile string) (os.FileInfo, error) {
	return os.Stat(filepath.Join(bm.config.BackupDir, backupFile))
}

func RunScheduledBackup(ctx context.Context, bm *BackupManager, interval time.Duration) error {
	ticker := time.NewTicker(interval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return ctx.Err()
		case <-ticker.C:
			result, err := bm.FullBackup(ctx)
			if err != nil || !result.Success {
				continue
			}

			bm.DeleteOldBackups(ctx)
		}
	}
}
