package service

import (
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strconv"
	"strings"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

// RiskRuleService 风控规则服务
type RiskRuleService struct {
	db *gorm.DB
}

// NewRiskRuleService 创建新的风控规则服务
func NewRiskRuleService() *RiskRuleService {
	return &RiskRuleService{
		db: database.DB,
	}
}

// ==================== 规则模板管理 ====================

// GetRuleTemplates 获取规则模板列表
func (s *RiskRuleService) GetRuleTemplates(category string, onlyActive bool) ([]models.RiskRuleTemplate, error) {
	var templates []models.RiskRuleTemplate
	query := s.db.Model(&models.RiskRuleTemplate{})
	if category != "" {
		query = query.Where("category = ?", category)
	}
	if onlyActive {
		query = query.Where("is_active = ?", true)
	}
	err := query.Order("is_system DESC, created_at DESC").Find(&templates).Error
	return templates, err
}

// GetRuleTemplate 获取单个规则模板
func (s *RiskRuleService) GetRuleTemplate(templateID uint) (*models.RiskRuleTemplate, error) {
	var template models.RiskRuleTemplate
	err := s.db.First(&template, templateID).Error
	if err != nil {
		return nil, err
	}
	return &template, nil
}

// CreateRuleTemplate 创建规则模板
func (s *RiskRuleService) CreateRuleTemplate(template *models.RiskRuleTemplate, adminID uint) (*models.RiskRuleTemplate, error) {
	template.CreatedBy = adminID
	template.IsSystem = false
	if err := s.db.Create(template).Error; err != nil {
		return nil, err
	}
	return template, nil
}

// UpdateRuleTemplate 更新规则模板
func (s *RiskRuleService) UpdateRuleTemplate(templateID uint, template *models.RiskRuleTemplate) (*models.RiskRuleTemplate, error) {
	var existingTemplate models.RiskRuleTemplate
	if err := s.db.First(&existingTemplate, templateID).Error; err != nil {
		return nil, err
	}
	
	if existingTemplate.IsSystem {
		return nil, fmt.Errorf("system template cannot be modified")
	}
	
	if err := s.db.Model(&existingTemplate).Updates(template).Error; err != nil {
		return nil, err
	}
	return &existingTemplate, nil
}

// DeleteRuleTemplate 删除规则模板
func (s *RiskRuleService) DeleteRuleTemplate(templateID uint) error {
	var template models.RiskRuleTemplate
	if err := s.db.First(&template, templateID).Error; err != nil {
		return err
	}
	
	if template.IsSystem {
		return fmt.Errorf("system template cannot be deleted")
	}
	
	return s.db.Delete(&template).Error
}

// ApplyTemplate 应用模板创建规则
func (s *RiskRuleService) ApplyTemplate(templateID uint, ruleName string, adminID uint) (*models.RiskRule, error) {
	template, err := s.GetRuleTemplate(templateID)
	if err != nil {
		return nil, err
	}
	
	rule := &models.RiskRule{
		Name:          ruleName,
		Description:   template.Description,
		TemplateID:    &template.ID,
		RuleType:      template.RuleType,
		Condition:     template.Condition,
		Action:        template.Action,
		Params:        template.Params,
		Severity:      template.Severity,
		Priority:      100,
		IsEnabled:     true,
		CreatedBy:     adminID,
	}
	
	if err := s.db.Create(rule).Error; err != nil {
		return nil, err
	}
	
	// 记录审计日志
	newValue, _ := json.Marshal(rule)
	s.logRuleChange(rule.ID, rule.Name, "create", "", string(newValue), adminID, "从模板创建规则")
	
	return rule, nil
}

// ==================== 规则管理 ====================

// GetRiskRules 获取规则列表
func (s *RiskRuleService) GetRiskRules(ruleType string, status string, page, pageSize int) ([]models.RiskRule, int64, error) {
	var rules []models.RiskRule
	var total int64
	
	query := s.db.Model(&models.RiskRule{})
	if ruleType != "" {
		query = query.Where("rule_type = ?", ruleType)
	}
	if status != "" {
		if status == "enabled" {
			query = query.Where("is_enabled = ?", true)
		} else if status == "disabled" {
			query = query.Where("is_enabled = ?", false)
		}
	}
	
	query.Count(&total)
	err := query.Order("priority DESC, created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Preload("Template").
		Find(&rules).Error
	
	return rules, total, err
}

// GetRiskRule 获取单个规则
func (s *RiskRuleService) GetRiskRule(ruleID uint) (*models.RiskRule, error) {
	var rule models.RiskRule
	err := s.db.Preload("Template").First(&rule, ruleID).Error
	if err != nil {
		return nil, err
	}
	return &rule, nil
}

// CreateRiskRule 创建规则
func (s *RiskRuleService) CreateRiskRule(rule *models.RiskRule, adminID uint) (*models.RiskRule, error) {
	rule.CreatedBy = adminID
	if err := s.db.Create(rule).Error; err != nil {
		return nil, err
	}
	
	newValue, _ := json.Marshal(rule)
	s.logRuleChange(rule.ID, rule.Name, "create", "", string(newValue), adminID, "创建新规则")
	return rule, nil
}

// UpdateRiskRule 更新规则
func (s *RiskRuleService) UpdateRiskRule(ruleID uint, rule *models.RiskRule, adminID uint) (*models.RiskRule, error) {
	var existingRule models.RiskRule
	if err := s.db.First(&existingRule, ruleID).Error; err != nil {
		return nil, err
	}
	
	oldValue, _ := json.Marshal(existingRule)
	
	if err := s.db.Model(&existingRule).Updates(rule).Error; err != nil {
		return nil, err
	}
	
	newValue, _ := json.Marshal(existingRule)
	s.logRuleChange(existingRule.ID, existingRule.Name, "update", string(oldValue), string(newValue), adminID, "更新规则")
	
	return &existingRule, nil
}

// DeleteRiskRule 删除规则
func (s *RiskRuleService) DeleteRiskRule(ruleID uint, adminID uint) error {
	var rule models.RiskRule
	if err := s.db.First(&rule, ruleID).Error; err != nil {
		return err
	}
	
	oldValue, _ := json.Marshal(rule)
	s.logRuleChange(rule.ID, rule.Name, "delete", string(oldValue), "", adminID, "删除规则")
	
	return s.db.Delete(&rule).Error
}

// ToggleRule 切换规则启用状态
func (s *RiskRuleService) ToggleRule(ruleID uint, enabled bool, adminID uint) error {
	var rule models.RiskRule
	if err := s.db.First(&rule, ruleID).Error; err != nil {
		return err
	}
	
	action := "enable"
	if !enabled {
		action = "disable"
	}
	
	rule.IsEnabled = enabled
	if err := s.db.Save(&rule).Error; err != nil {
		return err
	}
	
	summary := "启用规则"
	if !enabled {
		summary = "禁用规则"
	}
	s.logRuleChange(rule.ID, rule.Name, action, "", "", adminID, summary)
	
	return nil
}

// ==================== 规则触发历史 ====================

// GetRuleTriggerHistories 获取规则触发历史
func (s *RiskRuleService) GetRuleTriggerHistories(ruleID uint, page, pageSize int) ([]models.RiskRuleTriggerHistory, int64, error) {
	var histories []models.RiskRuleTriggerHistory
	var total int64
	
	query := s.db.Model(&models.RiskRuleTriggerHistory{})
	if ruleID > 0 {
		query = query.Where("rule_id = ?", ruleID)
	}
	
	query.Count(&total)
	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Preload("Rule").
		Find(&histories).Error
	
	return histories, total, err
}

// RecordRuleTrigger 记录规则触发
func (s *RiskRuleService) RecordRuleTrigger(ruleID uint, sessionID string, appID *uint, userID *uint, ip string, inputData interface{}, triggerResult bool, actionTaken string, executionTime int64) (*models.RiskRuleTriggerHistory, error) {
	rule, err := s.GetRiskRule(ruleID)
	if err != nil {
		return nil, err
	}
	
	inputJSON, _ := json.Marshal(inputData)
	history := &models.RiskRuleTriggerHistory{
		RuleID:        ruleID,
		RuleName:      rule.Name,
		SessionID:     sessionID,
		ApplicationID: appID,
		UserID:        userID,
		IPAddress:     ip,
		InputData:     string(inputJSON),
		TriggerResult: triggerResult,
		ActionTaken:   actionTaken,
		ExecutionTime: executionTime,
	}
	
	if err := s.db.Create(history).Error; err != nil {
		return nil, err
	}
	
	// 更新性能统计
	s.updateRulePerformance(ruleID, executionTime, triggerResult)
	
	return history, nil
}

// ==================== 规则性能分析 ====================

// GetRulePerformance 获取规则性能数据
func (s *RiskRuleService) GetRulePerformance(ruleID uint, days int) ([]models.RiskRulePerformance, error) {
	var performances []models.RiskRulePerformance
	query := s.db.Model(&models.RiskRulePerformance{}).Where("rule_id = ?", ruleID)
	if days > 0 {
		sinceDate := time.Now().AddDate(0, 0, -days).Format("2006-01-02")
		query = query.Where("date >= ?", sinceDate)
	}
	err := query.Order("date DESC").Find(&performances).Error
	return performances, err
}

// GetAllRulesPerformance 获取所有规则性能概览
func (s *RiskRuleService) GetAllRulesPerformance(date string) ([]map[string]interface{}, error) {
	if date == "" {
		date = time.Now().Format("2006-01-02")
	}
	
	var result []map[string]interface{}
	query := `
		SELECT 
			r.id,
			r.name,
			r.rule_type,
			r.severity,
			r.is_enabled,
			COALESCE(p.trigger_count, 0) as trigger_count,
			COALESCE(p.hit_count, 0) as hit_count,
			COALESCE(p.avg_execution_time, 0) as avg_execution_time,
			COALESCE(p.error_count, 0) as error_count
		FROM risk_rules r
		LEFT JOIN risk_rule_performances p ON r.id = p.rule_id AND p.date = ?
		ORDER BY p.hit_count DESC
	`
	
	err := s.db.Raw(query, date).Scan(&result).Error
	return result, err
}

// updateRulePerformance 更新规则性能统计
func (s *RiskRuleService) updateRulePerformance(ruleID uint, executionTime int64, hit bool) {
	date := time.Now().Format("2006-01-02")
	var perf models.RiskRulePerformance
	
	err := s.db.Where("rule_id = ? AND date = ?", ruleID, date).First(&perf).Error
	if err == gorm.ErrRecordNotFound {
		perf = models.RiskRulePerformance{
			RuleID:           ruleID,
			Date:             date,
			TriggerCount:     1,
			AvgExecutionTime: float64(executionTime),
			MinExecutionTime: float64(executionTime),
			MaxExecutionTime: float64(executionTime),
		}
		if hit {
			perf.HitCount = 1
		}
		s.db.Create(&perf)
	} else if err == nil {
		perf.TriggerCount++
		if hit {
			perf.HitCount++
		}
		
		// 更新平均执行时间
		totalTime := perf.AvgExecutionTime * float64(perf.TriggerCount-1)
		perf.AvgExecutionTime = (totalTime + float64(executionTime)) / float64(perf.TriggerCount)
		
		// 更新极值
		if float64(executionTime) < perf.MinExecutionTime {
			perf.MinExecutionTime = float64(executionTime)
		}
		if float64(executionTime) > perf.MaxExecutionTime {
			perf.MaxExecutionTime = float64(executionTime)
		}
		
		s.db.Save(&perf)
	}
}

// ==================== 审计日志 ====================

// GetRuleAuditLogs 获取规则审计日志
func (s *RiskRuleService) GetRuleAuditLogs(ruleID uint, page, pageSize int) ([]models.RiskRuleAuditLog, int64, error) {
	var logs []models.RiskRuleAuditLog
	var total int64
	
	query := s.db.Model(&models.RiskRuleAuditLog{})
	if ruleID > 0 {
		query = query.Where("rule_id = ?", ruleID)
	}
	
	query.Count(&total)
	err := query.Order("created_at DESC").
		Offset((page - 1) * pageSize).
		Limit(pageSize).
		Preload("Rule").
		Find(&logs).Error
	
	return logs, total, err
}

// logRuleChange 记录规则变更
func (s *RiskRuleService) logRuleChange(ruleID uint, ruleName, action, oldValue, newValue string, adminID uint, summary string) {
	log := models.RiskRuleAuditLog{
		RuleID:        ruleID,
		RuleName:      ruleName,
		Action:        action,
		OldValue:      oldValue,
		NewValue:      newValue,
		ChangeSummary: summary,
		AdminID:       adminID,
	}
	s.db.Create(&log)
}

// ==================== 规则引擎执行 ====================

// RiskContext 规则评估上下文
type RiskContext struct {
	SessionID        string                 `json:"session_id"`
	IPAddress       string                 `json:"ip_address"`
	UserAgent       string                 `json:"user_agent"`
	Fingerprint     string                 `json:"fingerprint"`
	DeviceType      string                 `json:"device_type"`
	RequestCount    int                    `json:"request_count"`
	FailureCount    int                    `json:"failure_count"`
	SuccessCount    int                    `json:"success_count"`
	Timestamp       int64                  `json:"timestamp"`
	Hour            int                    `json:"hour"`
	IsBlacklisted  bool                   `json:"is_blacklisted"`
	IsWhitelisted   bool                   `json:"is_whitelisted"`
	IsVPN           bool                   `json:"is_vpn"`
	IsProxy         bool                   `json:"is_proxy"`
	IsTor           bool                   `json:"is_tor"`
	IsHosting       bool                   `json:"is_hosting"`
	IPReputation    string                 `json:"ip_reputation"`
	Country         string                 `json:"country"`
	ASN             int                    `json:"asn"`
	CustomData      map[string]interface{} `json:"custom_data"`
	SpeedMetrics    *SpeedMetrics          `json:"speed_metrics,omitempty"`
	TrajectoryData  *TrajectoryMetrics     `json:"trajectory_data,omitempty"`
	BehaviorData    *BehaviorMetrics       `json:"behavior_data,omitempty"`
}

// SpeedMetrics 速度相关指标
type SpeedMetrics struct {
	AvgSpeed         float64 `json:"avg_speed"`
	MaxSpeed         float64 `json:"max_speed"`
	MinSpeed         float64 `json:"min_speed"`
	SpeedVariance    float64 `json:"speed_variance"`
	SpeedConsistency float64 `json:"speed_consistency"`
}

// TrajectoryMetrics 轨迹相关指标
type TrajectoryMetrics struct {
	PathEfficiency    float64 `json:"path_efficiency"`
	CurvatureAvg      float64 `json:"curvature_avg"`
	CurvatureVariance float64 `json:"curvature_variance"`
	DirectionChanges  int     `json:"direction_changes"`
	MicroCorrections  int     `json:"micro_corrections"`
	BacktrackCount    int     `json:"backtrack_count"`
	Sinuosity         float64 `json:"sinuosity"`
}

// BehaviorMetrics 行为相关指标
type BehaviorMetrics struct {
	PauseCount         int     `json:"pause_count"`
	TotalPauseDuration  float64 `json:"total_pause_duration"`
	HesitationTime     float64 `json:"hesitation_time"`
	ClickRegularity     float64 `json:"click_regularity"`
	PositionEntropy     float64 `json:"position_entropy"`
	HumanLikenessScore float64 `json:"human_likeness_score"`
	AnomalyScore        float64 `json:"anomaly_score"`
}

// EvaluateRule 评估规则
func (s *RiskRuleService) EvaluateRule(ruleID uint, inputData map[string]interface{}) (bool, string, error) {
	rule, err := s.GetRiskRule(ruleID)
	if err != nil {
		return false, "", err
	}

	if !rule.IsEnabled {
		return false, "rule is disabled", nil
	}

	startTime := time.Now()

	result := s.evaluateConditionWithContext(rule.Condition, inputData)

	executionTime := time.Since(startTime).Milliseconds()

	actionTaken := ""
	if result {
		actionTaken = rule.Action
	}

	s.RecordRuleTrigger(ruleID, "", nil, nil, "", inputData, result, actionTaken, executionTime)

	return result, actionTaken, nil
}

// evaluateConditionWithContext 使用上下文评估条件
func (s *RiskRuleService) evaluateConditionWithContext(condition string, inputData map[string]interface{}) bool {
	if condition == "" {
		return false
	}

	ctx := s.buildRiskContext(inputData)
	return s.evaluateExpression(condition, ctx)
}

// buildRiskContext 从输入数据构建风险上下文
func (s *RiskRuleService) buildRiskContext(inputData map[string]interface{}) *RiskContext {
	ctx := &RiskContext{
		CustomData: make(map[string]interface{}),
	}

	if v, ok := inputData["session_id"]; ok {
		if str, ok := v.(string); ok {
			ctx.SessionID = str
		}
	}
	if v, ok := inputData["ip_address"]; ok {
		if str, ok := v.(string); ok {
			ctx.IPAddress = str
		}
	}
	if v, ok := inputData["user_agent"]; ok {
		if str, ok := v.(string); ok {
			ctx.UserAgent = str
		}
	}
	if v, ok := inputData["fingerprint"]; ok {
		if str, ok := v.(string); ok {
			ctx.Fingerprint = str
		}
	}
	if v, ok := inputData["device_type"]; ok {
		if str, ok := v.(string); ok {
			ctx.DeviceType = str
		}
	}
	if v, ok := inputData["request_count"]; ok {
		if num, ok := v.(float64); ok {
			ctx.RequestCount = int(num)
		}
	}
	if v, ok := inputData["failure_count"]; ok {
		if num, ok := v.(float64); ok {
			ctx.FailureCount = int(num)
		}
	}
	if v, ok := inputData["success_count"]; ok {
		if num, ok := v.(float64); ok {
			ctx.SuccessCount = int(num)
		}
	}
	if v, ok := inputData["timestamp"]; ok {
		if num, ok := v.(float64); ok {
			ctx.Timestamp = int64(num)
		}
	}
	if v, ok := inputData["hour"]; ok {
		if num, ok := v.(float64); ok {
			ctx.Hour = int(num)
		}
	}
	if v, ok := inputData["is_blacklisted"]; ok {
		if b, ok := v.(bool); ok {
			ctx.IsBlacklisted = b
		}
	}
	if v, ok := inputData["is_whitelisted"]; ok {
		if b, ok := v.(bool); ok {
			ctx.IsWhitelisted = b
		}
	}
	if v, ok := inputData["is_vpn"]; ok {
		if b, ok := v.(bool); ok {
			ctx.IsVPN = b
		}
	}
	if v, ok := inputData["is_proxy"]; ok {
		if b, ok := v.(bool); ok {
			ctx.IsProxy = b
		}
	}
	if v, ok := inputData["is_tor"]; ok {
		if b, ok := v.(bool); ok {
			ctx.IsTor = b
		}
	}
	if v, ok := inputData["is_hosting"]; ok {
		if b, ok := v.(bool); ok {
			ctx.IsHosting = b
		}
	}
	if v, ok := inputData["ip_reputation"]; ok {
		if str, ok := v.(string); ok {
			ctx.IPReputation = str
		}
	}
	if v, ok := inputData["country"]; ok {
		if str, ok := v.(string); ok {
			ctx.Country = str
		}
	}
	if v, ok := inputData["asn"]; ok {
		if num, ok := v.(float64); ok {
			ctx.ASN = int(num)
		}
	}

	if speedData, ok := inputData["speed_metrics"].(map[string]interface{}); ok {
		ctx.SpeedMetrics = &SpeedMetrics{}
		if v, ok := speedData["avg_speed"].(float64); ok {
			ctx.SpeedMetrics.AvgSpeed = v
		}
		if v, ok := speedData["max_speed"].(float64); ok {
			ctx.SpeedMetrics.MaxSpeed = v
		}
		if v, ok := speedData["min_speed"].(float64); ok {
			ctx.SpeedMetrics.MinSpeed = v
		}
		if v, ok := speedData["speed_variance"].(float64); ok {
			ctx.SpeedMetrics.SpeedVariance = v
		}
		if v, ok := speedData["speed_consistency"].(float64); ok {
			ctx.SpeedMetrics.SpeedConsistency = v
		}
	}

	if trajData, ok := inputData["trajectory_data"].(map[string]interface{}); ok {
		ctx.TrajectoryData = &TrajectoryMetrics{}
		if v, ok := trajData["path_efficiency"].(float64); ok {
			ctx.TrajectoryData.PathEfficiency = v
		}
		if v, ok := trajData["curvature_avg"].(float64); ok {
			ctx.TrajectoryData.CurvatureAvg = v
		}
		if v, ok := trajData["curvature_variance"].(float64); ok {
			ctx.TrajectoryData.CurvatureVariance = v
		}
		if v, ok := trajData["direction_changes"].(float64); ok {
			ctx.TrajectoryData.DirectionChanges = int(v)
		}
		if v, ok := trajData["micro_corrections"].(float64); ok {
			ctx.TrajectoryData.MicroCorrections = int(v)
		}
		if v, ok := trajData["backtrack_count"].(float64); ok {
			ctx.TrajectoryData.BacktrackCount = int(v)
		}
		if v, ok := trajData["sinuosity"].(float64); ok {
			ctx.TrajectoryData.Sinuosity = v
		}
	}

	if behData, ok := inputData["behavior_data"].(map[string]interface{}); ok {
		ctx.BehaviorData = &BehaviorMetrics{}
		if v, ok := behData["pause_count"].(float64); ok {
			ctx.BehaviorData.PauseCount = int(v)
		}
		if v, ok := behData["total_pause_duration"].(float64); ok {
			ctx.BehaviorData.TotalPauseDuration = v
		}
		if v, ok := behData["hesitation_time"].(float64); ok {
			ctx.BehaviorData.HesitationTime = v
		}
		if v, ok := behData["click_regularity"].(float64); ok {
			ctx.BehaviorData.ClickRegularity = v
		}
		if v, ok := behData["position_entropy"].(float64); ok {
			ctx.BehaviorData.PositionEntropy = v
		}
		if v, ok := behData["human_likeness_score"].(float64); ok {
			ctx.BehaviorData.HumanLikenessScore = v
		}
		if v, ok := behData["anomaly_score"].(float64); ok {
			ctx.BehaviorData.AnomalyScore = v
		}
	}

	if customData, ok := inputData["custom_data"].(map[string]interface{}); ok {
		for k, v := range customData {
			ctx.CustomData[k] = v
		}
	}

	return ctx
}

// evaluateExpression 评估表达式
func (s *RiskRuleService) evaluateExpression(condition string, ctx *RiskContext) bool {
	result := s.evaluateSimpleCondition(condition, ctx)
	return result
}

// evaluateSimpleCondition 评估简单条件
func (s *RiskRuleService) evaluateSimpleCondition(condition string, ctx *RiskContext) bool {
	if ctx.IsWhitelisted {
		return false
	}

	condition = strings.TrimSpace(condition)

	// 检查是否包含 AND
	if strings.Contains(condition, " AND ") {
		parts := s.splitByTopLevel(condition, " AND ")
		for _, part := range parts {
			if !s.evaluateSimpleCondition(part, ctx) {
				return false
			}
		}
		return true
	}

	// 如果是括号表达式且包含 OR
	if strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		inner := strings.TrimSpace(condition[1 : len(condition)-1])
		if strings.Contains(inner, " OR ") {
			return s.evaluateOrExpression(inner, ctx)
		}
		if strings.Contains(inner, " AND ") {
			parts := s.splitByTopLevel(inner, " AND ")
			for _, part := range parts {
				if !s.evaluateSimpleCondition(part, ctx) {
					return false
				}
			}
			return true
		}
		return s.evaluateSimpleCondition(inner, ctx)
	}

	// 处理 OR 表达式
	if strings.Contains(condition, " OR ") {
		return s.evaluateOrExpression(condition, ctx)
	}

	return s.evaluateSingleCondition(condition, ctx)
}

// splitByTopLevel 按顶层分隔符分割（不考虑括号内的分隔符）
func (s *RiskRuleService) splitByTopLevel(str, delimiter string) []string {
	result := make([]string, 0)
	current := ""
	depth := 0

	for i := 0; i < len(str); i++ {
		ch := str[i]

		if ch == '(' {
			depth++
			current += string(ch)
		} else if ch == ')' {
			depth--
			current += string(ch)
		} else if depth == 0 && i+len(delimiter) <= len(str) && str[i:i+len(delimiter)] == delimiter {
			part := strings.TrimSpace(current)
			if part != "" {
				result = append(result, part)
			}
			current = ""
			i += len(delimiter) - 1
		} else {
			current += string(ch)
		}
	}

	part := strings.TrimSpace(current)
	if part != "" {
		result = append(result, part)
	}

	return result
}

// evaluateOrExpression 评估 OR 表达式
func (s *RiskRuleService) evaluateOrExpression(condition string, ctx *RiskContext) bool {
	parts := s.splitByTopLevel(condition, " OR ")

	for _, part := range parts {
		part = strings.TrimSpace(part)
		if s.evaluateSingleCondition(part, ctx) {
			return true
		}
	}
	return false
}

// parseConditions 解析条件字符串
func (s *RiskRuleService) parseConditions(condition string) []string {
	conditions := make([]string, 0)

	condition = strings.TrimSpace(condition)

	if strings.Contains(condition, " AND ") {
		parts := strings.Split(condition, " AND ")
		for _, part := range parts {
			parsed := s.parseConditions(strings.TrimSpace(part))
			conditions = append(conditions, parsed...)
		}
		return conditions
	}

	if strings.Contains(condition, " OR ") {
		parts := strings.Split(condition, " OR ")
		orConditions := make([]string, 0)
		for _, part := range parts {
			parsed := s.parseConditions(strings.TrimSpace(part))
			orConditions = append(orConditions, parsed...)
		}
		conditions = append(conditions, "("+strings.Join(orConditions, " OR ")+")")
		return conditions
	}

	return append(conditions, condition)
}

// evaluateSingleCondition 评估单个条件
func (s *RiskRuleService) evaluateSingleCondition(condition string, ctx *RiskContext) bool {
	condition = strings.TrimSpace(condition)

	// 处理括号表达式
	if strings.HasPrefix(condition, "(") && strings.HasSuffix(condition, ")") {
		inner := condition[1 : len(condition)-1]
		inner = strings.TrimSpace(inner)

		// 处理括号内的 OR
		if strings.Contains(inner, " OR ") {
			parts := strings.Split(inner, " OR ")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if s.evaluateSingleCondition(part, ctx) {
					return true
				}
			}
			return false
		}

		// 处理括号内的 AND
		if strings.Contains(inner, " AND ") {
			parts := strings.Split(inner, " AND ")
			for _, part := range parts {
				part = strings.TrimSpace(part)
				if !s.evaluateSingleCondition(part, ctx) {
					return false
				}
			}
			return true
		}

		return s.evaluateSingleCondition(inner, ctx)
	}

	// 处理嵌套括号
	openCount := 0
	startIdx := 0
	for i, ch := range condition {
		if ch == '(' {
			if openCount == 0 {
				startIdx = i
			}
			openCount++
		} else if ch == ')' {
			openCount--
			if openCount == 0 {
				inner := condition[startIdx+1 : i]
				return s.evaluateSingleCondition(inner, ctx)
			}
		}
	}

	return s.evaluateConditionExpr(condition, ctx)
}

