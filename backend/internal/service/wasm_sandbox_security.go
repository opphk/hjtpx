package service

import (
	"context"
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type WASMSandboxSecurity struct {
	ctx             context.Context
	cancel          context.CancelFunc
	mu              sync.RWMutex
	isRunning        bool
	policies         map[string]*SandboxPolicy
	sessions         map[string]*SandboxSession
	auditLog         *AuditLogger
	resourceLimits   *ResourceLimits
	enableIsolation  bool
	enableMonitoring bool
}

type SandboxPolicy struct {
	ID               string
	Name             string
	MaxMemory        uint64
	MaxCPU           time.Duration
	MaxExecutionTime time.Duration
	MaxNetworkCalls  int
	AllowedImports   []string
	BlockedImports   []string
	MaxTableSize     int
	MaxMemoryPages   int
	EnableStackTrace bool
	EnableDebug      bool
}

type SandboxSession struct {
	ID            string
	PolicyID      string
	CreatedAt     time.Time
	LastAccess    time.Time
	AccessCount   int64
	MemoryUsage   uint64
	CPUUsage      time.Duration
	Executions    int64
	Violations    int64
	Isolated      bool
	State         SessionState
}

type SessionState int

const (
	SessionActive SessionState = iota
	SessionPaused
	SessionTerminated
)

type AuditLogger struct {
	mu      sync.RWMutex
	entries []*AuditEntry
	maxSize int
}

type AuditEntry struct {
	Timestamp   time.Time
	SessionID   string
	Action      string
	Resource    string
	Allowed     bool
	Details     string
	RiskLevel   RiskLevel
}

type RiskLevel int

const (
	RiskLow RiskLevel = iota
	RiskMedium
	RiskHigh
	RiskCritical
)

type ResourceLimits struct {
	mu                sync.RWMutex
	defaultMemory     uint64
	defaultCPU        time.Duration
	maxConcurrent     int
	currentConcurrent int
	enforcementMode   EnforcementMode
}

type EnforcementMode int

const (
	EnforceHard EnforcementMode = iota
	EnforceSoft
	EnforceAudit
)

type WASMModuleInfo struct {
	ID            string
	Name          string
	Hash          string
	Size          uint64
	Trusted       bool
	Signature     []byte
	Imports       []string
	Exports       []string
	MemoryPages   int
	TableSize     int
	CompileTime   time.Duration
}

type ExecutionResult struct {
	Success      bool
	Output       []byte
	ErrorMessage string
	ExecTime     time.Duration
	MemoryUsed   uint64
	Violations   []Violation
}

type Violation struct {
	Type       ViolationType
	PolicyID   string
	SessionID  string
	Details    string
	Timestamp  time.Time
	Severity   Severity
}

type ViolationType int

const (
	ViolationMemory ViolationType = iota
	ViolationCPU
	ViolationTime
	ViolationNetwork
	ViolationImport
	ViolationExport
	ViolationStack
)

type Severity int

const (
	SeverityWarning Severity = iota
	SeverityError
	SeverityCritical
)

type SecurityConfig struct {
	EnableSeccomp    bool
	EnableLandlock   bool
	EnableNamespace  bool
	EnableCapability bool
}

func NewWASMSandboxSecurity() *WASMSandboxSecurity {
	ctx, cancel := context.WithCancel(context.Background())

	return &WASMSandboxSecurity{
		ctx:             ctx,
		cancel:          cancel,
		policies:        make(map[string]*SandboxPolicy),
		sessions:        make(map[string]*SandboxSession),
		auditLog:        NewAuditLogger(10000),
		resourceLimits:  NewResourceLimits(),
		enableIsolation: true,
		enableMonitoring: true,
	}
}

func NewAuditLogger(maxSize int) *AuditLogger {
	return &AuditLogger{
		entries: make([]*AuditEntry, 0, maxSize),
		maxSize: maxSize,
	}
}

func NewResourceLimits() *ResourceLimits {
	return &ResourceLimits{
		defaultMemory:     64 * 1024 * 1024, // 64MB
		defaultCPU:        100 * time.Millisecond,
		maxConcurrent:     1000,
		enforcementMode:   EnforceHard,
	}
}

func (w *WASMSandboxSecurity) Start() error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if w.isRunning {
		return nil
	}

	w.isRunning = true
	go w.monitorResources()
	go w.cleanupSessions()

	fmt.Println("[WASMSandboxSecurity] Started successfully")
	return nil
}

