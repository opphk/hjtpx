package service

import (
	"encoding/json"
	"fmt"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestRiskEngine_Evaluate(t *testing.T) {
	t.Run("BasicConditionEvaluation", func(t *testing.T) {
		config := &RiskConfig{
			ID:           "test_config",
			Name:         "Test Config",
			DefaultScore: 10,
			Threshold:   50,
			Timeout:     100 * time.Millisecond,
			Expressions: []RiskExpression{
				{
					ID:       "speed_rule",
					Name:     "Speed Check",
					Type:     RiskRuleTypeCondition,
					Priority: 100,
					Weight:   0.5,
					Enabled:  true,
					Condition: &RiskCondition{
						Field:    "speed",
						Operator: RiskOperatorGt,
						Value:    1000,
					},
				},
			},
		}

		engine := NewRiskEngine(config)

		t.Run("TriggeredRule", func(t *testing.T) {
			ctx := &RiskContext{
				Data: map[string]interface{}{
					"speed": float64(2000),
				},
			}

			result, err := engine.Evaluate(ctx)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.True(t, len(result.TriggeredRules) > 0)
			assert.Greater(t, result.TotalScore, float64(0))
		})

		t.Run("NonTriggeredRule", func(t *testing.T) {
			ctx := &RiskContext{
				Data: map[string]interface{}{
					"speed": float64(500),
				},
			}

			result, err := engine.Evaluate(ctx)
			require.NoError(t, err)
			assert.NotNil(t, result)
			assert.Equal(t, 0, len(result.TriggeredRules))
		})
	})

	t.Run("MultipleRules", func(t *testing.T) {
		config := &RiskConfig{
			ID:           "multi_config",
			Name:         "Multiple Rules Config",
			DefaultScore:  10,
			Threshold:    50,
			Timeout:      100 * time.Millisecond,
			Expressions: []RiskExpression{
				{
					ID:       "rule1",
					Name:     "Rule 1",
					Type:     RiskRuleTypeCondition,
					Priority: 100,
					Weight:   0.3,
					Enabled:  true,
					Condition: &RiskCondition{
						Field:    "speed",
						Operator: RiskOperatorGt,
						Value:    1000,
					},
				},
				{
					ID:       "rule2",
					Name:     "Rule 2",
					Type:     RiskRuleTypeCondition,
					Priority: 90,
					Weight:   0.4,
					Enabled:  true,
					Condition: &RiskCondition{
						Field:    "acceleration",
						Operator: RiskOperatorLt,
						Value:    0.5,
					},
				},
				{
					ID:       "rule3",
					Name:     "Rule 3",
					Type:     RiskRuleTypeCondition,
					Priority: 80,
					Weight:   0.3,
					Enabled:  true,
					Condition: &RiskCondition{
						Field:    "trajectory",
						Operator: RiskOperatorEq,
						Value:    "smooth",
					},
				},
			},
		}

		engine := NewRiskEngine(config)

		ctx := &RiskContext{
			Data: map[string]interface{}{
				"speed":       float64(2000),
				"acceleration": float64(0.2),
				"trajectory":  "smooth",
			},
		}

		result, err := engine.Evaluate(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Equal(t, 3, len(result.TriggeredRules))
	})
}

func TestRiskEngine_Operators(t *testing.T) {
	testCases := []struct {
		name     string
		operator RiskOperatorType
		field    string
		value    interface{}
		testData map[string]interface{}
		expected bool
	}{
		{
			name:     "Equal Operator - Match",
			operator: RiskOperatorEq,
			field:    "status",
			value:    "active",
			testData: map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "Equal Operator - No Match",
			operator: RiskOperatorEq,
			field:    "status",
			value:    "inactive",
			testData: map[string]interface{}{"status": "active"},
			expected: false,
		},
		{
			name:     "Not Equal Operator",
			operator: RiskOperatorNe,
			field:    "status",
			value:    "blocked",
			testData: map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "Greater Than Operator - Match",
			operator: RiskOperatorGt,
			field:    "score",
			value:    float64(50),
			testData: map[string]interface{}{"score": float64(75)},
			expected: true,
		},
		{
			name:     "Greater Than Operator - No Match",
			operator: RiskOperatorGt,
			field:    "score",
			value:    float64(50),
			testData: map[string]interface{}{"score": float64(25)},
			expected: false,
		},
		{
			name:     "Greater Than or Equal Operator",
			operator: RiskOperatorGte,
			field:    "score",
			value:    float64(50),
			testData: map[string]interface{}{"score": float64(50)},
			expected: true,
		},
		{
			name:     "Less Than Operator",
			operator: RiskOperatorLt,
			field:    "score",
			value:    float64(50),
			testData: map[string]interface{}{"score": float64(25)},
			expected: true,
		},
		{
			name:     "Less Than or Equal Operator",
			operator: RiskOperatorLte,
			field:    "score",
			value:    float64(50),
			testData: map[string]interface{}{"score": float64(50)},
			expected: true,
		},
		{
			name:     "Contains Operator - Match",
			operator: RiskOperatorContains,
			field:    "email",
			value:    "@gmail.com",
			testData: map[string]interface{}{"email": "user@gmail.com"},
			expected: true,
		},
		{
			name:     "Contains Operator - No Match",
			operator: RiskOperatorContains,
			field:    "email",
			value:    "@yahoo.com",
			testData: map[string]interface{}{"email": "user@gmail.com"},
			expected: false,
		},
		{
			name:     "Not Contains Operator",
			operator: RiskOperatorNotContain,
			field:    "email",
			value:    "@spam.com",
			testData: map[string]interface{}{"email": "user@gmail.com"},
			expected: true,
		},
		{
			name:     "In Operator - Match",
			operator: RiskOperatorIn,
			field:    "status",
			value:    nil,
			testData: map[string]interface{}{"status": "active"},
			expected: true,
		},
		{
			name:     "Not In Operator - Match",
			operator: RiskOperatorNotIn,
			field:    "status",
			value:    nil,
			testData: map[string]interface{}{"status": "unknown"},
			expected: true,
		},
		{
			name:     "Starts With Operator",
			operator: RiskOperatorStartsWith,
			field:    "name",
			value:    "John",
			testData: map[string]interface{}{"name": "John Doe"},
			expected: true,
		},
		{
			name:     "Ends With Operator",
			operator: RiskOperatorEndsWith,
			field:    "email",
			value:    ".com",
			testData: map[string]interface{}{"email": "user@example.com"},
			expected: true,
		},
		{
			name:     "Regex Operator - Match",
			operator: RiskOperatorRegex,
			field:    "ip",
			value:    `^\d{1,3}\.\d{1,3}\.\d{1,3}\.\d{1,3}$`,
			testData: map[string]interface{}{"ip": "192.168.1.1"},
			expected: true,
		},
		{
			name:     "Between Operator - Match",
			operator: RiskOperatorBetween,
			field:    "score",
			value:    nil,
			testData: map[string]interface{}{"score": float64(50)},
			expected: true,
		},
		{
			name:     "Is Empty Operator - Empty String",
			operator: RiskOperatorIsEmpty,
			field:    "comment",
			value:    nil,
			testData: map[string]interface{}{"comment": ""},
			expected: true,
		},
		{
			name:     "Is Not Empty Operator",
			operator: RiskOperatorIsNotEmpty,
			field:    "comment",
			value:    nil,
			testData: map[string]interface{}{"comment": "some text"},
			expected: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			var values []interface{}
			if tc.operator == RiskOperatorIn || tc.operator == RiskOperatorNotIn || tc.operator == RiskOperatorBetween {
				values = []interface{}{"active", "pending", "approved"}
				if tc.operator == RiskOperatorBetween {
					values = []interface{}{float64(0), float64(100)}
				}
			}

			config := &RiskConfig{
				ID:           "operator_test",
				Name:         "Operator Test",
				DefaultScore: 10,
				Timeout:      100 * time.Millisecond,
				Expressions: []RiskExpression{
					{
						ID:       "test_rule",
						Name:     "Test Rule",
						Type:     RiskRuleTypeCondition,
						Priority: 100,
						Weight:   1.0,
						Enabled:  true,
						Condition: &RiskCondition{
							Field:    tc.field,
							Operator: tc.operator,
							Value:    tc.value,
							Values:   values,
						},
					},
				},
			}

			engine := NewRiskEngine(config)
			ctx := &RiskContext{Data: tc.testData}
			result, err := engine.Evaluate(ctx)

			require.NoError(t, err)
			assert.NotNil(t, result)

			triggered := len(result.TriggeredRules) > 0
			assert.Equal(t, tc.expected, triggered, "Operator: %s, Field: %s, Value: %v", tc.operator, tc.field, tc.value)
		})
	}
}

func TestRiskEngine_LogicOperators(t *testing.T) {
	t.Run("AND Group - All Match", func(t *testing.T) {
		config := &RiskConfig{
			ID:           "and_test",
			Name:         "AND Test",
			DefaultScore: 10,
			Timeout:      100 * time.Millisecond,
			Expressions: []RiskExpression{
				{
					ID:       "and_group",
					Name:     "AND Group",
					Type:     RiskRuleTypeGroup,
					Priority: 100,
					Weight:   1.0,
					Enabled:  true,
					Group: &RiskGroup{
						Operator: RiskLogicAnd,
						Children: []RiskExpression{
							{
								ID:   "child1",
								Name: "Child 1",
								Type: RiskRuleTypeCondition,
								Condition: &RiskCondition{
									Field:    "speed",
									Operator: RiskOperatorGt,
									Value:    float64(1000),
								},
							},
							{
								ID:   "child2",
								Name: "Child 2",
								Type: RiskRuleTypeCondition,
								Condition: &RiskCondition{
									Field:    "acceleration",
									Operator: RiskOperatorLt,
									Value:    float64(0.5),
								},
							},
						},
					},
				},
			},
		}

		engine := NewRiskEngine(config)
		ctx := &RiskContext{
			Data: map[string]interface{}{
				"speed":       float64(2000),
				"acceleration": float64(0.2),
			},
		}

		result, err := engine.Evaluate(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.TriggeredRules), 0)
	})

	t.Run("OR Group - One Match", func(t *testing.T) {
		config := &RiskConfig{
			ID:           "or_test",
			Name:         "OR Test",
			DefaultScore: 10,
			Timeout:      100 * time.Millisecond,
			Expressions: []RiskExpression{
				{
					ID:       "or_group",
					Name:     "OR Group",
					Type:     RiskRuleTypeGroup,
					Priority: 100,
					Weight:   1.0,
					Enabled:  true,
					Group: &RiskGroup{
						Operator: RiskLogicOr,
						Children: []RiskExpression{
							{
								ID:   "child1",
								Name: "Child 1",
								Type: RiskRuleTypeCondition,
								Condition: &RiskCondition{
									Field:    "speed",
									Operator: RiskOperatorGt,
									Value:    float64(5000),
								},
							},
							{
								ID:   "child2",
								Name: "Child 2",
								Type: RiskRuleTypeCondition,
								Condition: &RiskCondition{
									Field:    "acceleration",
									Operator: RiskOperatorLt,
									Value:    float64(0.5),
								},
							},
						},
					},
				},
			},
		}

		engine := NewRiskEngine(config)
		ctx := &RiskContext{
			Data: map[string]interface{}{
				"speed":       float64(2000),
				"acceleration": float64(0.2),
			},
		}

		result, err := engine.Evaluate(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.TriggeredRules), 0)
	})

	t.Run("NOT Group", func(t *testing.T) {
		config := &RiskConfig{
			ID:           "not_test",
			Name:         "NOT Test",
			DefaultScore: 10,
			Timeout:      100 * time.Millisecond,
			Expressions: []RiskExpression{
				{
					ID:       "not_group",
					Name:     "NOT Group",
					Type:     RiskRuleTypeGroup,
					Priority: 100,
					Weight:   1.0,
					Enabled:  true,
					Group: &RiskGroup{
						Operator: RiskLogicNot,
						Children: []RiskExpression{
							{
								ID:   "child1",
								Name: "Child 1",
								Type: RiskRuleTypeCondition,
								Condition: &RiskCondition{
									Field:    "blocked",
									Operator: RiskOperatorEq,
									Value:    true,
								},
							},
						},
					},
				},
			},
		}

		engine := NewRiskEngine(config)
		ctx := &RiskContext{
			Data: map[string]interface{}{
				"blocked": false,
			},
		}

		result, err := engine.Evaluate(ctx)
		require.NoError(t, err)
		assert.NotNil(t, result)
		assert.Greater(t, len(result.TriggeredRules), 0)
	})
}