// evaluateConditionExpr 评估条件表达式
func (s *RiskRuleService) evaluateConditionExpr(condition string, ctx *RiskContext) bool {
	condition = strings.TrimSpace(condition)

	operators := []string{">=", "<=", "!=", "==", ">", "<", "contains", "startsWith", "endsWith", "in", "matches"}

	var operator string
	var left, right string

	// 检查是否有操作符
	for _, op := range operators {
		if strings.Contains(condition, op) {
			operator = op
			parts := strings.SplitN(condition, op, 2)
			if len(parts) == 2 {
				left = strings.TrimSpace(parts[0])
				right = strings.TrimSpace(parts[1])
				break
			}
		}
	}

	// 如果没有找到操作符，检查是否是纯布尔字段
	if operator == "" {
		left = condition
		right = "true"
		operator = "=="
	}

	leftVal := s.getFieldValue(left, ctx)
	rightVal := s.parseValue(right)

	switch operator {
	case ">":
		return s.compareNumeric(leftVal, rightVal, 1)
	case ">=":
		return s.compareNumeric(leftVal, rightVal, 0)
	case "<":
		return s.compareNumeric(leftVal, rightVal, -1)
	case "<=":
		return s.compareNumeric(leftVal, rightVal, -1) || s.compareNumeric(leftVal, rightVal, 0)
	case "==":
		return s.compareEqual(leftVal, rightVal)
	case "!=":
		return !s.compareEqual(leftVal, rightVal)
	case "contains":
		return s.stringContains(leftVal, rightVal)
	case "startsWith":
		return s.stringStartsWith(leftVal, rightVal)
	case "endsWith":
		return s.stringEndsWith(leftVal, rightVal)
	case "in":
		return s.valueIn(leftVal, rightVal)
	case "matches":
		return s.matchesRegex(leftVal, rightVal)
	}

	return false
}

