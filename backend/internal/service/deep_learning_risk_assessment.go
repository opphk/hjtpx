package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

// ============================================
// 增强深度学习风险评估服务
// ============================================

type DeepLearningRiskAssessmentService struct {
	featureExtractor       *AdvancedFeatureExtractor
	deepPredictor          *DeepNeuralNetworkPredictor
	anomalyDetector        *EnhancedAnomalyDetector
	ensembleClassifier     *EnsembleClassifier
	userBehaviorProfiles   *UserBehaviorProfiles
	realTimeUpdater        *RealTimeModelUpdater
	loaded                 bool
	mu                     sync.RWMutex
}

type DeepRiskResult struct {
	RiskScore       float64              `json:"risk_score"`
	RiskLevel       string               `json:"risk_level"`
	IsBot           bool                 `json:"is_bot"`
	Confidence      float64              `json:"confidence"`
	ProcessingTime  time.Duration        `json:"processing_time"`
	AnomalyPatterns []string             `json:"anomaly_patterns"`
	FeatureImportance map[string]float64  `json:"feature_importance"`
	ModelVersion    string               `json:"model_version"`
}

func NewDeepLearningRiskAssessmentService() *DeepLearningRiskAssessmentService {
	return &DeepLearningRiskAssessmentService{
		featureExtractor:     NewAdvancedFeatureExtractor(),
		deepPredictor:        NewDeepNeuralNetworkPredictor(),
		anomalyDetector:      NewEnhancedAnomalyDetector(),
		ensembleClassifier:   NewEnsembleClassifier(),
		userBehaviorProfiles: NewUserBehaviorProfiles(),
		realTimeUpdater:      NewRealTimeModelUpdater(),
	}
}

func (s *DeepLearningRiskAssessmentService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return nil
	}
	s.deepPredictor.Initialize(ctx)
	s.realTimeUpdater.Start(ctx)
	s.loaded = true
	return nil
}

func (s *DeepLearningRiskAssessmentService) AssessRisk(ctx context.Context, traceData *model.TraceData, userID string) (*DeepRiskResult, error) {
	start := time.Now()

	if traceData == nil || len(traceData.Points) < 2 {
		return &DeepRiskResult{
			RiskScore:      0.5,
			RiskLevel:      "unknown",
			IsBot:          false,
			Confidence:     0.5,
			ProcessingTime: time.Since(start),
			ModelVersion:   "v2.0",
		}, nil
	}

	features, _ := s.featureExtractor.ExtractAdvancedFeatures(ctx, traceData)
	
	predictorScore, _ := s.deepPredictor.Predict(ctx, features)
	
	anomalies := s.anomalyDetector.DetectAdvanced(ctx, traceData)
	anomalyScore := float64(len(anomalies)) / float64(len(EnhancedAnomalyTypeList))
	
	profileScore := s.userBehaviorProfiles.CompareWithProfile(userID, traceData)
	
	behaviorFeatures := &BehaviorFeatures{
		RiskScore: predictorScore,
	}
	_, ensembleScore := s.ensembleClassifier.Classify(behaviorFeatures)
	
	anomalyPatterns := make([]string, 0)
	for _, anomaly := range anomalies {
		anomalyPatterns = append(anomalyPatterns, anomaly.String())
	}

	return &DeepRiskResult{
		RiskScore:       ensembleScore,
		RiskLevel:       s.determineRiskLevel(ensembleScore),
		IsBot:           ensembleScore >= 0.5,
		Confidence:      math.Max(0.6, ensembleScore),
		ProcessingTime:  time.Since(start),
		AnomalyPatterns: anomalyPatterns,
		FeatureImportance: map[string]float64{
			"predictor_score": predictorScore,
			"anomaly_score":   anomalyScore,
			"profile_score":   profileScore,
		},
		ModelVersion: "v2.0",
	}, nil
}

func (s *DeepLearningRiskAssessmentService) determineRiskLevel(score float64) string {
	switch {
	case score >= 0.95:
		return "critical"
	case score >= 0.8:
		return "high"
	case score >= 0.6:
		return "medium"
	case score >= 0.4:
		return "low"
	default:
		return "none"
	}
}

func (s *DeepLearningRiskAssessmentService) UpdateModel(ctx context.Context, feedbackData *RiskFeedback) error {
	return s.realTimeUpdater.Update(ctx, feedbackData)
}

