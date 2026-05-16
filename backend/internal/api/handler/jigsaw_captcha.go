package handler

import (
	"context"
	"encoding/base64"
	"encoding/json"
	"image"
	"image/color"
	"image/draw"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type JigsawGridSize int

const (
	Grid2x2 JigsawGridSize = 2
	Grid3x3 JigsawGridSize = 3
	Grid4x4 JigsawGridSize = 4
)

type JigsawPiece struct {
	Index     int   `json:"index"`
	OriginalX int   `json:"originalX"`
	OriginalY int   `json:"originalY"`
	CurrentX  int   `json:"currentX"`
	CurrentY  int   `json:"currentY"`
	Width     int   `json:"width"`
	Height    int   `json:"height"`
	Rotation  int   `json:"rotation"`
}

type JigsawCaptchaConfig struct {
	Width      int
	Height     int
	GridSize   JigsawGridSize
	MaxAttempt int
	SessionTTL time.Duration
}

var defaultJigsawConfig = JigsawCaptchaConfig{
	Width:      300,
	Height:     300,
	GridSize:   Grid3x3,
	MaxAttempt: 5,
	SessionTTL: 5 * time.Minute,
}

type JigsawSession struct {
	SessionID     string         `json:"sessionId"`
	BackgroundImg *imageData     `json:"backgroundImg"`
	Pieces        []*JigsawPiece `json:"pieces"`
	PieceImages   []*imageData   `json:"pieceImages"`
	GridSize      JigsawGridSize `json:"gridSize"`
	Attempts      int            `json:"attempts"`
	Verified      bool           `json:"verified"`
	CreatedAt     time.Time      `json:"createdAt"`
	ExpiresAt     time.Time      `json:"expiresAt"`
	ClientIP      string         `json:"clientIp"`
	UserAgent     string         `json:"userAgent"`
}

type GenerateJigsawRequest struct {
	Width    int             `form:"width" json:"width"`
	Height   int             `form:"height" json:"height"`
	GridSize JigsawGridSize  `form:"gridSize" json:"gridSize"`
}

type GenerateJigsawResponse struct {
	SessionID     string         `json:"sessionId"`
	ImageURL      string         `json:"imageUrl"`
	Pieces        []*JigsawPiece `json:"pieces"`
	PieceImages   []string       `json:"pieceImages"`
	GridSize      JigsawGridSize `json:"gridSize"`
	PieceWidth    int            `json:"pieceWidth"`
	PieceHeight   int            `json:"pieceHeight"`
	ImageWidth    int            `json:"imageWidth"`
	ImageHeight   int            `json:"imageHeight"`
}

type VerifyJigsawRequest struct {
	SessionID string         `json:"sessionId" binding:"required"`
	Pieces    []*JigsawPiece `json:"pieces" binding:"required"`
}

type VerifyJigsawResponse struct {
	Success  bool   `json:"success"`
	Message  string `json:"message"`
	Remaining int   `json:"remainingAttempts"`
}

var (
	jigsawSessionStore = make(map[string]*JigsawSession)
	jigsawSessionMu    sync.RWMutex
	jigsawPrng         = rand.New(rand.NewSource(time.Now().UnixNano()))
	jigsawPrngMu       sync.Mutex
)

func init() {
	go cleanupExpiredJigsawSessions()
}

func cleanupExpiredJigsawSessions() {
	ticker := time.NewTicker(2 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		now := time.Now()
		jigsawSessionMu.Lock()
		for id, session := range jigsawSessionStore {
			if now.After(session.ExpiresAt) || session.Verified {
				delete(jigsawSessionStore, id)
			}
		}
		jigsawSessionMu.Unlock()
	}
}

func randIntJigsaw(min, max int) int {
	jigsawPrngMu.Lock()
	defer jigsawPrngMu.Unlock()
	return min + jigsawPrng.Intn(max-min+1)
}

func generateJigsawBackground(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	bgStyle := randIntJigsaw(0, 3)
	switch bgStyle {
	case 0:
		drawGradientBackground(img)
	case 1:
		drawPatternBackground(img, width, height)
	case 2:
		drawGeometricBackground(img, width, height)
	default:
		drawGradientBackground(img)
	}
	
	addImageNoise(img, width, height)
	return img
}

func splitImageToPieces(img image.Image, gridSize JigsawGridSize) ([]image.Image, int, int) {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()
	pieceWidth := width / int(gridSize)
	pieceHeight := height / int(gridSize)
	
	pieces := make([]image.Image, 0, int(gridSize)*int(gridSize))
	for y := 0; y < int(gridSize); y++ {
		for x := 0; x < int(gridSize); x++ {
			pieceRect := image.Rect(
				x*pieceWidth, 
				y*pieceHeight,
				(x+1)*pieceWidth,
				(y+1)*pieceHeight,
			)
			pieceImg := image.NewRGBA(pieceRect)
			draw.Draw(pieceImg, pieceRect, img, image.Pt(x*pieceWidth, y*pieceHeight), draw.Src)
			
			pieces = append(pieces, pieceImg)
		}
	}
	return pieces, pieceWidth, pieceHeight
}

func shufflePieces(originalPieces []*JigsawPiece) []*JigsawPiece {
	shuffled := make([]*JigsawPiece, len(originalPieces))
	copy(shuffled, originalPieces)
	
	for i := len(shuffled) - 1; i > 0; i-- {
		j := randIntJigsaw(0, i)
		shuffled[i], shuffled[j] = shuffled[j], shuffled[i]
	}
	
	gridSize := int(math.Sqrt(float64(len(shuffled))))
	for i, piece := range shuffled {
		piece.Index = i
		piece.CurrentX = i % gridSize
		piece.CurrentY = i / gridSize
		piece.Rotation = randIntJigsaw(0, 3) * 90
	}
	
	return shuffled
}

func rotatePiece(img image.Image, angle int) image.Image {
	if angle == 0 {
		return img
	}
	
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	var rotated *image.RGBA
	switch angle {
	case 90, 270:
		rotated = image.NewRGBA(image.Rect(0, 0, height, width))
	default:
		rotated = image.NewRGBA(image.Rect(0, 0, width, height))
	}
	
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			var nx, ny int
			switch angle {
			case 90:
				nx = height - 1 - y
				ny = x
			case 180:
				nx = width - 1 - x
				ny = height - 1 - y
			case 270:
				nx = y
				ny = width - 1 - x
			default:
				nx = x
				ny = y
			}
			
			rotated.Set(nx, ny, img.At(x, y))
		}
	}
	
	return rotated
}

