
package service

import (
	"context"
	"testing"
	"time"
)

func TestEdgeNodeManager(t *testing.T) {
	manager := NewEdgeNodeManager()
	ctx := context.Background()

	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	if !manager.initialized {
		t.Error("Manager should be initialized")
	}
}

func TestNodeRegistration(t *testing.T) {
	manager := NewEdgeNodeManager()
	ctx := context.Background()

	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	req := &NodeRegistrationRequest{
		Name:     "Test Node",
		Address:  "127.0.0.1",
		Port:     8080,
		Region:   "us-east-1",
		Capacity: 100,
		Features: []string{"detection", "analysis"},
	}

	resp, err := manager.RegisterNode(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	if !resp.Success {
		t.Error("Registration should succeed")
	}

	if resp.NodeID == "" {
		t.Error("Node ID should not be empty")
	}
}

func TestListNodes(t *testing.T) {
	manager := NewEdgeNodeManager()
	ctx := context.Background()

	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	req := &NodeRegistrationRequest{
		Name:     "Test Node 1",
		Address:  "127.0.0.1",
		Port:     8080,
		Region:   "us-east-1",
		Capacity: 100,
		Features: []string{"detection"},
	}

	_, err := manager.RegisterNode(ctx, req)
	if err != nil {
		t.Fatalf("Failed to register node: %v", err)
	}

	nodes := manager.ListNodes(ctx)
	if len(nodes) != 1 {
		t.Errorf("Expected 1 node, got %d", len(nodes))
	}
}

func TestDeregisterNode(t *testing.T) {
	manager := NewEdgeNodeManager()
	ctx := context.Background()

	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	req := &NodeRegistrationRequest{
		Name:     "Test Node",
		Address:  "127.0.0.1",
		Port:     8080,
		Region:   "us-east-1",
		Capacity: 100,
		Features: []string{"detection"},
	}

	resp, _ := manager.RegisterNode(ctx, req)
	nodeID := resp.NodeID

	if err := manager.DeregisterNode(ctx, nodeID); err != nil {
		t.Fatalf("Failed to deregister node: %v", err)
	}

	nodes := manager.ListNodes(ctx)
	if len(nodes) != 0 {
		t.Errorf("Expected 0 nodes, got %d", len(nodes))
	}
}

func TestSelectNode(t *testing.T) {
	manager := NewEdgeNodeManager()
	ctx := context.Background()

	if err := manager.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize manager: %v", err)
	}

	req := &NodeRegistrationRequest{
		Name:     "Test Node",
		Address:  "127.0.0.1",
		Port:     8080,
		Region:   "us-east-1",
		Capacity: 100,
		Features: []string{"detection", "analysis"},
	}

	_, _ = manager.RegisterNode(ctx, req)

	node, err := manager.SelectNode(ctx, "detection")
	if err != nil {
		t.Fatalf("Failed to select node: %v", err)
	}

	if node == nil {
		t.Error("Selected node should not be nil")
	}
}

func TestEdgeLoadBalancer(t *testing.T) {
	lb := NewEdgeLoadBalancer()

	node1 := &EdgeNode{
		ID:          "node1",
		Healthy:     true,
		Status:      "active",
		CurrentLoad: 50,
	}

	node2 := &EdgeNode{
		ID:          "node2",
		Healthy:     true,
		Status:      "active",
		CurrentLoad: 10,
	}

	lb.AddNode(node1)
	lb.AddNode(node2)

	lb.mu.Lock()
	if len(lb.nodes) != 2 {
		t.Errorf("Expected 2 nodes in load balancer, got %d", len(lb.nodes))
	}
	lb.mu.Unlock()

	lb.RemoveNode("node1")
	lb.mu.Lock()
	if len(lb.nodes) != 1 {
		t.Errorf("Expected 1 node in load balancer, got %d", len(lb.nodes))
	}
	lb.mu.Unlock()
}

func TestHealthChecker(t *testing.T) {
	hc := NewHealthChecker()

	if hc.checkInterval != 30*time.Second {
		t.Errorf("Expected check interval 30s, got %v", hc.checkInterval)
	}

	if hc.timeout != 5*time.Second {
		t.Errorf("Expected timeout 5s, got %v", hc.timeout)
	}
}

