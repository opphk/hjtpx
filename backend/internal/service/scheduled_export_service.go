package service

import (
	"encoding/json"
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/export"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/robfig/cron/v3"
)

// ScheduledExportService 定时导出服务
type ScheduledExportService struct {
	cronScheduler *cron.Cron
}

// NewScheduledExportService 创建定时导出服务
func NewScheduledExportService() *ScheduledExportService {
	return &ScheduledExportService{
		cronScheduler: cron.New(),
	}
}

// Start 启动调度器
func (s *ScheduledExportService) Start() {
	s.cronScheduler.Start()
	s.loadScheduledTasks()
}

// Stop 停止调度器
func (s *ScheduledExportService) Stop() {
	s.cronScheduler.Stop()
}

// CreateScheduledExport 创建定时导出任务
func (s *ScheduledExportService) CreateScheduledExport(task *models.ScheduledExport) error {
	if err := database.DB.Create(task).Error; err != nil {
		return err
	}
	s.scheduleTask(task)
	return nil
}

// UpdateScheduledExport 更新定时导出任务
func (s *ScheduledExportService) UpdateScheduledExport(task *models.ScheduledExport) error {
	// 移除旧的调度
	s.removeTask(task.ID)

	if err := database.DB.Save(task).Error; err != nil {
		return err
	}

	// 添加新的调度
	if task.IsEnabled {
		s.scheduleTask(task)
	}

	return nil
}

// DeleteScheduledExport 删除定时导出任务
func (s *ScheduledExportService) DeleteScheduledExport(id uint) error {
	s.removeTask(id)
	return database.DB.Delete(&models.ScheduledExport{}, id).Error
}

