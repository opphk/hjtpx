package cdn

import (
	"context"
	"fmt"
	"log"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type CDNProvider interface {
	Name() string
	Purge(ctx context.Context, urls []string) error
	Invalidate(ctx context.Context, paths []string) error
	FetchMetrics(ctx context.Context) (*CDNMetrics, error)
}

type CDNManager struct {
	providers    map[string]CDNProvider
	defaultCDN  string
	mu          sync.RWMutex
	metrics     *CDNMetricsCollector
}

type CDNMetricsCollector struct {
	TotalRequests     int64
	CacheHits        int64
	CacheMisses      int64
	BandwidthBytes   int64
	LatencyMs        int64
	mu               sync.Mutex
}

type CDNMetrics struct {
	CacheHitRate   float64   `json:"cache_hit_rate"`
	TotalRequests  int64     `json:"total_requests"`
	BandwidthMB    float64   `json:"bandwidth_mb"`
	AvgLatencyMs   float64   `json:"avg_latency_ms"`
	P95LatencyMs   float64   `json:"p95_latency_ms"`
	P99LatencyMs   float64   `json:"p99_latency_ms"`
	LastUpdated    time.Time `json:"last_updated"`
}

func NewCDNManager(cfg *config.CDNConfig) (*CDNManager, error) {
	manager := &CDNManager{
		providers: make(map[string]CDNProvider),
		metrics:   &CDNMetricsCollector{},
	}

	if cfg.Provider != "" {
		manager.providers[cfg.Provider] = createProvider(cfg.Provider, cfg)
		manager.defaultCDN = cfg.Provider
	}

	for _, endpoint := range cfg.Endpoints {
		provider := &MockCDNProvider{name: endpoint.Name, endpoint: endpoint.URL}
		manager.providers[endpoint.Name] = provider
	}

	return manager, nil
}

func createProvider(name string, cfg *config.CDNConfig) CDNProvider {
	switch name {
	case "cloudflare":
		return &CloudflareProvider{config: cfg}
	case "fastly":
		return &FastlyProvider{config: cfg}
	case "aws_cloudfront":
		return &AWSCloudFrontProvider{config: cfg}
	default:
		return &MockCDNProvider{name: name}
	}
}

func (m *CDNManager) GetProvider(name string) (CDNProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	provider, ok := m.providers[name]
	return provider, ok
}

func (m *CDNManager) GetDefaultProvider() (CDNProvider, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	provider, ok := m.providers[m.defaultCDN]
	return provider, ok
}

func (m *CDNManager) Purge(ctx context.Context, urls []string) error {
	provider, ok := m.GetDefaultProvider()
	if !ok {
		return fmt.Errorf("no default CDN provider configured")
	}
	return provider.Purge(ctx, urls)
}

func (m *CDNManager) PurgeAll(ctx context.Context, urls []string) error {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var lastErr error
	for _, provider := range m.providers {
		if err := provider.Purge(ctx, urls); err != nil {
			lastErr = err
			log.Printf("Failed to purge on provider %s: %v", provider.Name(), err)
		}
	}
	return lastErr
}

func (m *CDNManager) Invalidate(ctx context.Context, paths []string) error {
	provider, ok := m.GetDefaultProvider()
	if !ok {
		return fmt.Errorf("no default CDN provider configured")
	}
	return provider.Invalidate(ctx, paths)
}

func (m *CDNManager) GetMetrics(ctx context.Context) (*CDNMetrics, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var totalRequests, cacheHits, cacheMisses int64
	var totalBandwidth int64
	var totalLatency int64
	var count int64

	for _, provider := range m.providers {
		metrics, err := provider.FetchMetrics(ctx)
		if err != nil {
			continue
		}

		totalRequests += metrics.TotalRequests
		cacheHits += int64(float64(metrics.TotalRequests) * metrics.CacheHitRate / 100)
		cacheMisses += int64(float64(metrics.TotalRequests) * (100 - metrics.CacheHitRate) / 100)
		totalBandwidth += int64(metrics.BandwidthMB * 1024 * 1024)
		totalLatency += int64(metrics.AvgLatencyMs)
		count++
	}

	hitRate := 0.0
	if totalRequests > 0 {
		hitRate = float64(cacheHits) / float64(totalRequests) * 100
	}

	avgLatency := 0.0
	if count > 0 {
		avgLatency = float64(totalLatency) / float64(count)
	}

	return &CDNMetrics{
		CacheHitRate:  hitRate,
		TotalRequests: totalRequests,
		BandwidthMB:    float64(totalBandwidth) / (1024 * 1024),
		AvgLatencyMs:   avgLatency,
		LastUpdated:    time.Now(),
	}, nil
}

func (m *CDNManager) RecordRequest() {
	atomic.AddInt64(&m.metrics.TotalRequests, 1)
}

func (m *CDNManager) RecordCacheHit() {
	atomic.AddInt64(&m.metrics.CacheHits, 1)
}

func (m *CDNManager) RecordCacheMiss() {
	atomic.AddInt64(&m.metrics.CacheMisses, 1)
}

func (m *CDNManager) RecordBandwidth(bytes int64) {
	atomic.AddInt64(&m.metrics.BandwidthBytes, bytes)
}

func (m *CDNManager) RecordLatency(ms int64) {
	atomic.AddInt64(&m.metrics.LatencyMs, m.getPercentileLatency(95))
}

func (m *CDNManager) getPercentileLatency(percentile int) int64 {
	return atomic.LoadInt64(&m.metrics.LatencyMs)
}

type CloudflareProvider struct {
	config *config.CDNConfig
}

func (p *CloudflareProvider) Name() string {
	return "cloudflare"
}

func (p *CloudflareProvider) Purge(ctx context.Context, urls []string) error {
	log.Printf("[Cloudflare] Purging %d URLs", len(urls))
	return nil
}

func (p *CloudflareProvider) Invalidate(ctx context.Context, paths []string) error {
	log.Printf("[Cloudflare] Invalidating %d paths", len(paths))
	return nil
}

func (p *CloudflareProvider) FetchMetrics(ctx context.Context) (*CDNMetrics, error) {
	return &CDNMetrics{
		CacheHitRate:  85.5,
		TotalRequests: 1000000,
		BandwidthMB:   5000.0,
		AvgLatencyMs:  45.0,
		P95LatencyMs:  120.0,
		P99LatencyMs:  200.0,
		LastUpdated:   time.Now(),
	}, nil
}

type FastlyProvider struct {
	config *config.CDNConfig
}

func (p *FastlyProvider) Name() string {
	return "fastly"
}

func (p *FastlyProvider) Purge(ctx context.Context, urls []string) error {
	log.Printf("[Fastly] Purging %d URLs", len(urls))
	return nil
}

func (p *FastlyProvider) Invalidate(ctx context.Context, paths []string) error {
	log.Printf("[Fastly] Invalidating %d paths", len(paths))
	return nil
}

func (p *FastlyProvider) FetchMetrics(ctx context.Context) (*CDNMetrics, error) {
	return &CDNMetrics{
		CacheHitRate:  88.2,
		TotalRequests: 800000,
		BandwidthMB:   4200.0,
		AvgLatencyMs:  40.0,
		P95LatencyMs:  110.0,
		P99LatencyMs:  180.0,
		LastUpdated:   time.Now(),
	}, nil
}

type AWSCloudFrontProvider struct {
	config *config.CDNConfig
}

func (p *AWSCloudFrontProvider) Name() string {
	return "aws_cloudfront"
}

func (p *AWSCloudFrontProvider) Purge(ctx context.Context, urls []string) error {
	log.Printf("[AWS CloudFront] Purging %d URLs", len(urls))
	return nil
}

func (p *AWSCloudFrontProvider) Invalidate(ctx context.Context, paths []string) error {
	log.Printf("[AWS CloudFront] Invalidating %d paths", len(paths))
	return nil
}

func (p *AWSCloudFrontProvider) FetchMetrics(ctx context.Context) (*CDNMetrics, error) {
	return &CDNMetrics{
		CacheHitRate:  82.0,
		TotalRequests: 1200000,
		BandwidthMB:   6000.0,
		AvgLatencyMs:  50.0,
		P95LatencyMs:  130.0,
		P99LatencyMs:  220.0,
		LastUpdated:   time.Now(),
	}, nil
}

type MockCDNProvider struct {
	name     string
	endpoint string
}

func (p *MockCDNProvider) Name() string {
	return p.name
}

func (p *MockCDNProvider) Purge(ctx context.Context, urls []string) error {
	log.Printf("[MockCDN:%s] Purging %d URLs", p.name, len(urls))
	return nil
}

func (p *MockCDNProvider) Invalidate(ctx context.Context, paths []string) error {
	log.Printf("[MockCDN:%s] Invalidating %d paths", p.name, len(paths))
	return nil
}

func (p *MockCDNProvider) FetchMetrics(ctx context.Context) (*CDNMetrics, error) {
	return &CDNMetrics{
		CacheHitRate:  80.0,
		TotalRequests: 500000,
		BandwidthMB:   2500.0,
		AvgLatencyMs:  60.0,
		P95LatencyMs:  150.0,
		P99LatencyMs:  250.0,
		LastUpdated:   time.Now(),
	}, nil
}

type MultiRegionManager struct {
	regions     map[string]*Region
	strategy    string
	healthCheck *RegionHealthChecker
	mu          sync.RWMutex
}

type Region struct {
	Name         string
	ID           string
	Endpoint     string
	Priority     int
	Weight       int
	GeoTargeting bool
	Meta         map[string]string
	Healthy      bool
	LatencyMs    int64
	LoadPercent  float64
}

type RegionHealthChecker struct {
	manager  *MultiRegionManager
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

func NewMultiRegionManager(cfg *config.MultiRegionConfig) (*MultiRegionManager, error) {
	manager := &MultiRegionManager{
		regions:  make(map[string]*Region),
		strategy: "latency",
	}

	for _, regionCfg := range cfg.Regions {
		region := &Region{
			Name:         regionCfg.Name,
			ID:           regionCfg.ID,
			Endpoint:     regionCfg.Endpoint,
			Priority:     regionCfg.Priority,
			Weight:       regionCfg.Weight,
			GeoTargeting: regionCfg.GeoTargeting,
			Meta:         regionCfg.Meta,
			Healthy:      true,
		}
		manager.regions[region.ID] = region
	}

	manager.healthCheck = &RegionHealthChecker{
		manager:  manager,
		interval: 30 * time.Second,
		stopCh:   make(chan struct{}),
	}

	return manager, nil
}

func (m *MultiRegionManager) GetRegion(id string) (*Region, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	region, ok := m.regions[id]
	return region, ok
}

func (m *MultiRegionManager) GetAllRegions() []*Region {
	m.mu.RLock()
	defer m.mu.RUnlock()

	regions := make([]*Region, 0, len(m.regions))
	for _, region := range m.regions {
		regions = append(regions, region)
	}
	return regions
}

func (m *MultiRegionManager) GetHealthyRegions() []*Region {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var healthy []*Region
	for _, region := range m.regions {
		if region.Healthy {
			healthy = append(healthy, region)
		}
	}
	return healthy
}

func (m *MultiRegionManager) GetBestRegion(latencyThreshold int64) *Region {
	regions := m.GetHealthyRegions()
	if len(regions) == 0 {
		return nil
	}

	var best *Region
	var minLatency int64 = 1<<63 - 1

	for _, region := range regions {
		if region.LatencyMs > 0 && region.LatencyMs < minLatency && region.LatencyMs <= latencyThreshold {
			minLatency = region.LatencyMs
			best = region
		}
	}

	if best == nil {
		for _, region := range regions {
			if region.Priority > 0 && (best == nil || region.Priority > best.Priority) {
				best = region
			}
		}
	}

	return best
}

func (m *MultiRegionManager) UpdateRegionHealth(id string, healthy bool, latencyMs int64) {
	m.mu.Lock()
	defer m.mu.Unlock()

	if region, ok := m.regions[id]; ok {
		region.Healthy = healthy
		region.LatencyMs = latencyMs
	}
}

func (m *MultiRegionManager) StartHealthCheck() {
	m.healthCheck.Start()
}

func (m *MultiRegionManager) StopHealthCheck() {
	m.healthCheck.Stop()
}

func (hc *RegionHealthChecker) Start() {
	hc.wg.Add(1)
	go hc.checkLoop()
}

func (hc *RegionHealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *RegionHealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.check()
		}
	}
}

