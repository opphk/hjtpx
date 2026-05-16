package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"math/rand"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type CaptchaSession struct {
	ID         string
	Type       string
	TargetX    int
	TargetY    int
	Points     [][2]int
	Hint       string
	MaxPoints  int
	CreatedAt  time.Time
}

var (
	captchaSessions = make(map[string]*CaptchaSession)
	sessionMutex    sync.RWMutex
	behaviorService = service.NewBehaviorAnalysisService()
)

func init() {
	rand.Seed(time.Now().UnixNano())
	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredSessions()
		}
	}()
}

func generateSessionID() string {
	return fmt.Sprintf("sess_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func generateSliderImage() (string, string, int, int) {
	generator := NewCaptchaImageGenerator()

	backgroundBase64 := generator.EncodeBackgroundToBase64()
	sliderBase64 := generator.EncodeSliderToBase64()

	return backgroundBase64, sliderBase64, generator.GetTargetX(), generator.GetTargetY()
}

func generateClickImage() (string, string, [][2]int, int) {
	return generateAdvancedClickImage()
}

func generateAdvancedClickImage() (string, string, [][2]int, int) {
	width := 360
	height := 220
	
	rng := rand.New(rand.NewSource(time.Now().UnixNano()))
	
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	
	drawRandomBackground(img, width, height, rng)
	
	targetSymbols := []string{"★", "♦", "♣", "♥", "●", "▲", "■", "◆"}
	targetColors := []color.RGBA{
		{255, 215, 0, 255},
		{255, 100, 100, 255},
		{100, 255, 100, 255},
		{100, 100, 255, 255},
		{255, 150, 0, 255},
	}
	
	targetCount := 2 + rng.Intn(3)
	targets := make([][2]int, 0, targetCount)
	usedPositions := make(map[string]bool)
	
	margin := 40
	for len(targets) < targetCount {
		x := margin + rng.Intn(width-2*margin)
		y := margin + rng.Intn(height-2*margin)
		
		posKey := fmt.Sprintf("%d_%d", x/30, y/30)
		if !usedPositions[posKey] {
			usedPositions[posKey] = true
			targets = append(targets, [2]int{x, y})
		}
	}
	
	distractorCount := 3 + rng.Intn(4)
	distractorSymbols := []string{"☆", "♠", "○", "△", "□", "◇", "×", "+"}
	
	for i := 0; i < distractorCount; i++ {
		x := margin + rng.Intn(width-2*margin)
		y := margin + rng.Intn(height-2*margin)
		
		posKey := fmt.Sprintf("%d_%d", x/30, y/30)
		if !usedPositions[posKey] {
			usedPositions[posKey] = true
			
			symbol := distractorSymbols[rng.Intn(len(distractorSymbols))]
			drawTextOnImage(img, x, y, symbol, 24, color.RGBA{180, 180, 180, 200})
		}
	}
	
	hintSymbols := make([]string, 0, targetCount)
	for _, target := range targets {
		symbol := targetSymbols[rng.Intn(len(targetSymbols))]
		targetColor := targetColors[rng.Intn(len(targetColors))]
		
		drawCircleHighlight(img, target[0], target[1], 25, color.RGBA{255, 255, 255, 100})
		drawTextOnImage(img, target[0], target[1], symbol, 28, targetColor)
		
		hintSymbols = append(hintSymbols, symbol)
	}
	
	hintText := "请点击: "
	for i, sym := range hintSymbols {
		if i > 0 {
			hintText += ", "
		}
		hintText += sym
	}
	
	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		svg := generateFallbackSVG(targets, hintText)
		encoded := base64.StdEncoding.EncodeToString([]byte(svg))
		return "data:image/svg+xml;base64," + encoded, hintText, targets, targetCount
	}
	
	encoded := base64.StdEncoding.EncodeToString(buf.Bytes())
	return "data:image/png;base64," + encoded, hintText, targets, targetCount
}

func drawRandomBackground(img *image.RGBA, width, height int, rng *rand.Rand) {
	theme := rng.Intn(5)
	
	switch theme {
	case 0:
		drawLandscapeBG(img, width, height, rng)
	case 1:
		drawUrbanBG(img, width, height, rng)
	case 2:
		drawNatureBG(img, width, height, rng)
	case 3:
		drawAbstractBG(img, width, height, rng)
	case 4:
		drawOceanBG(img, width, height, rng)
	}
}

