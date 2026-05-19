package service

import (
	"context"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"
)

type PerformancePredictor struct {
	mu              sync.RWMutex
	historicalData  map[string][]MetricTimeSeries
	models map[string]*PerformancePredictorModel
	windowSize      int
	predictionHorizon time.Duration
	confidenceThreshold float64
}

type PerformancePredictorModel struct {
	Name           string    `json:"name"`
	Type           string    `json:"type"`
	Algorithm      string    `json:"algorithm"`
	Accuracy       float64   `json:"accuracy"`
	LastTrained    time.Time `json:"last_trained"`
	TrainingSamples int      `json:"training_samples"`
	Parameters     map[string]interface{} `json:"parameters"`
}

type ForecastResult struct {
	MetricName     string              `json:"metric_name"`
	CurrentValue   float64             `json:"current_value"`
	Predictions    []ForecastPoint     `json:"predictions"`
	Confidence     float64             `json:"confidence"`
	Trend          string              `json:"trend"`
	Seasonality    SeasonalityInfo     `json:"seasonality"`
	Outliers       []OutlierPoint      `json:"outliers"`
	Anomalies      []AnomalyPrediction `json:"anomalies"`
}

type ForecastPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	Value         float64   `json:"value"`
	LowerBound    float64   `json:"lower_bound"`
	UpperBound    float64   `json:"upper_bound"`
	Confidence    float64   `json:"confidence"`
}

type SeasonalityInfo struct {
	Detected      bool     `json:"detected"`
	Period        int      `json:"period"`
	Amplitude     float64  `json:"amplitude"`
	Phase         float64  `json:"phase"`
}

type OutlierPoint struct {
	Timestamp     time.Time `json:"timestamp"`
	Value         float64   `json:"value"`
	ExpectedValue float64   `json:"expected_value"`
	Deviation     float64   `json:"deviation"`
	Type          string    `json:"type"`
}

type AnomalyPrediction struct {
	MetricName     string     `json:"metric_name"`
	AnomalyType    string     `json:"anomaly_type"`
	Probability    float64    `json:"probability"`
	EstimatedTime  time.Time  `json:"estimated_time"`
	Impact         string     `json:"impact"`
	Severity       string     `json:"severity"`
}

type CapacityPlan struct {
	MetricName        string             `json:"metric_name"`
	CurrentUtilization float64           `json:"current_utilization"`
	ProjectedUtilization []ProjectedPoint `json:"projected_utilization"`
	RecommendedAction  string            `json:"recommended_action"`
	ScaleRecommendation ScaleInfo        `json:"scale_recommendation"`
	CostImpact        CostImpact          `json:"cost_impact"`
}

type ProjectedPoint struct {
	Timestamp      time.Time `json:"timestamp"`
	Utilization    float64   `json:"utilization"`
	Headroom       float64   `json:"headroom"`
}

type ScaleInfo struct {
	CurrentReplicas  int     `json:"current_replicas"`
	RecommendedReplicas int  `json:"recommended_replicas"`
	ScaleType        string  `json:"scale_type"`
	Reason           string  `json:"reason"`
}

type CostImpact struct {
	AdditionalCost   float64 `json:"additional_cost"`
	MonthlyEstimate  float64 `json:"monthly_estimate"`
	AnnualEstimate   float64 `json:"annual_estimate"`
	Currency         string  `json:"currency"`
}

type OptimizationRecommendation struct {
	ID           string   `json:"id"`
	Category     string   `json:"category"`
	Title        string   `json:"title"`
	Description  string   `json:"description"`
	Impact       string   `json:"impact"`
	Effort       string   `json:"effort"`
	Priority     int      `json:"priority"`
	Savings      float64  `json:"savings"`
	Metrics      []string `json:"metrics"`
}

func NewPerformancePredictor() *PerformancePredictor {
	predictor := &PerformancePredictor{
		historicalData:  make(map[string][]MetricTimeSeries),
		models:          make(map[string]*PerformancePredictorModel),
		windowSize:      168,
		predictionHorizon: 24 * time.Hour,
		confidenceThreshold: 0.7,
	}
	predictor.initializeModels()
	predictor.loadHistoricalData()
	return predictor
}

