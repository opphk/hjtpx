package services

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"math"
	"regexp"
	"sync"
	"time"

	"hjtpx/internal/captcha"
)

type BehaviorService struct {
	config     *BehaviorConfig
	automation *AutomationDetector
	pool       *sync.Pool
}

type BehaviorConfig struct {
	MinTrackPoints      int
	MaxSpeedThreshold   float64
	MinHumanTime        time.Duration
	RiskScoreThreshold  float64
	EnableAdvancedDetect bool
}

var DefaultBehaviorConfig = &BehaviorConfig{
	MinTrackPoints:      10,
	MaxSpeedThreshold:   2000.0,
	MinHumanTime:        500 * time.Millisecond,
	RiskScoreThreshold:  0.7,
	EnableAdvancedDetect: true,
}

type BehaviorAnalysisResult struct {
	Valid             bool                     `json:"valid"`
	RiskLevel         RiskLevel                `json:"risk_level"`
	RiskScore         float64                  `json:"risk_score"`
	Confidence        float64                  `json:"confidence"`
	Features          *BehaviorFeatures        `json:"features"`
	AutomationMarkers *AutomationMarkers       `json:"automation_markers"`
	Recommendations   []string                `json:"recommendations"`
	AnalysisTime      time.Duration            `json:"analysis_time"`
}

type BehaviorFeatures struct {
	TotalPoints      int           `json:"total_points"`
	TotalDistance    float64       `json:"total_distance"`
	AvgSpeed         float64       `json:"avg_speed"`
	MaxSpeed         float64       `json:"max_speed"`
	MinSpeed         float64       `json:"min_speed"`
	SpeedVariance    float64       `json:"speed_variance"`
	AvgAcceleration  float64       `json:"avg_acceleration"`
	MaxAcceleration  float64       `json:"max_acceleration"`
	DirectionChanges int           `json:"direction_changes"`
	HasHumanPattern  bool          `json:"has_human_pattern"`
	StraightnessRatio float64      `json:"straightness_ratio"`
	JitterFactor     float64       `json:"jitter_factor"`
	CurveSmoothness  float64       `json:"curve_smoothness"`
	Duration         time.Duration `json:"duration"`
}

type AutomationMarkers struct {
	IsAutomated       bool            `json:"is_automated"`
	AutomationType    string          `json:"automation_type,omitempty"`
	Confidence        float64         `json:"confidence"`
	Markers           []Marker        `json:"markers"`
	CommonTools       []string        `json:"common_tools,omitempty"`
}

type Marker struct {
	Type        string  `json:"type"`
	Description string  `json:"description"`
	Weight      float64 `json:"weight"`
	Detected    bool    `json:"detected"`
}

type RiskLevel string

const (
	RiskLevelSafe     RiskLevel = "safe"
	RiskLevelLow      RiskLevel = "low"
	RiskLevelMedium   RiskLevel = "medium"
	RiskLevelHigh     RiskLevel = "high"
	RiskLevelCritical RiskLevel = "critical"
)

var knownAutomationPatterns = map[string]*regexp.Regexp{
	"selenium":        regexp.MustCompile(`(?i)(selenium|webdriver|phantomjs)`),
	"puppeteer":       regexp.MustCompile(`(?i)(puppeteer|headless|chrome-cdp)`),
	"playwright":      regexp.MustCompile(`(?i)(playwright|webkit)`),
	"mechanize":       regexp.MustCompile(`(?i)(mechanize|mechanize/[\d.]+)`),
	"curl":            regexp.MustCompile(`(?i)(curl/[\d.]+)`),
	"python-requests": regexp.MustCompile(`(?i)(python-requests|requests/[\d.]+)`),
	"java-http":       regexp.MustCompile(`(?i)(java/[\d.]+|apache-httpclient)`),
	"go-http":         regexp.MustCompile(`(?i)(go-http-client|fasthttp)`),
	"node-http":       regexp.MustCompile(`(?i)(node-fetch|axios|superagent)`),
}

func NewBehaviorService(config *BehaviorConfig) *BehaviorService {
	if config == nil {
		config = DefaultBehaviorConfig
	}

	return &BehaviorService{
		config:     config,
		automation: NewAutomationDetector(),
		pool: &sync.Pool{
			New: func() interface{} {
				return &BehaviorFeatures{}
			},
		},
	}
}

