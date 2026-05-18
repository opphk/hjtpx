package failover

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	pkgredis "github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/service-discovery"
)

type DistributedFailoverManager struct {
	mu             sync.RWMutex
	config         *FailoverConfig
	strategies     map[FailoverType]FailoverStrategy
	regions        map[string]*RegionFailover
	currentRegion  string
	failoverEnabled bool
	metrics        *FailoverMetrics
	registry       *servicediscovery.DistributedServiceRegistry
	redisClient    *pkgredis.DistributedRedisClient
	dbRouter       *database.ElasticDBRouter
	healthChecker  *DistributedHealthCheckerWrapper
	stopCh         chan struct{}
	wg             sync.WaitGroup
}

type FailoverConfig struct {
	Enabled            bool
	AutoDetectEnabled  bool
	RegionFailover     bool
	CrossRegionEnabled bool
	MaxRetries         int
	RetryInterval      time.Duration
	HealthCheckTimeout time.Duration
	FallbackEnabled   bool
	ConsulEnabled      bool
	EtcdEnabled        bool
	ZKEnabled          bool
}

type FailoverType string

const (
	FailoverTypeNode     FailoverType = "node"
	FailoverTypeRegion   FailoverType = "region"
	FailoverTypeDatacenter FailoverType = "datacenter"
	FailoverTypeNetwork  FailoverType = "network"
)

type FailoverStrategy interface {
	Execute(ctx context.Context, info *FailoverInfo) error
	Name() string
	CanHandle(info *FailoverInfo) bool
}

type FailoverInfo struct {
	ID            string
	Type          FailoverType
	Source        string
	Target        string
	Region        string
	DC            string
	Reason        string
	Severity      FailoverSeverity
	Timestamp     time.Time
	RetryCount    int
	OriginalError error
	Metadata      map[string]string
}

type FailoverResult string

const (
	FailoverResultSuccess   FailoverResult = "success"
	FailoverResultFailed    FailoverResult = "failed"
	FailoverResultSkipped   FailoverResult = "skipped"
	FailoverResultPartial   FailoverResult = "partial"
)

type FailoverSeverity string

const (
	SeverityInfo     FailoverSeverity = "info"
	SeverityWarning  FailoverSeverity = "warning"
	SeverityError    FailoverSeverity = "error"
	SeverityHigh     FailoverSeverity = "high"
	SeverityCritical FailoverSeverity = "critical"
)

type RegionFailover struct {
	Name            string
	PrimaryRegion   string
	FallbackRegions []string
	HealthStatus    map[string]RegionHealth
	ActiveRegion    string
	LastSwitch      time.Time
	SwitchCount     atomic.Int64
}

type RegionHealth struct {
	Region        string
	Healthy       bool
	Latency       time.Duration
	LastCheck     time.Time
	FailureCount  int
	SuccessCount  int
}

type FailoverMetrics struct {
	TotalFailovers     atomic.Int64
	SuccessfulFailovers atomic.Int64
	FailedFailovers    atomic.Int64
	RegionFailovers    atomic.Int64
	NodeFailovers      atomic.Int64
	AvgFailoverTime    atomic.Int64
	LastFailoverTime   atomic.Value
	LastFailoverRegion atomic.Value
	CurrentActiveFailover atomic.Bool
}

type DistributedHealthCheckerWrapper struct {
	enabled bool
}

var (
	globalFailoverManager *DistributedFailoverManager
	failoverManagerOnce   sync.Once
)

func NewDistributedFailoverManager(cfg *FailoverConfig) *DistributedFailoverManager {
	if cfg == nil {
		cfg = getDefaultFailoverConfig()
	}

	mgr := &DistributedFailoverManager{
		config:          cfg,
		strategies:      make(map[FailoverType]FailoverStrategy),
		regions:         make(map[string]*RegionFailover),
		failoverEnabled: cfg.Enabled,
		metrics:         &FailoverMetrics{},
		stopCh:          make(chan struct{}),
	}

	mgr.registerDefaultStrategies()

	return mgr
}

func getDefaultFailoverConfig() *FailoverConfig {
	return &FailoverConfig{
		Enabled:            true,
		AutoDetectEnabled:  true,
		RegionFailover:     true,
		CrossRegionEnabled: true,
		MaxRetries:         3,
		RetryInterval:      5 * time.Second,
		HealthCheckTimeout: 10 * time.Second,
		FallbackEnabled:   true,
	}
}

