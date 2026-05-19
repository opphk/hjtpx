package service

import (
	"context"
	"math"
	"sync"
	"sync/atomic"
	"time"
)

type SLAConfig struct {
	TargetUptime    float64 `json:"target_uptime"`
	MaxDowntimeMin   float64 `json:"max_downtime_minutes_per_month"`
	ResponseTimeP99  int64   `json:"response_time_p99_ms"`
	ErrorRateMax     float64 `json:"max_error_rate_percent"`
}

type SLAMetrics struct {
	TotalUptimeSeconds   atomic.Int64     `json:"total_uptime_seconds"`
	TotalDowntimeSeconds atomic.Int64     `json:"total_downtime_seconds"`
	IncidentsCount       atomic.Int64     `json:"incidents_count"`
	LastIncidentTime     atomic.Int64     `json:"last_incident_time"`
	CurrentStreak        atomic.Int64     `json:"current_uptime_streak_hours"`
	LongestStreak        atomic.Int64     `json:"longest_uptime_streak_hours"`
}

type ServiceHealthStatus string

const (
	HealthStatusHealthy   ServiceHealthStatus = "healthy"
	HealthStatusDegraded  ServiceHealthStatus = "degraded"
	HealthStatusCritical ServiceHealthStatus = "critical"
	HealthStatusDown      ServiceHealthStatus = "down"
)

type ServiceComponent struct {
	Name            string              `json:"name"`
	Status          ServiceHealthStatus `json:"status"`
	IsCritical      bool                `json:"is_critical"`
	LastCheckTime   time.Time           `json:"last_check_time"`
	FailureCount    int                 `json:"failure_count"`
	SuccessCount    int                 `json:"success_count"`
	AvgResponseTime int64              `json:"avg_response_time_ms"`
}

type GracefulDegradationConfig struct {
	Enabled            bool                `json:"enabled"`
	DegradationLevels  []DegradationLevel  `json:"degradation_levels"`
	AutoRecovery       bool                `json:"auto_recovery"`
	RecoveryTimeoutSec int                 `json:"recovery_timeout_seconds"`
}

type DegradationLevel struct {
	Level             int      `json:"level"`
	Name              string   `json:"name"`
	Description       string   `json:"description"`
	DisabledFeatures  []string `json:"disabled_features"`
	ReducedCapacity   int      `json:"reduced_capacity_percent"`
	Priority          int      `json:"priority"`
}

type ServiceDegradationState struct {
	CurrentLevel   int               `json:"current_level"`
	IsDegraded     bool              `json:"is_degraded"`
	AffectedFeatures []string         `json:"affected_features"`
	StartedAt      time.Time          `json:"started_at"`
	Reason         string             `json:"reason"`
}

type CapacityPlanningConfig struct {
	MinReplicas        int     `json:"min_replicas"`
	MaxReplicas        int     `json:"max_replicas"`
	TargetUtilization   float64 `json:"target_utilization_percent"`
	ScaleUpThreshold   float64 `json:"scale_up_threshold_percent"`
	ScaleDownThreshold float64 `json:"scale_down_threshold_percent"`
	ScaleUpCooldown     int     `json:"scale_up_cooldown_seconds"`
	ScaleDownCooldown   int     `json:"scale_down_cooldown_seconds"`
}

type CapacityMetrics struct {
	CurrentReplicas    atomic.Int32 `json:"current_replicas"`
	TargetReplicas      atomic.Int32 `json:"target_replicas"`
	CPUUtilization      float64     `json:"cpu_utilization_percent"`
	MemoryUtilization   float64     `json:"memory_utilization_percent"`
	RequestRate        atomic.Int64  `json:"request_rate_per_sec"`
	QueueDepth         atomic.Int64  `json:"queue_depth"`
	LastScaleTime      atomic.Int64  `json:"last_scale_time"`
}

type ScaleEvent struct {
	ID           string    `json:"id"`
	Timestamp    time.Time `json:"timestamp"`
	Type         string    `json:"type"`
	OldReplicas  int       `json:"old_replicas"`
	NewReplicas  int       `json:"new_replicas"`
	Reason       string    `json:"reason"`
	TriggeredBy  string    `json:"triggered_by"`
}

