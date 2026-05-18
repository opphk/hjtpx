package trace

import (
	"math"
	"sort"

	"github.com/hjtpx/hjtpx/internal/model"
)

type DTWMatcher struct {
	extractor      *TraceExtractor
	multiScaleEnabled bool
	weightedEnabled bool
	windowSize     int
}

type DTWConfig struct {
	WindowSize     int
	MultiScale     bool
	Weighted       bool
	WeightVelocity float64
	WeightAccel    float64
	WeightDir      float64
	WeightCurv     float64
}

func NewDTWMatcher() *DTWMatcher {
	return &DTWMatcher{
		extractor:        NewTraceExtractor(),
		multiScaleEnabled: true,
		weightedEnabled:   true,
		windowSize:       10,
	}
}

func (d *DTWMatcher) SetConfig(config DTWConfig) {
	d.windowSize = config.WindowSize
	d.multiScaleEnabled = config.MultiScale
	d.weightedEnabled = config.Weighted
}

func (d *DTWMatcher) CalculateDTWDistance(trace1, trace2 *model.TraceData) float64 {
	if trace1 == nil || trace2 == nil || len(trace1.Points) < 2 || len(trace2.Points) < 2 {
		return math.MaxFloat64
	}

	if d.multiScaleEnabled && len(trace1.Points) > 20 && len(trace2.Points) > 20 {
		return d.calculateMultiScaleDTW(trace1, trace2)
	}

	n := len(trace1.Points)
	m := len(trace2.Points)

	distMatrix := make([][]float64, n)
	for i := range distMatrix {
		distMatrix[i] = make([]float64, m)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			distMatrix[i][j] = d.calculatePointDistance(trace1.Points[i], trace2.Points[j])
		}
	}

	dtwMatrix := d.computeDTWMatrixWithWindow(distMatrix, n, m)

	return dtwMatrix[n-1][m-1] / float64(n+m)
}

func (d *DTWMatcher) computeDTWMatrixWithWindow(distMatrix [][]float64, n, m int) [][]float64 {
	dtwMatrix := make([][]float64, n)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}

	dtwMatrix[0][0] = distMatrix[0][0]

	window := d.windowSize
	if window <= 0 {
		window = int(math.Max(float64(n), float64(m)))
	}

	for i := 1; i < n; i++ {
		for j := 1; j < m; j++ {
			if math.Abs(float64(i-j)) > float64(window) {
				continue
			}
			minPrev := math.Min(dtwMatrix[i-1][j], math.Min(dtwMatrix[i][j-1], dtwMatrix[i-1][j-1]))
			dtwMatrix[i][j] = distMatrix[i][j] + minPrev
		}
	}

	return dtwMatrix
}

func (d *DTWMatcher) calculateMultiScaleDTW(trace1, trace2 *model.TraceData) float64 {
	downsample1 := d.downsampleTrajectory(trace1, 2)
	downsample2 := d.downsampleTrajectory(trace2, 2)

	dtw1 := d.computeBasicDTW(trace1, trace2)
	dtw2 := d.computeBasicDTW(downsample1, downsample2)

	combinedScore := dtw1*0.7 + dtw2*0.3*2

	return combinedScore
}

func (d *DTWMatcher) downsampleTrajectory(trace *model.TraceData, factor int) *model.TraceData {
	if len(trace.Points) <= factor {
		return trace
	}

	downsampled := &model.TraceData{
		Points: make([]model.TracePoint, 0, len(trace.Points)/factor),
	}

	for i := 0; i < len(trace.Points); i += factor {
		downsampled.Points = append(downsampled.Points, trace.Points[i])
	}

	return downsampled
}

