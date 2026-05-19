package model

import (
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"sync"
	"time"
)

type IntegrityStatus string

const (
	IntegrityStatusOK       IntegrityStatus = "ok"
	IntegrityStatusModified IntegrityStatus = "modified"
	IntegrityStatusTampered IntegrityStatus = "tampered"
	IntegrityStatusUnknown  IntegrityStatus = "unknown"
)

type IntegrityViolationType string

const (
	ViolationCodeHashMismatch    IntegrityViolationType = "code_hash_mismatch"
	ViolationDynamicCodeLoad     IntegrityViolationType = "dynamic_code_load"
	ViolationMemoryModification  IntegrityViolationType = "memory_modification"
	ViolationFunctionHook       IntegrityViolationType = "function_hook"
	ViolationPrototypeModification IntegrityViolationType = "prototype_modification"
	ViolationObjectPropertyChange IntegrityViolationType = "object_property_change"
)

type IntegrityViolation struct {
	ID            int64                   `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID     string                  `json:"session_id" gorm:"size:100;index"`
	Type          IntegrityViolationType  `json:"type" gorm:"size:50;index"`
	Severity      int                     `json:"severity" gorm:"index"`
	Target        string                  `json:"target" gorm:"size:500"`
	ExpectedHash  string                  `json:"expected_hash" gorm:"size:64"`
	ActualHash    string                  `json:"actual_hash" gorm:"size:64"`
	StackTrace    string                  `json:"stack_trace" gorm:"type:text"`
	Timestamp     time.Time               `json:"timestamp" gorm:"index"`
	IPAddress     string                  `json:"ip_address" gorm:"size:50"`
	UserAgent     string                  `json:"user_agent" gorm:"size:500"`
	IsResolved    bool                    `json:"is_resolved" gorm:"default:false"`
	ResolvedAt    *time.Time              `json:"resolved_at,omitempty"`
	Metadata      string                  `json:"metadata" gorm:"type:text"`
}

type IntegrityCheckResult struct {
	IsValid        bool                    `json:"is_valid"`
	Status         IntegrityStatus         `json:"status"`
	Violations     []IntegrityViolation    `json:"violations"`
	CheckedAt      time.Time               `json:"checked_at"`
	Duration       time.Duration           `json:"duration"`
	HashValue      string                  `json:"hash_value"`
	TargetName     string                  `json:"target_name"`
	Metadata       map[string]interface{}  `json:"metadata,omitempty"`
}

type CodeIntegrityRecord struct {
	ID           int64     `json:"id" gorm:"primaryKey;autoIncrement"`
	TargetName    string    `json:"target_name" gorm:"size:100;uniqueIndex"`
	TargetType    string    `json:"target_type" gorm:"size:50"`
	OriginalHash  string    `json:"original_hash" gorm:"size:64;index"`
	CurrentHash   string    `json:"current_hash" gorm:"size:64"`
	FilePath      string    `json:"file_path" gorm:"size:500"`
	FileSize      int64     `json:"file_size"`
	Version       string    `json:"version" gorm:"size:50"`
	IsActive      bool      `json:"is_active" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at" gorm:"index"`
	UpdatedAt     time.Time `json:"updated_at"`
	LastCheckAt   *time.Time `json:"last_check_at,omitempty"`
	ViolationCount int      `json:"violation_count" gorm:"default:0"`
}

type MemoryRegion struct {
	Address uintptr `json:"address"`
	Size    uintptr `json:"size"`
	IsReadable bool  `json:"is_readable"`
	IsWritable bool  `json:"is_writable"`
	IsExecutable bool `json:"is_executable"`
	Protection int    `json:"protection"`
}

type FunctionIntegrity struct {
	Name           string    `json:"name"`
	OriginalHash   string    `json:"original_hash"`
	CurrentHash    string    `json:"current_hash"`
	Address        uintptr   `json:"address"`
	IsHooked       bool      `json:"is_hooked"`
	IsNative       bool      `json:"is_native"`
	LastCheckedAt  time.Time `json:"last_checked_at"`
}

type IntegrityConfig struct {
	EnableHashCheck           bool          `json:"enable_hash_check"`
	EnableDynamicCodeCheck    bool          `json:"enable_dynamic_code_check"`
	EnableMemoryCheck         bool          `json:"enable_memory_check"`
	EnableFunctionHookCheck   bool          `json:"enable_function_hook_check"`
	EnablePrototypeCheck      bool          `json:"enable_prototype_check"`
	CheckInterval             time.Duration `json:"check_interval"`
	AlertThreshold            int           `json:"alert_threshold"`
	AutoProtect               bool          `json:"auto_protect"`
	ReportEndpoint            string        `json:"report_endpoint"`
	AlertChannels             []string      `json:"alert_channels"`
}

type IntegrityAlert struct {
	ID          string                    `json:"id"`
	Type        IntegrityViolationType    `json:"type"`
	Severity    int                       `json:"severity"`
	Message     string                    `json:"message"`
	Target      string                    `json:"target"`
	SessionID   string                    `json:"session_id"`
	IPAddress   string                    `json:"ip_address"`
	Timestamp   time.Time                 `json:"timestamp"`
	Metadata    map[string]interface{}    `json:"metadata,omitempty"`
	IsSent      bool                      `json:"is_sent"`
	SentAt      *time.Time                `json:"sent_at,omitempty"`
}

