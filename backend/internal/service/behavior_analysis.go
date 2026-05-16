package service

import (
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
)

type BehaviorDataPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Event     string  `json:"event"`
}

type KeyboardDataPoint struct {
	Key          string `json:"key"`
	Timestamp    int64  `json:"timestamp"`
	KeyDownTime  int64  `json:"key_down_time,omitempty"`
	KeyUpTime    int64  `json:"key_up_time,omitempty"`
	HoldDuration int64  `json:"hold_duration,omitempty"`
	IsShiftHeld  bool   `json:"is_shift_held,omitempty"`
	IsCtrlHeld   bool   `json:"is_ctrl_held,omitempty"`
	IsAltHeld    bool   `json:"is_alt_held,omitempty"`
}

type MouseTrajectory struct {
	Points           []BehaviorDataPoint `json:"points"`
	TotalDistance   float64             `json:"total_distance"`
	AverageSpeed    float64             `json:"average_speed"`
	MaxSpeed        float64             `json:"max_speed"`
	MinSpeed        float64             `json:"min_speed"`
	PathEfficiency  float64             `json:"path_efficiency"`
	DirectionChanges int                `json:"direction_changes"`
	SmoothedDistance float64            `json:"smoothed_distance,omitempty"`
	SpeedVariance   float64             `json:"speed_variance,omitempty"`
	AccelerationAvg float64             `json:"acceleration_avg,omitempty"`
	CurvatureAvg    float64             `json:"curvature_avg,omitempty"`
	JitterScore     float64             `json:"jitter_score,omitempty"`
}

type ClickPattern struct {
	Clicks            []BehaviorDataPoint `json:"clicks"`
	ClickCount        int                 `json:"click_count"`
	AverageInterval   float64             `json:"average_interval"`
	ClickSpeed        float64             `json:"click_speed"`
	Regularity        float64             `json:"regularity"`
	IntervalVariance  float64             `json:"interval_variance"`
	IntervalStdDev    float64             `json:"interval_std_dev"`
	XDistribution     []int               `json:"x_distribution,omitempty"`
	YDistribution     []int               `json:"y_distribution,omitempty"`
	PositionEntropy   float64             `json:"position_entropy,omitempty"`
	IsDoubleClick     bool                `json:"is_double_click"`
	ClickAreaSize     float64             `json:"click_area_size,omitempty"`
}

type KeyboardPattern struct {
	KeyStrokes        []KeyboardDataPoint   `json:"key_strokes"`
	KeystrokeCount    int                  `json:"keystroke_count"`
	AverageInterval   float64              `json:"average_interval"`
	IntervalVariance  float64             `json:"interval_variance"`
	IntervalStdDev    float64             `json:"interval_std_dev"`
	AverageHoldTime   float64             `json:"average_hold_time"`
	HoldTimeVariance  float64             `json:"hold_time_variance"`
	TypingSpeed       float64             `json:"typing_speed"`
	Regularity        float64             `json:"regularity"`
	CommonPairs       map[string]int      `json:"common_pairs,omitempty"`
	MostFrequentKey   string              `json:"most_frequent_key,omitempty"`
	ErrorRate         float64             `json:"error_rate,omitempty"`
	ComboDetected     bool                `json:"combo_detected"`
	ComboPatterns     []string             `json:"combo_patterns,omitempty"`
}

type SpeedAnalysis struct {
	Speeds              []float64 `json:"speeds"`
	AverageSpeed        float64   `json:"average_speed"`
	MedianSpeed         float64   `json:"median_speed"`
	MaxSpeed            float64   `json:"max_speed"`
	MinSpeed            float64   `json:"min_speed"`
	SpeedVariance       float64   `json:"speed_variance"`
	SpeedStdDev         float64   `json:"speed_std_dev"`
	SpeedSkewness       float64   `json:"speed_skewness"`
	Accelerations       []float64 `json:"accelerations"`
	AverageAcceleration float64   `json:"average_acceleration"`
	MaxAcceleration     float64   `json:"max_acceleration"`
	JerkAvg             float64   `json:"jerk_avg"`
	JerkMax             float64   `json:"jerk_max"`
	IsSpeedConsistent   bool      `json:"is_speed_consistent"`
	SpeedOutliers       int       `json:"speed_outliers"`
}

type PathSimilarity struct {
	ComparedPathLength int     `json:"compared_path_length"`
	SimilarityScore    float64 `json:"similarity_score"`
	IsPathRepeated     bool    `json:"is_path_repeated"`
	RepeatedSegments   int     `json:"repeated_segments"`
	PathHashMatch      bool    `json:"path_hash_match"`
	FrechetDistance    float64 `json:"frechet_distance"`
	DTWDistance        float64 `json:"dtw_distance"`
}

type DwellPoint struct {
	StartTime    int64   `json:"start_time"`
	EndTime      int64   `json:"end_time"`
	Duration     int64   `json:"duration"`
	CenterX      int     `json:"center_x"`
	CenterY      int     `json:"center_y"`
	PointCount   int     `json:"point_count"`
	AvgSpeed     float64 `json:"avg_speed"`
	IsSuspicious bool    `json:"is_suspicious"`
}

type BehaviorFeatures struct {
	TrajectoryFeatures    []float64 `json:"trajectory_features"`
	SpeedFeatures        []float64 `json:"speed_features"`
	ClickFeatures        []float64 `json:"click_features"`
	KeyboardFeatures     []float64 `json:"keyboard_features,omitempty"`
	FeatureVector        []float64 `json:"feature_vector"`
	AnomalyScore         float64   `json:"anomaly_score"`
	IsAnomalous          bool      `json:"is_anomalous"`
	AnomalyIndicators    []string  `json:"anomaly_indicators"`
}

type AnalysisResult struct {
	Trajectory      MouseTrajectory    `json:"trajectory"`
	ClickPattern   ClickPattern       `json:"click_pattern"`
	KeyboardPattern KeyboardPattern   `json:"keyboard_pattern,omitempty"`
	SpeedAnalysis  SpeedAnalysis     `json:"speed_analysis,omitempty"`
	PathSimilarity PathSimilarity   `json:"path_similarity,omitempty"`
	DwellPoints    []DwellPoint      `json:"dwell_points,omitempty"`
	BehaviorFeatures BehaviorFeatures `json:"behavior_features,omitempty"`
	RiskScore      float64           `json:"risk_score"`
	RiskIndicators []string          `json:"risk_indicators"`
	IsBotLikely    bool              `json:"is_bot_likely"`
	Confidence     float64           `json:"confidence"`
	RiskFactors    map[string]float64 `json:"risk_factors"`
}

type BehaviorAnalysisService struct {
	storedPaths   [][]BehaviorDataPoint
	cache         map[string]*AnalysisResult
	cacheMutex    sync.RWMutex
	analysisCount int64
	maxCacheSize  int
}

