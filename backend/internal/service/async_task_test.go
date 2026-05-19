package service

import (
	"context"
	"testing"
	"time"
)

func TestNewAsyncTaskService(t *testing.T) {
	service := NewAsyncTaskService(nil, 5, nil)
	if service == nil {
		t.Error("NewAsyncTaskService returned nil")
	}
	if service.workerCount != 5 {
		t.Errorf("Expected worker count 5, got %d", service.workerCount)
	}
}

func TestNewAsyncTaskServiceWithConfig(t *testing.T) {
	config := &AsyncTaskConfig{
		WorkerCount:      10,
		MaxQueueSize:     5000,
		MaxRetries:       5,
		DefaultTimeout:   10 * time.Minute,
		EnableRedisQueue: false,
	}
	
	service := NewAsyncTaskService(nil, 0, config)
	if service == nil {
		t.Error("NewAsyncTaskService returned nil")
	}
	if service.workerCount != 10 {
		t.Errorf("Expected worker count 10, got %d", service.workerCount)
	}
	if service.config.MaxRetries != 5 {
		t.Errorf("Expected max retries 5, got %d", service.config.MaxRetries)
	}
}

func TestAsyncTaskService_RegisterHandler(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	handler := func(ctx context.Context, task *Task) (interface{}, error) {
		return "processed", nil
	}
	
	service.RegisterHandler(TaskTypeImage, handler)
	
	service.mu.RLock()
	_, exists := service.handlers[TaskTypeImage]
	service.mu.RUnlock()
	
	if !exists {
		t.Error("Handler was not registered")
	}
}

func TestAsyncTaskService_RegisterDefaultHandlers(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	service.RegisterDefaultHandlers()
	
	service.mu.RLock()
	defer service.mu.RUnlock()
	
	handlers := []TaskType{TaskTypeImage, TaskTypeEmail, TaskTypeExport, TaskTypeCleanup}
	for _, h := range handlers {
		if _, exists := service.handlers[h]; !exists {
			t.Errorf("Default handler for %s was not registered", h)
		}
	}
}

func TestAsyncTaskService_StartStop(t *testing.T) {
	service := NewAsyncTaskService(nil, 2, nil)
	
	service.Start()
	if !service.running {
		t.Error("Service should be running after Start")
	}
	
	service.Stop()
	if service.running {
		t.Error("Service should not be running after Stop")
	}
}

func TestAsyncTaskService_StartMultipleTimes(t *testing.T) {
	service := NewAsyncTaskService(nil, 2, nil)
	
	service.Start()
	service.Start()
	service.Stop()
}

func TestAsyncTaskService_Enqueue(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	service.RegisterDefaultHandlers()
	
	task := &Task{
		Type:    TaskTypeImage,
		Payload: map[string]interface{}{"test": "data"},
	}
	
	err := service.Enqueue(context.Background(), task)
	if err != nil {
		t.Errorf("Failed to enqueue task: %v", err)
	}
	
	if task.ID == "" {
		t.Error("Task ID should be generated")
	}
	
	if task.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %s", task.Status)
	}
}

func TestAsyncTaskService_EnqueueWithPriority(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{
		Type: TaskTypeExport,
	}
	
	err := service.EnqueueWithPriority(context.Background(), task, PriorityHigh)
	if err != nil {
		t.Errorf("Failed to enqueue task with priority: %v", err)
	}
	
	if task.Priority != int(PriorityHigh) {
		t.Errorf("Expected priority %d, got %d", PriorityHigh, task.Priority)
	}
}

func TestAsyncTaskService_GetTask(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{
		Type: TaskTypeEmail,
	}
	
	err := service.Enqueue(context.Background(), task)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}
	
	retrieved, found := service.GetTask(task.ID)
	if !found {
		t.Error("Task was not found")
	}
	
	if retrieved.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrieved.ID)
	}
}

func TestAsyncTaskService_CancelTask(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{
		Type: TaskTypeCleanup,
	}
	
	err := service.Enqueue(context.Background(), task)
	if err != nil {
		t.Fatalf("Failed to enqueue task: %v", err)
	}
	
	err = service.CancelTask(task.ID)
	if err != nil {
		t.Errorf("Failed to cancel task: %v", err)
	}
	
	cancelled, _ := service.GetTask(task.ID)
	if cancelled.Status != TaskStatusCancelled {
		t.Errorf("Expected status cancelled, got %s", cancelled.Status)
	}
}

