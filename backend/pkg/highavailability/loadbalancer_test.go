package highavailability

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestLoadBalancer_AddBackend(t *testing.T) {
	lb := NewLoadBalancer(nil)

	backend := &LBBackend{
		URL:    "http://localhost:8080",
		Weight: 10,
	}

	err := lb.AddBackend(backend)
	require.NoError(t, err)

	assert.Equal(t, 1, lb.GetBackendCount())
}

func TestLoadBalancer_AddBackend_Validation(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.AddBackend(nil)
	assert.Error(t, err)

	err = lb.AddBackend(&LBBackend{})
	assert.Error(t, err)
}

func TestLoadBalancer_RemoveBackend(t *testing.T) {
	lb := NewLoadBalancer(nil)

	backend := &LBBackend{
		URL:    "http://localhost:8080",
		Weight: 10,
	}

	err := lb.AddBackend(backend)
	require.NoError(t, err)

	err = lb.RemoveBackend("http://localhost:8080")
	require.NoError(t, err)

	assert.Equal(t, 0, lb.GetBackendCount())
}

func TestLoadBalancer_RemoveBackend_NotFound(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.RemoveBackend("http://localhost:8080")
	assert.Error(t, err)
}

func TestLoadBalancer_GetBackend_RoundRobin(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyRoundRobin)

	backends := []*LBBackend{
		{URL: "http://localhost:8080", Weight: 10},
		{URL: "http://localhost:8081", Weight: 10},
		{URL: "http://localhost:8082", Weight: 10},
	}

	for _, b := range backends {
		err := lb.AddBackend(b)
		require.NoError(t, err)
	}

	for i := 0; i < 6; i++ {
		backend, err := lb.GetBackend("127.0.0.1")
		require.NoError(t, err)
		assert.NotNil(t, backend)
	}
}

func TestLoadBalancer_GetBackend_WeightedRoundRobin(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyWeightedRoundRobin)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 3})
	require.NoError(t, err)
	err = lb.AddBackend(&LBBackend{URL: "http://localhost:8081", Weight: 1})
	require.NoError(t, err)

	backend, err := lb.GetBackend("127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestLoadBalancer_GetBackend_LeastConnection(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyLeastConnection)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)
	err = lb.AddBackend(&LBBackend{URL: "http://localhost:8081", Weight: 10})
	require.NoError(t, err)

	backend, err := lb.GetBackend("127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestLoadBalancer_GetBackend_IPHash(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyIPHash)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)
	err = lb.AddBackend(&LBBackend{URL: "http://localhost:8081", Weight: 10})
	require.NoError(t, err)

	urls := make(map[string]bool)
	for i := 0; i < 100; i++ {
		backend, err := lb.GetBackend("192.168.1.1")
		require.NoError(t, err)
		urls[backend.URL] = true
	}

	assert.LessOrEqual(t, len(urls), 2, "IP hash should select from limited backends")
}

func TestLoadBalancer_GetBackend_Random(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyRandom)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)

	backend, err := lb.GetBackend("127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestLoadBalancer_GetBackend_ConsistentHash(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyConsistentHash)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)
	err = lb.AddBackend(&LBBackend{URL: "http://localhost:8081", Weight: 10})
	require.NoError(t, err)

	backend, err := lb.GetBackend("client-key-1")
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestLoadBalancer_GetBackend_HealthBased(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyHealthBased)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10, Healthy: true})
	require.NoError(t, err)

	backend, err := lb.GetBackend("127.0.0.1")
	require.NoError(t, err)
	assert.NotNil(t, backend)
	assert.True(t, backend.Healthy)
}

func TestLoadBalancer_GetBackend_NoHealthyBackends(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.SetStrategy(StrategyRoundRobin)

	_, err := lb.GetBackend("127.0.0.1")
	assert.Error(t, err)
}

func TestLoadBalancer_RecordSuccess(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)

	backend, _ := lb.GetBackend("127.0.0.1")

	lb.RecordSuccess(backend, 100*time.Millisecond)

	stats := lb.GetStats()
	assert.Len(t, stats, 1)
}

func TestLoadBalancer_RecordFailure(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)

	backend, _ := lb.GetBackend("127.0.0.1")

	lb.RecordFailure(backend)
	lb.RecordFailure(backend)
	lb.RecordFailure(backend)

	assert.False(t, backend.Healthy)
}

func TestLoadBalancer_ReleaseBackend(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)

	backend, _ := lb.GetBackend("127.0.0.1")

	initialConns := backend.ActiveConns
	lb.ReleaseBackend(backend)
	assert.LessOrEqual(t, backend.ActiveConns, initialConns)
}

func TestLoadBalancer_UpdateBackendWeight(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	require.NoError(t, err)

	err = lb.UpdateBackendWeight("http://localhost:8080", 20)
	require.NoError(t, err)

	stats := lb.GetStats()
	assert.Equal(t, 20, stats[0].Weight)
}

func TestLoadBalancer_UpdateBackendWeight_NotFound(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.UpdateBackendWeight("http://localhost:8080", 20)
	assert.Error(t, err)
}

func TestLoadBalancer_SetBackendHealthy(t *testing.T) {
	lb := NewLoadBalancer(nil)

	err := lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10, Healthy: false})
	require.NoError(t, err)

	lb.SetBackendHealthy("http://localhost:8080", true)

	stats := lb.GetStats()
	assert.True(t, stats[0].Healthy)
}

func TestLoadBalancer_GetBackendCount(t *testing.T) {
	lb := NewLoadBalancer(nil)

	for i := 0; i < 5; i++ {
		err := lb.AddBackend(&LBBackend{URL: "http://localhost:808" + string(rune('0'+i)), Weight: 10})
		require.NoError(t, err)
	}

	assert.Equal(t, 5, lb.GetBackendCount())
}

