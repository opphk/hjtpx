package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type RiskManager struct {
	db            *gorm.DB
	engineCache   map[string]*RiskEngine
	mu            sync.RWMutex
	defaultConfig *RiskConfig
}

func NewRiskManager(db *gorm.DB) *RiskManager {
	rm := &RiskManager{
		db:          db,
		engineCache: make(map[string]*RiskEngine),
	}

	rm.initializeDefaultConfig()
	return rm
}

func (rm *RiskManager) initializeDefaultConfig() {
	rm.defaultConfig = &RiskConfig{
		ID:           "default",
		Name:         "Default Risk Config",
		Description:  "Default configuration for risk assessment",
		Version:      1,
		Expressions: []RiskExpression{
			{
				ID:          "speed_check",
				Name:        "Speed Check",
				Description: "Check if speed exceeds threshold",
				Type:        RiskRuleTypeCondition,
				Priority:    100,
				Weight:      0.3,
				Enabled:     true,
				Condition: &RiskCondition{
					Field:    "speed",
					Operator: RiskOperatorGt,
					Value:    1500,
				},
				Tags:      []string{"speed", "basic"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "trajectory_check",
				Name:        "Trajectory Smoothness Check",
				Description: "Check if trajectory is too smooth",
				Type:        RiskRuleTypeCondition,
				Priority:    90,
				Weight:      0.25,
				Enabled:     true,
				Condition: &RiskCondition{
					Field:    "trajectory_smoothness",
					Operator: RiskOperatorGt,
					Value:    0.95,
				},
				Tags:      []string{"trajectory", "pattern"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "path_complexity_check",
				Name:        "Path Complexity Check",
				Description: "Check if path is too simple",
				Type:        RiskRuleTypeCondition,
				Priority:    80,
				Weight:      0.2,
				Enabled:     true,
				Condition: &RiskCondition{
					Field:    "path_complexity",
					Operator: RiskOperatorLt,
					Value:    0.3,
				},
				Tags:      []string{"path", "pattern"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "acceleration_check",
				Name:        "Acceleration Check",
				Description: "Check for unusual acceleration patterns",
				Type:        RiskRuleTypeCondition,
				Priority:    85,
				Weight:      0.15,
				Enabled:     true,
				Condition: &RiskCondition{
					Field:    "acceleration",
					Operator: RiskOperatorLt,
					Value:    0.1,
				},
				Tags:      []string{"acceleration", "physics"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
			{
				ID:          "combined_bot_detection",
				Name:        "Combined Bot Detection",
				Description: "Detect bot using combined conditions",
				Type:        RiskRuleTypeGroup,
				Priority:    95,
				Weight:      0.4,
				Enabled:     true,
				Group: &RiskGroup{
					Operator: RiskLogicAnd,
					Children: []RiskExpression{
						{
							ID:       "combined_speed",
							Name:     "High Speed",
							Type:     RiskRuleTypeCondition,
							Priority: 100,
							Condition: &RiskCondition{
								Field:    "speed",
								Operator: RiskOperatorGt,
								Value:    1000,
							},
						},
						{
							ID:       "combined_smooth",
							Name:     "Too Smooth",
							Type:     RiskRuleTypeCondition,
							Priority: 90,
							Condition: &RiskCondition{
								Field:    "trajectory_smoothness",
								Operator: RiskOperatorGt,
								Value:    0.9,
							},
						},
					},
				},
				Tags:      []string{"bot", "combined"},
				CreatedAt: time.Now(),
				UpdatedAt: time.Now(),
			},
		},
		DefaultScore: 10,
		Threshold:    50,
		Enabled:      true,
		Timeout:      100 * time.Millisecond,
		CreatedAt:    time.Now(),
		UpdatedAt:    time.Now(),
	}
}

func (rm *RiskManager) GetEngine(appID string) (*RiskEngine, error) {
	rm.mu.RLock()
	if engine, ok := rm.engineCache[appID]; ok {
		rm.mu.RUnlock()
		return engine, nil
	}
	rm.mu.RUnlock()

	config, err := rm.LoadConfig(appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = rm.defaultConfig
		} else {
			return nil, err
		}
	}

	engine := NewRiskEngine(config)

	rm.mu.Lock()
	rm.engineCache[appID] = engine
	rm.mu.Unlock()

	return engine, nil
}

func (rm *RiskManager) LoadConfig(appID string) (*RiskConfig, error) {
	var ruleConfig models.RuleConfig
	if err := rm.db.Where("app_id = ?", appID).First(&ruleConfig).Error; err != nil {
		return nil, err
	}

	var config RiskConfig
	if err := json.Unmarshal([]byte(ruleConfig.Config), &config); err != nil {
		return nil, err
	}

	return &config, nil
}

func (rm *RiskManager) SaveConfig(appID string, config *RiskConfig) error {
	configBytes, err := json.Marshal(config)
	if err != nil {
		return err
	}

	now := time.Now()

	var existingConfig models.RuleConfig
	err = rm.db.Where("app_id = ?", appID).First(&existingConfig).Error

	if errors.Is(err, gorm.ErrRecordNotFound) {
		ruleConfig := models.RuleConfig{
			AppID:     appID,
			Config:    string(configBytes),
			CreatedAt: now,
			UpdatedAt: now,
		}
		return rm.db.Create(&ruleConfig).Error
	} else if err != nil {
		return err
	}

	existingConfig.Config = string(configBytes)
	existingConfig.UpdatedAt = now
	return rm.db.Save(&existingConfig).Error
}

func (rm *RiskManager) CreateRule(appID string, rule *RiskExpression) error {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = rm.defaultConfig
		} else {
			return err
		}
	}

	rule.ID = rm.generateRuleID()
	rule.CreatedAt = time.Now()
	rule.UpdatedAt = time.Now()

	if err := rm.compileAndValidate(rule); err != nil {
		return err
	}

	config.Expressions = append(config.Expressions, *rule)
	config.Version++
	config.UpdatedAt = time.Now()

	if err := rm.SaveConfig(appID, config); err != nil {
		return err
	}

	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()

	return nil
}

func (rm *RiskManager) UpdateRule(appID, ruleID string, rule *RiskExpression) error {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		return err
	}

	found := false
	for i, expr := range config.Expressions {
		if expr.ID == ruleID {
			rule.ID = ruleID
			rule.CreatedAt = expr.CreatedAt
			rule.UpdatedAt = time.Now()

			if err := rm.compileAndValidate(rule); err != nil {
				return err
			}

			config.Expressions[i] = *rule
			found = true
			break
		}
	}

	if !found {
		return errors.New("rule not found")
	}

	config.Version++
	config.UpdatedAt = time.Now()

	if err := rm.SaveConfig(appID, config); err != nil {
		return err
	}

	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()

	return nil
}

func (rm *RiskManager) DeleteRule(appID, ruleID string) error {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		return err
	}

	found := false
	newExpressions := make([]RiskExpression, 0, len(config.Expressions))
	for _, expr := range config.Expressions {
		if expr.ID == ruleID {
			found = true
		} else {
			newExpressions = append(newExpressions, expr)
		}
	}

	if !found {
		return errors.New("rule not found")
	}

	config.Expressions = newExpressions
	config.Version++
	config.UpdatedAt = time.Now()

	if err := rm.SaveConfig(appID, config); err != nil {
		return err
	}

	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()

	return nil
}

