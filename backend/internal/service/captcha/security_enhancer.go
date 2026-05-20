package captcha

import (
	"fmt"
	"math"
	"math/rand"
	"strings"
	"time"
)

type SliderSecurityEnhancer struct {
	minTrajectoryPoints int
	maxSpeed            float64
	maxAcceleration     float64
	enableHightSampling bool
}

type EnhancedTrajectoryAnalysis struct {
	TrajectoryPoints     []TrajectoryPoint
	SpeedProfile         SpeedProfile
	AccelerationProfile  AccelerationProfile
	DirectionChanges     int
	TotalDistance        float64
	AverageSpeed         float64
	MaxSpeed             float64
	MinSpeed             float64
	SpeedVariance        float64
	IsHumanLike          bool
	Confidence           float64
	RiskLevel            string
	AnomalyIndicators    []string
}

type TrajectoryPoint struct {
	X            float64
	Y            float64
	Timestamp    int64
	Speed        float64
	Acceleration float64
}

type SliderPoint struct {
	X         int
	Y         int
	Timestamp int64
}

type SpeedProfile struct {
	InitialSpeed   float64
	FinalSpeed     float64
	AverageSpeed   float64
	SpeedFluctuation float64
	SpeedConsistency float64
}

type AccelerationProfile struct {
	AverageAcceleration float64
	MaxAcceleration     float64
	MinAcceleration     float64
	JerkCount           int
	Smoothness          float64
}

func NewSliderSecurityEnhancer() *SliderSecurityEnhancer {
	return &SliderSecurityEnhancer{
		minTrajectoryPoints: 10,
		maxSpeed:            2000,
		maxAcceleration:     500,
		enableHightSampling: true,
	}
}

func (e *SliderSecurityEnhancer) EnhancedTrajectoryAnalysis(points []SliderPoint, targetPosition int) *EnhancedTrajectoryAnalysis {
	if len(points) < e.minTrajectoryPoints {
		return &EnhancedTrajectoryAnalysis{
			IsHumanLike:       false,
			Confidence:        0,
			RiskLevel:         "high",
			AnomalyIndicators: []string{"insufficient_trajectory_points"},
		}
	}

	trajectoryPoints := e.convertToTrajectoryPoints(points)
	
	e.calculateSpeedAndAcceleration(trajectoryPoints)
	
	speedProfile := e.analyzeSpeedProfile(trajectoryPoints)
	
	accelerationProfile := e.analyzeAccelerationProfile(trajectoryPoints)
	
	directionChanges := e.countDirectionChanges(trajectoryPoints)
	
	totalDistance := e.calculateTotalDistance(trajectoryPoints)
	
	speedVariance := e.calculateSpeedVariance(trajectoryPoints)
	
	humanLikeScore := e.calculateHumanLikeScore(trajectoryPoints, speedProfile, accelerationProfile, directionChanges)
	
	riskLevel := e.determineRiskLevel(humanLikeScore, speedProfile, accelerationProfile)
	
	anomalyIndicators := e.detectAnomalyIndicators(trajectoryPoints, speedProfile, accelerationProfile, directionChanges)

	return &EnhancedTrajectoryAnalysis{
		TrajectoryPoints:     trajectoryPoints,
		SpeedProfile:        speedProfile,
		AccelerationProfile: accelerationProfile,
		DirectionChanges:    directionChanges,
		TotalDistance:       totalDistance,
		AverageSpeed:        speedProfile.AverageSpeed,
		MaxSpeed:            speedProfile.SpeedFluctuation,
		MinSpeed:            speedProfile.SpeedConsistency,
		SpeedVariance:       speedVariance,
		IsHumanLike:         humanLikeScore > 0.5,
		Confidence:          humanLikeScore * 100,
		RiskLevel:           riskLevel,
		AnomalyIndicators:   anomalyIndicators,
	}
}

func (e *SliderSecurityEnhancer) convertToTrajectoryPoints(points []SliderPoint) []TrajectoryPoint {
	trajectory := make([]TrajectoryPoint, len(points))
	
	for i, p := range points {
		trajectory[i] = TrajectoryPoint{
			X:         float64(p.X),
			Y:         float64(p.Y),
			Timestamp: p.Timestamp,
		}
	}
	
	return trajectory
}

