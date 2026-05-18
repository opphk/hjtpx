package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type RiskScoringService struct {
	config      *model.RiskScoringConfig
	weights     *model.RiskScoringWeights
	thresholds  *model.RiskThresholds
	history     []*model.RiskScoringHistory
	mu          sync.RWMutex
	historyMu   sync.Mutex
}

func NewRiskScoringService() *RiskScoringService {
	config := model.DefaultRiskScoringConfig()
	return &RiskScoringService{
		config:     config,
		weights:    &config.Weights,
		thresholds: &config.Thresholds,
		history:    make([]*model.RiskScoringHistory, 0),
	}
}

func (s *RiskScoringService) GetConfig() *model.RiskScoringConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	configCopy := *s.config
	return &configCopy
}

func (s *RiskScoringService) UpdateConfig(config *model.RiskScoringConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config.Weights.TraceWeight < 0 || config.Weights.EnvWeight < 0 ||
		config.Weights.BehaviorWeight < 0 || config.Weights.DeviceWeight < 0 ||
		config.Weights.HistoryWeight < 0 {
		return fmt.Errorf("weights cannot be negative")
	}

	config.Weights.Normalize()

	s.config = config
	s.weights = &config.Weights
	s.thresholds = &config.Thresholds

	return nil
}

func (s *RiskScoringService) UpdateWeights(weights *model.RiskScoringWeights) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if weights.TraceWeight < 0 || weights.EnvWeight < 0 ||
		weights.BehaviorWeight < 0 || weights.DeviceWeight < 0 ||
		weights.HistoryWeight < 0 {
		return fmt.Errorf("weights cannot be negative")
	}

	weights.Normalize()
	s.weights = weights
	s.config.Weights = *weights

	return nil
}

func (s *RiskScoringService) GetWeights() *model.RiskScoringWeights {
	s.mu.RLock()
	defer s.mu.RUnlock()
	weightsCopy := *s.weights
	return &weightsCopy
}

func (s *RiskScoringService) UpdateThresholds(thresholds *model.RiskThresholds) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if thresholds.LowMax >= thresholds.MediumMax ||
		thresholds.MediumMax >= thresholds.HighMax ||
		thresholds.HighMax >= thresholds.CriticalMax {
		return fmt.Errorf("invalid threshold order")
	}

	s.thresholds = thresholds
	s.config.Thresholds = *thresholds

	return nil
}

func (s *RiskScoringService) GetThresholds() *model.RiskThresholds {
	s.mu.RLock()
	defer s.mu.RUnlock()
	thresholdsCopy := *s.thresholds
	return &thresholdsCopy
}

func (s *RiskScoringService) CalculateScore(ctx *model.RiskContext) *model.MultiDimensionalScore {
	s.mu.RLock()
	defer s.mu.RUnlock()

	traceScore := s.calculateTraceScore(ctx)
	envScore := s.calculateEnvScore(ctx)
	behaviorScore := s.calculateBehaviorScore(ctx)
	deviceScore := s.calculateDeviceScore(ctx)
	historyScore := s.calculateHistoryScore(ctx)

	totalScore := traceScore*s.weights.TraceWeight +
		envScore*s.weights.EnvWeight +
		behaviorScore*s.weights.BehaviorWeight +
		deviceScore*s.weights.DeviceWeight +
		historyScore*s.weights.HistoryWeight

	totalScore = math.Max(0, math.Min(100, totalScore))

	riskLevel := s.determineRiskLevel(totalScore)
	confidence := s.calculateConfidence(traceScore, envScore, behaviorScore, deviceScore, historyScore)

	return &model.MultiDimensionalScore{
		TraceScore:    traceScore,
		EnvScore:      envScore,
		BehaviorScore: behaviorScore,
		DeviceScore:   deviceScore,
		HistoryScore:  historyScore,
		TotalScore:    totalScore,
		RiskLevel:     riskLevel,
		Confidence:    confidence,
		Timestamp:     time.Now().Unix(),
	}
}

