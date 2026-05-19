package service

import (
	"encoding/json"
	"fmt"
	"strings"
	"sync"
	"time"

	"gorm.io/gorm"

	"github.com/hjtpx/hjtpx/pkg/models"
)

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

type AlertAggregator struct {
	AlertCounts     map[string]*AlertCountItem
	AlertSummaries  map[string]*AlertSummary
	mu              sync.RWMutex
	CleanupInterval time.Duration
}

type AlertEvent struct {
	EventType string                 `json:"event_type"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
}

type AlertEscalation struct {
	RuleID       uint
	Level        int
	Conditions   string
	Action       string
	NotifyRoles  []uint
	CreatedAt    time.Time
}

type AlertService struct {
	db         *gorm.DB
	channels   map[uint]AlertChannel
	rules      []models.AlertRule
	aggregator *AlertAggregator
	mu         sync.RWMutex
}

func NewAlertAggregator() *AlertAggregator {
	return &AlertAggregator{
		AlertCounts:     make(map[string]*AlertCountItem),
		AlertSummaries:  make(map[string]*AlertSummary),
		CleanupInterval: 5 * time.Minute,
	}
}

func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{
		db:         db,
		channels:   make(map[uint]AlertChannel),
		rules:      []models.AlertRule{},
		aggregator: NewAlertAggregator(),
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
		channel, err := CreateChannel(ChannelType(ch.Type), config)
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
	
	condition = strings.TrimSpace(condition)
	
	if strings.Contains(condition, "&&") {
		parts := strings.Split(condition, "&&")
		for _, part := range parts {
			if !as.parseSingleCondition(strings.TrimSpace(part), context) {
				return false
			}
		}
		return true
	}
	
	if strings.Contains(condition, "||") {
		parts := strings.Split(condition, "||")
		for _, part := range parts {
			if as.parseSingleCondition(strings.TrimSpace(part), context) {
				return true
			}
		}
		return false
	}
	
	return as.parseSingleCondition(condition, context)
}

func (as *AlertService) parseSingleCondition(condition string, context map[string]interface{}) bool {
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
	} else if strings.Contains(condition, ">") {
		parts := strings.SplitN(condition, ">", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if ctxVal, ok := context[key]; ok {
			numVal := as.parseNumber(ctxVal)
			compareVal := as.parseNumberString(value)
			return numVal > compareVal
		}
	} else if strings.Contains(condition, "<") {
		parts := strings.SplitN(condition, "<", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if ctxVal, ok := context[key]; ok {
			numVal := as.parseNumber(ctxVal)
			compareVal := as.parseNumberString(value)
			return numVal < compareVal
		}
	} else if strings.Contains(condition, ">=") {
		parts := strings.SplitN(condition, ">=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if ctxVal, ok := context[key]; ok {
			numVal := as.parseNumber(ctxVal)
			compareVal := as.parseNumberString(value)
			return numVal >= compareVal
		}
	} else if strings.Contains(condition, "<=") {
		parts := strings.SplitN(condition, "<=", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		if ctxVal, ok := context[key]; ok {
			numVal := as.parseNumber(ctxVal)
			compareVal := as.parseNumberString(value)
			return numVal <= compareVal
		}
	} else if strings.Contains(condition, "contains") {
		parts := strings.SplitN(condition, "contains", 2)
		key := strings.TrimSpace(parts[0])
		value := strings.Trim(strings.TrimSpace(parts[1]), "\"'")
		if ctxVal, ok := context[key]; ok {
			return strings.Contains(fmt.Sprintf("%v", ctxVal), value)
		}
	}
	return true
}

func (as *AlertService) parseNumber(val interface{}) float64 {
	switch v := val.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case string:
		return as.parseNumberString(v)
	default:
		return 0
	}
}

func (as *AlertService) parseNumberString(s string) float64 {
	var num float64
	fmt.Sscanf(s, "%f", &num)
	return num
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
	if shouldSend {
		as.sendAlert(alert, rule)
	}
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
				ch.Send(alertMsg)
			}(channel)
		}
	}
	return nil
}

func (as *AlertService) buildAggregationKey(rule models.AlertRule, event AlertEvent) string {
	return fmt.Sprintf("%d-%s", rule.ID, event.EventType)
}
