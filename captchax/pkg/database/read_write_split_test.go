package database

import (
	"context"
	"testing"
	"time"

	_ "github.com/lib/pq"
)

func getTestDBConfig() *DBConfig {
	return &DBConfig{
		Host:            "localhost",
		Port:            5432,
		User:            "postgres",
		Password:        "postgres",
		DBName:          "captcha_test",
		SSLMode:         "disable",
		MaxOpenConns:    5,
		MaxIdleConns:    2,
		ConnMaxLifetime: 5 * time.Minute,
	}
}

func setupTestSplitter(t *testing.T) *ReadWriteSplitter {
	config := getTestDBConfig()

	splitter, err := NewReadWriteSplitter(config, nil)
	if err != nil {
		t.Skipf("Skipping test: cannot create splitter: %v", err)
	}

	return splitter
}

func TestNewReadWriteSplitter_PrimaryOnly(t *testing.T) {
	config := getTestDBConfig()

	splitter, err := NewReadWriteSplitter(config, nil)
	if err != nil {
		t.Fatalf("Failed to create splitter: %v", err)
	}
	defer splitter.Close()

	if splitter.Primary() == nil {
		t.Error("Expected primary connection, got nil")
	}
}

func TestReadWriteSplitter_Ping(t *testing.T) {
	splitter := setupTestSplitter(t)
	defer splitter.Close()

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := splitter.Ping(ctx); err != nil {
		t.Errorf("Ping failed: %v", err)
	}
}