func drawLandscapeBG(img *image.RGBA, width, height int, rng *rand.Rand) {
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		r := uint8(40 + int(60*ratio))
		g := uint8(120 - int(40*ratio))
		b := uint8(180 - int(60*ratio))
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	
	for i := 0; i < 3; i++ {
		peakX := rng.Intn(width)
		peakY := height/2 + rng.Intn(height/4)
		peakHeight := 30 + rng.Intn(50)
		
		for y := peakY - peakHeight; y < height; y++ {
			distFromPeak := math.Abs(float64(peakX) - float64(y-peakY)*3)
			if distFromPeak < float64(height-y)*2 {
				shade := 0.3 + 0.7*(float64(y-(peakY-peakHeight))/float64(peakHeight))
				r := uint8(60 * shade)
				g := uint8(140 * shade)
				b := uint8(80 * shade)
				img.Set(peakX+int(y-peakY)*2-rng.Intn(4)+2, y, color.RGBA{R: r, G: g, B: b, A: 255})
			}
		}
	}
}

func drawUrbanBG(img *image.RGBA, width, height int, rng *rand.Rand) {
	for y := 0; y < height/2; y++ {
		ratio := float64(y) / float64(height/2)
		r := uint8(100 + int(60*ratio))
		g := uint8(100 + int(60*ratio))
		b := uint8(140 + int(40*ratio))
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	
	for y := height / 2; y < height; y++ {
		ratio := float64(y-height/2) / float64(height/2)
		r := uint8(80 - int(20*ratio))
		g := uint8(80 - int(20*ratio))
		b := uint8(120 - int(40*ratio))
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	
	for i := 0; i < 8; i++ {
		bx := i * (width / 8)
		bw := 20 + rng.Intn(30)
		bh := 60 + rng.Intn(80)
		by := height - bh
		
		buildingColor := color.RGBA{
			R: uint8(60 + rng.Intn(40)),
			G: uint8(60 + rng.Intn(40)),
			B: uint8(80 + rng.Intn(40)),
			A: 255,
		}
		
		for y := by; y < height; y++ {
			for x := bx; x < bx+bw && x < width; x++ {
				img.Set(x, y, buildingColor)
			}
		}
		
		windowRows := bh / 10
		windowCols := bw / 8
		for wy := 0; wy < windowRows; wy++ {
			for wx := 0; wx < windowCols; wx++ {
				if rng.Float32() > 0.3 {
					wxPos := bx + wx*8 + 3
					wyPos := by + wy*10 + 3
					windowColor := color.RGBA{
						R: uint8(200 + rng.Intn(55)),
						G: uint8(180 + rng.Intn(75)),
						B: uint8(100 + rng.Intn(155)),
						A: 255,
					}
					for dy := 0; dy < 6; dy++ {
						for dx := 0; dx < 5; dx++ {
							if wxPos+dx < width && wyPos+dy < height {
								img.Set(wxPos+dx, wyPos+dy, windowColor)
							}
						}
					}
				}
			}
		}
	}
}

func drawNatureBG(img *image.RGBA, width, height int, rng *rand.Rand) {
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		var r, g, b uint8
		if ratio < 0.4 {
			part := ratio / 0.4
			r = uint8(180 - int(80*part))
			g = uint8(220 - int(60*part))
			b = uint8(200 - int(80*part))
		} else {
			part := (ratio - 0.4) / 0.6
			r = uint8(100 - int(40*part))
			g = uint8(160 - int(60*part))
			b = uint8(120 - int(50*part))
		}
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	
	for i := 0; i < 5; i++ {
		tx := rng.Intn(width)
		ty := height - 20 - rng.Intn(40)
		treeHeight := 40 + rng.Intn(60)
		
		trunkWidth := 4 + rng.Intn(4)
		for y := ty; y < height; y++ {
			for x := tx - trunkWidth/2; x < tx+trunkWidth/2; x++ {
				if x >= 0 && x < width {
					img.Set(x, y, color.RGBA{R: 100, G: 70, B: 40, A: 255})
				}
			}
		}
		
		canopyRadius := 20 + rng.Intn(20)
		for angle := 0.0; angle < 2*math.Pi; angle += 0.1 {
			for rad := 0.0; rad < float64(canopyRadius); rad += 1 {
				cx := int(float64(tx) + rad*math.Cos(angle))
				cy := int(float64(ty-treeHeight/2) + rad*math.Sin(angle))
				if cx >= 0 && cx < width && cy >= 0 && cy < ty+treeHeight/2 {
					shade := 0.7 + 0.3*(rad/float64(canopyRadius))
					img.Set(cx, cy, color.RGBA{
						R: uint8(40 * shade),
						G: uint8(140 * shade),
						B: uint8(60 * shade),
						A: 255,
					})
				}
			}
		}
	}
}

func drawAbstractBG(img *image.RGBA, width, height int, rng *rand.Rand) {
	baseColor := color.RGBA{180, 80, 120, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, baseColor)
		}
	}
	
	for i := 0; i < 6; i++ {
		cx := rng.Intn(width)
		cy := rng.Intn(height)
		radius := 30 + rng.Intn(60)
		circleColor := color.RGBA{
			R: uint8(200 + rng.Intn(55)),
			G: uint8(100 + rng.Intn(155)),
			B: uint8(120 + rng.Intn(135)),
			A: 255,
		}
		
		for angle := 0.0; angle < 2*math.Pi; angle += 0.05 {
			for rad := 0.0; rad < float64(radius); rad += 1 {
				px := int(float64(cx) + rad*math.Cos(angle))
				py := int(float64(cy) + rad*math.Sin(angle))
				if px >= 0 && px < width && py >= 0 && py < height {
					alpha := uint8(150 - int(100*(rad/float64(radius))))
					img.Set(px, py, color.RGBA{
						R: circleColor.R,
						G: circleColor.G,
						B: circleColor.B,
						A: alpha,
					})
				}
			}
		}
	}
}

