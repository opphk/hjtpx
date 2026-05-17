package metrics

import (
	"context"
	"sync"
	"sync/atomic"
	"time"
)

type AlertManager struct {
	alertHandlers   map[string]AlertHandler
	alertBuffer     chan *Alert
	responseTimes   []time.Duration
	responseMu      sync.RWMutex
	maxBufferSize   int
	flushInterval   time.Duration
	stopCh          chan struct{}
	wg              sync.WaitGroup
	enabled         atomic.Bool
	targetResponseTime time.Duration

	metrics *AlertMetrics
}

type AlertHandler interface {
	Handle(ctx context.Context, alert *Alert) error
	Name() string
	Priority() int
}

type Alert struct {
	ID            string
	Type          string
	Severity      AlertSeverity
	Message       string
	Source        string
	Timestamp     time.Time
	Data          map[string]interface{}
	ResponseTime  time.Duration
	Handlers      []string
}

type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityHigh    AlertSeverity = "high"
	SeverityMedium  AlertSeverity = "medium"
	SeverityLow     AlertSeverity = "low"
	SeverityInfo    AlertSeverity = "info"
)

type AlertMetrics struct {
	alertsReceived   atomic.Uint64
	alertsProcessed  atomic.Uint64
	alertsFailed     atomic.Uint64
	responseTimeSum  atomic.Uint64
	responseCount    atomic.Uint64
}

func (am *AlertMetrics) RecordResponseTime(duration time.Duration) {
	am.responseTimeSum.Add(uint64(duration.Nanoseconds()))
	am.responseCount.Add(1)
}

func (am *AlertMetrics) GetAverageResponseTime() time.Duration {
	count := am.responseCount.Load()
	if count == 0 {
		return 0
	}
	sum := am.responseTimeSum.Load()
	return time.Duration(sum / count)
}

type AlertHandlerFunc func(ctx context.Context, alert *Alert) error

func (f AlertHandlerFunc) Handle(ctx context.Context, alert *Alert) error {
	return f(ctx, alert)
}

func (f AlertHandlerFunc) Name() string {
	return "anonymous"
}

func (f AlertHandlerFunc) Priority() int {
	return 0
}

func NewAlertManager(maxBufferSize int, flushInterval time.Duration) *AlertManager {
	am := &AlertManager{
		alertHandlers:   make(map[string]AlertHandler),
		alertBuffer:     make(chan *Alert, maxBufferSize),
		responseTimes:   make([]time.Duration, 0, 1000),
		maxBufferSize:   maxBufferSize,
		flushInterval:   flushInterval,
		stopCh:         make(chan struct{}),
		metrics:        &AlertMetrics{},
		targetResponseTime: 5 * time.Second,
	}
	am.enabled.Store(true)
	return am
}

func (am *AlertManager) Start(ctx context.Context) {
	am.wg.Add(2)
	go am.alertProcessor(ctx)
	go am.metricsCollector()
}

func (am *AlertManager) Stop() {
	close(am.stopCh)
	am.wg.Wait()
}

func (am *AlertManager) RegisterHandler(handler AlertHandler) {
	am.alertHandlers[handler.Name()] = handler
}

func (am *AlertManager) UnregisterHandler(name string) {
	delete(am.alertHandlers, name)
}

func (am *AlertManager) Send(alert *Alert) bool {
	if !am.enabled.Load() {
		return false
	}

	am.metrics.alertsReceived.Add(1)
	alert.Timestamp = time.Now()

	select {
	case am.alertBuffer <- alert:
		return true
	default:
		return false
	}
}

func (am *AlertManager) alertProcessor(ctx context.Context) {
	defer am.wg.Done()

	ticker := time.NewTicker(am.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case alert := <-am.alertBuffer:
			am.processAlert(ctx, alert)
		case <-ticker.C:
			am.flushBuffer(ctx)
		case <-am.stopCh:
			am.flushBuffer(ctx)
			return
		}
	}
}

