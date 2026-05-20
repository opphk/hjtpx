package service

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type SpatioTemporalFlow struct {
	UserID         string                 `json:"user_id"`
	SessionID      string                 `json:"session_id"`
	FlowID         string                 `json:"flow_id"`
	DataPoints     []FlowDataPoint        `json:"data_points"`
	StartTime      time.Time              `json:"start_time"`
	EndTime        time.Time              `json:"end_time"`
	TotalDuration  time.Duration          `json:"total_duration"`
	Trajectory     *TrajectoryPrediction  `json:"trajectory,omitempty"`
	AnomalyScore   float64                `json:"anomaly_score"`
	IsAnomalous    bool                   `json:"is_anomalous"`
	RiskLevel      string                 `json:"risk_level"`
	Features       map[string]float64     `json:"features"`
	Clusters       []int                  `json:"clusters"`
	Phase          string                 `json:"phase"`
	Continuity     float64                `json:"continuity"`
}

type FlowDataPoint struct {
	X           float64   `json:"x"`
	Y           float64   `json:"y"`
	Z           float64   `json:"z,omitempty"`
	Timestamp   time.Time `json:"timestamp"`
	Velocity    float64   `json:"velocity"`
	Acceleration float64  `json:"acceleration"`
	EventType   string    `json:"event_type"`
	ScreenPos   Position  `json:"screen_pos,omitempty"`
	DeviceOrientation Orientation `json:"orientation,omitempty"`
}

type Position struct {
	Latitude  float64 `json:"latitude,omitempty"`
	Longitude float64 `json:"longitude,omitempty"`
	Altitude  float64 `json:"altitude,omitempty"`
	Accuracy  float64 `json:"accuracy,omitempty"`
}

type Orientation struct {
	Alpha float64 `json:"alpha"`
	Beta  float64 `json:"beta"`
	Gamma float64 `json:"gamma"`
}

type TrajectoryPrediction struct {
	PredictedPoints []FlowDataPoint  `json:"predicted_points"`
	Confidence      float64          `json:"confidence"`
	PredictionHorizon time.Duration   `json:"prediction_horizon"`
	ModelType       string            `json:"model_type"`
	Accuracy        float64           `json:"accuracy"`
}

type AnomalyPattern struct {
	Type        string    `json:"type"`
	Severity    string    `json:"severity"`
	Description string    `json:"description"`
	Timestamp   time.Time `json:"timestamp"`
	Location    Position  `json:"location,omitempty"`
	Score       float64   `json:"score"`
}

type BehaviorFlowService struct {
	flowHistory     map[string][]*SpatioTemporalFlow
	anomalyPatterns []AnomalyPattern
	trajectoryCache map[string]*TrajectoryPrediction
	mu              sync.RWMutex
}

func NewBehaviorFlowService() *BehaviorFlowService {
	return &BehaviorFlowService{
		flowHistory:     make(map[string][]*SpatioTemporalFlow),
		anomalyPatterns: make([]AnomalyPattern, 0),
		trajectoryCache: make(map[string]*TrajectoryPrediction),
	}
}

func (s *BehaviorFlowService) AnalyzeFlow(ctx context.Context, userID string, sessionID string, dataPoints []FlowDataPoint) (*SpatioTemporalFlow, error) {
	if len(dataPoints) == 0 {
		return nil, fmt.Errorf("数据点为空")
	}

	flow := &SpatioTemporalFlow{
		UserID:        userID,
		SessionID:     sessionID,
		FlowID:       fmt.Sprintf("%s_%s_%d", userID, sessionID, time.Now().Unix()),
		DataPoints:    dataPoints,
		StartTime:    dataPoints[0].Timestamp,
		EndTime:      dataPoints[len(dataPoints)-1].Timestamp,
		TotalDuration: dataPoints[len(dataPoints)-1].Timestamp.Sub(dataPoints[0].Timestamp),
		Features:     make(map[string]float64),
	}

	s.computeTemporalFeatures(flow)
	s.computeSpatialFeatures(flow)
	s.computeBehavioralFeatures(flow)
	s.identifyPhase(flow)
	s.detectAnomalies(flow)
	s.computeContinuityScore(flow)
	s.clusterFlowPoints(flow)
	s.predictTrajectory(flow)

	s.mu.Lock()
	s.flowHistory[userID] = append(s.flowHistory[userID], flow)
	if len(s.flowHistory[userID]) > 100 {
		s.flowHistory[userID] = s.flowHistory[userID][len(s.flowHistory[userID])-100:]
	}
	s.mu.Unlock()

	return flow, nil
}

