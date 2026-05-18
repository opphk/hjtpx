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

// AlertService 告警服务
type AlertService struct {
	db         *gorm.DB
	channels   map[uint]AlertChannel
	rules      []models.AlertRule
	aggregator *AlertAggregator
	mu         sync.RWMutex
}

// AlertEvent 告警事件
type AlertEvent struct {
	EventType string                 `json:"event_type"`
	Message   string                 `json:"message"`
	Context   map[string]interface{} `json:"context"`
	Timestamp time.Time              `json:"timestamp"`
}

// AlertAggregator 告警聚合器
type AlertAggregator struct {
	AlertCounts     map[string]*AlertCountItem
	AlertSummaries  map[string]*AlertSummary
	mu              sync.RWMutex
	CleanupInterval time.Duration
}

// AlertCountItem 告警计数项
type AlertCountItem struct {
	RuleID         uint
	AggregationKey string
	Count          int
	FirstSeen      time.Time
	LastSeen       time.Time
	Severity       string
	Messages       []string
}

// AlertSummary 告警摘要
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

// AlertEscalation 告警升级
type AlertEscalation struct {
	RuleID       uint
	Level        int
	Conditions   string
	Action       string
	NotifyRoles  []uint
	CreatedAt    time.Time
}

// 保持向后兼容的别名
type alertCountItem = AlertCountItem

// NewAlertService 创建告警服务
func NewAlertService(db *gorm.DB) *AlertService {
	return &AlertService{
		db:         db,
		channels:   make(map[uint]AlertChannel),
		rules:      []models.AlertRule{},
		aggregator: NewAlertAggregator(),
	}
}

// NewAlertAggregator 创建告警聚合器
func NewAlertAggregator() *AlertAggregator {
	return &AlertAggregator{
		AlertCounts:     make(map[string]*AlertCountItem),
		AlertSummaries:  make(map[string]*AlertSummary),
		CleanupInterval: 5 * time.Minute,
	}
}

// LoadRules 加载告警规则
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

// LoadChannels 加载告警渠道
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

// ProcessEvent 处理事件
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
				ch.Send(alertMsg)
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

// ShouldTriggerAlert 判断是否应该触发告警
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

// Cleanup 清理过期的聚合数据
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

// CreateRule 创建告警规则
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

// UpdateRule 更新告警规则
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

// DeleteRule 删除告警规则
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

// GetRule 获取告警规则
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

// ListRules 列出所有告警规则
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

// CreateChannel 创建告警渠道
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

// UpdateChannel 更新告警渠道
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

// DeleteChannel 删除告警渠道
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

// GetChannel 获取告警渠道
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

// ListChannels 列出所有告警渠道
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

// ListAlerts 列出告警记录
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

// GetAlert 获取告警记录
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

// ResolveAlert 解决告警
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
	as.addHistory(id, "resolved", oldStatus, "resolved", note, performedBy)
	return nil
}

// GetAlertHistory 获取告警历史
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

// GetAlertStatistics 获取告警统计
func (as *AlertService) GetAlertStatistics(startTime, endTime time.Time) (map[string]interface{}, error) {
	var totalAlerts int64
	var criticalAlerts int64
	var warningAlerts int64
	var resolvedAlerts int64
	
	if as.db == nil {
		return map[string]interface{}{
			"total":       0,
			"critical":    0,
			"warning":     0,
			"resolved":    0,
			"resolution_rate": 0,
		}, nil
	}
	
	as.db.Model(&models.AlertRecord{}).
		Where("created_at BETWEEN ? AND ?", startTime, endTime).
		Count(&totalAlerts)
	
	as.db.Model(&models.AlertRecord{}).
		Where("created_at BETWEEN ? AND ? AND severity = ?", startTime, endTime, "critical").
		Count(&criticalAlerts)
	
	as.db.Model(&models.AlertRecord{}).
		Where("created_at BETWEEN ? AND ? AND severity = ?", startTime, endTime, "warning").
		Count(&warningAlerts)
	
	as.db.Model(&models.AlertRecord{}).
		Where("created_at BETWEEN ? AND ? AND status = ?", startTime, endTime, "resolved").
		Count(&resolvedAlerts)
	
	resolutionRate := 0.0
	if totalAlerts > 0 {
		resolutionRate = float64(resolvedAlerts) / float64(totalAlerts) * 100
	}
	
	return map[string]interface{}{
		"total":           totalAlerts,
		"critical":        criticalAlerts,
		"warning":         warningAlerts,
		"resolved":        resolvedAlerts,
		"resolution_rate": resolutionRate,
	}, nil
}

// GetAlertTrend 获取告警趋势
func (as *AlertService) GetAlertTrend(period string) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	if as.db == nil {
		return results, nil
	}
	
	groupBy := "DATE(created_at)"
	dateFormat := "%Y-%m-%d"
	
	switch period {
	case "hour":
		groupBy = "HOUR(created_at)"
		dateFormat = "%Y-%m-%d %H:00"
	case "day":
		groupBy = "DATE(created_at)"
		dateFormat = "%Y-%m-%d"
	case "week":
		groupBy = "YEARWEEK(created_at)"
		dateFormat = "%Y-W%v"
	case "month":
		groupBy = "DATE_FORMAT(created_at, '%Y-%m')"
		dateFormat = "%Y-%m"
	}
	
	rows, err := as.db.Raw(`
		SELECT 
			DATE_FORMAT(created_at, ?) as time,
			COUNT(*) as count,
			SUM(CASE WHEN severity = 'critical' THEN 1 ELSE 0 END) as critical,
			SUM(CASE WHEN severity = 'warning' THEN 1 ELSE 0 END) as warning,
			SUM(CASE WHEN status = 'resolved' THEN 1 ELSE 0 END) as resolved
		FROM alert_records
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY `+groupBy+`
		ORDER BY time ASC
	`, dateFormat).Rows()
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var timeStr string
		var count, critical, warning, resolved int64
		
		if err := rows.Scan(&timeStr, &count, &critical, &warning, &resolved); err != nil {
			continue
		}
		
		results = append(results, map[string]interface{}{
			"time":      timeStr,
			"count":     count,
			"critical":  critical,
			"warning":   warning,
			"resolved":  resolved,
		})
	}
	
	return results, nil
}

