package trace

import (
	"errors"
	"math"
	"math/rand"
	"sort"

	"github.com/hjtpx/hjtpx/internal/model"
)

type AnomalyDetector struct {
	extractor *TraceExtractor
}

func NewAnomalyDetector() *AnomalyDetector {
	return &AnomalyDetector{
		extractor: NewTraceExtractor(),
	}
}

type IsolationTree struct {
	Left     *IsolationTree
	Right    *IsolationTree
	SplitAt  float64
	SplitDim int
	Depth    int
	Size     int
}

type IsolationForest struct {
	Trees    []*IsolationTree
	MaxDepth int
	NumTrees int
}

func (d *AnomalyDetector) TrainIsolationForest(traces []*model.TraceData, numTrees, maxDepth int) (*IsolationForest, error) {
	if len(traces) == 0 {
		return nil, errors.New("no traces provided for training")
	}

	forest := &IsolationForest{
		Trees:    make([]*IsolationTree, numTrees),
		MaxDepth: maxDepth,
		NumTrees: numTrees,
	}

	featuresList := make([][]float64, len(traces))
	for i, trace := range traces {
		features, err := d.extractor.ExtractFeatures(trace)
		if err != nil {
			return nil, err
		}
		featuresList[i] = d.featuresToVector(features)
	}

	for i := 0; i < numTrees; i++ {
		sample := d.sampleWithReplacement(featuresList, len(featuresList))
		forest.Trees[i] = d.buildIsolationTree(sample, 0, maxDepth)
	}

	return forest, nil
}

func (d *AnomalyDetector) featuresToVector(f *model.TraceFeatures) []float64 {
	return []float64{
		f.AvgSpeed,
		f.MaxSpeed,
		f.SpeedVariance,
		f.MaxAcceleration,
		f.AvgAcceleration,
		f.Smoothness,
		f.PathRatio,
		f.AvgCurvature,
		f.JitterFrequency,
		f.JitterAmplitude,
	}
}

func (d *AnomalyDetector) sampleWithReplacement(data [][]float64, size int) [][]float64 {
	sample := make([][]float64, size)
	for i := 0; i < size; i++ {
		idx := rand.Intn(len(data))
		sample[i] = data[idx]
	}
	return sample
}

func (d *AnomalyDetector) buildIsolationTree(data [][]float64, depth, maxDepth int) *IsolationTree {
	if depth >= maxDepth || len(data) <= 1 {
		return &IsolationTree{
			Depth: depth,
			Size:  len(data),
		}
	}

	numDim := len(data[0])
	splitDim := rand.Intn(numDim)

	minVal, maxVal := d.getMinMax(data, splitDim)
	if minVal == maxVal {
		return &IsolationTree{
			Depth: depth,
			Size:  len(data),
		}
	}

	splitAt := minVal + rand.Float64()*(maxVal-minVal)

	leftData := make([][]float64, 0)
	rightData := make([][]float64, 0)

	for _, point := range data {
		if point[splitDim] < splitAt {
			leftData = append(leftData, point)
		} else {
			rightData = append(rightData, point)
		}
	}

	return &IsolationTree{
		Left:     d.buildIsolationTree(leftData, depth+1, maxDepth),
		Right:    d.buildIsolationTree(rightData, depth+1, maxDepth),
		SplitAt:  splitAt,
		SplitDim: splitDim,
		Depth:    depth,
		Size:     len(data),
	}
}

func (d *AnomalyDetector) getMinMax(data [][]float64, dim int) (float64, float64) {
	minVal := math.MaxFloat64
	maxVal := -math.MaxFloat64

	for _, point := range data {
		if point[dim] < minVal {
			minVal = point[dim]
		}
		if point[dim] > maxVal {
			maxVal = point[dim]
		}
	}

	return minVal, maxVal
}

func (d *AnomalyDetector) PredictAnomalyScore(forest *IsolationForest, trace *model.TraceData) (float64, error) {
	if forest == nil || trace == nil {
		return 0, errors.New("invalid forest or trace")
	}

	features, err := d.extractor.ExtractFeatures(trace)
	if err != nil {
		return 0, err
	}

	vector := d.featuresToVector(features)
	avgDepth := d.averagePathLength(forest, vector)

	return math.Pow(2, -avgDepth/d.avgPathLengthForSize(len(forest.Trees))), nil
}

func (d *AnomalyDetector) averagePathLength(forest *IsolationForest, vector []float64) float64 {
	totalDepth := 0.0
	for _, tree := range forest.Trees {
		totalDepth += float64(d.pathLength(tree, vector))
	}
	return totalDepth / float64(len(forest.Trees))
}

