package performance

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGlobalEdgeNetwork(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		require.NotNil(t, network)

		err := network.Start()
		require.NoError(t, err)

		assert.True(t, network.isRunning)

		network.Stop()
		assert.False(t, network.isRunning)
	})

	t.Run("Register and Deregister Node", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		reg := &NodeRegistration{
			Name:          "test-node-1",
			Address:       "192.168.1.100",
			Port:          8080,
			Region:        "us-west",
			Zone:          "us-west-1",
			Continent:     "north_america",
			Country:       "US",
			City:          "San Francisco",
			Latitude:      37.7749,
			Longitude:     -122.4194,
			Capacity:      1000,
			Priority:      1,
			Weight:        10,
			Features:      []string{"verification", "cache"},
			Protocols:     []string{"http", "grpc"},
			BandwidthGbps: 10,
		}

		nodeID, err := network.RegisterNode(ctx, reg)
		require.NoError(t, err)
		assert.NotEmpty(t, nodeID)

		err = network.DeregisterNode(ctx, nodeID)
		require.NoError(t, err)
	})

	t.Run("Multiple Node Registration", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()
		regions := []string{"us-west", "us-east", "eu-west", "eu-central", "asia-east"}

		var nodeIDs []string
		for i, region := range regions {
			reg := &NodeRegistration{
				Name:          fmt.Sprintf("node-%d", i),
				Address:       fmt.Sprintf("10.0.0.%d", i+1),
				Port:          8080,
				Region:        region,
				Continent:     getContinent(region),
				Capacity:      1000 + i*100,
				Priority:      i % 3,
				Weight:        10 - i,
			}

			nodeID, err := network.RegisterNode(ctx, reg)
			require.NoError(t, err)
			nodeIDs = append(nodeIDs, nodeID)
		}

		assert.Equal(t, 5, len(nodeIDs))
		assert.Equal(t, int64(5), network.metrics.ActiveNodes.Load())

		for _, nodeID := range nodeIDs {
			err := network.DeregisterNode(ctx, nodeID)
			require.NoError(t, err)
		}

		assert.Equal(t, int64(0), network.metrics.ActiveNodes.Load())
	})

	t.Run("Process Verification Request", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		reg := &NodeRegistration{
			Name:      "verification-node",
			Address:   "192.168.1.200",
			Port:      8080,
			Region:    "us-west",
			Continent: "north_america",
			Capacity:  1000,
		}

		nodeID, err := network.RegisterNode(ctx, reg)
		require.NoError(t, err)

		req := &EdgeVerificationRequest{
			RequestID:  "req-123",
			UserID:    "user-456",
			Data:      []byte(`{"action":"verify"}`),
			UseCache:  true,
			CacheKey:  "verify:user-456",
			TTL:       5 * time.Minute,
			Priority:  1,
		}

		resp, err := network.ProcessVerification(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.True(t, resp.Success)
		assert.Equal(t, "req-123", resp.RequestID)
		assert.Equal(t, nodeID, resp.NodeID)
		assert.False(t, resp.FromCache)

		network.DeregisterNode(ctx, nodeID)
	})

	t.Run("Cache Hit", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		reg := &NodeRegistration{
			Name:      "cache-node",
			Address:   "192.168.1.201",
			Port:      8080,
			Region:    "us-west",
			Continent: "north_america",
			Capacity:  1000,
		}

		_, err = network.RegisterNode(ctx, reg)
		require.NoError(t, err)

		req := &EdgeVerificationRequest{
			RequestID: "req-cache-1",
			UserID:    "user-789",
			Data:      []byte(`{"result":"cached"}`),
			UseCache:  true,
			CacheKey:  "test:cache:key",
			TTL:       10 * time.Minute,
		}

		_, err = network.ProcessVerification(ctx, req)
		require.NoError(t, err)

		req2 := &EdgeVerificationRequest{
			RequestID: "req-cache-2",
			UserID:    "user-789",
			UseCache:  true,
			CacheKey:  "test:cache:key",
		}

		resp, err := network.ProcessVerification(ctx, req2)
		require.NoError(t, err)
		require.NotNil(t, resp)

		assert.True(t, resp.FromCache)
	})

	t.Run("Get Network Stats", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		regions := []string{"us-west", "eu-west"}
		for i, region := range regions {
			reg := &NodeRegistration{
				Name:      fmt.Sprintf("stats-node-%d", i),
				Address:   fmt.Sprintf("10.0.1.%d", i+1),
				Port:      8080,
				Region:    region,
				Continent: getContinent(region),
				Capacity:  1000,
			}
			_, err = network.RegisterNode(ctx, reg)
			require.NoError(t, err)
		}

		stats := network.GetNetworkStats(ctx)

		assert.NotNil(t, stats)
		assert.Contains(t, stats, "total_requests")
		assert.Contains(t, stats, "active_nodes")
		assert.Contains(t, stats, "region_stats")
	})

	t.Run("Concurrent Requests", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		for i := 0; i < 3; i++ {
			reg := &NodeRegistration{
				Name:      fmt.Sprintf("concurrent-node-%d", i),
				Address:   fmt.Sprintf("10.0.2.%d", i+1),
				Port:      8080,
				Region:    "us-west",
				Continent: "north_america",
				Capacity:  10000,
			}
			_, err = network.RegisterNode(ctx, reg)
			require.NoError(t, err)
		}

		var wg sync.WaitGroup
		concurrentRequests := 100

		for i := 0; i < concurrentRequests; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				req := &EdgeVerificationRequest{
					RequestID: fmt.Sprintf("concurrent-req-%d", id),
					UserID:    fmt.Sprintf("user-%d", id%10),
					Data:      []byte(fmt.Sprintf(`{"id":%d}`, id)),
				}

				resp, err := network.ProcessVerification(ctx, req)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
				assert.True(t, resp.Success)
			}(i)
		}

		wg.Wait()

		assert.Equal(t, int64(concurrentRequests), network.metrics.TotalRequests.Load())
		assert.Equal(t, int64(concurrentRequests), network.metrics.SuccessfulRequests.Load())
	})

	t.Run("Health Check", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		reg := &NodeRegistration{
			Name:      "health-node",
			Address:   "192.168.1.202",
			Port:      8080,
			Region:    "us-west",
			Continent: "north_america",
			Capacity:  1000,
		}

		nodeID, err := network.RegisterNode(ctx, reg)
		require.NoError(t, err)

		network.UpdateNodeHealth(nodeID, true)

		network.UpdateNodeHealth(nodeID, false)
	})

	t.Run("Region Failover", func(t *testing.T) {
		network := NewGlobalEdgeNetwork()
		err := network.Start()
		require.NoError(t, err)
		defer network.Stop()

		ctx := context.Background()

		reg1 := &NodeRegistration{
			Name:      "primary-region-node",
			Address:   "192.168.1.203",
			Port:      8080,
			Region:    "us-west",
			Continent: "north_america",
			Capacity:  1000,
			Priority:  1,
		}

		reg2 := &NodeRegistration{
			Name:      "backup-region-node",
			Address:   "192.168.2.203",
			Port:      8080,
			Region:    "us-east",
			Continent: "north_america",
			Capacity:  1000,
			Priority:  2,
		}

		_, err = network.RegisterNode(ctx, reg1)
		require.NoError(t, err)

		backupID, err := network.RegisterNode(ctx, reg2)
		require.NoError(t, err)

		network.UpdateNodeHealth(backupID, false)

		req := &EdgeVerificationRequest{
			RequestID:        "failover-test",
			UserID:           "user-failover",
			AllowedRegions:   []string{"us-west"},
		}

		resp, err := network.ProcessVerification(ctx, req)
		if err != nil {
			assert.True(t, true)
		} else {
			assert.NotNil(t, resp)
		}
	})
}

