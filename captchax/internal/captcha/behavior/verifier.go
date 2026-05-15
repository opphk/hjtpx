package behavior

import (
	"context"
	"errors"
	"fmt"
	"math"

	"captchax/internal/risk"
)

var (
	ErrInvalidToken     = errors.New("invalid captcha token")
	ErrCaptchaNotFound  = errors.New("captcha not found or expired")
	ErrCaptchaVerified  = errors.New("captcha already verified")
	ErrInvalidChallenge = errors.New("invalid challenge response")
)

type VerifyRequest struct {
	Token        string          `json:"token"`
	ChallengeType string         `json:"challenge_type"`
	ClickSequence []ClickInput   `json:"click_sequence"`
	DragPath     []DragPoint     `json:"drag_path"`
	HoverSequence []HoverInput   `json:"hover_sequence"`
	BehaviorData *BehaviorInput  `json:"behavior_data"`
}

type ClickInput struct {
	X       int   `json:"x"`
	Y       int   `json:"y"`
	Index   int   `json:"index"`
	Time    int64 `json:"time"`
}

type DragPoint struct {
	X       int   `json:"x"`
	Y       int   `json:"y"`
	Time    int64 `json:"time"`
}

type HoverInput struct {
	X       int   `json:"x"`
	Y       int   `json:"y"`
	Time    int64 `json:"time"`
	Duration int64 `json:"duration"`
}

type BehaviorInput struct {
	MouseTracks    []MouseTrack   `json:"mouse_tracks"`
	ClickEvents    []ClickEvent   `json:"click_events"`
	KeyPressIntervals []int64     `json:"key_press_intervals"`
	ScrollPatterns []ScrollEvent  `json:"scroll_patterns"`
	Fingerprint    string         `json:"fingerprint"`
}

type MouseTrack struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Velocity  float64 `json:"velocity"`
}

type ClickEvent struct {
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Duration  int64   `json:"duration"`
	Pressure  float64 `json:"pressure"`
}

type ScrollEvent struct {
	X         int   `json:"x"`
	Y         int   `json:"y"`
	Timestamp int64 `json:"timestamp"`
	DeltaY    int   `json:"delta_y"`
}

type VerifyResult struct {
	Success      bool             `json:"success"`
	Score        float64          `json:"score"`
	RiskLevel    risk.RiskLevel   `json:"risk_level"`
	RiskScore    int              `json:"risk_score"`
	Message      string           `json:"message"`
	Factors      []risk.RiskFactor `json:"factors,omitempty"`
	PositionScore float64         `json:"position_score,omitempty"`
}

type VerifyService struct {
	captcha   *BehaviorCaptcha
	riskEngine *risk.RiskEngine
	tolerance int
}

func NewVerifyService(captcha *BehaviorCaptcha, riskEngine *risk.RiskEngine) *VerifyService {
	return &VerifyService{
		captcha:    captcha,
		riskEngine: riskEngine,
		tolerance:  15,
	}
}

func (vs *VerifyService) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResult, error) {
	if req.Token == "" {
		return &VerifyResult{
			Success: false,
			Message: "token is required",
		}, ErrInvalidToken
	}

	captchaData, err := vs.captcha.getCaptchaData(ctx, req.Token)
	if err != nil {
		return &VerifyResult{
			Success:   false,
			Message:   "captcha not found or expired",
			RiskLevel: risk.RiskLevelCritical,
			RiskScore: 100,
		}, ErrCaptchaNotFound
	}

	challengeType := req.ChallengeType
	if challengeType == "" {
		challengeType = captchaData.ChallengeType
	}

	var positionScore float64
	var success bool
	var message string

	switch challengeType {
	case "click_order":
		positionScore, success, message = vs.verifyClickOrder(captchaData, req.ClickSequence)
	case "drag_path":
		positionScore, success, message = vs.verifyDragPath(captchaData, req.DragPath)
	case "hover_sequence":
		positionScore, success, message = vs.verifyHoverSequence(captchaData, req.HoverSequence)
	default:
		return &VerifyResult{
			Success: false,
			Message: "unknown challenge type",
		}, ErrInvalidChallenge
	}

	if !success {
		vs.captcha.deleteCaptchaData(ctx, req.Token)
		return &VerifyResult{
			Success:       false,
			Score:         positionScore,
			RiskLevel:     risk.RiskLevelHigh,
			RiskScore:     int(100 - positionScore),
			Message:       message,
			PositionScore: positionScore,
		}, nil
	}

	riskResult := vs.analyzeBehavior(req.BehaviorData)

	if riskResult.Recommended == risk.ActionBlock {
		vs.captcha.deleteCaptchaData(ctx, req.Token)
		return &VerifyResult{
			Success:      false,
			Score:        positionScore,
			RiskLevel:    riskResult.Level,
			RiskScore:    riskResult.Score,
			Message:      "risk detection: blocked",
			Factors:      riskResult.Factors,
			PositionScore: positionScore,
		}, nil
	}

	combinedScore := (positionScore*0.6 + float64(100-riskResult.Score)*0.4)

	if combinedScore < 60 {
		vs.captcha.deleteCaptchaData(ctx, req.Token)
		return &VerifyResult{
			Success:      false,
			Score:        combinedScore,
			RiskLevel:    riskResult.Level,
			RiskScore:    riskResult.Score,
			Message:      "behavior analysis failed",
			Factors:      riskResult.Factors,
			PositionScore: positionScore,
		}, nil
	}

	vs.captcha.deleteCaptchaData(ctx, req.Token)

	return &VerifyResult{
		Success:      true,
		Score:        combinedScore,
		RiskLevel:    riskResult.Level,
		RiskScore:    riskResult.Score,
		Message:      "verification successful",
		Factors:      riskResult.Factors,
		PositionScore: positionScore,
	}, nil
}

