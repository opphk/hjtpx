package risk

import (
	"context"
	"encoding/json"
	"math"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"

	"captchax/internal/config"
)

type DynamicThreshold struct {
	client      *redis.Client
	keyPrefix   string
	mu          sync.RWMutex
	baseThresholds config.RiskScoreThresholds
	adjustedThresholds map[string]*AdaptiveThreshold
	windowSize       time.Duration
	learningRate      float64
	decayFactor       float64
}

type AdaptiveThreshold struct {
	UserID         string
	Domain         string
	HistoryScores  []int
	MeanScore      float64
	StdDevScore    float64
	AdjustedLow    int
	AdjustedMedium int
	AdjustedHigh   int
	AdjustedCritical int
	SuccessRate    float64
	TotalAttempts  int
	LastUpdateTime time.Time
	Confidence     float64
}

type ThresholdConfig struct {
	RedisAddr     string
	RedisPassword string
	RedisDB       int
	KeyPrefix     string
	WindowSize    time.Duration
	LearningRate  float64
	DecayFactor   float64
}

type ThresholdAdjustment struct {
	FactorName   string
	Adjustment   float64
	Reason       string
	Timestamp    time.Time
}

func NewDynamicThreshold(cfg *ThresholdConfig, baseThresholds config.RiskScoreThresholds) (*DynamicThreshold, error) {
	dt := &DynamicThreshold{
		keyPrefix:         cfg.KeyPrefix,
		baseThresholds:    baseThresholds,
		adjustedThresholds: make(map[string]*AdaptiveThreshold),
		windowSize:        cfg.WindowSize,
		learningRate:       cfg.LearningRate,
		decayFactor:        cfg.DecayFactor,
	}

	if cfg.WindowSize == 0 {
		dt.windowSize = 24 * time.Hour
	}
	if cfg.LearningRate == 0 {
		dt.learningRate = 0.1
	}
	if cfg.DecayFactor == 0 {
		dt.decayFactor = 0.95
	}

	if cfg.RedisAddr != "" {
		client := redis.NewClient(&redis.Options{
			Addr:     cfg.RedisAddr,
			Password: cfg.RedisPassword,
			DB:       cfg.RedisDB,
		})

		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		defer cancel()

		if err := client.Ping(ctx).Err(); err == nil {
			dt.client = client
		}
	}

	if dt.keyPrefix == "" {
		dt.keyPrefix = "captchax:threshold:"
	}

	return dt, nil
}

func (dt *DynamicThreshold) RecordBehaviorResult(ctx context.Context, userID, domain string, score int, success bool) error {
	key := dt.buildKey(userID, domain)

	dt.mu.Lock()
	defer dt.mu.Unlock()

	threshold, exists := dt.adjustedThresholds[key]
	if !exists {
		threshold = &AdaptiveThreshold{
			UserID:           userID,
			Domain:           domain,
			HistoryScores:    make([]int, 0),
			AdjustedLow:      dt.baseThresholds.Low,
			AdjustedMedium:   dt.baseThresholds.Medium,
			AdjustedHigh:     dt.baseThresholds.High,
			AdjustedCritical: dt.baseThresholds.Critical,
			LastUpdateTime:  time.Now(),
		}
		dt.adjustedThresholds[key] = threshold
	}

	threshold.HistoryScores = append(threshold.HistoryScores, score)
	threshold.TotalAttempts++

	if len(threshold.HistoryScores) > 100 {
		threshold.HistoryScores = threshold.HistoryScores[len(threshold.HistoryScores)-100:]
	}

	if success {
		threshold.SuccessRate = (threshold.SuccessRate*float64(threshold.TotalAttempts-1) + 1) / float64(threshold.TotalAttempts)
	} else {
		threshold.SuccessRate = threshold.SuccessRate * float64(threshold.TotalAttempts-1) / float64(threshold.TotalAttempts)
	}

	dt.updateThresholdStats(threshold)

	if dt.client != nil {
		data, _ := json.Marshal(threshold)
		dt.client.Set(ctx, dt.keyPrefix+key, data, dt.windowSize*7)
	}

	return nil
}

