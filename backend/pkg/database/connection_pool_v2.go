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
)

type EnhancedConnectionPoolManager struct {
	poolStats              *AdvancedPoolStats
	config                 *EnhancedPoolConfig
	autoTuningEnabled      bool
	healthChecker          *PoolHealthChecker
	warmupManager          *ConnectionWarmupManager
	metricsCollector       *PoolMetricsCollector
	eventChannel           chan PoolEvent
	stopCh                 chan struct{}
	wg                     sync.WaitGroup
	mu                     sync.RWMutex
	currentConfig          *EnhancedPoolConfig
	tuningHistory          []TuningRecord
	maxHistorySize         int
	highLoadDetector       *HighLoadDetector
}

type EnhancedPoolConfig struct {
	MaxOpenConns              int
	MaxIdleConns              int
	MinIdleConns              int
	ConnMaxLifetime           time.Duration
	ConnMaxIdleTime           time.Duration
	HealthCheckInterval       time.Duration
	WaitTimeout               time.Duration
	MaxWaitCount              int
	EnableAutoTuning          bool
	HighLoadThreshold         float64
	LowLoadThreshold          float64
	TuningSensitivity         float64
	MinPoolSize               int
	MaxPoolSize               int
}

type AdvancedPoolStats struct {
	MaxOpenConnections        int
	OpenConnections           int
	InUse                     int
	Idle                      int
	WaitCount                 int64
	WaitDuration              time.Duration
	MaxIdleClosed             int64
	MaxLifetimeClosed         int64
	ConnectionsOpened         int64
	ConnectionsClosed         int64
	ActiveTime                time.Duration
	ReuseRate                 float64
	HealthScore               float64
	LastHealthCheck           time.Time
}

type PoolEvent struct {
	EventType      string
	Timestamp      time.Time
	Details        map[string]interface{}
}

type PoolMetricsCollector struct {
	history      []PoolMetricsSnapshot
	maxHistory   int
	mu           sync.RWMutex
	collectorTicker *time.Ticker
	stopCh       chan struct{}
	wg           sync.WaitGroup
}

type PoolMetricsSnapshot struct {
	Timestamp        time.Time
	TotalConnections int
	ActiveConnections int
	IdleConnections  int
	WaitCount        int64
	WaitDuration     time.Duration
	ReuseRate        float64
	HealthScore      float64
}

type TuningRecord struct {
	Timestamp     time.Time
	OldConfig     *EnhancedPoolConfig
	NewConfig     *EnhancedPoolConfig
	Reason        string
	MetricsBefore PoolMetricsSnapshot
	MetricsAfter  PoolMetricsSnapshot
}

type PoolHealthChecker struct {
	interval       time.Duration
	checkTicker    *time.Ticker
	stopCh         chan struct{}
	wg             sync.WaitGroup
	lastHealth     bool
	healthHistory  []bool
	maxHistorySize int
}

type ConnectionWarmupManager struct {
	targetConns    int
	currentConns   int
	warmupInterval time.Duration
	maxParallel    int
	enabled        bool
	stopCh         chan struct{}
	wg             sync.WaitGroup
	progress       atomic.Int32
}

type HighLoadDetector struct {
	windowSize     int
	loadHistory    []float64
	threshold      float64
	triggerCount   int
	currentCount   int
	mu             sync.Mutex
}

var enhancedPoolManager *EnhancedConnectionPoolManager