func (vs *VerifyService) verifyClickOrder(captchaData *CaptchaData, clicks []ClickInput) (float64, bool, string) {
	if len(clicks) == 0 {
		return 0, false, "no clicks provided"
	}

	if len(clicks) != len(captchaData.Targets) {
		return 0, false, fmt.Sprintf("expected %d clicks, got %d", len(captchaData.Targets), len(clicks))
	}

	correctCount := 0
	totalScore := 0.0

	for i, click := range clicks {
		if click.Index != i {
			continue
		}

		minDistance := float64(vs.tolerance + 10)
		for _, target := range captchaData.Targets {
			dx := float64(click.X - target.X)
			dy := float64(click.Y - target.Y)
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance < minDistance {
				minDistance = distance
			}
		}

		clickScore := math.Max(0, 100-minDistance*2)
		totalScore += clickScore

		if minDistance <= float64(vs.tolerance) {
			correctCount++
		}
	}

	avgScore := totalScore / float64(len(clicks))

	if correctCount == len(captchaData.Targets) {
		return avgScore, true, "all clicks correct"
	}

	return avgScore, false, fmt.Sprintf("%d/%d clicks correct", correctCount, len(captchaData.Targets))
}

func (vs *VerifyService) verifyDragPath(captchaData *CaptchaData, path []DragPoint) (float64, bool, string) {
	if len(path) == 0 {
		return 0, false, "no drag path provided"
	}

	if len(path) < 5 {
		return 0, false, "drag path too short"
	}

	targets := captchaData.Targets
	if len(targets) < 2 {
		return 0, false, "invalid target configuration"
	}

	pathLength := vs.calculatePathLength(path)
	smoothness := vs.calculatePathSmoothness(path)

	directDistance := vs.calculateDistance(
		DragPoint{X: targets[0].X, Y: targets[0].Y},
		DragPoint{X: targets[len(targets)-1].X, Y: targets[len(targets)-1].Y},
	)

	pathRatio := directDistance / (pathLength + 1)
	smoothnessScore := smoothness * 100

	startPointScore := 0.0
	if vs.distanceToPoint(path[0], targets[0]) <= float64(vs.tolerance) {
		startPointScore = 50
	}

	endPointScore := 0.0
	if vs.distanceToPoint(path[len(path)-1], targets[len(targets)-1]) <= float64(vs.tolerance) {
		endPointScore = 50
	}

	totalScore := (pathRatio*30 + smoothnessScore*0.3 + startPointScore*0.2 + endPointScore*0.2)

	success := startPointScore > 0 && endPointScore > 0 && totalScore >= 60

	message := "path verification"
	if !success {
		message = fmt.Sprintf("path score %.1f too low", totalScore)
	}

	return totalScore, success, message
}

