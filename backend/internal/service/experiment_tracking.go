package service

import (
	"context"
	"time"
)

type ExperimentTrackingService struct{}

type Experiment struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Type        string    `json:"type"`
	Status      string    `json:"status"`
	CreatedBy   string    `json:"createdBy"`
	CreatedAt   time.Time `json:"createdAt"`
	StartedAt   time.Time `json:"startedAt,omitempty"`
	EndedAt     time.Time `json:"endedAt,omitempty"`
	Tags        []string  `json:"tags,omitempty"`
}

type ExperimentMetric struct {
	ID           uint      `json:"id"`
	ExperimentID uint      `json:"experimentId"`
	Name         string    `json:"name"`
	Value        float64   `json:"value"`
	Unit         string    `json:"unit"`
	Timestamp    time.Time `json:"timestamp"`
	Step         int64     `json:"step,omitempty"`
}

type ExperimentRun struct {
	ID           uint      `json:"id"`
	ExperimentID uint      `json:"experimentId"`
	Name         string    `json:"name"`
	Status       string    `json:"status"`
	StartTime    time.Time `json:"startTime"`
	EndTime      time.Time `json:"endTime,omitempty"`
	Duration     string    `json:"duration,omitempty"`
	Metrics      map[string]float64 `json:"metrics,omitempty"`
	Params       map[string]interface{} `json:"params,omitempty"`
}

type CreateExperimentRequest struct {
	Name        string                 `json:"name"`
	Description string                 `json:"description"`
	Type        string                 `json:"type"`
	Tags        []string               `json:"tags,omitempty"`
}

type LogMetricRequest struct {
	ExperimentID uint    `json:"experimentId"`
	RunID        uint    `json:"runId,omitempty"`
	Name         string  `json:"name"`
	Value        float64 `json:"value"`
	Unit         string  `json:"unit"`
	Step         int64   `json:"step,omitempty"`
}

func NewExperimentTrackingService() *ExperimentTrackingService {
	return &ExperimentTrackingService{}
}

func (s *ExperimentTrackingService) ListExperiments(ctx context.Context) ([]*Experiment, error) {
	experiments := []*Experiment{
		{
			ID:          1,
			Name:        "Captcha Model Optimization",
			Description: "优化验证码分类模型准确率和性能",
			Type:        "optimization",
			Status:      "running",
			CreatedBy:   "admin",
			CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
			StartedAt:   time.Now().Add(-25 * 24 * time.Hour),
			Tags:        []string{"captcha", "classification", "optimization"},
		},
		{
			ID:          2,
			Name:        "Risk Detection Model v2",
			Description: "开发新一代风险检测模型",
			Type:        "research",
			Status:      "completed",
			CreatedBy:   "researcher",
			CreatedAt:   time.Now().Add(-60 * 24 * time.Hour),
			StartedAt:   time.Now().Add(-55 * 24 * time.Hour),
			EndedAt:     time.Now().Add(-20 * 24 * time.Hour),
			Tags:        []string{"risk", "detection", "research"},
		},
	}
	return experiments, nil
}

func (s *ExperimentTrackingService) GetExperiment(ctx context.Context, id uint) (*Experiment, error) {
	experiment := &Experiment{
		ID:          id,
		Name:        "Captcha Model Optimization",
		Description: "优化验证码分类模型准确率和性能",
		Type:        "optimization",
		Status:      "running",
		CreatedBy:   "admin",
		CreatedAt:   time.Now().Add(-30 * 24 * time.Hour),
		StartedAt:   time.Now().Add(-25 * 24 * time.Hour),
		Tags:        []string{"captcha", "classification", "optimization"},
	}
	return experiment, nil
}

func (s *ExperimentTrackingService) CreateExperiment(ctx context.Context, req *CreateExperimentRequest) (*Experiment, error) {
	experiment := &Experiment{
		ID:          uint(time.Now().Unix()),
		Name:        req.Name,
		Description: req.Description,
		Type:        req.Type,
		Status:      "draft",
		CreatedBy:   "admin",
		CreatedAt:   time.Now(),
		Tags:        req.Tags,
	}
	return experiment, nil
}

