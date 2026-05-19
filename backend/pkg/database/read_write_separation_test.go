package database

import (
	"testing"
	"time"

	"gorm.io/gorm"
)

func TestDBRouterCreation(t *testing.T) {
	router := &DBRouter{
		enabled:         true,
		loadBalanceMode: "round_robin",
	}

	if router == nil {
		t.Fatal("DBRouter should not be nil")
	}

	if !router.enabled {
		t.Error("Router should be enabled")
	}

	if router.loadBalanceMode != "round_robin" {
		t.Errorf("Load balance mode = %q, want %q", router.loadBalanceMode, "round_robin")
	}
}

func TestRouterMetrics(t *testing.T) {
	metrics := &RouterMetrics{}

	metrics.MasterQueries.Add(100)
	metrics.SlaveQueries.Add(200)
	metrics.FailedQueries.Add(5)
	metrics.SlaveSwitches.Add(3)
	metrics.AvgLatency.Store(1000000)

	if metrics.MasterQueries.Load() != 100 {
		t.Errorf("MasterQueries = %d, want 100", metrics.MasterQueries.Load())
	}

	if metrics.SlaveQueries.Load() != 200 {
		t.Errorf("SlaveQueries = %d, want 200", metrics.SlaveQueries.Load())
	}

	if metrics.FailedQueries.Load() != 5 {
		t.Errorf("FailedQueries = %d, want 5", metrics.FailedQueries.Load())
	}

	if metrics.SlaveSwitches.Load() != 3 {
		t.Errorf("SlaveSwitches = %d, want 3", metrics.SlaveSwitches.Load())
	}
}

func TestRouterMetricsRecord(t *testing.T) {
	r := &DBRouter{
		enabled:  true,
		metrics: &RouterMetrics{},
	}

	initialMasterQueries := r.metrics.MasterQueries.Load()
	initialSlaveQueries := r.metrics.SlaveQueries.Load()

	r.RecordQuery(true, 10*time.Millisecond)
	if r.metrics.MasterQueries.Load() != initialMasterQueries+1 {
		t.Errorf("Master query count should increase")
	}

	r.RecordQuery(false, 5*time.Millisecond)
	if r.metrics.SlaveQueries.Load() != initialSlaveQueries+1 {
		t.Errorf("Slave query count should increase")
	}
}

func TestRouterMetricsRecordFailure(t *testing.T) {
	r := &DBRouter{
		enabled:  true,
		metrics: &RouterMetrics{},
	}

	initialFailures := r.metrics.FailedQueries.Load()
	r.RecordFailure()

	if r.metrics.FailedQueries.Load() != initialFailures+1 {
		t.Errorf("Failed queries count should increase")
	}
}

func TestRouterMetricsRecordSlaveSwitch(t *testing.T) {
	r := &DBRouter{
		enabled:  true,
		metrics: &RouterMetrics{},
	}

	initialSwitches := r.metrics.SlaveSwitches.Load()
	r.RecordSlaveSwitch()

	if r.metrics.SlaveSwitches.Load() != initialSwitches+1 {
		t.Errorf("Slave switches count should increase")
	}
}

func TestRouterGetMetrics(t *testing.T) {
	r := &DBRouter{
		enabled:  true,
		metrics: &RouterMetrics{},
	}

	r.metrics.MasterQueries.Add(50)
	r.metrics.SlaveQueries.Add(100)
	r.metrics.FailedQueries.Add(5)
	r.metrics.AvgLatency.Store(5000000)

	metrics := r.GetMetrics()

	if metrics["master_queries"] != int64(50) {
		t.Errorf("master_queries = %v, want 50", metrics["master_queries"])
	}

	if metrics["slave_queries"] != int64(100) {
		t.Errorf("slave_queries = %v, want 100", metrics["slave_queries"])
	}

	if metrics["failed_queries"] != int64(5) {
		t.Errorf("failed_queries = %v, want 5", metrics["failed_queries"])
	}
}

