package risk

import (
	"context"
	"math"
	"time"
)

type BezierCurve struct {
	ControlPoints []Point
}

type VelocityDistribution struct {
	MeanVelocity      float64
	StdDevVelocity    float64
	MaxVelocity       float64
	MinVelocity       float64
	AccelerationScore float64
	DecelerationScore float64
	TrendScore        float64
}

type ClickRhythmAnalysis struct {
	IntervalMean      float64
	IntervalStdDev    float64
	PositionVariance  float64
	IsMechanical      bool
	IsTooFast         bool
	IsTooRegular      bool
}

func (e *RiskEngine) FitBezierCurve(tracks []MouseTrack) (float64, *BezierCurve) {
	if len(tracks) < 4 {
		return 1.0, nil
	}

	controlPoints := make([]Point, 0)
	step := len(tracks) / 3
	if step < 1 {
		step = 1
	}

	for i := 0; i < len(tracks); i += step {
		controlPoints = append(controlPoints, Point{
			X: tracks[i].X,
			Y: tracks[i].Y,
		})
	}

	if len(controlPoints) < 4 && len(tracks) >= 4 {
		controlPoints = []Point{
			{tracks[0].X, tracks[0].Y},
			{tracks[len(tracks)/3].X, tracks[len(tracks)/3].Y},
			{tracks[2*len(tracks)/3].X, tracks[2*len(tracks)/3].Y},
			{tracks[len(tracks)-1].X, tracks[len(tracks)-1].Y},
		}
	}

	fitness := e.calculateBezierFitness(tracks, controlPoints)

	return fitness, &BezierCurve{ControlPoints: controlPoints}
}

func (e *RiskEngine) calculateBezierFitness(tracks []MouseTrack, controlPoints []Point) float64 {
	if len(controlPoints) < 4 {
		return 1.0
	}

	var totalDeviation float64
	numSamples := len(tracks)

	for _, track := range tracks {
		t := e.calculateNearestT(MouseTrack{X: track.X, Y: track.Y, Timestamp: track.Timestamp}, controlPoints)
		bezierPoint := e.evaluateBezier(controlPoints, t)

		dx := track.X - bezierPoint.X
		dy := track.Y - bezierPoint.Y
		deviation := math.Sqrt(dx*dx + dy*dy)
		totalDeviation += deviation
	}

	avgDeviation := totalDeviation / float64(numSamples)

	normalizedDeviation := avgDeviation / 100.0
	if normalizedDeviation > 1.0 {
		normalizedDeviation = 1.0
	}

	fitness := 1.0 - normalizedDeviation

	return fitness
}

func (e *RiskEngine) calculateNearestT(track MouseTrack, controlPoints []Point) float64 {
	trackPoint := Point{X: track.X, Y: track.Y}
	minDist := math.MaxFloat64
	nearestT := 0.5

	for i := 0; i <= 100; i++ {
		t := float64(i) / 100.0
		bezierPoint := e.evaluateBezier(controlPoints, t)

		dx := trackPoint.X - bezierPoint.X
		dy := trackPoint.Y - bezierPoint.Y
		dist := math.Sqrt(dx*dx + dy*dy)

		if dist < minDist {
			minDist = dist
			nearestT = t
		}
	}

	return nearestT
}

func (e *RiskEngine) evaluateBezier(controlPoints []Point, t float64) Point {
	if len(controlPoints) == 4 {
		t2 := t * t
		t3 := t2 * t
		mt := 1.0 - t
		mt2 := mt * mt
		mt3 := mt2 * mt

		return Point{
			X: mt3*controlPoints[0].X + 3*mt2*t*controlPoints[1].X + 3*mt*t2*controlPoints[2].X + t3*controlPoints[3].X,
			Y: mt3*controlPoints[0].Y + 3*mt2*t*controlPoints[1].Y + 3*mt*t2*controlPoints[2].Y + t3*controlPoints[3].Y,
		}
	}

	sumX := 0.0
	sumY := 0.0
	n := len(controlPoints) - 1

	for i, point := range controlPoints {
		coeff := e.binomialCoefficient(n, i) *
			math.Pow(t, float64(i)) *
			math.Pow(1.0-t, float64(n-i))
		sumX += coeff * point.X
		sumY += coeff * point.Y
	}

	return Point{X: sumX, Y: sumY}
}

func (e *RiskEngine) binomialCoefficient(n, k int) float64 {
	if k > n {
		return 0
	}
	if k == 0 || k == n {
		return 1
	}

	result := 1.0
	for i := 0; i < k; i++ {
		result *= float64(n-i) / float64(i+1)
	}

	return result
}