func TestEdgeComputingService(t *testing.T) {
	nodeManager := NewEdgeNodeManager()
	service := NewEdgeComputingService(nodeManager)
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	if !service.initialized {
		t.Error("Service should be initialized")
	}
}

func TestCacheOperations(t *testing.T) {
	nodeManager := NewEdgeNodeManager()
	service := NewEdgeComputingService(nodeManager)
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	key := "test_key"
	value := []byte("test_value")
	ttl := 10 * time.Minute

	service.SetCache(key, value, ttl)

	cached, found := service.GetCache(key)
	if !found {
		t.Error("Expected to find cached value")
	}

	if string(cached) != string(value) {
		t.Errorf("Expected cached value to be %s, got %s", string(value), string(cached))
	}
}

func TestExecuteTask(t *testing.T) {
	nodeManager := NewEdgeNodeManager()
	service := NewEdgeComputingService(nodeManager)
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	req := &EdgeComputeRequest{
		TaskID:    "test_task",
		TaskType:  "detection",
		InputData: map[string]interface{}{"data": "test"},
		UseCache:  false,
	}

	resp, err := service.ExecuteTask(ctx, req)
	if err != nil {
		t.Fatalf("Failed to execute task: %v", err)
	}

	if !resp.Success {
		t.Error("Task should succeed")
	}

	if resp.TaskID != "test_task" {
		t.Errorf("Expected task ID to be test_task, got %s", resp.TaskID)
	}
}

func TestCacheInvalidation(t *testing.T) {
	nodeManager := NewEdgeNodeManager()
	service := NewEdgeComputingService(nodeManager)
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	service.SetCache("key1", []byte("value1"), 10*time.Minute)
	service.SetCache("key2", []byte("value2"), 10*time.Minute)

	invReq := &CacheInvalidationRequest{
		Keys:          []string{"key1"},
		InvalidateAll: false,
	}

	service.InvalidateCache(invReq)

	_, found := service.GetCache("key1")
	if found {
		t.Error("key1 should have been invalidated")
	}

	_, found = service.GetCache("key2")
	if !found {
		t.Error("key2 should still exist")
	}
}

func TestStats(t *testing.T) {
	nodeManager := NewEdgeNodeManager()
	service := NewEdgeComputingService(nodeManager)
	ctx := context.Background()

	if err := service.Initialize(ctx); err != nil {
		t.Fatalf("Failed to initialize service: %v", err)
	}

	stats := service.GetStats()
	if stats == nil {
		t.Fatal("Stats should not be nil")
	}

	cacheStats := service.GetCacheStats()
	if cacheStats == nil {
		t.Fatal("Cache stats should not be nil")
	}
}

func TestCacheStrategies(t *testing.T) {
	cs := NewCacheStrategy()

	cs.SetDefaultTTL(30 * time.Minute)
	if cs.defaultTTL != 30*time.Minute {
		t.Errorf("Expected default TTL 30min, got %v", cs.defaultTTL)
	}

	cs.SetTTL("detection", 5*time.Minute)
	ttl := cs.GetTTL("detection")
	if ttl != 5*time.Minute {
		t.Errorf("Expected TTL 5min for detection, got %v", ttl)
	}

	ttl = cs.GetTTL("unknown")
	if ttl != 30*time.Minute {
		t.Errorf("Expected default TTL for unknown task, got %v", ttl)
	}
}

func TestEdgeCache(t *testing.T) {
	cache := NewEdgeCache(100, "lru")

	cache.Set("key1", []byte("value1"), 10*time.Minute)
	cache.Set("key2", []byte("value2"), 10*time.Minute)

	item, found := cache.Get("key1")
	if !found {
		t.Error("Expected to find key1")
	}

	if string(item.Value) != "value1" {
		t.Errorf("Expected value1, got %s", string(item.Value))
	}

	cache.Delete("key1")
	_, found = cache.Get("key1")
	if found {
		t.Error("key1 should have been deleted")
	}
}
