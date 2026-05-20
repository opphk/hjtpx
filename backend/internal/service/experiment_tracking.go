package service

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"time"
)

var (
	ErrRunNotFound = errors.New("run not found")
	ErrMetricNotFound = errors.New("metric not found")
)

type ExperimentTrackingService interface {
	CreateRun(ctx context.Context, run *ExperimentRun) error
	GetRun(ctx context.Context, runID string) (*ExperimentRun, error)
	UpdateRun(ctx context.Context, run *ExperimentRun) error
	DeleteRun(ctx context.Context, runID string) error
	ListRuns(ctx context.Context, filters *RunFilters) ([]*ExperimentRun, error)
	LogMetric(ctx context.Context, runID string, metric *MetricLog) error
	GetMetrics(ctx context.Context, runID string) ([]*MetricLog, error)
	LogParameter(ctx context.Context, runID string, param *ParameterLog) error
	GetParameters(ctx context.Context, runID string) ([]*ParameterLog, error)
	LogArtifact(ctx context.Context, runID string, artifact *ArtifactLog) error
	GetArtifacts(ctx context.Context, runID string) ([]*ArtifactLog, error)
	CompareRuns(ctx context.Context, runIDs []string) (*RunComparison, error)
	GetBestRun(ctx context.Context, experimentID string, metricName string) (*ExperimentRun, error)
}

type ExperimentRun struct {
	RunID        string          `json:"run_id"`
	ExperimentID string          `json:"experiment_id"`
	Name         string          `json:"name"`
	Description  string          `json:"description"`
	Status       string          `json:"status"`
	StartTime   time.Time       `json:"start_time"`
	EndTime     *time.Time      `json:"end_time,omitempty"`
	Duration    time.Duration   `json:"duration"`
	Parameters  map[string]interface{} `json:"parameters"`
	Metrics     map[string]float64 `json:"metrics"`
	Tags        []string        `json:"tags"`
	Metadata    map[string]interface{} `json:"metadata"`
	CreatedAt   time.Time       `json:"created_at"`
	UpdatedAt   time.Time       `json:"updated_at"`
}

type RunFilters struct {
	ExperimentID string   `json:"experiment_id,omitempty"`
	Status      string   `json:"status,omitempty"`
	Tags        []string `json:"tags,omitempty"`
	StartTime   *time.Time `json:"start_time,omitempty"`
	EndTime     *time.Time `json:"end_time,omitempty"`
	Page        int      `json:"page"`
	PageSize    int      `json:"page_size"`
}

type MetricLog struct {
	MetricID   string    `json:"metric_id"`
	RunID      string    `json:"run_id"`
	Name       string    `json:"name"`
	Value      float64   `json:"value"`
	Step       int64     `json:"step"`
	Timestamp  time.Time `json:"timestamp"`
	Metadata   map[string]string `json:"metadata,omitempty"`
}

type ParameterLog struct {
	ParameterID string                 `json:"parameter_id"`
	RunID      string                 `json:"run_id"`
	Name       string                 `json:"name"`
	Value      interface{}            `json:"value"`
	Type       string                 `json:"type"`
	Timestamp  time.Time             `json:"timestamp"`
}

type ArtifactLog struct {
	ArtifactID  string    `json:"artifact_id"`
	RunID      string    `json:"run_id"`
	Name       string    `json:"name"`
	Type       string    `json:"type"`
	URI        string    `json:"uri"`
	SizeBytes  int64     `json:"size_bytes"`
	Checksum   string    `json:"checksum"`
	Metadata   map[string]string `json:"metadata,omitempty"`
	CreatedAt  time.Time `json:"created_at"`
}

type RunComparison struct {
	RunIDs       []string           `json:"run_ids"`
	Metrics      map[string][]float64 `json:"metrics"`
	BestRunID    string             `json:"best_run_id"`
	BestMetric   string             `json:"best_metric"`
	BestValue    float64            `json:"best_value"`
	Improvements map[string]float64 `json:"improvements"`
}

type experimentTrackingService struct {
	runs       map[string]*ExperimentRun
	metrics    map[string][]*MetricLog
	parameters map[string][]*ParameterLog
	artifacts  map[string][]*ArtifactLog
	mu         sync.RWMutex
}

func NewExperimentTrackingService() ExperimentTrackingService {
	return &experimentTrackingService{
		runs:       make(map[string]*ExperimentRun),
		metrics:    make(map[string][]*MetricLog),
		parameters: make(map[string][]*ParameterLog),
		artifacts:  make(map[string][]*ArtifactLog),
	}
}

func (s *experimentTrackingService) CreateRun(ctx context.Context, run *ExperimentRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if run.RunID == "" {
		run.RunID = fmt.Sprintf("run-%d", time.Now().UnixNano())
	}

	run.CreatedAt = time.Now()
	run.UpdatedAt = time.Now()
	run.StartTime = time.Now()

	if run.Status == "" {
		run.Status = "running"
	}

	if run.Parameters == nil {
		run.Parameters = make(map[string]interface{})
	}

	if run.Metrics == nil {
		run.Metrics = make(map[string]float64)
	}

	s.runs[run.RunID] = run
	s.metrics[run.RunID] = []*MetricLog{}
	s.parameters[run.RunID] = []*ParameterLog{}
	s.artifacts[run.RunID] = []*ArtifactLog{}

	return nil
}

func (s *experimentTrackingService) GetRun(ctx context.Context, runID string) (*ExperimentRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	run, exists := s.runs[runID]
	if !exists {
		return nil, ErrRunNotFound
	}

	return run, nil
}

