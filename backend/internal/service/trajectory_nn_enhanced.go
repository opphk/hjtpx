package service

import (
	"math"
	"sort"
	"sync"
	"time"
)

type TrajectoryNNEnhanced struct {
	FeatureDim     int
	HiddenDim      int
	jitterDetector *JitterDetector
	mu             sync.RWMutex
}

type JitterDetector struct {
	Threshold      float64
	MinJitterCount int
	JitterPatterns map[string]*JitterPattern
}

type JitterPattern struct {
	PatternType    string
	AmplitudeRange [2]float64
	FrequencyRange [2]float64
	IsSuspicious   bool
}

func NewTrajectoryNNEnhanced() *TrajectoryNNEnhanced {
	return &TrajectoryNNEnhanced{
		FeatureDim: 768,
		HiddenDim:  256,
		jitterDetector: &JitterDetector{
			Threshold:      2.5,
			MinJitterCount: 3,
			JitterPatterns: map[string]*JitterPattern{
				"high_frequency_low_amplitude": {
					PatternType:    "high_frequency_low_amplitude",
					AmplitudeRange: [2]float64{0.1, 2.0},
					FrequencyRange: [2]float64{5.0, 20.0},
					IsSuspicious:   true,
				},
				"low_frequency_high_amplitude": {
					PatternType:    "low_frequency_high_amplitude",
					AmplitudeRange: [2]float64{5.0, 15.0},
					FrequencyRange: [2]float64{1.0, 5.0},
					IsSuspicious:   false,
				},
				"regular_sinusoidal": {
					PatternType:    "regular_sinusoidal",
					AmplitudeRange: [2]float64{1.0, 10.0},
					FrequencyRange: [2]float64{2.0, 8.0},
					IsSuspicious:   true,
				},
			},
		},
	}
}

func (t *TrajectoryNNEnhanced) ExtractFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 768)

	basicFeatures := t.extractBasicFeatures(trajectory)
	copy(features[0:64], basicFeatures)

	speedFeatures := t.extractSpeedFeatures(trajectory)
	copy(features[64:192], speedFeatures)

	directionFeatures := t.extractDirectionFeatures(trajectory)
	copy(features[192:320], directionFeatures)

	curvatureFeatures := t.extractCurvatureFeatures(trajectory)
	copy(features[320:448], curvatureFeatures)

	positionFeatures := t.extractPositionFeatures(trajectory)
	copy(features[448:576], positionFeatures)

	behaviorFeatures := t.extractBehaviorFeatures(trajectory)
	copy(features[576:768], behaviorFeatures)

	advancedFeatures := t.extractAdvancedFeatures(trajectory)
	if len(advancedFeatures) > 0 {
		copy(features[640:768], advancedFeatures[:128])
	}

	return features
}

func (t *TrajectoryNNEnhanced) extractAdvancedFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 128)

	if len(trajectory) < 3 {
		return features
	}

	jitterFeatures := t.extractJitterFeatures(trajectory)
	copy(features[0:32], jitterFeatures)

	temporalFeatures := t.extractTemporalFeatures(trajectory)
	copy(features[32:64], temporalFeatures)

	accelerationFeatures := t.extractAccelerationFeatures(trajectory)
	copy(features[64:96], accelerationFeatures)

	fourierFeatures := t.extractFourierFeatures(trajectory)
	copy(features[96:128], fourierFeatures)

	return features
}

func (t *TrajectoryNNEnhanced) extractBasicFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 64)

	if len(trajectory) == 0 {
		return features
	}

	totalDist := t.calculateTotalDistance(trajectory)
	features[0] = math.Min(totalDist/1000.0, 1.0)

	features[1] = math.Min(float64(len(trajectory))/1000.0, 1.0)

	features[2] = trajectory[0].X / 1000.0
	features[3] = trajectory[0].Y / 1000.0

	if len(trajectory) > 0 {
		features[4] = trajectory[len(trajectory)-1].X / 1000.0
		features[5] = trajectory[len(trajectory)-1].Y / 1000.0
	}

	features[6] = t.calculateAverageSpeed(trajectory)
	features[7] = t.calculateMaxSpeed(trajectory)
	features[8] = t.calculateMinSpeed(trajectory)
	features[9] = t.calculateSpeedVariance(trajectory)

	return features
}

func (t *TrajectoryNNEnhanced) extractSpeedFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 128)

	if len(trajectory) < 2 {
		return features
	}

	speeds := t.calculateSpeeds(trajectory)

	for i := 0; i < 16 && i*len(speeds)/16 < len(speeds); i++ {
		start := i * len(speeds) / 16
		end := (i + 1) * len(speeds) / 16
		if end > len(speeds) {
			end = len(speeds)
		}

		segmentSpeeds := speeds[start:end]
		features[i*8] = t.calculateAverage(segmentSpeeds)
		features[i*8+1] = t.calculateMax(segmentSpeeds)
		features[i*8+2] = t.calculateMin(segmentSpeeds)
		features[i*8+3] = t.calculateVarianceFromValues(segmentSpeeds)
		features[i*8+4] = t.calculateQuantile(segmentSpeeds, 0.25)
		features[i*8+5] = t.calculateQuantile(segmentSpeeds, 0.5)
		features[i*8+6] = t.calculateQuantile(segmentSpeeds, 0.75)
		features[i*8+7] = t.calculateQuantile(segmentSpeeds, 0.9)
	}

	return features
}

func (t *TrajectoryNNEnhanced) extractDirectionFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 128)

	if len(trajectory) < 2 {
		return features
	}

	directions := t.calculateDirections(trajectory)

	histogram := t.calculateDirectionHistogram(directions, 36)
	copy(features[0:36], histogram)

	directionChanges := t.calculateDirectionChanges(directions)
	copy(features[36:68], directionChanges)

	mainDirection := t.calculateMainDirection(directions)
	copy(features[68:128], mainDirection)

	return features
}

