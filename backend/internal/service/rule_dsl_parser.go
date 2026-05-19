package service

import (
	"encoding/json"
	"fmt"
	"regexp"
	"strconv"
	"strings"
)

type RuleDSLParser struct{}

func NewRuleDSLParser() *RuleDSLParser {
	return &RuleDSLParser{}
}

type DSLRule struct {
	ID          uint       `json:"id"`
	Name        string     `json:"name"`
	Description string     `json:"description"`
	Priority    int        `json:"priority"`
	Enabled     bool       `json:"enabled"`
	Conditions  []Condition `json:"conditions"`
	Actions     []Action   `json:"actions"`
	Expression  string     `json:"expression"`
	RiskScore   int        `json:"risk_score"`
}

type Condition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
	Type     string      `json:"type"`
}

type Action struct {
	Type  string      `json:"type"`
	Value interface{} `json:"value"`
}

type RuleEvaluationResult struct {
	RuleID    uint                 `json:"rule_id"`
	RuleName  string               `json:"rule_name"`
	Matched   bool                 `json:"matched"`
	Score     int                  `json:"score"`
	Factors   []MatchingFactor     `json:"factors"`
	Actions   []Action             `json:"actions"`
	Timestamp string               `json:"timestamp"`
}

type MatchingFactor struct {
	Field     string      `json:"field"`
	Expected  interface{} `json:"expected"`
	Actual    interface{} `json:"actual"`
	Matched   bool       `json:"matched"`
	Weight    float64    `json:"weight"`
}

func (p *RuleDSLParser) ParseExpression(expression string) (*DSLRule, error) {
	expression = strings.TrimSpace(expression)
	
	if strings.HasPrefix(expression, "{") {
		return p.parseJSONRule(expression)
	}
	
	return p.parseDSLRule(expression)
}

func (p *RuleDSLParser) parseJSONRule(expression string) (*DSLRule, error) {
	var rule DSLRule
	if err := json.Unmarshal([]byte(expression), &rule); err != nil {
		return nil, fmt.Errorf("invalid JSON format: %w", err)
	}
	
	if err := p.validateRule(&rule); err != nil {
		return nil, err
	}
	
	return &rule, nil
}

func (p *RuleDSLParser) parseDSLRule(expression string) (*DSLRule, error) {
	rule := &DSLRule{
		Conditions: make([]Condition, 0),
		Actions:    make([]Action, 0),
		Enabled:    true,
	}
	
	blocks := p.splitIntoBlocks(expression)
	
	for _, block := range blocks {
		block = strings.TrimSpace(block)
		if block == "" {
			continue
		}
		
		if strings.HasPrefix(block, "RULE") {
			rule.Name = p.extractRuleName(block)
		} else if strings.HasPrefix(block, "WHEN") {
			conditions, err := p.parseConditions(block)
			if err != nil {
				return nil, err
			}
			rule.Conditions = append(rule.Conditions, conditions...)
		} else if strings.HasPrefix(block, "THEN") {
			actions, err := p.parseActions(block)
			if err != nil {
				return nil, err
			}
			rule.Actions = append(rule.Actions, actions...)
		} else if strings.HasPrefix(block, "SCORE") {
			score, err := p.extractScore(block)
			if err != nil {
				return nil, err
			}
			rule.RiskScore = score
		}
	}
	
	rule.Expression = expression
	
	if err := p.validateRule(rule); err != nil {
		return nil, err
	}
	
	return rule, nil
}