func (s *RiskScoringService) calculateTraceScore(ctx *model.RiskContext) float64 {
	if len(ctx.TraceData) < 3 {
		return 50.0
	}

	score := 0.0
	points := ctx.TraceData

	totalDistance := 0.0
	speeds := []float64{}
	var prevPoint model.TracePoint

	for i, point := range points {
		if i == 0 {
			prevPoint = point
			continue
		}

		dx := point.X - prevPoint.X
		dy := point.Y - prevPoint.Y
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance

		dt := float64(point.Timestamp - prevPoint.Timestamp)
		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
		}

		prevPoint = point
	}

	if len(speeds) > 1 {
		avgSpeed := 0.0
		for _, speed := range speeds {
			avgSpeed += speed
		}
		avgSpeed /= float64(len(speeds))

		if avgSpeed > 5.0 {
			score += 30
		} else if avgSpeed > 2.0 {
			score += 15
		}

		maxSpeed := speeds[0]
		for _, speed := range speeds {
			if speed > maxSpeed {
				maxSpeed = speed
			}
		}
		if maxSpeed > 10.0 {
			score += 25
		} else if maxSpeed > 5.0 {
			score += 10
		}

		if len(speeds) > 2 {
			var variance float64
			for _, speed := range speeds {
				variance += math.Pow(speed-avgSpeed, 2)
			}
			variance /= float64(len(speeds))
			speedStdDev := math.Sqrt(variance)

			if avgSpeed > 0 && speedStdDev/avgSpeed < 0.1 {
				score += 20
			}
		}
	}

	firstPoint := points[0]
	lastPoint := points[len(points)-1]
	directDistance := math.Sqrt(
		math.Pow(lastPoint.X-firstPoint.X, 2) +
			math.Pow(lastPoint.Y-firstPoint.Y, 2))

	pathEfficiency := 0.0
	if totalDistance > 0 {
		pathEfficiency = directDistance / totalDistance
	}

	if pathEfficiency > 0.95 {
		score += 25
	} else if pathEfficiency > 0.85 {
		score += 10
	}

	pauseCount := 0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)

		if distance < 2 && dt > 50 {
			pauseCount++
		}
	}

	if pauseCount == 0 && len(points) > 20 {
		score += 15
	}

	return math.Min(score, 100)
}

func (s *RiskScoringService) calculateEnvScore(ctx *model.RiskContext) float64 {
	score := 0.0

	if ctx.IsProxy {
		score += 30
	}
	if ctx.IsVPN {
		score += 25
	}
	if ctx.IsTor {
		score += 35
	}
	if ctx.IsHosting {
		score += 20
	}

	if ctx.IPReputation == "bad" {
		score += 20
	} else if ctx.IPReputation == "unknown" {
		score += 10
	}

	if ctx.Timezone == "" || ctx.Language == "" {
		score += 10
	}

	if ctx.Referer == "" {
		score += 5
	}

	if len(ctx.BrowserPlugins) < 3 && !ctx.HasTouchDevice {
		score += 10
	}

	if ctx.ScreenRes == "" {
		score += 5
	}

	return math.Min(score, 100)
}

func (s *RiskScoringService) calculateBehaviorScore(ctx *model.RiskContext) float64 {
	score := 0.0

	if ctx.FailureCount >= 3 {
		score += 30
	} else if ctx.FailureCount >= 1 {
		score += 10 * float64(ctx.FailureCount)
	}

	if ctx.TimeFromStart > 0 && ctx.TimeFromStart < 500 {
		score += 25
	} else if ctx.TimeFromStart > 0 && ctx.TimeFromStart < 1000 {
		score += 15
	}

	if ctx.MouseSpeed > 2000 {
		score += 20
	} else if ctx.MouseSpeed > 1000 {
		score += 10
	}

	if ctx.VerificationCount == 0 && ctx.FailureCount == 0 {
		score += 5
	} else if ctx.VerificationCount > 0 && ctx.FailureCount == 0 {
		score -= 10
	}

	if ctx.PositionDiff > 100 {
		score += 15
	}

	return math.Max(0, math.Min(score, 100))
}

func (s *RiskScoringService) calculateDeviceScore(ctx *model.RiskContext) float64 {
	score := 0.0

	if ctx.Fingerprint == "" {
		score += 30
	}

	knownIndicators := 0
	totalIndicators := 0

	if ctx.Fingerprint != "" {
		totalIndicators++
		var device models.DeviceFingerprint
		result := database.DB.Where("fingerprint = ?", ctx.Fingerprint).First(&device)
		if result.Error == nil {
			knownIndicators++
			score -= 15

			if device.RiskScore > 50 {
				score += device.RiskScore * 0.5
			}
		}
	}

	if ctx.IPAddress != "" {
		totalIndicators++
		var devices []models.DeviceFingerprint
		database.DB.Where("ip_address = ?", ctx.IPAddress).Find(&devices)
		if len(devices) > 3 {
			score += float64(len(devices)) * 2
		}
	}

	if ctx.DeviceInfo != nil {
		if _, ok := ctx.DeviceInfo["canvas_hash"]; ok {
			totalIndicators++
			knownIndicators++
		}
		if _, ok := ctx.DeviceInfo["webgl_info"]; ok {
			totalIndicators++
			knownIndicators++
		}
	}

	return math.Max(0, math.Min(score, 100))
}

