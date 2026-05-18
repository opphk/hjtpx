package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"os"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/service/trace"
)

type ModelConfig struct {
	LSTMWeightsPath     string
	TransformerWeightsPath string
	EnableLSTM          bool
	EnableTransformer   bool
	ConfidenceThreshold float64
	RiskThreshold       float64
}

type TrajectoryNNService struct {
	lstmExtractor     *trace.LSTMFeatureExtractor
	transformerPredictor *trace.TransformerPredictor
	config             *ModelConfig
	mu                 sync.RWMutex
	isLoaded           bool
	lastUpdateTime     time.Time
}

type RiskPredictionResult struct {
	TotalRiskScore       float64            `json:"total_risk_score"`
	LSTMFeatureScore    float64            `json:"lstm_feature_score"`
	TransformerScore    float64            `json:"transformer_score"`
	CombinedScore       float64            `json:"combined_score"`
	BotProbability      float64            `json:"bot_probability"`
	HumanProbability    float64            `json:"human_probability"`
	Confidence          float64            `json:"confidence"`
	RiskLevel           string             `json:"risk_level"`
	IsBot               bool               `json:"is_bot"`
	LSTMFeatures        map[string]float64 `json:"lstm_features"`
	TransformerFeatures map[string]float64 `json:"transformer_features"`
	FeatureImportance   map[string]float64 `json:"feature_importance"`
	Warnings            []string           `json:"warnings"`
}

func NewTrajectoryNNService() *TrajectoryNNService {
	return &TrajectoryNNService{
		lstmExtractor:       trace.NewLSTMFeatureExtractor(),
		transformerPredictor: trace.NewTransformerPredictor(),
		config: &ModelConfig{
			EnableLSTM:          true,
			EnableTransformer:   true,
			ConfidenceThreshold: 0.6,
			RiskThreshold:       0.5,
		},
		isLoaded: false,
	}
}

func (s *TrajectoryNNService) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.lstmExtractor == nil {
		s.lstmExtractor = trace.NewLSTMFeatureExtractor()
	}

	if s.transformerPredictor == nil {
		s.transformerPredictor = trace.NewTransformerPredictor()
	}

	s.isLoaded = true
	s.lastUpdateTime = time.Now()

	return nil
}

func (s *TrajectoryNNService) LoadModelWeights(ctx context.Context, config *ModelConfig) error {
	if config == nil {
		return errors.New("模型配置不能为空")
	}

	s.mu.Lock()
	defer s.mu.Unlock()

	s.config = config

	if config.EnableLSTM && s.lstmExtractor != nil {
		if err := s.lstmExtractor.LoadModelWeights(config.LSTMWeightsPath); err != nil {
			return fmt.Errorf("加载LSTM模型权重失败: %w", err)
		}
	}

	if config.EnableTransformer && s.transformerPredictor != nil {
		if err := s.transformerPredictor.LoadModelWeights(config.TransformerWeightsPath); err != nil {
			return fmt.Errorf("加载Transformer模型权重失败: %w", err)
		}
	}

	s.isLoaded = true
	s.lastUpdateTime = time.Now()

	return nil
}

func (s *TrajectoryNNService) ExtractFeatures(ctx context.Context, traceData *model.TraceData) ([]float64, error) {
	if traceData == nil {
		return nil, errors.New("轨迹数据不能为空")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isLoaded {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	features, err := s.lstmExtractor.ExtractFeatures(traceData)
	if err != nil {
		return nil, fmt.Errorf("特征提取失败: %w", err)
	}

	return features, nil
}

func (s *TrajectoryNNService) PredictRisk(ctx context.Context, traceDataJSON []byte) (*RiskPredictionResult, error) {
	var traceData model.TraceData
	if err := json.Unmarshal(traceDataJSON, &traceData); err != nil {
		return nil, fmt.Errorf("轨迹数据解析失败: %w", err)
	}

	return s.PredictRiskFromData(ctx, &traceData)
}

func (s *TrajectoryNNService) PredictRiskFromData(ctx context.Context, traceData *model.TraceData) (*RiskPredictionResult, error) {
	if traceData == nil {
		return nil, errors.New("轨迹数据不能为空")
	}

	s.mu.RLock()
	defer s.mu.RUnlock()

	if !s.isLoaded {
		if err := s.Initialize(ctx); err != nil {
			return nil, err
		}
	}

	result := &RiskPredictionResult{
		LSTMFeatures:      make(map[string]float64),
		TransformerFeatures: make(map[string]float64),
		FeatureImportance: make(map[string]float64),
		Warnings:          []string{},
	}

	if s.config.EnableLSTM && s.lstmExtractor != nil {
		lstmFeatures, err := s.lstmExtractor.ExtractRiskFeatures(traceData)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("LSTM特征提取失败: %v", err))
		} else {
			for k, v := range lstmFeatures {
				result.LSTMFeatures[k] = v
			}

			var lstmScore float64
			count := 0
			for _, v := range lstmFeatures {
				if v > 0 && v < 10 {
					lstmScore += v
					count++
				}
			}
			if count > 0 {
				result.LSTMFeatureScore = lstmScore / float64(count)
			}
		}
	}

	if s.config.EnableTransformer && s.transformerPredictor != nil {
		transformerResult, err := s.transformerPredictor.PredictTrajectory(traceData)
		if err != nil {
			result.Warnings = append(result.Warnings, fmt.Sprintf("Transformer预测失败: %v", err))
		} else {
			result.TransformerScore = transformerResult.RiskScore
			result.BotProbability = transformerResult.BotProbability
			result.HumanProbability = transformerResult.HumanProbability
			result.Confidence = transformerResult.Confidence

			for k, v := range transformerResult.FeatureImportance {
				result.TransformerFeatures[k] = v
			}
		}
	}

	result.CombinedScore = s.combineScores(result.LSTMFeatureScore, result.TransformerScore)
	result.TotalRiskScore = result.CombinedScore

	if result.CombinedScore >= 0.7 {
		result.RiskLevel = "critical"
		result.IsBot = true
	} else if result.CombinedScore >= 0.5 {
		result.RiskLevel = "high"
		result.IsBot = true
	} else if result.CombinedScore >= 0.3 {
		result.RiskLevel = "medium"
		result.IsBot = false
	} else {
		result.RiskLevel = "low"
		result.IsBot = false
	}

	if result.BotProbability > 0.7 {
		result.IsBot = true
		result.RiskLevel = "critical"
	}

	result.FeatureImportance = s.computeFeatureImportance(traceData, result)

	return result, nil
}

