package trace

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestModelUpdateService_RegisterModel(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	t.Run("Register Valid Model", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "lstm",
			VersionID:  "v1.0.0",
			ModelPath:  "/models/lstm/v1.0.0",
			Checksum:   "abc123",
			CreatedBy:  "test-user",
			Metadata: map[string]interface{}{
				"framework": "tensorflow",
				"accuracy":  0.95,
			},
		}

		version, err := service.RegisterModel(ctx, reg)
		if err != nil {
			t.Fatalf("RegisterModel failed: %v", err)
		}

		if version == nil {
			t.Fatal("Version should not be nil")
		}

		if version.VersionID != "v1.0.0" {
			t.Errorf("Expected version ID v1.0.0, got %s", version.VersionID)
		}

		if version.Status != ModelStatusStaging {
			t.Errorf("Expected status staging, got %s", version.Status)
		}

		if version.ModelType != "lstm" {
			t.Errorf("Expected model type lstm, got %s", version.ModelType)
		}
	})

	t.Run("Register Duplicate Model", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "lstm",
			VersionID:  "v1.0.0",
			ModelPath:  "/models/lstm/v1.0.0",
			Checksum:   "abc123",
			CreatedBy:  "test-user",
		}

		_, err := service.RegisterModel(ctx, reg)
		if err == nil {
			t.Error("Expected error for duplicate registration")
		}
	})

	t.Run("Register Model Missing Fields", func(t *testing.T) {
		testCases := []ModelRegistration{
			{ModelType: "", VersionID: "v1.0.0", ModelPath: "/path"},
			{ModelType: "lstm", VersionID: "", ModelPath: "/path"},
			{ModelType: "lstm", VersionID: "v1.0.0", ModelPath: ""},
		}

		for i, reg := range testCases {
			_, err := service.RegisterModel(ctx, reg)
			if err == nil {
				t.Errorf("Test case %d: Expected error for missing fields", i)
			}
		}
	})
}

func TestModelUpdateService_ActivateVersion(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	reg := ModelRegistration{
		ModelType: "transformer",
		VersionID:  "v1.0.0",
		ModelPath:  "/models/transformer/v1.0.0",
		Checksum:   "def456",
		CreatedBy:  "test-user",
	}

	_, err := service.RegisterModel(ctx, reg)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	t.Run("Activate Existing Version", func(t *testing.T) {
		result, err := service.ActivateVersion(ctx, "transformer", "v1.0.0")
		if err != nil {
			t.Fatalf("ActivateVersion failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Activation should succeed: %s", result.Message)
		}

		if result.NewVersionID != "v1.0.0" {
			t.Errorf("Expected new version v1.0.0, got %s", result.NewVersionID)
		}
	})

	t.Run("Activate Non-existent Version", func(t *testing.T) {
		_, err := service.ActivateVersion(ctx, "transformer", "v999.0.0")
		if err == nil {
			t.Error("Expected error for non-existent version")
		}
	})

	t.Run("Double Activation", func(t *testing.T) {
		reg2 := ModelRegistration{
			ModelType: "transformer",
			VersionID:  "v1.1.0",
			ModelPath:  "/models/transformer/v1.1.0",
			Checksum:   "ghi789",
			CreatedBy:  "test-user",
		}

		_, err := service.RegisterModel(ctx, reg2)
		if err != nil {
			t.Fatalf("Failed to register second version: %v", err)
		}

		result, err := service.ActivateVersion(ctx, "transformer", "v1.1.0")
		if err != nil {
			t.Fatalf("ActivateVersion failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Second activation should succeed: %s", result.Message)
		}

		if result.OldVersionID != "v1.0.0" {
			t.Errorf("Expected old version v1.0.0, got %s", result.OldVersionID)
		}
	})
}

