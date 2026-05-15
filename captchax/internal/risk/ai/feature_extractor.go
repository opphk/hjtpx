package ai

import (
	"math"

	"captchax/internal/risk"
)

const (
	FeatureDimension = 25
)

type MouseTrack struct {
	X           float64
	Y           float64
	Timestamp   int64
	Velocity    float64
	Acceleration float64
	Pressure    float64
	EventType   string
}

type ClickEvent struct {
	Timestamp   int64
	X           float64
	Y           float64
	Pressure    float64
	Duration    int64
}

type HesitationPoint struct {
	X           float64
	Y           float64
	Timestamp   int64
	Duration    int64
}

type Point struct {
	X float64
	Y float64
}

type BehaviorData struct {
	UserID            string
	SessionID         string
	MouseTracks       []MouseTrack
	ClickEvents       []ClickEvent
	ClickTimes        []int64
	SlideStart        int64
	SlideEnd          int64
	SlidePath         []Point
	Success           bool
	HesitationPoints  []HesitationPoint
	DeviceOrientation []Point
}

type FeatureExtractor struct {
	enableAdvancedFeatures bool
	normalizationParams    *NormalizationParams
}

type NormalizationParams struct {
	Mean  []float64
	Std   []float64
	Min   []float64
	Max   []float64
}

type ExtractedFeatures struct {
	Features   []float64
	Dimension  int
	Valid      bool
	RawMetrics *RawMetrics
}

type RawMetrics struct {
	TrackCount       int
	ClickCount       int
	TotalDuration    int64
	TotalDistance    float64
	AvgVelocity      float64
	MaxVelocity      float64
	MinVelocity      float64
	VelocityStd      float64
	Smoothness       float64
	Jitter           float64
	AccelerationAvg  float64
	DirectionChanges int
	CurvatureAvg     float64
	Complexity       float64
}

func NewFeatureExtractor() *FeatureExtractor {
	return &FeatureExtractor{
		enableAdvancedFeatures: true,
		normalizationParams:    nil,
	}
}

