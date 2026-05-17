package database

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHotColdSeparatorInit(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)
	assert.True(t, separator.enabled)
	assert.NotNil(t, separator.hotStorage)
	assert.NotNil(t, separator.coldStorage)
}

func TestHotColdSeparatorClassifyData(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	classification, err := separator.ClassifyData(nil, "users", "1", time.Now().Add(-24*time.Hour), 100)
	require.NoError(t, err)

	assert.NotNil(t, classification)
	assert.Equal(t, "users", classification.TableName)
	assert.True(t, classification.Score >= 0 && classification.Score <= 1)
}

func TestHotColdSeparatorCalculateClassificationScore(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	score := separator.calculateClassificationScore(0, 100, time.Now())
	assert.Greater(t, score, 0.5)

	score = separator.calculateClassificationScore(365, 0, time.Now().Add(-365*24*time.Hour))
	assert.Less(t, score, 0.3)
}

func TestHotColdSeparatorAddArchiveRule(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	rule := &ArchiveRule{
		TableName:     "test_table",
		Condition:     "created_at < NOW() - INTERVAL '30 days'",
		TargetTier:    "cold",
		Priority:      1,
		IsActive:      true,
		RetentionDays: 365,
	}

	err = separator.AddArchiveRule(rule)
	require.NoError(t, err)

	rules := separator.GetArchiveRules()
	assert.Greater(t, len(rules), 0)
}

func TestHotColdSeparatorDeleteArchiveRule(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	rule := &ArchiveRule{
		ID:           "test_rule_1",
		TableName:    "test_table",
		Condition:    "created_at < NOW() - INTERVAL '30 days'",
		TargetTier:   "cold",
		Priority:     1,
		IsActive:     true,
	}

	err = separator.AddArchiveRule(rule)
	require.NoError(t, err)

	err = separator.DeleteArchiveRule("test_rule_1")
	require.NoError(t, err)

	rules := separator.GetArchiveRules()
	for _, r := range rules {
		assert.NotEqual(t, "test_rule_1", r.ID)
	}
}

func TestHotColdSeparatorGetTierStats(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	stats := separator.GetTierStats()
	assert.NotNil(t, stats)
	assert.Contains(t, stats, "hot")
	assert.Contains(t, stats, "cold")
	assert.Contains(t, stats, "active_rules")
}

func TestHotColdSeparatorUpdateStorageSize(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	separator.UpdateStorageSize("hot", 1024*1024*1024)

	assert.Equal(t, int64(1024*1024*1024), separator.hotStorage.CurrentSize)
}

func TestHotColdSeparatorGetStorageUtilization(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitHotColdSeparator(nil, cfg)
	require.NoError(t, err)

	separator := GetHotColdSeparator()
	require.NotNil(t, separator)

	separator.UpdateStorageSize("hot", 50*1024*1024*1024)

	utilization := separator.GetStorageUtilization()
	assert.Contains(t, utilization, "hot")
	assert.Greater(t, utilization["hot"], 0.0)
}

func TestDataClassification(t *testing.T) {
	classification := &DataClassification{
		RecordID:     "1",
		TableName:    "users",
		AccessCount:  50,
		LastAccessed: time.Now(),
		Tier:         "hot",
		Age:          7,
		IsHot:        true,
		Score:        0.85,
	}

	assert.Equal(t, "1", classification.RecordID)
	assert.Equal(t, "users", classification.TableName)
	assert.True(t, classification.IsHot)
	assert.Equal(t, "hot", classification.Tier)
}

func TestTierMigration(t *testing.T) {
	migration := &TierMigration{
		ID:          "migration_1",
		TableName:   "users",
		RecordCount: 1000,
		FromTier:    "hot",
		ToTier:      "cold",
		Status:      "in_progress",
		StartedAt:   time.Now(),
	}

	assert.Equal(t, "migration_1", migration.ID)
	assert.Equal(t, "hot", migration.FromTier)
	assert.Equal(t, "cold", migration.ToTier)
	assert.Equal(t, "in_progress", migration.Status)
}

func TestStorageTier(t *testing.T) {
	tier := &StorageTier{
		Name:        "hot",
		Type:        "ssd",
		MaxSizeGB:   100,
		CurrentSize: 0,
		IsActive:    true,
	}

	assert.Equal(t, "hot", tier.Name)
	assert.Equal(t, "ssd", tier.Type)
	assert.Equal(t, int64(100), tier.MaxSizeGB)
	assert.True(t, tier.IsActive)
}

func TestArchiveRule(t *testing.T) {
	rule := &ArchiveRule{
		ID:            "rule_1",
		TableName:     "verifications",
		Condition:     "created_at < NOW() - INTERVAL '30 days'",
		TargetTier:    "cold",
		Priority:      1,
		IsActive:      true,
		RetentionDays: 365,
		CreatedAt:     time.Now(),
		UpdatedAt:     time.Now(),
	}

	assert.Equal(t, "rule_1", rule.ID)
	assert.Equal(t, "verifications", rule.TableName)
	assert.Equal(t, "cold", rule.TargetTier)
	assert.True(t, rule.IsActive)
}

func TestMigrationResult(t *testing.T) {
	result := &MigrationResult{
		TableName:       "users",
		TotalRecords:    10000,
		MigratedRecords: 9500,
		FailedRecords:   500,
		Threshold:       time.Now().Add(-30 * 24 * time.Hour),
		Errors:          []string{"error1", "error2"},
	}

	assert.Equal(t, "users", result.TableName)
	assert.Equal(t, int64(10000), result.TotalRecords)
	assert.Equal(t, int64(9500), result.MigratedRecords)
	assert.Equal(t, 2, len(result.Errors))
}

func TestUnifiedQueryResult(t *testing.T) {
	result := &UnifiedQueryResult{
		HotData:   []map[string]interface{}{{"id": 1}, {"id": 2}},
		ColdData:  []map[string]interface{}{{"id": 3}},
		TotalHot:  2,
		TotalCold: 1,
		Total:     3,
	}

	assert.Equal(t, 2, result.TotalHot)
	assert.Equal(t, 1, result.TotalCold)
	assert.Equal(t, 3, result.Total)
}
