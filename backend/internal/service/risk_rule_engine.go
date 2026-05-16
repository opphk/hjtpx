package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type RiskRuleType string
type RiskOperatorType string
type RiskLogicOperator string

const (
	RiskRuleTypeCondition RiskRuleType = "condition"
	RiskRuleTypeGroup     RiskRuleType = "group"
	RiskRuleTypeScript    RiskRuleType = "script"

	RiskOperatorEq         RiskOperatorType = "eq"
	RiskOperatorNe         RiskOperatorType = "ne"
	RiskOperatorGt         RiskOperatorType = "gt"
	RiskOperatorGte        RiskOperatorType = "gte"
	RiskOperatorLt         RiskOperatorType = "lt"
	RiskOperatorLte        RiskOperatorType = "lte"
	RiskOperatorContains   RiskOperatorType = "contains"
	RiskOperatorNotContain RiskOperatorType = "not_contains"
	RiskOperatorIn         RiskOperatorType = "in"
	RiskOperatorNotIn      RiskOperatorType = "not_in"
	RiskOperatorRegex      RiskOperatorType = "regex"
	RiskOperatorStartsWith RiskOperatorType = "starts_with"
	RiskOperatorEndsWith   RiskOperatorType = "ends_with"
	RiskOperatorBetween    RiskOperatorType = "between"
	RiskOperatorIsEmpty    RiskOperatorType = "is_empty"
	RiskOperatorIsNotEmpty RiskOperatorType = "is_not_empty"

	RiskLogicAnd RiskLogicOperator = "AND"
	RiskLogicOr  RiskLogicOperator = "OR"
	RiskLogicNot RiskLogicOperator = "NOT"
)

type RiskCondition struct {
	Field    string           `json:"field"`
	Operator RiskOperatorType  `json:"operator"`
	Value    interface{}      `json:"value"`
	Values   []interface{}    `json:"values,omitempty"`
}

type RiskGroup struct {
	Operator RiskLogicOperator   `json:"operator"`
	Children []RiskExpression `json:"children"`
}

type RiskExpression struct {
	ID          string            `json:"id"`
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Type        RiskRuleType      `json:"type"`
	Priority    int               `json:"priority"`
	Weight      float64           `json:"weight"`
	Enabled     bool              `json:"enabled"`
	Condition   *RiskCondition   `json:"condition,omitempty"`
	Group       *RiskGroup       `json:"group,omitempty"`
	Script      string            `json:"script,omitempty"`
	Tags        []string          `json:"tags,omitempty"`
	CreatedAt   time.Time         `json:"created_at"`
	UpdatedAt   time.Time         `json:"updated_at"`
}

type RiskConfig struct {
	ID            string           `json:"id"`
	Name          string           `json:"name"`
	Description   string           `json:"description,omitempty"`
	Version       int              `json:"version"`
	Expressions   []RiskExpression `json:"expressions"`
	DefaultScore  float64          `json:"default_score"`
	Threshold     float64          `json:"threshold"`
	Enabled       bool             `json:"enabled"`
	Timeout       time.Duration     `json:"timeout"`
	CreatedAt     time.Time        `json:"created_at"`
	UpdatedAt     time.Time        `json:"updated_at"`
}

type RiskContext struct {
	Data map[string]interface{}
	Meta map[string]interface{}
}

type RiskResult struct {
	RuleID      string                 `json:"rule_id"`
	RuleName    string                 `json:"rule_name"`
	Triggered   bool                   `json:"triggered"`
	Score       float64                `json:"score"`
	Details     string                 `json:"details,omitempty"`
	EvaluatedAt time.Time              `json:"evaluated_at"`
}

type RiskAssessmentResult struct {
	TotalScore     float64            `json:"total_score"`
	RiskLevel      string             `json:"risk_level"`
	TriggeredRules []RiskResult       `json:"triggered_rules"`
	Context        *RiskContext        `json:"context,omitempty"`
	EvaluatedAt    time.Time          `json:"evaluated_at"`
	Duration       time.Duration      `json:"duration"`
}

type RiskEngine struct {
	config      *RiskConfig
	compiled    map[string]*RiskExpression
	mu          sync.RWMutex
	timeout     time.Duration
	hitCount    int64
	missCount   int64
	evalTime    time.Duration
	evalMutex   sync.Mutex
}

