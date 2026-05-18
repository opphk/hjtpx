package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type AdvancedDataArchiver struct {
	enabled              bool
	config               *ArchiveConfiguration
	strategies           []ArchiveStrategy
	archiveHistory       []ArchiveOperationRecord
	stats                *ArchiveStats
	scheduler            *ArchiveScheduler
}

type ArchiveConfiguration struct {
	Enabled              bool
	ArchiveThresholdDays int
	CleanupThresholdDays int
	BatchSize            int
	ArchivePrefix        string
	CleanupPrefix        string
	AutoArchiveEnabled   bool
	ArchiveInterval      time.Duration
	CompressionEnabled   bool
	VerificationEnabled  bool
	MaxConcurrentOps     int
}

type ArchiveStrategy interface {
	Execute(ctx context.Context) error
	GetName() string
	GetStatus() string
	GetPriority() int
}

type ArchiveOperationRecord struct {
	Timestamp     time.Time
	Operation     string
	TableName     string
	RecordsCount  int64
	Duration      time.Duration
	Success       bool
	ErrorMessage  string
	SpaceSaved   int64
}

type ArchiveScheduler struct {
	mu           sync.RWMutex
	tasks        map[string]*ScheduledTask
	enabled      bool
}

type ScheduledTask struct {
	Name         string
	TableName    string
	Frequency    time.Duration
	LastRun      time.Time
	NextRun      time.Time
	Enabled      bool
	Strategy     ArchiveStrategy
}

type DataRetentionPolicy struct {
	TableName          string
	RetentionDays      int
	ArchiveBeforeDelete bool
	ArchiveTableName   string
}

type PartitionManager struct {
	db                *gorm.DB
	partitionConfigs   map[string]*PartitionConfig
}

type PartitionConfig struct {
	TableName       string
	PartitionType   string
	PartitionColumn string
	Interval        string
	RetentionCount  int
}

type CompressionStrategy struct {
	enabled     bool
	algorithm   string
	level       int
}

var advancedArchiver *AdvancedDataArchiver

func InitAdvancedDataArchiver(cfg *config.Config) {
	archiveConfig := &ArchiveConfiguration{
		Enabled:              cfg.Database.DataArchiving.Enabled,
		ArchiveThresholdDays: cfg.Database.DataArchiving.ArchiveThresholdDays,
		CleanupThresholdDays: cfg.Database.DataArchiving.CleanupThresholdDays,
		BatchSize:            1000,
		ArchivePrefix:        "archive_",
		CleanupPrefix:        "cleanup_",
		AutoArchiveEnabled:   true,
		ArchiveInterval:      1 * time.Hour,
		CompressionEnabled:   false,
		VerificationEnabled:  true,
		MaxConcurrentOps:     3,
	}

	advancedArchiver = &AdvancedDataArchiver{
		enabled:        archiveConfig.Enabled,
		config:         archiveConfig,
		archiveHistory: make([]ArchiveOperationRecord, 0),
		stats: &ArchiveStats{
			TotalArchivedRecords: 0,
			TotalCleanedRecords:   0,
		},
		scheduler: &ArchiveScheduler{
			tasks:   make(map[string]*ScheduledTask),
			enabled: archiveConfig.AutoArchiveEnabled,
		},
	}

	if archiveConfig.AutoArchiveEnabled {
		go advancedArchiver.startScheduler()
		log.Println("Advanced data archiver initialized with auto-archiving")
	}
}

func GetAdvancedDataArchiver() *AdvancedDataArchiver {
	return advancedArchiver
}

func (a *AdvancedDataArchiver) startScheduler() {
	ticker := time.NewTicker(a.config.ArchiveInterval)
	defer ticker.Stop()

	for range ticker.C {
		a.runScheduledArchive()
	}
}

