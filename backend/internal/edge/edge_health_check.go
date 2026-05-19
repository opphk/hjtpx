package edge

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/http"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/redis/go-redis/v9"
)

type HealthStatus string

const (
	HealthStatusHealthy   HealthStatus = "healthy"
	HealthStatusDegraded HealthStatus = "degraded"
	HealthStatusUnhealthy HealthStatus = "unhealthy"
	HealthStatusUnknown   HealthStatus = "unknown"
)

type CheckType string

const (
	CheckTypeTCP       CheckType = "tcp"
	CheckTypeHTTP      CheckType = "http"
	CheckTypeHTTPS     CheckType = "https"
	CheckTypeICMP      CheckType = "icmp"
	CheckTypeDNS       CheckType = "dns"
	CheckTypePort      CheckType = "port"
)

type HealthCheck struct {
	ID            string            `json:"id"`
	NodeID        string            `json:"node_id"`
	Target        string            `json:"target"`
	Port          int               `json:"port"`
	Type          CheckType         `json:"type"`
	Interval      time.Duration     `json:"interval"`
	Timeout       time.Duration     `json:"timeout"`
	Threshold     int               `json:"threshold"`
	FailureThreshold int             `json:"failure_threshold"`
	SuccessThreshold int             `json:"success_threshold"`
	Enabled       bool              `json:"enabled"`
	Headers       map[string]string `json:"headers,omitempty"`
	Method        string            `json:"method,omitempty"`
	ExpectedStatus int              `json:"expected_status,omitempty"`
	ExpectedBody  string            `json:"expected_body,omitempty"`
}

type HealthCheckResult struct {
	CheckID     string        `json:"check_id"`
	NodeID      string        `json:"node_id"`
	Status      HealthStatus  `json:"status"`
	Latency     time.Duration `json:"latency"`
	LatencyMs   float64       `json:"latency_ms"`
	HTTPStatus  int           `json:"http_status,omitempty"`
	DNSResolved bool          `json:"dns_resolved,omitempty"`
	Connected   bool          `json:"connected,omitempty"`
	Message     string        `json:"message,omitempty"`
	Timestamp   time.Time     `json:"timestamp"`
	Details     map[string]interface{} `json:"details,omitempty"`
}

type HealthCheckHistory struct {
	CheckID    string              `json:"check_id"`
	NodeID     string              `json:"node_id"`
	Results    []*HealthCheckResult `json:"results"`
	mu         sync.RWMutex
	maxResults int
}

type NodeHealth struct {
	NodeID       string           `json:"node_id"`
	Status       HealthStatus     `json:"status"`
	OverallScore float64          `json:"overall_score"`
	Checks       map[string]*CheckStatus `json:"checks"`
	LastCheck    time.Time        `json:"last_check"`
	NextCheck    time.Time        `json:"next_check"`
	Uptime       float64          `json:"uptime"`
	TotalChecks  int64            `json:"total_checks"`
	FailedChecks int64            `json:"failed_checks"`
}

type CheckStatus struct {
	CheckID       string        `json:"check_id"`
	Type          CheckType     `json:"type"`
	Status        HealthStatus  `json:"status"`
	LatencyAvg    float64       `json:"latency_avg"`
	LatencyMin    float64       `json:"latency_min"`
	LatencyMax    float64       `json:"latency_max"`
	SuccessRate   float64       `json:"success_rate"`
	ConsecutiveFails int       `json:"consecutive_fails"`
	LastSuccess   time.Time     `json:"last_success"`
	LastFailure   time.Time     `json:"last_failure"`
}

type HealthCheckManager struct {
	checks      map[string]*HealthCheck
	nodeHealth  map[string]*NodeHealth
	history     map[string]*HealthCheckHistory
	nodeManager *EdgeNodeManager
	redisClient *redis.Client
	mu          sync.RWMutex
	stopChan    chan struct{}
	wg          sync.WaitGroup
	metrics     *HealthMetrics
}

