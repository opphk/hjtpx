package service

import (
	"context"
	"fmt"
	"log"
	"math"
	"sort"
	"sync"
	"sync/atomic"
	"time"
)

type ResourcePredictorV2 struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	models        map[string]*PredictionModel
	dataStore     *TimeSeriesStore
	stats         *PredictionStats
}

type PredictionModel struct {
	Name      string
	Type      ModelType
	Training  bool
	data      []TimeSeriesPoint
	params    ModelParams
	predictions []PredictionResult
}

type ModelType int

const (
	ModelLinear ModelType = iota
	ModelExponential
	ModelPolynomial
	ModelARIMA
	ModelLSTM
)

type ModelParams struct {
	LearningRate float64
	WindowSize   int
	ForecastHorizon int
	Seasonality  int
	TrendDegree int
}

type TimeSeriesPoint struct {
	Timestamp time.Time
	Value     float64
	Metadata  map[string]interface{}
}

type TimeSeriesStore struct {
	mu      sync.RWMutex
	series  map[string][]TimeSeriesPoint
	maxSize int
}

type PredictionResult struct {
	Timestamp  time.Time
	Value      float64
	Confidence float64
	UpperBound float64
	LowerBound float64
}

type PredictionStats struct {
	TotalPredictions   atomic.Int64
	AccuratePredictions atomic.Int64
	AvgError          atomic.Float64
	LastPredictionTime atomic.Value
	ModelAccuracy     map[string]float64
}

type ResourceUsagePredictor struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	cpuModel      *PredictionModel
	memoryModel   *PredictionModel
	networkModel  *PredictionModel
	diskModel     *PredictionModel
	predictor     *ResourcePredictorV2
}

type ResourceForecast struct {
	Timestamp     time.Time
	CPUUsage     float64
	MemoryUsage  float64
	NetworkUsage float64
	DiskUsage    float64
	Confidence   float64
}

type CostOptimizer struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	config        *CostConfig
	providers     map[string]*CloudProvider
	history       *CostHistory
	recommendations *CostRecommendationEngine
	stats         *CostStats
}

type CostConfig struct {
	BudgetLimit    float64
	BudgetPeriod   time.Duration
	AlertThreshold float64
	OptimizeFor   CostObjective
}

type CostObjective int

const (
	CostMinimize CostObjective = iota
	PerformanceMaximize
	Balanced
)

type CloudProvider struct {
	Name     string
	Region   string
	Pricing  *PricingModel
	Features map[string]bool
}

type PricingModel struct {
	ComputeUnitCost    float64
	MemoryUnitCost    float64
	StorageUnitCost   float64
	NetworkUnitCost   float64
	Currency          string
	Discounts        []PricingDiscount
}

type PricingDiscount struct {
	CommitLevel int
	PercentOff  float64
	Duration    time.Duration
}

type CostHistory struct {
	mu       sync.RWMutex
	records  []CostRecord
	maxSize  int
}

type CostRecord struct {
	Timestamp   time.Time
	Cost        float64
	ResourceType string
	Provider    string
	Quantity    float64
}

type CostRecommendationEngine struct {
	mu               sync.RWMutex
	recommendations  []CostRecommendation
	strategies      []OptimizationStrategy
}

type CostRecommendation struct {
	ID          string
	Type        RecommendationType
	Priority    int
	Title       string
	Description string
	Savings     float64
	Effort      string
	ROI         float64
}

type RecommendationType int

const (
	RecommendRightSizing RecommendationType = iota
	RecommendReservedInstance
	RecommendSpotInstance
	RecommendStorageTier
	RecommendRegion
	RecommendScaling
)

type OptimizationStrategy struct {
	Name       string
	Condition  func(*CostOptimizer) bool
	Apply      func(*CostOptimizer) error
	Savings    float64
}

