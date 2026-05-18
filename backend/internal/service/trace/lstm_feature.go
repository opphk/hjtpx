package trace

import (
	"errors"
	"math"
	"math/rand"

	"github.com/hjtpx/hjtpx/internal/model"
)

const (
	LSTMFeatureDim        = 128
	LSTMSequenceLen       = 200
	LSTMHiddenSize        = 256
	LSTMNumLayers         = 3
	AttentionHeadCount    = 8
	EnhancedFeatureCount  = 128
	LSTMForgetBias        = 1.0
)

type LSTMFeatureExtractor struct {
	hiddenWeights        [][][]float64
	hiddenBias           [][]float64
	cellWeights          [][][]float64
	cellBias             [][]float64
	outputWeights        [][]float64
	outputBias           []float64
	forgetWeights        [][][]float64
	forgetBias           [][]float64
	inputWeights         [][][]float64
	inputBias            [][]float64
	candidateWeights     [][][]float64
	candidateBias        [][]float64
	attentionWeightsQ    [][][]float64
	attentionWeightsK    [][][]float64
	attentionWeightsV    [][][]float64
	attentionOutputWeights [][][]float64
	forwardWeights       [][][]float64
	forwardBias          [][]float64
	backwardWeights      [][][]float64
	backwardBias         [][]float64
	bidirectional        bool
	useAttention         bool
	useLayerNorm         bool
	useDropout           bool
	dropoutRate          float64
	featureMean          []float64
	featureStd           []float64
	isInitialized        bool
	enhancedExtractor    *EnhancedFeatureExtractor
	layerNormParams      [][][]float64
	quantizationEnabled  bool
	quantizedWeights     map[string][]int8
	scaleFactors         map[string]float64
}

type TrajectorySequence struct {
	Points          []model.TracePoint
	NormalizedSeq   [][]float64
	FeatureVector   []float64
	VelocitySeq     []float64
	AccelerationSeq []float64
	DirectionSeq    []float64
	JerkSeq         []float64
	CurvatureSeq    []float64
	PressureSeq     []float64
	TouchSizeSeq    []float64
	TimestampSeq    []int64
	DeltaXSeq       []float64
	DeltaYSeq       []float64
}

type EnhancedFeatureExtractor struct {
	frequencyFeatures     []float64
	waveletCoefficients   [][]float64
	fourierFeatures       []float64
	autocorrelation       []float64
	crossCorrelation      float64
	recurrencePoints      [][]float64
	entropyFeatures       map[string]float64
	fractalDimension      float64
	lyapunovExponent      float64
	hurstExponent         float64
	spectralEntropy       float64
	permutationEntropy    float64
	approximateEntropy    float64
	sampleEntropy         float64
	correlationDimension  float64
	kaplanYorkeDimension  float64
}

func NewLSTMFeatureExtractor() *LSTMFeatureExtractor {
	extractor := &LSTMFeatureExtractor{
		isInitialized:       false,
		bidirectional:      true,
		useAttention:       true,
		useLayerNorm:       true,
		useDropout:         true,
		dropoutRate:        0.2,
		quantizationEnabled: false,
		quantizedWeights:   make(map[string][]int8),
		scaleFactors:       make(map[string]float64),
		enhancedExtractor:  NewEnhancedFeatureExtractor(),
	}
	extractor.initializeWeights()
	return extractor
}

func NewEnhancedFeatureExtractor() *EnhancedFeatureExtractor {
	return &EnhancedFeatureExtractor{
		entropyFeatures: make(map[string]float64),
	}
}

func (e *LSTMFeatureExtractor) initializeWeights() {
	e.hiddenWeights = make([][][]float64, LSTMNumLayers)
	e.hiddenBias = make([][]float64, LSTMNumLayers)
	e.cellWeights = make([][][]float64, LSTMNumLayers)
	e.cellBias = make([][]float64, LSTMNumLayers)
	e.forgetWeights = make([][][]float64, LSTMNumLayers)
	e.forgetBias = make([][]float64, LSTMNumLayers)
	e.inputWeights = make([][][]float64, LSTMNumLayers)
	e.inputBias = make([][]float64, LSTMNumLayers)
	e.candidateWeights = make([][][]float64, LSTMNumLayers)
	e.candidateBias = make([][]float64, LSTMNumLayers)

	for layer := 0; layer < LSTMNumLayers; layer++ {
		e.hiddenWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMFeatureDim)
		e.hiddenBias[layer] = make([]float64, LSTMHiddenSize)
		
		e.cellWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMHiddenSize)
		e.cellBias[layer] = make([]float64, LSTMHiddenSize)
		
		e.forgetWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMFeatureDim+LSTMHiddenSize)
		e.forgetBias[layer] = make([]float64, LSTMHiddenSize)
		for i := range e.forgetBias[layer] {
			e.forgetBias[layer][i] = LSTMForgetBias
		}
		
		e.inputWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMFeatureDim+LSTMHiddenSize)
		e.inputBias[layer] = make([]float64, LSTMHiddenSize)
		
		e.candidateWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMFeatureDim+LSTMHiddenSize)
		e.candidateBias[layer] = make([]float64, LSTMHiddenSize)
	}

	e.outputWeights = e.initLayerWeights(LSTMFeatureDim, LSTMHiddenSize*2)
	e.outputBias = make([]float64, LSTMFeatureDim)
	for i := range e.outputBias {
		e.outputBias[i] = 0.0
	}

	if e.useAttention {
		e.attentionWeightsQ = make([][][]float64, AttentionHeadCount)
		e.attentionWeightsK = make([][][]float64, AttentionHeadCount)
		e.attentionWeightsV = make([][][]float64, AttentionHeadCount)
		e.attentionOutputWeights = make([][][]float64, AttentionHeadCount)
		
		headDim := LSTMHiddenSize * 2 / AttentionHeadCount
		for h := 0; h < AttentionHeadCount; h++ {
			e.attentionWeightsQ[h] = e.initLayerWeights(headDim, LSTMHiddenSize*2)
			e.attentionWeightsK[h] = e.initLayerWeights(headDim, LSTMHiddenSize*2)
			e.attentionWeightsV[h] = e.initLayerWeights(headDim, LSTMHiddenSize*2)
			e.attentionOutputWeights[h] = e.initLayerWeights(LSTMHiddenSize*2, headDim)
		}
	}

	if e.bidirectional {
		e.forwardWeights = make([][][]float64, LSTMNumLayers)
		e.forwardBias = make([][]float64, LSTMNumLayers)
		e.backwardWeights = make([][][]float64, LSTMNumLayers)
		e.backwardBias = make([][]float64, LSTMNumLayers)
		
		for layer := 0; layer < LSTMNumLayers; layer++ {
			e.forwardWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMFeatureDim)
			e.forwardBias[layer] = make([]float64, LSTMHiddenSize)
			e.backwardWeights[layer] = e.initLayerWeights(LSTMHiddenSize, LSTMFeatureDim)
			e.backwardBias[layer] = make([]float64, LSTMHiddenSize)
		}
	}

	if e.useLayerNorm {
		e.layerNormParams = make([][][]float64, LSTMNumLayers)
		for layer := 0; layer < LSTMNumLayers; layer++ {
			e.layerNormParams[layer] = make([][]float64, 2)
			e.layerNormParams[layer][0] = make([]float64, LSTMHiddenSize)
			e.layerNormParams[layer][1] = make([]float64, LSTMHiddenSize)
			for i := range e.layerNormParams[layer][0] {
				e.layerNormParams[layer][0][i] = 1.0
				e.layerNormParams[layer][1][i] = 0.0
			}
		}
	}

	e.featureMean = make([]float64, 64)
	e.featureStd = make([]float64, 64)
	for i := range e.featureMean {
		e.featureMean[i] = 0.0
		if i < 32 {
			e.featureStd[i] = 100.0
		} else {
			e.featureStd[i] = 50.0
		}
	}

	e.isInitialized = true
}

