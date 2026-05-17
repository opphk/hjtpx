package database

import (
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestTablePartitionerInit(t *testing.T) {
	cfg := &config.Config{}

	err := InitTablePartitioner(nil, cfg)
	require.NoError(t, err)

	partitioner := GetTablePartitioner()
	require.NotNil(t, partitioner)
	assert.True(t, partitioner.enabled)
}

func TestGetTablePartitioner(t *testing.T) {
	cfg := &config.Config{}

	err := InitTablePartitioner(nil, cfg)
	require.NoError(t, err)

	partitioner := GetTablePartitioner()
	require.NotNil(t, partitioner)
	assert.NotNil(t, partitioner.timePartitioner)
	assert.NotNil(t, partitioner.appPartitioner)
}

func TestHashAppID(t *testing.T) {
	hash1 := hashAppID("app1", 8)
	hash2 := hashAppID("app2", 8)
	hash3 := hashAppID("app1", 8)

	assert.GreaterOrEqual(t, hash1, 0)
	assert.Less(t, hash1, 8)
	assert.GreaterOrEqual(t, hash2, 0)
	assert.Less(t, hash2, 8)
	assert.Equal(t, hash1, hash3)
	assert.NotEqual(t, hash1, hash2)
}

func TestPartitionConfig(t *testing.T) {
	cfg := &config.Config{}

	err := InitTablePartitioner(nil, cfg)
	require.NoError(t, err)

	partitioner := GetTablePartitioner()
	require.NotNil(t, partitioner)

	config := PartitionConfig{
		Strategy:         "daily",
		Unit:            "day",
		RetentionDays:   30,
		PreCreateDays:   7,
		AutoArchive:     true,
		ArchiveThreshold: 100000,
		PartitionNaming: "p_{table}_{date}",
	}

	partitioner.SetPartitionConfig(config)

	retrievedConfig := partitioner.GetPartitionConfig()
	assert.Equal(t, "daily", retrievedConfig.Strategy)
	assert.Equal(t, 30, retrievedConfig.RetentionDays)
}

func TestGetParentTable(t *testing.T) {
	testCases := []struct {
		partitionName string
		expected     string
	}{
		{"users_20240101", "users"},
		{"verifications_20240115", "verifications"},
		{"logs", "logs"},
	}

	for _, tc := range testCases {
		result := getParentTable(tc.partitionName)
		assert.Equal(t, tc.expected, result, "Partition: %s", tc.partitionName)
	}
}

func TestAppShardInfo(t *testing.T) {
	info := &AppShardInfo{
		AppID:     "test_app",
		ShardID:   3,
		DBName:    "app_shard_3",
		IsActive:  true,
		CreatedAt: time.Now(),
	}

	assert.Equal(t, "test_app", info.AppID)
	assert.Equal(t, 3, info.ShardID)
	assert.Equal(t, "app_shard_3", info.DBName)
	assert.True(t, info.IsActive)
}

func TestPartitionInfo(t *testing.T) {
	info := &PartitionInfo{
		TableName:     "users",
		PartitionName: "users_20240101",
		RangeStart:    time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		RangeEnd:      time.Date(2024, 1, 2, 0, 0, 0, 0, time.UTC),
		RowCount:      1000,
		SizeBytes:     1024000,
		IsActive:      true,
		IsArchived:    false,
	}

	assert.Equal(t, "users", info.TableName)
	assert.Equal(t, "users_20240101", info.PartitionName)
	assert.Equal(t, int64(1000), info.RowCount)
	assert.False(t, info.IsArchived)
}

func TestShardConfig(t *testing.T) {
	config := &ShardConfig{
		TotalShards:         8,
		ShardingKey:        "app_id",
		DBPrefix:           "app_shard",
		AppIDs:             []string{"app1", "app2"},
		AutoCreate:         true,
		ReplicationFactor:  3,
	}

	assert.Equal(t, 8, config.TotalShards)
	assert.Equal(t, "app_id", config.ShardingKey)
	assert.Equal(t, 2, len(config.AppIDs))
	assert.True(t, config.AutoCreate)
}

func TestPartitionBound(t *testing.T) {
	bounds := []PartitionBound{
		{Value: 0, BoundType: "min"},
		{Value: 100, BoundType: "mid"},
		{Value: 200, BoundType: "max"},
	}

	assert.Equal(t, 3, len(bounds))
	assert.Equal(t, int64(0), bounds[0].Value)
	assert.Equal(t, int64(100), bounds[1].Value)
}
