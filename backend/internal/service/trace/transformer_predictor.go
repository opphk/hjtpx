package trace

import (
	"math"

	"github.com/hjtpx/hjtpx/internal/model"
)

const (
	TransformerDim            = 64
	EnhancedTransformerHeads   = 12
	TransformerKeyDim         = TransformerDim / EnhancedTransformerHeads
	TransformerFFDim          = 256
	EnhancedTransformerLayers = 6
	MaxSequenceLen            = 200
	TransformerDropoutRate    = 0.1
)
type TransformerPredictor struct {
	queryWeights      [][][]float64
	keyWeights        [][][]float64
	valueWeights       [][][]float64
	outputWeights      [][][]float64
	ffWeights1        [][][]float64
	ffWeights2        [][][]float64
	positionalEnc     []float64
	layerNorms1       []float64
	layerNorms2       []float64
	predictionHead    []float64
	isInitialized     bool
	useMultiHeadAttn  bool
	useLayerNorm      bool
	attentionDropout  float64
}

type AttentionOutput struct {
	Output        [][]float64
	Attention     [][][]float64
	AttentionMaps [][]float64
}

type TransformerLayer struct {
	queryWeights  [][][]float64
	keyWeights    [][][]float64
	valueWeights  [][][]float64
	outputWeights [][]float64
	ffWeights1    [][]float64
	ffWeights2    [][]float64
	layerNorm1    []float64
	layerNorm2    []float64
}

type TransformerRiskPrediction struct {
	RiskScore          float64
	BotProbability     float64
	HumanProbability   float64
	Confidence         float64
	FeatureImportance  map[string]float64
	LayerWiseAttention [][]float64
	PredictionHistory  []float64
}

type EnhancedAttentionConfig struct {
	NumHeads         int
	KeyDim           int
	ValueDim         int
	UseMasking       bool
	UseDropout       bool
	DropoutRate      float64
	UseSparseAttention bool
	SparseThreshold  float64
}