func (s *RiskScoringService) calculateHistoryScore(ctx *model.RiskContext) float64 {
	score := 0.0

	var recentHistory []model.RiskScoringHistory
	query := database.DB.Model(&model.RiskScoringHistory{})

	if ctx.Fingerprint != "" {
		query = query.Where("fingerprint = ?", ctx.Fingerprint)
	} else if ctx.IPAddress != "" {
		query = query.Where("ip_address = ?", ctx.IPAddress)
	} else {
		return 50.0
	}

	query.Where("created_at > ?", time.Now().Add(-24*time.Hour).Unix()).
		Order("created_at DESC").
		Limit(10).
		Find(&recentHistory)

	if len(recentHistory) == 0 {
		return 50.0
	}

	var totalHistoryScore float64
	successCount := 0
	failCount := 0

	for _, h := range recentHistory {
		totalHistoryScore += h.TotalScore
		if h.Success {
			successCount++
		} else {
			failCount++
		}
	}

	avgHistoryScore := totalHistoryScore / float64(len(recentHistory))

	score = avgHistoryScore * 0.6

	if failCount > successCount && failCount > 0 {
		score += 20
	}

	highScoreCount := 0
	for _, h := range recentHistory {
		if h.TotalScore > 70 {
			highScoreCount++
		}
	}

	if highScoreCount > len(recentHistory)/2 {
		score += 15
	}

	return math.Max(0, math.Min(score, 100))
}

func (s *RiskScoringService) determineRiskLevel(totalScore float64) model.RiskLevel {
	switch {
	case totalScore < s.thresholds.LowMax:
		return model.RiskLevelLow
	case totalScore < s.thresholds.MediumMax:
		return model.RiskLevelMedium
	case totalScore < s.thresholds.HighMax:
		return model.RiskLevelHigh
	default:
		return model.RiskLevelCritical
	}
}

func (s *RiskScoringService) calculateConfidence(trace, env, behavior, device, history float64) float64 {
	confidence := 0.5

	if trace > 0 && trace < 100 {
		confidence += 0.1
	}

	if env > 0 && env < 100 {
		confidence += 0.1
	}

	variance := 0.0
	scores := []float64{trace, env, behavior, device, history}
	avg := 0.0
	for _, s := range scores {
		avg += s
	}
	avg /= 5.0

	for _, s := range scores {
		variance += math.Pow(s-avg, 2)
	}
	variance /= 5.0

	if variance < 100 {
		confidence += 0.15
	}

	return math.Min(confidence, 0.95)
}

func (s *RiskScoringService) GetAction(score *model.MultiDimensionalScore) string {
	switch {
	case score.TotalScore >= s.thresholds.BlockMin:
		return "block"
	case score.TotalScore >= s.thresholds.VerifyMin:
		return "verify"
	default:
		return "allow"
	}
}

func (s *RiskScoringService) RecordHistory(ctx *model.RiskContext, score *model.MultiDimensionalScore, action string, verified, success bool) error {
	s.historyMu.Lock()
	defer s.historyMu.Unlock()

	history := &model.RiskScoringHistory{
		SessionID:     ctx.SessionID,
		IPAddress:     ctx.IPAddress,
		Fingerprint:   ctx.Fingerprint,
		TraceScore:    score.TraceScore,
		EnvScore:      score.EnvScore,
		BehaviorScore: score.BehaviorScore,
		DeviceScore:   score.DeviceScore,
		HistoryScore:  score.HistoryScore,
		TotalScore:    score.TotalScore,
		RiskLevel:     string(score.RiskLevel),
		Action:        action,
		Verified:      verified,
		Success:       success,
		CreatedAt:     time.Now().Unix(),
	}

	s.history = append(s.history, history)
	if len(s.history) > 1000 {
		s.history = s.history[len(s.history)-1000:]
	}

	return database.DB.Create(history).Error
}