func TestAsyncTaskService_CancelCompletedTask(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{
		Type:   TaskTypeExport,
		Status: TaskStatusCompleted,
	}
	
	err := service.CancelTask(task.ID)
	if err == nil {
		t.Error("Should not be able to cancel completed task")
	}
}

func TestAsyncTaskService_RetryTask(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{
		ID:     "retry-test-task",
		Type:   TaskTypeImage,
		Status: TaskStatusFailed,
		Error:  "test error",
	}
	
	service.taskStore.Set(task)
	
	err := service.RetryTask(task.ID)
	if err != nil {
		t.Errorf("Failed to retry task: %v", err)
	}
	
	retried, _ := service.GetTask(task.ID)
	if retried.Status != TaskStatusPending {
		t.Errorf("Expected status pending, got %s", retried.Status)
	}
	
	if retried.RetryCount != 0 {
		t.Errorf("Expected retry count 0, got %d", retried.RetryCount)
	}
}

func TestAsyncTaskService_GetStats(t *testing.T) {
	service := NewAsyncTaskService(nil, 2, nil)
	
	for i := 0; i < 5; i++ {
		task := &Task{Type: TaskTypeExport}
		service.Enqueue(context.Background(), task)
	}
	
	stats := service.GetStats()
	if stats.Total != 5 {
		t.Errorf("Expected total 5, got %d", stats.Total)
	}
}

func TestAsyncTaskService_ListTasks(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	for i := 0; i < 3; i++ {
		task := &Task{Type: TaskTypeEmail}
		service.Enqueue(context.Background(), task)
	}
	
	tasks := service.ListTasks()
	if len(tasks) != 3 {
		t.Errorf("Expected 3 tasks, got %d", len(tasks))
	}
}

func TestAsyncTaskService_ListTasksByStatus(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{Type: TaskTypeImage}
	service.Enqueue(context.Background(), task)
	
	pendingTasks := service.ListTasksByStatus(TaskStatusPending)
	if len(pendingTasks) != 1 {
		t.Errorf("Expected 1 pending task, got %d", len(pendingTasks))
	}
}

func TestAsyncTaskService_ListTasksByType(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{Type: TaskTypeExport}
	service.Enqueue(context.Background(), task)
	
	exportTasks := service.ListTasksByType(TaskTypeExport)
	if len(exportTasks) != 1 {
		t.Errorf("Expected 1 export task, got %d", len(exportTasks))
	}
}

func TestAsyncTaskService_EnqueueBatch(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	tasks := []*Task{
		{Type: TaskTypeImage},
		{Type: TaskTypeEmail},
		{Type: TaskTypeExport},
	}
	
	batch, err := service.EnqueueBatch(context.Background(), tasks)
	if err != nil {
		t.Errorf("Failed to enqueue batch: %v", err)
	}
	
	if len(batch.Tasks) != 3 {
		t.Errorf("Expected 3 tasks in batch, got %d", len(batch.Tasks))
	}
	
	if batch.GroupID == "" {
		t.Error("Batch group ID should not be empty")
	}
}

func TestAsyncTaskService_GetBatchStatus(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	tasks := []*Task{
		{Type: TaskTypeImage},
		{Type: TaskTypeEmail},
	}
	
	batch, _ := service.EnqueueBatch(context.Background(), tasks)
	
	statuses := service.GetBatchStatus(batch.GroupID)
	if len(statuses) != 2 {
		t.Errorf("Expected 2 task statuses, got %d", len(statuses))
	}
}

func TestAsyncTaskService_GetBatchProgress(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	tasks := []*Task{
		{Type: TaskTypeImage},
		{Type: TaskTypeEmail},
	}
	
	batch, _ := service.EnqueueBatch(context.Background(), tasks)
	
	completed, failed, total := service.GetBatchProgress(batch.GroupID)
	if total != 2 {
		t.Errorf("Expected total 2, got %d", total)
	}
	if completed != 0 {
		t.Errorf("Expected completed 0, got %d", completed)
	}
	if failed != 0 {
		t.Errorf("Expected failed 0, got %d", failed)
	}
}

