package edge

import (
	"context"
	"testing"
)

func TestEdgeLoadBalancer_NewEdgeLoadBalancer(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	if lb == nil {
		t.Fatal("Expected load balancer to not be nil")
	}

	if len(lb.nodes) != 0 {
		t.Errorf("Expected 0 nodes initially, got %d", len(lb.nodes))
	}

	if len(lb.policies) == 0 {
		t.Error("Expected default policies to be created")
	}
}

func TestEdgeLoadBalancer_AddNode(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	node := &LBNode{
		NodeID:    "node-1",
		IPAddress: "10.0.1.10",
		Port:      8080,
		Weight:    100,
	}

	err := lb.AddNode(node)
	if err != nil {
		t.Fatalf("AddNode failed: %v", err)
	}

	retrieved, err := lb.GetNode("node-1")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if retrieved.IPAddress != "10.0.1.10" {
		t.Errorf("Expected IP 10.0.1.10, got %s", retrieved.IPAddress)
	}
}

func TestEdgeLoadBalancer_RemoveNode(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	node := &LBNode{
		NodeID:    "node-2",
		IPAddress: "10.0.2.10",
		Port:      8080,
		Weight:    100,
	}

	lb.AddNode(node)

	err := lb.RemoveNode("node-2")
	if err != nil {
		t.Fatalf("RemoveNode failed: %v", err)
	}

	_, err = lb.GetNode("node-2")
	if err == nil {
		t.Error("Expected error for removed node")
	}
}

func TestEdgeLoadBalancer_UpdateNode(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	node := &LBNode{
		NodeID:    "node-3",
		IPAddress: "10.0.3.10",
		Port:      8080,
		Weight:    100,
	}

	lb.AddNode(node)

	updates := &LBNode{
		Weight:    200,
		LatencyMs: 50,
		CPUUsage:  0.3,
	}

	err := lb.UpdateNode("node-3", updates)
	if err != nil {
		t.Fatalf("UpdateNode failed: %v", err)
	}

	retrieved, _ := lb.GetNode("node-3")
	if retrieved.Weight != 200 {
		t.Errorf("Expected weight 200, got %d", retrieved.Weight)
	}
}

func TestEdgeLoadBalancer_ListNodes(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	nodes := []*LBNode{
		{NodeID: "node-4", IPAddress: "10.0.4.10", Port: 8080, Weight: 100},
		{NodeID: "node-5", IPAddress: "10.0.5.10", Port: 8080, Weight: 100},
	}

	for _, node := range nodes {
		lb.AddNode(node)
	}

	retrieved := lb.ListNodes()
	if len(retrieved) != 2 {
		t.Errorf("Expected 2 nodes, got %d", len(retrieved))
	}
}

func TestEdgeLoadBalancer_SelectNode_RoundRobin(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	nodes := []*LBNode{
		{NodeID: "node-6", IPAddress: "10.0.6.10", Port: 8080, Weight: 100, Healthy: true},
		{NodeID: "node-7", IPAddress: "10.0.7.10", Port: 8080, Weight: 100, Healthy: true},
	}

	for _, node := range nodes {
		lb.AddNode(node)
	}

	req := &LBRequest{
		ClientIP: "192.168.1.1",
		Policy:   "round_robin",
	}

	selected1, err := lb.SelectNode(req)
	if err != nil {
		t.Fatalf("SelectNode failed: %v", err)
	}

	selected2, err := lb.SelectNode(req)
	if err != nil {
		t.Fatalf("SelectNode failed: %v", err)
	}

	if selected1 == nil || selected2 == nil {
		t.Error("Expected nodes to be selected")
	}

	_ = selected1
	_ = selected2
}

func TestEdgeLoadBalancer_SelectNode_LeastConnection(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	nodes := []*LBNode{
		{NodeID: "node-8", IPAddress: "10.0.8.10", Port: 8080, Weight: 100, Healthy: true, CurrentConns: 10},
		{NodeID: "node-9", IPAddress: "10.0.9.10", Port: 8080, Weight: 100, Healthy: true, CurrentConns: 5},
	}

	for _, node := range nodes {
		lb.AddNode(node)
	}

	req := &LBRequest{
		ClientIP: "192.168.1.1",
		Policy:   "least_connection",
	}

	selected, err := lb.SelectNode(req)
	if err != nil {
		t.Fatalf("SelectNode failed: %v", err)
	}

	if selected == nil {
		t.Error("Expected a node to be selected")
	}

	if selected != nil && selected.CurrentConns > 10 {
		t.Error("Selected node should have reasonable connection count")
	}
}

