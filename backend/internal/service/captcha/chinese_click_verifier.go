package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"sort"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type ClickPoint struct {
	X      int       `json:"x"`
	Y      int       `json:"y"`
	Time   int64     `json:"time"`
	Index  int       `json:"index"`
	Target bool      `json:"target"`
}

type VerifyChineseClickRequest struct {
	SessionID string      `json:"session_id" binding:"required"`
	Board     *ChineseClickBoard `json:"board" binding:"required"`
	Clicks    []ClickPoint `json:"clicks" binding:"required"`
	RiskScore float64     `json:"risk_score"`
}

type VerifyChineseClickResult struct {
	Success       bool    `json:"success"`
	Message       string  `json:"message"`
	Score         float64 `json:"score"`
	Accuracy      float64 `json:"accuracy"`
	TimingScore   float64 `json:"timing_score"`
	SequenceScore float64 `json:"sequence_score"`
}

type ChineseClickVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewChineseClickVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *ChineseClickVerifierService {
	return &ChineseClickVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *ChineseClickVerifierService) Verify(ctx context.Context, req *VerifyChineseClickRequest) (*VerifyChineseClickResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyChineseClickResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyChineseClickResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyChineseClickResult{
			Success:       true,
			Message:       "验证码已验证通过",
			Score:         100,
			Accuracy:      100,
			TimingScore:   100,
			SequenceScore: 100,
		}, nil
	}

	originalBoard := &ChineseClickBoard{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalBoard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	result := v.validateClicks(req.Clicks, originalBoard)

	if result.Success {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
	}

	return result, nil
}

func (v *ChineseClickVerifierService) validateClicks(clicks []ClickPoint, board *ChineseClickBoard) *VerifyChineseClickResult {
	if len(clicks) == 0 {
		return &VerifyChineseClickResult{
			Success:       false,
			Message:       "未检测到点击",
			Score:         0,
			Accuracy:      0,
			TimingScore:   0,
			SequenceScore: 0,
		}
	}

	targetChars := make(map[string]bool)
	for _, t := range board.Targets {
		targetChars[t.Char] = true
	}

	targetCount := len(board.Targets)

	matchedTargets, accuracy := v.matchClicksToTargets(clicks, board.Targets)

	timingScore := v.analyzeClickTiming(clicks)

	sequenceScore := v.analyzeClickSequence(clicks, board.Targets)

	faultToleranceScore := v.calculateFaultTolerance(clicks, board.Targets, targetCount)

	overallScore := (accuracy*0.4 + timingScore*0.2 + sequenceScore*0.2 + faultToleranceScore*0.2)

	isSuccess := matchedTargets >= targetCount && overallScore >= 60

	return &VerifyChineseClickResult{
		Success:       isSuccess,
		Message:       v.getMessage(isSuccess, matchedTargets, targetCount),
		Score:         math.Round(overallScore*100) / 100,
		Accuracy:      math.Round(accuracy*100) / 100,
		TimingScore:   math.Round(timingScore*100) / 100,
		SequenceScore: math.Round(sequenceScore*100) / 100,
	}
}

func (v *ChineseClickVerifierService) matchClicksToTargets(clicks []ClickPoint, targets []ChineseClickTarget) (int, float64) {
	if len(clicks) == 0 || len(targets) == 0 {
		return 0, 0
	}

	targetsCopy := make([]ChineseClickTarget, len(targets))
	copy(targetsCopy, targets)

	matchedCount := 0
	totalDistance := 0.0

	for _, click := range clicks {
		bestMatch, distance := v.findBestTargetMatch(click, targetsCopy)
		if bestMatch != nil && distance < float64(bestMatch.Width)*0.8 {
			matchedCount++
			totalDistance += distance

			for i, t := range targetsCopy {
				if t.Index == bestMatch.Index {
					targetsCopy = append(targetsCopy[:i], targetsCopy[i+1:]...)
					break
				}
			}
		}
	}

	accuracy := float64(matchedCount) / float64(len(targets))
	if matchedCount > 0 {
		avgDistance := totalDistance / float64(matchedCount)
		distanceFactor := 1.0 - math.Min(avgDistance/50.0, 1.0)
		accuracy *= distanceFactor
	}

	return matchedCount, accuracy * 100
}

func (v *ChineseClickVerifierService) findBestTargetMatch(click ClickPoint, targets []ChineseClickTarget) (*ChineseClickTarget, float64) {
	var bestMatch *ChineseClickTarget
	minDistance := math.MaxFloat64

	for _, target := range targets {
		distance := v.calculateDistance(click.X, click.Y, target.X+target.Width/2, target.Y+target.Height/2)
		if distance < minDistance {
			minDistance = distance
			bestMatch = &target
		}
	}

	return bestMatch, minDistance
}

func (v *ChineseClickVerifierService) calculateDistance(x1, y1, x2, y2 int) float64 {
	dx := x1 - x2
	dy := y1 - y2
	return math.Sqrt(float64(dx*dx + dy*dy))
}

