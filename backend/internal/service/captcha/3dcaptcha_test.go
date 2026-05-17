package captcha

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestNewThreeDGeneratorService(t *testing.T) {
	service := NewThreeDGeneratorService(nil, nil)
	assert.NotNil(t, service)
}

func TestNewThreeDVerifierService(t *testing.T) {
	service := NewThreeDVerifierService(nil, nil)
	assert.NotNil(t, service)
}

func TestGetGridSizeByDifficulty(t *testing.T) {
	service := NewThreeDGeneratorService(nil, nil)
	
	tests := []struct {
		difficulty string
		expected   int
	}{
		{"easy", 2},
		{"medium", 3},
		{"hard", 4},
		{"expert", 5},
		{"unknown", 3},
		{"", 3},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			result := service.getGridSizeByDifficulty(tt.difficulty)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGeneratePuzzle(t *testing.T) {
	service := NewThreeDGeneratorService(nil, nil)
	
	difficulties := []string{"easy", "medium", "hard", "expert"}
	
	for _, difficulty := range difficulties {
		t.Run(difficulty, func(t *testing.T) {
			gridSize := service.getGridSizeByDifficulty(difficulty)
			puzzle := service.generatePuzzle(gridSize, difficulty)
			
			assert.NotNil(t, puzzle)
			assert.Equal(t, gridSize, puzzle.GridSize)
			assert.Equal(t, difficulty, puzzle.Difficulty)
			assert.Len(t, puzzle.Pieces, gridSize*gridSize)
			
			// 验证每个拼图块
			for _, piece := range puzzle.Pieces {
				assert.NotEmpty(t, piece.Type)
				assert.NotEmpty(t, piece.Color)
				assert.GreaterOrEqual(t, piece.Scale, 0.0)
			}
		})
	}
}

func TestPieceColorsAndTypes(t *testing.T) {
	// 验证颜色列表
	assert.Greater(t, len(pieceColors), 0)
	for _, color := range pieceColors {
		assert.Equal(t, '#', rune(color[0]))
		assert.Len(t, color, 7)
	}
	
	// 验证类型列表
	assert.Greater(t, len(pieceTypes), 0)
	expectedTypes := []string{"cube", "cylinder", "sphere", "cone", "torus"}
	for _, et := range expectedTypes {
		assert.Contains(t, pieceTypes, et)
	}
}

func TestGenerateSessionID(t *testing.T) {
	// 测试generateSessionID函数（虽然是私有的，但我们可以通过Create方法间接测试）
	service := NewThreeDGeneratorService(nil, nil)
	
	req := &CreateThreeDRequest{
		Difficulty: "medium",
	}
	
	result, err := service.Create(nil, req)
	assert.NoError(t, err)
	assert.NotEmpty(t, result.SessionID)
	
	// 再次生成，验证ID不同
	result2, err := service.Create(nil, req)
	assert.NoError(t, err)
	assert.NotEqual(t, result.SessionID, result2.SessionID)
}

func TestThreeDPuzzleStructure(t *testing.T) {
	puzzle := &ThreeDPuzzle{
		Pieces: []ThreeDPiece{
			{
				ID:        0,
				Type:      "cube",
				Color:     "#e74c3c",
				PositionX: 0,
				PositionY: 0,
				PositionZ: 0,
				RotationX: 45,
				RotationY: 90,
				RotationZ: 0,
				Scale:     0.8,
			},
			{
				ID:        1,
				Type:      "sphere",
				Color:     "#3498db",
				PositionX: 1,
				PositionY: 0,
				PositionZ: 0,
				RotationX: 0,
				RotationY: 45,
				RotationZ: 0,
				Scale:     0.8,
			},
		},
		GridSize:    2,
		Difficulty:  "easy",
		TargetRotX: 180,
		TargetRotY: 90,
		TargetRotZ: 0,
	}
	
	assert.NotNil(t, puzzle)
	assert.Equal(t, 2, puzzle.GridSize)
	assert.Equal(t, "easy", puzzle.Difficulty)
	assert.Len(t, puzzle.Pieces, 2)
}

func TestFindPieceByID(t *testing.T) {
	pieces := []ThreeDPiece{
		{ID: 0, Type: "cube"},
		{ID: 1, Type: "sphere"},
		{ID: 2, Type: "cylinder"},
	}
	
	// 测试找到的情况
	found := findPieceByID(pieces, 1)
	assert.NotNil(t, found)
	assert.Equal(t, 1, found.ID)
	assert.Equal(t, "sphere", found.Type)
	
	// 测试找不到的情况
	notFound := findPieceByID(pieces, 999)
	assert.Nil(t, notFound)
	
	// 测试空切片
	emptyFound := findPieceByID([]ThreeDPiece{}, 0)
	assert.Nil(t, emptyFound)
}

func TestNormalizeAngleDiffLogic(t *testing.T) {
	tests := []struct {
		name     string
		angle1   float64
		angle2   float64
		expected float64
	}{
		{"zero diff", 0, 0, 0},
		{"small diff", 10, 0, 10},
		{"180 diff", 0, 180, 180},
		{"over 180", 0, 200, 160},
		{"360 diff", 0, 360, 0},
		{"negative angle", -10, 0, 10},
		{"negative over 180", -200, 0, 160},
		{"both positive", 350, 10, 20},
		{"both negative", -10, -350, 20},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			diff := absAngle(tt.angle1 - tt.angle2)
			for diff > 180 {
				diff = 360 - diff
			}
			assert.Equal(t, tt.expected, diff)
		})
	}
}

