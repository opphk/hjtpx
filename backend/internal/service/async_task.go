package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrTaskCancelled     = errors.New("task cancelled")
	ErrTaskTimeout       = errors.New("task timeout")
)

type TaskStatus string

const (
	TaskStatusPending   TaskStatus = "pending"
	TaskStatusRunning   TaskStatus = "running"
	TaskStatusCompleted TaskStatus = "completed"
	TaskStatusFailed    TaskStatus = "failed"
	TaskStatusCancelled TaskStatus = "cancelled"
)

type TaskType string

const (
	TaskTypeDefault TaskType = "default"
	TaskTypeImage   TaskType = "image"
	TaskTypeEmail   TaskType = "email"
	TaskTypeExport  TaskType = "export"
	TaskTypeCleanup TaskType = "cleanup"
)

type Task struct {
	ID          string                 `json:"id"`
	Type        TaskType               `json:"type"`
	Payload     map[string]interface{} `json:"payload"`
	Status      TaskStatus             `json:"status"`
	Result      interface{}            `json:"result,omitempty"`
	Error       string                 `json:"error,omitempty"`
	Priority    int                    `json:"priority"`
	RetryCount  int                    `json:"retry_count"`
	MaxRetries  int                    `json:"max_retries"`
	CreatedAt   time.Time              `json:"created_at"`
	StartedAt   *time.Time             `json:"started_at,omitempty"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	CancelledAt *time.Time             `json:"cancelled_at,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
}

type TaskHandler func(ctx context.Context, task *Task) (interface{}, error)

type AsyncTaskService struct {
	handlers    map[TaskType]TaskHandler
	workerCount int
	queue       chan *Task
	wg          sync.WaitGroup
	mu          sync.RWMutex
	ctx         context.Context
	cancel      context.CancelFunc
	running     bool
	taskStore   *TaskStore
	redisClient *goredis.Client
}

type TaskStore struct {
	mu    sync.RWMutex
	tasks map[string]*Task
}

func NewTaskStore() *TaskStore {
	return &TaskStore{
		tasks: make(map[string]*Task),
	}
}

func (ts *TaskStore) Set(task *Task) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	ts.tasks[task.ID] = task
}

func (ts *TaskStore) Get(id string) (*Task, bool) {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	task, ok := ts.tasks[id]
	return task, ok
}

func (ts *TaskStore) Delete(id string) {
	ts.mu.Lock()
	defer ts.mu.Unlock()
	delete(ts.tasks, id)
}

func (ts *TaskStore) List() []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	tasks := make([]*Task, 0, len(ts.tasks))
	for _, task := range ts.tasks {
		tasks = append(tasks, task)
	}
	return tasks
}

type TaskQueue struct {
	client *goredis.Client
}

func NewTaskQueue(redisClient *goredis.Client) *TaskQueue {
	return &TaskQueue{client: redisClient}
}

