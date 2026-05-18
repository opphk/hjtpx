package trace

import (
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

const (
	TransformerDim            = 128
	EnhancedTransformerHeads  = 16
	TransformerKeyDim         = TransformerDim / EnhancedTransformerHeads
	TransformerFFDim          = 512
	EnhancedTransformerLayers = 8
	MaxSequenceLen            = 200
	TransformerDropoutRate    = 0.1
)

type TransformerPredictor struct {
	mu                  sync.RWMutex
	queryWeights        [][][]float64
	keyWeights          [][][]float64
	valueWeights        [][][]float64
	outputWeights       [][][]float64
	ffWeights1          [][][]float64
	ffWeights2          [][][]float64
	positionalEnc       []float64
	layerNorms1Gamma    [][]float64
	layerNorms1Beta     [][]float64
	layerNorms2Gamma    [][]float64
	layerNorms2Beta     [][]float64
	predictionHead      []float64
	predictionBias      float64
	isInitialized       bool
	useLayerNorm        bool
	useDropout          bool
	dropoutRate         float64
	quantizationEnabled bool
	quantizedWeights    map[string][]int8
	scaleFactors        map[string]float64
	attentionOutput     [][][]float64
	causalMask          [][]float64
}

type AttentionOutput struct {
	Output    [][]float64
	Attention [][][]float64
}

type TransformerLayer struct {
	queryWeights  [][]float64
	keyWeights    [][]float64
	valueWeights  [][]float64
	outputWeights [][]float64
	ffWeights1    [][]float64
	ffWeights2    [][]float64
	layerNorm1Gamma []float64
	layerNorm1Beta  []float64
	layerNorm2Gamma []float64
	layerNorm2Beta  []float64
}

type TransformerRiskPrediction struct {
	RiskScore          float64
	BotProbability     float64
	HumanProbability   float64
	Confidence         float64
	FeatureImportance  map[string]float64
	AttentionAnalysis  map[string]float64
	SequenceComplexity float64
}

func NewTransformerPredictor() *TransformerPredictor {
	predictor := &TransformerPredictor{
		isInitialized:       false,
		useLayerNorm:        true,
		useDropout:          true,
		dropoutRate:         TransformerDropoutRate,
		quantizationEnabled: false,
		quantizedWeights:    make(map[string][]int8),
		scaleFactors:        make(map[string]float64),
	}
	predictor.initializeWeights()
	return predictor
}

func (t *TransformerPredictor) initializeWeights() {
	t.queryWeights = make([][][]float64, EnhancedTransformerLayers)
	t.keyWeights = make([][][]float64, EnhancedTransformerLayers)
	t.valueWeights = make([][][]float64, EnhancedTransformerLayers)
	t.outputWeights = make([][][]float64, EnhancedTransformerLayers)

	for layer := 0; layer < EnhancedTransformerLayers; layer++ {
		t.queryWeights[layer] = t.initLayerWeights(TransformerDim, TransformerDim)
		t.keyWeights[layer] = t.initLayerWeights(TransformerDim, TransformerDim)
		t.valueWeights[layer] = t.initLayerWeights(TransformerDim, TransformerDim)
		t.outputWeights[layer] = t.initLayerWeights(TransformerDim, TransformerDim)
	}

	t.ffWeights1 = make([][][]float64, EnhancedTransformerLayers)
	t.ffWeights2 = make([][][]float64, EnhancedTransformerLayers)

	for layer := 0; layer < EnhancedTransformerLayers; layer++ {
		t.ffWeights1[layer] = t.initLayerWeights(TransformerFFDim, TransformerDim)
		t.ffWeights2[layer] = t.initLayerWeights(TransformerDim, TransformerFFDim)
	}

	t.layerNorms1Gamma = make([][]float64, EnhancedTransformerLayers)
	t.layerNorms1Beta = make([][]float64, EnhancedTransformerLayers)
	t.layerNorms2Gamma = make([][]float64, EnhancedTransformerLayers)
	t.layerNorms2Beta = make([][]float64, EnhancedTransformerLayers)

	for layer := 0; layer < EnhancedTransformerLayers; layer++ {
		t.layerNorms1Gamma[layer] = make([]float64, TransformerDim)
		t.layerNorms1Beta[layer] = make([]float64, TransformerDim)
		t.layerNorms2Gamma[layer] = make([]float64, TransformerDim)
		t.layerNorms2Beta[layer] = make([]float64, TransformerDim)
		for i := range t.layerNorms1Gamma[layer] {
			t.layerNorms1Gamma[layer][i] = 1.0
			t.layerNorms1Beta[layer][i] = 0.0
			t.layerNorms2Gamma[layer][i] = 1.0
			t.layerNorms2Beta[layer][i] = 0.0
		}
	}

	t.predictionHead = t.initVector(TransformerDim)
	t.predictionBias = 0.0

	t.positionalEnc = make([]float64, MaxSequenceLen*TransformerDim)
	for i := 0; i < MaxSequenceLen; i++ {
		for j := 0; j < TransformerDim; j++ {
			if j%2 == 0 {
				t.positionalEnc[i*TransformerDim+j] = math.Sin(float64(i) / math.Pow(10000, float64(2*j)/float64(TransformerDim)))
			} else {
				t.positionalEnc[i*TransformerDim+j] = math.Cos(float64(i) / math.Pow(10000, float64(2*j)/float64(TransformerDim)))
			}
		}
	}

	t.causalMask = t.createCausalMask(MaxSequenceLen)

	t.isInitialized = true
}

func (t *TransformerPredictor) createCausalMask(size int) [][]float64 {
	mask := make([][]float64, size)
	for i := range mask {
		mask[i] = make([]float64, size)
		for j := range mask[i] {
			if j > i {
				mask[i][j] = -1e9
			} else {
				mask[i][j] = 0.0
			}
		}
	}
	return mask
}

func (t *TransformerPredictor) initLayerWeights(outDim, inDim int) [][]float64 {
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

func (t *TransformerPredictor) initVector(dim int) []float64 {
	vec := make([]float64, dim)
	for i := range vec {
		vec[i] = (rand.Float64() - 0.5) * 0.1
	}
	return vec
}

func (t *TransformerPredictor) Predict(seq *TrajectorySequence) (*TransformerRiskPrediction, error) {
	if !t.isInitialized {
		t.initializeWeights()
	}

	embeddings := t.encodeSequence(seq)

	transformerOutput, attentionMaps := t.applyTransformer(embeddings)

	prediction := t.decodePrediction(transformerOutput)

	prediction.AttentionAnalysis = t.analyzeAttentionPatterns(attentionMaps, seq)
	prediction.SequenceComplexity = t.computeSequenceComplexity(seq)

	return prediction, nil
}

func (t *TransformerPredictor) encodeSequence(seq *TrajectorySequence) [][]float64 {
	seqLen := len(seq.Points)
	if seqLen > MaxSequenceLen {
		seqLen = MaxSequenceLen
	}

	embeddings := make([][]float64, seqLen)
	for i := 0; i < seqLen; i++ {
		embeddings[i] = make([]float64, TransformerDim)
	}

	for i := 0; i < seqLen; i++ {
		featureIdx := 0

		if i < len(seq.NormalizedSeq) {
			norm := seq.NormalizedSeq[i]
			for j := 0; j < len(norm) && j < TransformerDim; j++ {
				embeddings[i][featureIdx] = norm[j] * 2.0
				featureIdx++
			}
		}

		if i < len(seq.VelocitySeq) {
			embeddings[i][featureIdx] = seq.VelocitySeq[i] / 100.0
			featureIdx++
		}
		if i < len(seq.AccelerationSeq) {
			embeddings[i][featureIdx] = seq.AccelerationSeq[i] / 1000.0
			featureIdx++
		}
		if i < len(seq.DirectionSeq) {
			embeddings[i][featureIdx] = seq.DirectionSeq[i] / math.Pi
			featureIdx++
		}
		if i < len(seq.JerkSeq) {
			embeddings[i][featureIdx] = seq.JerkSeq[i] / 10000.0
			featureIdx++
		}
		if i < len(seq.CurvatureSeq) {
			embeddings[i][featureIdx] = seq.CurvatureSeq[i]
			featureIdx++
		}

		for j := 0; j < TransformerDim && j*TransformerDim+i < len(t.positionalEnc); j++ {
			embeddings[i][j] += t.positionalEnc[i*TransformerDim+j] * 0.1
		}
	}

	return embeddings
}

func (t *TransformerPredictor) applyTransformer(embeddings [][]float64) ([][]float64, [][][][]float64) {
	output := embeddings
	attentionMaps := make([][][][]float64, EnhancedTransformerLayers)

	for layer := 0; layer < EnhancedTransformerLayers; layer++ {
		attnOutput, attnWeights := t.multiHeadAttention(output, layer)
		attentionMaps[layer] = attnWeights

		residual := t.layerNorm(addVectors(output, attnOutput), t.layerNorms1Gamma[layer], t.layerNorms1Beta[layer])

		if t.useDropout {
			residual = t.applyDropout(residual)
		}

		ffOutput := t.feedForward(residual, layer)

		output = t.layerNorm(addVectors(residual, ffOutput), t.layerNorms2Gamma[layer], t.layerNorms2Beta[layer])

		if t.useDropout {
			output = t.applyDropout(output)
		}
	}

	return output, attentionMaps
}

func (t *TransformerPredictor) applyDropout(x [][]float64) [][]float64 {
	result := make([][]float64, len(x))
	for i := range x {
		result[i] = make([]float64, len(x[i]))
		for j := range x[i] {
			if rand.Float64() > t.dropoutRate {
				result[i][j] = x[i][j] / (1 - t.dropoutRate)
			} else {
				result[i][j] = 0
			}
		}
	}
	return result
}

func (t *TransformerPredictor) multiHeadAttention(x [][]float64, layer int) ([][]float64, [][][]float64) {
	seqLen := len(x)
	dim := TransformerDim

	Q := t.matMul(x, t.queryWeights[layer])
	K := t.matMul(x, t.keyWeights[layer])
	V := t.matMul(x, t.valueWeights[layer])

	headSize := dim / EnhancedTransformerHeads
	heads := make([][][]float64, EnhancedTransformerHeads)
	attentionMaps := make([][][]float64, EnhancedTransformerHeads)

	for h := 0; h < EnhancedTransformerHeads; h++ {
		qHead := make([][]float64, seqLen)
		kHead := make([][]float64, seqLen)
		vHead := make([][]float64, seqLen)

		for i := range qHead {
			qHead[i] = Q[i][h*headSize : (h+1)*headSize]
			kHead[i] = K[i][h*headSize : (h+1)*headSize]
			vHead[i] = V[i][h*headSize : (h+1)*headSize]
		}

		scale := math.Sqrt(float64(headSize))
		attnScores := t.matMul(qHead, t.transpose(kHead))

		for i := range attnScores {
			for j := range attnScores[i] {
				attnScores[i][j] /= scale
				if j > i {
					attnScores[i][j] += -1e9
				}
			}
		}

		attnWeights := t.softmax2D(attnScores)
		attentionMaps[h] = attnWeights

		heads[h] = t.matMul(attnWeights, vHead)
	}

	output := make([][]float64, seqLen)
	for i := range output {
		output[i] = make([]float64, dim)
		for h := 0; h < EnhancedTransformerHeads; h++ {
			copy(output[i][h*headSize:(h+1)*headSize], heads[h][i])
		}
	}

	proj := t.matMul(output, t.outputWeights[layer])
	for i := range proj {
		for j := range proj[i] {
			output[i][j] += proj[i][j]
		}
	}

	return output, attentionMaps
}

func (t *TransformerPredictor) feedForward(x [][]float64, layer int) [][]float64 {
	hidden := t.matMul(x, t.ffWeights1[layer])
	for i := range hidden {
		for j := range hidden[i] {
			hidden[i][j] = t.gelu(hidden[i][j])
		}
	}

	output := t.matMul(hidden, t.ffWeights2[layer])

	return output
}

func (t *TransformerPredictor) gelu(x float64) float64 {
	return 0.5 * x * (1 + math.Tanh(math.Sqrt(2/math.Pi)*(x+0.044715*math.Pow(x, 3))))
}

func (t *TransformerPredictor) layerNorm(x [][]float64, gamma, beta []float64) [][]float64 {
	if len(x) == 0 || len(x[0]) == 0 {
		return x
	}

	seqLen := len(x)
	dim := len(x[0])

	normalized := make([][]float64, seqLen)
	for i := range normalized {
		normalized[i] = make([]float64, dim)
	}

	for i := 0; i < seqLen; i++ {
		mean := 0.0
		for j := 0; j < dim; j++ {
			mean += x[i][j]
		}
		mean /= float64(dim)

		var varSum float64
		for j := 0; j < dim; j++ {
			diff := x[i][j] - mean
			varSum += diff * diff
		}
		std := math.Sqrt(varSum/float64(dim) + 1e-8)

		for j := 0; j < dim; j++ {
			if j < len(gamma) && j < len(beta) {
				normalized[i][j] = (x[i][j]-mean)/std*gamma[j] + beta[j]
			} else {
				normalized[i][j] = (x[i][j] - mean) / std
			}
		}
	}

	return normalized
}

func (t *TransformerPredictor) matMul(a [][]float64, b [][]float64) [][]float64 {
	if len(a) == 0 || len(b) == 0 {
		return nil
	}

	rowsA := len(a)
	colsA := len(a[0])
	colsB := len(b[0])

	result := make([][]float64, rowsA)
	for i := range result {
		result[i] = make([]float64, colsB)
		for j := range result[i] {
			var sum float64
			for k := 0; k < colsA; k++ {
				sum += a[i][k] * b[k][j]
			}
			result[i][j] = sum
		}
	}

	return result
}

func (t *TransformerPredictor) transpose(a [][]float64) [][]float64 {
	if len(a) == 0 {
		return a
	}

	rows := len(a)
	cols := len(a[0])

	result := make([][]float64, cols)
	for i := range result {
		result[i] = make([]float64, rows)
		for j := range result[i] {
			result[i][j] = a[j][i]
		}
	}

	return result
}

func (t *TransformerPredictor) softmax2D(x [][]float64) [][]float64 {
	rows := len(x)
	if rows == 0 {
		return x
	}
	cols := len(x[0])

	result := make([][]float64, rows)
	for i := range result {
		result[i] = make([]float64, cols)

		maxVal := x[i][0]
		for j := 1; j < cols; j++ {
			if x[i][j] > maxVal {
				maxVal = x[i][j]
			}
		}

		var sum float64
		for j := 0; j < cols; j++ {
			result[i][j] = math.Exp(x[i][j] - maxVal)
			sum += result[i][j]
		}

		for j := 0; j < cols; j++ {
			result[i][j] /= sum
		}
	}

	return result
}

func addVectors(a, b [][]float64) [][]float64 {
	if len(a) != len(b) {
		return a
	}

	result := make([][]float64, len(a))
	for i := range result {
		result[i] = make([]float64, len(a[i]))
		for j := range result[i] {
			result[i][j] = a[i][j] + b[i][j]
		}
	}

	return result
}

func (t *TransformerPredictor) decodePrediction(transformerOutput [][]float64) *TransformerRiskPrediction {
	if len(transformerOutput) == 0 {
		return &TransformerRiskPrediction{
			RiskScore:          0.5,
			BotProbability:     0.5,
			HumanProbability:   0.5,
			Confidence:         0.0,
			FeatureImportance:  make(map[string]float64),
			AttentionAnalysis:  make(map[string]float64),
			SequenceComplexity: 0.5,
		}
	}

	seqLen := len(transformerOutput)
	lastHidden := transformerOutput[seqLen-1]

	var pooledOutput float64
	for i := range lastHidden {
		pooledOutput += lastHidden[i] * t.predictionHead[i]
	}
	pooledOutput += t.predictionBias
	pooledOutput /= float64(TransformerDim)

	riskScore := (math.Tanh(pooledOutput) + 1.0) / 2.0

	botLogit := pooledOutput
	botProb := 1.0 / (1.0 + math.Exp(-botLogit))
	humanProb := 1.0 - botProb

	confidence := math.Max(0.5, 1.0-math.Abs(pooledOutput))

	var velocityVar, accelVar, directionChange, jerkVar, curvatureVar float64
	for i := range transformerOutput {
		if i < len(transformerOutput) {
			if len(transformerOutput[i]) > 3 {
				velocityVar += math.Abs(transformerOutput[i][3])
			}
			if len(transformerOutput[i]) > 4 {
				accelVar += math.Abs(transformerOutput[i][4])
			}
			if len(transformerOutput[i]) > 5 {
				directionChange += math.Abs(transformerOutput[i][5])
			}
			if len(transformerOutput[i]) > 6 {
				jerkVar += math.Abs(transformerOutput[i][6])
			}
			if len(transformerOutput[i]) > 7 {
				curvatureVar += math.Abs(transformerOutput[i][7])
			}
		}
	}

	featureImportance := make(map[string]float64)
	if seqLen > 0 {
		featureImportance["velocity_pattern"] = velocityVar / float64(seqLen)
		featureImportance["acceleration_pattern"] = accelVar / float64(seqLen)
		featureImportance["direction_pattern"] = directionChange / float64(seqLen)
		featureImportance["jerk_pattern"] = jerkVar / float64(seqLen)
		featureImportance["curvature_pattern"] = curvatureVar / float64(seqLen)
		featureImportance["sequence_length"] = float64(seqLen) / float64(MaxSequenceLen)
	}

	return &TransformerRiskPrediction{
		RiskScore:          riskScore,
		BotProbability:     botProb,
		HumanProbability:   humanProb,
		Confidence:         confidence,
		FeatureImportance:  featureImportance,
		AttentionAnalysis:  make(map[string]float64),
		SequenceComplexity: 0.0,
	}
}

func (t *TransformerPredictor) analyzeAttentionPatterns(attentionMaps [][][][]float64, seq *TrajectorySequence) map[string]float64 {
	analysis := make(map[string]float64)

	if len(attentionMaps) == 0 {
		return analysis
	}

	var totalAttention, maxAttention, minAttention float64
	count := 0

	for _, layerMaps := range attentionMaps {
		for _, headMaps := range layerMaps {
			for _, row := range headMaps {
				for _, val := range row {
					totalAttention += val
					if val > maxAttention {
						maxAttention = val
					}
					if val < minAttention || count == 0 {
						minAttention = val
					}
					count++
				}
			}
		}
	}

	if count > 0 {
		analysis["average_attention"] = totalAttention / float64(count)
		analysis["max_attention"] = maxAttention
		analysis["min_attention"] = minAttention
		analysis["attention_range"] = maxAttention - minAttention
	}

	analysis["num_layers"] = float64(EnhancedTransformerLayers)
	analysis["num_heads"] = float64(EnhancedTransformerHeads)
	analysis["sequence_length"] = float64(len(seq.Points))

	return analysis
}

func (t *TransformerPredictor) computeSequenceComplexity(seq *TrajectorySequence) float64 {
	complexity := 0.0
	weightSum := 0.0

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		var variance float64
		mean := 0.0
		for _, v := range seq.VelocitySeq {
			mean += v
		}
		mean /= float64(len(seq.VelocitySeq))
		for _, v := range seq.VelocitySeq {
			variance += (v - mean) * (v - mean)
		}
		variance /= float64(len(seq.VelocitySeq))
		complexity += variance * 0.2
		weightSum += 0.2
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 0 {
		changes := 0
		for i := 1; i < len(seq.DirectionSeq); i++ {
			diff := math.Abs(seq.DirectionSeq[i] - seq.DirectionSeq[i-1])
			if diff > math.Pi {
				diff = 2*math.Pi - diff
			}
			if diff > 0.3 {
				changes++
			}
		}
		complexity += float64(changes) / float64(len(seq.DirectionSeq)) * 0.3
		weightSum += 0.3
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		var maxAbs float64
		for _, a := range seq.AccelerationSeq {
			if math.Abs(a) > maxAbs {
				maxAbs = math.Abs(a)
			}
		}
		complexity += maxAbs / 1000.0 * 0.3
		weightSum += 0.3
	}

	if seq.CurvatureSeq != nil && len(seq.CurvatureSeq) > 0 {
		var meanCurv float64
		for _, c := range seq.CurvatureSeq {
			meanCurv += c
		}
		meanCurv /= float64(len(seq.CurvatureSeq))
		complexity += meanCurv * 0.2
		weightSum += 0.2
	}

	if weightSum > 0 {
		complexity /= weightSum
	}

	return math.Min(1.0, math.Max(0.0, complexity))
}

func (t *TransformerPredictor) PredictWithFeatures(features []float64) (*TransformerRiskPrediction, error) {
	if !t.isInitialized {
		t.initializeWeights()
	}

	var pooledOutput float64
	for i := range features {
		if i < len(t.predictionHead) {
			pooledOutput += features[i] * t.predictionHead[i]
		}
	}
	pooledOutput += t.predictionBias
	pooledOutput /= float64(len(features))

	riskScore := (math.Tanh(pooledOutput) + 1.0) / 2.0

	botLogit := pooledOutput
	botProb := 1.0 / (1.0 + math.Exp(-botLogit))
	humanProb := 1.0 - botProb

	confidence := math.Max(0.5, 1.0-math.Abs(pooledOutput))

	featureImportance := make(map[string]float64)
	featureImportance["total_features"] = float64(len(features))

	return &TransformerRiskPrediction{
		RiskScore:          riskScore,
		BotProbability:     botProb,
		HumanProbability:   humanProb,
		Confidence:         confidence,
		FeatureImportance:  featureImportance,
		AttentionAnalysis:  make(map[string]float64),
		SequenceComplexity: 0.0,
	}, nil
}

func (t *TransformerPredictor) PredictTrajectory(traceData *model.TraceData) (*TransformerRiskPrediction, error) {
	seq, err := t.encodeToSequence(traceData)
	if err != nil {
		return nil, err
	}

	return t.Predict(seq)
}

func (t *TransformerPredictor) encodeToSequence(traceData *model.TraceData) (*TrajectorySequence, error) {
	extractor := NewLSTMFeatureExtractor()
	return extractor.PrepareSequence(traceData)
}

func (t *TransformerPredictor) LoadModelWeights(weightsPath string) error {
	if !t.isInitialized {
		t.initializeWeights()
	}
	return nil
}

func (t *TransformerPredictor) GetEmbeddingDimension() int {
	return TransformerDim
}

func (t *TransformerPredictor) GetAttentionHeads() int {
	return EnhancedTransformerHeads
}

func (t *TransformerPredictor) GetModelArchitecture() map[string]interface{} {
	return map[string]interface{}{
		"model_type":          "Transformer",
		"embedding_dim":       TransformerDim,
		"num_attention_heads": EnhancedTransformerHeads,
		"attention_head_dim":  TransformerKeyDim,
		"num_layers":          EnhancedTransformerLayers,
		"feed_forward_dim":    TransformerFFDim,
		"max_sequence_length": MaxSequenceLen,
		"use_layer_norm":      t.useLayerNorm,
		"use_dropout":         t.useDropout,
		"dropout_rate":        t.dropoutRate,
	}
}

func (t *TransformerPredictor) EnableQuantization(enabled bool) {
	t.quantizationEnabled = enabled
	if enabled {
		t.quantizeWeights()
	}
}

func (t *TransformerPredictor) quantizeWeights() {
	t.quantizedWeights = make(map[string][]int8)
	t.scaleFactors = make(map[string]float64)

	t.quantizeAndStore("query_weights", t.queryWeights)
	t.quantizeAndStore("key_weights", t.keyWeights)
	t.quantizeAndStore("value_weights", t.valueWeights)
	t.quantizeAndStore("output_weights", t.outputWeights)
	t.quantizeAndStore("ff_weights1", t.ffWeights1)
	t.quantizeAndStore("ff_weights2", t.ffWeights2)
}

func (t *TransformerPredictor) quantizeAndStore(name string, weights [][][]float64) {
	flat := make([]float64, 0)
	for _, layer := range weights {
		for _, row := range layer {
			flat = append(flat, row...)
		}
	}
	t.quantizedWeights[name], t.scaleFactors[name] = t.quantizeArray(flat)
}

func (t *TransformerPredictor) quantizeArray(data []float64) ([]int8, float64) {
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

func (t *TransformerPredictor) GetMemoryUsageBytes() int64 {
	total := int64(0)
	
	if t.quantizationEnabled {
		for _, weights := range t.quantizedWeights {
			total += int64(len(weights))
		}
	} else {
		for _, layer := range t.queryWeights {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, layer := range t.keyWeights {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, layer := range t.valueWeights {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, layer := range t.outputWeights {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, layer := range t.ffWeights1 {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
		for _, layer := range t.ffWeights2 {
			for _, row := range layer {
				total += int64(len(row)) * 8
			}
		}
	}
	
	return total
}

func (t *TransformerPredictor) AnalyzeAttentionPatterns(embeddings [][]float64) (map[string]float64, error) {
	attentionPatterns := make(map[string]float64)

	if len(embeddings) == 0 {
		return attentionPatterns, nil
	}

	_, attentionMaps := t.applyTransformer(embeddings)

	return t.analyzeAttentionPatterns(attentionMaps, &TrajectorySequence{Points: make([]model.TracePoint, len(embeddings))}), nil
}

func (t *TransformerPredictor) ComputeAttentionEntropy(attentionMaps [][][]float64) float64 {
	if len(attentionMaps) == 0 {
		return 0
	}

	var totalEntropy float64
	count := 0

	for _, headMap := range attentionMaps {
		for _, row := range headMap {
			entropy := 0.0
			for _, prob := range row {
				if prob > 0 {
					entropy -= prob * math.Log2(prob)
				}
			}
			totalEntropy += entropy
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalEntropy / float64(count)
}

type BehaviorPatternType string

const (
	BehaviorPatternNormal         BehaviorPatternType = "normal"
	BehaviorPatternSuspicious     BehaviorPatternType = "suspicious"
	BehaviorPatternBotLike        BehaviorPatternType = "bot_like"
	BehaviorPatternHumanLike      BehaviorPatternType = "human_like"
	BehaviorPatternRepeated       BehaviorPatternType = "repeated"
	BehaviorPatternAnomalous      BehaviorPatternType = "anomalous"
)

type BehaviorClassificationResult struct {
	PatternType        BehaviorPatternType `json:"pattern_type"`
	PatternConfidence  float64            `json:"pattern_confidence"`
	SubPatterns        []string           `json:"sub_patterns"`
	AnomalyScore       float64            `json:"anomaly_score"`
	ConsistencyScore   float64            `json:"consistency_score"`
	ComplexityScore    float64            `json:"complexity_score"`
	VelocityAnalysis   map[string]float64 `json:"velocity_analysis"`
	DirectionAnalysis  map[string]float64 `json:"direction_analysis"`
}

type IntentPrediction struct {
	IntentType         string                 `json:"intent_type"`
	Confidence         float64                `json:"confidence"`
	SubIntents         []IntentPrediction     `json:"sub_intents"`
	ExpectedNextAction string                 `json:"expected_next_action"`
	ActionProbabilities map[string]float64    `json:"action_probabilities"`
}

type SequenceAnomaly struct {
	AnomalyType        string  `json:"anomaly_type"`
	Position           int     `json:"position"`
	Severity           float64 `json:"severity"`
	Description        string  `json:"description"`
	SuggestedAction    string  `json:"suggested_action"`
}

type ComprehensiveBehaviorResult struct {
	RiskPrediction       *TransformerRiskPrediction `json:"risk_prediction"`
	BehaviorClassification *BehaviorClassificationResult `json:"behavior_classification"`
	IntentPrediction     *IntentPrediction         `json:"intent_prediction"`
	Anomalies           []SequenceAnomaly         `json:"anomalies"`
	ExecutionTime       time.Duration             `json:"execution_time_ms"`
}

func (t *TransformerPredictor) ClassifyBehavior(seq *TrajectorySequence) *BehaviorClassificationResult {
	result := &BehaviorClassificationResult{
		SubPatterns:      make([]string, 0),
		VelocityAnalysis: make(map[string]float64),
		DirectionAnalysis: make(map[string]float64),
	}

	complexity := t.computeSequenceComplexity(seq)
	result.ComplexityScore = complexity

	consistency := t.computeSequenceConsistency(seq)
	result.ConsistencyScore = consistency

	anomalyScore := t.detectAnomalyScore(seq)
	result.AnomalyScore = anomalyScore

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		result.VelocityAnalysis = t.analyzeVelocityPattern(seq.VelocitySeq)
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 0 {
		result.DirectionAnalysis = t.analyzeDirectionPattern(seq.DirectionSeq)
	}

	result.PatternType, result.PatternConfidence = t.determinePatternType(result)

	if result.PatternType == BehaviorPatternAnomalous {
		result.SubPatterns = append(result.SubPatterns, "unusual_pattern_detected")
	}
	if anomalyScore > 0.7 {
		result.SubPatterns = append(result.SubPatterns, "high_anomaly_score")
	}
	if consistency < 0.3 {
		result.SubPatterns = append(result.SubPatterns, "inconsistent_behavior")
	}

	return result
}

func (t *TransformerPredictor) computeSequenceConsistency(seq *TrajectorySequence) float64 {
	if seq.VelocitySeq == nil || len(seq.VelocitySeq) < 2 {
		return 0.5
	}

	var diffSum float64
	for i := 1; i < len(seq.VelocitySeq); i++ {
		diffSum += math.Abs(seq.VelocitySeq[i] - seq.VelocitySeq[i-1])
	}
	avgDiff := diffSum / float64(len(seq.VelocitySeq)-1)

	normalizedDiff := math.Min(1.0, avgDiff/50.0)
	return math.Max(0.0, 1.0-normalizedDiff)
}

func (t *TransformerPredictor) detectAnomalyScore(seq *TrajectorySequence) float64 {
	score := 0.0
	weightSum := 0.0

	if seq.VelocitySeq != nil && len(seq.VelocitySeq) > 0 {
		maxVel := 0.0
		for _, v := range seq.VelocitySeq {
			if v > maxVel {
				maxVel = v
			}
		}
		score += math.Min(1.0, maxVel/200.0) * 0.3
		weightSum += 0.3
	}

	if seq.AccelerationSeq != nil && len(seq.AccelerationSeq) > 0 {
		maxAcc := 0.0
		for _, a := range seq.AccelerationSeq {
			if math.Abs(a) > maxAcc {
				maxAcc = math.Abs(a)
			}
		}
		score += math.Min(1.0, maxAcc/500.0) * 0.3
		weightSum += 0.3
	}

	if seq.JerkSeq != nil && len(seq.JerkSeq) > 0 {
		maxJerk := 0.0
		for _, j := range seq.JerkSeq {
			if math.Abs(j) > maxJerk {
				maxJerk = math.Abs(j)
			}
		}
		score += math.Min(1.0, maxJerk/1000.0) * 0.2
		weightSum += 0.2
	}

	complexity := t.computeSequenceComplexity(seq)
	score += (1.0 - complexity) * 0.2
	weightSum += 0.2

	if weightSum > 0 {
		score /= weightSum
	}

	return score
}

func (t *TransformerPredictor) analyzeVelocityPattern(velocities []float64) map[string]float64 {
	result := make(map[string]float64)

	if len(velocities) == 0 {
		return result
	}

	sum := 0.0
	maxVel := 0.0
	minVel := float64(math.MaxFloat64)

	for _, v := range velocities {
		sum += v
		if v > maxVel {
			maxVel = v
		}
		if v < minVel {
			minVel = v
		}
	}

	result["mean"] = sum / float64(len(velocities))
	result["max"] = maxVel
	result["min"] = minVel
	result["range"] = maxVel - minVel

	var variance float64
	mean := result["mean"]
	for _, v := range velocities {
		variance += (v - mean) * (v - mean)
	}
	result["variance"] = variance / float64(len(velocities))
	result["std_dev"] = math.Sqrt(result["variance"])

	return result
}

func (t *TransformerPredictor) analyzeDirectionPattern(directions []float64) map[string]float64 {
	result := make(map[string]float64)

	if len(directions) == 0 {
		return result
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

	result["direction_changes"] = float64(changes)
	result["change_rate"] = float64(changes) / float64(len(directions)-1)

	sum := 0.0
	for _, d := range directions {
		sum += d
	}
	result["mean_direction"] = sum / float64(len(directions))

	return result
}

func (t *TransformerPredictor) determinePatternType(result *BehaviorClassificationResult) (BehaviorPatternType, float64) {
	anomalyScore := result.AnomalyScore
	consistency := result.ConsistencyScore
	complexity := result.ComplexityScore

	if anomalyScore > 0.8 {
		return BehaviorPatternAnomalous, anomalyScore
	}

	if anomalyScore > 0.6 && complexity < 0.3 {
		return BehaviorPatternBotLike, (anomalyScore + (1.0 - complexity)) / 2.0
	}

	if anomalyScore > 0.5 && consistency < 0.4 {
		return BehaviorPatternSuspicious, (anomalyScore + (1.0 - consistency)) / 2.0
	}

	if complexity > 0.6 && consistency > 0.6 {
		return BehaviorPatternHumanLike, (complexity + consistency) / 2.0
	}

	if result.VelocityAnalysis["variance"] < 5.0 {
		return BehaviorPatternRepeated, 0.7
	}

	return BehaviorPatternNormal, 0.5 + complexity*0.3 + consistency*0.2
}

func (t *TransformerPredictor) PredictIntent(seq *TrajectorySequence) *IntentPrediction {
	velAnalysis := t.analyzeVelocityPattern(seq.VelocitySeq)
	dirAnalysis := t.analyzeDirectionPattern(seq.DirectionSeq)

	intentType := "unknown"
	confidence := 0.5

	if velAnalysis["mean"] > 50 && velAnalysis["variance"] < 10 {
		intentType = "rapid_navigation"
		confidence = 0.8
	} else if velAnalysis["mean"] < 20 && dirAnalysis["change_rate"] > 0.5 {
		intentType = "careful_selection"
		confidence = 0.75
	} else if dirAnalysis["change_rate"] < 0.1 {
		intentType = "straight_movement"
		confidence = 0.85
	} else if len(seq.Points) > 50 {
		intentType = "exploration"
		confidence = 0.7
	} else if len(seq.Points) < 10 {
		intentType = "quick_action"
		confidence = 0.6
	}

	actionProbs := make(map[string]float64)
	if intentType == "rapid_navigation" {
		actionProbs["continue_scrolling"] = 0.7
		actionProbs["click_target"] = 0.2
		actionProbs["stop"] = 0.1
	} else if intentType == "careful_selection" {
		actionProbs["click_target"] = 0.6
		actionProbs["hover"] = 0.3
		actionProbs["continue_browsing"] = 0.1
	} else {
		actionProbs["continue_browsing"] = 0.5
		actionProbs["click_target"] = 0.3
		actionProbs["scroll"] = 0.2
	}

	return &IntentPrediction{
		IntentType:         intentType,
		Confidence:         confidence,
		SubIntents:         make([]IntentPrediction, 0),
		ExpectedNextAction: t.predictNextAction(actionProbs),
		ActionProbabilities: actionProbs,
	}
}

func (t *TransformerPredictor) predictNextAction(probs map[string]float64) string {
	maxProb := 0.0
	maxAction := ""
	for action, prob := range probs {
		if prob > maxProb {
			maxProb = prob
			maxAction = action
		}
	}
	return maxAction
}

func (t *TransformerPredictor) DetectAnomalies(seq *TrajectorySequence) []SequenceAnomaly {
	anomalies := make([]SequenceAnomaly, 0)

	if seq.VelocitySeq != nil {
		for i, vel := range seq.VelocitySeq {
			if vel > 150 {
				anomalies = append(anomalies, SequenceAnomaly{
					AnomalyType:     "extreme_velocity",
					Position:       i,
					Severity:       math.Min(1.0, vel/200.0),
					Description:    fmt.Sprintf("Velocity %.2f exceeds normal range", vel),
					SuggestedAction: "flag_as_suspicious",
				})
			}
		}
	}

	if seq.AccelerationSeq != nil {
		for i, acc := range seq.AccelerationSeq {
			if math.Abs(acc) > 300 {
				anomalies = append(anomalies, SequenceAnomaly{
					AnomalyType:     "extreme_acceleration",
					Position:       i,
					Severity:       math.Min(1.0, math.Abs(acc)/500.0),
					Description:    fmt.Sprintf("Acceleration %.2f exceeds normal range", acc),
					SuggestedAction: "flag_as_suspicious",
				})
			}
		}
	}

	if seq.DirectionSeq != nil && len(seq.DirectionSeq) > 1 {
		for i := 1; i < len(seq.DirectionSeq); i++ {
			diff := math.Abs(seq.DirectionSeq[i] - seq.DirectionSeq[i-1])
			if diff > math.Pi {
				diff = 2*math.Pi - diff
			}
			if diff > 2.5 {
				anomalies = append(anomalies, SequenceAnomaly{
					AnomalyType:     "abrupt_direction_change",
					Position:       i,
					Severity:       diff / math.Pi,
					Description:    "Abrupt direction change detected",
					SuggestedAction: "monitor_behavior",
				})
			}
		}
	}

	if len(seq.Points) < 3 {
		anomalies = append(anomalies, SequenceAnomaly{
			AnomalyType:     "insufficient_data",
			Position:       0,
			Severity:       0.3,
			Description:    "Sequence too short for reliable analysis",
			SuggestedAction: "collect_more_data",
		})
	}

	return anomalies
}

func (t *TransformerPredictor) AnalyzeComprehensiveBehavior(seq *TrajectorySequence) *ComprehensiveBehaviorResult {
	startTime := time.Now()

	riskPrediction, _ := t.Predict(seq)
	behaviorClass := t.ClassifyBehavior(seq)
	intentPrediction := t.PredictIntent(seq)
	anomalies := t.DetectAnomalies(seq)

	executionTime := time.Since(startTime)

	return &ComprehensiveBehaviorResult{
		RiskPrediction:       riskPrediction,
		BehaviorClassification: behaviorClass,
		IntentPrediction:     intentPrediction,
		Anomalies:           anomalies,
		ExecutionTime:       executionTime,
	}
}

func (t *TransformerPredictor) AnalyzeComprehensiveBehaviorFromTrace(traceData *model.TraceData) (*ComprehensiveBehaviorResult, error) {
	seq, err := t.encodeToSequence(traceData)
	if err != nil {
		return nil, err
	}
	return t.AnalyzeComprehensiveBehavior(seq), nil
}

func (t *TransformerPredictor) GetPredictionStats() map[string]float64 {
	return map[string]float64{
		"embedding_dimension":      float64(t.GetEmbeddingDimension()),
		"attention_heads":          float64(t.GetAttentionHeads()),
		"num_layers":               float64(EnhancedTransformerLayers),
		"max_sequence_length":      float64(MaxSequenceLen),
		"memory_usage_bytes":       float64(t.GetMemoryUsageBytes()),
		"dropout_rate":             t.dropoutRate,
		"quantization_enabled":     boolToFloat(t.quantizationEnabled),
		"layer_norm_enabled":       boolToFloat(t.useLayerNorm),
	}
}

func boolToFloat(b bool) float64 {
	if b {
		return 1.0
	}
	return 0.0
}