func (hc *RegionHealthChecker) check() {
	hc.manager.mu.RLock()
	defer hc.manager.mu.RUnlock()

	for _, region := range hc.manager.regions {
		_, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		start := time.Now()

		resp, err := http.Get(region.Endpoint + "/health")
		if err != nil || resp.StatusCode != http.StatusOK {
			hc.manager.UpdateRegionHealth(region.ID, false, 0)
			continue
		}
		resp.Body.Close()

		latency := time.Since(start).Milliseconds()
		hc.manager.UpdateRegionHealth(region.ID, true, latency)
		cancel()
	}
}

type SmartRouter struct {
	multiRegion *MultiRegionManager
	cdn        *CDNManager
	latencyThreshold int64
	mu         sync.RWMutex
}

func NewSmartRouter(cfg *config.SmartRoutingConfig, multiRegion *MultiRegionManager, cdn *CDNManager) *SmartRouter {
	return &SmartRouter{
		multiRegion:      multiRegion,
		cdn:              cdn,
		latencyThreshold: int64(cfg.LatencyThreshold),
	}
}

func (r *SmartRouter) GetBestEndpoint(latencyThreshold int64) string {
	if r.multiRegion == nil {
		return ""
	}

	region := r.multiRegion.GetBestRegion(latencyThreshold)
	if region != nil {
		return region.Endpoint
	}

	return ""
}

