package service

import (
	"fmt"
	"math"
	"math/rand"
	"time"
)

type TrajectoryNeuralNetwork struct {
	inputSize    int
	hiddenSize   int
	outputSize   int
	weights      [][]float64
	biases       []float64
	learningRate float64
}

func NewTrajectoryNeuralNetwork(inputSize, hiddenSize, outputSize int) *TrajectoryNeuralNetwork {
	nn := &TrajectoryNeuralNetwork{
		inputSize:    inputSize,
		hiddenSize:   hiddenSize,
		outputSize:   outputSize,
		weights:      make([][]float64, 0),
		biases:       make([]float64, 0),
		learningRate: 0.01,
	}

	rand.Seed(time.Now().UnixNano())

	totalWeights := (inputSize * hiddenSize) + (hiddenSize * outputSize)
	weights := make([]float64, totalWeights)
	for i := range weights {
		weights[i] = (rand.Float64() - 0.5) * 2 * 0.1
	}

	weightIdx := 0
	hiddenWeights := make([]float64, inputSize*hiddenSize)
	for i := 0; i < inputSize; i++ {
		for j := 0; j < hiddenSize; j++ {
			hiddenWeights[i*hiddenSize+j] = weights[weightIdx]
			weightIdx++
		}
	}
	nn.weights = append(nn.weights, hiddenWeights)

	outputWeights := make([]float64, hiddenSize*outputSize)
	for i := 0; i < hiddenSize; i++ {
		for j := 0; j < outputSize; j++ {
			outputWeights[i*outputSize+j] = weights[weightIdx]
			weightIdx++
		}
	}
	nn.weights = append(nn.weights, outputWeights)

	nn.biases = make([]float64, hiddenSize+outputSize)
	for i := range nn.biases {
		nn.biases[i] = (rand.Float64() - 0.5) * 0.1
	}

	return nn
}

func (nn *TrajectoryNeuralNetwork) Forward(input []float64) []float64 {
	hiddenLayer := make([]float64, nn.hiddenSize)
	for j := 0; j < nn.hiddenSize; j++ {
		sum := nn.biases[j]
		for i := 0; i < nn.inputSize; i++ {
			sum += input[i] * nn.weights[0][i*nn.hiddenSize+j]
		}
		hiddenLayer[j] = nn.relu(sum)
	}

	outputLayer := make([]float64, nn.outputSize)
	for j := 0; j < nn.outputSize; j++ {
		sum := nn.biases[nn.hiddenSize+j]
		for i := 0; i < nn.hiddenSize; i++ {
			sum += hiddenLayer[i] * nn.weights[1][i*nn.outputSize+j]
		}
		outputLayer[j] = nn.sigmoid(sum)
	}

	return outputLayer
}

func (nn *TrajectoryNeuralNetwork) relu(x float64) float64 {
	if x > 0 {
		return x
	}
	return 0
}

func (nn *TrajectoryNeuralNetwork) sigmoid(x float64) float64 {
	return 1.0 / (1.0 + math.Exp(-x))
}

type TrajectoryFeatureExtractor struct{}

func NewTrajectoryFeatureExtractor() *TrajectoryFeatureExtractor {
	return &TrajectoryFeatureExtractor{}
}

type ExtractedFeatures struct {
	BasicFeatures    []float64
	AdvancedFeatures []float64
	AllFeatures      []float64
}

func (tfe *TrajectoryFeatureExtractor) ExtractFeatures(trajectory []SliderPoint) *ExtractedFeatures {
	features := &ExtractedFeatures{
		BasicFeatures:    make([]float64, 0),
		AdvancedFeatures: make([]float64, 0),
	}

	basicFeats := tfe.extractBasicFeatures(trajectory)
	features.BasicFeatures = basicFeats

	advancedFeats := tfe.extractAdvancedFeatures(trajectory)
	features.AdvancedFeatures = advancedFeats

	features.AllFeatures = make([]float64, 0, len(basicFeats)+len(advancedFeats))
	features.AllFeatures = append(features.AllFeatures, basicFeats...)
	features.AllFeatures = append(features.AllFeatures, advancedFeats...)

	return features
}

