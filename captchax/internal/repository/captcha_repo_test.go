package repository

import (
	"context"
	"database/sql"
	"testing"
	"time"

	"captchax/internal/model"

	_ "github.com/lib/pq"
)

func setupTestDB(t *testing.T) *sql.DB {
	db, err := sql.Open("postgres", "host=localhost port=5432 user=postgres password=postgres dbname=captcha_test sslmode=disable")
	if err != nil {
		t.Skipf("Skipping test: database not available: %v", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := db.PingContext(ctx); err != nil {
		db.Close()
		t.Skipf("Skipping test: cannot connect to database: %v", err)
	}

	_, _ = db.ExecContext(ctx, "TRUNCATE captcha_logs, blacklist, whitelist, captcha_config, archive_metadata, archive_policy CASCADE")

	return db
}

func TestCaptchaRepo_Create(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	log := &model.CaptchaLog{
		Type:      "slider",
		ClientID:  "test_client_123",
		IP:        "192.168.1.1",
		UserAgent: sql.NullString{String: "Mozilla/5.0", Valid: true},
		Result:    true,
		Duration:  1500,
		RiskScore: 20,
	}

	id, err := repo.Create(ctx, log)
	if err != nil {
		t.Fatalf("Create failed: %v", err)
	}

	if id <= 0 {
		t.Errorf("Expected positive ID, got %d", id)
	}

	retrieved, err := repo.GetByID(ctx, id)
	if err != nil {
		t.Fatalf("GetByID failed: %v", err)
	}

	if retrieved == nil {
		t.Fatal("Expected to retrieve log, got nil")
	}

	if retrieved.Type != log.Type {
		t.Errorf("Expected type %s, got %s", log.Type, retrieved.Type)
	}

	if retrieved.ClientID != log.ClientID {
		t.Errorf("Expected client_id %s, got %s", log.ClientID, retrieved.ClientID)
	}
}

func TestCaptchaRepo_List(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	for i := 0; i < 5; i++ {
		log := &model.CaptchaLog{
			Type:       "click",
			ClientID:   "test_client_list",
			IP:         "10.0.0.1",
			Result:     i%2 == 0,
			Duration:   1000 + i*100,
			RiskScore:  i * 10,
		}
		_, _ = repo.Create(ctx, log)
		time.Sleep(10 * time.Millisecond)
	}

	filter := &model.CaptchaLogFilter{
		ClientID: "test_client_list",
		Page:     1,
		PageSize: 10,
	}

	logs, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List failed: %v", err)
	}

	if len(logs) != 5 {
		t.Errorf("Expected 5 logs, got %d", len(logs))
	}
}

func TestCaptchaRepo_ListWithDateFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	log := &model.CaptchaLog{
		Type:      "puzzle",
		ClientID:  "date_filter_test",
		IP:        "172.16.0.1",
		Result:    true,
		Duration:  2000,
		RiskScore: 15,
	}
	_, _ = repo.Create(ctx, log)

	now := time.Now()
	yesterday := now.Add(-24 * time.Hour)
	tomorrow := now.Add(24 * time.Hour)

	filter := &model.CaptchaLogFilter{
		StartDate: &yesterday,
		EndDate:   &tomorrow,
		Page:      1,
		PageSize:  10,
	}

	logs, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List with date filter failed: %v", err)
	}

	if len(logs) == 0 {
		t.Error("Expected at least one log within date range")
	}
}

func TestCaptchaRepo_ListWithResultFilter(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	for i := 0; i < 3; i++ {
		log := &model.CaptchaLog{
			Type:      "slider",
			ClientID:  "result_filter_test",
			IP:        "192.168.100.1",
			Result:    i%2 == 0,
			Duration:  500,
			RiskScore: 5,
		}
		_, _ = repo.Create(ctx, log)
	}

	successOnly := true
	filter := &model.CaptchaLogFilter{
		Result:   &successOnly,
		ClientID: "result_filter_test",
		Page:     1,
		PageSize: 10,
	}

	logs, err := repo.List(ctx, filter)
	if err != nil {
		t.Fatalf("List with result filter failed: %v", err)
	}

	for _, log := range logs {
		if !log.Result {
			t.Error("Expected only successful results")
		}
	}
}

func TestCaptchaRepo_CountByIP(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	testIP := "10.10.10.10"
	for i := 0; i < 3; i++ {
		log := &model.CaptchaLog{
			Type:    "slider",
			ClientID: "count_test",
			IP:      testIP,
			Result:  true,
			Duration: 1000,
			RiskScore: 10,
		}
		_, _ = repo.Create(ctx, log)
	}

	since := time.Now().Add(-1 * time.Hour)
	count, err := repo.CountByIP(ctx, testIP, since)
	if err != nil {
		t.Fatalf("CountByIP failed: %v", err)
	}

	if count < 3 {
		t.Errorf("Expected count >= 3, got %d", count)
	}
}

func TestCaptchaRepo_GetStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	for i := 0; i < 10; i++ {
		log := &model.CaptchaLog{
			Type:      "slider",
			ClientID:  "stats_test",
			IP:        "20.20.20.20",
			Result:    i%2 == 0,
			Duration:  1000 + i*100,
			RiskScore: i * 5,
		}
		_, _ = repo.Create(ctx, log)
	}

	startDate := time.Now().Add(-24 * time.Hour)
	endDate := time.Now().Add(1 * time.Hour)

	stats, err := repo.GetStats(ctx, startDate, endDate)
	if err != nil {
		t.Fatalf("GetStats failed: %v", err)
	}

	if stats.TotalCount < 10 {
		t.Errorf("Expected total count >= 10, got %d", stats.TotalCount)
	}

	if stats.SuccessCount+stats.FailCount != stats.TotalCount {
		t.Error("Success + Fail should equal Total")
	}

	if stats.TotalCount > 0 {
		expectedRate := float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
		if stats.SuccessRate != expectedRate {
			t.Errorf("Success rate mismatch: expected %.2f, got %.2f", expectedRate, stats.SuccessRate)
		}
	}
}

