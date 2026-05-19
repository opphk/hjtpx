package model

import (
	"math"
	"time"
)

type MouseBehaviorData struct {
	SessionID    string           `json:"session_id"`
	UserID       string           `json:"user_id"`
	MousePoints  []MousePoint     `json:"mouse_points"`
	ClickPoints  []ClickPoint     `json:"click_points"`
	ScrollEvents []ScrollEvent    `json:"scroll_events"`
	StartTime    time.Time        `json:"start_time"`
	EndTime      time.Time        `json:"end_time"`
}

type MousePoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	Button    int   `json:"button"`
}

type ClickPoint struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	Button    int   `json:"button"`
	ClickType string `json:"click_type"`
}

type ScrollEvent struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	DeltaX    int   `json:"delta_x"`
	DeltaY    int   `json:"delta_y"`
}

type MouseSpeedFeature struct {
	AverageSpeed     float64 `json:"average_speed"`
	MedianSpeed      float64 `json:"median_speed"`
	MaxSpeed         float64 `json:"max_speed"`
	MinSpeed         float64 `json:"min_speed"`
	SpeedVariance    float64 `json:"speed_variance"`
	SpeedStdDev      float64 `json:"speed_std_dev"`
	SpeedSkewness    float64 `json:"speed_skewness"`
	SpeedKurtosis    float64 `json:"speed_kurtosis"`
	SpeedRange       float64 `json:"speed_range"`
	ZeroSpeedCount   int     `json:"zero_speed_count"`
	LowSpeedCount    int     `json:"low_speed_count"`
	HighSpeedCount   int     `json:"high_speed_count"`
	SpeedOutliers    []int   `json:"speed_outliers"`
}

type MouseAccelerationFeature struct {
	AverageAcceleration float64   `json:"average_acceleration"`
	MaxAcceleration     float64   `json:"max_acceleration"`
	MinAcceleration     float64   `json:"min_acceleration"`
	AccelerationVariance float64  `json:"acceleration_variance"`
	AccelerationStdDev  float64   `json:"acceleration_std_dev"`
	JerkAvg            float64   `json:"jerk_avg"`
	JerkMax            float64   `json:"jerk_max"`
	JerkMin            float64   `json:"jerk_min"`
	JerkVariance       float64   `json:"jerk_variance"`
	PositiveAccelRatio float64   `json:"positive_accel_ratio"`
	NegativeAccelRatio float64   `json:"negative_accel_ratio"`
	AccelerationPeaks  []float64 `json:"acceleration_peaks"`
	AccelerationValleys []float64 `json:"acceleration_valleys"`
	DirectionChanges   int       `json:"direction_changes"`
}

type ClickDistributionFeature struct {
	XMean     float64 `json:"x_mean"`
	YMean     float64 `json:"y_mean"`
	XVariance float64 `json:"x_variance"`
	YVariance float64 `json:"y_variance"`
	XStdDev   float64 `json:"x_std_dev"`
	YStdDev   float64 `json:"y_std_dev"`
	XEntropy  float64 `json:"x_entropy"`
	YEntropy  float64 `json:"y_entropy"`
	XSkewness float64 `json:"x_skewness"`
	YSkewness float64 `json:"y_skewness"`
	XKurtosis float64 `json:"x_kurtosis"`
	YKurtosis float64 `json:"y_kurtosis"`
	SpreadX   float64 `json:"spread_x"`
	SpreadY   float64 `json:"spread_y"`
	CenterX   float64 `json:"center_x"`
	CenterY   float64 `json:"center_y"`
	Density   float64 `json:"density"`
	Clusters  int     `json:"clusters"`
}

type DoubleClickFeature struct {
	DoubleClickCount     int       `json:"double_click_count"`
	SingleClickCount     int       `json:"single_click_count"`
	TripleClickCount     int       `json:"triple_click_count"`
	AverageInterval      float64   `json:"average_interval"`
	MinInterval          float64   `json:"min_interval"`
	MaxInterval          float64   `json:"max_interval"`
	IntervalVariance     float64   `json:"interval_variance"`
	IntervalStdDev       float64   `json:"interval_std_dev"`
	FastDoubleClickRatio float64   `json:"fast_double_click_ratio"`
	NormalDoubleClickRatio float64 `json:"normal_double_click_ratio"`
	DoubleClickPositions [][]int   `json:"double_click_positions"`
	ClickBurstCount      int       `json:"click_burst_count"`
}

