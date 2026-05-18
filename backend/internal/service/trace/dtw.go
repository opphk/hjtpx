package trace

import (
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
