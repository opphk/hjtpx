package risk

import (
	"context"
	"encoding/json"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

type RuleType string

const (
	RuleTypeCondition  RuleType = "condition"
	RuleTypeAction     RuleType = "action"
	RuleTypeComposite  RuleType = "composite"
)

type Operator string

const (
	OpEq          Operator = "eq"
	OpNeq         Operator = "neq"
	OpGt          Operator = "gt"
	OpGte         Operator = "gte"
	OpLt          Operator = "lt"
	OpLte         Operator = "lte"
	OpContains    Operator = "contains"
	OpNotContains Operator = "not_contains"
	OpRegex       Operator = "regex"
	OpIn          Operator = "in"
	OpBetween     Operator = "between"
)

type Condition struct {
	Field    string      `json:"field"`
	Operator Operator    `json:"operator"`
	Value    interface{} `json:"value"`
}

type RuleAction struct {
	Type   Action     `json:"type"`
	Score  int        `json:"score"`
	Reason string     `json:"reason"`
	Next   string     `json:"next,omitempty"`
}

type Rule struct {
	ID          string         `json:"id"`
	Name        string         `json:"name"`
	Description string         `json:"description"`
	Type        RuleType       `json:"type"`
	Priority    int            `json:"priority"`
	Enabled     bool           `json:"enabled"`
	Conditions  []Condition    `json:"conditions,omitempty"`
	Logic       string         `json:"logic,omitempty"`
	Action      RuleAction     `json:"action"`
	Children    []string       `json:"children,omitempty"`
	Tags        []string       `json:"tags,omitempty"`
	CreatedAt   time.Time      `json:"created_at"`
	UpdatedAt   time.Time      `json:"updated_at"`
	Metadata    map[string]any `json:"metadata,omitempty"`
}

type RuleContext struct {
	UserID        string
	SessionID     string
	IP            string
	Domain        string
	Score         int
	Level         RiskLevel
	Behavior      *BehaviorData
	Fingerprint   *DeviceFingerprint
	Factors       []RiskFactor
	Timestamp     time.Time
	RequestCount  int64
	SuccessCount  int64
	FailureCount  int64
	CustomData    map[string]any
}

type RuleResult struct {
	RuleID       string
	RuleName     string
	Matched      bool
	Score        int
	Action       Action
	Reason       string
	ExecutionTime time.Duration
}

type RuleEngine struct {
	rules    map[string]*Rule
	mu       sync.RWMutex
	evalLock sync.Mutex
}

type RuleEngineConfig struct {
	DefaultAllow   bool
	MaxScore       int
	MaxDepth       int
	CacheEnabled   bool
	CacheSize      int
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make(map[string]*Rule),
	}
}

func (re *RuleEngine) AddRule(rule *Rule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	if _, exists := re.rules[rule.ID]; exists {
		return fmt.Errorf("规则ID %s 已存在", rule.ID)
	}

	if rule.CreatedAt.IsZero() {
		rule.CreatedAt = time.Now()
	}
	if rule.UpdatedAt.IsZero() {
		rule.UpdatedAt = time.Now()
	}

	re.rules[rule.ID] = rule

	return nil
}

func (re *RuleEngine) UpdateRule(rule *Rule) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	if _, exists := re.rules[rule.ID]; !exists {
		return fmt.Errorf("规则ID %s 不存在", rule.ID)
	}

	rule.UpdatedAt = time.Now()
	re.rules[rule.ID] = rule

	return nil
}

func (re *RuleEngine) DeleteRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	if _, exists := re.rules[ruleID]; !exists {
		return fmt.Errorf("规则ID %s 不存在", ruleID)
	}

	delete(re.rules, ruleID)

	return nil
}

func (re *RuleEngine) GetRule(ruleID string) (*Rule, bool) {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rule, exists := re.rules[ruleID]
	return rule, exists
}

func (re *RuleEngine) GetAllRules() []*Rule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules := make([]*Rule, 0, len(re.rules))
	for _, rule := range re.rules {
		rules = append(rules, rule)
	}

	return rules
}

func (re *RuleEngine) GetEnabledRules() []*Rule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	rules := make([]*Rule, 0)
	for _, rule := range re.rules {
		if rule.Enabled {
			rules = append(rules, rule)
		}
	}

	return rules
}