func TestModelUpdateService_GetVersion(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	reg := ModelRegistration{
		ModelType: "lstm",
		VersionID:  "test-v1.0",
		ModelPath:  "/models/lstm/test-v1.0",
		Checksum:   "chk001",
		CreatedBy:  "test",
	}

	_, err := service.RegisterModel(ctx, reg)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	t.Run("Get Existing Version", func(t *testing.T) {
		version, err := service.GetVersion("lstm", "test-v1.0")
		if err != nil {
			t.Fatalf("GetVersion failed: %v", err)
		}

		if version.VersionID != "test-v1.0" {
			t.Errorf("Expected version test-v1.0, got %s", version.VersionID)
		}
	})

	t.Run("Get Non-existent Version", func(t *testing.T) {
		_, err := service.GetVersion("lstm", "nonexistent")
		if err == nil {
			t.Error("Expected error for non-existent version")
		}
	})

	t.Run("Get Current Version", func(t *testing.T) {
		err := service.ActivateVersion(ctx, "lstm", "test-v1.0")
		if err != nil {
			t.Fatalf("ActivateVersion failed: %v", err)
		}

		version, err := service.GetCurrentVersion("lstm")
		if err != nil {
			t.Fatalf("GetCurrentVersion failed: %v", err)
		}

		if version.VersionID != "test-v1.0" {
			t.Errorf("Expected current version test-v1.0, got %s", version.VersionID)
		}
	})

	t.Run("Get Current Version None Active", func(t *testing.T) {
		_, err := service.GetCurrentVersion("nonexistent-type")
		if err == nil {
			t.Error("Expected error when no version is active")
		}
	})
}

func TestModelUpdateService_ListVersions(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	modelTypes := []string{"lstm", "transformer"}

	for i, mt := range modelTypes {
		for j := 1; j <= 3; j++ {
			reg := ModelRegistration{
				ModelType: mt,
				VersionID:  mt + "-v1." + string(rune('0'+j)),
				ModelPath:  "/models/" + mt + "/v1." + string(rune('0'+j)),
				Checksum:   "chk" + string(rune('0'+i)) + string(rune('0'+j)),
				CreatedBy:  "test",
			}
			_, _ = service.RegisterModel(ctx, reg)
		}
	}

	t.Run("List All Versions", func(t *testing.T) {
		query := VersionQuery{
			ModelType: "lstm",
			Limit:     10,
		}

		versions, err := service.ListVersions(query)
		if err != nil {
			t.Fatalf("ListVersions failed: %v", err)
		}

		if len(versions) != 3 {
			t.Errorf("Expected 3 versions, got %d", len(versions))
		}
	})

	t.Run("List With Limit", func(t *testing.T) {
		query := VersionQuery{
			ModelType: "lstm",
			Limit:     2,
		}

		versions, err := service.ListVersions(query)
		if err != nil {
			t.Fatalf("ListVersions failed: %v", err)
		}

		if len(versions) != 2 {
			t.Errorf("Expected 2 versions with limit, got %d", len(versions))
		}
	})

	t.Run("List With Status Filter", func(t *testing.T) {
		_ = service.ActivateVersion(ctx, "lstm", "lstm-v1.1")

		query := VersionQuery{
			ModelType: "lstm",
			Status:    ModelStatusActive,
			Limit:     10,
		}

		versions, err := service.ListVersions(query)
		if err != nil {
			t.Fatalf("ListVersions failed: %v", err)
		}

		for _, v := range versions {
			if v.Status != ModelStatusActive {
				t.Errorf("Expected all versions to be active, got %s", v.Status)
			}
		}
	})
}