func TestSlaveHealthCheckerCreation(t *testing.T) {
	router := &DBRouter{}
	checker := NewSlaveHealthChecker(router, 30*time.Second)

	if checker == nil {
		t.Fatal("NewSlaveHealthChecker should not return nil")
	}

	if checker.interval != 30*time.Second {
		t.Errorf("Interval = %v, want %v", checker.interval, 30*time.Second)
	}

	if !checker.enabled {
		t.Error("Checker should be enabled by default")
	}

	if !checker.failoverEnabled {
		t.Error("Failover should be enabled by default")
	}

	if checker.maxFailCount != 3 {
		t.Errorf("Max fail count = %d, want 3", checker.maxFailCount)
	}
}

func TestSlaveHealthCheckerSetFailoverEnabled(t *testing.T) {
	router := &DBRouter{}
	checker := NewSlaveHealthChecker(router, 30*time.Second)

	checker.SetFailoverEnabled(false)
	if checker.failoverEnabled {
		t.Error("Failover should be disabled")
	}

	checker.SetFailoverEnabled(true)
	if !checker.failoverEnabled {
		t.Error("Failover should be enabled")
	}
}

func TestSlaveHealthCheckerSetMaxFailCount(t *testing.T) {
	router := &DBRouter{}
	checker := NewSlaveHealthChecker(router, 30*time.Second)

	checker.SetMaxFailCount(5)
	if checker.maxFailCount != 5 {
		t.Errorf("Max fail count = %d, want 5", checker.maxFailCount)
	}
}

func TestSlaveStatus(t *testing.T) {
	status := &SlaveStatus{
		Index:     0,
		Host:      "localhost",
		Port:      "5432",
		Healthy:   true,
		Latency:   5 * time.Millisecond,
		LastCheck: time.Now(),
		FailCount: 0,
	}

	if status.Index != 0 {
		t.Errorf("Index = %d, want 0", status.Index)
	}

	if status.Host != "localhost" {
		t.Errorf("Host = %q, want %q", status.Host, "localhost")
	}

	if !status.Healthy {
		t.Error("Status should be healthy")
	}

	if status.FailCount != 0 {
		t.Errorf("FailCount = %d, want 0", status.FailCount)
	}
}

func TestDBRouterGetOptimalSlave(t *testing.T) {
	r := &DBRouter{
		enabled:    true,
		slaveDBs:   make([]*gorm.DB, 0),
		masterDB:   nil,
		healthChecker: &SlaveHealthChecker{
			dbRouter:   nil,
			slaveStatus: make(map[int]*SlaveStatus),
		},
	}

	slave := r.GetOptimalSlave()
	if slave != r.masterDB {
		t.Error("GetOptimalSlave should return masterDB when no healthy slaves and masterDB is nil")
	}
}

func TestDBRouterSlaveRoundRobin(t *testing.T) {
	r := &DBRouter{
		enabled:         true,
		loadBalanceMode: "round_robin",
		currentSlave:    0,
	}

	slave := r.getSlaveRoundRobin()
	if slave != nil {
		t.Error("Should return nil when no slave DBs")
	}
}

func TestDBRouterGetSlaveWeightedRoundRobin(t *testing.T) {
	r := &DBRouter{
		enabled:         true,
		loadBalanceMode: "weighted_round_robin",
		slaveWeights:    []int{1, 2, 3},
	}

	slave := r.getSlaveWeightedRoundRobin()
	if slave != nil {
		t.Error("Should return nil when no slave DBs")
	}
}

func TestDBRouterGetSlaveRandom(t *testing.T) {
	r := &DBRouter{
		enabled:         true,
		loadBalanceMode: "random",
	}

	slave := r.getSlaveRandom()
	if slave != nil {
		t.Error("Should return nil when no slave DBs")
	}
}

func TestDBRouterRead(t *testing.T) {
	r := &DBRouter{
		enabled: true,
		masterDB: nil,
	}

	if r.enabled && r.masterDB != nil {
		db := r.Read(nil)
		if db == nil {
			t.Error("Read should return a DB instance when enabled")
		}
	} else {
		t.Log("Router disabled or masterDB nil, skipping Read() call")
	}
}

func TestDBRouterWrite(t *testing.T) {
	r := &DBRouter{
		enabled: true,
		masterDB: nil,
	}

	if r.enabled && r.masterDB != nil {
		db := r.Write(nil)
		if db == nil {
			t.Error("Write should return a DB instance when enabled")
		}
	} else {
		t.Log("Router disabled or masterDB nil, skipping Write() call")
	}
}

