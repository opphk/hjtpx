package ai

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"os"
	"sync"
	"time"
)

type Trainer struct {
	model             *MLPModel
	config            *ModelConfig
	featureExtractor  *FeatureExtractor
	normalizationParams *NormalizationParams
	mu                sync.RWMutex
	isTraining        bool
	stopChan          chan struct{}
}

func NewTrainer(config *ModelConfig) *Trainer {
	return &Trainer{
		model:             NewMLPModel(config),
		config:            config,
		featureExtractor:  NewFeatureExtractor(),
		normalizationParams: nil,
		isTraining:        false,
		stopChan:          make(chan struct{}),
	}
}

func (t *Trainer) Train(ctx context.Context, behaviors []*BehaviorData, labels []float64) (*TrainingResult, error) {
	if len(behaviors) == 0 || len(behaviors) != len(labels) {
		return nil, fmt.Errorf("invalid training data: behaviors and labels must have the same non-zero length")
	}

	t.mu.Lock()
	t.isTraining = true
	t.mu.Unlock()

	defer func() {
		t.mu.Lock()
		t.isTraining = false
		t.mu.Unlock()
	}()

	features := t.featureExtractor.BatchExtract(behaviors)

	t.normalizationParams = CalculateNormalizationParams(features)
	t.featureExtractor.SetNormalization(t.normalizationParams)

	normalizedFeatures := make([][]float64, len(features))
	for i, f := range features {
		normalizedFeatures[i] = t.featureExtractor.Normalize(f)
	}

	validationSplit := t.config.ValidationSplit
	if validationSplit <= 0 || validationSplit >= 1 {
		validationSplit = 0.2
	}

	indices := rand.Perm(len(normalizedFeatures))
	splitIdx := int(float64(len(indices)) * (1 - validationSplit))

	var trainFeatures, trainLabels [][]float64
	var valFeatures, valLabels [][]float64

	for i, idx := range indices {
		if i < splitIdx {
			trainFeatures = append(trainFeatures, normalizedFeatures[idx])
			trainLabels = append(trainLabels, []float64{labels[idx]})
		} else {
			valFeatures = append(valFeatures, normalizedFeatures[idx])
			valLabels = append(valLabels, []float64{labels[idx]})
		}
	}

	flatTrainLabels := make([]float64, len(trainLabels))
	for i, label := range trainLabels {
		flatTrainLabels[i] = label[0]
	}

	startTime := time.Now()
	result := t.model.nn.Train(trainFeatures, flatTrainLabels, t.config)
	result.TrainingTime = time.Since(startTime)

	if len(valFeatures) > 0 && len(valLabels) > 0 {
		flatValLabels := make([]float64, len(valLabels))
		for i, label := range valLabels {
			flatValLabels[i] = label[0]
		}
		result.ValidationLoss = t.evaluate(valFeatures, flatValLabels)
		result.ValidationAccuracy = t.calculateAccuracy(valFeatures, flatValLabels)
		result.TrainAccuracy = t.calculateAccuracy(trainFeatures, flatTrainLabels)
	}

	if t.config.ModelPath != "" {
		if err := t.model.SaveWeights(t.config.ModelPath); err != nil {
			fmt.Printf("Warning: failed to save model weights: %v\n", err)
		} else {
			result.BestModelPath = t.config.ModelPath
		}
	}

	return result, nil
}

func (t *Trainer) evaluate(features [][]float64, labels []float64) float64 {
	if len(features) == 0 || len(features) != len(labels) {
		return 0
	}

	totalLoss := 0.0
	for i, f := range features {
		pred := t.model.nn.Predict(f)
		loss := crossEntropyLoss(pred, labels[i])
		totalLoss += loss
	}

	return totalLoss / float64(len(features))
}

func (t *Trainer) calculateAccuracy(features [][]float64, labels []float64) float64 {
	if len(features) == 0 || len(features) != len(labels) {
		return 0
	}

	correct := 0
	for i, f := range features {
		pred := t.model.nn.Predict(f)
		predLabel := 0.0
		if pred >= 0.5 {
			predLabel = 1.0
		}
		if predLabel == labels[i] {
			correct++
		}
	}

	return float64(correct) / float64(len(features))
}

