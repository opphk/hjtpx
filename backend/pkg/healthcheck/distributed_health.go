package healthcheck

import (
	"context"
	"fmt"
	"log"
	"net"
	"net/http"
	"runtime"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/database"
	pkgredis "github.com/hjtpx/hjtpx/pkg/redis"
)

type DistributedHealthChecker struct {
	mu            sync.RWMutex
	config        *HealthCheckConfig
	checks        map[string]HealthCheck
	results       map[string]*HealthCheckResult
	status        ServiceStatus
	interval      time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
	notifiers     []HealthStatusNotifier
	failoverMgr   *FailoverManager
	metrics       *HealthCheckMetrics
}

type HealthCheckConfig struct {
	Enabled            bool
	Interval           time.Duration
	Timeout            time.Duration
	MaxFailCount       int
	FailureThreshold   int
	RecoveryThreshold  int
	EnableAutoFailover bool
	CheckPostgres      bool
	CheckRedis         bool
	CheckMemcached     bool
	CheckExternal      bool
	ExternalEndpoints []string
}

type HealthCheck interface {
	Check(ctx context.Context) error
	Name() string
}

type HealthCheckResult struct {
	Name            string
	Status          CheckStatus
	Message         string
	Latency         time.Duration
	LastCheck       time.Time
	FailCount       int
	SuccessCount    int
	ConsecutiveFails int
	ConsecutiveSuccesses int
}

type CheckStatus string

const (
	StatusHealthy   CheckStatus = "healthy"
	StatusDegraded   CheckStatus = "degraded"
	StatusUnhealthy CheckStatus = "unhealthy"
	StatusUnknown   CheckStatus = "unknown"
)

type ServiceStatus string

const (
	ServiceStatusHealthy   ServiceStatus = "healthy"
	ServiceStatusDegraded  ServiceStatus = "degraded"
	ServiceStatusUnhealthy ServiceStatus = "unhealthy"
)

type HealthStatusNotifier interface {
	Notify(status ServiceStatus, results map[string]*HealthCheckResult)
}

type HealthCheckMetrics struct {
	TotalChecks      atomic.Int64
	HealthyChecks    atomic.Int64
	UnhealthyChecks  atomic.Int64
	DegradedChecks   atomic.Int64
	AvgLatency       atomic.Int64
	FailoverTriggers atomic.Int64
	RecoveryTriggers atomic.Int64
}

type FailoverManager struct {
	mu              sync.RWMutex
	enabled         bool
	failoverEnabled bool
	strategies      map[string]FailoverStrategy
	history         []*FailoverEvent
	maxHistory      int
	metrics         *FailoverMetrics
}

type FailoverStrategy interface {
	Execute(ctx context.Context, failure FailureInfo) error
	Name() string
}

type FailureInfo struct {
	CheckName     string
	FailureType   FailureType
	Message       string
	Timestamp     time.Time
	InstanceID    string
	Region        string
	Severity      FailureSeverity
}

type FailoverInfo struct {
	ID            string
	Type          FailoverType
	Source        string
	Target        string
	Region        string
	DC            string
	Reason        string
	Severity      FailureSeverity
	Timestamp     time.Time
	RetryCount    int
	OriginalError error
	Metadata      map[string]string
}

type FailureType string

type FailoverType string

const (
	FailureTypeTimeout     FailureType = "timeout"
	FailureTypeConnection  FailureType = "connection"
	FailureTypeAuth        FailureType = "auth"
	FailureTypeResource    FailureType = "resource"
	FailureTypeNetwork     FailureType = "network"
	FailureTypeUnknown     FailureType = "unknown"
)

const (
	FailoverTypeNode       FailoverType = "node"
	FailoverTypeRegion     FailoverType = "region"
	FailoverTypeDatacenter FailoverType = "datacenter"
	FailoverTypeNetwork    FailoverType = "network"
)

type FailureSeverity string

const (
	SeverityLow      FailureSeverity = "low"
	SeverityMedium   FailureSeverity = "medium"
	SeverityHigh     FailureSeverity = "high"
	SeverityCritical FailureSeverity = "critical"
)

type FailoverEvent struct {
	ID          string
	Timestamp   time.Time
	CheckName   string
	FailureInfo FailureInfo
	Action      string
	Result      FailoverResult
	Duration    time.Duration
}

type FailoverResult string

