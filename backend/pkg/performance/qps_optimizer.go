package performance

import (
	"context"
	"fmt"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type QPSOptimizer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	config        *QPSConfig
	metrics       *QPSMetrics
	batchProcessor *BatchProcessor
	connectionPool *ConnPool
	rateLimiter   *AdaptiveRateLimiter
}

type QPSConfig struct {
	TargetQPS        int
	MaxQPS          int
	BurstSize       int
	BatchSize       int
	BatchTimeout    time.Duration
	ConnPoolSize    int
	EnableBatching  bool
	EnableRateLimit bool
}

type QPSMetrics struct {
	TotalRequests   atomic.Int64
	CurrentQPS      atomic.Int64
	PeakQPS        atomic.Int64
	AvgLatencyMs   atomic.Int64
	P99LatencyMs   atomic.Int64
	BatchHits      atomic.Int64
	BatchMisses    atomic.Int64
	DroppedRequests atomic.Int64
	ThrottledRequests atomic.Int64
	LastUpdate      atomic.Value
}

type BatchProcessor struct {
	mu           sync.RWMutex
	batchSize    int
	timeout      time.Duration
	buffer       []*BatchRequest
	flushTicker  *time.Ticker
}

type BatchRequest struct {
	ID        string
	Data      interface{}
	Timestamp time.Time
	Response  chan *BatchResponse
}

type BatchResponse struct {
	ID   string
	Data interface{}
	Err  error
}

type ConnPool struct {
	mu       sync.RWMutex
	conns    chan *Conn
	maxSize  int
	active   int32
	idle     int32
	created  int64
}

type Conn struct {
	ID      int
	Busy    atomic.Bool
	Created time.Time
}

type AdaptiveRateLimiter struct {
	mu           sync.RWMutex
	rate         int
	burst        int
	tokens       float64
	lastRefill   time.Time
	refillAmount float64
}

const (
	TargetQPS = 20000
	MaxQPS    = 25000
)

func NewQPSOptimizer() *QPSOptimizer {
	ctx, cancel := context.WithCancel(context.Background())

	return &QPSOptimizer{
		ctx:            ctx,
		cancel:         cancel,
		config:         NewQPSConfig(),
		metrics:        &QPSMetrics{},
		batchProcessor: NewBatchProcessor(),
		connectionPool:  NewConnPool(100),
		rateLimiter:    NewAdaptiveRateLimiter(TargetQPS),
	}
}

func NewQPSConfig() *QPSConfig {
	return &QPSConfig{
		TargetQPS:       TargetQPS,
		MaxQPS:          MaxQPS,
		BurstSize:       TargetQPS * 2,
		BatchSize:       100,
		BatchTimeout:    10 * time.Millisecond,
		ConnPoolSize:    100,
		EnableBatching:  true,
		EnableRateLimit: true,
	}
}

func NewBatchProcessor() *BatchProcessor {
	return &BatchProcessor{
		batchSize: 100,
		timeout:   10 * time.Millisecond,
		buffer:    make([]*BatchRequest, 0, 100),
		flushTicker: time.NewTicker(10 * time.Millisecond),
	}
}

func NewConnPool(maxSize int) *ConnPool {
	pool := &ConnPool{
		conns:   make(chan *Conn, maxSize),
		maxSize: maxSize,
	}

	for i := 0; i < maxSize; i++ {
		pool.conns <- &Conn{ID: i, Created: time.Now()}
	}

	return pool
}

func NewAdaptiveRateLimiter(rate int) *AdaptiveRateLimiter {
	return &AdaptiveRateLimiter{
		rate:         rate,
		burst:        rate * 2,
		tokens:       float64(rate),
		lastRefill:   time.Now(),
		refillAmount: float64(rate) / float64(time.Second),
	}
}

func (o *QPSOptimizer) Start() error {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isRunning {
		return nil
	}

	o.isRunning = true

	go o.runQPSMonitor()
	go o.runAdaptiveScaling()
	go o.batchProcessor.runFlush(o.ctx)

	log.Printf("[QPSOptimizer] Started with target QPS: %d", o.config.TargetQPS)
	return nil
}

func (o *QPSOptimizer) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if !o.isRunning {
		return
	}

	o.cancel()
	o.isRunning = false
	log.Println("[QPSOptimizer] Stopped")
}

func (o *QPSOptimizer) ProcessRequest(ctx context.Context, req *Request) (*Response, error) {
	start := time.Now()

	o.metrics.TotalRequests.Add(1)

	if o.config.EnableRateLimit {
		if !o.rateLimiter.Allow() {
			o.metrics.ThrottledRequests.Add(1)
			return nil, fmt.Errorf("rate limit exceeded")
		}
	}

	if o.config.EnableBatching {
		resp := o.batchProcessor.Process(req)
		latency := time.Since(start).Milliseconds()
		o.recordLatency(latency)
		return &Response{
			RequestID: req.ID,
			Data:      req.Data,
			LatencyNs: time.Since(start).Nanoseconds(),
		}, resp.Err
	}

	result := req.Data
	latency := time.Since(start).Milliseconds()
	o.recordLatency(latency)

	return &Response{
		RequestID: req.ID,
		Data:      result,
		LatencyNs: time.Since(start).Nanoseconds(),
	}, nil
}