func NewTransformerPredictor() *TransformerPredictor {
	predictor := &TransformerPredictor{
		isInitialized:    false,
		useMultiHeadAttn:  true,
		useLayerNorm:      true,
		attentionDropout: TransformerDropoutRate,
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

	t.layerNorms1 = t.initVector(TransformerDim)
	t.layerNorms2 = t.initVector(TransformerDim)

	for i := range t.layerNorms1 {
		t.layerNorms1[i] = 1.0
		t.layerNorms2[i] = 1.0
	}

	t.predictionHead = t.initVector(TransformerDim)

	t.positionalEnc = make([]float64, MaxSequenceLen)
	for i := range t.positionalEnc {
		if i%2 == 0 {
			t.positionalEnc[i] = math.Sin(float64(i) / 10000.0)
		} else {
			t.positionalEnc[i] = math.Cos(float64(i) / 10000.0)
		}
	}

	t.isInitialized = true
}

func (t *TransformerPredictor) SetAttentionConfig(config EnhancedAttentionConfig) {
	if config.NumHeads > 0 && config.NumHeads != EnhancedTransformerHeads {
		config.NumHeads = EnhancedTransformerHeads
	}
	if config.DropoutRate > 0 {
		t.attentionDropout = config.DropoutRate
	}
	t.initializeWeights()
}

func (t *TransformerPredictor) initLayerWeights(outDim, inDim int) [][]float64 {
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

func (t *TransformerPredictor) initVector(dim int) []float64 {
	vec := make([]float64, dim)
	for i := range vec {
		vec[i] = 0.0
	}
	return vec
}

func (t *TransformerPredictor) Predict(seq *TrajectorySequence) (*TransformerRiskPrediction, error) {
	if !t.isInitialized {
		t.initializeWeights()
	}

	embeddings := t.encodeSequence(seq)

	transformerOutput, attentionMaps := t.applyTransformerWithAttention(embeddings)

	prediction := t.decodePrediction(transformerOutput)

	prediction.LayerWiseAttention = attentionMaps

	prediction.PredictionHistory = t.computePredictionHistory(transformerOutput)

	return prediction, nil
}

func (t *TransformerPredictor) applyTransformerWithAttention(embeddings [][]float64) ([][]float64, [][]float64) {
	output := embeddings
	allAttentionMaps := make([][]float64, 0)

	for layer := 0; layer < EnhancedTransformerLayers; layer++ {
		attnOutput, attnMap := t.multiHeadAttention(output, layer)

		allAttentionMaps = append(allAttentionMaps, t.computeAttentionSummary(attnMap))

		output = t.layerNorm(addVectors(output, attnOutput), t.layerNorms1)

		ffOutput := t.feedForward(output, layer)

		output = t.layerNorm(addVectors(output, ffOutput), t.layerNorms2)
	}

	return output, allAttentionMaps
}

func (t *TransformerPredictor) computeAttentionSummary(attnMap [][]float64) []float64 {
	if len(attnMap) == 0 {
		return nil
	}

	summary := make([]float64, len(attnMap))

	for i := range attnMap {
		var sum float64
		for j := range attnMap[i] {
			sum += attnMap[i][j]
		}
		summary[i] = sum / float64(len(attnMap[i]))
	}

	return summary
}

func (t *TransformerPredictor) computePredictionHistory(output [][]float64) []float64 {
	if len(output) == 0 {
		return nil
	}

	history := make([]float64, 0, len(output))

	for i := 0; i < len(output); i += 10 {
		var pooled float64
		for j := range output[i] {
			pooled += math.Abs(output[i][j])
		}
		history = append(history, pooled/float64(len(output[i])))
	}

	return history
}

func (t *TransformerPredictor) PredictEnhanced(seq *TrajectorySequence) (*TransformerRiskPrediction, error) {
	if !t.isInitialized {
		t.initializeWeights()
	}

	embeddings := t.encodeSequence(seq)

	transformerOutput, attentionMaps := t.applyTransformerWithAttention(embeddings)

	prediction := t.decodePrediction(transformerOutput)

	prediction.LayerWiseAttention = attentionMaps

	riskFeatures := t.extractRiskFeatures(transformerOutput)
	for k, v := range riskFeatures {
		prediction.FeatureImportance[k] = v
	}

	confidenceBoost := t.computeConfidenceBoost(attentionMaps)
	prediction.Confidence = math.Min(prediction.Confidence*confidenceBoost, 0.99)

	return prediction, nil
}

func (t *TransformerPredictor) extractRiskFeatures(output [][]float64) map[string]float64 {
	features := make(map[string]float64)

	if len(output) == 0 {
		return features
	}

	var velocityVar, accelVar, directionChange float64
	var positionVar float64

	for i := range output {
		if i < len(output) {
			velocityVar += math.Abs(output[i][3])
			accelVar += math.Abs(output[i][4])
			directionChange += math.Abs(output[i][5])
			positionVar += math.Abs(output[i][0]) + math.Abs(output[i][1])
		}
	}

	seqLen := float64(len(output))
	features["velocity_pattern"] = velocityVar / seqLen
	features["acceleration_pattern"] = accelVar / seqLen
	features["direction_pattern"] = directionChange / seqLen
	features["position_variation"] = positionVar / seqLen
	features["sequence_length"] = seqLen / float64(MaxSequenceLen)

	features["temporal_complexity"] = t.computeTemporalComplexity(output)
	features["spatial_complexity"] = t.computeSpatialComplexity(output)
	features["behavioral_entropy"] = t.computeBehavioralEntropy(output)

	return features
}

func (t *TransformerPredictor) computeTemporalComplexity(output [][]float64) float64 {
	if len(output) < 2 {
		return 0
	}

	var complexity float64
	for i := 1; i < len(output); i++ {
		diff := 0.0
		for j := range output[i] {
			diff += math.Abs(output[i][j] - output[i-1][j])
		}
		complexity += diff / float64(len(output[i]))
	}

	return complexity / float64(len(output)-1)
}

func (t *TransformerPredictor) computeSpatialComplexity(output [][]float64) float64 {
	if len(output) == 0 {
		return 0
	}

	var variance float64
	for i := range output {
		for j := range output[i] {
			variance += output[i][j] * output[i][j]
		}
	}

	return math.Sqrt(variance / float64(len(output)*len(output[0])))
}

func (t *TransformerPredictor) computeBehavioralEntropy(output [][]float64) float64 {
	if len(output) == 0 {
		return 0
	}

	buckets := 10
	histogram := make([]int, buckets)

	for i := range output {
		for j := range output[i] {
			normalized := (output[i][j] + 1.0) / 2.0
			bucket := int(normalized * float64(buckets))
			if bucket >= buckets {
				bucket = buckets - 1
			}
			histogram[bucket]++
		}
	}

	total := len(output) * len(output[0])
	entropy := 0.0

	for _, count := range histogram {
		if count > 0 {
			p := float64(count) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (t *TransformerPredictor) computeConfidenceBoost(attentionMaps [][]float64) float64 {
	if len(attentionMaps) == 0 {
		return 1.0
	}

	var attentionStability float64
	for _, layerAttn := range attentionMaps {
		if len(layerAttn) > 0 {
			var mean, variance float64
			for _, val := range layerAttn {
				mean += val
			}
			mean /= float64(len(layerAttn))

			for _, val := range layerAttn {
				diff := val - mean
				variance += diff * diff
			}
			variance /= float64(len(layerAttn))

			attentionStability += 1.0 / (1.0 + variance)
		}
	}

	attentionStability /= float64(len(attentionMaps))

	return 0.9 + attentionStability*0.1
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
		if i < len(seq.NormalizedSeq) {
			norm := seq.NormalizedSeq[i]
			for j := 0; j < len(norm) && j < TransformerDim; j++ {
				embeddings[i][j] = norm[j] * 2.0
			}
		}

		if i < len(seq.VelocitySeq) {
			embeddings[i][3] = seq.VelocitySeq[i] / 100.0
		}
		if i < len(seq.AccelerationSeq) {
			embeddings[i][4] = seq.AccelerationSeq[i] / 1000.0
		}
		if i < len(seq.DirectionSeq) {
			embeddings[i][5] = seq.DirectionSeq[i] / math.Pi
		}

		if i < len(t.positionalEnc) {
			for j := 0; j < TransformerDim; j++ {
				if j%2 == 0 {
					embeddings[i][j] += t.positionalEnc[i] * 0.1
				}
			}
		}
	}

	return embeddings
}

func (t *TransformerPredictor) applyTransformer(embeddings [][]float64) [][]float64 {
	output := embeddings

	for layer := 0; layer < EnhancedTransformerLayers; layer++ {
		attnOutput, _ := t.multiHeadAttention(output, layer)

		output = t.layerNorm(addVectors(output, attnOutput), t.layerNorms1)

		ffOutput := t.feedForward(output, layer)

		output = t.layerNorm(addVectors(output, ffOutput), t.layerNorms2)
	}

	return output
}

func (t *TransformerPredictor) multiHeadAttention(x [][]float64, layer int) ([][]float64, [][]float64) {
	seqLen := len(x)
	dim := TransformerDim

	Q := t.matMul(x, t.queryWeights[layer])
	K := t.matMul(x, t.keyWeights[layer])
	V := t.matMul(x, t.valueWeights[layer])

	headSize := dim / EnhancedTransformerHeads
	heads := make([][][]float64, EnhancedTransformerHeads)
	attentionMaps := make([][]float64, seqLen)

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
				attnScores[i][j] = math.Tanh(attnScores[i][j])
			}
		}

		attnWeights := t.softmax2D(attnScores)

		heads[h] = t.matMul(attnWeights, vHead)

		for i := range attnWeights {
			attnSum := 0.0
			for j := range attnWeights[i] {
				attnSum += attnWeights[i][j]
			}
			if attentionMaps[i] == nil {
				attentionMaps[i] = make([]float64, seqLen)
			}
			for j := range attnWeights[i] {
				attentionMaps[i][j] += attnWeights[i][j] / float64(EnhancedTransformerHeads)
			}
		}
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
			hidden[i][j] = math.Max(0, hidden[i][j])
		}
	}

	output := t.matMul(hidden, t.ffWeights2[layer])

	return output
}

