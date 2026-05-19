package alerting

import (
	"encoding/json"
	"fmt"
	"regexp"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/pkg/logger"
)

type AlertSeverity string

const (
	SeverityCritical AlertSeverity = "critical"
	SeverityWarning  AlertSeverity = "warning"
	SeverityInfo     AlertSeverity = "info"
	SeverityUnknown  AlertSeverity = "unknown"
)

type AlertStatus string

const (
	StatusFiring   AlertStatus = "firing"
	StatusResolved AlertStatus = "resolved"
)

type Alert struct {
	ID           string                 `json:"id"`
	Alertname    string                 `json:"alertname"`
	Severity     AlertSeverity          `json:"severity"`
	Status       AlertStatus            `json:"status"`
	Labels       map[string]string      `json:"labels"`
	Annotations  map[string]string      `json:"annotations"`
	StartsAt     time.Time              `json:"starts_at"`
	EndsAt       time.Time              `json:"ends_at"`
	GeneratorURL string                 `json:"generator_url"`
	Fingerprint  string                 `json:"fingerprint"`
	GroupID      string                 `json:"group_id,omitempty"`
	Metadata     map[string]interface{} `json:"metadata,omitempty"`
}

type AlertGroup struct {
	GroupID       string            `json:"group_id"`
	GroupName     string            `json:"group_name"`
	Alerts        []*Alert          `json:"alerts"`
	Severity      AlertSeverity     `json:"severity"`
	FirstSeen     time.Time         `json:"first_seen"`
	LastSeen      time.Time         `json:"last_seen"`
	Count         int               `json:"count"`
	IsSuppressed  bool              `json:"is_suppressed"`
	SuppressedBy  []string          `json:"suppressed_by,omitempty"`
}

type AlertAggregationResult struct {
	TotalAlerts   int64              `json:"total_alerts"`
	TotalGroups   int                `json:"total_groups"`
	Groups        []*AlertGroup      `json:"groups"`
	Stats         AlertStats         `json:"stats"`
	NoiseReduction float64           `json:"noise_reduction"`
}

type AlertStats struct {
	BySeverity    map[AlertSeverity]int64 `json:"by_severity"`
	ByStatus      map[AlertStatus]int64   `json:"by_status"`
	ByService     map[string]int64        `json:"by_service"`
	TimeRange     TimeRange               `json:"time_range"`
	Suppressed    int64                   `json:"suppressed"`
	Active        int64                   `json:"active"`
}

type TimeRange struct {
	Start time.Time `json:"start"`
	End   time.Time `json:"end"`
}

type AlertAggregator struct {
	mu                  sync.RWMutex
	alerts              map[string]*Alert
	groups              map[string]*AlertGroup
	suppressionRules    []SuppressionRule
	dedupWindow         time.Duration
	groupingKeys        []string
	maxAlertsPerGroup   int
}

type SuppressionRule struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	SourceMatch map[string]string      `json:"source_match"`
	TargetMatch map[string]string      `json:"target_match"`
	Equal       []string               `json:"equal"`
	Enabled     bool                   `json:"enabled"`
}

type NoiseReductionStrategy int

const (
	StrategyTimeWindow NoiseReductionStrategy = iota
	StrategyFingerprint
	StrategyPatternMatch
	StrategyMLBased
)

func NewAlertAggregator(dedupWindow time.Duration, groupingKeys []string, maxAlertsPerGroup int) *AlertAggregator {
	return &AlertAggregator{
		alerts:            make(map[string]*Alert),
		groups:            make(map[string]*AlertGroup),
		suppressionRules:  []SuppressionRule{},
		dedupWindow:       dedupWindow,
		groupingKeys:      groupingKeys,
		maxAlertsPerGroup: maxAlertsPerGroup,
	}
}

func (aa *AlertAggregator) AddAlert(alert *Alert) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	if alert.Fingerprint == "" {
		alert.Fingerprint = generateFingerprint(alert)
	}

	existing, exists := aa.alerts[alert.Fingerprint]
	if exists {
		if alert.Status == StatusResolved {
			delete(aa.alerts, alert.Fingerprint)
			aa.removeAlertFromGroups(alert.Fingerprint)
		} else {
			existing.EndsAt = alert.EndsAt
			existing.Annotations = alert.Annotations
			existing.Labels = alert.Labels
		}
		return
	}

	alert.ID = alert.Fingerprint
	aa.alerts[alert.Fingerprint] = alert
	aa.addAlertToGroup(alert)
}

func generateFingerprint(alert *Alert) string {
	data, _ := json.Marshal(struct {
		Alertname string
		Labels    map[string]string
	}{
		Alertname: alert.Alertname,
		Labels:    alert.Labels,
	})
	return fmt.Sprintf("%x", data)
}