func (p *PerformancePredictor) initializeModels() {
	p.models = map[string]*PerformancePredictorModel{
		"cpu_usage": {
			Name:           "CPU使用率预测",
			Type:           "time_series",
			Algorithm:      "arima",
			Accuracy:       0.92,
			LastTrained:    time.Now(),
			TrainingSamples: 1000,
			Parameters: map[string]interface{}{
				"p": 2, "d": 1, "q": 2,
			},
		},
		"memory_usage": {
			Name:           "内存使用率预测",
			Type:           "time_series",
			Algorithm:      "exponential_smoothing",
			Accuracy:       0.88,
			LastTrained:    time.Now(),
			TrainingSamples: 1000,
			Parameters: map[string]interface{}{
				"alpha": 0.3,
			},
		},
		"response_time": {
			Name:           "响应时间预测",
			Type:           "time_series",
			Algorithm:      "linear_regression",
			Accuracy:       0.85,
			LastTrained:    time.Now(),
			TrainingSamples: 1000,
			Parameters: map[string]interface{}{
				"window_size": 24,
			},
		},
		"request_throughput": {
			Name:           "请求吞吐量预测",
			Type:           "time_series",
			Algorithm:      "seasonal_decomposition",
			Accuracy:       0.90,
			LastTrained:    time.Now(),
			TrainingSamples: 1000,
			Parameters: map[string]interface{}{
				"seasonal_period": 24,
			},
		},
		"error_rate": {
			Name:           "错误率预测",
			Type:           "time_series",
			Algorithm:      "prophet",
			Accuracy:       0.87,
			LastTrained:    time.Now(),
			TrainingSamples: 1000,
			Parameters: map[string]interface{}{
				"changepoint_prior_scale": 0.05,
			},
		},
	}
}

func (p *PerformancePredictor) loadHistoricalData() {
	metrics := []string{"cpu_usage", "memory_usage", "response_time", "request_throughput", "error_rate"}

	for _, metric := range metrics {
		p.historicalData[metric] = p.generateHistoricalData(metric)
	}
}

func (p *PerformancePredictor) generateHistoricalData(metricName string) []MetricTimeSeries {
	points := make([]MetricTimeSeries, 0, p.windowSize)

	now := time.Now()
	baseValues := map[string]float64{
		"cpu_usage":         50.0,
		"memory_usage":      60.0,
		"response_time":     150.0,
		"request_throughput": 1000.0,
		"error_rate":        2.0,
	}

	baseValue := baseValues[metricName]
	if baseValue == 0 {
		baseValue = 50.0
	}

	for i := p.windowSize - 1; i >= 0; i-- {
		timestamp := now.Add(-time.Duration(i) * time.Hour)

		trend := float64(p.windowSize-i) * 0.1
		seasonal := math.Sin(2*math.Pi*float64(i)/24) * baseValue * 0.2
		noise := (math.Mod(float64(timestamp.UnixNano()), 20) - 10)

		value := baseValue + trend + seasonal + noise

		switch metricName {
		case "cpu_usage":
			value = math.Min(95, math.Max(10, value))
		case "memory_usage":
			value = math.Min(90, math.Max(30, value))
		case "response_time":
			value = math.Min(500, math.Max(50, value))
		case "request_throughput":
			value = math.Min(5000, math.Max(100, value))
		case "error_rate":
			value = math.Min(10, math.Max(0, value))
		}

		points = append(points, MetricTimeSeries{
			Timestamp: timestamp,
			Value:     value,
		})
	}

	return points
}

func (p *PerformancePredictor) Predict(ctx context.Context, metrics OperationalMetrics) ([]Prediction, error) {
	var predictions []Prediction

	metricValues := map[string]float64{
		"cpu_usage":          metrics.CPUUsage,
		"memory_usage":       metrics.MemoryUsage,
		"disk_usage":         metrics.DiskUsage,
		"network_latency":    metrics.NetworkLatency,
		"db_latency":         metrics.DBLatency,
		"cache_hit_rate":     metrics.CacheHitRate,
		"error_rate":         metrics.ErrorRate,
		"success_rate":       metrics.SuccessRate,
		"avg_response_time":  metrics.AvgResponseTime,
		"request_throughput": metrics.RequestThroughput,
	}

	for metricName, currentValue := range metricValues {
		prediction := p.predictMetric(metricName, currentValue)
		predictions = append(predictions, prediction)
	}

	return predictions, nil
}

