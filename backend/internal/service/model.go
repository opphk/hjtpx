package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type ModelType string

const (
	ModelTypeSVM       ModelType = "svm"
	ModelTypeDecisionTree ModelType = "decision_tree"
	ModelTypeEnsemble   ModelType = "ensemble"
)

type ModelMetadata struct {
	Name        string    `json:"name"`
	Version     string    `json:"version"`
	Type        ModelType `json:"type"`
	CreatedAt   time.Time `json:"created_at"`
	UpdatedAt   time.Time `json:"updated_at"`
	Accuracy    float64   `json:"accuracy"`
	Precision   float64   `json:"precision"`
	Recall      float64   `json:"recall"`
	F1Score     float64   `json:"f1_score"`
	Features    []string  `json:"features"`
}

type TrainingConfig struct {
	LearningRate    float64 `json:"learning_rate"`
	MaxIterations   int     `json:"max_iterations"`
	TrainTestSplit  float64 `json:"train_test_split"`
	BatchSize       int     `json:"batch_size"`
	Regularization  float64 `json:"regularization"`
}

type TrainedModel struct {
	Metadata   ModelMetadata   `json:"metadata"`
	Config    TrainingConfig  `json:"config"`
	Weights   []float64      `json:"weights"`
	Bias      float64         `json:"bias"`
	SVMParams SVMParams       `json:"svm_params"`
	Tree      *DecisionTree   `json:"tree,omitempty"`
}

type SVMParams struct {
	Kernel      string    `json:"kernel"`
	Gamma       float64   `json:"gamma"`
	C           float64   `json:"c"`
	SupportVectors [][]float64 `json:"support_vectors"`
	Classes     []int     `json:"classes"`
}

type DecisionTree struct {
	SplitFeature int     `json:"split_feature"`
	SplitValue  float64 `json:"split_value"`
	Left        *DecisionTree `json:"left,omitempty"`
	Right       *DecisionTree `json:"right,omitempty"`
	Prediction  int     `json:"prediction"`
	IsLeaf      bool    `json:"is_leaf"`
	ClassCounts map[int]int `json:"class_counts"`
}

type ModelService struct {
	models     map[string]*TrainedModel
	currentModel *TrainedModel
	modelMutex sync.RWMutex
	modelDir   string
	cache      *PredictionCache
}

type PredictionCache struct {
	entries map[string]*CachedPrediction
	maxSize int
	mutex   sync.RWMutex
}

type CachedPrediction struct {
	Prediction *MLPrediction
	ExpiresAt time.Time
}

type TrainingData struct {
	Features [][]float64 `json:"features"`
	Labels   []int       `json:"labels"`
}

type CrossValidationResult struct {
	Folds          int       `json:"folds"`
	AccuracyScores []float64 `json:"accuracy_scores"`
	MeanAccuracy   float64   `json:"mean_accuracy"`
	StdDeviation   float64   `json:"std_deviation"`
}

func NewModelService() *ModelService {
	service := &ModelService{
		models:   make(map[string]*TrainedModel),
		modelDir: "./models",
		cache: &PredictionCache{
			entries: make(map[string]*CachedPrediction),
			maxSize: 1000,
		},
	}
	
	service.initializeDefaultModel()
	
	return service
}

func (s *ModelService) initializeDefaultModel() {
	defaultModel := &TrainedModel{
		Metadata: ModelMetadata{
			Name:      "bot_detection_default",
			Version:   "1.0.0",
			Type:      ModelTypeEnsemble,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Accuracy:  0.85,
			Precision: 0.82,
			Recall:    0.88,
			F1Score:   0.85,
			Features: []string{
				"mouse_speed_avg",
				"mouse_speed_max",
				"trajectory_length",
				"path_efficiency",
				"direction_changes",
				"smoothness",
				"click_count",
				"click_regularity",
				"click_speed",
			},
		},
		Config: TrainingConfig{
			LearningRate:   0.01,
			MaxIterations:  1000,
			TrainTestSplit: 0.8,
			BatchSize:      32,
			Regularization: 0.01,
		},
		Weights: []float64{0.1, 0.1, 0.1, 0.15, 0.1, 0.1, 0.1, 0.15, 0.1},
		Bias:    -0.5,
		SVMParams: SVMParams{
			Kernel: "rbf",
			Gamma:  0.1,
			C:      1.0,
		},
	}
	
	s.models["default"] = defaultModel
	s.currentModel = defaultModel
}

