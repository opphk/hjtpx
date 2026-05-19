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

type DataArchiver struct {
	enabled          bool
	archiveThreshold time.Duration
	archivePrefix    string
	cleanupEnabled   bool
	cleanupThreshold time.Duration
	archiveStats     *ArchiveStats
}

type ArchiveStats struct {
	TotalArchivedRecords int64
	TotalCleanedRecords  int64
	LastArchiveTime      time.Time
	LastCleanupTime      time.Time
	ArchiveErrors        int64
	CleanupErrors        int64
}

var archiver *DataArchiver
var globalArchiveStats ArchiveStats

func InitDataArchiving(cfg *config.Config) {
	archiver = &DataArchiver{
		enabled:          cfg.Database.DataArchiving.Enabled,
		archiveThreshold: time.Duration(cfg.Database.DataArchiving.ArchiveThresholdDays) * 24 * time.Hour,
		archivePrefix:    cfg.Database.DataArchiving.ArchiveTablePrefix,
		cleanupEnabled:   cfg.Database.DataArchiving.AutoCleanupEnabled,
		cleanupThreshold: time.Duration(cfg.Database.DataArchiving.CleanupThresholdDays) * 24 * time.Hour,
		archiveStats:     &globalArchiveStats,
	}

	if archiver.enabled {
		log.Println("Data archiving initialized")
		go archiver.startAutoArchiving(cfg)
	}
}

func GetArchiver() *DataArchiver {
	return archiver
}

func (a *DataArchiver) ArchiveTable(tableName string, dateField string, olderThan time.Time) (int64, error) {
	if !a.enabled {
		return 0, fmt.Errorf("archiving not enabled")
	}

	archiveTableName := a.archivePrefix + tableName

	if err := a.ensureArchiveTableExists(tableName, archiveTableName); err != nil {
		return 0, err
	}

	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	cutoffDate := olderThan
	if cutoffDate.IsZero() {
		cutoffDate = time.Now().Add(-a.archiveThreshold)
	}

	var count int64
	if err := tx.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), cutoffDate).
		Count(&count).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if count == 0 {
		tx.Commit()
		return 0, nil
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s
		WHERE %s < ?
	`, archiveTableName, tableName, dateField)

	if err := tx.Exec(insertSQL, cutoffDate).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), cutoffDate).
		Delete(nil).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	atomic.AddInt64(&globalArchiveStats.TotalArchivedRecords, count)
	globalArchiveStats.LastArchiveTime = time.Now()

	log.Printf("Archived %d records from %s to %s", count, tableName, archiveTableName)
	return count, nil
}

func (a *DataArchiver) BatchArchive(tableName string, dateField string, batchSize int, olderThan time.Time) (int64, error) {
	if !a.enabled {
		return 0, fmt.Errorf("archiving not enabled")
	}

	totalArchived := int64(0)
	cutoffDate := olderThan
	if cutoffDate.IsZero() {
		cutoffDate = time.Now().Add(-a.archiveThreshold)
	}

	for {
		archived, err := a.archiveBatch(tableName, dateField, cutoffDate, batchSize)
		if err != nil {
			atomic.AddInt64(&globalArchiveStats.ArchiveErrors, 1)
			return totalArchived, err
		}

		totalArchived += archived

		if archived < int64(batchSize) {
			break
		}

		time.Sleep(100 * time.Millisecond)
	}

	return totalArchived, nil
}

func (a *DataArchiver) archiveBatch(tableName string, dateField string, cutoffDate time.Time, batchSize int) (int64, error) {
	archiveTableName := a.archivePrefix + tableName

	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var count int64
	if err := tx.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), cutoffDate).
		Limit(batchSize).
		Count(&count).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if count == 0 {
		tx.Commit()
		return 0, nil
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s
		WHERE %s < ?
		LIMIT %d
	`, archiveTableName, tableName, dateField, batchSize)

	if err := tx.Exec(insertSQL, cutoffDate).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	deleteSQL := fmt.Sprintf(`
		DELETE FROM %s
		WHERE ctid IN (
			SELECT ctid FROM %s
			WHERE %s < ?
			LIMIT %d
		)
	`, tableName, tableName, dateField, batchSize)

	if err := tx.Exec(deleteSQL, cutoffDate).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return count, nil
}

func (a *DataArchiver) ensureArchiveTableExists(sourceTable, archiveTable string) error {
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
		log.Printf("Created archive table: %s", archiveTable)
	}

	return nil
}

