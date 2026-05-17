package highavailability

import (
	"context"
	"fmt"
	"strconv"
	"sync"
	"sync/atomic"
	"time"

	"github.com/google/uuid"
)

type InstanceState string

const (
	InstanceStateActive   InstanceState = "active"
	InstanceStateStandby  InstanceState = "standby"
	InstanceStateDraining InstanceState = "draining"
	InstanceStateOffline  InstanceState = "offline"
)

type InstanceInfo struct {
	ID           string
	Name         string
	Host         string
	Port         int
	Weight       int
	State        InstanceState
	StartTime    time.Time
	LastHeartbeat time.Time
	Priority     int
	Tags         []string
	Metadata     map[string]string
	Region       string
	Zone         string
}

type InstanceRegistry struct {
	instances map[string]*InstanceInfo
	mu        sync.RWMutex
	selfID    string
	ttl       time.Duration
	cleanupInterval time.Duration
	stopCh    chan struct{}
}

var (
	globalRegistry *InstanceRegistry
	registryOnce   sync.Once
)

func GetInstanceRegistry() *InstanceRegistry {
	registryOnce.Do(func() {
		globalRegistry = NewInstanceRegistry(30 * time.Second)
	})
	return globalRegistry
}

func NewInstanceRegistry(ttl time.Duration) *InstanceRegistry {
	return &InstanceRegistry{
		instances:       make(map[string]*InstanceInfo),
		ttl:             ttl,
		cleanupInterval: 10 * time.Second,
		stopCh:          make(chan struct{}),
	}
}

func (r *InstanceRegistry) Register(instance *InstanceInfo) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if instance.ID == "" {
		instance.ID = uuid.New().String()
	}
	if instance.StartTime.IsZero() {
		instance.StartTime = time.Now()
	}
	instance.LastHeartbeat = time.Now()
	if instance.State == "" {
		instance.State = InstanceStateActive
	}

	r.instances[instance.ID] = instance
	return nil
}

func (r *InstanceRegistry) Unregister(instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, ok := r.instances[instanceID]; !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	delete(r.instances, instanceID)
	return nil
}

func (r *InstanceRegistry) GetInstance(instanceID string) (*InstanceInfo, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instance, ok := r.instances[instanceID]
	return instance, ok
}

func (r *InstanceRegistry) GetAllInstances() []*InstanceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*InstanceInfo, 0, len(r.instances))
	for _, instance := range r.instances {
		instances = append(instances, instance)
	}
	return instances
}

func (r *InstanceRegistry) GetActiveInstances() []*InstanceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*InstanceInfo, 0)
	for _, instance := range r.instances {
		if instance.State == InstanceStateActive {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (r *InstanceRegistry) UpdateHeartbeat(instanceID string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[instanceID]
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	instance.LastHeartbeat = time.Now()
	return nil
}

func (r *InstanceRegistry) UpdateState(instanceID string, state InstanceState) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	instance, ok := r.instances[instanceID]
	if !ok {
		return fmt.Errorf("instance not found: %s", instanceID)
	}

	instance.State = state
	return nil
}

func (r *InstanceRegistry) GetInstanceCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return len(r.instances)
}

func (r *InstanceRegistry) GetActiveCount() int {
	r.mu.RLock()
	defer r.mu.RUnlock()

	count := 0
	for _, instance := range r.instances {
		if instance.State == InstanceStateActive {
			count++
		}
	}
	return count
}

func (r *InstanceRegistry) StartCleanup(ctx context.Context) {
	ticker := time.NewTicker(r.cleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return
		case <-r.stopCh:
			return
		case <-ticker.C:
			r.cleanupStaleInstances()
		}
	}
}

func (r *InstanceRegistry) StopCleanup() {
	close(r.stopCh)
}

func (r *InstanceRegistry) cleanupStaleInstances() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for _, instance := range r.instances {
		if now.Sub(instance.LastHeartbeat) > r.ttl {
			instance.State = InstanceStateOffline
		}
	}
}

func (r *InstanceRegistry) SetSelfID(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.selfID = id
}

func (r *InstanceRegistry) GetSelfID() string {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.selfID
}

func (r *InstanceRegistry) GetInstancesByTag(tag string) []*InstanceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*InstanceInfo, 0)
	for _, instance := range r.instances {
		for _, t := range instance.Tags {
			if t == tag {
				instances = append(instances, instance)
				break
			}
		}
	}
	return instances
}

func (r *InstanceRegistry) GetInstancesByRegion(region string) []*InstanceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*InstanceInfo, 0)
	for _, instance := range r.instances {
		if instance.Region == region {
			instances = append(instances, instance)
		}
	}
	return instances
}

func (r *InstanceRegistry) GetInstancesByZone(zone string) []*InstanceInfo {
	r.mu.RLock()
	defer r.mu.RUnlock()

	instances := make([]*InstanceInfo, 0)
	for _, instance := range r.instances {
		if instance.Zone == zone {
			instances = append(instances, instance)
		}
	}
	return instances
}

type InstanceSelector struct {
	registry *InstanceRegistry
	strategy SelectionStrategy
	mu       sync.RWMutex
}