func TestModelUpdateService_UpdateMetrics(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	reg := ModelRegistration{
		ModelType: "lstm",
		VersionID:  "metrics-v1",
		ModelPath:  "/models/lstm/metrics-v1",
		Checksum:   "chk-m1",
		CreatedBy:  "test",
	}

	_, err := service.RegisterModel(ctx, reg)
	if err != nil {
		t.Fatalf("Failed to register model: %v", err)
	}

	t.Run("Update Metrics", func(t *testing.T) {
		update := ModelMetricsUpdate{
			ModelType:  "lstm",
			VersionID:  "metrics-v1",
			Prediction: 0.8,
			Actual:     1.0,
			LatencyMs:  50.0,
			IsError:    false,
		}

		err := service.UpdateMetrics(ctx, update)
		if err != nil {
			t.Fatalf("UpdateMetrics failed: %v", err)
		}

		metrics, err := service.GetPerformanceMetrics("lstm")
		if err != nil {
			t.Fatalf("GetPerformanceMetrics failed: %v", err)
		}

		if metrics.SampleCount != 1 {
			t.Errorf("Expected 1 sample, got %d", metrics.SampleCount)
		}
	})

	t.Run("Update Multiple Metrics", func(t *testing.T) {
		for i := 0; i < 10; i++ {
			update := ModelMetricsUpdate{
				ModelType:  "lstm",
				VersionID:  "metrics-v1",
				Prediction: 0.7,
				Actual:     0.7,
				LatencyMs:  40.0 + float64(i)*2,
				IsError:    false,
			}
			_ = service.UpdateMetrics(ctx, update)
		}

		metrics, err := service.GetPerformanceMetrics("lstm")
		if err != nil {
			t.Fatalf("GetPerformanceMetrics failed: %v", err)
		}

		if metrics.SampleCount != 11 {
			t.Errorf("Expected 11 samples, got %d", metrics.SampleCount)
		}

		if metrics.Accuracy <= 0 || metrics.Accuracy > 1 {
			t.Errorf("Accuracy should be between 0 and 1, got %f", metrics.Accuracy)
		}
	})

	t.Run("Update Metrics For Non-existent Model", func(t *testing.T) {
		update := ModelMetricsUpdate{
			ModelType:  "nonexistent",
			VersionID:  "v1",
			Prediction: 0.5,
			Actual:     0.5,
			LatencyMs:  30.0,
			IsError:    false,
		}

		err := service.UpdateMetrics(ctx, update)
		if err != nil {
			t.Fatalf("UpdateMetrics should not error for new model type: %v", err)
		}
	})
}

func TestModelUpdateService_HealthCheck(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	reg := ModelRegistration{
		ModelType: "transformer",
		VersionID:  "health-v1",
		ModelPath:  "/models/transformer/health-v1",
		Checksum:   "chk-h1",
		CreatedBy:  "test",
	}

	_, _ = service.RegisterModel(ctx, reg)
	_, _ = service.ActivateVersion(ctx, "transformer", "health-v1")

	t.Run("Health Check Insufficient Samples", func(t *testing.T) {
		result, err := service.PerformHealthCheck(ctx, "transformer")
		if err != nil {
			t.Fatalf("PerformHealthCheck failed: %v", err)
		}

		if result == nil {
			t.Fatal("Health check result should not be nil")
		}

		if result.IsHealthy {
			t.Log("Model healthy with insufficient samples (expected)")
		}
	})

	t.Run("Health Check With Metrics", func(t *testing.T) {
		for i := 0; i < 150; i++ {
			update := ModelMetricsUpdate{
				ModelType:  "transformer",
				VersionID:  "health-v1",
				Prediction: 0.9,
				Actual:     0.9,
				LatencyMs:  50.0,
				IsError:    false,
			}
			_ = service.UpdateMetrics(ctx, update)
		}

		result, err := service.PerformHealthCheck(ctx, "transformer")
		if err != nil {
			t.Fatalf("PerformHealthCheck failed: %v", err)
		}

		if result.Score < 0 || result.Score > 1 {
			t.Errorf("Health score should be between 0 and 1, got %f", result.Score)
		}
	})

	t.Run("Health Check No Active Version", func(t *testing.T) {
		_, err := service.PerformHealthCheck(ctx, "nonexistent-type")
		if err == nil {
			t.Error("Expected error when no active version")
		}
	})

	t.Run("Get Health Check Result", func(t *testing.T) {
		result, err := service.GetHealthCheckResult("transformer")
		if err != nil {
			t.Fatalf("GetHealthCheckResult failed: %v", err)
		}

		if result.ModelType != "transformer" {
			t.Errorf("Expected model type transformer, got %s", result.ModelType)
		}
	})
}