func (fe *FeatureExtractor) Extract(behavior *BehaviorData) *ExtractedFeatures {
	metrics := fe.calculateRawMetrics(behavior)

	features := make([]float64, FeatureDimension)

	features[0] = float64(len(behavior.MouseTracks)) / 100.0
	features[1] = float64(len(behavior.ClickEvents)) / 10.0
	features[2] = float64(len(behavior.HesitationPoints)) / 5.0

	if behavior.SlideEnd > behavior.SlideStart {
		duration := float64(behavior.SlideEnd-behavior.SlideStart) / 1000.0
		features[3] = math.Min(duration/30.0, 1.0)
	} else {
		features[3] = 0.0
	}

	features[4] = metrics.Smoothness
	features[5] = metrics.Jitter
	features[6] = math.Min(metrics.AccelerationAvg/100.0, 1.0)
	features[7] = math.Min(metrics.VelocityStd/50.0, 1.0)

	features[8] = float64(metrics.DirectionChanges) / 50.0
	features[9] = metrics.CurvatureAvg
	features[10] = metrics.Complexity

	features[11] = math.Min(metrics.AvgVelocity/200.0, 1.0)
	features[12] = math.Min(metrics.MaxVelocity/500.0, 1.0)

	if metrics.TotalDistance > 0 {
		features[13] = metrics.TotalDistance / 1000.0
	} else {
		features[13] = 0.0
	}

	features[14] = float64(len(behavior.MouseTracks)) / float64(metrics.TotalDuration+1)
	features[15] = float64(len(behavior.ClickEvents)) / float64(metrics.TotalDuration+1)

	if len(behavior.MouseTracks) > 0 {
		firstTrack := behavior.MouseTracks[0]
		lastTrack := behavior.MouseTracks[len(behavior.MouseTracks)-1]
		directDistance := math.Sqrt(
			math.Pow(lastTrack.X-firstTrack.X, 2) +
				math.Pow(lastTrack.Y-firstTrack.Y, 2),
		)
		if directDistance > 0 {
			features[16] = metrics.TotalDistance / directDistance
		} else {
			features[16] = 1.0
		}
	} else {
		features[16] = 1.0
	}

	if len(behavior.ClickEvents) > 0 {
		var pressureSum float64
		for _, click := range behavior.ClickEvents {
			pressureSum += click.Pressure
		}
		features[17] = pressureSum / float64(len(behavior.ClickEvents))
	} else {
		features[17] = 0.5
	}

	if len(behavior.ClickEvents) > 0 {
		var durationSum float64
		for _, click := range behavior.ClickEvents {
			durationSum += float64(click.Duration)
		}
		features[18] = durationSum / float64(len(behavior.ClickEvents)) / 1000.0
	} else {
		features[18] = 0.0
	}

	features[19] = 0.0
	for _, h := range behavior.HesitationPoints {
		features[19] += float64(h.Duration)
	}
	features[19] /= 1000.0

	if len(behavior.DeviceOrientation) > 0 {
		features[20] = float64(len(behavior.DeviceOrientation)) / 20.0
	} else {
		features[20] = 0.0
	}

	features[21] = 0.0
	if len(behavior.MouseTracks) > 2 {
		for i := 2; i < len(behavior.MouseTracks); i++ {
			dx1 := behavior.MouseTracks[i-1].X - behavior.MouseTracks[i-2].X
			dy1 := behavior.MouseTracks[i-1].Y - behavior.MouseTracks[i-2].Y
			dx2 := behavior.MouseTracks[i].X - behavior.MouseTracks[i-1].X
			dy2 := behavior.MouseTracks[i].Y - behavior.MouseTracks[i-1].Y
			dot := dx1*dx2 + dy1*dy2
			mag1 := math.Sqrt(dx1*dx1 + dy1*dy1)
			mag2 := math.Sqrt(dx2*dx2 + dy2*dy2)
			if mag1 > 0 && mag2 > 0 {
				cosAngle := dot / (mag1 * mag2)
				if cosAngle > 1 {
					cosAngle = 1
				}
				if cosAngle < -1 {
					cosAngle = -1
				}
				features[21] += math.Acos(cosAngle)
			}
		}
		features[21] /= float64(len(behavior.MouseTracks) - 2)
	}

	features[22] = 0.0
	if behavior.Success {
		features[22] = 1.0
	}

	features[23] = 0.0
	if len(behavior.MouseTracks) > 0 {
		validVelocityCount := 0
		for _, track := range behavior.MouseTracks {
			if track.Velocity > 0 {
				validVelocityCount++
			}
		}
		features[23] = float64(validVelocityCount) / float64(len(behavior.MouseTracks))
	}

	features[24] = 0.0
	if len(behavior.MouseTracks) > 0 {
		var xMin, xMax, yMin, yMax float64 = math.MaxFloat64, -math.MaxFloat64, math.MaxFloat64, -math.MaxFloat64
		for _, track := range behavior.MouseTracks {
			if track.X < xMin {
				xMin = track.X
			}
			if track.X > xMax {
				xMax = track.X
			}
			if track.Y < yMin {
				yMin = track.Y
			}
			if track.Y > yMax {
				yMax = track.Y
			}
		}
		coverage := (xMax - xMin) * (yMax - yMin)
		features[24] = coverage / 40000.0
		if features[24] > 1.0 {
			features[24] = 1.0
		}
	}

	return &ExtractedFeatures{
		Features:  features,
		Dimension: FeatureDimension,
		Valid:     true,
		RawMetrics: metrics,
	}
}