func (tq *TaskQueue) Enqueue(ctx context.Context, task *Task, priority int) error {
	key := "async_task_queue"
	score := float64(priority)*1000000000 + float64(task.CreatedAt.UnixNano())

	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	pipe := tq.client.Pipeline()
	pipe.ZAdd(ctx, key, goredis.Z{Score: score, Member: string(data)})
	pipe.Set(ctx, fmt.Sprintf("task:%s", task.ID), string(data), 24*time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

func (tq *TaskQueue) Dequeue(ctx context.Context, timeout time.Duration) (*Task, error) {
	key := "async_task_queue"

	result, err := tq.client.ZPopMin(ctx, key, 1).Result()
	if err != nil {
		return nil, fmt.Errorf("failed to dequeue task: %w", err)
	}

	if len(result) == 0 {
		return nil, ErrTaskNotFound
	}

	var task Task
	data, ok := result[0].Member.(string)
	if !ok {
		return nil, fmt.Errorf("invalid task data type")
	}

	if err := json.Unmarshal([]byte(data), &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

func (tq *TaskQueue) GetTask(ctx context.Context, taskID string) (*Task, error) {
	key := fmt.Sprintf("task:%s", taskID)
	data, err := tq.client.Get(ctx, key).Bytes()
	if err != nil {
		return nil, ErrTaskNotFound
	}

	var task Task
	if err := json.Unmarshal(data, &task); err != nil {
		return nil, fmt.Errorf("failed to unmarshal task: %w", err)
	}

	return &task, nil
}

func (tq *TaskQueue) UpdateTask(ctx context.Context, task *Task) error {
	key := fmt.Sprintf("task:%s", task.ID)
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}

	return tq.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (tq *TaskQueue) DeleteTask(ctx context.Context, taskID string) error {
	key := fmt.Sprintf("task:%s", taskID)
	return tq.client.Del(ctx, key).Err()
}

func (tq *TaskQueue) GetQueueLength(ctx context.Context) (int64, error) {
	return tq.client.ZCard(ctx, "async_task_queue").Result()
}

func NewAsyncTaskService(redisClient *goredis.Client, workerCount int) *AsyncTaskService {
	ctx, cancel := context.WithCancel(context.Background())

	s := &AsyncTaskService{
		handlers:    make(map[TaskType]TaskHandler),
		workerCount: workerCount,
		queue:       make(chan *Task, 1000),
		ctx:         ctx,
		cancel:      cancel,
		taskStore:   NewTaskStore(),
		redisClient: redisClient,
	}

	if redisClient != nil {
		s.queue = make(chan *Task, 1000)
	}

	return s
}

func (s *AsyncTaskService) RegisterHandler(taskType TaskType, handler TaskHandler) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.handlers[taskType] = handler
}

func (s *AsyncTaskService) RegisterDefaultHandlers() {
	s.RegisterHandler(TaskTypeImage, s.defaultImageHandler)
	s.RegisterHandler(TaskTypeEmail, s.defaultEmailHandler)
	s.RegisterHandler(TaskTypeExport, s.defaultExportHandler)
	s.RegisterHandler(TaskTypeCleanup, s.defaultCleanupHandler)
}

func (s *AsyncTaskService) defaultImageHandler(ctx context.Context, task *Task) (interface{}, error) {
	time.Sleep(100 * time.Millisecond)
	return map[string]interface{}{
		"processed": true,
		"task_id":   task.ID,
	}, nil
}

func (s *AsyncTaskService) defaultEmailHandler(ctx context.Context, task *Task) (interface{}, error) {
	time.Sleep(50 * time.Millisecond)
	return map[string]interface{}{
		"sent":    true,
		"task_id": task.ID,
	}, nil
}

func (s *AsyncTaskService) defaultExportHandler(ctx context.Context, task *Task) (interface{}, error) {
	time.Sleep(200 * time.Millisecond)
	return map[string]interface{}{
		"exported": true,
		"task_id":  task.ID,
	}, nil
}

func (s *AsyncTaskService) defaultCleanupHandler(ctx context.Context, task *Task) (interface{}, error) {
	time.Sleep(50 * time.Millisecond)
	return map[string]interface{}{
		"cleaned": true,
		"task_id": task.ID,
	}, nil
}

func (s *AsyncTaskService) Start() {
	s.mu.Lock()
	if s.running {
		s.mu.Unlock()
		return
	}
	s.running = true
	s.mu.Unlock()

	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}

	log.Printf("Async task service started with %d workers", s.workerCount)
}

func (s *AsyncTaskService) Stop() {
	s.mu.Lock()
	if !s.running {
		s.mu.Unlock()
		return
	}
	s.running = false
	s.mu.Unlock()

	s.cancel()

	close(s.queue)

	s.wg.Wait()
	log.Println("Async task service stopped")
}

func (s *AsyncTaskService) worker(id int) {
	defer s.wg.Done()

	for {
		select {
		case <-s.ctx.Done():
			return
		case task, ok := <-s.queue:
			if !ok {
				return
			}
			s.processTask(s.ctx, task)
		}
	}
}

func (s *AsyncTaskService) processTask(ctx context.Context, task *Task) {
	s.mu.RLock()
	handler, ok := s.handlers[task.Type]
	s.mu.RUnlock()

	if !ok {
		handler = s.defaultHandler
	}

	now := time.Now()
	task.Status = TaskStatusRunning
	task.StartedAt = &now
	s.taskStore.Set(task)

	result, err := handler(ctx, task)

	completedAt := time.Now()
	task.CompletedAt = &completedAt

	if err != nil {
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Status = TaskStatusPending
			s.queue <- task
		} else {
			task.Status = TaskStatusFailed
			task.Error = err.Error()
		}
	} else {
		task.Status = TaskStatusCompleted
		task.Result = result
	}

	s.taskStore.Set(task)
}

func (s *AsyncTaskService) defaultHandler(ctx context.Context, task *Task) (interface{}, error) {
	return map[string]interface{}{
		"processed": true,
		"task_id":   task.ID,
	}, nil
}

func (s *AsyncTaskService) Enqueue(ctx context.Context, task *Task) error {
	task.ID = fmt.Sprintf("task_%d_%s", time.Now().UnixNano(), generateID())
	task.Status = TaskStatusPending
	task.CreatedAt = time.Now()

	if task.MaxRetries == 0 {
		task.MaxRetries = 3
	}
	if task.Timeout == 0 {
		task.Timeout = 5 * time.Minute
	}

	s.taskStore.Set(task)

	if s.redisClient != nil {
		tq := NewTaskQueue(s.redisClient)
		return tq.Enqueue(ctx, task, task.Priority)
	}

	select {
	case s.queue <- task:
		return nil
	default:
		return errors.New("queue is full")
	}
}

func (s *AsyncTaskService) GetTask(taskID string) (*Task, bool) {
	return s.taskStore.Get(taskID)
}

func (s *AsyncTaskService) ListTasks() []*Task {
	return s.taskStore.List()
}

func (s *AsyncTaskService) CancelTask(taskID string) error {
	task, ok := s.taskStore.Get(taskID)
	if !ok {
		return ErrTaskNotFound
	}

	if task.Status == TaskStatusCompleted || task.Status == TaskStatusFailed {
		return fmt.Errorf("cannot cancel task in status: %s", task.Status)
	}

	now := time.Now()
	task.Status = TaskStatusCancelled
	task.CancelledAt = &now
	s.taskStore.Set(task)

	return nil
}

func (s *AsyncTaskService) GetStats() *TaskStats {
	tasks := s.taskStore.List()

	stats := &TaskStats{
		Total:    int64(len(tasks)),
		ByType:   make(map[TaskType]int64),
		ByStatus: make(map[TaskStatus]int64),
	}

	for _, task := range tasks {
		stats.ByType[task.Type]++
		stats.ByStatus[task.Status]++
	}

	return stats
}

type TaskStats struct {
	Total       int64                `json:"total"`
	Pending     int64                `json:"pending"`
	Running     int64                `json:"running"`
	Completed   int64                `json:"completed"`
	Failed      int64                `json:"failed"`
	Cancelled   int64                `json:"cancelled"`
	ByType      map[TaskType]int64   `json:"by_type"`
	ByStatus    map[TaskStatus]int64 `json:"by_status"`
	QueueLength int64                `json:"queue_length"`
}

func (s *AsyncTaskService) WaitForTask(ctx context.Context, taskID string, timeout time.Duration) (*Task, error) {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	deadline := time.Now().Add(timeout)

	for {
		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-ticker.C:
			if time.Now().After(deadline) {
				return nil, ErrTaskTimeout
			}

			task, ok := s.GetTask(taskID)
			if !ok {
				continue
			}

			switch task.Status {
			case TaskStatusCompleted:
				return task, nil
			case TaskStatusFailed:
				return task, fmt.Errorf("task failed: %s", task.Error)
			case TaskStatusCancelled:
				return task, ErrTaskCancelled
			}
		}
	}
}