func InitEnhancedConnectionPool(cfg *config.Config) error {
	poolConfig := &EnhancedPoolConfig{
		MaxOpenConns:              cfg.Database.ConnectionPool.MaxOpenConns,
		MaxIdleConns:              cfg.Database.ConnectionPool.MaxIdleConns,
		MinIdleConns:              cfg.Database.ConnectionPool.MinIdleConns,
		ConnMaxLifetime:           time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second,
		ConnMaxIdleTime:           time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second,
		HealthCheckInterval:       time.Duration(cfg.Database.ConnectionPool.HealthCheckInterval) * time.Second,
		WaitTimeout:               time.Duration(cfg.Database.ConnectionPool.WaitTimeoutSecs) * time.Second,
		MaxWaitCount:              cfg.Database.ConnectionPool.MaxWaitCount,
		EnableAutoTuning:          cfg.Database.ConnectionPool.EnableAutoTuning,
		HighLoadThreshold:         float64(cfg.Database.ConnectionPool.HighLoadThreshold) / 100,
		LowLoadThreshold:          float64(cfg.Database.ConnectionPool.LowLoadThreshold) / 100,
		TuningSensitivity:         0.2,
		MinPoolSize:               5,
		MaxPoolSize:               cfg.Database.ConnectionPool.MaxOpenConns * 2,
	}

	manager := &EnhancedConnectionPoolManager{
		config:             poolConfig,
		currentConfig:      poolConfig,
		autoTuningEnabled:  cfg.Database.ConnectionPool.EnableAutoTuning,
		eventChannel:       make(chan PoolEvent, 100),
		stopCh:             make(chan struct{}),
		tuningHistory:      make([]TuningRecord, 0),
		maxHistorySize:     100,
		highLoadDetector:   NewHighLoadDetector(10, poolConfig.HighLoadThreshold),
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return fmt.Errorf("failed to get database connection: %w", err)
	}

	sqlDB.SetMaxOpenConns(poolConfig.MaxOpenConns)
	sqlDB.SetMaxIdleConns(poolConfig.MaxIdleConns)
	sqlDB.SetConnMaxLifetime(poolConfig.ConnMaxLifetime)
	sqlDB.SetConnMaxIdleTime(poolConfig.ConnMaxIdleTime)

	manager.healthChecker = NewPoolHealthChecker(manager, poolConfig.HealthCheckInterval)
	manager.warmupManager = NewConnectionWarmupManager(cfg.Database.ConnectionPool.WarmupConns)
	manager.metricsCollector = NewPoolMetricsCollector(100)

	if cfg.Database.ConnectionPool.EnableWarmup {
		manager.warmupManager.Start()
	}

	if cfg.Database.Monitoring.EnableConnectionMetrics {
		manager.metricsCollector.Start()
	}

	if cfg.Database.ConnectionPool.EnableAutoTuning {
		manager.wg.Add(1)
		go manager.runAutoTuningLoop()
	}

	manager.wg.Add(1)
	go manager.processEvents()

	enhancedPoolManager = manager

	log.Println("Enhanced connection pool manager initialized successfully")
	return nil
}

func GetEnhancedPoolManager() *EnhancedConnectionPoolManager {
	return enhancedPoolManager
}

func NewPoolHealthChecker(manager *EnhancedConnectionPoolManager, interval time.Duration) *PoolHealthChecker {
	return &PoolHealthChecker{
		interval:       interval,
		healthHistory:  make([]bool, 0),
		maxHistorySize: 10,
	}
}

func (hc *PoolHealthChecker) Start() {
	hc.wg.Add(1)
	go hc.checkLoop()
}

func (hc *PoolHealthChecker) Stop() {
	if hc.checkTicker != nil {
		hc.checkTicker.Stop()
	}
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *PoolHealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			hc.performCheck()
		case <-hc.stopCh:
			return
		}
	}
}

func (hc *PoolHealthChecker) performCheck() {
	sqlDB, err := DB.DB()
	if err != nil {
		hc.lastHealth = false
		hc.addHealthHistory(false)
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := sqlDB.PingContext(ctx); err != nil {
		hc.lastHealth = false
		hc.addHealthHistory(false)
		log.Printf("[POOL_HEALTH] Connection pool health check failed: %v", err)
		return
	}

	hc.lastHealth = true
	hc.addHealthHistory(true)
}

func (hc *PoolHealthChecker) addHealthHistory(healthy bool) {
	hc.healthHistory = append(hc.healthHistory, healthy)
	if len(hc.healthHistory) > hc.maxHistorySize {
		hc.healthHistory = hc.healthHistory[1:]
	}
}

func (hc *PoolHealthChecker) GetHealthScore() float64 {
	if len(hc.healthHistory) == 0 {
		return 1.0
	}

	count := 0
	for _, h := range hc.healthHistory {
		if h {
			count++
		}
	}

	return float64(count) / float64(len(hc.healthHistory))
}

func (hc *PoolHealthChecker) IsHealthy() bool {
	return hc.lastHealth
}

func NewConnectionWarmupManager(targetConns int) *ConnectionWarmupManager {
	return &ConnectionWarmupManager{
		targetConns:    targetConns,
		currentConns:   0,
		warmupInterval: 500 * time.Millisecond,
		maxParallel:    5,
		enabled:        true,
		stopCh:         make(chan struct{}),
	}
}

func (wm *ConnectionWarmupManager) Start() {
	if !wm.enabled {
		return
	}

	wm.wg.Add(1)
	go wm.warmupLoop()
}

func (wm *ConnectionWarmupManager) Stop() {
	wm.enabled = false
	close(wm.stopCh)
	wm.wg.Wait()
}

func (wm *ConnectionWarmupManager) warmupLoop() {
	defer wm.wg.Done()

	if wm.targetConns <= 0 {
		return
	}

	log.Printf("[POOL_WARMUP] Starting connection warmup, target: %d connections", wm.targetConns)

	sem := make(chan struct{}, wm.maxParallel)
	var wg sync.WaitGroup

	for i := 0; i < wm.targetConns; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sem <- struct{}{}
			defer func() { <-sem }()

			time.Sleep(time.Duration(idx) * wm.warmupInterval)

			if !wm.enabled {
				return
			}

			if err := wm.warmupConnection(); err != nil {
				log.Printf("[POOL_WARMUP] Failed to warm up connection %d: %v", idx, err)
			} else {
				wm.progress.Add(1)
			}
		}(i)
	}

	wg.Wait()
	log.Printf("[POOL_WARMUP] Warmup completed, established %d connections", wm.progress.Load())
}

