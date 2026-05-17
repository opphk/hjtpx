package captcha

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGenerateLianLianKanBoard(t *testing.T) {
	board, err := generateLianLianKanBoard(6, 6, 8)
	assert.NoError(t, err)
	assert.NotNil(t, board)
	assert.Equal(t, 6, board.Width)
	assert.Equal(t, 6, board.Height)
	assert.Equal(t, 18, board.PairCount)
	assert.True(t, board.Shuffled)

	typeCount := make(map[int]int)
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			tile := board.Tiles[y][x]
			typeCount[tile.Type]++
		}
	}

	for _, count := range typeCount {
		assert.True(t, count%2 == 0)
	}
}

func TestLianLianKanGeneratorService(t *testing.T) {
	generator := NewLianLianKanGeneratorService(nil, nil)
	assert.NotNil(t, generator)

	req := &CreateLianLianKanRequest{
		Width:       6,
		Height:      6,
		TileTypes:   8,
		ClientIP:    "127.0.0.1",
		UserAgent:   "test-agent",
		Fingerprint: "test-fingerprint",
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)
	assert.NotNil(t, resp)
	assert.NotEmpty(t, resp.SessionID)
	assert.NotNil(t, resp.Board)
	assert.NotEmpty(t, resp.TileIcons)
}

func TestLianLianKanVerifierService(t *testing.T) {
	generator := NewLianLianKanGeneratorService(nil, nil)
	verifier := NewLianLianKanVerifierService(nil, nil)

	req := &CreateLianLianKanRequest{
		Width:     4,
		Height:    4,
		TileTypes: 4,
	}

	resp, err := generator.Create(context.Background(), req)
	assert.NoError(t, err)

	board := resp.Board
	pairs := make([]LianLianKanPair, 0)

	typeIndices := make(map[int][]int)
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			tile := board.Tiles[y][x]
			typeIndices[tile.Type] = append(typeIndices[tile.Type], tile.Index)
		}
	}

	for _, indices := range typeIndices {
		for i := 0; i < len(indices); i += 2 {
			if i+1 < len(indices) {
				tile1 := findTileByIndex(board, indices[i])
				tile2 := findTileByIndex(board, indices[i+1])
				if tile1 != nil && tile2 != nil {
					pairs = append(pairs, LianLianKanPair{
						Tile1: tile1,
						Tile2: tile2,
					})
				}
			}
		}
	}

	// verifyReq := &VerifyLianLianKanRequest{
	// 	SessionID: resp.SessionID,
	// 	Board:     board,
	// 	Pairs:     pairs,
	// 	RiskScore: 0.0,
	// }

	isValid, score := verifier.validateBoard(board, board, pairs)
	assert.True(t, isValid)
	assert.Equal(t, 100.0, score)
}

func findTileByIndex(board *LianLianKanBoard, index int) *LianLianKanTile {
	for y := 0; y < board.Height; y++ {
		for x := 0; x < board.Width; x++ {
			if board.Tiles[y][x].Index == index {
				return &board.Tiles[y][x]
			}
		}
	}
	return nil
}
