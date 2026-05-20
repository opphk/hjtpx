package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math"
	"time"

	github.com/hjtpx/hjtpx/internal/repository/cache"
	github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type LianLianKanPair struct {
	Tile1 *LianLianKanTile `json:"tile1"`
	Tile2 *LianLianKanTile `json:"tile2"`
}

type LianLianKanPath struct {
	Points []struct {
		X int `json:"x"`
		Y int `json:"y"`
	} `json:"points"`
}

type VerifyLianLianKanRequest struct {
	SessionID string            `json:"session_id" binding:"required"`
	Board     *LianLianKanBoard `json:"board" binding:"required"`
	Pairs     []LianLianKanPair `json:"pairs" binding:"required"`
	RiskScore float64           `json:"risk_score"`
}

type VerifyLianLianKanResult struct {
	Success bool    `json:"success"`
	Message string  `json:"message"`
	Score   float64 `json:"score"`
}

type LianLianKanVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

func NewLianLianKanVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *LianLianKanVerifierService {
	return &LianLianKanVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (v *LianLianKanVerifierService) Verify(ctx context.Context, req *VerifyLianLianKanRequest) (*VerifyLianLianKanResult, error) {
	session, err := v.getSession(req.SessionID)
	if err != nil {
		return nil, err
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyLianLianKanResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyLianLianKanResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	v.incrementVerifyCount(req.SessionID)

	if session.Status == "verified" {
		return &VerifyLianLianKanResult{
			Success: true,
			Message: "验证码已验证通过",
			Score:   100,
		}, nil
	}

	originalBoard := &LianLianKanBoard{}
	if err := json.Unmarshal([]byte(session.BackgroundURL), originalBoard); err != nil {
		return nil, fmt.Errorf("failed to unmarshal board: %w", err)
	}

	isValid, score := v.validateBoard(req.Board, originalBoard, req.Pairs)

	if isValid {
		v.markAsVerified(req.SessionID, req.RiskScore, 0, 0)
		return &VerifyLianLianKanResult{
			Success: true,
			Message: "验证成功",
			Score:   score,
		}, nil
	}

	return &VerifyLianLianKanResult{
		Success: false,
		Message: "验证失败",
		Score:   score,
	}, nil
}

func (v *LianLianKanVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if v.sessionCache != nil {
		session, err := v.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if v.captchaRepo != nil {
		session, err := v.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (v *LianLianKanVerifierService) incrementVerifyCount(sessionID string) {
	if v.sessionCache != nil {
		_ = v.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (v *LianLianKanVerifierService) markAsVerified(sessionID string, riskScore, traceScore, envScore float64) {
	if v.sessionCache != nil {
		_ = v.sessionCache.UpdateStatus(context.Background(), sessionID, "verified")
	}

	if v.captchaRepo != nil {
		_ = v.captchaRepo.UpdateStatus(sessionID, "verified")
		_ = v.captchaRepo.UpdateRiskScore(sessionID, riskScore, traceScore, envScore)
	}
}

func (v *LianLianKanVerifierService) validateBoard(userBoard, originalBoard *LianLianKanBoard, pairs []LianLianKanPair) (bool, float64) {
	if userBoard == nil || originalBoard == nil {
		return false, 0
	}

	if userBoard.Width != originalBoard.Width || userBoard.Height != originalBoard.Height {
		return false, 0
	}

	totalPairs := originalBoard.PairCount
	matchedPairs := 0
	validPairs := 0

	visited := make(map[int]bool)

	maxPathLen := originalBoard.MaxPathLen
	if maxPathLen == 0 {
		maxPathLen = 3
	}

	tempBoard := v.createTempBoard(userBoard)

	for _, pair := range pairs {
		if pair.Tile1 == nil || pair.Tile2 == nil {
			continue
		}

		tile1 := pair.Tile1
		tile2 := pair.Tile2

		if visited[tile1.Index] || visited[tile2.Index] {
			continue
		}

		if tile1.Index == tile2.Index {
			continue
		}

		originalTile1 := v.getTileAt(originalBoard, tile1.X, tile1.Y)
		originalTile2 := v.getTileAt(originalBoard, tile2.X, tile2.Y)

		if originalTile1 == nil || originalTile2 == nil {
			continue
		}

		isValidPair := originalTile1.Type == originalTile2.Type
		
		if isValidPair {
			isValidPath := v.hasValidPath(tempBoard, tile1.X, tile1.Y, tile2.X, tile2.Y, maxPathLen)

			if isValidPath {
				visited[tile1.Index] = true
				visited[tile2.Index] = true
				matchedPairs++
				validPairs++
				
				v.markTileRemoved(tempBoard, tile1.X, tile1.Y)
				v.markTileRemoved(tempBoard, tile2.X, tile2.Y)
			} else {
				validPairs++
			}
		}
	}

	baseScore := float64(matchedPairs) / float64(totalPairs) * 100
	pathPenalty := float64(validPairs-matchedPairs) * 5
	finalScore := math.Max(0, baseScore-pathPenalty)

	return matchedPairs == totalPairs, finalScore
}

func (v *LianLianKanVerifierService) createTempBoard(original *LianLianKanBoard) *LianLianKanBoard {
	tiles := make([][]LianLianKanTile, original.Height)
	for y := 0; y < original.Height; y++ {
		tiles[y] = make([]LianLianKanTile, original.Width)
		for x := 0; x < original.Width; x++ {
			tiles[y][x] = original.Tiles[y][x]
		}
	}
	return &LianLianKanBoard{
		Tiles:    tiles,
		Width:    original.Width,
		Height:   original.Height,
	}
}

func (v *LianLianKanVerifierService) markTileRemoved(board *LianLianKanBoard, x, y int) {
	if x >= 0 && x < board.Width && y >= 0 && y < board.Height {
		board.Tiles[y][x].Removed = true
	}
}

func (v *LianLianKanVerifierService) getTileAt(board *LianLianKanBoard, x, y int) *LianLianKanTile {
	if x < 0 || x >= board.Width || y < 0 || y >= board.Height {
		return nil
	}
	return &board.Tiles[y][x]
}

func (v *LianLianKanVerifierService) hasValidPath(board *LianLianKanBoard, x1, y1, x2, y2 int, maxPathLen int) bool {
	if x1 == x2 && y1 == y2 {
		return false
	}

	visited := make(map[string]bool)
	return v.findPath(board, x1, y1, x2, y2, 0, maxPathLen, visited, -1, -1)
}

func (v *LianLianKanVerifierService) findPath(board *LianLianKanBoard, x, y, targetX, targetY, pathLen, maxPathLen int, visited map[string]bool, prevX, prevY int) bool {
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
				if v.findPath(board, newX, newY, targetX, targetY, pathLen+1, maxPathLen, visited, x, y) {
					return true
				}
			}
		}
	}

	delete(visited, key)
	return false
}

func (v *LianLianKanVerifierService) CalculatePathComplexity(x1, y1, x2, y2 int) int {
	dx := absInt(x1 - x2)
	dy := absInt(y1 - y2)
	return dx + dy
}

func absInt(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func (v *LianLianKanVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return v.getSession(sessionID)
}

func (v *LianLianKanVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := v.getSession(sessionID)
	if err != nil {
		return false, "会话不存在"
	}

	if time.Now().After(session.ExpiredAt) {
		return false, "验证码已过期"
	}

	if session.Status == "verified" {
		return false, "验证码已验证通过"
	}

	if session.VerifyCount >= session.MaxAttempts {
		return false, "验证次数已用完"
	}

	return true, ""
}
