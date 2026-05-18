package captcha

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestPickRandomChars(t *testing.T) {
	generator := NewChineseClickGeneratorService(nil, nil)
	
	chars := generator.pickRandomChars(4)
	assert.Equal(t, 4, len(chars))
	
	chars2 := generator.pickRandomChars(4)
	assert.Equal(t, 4, len(chars2))
	
	used := make(map[string]bool)
	for _, c := range chars {
		assert.False(t, used[c], "duplicate char: %s", c)
		used[c] = true
	}
}

func TestGenerateAllChars(t *testing.T) {
	generator := NewChineseClickGeneratorService(nil, nil)
	
	targetChars := []string{"你", "我", "他", "她"}
	allChars := generator.generateAllChars(targetChars, 12)
	
	assert.Equal(t, 12, len(allChars))
	
	targetSet := make(map[string]bool)
	for _, c := range targetChars {
		targetSet[c] = true
	}
	
	count := 0
	for _, c := range allChars {
		if targetSet[c] {
			count++
		}
	}
	assert.Equal(t, 4, count)
}

func TestMatchClicksToTargets(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	targets := []ChineseClickTarget{
		{Char: "你", X: 100, Y: 100, Width: 30, Height: 30, Index: 0},
		{Char: "我", X: 200, Y: 100, Width: 30, Height: 30, Index: 1},
		{Char: "他", X: 100, Y: 200, Width: 30, Height: 30, Index: 2},
		{Char: "她", X: 200, Y: 200, Width: 30, Height: 30, Index: 3},
	}
	
	clicks := []ClickPoint{
		{X: 115, Y: 115, Time: 1000},
		{X: 215, Y: 115, Time: 1500},
		{X: 115, Y: 215, Time: 2000},
		{X: 215, Y: 215, Time: 2500},
	}
	
	matched, accuracy := verifier.matchClicksToTargets(clicks, targets)
	assert.Equal(t, 4, matched)
	assert.GreaterOrEqual(t, accuracy, 90.0)
}

func TestMatchClicksToTargetsWithError(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	targets := []ChineseClickTarget{
		{Char: "你", X: 100, Y: 100, Width: 30, Height: 30, Index: 0},
		{Char: "我", X: 200, Y: 100, Width: 30, Height: 30, Index: 1},
	}
	
	clicks := []ClickPoint{
		{X: 150, Y: 150, Time: 1000},
		{X: 250, Y: 250, Time: 1500},
	}
	
	matched, accuracy := verifier.matchClicksToTargets(clicks, targets)
	assert.Less(t, matched, 2)
	assert.Less(t, accuracy, 50.0)
}

func TestAnalyzeClickTiming(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	tests := []struct {
		name        string
		clicks      []ClickPoint
		expectHigh  bool
	}{
		{
			name: "normal timing",
			clicks: []ClickPoint{
				{X: 100, Y: 100, Time: 0},
				{X: 200, Y: 200, Time: 500},
				{X: 300, Y: 300, Time: 1000},
			},
			expectHigh: true,
		},
		{
			name: "too fast",
			clicks: []ClickPoint{
				{X: 100, Y: 100, Time: 0},
				{X: 200, Y: 200, Time: 50},
				{X: 300, Y: 300, Time: 100},
			},
			expectHigh: false,
		},
		{
			name: "single click",
			clicks: []ClickPoint{
				{X: 100, Y: 100, Time: 0},
			},
			expectHigh: true,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := verifier.analyzeClickTiming(tt.clicks)
			if tt.expectHigh {
				assert.GreaterOrEqual(t, score, 70.0)
			} else {
				assert.Less(t, score, 50.0)
			}
		})
	}
}

func TestAnalyzeClickSequence(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	targets := []ChineseClickTarget{
		{Char: "A", X: 50, Y: 50, Width: 30, Height: 30, Index: 0},
		{Char: "B", X: 150, Y: 50, Width: 30, Height: 30, Index: 1},
		{Char: "C", X: 250, Y: 50, Width: 30, Height: 30, Index: 2},
	}
	
	optimalClicks := []ClickPoint{
		{X: 65, Y: 65, Time: 0},
		{X: 165, Y: 65, Time: 500},
		{X: 265, Y: 65, Time: 1000},
	}
	
	score := verifier.analyzeClickSequence(optimalClicks, targets)
	assert.GreaterOrEqual(t, score, 80.0)
}