func drawOceanBG(img *image.RGBA, width, height int, rng *rand.Rand) {
	for y := 0; y < height; y++ {
		ratio := float64(y) / float64(height)
		r := uint8(30 + int(50*ratio))
		g := uint8(100 + int(60*ratio))
		b := uint8(160 + int(40*ratio))
		for x := 0; x < width; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
	
	for wave := 0; wave < 4; wave++ {
		waveY := height/4 + wave*height/5
		amplitude := 5 + rng.Intn(10)
		frequency := 0.02 + rng.Float64()*0.02
		
		for x := 0; x < width; x++ {
			y := waveY + int(float64(amplitude)*math.Sin(float64(x)*frequency+float64(wave)*math.Pi/2))
			if y >= 0 && y < height {
				shade := 1.0 - 0.3*float64(wave)/4.0
				img.Set(x, y, color.RGBA{
					R: uint8(100 * shade),
					G: uint8(160 * shade),
					B: uint8(200 * shade),
					A: 255,
				})
			}
		}
	}
}

func drawTextOnImage(img *image.RGBA, cx, cy int, text string, size int, col color.RGBA) {
	runes := []rune(text)
	totalWidth := len(runes) * size
	
	startX := cx - totalWidth/2
	startY := cy - size/2
	
	for y := startY; y < startY+size && y < img.Bounds().Dy(); y++ {
		for x := startX; x < startX+totalWidth && x < img.Bounds().Dx(); x++ {
			if x >= 0 && y >= 0 {
				img.Set(x, y, col)
			}
		}
	}
}

func drawCircleHighlight(img *image.RGBA, cx, cy, radius int, col color.RGBA) {
	for angle := 0.0; angle < 2*math.Pi; angle += 0.05 {
		for rad := 0.0; rad < float64(radius); rad += 0.5 {
			x := int(float64(cx) + rad*math.Cos(angle))
			y := int(float64(cy) + rad*math.Sin(angle))
			
			if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				alpha := uint8(float64(col.A) * (1 - rad/float64(radius)))
				blendedColor := color.RGBA{
					R: col.R,
					G: col.G,
					B: col.B,
					A: alpha,
				}
				img.Set(x, y, blendedColor)
			}
		}
	}
}

