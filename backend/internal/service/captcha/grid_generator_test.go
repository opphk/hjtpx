package captcha

import (
	"context"
	"testing"
)

func TestGridGeneratorService_Create(t *testing.T) {
	gen := NewGridGeneratorService(nil, nil)
	
	tests := []struct {
		name    string
		req     *CreateGridRequest
		wantErr bool
	}{
		{
			name: "default parameters",
			req: &CreateGridRequest{
				ClientIP: "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "custom grid size",
			req: &CreateGridRequest{
				GridSize:    4,
				TargetCount: 4,
				Difficulty:  "hard",
				ClientIP:    "127.0.0.1",
			},
			wantErr: false,
		},
		{
			name: "invalid grid size",
			req: &CreateGridRequest{
				GridSize: 1,
				ClientIP: "127.0.0.1",
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
				if result.Puzzle != nil && result.Puzzle.GridSize <= 0 {
					t.Error("Expected positive grid size")
				}
				if result.Puzzle != nil && result.Puzzle.RequiredCount <= 0 {
					t.Error("Expected positive required count")
				}
				
				t.Logf("Grid captcha: session=%s, gridSize=%d, required=%d", 
					result.SessionID, result.Puzzle.GridSize, result.Puzzle.RequiredCount)
			}
		})
	}
}

func TestGridGeneratorService_GenerateGridPuzzle(t *testing.T) {
	gen := NewGridGeneratorService(nil, nil)
	
	tests := []struct {
		gridSize    int
		targetCount int
		difficulty  string
		iconType    ImageCategory
	}{
		{3, 3, "easy", CategoryAnimal},
		{3, 3, "medium", CategoryFood},
		{4, 4, "hard", CategoryVehicle},
		{5, 4, "expert", CategoryNature},
	}
	
	for _, tt := range tests {
		puzzle := gen.generateGridPuzzle(tt.gridSize, tt.targetCount, tt.difficulty, tt.iconType)
		
		if puzzle.GridSize != tt.gridSize {
			t.Errorf("Expected gridSize %d, got %d", tt.gridSize, puzzle.GridSize)
		}
		
		expectedCells := tt.gridSize * tt.gridSize
		if len(puzzle.Cells) != expectedCells {
			t.Errorf("Expected %d cells, got %d", expectedCells, len(puzzle.Cells))
		}
		
		targetCount := 0
		for _, cell := range puzzle.Cells {
			if cell.IsTarget {
				targetCount++
			}
		}
		
		if targetCount != puzzle.RequiredCount {
			t.Errorf("Target count mismatch: declared %d, actual %d", 
				puzzle.RequiredCount, targetCount)
		}
		
		if len(puzzle.TargetOrder) != puzzle.RequiredCount {
			t.Errorf("Target order length mismatch")
		}
		
		t.Logf("Puzzle: gridSize=%d, targets=%d, order=%v", 
			puzzle.GridSize, puzzle.RequiredCount, puzzle.TargetOrder)
	}
}

func TestGridVerifierService_Verify(t *testing.T) {
	gen := NewGridGeneratorService(nil, nil)
	ver := NewGridVerifierService(nil, nil)
	
	ctx := context.Background()
	
	createResult, err := gen.Create(ctx, &CreateGridRequest{
		GridSize:    3,
		TargetCount: 2,
		Difficulty:  "easy",
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Failed to create captcha: %v", err)
	}
	
	expectedOrder := make([]int, len(createResult.Puzzle.TargetIndices))
	for i, idx := range createResult.Puzzle.TargetIndices {
		for j, cell := range createResult.Puzzle.Cells {
			if cell.Index == idx && cell.IsTarget {
				expectedOrder[i] = j
				break
			}
		}
	}
	
	verifyReq := &VerifyGridRequest{
		SessionID:     createResult.SessionID,
		SelectedOrder: expectedOrder,
		TimeSpent:     5000,
		RiskScore:     0.5,
	}
	
	result, err := ver.Verify(ctx, verifyReq)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result == nil {
		t.Fatal("Expected result, got nil")
	}
	
	if !result.Success {
		t.Logf("Verification message: %s", result.Message)
	}
	
	t.Logf("Grid verification: success=%v, score=%.2f, correct=%d/%d", 
		result.Success, result.Score, result.CorrectCount, result.TotalRequired)
}

func TestGridVerifierService_VerifyInvalidOrder(t *testing.T) {
	gen := NewGridGeneratorService(nil, nil)
	ver := NewGridVerifierService(nil, nil)
	
	ctx := context.Background()
	
	createResult, err := gen.Create(ctx, &CreateGridRequest{
		GridSize:    3,
		TargetCount: 2,
		Difficulty:  "medium",
		ClientIP:    "127.0.0.1",
	})
	if err != nil {
		t.Fatalf("Failed to create captcha: %v", err)
	}
	
	wrongOrder := []int{0, 1}
	
	verifyReq := &VerifyGridRequest{
		SessionID:     createResult.SessionID,
		SelectedOrder: wrongOrder,
		TimeSpent:     1000,
		RiskScore:     0.5,
	}
	
	result, err := ver.Verify(ctx, verifyReq)
	if err != nil {
		t.Fatalf("Expected no error, got: %v", err)
	}
	
	if result.Success {
		t.Error("Expected verification to fail for wrong order")
	}
	
	if result.CorrectCount >= result.TotalRequired {
		t.Errorf("Expected fewer correct than total, got correct=%d, total=%d", 
			result.CorrectCount, result.TotalRequired)
	}
}

func TestGridVerifierService_AnalyzeClickTime(t *testing.T) {
	ver := NewGridVerifierService(nil, nil)
	
	tests := []struct {
		timeSpent  int64
		clickCount int
		difficulty string
		expected   string
	}{
		{300, 3, "easy", "过快，可能为机器操作"},
		{10000, 3, "easy", "时间正常"},
		{30000, 3, "hard", "较慢但正常"},
	}
	
	for _, tt := range tests {
		result := ver.analyzeClickTime(tt.timeSpent, tt.clickCount, tt.difficulty)
		if result != tt.expected {
			t.Errorf("timeSpent=%d, difficulty=%s: expected '%s', got '%s'", 
				tt.timeSpent, tt.difficulty, tt.expected, result)
		}
	}
}

func TestGridVerifierService_GetSessionStatus(t *testing.T) {
	ver := NewGridVerifierService(nil, nil)
	
	valid, message := ver.CheckSessionValid(context.Background(), "non-existent")
	if valid {
		t.Error("Expected invalid for non-existent session")
	}
	if message == "" {
		t.Error("Expected error message")
	}
}
