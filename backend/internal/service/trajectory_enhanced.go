package service

import (
	"context"
	"errors"
	"math"
	"sort"

	"github.com/hjtpx/hjtpx/internal/model"
)

const (
	DefaultJitterThreshold      = 3.0
	DefaultJitterWindowSize     = 5
	MinJitterSegmentLength      = 3
	CurvatureThresholdSharpTurn = 0.5
	CurvatureThresholdSmoothTurn = 0.2
	SpeedFitDefaultDegree       = 3
	SpeedFitMinPoints           = 10
	SmoothnessMinPoints         = 3
)

type TrajectoryEnhancedAnalyzer struct {
	jitterThreshold   float64
	jitterWindowSize  int
	speedFitDegree    int
}

func NewTrajectoryEnhancedAnalyzer() *TrajectoryEnhancedAnalyzer {
	return &TrajectoryEnhancedAnalyzer{
		jitterThreshold:  DefaultJitterThreshold,
		jitterWindowSize: DefaultJitterWindowSize,
		speedFitDegree:   SpeedFitDefaultDegree,
	}
}

func (a *TrajectoryEnhancedAnalyzer) AnalyzeTrajectory(ctx context.Context, traceData *model.TraceData) (*model.EnhancedTrajectoryAnalysis, error) {
	if traceData == nil || len(traceData.Points) < 2 {
		return nil, errors.New("轨迹数据不足")
	}

	jitterResult := a.DetectJitter(traceData)
	curvatureResult := a.AnalyzeCurvature(traceData)
	speedFitResult, err := a.FitSpeedCurve(traceData)
	if err != nil {
		speedFitResult = &model.SpeedCurveFitResult{}
	}
	smoothnessResult := a.EvaluateSmoothness(traceData)

	overallScore := a.calculateOverallScore(jitterResult, curvatureResult, speedFitResult, smoothnessResult)
	anomalyIndicators := a.detectAnomalyIndicators(jitterResult, curvatureResult, speedFitResult, smoothnessResult)
	confidenceLevel := a.calculateConfidenceLevel(traceData)

	return &model.EnhancedTrajectoryAnalysis{
		JitterResult:      jitterResult,
		CurvatureResult:   curvatureResult,
		SpeedFitResult:    speedFitResult,
		SmoothnessResult:  smoothnessResult,
		OverallScore:      overallScore,
		AnomalyIndicators: anomalyIndicators,
		ConfidenceLevel:   confidenceLevel,
	}, nil
}

func (a *TrajectoryEnhancedAnalyzer) DetectJitter(traceData *model.TraceData) *model.JitterDetectionResult {
	if traceData == nil || len(traceData.Points) < 3 {
		return &model.JitterDetectionResult{
			JitterCount:        0,
			JitterRatio:        0,
			AvgJitterAmplitude: 0,
			MaxJitterAmplitude: 0,
			JitterFrequency:    0,
			IsJittery:         false,
			JitterScore:        0,
			JitterPositions:    []int{},
		}
	}

	points := traceData.Points
	jitterThreshold := a.jitterThreshold
	windowSize := a.jitterWindowSize

	if windowSize > len(points)-1 {
		windowSize = len(points) - 1
	}
	if windowSize < 3 {
		windowSize = 3
	}

	distances := make([]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distances[i-1] = math.Sqrt(dx*dx + dy*dy)
	}

	_ = a.calculateSpeedThreshold(distances, points)
	angularChanges := a.calculateAngularChanges(points)

	jitterCount := 0
	jitterAmplitudes := []float64{}
	jitterPositions := []int{}

	for i := windowSize; i < len(points)-windowSize; i++ {
		startIdx := i - windowSize/2
		endIdx := i + windowSize/2
		if startIdx < 0 {
			startIdx = 0
		}
		if endIdx >= len(points) {
			endIdx = len(points) - 1
		}

		localMean := 0.0
		localCount := 0
		for j := startIdx; j <= endIdx; j++ {
			if j > 0 {
				localMean += distances[j-1]
				localCount++
			}
		}
		if localCount > 0 {
			localMean /= float64(localCount)
		}

		centerDist := 0.0
		if i > 0 {
			centerDist = distances[i-1]
		}

		angularChange := 0.0
		if i < len(angularChanges) {
			angularChange = angularChanges[i]
		}

		isJitter := centerDist < jitterThreshold &&
			centerDist < localMean*0.3 &&
			angularChange > math.Pi/6

		if isJitter {
			jitterCount++
			jitterPositions = append(jitterPositions, i)
			jitterAmplitudes = append(jitterAmplitudes, centerDist)
		}
	}

	totalPoints := len(points)
	jitterRatio := float64(jitterCount) / float64(totalPoints)

	avgJitterAmplitude := 0.0
	maxJitterAmplitude := 0.0
	if len(jitterAmplitudes) > 0 {
		sum := 0.0
		for _, amp := range jitterAmplitudes {
			sum += amp
			if amp > maxJitterAmplitude {
				maxJitterAmplitude = amp
			}
		}
		avgJitterAmplitude = sum / float64(len(jitterAmplitudes))
	}

	jitterFrequency := float64(jitterCount) / float64(traceData.TotalTime) * 1000.0

	isJittery := jitterRatio > 0.1 || jitterFrequency > 5.0

	jitterScore := a.calculateJitterScore(jitterRatio, jitterFrequency, avgJitterAmplitude)

	return &model.JitterDetectionResult{
		JitterCount:        jitterCount,
		JitterRatio:        jitterRatio,
		AvgJitterAmplitude: avgJitterAmplitude,
		MaxJitterAmplitude: maxJitterAmplitude,
		JitterFrequency:    jitterFrequency,
		IsJittery:         isJittery,
		JitterScore:        jitterScore,
		JitterPositions:    jitterPositions,
	}
}