type HealthMetrics struct {
	TotalChecks      int64              `json:"total_checks"`
	HealthyNodes     int64              `json:"healthy_nodes"`
	DegradedNodes    int64              `json:"degraded_nodes"`
	UnhealthyNodes   int64              `json:"unhealthy_nodes"`
	CheckDurations   map[string]*DurationStats `json:"check_durations"`
	mu               sync.RWMutex
}

type DurationStats struct {
	Count    int64   `json:"count"`
	Sum      float64 `json:"sum"`
	Min      float64 `json:"min"`
	Max      float64 `json:"max"`
	Avg      float64 `json:"avg"`
}

type HealthAlert struct {
	ID        string        `json:"id"`
	NodeID    string        `json:"node_id"`
	CheckID   string        `json:"check_id"`
	Type      string        `json:"type"`
	Severity  string        `json:"severity"`
	Message   string        `json:"message"`
	Status    HealthStatus  `json:"status"`
	Timestamp time.Time     `json:"timestamp"`
}

func NewHealthCheckManager(nodeManager *EdgeNodeManager, redisClient *redis.Client) *HealthCheckManager {
	manager := &HealthCheckManager{
		checks:      make(map[string]*HealthCheck),
		nodeHealth:  make(map[string]*NodeHealth),
		history:     make(map[string]*HealthCheckHistory),
		nodeManager: nodeManager,
		redisClient: redisClient,
		stopChan:    make(chan struct{}),
		metrics: &HealthMetrics{
			CheckDurations: make(map[string]*DurationStats),
		},
	}

	manager.startHealthChecker()

	return manager
}

func (m *HealthCheckManager) AddCheck(check *HealthCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if check.ID == "" {
		check.ID = fmt.Sprintf("check-%d", time.Now().UnixNano())
	}

	if check.Interval == 0 {
		check.Interval = 30 * time.Second
	}
	if check.Timeout == 0 {
		check.Timeout = 5 * time.Second
	}
	if check.Threshold == 0 {
		check.Threshold = 3
	}
	if check.FailureThreshold == 0 {
		check.FailureThreshold = 3
	}
	if check.SuccessThreshold == 0 {
		check.SuccessThreshold = 2
	}

	check.Enabled = true
	m.checks[check.ID] = check

	m.history[check.ID] = &HealthCheckHistory{
		CheckID:    check.ID,
		NodeID:     check.NodeID,
		maxResults: 1000,
		Results:    make([]*HealthCheckResult, 0, 1000),
	}

	m.initializeNodeHealth(check.NodeID)

	m.wg.Add(1)
	go m.runCheckLoop(check)

	return nil
}

func (m *HealthCheckManager) initializeNodeHealth(nodeID string) {
	if _, exists := m.nodeHealth[nodeID]; !exists {
		m.nodeHealth[nodeID] = &NodeHealth{
			NodeID: nodeID,
			Status: HealthStatusUnknown,
			Checks: make(map[string]*CheckStatus),
		}
	}
}

func (m *HealthCheckManager) DeleteCheck(checkID string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	check, exists := m.checks[checkID]
	if !exists {
		return fmt.Errorf("check not found: %s", checkID)
	}

	m.checks[checkID].Enabled = false
	delete(m.checks, checkID)

	_ = check
	return nil
}

func (m *HealthCheckManager) GetCheck(checkID string) (*HealthCheck, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	check, exists := m.checks[checkID]
	if !exists {
		return nil, fmt.Errorf("check not found: %s", checkID)
	}

	checkCopy := *check
	return &checkCopy, nil
}

func (m *HealthCheckManager) ListChecks(nodeID string) []*HealthCheck {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var checks []*HealthCheck
	for _, check := range m.checks {
		if nodeID == "" || check.NodeID == nodeID {
			checkCopy := *check
			checks = append(checks, &checkCopy)
		}
	}
	return checks
}

