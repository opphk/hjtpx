package metrics

import (
	"github.com/prometheus/client_golang/prometheus"
)

type SecurityMetrics struct {
	attackAttemptsTotal    *prometheus.CounterVec
	blockedRequestsTotal   *prometheus.CounterVec

	riskScoreDistribution *prometheus.GaugeVec

	botDetectionTotal       *prometheus.CounterVec
	ddosDetectionTotal      prometheus.Counter
	bruteForceDetectionTotal prometheus.Counter
	replayAttackTotal      prometheus.Counter

	rateLimitHitsTotal    *prometheus.CounterVec
	rateLimitExceededTotal prometheus.Counter

	authenticationFailures prometheus.Counter
	authenticationSuccess  prometheus.Counter

	csrfTokenFailures prometheus.Counter
	xssAttemptsTotal  prometheus.Counter
	sqlInjectionAttempts prometheus.Counter

	suspiciousIPCount  prometheus.Gauge
	maliciousRequestRate prometheus.Gauge

	threatLevel prometheus.Gauge

	securityEventTotal *prometheus.CounterVec
}

func newSecurityMetrics(registry *prometheus.Registry) *SecurityMetrics {
	sm := &SecurityMetrics{
		attackAttemptsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_attack_attempts_total",
				Help: "Total number of security attack attempts",
			},
			[]string{"type", "severity"},
		),
		blockedRequestsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_blocked_requests_total",
				Help: "Total number of blocked requests",
			},
			[]string{"reason"},
		),
		riskScoreDistribution: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "security_risk_score_distribution",
				Help: "Distribution of risk scores by bucket",
			},
			[]string{"bucket"},
		),
		botDetectionTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_bot_detection_total",
				Help: "Total bot detection events",
			},
			[]string{"detection_type", "action"},
		),
		ddosDetectionTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_ddos_detection_total",
				Help: "Total DDoS attack detections",
			},
		),
		bruteForceDetectionTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_brute_force_detection_total",
				Help: "Total brute force attack detections",
			},
		),
		replayAttackTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_replay_attack_total",
				Help: "Total replay attack detections",
			},
		),
		rateLimitHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_rate_limit_hits_total",
				Help: "Total rate limit hits",
			},
			[]string{"type", "client"},
		),
		rateLimitExceededTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_rate_limit_exceeded_total",
				Help: "Total rate limit exceeded events",
			},
		),
		authenticationFailures: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_authentication_failures_total",
				Help: "Total authentication failures",
			},
		),
		authenticationSuccess: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_authentication_success_total",
				Help: "Total authentication successes",
			},
		),
		csrfTokenFailures: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_csrf_token_failures_total",
				Help: "Total CSRF token failures",
			},
		),
		xssAttemptsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_xss_attempts_total",
				Help: "Total XSS attack attempts",
			},
		),
		sqlInjectionAttempts: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "security_sql_injection_attempts_total",
				Help: "Total SQL injection attempts",
			},
		),
		suspiciousIPCount: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "security_suspicious_ip_count",
				Help: "Number of suspicious IPs",
			},
		),
		maliciousRequestRate: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "security_malicious_request_rate",
				Help: "Rate of malicious requests",
			},
		),
		threatLevel: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "security_threat_level",
				Help: "Current threat level (0-5)",
			},
		),
		securityEventTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "security_events_total",
				Help: "Total security events",
			},
			[]string{"category", "severity"},
		),
	}

	registry.MustRegister(sm.attackAttemptsTotal)
	registry.MustRegister(sm.blockedRequestsTotal)
	registry.MustRegister(sm.riskScoreDistribution)
	registry.MustRegister(sm.botDetectionTotal)
	registry.MustRegister(sm.ddosDetectionTotal)
	registry.MustRegister(sm.bruteForceDetectionTotal)
	registry.MustRegister(sm.replayAttackTotal)
	registry.MustRegister(sm.rateLimitHitsTotal)
	registry.MustRegister(sm.rateLimitExceededTotal)
	registry.MustRegister(sm.authenticationFailures)
	registry.MustRegister(sm.authenticationSuccess)
	registry.MustRegister(sm.csrfTokenFailures)
	registry.MustRegister(sm.xssAttemptsTotal)
	registry.MustRegister(sm.sqlInjectionAttempts)
	registry.MustRegister(sm.suspiciousIPCount)
	registry.MustRegister(sm.maliciousRequestRate)
	registry.MustRegister(sm.threatLevel)
	registry.MustRegister(sm.securityEventTotal)

	return sm
}

func (sm *SecurityMetrics) RecordAttackAttempt(attackType, severity string) {
	sm.attackAttemptsTotal.WithLabelValues(attackType, severity).Inc()
}

func (sm *SecurityMetrics) RecordBlockedRequest(reason string) {
	sm.blockedRequestsTotal.WithLabelValues(reason).Inc()
}

func (sm *SecurityMetrics) UpdateRiskScoreDistribution(low, medium, high, critical float64) {
	sm.riskScoreDistribution.WithLabelValues("low").Set(low)
	sm.riskScoreDistribution.WithLabelValues("medium").Set(medium)
	sm.riskScoreDistribution.WithLabelValues("high").Set(high)
	sm.riskScoreDistribution.WithLabelValues("critical").Set(critical)
}

func (sm *SecurityMetrics) RecordBotDetection(detectionType, action string) {
	sm.botDetectionTotal.WithLabelValues(detectionType, action).Inc()
}

func (sm *SecurityMetrics) RecordDDoSDetection() {
	sm.ddosDetectionTotal.Inc()
}

func (sm *SecurityMetrics) RecordBruteForceDetection() {
	sm.bruteForceDetectionTotal.Inc()
}

func (sm *SecurityMetrics) RecordReplayAttack() {
	sm.replayAttackTotal.Inc()
}

func (sm *SecurityMetrics) RecordRateLimitHit(rateLimitType, client string) {
	sm.rateLimitHitsTotal.WithLabelValues(rateLimitType, client).Inc()
}

func (sm *SecurityMetrics) RecordRateLimitExceeded() {
	sm.rateLimitExceededTotal.Inc()
}

func (sm *SecurityMetrics) RecordAuthenticationFailure() {
	sm.authenticationFailures.Inc()
}

func (sm *SecurityMetrics) RecordAuthenticationSuccess() {
	sm.authenticationSuccess.Inc()
}

func (sm *SecurityMetrics) RecordCSRFTokenFailure() {
	sm.csrfTokenFailures.Inc()
}

func (sm *SecurityMetrics) RecordXSSAttempt() {
	sm.xssAttemptsTotal.Inc()
}

func (sm *SecurityMetrics) RecordSQLInjectionAttempt() {
	sm.sqlInjectionAttempts.Inc()
}

func (sm *SecurityMetrics) SetSuspiciousIPCount(count float64) {
	sm.suspiciousIPCount.Set(count)
}

func (sm *SecurityMetrics) SetMaliciousRequestRate(rate float64) {
	sm.maliciousRequestRate.Set(rate)
}

func (sm *SecurityMetrics) SetThreatLevel(level float64) {
	sm.threatLevel.Set(level)
}

func (sm *SecurityMetrics) RecordSecurityEvent(category, severity string) {
	sm.securityEventTotal.WithLabelValues(category, severity).Inc()
}
