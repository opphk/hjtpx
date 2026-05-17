package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type HotColdSeparator struct {
	db           *gorm.DB
	config       *config.Config
	hotStorage   *StorageTier
	coldStorage  *StorageTier
	archiveRules []ArchiveRule
	enabled      bool
	mu           sync.RWMutex
}

type StorageTier struct {
	Name       string `json:"name"`
	Type       string `json:"type"`
	Path       string `json:"path"`
	MaxSizeGB  int64  `json:"max_size_gb"`
	CurrentSize int64  `json:"current_size"`
	IsActive   bool   `json:"is_active"`
}

type ArchiveRule struct {
	ID            string    `json:"id"`
	TableName     string    `json:"table_name"`
	Condition     string    `json:"condition"`
	TargetTier    string    `json:"target_tier"`
	Priority      int       `json:"priority"`
	IsActive      bool      `json:"is_active"`
	RetentionDays int       `json:"retention_days"`
	CreatedAt     time.Time `json:"created_at"`
	UpdatedAt     time.Time `json:"updated_at"`
}

type DataClassification struct {
	RecordID     string    `json:"record_id"`
	TableName    string    `json:"table_name"`
	AccessCount  int       `json:"access_count"`
	LastAccessed time.Time `json:"last_accessed"`
	Tier         string    `json:"tier"`
	Age          int       `json:"age_days"`
	IsHot        bool      `json:"is_hot"`
	Score        float64   `json:"classification_score"`
}

type TierMigration struct {
	ID           string    `json:"id"`
	TableName    string    `json:"table_name"`
	RecordCount  int64     `json:"record_count"`
	FromTier     string    `json:"from_tier"`
	ToTier       string    `json:"to_tier"`
	Status       string    `json:"status"`
	StartedAt    time.Time `json:"started_at"`
	CompletedAt  time.Time `json:"completed_at,omitempty"`
	Error        string    `json:"error,omitempty"`
}

var hotColdSeparator *HotColdSeparator

func InitHotColdSeparator(db *gorm.DB, cfg *config.Config) error {
	hotColdSeparator = &HotColdSeparator{
		db:    db,
		config: cfg,
		hotStorage: &StorageTier{
			Name:       "hot",
			Type:       "ssd",
			MaxSizeGB:  100,
			CurrentSize: 0,
			IsActive:   true,
		},
		coldStorage: &StorageTier{
			Name:       "cold",
			Type:       "hdd",
			MaxSizeGB:  500,
			CurrentSize: 0,
			IsActive:   true,
		},
		archiveRules: make([]ArchiveRule, 0),
		enabled:     cfg.Database.DataArchiving.Enabled,
	}

	if hotColdSeparator.enabled {
		hotColdSeparator.loadArchiveRules()
		go hotColdSeparator.startAutoClassification()
		log.Println("Hot-cold data separator initialized")
	}

	return nil
}

func GetHotColdSeparator() *HotColdSeparator {
	return hotColdSeparator
}

func (s *HotColdSeparator) ClassifyData(ctx context.Context, tableName string, recordID string, lastAccessed time.Time, accessCount int) (*DataClassification, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	now := time.Now()
	age := int(now.Sub(lastAccessed).Hours() / 24)

	score := s.calculateClassificationScore(age, accessCount, lastAccessed)

	isHot := score > 0.7
	tier := "hot"
	if !isHot {
		tier = "cold"
	}

	classification := &DataClassification{
		RecordID:     recordID,
		TableName:    tableName,
		AccessCount:  accessCount,
		LastAccessed: lastAccessed,
		Tier:         tier,
		Age:          age,
		IsHot:        isHot,
		Score:        score,
	}

	return classification, nil
}

func (s *HotColdSeparator) calculateClassificationScore(ageDays int, accessCount int, lastAccessed time.Time) float64 {
	ageScore := 1.0 - (float64(ageDays) / 365.0)
	if ageScore < 0 {
		ageScore = 0
	}

	recencyScore := 1.0
	daysSinceAccess := time.Since(lastAccessed).Hours() / 24
	if daysSinceAccess > 30 {
		recencyScore = 0.1
	} else if daysSinceAccess > 7 {
		recencyScore = 0.3
	} else if daysSinceAccess > 1 {
		recencyScore = 0.7
	}

	accessScore := float64(accessCount) / 100.0
	if accessScore > 1.0 {
		accessScore = 1.0
	}

	score := (ageScore * 0.3) + (recencyScore * 0.5) + (accessScore * 0.2)

	return score
}