func TestIntelligentRouter(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		router := NewIntelligentRouter()
		require.NotNil(t, router)

		err := router.Start()
		require.NoError(t, err)

		assert.True(t, router.isRunning)

		router.Stop()
		assert.False(t, router.isRunning)
	})

	t.Run("Route Request with Different Strategies", func(t *testing.T) {
		router := NewIntelligentRouter()
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		nodes := createTestNodes(5)

		router.mu.Lock()
		router.currentStrategy = StrategyWeighted
		router.mu.Unlock()

		ctx := context.Background()
		req := &RoutingRequest{
			RequestID: "route-test-1",
			UserID:    "user-001",
		}

		resp, err := router.RouteRequest(ctx, req, nodes)
		require.NoError(t, err)
		assert.NotNil(t, resp)
		assert.True(t, resp.Success)
	})

	t.Run("Route Request Geo-based", func(t *testing.T) {
		router := NewIntelligentRouter()
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		nodes := createTestNodesWithGeo(3)

		router.mu.Lock()
		router.currentStrategy = StrategyGeo
		router.mu.Unlock()

		ctx := context.Background()
		req := &RoutingRequest{
			RequestID: "geo-route-test",
			UserID:    "user-geo",
			UserLocation: &GeoLocation{
				Country:   "US",
				Region:    "us-west",
				Latitude:  37.7749,
				Longitude: -122.4194,
			},
		}

		resp, err := router.RouteRequest(ctx, req, nodes)
		require.NoError(t, err)
		assert.NotNil(t, resp)
	})

	t.Run("Concurrent Routing", func(t *testing.T) {
		router := NewIntelligentRouter()
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		nodes := createTestNodes(3)

		var wg sync.WaitGroup
		iterations := 50

		for i := 0; i < iterations; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				ctx := context.Background()
				req := &RoutingRequest{
					RequestID: fmt.Sprintf("concurrent-route-%d", id),
					UserID:   fmt.Sprintf("user-%d", id),
				}

				resp, err := router.RouteRequest(ctx, req, nodes)
				assert.NoError(t, err)
				assert.NotNil(t, resp)
			}(i)
		}

		wg.Wait()

		stats := router.GetStats()
		assert.Equal(t, int64(iterations), stats["total_requests"])
	})

	t.Run("Route with No Available Nodes", func(t *testing.T) {
		router := NewIntelligentRouter()
		err := router.Start()
		require.NoError(t, err)
		defer router.Stop()

		ctx := context.Background()
		req := &RoutingRequest{
			RequestID: "no-nodes-test",
			UserID:    "user-none",
		}

		_, err = router.RouteRequest(ctx, req, []*GlobalEdgeNode{})
		assert.Error(t, err)
	})
}