func (a *TrajectoryEnhancedAnalyzer) calculateSpeedThreshold(distances []float64, points []model.TracePoint) float64 {
	if len(distances) == 0 {
		return DefaultJitterThreshold
	}

	mean := 0.0
	for _, d := range distances {
		mean += d
	}
	mean /= float64(len(distances))

	variance := 0.0
	for _, d := range distances {
		diff := d - mean
		variance += diff * diff
	}
	variance /= float64(len(distances))

	stdDev := math.Sqrt(variance)
	threshold := mean + 2*stdDev

	if threshold < DefaultJitterThreshold {
		threshold = DefaultJitterThreshold
	}

	return threshold
}

func (a *TrajectoryEnhancedAnalyzer) calculateAngularChanges(points []model.TracePoint) []float64 {
	angularChanges := make([]float64, len(points))

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
			angularChanges[i] = math.Acos(cosAngle)
		}
	}

	return angularChanges
}

func (a *TrajectoryEnhancedAnalyzer) calculateJitterScore(jitterRatio, jitterFrequency, avgAmplitude float64) float64 {
	ratioScore := math.Min(jitterRatio*10, 1.0)
	frequencyScore := math.Min(jitterFrequency/10.0, 1.0)
	amplitudeScore := math.Min(avgAmplitude/DefaultJitterThreshold, 1.0)

	score := ratioScore*0.4 + frequencyScore*0.3 + amplitudeScore*0.3

	return math.Max(0, math.Min(1, score))
}

func (a *TrajectoryEnhancedAnalyzer) AnalyzeCurvature(traceData *model.TraceData) *model.TrajectoryCurvatureResult {
	if traceData == nil || len(traceData.Points) < 3 {
		return &model.TrajectoryCurvatureResult{
			AvgCurvature:       0,
			MaxCurvature:       0,
			MinCurvature:       0,
			CurvatureVariance:  0,
			CurvatureSkewness:  0,
			CurvatureKurtosis:  0,
			CurvatureEntropy:   0,
			SharpTurnCount:     0,
			SmoothTurnCount:    0,
			DirectionChanges:   0,
			CurvatureScore:    1.0,
		}
	}

	points := traceData.Points
	curvatures := make([]float64, len(points)-2)

	for i := 1; i < len(points)-1; i++ {
		curvatures[i-1] = a.computeCurvature(points[i-1], points[i], points[i+1])
	}

	avgCurvature := calculateMean(curvatures)
	maxCurvature := calculateMax(curvatures)
	minCurvature := calculateMin(curvatures)
	curvatureVariance := calculateVariance(curvatures, avgCurvature)
	curvatureSkewness := calculateSkewness(curvatures, avgCurvature, curvatureVariance)
	curvatureKurtosis := calculateKurtosis(curvatures, avgCurvature, curvatureVariance)
	curvatureEntropy := calculateEntropy(curvatures)

	directionChanges := a.countDirectionChanges(points)

	sharpTurnCount := 0
	smoothTurnCount := 0
	for _, c := range curvatures {
		if c > CurvatureThresholdSharpTurn {
			sharpTurnCount++
		} else if c > CurvatureThresholdSmoothTurn {
			smoothTurnCount++
		}
	}

	curvatureScore := a.calculateCurvatureScore(avgCurvature, curvatureVariance, directionChanges)

	return &model.TrajectoryCurvatureResult{
		AvgCurvature:      avgCurvature,
		MaxCurvature:      maxCurvature,
		MinCurvature:      minCurvature,
		CurvatureVariance: curvatureVariance,
		CurvatureSkewness: curvatureSkewness,
		CurvatureKurtosis: curvatureKurtosis,
		CurvatureEntropy:  curvatureEntropy,
		SharpTurnCount:    sharpTurnCount,
		SmoothTurnCount:   smoothTurnCount,
		DirectionChanges:  directionChanges,
		CurvatureScore:    curvatureScore,
	}
}