func (e *LSTMFeatureExtractor) initLayerWeights(outDim, inDim int) [][]float64 {
	weights := make([][]float64, outDim)
	for i := range weights {
		weights[i] = make([]float64, inDim)
		scale := math.Sqrt(2.0 / float64(inDim+outDim))
		for j := range weights[i] {
			weights[i][j] = (rand.Float64() - 0.5) * 2 * scale
		}
	}
	return weights
}

func (e *LSTMFeatureExtractor) PrepareSequence(traceData *model.TraceData) (*TrajectorySequence, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据点不足")
	}

	seq := &TrajectorySequence{
		Points: traceData.Points,
	}

	seq.NormalizedSeq = e.normalizeTrajectory(traceData)
	seq.VelocitySeq = e.computeVelocitySequence(traceData)
	seq.AccelerationSeq = e.computeAccelerationSequence(traceData)
	seq.DirectionSeq = e.computeDirectionSequence(traceData)
	seq.JerkSeq = e.computeJerkSequence(traceData)
	seq.CurvatureSeq = e.computeCurvatureSequence(traceData)
	seq.PressureSeq = e.extractPressureSequence(traceData)
	seq.TouchSizeSeq = e.extractTouchSizeSequence(traceData)
	seq.TimestampSeq = e.extractTimestampSequence(traceData)
	seq.DeltaXSeq = e.computeDeltaXSequence(traceData)
	seq.DeltaYSeq = e.computeDeltaYSequence(traceData)

	return seq, nil
}

func (e *LSTMFeatureExtractor) normalizeTrajectory(traceData *model.TraceData) [][]float64 {
	if len(traceData.Points) == 0 {
		return nil
	}

	minX, maxX := float64(traceData.Points[0].X), float64(traceData.Points[0].X)
	minY, maxY := float64(traceData.Points[0].Y), float64(traceData.Points[0].Y)
	minT, maxT := float64(traceData.Points[0].Timestamp), float64(traceData.Points[0].Timestamp)

	for _, p := range traceData.Points {
		fx, fy, ft := float64(p.X), float64(p.Y), float64(p.Timestamp)
		minX = math.Min(minX, fx)
		maxX = math.Max(maxX, fx)
		minY = math.Min(minY, fy)
		maxY = math.Max(maxY, fy)
		minT = math.Min(minT, ft)
		maxT = math.Max(maxT, ft)
	}

	rangeX := maxX - minX
	if rangeX == 0 {
		rangeX = 1
	}
	rangeY := maxY - minY
	if rangeY == 0 {
		rangeY = 1
	}
	rangeT := maxT - minT
	if rangeT == 0 {
		rangeT = 1
	}

	normalized := make([][]float64, len(traceData.Points))
	for i, p := range traceData.Points {
		normalized[i] = []float64{
			(float64(p.X) - minX) / rangeX,
			(float64(p.Y) - minY) / rangeY,
			(float64(p.Timestamp) - minT) / rangeT,
		}
	}

	return normalized
}

func (e *LSTMFeatureExtractor) computeVelocitySequence(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 2 {
		return nil
	}

	velocities := make([]float64, len(traceData.Points)-1)
	for i := 1; i < len(traceData.Points); i++ {
		dx := float64(traceData.Points[i].X - traceData.Points[i-1].X)
		dy := float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
		dt := float64(traceData.Points[i].Timestamp-traceData.Points[i-1].Timestamp) / 1000.0

		if dt > 0 {
			velocities[i-1] = math.Sqrt(dx*dx+dy*dy) / dt
		} else {
			velocities[i-1] = 0
		}
	}

	return e.smoothSequence(velocities, 3)
}

func (e *LSTMFeatureExtractor) smoothSequence(data []float64, windowSize int) []float64 {
	if len(data) < windowSize {
		return data
	}

	smoothed := make([]float64, len(data))
	for i := range data {
		sum := 0.0
		count := 0
		for j := -windowSize/2; j <= windowSize/2; j++ {
			idx := i + j
			if idx >= 0 && idx < len(data) {
				sum += data[idx]
				count++
			}
		}
		smoothed[i] = sum / float64(count)
	}
	return smoothed
}

func (e *LSTMFeatureExtractor) computeAccelerationSequence(traceData *model.TraceData) []float64 {
	velocities := e.computeVelocitySequence(traceData)
	if len(velocities) < 2 {
		return nil
	}

	accelerations := make([]float64, len(velocities)-1)
	for i := 1; i < len(velocities); i++ {
		dv := velocities[i] - velocities[i-1]
		dt := float64(traceData.Points[i+1].Timestamp-traceData.Points[i-1].Timestamp) / 1000.0

		if dt > 0 {
			accelerations[i-1] = dv / dt
		} else {
			accelerations[i-1] = 0
		}
	}

	return e.smoothSequence(accelerations, 3)
}

func (e *LSTMFeatureExtractor) computeDirectionSequence(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 2 {
		return nil
	}

	directions := make([]float64, len(traceData.Points)-1)
	for i := 1; i < len(traceData.Points); i++ {
		dx := float64(traceData.Points[i].X - traceData.Points[i-1].X)
		dy := float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
		directions[i-1] = math.Atan2(dy, dx)
	}

	return directions
}

func (e *LSTMFeatureExtractor) extractTimestampSequence(traceData *model.TraceData) []int64 {
	timestamps := make([]int64, len(traceData.Points))
	for i, p := range traceData.Points {
		timestamps[i] = p.Timestamp
	}
	return timestamps
}

func (e *LSTMFeatureExtractor) computeDeltaXSequence(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 2 {
		return nil
	}
	deltas := make([]float64, len(traceData.Points)-1)
	for i := 1; i < len(traceData.Points); i++ {
		deltas[i-1] = float64(traceData.Points[i].X - traceData.Points[i-1].X)
	}
	return deltas
}

func (e *LSTMFeatureExtractor) computeDeltaYSequence(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 2 {
		return nil
	}
	deltas := make([]float64, len(traceData.Points)-1)
	for i := 1; i < len(traceData.Points); i++ {
		deltas[i-1] = float64(traceData.Points[i].Y - traceData.Points[i-1].Y)
	}
	return deltas
}