func (d *DTWMatcher) computeBasicDTW(trace1, trace2 *model.TraceData) float64 {
	n := len(trace1.Points)
	m := len(trace2.Points)

	if n < 2 || m < 2 {
		return math.MaxFloat64
	}

	distMatrix := make([][]float64, n)
	for i := range distMatrix {
		distMatrix[i] = make([]float64, m)
		for j := range distMatrix[i] {
			if d.weightedEnabled {
				distMatrix[i][j] = d.calculateWeightedPointDistance(trace1.Points[i], trace2.Points[j])
			} else {
				distMatrix[i][j] = d.calculatePointDistance(trace1.Points[i], trace2.Points[j])
			}
		}
	}

	dtwMatrix := make([][]float64, n)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}

	dtwMatrix[0][0] = distMatrix[0][0]

	for i := 1; i < n; i++ {
		dtwMatrix[i][0] = dtwMatrix[i-1][0] + distMatrix[i][0]
	}

	for j := 1; j < m; j++ {
		dtwMatrix[0][j] = dtwMatrix[0][j-1] + distMatrix[0][j]
	}

	for i := 1; i < n; i++ {
		for j := 1; j < m; j++ {
			minPrev := math.Min(dtwMatrix[i-1][j], math.Min(dtwMatrix[i][j-1], dtwMatrix[i-1][j-1]))
			dtwMatrix[i][j] = distMatrix[i][j] + minPrev
		}
	}

	return dtwMatrix[n-1][m-1] / float64(n+m)
}

func (d *DTWMatcher) calculateWeightedPointDistance(p1, p2 model.TracePoint) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	dt := float64(p1.Timestamp-p2.Timestamp) / 1000.0

	spatialDist := math.Sqrt(dx*dx + dy*dy)
	spatialWeight := 1.0
	temporalWeight := 0.3

	return math.Sqrt(spatialDist*spatialDist*spatialWeight +
		dt*dt*temporalWeight)
}

func (d *DTWMatcher) calculatePointDistance(p1, p2 model.TracePoint) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	dt := float64(p1.Timestamp - p2.Timestamp) / 1000.0

	return math.Sqrt(dx*dx + dy*dy + dt*dt)
}

func (d *DTWMatcher) CalculateDTWSimilarity(trace1, trace2 *model.TraceData) float64 {
	distance := d.CalculateDTWDistance(trace1, trace2)
	if distance == math.MaxFloat64 {
		return 0
	}

	maxPossible := float64(len(trace1.Points)+len(trace2.Points)) * 1000.0
	similarity := 1.0 - math.Min(distance/maxPossible, 1.0)

	return similarity
}

func (d *DTWMatcher) CalculateDTWDistanceWithFeatures(trace1, trace2 *model.TraceData) (float64, error) {
	features1, err := d.extractor.ExtractFeatures(trace1)
	if err != nil {
		return 0, err
	}

	features2, err := d.extractor.ExtractFeatures(trace2)
	if err != nil {
		return 0, err
	}

	return d.compareFeatures(features1, features2), nil
}

func (d *DTWMatcher) CalculateEnhancedDTWScore(trace1, trace2 *model.TraceData) map[string]float64 {
	results := make(map[string]float64)

	results["basic_dtw"] = d.CalculateDTWDistance(trace1, trace2)
	results["similarity"] = d.CalculateDTWSimilarity(trace1, trace2)

	featureDist, _ := d.CalculateDTWDistanceWithFeatures(trace1, trace2)
	results["feature_distance"] = featureDist

	results["speed_consistency"] = d.calculateSpeedConsistency(trace1, trace2)
	results["acceleration_pattern"] = d.calculateAccelerationPattern(trace1, trace2)
	results["direction_similarity"] = d.calculateDirectionSimilarity(trace1, trace2)

	velocityDTW := d.calculateVelocityDTW(trace1, trace2)
	results["velocity_dtw"] = velocityDTW

	results["combined_score"] = (results["basic_dtw"]*0.3 +
		(1-results["similarity"])*100*0.2 +
		featureDist*0.2 +
		(1-results["speed_consistency"])*0.15 +
		velocityDTW*0.15)

	return results
}

