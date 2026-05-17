package highavailability

import (
	"context"
	"sync"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestInstanceRegistry_Register(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	instance := &InstanceInfo{
		ID:   "test-instance-1",
		Name: "test-instance",
		Host: "localhost",
		Port: 8080,
	}

	err := registry.Register(instance)
	require.NoError(t, err)

	retrieved, ok := registry.GetInstance("test-instance-1")
	assert.True(t, ok)
	assert.Equal(t, "test-instance-1", retrieved.ID)
	assert.Equal(t, "test-instance", retrieved.Name)
}

func TestInstanceRegistry_Unregister(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	instance := &InstanceInfo{
		ID:   "test-instance-1",
		Name: "test-instance",
	}

	err := registry.Register(instance)
	require.NoError(t, err)

	err = registry.Unregister("test-instance-1")
	require.NoError(t, err)

	_, ok := registry.GetInstance("test-instance-1")
	assert.False(t, ok)
}

func TestInstanceRegistry_GetAllInstances(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	for i := 0; i < 5; i++ {
		instance := &InstanceInfo{
			ID:   "test-instance-" + string(rune('1'+i)),
			Name: "test-instance",
		}
		err := registry.Register(instance)
		require.NoError(t, err)
	}

	instances := registry.GetAllInstances()
	assert.Len(t, instances, 5)
}

func TestInstanceRegistry_GetActiveInstances(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	instances := []*InstanceInfo{
		{ID: "1", Name: "instance1", State: InstanceStateActive},
		{ID: "2", Name: "instance2", State: InstanceStateActive},
		{ID: "3", Name: "instance3", State: InstanceStateStandby},
		{ID: "4", Name: "instance4", State: InstanceStateOffline},
	}

	for _, inst := range instances {
		err := registry.Register(inst)
		require.NoError(t, err)
	}

	activeInstances := registry.GetActiveInstances()
	assert.Len(t, activeInstances, 2)
}

func TestInstanceRegistry_UpdateHeartbeat(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	instance := &InstanceInfo{
		ID:   "test-instance-1",
		Name: "test-instance",
	}

	err := registry.Register(instance)
	require.NoError(t, err)

	time.Sleep(10 * time.Millisecond)

	err = registry.UpdateHeartbeat("test-instance-1")
	require.NoError(t, err)

	retrieved, _ := registry.GetInstance("test-instance-1")
	assert.True(t, retrieved.LastHeartbeat.After(instance.LastHeartbeat) || retrieved.LastHeartbeat.Equal(instance.LastHeartbeat))
}

func TestInstanceRegistry_UpdateState(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	instance := &InstanceInfo{
		ID:   "test-instance-1",
		Name: "test-instance",
	}

	err := registry.Register(instance)
	require.NoError(t, err)

	err = registry.UpdateState("test-instance-1", InstanceStateDraining)
	require.NoError(t, err)

	retrieved, _ := registry.GetInstance("test-instance-1")
	assert.Equal(t, InstanceStateDraining, retrieved.State)
}

func TestInstanceRegistry_GetInstanceCount(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	for i := 0; i < 3; i++ {
		instance := &InstanceInfo{
			ID:   "test-instance-" + string(rune('1'+i)),
			Name: "test-instance",
		}
		err := registry.Register(instance)
		require.NoError(t, err)
	}

	assert.Equal(t, 3, registry.GetInstanceCount())
}

func TestInstanceRegistry_GetActiveCount(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	instances := []*InstanceInfo{
		{ID: "1", Name: "instance1", State: InstanceStateActive},
		{ID: "2", Name: "instance2", State: InstanceStateActive},
		{ID: "3", Name: "instance3", State: InstanceStateStandby},
	}

	for _, inst := range instances {
		err := registry.Register(inst)
		require.NoError(t, err)
	}

	assert.Equal(t, 2, registry.GetActiveCount())
}

func TestInstanceRegistry_GetInstancesByTag(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	instances := []*InstanceInfo{
		{ID: "1", Name: "instance1", Tags: []string{"api", "v1"}},
		{ID: "2", Name: "instance2", Tags: []string{"api", "v2"}},
		{ID: "3", Name: "instance3", Tags: []string{"web"}},
	}

	for _, inst := range instances {
		err := registry.Register(inst)
		require.NoError(t, err)
	}

	apiInstances := registry.GetInstancesByTag("api")
	assert.Len(t, apiInstances, 2)
}

func TestInstanceRegistry_GetInstancesByRegion(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	instances := []*InstanceInfo{
		{ID: "1", Name: "instance1", Region: "us-east"},
		{ID: "2", Name: "instance2", Region: "us-east"},
		{ID: "3", Name: "instance3", Region: "eu-west"},
	}

	for _, inst := range instances {
		err := registry.Register(inst)
		require.NoError(t, err)
	}

	regionInstances := registry.GetInstancesByRegion("us-east")
	assert.Len(t, regionInstances, 2)
}

func TestInstanceRegistry_SetSelfID(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	registry.SetSelfID("my-instance-id")
	assert.Equal(t, "my-instance-id", registry.GetSelfID())
}

func TestInstanceRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			instance := &InstanceInfo{
				ID:   "test-instance-" + string(rune('0'+id)),
				Name: "test-instance",
			}
			_ = registry.Register(instance)
		}(i)
	}
	wg.Wait()

	assert.Equal(t, 10, registry.GetInstanceCount())
}