// stringStartsWith 检查字符串前缀
func (s *RiskRuleService) stringStartsWith(left, right interface{}) bool {
	leftStr, leftOk := left.(string)
	rightStr, rightOk := right.(string)
	if !leftOk || !rightOk {
		return false
	}
	return strings.HasPrefix(leftStr, rightStr)
}

// stringEndsWith 检查字符串后缀
func (s *RiskRuleService) stringEndsWith(left, right interface{}) bool {
	leftStr, leftOk := left.(string)
	rightStr, rightOk := right.(string)
	if !leftOk || !rightOk {
		return false
	}
	return strings.HasSuffix(leftStr, rightStr)
}

// getFieldValue 获取字段值
func (s *RiskRuleService) getFieldValue(field string, ctx *RiskContext) interface{} {
	field = strings.TrimSpace(field)

	switch field {
	case "request_count", "requests_per_minute", "requests":
		return float64(ctx.RequestCount)
	case "failure_count":
		return float64(ctx.FailureCount)
	case "success_count":
		return float64(ctx.SuccessCount)
	case "hour":
		return float64(ctx.Hour)
	case "timestamp":
		return float64(ctx.Timestamp)
	case "is_blacklisted", "ip_in_blacklist":
		return ctx.IsBlacklisted
	case "is_whitelisted":
		return ctx.IsWhitelisted
	case "is_vpn":
		return ctx.IsVPN
	case "is_proxy":
		return ctx.IsProxy
	case "is_tor":
		return ctx.IsTor
	case "is_hosting":
		return ctx.IsHosting
	case "ip_reputation":
		return ctx.IPReputation
	case "country":
		return ctx.Country
	case "asn":
		return float64(ctx.ASN)
	case "avg_speed":
		if ctx.SpeedMetrics != nil {
			return ctx.SpeedMetrics.AvgSpeed
		}
	case "max_speed":
		if ctx.SpeedMetrics != nil {
			return ctx.SpeedMetrics.MaxSpeed
		}
	case "min_speed":
		if ctx.SpeedMetrics != nil {
			return ctx.SpeedMetrics.MinSpeed
		}
	case "speed_variance":
		if ctx.SpeedMetrics != nil {
			return ctx.SpeedMetrics.SpeedVariance
		}
	case "speed_consistency":
		if ctx.SpeedMetrics != nil {
			return ctx.SpeedMetrics.SpeedConsistency
		}
	case "path_efficiency":
		if ctx.TrajectoryData != nil {
			return ctx.TrajectoryData.PathEfficiency
		}
	case "curvature_avg":
		if ctx.TrajectoryData != nil {
			return ctx.TrajectoryData.CurvatureAvg
		}
	case "curvature_variance":
		if ctx.TrajectoryData != nil {
			return ctx.TrajectoryData.CurvatureVariance
		}
	case "direction_changes":
		if ctx.TrajectoryData != nil {
			return float64(ctx.TrajectoryData.DirectionChanges)
		}
	case "micro_corrections":
		if ctx.TrajectoryData != nil {
			return float64(ctx.TrajectoryData.MicroCorrections)
		}
	case "backtrack_count":
		if ctx.TrajectoryData != nil {
			return float64(ctx.TrajectoryData.BacktrackCount)
		}
	case "sinuosity":
		if ctx.TrajectoryData != nil {
			return ctx.TrajectoryData.Sinuosity
		}
	case "pause_count":
		if ctx.BehaviorData != nil {
			return float64(ctx.BehaviorData.PauseCount)
		}
	case "total_pause_duration":
		if ctx.BehaviorData != nil {
			return ctx.BehaviorData.TotalPauseDuration
		}
	case "hesitation_time":
		if ctx.BehaviorData != nil {
			return ctx.BehaviorData.HesitationTime
		}
	case "click_regularity":
		if ctx.BehaviorData != nil {
			return ctx.BehaviorData.ClickRegularity
		}
	case "position_entropy":
		if ctx.BehaviorData != nil {
			return ctx.BehaviorData.PositionEntropy
		}
	case "human_likeness_score":
		if ctx.BehaviorData != nil {
			return ctx.BehaviorData.HumanLikenessScore
		}
	case "anomaly_score":
		if ctx.BehaviorData != nil {
			return ctx.BehaviorData.AnomalyScore
		}
	case "fingerprint_occurrences":
		if v, ok := ctx.CustomData["fingerprint_occurrences"]; ok {
			if f, ok := v.(float64); ok {
				return f
			}
		}
	default:
		if v, ok := ctx.CustomData[field]; ok {
			return v
		}
	}

	return nil
}

