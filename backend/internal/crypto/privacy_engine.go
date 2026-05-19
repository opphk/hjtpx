package crypto

import (
	"crypto/rand"
	"crypto/sha256"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"sync"
	"time"
)

var (
	ErrInvalidDataType      = errors.New("invalid data type for masking")
	ErrRuleNotFound         = errors.New("masking rule not found")
	ErrInvalidPattern       = errors.New("invalid regex pattern")
	ErrDataTooShort         = errors.New("data too short for partial masking")
)

type DataType string

const (
	DataTypeEmail      DataType = "email"
	DataTypePhone      DataType = "phone"
	DataTypeCreditCard DataType = "credit_card"
	DataTypeSSN        DataType = "ssn"
	DataTypeName       DataType = "name"
	DataTypeAddress    DataType = "address"
	DataTypeIPAddress  DataType = "ip_address"
	DataTypeCustom     DataType = "custom"
)

type MaskingStrategy string

const (
	StrategyFull    MaskingStrategy = "full"
	StrategyPartial MaskingStrategy = "partial"
	StrategyHash    MaskingStrategy = "hash"
	StrategyTokenize MaskingStrategy = "tokenize"
	StrategyNullify  MaskingStrategy = "nullify"
)

type MaskingRule struct {
	RuleID      string           `json:"rule_id"`
	DataType    DataType         `json:"data_type"`
	Strategy    MaskingStrategy  `json:"strategy"`
	VisibleHead int              `json:"visible_head"`
	VisibleTail int              `json:"visible_tail"`
	MaskChar    string           `json:"mask_char"`
	Pattern     string           `json:"pattern,omitempty"`
	Replacement string           `json:"replacement,omitempty"`
	Priority    int              `json:"priority"`
	Enabled     bool             `json:"enabled"`
}

type DataMask struct {
	OriginalValue string            `json:"-"`
	MaskedValue   string            `json:"masked_value"`
	MaskingType   MaskingStrategy   `json:"masking_type"`
	DataType      DataType          `json:"data_type"`
	Token         string            `json:"token,omitempty"`
	Timestamp     time.Time         `json:"timestamp"`
}

type BudgetConfig struct {
	MaxOperationsPerDay  int64     `json:"max_operations_per_day"`
	MaxDataSizePerDay    int64     `json:"max_data_size_per_day"`
	MaxBudget            int64     `json:"max_budget"`
	WarningThreshold     float64   `json:"warning_threshold"`
	ResetPeriod          time.Duration `json:"reset_period"`
}

type BudgetRecord struct {
	UserID          string    `json:"user_id"`
	OperationCount  int64     `json:"operation_count"`
	DataSizeUsed    int64     `json:"data_size_used"`
	BudgetUsed      int64     `json:"budget_used"`
	WarningSent     bool      `json:"warning_sent"`
	LastOperationAt time.Time `json:"last_operation_at"`
	ResetAt         time.Time `json:"reset_at"`
}

type BudgetAlert struct {
	AlertID      string    `json:"alert_id"`
	UserID       string    `json:"user_id"`
	AlertType    string    `json:"alert_type"`
	Message      string    `json:"message"`
	CurrentUsage float64   `json:"current_usage"`
	Threshold    float64   `json:"threshold"`
	Timestamp    time.Time `json:"timestamp"`
}

type TokenVault struct {
	mu       sync.RWMutex
	tokens   map[string]string
	reverse  map[string]string
}

type PrivacyEngine struct {
	mu            sync.RWMutex
	maskingRules  map[string]*MaskingRule
	dataTypeRules map[DataType]*MaskingRule
	budgets       map[string]*BudgetRecord
	budgetConfig  *BudgetConfig
	tokenVault    *TokenVault
	defaultLevel   PrivacyLevel
	stats         *EngineStats
}

type EngineStats struct {
	TotalOperations     int64            `json:"total_operations"`
	TotalMaskingOps      int64            `json:"total_masking_operations"`
	TotalBudgetOps       int64            `json:"total_budget_operations"`
	TotalTokensGenerated int64            `json:"total_tokens_generated"`
	AlertsTriggered      int64            `json:"alerts_triggered"`
	LastOperation        time.Time        `json:"last_operation"`
	mu                   sync.Mutex
}

