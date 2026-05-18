package service

import (
	"encoding/json"
	"fmt"
	"math"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type Action string

const (
	ActionAllow      Action = "allow"
	ActionBlock      Action = "block"
	ActionCaptcha    Action = "captcha"
	ActionReview     Action = "review"
	ActionWarn       Action = "warn"
	ActionLog        Action = "log"
)

type DisposalStrategy struct {
	ID          uint      `json:"id" gorm:"primaryKey"`
	Name        string    `json:"name" gorm:"size:255"`
	Description string    `json:"description" gorm:"type:text"`
	RiskLevel   string    `json:"risk_level" gorm:"size:20;index"`
	Priority    int       `json:"priority" gorm:"default:100"`
	Action      string    `json:"action" gorm:"size:50"`
	ActionConfig string   `json:"action_config" gorm:"type:text"`
	IsEnabled   bool      `json:"is_enabled" gorm:"default:true"`
	Conditions  string    `json:"conditions" gorm:"type:text"`
	Cooldown    int       `json:"cooldown" gorm:"default:300"`
	MaxActions  int       `json:"max_actions" gorm:"default:1"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
}

func (DisposalStrategy) TableName() string {
	return "disposal_strategies"
}

type DisposalRecord struct {
	ID            uint      `json:"id" gorm:"primaryKey"`
	StrategyID    uint      `json:"strategy_id" gorm:"index"`
	SessionID     string    `json:"session_id" gorm:"size:100;index"`
	IPAddress     string    `json:"ip_address" gorm:"size:50;index"`
	Fingerprint   string    `json:"fingerprint" gorm:"size:64;index"`
	RiskLevel     string    `json:"risk_level" gorm:"size:20"`
	RiskScore     float64   `json:"risk_score"`
	Action        string    `json:"action" gorm:"size:50"`
	ActionResult  string    `json:"action_result" gorm:"size:50"`
	Reason        string    `json:"reason" gorm:"type:text"`
	TriggeredRules string   `json:"triggered_rules" gorm:"type:text"`
	Duration      int64     `json:"duration" gorm:"comment:'处理耗时(ms)'"`
	Success       bool      `json:"success" gorm:"default:true"`
	CreatedAt     time.Time `json:"created_at" gorm:"index"`
}

func (DisposalRecord) TableName() string {
	return "disposal_records"
}

type DisposalStatistics struct {
	TotalRecords      int64            `json:"total_records"`
	SuccessCount      int64            `json:"success_count"`
	FailureCount      int64            `json:"failure_count"`
	ActionStats       map[string]int64 `json:"action_stats"`
	RiskLevelStats    map[string]int64 `json:"risk_level_stats"`
	AvgDuration       float64          `json:"avg_duration"`
	AvgRiskScore      float64          `json:"avg_risk_score"`
	TopStrategies     []StrategyStat   `json:"top_strategies"`
	TopTriggeredRules []RuleStat       `json:"top_triggered_rules"`
}

type StrategyStat struct {
	StrategyID   uint   `json:"strategy_id"`
	StrategyName string `json:"strategy_name"`
	Count        int64  `json:"count"`
	SuccessRate  float64 `json:"success_rate"`
}

type RuleStat struct {
	RuleID   uint   `json:"rule_id"`
	RuleName string `json:"rule_name"`
	Count    int64  `json:"count"`
}

type DisposalService struct {
	db          *gorm.DB
	strategies  map[string]*DisposalStrategy
	actionCounts map[string]int
	mu          sync.RWMutex
}

func NewDisposalService() *DisposalService {
	return &DisposalService{
		db:          database.DB,
		strategies:  make(map[string]*DisposalStrategy),
		actionCounts: make(map[string]int),
	}
}

func (s *DisposalService) Initialize() error {
	if err := s.db.AutoMigrate(&DisposalStrategy{}, &DisposalRecord{}); err != nil {
		return fmt.Errorf("自动迁移失败: %w", err)
	}

	if err := s.loadStrategies(); err != nil {
		return fmt.Errorf("加载策略失败: %w", err)
	}

	if err := s.initializeDefaultStrategies(); err != nil {
		return fmt.Errorf("初始化默认策略失败: %w", err)
	}

	return nil
}

func (s *DisposalService) loadStrategies() error {
	var strategies []DisposalStrategy
	if err := s.db.Where("is_enabled = ?", true).Find(&strategies).Error; err != nil {
		return err
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.strategies = make(map[string]*DisposalStrategy)
	for i := range strategies {
		s.strategies[strategies[i].RiskLevel] = &strategies[i]
	}

	return nil
}

func (s *DisposalService) initializeDefaultStrategies() error {
	defaultStrategies := []DisposalStrategy{
		{
			Name:        "最小风险处理",
			Description: "对最小风险请求的默认处理策略",
			RiskLevel:   "minimal",
			Priority:    100,
			Action:      "allow",
			ActionConfig: `{"verify": false, "log": true}`,
			IsEnabled:   true,
			Cooldown:    0,
			MaxActions:  1,
		},
		{
			Name:        "低风险处理",
			Description: "对低风险请求的默认处理策略",
			RiskLevel:   "low",
			Priority:    90,
			Action:      "allow",
			ActionConfig: `{"verify": false, "log": true}`,
			IsEnabled:   true,
			Cooldown:    0,
			MaxActions:  1,
		},
		{
			Name:        "中风险处理",
			Description: "对中风险请求的默认处理策略",
			RiskLevel:   "medium",
			Priority:    70,
			Action:      "captcha",
			ActionConfig: `{"verify": true, "verify_type": "slider", "log": true, "notify": true}`,
			IsEnabled:   true,
			Cooldown:    300,
			MaxActions:  3,
		},
		{
			Name:        "高风险处理",
			Description: "对高风险请求的默认处理策略",
			RiskLevel:   "high",
			Priority:    50,
			Action:      "captcha",
			ActionConfig: `{"verify": true, "verify_type": "complex", "log": true, "notify": true, "alert": true}`,
			IsEnabled:   true,
			Cooldown:    600,
			MaxActions:  5,
		},
		{
			Name:        "严重风险处理",
			Description: "对严重风险请求的默认处理策略",
			RiskLevel:   "critical",
			Priority:    30,
			Action:      "block",
			ActionConfig: `{"verify": false, "log": true, "notify": true, "alert": true, "blacklist": true}`,
			IsEnabled:   true,
			Cooldown:    3600,
			MaxActions:  1,
		},
		{
			Name:        "未知风险处理",
			Description: "对未知风险请求的默认处理策略",
			RiskLevel:   "unknown",
			Priority:    80,
			Action:      "captcha",
			ActionConfig: `{"verify": true, "verify_type": "basic", "log": true}`,
			IsEnabled:   true,
			Cooldown:    300,
			MaxActions:  3,
		},
	}

	for _, strategy := range defaultStrategies {
		var count int64
		s.db.Model(&DisposalStrategy{}).Where("risk_level = ? AND name = ?", strategy.RiskLevel, strategy.Name).Count(&count)
		if count == 0 {
			if err := s.db.Create(&strategy).Error; err != nil {
				return err
			}
		}
	}

	return s.loadStrategies()
}

type DisposalContext struct {
	SessionID        string
	IPAddress        string
	Fingerprint      string
	UserAgent        string
	RiskLevel        string
	RiskScore        float64
	TriggeredRules   []string
	RiskFactors      []string
	RequestCount     int
	FailureCount     int
	SuccessCount     int
	IPReputation     string
	IsVPN            bool
	IsProxy          bool
	IsTor            bool
	IsHosting        bool
	LastActionTime   time.Time
	CustomData       map[string]interface{}
}

type DisposalResult struct {
	Action       string                 `json:"action"`
	Success      bool                   `json:"success"`
	Reason       string                 `json:"reason"`
	StrategyID   uint                   `json:"strategy_id"`
	StrategyName string                 `json:"strategy_name"`
	Config       map[string]interface{} `json:"config"`
	NextAction   string                 `json:"next_action,omitempty"`
	Cooldown     int                    `json:"cooldown"`
	Duration     int64                  `json:"duration"`
}

func (s *DisposalService) ExecuteDisposal(ctx *DisposalContext) (*DisposalResult, error) {
	startTime := time.Now()

	if ctx == nil {
		return nil, fmt.Errorf("处置上下文不能为空")
	}

	strategy := s.getStrategy(ctx.RiskLevel)
	if strategy == nil {
		return &DisposalResult{
			Action:  "allow",
			Success: true,
			Reason:  "无匹配策略，默认放行",
		}, nil
	}

	if !s.checkCooldown(ctx, strategy) {
		return &DisposalResult{
			Action:  "allow",
			Success: true,
			Reason:  "冷却期内，跳过处置",
			Cooldown: strategy.Cooldown,
		}, nil
	}

	result := &DisposalResult{
		StrategyID:   strategy.ID,
		StrategyName: strategy.Name,
		Duration:     time.Since(startTime).Milliseconds(),
	}

	config := s.parseActionConfig(strategy.ActionConfig)
	result.Config = config

	action := Action(strategy.Action)
	switch action {
	case ActionAllow:
		result.Action = "allow"
		result.Success = true
		result.Reason = fmt.Sprintf("风险等级 %s 允许通过", ctx.RiskLevel)

	case ActionBlock:
		result.Action = "block"
		result.Success = s.executeBlockAction(ctx, config)
		result.Reason = fmt.Sprintf("风险等级 %s 阻止访问", ctx.RiskLevel)
		if config["blacklist"] == true {
			s.addToBlacklist(ctx, "auto", result.Reason)
		}

	case ActionCaptcha:
		result.Action = "captcha"
		result.Success = true
		result.Reason = fmt.Sprintf("风险等级 %s 需要验证", ctx.RiskLevel)
		result.Config["verify"] = true

	case ActionReview:
		result.Action = "review"
		result.Success = true
		result.Reason = fmt.Sprintf("风险等级 %s 需要人工审核", ctx.RiskLevel)
		s.notifyReview(ctx)

	case ActionWarn:
		result.Action = "warn"
		result.Success = true
		result.Reason = fmt.Sprintf("风险等级 %s 警告", ctx.RiskLevel)
		s.sendWarning(ctx)

	case ActionLog:
		result.Action = "log"
		result.Success = true
		result.Reason = fmt.Sprintf("风险等级 %s 仅记录", ctx.RiskLevel)

	default:
		result.Action = "allow"
		result.Success = true
		result.Reason = "未知处置动作，默认放行"
	}

	record := &DisposalRecord{
		StrategyID:     strategy.ID,
		SessionID:      ctx.SessionID,
		IPAddress:      ctx.IPAddress,
		Fingerprint:    ctx.Fingerprint,
		RiskLevel:      ctx.RiskLevel,
		RiskScore:      ctx.RiskScore,
		Action:         result.Action,
		ActionResult:   result.Reason,
		Reason:         result.Reason,
		TriggeredRules: strings.Join(ctx.TriggeredRules, ","),
		Duration:       result.Duration,
		Success:        result.Success,
	}
	s.saveRecord(record)

	s.updateActionCount(ctx.SessionID)

	return result, nil
}

func (s *DisposalService) getStrategy(riskLevel string) *DisposalStrategy {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if strategy, ok := s.strategies[riskLevel]; ok {
		return strategy
	}

	if strategy, ok := s.strategies["unknown"]; ok {
		return strategy
	}

	return nil
}

func (s *DisposalService) checkCooldown(ctx *DisposalContext, strategy *DisposalStrategy) bool {
	if strategy.Cooldown <= 0 {
		return true
	}

	key := fmt.Sprintf("%s:%s", ctx.SessionID, strategy.RiskLevel)
	s.mu.RLock()
	count := s.actionCounts[key]
	s.mu.RUnlock()

	return count < strategy.MaxActions
}

func (s *DisposalService) updateActionCount(key string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.actionCounts[key]++
	go func() {
		time.AfterFunc(time.Duration(3600)*time.Second, func() {
			s.mu.Lock()
			delete(s.actionCounts, key)
			s.mu.Unlock()
		})
	}()
}

func (s *DisposalService) parseActionConfig(config string) map[string]interface{} {
	result := make(map[string]interface{})
	if config == "" {
		return result
	}

	if err := json.Unmarshal([]byte(config), &result); err != nil {
		result["error"] = err.Error()
	}
	return result
}

func (s *DisposalService) executeBlockAction(ctx *DisposalContext, config map[string]interface{}) bool {
	return true
}

func (s *DisposalService) addToBlacklist(ctx *DisposalContext, source string, reason string) {
	blacklist := &models.Blacklist{
		Target: ctx.IPAddress,
		Type:   "ip",
		Source: source,
		Reason: reason,
		Action: "block",
		Status: "active",
	}

	var existing models.Blacklist
	if err := s.db.Where("target = ? AND type = ?", ctx.IPAddress, "ip").First(&existing).Error; err == gorm.ErrRecordNotFound {
		s.db.Create(blacklist)
	}
}

func (s *DisposalService) notifyReview(ctx *DisposalContext) {
}

func (s *DisposalService) sendWarning(ctx *DisposalContext) {
}

func (s *DisposalService) saveRecord(record *DisposalRecord) {
	if err := s.db.Create(record).Error; err != nil {
	}
}

func (s *DisposalService) GetStatistics(startDate, endDate string) (*DisposalStatistics, error) {
	stats := &DisposalStatistics{
		ActionStats:    make(map[string]int64),
		RiskLevelStats: make(map[string]int64),
	}

	query := s.db.Model(&DisposalRecord{})
	if startDate != "" && endDate != "" {
		query = query.Where("created_at BETWEEN ? AND ?", startDate, endDate)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, err
	}
	stats.TotalRecords = total

	var successCount, failureCount int64
	query.Where("success = ?", true).Count(&successCount)
	query.Where("success = ?", false).Count(&failureCount)
	stats.SuccessCount = successCount
	stats.FailureCount = failureCount

	type ActionStat struct {
		Action string
		Count  int64
	}
	var actionStats []ActionStat
	query.Select("action, count(*) as count").Group("action").Scan(&actionStats)
	for _, as := range actionStats {
		stats.ActionStats[as.Action] = as.Count
	}

	type RiskStat struct {
		RiskLevel string
		Count     int64
	}
	var riskStats []RiskStat
	query.Select("risk_level, count(*) as count").Group("risk_level").Scan(&riskStats)
	for _, rs := range riskStats {
		stats.RiskLevelStats[rs.RiskLevel] = rs.Count
	}

	var avgDuration, avgRiskScore float64
	query.Select("COALESCE(AVG(duration), 0), COALESCE(AVG(risk_score), 0)").Row().Scan(&avgDuration, &avgRiskScore)
	stats.AvgDuration = avgDuration
	stats.AvgRiskScore = avgRiskScore

	var topStrategies []StrategyStat
	query.Select("strategy_id, count(*) as count").
		Group("strategy_id").
		Order("count desc").
		Limit(10).
		Scan(&topStrategies)
	stats.TopStrategies = topStrategies

	var topRules []RuleStat
	s.db.Raw("SELECT 0 as rule_id, triggered_rules as rule_name, count(*) as count FROM disposal_records WHERE triggered_rules != '' GROUP BY triggered_rules ORDER BY count DESC LIMIT 10").Scan(&topRules)
	stats.TopTriggeredRules = topRules

	return stats, nil
}

func (s *DisposalService) CreateStrategy(strategy *DisposalStrategy) error {
	if err := s.db.Create(strategy).Error; err != nil {
		return err
	}
	return s.loadStrategies()
}

func (s *DisposalService) UpdateStrategy(id uint, strategy *DisposalStrategy) error {
	var existing DisposalStrategy
	if err := s.db.First(&existing, id).Error; err != nil {
		return err
	}

	if err := s.db.Model(&existing).Updates(strategy).Error; err != nil {
		return err
	}
	return s.loadStrategies()
}

func (s *DisposalService) DeleteStrategy(id uint) error {
	if err := s.db.Delete(&DisposalStrategy{}, id).Error; err != nil {
		return err
	}
	return s.loadStrategies()
}

func (s *DisposalService) GetStrategies() ([]DisposalStrategy, error) {
	var strategies []DisposalStrategy
	err := s.db.Order("priority DESC").Find(&strategies).Error
	return strategies, err
}

func (s *DisposalService) GetRecords(limit, offset int, filters map[string]interface{}) ([]DisposalRecord, int64, error) {
	var records []DisposalRecord
	var total int64

	query := s.db.Model(&DisposalRecord{})

	if sessionID, ok := filters["session_id"].(string); ok && sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}
	if ipAddress, ok := filters["ip_address"].(string); ok && ipAddress != "" {
		query = query.Where("ip_address = ?", ipAddress)
	}
	if riskLevel, ok := filters["risk_level"].(string); ok && riskLevel != "" {
		query = query.Where("risk_level = ?", riskLevel)
	}
	if action, ok := filters["action"].(string); ok && action != "" {
		query = query.Where("action = ?", action)
	}

	query.Count(&total)
	err := query.Order("created_at DESC").Limit(limit).Offset(offset).Find(&records).Error

	return records, total, err
}

type RiskAssessmentModel struct {
	Weights map[string]float64
}

func NewRiskAssessmentModel() *RiskAssessmentModel {
	return &RiskAssessmentModel{
		Weights: map[string]float64{
			"speed_anomaly":       0.20,
			"trajectory_anomaly":  0.25,
			"behavior_anomaly":   0.15,
			"device_anomaly":     0.10,
			"network_anomaly":    0.15,
			"ml_score":           0.10,
			"pattern_anomaly":    0.05,
		},
	}
}

type RiskAssessmentResult struct {
	TotalScore        float64            `json:"total_score"`
	ComponentScores   map[string]float64 `json:"component_scores"`
	RiskLevel         string             `json:"risk_level"`
	Confidence        float64            `json:"confidence"`
	Recommendations   []string           `json:"recommendations"`
}

func (m *RiskAssessmentModel) AssessRisk(components map[string]float64) *RiskAssessmentResult {
	result := &RiskAssessmentResult{
		ComponentScores: make(map[string]float64),
		Recommendations: make([]string, 0),
	}

	var totalWeight, weightedScore float64

	for component, score := range components {
		result.ComponentScores[component] = math.Min(math.Max(score, 0), 1)

		if weight, ok := m.Weights[component]; ok {
			weightedScore += score * weight
			totalWeight += weight
		}
	}

	if totalWeight > 0 {
		result.TotalScore = weightedScore / totalWeight
	}

	result.RiskLevel = m.classifyRiskLevel(result.TotalScore)
	result.Confidence = m.calculateConfidence(components)

	result.Recommendations = m.generateRecommendations(result)

	return result
}

func (m *RiskAssessmentModel) classifyRiskLevel(score float64) string {
	switch {
	case score >= 0.8:
		return "critical"
	case score >= 0.6:
		return "high"
	case score >= 0.4:
		return "medium"
	case score >= 0.2:
		return "low"
	default:
		return "minimal"
	}
}

func (m *RiskAssessmentModel) calculateConfidence(components map[string]float64) float64 {
	count := len(components)
	if count == 0 {
		return 0
	}

	confidence := 0.5 + float64(count)*0.05
	return math.Min(confidence, 0.99)
}

func (m *RiskAssessmentModel) generateRecommendations(result *RiskAssessmentResult) []string {
	recs := make([]string, 0)

	switch result.RiskLevel {
	case "critical":
		recs = append(recs, "立即阻止访问", "加入黑名单", "通知安全团队")
	case "high":
		recs = append(recs, "要求额外验证", "记录日志", "考虑限制操作")
	case "medium":
		recs = append(recs, "要求验证码验证", "持续监控")
	case "low":
		recs = append(recs, "正常通过", "记录日志")
	case "minimal":
		recs = append(recs, "直接通过")
	}

	return recs
}

type RiskAlertService struct {
	alerts    []RiskAlert
	thresholds map[string]AlertThreshold
	mu        sync.RWMutex
}

type RiskAlert struct {
	ID          string    `json:"id"`
	AlertType   string    `json:"alert_type"`
	Level       string    `json:"level"`
	Message     string    `json:"message"`
	SessionID   string    `json:"session_id"`
	IPAddress   string    `json:"ip_address"`
	RiskScore   float64   `json:"risk_score"`
	RiskLevel   string    `json:"risk_level"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time `json:"created_at"`
	Acknowledged bool     `json:"acknowledged"`
}

type AlertThreshold struct {
	Metric      string
	Operator    string
	Value       float64
	AlertLevel  string
	Cooldown    time.Duration
}

func NewRiskAlertService() *RiskAlertService {
	service := &RiskAlertService{
		alerts:     make([]RiskAlert, 0),
		thresholds: make(map[string]AlertThreshold),
	}
	service.initializeDefaultThresholds()
	return service
}

func (s *RiskAlertService) initializeDefaultThresholds() {
	s.thresholds["high_risk_rate"] = AlertThreshold{
		Metric:     "high_risk_rate",
		Operator:   ">",
		Value:      0.3,
		AlertLevel: "warning",
		Cooldown:   5 * time.Minute,
	}

	s.thresholds["critical_risk_rate"] = AlertThreshold{
		Metric:     "critical_risk_rate",
		Operator:   ">",
		Value:      0.1,
		AlertLevel: "critical",
		Cooldown:   1 * time.Minute,
	}

	s.thresholds["block_rate"] = AlertThreshold{
		Metric:     "block_rate",
		Operator:   ">",
		Value:      0.5,
		AlertLevel: "warning",
		Cooldown:   10 * time.Minute,
	}

	s.thresholds["avg_risk_score"] = AlertThreshold{
		Metric:     "avg_risk_score",
		Operator:   ">",
		Value:      0.6,
		AlertLevel: "warning",
		Cooldown:   5 * time.Minute,
	}

	s.thresholds["unique_ips"] = AlertThreshold{
		Metric:     "unique_ips",
		Operator:   ">",
		Value:      1000,
		AlertLevel: "info",
		Cooldown:   15 * time.Minute,
	}
}

func (s *RiskAlertService) CheckAndTriggerAlerts(metrics map[string]float64) []RiskAlert {
	s.mu.Lock()
	defer s.mu.Unlock()

	alerts := make([]RiskAlert, 0)

	for name, threshold := range s.thresholds {
		if value, ok := metrics[name]; ok {
			triggered := s.evaluateThreshold(value, threshold)
			if triggered {
				alert := RiskAlert{
					ID:          fmt.Sprintf("alert_%d", time.Now().UnixNano()),
					AlertType:   name,
					Level:       threshold.AlertLevel,
					Message:     fmt.Sprintf("指标 %s 超过阈值: %.2f > %.2f", name, value, threshold.Value),
					RiskScore:   value,
					Metadata:    map[string]interface{}{"metric": name, "value": value, "threshold": threshold.Value},
					CreatedAt:   time.Now(),
				}
				alerts = append(alerts, alert)
				s.alerts = append(s.alerts, alert)
			}
		}
	}

	return alerts
}

func (s *RiskAlertService) evaluateThreshold(value float64, threshold AlertThreshold) bool {
	switch threshold.Operator {
	case ">":
		return value > threshold.Value
	case ">=":
		return value >= threshold.Value
	case "<":
		return value < threshold.Value
	case "<=":
		return value <= threshold.Value
	case "==":
		return math.Abs(value-threshold.Value) < 0.0001
	}
	return false
}

func (s *RiskAlertService) GetAlerts(startTime, endTime time.Time, levels []string) []RiskAlert {
	s.mu.RLock()
	defer s.mu.RUnlock()

	alerts := make([]RiskAlert, 0)
	for _, alert := range s.alerts {
		if alert.CreatedAt.After(startTime) && alert.CreatedAt.Before(endTime) {
			if len(levels) == 0 {
				alerts = append(alerts, alert)
			} else {
				for _, level := range levels {
					if alert.Level == level {
						alerts = append(alerts, alert)
						break
					}
				}
			}
		}
	}

	return alerts
}

func (s *RiskAlertService) AcknowledgeAlert(alertID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	for i := range s.alerts {
		if s.alerts[i].ID == alertID {
			s.alerts[i].Acknowledged = true
			return nil
		}
	}
	return fmt.Errorf("alert not found")
}

func (s *RiskAlertService) ClearOldAlerts(olderThan time.Duration) int {
	s.mu.Lock()
	defer s.mu.Unlock()

	cutoff := time.Now().Add(-olderThan)
	oldCount := 0

	filtered := make([]RiskAlert, 0)
	for _, alert := range s.alerts {
		if alert.CreatedAt.After(cutoff) {
			filtered = append(filtered, alert)
		} else {
			oldCount++
		}
	}
	s.alerts = filtered

	return oldCount
}