func (aa *AlertAggregator) addAlertToGroup(alert *Alert) {
	groupID := aa.generateGroupID(alert)
	group, exists := aa.groups[groupID]

	if !exists {
		group = &AlertGroup{
			GroupID:   groupID,
			GroupName: alert.Alertname,
			Alerts:    make([]*Alert, 0),
			FirstSeen: alert.StartsAt,
			Severity:  alert.Severity,
		}
		aa.groups[groupID] = group
	}

	if len(group.Alerts) < aa.maxAlertsPerGroup {
		alert.GroupID = groupID
		group.Alerts = append(group.Alerts, alert)
	}

	group.LastSeen = alert.StartsAt
	group.Count = len(group.Alerts)

	if alert.Severity > group.Severity {
		group.Severity = alert.Severity
	}
}

func (aa *AlertAggregator) generateGroupID(alert *Alert) string {
	var keyParts []string
	for _, key := range aa.groupingKeys {
		if val, ok := alert.Labels[key]; ok {
			keyParts = append(keyParts, fmt.Sprintf("%s=%s", key, val))
		}
	}

	if len(keyParts) == 0 {
		return alert.Alertname
	}

	sort.Strings(keyParts)
	return fmt.Sprintf("%s:%s", alert.Alertname, join(keyParts, ","))
}

func join(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func (aa *AlertAggregator) removeAlertFromGroups(fingerprint string) {
	for _, group := range aa.groups {
		newAlerts := make([]*Alert, 0)
		for _, alert := range group.Alerts {
			if alert.Fingerprint != fingerprint {
				newAlerts = append(newAlerts, alert)
			}
		}
		group.Alerts = newAlerts
		group.Count = len(group.Alerts)

		if len(group.Alerts) == 0 {
			delete(aa.groups, group.GroupID)
		}
	}
}

func (aa *AlertAggregator) ApplySuppressionRules() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	for _, group := range aa.groups {
		group.IsSuppressed = false
		group.SuppressedBy = nil

		for _, rule := range aa.suppressionRules {
			if !rule.Enabled {
				continue
			}

			if aa.checkSuppressionRule(rule, group) {
				group.IsSuppressed = true
				group.SuppressedBy = append(group.SuppressedBy, rule.ID)
			}
		}
	}
}

func (aa *AlertAggregator) checkSuppressionRule(rule SuppressionRule, group *AlertGroup) bool {
	sourceMatches := aa.findAlertsMatching(rule.SourceMatch)

	for _, sourceAlert := range sourceMatches {
		if sourceAlert.Status != StatusFiring {
			continue
		}

		match := true
		for _, eq := range rule.Equal {
			if sourceAlert.Labels[eq] != group.Alerts[0].Labels[eq] {
				match = false
				break
			}
		}

		if match {
			for _, target := range group.Alerts {
				targetMatch := true
				for k, v := range rule.TargetMatch {
					if target.Labels[k] != v {
						targetMatch = false
						break
					}
				}
				if targetMatch {
					return true
				}
			}
		}
	}

	return false
}

func (aa *AlertAggregator) findAlertsMatching(match map[string]string) []*Alert {
	var result []*Alert
	for _, alert := range aa.alerts {
		matchAll := true
		for k, v := range match {
			if alert.Labels[k] != v {
				matchAll = false
				break
			}
		}
		if matchAll {
			result = append(result, alert)
		}
	}
	return result
}

func (aa *AlertAggregator) AddSuppressionRule(rule SuppressionRule) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	rule.ID = generateUUID()
	aa.suppressionRules = append(aa.suppressionRules, rule)
}

func generateUUID() string {
	return fmt.Sprintf("%d", time.Now().UnixNano())
}

func (aa *AlertAggregator) RemoveSuppressionRule(ruleID string) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	newRules := make([]SuppressionRule, 0)
	for _, rule := range aa.suppressionRules {
		if rule.ID != ruleID {
			newRules = append(newRules, rule)
		}
	}
	aa.suppressionRules = newRules
}

func (aa *AlertAggregator) ApplyNoiseReduction(strategy NoiseReductionStrategy) float64 {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	originalCount := len(aa.alerts)
	removedCount := 0

	switch strategy {
	case StrategyTimeWindow:
		removedCount = aa.dedupByTimeWindow()
	case StrategyFingerprint:
		removedCount = aa.dedupByFingerprint()
	case StrategyPatternMatch:
		removedCount = aa.dedupByPatternMatch()
	}

	if originalCount == 0 {
		return 0.0
	}

	return float64(removedCount) / float64(originalCount) * 100
}

func (aa *AlertAggregator) dedupByTimeWindow() int {
	removed := 0
	seen := make(map[string]time.Time)

	for fingerprint, alert := range aa.alerts {
		if lastSeen, exists := seen[alert.Alertname]; exists {
			if alert.StartsAt.Sub(lastSeen) < aa.dedupWindow {
				delete(aa.alerts, fingerprint)
				removed++
				continue
			}
		}
		seen[alert.Alertname] = alert.StartsAt
	}

	return removed
}