func (t *TrajectoryNNEnhanced) extractCurvatureFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 128)

	if len(trajectory) < 3 {
		return features
	}

	curvatures := t.calculateCurvatures(trajectory)

	features[0] = t.calculateAverage(curvatures)
	features[1] = t.calculateMax(curvatures)
	features[2] = t.calculateMin(curvatures)
	features[3] = t.calculateVarianceFromValues(curvatures)

	features[4] = t.calculateQuantile(curvatures, 0.1)
	features[5] = t.calculateQuantile(curvatures, 0.25)
	features[6] = t.calculateQuantile(curvatures, 0.5)
	features[7] = t.calculateQuantile(curvatures, 0.75)
	features[8] = t.calculateQuantile(curvatures, 0.9)
	features[9] = t.calculateQuantile(curvatures, 0.99)

	for i := 0; i < 100 && i < len(curvatures); i++ {
		features[10+i] = curvatures[i] / (t.calculateMax(curvatures) + 0.001)
	}

	return features
}

func (t *TrajectoryNNEnhanced) extractPositionFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 128)

	if len(trajectory) == 0 {
		return features
	}

	minX, maxX := trajectory[0].X, trajectory[0].X
	minY, maxY := trajectory[0].Y, trajectory[0].Y

	for _, p := range trajectory {
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

	features[0] = minX / 1000.0
	features[1] = maxX / 1000.0
	features[2] = minY / 1000.0
	features[3] = maxY / 1000.0
	features[4] = (maxX - minX) / 1000.0
	features[5] = (maxY - minY) / 1000.0
	features[6] = features[4] / (features[5] + 0.001)

	xHistogram := t.calculatePositionHistogram(trajectory, true, 32)
	yHistogram := t.calculatePositionHistogram(trajectory, false, 32)
	copy(features[7:39], xHistogram)
	copy(features[39:71], yHistogram)

	coverage := t.calculateCoverage(trajectory, 19)
	copy(features[71:128], coverage)

	return features
}

func (t *TrajectoryNNEnhanced) extractBehaviorFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 192)

	pattern := t.classifyBehaviorPattern(trajectory)
	copy(features[0:32], pattern)

	anomalies := t.detectAnomalies(trajectory)
	copy(features[32:64], anomalies)

	learningFeatures := t.extractLearningFeatures(trajectory)
	copy(features[64:160], learningFeatures)

	return features
}

func (t *TrajectoryNNEnhanced) calculateTotalDistance(trajectory []TrajectoryNNPoint) float64 {
	total := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total
}

func (t *TrajectoryNNEnhanced) calculateSpeeds(trajectory []TrajectoryNNPoint) []float64 {
	speeds := make([]float64, 0, len(trajectory)-1)
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			speed := math.Sqrt(dx*dx+dy*dy) / dt
			speeds = append(speeds, speed)
		}
	}
	return speeds
}

func (t *TrajectoryNNEnhanced) calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (t *TrajectoryNNEnhanced) calculateMax(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := values[0]
	for _, v := range values[1:] {
		if v > max {
			max = v
		}
	}
	return max
}

func (t *TrajectoryNNEnhanced) calculateMin(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	min := values[0]
	for _, v := range values[1:] {
		if v < min {
			min = v
		}
	}
	return min
}

func (t *TrajectoryNNEnhanced) calculateVariance(values []float64, mean float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		diff := v - mean
		sum += diff * diff
	}
	return sum / float64(len(values))
}

func (t *TrajectoryNNEnhanced) calculateQuantile(values []float64, quantile float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	idx := int(float64(len(sorted)-1) * quantile)
	return sorted[idx]
}

func (t *TrajectoryNNEnhanced) calculateVarianceFromValues(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := t.calculateAverage(values)
	return t.calculateVariance(values, mean)
}

func (t *TrajectoryNNEnhanced) calculateDirections(trajectory []TrajectoryNNPoint) []float64 {
	if len(trajectory) < 2 {
		return []float64{}
	}

	directions := make([]float64, len(trajectory)-1)
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		directions[i-1] = math.Atan2(dy, dx)
	}
	return directions
}

func (t *TrajectoryNNEnhanced) calculateDirectionHistogram(directions []float64, bins int) []float64 {
	histogram := make([]float64, bins)
	if len(directions) == 0 {
		return histogram
	}

	binWidth := 2 * math.Pi / float64(bins)
	for _, d := range directions {
		normalized := d
		if normalized < 0 {
			normalized += 2 * math.Pi
		}
		binIdx := int(normalized / binWidth)
		if binIdx >= bins {
			binIdx = bins - 1
		}
		histogram[binIdx]++
	}

	total := float64(len(directions))
	for i := range histogram {
		histogram[i] /= total
	}
	return histogram
}

func (t *TrajectoryNNEnhanced) calculateDirectionChanges(directions []float64) []float64 {
	changes := make([]float64, 32)
	if len(directions) < 2 {
		return changes
	}

	changeValues := make([]float64, len(directions)-1)
	for i := 1; i < len(directions); i++ {
		change := directions[i] - directions[i-1]
		if change > math.Pi {
			change -= 2 * math.Pi
		} else if change < -math.Pi {
			change += 2 * math.Pi
		}
		changeValues[i-1] = math.Abs(change)
	}

	if len(changeValues) > 0 {
		changes[0] = t.calculateAverage(changeValues)
		changes[1] = t.calculateMax(changeValues)
		changes[2] = t.calculateMin(changeValues)
		changes[3] = t.calculateVarianceFromValues(changeValues)

		for i := 4; i < 32 && i-4 < len(changeValues); i++ {
			changes[i] = changeValues[i-4]
		}
	}

	return changes
}

func (t *TrajectoryNNEnhanced) calculateMainDirection(directions []float64) []float64 {
	mainDir := make([]float64, 60)
	if len(directions) == 0 {
		return mainDir
	}

	histogram := t.calculateDirectionHistogram(directions, 60)
	copy(mainDir, histogram)

	return mainDir
}