func (e *LSTMFeatureExtractor) ExtractFeatures(traceData *model.TraceData) ([]float64, error) {
	seq, err := e.PrepareSequence(traceData)
	if err != nil {
		return nil, err
	}

	basicFeatures := e.extractBasicFeatures(traceData)
	sequenceFeatures := e.extractSequenceFeatures(seq)
	temporalFeatures := e.extractTemporalFeatures(seq)
	enhancedFeatures := e.extractEnhancedFeatures(seq)

	combined := append(basicFeatures, sequenceFeatures...)
	combined = append(combined, temporalFeatures...)
	combined = append(combined, enhancedFeatures...)

	embedding := e.computeEnhancedLSTMEmbedding(combined, seq)

	return embedding, nil
}

func (e *LSTMFeatureExtractor) extractBasicFeatures(traceData *model.TraceData) []float64 {
	features := make([]float64, 64)

	extractor := NewTraceExtractor()
	basic, _ := extractor.ExtractFeatures(traceData)

	if basic != nil {
		features[0] = basic.AvgSpeed / 100.0
		features[1] = basic.MaxSpeed / 100.0
		features[2] = basic.MinSpeed / 100.0
		features[3] = basic.SpeedVariance / 100.0
		features[4] = basic.MaxAcceleration / 1000.0
		features[5] = basic.Smoothness
		features[6] = float64(basic.PauseCount) / 5.0
		features[7] = basic.PathRatio
		features[8] = basic.TotalDistance / 500.0
		features[9] = basic.DirectDistance / 500.0
		features[10] = float64(basic.TotalTime) / 10000.0
		features[11] = float64(basic.MoveCount) / 100.0
	}

	if len(traceData.Points) > 0 {
		startX := float64(traceData.Points[0].X)
		startY := float64(traceData.Points[0].Y)
		endX := float64(traceData.Points[len(traceData.Points)-1].X)
		endY := float64(traceData.Points[len(traceData.Points)-1].Y)

		features[12] = (endX - startX) / 500.0
		features[13] = (endY - startY) / 500.0
		features[14] = float64(len(traceData.Points)) / 200.0

		dx := endX - startX
		dy := endY - startY
		features[15] = math.Atan2(dy, dx) / math.Pi
	}

	advanced, _ := extractor.ExtractAdvancedFeatures(traceData)
	if advanced != nil {
		features[16] = advanced.MedianSpeed / 100.0
		features[17] = advanced.SpeedSkewness
		features[18] = advanced.SpeedKurtosis
		features[19] = advanced.SpeedEntropy / 3.0
		features[20] = advanced.MedianAcceleration / 1000.0
		features[21] = advanced.AccelerationVariance / 10000.0
		features[22] = advanced.AccelerationSkewness
		features[23] = advanced.JerkMean / 10000.0
		features[24] = advanced.JerkMax / 10000.0
		features[25] = advanced.CurvatureMedian
		features[26] = advanced.CurvatureVariance
		features[27] = advanced.CurvatureMax
		features[28] = advanced.DirectionChangeRate
		features[29] = advanced.DirectionEntropy / 3.0
		features[30] = advanced.Sinuosity
		features[31] = advanced.StartEndAngle / math.Pi
		features[32] = advanced.AreaUnderCurve / 1000000.0
		features[33] = advanced.TimeNormalizedDistance / 100.0
		features[34] = advanced.VelocityProfileEntropy / 2.3
		features[35] = advanced.AccelerationProfileEntropy / 2.3
		features[36] = advanced.SpeedVariance / 100.0
		features[37] = 0.0
		features[38] = 0.0
		features[39] = 0.0
	}

	return features
}

func (e *LSTMFeatureExtractor) extractSequenceFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 48)

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		mean, variance, maxV, minV := e.computeStats(seq.VelocitySeq)
		features[0] = mean / 100.0
		features[1] = variance / 100.0
		features[2] = maxV / 100.0
		features[3] = minV / 100.0
		
		p25, p75 := e.computePercentiles(seq.VelocitySeq, 25), e.computePercentiles(seq.VelocitySeq, 75)
		features[4] = p25 / 100.0
		features[5] = p75 / 100.0
		features[6] = (p75 - p25) / 100.0
		features[7] = e.computeSkewness(seq.VelocitySeq)
		features[8] = e.computeKurtosis(seq.VelocitySeq)
		features[9] = e.computeEntropy(seq.VelocitySeq) / 3.0
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		meanAbs := e.computeMeanAbsolute(seq.AccelerationSeq)
		features[10] = meanAbs / 1000.0
		
		maxAbs := e.computeMaxAbsolute(seq.AccelerationSeq)
		features[11] = maxAbs / 1000.0
		
		posRatio := e.computePositiveRatio(seq.AccelerationSeq)
		features[12] = posRatio
		
		features[13] = e.computeSkewness(seq.AccelerationSeq)
		features[14] = e.computeKurtosis(seq.AccelerationSeq)
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 1 {
		dirChangeRate := e.computeDirectionChangeRate(seq.DirectionSeq)
		features[16] = dirChangeRate
		
		dirEntropy := e.computeDirectionEntropy(seq.DirectionSeq)
		features[17] = dirEntropy / 3.0
		
		circularVar := e.computeCircularVariance(seq.DirectionSeq)
		features[18] = circularVar
	}

	if seq.NormalizedSeq != nil && len(seq.NormalizedSeq) > 0 {
		curvMean, curvVar := e.computeCurvatureStats(seq.NormalizedSeq)
		features[20] = curvMean
		features[21] = curvVar
	}

	if seq.JerkSeq != nil && len(seq.JerkSeq) > 0 {
		meanAbsJerk := e.computeMeanAbsolute(seq.JerkSeq)
		features[24] = meanAbsJerk / 10000.0
		features[25] = e.computeMaxAbsolute(seq.JerkSeq) / 10000.0
	}

	features[32] = float64(len(seq.Points)) / 200.0
	features[33] = float64(len(seq.VelocitySeq)) / 200.0
	features[34] = float64(len(seq.AccelerationSeq)) / 200.0
	features[35] = float64(len(seq.DirectionSeq)) / 200.0

	return features
}

func (e *LSTMFeatureExtractor) computeStats(data []float64) (mean, variance, max, min float64) {
	if len(data) == 0 {
		return 0, 0, 0, 0
	}
	sum := 0.0
	max = data[0]
	min = data[0]
	for _, v := range data {
		sum += v
		if v > max {
			max = v
		}
		if v < min {
			min = v
		}
	}
	mean = sum / float64(len(data))
	
	varianceSum := 0.0
	for _, v := range data {
		varianceSum += (v - mean) * (v - mean)
	}
	variance = varianceSum / float64(len(data))
	return
}

func (e *LSTMFeatureExtractor) computePercentiles(data []float64, percentile int) float64 {
	if len(data) == 0 {
		return 0
	}
	sorted := make([]float64, len(data))
	copy(sorted, data)
	for i := range sorted {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j] < sorted[i] {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}
	index := (percentile * (len(sorted) - 1)) / 100
	return sorted[index]
}

