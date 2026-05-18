package redis

import (
	"context"
	"sync"
	"testing"
	"time"
)

func TestWarmupPolicy(t *testing.T) {
	policies := []WarmupPolicy{
		WarmupPolicyEager,
		WarmupPolicyLazy,
		WarmupPolicyScheduled,
		WarmupPolicyAdaptive,
		WarmupPolicyOnDemand,
	}
	
	expected := []int{0, 1, 2, 3, 4}
	for i, policy := range policies {
		if int(policy) != expected[i] {
			t.Errorf("WarmupPolicy[%d] = %d, want %d", i, policy, expected[i])
		}
	}
}

func TestWarmupPriority(t *testing.T) {
	priorities := []WarmupPriority{
		WarmupPriorityCritical,
		WarmupPriorityHigh,
		WarmupPriorityNormal,
		WarmupPriorityLow,
	}
	
	expected := []int{0, 1, 2, 3}
	for i, priority := range priorities {
		if int(priority) != expected[i] {
			t.Errorf("WarmupPriority[%d] = %d, want %d", i, priority, expected[i])
		}
	}
}

func TestWarmupItem(t *testing.T) {
	item := &WarmupItem{
		Key:   "test_key",
		Value: []byte("test_value"),
		TTL:   5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("loaded"), nil
		},
	}
	
	if item.Key != "test_key" {
		t.Errorf("Key = %q, want %q", item.Key, "test_key")
	}
	
	if item.TTL != 5*time.Minute {
		t.Errorf("TTL = %v, want %v", item.TTL, 5*time.Minute)
	}
	
	if item.Loader == nil {
		t.Error("Loader should not be nil")
	}
}

func TestCacheWarmupTask(t *testing.T) {
	task := &CacheWarmupTask{
		Name:        "test_task",
		Key:         "warmup:test",
		Priority:    WarmupPriorityHigh,
		Policy:      WarmupPolicyEager,
		TTL:         10 * time.Minute,
		Frequency:   5 * time.Minute,
		MaxRetries:  3,
		RetryCount:  0,
		Enabled:     true,
		Stats:       NewWarmupStats(),
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
	}
	
	if task.Name != "test_task" {
		t.Errorf("Name = %q, want %q", task.Name, "test_task")
	}
	
	if task.Priority != WarmupPriorityHigh {
		t.Errorf("Priority = %d, want %d", task.Priority, WarmupPriorityHigh)
	}
	
	if !task.Enabled {
		t.Error("Enabled should be true")
	}
	
	if task.Stats == nil {
		t.Error("Stats should not be nil")
	}
}

func TestWarmupStats(t *testing.T) {
	stats := NewWarmupStats()
	
	if stats == nil {
		t.Fatal("NewWarmupStats should not return nil")
	}
	
	stats.SuccessCount.Add(10)
	stats.FailureCount.Add(2)
	stats.TotalRuns.Add(12)
	
	if stats.SuccessCount.Load() != 10 {
		t.Errorf("SuccessCount = %d, want 10", stats.SuccessCount.Load())
	}
	
	if stats.FailureCount.Load() != 2 {
		t.Errorf("FailureCount = %d, want 2", stats.FailureCount.Load())
	}
}

func TestCacheWarmupManagerCreation(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	if manager == nil {
		t.Fatal("NewCacheWarmupManager should not return nil")
	}
	
	if manager.policy != WarmupPolicyAdaptive {
		t.Errorf("Policy = %d, want %d", manager.policy, WarmupPolicyAdaptive)
	}
	
	if manager.concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", manager.concurrency)
	}
	
	if manager.batchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", manager.batchSize)
	}
}

func TestCacheWarmupManagerAddTask(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	
	tasks := manager.GetTasks()
	if len(tasks) != 1 {
		t.Errorf("Tasks count after add = %d, want 1", len(tasks))
	}
}

func TestCacheWarmupManagerRemoveTask(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	manager.RemoveTask("test_task")
	
	tasks := manager.GetTasks()
	if len(tasks) != 0 {
		t.Errorf("Tasks count after remove = %d, want 0", len(tasks))
	}
}

func TestCacheWarmupManagerEnableDisableTask(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	manager.EnableTask("test_task")
	
	tasks := manager.GetTasks()
	if len(tasks) == 0 || !tasks[0].Enabled {
		t.Error("Task should be enabled")
	}
	
	manager.DisableTask("test_task")
	tasks = manager.GetTasks()
	if len(tasks) == 0 || tasks[0].Enabled {
		t.Error("Task should be disabled")
	}
}

func TestCacheWarmupManagerStartStop(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 1 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	manager.Start()
	manager.Stop()
	
	if manager.running {
		t.Error("Manager should not be running after Stop")
	}
}