func (s *RiskScoringService) GetHistory(sessionID string) ([]*model.RiskScoringHistory, error) {
	var history []*model.RiskScoringHistory
	query := database.DB.Model(&model.RiskScoringHistory{})

	if sessionID != "" {
		query = query.Where("session_id = ?", sessionID)
	}

	err := query.Order("created_at DESC").Limit(100).Find(&history).Error
	return history, err
}

func (s *RiskScoringService) GetDistribution() (*model.RiskScoreDistribution, error) {
	var totalCount int64
	var scores []float64

	database.DB.Model(&model.RiskScoringHistory{}).Count(&totalCount)

	if totalCount == 0 {
		return &model.RiskScoreDistribution{
			TotalCount: 0,
		}, nil
	}

	rows, err := database.DB.Model(&model.RiskScoringHistory{}).
		Select("total_score").Rows()
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	for rows.Next() {
		var score float64
		if err := rows.Scan(&score); err == nil {
			scores = append(scores, score)
		}
	}

	if len(scores) == 0 {
		return &model.RiskScoreDistribution{
			TotalCount: totalCount,
		}, nil
	}

	avgScore := 0.0
	minScore := scores[0]
	maxScore := scores[0]
	var lowCount int64
	var mediumCount int64
	var highCount int64
	var criticalCount int64

	for _, score := range scores {
		avgScore += score
		if score < minScore {
			minScore = score
		}
		if score > maxScore {
			maxScore = score
		}

		if score < 30 {
			lowCount++
		} else if score < 50 {
			mediumCount++
		} else if score < 70 {
			highCount++
		} else {
			criticalCount++
		}
	}

	avgScore /= float64(len(scores))

	var variance float64
	for _, score := range scores {
		variance += math.Pow(score-avgScore, 2)
	}
	variance /= float64(len(scores))
	stdDev := math.Sqrt(variance)

	sortedScores := make([]float64, len(scores))
	copy(sortedScores, scores)
	for i := 0; i < len(sortedScores)-1; i++ {
		for j := i + 1; j < len(sortedScores); j++ {
			if sortedScores[i] > sortedScores[j] {
				sortedScores[i], sortedScores[j] = sortedScores[j], sortedScores[i]
			}
		}
	}

	var medianScore float64
	n := len(sortedScores)
	if n%2 == 0 {
		medianScore = (sortedScores[n/2-1] + sortedScores[n/2]) / 2
	} else {
		medianScore = sortedScores[n/2]
	}

	return &model.RiskScoreDistribution{
		TotalCount:      totalCount,
		LowCount:        lowCount,
		MediumCount:     mediumCount,
		HighCount:       highCount,
		CriticalCount:   criticalCount,
		LowPercent:      float64(lowCount) / float64(len(scores)) * 100,
		MediumPercent:   float64(mediumCount) / float64(len(scores)) * 100,
		HighPercent:     float64(highCount) / float64(len(scores)) * 100,
		CriticalPercent: float64(criticalCount) / float64(len(scores)) * 100,
		AvgScore:        avgScore,
		MedianScore:     medianScore,
		MinScore:        minScore,
		MaxScore:        maxScore,
		StdDev:          stdDev,
	}, nil
}

func (s *RiskScoringService) AdjustThresholds() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.config.AutoAdjust {
		return nil
	}

	var recentHistory []model.RiskScoringHistory
	err := database.DB.Model(&model.RiskScoringHistory{}).
		Where("created_at > ?", time.Now().Add(-7*24*time.Hour).Unix()).
		Where("verified = ?", true).
		Order("created_at DESC").
		Limit(1000).
		Find(&recentHistory).Error

	if err != nil || len(recentHistory) < 100 {
		return err
	}

	var falsePositiveCount int64
	var falseNegativeCount int64
	var truePositiveCount int64
	var trueNegativeCount int64

	for _, h := range recentHistory {
		isHighRisk := h.TotalScore >= s.thresholds.VerifyMin
		shouldChallenge := !h.Success || h.Verified

		if !isHighRisk && !shouldChallenge {
			trueNegativeCount++
		} else if !isHighRisk && shouldChallenge {
			falsePositiveCount++
		} else if isHighRisk && shouldChallenge {
			truePositiveCount++
		} else {
			falseNegativeCount++
		}
	}

	total := float64(falsePositiveCount + trueNegativeCount)
	falsePositiveRate := 0.0
	if total > 0 {
		falsePositiveRate = float64(falsePositiveCount) / total
	}

	adjustment := 0.0
	if falsePositiveRate > 0.005 {
		adjustment = -5.0
	} else if falsePositiveRate < 0.001 {
		adjustment = 2.0
	}

	s.thresholds.VerifyMin += adjustment
	s.thresholds.VerifyMin = math.Max(20, math.Min(60, s.thresholds.VerifyMin))

	s.thresholds.LowMax = s.thresholds.VerifyMin * 0.6
	s.thresholds.MediumMax = s.thresholds.VerifyMin
	s.thresholds.HighMax = s.thresholds.VerifyMin + 20
	s.thresholds.BlockMin = s.thresholds.HighMax + 10

	s.config.Thresholds = *s.thresholds

	return nil
}

