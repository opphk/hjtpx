package metrics

import (
	"sort"
	"sync"
	"time"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

var (
	captchaRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_requests_total",
			Help: "Total number of captcha requests",
		},
		[]string{"captcha_type", "status"},
	)

	captchaVerificationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_captcha_verification_duration_seconds",
			Help:    "Captcha verification duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5, 5.0, 10.0},
		},
		[]string{"captcha_type"},
	)

	captchaSuccessRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hjtpx_captcha_success_rate",
			Help: "Captcha verification success rate percentage",
		},
		[]string{"captcha_type"},
	)

	captchaAttemptsPerSession = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_captcha_attempts_per_session",
			Help:    "Number of attempts per captcha session",
			Buckets: []float64{1, 2, 3, 5, 10},
		},
		[]string{"captcha_type"},
	)

	captchaBlockedTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_blocked_total",
			Help: "Total number of blocked captcha requests",
		},
		[]string{"captcha_type", "reason"},
	)

	riskScoreDistribution = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_risk_score_distribution",
			Help:    "Distribution of risk scores",
			Buckets: []float64{0, 10, 20, 30, 40, 50, 60, 70, 80, 90, 100},
		},
		[]string{"decision"},
	)

	businessRequestsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_business_requests_total",
			Help: "Total business API requests",
		},
		[]string{"endpoint", "method", "status_code"},
	)

	businessRequestDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_business_request_duration_seconds",
			Help:    "Business API request duration in seconds",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0, 2.5},
		},
		[]string{"endpoint", "method"},
	)

	activeUsersGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_active_users",
			Help: "Number of currently active users",
		},
	)

	userRegistrationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_user_registrations_total",
			Help: "Total user registrations",
		},
	)

	userLoginsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_user_logins_total",
			Help: "Total user logins",
		},
		[]string{"status"},
	)

	applicationCount = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_applications_total",
			Help: "Total number of applications",
		},
	)

	verificationByStatus = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_verification_by_status_total",
			Help: "Verifications grouped by status",
		},
		[]string{"captcha_type", "status"},
	)

	securityThreatsBlocked = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_security_threats_blocked_total",
			Help: "Total security threats blocked",
		},
		[]string{"threat_type"},
	)

	ddosProtectionTriggers = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_ddos_protection_triggers_total",
			Help: "Total DDoS protection triggers",
		},
	)

	blacklistHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_blacklist_hits_total",
			Help: "Total blacklist hits",
		},
		[]string{"type", "action"},
	)

	rateLimitHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_rate_limit_hits_total",
			Help: "Total rate limit hits",
		},
		[]string{"type", "tier"},
	)

	mlModelInferenceDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_ml_model_inference_duration_seconds",
			Help:    "ML model inference duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5, 1.0},
		},
		[]string{"model_type"},
	)

	mlModelConfidence = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_ml_model_confidence",
			Help:    "ML model prediction confidence",
			Buckets: []float64{0, 0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0},
		},
		[]string{"model_type", "prediction"},
	)

	alertTriggersTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_alert_triggers_total",
			Help: "Total alert triggers",
		},
		[]string{"severity", "rule_name"},
	)

	alertDeliveryLatency = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hjtpx_alert_delivery_latency_seconds",
			Help:    "Alert delivery latency in seconds",
			Buckets: []float64{0.1, 0.5, 1.0, 2.5, 5.0, 10.0, 30.0, 60.0},
		},
	)

	alertEscalationsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_alert_escalations_total",
			Help: "Total alert escalations",
		},
	)

	alertAcknowledgedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_alert_acknowledged_total",
			Help: "Total acknowledged alerts",
		},
	)

	alertResolvedTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_alert_resolved_total",
			Help: "Total resolved alerts",
		},
	)

	activeAlertsGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hjtpx_active_alerts",
			Help: "Number of currently active alerts",
		},
		[]string{"severity"},
	)

	cacheEvictionsTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_cache_evictions_total",
			Help: "Total cache evictions",
		},
	)

	cacheMemoryUsage = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_cache_memory_usage_bytes",
			Help: "Cache memory usage in bytes",
		},
	)

	websocketConnections = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_websocket_connections_active",
			Help: "Active WebSocket connections",
		},
	)

	websocketMessagesTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_websocket_messages_total",
			Help: "Total WebSocket messages",
		},
		[]string{"direction", "status"},
	)

	backupOperationsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_backup_operations_total",
			Help: "Total backup operations",
		},
		[]string{"status"},
	)

	backupDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hjtpx_backup_duration_seconds",
			Help:    "Backup operation duration",
			Buckets: []float64{1, 5, 10, 30, 60, 120, 300, 600},
		},
	)

	slowQueriesTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_slow_queries_total",
			Help: "Total slow queries detected",
		},
	)

	queryErrorsTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_query_errors_total",
			Help: "Total query errors",
		},
		[]string{"query_type", "error_type"},
	)

	slaComplianceGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hjtpx_sla_compliance_percent",
			Help: "SLA compliance percentage",
		},
		[]string{"sla_type"},
	)

	alertAccuracyGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_alert_accuracy_percent",
			Help: "Alert accuracy percentage",
		},
	)
)