func NewRiskEngine(config *RiskConfig) *RiskEngine {
	engine := &RiskEngine{
		config:   config,
		compiled: make(map[string]*RiskExpression),
		timeout:  config.Timeout,
	}

	if engine.timeout == 0 {
		engine.timeout = 100 * time.Millisecond
	}

	engine.compileRules()
	return engine
}

func (re *RiskEngine) compileRules() {
	re.mu.Lock()
	defer re.mu.Unlock()

	for i := range re.config.Expressions {
		expr := &re.config.Expressions[i]
		if expr.Enabled {
			re.compiled[expr.ID] = expr
		}
	}
}

func (re *RiskEngine) CompileRule(expr *RiskExpression) error {
	if expr == nil {
		return errors.New("rule expression cannot be nil")
	}

	switch expr.Type {
	case RiskRuleTypeCondition:
		if expr.Condition == nil {
			return fmt.Errorf("condition rule %s requires condition", expr.ID)
		}
		if expr.Condition.Field == "" {
			return fmt.Errorf("condition rule %s requires field", expr.ID)
		}
		if !isValidRiskOperator(expr.Condition.Operator) {
			return fmt.Errorf("invalid operator %s for rule %s", expr.Condition.Operator, expr.ID)
		}

	case RiskRuleTypeGroup:
		if expr.Group == nil {
			return fmt.Errorf("group rule %s requires group", expr.ID)
		}
		if len(expr.Group.Children) == 0 {
			return fmt.Errorf("group rule %s requires at least one child", expr.ID)
		}
		for _, child := range expr.Group.Children {
			if err := re.CompileRule(&child); err != nil {
				return err
			}
		}

	case RiskRuleTypeScript:
		if expr.Script == "" {
			return fmt.Errorf("script rule %s requires script", expr.ID)
		}
		_, err := regexp.Compile(expr.Script)
		if err != nil {
			return fmt.Errorf("invalid script regex for rule %s: %v", expr.ID, err)
		}

	default:
		return fmt.Errorf("unknown rule type %s for rule %s", expr.Type, expr.ID)
	}

	return nil
}

func isValidRiskOperator(op RiskOperatorType) bool {
	validOps := []RiskOperatorType{
		RiskOperatorEq, RiskOperatorNe, RiskOperatorGt, RiskOperatorGte,
		RiskOperatorLt, RiskOperatorLte, RiskOperatorContains, RiskOperatorNotContain,
		RiskOperatorIn, RiskOperatorNotIn, RiskOperatorRegex, RiskOperatorStartsWith,
		RiskOperatorEndsWith, RiskOperatorBetween, RiskOperatorIsEmpty, RiskOperatorIsNotEmpty,
	}
	for _, valid := range validOps {
		if op == valid {
			return true
		}
	}
	return false
}

func (re *RiskEngine) Evaluate(ctx *RiskContext) (*RiskAssessmentResult, error) {
	startTime := time.Now()

	if ctx == nil {
		ctx = &RiskContext{
			Data: make(map[string]interface{}),
			Meta: make(map[string]interface{}),
		}
	}

	if ctx.Data == nil {
		ctx.Data = make(map[string]interface{})
	}
	if ctx.Meta == nil {
		ctx.Meta = make(map[string]interface{})
	}

	results := make([]RiskResult, 0)
	totalScore := 0.0
	totalWeight := 0.0

	exprs := make([]*RiskExpression, 0)
	re.mu.RLock()
	for _, expr := range re.compiled {
		exprs = append(exprs, expr)
	}
	re.mu.RUnlock()

	sortRiskRulesByPriority(exprs)

	timeoutCh := time.After(re.timeout)
	doneCh := make(chan struct{})

	go func() {
		for _, expr := range exprs {
			result := re.evaluateExpression(expr, ctx)
			if result.Triggered {
				results = append(results, result)
				totalScore += result.Score * expr.Weight
				totalWeight += expr.Weight
			}
		}
		close(doneCh)
	}()

	select {
	case <-timeoutCh:
		return &RiskAssessmentResult{
			TotalScore:     re.config.DefaultScore,
			RiskLevel:      "timeout",
			TriggeredRules: results,
			Context:        ctx,
			EvaluatedAt:    time.Now(),
			Duration:       time.Since(startTime),
		}, errors.New("evaluation timeout")
	case <-doneCh:
	}

	if totalWeight > 0 {
		totalScore = totalScore / totalWeight
	} else {
		totalScore = re.config.DefaultScore
	}

	if totalScore > 100 {
		totalScore = 100
	}

	riskLevel := re.calculateRiskLevel(totalScore)

	assessment := &RiskAssessmentResult{
		TotalScore:     totalScore,
		RiskLevel:      riskLevel,
		TriggeredRules: results,
		Context:        ctx,
		EvaluatedAt:    time.Now(),
		Duration:       time.Since(startTime),
	}

	re.recordMetrics(assessment.Duration)

	return assessment, nil
}

