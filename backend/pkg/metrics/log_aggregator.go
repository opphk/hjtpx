package metrics

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"sync/atomic"
	"time"

	"github.com/prometheus/client_golang/prometheus"
)

type LogAggregator struct {
	collector     *PrometheusCollector
	lokiURL       string
	buffer        []LogEntry
	bufferMu      sync.Mutex
	bufferSize    int
	flushInterval time.Duration
	stopCh        chan struct{}
	wg            sync.WaitGroup
	client        *http.Client
	enabled       atomic.Bool
	batchSize     int
	metrics       *LogMetrics
}

type LogEntry struct {
	Timestamp  time.Time              `json:"ts"`
	Stream     map[string]string     `json:"stream"`
	Values     [][2]string          `json:"values"`
	LogLevel   string               `json:"level,omitempty"`
	Message    string               `json:"message"`
	Component  string               `json:"component,omitempty"`
	TraceID    string               `json:"trace_id,omitempty"`
	SpanID     string               `json:"span_id,omitempty"`
	Duration   time.Duration        `json:"duration,omitempty"`
	StatusCode int                  `json:"status_code,omitempty"`
	Error      string               `json:"error,omitempty"`
	Metadata   map[string]any       `json:"metadata,omitempty"`
}

type LogMetrics struct {
	logsReceivedTotal  prometheus.Counter
	logsSentTotal      prometheus.Counter
	logsSentFailed     prometheus.Counter
	logBufferSize      prometheus.Gauge
	logBatchSize       prometheus.Histogram
	logProcessingTime  prometheus.Histogram
	lokiRequestLatency prometheus.Histogram
}

func newLogMetrics(registry *prometheus.Registry) *LogMetrics {
	lm := &LogMetrics{
		logsReceivedTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "log_aggregator_logs_received_total",
				Help: "Total logs received by aggregator",
			},
		),
		logsSentTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "log_aggregator_logs_sent_total",
				Help: "Total logs sent to Loki",
			},
		),
		logsSentFailed: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "log_aggregator_logs_sent_failed_total",
				Help: "Total logs failed to send to Loki",
			},
		),
		logBufferSize: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "log_aggregator_buffer_size",
				Help: "Current log buffer size",
			},
		),
		logBatchSize: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "log_aggregator_batch_size",
				Help:    "Log batch sizes sent to Loki",
				Buckets: []float64{1, 5, 10, 25, 50, 100, 250, 500, 1000},
			},
		),
		logProcessingTime: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "log_aggregator_processing_seconds",
				Help:    "Log processing time in seconds",
				Buckets: []float64{.001, .005, .01, .025, .05, .1, .25, .5},
			},
		),
		lokiRequestLatency: prometheus.NewHistogram(
			prometheus.HistogramOpts{
				Name:    "log_aggregator_loki_request_seconds",
				Help:    "Loki API request latency in seconds",
				Buckets: []float64{.01, .025, .05, .1, .25, .5, 1},
			},
		),
	}

	registry.MustRegister(lm.logsReceivedTotal)
	registry.MustRegister(lm.logsSentTotal)
	registry.MustRegister(lm.logsSentFailed)
	registry.MustRegister(lm.logBufferSize)
	registry.MustRegister(lm.logBatchSize)
	registry.MustRegister(lm.logProcessingTime)
	registry.MustRegister(lm.lokiRequestLatency)

	return lm
}

func NewLogAggregator(lokiURL string, bufferSize int, flushInterval time.Duration) *LogAggregator {
	registry := prometheus.NewRegistry()
	la := &LogAggregator{
		lokiURL:        lokiURL,
		buffer:         make([]LogEntry, 0, bufferSize),
		bufferSize:     bufferSize,
		flushInterval: flushInterval,
		stopCh:        make(chan struct{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		batchSize: 100,
		metrics:   newLogMetrics(registry),
	}
	la.enabled.Store(true)
	return la
}

func (la *LogAggregator) Start() {
	la.wg.Add(2)
	go la.bufferFlusher()
	go la.metricsCollector()
}

func (la *LogAggregator) Stop() {
	close(la.stopCh)
	la.wg.Wait()
	la.flush()
}

func (la *LogAggregator) bufferFlusher() {
	defer la.wg.Done()
	ticker := time.NewTicker(la.flushInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			la.flush()
		case <-la.stopCh:
			return
		}
	}
}

func (la *LogAggregator) metricsCollector() {
	defer la.wg.Done()
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			la.bufferMu.Lock()
			la.metrics.logBufferSize.Set(float64(len(la.buffer)))
			la.bufferMu.Unlock()
		case <-la.stopCh:
			return
		}
	}
}