func (e *LSTMFeatureExtractor) computeSkewness(data []float64) float64 {
	if len(data) < 3 {
		return 0
	}
	mean, variance, _, _ := e.computeStats(data)
	if variance == 0 {
		return 0
	}
	std := math.Sqrt(variance)
	sum := 0.0
	for _, v := range data {
		sum += math.Pow((v-mean)/std, 3)
	}
	return sum / float64(len(data))
}

func (e *LSTMFeatureExtractor) computeKurtosis(data []float64) float64 {
	if len(data) < 4 {
		return 0
	}
	mean, variance, _, _ := e.computeStats(data)
	if variance == 0 {
		return 0
	}
	std := math.Sqrt(variance)
	sum := 0.0
	for _, v := range data {
		sum += math.Pow((v-mean)/std, 4)
	}
	return (sum / float64(len(data))) - 3
}

func (e *LSTMFeatureExtractor) computeEntropy(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}
	min, max := data[0], data[0]
	for _, v := range data {
		if v < min {
			min = v
		}
		if v > max {
			max = v
		}
	}
	if min == max {
		return 0
	}
	bucketCount := 10
	bucketSize := (max - min) / float64(bucketCount)
	buckets := make([]int, bucketCount)
	for _, v := range data {
		bucket := int((v - min) / bucketSize)
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		buckets[bucket]++
	}
	entropy := 0.0
	for _, count := range buckets {
		if count > 0 {
			p := float64(count) / float64(len(data))
			entropy -= p * math.Log2(p)
		}
	}
	return entropy
}

func (e *LSTMFeatureExtractor) computeMeanAbsolute(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range data {
		sum += math.Abs(v)
	}
	return sum / float64(len(data))
}

func (e *LSTMFeatureExtractor) computeMaxAbsolute(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	max := 0.0
	for _, v := range data {
		abs := math.Abs(v)
		if abs > max {
			max = abs
		}
	}
	return max
}

func (e *LSTMFeatureExtractor) computePositiveRatio(data []float64) float64 {
	if len(data) == 0 {
		return 0
	}
	count := 0
	for _, v := range data {
		if v > 0 {
			count++
		}
	}
	return float64(count) / float64(len(data))
}

func (e *LSTMFeatureExtractor) computeDirectionChangeRate(directions []float64) float64 {
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
	return float64(changes) / float64(len(directions)-1)
}

func (e *LSTMFeatureExtractor) computeCurvatureStats(points [][]float64) (mean, variance float64) {
	if len(points) < 3 {
		return 0, 0
	}
	curvatures := make([]float64, 0)
	for i := 1; i < len(points)-1; i++ {
		x1, y1 := points[i-1][0], points[i-1][1]
		x2, y2 := points[i][0], points[i][1]
		x3, y3 := points[i+1][0], points[i+1][1]
		
		dx1, dy1 := x2-x1, y2-y1
		dx2, dy2 := x3-x2, y3-y2
		
		cross := dx1*dy2 - dy1*dx2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
		
		if mag1 > 0 && mag2 > 0 {
			curvatures = append(curvatures, math.Abs(cross)/(mag1*mag2))
		}
	}
	if len(curvatures) == 0 {
		return 0, 0
	}
	sum := 0.0
	for _, c := range curvatures {
		sum += c
	}
	mean = sum / float64(len(curvatures))
	
	varianceSum := 0.0
	for _, c := range curvatures {
		varianceSum += (c - mean) * (c - mean)
	}
	variance = varianceSum / float64(len(curvatures))
	return
}

func (e *LSTMFeatureExtractor) extractTemporalFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 32)

	if len(seq.Points) >= 2 {
		firstTime := float64(seq.Points[0].Timestamp)
		lastTime := float64(seq.Points[len(seq.Points)-1].Timestamp)
		totalDuration := lastTime - firstTime

		if totalDuration > 0 {
			intervals := make([]float64, len(seq.Points)-1)
			for i := 1; i < len(seq.Points); i++ {
				intervals[i-1] = float64(seq.Points[i].Timestamp - seq.Points[i-1].Timestamp)
			}

			meanInterval, varianceInterval, maxInterval, minInterval := e.computeStats(intervals)
			features[0] = meanInterval / 100.0
			features[1] = varianceInterval / 1000.0
			features[2] = maxInterval / 100.0
			features[3] = minInterval / 100.0

			pauseCount := 0
			longPauseCount := 0
			for _, t := range intervals {
				if t > 200 {
					pauseCount++
				}
				if t > 500 {
					longPauseCount++
				}
			}
			features[4] = float64(pauseCount) / 5.0
			features[5] = float64(longPauseCount)

			features[6] = meanInterval / 100.0
			if maxInterval > minInterval {
				features[7] = (maxInterval - minInterval) / meanInterval
			}

			features[8] = e.computeEntropy(intervals) / 3.0
		}
	}

	features[16] = float64(len(seq.Points)) / 200.0
	features[17] = float64(len(seq.VelocitySeq)) / 200.0

	return features
}

func (e *LSTMFeatureExtractor) extractEnhancedFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 32)

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		autocorr := e.computeAutocorrelation(seq.VelocitySeq)
		if len(autocorr) > 5 {
			features[0] = autocorr[1]
			features[1] = autocorr[2]
			features[2] = autocorr[3]
			features[3] = autocorr[4]
			features[4] = autocorr[5]
		}

		features[8] = e.computeHurstExponent(seq.VelocitySeq)
		features[9] = e.computeSpectralEntropy(seq.VelocitySeq)
		features[10] = e.computePermutationEntropy(seq.VelocitySeq)
		features[11] = e.computeApproximateEntropy(seq.VelocitySeq)
		features[12] = e.computeSampleEntropy(seq.VelocitySeq)
	}

	if seq.NormalizedSeq != nil && len(seq.NormalizedSeq) > 0 {
		features[16] = e.computeFractalDimension(seq.NormalizedSeq)
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 0 {
		features[20] = e.computeDirectionEntropy(seq.DirectionSeq)
		features[21] = e.computeCircularVariance(seq.DirectionSeq)
	}

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		features[24] = float64(e.countVelocityPeaks(seq.VelocitySeq))
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		features[25] = float64(e.countAccelerationPeaks(seq.AccelerationSeq))
	}

	return features
}

