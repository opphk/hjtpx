package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type ColdStartMetrics struct {
	ColdStartCount      atomic.Int64
	WarmStartCount      atomic.Int64
	AvgColdStartTime    atomic.Int64
	AvgWarmStartTime    atomic.Int64
	MaxColdStartTime    atomic.Int64
	MinColdStartTime    atomic.Int64
	LastColdStart      time.Time
	InitDuration       atomic.Int64
}

type OptimizationStrategy int

const (
	StrategyPreWarming OptimizationStrategy = iota
	StrategyLazyLoading
	StrategyConnectionPooling
	StrategyDependencyOptimization
	StrategyRuntimeOptimization
	StrategyMultiLayerCache
)

type OptimizationConfig struct {
	Strategy           OptimizationStrategy
	PreWarmingEnabled  bool
	PreWarmingInterval time.Duration
	PreWarmingCount    int
	CacheEnabled       bool
	CacheSize          int
	PoolSize           int
	InitScriptEnabled  bool
	InitScript        string
}

type PreWarmingConfig struct {
	Enabled          bool
	Interval         time.Duration
	MinInstances     int
	TargetInstances   int
	WarmupRequests   int
	Schedule         string
}

type LazyLoadingConfig struct {
	Enabled            bool
	DelayThreshold     time.Duration
	PreloadModules     []string
	OnDemandImport     bool
}

type ConnectionPoolConfig struct {
	Enabled             bool
	MaxConnections      int
	MinConnections      int
	AcquireTimeout      time.Duration
	IdleTimeout         time.Duration
	MaxIdleTime         time.Duration
	HealthCheckInterval time.Duration
}

type DependencyOptimizationConfig struct {
	Enabled          bool
	PruneUnused      bool
	Minify           bool
	TreeShaking      bool
	BundleSizeLimit  int64
}

type RuntimeOptimizationConfig struct {
	Enabled         bool
	UseARM64        bool
	OptimizeMemory  bool
	StreamingInit   bool
	ContainerReuse  bool
	SnapshotEnabled bool
}

type ColdStartOptimizer struct {
	manager        *ServerlessManager
	configs        map[string]*OptimizationConfig
	preWarmers     map[string]*PreWarmer
	connPools      map[string]*ConnectionPool
	lazyLoaders    map[string]*LazyLoader
	metrics        *ColdStartMetrics
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	enabled        atomic.Bool
}

type PreWarmer struct {
	functionName   string
	config         *PreWarmingConfig
	running        atomic.Bool
	stopChan       chan struct{}
	 warmedInstances atomic.Int32
}

type ConnectionPool struct {
	functionName   string
	config         *ConnectionPoolConfig
	connections    chan *PooledConnection
	mu             sync.RWMutex
	activeCount    atomic.Int32
	idleCount      atomic.Int32
}

type PooledConnection struct {
	ID        string
	CreatedAt time.Time
	LastUsed  time.Time
	Active    atomic.Bool
}

type LazyLoader struct {
	functionName   string
	config         *LazyLoadingConfig
	loadedModules  map[string]bool
	mu             sync.RWMutex
}

type OptimizationResult struct {
	FunctionName       string                 `json:"function_name"`
	BeforeMetrics      *ColdStartSnapshot     `json:"before_metrics"`
	AfterMetrics       *ColdStartSnapshot     `json:"after_metrics"`
	ImprovementPercent float64                `json:"improvement_percent"`
	Recommendations    []string               `json:"recommendations"`
	AppliedStrategies  []OptimizationStrategy `json:"applied_strategies"`
}

type ColdStartSnapshot struct {
	AvgColdStartTime  int64     `json:"avg_cold_start_time_ns"`
	MaxColdStartTime  int64     `json:"max_cold_start_time_ns"`
	MinColdStartTime  int64     `json:"min_cold_start_time_ns"`
	ColdStartCount    int64     `json:"cold_start_count"`
	WarmStartCount    int64     `json:"warm_start_count"`
	InitDuration      int64     `json:"init_duration_ns"`
	Timestamp         time.Time `json:"timestamp"`
}

