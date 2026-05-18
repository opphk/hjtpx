package trace

import (
	"errors"
	"math"
	"math/rand"

	"github.com/hjtpx/hjtpx/internal/model"
)

const (
	LSTMFeatureDim      = 64
	LSTMSequenceLen   = 100
	LSTMHiddenSize    = 128
	LSTMNumLayers     = 2
	AttentionHeadCount = 8
	EnhancedFeatureCount = 128
)

type LSTMFeatureExtractor struct {
	hiddenWeights     [][][]float64
	hiddenBias        [][]float64
	cellWeights       [][][]float64
	cellBias          [][]float64
	outputWeights     []float64
	attentionWeights  [][][]float64
	forwardWeights    [][][]float64
	backwardWeights   [][][]float64
	bidirectional     bool
	useAttention      bool
	featureMean       []float64
	featureStd        []float64
	isInitialized     bool
	enhancedExtractor *EnhancedFeatureExtractor
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
		isInitialized:     false,
		bidirectional:     true,
		useAttention:      true,
		enhancedExtractor: NewEnhancedFeatureExtractor(),
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

	for layer := 0; layer < LSTMNumLayers; layer++ {
		e.hiddenWeights[layer] = make([][]float64, LSTMHiddenSize)
		for i := range e.hiddenWeights[layer] {
			e.hiddenWeights[layer][i] = e.initXavier(LSTMFeatureDim * 4)
		}

		e.hiddenBias[layer] = make([]float64, LSTMHiddenSize*4)
		for i := range e.hiddenBias[layer] {
			e.hiddenBias[layer][i] = 0.0
		}

		e.cellWeights[layer] = make([][]float64, LSTMHiddenSize)
		for i := range e.cellWeights[layer] {
			e.cellWeights[layer][i] = e.initXavier(LSTMFeatureDim)
		}

		e.cellBias[layer] = make([]float64, LSTMHiddenSize)
		for i := range e.cellBias[layer] {
			e.cellBias[layer][i] = 0.0
		}
	}

	e.outputWeights = make([]float64, LSTMHiddenSize*2)
	for i := range e.outputWeights {
		e.outputWeights[i] = 0.1
	}

	if e.useAttention {
		e.attentionWeights = make([][][]float64, 3)
		for i := range e.attentionWeights {
			e.attentionWeights[i] = e.initLayerWeights(LSTMHiddenSize*2, LSTMHiddenSize*2)
		}
	}

	if e.bidirectional {
		e.forwardWeights = make([][][]float64, LSTMNumLayers)
		e.backwardWeights = make([][][]float64, LSTMNumLayers)
		for layer := 0; layer < LSTMNumLayers; layer++ {
			e.forwardWeights[layer] = e.hiddenWeights[layer]
			e.backwardWeights[layer] = make([][]float64, LSTMHiddenSize)
			for i := range e.backwardWeights[layer] {
				e.backwardWeights[layer][i] = e.initXavier(LSTMFeatureDim * 4)
			}
		}
	}

	e.featureMean = make([]float64, 48)
	e.featureStd = make([]float64, 48)
	for i := range e.featureMean {
		e.featureMean[i] = 0.0
		if i < 24 {
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
			weights[i][j] = (mathrand() - 0.5) * 2 * scale
		}
	}
	return weights
}

func (e *LSTMFeatureExtractor) initXavier(size int) []float64 {
	weights := make([]float64, size)
	scale := math.Sqrt(2.0 / float64(size+LSTMFeatureDim))
	for i := range weights {
		weights[i] = (mathrand() - 0.5) * 2 * scale
	}
	return weights
}

func mathrand() float64 {
	return rand.Float64()
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

	return seq, nil
}

