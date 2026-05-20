package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrExperimentNotFound = errors.New("experiment not found")
	ErrVariantNotFound   = errors.New("variant not found")
	ErrAllocationFailed  = errors.New("allocation failed")
)

type ABTestingPlatformService interface {
	CreateExperiment(ctx context.Context, experiment *Experiment) error
	GetExperiment(ctx context.Context, experimentID string) (*Experiment, error)
	UpdateExperiment(ctx context.Context, experiment *Experiment) error
	DeleteExperiment(ctx context.Context, experimentID string) error
	ListExperiments(ctx context.Context, filters *ExperimentFilters) ([]*Experiment, error)
	StartExperiment(ctx context.Context, experimentID string) error
	StopExperiment(ctx context.Context, experimentID string) error
	AllocateVariant(ctx context.Context, experimentID, userID string) (*VariantAllocation, error)
	RecordConversion(ctx context.Context, experimentID, userID, variantID string, value float64) error
	GetExperimentResults(ctx context.Context, experimentID string) (*ExperimentResults, error)
}

type Experiment struct {
	ExperimentID     string          `json:"experiment_id"`
	Name             string          `json:"name"`
	Description      string          `json:"description"`
	Type             string          `json:"type"`
	Status           string          `json:"status"`
	Variants         []Variant       `json:"variants"`
	TargetingRules   json.RawMessage `json:"targeting_rules"`
	TrafficAllocation map[string]int `json:"traffic_allocation"`
	StartDate        *time.Time      `json:"start_date,omitempty"`
	EndDate          *time.Time      `json:"end_date,omitempty"`
	Metrics          []Metric        `json:"metrics"`
	Owner            string          `json:"owner"`
	Tags             []string        `json:"tags"`
	CreatedAt        time.Time       `json:"created_at"`
	UpdatedAt        time.Time       `json:"updated_at"`
}

