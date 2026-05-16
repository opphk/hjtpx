package service

import (
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type PressureDataPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Pressure  float64 `json:"pressure"`
	Timestamp int64   `json:"timestamp"`
}

type AdvancedTrajectoryFeatures struct {
	Points                   []BehaviorDataPoint
	TotalDistance            float64
	AverageSpeed             float64
	MaxSpeed                 float64
	MinSpeed                 float64
	PathEfficiency           float64
	DirectionChanges         int
	CurvatureAvg             float64
	JitterScore              float64
	PauseCount               int
	MicroCorrections         int
	SpeedVariance            float64
	AccelerationAvg          float64
	AccelerationMagVariance  float64
	PressureAvg              float64
	PressureVariance         float64
	PressureTrend            float64
	TiltXAvg                 float64
	TiltYAvg                 float64
	TouchAreaAvg             float64
	SwipeVelocityVariance    float64
	VelocityEntropy          float64
	AngularVelocityAvg       float64
	TrajectoryComplexity      float64
	SegmentLengthVariance    float64
	BezierCurveFitError      float64
	StraightnessDeviation    float64
	HumanLikelihood          float64
}

type PressureFeatures struct {
	Points              []PressureDataPoint
	AveragePressure     float64
	PressureVariance    float64
	MaxPressure         float64
	MinPressure         float64
	PressureDistribution []float64
	IsPressureConsistent bool
	PressureAnomalies   int
}

type SwipeJitterFeatures struct {
	JitterMagnitude    float64
	JitterFrequency    float64
	JitterDirection    float64
	JitterConsistency  float64
	IsHumanLike        bool
}

type NeuralNetwork struct {
	InputSize   int
	HiddenSize  int
	OutputSize  int
	WeightsIH   [][]float64
	WeightsHO   [][]float64
	BiasH       []float64
	BiasO       []float64
	LearningRate float64
}

func NewNeuralNetwork(inputSize, hiddenSize, outputSize int) *NeuralNetwork {
	rand.Seed(time.Now().UnixNano())
	nn := &NeuralNetwork{
		InputSize:   inputSize,
		HiddenSize:  hiddenSize,
		OutputSize:  outputSize,
		LearningRate: 0.1,
	}
	nn.WeightsIH = nn.initializeWeights(inputSize, hiddenSize)
	nn.WeightsHO = nn.initializeWeights(hiddenSize, outputSize)
	nn.BiasH = make([]float64, hiddenSize)
	nn.BiasO = make([]float64, outputSize)
	return nn
}

func (nn *NeuralNetwork) initializeWeights(rows, cols int) [][]float64 {
	weights := make([][]float64, rows)
	for i := range weights {
		weights[i] = make([]float64, cols)
		for j := range weights[i] {
			weights[i][j] = (rand.Float64()*2 - 1) * math.Sqrt(2.0/float64(rows))
		}
	}
	return weights
}

func (nn *NeuralNetwork) sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-math.Max(-500, math.Min(500, x))))
}

func (nn *NeuralNetwork) sigmoidDerivative(x float64) float64 {
	return x * (1 - x)
}

func (nn *NeuralNetwork) relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func (nn *NeuralNetwork) reluDerivative(x float64) float64 {
	if x > 0 {
		return 1
	}
	return 0
}

func (nn *NeuralNetwork) Forward(input []float64) ([]float64, []float64) {
	hiddenLayer := make([]float64, nn.HiddenSize)
	for j := 0; j < nn.HiddenSize; j++ {
		sum := nn.BiasH[j]
		for i := 0; i < nn.InputSize; i++ {
			sum += input[i] * nn.WeightsIH[i][j]
		}
		hiddenLayer[j] = nn.sigmoid(sum)
	}

	outputLayer := make([]float64, nn.OutputSize)
	for j := 0; j < nn.OutputSize; j++ {
		sum := nn.BiasO[j]
		for i := 0; i < nn.HiddenSize; i++ {
			sum += hiddenLayer[i] * nn.WeightsHO[i][j]
		}
		outputLayer[j] = nn.sigmoid(sum)
	}

	return outputLayer, hiddenLayer
}

func (nn *NeuralNetwork) Train(input, target []float64) {
	output, hiddenLayer := nn.Forward(input)

	outputErrors := make([]float64, nn.OutputSize)
	for i := 0; i < nn.OutputSize; i++ {
		outputErrors[i] = (target[i] - output[i]) * nn.sigmoidDerivative(output[i])
	}

	hiddenErrors := make([]float64, nn.HiddenSize)
	for i := 0; i < nn.HiddenSize; i++ {
		sum := 0.0
		for j := 0; j < nn.OutputSize; j++ {
			sum += outputErrors[j] * nn.WeightsHO[i][j]
		}
		hiddenErrors[i] = sum * nn.sigmoidDerivative(hiddenLayer[i])
	}

	for i := 0; i < nn.HiddenSize; i++ {
		for j := 0; j < nn.OutputSize; j++ {
			nn.WeightsHO[i][j] += nn.LearningRate * outputErrors[j] * hiddenLayer[i]
		}
	}

	for i := 0; i < nn.OutputSize; i++ {
		nn.BiasO[i] += nn.LearningRate * outputErrors[i]
	}

	for i := 0; i < nn.InputSize; i++ {
		for j := 0; j < nn.HiddenSize; j++ {
			nn.WeightsIH[i][j] += nn.LearningRate * hiddenErrors[j] * input[i]
		}
	}

	for i := 0; i < nn.HiddenSize; i++ {
		nn.BiasH[i] += nn.LearningRate * hiddenErrors[i]
	}
}

