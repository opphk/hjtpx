package service

import (
	"encoding/json"
	"fmt"
	"sync"
	"time"
)

// TestResult 测试结果
type TestResult struct {
	ID            string                 `json:"id"`
	RuleID        string                 `json:"rule_id"`
	RuleName      string                 `json:"rule_name"`
	InputData     map[string]interface{} `json:"input_data"`
	IsBot         bool                   `json:"is_bot"`
	TotalScore    float64                `json:"total_score"`
	RiskLevel     string                 `json:"risk_level"`
	Confidence    float64                `json:"confidence"`
	TriggeredRules []string              `json:"triggered_rules"`
	Recommendations []string             `json:"recommendations"`
	AnalysisTime  int64                  `json:"analysis_time_ms"`
	MatchedConditions int                `json:"matched_conditions"`
	TotalConditions   int                `json:"total_conditions"`
	CreatedAt     time.Time              `json:"created_at"`
}

// TestCase 测试用例
type TestCase struct {
	ID         string                 `json:"id"`
	Name       string                 `json:"name"`
	Description string                `json:"description"`
	InputData  map[string]interface{} `json:"input_data"`
	ExpectedResult bool               `json:"expected_result"`
	ExpectedScore float64             `json:"expected_score"`
	Enabled    bool                   `json:"enabled"`
	CreatedAt  time.Time              `json:"created_at"`
}

// RuleSandbox 规则测试沙箱
type RuleSandbox struct {
	testHistory   map[string]*TestResult
	testCases     map[string]*TestCase
	combinator    *RuleCombinator
	ruleEngine    *EnhancedRuleEngine
	mu            sync.RWMutex
	maxHistorySize int
}

func NewRuleSandbox(combinator *RuleCombinator, ruleEngine *EnhancedRuleEngine) *RuleSandbox {
	return &RuleSandbox{
		testHistory:   make(map[string]*TestResult),
		testCases:     make(map[string]*TestCase),
		combinator:    combinator,
		ruleEngine:    ruleEngine,
		maxHistorySize: 100,
	}
}

// RunTest 运行测试
func (rs *RuleSandbox) RunTest(ruleID string, inputData map[string]interface{}) (*TestResult, error) {
	startTime := time.Now()

	result := &TestResult{
		ID:        fmt.Sprintf("test_%d_%d", time.Now().Unix(), len(rs.testHistory)),
		RuleID:    ruleID,
		InputData: inputData,
	}

	// 获取规则信息
	rule, err := rs.combinator.GetCombinedRule(ruleID)
	if err != nil {
		return nil, err
	}
	result.RuleName = rule.Name

	// 评估规则
	triggered, err := rs.combinator.EvaluateRule(ruleID, inputData)
	if err != nil {
		return nil, err
	}

	result.IsBot = triggered
	result.TotalScore = calculateScoreFromResult(triggered, rule.Severity)
	result.RiskLevel = classifyRiskLevel(result.TotalScore)
	result.Confidence = calculateConfidence(triggered, inputData)
	result.AnalysisTime = time.Since(startTime).Milliseconds()

	// 生成触发规则列表
	rs.generateTriggeredRules(result, rule, inputData)

	// 生成建议
	result.Recommendations = rs.generateRecommendations(result)

	// 统计条件匹配情况
	rs.countConditions(result, rule)

	// 保存测试历史
	rs.mu.Lock()
	rs.testHistory[result.ID] = result
	if len(rs.testHistory) > rs.maxHistorySize {
		rs.cleanupOldestTest()
	}
	rs.mu.Unlock()

	return result, nil
}

// RunAllRulesTest 运行所有规则测试
func (rs *RuleSandbox) RunAllRulesTest(inputData map[string]interface{}) ([]*TestResult, error) {
	rules := rs.combinator.GetAllCombinedRules()
	results := make([]*TestResult, 0, len(rules))

	for _, rule := range rules {
		if !rule.Enabled {
			continue
		}
		result, err := rs.RunTest(rule.ID, inputData)
		if err != nil {
			return nil, err
		}
		results = append(results, result)
	}

	return results, nil
}

// RunTestCase 运行测试用例
func (rs *RuleSandbox) RunTestCase(testCaseID string) (*TestResult, error) {
	rs.mu.RLock()
	testCase, exists := rs.testCases[testCaseID]
	rs.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("测试用例不存在: %s", testCaseID)
	}

	if !testCase.Enabled {
		return nil, fmt.Errorf("测试用例已禁用: %s", testCaseID)
	}

	// 运行所有规则测试
	results, err := rs.RunAllRulesTest(testCase.InputData)
	if err != nil {
		return nil, err
	}

	if len(results) > 0 {
		return results[0], nil
	}
	return nil, fmt.Errorf("没有可用的规则")
}

