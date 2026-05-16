package ha

import (
	"context"
	"fmt"
	"math"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type HealthStatus string

const (
	StatusHealthy   HealthStatus = "healthy"
	StatusUnhealthy HealthStatus = "unhealthy"
	StatusDegraded HealthStatus = "degraded"
	StatusUnknown  HealthStatus = "unknown"
)

type NodeHealth struct {
	NodeID       string
	URL          string
	Status       HealthStatus
	LastCheck    time.Time
	Latency      time.Duration
	Failures     int32
	TotalChecks  int32
	Metadata     map[string]interface{}
	mu           sync.RWMutex
}

type HealthCheckResult struct {
	NodeID     string
	Status     HealthStatus
	Latency    time.Duration
	Error      error
	Timestamp  time.Time
	Metadata   map[string]interface{}
}

type HealthChecker struct {
	nodes          map[string]*NodeHealth
	checkInterval  time.Duration
	timeout        time.Duration
	maxFailures    int32
	httpClient     *http.Client
	healthEndpoint string
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	running        atomic.Bool
	checkFuncs     []HealthCheckFunc
	onStatusChange OnStatusChangeFunc
}

type HealthCheckFunc func(ctx context.Context, url string) (*HealthCheckResult, error)

type OnStatusChangeFunc func(nodeID string, oldStatus, newStatus HealthStatus)

func NewHealthChecker(checkInterval, timeout time.Duration) *HealthChecker {
	ctx, cancel := context.WithCancel(context.Background())
	
	return &HealthChecker{
		nodes:          make(map[string]*NodeHealth),
		checkInterval:  checkInterval,
		timeout:        timeout,
		maxFailures:    3,
		healthEndpoint: "/health",
		httpClient: &http.Client{
			Timeout: timeout,
			Transport: &http.Transport{
				MaxIdleConns:        100,
				MaxIdleConnsPerHost: 10,
				IdleConnTimeout:     90 * time.Second,
			},
		},
		ctx:        ctx,
		cancel:     cancel,
		checkFuncs: []HealthCheckFunc{defaultHTTPHealthCheck},
	}
}

func defaultHTTPHealthCheck(ctx context.Context, url string) (*HealthCheckResult, error) {
	result := &HealthCheckResult{
		Timestamp: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	fullURL := url + "/health"
	if len(url) > 0 && url[len(url)-1] == '/' {
		fullURL = url[:len(url)-1] + "/health"
	}

	start := time.Now()
	req, err := http.NewRequestWithContext(ctx, http.MethodGet, fullURL, nil)
	if err != nil {
		result.Status = StatusUnknown
		result.Error = err
		return result, err
	}

	req.Header.Set("User-Agent", "HA-HealthChecker/1.0")
	req.Header.Set("Accept", "application/json")

	resp, err := http.DefaultClient.Do(req)
	latency := time.Since(start)
	result.Latency = latency

	if err != nil {
		result.Status = StatusUnhealthy
		result.Error = err
		return result, err
	}
	defer resp.Body.Close()

	result.Metadata["status_code"] = resp.StatusCode

	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		result.Status = StatusHealthy
	} else if resp.StatusCode >= 500 {
		result.Status = StatusUnhealthy
		result.Error = fmt.Errorf("server error: status %d", resp.StatusCode)
	} else {
		result.Status = StatusDegraded
		result.Error = fmt.Errorf("client error: status %d", resp.StatusCode)
	}

	return result, nil
}

func (hc *HealthChecker) AddNode(nodeID, url string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.nodes[nodeID] = &NodeHealth{
		NodeID:    nodeID,
		URL:       url,
		Status:    StatusUnknown,
		LastCheck: time.Time{},
		Metadata:  make(map[string]interface{}),
	}
}

func (hc *HealthChecker) RemoveNode(nodeID string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.nodes, nodeID)
}

func (hc *HealthChecker) UpdateNodeURL(nodeID, url string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	if node, ok := hc.nodes[nodeID]; ok {
		node.URL = url
	}
}