type DataMaskingRequest struct {
	Data     interface{}        `json:"data"`
	DataType DataType           `json:"data_type"`
	Rules    []string           `json:"rules,omitempty"`
	Level    PrivacyLevel       `json:"level"`
	UserID   string             `json:"user_id"`
}

type DataMaskingResponse struct {
	OriginalData interface{}       `json:"original_data,omitempty"`
	MaskedData   interface{}       `json:"masked_data"`
	Tokens       []string          `json:"tokens,omitempty"`
	MaskingInfo  *MaskingInfo      `json:"masking_info"`
}

type MaskingInfo struct {
	Strategy    MaskingStrategy `json:"strategy"`
	DataType    DataType       `json:"data_type"`
	RuleID      string         `json:"rule_id"`
	Level       PrivacyLevel   `json:"level"`
	Timestamp   time.Time      `json:"timestamp"`
}

type BudgetCheckRequest struct {
	UserID     string `json:"user_id"`
	Operation  string `json:"operation"`
	DataSize   int64  `json:"data_size"`
}

type BudgetCheckResponse struct {
	Allowed    bool           `json:"allowed"`
	Remaining  int64          `json:"remaining"`
	Used       int64          `json:"used"`
	Alert      *BudgetAlert   `json:"alert,omitempty"`
	ResetAt    time.Time      `json:"reset_at"`
}

type BudgetAllocation struct {
	UserID       string    `json:"user_id"`
	Allocated    int64     `json:"allocated"`
	Used         int64     `json:"used"`
	Remaining    int64     `json:"remaining"`
	ExpiresAt    time.Time `json:"expires_at"`
}

func NewPrivacyEngine(config *BudgetConfig) *PrivacyEngine {
	if config == nil {
		config = &BudgetConfig{
			MaxOperationsPerDay: 1000,
			MaxDataSizePerDay:    10 * 1024 * 1024,
			MaxBudget:            10000,
			WarningThreshold:     0.8,
			ResetPeriod:          24 * time.Hour,
		}
	}

	engine := &PrivacyEngine{
		maskingRules:  make(map[string]*MaskingRule),
		dataTypeRules: make(map[DataType]*MaskingRule),
		budgets:       make(map[string]*BudgetRecord),
		budgetConfig:  config,
		tokenVault: &TokenVault{
			tokens:  make(map[string]string),
			reverse: make(map[string]string),
		},
		defaultLevel: PrivacyLevelMedium,
		stats:        &EngineStats{},
	}

	engine.initializeDefaultRules()

	return engine
}

func (e *PrivacyEngine) initializeDefaultRules() {
	emailRule := &MaskingRule{
		RuleID:      "default_email",
		DataType:    DataTypeEmail,
		Strategy:    StrategyPartial,
		VisibleHead: 2,
		VisibleTail: 2,
		MaskChar:    "*",
		Priority:    10,
		Enabled:     true,
	}
	e.maskingRules[emailRule.RuleID] = emailRule
	e.dataTypeRules[DataTypeEmail] = emailRule

	phoneRule := &MaskingRule{
		RuleID:      "default_phone",
		DataType:    DataTypePhone,
		Strategy:    StrategyPartial,
		VisibleHead: 3,
		VisibleTail: 4,
		MaskChar:    "*",
		Priority:    10,
		Enabled:     true,
	}
	e.maskingRules[phoneRule.RuleID] = phoneRule
	e.dataTypeRules[DataTypePhone] = phoneRule

	ccRule := &MaskingRule{
		RuleID:      "default_credit_card",
		DataType:    DataTypeCreditCard,
		Strategy:    StrategyPartial,
		VisibleHead: 0,
		VisibleTail: 4,
		MaskChar:    "*",
		Priority:    10,
		Enabled:     true,
	}
	e.maskingRules[ccRule.RuleID] = ccRule
	e.dataTypeRules[DataTypeCreditCard] = ccRule

	ssnRule := &MaskingRule{
		RuleID:      "default_ssn",
		DataType:    DataTypeSSN,
		Strategy:    StrategyPartial,
		VisibleHead: 0,
		VisibleTail: 4,
		MaskChar:    "*",
		Priority:    10,
		Enabled:     true,
	}
	e.maskingRules[ssnRule.RuleID] = ssnRule
	e.dataTypeRules[DataTypeSSN] = ssnRule

	nameRule := &MaskingRule{
		RuleID:      "default_name",
		DataType:    DataTypeName,
		Strategy:    StrategyPartial,
		VisibleHead: 1,
		VisibleTail: 0,
		MaskChar:    "*",
		Priority:    10,
		Enabled:     true,
	}
	e.maskingRules[nameRule.RuleID] = nameRule
	e.dataTypeRules[DataTypeName] = nameRule
}

