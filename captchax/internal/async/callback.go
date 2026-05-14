package async

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"sync"
	"sync/atomic"
	"time"
)

type CallbackType string

const (
	CallbackTypeHTTP    CallbackType = "http"
	CallbackTypeWebhook CallbackType = "webhook"
)

type VerificationResult struct {
	CaptchaID   string                 `json:"captcha_id"`
	AppID       string                 `json:"app_id"`
	Success     bool                   `json:"success"`
	Score       float64                `json:"score,omitempty"`
	Message     string                 `json:"message,omitempty"`
	VerifiedAt  time.Time              `json:"verified_at"`
	Metadata    map[string]interface{} `json:"metadata,omitempty"`
}

type CallbackRequest struct {
	Type       CallbackType       `json:"type"`
	URL        string             `json:"url"`
	Method     string             `json:"method"`
	Headers    map[string]string  `json:"headers,omitempty"`
	Timeout    time.Duration      `json:"timeout"`
	RetryCount int                `json:"retry_count"`
	Result     VerificationResult `json:"result"`
}

type CallbackStatus string

const (
	CallbackStatusPending  CallbackStatus = "pending"
	CallbackStatusSuccess CallbackStatus = "success"
	CallbackStatusFailed  CallbackStatus = "failed"
	CallbackStatusRetrying CallbackStatus = "retrying"
	CallbackStatusCanceled CallbackStatus = "canceled"
)

type CallbackTask struct {
	ID        string
	Request   CallbackRequest
	Status    CallbackStatus
	Attempts  int32
	CreatedAt time.Time
	UpdatedAt time.Time
	Error     error
}

type CallbackHandler interface {
	Handle(ctx context.Context, result VerificationResult) error
	Type() CallbackType
}

type HTTPCallbackHandler struct {
	client  *http.Client
	Request CallbackRequest
	retryMax int
}

func NewHTTPCallbackHandler(timeout time.Duration, retryMax int) *HTTPCallbackHandler {
	if retryMax < 0 {
		retryMax = 0
	}
	return &HTTPCallbackHandler{
		client: &http.Client{
			Timeout: timeout,
		},
		retryMax: retryMax,
	}
}

