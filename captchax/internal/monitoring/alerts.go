package monitoring

import (
	"sync"
	"sync/atomic"
	"time"
)

type AlertThresholds struct {
	CPUWarning        float64
	CPUCritical       float64
	MemoryWarning     float64
	MemoryCritical    float64
	LatencyWarning    time.Duration
	LatencyCritical   time.Duration
	ErrorRateWarning  float64
	ErrorRateCritical float64
	QPSMinWarning     float64
	ResponseSizeLimit int64
}

func DefaultAlertThresholds() AlertThresholds {
	return AlertThresholds{
		CPUWarning:        70.0,
		CPUCritical:       90.0,
		MemoryWarning:     75.0,
		MemoryCritical:    90.0,
		LatencyWarning:    500 * time.Millisecond,
		LatencyCritical:   2 * time.Second,
		ErrorRateWarning:  1.0,
		ErrorRateCritical: 5.0,
		QPSMinWarning:     10.0,
		ResponseSizeLimit: 10 * 1024 * 1024,
	}
}

type AlertLevel int

const (
	AlertLevelNormal AlertLevel = iota
	AlertLevelWarning
	AlertLevelCritical
)

func (a AlertLevel) String() string {
	switch a {
	case AlertLevelNormal:
		return "normal"
	case AlertLevelWarning:
		return "warning"
	case AlertLevelCritical:
		return "critical"
	default:
		return "unknown"
	}
}

type PerformanceAlert struct {
	Level         AlertLevel
	MetricName    string
	Message       string
	CurrentValue  float64
	Threshold     float64
	Timestamp     time.Time
	Duration      time.Duration
	Recommendation string
}

func (a *PerformanceAlert) IsResolved() bool {
	return a.Level == AlertLevelNormal
}

type AlertManager struct {
	thresholds    AlertThresholds
	activeAlerts  map[string]*PerformanceAlert
	alertHistory  []*PerformanceAlert
	mu            sync.RWMutex
	maxHistory    int
	notifyCallback func(*PerformanceAlert)
}

func NewAlertManager(thresholds AlertThresholds) *AlertManager {
	return &AlertManager{
		thresholds:   thresholds,
		activeAlerts: make(map[string]*PerformanceAlert),
		alertHistory: make([]*PerformanceAlert, 0, 100),
		maxHistory:   1000,
	}
}

func (am *AlertManager) SetThresholds(thresholds AlertThresholds) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.thresholds = thresholds
}

func (am *AlertManager) SetNotifyCallback(callback func(*PerformanceAlert)) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.notifyCallback = callback
}

func (am *AlertManager) CheckMetric(name string, value float64, threshold float64, level AlertLevel) *PerformanceAlert {
	am.mu.Lock()
	defer am.mu.Unlock()

	alert := &PerformanceAlert{
		Level:        level,
		MetricName:   name,
		CurrentValue: value,
		Threshold:    threshold,
		Timestamp:    time.Now(),
	}

	switch level {
	case AlertLevelNormal:
		if existing, ok := am.activeAlerts[name]; ok {
			if existing.Level != AlertLevelNormal {
				existing.Level = AlertLevelNormal
				existing.Timestamp = time.Now()
			}
		}
		return nil

	case AlertLevelWarning:
		alert.Message = generateWarningMessage(name, value, threshold)
		alert.Recommendation = getRecommendation(name, level)

	case AlertLevelCritical:
		alert.Message = generateCriticalMessage(name, value, threshold)
		alert.Recommendation = getRecommendation(name, level)
	}

	existing, exists := am.activeAlerts[name]
	if exists {
		if existing.Level != level {
			existing.Level = level
			existing.CurrentValue = value
			existing.Timestamp = time.Now()
			existing.Message = alert.Message
			existing.Recommendation = alert.Recommendation
		}
		existing.Duration = time.Since(existing.Timestamp)
		alert = existing
	} else {
		am.activeAlerts[name] = alert
		am.alertHistory = append(am.alertHistory, alert)
		if len(am.alertHistory) > am.maxHistory {
			am.alertHistory = am.alertHistory[len(am.alertHistory)-am.maxHistory:]
		}
	}

	if am.notifyCallback != nil {
		go am.notifyCallback(alert)
	}

	return alert
}

func generateWarningMessage(name string, value float64, threshold float64) string {
	return "Warning: " + name + " is " + formatFloat(value) + ", approaching threshold " + formatFloat(threshold)
}