func generateFallbackSVG(targets [][2]int, hint string) string {
	svg := `<?xml version="1.0" encoding="UTF-8"?>
<svg xmlns="http://www.w3.org/2000/svg" width="360" height="220">
<defs>
<linearGradient id="bg" x1="0%" y1="0%" x2="100%" y2="100%">
<stop offset="0%" style="stop-color:#667eea"/>
<stop offset="100%" style="stop-color:#764ba2"/>
</linearGradient>
</defs>
<rect width="100%" height="100%" fill="url(#bg)"/>`

	for _, target := range targets {
		svg += fmt.Sprintf(`<circle cx="%d" cy="%d" r="20" stroke="white" stroke-width="2" fill="rgba(255,255,255,0.3)"/>`, target[0], target[1])
	}

	svg += `</svg>`
	return svg
}

func GetSliderCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	backgroundImage, _, targetX, targetY := generateSliderImage()

	session := &CaptchaSession{
		ID:        sessionID,
		Type:      "slider",
		TargetX:   targetX,
		TargetY:   targetY,
		CreatedAt: time.Now(),
	}

	sessionMutex.Lock()
	captchaSessions[sessionID] = session
	sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"image_url":  backgroundImage,
		"puzzle_y":   targetY,
	})
}

func GetClickCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	imageURL, hint, points, maxPoints := generateClickImage()

	session := &CaptchaSession{
		ID:        sessionID,
		Type:      "click",
		Points:    points,
		Hint:      hint,
		MaxPoints: maxPoints,
		CreatedAt: time.Now(),
	}

	sessionMutex.Lock()
	captchaSessions[sessionID] = session
	sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"session_id": sessionID,
		"image_url":  imageURL,
		"hint":       hint,
		"max_points": maxPoints,
	})
}

type VerifyRequest struct {
	SessionID      string               `json:"session_id" binding:"required"`
	Type           string               `json:"type" binding:"required"`
	X              int                  `json:"x"`
	Y              int                  `json:"y"`
	Points         []ClickPoint         `json:"points"`
	BehaviorData   []BehaviorDataPoint  `json:"behavior_data"`
	ApplicationID  uint                 `json:"application_id"`
	VerificationTime int64              `json:"verification_time"`
}

type ClickPoint struct {
	X          int    `json:"x"`
	Y          int    `json:"y"`
	ImageIndex int    `json:"imageIndex"`
	ClickOrder int    `json:"clickOrder"`
}

type BehaviorDataPoint struct {
	X         int     `json:"x"`
	Y         int     `json:"y"`
	Timestamp int64   `json:"timestamp"`
	Event     string  `json:"event"`
}