func (s *ExperimentTrackingService) StartExperiment(ctx context.Context, id uint) (*Experiment, error) {
	experiment := &Experiment{
		ID:        id,
		Status:    "running",
		StartedAt: time.Now(),
	}
	return experiment, nil
}

func (s *ExperimentTrackingService) EndExperiment(ctx context.Context, id uint) (*Experiment, error) {
	experiment := &Experiment{
		ID:      id,
		Status:  "completed",
		EndedAt: time.Now(),
	}
	return experiment, nil
}

func (s *ExperimentTrackingService) DeleteExperiment(ctx context.Context, id uint) error {
	return nil
}

func (s *ExperimentTrackingService) ListRuns(ctx context.Context, experimentID uint) ([]*ExperimentRun, error) {
	runs := []*ExperimentRun{
		{
			ID:           1,
			ExperimentID: experimentID,
			Name:         "Run 1 - Baseline",
			Status:       "completed",
			StartTime:    time.Now().Add(-20 * 24 * time.Hour),
			EndTime:      time.Now().Add(-18 * 24 * time.Hour),
			Duration:     "48h",
			Metrics:      map[string]float64{"accuracy": 0.95, "loss": 0.15, "latency": 48.2},
			Params:       map[string]interface{}{"learning_rate": 0.001, "batch_size": 32},
		},
		{
			ID:           2,
			ExperimentID: experimentID,
			Name:         "Run 2 - Optimized",
			Status:       "running",
			StartTime:    time.Now().Add(-5 * 24 * time.Hour),
			Metrics:      map[string]float64{"accuracy": 0.97, "loss": 0.10, "latency": 42.5},
			Params:       map[string]interface{}{"learning_rate": 0.0005, "batch_size": 64},
		},
	}
	return runs, nil
}

func (s *ExperimentTrackingService) GetRun(ctx context.Context, id uint) (*ExperimentRun, error) {
	run := &ExperimentRun{
		ID:           id,
		ExperimentID: 1,
		Name:         "Run 1 - Baseline",
		Status:       "completed",
		StartTime:    time.Now().Add(-20 * 24 * time.Hour),
		EndTime:      time.Now().Add(-18 * 24 * time.Hour),
		Duration:     "48h",
		Metrics:      map[string]float64{"accuracy": 0.95, "loss": 0.15, "latency": 48.2},
		Params:       map[string]interface{}{"learning_rate": 0.001, "batch_size": 32},
	}
	return run, nil
}

func (s *ExperimentTrackingService) ListMetrics(ctx context.Context, experimentID uint, runID uint) ([]*ExperimentMetric, error) {
	metrics := []*ExperimentMetric{}
	baseTime := time.Now().Add(-24 * time.Hour)
	
	for i := 0; i < 24; i++ {
		timestamp := baseTime.Add(time.Duration(i) * time.Hour)
		accuracy := 0.94 + float64(i)*0.001
		loss := 0.16 - float64(i)*0.002
		
		metrics = append(metrics, &ExperimentMetric{
			ID:           uint(i*2 + 1),
			ExperimentID: experimentID,
			Name:         "accuracy",
			Value:        accuracy,
			Unit:         "",
			Timestamp:    timestamp,
			Step:         int64(i * 100),
		})
		metrics = append(metrics, &ExperimentMetric{
			ID:           uint(i*2 + 2),
			ExperimentID: experimentID,
			Name:         "loss",
			Value:        loss,
			Unit:         "",
			Timestamp:    timestamp,
			Step:         int64(i * 100),
		})
	}
	
	return metrics, nil
}

func (s *ExperimentTrackingService) LogMetric(ctx context.Context, req *LogMetricRequest) (*ExperimentMetric, error) {
	metric := &ExperimentMetric{
		ID:           uint(time.Now().Unix()),
		ExperimentID: req.ExperimentID,
		Name:         req.Name,
		Value:        req.Value,
		Unit:         req.Unit,
		Timestamp:    time.Now(),
		Step:         req.Step,
	}
	return metric, nil
}
