package config

import "time"

type RiskConfig struct {
	SlideSpeedThresholdFast       time.Duration
	SlideSpeedThresholdSlow       time.Duration
	SmoothnessThreshold           float64
	JitterThreshold               float64
	MaxFailureCount               int
	CriticalFailureCount          int
	BlockDuration                 time.Duration
	HighFrequencyThreshold        int64
	RiskScoreThresholds           RiskScoreThresholds
	// 新添加的配置项
	AccelerationThreshold         float64     // 加速度异常阈值
	VelocityConsistencyThreshold  float64     // 速度一致性阈值
	ClickIntervalMin              int64       // 最小点击间隔(毫秒)
	ClickIntervalMax              int64       // 最大点击间隔(毫秒)
	DoubleClickThreshold          int64       // 双击判断阈值(毫秒)
	PathCurvatureThreshold        float64     // 路径曲率阈值
	DirectionChangeThreshold      int         // 方向变化次数阈值
	PathComplexityThreshold       float64     // 路径复杂度阈值
	HesitationThreshold           int64       // 犹豫时间阈值(毫秒)
}

type RiskScoreThresholds struct {
	Low      int
	Medium   int
	High     int
	Critical int
}

func DefaultRiskConfig() *RiskConfig {
	return &RiskConfig{
		SlideSpeedThresholdFast:       1 * time.Second,
		SlideSpeedThresholdSlow:       30 * time.Second,
		SmoothnessThreshold:           0.95,
		JitterThreshold:               0.1,
		MaxFailureCount:               3,
		CriticalFailureCount:          5,
		BlockDuration:                 30 * time.Minute,
		HighFrequencyThreshold:        100,
		// 新添加的默认配置值
		AccelerationThreshold:         500.0,
		VelocityConsistencyThreshold:  0.85,
		ClickIntervalMin:              50,  // 人类通常不会小于50ms
		ClickIntervalMax:              5000, // 超过5秒可能有问题
		DoubleClickThreshold:          500,  // 双击通常在500ms内
		PathCurvatureThreshold:        0.98,
		DirectionChangeThreshold:      20,
		PathComplexityThreshold:       0.3,
		HesitationThreshold:           2000, // 犹豫超过2秒
		RiskScoreThresholds: RiskScoreThresholds{
			Low:      0,
			Medium:   25,
			High:     50,
			Critical: 80,
		},
	}
}

type AIConfig struct {
	Enable            bool     `yaml:"enable"`
	ModelPath         string   `yaml:"model_path"`
	HiddenDims        []int    `yaml:"hidden_dims"`
	LearningRate      float64  `yaml:"learning_rate"`
	WeightDecay       float64  `yaml:"weight_decay"`
	BatchSize         int      `yaml:"batch_size"`
	TrainingEpochs    int      `yaml:"training_epochs"`
	ValidationSplit   float64  `yaml:"validation_split"`
	DropoutRate       float64  `yaml:"dropout_rate"`
}

func DefaultAIConfig() *AIConfig {
	return &AIConfig{
		Enable:           true,
		ModelPath:        "models/risk_model.json",
		HiddenDims:      []int{64, 32, 16},
		LearningRate:     0.001,
		WeightDecay:      0.0001,
		BatchSize:        32,
		TrainingEpochs:   100,
		ValidationSplit:  0.2,
		DropoutRate:     0.0,
	}
}