func TestAsyncTaskService_SchedulePeriodicTask(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	payload := map[string]interface{}{"test": "data"}
	pt := service.SchedulePeriodicTask("test-task", TaskTypeCleanup, 1*time.Second, payload, nil)
	
	if pt == nil {
		t.Error("Periodic task should not be nil")
	}
	
	pt.Stop()
}

func TestAsyncTaskService_CleanupCompletedTasks(t *testing.T) {
	service := NewAsyncTaskService(nil, 1, nil)
	
	task := &Task{
		Type:        TaskTypeExport,
		Status:      TaskStatusCompleted,
		CompletedAt: func() *time.Time { t := time.Now().Add(-2 * time.Hour); return &t }(),
	}
	
	task.ID = "test-cleanup-task"
	service.taskStore.Set(task)
	
	cleaned := service.CleanupCompletedTasks(1 * time.Hour)
	if cleaned != 1 {
		t.Errorf("Expected 1 cleaned task, got %d", cleaned)
	}
}

func TestTaskStore(t *testing.T) {
	store := NewTaskStore()
	
	task := &Task{
		ID:   "test-1",
		Type: TaskTypeImage,
	}
	
	store.Set(task)
	
	retrieved, found := store.Get("test-1")
	if !found {
		t.Error("Task not found")
	}
	if retrieved.ID != task.ID {
		t.Errorf("Expected task ID %s, got %s", task.ID, retrieved.ID)
	}
	
	store.Delete("test-1")
	_, found = store.Get("test-1")
	if found {
		t.Error("Task should be deleted")
	}
}

func TestTaskStore_ListByStatus(t *testing.T) {
	store := NewTaskStore()
	
	store.Set(&Task{ID: "1", Status: TaskStatusPending})
	store.Set(&Task{ID: "2", Status: TaskStatusPending})
	store.Set(&Task{ID: "3", Status: TaskStatusCompleted})
	
	tasks := store.ListByStatus(TaskStatusPending)
	if len(tasks) != 2 {
		t.Errorf("Expected 2 pending tasks, got %d", len(tasks))
	}
}

func TestTaskStore_ListByType(t *testing.T) {
	store := NewTaskStore()
	
	store.Set(&Task{ID: "1", Type: TaskTypeImage})
	store.Set(&Task{ID: "2", Type: TaskTypeEmail})
	store.Set(&Task{ID: "3", Type: TaskTypeImage})
	
	tasks := store.ListByType(TaskTypeImage)
	if len(tasks) != 2 {
		t.Errorf("Expected 2 image tasks, got %d", len(tasks))
	}
}

func TestTaskMetrics(t *testing.T) {
	metrics := NewTaskMetrics()
	
	metrics.RecordEnqueue()
	metrics.RecordEnqueue()
	
	if metrics.TotalEnqueued.Load() != 2 {
		t.Errorf("Expected 2 enqueued, got %d", metrics.TotalEnqueued.Load())
	}
	
	metrics.RecordSuccess(100 * time.Millisecond)
	metrics.RecordSuccess(200 * time.Millisecond)
	metrics.RecordFailure(150 * time.Millisecond)
	
	if metrics.TotalProcessed.Load() != 3 {
		t.Errorf("Expected 3 processed, got %d", metrics.TotalProcessed.Load())
	}
	
	if metrics.TotalSucceeded.Load() != 2 {
		t.Errorf("Expected 2 succeeded, got %d", metrics.TotalSucceeded.Load())
	}
	
	if metrics.TotalFailed.Load() != 1 {
		t.Errorf("Expected 1 failed, got %d", metrics.TotalFailed.Load())
	}
}

func TestTaskDispatcher(t *testing.T) {
	dispatcher := NewTaskDispatcher(100)
	
	dispatcher.Start()
	
	task := &Task{Priority: int(PriorityHigh)}
	err := dispatcher.Enqueue(task)
	if err != nil {
		t.Errorf("Failed to enqueue task: %v", err)
	}
	
	stats := dispatcher.GetQueueStats()
	if stats == nil {
		t.Error("Queue stats should not be nil")
	}
	
	dispatcher.Stop()
}