func (p *RuleDSLParser) splitIntoBlocks(expression string) []string {
	var blocks []string
	var currentBlock strings.Builder
	inParenthesis := 0
	inBrace := 0
	
	for i, char := range expression {
		if char == '(' {
			inParenthesis++
			currentBlock.WriteRune(char)
		} else if char == ')' {
			inParenthesis--
			currentBlock.WriteRune(char)
		} else if char == '{' {
			inBrace++
			currentBlock.WriteRune(char)
		} else if char == '}' {
			inBrace--
			currentBlock.WriteRune(char)
		} else if char == ';' && inParenthesis == 0 && inBrace == 0 {
			blocks = append(blocks, currentBlock.String())
			currentBlock.Reset()
		} else {
			currentBlock.WriteRune(char)
		}
		
		if i == len(expression)-1 {
			block := strings.TrimSpace(currentBlock.String())
			if block != "" {
				blocks = append(blocks, block)
			}
		}
	}
	
	return blocks
}

func (p *RuleDSLParser) extractRuleName(block string) string {
	re := regexp.MustCompile(`(?i)RULE\s+(?:named\s+)?"([^"]+)"|(?i)RULE\s+(?:named\s+)?(\S+)`)
	matches := re.FindStringSubmatch(block)
	if len(matches) > 1 && matches[1] != "" {
		return matches[1]
	}
	if len(matches) > 2 && matches[2] != "" {
		return matches[2]
	}
	return "Unnamed Rule"
}

func (p *RuleDSLParser) parseConditions(block string) ([]Condition, error) {
	conditions := make([]Condition, 0)
	
	whenMatch := regexp.MustCompile(`(?i)WHEN\s+(.+)`).FindStringSubmatch(block)
	if len(whenMatch) < 2 {
		return conditions, nil
	}
	
	conditionStr := whenMatch[1]
	parts := p.splitConditions(conditionStr)
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		condition, err := p.parseSingleCondition(part)
		if err != nil {
			return nil, err
		}
		conditions = append(conditions, *condition)
	}
	
	return conditions, nil
}

func (p *RuleDSLParser) splitConditions(conditionStr string) []string {
	var parts []string
	var current strings.Builder
	inParenthesis := 0
	keywords := []string{"AND", "OR", "NOT"}
	
	for i := 0; i < len(conditionStr); i++ {
		char := conditionStr[i]
		
		if char == '(' {
			inParenthesis++
			current.WriteByte(char)
		} else if char == ')' {
			inParenthesis--
			current.WriteByte(char)
		} else if inParenthesis == 0 {
			for _, keyword := range keywords {
				if strings.HasPrefix(conditionStr[i:], keyword) {
					part := strings.TrimSpace(current.String())
					if part != "" {
						parts = append(parts, part)
					}
					current.Reset()
					i += len(keyword) - 1
					break
				}
			}
			if current.Len() == len(parts) {
				current.WriteByte(char)
			}
		} else {
			current.WriteByte(char)
		}
		
		if i == len(conditionStr)-1 {
			part := strings.TrimSpace(current.String())
			if part != "" {
				parts = append(parts, part)
			}
		}
	}
	
	return parts
}

func (p *RuleDSLParser) parseSingleCondition(part string) (*Condition, error) {
	part = strings.TrimSpace(part)
	
	operators := []string{"=", "!=", ">", "<", ">=", "<=", "CONTAINS", "MATCHES", "IN", "NOT IN", "STARTS WITH", "ENDS WITH"}
	
	var operator string
	var operatorIndex int
	
	for _, op := range operators {
		idx := strings.Index(part, op)
		if idx != -1 {
			operator = op
			operatorIndex = idx
			break
		}
	}
	
	if operator == "" {
		return nil, fmt.Errorf("invalid condition: %s", part)
	}
	
	field := strings.TrimSpace(part[:operatorIndex])
	valueStr := strings.TrimSpace(part[operatorIndex+len(operator):])
	
	value, err := p.parseValue(valueStr)
	if err != nil {
		return nil, err
	}
	
	return &Condition{
		Field:    field,
		Operator: operator,
		Value:    value,
		Type:     p.inferType(value),
	}, nil
}

