package trace

import (
	"context"
	"encoding/json"
	"errors"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type TraceService struct {
	extractor            *TraceExtractor
	matcher              *TraceMatcher
	lstmExtractor        *LSTMFeatureExtractor
	transformerPredictor *TransformerPredictor
	intentClassifier     *IntentClassifier
	anomalyDetector       *AnomalyDetector
	unifiedRiskScorer    *UnifiedRiskScorer
	nnService            interface{}
	enableNN             bool
	modelMonitor         *ModelPerformanceMonitor
	onlineUpdater        *OnlineModelUpdater
}

type NNAnalysisResult struct {
	RiskScore          float64            `json:"risk_score"`
	BotProbability     float64            `json:"bot_probability"`
	Confidence         float64            `json:"confidence"`
	NNFeatures         map[string]float64 `json:"nn_features"`
}

type ModelPerformanceMetrics struct {
	TotalPredictions   int64
	TruePositives     int64
	TrueNegatives     int64
	FalsePositives    int64
	FalseNegatives    int64
	TotalResponseTime time.Duration
	LastUpdated        time.Time
}

func (m *ModelPerformanceMetrics) Accuracy() float64 {
	total := m.TruePositives + m.TrueNegatives + m.FalsePositives + m.FalseNegatives
	if total == 0 {
		return 0.0
	}
	return float64(m.TruePositives + m.TrueNegatives) / float64(total)
}

func (m *ModelPerformanceMetrics) Precision() float64 {
	total := m.TruePositives + m.FalsePositives
	if total == 0 {
		return 0.0
	}
	return float64(m.TruePositives) / float64(total)
}

func (m *ModelPerformanceMetrics) Recall() float64 {
	total := m.TruePositives + m.FalseNegatives
	if total == 0 {
		return 0.0
	}
	return float64(m.TruePositives) / float64(total)
}

func (m *ModelPerformanceMetrics) F1Score() float64 {
	precision := m.Precision()
	recall := m.Recall()
	if precision + recall == 0 {
		return 0.0
	}
	return 2 * precision * recall / (precision + recall)
}

func (m *ModelPerformanceMetrics) AvgResponseTime() time.Duration {
	if m.TotalPredictions == 0 {
		return 0
	}
	return m.TotalResponseTime / time.Duration(m.TotalPredictions)
}

type ModelPerformanceMonitor struct {
	mu             sync.RWMutex
	lstmMetrics    *ModelPerformanceMetrics
	transformerMetrics *ModelPerformanceMetrics
}

func NewModelPerformanceMonitor() *ModelPerformanceMonitor {
	return &ModelPerformanceMonitor{
		lstmMetrics:    &ModelPerformanceMetrics{
			LastUpdated: time.Now(),
		},
		transformerMetrics: &ModelPerformanceMetrics{
			LastUpdated: time.Now(),
		},
	}
}

func (m *ModelPerformanceMonitor) RecordPrediction(modelType string, predicted, actual bool, responseTime time.Duration) {
	m.mu.Lock()
	defer m.mu.Unlock()

	var metrics *ModelPerformanceMetrics
	if modelType == "lstm" {
		metrics = m.lstmMetrics
	} else {
		metrics = m.transformerMetrics
	}

	metrics.TotalPredictions++
	metrics.TotalResponseTime += responseTime
	metrics.LastUpdated = time.Now()

	if predicted && actual {
		metrics.TruePositives++
	} else if !predicted && !actual {
		metrics.TrueNegatives++
	} else if predicted && !actual {
		metrics.FalsePositives++
	} else {
		metrics.FalseNegatives++
	}
}

func (m *ModelPerformanceMonitor) GetReport() map[string]interface{} {
	m.mu.RLock()
	defer m.mu.RUnlock()

	return map[string]interface{}{
		"lstm": map[string]interface{}{
			"accuracy":         m.lstmMetrics.Accuracy(),
			"precision":        m.lstmMetrics.Precision(),
			"recall":           m.lstmMetrics.Recall(),
			"f1_score":         m.lstmMetrics.F1Score(),
			"total_predictions":  m.lstmMetrics.TotalPredictions,
			"avg_response_time":  m.lstmMetrics.AvgResponseTime().Milliseconds(),
			"last_updated":       m.lstmMetrics.LastUpdated,
		},
		"transformer": map[string]interface{}{
			"accuracy":         m.transformerMetrics.Accuracy(),
			"precision":        m.transformerMetrics.Precision(),
			"recall":           m.transformerMetrics.Recall(),
			"f1_score":         m.transformerMetrics.F1Score(),
			"total_predictions":  m.transformerMetrics.TotalPredictions,
			"avg_response_time":  m.transformerMetrics.AvgResponseTime().Milliseconds(),
			"last_updated":       m.transformerMetrics.LastUpdated,
		},
	}
}

type TrajectorySample struct {
	TraceData   *model.TraceData
	IsBot        bool
	Confidence    float64
	Timestamp     time.Time
	FeatureVec    []float64
}

type OnlineModelUpdater struct {
	mu            sync.RWMutex
	traceService  *TraceService
	sampleQueue   []TrajectorySample
	isUpdating    bool
	stopChan      chan struct{}
	updateInterval time.Duration
	minSamplesForUpdate int
}

func NewOnlineModelUpdater(traceService *TraceService) *OnlineModelUpdater {
	return &OnlineModelUpdater{
		traceService:          traceService,
		sampleQueue:           make([]TrajectorySample, 0, 100),
		stopChan:              make(chan struct{}),
		updateInterval:         5 * time.Minute,
		minSamplesForUpdate:   10,
	}
}

func (o *OnlineModelUpdater) QueueSample(traceData *model.TraceData, isBot bool, confidence float64) {
	o.mu.Lock()
	defer o.mu.Unlock()

	o.sampleQueue = append(o.sampleQueue, TrajectorySample{
		TraceData: traceData,
		IsBot:     isBot,
		Confidence: confidence,
		Timestamp:  time.Now(),
	})
}

func (o *OnlineModelUpdater) Start() {
	o.mu.Lock()
	if o.isUpdating {
		o.mu.Unlock()
		return
	}
	o.isUpdating = true
	o.mu.Unlock()

	go o.runUpdateLoop()
}

func (o *OnlineModelUpdater) Stop() {
	o.mu.Lock()
	defer o.mu.Unlock()

	if o.isUpdating {
		close(o.stopChan)
		o.isUpdating = false
	}
}

func (o *OnlineModelUpdater) runUpdateLoop() {
	ticker := time.NewTicker(o.updateInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			o.processQueue()
		case <-o.stopChan:
			return
		}
	}
}

