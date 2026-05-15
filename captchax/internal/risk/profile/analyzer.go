package profile

import (
	"context"
	"fmt"
	"math"
	"time"
)

type Analyzer struct {
	repo *ProfileRepo
}

func NewAnalyzer(repo *ProfileRepo) *Analyzer {
	return &Analyzer{
		repo: repo,
	}
}

func (a *Analyzer) AnalyzeProfile(ctx context.Context, identifier string, identifierType IdentifierType) (*ProfileAnalysis, error) {
	profile, err := a.repo.GetByIdentifier(ctx, identifier, identifierType)
	if err != nil {
		return nil, fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return nil, nil
	}

	analysis := &ProfileAnalysis{
		ProfileID:       profile.ID,
		Identifier:     profile.Identifier,
		IdentifierType: profile.IdentifierType,
		Labels:         profile.CalculateLabels(),
		Anomalies:      a.detectAnomalies(profile),
		Recommendations: a.generateRecommendations(profile),
		RiskFactors:    a.analyzeRiskFactors(profile),
		ComparedToAverage: a.compareToAverage(ctx, profile),
		GeneratedAt:   time.Now(),
	}

	return analysis, nil
}

type ProfileAnalysis struct {
	ProfileID           int64                `json:"profile_id"`
	Identifier         string               `json:"identifier"`
	IdentifierType     IdentifierType        `json:"identifier_type"`
	Labels             *ProfileLabelSet      `json:"labels"`
	Anomalies          []Anomaly            `json:"anomalies"`
	Recommendations    []Recommendation     `json:"recommendations"`
	RiskFactors        []RiskFactor         `json:"risk_factors"`
	ComparedToAverage  *ComparisonResult    `json:"compared_to_average"`
	GeneratedAt        time.Time            `json:"generated_at"`
}

type Anomaly struct {
	Type        AnomalyType `json:"type"`
	Severity    int         `json:"severity"`
	Description string      `json:"description"`
	Value       interface{} `json:"value"`
	Threshold   interface{} `json:"threshold"`
}

type AnomalyType string

const (
	AnomalyTypeSuccessRate   AnomalyType = "success_rate"
	AnomalyTypeFrequency     AnomalyType = "frequency"
	AnomalyTypeResponseTime  AnomalyType = "response_time"
	AnomalyTypeLocation      AnomalyType = "location"
	AnomalyTypeDevice        AnomalyType = "device"
	AnomalyTypeBehavior      AnomalyType = "behavior"
)

type Recommendation struct {
	Priority    int           `json:"priority"`
	Action      string        `json:"action"`
	Reason      string        `json:"reason"`
	TargetLabel *TrustLevel   `json:"target_label,omitempty"`
}