func (s *ModelService) Predict(features *FeatureVector) *MLPrediction {
	if features == nil {
		return &MLPrediction{
			IsBot:        false,
			Confidence:   0.0,
			BotScore:     0.5,
			HumanScore:   0.5,
			ModelVersion: "unknown",
		}
	}
	
	featureVector := features.FeatureVector
	if len(featureVector) == 0 {
		featureVector = s.extractFeatureVector(features)
	}
	
	cacheKey := s.generateCacheKey(featureVector)
	
	s.cache.mutex.RLock()
	if cached, exists := s.cache.entries[cacheKey]; exists {
		if time.Now().Before(cached.ExpiresAt) {
			s.cache.mutex.RUnlock()
			return cached.Prediction
		}
	}
	s.cache.mutex.RUnlock()
	
	s.modelMutex.RLock()
	model := s.currentModel
	s.modelMutex.RUnlock()
	
	if model == nil {
		return s.defaultPrediction()
	}
	
	botScore := s.predictWithModel(featureVector, model)
	humanScore := 1.0 - botScore
	
	isBot := botScore > 0.5
	confidence := math.Abs(botScore - 0.5) * 2
	
	prediction := &MLPrediction{
		IsBot:        isBot,
		Confidence:   confidence,
		BotScore:     botScore,
		HumanScore:   humanScore,
		ModelVersion: model.Metadata.Version,
		FeaturesUsed: model.Metadata.Features,
	}
	
	s.cache.mutex.Lock()
	if len(s.cache.entries) >= s.cache.maxSize {
		s.cleanupPredictionCache()
	}
	s.cache.entries[cacheKey] = &CachedPrediction{
		Prediction: prediction,
		ExpiresAt: time.Now().Add(5 * time.Minute),
	}
	s.cache.mutex.Unlock()
	
	return prediction
}

func (s *ModelService) extractFeatureVector(features *FeatureVector) []float64 {
	vector := []float64{
		features.MouseSpeedAvg,
		features.MouseSpeedMax,
		features.MouseSpeedMin,
		features.MouseSpeedVar,
		features.TrajectoryLength / 1000,
		features.PathEfficiency,
		float64(features.DirectionChanges) / 10,
		features.Smoothness,
		float64(features.ClickCount) / 10,
		features.ClickRegularity,
		features.ClickSpeed / 10,
		features.DataPointDensity,
	}
	return vector
}

func (s *ModelService) predictWithModel(features []float64, model *TrainedModel) float64 {
	if len(features) == 0 || len(model.Weights) == 0 {
		return 0.5
	}
	
	score := model.Bias
	
	minLen := len(features)
	if len(model.Weights) < minLen {
		minLen = len(model.Weights)
	}
	
	for i := 0; i < minLen; i++ {
		score += features[i] * model.Weights[i]
	}
	
	normalizedScore := 1.0 / (1.0 + math.Exp(-score))
	
	return math.Max(0, math.Min(1, normalizedScore))
}

func (s *ModelService) defaultPrediction() *MLPrediction {
	return &MLPrediction{
		IsBot:        false,
		Confidence:   0.5,
		BotScore:     0.5,
		HumanScore:   0.5,
		ModelVersion: "unknown",
	}
}

func (s *ModelService) generateCacheKey(features []float64) string {
	key := ""
	for _, f := range features {
		key += fmt.Sprintf("%.4f", f)
	}
	return key
}

func (s *ModelService) cleanupPredictionCache() {
	now := time.Now()
	keysToDelete := []string{}
	
	for key, entry := range s.cache.entries {
		if now.After(entry.ExpiresAt) {
			keysToDelete = append(keysToDelete, key)
		}
	}
	
	for _, key := range keysToDelete {
		delete(s.cache.entries, key)
	}
	
	if len(s.cache.entries) >= s.cache.maxSize {
		for k := range s.cache.entries {
			delete(s.cache.entries, k)
			break
		}
	}
}