func TestModelUpdateService_Rollback(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		reg := ModelRegistration{
			ModelType: "rollback-test",
			VersionID:  "v1." + string(rune('0'+i)),
			ModelPath:  "/models/rollback/v1." + string(rune('0'+i)),
			Checksum:   "chk-r" + string(rune('0'+i)),
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)

		if i == 2 {
			_, _ = service.ActivateVersion(ctx, "rollback-test", "v1.2")
		}
	}

	_, _ = service.ActivateVersion(ctx, "rollback-test", "v1.3")

	t.Run("Manual Rollback", func(t *testing.T) {
		version, _ := service.GetCurrentVersion("rollback-test")
		if version.VersionID != "v1.3" {
			t.Fatalf("Expected current version v1.3, got %s", version.VersionID)
		}

		result, err := service.TriggerRollback(ctx, "rollback-test", "manual rollback test")
		if err != nil {
			t.Fatalf("TriggerRollback failed: %v", err)
		}

		if !result.Success {
			t.Errorf("Rollback should succeed: %s", result.Message)
		}

		if !result.RollbackTriggered {
			t.Error("RollbackTriggered should be true")
		}

		version, _ = service.GetCurrentVersion("rollback-test")
		if version.VersionID != "v1.2" {
			t.Errorf("Expected rolled back to v1.2, got %s", version.VersionID)
		}
	})

	t.Run("Rollback No Previous Version", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "single-version",
			VersionID:  "only-v1",
			ModelPath:  "/models/single/only-v1",
			Checksum:   "chk-s1",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)
		_, _ = service.ActivateVersion(ctx, "single-version", "only-v1")

		_, err := service.TriggerRollback(ctx, "single-version", "should fail")
		if err == nil {
			t.Error("Expected error when no previous version exists")
		}
	})

	t.Run("Rollback Max Attempts Reached", func(t *testing.T) {
		service.config.MaxRollbackAttempts = 2

		result, err := service.TriggerRollback(ctx, "rollback-test", "third rollback")
		if err == nil {
			if !result.Success {
				t.Log("Rollback correctly failed due to max attempts")
			}
		}

		service.config.MaxRollbackAttempts = 3
	})
}

func TestModelUpdateService_AutoRollback(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	reg := ModelRegistration{
		ModelType: "auto-rollback-test",
		VersionID:  "ar-v1",
		ModelPath:  "/models/auto/ar-v1",
		Checksum:   "chk-ar1",
		CreatedBy:  "test",
	}
	_, _ = service.RegisterModel(ctx, reg)
	_, _ = service.ActivateVersion(ctx, "auto-rollback-test", "ar-v1")

	t.Run("Auto Rollback Disabled", func(t *testing.T) {
		service.UpdateConfig(&UpdateConfig{
			AutoRollbackEnabled: false,
		})

		result, err := service.AutoRollbackIfNeeded(ctx, "auto-rollback-test")
		if err != nil {
			t.Fatalf("AutoRollbackIfNeeded failed: %v", err)
		}

		if result != nil {
			t.Error("Auto rollback should not trigger when disabled")
		}
	})

	t.Run("Auto Rollback With Low Accuracy", func(t *testing.T) {
		service.UpdateConfig(&UpdateConfig{
			AutoRollbackEnabled:       true,
			RollbackThresholdAccuracy: 0.5,
		})

		for i := 0; i < 150; i++ {
			update := ModelMetricsUpdate{
				ModelType:  "auto-rollback-test",
				VersionID:  "ar-v1",
				Prediction: 0.3,
				Actual:     0.9,
				LatencyMs:  50.0,
				IsError:    false,
			}
			_ = service.UpdateMetrics(ctx, update)
		}

		result, err := service.AutoRollbackIfNeeded(ctx, "auto-rollback-test")
		if err != nil {
			t.Fatalf("AutoRollbackIfNeeded failed: %v", err)
		}

		if result != nil && result.Success {
			t.Log("Auto rollback triggered due to low accuracy")
		}
	})
}

func TestModelUpdateService_Callbacks(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	var updateCalled bool
	var rollbackCalled bool

	service.RegisterUpdateCallback("callback-test", func(modelType string, oldVersion, newVersion *ModelVersion) error {
		updateCalled = true
		return nil
	})

	service.RegisterRollbackCallback("callback-test", func(modelType string, currentVersion, rollbackVersion *ModelVersion) error {
		rollbackCalled = true
		return nil
	})

	t.Run("Update Callback Triggered", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "callback-test",
			VersionID:  "cb-v1",
			ModelPath:  "/models/callback/cb-v1",
			Checksum:   "chk-cb1",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)

		reg2 := ModelRegistration{
			ModelType: "callback-test",
			VersionID:  "cb-v2",
			ModelPath:  "/models/callback/cb-v2",
			Checksum:   "chk-cb2",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg2)

		_, _ = service.ActivateVersion(ctx, "callback-test", "cb-v1")
		_, _ = service.ActivateVersion(ctx, "callback-test", "cb-v2")

		if !updateCalled {
			t.Error("Update callback should have been called")
		}
	})

	t.Run("Rollback Callback Triggered", func(t *testing.T) {
		_, _ = service.TriggerRollback(ctx, "callback-test", "callback test")

		if !rollbackCalled {
			t.Error("Rollback callback should have been called")
		}
	})
}