func (e *PrivacyEngine) MaskData(request *DataMaskingRequest) (*DataMaskingResponse, error) {
	if request == nil {
		return nil, errors.New("invalid request")
	}

	if err := e.checkBudget(request.UserID, 1, 100); err != nil {
		return nil, err
	}

	response := &DataMaskingResponse{
		Tokens:      make([]string, 0),
		MaskingInfo: &MaskingInfo{},
	}

	switch data := request.Data.(type) {
	case string:
		masked, tokens, info := e.maskString(data, request.DataType, request.Level)
		response.OriginalData = data
		response.MaskedData = masked
		response.Tokens = tokens
		response.MaskingInfo = info
	case map[string]interface{}:
		masked, tokens, info := e.maskMap(data, request.DataType, request.Level)
		response.OriginalData = data
		response.MaskedData = masked
		response.Tokens = tokens
		response.MaskingInfo = info
	case []interface{}:
		masked, tokens, info := e.maskSlice(data, request.DataType, request.Level)
		response.OriginalData = data
		response.MaskedData = masked
		response.Tokens = tokens
		response.MaskingInfo = info
	default:
		return nil, ErrInvalidDataType
	}

	e.recordOperation()
	return response, nil
}

func (e *PrivacyEngine) maskString(data string, dataType DataType, level PrivacyLevel) (string, []string, *MaskingInfo) {
	tokens := make([]string, 0)

	rule := e.getApplicableRule(dataType, level)

	var masked string
	var strategy MaskingStrategy

	switch rule.Strategy {
	case StrategyFull:
		masked = e.applyFullMasking(data, rule.MaskChar)
		strategy = StrategyFull
	case StrategyPartial:
		masked = e.applyPartialMasking(data, rule.VisibleHead, rule.VisibleTail, rule.MaskChar)
		strategy = StrategyPartial
	case StrategyHash:
		masked = e.applyHashMasking(data)
		strategy = StrategyHash
	case StrategyTokenize:
		masked, tokens = e.applyTokenization(data)
		strategy = StrategyTokenize
	case StrategyNullify:
		masked = e.applyNullification()
		strategy = StrategyNullify
	default:
		masked = e.applyPartialMasking(data, 2, 2, "*")
		strategy = StrategyPartial
	}

	info := &MaskingInfo{
		Strategy:  strategy,
		DataType:  dataType,
		RuleID:    rule.RuleID,
		Level:     level,
		Timestamp: time.Now(),
	}

	return masked, tokens, info
}

func (e *PrivacyEngine) maskMap(data map[string]interface{}, primaryType DataType, level PrivacyLevel) (map[string]interface{}, []string, *MaskingInfo) {
	result := make(map[string]interface{})
	tokens := make([]string, 0)

	inferredType := e.inferDataType(data)
	rule := e.getApplicableRule(inferredType, level)

	for key, value := range data {
		switch v := value.(type) {
		case string:
			result[key], tokens = e.maskAndCollect(v, rule, tokens)
		case map[string]interface{}:
			masked, subTokens, _ := e.maskMap(v, primaryType, level)
			result[key] = masked
			tokens = append(tokens, subTokens...)
		case []interface{}:
			masked, subTokens, _ := e.maskSlice(v, primaryType, level)
			result[key] = masked
			tokens = append(tokens, subTokens...)
		default:
			result[key] = v
		}
	}

	info := &MaskingInfo{
		Strategy:  rule.Strategy,
		DataType:  inferredType,
		RuleID:    rule.RuleID,
		Level:     level,
		Timestamp: time.Now(),
	}

	return result, tokens, info
}

func (e *PrivacyEngine) maskSlice(data []interface{}, primaryType DataType, level PrivacyLevel) ([]interface{}, []string, *MaskingInfo) {
	result := make([]interface{}, len(data))
	tokens := make([]string, 0)

	rule := e.getApplicableRule(primaryType, level)

	for i, value := range data {
		switch v := value.(type) {
		case string:
			result[i], tokens = e.maskAndCollect(v, rule, tokens)
		default:
			result[i] = v
		}
	}

	info := &MaskingInfo{
		Strategy:  rule.Strategy,
		DataType:  primaryType,
		RuleID:    rule.RuleID,
		Level:     level,
		Timestamp: time.Now(),
	}

	return result, tokens, info
}