func (t *TrajectoryNNEnhanced) calculateCurvatures(trajectory []TrajectoryNNPoint) []float64 {
	if len(trajectory) < 3 {
		return []float64{}
	}

	curvatures := make([]float64, len(trajectory)-2)
	for i := 1; i < len(trajectory)-1; i++ {
		x0, y0 := trajectory[i-1].X, trajectory[i-1].Y
		x1, y1 := trajectory[i].X, trajectory[i].Y
		x2, y2 := trajectory[i+1].X, trajectory[i+1].Y

		dx1 := x1 - x0
		dy1 := y1 - y0
		dx2 := x2 - x1
		dy2 := y2 - y1

		_ = dx1*dy2 - dy1*dx2
		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			} else if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			curvatures[i-1] = angle / (mag1 + mag2 + 0.001)
		}
	}

	return curvatures
}

func (t *TrajectoryNNEnhanced) calculatePositionHistogram(trajectory []TrajectoryNNPoint, isX bool, bins int) []float64 {
	histogram := make([]float64, bins)
	if len(trajectory) == 0 {
		return histogram
	}

	minVal := trajectory[0].X
	maxVal := trajectory[0].X
	if !isX {
		minVal = trajectory[0].Y
		maxVal = trajectory[0].Y
	}

	for _, p := range trajectory {
		val := p.X
		if !isX {
			val = p.Y
		}
		if val < minVal {
			minVal = val
		}
		if val > maxVal {
			maxVal = val
		}
	}

	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}

	for _, p := range trajectory {
		val := p.X
		if !isX {
			val = p.Y
		}
		normalized := (val - minVal) / rangeVal
		binIdx := int(normalized * float64(bins))
		if binIdx >= bins {
			binIdx = bins - 1
		}
		histogram[binIdx]++
	}

	total := float64(len(trajectory))
	for i := range histogram {
		histogram[i] /= total
	}

	return histogram
}

func (t *TrajectoryNNEnhanced) calculateCoverage(trajectory []TrajectoryNNPoint, gridSize int) []float64 {
	coverage := make([]float64, gridSize*gridSize)
	if len(trajectory) == 0 {
		return coverage
	}

	minX, maxX := trajectory[0].X, trajectory[0].X
	minY, maxY := trajectory[0].Y, trajectory[0].Y

	for _, p := range trajectory {
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

	rangeX := maxX - minX
	rangeY := maxY - minY
	if rangeX == 0 {
		rangeX = 1
	}
	if rangeY == 0 {
		rangeY = 1
	}

	for _, p := range trajectory {
		normX := (p.X - minX) / rangeX
		normY := (p.Y - minY) / rangeY

		gridX := int(normX * float64(gridSize-1))
		gridY := int(normY * float64(gridSize-1))

		if gridX >= gridSize {
			gridX = gridSize - 1
		}
		if gridY >= gridSize {
			gridY = gridSize - 1
		}

		coverage[gridY*gridSize+gridX] = 1.0
	}

	return coverage
}

func (t *TrajectoryNNEnhanced) classifyBehaviorPattern(trajectory []TrajectoryNNPoint) []float64 {
	pattern := make([]float64, 32)
	if len(trajectory) < 2 {
		return pattern
	}

	totalDist := t.calculateTotalDistance(trajectory)
	speeds := t.calculateSpeeds(trajectory)
	directions := t.calculateDirections(trajectory)

	avgSpeed := t.calculateAverage(speeds)
	speedVar := t.calculateVarianceFromValues(speeds)

	pattern[0] = math.Min(totalDist/100.0, 1.0)
	pattern[1] = math.Min(avgSpeed/10.0, 1.0)
	pattern[2] = math.Min(speedVar, 1.0)

	if len(speeds) > 0 {
		speedRatio := t.calculateMax(speeds) / (avgSpeed + 0.001)
		pattern[3] = math.Min(speedRatio/10.0, 1.0)
	}

	if len(directions) > 0 {
		directionEntropy := t.calculateDirectionEntropy(directions)
		pattern[4] = directionEntropy
	}

	if len(trajectory) >= 3 {
		curvatures := t.calculateCurvatures(trajectory)
		avgCurvature := t.calculateAverage(curvatures)
		pattern[5] = math.Min(avgCurvature*10.0, 1.0)
	}

	dx := trajectory[len(trajectory)-1].X - trajectory[0].X
	dy := trajectory[len(trajectory)-1].Y - trajectory[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)
	if totalDist > 0 {
		pattern[6] = directDist / totalDist
	}

	return pattern
}

func (t *TrajectoryNNEnhanced) calculateDirectionEntropy(directions []float64) float64 {
	if len(directions) == 0 {
		return 0
	}

	histogram := t.calculateDirectionHistogram(directions, 8)
	entropy := 0.0
	for _, p := range histogram {
		if p > 0 {
			entropy -= p * math.Log(p+0.0001)
		}
	}
	return entropy / math.Log(8.0)
}

func (t *TrajectoryNNEnhanced) detectAnomalies(trajectory []TrajectoryNNPoint) []float64 {
	anomalies := make([]float64, 32)
	if len(trajectory) < 3 {
		return anomalies
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) > 0 {
		avgSpeed := t.calculateAverage(speeds)
		speedStdDev := math.Sqrt(t.calculateVarianceFromValues(speeds))

		for i, speed := range speeds {
			if i < 32 {
				zScore := (speed - avgSpeed) / (speedStdDev + 0.001)
				if math.Abs(zScore) > 2.0 {
					anomalies[i] = 1.0
				}
			}
		}
	}

	if len(trajectory) >= 3 {
		curvatures := t.calculateCurvatures(trajectory)
		avgCurvature := t.calculateAverage(curvatures)
		curvatureStdDev := math.Sqrt(t.calculateVarianceFromValues(curvatures))

		for i, curvature := range curvatures {
			if i < 32 {
				zScore := (curvature - avgCurvature) / (curvatureStdDev + 0.001)
				if math.Abs(zScore) > 2.5 {
					anomalies[16+i] = 1.0
				}
			}
		}
	}

	return anomalies
}

func (t *TrajectoryNNEnhanced) extractLearningFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 96)
	if len(trajectory) < 2 {
		return features
	}

	speeds := t.calculateSpeeds(trajectory)
	directions := t.calculateDirections(trajectory)

	if len(speeds) > 0 {
		avgSpeed := t.calculateAverage(speeds)
		features[0] = avgSpeed
		features[1] = t.calculateMax(speeds)
		features[2] = t.calculateMin(speeds)
		features[3] = t.calculateVarianceFromValues(speeds)
		features[4] = t.calculateQuantile(speeds, 0.25)
		features[5] = t.calculateQuantile(speeds, 0.5)
		features[6] = t.calculateQuantile(speeds, 0.75)
	}

	if len(directions) > 0 {
		features[7] = t.calculateDirectionEntropy(directions)
	}

	if len(trajectory) >= 3 {
		curvatures := t.calculateCurvatures(trajectory)
		features[8] = t.calculateAverage(curvatures)
		features[9] = t.calculateMax(curvatures)
		features[10] = t.calculateVarianceFromValues(curvatures)
	}

	progress := float64(len(trajectory)) / 100.0
	features[11] = math.Min(progress, 1.0)

	totalDist := t.calculateTotalDistance(trajectory)
	features[12] = totalDist

	dx := trajectory[len(trajectory)-1].X - trajectory[0].X
	dy := trajectory[len(trajectory)-1].Y - trajectory[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)
	features[13] = directDist

	if totalDist > 0 {
		features[14] = directDist / totalDist
	}

	return features
}

