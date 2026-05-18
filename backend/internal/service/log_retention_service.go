package service

import (
	"context"
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type LogRetentionPolicy struct {
	tableName         string
	hotDataDays       int
	warmDataDays      int
	coldDataDays      int
	retentionDays     int
	archiveBatchSize  int
	cleanupBatchSize  int
}

func NewLogRetentionPolicy() *LogRetentionPolicy {
	return &LogRetentionPolicy{
		tableName:        "verification_logs",
		hotDataDays:      7,
		warmDataDays:     30,
		coldDataDays:     90,
		retentionDays:    365,
		archiveBatchSize: 1000,
		cleanupBatchSize: 500,
	}
}

func (p *LogRetentionPolicy) SetHotDataDays(days int) {
	if days > 0 {
		p.hotDataDays = days
	}
}

func (p *LogRetentionPolicy) SetWarmDataDays(days int) {
	if days > 0 {
		p.warmDataDays = days
	}
}

func (p *LogRetentionPolicy) SetColdDataDays(days int) {
	if days > 0 {
		p.coldDataDays = days
	}
}

func (p *LogRetentionPolicy) SetRetentionDays(days int) {
	if days > 0 {
		p.retentionDays = days
	}
}

func (p *LogRetentionPolicy) ExecuteRetentionPolicy(ctx context.Context) error {
	if err := p.archiveHotToWarm(ctx); err != nil {
		return fmt.Errorf("hot to warm archive failed: %w", err)
	}

	if err := p.archiveWarmToCold(ctx); err != nil {
		return fmt.Errorf("warm to cold archive failed: %w", err)
	}

	if err := p.cleanupExpiredLogs(ctx); err != nil {
		return fmt.Errorf("cleanup expired logs failed: %w", err)
	}

	return nil
}

func (p *LogRetentionPolicy) archiveHotToWarm(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -p.hotDataDays)

	var count int64
	if err := database.DB.Model(&models.VerificationLog{}).
		Where("created_at < ?", cutoffDate).
		Count(&count).Error; err != nil {
		return err
	}

	if count == 0 {
		return nil
	}

	log.Printf("Moving %d hot logs to warm storage (older than %d days)", count, p.hotDataDays)

	batchCount := (int(count) + p.archiveBatchSize - 1) / p.archiveBatchSize
	for i := 0; i < batchCount; i++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		archiveTable := fmt.Sprintf("warm_verification_logs_%s",
			time.Now().Format("200601"))

		if err := p.ensurePartitionExists(archiveTable); err != nil {
			log.Printf("Failed to create warm partition: %v", err)
			continue
		}

		moveSQL := fmt.Sprintf(`
			INSERT INTO %s
			SELECT * FROM %s
			WHERE created_at < ?
			ORDER BY created_at
			LIMIT ?
		`, archiveTable, p.tableName)

		result := database.DB.Exec(moveSQL, cutoffDate, p.archiveBatchSize)
		if result.Error != nil {
			log.Printf("Failed to move hot logs: %v", result.Error)
			continue
		}

		if result.RowsAffected > 0 {
			deleteSQL := fmt.Sprintf(`
				DELETE FROM %s
				WHERE ctid IN (
					SELECT ctid FROM %s
					WHERE created_at < ?
					ORDER BY created_at
					LIMIT ?
				)
			`, p.tableName, p.tableName)

			if err := database.DB.Exec(deleteSQL, cutoffDate, result.RowsAffected).Error; err != nil {
				log.Printf("Failed to delete moved hot logs: %v", err)
			}
		}

		time.Sleep(100 * time.Millisecond)
	}

	return nil
}

func (p *LogRetentionPolicy) archiveWarmToCold(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -p.warmDataDays)

	warmTotal := int64(0)
	for month := p.hotDataDays; month < p.warmDataDays; month++ {
		archiveTable := fmt.Sprintf("warm_verification_logs_%s",
			time.Now().AddDate(0, 0, -month).Format("200601"))

		var tableCount int64
		database.DB.Table(archiveTable).
			Where("created_at < ?", cutoffDate).
			Count(&tableCount)

		warmTotal += tableCount
	}

	if warmTotal == 0 {
		return nil
	}

	log.Printf("Moving %d warm logs to cold storage (older than %d days)", warmTotal, p.warmDataDays)

	coldTable := fmt.Sprintf("cold_verification_logs_%s",
		time.Now().Format("200601"))

	if err := p.ensurePartitionExists(coldTable); err != nil {
		return fmt.Errorf("failed to create cold partition: %w", err)
	}

	for month := p.hotDataDays; month < p.warmDataDays; month++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
		}

		archiveTable := fmt.Sprintf("warm_verification_logs_%s",
			time.Now().AddDate(0, 0, -month).Format("200601"))

		moveSQL := fmt.Sprintf(`
			INSERT INTO %s
			SELECT * FROM %s
			WHERE created_at < ?
		`, coldTable, archiveTable)

		if err := database.DB.Exec(moveSQL, cutoffDate).Error; err != nil {
			log.Printf("Failed to move warm logs to cold: %v", err)
		}

		time.Sleep(50 * time.Millisecond)
	}

	return nil
}