func (nn *NeuralNetwork) Predict(input []float64) []float64 {
	output, _ := nn.Forward(input)
	return output
}

type AnomalyDetector struct {
	Threshold      float64
	WindowSize     int
	RecentValues   []float64
	Mean           float64
	StdDev         float64
	IsAnomalous   bool
	AnomalyCount   int
	TotalAnomalies int
}

func NewAnomalyDetector(windowSize int, threshold float64) *AnomalyDetector {
	return &AnomalyDetector{
		Threshold:    threshold,
		WindowSize:   windowSize,
		RecentValues: make([]float64, 0),
		Mean:         0,
		StdDev:       1,
	}
}

func (ad *AnomalyDetector) Update(value float64) bool {
	ad.RecentValues = append(ad.RecentValues, value)
	if len(ad.RecentValues) > ad.WindowSize {
		ad.RecentValues = ad.RecentValues[1:]
	}

	ad.computeStatistics()
	isAnomaly := ad.checkAnomaly(value)
	if isAnomaly {
		ad.AnomalyCount++
		ad.TotalAnomalies++
	}
	ad.IsAnomalous = isAnomaly
	return isAnomaly
}

func (ad *AnomalyDetector) computeStatistics() {
	if len(ad.RecentValues) == 0 {
		return
	}
	sum := 0.0
	for _, v := range ad.RecentValues {
		sum += v
	}
	ad.Mean = sum / float64(len(ad.RecentValues))

	if len(ad.RecentValues) > 1 {
		variance := 0.0
		for _, v := range ad.RecentValues {
			variance += math.Pow(v-ad.Mean, 2)
		}
		ad.StdDev = math.Sqrt(variance / float64(len(ad.RecentValues)))
	}
}

func (ad *AnomalyDetector) checkAnomaly(value float64) bool {
	if ad.StdDev == 0 {
		return false
	}
	zScore := math.Abs(value - ad.Mean) / ad.StdDev
	return zScore > ad.Threshold
}

type AdvancedEnsembleClassifier struct {
	NeuralNet        *NeuralNetwork
	AdvancedRuleEng  *AdvancedRuleEngine
	AnomalyDetectors map[string]*AnomalyDetector
}

func NewAdvancedEnsembleClassifier() *AdvancedEnsembleClassifier {
	ec := &AdvancedEnsembleClassifier{
		NeuralNet:       NewNeuralNetwork(20, 15, 1),
		AdvancedRuleEng: NewAdvancedRuleEngine(),
		AnomalyDetectors: make(map[string]*AnomalyDetector),
	}
	ec.AnomalyDetectors["speed"] = NewAnomalyDetector(50, 3.0)
	ec.AnomalyDetectors["jitter"] = NewAnomalyDetector(30, 2.5)
	ec.AnomalyDetectors["pressure"] = NewAnomalyDetector(40, 2.8)
	ec.AnomalyDetectors["curvature"] = NewAnomalyDetector(35, 3.2)
	return ec
}

type AdvancedRuleEngine struct {
	Rules []AdvancedRule
	Score float64
}

type AdvancedRule struct {
	Name        string
	Condition   func(*AnalysisResult) bool
	Weight      float64
	Description string
}

func NewAdvancedRuleEngine() *AdvancedRuleEngine {
	re := &AdvancedRuleEngine{
		Rules: make([]AdvancedRule, 0),
	}
	re.initializeRules()
	return re
}

func (re *AdvancedRuleEngine) initializeRules() {
	re.Rules = append(re.Rules, AdvancedRule{
		Name: "超高速移动",
		Condition: func(r *AnalysisResult) bool {
			return r.SpeedAnalysis.MaxSpeed > 10
		},
		Weight:      10,
		Description: "检测到超高速移动",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "速度恒定",
		Condition: func(r *AnalysisResult) bool {
			if r.SpeedAnalysis.AverageSpeed == 0 {
				return false
			}
			cv := r.SpeedAnalysis.SpeedStdDev / r.SpeedAnalysis.AverageSpeed
			return cv < 0.1 && len(r.SpeedAnalysis.Speeds) > 5
		},
		Weight:      15,
		Description: "速度过于恒定(机器特征)",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "路径笔直",
		Condition: func(r *AnalysisResult) bool {
			return r.Trajectory.PathEfficiency > 0.92 && r.Trajectory.TotalDistance > 100
		},
		Weight:      25,
		Description: "路径过于笔直",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "低抖动",
		Condition: func(r *AnalysisResult) bool {
			return r.Trajectory.JitterScore < 0.03
		},
		Weight:      20,
		Description: "轨迹抖动过低(机器特征)",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "低曲率",
		Condition: func(r *AnalysisResult) bool {
			return r.Trajectory.CurvatureAvg < 0.05 && len(r.Trajectory.Points) > 20
		},
		Weight:      20,
		Description: "曲率过低(机器特征)",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "无停顿",
		Condition: func(r *AnalysisResult) bool {
			return r.Trajectory.PauseCount == 0 && len(r.Trajectory.Points) >= 20
		},
		Weight:      15,
		Description: "无停顿(机器特征)",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "无微修正",
		Condition: func(r *AnalysisResult) bool {
			return r.Trajectory.MicroCorrections == 0 && len(r.Trajectory.Points) >= 20
		},
		Weight:      15,
		Description: "无微修正(机器特征)",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "路径重复",
		Condition: func(r *AnalysisResult) bool {
			return r.PathSimilarity.IsPathRepeated
		},
		Weight:      30,
		Description: "路径重复检测",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "规律点击",
		Condition: func(r *AnalysisResult) bool {
			return r.ClickPattern.Regularity > 0.9 && r.ClickPattern.ClickCount > 2
		},
		Weight:      15,
		Description: "点击间隔过于规律",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "快速打字",
		Condition: func(r *AnalysisResult) bool {
			return len(r.KeyboardPattern.KeyStrokes) > 0 && r.KeyboardPattern.TypingSpeed > 15
		},
		Weight:      15,
		Description: "打字速度异常快",
	})

	re.Rules = append(re.Rules, AdvancedRule{
		Name: "数据点过少",
		Condition: func(r *AnalysisResult) bool {
			return len(r.Trajectory.Points) < 10
		},
		Weight:      10,
		Description: "行为数据点过少",
	})
}