func (s *DeepLearningRiskAssessmentService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

// ============================================
// 高级特征提取器 (扩展768维到1024维)
// ============================================

const AdvancedFeatureVectorSize = 1024

type AdvancedFeatureExtractor struct {
	lstmExtractor *LSTMFeatureExtractor
}

func NewAdvancedFeatureExtractor() *AdvancedFeatureExtractor {
	return &AdvancedFeatureExtractor{
		lstmExtractor: NewLSTMFeatureExtractor(),
	}
}

func (e *AdvancedFeatureExtractor) ExtractAdvancedFeatures(ctx context.Context, traceData *model.TraceData) ([]float64, error) {
	baseFeatures, _ := e.lstmExtractor.ExtractFeatures(ctx, traceData)
	
	features := make([]float64, AdvancedFeatureVectorSize)
	copy(features, baseFeatures)
	
	if len(traceData.Points) >= 2 {
		e.extractTemporalFeatures(traceData, features, 768)
		e.extractSpatialFeatures(traceData, features, 868)
		e.extractBehavioralFeatures(traceData, features, 928)
		e.extractContextualFeatures(traceData, features, 978)
	}
	
	for i := range features {
		features[i] = aiNormalizeFeature(features[i], -100, 100)
	}
	
	return features, nil
}

func (e *AdvancedFeatureExtractor) extractTemporalFeatures(traceData *model.TraceData, features []float64, offset int) {
	points := traceData.Points
	
	timeDiffs := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		timeDiffs = append(timeDiffs, float64(points[i].Timestamp-points[i-1].Timestamp))
	}
	
	if len(timeDiffs) > 0 {
		features[offset] = aiMeanFloatSlice(timeDiffs)
		features[offset+1] = aiVarianceFloatSlice(timeDiffs)
		features[offset+2] = aiMinFloatSlice(timeDiffs)
		features[offset+3] = aiMaxFloatSlice(timeDiffs)
		features[offset+4] = aiCalculateEntropy(timeDiffs)
		
		features[offset+5] = e.calculateTimeAutocorrelation(timeDiffs, 1)
		features[offset+6] = e.calculateTimeAutocorrelation(timeDiffs, 2)
		features[offset+7] = e.calculateTimeAutocorrelation(timeDiffs, 3)
		
		features[offset+8] = e.detectPeriodicity(timeDiffs)
		features[offset+9] = e.calculateBurstiness(timeDiffs)
		
		features[offset+10] = e.calculateAccelerationJerk(timeDiffs)
	}
	
	for lag := 0; lag < 87 && offset+11+lag < AdvancedFeatureVectorSize; lag++ {
		features[offset+11+lag] = aiAutocorrelation(timeDiffs, lag+1)
	}
}

func (e *AdvancedFeatureExtractor) extractSpatialFeatures(traceData *model.TraceData, features []float64, offset int) {
	points := traceData.Points
	
	xCoords := make([]float64, len(points))
	yCoords := make([]float64, len(points))
	for i, p := range points {
		xCoords[i] = p.X
		yCoords[i] = p.Y
	}
	
	fftX := aiFFT(xCoords)
	fftY := aiFFT(yCoords)
	
	for i := 0; i < 30 && offset+i < AdvancedFeatureVectorSize; i++ {
		features[offset+i] = math.Abs(real(fftX[i]))
	}
	for i := 0; i < 30 && offset+30+i < AdvancedFeatureVectorSize; i++ {
		features[offset+30+i] = math.Abs(real(fftY[i]))
	}
	
	features[offset+60] = e.calculatePathComplexity(points)
	features[offset+61] = e.calculateFractalDimension(points)
}

func (e *AdvancedFeatureExtractor) extractBehavioralFeatures(traceData *model.TraceData, features []float64, offset int) {
	points := traceData.Points
	
	pauseCount := 0
	totalPauseDuration := 0.0
	turningAngles := make([]float64, 0)
	
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 100 && dt < 500 {
			pauseCount++
			totalPauseDuration += dt
		}
		
		if i > 1 {
			dx1 := points[i-1].X - points[i-2].X
			dy1 := points[i-1].Y - points[i-2].Y
			dx2 := points[i].X - points[i-1].X
			dy2 := points[i].Y - points[i-1].Y
			
			dot := dx1*dx2 + dy1*dy2
			mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
			mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
			
			if mag1 > 0 && mag2 > 0 {
				cosAngle := dot / (mag1 * mag2)
				if cosAngle > 1 {
					cosAngle = 1
				}
				if cosAngle < -1 {
					cosAngle = -1
				}
				turningAngles = append(turningAngles, math.Abs(math.Acos(cosAngle)))
			}
		}
	}
	
	features[offset] = float64(pauseCount)
	features[offset+1] = totalPauseDuration
	features[offset+2] = aiMeanFloatSlice(turningAngles)
	features[offset+3] = aiVarianceFloatSlice(turningAngles)
	features[offset+4] = aiCalculateEntropy(turningAngles)
	
	features[offset+5] = e.calculateMicroJitter(points)
	features[offset+6] = e.calculateStrokeSpeedVariability(points)
	features[offset+7] = e.detectStealthPattern(points)
	features[offset+8] = e.calculateWritingPressure(points)
	features[offset+9] = e.detectMouseWheelPattern(points)
	
	for i := 0; i < 40 && offset+10+i < AdvancedFeatureVectorSize; i++ {
		features[offset+10+i] = float64(hashBehaviorPattern(points, i)) / float64(len(points))
	}
}