func (s *RiskScoringService) GetVisualizationData() (map[string]interface{}, error) {
	distribution, err := s.GetDistribution()
	if err != nil {
		return nil, err
	}

	weights := s.GetWeights()
	thresholds := s.GetThresholds()

	trendData, err := s.getTrendData()
	if err != nil {
		trendData = []map[string]interface{}{}
	}

	return map[string]interface{}{
		"distribution": distribution,
		"weights":     weights,
		"thresholds":  thresholds,
		"bands":       model.DefaultScoreBands,
		"trend":       trendData,
		"summary": map[string]interface{}{
			"total_records":    distribution.TotalCount,
			"avg_score":        distribution.AvgScore,
			"median_score":     distribution.MedianScore,
			"low_risk_percent": distribution.LowPercent,
			"false_positive_rate": 0.0,
		},
	}, nil
}

func (s *RiskScoringService) getTrendData() ([]map[string]interface{}, error) {
	type TrendPoint struct {
		Hour       int64   `gorm:"column:hour"`
		AvgScore   float64 `gorm:"column:avg_score"`
		RecordCount int64  `gorm:"column:record_count"`
	}

	var trends []TrendPoint
	startTime := time.Now().Add(-24 * time.Hour).Unix()

	err := database.DB.Model(&model.RiskScoringHistory{}).
		Select("CAST((created_at / 3600) AS INTEGER) * 3600 as hour, AVG(total_score) as avg_score, COUNT(*) as record_count").
		Where("created_at > ?", startTime).
		Group("hour").
		Order("hour ASC").
		Limit(24).
		Scan(&trends).Error

	if err != nil {
		return nil, err
	}

	result := make([]map[string]interface{}, len(trends))
	for i, t := range trends {
		result[i] = map[string]interface{}{
			"timestamp":    t.Hour * 1000,
			"avg_score":    t.AvgScore,
			"record_count": t.RecordCount,
		}
	}

	return result, nil
}

func (s *RiskScoringService) ExportConfig() (string, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	data, err := json.MarshalIndent(s.config, "", "  ")
	if err != nil {
		return "", err
	}

	return string(data), nil
}

func (s *RiskScoringService) ImportConfig(configJSON string) error {
	var config model.RiskScoringConfig
	if err := json.Unmarshal([]byte(configJSON), &config); err != nil {
		return err
	}

	return s.UpdateConfig(&config)
}

func (s *RiskScoringService) ResetToDefault() {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = model.DefaultRiskScoringConfig()
	s.weights = &s.config.Weights
	s.thresholds = &s.config.Thresholds
}

func (s *RiskScoringService) EvaluateWithVerification(ctx *model.RiskContext) (*model.RiskResult, error) {
	score := s.CalculateScore(ctx)
	action := s.GetAction(score)

	result := &model.RiskResult{
		RiskScore:        score.TotalScore,
		RiskLevel:        score.RiskLevel,
		TraceScore:       score.TraceScore,
		EnvScore:         score.EnvScore,
		PositionScore:    score.TraceScore,
		Action:           action,
		RecommendVerify:  action == "verify",
		HumanProbability: 100 - score.TotalScore,
		Details: map[string]float64{
			"trace_score":     score.TraceScore,
			"env_score":       score.EnvScore,
			"behavior_score":  score.BehaviorScore,
			"device_score":    score.DeviceScore,
			"history_score":   score.HistoryScore,
			"confidence":      score.Confidence,
		},
	}

	if action == "block" {
		result.RiskFactors = append(result.RiskFactors, "high_risk_score_blocked")
	} else if action == "verify" {
		result.RiskFactors = append(result.RiskFactors, "requires_verification")
	}

	return result, nil
}