func (re *AdvancedRuleEngine) Evaluate(result *AnalysisResult) float64 {
	totalScore := 0.0
	for _, rule := range re.Rules {
		if rule.Condition(result) {
			totalScore += rule.Weight
		}
	}
	re.Score = math.Min(totalScore, 100)
	return re.Score
}

func (ec *AdvancedEnsembleClassifier) Classify(input []float64) (bool, float64) {
	nnOutput := ec.NeuralNet.Predict(input)
	nnScore := nnOutput[0] * 100

	features := ec.extractFeaturesFromInput(input)
	ruleScore := ec.AdvancedRuleEng.Evaluate(features)

	ec.checkAnomalies(features)

	nnWeight := 0.4
	ruleWeight := 0.4
	anomalyWeight := 0.2

	anomalyScore := 0.0
	anomalyCount := 0
	for _, detector := range ec.AnomalyDetectors {
		if detector.IsAnomalous {
			anomalyScore += 50
			anomalyCount++
		}
	}
	if anomalyCount > 0 {
		anomalyScore /= float64(anomalyCount)
	}

	finalScore := nnWeight*nnScore + ruleWeight*ruleScore + anomalyWeight*anomalyScore

	isBot := finalScore >= 50

	return isBot, finalScore
}

func (ec *AdvancedEnsembleClassifier) extractFeaturesFromInput(input []float64) *AnalysisResult {
	result := &AnalysisResult{
		Trajectory:      MouseTrajectory{},
		SpeedAnalysis:   SpeedAnalysis{},
		ClickPattern:   ClickPattern{},
		KeyboardPattern: KeyboardPattern{},
	}

	if len(input) >= 20 {
		result.SpeedAnalysis.AverageSpeed = input[0]
		result.SpeedAnalysis.MaxSpeed = input[1]
		result.SpeedAnalysis.MinSpeed = input[2]
		result.SpeedAnalysis.SpeedStdDev = input[3]
		result.Trajectory.PathEfficiency = input[4]
		result.Trajectory.JitterScore = input[5]
		result.Trajectory.CurvatureAvg = input[6]
		result.Trajectory.PauseCount = int(input[7])
		result.Trajectory.MicroCorrections = int(input[8])
		result.Trajectory.TotalDistance = input[9]
		result.Trajectory.SpeedVariance = input[10]
		result.Trajectory.AccelerationAvg = input[11]
		result.ClickPattern.Regularity = input[12]
		result.ClickPattern.PositionEntropy = input[13]
		result.ClickPattern.ClickCount = int(input[14])
		result.KeyboardPattern.TypingSpeed = input[15]
		result.KeyboardPattern.AverageHoldTime = input[16]
		result.KeyboardPattern.Regularity = input[17]
		result.PathSimilarity.SimilarityScore = input[18]
		result.PathSimilarity.IsPathRepeated = input[19] > 0.85
	}

	return result
}

func (ec *AdvancedEnsembleClassifier) checkAnomalies(result *AnalysisResult) {
	ec.AnomalyDetectors["speed"].Update(result.SpeedAnalysis.AverageSpeed)
	ec.AnomalyDetectors["jitter"].Update(result.Trajectory.JitterScore)
	ec.AnomalyDetectors["curvature"].Update(result.Trajectory.CurvatureAvg)
}

type AdvancedBehaviorAnalyzer struct {
	Ensemble     *AdvancedEnsembleClassifier
	PressureData []PressureDataPoint
}

func NewAdvancedBehaviorAnalyzer() *AdvancedBehaviorAnalyzer {
	return &AdvancedBehaviorAnalyzer{
		Ensemble: NewAdvancedEnsembleClassifier(),
	}
}