func (fe *FeatureExtractor) calculateRawMetrics(behavior *BehaviorData) *RawMetrics {
	metrics := &RawMetrics{}

	if len(behavior.MouseTracks) == 0 {
		return metrics
	}

	metrics.TrackCount = len(behavior.MouseTracks)
	metrics.ClickCount = len(behavior.ClickEvents)

	if behavior.SlideEnd > behavior.SlideStart {
		metrics.TotalDuration = behavior.SlideEnd - behavior.SlideStart
	} else if len(behavior.MouseTracks) > 1 {
		metrics.TotalDuration = behavior.MouseTracks[len(behavior.MouseTracks)-1].Timestamp - behavior.MouseTracks[0].Timestamp
	}

	var velocities []float64
	var accelerations []float64
	var totalDistance float64
	directionChanges := 0
	var curvatureSum float64

	for i := 1; i < len(behavior.MouseTracks); i++ {
		dx := behavior.MouseTracks[i].X - behavior.MouseTracks[i-1].X
		dy := behavior.MouseTracks[i].Y - behavior.MouseTracks[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance

		dt := float64(behavior.MouseTracks[i].Timestamp - behavior.MouseTracks[i-1].Timestamp)
		if dt > 0 {
			velocity := distance / dt
			velocities = append(velocities, velocity)

			if len(velocities) >= 2 {
				acceleration := math.Abs(velocity - velocities[len(velocities)-1])
				accelerations = append(accelerations, acceleration)
			}
		}

		if i >= 2 {
			angle1 := math.Atan2(
				behavior.MouseTracks[i-1].Y-behavior.MouseTracks[i-2].Y,
				behavior.MouseTracks[i-1].X-behavior.MouseTracks[i-2].X,
			)
			angle2 := math.Atan2(
				behavior.MouseTracks[i].Y-behavior.MouseTracks[i-1].Y,
				behavior.MouseTracks[i].X-behavior.MouseTracks[i-1].X,
			)
			angleDiff := math.Abs(angle2 - angle1)
			if angleDiff > math.Pi {
				angleDiff = 2*math.Pi - angleDiff
			}
			if angleDiff > 0.5 {
				directionChanges++
			}

			p1 := Point{behavior.MouseTracks[i-2].X, behavior.MouseTracks[i-2].Y}
			p2 := Point{behavior.MouseTracks[i-1].X, behavior.MouseTracks[i-1].Y}
			p3 := Point{behavior.MouseTracks[i].X, behavior.MouseTracks[i].Y}
			curvature := calculateThreePointCurvature(p1, p2, p3)
			curvatureSum += curvature
		}
	}

	metrics.TotalDistance = totalDistance

	if len(velocities) > 0 {
		var velocitySum float64
		for _, v := range velocities {
			velocitySum += v
		}
		metrics.AvgVelocity = velocitySum / float64(len(velocities))

		metrics.MaxVelocity = velocities[0]
		metrics.MinVelocity = velocities[0]
		for _, v := range velocities {
			if v > metrics.MaxVelocity {
				metrics.MaxVelocity = v
			}
			if v < metrics.MinVelocity {
				metrics.MinVelocity = v
			}
		}

		var varianceSum float64
		for _, v := range velocities {
			diff := v - metrics.AvgVelocity
			varianceSum += diff * diff
		}
		metrics.VelocityStd = math.Sqrt(varianceSum / float64(len(velocities)))
	}

	if len(accelerations) > 0 {
		var accelSum float64
		for _, a := range accelerations {
			accelSum += a
		}
		metrics.AccelerationAvg = accelSum / float64(len(accelerations))
	}

	metrics.Smoothness = calculateSmoothness(behavior.MouseTracks)
	metrics.Jitter = calculateJitter(behavior.MouseTracks)
	metrics.DirectionChanges = directionChanges

	if len(behavior.MouseTracks) > 2 {
		metrics.CurvatureAvg = curvatureSum / float64(len(behavior.MouseTracks)-2)
	}

	metrics.Complexity = calculateComplexity(behavior.MouseTracks)

	return metrics
}

func calculateThreePointCurvature(p1, p2, p3 Point) float64 {
	area := math.Abs((p2.X-p1.X)*(p3.Y-p1.Y)-(p2.Y-p1.Y)*(p3.X-p1.X)) / 2

	a := math.Sqrt(math.Pow(p2.X-p1.X, 2) + math.Pow(p2.Y-p1.Y, 2))
	b := math.Sqrt(math.Pow(p3.X-p2.X, 2) + math.Pow(p3.Y-p2.Y, 2))
	c := math.Sqrt(math.Pow(p3.X-p1.X, 2) + math.Pow(p3.Y-p1.Y, 2))

	if a == 0 || b == 0 || c == 0 {
		return 0
	}

	if area == 0 {
		return 1.0
	}

	R := (a * b * c) / (4 * area)
	if R == 0 {
		return 1.0
	}

	return 1.0 / (1.0 + R/100.0)
}

func calculateSmoothness(tracks []MouseTrack) float64 {
	if len(tracks) < 3 {
		return 1.0
	}

	var totalAngleChange float64
	for i := 2; i < len(tracks); i++ {
		angle1 := math.Atan2(
			tracks[i-1].Y-tracks[i-2].Y,
			tracks[i-1].X-tracks[i-2].X,
		)
		angle2 := math.Atan2(
			tracks[i].Y-tracks[i-1].Y,
			tracks[i].X-tracks[i-1].X,
		)
		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}
		totalAngleChange += angleDiff
	}

	maxPossibleChange := float64(len(tracks)-1) * math.Pi
	if maxPossibleChange == 0 {
		return 1.0
	}

	return 1.0 - (totalAngleChange / maxPossibleChange)
}

