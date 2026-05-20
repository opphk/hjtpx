package service

import (
	"context"
	"crypto/sha256"
	"encoding/binary"
	"fmt"
	"math"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type OptimizedLSTMService struct {
	mu                sync.RWMutex
	initialized       atomic.Bool
	featureCache      *OptimizedModelCache
	weightCache       *OptimizedWeightCache
	config            *OptimizedConfig
	metrics           *OptimizedMetrics
	lstmLayer         *LSTMLayer
	attentionLayer    *AttentionLayer
	classifier        *ClassifierLayer
}

type OptimizedConfig struct {
	FeatureDim           int
	HiddenDim           int
	NumLayers           int
	DropoutRate         float64
	EnableCache         bool
	CacheSize           int
	MaxInferenceTimeMs  int
	AccuracyTarget      float64
	BatchSize           int
	UsePooling          bool
	PoolSize            int
}

type OptimizedModelCache struct {
	mu       sync.RWMutex
	cache    map[string]*OptimizedCacheEntry
	maxSize  int
	hits     atomic.Int64
	misses   atomic.Int64
	evictions atomic.Int64
}

type OptimizedCacheEntry struct {
	Features   []float64
	Prediction *OptimizedPredictionResult
	Timestamp  time.Time
	TTL        time.Duration
	Hash       string
}

type OptimizedWeightCache struct {
	mu       sync.RWMutex
	cache    map[string][]float64
	hits     atomic.Int64
	misses   atomic.Int64
}

type OptimizedMetrics struct {
	TotalRequests      atomic.Int64
	CacheHits         atomic.Int64
	CacheMisses       atomic.Int64
	AvgLatencyMs      atomic.Uint64
	MaxLatencyMs      atomic.Int64
	MinLatencyMs      atomic.Int64
	P95LatencyMs      atomic.Int64
	P99LatencyMs      atomic.Int64
	TotalLatencySum   atomic.Int64
	TotalLatencyCount atomic.Int64
	AccuracySum       atomic.Uint64
	AccuracyCount     atomic.Int64
}

type LSTMLayer struct {
	mu           sync.RWMutex
	inputWeights [][]float64
	hiddenWeights [][]float64
	inputBias    []float64
	hiddenBias   []float64
	forgetWeights [][]float64
	outputWeights [][]float64
	cellWeights  [][]float64
	hiddenDim    int
	inputDim     int
	dropoutRate  float64
}

type AttentionLayer struct {
	mu           sync.RWMutex
	queryWeights []float64
	keyWeights   []float64
	valueWeights []float64
	scale        float64
	numHeads     int
}

type ClassifierLayer struct {
	mu           sync.RWMutex
	weights      [][]float64
	bias         []float64
	outputDim    int
	inputDim     int
}

type OptimizedPredictionResult struct {
	Score        float64
	IsBot        bool
	Confidence   float64
	RiskLevel    string
	Features     map[string]float64
	LatencyMs    float64
	CacheHit     bool
}

type TrajectoryFeatures struct {
	Basic      []float64
	Speed      []float64
	Direction  []float64
	Curvature  []float64
	Position   []float64
	Behavior   []float64
	Advanced   []float64
}

func NewOptimizedLSTMService() *OptimizedLSTMService {
	return &OptimizedLSTMService{
		featureCache: NewOptimizedModelCache(10000),
		weightCache:  NewOptimizedWeightCache(),
		config: &OptimizedConfig{
			FeatureDim:          768,
			HiddenDim:           256,
			NumLayers:           2,
			DropoutRate:         0.3,
			EnableCache:         true,
			CacheSize:           10000,
			MaxInferenceTimeMs:  20,
			AccuracyTarget:      0.95,
			BatchSize:           32,
			UsePooling:          true,
			PoolSize:            4,
		},
		metrics: &OptimizedMetrics{},
	}
}

func NewOptimizedModelCache(maxSize int) *OptimizedModelCache {
	return &OptimizedModelCache{
		cache:   make(map[string]*OptimizedCacheEntry, maxSize),
		maxSize: maxSize,
	}
}

func NewOptimizedWeightCache() *OptimizedWeightCache {
	return &OptimizedWeightCache{
		cache: make(map[string][]float64),
	}
}

func (s *OptimizedLSTMService) Initialize(ctx context.Context) error {
	if s.initialized.Load() {
		return nil
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.lstmLayer = s.initializeLSTMLayer()
	s.attentionLayer = s.initializeAttentionLayer()
	s.classifier = s.initializeClassifier()

	s.initialized.Store(true)
	return nil
}

func (s *OptimizedLSTMService) initializeLSTMLayer() *LSTMLayer {
	inputDim := s.config.FeatureDim
	hiddenDim := s.config.HiddenDim

	return &LSTMLayer{
		inputWeights:  s.initWeights(inputDim, hiddenDim*4),
		hiddenWeights: s.initWeights(hiddenDim, hiddenDim*4),
		forgetWeights: s.initWeights(hiddenDim, hiddenDim*4),
		outputWeights: s.initWeights(hiddenDim, hiddenDim),
		cellWeights:   s.initWeights(hiddenDim, hiddenDim),
		inputBias:     s.initBias(hiddenDim * 4),
		hiddenBias:    s.initBias(hiddenDim * 4),
		hiddenDim:     hiddenDim,
		inputDim:      inputDim,
		dropoutRate:   s.config.DropoutRate,
	}
}

func (s *OptimizedLSTMService) initializeAttentionLayer() *AttentionLayer {
	return &AttentionLayer{
		queryWeights: s.initWeightsSingle(s.config.HiddenDim, s.config.HiddenDim),
		keyWeights:   s.initWeightsSingle(s.config.HiddenDim, s.config.HiddenDim),
		valueWeights: s.initWeightsSingle(s.config.HiddenDim, s.config.HiddenDim),
		scale:        1.0 / math.Sqrt(float64(s.config.HiddenDim)),
		numHeads:     8,
	}
}

func (s *OptimizedLSTMService) initializeClassifier() *ClassifierLayer {
	return &ClassifierLayer{
		weights:   s.initWeights(s.config.HiddenDim, 64),
		bias:     s.initBias(64),
		outputDim: 1,
		inputDim:  s.config.HiddenDim,
	}
}

func (s *OptimizedLSTMService) initWeights(inputDim, outputDim int) [][]float64 {
	weights := make([][]float64, inputDim)
	scale := math.Sqrt(2.0 / float64(inputDim+outputDim))
	for i := range weights {
		weights[i] = make([]float64, outputDim)
		for j := range weights[i] {
			weights[i][j] = (mathRand() - 0.5) * 2 * scale
		}
	}
	return weights
}

func (s *OptimizedLSTMService) initWeightsSingle(dim1, dim2 int) []float64 {
	weights := make([]float64, dim1*dim2)
	scale := math.Sqrt(2.0 / float64(dim1+dim2))
	for i := range weights {
		weights[i] = (mathRand() - 0.5) * 2 * scale
	}
	return weights
}

func (s *OptimizedLSTMService) initBias(size int) []float64 {
	bias := make([]float64, size)
	for i := range bias {
		bias[i] = 0.0
	}
	return bias
}

func mathRand() float64 {
	return float64(binary.LittleEndian.Uint64(generateRandomBytes())) / (1 << 64)
}

func generateRandomBytes() []byte {
	b := make([]byte, 8)
	sh := sha256.New()
	sh.Write([]byte(fmt.Sprintf("%d%d", time.Now().UnixNano(), time.Now().Unix())))
	copy(b, sh.Sum(nil))
	return b
}

func (s *OptimizedLSTMService) Predict(ctx context.Context, traceData *model.TraceData) (*OptimizedPredictionResult, error) {
	startTime := time.Now()
	s.metrics.TotalRequests.Add(1)

	points := s.convertToNNPoints(traceData)
	cacheKey := s.generateCacheKey(points)

	if s.config.EnableCache {
		if cachedResult := s.featureCache.Get(cacheKey); cachedResult != nil {
			s.metrics.CacheHits.Add(1)
			result := *cachedResult.Prediction
			result.LatencyMs = float64(time.Since(startTime).Milliseconds())
			result.CacheHit = true
			return &result, nil
		}
	}

	s.metrics.CacheMisses.Add(1)

	features := s.extractFeaturesOptimized(points)

	features = s.applyFeatureNormalization(features)

	lstmOutput := s.forwardLSTM(features)

	attentionOutput := s.applyAttention(lstmOutput)

	prediction := s.classify(attentionOutput)

	featuresMap := s.extractFeatureMap(points, features)

	result := &OptimizedPredictionResult{
		Score:      prediction,
		IsBot:      prediction > 0.5,
		Confidence: s.calculateConfidence(prediction),
		RiskLevel:  s.determineRiskLevel(prediction),
		Features:   featuresMap,
		LatencyMs:  float64(time.Since(startTime).Milliseconds()),
		CacheHit:   false,
	}

	if s.config.EnableCache {
		s.featureCache.Set(cacheKey, features, result)
	}

	s.updatePerformanceMetrics(result.LatencyMs, result.Confidence)

	return result, nil
}

func (s *OptimizedLSTMService) convertToNNPoints(traceData *model.TraceData) []TrajectoryNNPoint {
	points := make([]TrajectoryNNPoint, len(traceData.Points))
	for i, p := range traceData.Points {
		points[i] = TrajectoryNNPoint{
			X:         p.X,
			Y:         p.Y,
			Timestamp: p.Timestamp,
		}
	}
	return points
}

func (s *OptimizedLSTMService) generateCacheKey(points []TrajectoryNNPoint) string {
	if len(points) == 0 {
		return "empty"
	}

	hash := sha256.New()
	
	binary.Write(hash, binary.LittleEndian, int64(len(points)))
	
	for i := 0; i < len(points) && i < 100; i++ {
		binary.Write(hash, binary.LittleEndian, points[i].X)
		binary.Write(hash, binary.LittleEndian, points[i].Y)
		binary.Write(hash, binary.LittleEndian, points[i].Timestamp)
	}
	
	return fmt.Sprintf("%x", hash.Sum(nil))[:32]
}

func (s *OptimizedLSTMService) extractFeaturesOptimized(points []TrajectoryNNPoint) []float64 {
	if len(points) < 2 {
		return make([]float64, s.config.FeatureDim)
	}

	features := TrajectoryFeatures{
		Basic:     make([]float64, 64),
		Speed:     make([]float64, 128),
		Direction: make([]float64, 128),
		Curvature: make([]float64, 128),
		Position:  make([]float64, 128),
		Behavior:  make([]float64, 192),
		Advanced:  make([]float64, 128),
	}

	s.extractBasicFeaturesFast(points, features.Basic)
	s.extractSpeedFeaturesFast(points, features.Speed)
	s.extractDirectionFeaturesFast(points, features.Direction)
	s.extractCurvatureFeaturesFast(points, features.Curvature)
	s.extractPositionFeaturesFast(points, features.Position)
	s.extractBehaviorFeaturesFast(points, features.Behavior)
	s.extractAdvancedFeaturesFast(points, features.Advanced)

	return s.flattenAndNormalizeFeatures(features)
}

func (s *OptimizedLSTMService) extractBasicFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) == 0 {
		return
	}

	totalDist := s.calculateTotalDistanceFast(points)
	features[0] = math.Min(totalDist/1000.0, 1.0)
	features[1] = math.Min(float64(len(points))/1000.0, 1.0)
	features[2] = points[0].X / 1000.0
	features[3] = points[0].Y / 1000.0
	features[4] = points[len(points)-1].X / 1000.0
	features[5] = points[len(points)-1].Y / 1000.0

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) > 0 {
		features[6] = s.fastMean(speeds)
		features[7] = s.fastMax(speeds)
		features[8] = s.fastMin(speeds)
		features[9] = s.fastVariance(speeds, features[6])
	}
}

