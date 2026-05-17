package metrics

import (
	"sync"

	"github.com/prometheus/client_golang/prometheus"
)

type BusinessMetrics struct {
	captchaVerificationTotal     *prometheus.CounterVec
	captchaVerificationDuration *prometheus.HistogramVec
	captchaVerificationByType  *prometheus.GaugeVec

	activeSessions      prometheus.Gauge
	verificationSuccess prometheus.Gauge
	verificationFailure prometheus.Gauge

	userRegistrationsTotal prometheus.Counter
	userLoginsTotal        prometheus.Counter

	applicationUsageTotal *prometheus.CounterVec
	apiKeyUsageTotal      *prometheus.CounterVec

	blacklistHitsTotal *prometheus.CounterVec

	mu sync.RWMutex
}

func newBusinessMetrics(registry *prometheus.Registry) *BusinessMetrics {
	bm := &BusinessMetrics{
		captchaVerificationTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "captcha_verification_total",
				Help: "Total number of CAPTCHA verifications",
			},
			[]string{"type", "status", "application"},
		),
		captchaVerificationDuration: prometheus.NewHistogramVec(
			prometheus.HistogramOpts{
				Name:    "captcha_verification_duration_seconds",
				Help:    "CAPTCHA verification duration in seconds",
				Buckets: []float64{.01, .05, .1, .25, .5, 1, 2.5, 5, 10},
			},
			[]string{"type"},
		),
		captchaVerificationByType: prometheus.NewGaugeVec(
			prometheus.GaugeOpts{
				Name: "captcha_verification_by_type",
				Help: "Current CAPTCHA verifications by type",
			},
			[]string{"type"},
		),
		activeSessions: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "active_sessions_total",
				Help: "Total number of active sessions",
			},
		),
		verificationSuccess: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "verification_success_total",
				Help: "Total number of successful verifications",
			},
		),
		verificationFailure: prometheus.NewGauge(
			prometheus.GaugeOpts{
				Name: "verification_failure_total",
				Help: "Total number of failed verifications",
			},
		),
		userRegistrationsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "user_registrations_total",
				Help: "Total number of user registrations",
			},
		),
		userLoginsTotal: prometheus.NewCounter(
			prometheus.CounterOpts{
				Name: "user_logins_total",
				Help: "Total number of user logins",
			},
		),
		applicationUsageTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "application_usage_total",
				Help: "Total application usage",
			},
			[]string{"application", "action"},
		),
		apiKeyUsageTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "api_key_usage_total",
				Help: "Total API key usage",
			},
			[]string{"application", "action"},
		),
		blacklistHitsTotal: prometheus.NewCounterVec(
			prometheus.CounterOpts{
				Name: "blacklist_hits_total",
				Help: "Total blacklist hits",
			},
			[]string{"type", "action"},
		),
	}

	registry.MustRegister(bm.captchaVerificationTotal)
	registry.MustRegister(bm.captchaVerificationDuration)
	registry.MustRegister(bm.captchaVerificationByType)
	registry.MustRegister(bm.activeSessions)
	registry.MustRegister(bm.verificationSuccess)
	registry.MustRegister(bm.verificationFailure)
	registry.MustRegister(bm.userRegistrationsTotal)
	registry.MustRegister(bm.userLoginsTotal)
	registry.MustRegister(bm.applicationUsageTotal)
	registry.MustRegister(bm.apiKeyUsageTotal)
	registry.MustRegister(bm.blacklistHitsTotal)

	return bm
}

func (bm *BusinessMetrics) RecordCaptchaVerification(captchaType, status, application string, duration float64) {
	bm.captchaVerificationTotal.WithLabelValues(captchaType, status, application).Inc()
	bm.captchaVerificationDuration.WithLabelValues(captchaType).Observe(duration)

	if status == "success" {
		bm.verificationSuccess.Inc()
	} else if status == "failure" {
		bm.verificationFailure.Inc()
	}
}

func (bm *BusinessMetrics) SetActiveSessions(count float64) {
	bm.activeSessions.Set(count)
}

func (bm *BusinessMetrics) IncrementActiveSessions() {
	bm.activeSessions.Inc()
}

func (bm *BusinessMetrics) DecrementActiveSessions() {
	bm.activeSessions.Dec()
}

func (bm *BusinessMetrics) RecordUserRegistration() {
	bm.userRegistrationsTotal.Inc()
}

func (bm *BusinessMetrics) RecordUserLogin() {
	bm.userLoginsTotal.Inc()
}

func (bm *BusinessMetrics) RecordApplicationUsage(application, action string) {
	bm.applicationUsageTotal.WithLabelValues(application, action).Inc()
}

func (bm *BusinessMetrics) RecordAPIKeyUsage(application, action string) {
	bm.apiKeyUsageTotal.WithLabelValues(application, action).Inc()
}

func (bm *BusinessMetrics) RecordBlacklistHit(blacklistType, action string) {
	bm.blacklistHitsTotal.WithLabelValues(blacklistType, action).Inc()
}

func (bm *BusinessMetrics) SetCaptchaTypeCount(captchaType string, count float64) {
	bm.captchaVerificationByType.WithLabelValues(captchaType).Set(count)
}
