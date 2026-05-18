package database

import (
	"context"
	"fmt"
	"log"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type ElasticDBRouter struct {
	mu                 sync.RWMutex
	masterDB           *gorm.DB
	slaveDBs           []*gorm.DB
	replicaStats       []*ReplicaStats
	slaveWeights       []int
	currentSlave       uint32
	enabled            bool
	loadBalanceMode    string
	autoScalingEnabled bool
	scaleConfig        *ScaleConfig
	healthChecker      *ElasticHealthChecker
	metrics            *ElasticRouterMetrics
	regionRouter       *RegionRouter
	scaleController    *ScaleController
}

type ReplicaStats struct {
	Index           int
	Host            string
	Port            string
	Healthy         bool
	Latency         time.Duration
	LastCheck       time.Time
	FailCount       int
	QueryCount      atomic.Int64
	BytesReceived   atomic.Int64
	ReplicationLag  time.Duration
	MaxReplicationLag time.Duration
	CurrentState    ReplicaState
}

type ReplicaState string

const (
	ReplicaStateHealthy   ReplicaState = "healthy"
	ReplicaStateDegraded  ReplicaState = "degraded"
	ReplicaStateOffline   ReplicaState = "offline"
	ReplicaStateCatchingUp ReplicaState = "catching_up"
)

type ScaleConfig struct {
	MinReplicas       int
	MaxReplicas       int
	ScaleUpThreshold  float64
	ScaleDownThreshold float64
	ScaleUpCooldown   time.Duration
	ScaleDownCooldown time.Duration
	TargetLatency     time.Duration
	HighCPUThreshold  float64
	LowCPUThreshold   float64
}

type ElasticRouterMetrics struct {
	MasterQueries     atomic.Int64
	SlaveQueries      atomic.Int64
	FailedQueries     atomic.Int64
	SlaveSwitches     atomic.Int64
	LastSwitchTime    atomic.Value
	AvgLatencyMs      atomic.Int64
	MaxLatencyMs      atomic.Int64
	MinLatencyMs      atomic.Int64
	TotalBytesSent    atomic.Int64
	ScaleEvents       atomic.Int64
	FailoverCount     atomic.Int64
	HealthySlaves     atomic.Int64
	AvgReplicationLag atomic.Int64
}

type ElasticHealthChecker struct {
	router      *ElasticDBRouter
	interval    time.Duration
	enabled     bool
	stopCh      chan struct{}
	wg          sync.WaitGroup
	maxFailCount int
	mu          sync.RWMutex
}

type RegionRouter struct {
	mu           sync.RWMutex
	regions      map[string]*RegionConfig
	currentRegion string
	fallbackEnabled bool
}

type RegionConfig struct {
	Name            string
	MasterHost      string
	MasterPort      string
	SlaveHosts      []string
	LatencyWeight   float64
	Healthy         bool
}

type ScaleController struct {
	router         *ElasticDBRouter
	enabled        bool
	scaleUpTimer   *time.Timer
	scaleDownTimer *time.Timer
	mu             sync.Mutex
	scaleUpCooldown time.Duration
	scaleDownCooldown time.Duration
}

var (
	elasticRouter *ElasticDBRouter
	elasticRouterOnce sync.Once
)

func InitElasticDBRouter(cfg *config.Config) error {
	var err error
	elasticRouterOnce.Do(func() {
		elasticRouter, err = newElasticDBRouter(cfg)
	})
	return err
}

func newElasticDBRouter(cfg *config.Config) (*ElasticDBRouter, error) {
	router := &ElasticDBRouter{
		enabled:         cfg.Database.ReadWriteSeparation.Enabled,
		loadBalanceMode: cfg.Database.ReadWriteSeparation.LoadBalanceStrategy,
		metrics:         &ElasticRouterMetrics{},
		scaleConfig: &ScaleConfig{
			MinReplicas:        2,
			MaxReplicas:        10,
			ScaleUpThreshold:   0.8,
			ScaleDownThreshold: 0.3,
			ScaleUpCooldown:    5 * time.Minute,
			ScaleDownCooldown:  15 * time.Minute,
			TargetLatency:      50 * time.Millisecond,
			HighCPUThreshold:   80,
			LowCPUThreshold:    20,
		},
		autoScalingEnabled: cfg.Database.ConnectionPool.EnableAutoTuning,
	}

	if !router.enabled {
		log.Println("[ELASTIC_DB] Read-write separation disabled, using single DB")
		return router, nil
	}

	if err := router.connectMastersAndSlaves(cfg); err != nil {
		return nil, err
	}

	router.healthChecker = newElasticHealthChecker(router, 15*time.Second)
	router.healthChecker.Start()

	if cfg.Database.ReadWriteSeparation.AutoFailover {
		router.scaleController = newScaleController(router, router.scaleConfig)
		router.scaleController.Start()
	}

	router.regionRouter = &RegionRouter{
		regions:         make(map[string]*RegionConfig),
		currentRegion:   "default",
		fallbackEnabled: true,
	}

	log.Printf("[ELASTIC_DB] Initialized with %d slaves, auto-scaling: %v",
		len(router.slaveDBs), router.autoScalingEnabled)

	return router, nil
}

func (r *ElasticDBRouter) connectMastersAndSlaves(cfg *config.Config) error {
	masterDB, err := connectDBWithRetry(
		cfg.Database.ReadWriteSeparation.Master.Host,
		cfg.Database.ReadWriteSeparation.Master.Port,
		cfg.Database.ReadWriteSeparation.Master.User,
		cfg.Database.ReadWriteSeparation.Master.Password,
		cfg.Database.ReadWriteSeparation.Master.DBName,
		cfg.Database.ReadWriteSeparation.Master.SSLMode,
		3,
	)
	if err != nil {
		return fmt.Errorf("failed to connect master database: %w", err)
	}
	r.masterDB = masterDB

	r.slaveDBs = make([]*gorm.DB, 0, len(cfg.Database.ReadWriteSeparation.Slaves))
	r.replicaStats = make([]*ReplicaStats, 0, len(cfg.Database.ReadWriteSeparation.Slaves))
	r.slaveWeights = make([]int, 0, len(cfg.Database.ReadWriteSeparation.Slaves))

	for i, slaveCfg := range cfg.Database.ReadWriteSeparation.Slaves {
		slaveDB, err := connectDBWithRetry(
			slaveCfg.Host,
			slaveCfg.Port,
			slaveCfg.User,
			slaveCfg.Password,
			slaveCfg.DBName,
			slaveCfg.SSLMode,
			3,
		)
		if err != nil {
			log.Printf("[ELASTIC_DB] Warning: failed to connect slave %s: %v", slaveCfg.Host, err)
			continue
		}

		r.slaveDBs = append(r.slaveDBs, slaveDB)
		r.slaveWeights = append(r.slaveWeights, slaveCfg.Weight)

		stats := &ReplicaStats{
			Index:            i,
			Host:             slaveCfg.Host,
			Port:             slaveCfg.Port,
			Healthy:          true,
			LastCheck:        time.Now(),
			MaxReplicationLag: 100 * time.Millisecond,
			CurrentState:    ReplicaStateHealthy,
		}
		r.replicaStats = append(r.replicaStats, stats)
	}

	return nil
}

func connectDBWithRetry(host, port, user, password, dbname, sslmode string, maxRetries int) (*gorm.DB, error) {
	var lastErr error
	for i := 0; i < maxRetries; i++ {
		db, err := connectDB(host, port, user, password, dbname, sslmode)
		if err == nil {
			return db, nil
		}
		lastErr = err
		backoff := time.Duration(math.Pow(2, float64(i))) * time.Second
		log.Printf("[DB_CONNECT] Retry %d/%d for %s after %v: %v", i+1, maxRetries, host, backoff, err)
		time.Sleep(backoff)
	}
	return nil, fmt.Errorf("failed to connect after %d retries: %w", maxRetries, lastErr)
}

func newElasticHealthChecker(router *ElasticDBRouter, interval time.Duration) *ElasticHealthChecker {
	return &ElasticHealthChecker{
		router:        router,
		interval:      interval,
		enabled:       true,
		stopCh:        make(chan struct{}),
		maxFailCount:   3,
	}
}

func (hc *ElasticHealthChecker) Start() {
	hc.wg.Add(1)
	go hc.checkLoop()
}

func (hc *ElasticHealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *ElasticHealthChecker) SetMaxFailCount(count int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.maxFailCount = count
}

func (hc *ElasticHealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.checkAllSlaves()
			hc.evaluateHealth()
		}
	}
}