func (aba *AdvancedBehaviorAnalyzer) AnalyzeAdvancedTrajectory(points []BehaviorDataPoint) *AdvancedTrajectoryFeatures {
	features := &AdvancedTrajectoryFeatures{
		Points: points,
	}

	if len(points) < 2 {
		return features
	}

	smoothedPoints := aba.smoothTrajectory(points, 5)
	features.TotalDistance = aba.calculateTotalDistance(smoothedPoints)
	features.AverageSpeed, features.MaxSpeed, features.MinSpeed = aba.calculateSpeedStats(points)
	features.PathEfficiency = aba.calculatePathEfficiency(points)
	features.DirectionChanges = aba.countDirectionChanges(points)
	features.CurvatureAvg = aba.calculateCurvatureAvg(points)
	features.JitterScore = aba.calculateJitterScore(points, smoothedPoints)
	features.PauseCount = aba.countPauses(points)
	features.MicroCorrections = aba.countMicroCorrections(points)
	features.SpeedVariance = aba.calculateSpeedVariance(points)
	features.AccelerationAvg = aba.calculateAccelerationAvg(points)
	features.AccelerationMagVariance = aba.calculateAccelerationMagVariance(points)
	features.VelocityEntropy = aba.calculateVelocityEntropy(points)
	features.AngularVelocityAvg = aba.calculateAngularVelocityAvg(points)
	features.TrajectoryComplexity = aba.calculateTrajectoryComplexity(points)
	features.SegmentLengthVariance = aba.calculateSegmentLengthVariance(points)
	features.BezierCurveFitError = aba.calculateBezierCurveFitError(points)
	features.StraightnessDeviation = aba.calculateStraightnessDeviation(points)
	features.HumanLikelihood = aba.calculateHumanLikelihood(features)

	return features
}

func (aba *AdvancedBehaviorAnalyzer) smoothTrajectory(points []BehaviorDataPoint, windowSize int) []BehaviorDataPoint {
	if len(points) < windowSize {
		return points
	}

	if windowSize%2 == 0 {
		windowSize++
	}

	halfWindow := windowSize / 2
	smoothed := make([]BehaviorDataPoint, len(points))

	for i := range points {
		start := i - halfWindow
		end := i + halfWindow

		if start < 0 {
			start = 0
		}
		if end >= len(points) {
			end = len(points) - 1
		}

		sumX := 0
		sumY := 0
		count := 0

		for j := start; j <= end; j++ {
			sumX += points[j].X
			sumY += points[j].Y
			count++
		}

		smoothed[i] = points[i]
		smoothed[i].X = sumX / count
		smoothed[i].Y = sumY / count
	}

	return smoothed
}

func (aba *AdvancedBehaviorAnalyzer) calculateTotalDistance(points []BehaviorDataPoint) float64 {
	totalDistance := 0.0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	return totalDistance
}

func (aba *AdvancedBehaviorAnalyzer) calculateSpeedStats(points []BehaviorDataPoint) (avg, max, min float64) {
	if len(points) < 2 {
		return 0, 0, 0
	}

	speeds := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
		}
	}

	if len(speeds) == 0 {
		return 0, 0, 0
	}

	avg = aba.mean(speeds)
	max = aba.max(speeds)
	min = aba.min(speeds)

	return avg, max, min
}

func (aba *AdvancedBehaviorAnalyzer) calculatePathEfficiency(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	totalDistance := aba.calculateTotalDistance(points)

	firstPoint := points[0]
	lastPoint := points[len(points)-1]
	straightDistance := math.Sqrt(
		math.Pow(float64(lastPoint.X-firstPoint.X), 2) +
			math.Pow(float64(lastPoint.Y-firstPoint.Y), 2),
	)

	if totalDistance > 0 {
		return straightDistance / totalDistance
	}
	return 0
}

func (aba *AdvancedBehaviorAnalyzer) countDirectionChanges(points []BehaviorDataPoint) int {
	if len(points) < 3 {
		return 0
	}

	changes := 0
	prevAngle := 0.0

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)

		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.5 {
				changes++
			}
		}
		prevAngle = angle
	}

	return changes
}

func (aba *AdvancedBehaviorAnalyzer) calculateCurvatureAvg(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	curvatures := []float64{}
	for i := 1; i < len(points)-1; i++ {
		curv := aba.computeCurvature(points[i-1], points[i], points[i+1])
		curvatures = append(curvatures, math.Abs(curv))
	}

	if len(curvatures) == 0 {
		return 0
	}

	return aba.mean(curvatures)
}

func (aba *AdvancedBehaviorAnalyzer) computeCurvature(p1, p2, p3 BehaviorDataPoint) float64 {
	v1x := float64(p2.X - p1.X)
	v1y := float64(p2.Y - p1.Y)
	v2x := float64(p3.X - p2.X)
	v2y := float64(p3.Y - p2.Y)

	dot := v1x*v2x + v1y*v2y
	mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
	mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

	if mag1 == 0 || mag2 == 0 {
		return 0
	}

	cosAngle := dot / (mag1 * mag2)
	if cosAngle > 1 {
		cosAngle = 1
	}
	if cosAngle < -1 {
		cosAngle = -1
	}

	return math.Acos(cosAngle)
}

func (aba *AdvancedBehaviorAnalyzer) calculateJitterScore(original, smoothed []BehaviorDataPoint) float64 {
	if len(original) < 2 || len(smoothed) < 2 {
		return 0
	}

	originalDistance := aba.calculateTotalDistance(original)
	smoothedDistance := aba.calculateTotalDistance(smoothed)

	if originalDistance > 0 && smoothedDistance > 0 {
		return (originalDistance - smoothedDistance) / originalDistance
	}
	return 0
}

func (aba *AdvancedBehaviorAnalyzer) countPauses(points []BehaviorDataPoint) int {
	count := 0
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)

		if distance < 2 && dt > 50 {
			count++
		}
	}
	return count
}

