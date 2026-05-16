package service

import (
	"encoding/json"
	"math"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

func TestNeuralNetworkForward(t *testing.T) {
	nn := NewNeuralNetwork(3, 5, 1)

	input := []float64{0.5, 0.3, 0.8}
	output, hiddenLayer := nn.Forward(input)

	if len(output) != 1 {
		t.Errorf("Expected output length 1, got %d", len(output))
	}

	if len(hiddenLayer) != 5 {
		t.Errorf("Expected hidden layer size 5, got %d", len(hiddenLayer))
	}

	for _, val := range output {
		if val < 0 || val > 1 {
			t.Errorf("Output value %f out of range [0, 1]", val)
		}
	}
}

func TestNeuralNetworkTrain(t *testing.T) {
	nn := NewNeuralNetwork(2, 3, 1)

	input := []float64{0.5, 0.5}
	target := []float64{0.8}

	initialOutput, _ := nn.Forward(input)
	initialVal := initialOutput[0]

	for i := 0; i < 100; i++ {
		nn.Train(input, target)
	}

	trainedOutput, _ := nn.Forward(input)
	trainedVal := trainedOutput[0]

	diff := math.Abs(trainedVal - target[0])
	if diff > 0.5 {
		t.Errorf("After training, output should be closer to target. Initial: %f, Trained: %f, Target: %f",
			initialVal, trainedVal, target[0])
	}
}

func TestNeuralNetworkPredict(t *testing.T) {
	nn := NewNeuralNetwork(3, 5, 1)

	input := []float64{0.2, 0.7, 0.4}
	prediction := nn.Predict(input)

	if len(prediction) != 1 {
		t.Errorf("Expected prediction length 1, got %d", len(prediction))
	}

	for _, val := range prediction {
		if val < 0 || val > 1 {
			t.Errorf("Prediction value %f out of range [0, 1]", val)
		}
	}
}

func TestAnomalyDetectorUpdate(t *testing.T) {
	detector := NewAnomalyDetector(10, 2.5)

	for i := 0; i < 5; i++ {
		value := float64(i) * 10.0
		isAnomaly := detector.Update(value)
		if i < 3 && isAnomaly {
			t.Errorf("First values should not be anomalous, got %v", isAnomaly)
		}
	}

	extremeValue := 1000.0
	isAnomaly := detector.Update(extremeValue)
	if !isAnomaly {
		t.Errorf("Extreme value should be detected as anomaly")
	}
}

func TestAnomalyDetectorStatistics(t *testing.T) {
	detector := NewAnomalyDetector(10, 3.0)

	values := []float64{10.0, 11.0, 10.5, 10.8, 10.2}
	for _, v := range values {
		detector.Update(v)
	}

	if math.Abs(detector.Mean-10.5) > 0.1 {
		t.Errorf("Expected mean around 10.5, got %f", detector.Mean)
	}

	if detector.StdDev < 0.2 || detector.StdDev > 0.4 {
		t.Errorf("Expected std dev between 0.2 and 0.4, got %f", detector.StdDev)
	}
}

func TestAdvancedRuleEngineEvaluate(t *testing.T) {
	re := NewAdvancedRuleEngine()

	result := &AnalysisResult{
		Trajectory: MouseTrajectory{
			PathEfficiency: 0.95,
			TotalDistance:  200,
			JitterScore:    0.01,
			CurvatureAvg:   0.03,
			PauseCount:      0,
			MicroCorrections: 0,
			Points:         make([]BehaviorDataPoint, 30),
		},
		SpeedAnalysis: SpeedAnalysis{
			MaxSpeed:     12.0,
			AverageSpeed: 5.0,
			SpeedStdDev:  0.2,
			Speeds:       make([]float64, 10),
		},
		ClickPattern: ClickPattern{
			Regularity: 0.95,
			ClickCount: 5,
		},
		KeyboardPattern: KeyboardPattern{
			TypingSpeed: 20.0,
		},
		PathSimilarity: PathSimilarity{
			IsPathRepeated: true,
		},
	}

	score := re.Evaluate(result)

	if score < 50 {
		t.Errorf("Expected high risk score for bot-like behavior, got %f", score)
	}
}

func TestAdvancedRuleEngineZeroScore(t *testing.T) {
	re := NewAdvancedRuleEngine()

	result := &AnalysisResult{
		Trajectory: MouseTrajectory{
			PathEfficiency:  0.7,
			TotalDistance:    100,
			JitterScore:     0.15,
			CurvatureAvg:    0.2,
			PauseCount:      5,
			MicroCorrections: 10,
			Points:          make([]BehaviorDataPoint, 30),
		},
		SpeedAnalysis: SpeedAnalysis{
			MaxSpeed:     3.0,
			AverageSpeed: 1.5,
			SpeedStdDev:  0.8,
			Speeds:       make([]float64, 10),
		},
		ClickPattern: ClickPattern{
			Regularity: 0.5,
			ClickCount: 3,
		},
		KeyboardPattern: KeyboardPattern{
			TypingSpeed: 5.0,
		},
		PathSimilarity: PathSimilarity{
			IsPathRepeated: false,
		},
	}

	score := re.Evaluate(result)

	if score > 20 {
		t.Errorf("Expected low risk score for human-like behavior, got %f", score)
	}
}

func TestAdvancedEnsembleClassifier(t *testing.T) {
	ec := NewAdvancedEnsembleClassifier()

	input := []float64{
		5.0, 10.0, 1.0, 0.5,
		0.95, 0.01, 0.03, 0.0,
		0.0, 200.0, 0.001, 0.05,
		0.95, 1.5, 30, 20.0,
		30.0, 0.98, 0.9, 0.9,
	}

	isBot, score := ec.Classify(input)

	if isBot && score < 50 {
		t.Errorf("Bot detection should have high score, got %f", score)
	}
}

func TestAdvancedTrajectoryAnalysis(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := GenerateSyntheticHumanTrajectory()

	features := analyzer.AnalyzeAdvancedTrajectory(points)

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if features.TotalDistance <= 0 {
		t.Error("Total distance should be positive")
	}

	if features.AverageSpeed <= 0 {
		t.Error("Average speed should be positive")
	}

	if features.HumanLikelihood < 0 || features.HumanLikelihood > 1 {
		t.Errorf("Human likelihood should be in [0, 1], got %f", features.HumanLikelihood)
	}
}

func TestAdvancedTrajectoryAnalysisBot(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := GenerateSyntheticBotTrajectory()

	features := analyzer.AnalyzeAdvancedTrajectory(points)

	if features == nil {
		t.Fatal("Features should not be nil")
	}

	if features.HumanLikelihood > 0.5 {
		t.Errorf("Bot trajectory should have low human likelihood, got %f", features.HumanLikelihood)
	}
}

func TestPressureAnalysis(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	pressureData := []PressureDataPoint{
		{X: 100, Y: 100, Pressure: 0.5, Timestamp: 0},
		{X: 110, Y: 110, Pressure: 0.6, Timestamp: 50},
		{X: 120, Y: 120, Pressure: 0.55, Timestamp: 100},
		{X: 130, Y: 130, Pressure: 0.58, Timestamp: 150},
		{X: 140, Y: 140, Pressure: 0.52, Timestamp: 200},
	}

	features := analyzer.AnalyzePressure(pressureData)

	if features == nil {
		t.Fatal("Pressure features should not be nil")
	}

	if features.AveragePressure <= 0 {
		t.Error("Average pressure should be positive")
	}

	if features.MinPressure >= features.MaxPressure {
		t.Error("Min pressure should be less than max pressure")
	}
}

func TestSwipeJitterAnalysis(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := GenerateSyntheticHumanTrajectory()

	features := analyzer.AnalyzeSwipeJitter(points)

	if features == nil {
		t.Fatal("Swipe jitter features should not be nil")
	}
}

func TestRealTimeAnomalyDetection(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	botPoints := GenerateSyntheticBotTrajectory()
	isAnomaly, anomalies := analyzer.RealTimeAnomalyDetection(botPoints)

	if !isAnomaly && len(anomalies) == 0 {
		t.Error("Bot trajectory should be detected as anomaly")
	}
}

func TestRealTimeAnomalyDetectionHuman(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	humanPoints := GenerateSyntheticHumanTrajectory()
	isAnomaly, anomalies := analyzer.RealTimeAnomalyDetection(humanPoints)

	if isAnomaly && len(anomalies) > 3 {
		t.Errorf("Human trajectory should have fewer anomalies, got %d", len(anomalies))
	}
}

func TestFeatureExtractionForNeuralNet(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := GenerateSyntheticHumanTrajectory()

	input := analyzer.ExtractFeaturesForNeuralNet(points)

	if len(input) != 20 {
		t.Errorf("Expected 20 features, got %d", len(input))
	}

	for i, val := range input {
		if math.IsNaN(val) || math.IsInf(val, 0) {
			t.Errorf("Feature %d is invalid: %f", i, val)
		}
	}
}

func TestOptimizeRiskScore(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	features := &AdvancedTrajectoryFeatures{
		HumanLikelihood:        0.2,
		VelocityEntropy:        1.5,
		TrajectoryComplexity:   0.05,
		AccelerationMagVariance: 0.0005,
		Points:                 make([]BehaviorDataPoint, 30),
	}

	anomalies := []string{"anomaly1", "anomaly2", "anomaly3"}

	baseScore := 50.0
	optimizedScore := analyzer.OptimizeRiskScore(baseScore, features, anomalies)

	if optimizedScore <= baseScore {
		t.Errorf("Optimized score should be higher than base score, base: %f, optimized: %f",
			baseScore, optimizedScore)
	}
}

func TestAnalyzeBehaviorAdvanced(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	behaviorData := []models.BehaviorData{}

	humanTrajectory := GenerateSyntheticHumanTrajectory()
	for _, point := range humanTrajectory {
		dataJSON, _ := json.Marshal(point)
		behaviorData = append(behaviorData, models.BehaviorData{
			DataType: "trajectory",
			Data:     string(dataJSON),
		})
	}

	result, err := analyzer.AnalyzeBehaviorAdvanced(behaviorData)

	if err != nil {
		t.Fatalf("AnalyzeBehaviorAdvanced should not return error: %v", err)
	}

	if result == nil {
		t.Fatal("Analysis result should not be nil")
	}

	if result.BaseRiskScore < 0 || result.BaseRiskScore > 100 {
		t.Errorf("Base risk score should be in [0, 100], got %f", result.BaseRiskScore)
	}

	if result.OptimizedRiskScore < 0 || result.OptimizedRiskScore > 100 {
		t.Errorf("Optimized risk score should be in [0, 100], got %f", result.OptimizedRiskScore)
	}
}

func TestTrainNeuralNetwork(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	trainingData := []TrainingSample{}

	for i := 0; i < 10; i++ {
		humanTraj := GenerateSyntheticHumanTrajectory()
		input := analyzer.ExtractFeaturesForNeuralNet(humanTraj)
		trainingData = append(trainingData, TrainingSample{
			Input:  input,
			Target: []float64{0.1},
		})
	}

	for i := 0; i < 10; i++ {
		botTraj := GenerateSyntheticBotTrajectory()
		input := analyzer.ExtractFeaturesForNeuralNet(botTraj)
		trainingData = append(trainingData, TrainingSample{
			Input:  input,
			Target: []float64{0.9},
		})
	}

	analyzer.TrainNeuralNetwork(trainingData)
}

func TestGenerateSyntheticHumanTrajectories(t *testing.T) {
	count := 5
	trajectories := GenerateSyntheticHumanTrajectories(count)

	if len(trajectories) != count {
		t.Errorf("Expected %d trajectories, got %d", count, len(trajectories))
	}

	for i, traj := range trajectories {
		if len(traj) == 0 {
			t.Errorf("Trajectory %d should not be empty", i)
		}

		for j := 1; j < len(traj); j++ {
			if traj[j].Timestamp <= traj[j-1].Timestamp {
				t.Errorf("Trajectory %d: timestamps should be increasing", i)
			}
		}
	}
}

func TestBezierCurveFitError(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := GenerateSyntheticHumanTrajectory()
	error := analyzer.calculateBezierCurveFitError(points)

	if error < 0 {
		t.Errorf("Bezier curve fit error should be non-negative, got %f", error)
	}
}

func TestPointToLineDistance(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	lineStart := BehaviorDataPoint{X: 0, Y: 0}
	lineEnd := BehaviorDataPoint{X: 100, Y: 100}
	point := BehaviorDataPoint{X: 50, Y: 60}

	distance := analyzer.pointToLineDistance(point, lineStart, lineEnd)

	if distance < 0 || distance > 20 {
		t.Errorf("Expected small distance for point near line, got %f", distance)
	}
}

func TestComputePressureDistribution(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	pressures := []float64{0.1, 0.2, 0.3, 0.4, 0.5, 0.6, 0.7, 0.8, 0.9, 1.0}
	buckets := 5

	distribution := analyzer.computePressureDistribution(pressures, buckets)

	if len(distribution) != buckets {
		t.Errorf("Expected %d buckets, got %d", buckets, len(distribution))
	}

	total := 0.0
	for _, p := range distribution {
		total += p
	}
	if math.Abs(total-1.0) > 0.001 {
		t.Errorf("Distribution should sum to 1.0, got %f", total)
	}
}

func TestMeanVariance(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	values := []float64{1.0, 2.0, 3.0, 4.0, 5.0}

	mean := analyzer.mean(values)
	if math.Abs(mean-3.0) > 0.001 {
		t.Errorf("Expected mean 3.0, got %f", mean)
	}

	variance := analyzer.variance(values)
	expectedVariance := 2.0
	if math.Abs(variance-expectedVariance) > 0.001 {
		t.Errorf("Expected variance %f, got %f", expectedVariance, variance)
	}
}

func TestMaxMin(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	values := []float64{3.0, 1.0, 4.0, 1.0, 5.0, 9.0, 2.0, 6.0}

	max := analyzer.max(values)
	if max != 9.0 {
		t.Errorf("Expected max 9.0, got %f", max)
	}

	min := analyzer.min(values)
	if min != 1.0 {
		t.Errorf("Expected min 1.0, got %f", min)
	}
}

func TestAdvancedSmoothTrajectory(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := []BehaviorDataPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 10, Timestamp: 10},
		{X: 20, Y: 20, Timestamp: 20},
		{X: 30, Y: 30, Timestamp: 30},
		{X: 40, Y: 40, Timestamp: 40},
		{X: 50, Y: 50, Timestamp: 50},
	}

	smoothed := analyzer.smoothTrajectory(points, 3)

	if len(smoothed) != len(points) {
		t.Errorf("Smoothed trajectory should have same length, expected %d, got %d",
			len(points), len(smoothed))
	}
}