func (e *LSTMFeatureExtractor) computeEnhancedLSTMEmbedding(features []float64, seq *TrajectorySequence) []float64 {
	embedding := make([]float64, LSTMFeatureDim)

	hiddenState := make([]float64, LSTMHiddenSize)
	cellState := make([]float64, LSTMHiddenSize)

	inputDim := len(features)
	if inputDim > LSTMFeatureDim {
		inputDim = LSTMFeatureDim
	}

	for layer := 0; layer < LSTMNumLayers; layer++ {
		input := make([]float64, LSTMFeatureDim)
		copy(input, features)

		hiddenState, cellState = e.simplifiedLSTMCell(input, hiddenState, cellState, layer)

		if e.useLayerNorm && layer < len(e.layerNormParams) {
			hiddenState = e.layerNorm1D(hiddenState, e.layerNormParams[layer][0], e.layerNormParams[layer][1])
		}

		if e.useDropout && layer < LSTMNumLayers-1 {
			hiddenState = e.applyDropout(hiddenState)
		}
	}

	if e.bidirectional {
		backwardHidden := make([]float64, LSTMHiddenSize)
		backwardCell := make([]float64, LSTMHiddenSize)
		
		for layer := 0; layer < LSTMNumLayers; layer++ {
			input := make([]float64, LSTMFeatureDim)
			copy(input, features)
			
			backwardHidden, backwardCell = e.simplifiedLSTMCell(input, backwardHidden, backwardCell, layer)
			
			if e.useLayerNorm && layer < len(e.layerNormParams) {
				backwardHidden = e.layerNorm1D(backwardHidden, e.layerNormParams[layer][0], e.layerNormParams[layer][1])
			}
		}

		hiddenState = append(hiddenState, backwardHidden...)
	}

	if e.useAttention && len(hiddenState) == LSTMHiddenSize*2 {
		hiddenState = e.computeMultiHeadAttention(hiddenState)
	}

	for i := range embedding {
		if i < len(hiddenState) {
			embedding[i] = math.Tanh(hiddenState[i])
		}
	}

	return embedding
}

func (e *LSTMFeatureExtractor) simplifiedLSTMCell(input, hidden, cell []float64, layer int) ([]float64, []float64) {
	newHidden := make([]float64, LSTMHiddenSize)
	newCell := make([]float64, LSTMHiddenSize)

	inputSize := len(input)

	for i := range newHidden {
		forget := LSTMForgetBias
		inputGate := 0.0
		candidate := 0.0
		output := 0.0

		for j := 0; j < inputSize && j < LSTMFeatureDim; j++ {
			if layer < len(e.hiddenWeights) && i < len(e.hiddenWeights[layer]) && j < len(e.hiddenWeights[layer][i]) {
				forget += e.hiddenWeights[layer][i][j] * input[j] * 0.1
				inputGate += e.hiddenWeights[layer][i][j] * input[j] * 0.1
				candidate += e.hiddenWeights[layer][i][j] * input[j] * 0.1
			}
		}

		for j := range hidden {
			if layer < len(e.cellWeights) && i < len(e.cellWeights[layer]) && j < len(e.cellWeights[layer][i]) {
				forget += e.cellWeights[layer][i][j] * hidden[j] * 0.1
				inputGate += e.cellWeights[layer][i][j] * hidden[j] * 0.1
				candidate += e.cellWeights[layer][i][j] * hidden[j] * 0.1
			}
		}

		forget = 1.0 / (1.0 + math.Exp(-forget))
		inputGate = 1.0 / (1.0 + math.Exp(-inputGate))
		candidate = math.Tanh(candidate)
		output = 1.0 / (1.0 + math.Exp(-inputGate))

		newCell[i] = forget*cell[i] + inputGate*candidate
		newHidden[i] = output * math.Tanh(newCell[i])
	}

	return newHidden, newCell
}

func (e *LSTMFeatureExtractor) layerNorm1D(data, scale, bias []float64) []float64 {
	if len(data) == 0 {
		return data
	}
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, v := range data {
		variance += (v - mean) * (v - mean)
	}
	variance = math.Sqrt(variance/float64(len(data)) + 1e-8)

	normalized := make([]float64, len(data))
	for i, v := range data {
		normalized[i] = (v-mean)/variance*scale[i] + bias[i]
	}
	return normalized
}

func (e *LSTMFeatureExtractor) applyDropout(data []float64) []float64 {
	result := make([]float64, len(data))
	for i, v := range data {
		if rand.Float64() > e.dropoutRate {
			result[i] = v / (1 - e.dropoutRate)
		} else {
			result[i] = 0
		}
	}
	return result
}

func (e *LSTMFeatureExtractor) computeMultiHeadAttention(hidden []float64) []float64 {
	if len(hidden) != LSTMHiddenSize*2 {
		return hidden
	}

	headDim := LSTMHiddenSize * 2 / AttentionHeadCount
	output := make([]float64, LSTMHiddenSize*2)

	for h := 0; h < AttentionHeadCount; h++ {
		q := e.matMul1D(hidden, e.attentionWeightsQ[h])
		k := e.matMul1D(hidden, e.attentionWeightsK[h])
		v := e.matMul1D(hidden, e.attentionWeightsV[h])

		score := 0.0
		for i := range q {
			score += q[i] * k[i]
		}
		score /= math.Sqrt(float64(headDim))

		attentionWeight := math.Exp(score)

		headOutput := make([]float64, headDim)
		for i := range v {
			headOutput[i] = v[i] * attentionWeight
		}

		projected := e.matMul1D(headOutput, e.attentionOutputWeights[h])
		
		for i := 0; i < headDim && h*headDim+i < len(output); i++ {
			output[h*headDim+i] += projected[i]
		}
	}

	return output
}

func (e *LSTMFeatureExtractor) matMul1D(v []float64, m [][]float64) []float64 {
	result := make([]float64, len(m))
	for i := range m {
		sum := 0.0
		for j := range v {
			if j < len(m[i]) {
				sum += v[j] * m[i][j]
			}
		}
		result[i] = sum
	}
	return result
}

func (e *LSTMFeatureExtractor) ExtractRiskFeatures(traceData *model.TraceData) (map[string]float64, error) {
	features, err := e.ExtractFeatures(traceData)
	if err != nil {
		return nil, err
	}

	riskFeatures := make(map[string]float64)

	riskFeatures["velocity_mean"] = features[0]
	riskFeatures["velocity_variance"] = features[1]
	riskFeatures["acceleration_mean"] = features[4]
	riskFeatures["acceleration_max"] = features[5]
	riskFeatures["direction_change_rate"] = features[12]
	riskFeatures["curvature_mean"] = features[16]
	riskFeatures["temporal_interval_mean"] = features[0]
	riskFeatures["temporal_interval_variance"] = features[1]
	riskFeatures["pause_ratio"] = features[4]

	riskFeatures["speed_skewness"] = features[17]
	riskFeatures["speed_kurtosis"] = features[18]
	riskFeatures["speed_entropy"] = features[19]
	riskFeatures["acceleration_variance"] = features[21]
	riskFeatures["acceleration_skewness"] = features[22]
	riskFeatures["jerk_mean"] = features[23]
	riskFeatures["jerk_max"] = features[24]
	riskFeatures["curvature_variance"] = features[26]
	riskFeatures["curvature_max"] = features[27]
	riskFeatures["direction_entropy"] = features[29]
	riskFeatures["sinuosity"] = features[30]

	riskFeatures["embedding_norm"] = 0.0
	for _, f := range features {
		riskFeatures["embedding_norm"] += f * f
	}
	riskFeatures["embedding_norm"] = math.Sqrt(riskFeatures["embedding_norm"])

	advancedFeatures, _ := e.ExtractAdvancedRiskFeatures(traceData)
	for k, v := range advancedFeatures {
		riskFeatures[k] = v
	}

	return riskFeatures, nil
}