func (t *Trainer) Predict(ctx context.Context, behavior *BehaviorData) (*PredictionResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.model.IsReady() {
		return nil, fmt.Errorf("model is not ready")
	}

	extracted := t.featureExtractor.Extract(behavior)

	var features []float64
	if t.normalizationParams != nil {
		features = t.featureExtractor.Normalize(extracted.Features)
	} else {
		features = extracted.Features
	}

	score, err := t.model.Predict(ctx, features)
	if err != nil {
		return nil, err
	}

	riskLevel := getRiskLevelFromScore(score)
	action := getActionFromRiskLevel(riskLevel)

	return &PredictionResult{
		Score:      score,
		RiskLevel:  riskLevel,
		Action:     action,
		Confidence: calculateConfidence(score),
		Timestamp:  time.Now(),
		ModelType: string(t.model.GetModelType()),
		Features:  extracted.Features,
	}, nil
}

func (t *Trainer) PredictBatch(ctx context.Context, behaviors []*BehaviorData) ([]*PredictionResult, error) {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if !t.model.IsReady() {
		return nil, fmt.Errorf("model is not ready")
	}

	results := make([]*PredictionResult, len(behaviors))
	for i, behavior := range behaviors {
		extracted := t.featureExtractor.Extract(behavior)

		var features []float64
		if t.normalizationParams != nil {
			features = t.featureExtractor.Normalize(extracted.Features)
		} else {
			features = extracted.Features
		}

		score, err := t.model.Predict(ctx, features)
		if err != nil {
			results[i] = &PredictionResult{
				Score:     0.5,
				RiskLevel: RiskLevelMedium,
				Action:    ActionVerify,
			}
			continue
		}

		riskLevel := getRiskLevelFromScore(score)
		action := getActionFromRiskLevel(riskLevel)

		results[i] = &PredictionResult{
			Score:      score,
			RiskLevel:  riskLevel,
			Action:     action,
			Confidence: calculateConfidence(score),
			Timestamp:  time.Now(),
			ModelType:  string(t.model.GetModelType()),
			Features:   extracted.Features,
		}
	}

	return results, nil
}

func (t *Trainer) UpdateWithFeedback(ctx context.Context, behavior *BehaviorData, actualResult bool) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if !t.model.IsReady() || !t.isTraining {
		return fmt.Errorf("model is not ready for incremental training")
	}

	extracted := t.featureExtractor.Extract(behavior)

	var features []float64
	if t.normalizationParams != nil {
		features = t.featureExtractor.Normalize(extracted.Features)
	} else {
		features = extracted.Features
	}

	label := 0.0
	if actualResult {
		label = 1.0
	}

	t.model.nn.Train([][]float64{features}, []float64{label}, &ModelConfig{
		LearningRate: t.config.LearningRate * 2,
		BatchSize:    1,
		Epochs:       1,
	})

	return nil
}

func (t *Trainer) SaveModel(path string) error {
	t.mu.RLock()
	defer t.mu.RUnlock()

	if err := t.model.SaveWeights(path); err != nil {
		return err
	}

	paramsPath := path + ".params"
	file, err := os.Create(paramsPath)
	if err != nil {
		return err
	}
	defer file.Close()

	if t.normalizationParams != nil {
		for i, mean := range t.normalizationParams.Mean {
			fmt.Fprintf(file, "mean:%d=%.6f\n", i, mean)
		}
		for i, std := range t.normalizationParams.Std {
			fmt.Fprintf(file, "std:%d=%.6f\n", i, std)
		}
	}

	return nil
}