func (aba *AdvancedBehaviorAnalyzer) countMicroCorrections(points []BehaviorDataPoint) int {
	if len(points) < 3 {
		return 0
	}

	corrections := 0
	prevAngle := 0.0

	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		angle := math.Atan2(dy, dx)
		distance := math.Sqrt(dx*dx + dy*dy)

		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 2.0 && distance < 10 {
				corrections++
			}
		}
		prevAngle = angle
	}

	return corrections
}

func (aba *AdvancedBehaviorAnalyzer) calculateSpeedVariance(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	speeds := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}

	if len(speeds) < 2 {
		return 0
	}

	mean := aba.mean(speeds)
	variance := 0.0
	for _, s := range speeds {
		variance += math.Pow(s-mean, 2)
	}
	return variance / float64(len(speeds))
}

func (aba *AdvancedBehaviorAnalyzer) calculateAccelerationAvg(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	speeds := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}

	if len(speeds) < 2 {
		return 0
	}

	accelerations := []float64{}
	for i := 1; i < len(speeds); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) == 0 {
		return 0
	}

	return aba.mean(accelerations)
}

func (aba *AdvancedBehaviorAnalyzer) calculateAccelerationMagVariance(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	speeds := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}

	if len(speeds) < 2 {
		return 0
	}

	accelMagnitudes := []float64{}
	for i := 1; i < len(speeds); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			accel := math.Abs((speeds[i] - speeds[i-1]) / dt)
			accelMagnitudes = append(accelMagnitudes, accel)
		}
	}

	if len(accelMagnitudes) < 2 {
		return 0
	}

	mean := aba.mean(accelMagnitudes)
	variance := 0.0
	for _, m := range accelMagnitudes {
		variance += math.Pow(m-mean, 2)
	}
	return variance / float64(len(accelMagnitudes))
}

func (aba *AdvancedBehaviorAnalyzer) calculateVelocityEntropy(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	speeds := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}

	if len(speeds) == 0 {
		return 0
	}

	buckets := 10
	counts := make([]int, buckets)
	minSpeed := aba.min(speeds)
	maxSpeed := aba.max(speeds)
	rangeSpeed := maxSpeed - minSpeed

	if rangeSpeed == 0 {
		return 0
	}

	for _, s := range speeds {
		bucket := int((s - minSpeed) / rangeSpeed * float64(buckets))
		if bucket >= buckets {
			bucket = buckets - 1
		}
		counts[bucket]++
	}

	entropy := 0.0
	total := float64(len(speeds))
	for _, c := range counts {
		if c > 0 {
			p := float64(c) / total
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (aba *AdvancedBehaviorAnalyzer) calculateAngularVelocityAvg(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	angularVelocities := []float64{}
	for i := 1; i < len(points)-1; i++ {
		dx1 := float64(points[i].X - points[i-1].X)
		dy1 := float64(points[i].Y - points[i-1].Y)
		dx2 := float64(points[i+1].X - points[i].X)
		dy2 := float64(points[i+1].Y - points[i].Y)

		angle1 := math.Atan2(dy1, dx1)
		angle2 := math.Atan2(dy2, dx2)

		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}

		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			angularVelocities = append(angularVelocities, angleDiff/dt)
		}
	}

	if len(angularVelocities) == 0 {
		return 0
	}

	return aba.mean(angularVelocities)
}

func (aba *AdvancedBehaviorAnalyzer) calculateTrajectoryComplexity(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	totalDistance := aba.calculateTotalDistance(points)
	if totalDistance == 0 {
		return 0
	}

	firstPoint := points[0]
	lastPoint := points[len(points)-1]
	straightDistance := math.Sqrt(
		math.Pow(float64(lastPoint.X-firstPoint.X), 2) +
			math.Pow(float64(lastPoint.Y-firstPoint.Y), 2),
	)

	if straightDistance == 0 {
		return 0
	}

	return (totalDistance - straightDistance) / totalDistance
}

func (aba *AdvancedBehaviorAnalyzer) calculateSegmentLengthVariance(points []BehaviorDataPoint) float64 {
	if len(points) < 3 {
		return 0
	}

	segmentLengths := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		segmentLengths = append(segmentLengths, math.Sqrt(dx*dx+dy*dy))
	}

	if len(segmentLengths) < 2 {
		return 0
	}

	mean := aba.mean(segmentLengths)
	variance := 0.0
	for _, l := range segmentLengths {
		variance += math.Pow(l-mean, 2)
	}
	return variance / float64(len(segmentLengths))
}

func (aba *AdvancedBehaviorAnalyzer) calculateBezierCurveFitError(points []BehaviorDataPoint) float64 {
	if len(points) < 4 {
		return 0
	}

	n := len(points)
	p0 := points[0]
	p3 := points[n-1]

	tValues := []float64{}
	for i := 1; i < n-1; i++ {
		t := float64(i) / float64(n-1)
		tValues = append(tValues, t)
	}

	totalError := 0.0
	for i, t := range tValues {
		bezierPoint := aba.evaluateBezierPoint(p0, points[i+1], points[i+1], p3, t)
		actualPoint := points[i+1]

		dx := float64(bezierPoint.X - actualPoint.X)
		dy := float64(bezierPoint.Y - actualPoint.Y)
		error := math.Sqrt(dx*dx + dy*dy)
		totalError += error
	}

	return totalError / float64(len(tValues))
}

