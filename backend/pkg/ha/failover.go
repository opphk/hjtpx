package ha

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type FailoverState string

const (
	FailoverStateNormal     FailoverState = "normal"
	FailoverStateFailing   FailoverState = "failing"
	FailoverStateFailedOver FailoverState = "failed_over"
	FailoverStateRecovering FailoverState = "recovering"
)

type FailoverEvent struct {
	Type       FailoverEventType
	NodeID     string
	Timestamp  time.Time
	FromState  FailoverState
	ToState    FailoverState
	Message    string
	Metadata   map[string]interface{}
}

type FailoverEventType string

const (
	FailoverEventNodeDown       FailoverEventType = "node_down"
	FailoverEventNodeUp         FailoverEventType = "node_up"
	FailoverEventFailoverStart  FailoverEventType = "failover_start"
	FailoverEventFailoverEnd    FailoverEventType = "failover_end"
	FailoverEventRecoveryStart  FailoverEventType = "recovery_start"
	FailoverEventRecoveryEnd    FailoverEventType = "recovery_end"
	FailoverEventHealthCheckFail FailoverEventType = "health_check_fail"
)

type FailoverStrategy string

const (
	FailoverStrategyAutomatic FailoverStrategy = "automatic"
	FailoverStrategyManual    FailoverStrategy = "manual"
	FailoverStrategyScheduled FailoverStrategy = "scheduled"
)

type FailoverConfig struct {
	FailureThreshold     int
	RecoveryThreshold    int
	FailoverTimeout      time.Duration
	RecoveryTimeout      time.Duration
	MaxFailoverAttempts  int
	FailoverStrategy     FailoverStrategy
	HealthCheckInterval  time.Duration
	RecoveryCheckInterval time.Duration
	EnableAutoRecovery   bool
	EnableNotification   bool
}

func DefaultFailoverConfig() *FailoverConfig {
	return &FailoverConfig{
		FailureThreshold:     3,
		RecoveryThreshold:    2,
		FailoverTimeout:       30 * time.Second,
		RecoveryTimeout:       60 * time.Second,
		MaxFailoverAttempts:   3,
		FailoverStrategy:      FailoverStrategyAutomatic,
		HealthCheckInterval:   5 * time.Second,
		RecoveryCheckInterval: 10 * time.Second,
		EnableAutoRecovery:    true,
		EnableNotification:    true,
	}
}

type FailoverController struct {
	config         *FailoverConfig
	nodeStates     map[string]*NodeFailoverState
	healthChecker  *HealthChecker
	primaryNode    atomic.Value
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	eventHandlers  []FailoverEventHandler
	eventLog       []*FailoverEvent
	maxEventLog    int
	activeFailover atomic.Int32
	metrics        *FailoverMetrics
}

type NodeFailoverState struct {
	NodeID          string
	CurrentState    FailoverState
	FailureCount    int32
	RecoveryCount   int32
	LastFailure     time.Time
	LastRecovery    time.Time
	FailoverAttempts int32
	PrimarySince    time.Time
	Metadata        map[string]interface{}
	mu              sync.RWMutex
}

type FailoverEventHandler func(event *FailoverEvent)

type FailoverMetrics struct {
	TotalFailovers    atomic.Int64
	SuccessfulFailover atomic.Int64
	FailedFailovers    atomic.Int64
	TotalRecoveries   atomic.Int64
	AvgFailoverTime   atomic.Int64
	LastFailoverTime  atomic.Int64
	mu                sync.RWMutex
	failoverTimes     []time.Duration
}

func NewFailoverMetrics() *FailoverMetrics {
	return &FailoverMetrics{
		failoverTimes: make([]time.Duration, 0, 100),
	}
}

func (fm *FailoverMetrics) RecordFailover(duration time.Duration, success bool) {
	fm.TotalFailovers.Add(1)
	if success {
		fm.SuccessfulFailover.Add(1)
	} else {
		fm.FailedFailovers.Add(1)
	}

	fm.mu.Lock()
	fm.failoverTimes = append(fm.failoverTimes, duration)
	if len(fm.failoverTimes) > 100 {
		fm.failoverTimes = fm.failoverTimes[1:]
	}

	var total int64
	for _, d := range fm.failoverTimes {
		total += d.Nanoseconds()
	}
	fm.AvgFailoverTime.Store(total / int64(len(fm.failoverTimes)))
	fm.LastFailoverTime.Store(time.Now().UnixNano())
	fm.mu.Unlock()
}

func (fm *FailoverMetrics) RecordRecovery() {
	fm.TotalRecoveries.Add(1)
}

