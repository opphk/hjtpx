package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type HapticVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type HapticVerifyRequest struct {
	SessionID string           `json:"session_id" binding:"required"`
	UserInput *HapticUserInput `json:"user_input"`
}

type HapticUserInput struct {
	Sequence   []int               `json:"sequence"`
	Timestamps []int64             `json:"timestamps"`
	Pressures  []float64           `json:"pressures"`
}

type HapticVerifyResult struct {
	Success    bool    `json:"success"`
	Message    string  `json:"message"`
	MatchScore float64 `json:"match_score"`
	MatchLevel string  `json:"match_level"`
}

const (
	HapticMatchLevelHigh   = "high"
	HapticMatchLevelMedium = "medium"
	HapticMatchLevelLow    = "low"
)

func NewHapticVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *HapticVerifierService {
	return &HapticVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *HapticVerifierService) Verify(ctx context.Context, req *HapticVerifyRequest) (*HapticVerifyResult, error) {
	session, err := v.getSession(ctx, req.SessionID)
	if err != nil {
		return &HapticVerifyResult{
			Success: false,
			Message: "Session not found",
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &HapticVerifyResult{
			Success: false,
			Message: "验证码已过期",
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &HapticVerifyResult{
			Success: false,
			Message: "验证次数已用完",
		}, nil
	}

	v.incrementVerifyCount(ctx, session.SessionID)

	if session.Status == "verified" {
		return &HapticVerifyResult{
			Success:    true,
			Message:    "验证码已验证通过",
			MatchScore: session.MatchScore,
			MatchLevel: v.getMatchLevel(session.MatchScore),
		}, nil
	}

	var pattern HapticPattern
	if err := json.Unmarshal([]byte(session.Pattern), &pattern); err != nil {
		return &HapticVerifyResult{
			Success: false,
			Message: "Pattern parsing failed",
		}, nil
	}

	if req.UserInput == nil {
		return &HapticVerifyResult{
			Success: false,
			Message: "Invalid input data",
		}, nil
	}

	matchScore := v.calculateMatchScore(req.UserInput, &pattern)
	matchLevel := v.getMatchLevel(matchScore)

	v.updateMatchScore(ctx, session.SessionID, matchScore)

	if matchScore >= 0.75 {
		v.markAsVerified(ctx, session.SessionID)
		return &HapticVerifyResult{
			Success:    true,
			Message:    "触觉验证成功",
			MatchScore: matchScore,
			MatchLevel: matchLevel,
		}, nil
	}

	return &HapticVerifyResult{
		Success:    false,
		Message:    fmt.Sprintf("触觉模式不匹配，匹配度 %.0f%%", matchScore*100),
		MatchScore: matchScore,
		MatchLevel: matchLevel,
	}, nil
}

func (v *HapticVerifierService) getSession(ctx context.Context, sessionID string) (*models.HapticCaptchaSession, error) {
	if v.sessionCache != nil {
		if session, err := v.sessionCache.GetHaptic(ctx, sessionID); err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		if session, err := v.captchaRepo.GetHapticSession(sessionID); err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found")
}

func (v *HapticVerifierService) incrementVerifyCount(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		if err := v.sessionCache.IncrementHapticVerifyCount(ctx, sessionID); err != nil {
			log.Printf("增加触觉验证计数失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.IncrementHapticVerifyCount(sessionID); err != nil {
			log.Printf("数据库增加触觉验证计数失败: %v", err)
		}
	}
}

func (v *HapticVerifierService) updateMatchScore(ctx context.Context, sessionID string, score float64) {
	if v.sessionCache != nil {
		if err := v.sessionCache.UpdateHapticMatchScore(ctx, sessionID, score); err != nil {
			log.Printf("更新触觉匹配分数失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.UpdateHapticMatchScore(sessionID, score); err != nil {
			log.Printf("数据库更新触觉匹配分数失败: %v", err)
		}
	}
}

func (v *HapticVerifierService) markAsVerified(ctx context.Context, sessionID string) {
	if v.sessionCache != nil {
		if err := v.sessionCache.MarkHapticAsVerified(ctx, sessionID); err != nil {
			log.Printf("缓存标记触觉验证失败: %v", err)
		}
	}

	if v.captchaRepo != nil {
		if err := v.captchaRepo.MarkHapticAsVerified(sessionID); err != nil {
			log.Printf("数据库标记触觉验证失败: %v", err)
		}
	}
}

func (v *HapticVerifierService) calculateMatchScore(userInput *HapticUserInput, pattern *HapticPattern) float64 {
	if len(userInput.Sequence) == 0 || len(pattern.TargetSequence) == 0 {
		return 0
	}

	sequenceScore := v.calculateSequenceScore(userInput.Sequence, pattern.TargetSequence)

	timingScore := v.calculateTimingScore(userInput, pattern)

	pressureScore := v.calculatePressureScore(userInput, pattern)

	totalScore := sequenceScore*0.6 + timingScore*0.2 + pressureScore*0.2

	return math.Min(1.0, math.Max(0.0, totalScore))
}

func (v *HapticVerifierService) calculateSequenceScore(userSequence, targetSequence []int) float64 {
	if len(userSequence) != len(targetSequence) {
		return 0
	}

	matches := 0
	for i := range userSequence {
		if userSequence[i] == targetSequence[i] {
			matches++
		}
	}

	return float64(matches) / float64(len(targetSequence))
}

func (v *HapticVerifierService) calculateTimingScore(userInput *HapticUserInput, pattern *HapticPattern) float64 {
	if len(userInput.Timestamps) < 2 || len(pattern.Taps) < 2 {
		return 1.0
	}

	expectedInterval := 0.0
	for i := 1; i < len(pattern.Taps); i++ {
		expectedInterval += pattern.Taps[i].Duration
	}
	expectedInterval /= float64(len(pattern.Taps) - 1)

	var totalDeviation float64
	validIntervals := 0

	for i := 1; i < len(userInput.Timestamps) && i < len(pattern.Taps); i++ {
		actualInterval := float64(userInput.Timestamps[i] - userInput.Timestamps[i-1])
		deviation := math.Abs(actualInterval - expectedInterval)
		totalDeviation += deviation
		validIntervals++
	}

	if validIntervals == 0 {
		return 1.0
	}

	avgDeviation := totalDeviation / float64(validIntervals)
	tolerance := expectedInterval * 0.5

	score := math.Max(0, 1-avgDeviation/tolerance)
	return score
}

func (v *HapticVerifierService) calculatePressureScore(userInput *HapticUserInput, pattern *HapticPattern) float64 {
	if len(userInput.Pressures) != len(pattern.Taps) {
		return 1.0
	}

	var totalDeviation float64
	for i := range userInput.Pressures {
		expectedPressure := pattern.Taps[i].Pressure
		actualPressure := userInput.Pressures[i]
		deviation := math.Abs(actualPressure - expectedPressure)
		totalDeviation += deviation
	}

	avgDeviation := totalDeviation / float64(len(pattern.Taps))
	score := math.Max(0, 1-avgDeviation)

	return score
}

func (v *HapticVerifierService) getMatchLevel(score float64) string {
	if score >= 0.9 {
		return HapticMatchLevelHigh
	} else if score >= 0.75 {
		return HapticMatchLevelMedium
	}
	return HapticMatchLevelLow
}

func (v *HapticVerifierService) AnalyzeHapticPattern(userInput *HapticUserInput) map[string]interface{} {
	analysis := make(map[string]interface{})

	if len(userInput.Sequence) > 0 {
		analysis["sequence_length"] = len(userInput.Sequence)
		analysis["unique_positions"] = v.countUnique(userInput.Sequence)
	}

	if len(userInput.Timestamps) > 1 {
		var totalInterval int64
		for i := 1; i < len(userInput.Timestamps); i++ {
			totalInterval += userInput.Timestamps[i] - userInput.Timestamps[i-1]
		}
		avgInterval := float64(totalInterval) / float64(len(userInput.Timestamps)-1)
		analysis["average_interval_ms"] = avgInterval
	}

	if len(userInput.Pressures) > 0 {
		var sum, sqSum float64
		for _, p := range userInput.Pressures {
			sum += p
			sqSum += p * p
		}
		mean := sum / float64(len(userInput.Pressures))
		variance := (sqSum / float64(len(userInput.Pressures))) - (mean * mean)
		analysis["pressure_mean"] = mean
		analysis["pressure_variance"] = variance
	}

	return analysis
}

func (v *HapticVerifierService) countUnique(arr []int) int {
	seen := make(map[int]bool)
	count := 0
	for _, n := range arr {
		if !seen[n] {
			seen[n] = true
			count++
		}
	}
	return count
}

func (v *HapticVerifierService) ValidateInput(userInput *HapticUserInput) (bool, string) {
	if userInput == nil {
		return false, "输入数据不能为空"
	}

	if len(userInput.Sequence) == 0 {
		return false, "序列不能为空"
	}

	if len(userInput.Sequence) > 20 {
		return false, "序列长度超出限制"
	}

	for _, pos := range userInput.Sequence {
		if pos < 0 || pos >= 36 {
			return false, "无效的位置索引"
		}
	}

	if len(userInput.Timestamps) > 0 && len(userInput.Timestamps) != len(userInput.Sequence) {
		return false, "时间戳数量与序列长度不匹配"
	}

	for _, ts := range userInput.Timestamps {
		if ts < 0 {
			return false, "无效的时间戳"
		}
	}

	if len(userInput.Pressures) > 0 && len(userInput.Pressures) != len(userInput.Sequence) {
		return false, "压力数据数量与序列长度不匹配"
	}

	for _, p := range userInput.Pressures {
		if p < 0 || p > 1 {
			return false, "压力值必须在0-1之间"
		}
	}

	return true, "验证通过"
}

func (v *HapticVerifierService) GetSessionForStatus(ctx context.Context, sessionID string) (*models.HapticCaptchaSession, error) {
	return v.getSession(ctx, sessionID)
}