func (s *BehaviorService) AnalyzeMouseTrajectory(ctx context.Context, trackDataJSON string, metadata *BehaviorMetadata) (*BehaviorAnalysisResult, error) {
	startTime := time.Now()

	result := &BehaviorAnalysisResult{
		Recommendations: make([]string, 0),
	}

	trackData, err := s.parseTrackData(trackDataJSON)
	if err != nil {
		return nil, fmt.Errorf("failed to parse track data: %w", err)
	}

	if err := s.validateTrackData(trackData); err != nil {
		result.RiskLevel = RiskLevelHigh
		result.RiskScore = 1.0
		result.Recommendations = append(result.Recommendations, "Invalid track data format")
		return result, nil
	}

	features := s.extractFeatures(ctx, trackData)
	result.Features = features

	automationMarkers := s.detectAutomation(ctx, trackData, features, metadata)
	result.AutomationMarkers = automationMarkers

	result.RiskScore = s.calculateRiskScore(features, automationMarkers)
	result.Confidence = s.calculateConfidence(features, automationMarkers)

	result.RiskLevel = s.determineRiskLevel(result.RiskScore, automationMarkers)

	result.Valid = result.RiskLevel != RiskLevelHigh && result.RiskLevel != RiskLevelCritical

	result.Recommendations = s.generateRecommendations(result)

	result.AnalysisTime = time.Since(startTime)

	return result, nil
}

func (s *BehaviorService) AnalyzeBatch(ctx context.Context, requests []*BehaviorAnalysisRequest) ([]*BehaviorAnalysisResult, error) {
	results := make([]*BehaviorAnalysisResult, len(requests))
	var wg sync.WaitGroup
	var mu sync.Mutex
	errors := make([]error, 0)
	errMu := sync.Mutex{}

	for i, req := range requests {
		wg.Add(1)
		go func(idx int, request *BehaviorAnalysisRequest) {
			defer wg.Done()

			result, err := s.AnalyzeMouseTrajectory(ctx, request.TrackData, request.Metadata)
			mu.Lock()
			results[idx] = result
			if err != nil {
				errMu.Lock()
				errors = append(errors, fmt.Errorf("request %d: %w", idx, err))
				errMu.Unlock()
			}
			mu.Unlock()
		}(i, req)
	}

	wg.Wait()

	if len(errors) > 0 {
		return results, errors[0]
	}

	return results, nil
}

func (s *BehaviorService) parseTrackData(data string) (*captcha.TrackData, error) {
	var track captcha.TrackData
	if err := json.Unmarshal([]byte(data), &track); err != nil {
		return nil, fmt.Errorf("failed to parse track data: %w", err)
	}
	return &track, nil
}

func (s *BehaviorService) validateTrackData(track *captcha.TrackData) error {
	if track == nil {
		return errors.New("track data is nil")
	}

	if len(track.Points) < s.config.MinTrackPoints {
		return fmt.Errorf("insufficient track points: got %d, need %d", len(track.Points), s.config.MinTrackPoints)
	}

	for i, point := range track.Points {
		if point.X < 0 || point.Y < 0 {
			return fmt.Errorf("invalid coordinates at point %d", i)
		}
		if point.T < 0 {
			return fmt.Errorf("invalid timestamp at point %d", i)
		}
	}

	return nil
}