type SelectionStrategy string

const (
	InstanceStrategyWeightedRR  SelectionStrategy = "weighted_round_robin"
	InstanceStrategyLeastConn  SelectionStrategy = "least_connections"
	InstanceStrategyRandom     SelectionStrategy = "random"
	InstanceStrategyPriority   SelectionStrategy = "priority"
	InstanceStrategyRegion    SelectionStrategy = "region"
)

func NewInstanceSelector(registry *InstanceRegistry, strategy SelectionStrategy) *InstanceSelector {
	return &InstanceSelector{
		registry: registry,
		strategy: strategy,
	}
}

func (s *InstanceSelector) Select(ctx context.Context) (*InstanceInfo, error) {
	s.mu.RLock()
	strategy := s.strategy
	s.mu.RUnlock()

	instances := s.registry.GetActiveInstances()
	if len(instances) == 0 {
		return nil, fmt.Errorf("no active instances available")
	}

	switch strategy {
	case InstanceStrategyWeightedRR:
		return s.weightedRoundRobin(instances), nil
	case InstanceStrategyLeastConn:
		return s.leastConnections(instances), nil
	case InstanceStrategyRandom:
		return s.randomSelect(instances), nil
	case InstanceStrategyPriority:
		return s.prioritySelect(instances), nil
	case InstanceStrategyRegion:
		region := ctx.Value("region")
		if region == nil {
			return s.prioritySelect(instances), nil
		}
		return s.regionSelect(instances, region.(string)), nil
	default:
		return s.weightedRoundRobin(instances), nil
	}
}

func (s *InstanceSelector) weightedRoundRobin(instances []*InstanceInfo) *InstanceInfo {
	var totalWeight int
	for _, inst := range instances {
		totalWeight += inst.Weight
	}

	if totalWeight == 0 {
		return instances[0]
	}

	idx := time.Now().UnixNano() % int64(totalWeight)
	var sum int
	for _, inst := range instances {
		sum += inst.Weight
		if int64(sum) > idx {
			return inst
		}
	}

	return instances[0]
}

func (s *InstanceSelector) leastConnections(instances []*InstanceInfo) *InstanceInfo {
	var selected *InstanceInfo
	minConnections := int64(^uint64(0) >> 1)

	for _, inst := range instances {
		connStr := inst.Metadata["connections"]
		connCount := int64(0)
		if connStr != "" {
			if v, err := strconv.ParseInt(connStr, 10, 64); err == nil {
				connCount = v
			}
		}
		if connCount < minConnections {
			minConnections = connCount
			selected = inst
		}
	}

	if selected == nil {
		selected = instances[0]
	}
	return selected
}

func (s *InstanceSelector) randomSelect(instances []*InstanceInfo) *InstanceInfo {
	idx := time.Now().UnixNano() % int64(len(instances))
	return instances[idx]
}

func (s *InstanceSelector) prioritySelect(instances []*InstanceInfo) *InstanceInfo {
	var selected *InstanceInfo
	maxPriority := -1

	for _, inst := range instances {
		if inst.Priority > maxPriority {
			maxPriority = inst.Priority
			selected = inst
		}
	}

	if selected == nil {
		selected = instances[0]
	}
	return selected
}

func (s *InstanceSelector) regionSelect(instances []*InstanceInfo, region string) *InstanceInfo {
	var regionInstances []*InstanceInfo
	for _, inst := range instances {
		if inst.Region == region {
			regionInstances = append(regionInstances, inst)
		}
	}

	if len(regionInstances) == 0 {
		return instances[0]
	}

	return s.prioritySelect(regionInstances)
}

func (s *InstanceSelector) SetStrategy(strategy SelectionStrategy) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.strategy = strategy
}

type InstanceMetrics struct {
	instanceID    string
	requestCount  uint64
	errorCount    uint64
	successCount  uint64
	latencySum    uint64
	latencyCount  uint64
	mu            sync.RWMutex
}

func NewInstanceMetrics(instanceID string) *InstanceMetrics {
	return &InstanceMetrics{
		instanceID: instanceID,
	}
}

func (m *InstanceMetrics) RecordRequest() {
	atomic.AddUint64(&m.requestCount, 1)
}

func (m *InstanceMetrics) RecordSuccess(latency time.Duration) {
	atomic.AddUint64(&m.successCount, 1)
	atomic.AddUint64(&m.latencySum, uint64(latency.Milliseconds()))
	atomic.AddUint64(&m.latencyCount, 1)
}

func (m *InstanceMetrics) RecordError() {
	atomic.AddUint64(&m.errorCount, 1)
}

func (m *InstanceMetrics) GetStats() InstanceMetricsSnapshot {
	return InstanceMetricsSnapshot{
		InstanceID:   m.instanceID,
		RequestCount:  atomic.LoadUint64(&m.requestCount),
		SuccessCount:  atomic.LoadUint64(&m.successCount),
		ErrorCount:    atomic.LoadUint64(&m.errorCount),
		SuccessRate:   m.calculateSuccessRate(),
		AvgLatency:    m.calculateAvgLatency(),
	}
}