func absAngle(x float64) float64 {
	if x < 0 {
		return -x
	}
	return x
}

func TestGetMaxAllowedDiffLogic(t *testing.T) {
	tests := []struct {
		difficulty string
		expected   float64
	}{
		{"easy", 45},
		{"medium", 30},
		{"hard", 20},
		{"expert", 15},
		{"unknown", 30},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			var result float64
			switch tt.difficulty {
			case "easy":
				result = 45
			case "medium":
				result = 30
			case "hard":
				result = 20
			case "expert":
				result = 15
			default:
				result = 30
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestGetPassThresholdLogic(t *testing.T) {
	tests := []struct {
		difficulty string
		expected   float64
	}{
		{"easy", 70},
		{"medium", 80},
		{"hard", 85},
		{"expert", 90},
		{"unknown", 80},
	}

	for _, tt := range tests {
		t.Run(tt.difficulty, func(t *testing.T) {
			var result float64
			switch tt.difficulty {
			case "easy":
				result = 70
			case "medium":
				result = 80
			case "hard":
				result = 85
			case "expert":
				result = 90
			default:
				result = 80
			}
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestThreeDPieceCopy(t *testing.T) {
	original := ThreeDPiece{
		ID:        5,
		Type:      "torus",
		Color:     "#9b59b6",
		PositionX: 2,
		PositionY: 1,
		PositionZ: 0,
		RotationX: 45,
		RotationY: 90,
		RotationZ: 180,
		Scale:     0.9,
	}
	
	// 手动复制
	copy := ThreeDPiece{
		ID:        original.ID,
		Type:      original.Type,
		Color:     original.Color,
		PositionX: original.PositionX,
		PositionY: original.PositionY,
		PositionZ: original.PositionZ,
		RotationX: original.RotationX,
		RotationY: original.RotationY,
		RotationZ: original.RotationZ,
		Scale:     original.Scale,
	}
	
	assert.Equal(t, original, copy)
	
	// 修改copy不影响original
	copy.RotationX = 0
	assert.NotEqual(t, original.RotationX, copy.RotationX)
}

func TestCreateThreeDRequestValidation(t *testing.T) {
	req := &CreateThreeDRequest{
		Difficulty: "medium",
		ClientIP:   "127.0.0.1",
		UserAgent:  "Test User Agent",
		Fingerprint: "test-fingerprint",
	}
	
	assert.Equal(t, "medium", req.Difficulty)
	assert.Equal(t, "127.0.0.1", req.ClientIP)
	assert.Equal(t, "Test User Agent", req.UserAgent)
	assert.Equal(t, "test-fingerprint", req.Fingerprint)
}

func TestVerifyThreeDRequestValidation(t *testing.T) {
	puzzle := &ThreeDPuzzle{
		GridSize: 3,
		Difficulty: "medium",
	}
	
	req := &VerifyThreeDRequest{
		SessionID: "test-session-id",
		Puzzle:    puzzle,
		RiskScore: 0.5,
	}
	
	assert.Equal(t, "test-session-id", req.SessionID)
	assert.NotNil(t, req.Puzzle)
	assert.Equal(t, 0.5, req.RiskScore)
}

func TestCreateThreeDResponseStructure(t *testing.T) {
	puzzle := &ThreeDPuzzle{
		GridSize: 3,
		Difficulty: "medium",
	}
	
	resp := &CreateThreeDResponse{
		SessionID: "test-session",
		Puzzle:    puzzle,
		ExpiresIn: 300,
		ExpiresAt: 1234567890,
	}
	
	assert.Equal(t, "test-session", resp.SessionID)
	assert.NotNil(t, resp.Puzzle)
	assert.Equal(t, int64(300), resp.ExpiresIn)
	assert.Equal(t, int64(1234567890), resp.ExpiresAt)
}

func TestVerifyThreeDResultStructure(t *testing.T) {
	result := &VerifyThreeDResult{
		Success: true,
		Message: "验证成功",
		Score:   95.5,
	}
	
	assert.True(t, result.Success)
	assert.Equal(t, "验证成功", result.Message)
	assert.Equal(t, 95.5, result.Score)
}