func (p *LogRetentionPolicy) cleanupExpiredLogs(ctx context.Context) error {
	cutoffDate := time.Now().AddDate(0, 0, -p.retentionDays)

	log.Printf("Cleaning up logs older than %d days (before %s)",
		p.retentionDays, cutoffDate.Format("2006-01-02"))

	coldTables := []string{
		fmt.Sprintf("cold_verification_logs_%s", time.Now().Format("200601")),
	}

	for _, coldTable := range coldTables {
		var tableExists bool
		database.DB.Raw(`
			SELECT EXISTS (
				SELECT FROM information_schema.tables
				WHERE table_name = ?
			)
		`, coldTable).Scan(&tableExists)

		if !tableExists {
			continue
		}

		for {
			select {
			case <-ctx.Done():
				return ctx.Err()
			default:
			}

			result := database.DB.Table(coldTable).
				Where("created_at < ?", cutoffDate).
				Limit(p.cleanupBatchSize).
				Delete(nil)

			if result.Error != nil {
				log.Printf("Failed to cleanup expired logs: %v", result.Error)
				break
			}

			if result.RowsAffected == 0 {
				break
			}

			log.Printf("Cleaned up %d expired logs from %s", result.RowsAffected, coldTable)
			time.Sleep(50 * time.Millisecond)
		}
	}

	return nil
}

func (p *LogRetentionPolicy) ensurePartitionExists(tableName string) error {
	var exists bool
	database.DB.Raw(`
		SELECT EXISTS (
			SELECT FROM information_schema.tables
			WHERE table_name = ?
		)
	`, tableName).Scan(&exists)

	if !exists {
		createSQL := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s (
				LIKE %s INCLUDING ALL
			)
		`, tableName, p.tableName)

		if err := database.DB.Exec(createSQL).Error; err != nil {
			return fmt.Errorf("failed to create partition table %s: %w", tableName, err)
		}

		log.Printf("Created partition table: %s", tableName)
	}

	return nil
}

type LogRetentionStats struct {
	HotDataCount    int64     `json:"hot_data_count"`
	WarmDataCount   int64     `json:"warm_data_count"`
	ColdDataCount   int64     `json:"cold_data_count"`
	TotalCount      int64     `json:"total_count"`
	ArchivedCount   int64     `json:"archived_count"`
	CleanedCount    int64     `json:"cleaned_count"`
	LastArchivalAt  time.Time `json:"last_archival_at"`
	LastCleanupAt   time.Time `json:"last_cleanup_at"`
	RetentionDays   int       `json:"retention_days"`
}

func (p *LogRetentionPolicy) GetRetentionStats() (*LogRetentionStats, error) {
	stats := &LogRetentionStats{
		RetentionDays: p.retentionDays,
	}

	hotCutoff := time.Now().AddDate(0, 0, -p.hotDataDays)
	warmCutoff := time.Now().AddDate(0, 0, -p.warmDataDays)

	database.DB.Model(&models.VerificationLog{}).
		Where("created_at >= ?", hotCutoff).
		Count(&stats.HotDataCount)

	database.DB.Model(&models.VerificationLog{}).
		Where("created_at >= ? AND created_at < ?", warmCutoff, hotCutoff).
		Count(&stats.WarmDataCount)

	stats.TotalCount = stats.HotDataCount + stats.WarmDataCount + stats.ColdDataCount

	return stats, nil
}

type ScheduledArchivalTask struct {
	policy         *LogRetentionPolicy
	ticker         *time.Ticker
	stopChan       chan struct{}
	archiveHistory  []ArchiveRecord
}

type ArchiveRecord struct {
	ID          uint      `json:"id"`
	TableName   string    `json:"table_name"`
	RecordCount int64     `json:"record_count"`
	ArchivedAt  time.Time `json:"archived_at"`
	Duration    int64     `json:"duration_ms"`
	Status      string    `json:"status"`
	Error       string    `json:"error,omitempty"`
}

func NewScheduledArchivalTask() *ScheduledArchivalTask {
	return &ScheduledArchivalTask{
		policy:    NewLogRetentionPolicy(),
		ticker:    time.NewTicker(1 * time.Hour),
		stopChan:  make(chan struct{}),
	}
}

func (t *ScheduledArchivalTask) Start(ctx context.Context) {
	go func() {
		for {
			select {
			case <-t.ticker.C:
				t.runArchival(ctx)
			case <-t.stopChan:
				t.ticker.Stop()
				return
			case <-ctx.Done():
				t.ticker.Stop()
				return
			}
		}
	}()
}

func (t *ScheduledArchivalTask) Stop() {
	close(t.stopChan)
}

func (t *ScheduledArchivalTask) runArchival(ctx context.Context) {
	start := time.Now()

	if err := t.policy.ExecuteRetentionPolicy(ctx); err != nil {
		log.Printf("Scheduled archival failed: %v", err)
		t.archiveHistory = append(t.archiveHistory, ArchiveRecord{
			TableName:  t.policy.tableName,
			ArchivedAt: start,
			Duration:   time.Since(start).Milliseconds(),
			Status:     "failed",
			Error:      err.Error(),
		})
		return
	}

	t.archiveHistory = append(t.archiveHistory, ArchiveRecord{
		TableName:  t.policy.tableName,
		ArchivedAt: start,
		Duration:   time.Since(start).Milliseconds(),
		Status:     "success",
	})

	if len(t.archiveHistory) > 100 {
		t.archiveHistory = t.archiveHistory[len(t.archiveHistory)-100:]
	}

	log.Printf("Scheduled archival completed in %v", time.Since(start))
}

func (t *ScheduledArchivalTask) GetArchiveHistory() []ArchiveRecord {
	return t.archiveHistory
}