func addPieceBorder(img image.Image) image.Image {
	bounds := img.Bounds()
	width, height := bounds.Dx(), bounds.Dy()
	
	result := image.NewRGBA(bounds)
	draw.Draw(result, bounds, img, image.Pt(0, 0), draw.Src)
	
	borderColor := color.RGBA{R: 201, G: 169, B: 110, A: 255}
	for x := 0; x < width; x++ {
		result.Set(x, 0, borderColor)
		result.Set(x, height-1, borderColor)
	}
	for y := 0; y < height; y++ {
		result.Set(0, y, borderColor)
		result.Set(width-1, y, borderColor)
	}
	
	return result
}

func generateJigsawSessionID() string {
	bytes := make([]byte, 16)
	rand.Read(bytes)
	return base64.URLEncoding.EncodeToString(bytes)
}

func saveJigsawSessionToRedis(session *JigsawSession) {
	if redis.Client == nil {
		return
	}
	ctx := context.Background()
	data, err := json.Marshal(session)
	if err != nil {
		return
	}
	key := "jigsaw_session:" + session.SessionID
	redis.Client.Set(ctx, key, data, defaultJigsawConfig.SessionTTL)
}

func getJigsawSessionFromRedis(sessionID string) (*JigsawSession, bool) {
	if redis.Client == nil {
		jigsawSessionMu.RLock()
		defer jigsawSessionMu.RUnlock()
		session, exists := jigsawSessionStore[sessionID]
		return session, exists
	}
	
	ctx := context.Background()
	key := "jigsaw_session:" + sessionID
	data, err := redis.Client.Get(ctx, key).Result()
	if err != nil {
		return nil, false
	}
	
	var session JigsawSession
	if err := json.Unmarshal([]byte(data), &session); err != nil {
		return nil, false
	}
	return &session, true
}

func deleteJigsawSessionFromRedis(sessionID string) {
	if redis.Client == nil {
		jigsawSessionMu.Lock()
		delete(jigsawSessionStore, sessionID)
		jigsawSessionMu.Unlock()
		return
	}
	ctx := context.Background()
	key := "jigsaw_session:" + sessionID
	redis.Client.Del(ctx, key)
}

