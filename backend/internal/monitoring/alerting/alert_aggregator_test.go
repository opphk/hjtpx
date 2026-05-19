package alerting

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestAlertAggregator(t *testing.T) {
	aa := NewAlertAggregator(1*time.Minute, []string{"service", "instance"}, 100)

	alert := &Alert{
		Alertname: "TestAlert",
		Severity:  SeverityWarning,
		Status:    StatusFiring,
		Labels: map[string]string{
			"service":  "auth",
			"instance": "server-1",
		},
		Annotations: map[string]string{
			"summary":     "Test alert",
			"description": "This is a test alert",
		},
		StartsAt: time.Now(),
	}

	aa.AddAlert(alert)

	result := aa.GetAggregationResult()
	assert.Equal(t, int64(1), result.TotalAlerts)
	assert.Equal(t, 1, result.TotalGroups)
	assert.Len(t, result.Groups, 1)

	alertResolved := &Alert{
		Alertname:   "TestAlert",
		Severity:    SeverityWarning,
		Status:      StatusResolved,
		Fingerprint: alert.Fingerprint,
		Labels: map[string]string{
			"service":  "auth",
			"instance": "server-1",
		},
		EndsAt: time.Now(),
	}

	aa.AddAlert(alertResolved)

	result = aa.GetAggregationResult()
	assert.Equal(t, int64(0), result.TotalAlerts)
	assert.Equal(t, 0, result.TotalGroups)
}

func TestAlertGrouping(t *testing.T) {
	aa := NewAlertAggregator(1*time.Minute, []string{"service"}, 100)

	alert1 := &Alert{
		Alertname: "HighErrorRate",
		Severity:  SeverityCritical,
		Status:    StatusFiring,
		Labels: map[string]string{
			"service": "auth",
		},
		StartsAt: time.Now(),
	}

	alert2 := &Alert{
		Alertname: "HighErrorRate",
		Severity:  SeverityCritical,
		Status:    StatusFiring,
		Labels: map[string]string{
			"service": "auth",
		},
		StartsAt: time.Now(),
	}

	alert3 := &Alert{
		Alertname: "HighErrorRate",
		Severity:  SeverityWarning,
		Status:    StatusFiring,
		Labels: map[string]string{
			"service": "api",
		},
		StartsAt: time.Now(),
	}

	aa.AddAlert(alert1)
	aa.AddAlert(alert2)
	aa.AddAlert(alert3)

	result := aa.GetAggregationResult()
	assert.Equal(t, int64(3), result.TotalAlerts)
	assert.Equal(t, 2, result.TotalGroups)

	for _, group := range result.Groups {
		if group.GroupName == "HighErrorRate" && len(group.Alerts) == 2 {
			assert.Equal(t, SeverityCritical, group.Severity)
		}
	}
}

func TestNoiseReduction(t *testing.T) {
	aa := NewAlertAggregator(1*time.Minute, []string{"service"}, 100)

	for i := 0; i < 10; i++ {
		alert := &Alert{
			Alertname: "TestAlert",
			Severity:  SeverityInfo,
			Status:    StatusFiring,
			Labels: map[string]string{
				"service":  "test",
				"instance": "server-1",
			},
			StartsAt: time.Now(),
		}
		aa.AddAlert(alert)
	}

	reduction := aa.ApplyNoiseReduction(StrategyFingerprint)
	assert.Greater(t, reduction, 0.0)
}

func TestSuppressionRules(t *testing.T) {
	aa := NewAlertAggregator(1*time.Minute, []string{"service"}, 100)

	rule := SuppressionRule{
		Name: "Suppress warnings when critical fires",
		SourceMatch: map[string]string{
			"severity": "critical",
		},
		TargetMatch: map[string]string{
			"severity": "warning",
		},
		Equal:   []string{"service"},
		Enabled: true,
	}
	aa.AddSuppressionRule(rule)

	criticalAlert := &Alert{
		Alertname: "ServiceDown",
		Severity:  SeverityCritical,
		Status:    StatusFiring,
		Labels: map[string]string{
			"service": "auth",
		},
		StartsAt: time.Now(),
	}

	warningAlert := &Alert{
		Alertname: "HighLatency",
		Severity:  SeverityWarning,
		Status:    StatusFiring,
		Labels: map[string]string{
			"service": "auth",
		},
		StartsAt: time.Now(),
	}

	aa.AddAlert(criticalAlert)
	aa.AddAlert(warningAlert)
	aa.ApplySuppressionRules()

	result := aa.GetAggregationResult()
	assert.Equal(t, int64(2), result.TotalAlerts)
}

func TestAlertSeverity(t *testing.T) {
	assert.Equal(t, "critical", string(SeverityCritical))
	assert.Equal(t, "warning", string(SeverityWarning))
	assert.Equal(t, "info", string(SeverityInfo))
	assert.Equal(t, "unknown", string(SeverityUnknown))

	assert.True(t, SeverityCritical > SeverityWarning)
	assert.True(t, SeverityWarning > SeverityInfo)
	assert.True(t, SeverityInfo > SeverityUnknown)
}

func TestAlertStatus(t *testing.T) {
	assert.Equal(t, "firing", string(StatusFiring))
	assert.Equal(t, "resolved", string(StatusResolved))
}