func (d *AnomalyDetector) pathLength(tree *IsolationTree, vector []float64) int {
	if tree.Left == nil && tree.Right == nil {
		return tree.Depth
	}

	if vector[tree.SplitDim] < tree.SplitAt {
		if tree.Left != nil {
			return d.pathLength(tree.Left, vector)
		}
		return tree.Depth + 1
	}

	if tree.Right != nil {
		return d.pathLength(tree.Right, vector)
	}
	return tree.Depth + 1
}

func (d *AnomalyDetector) avgPathLengthForSize(n int) float64 {
	if n <= 1 {
		return 0
	}
	return 2 * (math.Log(float64(n-1)) + 0.5772156649) - 2*(float64(n)-1)/float64(n)
}

type Autoencoder struct {
	InputSize     int
	HiddenSize    int
	OutputSize    int
	Weights1      [][]float64
	Weights2      [][]float64
	Bias1         []float64
	Bias2         []float64
	LearningRate  float64
	Epochs        int
}

func (d *AnomalyDetector) NewAutoencoder(inputSize, hiddenSize int) *Autoencoder {
	ae := &Autoencoder{
		InputSize:    inputSize,
		HiddenSize:   hiddenSize,
		OutputSize:   inputSize,
		LearningRate: 0.01,
		Epochs:       100,
	}

	ae.Weights1 = d.initRandomMatrix(hiddenSize, inputSize)
	ae.Weights2 = d.initRandomMatrix(inputSize, hiddenSize)
	ae.Bias1 = make([]float64, hiddenSize)
	ae.Bias2 = make([]float64, inputSize)

	return ae
}

func (d *AnomalyDetector) initRandomMatrix(rows, cols int) [][]float64 {
	matrix := make([][]float64, rows)
	for i := range matrix {
		matrix[i] = make([]float64, cols)
		for j := range matrix[i] {
			matrix[i][j] = (rand.Float64() - 0.5) * 2
		}
	}
	return matrix
}

func (d *AnomalyDetector) sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

func (d *AnomalyDetector) sigmoidDerivative(x float64) float64 {
	s := d.sigmoid(x)
	return s * (1 - s)
}

