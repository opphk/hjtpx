package model

import (
	"encoding/json"
	"sync"
	"time"
)

type DebugDetectionType string

const (
	DebugTypeDevTools       DebugDetectionType = "devtools"
	DebugTypeDebugger       DebugDetectionType = "debugger"
	DebugTypeBreakpoint     DebugDetectionType = "breakpoint"
	DebugTypeTimingAnomaly  DebugDetectionType = "timing_anomaly"
	DebugTypeConsoleActivity DebugDetectionType = "console_activity"
	DebugTypeMemoryAccess   DebugDetectionType = "memory_access"
	DebugTypeCallStackDepth DebugDetectionType = "call_stack_depth"
)

type DebugSeverity int

const (
	SeverityInfo     DebugSeverity = 1
	SeverityWarning  DebugSeverity = 3
	SeverityMedium   DebugSeverity = 5
	SeverityHigh     DebugSeverity = 7
	SeverityCritical DebugSeverity = 9
)

type DebugDetectionEvent struct {
	ID            int64              `json:"id" gorm:"primaryKey;autoIncrement"`
	SessionID     string             `json:"session_id" gorm:"size:100;index"`
	Type          DebugDetectionType `json:"type" gorm:"size:50;index"`
	Severity      DebugSeverity      `json:"severity" gorm:"index"`
	DetectionTime time.Time          `json:"detection_time" gorm:"index"`
	IPAddress     string             `json:"ip_address" gorm:"size:50"`
	UserAgent     string             `json:"user_agent" gorm:"size:500"`
	ClientIP      string             `json:"client_ip" gorm:"size:50"`
	Fingerprint   string             `json:"fingerprint" gorm:"size:64"`
	Evidence      string             `json:"evidence" gorm:"type:text"`
	RequestData   string             `json:"request_data" gorm:"type:text"`
	IsBlocked     bool               `json:"is_blocked" gorm:"default:false"`
	BlockReason   string             `json:"block_reason" gorm:"size:200"`
	Metadata      string             `json:"metadata" gorm:"type:text"`
	ResponseDelay int64              `json:"response_delay_ms"`
	Headers       string             `json:"headers" gorm:"type:text"`
}

type DebugDetectionResult struct {
	IsDetected         bool                   `json:"is_detected"`
	Detections         []DebugDetectionEvent   `json:"detections"`
	TotalSeverity      int                    `json:"total_severity"`
	HighestSeverity    DebugSeverity          `json:"highest_severity"`
	DetectionCount     int                    `json:"detection_count"`
	DetectionTypes     map[DebugDetectionType]int `json:"detection_types"`
	ShouldBlock        bool                   `json:"should_block"`
	Recommendation     string                 `json:"recommendation"`
	RiskScore          float64                `json:"risk_score"`
	SessionID          string                 `json:"session_id"`
	IPAddress          string                 `json:"ip_address"`
	UserAgent          string                 `json:"user_agent"`
	Timestamp          time.Time              `json:"timestamp"`
	ResponseTimeMs     int64                  `json:"response_time_ms"`
	Metadata           map[string]interface{} `json:"metadata,omitempty"`
}

type DebugDetectorConfig struct {
	Enabled                 bool                  `json:"enabled"`
	CheckDevTools           bool                  `json:"check_devtools"`
	CheckDebugger          bool                  `json:"check_debugger"`
	CheckBreakpoints       bool                  `json:"check_breakpoints"`
	CheckTimingAnomaly     bool                  `json:"check_timing_anomaly"`
	CheckConsoleActivity   bool                  `json:"check_console_activity"`
	CheckMemoryAccess      bool                  `json:"check_memory_access"`
	CheckCallStackDepth    bool                  `json:"check_call_stack_depth"`
	BlockOnDetection       bool                  `json:"block_on_detection"`
	LogDetections          bool                  `json:"log_detections"`
	MaxDetectionsPerSession int                  `json:"max_detections_per_session"`
	DetectionThreshold      int                  `json:"detection_threshold"`
	TimeWindow             time.Duration         `json:"time_window"`
	AutoBlockDuration      time.Duration         `json:"auto_block_duration"`
	MinSeverityToBlock     DebugSeverity         `json:"min_severity_to_block"`
	EnableTimingCheck      bool                  `json:"enable_timing_check"`
	MinResponseTimeMs      int64                 `json:"min_response_time_ms"`
	MaxResponseTimeMs      int64                 `json:"max_response_time_ms"`
	EnableBreakpointCheck  bool                  `json:"enable_breakpoint_check"`
	BreakpointThreshold    int                  `json:"breakpoint_threshold"`
}