func (h *HTTPCallbackHandler) Handle(ctx context.Context, result VerificationResult) error {
	data, err := json.Marshal(result)
	if err != nil {
		return fmt.Errorf("failed to marshal result: %w", err)
	}

	req, err := http.NewRequestWithContext(ctx, h.Request.Method, h.Request.URL, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")
	req.Header.Set("X-Captcha-ID", result.CaptchaID)
	req.Header.Set("X-Timestamp", fmt.Sprintf("%d", result.VerifiedAt.Unix()))

	for k, v := range h.Request.Headers {
		req.Header.Set(k, v)
	}

	var lastErr error
	for attempt := 0; attempt <= h.retryMax; attempt++ {
		if attempt > 0 {
			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(time.Duration(attempt) * time.Second):
			}
		}

		resp, err := h.client.Do(req)
		if err != nil {
			lastErr = err
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
		lastErr = fmt.Errorf("unexpected status code: %d", resp.StatusCode)
	}

	if lastErr != nil {
		return fmt.Errorf("all retry attempts failed: %w", lastErr)
	}
	return nil
}

func (h *HTTPCallbackHandler) Type() CallbackType {
	return CallbackTypeHTTP
}

type CallbackManager struct {
	handlers map[CallbackType]CallbackHandler
	pool     *asyncPool
	taskStore map[string]*CallbackTask
	taskMu   sync.RWMutex
	stats    CallbackStats
	statsMu  sync.RWMutex
	cancelFunc context.CancelFunc
}

type CallbackStats struct {
	TotalSubmitted int64
	TotalSuccess   int64
	TotalFailed    int64
	TotalRetrying  int64
	PendingCount   int64
}

type asyncPool struct {
	workers   int
	taskQueue chan *CallbackTask
	wg        sync.WaitGroup
	ctx       context.Context
	cancel    context.CancelFunc
	stats     struct {
		processed atomic.Int64
		failed    atomic.Int64
	}
}

func newAsyncPool(workers int, queueSize int) *asyncPool {
	ctx, cancel := context.WithCancel(context.Background())

	if workers <= 0 {
		workers = 10
	}
	if queueSize <= 0 {
		queueSize = workers * 10
	}

	return &asyncPool{
		workers:   workers,
		taskQueue: make(chan *CallbackTask, queueSize),
		ctx:       ctx,
		cancel:    cancel,
	}
}

func (ap *asyncPool) Start() {
	for i := 0; i < ap.workers; i++ {
		ap.wg.Add(1)
		go ap.worker(i)
	}
}

func (ap *asyncPool) worker(id int) {
	defer ap.wg.Done()

	for {
		select {
		case <-ap.ctx.Done():
			return
		case task, ok := <-ap.taskQueue:
			if !ok {
				return
			}
			ap.processTask(task)
		}
	}
}

func (ap *asyncPool) processTask(task *CallbackTask) {
	defer func() {
		if r := recover(); r != nil {
			task.Status = CallbackStatusFailed
			task.Error = fmt.Errorf("panic recovered: %v", r)
		}
	}()

	task.Status = CallbackStatusRetrying
	task.UpdatedAt = time.Now()
}

func (ap *asyncPool) Submit(task *CallbackTask) bool {
	select {
	case ap.taskQueue <- task:
		return true
	default:
		return false
	}
}

func (ap *asyncPool) Stop() {
	ap.cancel()
	close(ap.taskQueue)
	ap.wg.Wait()
}

func NewCallbackManager(workers int) *CallbackManager {
	_, cancel := context.WithCancel(context.Background())

	mm := &CallbackManager{
		handlers:  make(map[CallbackType]CallbackHandler),
		pool:      newAsyncPool(workers, workers*10),
		taskStore: make(map[string]*CallbackTask),
		cancelFunc: cancel,
	}

	mm.pool.Start()

	return mm
}

func (m *CallbackManager) RegisterHandler(handler CallbackHandler) {
	m.handlers[handler.Type()] = handler
}

func (m *CallbackManager) SubmitCallback(ctx context.Context, callbackType CallbackType, url string, result VerificationResult, opts ...CallbackOption) (string, error) {
	req := CallbackRequest{
		Type:       callbackType,
		URL:        url,
		Method:     "POST",
		Timeout:    10 * time.Second,
		RetryCount: 3,
		Result:     result,
	}

	for _, opt := range opts {
		opt(&req)
	}

	task := &CallbackTask{
		ID:        generateTaskID(),
		Request:   req,
		Status:    CallbackStatusPending,
		CreatedAt: time.Now(),
		UpdatedAt: time.Now(),
	}

	m.taskMu.Lock()
	m.taskStore[task.ID] = task
	m.taskMu.Unlock()

	m.incTotalSubmitted()

	if !m.pool.Submit(task) {
		task.Status = CallbackStatusFailed
		task.Error = errors.New("queue is full")
		return task.ID, task.Error
	}

	return task.ID, nil
}

func (m *CallbackManager) SubmitCallbackAsync(ctx context.Context, callbackType CallbackType, url string, result VerificationResult, opts ...CallbackOption) <-chan CallbackResult {
	ch := make(chan CallbackResult, 1)

	go func() {
		taskID, err := m.SubmitCallback(ctx, callbackType, url, result, opts...)
		if err != nil {
			ch <- CallbackResult{
				TaskID: taskID,
				Error:  err,
			}
			return
		}

		status := m.WaitForTask(ctx, taskID)
		ch <- CallbackResult{
			TaskID: taskID,
			Status: status,
		}
	}()

	return ch
}

func (m *CallbackManager) WaitForTask(ctx context.Context, taskID string) CallbackStatus {
	ticker := time.NewTicker(100 * time.Millisecond)
	defer ticker.Stop()

	for {
		select {
		case <-ctx.Done():
			return CallbackStatusCanceled
		case <-ticker.C:
			m.taskMu.RLock()
			task, exists := m.taskStore[taskID]
			m.taskMu.RUnlock()

			if !exists {
				return CallbackStatusFailed
			}

			if task.Status == CallbackStatusSuccess || task.Status == CallbackStatusFailed {
				return task.Status
			}
		}
	}
}

func (m *CallbackManager) GetTaskStatus(taskID string) (*CallbackTask, bool) {
	m.taskMu.RLock()
	defer m.taskMu.RUnlock()

	task, exists := m.taskStore[taskID]
	return task, exists
}

func (m *CallbackManager) CancelTask(taskID string) bool {
	m.taskMu.Lock()
	defer m.taskMu.Unlock()

	task, exists := m.taskStore[taskID]
	if !exists {
		return false
	}

	if task.Status == CallbackStatusPending {
		task.Status = CallbackStatusCanceled
		task.UpdatedAt = time.Now()
		return true
	}

	return false
}

func (m *CallbackManager) GetStats() CallbackStats {
	m.statsMu.Lock()
	defer m.statsMu.Unlock()

	return CallbackStats{
		TotalSubmitted: atomic.LoadInt64(&m.stats.TotalSubmitted),
		TotalSuccess:   atomic.LoadInt64(&m.stats.TotalSuccess),
		TotalFailed:    atomic.LoadInt64(&m.stats.TotalFailed),
		TotalRetrying:  atomic.LoadInt64(&m.stats.TotalRetrying),
		PendingCount:   int64(len(m.pool.taskQueue)),
	}
}

func (m *CallbackManager) incTotalSubmitted() {
	atomic.AddInt64(&m.stats.TotalSubmitted, 1)
}

func (m *CallbackManager) incTotalSuccess() {
	atomic.AddInt64(&m.stats.TotalSuccess, 1)
}

func (m *CallbackManager) incTotalFailed() {
	atomic.AddInt64(&m.stats.TotalFailed, 1)
}

func (m *CallbackManager) incTotalRetrying() {
	atomic.AddInt64(&m.stats.TotalRetrying, 1)
}

func (m *CallbackManager) Stop() {
	m.cancelFunc()
	m.pool.Stop()
}

func (m *CallbackManager) Cleanup(olderThan time.Duration) int {
	m.taskMu.Lock()
	defer m.taskMu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	count := 0

	for id, task := range m.taskStore {
		if task.UpdatedAt.Before(cutoff) {
			delete(m.taskStore, id)
			count++
		}
	}

	return count
}

type CallbackOption func(*CallbackRequest)

func WithMethod(method string) CallbackOption {
	return func(req *CallbackRequest) {
		req.Method = method
	}
}

func WithTimeout(timeout time.Duration) CallbackOption {
	return func(req *CallbackRequest) {
		req.Timeout = timeout
	}
}

func WithHeaders(headers map[string]string) CallbackOption {
	return func(req *CallbackRequest) {
		req.Headers = headers
	}
}

func WithRetryCount(count int) CallbackOption {
	return func(req *CallbackRequest) {
		req.RetryCount = count
	}
}

type CallbackResult struct {
	TaskID string
	Status CallbackStatus
	Error  error
}

func generateTaskID() string {
	return fmt.Sprintf("cb_%d_%d", time.Now().UnixNano(), atomic.AddInt64(&taskCounter, 1))
}

var taskCounter int64

type BatchCallbackManager struct {
	manager       *CallbackManager
	batchSize     int
	flushInterval time.Duration
	pending       []*pendingCallback
	mu            sync.Mutex
	flushChan     chan struct{}
	ctx           context.Context
	cancel        context.CancelFunc
}

type pendingCallback struct {
	CallbackType CallbackType
	URL          string
	Result       VerificationResult
	Opts         []CallbackOption
}

func NewBatchCallbackManager(workers, batchSize int, flushInterval time.Duration) *BatchCallbackManager {
	ctx, cancel := context.WithCancel(context.Background())

	bm := &BatchCallbackManager{
		manager:       NewCallbackManager(workers),
		batchSize:    batchSize,
		flushInterval: flushInterval,
		pending:       make([]*pendingCallback, 0, batchSize),
		flushChan:    make(chan struct{}, 1),
		ctx:          ctx,
		cancel:       cancel,
	}

	go bm.flushLoop()

	return bm
}

func (bm *BatchCallbackManager) Submit(callbackType CallbackType, url string, result VerificationResult, opts ...CallbackOption) {
	bm.mu.Lock()
	defer bm.mu.Unlock()

	bm.pending = append(bm.pending, &pendingCallback{
		CallbackType: callbackType,
		URL:          url,
		Result:       result,
		Opts:         opts,
	})

	if len(bm.pending) >= bm.batchSize {
		select {
		case bm.flushChan <- struct{}{}:
		default:
		}
	}
}

func (bm *BatchCallbackManager) flushLoop() {
	ticker := time.NewTicker(bm.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-bm.ctx.Done():
			bm.flush()
			return
		case <-ticker.C:
			bm.flush()
		case <-bm.flushChan:
			bm.flush()
		}
	}
}

func (bm *BatchCallbackManager) flush() {
	bm.mu.Lock()
	if len(bm.pending) == 0 {
		bm.mu.Unlock()
		return
	}

	toFlush := bm.pending
	bm.pending = make([]*pendingCallback, 0, bm.batchSize)
	bm.mu.Unlock()

	for _, cb := range toFlush {
		bm.manager.SubmitCallback(bm.ctx, cb.CallbackType, cb.URL, cb.Result, cb.Opts...)
	}
}

func (bm *BatchCallbackManager) Stop() {
	bm.cancel()
	bm.manager.Stop()
}

func (bm *BatchCallbackManager) GetStats() CallbackStats {
	return bm.manager.GetStats()
}