func (t *Trainer) LoadModel(path string) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	if err := t.model.LoadWeights(path); err != nil {
		return err
	}

	paramsPath := path + ".params"
	file, err := os.Open(paramsPath)
	if err == nil {
		defer file.Close()

		params := &NormalizationParams{
			Mean: make([]float64, FeatureDimension),
			Std:  make([]float64, FeatureDimension),
		}

		var meanCount, stdCount int
		for {
			var paramType string
			var idx int
			var value float64
			_, err := fmt.Fscanf(file, "%[^:]:%d=%f\n", &paramType, &idx, &value)
			if err != nil {
				break
			}
			if paramType == "mean" && idx < FeatureDimension {
				params.Mean[idx] = value
				meanCount++
			} else if paramType == "std" && idx < FeatureDimension {
				params.Std[idx] = value
				stdCount++
			}
		}

		if meanCount == FeatureDimension && stdCount == FeatureDimension {
			t.normalizationParams = params
			t.featureExtractor.SetNormalization(params)
		}
	}

	t.model.SetReady(true)
	return nil
}

func (t *Trainer) IsReady() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.model.IsReady()
}

func (t *Trainer) IsTraining() bool {
	t.mu.RLock()
	defer t.mu.RUnlock()
	return t.isTraining
}

func (t *Trainer) GetModel() *MLPModel {
	return t.model
}

func (t *Trainer) Stop() {
	close(t.stopChan)
}

func (t *Trainer) GenerateSyntheticData(count int, isBot bool) ([]*BehaviorData, []float64) {
	behaviors := make([]*BehaviorData, count)
	labels := make([]float64, count)

	for i := 0; i < count; i++ {
		behavior := generateSyntheticBehavior(isBot)
		behaviors[i] = behavior
		labels[i] = 0.0
		if isBot {
			labels[i] = 1.0
		}
	}

	return behaviors, labels
}

func generateSyntheticBehavior(isBot bool) *BehaviorData {
	trackCount := 50
	if isBot {
		trackCount = 100
	}

	startTime := int64(time.Now().UnixMilli())
	var tracks []MouseTrack

	startX := 100.0
	startY := 200.0
	currentX := startX
	currentY := startY

	for i := 0; i < trackCount; i++ {
		timestamp := startTime + int64(i*20)

		var dx, dy float64
		if isBot {
			dx = 2.0 + rand.Float64()*0.1
			dy = math.Sin(float64(i)*0.1) * 2.0
		} else {
			dx = rand.Float64()*10 - 5
			dy = rand.Float64()*10 - 5
			if i%10 == 0 {
				dx *= 3
				dy *= 3
			}
		}

		currentX += dx
		currentY += dy

		track := MouseTrack{
			X:         currentX,
			Y:         currentY,
			Timestamp: timestamp,
			Velocity:  math.Sqrt(dx*dx + dy*dy) / 0.02,
		}

		if isBot {
			track.Velocity = 150 + rand.Float64()*50
		} else {
			track.Velocity = 100 + rand.Float64()*200
		}

		tracks = append(tracks, track)
	}

	var clickEvents []ClickEvent
	clickCount := 3
	if isBot {
		clickCount = 6
	}

	for i := 0; i < clickCount; i++ {
		clickIdx := (len(tracks) / (clickCount + 1)) * (i + 1)
		if clickIdx < len(tracks) {
			click := ClickEvent{
				Timestamp: tracks[clickIdx].Timestamp,
				X:         tracks[clickIdx].X,
				Y:         tracks[clickIdx].Y,
				Pressure:  0.5,
				Duration:  100,
			}
			if isBot {
				click.Pressure = 1.0
				click.Duration = 50
			} else {
				click.Pressure = 0.3 + rand.Float64()*0.6
				click.Duration = 50 + int64(rand.Intn(200))
			}
			clickEvents = append(clickEvents, click)
		}
	}

	slideDuration := int64(5000)
	if isBot {
		slideDuration = int64(500 + rand.Intn(300))
	}

	return &BehaviorData{
		UserID:       fmt.Sprintf("user_%d", rand.Intn(10000)),
		SessionID:    fmt.Sprintf("session_%d", rand.Intn(10000)),
		MouseTracks:  tracks,
		ClickEvents:  clickEvents,
		ClickTimes:   extractClickTimes(clickEvents),
		SlideStart:   startTime,
		SlideEnd:     startTime + slideDuration,
		Success:      !isBot,
	}
}

func extractClickTimes(clicks []ClickEvent) []int64 {
	times := make([]int64, len(clicks))
	for i, click := range clicks {
		times[i] = click.Timestamp
	}
	return times
}