func (s *TrajectoryNNService) combineScores(lstmScore, transformerScore float64) float64 {
	if !s.config.EnableLSTM && !s.config.EnableTransformer {
		return 0.5
	}

	if !s.config.EnableLSTM {
		return transformerScore
	}

	if !s.config.EnableTransformer {
		return lstmScore
	}

	lstmWeight := 0.4
	transformerWeight := 0.6

	return lstmScore*lstmWeight + transformerScore*transformerWeight
}

func (s *TrajectoryNNService) computeFeatureImportance(traceData *model.TraceData, result *RiskPredictionResult) map[string]float64 {
	importance := make(map[string]float64)

	if len(traceData.Points) > 0 {
		importance["point_count"] = float64(len(traceData.Points))

		var totalDist float64
		for i := 1; i < len(traceData.Points); i++ {
			dx := traceData.Points[i].X - traceData.Points[i-1].X
			dy := traceData.Points[i].Y - traceData.Points[i-1].Y
			totalDist += math.Sqrt(dx*dx + dy*dy)
		}
		importance["total_distance"] = totalDist
	}

	if val, ok := result.LSTMFeatures["velocity_mean"]; ok {
		importance["lstm_velocity_mean"] = val
	}
	if val, ok := result.LSTMFeatures["acceleration_mean"]; ok {
		importance["lstm_acceleration_mean"] = val
	}
	if val, ok := result.LSTMFeatures["direction_change_rate"]; ok {
		importance["lstm_direction_change"] = val
	}

	if val, ok := result.TransformerFeatures["velocity_pattern"]; ok {
		importance["transformer_velocity_pattern"] = val
	}
	if val, ok := result.TransformerFeatures["sequence_length"]; ok {
		importance["transformer_seq_length"] = val
	}

	return importance
}

func (s *TrajectoryNNService) GetConfig() *ModelConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()

	configCopy := *s.config
	return &configCopy
}

func (s *TrajectoryNNService) UpdateConfig(config *ModelConfig) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if config == nil {
		return errors.New("配置不能为空")
	}

	s.config = config
	s.lastUpdateTime = time.Now()

	return nil
}

func (s *TrajectoryNNService) IsLoaded() bool {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.isLoaded
}

func (s *TrajectoryNNService) GetLastUpdateTime() time.Time {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.lastUpdateTime
}

func (s *TrajectoryNNService) ValidateModelFiles() error {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if s.config.EnableLSTM && s.config.LSTMWeightsPath != "" {
		if _, err := os.Stat(s.config.LSTMWeightsPath); os.IsNotExist(err) {
			return fmt.Errorf("LSTM模型文件不存在: %s", s.config.LSTMWeightsPath)
		}
	}

	if s.config.EnableTransformer && s.transformerPredictor != nil {
		if _, err := os.Stat(s.config.TransformerWeightsPath); os.IsNotExist(err) {
			return fmt.Errorf("Transformer模型文件不存在: %s", s.config.TransformerWeightsPath)
		}
	}

	return nil
}

func (s *TrajectoryNNService) GetModelInfo() map[string]interface{} {
	s.mu.RLock()
	defer s.mu.RUnlock()

	info := make(map[string]interface{})

	info["is_loaded"] = s.isLoaded
	info["last_update"] = s.lastUpdateTime
	info["lstm_enabled"] = s.config.EnableLSTM
	info["transformer_enabled"] = s.config.EnableTransformer

	if s.lstmExtractor != nil {
		info["lstm_feature_dim"] = s.lstmExtractor.GetFeatureDimension()
	}

	if s.transformerPredictor != nil {
		info["transformer_embedding_dim"] = s.transformerPredictor.GetEmbeddingDimension()
	}

	return info
}

func (s *TrajectoryNNService) AnalyzeTrajectory(ctx context.Context, traceData *model.TraceData) (*RiskPredictionResult, error) {
	return s.PredictRiskFromData(ctx, traceData)
}

func (s *TrajectoryNNService) BatchPredict(ctx context.Context, traces [][]byte) ([]*RiskPredictionResult, error) {
	results := make([]*RiskPredictionResult, 0, len(traces))

	for i, traceData := range traces {
		result, err := s.PredictRisk(ctx, traceData)
		if err != nil {
			return nil, fmt.Errorf("批量预测第%d条轨迹失败: %w", i, err)
		}
		results = append(results, result)
	}

	return results, nil
}

func (s *TrajectoryNNService) Shutdown() error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lstmExtractor = nil
	s.transformerPredictor = nil
	s.isLoaded = false

	return nil
}
