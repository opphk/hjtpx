package cdn

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

func TestCDNManagerCreation(t *testing.T) {
	cfg := &config.CDNConfig{
		Enabled:  true,
		Provider: "cloudflare",
	}

	manager, err := NewCDNManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create CDN manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	if manager.defaultCDN != "cloudflare" {
		t.Errorf("Expected default CDN 'cloudflare', got '%s'", manager.defaultCDN)
	}
}

func TestCDNManagerGetProvider(t *testing.T) {
	cfg := &config.CDNConfig{
		Enabled:  true,
		Provider: "fastly",
	}

	manager, _ := NewCDNManager(cfg)

	provider, ok := manager.GetProvider("fastly")
	if !ok {
		t.Error("Expected fastly provider to exist")
	}

	if provider.Name() != "fastly" {
		t.Errorf("Expected provider name 'fastly', got '%s'", provider.Name())
	}

	_, ok = manager.GetProvider("nonexistent")
	if ok {
		t.Error("Expected nonexistent provider to not exist")
	}
}

func TestCDNManagerPurge(t *testing.T) {
	cfg := &config.CDNConfig{
		Enabled:  true,
		Provider: "cloudflare",
	}

	manager, _ := NewCDNManager(cfg)

	ctx := context.Background()
	urls := []string{"https://example.com/image1.png", "https://example.com/image2.png"}

	err := manager.Purge(ctx, urls)
	if err != nil {
		t.Errorf("Failed to purge: %v", err)
	}
}

func TestCDNMetrics(t *testing.T) {
	metrics := &CDNMetrics{
		CacheHitRate:  85.5,
		TotalRequests: 1000000,
		BandwidthMB:   5000.0,
		AvgLatencyMs:  45.0,
		P95LatencyMs:  120.0,
		P99LatencyMs:  200.0,
		LastUpdated:   time.Now(),
	}

	if metrics.CacheHitRate != 85.5 {
		t.Errorf("Expected cache hit rate 85.5, got %f", metrics.CacheHitRate)
	}

	if metrics.TotalRequests != 1000000 {
		t.Errorf("Expected total requests 1000000, got %d", metrics.TotalRequests)
	}
}

func TestMultiRegionManager(t *testing.T) {
	cfg := &config.MultiRegionConfig{
		Enabled: true,
		Regions: []config.RegionConfig{
			{Name: "North America", ID: "na-east", Endpoint: "https://na-east.example.com", Priority: 1, Weight: 100},
			{Name: "Europe", ID: "eu-west", Endpoint: "https://eu-west.example.com", Priority: 1, Weight: 100},
			{Name: "Asia Pacific", ID: "ap-east", Endpoint: "https://ap-east.example.com", Priority: 1, Weight: 100},
		},
	}

	manager, err := NewMultiRegionManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create multi-region manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	regions := manager.GetAllRegions()
	if len(regions) != 3 {
		t.Errorf("Expected 3 regions, got %d", len(regions))
	}

	region, ok := manager.GetRegion("na-east")
	if !ok {
		t.Error("Expected na-east region to exist")
	}

	if region.Name != "North America" {
		t.Errorf("Expected region name 'North America', got '%s'", region.Name)
	}

	healthyRegions := manager.GetHealthyRegions()
	if len(healthyRegions) != 3 {
		t.Errorf("Expected 3 healthy regions, got %d", len(healthyRegions))
	}
}

func TestMultiRegionGetBestRegion(t *testing.T) {
	cfg := &config.MultiRegionConfig{
		Enabled: true,
		Regions: []config.RegionConfig{
			{Name: "NA East", ID: "na-east", Endpoint: "https://na-east.example.com", Priority: 2, Weight: 100},
			{Name: "EU West", ID: "eu-west", Endpoint: "https://eu-west.example.com", Priority: 1, Weight: 100},
			{Name: "AP East", ID: "ap-east", Endpoint: "https://ap-east.example.com", Priority: 1, Weight: 100},
		},
	}

	manager, _ := NewMultiRegionManager(cfg)

	manager.UpdateRegionHealth("na-east", true, 50)
	manager.UpdateRegionHealth("eu-west", true, 30)
	manager.UpdateRegionHealth("ap-east", true, 200)

	best := manager.GetBestRegion(100)
	if best == nil {
		t.Fatal("Expected best region to be found")
	}

	if best.ID != "eu-west" {
		t.Errorf("Expected best region 'eu-west' (lowest latency), got '%s'", best.ID)
	}

	bestHighThreshold := manager.GetBestRegion(1000)
	if bestHighThreshold == nil {
		t.Fatal("Expected best region with high threshold")
	}

	if bestHighThreshold.ID != "na-east" && bestHighThreshold.ID != "eu-west" {
		t.Errorf("Expected best region to be na-east or eu-west, got '%s'", bestHighThreshold.ID)
	}
}

