package balancer

import (
	"context"
	"sync"
	"testing"
	"time"
)

func BenchmarkWeightedRandomSelect(b *testing.B) {
	balancer := NewWeightedRandomBalancer()

	for i := 0; i < 10; i++ {
		balancer.AddNode(NewBaseNode("node-"+string(rune('0'+i)), 100))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			balancer.SelectNode()
		}
	})
}

func BenchmarkWeightedRandomSelectManyNodes(b *testing.B) {
	balancer := NewWeightedRandomBalancer()

	for i := 0; i < 100; i++ {
		balancer.AddNode(NewBaseNode("node-"+string(rune('0'+i%10)), (i%10)+1))
	}

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			balancer.SelectNode()
		}
	})
}

func TestWeightedRandomBalancer(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("node1", 3))
	balancer.AddNode(NewBaseNode("node2", 2))
	balancer.AddNode(NewBaseNode("node3", 1))

	nodeCount := make(map[string]int)
	for i := 0; i < 1000; i++ {
		node, err := balancer.SelectNode()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		nodeCount[node.Address()]++
	}

	if nodeCount["node1"] <= nodeCount["node3"] {
		t.Errorf("expected node1 to be selected more than node3")
	}
	if nodeCount["node2"] <= nodeCount["node3"] {
		t.Errorf("expected node2 to be selected more than node3")
	}
}

func TestWeightedRandomBalancerRemoveNode(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("node1", 1))
	balancer.AddNode(NewBaseNode("node2", 1))

	_, err := balancer.SelectNode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	balancer.RemoveNode("node1")

	_, err = balancer.SelectNode()
	if err != nil {
		t.Fatalf("unexpected error after removing node: %v", err)
	}

	if balancer.NodeCount() != 1 {
		t.Errorf("expected 1 node, got %d", balancer.NodeCount())
	}
}

func TestWeightedRandomBalancerNoNodes(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	_, err := balancer.SelectNode()
	if err == nil {
		t.Error("expected error when no nodes available")
	}
}

func TestWeightedRandomBalancerHealthCheck(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	node1 := NewBaseNode("node1", 1)
	balancer.AddNode(node1)

	node2 := NewBaseNode("node2", 1)
	node2.SetHealthy(false)
	balancer.AddNode(node2)

	node, err := balancer.SelectNode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = node
}

func TestWeightedRandomBalancerFallback(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	node1 := NewBaseNode("node1", 1)
	balancer.AddNode(node1)

	node, err := balancer.SelectWithFallback()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = node
}

func TestWeightedRandomBalancerUpdateHealth(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("node1", 1))
	balancer.AddNode(NewBaseNode("node2", 1))

	balancer.UpdateNodeHealth("node1", false)

	node, err := balancer.SelectNode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	_ = node
}

func TestWeightedRandomBalancerUpdateWeight(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("node1", 1))
	balancer.AddNode(NewBaseNode("node2", 1))

	balancer.UpdateNodeWeight("node1", 10)

	nodeCount := make(map[string]int)
	for i := 0; i < 1000; i++ {
		node, _ := balancer.SelectNode()
		nodeCount[node.Address()]++
	}

	if nodeCount["node1"] <= nodeCount["node2"] {
		t.Errorf("expected node1 to be selected more often with higher weight")
	}
}

func TestWeightedRandomBalancerRecordRequest(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("node1", 1))
	balancer.AddNode(NewBaseNode("node2", 1))

	for i := 0; i < 100; i++ {
		balancer.RecordRequest("node1", 10.0, i%10 != 0)
	}

	stats, ok := balancer.GetNodeStats("node1")
	if !ok {
		t.Error("expected stats for node1")
	}

	if stats.RequestsTotal != 100 {
		t.Errorf("expected 100 requests, got %d", stats.RequestsTotal)
	}
	if stats.SuccessTotal != 90 {
		t.Errorf("expected 90 successes, got %d", stats.SuccessTotal)
	}
}

func TestWeightedRandomBalancerAllStats(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("node1", 1))
	balancer.AddNode(NewBaseNode("node2", 1))

	balancer.RecordRequest("node1", 10.0, true)

	stats := balancer.GetAllStats()
	if len(stats) != 2 {
		t.Errorf("expected 2 node stats, got %d", len(stats))
	}
}

func TestBaseNodeMetadata(t *testing.T) {
	node := NewBaseNode("node1", 1)

	node.SetMetadata("key1", "value1")
	node.SetMetadata("key2", 123)

	if v, ok := node.GetMetadata("key1"); !ok || v != "value1" {
		t.Errorf("unexpected metadata for key1: %v", v)
	}

	if v, ok := node.GetMetadata("key2"); !ok || v != 123 {
		t.Errorf("unexpected metadata for key2: %v", v)
	}
}

func TestWeightedSelectorSelect(t *testing.T) {
	nodes := []Node{
		NewBaseNode("node1", 3),
		NewBaseNode("node2", 2),
		NewBaseNode("node3", 1),
	}

	selector := newWeightedSelector(nodes)

	counts := make(map[string]int)
	for i := 0; i < 1000; i++ {
		node, err := selector.Select()
		if err != nil {
			t.Fatalf("unexpected error: %v", err)
		}
		counts[node.Address()]++
	}

	if counts["node1"] <= counts["node3"] {
		t.Error("expected node1 to be selected most frequently")
	}
}