func generateID() string {
	const charset = "abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ0123456789"
	id := make([]byte, 8)
	for i := range id {
		id[i] = charset[time.Now().UnixNano()%int64(len(charset))]
		time.Sleep(time.Nanosecond)
	}
	return string(id)
}

type TaskBatch struct {
	Tasks     []*Task   `json:"tasks"`
	GroupID   string    `json:"group_id"`
	CreatedAt time.Time `json:"created_at"`
}

func (s *AsyncTaskService) EnqueueBatch(ctx context.Context, tasks []*Task) (*TaskBatch, error) {
	batch := &TaskBatch{
		Tasks:     make([]*Task, 0, len(tasks)),
		GroupID:   fmt.Sprintf("batch_%d", time.Now().UnixNano()),
		CreatedAt: time.Now(),
	}

	for _, task := range tasks {
		if err := s.Enqueue(ctx, task); err != nil {
			return nil, fmt.Errorf("failed to enqueue task: %w", err)
		}
		batch.Tasks = append(batch.Tasks, task)
	}

	return batch, nil
}

func (s *AsyncTaskService) GetBatchStatus(groupID string) map[string]TaskStatus {
	results := make(map[string]TaskStatus)
	tasks := s.taskStore.List()

	for _, task := range tasks {
		if group, ok := task.Payload["batch_group"]; ok && group == groupID {
			results[task.ID] = task.Status
		}
	}

	return results
}

type PeriodicTask struct {
	ID       string
	Type     TaskType
	Payload  map[string]interface{}
	Interval time.Duration
	handler  TaskHandler
	stopCh   chan struct{}
}

func (s *AsyncTaskService) SchedulePeriodicTask(id string, taskType TaskType, interval time.Duration, payload map[string]interface{}, handler TaskHandler) *PeriodicTask {
	pt := &PeriodicTask{
		ID:       id,
		Type:     taskType,
		Payload:  payload,
		Interval: interval,
		handler:  handler,
		stopCh:   make(chan struct{}),
	}

	go func() {
		ticker := time.NewTicker(interval)
		defer ticker.Stop()

		for {
			select {
			case <-pt.stopCh:
				return
			case <-ticker.C:
				task := &Task{
					Type:     taskType,
					Payload:  payload,
					Priority: 0,
				}
				s.Enqueue(context.Background(), task)
			}
		}
	}()

	return pt
}

func (pt *PeriodicTask) Stop() {
	close(pt.stopCh)
}

func (s *AsyncTaskService) CleanupCompletedTasks(olderThan time.Duration) int {
	tasks := s.taskStore.List()
	cutoff := time.Now().Add(-olderThan)
	cleaned := 0

	for _, task := range tasks {
		if task.Status == TaskStatusCompleted && task.CompletedAt != nil {
			if task.CompletedAt.Before(cutoff) {
				s.taskStore.Delete(task.ID)
				cleaned++
			}
		}
	}

	return cleaned
}