func TestRiskEngine_Priority(t *testing.T) {
	config := &RiskConfig{
		ID:           "priority_test",
		Name:         "Priority Test",
		DefaultScore: 10,
		Timeout:      100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "low_priority",
				Name:     "Low Priority Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 10,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "score",
					Operator: RiskOperatorGt,
					Value:    float64(10),
				},
			},
			{
				ID:       "high_priority",
				Name:     "High Priority Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "score",
					Operator: RiskOperatorGt,
					Value:    float64(10),
				},
			},
			{
				ID:       "medium_priority",
				Name:     "Medium Priority Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 50,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "score",
					Operator: RiskOperatorGt,
					Value:    float64(10),
				},
			},
		},
	}

	engine := NewRiskEngine(config)
	ctx := &RiskContext{
		Data: map[string]interface{}{
			"score": float64(50),
		},
	}

	result, err := engine.Evaluate(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 3, len(result.TriggeredRules))
}

func TestRiskEngine_Timeout(t *testing.T) {
	config := &RiskConfig{
		ID:           "timeout_test",
		Name:         "Timeout Test",
		DefaultScore: 10,
		Timeout:      1 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "slow_rule",
				Name:     "Slow Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "value",
					Operator: RiskOperatorEq,
					Value:    "test",
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"value": "test",
		},
	}

	result, err := engine.Evaluate(ctx)
	if err != nil {
		assert.Error(t, err)
		assert.Contains(t, err.Error(), "timeout")
		assert.NotNil(t, result)
	}
}