func (e *SliderSecurityEnhancer) calculateSpeedAndAcceleration(points []TrajectoryPoint) {
	for i := 1; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-1].Timestamp)
		if dt <= 0 {
			dt = 1
		}
		
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		
		speed := distance / dt * 1000
		points[i].Speed = speed
		
		if i > 1 {
			prevSpeed := points[i-1].Speed
			acceleration := (speed - prevSpeed) / dt * 1000
			points[i].Acceleration = acceleration
		}
	}
	
	if len(points) > 0 {
		points[0].Speed = 0
	}
}

func (e *SliderSecurityEnhancer) analyzeSpeedProfile(points []TrajectoryPoint) SpeedProfile {
	if len(points) < 2 {
		return SpeedProfile{}
	}
	
	initialSpeed := points[0].Speed
	finalSpeed := points[len(points)-1].Speed
	
	var totalSpeed float64
	var maxSpeed float64
	var minSpeed float64 = 10000
	
	for _, p := range points {
		totalSpeed += p.Speed
		if p.Speed > maxSpeed {
			maxSpeed = p.Speed
		}
		if p.Speed < minSpeed {
			minSpeed = p.Speed
		}
	}
	
	averageSpeed := totalSpeed / float64(len(points))
	
	speedFluctuation := maxSpeed - averageSpeed
	speedConsistency := averageSpeed - minSpeed
	
	return SpeedProfile{
		InitialSpeed:     initialSpeed,
		FinalSpeed:       finalSpeed,
		AverageSpeed:     averageSpeed,
		SpeedFluctuation: speedFluctuation,
		SpeedConsistency: speedConsistency,
	}
}

func (e *SliderSecurityEnhancer) analyzeAccelerationProfile(points []TrajectoryPoint) AccelerationProfile {
	if len(points) < 3 {
		return AccelerationProfile{}
	}
	
	accelerations := make([]float64, 0, len(points)-2)
	
	for i := 2; i < len(points); i++ {
		dt := float64(points[i].Timestamp - points[i-2].Timestamp)
		if dt <= 0 {
			dt = 1
		}
		
		speedDiff := points[i].Speed - points[i-2].Speed
		acceleration := math.Abs(speedDiff / dt * 1000)
		accelerations = append(accelerations, acceleration)
	}
	
	if len(accelerations) == 0 {
		return AccelerationProfile{}
	}
	
	var totalAccel float64
	var maxAccel float64
	var minAccel float64 = 10000
	jerkCount := 0
	
	for i, accel := range accelerations {
		totalAccel += accel
		
		if accel > maxAccel {
			maxAccel = accel
		}
		if accel < minAccel {
			minAccel = accel
		}
		
		if i > 0 {
			jerk := math.Abs(accel - accelerations[i-1])
			if jerk > 100 {
				jerkCount++
			}
		}
	}
	
	averageAccel := totalAccel / float64(len(accelerations))
	
	smoothness := 1.0 - math.Min(float64(jerkCount)/float64(len(accelerations)), 1.0)
	
	return AccelerationProfile{
		AverageAcceleration: averageAccel,
		MaxAcceleration:     maxAccel,
		MinAcceleration:     minAccel,
		JerkCount:           jerkCount,
		Smoothness:          smoothness,
	}
}

func (e *SliderSecurityEnhancer) countDirectionChanges(points []TrajectoryPoint) int {
	if len(points) < 3 {
		return 0
	}
	
	changes := 0
	var prevAngle float64
	
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		
		if math.Abs(dx) < 0.1 && math.Abs(dy) < 0.1 {
			continue
		}
		
		angle := math.Atan2(dy, dx)
		
		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > math.Pi/4 {
				changes++
			}
		}
		
		prevAngle = angle
	}
	
	return changes
}