func (e *AdvancedFeatureExtractor) extractContextualFeatures(traceData *model.TraceData, features []float64, offset int) {
	points := traceData.Points
	
	screenWidth := 1920.0
	screenHeight := 1080.0
	
	minX, maxX, minY, maxY := points[0].X, points[0].X, points[0].Y, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	
	features[offset] = (maxX - minX) / screenWidth
	features[offset+1] = (maxY - minY) / screenHeight
	features[offset+2] = float64(len(points)) / float64(traceData.TotalTime) * 1000
	
	features[offset+3] = e.detectEdgeHugging(points, screenWidth, screenHeight)
	features[offset+4] = e.calculateScreenCoverage(points, screenWidth, screenHeight)
	features[offset+5] = e.detectTargetedMovement(points)
	
	for i := 0; i < 20 && offset+6+i < AdvancedFeatureVectorSize; i++ {
		features[offset+6+i] = e.extractContextSignature(points, i)
	}
}

func (e *AdvancedFeatureExtractor) calculateTimeAutocorrelation(values []float64, lag int) float64 {
	return aiAutocorrelation(values, lag)
}

func (e *AdvancedFeatureExtractor) detectPeriodicity(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	fft := aiFFT(values)
	maxAmplitude := 0.0
	for _, v := range fft {
		maxAmplitude = math.Max(maxAmplitude, math.Abs(real(v)))
	}
	return maxAmplitude / float64(len(values))
}

func (e *AdvancedFeatureExtractor) calculateBurstiness(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := aiMeanFloatSlice(values)
	if mean == 0 {
		return 0
	}
	stdDev := math.Sqrt(aiVarianceFloatSlice(values))
	return stdDev / mean
}

func (e *AdvancedFeatureExtractor) calculateAccelerationJerk(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	accelerations := make([]float64, len(values)-1)
	for i := 1; i < len(values); i++ {
		accelerations[i-1] = values[i] - values[i-1]
	}
	jerks := make([]float64, len(accelerations)-1)
	for i := 1; i < len(accelerations); i++ {
		jerks[i-1] = math.Abs(accelerations[i] - accelerations[i-1])
	}
	return aiMeanFloatSlice(jerks)
}

func (e *AdvancedFeatureExtractor) calculatePathComplexity(points []model.TracePoint) float64 {
	if len(points) < 3 {
		return 0
	}
	totalDistance := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	start, end := points[0], points[len(points)-1]
	straightDistance := math.Sqrt((end.X-start.X)*(end.X-start.X) + (end.Y-start.Y)*(end.Y-start.Y))
	if straightDistance == 0 {
		return 0
	}
	return totalDistance / straightDistance
}

func (e *AdvancedFeatureExtractor) calculateFractalDimension(points []model.TracePoint) float64 {
	if len(points) < 10 {
		return 1
	}
	levels := 3
	dimension := 0.0
	for level := 1; level <= levels; level++ {
		scale := float64(level) * 10.0
		count := 0
		prevX, prevY := points[0].X, points[0].Y
		for _, p := range points {
			dx := p.X - prevX
			dy := p.Y - prevY
			if math.Sqrt(dx*dx+dy*dy) >= scale {
				count++
				prevX, prevY = p.X, p.Y
			}
		}
		if count > 0 {
			dimension += math.Log(float64(count)) / math.Log(1/scale)
		}
	}
	return dimension / float64(levels)
}

func (e *AdvancedFeatureExtractor) calculateMicroJitter(points []model.TracePoint) float64 {
	if len(points) < 5 {
		return 0
	}
	jitterSum := 0.0
	for i := 2; i < len(points); i++ {
		dx1 := points[i-1].X - points[i-2].X
		dy1 := points[i-1].Y - points[i-2].Y
		dx2 := points[i].X - points[i-1].X
		dy2 := points[i].Y - points[i-1].Y
		jitterSum += math.Abs((dx1*dx2 + dy1*dy2) / (math.Sqrt(dx1*dx1+dy1*dy1) * math.Sqrt(dx2*dx2+dy2*dy2) + 1e-10))
	}
	return jitterSum / float64(len(points)-2)
}

func (e *AdvancedFeatureExtractor) calculateStrokeSpeedVariability(points []model.TracePoint) float64 {
	if len(points) < 3 {
		return 0
	}
	speeds := make([]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds[i-1] = math.Sqrt(dx*dx+dy*dy) / dt
		}
	}
	return aiVarianceFloatSlice(speeds)
}