func (a *AdvancedDataArchiver) runScheduledArchive() {
	a.scheduler.mu.RLock()
	defer a.scheduler.mu.RUnlock()

	for _, task := range a.scheduler.tasks {
		if !task.Enabled {
			continue
		}

		if time.Now().After(task.NextRun) {
			go a.executeTask(task)
			task.NextRun = time.Now().Add(task.Frequency)
			task.LastRun = time.Now()
		}
	}
}

func (a *AdvancedDataArchiver) executeTask(task *ScheduledTask) {
	ctx := context.Background()

	start := time.Now()
	err := task.Strategy.Execute(ctx)

	record := ArchiveOperationRecord{
		Timestamp:    time.Now(),
		Operation:    "SCHEDULED_ARCHIVE",
		TableName:    task.TableName,
		Duration:     time.Since(start),
		Success:      err == nil,
	}

	if err != nil {
		record.ErrorMessage = err.Error()
		log.Printf("[ARCHIVER] Scheduled task %s failed: %v", task.Name, err)
	} else {
		log.Printf("[ARCHIVER] Scheduled task %s completed in %v", task.Name, record.Duration)
	}

	a.recordOperation(record)
}

func (a *AdvancedDataArchiver) RegisterTask(tableName string, frequency time.Duration, strategy ArchiveStrategy) {
	a.scheduler.mu.Lock()
	defer a.scheduler.mu.Unlock()

	task := &ScheduledTask{
		Name:      fmt.Sprintf("archive_%s", tableName),
		TableName: tableName,
		Frequency: frequency,
		LastRun:   time.Time{},
		NextRun:   time.Now().Add(frequency),
		Enabled:   true,
		Strategy:  strategy,
	}

	a.scheduler.tasks[tableName] = task
	log.Printf("[ARCHIVER] Registered scheduled archive task for table %s (frequency: %v)", tableName, frequency)
}