func (hc *ElasticHealthChecker) checkAllSlaves() {
	hc.router.mu.RLock()
	defer hc.router.mu.RUnlock()

	for i, slave := range hc.router.slaveDBs {
		if i >= len(hc.router.replicaStats) {
			continue
		}

		stats := hc.router.replicaStats[i]
		stats.LastCheck = time.Now()

		sqlDB, err := slave.DB()
		if err != nil {
			stats.Healthy = false
			stats.FailCount++
			stats.CurrentState = ReplicaStateOffline
			continue
		}

		start := time.Now()
		if err := sqlDB.Ping(); err != nil {
			stats.Healthy = false
			stats.FailCount++
			stats.CurrentState = ReplicaStateOffline
			stats.Latency = time.Since(start)
			continue
		}

		stats.Latency = time.Since(start)
		stats.FailCount = 0

		if stats.ReplicationLag > stats.MaxReplicationLag {
			stats.CurrentState = ReplicaStateDegraded
		} else {
			stats.CurrentState = ReplicaStateHealthy
			stats.Healthy = true
		}
	}
}

func (hc *ElasticHealthChecker) evaluateHealth() {
	hc.router.mu.Lock()
	defer hc.router.mu.Unlock()

	healthyCount := 0
	var totalLag int64

	for _, stats := range hc.router.replicaStats {
		if stats.Healthy && stats.CurrentState == ReplicaStateHealthy {
			healthyCount++
			totalLag += int64(stats.ReplicationLag)
		}

		if stats.FailCount >= hc.maxFailCount && stats.Healthy {
			log.Printf("[ELASTIC_DB] Slave %s failed health check %d times, marking unhealthy",
				stats.Host, stats.FailCount)
			stats.Healthy = false
			stats.CurrentState = ReplicaStateOffline
			hc.router.metrics.FailoverCount.Add(1)

			if hc.router.scaleController != nil {
				hc.router.scaleController.NotifyReplicaFailure(stats.Index)
			}
		}
	}

	if len(hc.router.replicaStats) > 0 {
		avgLag := time.Duration(totalLag / int64(len(hc.router.replicaStats)))
		hc.router.metrics.AvgReplicationLag.Store(int64(avgLag))
	}

	hc.router.metrics.HealthySlaves.Store(int64(healthyCount))
}