func (o *OnlineModelUpdater) processQueue() {
	o.mu.Lock()
	if len(o.sampleQueue) < o.minSamplesForUpdate {
		o.mu.Unlock()
		return
	}

	samples := make([]TrajectorySample, len(o.sampleQueue))
	copy(samples, o.sampleQueue)
	o.sampleQueue = o.sampleQueue[:0]
	o.mu.Unlock()

	for _, sample := range samples {
		o.updateModelWithSample(sample)
	}
}

func (o *OnlineModelUpdater) updateModelWithSample(sample TrajectorySample) {
	featureVec := sample.FeatureVec
	if len(featureVec) == 0 {
		featureMap, err := o.traceService.ExtractNNFeatures(context.Background(), sample.TraceData)
		if err != nil {
			return
		}
		featureVec = make([]float64, 0, len(featureMap))
		for _, v := range featureMap {
			featureVec = append(featureVec, v)
		}
	}

	if o.traceService.transformerPredictor != nil {
		prediction, err := o.traceService.transformerPredictor.PredictWithFeatures(featureVec)
		if err != nil {
			return
		}
		predictedIsBot := prediction.BotProbability > 0.5
		actualIsBot := sample.IsBot
		o.traceService.modelMonitor.RecordPrediction("transformer", predictedIsBot, actualIsBot, 0)
		o.adjustPredictionHead(featureVec, actualIsBot, predictedIsBot)
	}
}

func (o *OnlineModelUpdater) adjustPredictionHead(features []float64, actual, predicted bool) {
}

type TrajectoryVisualizationData struct {
	Points            []model.TracePoint `json:"points"`
	VelocityProfile   []map[string]interface{} `json:"velocity_profile"`
	AccelerationProfile []map[string]interface{} `json:"acceleration_profile"`
	Statistics       map[string]interface{} `json:"statistics"`
}