func (d *DTWMatcher) calculateSpeedConsistency(trace1, trace2 *model.TraceData) float64 {
	speeds1 := d.extractSpeedSequence(trace1)
	speeds2 := d.extractSpeedSequence(trace2)

	if len(speeds1) == 0 || len(speeds2) == 0 {
		return 0
	}

	mean1 := d.mean(speeds1)
	mean2 := d.mean(speeds2)

	if mean1 == 0 || mean2 == 0 {
		return 0
	}

	ratio := math.Min(mean1/mean2, mean2/mean1)

	return math.Max(0, ratio)
}

func (d *DTWMatcher) extractSpeedSequence(trace *model.TraceData) []float64 {
	if len(trace.Points) < 2 {
		return nil
	}

	speeds := make([]float64, len(trace.Points)-1)
	for i := 1; i < len(trace.Points); i++ {
		dx := trace.Points[i].X - trace.Points[i-1].X
		dy := trace.Points[i].Y - trace.Points[i-1].Y
		dt := float64(trace.Points[i].Timestamp-trace.Points[i-1].Timestamp) / 1000.0

		if dt > 0 {
			speeds[i-1] = math.Sqrt(dx*dx+dy*dy) / dt
		}
	}

	return speeds
}

func (d *DTWMatcher) calculateAccelerationPattern(trace1, trace2 *model.TraceData) float64 {
	accel1 := d.extractAccelerationSequence(trace1)
	accel2 := d.extractAccelerationSequence(trace2)

	if len(accel1) == 0 || len(accel2) == 0 {
		return 1.0
	}

	var sumDiff float64
	minLen := int(math.Min(float64(len(accel1)), float64(len(accel2))))
	sampleSize := 20
	step := max(1, minLen/sampleSize)

	count := 0
	for i := 0; i < minLen; i += step {
		diff := math.Abs(accel1[i] - accel2[i])
		sumDiff += diff
		count++
	}

	if count == 0 {
		return 1.0
	}

	avgDiff := sumDiff / float64(count)

	similarity := 1.0 / (1.0 + avgDiff/100.0)

	return similarity
}

func (d *DTWMatcher) extractAccelerationSequence(trace *model.TraceData) []float64 {
	speeds := d.extractSpeedSequence(trace)
	if len(speeds) < 2 {
		return nil
	}

	accelerations := make([]float64, len(speeds)-1)
	for i := 1; i < len(speeds); i++ {
		accelerations[i-1] = speeds[i] - speeds[i-1]
	}

	return accelerations
}

func (d *DTWMatcher) calculateDirectionSimilarity(trace1, trace2 *model.TraceData) float64 {
	dir1 := d.extractDirectionSequence(trace1)
	dir2 := d.extractDirectionSequence(trace2)

	if len(dir1) == 0 || len(dir2) == 0 {
		return 0
	}

	minLen := int(math.Min(float64(len(dir1)), float64(len(dir2))))
	sampleSize := 20
	step := max(1, minLen/sampleSize)

	var totalSim float64
	count := 0

	for i := 0; i < minLen; i += step {
		sim := 1.0 - math.Abs(dir1[i]-dir2[i])/math.Pi
		if sim < 0 {
			sim = 0
		}
		totalSim += sim
		count++
	}

	if count == 0 {
		return 0
	}

	return totalSim / float64(count)
}

func (d *DTWMatcher) extractDirectionSequence(trace *model.TraceData) []float64 {
	if len(trace.Points) < 2 {
		return nil
	}

	directions := make([]float64, len(trace.Points)-1)
	for i := 1; i < len(trace.Points); i++ {
		dx := trace.Points[i].X - trace.Points[i-1].X
		dy := trace.Points[i].Y - trace.Points[i-1].Y
		directions[i-1] = math.Atan2(float64(dy), float64(dx))
	}

	return directions
}

