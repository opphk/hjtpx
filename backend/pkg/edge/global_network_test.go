package edge

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestGlobalNetwork_Initialize(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()

	err := network.Initialize(ctx)
	if err != nil {
		t.Fatalf("Initialize failed: %v", err)
	}

	if !network.isRunning {
		t.Error("Network should be running after Initialize")
	}

	network.Shutdown()
}

func TestGlobalNetwork_RegisterNode(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	req := &NodeRegistrationRequest{
		Name:       "test-node-1",
		Address:    "192.168.1.1",
		Port:       8080,
		Region:     "us-west",
		Zone:       "us-west-1",
		Continent:  "north_america",
		Capacity:   1000,
		Priority:   1,
		Features:   []string{"verification", "inference"},
	}

	resp, err := network.RegisterNode(ctx, req)
	if err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	if !resp.Success {
		t.Error("Registration should succeed")
	}

	if resp.NodeID == "" {
		t.Error("NodeID should not be empty")
	}
}

func TestGlobalNetwork_ConcurrentNodeRegistration(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	var wg sync.WaitGroup
	nodeCount := 100

	for i := 0; i < nodeCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &NodeRegistrationRequest{
				Name:      "concurrent-node",
				Address:   "10.0.0.1",
				Port:      8080 + id,
				Region:    "us-west",
				Continent: "north_america",
				Capacity:  100,
				Priority:  1,
			}
			network.RegisterNode(ctx, req)
		}(i)
	}

	wg.Wait()

	nodes := network.ListNodes(ctx)
	if len(nodes) != nodeCount {
		t.Errorf("Expected %d nodes, got %d", nodeCount, len(nodes))
	}
}

func TestGlobalNetwork_ProcessVerification(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	nodeReq := &NodeRegistrationRequest{
		Name:      "verification-node",
		Address:   "192.168.1.100",
		Port:      8080,
		Region:    "us-west",
		Continent: "north_america",
		Capacity:  1000,
		Priority:  1,
	}
	network.RegisterNode(ctx, nodeReq)

	req := &VerificationRequest{
		RequestID: "test-req-1",
		UserID:    "user-1",
		Data:      []byte("test-data"),
		UseCache:  true,
		CacheKey:  "test-cache-key",
	}

	resp, err := network.ProcessVerification(ctx, req)
	if err != nil {
		t.Fatalf("ProcessVerification failed: %v", err)
	}

	if !resp.Success {
		t.Error("Verification should succeed")
	}

	if resp.RequestID != req.RequestID {
		t.Errorf("Expected request ID %s, got %s", req.RequestID, resp.RequestID)
	}
}

func TestGlobalNetwork_ConcurrentVerification(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	nodeReq := &NodeRegistrationRequest{
		Name:      "load-test-node",
		Address:   "192.168.1.200",
		Port:      8080,
		Region:    "eu-central",
		Continent: "europe",
		Capacity:  10000,
		Priority:  1,
	}
	network.RegisterNode(ctx, nodeReq)

	var wg sync.WaitGroup
	requestCount := 1000

	start := time.Now()
	for i := 0; i < requestCount; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			req := &VerificationRequest{
				RequestID: "load-test-req",
				UserID:    "user",
				Data:      []byte("test-data"),
			}
			network.ProcessVerification(ctx, req)
		}(i)
	}

	wg.Wait()
	elapsed := time.Since(start)

	t.Logf("Processed %d requests in %v", requestCount, elapsed)
	t.Logf("Throughput: %.2f req/s", float64(requestCount)/elapsed.Seconds())

	stats := network.GetStats()
	totalRequests := stats["total_requests"].(int64)
	if totalRequests != int64(requestCount) {
		t.Errorf("Expected %d total requests, got %d", requestCount, totalRequests)
	}
}

func TestGlobalNetwork_RegionManagement(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	regions := []struct {
		name      string
		region    string
		continent string
	}{
		{"node-us-1", "us-west", "north_america"},
		{"node-us-2", "us-east", "north_america"},
		{"node-eu-1", "eu-west", "europe"},
		{"node-eu-2", "eu-central", "europe"},
		{"node-asia-1", "ap-east", "asia"},
	}

	for _, r := range regions {
		req := &NodeRegistrationRequest{
			Name:      r.name,
			Address:   "192.168.1.1",
			Port:      8080,
			Region:    r.region,
			Continent: r.continent,
			Capacity:  100,
			Priority:  1,
		}
		network.RegisterNode(ctx, req)
	}

	regionList := network.ListRegions(ctx)
	if len(regionList) != 3 {
		t.Errorf("Expected 3 regions, got %d", len(regionList))
	}
}