func (re *RiskEngine) evaluateExpression(expr *RiskExpression, ctx *RiskContext) RiskResult {
	result := RiskResult{
		RuleID:      expr.ID,
		RuleName:    expr.Name,
		Triggered:   false,
		Score:       0,
		EvaluatedAt: time.Now(),
	}

	var triggered bool
	var details string

	switch expr.Type {
	case RiskRuleTypeCondition:
		triggered, details = re.evaluateCondition(expr.Condition, ctx)
	case RiskRuleTypeGroup:
		triggered, details = re.evaluateGroup(expr.Group, ctx)
	case RiskRuleTypeScript:
		triggered, details = re.evaluateScript(expr.Script, ctx)
	default:
		triggered = false
		details = "unknown rule type"
	}

	result.Triggered = triggered
	result.Details = details

	if triggered {
		result.Score = expr.Weight * 100
	}

	return result
}

func (re *RiskEngine) evaluateCondition(cond *RiskCondition, ctx *RiskContext) (bool, string) {
	value, exists := getNestedRiskValue(ctx.Data, cond.Field)

	if !exists {
		return false, fmt.Sprintf("field %s not found", cond.Field)
	}

	var result bool
	var err error

	switch cond.Operator {
	case RiskOperatorEq:
		result = compareRiskEqual(value, cond.Value)
	case RiskOperatorNe:
		result = !compareRiskEqual(value, cond.Value)
	case RiskOperatorGt:
		result, err = compareRiskNumeric(value, cond.Value, "gt")
	case RiskOperatorGte:
		result, err = compareRiskNumeric(value, cond.Value, "gte")
	case RiskOperatorLt:
		result, err = compareRiskNumeric(value, cond.Value, "lt")
	case RiskOperatorLte:
		result, err = compareRiskNumeric(value, cond.Value, "lte")
	case RiskOperatorContains:
		result = stringRiskContains(value, cond.Value)
	case RiskOperatorNotContain:
		result = !stringRiskContains(value, cond.Value)
	case RiskOperatorIn:
		result = valueInRiskList(value, cond.Values)
	case RiskOperatorNotIn:
		result = !valueInRiskList(value, cond.Values)
	case RiskOperatorRegex:
		result = regexRiskMatch(value, cond.Value)
	case RiskOperatorStartsWith:
		result = strings.HasPrefix(toRiskString(value), toRiskString(cond.Value))
	case RiskOperatorEndsWith:
		result = strings.HasSuffix(toRiskString(value), toRiskString(cond.Value))
	case RiskOperatorBetween:
		if len(cond.Values) >= 2 {
			result, err = betweenRiskCheck(value, cond.Values[0], cond.Values[1])
		}
	case RiskOperatorIsEmpty:
		result = isRiskEmpty(value)
	case RiskOperatorIsNotEmpty:
		result = !isRiskEmpty(value)
	}

	if err != nil {
		return false, fmt.Sprintf("evaluation error: %v", err)
	}

	return result, fmt.Sprintf("field=%s operator=%s value=%v", cond.Field, cond.Operator, cond.Value)
}

func (re *RiskEngine) evaluateGroup(group *RiskGroup, ctx *RiskContext) (bool, string) {
	if len(group.Children) == 0 {
		return false, "empty group"
	}

	switch group.Operator {
	case RiskLogicAnd:
		for _, child := range group.Children {
			result := re.evaluateExpression(&child, ctx)
			if !result.Triggered {
				return false, fmt.Sprintf("AND failed at %s", child.Name)
			}
		}
		return true, "all AND conditions passed"

	case RiskLogicOr:
		for _, child := range group.Children {
			result := re.evaluateExpression(&child, ctx)
			if result.Triggered {
				return true, fmt.Sprintf("OR passed at %s", child.Name)
			}
		}
		return false, "no OR condition passed"

	case RiskLogicNot:
		if len(group.Children) == 0 {
			return false, "NOT requires at least one child"
		}
		result := re.evaluateExpression(&group.Children[0], ctx)
		return !result.Triggered, fmt.Sprintf("NOT %s", group.Children[0].Name)

	default:
		return false, fmt.Sprintf("unknown operator %s", group.Operator)
	}
}