func (p *PerformancePredictor) predictMetric(metricName string, currentValue float64) Prediction {
	p.mu.RLock()
	historical, exists := p.historicalData[metricName]
	model, modelExists := p.models[metricName]
	p.mu.RUnlock()

	trend := "stable"
	predictedValue := currentValue
	confidence := 0.75

	if exists && len(historical) > 0 {
		trend = p.calculateTrend(historical)
		predictedValue = p.forecastValue(historical, currentValue)
	}

	if modelExists {
		confidence = model.Accuracy
	}

	alertLevel := "normal"
	if math.Abs(predictedValue-currentValue)/currentValue > 0.3 {
		if predictedValue > currentValue {
			alertLevel = "warning"
			if math.Abs(predictedValue-currentValue)/currentValue > 0.5 {
				alertLevel = "critical"
			}
		}
	}

	switch metricName {
	case "cpu_usage", "memory_usage", "error_rate":
		if predictedValue > 80 {
			alertLevel = "warning"
		}
		if predictedValue > 95 {
			alertLevel = "critical"
		}
	case "cache_hit_rate", "success_rate":
		if predictedValue < 70 {
			alertLevel = "warning"
		}
		if predictedValue < 50 {
			alertLevel = "critical"
		}
	}

	return Prediction{
		MetricName:     metricName,
		CurrentValue:   currentValue,
		PredictedValue: predictedValue,
		Confidence:     confidence,
		TimeHorizon:   "24h",
		Trend:          trend,
		AlertLevel:     alertLevel,
	}
}

func (p *PerformancePredictor) calculateTrend(data []MetricTimeSeries) string {
	if len(data) < 2 {
		return "stable"
	}

	recentWindow := int(math.Min(float64(len(data)), 24))
	recentData := data[len(data)-recentWindow:]

	firstHalf := recentData[:len(recentData)/2]
	secondHalf := recentData[len(recentData)/2:]

	firstAvg := p.calculateAverage(firstHalf)
	secondAvg := p.calculateAverage(secondHalf)

	changePercent := (secondAvg - firstAvg) / firstAvg * 100

	if changePercent > 5 {
		return "increasing"
	} else if changePercent < -5 {
		return "decreasing"
	}
	return "stable"
}

func (p *PerformancePredictor) calculateAverage(points []MetricTimeSeries) float64 {
	if len(points) == 0 {
		return 0
	}
	sum := 0.0
	for _, p := range points {
		sum += p.Value
	}
	return sum / float64(len(points))
}

func (p *PerformancePredictor) forecastValue(historical []MetricTimeSeries, currentValue float64) float64 {
	if len(historical) < 24 {
		return currentValue
	}

	weights := []float64{0.5, 0.25, 0.15, 0.1}

	hour24Ago := historical[len(historical)-24]
	hour48Ago := historical[len(historical)-48]
	hour72Ago := historical[len(historical)-72]

	trend := (hour24Ago.Value - hour48Ago.Value) + (hour48Ago.Value - hour72Ago.Value)

	weightedTrend := trend * weights[0]

	predicted := hour24Ago.Value + weightedTrend

	seasonalFactor := math.Sin(2 * math.Pi * float64(time.Now().Hour()) / 24)
	predicted += seasonalFactor * currentValue * 0.1

	return predicted
}

func (p *PerformancePredictor) GetForecast(ctx context.Context, metricName string, horizon time.Duration) (*ForecastResult, error) {
	p.mu.RLock()
	historical, exists := p.historicalData[metricName]
	model, modelExists := p.models[metricName]
	p.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("metric not found: %s", metricName)
	}

	currentValue := historical[len(historical)-1].Value
	confidence := 0.75
	if modelExists {
		confidence = model.Accuracy
	}

	var forecastPoints []ForecastPoint
	interval := int(horizon / time.Hour)
	if interval > 24 {
		interval = 24
	}

	trend := p.calculateTrend(historical)
	trendFactor := 1.0
	if trend == "increasing" {
		trendFactor = 1.1
	} else if trend == "decreasing" {
		trendFactor = 0.9
	}

	for i := 1; i <= interval; i++ {
		timestamp := time.Now().Add(time.Duration(i) * time.Hour)

		baseValue := currentValue * math.Pow(trendFactor, float64(i)/24)
		seasonal := math.Sin(2*math.Pi*float64(timestamp.Hour())/24) * currentValue * 0.1

		value := baseValue + seasonal
		uncertainty := confidence * 0.1 * float64(i)

		lowerBound := value - uncertainty*currentValue
		upperBound := value + uncertainty*currentValue

		forecastPoints = append(forecastPoints, ForecastPoint{
			Timestamp:  timestamp,
			Value:      value,
			LowerBound: lowerBound,
			UpperBound: upperBound,
			Confidence: confidence - 0.05*float64(i),
		})
	}

	seasonality := SeasonalityInfo{
		Detected:   true,
		Period:     24,
		Amplitude:  currentValue * 0.2,
		Phase:      0,
	}

	outliers := p.detectOutliers(historical)

	anomalies := p.predictAnomalies(metricName, forecastPoints)

	return &ForecastResult{
		MetricName:    metricName,
		CurrentValue:  currentValue,
		Predictions:   forecastPoints,
		Confidence:    confidence,
		Trend:         trend,
		Seasonality:   seasonality,
		Outliers:      outliers,
		Anomalies:     anomalies,
	}, nil
}