type AutoRecoveryConfig struct {
	Enabled              bool   `json:"enabled"`
	MaxRetries           int    `json:"max_retries"`
	RetryIntervalSec     int    `json:"retry_interval_seconds"`
	HealthCheckIntervalSec int  `json:"health_check_interval_seconds"`
	TimeoutSec           int    `json:"timeout_seconds"`
}

type RecoveryAction struct {
	ID           string    `json:"id"`
	Type         string    `json:"type"`
	Target       string    `json:"target"`
	Status       string    `json:"status"`
	Attempts     int       `json:"attempts"`
	LastAttempt  time.Time `json:"last_attempt"`
	SuccessTime  time.Time `json:"success_time"`
	ErrorMessage string    `json:"error_message"`
}

type HighAvailabilityService struct {
	mu            sync.RWMutex
	slaConfig     SLAConfig
	slaMetrics    SLAMetrics
	components    map[string]*ServiceComponent
	gracefulConfig GracefulDegradationConfig
	degradationState ServiceDegradationState
	capacityConfig  CapacityPlanningConfig
	capacityMetrics CapacityMetrics
	autoRecoveryConfig AutoRecoveryConfig
	recoveryActions   []RecoveryAction
	scaleEvents    []ScaleEvent
	startedAt      time.Time
}

func NewHighAvailabilityService() *HighAvailabilityService {
	has := &HighAvailabilityService{
		slaConfig: SLAConfig{
			TargetUptime:   99.99,
			MaxDowntimeMin: 4.38,
			ResponseTimeP99: 100,
			ErrorRateMax:   0.01,
		},
		components:      make(map[string]*ServiceComponent),
		gracefulConfig:  GracefulDegradationConfig{
			Enabled:      true,
			AutoRecovery: true,
			RecoveryTimeoutSec: 300,
			DegradationLevels: []DegradationLevel{
				{Level: 0, Name: "Normal", Description: "Full service", DisabledFeatures: []string{}, ReducedCapacity: 100},
				{Level: 1, Name: "Minor", Description: "Non-critical features disabled", DisabledFeatures: []string{"analytics", "advanced_reports"}, ReducedCapacity: 90},
				{Level: 2, Name: "Moderate", Description: "Some features degraded", DisabledFeatures: []string{"analytics", "advanced_reports", "real_time_updates"}, ReducedCapacity: 70},
				{Level: 3, Name: "Severe", Description: "Only core features available", DisabledFeatures: []string{"analytics", "advanced_reports", "real_time_updates", "batch_processing"}, ReducedCapacity: 50},
			},
		},
		capacityConfig: CapacityPlanningConfig{
			MinReplicas:        3,
			MaxReplicas:        20,
			TargetUtilization:  70.0,
			ScaleUpThreshold:   80.0,
			ScaleDownThreshold: 30.0,
			ScaleUpCooldown:    300,
			ScaleDownCooldown: 600,
		},
		capacityMetrics: CapacityMetrics{
			CurrentReplicas: 3,
			TargetReplicas:  3,
		},
		autoRecoveryConfig: AutoRecoveryConfig{
			Enabled:             true,
			MaxRetries:          3,
			RetryIntervalSec:    30,
			HealthCheckIntervalSec: 10,
			TimeoutSec:          60,
		},
		recoveryActions: make([]RecoveryAction, 0),
		scaleEvents:     make([]ScaleEvent, 0),
		startedAt:       time.Now(),
	}

	has.components["api_gateway"] = &ServiceComponent{
		Name:           "API Gateway",
		Status:         HealthStatusHealthy,
		IsCritical:     true,
		LastCheckTime:  time.Now(),
	}
	has.components["verification_engine"] = &ServiceComponent{
		Name:           "Verification Engine",
		Status:         HealthStatusHealthy,
		IsCritical:     true,
		LastCheckTime:  time.Now(),
	}
	has.components["cache_service"] = &ServiceComponent{
		Name:           "Cache Service",
		Status:         HealthStatusHealthy,
		IsCritical:     false,
		LastCheckTime:  time.Now(),
	}

	return has
}