func (e *LSTMFeatureExtractor) normalizeTrajectory(traceData *model.TraceData) [][]float64 {
	if len(traceData.Points) == 0 {
		return nil
	}

	minX, maxX := traceData.Points[0].X, traceData.Points[0].X
	minY, maxY := traceData.Points[0].Y, traceData.Points[0].Y
	minT, maxT := traceData.Points[0].Timestamp, traceData.Points[0].Timestamp

	for _, p := range traceData.Points {
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
		if p.Timestamp < minT {
			minT = p.Timestamp
		}
		if p.Timestamp > maxT {
			maxT = p.Timestamp
		}
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
			(p.X - minX) / rangeX,
			(p.Y - minY) / rangeY,
			float64(p.Timestamp-minT) / float64(rangeT),
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
		dx := traceData.Points[i].X - traceData.Points[i-1].X
		dy := traceData.Points[i].Y - traceData.Points[i-1].Y
		dt := float64(traceData.Points[i].Timestamp-traceData.Points[i-1].Timestamp) / 1000.0

		if dt > 0 {
			velocities[i-1] = math.Sqrt(dx*dx+dy*dy) / dt
		} else {
			velocities[i-1] = 0
		}
	}

	return velocities
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

	return accelerations
}

func (e *LSTMFeatureExtractor) computeDirectionSequence(traceData *model.TraceData) []float64 {
	if len(traceData.Points) < 2 {
		return nil
	}

	directions := make([]float64, len(traceData.Points)-1)
	for i := 1; i < len(traceData.Points); i++ {
		dx := traceData.Points[i].X - traceData.Points[i-1].X
		dy := traceData.Points[i].Y - traceData.Points[i-1].Y

		directions[i-1] = math.Atan2(dy, dx)
	}

	return directions
}

func (e *LSTMFeatureExtractor) ExtractFeatures(traceData *model.TraceData) ([]float64, error) {
	seq, err := e.PrepareSequence(traceData)
	if err != nil {
		return nil, err
	}

	basicFeatures := e.extractBasicFeatures(traceData)

	sequenceFeatures := e.extractSequenceFeatures(seq)

	temporalFeatures := e.extractTemporalFeatures(seq)

	combined := append(basicFeatures, sequenceFeatures...)
	combined = append(combined, temporalFeatures...)

	embedding := e.computeLSTMEmbedding(combined)

	return embedding, nil
}

func (e *LSTMFeatureExtractor) extractBasicFeatures(traceData *model.TraceData) []float64 {
	features := make([]float64, 48)

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
		startX := traceData.Points[0].X
		startY := traceData.Points[0].Y
		endX := traceData.Points[len(traceData.Points)-1].X
		endY := traceData.Points[len(traceData.Points)-1].Y

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
	}

	return features
}

func (e *LSTMFeatureExtractor) extractSequenceFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 32)

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		var sum, sqSum float64
		for _, v := range seq.VelocitySeq {
			sum += v
			sqSum += v * v
		}
		mean := sum / float64(len(seq.VelocitySeq))
		features[0] = mean / 100.0

		if len(seq.VelocitySeq) > 1 {
			variance := (sqSum / float64(len(seq.VelocitySeq))) - (mean * mean)
			features[1] = variance / 100.0
		}

		maxV := seq.VelocitySeq[0]
		minV := seq.VelocitySeq[0]
		for _, v := range seq.VelocitySeq {
			if v > maxV {
				maxV = v
			}
			if v < minV {
				minV = v
			}
		}
		features[2] = maxV / 100.0
		features[3] = minV / 100.0

		percentile25 := seq.VelocitySeq[len(seq.VelocitySeq)/4]
		percentile75 := seq.VelocitySeq[3*len(seq.VelocitySeq)/4]
		features[4] = percentile25 / 100.0
		features[5] = percentile75 / 100.0
		features[6] = (percentile75 - percentile25) / 100.0
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		var sum float64
		for _, a := range seq.AccelerationSeq {
			sum += math.Abs(a)
		}
		features[8] = (sum / float64(len(seq.AccelerationSeq))) / 1000.0

		maxA := seq.AccelerationSeq[0]
		for _, a := range seq.AccelerationSeq {
			if math.Abs(a) > math.Abs(maxA) {
				maxA = a
			}
		}
		features[9] = math.Abs(maxA) / 1000.0

		positiveCount := 0
		for _, a := range seq.AccelerationSeq {
			if a > 0 {
				positiveCount++
			}
		}
		features[10] = float64(positiveCount) / float64(len(seq.AccelerationSeq))
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 1 {
		directionChanges := 0
		for i := 1; i < len(seq.DirectionSeq); i++ {
			diff := math.Abs(seq.DirectionSeq[i] - seq.DirectionSeq[i-1])
			if diff > math.Pi {
				diff = 2*math.Pi - diff
			}
			if diff > 0.5 {
				directionChanges++
			}
		}
		features[12] = float64(directionChanges) / float64(len(seq.DirectionSeq))
	}

	if seq.NormalizedSeq != nil && len(seq.NormalizedSeq) > 0 {
		var totalCurvature float64
		count := 0
		for i := 1; i < len(seq.NormalizedSeq)-1; i++ {
			x1, y1 := seq.NormalizedSeq[i-1][0], seq.NormalizedSeq[i-1][1]
			x2, y2 := seq.NormalizedSeq[i][0], seq.NormalizedSeq[i][1]
			x3, y3 := seq.NormalizedSeq[i+1][0], seq.NormalizedSeq[i+1][1]

			dx1, dy1 := x2-x1, y2-y1
			dx2, dy2 := x3-x2, y3-y2

			dot := dx1*dx2 + dy1*dy2
			mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
			mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

			if mag1 > 0 && mag2 > 0 {
				cosAngle := dot / (mag1 * mag2)
				if cosAngle > -1 && cosAngle < 1 {
					curvature := math.Acos(cosAngle)
					totalCurvature += curvature
					count++
				}
			}
		}
		if count > 0 {
			features[16] = totalCurvature / float64(count)
		}
	}

	features[20] = float64(len(seq.Points)) / 200.0

	features[24] = 0.0
	features[25] = 0.0
	features[26] = 0.0
	features[27] = 0.0
	features[28] = 0.0
	features[29] = 0.0
	features[30] = 0.0
	features[31] = 0.0

	return features
}