func (s *BehaviorFlowService) computeTemporalFeatures(flow *SpatioTemporalFlow) {
	if len(flow.DataPoints) < 2 {
		return
	}

	intervals := []float64{}
	for i := 1; i < len(flow.DataPoints); i++ {
		interval := flow.DataPoints[i].Timestamp.Sub(flow.DataPoints[i-1].Timestamp).Seconds()
		intervals = append(intervals, interval)
	}

	if len(intervals) > 0 {
		flow.Features["avg_interval"] = mean(intervals)
		flow.Features["max_interval"] = max(intervals)
		flow.Features["min_interval"] = min(intervals)
		flow.Features["interval_variance"] = variance(intervals)
		flow.Features["interval_entropy"] = entropy(intervals)
	}

	flow.Features["total_duration"] = flow.TotalDuration.Seconds()
	flow.Features["data_point_count"] = float64(len(flow.DataPoints))
	flow.Features["data_density"] = float64(len(flow.DataPoints)) / flow.TotalDuration.Seconds()
}

func (s *BehaviorFlowService) computeSpatialFeatures(flow *SpatioTemporalFlow) {
	if len(flow.DataPoints) == 0 {
		return
	}

	distances := []float64{}
	totalDistance := 0.0

	for i := 1; i < len(flow.DataPoints); i++ {
		dx := flow.DataPoints[i].X - flow.DataPoints[i-1].X
		dy := flow.DataPoints[i].Y - flow.DataPoints[i-1].Y
		dz := flow.DataPoints[i].Z - flow.DataPoints[i-1].Z

		dist := math.Sqrt(dx*dx + dy*dy + dz*dz)
		distances = append(distances, dist)
		totalDistance += dist
	}

	flow.Features["total_distance"] = totalDistance
	flow.Features["avg_distance"] = mean(distances)
	flow.Features["max_distance"] = max(distances)
	flow.Features["path_complexity"] = s.computePathComplexity(flow.DataPoints)
	flow.Features["spatial_entropy"] = s.computeSpatialEntropy(flow.DataPoints)

	firstPoint := flow.DataPoints[0]
	lastPoint := flow.DataPoints[len(flow.DataPoints)-1]
	directDistance := math.Sqrt(
		math.Pow(lastPoint.X-firstPoint.X, 2) +
			math.Pow(lastPoint.Y-firstPoint.Y, 2) +
			math.Pow(lastPoint.Z-firstPoint.Z, 2),
	)

	if totalDistance > 0 {
		flow.Features["efficiency"] = directDistance / totalDistance
	}

	flow.Features["area_covered"] = s.computeAreaCovered(flow.DataPoints)
	flow.Features["center_of_mass_x"] = s.computeCenterOfMass(flow.DataPoints, "x")
	flow.Features["center_of_mass_y"] = s.computeCenterOfMass(flow.DataPoints, "y")
}

func (s *BehaviorFlowService) computeBehavioralFeatures(flow *SpatioTemporalFlow) {
	velocities := []float64{}
	accelerations := []float64{}

	for i := 1; i < len(flow.DataPoints); i++ {
		velocities = append(velocities, flow.DataPoints[i].Velocity)

		if i > 1 {
			dt := flow.DataPoints[i].Timestamp.Sub(flow.DataPoints[i-1].Timestamp).Seconds()
			if dt > 0 {
				accel := (flow.DataPoints[i].Velocity - flow.DataPoints[i-1].Velocity) / dt
				accelerations = append(accelerations, math.Abs(accel))
			}
		}
	}

	if len(velocities) > 0 {
		flow.Features["avg_velocity"] = mean(velocities)
		flow.Features["max_velocity"] = max(velocities)
		flow.Features["min_velocity"] = min(velocities)
		flow.Features["velocity_variance"] = variance(velocities)
		flow.Features["velocity_skewness"] = skewness(velocities)
	}

	if len(accelerations) > 0 {
		flow.Features["avg_acceleration"] = mean(accelerations)
		flow.Features["max_acceleration"] = max(accelerations)
		flow.Features["acceleration_variance"] = variance(accelerations)
	}

	eventTypes := make(map[string]int)
	for _, dp := range flow.DataPoints {
		eventTypes[dp.EventType]++
	}
	flow.Features["event_diversity"] = float64(len(eventTypes))
	flow.Features["most_common_event"] = float64(eventTypes[mostFrequent(eventTypes)])
}