func GenerateJigsawCaptcha(c *gin.Context) {
	var req GenerateJigsawRequest
	if err := c.ShouldBind(&req); err != nil {
		req.Width = 0
		req.Height = 0
		req.GridSize = 0
	}
	
	config := defaultJigsawConfig
	if req.Width > 0 && req.Width <= 500 {
		config.Width = req.Width
	}
	if req.Height > 0 && req.Height <= 500 {
		config.Height = req.Height
	}
	if req.GridSize >= Grid2x2 && req.GridSize <= Grid4x4 {
		config.GridSize = req.GridSize
	}
	
	sessionID := generateJigsawSessionID()
	bgImg := generateJigsawBackground(config.Width, config.Height)
	
	pieceImgs, pieceWidth, pieceHeight := splitImageToPieces(bgImg, config.GridSize)
	
	originalPieces := make([]*JigsawPiece, 0, int(config.GridSize)*int(config.GridSize))
	for y := 0; y < int(config.GridSize); y++ {
		for x := 0; x < int(config.GridSize); x++ {
			idx := y*int(config.GridSize) + x
			originalPieces = append(originalPieces, &JigsawPiece{
				Index:     idx,
				OriginalX: x,
				OriginalY: y,
				CurrentX:  x,
				CurrentY:  y,
				Width:     pieceWidth,
				Height:    pieceHeight,
				Rotation:  0,
			})
		}
	}
	
	shuffledPieces := shufflePieces(originalPieces)
	
	pieceImages := make([]*imageData, 0, len(shuffledPieces))
	for i, piece := range shuffledPieces {
		_ = i
		rotatedImg := rotatePiece(pieceImgs[piece.Index], piece.Rotation)
		borderedImg := addPieceBorder(rotatedImg)
		pieceImages = append(pieceImages, &imageData{
			DataURL: imageToDataURL(borderedImg),
			Width:   pieceWidth,
			Height:  pieceHeight,
		})
	}
	
	session := &JigsawSession{
		SessionID: sessionID,
		BackgroundImg: &imageData{
			DataURL: imageToDataURL(bgImg),
			Width:   config.Width,
			Height:  config.Height,
		},
		Pieces:    shuffledPieces,
		PieceImages: pieceImages,
		GridSize:  config.GridSize,
		Attempts:  0,
		Verified:  false,
		CreatedAt: time.Now(),
		ExpiresAt: time.Now().Add(config.SessionTTL),
		ClientIP:  c.ClientIP(),
		UserAgent: c.GetHeader("User-Agent"),
	}
	
	jigsawSessionMu.Lock()
	jigsawSessionStore[sessionID] = session
	jigsawSessionMu.Unlock()
	
	saveJigsawSessionToRedis(session)
	
	pieceImgURLs := make([]string, 0, len(pieceImages))
	for _, img := range pieceImages {
		pieceImgURLs = append(pieceImgURLs, img.DataURL)
	}
	
	c.JSON(http.StatusOK, GenerateJigsawResponse{
		SessionID:   sessionID,
		ImageURL:    session.BackgroundImg.DataURL,
		Pieces:      shuffledPieces,
		PieceImages: pieceImgURLs,
		GridSize:    config.GridSize,
		PieceWidth:  pieceWidth,
		PieceHeight: pieceHeight,
		ImageWidth:  config.Width,
		ImageHeight: config.Height,
	})
}

func verifyJigsawPieces(sessionPieces, submittedPieces []*JigsawPiece, gridSize JigsawGridSize) bool {
	if len(sessionPieces) != len(submittedPieces) {
		return false
	}
	
	gridSizeInt := int(gridSize)
	_ = gridSizeInt
	expectedPositions := make(map[string]bool)
	
	for _, piece := range sessionPieces {
		key := positionKey(piece.OriginalX, piece.OriginalY, 0)
		expectedPositions[key] = true
	}
	
	correctCount := 0
	for _, submitted := range submittedPieces {
		sessionPiece := findPieceByIndex(sessionPieces, submitted.Index)
		if sessionPiece == nil {
			continue
		}
		
		if submitted.CurrentX == sessionPiece.OriginalX &&
		   submitted.CurrentY == sessionPiece.OriginalY &&
		   submitted.Rotation%360 == 0 {
			correctCount++
		}
	}
	
	return correctCount == len(sessionPieces)
}

func findPieceByIndex(pieces []*JigsawPiece, index int) *JigsawPiece {
	for _, p := range pieces {
		if p.Index == index {
			return p
		}
	}
	return nil
}

func positionKey(x, y, rotation int) string {
	return string(rune(x)) + "," + string(rune(y)) + "," + string(rune(rotation))
}

func VerifyJigsawCaptcha(c *gin.Context) {
	var req VerifyJigsawRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request parameters")
		return
	}
	
	session, exists := getJigsawSessionFromRedis(req.SessionID)
	if !exists {
		response.NotFound(c, "Captcha session does not exist or has expired")
		return
	}
	
	if time.Now().After(session.ExpiresAt) {
		deleteJigsawSessionFromRedis(req.SessionID)
		response.NotFound(c, "Captcha has expired, please refresh")
		return
	}
	
	if session.Verified {
		response.BadRequest(c, "Captcha already verified")
		return
	}
	
	session.Attempts++
	if session.Attempts > defaultJigsawConfig.MaxAttempt {
		deleteJigsawSessionFromRedis(req.SessionID)
		response.BadRequest(c, "Too many attempts, please refresh")
		return
	}
	
	verified := verifyJigsawPieces(session.Pieces, req.Pieces, session.GridSize)
	
	if verified {
		session.Verified = true
		deleteJigsawSessionFromRedis(req.SessionID)
		
		response.Success(c, VerifyJigsawResponse{
			Success:   true,
			Message:   "Verification successful",
			Remaining: defaultJigsawConfig.MaxAttempt - session.Attempts,
		})
		return
	}
	
	remaining := defaultJigsawConfig.MaxAttempt - session.Attempts
	if remaining <= 0 {
		deleteJigsawSessionFromRedis(req.SessionID)
		response.BadRequest(c, "Verification failed, attempts exhausted")
		return
	}
	
	response.Success(c, VerifyJigsawResponse{
		Success:   false,
		Message:   "Incorrect arrangement, please try again",
		Remaining: remaining,
	})
}