func (s *BehaviorService) extractFeatures(ctx context.Context, track *captcha.TrackData) *BehaviorFeatures {
	features := &BehaviorFeatures{
		TotalPoints: len(track.Points),
		Duration:    time.Duration(track.Duration) * time.Millisecond,
	}

	if len(track.Points) < 2 {
		return features
	}

	var totalDistance, directDistance float64
	var speeds []float64
	var accelerations []float64
	var prevSpeed float64 = 0
	directionChanges := 0
	var prevAngle float64 = 0
	var angles []float64

	startPoint := track.Points[0]
	endPoint := track.Points[len(track.Points)-1]
	directDistance = math.Sqrt(
		math.Pow(endPoint.X-startPoint.X, 2) +
			math.Pow(endPoint.Y-startPoint.Y, 2),
	)

	for i := 1; i < len(track.Points); i++ {
		prev := track.Points[i-1]
		curr := track.Points[i]

		dx := curr.X - prev.X
		dy := curr.Y - prev.Y
		dt := float64(curr.T-prev.T) / 1000.0

		if dt <= 0 {
			dt = 0.001
		}

		segmentDist := math.Sqrt(dx*dx + dy*dy)
		totalDistance += segmentDist

		speed := segmentDist / dt
		speeds = append(speeds, speed)

		if i > 1 && prevSpeed > 0 {
			accel := math.Abs((speed - prevSpeed) / dt)
			accelerations = append(accelerations, accel)
		}
		prevSpeed = speed

		angle := math.Atan2(dy, dx) * 180 / math.Pi
		angles = append(angles, angle)

		if i > 1 {
			angleDiff := math.Abs(angle - prevAngle)
			if angleDiff > 30 && angleDiff < 330 {
				directionChanges++
			}
		}
		prevAngle = angle
	}

	features.TotalDistance = totalDistance
	features.DirectionChanges = directionChanges

	if directDistance > 0 {
		features.StraightnessRatio = directDistance / totalDistance
	}

	if len(speeds) > 0 {
		var sumSpeed float64
		maxSpeed, minSpeed := speeds[0], speeds[0]
		for _, speed := range speeds {
			sumSpeed += speed
			if speed > maxSpeed {
				maxSpeed = speed
			}
			if speed < minSpeed {
				minSpeed = speed
			}
		}

		features.AvgSpeed = sumSpeed / float64(len(speeds))
		features.MaxSpeed = maxSpeed
		features.MinSpeed = minSpeed

		if len(speeds) > 1 {
			var sumSqDiff float64
			for _, speed := range speeds {
				diff := speed - features.AvgSpeed
				sumSqDiff += diff * diff
			}
			features.SpeedVariance = sumSqDiff / float64(len(speeds))
		}
	}

	if len(accelerations) > 0 {
		var sumAccel, maxAccel float64 = accelerations[0], accelerations[0]
		for _, accel := range accelerations {
			sumAccel += accel
			if accel > maxAccel {
				maxAccel = accel
			}
		}
		features.AvgAcceleration = sumAccel / float64(len(accelerations))
		features.MaxAcceleration = maxAccel
	}

	features.JitterFactor = s.calculateJitter(angles)
	features.CurveSmoothness = s.calculateSmoothness(angles)
	features.HasHumanPattern = s.detectHumanPattern(features)

	return features
}

func (s *BehaviorService) calculateJitter(angles []float64) float64 {
	if len(angles) < 3 {
		return 0
	}

	var jitterSum float64
	for i := 1; i < len(angles)-1; i++ {
		prevDiff := math.Abs(angles[i] - angles[i-1])
		nextDiff := math.Abs(angles[i+1] - angles[i])
		jitterSum += math.Min(prevDiff, nextDiff)
	}

	return jitterSum / float64(len(angles)-2)
}

func (s *BehaviorService) calculateSmoothness(angles []float64) float64 {
	if len(angles) < 2 {
		return 0
	}

	var totalChange float64
	for i := 1; i < len(angles); i++ {
		totalChange += math.Abs(angles[i] - angles[i-1])
	}

	maxPossibleChange := float64(len(angles)-1) * 180
	return 1.0 - (totalChange / maxPossibleChange)
}

func (s *BehaviorService) detectHumanPattern(features *BehaviorFeatures) bool {
	if features.DirectionChanges < 2 && features.TotalDistance > 100 {
		return false
	}

	if features.SpeedVariance < 10 && features.AvgSpeed > 100 {
		return false
	}

	if features.AvgAcceleration < 100 && features.TotalDistance > 150 {
		return false
	}

	if features.MaxSpeed > s.config.MaxSpeedThreshold {
		return false
	}

	humanScore := 0.0

	if features.DirectionChanges >= 2 {
		humanScore += 0.2
	}
	if features.DirectionChanges >= 5 {
		humanScore += 0.1
	}

	if features.SpeedVariance >= 30 {
		humanScore += 0.2
	} else if features.SpeedVariance >= 15 {
		humanScore += 0.1
	}

	if features.AvgAcceleration >= 150 {
		humanScore += 0.2
	} else if features.AvgAcceleration >= 80 {
		humanScore += 0.1
	}

	if features.Duration >= s.config.MinHumanTime {
		humanScore += 0.2
	}

	if features.TotalPoints >= 30 {
		humanScore += 0.1
	}

	if features.JitterFactor > 5 {
		humanScore += 0.1
	}

	return humanScore >= 0.6
}