func NewFailoverController(config *FailoverConfig, healthChecker *HealthChecker) *FailoverController {
	if config == nil {
		config = DefaultFailoverConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())
	
	controller := &FailoverController{
		config:        config,
		nodeStates:    make(map[string]*NodeFailoverState),
		healthChecker: healthChecker,
		mu:            sync.RWMutex{},
		ctx:           ctx,
		cancel:        cancel,
		eventLog:      make([]*FailoverEvent, 0),
		maxEventLog:   1000,
		metrics:       NewFailoverMetrics(),
	}

	healthChecker.SetStatusChangeHandler(controller.handleStatusChange)

	return controller
}

func (fc *FailoverController) SetPrimaryNode(nodeID string) {
	fc.primaryNode.Store(nodeID)
	
	fc.mu.Lock()
	defer fc.mu.Unlock()

	if state, ok := fc.nodeStates[nodeID]; ok {
		state.mu.Lock()
		state.PrimarySince = time.Now()
		state.mu.Unlock()
	}
}

func (fc *FailoverController) GetPrimaryNode() string {
	node := fc.primaryNode.Load()
	if node == nil {
		return ""
	}
	return node.(string)
}

func (fc *FailoverController) AddNode(nodeID string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()

	fc.nodeStates[nodeID] = &NodeFailoverState{
		NodeID:       nodeID,
		CurrentState: FailoverStateNormal,
		Metadata:     make(map[string]interface{}),
	}
}

func (fc *FailoverController) RemoveNode(nodeID string) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	delete(fc.nodeStates, nodeID)
}

func (fc *FailoverController) handleStatusChange(nodeID string, oldStatus, newStatus HealthStatus) {
	fc.mu.RLock()
	state, ok := fc.nodeStates[nodeID]
	fc.mu.RUnlock()

	if !ok {
		return
	}

	fc.logEvent(&FailoverEvent{
		Type:       FailoverEventHealthCheckFail,
		NodeID:     nodeID,
		Timestamp:  time.Now(),
		Message:    fmt.Sprintf("Health status changed from %s to %s", oldStatus, newStatus),
		Metadata:   map[string]interface{}{"old_status": oldStatus, "new_status": newStatus},
	})

	if newStatus == StatusUnhealthy || newStatus == StatusDegraded {
		fc.handleNodeUnhealthy(nodeID)
	} else if newStatus == StatusHealthy {
		fc.handleNodeHealthy(nodeID)
	}
}

func (fc *FailoverController) handleNodeUnhealthy(nodeID string) {
	fc.mu.Lock()
	state := fc.nodeStates[nodeID]
	fc.mu.Unlock()

	state.mu.Lock()
	state.FailureCount++
	state.LastFailure = time.Now()
	failureCount := state.FailureCount
	state.mu.Unlock()

	if failureCount >= int32(fc.config.FailureThreshold) {
		if state.CurrentState == FailoverStateNormal {
			fc.initiateFailover(nodeID)
		} else if state.CurrentState == FailoverStateRecovering {
			state.mu.Lock()
			state.CurrentState = FailoverStateFailing
			state.RecoveryCount = 0
			state.mu.Unlock()
		}
	}
}

func (fc *FailoverController) handleNodeHealthy(nodeID string) {
	fc.mu.Lock()
	state := fc.nodeStates[nodeID]
	fc.mu.Unlock()

	state.mu.Lock()
	state.RecoveryCount++
	state.LastRecovery = time.Now()
	recoveryCount := state.RecoveryCount
	state.mu.Unlock()

	if recoveryCount >= int32(fc.config.RecoveryThreshold) && state.CurrentState == FailoverStateFailedOver {
		if fc.config.EnableAutoRecovery {
			fc.initiateRecovery(nodeID)
		}
	}
}

func (fc *FailoverController) initiateFailover(failedNodeID string) {
	if !fc.activeFailover.CompareAndSwap(0, 1) {
		return
	}

	fc.logEvent(&FailoverEvent{
		Type:      FailoverEventFailoverStart,
		NodeID:    failedNodeID,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Initiating failover for node %s", failedNodeID),
	})

	startTime := time.Now()

	fc.mu.Lock()
	state := fc.nodeStates[failedNodeID]
	state.CurrentState = FailoverStateFailing
	fc.mu.Unlock()

	fc.mu.RLock()
	var healthyNodes []string
	for nodeID, nodeState := range fc.nodeStates {
		if nodeID != failedNodeID && nodeState.CurrentState == FailoverStateNormal {
			if fc.healthChecker.IsHealthy(nodeID) {
				healthyNodes = append(healthyNodes, nodeID)
			}
		}
	}
	fc.mu.RUnlock()

	if len(healthyNodes) == 0 {
		fc.logEvent(&FailoverEvent{
			Type:      FailoverEventFailoverEnd,
			NodeID:    failedNodeID,
			Timestamp: time.Now(),
			Message:   "No healthy nodes available for failover",
			Metadata:  map[string]interface{}{"success": false},
		})
		fc.metrics.RecordFailover(time.Since(startTime), false)
		fc.activeFailover.Store(0)
		return
	}

	newPrimary := fc.selectNewPrimary(healthyNodes)

	fc.mu.Lock()
	state.CurrentState = FailoverStateFailedOver
	state.FailoverAttempts++
	fc.mu.Unlock()

	fc.primaryNode.Store(newPrimary)

	fc.logEvent(&FailoverEvent{
		Type:       FailoverEventFailoverEnd,
		NodeID:     failedNodeID,
		Timestamp:  time.Now(),
		FromState:  FailoverStateFailing,
		ToState:    FailoverStateFailedOver,
		Message:    fmt.Sprintf("Failover completed: %s is now primary", newPrimary),
		Metadata:   map[string]interface{}{"new_primary": newPrimary, "success": true},
	})

	fc.metrics.RecordFailover(time.Since(startTime), true)
	fc.activeFailover.Store(0)
}