func TestModelUpdateService_Config(t *testing.T) {
	service := NewModelUpdateService()

	t.Run("Default Config", func(t *testing.T) {
		cfg := service.GetConfig()

		if !cfg.AutoRollbackEnabled {
			t.Error("Auto rollback should be enabled by default")
		}

		if cfg.MaxRollbackAttempts != 3 {
			t.Errorf("Expected max rollback attempts 3, got %d", cfg.MaxRollbackAttempts)
		}

		if cfg.HealthCheckInterval != 5*time.Minute {
			t.Errorf("Expected health check interval 5 minutes, got %v", cfg.HealthCheckInterval)
		}
	})

	t.Run("Update Config", func(t *testing.T) {
		service.UpdateConfig(&UpdateConfig{
			AutoRollbackEnabled:      false,
			MaxRollbackAttempts:      5,
			HealthCheckInterval:      10 * time.Minute,
			RollbackThresholdAccuracy: 0.1,
		})

		cfg := service.GetConfig()

		if cfg.AutoRollbackEnabled {
			t.Error("Auto rollback should be disabled")
		}

		if cfg.MaxRollbackAttempts != 5 {
			t.Errorf("Expected max rollback attempts 5, got %d", cfg.MaxRollbackAttempts)
		}

		if cfg.HealthCheckInterval != 10*time.Minute {
			t.Errorf("Expected health check interval 10 minutes, got %v", cfg.HealthCheckInterval)
		}
	})
}

func TestModelUpdateService_VersionManagement(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	t.Run("Mark Version Stable", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "stable-test",
			VersionID:  "stable-v1",
			ModelPath:  "/models/stable/stable-v1",
			Checksum:   "chk-st1",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)

		err := service.MarkVersionStable(ctx, "stable-test", "stable-v1")
		if err != nil {
			t.Fatalf("MarkVersionStable failed: %v", err)
		}

		version, _ := service.GetVersion("stable-test", "stable-v1")
		if version.Status != ModelStatusStable {
			t.Errorf("Expected status stable, got %s", version.Status)
		}
	})

	t.Run("Deprecate Version", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "deprecate-test",
			VersionID:  "dep-v1",
			ModelPath:  "/models/deprecate/dep-v1",
			Checksum:   "chk-dp1",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)
		_, _ = service.ActivateVersion(ctx, "deprecate-test", "dep-v1")

		err := service.DeprecateVersion(ctx, "deprecate-test", "dep-v1")
		if err == nil {
			t.Error("Should not be able to deprecate active version")
		}
	})

	t.Run("Deprecate Non-active Version", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "deprecate-test",
			VersionID:  "dep-v2",
			ModelPath:  "/models/deprecate/dep-v2",
			Checksum:   "chk-dp2",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)

		err := service.DeprecateVersion(ctx, "deprecate-test", "dep-v2")
		if err != nil {
			t.Fatalf("DeprecateVersion failed: %v", err)
		}

		version, _ := service.GetVersion("deprecate-test", "dep-v2")
		if version.Status != ModelStatusDeprecated {
			t.Errorf("Expected status deprecated, got %s", version.Status)
		}
	})
}