func (has *HighAvailabilityService) GetSLAStatus() map[string]interface{} {
	has.mu.RLock()
	defer has.mu.RUnlock()

	uptimeSeconds := has.slaMetrics.TotalUptimeSeconds.Load()
	downtimeSeconds := has.slaMetrics.TotalDowntimeSeconds.Load()
	totalSeconds := uptimeSeconds + downtimeSeconds

	var currentUptimePercent float64
	if totalSeconds > 0 {
		currentUptimePercent = float64(uptimeSeconds) / float64(totalSeconds) * 100
	}

	daysSinceStart := time.Since(has.startedAt).Hours() / 24
	var projectedDowntime float64
	if daysSinceStart > 0 {
		projectedDowntime = (downtimeSeconds / 3600) / daysSinceStart * 30
	}

	return map[string]interface{}{
		"target_uptime_percent":     has.slaConfig.TargetUptime,
		"current_uptime_percent":     currentUptimePercent,
		"uptime_seconds":            uptimeSeconds,
		"downtime_seconds":          downtimeSeconds,
		"incidents_count":           has.slaMetrics.IncidentsCount.Load(),
		"last_incident_time":        has.slaMetrics.LastIncidentTime.Load(),
		"current_streak_hours":      has.slaMetrics.CurrentStreak.Load(),
		"longest_streak_hours":      has.slaMetrics.LongestStreak.Load(),
		"projected_monthly_downtime_min": projectedDowntime,
		"sla_met":                    currentUptimePercent >= has.slaConfig.TargetUptime,
	}
}

func (has *HighAvailabilityService) RecordUptime(duration time.Duration) {
	has.slaMetrics.TotalUptimeSeconds.Add(int64(duration.Seconds()))

	currentStreak := has.slaMetrics.CurrentStreak.Load()
	newStreak := currentStreak + int64(duration.Hours())
	has.slaMetrics.CurrentStreak.Store(newStreak)

	longestStreak := has.slaMetrics.LongestStreak.Load()
	if newStreak > longestStreak {
		has.slaMetrics.LongestStreak.Store(newStreak)
	}
}

func (has *HighAvailabilityService) RecordDowntime(duration time.Duration) {
	has.slaMetrics.TotalDowntimeSeconds.Add(int64(duration.Seconds()))
	has.slaMetrics.CurrentStreak.Store(0)
	has.slaMetrics.IncidentsCount.Add(1)
	has.slaMetrics.LastIncidentTime.Store(time.Now().Unix())
}

func (has *HighAvailabilityService) UpdateComponentHealth(name string, status ServiceHealthStatus, responseTime int64) {
	has.mu.Lock()
	defer has.mu.Unlock()

	if comp, exists := has.components[name]; exists {
		comp.LastCheckTime = time.Now()
		comp.AvgResponseTime = responseTime

		if status == HealthStatusHealthy {
			comp.SuccessCount++
		} else {
			comp.FailureCount++
		}

		comp.Status = has.calculateComponentStatus(comp)
	}

	has.evaluateDegradation()
}

func (has *HighAvailabilityService) calculateComponentStatus(comp *ServiceComponent) ServiceHealthStatus {
	total := comp.SuccessCount + comp.FailureCount
	if total == 0 {
		return HealthStatusHealthy
	}

	successRate := float64(comp.SuccessCount) / float64(total)

	if successRate >= 0.99 {
		return HealthStatusHealthy
	} else if successRate >= 0.95 {
		return HealthStatusDegraded
	} else if successRate >= 0.80 {
		return HealthStatusCritical
	}

	return HealthStatusDown
}