func (m *InstanceMetrics) calculateSuccessRate() float64 {
	total := atomic.LoadUint64(&m.requestCount)
	if total == 0 {
		return 0
	}
	success := atomic.LoadUint64(&m.successCount)
	return float64(success) / float64(total) * 100
}

func (m *InstanceMetrics) calculateAvgLatency() time.Duration {
	sum := atomic.LoadUint64(&m.latencySum)
	count := atomic.LoadUint64(&m.latencyCount)
	if count == 0 {
		return 0
	}
	return time.Duration(sum/count) * time.Millisecond
}

type InstanceMetricsSnapshot struct {
	InstanceID  string        `json:"instance_id"`
	RequestCount uint64        `json:"request_count"`
	SuccessCount uint64        `json:"success_count"`
	ErrorCount   uint64        `json:"error_count"`
	SuccessRate  float64       `json:"success_rate"`
	AvgLatency   time.Duration `json:"avg_latency"`
}

type ConfigConsistencyManager struct {
	configs map[string]*ConfigVersion
	mu      sync.RWMutex
	version uint64
}

type ConfigVersion struct {
	Key       string
	Value     interface{}
	Version   uint64
	Timestamp time.Time
	Source    string
}

func NewConfigConsistencyManager() *ConfigConsistencyManager {
	return &ConfigConsistencyManager{
		configs: make(map[string]*ConfigVersion),
	}
}

func (cm *ConfigConsistencyManager) SetConfig(key string, value interface{}, source string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.version++
	cm.configs[key] = &ConfigVersion{
		Key:       key,
		Value:     value,
		Version:   cm.version,
		Timestamp: time.Now(),
		Source:    source,
	}
}

func (cm *ConfigConsistencyManager) GetConfig(key string) (interface{}, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, ok := cm.configs[key]
	if !ok {
		return nil, false
	}
	return config.Value, true
}

func (cm *ConfigConsistencyManager) GetConfigVersion(key string) (uint64, bool) {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	config, ok := cm.configs[key]
	if !ok {
		return 0, false
	}
	return config.Version, true
}

func (cm *ConfigConsistencyManager) GetAllConfigs() map[string]*ConfigVersion {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	result := make(map[string]*ConfigVersion, len(cm.configs))
	for k, v := range cm.configs {
		result[k] = v
	}
	return result
}

func (cm *ConfigConsistencyManager) GetGlobalVersion() uint64 {
	cm.mu.RLock()
	defer cm.mu.RUnlock()
	return cm.version
}

type DistributedLock struct {
	key       string
	holderID  string
	acquired  bool
	expiresAt time.Time
}

type LockManager struct {
	locks map[string]*DistributedLock
	mu    sync.RWMutex
	ttl   time.Duration
}

func NewLockManager(ttl time.Duration) *LockManager {
	return &LockManager{
		locks: make(map[string]*DistributedLock),
		ttl:   ttl,
	}
}

func (lm *LockManager) Acquire(key string, holderID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[key]
	if exists {
		if time.Now().Before(lock.expiresAt) && lock.holderID != holderID {
			return false
		}
	}

	lm.locks[key] = &DistributedLock{
		key:       key,
		holderID:  holderID,
		acquired:  true,
		expiresAt: time.Now().Add(lm.ttl),
	}
	return true
}

func (lm *LockManager) Release(key string, holderID string) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[key]
	if !exists || lock.holderID != holderID {
		return false
	}

	delete(lm.locks, key)
	return true
}

func (lm *LockManager) IsLocked(key string) bool {
	lm.mu.RLock()
	defer lm.mu.RUnlock()

	lock, exists := lm.locks[key]
	if !exists {
		return false
	}
	return time.Now().Before(lock.expiresAt)
}

func (lm *LockManager) Extend(key string, holderID string, ttl time.Duration) bool {
	lm.mu.Lock()
	defer lm.mu.Unlock()

	lock, exists := lm.locks[key]
	if !exists || lock.holderID != holderID {
		return false
	}

	lock.expiresAt = time.Now().Add(ttl)
	return true
}

type SessionAffinityManager struct {
	instanceMapping map[string]string
	mu              sync.RWMutex
	ttl             time.Duration
}

func NewSessionAffinityManager(ttl time.Duration) *SessionAffinityManager {
	return &SessionAffinityManager{
		instanceMapping: make(map[string]string),
		ttl:             ttl,
	}
}

func (sam *SessionAffinityManager) SetSessionInstance(sessionID string, instanceID string) {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	sam.instanceMapping[sessionID] = instanceID
}

func (sam *SessionAffinityManager) GetSessionInstance(sessionID string) (string, bool) {
	sam.mu.RLock()
	defer sam.mu.RUnlock()
	instanceID, ok := sam.instanceMapping[sessionID]
	return instanceID, ok
}

func (sam *SessionAffinityManager) RemoveSession(sessionID string) {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	delete(sam.instanceMapping, sessionID)
}

func (sam *SessionAffinityManager) Clear() {
	sam.mu.Lock()
	defer sam.mu.Unlock()
	sam.instanceMapping = make(map[string]string)
}
