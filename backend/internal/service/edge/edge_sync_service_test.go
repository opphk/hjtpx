package edge

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestEdgeSyncService_HealthCheck(t *testing.T) {
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
	ctx := context.Background()

	t.Run("health check success", func(t *testing.T) {
		result, err := syncService.HealthCheck(ctx, "node-001")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, "node-001", result.NodeID)
		assert.Equal(t, model.EdgeNodeStatusOnline, result.Status)
		assert.Greater(t, result.HealthScore, float64(80))
	})

	t.Run("health check with high load", func(t *testing.T) {
		nodeHighLoad := model.NewEdgeNode("node-002", "Node 002", "edge", "cn-east-1", "zone-a")
		nodeHighLoad.Status = model.EdgeNodeStatusOnline
		nodeHighLoad.LastHeartbeat = time.Now()
		nodeHighLoad.Capacity = model.EdgeCapacity{
			MaxRequestsPerSecond:  1000,
			MaxConcurrentRequests: 100,
			MemoryLimitMB:         4096,
			CPUCores:              4,
		}
		nodeHighLoad.CurrentLoad = model.EdgeLoadMetrics{
			CurrentRequestsPerSecond:   950,
			CurrentConcurrentRequests: 95,
			MemoryUsageMB:             3800,
			CPUUsagePercent:           95,
		}
		repo.CreateNode(context.Background(), nodeHighLoad)

		result, err := syncService.HealthCheck(ctx, "node-002")
		assert.NoError(t, err)
		assert.NotNil(t, result)
		assert.Less(t, result.HealthScore, float64(60))
		assert.Equal(t, model.EdgeNodeStatusDegraded, result.Status)
	})

	t.Run("health check node not found", func(t *testing.T) {
		result, err := syncService.HealthCheck(ctx, "nonexistent")
		assert.Error(t, err)
		assert.Nil(t, result)
	})
}

func TestEdgeSyncService_TriggerSync(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			MaxSyncBatchSize: 1000,
			CloudEndpoint:    "https://mock-cloud.example.com",
		},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	ctx := context.Background()

	t.Run("trigger sync success", func(t *testing.T) {
		err := syncService.TriggerSync(ctx, "node-001")
		assert.NoError(t, err)
	})

	t.Run("trigger sync with pending requests", func(t *testing.T) {
		req := model.NewEdgeVerificationRequest("node-001", "session-001", []byte("request"), []byte("response"), "success")
		repo.CreateVerificationRequest(ctx, req)

		err := syncService.TriggerSync(ctx, "node-001")
		assert.NoError(t, err)

		reqs, err := repo.GetUnsyncedRequests(ctx, "node-001", 10)
		assert.NoError(t, err)
		assert.Len(t, reqs, 0)
	})
}

func TestEdgeSyncService_StartAndStopScheduler(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			SyncIntervalSecs: 1,
		},
	}
	repo := newMockRepo()

	node := model.NewEdgeNode("node-001", "Node 001", "edge", "cn-east-1", "zone-a")
	node.Status = model.EdgeNodeStatusOnline
	node.LastHeartbeat = time.Now()
	repo.CreateNode(context.Background(), node)

	syncService := NewEdgeSyncService(repo, cfg)
	ctx, cancel := context.WithCancel(context.Background())

	t.Run("start scheduler", func(t *testing.T) {
		err := syncService.StartSyncScheduler(ctx)
		assert.NoError(t, err)
	})

	t.Run("start scheduler again", func(t *testing.T) {
		err := syncService.StartSyncScheduler(ctx)
		assert.Error(t, err)
	})

	time.Sleep(2 * time.Second)

	t.Run("stop scheduler", func(t *testing.T) {
		syncService.StopSyncScheduler()
		cancel()
	})
}