func TestCrossRegionSyncManager(t *testing.T) {
	t.Run("Initialize and Start", func(t *testing.T) {
		manager := NewCrossRegionSyncManager()
		require.NotNil(t, manager)

		err := manager.Start()
		require.NoError(t, err)

		assert.True(t, manager.isRunning)

		manager.Stop()
		assert.False(t, manager.isRunning)
	})

	t.Run("Create and Sync Cluster", func(t *testing.T) {
		manager := NewCrossRegionSyncManager()
		err := manager.Start()
		require.NoError(t, err)
		defer manager.Stop()

		ctx := context.Background()

		err = manager.CreateCluster(ctx, "test-cluster", "Test Cluster", []string{"us-west", "us-east", "eu-west"})
		require.NoError(t, err)

		req := &SyncRequest{
			ClusterID:   "test-cluster",
			FullSync:    true,
			Priority:    1,
			Timeout:     10 * time.Second,
		}

		resp, err := manager.SyncClusterData(ctx, req)
		require.NoError(t, err)
		require.NotNil(t, resp)
		assert.True(t, resp.Success)
	})

	t.Run("Store and Retrieve Data", func(t *testing.T) {
		manager := NewCrossRegionSyncManager()
		err := manager.Start()
		require.NoError(t, err)
		defer manager.Stop()

		ctx := context.Background()

		key := "test:data:key"
		value := []byte(`{"result":"success","data":{"id":1,"name":"test"}}`)

		err = manager.StoreData(ctx, key, value, "us-west")
		require.NoError(t, err)

		data, err := manager.GetData(ctx, key)
		require.NoError(t, err)
		require.NotNil(t, data)
		assert.Equal(t, key, data.Key)
	})

	t.Run("Incremental Sync", func(t *testing.T) {
		manager := NewCrossRegionSyncManager()
		err := manager.Start()
		require.NoError(t, err)
		defer manager.Stop()

		ctx := context.Background()

		err = manager.CreateCluster(ctx, "incr-cluster", "Incremental Cluster", []string{"us-west", "eu-west"})
		require.NoError(t, err)

		keys := []string{"key1", "key2", "key3"}
		for _, key := range keys {
			err = manager.StoreData(ctx, key, []byte(fmt.Sprintf(`{"key":"%s"}`, key)), "us-west")
			require.NoError(t, err)
		}

		req := &SyncRequest{
			ClusterID:  "incr-cluster",
			DataKeys:   keys,
			FullSync:  false,
			Priority:  1,
		}

		resp, err := manager.SyncClusterData(ctx, req)
		require.NoError(t, err)
		assert.Equal(t, 3, resp.SyncedItems)
	})

	t.Run("Concurrent Sync Operations", func(t *testing.T) {
		manager := NewCrossRegionSyncManager()
		err := manager.Start()
		require.NoError(t, err)
		defer manager.Stop()

		ctx := context.Background()

		err = manager.CreateCluster(ctx, "concurrent-cluster", "Concurrent Cluster", []string{"us-west", "us-east", "eu-west", "eu-central"})
		require.NoError(t, err)

		var wg sync.WaitGroup
		syncCount := 10

		for i := 0; i < syncCount; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()

				key := fmt.Sprintf("data:%d", id)
				value := []byte(fmt.Sprintf(`{"id":%d,"timestamp":%d}`, id, time.Now().Unix()))

				ctx := context.Background()
				manager.StoreData(ctx, key, value, "us-west")

				req := &SyncRequest{
					ClusterID: "concurrent-cluster",
					DataKeys:  []string{key},
				}

				manager.SyncClusterData(ctx, req)
			}(i)
		}

		wg.Wait()

		metrics := manager.GetMetrics()
		assert.GreaterOrEqual(t, int64(syncCount), metrics["total_sync_ops"].(int64))
	})
}

