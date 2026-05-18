package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"
)

type EnhancedCollectionMetrics struct {
	TotalPoints       int64              `json:"total_points"`
	CollectionDuration float64           `json:"collection_duration_ms"`
	Throughput        float64            `json:"throughput_points_per_sec"`
	MemoryUsage       int64              `json:"memory_usage_bytes"`
	DataCompleteness  float64            `json:"data_completeness_ratio"`
	DimensionCoverage map[string]float64 `json:"dimension_coverage"`
}

type BehaviorDataCollector struct {
	mu                sync.RWMutex
	buffer            []BehaviorDataItem
	maxBufferSize     int
	collectionStart   time.Time
	metrics           *CollectionMetrics
	dimensionCounters map[string]int
}

type BehaviorDataItem struct {
	DataType  string `json:"data_type"`
	Data      string `json:"data"`
	Timestamp int64  `json:"timestamp"`
}

type CollectionMetrics struct {
	MouseMoveCount     int64 `json:"mouse_move_count"`
	ClickCount        int64 `json:"click_count"`
	KeyboardCount     int64 `json:"keyboard_count"`
	ScrollCount       int64 `json:"scroll_count"`
	TouchCount        int64 `json:"touch_count"`
	TotalDataPoints   int64 `json:"total_data_points"`
	InvalidDataPoints int64 `json:"invalid_data_points"`
}

func NewBehaviorDataCollector(maxBufferSize int) *BehaviorDataCollector {
	return &BehaviorDataCollector{
		buffer:            make([]BehaviorDataItem, 0, maxBufferSize),
		maxBufferSize:     maxBufferSize,
		collectionStart:   time.Now(),
		metrics:           &CollectionMetrics{},
		dimensionCounters: make(map[string]int),
	}
}

func (c *BehaviorDataCollector) Collect(data BehaviorDataItem) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if err := c.validateData(data); err != nil {
		c.metrics.InvalidDataPoints++
		return err
	}

	c.buffer = append(c.buffer, data)
	c.updateMetrics(data)
	c.updateDimensionCounters(data)

	if len(c.buffer) >= c.maxBufferSize {
		return c.flushBuffer()
	}

	return nil
}

func (c *BehaviorDataCollector) validateData(data BehaviorDataItem) error {
	if data.DataType == "" {
		return fmt.Errorf("missing data type")
	}
	if data.Data == "" {
		return fmt.Errorf("missing data content")
	}
	if data.Timestamp <= 0 {
		return fmt.Errorf("invalid timestamp")
	}
	return nil
}

func (c *BehaviorDataCollector) updateMetrics(data BehaviorDataItem) {
	c.metrics.TotalDataPoints++

	switch data.DataType {
	case "mouse", "move":
		c.metrics.MouseMoveCount++
	case "click":
		c.metrics.ClickCount++
	case "keyboard":
		c.metrics.KeyboardCount++
	case "scroll":
		c.metrics.ScrollCount++
	case "touch":
		c.metrics.TouchCount++
	}
}

func (c *BehaviorDataCollector) updateDimensionCounters(data BehaviorDataItem) {
	c.dimensionCounters[data.DataType]++
}

func (c *BehaviorDataCollector) flushBuffer() error {
	if len(c.buffer) == 0 {
		return nil
	}
	c.buffer = make([]BehaviorDataItem, 0, c.maxBufferSize)
	return nil
}

func (c *BehaviorDataCollector) GetMetrics() *EnhancedCollectionMetrics {
	c.mu.RLock()
	defer c.mu.RUnlock()

	duration := time.Since(c.collectionStart).Seconds()
	throughput := float64(c.metrics.TotalDataPoints) / math.Max(duration, 0.001)

	dimensionCoverage := make(map[string]float64)
	totalDimensions := float64(c.metrics.TotalDataPoints)
	if totalDimensions > 0 {
		for dim, count := range c.dimensionCounters {
			dimensionCoverage[dim] = float64(count) / totalDimensions
		}
	}

	dataCompleteness := 1.0
	if c.metrics.TotalDataPoints > 0 {
		dataCompleteness = 1.0 - (float64(c.metrics.InvalidDataPoints) / float64(c.metrics.TotalDataPoints))
	}

	return &EnhancedCollectionMetrics{
		TotalPoints:        c.metrics.TotalDataPoints,
		CollectionDuration: duration * 1000,
		Throughput:         throughput,
		DimensionCoverage:  dimensionCoverage,
		DataCompleteness:   dataCompleteness,
	}
}

type EnhancedBehaviorCollector struct {
	collectors map[string]*BehaviorDataCollector
	mu         sync.RWMutex
}

func NewEnhancedBehaviorCollector() *EnhancedBehaviorCollector {
	return &EnhancedBehaviorCollector{
		collectors: make(map[string]*BehaviorDataCollector),
	}
}

func (ec *EnhancedBehaviorCollector) GetOrCreateCollector(sessionID string) *BehaviorDataCollector {
	ec.mu.Lock()
	defer ec.mu.Unlock()

	if collector, exists := ec.collectors[sessionID]; exists {
		return collector
	}

	collector := NewBehaviorDataCollector(1000)
	ec.collectors[sessionID] = collector
	return collector
}

func (ec *EnhancedBehaviorCollector) Collect(sessionID string, data BehaviorDataItem) error {
	collector := ec.GetOrCreateCollector(sessionID)
	return collector.Collect(data)
}

func (ec *EnhancedBehaviorCollector) GetSessionMetrics(sessionID string) *EnhancedCollectionMetrics {
	ec.mu.RLock()
	defer ec.mu.RUnlock()

	if collector, exists := ec.collectors[sessionID]; exists {
		return collector.GetMetrics()
	}
	return nil
}

func (ec *EnhancedBehaviorCollector) RemoveSession(sessionID string) {
	ec.mu.Lock()
	defer ec.mu.Unlock()
	delete(ec.collectors, sessionID)
}

type ComprehensiveAnalysisService struct {
	collector          *EnhancedBehaviorCollector
	featureExtractor    *AdvancedFeatureExtractor
	modelTrainer       *EnhancedModelTrainer
	predictionEngine   *EnhancedPredictionEngine
	performanceTracker *EnhancedPerformanceTracker
}

type AdvancedFeatureExtractor struct {
	mu sync.RWMutex
}

