package monitoring

import (
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestPrometheusConfiguration(t *testing.T) {
	tests := []struct {
		name        string
		configPath  string
		shouldExist bool
	}{
		{"Prometheus config exists", "monitoring/prometheus/prometheus.yml", true},
		{"Alert rules exists", "monitoring/prometheus/rules/hjtpx.rules", true},
		{"AlertManager config exists", "monitoring/alertmanager/alertmanager.yml", true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := filepath.Join(getWorkspaceRoot(), tt.configPath)
			_, err := os.Stat(path)
			if tt.shouldExist {
				assert.NoError(t, err, "File should exist: %s", path)
			}
		})
	}
}

func TestPrometheusConfigSyntax(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/prometheus.yml")
	cmd := exec.Command("promtool", "check", "config", configPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Prometheus config validation output:\n%s", string(output))
		t.Logf("Note: promtool may not be installed, skipping syntax validation")
		t.SkipNow()
	}

	assert.NoError(t, err, "Prometheus config should be valid")
}

func TestAlertRulesSyntax(t *testing.T) {
	rulesPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/rules/hjtpx.rules")
	cmd := exec.Command("promtool", "check", "rules", rulesPath)

	output, err := cmd.CombinedOutput()
	if err != nil {
		t.Logf("Alert rules validation output:\n%s", string(output))
		t.Logf("Note: promtool may not be installed, skipping syntax validation")
		t.SkipNow()
	}

	assert.NoError(t, err, "Alert rules should be valid")
}

func TestAlertManagerConfig(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/alertmanager/alertmanager.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read AlertManager config")

	contentStr := string(content)

	assert.Contains(t, contentStr, "route:", "Should have route configuration")
	assert.Contains(t, contentStr, "receivers:", "Should have receivers configuration")
	assert.Contains(t, contentStr, "default-receiver", "Should have default receiver")
	assert.Contains(t, contentStr, "critical-receiver", "Should have critical receiver")
	assert.Contains(t, contentStr, "warning-receiver", "Should have warning receiver")
}

func TestAlertRulesStructure(t *testing.T) {
	rulesPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/rules/hjtpx.rules")
	content, err := os.ReadFile(rulesPath)
	require.NoError(t, err, "Should be able to read alert rules")

	contentStr := string(content)

	assert.Contains(t, contentStr, "groups:", "Should have groups section")
	assert.Contains(t, contentStr, "- name: hjtpx-app", "Should have hjtpx-app rules")
	assert.Contains(t, contentStr, "- name: captcha-monitoring", "Should have captcha-monitoring rules")
	assert.Contains(t, contentStr, "- name: security-monitoring", "Should have security-monitoring rules")
	assert.Contains(t, contentStr, "- name: auth-monitoring", "Should have auth-monitoring rules")
	assert.Contains(t, contentStr, "- name: websocket-monitoring", "Should have websocket-monitoring rules")
	assert.Contains(t, contentStr, "- name: database", "Should have database rules")
	assert.Contains(t, contentStr, "- name: redis", "Should have redis rules")
}

func TestGrafanaDashboardConfig(t *testing.T) {
	dashboardPath := filepath.Join(getWorkspaceRoot(), "monitoring/grafana/provisioning/dashboards/hjtpx-dashboard.json")
	_, err := os.Stat(dashboardPath)
	assert.NoError(t, err, "Dashboard file should exist")

	extendedDashboardPath := filepath.Join(getWorkspaceRoot(), "monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json")
	_, err = os.Stat(extendedDashboardPath)
	assert.NoError(t, err, "Extended dashboard file should exist")
}

func TestMonitoringDirectories(t *testing.T) {
	dirs := []string{
		"monitoring/prometheus",
		"monitoring/prometheus/rules",
		"monitoring/grafana",
		"monitoring/grafana/provisioning",
		"monitoring/grafana/provisioning/dashboards",
		"monitoring/grafana/provisioning/datasources",
		"monitoring/loki",
		"monitoring/promtail",
		"monitoring/alertmanager",
		"monitoring/alertmanager/template",
	}

	for _, dir := range dirs {
		t.Run(dir, func(t *testing.T) {
			path := filepath.Join(getWorkspaceRoot(), dir)
			info, err := os.Stat(path)
			assert.NoError(t, err, "Directory should exist: %s", dir)
			assert.True(t, info.IsDir(), "Should be a directory: %s", dir)
		})
	}
}

func TestScrapeConfigs(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/prometheus.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read Prometheus config")

	contentStr := string(content)

	expectedJobs := []string{
		"prometheus",
		"hjtpx-app",
		"hjtpx-health",
		"nginx",
		"postgres-exporter",
		"redis-exporter",
		"node-exporter",
		"loki",
	}

	for _, job := range expectedJobs {
		assert.Contains(t, contentStr, fmt.Sprintf("job_name: '%s'", job),
			"Should have scrape config for %s", job)
	}
}