func TestCacheWarmupManagerGetTaskStats(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	stats := manager.GetTaskStats()
	
	if len(stats) != 1 {
		t.Errorf("Expected 1 stats entry, got %d", len(stats))
	}
	
	if stats["test_task"] == nil {
		t.Error("Stats for test_task should exist")
	}
}

func TestCacheWarmupManagerResetTaskStats(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	manager.ResetTaskStats("test_task")
	
	stats := manager.GetTaskStats()
	if stats["test_task"] == nil {
		t.Error("Stats should be reset but not nil")
	}
}

func TestCacheWarmupManagerGetWarmupStatus(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    true,
		MaxRetries: 3,
	}
	
	manager.AddTask(task)
	status := manager.GetWarmupStatus()
	
	if status["running"] != false {
		t.Errorf("Running status = %v, want false", status["running"])
	}
	
	if status["total_tasks"] != 1 {
		t.Errorf("Total tasks = %v, want 1", status["total_tasks"])
	}
	
	if status["enabled_tasks"] != 1 {
		t.Errorf("Enabled tasks = %v, want 1", status["enabled_tasks"])
	}
}

func TestCacheWarmupManagerPauseResume(t *testing.T) {
	manager := NewCacheWarmupManager(nil)
	
	manager.Pause()
	if manager.running {
		t.Error("Manager should be paused")
	}
	
	manager.Resume()
	if !manager.running {
		t.Error("Manager should be running after resume")
	}
}

func TestAdaptiveWarmupPolicy(t *testing.T) {
	policy := NewAdaptiveWarmupPolicy(100)
	
	if policy == nil {
		t.Fatal("NewAdaptiveWarmupPolicy should not return nil")
	}
	
	if policy.threshold != 100 {
		t.Errorf("Threshold = %d, want 100", policy.threshold)
	}
}

func TestAdaptiveWarmupPolicyRecordAccess(t *testing.T) {
	policy := NewAdaptiveWarmupPolicy(100)
	
	policy.RecordAccess("hot_key")
	policy.RecordAccess("hot_key")
	
	count := policy.tracker.GetAccessCount("hot_key")
	if count != 2 {
		t.Errorf("Access count = %d, want 2", count)
	}
}

func TestSmartWarmupStrategy(t *testing.T) {
	strategy := NewSmartWarmupStrategy(nil, 100)
	
	if strategy == nil {
		t.Fatal("NewSmartWarmupStrategy should not return nil")
	}
	
	if strategy.threshold != 100 {
		t.Errorf("Threshold = %d, want 100", strategy.threshold)
	}
}

func TestSmartWarmupStrategyRecordAccess(t *testing.T) {
	strategy := NewSmartWarmupStrategy(nil, 100)
	
	for i := 0; i < 150; i++ {
		strategy.RecordAccess("hot_key")
	}
	
	recommendations := strategy.GetWarmupRecommendations()
	if len(recommendations) != 1 {
		t.Errorf("Recommendations count = %d, want 1", len(recommendations))
	}
	
	if recommendations[0] != "hot_key" {
		t.Errorf("Recommendation = %q, want %q", recommendations[0], "hot_key")
	}
}

func TestAccessTracker(t *testing.T) {
	tracker := NewAccessTracker()
	
	if tracker == nil {
		t.Fatal("NewAccessTracker should not return nil")
	}
	
	tracker.RecordAccess("key1")
	tracker.RecordAccess("key1")
	tracker.RecordAccess("key2")
	
	count1 := tracker.GetAccessCount("key1")
	count2 := tracker.GetAccessCount("key2")
	
	if count1 != 2 {
		t.Errorf("key1 count = %d, want 2", count1)
	}
	
	if count2 != 1 {
		t.Errorf("key2 count = %d, want 1", count2)
	}
}

func TestAccessTrackerHotKeys(t *testing.T) {
	tracker := NewAccessTracker()
	
	for i := 0; i < 10; i++ {
		tracker.RecordAccess("hot_key")
	}
	for i := 0; i < 5; i++ {
		tracker.RecordAccess("warm_key")
	}
	
	hotKeys := tracker.GetHotKeys(8)
	if len(hotKeys) != 1 {
		t.Errorf("Hot keys count = %d, want 1", len(hotKeys))
	}
	
	if hotKeys[0] != "hot_key" {
		t.Errorf("Hot key = %q, want %q", hotKeys[0], "hot_key")
	}
}

func TestBatchWarmupProcessor(t *testing.T) {
	processor := NewBatchWarmupProcessor(100, 5)
	
	if processor == nil {
		t.Fatal("NewBatchWarmupProcessor should not return nil")
	}
	
	if processor.batchSize != 100 {
		t.Errorf("batchSize = %d, want 100", processor.batchSize)
	}
	
	if processor.workers != 5 {
		t.Errorf("workers = %d, want 5", processor.workers)
	}
}