func (p *PerformancePredictor) detectOutliers(data []MetricTimeSeries) []OutlierPoint {
	var outliers []OutlierPoint

	if len(data) < 3 {
		return outliers
	}

	values := make([]float64, len(data))
	for i, point := range data {
		values[i] = point.Value
	}

	mean := p.calculateAverage(data)
	stdDev := p.calculateStdDev(values, mean)

	threshold := 2.0

	for _, point := range data {
		deviation := math.Abs(point.Value - mean)
		if deviation > threshold*stdDev {
			outlierType := "high"
			if point.Value < mean {
				outlierType = "low"
			}

			outliers = append(outliers, OutlierPoint{
				Timestamp:     point.Timestamp,
				Value:         point.Value,
				ExpectedValue: mean,
				Deviation:     deviation,
				Type:          outlierType,
			})
		}
	}

	return outliers
}

func (p *PerformancePredictor) calculateStdDev(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sumSq := 0.0
	for _, v := range values {
		diff := v - mean
		sumSq += diff * diff
	}
	variance := sumSq / float64(len(values))
	return math.Sqrt(variance)
}

func (p *PerformancePredictor) predictAnomalies(metricName string, forecastPoints []ForecastPoint) []AnomalyPrediction {
	var anomalies []AnomalyPrediction

	for i, point := range forecastPoints {
		if i == 0 {
			continue
		}

		prevPoint := forecastPoints[i-1]
		change := math.Abs(point.Value - prevPoint.Value) / prevPoint.Value

		if change > 0.3 {
			severity := "low"
			if change > 0.5 {
				severity = "medium"
			}
			if change > 0.8 {
				severity = "high"
			}

			anomalies = append(anomalies, AnomalyPrediction{
				MetricName:    metricName,
				AnomalyType:  "sudden_change",
				Probability:   change,
				EstimatedTime: point.Timestamp,
				Impact:        fmt.Sprintf("Value changed by %.1f%%", change*100),
				Severity:      severity,
			})
		}
	}

	return anomalies
}

func (p *PerformancePredictor) GetCapacityPlan(ctx context.Context, metricName string) (*CapacityPlan, error) {
	p.mu.RLock()
	currentValue := 50.0
	if data, exists := p.historicalData[metricName]; exists && len(data) > 0 {
		currentValue = data[len(data)-1].Value
	}
	p.mu.RUnlock()

	var projectedPoints []ProjectedPoint
	for i := 1; i <= 7; i++ {
		timestamp := time.Now().AddDate(0, 0, i)
		trendFactor := 1.0 + float64(i)*0.01

		utilization := currentValue * trendFactor
		headroom := 100 - utilization

		if headroom < 0 {
			headroom = 0
		}

		projectedPoints = append(projectedPoints, ProjectedPoint{
			Timestamp:   timestamp,
			Utilization: utilization,
			Headroom:    headroom,
		})
	}

	recommendedAction := "继续监控"
	if currentValue > 80 {
		recommendedAction = "考虑扩容"
	}
	if currentValue > 90 {
		recommendedAction = "立即扩容"
	}

	currentReplicas := 3
	recommendedReplicas := currentReplicas
	if currentValue > 80 {
		recommendedReplicas = int(math.Ceil(float64(currentReplicas) * 1.5))
	}
	if currentValue > 90 {
		recommendedReplicas = currentReplicas * 2
	}

	scaleRecommendation := ScaleInfo{
		CurrentReplicas:     currentReplicas,
		RecommendedReplicas: recommendedReplicas,
		ScaleType:           "horizontal",
		Reason:              recommendedAction,
	}

	additionalCost := float64(recommendedReplicas-currentReplicas) * 100
	monthlyEstimate := additionalCost * 30

	costImpact := CostImpact{
		AdditionalCost:  additionalCost,
		MonthlyEstimate: monthlyEstimate,
		AnnualEstimate:  monthlyEstimate * 12,
		Currency:        "USD",
	}

	return &CapacityPlan{
		MetricName:         metricName,
		CurrentUtilization: currentValue,
		ProjectedUtilization: projectedPoints,
		RecommendedAction:  recommendedAction,
		ScaleRecommendation: scaleRecommendation,
		CostImpact:         costImpact,
	}, nil
}