func (a *AdvancedDataArchiver) ArchiveTableAdvanced(tableName string, dateField string, olderThan time.Time) (int64, error) {
	if !a.enabled {
		return 0, fmt.Errorf("archiving is disabled")
	}

	start := time.Now()
	archiveTable := a.config.ArchivePrefix + tableName

	if err := a.ensureArchiveTableExists(tableName, archiveTable); err != nil {
		return 0, err
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var count int64
	if err := tx.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), olderThan).
		Count(&count).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if count == 0 {
		tx.Commit()
		return 0, nil
	}

	batchCount := (count + int64(a.config.BatchSize) - 1) / int64(a.config.BatchSize)

	for i := int64(0); i < batchCount; i++ {
		insertSQL := fmt.Sprintf(`
			INSERT INTO %s
			SELECT * FROM %s
			WHERE %s < ?
			LIMIT %d
		`, archiveTable, tableName, dateField, a.config.BatchSize)

		if err := tx.Exec(insertSQL, olderThan).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to archive batch %d: %w", i, err)
		}

		deleteSQL := fmt.Sprintf(`
			DELETE FROM %s
			WHERE ctid IN (
				SELECT ctid FROM %s
				WHERE %s < ?
				LIMIT %d
			)
		`, tableName, tableName, dateField, a.config.BatchSize)

		if err := tx.Exec(deleteSQL, olderThan).Error; err != nil {
			tx.Rollback()
			return 0, fmt.Errorf("failed to delete batch %d: %w", i, err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	atomic.AddInt64(&a.stats.TotalArchivedRecords, count)
	a.stats.LastArchiveTime = time.Now()

	record := ArchiveOperationRecord{
		Timestamp:    time.Now(),
		Operation:    "ARCHIVE",
		TableName:    tableName,
		RecordsCount: count,
		Duration:     time.Since(start),
		Success:      true,
	}

	a.recordOperation(record)

	log.Printf("[ARCHIVER] Archived %d records from %s to %s in %v",
		count, tableName, archiveTable, record.Duration)

	return count, nil
}

func (a *AdvancedDataArchiver) CleanupArchiveAdvanced(tableName string, olderThan time.Time) (int64, error) {
	if !a.enabled {
		return 0, fmt.Errorf("archiving is disabled")
	}

	start := time.Now()
	archiveTable := a.config.ArchivePrefix + tableName

	result := DB.Table(archiveTable).
		Where("created_at < ?", olderThan).
		Delete(nil)

	if result.Error != nil {
		atomic.AddInt64(&a.stats.CleanupErrors, 1)
		return 0, result.Error
	}

	atomic.AddInt64(&a.stats.TotalCleanedRecords, result.RowsAffected)
	a.stats.LastCleanupTime = time.Now()

	record := ArchiveOperationRecord{
		Timestamp:    time.Now(),
		Operation:    "CLEANUP",
		TableName:    archiveTable,
		RecordsCount: result.RowsAffected,
		Duration:     time.Since(start),
		Success:      true,
	}

	a.recordOperation(record)

	log.Printf("[ARCHIVER] Cleaned up %d records from archive %s", result.RowsAffected, archiveTable)

	return result.RowsAffected, nil
}

func (a *AdvancedDataArchiver) ensureArchiveTableExists(sourceTable, archiveTable string) error {
	var exists bool
	checkSQL := `
		SELECT EXISTS (
			SELECT FROM information_schema.tables 
			WHERE table_name = ?
		)
	`

	if err := DB.Raw(checkSQL, archiveTable).Scan(&exists).Error; err != nil {
		return err
	}

	if !exists {
		createSQL := fmt.Sprintf("CREATE TABLE %s AS TABLE %s WITH NO DATA", archiveTable, sourceTable)
		if err := DB.Exec(createSQL).Error; err != nil {
			return err
		}
		log.Printf("[ARCHIVER] Created archive table: %s", archiveTable)
	}

	return nil
}

func (a *AdvancedDataArchiver) recordOperation(record ArchiveOperationRecord) {
	a.archiveHistory = append(a.archiveHistory, record)
	if len(a.archiveHistory) > 1000 {
		a.archiveHistory = a.archiveHistory[1:]
	}
}

func (a *AdvancedDataArchiver) GetArchiveHistory() []ArchiveOperationRecord {
	history := make([]ArchiveOperationRecord, len(a.archiveHistory))
	copy(history, a.archiveHistory)
	return history
}

func (a *AdvancedDataArchiver) GetArchiveStats() *ArchiveStats {
	return a.stats
}

func (a *AdvancedDataArchiver) GetTableArchiveStatus(tableName string) (map[string]interface{}, error) {
	status := make(map[string]interface{})

	var mainCount, archiveCount int64

	if err := DB.Table(tableName).Count(&mainCount).Error; err != nil {
		return nil, err
	}

	archiveTable := a.config.ArchivePrefix + tableName
	if err := DB.Table(archiveTable).Count(&archiveCount).Error; err != nil {
		archiveCount = 0
	}

	var mainSize, archiveSize string
	DB.Raw("SELECT pg_size_pretty(pg_total_relation_size(?))", tableName).Scan(&mainSize)
	DB.Raw("SELECT pg_size_pretty(pg_total_relation_size(?))", archiveTable).Scan(&archiveSize)

	status["table_name"] = tableName
	status["main_table_count"] = mainCount
	status["archive_table_count"] = archiveCount
	status["total_records"] = mainCount + archiveCount
	status["archive_ratio"] = float64(archiveCount) / float64(mainCount+archiveCount) * 100
	status["main_table_size"] = mainSize
	status["archive_table_size"] = archiveSize

	return status, nil
}

type TimeBasedArchiveStrategyImpl struct {
	tableName   string
	dateField   string
	threshold   time.Duration
	archiver    *AdvancedDataArchiver
	status      string
}

func NewTimeBasedArchiveStrategy(tableName, dateField string, threshold time.Duration) *TimeBasedArchiveStrategyImpl {
	return &TimeBasedArchiveStrategyImpl{
		tableName: tableName,
		dateField: dateField,
		threshold: threshold,
		status:    "pending",
	}
}

func (s *TimeBasedArchiveStrategyImpl) Execute(ctx context.Context) error {
	s.status = "running"

	cutoff := time.Now().Add(-s.threshold)
	_, err := s.archiver.ArchiveTableAdvanced(s.tableName, s.dateField, cutoff)

	s.status = "completed"
	return err
}

func (s *TimeBasedArchiveStrategyImpl) GetName() string {
	return "time_based_archive"
}

func (s *TimeBasedArchiveStrategyImpl) GetStatus() string {
	return s.status
}

func (s *TimeBasedArchiveStrategyImpl) GetPriority() int {
	return 1
}

type SizeBasedArchiveStrategyImpl struct {
	tableName string
	dateField string
	maxSize   int64
	archiver  *AdvancedDataArchiver
	status    string
}

func NewSizeBasedArchiveStrategy(tableName, dateField string, maxSize int64) *SizeBasedArchiveStrategyImpl {
	return &SizeBasedArchiveStrategyImpl{
		tableName: tableName,
		dateField: dateField,
		maxSize:   maxSize,
		status:    "pending",
	}
}

func (s *SizeBasedArchiveStrategyImpl) Execute(ctx context.Context) error {
	s.status = "running"

	var count int64
	if err := DB.Table(s.tableName).Count(&count).Error; err != nil {
		return err
	}

	if count <= s.maxSize {
		s.status = "completed"
		return nil
	}

	excess := count - s.maxSize
	batchSize := int64(1000)
	cutoff := time.Now().Add(-30 * 24 * time.Hour)

	for i := int64(0); i < excess; i += batchSize {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		_, err := s.archiver.ArchiveTableAdvanced(s.tableName, s.dateField, cutoff)
		if err != nil {
			return err
		}
	}

	s.status = "completed"
	return nil
}

func (s *SizeBasedArchiveStrategyImpl) GetName() string {
	return "size_based_archive"
}

func (s *SizeBasedArchiveStrategyImpl) GetStatus() string {
	return s.status
}

func (s *SizeBasedArchiveStrategyImpl) GetPriority() int {
	return 2
}

func (a *AdvancedDataArchiver) ScheduleArchive(tableName, dateField string, frequency time.Duration) {
	strategy := NewTimeBasedArchiveStrategy(tableName, dateField, 30*24*time.Hour)
	strategy.archiver = a
	a.RegisterTask(tableName, frequency, strategy)

	log.Printf("[ARCHIVER] Scheduled archive for table %s (frequency: %v)", tableName, frequency)
}

func (a *AdvancedDataArchiver) ScheduleCleanup(tableName string, frequency time.Duration, olderThan time.Duration) {
	task := &ScheduledTask{
		Name:      fmt.Sprintf("cleanup_%s", tableName),
		TableName: tableName,
		Frequency: frequency,
		NextRun:   time.Now().Add(frequency),
		Enabled:   true,
	}

	task.Strategy = &CleanupStrategy{
		tableName:   tableName,
		olderThan:   olderThan,
		archiver:    a,
	}

	a.scheduler.mu.Lock()
	a.scheduler.tasks[fmt.Sprintf("cleanup_%s", tableName)] = task
	a.scheduler.mu.Unlock()

	log.Printf("[ARCHIVER] Scheduled cleanup for table %s (frequency: %v)", tableName, frequency)
}

type CleanupStrategy struct {
	tableName string
	olderThan time.Duration
	archiver  *AdvancedDataArchiver
	status    string
}

func (s *CleanupStrategy) Execute(ctx context.Context) error {
	s.status = "running"

	cutoff := time.Now().Add(-s.olderThan)
	_, err := s.archiver.CleanupArchiveAdvanced(s.tableName, cutoff)

	s.status = "completed"
	return err
}

func (s *CleanupStrategy) GetName() string {
	return "cleanup"
}

func (s *CleanupStrategy) GetStatus() string {
	return s.status
}

func (s *CleanupStrategy) GetPriority() int {
	return 3
}

func (a *AdvancedDataArchiver) EnableScheduler(enabled bool) {
	a.scheduler.mu.Lock()
	defer a.scheduler.mu.Unlock()

	a.scheduler.enabled = enabled
	log.Printf("[ARCHIVER] Scheduler %s", map[bool]string{true: "enabled", false: "disabled"}[enabled])
}

func (a *AdvancedDataArchiver) GetScheduledTasks() map[string]*ScheduledTask {
	tasks := make(map[string]*ScheduledTask)

	a.scheduler.mu.RLock()
	for k, v := range a.scheduler.tasks {
		taskCopy := *v
		tasks[k] = &taskCopy
	}
	a.scheduler.mu.RUnlock()

	return tasks
}

func (a *AdvancedDataArchiver) GenerateArchiveReport() map[string]interface{} {
	report := make(map[string]interface{})

	stats := a.GetArchiveStats()
	report["total_archived"] = atomic.LoadInt64(&stats.TotalArchivedRecords)
	report["total_cleaned"] = atomic.LoadInt64(&stats.TotalCleanedRecords)
	report["archive_errors"] = atomic.LoadInt64(&stats.ArchiveErrors)
	report["cleanup_errors"] = atomic.LoadInt64(&stats.CleanupErrors)
	report["last_archive"] = stats.LastArchiveTime
	report["last_cleanup"] = stats.LastCleanupTime

	report["history_count"] = len(a.archiveHistory)

	recentOps := make([]ArchiveOperationRecord, 0)
	if len(a.archiveHistory) > 10 {
		recentOps = a.archiveHistory[len(a.archiveHistory)-10:]
	} else {
		recentOps = a.archiveHistory
	}
	report["recent_operations"] = recentOps

	report["enabled"] = a.enabled
	report["auto_archive"] = a.config.AutoArchiveEnabled
	report["scheduled_tasks"] = len(a.scheduler.tasks)

	return report
}

type DataRetentionManager struct {
	mu       sync.RWMutex
	policies []DataRetentionPolicy
}

func NewDataRetentionManager() *DataRetentionManager {
	return &DataRetentionManager{
		policies: make([]DataRetentionPolicy, 0),
	}
}

func (m *DataRetentionManager) AddPolicy(policy DataRetentionPolicy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.policies = append(m.policies, policy)
	log.Printf("[RETENTION] Added retention policy for table %s: %d days", policy.TableName, policy.RetentionDays)
}

func (m *DataRetentionManager) GetPolicies() []DataRetentionPolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policies := make([]DataRetentionPolicy, len(m.policies))
	copy(policies, m.policies)
	return policies
}

func (m *DataRetentionManager) EnforcePolicies(ctx context.Context, archiver *AdvancedDataArchiver) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, policy := range m.policies {
		cutoff := time.Now().AddDate(0, 0, -policy.RetentionDays)

		if policy.ArchiveBeforeDelete {
			_, err := archiver.ArchiveTableAdvanced(policy.TableName, "created_at", cutoff)
			if err != nil {
				log.Printf("[RETENTION] Failed to archive %s: %v", policy.TableName, err)
				continue
			}
		}

		result := DB.Table(policy.TableName).Where("created_at < ?", cutoff).Delete(nil)
		if result.Error != nil {
			log.Printf("[RETENTION] Failed to delete old records from %s: %v", policy.TableName, result.Error)
		} else {
			log.Printf("[RETENTION] Deleted %d old records from %s", result.RowsAffected, policy.TableName)
		}
	}

	return nil
}

var globalRetentionManager *DataRetentionManager

func init() {
	globalRetentionManager = NewDataRetentionManager()
}

func GetDataRetentionManager() *DataRetentionManager {
	return globalRetentionManager
}