func (aa *AlertAggregator) dedupByFingerprint() int {
	removed := 0
	seen := make(map[string]bool)

	for fingerprint, alert := range aa.alerts {
		key := fmt.Sprintf("%s:%s", alert.Alertname, alert.Labels["instance"])
		if seen[key] {
			delete(aa.alerts, fingerprint)
			removed++
		}
		seen[key] = true
	}

	return removed
}

func (aa *AlertAggregator) dedupByPatternMatch() int {
	removed := 0
	patterns := make(map[string][]string)

	for fingerprint, alert := range aa.alerts {
		instance := alert.Labels["instance"]
		baseName := extractBaseInstance(instance)

		if _, exists := patterns[alert.Alertname]; !exists {
			patterns[alert.Alertname] = make([]string, 0)
		}

		found := false
		for _, existing := range patterns[alert.Alertname] {
			if existing == baseName {
				delete(aa.alerts, fingerprint)
				removed++
				found = true
				break
			}
		}

		if !found {
			patterns[alert.Alertname] = append(patterns[alert.Alertname], baseName)
		}
	}

	return removed
}

func extractBaseInstance(instance string) string {
	re := regexp.MustCompile(`^([a-zA-Z0-9-]+)(-\d+)?(\..*)?$`)
	match := re.FindStringSubmatch(instance)
	if len(match) > 1 {
		return match[1]
	}
	return instance
}

func (aa *AlertAggregator) GetAggregationResult() *AlertAggregationResult {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	result := &AlertAggregationResult{
		TotalAlerts:   int64(len(aa.alerts)),
		TotalGroups:   len(aa.groups),
		Groups:        make([]*AlertGroup, 0),
		NoiseReduction: 0,
	}

	for _, group := range aa.groups {
		if !group.IsSuppressed {
			result.Groups = append(result.Groups, group)
		}
	}

	sort.Slice(result.Groups, func(i, j int) bool {
		return result.Groups[i].Severity > result.Groups[j].Severity
	})

	result.Stats = aa.calculateStats()

	return result
}

func (aa *AlertAggregator) calculateStats() AlertStats {
	stats := AlertStats{
		BySeverity: make(map[AlertSeverity]int64),
		ByStatus:   make(map[AlertStatus]int64),
		ByService:  make(map[string]int64),
		TimeRange: TimeRange{
			Start: time.Now(),
			End:   time.Time{},
		},
	}

	for _, alert := range aa.alerts {
		stats.BySeverity[alert.Severity]++
		stats.ByStatus[alert.Status]++

		if service, ok := alert.Labels["service"]; ok {
			stats.ByService[service]++
		}

		if alert.StartsAt.Before(stats.TimeRange.Start) {
			stats.TimeRange.Start = alert.StartsAt
		}
		if alert.StartsAt.After(stats.TimeRange.End) {
			stats.TimeRange.End = alert.StartsAt
		}

		if alert.Status == StatusFiring {
			stats.Active++
		}
	}

	for _, group := range aa.groups {
		if group.IsSuppressed {
			stats.Suppressed += int64(len(group.Alerts))
		}
	}

	return stats
}

func (aa *AlertAggregator) GetAlertsBySeverity(severity AlertSeverity) []*Alert {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	var result []*Alert
	for _, alert := range aa.alerts {
		if alert.Severity == severity {
			result = append(result, alert)
		}
	}

	sort.Slice(result, func(i, j int) bool {
		return result[i].StartsAt.After(result[j].StartsAt)
	})

	return result
}

func (aa *AlertAggregator) GetAlertByID(id string) *Alert {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.alerts[id]
}

func (aa *AlertAggregator) ResolveAlert(id string) {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	alert, exists := aa.alerts[id]
	if exists {
		alert.Status = StatusResolved
		alert.EndsAt = time.Now()
		delete(aa.alerts, id)
		aa.removeAlertFromGroups(id)
	}
}

func (aa *AlertAggregator) CleanupResolvedAlerts() {
	aa.mu.Lock()
	defer aa.mu.Unlock()

	for fingerprint, alert := range aa.alerts {
		if alert.Status == StatusResolved {
			delete(aa.alerts, fingerprint)
			aa.removeAlertFromGroups(fingerprint)
		}
	}
}

func (aa *AlertAggregator) GetSuppressionRules() []SuppressionRule {
	aa.mu.RLock()
	defer aa.mu.RUnlock()

	return aa.suppressionRules
}

func (aa *AlertAggregator) LogAlert(alert *Alert) {
	level := "info"
	if alert.Severity == SeverityCritical {
		level = "error"
	} else if alert.Severity == SeverityWarning {
		level = "warn"
	}

	message := fmt.Sprintf("[Alert] %s %s - %s", alert.Status, alert.Severity, alert.Alertname)
	logger.Log(level, message)
}