func (s *BehaviorFlowService) identifyPhase(flow *SpatioTemporalFlow) {
	duration := flow.TotalDuration.Seconds()

	if duration < 5 {
		flow.Phase = "initial"
	} else if duration < 30 {
		flow.Phase = "exploration"
	} else if duration < 120 {
		flow.Phase = "interaction"
	} else {
		flow.Phase = "extended"
	}

	if flow.Features["velocity_variance"] > 0.5 {
		flow.Phase += "_dynamic"
	} else {
		flow.Phase += "_stable"
	}

	if flow.Features["anomaly_score"] > 0.7 {
		flow.Phase += "_anomalous"
	}
}

func (s *BehaviorFlowService) detectAnomalies(flow *SpatioTemporalFlow) {
	anomalyScore := 0.0
	patterns := []AnomalyPattern{}

	if flow.Features["velocity_variance"] > 0.8 {
		anomalyScore += 0.3
		patterns = append(patterns, AnomalyPattern{
			Type:        "velocity_spike",
			Severity:    "medium",
			Description: "速度变化异常剧烈",
			Timestamp:   time.Now(),
			Score:       0.3,
		})
	}

	if flow.Features["interval_variance"] > 0.9 {
		anomalyScore += 0.25
		patterns = append(patterns, AnomalyPattern{
			Type:        "temporal_gap",
			Severity:    "medium",
			Description: "时间间隔不规律",
			Timestamp:   time.Now(),
			Score:       0.25,
		})
	}

	if flow.Features["efficiency"] > 0.95 {
		anomalyScore += 0.35
		patterns = append(patterns, AnomalyPattern{
			Type:        "unnatural_path",
			Severity:    "high",
			Description: "路径过于笔直，可能为机器行为",
			Timestamp:   time.Now(),
			Score:       0.35,
		})
	}

	if flow.Features["avg_velocity"] > 1000 {
		anomalyScore += 0.4
		patterns = append(patterns, AnomalyPattern{
			Type:        "extreme_speed",
			Severity:    "high",
			Description: "移动速度异常快",
			Timestamp:   time.Now(),
			Score:       0.4,
		})
	}

	if flow.Features["acceleration_variance"] < 0.01 {
		anomalyScore += 0.3
		patterns = append(patterns, AnomalyPattern{
			Type:        "mechanical_motion",
			Severity:    "medium",
			Description: "运动模式过于机械",
			Timestamp:   time.Now(),
			Score:       0.3,
		})
	}

	if flow.Features["spatial_entropy"] < 0.3 {
		anomalyScore += 0.2
		patterns = append(patterns, AnomalyPattern{
			Type:        "localized_movement",
			Severity:    "low",
			Description: "移动范围受限",
			Timestamp:   time.Now(),
			Score:       0.2,
		})
	}

	if len(flow.DataPoints) < 10 {
		anomalyScore += 0.15
		patterns = append(patterns, AnomalyPattern{
			Type:        "insufficient_data",
			Severity:    "low",
			Description: "行为数据点过少",
			Timestamp:   time.Now(),
			Score:       0.15,
		})
	}

	flow.AnomalyScore = math.Min(anomalyScore, 1.0)
	flow.IsAnomalous = flow.AnomalyScore > 0.6

	if flow.AnomalyScore > 0.8 {
		flow.RiskLevel = "critical"
	} else if flow.AnomalyScore > 0.6 {
		flow.RiskLevel = "high"
	} else if flow.AnomalyScore > 0.4 {
		flow.RiskLevel = "medium"
	} else {
		flow.RiskLevel = "low"
	}

	s.mu.Lock()
	s.anomalyPatterns = append(s.anomalyPatterns, patterns...)
	s.mu.Unlock()
}

func (s *BehaviorFlowService) computeContinuityScore(flow *SpatioTemporalFlow) {
	if len(flow.DataPoints) < 2 {
		flow.Continuity = 1.0
		return
	}

	temporalContinuity := 1.0 - flow.Features["interval_variance"]
	spatialContinuity := 1.0 - math.Abs(1.0-flow.Features["efficiency"])
	velocityContinuity := 1.0 - flow.Features["velocity_variance"]

	flow.Continuity = (temporalContinuity + spatialContinuity + velocityContinuity) / 3.0
}