func (e *PrivacyEngine) maskAndCollect(data string, rule *MaskingRule, tokens []string) (string, []string) {
	switch rule.Strategy {
	case StrategyTokenize:
		masked, newTokens := e.applyTokenization(data)
		tokens = append(tokens, newTokens...)
		return masked, tokens
	case StrategyHash:
		return e.applyHashMasking(data), tokens
	default:
		return e.applyPartialMasking(data, rule.VisibleHead, rule.VisibleTail, rule.MaskChar), tokens
	}
}

func (e *PrivacyEngine) applyFullMasking(data, maskChar string) string {
	if maskChar == "" {
		maskChar = "*"
	}
	return strings.Repeat(maskChar, len(data))
}

func (e *PrivacyEngine) applyPartialMasking(data string, visibleHead, visibleTail int, maskChar string) string {
	if maskChar == "" {
		maskChar = "*"
	}

	dataLen := len(data)

	if visibleHead < 0 {
		visibleHead = 0
	}
	if visibleTail < 0 {
		visibleTail = 0
	}

	if visibleHead+visibleTail >= dataLen {
		return strings.Repeat(maskChar, dataLen)
	}

	head := ""
	if visibleHead > 0 && visibleHead < dataLen {
		head = data[:visibleHead]
	}

	tail := ""
	if visibleTail > 0 && visibleHead+visibleTail < dataLen {
		tail = data[dataLen-visibleTail:]
	}

	maskLen := dataLen - visibleHead - visibleTail
	if maskLen < 0 {
		maskLen = 0
	}

	return head + strings.Repeat(maskChar, maskLen) + tail
}

func (e *PrivacyEngine) applyHashMasking(data string) string {
	hash := sha256.Sum256([]byte(data))
	return base64.StdEncoding.EncodeToString(hash[:16])
}

func (e *PrivacyEngine) applyTokenization(data string) (string, []string) {
	token, _ := e.tokenVault.Add(data)
	return token, []string{token}
}

func (e *PrivacyEngine) applyNullification() string {
	return "null"
}

func (e *PrivacyEngine) inferDataType(data map[string]interface{}) DataType {
	for key := range data {
		keyLower := strings.ToLower(key)
		if strings.Contains(keyLower, "email") {
			return DataTypeEmail
		}
		if strings.Contains(keyLower, "phone") || strings.Contains(keyLower, "mobile") {
			return DataTypePhone
		}
		if strings.Contains(keyLower, "credit") || strings.Contains(keyLower, "card") {
			return DataTypeCreditCard
		}
		if strings.Contains(keyLower, "ssn") || strings.Contains(keyLower, "social") {
			return DataTypeSSN
		}
		if strings.Contains(keyLower, "name") || strings.Contains(keyLower, "first") || strings.Contains(keyLower, "last") {
			return DataTypeName
		}
	}
	return DataTypeCustom
}

func (e *PrivacyEngine) getApplicableRule(dataType DataType, level PrivacyLevel) *MaskingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if rule, ok := e.dataTypeRules[dataType]; ok && rule.Enabled {
		return rule
	}

	defaultRule := &MaskingRule{
		RuleID:      "default_custom",
		DataType:    dataType,
		Strategy:    StrategyPartial,
		VisibleHead: 2,
		VisibleTail: 2,
		MaskChar:    "*",
		Priority:    0,
		Enabled:     true,
	}

	return defaultRule
}

func (e *PrivacyEngine) AddMaskingRule(rule *MaskingRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if rule == nil || rule.RuleID == "" {
		return errors.New("invalid rule")
	}

	if rule.MaskChar == "" {
		rule.MaskChar = "*"
	}

	rule.Enabled = true
	e.maskingRules[rule.RuleID] = rule
	e.dataTypeRules[rule.DataType] = rule

	return nil
}

func (e *PrivacyEngine) RemoveMaskingRule(ruleID string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	rule, exists := e.maskingRules[ruleID]
	if !exists {
		return ErrRuleNotFound
	}

	delete(e.maskingRules, ruleID)
	delete(e.dataTypeRules, rule.DataType)

	return nil
}