func (re *RiskEngine) evaluateScript(script string, ctx *RiskContext) (bool, string) {
	matches, err := regexp.MatchString(script, fmt.Sprintf("%v", ctx.Data))
	if err != nil {
		return false, fmt.Sprintf("script error: %v", err)
	}
	return matches, "script matched"
}

func (re *RiskEngine) calculateRiskLevel(score float64) string {
	switch {
	case score >= 80:
		return "critical"
	case score >= 60:
		return "high"
	case score >= 40:
		return "medium"
	case score >= 20:
		return "low"
	default:
		return "minimal"
	}
}

func (re *RiskEngine) recordMetrics(duration time.Duration) {
	re.evalMutex.Lock()
	defer re.evalMutex.Unlock()
	re.evalTime += duration
}

func (re *RiskEngine) GetMetrics() map[string]interface{} {
	re.evalMutex.Lock()
	defer re.evalMutex.Unlock()

	avgEvalTime := time.Duration(0)
	totalEvals := re.hitCount + re.missCount
	if totalEvals > 0 {
		avgEvalTime = re.evalTime / time.Duration(totalEvals)
	}

	return map[string]interface{}{
		"hit_count":        re.hitCount,
		"miss_count":       re.missCount,
		"total_evaluations": totalEvals,
		"total_eval_time":  re.evalTime,
		"avg_eval_time":    avgEvalTime,
	}
}

func sortRiskRulesByPriority(rules []*RiskExpression) {
	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}
}

func getNestedRiskValue(data map[string]interface{}, field string) (interface{}, bool) {
	parts := strings.Split(field, ".")
	current := interface{}(data)

	for _, part := range parts {
		switch c := current.(type) {
		case map[string]interface{}:
			if val, ok := c[part]; ok {
				current = val
			} else {
				return nil, false
			}
		default:
			return nil, false
		}
	}

	return current, true
}

func compareRiskEqual(a, b interface{}) bool {
	aStr := fmt.Sprintf("%v", a)
	bStr := fmt.Sprintf("%v", b)
	return aStr == bStr
}

func compareRiskNumeric(a, b interface{}, op string) (bool, error) {
	aNum, err := toRiskFloat64(a)
	if err != nil {
		return false, err
	}

	bNum, err := toRiskFloat64(b)
	if err != nil {
		return false, err
	}

	switch op {
	case "gt":
		return aNum > bNum, nil
	case "gte":
		return aNum >= bNum, nil
	case "lt":
		return aNum < bNum, nil
	case "lte":
		return aNum <= bNum, nil
	default:
		return false, errors.New("unknown numeric operator")
	}
}

func toRiskFloat64(v interface{}) (float64, error) {
	switch n := v.(type) {
	case float64:
		return n, nil
	case float32:
		return float64(n), nil
	case int:
		return float64(n), nil
	case int64:
		return float64(n), nil
	case int32:
		return float64(n), nil
	case json.Number:
		return n.Float64()
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func stringRiskContains(value, substr interface{}) bool {
	return strings.Contains(toRiskString(value), toRiskString(substr))
}

func toRiskString(v interface{}) string {
	return fmt.Sprintf("%v", v)
}

func valueInRiskList(value interface{}, list []interface{}) bool {
	valueStr := toRiskString(value)
	for _, item := range list {
		if toRiskString(item) == valueStr {
			return true
		}
	}
	return false
}

func regexRiskMatch(value, pattern interface{}) bool {
	re, err := regexp.Compile(toRiskString(pattern))
	if err != nil {
		return false
	}
	return re.MatchString(toRiskString(value))
}

func betweenRiskCheck(value, min, max interface{}) (bool, error) {
	val, err := toRiskFloat64(value)
	if err != nil {
		return false, err
	}
	minVal, err := toRiskFloat64(min)
	if err != nil {
		return false, err
	}
	maxVal, err := toRiskFloat64(max)
	if err != nil {
		return false, err
	}
	return val >= minVal && val <= maxVal, nil
}

func isRiskEmpty(value interface{}) bool {
	if value == nil {
		return true
	}
	switch v := value.(type) {
	case string:
		return len(v) == 0
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	default:
		return false
	}
}