type AnalysisCache struct {
	Result      *AnalysisResult
	CreatedAt   time.Time
	AccessCount int
}

func NewBehaviorAnalysisService() *BehaviorAnalysisService {
	return &BehaviorAnalysisService{
		storedPaths:   make([][]BehaviorDataPoint, 0),
		cache:         make(map[string]*AnalysisResult),
		maxCacheSize:  1000,
	}
}

func (s *BehaviorAnalysisService) AnalyzeBehavior(behaviorData []models.BehaviorData) (*AnalysisResult, error) {
	result := &AnalysisResult{
		RiskIndicators: []string{},
		RiskFactors:    make(map[string]float64),
	}

	cacheKey := s.generateCacheKey(behaviorData)
	s.cacheMutex.RLock()
	if cached, exists := s.cache[cacheKey]; exists {
		s.cacheMutex.RUnlock()
		atomic.AddInt64(&s.analysisCount, 1)
		return cached, nil
	}
	s.cacheMutex.RUnlock()

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
		smoothedPoints := s.smoothTrajectory(points, 5)
		result.Trajectory = s.analyzeMouseTrajectory(smoothedPoints, points)
		result.SpeedAnalysis = s.analyzeSpeed(points)
		result.PathSimilarity = s.checkPathSimilarity(smoothedPoints)
		result.DwellPoints = s.detectDwellPoints(points)
	}

	if len(clicks) > 0 {
		result.ClickPattern = s.analyzeClickPatternEnhanced(clicks)
	}

	if len(keyStrokes) > 0 {
		result.KeyboardPattern = s.analyzeKeyboardPattern(keyStrokes)
	}

	s.calculateRiskScoreEnhanced(result)
	result.BehaviorFeatures = s.extractBehaviorFeatures(result)

	s.cacheMutex.Lock()
	if len(s.cache) >= s.maxCacheSize {
		s.evictOldestCache()
	}
	s.cache[cacheKey] = result
	s.cacheMutex.Unlock()

	atomic.AddInt64(&s.analysisCount, 1)

	return result, nil
}