func (aba *AdvancedBehaviorAnalyzer) evaluateBezierPoint(p0, p1, p2, p3 BehaviorDataPoint, t float64) BehaviorDataPoint {
	t2 := t * t
	t3 := t2 * t
	mt := 1 - t
	mt2 := mt * mt
	mt3 := mt2 * mt

	x := mt3*float64(p0.X) + 3*mt2*t*float64(p1.X) + 3*mt*t2*float64(p2.X) + t3*float64(p3.X)
	y := mt3*float64(p0.Y) + 3*mt2*t*float64(p1.Y) + 3*mt*t2*float64(p2.Y) + t3*float64(p3.Y)

	return BehaviorDataPoint{
		X: int(x),
		Y: int(y),
	}
}

func (aba *AdvancedBehaviorAnalyzer) calculateStraightnessDeviation(points []BehaviorDataPoint) float64 {
	if len(points) < 2 {
		return 0
	}

	firstPoint := points[0]
	lastPoint := points[len(points)-1]
	dx := float64(lastPoint.X - firstPoint.X)
	dy := float64(lastPoint.Y - firstPoint.Y)
	straightDistance := math.Sqrt(dx*dx + dy*dy)

	if straightDistance == 0 {
		return 0
	}

	totalDeviation := 0.0
	for _, p := range points {
		lineDeviation := aba.pointToLineDistance(p, firstPoint, lastPoint)
		totalDeviation += lineDeviation
	}

	return totalDeviation / float64(len(points))
}

func (aba *AdvancedBehaviorAnalyzer) pointToLineDistance(p, lineStart, lineEnd BehaviorDataPoint) float64 {
	dx := float64(lineEnd.X - lineStart.X)
	dy := float64(lineEnd.Y - lineStart.Y)

	if dx == 0 && dy == 0 {
		return math.Sqrt(math.Pow(float64(p.X-lineStart.X), 2) + math.Pow(float64(p.Y-lineStart.Y), 2))
	}

	t := (float64(p.X-lineStart.X)*dx + float64(p.Y-lineStart.Y)*dy) / (dx*dx + dy*dy)

	closestX := float64(lineStart.X) + t*dx
	closestY := float64(lineStart.Y) + t*dy

	return math.Sqrt(math.Pow(float64(p.X)-closestX, 2) + math.Pow(float64(p.Y)-closestY, 2))
}

func (aba *AdvancedBehaviorAnalyzer) calculateHumanLikelihood(features *AdvancedTrajectoryFeatures) float64 {
	likelihood := 1.0

	if features.PathEfficiency > 0.92 {
		likelihood *= 0.3
	}

	if features.JitterScore < 0.03 {
		likelihood *= 0.4
	}

	if features.CurvatureAvg < 0.05 {
		likelihood *= 0.5
	}

	if features.PauseCount == 0 && len(features.Points) >= 20 {
		likelihood *= 0.6
	}

	if features.MicroCorrections == 0 && len(features.Points) >= 20 {
		likelihood *= 0.7
	}

	if features.VelocityEntropy < 2.0 {
		likelihood *= 0.8
	}

	if features.TrajectoryComplexity < 0.1 {
		likelihood *= 0.5
	}

	return math.Min(likelihood, 1.0)
}

func (aba *AdvancedBehaviorAnalyzer) AnalyzePressure(pressureData []PressureDataPoint) *PressureFeatures {
	features := &PressureFeatures{
		Points: pressureData,
	}

	if len(pressureData) == 0 {
		return features
	}

	pressures := []float64{}
	for _, p := range pressureData {
		pressures = append(pressures, p.Pressure)
	}

	features.AveragePressure = aba.mean(pressures)
	features.PressureVariance = aba.variance(pressures)
	features.MaxPressure = aba.max(pressures)
	features.MinPressure = aba.min(pressures)

	features.PressureDistribution = aba.computePressureDistribution(pressures, 10)

	cv := 0.0
	if features.AveragePressure > 0 {
		cv = math.Sqrt(features.PressureVariance) / features.AveragePressure
	}
	features.IsPressureConsistent = cv < 0.2

	for i := 1; i < len(pressures); i++ {
		diff := math.Abs(pressures[i] - pressures[i-1])
		if diff > features.AveragePressure*0.5 {
			features.PressureAnomalies++
		}
	}

	return features
}

func (aba *AdvancedBehaviorAnalyzer) computePressureDistribution(pressures []float64, buckets int) []float64 {
	if len(pressures) == 0 {
		return make([]float64, buckets)
	}

	distribution := make([]float64, buckets)
	minP := aba.min(pressures)
	maxP := aba.max(pressures)
	rangeP := maxP - minP

	if rangeP == 0 {
		distribution[0] = 1.0
		return distribution
	}

	for _, p := range pressures {
		bucket := int((p - minP) / rangeP * float64(buckets))
		if bucket >= buckets {
			bucket = buckets - 1
		}
		distribution[bucket]++
	}

	total := float64(len(pressures))
	for i := range distribution {
		distribution[i] /= total
	}

	return distribution
}