func newScaleController(router *ElasticDBRouter, config *ScaleConfig) *ScaleController {
	return &ScaleController{
		router:           router,
		enabled:          true,
		scaleUpCooldown:  config.ScaleUpCooldown,
		scaleDownCooldown: config.ScaleDownCooldown,
	}
}

func (sc *ScaleController) Start() {
	log.Println("[SCALE_CONTROLLER] Started")
}

func (sc *ScaleController) NotifyReplicaFailure(index int) {
	log.Printf("[SCALE_CONTROLLER] Replica %d failed, checking if scale-up needed", index)
}

func (sc *ScaleController) CheckScalingNeeded() (scaleUp bool, scaleDown bool) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	router := sc.router
	router.mu.RLock()
	defer router.mu.RUnlock()

	healthySlaves := int64(0)
	var totalLatency int64

	for _, stats := range router.replicaStats {
		if stats.Healthy {
			healthySlaves++
			totalLatency += int64(stats.Latency)
		}
	}

	totalSlaves := len(router.replicaStats)
	if totalSlaves == 0 {
		return false, false
	}

	avgLatency := time.Duration(totalLatency / int64(totalSlaves))
	loadFactor := float64(healthySlaves) / float64(totalSlaves)

	if loadFactor < float64(router.scaleConfig.ScaleUpThreshold) && totalSlaves < router.scaleConfig.MaxReplicas {
		log.Printf("[SCALE_CONTROLLER] Scale up triggered: load=%.2f, healthy=%d, latency=%v",
			loadFactor, healthySlaves, avgLatency)
		return true, false
	}

	if loadFactor > float64(router.scaleConfig.ScaleDownThreshold) && totalSlaves > router.scaleConfig.MinReplicas && avgLatency < router.scaleConfig.TargetLatency {
		log.Printf("[SCALE_CONTROLLER] Scale down triggered: load=%.2f, healthy=%d, latency=%v",
			loadFactor, healthySlaves, avgLatency)
		return false, true
	}

	return false, false
}

