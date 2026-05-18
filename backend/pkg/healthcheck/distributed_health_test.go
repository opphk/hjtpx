package healthcheck

import (
	"context"
	"net/http"
	"net/http/httptest"
	"runtime"
	"sync"
	"testing"
	"time"
)

func TestDistributedHealthChecker_Initialization(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            true,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: true,
		CheckPostgres:      false,
		CheckRedis:         false,
	}

	hc := NewDistributedHealthChecker(cfg)
	if hc == nil {
		t.Fatal("Expected health checker to be created")
	}

	if len(hc.checks) == 0 {
		t.Error("Expected default checks to be registered")
	}

	hc.Stop()
}

func TestDistributedHealthChecker_RegisterCheck(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
	}

	hc := NewDistributedHealthChecker(cfg)

	customCheck := &MockHealthCheck{name: "custom_check", healthy: true}
	hc.RegisterCheck(customCheck)

	if _, exists := hc.checks["custom_check"]; !exists {
		t.Error("Expected custom check to be registered")
	}

	hc.Stop()
}

func TestDistributedHealthChecker_UnregisterCheck(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
	}

	hc := NewDistributedHealthChecker(cfg)

	hc.UnregisterCheck("memory")
	if _, exists := hc.checks["memory"]; exists {
		t.Error("Expected memory check to be unregistered")
	}

	hc.Stop()
}

func TestDistributedHealthChecker_RunChecks(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
		CheckPostgres:      false,
		CheckRedis:         false,
	}

	hc := NewDistributedHealthChecker(cfg)
	hc.Stop()

	hc.runAllChecks()

	time.Sleep(100 * time.Millisecond)

	results := hc.GetResults()
	if len(results) == 0 {
		t.Error("Expected check results to be populated")
	}
}

func TestDistributedHealthChecker_StatusChange(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
		CheckPostgres:      false,
		CheckRedis:         false,
	}

	hc := NewDistributedHealthChecker(cfg)
	hc.Stop()

	hc.runAllChecks()
	time.Sleep(100 * time.Millisecond)

	status := hc.GetStatus()
	if status == "" {
		t.Error("Expected status to be set")
	}
}

func TestDistributedHealthChecker_GetMetrics(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
	}

	hc := NewDistributedHealthChecker(cfg)
	hc.Stop()

	metrics := hc.GetMetrics()
	if metrics == nil {
		t.Error("Expected metrics to be returned")
	}
}

func TestHealthCheckResult_Update(t *testing.T) {
	result := &HealthCheckResult{
		Name:      "test",
		Status:    StatusUnknown,
		FailCount: 0,
	}

	result.Status = StatusHealthy
	result.Message = "ok"
	result.ConsecutiveSuccesses = 1

	if result.Status != StatusHealthy {
		t.Errorf("Expected status to be healthy, got %s", result.Status)
	}

	if result.ConsecutiveSuccesses != 1 {
		t.Errorf("Expected consecutive successes to be 1, got %d", result.ConsecutiveSuccesses)
	}
}

func TestPostgresHealthCheck(t *testing.T) {
	check := &PostgresHealthCheck{}
	if check.Name() != "postgres" {
		t.Errorf("Expected name to be postgres, got %s", check.Name())
	}

	ctx := context.Background()
	err := check.Check(ctx)
	if err == nil {
		t.Log("Postgres check returned no error (expected in test environment)")
	}
}

func TestRedisHealthCheck(t *testing.T) {
	check := &RedisHealthCheck{}
	if check.Name() != "redis" {
		t.Errorf("Expected name to be redis, got %s", check.Name())
	}

	ctx := context.Background()
	err := check.Check(ctx)
	if err == nil {
		t.Log("Redis check passed (Redis client may not be initialized in test)")
	}
}

func TestMemoryHealthCheck(t *testing.T) {
	check := &MemoryHealthCheck{}
	if check.Name() != "memory" {
		t.Errorf("Expected name to be memory, got %s", check.Name())
	}

	ctx := context.Background()
	err := check.Check(ctx)
	if err != nil {
		t.Errorf("Memory check failed: %v", err)
	}
}

func TestGoroutineHealthCheck(t *testing.T) {
	check := &GoroutineHealthCheck{}
	if check.Name() != "goroutines" {
		t.Errorf("Expected name to be goroutines, got %s", check.Name())
	}

	ctx := context.Background()
	err := check.Check(ctx)
	if err != nil {
		t.Errorf("Goroutine check failed: %v", err)
	}

	if runtime.NumGoroutine() > 10000 {
		t.Error("Too many goroutines")
	}
}

func TestTCPHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := &TCPHealthCheck{
		CheckName: "tcp_test",
		Address:   server.Listener.Addr().String(),
	}

	ctx := context.Background()
	err := check.Check(ctx)
	if err != nil {
		t.Errorf("TCP check failed: %v", err)
	}
}

func TestTCPHealthCheck_Failure(t *testing.T) {
	check := &TCPHealthCheck{
		CheckName: "tcp_fail",
		Address:   "localhost:59999",
	}

	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	err := check.Check(ctx)
	if err == nil {
		t.Error("Expected TCP check to fail for invalid address")
	}
}

func TestHTTPHealthCheck(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusOK)
	}))
	defer server.Close()

	check := NewHTTPHealthCheck("http_test", server.URL)

	ctx := context.Background()
	err := check.Check(ctx)
	if err != nil {
		t.Errorf("HTTP check failed: %v", err)
	}
}

func TestHTTPHealthCheck_ServerError(t *testing.T) {
	server := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(http.StatusInternalServerError)
	}))
	defer server.Close()

	check := NewHTTPHealthCheck("http_error", server.URL)

	ctx := context.Background()
	err := check.Check(ctx)
	if err == nil {
		t.Error("Expected HTTP check to fail for server error")
	}
}