const (
	FailoverResultSuccess   FailoverResult = "success"
	FailoverResultFailed     FailoverResult = "failed"
	FailoverResultSkipped    FailoverResult = "skipped"
	FailoverResultPartial   FailoverResult = "partial"
)

type FailoverMetrics struct {
	TotalFailovers     atomic.Int64
	SuccessfulFailovers atomic.Int64
	FailedFailovers    atomic.Int64
	LastFailoverTime   atomic.Value
	LastFailoverTarget string
}

var (
	globalHealthChecker *DistributedHealthChecker
	healthCheckerOnce   sync.Once
)

func NewDistributedHealthChecker(cfg *HealthCheckConfig) *DistributedHealthChecker {
	if cfg == nil {
		cfg = getDefaultHealthCheckConfig()
	}

	hc := &DistributedHealthChecker{
		config:      cfg,
		checks:      make(map[string]HealthCheck),
		results:     make(map[string]*HealthCheckResult),
		interval:    cfg.Interval,
		stopCh:      make(chan struct{}),
		metrics:     &HealthCheckMetrics{},
		failoverMgr: newFailoverManager(cfg.EnableAutoFailover),
	}

	hc.registerDefaultChecks()

	return hc
}

func getDefaultHealthCheckConfig() *HealthCheckConfig {
	return &HealthCheckConfig{
		Enabled:            true,
		Interval:           10 * time.Second,
		Timeout:            5 * time.Second,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: true,
		CheckPostgres:      true,
		CheckRedis:         true,
	}
}

func (hc *DistributedHealthChecker) registerDefaultChecks() {
	hc.RegisterCheck(&PostgresHealthCheck{})
	hc.RegisterCheck(&RedisHealthCheck{})
	hc.RegisterCheck(&MemoryHealthCheck{})
	hc.RegisterCheck(&GoroutineHealthCheck{})
	hc.RegisterCheck(&TCPHealthCheck{CheckName: "tcp_local", Address: "localhost:8080"})
}

func (hc *DistributedHealthChecker) RegisterCheck(check HealthCheck) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.checks[check.Name()] = check
	hc.results[check.Name()] = &HealthCheckResult{
		Name:      check.Name(),
		Status:    StatusUnknown,
		LastCheck: time.Time{},
	}
}

func (hc *DistributedHealthChecker) UnregisterCheck(name string) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	delete(hc.checks, name)
	delete(hc.results, name)
}

func (hc *DistributedHealthChecker) Start() {
	if !hc.config.Enabled {
		log.Println("[HEALTH_CHECK] Health checker is disabled")
		return
	}

	hc.wg.Add(1)
	go hc.checkLoop()

	hc.wg.Add(1)
	go hc.statusEvaluator()

	log.Printf("[HEALTH_CHECK] Started with %d checks, interval: %v", len(hc.checks), hc.interval)
}

func (hc *DistributedHealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
	log.Println("[HEALTH_CHECK] Stopped")
}

func (hc *DistributedHealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	hc.runAllChecks()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.runAllChecks()
		}
	}
}

func (hc *DistributedHealthChecker) runAllChecks() {
	hc.mu.RLock()
	checks := make(map[string]HealthCheck)
	for k, v := range hc.checks {
		checks[k] = v
	}
	results := hc.results
	hc.mu.RUnlock()

	for name, check := range checks {
		go hc.runCheck(name, check, results)
	}
}

func (hc *DistributedHealthChecker) runCheck(name string, check HealthCheck, results map[string]*HealthCheckResult) {
	ctx, cancel := context.WithTimeout(context.Background(), hc.config.Timeout)
	defer cancel()

	start := time.Now()
	err := check.Check(ctx)
	latency := time.Since(start)

	result := &HealthCheckResult{
		Name:      name,
		LastCheck: time.Now(),
		Latency:   latency,
	}

	if err != nil {
		result.Status = StatusUnhealthy
		result.Message = err.Error()
		result.ConsecutiveFails++
		result.ConsecutiveSuccesses = 0
		result.FailCount++

		hc.metrics.UnhealthyChecks.Add(1)

		if result.ConsecutiveFails >= hc.config.FailureThreshold {
			if hc.failoverMgr.enabled {
				hc.triggerFailover(name, err)
			}
		}
	} else {
		result.Status = StatusHealthy
		result.Message = "ok"
		result.ConsecutiveSuccesses++
		result.ConsecutiveFails = 0
		result.SuccessCount++

		hc.metrics.HealthyChecks.Add(1)
	}

	hc.mu.Lock()
	results[name] = result
	hc.mu.Unlock()

	hc.metrics.TotalChecks.Add(1)

	totalLatency := hc.metrics.AvgLatency.Load()
	count := hc.metrics.TotalChecks.Load()
	if count > 0 {
		hc.metrics.AvgLatency.Store((totalLatency*(count-1) + latency.Nanoseconds()) / count)
	}
}