func (tfe *TrajectoryFeatureExtractor) extractBasicFeatures(trajectory []SliderPoint) []float64 {
	features := make([]float64, 15)

	if len(trajectory) < 2 {
		return features
	}

	totalDistance := 0.0
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	features[0] = math.Min(totalDistance/1000, 1.0)

	startX := float64(trajectory[0].X)
	endX := float64(trajectory[len(trajectory)-1].X)
	directDistance := math.Abs(endX - startX)
	features[1] = math.Min(directDistance/500, 1.0)

	if totalDistance > 0 {
		features[2] = directDistance / totalDistance
	}

	speeds := tfe.extractSpeeds(trajectory)
	if len(speeds) > 0 {
		mean := tfe.mean(speeds)
		features[3] = math.Min(mean/2000, 1.0)

		maxSpeed := tfe.max(speeds)
		features[4] = math.Min(maxSpeed/3000, 1.0)

		variance := tfe.variance(speeds)
		features[5] = math.Min(variance, 1.0)
	}

	duration := float64(trajectory[len(trajectory)-1].Timestamp - trajectory[0].Timestamp)
	features[6] = math.Min(duration/10000, 1.0)

	directionChanges := tfe.countDirectionChanges(trajectory)
	features[7] = math.Min(float64(directionChanges)/20, 1.0)

	microCorrections := tfe.countMicroCorrections(trajectory)
	features[8] = math.Min(float64(microCorrections)/30, 1.0)

	pauseCount, pauseDuration := tfe.countPauses(trajectory)
	features[9] = math.Min(float64(pauseCount)/20, 1.0)
	features[10] = math.Min(pauseDuration/1000, 1.0)

	backtrackCount, backtrackDistance := tfe.countBacktrack(trajectory)
	features[11] = math.Min(float64(backtrackCount)/10, 1.0)
	features[12] = math.Min(backtrackDistance/100, 1.0)

	features[13] = float64(len(trajectory)) / 100.0
	features[14] = math.Min(float64(trajectory[0].Timestamp)/5000, 1.0)

	return features
}

func (tfe *TrajectoryFeatureExtractor) extractAdvancedFeatures(trajectory []SliderPoint) []float64 {
	features := make([]float64, 20)

	if len(trajectory) < 3 {
		return features
	}

	accelerations := tfe.extractAccelerations(trajectory)
	if len(accelerations) > 0 {
		features[0] = math.Min(math.Abs(tfe.mean(accelerations)), 1.0)
		features[1] = math.Min(tfe.variance(accelerations), 1.0)
		features[2] = math.Min(math.Abs(tfe.max(accelerations)), 1.0)

		posCount := 0
		for _, acc := range accelerations {
			if acc > 0 {
				posCount++
			}
		}
		features[3] = float64(posCount) / float64(len(accelerations))
	}

	curvatures := tfe.extractCurvatures(trajectory)
	if len(curvatures) > 0 {
		features[4] = math.Min(tfe.mean(curvatures), 1.0)
		features[5] = math.Min(tfe.variance(curvatures), 1.0)

		significantCount := 0
		for _, c := range curvatures {
			if c > 0.1 {
				significantCount++
			}
		}
		features[6] = float64(significantCount) / float64(len(curvatures))
	}

	features[7] = tfe.calculateJitter(trajectory)

	features[8] = tfe.calculateSmoothness(trajectory)

	features[9] = tfe.calculateEntropy(trajectory)

	features[10] = tfe.calculateFractalDimension(trajectory)

	speeds := tfe.extractSpeeds(trajectory)
	if len(speeds) > 0 {
		features[11] = tfe.calculateSkewness(speeds)
		features[12] = tfe.calculateKurtosis(speeds)
		features[13] = (tfe.max(speeds) - tfe.min(speeds)) / 2000.0
	}

	features[14] = tfe.calculateEndBehavior(trajectory)

	features[15] = tfe.calculateWaveletEnergy(trajectory)

	features[16] = tfe.calculateFourierEntropy(trajectory)

	features[17] = tfe.calculateVelocityProfile(trajectory)

	features[18] = tfe.calculateAccelerationProfile(trajectory)

	features[19] = tfe.calculateHumanLikeness(features)

	return features
}