type CodeAnalysis struct {
	ModuleSize       map[string]int64 `json:"module_size"`
	UnusedImports    []string        `json:"unused_imports"`
	InitTimeEstimate int64           `json:"init_time_estimate_ns"`
	BundleSize       int64            `json:"bundle_size"`
	DependencyDepth  int              `json:"dependency_depth"`
}

func NewColdStartOptimizer(manager *ServerlessManager) *ColdStartOptimizer {
	ctx, cancel := context.WithCancel(context.Background())
	
	optimizer := &ColdStartOptimizer{
		manager:       manager,
		configs:       make(map[string]*OptimizationConfig),
		preWarmers:    make(map[string]*PreWarmer),
		connPools:     make(map[string]*ConnectionPool),
		lazyLoaders:   make(map[string]*LazyLoader),
		metrics:       &ColdStartMetrics{},
		ctx:           ctx,
		cancel:        cancel,
	}
	
	optimizer.enabled.Store(true)
	
	return optimizer
}

func (o *ColdStartOptimizer) Configure(functionName string, config *OptimizationConfig) error {
	if functionName == "" {
		return fmt.Errorf("function name is required")
	}
	
	if config == nil {
		return fmt.Errorf("config is required")
	}
	
	o.mu.Lock()
	defer o.mu.Unlock()
	
	o.configs[functionName] = config
	
	if config.PreWarmingEnabled {
		o.setupPreWarming(functionName, &PreWarmingConfig{
			Enabled:          true,
			Interval:         config.PreWarmingInterval,
			MinInstances:     1,
			TargetInstances:  config.PreWarmingCount,
			WarmupRequests:   1,
		})
	}
	
	if config.Strategy == StrategyConnectionPooling {
		o.setupConnectionPool(functionName, &ConnectionPoolConfig{
			Enabled:           true,
			MaxConnections:    config.PoolSize,
			MinConnections:    config.PoolSize / 2,
			AcquireTimeout:    5 * time.Second,
			IdleTimeout:       30 * time.Second,
			MaxIdleTime:       5 * time.Minute,
		})
	}
	
	return nil
}

func (o *ColdStartOptimizer) setupPreWarming(functionName string, config *PreWarmingConfig) {
	preWarmer := &PreWarmer{
		functionName: functionName,
		config:       config,
		stopChan:     make(chan struct{}),
	}
	
	o.preWarmers[functionName] = preWarmer
	
	if config.Enabled {
		go preWarmer.start()
	}
}

func (o *ColdStartOptimizer) setupConnectionPool(functionName string, config *ConnectionPoolConfig) {
	pool := &ConnectionPool{
		functionName: functionName,
		config:       config,
		connections:  make(chan *PooledConnection, config.MaxConnections),
	}
	
	for i := 0; i < config.MinConnections; i++ {
		conn := &PooledConnection{
			ID:        fmt.Sprintf("%s-%d", functionName, i),
			CreatedAt: time.Now(),
			LastUsed:  time.Now(),
		}
		conn.Active.Store(false)
		pool.connections <- conn
		pool.idleCount.Add(1)
	}
	
	o.connPools[functionName] = pool
}

func (w *PreWarmer) start() {
	if !w.running.CompareAndSwap(false, true) {
		return
	}
	
	ticker := time.NewTicker(w.config.Interval)
	defer ticker.Stop()
	defer w.running.Store(false)
	
	for {
		select {
		case <-w.stopChan:
			return
		case <-ticker.C:
			w.warmup()
		}
	}
}

func (w *PreWarmer) warmup() {
	currentInstances := w.warmedInstances.Load()
	
	if currentInstances < int32(w.config.TargetInstances) {
		w.warmedInstances.Add(1)
	}
}