func (s *OptimizedLSTMService) calculateTotalDistanceFast(points []TrajectoryNNPoint) float64 {
	total := 0.0
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total
}

func (s *OptimizedLSTMService) calculateSpeedsFast(points []TrajectoryNNPoint) []float64 {
	speeds := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, math.Sqrt(dx*dx+dy*dy)/dt)
		}
	}
	return speeds
}

func (s *OptimizedLSTMService) fastMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (s *OptimizedLSTMService) fastMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (s *OptimizedLSTMService) fastMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (s *OptimizedLSTMService) fastVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (s *OptimizedLSTMService) extractSpeedFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) < 2 {
		return
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) == 0 {
		return
	}

	numSegments := 16
	segmentSize := len(speeds) / numSegments
	if segmentSize == 0 {
		segmentSize = 1
	}

	for i := 0; i < numSegments; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if end > len(speeds) {
			end = len(speeds)
		}
		if start >= len(speeds) {
			break
		}

		segment := speeds[start:end]
		idx := i * 8
		if idx+7 >= len(features) {
			break
		}
		features[idx] = s.fastMean(segment)
		features[idx+1] = s.fastMax(segment)
		features[idx+2] = s.fastMin(segment)
		features[idx+3] = s.fastVariance(segment, features[idx])
		features[idx+4] = s.fastQuantile(segment, 0.25)
		features[idx+5] = s.fastQuantile(segment, 0.5)
		features[idx+6] = s.fastQuantile(segment, 0.75)
		features[idx+7] = s.fastQuantile(segment, 0.9)
	}
}

func (s *OptimizedLSTMService) fastQuantile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	if len(values) == 1 {
		return values[0]
	}

	sorted := make([]float64, len(values))
	copy(sorted, values)
	
	n := len(sorted)
	idx := int(float64(n-1) * quantile)
	if idx >= n {
		idx = n - 1
	}

	for i := 0; i < idx; i++ {
		minIdx := i
		for j := i + 1; j < n; j++ {
			if sorted[j] < sorted[minIdx] {
				minIdx = j
			}
		}
		if minIdx != i {
			sorted[i], sorted[minIdx] = sorted[minIdx], sorted[i]
		}
	}

	return sorted[idx]
}

func (s *OptimizedLSTMService) extractDirectionFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) < 2 {
		return
	}

	directions := make([]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		directions[i-1] = math.Atan2(dy, dx)
	}

	bins := 36
	histogram := s.fastHistogram(directions, bins, -math.Pi, math.Pi)
	copy(features[:bins], histogram)

	directionChanges := s.calculateDirectionChangesFast(directions)
	copy(features[bins:bins+32], directionChanges)
}

