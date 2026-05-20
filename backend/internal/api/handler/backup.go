package handler

import (
	"net/http"
	"sync"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/config"
)

var (
	backupHandlerInstance *BackupHandler
	backupHandlerOnce     sync.Once
)

type BackupHandler struct {
	backupService *service.BackupService
}

// CreateBackupRequest 创建备份请求
type CreateBackupRequest struct {
	Type string `json:"type" binding:"required,oneof=full incremental" example:"full"`
}

// RestoreBackupRequest 恢复备份请求
type RestoreBackupRequest struct {
	BackupID string `json:"backup_id" binding:"required" example:"backup-20240101-120000"`
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

// ListBackups 获取备份列表
// @Summary 获取备份列表
// @Description 获取所有备份记录列表
// @Tags 备份管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response{data=[]service.BackupRecord} "获取成功"
// @Router /api/v1/admin/backups [get]
func (h *BackupHandler) ListBackups(c *gin.Context) {
	backups := h.backupService.ListBackups()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    backups,
	})
}

// CreateBackup 创建备份
// @Summary 创建备份
// @Description 创建全量备份或增量备份
// @Tags 备份管理
// @Accept json
// @Produce json
// @Param body body CreateBackupRequest true "创建备份请求"
// @Success 200 {object} response.Response{data=service.BackupRecord} "创建成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/backups [post]
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

// GetBackup 获取备份详情
// @Summary 获取备份详情
// @Description 获取指定备份的详细信息
// @Tags 备份管理
// @Accept json
// @Produce json
// @Param id path string true "备份ID"
// @Success 200 {object} response.Response{data=service.BackupRecord} "获取成功"
// @Failure 404 {object} response.Response "备份不存在"
// @Router /api/v1/admin/backups/{id} [get]
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

// DeleteBackup 删除备份
// @Summary 删除备份
// @Description 删除指定的备份文件
// @Tags 备份管理
// @Accept json
// @Produce json
// @Param id path string true "备份ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/backups/{id} [delete]
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

// RestoreBackup 恢复备份
// @Summary 恢复备份
// @Description 从指定备份恢复数据
// @Tags 备份管理
// @Accept json
// @Produce json
// @Param id path string true "备份ID"
// @Success 200 {object} response.Response "恢复成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/backups/{id}/restore [post]
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

// VerifyBackup 验证备份
// @Summary 验证备份
// @Description 验证备份文件的完整性
// @Tags 备份管理
// @Accept json
// @Produce json
// @Param id path string true "备份ID"
// @Success 200 {object} response.Response "验证完成"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/backups/{id}/verify [get]
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

// CleanupOldBackups 清理旧备份
// @Summary 清理旧备份
// @Description 根据保留策略清理过期的备份文件
// @Tags 备份管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "清理成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/backups/cleanup [post]
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

// GetBackupConfig 获取备份配置
// @Summary 获取备份配置
// @Description 获取备份系统的当前配置参数
// @Tags 备份管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Router /api/v1/admin/backups/config [get]
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
