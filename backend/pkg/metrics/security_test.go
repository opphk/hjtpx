package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewSecurityMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)
	assert.NotNil(t, sm.attackAttemptsTotal)
	assert.NotNil(t, sm.blockedRequestsTotal)
	assert.NotNil(t, sm.botDetectionTotal)
	assert.NotNil(t, sm.threatLevel)
}

func TestRecordAttackAttempt(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordAttackAttempt("xss", "high")
	sm.RecordAttackAttempt("sql_injection", "critical")
	sm.RecordAttackAttempt("ddos", "high")
	sm.RecordAttackAttempt("brute_force", "medium")
}

func TestRecordBlockedRequest(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordBlockedRequest("rate_limit")
	sm.RecordBlockedRequest("blacklist")
	sm.RecordBlockedRequest("bot_detection")
	sm.RecordBlockedRequest("invalid_token")
}

func TestUpdateRiskScoreDistribution(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.UpdateRiskScoreDistribution(60, 30, 8, 2)
	sm.UpdateRiskScoreDistribution(70, 20, 7, 3)
}

func TestRecordBotDetection(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordBotDetection("fingerprint", "block")
	sm.RecordBotDetection("behavior", "challenge")
	sm.RecordBotDetection("headless_browser", "block")
}

func TestRecordDDoSDetection(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordDDoSDetection()
	sm.RecordDDoSDetection()
	sm.RecordDDoSDetection()
}

func TestRecordBruteForceDetection(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordBruteForceDetection()
	sm.RecordBruteForceDetection()
}

func TestRecordReplayAttack(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordReplayAttack()
	sm.RecordReplayAttack()
	sm.RecordReplayAttack()
}

func TestRecordRateLimitHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordRateLimitHit("global", "ip")
	sm.RecordRateLimitHit("per_user", "user")
	sm.RecordRateLimitHit("per_endpoint", "ip")
}

func TestRecordRateLimitExceeded(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordRateLimitExceeded()
	sm.RecordRateLimitExceeded()
	sm.RecordRateLimitExceeded()
}

func TestRecordAuthenticationFailure(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordAuthenticationFailure()
	sm.RecordAuthenticationFailure()
	sm.RecordAuthenticationFailure()
}

func TestRecordAuthenticationSuccess(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordAuthenticationSuccess()
	sm.RecordAuthenticationSuccess()
	sm.RecordAuthenticationSuccess()
}

func TestRecordCSRFTokenFailure(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordCSRFTokenFailure()
	sm.RecordCSRFTokenFailure()
}

func TestRecordXSSAttempt(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordXSSAttempt()
	sm.RecordXSSAttempt()
	sm.RecordXSSAttempt()
}

func TestRecordSQLInjectionAttempt(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordSQLInjectionAttempt()
	sm.RecordSQLInjectionAttempt()
}

func TestSetSuspiciousIPCount(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.SetSuspiciousIPCount(10)
	sm.SetSuspiciousIPCount(25)
	sm.SetSuspiciousIPCount(5)
}

func TestSetMaliciousRequestRate(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.SetMaliciousRequestRate(0.05)
	sm.SetMaliciousRequestRate(0.15)
	sm.SetMaliciousRequestRate(0.02)
}

func TestSetThreatLevel(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.SetThreatLevel(1)
	sm.SetThreatLevel(2)
	sm.SetThreatLevel(3)
	sm.SetThreatLevel(4)
	sm.SetThreatLevel(5)
}

func TestRecordSecurityEvent(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	sm.RecordSecurityEvent("authentication", "info")
	sm.RecordSecurityEvent("authorization", "warning")
	sm.RecordSecurityEvent("intrusion", "critical")
	sm.RecordSecurityEvent("data_leak", "high")
}

func TestConcurrentSecurityMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			sm.RecordAttackAttempt("xss", "high")
			sm.RecordBlockedRequest("rate_limit")
			sm.RecordBotDetection("fingerprint", "block")
			sm.RecordAuthenticationFailure()
			sm.RecordAuthenticationSuccess()
			sm.RecordSecurityEvent("authentication", "info")
		}(i)
	}
	wg.Wait()
}

func TestSecurityMetricsAllAlertTypes(t *testing.T) {
	registry := prometheus.NewRegistry()
	sm := newSecurityMetrics(registry)
	require.NotNil(t, sm)

	attackTypes := []string{"xss", "sql_injection", "csrf", "path_traversal", "command_injection"}
	severities := []string{"low", "medium", "high", "critical"}

	for _, attackType := range attackTypes {
		for _, severity := range severities {
			sm.RecordAttackAttempt(attackType, severity)
		}
	}
}