func (vs *VerifyService) verifyHoverSequence(captchaData *CaptchaData, hovers []HoverInput) (float64, bool, string) {
	if len(hovers) == 0 {
		return 0, false, "no hover data provided"
	}

	targets := captchaData.Targets
	if len(targets) != len(hovers) {
		return 0, false, fmt.Sprintf("expected %d hovers, got %d", len(targets), len(hovers))
	}

	correctCount := 0
	totalScore := 0.0

	for _, hover := range hovers {
		minDistance := float64(vs.tolerance + 5)
		for _, target := range targets {
			dx := float64(hover.X - target.X)
			dy := float64(hover.Y - target.Y)
			distance := math.Sqrt(dx*dx + dy*dy)
			if distance < minDistance {
				minDistance = distance
			}
		}

		hoverScore := math.Max(0, 100-minDistance*3)
		totalScore += hoverScore

		if minDistance <= float64(vs.tolerance) {
			correctCount++
		}
	}

	avgScore := totalScore / float64(len(hovers))

	successRatio := float64(correctCount) / float64(len(targets))
	success := successRatio >= 0.8

	message := "hover verification"
	if !success {
		message = fmt.Sprintf("%d/%d hovers correct", correctCount, len(targets))
	}

	return avgScore, success, message
}

func (vs *VerifyService) analyzeBehavior(behavior *BehaviorInput) *risk.RiskResult {
	if behavior == nil {
		return &risk.RiskResult{
			Score:      0,
			Level:      risk.RiskLevelLow,
			Recommended: risk.ActionAllow,
			Factors:    []risk.RiskFactor{},
		}
	}

	riskData := &risk.BehaviorData{
		MouseTracks: vs.convertMouseTracks(behavior.MouseTracks),
		ClickEvents: vs.convertClickEvents(behavior.ClickEvents),
	}

	return vs.riskEngine.EnhancedCalculateRiskScore(context.Background(), riskData, "", "")
}

func (vs *VerifyService) convertMouseTracks(tracks []MouseTrack) []risk.MouseTrack {
	if len(tracks) == 0 {
		return nil
	}

	result := make([]risk.MouseTrack, len(tracks))
	for i, t := range tracks {
		result[i] = risk.MouseTrack{
			X:         t.X,
			Y:         t.Y,
			Timestamp: t.Timestamp,
			Velocity:  t.Velocity,
		}
	}
	return result
}

func (vs *VerifyService) convertClickEvents(events []ClickEvent) []risk.ClickEvent {
	if len(events) == 0 {
		return nil
	}

	result := make([]risk.ClickEvent, len(events))
	for i, e := range events {
		result[i] = risk.ClickEvent{
			X:         e.X,
			Y:         e.Y,
			Timestamp: e.Timestamp,
			Duration:  e.Duration,
			Pressure:  e.Pressure,
		}
	}
	return result
}

func (vs *VerifyService) calculatePathLength(path []DragPoint) float64 {
	if len(path) < 2 {
		return 0
	}

	var length float64
	for i := 1; i < len(path); i++ {
		dx := float64(path[i].X - path[i-1].X)
		dy := float64(path[i].Y - path[i-1].Y)
		length += math.Sqrt(dx*dx + dy*dy)
	}
	return length
}

func (vs *VerifyService) calculateDistance(p1, p2 DragPoint) float64 {
	dx := float64(p2.X - p1.X)
	dy := float64(p2.Y - p1.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (vs *VerifyService) distanceToPoint(p DragPoint, target Point) float64 {
	dx := float64(p.X - target.X)
	dy := float64(p.Y - target.Y)
	return math.Sqrt(dx*dx + dy*dy)
}

func (vs *VerifyService) calculatePathSmoothness(path []DragPoint) float64 {
	if len(path) < 3 {
		return 1.0
	}

	var totalAngleChange float64
	for i := 1; i < len(path)-1; i++ {
		angle1 := math.Atan2(
			float64(path[i].Y-path[i-1].Y),
			float64(path[i].X-path[i-1].X),
		)
		angle2 := math.Atan2(
			float64(path[i+1].Y-path[i].Y),
			float64(path[i+1].X-path[i].X),
		)

		angleDiff := math.Abs(angle2 - angle1)
		if angleDiff > math.Pi {
			angleDiff = 2*math.Pi - angleDiff
		}
		totalAngleChange += angleDiff
	}

	avgAngleChange := totalAngleChange / float64(len(path)-2)
	smoothness := 1.0 - (avgAngleChange / math.Pi)

	return math.Max(0, math.Min(1, smoothness))
}

func (vs *VerifyService) SetTolerance(tolerance int) {
	if tolerance > 0 {
		vs.tolerance = tolerance
	}
}

func (vs *VerifyService) GetTolerance() int {
	return vs.tolerance
}