func (hc *DistributedHealthChecker) statusEvaluator() {
	defer hc.wg.Done()

	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.evaluateStatus()
		}
	}
}

func (hc *DistributedHealthChecker) evaluateStatus() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	healthyCount := 0
	unhealthyCount := 0
	degradedCount := 0

	for _, result := range hc.results {
		switch result.Status {
		case StatusHealthy:
			healthyCount++
		case StatusUnhealthy:
			unhealthyCount++
		case StatusDegraded:
			degradedCount++
		}
	}

	totalChecks := len(hc.results)
	if totalChecks == 0 {
		return
	}

	var newStatus ServiceStatus
	if unhealthyCount > 0 || float64(unhealthyCount)/float64(totalChecks) > 0.5 {
		newStatus = ServiceStatusUnhealthy
	} else if degradedCount > 0 || healthyCount < totalChecks {
		newStatus = ServiceStatusDegraded
	} else {
		newStatus = ServiceStatusHealthy
	}

	if newStatus != hc.status {
		oldStatus := hc.status
		hc.status = newStatus
		log.Printf("[HEALTH_CHECK] Status changed: %s -> %s (healthy=%d, degraded=%d, unhealthy=%d)",
			oldStatus, newStatus, healthyCount, degradedCount, unhealthyCount)
		hc.notifyStatusChange(newStatus)
	}
}

func (hc *DistributedHealthChecker) notifyStatusChange(status ServiceStatus) {
	hc.mu.RLock()
	results := make(map[string]*HealthCheckResult)
	for k, v := range hc.results {
		results[k] = v
	}
	notifiers := hc.notifiers
	hc.mu.RUnlock()

	for _, notifier := range notifiers {
		go notifier.Notify(status, results)
	}
}

func (hc *DistributedHealthChecker) AddNotifier(notifier HealthStatusNotifier) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.notifiers = append(hc.notifiers, notifier)
}

func (hc *DistributedHealthChecker) GetStatus() ServiceStatus {
	hc.mu.RLock()
	defer hc.mu.RUnlock()
	return hc.status
}

func (hc *DistributedHealthChecker) GetResults() map[string]*HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	results := make(map[string]*HealthCheckResult)
	for k, v := range hc.results {
		results[k] = v
	}
	return results
}

func (hc *DistributedHealthChecker) GetMetrics() *HealthCheckMetrics {
	return hc.metrics
}

func (hc *DistributedHealthChecker) triggerFailover(checkName string, err error) {
	failureInfo := FailureInfo{
		CheckName:   checkName,
		Message:     err.Error(),
		Timestamp:   time.Now(),
		Severity:    SeverityHigh,
		FailureType: FailureTypeConnection,
	}

	log.Printf("[FAILOVER] Triggering failover for check: %s, error: %v", checkName, err)
	hc.failoverMgr.ExecuteFailover(context.Background(), failureInfo)
	hc.metrics.FailoverTriggers.Add(1)
}

type PostgresHealthCheck struct{}

func (c *PostgresHealthCheck) Name() string {
	return "postgres"
}

func (c *PostgresHealthCheck) Check(ctx context.Context) error {
	db := database.DB
	if db == nil {
		return fmt.Errorf("database not initialized")
	}

	sqlDB, err := db.DB()
	if err != nil {
		return fmt.Errorf("failed to get sql.DB: %w", err)
	}

	return sqlDB.PingContext(ctx)
}

type RedisHealthCheck struct{}

func (c *RedisHealthCheck) Name() string {
	return "redis"
}

func (c *RedisHealthCheck) Check(ctx context.Context) error {
	client := pkgredis.GetClient()
	if client == nil {
		distributedClient := pkgredis.GetDistributedRedisClient()
		if distributedClient != nil {
			return distributedClient.Ping(ctx)
		}
		return fmt.Errorf("redis client not initialized")
	}

	return client.Ping(ctx).Err()
}

