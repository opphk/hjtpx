package captcha

import (
	"context"
	"testing"
)

func TestHapticGeneratorService_Generate(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	req := &HapticCaptchaRequest{
		PatternType: HapticPatternSequence,
		Difficulty: HapticDifficultyMedium,
		GridSize:   3,
		ClientIP:   "127.0.0.1",
		UserAgent:  "test-agent",
	}

	ctx := context.Background()
	result, err := gen.Generate(ctx, req)

	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.SessionID == "" {
		t.Error("SessionID is empty")
	}

	if result.Pattern == nil {
		t.Fatal("Pattern is nil")
	}

	if result.Pattern.Type != HapticPatternSequence {
		t.Errorf("Expected pattern type %s, got %s",
			HapticPatternSequence, result.Pattern.Type)
	}

	if result.Pattern.GridSize != 3 {
		t.Errorf("Expected grid size 3, got %d", result.Pattern.GridSize)
	}

	if len(result.Pattern.TargetSequence) == 0 {
		t.Error("TargetSequence is empty")
	}

	if len(result.Pattern.Taps) == 0 {
		t.Error("Taps is empty")
	}

	if result.ExpiresIn == 0 {
		t.Error("ExpiresIn is 0")
	}

	if result.Instructions == "" {
		t.Error("Instructions is empty")
	}
}

func TestHapticGeneratorService_GenerateWithDifferentDifficulties(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	testCases := []struct {
		difficulty string
		expectedLen int
	}{
		{HapticDifficultyEasy, 3},
		{HapticDifficultyMedium, 4},
		{HapticDifficultyHard, 6},
	}

	for _, tc := range testCases {
		t.Run(tc.difficulty, func(t *testing.T) {
			req := &HapticCaptchaRequest{
				Difficulty: tc.difficulty,
				GridSize:   3,
			}

			result, err := gen.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if len(result.Pattern.TargetSequence) != tc.expectedLen {
				t.Errorf("Difficulty %s: expected %d sequence length, got %d",
					tc.difficulty, tc.expectedLen, len(result.Pattern.TargetSequence))
			}
		})
	}
}

func TestHapticGeneratorService_GenerateWithDifferentGridSizes(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	testCases := []int{3, 4, 5, 6}

	for _, gridSize := range testCases {
		t.Run("", func(t *testing.T) {
			req := &HapticCaptchaRequest{
				GridSize: gridSize,
			}

			result, err := gen.Generate(context.Background(), req)
			if err != nil {
				t.Fatalf("Generate failed: %v", err)
			}

			if result.Pattern.GridSize != gridSize {
				t.Errorf("Expected grid size %d, got %d",
					gridSize, result.Pattern.GridSize)
			}
		})
	}
}

func TestHapticVerifierService_Verify(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)
	ver := NewHapticVerifierService(nil, nil)

	session, err := gen.Generate(context.Background(), &HapticCaptchaRequest{
		GridSize: 3,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	userInput := &HapticUserInput{
		Sequence:   session.Pattern.TargetSequence,
		Timestamps: []int64{0, 200, 400, 600},
		Pressures:  []float64{0.5, 0.6, 0.7, 0.8},
	}

	verifyReq := &HapticVerifyRequest{
		SessionID: session.SessionID,
		UserInput: userInput,
	}

	result, err := ver.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.MatchScore < 0 || result.MatchScore > 1 {
		t.Errorf("MatchScore out of range: %f", result.MatchScore)
	}

	if result.MatchLevel != "" {
		t.Logf("Verification result: score=%f, level=%s", result.MatchScore, result.MatchLevel)
	}
}

func TestHapticVerifierService_VerifyWrongSequence(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)
	ver := NewHapticVerifierService(nil, nil)

	session, err := gen.Generate(context.Background(), &HapticCaptchaRequest{
		GridSize: 3,
	})
	if err != nil {
		t.Fatalf("Generate failed: %v", err)
	}

	wrongSequence := make([]int, len(session.Pattern.TargetSequence))
	for i := range wrongSequence {
		wrongSequence[i] = (session.Pattern.TargetSequence[i] + 1) % 9
	}

	userInput := &HapticUserInput{
		Sequence:   wrongSequence,
		Timestamps: []int64{0, 200, 400, 600},
		Pressures:  []float64{0.5, 0.6, 0.7, 0.8},
	}

	verifyReq := &HapticVerifyRequest{
		SessionID: session.SessionID,
		UserInput: userInput,
	}

	result, err := ver.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.Success {
		t.Error("Expected verification to fail for wrong sequence")
	}
}