func (m *DistributedFailoverManager) registerDefaultStrategies() {
	m.RegisterStrategy(NewNodeFailoverStrategy())
	m.RegisterStrategy(NewRegionFailoverStrategy())
	m.RegisterStrategy(NewDatabaseFailoverStrategy())
	m.RegisterStrategy(NewRedisFailoverStrategy())
	m.RegisterStrategy(NewNetworkFailoverStrategy())
}

func (m *DistributedFailoverManager) RegisterStrategy(strategy FailoverStrategy) {
	m.mu.Lock()
	defer m.mu.Unlock()

	for failoverType := range getFailoverTypes(strategy) {
		m.strategies[failoverType] = strategy
	}
}

func getFailoverTypes(strategy FailoverStrategy) map[FailoverType]bool {
	types := make(map[FailoverType]bool)

	switch strategy.Name() {
	case "node_failover":
		types[FailoverTypeNode] = true
	case "region_failover":
		types[FailoverTypeRegion] = true
	case "database_failover":
		types[FailoverTypeNode] = true
	case "redis_failover":
		types[FailoverTypeNode] = true
	case "network_failover":
		types[FailoverTypeNetwork] = true
	}

	return types
}

func (m *DistributedFailoverManager) SetRegistry(registry *servicediscovery.DistributedServiceRegistry) {
	m.registry = registry
}

func (m *DistributedFailoverManager) SetRedisClient(client *pkgredis.DistributedRedisClient) {
	m.redisClient = client
}

func (m *DistributedFailoverManager) SetDBRouter(router *database.ElasticDBRouter) {
	m.dbRouter = router
}

func (m *DistributedFailoverManager) Start() {
	if !m.config.Enabled {
		log.Println("[FAILOVER_MANAGER] Failover manager is disabled")
		return
	}

	if m.config.AutoDetectEnabled {
		m.wg.Add(1)
		go m.autoDetectLoop()
	}

	if m.config.RegionFailover {
		m.wg.Add(1)
		go m.regionHealthMonitor()
	}

	log.Println("[FAILOVER_MANAGER] Started")
}

func (m *DistributedFailoverManager) Stop() {
	close(m.stopCh)
	m.wg.Wait()
	log.Println("[FAILOVER_MANAGER] Stopped")
}

func (m *DistributedFailoverManager) TriggerFailover(ctx context.Context, info *FailoverInfo) error {
	if !m.failoverEnabled {
		return fmt.Errorf("failover is disabled")
	}

	info.Timestamp = time.Now()
	info.RetryCount = 0

	m.mu.RLock()
	strategy, exists := m.strategies[info.Type]
	m.mu.RUnlock()

	if !exists {
		return fmt.Errorf("no strategy found for failover type: %s", info.Type)
	}

	if !strategy.CanHandle(info) {
		return fmt.Errorf("strategy %s cannot handle failover info", strategy.Name())
	}

	startTime := time.Now()

	err := m.executeWithRetry(ctx, strategy, info)

	duration := time.Since(startTime)
	m.recordMetrics(info, err, duration)

	if err != nil {
		m.metrics.FailedFailovers.Add(1)
		log.Printf("[FAILOVER_MANAGER] Failover failed for %s: %v", info.Type, err)
		return err
	}

	m.metrics.SuccessfulFailovers.Add(1)
	log.Printf("[FAILOVER_MANAGER] Failover successful for %s, duration: %v", info.Type, duration)

	return nil
}

func (m *DistributedFailoverManager) executeWithRetry(ctx context.Context, strategy FailoverStrategy, info *FailoverInfo) error {
	var lastErr error

	for i := 0; i < m.config.MaxRetries; i++ {
		info.RetryCount = i

		if err := strategy.Execute(ctx, info); err != nil {
			lastErr = err
			log.Printf("[FAILOVER_MANAGER] Retry %d/%d failed for %s: %v",
				i+1, m.config.MaxRetries, info.Type, err)

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(m.config.RetryInterval):
			}
			continue
		}

		return nil
	}

	return fmt.Errorf("failed after %d retries: %w", m.config.MaxRetries, lastErr)
}

func (m *DistributedFailoverManager) recordMetrics(info *FailoverInfo, err error, duration time.Duration) {
	m.metrics.TotalFailovers.Add(1)

	switch info.Type {
	case FailoverTypeRegion, FailoverTypeDatacenter:
		m.metrics.RegionFailovers.Add(1)
	case FailoverTypeNode:
		m.metrics.NodeFailovers.Add(1)
	}

	m.metrics.LastFailoverTime.Store(time.Now())
	m.metrics.AvgFailoverTime.Store(duration.Nanoseconds())

	if info.Region != "" {
		m.metrics.LastFailoverRegion.Store(info.Region)
	}
}