func (s *HotColdSeparator) MigrateToCold(ctx context.Context, tableName string, recordIDs []interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if len(recordIDs) == 0 {
		return nil
	}

	migration := &TierMigration{
		ID:          fmt.Sprintf("migration_%d", time.Now().UnixNano()),
		TableName:   tableName,
		RecordCount: int64(len(recordIDs)),
		FromTier:    "hot",
		ToTier:      "cold",
		Status:      "in_progress",
		StartedAt:   time.Now(),
	}

	log.Printf("Starting migration to cold storage: %s (%d records)", tableName, len(recordIDs))

	archiveTableName := fmt.Sprintf("archive_%s", tableName)

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s WHERE id IN ?
	`, archiveTableName, tableName)

	if err := s.db.WithContext(ctx).Exec(insertSQL, recordIDs).Error; err != nil {
		migration.Status = "failed"
		migration.Error = err.Error()
		return fmt.Errorf("failed to archive records: %w", err)
	}

	deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE id IN ?`, tableName)
	if err := s.db.WithContext(ctx).Exec(deleteSQL, recordIDs).Error; err != nil {
		migration.Status = "failed"
		migration.Error = err.Error()
		return fmt.Errorf("failed to delete from source: %w", err)
	}

	migration.Status = "completed"
	migration.CompletedAt = time.Now()

	log.Printf("Migration to cold storage completed: %s (%d records)", tableName, len(recordIDs))
	return nil
}

func (s *HotColdSeparator) MigrateToHot(ctx context.Context, tableName string, recordIDs []interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return fmt.Errorf("database connection not available")
	}

	if len(recordIDs) == 0 {
		return nil
	}

	log.Printf("Restoring %d records from cold storage to %s", len(recordIDs), tableName)

	archiveTableName := fmt.Sprintf("archive_%s", tableName)

	insertSQL := fmt.Sprintf(`
		INSERT INTO %s
		SELECT * FROM %s WHERE id IN ?
	`, tableName, archiveTableName)

	if err := s.db.WithContext(ctx).Exec(insertSQL, recordIDs).Error; err != nil {
		return fmt.Errorf("failed to restore records: %w", err)
	}

	deleteSQL := fmt.Sprintf(`DELETE FROM %s WHERE id IN ?`, archiveTableName)
	if err := s.db.WithContext(ctx).Exec(deleteSQL, recordIDs).Error; err != nil {
		log.Printf("Warning: failed to delete from archive: %v", err)
	}

	log.Printf("Restored %d records from cold storage to %s", len(recordIDs), tableName)
	return nil
}

func (s *HotColdSeparator) AddArchiveRule(rule *ArchiveRule) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if rule.ID == "" {
		rule.ID = fmt.Sprintf("rule_%d", time.Now().UnixNano())
	}
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	s.archiveRules = append(s.archiveRules, *rule)

	log.Printf("Added archive rule for table: %s", rule.TableName)
	return nil
}

func (s *HotColdSeparator) GetArchiveRules() []ArchiveRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]ArchiveRule, len(s.archiveRules))
	copy(rules, s.archiveRules)
	return rules
}

func (s *HotColdSeparator) DeleteArchiveRule(ruleID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i, rule := range s.archiveRules {
		if rule.ID == ruleID {
			s.archiveRules = append(s.archiveRules[:i], s.archiveRules[i+1:]...)
			log.Printf("Deleted archive rule: %s", ruleID)
			return nil
		}
	}

	return fmt.Errorf("rule not found: %s", ruleID)
}

func (s *HotColdSeparator) EvaluateAndMigrate(ctx context.Context, tableName string, threshold time.Time) (*MigrationResult, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	type countResult struct {
		Count int64 `gorm:"column:count"`
	}

	var count countResult
	query := fmt.Sprintf(`SELECT COUNT(*) as count FROM %s WHERE updated_at < ?`, tableName)
	if err := s.db.WithContext(ctx).Raw(query, threshold).Scan(&count).Error; err != nil {
		return nil, fmt.Errorf("failed to count records: %w", err)
	}

	result := &MigrationResult{
		TableName:    tableName,
		TotalRecords: count.Count,
		Threshold:    threshold,
	}

	if count.Count == 0 {
		return result, nil
	}

	batchSize := int64(1000)
	for i := int64(0); i < count.Count; i += batchSize {
		var ids []interface{}
		idQuery := fmt.Sprintf(`SELECT id FROM %s WHERE updated_at < ? LIMIT ? OFFSET ?`, tableName)
		if err := s.db.WithContext(ctx).Raw(idQuery, threshold, batchSize, i).Scan(&ids).Error; err != nil {
			result.Errors = append(result.Errors, err.Error())
			continue
		}

		if err := s.MigrateToCold(ctx, tableName, ids); err != nil {
			result.Errors = append(result.Errors, err.Error())
			result.FailedRecords += int64(len(ids))
		} else {
			result.MigratedRecords += int64(len(ids))
		}
	}

	return result, nil
}

type MigrationResult struct {
	TableName        string    `json:"table_name"`
	TotalRecords     int64     `json:"total_records"`
	MigratedRecords  int64     `json:"migrated_records"`
	FailedRecords    int64     `json:"failed_records"`
	Threshold        time.Time `json:"threshold"`
	Errors           []string  `json:"errors,omitempty"`
}

