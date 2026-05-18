package service

import (
	"math"
)

type OptimizedDTW struct {
	windowSize      int
	constraintType  string
	epsilon         float64
}

func NewOptimizedDTW() *OptimizedDTW {
	return &OptimizedDTW{
		windowSize:     10,
		constraintType: "sakoe_chiba",
		epsilon:        0.1,
	}
}

func NewOptimizedDTWWithParams(windowSize int, constraintType string, epsilon float64) *OptimizedDTW {
	return &OptimizedDTW{
		windowSize:     windowSize,
		constraintType: constraintType,
		epsilon:        epsilon,
	}
}

func (dtw *OptimizedDTW) ComputeDistance(traj1, traj2 []SliderPoint) float64 {
	if len(traj1) == 0 || len(traj2) == 0 {
		return math.MaxFloat64
	}

	n, m := len(traj1), len(traj2)

	switch dtw.constraintType {
	case "sakoe_chiba":
		return dtw.computeSakoeChibaDTW(traj1, traj2, n, m)
	case "itakura":
		return dtw.computeItakuraDTW(traj1, traj2, n, m)
	default:
		return dtw.computeFastDTW(traj1, traj2, n, m)
	}
}

func (dtw *OptimizedDTW) computeSakoeChibaDTW(traj1, traj2 []SliderPoint, n, m int) float64 {
	window := dtw.windowSize
	if window > n {
		window = n
	}
	if window > m {
		window = m
	}

	dtwMatrix := make([][]float64, n+1)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m+1)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}
	dtwMatrix[0][0] = 0

	for i := 1; i <= n; i++ {
		jStart := math.Max(1, float64(i-window))
		jEnd := math.Min(float64(m), float64(i+window))

		for j := int(jStart); j <= int(jEnd); j++ {
			dist := dtw.pointDistance(traj1[i-1], traj2[j-1])
			dtwMatrix[i][j] = dist + math.Min(
				math.Min(dtwMatrix[i-1][j], dtwMatrix[i][j-1]),
				dtwMatrix[i-1][j-1],
			)
		}
	}

	return dtwMatrix[n][m]
}

func (dtw *OptimizedDTW) computeItakuraDTW(traj1, traj2 []SliderPoint, n, m int) float64 {
	maxSlope := 2.0

	dtwMatrix := make([][]float64, n+1)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m+1)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}
	dtwMatrix[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			minJ := math.Max(1, float64(i)-float64(n)/float64(m)*float64(j)-float64(dtw.windowSize))
			maxJ := math.Min(float64(m), float64(i)+float64(dtw.windowSize))

			if float64(j) < minJ || float64(j) > maxJ {
				continue
			}

			slope1 := float64(j) / float64(i)
			slope2 := float64(j) / float64(i)
			if slope1 < 1.0/maxSlope || slope2 > maxSlope {
				continue
			}

			dist := dtw.pointDistance(traj1[i-1], traj2[j-1])
			dtwMatrix[i][j] = dist + math.Min(
				math.Min(dtwMatrix[i-1][j], dtwMatrix[i][j-1]),
				dtwMatrix[i-1][j-1],
			)
		}
	}

	return dtwMatrix[n][m]
}

func (dtw *OptimizedDTW) computeFastDTW(traj1, traj2 []SliderPoint, n, m int) float64 {
	dtwMatrix := make([][]float64, n+1)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m+1)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}
	dtwMatrix[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			dist := dtw.pointDistance(traj1[i-1], traj2[j-1])
			dtwMatrix[i][j] = dist + math.Min(
				math.Min(dtwMatrix[i-1][j], dtwMatrix[i][j-1]),
				dtwMatrix[i-1][j-1],
			)
		}
	}

	return dtwMatrix[n][m]
}