func (dt *DynamicThreshold) updateThresholdStats(threshold *AdaptiveThreshold) {
	if len(threshold.HistoryScores) < 5 {
		return
	}

	var sum float64
	for _, score := range threshold.HistoryScores {
		sum += float64(score)
	}
	threshold.MeanScore = sum / float64(len(threshold.HistoryScores))

	var variance float64
	for _, score := range threshold.HistoryScores {
		diff := float64(score) - threshold.MeanScore
		variance += diff * diff
	}
	variance /= float64(len(threshold.HistoryScores))
	threshold.StdDevScore = math.Sqrt(variance)

	threshold.Confidence = math.Min(1.0, float64(len(threshold.HistoryScores))/50.0)

	baseLow := float64(dt.baseThresholds.Low)
	baseMedium := float64(dt.baseThresholds.Medium)
	baseHigh := float64(dt.baseThresholds.High)
	baseCritical := float64(dt.baseThresholds.Critical)

	adjustment := (threshold.MeanScore - baseMedium) * dt.learningRate * threshold.Confidence

	threshold.AdjustedLow = int(baseLow + adjustment*0.5)
	threshold.AdjustedMedium = int(baseMedium + adjustment)
	threshold.AdjustedHigh = int(baseHigh + adjustment*1.5)
	threshold.AdjustedCritical = int(baseCritical + adjustment*2.0)

	if threshold.AdjustedLow < 0 {
		threshold.AdjustedLow = 0
	}
	if threshold.AdjustedMedium < threshold.AdjustedLow {
		threshold.AdjustedMedium = threshold.AdjustedLow + 1
	}
	if threshold.AdjustedHigh < threshold.AdjustedMedium {
		threshold.AdjustedHigh = threshold.AdjustedMedium + 1
	}
	if threshold.AdjustedCritical < threshold.AdjustedHigh {
		threshold.AdjustedCritical = threshold.AdjustedHigh + 1
	}

	threshold.LastUpdateTime = time.Now()
}

func (dt *DynamicThreshold) GetAdjustedThreshold(ctx context.Context, userID, domain string) *config.RiskScoreThresholds {
	key := dt.buildKey(userID, domain)

	dt.mu.RLock()
	if threshold, exists := dt.adjustedThresholds[key]; exists {
		thresholds := &config.RiskScoreThresholds{
			Low:      threshold.AdjustedLow,
			Medium:   threshold.AdjustedMedium,
			High:     threshold.AdjustedHigh,
			Critical: threshold.AdjustedCritical,
		}
		dt.mu.RUnlock()
		return thresholds
	}
	dt.mu.RUnlock()

	if dt.client != nil {
		data, err := dt.client.Get(ctx, dt.keyPrefix+key).Bytes()
		if err == nil {
			var threshold AdaptiveThreshold
			if json.Unmarshal(data, &threshold) == nil {
				dt.mu.Lock()
				dt.adjustedThresholds[key] = &threshold
				dt.mu.Unlock()

				return &config.RiskScoreThresholds{
					Low:      threshold.AdjustedLow,
					Medium:   threshold.AdjustedMedium,
					High:     threshold.AdjustedHigh,
					Critical: threshold.AdjustedCritical,
				}
			}
		}
	}

	return &config.RiskScoreThresholds{
		Low:      dt.baseThresholds.Low,
		Medium:   dt.baseThresholds.Medium,
		High:     dt.baseThresholds.High,
		Critical: dt.baseThresholds.Critical,
	}
}

func (dt *DynamicThreshold) buildKey(userID, domain string) string {
	if userID != "" {
		return "user:" + userID
	}
	return "domain:" + domain
}

