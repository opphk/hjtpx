package database

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type EnhancedConnectionPoolOptimizer struct {
	db                    *gorm.DB
	mu                    sync.RWMutex
	currentConfig         *EnhancedPoolConfig
	healthCheckInterval   time.Duration
	autoTuningEnabled      bool
	lastTuneTime          time.Time
	tuningHistory         []TuningRecord
	maxHistorySize        int
	recoveryEnabled        bool
}

type EnhancedPoolConfig struct {
	MaxOpenConns          int
	MaxIdleConns          int
	MinIdleConns          int
	ConnMaxLifetime       time.Duration
	ConnMaxIdleTime       time.Duration
	WaitTimeout           time.Duration
	MaxWaitCount          int
	HealthCheckInterval   time.Duration
	ConnectionTimeout     time.Duration
	RetryAttempts         int
	RetryDelay            time.Duration
}

type TuningRecord struct {
	Timestamp     time.Time
	OldConfig     *EnhancedPoolConfig
	NewConfig     *EnhancedPoolConfig
	Reason        string
	MetricsBefore *ConnectionPoolMetrics
	MetricsAfter  *ConnectionPoolMetrics
}

type PoolHealthStatus struct {
	IsHealthy      bool
	Score          float64
	Issues         []string
	Recommendations []string
	LastCheck      time.Time
}

type ConnectionPressure struct {
	Timestamp       time.Time
	OpenConnections int
	InUse          int
	Idle           int
	WaitCount      int64
	PressureLevel  string
	Advice         string
}

var globalEnhancedPoolOptimizer *EnhancedConnectionPoolOptimizer

func NewEnhancedConnectionPoolOptimizer(db *gorm.DB, cfg *config.Config) *EnhancedConnectionPoolOptimizer {
	optimizer := &EnhancedConnectionPoolOptimizer{
		db:                    db,
		healthCheckInterval:   30 * time.Second,
		autoTuningEnabled:     true,
		maxHistorySize:        100,
		recoveryEnabled:       true,
		tuningHistory:         make([]TuningRecord, 0),
	}

	if cfg != nil {
		optimizer.currentConfig = &EnhancedPoolConfig{
			MaxOpenConns:        cfg.Database.ConnectionPool.MaxOpenConns,
			MaxIdleConns:        cfg.Database.ConnectionPool.MaxIdleConns,
			MinIdleConns:        cfg.Database.ConnectionPool.MinIdleConns,
			ConnMaxLifetime:     time.Duration(cfg.Database.ConnectionPool.ConnMaxLifetimeSecs) * time.Second,
			ConnMaxIdleTime:     time.Duration(cfg.Database.ConnectionPool.ConnMaxIdleTimeSecs) * time.Second,
			HealthCheckInterval: time.Duration(cfg.Database.ConnectionPool.HealthCheckInterval) * time.Second,
			ConnectionTimeout:   30 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          100 * time.Millisecond,
		}
	}

	if optimizer.currentConfig == nil {
		optimizer.currentConfig = &EnhancedPoolConfig{
			MaxOpenConns:        100,
			MaxIdleConns:        20,
			MinIdleConns:        5,
			ConnMaxLifetime:     30 * time.Minute,
			ConnMaxIdleTime:     10 * time.Minute,
			HealthCheckInterval: 30 * time.Second,
			ConnectionTimeout:   30 * time.Second,
			RetryAttempts:       3,
			RetryDelay:          100 * time.Millisecond,
		}
	}

	return optimizer
}

func GetEnhancedPoolOptimizer() *EnhancedConnectionPoolOptimizer {
	return globalEnhancedPoolOptimizer
}

func (o *EnhancedConnectionPoolOptimizer) Start() {
	if o.autoTuningEnabled {
		go o.runHealthCheckLoop()
		go o.monitorConnectionPressure()
	}
}

func (o *EnhancedConnectionPoolOptimizer) runHealthCheckLoop() {
	ticker := time.NewTicker(o.healthCheckInterval)
	defer ticker.Stop()

	for range ticker.C {
		o.checkAndOptimize()
	}
}

func (o *EnhancedConnectionPoolOptimizer) checkAndOptimize() {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		log.Printf("[POOL_OPT] 获取连接池指标失败: %v", err)
		return
	}

	o.mu.Lock()
	defer o.mu.Unlock()

	healthStatus := o.evaluateHealth(metrics)
	if !healthStatus.IsHealthy {
		for _, issue := range healthStatus.Issues {
			log.Printf("[POOL_OPT] 健康问题: %s", issue)
		}
		o.applyOptimization(metrics, healthStatus)
	}

	if time.Since(o.lastTuneTime) > 5*time.Minute {
		o.evaluateAndTune(metrics)
	}
}