// GetTopAlertRules 获取触发最多的告警规则
func (as *AlertService) GetTopAlertRules(limit int) ([]map[string]interface{}, error) {
	var results []map[string]interface{}
	
	if as.db == nil {
		return results, nil
	}
	
	rows, err := as.db.Raw(`
		SELECT 
			rule_id,
			rule_name,
			COUNT(*) as count,
			MAX(severity) as max_severity,
			MAX(created_at) as last_triggered
		FROM alert_records
		WHERE created_at >= DATE_SUB(NOW(), INTERVAL 7 DAY)
		GROUP BY rule_id, rule_name
		ORDER BY count DESC
		LIMIT ?
	`, limit).Rows()
	
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	
	for rows.Next() {
		var ruleID uint
		var ruleName string
		var count int64
		var maxSeverity string
		var lastTriggered time.Time
		
		if err := rows.Scan(&ruleID, &ruleName, &count, &maxSeverity, &lastTriggered); err != nil {
			continue
		}
		
		results = append(results, map[string]interface{}{
			"rule_id":        ruleID,
			"rule_name":      ruleName,
			"count":          count,
			"max_severity":   maxSeverity,
			"last_triggered": lastTriggered,
		})
	}
	
	return results, nil
}

// CheckEscalation 检查是否需要告警升级
func (as *AlertService) CheckEscalation(alertID uint) error {
	var alert models.AlertRecord
	if as.db == nil {
		return nil
	}
	
	if err := as.db.First(&alert, alertID).Error; err != nil {
		return err
	}
	
	if alert.Status == "resolved" {
		return nil
	}
	
	var escalations []AlertEscalation
	if err := as.db.Where("rule_id = ?", alert.RuleID).Find(&escalations).Error; err != nil {
		return err
	}
	
	for _, esc := range escalations {
		if as.evaluateEscalationCondition(esc, alert) {
			as.executeEscalation(esc, alert)
		}
	}
	
	return nil
}

func (as *AlertService) evaluateEscalationCondition(esc AlertEscalation, alert models.AlertRecord) bool {
	if esc.Conditions == "" {
		return false
	}
	
	conditions := strings.Split(esc.Conditions, ";")
	for _, cond := range conditions {
		parts := strings.SplitN(strings.TrimSpace(cond), "=", 2)
		if len(parts) != 2 {
			continue
		}
		
		key := strings.TrimSpace(parts[0])
		value := strings.TrimSpace(parts[1])
		
		switch key {
		case "duration":
			duration, err := time.ParseDuration(value)
			if err != nil {
				continue
			}
			if alert.CreatedAt.Add(duration).Before(time.Now()) {
				return true
			}
		case "count":
			var count int64
			as.db.Model(&models.AlertRecord{}).
				Where("rule_id = ? AND status != ?", alert.RuleID, "resolved").
				Count(&count)
			
			expectedCount := int64(0)
			fmt.Sscanf(value, "%d", &expectedCount)
			
			if int64(count) >= expectedCount {
				return true
			}
		}
	}
	
	return false
}

func (as *AlertService) executeEscalation(esc AlertEscalation, alert models.AlertRecord) error {
	as.mu.RLock()
	channels := as.channels
	as.mu.RUnlock()
	
	msg := AlertMessage{
		Title:     fmt.Sprintf("[升级 L%d] %s", esc.Level, alert.RuleName),
		Message:   fmt.Sprintf("告警已升级到Level %d，需要人工介入处理", esc.Level),
		Severity:  "critical",
		EventID:   fmt.Sprintf("%d", alert.ID),
		Timestamp: time.Now(),
		Context: map[string]interface{}{
			"original_severity": alert.Severity,
			"escalation_level":  esc.Level,
			"alert_id":          alert.ID,
		},
	}
	
	for _, channelID := range esc.NotifyRoles {
		if channel, ok := channels[uint(channelID)]; ok {
			go func(ch AlertChannel) {
				ch.Send(msg)
			}(channel)
		}
	}
	
	return nil
}

// BatchResolveAlerts 批量解决告警
func (as *AlertService) BatchResolveAlerts(ids []uint, note string, performedBy uint) error {
	if as.db == nil {
		return nil
	}
	
	now := time.Now()
	for _, id := range ids {
		if err := as.db.Model(&models.AlertRecord{}).
			Where("id = ?", id).
			Updates(map[string]interface{}{
				"status":      "resolved",
				"resolved_at": now,
			}).Error; err != nil {
			continue
		}
		
		as.addHistory(id, "batch_resolved", "triggered", "resolved", note, performedBy)
	}
	
	return nil
}