func (v *ChineseClickVerifierService) analyzeClickTiming(clicks []ClickPoint) float64 {
	if len(clicks) < 2 {
		return 100
	}

	sortedClicks := make([]ClickPoint, len(clicks))
	copy(sortedClicks, clicks)
	sort.Slice(sortedClicks, func(i, j int) bool {
		return sortedClicks[i].Time < sortedClicks[j].Time
	})

	timeDifferences := make([]int64, 0, len(sortedClicks)-1)
	for i := 1; i < len(sortedClicks); i++ {
		diff := sortedClicks[i].Time - sortedClicks[i-1].Time
		timeDifferences = append(timeDifferences, diff)
	}

	avgTime := int64(0)
	for _, diff := range timeDifferences {
		avgTime += diff
	}
	avgTime /= int64(len(timeDifferences))

	minTime := int64(200)
	maxTime := int64(5000)

	if avgTime < minTime {
		return 30
	}
	if avgTime > maxTime {
		return 50
	}

	idealRange := int64(500)
	if avgTime >= idealRange-200 && avgTime <= idealRange+300 {
		return 100
	}

	return 70 + (1.0-math.Abs(float64(avgTime-idealRange)/2000.0))*30
}

func (v *ChineseClickVerifierService) analyzeClickSequence(clicks []ClickPoint, targets []ChineseClickTarget) float64 {
	if len(clicks) == 0 || len(targets) == 0 {
		return 100
	}

	targetCenters := make([]struct{ x, y, index int }, len(targets))
	for i, t := range targets {
		targetCenters[i] = struct{ x, y, index int }{
			x:     t.X + t.Width/2,
			y:     t.Y + t.Height/2,
			index: t.Index,
		}
	}

	if len(clicks) >= 2 {
		clickPathLength := 0.0
		for i := 1; i < len(clicks); i++ {
			clickPathLength += v.calculateDistance(
				clicks[i-1].X, clicks[i-1].Y,
				clicks[i].X, clicks[i].Y,
			)
		}

		optimalPath := v.calculateOptimalPath(targetCenters)

		if optimalPath > 0 && clickPathLength > 0 {
			ratio := optimalPath / clickPathLength
			if ratio > 1 {
				ratio = 1
			}
			return ratio * 100
		}
	}

	return 80
}

func (v *ChineseClickVerifierService) calculateOptimalPath(points []struct{ x, y, index int }) float64 {
	if len(points) < 2 {
		return 0
	}

	visited := make([]bool, len(points))
	totalDistance := 0.0
	currentIndex := 0
	visited[0] = true
	visitedCount := 1

	for visitedCount < len(points) {
		minDistance := math.MaxFloat64
		nextIndex := -1

		for i := 0; i < len(points); i++ {
			if !visited[i] {
				distance := v.calculateDistance(points[currentIndex].x, points[currentIndex].y, points[i].x, points[i].y)
				if distance < minDistance {
					minDistance = distance
					nextIndex = i
				}
			}
		}

		if nextIndex != -1 {
			totalDistance += minDistance
			visited[nextIndex] = true
			currentIndex = nextIndex
			visitedCount++
		} else {
			break
		}
	}

	return totalDistance
}

func (v *ChineseClickVerifierService) calculateFaultTolerance(clicks []ClickPoint, targets []ChineseClickTarget, expectedCount int) float64 {
	if len(clicks) == 0 {
		return 0
	}

	targetRadius := float64(targets[0].Width) / 2

	correctClicks := 0
	falseClicks := 0

	for _, click := range clicks {
		isInTarget := false
		for _, target := range targets {
			distance := v.calculateDistance(click.X, click.Y, target.X+target.Width/2, target.Y+target.Height/2)
			if distance <= targetRadius {
				isInTarget = true
				break
			}
		}

		if isInTarget {
			correctClicks++
		} else {
			falseClicks++
		}
	}

	extraClickPenalty := math.Min(float64(falseClicks)*10, 30)

	if correctClicks >= expectedCount {
		return math.Max(0, 100-extraClickPenalty)
	}

	return float64(correctClicks) / float64(expectedCount) * (100 - extraClickPenalty)
}

func (v *ChineseClickVerifierService) getMessage(isSuccess bool, matched, expected int) string {
	if isSuccess {
		return "验证成功"
	}

	if matched == 0 {
		return "未点击任何目标"
	}

	if matched < expected {
		return fmt.Sprintf("只点击了 %d/%d 个目标", matched, expected)
	}

	return "点击精度不足，请重试"
}

func (v *ChineseClickVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if v.sessionCache != nil {
		session, err := v.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		session, err := v.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (v *ChineseClickVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *ChineseClickVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *ChineseClickVerifierService) GetSession(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *ChineseClickVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *ChineseClickVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := v.getSession(sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "验证码已过期"
	}

	if session.Status == "verified" {
		return false, "验证码已验证通过"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, ""
}