func (e *RiskEngine) AnalyzeBezierFitting(tracks []MouseTrack) (int, []RiskFactor) {
	var factors []RiskFactor
	score := 0

	if len(tracks) < 4 {
		return score, factors
	}

	fitness, curve := e.FitBezierCurve(tracks)

	if curve == nil {
		return score, factors
	}

	if fitness > 0.98 {
		score += 25
		factors = append(factors, RiskFactor{
			Name:   "perfect_bezier_fit",
			Weight: 25,
			Reason: "轨迹完美拟合贝塞尔曲线，疑似机器生成",
		})
	} else if fitness > 0.95 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "high_bezier_fit",
			Weight: 15,
			Reason: "轨迹高度符合贝塞尔曲线特征",
		})
	}

	curvatureVariation := e.calculateCurvatureVariation(curve)
	if curvatureVariation < 0.05 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "abnormal_curvature",
			Weight: 15,
			Reason: "曲率变化异常，机械行为特征明显",
		})
	}

	return score, factors
}

func (e *RiskEngine) calculateCurvatureVariation(curve *BezierCurve) float64 {
	if len(curve.ControlPoints) < 3 {
		return 1.0
	}

	var curvatures []float64
	for i := 1; i < len(curve.ControlPoints)-1; i++ {
		v1x := curve.ControlPoints[i].X - curve.ControlPoints[i-1].X
		v1y := curve.ControlPoints[i].Y - curve.ControlPoints[i-1].Y
		v2x := curve.ControlPoints[i+1].X - curve.ControlPoints[i].X
		v2y := curve.ControlPoints[i+1].Y - curve.ControlPoints[i].Y

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
			curvatures = append(curvatures, math.Acos(cosAngle))
		}
	}

	if len(curvatures) < 2 {
		return 1.0
	}

	mean := 0.0
	for _, c := range curvatures {
		mean += c
	}
	mean /= float64(len(curvatures))

	if mean == 0 {
		return 0.0
	}

	var variance float64
	for _, c := range curvatures {
		diff := c - mean
		variance += diff * diff
	}
	variance /= float64(len(curvatures))

	return math.Sqrt(variance) / mean
}

func (e *RiskEngine) AnalyzeSlideVelocity(tracks []MouseTrack) (int, []RiskFactor, *VelocityDistribution) {
	var factors []RiskFactor
	score := 0

	if len(tracks) < 2 {
		return score, factors, nil
	}

	dist := e.calculateVelocityDistribution(tracks)

	accelerationRatio := dist.AccelerationScore / (dist.AccelerationScore + dist.DecelerationScore + 0.001)
	if accelerationRatio > 0.8 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "excessive_acceleration",
			Weight: 15,
			Reason: "滑动加速度异常，可能为机器行为",
		})
	}

	if dist.AccelerationScore < 0.1 && dist.DecelerationScore < 0.1 {
		score += 20
		factors = append(factors, RiskFactor{
			Name:   "constant_velocity",
			Weight: 20,
			Reason: "速度恒定，缺乏人类自然加减速特征",
		})
	}

	velocityCV := dist.StdDevVelocity / (dist.MeanVelocity + 0.001)
	if velocityCV < 0.1 {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "uniform_velocity_distribution",
			Weight: 15,
			Reason: "速度分布过于均匀",
		})
	}

	if dist.MaxVelocity/dist.MeanVelocity > 5.0 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "extreme_velocity_peaks",
			Weight: 10,
			Reason: "存在极端速度峰值",
		})
	}

	return score, factors, dist
}