func (r *ElasticDBRouter) Master() *gorm.DB {
	if !r.enabled || r.masterDB == nil {
		return DB
	}
	return r.masterDB
}

func (r *ElasticDBRouter) Slave() *gorm.DB {
	if !r.enabled || len(r.slaveDBs) == 0 {
		return r.masterDB
	}

	r.mu.RLock()
	defer r.mu.RUnlock()

	healthySlaves := r.getHealthySlaves()
	if len(healthySlaves) == 0 {
		return r.masterDB
	}

	switch r.loadBalanceMode {
	case "least_latency":
		return r.getSlaveLeastLatency(healthySlaves)
	case "weighted_round_robin":
		return r.getSlaveWeightedRoundRobin(healthySlaves)
	case "round_robin":
		return r.getSlaveRoundRobin(healthySlaves)
	case "random":
		return r.getSlaveRandom(healthySlaves)
	default:
		return r.getSlaveLeastLatency(healthySlaves)
	}
}

func (r *ElasticDBRouter) getHealthySlaves() []*ReplicaStats {
	var healthy []*ReplicaStats
	for _, stats := range r.replicaStats {
		if stats.Healthy && stats.CurrentState != ReplicaStateOffline {
			healthy = append(healthy, stats)
		}
	}
	return healthy
}

func (r *ElasticDBRouter) getSlaveLeastLatency(healthy []*ReplicaStats) *gorm.DB {
	if len(healthy) == 0 {
		return r.masterDB
	}

	var best *ReplicaStats
	minLatency := time.Hour

	for _, stats := range healthy {
		if stats.Latency < minLatency {
			minLatency = stats.Latency
			best = stats
		}
	}

	if best != nil {
		r.metrics.SlaveSwitches.Add(1)
		return r.slaveDBs[best.Index]
	}

	return r.slaveDBs[0]
}

func (r *ElasticDBRouter) getSlaveRoundRobin(healthy []*ReplicaStats) *gorm.DB {
	if len(healthy) == 0 {
		return r.masterDB
	}

	index := atomic.AddUint32(&r.currentSlave, 1) % uint32(len(r.slaveDBs))
	return r.slaveDBs[index]
}

func (r *ElasticDBRouter) getSlaveWeightedRoundRobin(healthy []*ReplicaStats) *gorm.DB {
	if len(healthy) == 0 {
		return r.masterDB
	}

	totalWeight := 0
	healthyWeights := make([]int, 0)
	healthyIndices := make([]int, 0)

	for _, stats := range healthy {
		if stats.Index < len(r.slaveWeights) {
			totalWeight += r.slaveWeights[stats.Index]
			healthyWeights = append(healthyWeights, r.slaveWeights[stats.Index])
			healthyIndices = append(healthyIndices, stats.Index)
		}
	}

	if totalWeight <= 0 {
		return r.slaveDBs[healthyIndices[0]]
	}

	randomVal := int(time.Now().UnixNano() % int64(totalWeight))
	currentWeight := 0

	for i, weight := range healthyWeights {
		currentWeight += weight
		if randomVal < currentWeight {
			return r.slaveDBs[healthyIndices[i]]
		}
	}

	return r.slaveDBs[healthyIndices[0]]
}

func (r *ElasticDBRouter) getSlaveRandom(healthy []*ReplicaStats) *gorm.DB {
	if len(healthy) == 0 {
		return r.masterDB
	}

	selected := healthy[time.Now().UnixNano()%int64(len(healthy))]
	return r.slaveDBs[selected.Index]
}

func (r *ElasticDBRouter) Read(ctx context.Context) *gorm.DB {
	r.metrics.SlaveQueries.Add(1)
	start := time.Now()
	defer func() {
		r.recordLatency(time.Since(start), false)
	}()
	return r.Slave().WithContext(ctx)
}

func (r *ElasticDBRouter) Write(ctx context.Context) *gorm.DB {
	r.metrics.MasterQueries.Add(1)
	start := time.Now()
	defer func() {
		r.recordLatency(time.Since(start), true)
	}()
	return r.Master().WithContext(ctx)
}