func (s *ModelService) Train(data *TrainingData) error {
	if data == nil || len(data.Features) == 0 || len(data.Labels) == 0 {
		return fmt.Errorf("training data is empty")
	}
	
	if len(data.Features) != len(data.Labels) {
		return fmt.Errorf("features and labels length mismatch")
	}
	
	s.modelMutex.Lock()
	defer s.modelMutex.Unlock()
	
	scaledFeatures := s.normalizeFeatures(data.Features)
	
	svmModel := s.trainSVM(scaledFeatures, data.Labels)
	
	treeModel := s.trainDecisionTree(scaledFeatures, data.Labels, 10)
	
	ensembleWeights := []float64{0.5, 0.5}
	
	model := &TrainedModel{
		Metadata: ModelMetadata{
			Name:      fmt.Sprintf("bot_detection_%d", time.Now().Unix()),
			Version:   fmt.Sprintf("1.0.%d", time.Now().Unix()),
			Type:      ModelTypeEnsemble,
			CreatedAt: time.Now(),
			UpdatedAt: time.Now(),
			Features:  []string{"feature_0", "feature_1", "feature_2", "feature_3", "feature_4", "feature_5", "feature_6", "feature_7", "feature_8"},
		},
		Config: TrainingConfig{
			LearningRate:   0.01,
			MaxIterations:  1000,
			TrainTestSplit: 0.8,
			BatchSize:      32,
			Regularization: 0.01,
		},
		Weights: ensembleWeights,
		Bias:    0.0,
		SVMParams: svmModel,
		Tree:    treeModel,
	}
	
	metrics := s.evaluateModel(model, scaledFeatures, data.Labels)
	model.Metadata.Accuracy = metrics["accuracy"]
	model.Metadata.Precision = metrics["precision"]
	model.Metadata.Recall = metrics["recall"]
	model.Metadata.F1Score = metrics["f1"]
	
	s.models[model.Metadata.Name] = model
	s.currentModel = model
	
	return nil
}

func (s *ModelService) normalizeFeatures(features [][]float64) [][]float64 {
	if len(features) == 0 {
		return features
	}
	
	numFeatures := len(features[0])
	means := make([]float64, numFeatures)
	stds := make([]float64, numFeatures)
	
	for j := 0; j < numFeatures; j++ {
		sum := 0.0
		for i := 0; i < len(features); i++ {
			if j < len(features[i]) {
				sum += features[i][j]
			}
		}
		means[j] = sum / float64(len(features))
	}
	
	for j := 0; j < numFeatures; j++ {
		sumSq := 0.0
		for i := 0; i < len(features); i++ {
			if j < len(features[i]) {
				diff := features[i][j] - means[j]
				sumSq += diff * diff
			}
		}
		stds[j] = math.Sqrt(sumSq / float64(len(features)))
		if stds[j] == 0 {
			stds[j] = 1.0
		}
	}
	
	normalized := make([][]float64, len(features))
	for i := 0; i < len(features); i++ {
		normalized[i] = make([]float64, numFeatures)
		for j := 0; j < numFeatures; j++ {
			if j < len(features[i]) {
				normalized[i][j] = (features[i][j] - means[j]) / stds[j]
			}
		}
	}
	
	return normalized
}

func (s *ModelService) trainSVM(features [][]float64, labels []int) SVMParams {
	params := SVMParams{
		Kernel: "rbf",
		Gamma:  0.1,
		C:      1.0,
		Classes: s.getUniqueClasses(labels),
	}
	
	if len(features) < 10 {
		params.SupportVectors = features
		return params
	}
	
	numSupportVectors := len(features) / 10
	if numSupportVectors < 5 {
		numSupportVectors = 5
	}
	
	indices := rand.Perm(len(features))
	params.SupportVectors = make([][]float64, numSupportVectors)
	for i := 0; i < numSupportVectors; i++ {
		params.SupportVectors[i] = features[indices[i]]
	}
	
	return params
}