type BusinessMetricsCollector struct {
	mu             sync.RWMutex
	captchaStats   map[string]*CaptchaStats
	alertStats     *AlertStats
	requestHistory []RequestRecord
	slaTracker     *slaTracker
}

type CaptchaStats struct {
	Total          uint64
	Success        uint64
	Failed         uint64
	Blocked        uint64
	TotalDuration  int64
	mu             sync.RWMutex
}

type AlertStats struct {
	Total         uint64
	Critical      uint64
	Warning       uint64
	Info          uint64
	Acknowledged  uint64
	Resolved      uint64
	Escalated     uint64
	mu            sync.RWMutex
}

type RequestRecord struct {
	Timestamp   time.Time
	Endpoint    string
	Method      string
	StatusCode  int
	Duration    time.Duration
}

type slaTracker struct {
	requests   []slaRequest
	windowSize time.Duration
	mu         sync.RWMutex
}

type slaRequest struct {
	Timestamp  time.Time
	Duration   time.Duration
	StatusCode int
}

var (
	businessMetrics *BusinessMetricsCollector
	businessOnce   sync.Once
)

func GetBusinessMetrics() *BusinessMetricsCollector {
	businessOnce.Do(func() {
		businessMetrics = &BusinessMetricsCollector{
			captchaStats:   make(map[string]*CaptchaStats),
			alertStats:     &AlertStats{},
			requestHistory: make([]RequestRecord, 0, 10000),
			slaTracker: &slaTracker{
				requests:   make([]slaRequest, 0, 10000),
				windowSize: 5 * time.Minute,
			},
		}
	})
	return businessMetrics
}

func (bm *BusinessMetricsCollector) RecordCaptchaRequest(captchaType, status string, duration time.Duration) {
	bm.mu.Lock()
	stats, exists := bm.captchaStats[captchaType]
	if !exists {
		stats = &CaptchaStats{}
		bm.captchaStats[captchaType] = stats
	}
	bm.mu.Unlock()

	stats.mu.Lock()
	stats.Total++
	stats.TotalDuration += duration.Nanoseconds()
	switch status {
	case "success":
		stats.Success++
	case "failed":
		stats.Failed++
	case "blocked":
		stats.Blocked++
	}
	stats.mu.Unlock()

	captchaRequestsTotal.WithLabelValues(captchaType, status).Inc()
	captchaVerificationDuration.WithLabelValues(captchaType).Observe(duration.Seconds())

	bm.updateCaptchaSuccessRate(captchaType)
}

func (bm *BusinessMetricsCollector) RecordCaptchaBlocked(captchaType, reason string) {
	captchaBlockedTotal.WithLabelValues(captchaType, reason).Inc()
}

func (bm *BusinessMetricsCollector) RecordRiskScore(score float64, decision string) {
	riskScoreDistribution.WithLabelValues(decision).Observe(score)
}

func (bm *BusinessMetricsCollector) RecordAttempt(captchaType string, attempts int) {
	captchaAttemptsPerSession.WithLabelValues(captchaType).Observe(float64(attempts))
}

func (bm *BusinessMetricsCollector) RecordVerification(captchaType, status string) {
	verificationByStatus.WithLabelValues(captchaType, status).Inc()
}