// RunAllTestCases 运行所有测试用例
func (rs *RuleSandbox) RunAllTestCases() ([]TestCaseResult, error) {
	rs.mu.RLock()
	testCases := make([]*TestCase, 0, len(rs.testCases))
	for _, tc := range rs.testCases {
		if tc.Enabled {
			testCases = append(testCases, tc)
		}
	}
	rs.mu.RUnlock()

	results := make([]TestCaseResult, 0, len(testCases))
	for _, tc := range testCases {
		result, err := rs.RunTestCase(tc.ID)
		if err != nil {
			results = append(results, TestCaseResult{
				TestCaseID: tc.ID,
				TestCaseName: tc.Name,
				Passed:      false,
				ErrorMessage: err.Error(),
			})
			continue
		}

		passed := rs.verifyTestCase(tc, result)
		results = append(results, TestCaseResult{
			TestCaseID:   tc.ID,
			TestCaseName: tc.Name,
			Passed:       passed,
			ActualScore:  result.TotalScore,
			ExpectedScore: tc.ExpectedScore,
			ActualResult: result.IsBot,
			ExpectedResult: tc.ExpectedResult,
		})
	}

	return results, nil
}

// TestCaseResult 测试用例执行结果
type TestCaseResult struct {
	TestCaseID    string  `json:"test_case_id"`
	TestCaseName  string  `json:"test_case_name"`
	Passed        bool    `json:"passed"`
	ActualScore   float64 `json:"actual_score"`
	ExpectedScore float64 `json:"expected_score"`
	ActualResult  bool    `json:"actual_result"`
	ExpectedResult bool   `json:"expected_result"`
	ErrorMessage  string  `json:"error_message,omitempty"`
}

// AddTestCase 添加测试用例
func (rs *RuleSandbox) AddTestCase(testCase *TestCase) error {
	if testCase.ID == "" {
		testCase.ID = fmt.Sprintf("tc_%d", time.Now().Unix())
	}
	if testCase.Name == "" {
		return fmt.Errorf("测试用例名称不能为空")
	}
	if testCase.InputData == nil {
		testCase.InputData = make(map[string]interface{})
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.testCases[testCase.ID] = testCase
	return nil
}

// RemoveTestCase 移除测试用例
func (rs *RuleSandbox) RemoveTestCase(testCaseID string) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if _, exists := rs.testCases[testCaseID]; !exists {
		return fmt.Errorf("测试用例不存在: %s", testCaseID)
	}

	delete(rs.testCases, testCaseID)
	return nil
}

// GetTestCase 获取测试用例
func (rs *RuleSandbox) GetTestCase(testCaseID string) (*TestCase, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	testCase, exists := rs.testCases[testCaseID]
	if !exists {
		return nil, fmt.Errorf("测试用例不存在: %s", testCaseID)
	}
	return testCase, nil
}

// GetAllTestCases 获取所有测试用例
func (rs *RuleSandbox) GetAllTestCases() []*TestCase {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	testCases := make([]*TestCase, 0, len(rs.testCases))
	for _, tc := range rs.testCases {
		testCases = append(testCases, tc)
	}
	return testCases
}

// GetTestHistory 获取测试历史
func (rs *RuleSandbox) GetTestHistory(limit int) []*TestResult {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	history := make([]*TestResult, 0, len(rs.testHistory))
	for _, result := range rs.testHistory {
		history = append(history, result)
	}

	// 按时间排序（最新的在前）
	for i := len(history) - 1; i > 0; i-- {
		for j := 0; j < i; j++ {
			if history[j].CreatedAt.Before(history[j+1].CreatedAt) {
				history[j], history[j+1] = history[j+1], history[j]
			}
		}
	}

	if limit > 0 && limit < len(history) {
		return history[:limit]
	}
	return history
}

// ClearTestHistory 清空测试历史
func (rs *RuleSandbox) ClearTestHistory() {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	rs.testHistory = make(map[string]*TestResult)
}

// GenerateSampleData 生成示例测试数据
func (rs *RuleSandbox) GenerateSampleData(dataType string) map[string]interface{} {
	switch dataType {
	case "bot":
		return map[string]interface{}{
			"average_speed":    2500.0,
			"path_efficiency": 0.99,
			"click_regularity": 0.999,
			"hesitation_time": 20.0,
			"ml_score":        0.92,
			"anomaly_score":   0.88,
			"smoothness":      0.98,
			"speed_variance":  0.0001,
			"max_speed":       3500.0,
		}
	case "human":
		return map[string]interface{}{
			"average_speed":    350.0,
			"path_efficiency": 0.82,
			"click_regularity": 0.72,
			"hesitation_time": 450.0,
			"ml_score":        0.15,
			"anomaly_score":   0.12,
			"smoothness":      0.68,
			"speed_variance":  0.05,
			"max_speed":       800.0,
		}
	case "suspicious":
		return map[string]interface{}{
			"average_speed":    1200.0,
			"path_efficiency": 0.91,
			"click_regularity": 0.85,
			"hesitation_time": 80.0,
			"ml_score":        0.55,
			"anomaly_score":   0.45,
			"smoothness":      0.75,
			"speed_variance":  0.02,
			"max_speed":       1500.0,
		}
	default:
		return map[string]interface{}{
			"average_speed":    500.0,
			"path_efficiency": 0.75,
			"click_regularity": 0.65,
			"hesitation_time": 200.0,
			"ml_score":        0.3,
			"anomaly_score":   0.25,
			"smoothness":      0.6,
		}
	}
}