func calculateJitter(tracks []MouseTrack) float64 {
	if len(tracks) < 3 {
		return 0.0
	}

	var jitterSum float64
	count := 0

	for i := 2; i < len(tracks); i++ {
		v1x := tracks[i-1].X - tracks[i-2].X
		v1y := tracks[i-1].Y - tracks[i-2].Y
		v2x := tracks[i].X - tracks[i-1].X
		v2y := tracks[i].Y - tracks[i-1].Y

		dx := v2x - v1x
		dy := v2y - v1y
		jitter := math.Sqrt(dx*dx + dy*dy)

		jitterSum += jitter
		count++
	}

	if count == 0 {
		return 0.0
	}

	return jitterSum / float64(count)
}

func calculateComplexity(tracks []MouseTrack) float64 {
	if len(tracks) < 2 {
		return 0
	}

	totalLength := 0.0
	for i := 1; i < len(tracks); i++ {
		dx := tracks[i].X - tracks[i-1].X
		dy := tracks[i].Y - tracks[i-1].Y
		totalLength += math.Sqrt(dx*dx + dy*dy)
	}

	firstTrack := tracks[0]
	lastTrack := tracks[len(tracks)-1]
	straightLineLength := math.Sqrt(
		math.Pow(lastTrack.X-firstTrack.X, 2) +
			math.Pow(lastTrack.Y-firstTrack.Y, 2),
	)

	if totalLength == 0 {
		return 0
	}

	complexity := totalLength / (straightLineLength + 1.0)
	return 1.0 - 1.0/(1.0+complexity)
}

func (fe *FeatureExtractor) SetNormalization(params *NormalizationParams) {
	fe.normalizationParams = params
}

func (fe *FeatureExtractor) GetNormalization() *NormalizationParams {
	return fe.normalizationParams
}

func (fe *FeatureExtractor) Normalize(features []float64) []float64 {
	if fe.normalizationParams == nil {
		return features
	}

	normalized := make([]float64, len(features))

	for i := 0; i < len(features) && i < len(fe.normalizationParams.Std); i++ {
		if fe.normalizationParams.Std[i] > Epsilon {
			normalized[i] = (features[i] - fe.normalizationParams.Mean[i]) / fe.normalizationParams.Std[i]
		} else {
			normalized[i] = features[i]
		}
	}

	return normalized
}

func (fe *FeatureExtractor) Denormalize(normalizedFeatures []float64) []float64 {
	if fe.normalizationParams == nil {
		return normalizedFeatures
	}

	denormalized := make([]float64, len(normalizedFeatures))

	for i := 0; i < len(normalizedFeatures) && i < len(fe.normalizationParams.Std); i++ {
		denormalized[i] = normalizedFeatures[i]*fe.normalizationParams.Std[i] + fe.normalizationParams.Mean[i]
	}

	return denormalized
}