func (w *PreWarmer) stop() {
	close(w.stopChan)
}

func (o *ColdStartOptimizer) OptimizeColdStart(functionName string) (*OptimizationResult, error) {
	before := o.captureSnapshot()
	
	codeAnalysis, err := o.AnalyzeCode(functionName)
	if err != nil {
		return nil, fmt.Errorf("code analysis failed: %w", err)
	}
	
	recommendations := o.generateRecommendations(codeAnalysis)
	
	for _, strategy := range recommendations {
		if err := o.applyStrategy(functionName, strategy); err != nil {
			return nil, fmt.Errorf("failed to apply strategy %v: %w", strategy, err)
		}
	}
	
	after := o.captureSnapshot()
	
	improvement := calculateImprovement(before, after)
	
	var recommendationStrings []string
	for _, r := range recommendations {
		recommendationStrings = append(recommendationStrings, fmt.Sprintf("Strategy: %v", r))
	}
	
	return &OptimizationResult{
		FunctionName:       functionName,
		BeforeMetrics:     before,
		AfterMetrics:      after,
		ImprovementPercent: improvement,
		Recommendations:    recommendationStrings,
		AppliedStrategies: recommendations,
	}, nil
}

func (o *ColdStartOptimizer) AnalyzeCode(functionName string) (*CodeAnalysis, error) {
	analysis := &CodeAnalysis{
		ModuleSize:      make(map[string]int64),
		UnusedImports:   []string{},
		DependencyDepth: 3,
	}
	
	analysis.InitTimeEstimate = 100 * 1e6
	
	analysis.BundleSize = int64(1 * 1024 * 1024)
	
	analysis.UnusedImports = []string{"unused/pkg1", "unused/pkg2"}
	
	return analysis, nil
}

func (o *ColdStartOptimizer) generateRecommendations(analysis *CodeAnalysis) []OptimizationStrategy {
	strategies := []OptimizationStrategy{}
	
	if len(analysis.UnusedImports) > 0 {
		strategies = append(strategies, StrategyDependencyOptimization)
	}
	
	if analysis.InitTimeEstimate > 50*1e6 {
		strategies = append(strategies, StrategyPreWarming)
		strategies = append(strategies, StrategyLazyLoading)
	}
	
	if analysis.BundleSize > 5*1024*1024 {
		strategies = append(strategies, StrategyDependencyOptimization)
	}
	
	strategies = append(strategies, StrategyRuntimeOptimization)
	
	return strategies
}

func (o *ColdStartOptimizer) applyStrategy(functionName string, strategy OptimizationStrategy) error {
	switch strategy {
	case StrategyPreWarming:
		o.mu.Lock()
		defer o.mu.Unlock()
		
		if preWarmer, exists := o.preWarmers[functionName]; exists {
			preWarmer.config.Enabled = true
			go preWarmer.start()
		}
		
	case StrategyLazyLoading:
		o.mu.Lock()
		defer o.mu.Unlock()
		
		o.lazyLoaders[functionName] = &LazyLoader{
			functionName: functionName,
			config: &LazyLoadingConfig{
				Enabled:        true,
				DelayThreshold: 10 * time.Millisecond,
			},
			loadedModules: make(map[string]bool),
		}
		
	case StrategyConnectionPooling:
		o.mu.Lock()
		defer o.mu.Unlock()
		
		if _, exists := o.connPools[functionName]; !exists {
			o.setupConnectionPool(functionName, &ConnectionPoolConfig{
				Enabled:        true,
				MaxConnections: 10,
				MinConnections: 2,
			})
		}
		
	case StrategyRuntimeOptimization:
		if err := o.manager.SetARM64(functionName, true); err != nil {
			return err
		}
	}
	
	return nil
}

