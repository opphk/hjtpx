package trace

import (
	"errors"
	"math"

	"github.com/hjtpx/hjtpx/internal/model"
)

type DTWMatcher struct {
	extractor *TraceExtractor
}

func NewDTWMatcher() *DTWMatcher {
	return &DTWMatcher{
		extractor: NewTraceExtractor(),
	}
}

func (d *DTWMatcher) CalculateDTWDistance(trace1, trace2 *model.TraceData) float64 {
	return d.CalculateDTWDistanceWithConstraint(trace1, trace2, 0)
}

func (d *DTWMatcher) CalculateDTWDistanceWithConstraint(trace1, trace2 *model.TraceData, windowSize int) float64 {
	if trace1 == nil || trace2 == nil || len(trace1.Points) < 2 || len(trace2.Points) < 2 {
		return math.MaxFloat64
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

	return d.computeDTW(distMatrix, n, m, windowSize)
}

func (d *DTWMatcher) computeDTW(distMatrix [][]float64, n, m, windowSize int) float64 {
	dtwMatrix := make([][]float64, n)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}

	dtwMatrix[0][0] = distMatrix[0][0]

	for i := 1; i < n; i++ {
		if windowSize == 0 || i <= windowSize {
			dtwMatrix[i][0] = dtwMatrix[i-1][0] + distMatrix[i][0]
		}
	}

	for j := 1; j < m; j++ {
		if windowSize == 0 || j <= windowSize {
			dtwMatrix[0][j] = dtwMatrix[0][j-1] + distMatrix[0][j]
		}
	}

	for i := 1; i < n; i++ {
		for j := 1; j < m; j++ {
			if windowSize > 0 && math.Abs(float64(i-j)) > float64(windowSize) {
				continue
			}

			minPrev := math.Min(dtwMatrix[i-1][j], math.Min(dtwMatrix[i][j-1], dtwMatrix[i-1][j-1]))
			dtwMatrix[i][j] = distMatrix[i][j] + minPrev
		}
	}

	return dtwMatrix[n-1][m-1] / float64(n+m)
}

func (d *DTWMatcher) calculatePointDistance(p1, p2 model.TracePoint) float64 {
	dx := p1.X - p2.X
	dy := p1.Y - p2.Y
	dt := float64(p1.Timestamp - p2.Timestamp) / 1000.0

	return math.Sqrt(dx*dx + dy*dy + dt*dt)
}

func (d *DTWMatcher) CalculateDTWSimilarity(trace1, trace2 *model.TraceData) float64 {
	return d.CalculateDTWSimilarityWithConstraint(trace1, trace2, 0)
}