func (w *WASMSandboxSecurity) Stop() {
	w.mu.Lock()
	defer w.mu.Unlock()

	if !w.isRunning {
		return
	}

	w.cancel()
	w.isRunning = false
	fmt.Println("[WASMSandboxSecurity] Stopped")
}

func (w *WASMSandboxSecurity) CreatePolicy(policy *SandboxPolicy) error {
	if policy.ID == "" {
		return errors.New("policy ID is required")
	}

	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.policies[policy.ID]; exists {
		return fmt.Errorf("policy %s already exists", policy.ID)
	}

	w.policies[policy.ID] = policy
	return nil
}

func (w *WASMSandboxSecurity) GetPolicy(policyID string) (*SandboxPolicy, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	policy, exists := w.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", policyID)
	}

	return policy, nil
}

func (w *WASMSandboxSecurity) UpdatePolicy(policyID string, updates *SandboxPolicy) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	policy, exists := w.policies[policyID]
	if !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}

	if updates.MaxMemory > 0 {
		policy.MaxMemory = updates.MaxMemory
	}
	if updates.MaxCPU > 0 {
		policy.MaxCPU = updates.MaxCPU
	}
	if updates.MaxExecutionTime > 0 {
		policy.MaxExecutionTime = updates.MaxExecutionTime
	}
	if updates.MaxNetworkCalls > 0 {
		policy.MaxNetworkCalls = updates.MaxNetworkCalls
	}
	if len(updates.AllowedImports) > 0 {
		policy.AllowedImports = updates.AllowedImports
	}
	if len(updates.BlockedImports) > 0 {
		policy.BlockedImports = updates.BlockedImports
	}

	return nil
}

func (w *WASMSandboxSecurity) DeletePolicy(policyID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	if _, exists := w.policies[policyID]; !exists {
		return fmt.Errorf("policy %s not found", policyID)
	}

	delete(w.policies, policyID)
	return nil
}

func (w *WASMSandboxSecurity) CreateSession(policyID string) (*SandboxSession, error) {
	w.mu.Lock()
	defer w.mu.Unlock()

	policy, exists := w.policies[policyID]
	if !exists {
		return nil, fmt.Errorf("policy %s not found", policyID)
	}

	w.resourceLimits.mu.RLock()
	if w.resourceLimits.currentConcurrent >= w.resourceLimits.maxConcurrent {
		w.resourceLimits.mu.RUnlock()
		return nil, errors.New("maximum concurrent sessions reached")
	}
	w.resourceLimits.mu.RUnlock()

	sessionID, err := generateSessionID()
	if err != nil {
		return nil, err
	}

	session := &SandboxSession{
		ID:          sessionID,
		PolicyID:    policyID,
		CreatedAt:   time.Now(),
		LastAccess:  time.Now(),
		AccessCount: 0,
		MemoryUsage: 0,
		CPUUsage:    0,
		Executions:  0,
		Isolated:    w.enableIsolation,
		State:       SessionActive,
	}

	w.sessions[sessionID] = session
	w.resourceLimits.mu.Lock()
	w.resourceLimits.currentConcurrent++
	w.resourceLimits.mu.Unlock()

	return session, nil
}

func (w *WASMSandboxSecurity) GetSession(sessionID string) (*SandboxSession, error) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	session, exists := w.sessions[sessionID]
	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	return session, nil
}

func (w *WASMSandboxSecurity) TerminateSession(sessionID string) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	session, exists := w.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.State = SessionTerminated
	delete(w.sessions, sessionID)

	w.resourceLimits.mu.Lock()
	if w.resourceLimits.currentConcurrent > 0 {
		w.resourceLimits.currentConcurrent--
	}
	w.resourceLimits.mu.Unlock()

	return nil
}