func (e *LSTMFeatureExtractor) extractTemporalFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 32)

	if len(seq.Points) >= 2 {
		firstTime := seq.Points[0].Timestamp
		lastTime := seq.Points[len(seq.Points)-1].Timestamp
		totalDuration := float64(lastTime - firstTime)

		if totalDuration > 0 {
			intervals := make([]float64, len(seq.Points)-1)
			for i := 1; i < len(seq.Points); i++ {
				intervals[i-1] = float64(seq.Points[i].Timestamp - seq.Points[i-1].Timestamp)
			}

			var sum float64
			for _, t := range intervals {
				sum += t
			}
			meanInterval := sum / float64(len(intervals))
			features[0] = meanInterval / 100.0

			var variance float64
			for _, t := range intervals {
				diff := t - meanInterval
				variance += diff * diff
			}
			features[1] = (variance / float64(len(intervals))) / 1000.0

			maxInterval := intervals[0]
			minInterval := intervals[0]
			for _, t := range intervals {
				if t > maxInterval {
					maxInterval = t
				}
				if t < minInterval {
					minInterval = t
				}
			}
			features[2] = maxInterval / 100.0
			features[3] = minInterval / 100.0

			pauseCount := 0
			for _, t := range intervals {
				if t > 200 {
					pauseCount++
				}
			}
			features[4] = float64(pauseCount) / 5.0

			longPauseCount := 0
			for _, t := range intervals {
				if t > 500 {
					longPauseCount++
				}
			}
			features[5] = float64(longPauseCount)

			features[6] = meanInterval / 100.0
			if maxInterval > minInterval {
				features[7] = (maxInterval - minInterval) / meanInterval
			}
		}
	}

	features[8] = float64(len(seq.Points)) / 200.0

	features[16] = 0.0
	features[17] = 0.0
	features[18] = 0.0
	features[19] = 0.0
	features[20] = 0.0
	features[21] = 0.0
	features[22] = 0.0
	features[23] = 0.0
	features[24] = 0.0
	features[25] = 0.0
	features[26] = 0.0
	features[27] = 0.0
	features[28] = 0.0
	features[29] = 0.0
	features[30] = 0.0
	features[31] = 0.0

	return features
}