type DebugStats struct {
	TotalDetections     int64                          `json:"total_detections"`
	BlockedSessions     int64                          `json:"blocked_sessions"`
	ActiveSessions      int64                          `json:"active_sessions"`
	DetectionsByType    map[DebugDetectionType]int64   `json:"detections_by_type"`
	DetectionsBySeverity map[DebugSeverity]int64      `json:"detections_by_severity"`
	AvgResponseTimeMs   float64                        `json:"avg_response_time_ms"`
	MaxResponseTimeMs   int64                          `json:"max_response_time_ms"`
	LastDetectionAt     *time.Time                    `json:"last_detection_at,omitempty"`
	LastBlockedAt       *time.Time                    `json:"last_blocked_at,omitempty"`
	TopOffenders        []DebugIPStat                  `json:"top_offenders"`
}

type DebugIPStat struct {
	IPAddress       string    `json:"ip_address"`
	DetectionCount  int64     `json:"detection_count"`
	BlockCount      int64     `json:"block_count"`
	FirstSeen       time.Time `json:"first_seen"`
	LastSeen        time.Time `json:"last_seen"`
}

type TimingViolation struct {
	Type            string    `json:"type"`
	ExpectedMs      int64     `json:"expected_ms"`
	ActualMs        int64     `json:"actual_ms"`
	Deviation       float64   `json:"deviation"`
	Severity        int       `json:"severity"`
	Timestamp       time.Time `json:"timestamp"`
	Evidence        string    `json:"evidence"`
}