func (d *DTWMatcher) CalculateDTWSimilarityWithConstraint(trace1, trace2 *model.TraceData, windowSize int) float64 {
	distance := d.CalculateDTWDistanceWithConstraint(trace1, trace2, windowSize)
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

func (d *DTWMatcher) max3(a, b, c float64) float64 {
	return math.Max(a, math.Max(b, c))
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

func (d *DTWMatcher) CalculateFastDTWDistance(trace1, trace2 *model.TraceData) float64 {
	if trace1 == nil || trace2 == nil || len(trace1.Points) < 2 || len(trace2.Points) < 2 {
		return math.MaxFloat64
	}

	return d.fastDTW(trace1.Points, trace2.Points, 5)
}

func (d *DTWMatcher) fastDTW(points1, points2 []model.TracePoint, radius int) float64 {
	n := len(points1)
	m := len(points2)

	if n <= radius*2+1 && m <= radius*2+1 {
		return d.classicDTW(points1, points2)
	}

	reduced1 := d.reduceByHalf(points1)
	reduced2 := d.reduceByHalf(points2)

	distance := d.fastDTW(reduced1, reduced2, radius)

	window := d.expandWindow(reduced1, reduced2, distance, radius)

	return d.constrainedDTW(points1, points2, window)
}

func (d *DTWMatcher) classicDTW(points1, points2 []model.TracePoint) float64 {
	n := len(points1)
	m := len(points2)

	dtw := make([][]float64, n)
	for i := range dtw {
		dtw[i] = make([]float64, m)
		for j := range dtw[i] {
			dtw[i][j] = math.MaxFloat64
		}
	}

	dtw[0][0] = d.calculatePointDistance(points1[0], points2[0])

	for i := 1; i < n; i++ {
		dtw[i][0] = dtw[i-1][0] + d.calculatePointDistance(points1[i], points2[0])
	}

	for j := 1; j < m; j++ {
		dtw[0][j] = dtw[0][j-1] + d.calculatePointDistance(points1[0], points2[j])
	}

	for i := 1; i < n; i++ {
		for j := 1; j < m; j++ {
			minPrev := math.Min(dtw[i-1][j], math.Min(dtw[i][j-1], dtw[i-1][j-1]))
			dtw[i][j] = d.calculatePointDistance(points1[i], points2[j]) + minPrev
		}
	}

	return dtw[n-1][m-1]
}

func (d *DTWMatcher) reduceByHalf(points []model.TracePoint) []model.TracePoint {
	if len(points) <= 2 {
		return points
	}

	reduced := make([]model.TracePoint, 0, (len(points)+1)/2)
	for i := 0; i < len(points); i += 2 {
		reduced = append(reduced, points[i])
	}

	if len(points)%2 == 1 {
		reduced = append(reduced, points[len(points)-1])
	}

	return reduced
}

func (d *DTWMatcher) expandWindow(reduced1, reduced2 []model.TracePoint, distance float64, radius int) [][]int {
	n := len(reduced1)
	m := len(reduced2)

	window := make([][]int, 0)

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			window = append(window, []int{i * 2, j * 2})
			window = append(window, []int{i * 2, j * 2 + 1})
			window = append(window, []int{i * 2 + 1, j * 2})
			window = append(window, []int{i * 2 + 1, j * 2 + 1})
		}
	}

	return window
}

func (d *DTWMatcher) constrainedDTW(points1, points2 []model.TracePoint, window [][]int) float64 {
	n := len(points1)
	m := len(points2)

	dtw := make([][]float64, n)
	for i := range dtw {
		dtw[i] = make([]float64, m)
		for j := range dtw[i] {
			dtw[i][j] = math.MaxFloat64
		}
	}

	dtw[0][0] = d.calculatePointDistance(points1[0], points2[0])

	for _, pair := range window {
		i, j := pair[0], pair[1]
		if i >= n || j >= m {
			continue
		}

		if i == 0 && j == 0 {
			continue
		}

		minPrev := math.MaxFloat64
		if i > 0 {
			minPrev = math.Min(minPrev, dtw[i-1][j])
		}
		if j > 0 {
			minPrev = math.Min(minPrev, dtw[i][j-1])
		}
		if i > 0 && j > 0 {
			minPrev = math.Min(minPrev, dtw[i-1][j-1])
		}

		if minPrev != math.MaxFloat64 {
			dtw[i][j] = d.calculatePointDistance(points1[i], points2[j]) + minPrev
		}
	}

	return dtw[n-1][m-1]
}

func (d *DTWMatcher) CalculateMultiFeatureDTW(trace1, trace2 *model.TraceData) (float64, error) {
	features1, err := d.extractor.ExtractFeatures(trace1)
	if err != nil {
		return 0, err
	}

	features2, err := d.extractor.ExtractFeatures(trace2)
	if err != nil {
		return 0, err
	}

	advanced1, err := d.extractor.ExtractAdvancedFeatures(trace1)
	if err != nil {
		return 0, err
	}

	advanced2, err := d.extractor.ExtractAdvancedFeatures(trace2)
	if err != nil {
		return 0, err
	}

	return d.multiFeatureDTW(features1, features2, advanced1, advanced2), nil
}