func (e *LSTMFeatureExtractor) computeAttention(forwardHidden, backwardHidden [][]float64) []float64 {
	combined := e.combineBidirectional(forwardHidden, backwardHidden)

	if combined == nil || len(combined) == 0 {
		return nil
	}

	if !e.useAttention {
		if len(combined) == 0 {
			return nil
		}
		return combined[len(combined)/2]
	}

	seqLen := len(combined)

	attentionScores := make([]float64, seqLen)
	for i := 0; i < seqLen; i++ {
		var score float64
		for j := 0; j < len(combined[0]); j++ {
			score += combined[i][j] * e.attentionWeights[0][0][j]
		}
		attentionScores[i] = math.Tanh(score)
	}

	sumExp := 0.0
	for i := range attentionScores {
		attentionScores[i] = math.Exp(attentionScores[i])
		sumExp += attentionScores[i]
	}

	if sumExp > 0 {
		for i := range attentionScores {
			attentionScores[i] /= sumExp
		}
	}

	attentionOutput := make([]float64, len(combined[0]))
	for i := range attentionOutput {
		for j := 0; j < seqLen; j++ {
			attentionOutput[i] += combined[j][i] * attentionScores[j]
		}
	}

	return attentionOutput
}

func (e *LSTMFeatureExtractor) combineBidirectional(forwardHidden, backwardHidden [][]float64) [][]float64 {
	if forwardHidden == nil && backwardHidden == nil {
		return nil
	}

	if forwardHidden == nil {
		return backwardHidden
	}
	if backwardHidden == nil {
		return forwardHidden
	}

	seqLen := len(forwardHidden)
	hiddenSize := len(forwardHidden[0])

	combined := make([][]float64, seqLen)
	for i := range combined {
		combined[i] = make([]float64, hiddenSize*2)
		for j := 0; j < hiddenSize; j++ {
			combined[i][j] = forwardHidden[i][j]
			if j < len(backwardHidden[i]) {
				combined[i][hiddenSize+j] = backwardHidden[i][j]
			}
		}
	}

	return combined
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

	return riskFeatures, nil
}

func (e *LSTMFeatureExtractor) computeLSTMEmbedding(features []float64) []float64 {
	embedding := make([]float64, LSTMFeatureDim)

	for i := range embedding {
		var sum float64
		for j, f := range features {
			weight := 0.1
			if j < len(e.outputWeights) {
				weight = e.outputWeights[j]
			}
			sum += f * weight
		}

		embedding[i] = math.Tanh(sum)
	}

	return embedding
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
			dot := v1x*v2x + v1y*v2y
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			curvatures[i-1] = math.Abs(cross) / (mag1 * mag2 + 1e-6)
			_ = angle
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
	for i := range data {
		fft[i] = data[i]
	}

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

	boxSizes := []float64{0.1, 0.2, 0.3, 0.4, 0.5}
	count := 0

	for _, boxSize := range boxSizes {
		counts := e.countBoxFractal(points, boxSize)
		if counts > 0 {
			count++
		}
	}

	if count < 2 {
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

	boxSet := make(map[int]bool)
	for _, p := range points {
		boxX := int((p[0] - minX) / boxSize)
		boxY := int((p[1] - minY) / boxSize)
		boxIndex := boxY*boxesX + boxX
		boxSet[boxIndex] = true
		_ = boxX
	}

	return len(boxSet)
}

func (e *LSTMFeatureExtractor) computeSpatialVariability(points []model.TracePoint) float64 {
	if len(points) < 2 {
		return 0
	}

	var sumX, sumY float64
	for _, p := range points {
		sumX += float64(p.X)
		sumY += float64(p.Y)
	}
	meanX := sumX / float64(len(points))
	meanY := sumY / float64(len(points))

	var variance float64
	for _, p := range points {
		dx := float64(p.X) - meanX
		dy := float64(p.Y) - meanY
		variance += dx*dx + dy*dy
	}

	return variance / float64(len(points))
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
		variance := 0.0
		mean := 0.0
		for _, v := range seq.VelocitySeq {
			mean += v
		}
		mean /= float64(len(seq.VelocitySeq))
		for _, v := range seq.VelocitySeq {
			diff := v - mean
			variance += diff * diff
		}
		variance /= float64(len(seq.VelocitySeq))

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
		var accelSum float64
		for _, a := range seq.AccelerationSeq {
			accelSum += math.Abs(a)
		}
		meanAccel := accelSum / float64(len(seq.AccelerationSeq))

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
		BasicCount:      48,
		SequenceCount:   32,
		TemporalCount:   32,
		EnhancedCount:   32,
		BehavioralCount: 16,
		ComplexityScore: complexity,
		IsHighRisk:      complexity < 0.3,
		AnomalyPatterns: anomalies,
	}

	return summary, nil
}