func (s *BehaviorService) detectAutomation(ctx context.Context, track *captcha.TrackData, features *BehaviorFeatures, metadata *BehaviorMetadata) *AutomationMarkers {
	markers := &AutomationMarkers{
		Markers: make([]Marker, 0),
	}

	markers.Markers = append(markers.Markers, s.checkUniformSpeed(features)...)
	markers.Markers = append(markers.Markers, s.checkConstantAcceleration(features)...)
	markers.Markers = append(markers.Markers, s.checkPerfectLine(features)...)
	markers.Markers = append(markers.Markers, s.checkTimeAnomalies(track, features)...)
	markers.Markers = append(markers.Markers, s.checkPointDensity(track, features)...)

	if metadata != nil {
		markers.Markers = append(markers.Markers, s.checkUserAgent(metadata.UserAgent)...)
		markers.Markers = append(markers.Markers, s.checkRequestPatterns(metadata)...)
	}

	var totalWeight float64
	var detectedWeight float64

	for _, marker := range markers.Markers {
		totalWeight += marker.Weight
		if marker.Detected {
			detectedWeight += marker.Weight
			markers.Markers = append(markers.Markers, marker)
		}
	}

	if totalWeight > 0 {
		markers.Confidence = detectedWeight / totalWeight
	}

	markers.IsAutomated = markers.Confidence >= 0.6

	if markers.IsAutomated {
		markers.AutomationType = s.identifyAutomationTool(markers)
		markers.CommonTools = s.suggestCommonTools(markers)
	}

	return markers
}

func (s *BehaviorService) checkUniformSpeed(features *BehaviorFeatures) []Marker {
	markers := make([]Marker, 0)

	if features.SpeedVariance < 10 && features.AvgSpeed > 50 {
		markers = append(markers, Marker{
			Type:        "uniform_speed",
			Description: "Speed is suspiciously uniform, typical of automated tools",
			Weight:      0.7,
			Detected:    true,
		})
	}

	if features.MaxSpeed < features.AvgSpeed*1.5 {
		markers = append(markers, Marker{
			Type:        "low_speed_variance",
			Description: "Maximum speed is too close to average speed",
			Weight:      0.5,
			Detected:    true,
		})
	}

	return markers
}

func (s *BehaviorService) checkConstantAcceleration(features *BehaviorFeatures) []Marker {
	markers := make([]Marker, 0)

	if features.AvgAcceleration < 50 && features.TotalDistance > 100 {
		markers = append(markers, Marker{
			Type:        "constant_acceleration",
			Description: "Acceleration is too constant, not typical of human movement",
			Weight:      0.6,
			Detected:    true,
		})
	}

	if features.MaxAcceleration < 200 && features.TotalDistance > 150 {
		markers = append(markers, Marker{
			Type:        "low_acceleration",
			Description: "Maximum acceleration is suspiciously low",
			Weight:      0.5,
			Detected:    true,
		})
	}

	return markers
}

func (s *BehaviorService) checkPerfectLine(features *BehaviorFeatures) []Marker {
	markers := make([]Marker, 0)

	if features.StraightnessRatio > 0.98 && features.TotalDistance > 200 {
		markers = append(markers, Marker{
			Type:        "perfect_line",
			Description: "Movement is almost perfectly linear, likely automated",
			Weight:      0.8,
			Detected:    true,
		})
	}

	if features.DirectionChanges == 0 && features.TotalDistance > 50 {
		markers = append(markers, Marker{
			Type:        "no_direction_change",
			Description: "No direction changes detected, very suspicious",
			Weight:      0.9,
			Detected:    true,
		})
	}

	return markers
}

func (s *BehaviorService) checkTimeAnomalies(track *captcha.TrackData, features *BehaviorFeatures) []Marker {
	markers := make([]Marker, 0)

	if features.Duration < s.config.MinHumanTime/2 {
		markers = append(markers, Marker{
			Type:        "too_fast",
			Description: "Completion time is suspiciously fast",
			Weight:      0.8,
			Detected:    true,
		})
	}

	if len(track.Points) > 0 {
		var timeGaps []int64
		for i := 1; i < len(track.Points); i++ {
			gap := track.Points[i].T - track.Points[i-1].T
			if gap > 0 {
				timeGaps = append(timeGaps, gap)
			}
		}

		if len(timeGaps) > 0 {
			var uniformGaps int
			for _, gap := range timeGaps {
				if gap == timeGaps[0] {
					uniformGaps++
				}
			}

			if float64(uniformGaps)/float64(len(timeGaps)) > 0.9 {
				markers = append(markers, Marker{
					Type:        "uniform_time_gaps",
					Description: "Time intervals between points are suspiciously uniform",
					Weight:      0.7,
					Detected:    true,
				})
			}
		}
	}

	return markers
}