func (s *OptimizedLSTMService) fastHistogram(values []float64, bins int, minVal, maxVal float64) []float64 {
	histogram := make([]float64, bins)
	if len(values) == 0 {
		return histogram
	}

	binWidth := (maxVal - minVal) / float64(bins)
	for _, v := range values {
		normalized := (v - minVal) / binWidth
		idx := int(normalized)
		if idx < 0 {
			idx = 0
		}
		if idx >= bins {
			idx = bins - 1
		}
		histogram[idx]++
	}

	total := float64(len(values))
	for i := range histogram {
		histogram[i] /= total
	}
	return histogram
}

func (s *OptimizedLSTMService) calculateDirectionChangesFast(directions []float64) []float64 {
	changes := make([]float64, 32)
	if len(directions) < 2 {
		return changes
	}

	changeValues := make([]float64, len(directions)-1)
	for i := 1; i < len(directions); i++ {
		change := directions[i] - directions[i-1]
		if change > math.Pi {
			change -= 2 * math.Pi
		} else if change < -math.Pi {
			change += 2 * math.Pi
		}
		changeValues[i-1] = math.Abs(change)
	}

	if len(changeValues) > 0 {
		changes[0] = s.fastMean(changeValues)
		changes[1] = s.fastMax(changeValues)
		changes[2] = s.fastMin(changeValues)
		changes[3] = s.fastVariance(changeValues, changes[0])
	}

	return changes
}

func (s *OptimizedLSTMService) extractCurvatureFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) < 3 {
		return
	}

	curvatures := s.calculateCurvaturesFast(points)
	if len(curvatures) == 0 {
		return
	}

	features[0] = s.fastMean(curvatures)
	features[1] = s.fastMax(curvatures)
	features[2] = s.fastMin(curvatures)
	features[3] = s.fastVariance(curvatures, features[0])
	features[4] = s.fastQuantile(curvatures, 0.1)
	features[5] = s.fastQuantile(curvatures, 0.25)
	features[6] = s.fastQuantile(curvatures, 0.5)
	features[7] = s.fastQuantile(curvatures, 0.75)
	features[8] = s.fastQuantile(curvatures, 0.9)
	features[9] = s.fastQuantile(curvatures, 0.99)

	maxCurv := features[1]
	if maxCurv < 0.001 {
		maxCurv = 0.001
	}
	
	for i := 0; i < 100 && i < len(curvatures); i++ {
		features[10+i] = curvatures[i] / maxCurv
	}
}

func (s *OptimizedLSTMService) calculateCurvaturesFast(points []TrajectoryNNPoint) []float64 {
	if len(points) < 3 {
		return []float64{}
	}

	curvatures := make([]float64, len(points)-2)
	for i := 1; i < len(points)-1; i++ {
		dx1 := points[i].X - points[i-1].X
		dy1 := points[i].Y - points[i-1].Y
		dx2 := points[i+1].X - points[i].X
		dy2 := points[i+1].Y - points[i].Y

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			} else if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			curvatures[i-1] = angle / (mag1 + mag2 + 0.001)
		}
	}
	return curvatures
}

func (s *OptimizedLSTMService) extractPositionFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) == 0 {
		return
	}

	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y

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

	features[0] = minX / 1000.0
	features[1] = maxX / 1000.0
	features[2] = minY / 1000.0
	features[3] = maxY / 1000.0
	features[4] = (maxX - minX) / 1000.0
	features[5] = (maxY - minY) / 1000.0
	features[6] = features[4] / (features[5] + 0.001)

	xHistogram := s.fastPositionHistogram(points, true, 32)
	yHistogram := s.fastPositionHistogram(points, false, 32)
	copy(features[7:39], xHistogram)
	copy(features[39:71], yHistogram)

	coverage := s.calculateCoverageFast(points, 19)
	copy(features[71:128], coverage)
}

func (s *OptimizedLSTMService) fastPositionHistogram(points []TrajectoryNNPoint, isX bool, bins int) []float64 {
	histogram := make([]float64, bins)
	if len(points) == 0 {
		return histogram
	}

	minVal := points[0].X
	maxVal := points[0].X
	if !isX {
		minVal = points[0].Y
		maxVal = points[0].Y
	}

	for _, p := range points {
		val := p.X
		if !isX {
			val = p.Y
		}
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}

	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}

	for _, p := range points {
		val := p.X
		if !isX {
			val = p.Y
		}
		normalized := (val - minVal) / rangeVal
		binIdx := int(normalized * float64(bins))
		if binIdx >= bins {
			binIdx = bins - 1
		}
		if binIdx < 0 {
			binIdx = 0
		}
		histogram[binIdx]++
	}

	total := float64(len(points))
	for i := range histogram {
		histogram[i] /= total
	}
	return histogram
}

func (s *OptimizedLSTMService) calculateCoverageFast(points []TrajectoryNNPoint, gridSize int) []float64 {
	coverage := make([]float64, gridSize*gridSize)
	if len(points) == 0 {
		return coverage
	}

	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y

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

	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	for _, p := range points {
		normX := (p.X - minX) / rangeX
		normY := (p.Y - minY) / rangeY

		gridX := int(normX * float64(gridSize-1))
		gridY := int(normY * float64(gridSize-1))

		if gridX >= gridSize {
			gridX = gridSize - 1
		}
		if gridY >= gridSize {
			gridY = gridSize - 1
		}
		if gridX < 0 {
			gridX = 0
		}
		if gridY < 0 {
			gridY = 0
		}

		coverage[gridY*gridSize+gridX] = 1.0
	}

	return coverage
}

func (s *OptimizedLSTMService) extractBehaviorFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) < 2 {
		return
	}

	pattern := s.classifyBehaviorPatternFast(points)
	copy(features[:32], pattern)

	anomalies := s.detectAnomaliesFast(points)
	copy(features[32:64], anomalies)

	learningFeatures := s.extractLearningFeaturesFast(points)
	copy(features[64:160], learningFeatures)
}

func (s *OptimizedLSTMService) classifyBehaviorPatternFast(points []TrajectoryNNPoint) []float64 {
	pattern := make([]float64, 32)
	if len(points) < 2 {
		return pattern
	}

	totalDist := s.calculateTotalDistanceFast(points)
	speeds := s.calculateSpeedsFast(points)
	directions := s.calculateDirectionsFast(points)

	avgSpeed := s.fastMean(speeds)
	speedVar := s.fastVariance(speeds, avgSpeed)

	pattern[0] = math.Min(totalDist/100.0, 1.0)
	pattern[1] = math.Min(avgSpeed/10.0, 1.0)
	pattern[2] = math.Min(speedVar, 1.0)

	maxSpeed := s.fastMax(speeds)
	if len(speeds) > 0 && avgSpeed > 0.001 {
		speedRatio := maxSpeed / avgSpeed
		pattern[3] = math.Min(speedRatio/10.0, 1.0)
	}

	if len(directions) > 0 {
		directionEntropy := s.calculateDirectionEntropyFast(directions)
		pattern[4] = directionEntropy
	}

	if len(points) >= 3 {
		curvatures := s.calculateCurvaturesFast(points)
		if len(curvatures) > 0 {
			avgCurvature := s.fastMean(curvatures)
			pattern[5] = math.Min(avgCurvature*10.0, 1.0)
		}
	}

	dx := points[len(points)-1].X - points[0].X
	dy := points[len(points)-1].Y - points[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)
	if totalDist > 0 {
		pattern[6] = directDist / totalDist
	}

	return pattern
}