func TestMultiRegionUpdateHealth(t *testing.T) {
	cfg := &config.MultiRegionConfig{
		Enabled: true,
		Regions: []config.RegionConfig{
			{Name: "NA East", ID: "na-east", Endpoint: "https://na-east.example.com"},
		},
	}

	manager, _ := NewMultiRegionManager(cfg)

	region, _ := manager.GetRegion("na-east")
	if !region.Healthy {
		t.Error("Expected initial health to be true")
	}

	manager.UpdateRegionHealth("na-east", false, 0)

	region, _ = manager.GetRegion("na-east")
	if region.Healthy {
		t.Error("Expected health to be false after update")
	}

	manager.UpdateRegionHealth("na-east", true, 100)

	region, _ = manager.GetRegion("na-east")
	if !region.Healthy {
		t.Error("Expected health to be true after update")
	}

	if region.LatencyMs != 100 {
		t.Errorf("Expected latency 100, got %d", region.LatencyMs)
	}
}

func TestSmartRouter(t *testing.T) {
	cfg := &config.MultiRegionConfig{
		Enabled: true,
		Regions: []config.RegionConfig{
			{Name: "NA East", ID: "na-east", Endpoint: "https://na-east.example.com", Priority: 2},
			{Name: "EU West", ID: "eu-west", Endpoint: "https://eu-west.example.com", Priority: 1},
		},
	}

	multiRegion, _ := NewMultiRegionManager(cfg)

	smartCfg := &config.SmartRoutingConfig{
		Enabled:          true,
		Strategy:         "latency",
		LatencyThreshold: 100,
	}

	router := NewSmartRouter(smartCfg, multiRegion, nil)

	endpoint := router.GetBestEndpoint(100)
	if endpoint == "" {
		t.Error("Expected best endpoint to be returned")
	}

	multiRegion.UpdateRegionHealth("na-east", true, 50)
	multiRegion.UpdateRegionHealth("eu-west", true, 30)

	ctx := context.Background()
	result, err := router.RouteRequest(ctx, "192.168.1.1")
	if err != nil {
		t.Errorf("Failed to route request: %v", err)
	}

	if result == nil {
		t.Fatal("Expected route result to be non-nil")
	}

	if !result.CacheEnabled {
		t.Error("Expected cache to be enabled")
	}
}

func TestEdgeComputingManager(t *testing.T) {
	cfg := &config.EdgeComputingConfig{
		Enabled: true,
		Nodes: []config.EdgeNodeConfig{
			{ID: "edge-us-east", Name: "US East Edge", Region: "us-east-1", Endpoint: "https://edge-us.example.com", Capacity: 1000},
			{ID: "edge-eu-west", Name: "EU West Edge", Region: "eu-west-1", Endpoint: "https://edge-eu.example.com", Capacity: 1000},
			{ID: "edge-ap-east", Name: "AP East Edge", Region: "ap-east-1", Endpoint: "https://edge-ap.example.com", Capacity: 1000},
		},
	}

	manager, err := NewEdgeComputingManager(cfg)
	if err != nil {
		t.Fatalf("Failed to create edge computing manager: %v", err)
	}

	if manager == nil {
		t.Fatal("Expected manager to be created")
	}

	nodes := manager.GetAllNodes()
	if len(nodes) != 3 {
		t.Errorf("Expected 3 nodes, got %d", len(nodes))
	}

	node, ok := manager.GetNode("edge-us-east")
	if !ok {
		t.Error("Expected edge-us-east node to exist")
	}

	if node.Name != "US East Edge" {
		t.Errorf("Expected node name 'US East Edge', got '%s'", node.Name)
	}

	if node.Capacity != 1000 {
		t.Errorf("Expected capacity 1000, got %d", node.Capacity)
	}
}

func TestEdgeComputingGetBestNode(t *testing.T) {
	cfg := &config.EdgeComputingConfig{
		Enabled: true,
		Nodes: []config.EdgeNodeConfig{
			{ID: "edge-us-east", Name: "US East", Region: "us-east-1", Capacity: 1000},
			{ID: "edge-eu-west", Name: "EU West", Region: "eu-west-1", Capacity: 1000},
			{ID: "edge-ap-east", Name: "AP East", Region: "ap-east-1", Capacity: 1000},
		},
	}

	manager, _ := NewEdgeComputingManager(cfg)

	best := manager.GetBestNode("")
	if best == nil {
		t.Fatal("Expected best node to be found")
	}

	bestUS := manager.GetBestNode("us-east-1")
	if bestUS == nil {
		t.Fatal("Expected best US node to be found")
	}

	if bestUS.Region != "us-east-1" {
		t.Errorf("Expected region 'us-east-1', got '%s'", bestUS.Region)
	}
}

func TestEdgeComputingExecuteFunction(t *testing.T) {
	cfg := &config.EdgeComputingConfig{
		Enabled: true,
		Nodes: []config.EdgeNodeConfig{
			{ID: "edge-us-east", Name: "US East", Region: "us-east-1", Capacity: 1000},
		},
	}

	manager, _ := NewEdgeComputingManager(cfg)

	params := map[string]interface{}{
		"user_id": "123",
		"action":  "validate",
	}

	err := manager.ExecuteFunction("captcha-validate", params)
	if err != nil {
		t.Errorf("Failed to execute function: %v", err)
	}
}

func TestCDNManagerRecordMetrics(t *testing.T) {
	cfg := &config.CDNConfig{
		Enabled:  true,
		Provider: "mock",
	}

	manager, _ := NewCDNManager(cfg)

	manager.RecordRequest()
	manager.RecordRequest()
	manager.RecordCacheHit()
	manager.RecordCacheMiss()
	manager.RecordBandwidth(1024 * 1024)
	manager.RecordLatency(50)
}