func NewAdvancedFeatureExtractor() *AdvancedFeatureExtractor {
	return &AdvancedFeatureExtractor{}
}

type ComprehensiveFeatures struct {
	BasicFeatures       *BasicFeatures      `json:"basic_features"`
	AdvancedFeatures    *AdvancedFeatures   `json:"advanced_features"`
	StatisticalFeatures *StatisticalFeatures `json:"statistical_features"`
	FrequencyFeatures   *FrequencyFeatures  `json:"frequency_features"`
	PatternFeatures     *PatternFeatures    `json:"pattern_features"`
	DerivedFeatures     *DerivedFeatures    `json:"derived_features"`
}

type BasicFeatures struct {
	PointCount     int     `json:"point_count"`
	TotalDistance  float64 `json:"total_distance"`
	DirectDistance float64 `json:"direct_distance"`
	TotalDuration  float64 `json:"total_duration"`
	AverageSpeed   float64 `json:"average_speed"`
	MaxSpeed       float64 `json:"max_speed"`
	MinSpeed       float64 `json:"min_speed"`
	SpeedVariance  float64 `json:"speed_variance"`
	PauseCount     int     `json:"pause_count"`
	ClickCount     int     `json:"click_count"`
}

type AdvancedFeatures struct {
	MedianSpeed              float64 `json:"median_speed"`
	SpeedVariance           float64 `json:"speed_variance"`
	SpeedSkewness           float64 `json:"speed_skewness"`
	SpeedKurtosis           float64 `json:"speed_kurtosis"`
	SpeedEntropy            float64 `json:"speed_entropy"`
	MedianAcceleration      float64 `json:"median_acceleration"`
	AccelerationVariance    float64 `json:"acceleration_variance"`
	AccelerationSkewness   float64 `json:"acceleration_skewness"`
	JerkMean                float64 `json:"jerk_mean"`
	JerkMax                 float64 `json:"jerk_max"`
	CurvatureMedian         float64 `json:"curvature_median"`
	CurvatureVariance      float64 `json:"curvature_variance"`
	CurvatureMax           float64 `json:"curvature_max"`
	DirectionChangeRate     float64 `json:"direction_change_rate"`
	DirectionEntropy        float64 `json:"direction_entropy"`
	Sinuosity               float64 `json:"sinuosity"`
	StartEndAngle           float64 `json:"start_end_angle"`
	AreaUnderCurve          float64 `json:"area_under_curve"`
	TimeNormalizedDistance  float64 `json:"time_normalized_distance"`
	VelocityProfileEntropy  float64 `json:"velocity_profile_entropy"`
	AccelerationProfileEntropy float64 `json:"acceleration_profile_entropy"`
}

type StatisticalFeatures struct {
	Mean           float64 `json:"mean"`
	Median         float64 `json:"median"`
	Mode           float64 `json:"mode"`
	StdDev         float64 `json:"std_dev"`
	Variance       float64 `json:"variance"`
	Skewness       float64 `json:"skewness"`
	Kurtosis       float64 `json:"kurtosis"`
	Range          float64 `json:"range"`
	IQR            float64 `json:"inter_quartile_range"`
	Percentile25   float64 `json:"percentile_25"`
	Percentile75   float64 `json:"percentile_75"`
	CoeffVariation float64 `json:"coefficient_of_variation"`
}

type FrequencyFeatures struct {
	DominantFrequency float64 `json:"dominant_frequency"`
	EnergyRatio       float64 `json:"energy_ratio"`
	SpectralEntropy   float64 `json:"spectral_entropy"`
	PeakCount         int     `json:"peak_count"`
	FrequencySpread   float64 `json:"frequency_spread"`
}

type PatternFeatures struct {
	PathRatio         float64 `json:"path_ratio"`
	Sinuosity         float64 `json:"sinuosity"`
	CurvatureMean     float64 `json:"curvature_mean"`
	CurvatureVariance float64 `json:"curvature_variance"`
	DirectionEntropy  float64 `json:"direction_entropy"`
	DirectionChanges  int     `json:"direction_changes"`
	ReversalCount     int     `json:"reversal_count"`
	ClusterCount      int     `json:"cluster_count"`
}

type DerivedFeatures struct {
	BotProbability        float64 `json:"bot_probability"`
	HumanLikelihood       float64 `json:"human_likelihood"`
	AnomalyScore          float64 `json:"anomaly_score"`
	RiskLevel             string  `json:"risk_level"`
	ConfidenceLevel       float64 `json:"confidence_level"`
	FeatureCompleteness   float64 `json:"feature_completeness"`
}

func (fe *AdvancedFeatureExtractor) ExtractComprehensiveFeatures(
	points []BehaviorDataPoint,
	clicks []BehaviorDataPoint,
	keyStrokes []KeyboardDataPoint,
) *ComprehensiveFeatures {
	fe.mu.Lock()
	defer fe.mu.Unlock()

	features := &ComprehensiveFeatures{
		BasicFeatures:       fe.extractBasicFeatures(points, clicks),
		AdvancedFeatures:    fe.extractAdvancedFeatures(points),
		StatisticalFeatures: fe.extractStatisticalFeatures(points),
		FrequencyFeatures:   fe.extractFrequencyFeatures(points),
		PatternFeatures:     fe.extractPatternFeatures(points),
		DerivedFeatures:     fe.deriveRiskFeatures(points, clicks),
	}

	return features
}