func TestCalculateFaultTolerance(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	targets := []ChineseClickTarget{
		{Char: "你", X: 100, Y: 100, Width: 30, Height: 30, Index: 0},
		{Char: "我", X: 200, Y: 100, Width: 30, Height: 30, Index: 1},
	}
	
	tests := []struct {
		name         string
		clicks       []ClickPoint
		expected     int
		expectHigh   bool
	}{
		{
			name: "all correct",
			clicks: []ClickPoint{
				{X: 115, Y: 115, Time: 0},
				{X: 215, Y: 115, Time: 500},
			},
			expected:   2,
			expectHigh: true,
		},
		{
			name: "one false click",
			clicks: []ClickPoint{
				{X: 115, Y: 115, Time: 0},
				{X: 215, Y: 115, Time: 500},
				{X: 50, Y: 50, Time: 1000},
			},
			expected:   2,
			expectHigh: true,
		},
		{
			name: "missing one target",
			clicks: []ClickPoint{
				{X: 115, Y: 115, Time: 0},
			},
			expected:   2,
			expectHigh: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := verifier.calculateFaultTolerance(tt.clicks, targets, tt.expected)
			if tt.expectHigh {
				assert.GreaterOrEqual(t, score, 50.0)
			} else {
				assert.LessOrEqual(t, score, 50.0)
			}
		})
	}
}

func TestValidateClicks(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	board := &ChineseClickBoard{
		Width:      400,
		Height:     300,
		TargetChars: "你我他她",
		TotalChars: 12,
		Targets: []ChineseClickTarget{
			{Char: "你", X: 80, Y: 80, Width: 30, Height: 30, Index: 0},
			{Char: "我", X: 180, Y: 80, Width: 30, Height: 30, Index: 1},
			{Char: "他", X: 80, Y: 180, Width: 30, Height: 30, Index: 2},
			{Char: "她", X: 180, Y: 180, Width: 30, Height: 30, Index: 3},
		},
	}
	
	tests := []struct {
		name     string
		clicks   []ClickPoint
		success  bool
	}{
		{
			name: "perfect clicks",
			clicks: []ClickPoint{
				{X: 95, Y: 95, Time: 0},
				{X: 195, Y: 95, Time: 500},
				{X: 95, Y: 195, Time: 1000},
				{X: 195, Y: 195, Time: 1500},
			},
			success: true,
		},
		{
			name: "with one false click",
			clicks: []ClickPoint{
				{X: 95, Y: 95, Time: 0},
				{X: 195, Y: 95, Time: 500},
				{X: 50, Y: 50, Time: 750},
				{X: 95, Y: 195, Time: 1000},
				{X: 195, Y: 195, Time: 1500},
			},
			success: true,
		},
		{
			name: "missing target",
			clicks: []ClickPoint{
				{X: 95, Y: 95, Time: 0},
				{X: 195, Y: 95, Time: 500},
				{X: 95, Y: 195, Time: 1000},
			},
			success: false,
		},
		{
			name: "empty clicks",
			clicks: []ClickPoint{},
			success: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := verifier.validateClicks(tt.clicks, board)
			assert.Equal(t, tt.success, result.Success)
		})
	}
}

func TestChineseClickGeneratorService(t *testing.T) {
	generator := NewChineseClickGeneratorService(nil, nil)
	
	req := &CreateChineseClickRequest{
		Width:       400,
		Height:      300,
		TargetCount: 4,
		TotalChars:  12,
		ClientIP:    "127.0.0.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
	}
	
	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotNil(t, resp.Board)
	assert.Equal(t, 4, len(resp.Board.Targets))
	
	targetCharSet := make(map[string]bool)
	for _, tgt := range resp.Board.Targets {
		targetCharSet[tgt.Char] = true
	}
	assert.Equal(t, 4, len(targetCharSet))
	
	assert.NotEmpty(t, resp.ImageURL)
	assert.Contains(t, resp.ImageURL, "data:image/png;base64,")
}

func TestChineseClickVerifierServiceGetSession(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	_, err := verifier.GetSession(context.Background(), "invalid-session")
	assert.Error(t, err)
}

func TestChineseClickVerifierServiceCheckSessionValid(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	valid, msg := verifier.CheckSessionValid(context.Background(), "invalid-session")
	assert.False(t, valid)
	assert.Equal(t, "会话不存在", msg)
}

func TestFindBestTargetMatch(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	targets := []ChineseClickTarget{
		{Char: "你", X: 100, Y: 100, Width: 30, Height: 30, Index: 0},
		{Char: "我", X: 200, Y: 200, Width: 30, Height: 30, Index: 1},
	}
	
	click := ClickPoint{X: 110, Y: 110, Time: 0}
	bestMatch, distance := verifier.findBestTargetMatch(click, targets)
	
	assert.NotNil(t, bestMatch)
	assert.Equal(t, 0, bestMatch.Index)
	assert.Less(t, distance, float64(20))
}

func TestCalculateDistance(t *testing.T) {
	verifier := NewChineseClickVerifierService(nil, nil)
	
	distance := verifier.calculateDistance(0, 0, 3, 4)
	assert.Equal(t, 5.0, distance)
	
	distance = verifier.calculateDistance(10, 10, 10, 10)
	assert.Equal(t, 0.0, distance)
}