func (bm *BusinessMetricsCollector) updateCaptchaSuccessRate(captchaType string) {
	bm.mu.RLock()
	stats, exists := bm.captchaStats[captchaType]
	bm.mu.RUnlock()

	if !exists {
		return
	}

	stats.mu.RLock()
	total := stats.Total
	success := stats.Success
	stats.mu.RUnlock()

	if total > 0 {
		rate := float64(success) / float64(total) * 100
		captchaSuccessRate.WithLabelValues(captchaType).Set(rate)
	}
}

func (bm *BusinessMetricsCollector) RecordSecurityThreat(threatType string) {
	securityThreatsBlocked.WithLabelValues(threatType).Inc()
}

func (bm *BusinessMetricsCollector) RecordDDoSTrigger() {
	ddosProtectionTriggers.Inc()
}

func (bm *BusinessMetricsCollector) RecordBlacklistHit(blacklistType, action string) {
	blacklistHits.WithLabelValues(blacklistType, action).Inc()
}

func (bm *BusinessMetricsCollector) RecordRateLimitHit(rateLimitType, tier string) {
	rateLimitHits.WithLabelValues(rateLimitType, tier).Inc()
}

func (bm *BusinessMetricsCollector) RecordMLInference(modelType string, duration time.Duration, confidence float64, prediction string) {
	mlModelInferenceDuration.WithLabelValues(modelType).Observe(duration.Seconds())
	mlModelConfidence.WithLabelValues(modelType, prediction).Observe(confidence)
}

func (bm *BusinessMetricsCollector) RecordBusinessRequest(endpoint, method string, statusCode int, duration time.Duration) {
	bm.mu.Lock()
	bm.requestHistory = append(bm.requestHistory, RequestRecord{
		Timestamp:   time.Now(),
		Endpoint:    endpoint,
		Method:      method,
		StatusCode:  statusCode,
		Duration:    duration,
	})
	if len(bm.requestHistory) > 10000 {
		bm.requestHistory = bm.requestHistory[len(bm.requestHistory)-5000:]
	}
	bm.mu.Unlock()

	businessRequestsTotal.WithLabelValues(endpoint, method, string(rune(statusCode))).Inc()
	businessRequestDuration.WithLabelValues(endpoint, method).Observe(duration.Seconds())

	bm.slaTracker.RecordRequest(duration, statusCode)
}

func (bm *BusinessMetricsCollector) RecordUserRegistration() {
	userRegistrationsTotal.Inc()
}

func (bm *BusinessMetricsCollector) RecordUserLogin(status string) {
	userLoginsTotal.WithLabelValues(status).Inc()
}

func (bm *BusinessMetricsCollector) RecordActiveUsers(count int64) {
	activeUsersGauge.Set(float64(count))
}

func (bm *BusinessMetricsCollector) SetApplicationCount(count int64) {
	applicationCount.Set(float64(count))
}

func (bm *BusinessMetricsCollector) RecordAlertTrigger(severity, ruleName string) {
	bm.mu.Lock()
	switch severity {
	case "critical":
		bm.alertStats.Critical++
	case "warning":
		bm.alertStats.Warning++
	case "info":
		bm.alertStats.Info++
	}
	bm.alertStats.Total++
	bm.mu.Unlock()

	alertTriggersTotal.WithLabelValues(severity, ruleName).Inc()
	activeAlertsGauge.WithLabelValues(severity).Inc()
}

func (bm *BusinessMetricsCollector) RecordAlertDeliveryLatency(duration time.Duration) {
	alertDeliveryLatency.Observe(duration.Seconds())
}

func (bm *BusinessMetricsCollector) RecordAlertEscalation() {
	bm.mu.Lock()
	bm.alertStats.Escalated++
	bm.mu.Unlock()
	alertEscalationsTotal.Inc()
}

func (bm *BusinessMetricsCollector) RecordAlertAcknowledged() {
	bm.mu.Lock()
	bm.alertStats.Acknowledged++
	bm.mu.Unlock()
	alertAcknowledgedTotal.Inc()
}

func (bm *BusinessMetricsCollector) RecordAlertResolved(severity string) {
	bm.mu.Lock()
	bm.alertStats.Resolved++
	bm.mu.Unlock()
	alertResolvedTotal.Inc()
	activeAlertsGauge.WithLabelValues(severity).Dec()
}

func (bm *BusinessMetricsCollector) RecordCacheEviction() {
	cacheEvictionsTotal.Inc()
}