func (e *AdvancedFeatureExtractor) detectStealthPattern(points []model.TracePoint) float64 {
	if len(points) < 5 {
		return 0
	}
	suspiciousCount := 0
	for i := 2; i < len(points)-2; i++ {
		prevSpeed := math.Sqrt((points[i].X-points[i-1].X)*(points[i].X-points[i-1].X) +
			(points[i].Y-points[i-1].Y)*(points[i].Y-points[i-1].Y))
		currSpeed := math.Sqrt((points[i+1].X-points[i].X)*(points[i+1].X-points[i].X) +
			(points[i+1].Y-points[i].Y)*(points[i+1].Y-points[i].Y))
		nextSpeed := math.Sqrt((points[i+2].X-points[i+1].X)*(points[i+2].X-points[i+1].X) +
			(points[i+2].Y-points[i+1].Y)*(points[i+2].Y-points[i+1].Y))
		
		if prevSpeed > 0 && nextSpeed > 0 && currSpeed < prevSpeed*0.1 && currSpeed < nextSpeed*0.1 {
			suspiciousCount++
		}
	}
	return float64(suspiciousCount) / float64(len(points))
}

func (e *AdvancedFeatureExtractor) calculateWritingPressure(points []model.TracePoint) float64 {
	if len(points) < 3 {
		return 0
	}
	pressureSum := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			pressureSum += math.Sqrt(dx*dx+dy*dy) / dt
		}
	}
	return pressureSum / float64(len(points)-1)
}

func (e *AdvancedFeatureExtractor) detectMouseWheelPattern(points []model.TracePoint) float64 {
	if len(points) < 10 {
		return 0
	}
	wheelCount := 0
	for i := 5; i < len(points)-5; i++ {
		isWheelPattern := true
		for j := -5; j <= 5; j++ {
			if j != 0 {
				dx := points[i+j].X - points[i].X
				dy := points[i+j].Y - points[i].Y
				if math.Abs(dx) > 2 || math.Abs(dy) > 2 {
					isWheelPattern = false
					break
				}
			}
		}
		if isWheelPattern {
			wheelCount++
		}
	}
	return float64(wheelCount) / float64(len(points))
}

func (e *AdvancedFeatureExtractor) detectEdgeHugging(points []model.TracePoint, screenWidth, screenHeight float64) float64 {
	if len(points) == 0 {
		return 0
	}
	edgeCount := 0
	for _, p := range points {
		if p.X < 10 || p.X > screenWidth-10 || p.Y < 10 || p.Y > screenHeight-10 {
			edgeCount++
		}
	}
	return float64(edgeCount) / float64(len(points))
}

func (e *AdvancedFeatureExtractor) calculateScreenCoverage(points []model.TracePoint, screenWidth, screenHeight float64) float64 {
	if len(points) < 2 {
		return 0
	}
	minX, maxX, minY, maxY := points[0].X, points[0].X, points[0].Y, points[0].Y
	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	return ((maxX-minX) * (maxY-minY)) / (screenWidth * screenHeight)
}

func (e *AdvancedFeatureExtractor) detectTargetedMovement(points []model.TracePoint) float64 {
	if len(points) < 5 {
		return 0
	}
	targetedCount := 0
	for i := 0; i < len(points)-5; i += 5 {
		group := points[i : i+5]
		minX, maxX, minY, maxY := group[0].X, group[0].X, group[0].Y, group[0].Y
		for _, p := range group {
			if p.X < minX {
				minX = p.X
			}
			if p.X > maxX {
				maxX = p.X
			}
			if p.Y < minY {
				minY = p.Y
			}
			if p.Y > maxY {
				maxY = p.Y
			}
		}
		area := (maxX - minX) * (maxY - minY)
		if area < 100 {
			targetedCount++
		}
	}
	return float64(targetedCount) / float64(len(points)/5)
}

func (e *AdvancedFeatureExtractor) extractContextSignature(points []model.TracePoint, seed int) float64 {
	if len(points) == 0 {
		return 0
	}
	sum := 0.0
	for i, p := range points {
		sum += float64(i*seed) * p.X * p.Y / float64(len(points))
	}
	return sum / 1000000.0
}

func hashBehaviorPattern(points []model.TracePoint, seed int) int {
	hash := seed
	for _, p := range points {
		hash = (hash * 31) ^ int(p.X)
		hash = (hash * 31) ^ int(p.Y)
		hash = (hash * 31) ^ int(p.Timestamp)
	}
	return hash & 0xFF
}

// ============================================
// 深度神经网络预测器
// ============================================

type DeepNeuralNetworkPredictor struct {
	initialized bool
	weights     []float64
	mu          sync.RWMutex
}

func NewDeepNeuralNetworkPredictor() *DeepNeuralNetworkPredictor {
	return &DeepNeuralNetworkPredictor{
		weights: make([]float64, AdvancedFeatureVectorSize),
	}
}

