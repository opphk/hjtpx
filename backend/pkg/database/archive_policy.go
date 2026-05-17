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

type ArchivePolicyManager struct {
	db           *gorm.DB
	config       *config.Config
	policies     map[string]*ArchivePolicy
	executors    map[string]*PolicyExecutor
	enabled      bool
	mu           sync.RWMutex
}

type ArchivePolicy struct {
	ID              string          `json:"id"`
	Name            string         `json:"name"`
	TableName       string         `json:"table_name"`
	Condition       string         `json:"condition"`
	TargetType      string         `json:"target_type"`
	TargetLocation  string         `json:"target_location"`
	Priority        int            `json:"priority"`
	Schedule        string         `json:"schedule"`
	IsActive        bool           `json:"is_active"`
	RetentionDays   int            `json:"retention_days"`
	BatchSize       int            `json:"batch_size"`
	CompressionEnabled bool        `json:"compression_enabled"`
	EncryptionEnabled bool        `json:"encryption_enabled"`
	LastExecuted    *time.Time     `json:"last_executed,omitempty"`
	NextExecution   *time.Time     `json:"next_execution,omitempty"`
	TotalArchived   int64          `json:"total_archived"`
	TotalRestored   int64          `json:"total_restored"`
	CreatedAt       time.Time      `json:"created_at"`
	UpdatedAt       time.Time      `json:"updated_at"`
}

type PolicyExecutor struct {
	policyID  string
	status    string
	startedAt time.Time
	stats     *ExecutorStats
}

type ExecutorStats struct {
	ProcessedRecords int64     `json:"processed_records"`
	SuccessCount    int64     `json:"success_count"`
	FailedCount     int64     `json:"failed_count"`
	SkippedCount    int64     `json:"skipped_count"`
	BytesArchived   int64     `json:"bytes_archived"`
	Duration        time.Duration `json:"duration"`
	LastError       string    `json:"last_error,omitempty"`
}

type ArchiveJob struct {
	ID          string    `json:"id"`
	PolicyID    string    `json:"policy_id"`
	Status      string    `json:"status"`
	StartedAt   time.Time `json:"started_at"`
	CompletedAt time.Time `json:"completed_at,omitempty"`
	Progress    float64   `json:"progress"`
	TotalRecords int64    `json:"total_records"`
	ProcessedRecords int64 `json:"processed_records"`
	Error       string    `json:"error,omitempty"`
}

var archivePolicyManager *ArchivePolicyManager

func InitArchivePolicyManager(db *gorm.DB, cfg *config.Config) error {
	archivePolicyManager = &ArchivePolicyManager{
		db:         db,
		config:     cfg,
		policies:   make(map[string]*ArchivePolicy),
		executors:  make(map[string]*PolicyExecutor),
		enabled:    cfg.Database.DataArchiving.Enabled,
	}

	if archivePolicyManager.enabled {
		archivePolicyManager.loadPolicies()
		go archivePolicyManager.startScheduler()
		log.Println("Archive policy manager initialized")
	}

	return nil
}

func GetArchivePolicyManager() *ArchivePolicyManager {
	return archivePolicyManager
}

func (m *ArchivePolicyManager) CreatePolicy(policy *ArchivePolicy) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if policy.ID == "" {
		policy.ID = fmt.Sprintf("policy_%d", time.Now().UnixNano())
	}
	policy.CreatedAt = time.Now()
	policy.UpdatedAt = time.Now()
	policy.IsActive = true

	if policy.BatchSize <= 0 {
		policy.BatchSize = 1000
	}

	if policy.RetentionDays <= 0 {
		policy.RetentionDays = 365
	}

	m.policies[policy.ID] = policy

	log.Printf("Created archive policy: %s (table: %s)", policy.Name, policy.TableName)
	return nil
}

func (m *ArchivePolicyManager) GetPolicy(policyID string) (*ArchivePolicy, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policy, exists := m.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	return policy, nil
}

func (m *ArchivePolicyManager) ListPolicies() []*ArchivePolicy {
	m.mu.RLock()
	defer m.mu.RUnlock()

	policies := make([]*ArchivePolicy, 0, len(m.policies))
	for _, p := range m.policies {
		policyCopy := *p
		policies = append(policies, &policyCopy)
	}

	return policies
}