func (rm *RiskManager) GetRule(appID, ruleID string) (*RiskExpression, error) {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		return nil, err
	}

	for _, expr := range config.Expressions {
		if expr.ID == ruleID {
			return &expr, nil
		}
	}

	return nil, errors.New("rule not found")
}

func (rm *RiskManager) ListRules(appID string) ([]RiskExpression, error) {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return rm.defaultConfig.Expressions, nil
		}
		return nil, err
	}

	return config.Expressions, nil
}

func (rm *RiskManager) EnableRule(appID, ruleID string) error {
	rule, err := rm.GetRule(appID, ruleID)
	if err != nil {
		return err
	}

	rule.Enabled = true
	return rm.UpdateRule(appID, ruleID, rule)
}

func (rm *RiskManager) DisableRule(appID, ruleID string) error {
	rule, err := rm.GetRule(appID, ruleID)
	if err != nil {
		return err
	}

	rule.Enabled = false
	return rm.UpdateRule(appID, ruleID, rule)
}

func (rm *RiskManager) EvaluateRisk(appID string, data map[string]interface{}) (*RiskAssessmentResult, error) {
	engine, err := rm.GetEngine(appID)
	if err != nil {
		return nil, err
	}

	ctx := &RiskContext{
		Data: data,
		Meta: map[string]interface{}{
			"app_id":       appID,
			"evaluated_at": time.Now(),
		},
	}

	return engine.Evaluate(ctx)
}

func (rm *RiskManager) compileAndValidate(rule *RiskExpression) error {
	tempEngine := &RiskEngine{}
	return tempEngine.CompileRule(rule)
}

func (rm *RiskManager) InvalidateCache(appID string) {
	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()
}

func (rm *RiskManager) InvalidateAllCache() {
	rm.mu.Lock()
	rm.engineCache = make(map[string]*RiskEngine)
	rm.mu.Unlock()
}

