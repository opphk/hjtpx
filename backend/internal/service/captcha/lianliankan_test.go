package captcha

import (
	"context"
	"fmt"
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
	verifier := NewLianLianKanVerifierService(nil, nil)

	board := createTestBoard()
	
	pairs := []LianLianKanPair{
		{Tile1: &board.Tiles[0][0], Tile2: &board.Tiles[0][1]},
		{Tile1: &board.Tiles[0][2], Tile2: &board.Tiles[0][3]},
		{Tile1: &board.Tiles[1][0], Tile2: &board.Tiles[1][1]},
		{Tile1: &board.Tiles[1][2], Tile2: &board.Tiles[1][3]},
		{Tile1: &board.Tiles[2][0], Tile2: &board.Tiles[2][1]},
		{Tile1: &board.Tiles[2][2], Tile2: &board.Tiles[2][3]},
		{Tile1: &board.Tiles[3][0], Tile2: &board.Tiles[3][1]},
		{Tile1: &board.Tiles[3][2], Tile2: &board.Tiles[3][3]},
	}

	isValid, score := verifier.validateBoard(board, board, pairs)
	assert.True(t, isValid)
	assert.Equal(t, 100.0, score)
}

func createTestBoard() *LianLianKanBoard {
	board := &LianLianKanBoard{
		Tiles:      make([][]LianLianKanTile, 4),
		Width:      4,
		Height:     4,
		PairCount:  8,
		Shuffled:   true,
		MaxPathLen: 2,
	}

	index := 0
	for y := 0; y < 4; y++ {
		board.Tiles[y] = make([]LianLianKanTile, 4)
		for x := 0; x < 4; x++ {
			tileType := x / 2
			board.Tiles[y][x] = LianLianKanTile{
				Type:  tileType,
				X:     x,
				Y:     y,
				Index: index,
			}
			index++
		}
	}

	return board
}

func generateValidPairs(board *LianLianKanBoard) []LianLianKanPair {
	pairs := make([]LianLianKanPair, 0)
	visited := make(map[int]bool)
	maxPathLen := board.MaxPathLen
	if maxPathLen == 0 {
		maxPathLen = 3
	}

	tempBoard := &LianLianKanBoard{
		Tiles:  make([][]LianLianKanTile, board.Height),
		Width:  board.Width,
		Height: board.Height,
	}
	for y := 0; y < board.Height; y++ {
		tempBoard.Tiles[y] = make([]LianLianKanTile, board.Width)
		for x := 0; x < board.Width; x++ {
			tempBoard.Tiles[y][x] = board.Tiles[y][x]
		}
	}

	for y1 := 0; y1 < board.Height; y1++ {
		for x1 := 0; x1 < board.Width; x1++ {
			tile1 := board.Tiles[y1][x1]
			if visited[tile1.Index] {
				continue
			}

			for y2 := 0; y2 < board.Height; y2++ {
				for x2 := 0; x2 < board.Width; x2++ {
					tile2 := board.Tiles[y2][x2]
					if visited[tile2.Index] || tile1.Index == tile2.Index {
						continue
					}

					if tile1.Type == tile2.Type && hasValidPath(tempBoard, x1, y1, x2, y2, maxPathLen) {
						pairs = append(pairs, LianLianKanPair{
							Tile1: &board.Tiles[y1][x1],
							Tile2: &board.Tiles[y2][x2],
						})
						visited[tile1.Index] = true
						visited[tile2.Index] = true
						tempBoard.Tiles[y1][x1].Removed = true
						tempBoard.Tiles[y2][x2].Removed = true
						break
					}
				}
				if visited[tile1.Index] {
					break
				}
			}
		}
	}

	return pairs
}

func hasValidPath(board *LianLianKanBoard, x1, y1, x2, y2, maxPathLen int) bool {
	if x1 == x2 && y1 == y2 {
		return false
	}

	visited := make(map[string]bool)
	return findPath(board, x1, y1, x2, y2, 0, maxPathLen, visited, -1, -1)
}

func findPath(board *LianLianKanBoard, x, y, targetX, targetY, pathLen, maxPathLen int, visited map[string]bool, prevX, prevY int) bool {
	key := fmt.Sprintf("%d_%d", x, y)
	if visited[key] {
		return false
	}

	if pathLen > maxPathLen {
		return false
	}

	if x == targetX && y == targetY {
		return true
	}

	visited[key] = true

	directions := []struct{ dx, dy int }{
		{-1, 0}, {1, 0}, {0, -1}, {0, 1},
	}

	for _, dir := range directions {
		newX, newY := x+dir.dx, y+dir.dy

		if newX == prevX && newY == prevY {
			continue
		}

		if newX >= -1 && newX <= board.Width && newY >= -1 && newY <= board.Height {
			isValidPosition := false

			if newX == -1 || newX == board.Width || newY == -1 || newY == board.Height {
				isValidPosition = true
			} else if newX >= 0 && newX < board.Width && newY >= 0 && newY < board.Height {
				isValidPosition = !board.Tiles[newY][newX].Removed
			}

			if isValidPosition {
				if findPath(board, newX, newY, targetX, targetY, pathLen+1, maxPathLen, visited, x, y) {
					return true
				}
			}
		}
	}

	delete(visited, key)
	return false
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