type IntegrityStats struct {
	TotalChecks      int64                  `json:"total_checks"`
	PassedChecks     int64                  `json:"passed_checks"`
	FailedChecks     int64                  `json:"failed_checks"`
	TotalViolations  int64                  `json:"total_violations"`
	ActiveAlerts     int                    `json:"active_alerts"`
	LastCheckAt      *time.Time             `json:"last_check_at,omitempty"`
	LastViolationAt  *time.Time             `json:"last_violation_at,omitempty"`
	CheckDuration    time.Duration          `json:"avg_check_duration"`
	ViolationByType  map[string]int64       `json:"violation_by_type"`
}

func NewIntegrityCheckResult() *IntegrityCheckResult {
	return &IntegrityCheckResult{
		IsValid:     true,
		Status:      IntegrityStatusOK,
		Violations:  make([]IntegrityViolation, 0),
		CheckedAt:   time.Now(),
		Metadata:    make(map[string]interface{}),
	}
}

func (r *IntegrityCheckResult) AddViolation(violation IntegrityViolation) {
	r.Violations = append(r.Violations, violation)
	r.IsValid = false
	r.Status = IntegrityStatusModified
}

func (r *IntegrityCheckResult) CalculateHash(data []byte) string {
	hash := sha256.Sum256(data)
	return hex.EncodeToString(hash[:])
}

func (r *IntegrityCheckResult) SetMetadata(key string, value interface{}) {
	r.Metadata[key] = value
}

func (v *IntegrityViolation) SetMetadata(data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	v.Metadata = string(jsonData)
	return nil
}

func (v *IntegrityViolation) GetMetadata() (map[string]interface{}, error) {
	if v.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var data map[string]interface{}
	err := json.Unmarshal([]byte(v.Metadata), &data)
	return data, err
}

func NewIntegrityConfig() *IntegrityConfig {
	return &IntegrityConfig{
		EnableHashCheck:          true,
		EnableDynamicCodeCheck:   true,
		EnableMemoryCheck:       true,
		EnableFunctionHookCheck: true,
		EnablePrototypeCheck:    true,
		CheckInterval:           30 * time.Second,
		AlertThreshold:          3,
		AutoProtect:             true,
		AlertChannels:           []string{"log", "webhook"},
	}
}

type IntegrityChecker interface {
	CheckIntegrity() (*IntegrityCheckResult, error)
	VerifyHash(data []byte, expectedHash string) bool
	GenerateHash(data []byte) string
	AddViolationCallback(func(IntegrityViolation))
	Start() error
	Stop()
}

type IntegrityReport struct {
	SessionID      string                `json:"session_id"`
	CheckResults   []*IntegrityCheckResult `json:"check_results"`
	OverallStatus  IntegrityStatus        `json:"overall_status"`
	TotalDuration  time.Duration          `json:"total_duration"`
	GeneratedAt    time.Time              `json:"generated_at"`
	Summary        *IntegrityStats       `json:"summary,omitempty"`
}

type ThreadSafeIntegrityChecker struct {
	mu      sync.RWMutex
	records map[string]*CodeIntegrityRecord
	config  *IntegrityConfig
	stats   *IntegrityStats
}

func NewThreadSafeIntegrityChecker() *ThreadSafeIntegrityChecker {
	return &ThreadSafeIntegrityChecker{
		records: make(map[string]*CodeIntegrityRecord),
		config:  NewIntegrityConfig(),
		stats: &IntegrityStats{
			ViolationByType: make(map[string]int64),
		},
	}
}

func (t *ThreadSafeIntegrityChecker) AddRecord(record *CodeIntegrityRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[record.TargetName] = record
}

func (t *ThreadSafeIntegrityChecker) GetRecord(targetName string) (*CodeIntegrityRecord, bool) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	record, exists := t.records[targetName]
	return record, exists
}

func (t *ThreadSafeIntegrityChecker) UpdateRecord(targetName string, record *CodeIntegrityRecord) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.records[targetName] = record
}

func (t *ThreadSafeIntegrityChecker) GetAllRecords() []*CodeIntegrityRecord {
	t.mu.RLock()
	defer t.mu.RUnlock()
	records := make([]*CodeIntegrityRecord, 0, len(t.records))
	for _, record := range t.records {
		records = append(records, record)
	}
	return records
}

func (t *ThreadSafeIntegrityChecker) DeleteRecord(targetName string) {
	t.mu.Lock()
	defer t.mu.Unlock()
	delete(t.records, targetName)
}

func (t *ThreadSafeIntegrityChecker) UpdateStats(violationType string, isViolation bool) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.stats.TotalChecks++
	if isViolation {
		t.stats.FailedChecks++
		t.stats.TotalViolations++
		t.stats.ViolationByType[violationType]++
		now := time.Now()
		t.stats.LastViolationAt = &now
	} else {
		t.stats.PassedChecks++
	}
}

func (t *ThreadSafeIntegrityChecker) GetStats() *IntegrityStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.stats
}