func (bm *BusinessMetricsCollector) SetCacheMemoryUsage(bytes int64) {
	cacheMemoryUsage.Set(float64(bytes))
}

func (bm *BusinessMetricsCollector) SetWebSocketConnections(count int64) {
	websocketConnections.Set(float64(count))
}

func (bm *BusinessMetricsCollector) RecordWebSocketMessage(direction string, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	websocketMessagesTotal.WithLabelValues(direction, status).Inc()
}

func (bm *BusinessMetricsCollector) RecordBackupOperation(status string, duration time.Duration) {
	backupOperationsTotal.WithLabelValues(status).Inc()
	backupDuration.Observe(duration.Seconds())
}

func (bm *BusinessMetricsCollector) RecordSlowQuery() {
	slowQueriesTotal.Inc()
}

func (bm *BusinessMetricsCollector) RecordQueryError(queryType, errorType string) {
	queryErrorsTotal.WithLabelValues(queryType, errorType).Inc()
}

func (bm *BusinessMetricsCollector) GetCaptchaStats(captchaType string) (total, success, failed, blocked uint64) {
	bm.mu.RLock()
	stats, exists := bm.captchaStats[captchaType]
	bm.mu.RUnlock()

	if !exists {
		return
	}

	stats.mu.RLock()
	total = stats.Total
	success = stats.Success
	failed = stats.Failed
	blocked = stats.Blocked
	stats.mu.RUnlock()

	return
}

func (bm *BusinessMetricsCollector) GetAlertStats() (total, critical, warning, info, acknowledged, resolved, escalated uint64) {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	total = bm.alertStats.Total
	critical = bm.alertStats.Critical
	warning = bm.alertStats.Warning
	info = bm.alertStats.Info
	acknowledged = bm.alertStats.Acknowledged
	resolved = bm.alertStats.Resolved
	escalated = bm.alertStats.Escalated

	return
}

func (bm *BusinessMetricsCollector) GetMetricsSummary() map[string]interface{} {
	bm.mu.RLock()
	defer bm.mu.RUnlock()

	summary := make(map[string]interface{})

	captchaStats := make(map[string]interface{})
	for captchaType, stats := range bm.captchaStats {
		stats.mu.RLock()
		avgDuration := float64(0)
		if stats.Total > 0 {
			avgDuration = float64(stats.TotalDuration) / float64(stats.Total) / 1e6
		}
		captchaStats[captchaType] = map[string]interface{}{
			"total":          stats.Total,
			"success":        stats.Success,
			"failed":         stats.Failed,
			"blocked":        stats.Blocked,
			"avg_duration_ms": avgDuration,
		}
		stats.mu.RUnlock()
	}
	summary["captcha"] = captchaStats

	summary["alert"] = map[string]interface{}{
		"total":        bm.alertStats.Total,
		"critical":     bm.alertStats.Critical,
		"warning":      bm.alertStats.Warning,
		"info":         bm.alertStats.Info,
		"acknowledged": bm.alertStats.Acknowledged,
		"resolved":     bm.alertStats.Resolved,
		"escalated":    bm.alertStats.Escalated,
	}

	slaAvailability, slaAvgLatency, slaP99Latency := bm.slaTracker.GetSLAMetrics()
	summary["sla"] = map[string]interface{}{
		"availability": slaAvailability,
		"avg_latency":   slaAvgLatency,
		"p99_latency":   slaP99Latency.Seconds(),
	}

	return summary
}

func (st *slaTracker) RecordRequest(duration time.Duration, statusCode int) {
	st.mu.Lock()
	defer st.mu.Unlock()

	st.requests = append(st.requests, slaRequest{
		Timestamp:  time.Now(),
		Duration:   duration,
		StatusCode: statusCode,
	})

	cutoff := time.Now().Add(-st.windowSize)
	var validRequests []slaRequest
	for _, req := range st.requests {
		if req.Timestamp.After(cutoff) {
			validRequests = append(validRequests, req)
		}
	}
	st.requests = validRequests
}

