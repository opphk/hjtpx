package captcha

import (
	"context"
	"encoding/json"
	"fmt"
	"math/rand"
	"time"

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type GridIconType string

const (
	GridIconAnimal   GridIconType = "animal"
	GridIconFood     GridIconType = "food"
	GridIconVehicle  GridIconType = "vehicle"
	GridIconFruit    GridIconType = "fruit"
	GridIconObject   GridIconType = "object"
)

type GridCell struct {
	Row      int          `json:"row"`
	Col      int          `json:"col"`
	IconType GridIconType `json:"icon_type"`
	IconID   int          `json:"icon_id"`
	Color    string       `json:"color"`
	IsTarget bool         `json:"is_target"`
	Index    int          `json:"index"`
}

type GridPuzzle struct {
	Cells         []GridCell `json:"cells"`
	GridSize      int        `json:"grid_size"`
	TargetIndices []int      `json:"target_indices"`
	TargetOrder   []int      `json:"target_order"`
	RequiredCount int        `json:"required_count"`
	Difficulty    string     `json:"difficulty"`
}

type CreateGridRequest struct {
	GridSize   int    `json:"grid_size"`
	TargetCount int   `json:"target_count"`
	Difficulty  string `json:"difficulty"`
	IconType    string `json:"icon_type"`
	ClientIP    string `json:"client_ip"`
	UserAgent   string `json:"user_agent"`
	Fingerprint string `json:"fingerprint"`
}

type CreateGridResponse struct {
	SessionID   string     `json:"session_id"`
	Puzzle      *GridPuzzle `json:"puzzle"`
	HintText    string     `json:"hint_text"`
	ImageDataURL string    `json:"image_data_url"`
	ExpiresIn   int64      `json:"expires_in"`
	ExpiresAt   int64      `json:"expires_at"`
}

type VerifyGridRequest struct {
	SessionID     string `json:"session_id" binding:"required"`
	SelectedOrder []int  `json:"selected_order" binding:"required"`
	TimeSpent     int64  `json:"time_spent"`
	ClickPattern  []ClickPoint `json:"click_pattern"`
	RiskScore     float64      `json:"risk_score"`
}

type ClickPoint struct {
	Row       int   `json:"row"`
	Col       int   `json:"col"`
	Timestamp int64 `json:"timestamp"`
}

type VerifyGridResult struct {
	Success          bool    `json:"success"`
	Message          string  `json:"message"`
	Score            float64 `json:"score"`
	CorrectCount     int     `json:"correct_count"`
	TotalRequired    int     `json:"total_required"`
	TimeAnalysis     string  `json:"time_analysis"`
	ClickPatternScore float64 `json:"click_pattern_score"`
}

type GridGeneratorService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

type GridVerifierService struct {
	sessionCache *cache.SessionCache
	captchaRepo  *db.CaptchaRepository
}

var gridIconColors = map[GridIconType][]string{
	GridIconAnimal:  {"#FF6B6B", "#4ECDC4", "#45B7D1", "#96CEB4", "#FFEAA7", "#DDA0DD", "#98D8C8"},
	GridIconFood:    {"#FF9F43", "#EE5A24", "#FFC312", "#7D6608", "#E74C3C", "#F39C12", "#27AE60"},
	GridIconVehicle: {"#3498DB", "#2980B9", "#8E44AD", "#9B59B6", "#1ABC9C", "#16A085", "#34495E"},
	GridIconFruit:   {"#E91E63", "#FF5722", "#FF9800", "#CDDC39", "#4CAF50", "#8BC34A", "#FFEB3B"},
	GridIconObject:  {"#607D8B", "#795548", "#9E9E9E", "#3F51B5", "#673AB7", "#00BCD4", "#009688"},
}

var iconNames = map[GridIconType][]string{
	GridIconAnimal:  {"cat", "dog", "bird", "fish", "rabbit", "bear", "lion", "tiger", "elephant", "monkey"},
	GridIconFood:    {"apple", "banana", "pizza", "cake", "coffee", "icecream", "bread", "rice", "sushi", "noodle"},
	GridIconVehicle: {"car", "bus", "train", "plane", "ship", "bike", "truck", "taxi", "subway", "boat"},
	GridIconFruit:   {"apple", "orange", "grape", "strawberry", "watermelon", "mango", "peach", "pear", "cherry", "lemon"},
	GridIconObject:  {"book", "phone", "cup", "chair", "lamp", "clock", "camera", "bag", "key", "hat"},
}

func NewGridGeneratorService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *GridGeneratorService {
	return &GridGeneratorService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func NewGridVerifierService(sessionCache *cache.SessionCache, captchaRepo *db.CaptchaRepository) *GridVerifierService {
	return &GridVerifierService{
		sessionCache: sessionCache,
		captchaRepo:  captchaRepo,
	}
}

func (s *GridGeneratorService) Create(ctx context.Context, req *CreateGridRequest) (*CreateGridResponse, error) {
	gridSize := req.GridSize
	if gridSize <= 0 {
		gridSize = 3
	}
	if gridSize < 2 {
		gridSize = 2
	}
	if gridSize > 5 {
		gridSize = 5
	}

	targetCount := req.TargetCount
	if targetCount <= 0 {
		targetCount = 3
	}
	if targetCount >= gridSize*gridSize {
		targetCount = gridSize*gridSize - 1
	}

	difficulty := req.Difficulty
	if difficulty == "" {
		difficulty = "medium"
	}

	iconType := GridIconType(req.IconType)
	if iconType == "" {
		iconTypes := []GridIconType{GridIconAnimal, GridIconFood, GridIconFruit, GridIconObject}
		iconType = iconTypes[rand.Intn(len(iconTypes))]
	}

	puzzle := s.generateGridPuzzle(gridSize, targetCount, difficulty, iconType)

	sessionID := generateSessionID()
	expiresAt := time.Now().Add(5 * time.Minute)

	puzzleData, err := json.Marshal(puzzle)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal puzzle: %w", err)
	}

	session := &models.CaptchaSession{
		SessionID:     sessionID,
		BackgroundURL: string(puzzleData),
		SliderURL:     string(puzzleData),
		Status:        "pending",
		VerifyCount:   0,
		MaxAttempts:   3,
		CreatedAt:     time.Now(),
		ExpiredAt:     expiresAt,
		ClientIP:      req.ClientIP,
		UserAgent:     req.UserAgent,
		Fingerprint:   req.Fingerprint,
	}

	if s.sessionCache != nil {
		if err := s.sessionCache.Set(ctx, session); err != nil {
			return nil, fmt.Errorf("failed to cache session: %w", err)
		}
	}

	if s.captchaRepo != nil {
		if err := s.captchaRepo.Create(session); err != nil {
			return nil, fmt.Errorf("failed to save session to database: %w", err)
		}
	}

	hintText := s.generateHintText(puzzle)

	imageDataURL := s.generateGridImage(puzzle)

	return &CreateGridResponse{
		SessionID:   sessionID,
		Puzzle:      puzzle,
		HintText:    hintText,
		ImageDataURL: imageDataURL,
		ExpiresIn:   int64(5 * time.Minute / time.Second),
		ExpiresAt:   expiresAt.Unix(),
	}, nil
}

func (s *GridGeneratorService) generateGridPuzzle(gridSize, targetCount int, difficulty string, iconType GridIconType) *GridPuzzle {
	rand.Seed(time.Now().UnixNano())

	cells := make([]GridCell, 0, gridSize*gridSize)
	colors := gridIconColors[iconType]
	icons := iconNames[iconType]

	cellID := 0
	for row := 0; row < gridSize; row++ {
		for col := 0; col < gridSize; col++ {
			cell := GridCell{
				Row:      row,
				Col:      col,
				IconType: iconType,
				IconID:   rand.Intn(len(icons)),
				Color:    colors[rand.Intn(len(colors))],
				IsTarget: false,
				Index:    cellID,
			}
			cells = append(cells, cell)
			cellID++
		}
	}

	targetIndices := make([]int, 0)
	usedPositions := make(map[int]bool)

	requiredTargets := s.calculateTargetCount(gridSize, targetCount, difficulty)

	for len(targetIndices) < requiredTargets {
		idx := rand.Intn(gridSize * gridSize)
		if !usedPositions[idx] {
			usedPositions[idx] = true
			targetIndices = append(targetIndices, idx)
		}
	}

	for i, idx := range targetIndices {
		cells[idx].IsTarget = true
		cells[idx].IconID = i
	}

	targetOrder := make([]int, len(targetIndices))
	for i := range targetOrder {
		targetOrder[i] = i
	}

	for i := len(targetOrder) - 1; i > 0; i-- {
		j := rand.Intn(i + 1)
		targetOrder[i], targetOrder[j] = targetOrder[j], targetOrder[i]
	}

	return &GridPuzzle{
		Cells:         cells,
		GridSize:      gridSize,
		TargetIndices: targetIndices,
		TargetOrder:   targetOrder,
		RequiredCount: requiredTargets,
		Difficulty:    difficulty,
	}
}

func (s *GridGeneratorService) calculateTargetCount(gridSize, requestedCount int, difficulty string) int {
	maxTargets := gridSize * gridSize / 2
	if maxTargets < 2 {
		maxTargets = 2
	}

	targetCount := requestedCount
	if targetCount > maxTargets {
		targetCount = maxTargets
	}

	switch difficulty {
	case "easy":
		if targetCount > 2 {
			targetCount = 2
		}
	case "medium":
		if targetCount > 3 {
			targetCount = 3
		}
	case "hard":
		if targetCount > 4 {
			targetCount = 4
		}
	}

	return targetCount
}

func (s *GridGeneratorService) generateHintText(puzzle *GridPuzzle) string {
	targets := puzzle.TargetIndices
	switch len(targets) {
	case 1:
		return "请选择绿色标记的目标"
	case 2:
		return fmt.Sprintf("请按正确顺序点击目标：第1个 → 第2个")
	case 3:
		return fmt.Sprintf("请按正确顺序点击目标：第1个 → 第2个 → 第3个")
	default:
		return fmt.Sprintf("请按正确顺序点击 %d 个目标", len(targets))
	}
}

func (s *GridGeneratorService) generateGridImage(puzzle *GridPuzzle) string {
	cellSize := 80
	padding := 10
	gridSize := puzzle.GridSize

	imageSize := gridSize*cellSize + (gridSize+1)*padding

	imageData := make([][]string, imageSize)
	for i := range imageData {
		imageData[i] = make([]string, imageSize)
		for j := range imageData[i] {
			imageData[i][j] = "#F0F0F0"
		}
	}

	for _, cell := range puzzle.Cells {
		startX := padding + cell.Col*(cellSize+padding)
		startY := padding + cell.Row*(cellSize+padding)

		targetColor := "#4CAF50"
		if cell.IsTarget {
			for y := startY; y < startY+cellSize && y < imageSize; y++ {
				for x := startX; x < startX+cellSize && x < imageSize; x++ {
					imageData[y][x] = targetColor
				}
			}
		}
	}

	imageJSON, _ := json.Marshal(imageData)
	return "data:image/json;base64," + string(imageJSON)
}

func (s *GridVerifierService) Verify(ctx context.Context, req *VerifyGridRequest) (*VerifyGridResult, error) {
	session, err := s.getSession(req.SessionID)
	if err != nil {
		return &VerifyGridResult{
			Success: false,
			Message: "会话不存在",
			Score:   0,
		}, nil
	}

	if time.Now().After(session.ExpiredAt) {
		return &VerifyGridResult{
			Success: false,
			Message: "验证码已过期",
			Score:   0,
		}, nil
	}

	if session.VerifyCount >= session.MaxAttempts {
		return &VerifyGridResult{
			Success: false,
			Message: "验证次数已用完",
			Score:   0,
		}, nil
	}

	s.incrementVerifyCount(req.SessionID)

	var puzzle GridPuzzle
	if err := json.Unmarshal([]byte(session.BackgroundURL), &puzzle); err != nil {
		return nil, fmt.Errorf("failed to unmarshal puzzle: %w", err)
	}

	if session.Status == "verified" {
		return &VerifyGridResult{
			Success:       true,
			Message:       "验证码已验证通过",
			Score:         100,
			CorrectCount:  puzzle.RequiredCount,
			TotalRequired: puzzle.RequiredCount,
		}, nil
	}

	correctCount := 0
	expectedOrder := make([]int, len(puzzle.TargetIndices))
	for i, idx := range puzzle.TargetIndices {
		for j, cell := range puzzle.Cells {
			if cell.Index == idx && cell.IsTarget {
				expectedOrder[i] = j
				break
			}
		}
	}

	for i := 0; i < len(req.SelectedOrder) && i < len(expectedOrder); i++ {
		if req.SelectedOrder[i] == expectedOrder[i] {
			correctCount++
		}
	}

	timeAnalysis := s.analyzeClickTime(req.TimeSpent, len(req.SelectedOrder), puzzle.Difficulty)

	clickPatternScore := s.analyzeClickPattern(req.ClickPattern, puzzle)

	totalScore := float64(correctCount) / float64(puzzle.RequiredCount) * 70
	totalScore += clickPatternScore * 30

	isSuccess := correctCount == puzzle.RequiredCount && clickPatternScore >= 0.5

	if isSuccess {
		session.Status = "verified"
		if s.sessionCache != nil {
			_ = s.sessionCache.UpdateStatus(ctx, req.SessionID, "verified")
		}
		if s.captchaRepo != nil {
			_ = s.captchaRepo.UpdateStatus(req.SessionID, "verified")
		}
	}

	return &VerifyGridResult{
		Success:          isSuccess,
		Message:          func() string {
			if isSuccess {
				return "验证成功"
			}
			return fmt.Sprintf("验证失败，正确 %d/%d", correctCount, puzzle.RequiredCount)
		}(),
		Score:            totalScore,
		CorrectCount:    correctCount,
		TotalRequired:    puzzle.RequiredCount,
		TimeAnalysis:     timeAnalysis,
		ClickPatternScore: clickPatternScore,
	}, nil
}

func (s *GridVerifierService) analyzeClickTime(timeSpent int64, clickCount int, difficulty string) string {
	avgTimePerClick := float64(timeSpent) / float64(clickCount)

	var expectedTime float64
	switch difficulty {
	case "easy":
		expectedTime = 2000
	case "medium":
		expectedTime = 1500
	case "hard":
		expectedTime = 1000
	default:
		expectedTime = 1500
	}

	if avgTimePerClick < 300 {
		return "过快，可能为机器操作"
	} else if avgTimePerClick > expectedTime*3 {
		return "过慢，可能存在异常"
	} else if avgTimePerClick > expectedTime*2 {
		return "较慢但正常"
	} else if avgTimePerClick < expectedTime/2 {
		return "较快但正常"
	}
	return "时间正常"
}

func (s *GridVerifierService) analyzeClickPattern(pattern []ClickPoint, puzzle GridPuzzle) float64 {
	if len(pattern) < 2 {
		return 0.5
	}

	var totalDistance float64
	for i := 1; i < len(pattern); i++ {
		dx := float64(pattern[i].Col - pattern[i-1].Col)
		dy := float64(pattern[i].Row - pattern[i-1].Row)
		totalDistance += dx*dx + dy*dy
	}

	avgDistance := totalDistance / float64(len(pattern)-1)

	if avgDistance < 0.1 {
		return 0.2
	}

	if avgDistance > 10 {
		return 0.6
	}

	var timeGaps int
	for i := 1; i < len(pattern); i++ {
		gap := pattern[i].Timestamp - pattern[i-1].Timestamp
		if gap > 100 && gap < 500 {
			timeGaps++
		}
	}

	variationScore := float64(timeGaps) / float64(len(pattern)-1)

	return 0.5 + variationScore*0.5
}

func (s *GridVerifierService) getSession(sessionID string) (*models.CaptchaSession, error) {
	if s.sessionCache != nil {
		session, err := s.sessionCache.Get(context.Background(), sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	if s.captchaRepo != nil {
		session, err := s.captchaRepo.GetBySessionID(sessionID)
		if err == nil && session != nil {
			return session, nil
		}
	}

	return nil, fmt.Errorf("session not found: %s", sessionID)
}

func (s *GridVerifierService) incrementVerifyCount(sessionID string) {
	if s.sessionCache != nil {
		_ = s.sessionCache.IncrementVerifyCount(context.Background(), sessionID)
	}

	if s.captchaRepo != nil {
		_ = s.captchaRepo.UpdateVerifyCount(sessionID)
	}
}

func (s *GridVerifierService) GetSessionStatus(ctx context.Context, sessionID string) (*models.CaptchaSession, error) {
	return s.getSession(sessionID)
}

func (s *GridVerifierService) CheckSessionValid(ctx context.Context, sessionID string) (bool, string) {
	session, err := s.getSession(sessionID)
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