func (t *TrajectoryNNEnhanced) extractJitterFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(trajectory) < 3 {
		return features
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 2 {
		return features
	}

	avgSpeed := t.calculateAverage(speeds)
	speedStdDev := math.Sqrt(t.calculateVarianceFromValues(speeds))

	jitterCount := 0
	totalJitterAmplitude := 0.0
	jitterPositions := []int{}

	for i := 1; i < len(speeds); i++ {
		if i < len(speeds) {
			change := math.Abs(speeds[i] - speeds[i-1])
			zScore := (change - avgSpeed) / (speedStdDev + 0.001)

			if zScore > t.jitterDetector.Threshold {
				jitterCount++
				totalJitterAmplitude += change
				jitterPositions = append(jitterPositions, i)
			}
		}
	}

	features[0] = float64(jitterCount)
	features[1] = totalJitterAmplitude / math.Max(1.0, float64(jitterCount))
	features[2] = float64(jitterCount) / float64(len(speeds))

	if len(jitterPositions) >= 2 {
		jitterIntervals := []float64{}
		for i := 1; i < len(jitterPositions); i++ {
			jitterIntervals = append(jitterIntervals, float64(jitterPositions[i]-jitterPositions[i-1]))
		}
		features[3] = t.calculateAverage(jitterIntervals)
		features[4] = t.calculateVarianceFromValues(jitterIntervals)
	}

	curvatures := t.calculateCurvatures(trajectory)
	curvatureVariance := t.calculateVarianceFromValues(curvatures)
	features[5] = curvatureVariance

	directions := t.calculateDirections(trajectory)
	directionChanges := t.calculateDirectionChangesFromDirections(directions)
	features[6] = t.calculateAverage(directionChanges)
	features[7] = t.calculateMax(directionChanges)

	temporalPatterns := t.detectTemporalPatterns(trajectory)
	copy(features[8:16], temporalPatterns)

	humanLikelihood := t.calculateHumanLikelihood(trajectory)
	features[16] = humanLikelihood

	features[17] = t.detectMechanicalMovement(trajectory)
	features[18] = t.detectPerfectStraightness(trajectory)
	features[19] = t.detectExcessiveSmoothness(trajectory)

	perceivedEffort := t.calculatePerceivedEffort(trajectory)
	features[20] = perceivedEffort

	features[21] = t.calculateMovementNaturalness(trajectory)
	features[22] = t.calculateTrajectoryComplexity(trajectory)

	features[23] = t.detectCopiedPattern(trajectory)
	features[24] = t.detectRepeatedPattern(trajectory)

	features[25] = t.calculateInterPointConsistency(trajectory)
	features[26] = t.calculateSegmentRegularity(trajectory)

	return features
}

func (t *TrajectoryNNEnhanced) calculateDirectionChangesFromDirections(directions []float64) []float64 {
	changes := make([]float64, len(directions)-1)
	for i := 1; i < len(directions); i++ {
		change := directions[i] - directions[i-1]
		if change > math.Pi {
			change -= 2 * math.Pi
		} else if change < -math.Pi {
			change += 2 * math.Pi
		}
		changes[i-1] = math.Abs(change)
	}
	return changes
}

func (t *TrajectoryNNEnhanced) detectTemporalPatterns(trajectory []TrajectoryNNPoint) []float64 {
	patterns := make([]float64, 8)
	if len(trajectory) < 2 {
		return patterns
	}

	timeDiffs := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) > 0 {
		avgInterval := t.calculateAverage(timeDiffs)
		variance := t.calculateVarianceFromValues(timeDiffs)

		patterns[0] = avgInterval
		patterns[1] = variance
		patterns[2] = variance / (avgInterval + 0.001)

		uniformCount := 0
		for _, diff := range timeDiffs {
			ratio := diff / (avgInterval + 0.001)
			if ratio > 0.9 && ratio < 1.1 {
				uniformCount++
			}
		}
		patterns[3] = float64(uniformCount) / float64(len(timeDiffs))

		rhythmScore := 0.0
		for i := 1; i < len(timeDiffs); i++ {
			ratio := timeDiffs[i] / (timeDiffs[i-1] + 0.001)
			if ratio > 0.8 && ratio < 1.2 {
				rhythmScore += 1.0
			}
		}
		patterns[4] = rhythmScore / float64(len(timeDiffs)-1)

		patterns[5] = t.detectUniformTiming(timeDiffs)
		patterns[6] = t.detectSuspiciousTiming(timeDiffs)
	}

	return patterns
}