func (tfe *TrajectoryFeatureExtractor) extractSpeeds(trajectory []SliderPoint) []float64 {
	speeds := make([]float64, 0)
	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)
		if dt > 0 {
			speeds = append(speeds, distance/dt*1000)
		}
	}
	return speeds
}

func (tfe *TrajectoryFeatureExtractor) extractAccelerations(trajectory []SliderPoint) []float64 {
	speeds := tfe.extractSpeeds(trajectory)
	accelerations := make([]float64, 0)
	for i := 2; i < len(speeds); i++ {
		dt := float64(trajectory[i+1].Timestamp-trajectory[i-1].Timestamp) / 2
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}
	return accelerations
}

func (tfe *TrajectoryFeatureExtractor) extractCurvatures(trajectory []SliderPoint) []float64 {
	curvatures := make([]float64, 0)
	for i := 1; i < len(trajectory)-1; i++ {
		v1x := float64(trajectory[i].X - trajectory[i-1].X)
		v1y := float64(trajectory[i].Y - trajectory[i-1].Y)
		v2x := float64(trajectory[i+1].X - trajectory[i].X)
		v2y := float64(trajectory[i+1].Y - trajectory[i].Y)

		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			dot := v1x*v2x + v1y*v2y
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			curvatures = append(curvatures, math.Abs(angle))
		}
	}
	return curvatures
}

func (tfe *TrajectoryFeatureExtractor) countDirectionChanges(trajectory []SliderPoint) int {
	if len(trajectory) < 3 {
		return 0
	}

	changes := 0
	var prevAngle float64

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
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

func (tfe *TrajectoryFeatureExtractor) countMicroCorrections(trajectory []SliderPoint) int {
	if len(trajectory) < 3 {
		return 0
	}

	corrections := 0
	for i := 2; i < len(trajectory); i++ {
		dx1 := float64(trajectory[i-1].X - trajectory[i-2].X)
		dy1 := float64(trajectory[i-1].Y - trajectory[i-2].Y)
		dx2 := float64(trajectory[i].X - trajectory[i-1].X)
		dy2 := float64(trajectory[i].Y - trajectory[i-1].Y)

		dot := dx1*dx2 + dy1*dy2
		mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if mag1 > 0 && mag2 > 0 {
			cosAngle := dot / (mag1 * mag2)
			if cosAngle < 0.9 && cosAngle > -0.9 {
				angle := math.Acos(cosAngle)
				if angle > 0.1 && angle < 0.5 {
					corrections++
				}
			}
		}
	}

	return corrections
}

func (tfe *TrajectoryFeatureExtractor) countPauses(trajectory []SliderPoint) (int, float64) {
	if len(trajectory) < 2 {
		return 0, 0
	}

	pauses := 0
	totalDuration := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(trajectory[i].Timestamp - trajectory[i-1].Timestamp)

		if distance < 3 && dt > 100 {
			pauses++
			totalDuration += dt
		}
	}

	return pauses, totalDuration
}

func (tfe *TrajectoryFeatureExtractor) countBacktrack(trajectory []SliderPoint) (int, float64) {
	if len(trajectory) < 2 {
		return 0, 0
	}

	backtracks := 0
	backtrackDistance := 0.0
	maxX := trajectory[0].X

	for i := 1; i < len(trajectory); i++ {
		if trajectory[i].X > maxX {
			maxX = trajectory[i].X
		} else if maxX-trajectory[i].X > 5 {
			backtracks++
			backtrackDistance += float64(maxX - trajectory[i].X)
		}
	}

	return backtracks, backtrackDistance
}

func (tfe *TrajectoryFeatureExtractor) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (tfe *TrajectoryFeatureExtractor) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := tfe.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += (v - mean) * (v - mean)
	}
	return sum / float64(len(values))
}