func (m *HealthCheckManager) UpdateCheck(check *HealthCheck) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	existing, exists := m.checks[check.ID]
	if !exists {
		return fmt.Errorf("check not found: %s", check.ID)
	}

	existing.Target = check.Target
	existing.Port = check.Port
	existing.Type = check.Type
	existing.Interval = check.Interval
	existing.Timeout = check.Timeout
	existing.Threshold = check.Threshold
	existing.FailureThreshold = check.FailureThreshold
	existing.SuccessThreshold = check.SuccessThreshold
	existing.Headers = check.Headers
	existing.Method = check.Method
	existing.ExpectedStatus = check.ExpectedStatus
	existing.ExpectedBody = check.ExpectedBody
	existing.Enabled = check.Enabled

	return nil
}

func (m *HealthCheckManager) runCheckLoop(check *HealthCheck) {
	defer m.wg.Done()

	ticker := time.NewTicker(check.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopChan:
			return
		case <-ticker.C:
			if check.Enabled {
				result := m.performCheck(check)
				m.processResult(result)
			}
		}
	}
}

func (m *HealthCheckManager) performCheck(check *HealthCheck) *HealthCheckResult {
	result := &HealthCheckResult{
		CheckID:   check.ID,
		NodeID:    check.NodeID,
		Timestamp: time.Now(),
	}

	target := check.Target
	if check.Port > 0 {
		target = fmt.Sprintf("%s:%d", check.Target, check.Port)
	}

	switch check.Type {
	case CheckTypeTCP, CheckTypePort:
		result = m.checkTCP(target, check.Timeout, result)
	case CheckTypeHTTP:
		result = m.checkHTTP("http", target, check, result)
	case CheckTypeHTTPS:
		result = m.checkHTTP("https", target, check, result)
	case CheckTypeICMP:
		result = m.checkICMP(target, check.Timeout, result)
	case CheckTypeDNS:
		result = m.checkDNS(target, check.Timeout, result)
	default:
		result.Status = HealthStatusUnknown
		result.Message = "Unknown check type"
	}

	return result
}

func (m *HealthCheckManager) checkTCP(target string, timeout time.Duration, result *HealthCheckResult) *HealthCheckResult {
	start := time.Now()

	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Message = err.Error()
		result.Latency = time.Since(start)
		result.LatencyMs = float64(result.Latency.Milliseconds())
		result.Connected = false
		return result
	}
	defer conn.Close()

	result.Status = HealthStatusHealthy
	result.Latency = time.Since(start)
	result.LatencyMs = float64(result.Latency.Milliseconds())
	result.Connected = true
	result.Message = "Connection successful"

	return result
}

func (m *HealthCheckManager) checkHTTP(scheme, target string, check *HealthCheck, result *HealthCheckResult) *HealthCheckResult {
	start := time.Now()

	if !strings.HasPrefix(target, "http") {
		target = fmt.Sprintf("%s://%s", scheme, target)
	}

	client := &http.Client{
		Timeout: check.Timeout,
	}

	req, err := http.NewRequest(check.Method, target, nil)
	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Message = err.Error()
		result.Latency = time.Since(start)
		result.LatencyMs = float64(result.Latency.Milliseconds())
		return result
	}

	for key, value := range check.Headers {
		req.Header.Set(key, value)
	}

	resp, err := client.Do(req)
	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Message = err.Error()
		result.Latency = time.Since(start)
		result.LatencyMs = float64(result.Latency.Milliseconds())
		return result
	}
	defer resp.Body.Close()

	result.HTTPStatus = resp.StatusCode
	result.Latency = time.Since(start)
	result.LatencyMs = float64(result.Latency.Milliseconds())

	if check.ExpectedStatus > 0 {
		if resp.StatusCode == check.ExpectedStatus {
			result.Status = HealthStatusHealthy
			result.Message = "Status code match"
		} else {
			result.Status = HealthStatusUnhealthy
			result.Message = fmt.Sprintf("Expected status %d, got %d", check.ExpectedStatus, resp.StatusCode)
		}
	} else if resp.StatusCode >= 200 && resp.StatusCode < 400 {
		result.Status = HealthStatusHealthy
		result.Message = "HTTP request successful"
	} else {
		result.Status = HealthStatusUnhealthy
		result.Message = fmt.Sprintf("HTTP status %d", resp.StatusCode)
	}

	if check.ExpectedBody != "" && result.Status == HealthStatusHealthy {
		buf := make([]byte, 1024)
		resp.Body.Read(buf)
		if !strings.Contains(string(buf), check.ExpectedBody) {
			result.Status = HealthStatusDegraded
			result.Message = "Expected body not found"
		}
	}

	return result
}

