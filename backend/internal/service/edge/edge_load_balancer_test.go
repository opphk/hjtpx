package edge

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestEdgeLoadBalancer_SelectNode(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			LoadBalanceStrategy:    "least_load",
			HeartbeatIntervalSecs: 10,
		},
	}
	repo := newMockRepo()

	node1 := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node1.Status = model.EdgeNodeStatusOnline
	node1.LastHeartbeat = time.Now()
	node1.Capacity = model.EdgeCapacity{
		MaxRequestsPerSecond:  1000,
		MaxConcurrentRequests: 100,
		MemoryLimitMB:         4096,
		CPUCores:              4,
	}
	node1.CurrentLoad = model.EdgeLoadMetrics{
		CurrentRequestsPerSecond:   100,
		CurrentConcurrentRequests: 10,
		MemoryUsageMB:             1024,
		CPUUsagePercent:           20,
	}
	repo.CreateNode(context.Background(), node1)

	node2 := model.NewEdgeNode("node-002", "Node 002", "edge", "cn-east-1", "zone-a")
	node2.Status = model.EdgeNodeStatusOnline
	node2.LastHeartbeat = time.Now()
	node2.Capacity = model.EdgeCapacity{
		MaxRequestsPerSecond:  1000,
		MaxConcurrentRequests: 100,
		MemoryLimitMB:         4096,
		CPUCores:              4,
	}
	node2.CurrentLoad = model.EdgeLoadMetrics{
		CurrentRequestsPerSecond:   800,
		CurrentConcurrentRequests: 80,
		MemoryUsageMB:             3072,
		CPUUsagePercent:           80,
	}
	repo.CreateNode(context.Background(), node2)

	lb := NewEdgeLoadBalancer(repo, cfg)
	ctx := context.Background()

	t.Run("select with least load strategy", func(t *testing.T) {
		node, err := lb.SelectNode(ctx, "cn-east-1", "zone-a")
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Equal(t, "node-001", node.NodeID)
	})

	t.Run("select with round robin strategy", func(t *testing.T) {
		cfg.Edge.LoadBalanceStrategy = "round_robin"
		lb2 := NewEdgeLoadBalancer(repo, cfg)

		node1, err := lb2.SelectNode(ctx, "cn-east-1", "zone-a")
		assert.NoError(t, err)
		assert.NotNil(t, node1)

		node2, err := lb2.SelectNode(ctx, "cn-east-1", "zone-a")
		assert.NoError(t, err)
		assert.NotNil(t, node2)
	})

	t.Run("select with random strategy", func(t *testing.T) {
		cfg.Edge.LoadBalanceStrategy = "random"
		lb3 := NewEdgeLoadBalancer(repo, cfg)

		node, err := lb3.SelectNode(ctx, "cn-east-1", "zone-a")
		assert.NoError(t, err)
		assert.NotNil(t, node)
		assert.Contains(t, []string{"node-001", "node-002"}, node.NodeID)
	})

	t.Run("no available nodes", func(t *testing.T) {
		emptyRepo := newMockRepo()
		lb4 := NewEdgeLoadBalancer(emptyRepo, cfg)

		node, err := lb4.SelectNode(ctx, "cn-east-1", "zone-a")
		assert.Error(t, err)
		assert.Nil(t, node)
		assert.Equal(t, ErrNoAvailableNodes, err)
	})
}

func TestEdgeLoadBalancer_GetOnlineNodes(t *testing.T) {
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

	node2 := model.NewEdgeNode("node-002", "Node 002", "edge", "cn-east-1", "zone-a")
	node2.Status = model.EdgeNodeStatusOnline
	node2.LastHeartbeat = time.Now().Add(-30 * time.Second)
	repo.CreateNode(context.Background(), node2)

	node3 := model.NewEdgeNode("node-003", "Node 003", "edge", "cn-east-1", "zone-b")
	node3.Status = model.EdgeNodeStatusOnline
	node3.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node3)

	node4 := model.NewEdgeNode("node-004", "Node 004", "edge", "cn-east-1", "zone-a")
	node4.Status = model.EdgeNodeStatusOffline
	node4.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node4)

	lb := NewEdgeLoadBalancer(repo, cfg)
	ctx := context.Background()

	t.Run("get all online nodes", func(t *testing.T) {
		nodes, err := lb.GetOnlineNodes(ctx, "", "")
		assert.NoError(t, err)
		assert.Len(t, nodes, 2)
	})

	t.Run("get online nodes by region and zone", func(t *testing.T) {
		nodes, err := lb.GetOnlineNodes(ctx, "cn-east-1", "zone-a")
		assert.NoError(t, err)
		assert.Len(t, nodes, 1)
		assert.Equal(t, "node-001", nodes[0].NodeID)
	})
}

func TestEdgeLoadBalancer_UpdateNodeLoad(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	lb := NewEdgeLoadBalancer(repo, cfg)
	ctx := context.Background()

	load := model.EdgeLoadMetrics{
		CurrentRequestsPerSecond:   500,
		CurrentConcurrentRequests: 50,
		MemoryUsageMB:             2048,
		CPUUsagePercent:           50,
	}

	err := lb.UpdateNodeLoad(ctx, "node-001", load)
	assert.NoError(t, err)

	updatedNode, err := repo.GetNodeByNodeID(ctx, "node-001")
	assert.NoError(t, err)
	assert.Equal(t, 500, updatedNode.CurrentLoad.CurrentRequestsPerSecond)
	assert.Equal(t, 50, updatedNode.CurrentLoad.CurrentConcurrentRequests)
}