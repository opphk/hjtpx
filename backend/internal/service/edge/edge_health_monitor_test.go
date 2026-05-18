package edge

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestEdgeHealthMonitor_GetHealthStatus(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			HeartbeatIntervalSecs: 10,
		},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	node.Capacity = model.EdgeCapacity{
		MaxRequestsPerSecond:  1000,
		MaxConcurrentRequests: 100,
		MemoryLimitMB:         4096,
		CPUCores:              4,
	}
	node.CurrentLoad = model.EdgeLoadMetrics{
		CurrentRequestsPerSecond:   100,
		CurrentConcurrentRequests: 10,
		MemoryUsageMB:             1024,
		CPUUsagePercent:           20,
	}
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	healthMonitor := NewEdgeHealthMonitor(repo, cfg, syncService)
	ctx := context.Background()

	t.Run("get health status", func(t *testing.T) {
		result, err := healthMonitor.GetHealthStatus(ctx, "node-001")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "node-001", result.NodeID)
		assert.Equal(t, model.EdgeNodeStatusOnline, result.Status)
	})
}

func TestEdgeHealthMonitor_GetAllNodesHealth(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			HeartbeatIntervalSecs: 10,
		},
	}
	repo := newMockRepo()

	node1 := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node1.Status = model.EdgeNodeStatusOnline
	node1.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node1)

	node2 := model.NewEdgeNode("node-002", "Node 002", "edge", "cn-east-1", "zone-b")
	node2.Status = model.EdgeNodeStatusOnline
	node2.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node2)

	syncService := NewEdgeSyncService(repo, cfg)
	healthMonitor := NewEdgeHealthMonitor(repo, cfg, syncService)
	ctx := context.Background()

	t.Run("get all nodes health", func(t *testing.T) {
		results, err := healthMonitor.GetAllNodesHealth(ctx)
		assert.NoError(t, err)
		assert.Len(t, results, 2)
	})
}

func TestEdgeHealthMonitor_GetNodeStatus(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	healthMonitor := NewEdgeHealthMonitor(repo, cfg, syncService)
	ctx := context.Background()

	t.Run("get node status", func(t *testing.T) {
		status, err := healthMonitor.GetNodeStatus(ctx, "node-001")
		assert.NoError(t, err)
		assert.Equal(t, model.EdgeNodeStatusOnline, status)
	})

	t.Run("get node status not found", func(t *testing.T) {
		status, err := healthMonitor.GetNodeStatus(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Empty(t, status)
	})
}

func TestEdgeHealthMonitor_UpdateNodeStatus(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	healthMonitor := NewEdgeHealthMonitor(repo, cfg, syncService)
	ctx := context.Background()

	t.Run("update node status", func(t *testing.T) {
		err := healthMonitor.UpdateNodeStatus(ctx, "node-001", model.EdgeNodeStatusMaintenance)
		assert.NoError(t, err)

		status, err := healthMonitor.GetNodeStatus(ctx, "node-001")
		assert.NoError(t, err)
		assert.Equal(t, model.EdgeNodeStatusMaintenance, status)
	})
}

func TestEdgeHealthMonitor_RegisterHealthCallback(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			HealthCheckIntervalSecs: 1,
		},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	healthMonitor := NewEdgeHealthMonitor(repo, cfg, syncService)

	callbackCalled := false
	healthMonitor.RegisterHealthCallback(func(nodeID string, status model.EdgeNodeStatus, healthScore float64) {
		callbackCalled = true
	})

	assert.False(t, callbackCalled)
}

func TestEdgeHealthMonitor_StartAndStopMonitoring(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			HealthCheckIntervalSecs: 1,
		},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	healthMonitor := NewEdgeHealthMonitor(repo, cfg, syncService)
	ctx, cancel := context.WithCancel(context.Background())

	t.Run("start monitoring", func(t *testing.T) {
		err := healthMonitor.StartMonitoring(ctx)
		assert.NoError(t, err)
	})

	t.Run("start monitoring again", func(t *testing.T) {
		err := healthMonitor.StartMonitoring(ctx)
		assert.Error(t, err)
	})

	time.Sleep(2 * time.Second)

	t.Run("stop monitoring", func(t *testing.T) {
		healthMonitor.StopMonitoring()
		cancel()
	})
}