func generateCriticalMessage(name string, value float64, threshold float64) string {
	return "Critical: " + name + " has exceeded threshold " + formatFloat(threshold) + " (current: " + formatFloat(value) + ")"
}

func getRecommendation(name string, level AlertLevel) string {
	if level == AlertLevelNormal {
		return "Metric returned to normal levels."
	}

	recommendations := map[string]map[AlertLevel]string{
		"cpu_usage": {
			AlertLevelWarning:   "Consider scaling horizontally or optimizing CPU-intensive operations.",
			AlertLevelCritical: "Immediate action required: scale infrastructure or optimize critical paths.",
		},
		"memory_usage": {
			AlertLevelWarning:   "Monitor memory trends and prepare for potential scaling.",
			AlertLevelCritical:  "Critical memory pressure: implement memory optimizations or scale immediately.",
		},
		"latency_p99": {
			AlertLevelWarning:   "High latency detected: review slow queries and optimize critical paths.",
			AlertLevelCritical:  "Severe latency issues: implement caching and optimize database queries.",
		},
		"error_rate": {
			AlertLevelWarning:   "Error rate elevated: investigate recent deployments and error logs.",
			AlertLevelCritical:  "Critical error rate: implement circuit breakers and rollback if needed.",
		},
		"qps": {
			AlertLevelWarning:   "QPS lower than expected: check for bottlenecks or resource constraints.",
			AlertLevelCritical:  "QPS critically low: investigate service degradation immediately.",
		},
	}

	if rec, ok := recommendations[name]; ok {
		if r, ok := rec[level]; ok {
			return r
		}
	}

	return "Monitor this metric closely and take appropriate action."
}

func formatFloat(f float64) string {
	return time.Duration(int64(f * float64(time.Second))).String()
}

func (am *AlertManager) GetActiveAlerts() []*PerformanceAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	alerts := make([]*PerformanceAlert, 0, len(am.activeAlerts))
	for _, alert := range am.activeAlerts {
		if alert.Level != AlertLevelNormal {
			alerts = append(alerts, alert)
		}
	}
	return alerts
}

func (am *AlertManager) GetAlertHistory(limit int) []*PerformanceAlert {
	am.mu.RLock()
	defer am.mu.RUnlock()

	if limit <= 0 || limit > len(am.alertHistory) {
		limit = len(am.alertHistory)
	}

	history := make([]*PerformanceAlert, limit)
	copy(history, am.alertHistory[len(am.alertHistory)-limit:])
	return history
}

func (am *AlertManager) ClearAlert(name string) {
	am.mu.Lock()
	defer am.mu.Unlock()

	if alert, ok := am.activeAlerts[name]; ok {
		alert.Level = AlertLevelNormal
		alert.Timestamp = time.Now()
	}
}

func (am *AlertManager) ClearAllAlerts() {
	am.mu.Lock()
	defer am.mu.Unlock()

	for _, alert := range am.activeAlerts {
		alert.Level = AlertLevelNormal
		alert.Timestamp = time.Now()
	}
}

func (am *AlertManager) CheckLatency(latency time.Duration) *PerformanceAlert {
	am.mu.RLock()
	thresholds := am.thresholds
	am.mu.RUnlock()

	var level AlertLevel
	var threshold float64

	if latency >= thresholds.LatencyCritical {
		level = AlertLevelCritical
		threshold = float64(thresholds.LatencyCritical)
	} else if latency >= thresholds.LatencyWarning {
		level = AlertLevelWarning
		threshold = float64(thresholds.LatencyWarning)
	} else {
		level = AlertLevelNormal
		threshold = float64(thresholds.LatencyWarning)
	}

	return am.CheckMetric("latency_p99", float64(latency), threshold, level)
}

func (am *AlertManager) CheckErrorRate(totalRequests, failedRequests int64) *PerformanceAlert {
	if totalRequests == 0 {
		return nil
	}

	am.mu.RLock()
	thresholds := am.thresholds
	am.mu.RUnlock()

	errorRate := float64(failedRequests) / float64(totalRequests) * 100

	var level AlertLevel
	var threshold float64

	if errorRate >= thresholds.ErrorRateCritical {
		level = AlertLevelCritical
		threshold = thresholds.ErrorRateCritical
	} else if errorRate >= thresholds.ErrorRateWarning {
		level = AlertLevelWarning
		threshold = thresholds.ErrorRateWarning
	} else {
		level = AlertLevelNormal
		threshold = thresholds.ErrorRateWarning
	}

	return am.CheckMetric("error_rate", errorRate, threshold, level)
}