func (t *TrajectoryNNEnhanced) detectUniformTiming(timeDiffs []float64) float64 {
	if len(timeDiffs) < 2 {
		return 0.0
	}

	avg := t.calculateAverage(timeDiffs)
	variance := t.calculateVarianceFromValues(timeDiffs)
	coefficientOfVariation := math.Sqrt(variance) / (avg + 0.001)

	if coefficientOfVariation < 0.05 {
		return 1.0
	}
	return 0.0
}

func (t *TrajectoryNNEnhanced) detectSuspiciousTiming(timeDiffs []float64) float64 {
	if len(timeDiffs) < 2 {
		return 0.0
	}

	avg := t.calculateAverage(timeDiffs)
	suspiciousCount := 0

	for _, diff := range timeDiffs {
		ratio := diff / (avg + 0.001)
		if ratio > 0.95 && ratio < 1.05 {
			suspiciousCount++
		}
	}

	return float64(suspiciousCount) / float64(len(timeDiffs))
}

func (t *TrajectoryNNEnhanced) extractTemporalFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(trajectory) < 2 {
		return features
	}

	if len(trajectory) > 0 {
		totalDuration := float64(trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp)
		features[0] = totalDuration / 1000.0
	}

	timeDiffs := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) > 0 {
		features[1] = t.calculateAverage(timeDiffs)
		features[2] = t.calculateMax(timeDiffs)
		features[3] = t.calculateMin(timeDiffs)
		features[4] = t.calculateVarianceFromValues(timeDiffs)
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) > 0 {
		timeSinceStart := 0.0
		for i := range speeds {
			if i < len(trajectory) {
				timeSinceStart += float64(trajectory[i+1].Timestamp - trajectory[i].Timestamp)
				progress := timeSinceStart / (features[0] * 1000.0 + 0.001)
				features[5] = math.Max(features[5], progress*speeds[i])
			}
		}
	}

	startTime := time.Now().Unix()
	_ = startTime

	return features
}

func (t *TrajectoryNNEnhanced) extractAccelerationFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(trajectory) < 3 {
		return features
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 2 {
		return features
	}

	accelerations := []float64{}
	for i := 1; i < len(speeds); i++ {
		dt := float64(trajectory[i+1].Timestamp - trajectory[i].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) > 0 {
		features[0] = t.calculateAverage(accelerations)
		features[1] = t.calculateMax(accelerations)
		features[2] = t.calculateMin(accelerations)
		features[3] = t.calculateVarianceFromValues(accelerations)
		features[4] = t.calculateQuantile(accelerations, 0.25)
		features[5] = t.calculateQuantile(accelerations, 0.5)
		features[6] = t.calculateQuantile(accelerations, 0.75)
	}

	jerkFeatures := t.calculateJerkFeatures(trajectory)
	copy(features[7:16], jerkFeatures)

	features[16] = t.calculateAccelerationEntropy(trajectory)
	features[17] = t.detectSuddenChanges(trajectory)

	return features
}

func (t *TrajectoryNNEnhanced) calculateJerkFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 9)
	if len(trajectory) < 4 {
		return features
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 3 {
		return features
	}

	jerks := []float64{}
	for i := 2; i < len(speeds); i++ {
		dt1 := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		dt2 := float64(trajectory[i-1].Timestamp - trajectory[i-2].Timestamp)
		avgDt := (dt1 + dt2) / 2.0
		if avgDt > 0 {
			jerk := (speeds[i] - 2*speeds[i-1] + speeds[i-2]) / (avgDt * avgDt)
			jerks = append(jerks, jerk)
		}
	}

	if len(jerks) > 0 {
		features[0] = t.calculateAverage(jerks)
		features[1] = t.calculateMax(jerks)
		features[2] = t.calculateMin(jerks)
		features[3] = t.calculateVarianceFromValues(jerks)
		features[4] = t.calculateQuantile(jerks, 0.1)
		features[5] = t.calculateQuantile(jerks, 0.5)
		features[6] = t.calculateQuantile(jerks, 0.9)
	}

	smoothJerkCount := 0
	for _, jerk := range jerks {
		if math.Abs(jerk) < 0.5 {
			smoothJerkCount++
		}
	}
	if len(jerks) > 0 {
		features[7] = float64(smoothJerkCount) / float64(len(jerks))
	}

	features[8] = t.calculateJerkEntropy(jerks)

	return features
}

func (t *TrajectoryNNEnhanced) calculateAccelerationEntropy(trajectory []TrajectoryNNPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 3 {
		return 0.0
	}

	accelerations := []float64{}
	for i := 1; i < len(speeds); i++ {
		dt := float64(trajectory[i+1].Timestamp - trajectory[i].Timestamp)
		if dt > 0 {
			accel := math.Abs(speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) == 0 {
		return 0.0
	}

	bins := 10
	histogram := make([]float64, bins)
	maxAccel := t.calculateMax(accelerations)
	if maxAccel == 0 {
		maxAccel = 1
	}

	for _, accel := range accelerations {
		binIdx := int(accel / maxAccel * float64(bins))
		if binIdx >= bins {
			binIdx = bins - 1
		}
		histogram[binIdx]++
	}

	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			p := count / float64(len(accelerations))
			entropy -= p * math.Log(p+0.0001)
		}
	}

	return entropy / math.Log(float64(bins))
}