func (e *PrivacyEngine) GetMaskingRule(ruleID string) *MaskingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.maskingRules[ruleID]
}

func (e *PrivacyEngine) ListMaskingRules() []*MaskingRule {
	e.mu.RLock()
	defer e.mu.RUnlock()

	rules := make([]*MaskingRule, 0, len(e.maskingRules))
	for _, rule := range e.maskingRules {
		rules = append(rules, rule)
	}

	return rules
}

func (e *PrivacyEngine) UpdateMaskingRule(rule *MaskingRule) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if _, exists := e.maskingRules[rule.RuleID]; !exists {
		return ErrRuleNotFound
	}

	e.maskingRules[rule.RuleID] = rule
	e.dataTypeRules[rule.DataType] = rule

	return nil
}

func (e *PrivacyEngine) SetDefaultPrivacyLevel(level PrivacyLevel) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.defaultLevel = level
}

func (e *PrivacyEngine) GetDefaultPrivacyLevel() PrivacyLevel {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.defaultLevel
}

func (e *PrivacyEngine) CheckBudget(request *BudgetCheckRequest) (*BudgetCheckResponse, error) {
	response := &BudgetCheckResponse{}

	e.mu.RLock()
	budget, exists := e.budgets[request.UserID]
	e.mu.RUnlock()

	if !exists {
		budget = &BudgetRecord{
			UserID:    request.UserID,
			ResetAt:   time.Now().Add(e.budgetConfig.ResetPeriod),
		}
		e.mu.Lock()
		e.budgets[request.UserID] = budget
		e.mu.Unlock()
	}

	if time.Now().After(budget.ResetAt) {
		e.resetBudget(request.UserID)
		budget, _ = e.budgets[request.UserID]
	}

	remaining := e.budgetConfig.MaxBudget - budget.BudgetUsed

	if budget.OperationCount >= e.budgetConfig.MaxOperationsPerDay {
		response.Allowed = false
		response.Remaining = 0
		response.Used = budget.OperationCount
		response.ResetAt = budget.ResetAt
		return response, nil
	}

	if budget.BudgetUsed >= e.budgetConfig.MaxBudget {
		response.Allowed = false
		response.Remaining = 0
		response.Used = budget.BudgetUsed
		response.ResetAt = budget.ResetAt
		return nil, ErrBudgetExceeded
	}

	usageRatio := float64(budget.BudgetUsed) / float64(e.budgetConfig.MaxBudget)
	if usageRatio >= e.budgetConfig.WarningThreshold && !budget.WarningSent {
		response.Alert = &BudgetAlert{
			AlertID:      generateAlertID(),
			UserID:       request.UserID,
			AlertType:    "budget_warning",
			Message:      fmt.Sprintf("Budget usage at %.0f%%", usageRatio*100),
			CurrentUsage: usageRatio,
			Threshold:    e.budgetConfig.WarningThreshold,
			Timestamp:    time.Now(),
		}
		budget.WarningSent = true
		e.stats.mu.Lock()
		e.stats.AlertsTriggered++
		e.stats.mu.Unlock()
	}

	response.Allowed = true
	response.Remaining = remaining
	response.Used = budget.BudgetUsed
	response.ResetAt = budget.ResetAt

	return response, nil
}

func (e *PrivacyEngine) resetBudget(userID string) {
	e.mu.Lock()
	defer e.mu.Unlock()

	if budget, exists := e.budgets[userID]; exists {
		budget.OperationCount = 0
		budget.BudgetUsed = 0
		budget.DataSizeUsed = 0
		budget.WarningSent = false
		budget.ResetAt = time.Now().Add(e.budgetConfig.ResetPeriod)
	}
}

func (e *PrivacyEngine) checkBudget(userID string, operations int64, dataSize int64) error {
	request := &BudgetCheckRequest{
		UserID:    userID,
		Operation: "masking",
		DataSize:  dataSize,
	}

	response, err := e.CheckBudget(request)
	if err != nil {
		return err
	}

	if !response.Allowed {
		return ErrBudgetExceeded
	}

	e.mu.Lock()
	if budget, exists := e.budgets[userID]; exists {
		budget.OperationCount += operations
		budget.BudgetUsed += operations
		budget.DataSizeUsed += dataSize
		budget.LastOperationAt = time.Now()
	}
	e.mu.Unlock()

	return nil
}