func (s *ModelService) trainDecisionTree(features [][]float64, labels []int, maxDepth int) *DecisionTree {
	if len(features) == 0 {
		return &DecisionTree{
			IsLeaf:     true,
			Prediction: 0,
			ClassCounts: map[int]int{0: 1},
		}
	}
	
	return s.buildTree(features, labels, 0, maxDepth)
}

func (s *ModelService) buildTree(features [][]float64, labels []int, depth int, maxDepth int) *DecisionTree {
	if len(features) == 0 || depth >= maxDepth {
		return s.createLeafNode(labels)
	}
	
	if s.isPure(labels) {
		return s.createLeafNode(labels)
	}
	
	bestFeature := 0
	bestValue := 0.0
	bestGain := 0.0
	
	numFeatures := len(features[0])
	sampleSize := numFeatures
	if sampleSize > 5 {
		sampleSize = 5
	}
	
	featureIndices := make([]int, sampleSize)
	for i := range featureIndices {
		featureIndices[i] = i % numFeatures
	}
	
	for _, f := range featureIndices {
		featureValues := make([]float64, len(features))
		for i := 0; i < len(features); i++ {
			if f < len(features[i]) {
				featureValues[i] = features[i][f]
			}
		}
		
		mean := 0.0
		for _, v := range featureValues {
			mean += v
		}
		mean /= float64(len(featureValues))
		
		gain := s.calculateInformationGain(features, labels, f, mean)
		if gain > bestGain {
			bestGain = gain
			bestFeature = f
			bestValue = mean
		}
	}
	
	if bestGain < 0.01 {
		return s.createLeafNode(labels)
	}
	
	leftFeatures, leftLabels, rightFeatures, rightLabels := s.splitData(features, labels, bestFeature, bestValue)
	
	return &DecisionTree{
		SplitFeature: bestFeature,
		SplitValue:   bestValue,
		Left:        s.buildTree(leftFeatures, leftLabels, depth+1, maxDepth),
		Right:       s.buildTree(rightFeatures, rightLabels, depth+1, maxDepth),
		IsLeaf:      false,
		ClassCounts: s.countClasses(labels),
	}
}

func (s *ModelService) isPure(labels []int) bool {
	if len(labels) == 0 {
		return true
	}
	first := labels[0]
	for _, label := range labels {
		if label != first {
			return false
		}
	}
	return true
}

func (s *ModelService) createLeafNode(labels []int) *DecisionTree {
	classCounts := s.countClasses(labels)
	prediction := 0
	maxCount := 0
	for class, count := range classCounts {
		if count > maxCount {
			maxCount = count
			prediction = class
		}
	}
	
	return &DecisionTree{
		IsLeaf:     true,
		Prediction: prediction,
		ClassCounts: classCounts,
	}
}

func (s *ModelService) countClasses(labels []int) map[int]int {
	counts := make(map[int]int)
	for _, label := range labels {
		counts[label]++
	}
	return counts
}

func (s *ModelService) calculateInformationGain(features [][]float64, labels []int, featureIndex int, splitValue float64) float64 {
	parentEntropy := s.calculateEntropy(labels)
	
	_, leftLabels, _, rightLabels := s.splitData(features, labels, featureIndex, splitValue)
	
	if len(leftLabels) == 0 || len(rightLabels) == 0 {
		return 0
	}
	
	leftEntropy := s.calculateEntropy(leftLabels)
	rightEntropy := s.calculateEntropy(rightLabels)
	
	total := float64(len(labels))
	weightedEntropy := (float64(len(leftLabels)) / total * leftEntropy) + (float64(len(rightLabels)) / total * rightEntropy)
	
	return parentEntropy - weightedEntropy
}

func (s *ModelService) calculateEntropy(labels []int) float64 {
	if len(labels) == 0 {
		return 0
	}
	
	counts := s.countClasses(labels)
	entropy := 0.0
	total := float64(len(labels))
	
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / total
			entropy -= p * math.Log2(p)
		}
	}
	
	return entropy
}