func (am *AlertManager) CheckQPS(qps float64) *PerformanceAlert {
	am.mu.RLock()
	thresholds := am.thresholds
	am.mu.RUnlock()

	var level AlertLevel
	threshold := thresholds.QPSMinWarning

	if qps < thresholds.QPSMinWarning {
		if qps < thresholds.QPSMinWarning*0.5 {
			level = AlertLevelCritical
		} else {
			level = AlertLevelWarning
		}
	} else {
		level = AlertLevelNormal
	}

	return am.CheckMetric("qps", qps, threshold, level)
}

type MetricsCollector struct {
	counters   map[string]*Counter
	gauges     map[string]*Gauge
	histograms map[string]*Histogram
	timers     map[string]*Timer
	mu         sync.RWMutex
}

type Counter struct {
	value int64
}

func (c *Counter) Inc() {
	atomic.AddInt64(&c.value, 1)
}

func (c *Counter) Add(v int64) {
	atomic.AddInt64(&c.value, v)
}

func (c *Counter) Value() int64 {
	return atomic.LoadInt64(&c.value)
}

type Gauge struct {
	value int64
}

func (g *Gauge) Set(v int64) {
	atomic.StoreInt64(&g.value, v)
}

func (g *Gauge) Value() int64 {
	return atomic.LoadInt64(&g.value)
}

func (g *Gauge) Inc() {
	atomic.AddInt64(&g.value, 1)
}

func (g *Gauge) Dec() {
	atomic.AddInt64(&g.value, -1)
}

type Timer struct {
	counts   [11]int64
	sum      int64
	min      int64
	max      int64
	mu       sync.Mutex
}

func (t *Timer) Observe(d time.Duration) {
	ms := d.Milliseconds()
	bucket := min(int(ms/10), 10)

	atomic.AddInt64(&t.counts[bucket], 1)
	atomic.AddInt64(&t.sum, int64(d))

	t.mu.Lock()
	if t.min == 0 || int64(d) < t.min {
		t.min = int64(d)
	}
	if int64(d) > t.max {
		t.max = int64(d)
	}
	t.mu.Unlock()
}

func (t *Timer) GetStats() (count int64, avg, min, max time.Duration) {
	var sum int64
	for i := 0; i < 11; i++ {
		count += atomic.LoadInt64(&t.counts[i])
	}
	sum = atomic.LoadInt64(&t.sum)

	t.mu.Lock()
	minDur := time.Duration(t.min)
	maxDur := time.Duration(t.max)
	t.mu.Unlock()

	if count > 0 {
		avg = time.Duration(sum / count)
	}

	return count, avg, minDur, maxDur
}

func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		counters:   make(map[string]*Counter),
		gauges:     make(map[string]*Gauge),
		histograms: make(map[string]*Histogram),
		timers:     make(map[string]*Timer),
	}
}

func (mc *MetricsCollector) GetCounter(name string) *Counter {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if c, ok := mc.counters[name]; ok {
		return c
	}

	c := &Counter{}
	mc.counters[name] = c
	return c
}

func (mc *MetricsCollector) GetGauge(name string) *Gauge {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if g, ok := mc.gauges[name]; ok {
		return g
	}

	g := &Gauge{}
	mc.gauges[name] = g
	return g
}

func (mc *MetricsCollector) GetTimer(name string) *Timer {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if t, ok := mc.timers[name]; ok {
		return t
	}

	t := &Timer{}
	mc.timers[name] = t
	return t
}

func (mc *MetricsCollector) GetAllStats() map[string]interface{} {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	stats := make(map[string]interface{})

	for name, counter := range mc.counters {
		stats["counter_"+name] = counter.Value()
	}

	for name, gauge := range mc.gauges {
		stats["gauge_"+name] = gauge.Value()
	}

	for name, timer := range mc.timers {
		count, avg, min, max := timer.GetStats()
		stats["timer_"+name+"_count"] = count
		stats["timer_"+name+"_avg"] = avg.String()
		stats["timer_"+name+"_min"] = min.String()
		stats["timer_"+name+"_max"] = max.String()
	}

	return stats
}