func (r *SmartRouter) RouteRequest(ctx context.Context, clientIP string) (*RouteResult, error) {
	latencyThreshold := r.latencyThreshold
	if latencyThreshold == 0 {
		latencyThreshold = 100
	}

	region := r.multiRegion.GetBestRegion(latencyThreshold)
	if region == nil {
		return nil, fmt.Errorf("no available region")
	}

	return &RouteResult{
		Region:       region.ID,
		Endpoint:     region.Endpoint,
		LatencyMs:    region.LatencyMs,
		CacheEnabled: true,
	}, nil
}

type RouteResult struct {
	Region       string
	Endpoint     string
	LatencyMs    int64
	CacheEnabled bool
	CDNUsed      bool
}

type EdgeNode struct {
	ID       string
	Name     string
	Region   string
	Endpoint string
	Capacity int
	Load     int
	Healthy  bool
}

type EdgeComputingManager struct {
	nodes map[string]*EdgeNode
	mu    sync.RWMutex
}

func NewEdgeComputingManager(cfg *config.EdgeComputingConfig) (*EdgeComputingManager, error) {
	manager := &EdgeComputingManager{
		nodes: make(map[string]*EdgeNode),
	}

	for _, nodeCfg := range cfg.Nodes {
		node := &EdgeNode{
			ID:       nodeCfg.ID,
			Name:     nodeCfg.Name,
			Region:   nodeCfg.Region,
			Endpoint: nodeCfg.Endpoint,
			Capacity: nodeCfg.Capacity,
			Load:     0,
			Healthy:  true,
		}
		manager.nodes[node.ID] = node
	}

	return manager, nil
}

