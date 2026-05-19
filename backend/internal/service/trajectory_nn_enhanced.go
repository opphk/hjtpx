package service

import (
	"math"
	"sort"
)

type TrajectoryNNEnhanced struct {
	FeatureDim int
	HiddenDim  int
}

func NewTrajectoryNNEnhanced() *TrajectoryNNEnhanced {
	return &TrajectoryNNEnhanced{
		FeatureDim: 768,
		HiddenDim:  256,
	}
}

func (t *TrajectoryNNEnhanced) ExtractFeatures(trajectory []TrajectoryPoint) []float64 {
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

	return features
}

func (t *TrajectoryNNEnhanced) extractBasicFeatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) extractSpeedFeatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) extractDirectionFeatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) extractCurvatureFeatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) extractPositionFeatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) extractBehaviorFeatures(trajectory []TrajectoryPoint) []float64 {
	features := make([]float64, 192)

	pattern := t.classifyBehaviorPattern(trajectory)
	copy(features[0:32], pattern)

	anomalies := t.detectAnomalies(trajectory)
	copy(features[32:64], anomalies)

	learningFeatures := t.extractLearningFeatures(trajectory)
	copy(features[64:160], learningFeatures)

	return features
}

func (t *TrajectoryNNEnhanced) calculateTotalDistance(trajectory []TrajectoryPoint) float64 {
	total := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := trajectory[i].X - trajectory[i-1].X
		dy := trajectory[i].Y - trajectory[i-1].Y
		total += math.Sqrt(dx*dx + dy*dy)
	}
	return total
}

func (t *TrajectoryNNEnhanced) calculateSpeeds(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) calculateDirections(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) calculateCurvatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) calculatePositionHistogram(trajectory []TrajectoryPoint, isX bool, bins int) []float64 {
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

func (t *TrajectoryNNEnhanced) calculateCoverage(trajectory []TrajectoryPoint, gridSize int) []float64 {
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

func (t *TrajectoryNNEnhanced) classifyBehaviorPattern(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) detectAnomalies(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) extractLearningFeatures(trajectory []TrajectoryPoint) []float64 {
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

func (t *TrajectoryNNEnhanced) calculateAverageSpeed(trajectory []TrajectoryPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateAverage(speeds)
}

func (t *TrajectoryNNEnhanced) calculateMaxSpeed(trajectory []TrajectoryPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateMax(speeds)
}

func (t *TrajectoryNNEnhanced) calculateMinSpeed(trajectory []TrajectoryPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateMin(speeds)
}

func (t *TrajectoryNNEnhanced) calculateSpeedVariance(trajectory []TrajectoryPoint) float64 {
	speeds := t.calculateSpeeds(trajectory)
	return t.calculateVarianceFromValues(speeds)
}

type TrajectoryPoint struct {
	X         float64
	Y         float64
	Timestamp int64
}