func (aba *AdvancedBehaviorAnalyzer) AnalyzeSwipeJitter(points []BehaviorDataPoint) *SwipeJitterFeatures {
	features := &SwipeJitterFeatures{}

	if len(points) < 3 {
		return features
	}

	jitterMagnitudes := []float64{}
	jitterDirections := []float64{}

	for i := 1; i < len(points)-1; i++ {
		dx1 := float64(points[i].X - points[i-1].X)
		dy1 := float64(points[i].Y - points[i-1].Y)
		dx2 := float64(points[i+1].X - points[i].X)
		dy2 := float64(points[i+1].Y - points[i].Y)

		segment1Len := math.Sqrt(dx1*dx1 + dy1*dy1)
		segment2Len := math.Sqrt(dx2*dx2 + dy2*dy2)

		if segment1Len > 0 && segment2Len > 0 {
			jitterMag := math.Sqrt(math.Pow(segment1Len-segment2Len, 2))
			jitterMagnitudes = append(jitterMagnitudes, jitterMag)

			angle1 := math.Atan2(dy1, dx1)
			angle2 := math.Atan2(dy2, dx2)
			jitterDir := math.Abs(angle2 - angle1)
			if jitterDir > math.Pi {
				jitterDir = 2*math.Pi - jitterDir
			}
			jitterDirections = append(jitterDirections, jitterDir)
		}
	}

	if len(jitterMagnitudes) > 0 {
		features.JitterMagnitude = aba.mean(jitterMagnitudes)
		features.JitterFrequency = 1.0 / (aba.mean(jitterMagnitudes) + 0.001)
	}

	if len(jitterDirections) > 0 {
		features.JitterDirection = aba.mean(jitterDirections)
		features.JitterConsistency = 1.0 - math.Min(aba.variance(jitterDirections), 1.0)
	}

	features.IsHumanLike = features.JitterMagnitude > 0.5 && features.JitterMagnitude < 20

	return features
}

func (aba *AdvancedBehaviorAnalyzer) ExtractFeaturesForNeuralNet(points []BehaviorDataPoint) []float64 {
	features := aba.AnalyzeAdvancedTrajectory(points)

	input := make([]float64, 20)

	if len(points) >= 2 {
		speeds := []float64{}
		for i := 1; i < len(points); i++ {
			dx := float64(points[i].X - points[i-1].X)
			dy := float64(points[i].Y - points[i-1].Y)
			distance := math.Sqrt(dx*dx + dy*dy)
			dt := float64(points[i].Timestamp - points[i-1].Timestamp)
			if dt > 0 {
				speeds = append(speeds, distance/dt)
			}
		}
		if len(speeds) > 0 {
			input[0] = aba.mean(speeds)
			input[1] = aba.max(speeds)
			input[2] = aba.min(speeds)
			input[3] = math.Sqrt(aba.variance(speeds))
		}
	}

	input[4] = features.PathEfficiency
	input[5] = features.JitterScore
	input[6] = features.CurvatureAvg
	input[7] = float64(features.PauseCount)
	input[8] = float64(features.MicroCorrections)
	input[9] = features.TotalDistance
	input[10] = features.SpeedVariance
	input[11] = features.AccelerationAvg

	input[12] = 0.5
	input[13] = 2.5
	input[14] = float64(len(points))
	input[15] = 5.0
	input[16] = 100.0
	input[17] = 0.8
	input[18] = features.TrajectoryComplexity
	input[19] = 1.0 - features.HumanLikelihood

	return input
}

func (aba *AdvancedBehaviorAnalyzer) RealTimeAnomalyDetection(points []BehaviorDataPoint) (bool, []string) {
	anomalies := []string{}

	if len(points) < 2 {
		return false, anomalies
	}

	speeds := []float64{}
	for i := 1; i < len(points); i++ {
		dx := float64(points[i].X - points[i-1].X)
		dy := float64(points[i].Y - points[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt)
		}
	}

	if len(speeds) > 0 {
		avgSpeed := aba.mean(speeds)
		maxSpeed := aba.max(speeds)

		if maxSpeed > 10 {
			anomalies = append(anomalies, fmt.Sprintf("超高速移动: %.2f", maxSpeed))
		}

		cv := 0.0
		if avgSpeed > 0 {
			cv = math.Sqrt(aba.variance(speeds)) / avgSpeed
		}
		if cv < 0.1 && len(speeds) > 5 {
			anomalies = append(anomalies, "速度过于恒定")
		}
	}

	pathEfficiency := aba.calculatePathEfficiency(points)
	if pathEfficiency > 0.92 {
		anomalies = append(anomalies, fmt.Sprintf("路径过于笔直: %.2f", pathEfficiency))
	}

	pauseCount := aba.countPauses(points)
	if pauseCount == 0 && len(points) >= 20 {
		anomalies = append(anomalies, "无停顿行为")
	}

	microCorrections := aba.countMicroCorrections(points)
	if microCorrections == 0 && len(points) >= 20 {
		anomalies = append(anomalies, "无微修正行为")
	}

	smoothed := aba.smoothTrajectory(points, 5)
	jitterScore := aba.calculateJitterScore(points, smoothed)
	if jitterScore < 0.03 {
		anomalies = append(anomalies, fmt.Sprintf("轨迹抖动过低: %.4f", jitterScore))
	}

	curvatureAvg := aba.calculateCurvatureAvg(points)
	if curvatureAvg < 0.05 && len(points) > 20 {
		anomalies = append(anomalies, fmt.Sprintf("曲率过低: %.4f", curvatureAvg))
	}

	isAnomaly := len(anomalies) >= 2

	return isAnomaly, anomalies
}