func (a *TrajectoryEnhancedAnalyzer) computeCurvature(p1, p2, p3 model.TracePoint) float64 {
	v1x := p2.X - p1.X
	v1y := p2.Y - p1.Y
	v2x := p3.X - p2.X
	v2y := p3.Y - p2.Y

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

func (a *TrajectoryEnhancedAnalyzer) countDirectionChanges(points []model.TracePoint) int {
	if len(points) < 2 {
		return 0
	}

	changes := 0
	var prevAngle float64 = -1000

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y

		if math.Abs(dx) < 0.001 && math.Abs(dy) < 0.001 {
			continue
		}

		angle := math.Atan2(dy, dx)

		if prevAngle != -1000 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > math.Pi/4 {
				changes++
			}
		}

		prevAngle = angle
	}

	return changes
}

func (a *TrajectoryEnhancedAnalyzer) calculateCurvatureScore(avgCurvature, variance float64, directionChanges int) float64 {
	curvatureScore := 1.0 - math.Min(avgCurvature/math.Pi, 1.0)
	varianceScore := 1.0 - math.Min(math.Sqrt(variance)/math.Pi, 1.0)
	changeScore := 1.0 - math.Min(float64(directionChanges)/20.0, 1.0)

	return (curvatureScore*0.4 + varianceScore*0.3 + changeScore*0.3)
}

func (a *TrajectoryEnhancedAnalyzer) FitSpeedCurve(traceData *model.TraceData) (*model.SpeedCurveFitResult, error) {
	if traceData == nil || len(traceData.Points) < SpeedFitMinPoints {
		return &model.SpeedCurveFitResult{
			Coefficients:        []float64{},
			FittedSpeeds:        []float64{},
			Residuals:           []float64{},
			RMSE:                0,
			R2Score:             0,
			Degree:              a.speedFitDegree,
			FittedCurvePoints:   []float64{},
			SpeedFluctuation:    0,
			AccelerationPattern: "insufficient_data",
		}, errors.New("轨迹点不足")
	}

	points := traceData.Points
	speeds := a.calculateSpeeds(points)

	if len(speeds) < SpeedFitMinPoints {
		return &model.SpeedCurveFitResult{
			Coefficients:     []float64{},
			FittedSpeeds:     []float64{},
			Residuals:        []float64{},
			RMSE:             0,
			R2Score:          0,
			Degree:           a.speedFitDegree,
			FittedCurvePoints: []float64{},
			SpeedFluctuation: 0,
			AccelerationPattern: "insufficient_data",
		}, errors.New("速度数据不足")
	}

	normalizedSpeeds := a.normalizeSpeeds(speeds)

	coefficients := a.polynomialFit(normalizedSpeeds, a.speedFitDegree)

	fittedSpeeds := a.evaluatePolynomial(normalizedSpeeds, coefficients)

	residuals := make([]float64, len(speeds))
	for i := range speeds {
		if i < len(fittedSpeeds) {
			residuals[i] = speeds[i] - fittedSpeeds[i]
		}
	}

	rmse := calculateRMSE(speeds, fittedSpeeds)
	r2Score := calculateR2Score(speeds, fittedSpeeds)

	speedFluctuation := a.calculateSpeedFluctuation(speeds, fittedSpeeds)
	accelerationPattern := a.determineAccelerationPattern(speeds)

	return &model.SpeedCurveFitResult{
		Coefficients:       coefficients,
		FittedSpeeds:       fittedSpeeds,
		Residuals:          residuals,
		RMSE:               rmse,
		R2Score:            r2Score,
		Degree:             a.speedFitDegree,
		FittedCurvePoints: a.generateFittedCurvePoints(len(speeds)),
		SpeedFluctuation:   speedFluctuation,
		AccelerationPattern: accelerationPattern,
	}, nil
}