func (a *DataArchiver) CleanupArchive(tableName string, olderThan time.Time) (int64, error) {
	if !a.cleanupEnabled {
		return 0, fmt.Errorf("cleanup not enabled")
	}

	archiveTableName := a.archivePrefix + tableName

	cutoffDate := olderThan
	if cutoffDate.IsZero() {
		cutoffDate = time.Now().Add(-a.cleanupThreshold)
	}

	result := DB.Table(archiveTableName).
		Where("created_at < ?", cutoffDate).
		Delete(nil)

	if result.Error != nil {
		atomic.AddInt64(&globalArchiveStats.CleanupErrors, 1)
		return 0, result.Error
	}

	atomic.AddInt64(&globalArchiveStats.TotalCleanedRecords, result.RowsAffected)
	globalArchiveStats.LastCleanupTime = time.Now()

	log.Printf("Cleaned up %d records from archive %s", result.RowsAffected, archiveTableName)
	return result.RowsAffected, nil
}

func (a *DataArchiver) BatchCleanup(tableName string, batchSize int, olderThan time.Time) (int64, error) {
	if !a.cleanupEnabled {
		return 0, fmt.Errorf("cleanup not enabled")
	}

	totalCleaned := int64(0)
	cutoffDate := olderThan
	if cutoffDate.IsZero() {
		cutoffDate = time.Now().Add(-a.cleanupThreshold)
	}

	for {
		cleaned, err := a.cleanupBatch(tableName, cutoffDate, batchSize)
		if err != nil {
			atomic.AddInt64(&globalArchiveStats.CleanupErrors, 1)
			return totalCleaned, err
		}

		totalCleaned += cleaned

		if cleaned < int64(batchSize) {
			break
		}

		time.Sleep(50 * time.Millisecond)
	}

	return totalCleaned, nil
}

func (a *DataArchiver) cleanupBatch(tableName string, cutoffDate time.Time, batchSize int) (int64, error) {
	archiveTableName := a.archivePrefix + tableName

	deleteSQL := fmt.Sprintf(`
		DELETE FROM %s
		WHERE ctid IN (
			SELECT ctid FROM %s
			WHERE created_at < ?
			LIMIT %d
		)
	`, archiveTableName, archiveTableName, batchSize)

	result := DB.Exec(deleteSQL, cutoffDate)
	if result.Error != nil {
		return 0, result.Error
	}

	return result.RowsAffected, nil
}

func (a *DataArchiver) GetArchiveStats(tableName string) (map[string]interface{}, error) {
	archiveTableName := a.archivePrefix + tableName

	var mainCount, archiveCount int64

	if err := DB.Table(tableName).Count(&mainCount).Error; err != nil {
		return nil, err
	}

	if err := DB.Table(archiveTableName).Count(&archiveCount).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"main_table_count":    mainCount,
		"archive_table_count": archiveCount,
		"total_records":       mainCount + archiveCount,
		"archive_ratio":       float64(archiveCount) / float64(mainCount+archiveCount) * 100,
	}, nil
}

func (a *DataArchiver) GetGlobalArchiveStats() *ArchiveStats {
	return &globalArchiveStats
}