func (e *SliderSecurityEnhancer) calculateTotalDistance(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	
	var totalDistance float64
	
	for i := 1; i < len(points); i++ {
		dx := points[i].X - points[i-1].X
		dy := points[i].Y - points[i-1].Y
		distance := math.Sqrt(dx*dx + dy*dy)
		totalDistance += distance
	}
	
	return totalDistance
}

func (e *SliderSecurityEnhancer) calculateSpeedVariance(points []TrajectoryPoint) float64 {
	if len(points) < 2 {
		return 0
	}
	
	var totalSpeed float64
	for _, p := range points {
		totalSpeed += p.Speed
	}
	mean := totalSpeed / float64(len(points))
	
	var varianceSum float64
	for _, p := range points {
		diff := p.Speed - mean
		varianceSum += diff * diff
	}
	
	return varianceSum / float64(len(points))
}

func (e *SliderSecurityEnhancer) calculateHumanLikeScore(points []TrajectoryPoint, speedProfile SpeedProfile, accelProfile AccelerationProfile, directionChanges int) float64 {
	score := 1.0
	
	if speedProfile.AverageSpeed > e.maxSpeed {
		score -= 0.3
	}
	
	if accelProfile.MaxAcceleration > e.maxAcceleration {
		score -= 0.2
	}
	
	smoothness := accelProfile.Smoothness
	score *= (0.5 + 0.5*smoothness)
	
	normalChanges := float64(directionChanges) / float64(len(points))
	if normalChanges > 0.5 {
		score -= 0.2
	}
	
	if len(points) < 20 {
		score -= 0.1
	}
	
	timeSpan := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if timeSpan < 500 {
		score -= 0.3
	} else if timeSpan > 30000 {
		score -= 0.1
	}
	
	speedVariance := e.calculateSpeedVariance(points)
	if speedVariance < 10 {
		score -= 0.3
	}
	
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *SliderSecurityEnhancer) determineRiskLevel(humanLikeScore float64, speedProfile SpeedProfile, accelProfile AccelerationProfile) string {
	if humanLikeScore > 0.8 && accelProfile.Smoothness > 0.7 {
		return "low"
	} else if humanLikeScore > 0.5 {
		return "medium"
	}
	return "high"
}

func (e *SliderSecurityEnhancer) detectAnomalyIndicators(points []TrajectoryPoint, speedProfile SpeedProfile, accelProfile AccelerationProfile, directionChanges int) []string {
	indicators := []string{}
	
	if len(points) < e.minTrajectoryPoints {
		indicators = append(indicators, "trajectory_too_short")
	}
	
	if speedProfile.AverageSpeed > e.maxSpeed {
		indicators = append(indicators, "abnormally_fast")
	}
	
	if accelProfile.MaxAcceleration > e.maxAcceleration {
		indicators = append(indicators, "sudden_acceleration")
	}
	
	if accelProfile.JerkCount > len(points)/5 {
		indicators = append(indicators, "jerky_movement")
	}
	
	if directionChanges > len(points)/3 {
		indicators = append(indicators, "too_many_direction_changes")
	}
	
	timeSpan := float64(points[len(points)-1].Timestamp - points[0].Timestamp)
	if timeSpan < 500 {
		indicators = append(indicators, "too_fast_completion")
	}
	
	speedVariance := e.calculateSpeedVariance(points)
	if speedVariance < 10 {
		indicators = append(indicators, "suspiciously_consistent_speed")
	}
	
	if len(indicators) == 0 {
		indicators = append(indicators, "normal_trajectory")
	}
	
	return indicators
}

type ClickCaptchaSecurityEnhancer struct {
	minClickPoints int
	maxClickSpeed int64
	enableZoneAnalysis bool
}

type EnhancedClickAnalysis struct {
	ClickPoints       []ClickPoint
	TotalClicks       int
	ClickTimeSpan     int64
	AverageInterval   int64
	IntervalVariance  float64
	ZoneDistribution  map[string]int
	ClickPattern      string
	IsHumanLike       bool
	Confidence        float64
	RiskLevel         string
	AnomalyIndicators []string
}

type ClickPoint struct {
	X         int
	Y         int
	Timestamp int64
	Zone      string
}

