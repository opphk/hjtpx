package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

type RiskCacheService interface {
	Get(ctx context.Context, key string) (int64, error)
	Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error
	Incr(ctx context.Context, key string) (int64, error)
	Expire(ctx context.Context, key string, duration time.Duration) error
}

type RiskRuleEngineV2 struct {
	db              *gorm.DB
	workflowEngine  *WorkflowEngine
	cacheService    RiskCacheService
	ruleMutex       sync.RWMutex
	compiledRules   map[uint]*CompiledRule
	ruleConditions  map[string]ConditionEvaluator
}

type CompiledRule struct {
	Rule       *models.RiskRule
	Condition  *CompiledRuleCondition
	Actions    []ActionConfig
	CompiledAt time.Time
}

type CompiledRuleCondition struct {
	Type       string
	Expression string
	Fields     []string
}

type ActionConfig struct {
	Type      string
	Target    string
	Params    map[string]interface{}
	Delay     time.Duration
	Priority  int
}

type WorkflowEngine struct {
	db              *gorm.DB
	workflows       map[string]*Workflow
	workflowMutex   sync.RWMutex
	eventQueue      chan *WorkflowEvent
	executorPool    *ExecutorPool
}

type Workflow struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Trigger     WorkflowTrigger        `json:"trigger"`
	Steps       []WorkflowStep         `json:"steps"`
	Status      string                 `json:"status"`
	CreatedAt   time.Time              `json:"created_at"`
	UpdatedAt   time.Time              `json:"updated_at"`
}

type WorkflowTrigger struct {
	Type      string                 `json:"type"`
	Condition map[string]interface{} `json:"condition"`
}

type WorkflowStep struct {
	ID          string                 `json:"id"`
	Name        string                 `json:"name"`
	Type        string                 `json:"type"`
	Config      map[string]interface{} `json:"config"`
	Conditions  []StepCondition        `json:"conditions"`
	OnError     string                 `json:"on_error"`
	RetryPolicy *RiskRuleEngineRetryPolicy          `json:"retry_policy"`
}

type StepCondition struct {
	Field    string      `json:"field"`
	Operator string      `json:"operator"`
	Value    interface{} `json:"value"`
}

type RiskRuleEngineRetryPolicy struct {
	MaxAttempts int           `json:"max_attempts"`
	Delay       time.Duration `json:"delay"`
	Backoff     string        `json:"backoff"`
}

type WorkflowEvent struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	Type        string                 `json:"type"`
	Data        map[string]interface{} `json:"data"`
	Timestamp   time.Time              `json:"timestamp"`
	Context     map[string]interface{} `json:"context"`
}

type WorkflowExecution struct {
	ID          string                 `json:"id"`
	WorkflowID  string                 `json:"workflow_id"`
	EventID     string                 `json:"event_id"`
	Status      string                 `json:"status"`
	Steps       []StepExecution        `json:"steps"`
	StartedAt   time.Time              `json:"started_at"`
	CompletedAt *time.Time             `json:"completed_at,omitempty"`
	Error       string                 `json:"error,omitempty"`
}

type StepExecution struct {
	StepID     string                 `json:"step_id"`
	Status     string                 `json:"status"`
	StartedAt  time.Time              `json:"started_at"`
	EndedAt    *time.Time             `json:"ended_at,omitempty"`
	Output     map[string]interface{} `json:"output,omitempty"`
	Error      string                 `json:"error,omitempty"`
}

type ExecutorPool struct {
	workers    int
	taskQueue  chan *WorkflowEvent
	results    chan *WorkflowExecution
	wg         sync.WaitGroup
}

func NewRiskRuleEngineV2(db *gorm.DB) *RiskRuleEngineV2 {
	engine := &RiskRuleEngineV2{
		db:             db,
		workflowEngine: NewWorkflowEngine(db),
		compiledRules: make(map[uint]*CompiledRule),
		ruleConditions: make(map[string]ConditionEvaluator),
	}
	
	engine.initConditionEvaluators()
	
	return engine
}