func (a *DataArchiver) RestoreFromArchive(tableName string, ids []interface{}) (int64, error) {
	if len(ids) == 0 {
		return 0, nil
	}

	archiveTableName := a.archivePrefix + tableName

	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s
		WHERE id IN ?
	`, tableName, archiveTableName)

	if err := tx.Exec(insertSQL, ids).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Table(archiveTableName).Where("id IN ?", ids).Delete(nil).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return int64(len(ids)), nil
}

func (a *DataArchiver) QueryArchive(tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	archiveTableName := a.archivePrefix + tableName

	query := DB.Table(archiveTableName)
	for k, v := range conditions {
		query = query.Where(k, v)
	}

	if limit > 0 {
		query = query.Limit(limit)
	}
	if offset > 0 {
		query = query.Offset(offset)
	}

	var results []map[string]interface{}
	if err := query.Find(&results).Error; err != nil {
		return nil, err
	}

	return results, nil
}

func (a *DataArchiver) startAutoArchiving(cfg *config.Config) {
	if !a.enabled {
		return
	}

	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		a.runScheduledArchiving()
	}
}

func (a *DataArchiver) runScheduledArchiving() {
	tables := []string{
		"verification_logs",
		"risk_logs",
		"audit_logs",
	}

	for _, table := range tables {
		stats, err := a.GetArchiveStats(table)
		if err != nil {
			continue
		}

		ratio, ok := stats["archive_ratio"].(float64)
		if !ok {
			continue
		}

		if ratio > 50 {
			_, err := a.ArchiveTable(table, "created_at", time.Now().Add(-a.archiveThreshold))
			if err != nil {
				log.Printf("Auto archive failed for %s: %v", table, err)
			}
		}
	}
}

type HotColdSeparator struct {
	hotThreshold  time.Duration
	coldThreshold time.Duration
	archivePrefix string
}

func NewHotColdSeparator(hotThreshold, coldThreshold time.Duration) *HotColdSeparator {
	return &HotColdSeparator{
		hotThreshold:  hotThreshold,
		coldThreshold: coldThreshold,
		archivePrefix: "archive_",
	}
}

func (hc *HotColdSeparator) ClassifyData(tableName string, dateField string) (map[string]interface{}, error) {
	now := time.Now()
	hotBoundary := now.Add(-hc.hotThreshold)
	coldBoundary := now.Add(-hc.coldThreshold)

	var hotCount, warmCount, coldCount int64

	if err := DB.Table(tableName).
		Where(fmt.Sprintf("%s >= ?", dateField), hotBoundary).
		Count(&hotCount).Error; err != nil {
		return nil, err
	}

	if err := DB.Table(tableName).
		Where(fmt.Sprintf("%s >= ? AND %s < ?", dateField, dateField), coldBoundary, hotBoundary).
		Count(&warmCount).Error; err != nil {
		return nil, err
	}

	if err := DB.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), coldBoundary).
		Count(&coldCount).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"hot_count":  hotCount,
		"warm_count": warmCount,
		"cold_count": coldCount,
		"total":      hotCount + warmCount + coldCount,
	}, nil
}

func (hc *HotColdSeparator) SeparateColdData(tableName string, dateField string) (int64, error) {
	coldBoundary := time.Now().Add(-hc.coldThreshold)
	archiveTable := hc.archivePrefix + tableName

	tx := DB.Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	var count int64
	if err := tx.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), coldBoundary).
		Count(&count).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if count == 0 {
		tx.Commit()
		return 0, nil
	}

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s
		WHERE %s < ?
	`, archiveTable, tableName, dateField)

	if err := tx.Exec(insertSQL, coldBoundary).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Table(tableName).
		Where(fmt.Sprintf("%s < ?", dateField), coldBoundary).
		Delete(nil).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	return count, nil
}

type ArchiveStrategy interface {
	Execute(ctx context.Context) error
	GetName() string
	GetStatus() string
}

type TimeBasedArchiveStrategy struct {
	tableName string
	dateField string
	threshold time.Duration
	archiver  *DataArchiver
	status    string
	mu        sync.RWMutex
}

func NewTimeBasedArchiveStrategy(tableName, dateField string, threshold time.Duration) *TimeBasedArchiveStrategy {
	return &TimeBasedArchiveStrategy{
		tableName: tableName,
		dateField: dateField,
		threshold: threshold,
		status:    "pending",
	}
}

func (s *TimeBasedArchiveStrategy) Execute(ctx context.Context) error {
	s.mu.Lock()
	s.status = "running"
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.status = "completed"
		s.mu.Unlock()
	}()

	cutoff := time.Now().Add(-s.threshold)
	_, err := s.archiver.ArchiveTable(s.tableName, s.dateField, cutoff)
	return err
}

func (s *TimeBasedArchiveStrategy) GetName() string {
	return "time_based_archive"
}

func (s *TimeBasedArchiveStrategy) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}

type SizeBasedArchiveStrategy struct {
	tableName string
	dateField string
	maxSize   int64
	archiver  *DataArchiver
	status    string
	mu        sync.RWMutex
}

func NewSizeBasedArchiveStrategy(tableName, dateField string, maxSize int64) *SizeBasedArchiveStrategy {
	return &SizeBasedArchiveStrategy{
		tableName: tableName,
		dateField: dateField,
		maxSize:   maxSize,
		status:    "pending",
	}
}

func (s *SizeBasedArchiveStrategy) Execute(ctx context.Context) error {
	s.mu.Lock()
	s.status = "running"
	s.mu.Unlock()

	defer func() {
		s.mu.Lock()
		s.status = "completed"
		s.mu.Unlock()
	}()

	var count int64
	if err := DB.Table(s.tableName).Count(&count).Error; err != nil {
		return err
	}

	if count <= s.maxSize {
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

		_, err := s.archiver.ArchiveTable(s.tableName, s.dateField, cutoff)
		if err != nil {
			return err
		}
	}

	return nil
}

func (s *SizeBasedArchiveStrategy) GetName() string {
	return "size_based_archive"
}

func (s *SizeBasedArchiveStrategy) GetStatus() string {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.status
}