func TestRiskEngine_WeightedScoring(t *testing.T) {
	config := &RiskConfig{
		ID:           "weighted_test",
		Name:         "Weighted Scoring Test",
		DefaultScore: 10,
		Threshold:    50,
		Timeout:      100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "high_weight",
				Name:     "High Weight Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   0.8,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "risk",
					Operator: RiskOperatorEq,
					Value:    "high",
				},
			},
			{
				ID:       "low_weight",
				Name:     "Low Weight Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 90,
				Weight:   0.2,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "risk",
					Operator: RiskOperatorEq,
					Value:    "medium",
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"risk": "high",
		},
	}

	result, err := engine.Evaluate(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, result.TotalScore, float64(0))
}

func TestRiskEngine_NestedFields(t *testing.T) {
	config := &RiskConfig{
		ID:           "nested_test",
		Name:         "Nested Fields Test",
		DefaultScore: 10,
		Timeout:      100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "nested_rule",
				Name:     "Nested Field Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "user.profile.score",
					Operator: RiskOperatorGt,
					Value:    float64(50),
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"user": map[string]interface{}{
				"profile": map[string]interface{}{
					"score": float64(75),
				},
			},
		},
	}

	result, err := engine.Evaluate(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Greater(t, len(result.TriggeredRules), 0)
}