func (o *EnhancedConnectionPoolOptimizer) CheckAndOptimize() {
	o.checkAndOptimize()
}

func (o *EnhancedConnectionPoolOptimizer) evaluateHealth(metrics *ConnectionPoolMetrics) *PoolHealthStatus {
	status := &PoolHealthStatus{
		IsHealthy:      true,
		Score:          100.0,
		Issues:         make([]string, 0),
		Recommendations: make([]string, 0),
		LastCheck:      time.Now(),
	}

	if metrics.TotalConnections == 0 {
		status.IsHealthy = false
		status.Score = 0
		status.Issues = append(status.Issues, "连接池未正确初始化")
		return status
	}

	usagePercent := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100

	if usagePercent > 90 {
		status.IsHealthy = false
		status.Score -= 50
		status.Issues = append(status.Issues, fmt.Sprintf("连接使用率过高: %.1f%%", usagePercent))
		status.Recommendations = append(status.Recommendations, "建议增加MaxOpenConns或优化查询")
	} else if usagePercent > 80 {
		status.Score -= 20
		status.Recommendations = append(status.Recommendations, "连接使用率偏高，密切监控")
	}

	if metrics.IdleConnections == 0 && metrics.ActiveConnections > 0 {
		status.Score -= 10
		status.Issues = append(status.Issues, "没有空闲连接，可能导致连接等待")
		status.Recommendations = append(status.Recommendations, "考虑增加MaxIdleConns")
	}

	if metrics.WaitCount > 1000 {
		status.Score -= 30
		status.Issues = append(status.Issues, fmt.Sprintf("等待连接数过多: %d", metrics.WaitCount))
		status.Recommendations = append(status.Recommendations, "查询耗时过长或连接池过小")
	}

	if metrics.StaleConnections > 0 {
		status.Score -= 15
		status.Issues = append(status.Issues, fmt.Sprintf("存在过期连接: %d", metrics.StaleConnections))
		status.Recommendations = append(status.Recommendations, "调整ConnMaxLifetime或ConnMaxIdleTime")
	}

	if status.Score < 50 {
		status.IsHealthy = false
	}

	return status
}

func (o *EnhancedConnectionPoolOptimizer) applyOptimization(metrics *ConnectionPoolMetrics, status *PoolHealthStatus) {
	if len(status.Issues) == 0 {
		return
	}

	config := o.currentConfig

	usagePercent := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100

	if usagePercent > 80 {
		oldConfig := *config

		newMaxOpen := int(float64(config.MaxOpenConns) * 1.2)
		if newMaxOpen > config.MaxOpenConns*3 {
			newMaxOpen = config.MaxOpenConns * 3
		}
		config.MaxOpenConns = newMaxOpen

		newMaxIdle := newMaxOpen / 2
		if newMaxIdle < config.MinIdleConns {
			newMaxIdle = config.MinIdleConns
		}
		config.MaxIdleConns = newMaxIdle

		o.applyConfig(config)

		o.recordTuning(&TuningRecord{
			Timestamp: time.Now(),
			OldConfig: &oldConfig,
			NewConfig: config,
			Reason:    fmt.Sprintf("连接使用率过高: %.1f%%", usagePercent),
		})

		log.Printf("[POOL_OPT] 已增加连接池: MaxOpen=%d, MaxIdle=%d", config.MaxOpenConns, config.MaxIdleConns)
	}

	if metrics.IdleConnections < config.MinIdleConns {
		config.MaxIdleConns = config.MinIdleConns * 2
		o.applyConfig(config)
		log.Printf("[POOL_OPT] 已调整空闲连接数: MaxIdle=%d", config.MaxIdleConns)
	}
}

func (o *EnhancedConnectionPoolOptimizer) evaluateAndTune(metrics *ConnectionPoolMetrics) {
	usagePercent := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100

	if usagePercent < 20 && metrics.TotalConnections > 50 {
		oldConfig := *o.currentConfig

		newMaxOpen := metrics.TotalConnections - 20
		if newMaxOpen < 20 {
			newMaxOpen = 20
		}
		o.currentConfig.MaxOpenConns = newMaxOpen

		o.applyConfig(o.currentConfig)

		o.recordTuning(&TuningRecord{
			Timestamp: time.Now(),
			OldConfig: &oldConfig,
			NewConfig: o.currentConfig,
			Reason:    fmt.Sprintf("低使用率优化: %.1f%%", usagePercent),
		})

		log.Printf("[POOL_OPT] 已缩减连接池: MaxOpen=%d", o.currentConfig.MaxOpenConns)
	}

	o.lastTuneTime = time.Now()
}