type CostStats struct {
	TotalCost      atomic.Float64
	ProjectedCost  atomic.Float64
	ActualCost     atomic.Float64
	TotalSavings   atomic.Float64
	Recommendations atomic.Int64
	LastUpdate     atomic.Value
}

func NewResourcePredictorV2(ctx context.Context) *ResourcePredictorV2 {
	if ctx == nil {
		ctx = context.Background()
	}
	childCtx, cancel := context.WithCancel(ctx)

	return &ResourcePredictorV2{
		ctx:       childCtx,
		cancel:    cancel,
		models:    make(map[string]*PredictionModel),
		dataStore: NewTimeSeriesStore(10000),
		stats:    &PredictionStats{
			ModelAccuracy: make(map[string]float64),
		},
	}
}

func NewTimeSeriesStore(maxSize int) *TimeSeriesStore {
	return &TimeSeriesStore{
		series:  make(map[string][]TimeSeriesPoint),
		maxSize: maxSize,
	}
}

func (r *ResourcePredictorV2) Start() error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if r.isRunning {
		return nil
	}

	r.isRunning = true

	go r.collectAndPredict()
	go r.evaluateAccuracy()

	log.Println("[ResourcePredictorV2] Started successfully")
	return nil
}

func (r *ResourcePredictorV2) Stop() {
	r.mu.Lock()
	defer r.mu.Unlock()

	if !r.isRunning {
		return
	}

	r.cancel()
	r.isRunning = false
	log.Println("[ResourcePredictorV2] Stopped")
}

func (r *ResourcePredictorV2) AddModel(name string, modelType ModelType, params ModelParams) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.models[name]; exists {
		return fmt.Errorf("model %s already exists", name)
	}

	model := &PredictionModel{
		Name:       name,
		Type:       modelType,
		params:     params,
		data:       make([]TimeSeriesPoint, 0),
		predictions: make([]PredictionResult, 0),
	}

	r.models[name] = model
	return nil
}

func (r *ResourcePredictorV2) AddDataPoint(name string, point TimeSeriesPoint) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	model, exists := r.models[name]
	if !exists {
		return fmt.Errorf("model %s not found", name)
	}

	model.data = append(model.data, point)

	if len(model.data) > r.dataStore.maxSize {
		model.data = model.data[len(model.data)-r.dataStore.maxSize:]
	}

	r.dataStore.mu.Lock()
	r.dataStore.series[name] = append(r.dataStore.series[name], point)
	if len(r.dataStore.series[name]) > r.dataStore.maxSize {
		r.dataStore.series[name] = r.dataStore.series[name][len(r.dataStore.series[name])-r.dataStore.maxSize:]
	}
	r.dataStore.mu.Unlock()

	return nil
}

func (r *ResourcePredictorV2) Predict(name string, horizon time.Duration) ([]PredictionResult, error) {
	r.mu.RLock()
	model, exists := r.models[name]
	r.mu.RUnlock()

	if !exists {
		return nil, fmt.Errorf("model %s not found", name)
	}

	if len(model.data) < model.params.WindowSize {
		return nil, fmt.Errorf("insufficient data for prediction")
	}

	r.stats.TotalPredictions.Add(1)

	var predictions []PredictionResult

	switch model.Type {
	case ModelLinear:
		predictions = r.linearPrediction(model, horizon)
	case ModelExponential:
		predictions = r.exponentialPrediction(model, horizon)
	case ModelPolynomial:
		predictions = r.polynomialPrediction(model, horizon)
	case ModelARIMA:
		predictions = r.arimaPrediction(model, horizon)
	default:
		predictions = r.linearPrediction(model, horizon)
	}

	model.predictions = predictions
	r.stats.LastPredictionTime.Store(time.Now())

	return predictions, nil
}