func (p *DeepNeuralNetworkPredictor) Initialize(ctx context.Context) error {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for i := range p.weights {
		p.weights[i] = (rand.Float64() - 0.5) * 0.1
	}
	
	p.weights[0] = 0.5
	p.weights[1] = 0.3
	p.weights[2] = 0.4
	p.weights[3] = -0.2
	p.weights[4] = 0.35
	
	p.initialized = true
	return nil
}

func (p *DeepNeuralNetworkPredictor) Predict(ctx context.Context, features []float64) (float64, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()
	
	if !p.initialized {
		return 0.5, nil
	}
	
	sum := 0.0
	minLen := aiMin(len(features), len(p.weights))
	for i := 0; i < minLen; i++ {
		sum += features[i] * p.weights[i]
	}
	
	hidden := p.applyHiddenLayer(sum)
	
	return aiSigmoid(hidden), nil
}

func (p *DeepNeuralNetworkPredictor) applyHiddenLayer(input float64) float64 {
	hidden1 := math.Tanh(input * 0.5)
	hidden2 := math.Tanh(input * 0.3)
	return hidden1*0.6 + hidden2*0.4
}

func (p *DeepNeuralNetworkPredictor) UpdateWeights(gradients []float64, learningRate float64) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	for i := 0; i < aiMin(len(gradients), len(p.weights)); i++ {
		p.weights[i] += gradients[i] * learningRate
	}
}

// ============================================
// 增强异常检测器
// ============================================

type EnhancedAnomalyType int

const (
	AnomalyRapidClicks EnhancedAnomalyType = iota
	AnomalyPatternRepetition
	AnomalyUnusualTiming
	AnomalyStraightLine
	AnomalyLowJitter
	AnomalyNoPause
	AnomalyNoMicroCorrection
	AnomalyUniformAcceleration
	AnomalyPathRepeating
	AnomalyHighSpeed
	AnomalyConstantSpeed
	AnomalyClusteredClicks
	AnomalySmallClickArea
	AnomalyShortHesitation
	AnomalyMechanicalPattern
	AnomalyBotTyping
	AnomalyCopyPaste
	AnomalyAutomatedNavigation
	AnomalySessionReplay
	AnomalyBehavioralBiometryMismatch
	AnomalyAbnormalCurvature
	AnomalyPredictablePattern
	AnomalyZeroVariance
	AnomalyExtremeAcceleration
	AnomalyDiscontinuousMovement
	AnomalyMouseWheelPattern
	AnomalyEdgeHugging
	AnomalyTargetedClicks
	AnomalyIdenticalPatterns
	AnomalyTimeSynchronization
)

var EnhancedAnomalyTypeList = []EnhancedAnomalyType{
	AnomalyRapidClicks,
	AnomalyPatternRepetition,
	AnomalyUnusualTiming,
	AnomalyStraightLine,
	AnomalyLowJitter,
	AnomalyNoPause,
	AnomalyNoMicroCorrection,
	AnomalyUniformAcceleration,
	AnomalyPathRepeating,
	AnomalyHighSpeed,
	AnomalyConstantSpeed,
	AnomalyClusteredClicks,
	AnomalySmallClickArea,
	AnomalyShortHesitation,
	AnomalyMechanicalPattern,
	AnomalyBotTyping,
	AnomalyCopyPaste,
	AnomalyAutomatedNavigation,
	AnomalySessionReplay,
	AnomalyBehavioralBiometryMismatch,
	AnomalyAbnormalCurvature,
	AnomalyPredictablePattern,
	AnomalyZeroVariance,
	AnomalyExtremeAcceleration,
	AnomalyDiscontinuousMovement,
	AnomalyMouseWheelPattern,
	AnomalyEdgeHugging,
	AnomalyTargetedClicks,
	AnomalyIdenticalPatterns,
	AnomalyTimeSynchronization,
}