func (s *TraceService) PrepareVisualizationData(traceData *model.TraceData) (*TrajectoryVisualizationData, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("invalid trace data")
	}

	result := &TrajectoryVisualizationData{
		Points:          make([]model.TracePoint, len(traceData.Points)),
		VelocityProfile: make([]map[string]interface{}, 0),
		AccelerationProfile: make([]map[string]interface{}, 0),
		Statistics:    make(map[string]interface{}),
	}
	copy(result.Points, traceData.Points)

	velocities := make([]float64, 0, len(traceData.Points))
	accelerations := make([]float64, 0, len(traceData.Points))

	for i := 1; i < len(traceData.Points); i++ {
		dx := traceData.Points[i].X - traceData.Points[i-1].X
		dy := traceData.Points[i].Y - traceData.Points[i-1].Y
		dt := traceData.Points[i].Timestamp - traceData.Points[i-1].Timestamp
		if dt <= 0 {
			continue
		}
		distance := float64(dx*dx + dy*dy)
		velocity := float64(distance) / float64(dt)
		velocities = append(velocities, velocity)
		result.VelocityProfile = append(result.VelocityProfile, map[string]interface{}{
			"timestamp": traceData.Points[i].Timestamp,
			"velocity": velocity,
		})
	}

	for i := 1; i < len(velocities); i++ {
		dt := traceData.Points[i+1].Timestamp - traceData.Points[i].Timestamp
		if dt > 0 {
			acceleration := (velocities[i] - velocities[i-1]) / float64(dt)
			accelerations = append(accelerations, acceleration)
			result.AccelerationProfile = append(result.AccelerationProfile, map[string]interface{}{
				"timestamp":   traceData.Points[i+1].Timestamp,
				"acceleration": acceleration,
			})
		}
	}

	startPoint := traceData.Points[0]
	endPoint := traceData.Points[len(traceData.Points)-1]
	result.Statistics["total_duration_ms"] = endPoint.Timestamp - startPoint.Timestamp
	result.Statistics["point_count"] = len(traceData.Points)
	result.Statistics["start_x"] = startPoint.X
	result.Statistics["start_y"] = startPoint.Y
	result.Statistics["end_x"] = endPoint.X
	result.Statistics["end_y"] = endPoint.Y

	return result, nil
}

func NewTraceService() *TraceService {
	monitor := NewModelPerformanceMonitor()
	service := &TraceService{
		extractor:             NewTraceExtractor(),
		matcher:              NewTraceMatcher(),
		lstmExtractor:        NewLSTMFeatureExtractor(),
		transformerPredictor: NewTransformerPredictor(),
		intentClassifier:     NewIntentClassifier(),
		anomalyDetector:       NewAnomalyDetector(),
		unifiedRiskScorer:    NewUnifiedRiskScorer(),
		enableNN:              true,
		modelMonitor:          monitor,
	}
	service.onlineUpdater = NewOnlineModelUpdater(service)
	return service
}

func (s *TraceService) GetModelPerformanceReport() map[string]interface{} {
	if s.modelMonitor == nil {
		s.modelMonitor = NewModelPerformanceMonitor()
	}
	return s.modelMonitor.GetReport()
}

func (s *TraceService) QueueTrainingSample(traceData *model.TraceData, isBot bool, confidence float64) {
	if s.onlineUpdater == nil {
		s.onlineUpdater = NewOnlineModelUpdater(s)
	}
	s.onlineUpdater.QueueSample(traceData, isBot, confidence)
}

func (s *TraceService) StartOnlineUpdate() {
	if s.onlineUpdater == nil {
		s.onlineUpdater = NewOnlineModelUpdater(s)
	}
	s.onlineUpdater.Start()
}

func (s *TraceService) StopOnlineUpdate() {
	if s.onlineUpdater != nil {
		s.onlineUpdater.Stop()
	}
}

func (s *TraceService) RecordPrediction(modelType string, predicted, actual bool, responseTime time.Duration) {
	if s.modelMonitor == nil {
		s.modelMonitor = NewModelPerformanceMonitor()
	}
	s.modelMonitor.RecordPrediction(modelType, predicted, actual, responseTime)
}

func (s *TraceService) ExtractNNFeatures(ctx context.Context, traceData *model.TraceData) (map[string]float64, error) {
	features := make(map[string]float64)
	if s.lstmExtractor != nil {
		lstmFeatures, err := s.lstmExtractor.ExtractRiskFeatures(traceData)
		if err == nil {
			for k, v := range lstmFeatures {
				features[k] = v
			}
		}
	}
	return features, nil
}

func (s *TraceService) EnableNNAnalysis(enable bool) {
	s.enableNN = enable
}