func (o *EnhancedConnectionPoolOptimizer) applyConfig(cfg *EnhancedPoolConfig) error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	if cfg.MaxOpenConns > 0 {
		sqlDB.SetMaxOpenConns(cfg.MaxOpenConns)
	}
	if cfg.MaxIdleConns > 0 {
		sqlDB.SetMaxIdleConns(cfg.MaxIdleConns)
	}
	if cfg.ConnMaxLifetime > 0 {
		sqlDB.SetConnMaxLifetime(cfg.ConnMaxLifetime)
	}
	if cfg.ConnMaxIdleTime > 0 {
		sqlDB.SetConnMaxIdleTime(cfg.ConnMaxIdleTime)
	}

	return nil
}

func (o *EnhancedConnectionPoolOptimizer) recordTuning(record *TuningRecord) {
	o.tuningHistory = append(o.tuningHistory, *record)
	if len(o.tuningHistory) > o.maxHistorySize {
		o.tuningHistory = o.tuningHistory[1:]
	}
}

func (o *EnhancedConnectionPoolOptimizer) GetTuningHistory() []TuningRecord {
	o.mu.RLock()
	defer o.mu.RUnlock()

	history := make([]TuningRecord, len(o.tuningHistory))
	copy(history, o.tuningHistory)
	return history
}

func (o *EnhancedConnectionPoolOptimizer) monitorConnectionPressure() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		pressure := o.calculatePressure()
		if pressure.PressureLevel == "high" || pressure.PressureLevel == "critical" {
			log.Printf("[POOL_PRESSURE] %s: %s", pressure.PressureLevel, pressure.Advice)
		}
	}
}

func (o *EnhancedConnectionPoolOptimizer) calculatePressure() *ConnectionPressure {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		return &ConnectionPressure{
			PressureLevel: "unknown",
			Advice:        "无法获取连接池指标",
		}
	}

	pressure := &ConnectionPressure{
		Timestamp:       time.Now(),
		OpenConnections: metrics.TotalConnections,
		InUse:          metrics.ActiveConnections,
		Idle:           metrics.IdleConnections,
		WaitCount:      metrics.WaitCount,
	}

	usagePercent := float64(metrics.ActiveConnections) / float64(metrics.TotalConnections) * 100

	if usagePercent > 95 || metrics.WaitCount > 5000 {
		pressure.PressureLevel = "critical"
		pressure.Advice = "立即扩容并检查慢查询"
	} else if usagePercent > 85 || metrics.WaitCount > 1000 {
		pressure.PressureLevel = "high"
		pressure.Advice = "考虑扩容并优化查询"
	} else if usagePercent > 70 {
		pressure.PressureLevel = "medium"
		pressure.Advice = "监控中，暂无紧急操作"
	} else {
		pressure.PressureLevel = "low"
		pressure.Advice = "连接池状态良好"
	}

	return pressure
}

func (o *EnhancedConnectionPoolOptimizer) GetCurrentConfig() *EnhancedPoolConfig {
	o.mu.RLock()
	defer o.mu.RUnlock()

	configCopy := *o.currentConfig
	return &configCopy
}

func (o *EnhancedConnectionPoolOptimizer) SetConfig(cfg *EnhancedPoolConfig) error {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldConfig := *o.currentConfig
	o.currentConfig = cfg

	if err := o.applyConfig(cfg); err != nil {
		o.currentConfig = &oldConfig
		return err
	}

	o.recordTuning(&TuningRecord{
		Timestamp: time.Now(),
		OldConfig: &oldConfig,
		NewConfig: cfg,
		Reason:    "手动配置更新",
	})

	return nil
}

func (o *EnhancedConnectionPoolOptimizer) EnableAutoTuning(enabled bool) {
	o.mu.Lock()
	defer o.mu.Unlock()
	o.autoTuningEnabled = enabled
}

func (o *EnhancedConnectionPoolOptimizer) IsAutoTuningEnabled() bool {
	o.mu.RLock()
	defer o.mu.RUnlock()
	return o.autoTuningEnabled
}

func (o *EnhancedConnectionPoolOptimizer) GetHealthStatus() *PoolHealthStatus {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		return &PoolHealthStatus{
			IsHealthy: false,
			Score:     0,
			Issues:    []string{fmt.Sprintf("获取指标失败: %v", err)},
		}
	}
	return o.evaluateHealth(metrics)
}