func (r *ElasticDBRouter) recordLatency(latency time.Duration, isMaster bool) {
	latencyMs := latency.Milliseconds()

	total := r.metrics.SlaveQueries.Load() + r.metrics.MasterQueries.Load()
	if total > 0 {
		avgLatency := (r.metrics.AvgLatencyMs.Load()*(total-1) + latencyMs) / total
		r.metrics.AvgLatencyMs.Store(avgLatency)
	}

	for {
		maxLatency := r.metrics.MaxLatencyMs.Load()
		if latencyMs <= maxLatency {
			break
		}
		if r.metrics.MaxLatencyMs.CompareAndSwap(maxLatency, latencyMs) {
			break
		}
	}

	for {
		minLatency := r.metrics.MinLatencyMs.Load()
		if latencyMs >= minLatency && minLatency > 0 {
			break
		}
		if r.metrics.MinLatencyMs.CompareAndSwap(minLatency, latencyMs) {
			break
		}
	}
}

func (r *ElasticDBRouter) RecordFailure() {
	r.metrics.FailedQueries.Add(1)
}

func (r *ElasticDBRouter) RecordSlaveSwitch() {
	r.metrics.SlaveSwitches.Add(1)
	r.metrics.LastSwitchTime.Store(time.Now())
}

func (r *ElasticDBRouter) GetMetrics() *ElasticRouterMetrics {
	return r.metrics
}

func (r *ElasticDBRouter) GetReplicaStats() []*ReplicaStats {
	r.mu.RLock()
	defer r.mu.RUnlock()

	stats := make([]*ReplicaStats, len(r.replicaStats))
	copy(stats, r.replicaStats)
	return stats
}

func (r *ElasticDBRouter) GetHealthStatus() []map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	status := make([]map[string]interface{}, 0, len(r.replicaStats))
	for _, stats := range r.replicaStats {
		status = append(status, map[string]interface{}{
			"index":            stats.Index,
			"host":             stats.Host,
			"port":             stats.Port,
			"healthy":          stats.Healthy,
			"latency_ms":       stats.Latency.Milliseconds(),
			"replication_lag":  stats.ReplicationLag.Milliseconds(),
			"state":            stats.CurrentState,
			"query_count":      stats.QueryCount.Load(),
			"fail_count":       stats.FailCount,
			"last_check":       stats.LastCheck.Format(time.RFC3339),
		})
	}
	return status
}

func (r *ElasticDBRouter) IsEnabled() bool {
	return r.enabled
}

func (r *ElasticDBRouter) Close() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.healthChecker != nil {
		r.healthChecker.Stop()
	}

	var err error

	if r.masterDB != nil {
		if sqlDB, err := r.masterDB.DB(); err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil {
				err = closeErr
			}
		}
	}

	for _, slave := range r.slaveDBs {
		if sqlDB, err := slave.DB(); err == nil {
			if closeErr := sqlDB.Close(); closeErr != nil && err == nil {
				err = closeErr
			}
		}
	}

	return err
}

func GetElasticRouter() *ElasticDBRouter {
	return elasticRouter
}

func (rr *RegionRouter) RegisterRegion(cfg *RegionConfig) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.regions[cfg.Name] = cfg
}

func (rr *RegionRouter) SetCurrentRegion(name string) {
	rr.mu.Lock()
	defer rr.mu.Unlock()
	rr.currentRegion = name
}

func (rr *RegionRouter) GetCurrentRegion() string {
	rr.mu.RLock()
	defer rr.mu.RUnlock()
	return rr.currentRegion
}

func (rr *RegionRouter) GetOptimalRegion() *RegionConfig {
	rr.mu.RLock()
	defer rr.mu.RUnlock()

	var best *RegionConfig
	minLatency := math.MaxFloat64

	for _, region := range rr.regions {
		if !region.Healthy {
			continue
		}

		latency := region.LatencyWeight
		if latency < minLatency {
			minLatency = latency
			best = region
		}
	}

	if best == nil && rr.fallbackEnabled {
		for _, region := range rr.regions {
			if region.Name == "default" {
				return region
			}
		}
	}

	return best
}