func (p *RuleDSLParser) parseValue(valueStr string) (interface{}, error) {
	valueStr = strings.TrimSpace(valueStr)
	
	if valueStr == "true" || valueStr == "false" {
		return valueStr == "true", nil
	}
	
	if strings.HasPrefix(valueStr, "\"") && strings.HasSuffix(valueStr, "\"") {
		return valueStr[1 : len(valueStr)-1], nil
	}
	
	if strings.HasPrefix(valueStr, "[") && strings.HasSuffix(valueStr, "]") {
		return p.parseArray(valueStr)
	}
	
	if strings.HasPrefix(valueStr, "(") && strings.HasSuffix(valueStr, ")") {
		inner := valueStr[1 : len(valueStr)-1]
		parts := strings.Split(inner, ",")
		return p.parseArray("[" + strings.Join(parts, ",") + "]")
	}
	
	if num, err := strconv.Atoi(valueStr); err == nil {
		return num, nil
	}
	
	if num, err := strconv.ParseFloat(valueStr, 64); err == nil {
		return num, nil
	}
	
	return valueStr, nil
}

func (p *RuleDSLParser) parseArray(arrStr string) ([]interface{}, error) {
	arrStr = strings.TrimSpace(arrStr)
	if !strings.HasPrefix(arrStr, "[") || !strings.HasSuffix(arrStr, "]") {
		return nil, fmt.Errorf("invalid array format")
	}
	
	content := arrStr[1 : len(arrStr)-1]
	if content == "" {
		return []interface{}{}, nil
	}
	
	parts := p.splitArrayElements(content)
	result := make([]interface{}, 0, len(parts))
	
	for _, part := range parts {
		part = strings.TrimSpace(part)
		value, err := p.parseValue(part)
		if err != nil {
			return nil, err
		}
		result = append(result, value)
	}
	
	return result, nil
}

func (p *RuleDSLParser) splitArrayElements(content string) []string {
	var parts []string
	var current strings.Builder
	inQuote := false
	
	for i := 0; i < len(content); i++ {
		char := content[i]
		
		if char == '"' {
			inQuote = !inQuote
			current.WriteByte(char)
		} else if char == ',' && !inQuote {
			parts = append(parts, current.String())
			current.Reset()
		} else {
			current.WriteByte(char)
		}
	}
	
	if current.Len() > 0 {
		parts = append(parts, current.String())
	}
	
	return parts
}

func (p *RuleDSLParser) inferType(value interface{}) string {
	switch value.(type) {
	case bool:
		return "boolean"
	case int, int64, int32, float64:
		return "number"
	case string:
		return "string"
	case []interface{}:
		return "array"
	default:
		return "unknown"
	}
}

func (p *RuleDSLParser) parseActions(block string) ([]Action, error) {
	actions := make([]Action, 0)
	
	thenMatch := regexp.MustCompile(`(?i)THEN\s+(.+)`).FindStringSubmatch(block)
	if len(thenMatch) < 2 {
		return actions, nil
	}
	
	actionStr := thenMatch[1]
	actionParts := strings.Split(actionStr, ";")
	
	for _, part := range actionParts {
		part = strings.TrimSpace(part)
		if part == "" {
			continue
		}
		
		action, err := p.parseSingleAction(part)
		if err != nil {
			return nil, err
		}
		actions = append(actions, *action)
	}
	
	return actions, nil
}