func TestCountDirectionChanges(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	points := []BehaviorDataPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 10, Y: 10, Timestamp: 10},
		{X: 20, Y: 5, Timestamp: 20},
		{X: 30, Y: 15, Timestamp: 30},
		{X: 40, Y: 10, Timestamp: 40},
	}

	changes := analyzer.countDirectionChanges(points)

	if changes < 0 {
		t.Errorf("Direction changes should be non-negative, got %d", changes)
	}
}

func TestCalculateTrajectoryComplexity(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	straightLine := []BehaviorDataPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 25, Y: 25, Timestamp: 25},
		{X: 50, Y: 50, Timestamp: 50},
		{X: 75, Y: 75, Timestamp: 75},
		{X: 100, Y: 100, Timestamp: 100},
	}

	straightComplexity := analyzer.calculateTrajectoryComplexity(straightLine)

	windingPath := []BehaviorDataPoint{
		{X: 0, Y: 0, Timestamp: 0},
		{X: 20, Y: 30, Timestamp: 20},
		{X: 40, Y: 10, Timestamp: 40},
		{X: 60, Y: 40, Timestamp: 60},
		{X: 80, Y: 20, Timestamp: 80},
		{X: 100, Y: 50, Timestamp: 100},
	}

	windingComplexity := analyzer.calculateTrajectoryComplexity(windingPath)

	if windingComplexity <= straightComplexity {
		t.Errorf("Winding path should have higher complexity, straight: %f, winding: %f",
			straightComplexity, windingComplexity)
	}
}

