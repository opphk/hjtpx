package service

import (
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

type RTBehaviorAnomaly struct {
	Type        string  `json:"type"`
	Severity    string  `json:"severity"`
	Score       float64 `json:"score"`
	Description string  `json:"description"`
}

type RTBehaviorPattern struct {
	Name       string  `json:"name"`
	Confidence float64 `json:"confidence"`
	MatchCount int     `json:"match_count"`
}

type RTPredictionResult struct {
	RiskScore   float64            `json:"risk_score"`
	Confidence  float64            `json:"confidence"`
	Anomalies   []RTBehaviorAnomaly `json:"anomalies"`
	BehaviorType string             `json:"behavior_type"`
	Patterns    []RTBehaviorPattern `json:"patterns"`
}

type RTRiskFactor struct {
	Name        string  `json:"name"`
	Contribution float64 `json:"contribution"`
	Description string  `json:"description"`
}

type RTRiskAssessment struct {
	RiskLevel       string          `json:"risk_level"`
	OverallRisk     float64         `json:"overall_risk"`
	RiskFactors     []RTRiskFactor  `json:"risk_factors"`
	Recommendations []string        `json:"recommendations"`
	Details         map[string]float64 `json:"details"`
}

type PredictionMetrics struct {
	TotalPredictions int     `json:"total_predictions"`
	AvgRiskScore     float64 `json:"avg_risk_score"`
	HighRiskCount    int     `json:"high_risk_count"`
	AvgLatencyMs     float64 `json:"avg_latency_ms"`
}

type RealtimeBehaviorPrediction struct {
	predictionHistory []*RTPredictionResult
	maxHistorySize    int
	totalPredictions  int
	totalRiskScore    float64
	highRiskCount     int
	totalLatencyNs    int64
	mu                sync.RWMutex
}

func NewRealtimeBehaviorPrediction() *RealtimeBehaviorPrediction {
	return &RealtimeBehaviorPrediction{
		predictionHistory: []*RTPredictionResult{},
		maxHistorySize:    100,
	}
}

func (p *RealtimeBehaviorPrediction) Predict(traceData *model.TraceData) (*RTPredictionResult, error) {
	startTime := time.Now()
	
	result := &RTPredictionResult{
		RiskScore:   0,
		Confidence:  0,
		Anomalies:   []RTBehaviorAnomaly{},
		BehaviorType: "normal",
		Patterns:    []RTBehaviorPattern{},
	}

	anomalies := p.detectAnomalies(traceData)
	result.Anomalies = anomalies

	totalRisk := 0.0
	totalWeight := 0.0
	for _, anomaly := range anomalies {
		severityWeight := 1.0
		switch anomaly.Severity {
		case "high":
			severityWeight = 1.0
		case "medium":
			severityWeight = 0.6
		case "low":
			severityWeight = 0.3
		}
		totalRisk += anomaly.Score * severityWeight
		totalWeight += severityWeight
	}

	if totalWeight > 0 {
		result.RiskScore = totalRisk / totalWeight
	} else {
		result.RiskScore = 0.1 + rand.Float64()*0.2
	}

	if len(anomalies) > 0 {
		result.Confidence = 0.7 + rand.Float64()*0.3
	} else {
		result.Confidence = 0.8 + rand.Float64()*0.2
	}

	if result.RiskScore > 0.7 {
		result.BehaviorType = "suspicious"
	} else if result.RiskScore > 0.4 {
		result.BehaviorType = "abnormal"
	}

	latency := time.Since(startTime)
	
	p.mu.Lock()
	p.totalPredictions++
	p.totalRiskScore += result.RiskScore
	if result.RiskScore > 0.7 {
		p.highRiskCount++
	}
	p.totalLatencyNs += latency.Nanoseconds()
	p.predictionHistory = append(p.predictionHistory, result)
	if len(p.predictionHistory) > p.maxHistorySize {
		p.predictionHistory = p.predictionHistory[1:]
	}
	p.mu.Unlock()

	return result, nil
}

func (p *RealtimeBehaviorPrediction) detectAnomalies(traceData *model.TraceData) []RTBehaviorAnomaly {
	anomalies := []RTBehaviorAnomaly{}
	pointCount := len(traceData.Points)

	if pointCount > 100 {
		anomalies = append(anomalies, RTBehaviorAnomaly{
			Type:        "high_velocity",
			Severity:    "high",
			Score:       0.9,
			Description: "Unusually high event frequency detected",
		})
	} else if pointCount > 50 {
		anomalies = append(anomalies, RTBehaviorAnomaly{
			Type:        "medium_velocity",
			Severity:    "medium",
			Score:       0.5,
			Description: "Moderately high event frequency",
		})
	}

	if traceData.TotalTime > 0 && pointCount > 1 {
		avgTimeBetween := float64(traceData.TotalTime) / float64(pointCount-1)
		if avgTimeBetween < 50 {
			anomalies = append(anomalies, RTBehaviorAnomaly{
				Type:        "rapid_fire",
				Severity:    "high",
				Score:       0.8,
				Description: "Abnormally rapid event sequence detected",
			})
		}
	}

	return anomalies
}

func (p *RealtimeBehaviorPrediction) AssessRisk(traceData *model.TraceData) (*RTRiskAssessment, error) {
	prediction, err := p.Predict(traceData)
	if err != nil {
		return nil, err
	}

	assessment := &RTRiskAssessment{
		RiskLevel:       "minimal",
		OverallRisk:     prediction.RiskScore,
		RiskFactors:     []RTRiskFactor{},
		Recommendations: []string{},
		Details:         map[string]float64{},
	}

	switch {
	case prediction.RiskScore >= 0.8:
		assessment.RiskLevel = "high"
	case prediction.RiskScore >= 0.5:
		assessment.RiskLevel = "medium"
	case prediction.RiskScore >= 0.3:
		assessment.RiskLevel = "low"
	default:
		assessment.RiskLevel = "minimal"
	}

	for _, anomaly := range prediction.Anomalies {
		assessment.RiskFactors = append(assessment.RiskFactors, RTRiskFactor{
			Name:        anomaly.Type,
			Contribution: anomaly.Score,
			Description: anomaly.Description,
		})
	}

	if assessment.RiskLevel == "high" {
		assessment.Recommendations = append(assessment.Recommendations,
			"Immediate investigation required",
			"Consider blocking the source",
			"Review recent activity patterns",
		)
	} else if assessment.RiskLevel == "medium" {
		assessment.Recommendations = append(assessment.Recommendations,
			"Monitor closely for further anomalies",
			"Consider additional verification steps",
		)
	} else if assessment.RiskLevel == "low" {
		assessment.Recommendations = append(assessment.Recommendations,
			"Continue monitoring",
		)
	}

	assessment.Details["anomaly_count"] = float64(len(prediction.Anomalies))
	assessment.Details["pattern_count"] = float64(len(prediction.Patterns))
	assessment.Details["confidence"] = prediction.Confidence

	return assessment, nil
}

func (p *RealtimeBehaviorPrediction) PredictBatch(traces []*model.TraceData) ([]*RTPredictionResult, error) {
	results := make([]*RTPredictionResult, len(traces))

	for i, trace := range traces {
		result, err := p.Predict(trace)
		if err != nil {
			return nil, err
		}
		results[i] = result
	}

	return results, nil
}

func (p *RealtimeBehaviorPrediction) GetMetrics() *PredictionMetrics {
	p.mu.RLock()
	defer p.mu.RUnlock()

	avgRisk := 0.0
	if p.totalPredictions > 0 {
		avgRisk = p.totalRiskScore / float64(p.totalPredictions)
	}

	avgLatency := 0.0
	if p.totalPredictions > 0 {
		avgLatency = float64(p.totalLatencyNs) / float64(p.totalPredictions) / 1e6
	}

	return &PredictionMetrics{
		TotalPredictions: p.totalPredictions,
		AvgRiskScore:     avgRisk,
		HighRiskCount:    p.highRiskCount,
		AvgLatencyMs:     avgLatency,
	}
}

func (p *RealtimeBehaviorPrediction) AnalyzeSequence(traces []*model.TraceData) (map[string]interface{}, error) {
	result := map[string]interface{}{
		"sequence_length":   len(traces),
		"trend":             "stable",
		"mean_risk_score":   0.0,
		"risk_increasing":   false,
		"anomaly_count":     0,
	}

	if len(traces) == 0 {
		return result, nil
	}

	totalRisk := 0.0
	totalAnomalies := 0
	for _, trace := range traces {
		prediction, _ := p.Predict(trace)
		totalRisk += prediction.RiskScore
		totalAnomalies += len(prediction.Anomalies)
	}

	meanRisk := totalRisk / float64(len(traces))
	result["mean_risk_score"] = meanRisk
	result["anomaly_count"] = totalAnomalies

	if len(traces) > 1 {
		firstRisk, _ := p.Predict(traces[0])
		lastRisk, _ := p.Predict(traces[len(traces)-1])
		result["risk_increasing"] = lastRisk.RiskScore > firstRisk.RiskScore*1.5
		if lastRisk.RiskScore > firstRisk.RiskScore*1.5 {
			result["trend"] = "increasing"
		} else if lastRisk.RiskScore < firstRisk.RiskScore*0.7 {
			result["trend"] = "decreasing"
		}
	}

	return result, nil
}

func (p *RealtimeBehaviorPrediction) GetRecentPredictions(count int) []*RTPredictionResult {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if count >= len(p.predictionHistory) {
		return append([]*RTPredictionResult{}, p.predictionHistory...)
	}

	return append([]*RTPredictionResult{}, p.predictionHistory[len(p.predictionHistory)-count:]...)
}

func (p *RealtimeBehaviorPrediction) ClearBuffer() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.predictionHistory = []*RTPredictionResult{}
}

func (p *RealtimeBehaviorPrediction) CompareTraces(trace1, trace2 *model.TraceData) (float64, error) {
	pred1, _ := p.Predict(trace1)
	pred2, _ := p.Predict(trace2)

	diff := math.Abs(pred1.RiskScore - pred2.RiskScore)
	similarity := 1.0 - diff

	if similarity < 0 {
		similarity = 0
	}

	return similarity, nil
}

func (p *RealtimeBehaviorPrediction) Reset() {
	p.mu.Lock()
	defer p.mu.Unlock()
	p.predictionHistory = []*RTPredictionResult{}
	p.totalPredictions = 0
	p.totalRiskScore = 0
	p.highRiskCount = 0
	p.totalLatencyNs = 0
}
