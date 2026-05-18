package service

import (
	"encoding/json"
	"fmt"
	"sync"
)

// LogicOperator 逻辑运算符类型
type LogicOperator string

const (
	OperatorAND LogicOperator = "AND"
	OperatorOR  LogicOperator = "OR"
	OperatorNOT LogicOperator = "NOT"
	OperatorXOR LogicOperator = "XOR"
)

// RuleCondition 规则条件
type RuleCondition struct {
	ID       string                 `json:"id"`
	Field    string                 `json:"field"`
	Operator string                 `json:"operator"`
	Value    interface{}            `json:"value"`
	Params   map[string]interface{} `json:"params,omitempty"`
}

// RuleGroup 规则组（支持嵌套）
type RuleGroup struct {
	ID       string          `json:"id"`
	Operator LogicOperator   `json:"operator"`
	Conditions []interface{} `json:"conditions"` // RuleCondition 或 RuleGroup
}

// CombinedRule 组合规则
type CombinedRule struct {
	ID          string     `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	RootGroup   *RuleGroup `json:"root_group"`
	Weight      float64    `json:"weight"`
	Severity    float64    `json:"severity"`
	Enabled     bool       `json:"enabled"`
}

// RuleCombinator 规则组合器
type RuleCombinator struct {
	rules      map[string]*CombinedRule
	ruleGroups map[string]*RuleGroup
	mu         sync.RWMutex
}

func NewRuleCombinator() *RuleCombinator {
	return &RuleCombinator{
		rules:      make(map[string]*CombinedRule),
		ruleGroups: make(map[string]*RuleGroup),
	}
}

// AddCombinedRule 添加组合规则
func (rc *RuleCombinator) AddCombinedRule(rule *CombinedRule) error {
	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}
	if rule.RootGroup == nil {
		return fmt.Errorf("规则必须包含根规则组")
	}

	rc.mu.Lock()
	defer rc.mu.Unlock()

	rc.rules[rule.ID] = rule
	return nil
}

// RemoveCombinedRule 移除组合规则
func (rc *RuleCombinator) RemoveCombinedRule(ruleID string) error {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if _, exists := rc.rules[ruleID]; !exists {
		return fmt.Errorf("规则不存在: %s", ruleID)
	}

	delete(rc.rules, ruleID)
	return nil
}

// GetCombinedRule 获取组合规则
func (rc *RuleCombinator) GetCombinedRule(ruleID string) (*CombinedRule, error) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	rule, exists := rc.rules[ruleID]
	if !exists {
		return nil, fmt.Errorf("规则不存在: %s", ruleID)
	}
	return rule, nil
}

// GetAllCombinedRules 获取所有组合规则
func (rc *RuleCombinator) GetAllCombinedRules() []*CombinedRule {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	rules := make([]*CombinedRule, 0, len(rc.rules))
	for _, rule := range rc.rules {
		rules = append(rules, rule)
	}
	return rules
}

// EvaluateRule 评估组合规则
func (rc *RuleCombinator) EvaluateRule(ruleID string, features map[string]interface{}) (bool, error) {
	rule, err := rc.GetCombinedRule(ruleID)
	if err != nil {
		return false, err
	}

	if !rule.Enabled {
		return false, nil
	}

	return rc.evaluateGroup(rule.RootGroup, features), nil
}

// evaluateGroup 评估规则组
func (rc *RuleCombinator) evaluateGroup(group *RuleGroup, features map[string]interface{}) bool {
	if group == nil || len(group.Conditions) == 0 {
		return false
	}

	results := make([]bool, 0, len(group.Conditions))
	for _, cond := range group.Conditions {
		switch v := cond.(type) {
		case map[string]interface{}:
			// 尝试解析为 RuleCondition 或 RuleGroup
			if operator, ok := v["operator"].(string); ok && isLogicOperator(operator) {
				// 是 RuleGroup
				subGroup := &RuleGroup{
					ID:       getStringValue(v, "id"),
					Operator: LogicOperator(operator),
				}
				if conditions, ok := v["conditions"].([]interface{}); ok {
					subGroup.Conditions = conditions
				}
				results = append(results, rc.evaluateGroup(subGroup, features))
			} else {
				// 是 RuleCondition
				condition := &RuleCondition{
					ID:       getStringValue(v, "id"),
					Field:    getStringValue(v, "field"),
					Operator: getStringValue(v, "operator"),
					Value:    v["value"],
				}
				if params, ok := v["params"].(map[string]interface{}); ok {
					condition.Params = params
				}
				results = append(results, rc.evaluateCondition(condition, features))
			}
		default:
			results = append(results, false)
		}
	}

	return rc.applyOperator(group.Operator, results)
}

// evaluateCondition 评估单个条件
func (rc *RuleCombinator) evaluateCondition(condition *RuleCondition, features map[string]interface{}) bool {
	if condition.Field == "" || condition.Operator == "" {
		return false
	}

	featureValue, exists := features[condition.Field]
	if !exists {
		return false
	}

	return rc.compareValues(featureValue, condition.Operator, condition.Value)
}

// compareValues 比较值
func (rc *RuleCombinator) compareValues(left interface{}, operator string, right interface{}) bool {
	leftFloat := toFloat(left)
	rightFloat := toFloat(right)

	switch operator {
	case "gt":
		return leftFloat > rightFloat
	case "gte":
		return leftFloat >= rightFloat
	case "lt":
		return leftFloat < rightFloat
	case "lte":
		return leftFloat <= rightFloat
	case "eq":
		return leftFloat == rightFloat
	case "neq":
		return leftFloat != rightFloat
	case "contains":
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return containsString(leftStr, rightStr)
	case "regex":
		leftStr := fmt.Sprintf("%v", left)
		rightStr := fmt.Sprintf("%v", right)
		return matchRegex(leftStr, rightStr)
	default:
		return false
	}
}

// applyOperator 应用逻辑运算符
func (rc *RuleCombinator) applyOperator(operator LogicOperator, results []bool) bool {
	if len(results) == 0 {
		return false
	}

	switch operator {
	case OperatorAND:
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true

	case OperatorOR:
		for _, r := range results {
			if r {
				return true
			}
		}
		return false

	case OperatorNOT:
		if len(results) > 0 {
			return !results[0]
		}
		return false

	case OperatorXOR:
		trueCount := 0
		for _, r := range results {
			if r {
				trueCount++
			}
		}
		return trueCount%2 == 1

	default:
		return false
	}
}

// isLogicOperator 判断是否为逻辑运算符
func isLogicOperator(op string) bool {
	switch LogicOperator(op) {
	case OperatorAND, OperatorOR, OperatorNOT, OperatorXOR:
		return true
	default:
		return false
	}
}

// toFloat 转换为float64
func toFloat(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case int32:
		return float64(val)
	case string:
		var f float64
		fmt.Sscanf(val, "%f", &f)
		return f
	default:
		return 0
	}
}

// containsString 检查字符串包含
func containsString(str, substr string) bool {
	return len(str) >= len(substr) && (str == substr || len(str) > 0 && containsStringHelper(str, substr))
}

func containsStringHelper(str, substr string) bool {
	for i := 0; i <= len(str)-len(substr); i++ {
		if str[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

// matchRegex 简单的正则匹配（简化实现）
func matchRegex(str, pattern string) bool {
	if pattern == "" || str == "" {
		return str == pattern
	}
	if pattern == "*" {
		return true
	}
	if pattern[0] == '*' && pattern[len(pattern)-1] == '*' {
		return containsString(str, pattern[1:len(pattern)-1])
	}
	if pattern[0] == '*' {
		return len(str) >= len(pattern)-1 && str[len(str)-len(pattern)+1:] == pattern[1:]
	}
	if pattern[len(pattern)-1] == '*' {
		return len(str) >= len(pattern)-1 && str[:len(pattern)-1] == pattern[:len(pattern)-1]
	}
	return str == pattern
}

// getStringValue 获取map中的字符串值
func getStringValue(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

// ExportRuleToJSON 导出规则为JSON
func (rc *RuleCombinator) ExportRuleToJSON(ruleID string) (string, error) {
	rule, err := rc.GetCombinedRule(ruleID)
	if err != nil {
		return "", err
	}

	data, err := json.MarshalIndent(rule, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportRuleFromJSON 从JSON导入规则
func (rc *RuleCombinator) ImportRuleFromJSON(jsonData string) error {
	var rule CombinedRule
	if err := json.Unmarshal([]byte(jsonData), &rule); err != nil {
		return err
	}
	return rc.AddCombinedRule(&rule)
}

// ValidateRule 验证规则结构
func (rc *RuleCombinator) ValidateRule(rule *CombinedRule) []string {
	var errors []string

	if rule.ID == "" {
		errors = append(errors, "规则ID不能为空")
	}
	if rule.Name == "" {
		errors = append(errors, "规则名称不能为空")
	}
	if rule.RootGroup == nil {
		errors = append(errors, "规则必须包含根规则组")
	} else {
		errs := rc.validateGroup(rule.RootGroup)
		errors = append(errors, errs...)
	}
	if rule.Weight < 0 {
		errors = append(errors, "规则权重不能为负数")
	}
	if rule.Severity < 0 || rule.Severity > 1 {
		errors = append(errors, "规则严重度必须在0-1之间")
	}

	return errors
}

// validateGroup 验证规则组
func (rc *RuleCombinator) validateGroup(group *RuleGroup) []string {
	var errors []string

	if group.Operator == "" {
		errors = append(errors, "规则组必须指定逻辑运算符")
	}
	if !isLogicOperator(string(group.Operator)) {
		errors = append(errors, fmt.Sprintf("无效的逻辑运算符: %s", group.Operator))
	}
	if len(group.Conditions) == 0 {
		errors = append(errors, "规则组必须包含至少一个条件")
	}

	for i, cond := range group.Conditions {
		switch v := cond.(type) {
		case map[string]interface{}:
			if operator, ok := v["operator"].(string); ok && isLogicOperator(operator) {
				subGroup := &RuleGroup{
					Operator: LogicOperator(operator),
				}
				if conditions, ok := v["conditions"].([]interface{}); ok {
					subGroup.Conditions = conditions
				}
				errs := rc.validateGroup(subGroup)
				errors = append(errors, errs...)
			} else {
				errs := rc.validateCondition(v)
				for _, err := range errs {
					errors = append(errors, fmt.Sprintf("条件[%d]: %s", i, err))
				}
			}
		}
	}

	return errors
}

// validateCondition 验证条件
func (rc *RuleCombinator) validateCondition(cond map[string]interface{}) []string {
	var errors []string

	field, _ := cond["field"].(string)
	operator, _ := cond["operator"].(string)

	if field == "" {
		errors = append(errors, "字段不能为空")
	}
	if operator == "" {
		errors = append(errors, "运算符不能为空")
	} else if !isComparisonOperator(operator) {
		errors = append(errors, fmt.Sprintf("无效的比较运算符: %s", operator))
	}
	if _, exists := cond["value"]; !exists {
		errors = append(errors, "条件值不能为空")
	}

	return errors
}

// isComparisonOperator 判断是否为比较运算符
func isComparisonOperator(op string) bool {
	switch op {
	case "gt", "gte", "lt", "lte", "eq", "neq", "contains", "regex":
		return true
	default:
		return false
	}
}