func (m *HealthCheckManager) checkICMP(target string, timeout time.Duration, result *HealthCheckResult) *HealthCheckResult {
	result.Status = HealthStatusHealthy
	result.Message = "ICMP check simulated"
	result.Latency = timeout / 2
	result.LatencyMs = float64(result.Latency.Milliseconds())
	return result
}

func (m *HealthCheckManager) checkDNS(target string, timeout time.Duration, result *HealthCheckResult) *HealthCheckResult {
	start := time.Now()

	var host string
	var port = "53"

	if strings.Contains(target, ":") {
		parts := strings.Split(target, ":")
		host = parts[0]
		port = parts[1]
	} else {
		host = target
	}

	_, err := net.ResolveTCPAddr("tcp", net.JoinHostPort(host, port))
	if err != nil {
		result.Status = HealthStatusUnhealthy
		result.Message = err.Error()
		result.Latency = time.Since(start)
		result.LatencyMs = float64(result.Latency.Milliseconds())
		result.DNSResolved = false
		return result
	}

	result.Status = HealthStatusHealthy
	result.Latency = time.Since(start)
	result.LatencyMs = float64(result.Latency.Milliseconds())
	result.DNSResolved = true
	result.Message = "DNS resolution successful"

	return result
}

func (m *HealthCheckManager) processResult(result *HealthCheckResult) {
	m.mu.Lock()
	defer m.mu.Unlock()

	history, exists := m.history[result.CheckID]
	if exists {
		history.mu.Lock()
		history.Results = append(history.Results, result)
		if len(history.Results) > history.maxResults {
			history.Results = history.Results[len(history.Results)-history.maxResults:]
		}
		history.mu.Unlock()
	}

	nodeHealth, exists := m.nodeHealth[result.NodeID]
	if !exists {
		nodeHealth = &NodeHealth{
			NodeID: result.NodeID,
			Checks: make(map[string]*CheckStatus),
		}
		m.nodeHealth[result.NodeID] = nodeHealth
	}

	checkStatus, exists := nodeHealth.Checks[result.CheckID]
	if !exists {
		checkStatus = &CheckStatus{
			CheckID: result.CheckID,
		}
		nodeHealth.Checks[result.CheckID] = checkStatus
	}

	checkStatus.LatencyAvg = (checkStatus.LatencyAvg*float64(checkStatus.SuccessRate) + result.LatencyMs) / (float64(checkStatus.SuccessRate) + 1)
	if checkStatus.LatencyMin == 0 || result.LatencyMs < checkStatus.LatencyMin {
		checkStatus.LatencyMin = result.LatencyMs
	}
	if result.LatencyMs > checkStatus.LatencyMax {
		checkStatus.LatencyMax = result.LatencyMs
	}

	atomic.AddInt64(&nodeHealth.TotalChecks, 1)

	if result.Status == HealthStatusHealthy {
		checkStatus.ConsecutiveFails = 0
		checkStatus.LastSuccess = result.Timestamp
		checkStatus.SuccessRate = float64(atomic.LoadInt64(&nodeHealth.TotalChecks)-atomic.LoadInt64(&nodeHealth.FailedChecks)) / float64(atomic.LoadInt64(&nodeHealth.TotalChecks))
		checkStatus.Status = HealthStatusHealthy
	} else {
		checkStatus.ConsecutiveFails++
		checkStatus.LastFailure = result.Timestamp
		atomic.AddInt64(&nodeHealth.FailedChecks, 1)

		if checkStatus.ConsecutiveFails >= 3 {
			checkStatus.Status = HealthStatusUnhealthy
		} else {
			checkStatus.Status = HealthStatusDegraded
		}
	}

	nodeHealth.LastCheck = result.Timestamp
	nodeHealth.calculateOverallStatus()
	nodeHealth.OverallScore = nodeHealth.calculateOverallScore()

	if m.nodeManager != nil {
		var healthStatus NodeStatus
		switch nodeHealth.Status {
		case HealthStatusHealthy:
			healthStatus = NodeStatusActive
		case HealthStatusDegraded:
			healthStatus = NodeStatusActive
		case HealthStatusUnhealthy:
			healthStatus = NodeStatusUnhealthy
		default:
			healthStatus = NodeStatusInactive
		}
		m.nodeManager.UpdateNodeStatus(context.Background(), result.NodeID, healthStatus)
	}

	m.updateMetrics(result)
}