func (a *TrajectoryEnhancedAnalyzer) calculateSpeeds(points []model.TracePoint) []float64 {
	speeds := make([]float64, len(points)-1)

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		dist := math.Sqrt(dx*dx + dy*dy)

		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			speeds[i-1] = dist / dt
		} else {
			speeds[i-1] = 0
		}
	}

	return speeds
}

func (a *TrajectoryEnhancedAnalyzer) normalizeSpeeds(speeds []float64) []float64 {
	if len(speeds) == 0 {
		return speeds
	}

	maxSpeed := calculateMax(speeds)
	if maxSpeed == 0 {
		maxSpeed = 1
	}

	normalized := make([]float64, len(speeds))
	for i, s := range speeds {
		normalized[i] = s / maxSpeed
	}

	return normalized
}

func (a *TrajectoryEnhancedAnalyzer) polynomialFit(x []float64, degree int) []float64 {
	n := len(x)
	if n == 0 {
		return []float64{}
	}

	X := make([][]float64, n)
	for i := range X {
		X[i] = make([]float64, degree+1)
		for j := 0; j <= degree; j++ {
			X[i][j] = math.Pow(float64(i), float64(j))
		}
	}

	XT := transpose(X)
	XTX := matMul(XT, X)
	XTY := matVecMul(XT, x)

	coefficients := solveLinearSystem(XTX, XTY, degree+1)

	return coefficients
}

func (a *TrajectoryEnhancedAnalyzer) evaluatePolynomial(x []float64, coefficients []float64) []float64 {
	result := make([]float64, len(x))

	for i, xi := range x {
		val := 0.0
		for j, coef := range coefficients {
			val += coef * math.Pow(xi, float64(j))
		}
		result[i] = val
	}

	return result
}

func (a *TrajectoryEnhancedAnalyzer) generateFittedCurvePoints(numPoints int) []float64 {
	points := make([]float64, numPoints)
	for i := 0; i < numPoints; i++ {
		points[i] = float64(i) / float64(numPoints-1)
	}
	return points
}