func (r *ResourcePredictorV2) linearPrediction(model *PredictionModel, horizon time.Duration) []PredictionResult {
	data := model.data
	n := len(data)

	if n < 2 {
		return nil
	}

	xMean := 0.0
	yMean := 0.0
	for i, p := range data {
		xMean += float64(i)
		yMean += p.Value
	}
	xMean /= float64(n)
	yMean /= float64(n)

	var slope, intercept float64
	var ssxx, ssyy, ssxy float64

	for i, p := range data {
		x := float64(i)
		y := p.Value
		dx := x - xMean
		dy := y - yMean
		ssxx += dx * dx
		ssyy += dy * dy
		ssxy += dx * dy
	}

	if ssxx != 0 {
		slope = ssxy / ssxx
		intercept = yMean - slope*xMean
	}

	var residuals []float64
	for i, p := range data {
		predicted := slope*float64(i) + intercept
		residuals = append(residuals, p.Value-predicted)
	}
	residualStd := stdDev(residuals)

	var predictions []PredictionResult
	step := time.Minute * 5
	for t := time.Now().Add(step); t.Before(time.Now().Add(horizon)); t = t.Add(step) {
		x := float64(n) + float64(t.Sub(data[n-1].Timestamp))/float64(step)
		predicted := slope*x + intercept

		z := 1.96
		confidence := math.Min(0.99, math.Max(0.5, 1-residualStd/math.Abs(predicted)))

		predictions = append(predictions, PredictionResult{
			Timestamp:  t,
			Value:      predicted,
			Confidence: confidence,
			UpperBound: predicted + z*residualStd,
			LowerBound: predicted - z*residualStd,
		})
	}

	return predictions
}

func (r *ResourcePredictorV2) exponentialPrediction(model *PredictionModel, horizon time.Duration) []PredictionResult {
	data := model.data
	n := len(data)

	if n < 3 {
		return r.linearPrediction(model, horizon)
	}

	var predictions []PredictionResult
	step := time.Minute * 5

	baseValue := data[0].Value
	growthRate := 0.0

	for i := 1; i < n; i++ {
		if data[i-1].Value > 0 {
			growthRate += math.Log(data[i].Value / data[i-1].Value)
		}
	}
	growthRate /= float64(n - 1)

	for t := time.Now().Add(step); t.Before(time.Now().Add(horizon)); t = t.Add(step) {
		elapsed := float64(t.Sub(data[0].Timestamp)) / float64(step)
		predicted := baseValue * math.Exp(growthRate*elapsed)
		predictions = append(predictions, PredictionResult{
			Timestamp:  t,
			Value:      predicted,
			Confidence: 0.8,
		})
	}

	return predictions
}

func (r *ResourcePredictorV2) polynomialPrediction(model *PredictionModel, horizon time.Duration) []PredictionResult {
	return r.linearPrediction(model, horizon)
}

func (r *ResourcePredictorV2) arimaPrediction(model *PredictionModel, horizon time.Duration) []PredictionResult {
	return r.linearPrediction(model, horizon)
}

func (r *ResourcePredictorV2) collectAndPredict() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.runPredictions()
		}
	}
}

func (r *ResourcePredictorV2) runPredictions() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, model := range r.models {
		if len(model.data) >= model.params.WindowSize {
			horizon := time.Duration(model.params.ForecastHorizon) * time.Minute
			predictions, err := r.Predict(name, horizon)
			if err == nil {
				log.Printf("[ResourcePredictorV2] Predictions for %s: %d points", name, len(predictions))
			}
		}
	}
}

func (r *ResourcePredictorV2) evaluateAccuracy() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-r.ctx.Done():
			return
		case <-ticker.C:
			r.evaluateModelAccuracy()
		}
	}
}

func (r *ResourcePredictorV2) evaluateModelAccuracy() {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for name, model := range r.models {
		if len(model.predictions) == 0 || len(model.data) < model.params.WindowSize+1 {
			continue
		}

		actualIndex := len(model.data) - 1
		actual := model.data[actualIndex].Value

		var totalError float64
		var count int

		for _, pred := range model.predictions {
			error := math.Abs(pred.Value - actual)
			totalError += error
			count++
		}

		if count > 0 {
			avgError := totalError / float64(count)
			accuracy := 1.0 - math.Min(1.0, avgError/actual)
			r.stats.ModelAccuracy[name] = accuracy

			if accuracy > 0.9 {
				r.stats.AccuratePredictions.Add(1)
			}

			r.stats.AvgError.Store(avgError)
		}
	}
}