func NewWorkflowEngine(db *gorm.DB) *WorkflowEngine {
	engine := &WorkflowEngine{
		db:         db,
		workflows:  make(map[string]*Workflow),
		eventQueue: make(chan *WorkflowEvent, 1000),
	}
	
	go engine.processEvents()
	
	return engine
}

func (e *WorkflowEngine) processEvents() {
	for event := range e.eventQueue {
		e.executeWorkflow(event)
	}
}

func (e *WorkflowEngine) executeWorkflow(event *WorkflowEvent) {
	e.workflowMutex.RLock()
	workflow, exists := e.workflows[event.WorkflowID]
	e.workflowMutex.RUnlock()
	
	if !exists {
		var w models.Workflow
		if err := e.db.Where("id = ? AND status = ?", event.WorkflowID, "active").First(&w).Error; err != nil {
			return
		}
		
		if err := json.Unmarshal([]byte(w.Definition), &workflow); err != nil {
			return
		}
		
		e.workflowMutex.Lock()
		e.workflows[event.WorkflowID] = workflow
		e.workflowMutex.Unlock()
	}
	
	execution := &WorkflowExecution{
		ID:         fmt.Sprintf("exec_%d", time.Now().UnixNano()),
		WorkflowID: event.WorkflowID,
		EventID:    event.ID,
		Status:     "running",
		Steps:      make([]StepExecution, len(workflow.Steps)),
		StartedAt:  time.Now(),
	}
	
	e.db.Create(&models.WorkflowExecution{
		ID:         execution.ID,
		WorkflowID: event.WorkflowID,
		Status:     "running",
		StartedAt:  execution.StartedAt,
	})
	
	for i, step := range workflow.Steps {
		execution.Steps[i] = e.executeStep(step, event, workflow)
		
		if execution.Steps[i].Status == "failed" && step.OnError == "stop" {
			execution.Status = "failed"
			execution.Error = execution.Steps[i].Error
			break
		}
	}
	
	if execution.Status != "failed" {
		execution.Status = "completed"
	}
	
	now := time.Now()
	execution.CompletedAt = &now
	e.db.Model(&models.WorkflowExecution{}).Where("id = ?", execution.ID).Updates(map[string]interface{}{
		"status":      execution.Status,
		"completed_at": now,
		"result":       string(mustMarshalJSON(execution.Steps)),
	})
}

func (e *WorkflowEngine) executeStep(step WorkflowStep, event *WorkflowEvent, workflow *Workflow) StepExecution {
	exec := StepExecution{
		StepID:    step.ID,
		Status:    "running",
		StartedAt: time.Now(),
		Output:    make(map[string]interface{}),
	}
	
	for _, cond := range step.Conditions {
		if !e.evaluateCondition(cond, event) {
			exec.Status = "skipped"
			return exec
		}
	}
	
	var err error
	switch step.Type {
	case "action":
		exec.Output, err = e.executeAction(step.Config, event)
	case "condition":
		exec.Output, err = e.executeConditionStep(step.Config, event)
	case "delay":
		exec.Output, err = e.executeDelayStep(step.Config, event)
	case "notification":
		exec.Output, err = e.executeNotificationStep(step.Config, event)
	case "webhook":
		exec.Output, err = e.executeWebhookStep(step.Config, event)
	case "transform":
		exec.Output, err = e.executeTransformStep(step.Config, event)
	default:
		err = fmt.Errorf("unknown step type: %s", step.Type)
	}
	
	if err != nil {
		exec.Status = "failed"
		exec.Error = err.Error()
	} else {
		exec.Status = "completed"
	}
	
	now := time.Now()
	exec.EndedAt = &now
	return exec
}