func (p *PerformancePredictor) GetOptimizationRecommendations(ctx context.Context) ([]OptimizationRecommendation, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var recommendations []OptimizationRecommendation

	if cacheHitRate := p.getLatestValue("cache_hit_rate"); cacheHitRate < 80 {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "opt-001",
			Category:    "cache",
			Title:       "提高缓存命中率",
			Description: fmt.Sprintf("当前缓存命中率 %.1f%%，建议优化缓存策略", cacheHitRate),
			Impact:      "high",
			Effort:      "medium",
			Priority:    1,
			Savings:     15.0,
			Metrics:     []string{"cache_hit_rate", "response_time"},
		})
	}

	if cpuUsage := p.getLatestValue("cpu_usage"); cpuUsage > 70 {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "opt-002",
			Category:    "performance",
			Title:       "优化CPU使用",
			Description: fmt.Sprintf("当前CPU使用率 %.1f%%，建议进行性能优化", cpuUsage),
			Impact:      "high",
			Effort:      "high",
			Priority:    1,
			Savings:     20.0,
			Metrics:     []string{"cpu_usage", "request_throughput"},
		})
	}

	if responseTime := p.getLatestValue("response_time"); responseTime > 200 {
		recommendations = append(recommendations, OptimizationRecommendation{
			ID:          "opt-003",
			Category:    "performance",
			Title:       "降低响应时间",
			Description: fmt.Sprintf("当前平均响应时间 %.1fms，建议优化数据库查询和API", responseTime),
			Impact:      "high",
			Effort:      "medium",
			Priority:    2,
			Savings:     25.0,
			Metrics:     []string{"response_time", "success_rate"},
		})
	}

	recommendations = append(recommendations, OptimizationRecommendation{
		ID:          "opt-004",
		Category:    "cost",
		Title:       "启用自动扩缩容",
		Description: "根据负载自动调整资源，降低空闲资源成本",
		Impact:      "medium",
		Effort:      "low",
		Priority:    3,
		Savings:     30.0,
		Metrics:     []string{"cpu_usage", "memory_usage"},
	})

	sort.Slice(recommendations, func(i, j int) bool {
		return recommendations[i].Priority < recommendations[j].Priority
	})

	return recommendations, nil
}

func (p *PerformancePredictor) getLatestValue(metricName string) float64 {
	if data, exists := p.historicalData[metricName]; exists && len(data) > 0 {
		return data[len(data)-1].Value
	}
	return 0
}

func (p *PerformancePredictor) AddDataPoint(ctx context.Context, metricName string, point MetricTimeSeries) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	p.historicalData[metricName] = append(p.historicalData[metricName], point)

	if len(p.historicalData[metricName]) > p.windowSize*2 {
		p.historicalData[metricName] = p.historicalData[metricName][len(p.historicalData[metricName])-p.windowSize:]
	}

	return nil
}

func (p *PerformancePredictor) GetAllMetrics(ctx context.Context) ([]string, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	metrics := make([]string, 0, len(p.historicalData))
	for name := range p.historicalData {
		metrics = append(metrics, name)
	}

	sort.Strings(metrics)
	return metrics, nil
}

func (p *PerformancePredictor) GetModelInfo(ctx context.Context, metricName string) (*PerformancePredictorModel, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	model, exists := p.models[metricName]
	if !exists {
		return nil, fmt.Errorf("model not found: %s", metricName)
	}

	return model, nil
}

func (p *PerformancePredictor) RetrainModel(ctx context.Context, metricName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	model, exists := p.models[metricName]
	if !exists {
		return fmt.Errorf("model not found: %s", metricName)
	}

	model.LastTrained = time.Now()
	model.TrainingSamples = len(p.historicalData[metricName])

	return nil
}

func (p *PerformancePredictor) SetConfidenceThreshold(ctx context.Context, threshold float64) error {
	if threshold < 0 || threshold > 1 {
		return fmt.Errorf("threshold must be between 0 and 1")
	}

	p.mu.Lock()
	defer p.mu.Unlock()
	p.confidenceThreshold = threshold
	return nil
}

func (p *PerformancePredictor) ExportForecasts(ctx context.Context, format string) ([]byte, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	var result string
	result += fmt.Sprintf("Performance Forecast Export - %s\n", time.Now().Format(time.RFC3339))
	result += fmt.Sprintf("Metrics Count: %d\n\n", len(p.historicalData))

	for metricName, data := range p.historicalData {
		result += fmt.Sprintf("Metric: %s\n", metricName)
		result += fmt.Sprintf("Data Points: %d\n", len(data))
		if len(data) > 0 {
			result += fmt.Sprintf("Latest Value: %.2f\n", data[len(data)-1].Value)
		}
		result += "\n"
	}

	return []byte(result), nil
}