func (h *NodeHealth) calculateOverallStatus() {
	if len(h.Checks) == 0 {
		h.Status = HealthStatusUnknown
		return
	}

	healthyCount := 0
	degradedCount := 0
	unhealthyCount := 0

	for _, check := range h.Checks {
		switch check.Status {
		case HealthStatusHealthy:
			healthyCount++
		case HealthStatusDegraded:
			degradedCount++
		case HealthStatusUnhealthy:
			unhealthyCount++
		}
	}

	total := len(h.Checks)

	if unhealthyCount > 0 {
		if unhealthyCount >= total/2 {
			h.Status = HealthStatusUnhealthy
		} else {
			h.Status = HealthStatusDegraded
		}
	} else if degradedCount > 0 {
		if degradedCount >= total/2 {
			h.Status = HealthStatusDegraded
		} else {
			h.Status = HealthStatusHealthy
		}
	} else {
		h.Status = HealthStatusHealthy
	}
}

func (h *NodeHealth) calculateOverallScore() float64 {
	if len(h.Checks) == 0 {
		return 0
	}

	var totalScore float64
	for _, check := range h.Checks {
		var checkScore float64

		switch check.Status {
		case HealthStatusHealthy:
			checkScore = 100
		case HealthStatusDegraded:
			checkScore = 50
		case HealthStatusUnhealthy:
			checkScore = 0
		default:
			checkScore = 25
		}

		latencyScore := 100.0 - (check.LatencyAvg / 10)
		if latencyScore < 0 {
			latencyScore = 0
		}

		checkScore = checkScore*0.7 + latencyScore*0.3

		totalScore += checkScore
	}

	return totalScore / float64(len(h.Checks))
}

func (m *HealthCheckManager) updateMetrics(result *HealthCheckResult) {
	atomic.AddInt64(&m.metrics.TotalChecks, 1)

	m.metrics.mu.Lock()
	defer m.metrics.mu.Unlock()

	if _, exists := m.metrics.CheckDurations[result.CheckID]; !exists {
		m.metrics.CheckDurations[result.CheckID] = &DurationStats{
			Min: result.LatencyMs,
			Max: result.LatencyMs,
		}
	}

	stats := m.metrics.CheckDurations[result.CheckID]
	stats.Count++
	stats.Sum += result.LatencyMs
	stats.Avg = stats.Sum / float64(stats.Count)
	if result.LatencyMs < stats.Min {
		stats.Min = result.LatencyMs
	}
	if result.LatencyMs > stats.Max {
		stats.Max = result.LatencyMs
	}
}

func (m *HealthCheckManager) GetNodeHealth(nodeID string) (*NodeHealth, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	health, exists := m.nodeHealth[nodeID]
	if !exists {
		return nil, fmt.Errorf("node health not found: %s", nodeID)
	}

	healthCopy := *health
	healthCopy.Checks = make(map[string]*CheckStatus)
	for k, v := range health.Checks {
		statusCopy := *v
		healthCopy.Checks[k] = &statusCopy
	}

	return &healthCopy, nil
}

func (m *HealthCheckManager) ListNodeHealth() []*NodeHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthList []*NodeHealth
	for _, health := range m.nodeHealth {
		healthCopy := *health
		healthCopy.Checks = make(map[string]*CheckStatus)
		for k, v := range health.Checks {
			statusCopy := *v
			healthCopy.Checks[k] = &statusCopy
		}
		healthList = append(healthList, &healthCopy)
	}

	return healthList
}