func (hc *HealthChecker) SetHealthEndpoint(endpoint string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.healthEndpoint = endpoint
}

func (hc *HealthChecker) SetMaxFailures(max int32) {
	atomic.StoreInt32(&hc.maxFailures, max)
}

func (hc *HealthChecker) AddCheckFunc(f HealthCheckFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.checkFuncs = append(hc.checkFuncs, f)
}

func (hc *HealthChecker) SetStatusChangeHandler(f OnStatusChangeFunc) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.onStatusChange = f
}

func (hc *HealthChecker) Start(ctx context.Context) {
	if hc.running.Load() {
		return
	}

	hc.ctx, hc.cancel = context.WithCancel(ctx)
	hc.running.Store(true)

	hc.wg.Add(1)
	go hc.checkLoop()
}

func (hc *HealthChecker) Stop() {
	if !hc.running.Load() {
		return
	}

	hc.cancel()
	hc.wg.Wait()
	hc.running.Store(false)
}

func (hc *HealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.checkInterval)
	defer ticker.Stop()

	hc.checkAllNodes()

	for {
		select {
		case <-hc.ctx.Done():
			return
		case <-ticker.C:
			hc.checkAllNodes()
		}
	}
}

func (hc *HealthChecker) checkAllNodes() {
	hc.mu.RLock()
	nodes := make([]*NodeHealth, 0, len(hc.nodes))
	for _, node := range hc.nodes {
		nodes = append(nodes, node)
	}
	hc.mu.RUnlock()

	var wg sync.WaitGroup
	for _, node := range nodes {
		wg.Add(1)
		go func(n *NodeHealth) {
			defer wg.Done()
			hc.checkNode(n)
		}(node)
	}
	wg.Wait()
}

func (hc *HealthChecker) checkNode(node *NodeHealth) {
	ctx, cancel := context.WithTimeout(hc.ctx, hc.timeout)
	defer cancel()

	var lastErr error
	var lastStatus HealthStatus
	_ = lastErr

	for _, checkFunc := range hc.checkFuncs {
		result, err := checkFunc(ctx, node.URL)
		if err == nil && result != nil {
			lastErr = result.Error
			lastStatus = result.Status
			
			node.mu.Lock()
			node.LastCheck = time.Now()
			node.Latency = result.Latency
			if result.Metadata != nil {
				for k, v := range result.Metadata {
					node.Metadata[k] = v
				}
			}
			node.mu.Unlock()

			hc.updateNodeStatus(node, result.Status)
			return
		}
		lastErr = err
		lastStatus = StatusUnhealthy
	}

	node.mu.Lock()
	node.LastCheck = time.Now()
	atomic.AddInt32(&node.Failures, 1)
	atomic.AddInt32(&node.TotalChecks, 1)
	node.mu.Unlock()

	hc.updateNodeStatus(node, lastStatus)
}

func (hc *HealthChecker) updateNodeStatus(node *NodeHealth, newStatus HealthStatus) {
	node.mu.Lock()
	oldStatus := node.Status
	
	if newStatus != oldStatus {
		node.Status = newStatus
		
		if newStatus == StatusHealthy {
			atomic.StoreInt32(&node.Failures, 0)
		}

		hc.mu.RLock()
		onChange := hc.onStatusChange
		hc.mu.RUnlock()

		if onChange != nil && oldStatus != StatusUnknown {
			go onChange(node.NodeID, oldStatus, newStatus)
		}
	}
	node.mu.Unlock()
}

func (hc *HealthChecker) GetNodeStatus(nodeID string) (HealthStatus, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	node, ok := hc.nodes[nodeID]
	if !ok {
		return StatusUnknown, fmt.Errorf("node not found: %s", nodeID)
	}

	node.mu.RLock()
	status := node.Status
	node.mu.RUnlock()

	return status, nil
}

func (hc *HealthChecker) GetAllNodeStatuses() map[string]HealthStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	statuses := make(map[string]HealthStatus)
	for nodeID, node := range hc.nodes {
		node.mu.RLock()
		statuses[nodeID] = node.Status
		node.mu.RUnlock()
	}
	return statuses
}