func (s *RiskScoringService) GetScoreBreakdown(ctx *model.RiskContext) map[string]interface{} {
	score := s.CalculateScore(ctx)

	return map[string]interface{}{
		"trace": map[string]interface{}{
			"score":      score.TraceScore,
			"weight":     s.weights.TraceWeight,
			"contrib":    score.TraceScore * s.weights.TraceWeight,
		},
		"environment": map[string]interface{}{
			"score":   score.EnvScore,
			"weight":  s.weights.EnvWeight,
			"contrib": score.EnvScore * s.weights.EnvWeight,
		},
		"behavior": map[string]interface{}{
			"score":   score.BehaviorScore,
			"weight":  s.weights.BehaviorWeight,
			"contrib": score.BehaviorScore * s.weights.BehaviorWeight,
		},
		"device": map[string]interface{}{
			"score":   score.DeviceScore,
			"weight":  s.weights.DeviceWeight,
			"contrib": score.DeviceScore * s.weights.DeviceWeight,
		},
		"history": map[string]interface{}{
			"score":   score.HistoryScore,
			"weight":  s.weights.HistoryWeight,
			"contrib": score.HistoryScore * s.weights.HistoryWeight,
		},
		"total": map[string]interface{}{
			"score":      score.TotalScore,
			"level":      score.RiskLevel,
			"confidence": score.Confidence,
			"action":     s.GetAction(score),
		},
	}
}

func (s *RiskScoringService) GetStats() (map[string]interface{}, error) {
	var totalCount, todayCount, weekCount int64
	var avgScore float64

	database.DB.Model(&model.RiskScoringHistory{}).Count(&totalCount)

	oneDayAgo := time.Now().Add(-24 * time.Hour).Unix()
	database.DB.Model(&model.RiskScoringHistory{}).Where("created_at > ?", oneDayAgo).Count(&todayCount)

	oneWeekAgo := time.Now().Add(-7 * 24 * time.Hour).Unix()
	database.DB.Model(&model.RiskScoringHistory{}).Where("created_at > ?", oneWeekAgo).Count(&weekCount)

	rows, _ := database.DB.Model(&model.RiskScoringHistory{}).
		Select("COALESCE(AVG(total_score), 0) as avg_score").Rows()
	if rows.Next() {
		rows.Scan(&avgScore)
	}
	rows.Close()

	var verifiedCount, blockedCount int64
	database.DB.Model(&model.RiskScoringHistory{}).Where("verified = ?", true).Count(&verifiedCount)
	database.DB.Model(&model.RiskScoringHistory{}).Where("action = ?", "block").Count(&blockedCount)

	return map[string]interface{}{
		"total_count":      totalCount,
		"today_count":      todayCount,
		"week_count":       weekCount,
		"avg_score":        avgScore,
		"verified_count":   verifiedCount,
		"blocked_count":    blockedCount,
		"verify_rate":      0.0,
		"block_rate":       0.0,
	}, nil
}

type RiskScoringServiceInterface interface {
	GetConfig() *model.RiskScoringConfig
	UpdateConfig(config *model.RiskScoringConfig) error
	UpdateWeights(weights *model.RiskScoringWeights) error
	GetWeights() *model.RiskScoringWeights
	UpdateThresholds(thresholds *model.RiskThresholds) error
	GetThresholds() *model.RiskThresholds
	CalculateScore(ctx *model.RiskContext) *model.MultiDimensionalScore
	RecordHistory(ctx *model.RiskContext, score *model.MultiDimensionalScore, action string, verified, success bool) error
	GetHistory(sessionID string) ([]*model.RiskScoringHistory, error)
	GetDistribution() (*model.RiskScoreDistribution, error)
	AdjustThresholds() error
	GetVisualizationData() (map[string]interface{}, error)
	ExportConfig() (string, error)
	ImportConfig(configJSON string) error
	ResetToDefault()
	EvaluateWithVerification(ctx *model.RiskContext) (*model.RiskResult, error)
	GetScoreBreakdown(ctx *model.RiskContext) map[string]interface{}
	GetStats() (map[string]interface{}, error)
}

var _ RiskScoringServiceInterface = (*RiskScoringService)(nil)