func (p *RuleDSLParser) parseSingleAction(actionStr string) (*Action, error) {
	actionStr = strings.TrimSpace(actionStr)
	
	setMatch := regexp.MustCompile(`(?i)SET\s+(\w+)\s+TO\s+(.+)`).FindStringSubmatch(actionStr)
	if len(setMatch) >= 3 {
		value, err := p.parseValue(setMatch[2])
		if err != nil {
			return nil, err
		}
		return &Action{
			Type:  "set",
			Value: map[string]interface{}{
				"field": setMatch[1],
				"value": value,
			},
		}, nil
	}
	
	blockMatch := regexp.MustCompile(`(?i)BLOCK\s*\((.+)\)`).FindStringSubmatch(actionStr)
	if len(blockMatch) >= 2 {
		return &Action{
			Type:  "block",
			Value: blockMatch[1],
		}, nil
	}
	
	alertMatch := regexp.MustCompile(`(?i)ALERT\s*\((.+)\)`).FindStringSubmatch(actionStr)
	if len(alertMatch) >= 2 {
		return &Action{
			Type:  "alert",
			Value: alertMatch[1],
		}, nil
	}
	
	scoreMatch := regexp.MustCompile(`(?i)ADD\s+SCORE\s+(\d+)`).FindStringSubmatch(actionStr)
	if len(scoreMatch) >= 2 {
		score, _ := strconv.Atoi(scoreMatch[1])
		return &Action{
			Type:  "add_score",
			Value: score,
		}, nil
	}
	
	return &Action{
		Type:  "unknown",
		Value: actionStr,
	}, nil
}

func (p *RuleDSLParser) extractScore(block string) (int, error) {
	scoreMatch := regexp.MustCompile(`(?i)SCORE\s*[:=]?\s*(\d+)`).FindStringSubmatch(block)
	if len(scoreMatch) >= 2 {
		return strconv.Atoi(scoreMatch[1])
	}
	return 0, nil
}

func (p *RuleDSLParser) validateRule(rule *DSLRule) error {
	if rule.Name == "" {
		return fmt.Errorf("rule name is required")
	}
	
	if len(rule.Conditions) == 0 {
		return fmt.Errorf("rule must have at least one condition")
	}
	
	if len(rule.Actions) == 0 {
		return fmt.Errorf("rule must have at least one action")
	}
	
	validFields := map[string]bool{
		"ip": true, "user_agent": true, "country": true, "city": true,
		"device_type": true, "browser": true, "os": true,
		"request_count": true, "fail_count": true, "session_count": true,
		"risk_score": true, "timestamp": true, "email_domain": true,
		"referer": true, "url_path": true, "method": true,
	}
	
	for _, cond := range rule.Conditions {
		if !validFields[cond.Field] {
			return fmt.Errorf("invalid field: %s", cond.Field)
		}
	}
	
	return nil
}

func (p *RuleDSLParser) EvaluateRule(rule *DSLRule, context map[string]interface{}) (*RuleEvaluationResult, error) {
	result := &RuleEvaluationResult{
		RuleID:    rule.ID,
		RuleName:  rule.Name,
		Matched:   true,
		Score:     0,
		Factors:   make([]MatchingFactor, 0),
		Actions:   make([]Action, 0),
	}
	
	for _, condition := range rule.Conditions {
		actualValue := context[condition.Field]
		matched := p.evaluateCondition(condition, actualValue)
		
		factor := MatchingFactor{
			Field:    condition.Field,
			Expected: condition.Value,
			Actual:   actualValue,
			Matched:  matched,
			Weight:   1.0,
		}
		result.Factors = append(result.Factors, factor)
		
		if !matched {
			result.Matched = false
		}
	}
	
	if result.Matched {
		result.Score = rule.RiskScore
		result.Actions = rule.Actions
	}
	
	return result, nil
}

func (p *RuleDSLParser) evaluateCondition(condition Condition, actualValue interface{}) bool {
	switch condition.Operator {
	case "=":
		return p.equals(actualValue, condition.Value)
	case "!=":
		return !p.equals(actualValue, condition.Value)
	case ">":
		return p.compare(actualValue, condition.Value) > 0
	case "<":
		return p.compare(actualValue, condition.Value) < 0
	case ">=":
		return p.compare(actualValue, condition.Value) >= 0
	case "<=":
		return p.compare(actualValue, condition.Value) <= 0
	case "CONTAINS":
		return p.contains(actualValue, condition.Value)
	case "MATCHES":
		return p.matches(actualValue, condition.Value)
	case "IN":
		return p.inArray(actualValue, condition.Value)
	case "NOT IN":
		return !p.inArray(actualValue, condition.Value)
	default:
		return false
	}
}