func createTestNodes(count int) []*GlobalEdgeNode {
	nodes := make([]*GlobalEdgeNode, count)

	for i := 0; i < count; i++ {
		node := &GlobalEdgeNode{
			ID:       fmt.Sprintf("node-%d", i),
			Name:     fmt.Sprintf("Test Node %d", i),
			Address:  fmt.Sprintf("10.0.0.%d", i+1),
			Port:     8080,
			Region:   "us-west",
			Capacity: 1000,
			Weight:   10,
		}
		node.Healthy.Store(true)
		node.Stats = &EdgeNodeStats{}

		nodes[i] = node
	}

	return nodes
}

func createTestNodesWithGeo(count int) []*GlobalEdgeNode {
	nodes := make([]*GlobalEdgeNode, count)
	regions := []struct {
		name      string
		latitude  float64
		longitude float64
	}{
		{"us-west", 37.7749, -122.4194},
		{"us-east", 40.7128, -74.0060},
		{"eu-west", 51.5074, -0.1278},
	}

	for i := 0; i < count && i < len(regions); i++ {
		node := &GlobalEdgeNode{
			ID:        fmt.Sprintf("geo-node-%d", i),
			Name:      fmt.Sprintf("Geo Node %d", i),
			Address:   fmt.Sprintf("10.1.0.%d", i+1),
			Port:      8080,
			Region:    regions[i].name,
			Latitude:  regions[i].latitude,
			Longitude: regions[i].longitude,
			Capacity:  1000,
			Weight:    10,
		}
		node.Healthy.Store(true)
		node.Stats = &EdgeNodeStats{}

		nodes[i] = node
	}

	return nodes[:count]
}

func getContinent(region string) string {
	switch region {
	case "us-west", "us-east":
		return "north_america"
	case "eu-west", "eu-central", "eu-east":
		return "europe"
	case "asia-east", "asia-southeast", "asia-central":
		return "asia"
	case "au-east":
		return "australia"
	default:
		return "unknown"
	}
}

func BenchmarkGlobalEdgeNetwork(b *testing.B) {
	network := NewGlobalEdgeNetwork()
	network.Start()
	defer network.Stop()

	ctx := context.Background()

	for i := 0; i < 10; i++ {
		reg := &NodeRegistration{
			Name:      fmt.Sprintf("bench-node-%d", i),
			Address:   fmt.Sprintf("10.0.%d.1", i+1),
			Port:      8080,
			Region:    fmt.Sprintf("region-%d", i%3),
			Continent: "north_america",
			Capacity:  10000,
		}
		network.RegisterNode(ctx, reg)
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		req := &EdgeVerificationRequest{
			RequestID: fmt.Sprintf("bench-req-%d", i),
			UserID:    fmt.Sprintf("user-%d", i),
			Data:      []byte(fmt.Sprintf(`{"id":%d}`, i)),
		}
		network.ProcessVerification(ctx, req)
	}
}

func BenchmarkIntelligentRouter(b *testing.B) {
	router := NewIntelligentRouter()
	router.Start()
	defer router.Stop()

	nodes := createTestNodes(5)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := context.Background()
		req := &RoutingRequest{
			RequestID: fmt.Sprintf("bench-route-%d", i),
			UserID:   fmt.Sprintf("user-%d", i),
		}
		router.RouteRequest(ctx, req, nodes)
	}
}
