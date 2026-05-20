package database

import (
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type DataArchivalPolicy struct {
	TableName           string
	ArchiveAfterDays   int
	DeleteAfterDays    int
	BatchSize          int
	Condition          string
	LastRun            time.Time
	RecordsArchived    int64
	RecordsDeleted     int64
}

type DataArchivalService struct {
	db      *gorm.DB
	config  *config.DataArchivingConfig
	policies []DataArchivalPolicy
}

func NewDataArchivalService(db *gorm.DB, cfg *config.DataArchivingConfig) *DataArchivalService {
	return &DataArchivalService{
		db:     db,
		config: cfg,
		policies: []DataArchivalPolicy{
			{
				TableName:         "verification_logs",
				ArchiveAfterDays: 30,
				DeleteAfterDays:  365,
				BatchSize:        10000,
				Condition:        "status = 'success' AND created_at < NOW() - INTERVAL '30 days'",
			},
			{
				TableName:         "admin_login_logs",
				ArchiveAfterDays: 90,
				DeleteAfterDays:  730,
				BatchSize:        5000,
				Condition:        "created_at < NOW() - INTERVAL '90 days'",
			},
			{
				TableName:         "behavior_data",
				ArchiveAfterDays: 14,
				DeleteAfterDays:  180,
				BatchSize:        20000,
				Condition:        "timestamp < NOW() - INTERVAL '14 days'",
			},
			{
				TableName:         "ab_test_events",
				ArchiveAfterDays: 180,
				DeleteAfterDays:  730,
				BatchSize:        15000,
				Condition:        "timestamp < NOW() - INTERVAL '180 days'",
			},
		},
	}
}

func (s *DataArchivalService) RunArchival() error {
	if !s.config.Enabled {
		return fmt.Errorf("data archival is disabled")
	}

	for _, policy := range s.policies {
		if err := s.archiveTable(policy); err != nil {
			return fmt.Errorf("failed to archive %s: %w", policy.TableName, err)
		}
	}

	return nil
}

func (s *DataArchivalService) archiveTable(policy DataArchivalPolicy) error {
	archiveTableName := s.config.ArchiveTablePrefix + policy.TableName

	if err := s.ensureArchiveTableExists(policy.TableName, archiveTableName); err != nil {
		return err
	}

	countQuery := fmt.Sprintf(`
		SELECT COUNT(*) FROM %s
		WHERE %s
	`, policy.TableName, policy.Condition)

	var count int64
	if err := s.db.Raw(countQuery).Scan(&count).Error; err != nil {
		return fmt.Errorf("failed to count records: %w", err)
	}

	if count == 0 {
		return nil
	}

	batchQuery := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s
		WHERE %s
		LIMIT %d
	`, archiveTableName, policy.TableName, policy.Condition, policy.BatchSize)

	result := s.db.Exec(batchQuery)
	if result.Error != nil {
		return fmt.Errorf("failed to archive batch: %w", result.Error)
	}

	policy.RecordsArchived += result.RowsAffected
	policy.LastRun = time.Now()

	return nil
}

func (s *DataArchivalService) ensureArchiveTableExists(sourceTable, archiveTable string) error {
	var exists bool
	query := `
		SELECT EXISTS (
			SELECT FROM pg_tables
			WHERE schemaname = 'public'
			AND tablename = ?
		)
	`
	if err := s.db.Raw(query, archiveTable).Scan(&exists).Error; err != nil {
		return fmt.Errorf("failed to check archive table: %w", err)
	}

	if !exists {
		createQuery := fmt.Sprintf(`
			CREATE TABLE %s (
				LIKE %s INCLUDING ALL
			)
		`, archiveTable, sourceTable)

		if err := s.db.Exec(createQuery).Error; err != nil {
			return fmt.Errorf("failed to create archive table: %w", err)
		}
	}

	return nil
}

func (s *DataArchivalService) DeleteArchivedData() error {
	if !s.config.AutoCleanupEnabled {
		return nil
	}

	for _, policy := range s.policies {
		if err := s.deleteOldData(policy); err != nil {
			return fmt.Errorf("failed to delete from %s: %w", policy.TableName, err)
		}
	}

	return nil
}

func (s *DataArchivalService) deleteOldData(policy DataArchivalPolicy) error {
	deleteThreshold := time.Now().AddDate(0, 0, -policy.DeleteAfterDays)

	deleteQuery := fmt.Sprintf(`
		DELETE FROM %s
		WHERE created_at < ?
		LIMIT %d
	`, policy.TableName, policy.BatchSize)

	for {
		result := s.db.Exec(deleteQuery, deleteThreshold)
		if result.Error != nil {
			return fmt.Errorf("failed to delete batch: %w", result.Error)
		}

		if result.RowsAffected == 0 {
			break
		}

		policy.RecordsDeleted += result.RowsAffected

		time.Sleep(time.Millisecond * 100)
	}

	return nil
}

func (s *DataArchivalService) GetArchiveStats() (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	for _, policy := range s.policies {
		archiveTableName := s.config.ArchiveTablePrefix + policy.TableName

		var sourceCount int64
		if err := s.db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", policy.TableName)).Scan(&sourceCount).Error; err != nil {
			sourceCount = -1
		}

		var archiveCount int64
		if err := s.db.Raw(fmt.Sprintf("SELECT COUNT(*) FROM %s", archiveTableName)).Scan(&archiveCount).Error; err != nil {
			archiveCount = -1
		}

		stats[policy.TableName] = map[string]interface{}{
			"source_records":    sourceCount,
			"archive_records":   archiveCount,
			"archive_after_days": policy.ArchiveAfterDays,
			"delete_after_days": policy.DeleteAfterDays,
			"last_run":          policy.LastRun,
			"records_archived":   policy.RecordsArchived,
			"records_deleted":   policy.RecordsDeleted,
		}
	}

	return stats, nil
}

func (s *DataArchivalService) AddPolicy(policy DataArchivalPolicy) {
	s.policies = append(s.policies, policy)
}

func (s *DataArchivalService) RemovePolicy(tableName string) {
	for i, policy := range s.policies {
		if policy.TableName == tableName {
			s.policies = append(s.policies[:i], s.policies[i+1:]...)
			return
		}
	}
}

func (s *DataArchivalService) UpdatePolicy(tableName string, updateFunc func(*DataArchivalPolicy)) {
	for i := range s.policies {
		if s.policies[i].TableName == tableName {
			updateFunc(&s.policies[i])
			return
		}
	}
}