type RiskFactor struct {
	Name        string  `json:"name"`
	Weight      int     `json:"weight"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
	Indicators  []string `json:"indicators"`
}

type ComparisonResult struct {
	SuccessRateDiff   float64 `json:"success_rate_diff"`
	ResponseTimeDiff  float64 `json:"response_time_diff"`
	ActivityLevelDiff float64 `json:"activity_level_diff"`
	Percentile        int     `json:"percentile"`
}

func (a *Analyzer) detectAnomalies(profile *UserProfile) []Anomaly {
	anomalies := make([]Anomaly, 0)

	if profile.TotalAttempts >= 10 {
		if profile.SuccessRate < 30 {
			anomalies = append(anomalies, Anomaly{
				Type:        AnomalyTypeSuccessRate,
				Severity:    3,
				Description: "异常低成功率",
				Value:       profile.SuccessRate,
				Threshold:   50.0,
			})
		} else if profile.SuccessRate > 99 {
			anomalies = append(anomalies, Anomaly{
				Type:        AnomalyTypeSuccessRate,
				Severity:    2,
				Description: "成功率异常高，可能存在异常行为",
				Value:       profile.SuccessRate,
				Threshold:   95.0,
			})
		}
	}

	if profile.TotalAttempts >= 10 && profile.AvgResponseTime > 0 {
		if profile.AvgResponseTime < 500 {
			anomalies = append(anomalies, Anomaly{
				Type:        AnomalyTypeResponseTime,
				Severity:    2,
				Description: "平均响应时间异常快",
				Value:       profile.AvgResponseTime,
				Threshold:   1000.0,
			})
		}
	}

	daysActive := len(profile.ActiveDays)
	if daysActive > 0 {
		attemptsPerDay := float64(profile.TotalAttempts) / float64(daysActive)
		if attemptsPerDay > 100 {
			anomalies = append(anomalies, Anomaly{
				Type:        AnomalyTypeFrequency,
				Severity:    2,
				Description: "日均验证次数异常高",
				Value:       attemptsPerDay,
				Threshold:   100.0,
			})
		}
	}

	if len(profile.LocationDistribution) > 10 {
		anomalies = append(anomalies, Anomaly{
			Type:        AnomalyTypeLocation,
			Severity:    3,
			Description: "位置分布异常分散",
			Value:       len(profile.LocationDistribution),
			Threshold:   10,
		})
	}

	if len(profile.DeviceDistribution) > 20 {
		anomalies = append(anomalies, Anomaly{
			Type:        AnomalyTypeDevice,
			Severity:    2,
			Description: "使用设备数量异常多",
			Value:       len(profile.DeviceDistribution),
			Threshold:   20,
		})
	}

	if profile.HighRiskEvents > 5 && float64(profile.HighRiskEvents)/float64(profile.TotalRiskEvents+1) > 0.3 {
		anomalies = append(anomalies, Anomaly{
			Type:        AnomalyTypeBehavior,
			Severity:    3,
			Description: "高风险事件比例异常",
			Value:       float64(profile.HighRiskEvents) / float64(profile.TotalRiskEvents+1),
			Threshold:   0.3,
		})
	}

	return anomalies
}

func (a *Analyzer) generateRecommendations(profile *UserProfile) []Recommendation {
	recommendations := make([]Recommendation, 0)

	labels := profile.CalculateLabels()

	switch labels.TrustLevel {
	case TrustLevelHighRisk:
		recommendations = append(recommendations, Recommendation{
			Priority:    1,
			Action:      "加强监控",
			Reason:      "该用户被标记为高风险，建议增加验证频率",
			TargetLabel: ptrTrustLevel(TrustLevelSuspicious),
		})
	case TrustLevelSuspicious:
		recommendations = append(recommendations, Recommendation{
			Priority:    2,
			Action:      "持续观察",
			Reason:      "该用户存在可疑行为，建议持续跟踪其行为模式",
			TargetLabel: ptrTrustLevel(TrustLevelTrusted),
		})
	}

	if profile.SuccessRate < 50 && profile.TotalAttempts >= 10 {
		recommendations = append(recommendations, Recommendation{
			Priority:    2,
			Action:      "降低验证难度",
			Reason:      fmt.Sprintf("当前成功率 %.2f%%，用户可能遇到验证困难", profile.SuccessRate),
		})
	}

	if profile.AvgResponseTime > 5000 {
		recommendations = append(recommendations, Recommendation{
			Priority:    3,
			Action:      "优化用户体验",
			Reason:      "平均响应时间过长，可能影响用户体验",
		})
	}

	if labels.Frequency == FrequencyLevelVeryHigh && labels.TrustLevel != TrustLevelTrusted {
		recommendations = append(recommendations, Recommendation{
			Priority:    2,
			Action:      "检查是否为机器人行为",
			Reason:      "验证频率异常高，需要排除自动化攻击",
		})
	}

	if profile.HighRiskEvents > 3 {
		recommendations = append(recommendations, Recommendation{
			Priority:    1,
			Action:      "审查风险事件",
			Reason:      fmt.Sprintf("存在 %d 个高风险事件，需要人工审查", profile.HighRiskEvents),
		})
	}

	if labels.Complexity == ComplexityLevelVeryComplex {
		recommendations = append(recommendations, Recommendation{
			Priority:    3,
			Action:      "关注多设备使用情况",
			Reason:      "用户使用多种设备和位置，可能是正常用户或存在账号共享行为",
		})
	}

	return recommendations
}

func (a *Analyzer) analyzeRiskFactors(profile *UserProfile) []RiskFactor {
	riskFactors := make([]RiskFactor, 0)

	if profile.TotalAttempts > 0 {
		successRateFactor := RiskFactor{
			Name:        "成功率风险",
			Weight:      30,
			Description: "基于用户历史成功率评估",
			Indicators:  []string{},
		}

		if profile.SuccessRate < 50 {
			successRateFactor.Score = 80.0
			successRateFactor.Indicators = append(successRateFactor.Indicators, fmt.Sprintf("成功率仅 %.2f%%", profile.SuccessRate))
		} else if profile.SuccessRate < 70 {
			successRateFactor.Score = 50.0
			successRateFactor.Indicators = append(successRateFactor.Indicators, fmt.Sprintf("成功率 %.2f%% 低于平均水平", profile.SuccessRate))
		} else {
			successRateFactor.Score = 20.0
			successRateFactor.Indicators = append(successRateFactor.Indicators, fmt.Sprintf("成功率 %.2f%% 表现良好", profile.SuccessRate))
		}
		riskFactors = append(riskFactors, successRateFactor)
	}

	if profile.TotalRiskEvents > 0 {
		riskEventFactor := RiskFactor{
			Name:        "风险事件风险",
			Weight:      40,
			Description: "基于历史风险事件评估",
			Indicators:  []string{},
		}

		highRiskRatio := float64(profile.HighRiskEvents) / float64(profile.TotalRiskEvents)
		if highRiskRatio > 0.5 {
			riskEventFactor.Score = 90.0
			riskEventFactor.Indicators = append(riskEventFactor.Indicators, fmt.Sprintf("高风险事件比例 %.2f%%", highRiskRatio*100))
		} else if highRiskRatio > 0.3 {
			riskEventFactor.Score = 60.0
			riskEventFactor.Indicators = append(riskEventFactor.Indicators, fmt.Sprintf("存在一定比例的高风险事件"))
		} else {
			riskEventFactor.Score = 20.0
			riskEventFactor.Indicators = append(riskEventFactor.Indicators, "风险事件控制良好")
		}
		riskFactors = append(riskFactors, riskEventFactor)
	}

	if profile.TotalAttempts >= 10 {
		behaviorFactor := RiskFactor{
			Name:        "行为模式风险",
			Weight:      30,
			Description: "基于用户行为模式评估",
			Indicators:  []string{},
		}

		behaviorScore := 0.0
		if len(profile.ActiveHours) < 3 {
			behaviorScore += 20
			behaviorFactor.Indicators = append(behaviorFactor.Indicators, "活跃时段过于集中")
		}
		if len(profile.LocationDistribution) > 5 {
			behaviorScore += 30
			behaviorFactor.Indicators = append(behaviorFactor.Indicators, "位置分布过于分散")
		}
		if len(profile.DeviceDistribution) > 10 {
			behaviorScore += 30
			behaviorFactor.Indicators = append(behaviorFactor.Indicators, "使用设备过多")
		}

		if behaviorScore > 50 {
			behaviorFactor.Score = 80.0
		} else if behaviorScore > 30 {
			behaviorFactor.Score = 50.0
		} else {
			behaviorFactor.Score = 20.0
		}
		riskFactors = append(riskFactors, behaviorFactor)
	}

	return riskFactors
}

func (a *Analyzer) compareToAverage(ctx context.Context, profile *UserProfile) *ComparisonResult {
	stats, err := a.repo.GetStats(ctx)
	if err != nil {
		return nil
	}

	comparison := &ComparisonResult{}

	if stats.AvgSuccessRate > 0 {
		comparison.SuccessRateDiff = profile.SuccessRate - stats.AvgSuccessRate
	}

	if stats.AvgResponseTime > 0 {
		comparison.ResponseTimeDiff = profile.AvgResponseTime - stats.AvgResponseTime
	}

	profileDaysActive := len(profile.ActiveDays)
	avgDaysActive := float64(stats.TotalVerifications) / float64(stats.TotalProfiles+1)
	comparison.ActivityLevelDiff = float64(profileDaysActive) - avgDaysActive

	comparison.Percentile = a.calculatePercentile(profile, stats)

	return comparison
}

func (a *Analyzer) calculatePercentile(profile *UserProfile, stats *ProfileStats) int {
	if stats.TotalProfiles == 0 {
		return 50
	}

	profileScore := a.calculateProfileScore(profile)
	avgScore := (stats.AvgSuccessRate * 0.5) + (100 - math.Min(stats.AvgResponseTime/50, 100)) * 0.5

	percentile := int(((profileScore - avgScore) / 100) * 100)
	percentile = int(math.Max(0, math.Min(100, float64(50+percentile))))

	return percentile
}

func (a *Analyzer) calculateProfileScore(profile *UserProfile) float64 {
	score := profile.SuccessRate * 0.5

	responseTimeScore := 100 - math.Min(profile.AvgResponseTime/50, 100)
	score += responseTimeScore * 0.5

	return score
}

func (a *Analyzer) GetOverallRiskScore(ctx context.Context, identifier string, identifierType IdentifierType) (int, error) {
	profile, err := a.repo.GetByIdentifier(ctx, identifier, identifierType)
	if err != nil {
		return 0, fmt.Errorf("failed to get profile: %w", err)
	}

	if profile == nil {
		return 50, nil
	}

	riskFactors := a.analyzeRiskFactors(profile)

	totalWeight := 0
	weightedScore := 0.0

	for _, factor := range riskFactors {
		totalWeight += factor.Weight
		weightedScore += factor.Score * float64(factor.Weight)
	}

	if totalWeight == 0 {
		return 50, nil
	}

	riskScore := int(weightedScore / float64(totalWeight))

	riskScore = int(math.Max(0, math.Min(100, float64(riskScore))))

	return riskScore, nil
}

func ptrTrustLevel(level TrustLevel) *TrustLevel {
	return &level
}