func (re *RuleEngine) Evaluate(ctx context.Context, rc *RuleContext) ([]RuleResult, int, Action) {
	re.evalLock.Lock()
	defer re.evalLock.Unlock()

	var results []RuleResult
	totalScore := rc.Score
	finalAction := ActionAllow

	enabledRules := re.GetEnabledRules()

	for _, rule := range enabledRules {
		start := time.Now()

		matched, score, action, reason := re.evaluateRule(rule, rc)

		result := RuleResult{
			RuleID:        rule.ID,
			RuleName:      rule.Name,
			Matched:       matched,
			Score:         score,
			Action:        action,
			Reason:        reason,
			ExecutionTime: time.Since(start),
		}

		results = append(results, result)

		if matched {
			totalScore += score

			if action == ActionBlock {
				finalAction = ActionBlock
			} else if action == ActionVerify && finalAction != ActionBlock {
				finalAction = ActionVerify
			}
		}
	}

	if totalScore > 100 {
		totalScore = 100
	}

	return results, totalScore, finalAction
}

func (re *RuleEngine) evaluateRule(rule *Rule, rc *RuleContext) (bool, int, Action, string) {
	if !rule.Enabled {
		return false, 0, "", ""
	}

	switch rule.Type {
	case RuleTypeCondition:
		return re.evaluateConditionRule(rule, rc)
	case RuleTypeAction:
		return re.evaluateActionRule(rule, rc)
	case RuleTypeComposite:
		return re.evaluateCompositeRule(rule, rc)
	default:
		return false, 0, "", ""
	}
}

func (re *RuleEngine) evaluateConditionRule(rule *Rule, rc *RuleContext) (bool, int, Action, string) {
	if len(rule.Conditions) == 0 {
		return true, rule.Action.Score, rule.Action.Type, rule.Action.Reason
	}

	matched := re.evaluateConditions(rule.Conditions, rule.Logic, rc)

	return matched, rule.Action.Score, rule.Action.Type, rule.Action.Reason
}

func (re *RuleEngine) evaluateConditions(conditions []Condition, logic string, rc *RuleContext) bool {
	if len(conditions) == 0 {
		return true
	}

	if logic == "" {
		logic = "AND"
	}
	logic = strings.ToUpper(logic)

	results := make([]bool, len(conditions))
	for i, cond := range conditions {
		results[i] = re.evaluateCondition(cond, rc)
	}

	switch logic {
	case "AND":
		for _, r := range results {
			if !r {
				return false
			}
		}
		return true
	case "OR":
		for _, r := range results {
			if r {
				return true
			}
		}
		return false
	case "NOT":
		if len(results) > 0 {
			return !results[0]
		}
		return false
	default:
		return false
	}
}

func (re *RuleEngine) evaluateCondition(cond Condition, rc *RuleContext) bool {
	fieldValue := re.getFieldValue(cond.Field, rc)

	return re.compareValues(fieldValue, cond.Operator, cond.Value)
}

func (re *RuleEngine) getFieldValue(field string, rc *RuleContext) interface{} {
	parts := strings.Split(field, ".")

	switch parts[0] {
	case "ip":
		if len(parts) > 1 {
			return nil
		}
		return rc.IP
	case "user_id":
		return rc.UserID
	case "session_id":
		return rc.SessionID
	case "domain":
		return rc.Domain
	case "score":
		return rc.Score
	case "level":
		return string(rc.Level)
	case "request_count":
		return rc.RequestCount
	case "success_count":
		return rc.SuccessCount
	case "failure_count":
		return rc.FailureCount
	case "time":
		if len(parts) > 1 {
			switch parts[1] {
			case "hour":
				return rc.Timestamp.Hour()
			case "minute":
				return rc.Timestamp.Minute()
			case "weekday":
				return int(rc.Timestamp.Weekday())
			}
		}
		return rc.Timestamp
	case "behavior":
		if rc.Behavior != nil && len(parts) > 1 {
			switch parts[1] {
			case "slide_duration":
				return rc.Behavior.SlideEnd - rc.Behavior.SlideStart
			case "track_count":
				return len(rc.Behavior.MouseTracks)
			case "click_count":
				return len(rc.Behavior.ClickTimes)
			case "success":
				return rc.Behavior.Success
			}
		}
	case "fingerprint":
		if rc.Fingerprint != nil && len(parts) > 1 {
			switch parts[1] {
			case "device_type":
				return string(rc.Fingerprint.DeviceType)
			case "os":
				return rc.Fingerprint.OS
			case "browser":
				return rc.Fingerprint.Browser
			case "is_bot":
				return rc.Fingerprint.IsBot
			}
		}
	default:
		if rc.CustomData != nil {
			if val, ok := rc.CustomData[parts[0]]; ok {
				return val
			}
		}
	}

	return nil
}