func VerifyCaptcha(c *gin.Context) {
	startTime := time.Now()
	var req VerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "无效的请求参数",
		})
		return
	}

	sessionMutex.RLock()
	session, exists := captchaSessions[req.SessionID]
	sessionMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "会话不存在或已过期",
		})
		return
	}

	if session.Type != req.Type {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "验证类型不匹配",
		})
		return
	}

	var captchaSuccess bool

	if req.Type == "slider" {
		tolerance := 10
		captchaSuccess = abs(req.X - session.TargetX) <= tolerance && abs(req.Y - session.TargetY) <= tolerance
	} else if req.Type == "click" {
		if len(req.Points) != session.MaxPoints {
			captchaSuccess = false
		} else {
			captchaSuccess = true
			
			adaptiveTolerance := calculateAdaptiveTolerance(session.Points, req.Points)
			
			for i := 0; i < session.MaxPoints; i++ {
				sessionPoint := session.Points[i]
				reqPoint := req.Points[i]
				
				pointDistance := math.Sqrt(
					math.Pow(float64(reqPoint.X-sessionPoint[0]), 2) +
					math.Pow(float64(reqPoint.Y-sessionPoint[1]), 2),
				)
				
				if pointDistance > float64(adaptiveTolerance) {
					captchaSuccess = false
					break
				}
			}
			
			if captchaSuccess && len(req.Points) > 1 {
				sessionOrderCorrect := verifyClickOrder(session.Points, req.Points)
				captchaSuccess = captchaSuccess && sessionOrderCorrect
			}
		}
	}
	
	if req.VerificationTime > 0 {
		fmt.Printf("Verification completed in %dms\n", req.VerificationTime)
	}

	db := database.GetDB()

	behaviorDataList := make([]models.BehaviorData, 0, len(req.BehaviorData))
	for _, dp := range req.BehaviorData {
		dataJSON, _ := json.Marshal(dp)
		behaviorDataList = append(behaviorDataList, models.BehaviorData{
			Data:      string(dataJSON),
			DataType:  dp.Event,
			Timestamp: time.UnixMilli(dp.Timestamp),
		})
	}

	finalSuccess, riskScore, analysisReport := behaviorService.VerifyWithBehaviorAnalysis(
		captchaSuccess,
		behaviorDataList,
	)

	status := "failed"
	if finalSuccess {
		status = "success"
		sessionMutex.Lock()
		delete(captchaSessions, req.SessionID)
		sessionMutex.Unlock()
	}

	duration := time.Since(startTime).Milliseconds()

	verification := &models.Verification{
		SessionID:     req.SessionID,
		CaptchaType:   req.Type,
		ApplicationID: req.ApplicationID,
		UserID:        1,
		Status:        status,
		IPAddress:     c.ClientIP(),
		UserAgent:     c.GetHeader("User-Agent"),
		RiskScore:     riskScore,
		BehaviorData:  behaviorDataList,
	}

	if err := db.Create(verification).Error; err != nil {
		fmt.Printf("Failed to save verification: %v\n", err)
	}

	logEntry := &models.VerificationLog{
		VerificationID: verification.ID,
		SessionID:      req.SessionID,
		ApplicationID:  req.ApplicationID,
		CaptchaType:    req.Type,
		Status:         status,
		IPAddress:     c.ClientIP(),
		UserAgent:     c.GetHeader("User-Agent"),
		RiskScore:      riskScore,
		AnalysisResult: analysisReport,
		Duration:       duration,
	}

	if err := db.Create(logEntry).Error; err != nil {
		fmt.Printf("Failed to save verification log: %v\n", err)
	}

	message := "验证失败"
	if finalSuccess {
		message = "验证成功"
	}

	c.JSON(http.StatusOK, gin.H{
		"success":      finalSuccess,
		"message":      message,
		"risk_score":   riskScore,
		"captcha_pass": captchaSuccess,
	})
}

func calculateAdaptiveTolerance(sessionPoints [][2]int, reqPoints []ClickPoint) int {
	baseTolerance := 35
	
	if len(sessionPoints) == 0 || len(reqPoints) == 0 {
		return baseTolerance
	}
	
	var totalDistance float64
	for i := 0; i < len(sessionPoints) && i < len(reqPoints); i++ {
		dx := float64(reqPoints[i].X - sessionPoints[i][0])
		dy := float64(reqPoints[i].Y - sessionPoints[i][1])
		totalDistance += math.Sqrt(dx*dx + dy*dy)
	}
	
	avgDistance := totalDistance / float64(len(sessionPoints))
	
	if avgDistance < 10 {
		return 25
	} else if avgDistance < 20 {
		return 30
	} else if avgDistance < 30 {
		return 35
	} else {
		return 40
	}
}

func verifyClickOrder(sessionPoints [][2]int, reqPoints []ClickPoint) bool {
	if len(sessionPoints) != len(reqPoints) {
		return false
	}
	
	matched := make([]bool, len(sessionPoints))
	
	for i := 0; i < len(reqPoints); i++ {
		found := false
		for j := 0; j < len(sessionPoints); j++ {
			if matched[j] {
				continue
			}
			
			dx := float64(reqPoints[i].X - sessionPoints[j][0])
			dy := float64(reqPoints[i].Y - sessionPoints[j][1])
			distance := math.Sqrt(dx*dx + dy*dy)
			
			if distance <= 40 {
				matched[j] = true
				found = true
				break
			}
		}
		
		if !found {
			return false
		}
	}
	
	return true
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func cleanupExpiredSessions() {
	sessionMutex.Lock()
	defer sessionMutex.Unlock()
	now := time.Now()
	for id, session := range captchaSessions {
		if now.Sub(session.CreatedAt) > 10*time.Minute {
			delete(captchaSessions, id)
		}
	}
}