func (wm *ConnectionWarmupManager) warmupConnection() error {
	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	conn, err := sqlDB.Conn(ctx)
	if err != nil {
		return err
	}

	var result int
	err = conn.QueryRowContext(ctx, "SELECT 1").Scan(&result)
	conn.Close()

	return err
}

func (wm *ConnectionWarmupManager) GetProgress() int {
	return int(wm.progress.Load())
}

func NewPoolMetricsCollector(maxHistory int) *PoolMetricsCollector {
	return &PoolMetricsCollector{
		history:    make([]PoolMetricsSnapshot, 0),
		maxHistory: maxHistory,
		stopCh:     make(chan struct{}),
	}
}

func (mc *PoolMetricsCollector) Start() {
	mc.wg.Add(1)
	go mc.collectLoop()
}

func (mc *PoolMetricsCollector) Stop() {
	if mc.collectorTicker != nil {
		mc.collectorTicker.Stop()
	}
	close(mc.stopCh)
	mc.wg.Wait()
}

func (mc *PoolMetricsCollector) collectLoop() {
	defer mc.wg.Done()

	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			mc.collect()
		case <-mc.stopCh:
			return
		}
	}
}

func (mc *PoolMetricsCollector) collect() {
	snapshot := mc.takeSnapshot()

	mc.mu.Lock()
	mc.history = append(mc.history, snapshot)
	if len(mc.history) > mc.maxHistory {
		mc.history = mc.history[1:]
	}
	mc.mu.Unlock()
}

func (mc *PoolMetricsCollector) takeSnapshot() PoolMetricsSnapshot {
	if DB == nil {
		return PoolMetricsSnapshot{Timestamp: time.Now()}
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return PoolMetricsSnapshot{Timestamp: time.Now()}
	}

	stats := sqlDB.Stats()

	return PoolMetricsSnapshot{
		Timestamp:        time.Now(),
		TotalConnections: stats.MaxOpenConnections,
		ActiveConnections: stats.InUse,
		IdleConnections:  stats.Idle,
		WaitCount:        stats.WaitCount,
		WaitDuration:     stats.WaitDuration,
	}
}

func (mc *PoolMetricsCollector) GetHistory() []PoolMetricsSnapshot {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	history := make([]PoolMetricsSnapshot, len(mc.history))
	copy(history, mc.history)
	return history
}

func NewHighLoadDetector(windowSize int, threshold float64) *HighLoadDetector {
	return &HighLoadDetector{
		windowSize:   windowSize,
		loadHistory:  make([]float64, 0),
		threshold:    threshold,
		triggerCount: 3,
		currentCount: 0,
	}
}

func (hd *HighLoadDetector) RecordLoad(load float64) bool {
	hd.mu.Lock()
	defer hd.mu.Unlock()

	hd.loadHistory = append(hd.loadHistory, load)
	if len(hd.loadHistory) > hd.windowSize {
		hd.loadHistory = hd.loadHistory[1:]
	}

	if load >= hd.threshold {
		hd.currentCount++
		if hd.currentCount >= hd.triggerCount {
			hd.currentCount = 0
			return true
		}
	} else {
		hd.currentCount = 0
	}

	return false
}

func (hd *HighLoadDetector) GetAverageLoad() float64 {
	hd.mu.Lock()
	defer hd.mu.Unlock()

	if len(hd.loadHistory) == 0 {
		return 0
	}

	sum := 0.0
	for _, load := range hd.loadHistory {
		sum += load
	}

	return sum / float64(len(hd.loadHistory))
}

func (m *EnhancedConnectionPoolManager) runAutoTuningLoop() {
	defer m.wg.Done()

	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			m.performAutoTuning()
		case <-m.stopCh:
			return
		}
	}
}