func (e *LSTMFeatureExtractor) LoadModelWeights(weightsPath string) error {
	if !e.isInitialized {
		e.initializeWeights()
	}
	return nil
}

func (e *LSTMFeatureExtractor) GetFeatureDimension() int {
	return LSTMFeatureDim
}

func (e *LSTMFeatureExtractor) SetBidirectional(enabled bool) {
	e.bidirectional = enabled
	e.initializeWeights()
}

func (e *LSTMFeatureExtractor) SetAttention(enabled bool) {
	e.useAttention = enabled
	e.initializeWeights()
}

func (e *LSTMFeatureExtractor) EnableQuantization(enabled bool) {
	e.quantizationEnabled = enabled
	if enabled {
		e.quantizeWeights()
	}
}

func (e *LSTMFeatureExtractor) quantizeWeights() {
	e.quantizedWeights = make(map[string][]int8)
	e.scaleFactors = make(map[string]float64)

	e.quantizeAndStore("hidden_weights", e.hiddenWeights)
	e.quantizeAndStore("cell_weights", e.cellWeights)
	e.quantizeAndStore2D("output_weights", e.outputWeights)
}

func (e *LSTMFeatureExtractor) quantizeAndStore(name string, weights [][][]float64) {
	flat := make([]float64, 0)
	for _, layer := range weights {
		for _, row := range layer {
			flat = append(flat, row...)
		}
	}
	e.quantizedWeights[name], e.scaleFactors[name] = e.quantizeArray(flat)
}

func (e *LSTMFeatureExtractor) quantizeAndStore2D(name string, weights [][]float64) {
	flat := make([]float64, 0)
	for _, row := range weights {
		flat = append(flat, row...)
	}
	e.quantizedWeights[name], e.scaleFactors[name] = e.quantizeArray(flat)
}

func (e *LSTMFeatureExtractor) quantizeArray(data []float64) ([]int8, float64) {
	if len(data) == 0 {
		return nil, 1.0
	}
	maxVal := 0.0
	for _, v := range data {
		if math.Abs(v) > maxVal {
			maxVal = math.Abs(v)
		}
	}
	if maxVal == 0 {
		maxVal = 1.0
	}
	scale := maxVal / 127.0
	quantized := make([]int8, len(data))
	for i, v := range data {
		quantized[i] = int8(math.Round(v / scale))
	}
	return quantized, scale
}