func (w *WASMSandboxSecurity) ValidateModule(info *WASMModuleInfo) (*ExecutionResult, error) {
	session, err := w.CreateSession("default")
	if err != nil {
		return nil, err
	}
	defer w.TerminateSession(session.ID)

	policy, _ := w.GetPolicy(session.PolicyID)

	result := &ExecutionResult{
		Success:    true,
		Output:     make([]byte, 0),
		Violations: make([]Violation, 0),
	}

	// Validate module size
	if policy != nil && info.Size > policy.MaxMemory {
		result.Violations = append(result.Violations, Violation{
			Type:      ViolationMemory,
			PolicyID:  policy.ID,
			SessionID: session.ID,
			Details:   fmt.Sprintf("module size %d exceeds limit %d", info.Size, policy.MaxMemory),
			Timestamp: time.Now(),
			Severity:  SeverityError,
		})
		result.Success = false
	}

	// Validate imports against blocked imports
	if policy != nil && len(policy.BlockedImports) > 0 {
		for _, imp := range info.Imports {
			for _, blocked := range policy.BlockedImports {
				if imp == blocked {
					result.Violations = append(result.Violations, Violation{
						Type:      ViolationImport,
						PolicyID:  policy.ID,
						SessionID: session.ID,
						Details:   fmt.Sprintf("import %s is blocked", imp),
						Timestamp: time.Now(),
						Severity:  SeverityCritical,
					})
					result.Success = false
				}
			}
		}
	}

	// Validate memory pages
	if policy != nil && info.MemoryPages > policy.MaxMemoryPages {
		result.Violations = append(result.Violations, Violation{
			Type:      ViolationMemory,
			PolicyID:  policy.ID,
			SessionID: session.ID,
			Details:   fmt.Sprintf("memory pages %d exceeds limit %d", info.MemoryPages, policy.MaxMemoryPages),
			Timestamp: time.Now(),
			Severity:  SeverityError,
		})
	}

	// Validate table size
	if policy != nil && info.TableSize > policy.MaxTableSize {
		result.Violations = append(result.Violations, Violation{
			Type:      ViolationImport,
			PolicyID:  policy.ID,
			SessionID: session.ID,
			Details:   fmt.Sprintf("table size %d exceeds limit %d", info.TableSize, policy.MaxTableSize),
			Timestamp: time.Now(),
			Severity:  SeverityError,
		})
	}

	return result, nil
}

func (w *WASMSandboxSecurity) ExecuteInSandbox(sessionID string, code []byte, input []byte) (*ExecutionResult, error) {
	w.mu.RLock()
	session, exists := w.sessions[sessionID]
	w.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("session %s not found", sessionID)
	}

	policy, _ := w.GetPolicy(session.PolicyID)
	start := time.Now()

	result := &ExecutionResult{
		Success:    true,
		Output:     make([]byte, 0),
		Violations: make([]Violation, 0),
	}

	// Update session stats
	session.LastAccess = time.Now()
	session.AccessCount++
	session.Executions++

	// Check execution time
	if policy != nil && policy.MaxExecutionTime > 0 {
		if time.Since(start) > policy.MaxExecutionTime {
			result.Violations = append(result.Violations, Violation{
				Type:      ViolationTime,
				PolicyID:  policy.ID,
				SessionID: session.ID,
				Details:   "execution time exceeded limit",
				Timestamp: time.Now(),
				Severity:  SeverityError,
			})
			result.Success = false
		}
	}

	// Check memory usage
	if policy != nil && session.MemoryUsage > policy.MaxMemory {
		result.Violations = append(result.Violations, Violation{
			Type:      ViolationMemory,
			PolicyID:  policy.ID,
			SessionID: session.ID,
			Details:   fmt.Sprintf("memory usage %d exceeds limit %d", session.MemoryUsage, policy.MaxMemory),
			Timestamp: time.Now(),
			Severity:  SeverityCritical,
		})
		result.Success = false
		session.Violations++
	}

	// Check CPU usage
	if policy != nil && session.CPUUsage > policy.MaxCPU {
		result.Violations = append(result.Violations, Violation{
			Type:      ViolationCPU,
			PolicyID:  policy.ID,
			SessionID: session.ID,
			Details:   fmt.Sprintf("CPU usage %v exceeds limit %v", session.CPUUsage, policy.MaxCPU),
			Timestamp: time.Now(),
			Severity:  SeverityWarning,
		})
	}

	result.ExecTime = time.Since(start)

	// Log audit entry
	w.auditLog.Log(&AuditEntry{
		Timestamp:  time.Now(),
		SessionID: sessionID,
		Action:    "EXECUTE",
		Allowed:   result.Success,
		RiskLevel: w.assessRiskLevel(result.Violations),
	})

	return result, nil
}