func TestHapticVerifierService_CalculateSequenceScore(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	testCases := []struct {
		name            string
		userSequence    []int
		targetSequence  []int
		expectedScore   float64
	}{
		{
			name:           "identical sequences",
			userSequence:   []int{0, 1, 2, 3},
			targetSequence: []int{0, 1, 2, 3},
			expectedScore:  1.0,
		},
		{
			name:           "completely different sequences",
			userSequence:   []int{3, 2, 1, 0},
			targetSequence: []int{0, 1, 2, 3},
			expectedScore:  0.0,
		},
		{
			name:           "half correct",
			userSequence:   []int{0, 1, 3, 2},
			targetSequence: []int{0, 1, 2, 3},
			expectedScore:  0.5,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ver.calculateSequenceScore(tc.userSequence, tc.targetSequence)
			if result < tc.expectedScore-0.01 || result > tc.expectedScore+0.01 {
				t.Errorf("calculateSequenceScore() = %f, expected %f",
					result, tc.expectedScore)
			}
		})
	}
}

func TestHapticVerifierService_GetMatchLevel(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	testCases := []struct {
		score     float64
		expected  string
	}{
		{0.95, HapticMatchLevelHigh},
		{0.9, HapticMatchLevelHigh},
		{0.8, HapticMatchLevelMedium},
		{0.75, HapticMatchLevelMedium},
		{0.7, HapticMatchLevelLow},
		{0.5, HapticMatchLevelLow},
	}

	for _, tc := range testCases {
		result := ver.getMatchLevel(tc.score)
		if result != tc.expected {
			t.Errorf("getMatchLevel(%f) = %s, expected %s",
				tc.score, result, tc.expected)
		}
	}
}

func TestHapticVerifierService_ValidateInput(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	testCases := []struct {
		name        string
		input       *HapticUserInput
		shouldPass  bool
	}{
		{
			name:       "valid input",
			input:      &HapticUserInput{Sequence: []int{0, 1, 2}},
			shouldPass: true,
		},
		{
			name:        "nil input",
			input:       nil,
			shouldPass:  false,
		},
		{
			name:        "empty sequence",
			input:       &HapticUserInput{Sequence: []int{}},
			shouldPass:  false,
		},
		{
			name:        "sequence too long",
			input:       &HapticUserInput{Sequence: make([]int, 25)},
			shouldPass:  false,
		},
		{
			name:        "invalid position",
			input:       &HapticUserInput{Sequence: []int{-1}},
			shouldPass:  false,
		},
		{
			name:        "position out of range",
			input:       &HapticUserInput{Sequence: []int{40}},
			shouldPass:  false,
		},
		{
			name:        "pressure out of range",
			input:       &HapticUserInput{Sequence: []int{0}, Pressures: []float64{1.5}},
			shouldPass:  false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			valid, _ := ver.ValidateInput(tc.input)
			if valid != tc.shouldPass {
				t.Errorf("ValidateInput() = %v, expected %v", valid, tc.shouldPass)
			}
		})
	}
}

func TestHapticGeneratorService_GetSequenceLength(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	testCases := []struct {
		difficulty   string
		expectedLen int
	}{
		{HapticDifficultyEasy, 3},
		{HapticDifficultyMedium, 4},
		{HapticDifficultyHard, 6},
	}

	for _, tc := range testCases {
		result := gen.getSequenceLength(tc.difficulty)
		if result != tc.expectedLen {
			t.Errorf("getSequenceLength(%s) = %d, expected %d",
				tc.difficulty, result, tc.expectedLen)
		}
	}
}

func TestHapticGeneratorService_GenerateTaps(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	sequence := []int{0, 1, 2, 3}

	testCases := []struct {
		difficulty   string
		expectedMinDuration float64
		expectedMaxDuration float64
	}{
		{HapticDifficultyEasy, 200.0, 230.0},
		{HapticDifficultyMedium, 150.0, 200.0},
		{HapticDifficultyHard, 100.0, 180.0},
	}

	for _, tc := range testCases {
		t.Run(tc.difficulty, func(t *testing.T) {
			taps := gen.generateTaps(sequence, tc.difficulty)

			if len(taps) != len(sequence) {
				t.Errorf("Expected %d taps, got %d", len(sequence), len(taps))
			}

			for i, tap := range taps {
				if tap.Position != sequence[i] {
					t.Errorf("Tap %d: expected position %d, got %d",
						i, sequence[i], tap.Position)
				}

				if tap.Duration < tc.expectedMinDuration || tap.Duration > tc.expectedMaxDuration {
					t.Errorf("Tap %d: duration %f out of range [%f, %f]",
						i, tap.Duration, tc.expectedMinDuration, tc.expectedMaxDuration)
				}

				if tap.Pressure < 0.5 || tap.Pressure > 1.0 {
					t.Errorf("Tap %d: pressure %f out of range [0.5, 1.0]",
						i, tap.Pressure)
				}
			}
		})
	}
}

func TestHapticGeneratorService_VisualHint(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	sequence := []int{0, 4, 8}
	gridSize := 3

	hint := gen.generateVisualHint(gridSize, sequence)

	if hint == nil {
		t.Fatal("VisualHint is nil")
	}

	if hint.GridSize != gridSize {
		t.Errorf("Expected grid size %d, got %d", gridSize, hint.GridSize)
	}

	if len(hint.Positions) != len(sequence) {
		t.Errorf("Expected %d positions, got %d", len(sequence), len(hint.Positions))
	}

	expectedPositions := []Point{
		{X: 0, Y: 0},
		{X: 1, Y: 1},
		{X: 2, Y: 2},
	}

	for i, pos := range hint.Positions {
		if pos.X != expectedPositions[i].X || pos.Y != expectedPositions[i].Y {
			t.Errorf("Position %d: expected (%d, %d), got (%d, %d)",
				i, expectedPositions[i].X, expectedPositions[i].Y, pos.X, pos.Y)
		}
	}
}