func (m *HealthCheckManager) GetCheckHistory(checkID string, limit int) ([]*HealthCheckResult, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history, exists := m.history[checkID]
	if !exists {
		return nil, fmt.Errorf("check history not found: %s", checkID)
	}

	history.mu.RLock()
	defer history.mu.RUnlock()

	results := history.Results
	if limit > 0 && limit < len(results) {
		results = results[len(results)-limit:]
	}

	resultsCopy := make([]*HealthCheckResult, len(results))
	copy(resultsCopy, results)

	return resultsCopy, nil
}

func (m *HealthCheckManager) GetMetrics() *HealthMetrics {
	m.mu.RLock()
	defer m.mu.RUnlock()

	metricsCopy := &HealthMetrics{
		TotalChecks:   atomic.LoadInt64(&m.metrics.TotalChecks),
		HealthyNodes:   0,
		DegradedNodes: 0,
		UnhealthyNodes: 0,
		CheckDurations: make(map[string]*DurationStats),
	}

	for _, health := range m.nodeHealth {
		switch health.Status {
		case HealthStatusHealthy:
			atomic.AddInt64(&metricsCopy.HealthyNodes, 1)
		case HealthStatusDegraded:
			atomic.AddInt64(&metricsCopy.DegradedNodes, 1)
		case HealthStatusUnhealthy:
			atomic.AddInt64(&metricsCopy.UnhealthyNodes, 1)
		}
	}

	m.metrics.mu.Lock()
	for k, v := range m.metrics.CheckDurations {
		statsCopy := *v
		metricsCopy.CheckDurations[k] = &statsCopy
	}
	m.metrics.mu.Unlock()

	return metricsCopy
}

func (m *HealthCheckManager) TriggerCheck(checkID string) (*HealthCheckResult, error) {
	m.mu.RLock()
	check, exists := m.checks[checkID]
	m.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("check not found: %s", checkID)
	}

	result := m.performCheck(check)
	m.processResult(result)

	return result, nil
}

func (m *HealthCheckManager) TriggerNodeChecks(nodeID string) []*HealthCheckResult {
	m.mu.RLock()
	var nodeChecks []*HealthCheck
	for _, check := range m.checks {
		if check.NodeID == nodeID && check.Enabled {
			checkCopy := *check
			nodeChecks = append(nodeChecks, &checkCopy)
		}
	}
	m.mu.RUnlock()

	var results []*HealthCheckResult
	for _, check := range nodeChecks {
		result := m.performCheck(check)
		m.processResult(result)
		results = append(results, result)
	}

	return results
}

func (m *HealthCheckManager) startHealthChecker() {
	go func() {
		ticker := time.NewTicker(1 * time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-m.stopChan:
				return
			case <-ticker.C:
				m.recalculateNodeHealth()
			}
		}
	}()
}

func (m *HealthCheckManager) recalculateNodeHealth() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	for _, health := range m.nodeHealth {
		health.calculateOverallStatus()
		health.OverallScore = health.calculateOverallScore()
	}
}

func (m *HealthCheckManager) Stop() {
	close(m.stopChan)
	m.wg.Wait()
}

func (m *HealthCheckManager) SyncToRedis(ctx context.Context) error {
	if m.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	data, err := json.Marshal(m.checks)
	if err != nil {
		return err
	}

	return m.redisClient.Set(ctx, "edge:health:checks", data, 24*time.Hour).Err()
}

func (m *HealthCheckManager) SyncFromRedis(ctx context.Context) error {
	if m.redisClient == nil {
		return fmt.Errorf("redis client not initialized")
	}

	data, err := m.redisClient.Get(ctx, "edge:health:checks").Bytes()
	if err != nil {
		return err
	}

	var checks map[string]*HealthCheck
	if err := json.Unmarshal(data, &checks); err != nil {
		return err
	}

	m.mu.Lock()
	defer m.mu.Unlock()

	for _, check := range checks {
		m.checks[check.ID] = check
		m.wg.Add(1)
		go m.runCheckLoop(check)
	}

	return nil
}