func (a EnhancedAnomalyType) String() string {
	switch a {
	case AnomalyRapidClicks:
		return "快速点击"
	case AnomalyPatternRepetition:
		return "模式重复"
	case AnomalyUnusualTiming:
		return "异常时序"
	case AnomalyStraightLine:
		return "直线移动"
	case AnomalyLowJitter:
		return "低抖动"
	case AnomalyNoPause:
		return "无停顿"
	case AnomalyNoMicroCorrection:
		return "无微修正"
	case AnomalyUniformAcceleration:
		return "匀速加速"
	case AnomalyPathRepeating:
		return "路径重复"
	case AnomalyHighSpeed:
		return "超高速"
	case AnomalyConstantSpeed:
		return "恒定速度"
	case AnomalyClusteredClicks:
		return "点击聚集"
	case AnomalySmallClickArea:
		return "点击区域过小"
	case AnomalyShortHesitation:
		return "犹豫过短"
	case AnomalyMechanicalPattern:
		return "机械模式"
	case AnomalyBotTyping:
		return "机器人打字"
	case AnomalyCopyPaste:
		return "复制粘贴"
	case AnomalyAutomatedNavigation:
		return "自动化导航"
	case AnomalySessionReplay:
		return "会话重放"
	case AnomalyBehavioralBiometryMismatch:
		return "行为生物特征不匹配"
	case AnomalyAbnormalCurvature:
		return "异常曲率"
	case AnomalyPredictablePattern:
		return "可预测模式"
	case AnomalyZeroVariance:
		return "零方差"
	case AnomalyExtremeAcceleration:
		return "极端加速"
	case AnomalyDiscontinuousMovement:
		return "不连续移动"
	case AnomalyMouseWheelPattern:
		return "滚轮模式"
	case AnomalyEdgeHugging:
		return "边缘紧贴"
	case AnomalyTargetedClicks:
		return "目标点击"
	case AnomalyIdenticalPatterns:
		return "完全相同模式"
	case AnomalyTimeSynchronization:
		return "时间同步"
	default:
		return "未知异常"
	}
}

type EnhancedAnomalyDetector struct{}

func NewEnhancedAnomalyDetector() *EnhancedAnomalyDetector {
	return &EnhancedAnomalyDetector{}
}

func (d *EnhancedAnomalyDetector) DetectAdvanced(ctx context.Context, traceData *model.TraceData) []EnhancedAnomalyType {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil
	}
	
	var anomalies []EnhancedAnomalyType
	points := traceData.Points
	
	if d.detectRapidClicks(points) {
		anomalies = append(anomalies, AnomalyRapidClicks)
	}
	if d.detectStraightLine(points) {
		anomalies = append(anomalies, AnomalyStraightLine)
	}
	if d.detectHighSpeed(points) {
		anomalies = append(anomalies, AnomalyHighSpeed)
	}
	if d.detectConstantSpeed(points) {
		anomalies = append(anomalies, AnomalyConstantSpeed)
	}
	if d.detectLowJitter(points) {
		anomalies = append(anomalies, AnomalyLowJitter)
	}
	if d.detectNoPause(points) {
		anomalies = append(anomalies, AnomalyNoPause)
	}
	if d.detectNoMicroCorrection(points) {
		anomalies = append(anomalies, AnomalyNoMicroCorrection)
	}
	if d.detectUniformAcceleration(points) {
		anomalies = append(anomalies, AnomalyUniformAcceleration)
	}
	if d.detectAbnormalCurvature(points) {
		anomalies = append(anomalies, AnomalyAbnormalCurvature)
	}
	if d.detectPredictablePattern(points) {
		anomalies = append(anomalies, AnomalyPredictablePattern)
	}
	if d.detectZeroVariance(points) {
		anomalies = append(anomalies, AnomalyZeroVariance)
	}
	if d.detectExtremeAcceleration(points) {
		anomalies = append(anomalies, AnomalyExtremeAcceleration)
	}
	if d.detectDiscontinuousMovement(points) {
		anomalies = append(anomalies, AnomalyDiscontinuousMovement)
	}
	if d.detectMouseWheelPattern(points) {
		anomalies = append(anomalies, AnomalyMouseWheelPattern)
	}
	if d.detectEdgeHugging(points) {
		anomalies = append(anomalies, AnomalyEdgeHugging)
	}
	if d.detectTargetedClicks(points) {
		anomalies = append(anomalies, AnomalyTargetedClicks)
	}
	
	return anomalies
}

func (d *EnhancedAnomalyDetector) detectRapidClicks(points []model.TracePoint) bool {
	clickCount := 0
	for _, p := range points {
		if p.Event == "click" {
			clickCount++
		}
	}
	return clickCount >= 5
}

func (d *EnhancedAnomalyDetector) detectStraightLine(points []model.TracePoint) bool {
	if len(points) < 5 {
		return false
	}
	start, end := points[0], points[len(points)-1]
	straightDist := math.Sqrt((end.X-start.X)*(end.X-start.X) + (end.Y-start.Y)*(end.Y-start.Y))
	totalDist := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}
	return totalDist > 100 && straightDist/totalDist > 0.92
}

func (d *EnhancedAnomalyDetector) detectHighSpeed(points []model.TracePoint) bool {
	maxSpeed := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			maxSpeed = math.Max(maxSpeed, dist/dt)
		}
	}
	return maxSpeed > 10
}

func (d *EnhancedAnomalyDetector) detectConstantSpeed(points []model.TracePoint) bool {
	if len(points) < 10 {
		return false
	}
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	if len(speeds) < 5 {
		return false
	}
	mean := aiMeanFloatSlice(speeds)
	stdDev := math.Sqrt(aiVarianceFloatSlice(speeds))
	return mean > 0 && stdDev/mean < 0.05
}