// parseValue 解析值
func (s *RiskRuleService) parseValue(value string) interface{} {
	value = strings.TrimSpace(value)

	if strings.HasPrefix(value, "\"") && strings.HasSuffix(value, "\"") {
		return value[1 : len(value)-1]
	}
	if strings.HasPrefix(value, "'") && strings.HasSuffix(value, "'") {
		return value[1 : len(value)-1]
	}

	if f, err := strconv.ParseFloat(value, 64); err == nil {
		return f
	}

	if value == "true" {
		return true
	}
	if value == "false" {
		return false
	}

	return value
}

// compareNumeric 比较数值
func (s *RiskRuleService) compareNumeric(left, right interface{}, op int) bool {
	leftNum := s.toFloat64(left)
	rightNum := s.toFloat64(right)

	switch op {
	case 1:
		return leftNum > rightNum
	case 0:
		return leftNum >= rightNum
	case -1:
		return leftNum < rightNum
	}
	return false
}

// compareEqual 比较相等
func (s *RiskRuleService) compareEqual(left, right interface{}) bool {
	if left == nil || right == nil {
		return left == right
	}

	if leftStr, ok := left.(string); ok {
		if rightStr, ok := right.(string); ok {
			return leftStr == rightStr
		}
	}

	leftNum := s.toFloat64(left)
	rightNum := s.toFloat64(right)

	return math.Abs(leftNum-rightNum) < 0.0001
}