func (p *RuleDSLParser) equals(actual, expected interface{}) bool {
	if actual == nil && expected == nil {
		return true
	}
	if actual == nil || expected == nil {
		return false
	}
	
	actualStr := fmt.Sprintf("%v", actual)
	expectedStr := fmt.Sprintf("%v", expected)
	
	return actualStr == expectedStr
}

func (p *RuleDSLParser) compare(actual, expected interface{}) int {
	actualNum, err1 := toFloat64(actual)
	expectedNum, err2 := toFloat64(expected)
	
	if err1 != nil || err2 != nil {
		return strings.Compare(fmt.Sprintf("%v", actual), fmt.Sprintf("%v", expected))
	}
	
	if actualNum < expectedNum {
		return -1
	} else if actualNum > expectedNum {
		return 1
	}
	return 0
}

func toFloat64(v interface{}) (float64, error) {
	switch val := v.(type) {
	case int:
		return float64(val), nil
	case int64:
		return float64(val), nil
	case int32:
		return float64(val), nil
	case float64:
		return val, nil
	case float32:
		return float64(val), nil
	case string:
		return strconv.ParseFloat(val, 64)
	default:
		return 0, fmt.Errorf("cannot convert %T to float64", v)
	}
}

func (p *RuleDSLParser) contains(actual, expected interface{}) bool {
	actualStr := strings.ToLower(fmt.Sprintf("%v", actual))
	expectedStr := strings.ToLower(fmt.Sprintf("%v", expected))
	return strings.Contains(actualStr, expectedStr)
}

func (p *RuleDSLParser) matches(actual, expected interface{}) bool {
	pattern := fmt.Sprintf("%v", expected)
	matched, err := regexp.MatchString(pattern, fmt.Sprintf("%v", actual))
	if err != nil {
		return false
	}
	return matched
}

func (p *RuleDSLParser) inArray(actual, expected interface{}) bool {
	arr, ok := expected.([]interface{})
	if !ok {
		return false
	}
	
	actualStr := fmt.Sprintf("%v", actual)
	for _, item := range arr {
		if fmt.Sprintf("%v", item) == actualStr {
			return true
		}
	}
	return false
}

func (p *RuleDSLParser) GenerateDSL(rule *DSLRule) string {
	var sb strings.Builder
	
	sb.WriteString(fmt.Sprintf(`RULE "%s";`, rule.Name))
	sb.WriteString("\n")
	
	if len(rule.Conditions) > 0 {
		sb.WriteString("WHEN ")
		for i, cond := range rule.Conditions {
			if i > 0 {
				sb.WriteString(" AND ")
			}
			sb.WriteString(fmt.Sprintf("%s %s %v", cond.Field, cond.Operator, cond.Value))
		}
		sb.WriteString(";\n")
	}
	
	if len(rule.Actions) > 0 {
		sb.WriteString("THEN ")
		for i, action := range rule.Actions {
			if i > 0 {
				sb.WriteString("; ")
			}
			sb.WriteString(p.actionToString(action))
		}
		sb.WriteString(";\n")
	}
	
	if rule.RiskScore > 0 {
		sb.WriteString(fmt.Sprintf("SCORE %d;\n", rule.RiskScore))
	}
	
	return sb.String()
}

func (p *RuleDSLParser) actionToString(action Action) string {
	switch action.Type {
	case "block":
		return fmt.Sprintf("BLOCK(%v)", action.Value)
	case "alert":
		return fmt.Sprintf("ALERT(%v)", action.Value)
	case "add_score":
		return fmt.Sprintf("ADD SCORE %v", action.Value)
	case "set":
		if m, ok := action.Value.(map[string]interface{}); ok {
			return fmt.Sprintf("SET %s TO %v", m["field"], m["value"])
		}
		return fmt.Sprintf("%v", action.Value)
	default:
		return fmt.Sprintf("%v", action.Value)
	}
}
