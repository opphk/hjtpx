package service

import (
	"encoding/json"
	"math"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type Rule struct {
	Name      string
	Condition func(*BehaviorFeatures) bool
	Weight    float64
}

type RuleEngine struct {
	rules []Rule
}

func NewRuleEngine() *RuleEngine {
	return &RuleEngine{
		rules: make([]Rule, 0),
	}
}

func (re *RuleEngine) AddRule(rule Rule) {
	re.rules = append(re.rules, rule)
}

func (re *RuleEngine) Evaluate(features *BehaviorFeatures) float64 {
	if features == nil {
		return 0
	}

	totalScore := 0.0
	totalWeight := 0.0

	for _, rule := range re.rules {
		if rule.Condition(features) {
			totalScore += rule.Weight
		}
		totalWeight += rule.Weight
	}

	if totalWeight == 0 {
		return 0
	}

	return (totalScore / totalWeight) * 100
}

func (re *RuleEngine) GetTriggeredRules(features *BehaviorFeatures) []string {
	triggered := make([]string, 0)

	for _, rule := range re.rules {
		if rule.Condition(features) {
			triggered = append(triggered, rule.Name)
		}
	}

	return triggered
}

func (re *RuleEngine) CountTriggeredRules(features *BehaviorFeatures) int {
	count := 0
	for _, rule := range re.rules {
		if rule.Condition(features) {
			count++
		}
	}
	return count
}

var BotDetectionRules = []Rule{
	{
		Name: "speed_too_fast",
		Condition: func(f *BehaviorFeatures) bool {
			return f.AvgSpeed > 1500
		},
		Weight: 30,
	},
	{
		Name: "speed_very_fast",
		Condition: func(f *BehaviorFeatures) bool {
			return f.AvgSpeed > 2000
		},
		Weight: 40,
	},
	{
		Name: "trajectory_too_smooth",
		Condition: func(f *BehaviorFeatures) bool {
			return f.TrajectorySmoothness > 0.95
		},
		Weight: 25,
	},
	{
		Name: "no_acceleration",
		Condition: func(f *BehaviorFeatures) bool {
			return f.Acceleration < 0.1 && f.AvgSpeed > 100
		},
		Weight: 20,
	},
	{
		Name: "very_low_acceleration",
		Condition: func(f *BehaviorFeatures) bool {
			return f.Acceleration < 0.05 && f.AvgSpeed > 50
		},
		Weight: 30,
	},
	{
		Name: "path_too_simple",
		Condition: func(f *BehaviorFeatures) bool {
			return f.PathComplexity < 0.3
		},
		Weight: 15,
	},
	{
		Name: "path_very_simple",
		Condition: func(f *BehaviorFeatures) bool {
			return f.PathComplexity < 0.1
		},
		Weight: 25,
	},
	{
		Name: "human_similarity_low",
		Condition: func(f *BehaviorFeatures) bool {
			return f.PathSimilarity < 0.5
		},
		Weight: 25,
	},
	{
		Name: "human_similarity_very_low",
		Condition: func(f *BehaviorFeatures) bool {
			return f.PathSimilarity < 0.3
		},
		Weight: 35,
	},
	{
		Name: "speed_variation_too_low",
		Condition: func(f *BehaviorFeatures) bool {
			return f.SpeedVariation < 0.1 && f.AvgSpeed > 50
		},
		Weight: 20,
	},
	{
		Name: "click_interval_too_short",
		Condition: func(f *BehaviorFeatures) bool {
			return f.ClickInterval > 0 && f.ClickInterval < 50
		},
		Weight: 15,
	},
	{
		Name: "click_interval_very_short",
		Condition: func(f *BehaviorFeatures) bool {
			return f.ClickInterval > 0 && f.ClickInterval < 30
		},
		Weight: 25,
	},
	{
		Name: "max_speed_exceeds_normal",
		Condition: func(f *BehaviorFeatures) bool {
			return f.MaxSpeed > 3000
		},
		Weight: 30,
	},
	{
		Name: "speed_inconsistent",
		Condition: func(f *BehaviorFeatures) bool {
			return f.SpeedVariation > 2.0 && f.AvgSpeed > 100
		},
		Weight: 15,
	},
	{
		Name: "perfect_trajectory",
		Condition: func(f *BehaviorFeatures) bool {
			return f.TrajectorySmoothness > 0.98 && f.PathComplexity < 0.2
		},
		Weight: 35,
	},
}

func NewBotDetectionRuleEngine() *RuleEngine {
	engine := NewRuleEngine()
	for _, rule := range BotDetectionRules {
		engine.AddRule(rule)
	}
	return engine
}

type MLClassifier struct {
	ruleEngine    *RuleEngine
	scoreCard     *ScoreCard
	weights       map[string]float64
	decisionBound float64
}

func NewMLClassifier() *MLClassifier {
	return &MLClassifier{
		ruleEngine: NewBotDetectionRuleEngine(),
		scoreCard:  NewScoreCard(),
		weights: map[string]float64{
			"rule_engine": 0.4,
			"score_card":  0.4,
			"risk_score":  0.2,
		},
		decisionBound: 50,
	}
}

func (ml *MLClassifier) Classify(features *BehaviorFeatures) (bool, float64) {
	if features == nil {
		return false, 0
	}

	ruleScore := ml.ruleEngine.Evaluate(features)

	scoreCardScore := ml.scoreCard.Evaluate(features)

	riskScore := features.RiskScore

	finalScore := ml.weights["rule_engine"]*ruleScore +
		ml.weights["score_card"]*scoreCardScore +
		ml.weights["risk_score"]*riskScore

	isBot := finalScore >= ml.decisionBound

	return isBot, finalScore
}

func (ml *MLClassifier) GetConfidence(features *BehaviorFeatures) float64 {
	if features == nil {
		return 0
	}

	triggeredRules := ml.ruleEngine.CountTriggeredRules(features)
	totalRules := len(ml.ruleEngine.rules)

	ruleConfidence := float64(triggeredRules) / float64(totalRules)

	return math.Min(ruleConfidence*1.5, 1.0)
}

func (ml *MLClassifier) GetDetailedAnalysis(features *BehaviorFeatures) map[string]interface{} {
	if features == nil {
		return nil
	}

	analysis := make(map[string]interface{})

	analysis["triggered_rules"] = ml.ruleEngine.GetTriggeredRules(features)
	analysis["rule_count"] = ml.ruleEngine.CountTriggeredRules(features)
	analysis["total_rules"] = len(ml.ruleEngine.rules)

	analysis["rule_score"] = ml.ruleEngine.Evaluate(features)
	analysis["score_card_score"] = ml.scoreCard.Evaluate(features)
	analysis["risk_score"] = features.RiskScore

	analysis["features"] = features

	analysis["confidence"] = ml.GetConfidence(features)

	isBot, finalScore := ml.Classify(features)
	analysis["is_bot"] = isBot
	analysis["final_score"] = finalScore

	return analysis
}

type EnsembleClassifier struct {
	classifiers []struct {
		name     string
		classify func(*BehaviorFeatures) (bool, float64)
		weight   float64
	}
}

func NewEnsembleClassifier() *EnsembleClassifier {
	ec := &EnsembleClassifier{
		classifiers: make([]struct {
			name     string
			classify func(*BehaviorFeatures) (bool, float64)
			weight   float64
		}, 0),
	}

	ec.AddClassifier("ml_classifier", func(f *BehaviorFeatures) (bool, float64) {
		ml := NewMLClassifier()
		return ml.Classify(f)
	}, 0.4)

	ec.AddClassifier("rule_engine", func(f *BehaviorFeatures) (bool, float64) {
		engine := NewBotDetectionRuleEngine()
		score := engine.Evaluate(f)
		return score >= 50, score
	}, 0.3)

	ec.AddClassifier("score_card", func(f *BehaviorFeatures) (bool, float64) {
		sc := NewScoreCard()
		score := sc.Evaluate(f)
		return score >= 50, score
	}, 0.3)

	return ec
}

func (ec *EnsembleClassifier) AddClassifier(name string, classify func(*BehaviorFeatures) (bool, float64), weight float64) {
	ec.classifiers = append(ec.classifiers, struct {
		name     string
		classify func(*BehaviorFeatures) (bool, float64)
		weight   float64
	}{
		name:     name,
		classify: classify,
		weight:   weight,
	})
}

func (ec *EnsembleClassifier) Classify(features *BehaviorFeatures) (bool, float64) {
	if features == nil {
		return false, 0
	}

	totalWeight := 0.0
	weightedScore := 0.0
	botVotes := 0
	totalVotes := 0

	for _, clf := range ec.classifiers {
		isBot, score := clf.classify(features)
		weightedScore += score * clf.weight
		totalWeight += clf.weight

		if isBot {
			botVotes++
		}
		totalVotes++
	}

	if totalWeight == 0 {
		return false, 0
	}

	finalScore := weightedScore / totalWeight

	majorityVote := float64(botVotes)/float64(totalVotes) >= 0.5

	finalBot := finalScore >= 50 || majorityVote

	return finalBot, finalScore
}

func (ec *EnsembleClassifier) GetDetailedAnalysis(features *BehaviorFeatures) map[string]interface{} {
	if features == nil {
		return nil
	}

	analysis := make(map[string]interface{})

	classifierResults := make([]map[string]interface{}, 0)
	botVotes := 0

	for _, clf := range ec.classifiers {
		isBot, score := clf.classify(features)
		result := map[string]interface{}{
			"name":   clf.name,
			"score":  score,
			"isBot":  isBot,
			"weight": clf.weight,
		}
		classifierResults = append(classifierResults, result)

		if isBot {
			botVotes++
		}
	}

	analysis["classifier_results"] = classifierResults
	analysis["bot_votes"] = botVotes
	analysis["total_votes"] = len(ec.classifiers)
	analysis["vote_ratio"] = float64(botVotes) / float64(len(ec.classifiers))

	isBot, finalScore := ec.Classify(features)
	analysis["is_bot"] = isBot
	analysis["final_score"] = finalScore

	return analysis
}

func ValidateTrajectory(trajectory []TrajectoryPoint) bool {
	if len(trajectory) < 3 {
		return false
	}

	for i := 1; i < len(trajectory); i++ {
		if trajectory[i].Timestamp <= trajectory[i-1].Timestamp {
			return false
		}
	}

	return true
}

func PreprocessTrajectory(trajectory []TrajectoryPoint, targetLength int) []TrajectoryPoint {
	if len(trajectory) == 0 {
		return trajectory
	}

	if len(trajectory) == targetLength {
		return trajectory
	}

	if len(trajectory) > targetLength {
		return downsampleTrajectory(trajectory, targetLength)
	}

	return upsampleTrajectory(trajectory, targetLength)
}

func downsampleTrajectory(trajectory []TrajectoryPoint, targetLength int) []TrajectoryPoint {
	if len(trajectory) <= targetLength {
		return trajectory
	}

	ratio := float64(len(trajectory)) / float64(targetLength)
	result := make([]TrajectoryPoint, targetLength)

	for i := 0; i < targetLength; i++ {
		srcIdx := int(float64(i) * ratio)
		if srcIdx >= len(trajectory) {
			srcIdx = len(trajectory) - 1
		}
		result[i] = trajectory[srcIdx]
	}

	return result
}

func upsampleTrajectory(trajectory []TrajectoryPoint, targetLength int) []TrajectoryPoint {
	if len(trajectory) >= targetLength {
		return trajectory
	}

	result := make([]TrajectoryPoint, targetLength)

	for i := 0; i < targetLength; i++ {
		pos := float64(i) / float64(targetLength-1) * float64(len(trajectory)-1)
		idx := int(pos)
		frac := pos - float64(idx)

		if idx >= len(trajectory)-1 {
			result[i] = trajectory[len(trajectory)-1]
			continue
		}

		p1 := trajectory[idx]
		p2 := trajectory[idx+1]

		result[i] = TrajectoryPoint{
			X:         int(float64(p1.X)*(1-frac) + float64(p2.X)*frac),
			Y:         int(float64(p1.Y)*(1-frac) + float64(p2.Y)*frac),
			Timestamp: int64(float64(p1.Timestamp)*(1-frac) + float64(p2.Timestamp)*frac),
		}
	}

	return result
}

func ExtractFeaturesFromBehaviorData(trajectory []models.BehaviorData) *BehaviorFeatures {
	points := make([]TrajectoryPoint, 0, len(trajectory))

	for _, td := range trajectory {
		var dp BehaviorDataPoint
		if err := json.Unmarshal([]byte(td.Data), &dp); err == nil {
			points = append(points, TrajectoryPoint{
				X:         dp.X,
				Y:         dp.Y,
				Timestamp: dp.Timestamp,
			})
		}
	}

	return ExtractFeatures(points)
}

func ExtractFeaturesFromDataPoints(trajectory []BehaviorDataPoint) *BehaviorFeatures {
	points := make([]TrajectoryPoint, len(trajectory))

	for i, td := range trajectory {
		points[i] = TrajectoryPoint{
			X:         td.X,
			Y:         td.Y,
			Timestamp: td.Timestamp,
		}
	}

	return ExtractFeatures(points)
}