func (s *HotColdSeparator) GetTierStats() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	return map[string]interface{}{
		"hot": map[string]interface{}{
			"name":         s.hotStorage.Name,
			"type":         s.hotStorage.Type,
			"current_size": s.hotStorage.CurrentSize,
			"max_size":     s.hotStorage.MaxSizeGB * 1024 * 1024 * 1024,
			"is_active":    s.hotStorage.IsActive,
		},
		"cold": map[string]interface{}{
			"name":         s.coldStorage.Name,
			"type":         s.coldStorage.Type,
			"current_size": s.coldStorage.CurrentSize,
			"max_size":     s.coldStorage.MaxSizeGB * 1024 * 1024 * 1024,
			"is_active":    s.coldStorage.IsActive,
		},
		"active_rules": len(s.archiveRules),
	}
}

func (s *HotColdSeparator) loadArchiveRules() {
	s.mu.Lock()
	defer s.mu.Unlock()

	defaultRules := []ArchiveRule{
		{
			ID:            "rule_verification_30d",
			TableName:     "verifications",
			Condition:     "created_at < NOW() - INTERVAL '30 days'",
			TargetTier:    "cold",
			Priority:      1,
			IsActive:      true,
			RetentionDays: 365,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            "rule_logs_7d",
			TableName:     "verification_logs",
			Condition:     "created_at < NOW() - INTERVAL '7 days'",
			TargetTier:    "cold",
			Priority:      2,
			IsActive:      true,
			RetentionDays: 90,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
		{
			ID:            "rule_behavior_14d",
			TableName:     "behavior_data",
			Condition:     "created_at < NOW() - INTERVAL '14 days'",
			TargetTier:    "cold",
			Priority:      3,
			IsActive:      true,
			RetentionDays: 180,
			CreatedAt:     time.Now(),
			UpdatedAt:     time.Now(),
		},
	}

	s.archiveRules = defaultRules
	log.Printf("Loaded %d default archive rules", len(defaultRules))
}

func (s *HotColdSeparator) startAutoClassification() {
	ticker := time.NewTicker(6 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		s.performAutoClassification()
	}
}

func (s *HotColdSeparator) performAutoClassification() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	s.mu.RLock()
	rules := s.archiveRules
	s.mu.RUnlock()

	for _, rule := range rules {
		if !rule.IsActive {
			continue
		}

		threshold := time.Now().AddDate(0, 0, -30)
		result, err := s.EvaluateAndMigrate(ctx, rule.TableName, threshold)
		if err != nil {
			log.Printf("Auto-classification failed for %s: %v", rule.TableName, err)
		} else if result.MigratedRecords > 0 {
			log.Printf("Auto-classification completed: %s, migrated %d records",
				rule.TableName, result.MigratedRecords)
		}
	}
}

func (s *HotColdSeparator) QueryHotData(ctx context.Context, tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := s.db.WithContext(ctx).Table(tableName)

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

func (s *HotColdSeparator) QueryColdData(ctx context.Context, tableName string, conditions map[string]interface{}, limit, offset int) ([]map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	archiveTableName := fmt.Sprintf("archive_%s", tableName)

	query := s.db.WithContext(ctx).Table(archiveTableName)

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

func (s *HotColdSeparator) UnifiedQuery(ctx context.Context, tableName string, conditions map[string]interface{}, limit, offset int, includeCold bool) (*UnifiedQueryResult, error) {
	result := &UnifiedQueryResult{
		HotData:  make([]map[string]interface{}, 0),
		ColdData: make([]map[string]interface{}, 0),
	}

	hotResults, err := s.QueryHotData(ctx, tableName, conditions, limit, offset)
	if err != nil {
		return nil, fmt.Errorf("failed to query hot data: %w", err)
	}
	result.HotData = hotResults
	result.TotalHot = len(hotResults)

	if includeCold {
		coldResults, err := s.QueryColdData(ctx, tableName, conditions, limit, offset)
		if err != nil {
			return nil, fmt.Errorf("failed to query cold data: %w", err)
		}
		result.ColdData = coldResults
		result.TotalCold = len(coldResults)
	}

	result.Total = result.TotalHot + result.TotalCold
	return result, nil
}

type UnifiedQueryResult struct {
	HotData   []map[string]interface{} `json:"hot_data"`
	ColdData  []map[string]interface{} `json:"cold_data"`
	TotalHot  int                     `json:"total_hot"`
	TotalCold int                     `json:"total_cold"`
	Total     int                     `json:"total"`
}

func (s *HotColdSeparator) UpdateStorageSize(tier string, sizeBytes int64) {
	s.mu.Lock()
	defer s.mu.Unlock()

	switch tier {
	case "hot":
		s.hotStorage.CurrentSize = sizeBytes
	case "cold":
		s.coldStorage.CurrentSize = sizeBytes
	}
}

func (s *HotColdSeparator) GetStorageUtilization() map[string]float64 {
	s.mu.RLock()
	defer s.mu.RUnlock()

	utilization := make(map[string]float64)

	if s.hotStorage.MaxSizeGB > 0 {
		utilization["hot"] = float64(s.hotStorage.CurrentSize) / float64(s.hotStorage.MaxSizeGB*1024*1024*1024) * 100
	}

	if s.coldStorage.MaxSizeGB > 0 {
		utilization["cold"] = float64(s.coldStorage.CurrentSize) / float64(s.coldStorage.MaxSizeGB*1024*1024*1024) * 100
	}

	return utilization
}