func (t *TrajectoryNNEnhanced) detectSuddenChanges(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 3 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 2 {
		return 0.0
	}

	stdDev := math.Sqrt(t.calculateVarianceFromValues(speeds))

	suddenChangeCount := 0
	for i := 1; i < len(speeds); i++ {
		speedChange := math.Abs(speeds[i] - speeds[i-1])
		zScore := speedChange / (stdDev + 0.001)
		if zScore > 3.0 {
			suddenChangeCount++
		}
	}

	return float64(suddenChangeCount) / float64(len(speeds))
}

func (t *TrajectoryNNEnhanced) calculateJerkEntropy(jerks []float64) float64 {
	if len(jerks) == 0 {
		return 0.0
	}

	bins := 8
	histogram := make([]float64, bins)
	maxJerk := t.calculateMax(jerks)
	minJerk := math.Abs(t.calculateMin(jerks))
	rangeJerk := maxJerk + minJerk
	if rangeJerk == 0 {
		rangeJerk = 1
	}

	for _, jerk := range jerks {
		normalizedJerk := (jerk + minJerk) / rangeJerk
		binIdx := int(normalizedJerk * float64(bins))
		if binIdx >= bins {
			binIdx = bins - 1
		}
		histogram[binIdx]++
	}

	entropy := 0.0
	for _, count := range histogram {
		if count > 0 {
			p := count / float64(len(jerks))
			entropy -= p * math.Log(p+0.0001)
		}
	}

	return entropy / math.Log(float64(bins))
}

func (t *TrajectoryNNEnhanced) extractFourierFeatures(trajectory []TrajectoryNNPoint) []float64 {
	features := make([]float64, 32)
	if len(trajectory) < 8 {
		return features
	}

	xCoords := make([]float64, len(trajectory))
	yCoords := make([]float64, len(trajectory))
	for i, p := range trajectory {
		xCoords[i] = p.X
		yCoords[i] = p.Y
	}

	xSpectrum := t.simpleFFT(xCoords)
	ySpectrum := t.simpleFFT(yCoords)

	if len(xSpectrum) > 0 {
		features[0] = t.calculateAverage(xSpectrum[:len(xSpectrum)/2])
		features[1] = t.calculateMax(xSpectrum[:len(xSpectrum)/2])
		features[2] = t.calculateVarianceFromValues(xSpectrum[:len(xSpectrum)/2])
	}

	if len(ySpectrum) > 0 {
		features[3] = t.calculateAverage(ySpectrum[:len(ySpectrum)/2])
		features[4] = t.calculateMax(ySpectrum[:len(ySpectrum)/2])
		features[5] = t.calculateVarianceFromValues(ySpectrum[:len(ySpectrum)/2])
	}

	dominantFreqX := t.findDominantFrequency(xSpectrum)
	dominantFreqY := t.findDominantFrequency(ySpectrum)
	features[6] = dominantFreqX
	features[7] = dominantFreqY

	spectralEntropy := t.calculateSpectralEntropy(xSpectrum)
	features[8] = spectralEntropy

	periodicityScore := t.calculatePeriodicityScore(xSpectrum)
	features[9] = periodicityScore

	return features
}

func (t *TrajectoryNNEnhanced) simpleFFT(signal []float64) []float64 {
	n := len(signal)
	if n == 0 || n&(n-1) != 0 {
		nearestPow2 := 1
		for nearestPow2 < n {
			nearestPow2 *= 2
		}
		padded := make([]float64, nearestPow2)
		copy(padded, signal)
		signal = padded
	}

	magnitude := make([]float64, len(signal))
	for k := 0; k < len(signal)/2; k++ {
		realSum := 0.0
		imagSum := 0.0
		for n := 0; n < len(signal); n++ {
			angle := 2 * math.Pi * float64(k) * float64(n) / float64(len(signal))
			realSum += signal[n] * math.Cos(angle)
			imagSum += signal[n] * math.Sin(angle)
		}
		magnitude[k] = math.Sqrt(realSum*realSum + imagSum*imagSum)
		magnitude[len(signal)-k-1] = magnitude[k]
	}

	return magnitude
}

func (t *TrajectoryNNEnhanced) findDominantFrequency(spectrum []float64) float64 {
	if len(spectrum) < 2 {
		return 0.0
	}

	maxMag := 0.0
	dominantIdx := 0
	for i := 1; i < len(spectrum)/2; i++ {
		if spectrum[i] > maxMag {
			maxMag = spectrum[i]
			dominantIdx = i
		}
	}

	return float64(dominantIdx)
}

func (t *TrajectoryNNEnhanced) calculateSpectralEntropy(spectrum []float64) float64 {
	if len(spectrum) == 0 {
		return 0.0
	}

	total := 0.0
	for _, mag := range spectrum {
		total += mag * mag
	}

	if total == 0 {
		return 0.0
	}

	entropy := 0.0
	for _, mag := range spectrum {
		p := (mag * mag) / total
		if p > 0 {
			entropy -= p * math.Log(p+0.0001)
		}
	}

	return entropy / math.Log(float64(len(spectrum)))
}

func (t *TrajectoryNNEnhanced) calculatePeriodicityScore(spectrum []float64) float64 {
	if len(spectrum) < 4 {
		return 0.0
	}

	harmonics := []float64{}
	for i := 1; i < len(spectrum)/2; i++ {
		harmonics = append(harmonics, spectrum[i])
	}

	if len(harmonics) == 0 {
		return 0.0
	}

	avgHarmonic := t.calculateAverage(harmonics)
	variance := t.calculateVarianceFromValues(harmonics)

	if avgHarmonic == 0 {
		return 0.0
	}

	coefficientOfVariation := math.Sqrt(variance) / avgHarmonic

	return 1.0 / (1.0 + coefficientOfVariation)
}