func NewClickCaptchaSecurityEnhancer() *ClickCaptchaSecurityEnhancer {
	return &ClickCaptchaSecurityEnhancer{
		minClickPoints:     3,
		maxClickSpeed:      100,
		enableZoneAnalysis: true,
	}
}

func (e *ClickCaptchaSecurityEnhancer) AnalyzeClickPattern(clicks []ClickPoint) *EnhancedClickAnalysis {
	if len(clicks) < e.minClickPoints {
		return &EnhancedClickAnalysis{
			IsHumanLike:       false,
			Confidence:        0,
			RiskLevel:         "high",
			AnomalyIndicators: []string{"insufficient_clicks"},
		}
	}
	
	timeSpan := clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp
	
	intervals := e.calculateIntervals(clicks)
	avgInterval := e.calculateAverage(intervals)
	intervalVariance := e.calculateVariance(intervals)
	
	zoneDistribution := e.analyzeZoneDistribution(clicks)
	
	pattern := e.identifyClickPattern(clicks, avgInterval)
	
	humanLikeScore := e.calculateHumanLikeScore(clicks, avgInterval, intervalVariance)
	
	riskLevel := e.determineRiskLevel(humanLikeScore)
	
	anomalyIndicators := e.detectAnomalyIndicators(clicks, avgInterval, intervalVariance)
	
	return &EnhancedClickAnalysis{
		ClickPoints:       clicks,
		TotalClicks:       len(clicks),
		ClickTimeSpan:     timeSpan,
		AverageInterval:   avgInterval,
		IntervalVariance:  intervalVariance,
		ZoneDistribution: zoneDistribution,
		ClickPattern:     pattern,
		IsHumanLike:      humanLikeScore > 0.5,
		Confidence:       humanLikeScore * 100,
		RiskLevel:        riskLevel,
		AnomalyIndicators: anomalyIndicators,
	}
}

func (e *ClickCaptchaSecurityEnhancer) calculateIntervals(clicks []ClickPoint) []int64 {
	intervals := make([]int64, 0, len(clicks)-1)
	
	for i := 1; i < len(clicks); i++ {
		interval := clicks[i].Timestamp - clicks[i-1].Timestamp
		intervals = append(intervals, interval)
	}
	
	return intervals
}

func (e *ClickCaptchaSecurityEnhancer) calculateAverage(values []int64) int64 {
	if len(values) == 0 {
		return 0
	}
	
	var total int64
	for _, v := range values {
		total += v
	}
	
	return total / int64(len(values))
}

func (e *ClickCaptchaSecurityEnhancer) calculateVariance(values []int64) float64 {
	if len(values) < 2 {
		return 0
	}
	
	mean := float64(e.calculateAverage(values))
	
	var varianceSum float64
	for _, v := range values {
		diff := float64(v) - mean
		varianceSum += diff * diff
	}
	
	return varianceSum / float64(len(values))
}

func (e *ClickCaptchaSecurityEnhancer) analyzeZoneDistribution(clicks []ClickPoint) map[string]int {
	zones := make(map[string]int)
	
	for _, click := range clicks {
		zone := e.getZone(click.X, click.Y)
		zones[zone]++
	}
	
	return zones
}

func (e *ClickCaptchaSecurityEnhancer) getZone(x, y int) string {
	zoneX := x / 100
	zoneY := y / 100
	return fmt.Sprintf("zone_%d_%d", zoneX, zoneY)
}

func (e *ClickCaptchaSecurityEnhancer) identifyClickPattern(clicks []ClickPoint, avgInterval int64) string {
	if len(clicks) < 2 {
		return "insufficient_data"
	}
	
	if avgInterval < 200 {
		return "rapid"
	} else if avgInterval < 1000 {
		return "normal"
	}
	return "slow"
}

func (e *ClickCaptchaSecurityEnhancer) calculateHumanLikeScore(clicks []ClickPoint, avgInterval int64, variance float64) float64 {
	score := 1.0
	
	if avgInterval < e.maxClickSpeed {
		score -= 0.4
	}
	
	if variance < 100 {
		score -= 0.3
	}
	
	if len(clicks) < 3 {
		score -= 0.2
	}
	
	timeSpan := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if timeSpan < 300 {
		score -= 0.3
	}
	
	if score < 0 {
		score = 0
	}
	if score > 1 {
		score = 1
	}
	
	return score
}