func TestTaskDispatcher_EnqueueFull(t *testing.T) {
	dispatcher := NewTaskDispatcher(1)
	dispatcher.Start()
	defer dispatcher.Stop()
	
	task1 := &Task{Priority: 0}
	task2 := &Task{Priority: 0}
	
	if err := dispatcher.Enqueue(task1); err != nil {
		t.Errorf("First enqueue should succeed: %v", err)
	}
	
	if err := dispatcher.Enqueue(task2); err == nil {
		t.Error("Second enqueue should fail with full queue")
	}
}

func TestTaskPriorityConstants(t *testing.T) {
	if PriorityLow != 0 {
		t.Errorf("Expected PriorityLow to be 0, got %d", PriorityLow)
	}
	if PriorityNormal != 5 {
		t.Errorf("Expected PriorityNormal to be 5, got %d", PriorityNormal)
	}
	if PriorityHigh != 8 {
		t.Errorf("Expected PriorityHigh to be 8, got %d", PriorityHigh)
	}
	if PriorityUrgent != 10 {
		t.Errorf("Expected PriorityUrgent to be 10, got %d", PriorityUrgent)
	}
}

func TestTaskStatusConstants(t *testing.T) {
	statuses := []TaskStatus{
		TaskStatusPending,
		TaskStatusRunning,
		TaskStatusCompleted,
		TaskStatusFailed,
		TaskStatusCancelled,
	}
	
	expected := []string{"pending", "running", "completed", "failed", "cancelled"}
	
	for i, status := range statuses {
		if string(status) != expected[i] {
			t.Errorf("Expected status %s, got %s", expected[i], status)
		}
	}
}

func TestTaskTypeConstants(t *testing.T) {
	types := []TaskType{
		TaskTypeDefault,
		TaskTypeImage,
		TaskTypeEmail,
		TaskTypeExport,
		TaskTypeCleanup,
	}
	
	expected := []string{"default", "image", "email", "export", "cleanup"}
	
	for i, typ := range types {
		if string(typ) != expected[i] {
			t.Errorf("Expected type %s, got %s", expected[i], typ)
		}
	}
}

func TestAsyncTaskConfig(t *testing.T) {
	config := DefaultAsyncTaskConfig
	
	if config.MaxQueueSize != 10000 {
		t.Errorf("Expected MaxQueueSize 10000, got %d", config.MaxQueueSize)
	}
	
	if config.MaxRetries != 3 {
		t.Errorf("Expected MaxRetries 3, got %d", config.MaxRetries)
	}
	
	if config.DefaultTimeout != 5*time.Minute {
		t.Errorf("Expected DefaultTimeout 5 minutes, got %v", config.DefaultTimeout)
	}
}

func TestGenerateID(t *testing.T) {
	id1 := generateID()
	id2 := generateID()
	
	if id1 == "" || id2 == "" {
		t.Error("Generated ID should not be empty")
	}
	
	if id1 == id2 {
		t.Error("Generated IDs should be unique")
	}
	
	if len(id1) != 8 {
		t.Errorf("Expected ID length 8, got %d", len(id1))
	}
}

func TestTaskBatch(t *testing.T) {
	batch := &TaskBatch{
		GroupID:   "test-batch",
		CreatedAt: time.Now(),
	}
	
	if batch.GroupID != "test-batch" {
		t.Errorf("Expected group ID 'test-batch', got '%s'", batch.GroupID)
	}
}

func TestPeriodicTask(t *testing.T) {
	pt := &PeriodicTask{
		ID:       "test-periodic",
		Type:     TaskTypeCleanup,
		Interval: 1 * time.Hour,
		stopCh:   make(chan struct{}),
	}
	
	if pt.ID != "test-periodic" {
		t.Errorf("Expected ID 'test-periodic', got '%s'", pt.ID)
	}
	
	select {
	case <-pt.stopCh:
		t.Error("stopCh should not be closed initially")
	default:
	}
	
	close(pt.stopCh)
	
	select {
	case <-pt.stopCh:
	default:
		t.Error("stopCh should be closed")
	}
}

func TestGlobalAsyncTaskService(t *testing.T) {
	InitGlobalAsyncTaskService(nil, 3, nil)
	
	service := GetGlobalAsyncTaskService()
	if service == nil {
		t.Error("GetGlobalAsyncTaskService returned nil")
	}
	
	if service.workerCount != 3 {
		t.Errorf("Expected worker count 3, got %d", service.workerCount)
	}
}