func (re *RuleEngine) compareValues(fieldValue interface{}, op Operator, targetValue interface{}) bool {
	switch op {
	case OpEq:
		return fmt.Sprintf("%v", fieldValue) == fmt.Sprintf("%v", targetValue)
	case OpNeq:
		return fmt.Sprintf("%v", fieldValue) != fmt.Sprintf("%v", targetValue)
	case OpGt:
		return re.compareNumeric(fieldValue, targetValue) > 0
	case OpGte:
		return re.compareNumeric(fieldValue, targetValue) >= 0
	case OpLt:
		return re.compareNumeric(fieldValue, targetValue) < 0
	case OpLte:
		return re.compareNumeric(fieldValue, targetValue) <= 0
	case OpContains:
		return strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", targetValue))
	case OpNotContains:
		return !strings.Contains(fmt.Sprintf("%v", fieldValue), fmt.Sprintf("%v", targetValue))
	case OpRegex:
		if str, ok := fieldValue.(string); ok {
			if regex, err := regexp.Compile(fmt.Sprintf("%v", targetValue)); err == nil {
				return regex.MatchString(str)
			}
		}
		return false
	case OpIn:
		if targetSlice, ok := targetValue.([]interface{}); ok {
			fieldStr := fmt.Sprintf("%v", fieldValue)
			for _, v := range targetSlice {
				if fmt.Sprintf("%v", v) == fieldStr {
					return true
				}
			}
		}
		return false
	case OpBetween:
		if targetSlice, ok := targetValue.([]interface{}); ok && len(targetSlice) == 2 {
			num := re.toFloat64(fieldValue)
			min := re.toFloat64(targetSlice[0])
			max := re.toFloat64(targetSlice[1])
			return num >= min && num <= max
		}
		return false
	default:
		return false
	}
}

func (re *RuleEngine) compareNumeric(a, b interface{}) float64 {
	return re.toFloat64(a) - re.toFloat64(b)
}

func (re *RuleEngine) toFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case float64:
		return val
	case float32:
		return float64(val)
	case string:
		return 0
	default:
		return 0
	}
}

func (re *RuleEngine) evaluateActionRule(rule *Rule, rc *RuleContext) (bool, int, Action, string) {
	return true, rule.Action.Score, rule.Action.Type, rule.Action.Reason
}

func (re *RuleEngine) evaluateCompositeRule(rule *Rule, rc *RuleContext) (bool, int, Action, string) {
	if len(rule.Children) == 0 {
		return false, 0, "", ""
	}

	childResults := make([]bool, 0, len(rule.Children))
	maxScore := 0
	action := ActionAllow

	for _, childID := range rule.Children {
		childRule, exists := re.GetRule(childID)
		if !exists {
			continue
		}

		matched, score, childAction, _ := re.evaluateRule(childRule, rc)
		childResults = append(childResults, matched)

		if matched {
			maxScore += score
			if childAction == ActionBlock {
				action = ActionBlock
			} else if childAction == ActionVerify && action != ActionBlock {
				action = ActionVerify
			}
		}
	}

	matched := re.evaluateConditions(
		[]Condition{{Field: "composite_result", Operator: OpContains, Value: "true"}},
		rule.Logic,
		&RuleContext{CustomData: map[string]any{"composite_result": childResults}},
	)

	return matched, maxScore, action, rule.Action.Reason
}

func (re *RuleEngine) LoadRulesFromJSON(data []byte) error {
	var rules []*Rule
	if err := json.Unmarshal(data, &rules); err != nil {
		return fmt.Errorf("解析规则JSON失败: %w", err)
	}

	for _, rule := range rules {
		if err := re.AddRule(rule); err != nil {
			return fmt.Errorf("添加规则 %s 失败: %w", rule.ID, err)
		}
	}

	return nil
}

func (re *RuleEngine) ExportRulesToJSON() ([]byte, error) {
	rules := re.GetAllRules()
	return json.MarshalIndent(rules, "", "  ")
}