func (s *experimentTrackingService) UpdateRun(ctx context.Context, run *ExperimentRun) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[run.RunID]; !exists {
		return ErrRunNotFound
	}

	run.UpdatedAt = time.Now()

	if run.EndTime != nil && run.StartTime.IsZero() == false {
		run.Duration = run.EndTime.Sub(run.StartTime)
	}

	s.runs[run.RunID] = run
	return nil
}

func (s *experimentTrackingService) DeleteRun(ctx context.Context, runID string) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[runID]; !exists {
		return ErrRunNotFound
	}

	delete(s.runs, runID)
	delete(s.metrics, runID)
	delete(s.parameters, runID)
	delete(s.artifacts, runID)

	return nil
}

func (s *experimentTrackingService) ListRuns(ctx context.Context, filters *RunFilters) ([]*ExperimentRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var result []*ExperimentRun
	for _, run := range s.runs {
		if s.matchesFilters(run, filters) {
			result = append(result, run)
		}
	}

	return result, nil
}

func (s *experimentTrackingService) matchesFilters(run *ExperimentRun, filters *RunFilters) bool {
	if filters == nil {
		return true
	}

	if filters.ExperimentID != "" && run.ExperimentID != filters.ExperimentID {
		return false
	}

	if filters.Status != "" && run.Status != filters.Status {
		return false
	}

	if len(filters.Tags) > 0 {
		hasTag := false
		for _, tag := range filters.Tags {
			for _, runTag := range run.Tags {
				if tag == runTag {
					hasTag = true
					break
				}
			}
			if hasTag {
				break
			}
		}
		if !hasTag {
			return false
		}
	}

	return true
}

func (s *experimentTrackingService) LogMetric(ctx context.Context, runID string, metric *MetricLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[runID]; !exists {
		return ErrRunNotFound
	}

	if metric.MetricID == "" {
		metric.MetricID = fmt.Sprintf("metric-%d", time.Now().UnixNano())
	}

	metric.RunID = runID
	metric.Timestamp = time.Now()

	s.metrics[runID] = append(s.metrics[runID], metric)

	if s.runs[runID].Metrics == nil {
		s.runs[runID].Metrics = make(map[string]float64)
	}
	s.runs[runID].Metrics[metric.Name] = metric.Value
	s.runs[runID].UpdatedAt = time.Now()

	return nil
}

func (s *experimentTrackingService) GetMetrics(ctx context.Context, runID string) ([]*MetricLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.runs[runID]; !exists {
		return nil, ErrRunNotFound
	}

	return s.metrics[runID], nil
}

func (s *experimentTrackingService) LogParameter(ctx context.Context, runID string, param *ParameterLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[runID]; !exists {
		return ErrRunNotFound
	}

	if param.ParameterID == "" {
		param.ParameterID = fmt.Sprintf("param-%d", time.Now().UnixNano())
	}

	param.RunID = runID
	param.Timestamp = time.Now()

	s.parameters[runID] = append(s.parameters[runID], param)

	if s.runs[runID].Parameters == nil {
		s.runs[runID].Parameters = make(map[string]interface{})
	}
	s.runs[runID].Parameters[param.Name] = param.Value
	s.runs[runID].UpdatedAt = time.Now()

	return nil
}

func (s *experimentTrackingService) GetParameters(ctx context.Context, runID string) ([]*ParameterLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.runs[runID]; !exists {
		return nil, ErrRunNotFound
	}

	return s.parameters[runID], nil
}

func (s *experimentTrackingService) LogArtifact(ctx context.Context, runID string, artifact *ArtifactLog) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if _, exists := s.runs[runID]; !exists {
		return ErrRunNotFound
	}

	if artifact.ArtifactID == "" {
		artifact.ArtifactID = fmt.Sprintf("artifact-%d", time.Now().UnixNano())
	}

	artifact.RunID = runID
	artifact.CreatedAt = time.Now()

	s.artifacts[runID] = append(s.artifacts[runID], artifact)
	s.runs[runID].UpdatedAt = time.Now()

	return nil
}

func (s *experimentTrackingService) GetArtifacts(ctx context.Context, runID string) ([]*ArtifactLog, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if _, exists := s.runs[runID]; !exists {
		return nil, ErrRunNotFound
	}

	return s.artifacts[runID], nil
}

func (s *experimentTrackingService) CompareRuns(ctx context.Context, runIDs []string) (*RunComparison, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	metrics := make(map[string][]float64)
	var bestRunID string
	var bestMetric string
	var bestValue float64

	for _, runID := range runIDs {
		run, exists := s.runs[runID]
		if !exists {
			continue
		}

		for name, value := range run.Metrics {
			metrics[name] = append(metrics[name], value)
			if bestValue == 0 || value > bestValue {
				bestValue = value
				bestMetric = name
				bestRunID = runID
			}
		}
	}

	improvements := make(map[string]float64)
	if len(runIDs) > 1 {
		firstRun := s.runs[runIDs[0]]
		if firstRun != nil {
			for name, value := range firstRun.Metrics {
				if bestValue > 0 && value > 0 {
					improvements[name] = (bestValue - value) / value * 100
				}
			}
		}
	}

	return &RunComparison{
		RunIDs:       runIDs,
		Metrics:      metrics,
		BestRunID:    bestRunID,
		BestMetric:   bestMetric,
		BestValue:    bestValue,
		Improvements: improvements,
	}, nil
}

func (s *experimentTrackingService) GetBestRun(ctx context.Context, experimentID string, metricName string) (*ExperimentRun, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var bestRun *ExperimentRun
	var bestValue float64

	for _, run := range s.runs {
		if run.ExperimentID != experimentID {
			continue
		}

		if value, exists := run.Metrics[metricName]; exists {
			if bestRun == nil || value > bestValue {
				bestValue = value
				bestRun = run
			}
		}
	}

	if bestRun == nil {
		return nil, ErrRunNotFound
	}

	return bestRun, nil
}
