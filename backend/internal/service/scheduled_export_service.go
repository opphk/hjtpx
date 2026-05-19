package service

import (
	"encoding/json"
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/export"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/robfig/cron/v3"
)

type ScheduledExportService struct {
	cronScheduler    *cron.Cron
	taskEntries      map[uint]cron.EntryID
	mu               sync.RWMutex
	config           *ScheduledExportConfig
	executor         *TaskExecutor
	metricsCollector *ScheduledExportMetrics
}

type ScheduledExportConfig struct {
	MaxConcurrentExports int
	DefaultTimeout        time.Duration
	RetryCount            int
	RetryDelay            time.Duration
	EnableMetrics         bool
	MetricsInterval       time.Duration
	EnableNotifications   bool
	DefaultExportFormat   string
	DefaultExportPath     string
}

var DefaultScheduledExportConfig = &ScheduledExportConfig{
	MaxConcurrentExports: 5,
	DefaultTimeout:        10 * time.Minute,
	RetryCount:            3,
	RetryDelay:            1 * time.Minute,
	EnableMetrics:         true,
	MetricsInterval:       1 * time.Minute,
	EnableNotifications:   true,
	DefaultExportFormat:   "xlsx",
	DefaultExportPath:     "/tmp/exports",
}

type TaskExecutor struct {
	maxConcurrent int
	semaphore     chan struct{}
	timeout       time.Duration
}

func NewTaskExecutor(config *ScheduledExportConfig) *TaskExecutor {
	if config == nil {
		config = DefaultScheduledExportConfig
	}
	
	return &TaskExecutor{
		maxConcurrent: config.MaxConcurrentExports,
		semaphore:     make(chan struct{}, config.MaxConcurrentExports),
		timeout:       config.DefaultTimeout,
	}
}

func (e *TaskExecutor) Execute(task func() error) error {
	e.semaphore <- struct{}{}
	defer func() { <-e.semaphore }()
	
	done := make(chan error, 1)
	go func() {
		done <- task()
	}()
	
	select {
	case err := <-done:
		return err
	case <-time.After(e.timeout):
		return fmt.Errorf("task execution timeout after %v", e.timeout)
	}
}

type ScheduledExportMetrics struct {
	TotalExecutions    int64
	SuccessfulExecutions int64
	FailedExecutions   int64
	TotalExportSize    int64
	TotalRecords       int64
	LastExecutionTime  time.Time
	LastExecutionStatus string
	AverageExecTime    time.Duration
}

func NewScheduledExportMetrics() *ScheduledExportMetrics {
	return &ScheduledExportMetrics{
		AverageExecTime: 0,
	}
}

func NewScheduledExportService(config *ScheduledExportConfig) *ScheduledExportService {
	if config == nil {
		config = DefaultScheduledExportConfig
	}
	
	return &ScheduledExportService{
		cronScheduler:    cron.New(cron.WithSeconds()),
		taskEntries:      make(map[uint]cron.EntryID),
		config:           config,
		executor:         NewTaskExecutor(config),
		metricsCollector: NewScheduledExportMetrics(),
	}
}

func (s *ScheduledExportService) Start() {
	s.cronScheduler.Start()
	s.loadScheduledTasks()
	
	if s.config.EnableMetrics {
		go s.collectMetrics()
	}
	
	log.Printf("Scheduled export service started")
}

func (s *ScheduledExportService) Stop() {
	s.cronScheduler.Stop()
	log.Printf("Scheduled export service stopped")
}

func (s *ScheduledExportService) CreateScheduledExport(task *models.ScheduledExport) error {
	if err := database.DB.Create(task).Error; err != nil {
		return err
	}
	s.scheduleTask(task)
	return nil
}