func TestHapticVerifierService_AnalyzeHapticPattern(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	userInput := &HapticUserInput{
		Sequence:   []int{0, 1, 2, 3},
		Timestamps:  []int64{0, 200, 400, 600},
		Pressures:   []float64{0.5, 0.6, 0.7, 0.8},
	}

	analysis := ver.AnalyzeHapticPattern(userInput)

	if analysis == nil {
		t.Fatal("Analysis is nil")
	}

	if analysis["sequence_length"] != 4 {
		t.Errorf("Expected sequence_length 4, got %v", analysis["sequence_length"])
	}

	if analysis["unique_positions"] != 4 {
		t.Errorf("Expected unique_positions 4, got %v", analysis["unique_positions"])
	}

	if analysis["average_interval_ms"] == nil {
		t.Error("Expected average_interval_ms to be set")
	}
}

func TestHapticVerifierService_CountUnique(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	testCases := []struct {
		name     string
		input    []int
		expected int
	}{
		{"all unique", []int{0, 1, 2, 3}, 4},
		{"all same", []int{0, 0, 0, 0}, 1},
		{"mixed", []int{0, 1, 0, 2}, 3},
		{"empty", []int{}, 0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			result := ver.countUnique(tc.input)
			if result != tc.expected {
				t.Errorf("countUnique() = %d, expected %d", result, tc.expected)
			}
		})
	}
}

func TestHapticVerifierService_ExpiredSession(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	verifyReq := &HapticVerifyRequest{
		SessionID: "non-existent-session",
		UserInput: &HapticUserInput{Sequence: []int{0, 1, 2}},
	}

	result, err := ver.Verify(context.Background(), verifyReq)
	if err != nil {
		t.Fatalf("Verify failed: %v", err)
	}

	if result.Success {
		t.Error("Expected verification to fail for non-existent session")
	}

	if result.Message != "Session not found" {
		t.Errorf("Expected 'Session not found', got '%s'", result.Message)
	}
}

func TestHapticGeneratorService_GenerateDemo(t *testing.T) {
	gen := NewHapticGeneratorService(nil, nil)

	req := &HapticCaptchaRequest{
		Difficulty: HapticDifficultyHard,
		GridSize:   5,
	}

	ctx := context.Background()
	result, err := gen.GenerateDemo(ctx, req)

	if err != nil {
		t.Fatalf("GenerateDemo failed: %v", err)
	}

	if result == nil {
		t.Fatal("Result is nil")
	}

	if result.Pattern.GridSize != 3 {
		t.Errorf("Expected demo grid size 3, got %d", result.Pattern.GridSize)
	}
}

func TestHapticVerifierService_CalculateTimingScore(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	pattern := &HapticPattern{
		Taps: []HapticTapConfig{
			{Duration: 150.0},
			{Duration: 150.0},
			{Duration: 150.0},
			{Duration: 150.0},
		},
	}

	testCases := []struct {
		name        string
		timestamps  []int64
		expectedMin float64
	}{
		{"perfect timing", []int64{0, 150, 300, 450}, 0.9},
		{"acceptable timing", []int64{0, 160, 320, 480}, 0.7},
		{"bad timing", []int64{0, 300, 600, 900}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userInput := &HapticUserInput{
				Timestamps: tc.timestamps,
			}

			result := ver.calculateTimingScore(userInput, pattern)
			if result < tc.expectedMin {
				t.Errorf("calculateTimingScore() = %f, expected >= %f",
					result, tc.expectedMin)
			}
		})
	}
}

func TestHapticVerifierService_CalculatePressureScore(t *testing.T) {
	ver := NewHapticVerifierService(nil, nil)

	pattern := &HapticPattern{
		Taps: []HapticTapConfig{
			{Pressure: 0.5},
			{Pressure: 0.6},
			{Pressure: 0.7},
			{Pressure: 0.8},
		},
	}

	testCases := []struct {
		name       string
		pressures  []float64
		expectedMin float64
	}{
		{"perfect pressure", []float64{0.5, 0.6, 0.7, 0.8}, 0.9},
		{"close pressure", []float64{0.55, 0.65, 0.75, 0.85}, 0.7},
		{"wrong pressure", []float64{0.1, 0.2, 0.3, 0.4}, 0.0},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			userInput := &HapticUserInput{
				Pressures: tc.pressures,
			}

			result := ver.calculatePressureScore(userInput, pattern)
			if result < tc.expectedMin {
				t.Errorf("calculatePressureScore() = %f, expected >= %f",
					result, tc.expectedMin)
			}
		})
	}
}