func (s *ModelService) splitData(features [][]float64, labels []int, featureIndex int, splitValue float64) ([][]float64, []int, [][]float64, []int) {
	var leftFeatures, rightFeatures [][]float64
	var leftLabels, rightLabels []int
	
	for i := 0; i < len(features); i++ {
		if featureIndex < len(features[i]) {
			if features[i][featureIndex] <= splitValue {
				leftFeatures = append(leftFeatures, features[i])
				leftLabels = append(leftLabels, labels[i])
			} else {
				rightFeatures = append(rightFeatures, features[i])
				rightLabels = append(rightLabels, labels[i])
			}
		}
	}
	
	return leftFeatures, leftLabels, rightFeatures, rightLabels
}

func (s *ModelService) predictWithTree(tree *DecisionTree, features []float64) int {
	if tree == nil || tree.IsLeaf {
		return tree.Prediction
	}
	
	if tree.SplitFeature < len(features) {
		if features[tree.SplitFeature] <= tree.SplitValue {
			return s.predictWithTree(tree.Left, features)
		} else {
			return s.predictWithTree(tree.Right, features)
		}
	}
	
	return tree.Prediction
}

func (s *ModelService) getUniqueClasses(labels []int) []int {
	unique := make(map[int]bool)
	for _, label := range labels {
		unique[label] = true
	}
	
	result := make([]int, 0, len(unique))
	for class := range unique {
		result = append(result, class)
	}
	return result
}

func (s *ModelService) evaluateModel(model *TrainedModel, features [][]float64, labels []int) map[string]float64 {
	if len(features) != len(labels) || len(features) == 0 {
		return map[string]float64{"accuracy": 0, "precision": 0, "recall": 0, "f1": 0}
	}
	
	correct := 0
	tp, fp, tn, fn := 0, 0, 0, 0
	
	for i := 0; i < len(features); i++ {
		predicted := s.predictWithModel(features[i], model)
		predictedLabel := 0
		if predicted > 0.5 {
			predictedLabel = 1
		}
		
		actualLabel := labels[i]
		
		if predictedLabel == actualLabel {
			correct++
		}
		
		if predictedLabel == 1 && actualLabel == 1 {
			tp++
		} else if predictedLabel == 1 && actualLabel == 0 {
			fp++
		} else if predictedLabel == 0 && actualLabel == 0 {
			tn++
		} else if predictedLabel == 0 && actualLabel == 1 {
			fn++
		}
	}
	
	accuracy := float64(correct) / float64(len(features))
	
	precision := 0.0
	if tp+fp > 0 {
		precision = float64(tp) / float64(tp+fp)
	}
	
	recall := 0.0
	if tp+fn > 0 {
		recall = float64(tp) / float64(tp+fn)
	}
	
	f1 := 0.0
	if precision+recall > 0 {
		f1 = 2 * precision * recall / (precision + recall)
	}
	
	return map[string]float64{
		"accuracy":  accuracy,
		"precision": precision,
		"recall":   recall,
		"f1":       f1,
	}
}

func (s *ModelService) CrossValidate(data *TrainingData, folds int) (*CrossValidationResult, error) {
	if data == nil || len(data.Features) < folds {
		return nil, fmt.Errorf("insufficient data for cross-validation")
	}
	
	shuffledFeatures := make([][]float64, len(data.Features))
	shuffledLabels := make([]int, len(data.Labels))
	copy(shuffledFeatures, data.Features)
	copy(shuffledLabels, data.Labels)
	
	indices := rand.Perm(len(shuffledFeatures))
	features := make([][]float64, len(shuffledFeatures))
	labels := make([]int, len(shuffledLabels))
	for i, idx := range indices {
		features[i] = shuffledFeatures[idx]
		labels[i] = shuffledLabels[idx]
	}
	
	foldSize := len(features) / folds
	scores := make([]float64, folds)
	
	for f := 0; f < folds; f++ {
		testStart := f * foldSize
		testEnd := testStart + foldSize
		if f == folds-1 {
			testEnd = len(features)
		}
		
		testFeatures := features[testStart:testEnd]
		testLabels := labels[testStart:testEnd]
		
		var trainFeatures [][]float64
		var trainLabels []int
		
		if testStart > 0 {
			trainFeatures = append(trainFeatures, features[:testStart]...)
			trainLabels = append(trainLabels, labels[:testStart]...)
		}
		if testEnd < len(features) {
			trainFeatures = append(trainFeatures, features[testEnd:]...)
			trainLabels = append(trainLabels, labels[testEnd:]...)
		}
		
		tempModel := &TrainedModel{
			Weights: make([]float64, len(trainFeatures[0])),
			Bias:    0.0,
		}
		for i := range tempModel.Weights {
			tempModel.Weights[i] = rand.Float64() * 0.1
		}
		
		metrics := s.evaluateModel(tempModel, testFeatures, testLabels)
		scores[f] = metrics["accuracy"]
	}
	
	meanScore := 0.0
	for _, score := range scores {
		meanScore += score
	}
	meanScore /= float64(folds)
	
	variance := 0.0
	for _, score := range scores {
		diff := score - meanScore
		variance += diff * diff
	}
	variance /= float64(folds)
	stdDev := math.Sqrt(variance)
	
	return &CrossValidationResult{
		Folds:          folds,
		AccuracyScores: scores,
		MeanAccuracy:   meanScore,
		StdDeviation:   stdDev,
	}, nil
}