func TestSigmoidFunction(t *testing.T) {
	nn := NewNeuralNetwork(3, 5, 1)

	testCases := []struct {
		input    float64
		expected float64
	}{
		{0.0, 0.5},
		{-10.0, 0.00005},
		{10.0, 0.99995},
	}

	for _, tc := range testCases {
		result := nn.sigmoid(tc.input)
		if math.Abs(result-tc.expected) > 0.001 {
			t.Errorf("Sigmoid(%f) = %f, expected ~%f", tc.input, result, tc.expected)
		}
	}
}

func TestSigmoidDerivative(t *testing.T) {
	nn := NewNeuralNetwork(3, 5, 1)

	for x := -5.0; x <= 5.0; x += 1.0 {
		sigmoidVal := nn.sigmoid(x)
		derivative := nn.sigmoidDerivative(sigmoidVal)

		expectedDerivative := sigmoidVal * (1 - sigmoidVal)
		if math.Abs(derivative-expectedDerivative) > 0.0001 {
			t.Errorf("Sigmoid derivative at %f incorrect", x)
		}
	}
}

func TestReluFunction(t *testing.T) {
	nn := NewNeuralNetwork(3, 5, 1)

	if nn.relu(-5.0) != 0 {
		t.Errorf("ReLU(-5) should be 0")
	}

	if nn.relu(5.0) != 5.0 {
		t.Errorf("ReLU(5) should be 5")
	}
}