func (s *TraceService) ProcessTrace(ctx context.Context, sessionID string, traceDataJSON []byte) (*model.TraceFeatures, *model.TraceScore, error) {
	var traceData model.TraceData
	if err := json.Unmarshal(traceDataJSON, &traceData); err != nil {
		return nil, nil, errors.New("轨迹数据格式错误: " + err.Error())
	}

	if len(traceData.Points) < 2 {
		return nil, nil, errors.New("轨迹数据点不足")
	}

	features, score, err := s.matcher.ExtractAndScore(&traceData)
	if err != nil {
		return nil, nil, err
	}

	features.SessionID = sessionID

	return features, score, nil
}

func (s *TraceService) ProcessTraceWithNN(ctx context.Context, sessionID string, traceDataJSON []byte) (*model.TraceFeatures, *model.TraceScore, *NNAnalysisResult, error) {
	features, score, err := s.ProcessTrace(ctx, sessionID, traceDataJSON)
	if err != nil {
		return nil, nil, nil, err
	}

	var nnResult *NNAnalysisResult
	if s.enableNN {
		nnResult = s.analyzeWithNN(ctx, traceDataJSON)
	}

	return features, score, nnResult, nil
}

func (s *TraceService) analyzeWithNN(ctx context.Context, traceDataJSON []byte) *NNAnalysisResult {
	result := &NNAnalysisResult{
		NNFeatures: make(map[string]float64),
	}

	var traceData model.TraceData
	if err := json.Unmarshal(traceDataJSON, &traceData); err != nil {
		return result
	}

	if s.lstmExtractor != nil {
		lstmFeatures, err := s.lstmExtractor.ExtractRiskFeatures(&traceData)
		if err == nil {
			for k, v := range lstmFeatures {
				result.NNFeatures["lstm_"+k] = v
			}
		}
	}

	if s.transformerPredictor != nil {
		transformerResult, err := s.transformerPredictor.PredictTrajectory(&traceData)
		if err == nil {
			result.RiskScore = transformerResult.RiskScore
			result.BotProbability = transformerResult.BotProbability
			result.Confidence = transformerResult.Confidence

			for k, v := range transformerResult.FeatureImportance {
				result.NNFeatures["transformer_"+k] = v
			}
		}
	}

	if result.RiskScore == 0 && len(result.NNFeatures) > 0 {
		var sum float64
		count := 0
		for _, v := range result.NNFeatures {
			if v > 0 && v < 1 {
				sum += v
				count++
			}
		}
		if count > 0 {
			result.RiskScore = sum / float64(count)
		}
	}

	return result
}



func (s *TraceService) PredictRiskScore(ctx context.Context, traceData *model.TraceData) (float64, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return 0.5, errors.New("轨迹数据点不足")
	}

	if s.transformerPredictor != nil {
		result, err := s.transformerPredictor.PredictTrajectory(traceData)
		if err == nil {
			return result.RiskScore, nil
		}
	}

	return 0.5, nil
}

func (s *TraceService) AnalyzeRiskLevel(features *model.TraceFeatures) (string, bool) {
	score := &model.TraceScore{
		TotalScore: 100 - float64(len(features.RiskFactors)*10),
	}
	return s.matcher.GetRiskLevel(score), s.matcher.IsBot(score)
}

func (s *TraceService) GetModelInfo() map[string]interface{} {
	info := make(map[string]interface{})

	info["nn_enabled"] = s.enableNN

	if s.lstmExtractor != nil {
		info["lstm_feature_dim"] = s.lstmExtractor.GetFeatureDimension()
	}

	if s.transformerPredictor != nil {
		info["transformer_embedding_dim"] = s.transformerPredictor.GetEmbeddingDimension()
		info["transformer_attention_heads"] = s.transformerPredictor.GetAttentionHeads()
	}

	info["intent_recognition_enabled"] = true
	info["anomaly_detection_enabled"] = true

	return info
}

func (s *TraceService) SetLSTMExtractor(extractor *LSTMFeatureExtractor) {
	s.lstmExtractor = extractor
}

func (s *TraceService) SetTransformerPredictor(predictor *TransformerPredictor) {
	s.transformerPredictor = predictor
}

func (s *TraceService) GetLSTMExtractor() *LSTMFeatureExtractor {
	return s.lstmExtractor
}

func (s *TraceService) GetTransformerPredictor() *TransformerPredictor {
	return s.transformerPredictor
}