func TestInstanceSelector_WeightedRoundRobin(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	for i := 0; i < 3; i++ {
		registry.Register(&InstanceInfo{
			ID:     string(rune('1' + i)),
			Name:   "instance",
			Weight: i + 1,
		})
	}

	selector := NewInstanceSelector(registry, InstanceStrategyWeightedRR)
	ctx := context.Background()

	backend, err := selector.Select(ctx)
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestInstanceSelector_LeastConnections(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	for i := 0; i < 3; i++ {
		registry.Register(&InstanceInfo{
			ID:   string(rune('1' + i)),
			Name: "instance",
		})
	}

	selector := NewInstanceSelector(registry, InstanceStrategyLeastConn)
	ctx := context.Background()

	backend, err := selector.Select(ctx)
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestInstanceSelector_Random(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	for i := 0; i < 5; i++ {
		registry.Register(&InstanceInfo{
			ID:   string(rune('1' + i)),
			Name: "instance",
		})
	}

	selector := NewInstanceSelector(registry, InstanceStrategyRandom)
	ctx := context.Background()

	backend, err := selector.Select(ctx)
	require.NoError(t, err)
	assert.NotNil(t, backend)
}

func TestInstanceSelector_Priority(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	registry.Register(&InstanceInfo{ID: "1", Name: "low", Priority: 1})
	registry.Register(&InstanceInfo{ID: "2", Name: "high", Priority: 100})

	selector := NewInstanceSelector(registry, InstanceStrategyPriority)
	ctx := context.Background()

	backend, err := selector.Select(ctx)
	require.NoError(t, err)
	assert.Equal(t, "high", backend.Name)
}

func TestInstanceSelector_SetStrategy(t *testing.T) {
	registry := NewInstanceRegistry(30 * time.Second)
	selector := NewInstanceSelector(registry, InstanceStrategyLeastConn)

	selector.SetStrategy(InstanceStrategyLeastConn)
	assert.Equal(t, InstanceStrategyLeastConn, selector.strategy)
}

func TestInstanceMetrics_RecordRequest(t *testing.T) {
	metrics := NewInstanceMetrics("test-instance")

	metrics.RecordRequest()
	metrics.RecordRequest()
	metrics.RecordRequest()

	stats := metrics.GetStats()
	assert.Equal(t, uint64(3), stats.RequestCount)
}

func TestInstanceMetrics_RecordSuccess(t *testing.T) {
	metrics := NewInstanceMetrics("test-instance")

	metrics.RecordSuccess(100 * time.Millisecond)
	metrics.RecordSuccess(200 * time.Millisecond)

	stats := metrics.GetStats()
	assert.Equal(t, uint64(2), stats.SuccessCount)
	assert.True(t, stats.AvgLatency > 0)
}

func TestInstanceMetrics_RecordError(t *testing.T) {
	metrics := NewInstanceMetrics("test-instance")

	metrics.RecordError()

	stats := metrics.GetStats()
	assert.Equal(t, uint64(1), stats.ErrorCount)
}

func TestInstanceMetrics_CalculateSuccessRate(t *testing.T) {
	metrics := NewInstanceMetrics("test-instance")

	for i := 0; i < 10; i++ {
		metrics.RecordRequest()
		if i < 8 {
			metrics.RecordSuccess(100 * time.Millisecond)
		} else {
			metrics.RecordError()
		}
	}

	stats := metrics.GetStats()
	assert.InDelta(t, 80.0, stats.SuccessRate, 1.0)
}

func TestConfigConsistencyManager_SetConfig(t *testing.T) {
	manager := NewConfigConsistencyManager()

	manager.SetConfig("key1", "value1", "source1")
	manager.SetConfig("key2", 123, "source2")

	value, ok := manager.GetConfig("key1")
	assert.True(t, ok)
	assert.Equal(t, "value1", value)

	value2, ok := manager.GetConfig("key2")
	assert.True(t, ok)
	assert.Equal(t, 123, value2)
}

func TestConfigConsistencyManager_GetConfigVersion(t *testing.T) {
	manager := NewConfigConsistencyManager()

	manager.SetConfig("key1", "value1", "source1")
	v1, _ := manager.GetConfigVersion("key1")

	manager.SetConfig("key1", "value2", "source2")
	v2, _ := manager.GetConfigVersion("key1")

	assert.Greater(t, v2, v1)
}

func TestConfigConsistencyManager_GetAllConfigs(t *testing.T) {
	manager := NewConfigConsistencyManager()

	manager.SetConfig("key1", "value1", "source1")
	manager.SetConfig("key2", "value2", "source2")

	configs := manager.GetAllConfigs()
	assert.Len(t, configs, 2)
}

func TestConfigConsistencyManager_GetGlobalVersion(t *testing.T) {
	manager := NewConfigConsistencyManager()

	initialVersion := manager.GetGlobalVersion()

	manager.SetConfig("key1", "value1", "source1")
	manager.SetConfig("key2", "value2", "source2")

	assert.Greater(t, manager.GetGlobalVersion(), initialVersion)
}

func TestLockManager_Acquire(t *testing.T) {
	manager := NewLockManager(10 * time.Second)

	result := manager.Acquire("lock1", "holder1")
	assert.True(t, result)

	result = manager.Acquire("lock1", "holder2")
	assert.False(t, result)
}

func TestLockManager_Release(t *testing.T) {
	manager := NewLockManager(10 * time.Second)

	manager.Acquire("lock1", "holder1")

	result := manager.Release("lock1", "holder1")
	assert.True(t, result)

	result = manager.Release("lock1", "holder1")
	assert.False(t, result)
}

func TestLockManager_IsLocked(t *testing.T) {
	manager := NewLockManager(10 * time.Second)

	assert.False(t, manager.IsLocked("lock1"))

	manager.Acquire("lock1", "holder1")
	assert.True(t, manager.IsLocked("lock1"))

	manager.Release("lock1", "holder1")
	assert.False(t, manager.IsLocked("lock1"))
}

func TestLockManager_Extend(t *testing.T) {
	manager := NewLockManager(100 * time.Millisecond)

	manager.Acquire("lock1", "holder1")

	time.Sleep(50 * time.Millisecond)

	result := manager.Extend("lock1", "holder1", 100*time.Millisecond)
	assert.True(t, result)
}

func TestSessionAffinityManager_SetSessionInstance(t *testing.T) {
	manager := NewSessionAffinityManager(30 * time.Minute)

	manager.SetSessionInstance("session1", "instance1")
	manager.SetSessionInstance("session2", "instance2")

	instance, ok := manager.GetSessionInstance("session1")
	assert.True(t, ok)
	assert.Equal(t, "instance1", instance)
}

func TestSessionAffinityManager_RemoveSession(t *testing.T) {
	manager := NewSessionAffinityManager(30 * time.Minute)

	manager.SetSessionInstance("session1", "instance1")
	manager.RemoveSession("session1")

	_, ok := manager.GetSessionInstance("session1")
	assert.False(t, ok)
}

func TestSessionAffinityManager_Clear(t *testing.T) {
	manager := NewSessionAffinityManager(30 * time.Minute)

	manager.SetSessionInstance("session1", "instance1")
	manager.SetSessionInstance("session2", "instance2")

	manager.Clear()

	_, ok1 := manager.GetSessionInstance("session1")
	_, ok2 := manager.GetSessionInstance("session2")
	assert.False(t, ok1)
	assert.False(t, ok2)
}
