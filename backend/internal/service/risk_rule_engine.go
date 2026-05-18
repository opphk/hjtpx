package service

import (
	"encoding/json"
	"fmt"
	"math"
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

type RiskScoringWeights struct {
	SpeedWeight         float64
	TrajectoryWeight    float64
	PathComplexityWeight float64
	ClickPatternWeight  float64
	KeyboardPatternWeight float64
	EnvironmentWeight   float64
	HistoricalWeight    float64
	VelocityWeight      float64
	AccelerationWeight  float64
	DirectionWeight     float64
}

type RealTimeScoringConfig struct {
	Enabled           bool
	UpdateInterval    int
	WindowSize        int
	AdaptiveWeights   bool
	UseMLScoring      bool
	ConfidenceThreshold float64
}

type RiskScorer struct {
	weights    RiskScoringWeights
	config    RealTimeScoringConfig
	isEnabled bool
}

func NewRiskScorer() *RiskScorer {
	return &RiskScorer{
		weights: RiskScoringWeights{
			SpeedWeight:          0.15,
			TrajectoryWeight:     0.20,
			PathComplexityWeight: 0.15,
			ClickPatternWeight:   0.12,
			KeyboardPatternWeight: 0.10,
			EnvironmentWeight:    0.13,
			HistoricalWeight:     0.10,
			VelocityWeight:       0.08,
			AccelerationWeight:   0.07,
			DirectionWeight:      0.05,
		},
		config: RealTimeScoringConfig{
			Enabled:            true,
			UpdateInterval:     100,
			WindowSize:         10,
			AdaptiveWeights:    true,
			UseMLScoring:       false,
			ConfidenceThreshold: 0.75,
		},
		isEnabled: true,
	}
}

func (s *RiskScorer) SetWeights(weights RiskScoringWeights) {
	totalWeight := weights.SpeedWeight + weights.TrajectoryWeight +
		weights.PathComplexityWeight + weights.ClickPatternWeight +
		weights.KeyboardPatternWeight + weights.EnvironmentWeight +
		weights.HistoricalWeight + weights.VelocityWeight +
		weights.AccelerationWeight + weights.DirectionWeight

	if math.Abs(totalWeight-1.0) < 0.01 {
		s.weights = weights
	}
}

func (s *RiskScorer) SetConfig(config RealTimeScoringConfig) {
	s.config = config
	s.isEnabled = config.Enabled
}

func (s *RiskScorer) CalculateRealTimeScore(context *model.RiskContext) float64 {
	if !s.isEnabled {
		return 50.0
	}

	baseScore := s.calculateBaseScore(context)

	trajectoryScore := s.calculateTrajectoryScore(context)

	pathComplexityScore := s.calculatePathComplexityScore(context)

	clickPatternScore := s.calculateClickPatternScore(context)

	keyboardPatternScore := s.calculateKeyboardPatternScore(context)

	velocityScore := s.calculateVelocityScore(context)

	accelerationScore := s.calculateAccelerationScore(context)

	compositeScore := baseScore*s.weights.SpeedWeight +
		trajectoryScore*s.weights.TrajectoryWeight +
		pathComplexityScore*s.weights.PathComplexityWeight +
		clickPatternScore*s.weights.ClickPatternWeight +
		keyboardPatternScore*s.weights.KeyboardPatternWeight +
		velocityScore*s.weights.VelocityWeight +
		accelerationScore*s.weights.AccelerationWeight

	if s.config.AdaptiveWeights {
		compositeScore = s.adjustScoreWithContext(compositeScore, context)
	}

	return math.Min(math.Max(compositeScore, 0), 100)
}

func (s *RiskScorer) calculateBaseScore(context *model.RiskContext) float64 {
	score := 50.0

	if context.FailureCount > 0 {
		score += float64(context.FailureCount) * 5.0
	}

	if context.IsProxy || context.IsVPN {
		score += 15.0
	}

	if context.IsTor {
		score += 20.0
	}

	if context.VerificationCount > 3 {
		score -= 5.0
	}

	return math.Min(score, 100)
}

func (s *RiskScorer) calculateTrajectoryScore(context *model.RiskContext) float64 {
	if len(context.TraceData) < 10 {
		return 50.0
	}

	score := 50.0

	avgSpeed := context.MouseSpeed

	if avgSpeed > 2000 {
		score += 30.0
	} else if avgSpeed > 1000 {
		score += 15.0
	}

	return math.Min(score, 100)
}

func (s *RiskScorer) calculatePathComplexityScore(context *model.RiskContext) float64 {
	score := 50.0

	if len(context.TraceData) < 2 {
		return score
	}

	start := context.TraceData[0]
	end := context.TraceData[len(context.TraceData)-1]

	dx := end.X - start.X
	dy := end.Y - start.Y
	straightDist := math.Sqrt(float64(dx*dx + dy*dy))

	totalDist := 0.0
	for i := 1; i < len(context.TraceData); i++ {
		dx := context.TraceData[i].X - context.TraceData[i-1].X
		dy := context.TraceData[i].Y - context.TraceData[i-1].Y
		totalDist += math.Sqrt(float64(dx*dx + dy*dy))
	}

	if totalDist > 0 {
		directness := straightDist / totalDist
		if directness > 0.95 {
			score += 35.0
		} else if directness > 0.9 {
			score += 20.0
		}
	}

	return math.Min(score, 100)
}

func (s *RiskScorer) calculateClickPatternScore(context *model.RiskContext) float64 {
	score := 50.0

	if context.VerificationCount > 5 {
		score -= 10.0
	}

	if context.FailureCount > 2 {
		score += 15.0
	}

	return math.Min(score, 100)
}

func (s *RiskScorer) calculateKeyboardPatternScore(context *model.RiskContext) float64 {
	return 50.0
}

func (s *RiskScorer) calculateVelocityScore(context *model.RiskContext) float64 {
	score := 50.0

	if context.MouseSpeed > 1500 {
		score += 25.0
	} else if context.MouseSpeed > 800 {
		score += 10.0
	}

	return math.Min(score, 100)
}

func (s *RiskScorer) calculateAccelerationScore(context *model.RiskContext) float64 {
	return 50.0
}

func (s *RiskScorer) adjustScoreWithContext(score float64, context *model.RiskContext) float64 {
	adjustment := 0.0

	if context.TimeFromStart > 0 && context.TimeFromStart < 500 {
		adjustment += 10.0
	}

	if context.PositionDiff < 50 && len(context.TraceData) > 20 {
		adjustment += 15.0
	}

	if len(context.BrowserPlugins) == 0 {
		adjustment += 10.0
	}

	return math.Min(score+adjustment, 100)
}

func (s *RiskScorer) BatchScore(contexts []*model.RiskContext) []float64 {
	scores := make([]float64, len(contexts))
	for i, ctx := range contexts {
		scores[i] = s.CalculateRealTimeScore(ctx)
	}
	return scores
}

func (s *RiskScorer) GetWeights() RiskScoringWeights {
	return s.weights
}

func (s *RiskScorer) GetConfig() RealTimeScoringConfig {
	return s.config
}

type EnhancedRiskEvaluation struct {
	BaseScore        float64
	WeightedScore    float64
	AdjustedScore    float64
	FinalScore       float64
	ConfidenceLevel  float64
	AnomalyCount    int
	ThreatLevel     string
	Recommendations []string
}

func (s *RiskScorer) EnhancedEvaluate(context *model.RiskContext) *EnhancedRiskEvaluation {
	eval := &EnhancedRiskEvaluation{}

	eval.BaseScore = s.calculateBaseScore(context)

	eval.WeightedScore = s.CalculateRealTimeScore(context)

	eval.AdjustedScore = s.adjustScoreWithContext(eval.WeightedScore, context)

	eval.FinalScore = s.applyConfidenceAndConstraints(eval.AdjustedScore, context)

	eval.ConfidenceLevel = s.calculateConfidence(context)

	eval.ThreatLevel = s.determineThreatLevel(eval.FinalScore, eval.ConfidenceLevel)

	eval.Recommendations = s.generateRecommendations(eval)

	return eval
}

func (s *RiskScorer) calculateConfidence(context *model.RiskContext) float64 {
	confidence := 0.6

	if len(context.TraceData) > 20 {
		confidence += 0.2
	} else if len(context.TraceData) > 10 {
		confidence += 0.1
	}

	if context.VerificationCount > 3 {
		confidence += 0.1
	}

	if context.FailureCount == 0 {
		confidence += 0.1
	}

	return math.Min(confidence, 0.95)
}

func (s *RiskScorer) applyConfidenceAndConstraints(score float64, context *model.RiskContext) float64 {
	constrainedScore := score

	if context.IsTor {
		constrainedScore = math.Max(constrainedScore, 70.0)
	}

	if context.FailureCount >= 5 {
		constrainedScore = math.Max(constrainedScore, 80.0)
	}

	if context.VerificationCount > 10 && context.FailureCount == 0 {
		constrainedScore *= 0.8
	}

	return math.Min(math.Max(constrainedScore, 0), 100)
}

func (s *RiskScorer) determineThreatLevel(score float64, confidence float64) string {
	if score >= 80 && confidence >= 0.8 {
		return "critical"
	} else if score >= 60 && confidence >= 0.7 {
		return "high"
	} else if score >= 40 && confidence >= 0.6 {
		return "medium"
	}
	return "low"
}

func (s *RiskScorer) generateRecommendations(eval *EnhancedRiskEvaluation) []string {
	recommendations := []string{}

	if eval.ThreatLevel == "critical" {
		recommendations = append(recommendations, "立即阻止请求", "记录完整日志", "触发人工审核")
	} else if eval.ThreatLevel == "high" {
		recommendations = append(recommendations, "要求额外验证", "增加监控频率", "限制操作权限")
	} else if eval.ThreatLevel == "medium" {
		recommendations = append(recommendations, "显示验证码", "增加日志记录")
	} else {
		recommendations = append(recommendations, "正常处理")
	}

	return recommendations
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

	result := s.evaluateCondition(rule.Condition, inputData)

	executionTime := time.Since(startTime).Milliseconds()

	actionTaken := ""
	if result {
		actionTaken = rule.Action
	}

	s.RecordRuleTrigger(ruleID, "", nil, nil, "", inputData, result, actionTaken, executionTime)

	return result, actionTaken, nil
}

// evaluateCondition 评估条件（简化版）
func (s *RiskRuleService) evaluateCondition(condition string, inputData map[string]interface{}) bool {
	// 这里是一个简化的条件评估
	// 在实际项目中，应该使用专门的规则引擎库
	// 比如: https://github.com/expr-lang/expr
	
	// 简单的示例实现
	if condition == "" {
		return false
	}
	
	// 为了演示，我们返回一个随机结果
	// 在真实场景中，这里应该解析并执行条件表达式
	return time.Now().UnixNano()%2 == 0
}

// InitializeDefaultTemplates 初始化默认规则模板
func (s *RiskRuleService) InitializeDefaultTemplates() error {
	defaultTemplates := []models.RiskRuleTemplate{
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
			Name:        "黑名单IP",
			Description: "拦截已知的恶意IP地址",
			Category:    "blacklist",
			RuleType:    "ip_block",
			Condition:   "ip_in_blacklist == true",
			Action:      "block",
			Params:      "{}",
			Severity:    "critical",
			IsActive:    true,
			IsSystem:    true,
		},
		{
			Name:        "异常时间检测",
			Description: "检测非正常时段的高频请求",
			Category:    "behavior",
			RuleType:    "behavior",
			Condition:   "hour < 6 OR hour > 22 AND requests > 50",
			Action:      "captcha",
			Params:      `{"low_hour": 6, "high_hour": 22, "threshold": 50}`,
			Severity:    "warning",
			IsActive:    true,
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