func (m *EdgeComputingManager) GetNode(id string) (*EdgeNode, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	node, ok := m.nodes[id]
	return node, ok
}

func (m *EdgeComputingManager) GetAllNodes() []*EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	nodes := make([]*EdgeNode, 0, len(m.nodes))
	for _, node := range m.nodes {
		nodes = append(nodes, node)
	}
	return nodes
}

func (m *EdgeComputingManager) GetBestNode(region string) *EdgeNode {
	m.mu.RLock()
	defer m.mu.RUnlock()

	var best *EdgeNode
	var minLoad int = 1<<63 - 1

	for _, node := range m.nodes {
		if !node.Healthy {
			continue
		}
		if region != "" && node.Region != region {
			continue
		}
		if node.Load < minLoad {
			minLoad = node.Load
			best = node
		}
	}

	return best
}

func (m *EdgeComputingManager) ExecuteFunction(functionName string, params map[string]interface{}) error {
	node := m.GetBestNode("")
	if node == nil {
		return fmt.Errorf("no available edge node")
	}

	log.Printf("[Edge] Executing %s on node %s", functionName, node.Name)

	m.mu.Lock()
	node.Load++
	m.mu.Unlock()

	defer func() {
		m.mu.Lock()
		node.Load--
		m.mu.Unlock()
	}()

	return nil
}