func (dtw *OptimizedDTW) ComputeDistanceLowerBound(traj1, traj2 []SliderPoint) float64 {
	if len(traj1) == 0 || len(traj2) == 0 {
		return 0
	}

	n, m := len(traj1), len(traj2)
	lb := 0.0

	minLen := n
	if m < minLen {
		minLen = m
	}

	for i := 0; i < minLen; i++ {
		idx1 := int(float64(i) * float64(n) / float64(minLen))
		idx2 := int(float64(i) * float64(m) / float64(minLen))

		if idx1 >= n {
			idx1 = n - 1
		}
		if idx2 >= m {
			idx2 = m - 1
		}

		dist := dtw.pointDistance(traj1[idx1], traj2[idx2])
		lb += dist
	}

	return lb
}

func (dtw *OptimizedDTW) ComputeSimilarity(traj1, traj2 []SliderPoint) float64 {
	distance := dtw.ComputeDistance(traj1, traj2)
	maxPossibleDistance := 1000.0
	similarity := 1.0 - math.Min(distance/maxPossibleDistance, 1.0)
	return math.Max(0, similarity)
}

func (dtw *OptimizedDTW) pointDistance(p1, p2 SliderPoint) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

type DTWResult struct {
	Distance      float64
	Path          [][2]int
	PathLength    int
	WarpingPoints int
}

func (dtw *OptimizedDTW) ComputeWithPath(traj1, traj2 []SliderPoint) DTWResult {
	result := DTWResult{}

	if len(traj1) == 0 || len(traj2) == 0 {
		return result
	}

	n, m := len(traj1), len(traj2)

	dtwMatrix := make([][]float64, n+1)
	for i := range dtwMatrix {
		dtwMatrix[i] = make([]float64, m+1)
		for j := range dtwMatrix[i] {
			dtwMatrix[i][j] = math.MaxFloat64
		}
	}
	dtwMatrix[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			dist := dtw.pointDistance(traj1[i-1], traj2[j-1])
			dtwMatrix[i][j] = dist + math.Min(
				math.Min(dtwMatrix[i-1][j], dtwMatrix[i][j-1]),
				dtwMatrix[i-1][j-1],
			)
		}
	}

	result.Distance = dtwMatrix[n][m]

	path := dtw.backtrackPath(dtwMatrix, n, m)
	result.Path = path
	result.PathLength = len(path)
	result.WarpingPoints = dtw.countWarpingPoints(path)

	return result
}

func (dtw *OptimizedDTW) backtrackPath(matrix [][]float64, n, m int) [][2]int {
	path := make([][2]int, 0, n+m)

	i, j := n, m
	for i > 0 || j > 0 {
		path = append(path, [2]int{i - 1, j - 1})

		if i == 0 {
			j--
		} else if j == 0 {
			i--
		} else {
			minVal := math.Min(math.Min(matrix[i-1][j], matrix[i][j-1]), matrix[i-1][j-1])

			if matrix[i-1][j-1] == minVal {
				i--
				j--
			} else if matrix[i-1][j] == minVal {
				i--
			} else {
				j--
			}
		}
	}

	for i, j = 0, len(path)-1; i < j; i, j = i+1, j-1 {
		path[i], path[j] = path[j], path[i]
	}

	return path
}

func (dtw *OptimizedDTW) countWarpingPoints(path [][2]int) int {
	count := 0
	for i := 1; i < len(path); i++ {
		dx := path[i][0] - path[i-1][0]
		dy := path[i][1] - path[i-1][1]

		if (dx == 0 && dy > 1) || (dx > 1 && dy == 0) {
			count++
		}
	}
	return count
}

type DTWBatch struct {
	dtw *OptimizedDTW
}

func NewDTWBatch() *DTWBatch {
	return &DTWBatch{
		dtw: NewOptimizedDTW(),
	}
}

func (b *DTWBatch) ComputeDistances(reference []SliderPoint, candidates [][]SliderPoint) []float64 {
	distances := make([]float64, len(candidates))

	for i, candidate := range candidates {
		distances[i] = b.dtw.ComputeDistance(reference, candidate)
	}

	return distances
}