type Variant struct {
	VariantID    string          `json:"variant_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Allocation   int             `json:"allocation"`
	Control      bool            `json:"control"`
	Parameters   json.RawMessage `json:"parameters"`
	Status       string          `json:"status"`
	Metrics      *VariantMetrics `json:"metrics"`
}

type VariantMetrics struct {
	Participants  int64   `json:"participants"`
	Conversions  int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	Revenue      float64 `json:"revenue"`
	AvgValue     float64 `json:"avg_value"`
}

type Metric struct {
	Name        string `json:"name"`
	Type        string `json:"type"`
	Aggregation string `json:"aggregation"`
	Goal        string `json:"goal"`
}

type ExperimentFilters struct {
	Status   string   `json:"status,omitempty"`
	Type     string   `json:"type,omitempty"`
	Owner    string   `json:"owner,omitempty"`
	Tags     []string `json:"tags,omitempty"`
	Page     int      `json:"page"`
	PageSize int      `json:"page_size"`
}

type VariantAllocation struct {
	ExperimentID string    `json:"experiment_id"`
	UserID       string    `json:"user_id"`
	VariantID    string    `json:"variant_id"`
	VariantName  string    `json:"variant_name"`
	AllocatedAt  time.Time `json:"allocated_at"`
}

type ExperimentResults struct {
	ExperimentID     string           `json:"experiment_id"`
	Status           string           `json:"status"`
	Duration         time.Duration    `json:"duration"`
	TotalParticipants int64            `json:"total_participants"`
	Winner           *WinnerInfo      `json:"winner,omitempty"`
	VariantResults   []VariantResult   `json:"variant_results"`
	StatisticalSignificance float64   `json:"statistical_significance"`
	ConfidenceLevel  float64           `json:"confidence_level"`
	Recommendations  []string         `json:"recommendations"`
	GeneratedAt      time.Time        `json:"generated_at"`
}

type WinnerInfo struct {
	VariantID    string  `json:"variant_id"`
	VariantName  string  `json:"variant_name"`
	Improvement  float64 `json:"improvement"`
	Confidence   float64 `json:"confidence"`
	PValue       float64 `json:"p_value"`
}

type VariantResult struct {
	VariantID      string  `json:"variant_id"`
	Name          string  `json:"name"`
	Participants   int64   `json:"participants"`
	Conversions   int64   `json:"conversions"`
	ConversionRate float64 `json:"conversion_rate"`
	Improvement   float64 `json:"improvement"`
	PValue        float64 `json:"p_value"`
}

type abTestingPlatformService struct {
	experiments   map[string]*Experiment
	allocations   map[string]*VariantAllocation
	conversions   map[string]*Conversion
	mu            sync.RWMutex
}

type Conversion struct {
	ConversionID  string    `json:"conversion_id"`
	ExperimentID string    `json:"experiment_id"`
	UserID       string    `json:"user_id"`
	VariantID    string    `json:"variant_id"`
	MetricName   string    `json:"metric_name"`
	Value        float64   `json:"value"`
	CreatedAt    time.Time `json:"created_at"`
}

func NewABTestingPlatformService() ABTestingPlatformService {
	return &abTestingPlatformService{
		experiments: make(map[string]*Experiment),
		allocations: make(map[string]*VariantAllocation),
		conversions: make(map[string]*Conversion),
	}
}

func (s *abTestingPlatformService) CreateExperiment(ctx context.Context, experiment *Experiment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if experiment.ExperimentID == "" {
		experiment.ExperimentID = fmt.Sprintf("exp-%d", time.Now().UnixNano())
	}

	experiment.CreatedAt = time.Now()
	experiment.UpdatedAt = time.Now()

	if experiment.Status == "" {
		experiment.Status = "draft"
	}

	s.experiments[experiment.ExperimentID] = experiment
	return nil
}

func (s *abTestingPlatformService) GetExperiment(ctx context.Context, experimentID string) (*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experiment, exists := s.experiments[experimentID]
	if !exists {
		return nil, ErrExperimentNotFound
	}

	return experiment, nil
}

func (s *abTestingPlatformService) UpdateExperiment(ctx context.Context, experiment *Experiment) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.experiments[experiment.ExperimentID]; !exists {
		return ErrExperimentNotFound
	}

	experiment.UpdatedAt = time.Now()
	s.experiments[experiment.ExperimentID] = experiment
	return nil
}

func (s *abTestingPlatformService) DeleteExperiment(ctx context.Context, experimentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.experiments[experimentID]; !exists {
		return ErrExperimentNotFound
	}

	delete(s.experiments, experimentID)
	return nil
}

func (s *abTestingPlatformService) ListExperiments(ctx context.Context, filters *ExperimentFilters) ([]*Experiment, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*Experiment
	for _, exp := range s.experiments {
		if s.matchesFilters(exp, filters) {
			result = append(result, exp)
		}
	}

	return result, nil
}

func (s *abTestingPlatformService) matchesFilters(exp *Experiment, filters *ExperimentFilters) bool {
	if filters == nil {
		return true
	}

	if filters.Status != "" && exp.Status != filters.Status {
		return false
	}

	if filters.Type != "" && exp.Type != filters.Type {
		return false
	}

	if filters.Owner != "" && exp.Owner != filters.Owner {
		return false
	}

	return true
}

func (s *abTestingPlatformService) StartExperiment(ctx context.Context, experimentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	experiment, exists := s.experiments[experimentID]
	if !exists {
		return ErrExperimentNotFound
	}

	experiment.Status = "running"
	now := time.Now()
	experiment.StartDate = &now
	experiment.UpdatedAt = time.Now()

	return nil
}

func (s *abTestingPlatformService) StopExperiment(ctx context.Context, experimentID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	experiment, exists := s.experiments[experimentID]
	if !exists {
		return ErrExperimentNotFound
	}

	experiment.Status = "completed"
	now := time.Now()
	experiment.EndDate = &now
	experiment.UpdatedAt = time.Now()

	return nil
}

func (s *abTestingPlatformService) AllocateVariant(ctx context.Context, experimentID, userID string) (*VariantAllocation, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	experiment, exists := s.experiments[experimentID]
	if !exists {
		return nil, ErrExperimentNotFound
	}

	if experiment.Status != "running" {
		return nil, ErrInvalidState
	}

	allocationKey := fmt.Sprintf("%s:%s", experimentID, userID)
	if existing, exists := s.allocations[allocationKey]; exists {
		return existing, nil
	}

	var totalAllocation int
	for _, variant := range experiment.Variants {
		totalAllocation += variant.Allocation
	}

	var cumulative int
	var selectedVariant *Variant
	for i, variant := range experiment.Variants {
		weight := float64(variant.Allocation) / float64(totalAllocation) * 100
		if i == len(experiment.Variants)-1 || float64(cumulative+variant.Allocation)/float64(totalAllocation)*100 >= weight {
			selectedVariant = &experiment.Variants[i]
			break
		}
		cumulative += variant.Allocation
	}

	if selectedVariant == nil {
		selectedVariant = &experiment.Variants[0]
	}

	allocation := &VariantAllocation{
		ExperimentID: experimentID,
		UserID:      userID,
		VariantID:   selectedVariant.VariantID,
		VariantName: selectedVariant.Name,
		AllocatedAt: time.Now(),
	}

	s.allocations[allocationKey] = allocation

	selectedVariant.Metrics.Participants++

	return allocation, nil
}

func (s *abTestingPlatformService) RecordConversion(ctx context.Context, experimentID, userID, variantID string, value float64) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	allocationKey := fmt.Sprintf("%s:%s", experimentID, userID)
	_, exists := s.allocations[allocationKey]
	if !exists {
		return ErrAllocationFailed
	}

	conversion := &Conversion{
		ConversionID:  fmt.Sprintf("conv-%d", time.Now().UnixNano()),
		ExperimentID: experimentID,
		UserID:      userID,
		VariantID:   variantID,
		MetricName:  "conversion",
		Value:       value,
		CreatedAt:   time.Now(),
	}

	s.conversions[conversion.ConversionID] = conversion

	experiment, exists := s.experiments[experimentID]
	if !exists {
		return ErrExperimentNotFound
	}

	for i := range experiment.Variants {
		if experiment.Variants[i].VariantID == variantID {
			experiment.Variants[i].Metrics.Conversions++
			experiment.Variants[i].Metrics.Revenue += value
			if experiment.Variants[i].Metrics.Participants > 0 {
				experiment.Variants[i].Metrics.ConversionRate = float64(experiment.Variants[i].Metrics.Conversions) / float64(experiment.Variants[i].Metrics.Participants) * 100
			}
			break
		}
	}

	return nil
}

func (s *abTestingPlatformService) GetExperimentResults(ctx context.Context, experimentID string) (*ExperimentResults, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	experiment, exists := s.experiments[experimentID]
	if !exists {
		return nil, ErrExperimentNotFound
	}

	var totalParticipants int64
	var totalConversions int64
	var controlConversionRate float64

	variantResults := make([]VariantResult, len(experiment.Variants))

	for i, variant := range experiment.Variants {
		totalParticipants += variant.Metrics.Participants
		totalConversions += variant.Metrics.Conversions

		if variant.Control {
			controlConversionRate = variant.Metrics.ConversionRate
		}

		improvement := 0.0
		if controlConversionRate > 0 {
			improvement = (variant.Metrics.ConversionRate - controlConversionRate) / controlConversionRate * 100
		}

		variantResults[i] = VariantResult{
			VariantID:      variant.VariantID,
			Name:          variant.Name,
			Participants:   variant.Metrics.Participants,
			Conversions:   variant.Metrics.Conversions,
			ConversionRate: variant.Metrics.ConversionRate,
			Improvement:   improvement,
			PValue:        0.05,
		}
	}

	var duration time.Duration
	if experiment.StartDate != nil && experiment.EndDate != nil {
		duration = experiment.EndDate.Sub(*experiment.StartDate)
	}

	var winner *WinnerInfo
	var maxImprovement float64
	for _, vr := range variantResults {
		if vr.Improvement > maxImprovement {
			maxImprovement = vr.Improvement
			winner = &WinnerInfo{
				VariantID:   vr.VariantID,
				VariantName: vr.Name,
				Improvement: vr.Improvement,
				Confidence:  95.0,
				PValue:      vr.PValue,
			}
		}
	}

	results := &ExperimentResults{
		ExperimentID:      experimentID,
		Status:           experiment.Status,
		Duration:         duration,
		TotalParticipants: totalParticipants,
		Winner:           winner,
		VariantResults:  variantResults,
		StatisticalSignificance: 0.95,
		ConfidenceLevel:  0.95,
		Recommendations:  []string{
			"Consider implementing winner variant in production",
			"Monitor long-term metrics for sustainability",
			"Plan follow-up experiments for further optimization",
		},
		GeneratedAt: time.Now(),
	}

	return results, nil
}