func (s *OptimizedLSTMService) calculateDirectionsFast(points []TrajectoryNNPoint) []float64 {
	if len(points) < 2 {
		return []float64{}
	}

	directions := make([]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		directions[i-1] = math.Atan2(dy, dx)
	}
	return directions
}

func (s *OptimizedLSTMService) calculateDirectionEntropyFast(directions []float64) float64 {
	if len(directions) == 0 {
		return 0
	}

	histogram := s.fastHistogram(directions, 8, 0, 2*math.Pi)
	entropy := 0.0
	for _, p := range histogram {
		if p > 0 {
			entropy -= p * math.Log(p+0.0001)
		}
	}
	return entropy / math.Log(8.0)
}

func (s *OptimizedLSTMService) detectAnomaliesFast(points []TrajectoryNNPoint) []float64 {
	anomalies := make([]float64, 32)
	if len(points) < 3 {
		return anomalies
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) > 0 {
		avgSpeed := s.fastMean(speeds)
		speedStdDev := math.Sqrt(s.fastVariance(speeds, avgSpeed))

		for i, speed := range speeds {
			if i < 32 {
				zScore := (speed - avgSpeed) / (speedStdDev + 0.001)
				if math.Abs(zScore) > 2.0 {
					anomalies[i] = 1.0
				}
			}
		}
	}

	if len(points) >= 3 {
		curvatures := s.calculateCurvaturesFast(points)
		if len(curvatures) > 0 {
			avgCurvature := s.fastMean(curvatures)
			curvatureStdDev := math.Sqrt(s.fastVariance(curvatures, avgCurvature))

			for i, curvature := range curvatures {
				if i < 32 {
					zScore := (curvature - avgCurvature) / (curvatureStdDev + 0.001)
					if math.Abs(zScore) > 2.5 {
						anomalies[16+i] = 1.0
					}
				}
			}
		}
	}

	return anomalies
}

func (s *OptimizedLSTMService) extractLearningFeaturesFast(points []TrajectoryNNPoint) []float64 {
	features := make([]float64, 96)
	if len(points) < 2 {
		return features
	}

	speeds := s.calculateSpeedsFast(points)
	directions := s.calculateDirectionsFast(points)

	if len(speeds) > 0 {
		avgSpeed := s.fastMean(speeds)
		features[0] = avgSpeed
		features[1] = s.fastMax(speeds)
		features[2] = s.fastMin(speeds)
		features[3] = s.fastVariance(speeds, avgSpeed)
		features[4] = s.fastQuantile(speeds, 0.25)
		features[5] = s.fastQuantile(speeds, 0.5)
		features[6] = s.fastQuantile(speeds, 0.75)
	}

	if len(directions) > 0 {
		features[7] = s.calculateDirectionEntropyFast(directions)
	}

	if len(points) >= 3 {
		curvatures := s.calculateCurvaturesFast(points)
		if len(curvatures) > 0 {
			features[8] = s.fastMean(curvatures)
			features[9] = s.fastMax(curvatures)
			features[10] = s.fastVariance(curvatures, features[8])
		}
	}

	progress := float64(len(points)) / 100.0
	features[11] = math.Min(progress, 1.0)

	totalDist := s.calculateTotalDistanceFast(points)
	features[12] = totalDist

	dx := points[len(points)-1].X - points[0].X
	dy := points[len(points)-1].Y - points[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)
	features[13] = directDist

	if totalDist > 0 {
		features[14] = directDist / totalDist
	}

	return features
}

func (s *OptimizedLSTMService) extractAdvancedFeaturesFast(points []TrajectoryNNPoint, features []float64) {
	if len(points) < 3 {
		return
	}

	jitterFeatures := s.extractJitterFeaturesFast(points)
	copy(features[:32], jitterFeatures)

	temporalFeatures := s.extractTemporalFeaturesFast(points)
	copy(features[32:64], temporalFeatures)

	accelFeatures := s.extractAccelerationFeaturesFast(points)
	copy(features[64:96], accelFeatures)

	fourierFeatures := s.extractFourierFeaturesFast(points)
	copy(features[96:128], fourierFeatures)
}

func (s *OptimizedLSTMService) extractJitterFeaturesFast(points []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(points) < 3 {
		return features
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 2 {
		return features
	}

	avgSpeed := s.fastMean(speeds)
	speedStdDev := math.Sqrt(s.fastVariance(speeds, avgSpeed))

	jitterCount := 0
	totalJitterAmplitude := 0.0

	for i := 1; i < len(speeds); i++ {
		change := math.Abs(speeds[i] - speeds[i-1])
		zScore := (change - avgSpeed) / (speedStdDev + 0.001)

		if zScore > 2.5 {
			jitterCount++
			totalJitterAmplitude += change
		}
	}

	features[0] = float64(jitterCount)
	features[1] = totalJitterAmplitude / math.Max(1.0, float64(jitterCount))
	features[2] = float64(jitterCount) / float64(len(speeds))

	curvatures := s.calculateCurvaturesFast(points)
	curvatureVariance := s.fastVariance(curvatures, s.fastMean(curvatures))
	features[5] = curvatureVariance

	directions := s.calculateDirectionsFast(points)
	directionChanges := s.calculateDirectionChangesFast(directions)
	features[6] = s.fastMean(directionChanges)
	features[7] = s.fastMax(directionChanges)

	humanLikelihood := s.calculateHumanLikelihoodFast(points)
	features[16] = humanLikelihood

	features[17] = s.detectMechanicalMovementFast(points)
	features[18] = s.detectPerfectStraightnessFast(points)
	features[19] = s.detectExcessiveSmoothnessFast(points)

	features[20] = s.calculatePerceivedEffortFast(points)
	features[21] = s.calculateMovementNaturalnessFast(points)
	features[22] = s.calculateTrajectoryComplexityFast(points)

	features[23] = s.detectCopiedPatternFast(points)
	features[24] = s.detectRepeatedPatternFast(points)

	features[25] = s.calculateInterPointConsistencyFast(points)
	features[26] = s.calculateSegmentRegularityFast(points)

	return features
}

func (s *OptimizedLSTMService) extractTemporalFeaturesFast(points []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(points) < 2 {
		return features
	}

	if len(points) > 0 {
		totalDuration := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
		features[0] = totalDuration / 1000.0
	}

	timeDiffs := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) > 0 {
		features[1] = s.fastMean(timeDiffs)
		features[2] = s.fastMax(timeDiffs)
		features[3] = s.fastMin(timeDiffs)
		features[4] = s.fastVariance(timeDiffs, features[1])
	}

	return features
}