// GetScheduledExport 获取单个定时导出任务
func (s *ScheduledExportService) GetScheduledExport(id uint) (*models.ScheduledExport, error) {
	var task models.ScheduledExport
	err := database.DB.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

// ListScheduledExports 列出所有定时导出任务
func (s *ScheduledExportService) ListScheduledExports() ([]models.ScheduledExport, error) {
	var tasks []models.ScheduledExport
	err := database.DB.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

// ExecuteScheduledExport 立即执行定时导出任务
func (s *ScheduledExportService) ExecuteScheduledExport(id uint) error {
	task, err := s.GetScheduledExport(id)
	if err != nil {
		return err
	}
	return s.executeTask(task)
}

// 内部方法
func (s *ScheduledExportService) loadScheduledTasks() {
	var tasks []models.ScheduledExport
	if err := database.DB.Where("is_enabled = ?", true).Find(&tasks).Error; err != nil {
		return
	}

	for _, task := range tasks {
		s.scheduleTask(&task)
	}
}

func (s *ScheduledExportService) scheduleTask(task *models.ScheduledExport) {
	_, err := s.cronScheduler.AddFunc(task.CronExpression, func() {
		if err := s.executeTask(task); err != nil {
			log.Printf("定时导出任务执行失败: %v", err)
		}
	})

	if err == nil {
		s.calculateNextRun(task)
		if err := database.DB.Save(task).Error; err != nil {
			log.Printf("保存定时任务失败: %v", err)
		}
	}
}

func (s *ScheduledExportService) removeTask(id uint) {
	// cron 包没有直接移除任务的方法，这里简化处理
	// 实际项目中可能需要用更复杂的方式管理任务
}

func (s *ScheduledExportService) executeTask(task *models.ScheduledExport) error {
	now := time.Now()
	task.LastRunAt = &now
	task.LastStatus = "running"
	if err := database.DB.Save(task).Error; err != nil {
		log.Printf("保存任务状态失败: %v", err)
	}

	// 执行导出
	history, err := s.performExport(task)
	if err != nil {
		task.LastStatus = "failed"
		task.LastErrorMessage = err.Error()
		if saveErr := database.DB.Save(task).Error; saveErr != nil {
			log.Printf("保存任务失败状态失败: %v", saveErr)
		}
		return err
	}

	task.LastStatus = "success"
	task.LastErrorMessage = ""
	s.calculateNextRun(task)
	if err := database.DB.Save(task).Error; err != nil {
		log.Printf("保存任务成功状态失败: %v", err)
	}

	// 记录导出历史
	if history != nil {
		if err := database.DB.Create(history).Error; err != nil {
			log.Printf("创建导出历史记录失败: %v", err)
		}
	}

	return nil
}

func (s *ScheduledExportService) performExport(task *models.ScheduledExport) (*models.ExportHistory, error) {
	// 解析过滤条件
	var filters map[string]interface{}
	if task.Filters != "" {
		_ = json.Unmarshal([]byte(task.Filters), &filters)
	}

	// 获取数据
	var logs []models.VerificationLog
	query := database.DB.Model(&models.VerificationLog{}).Preload("Application")

	// 应用过滤条件
	if appID, ok := filters["application_id"].(float64); ok {
		query = query.Where("application_id = ?", uint(appID))
	}
	if status, ok := filters["status"].(string); ok {
		query = query.Where("status = ?", status)
	}

	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return nil, err
	}

	// 导出数据
	exportData := export.ConvertLogsToExportData(logs, task.Name)
	exporter := export.GetExporter(task.ExportFormat)
	data, err := exporter.Export(exportData)
	if err != nil {
		return nil, err
	}

	// 这里应该保存文件到存储服务，现在简化处理
	filePath := fmt.Sprintf("/tmp/export_%d_%s.%s", task.ID, time.Now().Format("20060102150405"), task.ExportFormat)

	// 创建导出历史记录
	history := &models.ExportHistory{
		ScheduledExportID: &task.ID,
		Name:              task.Name,
		ExportType:        task.ExportType,
		ExportFormat:      task.ExportFormat,
		FileSize:          int64(len(data)),
		RecordCount:       len(logs),
		FilePath:          filePath,
		Status:            "completed",
		TriggeredBy:       "scheduler",
	}

	return history, nil
}

func (s *ScheduledExportService) calculateNextRun(task *models.ScheduledExport) {
	schedule, err := cron.ParseStandard(task.CronExpression)
	if err == nil {
		next := schedule.Next(time.Now())
		task.NextRunAt = &next
	}
}

// ReportTemplateService 报表模板服务
type ReportTemplateService struct{}

// NewReportTemplateService 创建报表模板服务
func NewReportTemplateService() *ReportTemplateService {
	return &ReportTemplateService{}
}

// CreateReportTemplate 创建报表模板
func (s *ReportTemplateService) CreateReportTemplate(template *models.ReportTemplate) error {
	return database.DB.Create(template).Error
}

// UpdateReportTemplate 更新报表模板
func (s *ReportTemplateService) UpdateReportTemplate(template *models.ReportTemplate) error {
	return database.DB.Save(template).Error
}

// DeleteReportTemplate 删除报表模板
func (s *ReportTemplateService) DeleteReportTemplate(id uint) error {
	return database.DB.Delete(&models.ReportTemplate{}, id).Error
}

// GetReportTemplate 获取单个报表模板
func (s *ReportTemplateService) GetReportTemplate(id uint) (*models.ReportTemplate, error) {
	var template models.ReportTemplate
	err := database.DB.Preload("VisualizationChart").First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// ListReportTemplates 列出所有报表模板
func (s *ReportTemplateService) ListReportTemplates() ([]models.ReportTemplate, error) {
	var templates []models.ReportTemplate
	err := database.DB.Order("created_at DESC").Find(&templates).Error
	return templates, err
}

// ExportHistoryService 导出历史服务
type ExportHistoryService struct{}

// NewExportHistoryService 创建导出历史服务
func NewExportHistoryService() *ExportHistoryService {
	return &ExportHistoryService{}
}

// ListExportHistory 列出导出历史
func (s *ExportHistoryService) ListExportHistory(page, pageSize int) ([]models.ExportHistory, int64, error) {
	var histories []models.ExportHistory
	var total int64

	offset := (page - 1) * pageSize

	if err := database.DB.Model(&models.ExportHistory{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}

	if err := database.DB.Preload("ScheduledExport").
		Order("created_at DESC").
		Offset(offset).
		Limit(pageSize).
		Find(&histories).Error; err != nil {
		return nil, 0, err
	}

	return histories, total, nil
}