func (e *WorkflowEngine) evaluateCondition(cond StepCondition, event *WorkflowEvent) bool {
	value := e.getFieldValue(cond.Field, event.Data)
	
	switch cond.Operator {
	case "eq":
		return fmt.Sprintf("%v", value) == fmt.Sprintf("%v", cond.Value)
	case "ne":
		return fmt.Sprintf("%v", value) != fmt.Sprintf("%v", cond.Value)
	case "gt":
		return riskRuleEngineToFloat64(value) > riskRuleEngineToFloat64(cond.Value)
	case "gte":
		return riskRuleEngineToFloat64(value) >= riskRuleEngineToFloat64(cond.Value)
	case "lt":
		return riskRuleEngineToFloat64(value) < riskRuleEngineToFloat64(cond.Value)
	case "lte":
		return riskRuleEngineToFloat64(value) <= riskRuleEngineToFloat64(cond.Value)
	case "contains":
		return strings.Contains(fmt.Sprintf("%v", value), fmt.Sprintf("%v", cond.Value))
	case "matches":
		matched, _ := regexp.MatchString(fmt.Sprintf("%v", cond.Value), fmt.Sprintf("%v", value))
		return matched
	case "in":
		return e.valueInList(value, cond.Value)
	default:
		return false
	}
}

func (e *WorkflowEngine) getFieldValue(field string, data map[string]interface{}) interface{} {
	parts := strings.Split(field, ".")
	var current interface{} = data
	
	for _, part := range parts {
		if m, ok := current.(map[string]interface{}); ok {
			current = m[part]
		} else {
			return nil
		}
	}
	
	return current
}

func (e *WorkflowEngine) valueInList(value interface{}, list interface{}) bool {
	switch l := list.(type) {
	case []interface{}:
		valStr := fmt.Sprintf("%v", value)
		for _, item := range l {
			if fmt.Sprintf("%v", item) == valStr {
				return true
			}
		}
	case []string:
		valStr := fmt.Sprintf("%v", value)
		for _, item := range l {
			if item == valStr {
				return true
			}
		}
	}
	return false
}

func (e *WorkflowEngine) executeAction(config map[string]interface{}, event *WorkflowEvent) (map[string]interface{}, error) {
	actionType := config["type"].(string)
	result := make(map[string]interface{})
	
	switch actionType {
	case "block":
		result["blocked"] = true
		result["reason"] = config["reason"]
	case "challenge":
		result["challenge_required"] = true
		result["captcha_type"] = config["captcha_type"]
	case "allow":
		result["allowed"] = true
	case "flag":
		result["flagged"] = true
		result["flag_reason"] = config["reason"]
	case "rate_limit":
		result["rate_limited"] = true
		result["limit"] = config["limit"]
		result["window"] = config["window"]
	case "notify":
		result["notified"] = true
		result["channels"] = config["channels"]
	default:
		return nil, fmt.Errorf("unknown action type: %s", actionType)
	}
	
	return result, nil
}

func (e *WorkflowEngine) executeConditionStep(config map[string]interface{}, event *WorkflowEvent) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	if condition, ok := config["condition"].(string); ok {
		ops := strings.Split(condition, " ")
		if len(ops) >= 3 {
			field := ops[0]
			operator := ops[1]
			value := strings.Join(ops[2:], " ")
			
			cond := StepCondition{
				Field:    field,
				Operator: operator,
				Value:    value,
			}
			
			result["matched"] = e.evaluateCondition(cond, event)
		}
	}
	
	return result, nil
}

func (e *WorkflowEngine) executeDelayStep(config map[string]interface{}, event *WorkflowEvent) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	if duration, ok := config["duration"].(float64); ok {
		time.Sleep(time.Duration(duration) * time.Millisecond)
		result["delayed"] = true
		result["duration"] = duration
	}
	
	return result, nil
}

func (e *WorkflowEngine) executeNotificationStep(config map[string]interface{}, event *WorkflowEvent) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	channel := config["channel"].(string)
	message := config["message"].(string)
	
	result["sent"] = true
	result["channel"] = channel
	result["message"] = message
	
	return result, nil
}

func (e *WorkflowEngine) executeWebhookStep(config map[string]interface{}, event *WorkflowEvent) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	url := config["url"].(string)
	method := "POST"
	if m, ok := config["method"].(string); ok {
		method = m
	}
	
	result["webhook_called"] = true
	result["url"] = url
	result["method"] = method
	result["payload"] = event.Data
	
	return result, nil
}