func (m *EnhancedConnectionPoolManager) performAutoTuning() {
	if !m.autoTuningEnabled {
		return
	}

	metrics := m.GetMetrics()
	if metrics == nil {
		return
	}

	usagePercent := float64(metrics.InUse) / float64(metrics.OpenConnections)

	isHighLoad := m.highLoadDetector.RecordLoad(usagePercent)

	if isHighLoad {
		m.handleHighLoad(metrics)
	} else if usagePercent < m.config.LowLoadThreshold {
		m.handleLowLoad(metrics)
	}
}

func (m *EnhancedConnectionPoolManager) handleHighLoad(metrics *AdvancedPoolStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sqlDB, err := DB.DB()
	if err != nil {
		return
	}

	currentMaxOpen := sqlDB.Stats().MaxOpenConnections
	newMaxOpen := int(float64(currentMaxOpen) * (1 + m.config.TuningSensitivity))

	if newMaxOpen > m.config.MaxPoolSize {
		newMaxOpen = m.config.MaxPoolSize
	}

	if newMaxOpen > currentMaxOpen {
		oldConfig := &EnhancedPoolConfig{MaxOpenConns: currentMaxOpen}
		sqlDB.SetMaxOpenConns(newMaxOpen)

		newMaxIdle := newMaxOpen / 2
		currentMaxIdle := sqlDB.Stats().OpenConnections
		if newMaxIdle > currentMaxIdle {
			sqlDB.SetMaxIdleConns(newMaxIdle)
		}

		m.currentConfig.MaxOpenConns = newMaxOpen

		m.recordTuning(oldConfig, &EnhancedPoolConfig{MaxOpenConns: newMaxOpen}, "high load detected")
		log.Printf("[AUTO_TUNE] Increased pool size: MaxOpen=%d, MaxIdle=%d (load: %.1f%%)", newMaxOpen, newMaxIdle, float64(metrics.InUse)/float64(metrics.OpenConnections)*100)
	}
}

func (m *EnhancedConnectionPoolManager) handleLowLoad(metrics *AdvancedPoolStats) {
	m.mu.Lock()
	defer m.mu.Unlock()

	sqlDB, err := DB.DB()
	if err != nil {
		return
	}

	stats := sqlDB.Stats()
	if stats.Idle > m.config.MinIdleConns {
		newMaxIdle := int(float64(stats.Idle) * (1 - m.config.TuningSensitivity))
		if newMaxIdle < m.config.MinIdleConns {
			newMaxIdle = m.config.MinIdleConns
		}

		currentMaxIdle := sqlDB.Stats().Idle
		if newMaxIdle < currentMaxIdle {
			sqlDB.SetMaxIdleConns(newMaxIdle)
			log.Printf("[AUTO_TUNE] Reduced idle connections: MaxIdle=%d (load: %.1f%%)", newMaxIdle, float64(metrics.InUse)/float64(metrics.OpenConnections)*100)
		}
	}
}

func (m *EnhancedConnectionPoolManager) recordTuning(oldConfig, newConfig *EnhancedPoolConfig, reason string) {
	record := TuningRecord{
		Timestamp: time.Now(),
		OldConfig: oldConfig,
		NewConfig: newConfig,
		Reason:    reason,
	}

	m.tuningHistory = append(m.tuningHistory, record)
	if len(m.tuningHistory) > m.maxHistorySize {
		m.tuningHistory = m.tuningHistory[1:]
	}
}

func (m *EnhancedConnectionPoolManager) processEvents() {
	defer m.wg.Done()

	for {
		select {
		case event := <-m.eventChannel:
			m.handleEvent(event)
		case <-m.stopCh:
			return
		}
	}
}

func (m *EnhancedConnectionPoolManager) handleEvent(event PoolEvent) {
	switch event.EventType {
	case "high_load":
		log.Printf("[POOL_EVENT] High load detected: %v", event.Details)
	case "low_load":
		log.Printf("[POOL_EVENT] Low load detected: %v", event.Details)
	case "health_degraded":
		log.Printf("[POOL_EVENT] Health degraded: %v", event.Details)
	case "pool_resized":
		log.Printf("[POOL_EVENT] Pool resized: %v", event.Details)
	}
}

