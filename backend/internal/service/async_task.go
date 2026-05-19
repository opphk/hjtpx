package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

var (
	ErrTaskNotFound      = errors.New("task not found")
	ErrTaskAlreadyExists = errors.New("task already exists")
	ErrTaskCancelled     = errors.New("task cancelled")
	ErrTaskTimeout       = errors.New("task timeout")
	ErrQueueFull         = errors.New("queue is full")
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

type TaskPriority int

const (
	PriorityLow    TaskPriority = 0
	PriorityNormal TaskPriority = 5
	PriorityHigh   TaskPriority = 8
	PriorityUrgent TaskPriority = 10
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
	CancelledAt *time.Time            `json:"cancelled_at,omitempty"`
	Timeout     time.Duration          `json:"timeout"`
	RetryDelay  time.Duration          `json:"retry_delay"`
	Metadata    map[string]string      `json:"metadata,omitempty"`
}

type TaskHandler func(ctx context.Context, task *Task) (interface{}, error)

type AsyncTaskService struct {
	handlers     map[TaskType]TaskHandler
	workerCount  int
	queue        chan *Task
	wg           sync.WaitGroup
	mu           sync.RWMutex
	ctx          context.Context
	cancel       context.CancelFunc
	running      bool
	taskStore    *TaskStore
	redisClient  *goredis.Client
	taskQueue    *TaskQueue
	dispatcher   *TaskDispatcher
	metrics      *TaskMetrics
	config       *AsyncTaskConfig
}

type AsyncTaskConfig struct {
	MaxQueueSize       int
	MaxRetries         int
	DefaultTimeout     time.Duration
	DefaultRetryDelay  time.Duration
	WorkerCount        int
	EnableRedisQueue   bool
	QueueKey           string
	TaskKeyPrefix      string
	MetricsEnabled     bool
	MonitoringInterval time.Duration
}

var DefaultAsyncTaskConfig = &AsyncTaskConfig{
	MaxQueueSize:       10000,
	MaxRetries:         3,
	DefaultTimeout:     5 * time.Minute,
	DefaultRetryDelay:  1 * time.Second,
	WorkerCount:        10,
	EnableRedisQueue:   true,
	QueueKey:          "async_task_queue",
	TaskKeyPrefix:     "task:",
	MetricsEnabled:     true,
	MonitoringInterval: 30 * time.Second,
}

type TaskMetrics struct {
	TotalEnqueued    atomic.Int64
	TotalProcessed   atomic.Int64
	TotalSucceeded   atomic.Int64
	TotalFailed      atomic.Int64
	TotalRetried     atomic.Int64
	TotalCancelled   atomic.Int64
	AvgWaitTime      atomic.Int64
	AvgExecTime      atomic.Int64
	MaxQueueDepth    atomic.Int64
	CurrentQueueDepth atomic.Int64
}

func NewTaskMetrics() *TaskMetrics {
	return &TaskMetrics{}
}

func (m *TaskMetrics) RecordEnqueue() {
	m.TotalEnqueued.Add(1)
	m.CurrentQueueDepth.Add(1)
	
	depth := m.CurrentQueueDepth.Load()
	for {
		maxDepth := m.MaxQueueDepth.Load()
		if depth <= maxDepth {
			break
		}
		if m.MaxQueueDepth.CompareAndSwap(maxDepth, depth) {
			break
		}
	}
}

func (m *TaskMetrics) RecordDequeue() {
	m.CurrentQueueDepth.Add(-1)
}

func (m *TaskMetrics) RecordSuccess(execTime time.Duration) {
	m.TotalProcessed.Add(1)
	m.TotalSucceeded.Add(1)
	m.updateAvgExecTime(execTime)
}

func (m *TaskMetrics) RecordFailure(execTime time.Duration) {
	m.TotalProcessed.Add(1)
	m.TotalFailed.Add(1)
	m.updateAvgExecTime(execTime)
}

func (m *TaskMetrics) RecordRetry() {
	m.TotalRetried.Add(1)
}

func (m *TaskMetrics) RecordCancellation() {
	m.TotalCancelled.Add(1)
}

func (m *TaskMetrics) RecordWait(waitTime time.Duration) {
	avg := m.AvgWaitTime.Load()
	newAvg := (avg + waitTime.Nanoseconds()) / 2
	m.AvgWaitTime.Store(newAvg)
}

func (m *TaskMetrics) updateAvgExecTime(execTime time.Duration) {
	avg := m.AvgExecTime.Load()
	newAvg := (avg + execTime.Nanoseconds()) / 2
	m.AvgExecTime.Store(newAvg)
}

type TaskDispatcher struct {
	mu         sync.RWMutex
	queues     map[TaskPriority]chan *Task
	priorities []TaskPriority
	maxSize    int
	running    bool
	workerWg   sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

func NewTaskDispatcher(maxSize int) *TaskDispatcher {
	priorities := []TaskPriority{PriorityUrgent, PriorityHigh, PriorityNormal, PriorityLow}
	queues := make(map[TaskPriority]chan *Task)
	
	for _, p := range priorities {
		queues[p] = make(chan *Task, maxSize/len(priorities))
	}
	
	return &TaskDispatcher{
		queues:    queues,
		priorities: priorities,
		maxSize:  maxSize,
	}
}

func (d *TaskDispatcher) Start() {
	d.mu.Lock()
	if d.running {
		d.mu.Unlock()
		return
	}
	d.running = true
	d.ctx, d.cancel = context.WithCancel(context.Background())
	d.mu.Unlock()
	
	d.workerWg.Add(len(d.queues))
	for i, p := range d.priorities {
		go d.priorityWorker(i, p)
	}
}

func (d *TaskDispatcher) Stop() {
	d.mu.Lock()
	if !d.running {
		d.mu.Unlock()
		return
	}
	d.running = false
	d.mu.Unlock()
	
	d.cancel()
	d.workerWg.Wait()
}

func (d *TaskDispatcher) Enqueue(task *Task) error {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	priority := TaskPriority(task.Priority)
	if priority < PriorityLow || priority > PriorityUrgent {
		priority = PriorityNormal
	}
	
	queue, ok := d.queues[priority]
	if !ok {
		queue = d.queues[PriorityNormal]
	}
	
	select {
	case queue <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (d *TaskDispatcher) Dequeue(timeout time.Duration) (*Task, error) {
	deadline := time.Now().Add(timeout)
	
	for time.Now().Before(deadline) {
		d.mu.RLock()
		for _, priority := range d.priorities {
			queue := d.queues[priority]
			select {
			case task := <-queue:
				d.mu.RUnlock()
				return task, nil
			default:
			}
		}
		d.mu.RUnlock()
		
		time.Sleep(10 * time.Millisecond)
	}
	
	return nil, ErrTaskTimeout
}

func (d *TaskDispatcher) priorityWorker(id int, priority TaskPriority) {
	defer d.workerWg.Done()
	
	for {
		select {
		case <-d.ctx.Done():
			return
		case task, ok := <-d.queues[priority]:
			if !ok {
				return
			}
			// Task will be processed by the main worker
			// This is a placeholder for priority queue processing logic
			_ = task
		}
	}
}

func (d *TaskDispatcher) GetQueueStats() map[TaskPriority]int {
	d.mu.RLock()
	defer d.mu.RUnlock()
	
	stats := make(map[TaskPriority]int)
	for p, queue := range d.queues {
		stats[p] = len(queue)
	}
	return stats
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

func (ts *TaskStore) ListByStatus(status TaskStatus) []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	tasks := make([]*Task, 0)
	for _, task := range ts.tasks {
		if task.Status == status {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (ts *TaskStore) ListByType(taskType TaskType) []*Task {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	tasks := make([]*Task, 0)
	for _, task := range ts.tasks {
		if task.Type == taskType {
			tasks = append(tasks, task)
		}
	}
	return tasks
}

func (ts *TaskStore) CountByStatus(status TaskStatus) int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	count := 0
	for _, task := range ts.tasks {
		if task.Status == status {
			count++
		}
	}
	return count
}

func (ts *TaskStore) CountByType(taskType TaskType) int {
	ts.mu.RLock()
	defer ts.mu.RUnlock()
	count := 0
	for _, task := range ts.tasks {
		if task.Type == taskType {
			count++
		}
	}
	return count
}

type TaskQueue struct {
	client     *goredis.Client
	queueKey    string
	taskPrefix string
}

func NewTaskQueue(redisClient *goredis.Client, config *AsyncTaskConfig) *TaskQueue {
	if config == nil {
		config = DefaultAsyncTaskConfig
	}
	return &TaskQueue{
		client:     redisClient,
		queueKey:   config.QueueKey,
		taskPrefix: config.TaskKeyPrefix,
	}
}

func (tq *TaskQueue) Enqueue(ctx context.Context, task *Task, priority int) error {
	if tq.client == nil {
		return errors.New("redis client is nil")
	}
	
	score := float64(priority)*1000000000 + float64(task.CreatedAt.UnixNano())
	
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	
	pipe := tq.client.Pipeline()
	pipe.ZAdd(ctx, tq.queueKey, goredis.Z{Score: score, Member: string(data)})
	pipe.Set(ctx, tq.taskPrefix+task.ID, string(data), 24*time.Hour)
	_, err = pipe.Exec(ctx)
	return err
}

func (tq *TaskQueue) Dequeue(ctx context.Context, timeout time.Duration) (*Task, error) {
	if tq.client == nil {
		return nil, errors.New("redis client is nil")
	}
	
	result, err := tq.client.ZPopMin(ctx, tq.queueKey, 1).Result()
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
	if tq.client == nil {
		return nil, errors.New("redis client is nil")
	}
	
	key := tq.taskPrefix + taskID
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
	if tq.client == nil {
		return errors.New("redis client is nil")
	}
	
	key := tq.taskPrefix + task.ID
	data, err := json.Marshal(task)
	if err != nil {
		return fmt.Errorf("failed to marshal task: %w", err)
	}
	
	return tq.client.Set(ctx, key, data, 24*time.Hour).Err()
}

func (tq *TaskQueue) DeleteTask(ctx context.Context, taskID string) error {
	if tq.client == nil {
		return errors.New("redis client is nil")
	}
	
	key := tq.taskPrefix + taskID
	return tq.client.Del(ctx, key).Err()
}

func (tq *TaskQueue) GetQueueLength(ctx context.Context) (int64, error) {
	if tq.client == nil {
		return 0, errors.New("redis client is nil")
	}
	
	return tq.client.ZCard(ctx, tq.queueKey).Result()
}

func (tq *TaskQueue) RequeueWithDelay(ctx context.Context, task *Task, delay time.Duration) error {
	if tq.client == nil {
		return errors.New("redis client is nil")
	}
	
	go func() {
		time.Sleep(delay)
		
		score := float64(task.Priority)*1000000000 + float64(time.Now().UnixNano())
		data, err := json.Marshal(task)
		if err != nil {
			log.Printf("Failed to marshal task for requeue: %v", err)
			return
		}
		
		tq.client.ZAdd(ctx, tq.queueKey, goredis.Z{Score: score, Member: string(data)})
	}()
	
	return nil
}

func NewAsyncTaskService(redisClient *goredis.Client, workerCount int, config *AsyncTaskConfig) *AsyncTaskService {
	if config == nil {
		config = DefaultAsyncTaskConfig
	}
	if workerCount <= 0 {
		workerCount = config.WorkerCount
	}
	
	ctx, cancel := context.WithCancel(context.Background())
	
	s := &AsyncTaskService{
		handlers:   make(map[TaskType]TaskHandler),
		workerCount: workerCount,
		queue:       make(chan *Task, config.MaxQueueSize),
		ctx:         ctx,
		cancel:      cancel,
		taskStore:   NewTaskStore(),
		redisClient: redisClient,
		metrics:     NewTaskMetrics(),
		config:      config,
	}
	
	if redisClient != nil && config.EnableRedisQueue {
		s.taskQueue = NewTaskQueue(redisClient, config)
	}
	
	s.dispatcher = NewTaskDispatcher(config.MaxQueueSize)
	
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
	
	s.dispatcher.Start()
	
	for i := 0; i < s.workerCount; i++ {
		s.wg.Add(1)
		go s.worker(i)
	}
	
	if s.config.MetricsEnabled {
		go s.monitorMetrics()
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
	s.dispatcher.Stop()
	
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
	startTime := time.Now()
	waitTime := startTime.Sub(task.CreatedAt)
	s.metrics.RecordWait(waitTime)
	
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
	
	if s.taskQueue != nil {
		if err := s.taskQueue.UpdateTask(ctx, task); err != nil {
			log.Printf("Failed to update task in Redis: %v", err)
		}
	}
	
	result, err := handler(ctx, task)
	
	completedAt := time.Now()
	task.CompletedAt = &completedAt
	execTime := completedAt.Sub(startTime)
	
	if err != nil {
		if task.RetryCount < task.MaxRetries {
			task.RetryCount++
			task.Status = TaskStatusPending
			s.metrics.RecordRetry()
			
			delay := task.RetryDelay
			if delay == 0 {
				delay = s.config.DefaultRetryDelay * time.Duration(task.RetryCount)
			}
			
			if s.taskQueue != nil {
				if err := s.taskQueue.RequeueWithDelay(ctx, task, delay); err != nil {
					log.Printf("Failed to requeue task with delay: %v", err)
					s.queue <- task
				}
			} else {
				go func() {
					time.Sleep(delay)
					s.queue <- task
				}()
			}
		} else {
			task.Status = TaskStatusFailed
			task.Error = err.Error()
			s.metrics.RecordFailure(execTime)
		}
	} else {
		task.Status = TaskStatusCompleted
		task.Result = result
		s.metrics.RecordSuccess(execTime)
	}
	
	s.taskStore.Set(task)
	
	if s.taskQueue != nil {
		if err := s.taskQueue.UpdateTask(ctx, task); err != nil {
			log.Printf("Failed to update task in Redis: %v", err)
		}
	}
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
		task.MaxRetries = s.config.MaxRetries
	}
	if task.Timeout == 0 {
		task.Timeout = s.config.DefaultTimeout
	}
	if task.RetryDelay == 0 {
		task.RetryDelay = s.config.DefaultRetryDelay
	}
	
	s.taskStore.Set(task)
	s.metrics.RecordEnqueue()
	
	if s.taskQueue != nil {
		return s.taskQueue.Enqueue(ctx, task, task.Priority)
	}
	
	select {
	case s.queue <- task:
		return nil
	default:
		return ErrQueueFull
	}
}

func (s *AsyncTaskService) EnqueueWithPriority(ctx context.Context, task *Task, priority TaskPriority) error {
	task.Priority = int(priority)
	return s.Enqueue(ctx, task)
}

func (s *AsyncTaskService) GetTask(taskID string) (*Task, bool) {
	return s.taskStore.Get(taskID)
}

func (s *AsyncTaskService) GetTaskFromRedis(ctx context.Context, taskID string) (*Task, error) {
	if s.taskQueue == nil {
		return nil, errors.New("redis queue not available")
	}
	return s.taskQueue.GetTask(ctx, taskID)
}

func (s *AsyncTaskService) ListTasks() []*Task {
	return s.taskStore.List()
}

func (s *AsyncTaskService) ListTasksByStatus(status TaskStatus) []*Task {
	return s.taskStore.ListByStatus(status)
}

func (s *AsyncTaskService) ListTasksByType(taskType TaskType) []*Task {
	return s.taskStore.ListByType(taskType)
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
	s.metrics.RecordCancellation()
	
	return nil
}

func (s *AsyncTaskService) RetryTask(taskID string) error {
	task, ok := s.taskStore.Get(taskID)
	if !ok {
		return ErrTaskNotFound
	}
	
	if task.Status != TaskStatusFailed {
		return fmt.Errorf("can only retry failed tasks, current status: %s", task.Status)
	}
	
	task.Status = TaskStatusPending
	task.RetryCount = 0
	task.Error = ""
	s.taskStore.Set(task)
	
	select {
	case s.queue <- task:
		return nil
	default:
		return ErrQueueFull
	}
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
	
	stats.Pending = int64(s.taskStore.CountByStatus(TaskStatusPending))
	stats.Running = int64(s.taskStore.CountByStatus(TaskStatusRunning))
	stats.Completed = int64(s.taskStore.CountByStatus(TaskStatusCompleted))
	stats.Failed = int64(s.taskStore.CountByStatus(TaskStatusFailed))
	stats.Cancelled = int64(s.taskStore.CountByStatus(TaskStatusCancelled))
	stats.QueueLength = int64(len(s.queue))
	
	if s.config != nil && s.config.EnableRedisQueue && s.taskQueue != nil {
		if length, err := s.taskQueue.GetQueueLength(context.Background()); err == nil {
			stats.QueueLength = length
		}
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

func (s *AsyncTaskService) GetMetrics() *TaskMetrics {
	return s.metrics
}

func (s *AsyncTaskService) monitorMetrics() {
	ticker := time.NewTicker(s.config.MonitoringInterval)
	defer ticker.Stop()
	
	for {
		select {
		case <-s.ctx.Done():
			return
		case <-ticker.C:
			s.updateRedisMetrics()
		}
	}
}

func (s *AsyncTaskService) updateRedisMetrics() {
	if s.taskQueue == nil || s.redisClient == nil {
		return
	}
	
	ctx := context.Background()
	length, err := s.taskQueue.GetQueueLength(ctx)
	if err != nil {
		log.Printf("Failed to get queue length: %v", err)
		return
	}
	
	s.metrics.CurrentQueueDepth.Store(length)
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
		if task.Payload == nil {
			task.Payload = make(map[string]interface{})
		}
		task.Payload["batch_group"] = batch.GroupID
		
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

func (s *AsyncTaskService) GetBatchProgress(groupID string) (completed, failed, total int) {
	statuses := s.GetBatchStatus(groupID)
	total = len(statuses)
	
	for _, status := range statuses {
		switch status {
		case TaskStatusCompleted:
			completed++
		case TaskStatusFailed:
			failed++
		}
	}
	
	return
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
				if s.taskQueue != nil {
					s.taskQueue.DeleteTask(context.Background(), task.ID)
				}
				cleaned++
			}
		}
	}
	
	return cleaned
}

func (s *AsyncTaskService) GetQueueStats() map[TaskPriority]int {
	if s.dispatcher == nil {
		return nil
	}
	return s.dispatcher.GetQueueStats()
}

type GlobalAsyncTaskService struct {
	instance *AsyncTaskService
	once     sync.Once
}

var globalAsyncTaskService *GlobalAsyncTaskService

func InitGlobalAsyncTaskService(redisClient *goredis.Client, workerCount int, config *AsyncTaskConfig) {
	globalAsyncTaskService = &GlobalAsyncTaskService{}
	globalAsyncTaskService.once.Do(func() {
		globalAsyncTaskService.instance = NewAsyncTaskService(redisClient, workerCount, config)
		globalAsyncTaskService.instance.RegisterDefaultHandlers()
	})
}

func GetGlobalAsyncTaskService() *AsyncTaskService {
	if globalAsyncTaskService == nil {
		return nil
	}
	return globalAsyncTaskService.instance
}