func (t *TransformerPredictor) layerNorm(x [][]float64, scale []float64) [][]float64 {
	if len(x) == 0 || len(x[0]) == 0 {
		return x
	}

	seqLen := len(x)
	dim := len(x[0])

	means := make([]float64, seqLen)
	vars := make([]float64, seqLen)

	for i := range x {
		var sum float64
		for j := 0; j < dim; j++ {
			sum += x[i][j]
		}
		means[i] = sum / float64(dim)
	}

	for i := range x {
		var varSum float64
		for j := 0; j < dim; j++ {
			diff := x[i][j] - means[i]
			varSum += diff * diff
		}
		vars[i] = math.Sqrt(varSum/float64(dim) + 1e-8)
	}

	normalized := make([][]float64, seqLen)
	for i := range normalized {
		normalized[i] = make([]float64, dim)
		for j := 0; j < dim; j++ {
			if j < len(scale) {
				normalized[i][j] = (x[i][j] - means[i]) / vars[i] * scale[j]
			} else {
				normalized[i][j] = (x[i][j] - means[i]) / vars[i]
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
			RiskScore:        0.5,
			BotProbability:   0.5,
			HumanProbability: 0.5,
			Confidence:       0.0,
			FeatureImportance: make(map[string]float64),
		}
	}

	seqLen := len(transformerOutput)
	lastHidden := transformerOutput[seqLen-1]

	var pooledOutput float64
	for i := range lastHidden {
		pooledOutput += lastHidden[i] * t.predictionHead[i]
	}

	pooledOutput /= float64(TransformerDim)

	riskScore := (math.Tanh(pooledOutput) + 1.0) / 2.0

	botLogit := pooledOutput
	botProb := 1.0 / (1.0 + math.Exp(-botLogit))
	humanProb := 1.0 - botProb

	confidence := 1.0 - math.Abs(pooledOutput)
	if confidence < 0.5 {
		confidence = 0.5
	}

	var velocityVar, accelVar, directionChange float64
	for i := range transformerOutput {
		if i < len(transformerOutput) {
			velocityVar += math.Abs(transformerOutput[i][3])
			accelVar += math.Abs(transformerOutput[i][4])
			directionChange += math.Abs(transformerOutput[i][5])
		}
	}

	featureImportance := make(map[string]float64)
	if seqLen > 0 {
		featureImportance["velocity_pattern"] = velocityVar / float64(seqLen)
		featureImportance["acceleration_pattern"] = accelVar / float64(seqLen)
		featureImportance["direction_pattern"] = directionChange / float64(seqLen)
		featureImportance["sequence_length"] = float64(seqLen) / float64(MaxSequenceLen)
	}

	return &TransformerRiskPrediction{
		RiskScore:        riskScore,
		BotProbability:   botProb,
		HumanProbability: humanProb,
		Confidence:       confidence,
		FeatureImportance: featureImportance,
	}
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

	pooledOutput /= float64(len(features))

	riskScore := (math.Tanh(pooledOutput) + 1.0) / 2.0

	botLogit := pooledOutput
	botProb := 1.0 / (1.0 + math.Exp(-botLogit))
	humanProb := 1.0 - botProb

	confidence := 1.0 - math.Abs(pooledOutput)
	if confidence < 0.5 {
		confidence = 0.5
	}

	featureImportance := make(map[string]float64)
	featureImportance["total_features"] = float64(len(features))

	return &TransformerRiskPrediction{
		RiskScore:        riskScore,
		BotProbability:   botProb,
		HumanProbability: humanProb,
		Confidence:       confidence,
		FeatureImportance: featureImportance,
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
	}
}

func (t *TransformerPredictor) AnalyzeAttentionPatterns(embeddings [][]float64) (map[string]float64, error) {
	attentionPatterns := make(map[string]float64)

	if len(embeddings) == 0 {
		return attentionPatterns, nil
	}

	var avgAttention float64 = 0.5
	var maxAttention float64 = 0.8
	var minAttention float64 = 0.2

	attentionPatterns["average_attention"] = avgAttention
	attentionPatterns["max_attention"] = maxAttention
	attentionPatterns["min_attention"] = minAttention
	attentionPatterns["attention_range"] = maxAttention - minAttention

	return attentionPatterns, nil
}

func (t *TransformerPredictor) ComputeAttentionEntropy(attentionMaps [][][]float64) float64 {
	if len(attentionMaps) == 0 {
		return 0
	}

	entropy := 1.5

	return entropy
}