func (fc *FailoverController) selectNewPrimary(candidates []string) string {
	if len(candidates) == 0 {
		return ""
	}

	var bestNode string
	var bestScore float64

	for _, nodeID := range candidates {
		stats, err := fc.healthChecker.GetNodeStats(nodeID)
		if err != nil {
			continue
		}

		score := 1.0 / (stats.Latency.Seconds() + 0.001)

		fc.mu.RLock()
		if state, ok := fc.nodeStates[nodeID]; ok {
			state.mu.RLock()
			if state.CurrentState == FailoverStateRecovering {
				score *= 0.8
			}
			state.mu.RUnlock()
		}
		fc.mu.RUnlock()

		if score > bestScore {
			bestScore = score
			bestNode = nodeID
		}
	}

	return bestNode
}

func (fc *FailoverController) initiateRecovery(recoveredNodeID string) {
	fc.logEvent(&FailoverEvent{
		Type:      FailoverEventRecoveryStart,
		NodeID:    recoveredNodeID,
		Timestamp: time.Now(),
		Message:   fmt.Sprintf("Initiating recovery for node %s", recoveredNodeID),
	})

	fc.mu.Lock()
	state := fc.nodeStates[recoveredNodeID]
	state.CurrentState = FailoverStateRecovering
	fc.mu.Unlock()

	currentPrimary := fc.GetPrimaryNode()
	
	if fc.healthChecker.IsHealthy(recoveredNodeID) {
		fc.mu.Lock()
		state.CurrentState = FailoverStateNormal
		state.FailureCount = 0
		state.RecoveryCount = 0
		fc.mu.Unlock()

		fc.logEvent(&FailoverEvent{
			Type:       FailoverEventRecoveryEnd,
			NodeID:     recoveredNodeID,
			Timestamp:  time.Now(),
			FromState:  FailoverStateRecovering,
			ToState:    FailoverStateNormal,
			Message:    fmt.Sprintf("Node %s recovered successfully", recoveredNodeID),
			Metadata:   map[string]interface{}{"is_primary": recoveredNodeID == currentPrimary},
		})

		fc.metrics.RecordRecovery()
	}
}

func (fc *FailoverController) ManualFailover(fromNodeID, toNodeID string) error {
	fc.mu.RLock()
	fromState, fromExists := fc.nodeStates[fromNodeID]
	toState, toExists := fc.nodeStates[toNodeID]
	fc.mu.RUnlock()

	if !fromExists || !toExists {
		return fmt.Errorf("node not found")
	}

	if !fc.healthChecker.IsHealthy(toNodeID) {
		return fmt.Errorf("target node %s is not healthy", toNodeID)
	}

	fc.logEvent(&FailoverEvent{
		Type:       FailoverEventFailoverEnd,
		NodeID:     fromNodeID,
		Timestamp:  time.Now(),
		Message:    fmt.Sprintf("Manual failover from %s to %s", fromNodeID, toNodeID),
		Metadata:   map[string]interface{}{"from": fromNodeID, "to": toNodeID, "type": "manual"},
	})

	fc.mu.Lock()
	fromState.CurrentState = FailoverStateFailedOver
	toState.CurrentState = FailoverStateNormal
	fc.mu.Unlock()

	fc.primaryNode.Store(toNodeID)

	return nil
}

func (fc *FailoverController) GetNodeState(nodeID string) (FailoverState, error) {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	state, ok := fc.nodeStates[nodeID]
	if !ok {
		return "", fmt.Errorf("node not found: %s", nodeID)
	}

	state.mu.RLock()
	defer state.mu.RUnlock()
	return state.CurrentState, nil
}

func (fc *FailoverController) GetAllNodeStates() map[string]FailoverState {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	states := make(map[string]FailoverState)
	for nodeID, state := range fc.nodeStates {
		state.mu.RLock()
		states[nodeID] = state.CurrentState
		state.mu.RUnlock()
	}
	return states
}