func (e *WorkflowEngine) executeTransformStep(config map[string]interface{}, event *WorkflowEvent) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	if mappings, ok := config["mappings"].([]interface{}); ok {
		for _, m := range mappings {
			if mapping, ok := m.(map[string]interface{}); ok {
				source := mapping["source"].(string)
				target := mapping["target"].(string)
				value := e.getFieldValue(source, event.Data)
				e.setFieldValue(target, value, result)
			}
		}
	}
	
	return result, nil
}

func (e *WorkflowEngine) setFieldValue(field string, value interface{}, data map[string]interface{}) {
	parts := strings.Split(field, ".")
	current := data
	
	for i := 0; i < len(parts)-1; i++ {
		if _, exists := current[parts[i]]; !exists {
			current[parts[i]] = make(map[string]interface{})
		}
		current = current[parts[i]].(map[string]interface{})
	}
	
	current[parts[len(parts)-1]] = value
}

func (e *RiskRuleEngineV2) initConditionEvaluators() {
	e.ruleConditions["ip_frequency"] = e.evalIPFrequency
	e.ruleConditions["velocity"] = e.evalVelocity
	e.ruleConditions["pattern_match"] = e.evalPatternMatch
	e.ruleConditions["risk_score"] = e.evalRiskScore
	e.ruleConditions["geo_anomaly"] = e.evalGeoAnomaly
	e.ruleConditions["device_reputation"] = e.evalDeviceReputation
	e.ruleConditions["behavior_anomaly"] = e.evalBehaviorAnomaly
	e.ruleConditions["time_window"] = e.evalTimeWindow
	e.ruleConditions["threshold"] = e.evalThreshold
	e.ruleConditions["expression"] = e.evalExpression
}

type ConditionEvaluator func(context *model.RiskContext, params map[string]interface{}) bool

func (e *RiskRuleEngineV2) CompileRule(rule *models.RiskRule) (*CompiledRule, error) {
	e.ruleMutex.Lock()
	defer e.ruleMutex.Unlock()
	
	compiled := &CompiledRule{
		Rule:       rule,
		CompiledAt: time.Now(),
	}
	
	var params map[string]interface{}
	if rule.Params != "" {
		if err := json.Unmarshal([]byte(rule.Params), &params); err != nil {
			return nil, err
		}
	}
	
	if rule.Condition != "" {
		compiled.Condition = &CompiledRuleCondition{
			Type:       params["condition_type"].(string),
			Expression: rule.Condition,
			Fields:     e.extractFields(rule.Condition),
		}
	}
	
	if actions, ok := params["actions"].([]interface{}); ok {
		for _, a := range actions {
			if actionMap, ok := a.(map[string]interface{}); ok {
				action := ActionConfig{
					Type:     actionMap["type"].(string),
					Target:   actionMap["target"].(string),
					Params:   actionMap,
					Priority: 100,
				}
				
				if delay, ok := actionMap["delay"].(float64); ok {
					action.Delay = time.Duration(delay) * time.Millisecond
				}
				
				if priority, ok := actionMap["priority"].(float64); ok {
					action.Priority = int(priority)
				}
				
				compiled.Actions = append(compiled.Actions, action)
			}
		}
	}
	
	e.compiledRules[rule.ID] = compiled
	return compiled, nil
}

func (e *RiskRuleEngineV2) extractFields(expression string) []string {
	re := regexp.MustCompile(`\{\{(\w+)\}\}|\b(\w+)\b`)
	matches := re.FindAllStringSubmatch(expression, -1)
	
	fields := make([]string, 0)
	seen := make(map[string]bool)
	
	for _, match := range matches {
		for _, group := range match[1:] {
			if group != "" && !seen[group] {
				seen[group] = true
				fields = append(fields, group)
			}
		}
	}
	
	return fields
}

func (e *RiskRuleEngineV2) EvaluateRules(ctx *model.RiskContext) ([]CompiledRule, []ActionConfig, error) {
	e.ruleMutex.RLock()
	defer e.ruleMutex.RUnlock()
	
	var triggeredRules []CompiledRule
	var actions []ActionConfig
	
	for _, compiled := range e.compiledRules {
		if !compiled.Rule.IsEnabled {
			continue
		}
		
		matched, err := e.evaluateCompiledRule(compiled, ctx)
		if err != nil {
			continue
		}
		
		if matched {
			triggeredRules = append(triggeredRules, *compiled)
			actions = append(actions, compiled.Actions...)
		}
	}
	
	return triggeredRules, actions, nil
}