func (o *ColdStartOptimizer) captureSnapshot() *ColdStartSnapshot {
	return &ColdStartSnapshot{
		AvgColdStartTime:  o.metrics.AvgColdStartTime.Load(),
		MaxColdStartTime:  o.metrics.MaxColdStartTime.Load(),
		MinColdStartTime:  o.metrics.MinColdStartTime.Load(),
		ColdStartCount:    o.metrics.ColdStartCount.Load(),
		WarmStartCount:     o.metrics.WarmStartCount.Load(),
		InitDuration:       o.metrics.InitDuration.Load(),
		Timestamp:          time.Now(),
	}
}

func calculateImprovement(before, after *ColdStartSnapshot) float64 {
	if before.AvgColdStartTime == 0 {
		return 0
	}
	
	improvement := float64(before.AvgColdStartTime-after.AvgColdStartTime) / float64(before.AvgColdStartTime) * 100
	
	return improvement
}

func (o *ColdStartOptimizer) RecordColdStart(duration time.Duration) {
	o.metrics.ColdStartCount.Add(1)
	o.metrics.LastColdStart = time.Now()
	
	current := o.metrics.AvgColdStartTime.Load()
	count := o.metrics.ColdStartCount.Load()
	if count > 0 {
		o.metrics.AvgColdStartTime.Store((current*(count-1) + duration.Nanoseconds()) / count)
	}
	
	if duration.Nanoseconds() > o.metrics.MaxColdStartTime.Load() {
		o.metrics.MaxColdStartTime.Store(duration.Nanoseconds())
	}
	
	if o.metrics.MinColdStartTime.Load() == 0 || duration.Nanoseconds() < o.metrics.MinColdStartTime.Load() {
		o.metrics.MinColdStartTime.Store(duration.Nanoseconds())
	}
}

func (o *ColdStartOptimizer) RecordWarmStart(duration time.Duration) {
	o.metrics.WarmStartCount.Add(1)
	
	current := o.metrics.AvgWarmStartTime.Load()
	count := o.metrics.WarmStartCount.Load()
	if count > 0 {
		o.metrics.AvgWarmStartTime.Store((current*(count-1) + duration.Nanoseconds()) / count)
	}
}

func (o *ColdStartOptimizer) RecordInitDuration(duration time.Duration) {
	o.metrics.InitDuration.Store(duration.Nanoseconds())
}

func (o *ColdStartOptimizer) GetMetrics() map[string]interface{} {
	coldCount := o.metrics.ColdStartCount.Load()
	avgCold := o.metrics.AvgColdStartTime.Load()
	avgWarm := o.metrics.AvgWarmStartTime.Load()
	
	var improvement float64
	if avgWarm > 0 {
		improvement = float64(avgCold-avgWarm) / float64(avgWarm) * 100
	}
	
	return map[string]interface{}{
		"cold_start_count":      coldCount,
		"warm_start_count":     o.metrics.WarmStartCount.Load(),
		"avg_cold_start_ms":    float64(avgCold) / 1e6,
		"avg_warm_start_ms":    float64(avgWarm) / 1e6,
		"max_cold_start_ms":    float64(o.metrics.MaxColdStartTime.Load()) / 1e6,
		"min_cold_start_ms":    float64(o.metrics.MinColdStartTime.Load()) / 1e6,
		"improvement_percent":  improvement,
		"last_cold_start":      o.metrics.LastColdStart,
	}
}

func (o *ColdStartOptimizer) AcquireConnection(functionName string) (*PooledConnection, error) {
	o.mu.RLock()
	pool, exists := o.connPools[functionName]
	o.mu.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("connection pool not found for function %s", functionName)
	}
	
	select {
	case conn := <-pool.connections:
		conn.LastUsed = time.Now()
		conn.Active.Store(true)
		pool.activeCount.Add(1)
		pool.idleCount.Add(-1)
		return conn, nil
	case <-time.After(pool.config.AcquireTimeout):
		return nil, fmt.Errorf("acquire connection timeout")
	}
}