// ExportTestCases 导出测试用例
func (rs *RuleSandbox) ExportTestCases() (string, error) {
	rs.mu.RLock()
	defer rs.mu.RUnlock()

	testCases := make([]*TestCase, 0, len(rs.testCases))
	for _, tc := range rs.testCases {
		testCases = append(testCases, tc)
	}

	data, err := json.MarshalIndent(testCases, "", "  ")
	if err != nil {
		return "", err
	}
	return string(data), nil
}

// ImportTestCases 导入测试用例
func (rs *RuleSandbox) ImportTestCases(jsonData string) error {
	var testCases []*TestCase
	if err := json.Unmarshal([]byte(jsonData), &testCases); err != nil {
		return err
	}

	rs.mu.Lock()
	defer rs.mu.Unlock()

	for _, tc := range testCases {
		if tc.ID == "" {
			tc.ID = fmt.Sprintf("tc_%d", time.Now().UnixNano())
		}
		tc.CreatedAt = time.Now()
		rs.testCases[tc.ID] = tc
	}

	return nil
}

// cleanupOldestTest 清理最老的测试记录
func (rs *RuleSandbox) cleanupOldestTest() {
	oldestID := ""
	var oldestTime time.Time

	for id, result := range rs.testHistory {
		if oldestTime.IsZero() || result.CreatedAt.Before(oldestTime) {
			oldestTime = result.CreatedAt
			oldestID = id
		}
	}

	if oldestID != "" {
		delete(rs.testHistory, oldestID)
	}
}

// verifyTestCase 验证测试用例结果
func (rs *RuleSandbox) verifyTestCase(testCase *TestCase, result *TestResult) bool {
	// 检查结果是否匹配
	if testCase.ExpectedResult != result.IsBot {
		return false
	}

	// 检查分数是否在容忍范围内
	scoreDiff := result.TotalScore - testCase.ExpectedScore
	if scoreDiff < -0.1 || scoreDiff > 0.1 {
		return false
	}

	return true
}

// calculateScoreFromResult 根据结果计算分数
func calculateScoreFromResult(triggered bool, severity float64) float64 {
	if !triggered {
		return 0
	}
	return severity
}

// classifyRiskLevel 根据分数分类风险等级
func classifyRiskLevel(score float64) string {
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

// calculateConfidence 计算置信度
func calculateConfidence(triggered bool, inputData map[string]interface{}) float64 {
	confidence := 0.7

	// 根据输入数据丰富度调整置信度
	dataCount := len(inputData)
	if dataCount >= 10 {
		confidence += 0.1
	}
	if dataCount >= 15 {
		confidence += 0.1
	}

	// 如果触发了规则，置信度更高
	if triggered {
		confidence += 0.05
	}

	return confidence
}

// generateTriggeredRules 生成触发规则列表
func (rs *RuleSandbox) generateTriggeredRules(result *TestResult, rule *CombinedRule, inputData map[string]interface{}) {
	triggered := make([]string, 0)
	
	if result.IsBot && rule.ID != "" {
		triggered = append(triggered, rule.ID)
	}
	
	result.TriggeredRules = triggered
}

// generateRecommendations 生成建议
func (rs *RuleSandbox) generateRecommendations(result *TestResult) []string {
	recommendations := make([]string, 0)

	if result.TotalScore > 0.7 {
		recommendations = append(recommendations, "建议增加额外的验证步骤")
	}
	if result.TotalScore > 0.85 {
		recommendations = append(recommendations, "建议直接拒绝访问")
	}
	if len(result.TriggeredRules) >= 3 {
		recommendations = append(recommendations, "触发多条规则，建议深度分析")
	}

	return recommendations
}

// countConditions 统计条件匹配情况
func (rs *RuleSandbox) countConditions(result *TestResult, rule *CombinedRule) {
	if rule.RootGroup != nil {
		result.TotalConditions = countConditionsInGroup(rule.RootGroup)
		if result.IsBot {
			result.MatchedConditions = result.TotalConditions
		}
	}
}

// countConditionsInGroup 统计规则组中的条件数量
func countConditionsInGroup(group *RuleGroup) int {
	count := 0
	for _, cond := range group.Conditions {
		if m, ok := cond.(map[string]interface{}); ok {
			if operator, ok := m["operator"].(string); ok && isLogicOperator(operator) {
				// 是嵌套规则组
				subGroup := &RuleGroup{Operator: LogicOperator(operator)}
				if conditions, ok := m["conditions"].([]interface{}); ok {
					subGroup.Conditions = conditions
				}
				count += countConditionsInGroup(subGroup)
			} else {
				// 是条件
				count++
			}
		}
	}
	return count
}