func (e *RiskRuleEngineV2) evaluateCompiledRule(compiled *CompiledRule, ctx *model.RiskContext) (bool, error) {
	if compiled.Condition == nil {
		return false, nil
	}
	
	evaluator, exists := e.ruleConditions[compiled.Condition.Type]
	if !exists {
		return false, fmt.Errorf("unknown condition type: %s", compiled.Condition.Type)
	}
	
	var params map[string]interface{}
	if compiled.Rule.Params != "" {
		json.Unmarshal([]byte(compiled.Rule.Params), &params)
	}
	
	return evaluator(ctx, params), nil
}

func (e *RiskRuleEngineV2) evalIPFrequency(ctx *model.RiskContext, params map[string]interface{}) bool {
	threshold := 100
	if t, ok := params["threshold"].(float64); ok {
		threshold = int(t)
	}
	
	window := 60
	if w, ok := params["window"].(float64); ok {
		window = int(w)
	}
	
	count, err := e.getIPRequestCount(ctx.IPAddress, window)
	if err != nil {
		return false
	}
	
	return count > threshold
}

func (e *RiskRuleEngineV2) evalVelocity(ctx *model.RiskContext, params map[string]interface{}) bool {
	speedThreshold := 2000.0
	if s, ok := params["speed_threshold"].(float64); ok {
		speedThreshold = s
	}
	
	if ctx.MouseSpeed > speedThreshold {
		return true
	}
	
	return false
}

func (e *RiskRuleEngineV2) evalPatternMatch(ctx *model.RiskContext, params map[string]interface{}) bool {
	pattern := params["pattern"].(string)
	
	for _, trace := range ctx.TraceData {
		matched, _ := regexp.MatchString(pattern, fmt.Sprintf("%f,%f", trace.X, trace.Y))
		if matched {
			return true
		}
	}
	
	return false
}

func (e *RiskRuleEngineV2) evalRiskScore(ctx *model.RiskContext, params map[string]interface{}) bool {
	scoreThreshold := 70.0
	if s, ok := params["score_threshold"].(float64); ok {
		scoreThreshold = s
	}
	
	return ctx.RiskScore > scoreThreshold
}

func (e *RiskRuleEngineV2) evalGeoAnomaly(ctx *model.RiskContext, params map[string]interface{}) bool {
	if ctx.LastKnownLocation == "" {
		return false
	}
	
	allowedRegions := params["allowed_regions"].([]interface{})
	
	for _, region := range allowedRegions {
		if ctx.CurrentLocation == region.(string) {
			return false
		}
	}
	
	return ctx.CurrentLocation != "" && ctx.CurrentLocation != ctx.LastKnownLocation
}

func (e *RiskRuleEngineV2) evalDeviceReputation(ctx *model.RiskContext, params map[string]interface{}) bool {
	minReputation := 50.0
	if r, ok := params["min_reputation"].(float64); ok {
		minReputation = r
	}
	
	return ctx.DeviceReputationScore < minReputation
}

func (e *RiskRuleEngineV2) evalBehaviorAnomaly(ctx *model.RiskContext, params map[string]interface{}) bool {
	if len(ctx.TraceData) < 10 {
		return false
	}
	
	pathEfficiency := e.calculatePathEfficiency(ctx.TraceData)
	efficiencyThreshold := 0.95
	if t, ok := params["efficiency_threshold"].(float64); ok {
		efficiencyThreshold = t
	}
	
	return pathEfficiency > efficiencyThreshold
}

func (e *RiskRuleEngineV2) evalTimeWindow(ctx *model.RiskContext, params map[string]interface{}) bool {
	startHour := 0
	endHour := 24
	
	if s, ok := params["start_hour"].(float64); ok {
		startHour = int(s)
	}
	if e, ok := params["end_hour"].(float64); ok {
		endHour = int(e)
	}
	
	currentHour := time.Now().Hour()
	
	if startHour <= endHour {
		return currentHour < startHour || currentHour > endHour
	}
	
	return currentHour < startHour && currentHour > endHour
}

