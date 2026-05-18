package service

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type StrategyVersion struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Version     string    `json:"version" gorm:"size:50;uniqueIndex"`
	StrategyType string   `json:"strategy_type" gorm:"size:50"`
	Description string    `json:"description" gorm:"type:text"`
	Rules       string    `json:"rules" gorm:"type:text"`
	IsActive    bool      `json:"is_active"`
	IsPublished bool      `json:"is_published"`
	PublishedAt *time.Time `json:"published_at"`
	CreatedBy   uint      `json:"created_by"`
	ApprovedBy  *uint     `json:"approved_by"`
	ApprovedAt  *time.Time `json:"approved_at"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

type StrategyRule struct {
	ID              uint      `json:"id" gorm:"primaryKey"`
	VersionID       uint      `json:"version_id" gorm:"index"`
	Name            string    `json:"name" gorm:"size:100"`
	RuleType        string    `json:"rule_type" gorm:"size:50"`
	Condition       string    `json:"condition" gorm:"type:text"`
	Action          string    `json:"action" gorm:"size:50"`
	Parameters      string    `json:"parameters" gorm:"type:text"`
	Priority        int       `json:"priority"`
	Weight          float64   `json:"weight"`
	Enabled         bool      `json:"enabled"`
	RiskThreshold   float64   `json:"risk_threshold"`
	TimeWindow      int       `json:"time_window"`
	MaxViolations   int       `json:"max_violations"`
	Cooldown        int       `json:"cooldown"`
	CreatedAt       time.Time `json:"created_at"`
	UpdatedAt       time.Time `json:"updated_at"`
}

type StrategyUpdate struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	VersionID     uint      `json:"version_id" gorm:"index"`
	UpdateType    string    `json:"update_type" gorm:"size:50"`
	EntityType    string    `json:"entity_type" gorm:"size:50"`
	EntityID      uint      `json:"entity_id"`
	OldValue      string    `json:"old_value" gorm:"type:text"`
	NewValue      string    `json:"new_value" gorm:"type:text"`
	Status        string    `json:"status" gorm:"size:20"`
	ErrorMessage  string    `json:"error_message" gorm:"type:text"`
	AppliedAt     *time.Time `json:"applied_at"`
	RollbackAt    *time.Time `json:"rollback_at"`
	CreatedAt     time.Time `json:"created_at"`
}

type StrategyAuditLog struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	VersionID   uint      `json:"version_id" gorm:"index"`
	Action      string    `json:"action" gorm:"size:50"`
	UserID      uint      `json:"user_id"`
	Changes     string    `json:"changes" gorm:"type:text"`
	IPAddress   string    `json:"ip_address" gorm:"size:50"`
	UserAgent   string    `json:"user_agent" gorm:"size:500"`
	CreatedAt   time.Time `json:"created_at"`
}

type HotUpdateService struct {
	mu           sync.RWMutex
	currentVersion *StrategyVersion
	rules        map[string]*StrategyRule
	updateChan   chan *StrategyUpdate
	subscribers  []chan *StrategyUpdate
	subMu        sync.RWMutex
}

var hotUpdateInstance *HotUpdateService
var hotUpdateOnce sync.Once

func NewHotUpdateService() *HotUpdateService {
	hotUpdateOnce.Do(func() {
		hotUpdateInstance = &HotUpdateService{
			rules:       make(map[string]*StrategyRule),
			updateChan:  make(chan *StrategyUpdate, 1000),
			subscribers: make([]chan *StrategyUpdate, 0),
		}
		hotUpdateInstance.loadCurrentVersion()
		go hotUpdateInstance.processUpdates()
	})
	return hotUpdateInstance
}

func (s *HotUpdateService) loadCurrentVersion() {
	var version StrategyVersion
	err := database.DB.Where("is_active = ?", true).Order("created_at DESC").First(&version).Error
	if err == nil {
		s.mu.Lock()
		s.currentVersion = &version
		s.mu.Unlock()
		s.loadRules(version.ID)
	} else {
		s.initializeDefaultVersion()
	}
}

func (s *HotUpdateService) initializeDefaultVersion() {
	version := &StrategyVersion{
		Version:      "v1.0.0",
		StrategyType: "default",
		Description: "默认风控策略版本",
		IsActive:    true,
		IsPublished: true,
	}

	now := time.Now()
	version.PublishedAt = &now

	if err := database.DB.Create(version).Error; err == nil {
		s.mu.Lock()
		s.currentVersion = version
		s.mu.Unlock()

		s.createDefaultRules(version.ID)
		s.loadRules(version.ID)
	}
}

func (s *HotUpdateService) createDefaultRules(versionID uint) {
	defaultRules := []StrategyRule{
		{
			VersionID:     versionID,
			Name:         "IP频率限制",
			RuleType:     "rate_limit",
			Condition:    "ip_request_count > threshold",
			Action:       "captcha",
			Parameters:   `{"threshold": 100, "window": 60}`,
			Priority:     100,
			Weight:       0.15,
			Enabled:      true,
			RiskThreshold: 60,
			TimeWindow:   60,
			MaxViolations: 5,
			Cooldown:    300,
		},
		{
			VersionID:     versionID,
			Name:         "异常速度检测",
			RuleType:     "behavior",
			Condition:    "mouse_speed > 2000 OR path_efficiency > 0.95",
			Action:       "block",
			Parameters:   `{"speed_threshold": 2000, "efficiency_threshold": 0.95}`,
			Priority:     90,
			Weight:       0.25,
			Enabled:      true,
			RiskThreshold: 40,
			TimeWindow:   0,
			MaxViolations: 1,
			Cooldown:    600,
		},
		{
			VersionID:     versionID,
			Name:         "设备指纹重复",
			RuleType:     "device_fingerprint",
			Condition:    "fingerprint_occurrences > 5",
			Action:       "review",
			Parameters:   `{"threshold": 5}`,
			Priority:     80,
			Weight:       0.20,
			Enabled:      true,
			RiskThreshold: 50,
			TimeWindow:   3600,
			MaxViolations: 3,
			Cooldown:    1800,
		},
		{
			VersionID:     versionID,
			Name:         "VPN/代理检测",
			RuleType:     "network",
			Condition:    "is_vpn == true OR is_proxy == true OR is_tor == true",
			Action:       "challenge",
			Parameters:   `{"require_additional_verify": true}`,
			Priority:     70,
			Weight:       0.20,
			Enabled:      true,
			RiskThreshold: 30,
			TimeWindow:   0,
			MaxViolations: 1,
			Cooldown:    3600,
		},
		{
			VersionID:     versionID,
			Name:         "地理位置异常",
			RuleType:     "geo",
			Condition:    "impossible_travel == true OR country_changes > 2",
			Action:       "review",
			Parameters:   `{"travel_speed_threshold": 800, "country_change_threshold": 2}`,
			Priority:     60,
			Weight:       0.10,
			Enabled:      true,
			RiskThreshold: 40,
			TimeWindow:   86400,
			MaxViolations: 2,
			Cooldown:    3600,
		},
		{
			VersionID:     versionID,
			Name:         "黑名单拦截",
			RuleType:     "blacklist",
			Condition:    "ip_in_blacklist == true OR device_in_blacklist == true",
			Action:       "block",
			Parameters:   `{}`,
			Priority:     1000,
			Weight:       1.0,
			Enabled:      true,
			RiskThreshold: 0,
			TimeWindow:   0,
			MaxViolations: 1,
			Cooldown:    0,
		},
	}

	for _, rule := range defaultRules {
		database.DB.Create(&rule)
	}
}

func (s *HotUpdateService) loadRules(versionID uint) {
	var rules []StrategyRule
	if err := database.DB.Where("version_id = ? AND enabled = ?", versionID, true).Order("priority DESC").Find(&rules).Error; err == nil {
		s.mu.Lock()
		s.rules = make(map[string]*StrategyRule)
		for i := range rules {
			s.rules[rules[i].Name] = &rules[i]
		}
		s.mu.Unlock()

		s.cacheRules()
	}
}

func (s *HotUpdateService) cacheRules() {
	ctx := context.Background()

	s.mu.RLock()
	defer s.mu.RUnlock()

	for name, rule := range s.rules {
		key := fmt.Sprintf("strategy:rule:%s", name)
		data, _ := json.Marshal(rule)
		redis.GetClient().Set(ctx, key, data, 24*time.Hour)
	}

	versionInfo := map[string]interface{}{
		"version_id": s.currentVersion.ID,
		"version":     s.currentVersion.Version,
		"rule_count":  len(s.rules),
		"updated_at":  time.Now(),
	}
	versionData, _ := json.Marshal(versionInfo)
	redis.GetClient().Set(ctx, "strategy:current_version", versionData, 24*time.Hour)
}

func (s *HotUpdateService) GetCurrentVersion() *StrategyVersion {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.currentVersion
}

func (s *HotUpdateService) GetAllRules() []*StrategyRule {
	s.mu.RLock()
	defer s.mu.RUnlock()

	rules := make([]*StrategyRule, 0, len(s.rules))
	for _, rule := range s.rules {
		rules = append(rules, rule)
	}
	return rules
}

func (s *HotUpdateService) GetRule(name string) (*StrategyRule, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if rule, exists := s.rules[name]; exists {
		return rule, nil
	}
	return nil, fmt.Errorf("规则不存在: %s", name)
}

func (s *HotUpdateService) CreateNewVersion(baseVersion string, newVersion string, description string, userID uint) (*StrategyVersion, error) {
	var existing StrategyVersion
	if err := database.DB.Where("version = ?", newVersion).First(&existing).Error; err == nil {
		return nil, fmt.Errorf("版本已存在: %s", newVersion)
	}

	newVer := &StrategyVersion{
		Version:      newVersion,
		StrategyType: "custom",
		Description:  description,
		IsActive:     false,
		IsPublished:  false,
		CreatedBy:    userID,
	}

	if err := database.DB.Create(newVer).Error; err != nil {
		return nil, err
	}

	var oldVersion StrategyVersion
	if err := database.DB.Where("version = ?", baseVersion).First(&oldVersion).Error; err != nil {
		return nil, fmt.Errorf("基础版本不存在: %s", baseVersion)
	}

	var oldRules []StrategyRule
	database.DB.Where("version_id = ?", oldVersion.ID).Find(&oldRules)

	for _, rule := range oldRules {
		newRule := rule
		newRule.ID = 0
		newRule.VersionID = newVer.ID
		database.DB.Create(&newRule)
	}

	s.logAudit(newVer.ID, "create_version", userID, fmt.Sprintf("基于 %s 创建新版本 %s", baseVersion, newVersion), "", "")

	return newVer, nil
}

func (s *HotUpdateService) UpdateRule(versionID uint, ruleID uint, updates map[string]interface{}) error {
	var rule StrategyRule
	if err := database.DB.First(&rule, ruleID).Error; err != nil {
		return err
	}

	if rule.VersionID != versionID {
		return fmt.Errorf("规则不属于指定版本")
	}

	oldValue, _ := json.Marshal(rule)

	update := &StrategyUpdate{
		VersionID:  versionID,
		UpdateType: "update",
		EntityType: "rule",
		EntityID:  ruleID,
		OldValue:  string(oldValue),
		NewValue:  "",
		Status:    "pending",
	}

	if name, ok := updates["name"].(string); ok {
		rule.Name = name
	}
	if ruleType, ok := updates["rule_type"].(string); ok {
		rule.RuleType = ruleType
	}
	if condition, ok := updates["condition"].(string); ok {
		rule.Condition = condition
	}
	if action, ok := updates["action"].(string); ok {
		rule.Action = action
	}
	if params, ok := updates["parameters"].(string); ok {
		rule.Parameters = params
	}
	if priority, ok := updates["priority"].(int); ok {
		rule.Priority = priority
	}
	if weight, ok := updates["weight"].(float64); ok {
		rule.Weight = weight
	}
	if enabled, ok := updates["enabled"].(bool); ok {
		rule.Enabled = enabled
	}
	if threshold, ok := updates["risk_threshold"].(float64); ok {
		rule.RiskThreshold = threshold
	}
	if window, ok := updates["time_window"].(int); ok {
		rule.TimeWindow = window
	}
	if maxViol, ok := updates["max_violations"].(int); ok {
		rule.MaxViolations = maxViol
	}
	if cooldown, ok := updates["cooldown"].(int); ok {
		rule.Cooldown = cooldown
	}

	rule.UpdatedAt = time.Now()

	newValue, _ := json.Marshal(rule)
	update.NewValue = string(newValue)

	database.DB.Save(&rule)
	database.DB.Create(update)

	s.mu.Lock()
	s.rules[rule.Name] = &rule
	s.mu.Unlock()

	s.cacheRules()

	s.notifySubscribers(update)

	return nil
}

func (s *HotUpdateService) AddRule(versionID uint, rule *StrategyRule) error {
	rule.VersionID = versionID

	if err := database.DB.Create(rule).Error; err != nil {
		return err
	}

	update := &StrategyUpdate{
		VersionID:  versionID,
		UpdateType: "add",
		EntityType: "rule",
		EntityID:  rule.ID,
		NewValue:  "",
		Status:    "applied",
	}
	newValue, _ := json.Marshal(rule)
	update.NewValue = string(newValue)
	database.DB.Create(update)

	s.mu.Lock()
	s.rules[rule.Name] = rule
	s.mu.Unlock()

	s.cacheRules()

	s.notifySubscribers(update)

	return nil
}

func (s *HotUpdateService) DeleteRule(versionID uint, ruleID uint) error {
	var rule StrategyRule
	if err := database.DB.First(&rule, ruleID).Error; err != nil {
		return err
	}

	if rule.VersionID != versionID {
		return fmt.Errorf("规则不属于指定版本")
	}

	oldValue, _ := json.Marshal(rule)

	update := &StrategyUpdate{
		VersionID:  versionID,
		UpdateType: "delete",
		EntityType: "rule",
		EntityID:  ruleID,
		OldValue:  string(oldValue),
		Status:    "applied",
	}

	database.DB.Delete(&rule)
	database.DB.Create(update)

	s.mu.Lock()
	delete(s.rules, rule.Name)
	s.mu.Unlock()

	s.cacheRules()

	s.notifySubscribers(update)

	return nil
}

func (s *HotUpdateService) PublishVersion(versionID uint, approvedBy uint) error {
	var version StrategyVersion
	if err := database.DB.First(&version, versionID).Error; err != nil {
		return err
	}

	if err := database.DB.Model(&StrategyVersion{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	now := time.Now()
	version.IsActive = true
	version.IsPublished = true
	version.PublishedAt = &now
	version.ApprovedBy = &approvedBy
	version.ApprovedAt = &now

	if err := database.DB.Save(&version).Error; err != nil {
		return err
	}

	s.mu.Lock()
	s.currentVersion = &version
	s.mu.Unlock()

	s.loadRules(version.ID)

	s.logAudit(version.ID, "publish", approvedBy, fmt.Sprintf("发布版本 %s", version.Version), "", "")

	return nil
}

func (s *HotUpdateService) RollbackVersion(targetVersionID uint, userID uint) error {
	var targetVersion StrategyVersion
	if err := database.DB.First(&targetVersion, targetVersionID).Error; err != nil {
		return err
	}

	if !targetVersion.IsPublished {
		return fmt.Errorf("只能回滚已发布的版本")
	}

	current := s.GetCurrentVersion()
	if current.ID == targetVersionID {
		return fmt.Errorf("当前已是该版本")
	}

	if err := database.DB.Model(&StrategyVersion{}).Where("is_active = ?", true).Update("is_active", false).Error; err != nil {
		return err
	}

	now := time.Now()
	targetVersion.IsActive = true
	targetVersion.UpdatedAt = now

	if err := database.DB.Save(&targetVersion).Error; err != nil {
		return err
	}

	s.mu.Lock()
	s.currentVersion = &targetVersion
	s.mu.Unlock()

	s.loadRules(targetVersion.ID)

	s.logAudit(targetVersion.ID, "rollback", userID, fmt.Sprintf("回滚到版本 %s", targetVersion.Version), "", "")

	return nil
}

func (s *HotUpdateService) GetVersionHistory(limit int) ([]StrategyVersion, error) {
	var versions []StrategyVersion
	err := database.DB.Order("created_at DESC").Limit(limit).Find(&versions).Error
	return versions, err
}

func (s *HotUpdateService) GetVersionUpdates(versionID uint, limit int) ([]StrategyUpdate, error) {
	var updates []StrategyUpdate
	err := database.DB.Where("version_id = ?", versionID).Order("created_at DESC").Limit(limit).Find(&updates).Error
	return updates, err
}

func (s *HotUpdateService) EvaluateRules(ctx context.Context, riskContext map[string]interface{}) (string, float64, []string) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var triggeredRules []string
	var totalRiskScore float64 = 0.0

	for _, rule := range s.rules {
		if !rule.Enabled {
			continue
		}

		triggered, score := s.evaluateRule(rule, riskContext)
		if triggered {
			triggeredRules = append(triggeredRules, rule.Name)
			totalRiskScore += score * rule.Weight
		}
	}

	action := s.determineAction(totalRiskScore)

	return action, totalRiskScore, triggeredRules
}

func (s *HotUpdateService) evaluateRule(rule *StrategyRule, riskContext map[string]interface{}) (bool, float64) {
	var params map[string]interface{}
	json.Unmarshal([]byte(rule.Parameters), &params)

	riskScore := 100.0

	switch rule.RuleType {
	case "rate_limit":
		if ipCount, ok := riskContext["ip_request_count"].(float64); ok {
			if threshold, ok := params["threshold"].(float64); ok {
				if ipCount > threshold {
					return true, mathMin(100, (ipCount/threshold)*rule.RiskThreshold)
				}
			}
		}

	case "behavior":
		if mouseSpeed, ok := riskContext["mouse_speed"].(float64); ok {
			if threshold, ok := params["speed_threshold"].(float64); ok {
				if mouseSpeed > threshold {
					riskScore -= (mouseSpeed - threshold) / threshold * 50
				}
			}
		}
		if pathEff, ok := riskContext["path_efficiency"].(float64); ok {
			if threshold, ok := params["efficiency_threshold"].(float64); ok {
				if pathEff > threshold {
					riskScore -= (pathEff - threshold) * 100
				}
			}
		}
		if riskScore < rule.RiskThreshold {
			return true, 100 - riskScore
		}

	case "network":
		if isVPN, _ := riskContext["is_vpn"].(bool); isVPN {
			riskScore -= 30
		}
		if isProxy, _ := riskContext["is_proxy"].(bool); isProxy {
			riskScore -= 25
		}
		if isTor, _ := riskContext["is_tor"].(bool); isTor {
			riskScore -= 40
		}
		if riskScore < rule.RiskThreshold {
			return true, 100 - riskScore
		}

	case "geo":
		if impossible, _ := riskContext["impossible_travel"].(bool); impossible {
			riskScore -= 50
		}
		if countryChanges, _ := riskContext["country_changes"].(int); countryChanges > 2 {
			riskScore -= 30
		}
		if riskScore < rule.RiskThreshold {
			return true, 100 - riskScore
		}

	case "blacklist":
		if ipBlocked, _ := riskContext["ip_in_blacklist"].(bool); ipBlocked {
			return true, 100
		}
		if deviceBlocked, _ := riskContext["device_in_blacklist"].(bool); deviceBlocked {
			return true, 100
		}
	}

	return false, 0
}

func (s *HotUpdateService) determineAction(riskScore float64) string {
	switch {
	case riskScore >= 80:
		return "allow"
	case riskScore >= 60:
		return "captcha"
	case riskScore >= 40:
		return "review"
	case riskScore >= 20:
		return "block"
	default:
		return "challenge"
	}
}

func (s *HotUpdateService) processUpdates() {
	for update := range s.updateChan {
		s.mu.RLock()
		currentVersion := s.currentVersion
		s.mu.RUnlock()

		if update.VersionID == currentVersion.ID {
			s.notifySubscribers(update)
		}
	}
}

func (s *HotUpdateService) Subscribe() chan *StrategyUpdate {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	ch := make(chan *StrategyUpdate, 100)
	s.subscribers = append(s.subscribers, ch)
	return ch
}

func (s *HotUpdateService) Unsubscribe(ch chan *StrategyUpdate) {
	s.subMu.Lock()
	defer s.subMu.Unlock()

	for i, subscriber := range s.subscribers {
		if subscriber == ch {
			s.subscribers = append(s.subscribers[:i], s.subscribers[i+1:]...)
			close(ch)
			break
		}
	}
}

func (s *HotUpdateService) notifySubscribers(update *StrategyUpdate) {
	s.subMu.RLock()
	defer s.subMu.RUnlock()

	for _, ch := range s.subscribers {
		select {
		case ch <- update:
		default:
		}
	}
}

func (s *HotUpdateService) logAudit(versionID uint, action string, userID uint, changes, ipAddress, userAgent string) {
	log := &StrategyAuditLog{
		VersionID: versionID,
		Action:    action,
		UserID:    userID,
		Changes:   changes,
		IPAddress: ipAddress,
		UserAgent: userAgent,
	}
	database.DB.Create(log)
}

func (s *HotUpdateService) GetAuditLogs(versionID uint, limit int) ([]StrategyAuditLog, error) {
	var logs []StrategyAuditLog
	err := database.DB.Where("version_id = ?", versionID).Order("created_at DESC").Limit(limit).Find(&logs).Error
	return logs, err
}

func (s *HotUpdateService) ValidateRule(rule *StrategyRule) error {
	if rule.Name == "" {
		return fmt.Errorf("规则名称不能为空")
	}
	if rule.RuleType == "" {
		return fmt.Errorf("规则类型不能为空")
	}
	if rule.Action == "" {
		return fmt.Errorf("规则动作不能为空")
	}
	if rule.Weight < 0 || rule.Weight > 1 {
		return fmt.Errorf("权重必须在0-1之间")
	}
	if rule.Priority < 0 {
		return fmt.Errorf("优先级不能为负数")
	}
	return nil
}

func (s *HotUpdateService) GetRuleStatistics(ruleName string) (map[string]interface{}, error) {
	stats := make(map[string]interface{})

	var totalTriggers int64
	database.DB.Model(&StrategyUpdate{}).Where("entity_type = ? AND entity_id = (SELECT id FROM strategy_rules WHERE name = ? LIMIT 1) AND update_type = ?", "rule", ruleName, "add").Count(&totalTriggers)
	stats["total_triggers"] = totalTriggers

	var versionStats []map[string]interface{}
	database.DB.Raw(`
		SELECT 
			sv.version,
			sv.is_active,
			COUNT(sr.id) as rule_count,
			MAX(sr.updated_at) as last_updated
		FROM strategy_versions sv
		LEFT JOIN strategy_rules sr ON sv.id = sr.version_id
		GROUP BY sv.id, sv.version, sv.is_active
		ORDER BY sv.created_at DESC
		LIMIT 10
	`).Scan(&versionStats)
	stats["version_stats"] = versionStats

	return stats, nil
}

func (s *HotUpdateService) ExportVersion(versionID uint) (string, error) {
	var version StrategyVersion
	if err := database.DB.First(&version, versionID).Error; err != nil {
		return "", err
	}

	var rules []StrategyRule
	database.DB.Where("version_id = ?", versionID).Find(&rules)

	exportData := map[string]interface{}{
		"version": version,
		"rules":   rules,
		"exported_at": time.Now(),
	}

	data, err := json.MarshalIndent(exportData, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *HotUpdateService) ImportVersion(jsonData string, userID uint) (*StrategyVersion, error) {
	var importData map[string]interface{}
	if err := json.Unmarshal([]byte(jsonData), &importData); err != nil {
		return nil, err
	}

	versionData, ok := importData["version"].(map[string]interface{})
	if !ok {
		return nil, fmt.Errorf("无效的版本数据")
	}

	version := &StrategyVersion{
		Version:      fmt.Sprintf("%v", versionData["version"]),
		StrategyType: fmt.Sprintf("%v", versionData["strategy_type"]),
		Description:  fmt.Sprintf("%v", versionData["description"]),
		IsActive:     false,
		IsPublished:  false,
		CreatedBy:    userID,
	}

	if err := database.DB.Create(version).Error; err != nil {
		return nil, err
	}

	if rulesData, ok := importData["rules"].([]interface{}); ok {
		for _, r := range rulesData {
			if ruleData, ok := r.(map[string]interface{}); ok {
				priority, _ := ruleData["priority"].(float64)
				weight, _ := ruleData["weight"].(float64)
				riskThreshold, _ := ruleData["risk_threshold"].(float64)
				timeWindow, _ := ruleData["time_window"].(float64)
				maxViolations, _ := ruleData["max_violations"].(float64)
				cooldown, _ := ruleData["cooldown"].(float64)
				
				rule := &StrategyRule{
					VersionID:     version.ID,
					Name:         fmt.Sprintf("%v", ruleData["name"]),
					RuleType:     fmt.Sprintf("%v", ruleData["rule_type"]),
					Condition:    fmt.Sprintf("%v", ruleData["condition"]),
					Action:       fmt.Sprintf("%v", ruleData["action"]),
					Parameters:   fmt.Sprintf("%v", ruleData["parameters"]),
					Priority:     int(priority),
					Weight:       weight,
					Enabled:      fmt.Sprintf("%v", ruleData["enabled"]) == "true",
					RiskThreshold: riskThreshold,
					TimeWindow:   int(timeWindow),
					MaxViolations: int(maxViolations),
					Cooldown:     int(cooldown),
				}
				database.DB.Create(rule)
			}
		}
	}

	return version, nil
}

func mathMin(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}