func (r *ResourcePredictorV2) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_predictions":     r.stats.TotalPredictions.Load(),
		"accurate_predictions":  r.stats.AccuratePredictions.Load(),
		"avg_error":            r.stats.AvgError.Load(),
		"last_prediction_time": r.stats.LastPredictionTime.Load(),
		"model_count":          len(r.models),
		"model_accuracy":       r.stats.ModelAccuracy,
	}
}

func stdDev(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}

	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	var variance float64
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(values))

	return math.Sqrt(variance)
}

func NewCostOptimizer(config *CostConfig) *CostOptimizer {
	if config == nil {
		config = DefaultCostConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	optimizer := &CostOptimizer{
		ctx:            ctx,
		cancel:         cancel,
		config:         config,
		providers:      make(map[string]*CloudProvider),
		history:        NewCostHistory(10000),
		recommendations: NewCostRecommendationEngine(),
		stats:          &CostStats{},
	}

	optimizer.initializeProviders()
	optimizer.initializeStrategies()

	return optimizer
}

func DefaultCostConfig() *CostConfig {
	return &CostConfig{
		BudgetLimit:    10000,
		BudgetPeriod:   30 * 24 * time.Hour,
		AlertThreshold: 0.8,
		OptimizeFor:    Balanced,
	}
}

func (c *CostOptimizer) Start() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.isRunning {
		return nil
	}

	c.isRunning = true

	go c.monitor()
	go c.generateRecommendations()

	log.Println("[CostOptimizer] Started successfully")
	return nil
}

func (c *CostOptimizer) Stop() {
	c.mu.Lock()
	defer c.mu.Unlock()

	if !c.isRunning {
		return
	}

	c.cancel()
	c.isRunning = false
	log.Println("[CostOptimizer] Stopped")
}

func (c *CostOptimizer) initializeProviders() {
	c.providers["aws"] = &CloudProvider{
		Name:  "AWS",
		Region: "us-east-1",
		Pricing: &PricingModel{
			ComputeUnitCost:  0.0416,
			MemoryUnitCost:   0.0052,
			StorageUnitCost:  0.0001,
			NetworkUnitCost:  0.01,
			Currency:        "USD",
		},
		Features: map[string]bool{
			"spot_instances": true,
			"reserved_instances": true,
		},
	}

	c.providers["gcp"] = &CloudProvider{
		Name:  "GCP",
		Region: "us-central1",
		Pricing: &PricingModel{
			ComputeUnitCost:  0.0335,
			MemoryUnitCost:   0.0044,
			StorageUnitCost:  0.0001,
			NetworkUnitCost:  0.01,
			Currency:         "USD",
		},
		Features: map[string]bool{
			"spot_instances": true,
			"reserved_instances": true,
		},
	}
}

func (c *CostOptimizer) initializeStrategies() {
	c.recommendations.strategies = append(c.recommendations.strategies,
		OptimizationStrategy{
			Name: "auto-scaling",
			Condition: func(o *CostOptimizer) bool {
				return o.stats.TotalCost.Load() > o.config.BudgetLimit*0.8
			},
			Apply: func(o *CostOptimizer) error {
				log.Println("[CostOptimizer] Applying auto-scaling strategy")
				return nil
			},
			Savings: 0.25,
		},
		OptimizationStrategy{
			Name: "spot-instance",
			Condition: func(o *CostOptimizer) bool {
				return o.config.OptimizeFor == CostMinimize
			},
			Apply: func(o *CostOptimizer) error {
				log.Println("[CostOptimizer] Applying spot instance strategy")
				return nil
			},
			Savings: 0.70,
		},
		OptimizationStrategy{
			Name: "reserved-capacity",
			Condition: func(o *CostOptimizer) bool {
				return o.stats.TotalCost.Load() > o.config.BudgetLimit*0.5
			},
			Apply: func(o *CostOptimizer) error {
				log.Println("[CostOptimizer] Applying reserved capacity strategy")
				return nil
			},
			Savings: 0.40,
		},
	)
}