func (fe *AdvancedFeatureExtractor) extractBasicFeatures(points []BehaviorDataPoint, clicks []BehaviorDataPoint) *BasicFeatures {
	bf := &BasicFeatures{}

	if len(points) == 0 {
		return bf
	}

	bf.PointCount = len(points)

	if len(points) >= 2 {
		bf.TotalDuration = float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	bf.TotalDistance = totalDistance

	if len(points) >= 2 {
		dx := float64(points[len(points)-1].X - points[0].X)
		dy := float64(points[len(points)-1].Y - points[0].Y)
		bf.DirectDistance = math.Sqrt(dx*dx + dy*dy)
	}

	speeds := fe.calculateSpeeds(points)
	if len(speeds) > 0 {
		bf.AverageSpeed = fe.mean(speeds)
		bf.MaxSpeed = fe.max(speeds)
		bf.MinSpeed = fe.min(speeds)
		bf.SpeedVariance = fe.variance(speeds)
	}

	bf.PauseCount = fe.countPauses(points)
	bf.ClickCount = len(clicks)

	return bf
}

func (fe *AdvancedFeatureExtractor) extractAdvancedFeatures(points []BehaviorDataPoint) *AdvancedFeatures {
	af := &AdvancedFeatures{}

	if len(points) < 3 {
		return af
	}

	speeds := fe.calculateSpeeds(points)
	if len(speeds) > 0 {
		af.MedianSpeed = fe.median(speeds)
		af.SpeedVariance = fe.variance(speeds)
		af.SpeedSkewness = fe.calculateSkewness(speeds)
		af.SpeedKurtosis = fe.calculateKurtosis(speeds)
		af.SpeedEntropy = fe.calculateEntropy(speeds, 10)
	}

	accelerations := fe.calculateAccelerations(points)
	if len(accelerations) > 0 {
		af.MedianAcceleration = fe.median(accelerations)
		af.AccelerationVariance = fe.variance(accelerations)
		af.AccelerationSkewness = fe.calculateSkewness(accelerations)
	}

	jerks := fe.calculateJerks(points)
	if len(jerks) > 0 {
		af.JerkMean = fe.mean(jerks)
		af.JerkMax = fe.maxAbs(jerks)
	}

	curvatures := fe.calculateCurvatures(points)
	if len(curvatures) > 0 {
		af.CurvatureMedian = fe.median(curvatures)
		af.CurvatureVariance = fe.variance(curvatures)
		af.CurvatureMax = fe.maxAbs(curvatures)
	}

	directions := fe.calculateDirections(points)
	if len(directions) > 1 {
		af.DirectionChangeRate = fe.calculateDirectionChangeRate(directions)
		af.DirectionEntropy = fe.calculateDirectionEntropy(directions)
	}

	if len(points) > 1 {
		af.Sinuosity = fe.calculateSinuosity(points)
		af.StartEndAngle = fe.calculateStartEndAngle(points)
		af.AreaUnderCurve = fe.calculateAreaUnderCurve(points)
		af.TimeNormalizedDistance = fe.calculateTimeNormalizedDistance(points)
	}

	af.VelocityProfileEntropy = fe.calculateProfileEntropy(speeds, 5)
	af.AccelerationProfileEntropy = fe.calculateProfileEntropy(accelerations, 5)

	return af
}

func (fe *AdvancedFeatureExtractor) extractStatisticalFeatures(points []BehaviorDataPoint) *StatisticalFeatures {
	sf := &StatisticalFeatures{}

	if len(points) == 0 {
		return sf
	}

	xCoords := make([]float64, len(points))
	yCoords := make([]float64, len(points))
	for i, p := range points {
		xCoords[i] = float64(p.X)
		yCoords[i] = float64(p.Y)
	}

	sf.Mean = fe.mean(xCoords)
	sf.Median = fe.median(xCoords)
	sf.StdDev = math.Sqrt(fe.variance(xCoords))
	sf.Variance = fe.variance(xCoords)
	sf.Skewness = fe.calculateSkewness(xCoords)
	sf.Kurtosis = fe.calculateKurtosis(xCoords)

	minVal := fe.min(xCoords)
	maxVal := fe.max(xCoords)
	sf.Range = maxVal - minVal

	sorted := make([]float64, len(xCoords))
	copy(sorted, xCoords)
	fe.sortFloat64(sorted)
	sf.Percentile25 = fe.percentile(sorted, 25)
	sf.Percentile75 = fe.percentile(sorted, 75)
	sf.IQR = sf.Percentile75 - sf.Percentile25

	if sf.Mean > 0 {
		sf.CoeffVariation = sf.StdDev / sf.Mean
	}

	return sf
}

func (fe *AdvancedFeatureExtractor) extractFrequencyFeatures(points []BehaviorDataPoint) *FrequencyFeatures {
	ff := &FrequencyFeatures{}

	if len(points) < 4 {
		return ff
	}

	n := len(points)
	for n&(n-1) != 0 {
		n--
	}
	if n < 2 {
		return ff
	}

	x := make([]float64, n)
	y := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(points[i].X)
		y[i] = float64(points[i].Y)
	}

	magnitudes := fe.computeFFTMagnitudes(x, y)

	ff.DominantFrequency = fe.findDominantFrequency(magnitudes, points)
	ff.EnergyRatio = fe.calculateEnergyRatio(magnitudes)
	ff.SpectralEntropy = fe.calculateSpectralEntropy(magnitudes)
	ff.PeakCount = fe.countPeaks(magnitudes)
	ff.FrequencySpread = fe.calculateFrequencySpread(magnitudes)

	return ff
}