func (has *HighAvailabilityService) evaluateDegradation() {
	criticalUnhealthy := 0
	nonCriticalUnhealthy := 0

	for _, comp := range has.components {
		if comp.Status != HealthStatusHealthy {
			if comp.IsCritical {
				criticalUnhealthy++
			} else {
				nonCriticalUnhealthy++
			}
		}
	}

	var newLevel int
	var reason string

	if criticalUnhealthy > 0 {
		if criticalUnhealthy >= 2 {
			newLevel = 3
			reason = "Multiple critical components failed"
		} else {
			newLevel = 2
			reason = "Critical component failed"
		}
	} else if nonCriticalUnhealthy > 0 {
		newLevel = 1
		reason = "Non-critical component failed"
	} else {
		newLevel = 0
		reason = "All components healthy"
	}

	if newLevel != has.degradationState.CurrentLevel {
		has.degradationState.CurrentLevel = newLevel
		has.degradationState.IsDegraded = newLevel > 0
		has.degradationState.AffectedFeatures = has.gracefulConfig.DegradationLevels[newLevel].DisabledFeatures
		has.degradationState.Reason = reason
		if newLevel > 0 && has.degradationState.StartedAt.IsZero() {
			has.degradationState.StartedAt = time.Now()
		} else if newLevel == 0 {
			has.degradationState.StartedAt = time.Time{}
		}
	}
}

func (has *HighAvailabilityService) GetDegradationStatus() ServiceDegradationState {
	has.mu.RLock()
	defer has.mu.RUnlock()

	return has.degradationState
}

func (has *HighAvailabilityService) GetAllComponents() []ServiceComponent {
	has.mu.RLock()
	defer has.mu.RUnlock()

	components := make([]ServiceComponent, 0, len(has.components))
	for _, comp := range has.components {
		components = append(components, *comp)
	}

	return components
}

func (has *HighAvailabilityService) GetOverallHealth() ServiceHealthStatus {
	has.mu.RLock()
	defer has.mu.RUnlock()

	criticalHealthy := 0
	criticalTotal := 0

	for _, comp := range has.components {
		if comp.IsCritical {
			criticalTotal++
			if comp.Status == HealthStatusHealthy {
				criticalHealthy++
			}
		}
	}

	if criticalTotal == 0 || criticalHealthy == criticalTotal {
		has.mu.RUnlock()
		has.mu.RLock()
		for _, comp := range has.components {
			if comp.Status == HealthStatusCritical {
				return HealthStatusDegraded
			}
		}
		return HealthStatusHealthy
	}

	if criticalHealthy >= criticalTotal/2 {
		return HealthStatusDegraded
	}

	return HealthStatusCritical
}

func (has *HighAvailabilityService) GetCapacityStatus() map[string]interface{} {
	return map[string]interface{}{
		"current_replicas":       has.capacityMetrics.CurrentReplicas.Load(),
		"target_replicas":         has.capacityMetrics.TargetReplicas.Load(),
		"cpu_utilization_percent": has.capacityMetrics.CPUUtilization.Load(),
		"memory_utilization_percent": has.capacityMetrics.MemoryUtilization.Load(),
		"request_rate_per_sec":   has.capacityMetrics.RequestRate.Load(),
		"queue_depth":            has.capacityMetrics.QueueDepth.Load(),
		"last_scale_time":        has.capacityMetrics.LastScaleTime.Load(),
		"min_replicas":           has.capacityConfig.MinReplicas,
		"max_replicas":           has.capacityConfig.MaxReplicas,
	}
}

func (has *HighAvailabilityService) UpdateCapacityMetrics(cpu, memory float64, requestRate, queueDepth int64) {
	has.capacityMetrics.CPUUtilization = cpu
	has.capacityMetrics.MemoryUtilization = memory
	has.capacityMetrics.RequestRate.Store(requestRate)
	has.capacityMetrics.QueueDepth.Store(queueDepth)

	has.evaluateScaling()
}

func (has *HighAvailabilityService) evaluateScaling() {
	currentReplicas := has.capacityMetrics.CurrentReplicas.Load()
	targetUtilization := has.capacityConfig.TargetUtilization
	scaleUpThreshold := has.capacityConfig.ScaleUpThreshold
	scaleDownThreshold := has.capacityConfig.ScaleDownThreshold

	cpuUtil := has.capacityMetrics.CPUUtilization.Load()
	memUtil := has.capacityMetrics.MemoryUtilization.Load()
	avgUtil := (cpuUtil + memUtil) / 2

	var newTargetReplicas int32

	if avgUtil > scaleUpThreshold {
		newTargetReplicas = int32(math.Ceil(float64(currentReplicas) * avgUtil / targetUtilization))
	} else if avgUtil < scaleDownThreshold {
		newTargetReplicas = int32(math.Floor(float64(currentReplicas) * avgUtil / targetUtilization))
	} else {
		newTargetReplicas = currentReplicas
	}

	if newTargetReplicas < int32(has.capacityConfig.MinReplicas) {
		newTargetReplicas = int32(has.capacityConfig.MinReplicas)
	}
	if newTargetReplicas > int32(has.capacityConfig.MaxReplicas) {
		newTargetReplicas = int32(has.capacityConfig.MaxReplicas)
	}

	if newTargetReplicas != currentReplicas {
		has.scaleReplicas(int(newTargetReplicas), "auto_scaling")
	}
}