func (d *DTWMatcher) calculateVelocityDTW(trace1, trace2 *model.TraceData) float64 {
	speeds1 := d.extractSpeedSequence(trace1)
	speeds2 := d.extractSpeedSequence(trace2)

	if len(speeds1) < 2 || len(speeds2) < 2 {
		return math.MaxFloat64
	}

	n := len(speeds1)
	m := len(speeds2)

	dtwMatrix := make([][]float64, n)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}

	dtwMatrix[0][0] = math.Abs(speeds1[0] - speeds2[0])

	for i := 1; i < n; i++ {
		dtwMatrix[i][0] = dtwMatrix[i-1][0] + math.Abs(speeds1[i]-speeds2[0])
	}

	for j := 1; j < m; j++ {
		dtwMatrix[0][j] = dtwMatrix[0][j-1] + math.Abs(speeds1[0]-speeds2[j])
	}

	for i := 1; i < n; i++ {
		for j := 1; j < m; j++ {
			cost := math.Abs(speeds1[i] - speeds2[j])
			minPrev := math.Min(dtwMatrix[i-1][j], math.Min(dtwMatrix[i][j-1], dtwMatrix[i-1][j-1]))
			dtwMatrix[i][j] = cost + minPrev
		}
	}

	return dtwMatrix[n-1][m-1] / float64(n+m)
}

func (d *DTWMatcher) max3(a, b, c float64) float64 {
	return math.Max(a, math.Max(b, c))
}

func (d *DTWMatcher) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	var sum float64
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (d *DTWMatcher) compareFeatures(f1, f2 *model.TraceFeatures) float64 {
	var totalDiff float64
	count := 0

	if f1.AvgSpeed > 0 || f2.AvgSpeed > 0 {
		totalDiff += math.Abs(f1.AvgSpeed-f2.AvgSpeed) / d.max3(f1.AvgSpeed, f2.AvgSpeed, 1.0)
		count++
	}

	if f1.MaxSpeed > 0 || f2.MaxSpeed > 0 {
		totalDiff += math.Abs(f1.MaxSpeed-f2.MaxSpeed) / d.max3(f1.MaxSpeed, f2.MaxSpeed, 1.0)
		count++
	}

	if f1.SpeedVariance > 0 || f2.SpeedVariance > 0 {
		totalDiff += math.Abs(f1.SpeedVariance-f2.SpeedVariance) / d.max3(f1.SpeedVariance, f2.SpeedVariance, 1.0)
		count++
	}

	if f1.MaxAcceleration > 0 || f2.MaxAcceleration > 0 {
		totalDiff += math.Abs(f1.MaxAcceleration-f2.MaxAcceleration) / d.max3(f1.MaxAcceleration, f2.MaxAcceleration, 1.0)
		count++
	}

	if f1.AvgAcceleration > 0 || f2.AvgAcceleration > 0 {
		totalDiff += math.Abs(f1.AvgAcceleration-f2.AvgAcceleration) / d.max3(f1.AvgAcceleration, f2.AvgAcceleration, 1.0)
		count++
	}

	if f1.Smoothness > 0 || f2.Smoothness > 0 {
		totalDiff += math.Abs(f1.Smoothness-f2.Smoothness) / d.max3(f1.Smoothness, f2.Smoothness, 0.01)
		count++
	}

	if f1.PathRatio > 0 || f2.PathRatio > 0 {
		totalDiff += math.Abs(f1.PathRatio-f2.PathRatio) / d.max3(f1.PathRatio, f2.PathRatio, 1.0)
		count++
	}

	if f1.AvgCurvature > 0 || f2.AvgCurvature > 0 {
		totalDiff += math.Abs(f1.AvgCurvature-f2.AvgCurvature) / d.max3(f1.AvgCurvature, f2.AvgCurvature, 0.001)
		count++
	}

	if f1.JitterFrequency > 0 || f2.JitterFrequency > 0 {
		totalDiff += math.Abs(f1.JitterFrequency-f2.JitterFrequency) / d.max3(f1.JitterFrequency, f2.JitterFrequency, 0.01)
		count++
	}

	if f1.JitterAmplitude > 0 || f2.JitterAmplitude > 0 {
		totalDiff += math.Abs(f1.JitterAmplitude-f2.JitterAmplitude) / d.max3(f1.JitterAmplitude, f2.JitterAmplitude, 1.0)
		count++
	}

	if count == 0 {
		return 1.0
	}

	return totalDiff / float64(count)
}

