package handler

import (
	"bytes"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math"
	"math/rand"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type CaptchaMode string

const (
	ModeNumber   CaptchaMode = "number"
	ModeLetter   CaptchaMode = "letter"
	ModeChinese  CaptchaMode = "chinese"
	ModeMixed    CaptchaMode = "mixed"
)

type ClickPoint struct {
	X      int `json:"x"`
	Y      int `json:"y"`
	Index  int `json:"index"`
}

type CaptchaSession struct {
	ID            string
	Type          string
	Mode          CaptchaMode
	TargetPoints  []ClickPoint
	HintOrder     []int
	AllowShuffle  bool
	Points        [][2]int
	Hint          string
	MaxPoints     int
	CreatedAt     time.Time
	Tolerance     int
	ImageWidth    int
	ImageHeight   int
	ImageSeed     int64
	TargetX       int
	TargetY       int
}

var (
	captchaSessions = make(map[string]*CaptchaSession)
	sessionMutex    sync.RWMutex
	behaviorService = service.NewBehaviorAnalysisService()
)

var clickChineseChars = []string{
	"中", "国", "人", "民", "友", "好", "太", "阳", "月", "亮",
	"星", "辰", "海", "洋", "山", "川", "河", "流", "风", "雨",
	"雪", "云", "花", "草", "树", "木", "林", "森", "天", "地",
	"东", "西", "南", "北", "春", "夏", "秋", "冬", "日", "夜",
}

var clickLetterChars = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "J", "K",
	"L", "M", "N", "P", "Q", "R", "S", "T", "U", "V",
	"W", "X", "Y", "Z",
}

var clickNumberChars = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
}

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

func shuffleInts(arr []int) []int {
	result := make([]int, len(arr))
	perm := rand.Perm(len(arr))
	for i := 0; i < len(arr); i++ {
		result[i] = arr[perm[i]]
	}
	return result
}

func generateClickImageWithBackground(session *CaptchaSession) (string, []ClickPoint, []int, string) {
	session.ImageWidth = 320
	session.ImageHeight = 200
	session.Tolerance = 35

	img := image.NewRGBA(image.Rect(0, 0, session.ImageWidth, session.ImageHeight))

	bgColor := color.RGBA{
		R: uint8(102 + rand.Intn(80)),
		G: uint8(102 + rand.Intn(80)),
		B: uint8(180 + rand.Intn(75)),
		A: 255,
	}
	draw.Draw(img, img.Bounds(), &image.Uniform{bgColor}, image.Point{}, draw.Src)

	for i := 0; i < 50; i++ {
		x := rand.Intn(session.ImageWidth)
		y := rand.Intn(session.ImageHeight)
		r := rand.Intn(3) + 1
		c := color.RGBA{
			R: uint8(255),
			G: uint8(255),
			B: uint8(255),
			A: uint8(30 + rand.Intn(50)),
		}
		for dx := -r; dx <= r; dx++ {
			for dy := -r; dy <= r; dy++ {
				if dx*dx+dy*dy <= r*r {
					px, py := x+dx, y+dy
					if px >= 0 && px < session.ImageWidth && py >= 0 && py < session.ImageHeight {
						img.Set(px, py, c)
					}
				}
			}
		}
	}

	margin := 40
	availableWidth := session.ImageWidth - 2*margin
	availableHeight := session.ImageHeight - 2*margin
	spacingX := availableWidth / session.MaxPoints

	targetPoints := make([]ClickPoint, session.MaxPoints)
	displayChars := make([]string, session.MaxPoints)

	for i := 0; i < session.MaxPoints; i++ {
		targetPoints[i].Index = i
		targetPoints[i].X = margin + spacingX/2 + i*spacingX + rand.Intn(spacingX/2)
		targetPoints[i].Y = margin + availableHeight/2 + rand.Intn(availableHeight/2)

		if session.ImageWidth > 200 {
			targetPoints[i].X = clampValue(targetPoints[i].X, margin, session.ImageWidth-margin-40)
		}
		if session.ImageHeight > 150 {
			targetPoints[i].Y = clampValue(targetPoints[i].Y, margin, session.ImageHeight-margin-30)
		}
	}

	for i, pt := range targetPoints {
		displayChars[i] = getCharForIndex(i, session.Mode)
		drawCharOnImage(img, pt.X, pt.Y, displayChars[i])
	}

	session.TargetPoints = targetPoints

	hintOrder := make([]int, session.MaxPoints)
	for i := 0; i < session.MaxPoints; i++ {
		hintOrder[i] = i
	}

	if session.AllowShuffle && rand.Float32() > 0.5 {
		hintOrder = shuffleInts(hintOrder)
	}

	session.HintOrder = hintOrder

	hintParts := make([]string, session.MaxPoints)
	for i, idx := range hintOrder {
		hintParts[i] = displayChars[idx]
	}
	session.Hint = "点击: " + strings.Join(hintParts, " → ")

	session.Points = make([][2]int, session.MaxPoints)
	for i, pt := range targetPoints {
		session.Points[i] = [2]int{pt.X, pt.Y}
	}

	base64Data := imageToBase64(img)
	return "data:image/png;base64," + base64Data, targetPoints, hintOrder, session.Hint
}