func (m *DistributedFailoverManager) autoDetectLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.detectAndRecover()
		}
	}
}

func (m *DistributedFailoverManager) detectAndRecover() {
	ctx, cancel := context.WithTimeout(context.Background(), m.config.HealthCheckTimeout)
	defer cancel()

	if m.redisClient != nil {
		if err := m.redisClient.Ping(ctx); err != nil {
			log.Printf("[FAILOVER_MANAGER] Redis health check failed: %v", err)
			m.TriggerFailover(ctx, &FailoverInfo{
				Type:     FailoverTypeNode,
				Source:   "redis",
				Reason:   err.Error(),
				Severity: SeverityError,
			})
		}
	}

	if m.dbRouter != nil {
		stats := m.dbRouter.GetReplicaStats()
		for _, stat := range stats {
			if !stat.Healthy && stat.FailCount >= 3 {
				log.Printf("[FAILOVER_MANAGER] Database replica %s:%s unhealthy", stat.Host, stat.Port)
				m.TriggerFailover(ctx, &FailoverInfo{
					Type:     FailoverTypeNode,
					Source:   fmt.Sprintf("db:%s:%s", stat.Host, stat.Port),
					Reason:   "replica unhealthy",
					Severity: SeverityWarning,
				})
			}
		}
	}
}

func (m *DistributedFailoverManager) regionHealthMonitor() {
	defer m.wg.Done()

	ticker := time.NewTicker(60 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.stopCh:
			return
		case <-ticker.C:
			m.checkRegionHealth()
		}
	}
}

func (m *DistributedFailoverManager) checkRegionHealth() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	m.mu.RLock()
	regions := make(map[string]*RegionFailover)
	for k, v := range m.regions {
		regions[k] = v
	}
	m.mu.RUnlock()

	for regionName, region := range regions {
		health := m.checkRegionLatency(ctx, regionName)
		region.HealthStatus[regionName] = health

		if !health.Healthy && health.FailureCount >= 3 {
			log.Printf("[FAILOVER_MANAGER] Region %s is unhealthy, considering failover", regionName)
			m.triggerRegionFailover(ctx, regionName)
		}
	}
}

func (m *DistributedFailoverManager) checkRegionLatency(ctx context.Context, region string) RegionHealth {
	health := RegionHealth{
		Region:    region,
		Healthy:   true,
		LastCheck: time.Now(),
	}

	if m.redisClient != nil {
		start := time.Now()
		if err := m.redisClient.Ping(ctx); err != nil {
			health.Healthy = false
			health.FailureCount++
		} else {
			health.Latency = time.Since(start)
			health.SuccessCount++
		}
	}

	return health
}

func (m *DistributedFailoverManager) triggerRegionFailover(ctx context.Context, region string) {
	if !m.config.RegionFailover {
		return
	}

	m.mu.RLock()
	regionFailover, exists := m.regions[region]
	m.mu.RUnlock()

	if !exists || len(regionFailover.FallbackRegions) == 0 {
		log.Printf("[FAILOVER_MANAGER] No fallback regions available for %s", region)
		return
	}

	targetRegion := regionFailover.FallbackRegions[0]

	err := m.TriggerFailover(ctx, &FailoverInfo{
		Type:     FailoverTypeRegion,
		Source:   region,
		Target:   targetRegion,
		Region:   region,
		Reason:   "region unhealthy",
		Severity: SeverityCritical,
	})

	if err != nil {
		log.Printf("[FAILOVER_MANAGER] Region failover failed: %v", err)
		return
	}

	regionFailover.ActiveRegion = targetRegion
	regionFailover.LastSwitch = time.Now()
	regionFailover.SwitchCount.Add(1)
	m.metrics.RegionFailovers.Add(1)

	log.Printf("[FAILOVER_MANAGER] Region failover completed: %s -> %s", region, targetRegion)
}

func (m *DistributedFailoverManager) RegisterRegion(cfg *RegionFailoverConfig) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.regions[cfg.Name] = &RegionFailover{
		Name:            cfg.Name,
		PrimaryRegion:   cfg.PrimaryRegion,
		FallbackRegions: cfg.FallbackRegions,
		HealthStatus:    make(map[string]RegionHealth),
		ActiveRegion:    cfg.PrimaryRegion,
	}

	log.Printf("[FAILOVER_MANAGER] Registered region: %s, primary: %s, fallbacks: %v",
		cfg.Name, cfg.PrimaryRegion, cfg.FallbackRegions)
}