func (c *CostOptimizer) RecordCost(record CostRecord) {
	c.history.mu.Lock()
	c.history.records = append(c.history.records, record)
	if len(c.history.records) > c.history.maxSize {
		c.history.records = c.history.records[len(c.history.records)-c.history.maxSize:]
	}
	c.history.mu.Unlock()

	c.stats.TotalCost.Add(record.Cost)
}

func (c *CostOptimizer) CalculateProjectedCost() float64 {
	c.history.mu.RLock()
	defer c.history.mu.RUnlock()

	if len(c.history.records) == 0 {
		return 0
	}

	var totalCost float64
	var maxTimestamp time.Time
	var minTimestamp time.Time

	for _, record := range c.history.records {
		totalCost += record.Cost
		if maxTimestamp.IsZero() || record.Timestamp.After(maxTimestamp) {
			maxTimestamp = record.Timestamp
		}
		if minTimestamp.IsZero() || record.Timestamp.Before(minTimestamp) {
			minTimestamp = record.Timestamp
		}
	}

	elapsed := maxTimestamp.Sub(minTimestamp)
	if elapsed == 0 {
		return totalCost
	}

	projected := totalCost * float64(c.config.BudgetPeriod) / float64(elapsed)
	c.stats.ProjectedCost.Store(projected)

	return projected
}

func (c *CostOptimizer) monitor() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.checkBudget()
		}
	}
}

func (c *CostOptimizer) checkBudget() {
	projected := c.CalculateProjectedCost()

	if projected > c.config.BudgetLimit*c.config.AlertThreshold {
		log.Printf("[CostOptimizer] Budget alert: projected cost $%.2f exceeds threshold $%.2f",
			projected, c.config.BudgetLimit*c.config.AlertThreshold)
	}
}

func (c *CostOptimizer) generateRecommendations() {
	ticker := time.NewTicker(24 * time.Hour)
	defer ticker.Stop()

	for {
		select {
		case <-c.ctx.Done():
			return
		case <-ticker.C:
			c.updateRecommendations()
		}
	}
}

func (c *CostOptimizer) updateRecommendations() {
	c.recommendations.mu.Lock()
	defer c.recommendations.mu.Unlock()

	c.recommendations.recommendations = nil

	if c.stats.TotalCost.Load() > c.config.BudgetLimit*0.8 {
		c.recommendations.recommendations = append(c.recommendations.recommendations,
			CostRecommendation{
				ID:          "rec-001",
				Type:        RecommendRightSizing,
				Priority:    1,
				Title:       "Right-size your instances",
				Description: "Analyze current instance usage and downsize underutilized resources",
				Savings:     0.20,
				Effort:      "Medium",
				ROI:         3.0,
			},
		)
	}

	if c.config.OptimizeFor == CostMinimize {
		c.recommendations.recommendations = append(c.recommendations.recommendations,
			CostRecommendation{
				ID:          "rec-002",
				Type:        RecommendSpotInstance,
				Priority:    1,
				Title:       "Use Spot Instances",
				Description: "Migrate suitable workloads to Spot/Preemptible instances",
				Savings:     0.70,
				Effort:      "High",
				ROI:         5.0,
			},
		)
	}

	c.stats.Recommendations.Store(int64(len(c.recommendations.recommendations)))
}

