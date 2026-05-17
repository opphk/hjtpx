package trace

import (
	"context"
	"encoding/json"
	"errors"

	"github.com/hjtpx/hjtpx/internal/model"
)

type TraceService struct {
	extractor *TraceExtractor
	matcher   *TraceMatcher
}

func NewTraceService() *TraceService {
	return &TraceService{
		extractor: NewTraceExtractor(),
		matcher:   NewTraceMatcher(),
	}
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

func (s *TraceService) AnalyzeRiskLevel(features *model.TraceFeatures) (string, bool) {
	score := &model.TraceScore{
		TotalScore: 100 - float64(len(features.RiskFactors)*10),
	}
	return s.matcher.GetRiskLevel(score), s.matcher.IsBot(score)
}