func (s *BehaviorService) checkPointDensity(track *captcha.TrackData, features *BehaviorFeatures) []Marker {
	markers := make([]Marker, 0)

	if features.Duration > 0 {
		pointsPerSecond := float64(len(track.Points)) / features.Duration.Seconds()

		if pointsPerSecond < 5 {
			markers = append(markers, Marker{
				Type:        "low_point_density",
				Description: "Too few track points for the duration",
				Weight:      0.4,
				Detected:    true,
			})
		}

		if pointsPerSecond > 200 {
			markers = append(markers, Marker{
				Type:        "high_point_density",
				Description: "Suspiciously high point density",
				Weight:      0.3,
				Detected:    true,
			})
		}
	}

	if len(track.Points) > 1 {
		var avgDist float64
		for i := 1; i < len(track.Points); i++ {
			dx := track.Points[i].X - track.Points[i-1].X
			dy := track.Points[i].Y - track.Points[i-1].Y
			avgDist += math.Sqrt(dx*dx + dy*dy)
		}
		avgDist /= float64(len(track.Points) - 1)

		if avgDist < 1.0 && features.TotalDistance > 50 {
			markers = append(markers, Marker{
				Type:        "micro_movements",
				Description: "Suspiciously small movements between points",
				Weight:      0.3,
				Detected:    true,
			})
		}

		if avgDist > 50 && len(track.Points) > 5 {
			markers = append(markers, Marker{
				Type:        "large_movements",
				Description: "Large jumps between points, may indicate interpolation",
				Weight:      0.4,
				Detected:    true,
			})
		}
	}

	return markers
}

func (s *BehaviorService) checkUserAgent(userAgent string) []Marker {
	markers := make([]Marker, 0)

	if userAgent == "" {
		return markers
	}

	for toolName, pattern := range knownAutomationPatterns {
		if pattern.MatchString(userAgent) {
			markers = append(markers, Marker{
				Type:        "user_agent",
				Description: fmt.Sprintf("User-Agent contains %s signature", toolName),
				Weight:      0.9,
				Detected:    true,
			})
		}
	}

	return markers
}

func (s *BehaviorService) checkRequestPatterns(metadata *BehaviorMetadata) []Marker {
	markers := make([]Marker, 0)

	if metadata != nil {
		if metadata.RequestInterval > 0 && metadata.RequestInterval < 100*time.Millisecond {
			markers = append(markers, Marker{
				Type:        "rapid_requests",
				Description: "Requests coming in too quickly",
				Weight:      0.7,
				Detected:    true,
			})
		}

		if metadata.SessionDuration > 0 && metadata.RequestCount > 0 {
			avgTimeBetweenRequests := metadata.SessionDuration / time.Duration(metadata.RequestCount)
			if avgTimeBetweenRequests < 500*time.Millisecond {
				markers = append(markers, Marker{
					Type:        "fast_session",
					Description: "Too fast between requests on average",
					Weight:      0.6,
					Detected:    true,
				})
			}
		}
	}

	return markers
}

func (s *BehaviorService) identifyAutomationTool(markers *AutomationMarkers) string {
	for _, marker := range markers.Markers {
		if marker.Type == "user_agent" {
			if matched, _ := regexp.MatchString("(?i)selenium|webdriver", marker.Description); matched {
				return "selenium"
			}
			if matched, _ := regexp.MatchString("(?i)puppeteer|headless", marker.Description); matched {
				return "puppeteer"
			}
			if matched, _ := regexp.MatchString("(?i)playwright", marker.Description); matched {
				return "playwright"
			}
		}
	}

	return "unknown"
}

func (s *BehaviorService) suggestCommonTools(markers *AutomationMarkers) []string {
	tools := make([]string, 0, 5)

	toolScores := map[string]float64{
		"Selenium":        0,
		"Puppeteer":       0,
		"Playwright":      0,
		"Cheerio":         0,
		"Requests":        0,
	}

	for _, marker := range markers.Markers {
		if marker.Detected {
			switch marker.Type {
			case "uniform_speed":
				toolScores["Selenium"] += 0.3
				toolScores["Puppeteer"] += 0.3
				toolScores["Playwright"] += 0.3
			case "user_agent":
				if regexp.MustCompile(`(?i)selenium`).MatchString(marker.Description) {
					toolScores["Selenium"] += 0.8
				}
				if regexp.MustCompile(`(?i)puppeteer`).MatchString(marker.Description) {
					toolScores["Puppeteer"] += 0.8
				}
				if regexp.MustCompile(`(?i)playwright`).MatchString(marker.Description) {
					toolScores["Playwright"] += 0.8
				}
			case "perfect_line":
				toolScores["Selenium"] += 0.4
				toolScores["Cheerio"] += 0.3
				toolScores["Requests"] += 0.3
			}
		}
	}

	for tool, score := range toolScores {
		if score > 0.5 {
			tools = append(tools, fmt.Sprintf("%s (%.2f)", tool, score))
		}
	}

	return tools
}

