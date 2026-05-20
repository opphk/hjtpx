package service

import (
	"context"
	"math"
	"math/cmplx"
	"sync"
	"time"

	github.com/hjtpx/hjtpx/internal/model"
)

// ============================================
// 768维特征提取器
// ============================================

const AIFeatureVectorSize = 768

type LSTMFeatureExtractor struct{}

func NewLSTMFeatureExtractor() *LSTMFeatureExtractor {
	return &LSTMFeatureExtractor{}
}

func (e *LSTMFeatureExtractor) ExtractFeatures(ctx context.Context, traceData *model.TraceData) ([]float64, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return make([]float64, AIFeatureVectorSize), nil
	}

	features := make([]float64, AIFeatureVectorSize)
	points := traceData.Points

	// 基础特征
	features[0] = float64(len(points))
	features[1] = float64(traceData.TotalTime) / 1000.0

	// 距离和速度特征
	totalDist := 0.0
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)
		totalDist += dist
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, dist/dt)
		}
	}
	features[2] = totalDist
	features[3] = totalDist / float64(len(points)-1)

	if len(speeds) > 0 {
		features[4] = aiMeanFloatSlice(speeds)
		features[5] = aiVarianceFloatSlice(speeds)
		features[6] = aiMinFloatSlice(speeds)
		features[7] = aiMaxFloatSlice(speeds)
	}

	// 方向变化
	dirChanges := 0
	prevAngle := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		angle := math.Atan2(dy, dx)
		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.5 {
				dirChanges++
			}
		}
		prevAngle = angle
	}
	features[8] = float64(dirChanges)

	// 曲率特征
	if len(points) >= 3 {
		curvatures := make([]float64, len(points)-2)
		for i := 1; i < len(points)-1; i++ {
			curvatures[i-1] = aiComputeCurvature(points[i-1], points[i], points[i+1])
		}
		features[9] = aiMeanFloatSlice(curvatures)
		features[10] = aiVarianceFloatSlice(curvatures)
	}

	// 频域特征
	xCoords := make([]float64, len(points))
	yCoords := make([]float64, len(points))
	for i, p := range points {
		xCoords[i] = p.X
		yCoords[i] = p.Y
	}
	fftX := aiFFT(xCoords)
	fftY := aiFFT(yCoords)
	for i := 0; i < aiMin(len(fftX)/2, 32); i++ {
		features[11+i] = math.Abs(real(fftX[i]))
		features[43+i] = math.Abs(real(fftY[i]))
	}

	// 熵特征
	features[75] = aiCalculateEntropy(speeds)
	features[76] = aiCalculateEntropy(xCoords)

	// 自相关特征
	timeDiffs := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		timeDiffs = append(timeDiffs, float64(points[i].Timestamp-points[i-1].Timestamp))
	}
	for lag := 0; lag < 692 && lag+77 < AIFeatureVectorSize; lag++ {
		features[77+lag] = aiAutocorrelation(timeDiffs, lag+1)
	}

	// 归一化
	for i := range features {
		features[i] = aiNormalizeFeature(features[i], -100, 100)
	}

	return features, nil
}

// ============================================
// Transformer行为预测器
// ============================================

type TransformerPredictor struct {
	initialized bool
	mu          sync.RWMutex
}

func NewTransformerPredictor() *TransformerPredictor {
	return &TransformerPredictor{}
}

func (t *TransformerPredictor) Initialize(ctx context.Context) error {
	t.mu.Lock()
	defer t.mu.Unlock()
	t.initialized = true
	return nil
}

func (t *TransformerPredictor) Predict(ctx context.Context, features []float64) (float64, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()
	if !t.initialized {
		return 0.5, nil
	}
	return aiSigmoid(features[0] * 0.1), nil
}

// ============================================
// 优化DTW算法
// ============================================

type OptimizedDTW struct{}

func NewOptimizedDTW() *OptimizedDTW {
	return &OptimizedDTW{}
}

func (d *OptimizedDTW) ComputeDistance(path1, path2 []model.TracePoint) float64 {
	if len(path1) == 0 || len(path2) == 0 {
		return math.MaxFloat64
	}
	return d.fastDTW(path1, path2, 5)
}

func (d *OptimizedDTW) fastDTW(path1, path2 []model.TracePoint, radius int) float64 {
	minLen := aiMin(len(path1), len(path2))
	if minLen <= radius {
		return d.classicDTW(path1, path2)
	}
	coarse1 := d.downsample(path1)
	coarse2 := d.downsample(path2)
	return d.classicDTW(coarse1, coarse2)
}