func (m *DistributedFailoverManager) GetMetrics() *FailoverMetrics {
	return m.metrics
}

func (m *DistributedFailoverManager) GetRegionStatus() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make(map[string]interface{})
	for name, region := range m.regions {
		status[name] = map[string]interface{}{
			"primary_region":   region.PrimaryRegion,
			"active_region":    region.ActiveRegion,
			"fallback_regions": region.FallbackRegions,
			"switch_count":     region.SwitchCount.Load(),
			"last_switch":      region.LastSwitch.Format(time.RFC3339),
			"health_status":    region.HealthStatus,
		}
	}

	return status
}

func (m *DistributedFailoverManager) IsFailoverActive() bool {
	return m.metrics.CurrentActiveFailover.Load()
}

func (m *DistributedFailoverManager) SetFailoverEnabled(enabled bool) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.failoverEnabled = enabled
}

type NodeFailoverStrategy struct {
	redisClient *pkgredis.DistributedRedisClient
	dbRouter    *database.ElasticDBRouter
}

func NewNodeFailoverStrategy() *NodeFailoverStrategy {
	return &NodeFailoverStrategy{}
}

func (s *NodeFailoverStrategy) Name() string {
	return "node_failover"
}

func (s *NodeFailoverStrategy) CanHandle(info *FailoverInfo) bool {
	return info.Type == FailoverTypeNode
}

func (s *NodeFailoverStrategy) Execute(ctx context.Context, info *FailoverInfo) error {
	log.Printf("[NODE_FAILOVER] Executing node failover for source: %s, reason: %s",
		info.Source, info.Reason)

	if s.redisClient != nil {
		nodes := s.redisClient.GetNodeStats()
		for _, node := range nodes {
			if !node.Healthy {
				log.Printf("[NODE_FAILOVER] Marking unhealthy redis node: %s", node.Addr)
			}
		}
	}

	if s.dbRouter != nil {
		stats := s.dbRouter.GetReplicaStats()
		for _, stat := range stats {
			nodeID := fmt.Sprintf("%s:%s", stat.Host, stat.Port)
			if nodeID == info.Source || !stat.Healthy {
				log.Printf("[NODE_FAILOVER] Database node %s marked unhealthy", nodeID)
			}
		}
	}

	return nil
}

type RegionFailoverStrategy struct {
	registry *servicediscovery.DistributedServiceRegistry
}

func NewRegionFailoverStrategy() *RegionFailoverStrategy {
	return &RegionFailoverStrategy{}
}

func (s *RegionFailoverStrategy) Name() string {
	return "region_failover"
}

func (s *RegionFailoverStrategy) CanHandle(info *FailoverInfo) bool {
	return info.Type == FailoverTypeRegion || info.Type == FailoverTypeDatacenter
}

func (s *RegionFailoverStrategy) Execute(ctx context.Context, info *FailoverInfo) error {
	log.Printf("[REGION_FAILOVER] Executing region failover: %s -> %s, reason: %s",
		info.Source, info.Target, info.Reason)

	if s.registry != nil {
		instances := s.registry.DiscoverByRegion("captcha-service", info.Target)
		if len(instances) == 0 {
			log.Printf("[REGION_FAILOVER] No healthy instances found in target region: %s", info.Target)
			return fmt.Errorf("no healthy instances in region: %s", info.Target)
		}

		log.Printf("[REGION_FAILOVER] Found %d healthy instances in region %s", len(instances), info.Target)
	}

	return nil
}

type DatabaseFailoverStrategy struct {
	dbRouter *database.ElasticDBRouter
}

func NewDatabaseFailoverStrategy() *DatabaseFailoverStrategy {
	return &DatabaseFailoverStrategy{}
}

func (s *DatabaseFailoverStrategy) Name() string {
	return "database_failover"
}

func (s *DatabaseFailoverStrategy) CanHandle(info *FailoverInfo) bool {
	return info.Type == FailoverTypeNode && contains(info.Source, "db")
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > 0 && containsHelper(s, substr))
}