func (e *ClickCaptchaSecurityEnhancer) determineRiskLevel(humanLikeScore float64) string {
	if humanLikeScore > 0.7 {
		return "low"
	} else if humanLikeScore > 0.4 {
		return "medium"
	}
	return "high"
}

func (e *ClickCaptchaSecurityEnhancer) detectAnomalyIndicators(clicks []ClickPoint, avgInterval int64, variance float64) []string {
	indicators := []string{}
	
	if len(clicks) < e.minClickPoints {
		indicators = append(indicators, "too_few_clicks")
	}
	
	if avgInterval < e.maxClickSpeed {
		indicators = append(indicators, "abnormally_fast_clicks")
	}
	
	if variance < 100 {
		indicators = append(indicators, "suspiciously_regular_intervals")
	}
	
	timeSpan := float64(clicks[len(clicks)-1].Timestamp - clicks[0].Timestamp)
	if timeSpan < 300 {
		indicators = append(indicators, "completed_too_quickly")
	}
	
	if len(indicators) == 0 {
		indicators = append(indicators, "normal_pattern")
	}
	
	return indicators
}

func init() {
	rand.Seed(time.Now().UnixNano())
}

type AntiScrapingProtector struct {
	enableRateLimiting     bool
	enableIPValidation     bool
	enableFingerprintCheck bool
	enableRequestValidation bool
	maxRequestsPerMinute   int
	blockDuration         time.Duration
}

type ScrapingProtectionResult struct {
	IsBlocked          bool
	BlockReason       string
	RemainingRequests int
	ResetTime         time.Time
	RiskLevel         string
	Recommendations   []string
}

func NewAntiScrapingProtector() *AntiScrapingProtector {
	return &AntiScrapingProtector{
		enableRateLimiting:     true,
		enableIPValidation:     true,
		enableFingerprintCheck: true,
		enableRequestValidation: true,
		maxRequestsPerMinute:   30,
		blockDuration:         5 * time.Minute,
	}
}

func (protector *AntiScrapingProtector) CheckRequest(ip string, fingerprint string, userAgent string) *ScrapingProtectionResult {
	result := &ScrapingProtectionResult{
		Recommendations: make([]string, 0),
		RiskLevel:       "low",
	}

	if protector.enableIPValidation {
		if protector.isSuspiciousIP(ip) {
			result.IsBlocked = true
			result.BlockReason = "suspicious_ip"
			result.RiskLevel = "high"
			result.Recommendations = append(result.Recommendations, "IP地址被标记为可疑")
			return result
		}
	}

	if protector.enableFingerprintCheck {
		fingerprintRisk := protector.checkFingerprint(fingerprint)
		if fingerprintRisk > 0.7 {
			result.IsBlocked = true
			result.BlockReason = "suspicious_fingerprint"
			result.RiskLevel = "high"
			result.Recommendations = append(result.Recommendations, "设备指纹异常")
			return result
		} else if fingerprintRisk > 0.4 {
			result.RiskLevel = "medium"
			result.Recommendations = append(result.Recommendations, "设备指纹存在轻微异常")
		}
	}

	if protector.enableRequestValidation {
		validationResult := protector.validateRequestPattern(ip, userAgent)
		if !validationResult.IsValid {
			result.IsBlocked = true
			result.BlockReason = validationResult.Reason
			result.RiskLevel = "high"
			result.Recommendations = append(result.Recommendations, validationResult.Details)
			return result
		}

		result.RemainingRequests = validationResult.RemainingRequests
		result.ResetTime = validationResult.ResetTime
	}

	if result.RiskLevel == "low" {
		result.Recommendations = append(result.Recommendations, "请求验证通过")
	}

	return result
}