func TestBatchWarmupProcessorWithDefaults(t *testing.T) {
	processor := NewBatchWarmupProcessor(0, 0)
	
	if processor.batchSize != 100 {
		t.Errorf("Default batchSize = %d, want 100", processor.batchSize)
	}
	
	if processor.workers != 5 {
		t.Errorf("Default workers = %d, want 5", processor.workers)
	}
}

func TestDefaultWarmupConfig(t *testing.T) {
	config := DefaultWarmupConfig
	
	if config.Policy != WarmupPolicyAdaptive {
		t.Errorf("Policy = %d, want %d", config.Policy, WarmupPolicyAdaptive)
	}
	
	if config.Concurrency != 5 {
		t.Errorf("Concurrency = %d, want 5", config.Concurrency)
	}
	
	if config.BatchSize != 100 {
		t.Errorf("BatchSize = %d, want 100", config.BatchSize)
	}
	
	if !config.Enabled {
		t.Error("Enabled should be true")
	}
}

func TestInitAndGetCacheWarmupManager(t *testing.T) {
	globalWarmupManager = nil
	globalWarmupOnce = sync.Once{}
	
	InitCacheWarmupManager(nil)
	manager := GetCacheWarmupManager()
	
	if manager == nil {
		t.Fatal("GetCacheWarmupManager should not return nil")
	}
	
	manager2 := GetCacheWarmupManager()
	if manager != manager2 {
		t.Error("GetCacheWarmupManager should return the same instance")
	}
}

func TestWarmupScheduler(t *testing.T) {
	scheduler := NewWarmupScheduler(nil)
	
	if scheduler == nil {
		t.Fatal("NewWarmupScheduler should not return nil")
	}
	
	if scheduler.maxWorkers != 10 {
		t.Errorf("maxWorkers = %d, want 10", scheduler.maxWorkers)
	}
	
	if scheduler.priorityMode != "adaptive" {
		t.Errorf("priorityMode = %q, want %q", scheduler.priorityMode, "adaptive")
	}
}

func TestWarmupSchedulerAddRemoveTask(t *testing.T) {
	scheduler := NewWarmupScheduler(nil)
	
	task := &CacheWarmupTask{
		Name:      "scheduled_task",
		Key:       "warmup:scheduled",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("scheduled_data"), nil
		},
		Enabled:    false,
		MaxRetries: 3,
	}
	
	scheduler.AddScheduledTask(task, 5*time.Minute)
	
	tasks := scheduler.GetScheduledTasks()
	if len(tasks) != 1 {
		t.Errorf("Scheduled tasks count = %d, want 1", len(tasks))
	}
	
	scheduler.RemoveScheduledTask("scheduled_task")
	tasks = scheduler.GetScheduledTasks()
	if len(tasks) != 0 {
		t.Errorf("Scheduled tasks count after remove = %d, want 0", len(tasks))
	}
}

func TestWarmupSchedulerStartStop(t *testing.T) {
	scheduler := NewWarmupScheduler(nil)
	scheduler.Start()
	scheduler.Stop()
	
	if scheduler.running {
		t.Error("Scheduler should not be running after Stop")
	}
}

func TestWarmupSchedulerPauseResumeTask(t *testing.T) {
	scheduler := NewWarmupScheduler(nil)
	
	task := &CacheWarmupTask{
		Name:      "test_task",
		Key:       "warmup:test",
		TTL:       10 * time.Minute,
		Frequency: 5 * time.Minute,
		Loader: func(ctx context.Context) ([]byte, error) {
			return []byte("test_data"), nil
		},
		Enabled:    true,
		MaxRetries: 3,
	}
	
	scheduler.AddScheduledTask(task, 5*time.Minute)
	scheduler.PauseTask("test_task")
	
	tasks := scheduler.GetScheduledTasks()
	if len(tasks) > 0 && tasks[0].Enabled {
		t.Error("Task should be paused")
	}
	
	scheduler.ResumeTask("test_task")
	tasks = scheduler.GetScheduledTasks()
	if len(tasks) > 0 && !tasks[0].Enabled {
		t.Error("Task should be resumed")
	}
}

func TestInitAndGetWarmupScheduler(t *testing.T) {
	globalWarmupScheduler = nil
	globalSchedulerOnce = sync.Once{}
	
	InitWarmupScheduler(nil)
	scheduler := GetWarmupScheduler()
	
	if scheduler == nil {
		t.Fatal("GetWarmupScheduler should not return nil")
	}
	
	scheduler2 := GetWarmupScheduler()
	if scheduler != scheduler2 {
		t.Error("GetWarmupScheduler should return the same instance")
	}
}