func TestModelUpdateService_Statistics(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	for i := 1; i <= 3; i++ {
		reg := ModelRegistration{
			ModelType: "stats-test",
			VersionID:  "stats-v" + string(rune('0'+i)),
			ModelPath:  "/models/stats/stats-v" + string(rune('0'+i)),
			Checksum:   "chk-st" + string(rune('0'+i)),
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)
		_, _ = service.ActivateVersion(ctx, "stats-test", "stats-v"+string(rune('0'+i)))
	}

	_, _ = service.TriggerRollback(ctx, "stats-test", "test rollback")
	_, _ = service.TriggerRollback(ctx, "stats-test", "test rollback 2")

	t.Run("Get Version Statistics", func(t *testing.T) {
		stats, err := service.GetVersionStatistics("stats-test")
		if err != nil {
			t.Fatalf("GetVersionStatistics failed: %v", err)
		}

		if stats == nil {
			t.Fatal("Statistics should not be nil")
		}

		if total, ok := stats["total_versions"].(int); !ok || total != 3 {
			t.Errorf("Expected 3 total versions, got %v", stats["total_versions"])
		}

		if rollbacks, ok := stats["total_rollbacks"].(int); !ok || rollbacks < 2 {
			t.Errorf("Expected at least 2 rollbacks, got %v", stats["total_rollbacks"])
		}
	})

	t.Run("Compare Versions", func(t *testing.T) {
		comparison, err := service.CompareVersions("stats-test", "stats-v1", "stats-v3")
		if err != nil {
			t.Fatalf("CompareVersions failed: %v", err)
		}

		if comparison == nil {
			t.Fatal("Comparison should not be nil")
		}

		if _, ok := comparison["version1"]; !ok {
			t.Error("Comparison should contain version1")
		}

		if _, ok := comparison["version2"]; !ok {
			t.Error("Comparison should contain version2")
		}
	})

	t.Run("Export Version History", func(t *testing.T) {
		data, err := service.ExportVersionHistory("stats-test")
		if err != nil {
			t.Fatalf("ExportVersionHistory failed: %v", err)
		}

		if len(data) == 0 {
			t.Error("Exported data should not be empty")
		}
	})
}

func TestModelUpdateService_Concurrency(t *testing.T) {
	service := NewModelUpdateService()
	ctx := context.Background()

	t.Run("Concurrent Registration", func(t *testing.T) {
		var wg sync.WaitGroup
		numGoroutines := 10

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				reg := ModelRegistration{
					ModelType: "concurrent-test",
					VersionID:  "concurrent-v" + string(rune('0'+idx)),
					ModelPath:  "/models/concurrent/concurrent-v" + string(rune('0'+idx)),
					Checksum:   "chk-c" + string(rune('0'+idx)),
					CreatedBy:  "test",
				}
				_, err := service.RegisterModel(ctx, reg)
				if err != nil {
					t.Errorf("Concurrent registration failed: %v", err)
				}
			}(i)
		}

		wg.Wait()

		versions, _ := service.ListVersions(VersionQuery{
			ModelType: "concurrent-test",
			Limit:     100,
		})

		if len(versions) != numGoroutines {
			t.Errorf("Expected %d versions registered, got %d", numGoroutines, len(versions))
		}
	})

	t.Run("Concurrent Metric Updates", func(t *testing.T) {
		reg := ModelRegistration{
			ModelType: "metrics-concurrent",
			VersionID:  "mc-v1",
			ModelPath:  "/models/metrics/mc-v1",
			Checksum:   "chk-mc1",
			CreatedBy:  "test",
		}
		_, _ = service.RegisterModel(ctx, reg)
		_, _ = service.ActivateVersion(ctx, "metrics-concurrent", "mc-v1")

		var wg sync.WaitGroup
		numGoroutines := 20

		for i := 0; i < numGoroutines; i++ {
			wg.Add(1)
			go func(idx int) {
				defer wg.Done()

				update := ModelMetricsUpdate{
					ModelType:  "metrics-concurrent",
					VersionID:  "mc-v1",
					Prediction: 0.8,
					Actual:     0.8,
					LatencyMs:  50.0,
					IsError:    false,
				}
				_ = service.UpdateMetrics(ctx, update)
			}(i)
		}

		wg.Wait()

		metrics, _ := service.GetPerformanceMetrics("metrics-concurrent")
		if metrics.SampleCount != int64(numGoroutines) {
			t.Errorf("Expected %d samples, got %d", numGoroutines, metrics.SampleCount)
		}
	})
}

func TestModelUpdateService_StartStop(t *testing.T) {
	service := NewModelUpdateService()
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	t.Run("Start and Stop", func(t *testing.T) {
		service.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		service.Stop()
	})

	t.Run("Multiple Start Calls", func(t *testing.T) {
		service.Start(ctx)
		service.Start(ctx)
		time.Sleep(100 * time.Millisecond)
		service.Stop()
	})
}
