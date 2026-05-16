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
	AlertCounts map[string]*AlertCountItem
	mu          sync.RWMutex
}

// AlertCountItem 告警计数项
type AlertCountItem struct {
	RuleID         uint
	AggregationKey string
	Count          int
	FirstSeen      time.Time
	LastSeen       time.Time
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
		AlertCounts: make(map[string]*AlertCountItem),
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
	shouldSend, count := as.aggregator.ShouldTriggerAlert(rule.ID, aggKey, rule.AggregationWindow, rule.Threshold)
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
func (aa *AlertAggregator) ShouldTriggerAlert(ruleID uint, aggKey string, windowSecs, threshold int) (bool, int) {
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
		}
		aa.AlertCounts[key] = item
		return true, 1
	}
	window := time.Duration(windowSecs) * time.Second
	if now.Sub(item.FirstSeen) > window {
		item.Count = 1
		item.FirstSeen = now
		item.LastSeen = now
		return true, 1
	}
	item.Count++
	item.LastSeen = now
	if item.Count == threshold || threshold == 1 {
		return true, item.Count
	}
	if threshold > 1 && item.Count%threshold == 0 {
		return true, item.Count
	}
	return false, item.Count
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