func (s *BehaviorFlowService) clusterFlowPoints(flow *SpatioTemporalFlow) {
	if len(flow.DataPoints) < 3 {
		flow.Clusters = make([]int, len(flow.DataPoints))
		for i := range flow.Clusters {
			flow.Clusters[i] = 0
		}
		return
	}

	k := 3
	if len(flow.DataPoints) < 10 {
		k = 2
	}

	centroids := s.initializeCentroids(flow.DataPoints, k)

	for iteration := 0; iteration < 10; iteration++ {
		clusters := make([][]int, k)
		for i := range clusters {
			clusters[i] = []int{}
		}

		for i, point := range flow.DataPoints {
			minDist := math.MaxFloat64
			cluster := 0
			for j, centroid := range centroids {
				dist := s.distance(point, centroid)
				if dist < minDist {
					minDist = dist
					cluster = j
				}
			}
			clusters[cluster] = append(clusters[cluster], i)
		}

		for j := 0; j < k; j++ {
			if len(clusters[j]) > 0 {
				var sumX, sumY, sumZ float64
				for _, idx := range clusters[j] {
					sumX += flow.DataPoints[idx].X
					sumY += flow.DataPoints[idx].Y
					sumZ += flow.DataPoints[idx].Z
				}
				centroids[j].X = sumX / float64(len(clusters[j]))
				centroids[j].Y = sumY / float64(len(clusters[j]))
				centroids[j].Z = sumZ / float64(len(clusters[j]))
			}
		}
	}

	flow.Clusters = make([]int, len(flow.DataPoints))
	for i, point := range flow.DataPoints {
		minDist := math.MaxFloat64
		cluster := 0
		for j, centroid := range centroids {
			dist := s.distance(point, centroid)
			if dist < minDist {
				minDist = dist
				cluster = j
			}
		}
		flow.Clusters[i] = cluster
	}
}

func (s *BehaviorFlowService) initializeCentroids(points []FlowDataPoint, k int) []FlowDataPoint {
	centroids := make([]FlowDataPoint, k)
	centroids[0] = points[0]
	centroids[1] = points[len(points)/2]
	centroids[2] = points[len(points)-1]

	return centroids
}

func (s *BehaviorFlowService) distance(p1, p2 FlowDataPoint) float64 {
	return math.Sqrt(
		math.Pow(p1.X-p2.X, 2) +
			math.Pow(p1.Y-p2.Y, 2) +
			math.Pow(p1.Z-p2.Z, 2),
	)
}

func (s *BehaviorFlowService) predictTrajectory(flow *SpatioTemporalFlow) {
	if len(flow.DataPoints) < 5 {
		return
	}

	predictionHorizon := time.Duration(len(flow.DataPoints)/2) * time.Second
	predictedPoints := []FlowDataPoint{}

	lastPoints := flow.DataPoints[len(flow.DataPoints)-5:]
	
	avgVelocity := mean([]float64{lastPoints[len(lastPoints)-1].Velocity, lastPoints[len(lastPoints)-2].Velocity})
	avgAcceleration := 0.0
	if len(flow.DataPoints) > 1 {
		accels := []float64{}
		for i := 2; i < len(flow.DataPoints); i++ {
			dt := flow.DataPoints[i].Timestamp.Sub(flow.DataPoints[i-1].Timestamp).Seconds()
			if dt > 0 {
				accel := (flow.DataPoints[i].Velocity - flow.DataPoints[i-1].Velocity) / dt
				accels = append(accels, accel)
			}
		}
		avgAcceleration = mean(accels)
	}

	dx := lastPoints[len(lastPoints)-1].X - lastPoints[0].X
	dy := lastPoints[len(lastPoints)-1].Y - lastPoints[0].Y
	dz := lastPoints[len(lastPoints)-1].Z - lastPoints[0].Z

	predictionSteps := 5
	for i := 1; i <= predictionSteps; i++ {
		t := float64(i) * predictionHorizon.Seconds() / float64(predictionSteps)
		predictedPoint := FlowDataPoint{
			Timestamp:    lastPoints[len(lastPoints)-1].Timestamp.Add(time.Duration(t*float64(time.Second))),
			Velocity:     avgVelocity + avgAcceleration*t,
			Acceleration: avgAcceleration,
		}
		predictedPoint.X = lastPoints[len(lastPoints)-1].X + dx*t/float64(len(lastPoints)-1)
		predictedPoint.Y = lastPoints[len(lastPoints)-1].Y + dy*t/float64(len(lastPoints)-1)
		predictedPoint.Z = lastPoints[len(lastPoints)-1].Z + dz*t/float64(len(lastPoints)-1)

		predictedPoints = append(predictedPoints, predictedPoint)
	}

	flow.Trajectory = &TrajectoryPrediction{
		PredictedPoints:   predictedPoints,
		Confidence:        s.computePredictionConfidence(flow),
		PredictionHorizon: predictionHorizon,
		ModelType:        "linear_regression",
		Accuracy:         flow.Continuity,
	}

	s.mu.Lock()
	s.trajectoryCache[flow.FlowID] = flow.Trajectory
	s.mu.Unlock()
}