func (t *TrajectoryNNEnhanced) calculateHumanLikelihood(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 3 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) == 0 {
		return 0.0
	}

	avgSpeed := t.calculateAverage(speeds)
	speedVariance := t.calculateVarianceFromValues(speeds)

	humanSpeedRange := avgSpeed >= 0.5 && avgSpeed <= 5.0
	naturalVariance := speedVariance > 0.1 && speedVariance < 2.0

	curvatures := t.calculateCurvatures(trajectory)
	avgCurvature := 0.0
	if len(curvatures) > 0 {
		avgCurvature = t.calculateAverage(curvatures)
	}
	naturalCurvature := avgCurvature > 0.01 && avgCurvature < 0.5

	directions := t.calculateDirections(trajectory)
	entropy := t.calculateDirectionEntropy(directions)
	naturalEntropy := entropy > 0.3 && entropy < 0.9

	score := 0.0
	if humanSpeedRange {
		score += 0.3
	}
	if naturalVariance {
		score += 0.3
	}
	if naturalCurvature {
		score += 0.2
	}
	if naturalEntropy {
		score += 0.2
	}

	return score
}

func (t *TrajectoryNNEnhanced) detectMechanicalMovement(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 4 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 2 {
		return 0.0
	}

	avgSpeed := t.calculateAverage(speeds)
	speedVariance := t.calculateVarianceFromValues(speeds)

	if avgSpeed == 0 {
		return 0.0
	}
	coefficientOfVariation := math.Sqrt(speedVariance) / avgSpeed

	if coefficientOfVariation < 0.05 {
		return 1.0
	}

	timeDiffs := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) > 1 {
		uniformTiming := t.detectUniformTiming(timeDiffs)
		if uniformTiming > 0.9 {
			return 1.0
		}
	}

	return math.Max(0, 1.0-coefficientOfVariation*5)
}

func (t *TrajectoryNNEnhanced) detectPerfectStraightness(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 2 {
		return 0.0
	}

	totalDist := t.calculateTotalDistance(trajectory)
	if totalDist == 0 {
		return 0.0
	}

	dx := trajectory[len(trajectory)-1].X - trajectory[0].X
	dy := trajectory[len(trajectory)-1].Y - trajectory[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)

	straightnessRatio := directDist / totalDist

	if straightnessRatio > 0.999 {
		return 1.0
	}

	return straightnessRatio
}

func (t *TrajectoryNNEnhanced) detectExcessiveSmoothness(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 4 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 2 {
		return 0.0
	}

	curvatures := t.calculateCurvatures(trajectory)
	if len(curvatures) == 0 {
		return 0.0
	}

	maxCurvature := t.calculateMax(curvatures)
	avgCurvature := t.calculateAverage(curvatures)

	smoothnessRatio := avgCurvature / (maxCurvature + 0.001)

	avgSpeed := t.calculateAverage(speeds)
	speedVariance := t.calculateVarianceFromValues(speeds)

	if avgSpeed > 0 {
		cv := math.Sqrt(speedVariance) / avgSpeed
		if cv < 0.1 && smoothnessRatio > 0.9 {
			return 1.0
		}
	}

	return smoothnessRatio
}

func (t *TrajectoryNNEnhanced) calculatePerceivedEffort(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 2 {
		return 0.0
	}

	totalDist := t.calculateTotalDistance(trajectory)

	totalDuration := 0.0
	if len(trajectory) > 1 {
		totalDuration = float64(trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp)
	}

	if totalDuration == 0 {
		return 0.0
	}

	avgSpeed := totalDist / totalDuration

	curvatures := t.calculateCurvatures(trajectory)
	avgCurvature := 0.0
	if len(curvatures) > 0 {
		avgCurvature = t.calculateAverage(curvatures)
	}

	directions := t.calculateDirections(trajectory)
	directionEntropy := t.calculateDirectionEntropy(directions)

	effort := avgSpeed * 0.3
	effort += avgCurvature * 10.0 * 0.3
	effort += directionEntropy * 0.4

	return math.Min(1.0, effort)
}

func (t *TrajectoryNNEnhanced) calculateMovementNaturalness(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 3 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) == 0 {
		return 0.0
	}

	speedEntropy := 0.0
	speedHistogram := t.calculateSpeedHistogram(speeds, 10)
	for _, p := range speedHistogram {
		if p > 0 {
			speedEntropy -= p * math.Log(p+0.0001)
		}
	}
	speedEntropy /= math.Log(10.0)

	curvatures := t.calculateCurvatures(trajectory)
	curvatureEntropy := 0.0
	if len(curvatures) > 0 {
		curvatureHistogram := t.calculateSpeedHistogram(curvatures, 8)
		for _, p := range curvatureHistogram {
			if p > 0 {
				curvatureEntropy -= p * math.Log(p+0.0001)
			}
		}
		curvatureEntropy /= math.Log(8.0)
	}

	timeDiffs := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	timingEntropy := 0.0
	if len(timeDiffs) > 0 {
		timingHistogram := t.calculateSpeedHistogram(timeDiffs, 8)
		for _, p := range timingHistogram {
			if p > 0 {
				timingEntropy -= p * math.Log(p+0.0001)
			}
		}
		timingEntropy /= math.Log(8.0)
	}

	naturalness := speedEntropy*0.4 + curvatureEntropy*0.3 + timingEntropy*0.3

	return naturalness
}

func (t *TrajectoryNNEnhanced) calculateSpeedHistogram(values []float64, bins int) []float64 {
	histogram := make([]float64, bins)
	if len(values) == 0 {
		return histogram
	}

	maxVal := t.calculateMax(values)
	minVal := t.calculateMin(values)
	rangeVal := maxVal - minVal
	if rangeVal == 0 {
		rangeVal = 1
	}

	for _, val := range values {
		normalized := (val - minVal) / rangeVal
		binIdx := int(normalized * float64(bins))
		if binIdx >= bins {
			binIdx = bins - 1
		}
		histogram[binIdx]++
	}

	total := float64(len(values))
	for i := range histogram {
		histogram[i] /= total
	}

	return histogram
}

