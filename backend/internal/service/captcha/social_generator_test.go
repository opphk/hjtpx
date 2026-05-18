package captcha

import (
	"context"
	"testing"
)

func TestSocialGeneratorService_Create(t *testing.T) {
	gen := NewSocialGeneratorService(nil, nil)
	
	tests := []struct {
		name    string
		req     *CreateSocialRequest
		wantErr bool
	}{
		{
			name: "default parameters",
			req: &CreateSocialRequest{
				ClientIP: "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "trace pattern",
			req: &CreateSocialRequest{
				Difficulty:   "medium",
				BehaviorType: "trace_pattern",
				PatternCount: 1,
				ClientIP:     "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "multiple patterns",
			req: &CreateSocialRequest{
				Difficulty:   "hard",
				BehaviorType: "gesture_connect",
				PatternCount: 2,
				ClientIP:     "127.0.0.1",
			},
			wantErr: false,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := gen.Create(context.Background(), tt.req)
			
			if (err != nil) != tt.wantErr {
				t.Errorf("Create() error = %v, wantErr %v", err, tt.wantErr)
				return
			}
			
			if result != nil {
				if result.SessionID == "" {
					t.Error("Expected non-empty session ID")
				}
				if result.Puzzle == nil {
					t.Error("Expected puzzle data")
				}
				if result.Puzzle != nil {
					if len(result.Puzzle.Patterns) == 0 {
						t.Error("Expected at least one pattern")
					}
					if result.Puzzle.BehaviorType == "" {
						t.Error("Expected behavior type")
					}
					if result.Puzzle.Instructions == "" {
						t.Error("Expected instructions")
					}
					if result.Puzzle.TimeLimit <= 0 {
						t.Error("Expected positive time limit")
					}
					
					t.Logf("Social captcha: session=%s, behavior=%s, patterns=%d, timeLimit=%ds",
						result.SessionID, result.Puzzle.BehaviorType, 
						len(result.Puzzle.Patterns), result.Puzzle.TimeLimit)
				}
			}
		})
	}
}

func TestSocialGeneratorService_GenerateTracePattern(t *testing.T) {
	gen := NewSocialGeneratorService(nil, nil)
	
	behaviorTypes := []SocialBehaviorType{
		SocialTypeTracePattern,
		SocialTypeGestureConnect,
		SocialTypeTimingSequence,
	}
	difficulties := []string{"easy", "medium", "hard", "expert"}
	
	for _, bt := range behaviorTypes {
		for _, diff := range difficulties {
			pattern := gen.generateTracePattern(bt, diff)
			
			if pattern.ID == "" {
				t.Error("Expected non-empty pattern ID")
			}
			
			if pattern.Type != string(bt) {
				t.Errorf("Expected type %s, got %s", bt, pattern.Type)
			}
			
			if pattern.TargetShape == "" {
				t.Error("Expected target shape")
			}
			
			if pattern.StartPoint == nil {
				t.Error("Expected start point")
			}
			
			if pattern.EndPoint == nil {
				t.Error("Expected end point")
			}
			
			if len(pattern.TracePoints) == 0 {
				t.Error("Expected trace points")
			}
			
			t.Logf("Pattern: type=%s, shape=%s, points=%d",
				bt, pattern.TargetShape, len(pattern.TracePoints))
		}
	}
}

func TestSocialVerifierService_Verify(t *testing.T) {
	gen := NewSocialGeneratorService(nil, nil)
	ver := NewSocialVerifierService(nil, nil)
	
	ctx := context.Background()
	
	createResult, err := gen.Create(ctx, &CreateSocialRequest{
		Difficulty:   "medium",
		BehaviorType: "trace_pattern",
		ClientIP:     "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Failed to create captcha: %v", err)
	}
	
	if len(createResult.Puzzle.Patterns) == 0 {
		t.Fatal("Expected at least one pattern")
	}
	
	pattern := createResult.Puzzle.Patterns[0]
	
	verifyReq := &VerifySocialRequest{
		SessionID:  createResult.SessionID,
		TraceData:  pattern.TracePoints,
		PatternType: pattern.TargetShape,
		StartTime:  1000,
		EndTime:    5000,
		RiskScore:  0.5,
	}
	
	result, err := ver.Verify(ctx, verifyReq)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	t.Logf("Social verification: success=%v, score=%.2f, similarity=%.2f",
		result.Success, result.Score, result.ShapeSimilarity)
}

func TestSocialVerifierService_CalculateShapeSimilarity(t *testing.T) {
	ver := NewSocialVerifierService(nil, nil)
	
	puzzle := &SocialPuzzle{
		Patterns: []TracePattern{
			{
				TracePoints: []TracePoint{
					{X: 0, Y: 0, Timestamp: 0},
					{X: 100, Y: 50, Timestamp: 100},
					{X: 200, Y: 0, Timestamp: 200},
					{X: 300, Y: 50, Timestamp: 300},
					{X: 400, Y: 0, Timestamp: 400},
				},
			},
		},
		SimilarityThreshold: 0.5,
	}
	
	tests := []struct {
		name         string
		userTrace    []TracePoint
		minSimilarity float64
	}{
		{
			name: "exact match",
			userTrace: []TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 50, Timestamp: 100},
				{X: 200, Y: 0, Timestamp: 200},
				{X: 300, Y: 50, Timestamp: 300},
				{X: 400, Y: 0, Timestamp: 400},
			},
			minSimilarity: 0.9,
		},
		{
			name: "similar trace",
			userTrace: []TracePoint{
				{X: 5, Y: 5, Timestamp: 0},
				{X: 105, Y: 55, Timestamp: 100},
				{X: 205, Y: 5, Timestamp: 200},
				{X: 305, Y: 55, Timestamp: 300},
				{X: 405, Y: 5, Timestamp: 400},
			},
			minSimilarity: 0.7,
		},
		{
			name: "different trace",
			userTrace: []TracePoint{
				{X: 0, Y: 100, Timestamp: 0},
				{X: 100, Y: 150, Timestamp: 100},
				{X: 200, Y: 100, Timestamp: 200},
				{X: 300, Y: 150, Timestamp: 300},
				{X: 400, Y: 100, Timestamp: 400},
			},
			maxSimilarity: 0.5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			similarity := ver.calculateShapeSimilarity(puzzle, tt.userTrace)
			
			if tt.minSimilarity > 0 && similarity < tt.minSimilarity {
				t.Errorf("Expected similarity >= %f, got %f", tt.minSimilarity, similarity)
			}
			
			if tt.maxSimilarity > 0 && similarity > tt.maxSimilarity {
				t.Errorf("Expected similarity <= %f, got %f", tt.maxSimilarity, similarity)
			}
		})
	}
}