func (fc *FailoverController) IsFailoverActive() bool {
	return fc.activeFailover.Load() == 1
}

func (fc *FailoverController) GetMetrics() map[string]interface{} {
	m := fc.metrics
	return map[string]interface{}{
		"total_failovers":     m.TotalFailovers.Load(),
		"successful_failovers": m.SuccessfulFailover.Load(),
		"failed_failovers":    m.FailedFailovers.Load(),
		"total_recoveries":    m.TotalRecoveries.Load(),
		"avg_failover_time_ms": m.AvgFailoverTime.Load() / 1e6,
	}
}

func (fc *FailoverController) logEvent(event *FailoverEvent) {
	fc.mu.Lock()
	fc.eventLog = append(fc.eventLog, event)
	if len(fc.eventLog) > fc.maxEventLog {
		fc.eventLog = fc.eventLog[1:]
	}
	fc.mu.Unlock()

	for _, handler := range fc.eventHandlers {
		go handler(event)
	}
}

func (fc *FailoverController) AddEventHandler(handler FailoverEventHandler) {
	fc.mu.Lock()
	defer fc.mu.Unlock()
	fc.eventHandlers = append(fc.eventHandlers, handler)
}

func (fc *FailoverController) GetEventLog(limit int) []*FailoverEvent {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	if limit <= 0 || limit > len(fc.eventLog) {
		limit = len(fc.eventLog)
	}

	events := make([]*FailoverEvent, limit)
	copy(events, fc.eventLog[len(fc.eventLog)-limit:])
	return events
}

func (fc *FailoverController) GetClusterStatus() *ClusterFailoverStatus {
	fc.mu.RLock()
	defer fc.mu.RUnlock()

	status := &ClusterFailoverStatus{
		PrimaryNode:       fc.GetPrimaryNode(),
		FailoverActive:    fc.IsFailoverActive(),
		NodeStates:        make(map[string]FailoverState),
		HealthyNodes:      fc.healthChecker.GetHealthyNodes(),
		ClusterHealth:     fc.healthChecker.GetClusterHealth(),
		FailoverMetrics:   fc.GetMetrics(),
	}

	for nodeID, state := range fc.nodeStates {
		state.mu.RLock()
		status.NodeStates[nodeID] = state.CurrentState
		state.mu.RUnlock()
	}

	return status
}

type ClusterFailoverStatus struct {
	PrimaryNode     string
	FailoverActive  bool
	NodeStates      map[string]FailoverState
	HealthyNodes    []string
	ClusterHealth   *ClusterHealth
	FailoverMetrics map[string]interface{}
}

type ScheduledFailover struct {
	FailoverController
	schedule  *FailoverSchedule
	mu        sync.RWMutex
	timers    map[string]*time.Timer
	stopChan  chan struct{}
}

type FailoverSchedule struct {
	MaintenanceWindows []MaintenanceWindow
	PreferredPrimary  string
	FailoverPriority   []string
}

type MaintenanceWindow struct {
	StartTime time.Time
	EndTime   time.Time
	Reason    string
}

func NewScheduledFailover(config *FailoverConfig, healthChecker *HealthChecker, schedule *FailoverSchedule) *ScheduledFailover {
	sf := &ScheduledFailover{
		FailoverController: *NewFailoverController(config, healthChecker),
		schedule:           schedule,
		timers:             make(map[string]*time.Timer),
		stopChan:          make(chan struct{}),
	}

	return sf
}

func (sf *ScheduledFailover) ScheduleFailover(nodeID string, failoverTime time.Time) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if timer, exists := sf.timers[nodeID]; exists {
		timer.Stop()
	}

	delay := time.Until(failoverTime)
	if delay <= 0 {
		return
	}

	sf.timers[nodeID] = time.AfterFunc(delay, func() {
		sf.ManualFailover(sf.GetPrimaryNode(), nodeID)
		sf.mu.Lock()
		delete(sf.timers, nodeID)
		sf.mu.Unlock()
	})
}

func (sf *ScheduledFailover) CancelScheduledFailover(nodeID string) {
	sf.mu.Lock()
	defer sf.mu.Unlock()

	if timer, exists := sf.timers[nodeID]; exists {
		timer.Stop()
		delete(sf.timers, nodeID)
	}
}

func (sf *ScheduledFailover) GetNextScheduledFailover() (nodeID string, time time.Time, exists bool) {
	sf.mu.RLock()
	defer sf.mu.RUnlock()

	if len(sf.timers) == 0 {
		return "", time.Time{}, false
	}

	var earliest time.Time
	for id, timer := range sf.timers {
		t := timer.Reset(time.Hour * 24 * 365)
		if !exists || t.Before(earliest) {
			nodeID = id
			earliest = t
			exists = true
		}
	}

	return nodeID, earliest, exists
}