func getRiskLevelFromScore(score float64) RiskLevel {
	switch {
	case score >= 0.8:
		return RiskLevelCritical
	case score >= 0.5:
		return RiskLevelHigh
	case score >= 0.25:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

func getActionFromRiskLevel(level RiskLevel) Action {
	switch level {
	case RiskLevelLow:
		return ActionAllow
	case RiskLevelMedium:
		return ActionVerify
	case RiskLevelHigh:
		return ActionVerify
	case RiskLevelCritical:
		return ActionBlock
	default:
		return ActionVerify
	}
}

func crossEntropyLoss(predicted, actual float64) float64 {
	eps := Epsilon
	predicted = math.Max(eps, math.Min(1-eps, predicted))
	return -actual*math.Log(predicted) - (1-actual)*math.Log(1-predicted)
}

func CalculateMetrics(predictions []float64, labels []float64) *ValidationMetrics {
	if len(predictions) == 0 || len(predictions) != len(labels) {
		return nil
	}

	metrics := &ValidationMetrics{
		ConfusionMatrix: make([][]int, 2),
	}

	for i := 0; i < 2; i++ {
		metrics.ConfusionMatrix[i] = make([]int, 2)
	}

	tp, tn, fp, fn := 0, 0, 0, 0

	for i, pred := range predictions {
		predLabel := 0.0
		if pred >= 0.5 {
			predLabel = 1.0
		}

		actual := labels[i]

		if predLabel == 1.0 && actual == 1.0 {
			tp++
		} else if predLabel == 0.0 && actual == 0.0 {
			tn++
		} else if predLabel == 1.0 && actual == 0.0 {
			fp++
		} else if predLabel == 0.0 && actual == 1.0 {
			fn++
		}
	}

	metrics.ConfusionMatrix[0][0] = tn
	metrics.ConfusionMatrix[0][1] = fp
	metrics.ConfusionMatrix[1][0] = fn
	metrics.ConfusionMatrix[1][1] = tp

	total := tp + tn + fp + fn
	if total > 0 {
		metrics.Accuracy = float64(tp+tn) / float64(total)
	}

	if (tp + fp) > 0 {
		metrics.Precision = float64(tp) / float64(tp+fp)
	}

	if (tp + fn) > 0 {
		metrics.Recall = float64(tp) / float64(tp+fn)
	}

	if metrics.Precision+metrics.Recall > 0 {
		metrics.F1Score = 2 * (metrics.Precision * metrics.Recall) / (metrics.Precision + metrics.Recall)
	}

	metrics.AUC = calculateAUC(predictions, labels)

	return metrics
}

func calculateAUC(predictions []float64, labels []float64) float64 {
	if len(predictions) == 0 || len(predictions) != len(labels) {
		return 0
	}

	pairs := make([]struct {
		prediction float64
		label      float64
	}, len(predictions))

	for i := range predictions {
		pairs[i] = struct {
			prediction float64
			label      float64
		}{predictions[i], labels[i]}
	}

	sortedPairs := sortByPrediction(pairs)

	positives := 0.0
	for _, p := range pairs {
		if p.label == 1.0 {
			positives++
		}
	}

	if positives == 0 || positives == float64(len(pairs)) {
		return 0.5
	}

	tp := 0.0
	auc := 0.0
	prevPred := sortedPairs[0].prediction

	for _, pair := range sortedPairs {
		if pair.prediction != prevPred {
			auc += float64(tp) * (float64(tp) / positives)
			prevPred = pair.prediction
			tp = 0
		}
		if pair.label == 1.0 {
			tp++
		}
	}

	return auc / (positives * positives)
}

func sortByPrediction(pairs []struct {
	prediction float64
	label      float64
}) []struct {
	prediction float64
	label      float64
} {
	sorted := make([]struct {
		prediction float64
		label      float64
	}, len(pairs))
	copy(sorted, pairs)

	for i := 0; i < len(sorted); i++ {
		for j := i + 1; j < len(sorted); j++ {
			if sorted[j].prediction > sorted[i].prediction {
				sorted[i], sorted[j] = sorted[j], sorted[i]
			}
		}
	}

	return sorted
}
