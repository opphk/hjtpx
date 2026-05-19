package migration

import (
	"fmt"
	"strings"
	"time"

	"gorm.io/gorm"
)

type IndexMigration struct {
	ID            uint      `gorm:"primarykey" json:"id"`
	IndexName     string    `gorm:"uniqueIndex;size:255;not null" json:"index_name"`
	TargetTable   string    `gorm:"size:255;not null" json:"table_name"`
	Columns       string    `gorm:"type:text;not null" json:"columns"`
	IsUnique      bool      `gorm:"default:false" json:"is_unique"`
	IsPartial     bool      `gorm:"default:false" json:"is_partial"`
	WhereClause   string    `gorm:"type:text" json:"where_clause"`
	Description   string    `gorm:"type:text" json:"description"`
	CreatedAt     time.Time `json:"created_at"`
	AppliedAt     *time.Time `json:"applied_at"`
	Status        string    `gorm:"size:50;default:'pending'" json:"status"`
}

func (IndexMigration) TableName() string {
	return "index_migrations"
}

type IndexMigrationManager struct {
	db *gorm.DB
}

func NewIndexMigrationManager(db *gorm.DB) *IndexMigrationManager {
	return &IndexMigrationManager{db: db}
}

func (m *IndexMigrationManager) AutoMigrate() error {
	return m.db.AutoMigrate(&IndexMigration{})
}

func (m *IndexMigrationManager) RegisterIndexes(indexes []IndexMigration) error {
	for _, idx := range indexes {
		exists, err := m.IndexExists(idx.IndexName)
		if err != nil {
			return fmt.Errorf("failed to check index existence: %w", err)
		}

		if !exists {
			if err := m.db.Create(&idx).Error; err != nil {
				if !strings.Contains(err.Error(), "duplicate") {
					return fmt.Errorf("failed to register index %s: %w", idx.IndexName, err)
				}
			}
		}
	}

	return nil
}

func (m *IndexMigrationManager) IndexExists(indexName string) (bool, error) {
	var count int64
	err := m.db.Model(&IndexMigration{}).Where("index_name = ?", indexName).Count(&count).Error
	return count > 0, err
}

func (m *IndexMigrationManager) ApplyPendingMigrations() error {
	var pending []IndexMigration
	if err := m.db.Where("status = ?", "pending").Find(&pending).Error; err != nil {
		return fmt.Errorf("failed to fetch pending migrations: %w", err)
	}

	for _, migration := range pending {
		if err := m.applyMigration(&migration); err != nil {
			return fmt.Errorf("failed to apply migration %s: %w", migration.IndexName, err)
		}
	}

	return nil
}

func (m *IndexMigrationManager) applyMigration(migration *IndexMigration) error {
	createSQL := m.buildCreateIndexSQL(migration)

	if err := m.db.Exec(createSQL).Error; err != nil {
		migration.Status = "failed"
		m.db.Save(migration)
		return fmt.Errorf("failed to execute: %w", err)
	}

	now := time.Now()
	migration.AppliedAt = &now
	migration.Status = "applied"
	return m.db.Save(migration).Error
}

func (m *IndexMigrationManager) buildCreateIndexSQL(migration *IndexMigration) string {
	var sql strings.Builder
	sql.WriteString("CREATE ")

	if migration.IsUnique {
		sql.WriteString("UNIQUE ")
	}

	sql.WriteString("INDEX CONCURRENTLY ")
	sql.WriteString(migration.IndexName)
	sql.WriteString(" ON ")
	sql.WriteString(migration.TargetTable)
	sql.WriteString(" (")

	columns := strings.Split(migration.Columns, ",")
	for i, col := range columns {
		if i > 0 {
			sql.WriteString(", ")
		}
		sql.WriteString(strings.TrimSpace(col))
	}
	sql.WriteString(")")

	if migration.IsPartial && migration.WhereClause != "" {
		sql.WriteString(" WHERE ")
		sql.WriteString(migration.WhereClause)
	}

	return sql.String()
}

func (m *IndexMigrationManager) RollbackMigration(indexName string) error {
	var migration IndexMigration
	if err := m.db.Where("index_name = ?", indexName).First(&migration).Error; err != nil {
		return fmt.Errorf("migration not found: %w", err)
	}

	dropSQL := fmt.Sprintf("DROP INDEX CONCURRENTLY IF EXISTS %s", migration.IndexName)
	if err := m.db.Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("failed to drop index: %w", err)
	}

	now := time.Now()
	migration.AppliedAt = &now
	migration.Status = "rolled_back"
	return m.db.Save(&migration).Error
}

func (m *IndexMigrationManager) GetMigrationStatus() ([]IndexMigration, error) {
	var migrations []IndexMigration
	err := m.db.Order("created_at DESC").Find(&migrations).Error
	return migrations, err
}

func (m *IndexMigrationManager) GetAppliedMigrations() ([]IndexMigration, error) {
	var migrations []IndexMigration
	err := m.db.Where("status = ?", "applied").Order("applied_at DESC").Find(&migrations).Error
	return migrations, err
}