func (e *RiskRuleEngineV2) evalThreshold(ctx *model.RiskContext, params map[string]interface{}) bool {
	metric := params["metric"].(string)
	threshold := riskRuleEngineToFloat64(params["threshold"])
	operator := params["operator"].(string)
	
	var value float64
	switch metric {
	case "failure_count":
		value = float64(ctx.FailureCount)
	case "verification_count":
		value = float64(ctx.VerificationCount)
	case "mouse_speed":
		value = ctx.MouseSpeed
	case "click_count":
		value = float64(ctx.ClickCount)
	default:
		return false
	}
	
	switch operator {
	case "gt":
		return value > threshold
	case "gte":
		return value >= threshold
	case "lt":
		return value < threshold
	case "lte":
		return value <= threshold
	case "eq":
		return math.Abs(value-threshold) < 0.001
	default:
		return false
	}
}

func (e *RiskRuleEngineV2) evalExpression(ctx *model.RiskContext, params map[string]interface{}) bool {
	expression := params["expression"].(string)
	
	replacer := strings.NewReplacer(
		"{{failure_count}}", fmt.Sprintf("%d", ctx.FailureCount),
		"{{verification_count}}", fmt.Sprintf("%d", ctx.VerificationCount),
		"{{mouse_speed}}", fmt.Sprintf("%f", ctx.MouseSpeed),
		"{{risk_score}}", fmt.Sprintf("%f", ctx.RiskScore),
		"{{is_proxy}}", fmt.Sprintf("%t", ctx.IsProxy),
		"{{is_vpn}}", fmt.Sprintf("%t", ctx.IsVPN),
		"{{is_tor}}", fmt.Sprintf("%t", ctx.IsTor),
		"{{position_diff}}", fmt.Sprintf("%f", ctx.PositionDiff),
	)
	
	expr := replacer.Replace(expression)
	
	ops := strings.Fields(expr)
	if len(ops) < 3 {
		return false
	}

	left, _ := parseFloat(ops[0])
	op := ops[1]
	right, _ := parseFloat(ops[2])
	
	switch op {
	case ">":
		return left > right
	case ">=":
		return left >= right
	case "<":
		return left < right
	case "<=":
		return left <= right
	case "==":
		return math.Abs(left-right) < 0.001
	case "!=":
		return math.Abs(left-right) >= 0.001
	default:
		return false
	}
}

func (e *RiskRuleEngineV2) getIPRequestCount(ip string, windowSeconds int) (int, error) {
	ctx := context.Background()
	cacheKey := fmt.Sprintf("ip_count:%s:%d", ip, windowSeconds/60)
	
	if e.cacheService != nil {
		if cached, err := e.cacheService.Get(ctx, cacheKey); err == nil {
			return int(cached), nil
		}
	}
	
	var count int64
	since := time.Now().Add(-time.Duration(windowSeconds) * time.Second)
	e.db.Model(&models.VerificationLog{}).Where("ip_address = ? AND created_at >= ?", ip, since).Count(&count)
	
	if e.cacheService != nil {
		e.cacheService.Set(ctx, cacheKey, count, 30*time.Second)
	}
	
	return int(count), nil
}

func (e *RiskRuleEngineV2) calculatePathEfficiency(traceData []model.TracePoint) float64 {
	if len(traceData) < 2 {
		return 0
	}
	
	start := traceData[0]
	end := traceData[len(traceData)-1]
	
	directDist := math.Sqrt(math.Pow(end.X-start.X, 2) + math.Pow(end.Y-start.Y, 2))
	
	totalDist := 0.0
	for i := 1; i < len(traceData); i++ {
		dx := traceData[i].X - traceData[i-1].X
		dy := traceData[i].Y - traceData[i-1].Y
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}
	
	if totalDist == 0 {
		return 0
	}
	
	return directDist / totalDist
}