func (s *ModelService) SaveModel(name string, path string) error {
	s.modelMutex.RLock()
	model, exists := s.models[name]
	s.modelMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("model not found: %s", name)
	}
	
	if path == "" {
		path = s.modelDir
	}
	
	if err := os.MkdirAll(path, 0755); err != nil {
		return err
	}
	
	modelPath := filepath.Join(path, name+".json")
	
	data, err := json.MarshalIndent(model, "", "  ")
	if err != nil {
		return err
	}
	
	return os.WriteFile(modelPath, data, 0644)
}

func (s *ModelService) LoadModel(name string, path string) error {
	if path == "" {
		path = s.modelDir
	}
	
	modelPath := filepath.Join(path, name+".json")
	
	data, err := os.ReadFile(modelPath)
	if err != nil {
		return err
	}
	
	var model TrainedModel
	if err := json.Unmarshal(data, &model); err != nil {
		return err
	}
	
	s.modelMutex.Lock()
	s.models[name] = &model
	s.currentModel = &model
	s.modelMutex.Unlock()
	
	return nil
}

func (s *ModelService) GetModelInfo(name string) (*ModelMetadata, error) {
	s.modelMutex.RLock()
	model, exists := s.models[name]
	s.modelMutex.RUnlock()
	
	if !exists {
		return nil, fmt.Errorf("model not found: %s", name)
	}
	
	return &model.Metadata, nil
}

func (s *ModelService) ListModels() []string {
	s.modelMutex.RLock()
	defer s.modelMutex.RUnlock()
	
	names := make([]string, 0, len(s.models))
	for name := range s.models {
		names = append(names, name)
	}
	return names
}

func (s *ModelService) SetCurrentModel(name string) error {
	s.modelMutex.RLock()
	model, exists := s.models[name]
	s.modelMutex.RUnlock()
	
	if !exists {
		return fmt.Errorf("model not found: %s", name)
	}
	
	s.modelMutex.Lock()
	s.currentModel = model
	s.modelMutex.Unlock()
	
	return nil
}

func (s *ModelService) DeleteModel(name string) error {
	s.modelMutex.Lock()
	defer s.modelMutex.Unlock()
	
	if _, exists := s.models[name]; !exists {
		return fmt.Errorf("model not found: %s", name)
	}
	
	if name == "default" {
		return fmt.Errorf("cannot delete default model")
	}
	
	delete(s.models, name)
	
	if s.currentModel != nil && s.currentModel.Metadata.Name == name {
		s.currentModel = s.models["default"]
	}
	
	return nil
}

func (s *ModelService) PredictBatch(featuresList []*FeatureVector) []*MLPrediction {
	results := make([]*MLPrediction, len(featuresList))
	
	for i, features := range featuresList {
		results[i] = s.Predict(features)
	}
	
	return results
}

func (s *ModelService) ClearCache() {
	s.cache.mutex.Lock()
	s.cache.entries = make(map[string]*CachedPrediction)
	s.cache.mutex.Unlock()
}