func (e *RiskEngine) calculateVelocityDistribution(tracks []MouseTrack) *VelocityDistribution {
	if len(tracks) < 2 {
		return &VelocityDistribution{}
	}

	var velocities []float64
	for i := 1; i < len(tracks); i++ {
		dx := tracks[i].X - tracks[i-1].X
		dy := tracks[i].Y - tracks[i-1].Y
		dt := float64(tracks[i].Timestamp - tracks[i-1].Timestamp)

		if dt == 0 {
			continue
		}

		distance := math.Sqrt(dx*dx + dy*dy)
		velocity := distance / dt * 1000
		velocities = append(velocities, velocity)
	}

	if len(velocities) == 0 {
		return &VelocityDistribution{}
	}

	var sum, maxV, minV float64
	maxV = velocities[0]
	minV = velocities[0]

	for _, v := range velocities {
		sum += v
		if v > maxV {
			maxV = v
		}
		if v < minV {
			minV = v
		}
	}

	mean := sum / float64(len(velocities))

	var variance float64
	for _, v := range velocities {
		diff := v - mean
		variance += diff * diff
	}
	variance /= float64(len(velocities))
	stdDev := math.Sqrt(variance)

	var accelerationCount, decelerationCount int
	for i := 1; i < len(velocities); i++ {
		diff := velocities[i] - velocities[i-1]
		if diff > 0 {
			accelerationCount++
		} else if diff < 0 {
			decelerationCount++
		}
	}

	accelerationScore := float64(accelerationCount) / float64(len(velocities)-1)
	decelerationScore := float64(decelerationCount) / float64(len(velocities)-1)

	trendScore := 0.0
	if len(velocities) >= 3 {
		firstHalf := velocities[:len(velocities)/2]
		secondHalf := velocities[len(velocities)/2:]

		firstMean := sum / float64(len(firstHalf))
		secondSum := 0.0
		for _, v := range secondHalf {
			secondSum += v
		}
		secondMean := secondSum / float64(len(secondHalf))

		if firstMean > 0 {
			trendScore = (secondMean - firstMean) / firstMean
		}
	}

	return &VelocityDistribution{
		MeanVelocity:      mean,
		StdDevVelocity:    stdDev,
		MaxVelocity:       maxV,
		MinVelocity:       minV,
		AccelerationScore: accelerationScore,
		DecelerationScore: decelerationScore,
		TrendScore:        trendScore,
	}
}

func (e *RiskEngine) AnalyzeEnhancedClickRhythm(clicks []int64, positions []Point) (int, []RiskFactor, *ClickRhythmAnalysis) {
	var factors []RiskFactor
	score := 0

	if len(clicks) < 2 {
		return score, factors, nil
	}

	analysis := e.calculateClickRhythmAnalysis(clicks, positions)

	if analysis.IsMechanical {
		score += 20
		factors = append(factors, RiskFactor{
			Name:   "mechanical_click_pattern",
			Weight: 20,
			Reason: "点击节奏完全机械规律",
		})
	}

	if analysis.IsTooFast {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "unusually_fast_clicking",
			Weight: 15,
			Reason: "点击速度异常快，超出人类极限",
		})
	}

	if analysis.IsTooRegular {
		score += 15
		factors = append(factors, RiskFactor{
			Name:   "excessively_regular_clicks",
			Weight: 15,
			Reason: "点击间隔过于规律",
		})
	}

	if len(positions) >= 3 && analysis.PositionVariance < 0.01 {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "static_click_position",
			Weight: 10,
			Reason: "点击位置固定不变",
		})
	}

	if analysis.IntervalStdDev/analysis.IntervalMean < 0.05 && !analysis.IsMechanical {
		score += 10
		factors = append(factors, RiskFactor{
			Name:   "suspicious_regularity",
			Weight: 10,
			Reason: "点击间隔标准差极低",
		})
	}

	return score, factors, analysis
}

func (e *RiskEngine) calculateClickRhythmAnalysis(clicks []int64, positions []Point) *ClickRhythmAnalysis {
	analysis := &ClickRhythmAnalysis{}

	if len(clicks) < 2 {
		return analysis
	}

	var intervals []float64
	for i := 1; i < len(clicks); i++ {
		interval := float64(clicks[i] - clicks[i-1])
		intervals = append(intervals, interval)
	}

	sum := 0.0
	for _, interval := range intervals {
		sum += interval
	}
	analysis.IntervalMean = sum / float64(len(intervals))

	if analysis.IntervalMean > 0 {
		var variance float64
		for _, interval := range intervals {
			diff := interval - analysis.IntervalMean
			variance += diff * diff
		}
		variance /= float64(len(intervals))
		analysis.IntervalStdDev = math.Sqrt(variance)

		coefficientOfVariation := analysis.IntervalStdDev / analysis.IntervalMean
		analysis.IsTooRegular = coefficientOfVariation < 0.05

		intervalDiffs := math.Abs(intervals[1] - intervals[0])
		allSame := true
		for i := 2; i < len(intervals); i++ {
			if math.Abs(intervals[i]-intervals[i-1]) > intervalDiffs*0.01 {
				allSame = false
				break
			}
		}
		analysis.IsMechanical = allSame && len(intervals) >= 3
	}

	minInterval := float64(1000000)
	for _, interval := range intervals {
		if interval < minInterval {
			minInterval = interval
		}
	}
	analysis.IsTooFast = minInterval < 30

	if len(positions) >= 2 {
		var posVariance float64
		meanX, meanY := 0.0, 0.0
		for _, p := range positions {
			meanX += p.X
			meanY += p.Y
		}
		meanX /= float64(len(positions))
		meanY /= float64(len(positions))

		for _, p := range positions {
			dx := p.X - meanX
			dy := p.Y - meanY
			posVariance += dx*dx + dy*dy
		}
		posVariance /= float64(len(positions))

		maxRange := 10000.0
		analysis.PositionVariance = posVariance / (maxRange * maxRange)
		if analysis.PositionVariance > 1.0 {
			analysis.PositionVariance = 1.0
		}
	}

	return analysis
}