func (e *RiskRuleEngineV2) ExecuteAutomatedResponse(ctx *model.RiskContext, actions []ActionConfig) (*RiskRuleAutomatedResponse, error) {
	response := &RiskRuleAutomatedResponse{
		ID:        fmt.Sprintf("resp_%d", time.Now().UnixNano()),
		Timestamp: time.Now(),
		RiskScore: ctx.RiskScore,
		Actions:   make([]ExecutedAction, 0),
	}
	
	sortedActions := make([]ActionConfig, len(actions))
	copy(sortedActions, actions)
	
	for _, action := range sortedActions {
		if action.Delay > 0 {
			time.Sleep(action.Delay)
		}
		
		executed, err := e.executeAction(action, ctx)
		if err != nil {
			response.Actions = append(response.Actions, ExecutedAction{
				Action: action,
				Status: "failed",
				Error:  err.Error(),
			})
			continue
		}
		
		response.Actions = append(response.Actions, ExecutedAction{
			Action: action,
			Status: "completed",
			Output: executed,
		})
		
		if action.Type == "block" || action.Type == "challenge" {
			response.Decision = action.Type
			break
		}
	}
	
	if response.Decision == "" {
		response.Decision = "allow"
	}
	
	e.recordResponse(response, ctx)
	
	return response, nil
}

type RiskRuleAutomatedResponse struct {
	ID        string          `json:"id"`
	Timestamp time.Time       `json:"timestamp"`
	RiskScore float64         `json:"risk_score"`
	Decision  string          `json:"decision"`
	Actions   []ExecutedAction `json:"actions"`
}

type ExecutedAction struct {
	Action ActionConfig        `json:"action"`
	Status string              `json:"status"`
	Output map[string]interface{} `json:"output,omitempty"`
	Error  string              `json:"error,omitempty"`
}

func (e *RiskRuleEngineV2) executeAction(action ActionConfig, ctx *model.RiskContext) (map[string]interface{}, error) {
	result := make(map[string]interface{})
	
	switch action.Type {
	case "block":
		result["blocked"] = true
		result["reason"] = action.Params["reason"]
		
	case "challenge":
		result["challenge_required"] = true
		result["captcha_type"] = action.Params["captcha_type"]
		
	case "allow":
		result["allowed"] = true
		
	case "flag":
		result["flagged"] = true
		e.flagUser(ctx, action.Params["reason"].(string))
		
	case "rate_limit":
		result["rate_limited"] = true
		e.applyRateLimit(ctx, action.Params)
		
	case "notify":
		result["notified"] = true
		e.sendNotification(ctx, action.Params)
		
	case "webhook":
		result["webhook_triggered"] = true
		e.triggerWebhook(ctx, action.Params)
		
	case "blacklist":
		result["blacklisted"] = true
		e.addToBlacklist(ctx, action.Params)
		
	default:
		return nil, fmt.Errorf("unknown action type: %s", action.Type)
	}
	
	return result, nil
}

func (e *RiskRuleEngineV2) flagUser(ctx *model.RiskContext, reason string) {
	e.db.Create(&models.Blacklist{
		Target:  ctx.IPAddress,
		Type:    "ip",
		Source:  "automated",
		Reason:  reason,
		Action:  "flag",
		Status:  "active",
		Note:    fmt.Sprintf("Automated flag: %s", reason),
	})
}

func (e *RiskRuleEngineV2) applyRateLimit(ctx *model.RiskContext, params map[string]interface{}) {
	limit := int(params["limit"].(float64))
	window := int(params["window"].(float64))
	
	ctxKey := fmt.Sprintf("rate_limit:%s", ctx.IPAddress)
	
	if e.cacheService != nil {
		ctx := context.Background()
		count, _ := e.cacheService.Incr(ctx, ctxKey)
		if count == 1 {
			e.cacheService.Expire(ctx, ctxKey, time.Duration(window)*time.Second)
		}
	}
	
	e.db.Create(&models.Blacklist{
		Target:  ctx.IPAddress,
		Type:    "ip",
		Source:  "automated",
		Reason:  fmt.Sprintf("Rate limit exceeded: %d/%ds", limit, window),
		Action:  "rate_limit",
		Status:  "active",
	})
}

