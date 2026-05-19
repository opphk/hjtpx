package monitoring

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

type LogAggregator struct {
	provider   string
	endpoints []string
	policy    *config.AggregationPolicy
	buffer    []*LogEntry
	mu        sync.Mutex
	maxBuffer int
	flushTicker *time.Ticker
	stopCh     chan struct{}
	client    *http.Client
	enabled   bool
}

type LogEntry struct {
	Timestamp time.Time              `json:"timestamp"`
	Level     string                  `json:"level"`
	Service   string                  `json:"service"`
	Message   string                  `json:"message"`
	Fields    map[string]interface{}  `json:"fields,omitempty"`
	TraceID   string                  `json:"trace_id,omitempty"`
	SpanID    string                  `json:"span_id,omitempty"`
}

type LogBatch struct {
	Streams []*LogStream `json:"streams"`
}

type LogStream struct {
	Stream map[string]string `json:"stream"`
	Values [][]string       `json:"values"`
}

func NewLogAggregator(cfg *config.LogAggregationConfig) (*LogAggregator, error) {
	if !cfg.Enabled {
		return &LogAggregator{enabled: false}, nil
	}

	aggregator := &LogAggregator{
		provider:   cfg.Provider,
		endpoints:  cfg.Endpoints,
		policy:     &cfg.Aggregation,
		buffer:     make([]*LogEntry, 0, cfg.Aggregation.MaxBatchSize),
		maxBuffer:  cfg.Aggregation.MaxBatchSize,
		flushTicker: time.NewTicker(time.Duration(cfg.Aggregation.FlushInterval) * time.Second),
		stopCh:     make(chan struct{}),
		client: &http.Client{
			Timeout: 30 * time.Second,
		},
		enabled: true,
	}

	return aggregator, nil
}

func (a *LogAggregator) Start() {
	if !a.enabled {
		return
	}

	go a.flushLoop()
	log.Printf("Log aggregator started, provider: %s", a.provider)
}

func (a *LogAggregator) Stop() {
	if !a.enabled {
		return
	}

	close(a.stopCh)
	a.flush()
	log.Println("Log aggregator stopped")
}

func (a *LogAggregator) Log(entry *LogEntry) {
	if !a.enabled {
		return
	}

	a.mu.Lock()
	defer a.mu.Unlock()

	if a.policy != nil {
		if a.policy.ByService {
			if entry.Fields == nil {
				entry.Fields = make(map[string]interface{})
			}
			if _, ok := entry.Fields["service"]; !ok {
				entry.Fields["service"] = entry.Service
			}
		}

		if a.policy.BySeverity {
			if entry.Fields == nil {
				entry.Fields = make(map[string]interface{})
			}
			if _, ok := entry.Fields["level"]; !ok {
				entry.Fields["level"] = entry.Level
			}
		}
	}

	a.buffer = append(a.buffer, entry)

	if len(a.buffer) >= a.maxBuffer {
		go a.flush()
	}
}

func (a *LogAggregator) LogInfo(service, message string, fields map[string]interface{}) {
	a.Log(&LogEntry{
		Timestamp: time.Now(),
		Level:     "info",
		Service:   service,
		Message:   message,
		Fields:    fields,
	})
}

func (a *LogAggregator) LogError(service, message string, err error, fields map[string]interface{}) {
	if fields == nil {
		fields = make(map[string]interface{})
	}
	if err != nil {
		fields["error"] = err.Error()
	}

	a.Log(&LogEntry{
		Timestamp: time.Now(),
		Level:     "error",
		Service:   service,
		Message:   message,
		Fields:    fields,
	})
}

func (a *LogAggregator) LogWarn(service, message string, fields map[string]interface{}) {
	a.Log(&LogEntry{
		Timestamp: time.Now(),
		Level:     "warn",
		Service:   service,
		Message:   message,
		Fields:    fields,
	})
}

func (a *LogAggregator) LogDebug(service, message string, fields map[string]interface{}) {
	a.Log(&LogEntry{
		Timestamp: time.Now(),
		Level:     "debug",
		Service:   service,
		Message:   message,
		Fields:    fields,
	})
}

func (a *LogAggregator) flushLoop() {
	for {
		select {
		case <-a.stopCh:
			return
		case <-a.flushTicker.C:
			a.flush()
		}
	}
}