func (o *QPSOptimizer) recordLatency(latencyMs int64) {
	total := o.metrics.TotalRequests.Load()
	if total == 0 {
		o.metrics.AvgLatencyMs.Store(latencyMs)
		return
	}

	prevAvg := o.metrics.AvgLatencyMs.Load()
	newAvg := (prevAvg*(total-1) + latencyMs) / total
	o.metrics.AvgLatencyMs.Store(newAvg)

	if latencyMs > o.metrics.P99LatencyMs.Load() {
		o.metrics.P99LatencyMs.Store(latencyMs)
	}
}

func (o *QPSOptimizer) runQPSMonitor() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	var lastCount int64
	var window []int64

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			currentCount := o.metrics.TotalRequests.Load()
			qps := currentCount - lastCount
			lastCount = currentCount

			o.metrics.CurrentQPS.Store(qps)

			if qps > o.metrics.PeakQPS.Load() {
				o.metrics.PeakQPS.Store(qps)
			}

			window = append(window, qps)
			if len(window) > 60 {
				window = window[1:]
			}

			o.metrics.LastUpdate.Store(time.Now())
		}
	}
}

func (o *QPSOptimizer) runAdaptiveScaling() {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-o.ctx.Done():
			return
		case <-ticker.C:
			currentQPS := int(o.metrics.CurrentQPS.Load())

			if currentQPS < o.config.TargetQPS*80/100 {
				o.scaleDown()
			} else if currentQPS > o.config.TargetQPS*90/100 {
				o.scaleUp()
			}
		}
	}
}

func (o *QPSOptimizer) scaleUp() {
	o.mu.Lock()
	defer o.mu.Unlock()

	log.Printf("[QPSOptimizer] Scaling up: current target QPS %d", o.config.TargetQPS)
}

func (o *QPSOptimizer) scaleDown() {
	o.mu.Lock()
	defer o.mu.Unlock()

	log.Printf("[QPSOptimizer] Scaling down: current target QPS %d", o.config.TargetQPS)
}

func (b *BatchProcessor) Process(req *Request) *BatchResponse {
	responseCh := make(chan *BatchResponse, 1)

	batchReq := &BatchRequest{
		ID:        req.ID,
		Data:      req.Data,
		Timestamp: time.Now(),
		Response:  responseCh,
	}

	b.mu.Lock()
	b.buffer = append(b.buffer, batchReq)
	shouldFlush := len(b.buffer) >= b.batchSize
	b.mu.Unlock()

	if shouldFlush {
		b.flush()
	}

	select {
	case resp := <-responseCh:
		return resp
	case <-time.After(b.timeout):
		b.mu.Lock()
		for i, br := range b.buffer {
			if br.ID == req.ID {
				b.buffer = append(b.buffer[:i], b.buffer[i+1:]...)
				break
			}
		}
		b.mu.Unlock()
		return &BatchResponse{ID: req.ID, Err: fmt.Errorf("timeout")}
	}
}

func (b *BatchProcessor) flush() {
	b.mu.Lock()
	if len(b.buffer) == 0 {
		b.mu.Unlock()
		return
	}

	batch := b.buffer
	b.buffer = make([]*BatchRequest, 0, b.batchSize)
	b.mu.Unlock()

	go b.processBatch(batch)
}

func (b *BatchProcessor) processBatch(batch []*BatchRequest) {
	for _, req := range batch {
		resp := &BatchResponse{
			ID:   req.ID,
			Data: req.Data,
		}
		req.Response <- resp
	}
}

func (b *BatchProcessor) runFlush(ctx context.Context) {
	for {
		select {
		case <-ctx.Done():
			return
		case <-b.flushTicker.C:
			b.flush()
		}
	}
}

func (p *ConnPool) Get(ctx context.Context) (*Conn, error) {
	select {
	case conn := <-p.conns:
		atomic.AddInt32(&p.active, 1)
		atomic.AddInt32(&p.idle, -1)
		return conn, nil
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		conn := &Conn{
			ID:      int(atomic.AddInt64(&p.created, 1)),
			Created: time.Now(),
		}
		atomic.AddInt32(&p.active, 1)
		return conn, nil
	}
}

func (p *ConnPool) Put(conn *Conn) {
	conn.Busy.Store(false)
	atomic.AddInt32(&p.active, -1)
	atomic.AddInt32(&p.idle, 1)

	select {
	case p.conns <- conn:
	default:
	}
}

func (r *AdaptiveRateLimiter) Allow() bool {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	elapsed := now.Sub(r.lastRefill)
	r.tokens += r.refillAmount * float64(elapsed)
	r.lastRefill = now

	if r.tokens > float64(r.burst) {
		r.tokens = float64(r.burst)
	}

	if r.tokens >= 1 {
		r.tokens--
		return true
	}

	return false
}

func (o *QPSOptimizer) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_requests":     o.metrics.TotalRequests.Load(),
		"current_qps":       o.metrics.CurrentQPS.Load(),
		"peak_qps":          o.metrics.PeakQPS.Load(),
		"target_qps":        o.config.TargetQPS,
		"avg_latency_ms":    o.metrics.AvgLatencyMs.Load(),
		"p99_latency_ms":    o.metrics.P99LatencyMs.Load(),
		"batch_hits":        o.metrics.BatchHits.Load(),
		"batch_misses":      o.metrics.BatchMisses.Load(),
		"dropped_requests":  o.metrics.DroppedRequests.Load(),
		"throttled_requests": o.metrics.ThrottledRequests.Load(),
		"last_update":        o.metrics.LastUpdate.Load(),
	}
}