func (re *RuleEngine) CreateDefaultRules() error {
	defaultRules := []*Rule{
		{
			ID:          "rule_ip_block",
			Name:        "IP黑名单拦截",
			Description: "直接拦截黑名单IP",
			Type:        RuleTypeCondition,
			Priority:    100,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "ip", Operator: OpIn, Value: []interface{}{"blacklist"}},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionBlock,
				Score: 100,
				Reason: "IP地址在黑名单中",
			},
			Tags: []string{"ip", "security"},
		},
		{
			ID:          "rule_bot_detection",
			Name:        "机器人检测",
			Description: "检测自动化工具",
			Type:        RuleTypeCondition,
			Priority:    90,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "fingerprint.is_bot", Operator: OpEq, Value: true},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionBlock,
				Score: 50,
				Reason: "检测到机器人行为",
			},
			Tags: []string{"bot", "security"},
		},
		{
			ID:          "rule_high_frequency",
			Name:        "高频访问限制",
			Description: "限制单IP访问频率",
			Type:        RuleTypeCondition,
			Priority:    80,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "request_count", Operator: OpGt, Value: 100},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionVerify,
				Score: 30,
				Reason: "访问频率过高",
			},
			Tags: []string{"frequency", "rate_limit"},
		},
		{
			ID:          "rule_fast_slide",
			Name:        "快速滑动检测",
			Description: "检测异常快速完成滑动",
			Type:        RuleTypeCondition,
			Priority:    70,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "behavior.slide_duration", Operator: OpLt, Value: 1000},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionVerify,
				Score: 25,
				Reason: "滑动速度异常快",
			},
			Tags: []string{"behavior", "speed"},
		},
		{
			ID:          "rule_low_success_rate",
			Name:        "低成功率检测",
			Description: "检测成功率过低的用户",
			Type:        RuleTypeCondition,
			Priority:    60,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "failure_count", Operator: OpGt, Value: 10},
				{Field: "request_count", Operator: OpGt, Value: 15},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionVerify,
				Score: 20,
				Reason: "验证成功率过低",
			},
			Tags: []string{"success_rate", "behavior"},
		},
		{
			ID:          "rule_suspicious_time",
			Name:        "可疑时间检测",
			Description: "检测凌晨时段的异常行为",
			Type:        RuleTypeCondition,
			Priority:    50,
			Enabled:     true,
			Conditions: []Condition{
				{Field: "time.hour", Operator: OpBetween, Value: []interface{}{0, 4}},
				{Field: "request_count", Operator: OpGt, Value: 20},
			},
			Logic: "AND",
			Action: RuleAction{
				Type:  ActionVerify,
				Score: 15,
				Reason: "可疑时间段的高频访问",
			},
			Tags: []string{"time", "behavior"},
		},
	}

	for _, rule := range defaultRules {
		if err := re.AddRule(rule); err != nil {
			return err
		}
	}

	return nil
}

func (re *RuleEngine) ValidateRule(rule *Rule) error {
	if rule.ID == "" {
		return fmt.Errorf("规则ID不能为空")
	}

	if rule.Type != RuleTypeCondition && rule.Type != RuleTypeAction && rule.Type != RuleTypeComposite {
		return fmt.Errorf("无效的规则类型: %s", rule.Type)
	}

	if rule.Type == RuleTypeCondition && len(rule.Conditions) == 0 {
		return fmt.Errorf("条件规则必须包含至少一个条件")
	}

	for _, cond := range rule.Conditions {
		if cond.Field == "" {
			return fmt.Errorf("条件字段不能为空")
		}
		if !re.isValidOperator(cond.Operator) {
			return fmt.Errorf("无效的操作符: %s", cond.Operator)
		}
	}

	return nil
}

func (re *RuleEngine) isValidOperator(op Operator) bool {
	validOps := []Operator{
		OpEq, OpNeq, OpGt, OpGte, OpLt, OpLte,
		OpContains, OpNotContains, OpRegex, OpIn, OpBetween,
	}
	for _, valid := range validOps {
		if op == valid {
			return true
		}
	}
	return false
}

func (re *RuleEngine) EnableRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("规则 %s 不存在", ruleID)
	}

	rule.Enabled = true
	rule.UpdatedAt = time.Now()

	return nil
}

func (re *RuleEngine) DisableRule(ruleID string) error {
	re.mu.Lock()
	defer re.mu.Unlock()

	rule, exists := re.rules[ruleID]
	if !exists {
		return fmt.Errorf("规则 %s 不存在", ruleID)
	}

	rule.Enabled = false
	rule.UpdatedAt = time.Now()

	return nil
}

func (re *RuleEngine) GetRulesByTag(tag string) []*Rule {
	re.mu.RLock()
	defer re.mu.RUnlock()

	var rules []*Rule
	for _, rule := range re.rules {
		for _, t := range rule.Tags {
			if t == tag {
				rules = append(rules, rule)
				break
			}
		}
	}

	return rules
}

func (re *RuleEngine) GetRulesByPriority() []*Rule {
	rules := re.GetAllRules()

	for i := 0; i < len(rules)-1; i++ {
		for j := i + 1; j < len(rules); j++ {
			if rules[i].Priority < rules[j].Priority {
				rules[i], rules[j] = rules[j], rules[i]
			}
		}
	}

	return rules
}
