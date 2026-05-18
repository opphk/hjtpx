package captcha

import (
	"context"
	"testing"
)

func TestARGeneratorService_Create(t *testing.T) {
	gen := NewARGeneratorService(nil, nil)
	
	tests := []struct {
		name    string
		req     *CreateARRequest
		wantErr bool
	}{
		{
			name: "default parameters",
			req: &CreateARRequest{
				ClientIP: "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "custom difficulty",
			req: &CreateARRequest{
				Difficulty:  "hard",
				ObjectType:  "cube",
				GestureType: "rotate_y",
				ClientIP:    "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "expert difficulty",
			req: &CreateARRequest{
				Difficulty: "expert",
				ClientIP:   "127.0.0.1",
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
					if result.Puzzle.Object == nil {
						t.Error("Expected 3D object in puzzle")
					}
					if result.Puzzle.GestureType == "" {
						t.Error("Expected gesture type")
					}
					if result.Puzzle.TargetAngle <= 0 {
						t.Error("Expected positive target angle")
					}
					if result.Puzzle.Tolerance <= 0 {
						t.Error("Expected positive tolerance")
					}
					
					t.Logf("AR captcha: session=%s, object=%s, gesture=%s, angle=%.1f",
						result.SessionID, result.Puzzle.Object.Type, 
						result.Puzzle.GestureType, result.Puzzle.TargetAngle)
				}
			}
		})
	}
}

func TestARGeneratorService_GenerateARPuzzle(t *testing.T) {
	gen := NewARGeneratorService(nil, nil)
	
	difficulties := []string{"easy", "medium", "hard", "expert"}
	gestures := []ARGestureType{ARGestureRotateX, ARGestureRotateY, ARGestureRotateZ}
	objectTypes := []string{"cube", "sphere", "pyramid", "cylinder"}
	
	for _, diff := range difficulties {
		for _, gesture := range gestures {
			for _, objType := range objectTypes {
				puzzle := gen.generateARPuzzle(objType, gesture, diff)
				
				if puzzle.Object.Type != objType {
					t.Errorf("Expected object type %s, got %s", objType, puzzle.Object.Type)
				}
				
				if puzzle.GestureType != gesture {
					t.Errorf("Expected gesture type %s, got %s", gesture, puzzle.GestureType)
				}
				
				if puzzle.Difficulty != diff {
					t.Errorf("Expected difficulty %s, got %s", diff, puzzle.Difficulty)
				}
				
				if len(puzzle.Object.Vertices) == 0 {
					t.Error("Expected vertices in object")
				}
				
				if puzzle.TargetAngle <= 0 {
					t.Error("Expected positive target angle")
				}
				
				t.Logf("Generated: difficulty=%s, gesture=%s, angle=%.1f, tolerance=%.1f",
					diff, gesture, puzzle.TargetAngle, puzzle.Tolerance)
			}
		}
	}
}

func TestARVerifierService_Verify(t *testing.T) {
	gen := NewARGeneratorService(nil, nil)
	ver := NewARVerifierService(nil, nil)
	
	ctx := context.Background()
	
	createResult, err := gen.Create(ctx, &CreateARRequest{
		Difficulty:  "medium",
		ObjectType:  "cube",
		GestureType: "rotate_y",
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Failed to create captcha: %v", err)
	}
	
	verifyReq := &VerifyARRequest{
		SessionID:  createResult.SessionID,
		RotationY:  createResult.Puzzle.Object.TargetRotY,
		Scale:      1.0,
		RiskScore:  0.5,
	}
	
	result, err := ver.Verify(ctx, verifyReq)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	t.Logf("AR verification: success=%v, score=%.2f, accuracy=%.2f",
		result.Success, result.Score, result.Accuracy)
}

func TestARVerifierService_CalculateRotationScore(t *testing.T) {
	ver := NewARVerifierService(nil, nil)
	
	puzzle := &ARCaptchaPuzzle{
		Object: &ARObject{
			TargetRotX: 90,
			TargetRotY: 180,
			TargetRotZ: 45,
		},
		GestureType: ARGestureRotateY,
		Tolerance:   10,
	}
	
	tests := []struct {
		name      string
		req       *VerifyARRequest
		minScore  float64
	}{
		{
			name: "exact match",
			req: &VerifyARRequest{
				RotationY: 180,
			},
			minScore: 0.95,
		},
		{
			name: "within tolerance",
			req: &VerifyARRequest{
				RotationY: 185,
			},
			minScore: 0.5,
		},
		{
			name: "outside tolerance",
			req: &VerifyARRequest{
				RotationY: 200,
			},
			minScore: 0,
			maxScore: 0.5,
		},
	}
	
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			score := ver.calculateRotationScore(puzzle, tt.req)
			
			if score < tt.minScore {
				t.Errorf("Expected score >= %f, got %f", tt.minScore, score)
			}
			
			if tt.maxScore > 0 && score > tt.maxScore {
				t.Errorf("Expected score <= %f, got %f", tt.maxScore, score)
			}
		})
	}
}

func TestNormalizeAngle(t *testing.T) {
	tests := []struct {
		input    float64
		expected float64
	}{
		{0, 0},
		{90, 90},
		{180, 180},
		{270, 270},
		{360, 0},
		{450, 90},
		{-90, 270},
		{-180, 180},
		{-360, 0},
	}
	
	for _, tt := range tests {
		result := normalizeAngle(tt.input)
		if result != tt.expected {
			t.Errorf("normalizeAngle(%f) = %f, expected %f", tt.input, result, tt.expected)
		}
	}
}