func (s *BehaviorFlowService) computePredictionConfidence(flow *SpatioTemporalFlow) float64 {
	confidence := 1.0

	if flow.Features["velocity_variance"] > 0.5 {
		confidence -= 0.2
	}

	if flow.Features["interval_variance"] > 0.5 {
		confidence -= 0.15
	}

	if len(flow.DataPoints) < 10 {
		confidence -= 0.25
	}

	return math.Max(0.0, math.Min(1.0, confidence))
}

func (s *BehaviorFlowService) computePathComplexity(points []FlowDataPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	totalAngleChange := 0.0
	angleCount := 0

	for i := 1; i < len(points)-1; i++ {
		v1x := points[i].X - points[i-1].X
		v1y := points[i].Y - points[i-1].Y
		v2x := points[i+1].X - points[i].X
		v2y := points[i+1].Y - points[i].Y

		dot := v1x*v2x + v1y*v2y
		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			totalAngleChange += angle
			angleCount++
		}
	}

	if angleCount > 0 {
		return totalAngleChange / float64(angleCount)
	}
	return 0.0
}

func (s *BehaviorFlowService) computeSpatialEntropy(points []FlowDataPoint) float64 {
	if len(points) == 0 {
		return 0.0
	}

	xValues := make([]float64, len(points))
	yValues := make([]float64, len(points))
	for i, p := range points {
		xValues[i] = p.X
		yValues[i] = p.Y
	}

	entropyX := entropy(xValues)
	entropyY := entropy(yValues)

	return (entropyX + entropyY) / 2.0
}

func (s *BehaviorFlowService) computeAreaCovered(points []FlowDataPoint) float64 {
	if len(points) < 3 {
		return 0.0
	}

	minX, maxX := points[0].X, points[0].X
	minY, maxY := points[0].Y, points[0].Y

	for _, p := range points {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}

	return (maxX - minX) * (maxY - minY)
}

func (s *BehaviorFlowService) computeCenterOfMass(points []FlowDataPoint, axis string) float64 {
	if len(points) == 0 {
		return 0.0
	}

	sum := 0.0
	for _, p := range points {
		switch axis {
		case "x":
			sum += p.X
		case "y":
			sum += p.Y
		case "z":
			sum += p.Z
		}
	}

	return sum / float64(len(points))
}

func (s *BehaviorFlowService) GetFlowHistory(ctx context.Context, userID string) ([]*SpatioTemporalFlow, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	history, ok := s.flowHistory[userID]
	if !ok {
		return []*SpatioTemporalFlow{}, nil
	}

	return history, nil
}

func (s *BehaviorFlowService) GetAnomalyPatterns(ctx context.Context, limit int) ([]AnomalyPattern, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	if limit <= 0 || limit > len(s.anomalyPatterns) {
		limit = len(s.anomalyPatterns)
	}

	return s.anomalyPatterns[len(s.anomalyPatterns)-limit:], nil
}

func (s *BehaviorFlowService) CompareFlows(ctx context.Context, flow1ID, flow2ID string) (float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	var flow1, flow2 *SpatioTemporalFlow

	for _, flows := range s.flowHistory {
		for _, f := range flows {
			if f.FlowID == flow1ID {
				flow1 = f
			}
			if f.FlowID == flow2ID {
				flow2 = f
			}
		}
	}

	if flow1 == nil || flow2 == nil {
		return 0.0, fmt.Errorf("flow not found")
	}

	similarity := 0.0
	featureCount := 0

	for key, val1 := range flow1.Features {
		if val2, ok := flow2.Features[key]; ok {
			diff := math.Abs(val1 - val2)
			maxVal := math.Max(math.Abs(val1), math.Abs(val2))
			if maxVal > 0 {
				similarity += 1.0 - diff/maxVal
			}
			featureCount++
		}
	}

	if featureCount > 0 {
		return similarity / float64(featureCount), nil
	}

	return 0.0, nil
}