func (aba *AdvancedBehaviorAnalyzer) OptimizeRiskScore(baseScore float64, features *AdvancedTrajectoryFeatures, anomalies []string) float64 {
	optimizedScore := baseScore

	if len(anomalies) >= 3 {
		optimizedScore += 15
	}

	if features.HumanLikelihood < 0.3 {
		optimizedScore += 10
	}

	if features.VelocityEntropy < 2.0 {
		optimizedScore += 5
	}

	if features.TrajectoryComplexity < 0.1 {
		optimizedScore += 10
	}

	if features.AccelerationMagVariance < 0.001 {
		optimizedScore += 8
	}

	return math.Min(optimizedScore, 100)
}

func (aba *AdvancedBehaviorAnalyzer) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (aba *AdvancedBehaviorAnalyzer) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := aba.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}

func (aba *AdvancedBehaviorAnalyzer) max(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values {
		if v > max {
			max = v
		}
	}
	return max
}

func (aba *AdvancedBehaviorAnalyzer) min(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values {
		if v < min {
			min = v
		}
	}
	return min
}

func (aba *AdvancedBehaviorAnalyzer) AnalyzeBehaviorAdvanced(behaviorData []models.BehaviorData) (*AdvancedAnalysisResult, error) {
	result := &AdvancedAnalysisResult{
		Anomalies: []string{},
	}

	var points []BehaviorDataPoint
	var clicks []BehaviorDataPoint
	var keyStrokes []KeyboardDataPoint

	for _, bd := range behaviorData {
		switch bd.DataType {
		case "keyboard":
			var kp KeyboardDataPoint
			if err := json.Unmarshal([]byte(bd.Data), &kp); err == nil {
				keyStrokes = append(keyStrokes, kp)
			}
		default:
			var dp BehaviorDataPoint
			if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
				points = append(points, dp)
				if dp.Event == "click" {
					clicks = append(clicks, dp)
				}
			}
		}
	}

	if len(points) > 0 {
		result.TrajectoryFeatures = aba.AnalyzeAdvancedTrajectory(points)
		result.IsAnomaly, result.Anomalies = aba.RealTimeAnomalyDetection(points)
		result.NeuralNetFeatures = aba.ExtractFeaturesForNeuralNet(points)
	}

	result.BaseRiskScore = aba.Ensemble.AdvancedRuleEng.Evaluate(aba.Ensemble.extractFeaturesFromInput(result.NeuralNetFeatures))
	result.OptimizedRiskScore = aba.OptimizeRiskScore(result.BaseRiskScore, result.TrajectoryFeatures, result.Anomalies)
	result.IsBot, result.Confidence = aba.Ensemble.Classify(result.NeuralNetFeatures)

	return result, nil
}

type AdvancedAnalysisResult struct {
	TrajectoryFeatures  *AdvancedTrajectoryFeatures
	NeuralNetFeatures   []float64
	IsAnomaly          bool
	Anomalies          []string
	BaseRiskScore      float64
	OptimizedRiskScore float64
	IsBot              bool
	Confidence         float64
}

func (aba *AdvancedBehaviorAnalyzer) TrainNeuralNetwork(trainingData []TrainingSample) {
	for _, sample := range trainingData {
		aba.Ensemble.NeuralNet.Train(sample.Input, sample.Target)
	}
}

type TrainingSample struct {
	Input  []float64
	Target []float64
}

func GenerateSyntheticHumanTrajectories(count int) [][]BehaviorDataPoint {
	trajectories := make([][]BehaviorDataPoint, 0)

	for i := 0; i < count; i++ {
		trajectory := GenerateSyntheticHumanTrajectory()
		trajectories = append(trajectories, trajectory)
	}

	return trajectories
}

func GenerateSyntheticHumanTrajectory() []BehaviorDataPoint {
	rand.Seed(time.Now().UnixNano())

	points := make([]BehaviorDataPoint, 0)

	startX := rand.Intn(200) + 100
	startY := rand.Intn(200) + 100
	timestamp := int64(0)

	currentX := startX
	currentY := startY

	targetX := startX + rand.Intn(200) + 100
	targetY := startY + rand.Intn(200) + 100

	steps := rand.Intn(30) + 20

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)

		targetXCurrent := float64(startX) + t*float64(targetX-startX)
		targetYCurrent := float64(startY) + t*float64(targetY-startY)

		jitterX := (rand.Float64()*2 - 1) * 15
		jitterY := (rand.Float64()*2 - 1) * 15

		currentX = int(targetXCurrent + jitterX)
		currentY = int(targetYCurrent + jitterY)

		timestamp += int64(rand.Intn(30) + 20)

		points = append(points, BehaviorDataPoint{
			X:         currentX,
			Y:         currentY,
			Timestamp: timestamp,
			Event:     "move",
		})
	}

	return points
}

func GenerateSyntheticBotTrajectory() []BehaviorDataPoint {
	rand.Seed(time.Now().UnixNano())

	points := make([]BehaviorDataPoint, 0)

	startX := rand.Intn(200) + 100
	startY := rand.Intn(200) + 100
	timestamp := int64(0)

	targetX := startX + rand.Intn(200) + 100
	targetY := startY + rand.Intn(200) + 100

	steps := rand.Intn(20) + 15

	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)

		currentX := int(float64(startX) + t*float64(targetX-startX))
		currentY := int(float64(startY) + t*float64(targetY-startY))

		timestamp += int64(rand.Intn(5) + 5)

		points = append(points, BehaviorDataPoint{
			X:         currentX,
			Y:         currentY,
			Timestamp: timestamp,
			Event:     "move",
		})
	}

	return points
}