func (m *IndexMigrationManager) GetPendingMigrations() ([]IndexMigration, error) {
	var migrations []IndexMigration
	err := m.db.Where("status = ?", "pending").Order("created_at ASC").Find(&migrations).Error
	return migrations, err
}

func (m *IndexMigrationManager) GetFailedMigrations() ([]IndexMigration, error) {
	var migrations []IndexMigration
	err := m.db.Where("status = ?", "failed").Order("created_at DESC").Find(&migrations).Error
	return migrations, err
}

func (m *IndexMigrationManager) CheckDatabaseIndexes() ([]map[string]interface{}, error) {
	var indexes []map[string]interface{}
	err := m.db.Raw(`
		SELECT 
			indexname,
			tablename,
			indexdef
		FROM pg_indexes
		WHERE schemaname = 'public'
		ORDER BY tablename, indexname
	`).Scan(&indexes).Error

	return indexes, err
}

func (m *IndexMigrationManager) SyncDatabaseIndexes() error {
	dbIndexes, err := m.CheckDatabaseIndexes()
	if err != nil {
		return fmt.Errorf("failed to check database indexes: %w", err)
	}

	dbIndexMap := make(map[string]bool)
	for _, idx := range dbIndexes {
		if name, ok := idx["indexname"].(string); ok {
			dbIndexMap[name] = true
		}
	}

	var registeredIndexes []IndexMigration
	if err := m.db.Where("status = ?", "applied").Find(&registeredIndexes).Error; err != nil {
		return fmt.Errorf("failed to fetch registered indexes: %w", err)
	}

	for _, idx := range registeredIndexes {
		if _, exists := dbIndexMap[idx.IndexName]; !exists {
			idx.Status = "missing"
			m.db.Save(&idx)
		}
	}

	return nil
}

func GetDefaultIndexes() []IndexMigration {
	now := time.Now()
	return []IndexMigration{
		{
			IndexName:   "idx_users_email",
			TargetTable: "users",
			Columns:     "email",
			IsUnique:    true,
			Description: "User email unique index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_users_username",
			TargetTable: "users",
			Columns:     "username",
			IsUnique:    false,
			Description: "User username index for login",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_users_status_created",
			TargetTable: "users",
			Columns:     "status, created_at",
			IsUnique:    false,
			Description: "User status and creation time composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_applications_app_key",
			TargetTable: "applications",
			Columns:     "app_key",
			IsUnique:    true,
			Description: "Application app_key unique index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_applications_user_active",
			TargetTable: "applications",
			Columns:     "user_id, is_active",
			IsUnique:    false,
			Description: "Application user and active status composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_verifications_session",
			TargetTable: "verifications",
			Columns:     "session_id",
			IsUnique:    true,
			Description: "Verification session ID unique index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_verifications_status_created",
			TargetTable: "verifications",
			Columns:     "status, created_at",
			IsUnique:    false,
			Description: "Verification status and time composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_verifications_app_status",
			TargetTable: "verifications",
			Columns:     "application_id, status",
			IsUnique:    false,
			Description: "Verification application and status composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_blacklist_value_type",
			TargetTable: "blacklist",
			Columns:     "blacklisted_value, blacklist_type",
			IsUnique:    true,
			IsPartial:   true,
			WhereClause: "is_active = true",
			Description: "Active blacklist entries composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_logs_session_created",
			TargetTable: "verification_logs",
			Columns:     "session_id, created_at",
			IsUnique:    false,
			Description: "Verification logs session and time composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_logs_app_status_created",
			TargetTable: "verification_logs",
			Columns:     "application_id, status, created_at",
			IsUnique:    false,
			Description: "Verification logs application, status and time index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_captcha_session_id",
			TargetTable: "captcha_sessions",
			Columns:     "session_id",
			IsUnique:    true,
			Description: "Captcha session ID unique index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_captcha_status_expires",
			TargetTable: "captcha_sessions",
			Columns:     "status, expired_at",
			IsUnique:    false,
			IsPartial:   true,
			WhereClause: "status = 'pending'",
			Description: "Pending captcha sessions expiration index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_risk_session_created",
			TargetTable: "risk_logs",
			Columns:     "session_id, created_at",
			IsUnique:    false,
			Description: "Risk logs session and time composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_risk_level_created",
			TargetTable: "risk_logs",
			Columns:     "risk_level, created_at",
			IsUnique:    false,
			Description: "Risk logs level and time composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_admin_login_user_created",
			TargetTable: "admin_login_logs",
			Columns:     "username, created_at",
			IsUnique:    false,
			Description: "Admin login logs user and time index",
			CreatedAt:   now,
			Status:      "pending",
		},
		{
			IndexName:   "idx_configs_category_key",
			TargetTable: "configs",
			Columns:     "category, config_key",
			IsUnique:    true,
			Description: "Config category and key unique composite index",
			CreatedAt:   now,
			Status:      "pending",
		},
	}
}