func (s *BehaviorFlowService) GenerateFlowReport(flow *SpatioTemporalFlow) string {
	report := fmt.Sprintf("时空行为流分析报告\n")
	report += fmt.Sprintf("==================\n")
	report += fmt.Sprintf("流ID: %s\n", flow.FlowID)
	report += fmt.Sprintf("用户ID: %s\n", flow.UserID)
	report += fmt.Sprintf("会话ID: %s\n", flow.SessionID)
	report += fmt.Sprintf("阶段: %s\n", flow.Phase)
	report += fmt.Sprintf("风险等级: %s\n", flow.RiskLevel)
	report += fmt.Sprintf("异常评分: %.2f\n", flow.AnomalyScore)
	report += fmt.Sprintf("连续性评分: %.2f\n", flow.Continuity)
	report += fmt.Sprintf("\n特征分析:\n")
	report += fmt.Sprintf("- 总距离: %.2f\n", flow.Features["total_distance"])
	report += fmt.Sprintf("- 平均速度: %.2f\n", flow.Features["avg_velocity"])
	report += fmt.Sprintf("- 最大速度: %.2f\n", flow.Features["max_velocity"])
	report += fmt.Sprintf("- 路径效率: %.2f\n", flow.Features["efficiency"])
	report += fmt.Sprintf("- 路径复杂度: %.2f\n", flow.Features["path_complexity"])
	report += fmt.Sprintf("- 空间熵: %.2f\n", flow.Features["spatial_entropy"])
	report += fmt.Sprintf("- 数据点数量: %.0f\n", flow.Features["data_point_count"])
	report += fmt.Sprintf("- 总时长: %.2f秒\n", flow.Features["total_duration"])

	if len(flow.DataPoints) > 0 {
		report += fmt.Sprintf("\n轨迹预测:\n")
		if flow.Trajectory != nil {
			report += fmt.Sprintf("- 预测模型: %s\n", flow.Trajectory.ModelType)
			report += fmt.Sprintf("- 预测置信度: %.2f\n", flow.Trajectory.Confidence)
			report += fmt.Sprintf("- 预测精度: %.2f\n", flow.Trajectory.Accuracy)
		}
	}

	return report
}

func mean(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func max(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	maxVal := values[0]
	for _, v := range values {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func min(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}
	minVal := values[0]
	for _, v := range values {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func variance(values []float64) float64 {
	if len(values) < 2 {
		return 0.0
	}
	meanVal := mean(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-meanVal, 2)
	}
	return sum / float64(len(values)-1)
}

func skewness(values []float64) float64 {
	if len(values) < 3 {
		return 0.0
	}
	meanVal := mean(values)
	stdDev := math.Sqrt(variance(values))
	if stdDev == 0 {
		return 0.0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-meanVal)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func entropy(values []float64) float64 {
	if len(values) == 0 {
		return 0.0
	}

	minVal := min(values)
	maxVal := max(values)

	if maxVal-minVal < 0.001 {
		return 0.0
	}

	buckets := 10
	bucketSize := (maxVal - minVal) / float64(buckets)
	counts := make([]int, buckets)

	for _, v := range values {
		bucket := int((v - minVal) / bucketSize)
		if bucket >= buckets {
			bucket = buckets - 1
		}
		if bucket < 0 {
			bucket = 0
		}
		counts[bucket]++
	}

	total := len(values)
	h := 0.0
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / float64(total)
			h -= p * math.Log2(p)
		}
	}

	return h
}

func mostFrequent(counter map[string]int) string {
	maxCount := 0
	result := ""
	for key, count := range counter {
		if count > maxCount {
			maxCount = count
			result = key
		}
	}
	return result
}

func AnalyzeBehaviorFlowFromData(ctx context.Context, behaviorData []models.BehaviorData, userID string) (*SpatioTemporalFlow, error) {
	flowService := NewBehaviorFlowService()

	dataPoints := make([]FlowDataPoint, 0, len(behaviorData))

	for _, bd := range behaviorData {
		var dp FlowDataPoint
		if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
			dp.Timestamp = bd.Timestamp
			dataPoints = append(dataPoints, dp)
		}
	}

	sort.Slice(dataPoints, func(i, j int) bool {
		return dataPoints[i].Timestamp.Before(dataPoints[j].Timestamp)
	})

	flow, err := flowService.AnalyzeFlow(ctx, userID, "analysis", dataPoints)
	if err != nil {
		return nil, err
	}

	return flow, nil
}