func (w *WASMSandboxSecurity) assessRiskLevel(violations []Violation) RiskLevel {
	if len(violations) == 0 {
		return RiskLow
	}

	maxSeverity := SeverityWarning
	for _, v := range violations {
		if v.Severity > maxSeverity {
			maxSeverity = v.Severity
		}
	}

	switch maxSeverity {
	case SeverityWarning:
		return RiskMedium
	case SeverityError:
		return RiskHigh
	case SeverityCritical:
		return RiskCritical
	default:
		return RiskLow
	}
}

func (w *WASMSandboxSecurity) LogAudit(entry *AuditEntry) {
	w.auditLog.Log(entry)
}

func (a *AuditLogger) Log(entry *AuditEntry) {
	a.mu.Lock()
	defer a.mu.Unlock()

	if len(a.entries) >= a.maxSize {
		a.entries = a.entries[1:]
	}

	a.entries = append(a.entries, entry)
}

func (a *AuditLogger) GetEntries(since time.Time, limit int) []*AuditEntry {
	a.mu.RLock()
	defer a.mu.RUnlock()

	var results []*AuditEntry
	for i := len(a.entries) - 1; i >= 0 && len(results) < limit; i-- {
		if a.entries[i].Timestamp.After(since) {
			results = append(results, a.entries[i])
		}
	}

	return results
}

func (w *WASMSandboxSecurity) GetAuditLog(sessionID string, limit int) []*AuditEntry {
	entries := w.auditLog.GetEntries(time.Time{}, limit)

	var filtered []*AuditEntry
	for _, e := range entries {
		if e.SessionID == sessionID {
			filtered = append(filtered, e)
		}
	}

	return filtered
}

func (w *WASMSandboxSecurity) monitorResources() {
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.mu.RLock()
			activeSessions := len(w.sessions)
			w.mu.RUnlock()

			if w.enableMonitoring {
				fmt.Printf("[WASMSandboxSecurity] Monitoring: %d active sessions\n", activeSessions)
			}
		}
	}
}

func (w *WASMSandboxSecurity) cleanupSessions() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-w.ctx.Done():
			return
		case <-ticker.C:
			w.mu.Lock()
			now := time.Now()
			for id, session := range w.sessions {
				if session.State == SessionTerminated ||
					now.Sub(session.LastAccess) > 30*time.Minute {
					delete(w.sessions, id)
					w.resourceLimits.mu.Lock()
					if w.resourceLimits.currentConcurrent > 0 {
						w.resourceLimits.currentConcurrent--
					}
					w.resourceLimits.mu.Unlock()
				}
			}
			w.mu.Unlock()
		}
	}
}

func (w *WASMSandboxSecurity) GetStats() map[string]interface{} {
	w.mu.RLock()
	activeSessions := len(w.sessions)
	totalPolicies := len(w.policies)
	w.mu.RUnlock()

	w.resourceLimits.mu.RLock()
	concurrent := w.resourceLimits.currentConcurrent
	maxConcurrent := w.resourceLimits.maxConcurrent
	w.resourceLimits.mu.RUnlock()

	return map[string]interface{}{
		"is_running":         w.isRunning,
		"active_sessions":    activeSessions,
		"total_policies":     totalPolicies,
		"current_concurrent": concurrent,
		"max_concurrent":     maxConcurrent,
		"isolation_enabled":  w.enableIsolation,
		"monitoring_enabled": w.enableMonitoring,
	}
}

func (w *WASMSandboxSecurity) SetResourceLimits(memory uint64, cpu time.Duration, maxConcurrent int) {
	w.resourceLimits.mu.Lock()
	defer w.resourceLimits.mu.Unlock()

	if memory > 0 {
		w.resourceLimits.defaultMemory = memory
	}
	if cpu > 0 {
		w.resourceLimits.defaultCPU = cpu
	}
	if maxConcurrent > 0 {
		w.resourceLimits.maxConcurrent = maxConcurrent
	}
}