func (tfe *TrajectoryFeatureExtractor) max(values []float64) float64 {
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

func (tfe *TrajectoryFeatureExtractor) min(values []float64) float64 {
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

func (tfe *TrajectoryFeatureExtractor) calculateJitter(trajectory []SliderPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	smoothed := tfe.smoothTrajectory(trajectory, 3)
	totalJitter := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx1 := float64(trajectory[i].X - trajectory[i-1].X)
		dy1 := float64(trajectory[i].Y - trajectory[i-1].Y)
		dx2 := float64(smoothed[i].X - smoothed[i-1].X)
		dy2 := float64(smoothed[i].Y - smoothed[i-1].Y)

		distance1 := math.Sqrt(dx1*dx1 + dy1*dy1)
		distance2 := math.Sqrt(dx2*dx2 + dy2*dy2)

		if distance1 > 0 {
			totalJitter += math.Abs(distance1-distance2) / distance1
		}
	}

	return math.Min((totalJitter/float64(len(trajectory)-1))*10, 1.0)
}

func (tfe *TrajectoryFeatureExtractor) smoothTrajectory(trajectory []SliderPoint, windowSize int) []SliderPoint {
	if len(trajectory) < windowSize {
		return trajectory
	}

	if windowSize%2 == 0 {
		windowSize++
	}

	halfWindow := windowSize / 2
	smoothed := make([]SliderPoint, len(trajectory))

	for i := range trajectory {
		start := i - halfWindow
		end := i + halfWindow

		if start < 0 {
			start = 0
		}
		if end >= len(trajectory) {
			end = len(trajectory) - 1
		}

		sumX := 0
		sumY := 0
		count := 0

		for j := start; j <= end; j++ {
			sumX += trajectory[j].X
			sumY += trajectory[j].Y
			count++
		}

		smoothed[i] = trajectory[i]
		smoothed[i].X = sumX / count
		smoothed[i].Y = sumY / count
	}

	return smoothed
}

func (tfe *TrajectoryFeatureExtractor) calculateSmoothness(trajectory []SliderPoint) float64 {
	if len(trajectory) < 3 {
		return 0
	}

	totalAngleChange := 0.0
	count := 0

	for i := 1; i < len(trajectory)-1; i++ {
		v1x := float64(trajectory[i].X - trajectory[i-1].X)
		v1y := float64(trajectory[i].Y - trajectory[i-1].Y)
		v2x := float64(trajectory[i+1].X - trajectory[i].X)
		v2y := float64(trajectory[i+1].Y - trajectory[i].Y)

		mag1 := math.Sqrt(v1x*v1x + v1y*v1y)
		mag2 := math.Sqrt(v2x*v2x + v2y*v2y)

		if mag1 > 0 && mag2 > 0 {
			dot := v1x*v2x + v1y*v2y
			cosAngle := dot / (mag1 * mag2)
			if cosAngle > 1 {
				cosAngle = 1
			}
			if cosAngle < -1 {
				cosAngle = -1
			}
			angle := math.Acos(cosAngle)
			totalAngleChange += angle
			count++
		}
	}

	if count == 0 {
		return 1.0
	}

	avgAngleChange := totalAngleChange / float64(count)
	return 1.0 - math.Min(avgAngleChange/math.Pi, 1.0)
}

func (tfe *TrajectoryFeatureExtractor) calculateEntropy(trajectory []SliderPoint) float64 {
	if len(trajectory) < 2 {
		return 0
	}

	buckets := 20
	bucketCounts := make([]int, buckets)
	totalDistance := 0.0

	for i := 1; i < len(trajectory); i++ {
		dx := float64(trajectory[i].X - trajectory[i-1].X)
		dy := float64(trajectory[i].Y - trajectory[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance
	}

	if totalDistance == 0 {
		return 0
	}

	for i := 1; i < len(trajectory); i++ {
		progress := 0.0
		dist := 0.0

		for j := 1; j <= i; j++ {
			dx := float64(trajectory[j].X - trajectory[j-1].X)
			dy := float64(trajectory[j].Y - trajectory[j-1].Y)
			dist += math.Sqrt(dx*dx + dy*dy)
		}

		progress = dist / totalDistance
		bucket := int(progress * float64(buckets))
		if bucket >= buckets {
			bucket = buckets - 1
		}
		bucketCounts[bucket]++
	}

	entropy := 0.0
	for _, count := range bucketCounts {
		if count > 0 {
			p := float64(count) / float64(len(trajectory))
			entropy -= p * math.Log2(p)
		}
	}

	return entropy / 4.0
}

func (tfe *TrajectoryFeatureExtractor) calculateFractalDimension(trajectory []SliderPoint) float64 {
	if len(trajectory) < 10 {
		return 1.0
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

	width := maxX - minX
	height := maxY - minY

	if width == 0 && height == 0 {
		return 1.0
	}

	maxScale := 5
	logScales := make([]float64, maxScale)
	logCounts := make([]float64, maxScale)

	for scale := 0; scale < maxScale; scale++ {
		boxSize := int(math.Pow(2, float64(maxScale-scale)))
		grid := make(map[string]bool)

		for _, p := range trajectory {
			gx := (p.X - minX) / boxSize
			gy := (p.Y - minY) / boxSize
			key := fmt.Sprintf("%d,%d", gx, gy)
			grid[key] = true
		}

		logScales[scale] = math.Log(1.0 / float64(boxSize))
		logCounts[scale] = math.Log(float64(len(grid)))
	}

	return math.Min(tfe.linearRegression(logScales, logCounts)/2, 1.0)
}

func (tfe *TrajectoryFeatureExtractor) linearRegression(x, y []float64) float64 {
	n := len(x)
	if n != len(y) || n < 2 {
		return 1.0
	}

	sumX, sumY, sumXY, sumX2 := 0.0, 0.0, 0.0, 0.0
	for i := 0; i < n; i++ {
		sumX += x[i]
		sumY += y[i]
		sumXY += x[i] * y[i]
		sumX2 += x[i] * x[i]
	}

	denominator := float64(n)*sumX2 - sumX*sumX
	if denominator == 0 {
		return 1.0
	}

	return (float64(n)*sumXY - sumX*sumY) / denominator
}

func (tfe *TrajectoryFeatureExtractor) calculateSkewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := tfe.mean(values)
	stdDev := math.Sqrt(tfe.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}

func (tfe *TrajectoryFeatureExtractor) calculateKurtosis(values []float64) float64 {
	if len(values) < 4 {
		return 0
	}
	mean := tfe.mean(values)
	stdDev := math.Sqrt(tfe.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 4)
	}
	return (sum / float64(len(values))) - 3
}

func (tfe *TrajectoryFeatureExtractor) calculateEndBehavior(trajectory []SliderPoint) float64 {
	if len(trajectory) < 5 {
		return 0.5
	}

	lastPoints := trajectory[len(trajectory)-5:]
	totalDist := 0.0

	for i := 1; i < len(lastPoints); i++ {
		dx := float64(lastPoints[i].X - lastPoints[i-1].X)
		dy := float64(lastPoints[i].Y - lastPoints[i-1].Y)
		totalDist += math.Sqrt(dx*dx + dy*dy)
	}

	startX := float64(lastPoints[0].X)
	startY := float64(lastPoints[0].Y)
	endX := float64(lastPoints[len(lastPoints)-1].X)
	endY := float64(lastPoints[len(lastPoints)-1].Y)
	netDist := math.Sqrt((endX-startX)*(endX-startX) + (endY-startY)*(endY-startY))

	if totalDist == 0 {
		return 0.5
	}

	return netDist / totalDist
}

func (tfe *TrajectoryFeatureExtractor) calculateWaveletEnergy(trajectory []SliderPoint) float64 {
	if len(trajectory) < 4 {
		return 0
	}

	levels := 3
	energy := 0.0

	for level := 0; level < levels && len(trajectory) > 1; level++ {
		for i := 0; i < len(trajectory)-1; i += 2 {
			detail := float64(trajectory[i].X - trajectory[i+1].X)
			energy += detail * detail
		}

		newTraj := make([]SliderPoint, len(trajectory)/2)
		for i := 0; i < len(newTraj); i++ {
			newTraj[i] = SliderPoint{
				X:         (trajectory[i*2].X + trajectory[i*2+1].X) / 2,
				Y:         trajectory[i].Y,
				Timestamp: trajectory[i].Timestamp,
			}
		}
		trajectory = newTraj
	}

	return math.Min(energy/10000, 1.0)
}

func (tfe *TrajectoryFeatureExtractor) calculateFourierEntropy(trajectory []SliderPoint) float64 {
	if len(trajectory) < 8 {
		return 0
	}

	n := len(trajectory)
	for n&(n-1) != 0 {
		n--
	}
	if n < 8 {
		return 0
	}

	x := make([]float64, n)
	for i := 0; i < n; i++ {
		x[i] = float64(trajectory[i].X)
	}

	fft := tfe.fft(x)
	totalEnergy := 0.0
	for i := 1; i < n/2; i++ {
		mag := real(fft[i])*real(fft[i]) + imag(fft[i])*imag(fft[i])
		totalEnergy += mag
	}

	if totalEnergy == 0 {
		return 0
	}

	entropy := 0.0
	for i := 1; i < n/2; i++ {
		mag := real(fft[i])*real(fft[i]) + imag(fft[i])*imag(fft[i])
		if mag > 0 {
			p := mag / totalEnergy
			entropy -= p * math.Log2(p)
		}
	}

	return entropy / 4.0
}

func (tfe *TrajectoryFeatureExtractor) fft(x []float64) []complex128 {
	n := len(x)
	if n <= 1 {
		result := make([]complex128, n)
		for i, val := range x {
			result[i] = complex(val, 0)
		}
		return result
	}

	even := make([]float64, n/2)
	odd := make([]float64, n/2)
	for i := 0; i < n/2; i++ {
		even[i] = x[2*i]
		odd[i] = x[2*i+1]
	}

	fftEven := tfe.fft(even)
	fftOdd := tfe.fft(odd)

	result := make([]complex128, n)
	for k := 0; k < n/2; k++ {
		theta := -2 * math.Pi * float64(k) / float64(n)
		t := complex(math.Cos(theta), math.Sin(theta)) * fftOdd[k]
		result[k] = complex(real(fftEven[k])+real(t), imag(fftEven[k])+imag(t))
		result[k+n/2] = complex(real(fftEven[k])-real(t), imag(fftEven[k])-imag(t))
	}

	return result
}

func (tfe *TrajectoryFeatureExtractor) calculateVelocityProfile(trajectory []SliderPoint) float64 {
	if len(trajectory) < 10 {
		return 0.5
	}

	segments := 10
	segmentSize := len(trajectory) / segments

	profileVariance := 0.0
	segmentSpeeds := make([]float64, segments)

	for i := 0; i < segments; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == segments-1 {
			end = len(trajectory)
		}

		segment := trajectory[start:end]
		speeds := tfe.extractSpeeds(segment)
		if len(speeds) > 0 {
			segmentSpeeds[i] = tfe.mean(speeds)
		}
	}

	mean := tfe.mean(segmentSpeeds)
	for _, speed := range segmentSpeeds {
		profileVariance += (speed - mean) * (speed - mean)
	}

	return math.Min(profileVariance/1000000, 1.0)
}

func (tfe *TrajectoryFeatureExtractor) calculateAccelerationProfile(trajectory []SliderPoint) float64 {
	if len(trajectory) < 10 {
		return 0.5
	}

	segments := 10
	segmentSize := len(trajectory) / segments

	profileVariance := 0.0
	segmentAccels := make([]float64, segments)

	for i := 0; i < segments; i++ {
		start := i * segmentSize
		end := start + segmentSize
		if i == segments-1 {
			end = len(trajectory)
		}

		segment := trajectory[start:end]
		accelerations := tfe.extractAccelerations(segment)
		if len(accelerations) > 0 {
			segmentAccels[i] = tfe.mean(accelerations)
		}
	}

	mean := tfe.mean(segmentAccels)
	for _, accel := range segmentAccels {
		profileVariance += (accel - mean) * (accel - mean)
	}

	return math.Min(profileVariance, 1.0)
}

func (tfe *TrajectoryFeatureExtractor) calculateHumanLikeness(features []float64) float64 {
	if len(features) < 20 {
		return 0.5
	}

	score := 0.5

	if features[2] > 0.7 && features[2] < 0.98 {
		score += 0.15
	} else if features[2] >= 0.98 {
		score -= 0.3
	}

	speedConsistency := 1.0 - features[5]
	if speedConsistency > 0.3 && speedConsistency < 0.9 {
		score += 0.15
	} else if speedConsistency <= 0.1 {
		score -= 0.2
	}

	if features[8] > 0.1 && features[8] < 0.5 {
		score += 0.1
	}

	if features[9] > 0 && features[9] < 10 {
		score += 0.1
	}

	if features[4] > 0.01 && features[4] < 0.5 {
		score += 0.1
	}

	if features[7] > 0.01 && features[7] < 0.3 {
		score += 0.1
	}

	if features[8] > 0.3 && features[8] < 0.9 {
		score += 0.1
	}

	if features[11] > 0 && features[11] < 5 {
		score += 0.05
	}

	return math.Max(0, math.Min(1, score))
}

type TrajectoryClassifier struct {
	neuralNetwork    *TrajectoryNeuralNetwork
	featureExtractor *TrajectoryFeatureExtractor
}

func NewTrajectoryClassifier() *TrajectoryClassifier {
	return &TrajectoryClassifier{
		neuralNetwork:    NewTrajectoryNeuralNetwork(35, 20, 1),
		featureExtractor: NewTrajectoryFeatureExtractor(),
	}
}

func (tc *TrajectoryClassifier) Classify(trajectory []SliderPoint) (float64, string) {
	features := tc.featureExtractor.ExtractFeatures(trajectory)

	input := features.AllFeatures
	if len(input) < 35 {
		padding := make([]float64, 35-len(input))
		input = append(input, padding...)
	} else if len(input) > 35 {
		input = input[:35]
	}

	output := tc.neuralNetwork.Forward(input)

	botProbability := output[0]

	category := "normal"
	if botProbability > 0.7 {
		category = "high_risk"
	} else if botProbability > 0.5 {
		category = "medium_risk"
	} else if botProbability > 0.3 {
		category = "low_risk"
	}

	return botProbability, category
}

func (tc *TrajectoryClassifier) GetDetailedAnalysis(trajectory []SliderPoint) map[string]interface{} {
	features := tc.featureExtractor.ExtractFeatures(trajectory)

	input := features.AllFeatures
	if len(input) < 35 {
		padding := make([]float64, 35-len(input))
		input = append(input, padding...)
	} else if len(input) > 35 {
		input = input[:35]
	}

	output := tc.neuralNetwork.Forward(input)

	analysis := make(map[string]interface{})
	analysis["bot_probability"] = output[0]
	analysis["human_probability"] = 1.0 - output[0]

	analysis["basic_features"] = features.BasicFeatures
	analysis["advanced_features"] = features.AdvancedFeatures

	basicNames := []string{
		"total_distance", "direct_distance", "path_efficiency",
		"average_speed", "max_speed", "speed_variance",
		"duration", "direction_changes", "micro_corrections",
		"pause_count", "pause_duration", "backtrack_count",
		"backtrack_distance", "point_count", "start_delay",
	}

	advancedNames := []string{
		"acceleration_mean", "acceleration_variance", "acceleration_max",
		"acceleration_pos_ratio", "curvature_mean", "curvature_variance",
		"curvature_significant_ratio", "jitter", "smoothness",
		"entropy", "fractal_dimension", "speed_skewness",
		"speed_kurtosis", "speed_range", "end_behavior",
		"wavelet_energy", "fourier_entropy", "velocity_profile",
		"acceleration_profile", "human_likeness",
	}

	basicFeats := make(map[string]float64)
	for i, name := range basicNames {
		if i < len(features.BasicFeatures) {
			basicFeats[name] = features.BasicFeatures[i]
		}
	}
	analysis["basic_features_named"] = basicFeats

	advancedFeats := make(map[string]float64)
	for i, name := range advancedNames {
		if i < len(features.AdvancedFeatures) {
			advancedFeats[name] = features.AdvancedFeatures[i]
		}
	}
	analysis["advanced_features_named"] = advancedFeats

	category := "normal"
	if output[0] > 0.7 {
		category = "high_risk"
	} else if output[0] > 0.5 {
		category = "medium_risk"
	} else if output[0] > 0.3 {
		category = "low_risk"
	}
	analysis["category"] = category

	return analysis
}