func (has *HighAvailabilityService) scaleReplicas(newReplicas int, reason string) {
	has.mu.Lock()
	defer has.mu.Unlock()

	oldReplicas := has.capacityMetrics.CurrentReplicas.Load()
	has.capacityMetrics.CurrentReplicas.Store(int32(newReplicas))
	has.capacityMetrics.TargetReplicas.Store(int32(newReplicas))
	has.capacityMetrics.LastScaleTime.Store(time.Now().Unix())

	event := ScaleEvent{
		ID:          generateID(),
		Timestamp:   time.Now(),
		OldReplicas: int(oldReplicas),
		NewReplicas: newReplicas,
		Reason:      reason,
		TriggeredBy: "system",
	}

	has.scaleEvents = append(has.scaleEvents, event)
}

func (has *HighAvailabilityService) GetScaleHistory(limit int) []ScaleEvent {
	has.mu.RLock()
	defer has.mu.RUnlock()

	if limit <= 0 || limit > len(has.scaleEvents) {
		limit = len(has.scaleEvents)
	}

	events := make([]ScaleEvent, limit)
	copy(events, has.scaleEvents[len(has.scaleEvents)-limit:])

	return events
}

func (has *HighAvailabilityService) StartRecovery(component string) *RecoveryAction {
	has.mu.Lock()
	defer has.mu.Unlock()

	action := &RecoveryAction{
		ID:          generateID(),
		Type:        "restart",
		Target:      component,
		Status:      "in_progress",
		Attempts:    0,
		LastAttempt: time.Now(),
	}

	has.recoveryActions = append(has.recoveryActions, *action)

	return action
}

func (has *HighAvailabilityService) UpdateRecoveryStatus(actionID string, success bool, errorMsg string) {
	has.mu.Lock()
	defer has.mu.Unlock()

	for i := range has.recoveryActions {
		if has.recoveryActions[i].ID == actionID {
			has.recoveryActions[i].Attempts++
			has.recoveryActions[i].LastAttempt = time.Now()

			if success {
				has.recoveryActions[i].Status = "success"
				has.recoveryActions[i].SuccessTime = time.Now()
			} else {
				has.recoveryActions[i].ErrorMessage = errorMsg
				if has.recoveryActions[i].Attempts >= has.autoRecoveryConfig.MaxRetries {
					has.recoveryActions[i].Status = "failed"
				}
			}

			break
		}
	}
}

func (has *HighAvailabilityService) GetRecoveryActions() []RecoveryAction {
	has.mu.RLock()
	defer has.mu.RUnlock()

	actions := make([]RecoveryAction, len(has.recoveryActions))
	copy(actions, has.recoveryActions)

	return actions
}

func (has *HighAvailabilityService) HealthCheck(ctx context.Context) map[string]interface{} {
	status := has.GetOverallHealth()
	slaStatus := has.GetSLAStatus()
	capacityStatus := has.GetCapacityStatus()
	degradationState := has.GetDegradationStatus()

	result := map[string]interface{}{
		"overall_status":     status,
		"sla_status":         slaStatus,
		"capacity_status":    capacityStatus,
		"degradation_state":  degradationState,
		"components_count":   len(has.components),
		"timestamp":          time.Now().Unix(),
	}

	return result
}

func generateID() string {
	return time.Now().Format("20060102150405") + "-" + randomString(8)
}

func randomString(n int) string {
	const letters = "abcdefghijklmnopqrstuvwxyz0123456789"
	b := make([]byte, n)
	for i := range b {
		b[i] = letters[time.Now().UnixNano()%int64(len(letters))]
	}
	return string(b)
}