func (protector *AntiScrapingProtector) isSuspiciousIP(ip string) bool {
	suspiciousPatterns := []string{
		"10.0.0.",
		"192.168.",
		"172.16.",
		"127.0.0.",
	}

	for _, pattern := range suspiciousPatterns {
		if len(ip) >= len(pattern) && ip[:len(pattern)] == pattern {
			return true
		}
	}

	return false
}

func (protector *AntiScrapingProtector) checkFingerprint(fingerprint string) float64 {
	if len(fingerprint) == 0 {
		return 0.8
	}

	suspiciousIndicators := 0

	if len(fingerprint) < 10 {
		suspiciousIndicators++
	}

	hasDuplicateChars := false
	charMap := make(map[rune]int)
	for _, c := range fingerprint {
		if charMap[c] > 0 {
			hasDuplicateChars = true
		}
		charMap[c]++
	}
	if hasDuplicateChars && len(fingerprint) < 20 {
		suspiciousIndicators++
	}

	knownPatterns := []string{
		"undefined",
		"null",
		"unknown",
		"fake",
	}
	for _, pattern := range knownPatterns {
		if len(fingerprint) >= len(pattern) && fingerprint[:len(pattern)] == pattern {
			suspiciousIndicators++
			break
		}
	}

	baseScore := float64(suspiciousIndicators) / 3.0

	return math.Min(baseScore, 1.0)
}

type ValidationResult struct {
	IsValid         bool
	Reason         string
	Details        string
	RemainingRequests int
	ResetTime      time.Time
}

func (protector *AntiScrapingProtector) validateRequestPattern(ip string, userAgent string) *ValidationResult {
	result := &ValidationResult{
		IsValid: true,
	}

	if len(userAgent) == 0 {
		result.IsValid = false
		result.Reason = "missing_user_agent"
		result.Details = "缺少User-Agent头"
		return result
	}

	knownBotPatterns := []string{
		"curl",
		"wget",
		"python-requests",
		"scrapy",
		"bot",
		"spider",
		"crawler",
	}

	userAgentLower := strings.ToLower(userAgent)
	for _, pattern := range knownBotPatterns {
		if strings.Contains(userAgentLower, pattern) {
			result.IsValid = false
			result.Reason = "bot_user_agent"
			result.Details = fmt.Sprintf("User-Agent包含已知的爬虫标识: %s", pattern)
			return result
		}
	}

	result.RemainingRequests = protector.maxRequestsPerMinute
	result.ResetTime = time.Now().Add(time.Minute)

	return result
}

type ImageWatermarkGenerator struct {
	opacityRange       []float64
	fontSizeRange     []int
	positionStrategies []string
}

func NewImageWatermarkGenerator() *ImageWatermarkGenerator {
	return &ImageWatermarkGenerator{
		opacityRange:       []float64{0.05, 0.15},
		fontSizeRange:     []int{10, 16},
		positionStrategies: []string{"corners", "diagonal", "random"},
	}
}

func (gen *ImageWatermarkGenerator) ApplyWatermark(img *image.RGBA, watermarkText string, strategy string) *image.RGBA {
	result := image.NewRGBA(img.Bounds())
	draw.Draw(result, result.Bounds(), img, img.Bounds().Min, draw.Src)

	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	positions := gen.generateWatermarkPositions(width, height, strategy, watermarkText)

	opacity := gen.opacityRange[0] + rand.Float64()*(gen.opacityRange[1]-gen.opacityRange[0])
	fontSize := gen.fontSizeRange[0] + rand.Intn(gen.fontSizeRange[1]-gen.fontSizeRange[0])

	for _, pos := range positions {
		gen.drawWatermarkText(result, watermarkText, pos.x, pos.y, fontSize, opacity)
	}

	return result
}

type watermarkPosition struct {
	x, y int
}

func (gen *ImageWatermarkGenerator) generateWatermarkPositions(width, height int, strategy string, text string) []watermarkPosition {
	positions := make([]watermarkPosition, 0)

	charWidth := 8
	textWidth := len(text) * charWidth

	switch strategy {
	case "corners":
		positions = append(positions, watermarkPosition{10, height - 25})
		positions = append(positions, watermarkPosition{width - textWidth - 10, 10})
	case "diagonal":
		positions = append(positions, watermarkPosition{10, 10})
		positions = append(positions, watermarkPosition{width - textWidth - 10, height - 25})
	case "random":
		positions = append(positions, watermarkPosition{
			rand.Intn(width - textWidth - 20),
			rand.Intn(height - 30),
		})
	default:
		positions = append(positions, watermarkPosition{10, height - 25})
	}

	return positions
}