func TestReadWriteSplitter_WriteExec(t *testing.T) {
	splitter := setupTestSplitter(t)
	defer splitter.Close()

	ctx := context.Background()

	query := `INSERT INTO captcha_logs (captcha_type, client_id, ip, user_agent, result, duration, risk_score, created_at)
			  VALUES ($1, $2, $3, $4, $5, $6, $7, $8)
			  RETURNING id`

	result, err := splitter.WriteExec(ctx, query,
		"slider", "test_write", "192.168.1.1", "TestAgent", true, 1000, 10, time.Now())

	if err != nil {
		t.Fatalf("WriteExec failed: %v", err)
	}

	id, err := result.LastInsertId()
	if err != nil {
		t.Fatalf("LastInsertId failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}
}

func TestReadWriteSplitter_ReadQuery(t *testing.T) {
	splitter := setupTestSplitter(t)
	defer splitter.Close()

	ctx := context.Background()

	_, err := splitter.WriteExec(ctx,
		`INSERT INTO captcha_logs (captcha_type, client_id, ip, result, duration, risk_score, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		"click", "test_read", "10.0.0.1", true, 500, 15, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	rows, err := splitter.ReadQuery(ctx,
		`SELECT id, captcha_type, client_id, ip FROM captcha_logs WHERE client_id = $1`,
		"test_read")

	if err != nil {
		t.Fatalf("ReadQuery failed: %v", err)
	}
	defer rows.Close()

	count := 0
	for rows.Next() {
		var id int64
		var captchaType, clientID, ip string
		if err := rows.Scan(&id, &captchaType, &clientID, &ip); err != nil {
			t.Fatalf("Scan failed: %v", err)
		}
		count++
	}

	if count == 0 {
		t.Error("Expected at least one row")
	}
}

func TestReadWriteSplitter_ReadQueryRow(t *testing.T) {
	splitter := setupTestSplitter(t)
	defer splitter.Close()

	ctx := context.Background()

	_, err := splitter.WriteExec(ctx,
		`INSERT INTO captcha_logs (captcha_type, client_id, ip, result, duration, risk_score, created_at)
		 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
		"puzzle", "test_row", "172.16.0.1", true, 2000, 30, time.Now())
	if err != nil {
		t.Fatalf("Failed to insert test data: %v", err)
	}

	var id int64
	var captchaType string

	err = splitter.ReadQueryRow(ctx,
		`SELECT id, captcha_type FROM captcha_logs WHERE client_id = $1`,
		"test_row").Scan(&id, &captchaType)

	if err != nil {
		t.Fatalf("ReadQueryRow failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	if captchaType != "puzzle" {
		t.Errorf("Expected captcha_type 'puzzle', got '%s'", captchaType)
	}
}

func TestReadWriteSplitter_GetPoolStats(t *testing.T) {
	splitter := setupTestSplitter(t)
	defer splitter.Close()

	ctx := context.Background()

	for i := 0; i < 3; i++ {
		_, _ = splitter.WriteExec(ctx,
			`INSERT INTO captcha_logs (captcha_type, client_id, ip, result, duration, risk_score, created_at)
			 VALUES ($1, $2, $3, $4, $5, $6, $7)`,
			"slider", "test_stats", "10.10.10.10", true, 1000, 20, time.Now())
	}

	primary, replicas := splitter.GetPoolStats()

	if len(primary) == 0 {
		t.Error("Expected at least one primary pool stat")
	}

	for _, p := range primary {
		if p.MaxOpenConns != 5 {
			t.Errorf("Expected MaxOpenConns 5, got %d", p.MaxOpenConns)
		}
	}

	if len(replicas) != 0 {
		t.Errorf("Expected 0 replicas, got %d", len(replicas))
	}
}

func TestReadWriteSplitter_Close(t *testing.T) {
	config := getTestDBConfig()

	splitter, err := NewReadWriteSplitter(config, nil)
	if err != nil {
		t.Fatalf("Failed to create splitter: %v", err)
	}

	if err := splitter.Close(); err != nil {
		t.Errorf("Close failed: %v", err)
	}
}

func TestReadWriteSplitter_HasHealthyReplicas(t *testing.T) {
	config := getTestDBConfig()

	splitter, err := NewReadWriteSplitter(config, nil)
	if err != nil {
		t.Fatalf("Failed to create splitter: %v", err)
	}
	defer splitter.Close()

	if splitter.hasHealthyReplicas() {
		t.Error("Expected no healthy replicas when none configured")
	}
}

func TestReadWriteSplitter_GetReadDB(t *testing.T) {
	config := getTestDBConfig()

	splitter, err := NewReadWriteSplitter(config, nil)
	if err != nil {
		t.Fatalf("Failed to create splitter: %v", err)
	}
	defer splitter.Close()

	db := splitter.getReadDB()
	if db == nil {
		t.Error("Expected non-nil read DB")
	}
}

func TestReadWriteSplitter_GetWriteDB(t *testing.T) {
	config := getTestDBConfig()

	splitter, err := NewReadWriteSplitter(config, nil)
	if err != nil {
		t.Fatalf("Failed to create splitter: %v", err)
	}
	defer splitter.Close()

	db := splitter.getWriteDB()
	if db == nil {
		t.Error("Expected non-nil write DB")
	}
}

func TestPoolStats_Structure(t *testing.T) {
	stats := ConnectionPoolStats{
		MaxOpenConns:      10,
		OpenConns:         5,
		IdleConns:         3,
		WaitCount:         2,
		WaitDurationMs:    100,
		MaxIdleClosed:     1,
		MaxLifetimeClosed: 0,
	}

	if stats.MaxOpenConns != 10 {
		t.Errorf("MaxOpenConns: expected 10, got %d", stats.MaxOpenConns)
	}

	if stats.OpenConns != 5 {
		t.Errorf("OpenConns: expected 5, got %d", stats.OpenConns)
	}

	if stats.IdleConns != 3 {
		t.Errorf("IdleConns: expected 3, got %d", stats.IdleConns)
	}
}

func TestRoutingHint_Structure(t *testing.T) {
	hint := RoutingHint{
		ForcePrimary:      true,
		ForceReplica:      false,
		PreferredReplica:   0,
	}

	if !hint.ForcePrimary {
		t.Error("Expected ForcePrimary to be true")
	}

	if hint.ForceReplica {
		t.Error("Expected ForceReplica to be false")
	}
}

func TestHealthStatus_Structure(t *testing.T) {
	status := HealthStatus{
		Healthy:    true,
		LagBytes:   1024,
		LagSeconds: 0.5,
		LastChecked: time.Now(),
	}

	if !status.Healthy {
		t.Error("Expected Healthy to be true")
	}

	if status.LagBytes != 1024 {
		t.Errorf("LagBytes: expected 1024, got %d", status.LagBytes)
	}
}