func (s *OptimizedLSTMService) extractAccelerationFeaturesFast(points []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(points) < 3 {
		return features
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 2 {
		return features
	}

	accelerations := make([]float64, 0, len(speeds)-1)
	for i := 1; i < len(speeds); i++ {
		dt := float64(points[i+1].Timestamp - points[i].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) > 0 {
		features[0] = s.fastMean(accelerations)
		features[1] = s.fastMax(accelerations)
		features[2] = s.fastMin(accelerations)
		features[3] = s.fastVariance(accelerations, features[0])
		features[4] = s.fastQuantile(accelerations, 0.25)
		features[5] = s.fastQuantile(accelerations, 0.5)
		features[6] = s.fastQuantile(accelerations, 0.75)
	}

	features[16] = s.calculateAccelerationEntropyFast(points)
	features[17] = s.detectSuddenChangesFast(points)

	return features
}

func (s *OptimizedLSTMService) extractFourierFeaturesFast(points []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(points) < 8 {
		return features
	}

	xCoords := make([]float64, len(points))
	yCoords := make([]float64, len(points))
	for i, p := range points {
		xCoords[i] = p.X
		yCoords[i] = p.Y
	}

	xSpectrum := s.simpleFFTFast(xCoords)
	ySpectrum := s.simpleFFTFast(yCoords)

	if len(xSpectrum) > 0 {
		features[0] = s.fastMean(xSpectrum[:len(xSpectrum)/2])
		features[1] = s.fastMax(xSpectrum[:len(xSpectrum)/2])
		features[2] = s.fastVariance(xSpectrum[:len(xSpectrum)/2], features[0])
	}

	if len(ySpectrum) > 0 {
		features[3] = s.fastMean(ySpectrum[:len(ySpectrum)/2])
		features[4] = s.fastMax(ySpectrum[:len(ySpectrum)/2])
		features[5] = s.fastVariance(ySpectrum[:len(ySpectrum)/2], features[3])
	}

	features[6] = s.findDominantFrequencyFast(xSpectrum)
	features[7] = s.findDominantFrequencyFast(ySpectrum)

	features[8] = s.calculateSpectralEntropyFast(xSpectrum)

	return features
}

func (s *OptimizedLSTMService) simpleFFTFast(signal []float64) []float64 {
	n := len(signal)
	if n == 0 {
		return []float64{}
	}

	pow2 := 1
	for pow2 < n {
		pow2 *= 2
	}

	magnitude := make([]float64, pow2)
	scale := 1.0 / float64(pow2)

	for k := 0; k < pow2/2; k++ {
		realSum := 0.0
		imagSum := 0.0
		angleBase := 2.0 * math.Pi * float64(k) * scale

		for n := 0; n < pow2; n++ {
			angle := angleBase * float64(n)
			idx := n
			if idx >= len(signal) {
				idx = len(signal) - 1
			}
			realSum += signal[idx] * math.Cos(angle)
			imagSum += signal[idx] * math.Sin(angle)
		}

		magnitude[k] = math.Sqrt(realSum*realSum + imagSum*imagSum) * scale
		magnitude[pow2-k-1] = magnitude[k]
	}

	return magnitude
}

func (s *OptimizedLSTMService) findDominantFrequencyFast(spectrum []float64) float64 {
	if len(spectrum) < 2 {
		return 0.0
	}

	maxMag := 0.0
	dominantIdx := 0
	for i := 1; i < len(spectrum)/2; i++ {
		if spectrum[i] > maxMag {
			maxMag = spectrum[i]
			dominantIdx = i
		}
	}

	return float64(dominantIdx)
}

func (s *OptimizedLSTMService) calculateSpectralEntropyFast(spectrum []float64) float64 {
	if len(spectrum) == 0 {
		return 0.0
	}

	total := 0.0
	for _, mag := range spectrum {
		total += mag * mag
	}

	if total == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, mag := range spectrum {
		p := (mag * mag) / total
		if p > 0 {
			entropy -= p * math.Log(p+0.0001)
		}
	}

	return entropy / math.Log(float64(len(spectrum)))
}

func (s *OptimizedLSTMService) calculateAccelerationEntropyFast(points []TrajectoryNNPoint) float64 {
	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 3 {
		return 0.0
	}

	accelerations := make([]float64, 0, len(speeds)-1)
	for i := 1; i < len(speeds); i++ {
		dt := float64(points[i+1].Timestamp - points[i].Timestamp)
		if dt > 0 {
			accel := math.Abs(speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) == 0 {
		return 0.0
	}

	bins := 10
	histogram := make([]float64, bins)
	maxAccel := s.fastMax(accelerations)
	if maxAccel == 0 {
		maxAccel = 1
	}

	for _, accel := range accelerations {
		binIdx := int(accel / maxAccel * float64(bins))
		if binIdx >= bins {
			binIdx = bins - 1
		}
		histogram[binIdx]++
	}

	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			p := count / float64(len(accelerations))
			entropy -= p * math.Log(p+0.0001)
		}
	}

	return entropy / math.Log(float64(bins))
}

func (s *OptimizedLSTMService) detectSuddenChangesFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 2 {
		return 0.0
	}

	stdDev := math.Sqrt(s.fastVariance(speeds, s.fastMean(speeds)))

	suddenChangeCount := 0
	for i := 1; i < len(speeds); i++ {
		speedChange := math.Abs(speeds[i] - speeds[i-1])
		zScore := speedChange / (stdDev + 0.001)
		if zScore > 3.0 {
			suddenChangeCount++
		}
	}

	return float64(suddenChangeCount) / float64(len(speeds))
}

func (s *OptimizedLSTMService) calculateHumanLikelihoodFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) == 0 {
		return 0.0
	}

	avgSpeed := s.fastMean(speeds)
	speedVariance := s.fastVariance(speeds, avgSpeed)

	humanSpeedRange := avgSpeed >= 0.5 && avgSpeed <= 5.0
	naturalVariance := speedVariance > 0.1 && speedVariance < 2.0

	curvatures := s.calculateCurvaturesFast(points)
	avgCurvature := 0.0
	if len(curvatures) > 0 {
		avgCurvature = s.fastMean(curvatures)
	}
	naturalCurvature := avgCurvature > 0.01 && avgCurvature < 0.5

	directions := s.calculateDirectionsFast(points)
	entropy := s.calculateDirectionEntropyFast(directions)
	naturalEntropy := entropy > 0.3 && entropy < 0.9

	score := 0.0
	if humanSpeedRange {
		score += 0.3
	}
	if naturalVariance {
		score += 0.3
	}
	if naturalCurvature {
		score += 0.2
	}
	if naturalEntropy {
		score += 0.2
	}

	return score
}

func (s *OptimizedLSTMService) detectMechanicalMovementFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 4 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 2 {
		return 0.0
	}

	avgSpeed := s.fastMean(speeds)
	speedVariance := s.fastVariance(speeds, avgSpeed)

	if avgSpeed == 0 {
		return 0.0
	}
	coefficientOfVariation := math.Sqrt(speedVariance) / avgSpeed

	if coefficientOfVariation < 0.05 {
		return 1.0
	}

	timeDiffs := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) > 1 {
		uniformTiming := s.detectUniformTimingFast(timeDiffs)
		if uniformTiming > 0.9 {
			return 1.0
		}
	}

	return math.Max(0, 1.0-coefficientOfVariation*5)
}

func (s *OptimizedLSTMService) detectUniformTimingFast(timeDiffs []float64) float64 {
	if len(timeDiffs) < 2 {
		return 0.0
	}

	avg := s.fastMean(timeDiffs)
	variance := s.fastVariance(timeDiffs, avg)
	coefficientOfVariation := math.Sqrt(variance) / (avg + 0.001)

	if coefficientOfVariation < 0.05 {
		return 1.0
	}
	return 0.0
}

