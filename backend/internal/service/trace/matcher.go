package trace

import (
	"math"

	"github.com/hjtpx/hjtpx/internal/model"
)

type TraceMatcher struct {
	extractor *TraceExtractor
}

func NewTraceMatcher() *TraceMatcher {
	return &TraceMatcher{
		extractor: NewTraceExtractor(),
	}
}

func (m *TraceMatcher) ExtractAndScore(traceData *model.TraceData) (*model.TraceFeatures, *model.TraceScore, error) {
	features, err := m.extractor.ExtractFeatures(traceData)
	if err != nil {
		return nil, nil, err
	}

	score := m.CalculateScore(features)

	return features, score, nil
}

func (m *TraceMatcher) CalculateScore(features *model.TraceFeatures) *model.TraceScore {
	score := &model.TraceScore{
		Features:    make(map[string]float64),
		RiskFactors: features.RiskFactors,
	}

	score.Features["avg_speed"] = features.AvgSpeed
	score.Features["speed_variance"] = features.SpeedVariance
	score.Features["max_acceleration"] = features.MaxAcceleration
	score.Features["smoothness"] = features.Smoothness
	score.Features["pause_count"] = float64(features.PauseCount)
	score.Features["path_ratio"] = features.PathRatio
	score.Features["max_speed"] = features.MaxSpeed
	score.Features["min_speed"] = features.MinSpeed
	score.Features["total_distance"] = features.TotalDistance
	score.Features["direct_distance"] = features.DirectDistance

	score.SpeedScore = m.scoreSpeed(features.AvgSpeed, features.SpeedVariance)
	score.AccelScore = m.scoreAcceleration(features.MaxAcceleration)
	score.SmoothScore = m.scoreSmoothness(features.Smoothness)
	score.PauseScore = m.scorePause(features.PauseCount, features.TotalTime)

	score.TotalScore = score.SpeedScore*0.3 +
		score.AccelScore*0.3 +
		score.SmoothScore*0.2 +
		score.PauseScore*0.2

	score.RiskFactors = features.RiskFactors

	return score
}

func (m *TraceMatcher) scoreSpeed(avgSpeed, speedVariance float64) float64 {
	var speedScore float64

	if avgSpeed < 50 {
		speedScore = avgSpeed / 50 * 30
		if speedScore < 0 {
			speedScore = 0
		}
	} else if avgSpeed <= 300 {
		speedScore = 30 + (1-math.Abs(avgSpeed-175)/125)*50
	} else {
		speedScore = 80 - (avgSpeed-300)/100*30
		if speedScore < 50 {
			speedScore = 50
		}
	}

	var varianceScore float64
	if speedVariance < 50 {
		varianceScore = 30 + speedVariance/50*20
	} else if speedVariance <= 200 {
		varianceScore = 50
	} else {
		varianceScore = 70 - (speedVariance-200)/100*20
		if varianceScore < 50 {
			varianceScore = 50
		}
	}

	return (speedScore + varianceScore) / 2
}

func (m *TraceMatcher) scoreAcceleration(maxAccel float64) float64 {
	if maxAccel < 100 {
		return 50 + maxAccel/100*20
	} else if maxAccel <= 1000 {
		return 70
	} else {
		score := 70 - (maxAccel-1000)/1000*30
		if score < 40 {
			return 40
		}
		return score
	}
}

func (m *TraceMatcher) scoreSmoothness(smoothness float64) float64 {
	if smoothness < 0.2 {
		return 50 + smoothness/0.2*20
	} else if smoothness <= 0.8 {
		return 70 + (0.8-smoothness)/0.6*30
	} else {
		score := 100 - (smoothness-0.8)/0.2*20
		if score < 50 {
			return 50
		}
		return score
	}
}

func (m *TraceMatcher) scorePause(pauseCount int, totalTime int64) float64 {
	expectedPause := 0
	if totalTime > 2000 && totalTime <= 5000 {
		expectedPause = 1
	} else if totalTime > 5000 {
		expectedPause = 2
	}

	diff := math.Abs(float64(pauseCount - expectedPause))

	if diff == 0 {
		return 100
	} else if diff == 1 {
		return 75
	} else if diff == 2 {
		return 50
	} else {
		return 30
	}
}

func (m *TraceMatcher) GetRiskLevel(score *model.TraceScore) string {
	if score.TotalScore >= 80 {
		return "low"
	} else if score.TotalScore >= 60 {
		return "medium"
	} else if score.TotalScore >= 40 {
		return "high"
	} else {
		return "critical"
	}
}

func (m *TraceMatcher) IsBot(score *model.TraceScore) bool {
	riskCount := len(score.RiskFactors)

	if riskCount >= 3 {
		return true
	}

	if score.TotalScore < 30 {
		return true
	}

	if score.SpeedScore < 30 && score.SmoothScore < 30 {
		return true
	}

	return false
}