func TestSocialVerifierService_AnalyzeSpeed(t *testing.T) {
	ver := NewSocialVerifierService(nil, nil)
	
	tests := []struct {
		name      string
		trace     []TracePoint
		startTime int64
		endTime   int64
		contains  string
	}{
		{
			name: "too fast",
			trace: []TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 100, Y: 0, Timestamp: 10},
				{X: 200, Y: 0, Timestamp: 20},
			},
			startTime: 0,
			endTime:   20,
			contains:  "过快",
		},
		{
			name: "normal speed",
			trace: []TracePoint{
				{X: 0, Y: 0, Timestamp: 0},
				{X: 50, Y: 0, Timestamp: 1000},
				{X: 100, Y: 0, Timestamp: 2000},
			},
			startTime: 0,
			endTime:   2000,
			contains:  "正常",
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := ver.analyzeSpeed(tt.trace, tt.startTime, tt.endTime)
			if result == "" {
				t.Error("Expected non-empty result")
			}
			t.Logf("Speed analysis: %s", result)
		})
	}
}

func TestSocialVerifierService_NormalizeTrace(t *testing.T) {
	ver := NewSocialVerifierService(nil, nil)
	
	trace := []TracePoint{
		{X: 100, Y: 100, Timestamp: 0},
		{X: 200, Y: 150, Timestamp: 100},
		{X: 300, Y: 200, Timestamp: 200},
	}
	
	normalized := ver.normalizeTrace(trace)
	
	if len(normalized) != len(trace) {
		t.Errorf("Expected length %d, got %d", len(trace), len(normalized))
	}
	
	minX, maxX := normalized[0].X, normalized[0].X
	minY, maxY := normalized[0].Y, normalized[0].Y
	
	for _, p := range normalized {
		if p.X < minX {
			minX = p.X
		}
		if p.X > maxX {
			maxX = p.X
		}
		if p.Y < minY {
			minY = p.Y
		}
		if p.Y > maxY {
			maxY = p.Y
		}
	}
	
	if minX < 0 || maxX > 1 || minY < 0 || maxY > 1 {
		t.Errorf("Normalized values out of [0,1] range: minX=%.2f, maxX=%.2f, minY=%.2f, maxY=%.2f",
			minX, maxX, minY, maxY)
	}
}