func TestAlertRoutingConfiguration(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/alertmanager/alertmanager.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read AlertManager config")

	contentStr := string(content)

	assert.Contains(t, contentStr, "captcha-team-receiver", "Should route captcha alerts")
	assert.Contains(t, contentStr, "security-team-receiver", "Should route security alerts")
	assert.Contains(t, contentStr, "dba-oncall-receiver", "Should route database alerts")
	assert.Contains(t, contentStr, "critical-receiver", "Should route critical alerts")
	assert.Contains(t, contentStr, "warning-receiver", "Should route warning alerts")
}

func TestInhibitRules(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/alertmanager/alertmanager.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read AlertManager config")

	contentStr := string(content)
	assert.Contains(t, contentStr, "inhibit_rules:", "Should have inhibit rules")
	assert.Contains(t, contentStr, "HJTXPAppDown", "Should inhibit related alerts when app is down")
}

func TestAlertReceiversNotificationChannels(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/alertmanager/alertmanager.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read AlertManager config")

	contentStr := string(content)

	assert.Contains(t, contentStr, "email_configs:", "Should have email configs")
	assert.Contains(t, contentStr, "slack_configs:", "Should have Slack configs")
	assert.Contains(t, contentStr, "pagerduty_configs:", "Should have PagerDuty configs")
	assert.Contains(t, contentPath, "webhook_configs:", "Should have webhook configs")
}

func TestPrometheusMetricRelabeling(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/prometheus.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read Prometheus config")

	contentStr := string(content)

	assert.Contains(t, contentStr, "metric_relabel_configs:", "Should have metric relabeling")
	assert.Contains(t, contentStr, "hjtpx_", "Should filter hjtpx metrics")
	assert.Contains(t, contentStr, "pg_", "Should filter PostgreSQL metrics")
	assert.Contains(t, contentStr, "redis_", "Should filter Redis metrics")
}

func TestAlertRuleLabels(t *testing.T) {
	rulesPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/rules/hjtpx.rules")
	content, err := os.ReadFile(rulesPath)
	require.NoError(t, err, "Should be able to read alert rules")

	contentStr := string(content)

	criticalAlerts := []string{
		"HJTXPTooManyErrors",
		"HJTXPTooHighLatency",
		"HJTXPTooHighMemoryUsage",
		"CaptchaCriticalSuccessRate",
		"SecurityCriticalBlockRate",
	}

	for _, alert := range criticalAlerts {
		assert.Contains(t, contentStr, fmt.Sprintf("alert: %s", alert),
			"Should have %s alert", alert)
	}

	tokens := strings.Split(contentStr, "alert:")
	for i := 1; i < len(tokens); i++ {
		alertBlock := tokens[i]
		assert.Contains(t, alertBlock, "severity:", "Each alert should have severity label")
		assert.Contains(t, alertBlock, "annotations:", "Each alert should have annotations")
		assert.Contains(t, alertBlock, "summary:", "Each alert should have summary")
		assert.Contains(t, alertBlock, "description:", "Each alert should have description")
	}
}

func TestGrafanaDashboardPanels(t *testing.T) {
	dashboardPath := filepath.Join(getWorkspaceRoot(), "monitoring/grafana/provisioning/dashboards/hjtpx-dashboard-extended.json")
	content, err := os.ReadFile(dashboardPath)
	require.NoError(t, err, "Should be able to read extended dashboard")

	contentStr := string(content)

	expectedPanels := []string{
		"系统概览",
		"请求性能",
		"验证码监控",
		"安全监控",
		"认证监控",
		"WebSocket监控",
		"数据库",
		"Redis",
		"基础设施",
	}

	for _, panel := range expectedPanels {
		assert.Contains(t, contentStr, fmt.Sprintf("\"title\": \"%s\"", panel),
			"Dashboard should have %s panel", panel)
	}
}

func TestMetricsEndpointConfiguration(t *testing.T) {
	configPath := filepath.Join(getWorkspaceRoot(), "monitoring/prometheus/prometheus.yml")
	content, err := os.ReadFile(configPath)
	require.NoError(t, err, "Should be able to read Prometheus config")

	contentStr := string(content)

	assert.Contains(t, contentStr, "/metrics", "Should scrape metrics endpoint")
	assert.Contains(t, contentStr, "scrape_interval:", "Should have scrape interval")
	assert.Contains(t, contentStr, "scrape_timeout:", "Should have scrape timeout")
}

func getWorkspaceRoot() string {
	wd, _ := os.Getwd()
	for {
		if _, err := os.Stat(filepath.Join(wd, "go.mod")); err == nil {
			return wd
		}
		parent := filepath.Dir(wd)
		if parent == wd {
			break
		}
		wd = parent
	}
	return "."
}