func (d *DTWMatcher) multiFeatureDTW(f1, f2 *model.TraceFeatures, a1, a2 *AdvancedFeatures) float64 {
	weights := map[string]float64{
		"avg_speed":          0.15,
		"max_speed":          0.10,
		"speed_variance":     0.15,
		"max_acceleration":   0.10,
		"avg_acceleration":   0.10,
		"smoothness":         0.10,
		"path_ratio":         0.10,
		"avg_curvature":      0.05,
		"jitter_frequency":   0.05,
		"jitter_amplitude":   0.05,
		"hurst_exponent":     0.025,
		"fractal_dimension":  0.025,
	}

	totalDistance := 0.0
	totalWeight := 0.0

	distance, ok := d.featureDistance("avg_speed", f1.AvgSpeed, f2.AvgSpeed, 1000.0)
	if ok {
		totalDistance += distance * weights["avg_speed"]
		totalWeight += weights["avg_speed"]
	}

	distance, ok = d.featureDistance("max_speed", f1.MaxSpeed, f2.MaxSpeed, 2000.0)
	if ok {
		totalDistance += distance * weights["max_speed"]
		totalWeight += weights["max_speed"]
	}

	distance, ok = d.featureDistance("speed_variance", f1.SpeedVariance, f2.SpeedVariance, 1000.0)
	if ok {
		totalDistance += distance * weights["speed_variance"]
		totalWeight += weights["speed_variance"]
	}

	distance, ok = d.featureDistance("max_acceleration", f1.MaxAcceleration, f2.MaxAcceleration, 5000.0)
	if ok {
		totalDistance += distance * weights["max_acceleration"]
		totalWeight += weights["max_acceleration"]
	}

	distance, ok = d.featureDistance("avg_acceleration", f1.AvgAcceleration, f2.AvgAcceleration, 2000.0)
	if ok {
		totalDistance += distance * weights["avg_acceleration"]
		totalWeight += weights["avg_acceleration"]
	}

	distance, ok = d.featureDistance("smoothness", f1.Smoothness, f2.Smoothness, 1.0)
	if ok {
		totalDistance += distance * weights["smoothness"]
		totalWeight += weights["smoothness"]
	}

	distance, ok = d.featureDistance("path_ratio", f1.PathRatio, f2.PathRatio, 5.0)
	if ok {
		totalDistance += distance * weights["path_ratio"]
		totalWeight += weights["path_ratio"]
	}

	distance, ok = d.featureDistance("avg_curvature", f1.AvgCurvature, f2.AvgCurvature, 1.0)
	if ok {
		totalDistance += distance * weights["avg_curvature"]
		totalWeight += weights["avg_curvature"]
	}

	distance, ok = d.featureDistance("jitter_frequency", f1.JitterFrequency, f2.JitterFrequency, 1.0)
	if ok {
		totalDistance += distance * weights["jitter_frequency"]
		totalWeight += weights["jitter_frequency"]
	}

	distance, ok = d.featureDistance("jitter_amplitude", f1.JitterAmplitude, f2.JitterAmplitude, 50.0)
	if ok {
		totalDistance += distance * weights["jitter_amplitude"]
		totalWeight += weights["jitter_amplitude"]
	}

	if a1 != nil && a2 != nil {
		distance, ok = d.featureDistance("hurst_exponent", a1.HurstExponent, a2.HurstExponent, 1.0)
		if ok {
			totalDistance += distance * weights["hurst_exponent"]
			totalWeight += weights["hurst_exponent"]
		}

		distance, ok = d.featureDistance("fractal_dimension", a1.FractalDimension, a2.FractalDimension, 1.0)
		if ok {
			totalDistance += distance * weights["fractal_dimension"]
			totalWeight += weights["fractal_dimension"]
		}
	}

	if totalWeight == 0 {
		return 1.0
	}

	return totalDistance / totalWeight
}

func (d *DTWMatcher) featureDistance(name string, v1, v2, maxValue float64) (float64, bool) {
	if maxValue <= 0 {
		return 0, false
	}
	return math.Abs(v1-v2) / maxValue, true
}

func (d *DTWMatcher) CalculateDTWDistanceWithSakoeChiba(trace1, trace2 *model.TraceData, windowSize int) float64 {
	return d.CalculateDTWDistanceWithConstraint(trace1, trace2, windowSize)
}