func (fe *AdvancedFeatureExtractor) extractPatternFeatures(points []BehaviorDataPoint) *PatternFeatures {
	pf := &PatternFeatures{}

	if len(points) < 2 {
		return pf
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	if len(points) >= 2 {
		dx := float64(points[len(points)-1].X - points[0].X)
		dy := float64(points[len(points)-1].Y - points[0].Y)
		directDistance := math.Sqrt(dx*dx + dy*dy)
		if directDistance > 0 {
			pf.PathRatio = totalDistance / directDistance
		}
	}

	pf.Sinuosity = pf.PathRatio

	curvatures := fe.calculateCurvatures(points)
	if len(curvatures) > 0 {
		pf.CurvatureMean = fe.mean(curvatures)
		pf.CurvatureVariance = fe.variance(curvatures)
	}

	directions := fe.calculateDirections(points)
	if len(directions) > 0 {
		pf.DirectionEntropy = fe.calculateDirectionEntropy(directions)
	}
	pf.DirectionChanges = fe.countDirectionChanges(points)
	pf.ReversalCount = fe.countReversals(points)

	return pf
}

func (fe *AdvancedFeatureExtractor) deriveRiskFeatures(points []BehaviorDataPoint, clicks []BehaviorDataPoint) *DerivedFeatures {
	df := &DerivedFeatures{}

	if len(points) == 0 {
		df.RiskLevel = "unknown"
		return df
	}

	df.BotProbability = fe.calculateBotProbability(points)
	df.HumanLikelihood = 1.0 - df.BotProbability

	df.AnomalyScore = fe.calculateAnomalyScore(points)

	if df.BotProbability > 0.8 {
		df.RiskLevel = "critical"
	} else if df.BotProbability > 0.6 {
		df.RiskLevel = "high"
	} else if df.BotProbability > 0.4 {
		df.RiskLevel = "medium"
	} else if df.BotProbability > 0.2 {
		df.RiskLevel = "low"
	} else {
		df.RiskLevel = "minimal"
	}

	df.ConfidenceLevel = fe.calculateConfidenceLevel(points)

	baseCompleteness := math.Min(float64(len(points))/100.0, 1.0)
	hasClicks := 0.0
	if len(clicks) > 0 {
		hasClicks = 1.0
	}
	df.FeatureCompleteness = baseCompleteness*0.8 + hasClicks*0.2

	return df
}

func (fe *AdvancedFeatureExtractor) calculateBotProbability(points []BehaviorDataPoint) float64 {
	if len(points) < 10 {
		return 0.5
	}

	score := 0.0
	weight := 0.0

	speeds := fe.calculateSpeeds(points)
	if len(speeds) > 5 {
		speedVariance := fe.variance(speeds)
		if speedVariance < 5 && fe.mean(speeds) > 50 {
			score += 0.3
		}
		weight += 0.2
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	if len(points) >= 2 {
		dx := float64(points[len(points)-1].X - points[0].X)
		dy := float64(points[len(points)-1].Y - points[0].Y)
		directDistance := math.Sqrt(dx*dx + dy*dy)
		if directDistance > 0 {
			pathRatio := totalDistance / directDistance
			if pathRatio < 1.05 {
				score += 0.35
			}
		}
		weight += 0.25
	}

	pauseCount := fe.countPauses(points)
	if pauseCount == 0 && len(points) > 20 {
		score += 0.25
	}
	weight += 0.2

	curvatures := fe.calculateCurvatures(points)
	if len(curvatures) > 10 {
		curvVariance := fe.variance(curvatures)
		if curvVariance < 0.001 {
			score += 0.2
		}
		weight += 0.15
	}

	reversalCount := fe.countReversals(points)
	if reversalCount == 0 && len(points) > 15 {
		score += 0.15
	}
	weight += 0.1

	if reversalCount > len(points)/5 {
		score -= 0.2
	}

	fe.sortFloat64(speeds)
	entropy := fe.calculateEntropy(speeds, 10)
	if entropy < 1.5 {
		score += 0.15
	}
	weight += 0.1

	if weight > 0 {
		return math.Min(1.0, math.Max(0.0, score/weight))
	}

	return 0.5
}

func (fe *AdvancedFeatureExtractor) calculateAnomalyScore(points []BehaviorDataPoint) float64 {
	if len(points) < 5 {
		return 0.5
	}

	anomalyCount := 0
	totalChecks := 0

	speeds := fe.calculateSpeeds(points)
	if len(speeds) > 3 {
		totalChecks++
		if fe.variance(speeds) < 1 {
			anomalyCount++
		}
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	if len(points) >= 2 {
		dx := float64(points[len(points)-1].X - points[0].X)
		dy := float64(points[len(points)-1].Y - points[0].Y)
		directDistance := math.Sqrt(dx*dx + dy*dy)
		if directDistance > 0 && totalDistance/directDistance < 1.05 {
			anomalyCount++
		}
		totalChecks++
	}

	totalChecks++
	if fe.countPauses(points) == 0 && len(points) > 20 {
		anomalyCount++
	}

	totalChecks++
	curvatures := fe.calculateCurvatures(points)
	if len(curvatures) > 10 && fe.variance(curvatures) < 0.0001 {
		anomalyCount++
	}

	totalChecks++
	if fe.countReversals(points) > len(points)/3 {
		anomalyCount++
	}

	if totalChecks > 0 {
		return float64(anomalyCount) / float64(totalChecks)
	}

	return 0.0
}

func (fe *AdvancedFeatureExtractor) calculateConfidenceLevel(points []BehaviorDataPoint) float64 {
	baseConfidence := 0.3

	baseConfidence += math.Min(float64(len(points))/100.0, 0.4)

	if len(points) >= 20 {
		baseConfidence += 0.1
	}

	if len(points) >= 50 {
		baseConfidence += 0.1
	}

	return math.Min(1.0, baseConfidence)
}

func (fe *AdvancedFeatureExtractor) calculateSpeeds(points []BehaviorDataPoint) []float64 {
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}
	return speeds
}

func (fe *AdvancedFeatureExtractor) calculateAccelerations(points []BehaviorDataPoint) []float64 {
	speeds := fe.calculateSpeeds(points)
	if len(speeds) < 2 || len(points) < 3 {
		return nil
	}

	accelerations := make([]float64, 0, len(speeds)-1)
	for i := 1; i < len(speeds) && i+1 < len(points); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}
	return accelerations
}

func (fe *AdvancedFeatureExtractor) calculateJerks(points []BehaviorDataPoint) []float64 {
	accelerations := fe.calculateAccelerations(points)
	if len(accelerations) < 2 || len(points) < 4 {
		return nil
	}

	jerks := make([]float64, 0, len(accelerations)-1)
	for i := 1; i < len(accelerations); i++ {
		dt := float64(points[i+2].Timestamp - points[i].Timestamp)
		if dt > 0 {
			jerk := (accelerations[i] - accelerations[i-1]) / dt
			jerks = append(jerks, jerk)
		}
	}
	return jerks
}

func (fe *AdvancedFeatureExtractor) calculateCurvatures(points []BehaviorDataPoint) []float64 {
	if len(points) < 3 {
		return nil
	}

	curvatures := make([]float64, 0, len(points)-2)
	for i := 1; i < len(points)-1; i++ {
		p1 := points[i-1]
		p2 := points[i]
		p3 := points[i+1]

		v1x := float64(p2.X - p1.X)
		v1y := float64(p2.Y - p1.Y)
		v2x := float64(p3.X - p2.X)
		v2y := float64(p3.Y - p2.Y)

		dot := v1x*v2x + v1y*v2y
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			curvatures = append(curvatures, math.Abs(angle))
		}
	}
	return curvatures
}

func (fe *AdvancedFeatureExtractor) calculateDirections(points []BehaviorDataPoint) []float64 {
	if len(points) < 2 {
		return nil
	}

	directions := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)
		directions = append(directions, angle)
	}
	return directions
}