func (a *TrajectoryEnhancedAnalyzer) calculateSpeedFluctuation(speeds, fittedSpeeds []float64) float64 {
	if len(speeds) == 0 || len(fittedSpeeds) == 0 {
		return 0
	}

	meanSpeed := calculateMean(speeds)
	if meanSpeed == 0 {
		return 0
	}

	totalFluctuation := 0.0
	count := 0
	for i := range speeds {
		if i < len(fittedSpeeds) {
			fluctuation := math.Abs(speeds[i] - fittedSpeeds[i]) / meanSpeed
			totalFluctuation += fluctuation
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return totalFluctuation / float64(count)
}

func (a *TrajectoryEnhancedAnalyzer) determineAccelerationPattern(speeds []float64) string {
	if len(speeds) < 3 {
		return "unknown"
	}

	accelerations := make([]float64, len(speeds)-1)
	for i := 0; i < len(speeds)-1; i++ {
		accelerations[i] = speeds[i+1] - speeds[i]
	}

	positiveAccel := 0
	negativeAccel := 0
	constantAccel := 0

	for _, acc := range accelerations {
		if acc > 0.1 {
			positiveAccel++
		} else if acc < -0.1 {
			negativeAccel++
		} else {
			constantAccel++
		}
	}

	avgAccel := calculateMean(accelerations)
	accelVariance := calculateVariance(accelerations, avgAccel)

	if accelVariance < 0.01 {
		return "constant"
	}

	if float64(positiveAccel) > float64(len(accelerations))*0.7 {
		return "accelerating"
	}

	if float64(negativeAccel) > float64(len(accelerations))*0.7 {
		return "decelerating"
	}

	return "variable"
}

func (a *TrajectoryEnhancedAnalyzer) EvaluateSmoothness(traceData *model.TraceData) *model.TrajectorySmoothnessResult {
	if traceData == nil || len(traceData.Points) < SmoothnessMinPoints {
		return &model.TrajectorySmoothnessResult{
			SmoothnessScore:     1.0,
			AvgAngularChange:    0,
			MaxAngularChange:    0,
			AngularVariance:     0,
			LinearDeviation:     0,
			PathEfficiency:      1.0,
			MovementContinuity:  1.0,
			SmoothRatio:         1.0,
			RaggedRatio:         0,
			OverallFluidity:     1.0,
		}
	}

	points := traceData.Points

	angularChanges := a.calculateAngularChangesFull(points)
	avgAngularChange := calculateMean(angularChanges)
	maxAngularChange := calculateMax(angularChanges)
	angularVariance := calculateVariance(angularChanges, avgAngularChange)

	linearDeviation := a.calculateLinearDeviation(points)
	pathEfficiency := a.calculatePathEfficiency(points)
	movementContinuity := a.calculateMovementContinuity(points)

	smoothCount := 0
	raggedCount := 0
	for _, change := range angularChanges {
		if change < math.Pi/12 {
			smoothCount++
		} else if change > math.Pi/4 {
			raggedCount++
		}
	}

	totalSegments := len(angularChanges)
	smoothRatio := 0.0
	raggedRatio := 0.0
	if totalSegments > 0 {
		smoothRatio = float64(smoothCount) / float64(totalSegments)
		raggedRatio = float64(raggedCount) / float64(totalSegments)
	}

	smoothnessScore := a.calculateSmoothnessScore(avgAngularChange, angularVariance, smoothRatio)
	overallFluidity := a.calculateOverallFluidity(smoothnessScore, pathEfficiency, movementContinuity)

	return &model.TrajectorySmoothnessResult{
		SmoothnessScore:    smoothnessScore,
		AvgAngularChange:   avgAngularChange,
		MaxAngularChange:   maxAngularChange,
		AngularVariance:    angularVariance,
		LinearDeviation:    linearDeviation,
		PathEfficiency:     pathEfficiency,
		MovementContinuity: movementContinuity,
		SmoothRatio:        smoothRatio,
		RaggedRatio:        raggedRatio,
		OverallFluidity:    overallFluidity,
	}
}

func (a *TrajectoryEnhancedAnalyzer) calculateAngularChangesFull(points []model.TracePoint) []float64 {
	angularChanges := make([]float64, 0, len(points)-2)

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
			angularChanges = append(angularChanges, math.Acos(cosAngle))
		}
	}

	return angularChanges
}

func (a *TrajectoryEnhancedAnalyzer) calculateLinearDeviation(points []model.TracePoint) float64 {
	if len(points) < 2 {
		return 0
	}

	startX, startY := points[0].X, points[0].Y
	endX, endY := points[len(points)-1].X, points[len(points)-1].Y

	totalDistance := 0.0
	directDistance := math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	if directDistance == 0 {
		return 0
	}

	return (totalDistance - directDistance) / directDistance
}

func (a *TrajectoryEnhancedAnalyzer) calculatePathEfficiency(points []model.TracePoint) float64 {
	if len(points) < 2 {
		return 1.0
	}

	startX, startY := points[0].X, points[0].Y
	endX, endY := points[len(points)-1].X, points[len(points)-1].Y

	totalDistance := 0.0
	directDistance := math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))

	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}

	if totalDistance == 0 {
		return 0
	}

	return directDistance / totalDistance
}

func (a *TrajectoryEnhancedAnalyzer) calculateMovementContinuity(points []model.TracePoint) float64 {
	if len(points) < 2 {
		return 1.0
	}

	timeGaps := make([]float64, len(points)-1)
	for i := 1; i < len(points); i++ {
		timeGaps[i-1] = float64(points[i].Timestamp - points[i-1].Timestamp)
	}

	avgGap := calculateMean(timeGaps)
	if avgGap == 0 {
		return 1.0
	}

	variance := calculateVariance(timeGaps, avgGap)
	normalizedVariance := variance / (avgGap * avgGap)

	continuity := 1.0 / (1.0 + normalizedVariance)

	return math.Max(0, math.Min(1, continuity))
}

func (a *TrajectoryEnhancedAnalyzer) calculateSmoothnessScore(avgAngularChange, angularVariance, smoothRatio float64) float64 {
	angularScore := 1.0 - math.Min(avgAngularChange/(math.Pi/2), 1.0)
	varianceScore := 1.0 - math.Min(math.Sqrt(angularVariance)/(math.Pi/2), 1.0)
	smoothScore := smoothRatio

	return (angularScore*0.4 + varianceScore*0.3 + smoothScore*0.3)
}

