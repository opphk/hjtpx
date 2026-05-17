package metrics

import (
	"sync"
	"testing"

	"github.com/prometheus/client_golang/prometheus"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewBusinessMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)
	assert.NotNil(t, bm.captchaVerificationTotal)
	assert.NotNil(t, bm.captchaVerificationDuration)
	assert.NotNil(t, bm.activeSessions)
}

func TestRecordCaptchaVerification(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.RecordCaptchaVerification("slider", "success", "app1", 0.5)
	bm.RecordCaptchaVerification("slider", "failure", "app1", 0.3)
	bm.RecordCaptchaVerification("image", "success", "app2", 0.8)
	bm.RecordCaptchaVerification("gesture", "success", "app1", 1.2)
}

func TestSetActiveSessions(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.SetActiveSessions(100)
	bm.SetActiveSessions(50)
}

func TestIncrementDecrementActiveSessions(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.IncrementActiveSessions()
	bm.IncrementActiveSessions()
	bm.DecrementActiveSessions()
}

func TestRecordUserRegistration(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.RecordUserRegistration()
	bm.RecordUserRegistration()
	bm.RecordUserRegistration()
}

func TestRecordUserLogin(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.RecordUserLogin()
	bm.RecordUserLogin()
}

func TestRecordApplicationUsage(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.RecordApplicationUsage("app1", "create")
	bm.RecordApplicationUsage("app1", "read")
	bm.RecordApplicationUsage("app1", "update")
	bm.RecordApplicationUsage("app2", "delete")
}

func TestRecordAPIKeyUsage(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.RecordAPIKeyUsage("app1", "verify")
	bm.RecordAPIKeyUsage("app1", "verify")
	bm.RecordAPIKeyUsage("app2", "verify")
}

func TestRecordBlacklistHit(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.RecordBlacklistHit("ip", "block")
	bm.RecordBlacklistHit("ip", "challenge")
	bm.RecordBlacklistHit("user", "block")
}

func TestSetCaptchaTypeCount(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	bm.SetCaptchaTypeCount("slider", 50)
	bm.SetCaptchaTypeCount("image", 30)
	bm.SetCaptchaTypeCount("gesture", 20)
}

func TestConcurrentBusinessMetrics(t *testing.T) {
	registry := prometheus.NewRegistry()
	bm := newBusinessMetrics(registry)
	require.NotNil(t, bm)

	var wg sync.WaitGroup
	for i := 0; i < 50; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			bm.RecordCaptchaVerification("slider", "success", "app1", 0.5)
			bm.RecordUserLogin()
			bm.RecordApplicationUsage("app1", "read")
			bm.RecordAPIKeyUsage("app1", "verify")
			bm.RecordBlacklistHit("ip", "block")
		}(i)
	}
	wg.Wait()
}