func (o *EnhancedConnectionPoolOptimizer) GetConnectionPressure() *ConnectionPressure {
	return o.calculatePressure()
}

func (o *EnhancedConnectionPoolOptimizer) EmergencyExpand() {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldConfig := *o.currentConfig

	o.currentConfig.MaxOpenConns = oldConfig.MaxOpenConns * 2
	o.currentConfig.MaxIdleConns = oldConfig.MaxIdleConns * 2

	o.applyConfig(o.currentConfig)

	o.recordTuning(&TuningRecord{
		Timestamp:     time.Now(),
		OldConfig:     &oldConfig,
		NewConfig:     o.currentConfig,
		Reason:        "紧急扩容",
		MetricsBefore: nil,
		MetricsAfter:  nil,
	})

	log.Printf("[POOL_EMERGENCY] 紧急扩容完成: MaxOpen=%d, MaxIdle=%d",
		o.currentConfig.MaxOpenConns, o.currentConfig.MaxIdleConns)
}

func (o *EnhancedConnectionPoolOptimizer) EmergencyShrink() {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldConfig := *o.currentConfig

	newMaxOpen := oldConfig.MaxOpenConns / 2
	if newMaxOpen < 20 {
		newMaxOpen = 20
	}
	o.currentConfig.MaxOpenConns = newMaxOpen

	newMaxIdle := newMaxOpen / 2
	if newMaxIdle < 5 {
		newMaxIdle = 5
	}
	o.currentConfig.MaxIdleConns = newMaxIdle

	o.applyConfig(o.currentConfig)

	o.recordTuning(&TuningRecord{
		Timestamp: time.Now(),
		OldConfig: &oldConfig,
		NewConfig: o.currentConfig,
		Reason:    "紧急收缩",
	})

	log.Printf("[POOL_EMERGENCY] 紧急收缩完成: MaxOpen=%d, MaxIdle=%d",
		o.currentConfig.MaxOpenConns, o.currentConfig.MaxIdleConns)
}