func (fe *AdvancedFeatureExtractor) countPauses(points []BehaviorDataPoint) int {
	if len(points) < 2 {
		return 0
	}

	pauseCount := 0
	const pauseThresholdMs = 200

	for i := 1; i < len(points); i++ {
		prev := points[i-1]
		curr := points[i]

		timeDiff := curr.Timestamp - prev.Timestamp
		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		distance := math.Sqrt(float64(dx*dx + dy*dy))

		if timeDiff > pauseThresholdMs && distance < 2 {
			pauseCount++
		}
	}
	return pauseCount
}

func (fe *AdvancedFeatureExtractor) countDirectionChanges(points []BehaviorDataPoint) int {
	if len(points) < 3 {
		return 0
	}

	changes := 0
	for i := 2; i < len(points); i++ {
		dx1 := float64(points[i-1].X - points[i-2].X)
		dy1 := float64(points[i-1].Y - points[i-2].Y)
		dx2 := float64(points[i].X - points[i-1].X)
		dy2 := float64(points[i].Y - points[i-1].Y)

		angle1 := math.Atan2(dy1, dx1)
		angle2 := math.Atan2(dy2, dx2)

		diff := math.Abs(angle2 - angle1)
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		if diff > 0.5 {
			changes++
		}
	}
	return changes
}

func (fe *AdvancedFeatureExtractor) countReversals(points []BehaviorDataPoint) int {
	if len(points) < 3 {
		return 0
	}

	reversals := 0
	for i := 2; i < len(points); i++ {
		dx1 := float64(points[i-1].X - points[i-2].X)
		dy1 := float64(points[i-1].Y - points[i-2].Y)
		dx2 := float64(points[i].X - points[i-1].X)
		dy2 := float64(points[i].Y - points[i-1].Y)

		if dx1*dx2 < -10 || dy1*dy2 < -10 {
			reversals++
		}
	}
	return reversals
}

func (fe *AdvancedFeatureExtractor) calculateSinuosity(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 1.0
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	dx := float64(points[len(points)-1].X - points[0].X)
	dy := float64(points[len(points)-1].Y - points[0].Y)
	directDistance := math.Sqrt(dx*dx + dy*dy)

	if directDistance == 0 {
		return 1.0
	}

	return totalDistance / directDistance
}

func (fe *AdvancedFeatureExtractor) calculateStartEndAngle(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	dx := float64(points[len(points)-1].X - points[0].X)
	dy := float64(points[len(points)-1].Y - points[0].Y)

	return math.Atan2(dy, dx)
}

func (fe *AdvancedFeatureExtractor) calculateAreaUnderCurve(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	area := 0.0
	for i := 1; i < len(points); i++ {
		y1 := float64(points[i-1].Y)
		y2 := float64(points[i].Y)
		avgY := (y1 + y2) / 2

		dx := float64(points[i].X - points[i-1].X)

		area += avgY * dx
	}

	return math.Abs(area)
}

func (fe *AdvancedFeatureExtractor) calculateTimeNormalizedDistance(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	var totalDistance float64
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	totalTime := float64(points[len(points)-1].Timestamp-points[0].Timestamp) / 1000.0

	if totalTime == 0 {
		return 0
	}

	return totalDistance / totalTime
}

func (fe *AdvancedFeatureExtractor) calculateDirectionChangeRate(directions []float64) float64 {
	if len(directions) < 2 {
		return 0
	}

	changes := 0
	for i := 1; i < len(directions); i++ {
		diff := math.Abs(directions[i] - directions[i-1])
		if diff > math.Pi {
			diff = 2*math.Pi - diff
		}
		if diff > 0.5 {
			changes++
		}
	}

	return float64(changes) / float64(len(directions))
}

