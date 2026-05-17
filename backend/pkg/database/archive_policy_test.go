package database

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestArchivePolicyManagerInit(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)
	assert.True(t, manager.enabled)
}

func TestArchivePolicyManagerCreatePolicy(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		Name:           "Test Policy",
		TableName:      "users",
		Condition:      "created_at < NOW() - INTERVAL '30 days'",
		TargetType:    "table",
		Priority:       1,
		RetentionDays:  365,
		BatchSize:      1000,
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)
	assert.NotEmpty(t, policy.ID)
}

func TestArchivePolicyManagerGetPolicy(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		ID:       "test_policy_1",
		Name:     "Test Policy",
		TableName: "users",
		Condition: "created_at < NOW() - INTERVAL '30 days'",
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)

	retrieved, err := manager.GetPolicy("test_policy_1")
	require.NoError(t, err)
	assert.Equal(t, "Test Policy", retrieved.Name)
}

func TestArchivePolicyManagerListPolicies(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		ID:       "test_policy_list",
		Name:     "List Test Policy",
		TableName: "users",
		Condition: "created_at < NOW() - INTERVAL '30 days'",
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)

	policies := manager.ListPolicies()
	assert.GreaterOrEqual(t, len(policies), 1)
}

func TestArchivePolicyManagerUpdatePolicy(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		ID:       "test_policy_update",
		Name:     "Original Name",
		TableName: "users",
		Condition: "created_at < NOW() - INTERVAL '30 days'",
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)

	updates := map[string]interface{}{
		"name":   "Updated Name",
		"priority": 5,
	}

	err = manager.UpdatePolicy("test_policy_update", updates)
	require.NoError(t, err)

	updated, err := manager.GetPolicy("test_policy_update")
	require.NoError(t, err)
	assert.Equal(t, "Updated Name", updated.Name)
	assert.Equal(t, 5, updated.Priority)
}

func TestArchivePolicyManagerDeletePolicy(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		ID:       "test_policy_delete",
		Name:     "Delete Test Policy",
		TableName: "users",
		Condition: "created_at < NOW() - INTERVAL '30 days'",
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)

	err = manager.DeletePolicy("test_policy_delete")
	require.NoError(t, err)

	_, err = manager.GetPolicy("test_policy_delete")
	assert.Error(t, err)
}

func TestArchivePolicyManagerValidatePolicy(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	validPolicy := &ArchivePolicy{
		TableName:     "users",
		Condition:     "created_at < NOW() - INTERVAL '30 days'",
		RetentionDays: 365,
		BatchSize:     1000,
	}

	err = manager.ValidatePolicy(validPolicy)
	require.NoError(t, err)

	invalidPolicy := &ArchivePolicy{
		TableName:     "",
		Condition:     "created_at < NOW() - INTERVAL '30 days'",
		RetentionDays: 365,
		BatchSize:     1000,
	}

	err = manager.ValidatePolicy(invalidPolicy)
	assert.Error(t, err)
}

func TestArchivePolicyManagerClonePolicy(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	original := &ArchivePolicy{
		ID:            "test_policy_clone",
		Name:          "Original Policy",
		TableName:     "users",
		Condition:     "created_at < NOW() - INTERVAL '30 days'",
		RetentionDays: 365,
		BatchSize:     1000,
	}

	err = manager.CreatePolicy(original)
	require.NoError(t, err)

	cloned, err := manager.ClonePolicy("test_policy_clone", "Cloned Policy")
	require.NoError(t, err)
	assert.NotEqual(t, "test_policy_clone", cloned.ID)
	assert.Equal(t, "Cloned Policy", cloned.Name)
}

func TestArchivePolicyManagerGetPolicyStats(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		ID:            "test_policy_stats",
		Name:          "Stats Test Policy",
		TableName:     "users",
		Condition:     "created_at < NOW() - INTERVAL '30 days'",
		TotalArchived: 1000,
		TotalRestored: 100,
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)

	stats, err := manager.GetPolicyStats("test_policy_stats")
	require.NoError(t, err)
	assert.Contains(t, stats, "policy_id")
	assert.Contains(t, stats, "total_archived")
}

func TestArchivePolicyManagerCalculateNextExecution(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	nextExec := manager.calculateNextExecution("0 2 * * *")
	assert.True(t, nextExec.After(time.Now()))

	nextExec = manager.calculateNextExecution("0 3 * * *")
	assert.True(t, nextExec.After(time.Now()))

	nextExec = manager.calculateNextExecution("invalid")
	assert.True(t, nextExec.After(time.Now()))
}

func TestArchivePolicy(t *testing.T) {
	now := time.Now()
	policy := &ArchivePolicy{
		ID:                 "test_policy",
		Name:               "Test Policy",
		TableName:          "users",
		Condition:          "created_at < NOW() - INTERVAL '30 days'",
		TargetType:         "table",
		TargetLocation:     "archive_users",
		Priority:          1,
		Schedule:          "0 2 * * *",
		IsActive:          true,
		RetentionDays:      365,
		BatchSize:         1000,
		CompressionEnabled: true,
		EncryptionEnabled:  false,
		LastExecuted:      &now,
		NextExecution:     &now,
		TotalArchived:     5000,
		TotalRestored:    100,
		CreatedAt:        now,
		UpdatedAt:        now,
	}

	assert.Equal(t, "test_policy", policy.ID)
	assert.Equal(t, "Test Policy", policy.Name)
	assert.True(t, policy.IsActive)
	assert.Equal(t, int64(5000), policy.TotalArchived)
}