func (m *ArchivePolicyManager) UpdatePolicy(policyID string, updates map[string]interface{}) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	policy, exists := m.policies[policyID]
	if !exists {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	if name, ok := updates["name"].(string); ok {
		policy.Name = name
	}
	if condition, ok := updates["condition"].(string); ok {
		policy.Condition = condition
	}
	if priority, ok := updates["priority"].(int); ok {
		policy.Priority = priority
	}
	if schedule, ok := updates["schedule"].(string); ok {
		policy.Schedule = schedule
	}
	if isActive, ok := updates["is_active"].(bool); ok {
		policy.IsActive = isActive
	}
	if retentionDays, ok := updates["retention_days"].(int); ok {
		policy.RetentionDays = retentionDays
	}
	if batchSize, ok := updates["batch_size"].(int); ok {
		policy.BatchSize = batchSize
	}

	policy.UpdatedAt = time.Now()

	return nil
}

func (m *ArchivePolicyManager) DeletePolicy(policyID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.policies[policyID]; !exists {
		return fmt.Errorf("policy not found: %s", policyID)
	}

	delete(m.policies, policyID)
	log.Printf("Deleted archive policy: %s", policyID)
	return nil
}

func (m *ArchivePolicyManager) ExecutePolicy(ctx context.Context, policyID string) (*ArchiveJob, error) {
	m.mu.RLock()
	policy, exists := m.policies[policyID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	job := &ArchiveJob{
		ID:          fmt.Sprintf("job_%d", time.Now().UnixNano()),
		PolicyID:    policyID,
		Status:      "running",
		StartedAt:   time.Now(),
		Progress:    0,
	}

	executor := &PolicyExecutor{
		policyID: policyID,
		status:   "running",
		startedAt: time.Now(),
		stats:    &ExecutorStats{},
	}

	m.mu.Lock()
	m.executors[job.ID] = executor
	m.mu.Unlock()

	go m.runArchiveJob(ctx, job, policy, executor)

	return job, nil
}

func (m *ArchivePolicyManager) runArchiveJob(ctx context.Context, job *ArchiveJob, policy *ArchivePolicy, executor *PolicyExecutor) {
	defer func() {
		if r := recover(); r != nil {
			job.Status = "failed"
			job.Error = fmt.Sprintf("panic: %v", r)
			executor.status = "failed"
		}
	}()

	startTime := time.Now()

	log.Printf("Starting archive job: %s for policy: %s", job.ID, policy.Name)

	if m.db == nil {
		job.Status = "failed"
		job.Error = "database connection not available"
		executor.status = "failed"
		executor.stats.LastError = job.Error
		return
	}

	var totalCount int64
	countQuery := fmt.Sprintf("SELECT COUNT(*) FROM %s WHERE %s", policy.TableName, policy.Condition)
	if err := m.db.WithContext(ctx).Raw(countQuery).Scan(&totalCount).Error; err != nil {
		job.Status = "failed"
		job.Error = fmt.Sprintf("failed to count records: %v", err)
		executor.status = "failed"
		executor.stats.LastError = job.Error
		return
	}

	job.TotalRecords = totalCount

	if totalCount == 0 {
		job.Status = "completed"
		job.CompletedAt = time.Now()
		job.Progress = 100
		executor.status = "completed"
		return
	}

	batchSize := int64(policy.BatchSize)
	processedCount := int64(0)

	archiveTableName := fmt.Sprintf("archive_%s", policy.TableName)

	for processedCount < totalCount {
		select {
		case <-ctx.Done():
			job.Status = "cancelled"
			job.Error = "job cancelled"
			executor.status = "cancelled"
			return
		default:
		}

		var batchIDs []interface{}
		selectQuery := fmt.Sprintf(
			"SELECT id FROM %s WHERE %s ORDER BY id LIMIT %d OFFSET %d",
			policy.TableName, policy.Condition, batchSize, processedCount,
		)
		if err := m.db.WithContext(ctx).Raw(selectQuery).Scan(&batchIDs).Error; err != nil {
			executor.stats.FailedCount += batchSize
			executor.stats.LastError = err.Error()
			continue
		}

		if len(batchIDs) == 0 {
			break
		}

		tx := m.db.WithContext(ctx).Begin()
		if tx.Error != nil {
			executor.stats.FailedCount += int64(len(batchIDs))
			executor.stats.LastError = tx.Error.Error()
			continue
		}

		insertSQL := fmt.Sprintf(
			"INSERT INTO %s SELECT * FROM %s WHERE id IN ?",
			archiveTableName, policy.TableName,
		)
		if err := tx.Exec(insertSQL, batchIDs).Error; err != nil {
			tx.Rollback()
			executor.stats.FailedCount += int64(len(batchIDs))
			executor.stats.LastError = err.Error()
			continue
		}

		deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE id IN ?", policy.TableName)
		if err := tx.Exec(deleteSQL, batchIDs).Error; err != nil {
			tx.Rollback()
			executor.stats.FailedCount += int64(len(batchIDs))
			executor.stats.LastError = err.Error()
			continue
		}

		if err := tx.Commit().Error; err != nil {
			executor.stats.FailedCount += int64(len(batchIDs))
			executor.stats.LastError = err.Error()
			continue
		}

		processedCount += int64(len(batchIDs))
		executor.stats.ProcessedRecords = processedCount
		executor.stats.SuccessCount += int64(len(batchIDs))

		job.ProcessedRecords = processedCount
		job.Progress = float64(processedCount) / float64(totalCount) * 100

		log.Printf("Archive job %s progress: %.1f%% (%d/%d)", job.ID, job.Progress, processedCount, totalCount)
	}

	job.Status = "completed"
	job.CompletedAt = time.Now()
	job.Progress = 100
	executor.status = "completed"
	executor.stats.Duration = time.Since(startTime)

	m.mu.Lock()
	if policy.LastExecuted != nil {
		policy.LastExecuted = &startTime
	} else {
		policy.LastExecuted = &startTime
	}
	policy.TotalArchived += executor.stats.SuccessCount
	m.mu.Unlock()

	log.Printf("Archive job %s completed: processed %d records in %v",
		job.ID, executor.stats.SuccessCount, executor.stats.Duration)
}

func (m *ArchivePolicyManager) GetJobStatus(jobID string) (*ArchiveJob, error) {
	m.mu.RLock()
	executor, exists := m.executors[jobID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("job not found: %s", jobID)
	}

	job := &ArchiveJob{
		ID: jobID,
	}

	switch executor.status {
	case "running":
		job.Status = "running"
	case "completed":
		job.Status = "completed"
	case "failed":
		job.Status = "failed"
		job.Error = executor.stats.LastError
	case "cancelled":
		job.Status = "cancelled"
	}

	job.ProcessedRecords = executor.stats.ProcessedRecords
	job.TotalRecords = executor.stats.ProcessedRecords

	return job, nil
}

func (m *ArchivePolicyManager) CancelJob(jobID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	executor, exists := m.executors[jobID]
	if !exists {
		return fmt.Errorf("job not found: %s", jobID)
	}

	if executor.status == "running" {
		executor.status = "cancelled"
		log.Printf("Cancelled archive job: %s", jobID)
	}

	return nil
}

func (m *ArchivePolicyManager) GetPolicyStats(policyID string) (map[string]interface{}, error) {
	m.mu.RLock()
	policy, exists := m.policies[policyID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	return map[string]interface{}{
		"policy_id":      policy.ID,
		"policy_name":    policy.Name,
		"total_archived": policy.TotalArchived,
		"total_restored": policy.TotalRestored,
		"last_executed":  policy.LastExecuted,
		"next_execution": policy.NextExecution,
		"is_active":      policy.IsActive,
	}, nil
}

func (m *ArchivePolicyManager) RestoreFromArchive(ctx context.Context, policyID string, recordIDs []interface{}) (int64, error) {
	m.mu.RLock()
	policy, exists := m.policies[policyID]
	m.mu.RUnlock()

	if !exists {
		return 0, fmt.Errorf("policy not found: %s", policyID)
	}

	if m.db == nil {
		return 0, fmt.Errorf("database connection not available")
	}

	if len(recordIDs) == 0 {
		return 0, nil
	}

	archiveTableName := fmt.Sprintf("archive_%s", policy.TableName)

	tx := m.db.WithContext(ctx).Begin()
	if tx.Error != nil {
		return 0, tx.Error
	}

	insertSQL := fmt.Sprintf("INSERT INTO %s SELECT * FROM %s WHERE id IN ?", policy.TableName, archiveTableName)
	if err := tx.Exec(insertSQL, recordIDs).Error; err != nil {
		tx.Rollback()
		return 0, err
	}

	deleteSQL := fmt.Sprintf("DELETE FROM %s WHERE id IN ?", archiveTableName)
	if err := tx.Exec(deleteSQL, recordIDs).Error; err != nil {
		log.Printf("Warning: failed to delete from archive: %v", err)
	}

	if err := tx.Commit().Error; err != nil {
		return 0, err
	}

	restoredCount := int64(len(recordIDs))

	m.mu.Lock()
	policy.TotalRestored += restoredCount
	m.mu.Unlock()

	log.Printf("Restored %d records from archive for policy: %s", restoredCount, policy.Name)
	return restoredCount, nil
}

func (m *ArchivePolicyManager) loadPolicies() {
	m.mu.Lock()
	defer m.mu.Unlock()

	defaultPolicies := []*ArchivePolicy{
		{
			ID:              "policy_verification",
			Name:            "Verification Archive",
			TableName:       "verifications",
			Condition:       "created_at < NOW() - INTERVAL '30 days'",
			TargetType:      "table",
			TargetLocation:  "archive_verifications",
			Priority:        1,
			Schedule:        "0 2 * * *",
			IsActive:        true,
			RetentionDays:   365,
			BatchSize:       1000,
			CompressionEnabled: true,
			EncryptionEnabled: false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "policy_logs",
			Name:            "Logs Archive",
			TableName:       "verification_logs",
			Condition:       "created_at < NOW() - INTERVAL '7 days'",
			TargetType:      "table",
			TargetLocation:  "archive_verification_logs",
			Priority:        2,
			Schedule:        "0 3 * * *",
			IsActive:        true,
			RetentionDays:   90,
			BatchSize:       5000,
			CompressionEnabled: true,
			EncryptionEnabled: false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
		{
			ID:              "policy_behavior",
			Name:            "Behavior Data Archive",
			TableName:       "behavior_data",
			Condition:       "created_at < NOW() - INTERVAL '14 days'",
			TargetType:      "table",
			TargetLocation:  "archive_behavior_data",
			Priority:        3,
			Schedule:        "0 4 * * *",
			IsActive:        true,
			RetentionDays:   180,
			BatchSize:       2000,
			CompressionEnabled: true,
			EncryptionEnabled: false,
			CreatedAt:       time.Now(),
			UpdatedAt:       time.Now(),
		},
	}

	for _, p := range defaultPolicies {
		m.policies[p.ID] = p
	}

	log.Printf("Loaded %d default archive policies", len(defaultPolicies))
}

func (m *ArchivePolicyManager) startScheduler() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		m.checkAndExecuteScheduledPolicies()
	}
}

func (m *ArchivePolicyManager) checkAndExecuteScheduledPolicies() {
	m.mu.RLock()
	policies := make([]*ArchivePolicy, 0)
	for _, p := range m.policies {
		if p.IsActive {
			policies = append(policies, p)
		}
	}
	m.mu.RUnlock()

	now := time.Now()

	for _, policy := range policies {
		if policy.NextExecution == nil {
			nextExec := m.calculateNextExecution(policy.Schedule)
			policy.NextExecution = &nextExec
		}

		if now.After(*policy.NextExecution) {
			log.Printf("Executing scheduled policy: %s", policy.Name)
			go func(p *ArchivePolicy) {
				ctx, cancel := context.WithTimeout(context.Background(), 1*time.Hour)
				defer cancel()

				_, err := m.ExecutePolicy(ctx, p.ID)
				if err != nil {
					log.Printf("Failed to execute policy %s: %v", p.Name, err)
				}

				nextExec := m.calculateNextExecution(p.Schedule)
				p.NextExecution = &nextExec
			}(policy)
		}
	}
}

func (m *ArchivePolicyManager) calculateNextExecution(schedule string) time.Time {
	now := time.Now()

	switch schedule {
	case "0 2 * * *":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 2, 0, 0, 0, now.Location())
	case "0 3 * * *":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 3, 0, 0, 0, now.Location())
	case "0 4 * * *":
		tomorrow := now.AddDate(0, 0, 1)
		return time.Date(tomorrow.Year(), tomorrow.Month(), tomorrow.Day(), 4, 0, 0, 0, now.Location())
	default:
		return now.Add(24 * time.Hour)
	}
}

func (m *ArchivePolicyManager) ValidatePolicy(policy *ArchivePolicy) error {
	if policy.TableName == "" {
		return fmt.Errorf("table name is required")
	}

	if policy.Condition == "" {
		return fmt.Errorf("archive condition is required")
	}

	if policy.RetentionDays <= 0 {
		return fmt.Errorf("retention days must be positive")
	}

	if policy.BatchSize <= 0 {
		return fmt.Errorf("batch size must be positive")
	}

	return nil
}

func (m *ArchivePolicyManager) ClonePolicy(policyID string, newName string) (*ArchivePolicy, error) {
	m.mu.RLock()
	original, exists := m.policies[policyID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("policy not found: %s", policyID)
	}

	clone := *original
	clone.ID = fmt.Sprintf("policy_%d", time.Now().UnixNano())
	clone.Name = newName
	clone.CreatedAt = time.Now()
	clone.UpdatedAt = time.Now()
	clone.LastExecuted = nil
	clone.NextExecution = nil
	clone.TotalArchived = 0
	clone.TotalRestored = 0

	if err := m.CreatePolicy(&clone); err != nil {
		return nil, err
	}

	return &clone, nil
}