func (a *LogAggregator) flush() {
	a.mu.Lock()
	if len(a.buffer) == 0 {
		a.mu.Unlock()
		return
	}

	logs := a.buffer
	a.buffer = make([]*LogEntry, 0, a.maxBuffer)
	a.mu.Unlock()

	if err := a.sendLogs(logs); err != nil {
		log.Printf("Failed to send logs: %v", err)
		a.mu.Lock()
		a.buffer = append(logs, a.buffer...)
		a.mu.Unlock()
	}
}

func (a *LogAggregator) sendLogs(logs []*LogEntry) error {
	if len(a.endpoints) == 0 {
		return nil
	}

	switch a.provider {
	case "loki":
		return a.sendToLoki(logs)
	case "elasticsearch":
		return a.sendToElasticsearch(logs)
	default:
		return a.sendToHTTP(logs)
	}
}

func (a *LogAggregator) sendToLoki(logs []*LogEntry) error {
	streams := make(map[string][]*LogEntry)

	for _, log := range logs {
		service := log.Service
		if service == "" {
			service = "unknown"
		}

		streams[service] = append(streams[service], log)
	}

	batch := &LogBatch{
		Streams: make([]*LogStream, 0, len(streams)),
	}

	for service, entries := range streams {
		values := make([][]string, 0, len(entries))
		for _, entry := range entries {
			ts := fmt.Sprintf("%d", entry.Timestamp.UnixNano())
			data, _ := json.Marshal(entry)
			values = append(values, []string{ts, string(data)})
		}

		batch.Streams = append(batch.Streams, &LogStream{
			Stream: map[string]string{"service": service},
			Values: values,
		})
	}

	_, err := json.Marshal(batch)
	if err != nil {
		return fmt.Errorf("failed to marshal batch: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, endpoint := range a.endpoints {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint+"/loki/api/v1/push", nil)
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			log.Printf("Failed to send to Loki endpoint %s: %v", endpoint, err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode == http.StatusNoContent || resp.StatusCode == http.StatusOK {
			return nil
		}
	}

	return fmt.Errorf("failed to send to any Loki endpoint")
}

func (a *LogAggregator) sendToElasticsearch(logs []*LogEntry) error {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	for _, endpoint := range a.endpoints {
		for _, logEntry := range logs {
			_, err := json.Marshal(logEntry)
			if err != nil {
				continue
			}

			url := fmt.Sprintf("%s/logs-%s/_doc", endpoint, logEntry.Service)
			req, err := http.NewRequestWithContext(ctx, "POST", url, nil)
			if err != nil {
				continue
			}
			req.Header.Set("Content-Type", "application/json")

			resp, err := a.client.Do(req)
			if err != nil {
				log.Printf("Failed to send to Elasticsearch: %v", err)
				continue
			}
			resp.Body.Close()
		}
	}

	return nil
}

func (a *LogAggregator) sendToHTTP(logs []*LogEntry) error {
	_, err := json.Marshal(logs)
	if err != nil {
		return fmt.Errorf("failed to marshal logs: %w", err)
	}

	ctx, cancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer cancel()

	for _, endpoint := range a.endpoints {
		req, err := http.NewRequestWithContext(ctx, "POST", endpoint, nil)
		if err != nil {
			continue
		}
		req.Header.Set("Content-Type", "application/json")

		resp, err := a.client.Do(req)
		if err != nil {
			log.Printf("Failed to send to HTTP endpoint: %v", err)
			continue
		}
		resp.Body.Close()

		if resp.StatusCode >= 200 && resp.StatusCode < 300 {
			return nil
		}
	}

	return fmt.Errorf("failed to send to any HTTP endpoint")
}

type AlertAggregator struct {
	provider     string
	groupBy      []string
	timeWindow   time.Duration
	threshold    int
	dedup        *config.DeduplicationConfig
	alerts       map[string]*AggregatedAlert
	mu           sync.RWMutex
	dedupMap     map[string]*DedupEntry
	dedupMu      sync.RWMutex
	stopCh       chan struct{}
	client       *http.Client
	enabled      bool
}

type AggregatedAlert struct {
	ID          string
	Name        string
	GroupKey    string
	Count       int
	FirstSeen   time.Time
	LastSeen    time.Time
	Severity    string
	Service     string
	Description string
	Labels      map[string]string
	Annotations map[string]string
}

type DedupEntry struct {
	Count     int
	FirstTime time.Time
	LastTime  time.Time
}

func NewAlertAggregator(cfg *config.AlertAggregationConfig) (*AlertAggregator, error) {
	if !cfg.Enabled {
		return &AlertAggregator{enabled: false}, nil
	}

	return &AlertAggregator{
		provider:    cfg.Provider,
		groupBy:     cfg.GroupBy,
		timeWindow:  time.Duration(cfg.TimeWindow) * time.Second,
		threshold:   cfg.Threshold,
		dedup:       &cfg.Deduplication,
		alerts:      make(map[string]*AggregatedAlert),
		dedupMap:    make(map[string]*DedupEntry),
		stopCh:      make(chan struct{}),
		client: &http.Client{
			Timeout: 10 * time.Second,
		},
		enabled: true,
	}, nil
}

func (a *AlertAggregator) Start() {
	if !a.enabled {
		return
	}

	go a.cleanupLoop()
	log.Printf("Alert aggregator started, provider: %s", a.provider)
}

func (a *AlertAggregator) Stop() {
	if !a.enabled {
		return
	}

	close(a.stopCh)
	log.Println("Alert aggregator stopped")
}

func (a *AlertAggregator) ProcessAlert(alert *AggregatedAlert) {
	a.mu.Lock()
	defer a.mu.Unlock()

	key := a.makeGroupKey(alert)

	if existing, ok := a.alerts[key]; ok {
		existing.Count++
		existing.LastSeen = time.Now()
	} else {
		alert.Count = 1
		alert.FirstSeen = time.Now()
		alert.LastSeen = time.Now()
		a.alerts[key] = alert
	}

	a.dedupMu.Lock()
	dedupKey := a.makeDedupKey(alert)
	if entry, ok := a.dedupMap[dedupKey]; ok {
		entry.Count++
		entry.LastTime = time.Now()

		if entry.Count > a.dedup.MaxCount {
			delete(a.alerts, key)
		}
	} else {
		a.dedupMap[dedupKey] = &DedupEntry{
			Count:     1,
			FirstTime: time.Now(),
			LastTime:  time.Now(),
		}
	}
	a.dedupMu.Unlock()
}

func (a *AlertAggregator) GetAlerts() []*AggregatedAlert {
	a.mu.RLock()
	defer a.mu.RUnlock()

	alerts := make([]*AggregatedAlert, 0, len(a.alerts))
	for _, alert := range a.alerts {
		alerts = append(alerts, alert)
	}
	return alerts
}

func (a *AlertAggregator) GetAlertByGroupKey(groupKey string) (*AggregatedAlert, bool) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	alert, ok := a.alerts[groupKey]
	return alert, ok
}

func (a *AlertAggregator) ResolveAlert(groupKey string) {
	a.mu.Lock()
	defer a.mu.Unlock()

	delete(a.alerts, groupKey)
}

func (a *AlertAggregator) cleanupLoop() {
	ticker := time.NewTicker(a.timeWindow)
	defer ticker.Stop()

	for {
		select {
		case <-a.stopCh:
			return
		case <-ticker.C:
			a.cleanup()
		}
	}
}

func (a *AlertAggregator) cleanup() {
	a.mu.Lock()
	defer a.mu.Unlock()

	now := time.Now()
	for key, alert := range a.alerts {
		if now.Sub(alert.LastSeen) > a.timeWindow {
			delete(a.alerts, key)
		}
	}

	a.dedupMu.Lock()
	defer a.dedupMu.Unlock()

	for key, entry := range a.dedupMap {
		if now.Sub(entry.LastTime) > time.Duration(a.dedup.WindowSecs)*time.Second {
			delete(a.dedupMap, key)
		}
	}
}

func (a *AlertAggregator) makeGroupKey(alert *AggregatedAlert) string {
	keyParts := make([]string, 0, len(a.groupBy))

	for _, g := range a.groupBy {
		switch g {
		case "service":
			keyParts = append(keyParts, alert.Service)
		case "severity":
			keyParts = append(keyParts, alert.Severity)
		case "alertname":
			keyParts = append(keyParts, alert.Name)
		default:
			if val, ok := alert.Labels[g]; ok {
				keyParts = append(keyParts, val)
			}
		}
	}

	if len(keyParts) == 0 {
		return alert.GroupKey
	}

	result := keyParts[0]
	for i := 1; i < len(keyParts); i++ {
		result += ":" + keyParts[i]
	}
	return result
}

func (a *AlertAggregator) makeDedupKey(alert *AggregatedAlert) string {
	key := alert.Name
	if alert.Service != "" {
		key += ":" + alert.Service
	}
	return key
}

func (a *AlertAggregator) SendToAlertManager(alert *AggregatedAlert) error {
	if !a.enabled {
		return nil
	}

	log.Printf("[AlertManager] Alert fired: %s (count: %d)", alert.Name, alert.Count)
	return nil
}