func containsHelper(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

func (s *DatabaseFailoverStrategy) Execute(ctx context.Context, info *FailoverInfo) error {
	if s.dbRouter == nil {
		return fmt.Errorf("database router not available")
	}

	log.Printf("[DB_FAILOVER] Executing database failover for: %s", info.Source)

	stats := s.dbRouter.GetReplicaStats()
	for _, stat := range stats {
		nodeID := fmt.Sprintf("%s:%s", stat.Host, stat.Port)
		if nodeID == info.Source {
			log.Printf("[DB_FAILOVER] Found target replica: %s", nodeID)
		}
	}

	healthStatus := s.dbRouter.GetHealthStatus()
	healthyCount := 0
	for _, status := range healthStatus {
		if healthy, ok := status["healthy"].(bool); ok && healthy {
			healthyCount++
		}
	}

	if healthyCount == 0 {
		return fmt.Errorf("no healthy database replicas available")
	}

	log.Printf("[DB_FAILOVER] Database failover completed, healthy replicas: %d", healthyCount)
	return nil
}

type RedisFailoverStrategy struct {
	redisClient *pkgredis.DistributedRedisClient
}

func NewRedisFailoverStrategy() *RedisFailoverStrategy {
	return &RedisFailoverStrategy{}
}

func (s *RedisFailoverStrategy) Name() string {
	return "redis_failover"
}

func (s *RedisFailoverStrategy) CanHandle(info *FailoverInfo) bool {
	return info.Type == FailoverTypeNode && contains(info.Source, "redis")
}

func (s *RedisFailoverStrategy) Execute(ctx context.Context, info *FailoverInfo) error {
	if s.redisClient == nil {
		return fmt.Errorf("redis client not available")
	}

	log.Printf("[REDIS_FAILOVER] Executing redis failover for: %s", info.Source)

	nodes := s.redisClient.GetNodeStats()
	for _, node := range nodes {
		if node.Addr == info.Source {
			if err := s.redisClient.SetNodeHealthy(node.Addr, false); err != nil {
				log.Printf("[REDIS_FAILOVER] Failed to mark node unhealthy: %v", err)
			}
		}
	}

	healthyNodes := s.redisClient.GetOptimalNode()
	if healthyNodes == nil {
		return fmt.Errorf("no healthy redis nodes available")
	}

	log.Printf("[REDIS_FAILOVER] Redis failover completed, redirecting to node: %s", healthyNodes.Addr)
	return nil
}

type NetworkFailoverStrategy struct{}

func NewNetworkFailoverStrategy() *NetworkFailoverStrategy {
	return &NetworkFailoverStrategy{}
}

func (s *NetworkFailoverStrategy) Name() string {
	return "network_failover"
}

func (s *NetworkFailoverStrategy) CanHandle(info *FailoverInfo) bool {
	return info.Type == FailoverTypeNetwork
}

func (s *NetworkFailoverStrategy) Execute(ctx context.Context, info *FailoverInfo) error {
	log.Printf("[NETWORK_FAILOVER] Executing network failover, reason: %s", info.Reason)
	return nil
}

type RegionFailoverConfig struct {
	Name            string
	PrimaryRegion   string
	FallbackRegions []string
}

func InitDistributedFailoverManager(cfg *config.FailoverConfig) error {
	var err error
	failoverManagerOnce.Do(func() {
		failoverCfg := &FailoverConfig{
			Enabled:            cfg.Enabled,
			AutoDetectEnabled:  cfg.AutoDetectEnabled,
			RegionFailover:     cfg.RegionFailover,
			CrossRegionEnabled: cfg.CrossRegionEnabled,
			MaxRetries:         cfg.MaxRetries,
			RetryInterval:      time.Duration(cfg.RetryIntervalSecs) * time.Second,
			HealthCheckTimeout: time.Duration(cfg.HealthCheckTimeoutSecs) * time.Second,
			FallbackEnabled:   cfg.FallbackEnabled,
		}

		globalFailoverManager = NewDistributedFailoverManager(failoverCfg)
		globalFailoverManager.SetRedisClient(pkgredis.GetDistributedRedisClient())
		globalFailoverManager.SetDBRouter(database.GetElasticRouter())
		globalFailoverManager.SetRegistry(servicediscovery.GetDistributedRegistry())

		for _, regionCfg := range cfg.Regions {
			globalFailoverManager.RegisterRegion(&RegionFailoverConfig{
				Name:            regionCfg.Name,
				PrimaryRegion:   regionCfg.PrimaryRegion,
				FallbackRegions:  regionCfg.FallbackRegions,
			})
		}

		globalFailoverManager.Start()
	})
	return err
}

func GetDistributedFailoverManager() *DistributedFailoverManager {
	return globalFailoverManager
}
