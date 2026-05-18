package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/metrics"
	"gorm.io/gorm"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type AlertService struct {
	db                *gorm.DB
	channels          map[uint]AlertChannel
	rules             []models.AlertRule
	aggregator        *AlertAggregator
	escalationManager *EscalationManager
	stats             *AlertStatistics
	mu                sync.RWMutex
}

type AlertEvent struct {
	EventType string                 `json:"event_type"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
}

type AlertAggregator struct {
	AlertCounts    map[string]*AlertCountItem
	AlertSummaries map[string]*AlertSummary
	mu             sync.RWMutex
	CleanupInterval time.Duration
}

type AlertCountItem struct {
	RuleID         uint
	AggregationKey string
	Count          int
	FirstSeen      time.Time
	LastSeen       time.Time
	Severity       string
	Messages       []string
}

type AlertSummary struct {
	RuleID         uint
	AggregationKey string
	TotalCount     int
	CriticalCount  int
	WarningCount   int
	InfoCount      int
	FirstSeen      time.Time
	LastSeen       time.Time
	UniqueMessages map[string]int
}

type EscalationManager struct {
	escalations map[string]*EscalationState
	mu          sync.RWMutex
	timeouts    map[string]time.Duration
}

type EscalationState struct {
	Level       int
	LastLevelUp time.Time
	NextLevelAt time.Time
	AlertID     uint
	RuleID      uint
}

type AlertStatistics struct {
	TotalTriggered    uint64
	TotalResolved    uint64
	TotalEscalated    uint64
	TotalAcknowledged uint64
	CriticalCount     uint64
	WarningCount      uint64
	InfoCount         uint64
	AvgResponseTime   float64
	LastResetAt       time.Time
	mu                sync.RWMutex
}

type AlertMetrics struct {
	Total         uint64  `json:"total"`
	Active        uint64  `json:"active"`
	Resolved      uint64  `json:"resolved"`
	Acknowledged  uint64  `json:"acknowledged"`
	Escalated     uint64  `json:"escalated"`
	Critical      uint64  `json:"critical"`
	Warning       uint64  `json:"warning"`
	Info          uint64  `json:"info"`
	AvgResponseMs float64 `json:"avg_response_time_ms"`
}

type AlertPerformance struct {
	TotalTriggers    uint64    `json:"total_triggers"`
	TruePositives    uint64    `json:"true_positives"`
	FalsePositives   uint64    `json:"false_positives"`
	Accuracy         float64   `json:"accuracy"`
	AvgResolutionMin float64   `json:"avg_resolution_min"`
	LastCalculated   time.Time `json:"last_calculated"`
}

type alertCountItem = AlertCountItem

func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{
		db:                db,
		channels:          make(map[uint]AlertChannel),
		rules:             []models.AlertRule{},
		aggregator:        NewAlertAggregator(),
		escalationManager: NewEscalationManager(),
		stats:             NewAlertStatistics(),
	}
}

func NewAlertAggregator() *AlertAggregator {
	return &AlertAggregator{
		AlertCounts:    make(map[string]*AlertCountItem),
		AlertSummaries: make(map[string]*AlertSummary),
		CleanupInterval: 5 * time.Minute,
	}
}

func NewEscalationManager() *EscalationManager {
	em := &EscalationManager{
		escalations: make(map[string]*EscalationState),
		timeouts:    make(map[string]time.Duration),
	}
	em.timeouts["warning"] = 15 * time.Minute
	em.timeouts["critical"] = 5 * time.Minute
	em.timeouts["info"] = 30 * time.Minute
	return em
}

func NewAlertStatistics() *AlertStatistics {
	return &AlertStatistics{
		LastResetAt: time.Now(),
	}
}

func (as *AlertService) LoadRules() error {
	var rules []models.AlertRule
	if as.db == nil {
		return nil
	}
	if err := as.db.Where("is_enabled = ?", true).Find(&rules).Error; err != nil {
		return err
	}
	as.mu.Lock()
	as.rules = rules
	as.mu.Unlock()
	return nil
}

func (as *AlertService) LoadChannels() error {
	var dbChannels []models.AlertChannel
	if as.db == nil {
		return nil
	}
	if err := as.db.Where("is_enabled = ?", true).Find(&dbChannels).Error; err != nil {
		return err
	}
	as.mu.Lock()
	defer as.mu.Unlock()
	for _, ch := range dbChannels {
		var config map[string]interface{}
		if err := json.Unmarshal([]byte(ch.Config), &config); err != nil {
			continue
		}
		channel, err := CreateChannel(ch.Type, config)
		if err != nil {
			continue
		}
		as.channels[ch.ID] = channel
	}
	return nil
}

func (as *AlertService) ProcessEvent(event AlertEvent) error {
	as.mu.RLock()
	rules := as.rules
	as.mu.RUnlock()
	for _, rule := range rules {
		if rule.EventType == event.EventType || rule.EventType == "*" {
			if as.evaluateRule(rule, event) {
				as.triggerAlert(rule, event)
			}
		}
	}
	return nil
}

func (as *AlertService) evaluateRule(rule models.AlertRule, event AlertEvent) bool {
	if rule.Condition == "" {
		return true
	}
	return as.parseCondition(rule.Condition, event.Context)
}

func (as *AlertService) parseCondition(condition string, context map[string]interface{}) bool {
	if condition == "" {
		return true
	}
	if strings.Contains(condition, "==") {
		parts := strings.SplitN(condition, "==", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if ctxVal, ok := context[key]; ok {
			return fmt.Sprintf("%v", ctxVal) == value
		}
	} else if strings.Contains(condition, "!=") {
		parts := strings.SplitN(condition, "!=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if ctxVal, ok := context[key]; ok {
			return fmt.Sprintf("%v", ctxVal) != value
		}
	}
	return true
}

func (as *AlertService) triggerAlert(rule models.AlertRule, event AlertEvent) error {
	aggKey := as.buildAggregationKey(rule, event)
	shouldSend, count := as.aggregator.ShouldTriggerAlert(rule.ID, aggKey, rule.AggregationWindow, rule.Threshold, rule.Severity, event.Message)
	now := time.Now()
	alert := &models.AlertRecord{
		RuleID:         rule.ID,
		RuleName:       rule.Name,
		EventType:      event.EventType,
		Severity:       rule.Severity,
		Message:        event.Message,
		Context:        as.contextToJSON(event.Context),
		Status:         "triggered",
		AggregationKey: aggKey,
		Count:          count,
	}
	if count == 1 {
		alert.FirstTriggeredAt = &now
	}
	alert.LastTriggeredAt = &now
	if as.db != nil {
		if err := as.db.Create(alert).Error; err != nil {
			return err
		}
	}
	as.updateStatistics(rule.Severity, true, false, false)
	as.checkEscalation(alert, rule)
	if shouldSend {
		as.sendAlert(alert, rule)
	}
	as.addHistory(alert.ID, "triggered", "", "triggered", "", 0)
	return nil
}

func (as *AlertService) sendAlert(alert *models.AlertRecord, rule models.AlertRule) error {
	as.mu.RLock()
	channels := as.channels
	as.mu.RUnlock()
	var channelIDs []uint
	if err := json.Unmarshal([]byte(rule.ChannelIDs), &channelIDs); err != nil {
		return err
	}
	alertMsg := AlertMessage{
		Title:     alert.RuleName,
		Message:   alert.Message,
		Severity:  alert.Severity,
		EventID:   fmt.Sprintf("%d", alert.ID),
		Timestamp: alert.CreatedAt,
		Context:   as.jsonToContext(alert.Context),
	}
	for _, id := range channelIDs {
		if channel, ok := channels[id]; ok {
			go func(ch AlertChannel) {
				startTime := time.Now()
				ch.Send(alertMsg)
				duration := time.Since(startTime)
				RecordAlertResponseTime(alert.Severity, duration)
			}(channel)
		}
	}
	return nil
}

func (as *AlertService) buildAggregationKey(rule models.AlertRule, event AlertEvent) string {
	return fmt.Sprintf("%d-%s", rule.ID, event.EventType)
}

func (as *AlertService) contextToJSON(ctx map[string]interface{}) string {
	if ctx == nil {
		return "{}"
	}
	data, _ := json.Marshal(ctx)
	return string(data)
}

func (as *AlertService) jsonToContext(data string) map[string]interface{} {
	var ctx map[string]interface{}
	_ = json.Unmarshal([]byte(data), &ctx)
	return ctx
}

func (as *AlertService) addHistory(alertID uint, action, oldStatus, newStatus, note string, performedBy uint) {
	if as.db == nil {
		return
	}
	history := &models.AlertHistory{
		AlertID:     alertID,
		Action:      action,
		OldStatus:   oldStatus,
		NewStatus:   newStatus,
		Note:        note,
		PerformedBy: performedBy,
	}
	as.db.Create(history)
}

func (as *AlertService) ShouldTriggerAlert(ruleID uint, aggKey string, windowSecs, threshold int, severity, message string) (bool, int) {
	return as.aggregator.ShouldTriggerAlert(ruleID, aggKey, windowSecs, threshold, severity, message)
}

func (aa *AlertAggregator) ShouldTriggerAlert(ruleID uint, aggKey string, windowSecs, threshold int, severity, message string) (bool, int) {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	now := time.Now()
	key := fmt.Sprintf("%s", aggKey)
	item, exists := aa.AlertCounts[key]
	if !exists {
		item = &AlertCountItem{
			RuleID:         ruleID,
			AggregationKey: aggKey,
			Count:          1,
			FirstSeen:      now,
			LastSeen:       now,
			Severity:       severity,
			Messages:       []string{message},
		}
		aa.AlertCounts[key] = item
		aa.updateSummary(key, ruleID, aggKey, severity, message)
		return true, 1
	}
	window := time.Duration(windowSecs) * time.Second
	if now.Sub(item.FirstSeen) > window {
		item.Count = 1
		item.FirstSeen = now
		item.LastSeen = now
		item.Severity = severity
		item.Messages = []string{message}
		aa.updateSummary(key, ruleID, aggKey, severity, message)
		return true, 1
	}
	item.Count++
	item.LastSeen = now
	if !aa.containsMessage(item.Messages, message) && len(item.Messages) < 10 {
		item.Messages = append(item.Messages, message)
	}
	aa.updateSummary(key, ruleID, aggKey, severity, message)
	if item.Count == threshold || threshold == 1 {
		return true, item.Count
	}
	if threshold > 1 && item.Count%threshold == 0 {
		return true, item.Count
	}
	return false, item.Count
}

func (aa *AlertAggregator) containsMessage(messages []string, message string) bool {
	for _, msg := range messages {
		if msg == message {
			return true
		}
	}
	return false
}

func (aa *AlertAggregator) updateSummary(key string, ruleID uint, aggKey, severity, message string) {
	summary, exists := aa.AlertSummaries[key]
	now := time.Now()
	if !exists {
		summary = &AlertSummary{
			RuleID:         ruleID,
			AggregationKey: aggKey,
			TotalCount:     0,
			CriticalCount:  0,
			WarningCount:   0,
			InfoCount:      0,
			FirstSeen:      now,
			LastSeen:       now,
			UniqueMessages: make(map[string]int),
		}
		aa.AlertSummaries[key] = summary
	}
	summary.TotalCount++
	summary.LastSeen = now
	switch severity {
	case "critical":
		summary.CriticalCount++
	case "warning":
		summary.WarningCount++
	case "info":
		summary.InfoCount++
	}
	summary.UniqueMessages[message]++
}

func (aa *AlertAggregator) Cleanup(oldThreshold time.Duration) {
	aa.mu.Lock()
	defer aa.mu.Unlock()
	now := time.Now()
	for key, item := range aa.AlertCounts {
		if now.Sub(item.LastSeen) > oldThreshold {
			delete(aa.AlertCounts, key)
		}
	}
}

func (as *AlertService) checkEscalation(alert *models.AlertRecord, rule models.AlertRule) {
	if as.escalationManager == nil {
		return
	}
	escalationKey := fmt.Sprintf("%d-%d", alert.ID, rule.ID)
	as.escalationManager.mu.Lock()
	defer as.escalationManager.mu.Unlock()

	state, exists := as.escalationManager.escalations[escalationKey]
	if !exists {
		as.escalationManager.escalations[escalationKey] = &EscalationState{
			Level:       1,
			LastLevelUp: time.Now(),
			NextLevelAt: time.Now().Add(as.escalationManager.getTimeout(rule.Severity)),
			AlertID:     alert.ID,
			RuleID:      rule.ID,
		}
		return
	}

	if time.Now().After(state.NextLevelAt) {
		state.Level++
		state.LastLevelUp = time.Now()
		state.NextLevelAt = time.Now().Add(as.escalationManager.getTimeout(rule.Severity) * time.Duration(state.Level))
		as.stats.mu.Lock()
		as.stats.TotalEscalated++
		as.stats.mu.Unlock()
		as.performEscalationActions(alert, rule, state.Level)
	}
}

func (em *EscalationManager) getTimeout(severity string) time.Duration {
	if timeout, ok := em.timeouts[severity]; ok {
		return timeout
	}
	return 15 * time.Minute
}

func (as *AlertService) performEscalationActions(alert *models.AlertRecord, rule models.AlertRule, level int) {
	go func() {
		escalationMsg := fmt.Sprintf("[ESCALATION L%d] %s", level, alert.Message)
		escalatedAlert := &models.AlertRecord{
			RuleID:         rule.ID,
			RuleName:       rule.Name,
			EventType:      alert.EventType,
			Severity:       as.getEscalatedSeverity(alert.Severity, level),
			Message:        escalationMsg,
			Context:        alert.Context,
			Status:         "triggered",
			AggregationKey: fmt.Sprintf("%s-escalated-%d", alert.AggregationKey, level),
			Count:          alert.Count,
			FirstTriggeredAt: alert.FirstTriggeredAt,
			LastTriggeredAt:  alert.LastTriggeredAt,
		}
		as.sendAlert(escalatedAlert, rule)
	}()
}

func (as *AlertService) getEscalatedSeverity(currentSeverity string, level int) string {
	switch currentSeverity {
	case "info":
		if level >= 3 {
			return "warning"
		}
		return "info"
	case "warning":
		if level >= 2 {
			return "critical"
		}
		return "warning"
	case "critical":
		return "critical"
	default:
		return currentSeverity
	}
}

func (as *AlertService) updateStatistics(severity string, triggered, resolved, acknowledged bool) {
	as.stats.mu.Lock()
	defer as.stats.mu.Unlock()

	if triggered {
		as.stats.TotalTriggered++
		switch severity {
		case "critical":
			as.stats.CriticalCount++
		case "warning":
			as.stats.WarningCount++
		case "info":
			as.stats.InfoCount++
		}
	}
	if resolved {
		as.stats.TotalResolved++
	}
	if acknowledged {
		as.stats.TotalAcknowledged++
	}
}

func (as *AlertService) GetStatistics() AlertStatistics {
	as.stats.mu.RLock()
	defer as.stats.mu.RUnlock()

	statsCopy := AlertStatistics{
		TotalTriggered:    as.stats.TotalTriggered,
		TotalResolved:     as.stats.TotalResolved,
		TotalEscalated:    as.stats.TotalEscalated,
		TotalAcknowledged: as.stats.TotalAcknowledged,
		CriticalCount:     as.stats.CriticalCount,
		WarningCount:      as.stats.WarningCount,
		InfoCount:         as.stats.InfoCount,
		AvgResponseTime:   as.stats.AvgResponseTime,
		LastResetAt:       as.stats.LastResetAt,
	}
	return statsCopy
}

func (as *AlertService) GetMetrics() AlertMetrics {
	stats := as.GetStatistics()
	return AlertMetrics{
		Total:         stats.TotalTriggered,
		Resolved:      stats.TotalResolved,
		Acknowledged:  stats.TotalAcknowledged,
		Escalated:     stats.TotalEscalated,
		Critical:      stats.CriticalCount,
		Warning:       stats.WarningCount,
		Info:          stats.InfoCount,
		AvgResponseMs: stats.AvgResponseTime,
	}
}

func (as *AlertService) GetPerformance() AlertPerformance {
	stats := as.GetStatistics()
	performance := AlertPerformance{
		TotalTriggers:  stats.TotalTriggered,
		LastCalculated: time.Now(),
	}

	if stats.TotalTriggered > 0 {
		resolvedAndEscalated := stats.TotalResolved + stats.TotalEscalated
		performance.Accuracy = float64(resolvedAndEscalated) / float64(stats.TotalTriggered) * 100
	}

	return performance
}

func (as *AlertService) AcknowledgeAlert(id uint, note string, performedBy uint) error {
	alert, err := as.GetAlert(id)
	if err != nil {
		return err
	}
	oldStatus := alert.Status
	alert.Status = "acknowledged"
	if as.db != nil {
		if err := as.db.Save(alert).Error; err != nil {
			return err
		}
	}
	as.updateStatistics(alert.Severity, false, false, true)
	as.addHistory(id, "acknowledged", oldStatus, "acknowledged", note, performedBy)
	return nil
}

func (as *AlertService) CreateRule(rule *models.AlertRule) error {
	if as.db == nil {
		return nil
	}
	if err := as.db.Create(rule).Error; err != nil {
		return err
	}
	as.LoadRules()
	return nil
}

func (as *AlertService) UpdateRule(rule *models.AlertRule) error {
	if as.db == nil {
		return nil
	}
	if err := as.db.Save(rule).Error; err != nil {
		return err
	}
	as.LoadRules()
	return nil
}

func (as *AlertService) DeleteRule(id uint) error {
	if as.db == nil {
		return nil
	}
	if err := as.db.Delete(&models.AlertRule{}, id).Error; err != nil {
		return err
	}
	as.LoadRules()
	return nil
}

func (as *AlertService) GetRule(id uint) (*models.AlertRule, error) {
	var rule models.AlertRule
	if as.db == nil {
		return nil, nil
	}
	if err := as.db.First(&rule, id).Error; err != nil {
		return nil, err
	}
	return &rule, nil
}

func (as *AlertService) ListRules() ([]models.AlertRule, error) {
	var rules []models.AlertRule
	if as.db == nil {
		return []models.AlertRule{}, nil
	}
	if err := as.db.Find(&rules).Error; err != nil {
		return nil, err
	}
	return rules, nil
}

func (as *AlertService) CreateChannel(channel *models.AlertChannel) error {
	if as.db == nil {
		return nil
	}
	if err := as.db.Create(channel).Error; err != nil {
		return err
	}
	as.LoadChannels()
	return nil
}

func (as *AlertService) UpdateChannel(channel *models.AlertChannel) error {
	if as.db == nil {
		return nil
	}
	if err := as.db.Save(channel).Error; err != nil {
		return err
	}
	as.LoadChannels()
	return nil
}

func (as *AlertService) DeleteChannel(id uint) error {
	if as.db == nil {
		return nil
	}
	if err := as.db.Delete(&models.AlertChannel{}, id).Error; err != nil {
		return err
	}
	as.LoadChannels()
	return nil
}

func (as *AlertService) GetChannel(id uint) (*models.AlertChannel, error) {
	var channel models.AlertChannel
	if as.db == nil {
		return nil, nil
	}
	if err := as.db.First(&channel, id).Error; err != nil {
		return nil, err
	}
	return &channel, nil
}

func (as *AlertService) ListChannels() ([]models.AlertChannel, error) {
	var channels []models.AlertChannel
	if as.db == nil {
		return []models.AlertChannel{}, nil
	}
	if err := as.db.Find(&channels).Error; err != nil {
		return nil, err
	}
	return channels, nil
}

func (as *AlertService) ListAlerts(page, pageSize int) ([]models.AlertRecord, int64, error) {
	var alerts []models.AlertRecord
	var total int64
	if as.db == nil {
		return []models.AlertRecord{}, 0, nil
	}
	if err := as.db.Model(&models.AlertRecord{}).Count(&total).Error; err != nil {
		return nil, 0, err
	}
	offset := (page - 1) * pageSize
	if err := as.db.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&alerts).Error; err != nil {
		return nil, 0, err
	}
	return alerts, total, nil
}

func (as *AlertService) GetAlert(id uint) (*models.AlertRecord, error) {
	var alert models.AlertRecord
	if as.db == nil {
		return nil, nil
	}
	if err := as.db.First(&alert, id).Error; err != nil {
		return nil, err
	}
	return &alert, nil
}

func (as *AlertService) ResolveAlert(id uint, note string, performedBy uint) error {
	alert, err := as.GetAlert(id)
	if err != nil {
		return err
	}
	oldStatus := alert.Status
	now := time.Now()
	alert.Status = "resolved"
	alert.ResolvedAt = &now
	if as.db != nil {
		if err := as.db.Save(alert).Error; err != nil {
			return err
		}
	}
	as.updateStatistics(alert.Severity, false, true, false)
	as.addHistory(id, "resolved", oldStatus, "resolved", note, performedBy)
	as.clearEscalation(id)
	return nil
}

func (as *AlertService) clearEscalation(alertID uint) {
	if as.escalationManager == nil {
		return
	}
	as.escalationManager.mu.Lock()
	defer as.escalationManager.mu.Unlock()

	for key, state := range as.escalationManager.escalations {
		if state.AlertID == alertID {
			delete(as.escalationManager.escalations, key)
			break
		}
	}
}

func (as *AlertService) GetAlertHistory(alertID uint) ([]models.AlertHistory, error) {
	var histories []models.AlertHistory
	if as.db == nil {
		return []models.AlertHistory{}, nil
	}
	if err := as.db.Where("alert_id = ?", alertID).Order("created_at DESC").Find(&histories).Error; err != nil {
		return nil, err
	}
	return histories, nil
}

func (as *AlertService) GetActiveAlerts() ([]models.AlertRecord, error) {
	var alerts []models.AlertRecord
	if as.db == nil {
		return []models.AlertRecord{}, nil
	}
	if err := as.db.Where("status NOT IN ?", []string{"resolved", "acknowledged"}).Order("created_at DESC").Find(&alerts).Error; err != nil {
		return nil, err
	}
	return alerts, nil
}

func (as *AlertService) GetAlertsBySeverity(severity string) ([]models.AlertRecord, error) {
	var alerts []models.AlertRecord
	if as.db == nil {
		return []models.AlertRecord{}, nil
	}
	if err := as.db.Where("severity = ? AND status NOT IN ?", severity, []string{"resolved"}).Order("created_at DESC").Find(&alerts).Error; err != nil {
		return nil, err
	}
	return alerts, nil
}

func (as *AlertService) GetAlertTrends(days int) (map[string]interface{}, error) {
	trends := make(map[string]interface{})
	if as.db == nil {
		return trends, nil
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	var dailyCounts []struct {
		Date  string
		Count int
	}

	if err := as.db.Model(&models.AlertRecord{}).
		Select("DATE(created_at) as date, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("DATE(created_at)").
		Order("date").
		Find(&dailyCounts).Error; err != nil {
		return nil, err
	}

	trends["daily_counts"] = dailyCounts

	var severityCounts []struct {
		Severity string
		Count    int
	}
	if err := as.db.Model(&models.AlertRecord{}).
		Select("severity, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("severity").
		Find(&severityCounts).Error; err != nil {
		return nil, err
	}
	trends["severity_counts"] = severityCounts

	var statusCounts []struct {
		Status string
		Count  int
	}
	if err := as.db.Model(&models.AlertRecord{}).
		Select("status, COUNT(*) as count").
		Where("created_at >= ?", startDate).
		Group("status").
		Find(&statusCounts).Error; err != nil {
		return nil, err
	}
	trends["status_counts"] = statusCounts

	return trends, nil
}

func RecordAlertResponseTime(alertType string, duration time.Duration) {
	bm := metrics.GetBusinessMetrics()
	bm.RecordAlertDeliveryLatency(duration)
}

func TrackAlertAccuracy(alertType string, isTruePositive bool) {
	bm := metrics.GetBusinessMetrics()
	bm.RecordAlertTrigger("info", alertType)
}