func (hc *HealthChecker) GetHealthyNodes() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy []string
	for nodeID, node := range hc.nodes {
		node.mu.RLock()
		if node.Status == StatusHealthy {
			healthy = append(healthy, nodeID)
		}
		node.mu.RUnlock()
	}
	return healthy
}

func (hc *HealthChecker) GetNodeStats(nodeID string) (*NodeHealth, error) {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	node, ok := hc.nodes[nodeID]
	if !ok {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	return &NodeHealth{
		NodeID:      node.NodeID,
		URL:         node.URL,
		Status:      node.Status,
		LastCheck:   node.LastCheck,
		Latency:     node.Latency,
		Failures:    atomic.LoadInt32(&node.Failures),
		TotalChecks: atomic.LoadInt32(&node.TotalChecks),
		Metadata:    node.Metadata,
	}, nil
}

func (hc *HealthChecker) IsHealthy(nodeID string) bool {
	status, err := hc.GetNodeStatus(nodeID)
	return err == nil && status == StatusHealthy
}

func (hc *HealthChecker) IsRunning() bool {
	return hc.running.Load()
}

type ClusterHealth struct {
	TotalNodes    int
	HealthyNodes  int
	UnhealthyNodes int
	DegradedNodes int
	ClusterStatus HealthStatus
	NodeStatuses  map[string]HealthStatus
	AvgLatency    time.Duration
	LastUpdate    time.Time
}

func (hc *HealthChecker) GetClusterHealth() *ClusterHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	health := &ClusterHealth{
		NodeStatuses: make(map[string]HealthStatus),
		LastUpdate:   time.Now(),
	}

	if len(hc.nodes) == 0 {
		health.ClusterStatus = StatusUnknown
		return health
	}

	health.TotalNodes = len(hc.nodes)

	var totalLatency int64
	var latencyCount int

	for nodeID, node := range hc.nodes {
		node.mu.RLock()
		status := node.Status
		latency := node.Latency
		node.mu.RUnlock()

		health.NodeStatuses[nodeID] = status

		switch status {
		case StatusHealthy:
			health.HealthyNodes++
			if latency > 0 {
				totalLatency += latency.Nanoseconds()
				latencyCount++
			}
		case StatusUnhealthy:
			health.UnhealthyNodes++
		case StatusDegraded:
			health.DegradedNodes++
		}
	}

	if latencyCount > 0 {
		health.AvgLatency = time.Duration(totalLatency / int64(latencyCount))
	}

	if health.HealthyNodes == health.TotalNodes {
		health.ClusterStatus = StatusHealthy
	} else if health.HealthyNodes > 0 {
		if health.UnhealthyNodes > 0 {
			health.ClusterStatus = StatusDegraded
		} else {
			health.ClusterStatus = StatusDegraded
		}
	} else {
		health.ClusterStatus = StatusUnhealthy
	}

	return health
}

func (hc *HealthChecker) CheckNodeOnce(nodeID string) error {
	hc.mu.RLock()
	node, ok := hc.nodes[nodeID]
	hc.mu.RUnlock()

	if !ok {
		return fmt.Errorf("node not found: %s", nodeID)
	}

	hc.checkNode(node)
	return nil
}

type HealthCheckConfig struct {
	Interval      time.Duration
	Timeout       time.Duration
	MaxRetries    int32
	SuccessThreshold int32
	FailureThreshold int32
	Endpoint      string
}

func DefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Interval:         10 * time.Second,
		Timeout:          5 * time.Second,
		MaxRetries:       3,
		SuccessThreshold: 2,
		FailureThreshold: 3,
		Endpoint:         "/health",
	}
}

type RollingWindowHealthChecker struct {
	*HealthChecker
	windowSize    time.Duration
	checkInterval time.Duration
	successCounts map[string]*RollingCounter
	failureCounts map[string]*RollingCounter
	mu            sync.RWMutex
}

type RollingCounter struct {
	mu       sync.RWMutex
	events   []time.Time
	window   time.Duration
}