func (la *LogAggregator) Collect(entry LogEntry) {
	if !la.enabled.Load() {
		return
	}

	start := time.Now()
	defer func() {
		la.metrics.logProcessingTime.Observe(time.Since(start).Seconds())
	}()

	la.metrics.logsReceivedTotal.Inc()

	la.bufferMu.Lock()
	la.buffer = append(la.buffer, entry)
	shouldFlush := len(la.buffer) >= la.bufferSize
	la.bufferMu.Unlock()

	if shouldFlush {
		go la.flush()
	}
}

func (la *LogAggregator) flush() {
	la.bufferMu.Lock()
	if len(la.buffer) == 0 {
		la.bufferMu.Unlock()
		return
	}

	entries := make([]LogEntry, len(la.buffer))
	copy(entries, la.buffer)
	la.buffer = la.buffer[:0]
	la.bufferMu.Unlock()

	if err := la.sendToLoki(entries); err != nil {
		la.metrics.logsSentFailed.Add(float64(len(entries)))
		return
	}

	la.metrics.logsSentTotal.Add(float64(len(entries)))
	la.metrics.logBatchSize.Observe(float64(len(entries)))
}

func (la *LogAggregator) sendToLoki(entries []LogEntry) error {
	if la.lokiURL == "" {
		return nil
	}

	start := time.Now()
	defer func() {
		la.metrics.lokiRequestLatency.Observe(time.Since(start).Seconds())
	}()

	streams := make(map[string][]LogEntry)
	for _, entry := range entries {
		key := fmt.Sprintf("%v", entry.Stream)
		streams[key] = append(streams[key], entry)
	}

	for _, streamEntries := range streams {
		if err := la.sendStream(streamEntries); err != nil {
			return err
		}
	}

	return nil
}

func (la *LogAggregator) sendStream(entries []LogEntry) error {
	stream := make(map[string]string)
	if len(entries) > 0 {
		stream = entries[0].Stream
	}

	lokiEntries := make([][2]string, 0, len(entries))
	for _, entry := range entries {
		logLine := entry.Message
		if entry.LogLevel != "" {
			logLine = fmt.Sprintf("[%s] %s", entry.LogLevel, entry.Message)
		}
		if entry.Error != "" {
			logLine = fmt.Sprintf("%s | error=%s", logLine, entry.Error)
		}
		if entry.TraceID != "" {
			logLine = fmt.Sprintf("%s | trace_id=%s", logLine, entry.TraceID)
		}
		lokiEntries = append(lokiEntries, [2]string{
			fmt.Sprintf("%d", entry.Timestamp.UnixNano()),
			logLine,
		})
	}

	payload := map[string]interface{}{
		"streams": []map[string]interface{}{
			{
				"stream": stream,
				"values": lokiEntries,
			},
		},
	}

	jsonPayload, err := json.Marshal(payload)
	if err != nil {
		return err
	}

	resp, err := la.client.Post(
		la.lokiURL+"/loki/api/v1/push",
		"application/json",
		bytes.NewBuffer(jsonPayload),
	)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusNoContent && resp.StatusCode != http.StatusOK {
		body, _ := io.ReadAll(resp.Body)
		return fmt.Errorf("loki returned status %d: %s", resp.StatusCode, string(body))
	}

	return nil
}

func (la *LogAggregator) SetEnabled(enabled bool) {
	la.enabled.Store(enabled)
}

func (la *LogAggregator) GetBufferSize() int {
	la.bufferMu.Lock()
	defer la.bufferMu.Unlock()
	return len(la.buffer)
}