func (e *PrivacyEngine) recordOperation() {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	e.stats.TotalOperations++
	e.stats.TotalMaskingOps++
	e.stats.LastOperation = time.Now()
}

func (e *PrivacyEngine) GetStats() *EngineStats {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	return e.stats
}

func (e *PrivacyEngine) ResetStats() {
	e.stats.mu.Lock()
	defer e.stats.mu.Unlock()

	e.stats.TotalOperations = 0
	e.stats.TotalMaskingOps = 0
	e.stats.TotalBudgetOps = 0
	e.stats.TotalTokensGenerated = 0
	e.stats.AlertsTriggered = 0
}

func (tv *TokenVault) Add(data string) (string, error) {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	if token, exists := tv.reverse[data]; exists {
		return token, nil
	}

	tokenBytes := make([]byte, 32)
	if _, err := rand.Read(tokenBytes); err != nil {
		return "", fmt.Errorf("failed to generate token: %w", err)
	}

	token := base64.URLEncoding.EncodeToString(tokenBytes)

	tv.tokens[token] = data
	tv.reverse[data] = token

	return token, nil
}

func (tv *TokenVault) Get(token string) (string, bool) {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	data, exists := tv.tokens[token]
	return data, exists
}

func (tv *TokenVault) Remove(token string) bool {
	tv.mu.Lock()
	defer tv.mu.Unlock()

	if data, exists := tv.tokens[token]; exists {
		delete(tv.tokens, token)
		delete(tv.reverse, data)
		return true
	}

	return false
}

func (tv *TokenVault) Count() int {
	tv.mu.RLock()
	defer tv.mu.RUnlock()

	return len(tv.tokens)
}

func (e *PrivacyEngine) CreateCustomRule(ruleID string, pattern string, replacement string) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	_, err := regexp.Compile(pattern)
	if err != nil {
		return ErrInvalidPattern
	}

	rule := &MaskingRule{
		RuleID:      ruleID,
		DataType:    DataTypeCustom,
		Strategy:    StrategyPartial,
		Pattern:     pattern,
		Replacement: replacement,
		Priority:    5,
		Enabled:     true,
	}

	e.maskingRules[ruleID] = rule

	return nil
}

func (e *PrivacyEngine) ApplyCustomMasking(data string, ruleID string) (string, error) {
	e.mu.RLock()
	rule, exists := e.maskingRules[ruleID]
	e.mu.RUnlock()

	if !exists {
		return "", ErrRuleNotFound
	}

	if rule.Pattern == "" {
		return data, nil
	}

	re := regexp.MustCompile(rule.Pattern)
	return re.ReplaceAllString(data, rule.Replacement), nil
}

func (e *PrivacyEngine) GetBudgetRecord(userID string) *BudgetRecord {
	e.mu.RLock()
	defer e.mu.RUnlock()

	budget, exists := e.budgets[userID]
	if !exists {
		return &BudgetRecord{
			UserID:    userID,
			ResetAt:   time.Now().Add(e.budgetConfig.ResetPeriod),
		}
	}

	return budget
}

func (e *PrivacyEngine) AllocateBudget(allocation *BudgetAllocation) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	if budget, exists := e.budgets[allocation.UserID]; exists {
		budget.BudgetUsed -= allocation.Allocated
		if budget.BudgetUsed < 0 {
			budget.BudgetUsed = 0
		}
	}

	return nil
}

func (e *PrivacyEngine) ExportConfig() ([]byte, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	config := struct {
		Rules       []*MaskingRule `json:"rules"`
		BudgetConfig *BudgetConfig  `json:"budget_config"`
	}{
		Rules:        e.ListMaskingRules(),
		BudgetConfig: e.budgetConfig,
	}

	return json.MarshalIndent(config, "", "  ")
}

func (e *PrivacyEngine) ImportConfig(data []byte) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	var config struct {
		Rules        []*MaskingRule `json:"rules"`
		BudgetConfig *BudgetConfig  `json:"budget_config"`
	}

	if err := json.Unmarshal(data, &config); err != nil {
		return fmt.Errorf("failed to unmarshal config: %w", err)
	}

	for _, rule := range config.Rules {
		e.maskingRules[rule.RuleID] = rule
		e.dataTypeRules[rule.DataType] = rule
	}

	if config.BudgetConfig != nil {
		e.budgetConfig = config.BudgetConfig
	}

	return nil
}