type MemoryHealthCheck struct{}

func (c *MemoryHealthCheck) Name() string {
	return "memory"
}

func (c *MemoryHealthCheck) Check(ctx context.Context) error {
	var m runtime.MemStats
	runtime.ReadMemStats(&m)

	if m.Alloc > 1<<30 {
		return fmt.Errorf("memory usage too high: %d bytes", m.Alloc)
	}

	return nil
}

type GoroutineHealthCheck struct{}

func (c *GoroutineHealthCheck) Name() string {
	return "goroutines"
}

func (c *GoroutineHealthCheck) Check(ctx context.Context) error {
	numGoroutines := runtime.NumGoroutine()
	if numGoroutines > 10000 {
		return fmt.Errorf("too many goroutines: %d", numGoroutines)
	}
	return nil
}

type TCPHealthCheck struct {
	CheckName string
	Address   string
}

func (c *TCPHealthCheck) Name() string {
	return c.CheckName
}

func (c *TCPHealthCheck) Check(ctx context.Context) error {
	dialer := &net.Dialer{Timeout: 3 * time.Second}
	conn, err := dialer.DialContext(ctx, "tcp", c.Address)
	if err != nil {
		return fmt.Errorf("failed to connect to %s: %w", c.Address, err)
	}
	defer conn.Close()
	return nil
}

type HTTPHealthCheck struct {
	CheckName string
	URL      string
	Method   string
	Headers  map[string]string
}

func NewHTTPHealthCheck(name, url string) *HTTPHealthCheck {
	return &HTTPHealthCheck{
		CheckName: name,
		URL:       url,
		Method:    "GET",
		Headers:   make(map[string]string),
	}
}

func (c *HTTPHealthCheck) Name() string {
	return c.CheckName
}