func TestLoadBalancer_GetHealthyBackendCount(t *testing.T) {
	lb := NewLoadBalancer(nil)

	lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	lb.AddBackend(&LBBackend{URL: "http://localhost:8081", Weight: 10})
	lb.AddBackend(&LBBackend{URL: "http://localhost:8082", Weight: 10})
	lb.SetBackendHealthy("http://localhost:8082", false)

	assert.Equal(t, 2, lb.GetHealthyBackendCount())
}

func TestLoadBalancer_GetStrategy(t *testing.T) {
	lb := NewLoadBalancer(nil)

	assert.Equal(t, StrategyRoundRobin, lb.GetStrategy())

	lb.SetStrategy(StrategyLeastConnection)
	assert.Equal(t, StrategyLeastConnection, lb.GetStrategy())
}

func TestLoadBalancer_ConcurrentAccess(t *testing.T) {
	lb := NewLoadBalancer(nil)

	for i := 0; i < 5; i++ {
		err := lb.AddBackend(&LBBackend{URL: "http://localhost:808" + string(rune('0'+i)), Weight: 10})
		require.NoError(t, err)
	}

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, _ = lb.GetBackend("127.0.0.1")
		}()
	}
	wg.Wait()

	assert.Equal(t, 5, lb.GetBackendCount())
}

func TestLBHealthChecker_Results(t *testing.T) {
	config := &LoadBalancerConfig{
		HealthCheckPeriod:  10 * time.Millisecond,
		HealthCheckTimeout: 5 * time.Millisecond,
		HealthCheckPath:    "/health",
	}
	hc := NewLBHealthChecker(config)

	backends := []*LBBackend{
		{URL: "http://localhost:8080"},
		{URL: "http://localhost:8081"},
	}

	ctx, cancel := context.WithTimeout(context.Background(), 50*time.Millisecond)
	defer cancel()

	go hc.Start(ctx, backends)

	time.Sleep(100 * time.Millisecond)

	results := hc.GetAllResults()
	assert.Len(t, results, 2)
}

func TestFailoverManager_MarkFailed(t *testing.T) {
	fm := NewFailoverManager(nil)

	fm.MarkFailed("http://localhost:8080")
	fm.MarkFailed("http://localhost:8080")

	assert.True(t, fm.IsFailed("http://localhost:8080"))
}

func TestFailoverManager_MarkRecovered(t *testing.T) {
	fm := NewFailoverManager(nil)

	fm.MarkFailed("http://localhost:8080")
	fm.MarkRecovered("http://localhost:8080")

	assert.False(t, fm.IsFailed("http://localhost:8080"))
}

func TestFailoverManager_GetFailedBackends(t *testing.T) {
	fm := NewFailoverManager(nil)

	fm.MarkFailed("http://localhost:8080")
	fm.MarkFailed("http://localhost:8081")

	failed := fm.GetFailedBackends()
	assert.Len(t, failed, 2)
}

func TestFailoverManager_ShouldFailover(t *testing.T) {
	fm := NewFailoverManager(nil)

	assert.False(t, fm.ShouldFailover(2))

	fm.MarkFailed("http://localhost:8080")
	fm.MarkFailed("http://localhost:8081")

	assert.True(t, fm.ShouldFailover(2))
}

func TestHashString(t *testing.T) {
	hash1 := hashString("test")
	hash2 := hashString("test")
	hash3 := hashString("different")

	assert.Equal(t, hash1, hash2)
	assert.NotEqual(t, hash1, hash3)
}

func TestBinarySearchHash(t *testing.T) {
	arr := []uint32{1, 5, 10, 15, 20}

	assert.Equal(t, 0, binarySearchHash(arr, 1))
	assert.Equal(t, 2, binarySearchHash(arr, 10))
	assert.Equal(t, 4, binarySearchHash(arr, 20))
	assert.Equal(t, 0, binarySearchHash(arr, 0))
}

func TestBinarySearchHash_Empty(t *testing.T) {
	arr := []uint32{}
	assert.Equal(t, 0, binarySearchHash(arr, 10))
}

func TestSortUint32Slice(t *testing.T) {
	arr := []uint32{5, 3, 1, 4, 2}
	sortUint32Slice(arr)

	expected := []uint32{1, 2, 3, 4, 5}
	assert.Equal(t, expected, arr)
}

func TestProxyHandler(t *testing.T) {
	lb := NewLoadBalancer(nil)
	lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})

	handler := NewProxyHandler(lb, 30*time.Second, 3)
	assert.NotNil(t, handler)
}

func TestLoadBalancer_CalculateHealthScore(t *testing.T) {
	lb := NewLoadBalancer(nil)

	backend := &LBBackend{
		URL:     "http://localhost:8080",
		Weight:  10,
		Healthy: true,
	}

	lb.AddBackend(backend)

	lb.RecordSuccess(backend, 50*time.Millisecond)

	stats := lb.GetStats()
	assert.Greater(t, stats[0].HealthScore, 0.0)
}

func TestLoadBalancer_GetStats(t *testing.T) {
	lb := NewLoadBalancer(nil)

	lb.AddBackend(&LBBackend{URL: "http://localhost:8080", Weight: 10})
	lb.AddBackend(&LBBackend{URL: "http://localhost:8081", Weight: 5})

	stats := lb.GetStats()
	assert.Len(t, stats, 2)

	for _, stat := range stats {
		assert.NotEmpty(t, stat.URL)
	}
}