func (s *OptimizedLSTMService) detectPerfectStraightnessFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	totalDist := s.calculateTotalDistanceFast(points)
	if totalDist == 0 {
		return 0.0
	}

	dx := points[len(points)-1].X - points[0].X
	dy := points[len(points)-1].Y - points[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)

	straightnessRatio := directDist / totalDist

	if straightnessRatio > 0.999 {
		return 1.0
	}

	return straightnessRatio
}

func (s *OptimizedLSTMService) detectExcessiveSmoothnessFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 4 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 2 {
		return 0.0
	}

	curvatures := s.calculateCurvaturesFast(points)
	if len(curvatures) == 0 {
		return 0.0
	}

	maxCurvature := s.fastMax(curvatures)
	avgCurvature := s.fastMean(curvatures)

	smoothnessRatio := avgCurvature / (maxCurvature + 0.001)

	avgSpeed := s.fastMean(speeds)
	speedVariance := s.fastVariance(speeds, avgSpeed)

	if avgSpeed > 0 {
		cv := math.Sqrt(speedVariance) / avgSpeed
		if cv < 0.1 && smoothnessRatio > 0.9 {
			return 1.0
		}
	}

	return smoothnessRatio
}

func (s *OptimizedLSTMService) calculatePerceivedEffortFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	totalDist := s.calculateTotalDistanceFast(points)

	totalDuration := 0.0
	if len(points) > 1 {
		totalDuration = float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	}

	if totalDuration == 0 {
		return 0.0
	}

	avgSpeed := totalDist / totalDuration

	curvatures := s.calculateCurvaturesFast(points)
	avgCurvature := 0.0
	if len(curvatures) > 0 {
		avgCurvature = s.fastMean(curvatures)
	}

	directions := s.calculateDirectionsFast(points)
	directionEntropy := s.calculateDirectionEntropyFast(directions)

	effort := avgSpeed * 0.3
	effort += avgCurvature * 10.0 * 0.3
	effort += directionEntropy * 0.4

	return math.Min(1.0, effort)
}

func (s *OptimizedLSTMService) calculateMovementNaturalnessFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) == 0 {
		return 0.0
	}

	speedEntropy := 0.0
	speedHistogram := s.fastHistogram(speeds, 10, s.fastMin(speeds), s.fastMax(speeds)+0.001)
	for _, p := range speedHistogram {
		if p > 0 {
			speedEntropy -= p * math.Log(p+0.0001)
		}
	}
	speedEntropy /= math.Log(10.0)

	curvatures := s.calculateCurvaturesFast(points)
	curvatureEntropy := 0.0
	if len(curvatures) > 0 {
		curvatureHistogram := s.fastHistogram(curvatures, 8, s.fastMin(curvatures), s.fastMax(curvatures)+0.001)
		for _, p := range curvatureHistogram {
			if p > 0 {
				curvatureEntropy -= p * math.Log(p+0.0001)
			}
		}
		curvatureEntropy /= math.Log(8.0)
	}

	timeDiffs := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	timingEntropy := 0.0
	if len(timeDiffs) > 0 {
		timingHistogram := s.fastHistogram(timeDiffs, 8, s.fastMin(timeDiffs), s.fastMax(timeDiffs)+0.001)
		for _, p := range timingHistogram {
			if p > 0 {
				timingEntropy -= p * math.Log(p+0.0001)
			}
		}
		timingEntropy /= math.Log(8.0)
	}

	naturalness := speedEntropy*0.4 + curvatureEntropy*0.3 + timingEntropy*0.3

	return naturalness
}

func (s *OptimizedLSTMService) calculateTrajectoryComplexityFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 2 {
		return 0.0
	}

	totalDist := s.calculateTotalDistanceFast(points)

	dx := points[len(points)-1].X - points[0].X
	dy := points[len(points)-1].Y - points[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)

	if directDist == 0 {
		return 0.0
	}

	complexity := totalDist / directDist

	directions := s.calculateDirectionsFast(points)
	curvatures := s.calculateCurvaturesFast(points)

	directionChanges := 0
	for i := 1; i < len(directions); i++ {
		change := math.Abs(directions[i] - directions[i-1])
		if change > 0.1 {
			directionChanges++
		}
	}

	complexity += float64(directionChanges) * 0.5

	avgCurvature := 0.0
	if len(curvatures) > 0 {
		avgCurvature = s.fastMean(curvatures)
	}
	complexity += avgCurvature * float64(len(points))

	return math.Min(10.0, complexity)
}

func (s *OptimizedLSTMService) detectCopiedPatternFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 10 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 4 {
		return 0.0
	}

	windowSize := len(speeds) / 4
	if windowSize < 2 {
		windowSize = 2
	}

	maxCorrelation := 0.0

	for offset := 1; offset < windowSize; offset++ {
		correlation := 0.0
		count := 0

		for i := 0; i+offset < len(speeds); i++ {
			diff := math.Abs(speeds[i] - speeds[i+offset])
			correlation += 1.0 / (diff + 0.001)
			count++
		}

		if count > 0 {
			correlation /= float64(count)
			if correlation > maxCorrelation {
				maxCorrelation = correlation
			}
		}
	}

	avgSpeed := s.fastMean(speeds)
	normalizedCorrelation := maxCorrelation / (avgSpeed + 0.001)

	if normalizedCorrelation > 0.9 {
		return 1.0
	}

	return math.Min(1.0, normalizedCorrelation)
}

func (s *OptimizedLSTMService) detectRepeatedPatternFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 8 {
		return 0.0
	}

	speeds := s.calculateSpeedsFast(points)
	if len(speeds) < 4 {
		return 0.0
	}

	patternLengths := []int{2, 3, 4, 5}
	maxSimilarity := 0.0

	for _, patternLen := range patternLengths {
		if patternLen*2 > len(speeds) {
			continue
		}

		pattern1 := speeds[:patternLen]
		pattern2 := speeds[patternLen : patternLen*2]

		similarity := s.calculateSequenceSimilarityFast(pattern1, pattern2)

		if similarity > maxSimilarity {
			maxSimilarity = similarity
		}
	}

	if maxSimilarity > 0.95 {
		return 1.0
	}

	return maxSimilarity
}

func (s *OptimizedLSTMService) calculateSequenceSimilarityFast(seq1, seq2 []float64) float64 {
	if len(seq1) != len(seq2) || len(seq1) == 0 {
		return 0.0
	}

	sumDiff := 0.0
	for i := range seq1 {
		sumDiff += math.Abs(seq1[i] - seq2[i])
	}

	avg1 := s.fastMean(seq1)
	avg2 := s.fastMean(seq2)

	if avg1 == 0 && avg2 == 0 {
		return 1.0
	}

	normalizedDiff := sumDiff / float64(len(seq1)) / ((avg1+avg2)/2.0+0.001)

	return 1.0 - math.Min(1.0, normalizedDiff)
}