func (e *RiskEngine) EnhancedRiskAnalysis(ctx context.Context, behavior *BehaviorData, ip string, domain string, fingerprint *DeviceFingerprint) *RiskResult {
	result := &RiskResult{
		Factors:   make([]RiskFactor, 0),
		Timestamp: time.Now(),
	}

	totalScore := 0

	if e.whitelist != nil && e.whitelist.IsWhiteListed(ctx, ip, domain, behavior.UserID) {
		result.Score = 0
		result.Level = RiskLevelLow
		result.Recommended = ActionAllow
		return result
	}

	if e.ipLimit != nil {
		blacklisted, _ := e.ipLimit.IsBlacklisted(ctx, ip)
		if blacklisted {
			totalScore += 100
			result.Factors = append(result.Factors, RiskFactor{
				Name:   "ip_blacklisted",
				Weight: 100,
				Reason: "IP地址已在黑名单中",
			})
		}
	}

	slideDuration := behavior.SlideEnd - behavior.SlideStart
	if slideDuration > 0 {
		slideSeconds := float64(slideDuration) / 1000.0

		if slideSeconds < 1.0 {
			totalScore += 30
			result.Factors = append(result.Factors, RiskFactor{
				Name:   "slide_too_fast",
				Weight: 30,
				Reason: "滑动完成时间过短(<1秒)，疑似机器行为",
			})
		} else if slideSeconds > 30.0 {
			totalScore += 20
			result.Factors = append(result.Factors, RiskFactor{
				Name:   "slide_too_slow",
				Weight: 20,
				Reason: "滑动完成时间过长(>30秒)，异常行为",
			})
		}
	}

	bezierScore, bezierFactors := e.AnalyzeBezierFitting(behavior.MouseTracks)
	totalScore += bezierScore
	result.Factors = append(result.Factors, bezierFactors...)

	velocityScore, velocityFactors, _ := e.AnalyzeSlideVelocity(behavior.MouseTracks)
	totalScore += velocityScore
	result.Factors = append(result.Factors, velocityFactors...)

	var positions []Point
	if len(behavior.ClickTimes) > 0 && len(behavior.MouseTracks) >= len(behavior.ClickTimes) {
		for i := 0; i < len(behavior.ClickTimes) && i < len(behavior.MouseTracks); i++ {
			positions = append(positions, Point{
				X: behavior.MouseTracks[i].X,
				Y: behavior.MouseTracks[i].Y,
			})
		}
	}
	clickScore, clickFactors, _ := e.AnalyzeEnhancedClickRhythm(behavior.ClickTimes, positions)
	totalScore += clickScore
	result.Factors = append(result.Factors, clickFactors...)

	if fingerprint != nil {
		fpScore, fpFactors := e.AnalyzeDeviceFingerprint(fingerprint)
		totalScore += fpScore
		result.Factors = append(result.Factors, fpFactors...)
	}

	hmmScore, hmmFactors := e.AnalyzeBehaviorSequence(behavior)
	totalScore += hmmScore
	result.Factors = append(result.Factors, hmmFactors...)

	if e.ipLimit != nil {
		ipScore, ipFactors := e.ipLimit.CheckIPRisk(ctx, ip)
		totalScore += ipScore
		result.Factors = append(result.Factors, ipFactors...)
	}

	adjustedScore := e.AdjustScoreWithDynamicThreshold(totalScore, behavior)
	totalScore = adjustedScore

	if totalScore > 100 {
		totalScore = 100
	}

	result.Score = totalScore
	result.Level = e.GetRiskLevel(totalScore)
	result.Recommended = e.getRecommendedAction(result.Level)

	return result
}