// stringContains 检查字符串包含
func (s *RiskRuleService) stringContains(left, right interface{}) bool {
	leftStr := fmt.Sprintf("%v", left)
	rightStr := fmt.Sprintf("%v", right)
	return strings.Contains(leftStr, rightStr)
}

// valueIn 检查值是否在列表中
func (s *RiskRuleService) valueIn(value, list interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)

	if listStr, ok := list.(string); ok {
		listStr = strings.Trim(listStr, "[]()")
		parts := strings.Split(listStr, ",")
		for _, part := range parts {
			if strings.TrimSpace(part) == valueStr {
				return true
			}
		}
	}
	return false
}

// matchesRegex 匹配正则表达式
func (s *RiskRuleService) matchesRegex(value, pattern interface{}) bool {
	valueStr := fmt.Sprintf("%v", value)
	patternStr := fmt.Sprintf("%v", pattern)

	matched, _ := regexp.MatchString(patternStr, valueStr)
	return matched
}

// toFloat64 转换为float64
func (s *RiskRuleService) toFloat64(value interface{}) float64 {
	if value == nil {
		return 0
	}

	switch v := value.(type) {
	case float64:
		return v
	case float32:
		return float64(v)
	case int:
		return float64(v)
	case int64:
		return float64(v)
	case int32:
		return float64(v)
	case string:
		if f, err := strconv.ParseFloat(v, 64); err == nil {
			return f
		}
	}
	return 0
}

