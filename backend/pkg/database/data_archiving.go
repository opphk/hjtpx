package database

import (
	"fmt"
	"log"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type DataArchiver struct {
	enabled          bool
	archiveThreshold time.Duration
	archivePrefix    string
	cleanupEnabled   bool
	cleanupThreshold time.Duration
}

var archiver *DataArchiver

func InitDataArchiving(cfg *config.Config) {
	archiver = &DataArchiver{
		enabled:          cfg.Database.DataArchiving.Enabled,
		archiveThreshold: time.Duration(cfg.Database.DataArchiving.ArchiveThresholdDays) * 24 * time.Hour,
		archivePrefix:    cfg.Database.DataArchiving.ArchiveTablePrefix,
		cleanupEnabled:   cfg.Database.DataArchiving.AutoCleanupEnabled,
		cleanupThreshold: time.Duration(cfg.Database.DataArchiving.CleanupThresholdDays) * 24 * time.Hour,
	}

	if archiver.enabled {
		log.Println("Data archiving initialized")
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

	log.Printf("Archived %d records from %s to %s", count, tableName, archiveTableName)
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
		return 0, result.Error
	}

	log.Printf("Cleaned up %d records from archive %s", result.RowsAffected, archiveTableName)
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
	}, nil
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