type BreakpointDetection struct {
	Type        string    `json:"type"`
	LineNumber  int       `json:"line_number"`
	FunctionName string   `json:"function_name"`
	Severity    int       `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	Evidence    string    `json:"evidence"`
	CallStack   []string  `json:"call_stack"`
}

type ConsoleActivity struct {
	Type        string    `json:"type"`
	Method      string    `json:"method"`
	Arguments   []string  `json:"arguments"`
	Severity    int       `json:"severity"`
	Timestamp   time.Time `json:"timestamp"`
	IsOverridden bool     `json:"is_overridden"`
	Evidence    string    `json:"evidence"`
}

type CallStackAnalysis struct {
	Depth         int       `json:"depth"`
	MaxDepth      int       `json:"max_depth"`
	IsSuspicious  bool      `json:"is_suspicious"`
	Severity      int       `json:"severity"`
	Functions     []string  `json:"functions"`
	Timestamp     time.Time `json:"timestamp"`
	Evidence      string    `json:"evidence"`
}

type DevToolsIndicators struct {
	IsDevToolsOpen      bool   `json:"is_devtools_open"`
	DetectionMethod     string `json:"detection_method"`
	WindowSizeChanged   bool   `json:"window_size_changed"`
	OrientationChanged  bool   `json:"orientation_changed"`
	Severity            int    `json:"severity"`
	Evidence           string `json:"evidence"`
	Timestamp          time.Time `json:"timestamp"`
}

func NewDebugDetectorConfig() *DebugDetectorConfig {
	return &DebugDetectorConfig{
		Enabled:                  true,
		CheckDevTools:            true,
		CheckDebugger:            true,
		CheckBreakpoints:         true,
		CheckTimingAnomaly:       true,
		CheckConsoleActivity:     true,
		CheckMemoryAccess:        true,
		CheckCallStackDepth:      true,
		BlockOnDetection:         false,
		LogDetections:            true,
		MaxDetectionsPerSession:  10,
		DetectionThreshold:       5,
		TimeWindow:               5 * time.Minute,
		AutoBlockDuration:        10 * time.Minute,
		MinSeverityToBlock:       SeverityHigh,
		EnableTimingCheck:        true,
		MinResponseTimeMs:        5,
		MaxResponseTimeMs:        30000,
		EnableBreakpointCheck:    true,
		BreakpointThreshold:      3,
	}
}

func NewDebugDetectionResult() *DebugDetectionResult {
	return &DebugDetectionResult{
		IsDetected:      false,
		Detections:      make([]DebugDetectionEvent, 0),
		DetectionTypes:  make(map[DebugDetectionType]int),
		ShouldBlock:     false,
		RiskScore:       0.0,
		Timestamp:       time.Now(),
		Metadata:        make(map[string]interface{}),
	}
}

func NewDebugDetectionEvent(detectionType DebugDetectionType, severity DebugSeverity) *DebugDetectionEvent {
	return &DebugDetectionEvent{
		Type:          detectionType,
		Severity:      severity,
		DetectionTime: time.Now(),
	}
}

func (r *DebugDetectionResult) AddDetection(event DebugDetectionEvent) {
	r.IsDetected = true
	r.Detections = append(r.Detections, event)
	r.TotalSeverity += int(event.Severity)
	r.DetectionCount++
	
	if event.Severity > r.HighestSeverity {
		r.HighestSeverity = event.Severity
	}
	
	r.DetectionTypes[event.Type]++
	r.RiskScore += float64(event.Severity) * 10.0
}

func (r *DebugDetectionResult) CalculateRiskScore() float64 {
	if r.DetectionCount == 0 {
		return 0.0
	}
	
	baseScore := float64(r.TotalSeverity) / float64(r.DetectionCount)
	typeMultiplier := 1.0 + float64(len(r.DetectionTypes))*0.1
	
	finalScore := baseScore * typeMultiplier
	if finalScore > 100 {
		finalScore = 100
	}
	
	return finalScore
}

func (r *DebugDetectionResult) ShouldBlockSession(threshold int) bool {
	return r.DetectionCount >= threshold
}

func (r *DebugDetectionResult) GetMostSevereDetection() *DebugDetectionEvent {
	if len(r.Detections) == 0 {
		return nil
	}
	
	maxSeverity := DebugDetectionEvent{}
	for i, detection := range r.Detections {
		if detection.Severity > maxSeverity.Severity || i == 0 {
			maxSeverity = detection
		}
	}
	
	return &maxSeverity
}

func (r *DebugDetectionResult) ToJSON() (string, error) {
	data, err := json.MarshalIndent(r, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func ParseDebugDetectionResult(data string) (*DebugDetectionResult, error) {
	var result DebugDetectionResult
	err := json.Unmarshal([]byte(data), &result)
	return &result, err
}

func (c *DebugDetectorConfig) IsValid() bool {
	if c.MaxDetectionsPerSession <= 0 {
		return false
	}
	if c.DetectionThreshold <= 0 {
		return false
	}
	if c.TimeWindow <= 0 {
		return false
	}
	if c.AutoBlockDuration <= 0 {
		return false
	}
	return true
}

func (e *DebugDetectionEvent) SetMetadata(data map[string]interface{}) error {
	jsonData, err := json.Marshal(data)
	if err != nil {
		return err
	}
	e.Metadata = string(jsonData)
	return nil
}

func (e *DebugDetectionEvent) GetMetadata() (map[string]interface{}, error) {
	if e.Metadata == "" {
		return make(map[string]interface{}), nil
	}
	var data map[string]interface{}
	err := json.Unmarshal([]byte(e.Metadata), &data)
	return data, err
}

type ThreadSafeDebugDetector struct {
	mu         sync.RWMutex
	sessions   map[string][]DebugDetectionEvent
	config     *DebugDetectorConfig
	stats      *DebugStats
	blockedIPs map[string]time.Time
}

func NewThreadSafeDebugDetector() *ThreadSafeDebugDetector {
	return &ThreadSafeDebugDetector{
		sessions:   make(map[string][]DebugDetectionEvent),
		config:     NewDebugDetectorConfig(),
		stats:      NewDebugStats(),
		blockedIPs: make(map[string]time.Time),
	}
}

func NewDebugStats() *DebugStats {
	return &DebugStats{
		DetectionsByType:     make(map[DebugDetectionType]int64),
		DetectionsBySeverity: make(map[DebugSeverity]int64),
		TopOffenders:         make([]DebugIPStat, 0),
	}
}

func (t *ThreadSafeDebugDetector) AddDetection(sessionID string, event DebugDetectionEvent) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.sessions[sessionID] = append(t.sessions[sessionID], event)
	t.stats.TotalDetections++
	t.stats.DetectionsByType[event.Type]++
	t.stats.DetectionsBySeverity[event.Severity]++
	
	now := time.Now()
	t.stats.LastDetectionAt = &now
}

func (t *ThreadSafeDebugDetector) GetSessionDetections(sessionID string) []DebugDetectionEvent {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	detections := t.sessions[sessionID]
	result := make([]DebugDetectionEvent, len(detections))
	copy(result, detections)
	return result
}

func (t *ThreadSafeDebugDetector) GetSessionDetectionCount(sessionID string) int {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	return len(t.sessions[sessionID])
}

func (t *ThreadSafeDebugDetector) BlockIP(ip string, duration time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	t.stats.BlockedSessions++
	now := time.Now()
	t.stats.LastBlockedAt = &now
	
	t.blockedIPs[ip] = time.Now().Add(duration)
}

func (t *ThreadSafeDebugDetector) IsIPBlocked(ip string) bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	if expiry, exists := t.blockedIPs[ip]; exists {
		if time.Now().Before(expiry) {
			return true
		}
	}
	return false
}

func (t *ThreadSafeDebugDetector) GetStats() *DebugStats {
	t.mu.RLock()
	defer t.mu.RUnlock()
	
	statsCopy := &DebugStats{
		TotalDetections:     t.stats.TotalDetections,
		BlockedSessions:     t.stats.BlockedSessions,
		ActiveSessions:       int64(len(t.sessions)),
		DetectionsByType:     t.stats.DetectionsByType,
		DetectionsBySeverity: t.stats.DetectionsBySeverity,
		AvgResponseTimeMs:   t.stats.AvgResponseTimeMs,
		MaxResponseTimeMs:   t.stats.MaxResponseTimeMs,
		LastDetectionAt:     t.stats.LastDetectionAt,
		LastBlockedAt:        t.stats.LastBlockedAt,
		TopOffenders:         t.stats.TopOffenders,
	}
	
	return statsCopy
}

func (t *ThreadSafeDebugDetector) CleanupOldSessions(maxAge time.Duration) {
	t.mu.Lock()
	defer t.mu.Unlock()
	
	cutoff := time.Now().Add(-maxAge)
	for sessionID, detections := range t.sessions {
		if len(detections) > 0 {
			lastDetection := detections[len(detections)-1]
			if lastDetection.DetectionTime.Before(cutoff) {
				delete(t.sessions, sessionID)
			}
		}
	}
	
	for ip, expiry := range t.blockedIPs {
		if time.Now().After(expiry) {
			delete(t.blockedIPs, ip)
		}
	}
}

func (t *ThreadSafeDebugDetector) SetConfig(config *DebugDetectorConfig) {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.config = config
}

func (t *ThreadSafeDebugDetector) GetConfig() *DebugDetectorConfig {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.config
}