type ClickLatencyFeature struct {
	AverageLatency      float64   `json:"average_latency"`
	MedianLatency       float64   `json:"median_latency"`
	MinLatency          float64   `json:"min_latency"`
	MaxLatency          float64   `json:"max_latency"`
	LatencyVariance     float64   `json:"latency_variance"`
	LatencyStdDev       float64   `json:"latency_std_dev"`
	FastClickRatio      float64   `json:"fast_click_ratio"`
	SlowClickRatio      float64   `json:"slow_click_ratio"`
	FirstClickDelay     float64   `json:"first_click_delay"`
	LastClickDelay      float64   `json:"last_click_delay"`
	HesitationCount     int       `json:"hesitation_count"`
	HesitationRatio     float64   `json:"hesitation_ratio"`
	ReactionTimeTrend   float64   `json:"reaction_time_trend"`
	LatencyOutliers     []int     `json:"latency_outliers"`
}

type MouseBehaviorFeatures struct {
	SpeedFeatures        MouseSpeedFeature        `json:"speed_features"`
	AccelerationFeatures MouseAccelerationFeature `json:"acceleration_features"`
	DistributionFeatures ClickDistributionFeature `json:"distribution_features"`
	DoubleClickFeatures  DoubleClickFeature       `json:"double_click_features"`
	LatencyFeatures      ClickLatencyFeature       `json:"latency_features"`
	OverallScore         float64                  `json:"overall_score"`
	IsHumanLike          bool                     `json:"is_human_like"`
	Confidence           float64                  `json:"confidence"`
	AnomalyIndicators    []string                 `json:"anomaly_indicators"`
}

func NewMouseBehaviorData() *MouseBehaviorData {
	return &MouseBehaviorData{
		MousePoints:  make([]MousePoint, 0),
		ClickPoints:  make([]ClickPoint, 0),
		ScrollEvents: make([]ScrollEvent, 0),
	}
}

func (f *MouseSpeedFeature) CalculateSpeedRange() float64 {
	return f.MaxSpeed - f.MinSpeed
}

func (f *MouseSpeedFeature) CalculateSpeedConsistency() float64 {
	if f.AverageSpeed <= 0 {
		return 0
	}
	return 1.0 - math.Min(f.SpeedStdDev/f.AverageSpeed, 1.0)
}

func (f *MouseAccelerationFeature) CalculateAccelerationConsistency() float64 {
	if math.Abs(f.AverageAcceleration) < 0.001 {
		return 1.0
	}
	return 1.0 - math.Min(math.Abs(f.AccelerationStdDev/f.AverageAcceleration), 1.0)
}

func (f *ClickDistributionFeature) CalculateDistanceFromCenter(x, y int) float64 {
	dx := float64(x) - f.CenterX
	dy := float64(y) - f.CenterY
	return math.Sqrt(dx*dx + dy*dy)
}

func (f *ClickDistributionFeature) IsWithinOneStdDev(x, y int) bool {
	dx := float64(x) - f.XMean
	dy := float64(y) - f.YMean
	return (dx*dx)/(f.XStdDev*f.XStdDev)+(dy*dy)/(f.YStdDev*f.YStdDev) <= 1.0
}

func (f *DoubleClickFeature) CalculateFastClickRatio() float64 {
	total := f.DoubleClickCount + f.SingleClickCount + f.TripleClickCount
	if total == 0 {
		return 0
	}
	return float64(f.FastDoubleClickRatio) / float64(total)
}

func (f *ClickLatencyFeature) CalculateHesitationRate() float64 {
	if f.AverageLatency <= 0 {
		return 0
	}
	return float64(f.HesitationCount) / f.AverageLatency
}

func (f *MouseBehaviorFeatures) AddAnomalyIndicator(indicator string) {
	for _, existing := range f.AnomalyIndicators {
		if existing == indicator {
			return
		}
	}
	f.AnomalyIndicators = append(f.AnomalyIndicators, indicator)
}

func (f *MouseBehaviorFeatures) CalculateOverallScore() float64 {
	score := 0.0
	
	speedConsistency := f.SpeedFeatures.CalculateSpeedConsistency()
	if speedConsistency > 0.95 {
		score += 20
		f.AddAnomalyIndicator("速度过于恒定")
	}
	
	accelConsistency := f.AccelerationFeatures.CalculateAccelerationConsistency()
	if accelConsistency > 0.98 {
		score += 15
		f.AddAnomalyIndicator("加速度过于恒定")
	}
	
	if f.DoubleClickFeatures.FastDoubleClickRatio > 0.8 {
		score += 10
		f.AddAnomalyIndicator("双击速度异常快")
	}
	
	if f.LatencyFeatures.FastClickRatio > 0.9 {
		score += 15
		f.AddAnomalyIndicator("反应时间异常快")
	}
	
	if f.LatencyFeatures.HesitationCount == 0 && len(f.LatencyFeatures.LatencyOutliers) > 3 {
		score += 10
		f.AddAnomalyIndicator("无犹豫时间")
	}
	
	f.OverallScore = math.Min(score, 100)
	f.IsHumanLike = score < 30
	f.Confidence = math.Min(score/100+0.5, 0.99)
	
	return f.OverallScore
}