func TestRiskEngine_RiskLevels(t *testing.T) {
	testCases := []struct {
		name           string
		triggeredRules []RiskResult
		expectedLevel  string
	}{
		{
			name:           "Critical Risk",
			triggeredRules: []RiskResult{{Score: 90}},
			expectedLevel:  "critical",
		},
		{
			name:           "High Risk",
			triggeredRules: []RiskResult{{Score: 70}},
			expectedLevel:  "high",
		},
		{
			name:           "Medium Risk",
			triggeredRules: []RiskResult{{Score: 50}},
			expectedLevel:  "medium",
		},
		{
			name:           "Low Risk",
			triggeredRules: []RiskResult{{Score: 30}},
			expectedLevel:  "low",
		},
		{
			name:           "Minimal Risk",
			triggeredRules: []RiskResult{{Score: 10}},
			expectedLevel:  "minimal",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			assessment := &RiskAssessmentResult{
				TotalScore:     tc.triggeredRules[0].Score,
				TriggeredRules: tc.triggeredRules,
				EvaluatedAt:    time.Now(),
			}

			engine := &RiskEngine{}
			riskLevel := engine.calculateRiskLevel(assessment.TotalScore)
			assert.Equal(t, tc.expectedLevel, riskLevel)
		})
	}
}

func TestRiskEngine_JSONSerialization(t *testing.T) {
	config := &RiskConfig{
		ID:           "json_test",
		Name:         "JSON Serialization Test",
		DefaultScore: 10,
		Threshold:   50,
		Timeout:     100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:          "json_rule",
				Name:        "JSON Rule",
				Description: "Test rule for JSON serialization",
				Type:        RiskRuleTypeCondition,
				Priority:    100,
				Weight:      0.5,
				Enabled:     true,
				Tags:        []string{"test", "json"},
				Condition: &RiskCondition{
					Field:    "data.value",
					Operator: RiskOperatorGt,
					Value:    float64(100),
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	data, err := json.Marshal(config)
	require.NoError(t, err)
	assert.NotEmpty(t, data)

	var unmarshaledConfig RiskConfig
	err = json.Unmarshal(data, &unmarshaledConfig)
	require.NoError(t, err)
	assert.Equal(t, config.ID, unmarshaledConfig.ID)
	assert.Equal(t, config.Name, unmarshaledConfig.Name)
	assert.Equal(t, len(config.Expressions), len(unmarshaledConfig.Expressions))

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"data": map[string]interface{}{
				"value": float64(150),
			},
		},
	}

	result, err := engine.Evaluate(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)

	resultData, err := json.Marshal(result)
	require.NoError(t, err)
	assert.NotEmpty(t, resultData)
}

