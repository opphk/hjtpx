package edge

import (
	"context"
	"testing"
	"time"
)

func TestEdgeNodeManager_RegisterNode(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-1",
		Name:      "Test Node 1",
		Region:    RegionAP,
		IPAddress: "10.0.1.10",
		Port:      8080,
		Type:      NodeTypeMixed,
		Capacity:  1000,
	}

	err := manager.RegisterNode(ctx, node)
	if err != nil {
		t.Fatalf("RegisterNode failed: %v", err)
	}

	retrieved, err := manager.GetNode("test-node-1")
	if err != nil {
		t.Fatalf("GetNode failed: %v", err)
	}

	if retrieved.ID != node.ID {
		t.Errorf("Expected ID %s, got %s", node.ID, retrieved.ID)
	}
	if retrieved.Region != node.Region {
		t.Errorf("Expected Region %s, got %s", node.Region, retrieved.Region)
	}
}

func TestEdgeNodeManager_UnregisterNode(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-2",
		Name:      "Test Node 2",
		Region:    RegionEU,
		IPAddress: "10.0.2.10",
		Port:      8080,
		Type:      NodeTypeCDN,
		Capacity:  500,
	}

	manager.RegisterNode(ctx, node)

	err := manager.UnregisterNode(ctx, "test-node-2")
	if err != nil {
		t.Fatalf("UnregisterNode failed: %v", err)
	}

	time.Sleep(6 * time.Second)

	_, err = manager.GetNode("test-node-2")
	if err == nil {
		t.Error("Expected error for unregistered node")
	}
}

func TestEdgeNodeManager_UpdateNodeStatus(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-3",
		Name:      "Test Node 3",
		Region:    RegionUS,
		IPAddress: "10.0.3.10",
		Port:      8080,
		Type:      NodeTypeInference,
		Capacity:  800,
	}

	manager.RegisterNode(ctx, node)

	err := manager.UpdateNodeStatus(ctx, "test-node-3", NodeStatusUnhealthy)
	if err != nil {
		t.Fatalf("UpdateNodeStatus failed: %v", err)
	}

	retrieved, _ := manager.GetNode("test-node-3")
	if retrieved.Status != NodeStatusUnhealthy {
		t.Errorf("Expected status %s, got %s", NodeStatusUnhealthy, retrieved.Status)
	}
}

func TestEdgeNodeManager_SelectNode(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	nodes := []*EdgeNode{
		{
			ID:        "test-node-4",
			Name:      "Test Node 4",
			Region:    RegionAP,
			IPAddress: "10.0.4.10",
			Port:      8080,
			Type:      NodeTypeMixed,
			Capacity:  1000,
			Weight:    100,
			LatencyMs: 50,
		},
		{
			ID:        "test-node-5",
			Name:      "Test Node 5",
			Region:    RegionAP,
			IPAddress: "10.0.5.10",
			Port:      8080,
			Type:      NodeTypeMixed,
			Capacity:  1000,
			Weight:    100,
			LatencyMs: 30,
		},
	}

	for _, node := range nodes {
		manager.RegisterNode(ctx, node)
	}

	selected, err := manager.SelectNode(&NodeSelectorOptions{
		Region:   RegionAP,
		NodeType: NodeTypeMixed,
		Status:   NodeStatusActive,
	})

	if err != nil {
		t.Fatalf("SelectNode failed: %v", err)
	}

	if selected == nil {
		t.Error("Expected selected node to not be nil")
	}
}

func TestEdgeNodeManager_GetClusterStats(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-6",
		Name:      "Test Node 6",
		Region:    RegionCN,
		IPAddress: "10.0.6.10",
		Port:      8080,
		Type:      NodeTypeMixed,
		Capacity:  1000,
	}

	manager.RegisterNode(ctx, node)

	stats := manager.GetClusterStats()

	if stats.TotalNodes != 1 {
		t.Errorf("Expected TotalNodes 1, got %d", stats.TotalNodes)
	}
	if stats.ActiveNodes != 1 {
		t.Errorf("Expected ActiveNodes 1, got %d", stats.ActiveNodes)
	}
}