func (rm *RiskManager) GetDefaultConfig() *RiskConfig {
	return rm.defaultConfig
}

func (rm *RiskManager) ImportConfig(appID string, config *RiskConfig) error {
	for _, expr := range config.Expressions {
		if err := rm.compileAndValidate(&expr); err != nil {
			return fmt.Errorf("invalid rule %s: %w", expr.ID, err)
		}
	}

	config.UpdatedAt = time.Now()
	return rm.SaveConfig(appID, config)
}

func (rm *RiskManager) ExportConfig(appID string) (*RiskConfig, error) {
	return rm.LoadConfig(appID)
}

func (rm *RiskManager) ValidateConfig(config *RiskConfig) ([]string, error) {
	validationErrors := make([]string, 0)

	for _, expr := range config.Expressions {
		tempEngine := &RiskEngine{}
		if err := tempEngine.CompileRule(&expr); err != nil {
			validationErrors = append(validationErrors, fmt.Sprintf("rule %s: %v", expr.ID, err))
		}
	}

	if len(validationErrors) > 0 {
		return validationErrors, fmt.Errorf("validation failed: %s", validationErrors[0])
	}

	return nil, nil
}

func (rm *RiskManager) GetStatistics(appID string) (map[string]interface{}, error) {
	engine, err := rm.GetEngine(appID)
	if err != nil {
		return nil, err
	}

	stats := engine.GetMetrics()

	config, err := rm.LoadConfig(appID)
	if err != nil {
		if !errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, err
		}
		config = rm.defaultConfig
	}

	stats["total_rules"] = len(config.Expressions)
	stats["enabled_rules"] = countEnabledRules(config.Expressions)
	stats["version"] = config.Version
	stats["last_updated"] = config.UpdatedAt

	return stats, nil
}

func countEnabledRules(expressions []RiskExpression) int {
	count := 0
	for _, expr := range expressions {
		if expr.Enabled {
			count++
		}
	}
	return count
}

func (rm *RiskManager) generateRuleID() string {
	return fmt.Sprintf("rule_%d_%d", time.Now().UnixNano(), time.Now().UnixNano()%10000)
}

func (rm *RiskManager) BatchCreateRules(appID string, rules []RiskExpression) error {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			config = rm.defaultConfig
		} else {
			return err
		}
	}

	now := time.Now()

	for i := range rules {
		rule := &rules[i]
		rule.ID = rm.generateRuleID()
		rule.CreatedAt = now
		rule.UpdatedAt = now

		if err := rm.compileAndValidate(rule); err != nil {
			return fmt.Errorf("invalid rule %s: %w", rule.Name, err)
		}

		config.Expressions = append(config.Expressions, *rule)
	}

	config.Version++
	config.UpdatedAt = now

	if err := rm.SaveConfig(appID, config); err != nil {
		return err
	}

	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()

	return nil
}

func (rm *RiskManager) BatchUpdateRules(appID string, rules []RiskExpression) error {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		return err
	}

	now := time.Now()

	for _, updatedRule := range rules {
		found := false
		for i, expr := range config.Expressions {
			if expr.ID == updatedRule.ID {
				updatedRule.CreatedAt = expr.CreatedAt
				updatedRule.UpdatedAt = now

				if err := rm.compileAndValidate(&updatedRule); err != nil {
					return fmt.Errorf("invalid rule %s: %w", updatedRule.ID, err)
				}

				config.Expressions[i] = updatedRule
				found = true
				break
			}
		}

		if !found {
			return fmt.Errorf("rule %s not found", updatedRule.ID)
		}
	}

	config.Version++
	config.UpdatedAt = now

	if err := rm.SaveConfig(appID, config); err != nil {
		return err
	}

	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()

	return nil
}

func (rm *RiskManager) BatchDeleteRules(appID string, ruleIDs []string) error {
	config, err := rm.LoadConfig(appID)
	if err != nil {
		return err
	}

	ruleIDSet := make(map[string]bool)
	for _, id := range ruleIDs {
		ruleIDSet[id] = true
	}

	newExpressions := make([]RiskExpression, 0)
	for _, expr := range config.Expressions {
		if !ruleIDSet[expr.ID] {
			newExpressions = append(newExpressions, expr)
		}
	}

	if len(newExpressions) == len(config.Expressions) {
		return errors.New("no rules deleted, IDs not found")
	}

	config.Expressions = newExpressions
	config.Version++
	config.UpdatedAt = time.Now()

	if err := rm.SaveConfig(appID, config); err != nil {
		return err
	}

	rm.mu.Lock()
	delete(rm.engineCache, appID)
	rm.mu.Unlock()

	return nil
}