func CalculateNormalizationParams(samples [][]float64) *NormalizationParams {
	if len(samples) == 0 || len(samples[0]) == 0 {
		return nil
	}

	dimension := len(samples[0])
	params := &NormalizationParams{
		Mean: make([]float64, dimension),
		Std:  make([]float64, dimension),
		Min:  make([]float64, dimension),
		Max:  make([]float64, dimension),
	}

	for j := 0; j < dimension; j++ {
		params.Min[j] = samples[0][j]
		params.Max[j] = samples[0][j]
	}

	for _, sample := range samples {
		for j := 0; j < dimension && j < len(sample); j++ {
			params.Mean[j] += sample[j]
			if sample[j] < params.Min[j] {
				params.Min[j] = sample[j]
			}
			if sample[j] > params.Max[j] {
				params.Max[j] = sample[j]
			}
		}
	}

	for j := 0; j < dimension; j++ {
		params.Mean[j] /= float64(len(samples))
	}

	for _, sample := range samples {
		for j := 0; j < dimension && j < len(sample); j++ {
			diff := sample[j] - params.Mean[j]
			params.Std[j] += diff * diff
		}
	}

	for j := 0; j < dimension; j++ {
		params.Std[j] = math.Sqrt(params.Std[j] / float64(len(samples)))
		if params.Std[j] < Epsilon {
			params.Std[j] = 1.0
		}
	}

	return params
}

func (fe *FeatureExtractor) BatchExtract(behaviors []*BehaviorData) [][]float64 {
	featuresBatch := make([][]float64, len(behaviors))
	for i, behavior := range behaviors {
		extracted := fe.Extract(behavior)
		featuresBatch[i] = extracted.Features
	}
	return featuresBatch
}

func (fe *FeatureExtractor) ExtractFromMultiple(behaviors []*BehaviorData) []*ExtractedFeatures {
	results := make([]*ExtractedFeatures, len(behaviors))
	for i, behavior := range behaviors {
		results[i] = fe.Extract(behavior)
	}
	return results
}

func ConvertFromRiskBehavior(from *risk.BehaviorData) *BehaviorData {
	if from == nil {
		return nil
	}

	data := &BehaviorData{
		UserID:       from.UserID,
		SessionID:    from.SessionID,
		SlideStart:   from.SlideStart,
		SlideEnd:     from.SlideEnd,
		Success:      from.Success,
		ClickTimes:   make([]int64, len(from.ClickTimes)),
	}

	copy(data.ClickTimes, from.ClickTimes)

	data.MouseTracks = make([]MouseTrack, len(from.MouseTracks))
	for i, track := range from.MouseTracks {
		data.MouseTracks[i] = MouseTrack{
			X:           track.X,
			Y:           track.Y,
			Timestamp:   track.Timestamp,
			Velocity:    track.Velocity,
			Acceleration: track.Acceleration,
			Pressure:    track.Pressure,
			EventType:   track.EventType,
		}
	}

	data.ClickEvents = make([]ClickEvent, len(from.ClickEvents))
	for i, click := range from.ClickEvents {
		data.ClickEvents[i] = ClickEvent{
			Timestamp: click.Timestamp,
			X:         click.X,
			Y:         click.Y,
			Pressure:  click.Pressure,
			Duration:  click.Duration,
		}
	}

	data.HesitationPoints = make([]HesitationPoint, len(from.HesitationPoints))
	for i, hp := range from.HesitationPoints {
		data.HesitationPoints[i] = HesitationPoint{
			X:         hp.X,
			Y:         hp.Y,
			Timestamp: hp.Timestamp,
			Duration:  hp.Duration,
		}
	}

	data.SlidePath = make([]Point, len(from.SlidePath))
	for i, p := range from.SlidePath {
		data.SlidePath[i] = Point{X: p.X, Y: p.Y}
	}

	data.DeviceOrientation = make([]Point, len(from.DeviceOrientation))
	for i, p := range from.DeviceOrientation {
		data.DeviceOrientation[i] = Point{X: p.X, Y: p.Y}
	}

	return data
}