func (o *EnhancedConnectionPoolOptimizer) WarmUpConnections(count int) error {
	sqlDB, err := o.db.DB()
	if err != nil {
		return err
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	log.Printf("[POOL_WARMUP] 开始预热 %d 个连接", count)

	for i := 0; i < count; i++ {
		conn, err := sqlDB.Conn(ctx)
		if err != nil {
			log.Printf("[POOL_WARMUP] 预热连接 %d 失败: %v", i, err)
			continue
		}

		var result int
		if err := conn.QueryRowContext(ctx, "SELECT 1").Scan(&result); err != nil {
			log.Printf("[POOL_WARMUP] 测试连接 %d 失败: %v", i, err)
		}

		conn.Close()
	}

	log.Printf("[POOL_WARMUP] 预热完成")
	return nil
}

func (o *EnhancedConnectionPoolOptimizer) ValidateConfig(cfg *EnhancedPoolConfig) error {
	if cfg.MaxOpenConns <= 0 {
		return fmt.Errorf("MaxOpenConns 必须大于0")
	}
	if cfg.MaxIdleConns <= 0 {
		return fmt.Errorf("MaxIdleConns 必须大于0")
	}
	if cfg.MaxIdleConns > cfg.MaxOpenConns {
		return fmt.Errorf("MaxIdleConns 不能超过 MaxOpenConns")
	}
	if cfg.MinIdleConns < 0 {
		return fmt.Errorf("MinIdleConns 不能为负数")
	}
	if cfg.MinIdleConns > cfg.MaxIdleConns {
		return fmt.Errorf("MinIdleConns 不能超过 MaxIdleConns")
	}
	if cfg.ConnMaxLifetime <= 0 {
		return fmt.Errorf("ConnMaxLifetime 必须大于0")
	}
	if cfg.ConnMaxIdleTime <= 0 {
		return fmt.Errorf("ConnMaxIdleTime 必须大于0")
	}
	return nil
}

type PoolMetricsCollector struct {
	db              *gorm.DB
	collectionInterval time.Duration
	metricsHistory     []PoolMetricsSnapshot
	maxHistorySize      int
	mu                 sync.RWMutex
	stopCh             chan struct{}
}

type PoolMetricsSnapshot struct {
	Timestamp       time.Time
	TotalConnections int
	ActiveConnections int
	IdleConnections   int
	WaitCount        int64
	MaxLifetimeClosed int64
	MaxIdleClosed    int64
}

func NewPoolMetricsCollector(db *gorm.DB) *PoolMetricsCollector {
	return &PoolMetricsCollector{
		db:                  db,
		collectionInterval: 1 * time.Minute,
		metricsHistory:      make([]PoolMetricsSnapshot, 0),
		maxHistorySize:      1440,
		stopCh:              make(chan struct{}),
	}
}

func (c *PoolMetricsCollector) Start() {
	go c.collectLoop()
}

func (c *PoolMetricsCollector) Stop() {
	close(c.stopCh)
}

func (c *PoolMetricsCollector) collectLoop() {
	ticker := time.NewTicker(c.collectionInterval)
	defer ticker.Stop()

	for {
		select {
		case <-c.stopCh:
			return
		case <-ticker.C:
			c.collect()
		}
	}
}

func (c *PoolMetricsCollector) collect() {
	metrics, err := GetConnectionPoolMetrics()
	if err != nil {
		return
	}

	snapshot := PoolMetricsSnapshot{
		Timestamp:         time.Now(),
		TotalConnections:  metrics.TotalConnections,
		ActiveConnections: metrics.ActiveConnections,
		IdleConnections:   metrics.IdleConnections,
		WaitCount:         metrics.WaitCount,
		MaxLifetimeClosed: metrics.MaxLifetimeClosed,
		MaxIdleClosed:     metrics.MaxIdleClosed,
	}

	c.mu.Lock()
	c.metricsHistory = append(c.metricsHistory, snapshot)
	if len(c.metricsHistory) > c.maxHistorySize {
		c.metricsHistory = c.metricsHistory[1:]
	}
	c.mu.Unlock()
}

func (c *PoolMetricsCollector) GetHistory(limit int) []PoolMetricsSnapshot {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if limit <= 0 || limit > len(c.metricsHistory) {
		limit = len(c.metricsHistory)
	}

	history := make([]PoolMetricsSnapshot, limit)
	copy(history, c.metricsHistory[len(c.metricsHistory)-limit:])
	return history
}

func (c *PoolMetricsCollector) GetAverageUsage() float64 {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if len(c.metricsHistory) == 0 {
		return 0
	}

	var totalUsage float64
	for _, snapshot := range c.metricsHistory {
		if snapshot.TotalConnections > 0 {
			usage := float64(snapshot.ActiveConnections) / float64(snapshot.TotalConnections) * 100
			totalUsage += usage
		}
	}

	return totalUsage / float64(len(c.metricsHistory))
}

type AdaptivePoolConfig struct {
	BaseMaxOpen     int
	BaseMaxIdle     int
	MinConnections  int
	MaxConnections  int
	ScaleFactor     float64
	ScaleThreshold  float64
}

func (o *EnhancedConnectionPoolOptimizer) AdaptToWorkload(ctx context.Context) error {
	collector := NewPoolMetricsCollector(o.db)
	avgUsage := collector.GetAverageUsage()

	if avgUsage > 80 {
		return o.scaleUp()
	} else if avgUsage < 20 {
		return o.scaleDown()
	}

	return nil
}

func (o *EnhancedConnectionPoolOptimizer) scaleUp() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldConfig := *o.currentConfig

	newMax := int(float64(o.currentConfig.MaxOpenConns) * 1.5)
	if newMax > 500 {
		newMax = 500
	}
	o.currentConfig.MaxOpenConns = newMax
	o.currentConfig.MaxIdleConns = newMax / 2

	o.applyConfig(o.currentConfig)

	o.recordTuning(&TuningRecord{
		Timestamp: time.Now(),
		OldConfig: &oldConfig,
		NewConfig: o.currentConfig,
		Reason:    "根据负载扩容",
	})

	log.Printf("[POOL_ADAPT] 扩容: MaxOpen=%d", o.currentConfig.MaxOpenConns)
	return nil
}

func (o *EnhancedConnectionPoolOptimizer) scaleDown() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	oldConfig := *o.currentConfig

	newMax := int(float64(o.currentConfig.MaxOpenConns) * 0.7)
	if newMax < 20 {
		newMax = 20
	}
	o.currentConfig.MaxOpenConns = newMax
	o.currentConfig.MaxIdleConns = newMax / 2

	o.applyConfig(o.currentConfig)

	o.recordTuning(&TuningRecord{
		Timestamp: time.Now(),
		OldConfig: &oldConfig,
		NewConfig: o.currentConfig,
		Reason:    "根据负载收缩",
	})

	log.Printf("[POOL_ADAPT] 收缩: MaxOpen=%d", o.currentConfig.MaxOpenConns)
	return nil
}