func TestRiskEngine_DisabledRules(t *testing.T) {
	config := &RiskConfig{
		ID:           "disabled_test",
		Name:         "Disabled Rules Test",
		DefaultScore: 10,
		Timeout:      100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "enabled_rule",
				Name:     "Enabled Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "value",
					Operator: RiskOperatorGt,
					Value:    float64(10),
				},
			},
			{
				ID:       "disabled_rule",
				Name:     "Disabled Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 90,
				Weight:   1.0,
				Enabled:  false,
				Condition: &RiskCondition{
					Field:    "value",
					Operator: RiskOperatorGt,
					Value:    float64(5),
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"value": float64(100),
		},
	}

	result, err := engine.Evaluate(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 1, len(result.TriggeredRules))
	assert.Equal(t, "enabled_rule", result.TriggeredRules[0].RuleID)
}

func TestRiskEngine_ConcurrentEvaluation(t *testing.T) {
	config := &RiskConfig{
		ID:           "concurrent_test",
		Name:         "Concurrent Test",
		DefaultScore: 10,
		Timeout:      100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "test_rule",
				Name:     "Test Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "value",
					Operator: RiskOperatorGt,
					Value:    float64(10),
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			ctx := &RiskContext{
				Data: map[string]interface{}{
					"value": float64(100),
				},
			}

			result, err := engine.Evaluate(ctx)
			require.NoError(t, err)
			assert.NotNil(t, result)

			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}
}

func TestRiskEngine_DefaultScore(t *testing.T) {
	config := &RiskConfig{
		ID:           "default_score_test",
		Name:         "Default Score Test",
		DefaultScore: 25,
		Timeout:      100 * time.Millisecond,
		Expressions: []RiskExpression{
			{
				ID:       "test_rule",
				Name:     "Test Rule",
				Type:     RiskRuleTypeCondition,
				Priority: 100,
				Weight:   1.0,
				Enabled:  true,
				Condition: &RiskCondition{
					Field:    "value",
					Operator: RiskOperatorGt,
					Value:    float64(1000),
				},
			},
		},
	}

	engine := NewRiskEngine(config)

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"value": float64(50),
		},
	}

	result, err := engine.Evaluate(ctx)
	require.NoError(t, err)
	assert.NotNil(t, result)
	assert.Equal(t, 0, len(result.TriggeredRules))
}

func TestRiskEngine_Performance(t *testing.T) {
	config := &RiskConfig{
		ID:           "perf_test",
		Name:         "Performance Test",
		DefaultScore: 10,
		Timeout:      100 * time.Millisecond,
		Expressions:  make([]RiskExpression, 100),
	}

	for i := 0; i < 100; i++ {
		config.Expressions[i] = RiskExpression{
			ID:       fmt.Sprintf("rule_%d", i),
			Name:     fmt.Sprintf("Rule %d", i),
			Type:     RiskRuleTypeCondition,
			Priority: 100 - i,
			Weight:   1.0,
			Enabled:  true,
			Condition: &RiskCondition{
				Field:    "value",
				Operator: RiskOperatorGt,
				Value:    float64(10),
			},
		}
	}

	engine := NewRiskEngine(config)

	ctx := &RiskContext{
		Data: map[string]interface{}{
			"value": float64(100),
		},
	}

	start := time.Now()
	for i := 0; i < 100; i++ {
		_, err := engine.Evaluate(ctx)
		require.NoError(t, err)
	}
	duration := time.Since(start)

	avgDuration := duration / 100
	assert.Less(t, avgDuration.Microseconds(), int64(10000), fmt.Sprintf("Average evaluation time should be less than 10ms, got %v", avgDuration))
}