func (d *AnomalyDetector) ReLU(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func (d *AnomalyDetector) ReLUDerivative(x float64) float64 {
	if x > 0 {
		return 1
	}
	return 0
}

func (d *AnomalyDetector) TrainAutoencoder(ae *Autoencoder, traces []*model.TraceData) error {
	if len(traces) == 0 {
		return errors.New("no traces provided for training")
	}

	featuresList := make([][]float64, len(traces))
	for i, trace := range traces {
		features, err := d.extractor.ExtractFeatures(trace)
		if err != nil {
			return err
		}
		featuresList[i] = d.featuresToVector(features)
	}

	d.normalizeFeatures(featuresList)

	for epoch := 0; epoch < ae.Epochs; epoch++ {
		for _, input := range featuresList {
			hidden, output := d.forwardPass(ae, input)
			d.backwardPass(ae, input, hidden, output)
		}
	}

	return nil
}

func (d *AnomalyDetector) normalizeFeatures(data [][]float64) {
	for dim := 0; dim < len(data[0]); dim++ {
		sum := 0.0
		sqSum := 0.0
		for _, point := range data {
			sum += point[dim]
			sqSum += point[dim] * point[dim]
		}
		mean := sum / float64(len(data))
		std := math.Sqrt(sqSum/float64(len(data)) - mean*mean)
		if std > 0 {
			for i := range data {
				data[i][dim] = (data[i][dim] - mean) / std
			}
		}
	}
}

func (d *AnomalyDetector) forwardPass(ae *Autoencoder, input []float64) ([]float64, []float64) {
	hidden := make([]float64, ae.HiddenSize)
	for i := range hidden {
		sum := ae.Bias1[i]
		for j := range input {
			sum += ae.Weights1[i][j] * input[j]
		}
		hidden[i] = d.ReLU(sum)
	}

	output := make([]float64, ae.OutputSize)
	for i := range output {
		sum := ae.Bias2[i]
		for j := range hidden {
			sum += ae.Weights2[i][j] * hidden[j]
		}
		output[i] = d.sigmoid(sum)
	}

	return hidden, output
}

func (d *AnomalyDetector) backwardPass(ae *Autoencoder, input, hidden, output []float64) {
	outputError := make([]float64, ae.OutputSize)
	for i := range outputError {
		outputError[i] = (output[i] - input[i]) * d.sigmoidDerivative(output[i])
	}

	hiddenError := make([]float64, ae.HiddenSize)
	for i := range hiddenError {
		sum := 0.0
		for j := range outputError {
			sum += ae.Weights2[j][i] * outputError[j]
		}
		hiddenError[i] = sum * d.ReLUDerivative(hidden[i])
	}

	for i := range ae.Weights2 {
		for j := range ae.Weights2[i] {
			ae.Weights2[i][j] -= ae.LearningRate * outputError[i] * hidden[j]
		}
		ae.Bias2[i] -= ae.LearningRate * outputError[i]
	}

	for i := range ae.Weights1 {
		for j := range ae.Weights1[i] {
			ae.Weights1[i][j] -= ae.LearningRate * hiddenError[i] * input[j]
		}
		ae.Bias1[i] -= ae.LearningRate * hiddenError[i]
	}
}

func (d *AnomalyDetector) PredictAutoencoderAnomaly(ae *Autoencoder, trace *model.TraceData) (float64, error) {
	if ae == nil || trace == nil {
		return 0, errors.New("invalid autoencoder or trace")
	}

	features, err := d.extractor.ExtractFeatures(trace)
	if err != nil {
		return 0, err
	}

	input := d.featuresToVector(features)

	_, output := d.forwardPass(ae, input)

	return d.calculateReconstructionError(input, output), nil
}

func (d *AnomalyDetector) calculateReconstructionError(input, output []float64) float64 {
	sum := 0.0
	for i := range input {
		sum += math.Pow(input[i]-output[i], 2)
	}
	return math.Sqrt(sum / float64(len(input)))
}

type AnomalyResult struct {
	Score      float64
	IsAnomaly  bool
	Method     string
	Confidence float64
}

func (d *AnomalyDetector) DetectAnomaly(trace *model.TraceData, forest *IsolationForest, ae *Autoencoder) (*AnomalyResult, error) {
	if forest != nil {
		score, err := d.PredictAnomalyScore(forest, trace)
		if err != nil {
			return nil, err
		}
		return &AnomalyResult{
			Score:      score,
			IsAnomaly:  score > 0.5,
			Method:     "isolation_forest",
			Confidence: score,
		}, nil
	}

	if ae != nil {
		score, err := d.PredictAutoencoderAnomaly(ae, trace)
		if err != nil {
			return nil, err
		}
		return &AnomalyResult{
			Score:      score,
			IsAnomaly:  score > 0.1,
			Method:     "autoencoder",
			Confidence: 1.0 - score,
		}, nil
	}

	return nil, errors.New("no detection method provided")
}

func (d *AnomalyDetector) BatchDetectAnomalies(traces []*model.TraceData, forest *IsolationForest) ([]*AnomalyResult, error) {
	results := make([]*AnomalyResult, len(traces))
	for i, trace := range traces {
		result, err := d.DetectAnomaly(trace, forest, nil)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}
	return results, nil
}

func (d *AnomalyDetector) GetAnomalyThreshold(traces []*model.TraceData, forest *IsolationForest, percentile float64) (float64, error) {
	if len(traces) == 0 {
		return 0, errors.New("no traces provided")
	}

	scores := make([]float64, len(traces))
	for i, trace := range traces {
		score, err := d.PredictAnomalyScore(forest, trace)
		if err != nil {
			return 0, err
		}
		scores[i] = score
	}

	sort.Float64s(scores)
	index := int(math.Ceil(percentile * float64(len(scores))))
	if index >= len(scores) {
		index = len(scores) - 1
	}

	return scores[index], nil
}

func (d *AnomalyDetector) HybridAnomalyDetection(trace *model.TraceData, forest *IsolationForest, ae *Autoencoder, weights ...float64) (*AnomalyResult, error) {
	if forest == nil && ae == nil {
		return nil, errors.New("no detection method provided")
	}

	var ifScore, aeScore float64
	var ifErr, aeErr error

	if forest != nil {
		ifScore, ifErr = d.PredictAnomalyScore(forest, trace)
		if ifErr != nil {
			return nil, ifErr
		}
	}

	if ae != nil {
		aeScore, aeErr = d.PredictAutoencoderAnomaly(ae, trace)
		if aeErr != nil {
			return nil, aeErr
		}
	}

	ifScoreNorm := ifScore
	aeScoreNorm := 1.0 - aeScore

	var weight1, weight2 float64
	if len(weights) >= 2 {
		weight1 = weights[0]
		weight2 = weights[1]
	} else {
		weight1 = 0.5
		weight2 = 0.5
	}

	hybridScore := weight1*ifScoreNorm + weight2*aeScoreNorm

	return &AnomalyResult{
		Score:      hybridScore,
		IsAnomaly:  hybridScore > 0.5,
		Method:     "hybrid",
		Confidence: hybridScore,
	}, nil
}