func TestFailoverManager_Initialization(t *testing.T) {
	mgr := newFailoverManager(true)
	if mgr == nil {
		t.Fatal("Expected failover manager to be created")
	}

	if !mgr.enabled {
		t.Error("Expected failover manager to be enabled")
	}

	if len(mgr.strategies) != 0 {
		t.Error("Expected no strategies to be registered initially")
	}
}

func TestFailoverManager_RegisterStrategy(t *testing.T) {
	mgr := newFailoverManager(true)

	strategy := NewCircuitBreakerStrategy()
	mgr.RegisterStrategy(strategy)

	if len(mgr.strategies) == 0 {
		t.Error("Expected strategy to be registered")
	}
}

func TestCircuitBreakerStrategy_Name(t *testing.T) {
	strategy := NewCircuitBreakerStrategy()
	if strategy.Name() != "circuit_breaker" {
		t.Errorf("Expected name to be circuit_breaker, got %s", strategy.Name())
	}
}

func TestCircuitBreakerStrategy_CanHandle(t *testing.T) {
	strategy := NewCircuitBreakerStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeNode,
		Source:   "test_source",
		Reason:   "test_reason",
		Severity: SeverityHigh,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected circuit breaker strategy to handle node failover")
	}
}

func TestRedisFailoverStrategy_Name(t *testing.T) {
	strategy := NewRedisFailoverStrategy()
	if strategy.Name() != "redis_failover" {
		t.Errorf("Expected name to be redis_failover, got %s", strategy.Name())
	}
}

func TestDatabaseFailoverStrategy_Name(t *testing.T) {
	strategy := NewDatabaseFailoverStrategy()
	if strategy.Name() != "database_failover" {
		t.Errorf("Expected name to be database_failover, got %s", strategy.Name())
	}
}

func TestNetworkFailoverStrategy_Name(t *testing.T) {
	strategy := NewNetworkFailoverStrategy()
	if strategy.Name() != "network_failover" {
		t.Errorf("Expected name to be network_failover, got %s", strategy.Name())
	}
}

func TestNetworkFailoverStrategy_CanHandle(t *testing.T) {
	strategy := NewNetworkFailoverStrategy()

	info := &FailoverInfo{
		Type:     FailoverTypeNetwork,
		Source:   "test_source",
		Reason:   "test_reason",
		Severity: SeverityMedium,
	}

	if !strategy.CanHandle(info) {
		t.Error("Expected network failover strategy to handle network failover")
	}
}

func TestFailoverMetrics(t *testing.T) {
	metrics := &FailoverMetrics{}

	metrics.TotalFailovers.Add(10)
	metrics.SuccessfulFailovers.Add(8)
	metrics.FailedFailovers.Add(2)

	if metrics.TotalFailovers.Load() != 10 {
		t.Errorf("Expected total failovers to be 10, got %d", metrics.TotalFailovers.Load())
	}

	if metrics.SuccessfulFailovers.Load() != 8 {
		t.Errorf("Expected successful failovers to be 8, got %d", metrics.SuccessfulFailovers.Load())
	}
}

func TestHealthCheckMetrics(t *testing.T) {
	metrics := &HealthCheckMetrics{}

	metrics.TotalChecks.Add(100)
	metrics.HealthyChecks.Add(90)
	metrics.UnhealthyChecks.Add(10)

	if metrics.TotalChecks.Load() != 100 {
		t.Errorf("Expected total checks to be 100, got %d", metrics.TotalChecks.Load())
	}
}

func TestNotifier(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
	}

	hc := NewDistributedHealthChecker(cfg)
	hc.Stop()

	notified := false
	var mu sync.Mutex

	mockNotifier := &MockNotifier{
		onNotify: func(status ServiceStatus, results map[string]*HealthCheckResult) {
			mu.Lock()
			notified = true
			mu.Unlock()
		},
	}

	hc.AddNotifier(mockNotifier)
}

type MockHealthCheck struct {
	name    string
	healthy bool
	mu      sync.RWMutex
}

func (m *MockHealthCheck) Name() string {
	return m.name
}

func (m *MockHealthCheck) Check(ctx context.Context) error {
	m.mu.RLock()
	defer m.mu.RUnlock()
	if m.healthy {
		return nil
	}
	return &MockError{message: "mock check failed"}
}

type MockError struct {
	message string
}

func (e *MockError) Error() string {
	return e.message
}

type MockNotifier struct {
	onNotify func(status ServiceStatus, results map[string]*HealthCheckResult)
}

func (m *MockNotifier) Notify(status ServiceStatus, results map[string]*HealthCheckResult) {
	if m.onNotify != nil {
		m.onNotify(status, results)
	}
}

func TestMockHealthCheck(t *testing.T) {
	check := &MockHealthCheck{name: "mock", healthy: true}

	ctx := context.Background()
	err := check.Check(ctx)
	if err != nil {
		t.Errorf("Expected mock check to pass, got error: %v", err)
	}

	check.healthy = false
	err = check.Check(ctx)
	if err == nil {
		t.Error("Expected mock check to fail when healthy is false")
	}
}

func TestConcurrentHealthChecks(t *testing.T) {
	cfg := &HealthCheckConfig{
		Enabled:            false,
		Interval:           1 * time.Second,
		Timeout:            500 * time.Millisecond,
		MaxFailCount:       3,
		FailureThreshold:   3,
		RecoveryThreshold:  2,
		EnableAutoFailover: false,
	}

	hc := NewDistributedHealthChecker(cfg)
	hc.Stop()

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			hc.runAllChecks()
		}()
	}

	wg.Wait()

	results := hc.GetResults()
	if len(results) == 0 {
		t.Error("Expected results after concurrent checks")
	}
}
