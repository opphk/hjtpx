package ai

import (
	"time"
)

type ModelType string

const (
	ModelTypeMLP ModelType = "mlp"
)

type AIModel interface {
	GetModelType() ModelType
	GetInputDimension() int
	GetOutputDimension() int
	IsReady() bool
}

type ModelConfig struct {
	ModelType       ModelType
	InputDim        int
	HiddenDims      []int
	OutputDim       int
	LearningRate    float64
	WeightDecay     float64
	ModelPath       string
	EnableCache     bool
	BatchSize       int
	Epochs          int
	EarlyStopping   bool
	ValidationSplit float64
	DropoutRate     float64
}

func DefaultModelConfig() *ModelConfig {
	return &ModelConfig{
		ModelType:       ModelTypeMLP,
		InputDim:        FeatureDimension,
		HiddenDims:      []int{64, 32, 16},
		OutputDim:       1,
		LearningRate:    0.001,
		WeightDecay:     0.0001,
		BatchSize:       32,
		Epochs:          100,
		EarlyStopping:   true,
		ValidationSplit: 0.2,
		DropoutRate:    0.0,
	}
}

type RiskLevel string

const (
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

type Action string

const (
	ActionAllow  Action = "allow"
	ActionVerify Action = "verify"
	ActionBlock  Action = "block"
)

type PredictionResult struct {
	Score      float64
	RiskLevel  RiskLevel
	Action     Action
	Confidence float64
	Timestamp  time.Time
	ModelType string
	Features  []float64
}

type TrainingResult struct {
	Epoch                int
	TrainLoss            float64
	ValidationLoss       float64
	TrainAccuracy        float64
	ValidationAccuracy   float64
	BestModelPath        string
	TrainingTime         time.Duration
}

type TrainingData struct {
	Features [][]float64
	Labels   []float64
	Weights  []float64
}

type ValidationMetrics struct {
	Accuracy        float64
	Precision       float64
	Recall          float64
	F1Score         float64
	AUC             float64
	ConfusionMatrix [][]int
}

func NewPredictionResult(score float64, level RiskLevel, action Action) *PredictionResult {
	return &PredictionResult{
		Score:      score,
		RiskLevel:  level,
		Action:     action,
		Confidence: calculateConfidence(score),
		Timestamp:  time.Now(),
	}
}

func calculateConfidence(score float64) float64 {
	absScore := score
	if score < 0 {
		absScore = 1.0 + score
	}
	confidence := absScore * 2.0
	if confidence > 1.0 {
		confidence = 1.0
	}
	return confidence
}

func (p *PredictionResult) IsHighRisk() bool {
	return p.RiskLevel == RiskLevelHigh || p.RiskLevel == RiskLevelCritical
}

func (p *PredictionResult) ShouldBlock() bool {
	return p.Action == ActionBlock
}