func NewRollingCounter(window time.Duration) *RollingCounter {
	return &RollingCounter{
		events: make([]time.Time, 0),
		window: window,
	}
}

func (rc *RollingCounter) Add() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	
	now := time.Now()
	rc.events = append(rc.events, now)
	rc.cleanup(now)
}

func (rc *RollingCounter) Count() int {
	rc.mu.RLock()
	defer rc.mu.RUnlock()
	rc.cleanup(time.Now())
	return len(rc.events)
}

func (rc *RollingCounter) cleanup(now time.Time) {
	cutoff := now.Add(-rc.window)
	idx := 0
	for i, t := range rc.events {
		if t.After(cutoff) {
			idx = i
			break
		}
		idx = i + 1
	}
	if idx > 0 && idx <= len(rc.events) {
		rc.events = rc.events[idx:]
	}
}

func NewRollingWindowHealthChecker(windowSize, checkInterval, timeout time.Duration) *RollingWindowHealthChecker {
	rwhc := &RollingWindowHealthChecker{
		HealthChecker: NewHealthChecker(checkInterval, timeout),
		windowSize:    windowSize,
		checkInterval: checkInterval,
		successCounts: make(map[string]*RollingCounter),
		failureCounts: make(map[string]*RollingCounter),
	}

	originalHandler := rwhc.HealthChecker.onStatusChange
	rwhc.SetStatusChangeHandler(func(nodeID string, oldStatus, newStatus HealthStatus) {
		if newStatus == StatusHealthy {
			rwhc.recordSuccess(nodeID)
		} else if newStatus == StatusUnhealthy {
			rwhc.recordFailure(nodeID)
		}
		if originalHandler != nil {
			originalHandler(nodeID, oldStatus, newStatus)
		}
	})

	return rwhc
}

func (rwhc *RollingWindowHealthChecker) recordSuccess(nodeID string) {
	rwhc.mu.Lock()
	defer rwhc.mu.Unlock()

	if _, ok := rwhc.successCounts[nodeID]; !ok {
		rwhc.successCounts[nodeID] = NewRollingCounter(rwhc.windowSize)
	}
	rwhc.successCounts[nodeID].Add()
}

func (rwhc *RollingWindowHealthChecker) recordFailure(nodeID string) {
	rwhc.mu.Lock()
	defer rwhc.mu.Unlock()

	if _, ok := rwhc.failureCounts[nodeID]; !ok {
		rwhc.failureCounts[nodeID] = NewRollingCounter(rwhc.windowSize)
	}
	rwhc.failureCounts[nodeID].Add()
}

func (rwhc *RollingWindowHealthChecker) GetSuccessRate(nodeID string) float64 {
	rwhc.mu.RLock()
	successCounter := rwhc.successCounts[nodeID]
	failureCounter := rwhc.failureCounts[nodeID]
	rwhc.mu.RUnlock()

	if successCounter == nil {
		return 0
	}

	successes := float64(successCounter.Count())
	if failureCounter != nil {
		failures := float64(failureCounter.Count())
		total := successes + failures
		if total == 0 {
			return 0
		}
		return successes / total
	}

	return successes
}

func (rwhc *RollingWindowHealthChecker) GetHealthScore(nodeID string) float64 {
	successRate := rwhc.GetSuccessRate(nodeID)
	
	latencyScore := 1.0
	latency, err := rwhc.HealthChecker.GetNodeStats(nodeID)
	if err == nil && latency.Latency > 0 {
		if latency.Latency < 100*time.Millisecond {
			latencyScore = 1.0
		} else if latency.Latency < 500*time.Millisecond {
			latencyScore = 0.8
		} else if latency.Latency < 1*time.Second {
			latencyScore = 0.5
		} else {
			latencyScore = 0.2
		}
	}

	return successRate*0.7 + latencyScore*0.3
}

func roundToDecimalPlaces(f float64, places int) float64 {
	ratio := math.Pow(10, float64(places))
	return math.Round(f*ratio) / ratio
}
