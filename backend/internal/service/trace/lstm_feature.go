package trace

import (
	"errors"
	"math"
	"math/rand"

	"github.com/hjtpx/hjtpx/internal/model"
)

const (
	LSTMFeatureDim    = 64
	LSTMSequenceLen   = 100
	LSTMHiddenSize    = 128
	LSTMNumLayers     = 2
)

type LSTMFeatureExtractor struct {
	hiddenWeights [][]float64
	hiddenBias    []float64
	cellWeights   [][]float64
	cellBias      []float64
	outputWeights []float64
	featureMean   []float64
	featureStd    []float64
	isInitialized bool
}

type TrajectorySequence struct {
	Points          []model.TracePoint
	NormalizedSeq  [][]float64
	FeatureVector   []float64
	VelocitySeq     []float64
	AccelerationSeq []float64
	DirectionSeq    []float64
}

func NewLSTMFeatureExtractor() *LSTMFeatureExtractor {
	extractor := &LSTMFeatureExtractor{
		isInitialized: false,
	}
	extractor.initializeWeights()
	return extractor
}

func (e *LSTMFeatureExtractor) initializeWeights() {
	e.hiddenWeights = make([][]float64, LSTMFeatureDim)
	for i := range e.hiddenWeights {
		e.hiddenWeights[i] = e.initXavier(LSTMFeatureDim)
	}

	e.hiddenBias = make([]float64, LSTMFeatureDim*4)
	for i := range e.hiddenBias {
		e.hiddenBias[i] = 0.0
	}

	e.cellWeights = make([][]float64, LSTMFeatureDim)
	for i := range e.cellWeights {
		e.cellWeights[i] = e.initXavier(LSTMFeatureDim)
	}

	e.cellBias = make([]float64, LSTMFeatureDim)
	for i := range e.cellBias {
		e.cellBias[i] = 0.0
	}

	e.outputWeights = make([]float64, LSTMFeatureDim)
	for i := range e.outputWeights {
		e.outputWeights[i] = 0.1
	}

	e.featureMean = []float64{
		0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
		0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0,
	}
	e.featureStd = []float64{
		100.0, 100.0, 100.0, 100.0, 100.0, 100.0, 100.0, 100.0,
		50.0, 50.0, 50.0, 50.0, 50.0, 50.0, 50.0, 50.0,
	}

	e.isInitialized = true
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
	features := make([]float64, 16)

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

	return features
}

func (e *LSTMFeatureExtractor) extractSequenceFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 16)

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
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		var sum float64
		for _, a := range seq.AccelerationSeq {
			sum += math.Abs(a)
		}
		features[4] = (sum / float64(len(seq.AccelerationSeq))) / 1000.0

		maxA := seq.AccelerationSeq[0]
		for _, a := range seq.AccelerationSeq {
			if math.Abs(a) > math.Abs(maxA) {
				maxA = a
			}
		}
		features[5] = math.Abs(maxA) / 1000.0
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 1 {
		directionChanges := 0
		for i := 1; i < len(seq.DirectionSeq); i++ {
			diff := math.Abs(seq.DirectionSeq[i] - seq.DirectionSeq[i-1])
			if diff > 0.5 {
				directionChanges++
			}
		}
		features[6] = float64(directionChanges) / float64(len(seq.DirectionSeq))
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
			features[7] = totalCurvature / float64(count)
		}
	}

	features[8] = float64(len(seq.Points)) / 200.0

	features[9] = 0.0
	features[10] = 0.0
	features[11] = 0.0
	features[12] = 0.0
	features[13] = 0.0
	features[14] = 0.0
	features[15] = 0.0

	return features
}

func (e *LSTMFeatureExtractor) extractTemporalFeatures(seq *TrajectorySequence) []float64 {
	features := make([]float64, 16)

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
		}
	}

	features[5] = float64(len(seq.Points)) / 200.0

	features[6] = 0.0
	features[7] = 0.0
	features[8] = 0.0
	features[9] = 0.0
	features[10] = 0.0
	features[11] = 0.0
	features[12] = 0.0
	features[13] = 0.0
	features[14] = 0.0
	features[15] = 0.0

	return features
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
	riskFeatures["direction_change_rate"] = features[6]
	riskFeatures["curvature_mean"] = features[7]
	riskFeatures["temporal_interval_mean"] = features[16]
	riskFeatures["temporal_interval_variance"] = features[17]
	riskFeatures["pause_ratio"] = features[20]

	riskFeatures["embedding_norm"] = 0.0
	for _, f := range features {
		riskFeatures["embedding_norm"] += f * f
	}
	riskFeatures["embedding_norm"] = math.Sqrt(riskFeatures["embedding_norm"])

	return riskFeatures, nil
}
