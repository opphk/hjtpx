package trace

import (
	"context"
	"encoding/json"
	"errors"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type TraceService struct {
	extractor          *TraceExtractor
	matcher            *TraceMatcher
	lstmExtractor      *LSTMFeatureExtractor
	transformerPredictor *TransformerPredictor
	nnService          interface{}
	enableNN           bool
}

type NNAnalysisResult struct {
	RiskScore          float64            `json:"risk_score"`
	BotProbability     float64            `json:"bot_probability"`
	Confidence         float64            `json:"confidence"`
	NNFeatures         map[string]float64 `json:"nn_features"`
}

func NewTraceService() *TraceService {
	return &TraceService{
		extractor:            NewTraceExtractor(),
		matcher:              NewTraceMatcher(),
		lstmExtractor:        NewLSTMFeatureExtractor(),
		transformerPredictor: NewTransformerPredictor(),
		enableNN:             true,
	}
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

func (s *TraceService) ExtractNNFeatures(ctx context.Context, traceData *model.TraceData) (map[string]float64, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据点不足")
	}

	features := make(map[string]float64)

	if s.lstmExtractor != nil {
		lstmFeatures, err := s.lstmExtractor.ExtractRiskFeatures(traceData)
		if err == nil {
			for k, v := range lstmFeatures {
				features["lstm_"+k] = v
			}
		}
	}

	if s.transformerPredictor != nil {
		transformerResult, err := s.transformerPredictor.PredictTrajectory(traceData)
		if err == nil {
			for k, v := range transformerResult.FeatureImportance {
				features["transformer_"+k] = v
			}
		}
	}

	return features, nil
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
	}

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