func (s *ScheduledExportService) UpdateScheduledExport(task *models.ScheduledExport) error {
	s.mu.Lock()
	if entryID, exists := s.taskEntries[task.ID]; exists {
		s.cronScheduler.Remove(entryID)
		delete(s.taskEntries, task.ID)
	}
	s.mu.Unlock()
	
	if err := database.DB.Save(task).Error; err != nil {
		return err
	}
	
	if task.IsEnabled {
		s.scheduleTask(task)
	}
	
	return nil
}

func (s *ScheduledExportService) DeleteScheduledExport(id uint) error {
	s.mu.Lock()
	if entryID, exists := s.taskEntries[id]; exists {
		s.cronScheduler.Remove(entryID)
		delete(s.taskEntries, id)
	}
	s.mu.Unlock()
	
	return database.DB.Delete(&models.ScheduledExport{}, id).Error
}

func (s *ScheduledExportService) GetScheduledExport(id uint) (*models.ScheduledExport, error) {
	var task models.ScheduledExport
	err := database.DB.First(&task, id).Error
	if err != nil {
		return nil, err
	}
	return &task, nil
}

func (s *ScheduledExportService) ListScheduledExports() ([]models.ScheduledExport, error) {
	var tasks []models.ScheduledExport
	err := database.DB.Order("created_at DESC").Find(&tasks).Error
	return tasks, err
}

func (s *ScheduledExportService) ExecuteScheduledExport(id uint) error {
	task, err := s.GetScheduledExport(id)
	if err != nil {
		return err
	}
	return s.executeTask(task)
}

func (s *ScheduledExportService) PauseScheduledExport(id uint) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if entryID, exists := s.taskEntries[id]; exists {
		s.cronScheduler.Remove(entryID)
		delete(s.taskEntries, id)
	}
	
	task, err := s.GetScheduledExport(id)
	if err != nil {
		return err
	}
	
	task.IsEnabled = false
	return database.DB.Save(task).Error
}

func (s *ScheduledExportService) ResumeScheduledExport(id uint) error {
	task, err := s.GetScheduledExport(id)
	if err != nil {
		return err
	}
	
	task.IsEnabled = true
	if err := database.DB.Save(task).Error; err != nil {
		return err
	}
	
	s.scheduleTask(task)
	return nil
}

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
	s.mu.Lock()
	defer s.mu.Unlock()
	
	cronExpr := s.convertToCronExpression(task.CronExpression)
	
	entryID, err := s.cronScheduler.AddFunc(cronExpr, func() {
		s.executeTaskWithRetry(task)
	})
	
	if err != nil {
		log.Printf("Failed to schedule task %d: %v", task.ID, err)
		return
	}
	
	s.taskEntries[task.ID] = entryID
	s.calculateNextRun(task)
	if err := database.DB.Save(task).Error; err != nil {
		log.Printf("Failed to save task schedule: %v", err)
	}
}

func (s *ScheduledExportService) convertToCronExpression(cronExpr string) string {
	parts := parseCronExpression(cronExpr)
	if len(parts) == 5 {
		return "0 " + cronExpr
	}
	return cronExpr
}

func parseCronExpression(cronExpr string) []string {
	var parts []string
	current := ""
	for _, c := range cronExpr {
		if c == ' ' {
			if current != "" {
				parts = append(parts, current)
				current = ""
			}
		} else {
			current += string(c)
		}
	}
	if current != "" {
		parts = append(parts, current)
	}
	return parts
}

func (s *ScheduledExportService) executeTaskWithRetry(task *models.ScheduledExport) {
	var lastErr error
	for i := 0; i <= s.config.RetryCount; i++ {
		if i > 0 {
			log.Printf("Retrying task %d, attempt %d/%d", task.ID, i, s.config.RetryCount)
			time.Sleep(s.config.RetryDelay)
		}
		
		lastErr = s.executeTask(task)
		if lastErr == nil {
			return
		}
		log.Printf("Task %d failed on attempt %d: %v", task.ID, i+1, lastErr)
	}
	
	task.LastStatus = "failed"
	task.LastErrorMessage = lastErr.Error()
	database.DB.Save(task)
}