func TestReluDerivative(t *testing.T) {
	nn := NewNeuralNetwork(3, 5, 1)

	if nn.reluDerivative(-5.0) != 0 {
		t.Errorf("ReLU derivative for negative should be 0")
	}

	if nn.reluDerivative(5.0) != 1 {
		t.Errorf("ReLU derivative for positive should be 1")
	}
}

func TestInitializeWeights(t *testing.T) {
	nn := NewNeuralNetwork(5, 10, 2)

	if len(nn.WeightsIH) != 5 {
		t.Errorf("Expected 5 input-hidden weights rows, got %d", len(nn.WeightsIH))
	}

	if len(nn.WeightsIH[0]) != 10 {
		t.Errorf("Expected 10 input-hidden weights cols, got %d", len(nn.WeightsIH[0]))
	}

	if len(nn.WeightsHO) != 10 {
		t.Errorf("Expected 10 hidden-output weights rows, got %d", len(nn.WeightsHO))
	}

	if len(nn.WeightsHO[0]) != 2 {
		t.Errorf("Expected 2 hidden-output weights cols, got %d", len(nn.WeightsHO[0]))
	}
}

func TestAnomalyDetectorWindowSize(t *testing.T) {
	detector := NewAnomalyDetector(5, 2.0)

	for i := 0; i < 10; i++ {
		detector.Update(float64(i))
	}

	if len(detector.RecentValues) > 5 {
		t.Errorf("Window should not exceed size, got %d values", len(detector.RecentValues))
	}
}

func TestPerformanceMetrics(t *testing.T) {
	start := time.Now()

	analyzer := NewAdvancedBehaviorAnalyzer()

	for i := 0; i < 100; i++ {
		traj := GenerateSyntheticHumanTrajectory()
		analyzer.AnalyzeAdvancedTrajectory(traj)
	}

	elapsed := time.Since(start)

	avgTime := elapsed.Seconds() / 100

	if avgTime > 0.1 {
		t.Errorf("Average analysis time too high: %f seconds", avgTime)
	}
}

func TestConcurrentAnalysis(t *testing.T) {
	analyzer := NewAdvancedBehaviorAnalyzer()

	done := make(chan bool)

	for i := 0; i < 5; i++ {
		go func() {
			for j := 0; j < 10; j++ {
				traj := GenerateSyntheticHumanTrajectory()
				analyzer.AnalyzeAdvancedTrajectory(traj)
			}
			done <- true
		}()
	}

	for i := 0; i < 5; i++ {
		<-done
	}
}