func (t *TrajectoryNNEnhanced) calculateTrajectoryComplexity(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 2 {
		return 0.0
	}

	totalDist := t.calculateTotalDistance(trajectory)

	dx := trajectory[len(trajectory)-1].X - trajectory[0].X
	dy := trajectory[len(trajectory)-1].Y - trajectory[0].Y
	directDist := math.Sqrt(dx*dx + dy*dy)

	if directDist == 0 {
		return 0.0
	}

	complexity := totalDist / directDist

	directions := t.calculateDirections(trajectory)
	curvatures := t.calculateCurvatures(trajectory)

	directionChanges := 0
	for i := 1; i < len(directions); i++ {
		change := math.Abs(directions[i] - directions[i-1])
		if change > 0.1 {
			directionChanges++
		}
	}

	complexity += float64(directionChanges) * 0.5

	avgCurvature := 0.0
	if len(curvatures) > 0 {
		avgCurvature = t.calculateAverage(curvatures)
	}
	complexity += avgCurvature * float64(len(trajectory))

	return math.Min(10.0, complexity)
}

func (t *TrajectoryNNEnhanced) detectCopiedPattern(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 10 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 4 {
		return 0.0
	}

	windowSize := len(speeds) / 4
	if windowSize < 2 {
		windowSize = 2
	}

	maxCorrelation := 0.0

	for offset := 1; offset < windowSize; offset++ {
		correlation := 0.0
		count := 0

		for i := 0; i+offset < len(speeds); i++ {
			diff := math.Abs(speeds[i] - speeds[i+offset])
			correlation += 1.0 / (diff + 0.001)
			count++
		}

		if count > 0 {
			correlation /= float64(count)
			if correlation > maxCorrelation {
				maxCorrelation = correlation
			}
		}
	}

	normalizedCorrelation := maxCorrelation / (t.calculateAverage(speeds) + 0.001)

	if normalizedCorrelation > 0.9 {
		return 1.0
	}

	return math.Min(1.0, normalizedCorrelation)
}

func (t *TrajectoryNNEnhanced) detectRepeatedPattern(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 8 {
		return 0.0
	}

	speeds := t.calculateSpeeds(trajectory)
	if len(speeds) < 4 {
		return 0.0
	}

	patternLengths := []int{2, 3, 4, 5}
	maxSimilarity := 0.0

	for _, patternLen := range patternLengths {
		if patternLen*2 > len(speeds) {
			continue
		}

		pattern1 := speeds[:patternLen]
		pattern2 := speeds[patternLen : patternLen*2]

		similarity := t.calculateSequenceSimilarity(pattern1, pattern2)

		if similarity > maxSimilarity {
			maxSimilarity = similarity
		}
	}

	if maxSimilarity > 0.95 {
		return 1.0
	}

	return maxSimilarity
}

func (t *TrajectoryNNEnhanced) calculateSequenceSimilarity(seq1, seq2 []float64) float64 {
	if len(seq1) != len(seq2) || len(seq1) == 0 {
		return 0.0
	}

	sumDiff := 0.0
	for i := range seq1 {
		sumDiff += math.Abs(seq1[i] - seq2[i])
	}

	avg1 := t.calculateAverage(seq1)
	avg2 := t.calculateAverage(seq2)

	if avg1 == 0 && avg2 == 0 {
		return 1.0
	}

	normalizedDiff := sumDiff / float64(len(seq1)) / ((avg1+avg2)/2.0+0.001)

	return 1.0 - math.Min(1.0, normalizedDiff)
}

func (t *TrajectoryNNEnhanced) calculateInterPointConsistency(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 3 {
		return 0.0
	}

	segments := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		segment := math.Sqrt(dx*dx + dy*dy)
		segments = append(segments, segment)
	}

	if len(segments) == 0 {
		return 0.0
	}

	avgSegment := t.calculateAverage(segments)
	variance := t.calculateVarianceFromValues(segments)

	if avgSegment == 0 {
		return 0.0
	}

	coefficientOfVariation := math.Sqrt(variance) / avgSegment

	return 1.0 / (1.0 + coefficientOfVariation)
}

func (t *TrajectoryNNEnhanced) calculateSegmentRegularity(trajectory []TrajectoryNNPoint) float64 {
	if len(trajectory) < 4 {
		return 0.0
	}

	segments := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		segment := math.Sqrt(dx*dx + dy*dy)
		segments = append(segments, segment)
	}

	if len(segments) < 2 {
		return 0.0
	}

	timeDiffs := []float64{}
	for i := 1; i < len(trajectory); i++ {
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			timeDiffs = append(timeDiffs, dt)
		}
	}

	if len(timeDiffs) != len(segments) {
		return 0.0
	}

	speedRatio := 0.0
	count := 0

	for i := range segments {
		if timeDiffs[i] > 0 {
			speed := segments[i] / timeDiffs[i]
			if i > 0 && timeDiffs[i-1] > 0 {
				prevSpeed := segments[i-1] / timeDiffs[i-1]
				ratio := speed / (prevSpeed + 0.001)
				if ratio > 0.5 && ratio < 2.0 {
					speedRatio += ratio
					count++
				}
			}
		}
	}

	if count == 0 {
		return 0.0
	}

	avgRatio := speedRatio / float64(count)
	regularity := 1.0 - math.Abs(1.0-avgRatio)

	return math.Max(0, math.Min(1, regularity))
}

func (t *TrajectoryNNEnhanced) calculateAverageSpeed(trajectory []TrajectoryNNPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateAverage(speeds)
}

func (t *TrajectoryNNEnhanced) calculateMaxSpeed(trajectory []TrajectoryNNPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateMax(speeds)
}

func (t *TrajectoryNNEnhanced) calculateMinSpeed(trajectory []TrajectoryNNPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateMin(speeds)
}

func (t *TrajectoryNNEnhanced) calculateSpeedVariance(trajectory []TrajectoryNNPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateVarianceFromValues(speeds)
}

type TrajectoryNNPoint struct {
	X         float64
	Y         float64
	Timestamp int64
}