func (c *CostOptimizer) GetRecommendations() []CostRecommendation {
	c.recommendations.mu.RLock()
	defer c.recommendations.mu.RUnlock()

	result := make([]CostRecommendation, len(c.recommendations.recommendations))
	copy(result, c.recommendations.recommendations)

	sort.Slice(result, func(i, j int) bool {
		return result[i].Priority < result[j].Priority
	})

	return result
}

func (c *CostOptimizer) GetStats() map[string]interface{} {
	return map[string]interface{}{
		"total_cost":          c.stats.TotalCost.Load(),
		"projected_cost":     c.stats.ProjectedCost.Load(),
		"total_savings":      c.stats.TotalSavings.Load(),
		"recommendations":     c.stats.Recommendations.Load(),
		"budget_limit":       c.config.BudgetLimit,
		"budget_period":      c.config.BudgetPeriod,
		"optimize_for":       c.config.OptimizeFor.String(),
	}
}

func (o CostObjective) String() string {
	switch o {
	case CostMinimize:
		return "CostMinimize"
	case PerformanceMaximize:
		return "PerformanceMaximize"
	case Balanced:
		return "Balanced"
	default:
		return "Unknown"
	}
}

func NewCostHistory(maxSize int) *CostHistory {
	return &CostHistory{
		records: make([]CostRecord, 0, maxSize),
		maxSize: maxSize,
	}
}

func NewCostRecommendationEngine() *CostRecommendationEngine {
	return &CostRecommendationEngine{
		recommendations: make([]CostRecommendation, 0),
		strategies:      make([]OptimizationStrategy, 0),
	}
}

func NewResourceUsagePredictor(ctx context.Context) *ResourceUsagePredictor {
	if ctx == nil {
		ctx = context.Background()
	}
	childCtx, cancel := context.WithCancel(ctx)

	return &ResourceUsagePredictor{
		ctx:           childCtx,
		cancel:        cancel,
		predictor:     NewResourcePredictorV2(ctx),
	}
}

func (p *ResourceUsagePredictor) Start() error {
	return p.predictor.Start()
}

func (p *ResourceUsagePredictor) Stop() {
	p.predictor.Stop()
}

func (p *ResourceUsagePredictor) RecordResourceUsage(usage *ResourceUsage) error {
	timestamp := time.Now()

	cpuPoint := TimeSeriesPoint{
		Timestamp: timestamp,
		Value:     usage.CPUPercent,
	}
	if err := p.predictor.AddDataPoint("cpu", cpuPoint); err != nil {
		return err
	}

	memoryPoint := TimeSeriesPoint{
		Timestamp: timestamp,
		Value:     usage.MemoryPercent,
	}
	if err := p.predictor.AddDataPoint("memory", memoryPoint); err != nil {
		return err
	}

	return nil
}

func (p *ResourceUsagePredictor) Forecast(horizon time.Duration) (*ResourceForecast, error) {
	cpuPredictions, err := p.predictor.Predict("cpu", horizon)
	if err != nil {
		return nil, err
	}

	memoryPredictions, err := p.predictor.Predict("memory", horizon)
	if err != nil {
		return nil, err
	}

	var cpuAvg, memoryAvg float64
	if len(cpuPredictions) > 0 {
		for _, p := range cpuPredictions {
			cpuAvg += p.Value
		}
		cpuAvg /= float64(len(cpuPredictions))
	}
	if len(memoryPredictions) > 0 {
		for _, p := range memoryPredictions {
			memoryAvg += p.Value
		}
		memoryAvg /= float64(len(memoryPredictions))
	}

	var confidence float64
	if len(cpuPredictions) > 0 {
		confidence = cpuPredictions[0].Confidence
	}

	return &ResourceForecast{
		Timestamp:    time.Now().Add(horizon),
		CPUUsage:     cpuAvg,
		MemoryUsage:  memoryAvg,
		Confidence:   confidence,
	}, nil
}

type ResourceUsage struct {
	CPUPercent    float64
	MemoryPercent float64
	DiskPercent   float64
	NetworkBytes  int64
}

func init() {
	go func() {
		select {}
	}()
}