func (d *EnhancedAnomalyDetector) detectLowJitter(points []model.TracePoint) bool {
	if len(points) < 10 {
		return false
	}
	totalJitter := 0.0
	for i := 2; i < len(points); i++ {
		dx1 := points[i-1].X - points[i-2].X
		dy1 := points[i-1].Y - points[i-2].Y
		dx2 := points[i].X - points[i-1].X
		dy2 := points[i].Y - points[i-1].Y
		totalJitter += math.Abs((dx1*dx2 + dy1*dy2) / (math.Sqrt(dx1*dx1+dy1*dy1) * math.Sqrt(dx2*dx2+dy2*dy2) + 1e-10))
	}
	return totalJitter/float64(len(points)-2) > 0.95
}

func (d *EnhancedAnomalyDetector) detectNoPause(points []model.TracePoint) bool {
	if len(points) < 20 {
		return false
	}
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 100 {
			return false
		}
	}
	return true
}

func (d *EnhancedAnomalyDetector) detectNoMicroCorrection(points []model.TracePoint) bool {
	if len(points) < 20 {
		return false
	}
	microCorrections := 0
	for i := 2; i < len(points); i++ {
		dx1 := points[i-1].X - points[i-2].X
		dy1 := points[i-1].Y - points[i-2].Y
		dx2 := points[i].X - points[i-1].X
		dy2 := points[i].Y - points[i-1].Y
		if math.Abs(dx1+dx2) < 5 && math.Abs(dy1+dy2) < 5 {
			microCorrections++
		}
	}
	return microCorrections == 0
}

func (d *EnhancedAnomalyDetector) detectUniformAcceleration(points []model.TracePoint) bool {
	if len(points) < 10 {
		return false
	}
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	if len(speeds) < 5 {
		return false
	}
	accels := make([]float64, len(speeds)-1)
	for i := 1; i < len(speeds); i++ {
		accels[i-1] = speeds[i] - speeds[i-1]
	}
	stdDev := math.Sqrt(aiVarianceFloatSlice(accels))
	return stdDev < 0.1
}

func (d *EnhancedAnomalyDetector) detectAbnormalCurvature(points []model.TracePoint) bool {
	if len(points) < 5 {
		return false
	}
	curvatures := make([]float64, len(points)-2)
	for i := 1; i < len(points)-1; i++ {
		curvatures[i-1] = aiComputeCurvature(points[i-1], points[i], points[i+1])
	}
	stdDev := math.Sqrt(aiVarianceFloatSlice(curvatures))
	return stdDev < 0.05
}

func (d *EnhancedAnomalyDetector) detectPredictablePattern(points []model.TracePoint) bool {
	if len(points) < 10 {
		return false
	}
	for i := 0; i < len(points)-5; i += 5 {
		for j := i + 5; j < len(points)-5; j += 5 {
			similar := true
			for k := 0; k < 5 && i+k < len(points) && j+k < len(points); k++ {
				dx1 := points[i+k].X - points[i].X
				dy1 := points[i+k].Y - points[i].Y
				dx2 := points[j+k].X - points[j].X
				dy2 := points[j+k].Y - points[j].Y
				if math.Abs(dx1-dx2) > 2 || math.Abs(dy1-dy2) > 2 {
					similar = false
					break
				}
			}
			if similar {
				return true
			}
		}
	}
	return false
}

func (d *EnhancedAnomalyDetector) detectZeroVariance(points []model.TracePoint) bool {
	if len(points) < 5 {
		return false
	}
	xCoords := make([]float64, len(points))
	yCoords := make([]float64, len(points))
	for i, p := range points {
		xCoords[i] = p.X
		yCoords[i] = p.Y
	}
	return aiVarianceFloatSlice(xCoords) < 0.1 && aiVarianceFloatSlice(yCoords) < 0.1
}

func (d *EnhancedAnomalyDetector) detectExtremeAcceleration(points []model.TracePoint) bool {
	if len(points) < 5 {
		return false
	}
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	if len(speeds) < 3 {
		return false
	}
	for i := 1; i < len(speeds); i++ {
		if math.Abs(speeds[i]-speeds[i-1]) > 5 {
			return true
		}
	}
	return false
}

func (d *EnhancedAnomalyDetector) detectDiscontinuousMovement(points []model.TracePoint) bool {
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		if math.Sqrt(dx*dx+dy*dy) > 100 {
			return true
		}
	}
	return false
}