func (a *TrajectoryEnhancedAnalyzer) calculateOverallFluidity(smoothnessScore, pathEfficiency, continuity float64) float64 {
	return (smoothnessScore*0.5 + pathEfficiency*0.3 + continuity*0.2)
}

func (a *TrajectoryEnhancedAnalyzer) calculateOverallScore(jitter *model.JitterDetectionResult, curvature *model.TrajectoryCurvatureResult, speedFit *model.SpeedCurveFitResult, smoothness *model.TrajectorySmoothnessResult) float64 {
	jitterScore := 1.0 - jitter.JitterScore
	curvatureScore := curvature.CurvatureScore
	speedScore := 1.0 - math.Min(speedFit.SpeedFluctuation, 1.0)
	smoothnessScore := smoothness.SmoothnessScore

	overallScore := jitterScore*0.25 + curvatureScore*0.25 + speedScore*0.25 + smoothnessScore*0.25

	return math.Max(0, math.Min(1, overallScore))
}

func (a *TrajectoryEnhancedAnalyzer) detectAnomalyIndicators(jitter *model.JitterDetectionResult, curvature *model.TrajectoryCurvatureResult, speedFit *model.SpeedCurveFitResult, smoothness *model.TrajectorySmoothnessResult) []string {
	indicators := []string{}

	if jitter.IsJittery {
		indicators = append(indicators, "excessive_jitter")
	}

	if jitter.JitterRatio > 0.15 {
		indicators = append(indicators, "high_jitter_ratio")
	}

	if curvature.DirectionChanges > 15 {
		indicators = append(indicators, "too_many_direction_changes")
	}

	if curvature.SharpTurnCount > curvature.SmoothTurnCount*3 {
		indicators = append(indicators, "sharp_turn_pattern")
	}

	if speedFit.AccelerationPattern == "constant" && speedFit.SpeedFluctuation < 0.05 {
		indicators = append(indicators, "mechanical_movement")
	}

	if smoothness.RaggedRatio > 0.5 {
		indicators = append(indicators, "highly_ragged_trajectory")
	}

	if smoothness.PathEfficiency < 0.7 {
		indicators = append(indicators, "inefficient_path")
	}

	if speedFit.R2Score < 0.3 {
		indicators = append(indicators, "irregular_speed_pattern")
	}

	return indicators
}

func (a *TrajectoryEnhancedAnalyzer) calculateConfidenceLevel(traceData *model.TraceData) float64 {
	pointCount := len(traceData.Points)
	totalTime := traceData.TotalTime

	pointScore := math.Min(float64(pointCount)/50.0, 1.0)
	timeScore := 1.0
	if totalTime > 0 {
		avgTimeBetweenPoints := float64(totalTime) / float64(pointCount)
		if avgTimeBetweenPoints > 1000 {
			timeScore = 0.5
		} else if avgTimeBetweenPoints > 500 {
			timeScore = 0.8
		}
	}

	confidence := (pointScore*0.6 + timeScore*0.4)

	return math.Max(0.1, math.Min(1.0, confidence))
}

func calculateMean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	maxVal := values[0]
	for _, v := range values[1:] {
		if v > maxVal {
			maxVal = v
		}
	}
	return maxVal
}

func calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	minVal := values[0]
	for _, v := range values[1:] {
		if v < minVal {
			minVal = v
		}
	}
	return minVal
}