func (st *slaTracker) GetSLAMetrics() (availability, avgLatency float64, p99Latency time.Duration) {
	st.mu.RLock()
	defer st.mu.RUnlock()

	if len(st.requests) == 0 {
		return 100.0, 0, 0
	}

	var totalLatency int64
	successCount := 0
	latencies := make([]time.Duration, 0, len(st.requests))

	for _, req := range st.requests {
		totalLatency += req.Duration.Nanoseconds()
		latencies = append(latencies, req.Duration)
		if req.StatusCode >= 200 && req.StatusCode < 500 {
			successCount++
		}
	}

	availability = float64(successCount) / float64(len(st.requests)) * 100
	avgLatency = float64(totalLatency) / float64(len(st.requests)) / 1e6

	if len(latencies) > 0 {
		sort.Slice(latencies, func(i, j int) bool {
			return latencies[i] < latencies[j]
		})
		p99Index := int(float64(len(latencies)) * 0.99)
		if p99Index >= len(latencies) {
			p99Index = len(latencies) - 1
		}
		p99Latency = latencies[p99Index]
	}

	return
}

func (bm *BusinessMetricsCollector) GetSLAMetrics() (availability, avgLatency float64, p99Latency time.Duration) {
	return bm.slaTracker.GetSLAMetrics()
}

type alertResponseTimeTracker struct {
	mu           sync.RWMutex
	responseTimes map[string][]time.Duration
}

var alertTracker *alertResponseTimeTracker

func init() {
	alertTracker = &alertResponseTimeTracker{
		responseTimes: make(map[string][]time.Duration),
	}
	go alertTracker.cleanup()
}

func (art *alertResponseTimeTracker) RecordResponseTime(alertType string, duration time.Duration) {
	art.mu.Lock()
	defer art.mu.Unlock()

	art.responseTimes[alertType] = append(art.responseTimes[alertType], duration)
	if len(art.responseTimes[alertType]) > 100 {
		art.responseTimes[alertType] = art.responseTimes[alertType][len(art.responseTimes[alertType])-50:]
	}
}

func (art *alertResponseTimeTracker) GetAverageResponseTime(alertType string) time.Duration {
	art.mu.RLock()
	defer art.mu.RUnlock()

	times, exists := art.responseTimes[alertType]
	if !exists || len(times) == 0 {
		return 0
	}

	var total int64
	for _, t := range times {
		total += t.Nanoseconds()
	}
	return time.Duration(total / int64(len(times)))
}

func (art *alertResponseTimeTracker) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		art.mu.Lock()
		now := time.Now()
		for alertType, times := range art.responseTimes {
			var validTimes []time.Duration
			for _, t := range times {
				if now.Sub(time.Now().Add(-t)) < 10*time.Minute {
					validTimes = append(validTimes, t)
				}
			}
			if len(validTimes) == 0 {
				delete(art.responseTimes, alertType)
			} else {
				art.responseTimes[alertType] = validTimes
			}
		}
		art.mu.Unlock()
	}
}

func RecordAlertResponseTime(alertType string, duration time.Duration) {
	alertTracker.RecordResponseTime(alertType, duration)
}

func GetAlertResponseTime(alertType string) time.Duration {
	return alertTracker.GetAverageResponseTime(alertType)
}

var (
	alertResponseTimeGauge = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Name: "hjtpx_alert_response_time_seconds",
			Help: "Alert response time in seconds",
		},
		[]string{"alert_type"},
	)

	alertAccuracyCounter = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_alert_accuracy_total",
			Help: "Alert accuracy tracking",
		},
		[]string{"alert_type", "result"},
	)
)

func RecordAlertResponseTimeWithMetric(alertType string, duration time.Duration) {
	RecordAlertResponseTime(alertType, duration)
	alertResponseTimeGauge.WithLabelValues(alertType).Set(duration.Seconds())
}

func RecordAlertAccuracy(alertType, result string) {
	alertAccuracyCounter.WithLabelValues(alertType, result).Inc()
}

type alertAccuracyTracker struct {
	mu          sync.RWMutex
	trackedAlerts map[string]*alertAccuracyRecord
}

type alertAccuracyRecord struct {
	Total      uint64
	TruePositive uint64
	FalsePositive uint64
	mu          sync.RWMutex
}

var accuracyTracker *alertAccuracyTracker

func init() {
	accuracyTracker = &alertAccuracyTracker{
		trackedAlerts: make(map[string]*alertAccuracyRecord),
	}
}