func (gen *ImageWatermarkGenerator) drawWatermarkText(img *image.RGBA, text string, x, y, size int, opacity float64) {
	for i, char := range text {
		charX := x + i*(size/2)
		charY := y

		if charX+size >= img.Bounds().Dx() || charY+size >= img.Bounds().Dy() {
			continue
		}

		brightness := uint8(180 + rand.Intn(40))
		watermarkColor := color.RGBA{
			R: brightness,
			G: brightness,
			B: brightness,
			A: uint8(opacity * 255),
		}

		gen.drawCharacter(img, char, charX, charY, size, watermarkColor)
	}
}

func (gen *ImageWatermarkGenerator) drawCharacter(img *image.RGBA, char rune, x, y, size int, c color.RGBA) {
	for dy := 0; dy < size; dy++ {
		for dx := 0; dx < size; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				if gen.isCharPixel(int(char), dx, dy, size) {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func (gen *ImageWatermarkGenerator) isCharPixel(charCode, px, py, size int) bool {
	normalizedX := float64(px) / float64(size)
	normalizedY := float64(py) / float64(size)

	switch charCode {
	case 'C':
		return normalizedX < 0.3 || normalizedY < 0.2 || normalizedY > 0.8
	case 'A':
		return normalizedX < 0.2 || normalizedX > 0.8 || (normalizedY > 0.4 && normalizedY < 0.6)
	case 'P':
		return normalizedX < 0.2 || (normalizedY < 0.5 && normalizedX < 0.7)
	case 'T':
		return normalizedY < 0.2 || (normalizedX > 0.3 && normalizedX < 0.7)
	case 'H':
		return normalizedX < 0.2 || normalizedX > 0.8 || (normalizedY > 0.4 && normalizedY < 0.6)
	default:
		return px+py < size
	}
}

type VerificationSecurityEnhancer struct {
	enableAdvancedChecks     bool
	enableEntropyAnalysis    bool
	enablePatternMatching    bool
	minEntropyThreshold      float64
	maxPatternSimilarity     float64
}

func NewVerificationSecurityEnhancer() *VerificationSecurityEnhancer {
	return &VerificationSecurityEnhancer{
		enableAdvancedChecks:     true,
		enableEntropyAnalysis:    true,
		enablePatternMatching:    true,
		minEntropyThreshold:      3.5,
		maxPatternSimilarity:     0.85,
	}
}

type SecurityEnhancementResult struct {
	OverallRiskScore    float64
	EntropyScore        float64
	PatternRiskScore   float64
	AnomalyIndicators  []string
	IsSecure           bool
	Recommendations     []string
}

func (enhancer *VerificationSecurityEnhancer) AnalyzeSecurity(verificationData map[string]interface{}) *SecurityEnhancementResult {
	result := &SecurityEnhancementResult{
		AnomalyIndicators: make([]string, 0),
		Recommendations:    make([]string, 0),
	}

	if enhancer.enableEntropyAnalysis {
		result.EntropyScore = enhancer.calculateEntropyScore(verificationData)
		if result.EntropyScore < enhancer.minEntropyThreshold {
			result.AnomalyIndicators = append(result.AnomalyIndicators, "熵值过低，可能存在模式重复")
			result.OverallRiskScore += 0.3
		}
	}

	if enhancer.enablePatternMatching {
		result.PatternRiskScore = enhancer.detectPatternSimilarity(verificationData)
		if result.PatternRiskScore > enhancer.maxPatternSimilarity {
			result.AnomalyIndicators = append(result.AnomalyIndicators, "检测到高度相似的模式")
			result.OverallRiskScore += 0.4
		}
	}

	result.OverallRiskScore = math.Min(result.OverallRiskScore, 1.0)
	result.IsSecure = result.OverallRiskScore < 0.5

	if !result.IsSecure {
		result.Recommendations = append(result.Recommendations, "验证存在安全风险，建议拒绝")
	} else {
		result.Recommendations = append(result.Recommendations, "验证通过安全检查")
	}

	return result
}

func (enhancer *VerificationSecurityEnhancer) calculateEntropyScore(data map[string]interface{}) float64 {
	if len(data) == 0 {
		return 0
	}

	entropy := 0.0
	totalSymbols := 0

	for key, value := range data {
		switch v := value.(type) {
		case string:
			totalSymbols += len(v)
		case []byte:
			totalSymbols += len(v)
		case []int:
			totalSymbols += len(v)
		case []float64:
			totalSymbols += len(v)
		}

		entropy += enhancer.calculateStringEntropy(key)
	}

	if totalSymbols > 0 {
		entropy /= float64(totalSymbols)
	}

	return entropy
}

func (enhancer *VerificationSecurityEnhancer) calculateStringEntropy(s string) float64 {
	if len(s) == 0 {
		return 0
	}

	frequency := make(map[rune]int)
	for _, c := range s {
		frequency[c]++
	}

	entropy := 0.0
	n := float64(len(s))

	for _, count := range frequency {
		p := float64(count) / n
		if p > 0 {
			entropy -= p * math.Log2(p)
		}
	}

	return entropy
}

func (enhancer *VerificationSecurityEnhancer) detectPatternSimilarity(data map[string]interface{}) float64 {
	patterns := make([]string, 0)

	for key, value := range data {
		pattern := fmt.Sprintf("%s:%v", key, enhancer.normalizeValue(value))
		patterns = append(patterns, pattern)
	}

	if len(patterns) < 2 {
		return 0
	}

	similarityCount := 0
	for i := 0; i < len(patterns); i++ {
		for j := i + 1; j < len(patterns); j++ {
			if enhancer.calculateStringSimilarity(patterns[i], patterns[j]) > 0.8 {
				similarityCount++
			}
		}
	}

	maxComparisons := (len(patterns) * (len(patterns) - 1)) / 2
	if maxComparisons == 0 {
		return 0
	}

	return float64(similarityCount) / float64(maxComparisons)
}

func (enhancer *VerificationSecurityEnhancer) calculateStringSimilarity(s1, s2 string) float64 {
	if len(s1) == 0 || len(s2) == 0 {
		return 0
	}

	longer := s1
	shorter := s2
	if len(s1) < len(s2) {
		longer = s2
		shorter = s1
	}

	longerLength := float64(len(longer))
	return (longerLength - float64(enhancer.levenshteinDistance(longer, shorter))) / longerLength
}

func (enhancer *VerificationSecurityEnhancer) levenshteinDistance(s1, s2 string) int {
	runes1 := []rune(s1)
	runes2 := []rune(s2)

	matrix := make([][]int, len(runes1)+1)
	for i := range matrix {
		matrix[i] = make([]int, len(runes2)+1)
	}

	for i := range runes1 {
		matrix[i][0] = i
	}
	for j := range runes2 {
		matrix[0][j] = j
	}

	for i := 1; i <= len(runes1); i++ {
		for j := 1; j <= len(runes2); j++ {
			cost := 0
			if runes1[i-1] != runes2[j-1] {
				cost = 1
			}
			matrix[i][j] = min(
				matrix[i-1][j]+1,
				min(matrix[i][j-1]+1, matrix[i-1][j-1]+cost),
			)
		}
	}

	return matrix[len(runes1)][len(runes2)]
}

func (enhancer *VerificationSecurityEnhancer) normalizeValue(value interface{}) string {
	switch v := value.(type) {
	case string:
		return v
	case int:
		return fmt.Sprintf("%d", v)
	case int64:
		return fmt.Sprintf("%d", v)
	case float64:
		return fmt.Sprintf("%.2f", v)
	case bool:
		return fmt.Sprintf("%t", v)
	default:
		return fmt.Sprintf("%v", v)
	}
}

func init() {
	rand.Seed(time.Now().UnixNano())
}