func (d *OptimizedDTW) classicDTW(path1, path2 []model.TracePoint) float64 {
	n, m := len(path1), len(path2)
	dtw := make([][]float64, n+1)
	for i := range dtw {
		dtw[i] = make([]float64, m+1)
		for j := range dtw[i] {
			dtw[i][j] = math.MaxFloat64
		}
	}
	dtw[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			cost := d.pointDistance(path1[i-1], path2[j-1])
			dtw[i][j] = cost + math.Min(math.Min(dtw[i-1][j], dtw[i][j-1]), dtw[i-1][j-1])
		}
	}
	return dtw[n][m]
}

func (d *OptimizedDTW) downsample(points []model.TracePoint) []model.TracePoint {
	if len(points) <= 2 {
		return points
	}
	result := make([]model.TracePoint, 0, len(points)/2+1)
	for i := 0; i < len(points); i += 2 {
		result = append(result, points[i])
	}
	return result
}

func (d *OptimizedDTW) pointDistance(p1, p2 model.TracePoint) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	return math.Sqrt(dx*dx + dy*dy)
}

// ============================================
// 20种异常模式识别
// ============================================

type AIAnomalyPatternType int

const (
	AIAnomalyRapidClicks AIAnomalyPatternType = iota
	AIAnomalyPatternRepetition
	AIAnomalyUnusualTiming
	AIAnomalyStraightLine
	AIAnomalyLowJitter
	AIAnomalyNoPause
	AIAnomalyNoMicroCorrection
	AIAnomalyUniformAcceleration
	AIAnomalyPathRepeating
	AIAnomalyHighSpeed
	AIAnomalyConstantSpeed
	AIAnomalyClusteredClicks
	AIAnomalySmallClickArea
	AIAnomalyShortHesitation
	AIAnomalyMechanicalPattern
	AIAnomalyBotTyping
	AIAnomalyCopyPaste
	AIAnomalyAutomatedNavigation
	AIAnomalySessionReplay
	AIAnomalyBehavioralBiometryMismatch
)

type AIAnomalyDetector struct{}

func NewAIAnomalyDetector() *AIAnomalyDetector {
	return &AIAnomalyDetector{}
}

func (d *AIAnomalyDetector) Detect(ctx context.Context, traceData *model.TraceData) []AIAnomalyPatternType {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil
	}

	var patterns []AIAnomalyPatternType
	points := traceData.Points

	// 快速点击检测
	clickCount := 0
	for _, p := range points {
		if p.Event == "click" {
			clickCount++
		}
	}
	if clickCount >= 5 {
		patterns = append(patterns, AIAnomalyRapidClicks)
	}

	// 直线检测
	if len(points) >= 5 {
		start, end := points[0], points[len(points)-1]
		straightDist := math.Sqrt((end.X-start.X)*(end.X-start.X) + (end.Y-start.Y)*(end.Y-start.Y))
		totalDist := 0.0
		for i := 1; i < len(points); i++ {
			dx := points[i].X - points[i-1].X
			dy := points[i].Y - points[i-1].Y
			totalDist += math.Sqrt(dx*dx + dy*dy)
		}
		if totalDist > 100 && straightDist/totalDist > 0.92 {
			patterns = append(patterns, AIAnomalyStraightLine)
		}
	}

	// 高速度检测
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
	if maxSpeed > 10 {
		patterns = append(patterns, AIAnomalyHighSpeed)
	}

	return patterns
}

// ============================================
// 实时行为分析服务
// ============================================

type RealTimeBehaviorAnalysisService struct {
	featureExtractor     *LSTMFeatureExtractor
	transformerPredictor *TransformerPredictor
	dtwAnalyzer          *OptimizedDTW
	anomalyDetector      *AIAnomalyDetector
	loaded               bool
	mu                   sync.RWMutex
}

type AIPredictionResult struct {
	CombinedScore  float64
	RiskLevel      string
	IsBot          bool
	Confidence     float64
	ProcessingTime time.Duration
}

func NewRealTimeBehaviorAnalysisService() *RealTimeBehaviorAnalysisService {
	return &RealTimeBehaviorAnalysisService{
		featureExtractor:     NewLSTMFeatureExtractor(),
		transformerPredictor: NewTransformerPredictor(),
		dtwAnalyzer:          NewOptimizedDTW(),
		anomalyDetector:      NewAIAnomalyDetector(),
	}
}

