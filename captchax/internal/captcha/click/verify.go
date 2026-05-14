package click

import (
	"context"
	"fmt"
	"math"
	"sort"
)

type ClickVerifier struct {
	cache CaptchaCache
}

func NewClickVerifier(cache CaptchaCache) *ClickVerifier {
	return &ClickVerifier{
		cache: cache,
	}
}

func (cv *ClickVerifier) Verify(ctx context.Context, req *VerifyRequest) (*VerifyResponse, error) {
	if req.CaptchaID == "" {
		return &VerifyResponse{
			Success: false,
			Score:   0,
			Message: "captcha ID is required",
		}, nil
	}

	if len(req.Clicks) == 0 {
		return &VerifyResponse{
			Success: false,
			Score:   0,
			Message: "no click positions provided",
		}, nil
	}

	captchaData, err := cv.cache.Get(ctx, req.CaptchaID)
	if err != nil {
		return &VerifyResponse{
			Success: false,
			Score:   0,
			Message: fmt.Sprintf("captcha expired or not found: %v", err),
		}, nil
	}

	if len(req.Clicks) != len(captchaData.TargetChars) {
		return &VerifyResponse{
			Success: false,
			Score:   0,
			Message: fmt.Sprintf("invalid click count: expected %d, got %d", len(captchaData.TargetChars), len(req.Clicks)),
		}, nil
	}

	charMatchResults := cv.findClickedChars(req.Clicks, captchaData.CharPositions)

	orderCorrect, orderMessage := cv.verifyClickOrder(req.Clicks, captchaData.TargetChars, charMatchResults)
	if !orderCorrect {
		return &VerifyResponse{
			Success: false,
			Score:   0,
			Message: orderMessage,
		}, nil
	}

	positionScore := cv.calculatePositionAccuracy(req.Clicks, captchaData.CharPositions, charMatchResults)

	if positionScore >= 0.8 {
		cv.cache.Delete(ctx, req.CaptchaID)
		return &VerifyResponse{
			Success: true,
			Score:   positionScore,
			Message: "verification passed",
		}, nil
	} else if positionScore >= 0.5 {
		return &VerifyResponse{
			Success: false,
			Score:   positionScore,
			Message: "verification failed: click position not accurate enough",
		}, nil
	} else {
		return &VerifyResponse{
			Success: false,
			Score:   positionScore,
			Message: "verification failed",
		}, nil
	}
}

type charMatchResult struct {
	clickIndex   int
	charIndex    int
	clickedChar  string
	charPosition CharPosition
	distance     float64
}

func (cv *ClickVerifier) findClickedChars(clicks []ClickPosition, charPositions []CharPosition) []charMatchResult {
	var results []charMatchResult

	for i, click := range clicks {
		for j, charPos := range charPositions {
			charCenterX := charPos.X + charPos.Width/2
			charCenterY := charPos.Y + charPos.Height/2

			dx := float64(click.X - charCenterX)
			dy := float64(click.Y - charCenterY)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance <= float64(ClickTolerance+30) {
				results = append(results, charMatchResult{
					clickIndex:   i,
					charIndex:    j,
					clickedChar:  charPos.Char,
					charPosition: charPos,
					distance:     distance,
				})
			}
		}
	}

	return results
}

func (cv *ClickVerifier) verifyClickOrder(clicks []ClickPosition, targetChars []string, matches []charMatchResult) (bool, string) {
	if len(clicks) != len(targetChars) {
		return false, fmt.Sprintf("click count mismatch: expected %d, got %d", len(targetChars), len(clicks))
	}

	clickedChars := make([]string, len(clicks))

	for _, match := range matches {
		if match.clickIndex < len(clickedChars) {
			clickedChars[match.clickIndex] = match.clickedChar
		}
	}

	for i := 0; i < len(targetChars); i++ {
		if clickedChars[i] == "" {
			return false, fmt.Sprintf("no character clicked at position %d", i+1)
		}
		if clickedChars[i] != targetChars[i] {
			return false, fmt.Sprintf("click order incorrect: expected '%s', got '%s' at position %d", targetChars[i], clickedChars[i], i+1)
		}
	}

	return true, ""
}

func (cv *ClickVerifier) calculatePositionAccuracy(clicks []ClickPosition, charPositions []CharPosition, matches []charMatchResult) float64 {
	if len(clicks) == 0 || len(charPositions) == 0 {
		return 0
	}

	usedClicks := make(map[int]bool)
	usedChars := make(map[int]bool)
	correctMatches := 0

	sortedMatches := make([]charMatchResult, len(matches))
	copy(sortedMatches, matches)

	sort.Slice(sortedMatches, func(i, j int) bool {
		return sortedMatches[i].distance < sortedMatches[j].distance
	})

	for _, match := range sortedMatches {
		if usedClicks[match.clickIndex] || usedChars[match.charIndex] {
			continue
		}

		if match.distance <= float64(ClickTolerance+30) {
			usedClicks[match.clickIndex] = true
			usedChars[match.charIndex] = true
			correctMatches++
		}
	}

	expectedMatches := len(charPositions)
	if expectedMatches == 0 {
		return 0
	}

	score := float64(correctMatches) / float64(expectedMatches)

	return score
}

func VerifyWithTolerance(x1, y1, x2, y2, tolerance int) bool {
	dx := x1 - x2
	dy := y1 - y2
	distanceSquared := dx*dx + dy*dy
	toleranceSquared := tolerance * tolerance

	return distanceSquared <= toleranceSquared
}

func CalculateDistance(x1, y1, x2, y2 int) float64 {
	dx := float64(x1 - x2)
	dy := float64(y1 - y2)
	return math.Sqrt(dx*dx + dy*dy)
}