func (aat *alertAccuracyTracker) TrackAlert(alertType string, isTruePositive bool) {
	aat.mu.Lock()
	record, exists := aat.trackedAlerts[alertType]
	if !exists {
		record = &alertAccuracyRecord{}
		aat.trackedAlerts[alertType] = record
	}
	aat.mu.Unlock()

	record.mu.Lock()
	record.Total++
	if isTruePositive {
		record.TruePositive++
	} else {
		record.FalsePositive++
	}
	record.mu.Unlock()

	accuracy := 0.0
	if record.Total > 0 {
		accuracy = float64(record.TruePositive) / float64(record.Total) * 100
	}
	alertAccuracyGauge.Set(accuracy)
}

func TrackAlertAccuracy(alertType string, isTruePositive bool) {
	accuracyTracker.TrackAlert(alertType, isTruePositive)
	result := "true_positive"
	if !isTruePositive {
		result = "false_positive"
	}
	RecordAlertAccuracy(alertType, result)
}

var (
	captchaGenerationDuration = promauto.NewHistogramVec(
		prometheus.HistogramOpts{
			Name:    "hjtpx_captcha_generation_duration_seconds",
			Help:    "Captcha generation duration in seconds",
			Buckets: []float64{0.01, 0.05, 0.1, 0.25, 0.5, 1.0, 2.0, 3.0, 5.0},
		},
		[]string{"captcha_type"},
	)

	captchaGenerationTotal = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_captcha_generation_total",
			Help: "Total captcha generations",
		},
		[]string{"captcha_type", "status"},
	)

	sessionActiveGauge = promauto.NewGauge(
		prometheus.GaugeOpts{
			Name: "hjtpx_active_sessions",
			Help: "Number of active captcha sessions",
		},
	)

	sessionTimeoutTotal = promauto.NewCounter(
		prometheus.CounterOpts{
			Name: "hjtpx_session_timeout_total",
			Help: "Total session timeouts",
		},
	)

	behaviorAnalysisDuration = promauto.NewHistogram(
		prometheus.HistogramOpts{
			Name:    "hjtpx_behavior_analysis_duration_seconds",
			Help:    "Behavior analysis duration",
			Buckets: []float64{0.001, 0.005, 0.01, 0.025, 0.05, 0.1, 0.25, 0.5},
		},
	)

	proxyDetectionResults = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_proxy_detection_total",
			Help: "Proxy detection results",
		},
		[]string{"result"},
	)

	botDetectionResults = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Name: "hjtpx_bot_detection_total",
			Help: "Bot detection results",
		},
		[]string{"result", "type"},
	)
)

func RecordCaptchaGeneration(captchaType string, duration time.Duration, success bool) {
	status := "success"
	if !success {
		status = "failed"
	}
	captchaGenerationTotal.WithLabelValues(captchaType, status).Inc()
	captchaGenerationDuration.WithLabelValues(captchaType).Observe(duration.Seconds())
}

func RecordSessionActive(count int64) {
	sessionActiveGauge.Set(float64(count))
}

func RecordSessionTimeout() {
	sessionTimeoutTotal.Inc()
}

func RecordBehaviorAnalysis(duration time.Duration) {
	behaviorAnalysisDuration.Observe(duration.Seconds())
}

func RecordProxyDetection(detected bool) {
	result := "clean"
	if detected {
		result = "proxy_detected"
	}
	proxyDetectionResults.WithLabelValues(result).Inc()
}

func RecordBotDetection(isBot bool, detectionType string) {
	result := "human"
	if isBot {
		result = "bot"
	}
	botDetectionResults.WithLabelValues(result, detectionType).Inc()
}

var globalMetrics *BusinessMetricsCollector

func init() {
	globalMetrics = GetBusinessMetrics()
	go func() {
		ticker := time.NewTicker(30 * time.Second)
		defer ticker.Stop()
		for range ticker.C {
			updateSLAMetrics()
		}
	}()
}

func updateSLAMetrics() {
	availability, avgLatency, p99Latency := globalMetrics.GetSLAMetrics()
	slaComplianceGauge.WithLabelValues("availability").Set(availability)
	slaComplianceGauge.WithLabelValues("avg_latency").Set(avgLatency)
	slaComplianceGauge.WithLabelValues("p99_latency").Set(p99Latency.Seconds())
}