func (d *EnhancedAnomalyDetector) detectMouseWheelPattern(points []model.TracePoint) bool {
	if len(points) < 10 {
		return false
	}
	stationaryCount := 0
	for i := 5; i < len(points)-5; i++ {
		isStationary := true
		for j := -5; j <= 5; j++ {
			if math.Abs(points[i+j].X-points[i].X) > 2 || math.Abs(points[i+j].Y-points[i].Y) > 2 {
				isStationary = false
				break
			}
		}
		if isStationary {
			stationaryCount++
		}
	}
	return stationaryCount > len(points)/10
}

func (d *EnhancedAnomalyDetector) detectEdgeHugging(points []model.TracePoint) bool {
	edgeCount := 0
	for _, p := range points {
		if p.X < 10 || p.X > 1910 || p.Y < 10 || p.Y > 1070 {
			edgeCount++
		}
	}
	return float64(edgeCount)/float64(len(points)) > 0.3
}

func (d *EnhancedAnomalyDetector) detectTargetedClicks(points []model.TracePoint) bool {
	clicks := []model.TracePoint{}
	for _, p := range points {
		if p.Event == "click" {
			clicks = append(clicks, p)
		}
	}
	if len(clicks) < 3 {
		return false
	}
	minX, maxX, minY, maxY := clicks[0].X, clicks[0].X, clicks[0].Y, clicks[0].Y
	for _, c := range clicks {
		if c.X < minX {
			minX = c.X
		}
		if c.X > maxX {
			maxX = c.X
		}
		if c.Y < minY {
			minY = c.Y
		}
		if c.Y > maxY {
			maxY = c.Y
		}
	}
	area := (maxX - minX) * (maxY - minY)
	return area < 1000
}

// ============================================
// 用户行为轮廓
// ============================================

type UserBehaviorProfiles struct {
	profiles map[string]*UserBehaviorProfile
	mu       sync.RWMutex
}

type UserBehaviorProfile struct {
	UserID           string
	AverageSpeed     float64
	AverageJitter    float64
	AverageCurvature float64
	PatternHash      string
	UpdateCount      int
}

func NewUserBehaviorProfiles() *UserBehaviorProfiles {
	return &UserBehaviorProfiles{
		profiles: make(map[string]*UserBehaviorProfile),
	}
}

func (p *UserBehaviorProfiles) CompareWithProfile(userID string, traceData *model.TraceData) float64 {
	p.mu.RLock()
	profile, exists := p.profiles[userID]
	p.mu.RUnlock()
	
	if !exists {
		return 0.5
	}
	
	points := traceData.Points
	if len(points) < 2 {
		return 0.5
	}
	
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	
	currentSpeed := aiMeanFloatSlice(speeds)
	speedDiff := math.Abs(currentSpeed - profile.AverageSpeed) / (profile.AverageSpeed + 1e-10)
	
	if speedDiff > 0.5 {
		return 0.8
	} else if speedDiff > 0.3 {
		return 0.6
	}
	return 0.2
}

func (p *UserBehaviorProfiles) UpdateProfile(userID string, traceData *model.TraceData) {
	p.mu.Lock()
	defer p.mu.Unlock()
	
	profile, exists := p.profiles[userID]
	if !exists {
		profile = &UserBehaviorProfile{UserID: userID}
		p.profiles[userID] = profile
	}
	
	points := traceData.Points
	if len(points) < 2 {
		return
	}
	
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	
	learningRate := 0.1
	profile.AverageSpeed = profile.AverageSpeed*(1-learningRate) + aiMeanFloatSlice(speeds)*learningRate
	profile.UpdateCount++
}

// ============================================
// 实时模型更新器
// ============================================

type RiskFeedback struct {
	SessionID    string
	Features     []float64
	ActualLabel  bool
	Confidence   float64
}

type RealTimeModelUpdater struct {
	running      bool
	feedbackChan chan *RiskFeedback
	mu           sync.RWMutex
}

func NewRealTimeModelUpdater() *RealTimeModelUpdater {
	return &RealTimeModelUpdater{
		feedbackChan: make(chan *RiskFeedback, 100),
	}
}

func (u *RealTimeModelUpdater) Start(ctx context.Context) {
	u.mu.Lock()
	if u.running {
		u.mu.Unlock()
		return
	}
	u.running = true
	u.mu.Unlock()
	
	go func() {
		for {
			select {
			case feedback := <-u.feedbackChan:
				u.processFeedback(feedback)
			case <-ctx.Done():
				return
			}
		}
	}()
}

func (u *RealTimeModelUpdater) Update(ctx context.Context, feedback *RiskFeedback) error {
	select {
	case u.feedbackChan <- feedback:
		return nil
	case <-time.After(1 * time.Second):
		return fmt.Errorf("feedback channel full")
	}
}

func (u *RealTimeModelUpdater) processFeedback(feedback *RiskFeedback) {
	
}

func (u *RealTimeModelUpdater) Stop() {
	u.mu.Lock()
	u.running = false
	u.mu.Unlock()
}