func (c *HTTPHealthCheck) Check(ctx context.Context) error {
	req, err := http.NewRequestWithContext(ctx, c.Method, c.URL, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	for k, v := range c.Headers {
		req.Header.Set(k, v)
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return fmt.Errorf("request failed: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode >= 500 {
		return fmt.Errorf("server error: %d", resp.StatusCode)
	}

	return nil
}

func newFailoverManager(enabled bool) *FailoverManager {
	return &FailoverManager{
		enabled:         enabled,
		failoverEnabled: enabled,
		strategies:      make(map[string]FailoverStrategy),
		history:         make([]*FailoverEvent, 0),
		maxHistory:      100,
		metrics:         &FailoverMetrics{},
	}
}

func (fm *FailoverManager) RegisterStrategy(strategy FailoverStrategy) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.strategies[strategy.Name()] = strategy
}

func (fm *FailoverManager) ExecuteFailover(ctx context.Context, info FailureInfo) error {
	if !fm.failoverEnabled {
		return fmt.Errorf("failover is disabled")
	}

	fm.mu.Lock()
	fm.metrics.TotalFailovers.Add(1)
	fm.metrics.LastFailoverTime.Store(time.Now())
	fm.mu.Unlock()

	var lastErr error
	for _, strategy := range fm.strategies {
		if err := strategy.Execute(ctx, info); err != nil {
			lastErr = err
			fm.metrics.FailedFailovers.Add(1)
			log.Printf("[FAILOVER] Strategy %s failed: %v", strategy.Name(), err)
		} else {
			fm.metrics.SuccessfulFailovers.Add(1)
		}
	}

	event := &FailoverEvent{
		ID:          fmt.Sprintf("failover-%d", time.Now().UnixNano()),
		Timestamp:   time.Now(),
		CheckName:   info.CheckName,
		FailureInfo: info,
		Action:      "execute_strategies",
		Duration:    time.Since(info.Timestamp),
	}

	if lastErr == nil {
		event.Result = FailoverResultSuccess
	} else {
		event.Result = FailoverResultFailed
	}

	fm.addEvent(event)

	return lastErr
}

func (fm *FailoverManager) addEvent(event *FailoverEvent) {
	fm.mu.Lock()
	defer fm.mu.Unlock()

	fm.history = append(fm.history, event)
	if len(fm.history) > fm.maxHistory {
		fm.history = fm.history[len(fm.history)-fm.maxHistory:]
	}
}

func (fm *FailoverManager) GetHistory() []*FailoverEvent {
	fm.mu.RLock()
	defer fm.mu.RUnlock()

	history := make([]*FailoverEvent, len(fm.history))
	copy(history, fm.history)
	return history
}

func (fm *FailoverManager) GetMetrics() *FailoverMetrics {
	return fm.metrics
}

func (fm *FailoverManager) SetEnabled(enabled bool) {
	fm.mu.Lock()
	defer fm.mu.Unlock()
	fm.failoverEnabled = enabled
}

type CircuitBreakerStrategy struct {
	name          string
	redisStrategy *RedisFailoverStrategy
}

func NewCircuitBreakerStrategy() *CircuitBreakerStrategy {
	return &CircuitBreakerStrategy{
		name:          "circuit_breaker",
		redisStrategy: NewRedisFailoverStrategy(),
	}
}

func (s *CircuitBreakerStrategy) Name() string {
	return s.name
}

func (s *CircuitBreakerStrategy) CanHandle(info *FailoverInfo) bool {
	return info.Type == FailoverTypeNode
}

func (s *CircuitBreakerStrategy) Execute(ctx context.Context, info FailureInfo) error {
	log.Printf("[CIRCUIT_BREAKER] Handling failure for %s: %v", info.CheckName, info.Message)
	if s.redisStrategy != nil {
		return s.redisStrategy.Execute(ctx, info)
	}
	return nil
}

type RedisFailoverStrategy struct{}

func NewRedisFailoverStrategy() *RedisFailoverStrategy {
	return &RedisFailoverStrategy{}
}

func (s *RedisFailoverStrategy) Name() string {
	return "redis_failover"
}

func (s *RedisFailoverStrategy) Execute(ctx context.Context, info FailureInfo) error {
	distributedRedis := pkgredis.GetDistributedRedisClient()
	if distributedRedis == nil {
		return fmt.Errorf("distributed redis client not available")
	}

	key := fmt.Sprintf("failover:circuit:%s", info.CheckName)
	if err := distributedRedis.Set(ctx, key, "open", 30*time.Second); err != nil {
		return fmt.Errorf("failed to set circuit breaker state: %w", err)
	}

	log.Printf("[REDIS_FAILOVER] Circuit breaker opened for %s", info.CheckName)
	return nil
}

type DatabaseFailoverStrategy struct{}

func NewDatabaseFailoverStrategy() *DatabaseFailoverStrategy {
	return &DatabaseFailoverStrategy{}
}

func (s *DatabaseFailoverStrategy) Name() string {
	return "database_failover"
}

func (s *DatabaseFailoverStrategy) Execute(ctx context.Context, info FailureInfo) error {
	elasticRouter := database.GetElasticRouter()
	if elasticRouter == nil {
		return nil
	}

	log.Printf("[DB_FAILOVER] Evaluating database failover for %s", info.CheckName)

	stats := elasticRouter.GetReplicaStats()
	for _, stat := range stats {
		if !stat.Healthy {
			log.Printf("[DB_FAILOVER] Slave %s:%s is unhealthy", stat.Host, stat.Port)
		}
	}

	return nil
}

func InitDistributedHealthChecker(cfg *config.HealthCheckConfig) error {
	var err error
	healthCheckerOnce.Do(func() {
		healthCfg := &HealthCheckConfig{
			Enabled:            cfg.Enabled,
			Interval:           time.Duration(cfg.IntervalSecs) * time.Second,
			Timeout:            time.Duration(cfg.TimeoutSecs) * time.Second,
			MaxFailCount:       cfg.MaxFailCount,
			FailureThreshold:   cfg.FailureThreshold,
			RecoveryThreshold:  cfg.RecoveryThreshold,
			EnableAutoFailover: cfg.EnableAutoFailover,
			CheckPostgres:      cfg.CheckPostgres,
			CheckRedis:         cfg.CheckRedis,
		}

		globalHealthChecker = NewDistributedHealthChecker(healthCfg)
		globalHealthChecker.Start()

		failoverMgr := globalHealthChecker.failoverMgr
		failoverMgr.RegisterStrategy(NewCircuitBreakerStrategy())
		failoverMgr.RegisterStrategy(NewRedisFailoverStrategy())
		failoverMgr.RegisterStrategy(NewDatabaseFailoverStrategy())
	})
	return err
}

func GetDistributedHealthChecker() *DistributedHealthChecker {
	return globalHealthChecker
}