func (s *TraceService) LoadModelWeights(ctx context.Context, lstmPath, transformerPath string) error {
	if s.lstmExtractor != nil && lstmPath != "" {
		if err := s.lstmExtractor.LoadModelWeights(lstmPath); err != nil {
			return err
		}
	}

	if s.transformerPredictor != nil && transformerPath != "" {
		if err := s.transformerPredictor.LoadModelWeights(transformerPath); err != nil {
			return err
		}
	}

	return nil
}

func (s *TraceService) GetLastUpdateTime() time.Time {
	return time.Now()
}

func (s *TraceService) ProcessTraceWithComprehensiveRisk(ctx context.Context, sessionID string, traceDataJSON []byte) (*ComprehensiveRiskResult, error) {
	var traceData model.TraceData
	if err := json.Unmarshal(traceDataJSON, &traceData); err != nil {
		return nil, errors.New("轨迹数据格式错误: " + err.Error())
	}

	if len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据点不足")
	}

	if s.unifiedRiskScorer != nil {
		return s.unifiedRiskScorer.AnalyzeComprehensiveRisk(ctx, &traceData)
	}

	result := &ComprehensiveRiskResult{
		TotalRiskScore:  0.5,
		BotProbability:  0.5,
		HumanProbability: 0.5,
		Confidence:     0.5,
	}

	return result, nil
}

func (s *TraceService) RecognizeIntent(traceData *model.TraceData) (*IntentRecognitionResult, error) {
	if s.intentClassifier == nil {
		s.intentClassifier = NewIntentClassifier()
	}
	return s.intentClassifier.RecognizeIntent(traceData)
}

func (s *TraceService) DetectAnomalies(traceData *model.TraceData) ([]AnomalyPattern, error) {
	if s.anomalyDetector == nil {
		s.anomalyDetector = NewAnomalyDetector()
	}
	return s.anomalyDetector.DetectAnomalies(traceData)
}

func (s *TraceService) GetComprehensiveRiskAnalysis(ctx context.Context, traceData *model.TraceData) (*ComprehensiveRiskResult, error) {
	if s.unifiedRiskScorer == nil {
		s.unifiedRiskScorer = NewUnifiedRiskScorer()
	}
	return s.unifiedRiskScorer.AnalyzeComprehensiveRisk(ctx, traceData)
}

func (s *TraceService) BatchComprehensiveRiskAnalysis(ctx context.Context, traces []*model.TraceData) ([]*ComprehensiveRiskResult, error) {
	if s.unifiedRiskScorer == nil {
		s.unifiedRiskScorer = NewUnifiedRiskScorer()
	}
	return s.unifiedRiskScorer.BatchAnalyze(ctx, traces)
}

func (s *TraceService) GetThresholds() map[string]float64 {
	if s.unifiedRiskScorer != nil {
		return s.unifiedRiskScorer.GetThresholds()
	}
	return make(map[string]float64)
}

func (s *TraceService) UpdateThreshold(name string, value float64) error {
	if s.unifiedRiskScorer != nil {
		return s.unifiedRiskScorer.UpdateThreshold(name, value)
	}
	return errors.New("unified risk scorer not initialized")
}

func (s *TraceService) ResetRiskScorer() {
	if s.unifiedRiskScorer != nil {
		s.unifiedRiskScorer.Reset()
	}
}

func (s *TraceService) ExtractTrajectoryComplexity(traceData *model.TraceData) (float64, error) {
	if s.lstmExtractor == nil {
		s.lstmExtractor = NewLSTMFeatureExtractor()
	}
	return s.lstmExtractor.AnalyzeTrajectoryComplexity(traceData)
}

func (s *TraceService) DetectAnomalousPatterns(traceData *model.TraceData) ([]string, error) {
	if s.lstmExtractor == nil {
		s.lstmExtractor = NewLSTMFeatureExtractor()
	}
	return s.lstmExtractor.DetectAnomalousPatterns(traceData)
}

func (s *TraceService) ExtractComprehensiveFeatures(traceData *model.TraceData) (*TraceFeatureSummary, error) {
	if s.lstmExtractor == nil {
		s.lstmExtractor = NewLSTMFeatureExtractor()
	}
	return s.lstmExtractor.ExtractComprehensiveFeatures(traceData)
}

func (s *TraceService) GetUnifiedRiskScorer() *UnifiedRiskScorer {
	return s.unifiedRiskScorer
}

func (s *TraceService) GetIntentClassifier() *IntentClassifier {
	return s.intentClassifier
}

func (s *TraceService) GetAnomalyDetector() *AnomalyDetector {
	return s.anomalyDetector
}