func TestCaptchaRepo_DeleteOlderThan(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	log := &model.CaptchaLog{
		Type:      "click",
		ClientID:  "delete_test",
		IP:        "30.30.30.30",
		Result:    false,
		Duration:  500,
		RiskScore: 80,
	}
	_, _ = repo.Create(ctx, log)

	deleted, err := repo.DeleteOlderThan(ctx, time.Now().Add(1*time.Hour))
	if err != nil {
		t.Fatalf("DeleteOlderThan failed: %v", err)
	}

	if deleted < 0 {
		t.Errorf("Expected deleted >= 0, got %d", deleted)
	}
}

func TestCaptchaRepo_CountByIPWithWindow(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	testIP := "40.40.40.40"
	for i := 0; i < 5; i++ {
		log := &model.CaptchaLog{
			Type:      "puzzle",
			ClientID:  "window_test",
			IP:        testIP,
			Result:    true,
			Duration:  800,
			RiskScore: 25,
		}
		_, _ = repo.Create(ctx, log)
		time.Sleep(5 * time.Millisecond)
	}

	windowStart := time.Now().Add(-1 * time.Hour)
	windowEnd := time.Now().Add(1 * time.Hour)

	count, err := repo.CountByIPWithWindow(ctx, testIP, windowStart, windowEnd)
	if err != nil {
		t.Fatalf("CountByIPWithWindow failed: %v", err)
	}

	if count < 5 {
		t.Errorf("Expected count >= 5, got %d", count)
	}
}

func TestCaptchaRepo_GetIPStats(t *testing.T) {
	db := setupTestDB(t)
	defer db.Close()

	repo := NewCaptchaRepo(db)
	ctx := context.Background()

	testIP := "50.50.50.50"
	for i := 0; i < 15; i++ {
		log := &model.CaptchaLog{
			Type:      "slider",
			ClientID:  "ip_stats_test",
			IP:        testIP,
			Result:    i%2 == 0,
			Duration:  1200,
			RiskScore: 30 + i*2,
		}
		_, _ = repo.Create(ctx, log)
	}

	stats, err := repo.GetIPStats(ctx, 10)
	if err != nil {
		t.Fatalf("GetIPStats failed: %v", err)
	}

	if len(stats) == 0 {
		t.Error("Expected at least one IP stat entry")
	}
}

func TestCaptchaLogFilter_Limit(t *testing.T) {
	tests := []struct {
		name     string
		pageSize int
		expected int
	}{
		{"Default page size", 0, 20},
		{"Custom page size", 50, 50},
		{"Max page size exceeded", 150, 100},
		{"Negative page size", -10, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &model.CaptchaLogFilter{
				PageSize: tt.pageSize,
			}
			limit := filter.Limit()
			if limit != tt.expected {
				t.Errorf("Expected limit %d, got %d", tt.expected, limit)
			}
		})
	}
}

func TestCaptchaLogFilter_Offset(t *testing.T) {
	tests := []struct {
		name           string
		page           int
		pageSize       int
		expectedOffset int
	}{
		{"First page", 1, 20, 0},
		{"Second page", 2, 20, 20},
		{"Third page", 3, 10, 20},
		{"Zero page defaults to 1", 0, 20, 0},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			filter := &model.CaptchaLogFilter{
				Page:     tt.page,
				PageSize: tt.pageSize,
			}
			offset := filter.Offset()
			if offset != tt.expectedOffset {
				t.Errorf("Expected offset %d, got %d", tt.expectedOffset, offset)
			}
		})
	}
}

func TestCaptchaLog_ToDTO(t *testing.T) {
	log := &model.CaptchaLog{
		ID:        1,
		Type:      "slider",
		ClientID:  "test_client",
		IP:        "127.0.0.1",
		UserAgent: sql.NullString{String: "TestAgent/1.0", Valid: true},
		Result:    true,
		Duration:  1000,
		RiskScore: 25,
		CreatedAt: time.Date(2026, 5, 14, 10, 30, 0, 0, time.UTC),
	}

	dto := log.ToDTO()

	if dto.ID != log.ID {
		t.Errorf("ID mismatch: expected %d, got %d", log.ID, dto.ID)
	}

	if dto.Type != log.Type {
		t.Errorf("Type mismatch: expected %s, got %s", log.Type, dto.Type)
	}

	if dto.UserAgent != log.UserAgent.String {
		t.Errorf("UserAgent mismatch: expected %s, got %s", log.UserAgent.String, dto.UserAgent)
	}

	if dto.Result != log.Result {
		t.Errorf("Result mismatch: expected %v, got %v", log.Result, dto.Result)
	}
}

func TestCaptchaLog_ToDTO_NullUserAgent(t *testing.T) {
	log := &model.CaptchaLog{
		ID:        2,
		Type:      "click",
		ClientID:  "test_client",
		IP:        "127.0.0.1",
		UserAgent: sql.NullString{Valid: false},
		Result:    false,
		Duration:  500,
		RiskScore: 75,
		CreatedAt: time.Now(),
	}

	dto := log.ToDTO()

	if dto.UserAgent != "" {
		t.Errorf("Expected empty UserAgent, got %s", dto.UserAgent)
	}
}