func (o *ColdStartOptimizer) ReleaseConnection(functionName string, conn *PooledConnection) error {
	o.mu.RLock()
	pool, exists := o.connPools[functionName]
	o.mu.RUnlock()
	
	if !exists {
		return fmt.Errorf("connection pool not found for function %s", functionName)
	}
	
	conn.Active.Store(false)
	pool.activeCount.Add(-1)
	pool.idleCount.Add(1)
	
	select {
	case pool.connections <- conn:
		return nil
	default:
		return fmt.Errorf("connection pool is full")
	}
}

func (o *ColdStartOptimizer) Enable() {
	o.enabled.Store(true)
}

func (o *ColdStartOptimizer) Disable() {
	o.enabled.Store(false)
}

func (o *ColdStartOptimizer) IsEnabled() bool {
	return o.enabled.Load()
}

func (o *ColdStartOptimizer) GetOptimizationConfig(functionName string) (*OptimizationConfig, error) {
	o.mu.RLock()
	defer o.mu.RUnlock()
	
	config, exists := o.configs[functionName]
	if !exists {
		return nil, fmt.Errorf("no optimization config for function %s", functionName)
	}
	
	return config, nil
}

func (o *ColdStartOptimizer) Stop() {
	o.cancel()
	
	o.mu.Lock()
	defer o.mu.Unlock()
	
	for _, preWarmer := range o.preWarmers {
		preWarmer.stop()
	}
}

func OptimizeDependencyTree(imports []string) ([]string, error) {
	unused := []string{}
	
	for _, imp := range imports {
		if !isUsed(imp) {
			unused = append(unused, imp)
		}
	}
	
	return unused, nil
}

func isUsed(importPath string) bool {
	usedPackages := map[string]bool{
		"context":      true,
		"fmt":          true,
		"time":         true,
		"sync":         true,
	}
	
	return usedPackages[importPath]
}

func EstimateInitTime(bundleSize int64, dependencyCount int) time.Duration {
	baseTime := 50 * time.Millisecond
	
	sizeFactor := float64(bundleSize) / (1 * 1024 * 1024)
	
	depFactor := time.Duration(float64(dependencyCount) * 5 * float64(time.Millisecond))
	
	total := baseTime + time.Duration(sizeFactor*100*float64(time.Millisecond)) + depFactor
	
	if total > 500*time.Millisecond {
		total = 500 * time.Millisecond
	}
	
	return total
}

func MinifyCode(code []byte) ([]byte, error) {
	return code, nil
}

func GenerateInitScript(functionName string, config *FunctionConfig) (string, error) {
	script := fmt.Sprintf(`#!/bin/sh
export FUNCTION_NAME=%s
export RUNTIME=%s
export MEMORY=%d
export TIMEOUT=%d
`, functionName, config.Runtime, config.Memory, config.Timeout)
	
	return script, nil
}

func CreateSnapshot() *ColdStartSnapshot {
	return &ColdStartSnapshot{
		AvgColdStartTime: 0,
		MaxColdStartTime: 0,
		MinColdStartTime: 0,
		ColdStartCount:   0,
		WarmStartCount:   0,
		InitDuration:     0,
		Timestamp:        time.Now(),
	}
}

func AnalyzeDependencyGraph(imports []string) *DependencyGraph {
	graph := &DependencyGraph{
		Nodes: make([]DependencyNode, len(imports)),
		Edges: []DependencyEdge{},
	}
	
	for i, imp := range imports {
		graph.Nodes[i] = DependencyNode{
			Name:     imp,
			Children: []string{},
		}
	}
	
	return graph
}

type DependencyGraph struct {
	Nodes []DependencyNode   `json:"nodes"`
	Edges []DependencyEdge   `json:"edges"`
}

type DependencyNode struct {
	Name     string   `json:"name"`
	Children []string `json:"children"`
}

type DependencyEdge struct {
	From string `json:"from"`
	To   string `json:"to"`
}