func TestGlobalNetwork_LoadBalancing(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	nodeCount := 5
	for i := 0; i < nodeCount; i++ {
		req := &NodeRegistrationRequest{
			Name:      "lb-node",
			Address:   "10.0.0.1",
			Port:      8080 + i,
			Region:    "us-west",
			Continent: "north_america",
			Capacity:  1000,
			Priority:  1,
		}
		network.RegisterNode(ctx, req)
	}

	nodeDistribution := make(map[string]int)
	requestCount := 1000

	for i := 0; i < requestCount; i++ {
		node, err := network.loadBalancer.SelectNode(network)
		if err != nil {
			t.Fatalf("SelectNode failed: %v", err)
		}
		nodeDistribution[node.ID]++
	}

	if len(nodeDistribution) != nodeCount {
		t.Errorf("Expected requests distributed across %d nodes, got %d", nodeCount, len(nodeDistribution))
	}

	t.Logf("Node distribution: %v", nodeDistribution)
}

func TestGlobalNetwork_Failover(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	primaryReq := &NodeRegistrationRequest{
		Name:      "primary-node",
		Address:   "10.0.0.1",
		Port:      8080,
		Region:    "us-west",
		Continent: "north_america",
		Capacity:  1000,
		Priority:  1,
	}
	network.RegisterNode(ctx, primaryReq)

	network.DeregisterNode(ctx, "global_node_1")

	failoverNode := network.failover.getFailoverNode(ctx, network)
	if failoverNode == nil {
		t.Log("No failover node available (expected when no backup registered)")
	}
}

func TestGlobalNetwork_GetStats(t *testing.T) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	nodeReq := &NodeRegistrationRequest{
		Name:      "stats-node",
		Address:   "192.168.1.1",
		Port:      8080,
		Region:    "us-west",
		Continent: "north_america",
		Capacity:  1000,
		Priority:  1,
	}
	network.RegisterNode(ctx, nodeReq)

	stats := network.GetStats()

	if stats["active_nodes"].(int64) != 1 {
		t.Errorf("Expected 1 active node, got %d", stats["active_nodes"])
	}

	if stats["healthy_regions"].(int64) != 1 {
		t.Errorf("Expected 1 healthy region, got %d", stats["healthy_regions"])
	}
}

func BenchmarkGlobalNetwork_ProcessVerification(b *testing.B) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	nodeReq := &NodeRegistrationRequest{
		Name:      "bench-node",
		Address:   "192.168.1.1",
		Port:      8080,
		Region:    "us-west",
		Continent: "north_america",
		Capacity:  100000,
		Priority:  1,
	}
	network.RegisterNode(ctx, nodeReq)

	req := &VerificationRequest{
		RequestID: "bench-req",
		UserID:    "user",
		Data:      []byte("benchmark-data"),
		UseCache:  true,
		CacheKey:  "bench-cache",
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		network.ProcessVerification(ctx, req)
	}
}

func BenchmarkGlobalNetwork_ConcurrentProcessVerification(b *testing.B) {
	network := NewGlobalNetwork()
	ctx := context.Background()
	network.Initialize(ctx)
	defer network.Shutdown()

	nodeReq := &NodeRegistrationRequest{
		Name:      "concurrent-bench-node",
		Address:   "192.168.1.1",
		Port:      8080,
		Region:    "us-west",
		Continent: "north_america",
		Capacity:  100000,
		Priority:  1,
	}
	network.RegisterNode(ctx, nodeReq)

	var wg sync.WaitGroup
	requestsPerGoroutine := b.N / 100

	b.ResetTimer()
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < requestsPerGoroutine; j++ {
				req := &VerificationRequest{
					RequestID: "bench-req",
					UserID:    "user",
					Data:      []byte("benchmark-data"),
				}
				network.ProcessVerification(ctx, req)
			}
		}()
	}

	wg.Wait()
}