func (d *DTWMatcher) CalculateDTWDistanceWithItakura(trace1, trace2 *model.TraceData) float64 {
	if trace1 == nil || trace2 == nil || len(trace1.Points) < 2 || len(trace2.Points) < 2 {
		return math.MaxFloat64
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

	slopeMin := 0.5
	slopeMax := 2.0

	dtwMatrix := make([][]float64, n)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}

	dtwMatrix[0][0] = distMatrix[0][0]

	for i := 1; i < n; i++ {
		slope := float64(i) / float64(1)
		if slope >= slopeMin && slope <= slopeMax {
			dtwMatrix[i][0] = dtwMatrix[i-1][0] + distMatrix[i][0]
		}
	}

	for j := 1; j < m; j++ {
		slope := float64(1) / float64(j)
		if slope >= slopeMin && slope <= slopeMax {
			dtwMatrix[0][j] = dtwMatrix[0][j-1] + distMatrix[0][j]
		}
	}

	for i := 1; i < n; i++ {
		for j := 1; j < m; j++ {
			slope := float64(i) / float64(j)
			if slope < slopeMin || slope > slopeMax {
				continue
			}

			minPrev := math.Min(dtwMatrix[i-1][j], math.Min(dtwMatrix[i][j-1], dtwMatrix[i-1][j-1]))
			dtwMatrix[i][j] = distMatrix[i][j] + minPrev
		}
	}

	return dtwMatrix[n-1][m-1] / float64(n+m)
}

type DTWAlignment struct {
	Path        [][2]int
	Distance    float64
	Similarity  float64
}

func (d *DTWMatcher) GetDTWAlignment(trace1, trace2 *model.TraceData) (*DTWAlignment, error) {
	if trace1 == nil || trace2 == nil || len(trace1.Points) < 2 || len(trace2.Points) < 2 {
		return nil, errors.New("invalid trace data")
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

	path := d.backtrackPath(dtwMatrix, n-1, m-1)

	maxPossible := float64(n+m) * 1000.0
	similarity := 1.0 - math.Min(dtwMatrix[n-1][m-1]/maxPossible, 1.0)

	return &DTWAlignment{
		Path:       path,
		Distance:   dtwMatrix[n-1][m-1] / float64(n+m),
		Similarity: similarity,
	}, nil
}

func (d *DTWMatcher) backtrackPath(dtwMatrix [][]float64, i, j int) [][2]int {
	path := make([][2]int, 0)
	path = append(path, [2]int{i, j})

	for i > 0 || j > 0 {
		var minVal float64
		var nextI, nextJ int

		if i == 0 {
			nextI, nextJ = 0, j-1
			minVal = dtwMatrix[0][j-1]
		} else if j == 0 {
			nextI, nextJ = i-1, 0
			minVal = dtwMatrix[i-1][0]
		} else {
			minVal = dtwMatrix[i-1][j]
			nextI, nextJ = i-1, j

			if dtwMatrix[i][j-1] < minVal {
				minVal = dtwMatrix[i][j-1]
				nextI, nextJ = i, j-1
			}

			if dtwMatrix[i-1][j-1] < minVal {
				minVal = dtwMatrix[i-1][j-1]
				nextI, nextJ = i-1, j-1
			}
		}

		i, j = nextI, nextJ
		path = append(path, [2]int{i, j})
	}

	for k, l := 0, len(path)-1; k < l; k, l = k+1, l-1 {
		path[k], path[l] = path[l], path[k]
	}

	return path
}

func (d *DTWMatcher) CalculateDTWDistanceWithWeightedWindow(trace1, trace2 *model.TraceData, windowSize int, weights []float64) float64 {
	if trace1 == nil || trace2 == nil || len(trace1.Points) < 2 || len(trace2.Points) < 2 {
		return math.MaxFloat64
	}

	n := len(trace1.Points)
	m := len(trace2.Points)

	if len(weights) == 0 {
		weights = make([]float64, n)
		for i := range weights {
			weights[i] = 1.0
		}
	}

	distMatrix := make([][]float64, n)
	for i := range distMatrix {
		distMatrix[i] = make([]float64, m)
	}

	for i := 0; i < n; i++ {
		for j := 0; j < m; j++ {
			dist := d.calculatePointDistance(trace1.Points[i], trace2.Points[j])
			if i < len(weights) {
				distMatrix[i][j] = dist * weights[i]
			} else {
				distMatrix[i][j] = dist
			}
		}
	}

	return d.computeDTW(distMatrix, n, m, windowSize)
}