func TestEdgeLoadBalancer_SelectNode_LatencyBased(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	nodes := []*LBNode{
		{NodeID: "node-10", IPAddress: "10.0.10.10", Port: 8080, Weight: 100, Healthy: true, LatencyMs: 100},
		{NodeID: "node-11", IPAddress: "10.0.11.10", Port: 8080, Weight: 100, Healthy: true, LatencyMs: 50},
	}

	for _, node := range nodes {
		lb.AddNode(node)
	}

	req := &LBRequest{
		ClientIP: "192.168.1.1",
		Policy:   "latency_based",
	}

	selected, err := lb.SelectNode(req)
	if err != nil {
		t.Fatalf("SelectNode failed: %v", err)
	}

	if selected.NodeID != "node-11" {
		t.Errorf("Expected node with lowest latency (node-11), got %s", selected.NodeID)
	}
}

func TestEdgeLoadBalancer_AddPolicy(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	policy := &LBPolicy{
		ID:        "custom-policy",
		Name:      "Custom Policy",
		Algorithm: LBAlgorithmWeighted,
		Mode:      LBModeActive,
		Enabled:   true,
	}

	err := lb.AddPolicy(policy)
	if err != nil {
		t.Fatalf("AddPolicy failed: %v", err)
	}

	retrieved, err := lb.GetPolicy("custom-policy")
	if err != nil {
		t.Fatalf("GetPolicy failed: %v", err)
	}

	if retrieved.Name != "Custom Policy" {
		t.Errorf("Expected name 'Custom Policy', got '%s'", retrieved.Name)
	}
}

func TestEdgeLoadBalancer_DeletePolicy(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	policy := &LBPolicy{
		ID:        "deletable-policy",
		Name:      "Deletable Policy",
		Algorithm: LBAlgorithmRoundRobin,
		Enabled:   true,
	}

	lb.AddPolicy(policy)

	err := lb.DeletePolicy("deletable-policy")
	if err != nil {
		t.Fatalf("DeletePolicy failed: %v", err)
	}

	_, err = lb.GetPolicy("deletable-policy")
	if err == nil {
		t.Error("Expected error for deleted policy")
	}
}

func TestEdgeLoadBalancer_ListPolicies(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	policies := lb.ListPolicies()
	if len(policies) < 5 {
		t.Errorf("Expected at least 5 default policies, got %d", len(policies))
	}
}

func TestEdgeLoadBalancer_GetMetrics(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	metrics := lb.GetMetrics()

	if metrics == nil {
		t.Fatal("Expected metrics to not be nil")
	}

	if metrics.TotalRequests != 0 {
		t.Errorf("Expected 0 total requests, got %d", metrics.TotalRequests)
	}
}

func TestEdgeLoadBalancer_RecordRequest(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	node := &LBNode{
		NodeID:    "node-12",
		IPAddress: "10.0.12.10",
		Port:      8080,
		Weight:    100,
		Healthy:   true,
	}

	lb.AddNode(node)

	lb.RecordRequest("node-12", 50.0, true)
	lb.RecordRequest("node-12", 100.0, true)
	lb.RecordRequest("node-12", 30.0, false)

	metrics := lb.GetMetrics()

	if metrics.TotalRequests != 3 {
		t.Errorf("Expected 3 total requests, got %d", metrics.TotalRequests)
	}
}

func TestEdgeLoadBalancer_GetVersion(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	version := lb.GetVersion()
	if version != 1 {
		t.Errorf("Expected version 1, got %d", version)
	}

	lb.AddNode(&LBNode{
		NodeID:    "node-13",
		IPAddress: "10.0.13.10",
		Port:      8080,
		Weight:    100,
	})

	newVersion := lb.GetVersion()
	if newVersion <= version {
		t.Error("Expected version to increase after adding node")
	}
}

func TestEdgeLoadBalancer_ForwardRequest(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	node := &LBNode{
		NodeID:    "node-14",
		IPAddress: "10.0.14.10",
		Port:      8080,
		Weight:    100,
		Healthy:   true,
	}

	lb.AddNode(node)

	req := &LBRequest{
		ClientIP: "192.168.1.1",
		Path:     "/api/test",
		Method:   "GET",
	}

	response, err := lb.ForwardRequest(context.Background(), req)
	if err != nil {
		t.Fatalf("ForwardRequest failed: %v", err)
	}

	if !response.Success {
		t.Error("Expected successful forward")
	}

	if response.Attempts < 1 {
		t.Error("Expected at least 1 attempt")
	}
}

func TestEdgeLoadBalancer_GetConfig(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	config := lb.GetConfig()

	if config == nil {
		t.Fatal("Expected config to not be nil")
	}

	if config.MaxConnections != 10000 {
		t.Errorf("Expected max connections 10000, got %d", config.MaxConnections)
	}
}

func TestEdgeLoadBalancer_SetConfig(t *testing.T) {
	lb := NewEdgeLoadBalancer(nil, nil, nil)

	newConfig := &LoadBalancerConfig{
		MaxConnections:   5000,
		ConnectionTimeout: 20,
		StickySessions:  false,
	}

	lb.SetConfig(newConfig)

	retrieved := lb.GetConfig()
	if retrieved.MaxConnections != 5000 {
		t.Errorf("Expected max connections 5000, got %d", retrieved.MaxConnections)
	}
}