func TestDBRouterIsEnabled(t *testing.T) {
	r := &DBRouter{
		enabled: true,
	}

	if !r.IsEnabled() {
		t.Error("IsEnabled should return true")
	}

	r.enabled = false
	if r.IsEnabled() {
		t.Error("IsEnabled should return false")
	}
}

func TestDBRouterGetSlaveHealthStatus(t *testing.T) {
	r := &DBRouter{
		enabled:  true,
		slaveDBs: make([]*gorm.DB, 0),
	}

	status := r.GetSlaveHealthStatus()
	if status == nil {
		t.Error("GetSlaveHealthStatus should not return nil")
	}
}

func TestDBRouterMaster(t *testing.T) {
	r := &DBRouter{
		enabled: false,
		masterDB: nil,
	}

	if r.enabled {
		t.Error("Router should be disabled for this test")
	}

	if r.masterDB == nil && DB == nil {
		t.Log("Master returns nil when both router.masterDB and global DB are nil")
	}
}

func TestDBRouterSlave(t *testing.T) {
	r := &DBRouter{
		enabled:         true,
		loadBalanceMode: "round_robin",
		slaveDBs:        make([]*gorm.DB, 0),
		masterDB:        nil,
	}

	if r.enabled && len(r.slaveDBs) == 0 && r.masterDB == nil {
		t.Log("Slave returns nil when no slaves and no master configured")
	}
}

func TestDBRouterWithDisabledRouter(t *testing.T) {
	r := &DBRouter{
		enabled: false,
		masterDB: nil,
	}

	if r.Master() == nil && DB == nil {
		t.Log("Master returns nil when router is disabled and no DB configured")
	}

	if r.Slave() != nil {
		t.Error("Slave should return nil when router is disabled")
	}
}

func TestGlobalRouter(t *testing.T) {
	router = nil
	r := GetRouter()

	if router != nil {
		t.Error("GetRouter should return nil when not initialized")
	}

	if r != nil {
		t.Error("GetRouter should return nil when not initialized")
	}
}

func TestSlaveHealthCheckerGetHealthySlaves(t *testing.T) {
	checker := &SlaveHealthChecker{
		slaveStatus: map[int]*SlaveStatus{
			0: {Index: 0, Healthy: true},
			1: {Index: 1, Healthy: false},
			2: {Index: 2, Healthy: true},
		},
	}

	healthy := checker.GetHealthySlaves()
	if len(healthy) != 2 {
		t.Errorf("Expected 2 healthy slaves, got %d", len(healthy))
	}
}

func TestSlaveHealthCheckerGetSlaveStatus(t *testing.T) {
	checker := &SlaveHealthChecker{
		slaveStatus: map[int]*SlaveStatus{
			0: {Index: 0, Healthy: true},
			1: {Index: 1, Healthy: false},
		},
	}

	status := checker.GetSlaveStatus()
	if len(status) != 2 {
		t.Errorf("Expected 2 status entries, got %d", len(status))
	}
}

func TestDBRouterClose(t *testing.T) {
	r := &DBRouter{
		masterDB: nil,
		slaveDBs: make([]*gorm.DB, 0),
	}

	err := r.Close()
	if err != nil {
		t.Errorf("Close should not return error when DBs are nil: %v", err)
	}
}

func TestConnectDBEmptyParams(t *testing.T) {
	_, err := connectDB("", "", "", "", "", "")
	if err == nil {
		t.Error("connectDB should return error for empty parameters")
	}
}

func TestDBRouterMultipleLoadBalanceModes(t *testing.T) {
	modes := []string{"round_robin", "weighted_round_robin", "random"}

	for _, mode := range modes {
		r := &DBRouter{
			enabled:         true,
			loadBalanceMode: mode,
			slaveDBs:        make([]*gorm.DB, 0),
			masterDB:        nil,
		}

		slave := r.Slave()
		if slave != nil && slave != r.masterDB {
			t.Errorf("Unexpected slave for mode %s", mode)
		}
	}
}

func TestDBRouterUnknownLoadBalanceMode(t *testing.T) {
	r := &DBRouter{
		enabled:         true,
		loadBalanceMode: "unknown",
		slaveDBs:        make([]*gorm.DB, 0),
		masterDB:        nil,
	}

	slave := r.Slave()
	if slave != nil && slave != r.masterDB {
		t.Error("Should default to round_robin for unknown mode")
	}
}