func getCharForIndex(index int, mode CaptchaMode) string {
	switch mode {
	case ModeNumber:
		return clickNumberChars[rand.Intn(len(clickNumberChars))]
	case ModeLetter:
		return clickLetterChars[rand.Intn(len(clickLetterChars))]
	case ModeChinese:
		return clickChineseChars[rand.Intn(len(clickChineseChars))]
	case ModeMixed:
		switch rand.Intn(3) {
		case 0:
			return clickNumberChars[rand.Intn(len(clickNumberChars))]
		case 1:
			return clickLetterChars[rand.Intn(len(clickLetterChars))]
		default:
			return clickChineseChars[rand.Intn(len(clickChineseChars))]
		}
	default:
		return clickNumberChars[rand.Intn(len(clickNumberChars))]
	}
}

func drawCharOnImage(img *image.RGBA, x, y int, char string) {
	circleRadius := 20
	circleColor := color.RGBA{
		R: 255,
		G: 255,
		B: 255,
		A: 220,
	}
	for dx := -circleRadius; dx <= circleRadius; dx++ {
		for dy := -circleRadius; dy <= circleRadius; dy++ {
			if dx*dx+dy*dy <= circleRadius*circleRadius {
				px, py := x+dx, y+dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, circleColor)
				}
			}
		}
	}

	borderRadius := circleRadius
	borderColor := color.RGBA{
		R: 50,
		G: 50,
		B: 100,
		A: 255,
	}
	for dx := -borderRadius; dx <= borderRadius; dx++ {
		for dy := -borderRadius; dy <= borderRadius; dy++ {
			distSq := dx*dx + dy*dy
			if distSq <= (borderRadius+2)*(borderRadius+2) && distSq >= (borderRadius-2)*(borderRadius-2) {
				px, py := x+dx, y+dy
				if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
					img.Set(px, py, borderColor)
				}
			}
		}
	}
}

func clampValue(val, min, max int) int {
	if val < min {
		return min
	}
	if val > max {
		return max
	}
	return val
}

func imageToBase64(img image.Image) string {
	var buf bytes.Buffer
	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	_ = encoder.Encode(&buf, img)
	return base64.StdEncoding.EncodeToString(buf.Bytes())
}

func generateSliderImage() (string, int, int) {
	targetX := 150 + rand.Intn(100)
	targetY := 50 + rand.Intn(100)

	svg := fmt.Sprintf(`<svg xmlns="http://www.w3.org/2000/svg" width="360" height="220">
		<defs>
			<linearGradient id="bg" x1="0%%" y1="0%%" x2="100%%" y2="100%%">
				<stop offset="0%%" style="stop-color:#667eea"/>
				<stop offset="100%%" style="stop-color:#764ba2"/>
			</linearGradient>
		</defs>
		<rect width="100%%" height="100%%" fill="url(#bg)"/>
		<text x="180" y="40" text-anchor="middle" fill="white" font-size="18" font-family="Arial">
			拖动滑块完成验证
		</text>
		<rect x="%d" y="%d" width="50" height="50" fill="rgba(255,255,255,0.25)" stroke="white" stroke-width="2"/>
	</svg>`, targetX, targetY)

	encoded := base64.StdEncoding.EncodeToString([]byte(svg))
	return "data:image/svg+xml;base64," + encoded, targetX, targetY
}

func GetSliderCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	imageURL, targetX, targetY := generateSliderImage()

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
		"image_url":  imageURL,
		"puzzle_y":   targetY,
	})
}

func GetClickCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	modeStr := c.DefaultQuery("mode", "number")
	shuffleStr := c.DefaultQuery("shuffle", "true")
	maxPointsStr := c.DefaultQuery("points", "3")

	var mode CaptchaMode
	switch modeStr {
	case "letter":
		mode = ModeLetter
	case "chinese":
		mode = ModeChinese
	case "mixed":
		mode = ModeMixed
	default:
		mode = ModeNumber
	}

	allowShuffle := shuffleStr == "true"

	maxPoints := 3
	fmt.Sscanf(maxPointsStr, "%d", &maxPoints)
	if maxPoints < 2 {
		maxPoints = 2
	}
	if maxPoints > 6 {
		maxPoints = 6
	}

	session := &CaptchaSession{
		ID:           sessionID,
		Type:         "click",
		Mode:         mode,
		MaxPoints:    maxPoints,
		AllowShuffle: allowShuffle,
		CreatedAt:    time.Now(),
		ImageSeed:    time.Now().UnixNano(),
	}

	imageURL, _, hintOrder, hint := generateClickImageWithBackground(session)

	sessionMutex.Lock()
	captchaSessions[sessionID] = session
	sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"session_id":   sessionID,
		"image_url":    imageURL,
		"hint":         hint,
		"hint_order":   hintOrder,
		"max_points":   maxPoints,
		"mode":         string(mode),
		"allow_shuffle": allowShuffle,
	})
}