func TestEdgeNodeManager_RecordHeartbeat(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-7",
		Name:      "Test Node 7",
		Region:    RegionJP,
		IPAddress: "10.0.7.10",
		Port:      8080,
		Type:      NodeTypeCDN,
		Capacity:  500,
	}

	manager.RegisterNode(ctx, node)

	err := manager.RecordHeartbeat("test-node-7")
	if err != nil {
		t.Fatalf("RecordHeartbeat failed: %v", err)
	}
}

func TestEdgeNodeManager_FindNearestNode(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-8",
		Name:      "Test Node 8",
		Region:    RegionAP,
		IPAddress: "10.0.8.10",
		Port:      8080,
		Type:      NodeTypeMixed,
		Capacity:  1000,
		Latitude:  35.6762,
		Longitude: 139.6503,
	}

	manager.RegisterNode(ctx, node)

	nearest, err := manager.FindNearestNode(35.6762, 139.6503)
	if err != nil {
		t.Fatalf("FindNearestNode failed: %v", err)
	}

	if nearest.ID != "test-node-8" {
		t.Errorf("Expected nearest node ID test-node-8, got %s", nearest.ID)
	}
}

func TestEdgeNodeManager_ResolveIPRegion(t *testing.T) {
	manager := NewEdgeNodeManager(nil)

	location, err := manager.ResolveIPRegion("8.8.8.8")
	if err != nil {
		t.Fatalf("ResolveIPRegion failed: %v", err)
	}

	if !location.IsPublic {
		t.Error("Expected IP to be public")
	}
}

func TestEdgeNodeManager_ListNodes(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	nodes := []*EdgeNode{
		{
			ID:        "test-node-9",
			Name:      "Test Node 9",
			Region:    RegionAP,
			IPAddress: "10.0.9.10",
			Port:      8080,
			Type:      NodeTypeCDN,
			Capacity:  500,
		},
		{
			ID:        "test-node-10",
			Name:      "Test Node 10",
			Region:    RegionEU,
			IPAddress: "10.0.10.10",
			Port:      8080,
			Type:      NodeTypeCDN,
			Capacity:  500,
		},
	}

	for _, node := range nodes {
		manager.RegisterNode(ctx, node)
	}

	apNodes, err := manager.ListNodes(&NodeSelectorOptions{Region: RegionAP})
	if err != nil {
		t.Fatalf("ListNodes failed: %v", err)
	}

	if len(apNodes) != 1 {
		t.Errorf("Expected 1 AP node, got %d", len(apNodes))
	}
}

func TestEdgeNodeManager_UpdateNodeMetrics(t *testing.T) {
	manager := NewEdgeNodeManager(nil)
	ctx := context.Background()

	node := &EdgeNode{
		ID:        "test-node-11",
		Name:      "Test Node 11",
		Region:    RegionUS,
		IPAddress: "10.0.11.10",
		Port:      8080,
		Type:      NodeTypeInference,
		Capacity:  1000,
	}

	manager.RegisterNode(ctx, node)

	metrics := &NodeMetrics{
		CPUUsage:     0.5,
		MemoryUsage:  0.6,
		DiskUsage:    0.3,
		RequestCount: 1000,
	}

	err := manager.UpdateNodeMetrics(ctx, "test-node-11", metrics)
	if err != nil {
		t.Fatalf("UpdateNodeMetrics failed: %v", err)
	}
}

func TestEdgeNodeManager_CalculateDistance(t *testing.T) {
	manager := NewEdgeNodeManager(nil)

	distance := manager.CalculateDistance(35.6762, 139.6503, 40.7128, -74.0060)

	if distance <= 0 {
		t.Error("Expected positive distance between Tokyo and New York")
	}

	expectedApprox := 10850.0
	if distance < expectedApprox-500 || distance > expectedApprox+500 {
		t.Errorf("Expected distance approx %f km, got %f km", expectedApprox, distance)
	}
}