func TestWeightedSelectorRebuild(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	balancer.AddNode(NewBaseNode("node1", 1))
	balancer.AddNode(NewBaseNode("node2", 1))

	_, err := balancer.SelectNode()
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	balancer.UpdateNodeWeight("node1", 10)

	node2, err := balancer.SelectNode()
	if err != nil {
		t.Fatalf("unexpected error after weight update: %v", err)
	}

	_ = node2
}

func TestBalancerWithHealthCheck(t *testing.T) {
	balancer := NewWeightedRandomBalancer()
	balancer.AddNode(NewBaseNode("node1", 1))

	checker := NewTCPPingChecker(5 * time.Second)
	bwh := NewBalancerWithHealthCheck(balancer, checker, 1*time.Second)

	bwh.Start()
	time.Sleep(100 * time.Millisecond)
	bwh.Stop()
}

func TestBalancerConcurrency(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	for i := 0; i < 10; i++ {
		balancer.AddNode(NewBaseNode("node-"+string(rune('0'+i)), 1))
	}

	var wg sync.WaitGroup
	concurrency := 100

	for i := 0; i < concurrency; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			for j := 0; j < 100; j++ {
				balancer.SelectNode()
				balancer.GetAllStats()
			}
		}()
	}

	wg.Wait()
}

func TestNodeMetrics(t *testing.T) {
	metrics := &NodeMetrics{
		NodeAddress:   "test",
		RequestsTotal: 100,
		SuccessTotal:  90,
		FailureTotal: 10,
		AvgLatencyMs: 5.5,
		Weight:       1,
		Status:       NodeStatusHealthy,
	}

	if metrics.NodeAddress != "test" {
		t.Errorf("unexpected node address: %s", metrics.NodeAddress)
	}
	if metrics.RequestsTotal != 100 {
		t.Errorf("unexpected requests total: %d", metrics.RequestsTotal)
	}
}

func TestEnsureLoadBalancer(t *testing.T) {
	var nilBalancer LoadBalancer
	result := EnsureLoadBalancer(nilBalancer)
	if result == nil {
		t.Error("expected non-nil balancer")
	}

	balancer := NewWeightedRandomBalancer()
	result = EnsureLoadBalancer(balancer)
	if result != balancer {
		t.Error("expected same balancer")
	}
}

func BenchmarkNodeHealthUpdate(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	for i := 0; i < 10; i++ {
		balancer.AddNode(NewBaseNode("node-"+string(rune('0'+i)), 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.UpdateNodeHealth("node-1", i%2 == 0)
	}
}

func BenchmarkRecordRequest(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	balancer.AddNode(NewBaseNode("node1", 1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.RecordRequest("node1", float64(i%100), i%2 == 0)
	}
}

func BenchmarkGetAllStats(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	for i := 0; i < 10; i++ {
		balancer.AddNode(NewBaseNode("node-"+string(rune('0'+i)), 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.GetAllStats()
	}
}

func BenchmarkGetNodeStats(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	balancer.AddNode(NewBaseNode("node1", 1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.GetNodeStats("node1")
	}
}

func BenchmarkHealthyNodeCount(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	for i := 0; i < 10; i++ {
		balancer.AddNode(NewBaseNode("node-"+string(rune('0'+i)), 1))
	}

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.HealthyNodeCount()
	}
}

func TestNodeWeightBounds(t *testing.T) {
	node := NewBaseNode("test", 0)
	if node.Weight() != 1 {
		t.Errorf("expected weight to be at least 1, got %d", node.Weight())
	}

	node.SetWeight(0)
	if node.Weight() != 1 {
		t.Errorf("expected weight to be at least 1 after setting 0, got %d", node.Weight())
	}
}

func TestNodeAddress(t *testing.T) {
	node := NewBaseNode("test-address", 1)
	if node.Address() != "test-address" {
		t.Errorf("unexpected address: %s", node.Address())
	}
}

func TestNodeHealthy(t *testing.T) {
	node := NewBaseNode("test", 1)

	if !node.IsHealthy() {
		t.Error("expected node to be healthy by default")
	}

	node.SetHealthy(false)
	if node.IsHealthy() {
		t.Error("expected node to be unhealthy after SetHealthy(false)")
	}

	node.SetHealthy(true)
	if !node.IsHealthy() {
		t.Error("expected node to be healthy after SetHealthy(true)")
	}
}

type mockHealthChecker struct {
	result bool
}

func (m *mockHealthChecker) CheckHealth(ctx context.Context, node Node) bool {
	return m.result
}

func TestMockHealthChecker(t *testing.T) {
	checker := NewTCPPingChecker(5 * time.Second)
	node := NewBaseNode("test", 1)

	ctx := context.Background()
	result := checker.CheckHealth(ctx, node)

	if !result {
		t.Error("expected health check to return true")
	}
}

func BenchmarkBalancerWithStateTransitions(b *testing.B) {
	balancer := NewWeightedRandomBalancer()
	balancer.AddNode(NewBaseNode("node1", 1))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		balancer.AddNode(NewBaseNode("new-node", 1))
		balancer.RemoveNode("new-node")
	}
}

func TestBalancerSelectDistribution(t *testing.T) {
	balancer := NewWeightedRandomBalancer()

	balancer.AddNode(NewBaseNode("high", 100))
	balancer.AddNode(NewBaseNode("low", 1))

	highCount := 0
	lowCount := 0

	for i := 0; i < 1000; i++ {
		node, _ := balancer.SelectNode()
		if node.Address() == "high" {
			highCount++
		} else {
			lowCount++
		}
	}

	ratio := float64(highCount) / float64(lowCount)
	if ratio < 50 || ratio > 150 {
		t.Errorf("unexpected ratio %f, expected approximately 100", ratio)
	}
}