func (e *RiskRuleEngineV2) sendNotification(ctx *model.RiskContext, params map[string]interface{}) {
	channels := params["channels"].([]interface{})
	message := params["message"].(string)
	
	for _, ch := range channels {
		channel := ch.(string)
		
		e.db.Create(&models.AlertRecord{
			RuleID:   0,
			RuleName: "Automated Notification",
			EventType: "notification",
			Severity: "info",
			Message:  message,
			Context:  string(mustMarshalJSON(ctx)),
			Status:   "triggered",
		})
		
		switch channel {
		case "email":
		case "webhook":
		case "slack":
		case "sms":
		}
	}
}

func (e *RiskRuleEngineV2) triggerWebhook(ctx *model.RiskContext, params map[string]interface{}) {
	webhookURL := params["url"].(string)

	e.db.Create(&models.WebhookConfig{
		Name:   "automated_webhook",
		Type:   "risk_event",
		URL:    webhookURL,
		Events: string(mustMarshalJSON([]string{"risk.detected"})),
		Active: true,
	})
}

func (e *RiskRuleEngineV2) addToBlacklist(ctx *model.RiskContext, params map[string]interface{}) {
	blacklistType := params["type"].(string)
	reason := params["reason"].(string)
	duration := int(params["duration"].(float64))
	
	bl := &models.Blacklist{
		Target:  ctx.IPAddress,
		Type:    blacklistType,
		Source:  "automated",
		Reason:  reason,
		Action:  "block",
		Status:  "active",
	}
	
	if duration > 0 {
		bl.Expiration = time.Now().Add(time.Duration(duration) * time.Hour).Format("2006-01-02 15:04:05")
	}
	
	e.db.Create(bl)
}

func (e *RiskRuleEngineV2) recordResponse(response *RiskRuleAutomatedResponse, ctx *model.RiskContext) {
	e.db.Create(&models.RiskRuleTriggerHistory{
		RuleID:        0,
		RuleName:      "Automated Response",
		SessionID:     ctx.SessionID,
		ApplicationID: ctx.ApplicationID,
		IPAddress:     ctx.IPAddress,
		TriggerResult: true,
		ActionTaken:   response.Decision,
		ExecutionTime: time.Since(response.Timestamp).Milliseconds(),
	})
}

func (e *RiskRuleEngineV2) CreateWorkflow(workflow *Workflow) error {
	definition, _ := json.Marshal(workflow)
	
	w := &models.Workflow{
		ID:         workflow.ID,
		Name:       workflow.Name,
		Description: workflow.Description,
		TriggerType: workflow.Trigger.Type,
		Definition:  string(definition),
		Status:     "active",
	}
	
	return e.db.Create(w).Error
}

func (e *RiskRuleEngineV2) TriggerWorkflow(workflowID string, eventData map[string]interface{}) error {
	event := &WorkflowEvent{
		ID:         fmt.Sprintf("evt_%d", time.Now().UnixNano()),
		WorkflowID: workflowID,
		Type:       "manual",
		Data:       eventData,
		Timestamp:  time.Now(),
	}
	
	e.workflowEngine.eventQueue <- event
	return nil
}

func (e *RiskRuleEngineV2) GetWorkflowExecutions(workflowID string, page, pageSize int) ([]models.WorkflowExecution, int64, error) {
	var executions []models.WorkflowExecution
	var total int64
	
	query := e.db.Model(&models.WorkflowExecution{}).Where("workflow_id = ?", workflowID)
	
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	
	offset := (page - 1) * pageSize
	if err := query.Order("started_at DESC").Offset(offset).Limit(pageSize).Find(&executions).Error; err != nil {
		return nil, 0, err
	}
	
	return executions, total, nil
}

func riskRuleEngineToFloat64(v interface{}) float64 {
	switch val := v.(type) {
	case float64:
		return val
	case float32:
		return float64(val)
	case int:
		return float64(val)
	case int64:
		return float64(val)
	case string:
		f, _ := parseFloat(val)
		return f
	default:
		return 0
	}
}

func parseFloat(s string) (float64, error) {
	var f float64
	_, err := fmt.Sscanf(s, "%f", &f)
	return f, err
}

func mustMarshalJSON(v interface{}) []byte {
	b, _ := json.Marshal(v)
	return b
}
