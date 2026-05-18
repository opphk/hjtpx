package handler

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/config"
)

var (
	backupHandlerInstance *BackupHandler
	backupHandlerOnce     sync.Once
)

type BackupHandler struct {
	backupService *service.BackupService
}

type CreateBackupRequest struct {
	Type string `json:"type" binding:"required,oneof=full incremental"`
}

type RestoreBackupRequest struct {
	BackupID string `json:"backup_id" binding:"required"`
}

func NewBackupHandler(cfg *config.Config) *BackupHandler {
	return &BackupHandler{
		backupService: service.GetBackupService(cfg),
	}
}

func GetBackupHandler(cfg *config.Config) *BackupHandler {
	backupHandlerOnce.Do(func() {
		backupHandlerInstance = NewBackupHandler(cfg)
	})
	return backupHandlerInstance
}

func (h *BackupHandler) ListBackups(c *gin.Context) {
	backups := h.backupService.ListBackups()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    backups,
	})
}

func (h *BackupHandler) CreateBackup(c *gin.Context) {
	var req CreateBackupRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"code":    -1,
			"message": "invalid request",
			"error":   err.Error(),
		})
		return
	}

	var record *service.BackupRecord
	var err error

	if req.Type == "full" {
		record, err = h.backupService.CreateFullBackup()
	} else {
		record, err = h.backupService.CreateIncrementalBackup()
	}

	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "failed to create backup",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "backup created",
		"data":    record,
	})
}

func (h *BackupHandler) GetBackup(c *gin.Context) {
	backupID := c.Param("id")

	backups := h.backupService.ListBackups()
	var found *service.BackupRecord
	for _, backup := range backups {
		if backup.ID == backupID {
			found = backup
			break
		}
	}

	if found == nil {
		c.JSON(http.StatusNotFound, gin.H{
			"code":    -1,
			"message": "backup not found",
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    found,
	})
}

func (h *BackupHandler) DeleteBackup(c *gin.Context) {
	backupID := c.Param("id")

	if err := h.backupService.DeleteBackup(backupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "failed to delete backup",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "backup deleted",
	})
}

func (h *BackupHandler) RestoreBackup(c *gin.Context) {
	backupID := c.Param("id")

	if err := h.backupService.RestoreBackup(backupID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "failed to restore backup",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "backup restored successfully",
	})
}

func (h *BackupHandler) VerifyBackup(c *gin.Context) {
	backupID := c.Param("id")

	verified, err := h.backupService.VerifyBackup(backupID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "failed to verify backup",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":     0,
		"message":  "success",
		"verified": verified,
	})
}

func (h *BackupHandler) CleanupOldBackups(c *gin.Context) {
	if err := h.backupService.CleanupOldBackups(); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{
			"code":    -1,
			"message": "failed to cleanup old backups",
			"error":   err.Error(),
		})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "old backups cleaned up",
	})
}

func (h *BackupHandler) GetBackupConfig(c *gin.Context) {
	cfg := config.GetConfig()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"enabled":                    cfg.Backup.Enabled,
			"backup_dir":                 cfg.Backup.BackupDir,
			"auto_backup_enabled":        cfg.Backup.AutoBackupEnabled,
			"auto_backup_interval_hours": cfg.Backup.AutoBackupIntervalHours,
			"incremental_enabled":        cfg.Backup.IncrementalEnabled,
			"incremental_interval_mins":  cfg.Backup.IncrementalIntervalMins,
			"remote_backup_enabled":      cfg.Backup.RemoteBackupEnabled,
			"retention_days":             cfg.Backup.RetentionDays,
			"compression_enabled":        cfg.Backup.CompressionEnabled,
			"encryption_enabled":         cfg.Backup.EncryptionEnabled,
		},
	})
}