func (d *DTWMatcher) MatchTraces(trace1, trace2 *model.TraceData) (float64, string) {
	distance := d.CalculateDTWDistance(trace1, trace2)
	similarity := d.CalculateDTWSimilarity(trace1, trace2)

	var level string
	if similarity > 0.8 {
		level = "high"
	} else if similarity > 0.5 {
		level = "medium"
	} else if similarity > 0.3 {
		level = "low"
	} else {
		level = "none"
	}

	return distance, level
}

func (d *DTWMatcher) BatchCompare(traces []*model.TraceData, target *model.TraceData) []float64 {
	results := make([]float64, len(traces))
	for i, trace := range traces {
		results[i] = d.CalculateDTWSimilarity(trace, target)
	}
	return results
}

func (d *DTWMatcher) FindNearestNeighbor(traces []*model.TraceData, target *model.TraceData) (int, float64) {
	if len(traces) == 0 {
		return -1, 0
	}

	bestIdx := 0
	bestSimilarity := d.CalculateDTWSimilarity(traces[0], target)

	for i := 1; i < len(traces); i++ {
		similarity := d.CalculateDTWSimilarity(traces[i], target)
		if similarity > bestSimilarity {
			bestSimilarity = similarity
			bestIdx = i
		}
	}

	return bestIdx, bestSimilarity
}

func (d *DTWMatcher) DetectPatternAnomalies(trace1, trace2 *model.TraceData) []string {
	anomalies := []string{}

	dtwScore := d.CalculateDTWDistance(trace1, trace2)
	featureDist, _ := d.CalculateDTWDistanceWithFeatures(trace1, trace2)

	if dtwScore < 30 && len(trace1.Points) > 10 {
		anomalies = append(anomalies, "极低DTW距离-可能重复轨迹")
	}

	if featureDist < 0.1 {
		anomalies = append(anomalies, "特征高度相似-可疑重复行为")
	}

	speedConsistency := d.calculateSpeedConsistency(trace1, trace2)
	if speedConsistency > 0.95 {
		anomalies = append(anomalies, "速度模式异常一致")
	}

	accelPattern := d.calculateAccelerationPattern(trace1, trace2)
	if accelPattern > 0.9 {
		anomalies = append(anomalies, "加速度模式高度相似")
	}

	return anomalies
}

func (d *DTWMatcher) GetTopKSimilar(traces []*model.TraceData, target *model.TraceData, k int) []struct {
	Index     int
	Similarity float64
} {
	if len(traces) == 0 || k <= 0 {
		return nil
	}

	type similarityPair struct {
		index     int
		similarity float64
	}

	pairs := make([]similarityPair, len(traces))
	for i, trace := range traces {
		pairs[i] = similarityPair{
			index:     i,
			similarity: d.CalculateDTWSimilarity(trace, target),
		}
	}

	sort.Slice(pairs, func(i, j int) bool {
		return pairs[i].similarity > pairs[j].similarity
	})

	resultLen := int(math.Min(float64(k), float64(len(pairs))))
	result := make([]struct {
		Index     int
		Similarity float64
	}, resultLen)

	for i := 0; i < resultLen; i++ {
		result[i] = struct {
			Index     int
			Similarity float64
		}{
			Index:      pairs[i].index,
			Similarity: pairs[i].similarity,
		}
	}

	return result
}

func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