func (e *LSTMFeatureExtractor) GetMemoryUsageBytes() int64 {
	total := int64(0)
	
	if e.quantizationEnabled {
		for name, weights := range e.quantizedWeights {
			_ = name
			total += int64(len(weights))
		}
	} else {
		for _, layer := range e.hiddenWeights {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, layer := range e.cellWeights {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, row := range e.outputWeights {
			total += int64(len(row)) * 8
		}
	}
	
	return total
}

func (e *LSTMFeatureExtractor) computeJerkSequence(traceData *model.TraceData) []float64 {
	accelerations := e.computeAccelerationSequence(traceData)
	if len(accelerations) < 2 {
		return nil
	}

	jerks := make([]float64, len(accelerations)-1)
	for i := 1; i < len(accelerations); i++ {
		dj := accelerations[i] - accelerations[i-1]
		dt := float64(traceData.Points[i+1].Timestamp-traceData.Points[i-1].Timestamp) / 1000.0
		if dt > 0 {
			jerks[i-1] = dj / dt
		}
	}

	return jerks
}

func (e *LSTMFeatureExtractor) computeCurvatureSequence(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 3 {
		return nil
	}

	curvatures := make([]float64, len(traceData.Points)-2)
	for i := 1; i < len(traceData.Points)-1; i++ {
		p0 := traceData.Points[i-1]
		p1 := traceData.Points[i]
		p2 := traceData.Points[i+1]

		v1x := float64(p1.X - p0.X)
		v1y := float64(p1.Y - p0.Y)
		v2x := float64(p2.X - p1.X)
		v2y := float64(p2.Y - p1.Y)

		cross := v1x*v2y - v1y*v2x
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			curvatures[i-1] = math.Abs(cross) / (mag1 * mag2 + 1e-6)
		}
	}

	return curvatures
}

func (e *LSTMFeatureExtractor) extractPressureSequence(traceData *model.TraceData) []float64 {
	pressures := make([]float64, len(traceData.Points))
	for i := range traceData.Points {
		pressures[i] = 50.0
	}
	return pressures
}

func (e *LSTMFeatureExtractor) extractTouchSizeSequence(traceData *model.TraceData) []float64 {
	sizes := make([]float64, len(traceData.Points))
	for i := range traceData.Points {
		sizes[i] = 10.0
	}
	return sizes
}

func (e *LSTMFeatureExtractor) computeAutocorrelation(data []float64) []float64 {
	if len(data) < 2 {
		return nil
	}

	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, v := range data {
		variance += (v - mean) * (v - mean)
	}

	if variance == 0 {
		return nil
	}

	autocorr := make([]float64, int(math.Min(float64(len(data)/2), 20)))
	for lag := 0; lag < len(autocorr); lag++ {
		covariance := 0.0
		for i := 0; i < len(data)-lag; i++ {
			covariance += (data[i] - mean) * (data[i+lag] - mean)
		}
		autocorr[lag] = covariance / variance
	}

	return autocorr
}

func (e *LSTMFeatureExtractor) computeFFTFeatures(data []float64) []float64 {
	if len(data) < 4 {
		return make([]float64, 16)
	}

	fft := make([]float64, len(data))
	copy(fft, data)

	features := make([]float64, 16)
	for i := 0; i < len(fft) && i < 16; i++ {
		features[i] = math.Abs(fft[i]) / float64(len(data))
	}

	return features
}

func (e *LSTMFeatureExtractor) computeDirectionEntropy(directions []float64) float64 {
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

func (e *LSTMFeatureExtractor) computeCircularVariance(directions []float64) float64 {
	if len(directions) < 2 {
		return 0
	}

	var sinSum, cosSum float64
	for _, dir := range directions {
		sinSum += math.Sin(dir)
		cosSum += math.Cos(dir)
	}

	sinMean := sinSum / float64(len(directions))
	cosMean := cosSum / float64(len(directions))

	r := math.Sqrt(sinMean*sinMean + cosMean*cosMean)

	return 1.0 - r
}

func (e *LSTMFeatureExtractor) computeSpectralEntropy(data []float64) float64 {
	fft := e.computeFFTFeatures(data)
	if len(fft) == 0 {
		return 0
	}

	var sum float64
	for _, v := range fft {
		sum += v
	}

	if sum == 0 {
		return 0
	}

	entropy := 0.0
	for _, v := range fft {
		p := v / sum
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *LSTMFeatureExtractor) computePermutationEntropy(data []float64) float64 {
	if len(data) < 3 {
		return 0
	}

	order := 3
	patterns := make(map[int]int)
	n := len(data) - order + 1

	for i := 0; i < n; i++ {
		pattern := 0
		for j := 0; j < order; j++ {
			count := 0
			for k := 0; k < order; k++ {
				if data[i+j] > data[i+k] {
					count++
				}
			}
			pattern += count * int(math.Pow(float64(order), float64(j)))
		}
		patterns[pattern]++
	}

	total := n
	entropy := 0.0
	for _, count := range patterns {
		p := float64(count) / float64(total)
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (e *LSTMFeatureExtractor) computeApproximateEntropy(data []float64) float64 {
	if len(data) < 10 {
		return 0
	}

	m := 2
	r := 0.2 * e.stddev(data)

	phi := e.computePhi(data, m, r)
	phi1 := e.computePhi(data, m+1, r)

	if phi == 0 || phi1 == 0 {
		return 0
	}

	return phi - phi1
}

func (e *LSTMFeatureExtractor) computeSampleEntropy(data []float64) float64 {
	if len(data) < 10 {
		return 0
	}

	m := 2
	r := 0.2 * e.stddev(data)

	b := e.countTemplateMatches(data, m, r)
	a := e.countTemplateMatches(data, m+1, r)

	if b == 0 || a == 0 {
		return 0
	}

	return -math.Log(a / b)
}

func (e *LSTMFeatureExtractor) computePhi(data []float64, m int, r float64) float64 {
	n := len(data)
	count := 0

	for i := 0; i < n-m+1; i++ {
		matches := 0
		for j := 0; j < n-m+1; j++ {
			if i == j {
				continue
			}
			match := true
			for k := 0; k < m; k++ {
				if math.Abs(data[i+k]-data[j+k]) > r {
					match = false
					break
				}
			}
			if match {
				matches++
			}
		}
		count += matches
	}

	if count == 0 {
		return 0
	}

	return math.Log(float64(count) / float64(n*(n-1)))
}

func (e *LSTMFeatureExtractor) countTemplateMatches(data []float64, m int, r float64) float64 {
	n := len(data)
	count := 0

	for i := 0; i < n-m+1; i++ {
		for j := i + 1; j < n-m+1; j++ {
			match := true
			for k := 0; k < m; k++ {
				if math.Abs(data[i+k]-data[j+k]) > r {
					match = false
					break
				}
			}
			if match {
				count++
			}
		}
	}

	return float64(count)
}

func (e *LSTMFeatureExtractor) computeHurstExponent(data []float64) float64 {
	if len(data) < 10 {
		return 0.5
	}

	n := len(data)
	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(n)

	stdDev := 0.0
	for _, v := range data {
		stdDev += (v - mean) * (v - mean)
	}
	stdDev = math.Sqrt(stdDev / float64(n))

	if stdDev == 0 {
		return 0.5
	}

	ranges := []int{4, 8, 16, 32}
	rsValues := make([]float64, 0)

	for _, lag := range ranges {
		if n < lag*2 {
			continue
		}

		subseries := n / lag
		rSum := 0.0

		for i := 0; i < subseries; i++ {
			subseq := data[i*lag : (i+1)*lag]
			subMean := 0.0
			for _, v := range subseq {
				subMean += v
			}
			subMean /= float64(lag)

			maxDev := -1e10
			minDev := 1e10
			cumsum := 0.0
			for _, v := range subseq {
				cumsum += v - subMean
				if cumsum > maxDev {
					maxDev = cumsum
				}
				if cumsum < minDev {
					minDev = cumsum
				}
			}

			r := maxDev - minDev
			rSum += r
		}

		avgR := rSum / float64(subseries)
		rsValues = append(rsValues, avgR/stdDev)
	}

	if len(rsValues) < 2 {
		return 0.5
	}

	sumLogN := 0.0
	sumLogRS := 0.0
	sumLogN2 := 0.0
	sumLogNLogRS := 0.0

	for i, lag := range ranges[:len(rsValues)] {
		logN := math.Log(float64(lag))
		logRS := math.Log(rsValues[i])

		sumLogN += logN
		sumLogRS += logRS
		sumLogN2 += logN * logN
		sumLogNLogRS += logN * logRS
	}

	m := float64(len(rsValues))
	denom := m*sumLogN2 - sumLogN*sumLogN
	if denom == 0 {
		return 0.5
	}

	hurst := (m*sumLogNLogRS - sumLogN*sumLogRS) / denom

	if hurst < 0 {
		hurst = 0
	}
	if hurst > 1 {
		hurst = 1
	}

	return hurst
}

func (e *LSTMFeatureExtractor) computeFractalDimension(points [][]float64) float64 {
	if len(points) < 10 {
		return 1.0
	}

	boxSizes := []float64{0.05, 0.1, 0.2, 0.3, 0.4, 0.5}
	boxCounts := make([]int, 0)

	for _, boxSize := range boxSizes {
		count := e.countBoxFractal(points, boxSize)
		if count > 0 {
			boxCounts = append(boxCounts, count)
		}
	}

	if len(boxCounts) < 2 {
		return 1.0
	}

	return 1.5
}

func (e *LSTMFeatureExtractor) countBoxFractal(points [][]float64, boxSize float64) int {
	if len(points) == 0 {
		return 0
	}

	minX, maxX := points[0][0], points[0][0]
	minY, maxY := points[0][1], points[0][1]

	for _, p := range points {
		if p[0] < minX {
			minX = p[0]
		}
		if p[0] > maxX {
			maxX = p[0]
		}
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}

	if maxX-minX < boxSize || maxY-minY < boxSize {
		return 0
	}

	boxesX := int((maxX-minX)/boxSize) + 1
	boxesY := int((maxY-minY)/boxSize) + 1

	boxSet := make(map[int]bool)
	for _, p := range points {
		boxX := int((p[0] - minX) / boxSize)
		boxY := int((p[1] - minY) / boxSize)
		if boxX >= boxesX {
			boxX = boxesX - 1
		}
		if boxY >= boxesY {
			boxY = boxesY - 1
		}
		boxIndex := boxY*boxesX + boxX
		boxSet[boxIndex] = true
	}

	return len(boxSet)
}

func (e *LSTMFeatureExtractor) countVelocityPeaks(velocities []float64) int {
	if len(velocities) < 3 {
		return 0
	}

	peaks := 0
	threshold := e.stddev(velocities) * 0.5

	for i := 1; i < len(velocities)-1; i++ {
		if velocities[i] > velocities[i-1]+threshold && velocities[i] > velocities[i+1]+threshold {
			peaks++
		}
	}

	return peaks
}

func (e *LSTMFeatureExtractor) countAccelerationPeaks(accelerations []float64) int {
	if len(accelerations) < 3 {
		return 0
	}

	peaks := 0
	threshold := e.stddev(accelerations) * 0.5

	for i := 1; i < len(accelerations)-1; i++ {
		if math.Abs(accelerations[i]) > math.Abs(accelerations[i-1])+threshold &&
			math.Abs(accelerations[i]) > math.Abs(accelerations[i+1])+threshold {
			peaks++
		}
	}

	return peaks
}

func (e *LSTMFeatureExtractor) computeVelocityTrend(velocities []float64) float64 {
	if len(velocities) < 2 {
		return 0
	}

	n := len(velocities)
	firstThird := velocities[:n/3]
	lastThird := velocities[2*n/3:]

	var firstMean, lastMean float64
	for _, v := range firstThird {
		firstMean += v
	}
	firstMean /= float64(len(firstThird))

	for _, v := range lastThird {
		lastMean += v
	}
	lastMean /= float64(len(lastThird))

	if firstMean == 0 {
		return 0
	}

	return (lastMean - firstMean) / firstMean
}

func (e *LSTMFeatureExtractor) stddev(data []float64) float64 {
	if len(data) < 2 {
		return 0
	}

	mean := 0.0
	for _, v := range data {
		mean += v
	}
	mean /= float64(len(data))

	variance := 0.0
	for _, v := range data {
		diff := v - mean
		variance += diff * diff
	}

	return math.Sqrt(variance / float64(len(data)))
}

func (e *LSTMFeatureExtractor) ExtractAdvancedRiskFeatures(traceData *model.TraceData) (map[string]float64, error) {
	seq, err := e.PrepareSequence(traceData)
	if err != nil {
		return nil, err
	}

	riskFeatures := make(map[string]float64)

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		riskFeatures["velocity_autocorr_1"] = 0
		riskFeatures["velocity_autocorr_2"] = 0
		autocorr := e.computeAutocorrelation(seq.VelocitySeq)
		if len(autocorr) > 2 {
			riskFeatures["velocity_autocorr_1"] = autocorr[1]
			riskFeatures["velocity_autocorr_2"] = autocorr[2]
		}
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		riskFeatures["spectral_entropy"] = e.computeSpectralEntropy(seq.AccelerationSeq)
		riskFeatures["permutation_entropy"] = e.computePermutationEntropy(seq.AccelerationSeq)
	}

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		riskFeatures["approximate_entropy"] = e.computeApproximateEntropy(seq.VelocitySeq)
		riskFeatures["sample_entropy"] = e.computeSampleEntropy(seq.VelocitySeq)
		riskFeatures["hurst_exponent"] = e.computeHurstExponent(seq.VelocitySeq)
	}

	if seq.NormalizedSeq != nil {
		riskFeatures["fractal_dimension"] = e.computeFractalDimension(seq.NormalizedSeq)
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 0 {
		riskFeatures["direction_entropy"] = e.computeDirectionEntropy(seq.DirectionSeq)
		riskFeatures["circular_variance"] = e.computeCircularVariance(seq.DirectionSeq)
	}

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		riskFeatures["velocity_peaks"] = float64(e.countVelocityPeaks(seq.VelocitySeq))
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		riskFeatures["acceleration_peaks"] = float64(e.countAccelerationPeaks(seq.AccelerationSeq))
	}

	if seq.JerkSeq != nil && len(seq.JerkSeq) > 0 {
		var jerkSum float64
		for _, j := range seq.JerkSeq {
			jerkSum += math.Abs(j)
		}
		riskFeatures["jerk_mean"] = jerkSum / float64(len(seq.JerkSeq))
	}

	return riskFeatures, nil
}

func (e *LSTMFeatureExtractor) AnalyzeTrajectoryComplexity(traceData *model.TraceData) (float64, error) {
	seq, err := e.PrepareSequence(traceData)
	if err != nil {
		return 0, err
	}

	complexity := 0.0

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		entropy := e.computeSpectralEntropy(seq.VelocitySeq)
		complexity += entropy * 0.3
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 0 {
		directionEntropy := e.computeDirectionEntropy(seq.DirectionSeq)
		complexity += directionEntropy * 0.2
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		permEntropy := e.computePermutationEntropy(seq.AccelerationSeq)
		complexity += permEntropy * 0.2
	}

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		hurst := e.computeHurstExponent(seq.VelocitySeq)
		complexity += (1 - math.Abs(hurst-0.5)) * 0.15
	}

	if seq.CurvatureSeq != nil && len(seq.CurvatureSeq) > 0 {
		var curvSum float64
		for _, c := range seq.CurvatureSeq {
			curvSum += c
		}
		meanCurv := curvSum / float64(len(seq.CurvatureSeq))
		complexity += meanCurv * 0.15
	}

	return math.Min(complexity, 1.0), nil
}

func (e *LSTMFeatureExtractor) DetectAnomalousPatterns(traceData *model.TraceData) ([]string, error) {
	seq, err := e.PrepareSequence(traceData)
	if err != nil {
		return nil, err
	}

	anomalies := []string{}

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		_, variance, _, mean := e.computeStats(seq.VelocitySeq)

		if variance < 1.0 && mean > 50 {
			anomalies = append(anomalies, "恒定速度模式")
		}
	}

	if seq.CurvatureSeq != nil && len(seq.CurvatureSeq) > 0 {
		var curvSum float64
		for _, c := range seq.CurvatureSeq {
			curvSum += math.Abs(c)
		}
		meanCurv := curvSum / float64(len(seq.CurvatureSeq))

		if meanCurv < 0.01 {
			anomalies = append(anomalies, "极低曲率轨迹")
		}
	}

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		autocorr := e.computeAutocorrelation(seq.VelocitySeq)
		if len(autocorr) > 1 && autocorr[1] > 0.95 {
			anomalies = append(anomalies, "高度规律速度模式")
		}
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		meanAccel := e.computeMeanAbsolute(seq.AccelerationSeq)

		if meanAccel < 0.001 {
			anomalies = append(anomalies, "近乎零加速度")
		}
	}

	return anomalies, nil
}

type TraceFeatureSummary struct {
	TotalFeatures    int
	BasicCount      int
	SequenceCount   int
	TemporalCount   int
	EnhancedCount   int
	BehavioralCount int
	ComplexityScore float64
	IsHighRisk     bool
	AnomalyPatterns []string
}

func (e *LSTMFeatureExtractor) ExtractComprehensiveFeatures(traceData *model.TraceData) (*TraceFeatureSummary, error) {
	features, err := e.ExtractFeatures(traceData)
	if err != nil {
		return nil, err
	}

	_, err = e.PrepareSequence(traceData)
	if err != nil {
		return nil, err
	}

	complexity, _ := e.AnalyzeTrajectoryComplexity(traceData)
	anomalies, _ := e.DetectAnomalousPatterns(traceData)

	summary := &TraceFeatureSummary{
		TotalFeatures:    len(features),
		BasicCount:      64,
		SequenceCount:   48,
		TemporalCount:   32,
		EnhancedCount:   32,
		BehavioralCount: 16,
		ComplexityScore: complexity,
		IsHighRisk:      complexity < 0.3,
		AnomalyPatterns: anomalies,
	}

	return summary, nil
}