func (e *PrivacyEngine) MaskCreditCard(cc string) string {
	if len(cc) < 4 {
		return strings.Repeat("*", len(cc))
	}
	return strings.Repeat("*", len(cc)-4) + cc[len(cc)-4:]
}

func (e *PrivacyEngine) MaskEmail(email string) string {
	parts := strings.Split(email, "@")
	if len(parts) != 2 {
		return e.applyFullMasking(email, "*")
	}

	local := parts[0]
	domain := parts[1]

	if len(local) <= 2 {
		return e.applyFullMasking(local, "*") + "@" + domain
	}

	return local[:1] + strings.Repeat("*", len(local)-2) + local[len(local)-1:] + "@" + domain
}

func (e *PrivacyEngine) MaskPhone(phone string) string {
	digits := extractDigits(phone)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}
	return strings.Repeat("*", len(digits)-4) + digits[len(digits)-4:]
}

func (e *PrivacyEngine) MaskSSN(ssn string) string {
	digits := extractDigits(ssn)
	if len(digits) < 4 {
		return strings.Repeat("*", len(digits))
	}
	return "***-**-" + digits[len(digits)-4:]
}

func extractDigits(s string) string {
	var result strings.Builder
	for _, c := range s {
		if c >= '0' && c <= '9' {
			result.WriteRune(c)
		}
	}
	return result.String()
}

func generateAlertID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return "alert_" + base64.URLEncoding.EncodeToString(bytes)
}

func (e *PrivacyEngine) BatchMask(requests []*DataMaskingRequest) ([]*DataMaskingResponse, error) {
	results := make([]*DataMaskingResponse, 0, len(requests))

	for _, req := range requests {
		resp, err := e.MaskData(req)
		if err != nil {
			results = append(results, &DataMaskingResponse{
				MaskedData:  nil,
				MaskingInfo: &MaskingInfo{
					Strategy: StrategyNullify,
					Timestamp: time.Now(),
				},
			})
		} else {
			results = append(results, resp)
		}
	}

	return results, nil
}

func (e *PrivacyEngine) ValidateMaskingRule(rule *MaskingRule) error {
	if rule == nil {
		return errors.New("rule is nil")
	}

	if rule.RuleID == "" {
		return errors.New("rule ID is required")
	}

	if rule.DataType == "" {
		return errors.New("data type is required")
	}

	if rule.Strategy == "" {
		return errors.New("strategy is required")
	}

	if rule.Pattern != "" {
		_, err := regexp.Compile(rule.Pattern)
		if err != nil {
			return ErrInvalidPattern
		}
	}

	return nil
}

func ParseDataType(s string) (DataType, error) {
	switch strings.ToLower(s) {
	case "email":
		return DataTypeEmail, nil
	case "phone":
		return DataTypePhone, nil
	case "credit_card", "creditcard", "cc":
		return DataTypeCreditCard, nil
	case "ssn", "social":
		return DataTypeSSN, nil
	case "name":
		return DataTypeName, nil
	case "address":
		return DataTypeAddress, nil
	case "ip", "ipaddress":
		return DataTypeIPAddress, nil
	default:
		return DataTypeCustom, nil
	}
}

func ParseMaskingStrategy(s string) (MaskingStrategy, error) {
	switch strings.ToLower(s) {
	case "full":
		return StrategyFull, nil
	case "partial":
		return StrategyPartial, nil
	case "hash":
		return StrategyHash, nil
	case "tokenize":
		return StrategyTokenize, nil
	case "nullify":
		return StrategyNullify, nil
	default:
		return "", fmt.Errorf("unknown strategy: %s", s)
	}
}

func (e *PrivacyEngine) SetBudgetConfig(config *BudgetConfig) {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.budgetConfig = config
}

func (e *PrivacyEngine) GetBudgetConfig() *BudgetConfig {
	e.mu.RLock()
	defer e.mu.RUnlock()

	return e.budgetConfig
}

func (e *PrivacyEngine) ClearTokenVault() {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.tokenVault.tokens = make(map[string]string)
	e.tokenVault.reverse = make(map[string]string)
}

func (e *PrivacyEngine) GetTokenVaultSize() int {
	return e.tokenVault.Count()
}