func (m *EnhancedConnectionPoolManager) GetMetrics() *AdvancedPoolStats {
	if DB == nil {
		return &AdvancedPoolStats{}
	}

	sqlDB, err := DB.DB()
	if err != nil {
		return &AdvancedPoolStats{}
	}

	stats := sqlDB.Stats()

	healthScore := 1.0
	if m.healthChecker != nil {
		healthScore = m.healthChecker.GetHealthScore()
	}

	reuseRate := 0.0
	if stats.InUse+stats.Idle > 0 {
		reuseRate = float64(stats.InUse) / float64(stats.InUse+stats.Idle) * 100
	}

	return &AdvancedPoolStats{
		MaxOpenConnections:  stats.MaxOpenConnections,
		OpenConnections:     stats.OpenConnections,
		InUse:              stats.InUse,
		Idle:               stats.Idle,
		WaitCount:          stats.WaitCount,
		WaitDuration:       stats.WaitDuration,
		MaxIdleClosed:      stats.MaxIdleClosed,
		MaxLifetimeClosed:  stats.MaxLifetimeClosed,
		ReuseRate:          reuseRate,
		HealthScore:        healthScore,
		LastHealthCheck:    time.Now(),
	}
}

func (m *EnhancedConnectionPoolManager) GetTuningHistory() []TuningRecord {
	m.mu.RLock()
	defer m.mu.RUnlock()

	history := make([]TuningRecord, len(m.tuningHistory))
	copy(history, m.tuningHistory)
	return history
}

func (m *EnhancedConnectionPoolManager) UpdateConfig(config *EnhancedPoolConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	sqlDB, err := DB.DB()
	if err != nil {
		return err
	}

	if config.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(config.MaxOpenConns)
	}
	if config.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(config.MaxIdleConns)
	}
	if config.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(config.ConnMaxLifetime)
	}
	if config.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(config.ConnMaxIdleTime)
	}

	m.currentConfig = config
	m.config = config

	return nil
}

func (m *EnhancedConnectionPoolManager) EnableAutoTuning(enabled bool) {
	m.mu.Lock()
	m.autoTuningEnabled = enabled
	m.mu.Unlock()
}

func (m *EnhancedConnectionPoolManager) GetHealthStatus() *PoolHealthStatus {
	metrics := m.GetMetrics()
	isHealthy := metrics.HealthScore >= 0.8

	issues := make([]string, 0)
	recommendations := make([]string, 0)

	if metrics.WaitCount > int64(m.config.MaxWaitCount) {
		issues = append(issues, "high wait count detected")
		recommendations = append(recommendations, "consider increasing max_open_conns")
	}

	if metrics.HealthScore < 0.8 {
		issues = append(issues, "health score below threshold")
		recommendations = append(recommendations, "check database connectivity")
	}

	if metrics.ReuseRate > 95 {
		recommendations = append(recommendations, "high connection reuse rate, consider increasing pool size")
	}

	return &PoolHealthStatus{
		IsHealthy:       isHealthy,
		Score:           metrics.HealthScore,
		Issues:          issues,
		Recommendations: recommendations,
		LastCheck:       time.Now(),
	}
}

type PoolHealthStatus struct {
	IsHealthy       bool
	Score           float64
	Issues          []string
	Recommendations []string
	LastCheck       time.Time
}

func (m *EnhancedConnectionPoolManager) Close() {
	close(m.stopCh)
	m.wg.Wait()

	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}
	if m.warmupManager != nil {
		m.warmupManager.Stop()
	}
	if m.metricsCollector != nil {
		m.metricsCollector.Stop()
	}
}

func (m *EnhancedConnectionPoolManager) GetConnectionPressure() *ConnectionPressure {
	metrics := m.GetMetrics()

	var pressureLevel string
	var advice string

	usage := float64(metrics.InUse) / float64(metrics.MaxOpenConnections)

	switch {
	case usage > 0.9:
		pressureLevel = "critical"
		advice = "immediate action needed: connections exhausted"
	case usage > 0.7:
		pressureLevel = "high"
		advice = "consider increasing pool size"
	case usage > 0.5:
		pressureLevel = "normal"
		advice = "connections healthy"
	default:
		pressureLevel = "low"
		advice = "underutilized, consider reducing pool size"
	}

	return &ConnectionPressure{
		Timestamp:      time.Now(),
		OpenConnections: metrics.MaxOpenConnections,
		InUse:          metrics.InUse,
		Idle:           metrics.Idle,
		WaitCount:      metrics.WaitCount,
		PressureLevel:  pressureLevel,
		Advice:         advice,
	}
}

type ConnectionPressure struct {
	Timestamp      time.Time
	OpenConnections int
	InUse          int
	Idle           int
	WaitCount      int64
	PressureLevel  string
	Advice         string
}

func (m *EnhancedConnectionPoolManager) GetOptimalPoolSize() int {
	history := m.metricsCollector.GetHistory()
	if len(history) < 5 {
		return m.config.MaxOpenConns
	}

	maxActive := 0
	for _, snapshot := range history {
		if snapshot.ActiveConnections > maxActive {
			maxActive = snapshot.ActiveConnections
		}
	}

	return int(math.Ceil(float64(maxActive) * 1.2))
}