func (s *OptimizedLSTMService) calculateInterPointConsistencyFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	segments := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		segment := math.Sqrt(dx*dx + dy*dy)
		segments = append(segments, segment)
	}

	if len(segments) == 0 {
		return 0.0
	}

	avgSegment := s.fastMean(segments)
	variance := s.fastVariance(segments, avgSegment)

	if avgSegment == 0 {
		return 0.0
	}

	coefficientOfVariation := math.Sqrt(variance) / avgSegment

	return 1.0 / (1.0 + coefficientOfVariation)
}

func (s *OptimizedLSTMService) calculateSegmentRegularityFast(points []TrajectoryNNPoint) float64 {
	if len(points) < 4 {
		return 0.0
	}

	segments := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		segment := math.Sqrt(dx*dx + dy*dy)
		segments = append(segments, segment)
	}

	if len(segments) < 2 {
		return 0.0
	}

	timeDiffs := make([]float64, 0, len(points)-1)
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) != len(segments) {
		return 0.0
	}

	speedRatio := 0.0
	count := 0

	for i := range segments {
		if timeDiffs[i] > 0 {
			speed := segments[i] / timeDiffs[i]
			if i > 0 && timeDiffs[i-1] > 0 {
				prevSpeed := segments[i-1] / timeDiffs[i-1]
				ratio := speed / (prevSpeed + 0.001)
				if ratio > 0.5 && ratio < 2.0 {
					speedRatio += ratio
					count++
				}
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	avgRatio := speedRatio / float64(count)
	regularity := 1.0 - math.Abs(1.0-avgRatio)

	return math.Max(0, math.Min(1, regularity))
}

func (s *OptimizedLSTMService) flattenAndNormalizeFeatures(features TrajectoryFeatures) []float64 {
	result := make([]float64, s.config.FeatureDim)
	
	offset := 0
	copy(result[offset:offset+len(features.Basic)], features.Basic)
	offset += len(features.Basic)
	
	copy(result[offset:offset+len(features.Speed)], features.Speed)
	offset += len(features.Speed)
	
	copy(result[offset:offset+len(features.Direction)], features.Direction)
	offset += len(features.Direction)
	
	copy(result[offset:offset+len(features.Curvature)], features.Curvature)
	offset += len(features.Curvature)
	
	copy(result[offset:offset+len(features.Position)], features.Position)
	offset += len(features.Position)
	
	copy(result[offset:offset+len(features.Behavior)], features.Behavior)
	offset += len(features.Behavior)
	
	if offset < s.config.FeatureDim {
		copy(result[offset:offset+len(features.Advanced)], features.Advanced)
	}

	return result
}

func (s *OptimizedLSTMService) applyFeatureNormalization(features []float64) []float64 {
	normalized := make([]float64, len(features))
	norm := 0.0
	
	for _, f := range features {
		norm += f * f
	}
	norm = math.Sqrt(norm)
	
	if norm > 0.001 {
		for i, f := range features {
			normalized[i] = f / norm
		}
	}
	
	return normalized
}

func (s *OptimizedLSTMService) forwardLSTM(features []float64) []float64 {
	hidden := make([]float64, s.config.HiddenDim)
	cell := make([]float64, s.config.HiddenDim)

	for layer := 0; layer < s.config.NumLayers; layer++ {
		hidden, cell = s.lstmStep(features, hidden, cell, layer)
		
		if layer < s.config.NumLayers-1 && s.config.DropoutRate > 0 {
			hidden = s.applyDropout(hidden, s.config.DropoutRate)
		}
	}

	return hidden
}

func (s *OptimizedLSTMService) lstmStep(input, hidden, cell []float64, layer int) ([]float64, []float64) {
	inputGate := s.computeGate(input, hidden, s.lstmLayer.inputWeights, s.lstmLayer.hiddenWeights, s.lstmLayer.inputBias, 0, s.config.HiddenDim)
	forgetGate := s.computeGate(input, hidden, s.lstmLayer.inputWeights, s.lstmLayer.hiddenWeights, s.lstmLayer.inputBias, s.config.HiddenDim, s.config.HiddenDim*2)
	outputGate := s.computeGate(input, hidden, s.lstmLayer.inputWeights, s.lstmLayer.hiddenWeights, s.lstmLayer.inputBias, s.config.HiddenDim*2, s.config.HiddenDim*3)
	cellInput := s.computeGate(input, hidden, s.lstmLayer.inputWeights, s.lstmLayer.hiddenWeights, s.lstmLayer.inputBias, s.config.HiddenDim*3, s.config.HiddenDim*4)

	for i := 0; i < len(cell); i++ {
		inputGate[i] = 1.0 / (1.0 + math.Exp(-inputGate[i]))
		forgetGate[i] = 1.0 / (1.0 + math.Exp(-forgetGate[i]))
		outputGate[i] = 1.0 / (1.0 + math.Exp(-outputGate[i]))
		cellInput[i] = math.Tanh(cellInput[i])

		cell[i] = forgetGate[i]*cell[i] + inputGate[i]*cellInput[i]
		hidden[i] = outputGate[i] * math.Tanh(cell[i])
	}

	return hidden, cell
}

func (s *OptimizedLSTMService) computeGate(input, hidden []float64, inputWeights, hiddenWeights [][]float64, bias []float64, biasOffset, biasEnd int) []float64 {
	gate := make([]float64, s.config.HiddenDim)

	for i := 0; i < s.config.HiddenDim && biasOffset+i < biasEnd; i++ {
		sum := bias[biasOffset+i]
		
		for j := 0; j < len(input) && j < len(inputWeights); j++ {
			sum += input[j] * inputWeights[j][biasOffset+i]
		}
		
		for j := 0; j < len(hidden) && j < len(hiddenWeights); j++ {
			sum += hidden[j] * hiddenWeights[j][biasOffset+i]
		}
		
		gate[i] = sum
	}

	return gate
}

func (s *OptimizedLSTMService) applyDropout(input []float64, rate float64) []float64 {
	output := make([]float64, len(input))
	for i := range input {
		if mathRand() > rate {
			output[i] = input[i] / (1.0 - rate)
		}
	}
	return output
}

func (s *OptimizedLSTMService) applyAttention(lstmOutput []float64) []float64 {
	query := s.matVecMul(s.attentionLayer.queryWeights, lstmOutput, s.config.HiddenDim, s.config.HiddenDim)
	key := s.matVecMul(s.attentionLayer.keyWeights, lstmOutput, s.config.HiddenDim, s.config.HiddenDim)
	value := s.matVecMul(s.attentionLayer.valueWeights, lstmOutput, s.config.HiddenDim, s.config.HiddenDim)

	attentionScore := s.dotProduct(query, key) * s.attentionLayer.scale
	attentionWeight := math.Tanh(attentionScore)

	output := make([]float64, len(value))
	for i := range output {
		output[i] = value[i] * attentionWeight
	}

	return output
}

func (s *OptimizedLSTMService) matVecMul(weights []float64, vec []float64, rows, cols int) []float64 {
	result := make([]float64, rows)
	for i := 0; i < rows; i++ {
		sum := 0.0
		for j := 0; j < cols && i*cols+j < len(weights) && j < len(vec); j++ {
			sum += weights[i*cols+j] * vec[j]
		}
		result[i] = sum
	}
	return result
}

func (s *OptimizedLSTMService) dotProduct(a, b []float64) float64 {
	sum := 0.0
	for i := 0; i < len(a) && i < len(b); i++ {
		sum += a[i] * b[i]
	}
	return sum
}

func (s *OptimizedLSTMService) classify(attentionOutput []float64) float64 {
	logits := s.matVecMul(s.flatten2D(s.classifier.weights), attentionOutput, s.config.HiddenDim, 64)
	
	sum := 0.0
	for i := 0; i < len(logits) && i < len(s.classifier.bias); i++ {
		sum += logits[i] + s.classifier.bias[i]
	}
	
	sum /= float64(len(logits))
	
	score := 1.0 / (1.0 + math.Exp(-sum))
	
	return math.Max(0.0, math.Min(1.0, score))
}

func (s *OptimizedLSTMService) flatten2D(weights [][]float64) []float64 {
	result := make([]float64, 0, len(weights)*len(weights[0]))
	for _, row := range weights {
		result = append(result, row...)
	}
	return result
}

func (s *OptimizedLSTMService) calculateConfidence(score float64) float64 {
	confidence := 0.5 + math.Abs(score-0.5)
	return math.Max(0.0, math.Min(0.95, confidence))
}

func (s *OptimizedLSTMService) determineRiskLevel(score float64) string {
	switch {
	case score >= 0.85:
		return "extreme"
	case score >= 0.7:
		return "critical"
	case score >= 0.5:
		return "high"
	case score >= 0.3:
		return "medium"
	case score >= 0.15:
		return "low"
	default:
		return "safe"
	}
}

func (s *OptimizedLSTMService) extractFeatureMap(points []TrajectoryNNPoint, features []float64) map[string]float64 {
	featureMap := make(map[string]float64)
	
	if len(points) == 0 {
		return featureMap
	}
	
	speeds := s.calculateSpeedsFast(points)
	if len(speeds) > 0 {
		featureMap["avg_speed"] = s.fastMean(speeds)
		featureMap["max_speed"] = s.fastMax(speeds)
		featureMap["min_speed"] = s.fastMin(speeds)
		featureMap["speed_variance"] = s.fastVariance(speeds, featureMap["avg_speed"])
	}
	
	featureMap["point_count"] = float64(len(points))
	featureMap["total_distance"] = s.calculateTotalDistanceFast(points)
	
	directions := s.calculateDirectionsFast(points)
	if len(directions) > 0 {
		featureMap["direction_entropy"] = s.calculateDirectionEntropyFast(directions)
	}
	
	curvatures := s.calculateCurvaturesFast(points)
	if len(curvatures) > 0 {
		featureMap["avg_curvature"] = s.fastMean(curvatures)
		featureMap["curvature_variance"] = s.fastVariance(curvatures, featureMap["avg_curvature"])
	}
	
	featureMap["human_likelihood"] = s.calculateHumanLikelihoodFast(points)
	featureMap["mechanical_score"] = s.detectMechanicalMovementFast(points)
	featureMap["naturalness"] = s.calculateMovementNaturalnessFast(points)
	featureMap["complexity"] = s.calculateTrajectoryComplexityFast(points)
	
	return featureMap
}

func (s *OptimizedLSTMService) updatePerformanceMetrics(latencyMs, accuracy float64) {
	prevSum := s.metrics.TotalLatencySum.Load()
	prevCount := s.metrics.TotalLatencyCount.Load()
	s.metrics.TotalLatencySum.Store(prevSum + int64(latencyMs))
	s.metrics.TotalLatencyCount.Store(prevCount + 1)
	
	newCount := s.metrics.TotalLatencyCount.Load()
	avgLatency := float64(prevSum+int64(latencyMs)) / float64(newCount)
	s.metrics.AvgLatencyMs.Store(uint64(avgLatency))
	
	currentMax := s.metrics.MaxLatencyMs.Load()
	if int64(latencyMs) > currentMax {
		s.metrics.MaxLatencyMs.Store(int64(latencyMs))
	}
	
	currentMin := s.metrics.MinLatencyMs.Load()
	if currentMin == 0 || int64(latencyMs) < currentMin {
		s.metrics.MinLatencyMs.Store(int64(latencyMs))
	}
	
	s.metrics.AccuracySum.Add(uint64(accuracy * 1000000))
	s.metrics.AccuracyCount.Add(1)
}

func (c *OptimizedModelCache) Get(key string) *OptimizedCacheEntry {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.cache[key]
	if !exists {
		c.misses.Add(1)
		return nil
	}

	if time.Now().After(entry.Timestamp.Add(entry.TTL)) {
		delete(c.cache, key)
		c.misses.Add(1)
		return nil
	}

	c.hits.Add(1)
	return entry
}

func (c *OptimizedModelCache) Set(key string, features []float64, prediction *OptimizedPredictionResult) {
	c.mu.Lock()
	defer c.mu.Unlock()

	if len(c.cache) >= c.maxSize {
		c.evict()
	}

	entry := &OptimizedCacheEntry{
		Features:   features,
		Prediction: prediction,
		Timestamp:  time.Now(),
		TTL:        5 * time.Minute,
		Hash:       key,
	}

	c.cache[key] = entry
}

func (c *OptimizedModelCache) evict() {
	var oldestKey string
	var oldestTime time.Time

	for k, v := range c.cache {
		if oldestKey == "" || v.Timestamp.Before(oldestTime) {
			oldestKey = k
			oldestTime = v.Timestamp
		}
	}

	if oldestKey != "" {
		delete(c.cache, oldestKey)
		c.evictions.Add(1)
	}
}

func (w *OptimizedWeightCache) Get(key string) ([]float64, bool) {
	w.mu.RLock()
	defer w.mu.RUnlock()

	weights, exists := w.cache[key]
	if !exists {
		w.misses.Add(1)
		return nil, false
	}

	w.hits.Add(1)
	return weights, true
}

func (w *OptimizedWeightCache) Set(key string, weights []float64) {
	w.mu.Lock()
	defer w.mu.Unlock()
	w.cache[key] = weights
}

func (s *OptimizedLSTMService) GetMetrics() map[string]interface{} {
	total := s.metrics.TotalRequests.Load()
	cacheHits := s.metrics.CacheHits.Load()
	cacheMisses := s.metrics.CacheMisses.Load()
	
	cacheHitRate := 0.0
	if total > 0 {
		cacheHitRate = float64(cacheHits) / float64(total) * 100
	}
	
	avgAccuracy := 0.0
	accuracyCount := s.metrics.AccuracyCount.Load()
	if accuracyCount > 0 {
		avgAccuracy = float64(s.metrics.AccuracySum.Load()) / float64(accuracyCount) / 1000000.0
	}

	return map[string]interface{}{
		"total_requests":     total,
		"cache_hits":         cacheHits,
		"cache_misses":       cacheMisses,
		"cache_hit_rate":     fmt.Sprintf("%.2f%%", cacheHitRate),
		"avg_latency_ms":     fmt.Sprintf("%.2f", float64(s.metrics.AvgLatencyMs.Load())),
		"max_latency_ms":     s.metrics.MaxLatencyMs.Load(),
		"min_latency_ms":     s.metrics.MinLatencyMs.Load(),
		"accuracy":           fmt.Sprintf("%.2f%%", avgAccuracy*100),
		"feature_cache_size": len(s.featureCache.cache),
	}
}