func (s *ScheduledExportService) executeTask(task *models.ScheduledExport) error {
	now := time.Now()
	task.LastRunAt = &now
	task.LastStatus = "running"
	if err := database.DB.Save(task).Error; err != nil {
		log.Printf("Failed to save task status: %v", err)
	}
	
	startTime := time.Now()
	
	err := s.executor.Execute(func() error {
		return s.performExport(task)
	})
	
	execDuration := time.Since(startTime)
	
	s.mu.Lock()
	s.metricsCollector.TotalExecutions++
	s.metricsCollector.LastExecutionTime = time.Now()
	s.metricsCollector.LastExecutionStatus = task.LastStatus
	
	if err == nil {
		s.metricsCollector.SuccessfulExecutions++
		task.LastStatus = "success"
		task.LastErrorMessage = ""
	} else {
		s.metricsCollector.FailedExecutions++
		task.LastStatus = "failed"
		task.LastErrorMessage = err.Error()
	}
	
	s.metricsCollector.AverageExecTime = (
		s.metricsCollector.AverageExecTime + execDuration) / 2
	
	s.mu.Unlock()
	
	s.calculateNextRun(task)
	if saveErr := database.DB.Save(task).Error; saveErr != nil {
		log.Printf("Failed to save task result: %v", saveErr)
	}
	
	return err
}

func (s *ScheduledExportService) performExport(task *models.ScheduledExport) error {
	var filters map[string]interface{}
	if task.Filters != "" {
		if err := json.Unmarshal([]byte(task.Filters), &filters); err != nil {
			return fmt.Errorf("failed to parse filters: %w", err)
		}
	}
	
	var logs []models.VerificationLog
	query := database.DB.Model(&models.VerificationLog{}).Preload("Application")
	
	if appID, ok := filters["application_id"].(float64); ok {
		query = query.Where("application_id = ?", uint(appID))
	}
	if status, ok := filters["status"].(string); ok {
		query = query.Where("status = ?", status)
	}
	if startDate, ok := filters["start_date"].(string); ok {
		if t, err := time.Parse("2006-01-02", startDate); err == nil {
			query = query.Where("created_at >= ?", t)
		}
	}
	if endDate, ok := filters["end_date"].(string); ok {
		if t, err := time.Parse("2006-01-02", endDate); err == nil {
			query = query.Where("created_at <= ?", t.Add(24*time.Hour))
		}
	}
	
	if err := query.Order("created_at DESC").Find(&logs).Error; err != nil {
		return fmt.Errorf("failed to query logs: %w", err)
	}
	
	exportData := export.ConvertLogsToExportData(logs, task.Name)
	exporter := export.GetExporter(task.ExportFormat)
	data, err := exporter.Export(exportData)
	if err != nil {
		return fmt.Errorf("failed to export data: %w", err)
	}
	
	fileName := fmt.Sprintf("export_%d_%s.%s", task.ID, time.Now().Format("20060102150405"), task.ExportFormat)
	filePath := fmt.Sprintf("%s/%s", s.config.DefaultExportPath, fileName)
	
	if err := s.saveExportFile(filePath, data); err != nil {
		return fmt.Errorf("failed to save export file: %w", err)
	}
	
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
	
	s.mu.Lock()
	s.metricsCollector.TotalExportSize += int64(len(data))
	s.metricsCollector.TotalRecords += int64(len(logs))
	s.mu.Unlock()
	
	if err := database.DB.Create(history).Error; err != nil {
		log.Printf("Failed to create export history: %v", err)
	}
	
	if s.config.EnableNotifications && task.EmailRecipients != "" {
		go s.sendExportNotification(task, history)
	}
	
	return nil
}

func (s *ScheduledExportService) saveExportFile(path string, data []byte) error {
	dir := path[:len(path)-len("/"+path[len(path)-strings.LastIndex(path, "/"):])]
	if err := os.MkdirAll(dir, 0755); err != nil {
		return err
	}
	
	return os.WriteFile(path, data, 0644)
}