func (s *BehaviorAnalysisService) smoothTrajectory(points []BehaviorDataPoint, windowSize int) []BehaviorDataPoint {
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

func (s *BehaviorAnalysisService) savitzkyGolaySmooth(points []BehaviorDataPoint, windowSize int, order int) []BehaviorDataPoint {
	if len(points) < windowSize || order >= windowSize {
		return points
	}

	if windowSize%2 == 0 {
		windowSize++
	}
	if order >= windowSize {
		order = windowSize - 1
	}

	halfWindow := windowSize / 2
	smoothed := make([]BehaviorDataPoint, len(points))

	coeffs := s.computeSGCoefficients(windowSize, order)

	for i := range points {
		start := i - halfWindow
		end := i + halfWindow

		if start < 0 {
			start = 0
		}
		if end >= len(points) {
			end = len(points) - 1
		}

		sumX := 0.0
		sumY := 0.0
		idx := 0

		for j := start; j <= end; j++ {
			sumX += float64(points[j].X) * coeffs[idx]
			sumY += float64(points[j].Y) * coeffs[idx]
			idx++
		}

		smoothed[i] = points[i]
		smoothed[i].X = int(math.Round(sumX))
		smoothed[i].Y = int(math.Round(sumY))
	}

	return smoothed
}

func (s *BehaviorAnalysisService) computeSGCoefficients(windowSize int, order int) []float64 {
	coeffs := make([]float64, windowSize)
	m := (windowSize - 1) / 2

	B := make([][]float64, windowSize)
	for i := range B {
		B[i] = make([]float64, order+1)
		for k := 0; k <= order; k++ {
			val := math.Pow(float64(i-m), float64(k))
			B[i][k] = val
		}
	}

	Bt := make([][]float64, order+1)
	for i := 0; i <= order; i++ {
		Bt[i] = make([]float64, windowSize)
		for j := 0; j < windowSize; j++ {
			Bt[i][j] = B[j][i]
		}
	}

	XtX := make([][]float64, order+1)
	for i := 0; i <= order; i++ {
		XtX[i] = make([]float64, order+1)
		for j := 0; j <= order; j++ {
			sum := 0.0
			for k := 0; k < windowSize; k++ {
				sum += Bt[i][k] * B[k][j]
			}
			XtX[i][j] = sum
		}
	}

	XtXInv := s.invertMatrix(XtX)

	for i := 0; i <= order; i++ {
		for j := 0; j <= order; j++ {
			if i == 0 {
				for k := 0; k < windowSize; k++ {
					coeffs[k] += XtXInv[i][j] * Bt[0][k]
				}
			}
		}
	}

	if coeffs[0] == 0 {
		denom := 0.0
		for _, c := range coeffs {
			denom += c
		}
		if denom != 0 {
			for i := range coeffs {
				coeffs[i] /= denom
			}
		} else {
			for i := range coeffs {
				coeffs[i] = 1.0 / float64(windowSize)
			}
		}
	}

	return coeffs
}

func (s *BehaviorAnalysisService) invertMatrix(matrix [][]float64) [][]float64 {
	n := len(matrix)
	augmented := make([][]float64, n)
	for i := range augmented {
		augmented[i] = make([]float64, 2*n)
		for j := 0; j < n; j++ {
			augmented[i][j] = matrix[i][j]
		}
		augmented[i][n+i] = 1.0
	}

	for i := 0; i < n; i++ {
		pivot := augmented[i][i]
		if math.Abs(pivot) < 1e-10 {
			for k := i + 1; k < n; k++ {
				if math.Abs(augmented[k][i]) > 1e-10 {
					augmented[i], augmented[k] = augmented[k], augmented[i]
					pivot = augmented[i][i]
					break
				}
			}
		}

		for j := 0; j < 2*n; j++ {
			augmented[i][j] /= pivot
		}

		for k := 0; k < n; k++ {
			if k != i {
				factor := augmented[k][i]
				for j := 0; j < 2*n; j++ {
					augmented[k][j] -= factor * augmented[i][j]
				}
			}
		}
	}

	inverse := make([][]float64, n)
	for i := 0; i < n; i++ {
		inverse[i] = make([]float64, n)
		for j := 0; j < n; j++ {
			inverse[i][j] = augmented[i][n+j]
		}
	}

	return inverse
}

func (s *BehaviorAnalysisService) analyzeSpeed(points []BehaviorDataPoint) SpeedAnalysis {
	analysis := SpeedAnalysis{}

	if len(points) < 2 {
		return analysis
	}

	speeds := []float64{}
	accelerations := []float64{}

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

	if len(speeds) > 0 {
		analysis.Speeds = speeds
		analysis.AverageSpeed = s.mean(speeds)
		analysis.MaxSpeed = s.max(speeds)
		analysis.MinSpeed = s.min(speeds)
		analysis.SpeedVariance = s.variance(speeds)
		analysis.SpeedStdDev = math.Sqrt(analysis.SpeedVariance)
		analysis.SpeedSkewness = s.skewness(speeds)
		analysis.MedianSpeed = s.median(speeds)

		varianceThreshold := analysis.SpeedVariance * 3
		for _, speed := range speeds {
			if math.Abs(speed-analysis.AverageSpeed) > varianceThreshold {
				analysis.SpeedOutliers++
			}
		}
	}

	for i := 2; i < len(speeds); i++ {
		dt := float64(points[i].Timestamp - points[i-2].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}

	if len(accelerations) > 0 {
		analysis.Accelerations = accelerations
		analysis.AverageAcceleration = s.mean(accelerations)
		analysis.MaxAcceleration = s.maxAbs(accelerations)
	}

	jerks := []float64{}
	for i := 2; i < len(accelerations); i++ {
		dt := float64(points[i+1].Timestamp - points[i-1].Timestamp)
		if dt > 0 {
			jerk := (accelerations[i] - accelerations[i-1]) / dt
			jerks = append(jerks, jerk)
		}
	}

	if len(jerks) > 0 {
		analysis.JerkAvg = s.mean(jerks)
		analysis.JerkMax = s.maxAbs(jerks)
	}

	analysis.IsSpeedConsistent = analysis.SpeedStdDev < analysis.AverageSpeed*0.3

	return analysis
}

func (s *BehaviorAnalysisService) checkPathSimilarity(currentPath []BehaviorDataPoint) PathSimilarity {
	similarity := PathSimilarity{
		ComparedPathLength: len(currentPath),
	}

	if len(currentPath) < 5 {
		return similarity
	}

	hash := s.computePathHash(currentPath)

	for _, storedPath := range s.storedPaths {
		if len(storedPath) < 5 || len(storedPath) != len(currentPath) {
			continue
		}

		dtwDist := s.computeDTWDistance(currentPath, storedPath)
		if similarity.DTWDistance == 0 || dtwDist < similarity.DTWDistance {
			similarity.DTWDistance = dtwDist
		}

		frechetDist := s.computeFrechetDistance(currentPath, storedPath)
		if similarity.FrechetDistance == 0 || frechetDist < similarity.FrechetDistance {
			similarity.FrechetDistance = frechetDist
		}

		similarityScore := s.computePathCorrelation(currentPath, storedPath)
		if similarityScore > similarity.SimilarityScore {
			similarity.SimilarityScore = similarityScore
		}
	}

	similarity.IsPathRepeated = similarity.SimilarityScore > 0.85
	similarity.PathHashMatch = s.checkPathHashMatch(hash)

	s.storedPaths = append(s.storedPaths, currentPath)
	if len(s.storedPaths) > 100 {
		s.storedPaths = s.storedPaths[1:]
	}

	return similarity
}

func (s *BehaviorAnalysisService) computePathHash(points []BehaviorDataPoint) string {
	hashParts := []string{}
	for i := 0; i < len(points) && i < 20; i++ {
		bucketX := points[i].X / 50
		bucketY := points[i].Y / 50
		hashParts = append(hashParts, fmt.Sprintf("%d,%d", bucketX, bucketY))
	}
	return strings.Join(hashParts, "|")
}

func (s *BehaviorAnalysisService) checkPathHashMatch(hash string) bool {
	parts := strings.Split(hash, "|")
	for _, storedPath := range s.storedPaths {
		if len(storedPath) < 5 {
			continue
		}
		storedHash := s.computePathHash(storedPath)
		storedParts := strings.Split(storedHash, "|")

		matches := 0
		for i := 0; i < len(parts) && i < len(storedParts); i++ {
			if parts[i] == storedParts[i] {
				matches++
			}
		}

		if matches >= len(parts)/2 {
			return true
		}
	}
	return false
}

func (s *BehaviorAnalysisService) computeDTWDistance(path1, path2 []BehaviorDataPoint) float64 {
	n, m := len(path1), len(path2)
	dtw := make([][]float64, n+1)
	for i := range dtw {
		dtw[i] = make([]float64, m+1)
		for j := range dtw[i] {
			dtw[i][j] = math.MaxFloat64
		}
	}
	dtw[0][0] = 0

	for i := 1; i <= n; i++ {
		for j := 1; j <= m; j++ {
			dist := s.pointDistance(path1[i-1], path2[j-1])
			dtw[i][j] = dist + math.Min(math.Min(dtw[i-1][j], dtw[i][j-1]), dtw[i-1][j-1])
		}
	}

	return dtw[n][m]
}

func (s *BehaviorAnalysisService) computeFrechetDistance(path1, path2 []BehaviorDataPoint) float64 {
	n, m := len(path1), len(path2)
	ca := make([][]float64, n)
	for i := range ca {
		ca[i] = make([]float64, m)
		for j := range ca[i] {
			ca[i][j] = -1
		}
	}

	var compute func(i, j int) float64
	compute = func(i, j int) float64 {
		if ca[i][j] > -0.5 {
			return ca[i][j]
		}

		dist := s.pointDistance(path1[i], path2[j])

		if i == 0 && j == 0 {
			ca[i][j] = dist
		} else if i > 0 && j == 0 {
			ca[i][j] = math.Max(compute(i-1, 0), dist)
		} else if i == 0 && j > 0 {
			ca[i][j] = math.Max(compute(0, j-1), dist)
		} else {
			ca[i][j] = math.Max(dist, math.Min(math.Min(compute(i-1, j), compute(i-1, j-1)), compute(i, j-1)))
		}

		return ca[i][j]
	}

	return compute(n-1, m-1)
}

func (s *BehaviorAnalysisService) computePathCorrelation(path1, path2 []BehaviorDataPoint) float64 {
	if len(path1) != len(path2) {
		return 0
	}

	x1, y1 := s.extractCoordinates(path1)
	x2, y2 := s.extractCoordinates(path2)

	corrX := s.pearsonCorrelation(x1, x2)
	corrY := s.pearsonCorrelation(y1, y2)

	return (corrX + corrY) / 2
}

func (s *BehaviorAnalysisService) extractCoordinates(points []BehaviorDataPoint) ([]float64, []float64) {
	x := make([]float64, len(points))
	y := make([]float64, len(points))
	for i, p := range points {
		x[i] = float64(p.X)
		y[i] = float64(p.Y)
	}
	return x, y
}

func (s *BehaviorAnalysisService) pearsonCorrelation(x, y []float64) float64 {
	if len(x) != len(y) || len(x) < 2 {
		return 0
	}

	meanX := s.mean(x)
	meanY := s.mean(y)

	numerator := 0.0
	denomX := 0.0
	denomY := 0.0

	for i := range x {
		dx := x[i] - meanX
		dy := y[i] - meanY
		numerator += dx * dy
		denomX += dx * dx
		denomY += dy * dy
	}

	denom := math.Sqrt(denomX * denomY)
	if denom == 0 {
		return 0
	}

	return numerator / denom
}

func (s *BehaviorAnalysisService) pointDistance(p1, p2 BehaviorDataPoint) float64 {
	dx := float64(p1.X - p2.X)
	dy := float64(p1.Y - p2.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (s *BehaviorAnalysisService) analyzeMouseTrajectory(smoothedPoints []BehaviorDataPoint, originalPoints []BehaviorDataPoint) MouseTrajectory {
	traj := MouseTrajectory{
		Points: originalPoints,
	}

	if len(originalPoints) < 2 {
		return traj
	}

	totalDistance := 0.0
	maxSpeed := 0.0
	minSpeed := math.MaxFloat64
	speeds := []float64{}
	directionChanges := 0
	prevAngle := 0.0
	smoothedDistance := 0.0
	curvatures := []float64{}

	for i := 1; i < len(originalPoints); i++ {
		dx := float64(originalPoints[i].X - originalPoints[i-1].X)
		dy := float64(originalPoints[i].Y - originalPoints[i-1].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance

		if i < len(smoothedPoints) {
			dxS := float64(smoothedPoints[i].X - smoothedPoints[i-1].X)
			dyS := float64(smoothedPoints[i].Y - smoothedPoints[i-1].Y)
			smoothedDistance += math.Sqrt(dxS*dxS + dyS*dyS)
		}

		dt := float64(originalPoints[i].Timestamp - originalPoints[i-1].Timestamp)
		if dt > 0 {
			speed := distance / dt
			speeds = append(speeds, speed)
			if speed > maxSpeed {
				maxSpeed = speed
			}
			if speed < minSpeed {
				minSpeed = speed
			}
		}

		if i > 1 {
			angle := math.Atan2(dy, dx)
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.5 {
				directionChanges++
			}
			prevAngle = angle

			if i > 1 && i < len(originalPoints) {
				curv := s.computeCurvature(originalPoints[i-2], originalPoints[i-1], originalPoints[i])
				curvatures = append(curvatures, math.Abs(curv))
			}
		}
	}

	traj.TotalDistance = totalDistance
	traj.SmoothedDistance = smoothedDistance
	traj.MaxSpeed = maxSpeed
	traj.MinSpeed = minSpeed
	traj.DirectionChanges = directionChanges

	if len(speeds) > 0 {
		avgSpeed := 0.0
		for _, speed := range speeds {
			avgSpeed += speed
		}
		traj.AverageSpeed = avgSpeed / float64(len(speeds))

		variance := 0.0
		for _, speed := range speeds {
			variance += math.Pow(speed-traj.AverageSpeed, 2)
		}
		traj.SpeedVariance = variance / float64(len(speeds))
	}

	firstPoint := originalPoints[0]
	lastPoint := originalPoints[len(originalPoints)-1]
	straightDistance := math.Sqrt(
		math.Pow(float64(lastPoint.X-firstPoint.X), 2) +
			math.Pow(float64(lastPoint.Y-firstPoint.Y), 2),
	)

	if totalDistance > 0 {
		traj.PathEfficiency = straightDistance / totalDistance
	}

	if len(curvatures) > 0 {
		avgCurv := 0.0
		for _, c := range curvatures {
			avgCurv += c
		}
		traj.CurvatureAvg = avgCurv / float64(len(curvatures))
	}

	jitter := 0.0
	if totalDistance > 0 && smoothedDistance > 0 {
		jitter = (totalDistance - smoothedDistance) / totalDistance
	}
	traj.JitterScore = jitter

	accelerations := []float64{}
	for i := 2; i < len(speeds); i++ {
		dt := float64(originalPoints[i].Timestamp - originalPoints[i-2].Timestamp)
		if dt > 0 {
			accel := (speeds[i] - speeds[i-1]) / dt
			accelerations = append(accelerations, accel)
		}
	}
	if len(accelerations) > 0 {
		avgAccel := 0.0
		for _, a := range accelerations {
			avgAccel += a
		}
		traj.AccelerationAvg = avgAccel / float64(len(accelerations))
	}

	return traj
}

func (s *BehaviorAnalysisService) computeCurvature(p1, p2, p3 BehaviorDataPoint) float64 {
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

	angle := math.Acos(cosAngle)

	cross := v1x*v2y - v1y*v2x
	if cross < 0 {
		angle = -angle
	}

	return angle
}

func (s *BehaviorAnalysisService) analyzeClickPatternEnhanced(clicks []BehaviorDataPoint) ClickPattern {
	pattern := ClickPattern{
		Clicks:     clicks,
		ClickCount: len(clicks),
	}

	if len(clicks) < 2 {
		return pattern
	}

	intervals := []float64{}
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i].Timestamp - clicks[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	if len(intervals) > 0 {
		avgInterval := 0.0
		for _, interval := range intervals {
			avgInterval += interval
		}
		avgInterval = avgInterval / float64(len(intervals))
		pattern.AverageInterval = avgInterval

		variance := 0.0
		for _, interval := range intervals {
			variance += math.Pow(interval-avgInterval, 2)
		}
		variance = variance / float64(len(intervals))
		pattern.IntervalVariance = variance
		pattern.IntervalStdDev = math.Sqrt(variance)

		if avgInterval > 0 {
			pattern.Regularity = 1 - math.Min(pattern.IntervalStdDev/avgInterval, 1)
		}
	}

	totalTime := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if totalTime > 0 {
		pattern.ClickSpeed = float64(len(clicks)) / (totalTime / 1000)
	}

	buckets := 10
	pattern.XDistribution = s.computePositionDistribution(clicks, true, buckets)
	pattern.YDistribution = s.computePositionDistribution(clicks, false, buckets)
	pattern.PositionEntropy = s.computeEntropy(append(pattern.XDistribution, pattern.YDistribution...))

	minX, maxX := clicks[0].X, clicks[0].X
	minY, maxY := clicks[0].Y, clicks[0].Y
	for _, c := range clicks {
		if c.X < minX {
			minX = c.X
		}
		if c.X > maxX {
			maxX = c.X
		}
		if c.Y < minY {
			minY = c.Y
		}
		if c.Y > maxY {
			maxY = c.Y
		}
	}
	pattern.ClickAreaSize = float64((maxX-minX)*(maxY-minY)) / 10000.0

	if len(clicks) >= 2 {
		lastInterval := float64(clicks[len(clicks)-1].Timestamp - clicks[len(clicks)-2].Timestamp)
		pattern.IsDoubleClick = lastInterval < 300
	}

	return pattern
}

func (s *BehaviorAnalysisService) computePositionDistribution(clicks []BehaviorDataPoint, isX bool, buckets int) []int {
	dist := make([]int, buckets)
	if len(clicks) == 0 {
		return dist
	}

	minVal, maxVal := 0, 1000
	if isX {
		minVal, maxVal = 0, 1920
	} else {
		minVal, maxVal = 0, 1080
	}

	bucketSize := float64(maxVal-minVal) / float64(buckets)

	for _, click := range clicks {
		val := click.X
		if !isX {
			val = click.Y
		}
		bucket := int(float64(val-minVal) / bucketSize)
		if bucket >= buckets {
			bucket = buckets - 1
		}
		if bucket < 0 {
			bucket = 0
		}
		dist[bucket]++
	}

	return dist
}

func (s *BehaviorAnalysisService) computeEntropy(counts []int) float64 {
	total := 0
	for _, c := range counts {
		total += c
	}

	if total == 0 {
		return 0
	}

	entropy := 0.0
	for _, c := range counts {
		if c > 0 {
			p := float64(c) / float64(total)
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (s *BehaviorAnalysisService) analyzeKeyboardPattern(keyStrokes []KeyboardDataPoint) KeyboardPattern {
	pattern := KeyboardPattern{
		KeyStrokes:     keyStrokes,
		KeystrokeCount: len(keyStrokes),
		CommonPairs:    make(map[string]int),
		ComboPatterns:  []string{},
	}

	if len(keyStrokes) < 2 {
		return pattern
	}

	intervals := []float64{}
	holdTimes := []float64{}

	for i := 1; i < len(keyStrokes); i++ {
		interval := float64(keyStrokes[i].Timestamp - keyStrokes[i-1].Timestamp)
		intervals = append(intervals, interval)
	}

	for i := range keyStrokes {
		if keyStrokes[i].HoldDuration > 0 {
			holdTimes = append(holdTimes, float64(keyStrokes[i].HoldDuration))
		}
	}

	if len(intervals) > 0 {
		avgInterval := 0.0
		for _, interval := range intervals {
			avgInterval += interval
		}
		avgInterval = avgInterval / float64(len(intervals))
		pattern.AverageInterval = avgInterval

		variance := 0.0
		for _, interval := range intervals {
			variance += math.Pow(interval-avgInterval, 2)
		}
		variance = variance / float64(len(intervals))
		pattern.IntervalVariance = variance
		pattern.IntervalStdDev = math.Sqrt(variance)

		if avgInterval > 0 {
			pattern.Regularity = 1 - math.Min(pattern.IntervalStdDev/avgInterval, 1)
		}
	}

	if len(holdTimes) > 0 {
		avgHold := 0.0
		for _, h := range holdTimes {
			avgHold += h
		}
		pattern.AverageHoldTime = avgHold / float64(len(holdTimes))

		variance := 0.0
		for _, h := range holdTimes {
			variance += math.Pow(h-pattern.AverageHoldTime, 2)
		}
		pattern.HoldTimeVariance = variance / float64(len(holdTimes))
	}

	keyFreq := make(map[string]int)
	for _, ks := range keyStrokes {
		keyFreq[ks.Key]++
		pattern.CommonPairs[ks.Key]++
	}

	maxFreq := 0
	for key, freq := range keyFreq {
		if freq > maxFreq {
			maxFreq = freq
			pattern.MostFrequentKey = key
		}
	}

	for i := 1; i < len(keyStrokes); i++ {
		pair := keyStrokes[i-1].Key + ":" + keyStrokes[i].Key
		pattern.CommonPairs[pair]++
	}

	totalTime := float64(keyStrokes[len(keyStrokes)-1].Timestamp - keyStrokes[0].Timestamp)
	if totalTime > 0 {
		pattern.TypingSpeed = float64(len(keyStrokes)) / (totalTime / 1000)
	}

	commonCombos := []string{"ctrl:c", "ctrl:v", "ctrl:a", "ctrl:s", "ctrl:z", "ctrl:x", "shift", "alt"}
	for _, combo := range commonCombos {
		parts := strings.Split(combo, ":")
		if len(parts) == 1 {
			for _, ks := range keyStrokes {
				if ks.Key == combo {
					pattern.ComboDetected = true
					pattern.ComboPatterns = append(pattern.ComboPatterns, combo)
					break
				}
			}
		} else {
			for i := 1; i < len(keyStrokes); i++ {
				if keyStrokes[i-1].Key == parts[0] && keyStrokes[i].Key == parts[1] {
					pattern.ComboDetected = true
					pattern.ComboPatterns = append(pattern.ComboPatterns, combo)
					break
				}
			}
		}
	}

	return pattern
}

func (s *BehaviorAnalysisService) calculateRiskScoreEnhanced(result *AnalysisResult) {
	riskScore := 0.0
	indicators := []string{}
	factors := make(map[string]float64)

	if result.SpeedAnalysis.SpeedOutliers > len(result.SpeedAnalysis.Speeds)/3 {
		riskScore += 15
		indicators = append(indicators, "速度异常波动大")
		factors["speed_outliers"] = 15
	}

	if result.SpeedAnalysis.MaxSpeed > 10 {
		riskScore += 10
		indicators = append(indicators, "检测到超高速移动")
		factors["extreme_speed"] = 10
	}

	if !result.SpeedAnalysis.IsSpeedConsistent && result.SpeedAnalysis.SpeedStdDev > 0 {
		riskScore += 10
		indicators = append(indicators, "速度变化不自然")
		factors["inconsistent_speed"] = 10
	}

	if result.Trajectory.PathEfficiency > 0.95 && result.Trajectory.TotalDistance > 100 {
		riskScore += 20
		indicators = append(indicators, "路径过于笔直")
		factors["straight_path"] = 20
	}

	if result.Trajectory.JitterScore < 0.05 {
		riskScore += 15
		indicators = append(indicators, "轨迹抖动过低(机器特征)")
		factors["low_jitter"] = 15
	}

	if result.Trajectory.CurvatureAvg < 0.1 && len(result.Trajectory.Points) > 20 {
		riskScore += 15
		indicators = append(indicators, "曲率过低")
		factors["low_curvature"] = 15
	}

	if result.PathSimilarity.IsPathRepeated {
		riskScore += 30
		indicators = append(indicators, "路径重复检测")
		factors["path_repeat"] = 30
	}

	if result.PathSimilarity.PathHashMatch {
		riskScore += 25
		indicators = append(indicators, "路径哈希匹配")
		factors["path_hash_match"] = 25
	}

	if result.PathSimilarity.DTWDistance < 50 && result.PathSimilarity.ComparedPathLength > 10 {
		riskScore += 20
		indicators = append(indicators, "DTW距离异常小")
		factors["low_dtw"] = 20
	}

	if result.ClickPattern.Regularity > 0.9 && result.ClickPattern.ClickCount > 2 {
		riskScore += 15
		indicators = append(indicators, "点击间隔过于规律")
		factors["regular_clicks"] = 15
	}

	if result.ClickPattern.PositionEntropy < 2.0 && result.ClickPattern.ClickCount > 3 {
		riskScore += 10
		indicators = append(indicators, "点击位置集中")
		factors["clustered_clicks"] = 10
	}

	if result.ClickPattern.ClickAreaSize < 5.0 && result.ClickPattern.ClickCount > 3 {
		riskScore += 10
		indicators = append(indicators, "点击区域过小")
		factors["small_click_area"] = 10
	}

	if result.ClickPattern.IsDoubleClick {
		riskScore += 5
		indicators = append(indicators, "快速双击")
		factors["double_click"] = 5
	}

	if len(result.KeyboardPattern.KeyStrokes) > 0 {
		if result.KeyboardPattern.TypingSpeed > 15 {
			riskScore += 15
			indicators = append(indicators, "打字速度异常快")
			factors["fast_typing"] = 15
		}

		if result.KeyboardPattern.AverageHoldTime < 50 {
			riskScore += 10
			indicators = append(indicators, "按键保持时间过短")
			factors["short_hold"] = 10
		}

		if result.KeyboardPattern.Regularity > 0.95 {
			riskScore += 10
			indicators = append(indicators, "按键间隔过于规律")
			factors["regular_typing"] = 10
		}
	}

	if len(result.Trajectory.Points) < 10 {
		riskScore += 10
		indicators = append(indicators, "行为数据点过少")
		factors["insufficient_data"] = 10
	}

	if len(result.Trajectory.Points) > 500 {
		riskScore += 5
		indicators = append(indicators, "数据点异常多")
		factors["excessive_data"] = 5
	}

	if len(result.DwellPoints) > 0 {
		suspiciousDwellCount := 0
		for _, dwell := range result.DwellPoints {
			if dwell.IsSuspicious {
				suspiciousDwellCount++
			}
		}
		if suspiciousDwellCount > len(result.DwellPoints)/2 {
			riskScore += 15
			indicators = append(indicators, "停留点异常")
			factors["abnormal_dwell"] = 15
		}
	}

	if result.BehaviorFeatures.IsAnomalous {
		riskScore += 20
		indicators = append(indicators, "行为特征异常")
		factors["behavior_anomaly"] = 20
	}

	result.RiskScore = math.Min(riskScore, 100)
	result.RiskIndicators = indicators
	result.RiskFactors = factors
	result.IsBotLikely = riskScore >= 50
	result.Confidence = math.Min(riskScore/100+0.3, 0.95)
}

func (s *BehaviorAnalysisService) CalculateRiskScore(verification *models.Verification, behaviorData []models.BehaviorData) float64 {
	result, err := s.AnalyzeBehavior(behaviorData)
	if err != nil {
		return 50.0
	}
	return result.RiskScore
}

func (s *BehaviorAnalysisService) GenerateAnalysisReport(result *AnalysisResult) string {
	report := fmt.Sprintf("行为分析报告:\n")
	report += fmt.Sprintf("- 风险评分: %.2f\n", result.RiskScore)
	report += fmt.Sprintf("- 疑似机器人: %v\n", result.IsBotLikely)
	report += fmt.Sprintf("- 置信度: %.2f\n", result.Confidence)
	report += fmt.Sprintf("- 风险指标:\n")
	for _, indicator := range result.RiskIndicators {
		report += fmt.Sprintf("  * %s\n", indicator)
	}
	report += fmt.Sprintf("- 轨迹分析:\n")
	report += fmt.Sprintf("  * 总距离: %.2f\n", result.Trajectory.TotalDistance)
	report += fmt.Sprintf("  * 平均速度: %.6f\n", result.Trajectory.AverageSpeed)
	report += fmt.Sprintf("  * 最大速度: %.6f\n", result.Trajectory.MaxSpeed)
	report += fmt.Sprintf("  * 最小速度: %.6f\n", result.Trajectory.MinSpeed)
	report += fmt.Sprintf("  * 路径效率: %.4f\n", result.Trajectory.PathEfficiency)
	report += fmt.Sprintf("  * 平滑距离: %.2f\n", result.Trajectory.SmoothedDistance)
	report += fmt.Sprintf("  * 速度方差: %.6f\n", result.Trajectory.SpeedVariance)
	report += fmt.Sprintf("  * 平均曲率: %.6f\n", result.Trajectory.CurvatureAvg)
	report += fmt.Sprintf("  * 抖动分数: %.4f\n", result.Trajectory.JitterScore)
	report += fmt.Sprintf("  * 方向变化: %d\n", result.Trajectory.DirectionChanges)

	if len(result.DwellPoints) > 0 {
		report += fmt.Sprintf("- 停留点分析:\n")
		report += fmt.Sprintf("  * 停留点数量: %d\n", len(result.DwellPoints))
		suspiciousCount := 0
		for _, dwell := range result.DwellPoints {
			if dwell.IsSuspicious {
				suspiciousCount++
			}
		}
		report += fmt.Sprintf("  * 可疑停留点: %d\n", suspiciousCount)
	}

	if result.SpeedAnalysis.AverageSpeed > 0 {
		report += fmt.Sprintf("- 速度分析:\n")
		report += fmt.Sprintf("  * 平均速度: %.6f\n", result.SpeedAnalysis.AverageSpeed)
		report += fmt.Sprintf("  * 中位速度: %.6f\n", result.SpeedAnalysis.MedianSpeed)
		report += fmt.Sprintf("  * 速度标准差: %.6f\n", result.SpeedAnalysis.SpeedStdDev)
		report += fmt.Sprintf("  * 速度偏度: %.6f\n", result.SpeedAnalysis.SpeedSkewness)
		report += fmt.Sprintf("  * 平均加速度: %.6f\n", result.SpeedAnalysis.AverageAcceleration)
		report += fmt.Sprintf("  * 最大加速度: %.6f\n", result.SpeedAnalysis.MaxAcceleration)
		report += fmt.Sprintf("  * 速度一致性: %v\n", result.SpeedAnalysis.IsSpeedConsistent)
		report += fmt.Sprintf("  * 速度异常点: %d\n", result.SpeedAnalysis.SpeedOutliers)
	}

	if result.PathSimilarity.ComparedPathLength > 0 {
		report += fmt.Sprintf("- 路径相似度:\n")
		report += fmt.Sprintf("  * 相似度分数: %.4f\n", result.PathSimilarity.SimilarityScore)
		report += fmt.Sprintf("  * 路径重复: %v\n", result.PathSimilarity.IsPathRepeated)
		report += fmt.Sprintf("  * 路径哈希匹配: %v\n", result.PathSimilarity.PathHashMatch)
		report += fmt.Sprintf("  * DTW距离: %.2f\n", result.PathSimilarity.DTWDistance)
		report += fmt.Sprintf("  * Fréchet距离: %.2f\n", result.PathSimilarity.FrechetDistance)
	}

	report += fmt.Sprintf("- 点击模式:\n")
	report += fmt.Sprintf("  * 点击次数: %d\n", result.ClickPattern.ClickCount)
	report += fmt.Sprintf("  * 平均间隔: %.2fms\n", result.ClickPattern.AverageInterval)
	report += fmt.Sprintf("  * 间隔方差: %.2f\n", result.ClickPattern.IntervalVariance)
	report += fmt.Sprintf("  * 点击速度: %.2f点击/秒\n", result.ClickPattern.ClickSpeed)
	report += fmt.Sprintf("  * 规律性: %.4f\n", result.ClickPattern.Regularity)
	report += fmt.Sprintf("  * 位置熵: %.4f\n", result.ClickPattern.PositionEntropy)
	report += fmt.Sprintf("  * 点击区域: %.2f\n", result.ClickPattern.ClickAreaSize)
	report += fmt.Sprintf("  * 双击: %v\n", result.ClickPattern.IsDoubleClick)

	if len(result.KeyboardPattern.KeyStrokes) > 0 {
		report += fmt.Sprintf("- 键盘模式:\n")
		report += fmt.Sprintf("  * 按键次数: %d\n", result.KeyboardPattern.KeystrokeCount)
		report += fmt.Sprintf("  * 平均间隔: %.2fms\n", result.KeyboardPattern.AverageInterval)
		report += fmt.Sprintf("  * 平均保持时间: %.2fms\n", result.KeyboardPattern.AverageHoldTime)
		report += fmt.Sprintf("  * 打字速度: %.2f字符/秒\n", result.KeyboardPattern.TypingSpeed)
		report += fmt.Sprintf("  * 规律性: %.4f\n", result.KeyboardPattern.Regularity)
		report += fmt.Sprintf("  * 最常用键: %s\n", result.KeyboardPattern.MostFrequentKey)
		report += fmt.Sprintf("  * 快捷键检测: %v\n", result.KeyboardPattern.ComboDetected)
		if len(result.KeyboardPattern.ComboPatterns) > 0 {
			report += fmt.Sprintf("  * 检测到的组合: %v\n", result.KeyboardPattern.ComboPatterns)
		}
	}

	if len(result.BehaviorFeatures.FeatureVector) > 0 {
		report += fmt.Sprintf("- 行为特征:\n")
		report += fmt.Sprintf("  * 特征向量长度: %d\n", len(result.BehaviorFeatures.FeatureVector))
		report += fmt.Sprintf("  * 异常评分: %.4f\n", result.BehaviorFeatures.AnomalyScore)
		report += fmt.Sprintf("  * 是否异常: %v\n", result.BehaviorFeatures.IsAnomalous)
		if len(result.BehaviorFeatures.AnomalyIndicators) > 0 {
			report += fmt.Sprintf("  * 异常指标: %v\n", result.BehaviorFeatures.AnomalyIndicators)
		}
	}

	return report
}

func (s *BehaviorAnalysisService) VerifyWithBehaviorAnalysis(
	captchaSuccess bool,
	behaviorData []models.BehaviorData,
) (bool, float64, string) {
	result, _ := s.AnalyzeBehavior(behaviorData)

	analysisReport := s.GenerateAnalysisReport(result)

	var finalResult bool
	if result.RiskScore < 30 {
		finalResult = captchaSuccess
	} else if result.RiskScore < 70 {
		finalResult = captchaSuccess && result.RiskScore < 50
	} else {
		finalResult = false
	}

	return finalResult, result.RiskScore, analysisReport
}

func (s *BehaviorAnalysisService) AnalyzeSpeed(behaviorData []models.BehaviorData) (*SpeedAnalysis, error) {
	var points []BehaviorDataPoint
	for _, bd := range behaviorData {
		var dp BehaviorDataPoint
		if err := json.Unmarshal([]byte(bd.Data), &dp); err == nil {
			points = append(points, dp)
		}
	}

	analysis := s.analyzeSpeed(points)
	return &analysis, nil
}

func (s *BehaviorAnalysisService) AnalyzePathSimilarity(path1, path2 []BehaviorDataPoint) *PathSimilarity {
	similarity := PathSimilarity{
		ComparedPathLength: len(path1),
	}

	if len(path1) < 5 || len(path2) < 5 {
		return &similarity
	}

	similarity.DTWDistance = s.computeDTWDistance(path1, path2)
	similarity.FrechetDistance = s.computeFrechetDistance(path1, path2)
	similarity.SimilarityScore = s.computePathCorrelation(path1, path2)

	similarity.IsPathRepeated = similarity.SimilarityScore > 0.85

	return &similarity
}

func (s *BehaviorAnalysisService) detectDwellPoints(points []BehaviorDataPoint) []DwellPoint {
	dwellPoints := []DwellPoint{}
	if len(points) < 3 {
		return dwellPoints
	}

	speedThreshold := 0.5
	minDwellDuration := int64(200)

	var currentDwell *DwellPoint
	for i := 0; i < len(points)-1; i++ {
		dx := float64(points[i+1].X - points[i].X)
		dy := float64(points[i+1].Y - points[i].Y)
		distance := math.Sqrt(dx*dx + dy*dy)
		dt := float64(points[i+1].Timestamp - points[i].Timestamp)

		var speed float64
		if dt > 0 {
			speed = distance / dt
		}

		if speed < speedThreshold {
			if currentDwell == nil {
				currentDwell = &DwellPoint{
					StartTime:  points[i].Timestamp,
					EndTime:    points[i].Timestamp,
					CenterX:    points[i].X,
					CenterY:    points[i].Y,
					PointCount: 1,
					AvgSpeed:   speed,
				}
			} else {
				currentDwell.EndTime = points[i].Timestamp
				currentDwell.PointCount++
				totalX, totalY := 0, 0
				for j := 0; j <= i; j++ {
					totalX += points[j].X
					totalY += points[j].Y
				}
				currentDwell.CenterX = totalX / (i + 1)
				currentDwell.CenterY = totalY / (i + 1)
				currentDwell.AvgSpeed = (currentDwell.AvgSpeed*float64(currentDwell.PointCount-1) + speed) / float64(currentDwell.PointCount)
			}
		} else {
			if currentDwell != nil {
				currentDwell.Duration = currentDwell.EndTime - currentDwell.StartTime
				currentDwell.IsSuspicious = currentDwell.Duration < minDwellDuration
				dwellPoints = append(dwellPoints, *currentDwell)
				currentDwell = nil
			}
		}
	}

	if currentDwell != nil {
		currentDwell.Duration = currentDwell.EndTime - currentDwell.StartTime
		currentDwell.IsSuspicious = currentDwell.Duration < minDwellDuration
		dwellPoints = append(dwellPoints, *currentDwell)
	}

	return dwellPoints
}

func (s *BehaviorAnalysisService) extractBehaviorFeatures(result *AnalysisResult) BehaviorFeatures {
	features := BehaviorFeatures{
		TrajectoryFeatures: make([]float64, 0),
		SpeedFeatures:     make([]float64, 0),
		ClickFeatures:     make([]float64, 0),
		KeyboardFeatures:  make([]float64, 0),
		AnomalyIndicators: []string{},
	}

	features.TrajectoryFeatures = append(features.TrajectoryFeatures,
		result.Trajectory.TotalDistance,
		result.Trajectory.AverageSpeed,
		result.Trajectory.MaxSpeed,
		result.Trajectory.PathEfficiency,
		result.Trajectory.JitterScore,
		result.Trajectory.CurvatureAvg,
		float64(len(result.Trajectory.Points)),
	)

	if len(result.SpeedAnalysis.Speeds) > 0 {
		features.SpeedFeatures = append(features.SpeedFeatures,
			result.SpeedAnalysis.AverageSpeed,
			result.SpeedAnalysis.MaxSpeed,
			result.SpeedAnalysis.MedianSpeed,
			result.SpeedAnalysis.SpeedStdDev,
			result.SpeedAnalysis.AverageAcceleration,
			result.SpeedAnalysis.JerkAvg,
		)
	}

	features.ClickFeatures = append(features.ClickFeatures,
		float64(result.ClickPattern.ClickCount),
		result.ClickPattern.AverageInterval,
		result.ClickPattern.ClickSpeed,
		result.ClickPattern.Regularity,
		result.ClickPattern.PositionEntropy,
	)

	if len(result.KeyboardPattern.KeyStrokes) > 0 {
		features.KeyboardFeatures = append(features.KeyboardFeatures,
			float64(result.KeyboardPattern.KeystrokeCount),
			result.KeyboardPattern.AverageInterval,
			result.KeyboardPattern.AverageHoldTime,
			result.KeyboardPattern.TypingSpeed,
		)
	}

	allFeatures := append(features.TrajectoryFeatures, features.SpeedFeatures...)
	allFeatures = append(allFeatures, features.ClickFeatures...)
	allFeatures = append(allFeatures, features.KeyboardFeatures...)
	features.FeatureVector = allFeatures

	features.AnomalyScore = s.calculateAnomalyScore(result)
	features.IsAnomalous = features.AnomalyScore > 0.7

	if features.IsAnomalous {
		if result.Trajectory.JitterScore < 0.05 {
			features.AnomalyIndicators = append(features.AnomalyIndicators, "low_jitter")
		}
		if result.Trajectory.PathEfficiency > 0.95 {
			features.AnomalyIndicators = append(features.AnomalyIndicators, "straight_path")
		}
		if result.SpeedAnalysis.MaxSpeed > 10 {
			features.AnomalyIndicators = append(features.AnomalyIndicators, "extreme_speed")
		}
	}

	return features
}

func (s *BehaviorAnalysisService) calculateAnomalyScore(result *AnalysisResult) float64 {
	score := 0.0

	if result.Trajectory.JitterScore < 0.05 {
		score += 0.2
	}
	if result.Trajectory.PathEfficiency > 0.95 {
		score += 0.2
	}
	if result.SpeedAnalysis.MaxSpeed > 10 {
		score += 0.15
	}
	if result.PathSimilarity.SimilarityScore > 0.85 {
		score += 0.25
	}
	if result.ClickPattern.Regularity > 0.9 {
		score += 0.1
	}
	if result.RiskScore > 50 {
		score += 0.1
	}

	return math.Min(score, 1.0)
}

func (s *BehaviorAnalysisService) generateCacheKey(behaviorData []models.BehaviorData) string {
	if len(behaviorData) == 0 {
		return "empty"
	}

	var hashParts []string
	for _, bd := range behaviorData {
		hashParts = append(hashParts, fmt.Sprintf("%s:%s", bd.DataType, bd.Data))
	}
	return strings.Join(hashParts, "|")
}

func (s *BehaviorAnalysisService) evictOldestCache() {
	if len(s.cache) == 0 {
		return
	}

	oldestKey := ""

	for key := range s.cache {
		oldestKey = key
		break
	}

	if oldestKey != "" {
		delete(s.cache, oldestKey)
	}
}

func (s *BehaviorAnalysisService) ClearCache() {
	s.cacheMutex.Lock()
	defer s.cacheMutex.Unlock()
	s.cache = make(map[string]*AnalysisResult)
}

func (s *BehaviorAnalysisService) GetCacheSize() int {
	s.cacheMutex.RLock()
	defer s.cacheMutex.RUnlock()
	return len(s.cache)
}

func (s *BehaviorAnalysisService) GetAnalysisCount() int64 {
	return atomic.LoadInt64(&s.analysisCount)
}

func (s *BehaviorAnalysisService) mean(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}

func (s *BehaviorAnalysisService) variance(values []float64) float64 {
	if len(values) < 2 {
		return 0
	}
	mean := s.mean(values)
	sum := 0.0
	for _, v := range values {
		sum += math.Pow(v-mean, 2)
	}
	return sum / float64(len(values))
}

func (s *BehaviorAnalysisService) max(values []float64) float64 {
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

func (s *BehaviorAnalysisService) min(values []float64) float64 {
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

func (s *BehaviorAnalysisService) maxAbs(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	max := math.Abs(values[0])
	for _, v := range values {
		if math.Abs(v) > max {
			max = math.Abs(v)
		}
	}
	return max
}

func (s *BehaviorAnalysisService) median(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sorted := make([]float64, len(values))
	copy(sorted, values)
	sort.Float64s(sorted)
	n := len(sorted)
	if n%2 == 0 {
		return (sorted[n/2-1] + sorted[n/2]) / 2
	}
	return sorted[n/2]
}

func (s *BehaviorAnalysisService) skewness(values []float64) float64 {
	if len(values) < 3 {
		return 0
	}
	mean := s.mean(values)
	stdDev := math.Sqrt(s.variance(values))
	if stdDev == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += math.Pow((v-mean)/stdDev, 3)
	}
	return sum / float64(len(values))
}
