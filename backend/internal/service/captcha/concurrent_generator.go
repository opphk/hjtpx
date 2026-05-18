package captcha

import (
	"context"
	"fmt"
	"runtime"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type ConcurrentGenerator struct {
	imageGenerator *ImageGenerator
	workerPool     *WorkerPool
	priorityQueue  *PriorityQueue
	stats          *GeneratorStats
	mu             sync.RWMutex
}

type WorkerPool struct {
	workers    int
	taskQueue  chan *Task
	resultChan chan *TaskResult
	wg         sync.WaitGroup
	ctx        context.Context
	cancel     context.CancelFunc
}

type Task struct {
	ID       string
	Type     string
	Priority int
	Metadata map[string]interface{}
	CreatedAt time.Time
}

type TaskResult struct {
	TaskID    string
	Result    *ImageResult
	Error     error
	StartTime time.Time
	EndTime   time.Time
}

type PriorityQueue struct {
	items    []*PriorityItem
	mu       sync.Mutex
	notEmpty *sync.Cond
}

type PriorityItem struct {
	Request GenerateRequest
	Priority int
	AddedAt  time.Time
}

type GeneratorStats struct {
	TotalGenerated  int64
	TotalFailed     int64
	ActiveWorkers   int64
	QueueLength     int64
	AverageDuration time.Duration
	mu              sync.Mutex
	durations       []time.Duration
}

func NewConcurrentGenerator() *ConcurrentGenerator {
	ctx, cancel := context.WithCancel(context.Background())
	
	generator := &ConcurrentGenerator{
		imageGenerator: NewImageGenerator(),
		workerPool: &WorkerPool{
			workers:    runtime.NumCPU(),
			taskQueue:  make(chan *Task, runtime.NumCPU()*10),
			resultChan: make(chan *TaskResult, runtime.NumCPU()*10),
			ctx:        ctx,
			cancel:     cancel,
		},
		priorityQueue: &PriorityQueue{
			items: make([]*PriorityItem, 0),
		},
		stats: &GeneratorStats{
			durations: make([]time.Duration, 0, 1000),
		},
	}

	generator.priorityQueue.notEmpty = sync.NewCond(&generator.priorityQueue.mu)
	
	generator.startWorkers()
	
	return generator
}

func (g *ConcurrentGenerator) startWorkers() {
	for i := 0; i < g.workerPool.workers; i++ {
		g.workerPool.wg.Add(1)
		go g.worker(i)
	}
}

func (g *ConcurrentGenerator) worker(id int) {
	defer g.workerPool.wg.Done()
	
	atomic.AddInt64(&g.stats.ActiveWorkers, 1)
	defer atomic.AddInt64(&g.stats.ActiveWorkers, -1)
	
	for {
		select {
		case <-g.workerPool.ctx.Done():
			return
		case task := <-g.priorityQueue.Dequeue():
			g.processTask(task)
		case task := <-g.workerPool.taskQueue:
			if task != nil {
				g.processTask(task)
			}
		}
	}
}

func (g *ConcurrentGenerator) processTask(task *Task) {
	startTime := time.Now()
	
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()
	
	result, err := g.imageGenerator.GenerateImageWithPool(ctx, task.Type)
	
	endTime := time.Now()
	duration := endTime.Sub(startTime)
	
	taskResult := &TaskResult{
		TaskID:    task.ID,
		Result:    result,
		Error:     err,
		StartTime: startTime,
		EndTime:   endTime,
	}
	
	select {
	case g.workerPool.resultChan <- taskResult:
	default:
	}
	
	g.updateStats(duration, err != nil)
}

func (g *ConcurrentGenerator) updateStats(duration time.Duration, failed bool) {
	g.stats.mu.Lock()
	defer g.stats.mu.Unlock()
	
	if failed {
		atomic.AddInt64(&g.stats.TotalFailed, 1)
	} else {
		atomic.AddInt64(&g.stats.TotalGenerated, 1)
	}
	
	g.stats.durations = append(g.stats.durations, duration)
	if len(g.stats.durations) > 1000 {
		g.stats.durations = g.stats.durations[1:]
	}
	
	var total time.Duration
	for _, d := range g.stats.durations {
		total += d
	}
	g.stats.AverageDuration = total / time.Duration(len(g.stats.durations))
}

func (g *ConcurrentGenerator) ConcurrentGenerate(ctx context.Context, requests []GenerateRequest) ([]*ImageResult, []error) {
	results := make([]*ImageResult, len(requests))
	errors := make([]error, len(requests))
	
	var wg sync.WaitGroup
	var mu sync.Mutex
	resultIndex := 0
	
	tasks := make([]*Task, len(requests))
	for i, req := range requests {
		taskID := fmt.Sprintf("task_%d_%d", time.Now().UnixNano(), i)
		tasks[i] = &Task{
			ID:       taskID,
			Type:     req.Type,
			Priority: 0,
			Metadata: req.Metadata,
			CreatedAt: time.Now(),
		}
	}
	
	for _, task := range tasks {
		wg.Add(1)
		go func(t *Task, idx int) {
			defer wg.Done()
			
			taskCtx, cancel := context.WithTimeout(ctx, 30*time.Second)
			defer cancel()
			
			result, err := g.imageGenerator.GenerateImageWithPool(taskCtx, t.Type)
			
			mu.Lock()
			results[idx] = result
			errors[idx] = err
			mu.Unlock()
			
		}(task, resultIndex)
		resultIndex++
	}
	
	wg.Wait()
	
	return results, errors
}

func (g *ConcurrentGenerator) EnqueueWithPriority(ctx context.Context, request GenerateRequest, priority int) error {
	select {
	case <-ctx.Done():
		return ctx.Err()
	default:
	}
	
	item := &PriorityItem{
		Request:  request,
		Priority: priority,
		AddedAt:  time.Now(),
	}
	
	g.priorityQueue.Enqueue(item)
	
	atomic.AddInt64(&g.stats.QueueLength, 1)
	
	return nil
}

func (g *ConcurrentGenerator) GenerateWithPool(ctx context.Context, captchaType string) (*ImageResult, error) {
	return g.imageGenerator.GenerateImageWithPool(ctx, captchaType)
}

func (g *ConcurrentGenerator) Close() error {
	g.workerPool.cancel()
	g.workerPool.wg.Wait()
	close(g.workerPool.taskQueue)
	close(g.workerPool.resultChan)
	return nil
}

func (g *ConcurrentGenerator) GetStats() GeneratorStats {
	return GeneratorStats{
		TotalGenerated:  atomic.LoadInt64(&g.stats.TotalGenerated),
		TotalFailed:     atomic.LoadInt64(&g.stats.TotalFailed),
		ActiveWorkers:   atomic.LoadInt64(&g.stats.ActiveWorkers),
		QueueLength:     atomic.LoadInt64(&g.stats.QueueLength),
		AverageDuration: g.stats.AverageDuration,
	}
}

func (q *PriorityQueue) Enqueue(item *PriorityItem) {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	q.items = append(q.items, item)
	q.bubbleUp(len(q.items) - 1)
	q.notEmpty.Signal()
}

func (q *PriorityQueue) Dequeue() <-chan *Task {
	ch := make(chan *Task, 1)
	
	go func() {
		q.mu.Lock()
		defer q.mu.Unlock()
		
		for len(q.items) == 0 {
			q.notEmpty.Wait()
		}
		
		item := q.items[0]
		last := q.items[len(q.items)-1]
		q.items = q.items[:len(q.items)-1]
		
		if len(q.items) > 0 {
			q.items[0] = last
			q.bubbleDown(0)
		}
		
		task := &Task{
			ID:        item.Request.RequestID,
			Type:      item.Request.Type,
			Priority:  item.Priority,
			Metadata:  item.Request.Metadata,
			CreatedAt: item.AddedAt,
		}
		
		atomic.AddInt64(&getGeneratorStats().QueueLength, -1)
		
		ch <- task
	}()
	
	return ch
}

func (q *PriorityQueue) bubbleUp(idx int) {
	for idx > 0 {
		parent := (idx - 1) / 2
		if q.items[parent].Priority >= q.items[idx].Priority {
			break
		}
		q.items[parent], q.items[idx] = q.items[idx], q.items[parent]
		idx = parent
	}
}

func (q *PriorityQueue) bubbleDown(idx int) {
	length := len(q.items)
	for {
		left := 2*idx + 1
		right := 2*idx + 2
		largest := idx
		
		if left < length && q.items[left].Priority > q.items[largest].Priority {
			largest = left
		}
		
		if right < length && q.items[right].Priority > q.items[largest].Priority {
			largest = right
		}
		
		if largest == idx {
			break
		}
		
		q.items[idx], q.items[largest] = q.items[largest], q.items[idx]
		idx = largest
	}
}

func (q *PriorityQueue) Len() int {
	q.mu.Lock()
	defer q.mu.Unlock()
	return len(q.items)
}

func (q *PriorityQueue) Peek() *PriorityItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	if len(q.items) == 0 {
		return nil
	}
	return q.items[0]
}

func (q *PriorityQueue) Clear() {
	q.mu.Lock()
	defer q.mu.Unlock()
	q.items = make([]*PriorityItem, 0)
}

func (q *PriorityQueue) GetAll() []*PriorityItem {
	q.mu.Lock()
	defer q.mu.Unlock()
	
	result := make([]*PriorityItem, len(q.items))
	copy(result, q.items)
	
	sort.Slice(result, func(i, j int) bool {
		if result[i].Priority == result[j].Priority {
			return result[i].AddedAt.Before(result[j].AddedAt)
		}
		return result[i].Priority > result[j].Priority
	})
	
	return result
}

var globalStats *GeneratorStats

func getGeneratorStats() *GeneratorStats {
	if globalStats == nil {
		globalStats = &GeneratorStats{
			durations: make([]time.Duration, 0, 1000),
		}
	}
	return globalStats
}