func (am *AlertManager) processAlert(ctx context.Context, alert *Alert) {
	startTime := time.Now()

	handlers := am.getSortedHandlers()

	var wg sync.WaitGroup
	errCh := make(chan error, len(handlers))

	for _, handler := range handlers {
		wg.Add(1)
		go func(h AlertHandler) {
			defer wg.Done()
			if err := h.Handle(ctx, alert); err != nil {
				select {
				case errCh <- err:
				default:
				}
			}
		}(handler)
	}

	wg.Wait()
	close(errCh)

	responseTime := time.Since(startTime)
	alert.ResponseTime = responseTime

	am.responseMu.Lock()
	am.responseTimes = append(am.responseTimes, responseTime)
	if len(am.responseTimes) > 1000 {
		am.responseTimes = am.responseTimes[len(am.responseTimes)-1000:]
	}
	am.responseMu.Unlock()

	am.metrics.RecordResponseTime(responseTime)

	select {
	case err := <-errCh:
		if err != nil {
			am.metrics.alertsFailed.Add(1)
			return
		}
	default:
	}

	am.metrics.alertsProcessed.Add(1)
}

func (am *AlertManager) flushBuffer(ctx context.Context) {
	for {
		select {
		case alert := <-am.alertBuffer:
			am.processAlert(ctx, alert)
		default:
			return
		}
	}
}

func (am *AlertManager) getSortedHandlers() []AlertHandler {
	handlers := make([]AlertHandler, 0, len(am.alertHandlers))
	for _, handler := range am.alertHandlers {
		handlers = append(handlers, handler)
	}

	for i := 0; i < len(handlers)-1; i++ {
		for j := i + 1; j < len(handlers); j++ {
			if handlers[i].Priority() > handlers[j].Priority() {
				handlers[i], handlers[j] = handlers[j], handlers[i]
			}
		}
	}

	return handlers
}

func (am *AlertManager) metricsCollector() {
	defer am.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			am.updateMetrics()
		case <-am.stopCh:
			return
		}
	}
}

func (am *AlertManager) updateMetrics() {
}

func (am *AlertManager) GetStats() AlertStats {
	am.responseMu.RLock()
	defer am.responseMu.RUnlock()

	var avg, p50, p95, p99 time.Duration
	count := len(am.responseTimes)

	if count > 0 {
		sorted := make([]time.Duration, count)
		copy(sorted, am.responseTimes)
		for i := 0; i < count-1; i++ {
			for j := i + 1; j < count; j++ {
				if sorted[j] < sorted[i] {
					sorted[i], sorted[j] = sorted[j], sorted[i]
				}
			}
		}

		sum := time.Duration(0)
		for _, t := range sorted {
			sum += t
		}
		avg = sum / time.Duration(count)

		p50 = sorted[count/2]
		p95 = sorted[int(float64(count)*0.95)]
		p99 = sorted[int(float64(count)*0.99)]
	}

	return AlertStats{
		TotalReceived:  am.metrics.alertsReceived.Load(),
		TotalProcessed: am.metrics.alertsProcessed.Load(),
		TotalFailed:    am.metrics.alertsFailed.Load(),
		AverageResponseTime: avg,
		P50ResponseTime:     p50,
		P95ResponseTime:     p95,
		P99ResponseTime:     p99,
		BufferSize:    len(am.alertBuffer),
		HandlerCount:  len(am.alertHandlers),
	}
}

type AlertStats struct {
	TotalReceived      uint64
	TotalProcessed    uint64
	TotalFailed       uint64
	AverageResponseTime time.Duration
	P50ResponseTime     time.Duration
	P95ResponseTime     time.Duration
	P99ResponseTime     time.Duration
	BufferSize        int
	HandlerCount      int
}

func (am *AlertManager) SetEnabled(enabled bool) {
	am.enabled.Store(enabled)
}

func (am *AlertManager) SetTargetResponseTime(duration time.Duration) {
	am.targetResponseTime = duration
}

func (am *AlertManager) MeetsTargetResponseTime() bool {
	avg := am.metrics.GetAverageResponseTime()
	return avg <= am.targetResponseTime
}