type VerifyRequest struct {
	SessionID     string                  `json:"session_id" binding:"required"`
	Type          string                  `json:"type" binding:"required"`
	X             int                     `json:"x"`
	Y             int                     `json:"y"`
	Points        [][2]int                `json:"points"`
	ClickSequence []int                   `json:"click_sequence"`
	BehaviorData  []BehaviorDataPoint     `json:"behavior_data"`
	ApplicationID uint                    `json:"application_id"`
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
	var failReason string

	if req.Type == "slider" {
		tolerance := 10
		captchaSuccess = intAbs(req.X-session.TargetX) <= tolerance && intAbs(req.Y-session.TargetY) <= tolerance
		if !captchaSuccess {
			failReason = fmt.Sprintf("滑块位置偏差过大: 期望(%d,%d), 实际(%d,%d), 容差(%d)",
				session.TargetX, session.TargetY, req.X, req.Y, tolerance)
		}
	} else if req.Type == "click" {
		captchaSuccess, failReason = verifyClickPoints(session, req)
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
		RiskScore:     riskScore,
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

	response := gin.H{
		"success":      finalSuccess,
		"message":      message,
		"risk_score":   riskScore,
		"captcha_pass": captchaSuccess,
	}

	if !captchaSuccess && failReason != "" {
		response["fail_reason"] = failReason
	}

	c.JSON(http.StatusOK, response)
}

func verifyClickPoints(session *CaptchaSession, req VerifyRequest) (bool, string) {
	if len(req.Points) == 0 {
		return false, "未提供点击坐标"
	}

	clickCount := len(req.Points)
	expectedCount := session.MaxPoints

	if clickCount != expectedCount {
		return false, fmt.Sprintf("点击数量不匹配: 期望%d个点, 实际点击%d个点",
			expectedCount, clickCount)
	}

	tolerance := session.Tolerance
	if tolerance <= 0 {
		tolerance = 35
	}

	expectedOrder := session.HintOrder
	if expectedOrder == nil || len(expectedOrder) == 0 {
		expectedOrder = make([]int, session.MaxPoints)
		for i := 0; i < session.MaxPoints; i++ {
			expectedOrder[i] = i
		}
	}

	matchedIndices := make([]int, clickCount)
	usedTargets := make([]bool, session.MaxPoints)
	_ = usedTargets

	for clickIdx := 0; clickIdx < clickCount; clickIdx++ {
		clickX := req.Points[clickIdx][0]
		clickY := req.Points[clickIdx][1]

		found := false
		for targetIdx := 0; targetIdx < session.MaxPoints; targetIdx++ {
			if usedTargets[targetIdx] {
				continue
			}

			targetX := session.TargetPoints[targetIdx].X
			targetY := session.TargetPoints[targetIdx].Y

			dx := float64(clickX - targetX)
			dy := float64(clickY - targetY)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance <= float64(tolerance) {
				matchedIndices[clickIdx] = targetIdx
				usedTargets[targetIdx] = true
				found = true
				break
			}
		}

		if !found {
			return false, fmt.Sprintf("点击位置(%d,%d)无法匹配任何目标点，容差范围%d",
				clickX, clickY, tolerance)
		}
	}

	if len(req.ClickSequence) > 0 {
		if len(req.ClickSequence) != clickCount {
			return false, fmt.Sprintf("点击时序长度不匹配: 提供%d个时序, 实际%d个点击",
				len(req.ClickSequence), clickCount)
		}

		for i := 0; i < clickCount-1; i++ {
			firstIdx := req.ClickSequence[i]
			secondIdx := req.ClickSequence[i+1]

			if firstIdx < 0 || firstIdx >= clickCount || secondIdx < 0 || secondIdx >= clickCount {
				return false, "点击时序索引无效"
			}

			if usedTargets[firstIdx] && usedTargets[secondIdx] {
				firstTarget := matchedIndices[firstIdx]
				secondTarget := matchedIndices[secondIdx]

				firstExpectedPos := -1
				secondExpectedPos := -1
				for j, expectedIdx := range expectedOrder {
					if expectedIdx == firstTarget {
						firstExpectedPos = j
					}
					if expectedIdx == secondTarget {
						secondExpectedPos = j
					}
				}

				if firstExpectedPos > secondExpectedPos {
					return false, fmt.Sprintf("点击顺序错误: 期望按%s顺序点击",
						formatHintOrder(expectedOrder))
				}
			}
		}
	} else {
		clickToTarget := make([]int, clickCount)
		for i := 0; i < clickCount; i++ {
			clickToTarget[i] = matchedIndices[i]
		}

		for i := 0; i < clickCount-1; i++ {
			for j := i + 1; j < clickCount; j++ {
				if clickToTarget[i] > clickToTarget[j] {
					clickToTarget[i], clickToTarget[j] = clickToTarget[j], clickToTarget[i]
				}
			}
		}

		sortedByExpected := make([]int, session.MaxPoints)
		for i, expectedIdx := range expectedOrder {
			sortedByExpected[i] = expectedIdx
		}
		for i := 0; i < session.MaxPoints-1; i++ {
			for j := i + 1; j < session.MaxPoints; j++ {
				if sortedByExpected[i] > sortedByExpected[j] {
					sortedByExpected[i], sortedByExpected[j] = sortedByExpected[j], sortedByExpected[i]
				}
			}
		}

		for i := 0; i < session.MaxPoints; i++ {
			if clickToTarget[i] != sortedByExpected[i] {
				return false, fmt.Sprintf("点击顺序不符合要求，期望按%s顺序点击",
					formatHintOrder(expectedOrder))
			}
		}
	}

	return true, ""
}

func formatHintOrder(order []int) string {
	if len(order) == 0 {
		return ""
	}
	parts := make([]string, len(order))
	for i, idx := range order {
		parts[i] = fmt.Sprintf("%d", idx+1)
	}
	return strings.Join(parts, "→")
}

func intAbs(x int) int {
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