func (s *BehaviorService) calculateRiskScore(features *BehaviorFeatures, markers *AutomationMarkers) float64 {
	score := 0.0

	if !features.HasHumanPattern {
		score += 0.3
	}

	if features.MaxSpeed > s.config.MaxSpeedThreshold {
		score += 0.25
	}

	if features.StraightnessRatio > 0.98 {
		score += 0.2
	}

	if features.SpeedVariance < 10 {
		score += 0.15
	}

	if features.Duration < s.config.MinHumanTime {
		score += 0.2
	}

	if markers.Confidence > 0 {
		score += markers.Confidence * 0.4
	}

	if score > 1.0 {
		score = 1.0
	}

	return score
}

func (s *BehaviorService) calculateConfidence(features *BehaviorFeatures, markers *AutomationMarkers) float64 {
	confidence := 0.5

	if features.TotalPoints >= 30 {
		confidence += 0.1
	}

	if features.SpeedVariance > 30 {
		confidence += 0.1
	}

	if features.DirectionChanges >= 3 {
		confidence += 0.1
	}

	if features.HasHumanPattern {
		confidence += 0.2
	}

	confidence -= markers.Confidence * 0.5

	if confidence < 0 {
		confidence = 0
	}
	if confidence > 1 {
		confidence = 1
	}

	return confidence
}

func (s *BehaviorService) determineRiskLevel(score float64, markers *AutomationMarkers) RiskLevel {
	if score >= 0.9 || (markers.IsAutomated && markers.Confidence > 0.8) {
		return RiskLevelCritical
	}

	if score >= 0.7 || markers.IsAutomated {
		return RiskLevelHigh
	}

	if score >= 0.5 {
		return RiskLevelMedium
	}

	return RiskLevelLow
}

func (s *BehaviorService) generateRecommendations(result *BehaviorAnalysisResult) []string {
	recs := make([]string, 0)

	switch result.RiskLevel {
	case RiskLevelCritical:
		recs = append(recs, "CRITICAL: Strong evidence of automation detected")
		recs = append(recs, "Recommendation: Block or require additional verification")
	case RiskLevelHigh:
		recs = append(recs, "HIGH RISK: Multiple automation indicators found")
		recs = append(recs, "Recommendation: Additional verification steps recommended")
	case RiskLevelMedium:
		recs = append(recs, "MEDIUM RISK: Some suspicious patterns detected")
		recs = append(recs, "Recommendation: Monitor closely")
	case RiskLevelLow:
		recs = append(recs, "LOW RISK: Behavior appears human-like")
	}

	if result.AutomationMarkers != nil && len(result.AutomationMarkers.Markers) > 0 {
		recs = append(recs, fmt.Sprintf("Detected %d automation markers", len(result.AutomationMarkers.Markers)))
	}

	if result.Features != nil {
		if !result.Features.HasHumanPattern {
			recs = append(recs, "Warning: Movement patterns do not match human behavior")
		}

		if result.Features.MaxSpeed > 1500 {
			recs = append(recs, fmt.Sprintf("Speed exceeds typical human threshold: %.0f", result.Features.MaxSpeed))
		}
	}

	return recs
}

type AutomationDetector struct {
	mu sync.RWMutex
}

func NewAutomationDetector() *AutomationDetector {
	return &AutomationDetector{}
}

func (d *AutomationDetector) Detect(track *captcha.TrackData, metadata *BehaviorMetadata) (bool, string) {
	d.mu.Lock()
	defer d.mu.Unlock()

	return false, ""
}

type BehaviorAnalysisRequest struct {
	TrackData string
	Metadata  *BehaviorMetadata
}

type BehaviorMetadata struct {
	UserAgent         string
	IPAddress         string
	SessionID         string
	RequestInterval   time.Duration
	SessionDuration   time.Duration
	RequestCount      int
	Referer           string
	AcceptLanguage    string
}