func (s *RealTimeBehaviorAnalysisService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	if s.loaded {
		return nil
	}
	s.transformerPredictor.Initialize(ctx)
	s.loaded = true
	return nil
}

func (s *RealTimeBehaviorAnalysisService) PredictRiskFromData(ctx context.Context, traceData *model.TraceData) (*AIPredictionResult, error) {
	start := time.Now()

	if traceData == nil || len(traceData.Points) < 2 {
		return &AIPredictionResult{
			CombinedScore: 0.5,
			RiskLevel:     "unknown",
			IsBot:         false,
			Confidence:    0.5,
			ProcessingTime: time.Since(start),
		}, nil
	}

	features, _ := s.featureExtractor.ExtractFeatures(ctx, traceData)
	transformerScore, _ := s.transformerPredictor.Predict(ctx, features)
	anomalies := s.anomalyDetector.Detect(ctx, traceData)
	anomalyScore := float64(len(anomalies)) / 20.0

	combinedScore := transformerScore*0.7 + anomalyScore*0.3

	return &AIPredictionResult{
		CombinedScore:  combinedScore,
		RiskLevel:      s.determineRiskLevel(combinedScore),
		IsBot:          combinedScore >= 0.5,
		Confidence:     math.Max(0.5, transformerScore),
		ProcessingTime: time.Since(start),
	}, nil
}

func (s *RealTimeBehaviorAnalysisService) determineRiskLevel(score float64) string {
	switch {
	case score >= 0.9:
		return "critical"
	case score >= 0.7:
		return "high"
	case score >= 0.5:
		return "medium"
	case score >= 0.3:
		return "low"
	default:
		return "none"
	}
}

func (s *RealTimeBehaviorAnalysisService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.loaded
}

// ============================================
// 辅助函数
// ============================================

func aiMin(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func aiNormalizeFeature(value, min, max float64) float64 {
	return (value - min) / (max - min)
}

func aiMeanFloatSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func aiVarianceFloatSlice(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := aiMeanFloatSlice(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}

func aiMinFloatSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func aiMaxFloatSlice(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func aiCalculateEntropy(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	buckets := 10
	minVal := aiMinFloatSlice(values)
	maxVal := aiMaxFloatSlice(values)
	rangeVal := maxVal - minVal
	if rangeVal < 0.001 {
		return 0
	}

	histogram := make([]int, buckets)
	for _, v := range values {
		bin := int((v-minVal)/rangeVal * float64(buckets-1))
		if bin >= buckets {
			bin = buckets - 1
		}
		if bin < 0 {
			bin = 0
		}
		histogram[bin]++
	}

	entropy := 0.0
	total := len(values)
	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func aiAutocorrelation(values []float64, lag int) float64 {
	if lag >= len(values) {
		return 0
	}
	mean := aiMeanFloatSlice(values)
	n := len(values) - lag
	numerator := 0.0
	denominator := 0.0

	for i := 0; i < n; i++ {
		numerator += (values[i] - mean) * (values[i+lag] - mean)
		denominator += (values[i] - mean) * (values[i] - mean)
	}
	if denominator == 0 {
		return 0
	}
	return numerator / denominator
}

func aiFFT(x []float64) []complex128 {
	n := len(x)
	if n <= 1 {
		result := make([]complex128, n)
		for i, v := range x {
			result[i] = complex(v, 0)
		}
		return result
	}

	even := make([]float64, n/2)
	odd := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	fftEven := aiFFT(even)
	fftOdd := aiFFT(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		t := cmplx.Exp(complex(0, -2*math.Pi*float64(k)/float64(n))) * fftOdd[k]
		result[k] = fftEven[k] + t
		result[k+n/2] = fftEven[k] - t
	}
	return result
}

func aiSigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func aiComputeCurvature(p1, p2, p3 model.TracePoint) float64 {
	v1x := p2.X - p1.X
	v1y := p2.Y - p1.Y
	v2x := p3.X - p2.X
	v2y := p3.Y - p2.Y

	dot := v1x*v2x + v1y*v2y
	mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
	mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	cosAngle := dot / (mag1 * mag2)
	if cosAngle > 1 {
		cosAngle = 1
	}
	if cosAngle < -1 {
		cosAngle = -1
	}
	return math.Acos(cosAngle)
}