func TestPolicyExecutor(t *testing.T) {
	executor := &PolicyExecutor{
		policyID:  "test_executor",
		status:   "running",
		startedAt: time.Now(),
		stats: &ExecutorStats{
			ProcessedRecords: 1000,
			SuccessCount:    950,
			FailedCount:     50,
			SkippedCount:    10,
			BytesArchived:   1024000,
			Duration:        5 * time.Minute,
		},
	}

	assert.Equal(t, "running", executor.status)
	assert.Equal(t, int64(1000), executor.stats.ProcessedRecords)
	assert.Equal(t, int64(950), executor.stats.SuccessCount)
}

func TestExecutorStats(t *testing.T) {
	stats := &ExecutorStats{
		ProcessedRecords: 5000,
		SuccessCount:    4900,
		FailedCount:     80,
		SkippedCount:   20,
		BytesArchived:   5120000,
		Duration:        10 * time.Minute,
		LastError:      "some error",
	}

	assert.Equal(t, int64(5000), stats.ProcessedRecords)
	assert.Equal(t, int64(4900), stats.SuccessCount)
	assert.Equal(t, int64(80), stats.FailedCount)
	assert.NotEmpty(t, stats.LastError)
}

func TestArchiveJob(t *testing.T) {
	now := time.Now()
	job := &ArchiveJob{
		ID:                "test_job",
		PolicyID:          "test_policy",
		Status:            "running",
		StartedAt:         now,
		CompletedAt:       now.Add(5 * time.Minute),
		Progress:          100.0,
		TotalRecords:      10000,
		ProcessedRecords:  10000,
		Error:             "",
	}

	assert.Equal(t, "test_job", job.ID)
	assert.Equal(t, "running", job.Status)
	assert.Equal(t, float64(100), job.Progress)
}

func TestArchiveJobStatus(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	manager.executors["test_job_1"] = &PolicyExecutor{
		policyID: "test_policy",
		status:   "running",
		stats:    &ExecutorStats{},
	}

	manager.executors["test_job_2"] = &PolicyExecutor{
		policyID: "test_policy",
		status:   "completed",
		stats: &ExecutorStats{
			ProcessedRecords: 1000,
		},
	}

	status, err := manager.GetJobStatus("test_job_1")
	require.NoError(t, err)
	assert.Equal(t, "running", status.Status)

	status, err = manager.GetJobStatus("test_job_2")
	require.NoError(t, err)
	assert.Equal(t, "completed", status.Status)
}

func TestArchivePolicyManagerCancelJob(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	manager.executors["test_job_cancel"] = &PolicyExecutor{
		policyID: "test_policy",
		status:   "running",
		stats:    &ExecutorStats{},
	}

	err = manager.CancelJob("test_job_cancel")
	require.NoError(t, err)

	executor := manager.executors["test_job_cancel"]
	assert.Equal(t, "cancelled", executor.status)
}

func TestArchivePolicyManagerRestoreFromArchive(t *testing.T) {
	cfg := &config.Config{
		Database: config.DatabaseConfig{
			DataArchiving: config.DataArchivingConfig{
				Enabled: true,
			},
		},
	}

	err := InitArchivePolicyManager(nil, cfg)
	require.NoError(t, err)

	manager := GetArchivePolicyManager()
	require.NotNil(t, manager)

	policy := &ArchivePolicy{
		ID:       "test_restore_policy",
		Name:     "Restore Test Policy",
		TableName: "users",
		Condition: "created_at < NOW() - INTERVAL '30 days'",
	}

	err = manager.CreatePolicy(policy)
	require.NoError(t, err)

	ctx := context.Background()
	restored, err := manager.RestoreFromArchive(ctx, "test_restore_policy", []interface{}{})
	assert.Error(t, err, "Should return error when database is not available")
	assert.Equal(t, int64(0), restored)
}

func TestArchivePolicyWithAllFields(t *testing.T) {
	now := time.Now()
	policy := &ArchivePolicy{
		ID:                  "full_policy",
		Name:                "Full Test Policy",
		TableName:           "verification_logs",
		Condition:           "created_at < NOW() - INTERVAL '7 days'",
		TargetType:          "table",
		TargetLocation:      "archive_verification_logs",
		Priority:            2,
		Schedule:            "0 3 * * *",
		IsActive:            true,
		RetentionDays:       90,
		BatchSize:           5000,
		CompressionEnabled:  true,
		EncryptionEnabled:   false,
		LastExecuted:        &now,
		TotalArchived:       10000,
		TotalRestored:      200,
		CreatedAt:          now,
		UpdatedAt:          now,
	}

	nextExec := now.Add(24 * time.Hour)
	policy.NextExecution = &nextExec

	assert.Equal(t, "full_policy", policy.ID)
	assert.Equal(t, "Full Test Policy", policy.Name)
	assert.Equal(t, "verification_logs", policy.TableName)
	assert.Equal(t, int64(10000), policy.TotalArchived)
	assert.True(t, policy.CompressionEnabled)
	assert.False(t, policy.EncryptionEnabled)
}