func (w *WASMSandboxSecurity) IsolateSession(sessionID string, isolated bool) error {
	w.mu.Lock()
	defer w.mu.Unlock()

	session, exists := w.sessions[sessionID]
	if !exists {
		return fmt.Errorf("session %s not found", sessionID)
	}

	session.Isolated = isolated
	return nil
}

func (w *WASMSandboxSecurity) CreateDefaultPolicies() {
	defaultPolicies := []*SandboxPolicy{
		{
			ID:                "strict",
			Name:              "Strict Security Policy",
			MaxMemory:         32 * 1024 * 1024,
			MaxCPU:            50 * time.Millisecond,
			MaxExecutionTime:  100 * time.Millisecond,
			MaxNetworkCalls:   0,
			MaxMemoryPages:    16,
			MaxTableSize:      100,
			EnableStackTrace:  false,
			EnableDebug:       false,
		},
		{
			ID:                "moderate",
			Name:              "Moderate Policy",
			MaxMemory:         64 * 1024 * 1024,
			MaxCPU:            100 * time.Millisecond,
			MaxExecutionTime:  500 * time.Millisecond,
			MaxNetworkCalls:   10,
			MaxMemoryPages:    32,
			MaxTableSize:      500,
			EnableStackTrace:  true,
			EnableDebug:       false,
		},
		{
			ID:                "permissive",
			Name:              "Permissive Policy",
			MaxMemory:         128 * 1024 * 1024,
			MaxCPU:            200 * time.Millisecond,
			MaxExecutionTime:  2 * time.Second,
			MaxNetworkCalls:   100,
			MaxMemoryPages:    64,
			MaxTableSize:      1000,
			EnableStackTrace:  true,
			EnableDebug:       true,
		},
	}

	for _, p := range defaultPolicies {
		w.CreatePolicy(p)
	}
}

func generateSessionID() (string, error) {
	b := make([]byte, 16)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return fmt.Sprintf("sess_%x", b), nil
}

type WASMValidator struct {
	mu          sync.RWMutex
	validators  map[string]ValidationFunc
}

type ValidationFunc func([]byte) (bool, error)

func NewWASMValidator() *WASMValidator {
	return &WASMValidator{
		validators: make(map[string]ValidationFunc),
	}
}

func (v *WASMValidator) RegisterValidator(name string, fn ValidationFunc) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.validators[name] = fn
}

func (v *WASMValidator) Validate(moduleData []byte) (bool, []error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	var errs []error
	allValid := true

	for name, fn := range v.validators {
		valid, err := fn(moduleData)
		if err != nil {
			errs = append(errs, fmt.Errorf("validator %s: %w", name, err))
			allValid = false
		} else if !valid {
			errs = append(errs, fmt.Errorf("validator %s: validation failed", name))
			allValid = false
		}
	}

	return allValid, errs
}

func (v *WASMValidator) ValidateMagicNumber(data []byte) (bool, error) {
	if len(data) < 4 {
		return false, errors.New("data too short for magic number")
	}

	magic := []byte{0x00, 0x61, 0x73, 0x6d}
	if data[0] != magic[0] || data[1] != magic[1] || data[2] != magic[2] || data[3] != magic[3] {
		return false, errors.New("invalid WASM magic number")
	}

	return true, nil
}

func (v *WASMValidator) ValidateVersion(data []byte) (bool, error) {
	if len(data) < 8 {
		return false, errors.New("data too short for version")
	}

	version := binary.LittleEndian.Uint32(data[4:8])
	if version != 1 && version != 2 {
		return false, fmt.Errorf("unsupported WASM version: %d", version)
	}

	return true, nil
}

func (v *WASMValidator) ValidateSectionOrder(data []byte) (bool, error) {
	if len(data) < 8 {
		return false, errors.New("data too short for section validation")
	}

	sections := make(map[byte]bool)
	pos := 8

	for pos < len(data) {
		if pos >= len(data) {
			break
		}

		sectionID := data[pos]
		pos++

		size := 0
		for pos < len(data) {
			b := data[pos]
			pos++
			size = (size << 7) | int(b&0x7F)
			if b&0x80 == 0 {
				break
			}
		}

		if sections[sectionID] && sectionID <= 12 {
			return false, fmt.Errorf("duplicate section ID: %d", sectionID)
		}
		sections[sectionID] = true

		pos += size
	}

	return true, nil
}