func (s *ScheduledExportService) sendExportNotification(task *models.ScheduledExport, history *models.ExportHistory) {
	log.Printf("Export notification would be sent to: %s", task.EmailRecipients)
}

func (s *ScheduledExportService) calculateNextRun(task *models.ScheduledExport) {
	schedule, err := cron.ParseStandard(task.CronExpression)
	if err == nil {
		next := schedule.Next(time.Now())
		task.NextRunAt = &next
	}
}

func (s *ScheduledExportService) GetMetrics() *ScheduledExportMetrics {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	metrics := *s.metricsCollector
	return &metrics
}

func (s *ScheduledExportService) collectMetrics() {
	ticker := time.NewTicker(s.config.MetricsInterval)
	defer ticker.Stop()
	
	for range ticker.C {
		var recentExports []models.ScheduledExport
		if err := database.DB.Where("last_run_at > ?", time.Now().Add(-s.config.MetricsInterval)).
			Find(&recentExports).Error; err != nil {
			continue
		}
		
		s.mu.Lock()
		for _, export := range recentExports {
			if export.LastStatus == "success" {
				s.metricsCollector.SuccessfulExecutions++
			} else if export.LastStatus == "failed" {
				s.metricsCollector.FailedExecutions++
			}
			s.metricsCollector.TotalExecutions++
		}
		s.mu.Unlock()
	}
}

func (s *ScheduledExportService) ValidateCronExpression(cronExpr string) error {
	_, err := cron.ParseStandard(cronExpr)
	return err
}

func (s *ScheduledExportService) GetScheduledTaskStatus(id uint) (string, error) {
	task, err := s.GetScheduledExport(id)
	if err != nil {
		return "", err
	}
	
	s.mu.RLock()
	_, exists := s.taskEntries[id]
	s.mu.RUnlock()
	
	status := "disabled"
	if task.IsEnabled {
		if exists {
			status = "active"
		} else {
			status = "scheduled"
		}
	}
	
	return status, nil
}

type ReportTemplateService struct {
	db *models.ReportTemplate
}

func NewReportTemplateService() *ReportTemplateService {
	return &ReportTemplateService{}
}

func (s *ReportTemplateService) CreateReportTemplate(template *models.ReportTemplate) error {
	return database.DB.Create(template).Error
}

func (s *ReportTemplateService) UpdateReportTemplate(template *models.ReportTemplate) error {
	return database.DB.Save(template).Error
}

func (s *ReportTemplateService) DeleteReportTemplate(id uint) error {
	return database.DB.Delete(&models.ReportTemplate{}, id).Error
}

func (s *ReportTemplateService) GetReportTemplate(id uint) (*models.ReportTemplate, error) {
	var template models.ReportTemplate
	err := database.DB.Preload("VisualizationChart").First(&template, id).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

func (s *ReportTemplateService) ListReportTemplates() ([]models.ReportTemplate, error) {
	var templates []models.ReportTemplate
	err := database.DB.Order("created_at DESC").Find(&templates).Error
	return templates, err
}

type ExportHistoryService struct{}

func NewExportHistoryService() *ExportHistoryService {
	return &ExportHistoryService{}
}

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

func (s *ExportHistoryService) GetExportHistory(id uint) (*models.ExportHistory, error) {
	var history models.ExportHistory
	err := database.DB.Preload("ScheduledExport").First(&history, id).Error
	if err != nil {
		return nil, err
	}
	return &history, nil
}

func (s *ExportHistoryService) DeleteExportHistory(id uint) error {
	return database.DB.Delete(&models.ExportHistory{}, id).Error
}

func (s *ExportHistoryService) GetExportHistoryByScheduledExport(scheduledExportID uint) ([]models.ExportHistory, error) {
	var histories []models.ExportHistory
	err := database.DB.Where("scheduled_export_id = ?", scheduledExportID).
		Order("created_at DESC").
		Find(&histories).Error
	return histories, err
}
