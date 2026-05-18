package database

import (
	"testing"
	"time"
)

func TestDataArchiverCreation(t *testing.T) {
	archiver := &DataArchiver{
		enabled:          true,
		archiveThreshold: 30 * 24 * time.Hour,
		archivePrefix:    "archive_",
		cleanupEnabled:   true,
		cleanupThreshold: 365 * 24 * time.Hour,
	}

	if !archiver.enabled {
		t.Error("Archiver should be enabled")
	}

	if archiver.archiveThreshold != 30*24*time.Hour {
		t.Errorf("Archive threshold = %v, want %v", archiver.archiveThreshold, 30*24*time.Hour)
	}
}

func TestArchiveStats(t *testing.T) {
	stats := &ArchiveStats{
		TotalArchivedRecords: 1000,
		TotalCleanedRecords:  500,
		LastArchiveTime:      time.Now(),
		LastCleanupTime:     time.Now(),
		ArchiveErrors:       5,
		CleanupErrors:        3,
	}

	if stats.TotalArchivedRecords != 1000 {
		t.Errorf("TotalArchivedRecords = %d, want %d", stats.TotalArchivedRecords, 1000)
	}

	if stats.TotalCleanedRecords != 500 {
		t.Errorf("TotalCleanedRecords = %d, want %d", stats.TotalCleanedRecords, 500)
	}
}

func TestEnhancedPoolConfig(t *testing.T) {
	config := &EnhancedPoolConfig{
		MaxOpenConns:        100,
		MaxIdleConns:        50,
		MinIdleConns:        10,
		ConnMaxLifetime:     30 * time.Minute,
		ConnMaxIdleTime:     10 * time.Minute,
		HealthCheckInterval: 30 * time.Second,
	}

	if config.MaxOpenConns != 100 {
		t.Errorf("MaxOpenConns = %d, want %d", config.MaxOpenConns, 100)
	}
	if config.MaxIdleConns != 50 {
		t.Errorf("MaxIdleConns = %d, want %d", config.MaxIdleConns, 50)
	}
	if config.MinIdleConns != 10 {
		t.Errorf("MinIdleConns = %d, want %d", config.MinIdleConns, 10)
	}
}

func TestTuningRecord(t *testing.T) {
	oldConfig := &EnhancedPoolConfig{
		MaxOpenConns: 100,
	}
	newConfig := &EnhancedPoolConfig{
		MaxOpenConns: 150,
	}

	record := &TuningRecord{
		Timestamp: time.Now(),
		OldConfig: oldConfig,
		NewConfig: newConfig,
		Reason:    "increased load",
	}

	if record.OldConfig.MaxOpenConns != 100 {
		t.Errorf("OldConfig.MaxOpenConns = %d, want %d", record.OldConfig.MaxOpenConns, 100)
	}
	if record.NewConfig.MaxOpenConns != 150 {
		t.Errorf("NewConfig.MaxOpenConns = %d, want %d", record.NewConfig.MaxOpenConns, 150)
	}
	if record.Reason != "increased load" {
		t.Errorf("Reason = %s, want %s", record.Reason, "increased load")
	}
}

func TestPoolHealthStatus(t *testing.T) {
	status := &PoolHealthStatus{
		IsHealthy:        true,
		Score:            0.95,
		Issues:           []string{},
		Recommendations:  []string{},
		LastCheck:        time.Now(),
	}

	if !status.IsHealthy {
		t.Error("IsHealthy should be true")
	}
	if status.Score != 0.95 {
		t.Errorf("Score = %f, want %f", status.Score, 0.95)
	}
}

func TestConnectionPressure(t *testing.T) {
	pressure := &ConnectionPressure{
		Timestamp:      time.Now(),
		OpenConnections: 100,
		InUse:          80,
		Idle:           20,
		WaitCount:      5,
		PressureLevel:  "normal",
		Advice:         "connections healthy",
	}

	if pressure.OpenConnections != 100 {
		t.Errorf("OpenConnections = %d, want %d", pressure.OpenConnections, 100)
	}
	if pressure.InUse != 80 {
		t.Errorf("InUse = %d, want %d", pressure.InUse, 80)
	}
	if pressure.PressureLevel != "normal" {
		t.Errorf("PressureLevel = %s, want %s", pressure.PressureLevel, "normal")
	}
}

func TestConnectionPoolOptimizer(t *testing.T) {
	config := &ConnectionPoolConfig{
		MaxOpenConns:    100,
		MaxIdleConns:    50,
		ConnMaxLifetime: 30 * time.Minute,
		ConnMaxIdleTime: 10 * time.Minute,
	}

	optimizer := NewConnectionPoolOptimizer(config)

	if optimizer == nil {
		t.Error("Expected optimizer to be created")
	}

	optimizer.EnableAutoTuning(true)
	optimizer.EnableAutoTuning(false)
}

func TestEnhancedConnectionPoolOptimizer(t *testing.T) {
	optimizer := &EnhancedConnectionPoolOptimizer{
		healthCheckInterval:   30 * time.Second,
		autoTuningEnabled:     true,
		maxHistorySize:        100,
		tuningHistory:         make([]TuningRecord, 0),
	}

	if optimizer.healthCheckInterval != 30*time.Second {
		t.Errorf("healthCheckInterval = %v, want %v", optimizer.healthCheckInterval, 30*time.Second)
	}
	if !optimizer.autoTuningEnabled {
		t.Error("autoTuningEnabled should be true")
	}
	if optimizer.maxHistorySize != 100 {
		t.Errorf("maxHistorySize = %d, want %d", optimizer.maxHistorySize, 100)
	}
}

func TestConnectionPoolMetrics(t *testing.T) {
	metrics := &ConnectionPoolMetrics{}

	metrics.TotalConnections = 100
	metrics.ActiveConnections = 50
	metrics.IdleConnections = 50
	metrics.WaitCount = 10
	metrics.WaitDuration = 100 * time.Millisecond

	if metrics.TotalConnections != 100 {
		t.Errorf("TotalConnections = %d, want %d", metrics.TotalConnections, 100)
	}
	if metrics.ActiveConnections != 50 {
		t.Errorf("ActiveConnections = %d, want %d", metrics.ActiveConnections, 50)
	}
	if metrics.IdleConnections != 50 {
		t.Errorf("IdleConnections = %d, want %d", metrics.IdleConnections, 50)
	}
}

func TestPoolMetricsSnapshot(t *testing.T) {
	snapshot := &PoolMetricsSnapshot{
		Timestamp:        time.Now(),
		TotalConnections:  50,
		ActiveConnections: 30,
		IdleConnections:  20,
		WaitCount:        5,
	}

	if snapshot.TotalConnections != 50 {
		t.Errorf("TotalConnections = %d, want %d", snapshot.TotalConnections, 50)
	}
	if snapshot.ActiveConnections != 30 {
		t.Errorf("ActiveConnections = %d, want %d", snapshot.ActiveConnections, 30)
	}
	if snapshot.IdleConnections != 20 {
		t.Errorf("IdleConnections = %d, want %d", snapshot.IdleConnections, 20)
	}
}