// InitializeDefaultTemplates 初始化默认规则模板
func (s *RiskRuleService) InitializeDefaultTemplates() error {
	defaultTemplates := []models.RiskRuleTemplate{
		// ========== 频率限制类 ==========
		{
			Name:        "IP频率限制",
			Description: "限制单个IP在指定时间窗口内的请求次数",
			Category:    "rate_limit",
			RuleType:    "rate_limit",
			Condition:   "requests_per_minute > 100",
			Action:      "captcha",
			Params:      `{"window": 60, "threshold": 100}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "会话频率限制",
			Description: "限制单个会话在指定时间内的请求次数",
			Category:    "rate_limit",
			RuleType:    "rate_limit",
			Condition:   "request_count > 50",
			Action:      "captcha",
			Params:      `{"window": 300, "threshold": 50}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "全局频率限制",
			Description: "全局请求频率限制",
			Category:    "rate_limit",
			RuleType:    "rate_limit",
			Condition:   "request_count > 1000",
			Action:      "block",
			Params:      `{"window": 60, "threshold": 1000}`,
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 行为检测类 ==========
		{
			Name:        "异常速度检测",
			Description: "检测鼠标/触摸移动速度异常的请求",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "avg_speed > 2000 OR path_efficiency > 0.95",
			Action:      "block",
			Params:      `{"speed_threshold": 2000, "efficiency_threshold": 0.95}`,
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "过于完美的轨迹",
			Description: "检测过于完美、缺乏人类特征的轨迹",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "path_efficiency > 0.98 AND speed_consistency > 0.98",
			Action:      "block",
			Params:      `{"efficiency_threshold": 0.98, "consistency_threshold": 0.98}`,
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "无停顿检测",
			Description: "检测长时间操作无任何停顿的行为",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "pause_count == 0",
			Action:      "captcha",
			Params:      `{"min_duration": 1000}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "犹豫时间过短",
			Description: "检测点击前犹豫时间过短的行为",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "hesitation_time < 50",
			Action:      "captcha",
			Params:      `{"min_hesitation": 50}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "点击间隔过于规律",
			Description: "检测点击间隔过于规律的自动化行为",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "click_regularity > 0.98",
			Action:      "block",
			Params:      `{"regularity_threshold": 0.98}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "异常时间检测",
			Description: "检测非正常时段的高频请求",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "(hour < 6 OR hour > 22) AND request_count > 50",
			Action:      "captcha",
			Params:      `{"low_hour": 6, "high_hour": 22, "threshold": 50}`,
			Severity:    "warning",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 设备指纹类 ==========
		{
			Name:        "设备指纹重复",
			Description: "检测设备指纹重复出现的情况",
			Category:    "device",
			RuleType:    "device_fingerprint",
			Condition:   "fingerprint_occurrences > 5",
			Action:      "review",
			Params:      `{"threshold": 5}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "新设备首次操作",
			Description: "检测新设备首次操作时的异常行为",
			Category:    "device",
			RuleType:    "device",
			Condition:   "failure_count > 3",
			Action:      "captcha",
			Params:      `{"failure_threshold": 3}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 黑名单类 ==========
		{
			Name:        "黑名单IP",
			Description: "拦截已知的恶意IP地址",
			Category:    "blacklist",
			RuleType:    "ip_block",
			Condition:   "is_blacklisted == true",
			Action:      "block",
			Params:      "{}",
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "VPN检测",
			Description: "检测VPN使用并根据配置采取行动",
			Category:    "blacklist",
			RuleType:    "vpn_detection",
			Condition:   "is_vpn == true",
			Action:      "captcha",
			Params:      `{"strict_mode": false}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "代理检测",
			Description: "检测代理服务器使用",
			Category:    "blacklist",
			RuleType:    "proxy_detection",
			Condition:   "is_proxy == true",
			Action:      "captcha",
			Params:      `{"strict_mode": false}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "Tor节点检测",
			Description: "检测Tor网络出口节点",
			Category:    "blacklist",
			RuleType:    "tor_detection",
			Condition:   "is_tor == true",
			Action:      "review",
			Params:      `{}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "托管服务检测",
			Description: "检测托管服务商IP地址",
			Category:    "blacklist",
			RuleType:    "hosting_detection",
			Condition:   "is_hosting == true",
			Action:      "captcha",
			Params:      `{}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "IP信誉低",
			Description: "检测IP信誉评分低的请求",
			Category:    "blacklist",
			RuleType:    "ip_reputation",
			Condition:   "ip_reputation == \"low\"",
			Action:      "captcha",
			Params:      `{}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 轨迹分析类 ==========
		{
			Name:        "轨迹曲率异常",
			Description: "检测轨迹曲率异常的行为",
			Category:    "trajectory",
			RuleType:    "trajectory",
			Condition:   "curvature_avg < 0.02",
			Action:      "captcha",
			Params:      `{"min_curvature": 0.02}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "方向变化过少",
			Description: "检测轨迹方向变化过少的自动化行为",
			Category:    "trajectory",
			RuleType:    "trajectory",
			Condition:   "direction_changes < 3",
			Action:      "captcha",
			Params:      `{"min_direction_changes": 3}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "无微修正动作",
			Description: "检测轨迹无微修正动作的自动化行为",
			Category:    "trajectory",
			RuleType:    "trajectory",
			Condition:   "micro_corrections == 0",
			Action:      "captcha",
			Params:      `{}`,
			Severity:    "medium",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "回退行为检测",
			Description: "检测轨迹中的回退行为",
			Category:    "trajectory",
			RuleType:    "trajectory",
			Condition:   "backtrack_count > 3",
			Action:      "review",
			Params:      `{"threshold": 3}`,
			Severity:    "low",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 机器学习检测类 ==========
		{
			Name:        "ML模型检测-高风险",
			Description: "机器学习模型判定为高风险机器人",
			Category:    "ml",
			RuleType:    "ml_detection",
			Condition:   "anomaly_score > 0.85",
			Action:      "block",
			Params:      `{"threshold": 0.85}`,
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "ML模型检测-中风险",
			Description: "机器学习模型判定为中风险",
			Category:    "ml",
			RuleType:    "ml_detection",
			Condition:   "anomaly_score > 0.7",
			Action:      "captcha",
			Params:      `{"threshold": 0.7}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "人类相似度低",
			Description: "行为特征与人类相似度低",
			Category:    "ml",
			RuleType:    "human_likeness",
			Condition:   "human_likeness_score < 0.2",
			Action:      "captcha",
			Params:      `{"threshold": 0.2}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 综合风险类 ==========
		{
			Name:        "组合风险-多指标异常",
			Description: "多个风险指标同时异常",
			Category:    "combined",
			RuleType:    "combined",
			Condition:   "path_efficiency > 0.95 AND avg_speed > 1500",
			Action:      "block",
			Params:      `{}`,
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "失败次数过多",
			Description: "检测短时间内失败次数过多的请求",
			Category:    "combined",
			RuleType:    "failure",
			Condition:   "failure_count > 5",
			Action:      "block",
			Params:      `{"threshold": 5, "window": 600}`,
			Severity:    "high",
			IsActive:    true,
			IsSystem:    true,
		},

		// ========== 地理位置类 ==========
		{
			Name:        "高风险国家",
			Description: "来自高风险国家的请求",
			Category:    "geo",
			RuleType:    "geo_block",
			Condition:   "country in [\"XX\", \"YY\"]",
			Action:      "review",
			Params:      `{"high_risk_countries": ["XX", "YY"]}`,
			Severity:    "medium",
			IsActive:    false,
			IsSystem:    true,
		},
	}

	for _, tpl := range defaultTemplates {
		var count int64
		s.db.Model(&models.RiskRuleTemplate{}).Where("name = ? AND is_system = true", tpl.Name).Count(&count)
		if count == 0 {
			if err := s.db.Create(&tpl).Error; err != nil {
				return err
			}
		}
	}

	return nil
}