func (b *DTWBatch) FindMostSimilar(reference []SliderPoint, candidates [][]SliderPoint) (int, float64) {
	if len(candidates) == 0 {
		return -1, math.MaxFloat64
	}

	minDist := math.MaxFloat64
	minIdx := 0

	for i, candidate := range candidates {
		dist := b.dtw.ComputeDistance(reference, candidate)
		if dist < minDist {
			minDist = dist
			minIdx = i
		}
	}

	return minIdx, minDist
}

func (b *DTWBatch) ComputeAverageDistance(traj1, traj2 []SliderPoint) float64 {
	return b.dtw.ComputeDistance(traj1, traj2)
}

type DTWTemplate struct {
	Name      string
	Trajectory []SliderPoint
	DTW       *OptimizedDTW
}

type DTWClassifier struct {
	templates []DTWTemplate
}

func NewDTWClassifier() *DTWClassifier {
	return &DTWClassifier{
		templates: make([]DTWTemplate, 0),
	}
}

func (c *DTWClassifier) AddTemplate(name string, trajectory []SliderPoint) {
	c.templates = append(c.templates, DTWTemplate{
		Name:       name,
		Trajectory: trajectory,
		DTW:        NewOptimizedDTW(),
	})
}

func (c *DTWClassifier) Classify(trajectory []SliderPoint) (string, float64) {
	if len(c.templates) == 0 {
		return "", 0
	}

	minDist := math.MaxFloat64
	bestMatch := ""

	for _, template := range c.templates {
		dist := template.DTW.ComputeDistance(trajectory, template.Trajectory)
		if dist < minDist {
			minDist = dist
			bestMatch = template.Name
		}
	}

	similarity := 1.0 - math.Min(minDist/500.0, 1.0)

	return bestMatch, similarity
}

func (c *DTWClassifier) ComputeSimilarity(trajectory []SliderPoint) []float64 {
	similarities := make([]float64, len(c.templates))

	for i, template := range c.templates {
		dist := template.DTW.ComputeDistance(trajectory, template.Trajectory)
		similarities[i] = 1.0 - math.Min(dist/500.0, 1.0)
	}

	return similarities
}

type MultiScaleDTW struct {
	scales     []int
	dtw        *OptimizedDTW
}

func NewMultiScaleDTW() *MultiScaleDTW {
	return &MultiScaleDTW{
		scales: []int{1, 2, 4},
		dtw:    NewOptimizedDTW(),
	}
}

func (ms *MultiScaleDTW) ComputeDistance(traj1, traj2 []SliderPoint) float64 {
	if len(traj1) < 2 || len(traj2) < 2 {
		return ms.dtw.ComputeDistance(traj1, traj2)
	}

	coarseDist := ms.computeAtScale(traj1, traj2, ms.scales[len(ms.scales)-1])

	refinedDist := coarseDist
	for i := len(ms.scales) - 2; i >= 0; i-- {
		scale := ms.scales[i]
		refined := ms.computeAtScale(traj1, traj2, scale)
		refinedDist = refinedDist*0.3 + refined*0.7
	}

	return refinedDist
}

func (ms *MultiScaleDTW) computeAtScale(traj1, traj2 []SliderPoint, scale int) float64 {
	scaled1 := ms.downsample(traj1, scale)
	scaled2 := ms.downsample(traj2, scale)

	return ms.dtw.ComputeDistance(scaled1, scaled2)
}

func (ms *MultiScaleDTW) downsample(points []SliderPoint, factor int) []SliderPoint {
	if factor <= 1 || len(points) <= factor {
		return points
	}

	result := make([]SliderPoint, 0, len(points)/factor+1)

	for i := 0; i < len(points); i += factor {
		result = append(result, points[i])
	}

	if len(points)%factor != 0 && len(result) > 0 {
		result = append(result, points[len(points)-1])
	}

	return result
}