func calculateVariance(values []float64, mean float64) float64 {
	if len(values) < 2 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func calculateSkewness(values []float64, mean, variance float64) float64 {
	if len(values) < 3 || variance == 0 {
		return 0
	}
	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func calculateKurtosis(values []float64, mean, variance float64) float64 {
	if len(values) < 4 || variance == 0 {
		return 0
	}
	stdDev := math.Sqrt(variance)
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	return (sum / float64(len(values))) - 3.0
}

func calculateEntropy(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}

	sortedValues := make([]float64, len(values))
	copy(sortedValues, values)
	sort.Float64s(sortedValues)

	bucketCount := 10
	bucketSize := (sortedValues[len(sortedValues)-1] - sortedValues[0]) / float64(bucketCount)
	if bucketSize == 0 {
		return 0
	}

	counts := make([]int, bucketCount)
	for _, v := range values {
		bucket := int((v - sortedValues[0]) / bucketSize)
		if bucket >= bucketCount {
			bucket = bucketCount - 1
		}
		if bucket < 0 {
			bucket = 0
		}
		counts[bucket]++
	}

	entropy := 0.0
	total := float64(len(values))
	for _, count := range counts {
		if count > 0 {
			p := float64(count) / total
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func calculateRMSE(actual, predicted []float64) float64 {
	if len(actual) == 0 || len(predicted) == 0 {
		return 0
	}

	sumSquaredError := 0.0
	count := 0
	for i := range actual {
		if i < len(predicted) {
			diff := actual[i] - predicted[i]
			sumSquaredError += diff * diff
			count++
		}
	}

	if count == 0 {
		return 0
	}

	return math.Sqrt(sumSquaredError / float64(count))
}

func calculateR2Score(actual, predicted []float64) float64 {
	if len(actual) == 0 || len(predicted) == 0 {
		return 0
	}

	mean := calculateMean(actual)
	totalSumSquares := 0.0
	residualSumSquares := 0.0

	count := 0
	for i := range actual {
		if i < len(predicted) {
			totalSumSquares += (actual[i] - mean) * (actual[i] - mean)
			residualSumSquares += (actual[i] - predicted[i]) * (actual[i] - predicted[i])
			count++
		}
	}

	if count == 0 || totalSumSquares == 0 {
		return 0
	}

	return 1.0 - (residualSumSquares / totalSumSquares)
}

func transpose(matrix [][]float64) [][]float64 {
	if len(matrix) == 0 {
		return [][]float64{}
	}
	rows := len(matrix)
	cols := len(matrix[0])
	result := make([][]float64, cols)
	for i := range result {
		result[i] = make([]float64, rows)
	}
	for i := 0; i < rows; i++ {
		for j := 0; j < cols; j++ {
			result[j][i] = matrix[i][j]
		}
	}
	return result
}

func matMul(a, b [][]float64) [][]float64 {
	if len(a) == 0 || len(b) == 0 {
		return [][]float64{}
	}
	rowsA, colsA := len(a), len(a[0])
	rowsB, colsB := len(b), len(b[0])

	if colsA != rowsB {
		return [][]float64{}
	}

	result := make([][]float64, rowsA)
	for i := range result {
		result[i] = make([]float64, colsB)
	}

	for i := 0; i < rowsA; i++ {
		for j := 0; j < colsB; j++ {
			sum := 0.0
			for k := 0; k < colsA; k++ {
				sum += a[i][k] * b[k][j]
			}
			result[i][j] = sum
		}
	}

	return result
}

func matVecMul(matrix [][]float64, vector []float64) []float64 {
	if len(matrix) == 0 || len(vector) == 0 {
		return []float64{}
	}
	result := make([]float64, len(matrix))
	for i := range matrix {
		sum := 0.0
		for j := range vector {
			if j < len(matrix[i]) {
				sum += matrix[i][j] * vector[j]
			}
		}
		result[i] = sum
	}
	return result
}

func solveLinearSystem(a [][]float64, b []float64, n int) []float64 {
	if n == 0 {
		return []float64{}
	}

	augmented := make([][]float64, n)
	for i := range augmented {
		augmented[i] = make([]float64, n+1)
		for j := 0; j < n; j++ {
			if i < len(a) && j < len(a[i]) {
				augmented[i][j] = a[i][j]
			}
		}
		if i < len(b) {
			augmented[i][n] = b[i]
		}
	}

	for i := 0; i < n; i++ {
		maxRow := i
		for k := i + 1; k < n; k++ {
			if math.Abs(augmented[k][i]) > math.Abs(augmented[maxRow][i]) {
				maxRow = k
			}
		}

		augmented[i], augmented[maxRow] = augmented[maxRow], augmented[i]

		if math.Abs(augmented[i][i]) < 1e-10 {
			continue
		}

		for k := i + 1; k < n; k++ {
			factor := augmented[k][i] / augmented[i][i]
			for j := i; j <= n; j++ {
				augmented[k][j] -= factor * augmented[i][j]
			}
		}
	}

	x := make([]float64, n)
	for i := n - 1; i >= 0; i-- {
		if math.Abs(augmented[i][i]) < 1e-10 {
			x[i] = 0
			continue
		}
		x[i] = augmented[i][n]
		for j := i + 1; j < n; j++ {
			x[i] -= augmented[i][j] * x[j]
		}
		x[i] /= augmented[i][i]
	}

	return x
}