func (dt *DynamicThreshold) GetRiskLevelWithAdjustment(score int, userID, domain string, ctx context.Context) RiskLevel {
	thresholds := dt.GetAdjustedThreshold(ctx, userID, domain)

	switch {
	case score >= thresholds.Critical:
		return RiskLevelCritical
	case score >= thresholds.High:
		return RiskLevelHigh
	case score >= thresholds.Medium:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

func (dt *DynamicThreshold) GetThresholdAdjustment(userID, domain string) *ThresholdAdjustment {
	key := dt.buildKey(userID, domain)

	dt.mu.RLock()
	defer dt.mu.RUnlock()

	if threshold, exists := dt.adjustedThresholds[key]; exists {
		return &ThresholdAdjustment{
			FactorName: key,
			Adjustment: (float64(threshold.AdjustedMedium) - float64(dt.baseThresholds.Medium)) / float64(dt.baseThresholds.Medium+1),
			Reason:    "基于历史行为数据分析",
			Timestamp: threshold.LastUpdateTime,
		}
	}

	return nil
}

func (dt *DynamicThreshold) ApplyDomainAdjustment(ctx context.Context, domain string, adjustment float64) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	pattern := dt.keyPrefix + "*domain:" + domain + "*"
	keys, _ := dt.client.Keys(ctx, pattern).Result()

	for _, key := range keys {
		data, err := dt.client.Get(ctx, key).Bytes()
		if err != nil {
			continue
		}

		var threshold AdaptiveThreshold
		if json.Unmarshal(data, &threshold) == nil {
			threshold.AdjustedLow = int(float64(threshold.AdjustedLow) * (1 + adjustment))
			threshold.AdjustedMedium = int(float64(threshold.AdjustedMedium) * (1 + adjustment))
			threshold.AdjustedHigh = int(float64(threshold.AdjustedHigh) * (1 + adjustment))
			threshold.AdjustedCritical = int(float64(threshold.AdjustedCritical) * (1 + adjustment))

			newData, _ := json.Marshal(threshold)
			dt.client.Set(ctx, key, newData, dt.windowSize*7)

			dt.adjustedThresholds[key] = &threshold
		}
	}

	return nil
}

func (dt *DynamicThreshold) ApplyTimeBasedDecay(ctx context.Context) error {
	dt.mu.Lock()
	defer dt.mu.Unlock()

	for key, threshold := range dt.adjustedThresholds {
		if time.Since(threshold.LastUpdateTime) > dt.windowSize {
			threshold.MeanScore *= dt.decayFactor
			threshold.StdDevScore *= dt.decayFactor
			threshold.Confidence *= dt.decayFactor

			if threshold.Confidence < 0.1 {
				delete(dt.adjustedThresholds, key)
				if dt.client != nil {
					dt.client.Del(ctx, dt.keyPrefix+key)
				}
				continue
			}

			dt.updateThresholdStats(threshold)

			if dt.client != nil {
				data, _ := json.Marshal(threshold)
				dt.client.Set(ctx, dt.keyPrefix+key, data, dt.windowSize*7)
			}
		}
	}

	return nil
}

func (dt *DynamicThreshold) GetStatistics(ctx context.Context, userID, domain string) (*ThresholdStatistics, error) {
	key := dt.buildKey(userID, domain)

	dt.mu.RLock()
	threshold, exists := dt.adjustedThresholds[key]
	dt.mu.RUnlock()

	if !exists {
		return &ThresholdStatistics{
			HasHistory:     false,
			BaseThresholds: dt.baseThresholds,
		}, nil
	}

	stats := &ThresholdStatistics{
		HasHistory:      true,
		TotalAttempts:   threshold.TotalAttempts,
		SuccessRate:      threshold.SuccessRate,
		MeanScore:       threshold.MeanScore,
		StdDevScore:     threshold.StdDevScore,
		Confidence:      threshold.Confidence,
		LastUpdateTime:  threshold.LastUpdateTime,
		BaseThresholds:  dt.baseThresholds,
		CurrentThresholds: config.RiskScoreThresholds{
			Low:      threshold.AdjustedLow,
			Medium:   threshold.AdjustedMedium,
			High:     threshold.AdjustedHigh,
			Critical: threshold.AdjustedCritical,
		},
	}

	var sum float64
	for _, s := range threshold.HistoryScores {
		sum += float64(s)
	}
	stats.AverageHistoricalScore = sum / float64(len(threshold.HistoryScores))

	if len(threshold.HistoryScores) > 0 {
		stats.MaxHistoricalScore = threshold.HistoryScores[0]
		stats.MinHistoricalScore = threshold.HistoryScores[0]
		for _, s := range threshold.HistoryScores {
			if s > stats.MaxHistoricalScore {
				stats.MaxHistoricalScore = s
			}
			if s < stats.MinHistoricalScore {
				stats.MinHistoricalScore = s
			}
		}
	}

	return stats, nil
}

type ThresholdStatistics struct {
	HasHistory            bool
	TotalAttempts         int
	SuccessRate           float64
	MeanScore             float64
	StdDevScore           float64
	Confidence            float64
	LastUpdateTime        time.Time
	AverageHistoricalScore float64
	MaxHistoricalScore    int
	MinHistoricalScore    int
	BaseThresholds        config.RiskScoreThresholds
	CurrentThresholds     config.RiskScoreThresholds
}

func (dt *DynamicThreshold) ExportThresholds(ctx context.Context) (map[string]*AdaptiveThreshold, error) {
	dt.mu.RLock()
	defer dt.mu.RUnlock()

	result := make(map[string]*AdaptiveThreshold)

	for key, threshold := range dt.adjustedThresholds {
		result[key] = threshold
	}

	if dt.client != nil {
		pattern := dt.keyPrefix + "*"
		keys, _ := dt.client.Keys(ctx, pattern).Result()

		for _, key := range keys {
			data, err := dt.client.Get(ctx, key).Bytes()
			if err != nil {
				continue
			}

			var threshold AdaptiveThreshold
			if json.Unmarshal(data, &threshold) == nil {
				shortKey := key[len(dt.keyPrefix):]
				result[shortKey] = &threshold
			}
		}
	}

	return result, nil
}

func (dt *DynamicThreshold) Close() error {
	if dt.client != nil {
		return dt.client.Close()
	}
	return nil
}

func (e *RiskEngine) AdjustScoreWithDynamicThreshold(score int, behavior *BehaviorData) int {
	if behavior == nil {
		return score
	}

	return score
}

func (e *RiskEngine) ApplyTemporalDecay(score int, lastScoreTime time.Time) int {
	elapsed := time.Since(lastScoreTime)

	if elapsed < time.Hour {
		return score
	}

	decayFactor := math.Exp(-elapsed.Hours() / 24.0)

	decayedScore := int(float64(score) * decayFactor)

	if decayedScore < score/10 {
		decayedScore = score / 10
	}

	return decayedScore
}

func (e *RiskEngine) CalculateCumulativeRisk(baseScore int, historicalScores []int, weights []float64) int {
	if len(historicalScores) == 0 {
		return baseScore
	}

	if len(weights) != len(historicalScores) {
		weights = make([]float64, len(historicalScores))
		sum := 0.0
		for i := range weights {
			weights[i] = math.Pow(0.5, float64(len(weights)-1-i))
			sum += weights[i]
		}
		for i := range weights {
			weights[i] /= sum
		}
	}

	var weightedSum float64
	for i, score := range historicalScores {
		weight := weights[i]
		if i > 0 && i < len(weights) {
			weight = weights[i]
		}
		weightedSum += float64(score) * weight
	}

	cumulativeScore := int(float64(baseScore)*0.5 + weightedSum*0.5)

	if cumulativeScore > 100 {
		cumulativeScore = 100
	}

	return cumulativeScore
}

func (e *RiskEngine) DetectScoreAnomaly(score int, historicalScores []int) bool {
	if len(historicalScores) < 10 {
		return false
	}

	var mean, variance float64
	for _, s := range historicalScores {
		mean += float64(s)
	}
	mean /= float64(len(historicalScores))

	for _, s := range historicalScores {
		diff := float64(s) - mean
		variance += diff * diff
	}
	variance /= float64(len(historicalScores))
	stdDev := math.Sqrt(variance)

	if stdDev == 0 {
		return false
	}

	zScore := math.Abs((float64(score) - mean) / stdDev)

	return zScore > 2.5
}