func (fe *AdvancedFeatureExtractor) calculateDirectionEntropy(directions []float64) float64 {
	if len(directions) < 4 {
		return 0
	}

	buckets := 8
	bucketSize := 2 * math.Pi / float64(buckets)

	counts := make([]int, buckets)
	for _, dir := range directions {
		normalized := dir
		if normalized < 0 {
			normalized += 2 * math.Pi
		}
		bucket := int(normalized / bucketSize)
		if bucket >= buckets {
			bucket = buckets - 1
		}
		counts[bucket]++
	}

	total := len(directions)
	entropy := 0.0
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (fe *AdvancedFeatureExtractor) computeFFTMagnitudes(x, y []float64) []float64 {
	n := len(x)
	magnitudes := make([]float64, n/2)

	for i := 0; i < n/2; i++ {
		magX := 0.0
		magY := 0.0
		for j := 0; j < n; j++ {
			angle := 2 * math.Pi * float64(i*j) / float64(n)
			magX += x[j] * math.Cos(angle)
			magY += y[j] * math.Sin(angle)
		}
		magnitudes[i] = math.Sqrt(magX*magX + magY*magY)
	}

	return magnitudes
}

func (fe *AdvancedFeatureExtractor) findDominantFrequency(magnitudes []float64, points []BehaviorDataPoint) float64 {
	if len(magnitudes) == 0 {
		return 0
	}

	maxMag := 0.0
	dominantIdx := 0
	for i := 1; i < len(magnitudes); i++ {
		if magnitudes[i] > maxMag {
			maxMag = magnitudes[i]
			dominantIdx = i
		}
	}

	if len(points) >= 2 {
		totalTime := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
		if totalTime > 0 {
			return float64(dominantIdx) / totalTime * 1000
		}
	}

	return 0
}

func (fe *AdvancedFeatureExtractor) calculateEnergyRatio(magnitudes []float64) float64 {
	if len(magnitudes) == 0 {
		return 0
	}

	top10Percent := int(float64(len(magnitudes)) * 0.1)
	if top10Percent < 1 {
		top10Percent = 1
	}

	sorted := make([]float64, len(magnitudes))
	copy(sorted, magnitudes)
	fe.sortFloat64(sorted)

	topEnergy := 0.0
	totalEnergy := 0.0
	for i := 0; i < len(sorted); i++ {
		mag := sorted[len(sorted)-1-i]
		totalEnergy += mag * mag
		if i < top10Percent {
			topEnergy += mag * mag
		}
	}

	if totalEnergy > 0 {
		return topEnergy / totalEnergy
	}

	return 0
}

func (fe *AdvancedFeatureExtractor) calculateSpectralEntropy(magnitudes []float64) float64 {
	if len(magnitudes) == 0 {
		return 0
	}

	total := 0.0
	for _, mag := range magnitudes {
		total += mag * mag
	}

	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, mag := range magnitudes {
		p := (mag * mag) / total
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (fe *AdvancedFeatureExtractor) countPeaks(magnitudes []float64) int {
	if len(magnitudes) < 3 {
		return 0
	}

	mean := fe.mean(magnitudes)
	peaks := 0
	for i := 1; i < len(magnitudes)-1; i++ {
		if magnitudes[i] > magnitudes[i-1] && magnitudes[i] > magnitudes[i+1] && magnitudes[i] > mean*1.5 {
			peaks++
		}
	}

	return peaks
}

func (fe *AdvancedFeatureExtractor) calculateFrequencySpread(magnitudes []float64) float64 {
	if len(magnitudes) == 0 {
		return 0
	}

	mean := fe.mean(magnitudes)
	variance := 0.0
	for _, mag := range magnitudes {
		variance += (mag - mean) * (mag - mean)
	}
	variance /= float64(len(magnitudes))

	return math.Sqrt(variance)
}

func (fe *AdvancedFeatureExtractor) calculateEntropy(values []float64, bucketCount int) float64 {
	if len(values) < 2 || bucketCount < 2 {
		return 0
	}

	minVal := values[0]
	maxVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
		if v > maxVal {
			maxVal = v
		}
	}

	rangeVal := maxVal - minVal
	if rangeVal < 0.001 {
		return 0
	}

	bucketSize := rangeVal / float64(bucketCount)
	if bucketSize == 0 {
		return 0
	}

	counts := make([]int, bucketCount)
	for _, v := range values {
		bucket := int((v - minVal) / bucketSize)
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		if bucket < 0 {
			bucket = 0
		}
		counts[bucket]++
	}

	total := len(values)
	entropy := 0.0
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (fe *AdvancedFeatureExtractor) calculateProfileEntropy(values []float64, windowSize int) float64 {
	if len(values) < windowSize || windowSize < 2 {
		return 0
	}

	profiles := make([]int, windowSize)
	for i := 0; i < len(values); i++ {
		bucket := (i * windowSize) / len(values)
		if bucket >= windowSize {
			bucket = windowSize - 1
		}
		profiles[bucket]++
	}

	total := len(values)
	entropy := 0.0
	for _, count := range profiles {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (fe *AdvancedFeatureExtractor) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (fe *AdvancedFeatureExtractor) median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	fe.sortFloat64(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (fe *AdvancedFeatureExtractor) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := fe.mean(values)
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (fe *AdvancedFeatureExtractor) stdDev(values []float64) float64 {
	return math.Sqrt(fe.variance(values))
}

func (fe *AdvancedFeatureExtractor) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (fe *AdvancedFeatureExtractor) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (fe *AdvancedFeatureExtractor) maxAbs(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := math.Abs(values[0])
	for _, v := range values {
		if math.Abs(v) > max {
			max = math.Abs(v)
		}
	}
	return max
}

func (fe *AdvancedFeatureExtractor) percentile(sorted []float64, p float64) float64 {
	if len(sorted) == 0 {
		return 0
	}
	index := (p / 100.0) * float64(len(sorted)-1)
	i := int(index)
	f := index - float64(i)
	if i+1 < len(sorted) {
		return sorted[i]*(1-f) + sorted[i+1]*f
	}
	return sorted[i]
}

func (fe *AdvancedFeatureExtractor) calculateSkewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := fe.mean(values)
	stdDev := fe.stdDev(values)
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (fe *AdvancedFeatureExtractor) calculateKurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	mean := fe.mean(values)
	stdDev := fe.stdDev(values)
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	n := float64(len(values))
	return (sum / n) - 3.0
}

func (fe *AdvancedFeatureExtractor) sortFloat64(values []float64) {
	for i := 0; i < len(values)-1; i++ {
		for j := i + 1; j < len(values); j++ {
			if values[i] > values[j] {
				values[i], values[j] = values[j], values[i]
			}
		}
	}
}

type EnhancedModelTrainer struct {
	mu            sync.RWMutex
	isTraining    bool
	lastTrainTime time.Time
	modelVersion  int
	trainingData  []EnhancedTrainingSample
}

type EnhancedTrainingSample struct {
	Features  []float64
	Label     float64
	Weight    float64
	Timestamp time.Time
}

func NewEnhancedModelTrainer() *EnhancedModelTrainer {
	return &EnhancedModelTrainer{
		modelVersion: 1,
		trainingData: make([]EnhancedTrainingSample, 0),
	}
}

func (mt *EnhancedModelTrainer) AddSample(features []float64, label float64) {
	mt.mu.Lock()
	defer mt.mu.Unlock()

	sample := EnhancedTrainingSample{
		Features:  features,
		Label:     label,
		Weight:    1.0,
		Timestamp: time.Now(),
	}

	mt.trainingData = append(mt.trainingData, sample)

	if len(mt.trainingData) > 10000 {
		mt.trainingData = mt.trainingData[len(mt.trainingData)-10000:]
	}

	if len(mt.trainingData)%100 == 0 && !mt.isTraining {
		go mt.trainModel()
	}
}

func (mt *EnhancedModelTrainer) trainModel() {
	mt.mu.Lock()
	if mt.isTraining {
		mt.mu.Unlock()
		return
	}
	mt.isTraining = true
	mt.mu.Unlock()

	time.Sleep(100 * time.Millisecond)

	mt.mu.Lock()
	mt.modelVersion++
	mt.lastTrainTime = time.Now()
	mt.isTraining = false
	mt.mu.Unlock()
}

func (mt *EnhancedModelTrainer) GetModelVersion() int {
	mt.mu.RLock()
	defer mt.mu.RUnlock()
	return mt.modelVersion
}

type EnhancedPredictionEngine struct {
	mu         sync.RWMutex
	thresholds *EnhancedPredictionThresholds
	cache      *EnhancedPredictionCache
}

type EnhancedPredictionThresholds struct {
	BotThreshold      float64 `json:"bot_threshold"`
	AnomalyThreshold  float64 `json:"anomaly_threshold"`
	ConfidenceMin     float64 `json:"confidence_min"`
	PredictionLatency float64 `json:"prediction_latency_ms"`
}

type EnhancedPredictionCache struct {
	mu      sync.RWMutex
	entries map[string]*EnhancedCachedPrediction
	maxSize int
}

type EnhancedCachedPrediction struct {
	Prediction *EnhancedPredictionResult
	ExpiresAt  time.Time
}

type EnhancedPredictionResult struct {
	BotProbability        float64            `json:"bot_probability"`
	AnomalyScore          float64            `json:"anomaly_score"`
	RiskLevel             string             `json:"risk_level"`
	Confidence            float64            `json:"confidence"`
	LatencyMs             float64            `json:"latency_ms"`
	FeatureContributions  map[string]float64 `json:"feature_contributions"`
	Timestamp             time.Time          `json:"timestamp"`
}

func NewEnhancedPredictionEngine() *EnhancedPredictionEngine {
	return &EnhancedPredictionEngine{
		thresholds: &EnhancedPredictionThresholds{
			BotThreshold:      0.5,
			AnomalyThreshold:  0.5,
			ConfidenceMin:     0.3,
			PredictionLatency: 100,
		},
		cache: &EnhancedPredictionCache{
			entries: make(map[string]*EnhancedCachedPrediction),
			maxSize: 1000,
		},
	}
}

func (pe *EnhancedPredictionEngine) Predict(features *ComprehensiveFeatures) *EnhancedPredictionResult {
	startTime := time.Now()

	result := &EnhancedPredictionResult{
		FeatureContributions: make(map[string]float64),
		Timestamp:            startTime,
	}

	if features.DerivedFeatures != nil {
		result.BotProbability = features.DerivedFeatures.BotProbability
		result.AnomalyScore = features.DerivedFeatures.AnomalyScore
		result.RiskLevel = features.DerivedFeatures.RiskLevel
		result.Confidence = features.DerivedFeatures.ConfidenceLevel

		result.FeatureContributions["bot_probability"] = result.BotProbability * 0.4
		result.FeatureContributions["anomaly_score"] = result.AnomalyScore * 0.3
		result.FeatureContributions["speed_variance"] = pe.evaluateSpeedVariance(features.BasicFeatures)
		result.FeatureContributions["path_ratio"] = pe.evaluatePathRatio(features.PatternFeatures)
		result.FeatureContributions["pause_count"] = pe.evaluatePauseCount(features.BasicFeatures)
		result.FeatureContributions["curvature"] = pe.evaluateCurvature(features.PatternFeatures)
	}

	result.LatencyMs = float64(time.Since(startTime).Milliseconds())

	return result
}

func (pe *EnhancedPredictionEngine) evaluateSpeedVariance(bf *BasicFeatures) float64 {
	if bf == nil {
		return 0.5
	}
	if bf.SpeedVariance < 5 && bf.AverageSpeed > 50 {
		return 0.8
	}
	return math.Min(1.0, bf.SpeedVariance/100.0)
}

func (pe *EnhancedPredictionEngine) evaluatePathRatio(pf *PatternFeatures) float64 {
	if pf == nil {
		return 0.5
	}
	if pf.PathRatio < 1.05 {
		return 0.9
	}
	if pf.PathRatio < 1.2 {
		return 0.6
	}
	return 0.2
}

func (pe *EnhancedPredictionEngine) evaluatePauseCount(bf *BasicFeatures) float64 {
	if bf == nil {
		return 0.5
	}
	if bf.PauseCount == 0 && bf.PointCount > 20 {
		return 0.7
	}
	return math.Min(1.0, float64(bf.PauseCount)/10.0)
}

func (pe *EnhancedPredictionEngine) evaluateCurvature(pf *PatternFeatures) float64 {
	if pf == nil {
		return 0.5
	}
	if pf.CurvatureVariance < 0.001 {
		return 0.8
	}
	return math.Min(1.0, pf.CurvatureVariance*10)
}

type EnhancedPerformanceTracker struct {
	mu           sync.RWMutex
	measurements []EnhancedPerformanceMeasurement
	maxMeasurements int
}

type EnhancedPerformanceMeasurement struct {
	Operation     string    `json:"operation"`
	DurationMs    float64   `json:"duration_ms"`
	MemoryUsageMB float64   `json:"memory_usage_mb"`
	Timestamp     time.Time `json:"timestamp"`
}

func NewEnhancedPerformanceTracker() *EnhancedPerformanceTracker {
	return &EnhancedPerformanceTracker{
		measurements:    make([]EnhancedPerformanceMeasurement, 0),
		maxMeasurements: 1000,
	}
}

func (pt *EnhancedPerformanceTracker) Record(operation string, durationMs float64, memoryMB float64) {
	pt.mu.Lock()
	defer pt.mu.Unlock()

	measurement := EnhancedPerformanceMeasurement{
		Operation:     operation,
		DurationMs:    durationMs,
		MemoryUsageMB: memoryMB,
		Timestamp:     time.Now(),
	}

	pt.measurements = append(pt.measurements, measurement)

	if len(pt.measurements) > pt.maxMeasurements {
		pt.measurements = pt.measurements[len(pt.measurements)-pt.maxMeasurements:]
	}
}

func (pt *EnhancedPerformanceTracker) GetAverageLatency(operation string) float64 {
	pt.mu.RLock()
	defer pt.mu.RUnlock()

	var total float64
	var count int
	for _, m := range pt.measurements {
		if m.Operation == operation {
			total += m.DurationMs
			count++
		}
	}

	if count > 0 {
		return total / float64(count)
	}
	return 0
}

func NewComprehensiveAnalysisService() *ComprehensiveAnalysisService {
	return &ComprehensiveAnalysisService{
		collector:          NewEnhancedBehaviorCollector(),
		featureExtractor:   NewAdvancedFeatureExtractor(),
		modelTrainer:       NewEnhancedModelTrainer(),
		predictionEngine:   NewEnhancedPredictionEngine(),
		performanceTracker: NewEnhancedPerformanceTracker(),
	}
}

func (cas *ComprehensiveAnalysisService) AnalyzeComprehensively(
	behaviorData []BehaviorDataItem,
) (*ComprehensiveAnalysisResult, error) {
	startTime := time.Now()

	var points []BehaviorDataPoint
	var clicks []BehaviorDataPoint
	var keyStrokes []KeyboardDataPoint

	for _, bd := range behaviorData {
		switch bd.DataType {
		case "keyboard":
			var kp KeyboardDataPoint
			if err := json.Unmarshal([]byte(bd.Data), &kp); err == nil {
				keyStrokes = append(keyStrokes, kp)
			}
		case "click":
			var dp BehaviorDataPoint
			if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
				clicks = append(clicks, dp)
				points = append(points, dp)
			}
		default:
			var dp BehaviorDataPoint
			if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
				points = append(points, dp)
			}
		}
	}

	comprehensiveFeatures := cas.featureExtractor.ExtractComprehensiveFeatures(points, clicks, keyStrokes)

	prediction := cas.predictionEngine.Predict(comprehensiveFeatures)

	features := cas.extractFeatureVector(comprehensiveFeatures)
	cas.modelTrainer.AddSample(features, prediction.BotProbability)

	latencyMs := float64(time.Since(startTime).Milliseconds())

	result := &ComprehensiveAnalysisResult{
		ComprehensiveFeatures: comprehensiveFeatures,
		Prediction:            prediction,
		LatencyMs:             latencyMs,
		DataCompleteness:      cas.calculateDataCompleteness(points, clicks, keyStrokes),
		Accuracy:              cas.estimateAccuracy(comprehensiveFeatures),
	}

	return result, nil
}

func (cas *ComprehensiveAnalysisService) extractFeatureVector(features *ComprehensiveFeatures) []float64 {
	vector := make([]float64, 0)

	if features.BasicFeatures != nil {
		vector = append(vector, float64(features.BasicFeatures.PointCount))
		vector = append(vector, features.BasicFeatures.TotalDistance)
		vector = append(vector, features.BasicFeatures.DirectDistance)
		vector = append(vector, features.BasicFeatures.TotalDuration)
		vector = append(vector, features.BasicFeatures.AverageSpeed)
		vector = append(vector, features.BasicFeatures.MaxSpeed)
		vector = append(vector, features.BasicFeatures.MinSpeed)
		vector = append(vector, features.BasicFeatures.SpeedVariance)
		vector = append(vector, float64(features.BasicFeatures.PauseCount))
		vector = append(vector, float64(features.BasicFeatures.ClickCount))
	}

	if features.PatternFeatures != nil {
		vector = append(vector, features.PatternFeatures.PathRatio)
		vector = append(vector, features.PatternFeatures.Sinuosity)
		vector = append(vector, features.PatternFeatures.CurvatureMean)
		vector = append(vector, features.PatternFeatures.CurvatureVariance)
		vector = append(vector, features.PatternFeatures.DirectionEntropy)
		vector = append(vector, float64(features.PatternFeatures.DirectionChanges))
	}

	if features.DerivedFeatures != nil {
		vector = append(vector, features.DerivedFeatures.BotProbability)
		vector = append(vector, features.DerivedFeatures.AnomalyScore)
	}

	return vector
}

func (cas *ComprehensiveAnalysisService) calculateDataCompleteness(
	points []BehaviorDataPoint,
	clicks []BehaviorDataPoint,
	keyStrokes []KeyboardDataPoint,
) float64 {
	completeness := 0.0

	if len(points) >= 20 {
		completeness += 0.4
	} else if len(points) >= 10 {
		completeness += 0.2
	}

	if len(clicks) > 0 {
		completeness += 0.2
	}

	if len(keyStrokes) > 0 {
		completeness += 0.2
	}

	if len(points) >= 2 {
		var totalDuration float64
		if points[len(points)-1].Timestamp > points[0].Timestamp {
			totalDuration = float64(points[len(points)-1].Timestamp - points[0].Timestamp)
		}
		if totalDuration > 500 {
			completeness += 0.2
		}
	}

	return math.Min(1.0, completeness)
}

func (cas *ComprehensiveAnalysisService) estimateAccuracy(features *ComprehensiveFeatures) float64 {
	if features.DerivedFeatures == nil {
		return 0.5
	}

	baseAccuracy := features.DerivedFeatures.ConfidenceLevel

	completenessBonus := 0.0
	if features.DerivedFeatures.FeatureCompleteness > 0.9 {
		completenessBonus = 0.05
	}

	return math.Min(0.98, baseAccuracy+completenessBonus)
}

type ComprehensiveAnalysisResult struct {
	ComprehensiveFeatures *ComprehensiveFeatures     `json:"comprehensive_features"`
	Prediction           *EnhancedPredictionResult  `json:"prediction"`
	LatencyMs            float64                   `json:"latency_ms"`
	DataCompleteness     float64                   `json:"data_completeness"`
	Accuracy             float64                   `json:"estimated_accuracy"`
}

func GenerateSyntheticBehaviorData(count int, isBot bool) []BehaviorDataItem {
	data := make([]BehaviorDataItem, 0, count)

	startTime := time.Now().UnixMilli()

	for i := 0; i < count; i++ {
		var x, y int
		var event string

		if isBot {
			progress := float64(i) / float64(count)
			x = int(progress * 400)
			y = int(progress * 300)
			event = "move"
		} else {
			x = rand.Intn(1920)
			y = rand.Intn(1080)
			x += rand.Intn(20) - 10
			y += rand.Intn(20) - 10

			if rand.Float64() < 0.1 {
				event = "click"
			} else {
				event = "move"
			}
		}

		dp := BehaviorDataPoint{
			X:         x,
			Y:         y,
			Timestamp: startTime + int64(i*16),
			Event:     event,
		}

		jsonData, _ := json.Marshal(dp)

		data = append(data, BehaviorDataItem{
			DataType:  event,
			Data:      string(jsonData),
			Timestamp: startTime + int64(i*16),
		})
	}

	return data
}

func BenchmarkBehaviorAnalysis(data []BehaviorDataItem) map[string]interface{} {
	service := NewComprehensiveAnalysisService()

	startTime := time.Now()
	result, _ := service.AnalyzeComprehensively(data)
	latency := time.Since(startTime).Milliseconds()

	return map[string]interface{}{
		"latency_ms":         latency,
		"data_points":        len(data),
		"bot_probability":    result.Prediction.BotProbability,
		"anomaly_score":      result.Prediction.AnomalyScore,
		"risk_level":         result.Prediction.RiskLevel,
		"data_completeness":  result.DataCompleteness,
		"estimated_accuracy": result.Accuracy,
	}
}
