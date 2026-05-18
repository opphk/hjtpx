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
	"os"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

type CaptchaMode string

const (
	ModeNumber  CaptchaMode = "number"
	ModeLetter  CaptchaMode = "letter"
	ModeChinese CaptchaMode = "chinese"
	ModeMixed   CaptchaMode = "mixed"
	ModeIcon    CaptchaMode = "icon"
)

type ClickPoint struct {
	X     int `json:"x"`
	Y     int `json:"y"`
	Index int `json:"index"`
}

type CaptchaSession struct {
	ID           string
	Type         string
	Mode         CaptchaMode
	TargetPoints []ClickPoint
	HintOrder    []int
	AllowShuffle bool
	Points       [][2]int
	Hint         string
	MaxPoints    int
	CreatedAt    time.Time
	Tolerance    int
	ImageWidth   int
	ImageHeight  int
	ImageSeed    int64
	TargetX      int
	TargetY      int
	Language     string
	SmartTarget  bool
}

var (
	captchaSessions = make(map[string]*CaptchaSession)
	sessionMutex    sync.RWMutex
	behaviorService = service.NewBehaviorAnalysisService()
)

var (
	chineseFont     *sfnt.Font
	chineseFontData []byte
	fontLoadOnce    sync.Once
	fontLoadError   error
)

var clickChineseChars = []string{
	"中", "国", "人", "民", "友", "好", "太", "阳", "月", "亮",
	"星", "辰", "海", "洋", "山", "川", "河", "流", "风", "雨",
	"雪", "云", "花", "草", "树", "木", "林", "森", "天", "地",
	"东", "西", "南", "北", "春", "夏", "秋", "冬", "日", "夜",
	"红", "蓝", "绿", "黄", "紫", "橙", "青", "白", "黑", "灰",
	"书", "笔", "墨", "纸", "画", "琴", "棋", "歌", "舞", "诗",
	"茶", "酒", "米", "面", "肉", "鱼", "蛋", "果", "蔬", "豆",
	"牛", "羊", "马", "猪", "鸡", "犬", "兔", "龙", "蛇", "虎",
	"鸟", "鱼", "虫", "花", "树", "草", "竹", "松", "梅", "兰",
	"菊", "荷", "桃", "梨", "杏", "梅", "枣", "桂", "榴", "荔",
	"江", "河", "湖", "海", "泉", "溪", "波", "浪", "潮", "涛",
	"爱", "恨", "喜", "怒", "哀", "乐", "思", "念", "梦", "醒",
	"知", "识", "学", "问", "书", "读", "写", "算", "数", "理",
}

var clickLetterChars = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "J", "K",
	"L", "M", "N", "P", "Q", "R", "S", "T", "U", "V",
	"W", "X", "Y", "Z",
}

var clickNumberChars = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
}

type IconType string

const (
	IconCircle   IconType = "circle"
	IconSquare   IconType = "square"
	IconTriangle IconType = "triangle"
	IconStar     IconType = "star"
	IconDiamond  IconType = "diamond"
	IconHeart    IconType = "heart"
	IconArrow    IconType = "arrow"
	IconCross    IconType = "cross"
	IconMoon     IconType = "moon"
	IconRing     IconType = "ring"
)

var clickIcons = []IconType{
	IconCircle, IconSquare, IconTriangle, IconStar, IconDiamond,
	IconHeart, IconArrow, IconCross, IconMoon, IconRing,
}

var iconNames = map[IconType]string{
	IconCircle:   "圆形",
	IconSquare:   "方形",
	IconTriangle: "三角形",
	IconStar:     "星形",
	IconDiamond:  "菱形",
	IconHeart:    "心形",
	IconArrow:    "箭头",
	IconCross:    "十字",
	IconMoon:     "月牙",
	IconRing:     "圆环",
}

func init() {
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

func getHintPrefix(mode CaptchaMode, language string) string {
	if language == "zh" || language == "zh-CN" {
		switch mode {
		case ModeNumber:
			return "依次点击: "
		case ModeLetter:
			return "依次点击字母: "
		case ModeChinese:
			return "依次点击汉字: "
		case ModeIcon:
			return "依次点击图标: "
		case ModeMixed:
			return "依次点击: "
		default:
			return "依次点击: "
		}
	}
	switch mode {
	case ModeNumber:
		return "Click in order: "
	case ModeLetter:
		return "Click letters: "
	case ModeChinese:
		return "Click Chinese characters: "
	case ModeIcon:
		return "Click icons: "
	case ModeMixed:
		return "Click in order: "
	default:
		return "Click in order: "
	}
}

func getArrowSeparator(language string) string {
	if language == "zh" || language == "zh-CN" {
		return " → "
	}
	return " → "
}

func getIconNameLocalized(icon IconType, language string) string {
	if language == "zh" || language == "zh-CN" {
		if name, ok := iconNames[icon]; ok {
			return name
		}
	}
	iconENNames := map[IconType]string{
		IconCircle:   "Circle",
		IconSquare:   "Square",
		IconTriangle: "Triangle",
		IconStar:     "Star",
		IconDiamond:  "Diamond",
		IconHeart:    "Heart",
		IconArrow:    "Arrow",
		IconCross:    "Cross",
		IconMoon:     "Moon",
		IconRing:     "Ring",
	}
	if name, ok := iconENNames[icon]; ok {
		return name
	}
	return string(icon)
}

func generateSmartHint(session *CaptchaSession, displayChars []string, language string) string {
	prefix := getHintPrefix(session.Mode, language)
	arrow := getArrowSeparator(language)
	
	parts := make([]string, len(session.HintOrder))
	for i, idx := range session.HintOrder {
		if session.Mode == ModeIcon {
			parts[i] = getIconNameLocalized(IconType(displayChars[idx]), language)
		} else {
			parts[i] = displayChars[idx]
		}
	}
	
	return prefix + strings.Join(parts, arrow)
}

func shuffleInts(arr []int) []int {
	result := make([]int, len(arr))
	perm := rand.Perm(len(arr))
	for i := 0; i < len(arr); i++ {
		result[i] = arr[perm[i]]
	}
	return result
}

func loadChineseFont() error {
	fontLoadOnce.Do(func() {
		chineseFontData, fontLoadError = os.ReadFile("/usr/share/fonts/truetype/wqy/wqy-microhei.ttc")
		if fontLoadError != nil {
			return
		}
		var coll *sfnt.Collection
		coll, fontLoadError = sfnt.ParseCollection(chineseFontData)
		if fontLoadError != nil {
			return
		}
		chineseFont, fontLoadError = coll.Font(0)
	})
	return fontLoadError
}

func renderCharToMask(char rune, size int) *image.Alpha {
	if err := loadChineseFont(); err != nil {
		return nil
	}
	var buf sfnt.Buffer
	idx, err := chineseFont.GlyphIndex(&buf, char)
	if err != nil || idx == 0 {
		return nil
	}
	ppem := fixed.I(size)
	segs, err := chineseFont.LoadGlyph(&buf, idx, ppem, nil)
	if err != nil {
		return nil
	}
	var minFX, minFY, maxFX, maxFY fixed.Int26_6
	first := true
	for _, seg := range segs {
		for _, arg := range seg.Args {
			if first {
				minFX, maxFX = arg.X, arg.X
				minFY, maxFY = arg.Y, arg.Y
				first = false
			} else {
				if arg.X < minFX {
					minFX = arg.X
				}
				if arg.X > maxFX {
					maxFX = arg.X
				}
				if arg.Y < minFY {
					minFY = arg.Y
				}
				if arg.Y > maxFY {
					maxFY = arg.Y
				}
			}
		}
	}
	padding := fixed.I(4)
	glyphW := (maxFX - minFX + padding*2).Ceil()
	glyphH := (maxFY - minFY + padding*2).Ceil()
	if glyphW < 4 {
		glyphW = 4
	}
	if glyphH < 4 {
		glyphH = 4
	}
	r := vector.NewRasterizer(glyphW, glyphH)
	r.DrawOp = draw.Src
	offsetX := -minFX + padding
	offsetY := -minFY + padding
	for _, seg := range segs {
		switch seg.Op {
		case sfnt.SegmentOpMoveTo:
			r.MoveTo(
				float32(seg.Args[0].X+offsetX)/64,
				float32(seg.Args[0].Y+offsetY)/64,
			)
		case sfnt.SegmentOpLineTo:
			r.LineTo(
				float32(seg.Args[0].X+offsetX)/64,
				float32(seg.Args[0].Y+offsetY)/64,
			)
		case sfnt.SegmentOpQuadTo:
			r.QuadTo(
				float32(seg.Args[0].X+offsetX)/64,
				float32(seg.Args[0].Y+offsetY)/64,
				float32(seg.Args[1].X+offsetX)/64,
				float32(seg.Args[1].Y+offsetY)/64,
			)
		case sfnt.SegmentOpCubeTo:
			r.CubeTo(
				float32(seg.Args[0].X+offsetX)/64,
				float32(seg.Args[0].Y+offsetY)/64,
				float32(seg.Args[1].X+offsetX)/64,
				float32(seg.Args[1].Y+offsetY)/64,
				float32(seg.Args[2].X+offsetX)/64,
				float32(seg.Args[2].Y+offsetY)/64,
			)
		}
	}
	alpha := image.NewAlpha(image.Rect(0, 0, glyphW, glyphH))
	r.Draw(alpha, alpha.Bounds(), image.Opaque, image.Point{})
	return alpha
}

func renderCharToRGBA(char rune, size int, textColor color.RGBA) *image.RGBA {
	alpha := renderCharToMask(char, size)
	if alpha == nil {
		return nil
	}
	b := alpha.Bounds()
	dst := image.NewRGBA(b)
	for y := b.Min.Y; y < b.Max.Y; y++ {
		for x := b.Min.X; x < b.Max.X; x++ {
			a := alpha.AlphaAt(x, y).A
			if a > 0 {
				c := textColor
				c.A = a
				dst.Set(x, y, c)
			}
		}
	}
	return dst
}

func rotateImageRGBA(src *image.RGBA, angleDeg float64) *image.RGBA {
	if angleDeg == 0 {
		dst := image.NewRGBA(src.Bounds())
		draw.Draw(dst, dst.Bounds(), src, src.Bounds().Min, draw.Over)
		return dst
	}
	rad := angleDeg * math.Pi / 180
	b := src.Bounds()
	w := b.Dx()
	h := b.Dy()
	cos := math.Cos(rad)
	sin := math.Sin(rad)
	corners := [4][2]float64{
		{0, 0},
		{float64(w), 0},
		{float64(w), float64(h)},
		{0, float64(h)},
	}
	minX := math.MaxFloat64
	minY := math.MaxFloat64
	maxX := -math.MaxFloat64
	maxY := -math.MaxFloat64
	for _, c := range corners {
		rx := c[0]*cos - c[1]*sin
		ry := c[0]*sin + c[1]*cos
		if rx < minX {
			minX = rx
		}
		if rx > maxX {
			maxX = rx
		}
		if ry < minY {
			minY = ry
		}
		if ry > maxY {
			maxY = ry
		}
	}
	newW := int(math.Ceil(maxX - minX))
	newH := int(math.Ceil(maxY - minY))
	if newW < 1 {
		newW = 1
	}
	if newH < 1 {
		newH = 1
	}
	dst := image.NewRGBA(image.Rect(0, 0, newW, newH))
	cx := float64(w) / 2
	cy := float64(h) / 2
	dcx := float64(newW) / 2
	dcy := float64(newH) / 2
	for dy := 0; dy < newH; dy++ {
		for dx := 0; dx < newW; dx++ {
			px := float64(dx) - dcx
			py := float64(dy) - dcy
			sx := px*cos + py*sin + cx
			sy := -px*sin + py*cos + cy
			sxInt := int(math.Round(sx))
			syInt := int(math.Round(sy))
			if sxInt >= 0 && sxInt < w && syInt >= 0 && syInt < h {
				c := src.RGBAAt(sxInt, syInt)
				if c.A > 0 {
					dst.SetRGBA(dx, dy, c)
				}
			}
		}
	}
	return dst
}

func randomVibrantColor() color.RGBA {
	hue := rand.Intn(360)
	saturation := 0.6 + rand.Float64()*0.35
	value := 0.5 + rand.Float64()*0.4
	r, g, b := hsvToRGB(hue, saturation, value)
	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 200 + uint8(rand.Intn(56)),
	}
}

func hsvToRGB(h int, s, v float64) (float64, float64, float64) {
	hf := float64(h) / 60.0
	i := int(hf)
	f := hf - float64(i)
	p := v * (1 - s)
	q := v * (1 - s*f)
	t := v * (1 - s*(1-f))
	switch i % 6 {
	case 0:
		return v, t, p
	case 1:
		return q, v, p
	case 2:
		return p, v, t
	case 3:
		return p, q, v
	case 4:
		return t, p, v
	default:
		return v, p, q
	}
}

func drawGradientBackground(img *image.RGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	r1 := uint8(180 + rand.Intn(60))
	g1 := uint8(180 + rand.Intn(60))
	b1 := uint8(200 + rand.Intn(55))
	r2 := uint8(100 + rand.Intn(80))
	g2 := uint8(120 + rand.Intn(60))
	b2 := uint8(160 + rand.Intn(60))
	for y := 0; y < h; y++ {
		t := float64(y) / float64(h)
		r := uint8(float64(r1)*(1-t) + float64(r2)*t)
		g := uint8(float64(g1)*(1-t) + float64(g2)*t)
		b := uint8(float64(b1)*(1-t) + float64(b2)*t)
		for x := 0; x < w; x++ {
			img.Set(x, y, color.RGBA{R: r, G: g, B: b, A: 255})
		}
	}
}

func addNoiseDots(img *image.RGBA, count int) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	for i := 0; i < count; i++ {
		x := rand.Intn(w)
		y := rand.Intn(h)
		radius := rand.Intn(3) + 1
		c := color.RGBA{
			R: uint8(rand.Intn(256)),
			G: uint8(rand.Intn(256)),
			B: uint8(rand.Intn(256)),
			A: uint8(20 + rand.Intn(60)),
		}
		for dx := -radius; dx <= radius; dx++ {
			for dy := -radius; dy <= radius; dy++ {
				if dx*dx+dy*dy <= radius*radius {
					px, py := x+dx, y+dy
					if px >= 0 && px < w && py >= 0 && py < h {
						img.Set(px, py, c)
					}
				}
			}
		}
	}
}

func addInterferenceLines(img *image.RGBA, count int) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	for i := 0; i < count; i++ {
		lineColor := color.RGBA{
			R: uint8(rand.Intn(256)),
			G: uint8(rand.Intn(256)),
			B: uint8(rand.Intn(256)),
			A: uint8(30 + rand.Intn(50)),
		}
		startY := rand.Intn(h)
		amplitude := 10 + rand.Intn(30)
		frequency := 0.02 + rand.Float64()*0.04
		for x := 0; x < w; x++ {
			y := startY + int(float64(amplitude)*math.Sin(float64(x)*frequency))
			if y >= 0 && y < h {
				img.Set(x, y, lineColor)
			}
			y2 := y + 1
			if y2 >= 0 && y2 < h {
				img.Set(x, y2, lineColor)
			}
		}
	}
}

func addGridLines(img *image.RGBA) {
	w := img.Bounds().Dx()
	h := img.Bounds().Dy()
	gridColor := color.RGBA{
		R: uint8(200 + rand.Intn(56)),
		G: uint8(200 + rand.Intn(56)),
		B: uint8(220 + rand.Intn(36)),
		A: 25,
	}
	stepX := 20 + rand.Intn(20)
	stepY := 20 + rand.Intn(20)
	for x := 0; x < w; x += stepX {
		for yy := 0; yy < h; yy++ {
			img.Set(x, yy, gridColor)
		}
	}
	for y := 0; y < h; y += stepY {
		for xx := 0; xx < w; xx++ {
			img.Set(xx, y, gridColor)
		}
	}
}

func isOverlapping(x, y, size int, placed []image.Rectangle) bool {
	candidate := image.Rect(x-size/2, y-size/2, x+size/2, y+size/2)
	for _, r := range placed {
		overlap := candidate.Intersect(r)
		if overlap.Dx() > 10 && overlap.Dy() > 10 {
			return true
		}
	}
	return false
}

// generateClickImageWithBackground 生成带背景的点击验证码图片
// 该函数生成一个包含多个随机字符/图标的验证码图片，用户需要按照提示顺序点击
// 主要流程：
// 1. 绘制渐变背景并添加干扰元素（噪点、线条、网格）
// 2. 生成目标字符（需要点击的）和干扰字符（不需要点击的）
// 3. 使用Fisher-Yates算法打乱字符位置
// 4. 使用碰撞检测算法确保字符不重叠
// 5. 渲染字符并应用随机旋转
// 6. 计算目标点的精确坐标用于后续验证
func generateClickImageWithBackground(session *CaptchaSession) (string, []ClickPoint, []int, string) {
	session.ImageWidth = 300
	session.ImageHeight = 300
	session.Tolerance = 25

	img := image.NewRGBA(image.Rect(0, 0, session.ImageWidth, session.ImageHeight))

	drawGradientBackground(img)
	addNoiseDots(img, 200)
	addInterferenceLines(img, 4)
	addGridLines(img)

	maxPoints := session.MaxPoints
	totalChars := 6 + rand.Intn(3)
	if maxPoints > totalChars {
		maxPoints = totalChars
	}

	targetChars := make([]string, maxPoints)
	for i := 0; i < maxPoints; i++ {
		targetChars[i] = getCharForIndex(i, session.Mode)
	}

	decoyCount := totalChars - maxPoints
	allChars := make([]string, totalChars)
	for i := 0; i < maxPoints; i++ {
		allChars[i] = targetChars[i]
	}
	for i := 0; i < decoyCount; i++ {
		allChars[maxPoints+i] = getCharForIndex(i+maxPoints+100, session.Mode)
	}

	perm := rand.Perm(totalChars)
	shuffledChars := make([]string, totalChars)
	charPositions := make([]int, totalChars)
	for i, p := range perm {
		shuffledChars[i] = allChars[p]
		charPositions[i] = p
	}

	placedRects := make([]image.Rectangle, 0, totalChars)
	charCenters := make([]struct{ X, Y int }, totalChars)
	charSizes := make([]int, totalChars)

	for i := 0; i < totalChars; i++ {
		charSize := 30 + rand.Intn(20)
		charSizes[i] = charSize
		halfSize := charSize
		maxAttempts := 30
		var cx, cy int
		placed := false
		for attempt := 0; attempt < maxAttempts; attempt++ {
			margin := halfSize/2 + 10
			cx = margin + rand.Intn(session.ImageWidth-2*margin)
			cy = margin + rand.Intn(session.ImageHeight-2*margin)
			if !isOverlapping(cx, cy, halfSize, placedRects) {
				placed = true
				break
			}
		}
		if !placed {
			margin := halfSize/2 + 10
			cx = margin + rand.Intn(session.ImageWidth-2*margin)
			cy = margin + rand.Intn(session.ImageHeight-2*margin)
		}
		charCenters[i] = struct{ X, Y int }{cx, cy}
		placedRects = append(placedRects, image.Rect(cx-halfSize/2, cy-halfSize/2, cx+halfSize/2, cy+halfSize/2))
	}

	for i := 0; i < totalChars; i++ {
		char := []rune(shuffledChars[i])[0]
		charSize := charSizes[i]
		rotation := float64(rand.Intn(40) - 20)
		textColor := randomVibrantColor()

		if session.Mode == ModeIcon {
			iconType := IconType(shuffledChars[i])
			rendered := renderIcon(iconType, charSize, textColor)
			rotated := rotateImageRGBA(rendered, rotation)
			draw.Draw(img, image.Rect(
				charCenters[i].X-rotated.Bounds().Dx()/2,
				charCenters[i].Y-rotated.Bounds().Dy()/2,
				charCenters[i].X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(),
				charCenters[i].Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy(),
			), rotated, image.Point{}, draw.Over)
		} else {
			rendered := renderCharToRGBA(char, charSize, textColor)
			if rendered == nil {
				rendered = image.NewRGBA(image.Rect(0, 0, 10, 10))
				draw.Draw(rendered, rendered.Bounds(), &image.Uniform{textColor}, image.Point{}, draw.Src)
			}
			rotated := rotateImageRGBA(rendered, rotation)
			draw.Draw(img, image.Rect(
				charCenters[i].X-rotated.Bounds().Dx()/2,
				charCenters[i].Y-rotated.Bounds().Dy()/2,
				charCenters[i].X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(),
				charCenters[i].Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy(),
			), rotated, image.Point{}, draw.Over)
		}
	}

	targetPoints := make([]ClickPoint, maxPoints)
	usedTargets := make([]bool, maxPoints)
	displayChars := make([]string, maxPoints)

	for _, origIdx := range perm[:maxPoints] {
		for j := 0; j < maxPoints; j++ {
			if usedTargets[j] {
				continue
			}
			ci := -1
			for k := 0; k < totalChars; k++ {
				if charPositions[k] == j && k == origIdx {
					ci = k
					break
				}
			}
			if ci >= 0 {
				targetPoints[j].Index = j
				targetPoints[j].X = charCenters[ci].X
				targetPoints[j].Y = charCenters[ci].Y
				displayChars[j] = targetChars[j]
				usedTargets[j] = true
				break
			}
		}
	}

	fallbackIdx := 0
	for j := 0; j < maxPoints; j++ {
		if !usedTargets[j] {
			for fallbackIdx < totalChars {
				ci := -1
				for k := 0; k < totalChars; k++ {
					if charPositions[k] == j && k < totalChars {
						ci = k
						break
					}
				}
				if ci >= 0 {
					targetPoints[j].Index = j
					targetPoints[j].X = charCenters[ci].X
					targetPoints[j].Y = charCenters[ci].Y
					displayChars[j] = targetChars[j]
					usedTargets[j] = true
					fallbackIdx++
					break
				}
				fallbackIdx++
			}
		}
	}

	session.TargetPoints = targetPoints

	hintOrder := make([]int, maxPoints)
	for i := 0; i < maxPoints; i++ {
		hintOrder[i] = i
	}

	if session.AllowShuffle && rand.Float32() > 0.5 {
		hintOrder = shuffleInts(hintOrder)
	}

	session.HintOrder = hintOrder

	session.Hint = generateSmartHint(session, displayChars, session.Language)

	session.Points = make([][2]int, maxPoints)
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
	case ModeIcon:
		return string(clickIcons[rand.Intn(len(clickIcons))])
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

func renderIcon(iconType IconType, size int, iconColor color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	centerX, centerY := size/2, size/2
	halfSize := size / 3

	switch iconType {
	case IconCircle:
		drawFilledCircle(img, centerX, centerY, halfSize, iconColor)
	case IconSquare:
		drawFilledRect(img, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, iconColor)
	case IconTriangle:
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				relY := float64(y-centerY) / float64(halfSize)
				relX := float64(x-centerX) / float64(halfSize)
				if relY >= -1 && relY <= 0 && math.Abs(relX) <= -relY {
					img.Set(x, y, iconColor)
				}
			}
		}
	case IconStar:
		drawStar(img, centerX, centerY, halfSize, iconColor)
	case IconDiamond:
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				relX := math.Abs(float64(x - centerX))
				relY := math.Abs(float64(y - centerY))
				if relX+relY <= float64(halfSize) {
					img.Set(x, y, iconColor)
				}
			}
		}
	case IconHeart:
		drawHeart(img, centerX, centerY, halfSize, iconColor)
	case IconArrow:
		drawArrow(img, centerX, centerY, halfSize, iconColor)
	case IconCross:
		thickness := halfSize / 2
		if thickness < 2 {
			thickness = 2
		}
		drawFilledRect(img, centerX-thickness, centerY-halfSize, thickness*2, halfSize*2, iconColor)
		drawFilledRect(img, centerX-halfSize, centerY-thickness, halfSize*2, thickness*2, iconColor)
	case IconMoon:
		drawFilledCircle(img, centerX, centerY, halfSize, iconColor)
		coverColor := color.RGBA{
			R: 220, G: 220, B: 225, A: 255,
		}
		drawFilledCircle(img, centerX+halfSize/3, centerY-halfSize/4, halfSize*3/4, coverColor)
	case IconRing:
		outerR := halfSize
		innerR := halfSize * 2 / 3
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				dx := float64(x - centerX)
				dy := float64(y - centerY)
				dist := math.Sqrt(dx*dx + dy*dy)
				if dist <= float64(outerR) && dist >= float64(innerR) {
					img.Set(x, y, iconColor)
				}
			}
		}
	}

	return img
}

func drawStar(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	points := make([][2]float64, 10)
	for i := 0; i < 10; i++ {
		angle := float64(i)*math.Pi/5 - math.Pi/2
		r := float64(radius)
		if i%2 == 1 {
			r = float64(radius) * 0.4
		}
		points[i] = [2]float64{
			float64(cx) + r*math.Cos(angle),
			float64(cy) + r*math.Sin(angle),
		}
	}
	fillPolygon(img, points, c)
}

func drawHeart(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	for y := -radius; y <= radius; y++ {
		for x := -radius; x <= radius; x++ {
			dx := float64(x) / float64(radius)
			dy := float64(y) / float64(radius)
			heart := dx*dx + (dy-math.Sqrt(math.Abs(dx)))*(dy-math.Sqrt(math.Abs(dx))) - 1
			if heart <= 0 {
				px, py := cx+x, cy+y
				if px >= 0 && px < 300 && py >= 0 && py < 300 {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func drawArrow(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	shaftLen := radius * 3 / 4
	shaftThick := radius / 4
	if shaftThick < 2 {
		shaftThick = 2
	}
	headSize := radius / 2

	drawFilledRect(img, cx-shaftLen, cy-shaftThick/2, shaftLen, shaftThick, c)

	for y := 0; y < headSize*2; y++ {
		for x := 0; x < headSize; x++ {
			relY := float64(y-headSize) / float64(headSize)
			relX := float64(x) / float64(headSize)
			if math.Abs(relY) <= relX {
				px, py := cx+x, cy+y-headSize
				if px >= 0 && px < 300 && py >= 0 && py < 300 {
					img.Set(px, py, c)
				}
			}
		}
	}
}

func fillPolygon(img *image.RGBA, points [][2]float64, c color.RGBA) {
	if len(points) < 3 {
		return
	}
	minY, maxY := points[0][1], points[0][1]
	for _, p := range points {
		if p[1] < minY {
			minY = p[1]
		}
		if p[1] > maxY {
			maxY = p[1]
		}
	}
	for y := int(minY); y <= int(maxY); y++ {
		intersections := make([]float64, 0)
		for i := 0; i < len(points); i++ {
			j := (i + 1) % len(points)
			y1, y2 := points[i][1], points[j][1]
			if (y1 <= float64(y) && y2 > float64(y)) || (y2 <= float64(y) && y1 > float64(y)) {
				t := (float64(y) - y1) / (y2 - y1)
				x := points[i][0] + t*(points[j][0]-points[i][0])
				intersections = append(intersections, x)
			}
		}
		for i := 0; i < len(intersections)-1; i++ {
			for j := i + 1; j < len(intersections); j++ {
				if intersections[i] > intersections[j] {
					intersections[i], intersections[j] = intersections[j], intersections[i]
				}
			}
		}
		for i := 0; i < len(intersections)-1; i += 2 {
			x1 := int(intersections[i])
			x2 := int(intersections[i+1])
			for x := x1; x <= x2; x++ {
				if x >= 0 && x < 300 && y >= 0 && y < 300 {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func getIconName(iconStr string) string {
	iconType := IconType(iconStr)
	if name, ok := iconNames[iconType]; ok {
		return name
	}
	return iconStr
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

func clampU8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func drawFilledCircle(img *image.RGBA, cx, cy, radius int, c color.Color) {
	for dy := -radius; dy <= radius; dy++ {
		for dx := -radius; dx <= radius; dx++ {
			if dx*dx+dy*dy <= radius*radius {
				x, y := cx+dx, cy+dy
				if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
					img.Set(x, y, c)
				}
			}
		}
	}
}

func drawFilledRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				img.Set(px, py, c)
			}
		}
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
	dx := x2 - x1
	dy := y2 - y1
	steps := int(math.Sqrt(float64(dx*dx + dy*dy)))
	if steps < 1 {
		steps = 1
	}
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		x := x1 + int(float64(dx)*t+0.5)
		y := y1 + int(float64(dy)*t+0.5)
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
	}
}

func drawBezier(img *image.RGBA, x0, y0, x1, y1, x2, y2 int, c color.Color) {
	steps := 60
	for i := 0; i <= steps; i++ {
		t := float64(i) / float64(steps)
		mt := 1.0 - t
		x := int(mt*mt*float64(x0) + 2.0*mt*t*float64(x1) + t*t*float64(x2) + 0.5)
		y := int(mt*mt*float64(y0) + 2.0*mt*t*float64(y1) + t*t*float64(y2) + 0.5)
		if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
			img.Set(x, y, c)
		}
	}
}

func isInPuzzlePiece(x, y, pieceSize, radius int) bool {
	if y < 0 || y >= pieceSize {
		return false
	}
	midY := pieceSize / 2

	if y >= midY-radius && y <= midY+radius {
		dy := y - midY
		leftBoundary := int(math.Sqrt(float64(radius*radius - dy*dy)))
		if x < leftBoundary {
			return false
		}
	} else if x < 0 {
		return false
	}

	if y >= midY-radius && y <= midY+radius {
		dy := y - midY
		rightBoundary := pieceSize + int(math.Sqrt(float64(radius*radius-dy*dy)))
		if x > rightBoundary {
			return false
		}
	} else if x > pieceSize {
		return false
	}

	return true
}

func addPuzzleShadow(pieceImg *image.RGBA, pieceSize, radius int) {
	bounds := pieceImg.Bounds()
	result := image.NewRGBA(bounds)

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			p := pieceImg.RGBAAt(x, y)
			if p.A > 0 {
				sx, sy := x+2, y+2
				if sx < bounds.Dx() && sy < bounds.Dy() {
					result.Set(sx, sy, color.RGBA{0, 0, 0, 100})
				}
			}
		}
	}

	for y := 0; y < bounds.Dy(); y++ {
		for x := 0; x < bounds.Dx(); x++ {
			p := pieceImg.RGBAAt(x, y)
			if p.A > 0 {
				result.Set(x, y, p)
			}
		}
	}

	copy(pieceImg.Pix, result.Pix)
}

func addCutoutBorder(img *image.RGBA, targetX, targetY, pieceSize, radius int) {
	borderColor := color.RGBA{255, 255, 255, 200}

	for y := 0; y < pieceSize; y++ {
		for x := 0; x < pieceSize+radius; x++ {
			if !isInPuzzlePiece(x, y, pieceSize, radius) {
				continue
			}
			isBorder := false
			for dy := -1; dy <= 1 && !isBorder; dy++ {
				for dx := -1; dx <= 1 && !isBorder; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if !isInPuzzlePiece(x+dx, y+dy, pieceSize, radius) {
						isBorder = true
					}
				}
			}
			if isBorder {
				absX, absY := targetX+x, targetY+y
				if absX >= 0 && absX < img.Bounds().Dx() && absY >= 0 && absY < img.Bounds().Dy() {
					img.Set(absX, absY, borderColor)
				}
				for d := 1; d <= 3; d++ {
					absX2, absY2 := targetX+x+d, targetY+y+d
					if absX2 >= 0 && absX2 < img.Bounds().Dx() && absY2 >= 0 && absY2 < img.Bounds().Dy() {
						orig := img.RGBAAt(absX2, absY2)
						factor := 100 - d*12
						if factor < 60 {
							factor = 60
						}
						img.Set(absX2, absY2, color.RGBA{
							clampU8(int(orig.R) * factor / 100),
							clampU8(int(orig.G) * factor / 100),
							clampU8(int(orig.B) * factor / 100),
							255,
						})
					}
				}
			}
		}
	}
}

func generateSliderCaptchaImages() (string, string, int, int) {
	width := 360
	height := 220
	pieceSize := 50
	bumpRadius := 8 + rand.Intn(5)

	img := image.NewRGBA(image.Rect(0, 0, width, height))

	r1, g1, b1 := rand.Intn(80)+40, rand.Intn(80)+40, rand.Intn(80)+120
	r2, g2, b2 := rand.Intn(80)+120, rand.Intn(80)+40, rand.Intn(80)+40
	r3, g3, b3 := rand.Intn(60)+80, rand.Intn(60)+120, rand.Intn(60)+40

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			ratio1 := float64(x+y) / float64(width+height)
			ratio2 := float64(x) / float64(width)
			r := uint8(float64(r1)*(1-ratio1)*0.6 + float64(r2)*ratio1*0.6 + float64(r3)*ratio2*0.4)
			g := uint8(float64(g1)*(1-ratio1)*0.6 + float64(g2)*ratio1*0.6 + float64(g3)*ratio2*0.4)
			b := uint8(float64(b1)*(1-ratio1)*0.6 + float64(b2)*ratio1*0.6 + float64(b3)*ratio2*0.4)
			img.Set(x, y, color.RGBA{r, g, b, 255})
		}
	}

	for i := 0; i < 25; i++ {
		c := color.RGBA{
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(rand.Intn(200)),
			uint8(25 + rand.Intn(55)),
		}
		switch rand.Intn(4) {
		case 0:
			drawFilledCircle(img, rand.Intn(width), rand.Intn(height), 8+rand.Intn(35), c)
		case 1:
			drawFilledRect(img, rand.Intn(width), rand.Intn(height), 15+rand.Intn(50), 8+rand.Intn(30), c)
		case 2:
			drawLine(img, rand.Intn(width), rand.Intn(height), rand.Intn(width), rand.Intn(height), c)
		case 3:
			drawBezier(img, rand.Intn(width), rand.Intn(height), rand.Intn(width), rand.Intn(height), rand.Intn(width), rand.Intn(height), c)
		}
	}

	for i := 0; i < 1000; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		noise := rand.Intn(50) - 25
		p := img.RGBAAt(x, y)
		img.Set(x, y, color.RGBA{
			clampU8(int(p.R) + noise),
			clampU8(int(p.G) + noise),
			clampU8(int(p.B) + noise),
			255,
		})
	}

	margin := 30
	maxX := width - pieceSize - bumpRadius - margin
	if maxX <= margin {
		maxX = margin + 1
	}
	targetX := margin + rand.Intn(maxX-margin)
	targetY := margin + rand.Intn(height-pieceSize-2*margin)

	pieceWidth := pieceSize + bumpRadius
	pieceImg := image.NewRGBA(image.Rect(0, 0, pieceWidth, pieceSize))

	for py := 0; py < pieceSize; py++ {
		for px := 0; px < pieceWidth; px++ {
			absX := targetX + px
			absY := targetY + py
			if absX >= 0 && absX < width && absY >= 0 && absY < height {
				if isInPuzzlePiece(px, py, pieceSize, bumpRadius) {
					p := img.RGBAAt(absX, absY)
					pieceImg.Set(px, py, p)
					img.Set(absX, absY, color.RGBA{
						uint8(int(p.R) * 35 / 100),
						uint8(int(p.G) * 35 / 100),
						uint8(int(p.B) * 35 / 100),
						255,
					})
				} else {
					pieceImg.Set(px, py, color.RGBA{0, 0, 0, 0})
				}
			}
		}
	}

	addPuzzleShadow(pieceImg, pieceSize, bumpRadius)
	addCutoutBorder(img, targetX, targetY, pieceSize, bumpRadius)

	imageURL := "data:image/png;base64," + imageToBase64(img)
	puzzleImage := "data:image/png;base64," + imageToBase64(pieceImg)

	return imageURL, puzzleImage, targetX, targetY
}

// GetSliderCaptcha 获取滑动验证码
// @Summary 获取滑动验证码
// @Description 生成并返回一个新的滑动验证码，包含验证码图片和滑动块图片
// @Tags 验证码
// @Accept json
// @Produce json
// @Success 200 {object} map[string]interface{} "成功返回验证码数据"
// @Failure 500 {object} map[string]interface{} "服务器内部错误"
// @Router /api/v1/captcha/slider [get]
func GetSliderCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	imageURL, puzzleImage, targetX, targetY := generateSliderCaptchaImages()

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
		"session_id":   sessionID,
		"image_url":    imageURL,
		"puzzle_image": puzzleImage,
		"target_x":     targetX,
		"target_y":     targetY,
		"puzzle_y":     targetY,
		"puzzle_style": 0,
		"tolerance":    10,
	})
}

// GetClickCaptcha 获取点击验证码
// @Summary 获取点击验证码
// @Description 生成并返回一个点击式验证码，支持多种模式（数字、字母、中文、图标、混合等)
// @Tags 验证码
// @Accept json
// @Produce json
// @Param mode query string false "验证码模式" Enums(number, letter, chinese, mixed, icon)"
// @Param shuffle query string false "是否允许打乱顺序"
// @Param points query int false "点击点数"
// @Param lang query string false "语言设置" Enums(zh-CN, en-US)
// @Success 200 {object} map[string]interface{} "成功返回验证码数据"
// @Router /api/v1/captcha/click [get]
func GetClickCaptcha(c *gin.Context) {
	sessionID := generateSessionID()
	modeStr := c.DefaultQuery("mode", "number")
	shuffleStr := c.DefaultQuery("shuffle", "true")
	maxPointsStr := c.DefaultQuery("points", "3")
	lang := c.DefaultQuery("lang", "en-US")

	var mode CaptchaMode
	switch modeStr {
	case "letter":
		mode = ModeLetter
	case "chinese":
		mode = ModeChinese
	case "mixed":
		mode = ModeMixed
	case "icon":
		mode = ModeIcon
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
		Language:     lang,
		SmartTarget:  true,
	}

	imageURL, _, hintOrder, hint := generateClickImageWithBackground(session)

	sessionMutex.Lock()
	captchaSessions[sessionID] = session
	sessionMutex.Unlock()

	c.JSON(http.StatusOK, gin.H{
		"session_id":    sessionID,
		"image_url":     imageURL,
		"hint":          hint,
		"hint_order":    hintOrder,
		"max_points":    maxPoints,
		"mode":          string(mode),
		"allow_shuffle": allowShuffle,
		"points":        session.Points,
		"language":      lang,
	})
}

// VerifyRequest 验证码验证请求参数
// @Description 验证验证码时发送的请求结构，支持滑动验证码和点击验证码
type VerifyRequest struct {
	SessionID       string              `json:"session_id" binding:"required"`                   // 会话 ID
	Type            string              `json:"type" binding:"required"`                         // 验证码类型: slider 或 click
	X               int                 `json:"x"`                                               // 滑动验证码 X 坐标
	Y               int                 `json:"y"`                                               // 滑动验证码 Y 坐标
	Points          [][2]int            `json:"points"`                                          // 点击验证码坐标数组
	ClickSequence   []int               `json:"click_sequence"`                                  // 点击顺序
	BehaviorData    []BehaviorDataPoint `json:"behavior_data"`                                   // 行为分析数据
	SpeedData       json.RawMessage     `json:"speed_data,omitempty" swaggerignore:"true"`       // 速度数据 (内部使用)
	ApplicationID   uint                `json:"application_id"`                                  // 应用 ID
	EnvironmentData json.RawMessage     `json:"environment_data,omitempty" swaggerignore:"true"` // 环境检测数据 (内部使用)
}

// VerifyCaptcha 验证码验证
// @Summary 验证用户输入
// @Description 验证用户对验证码的操作（滑动或点击）是否正确，并进行行为分析
// @Tags 验证码
// @Accept json
// @Produce json
// @Param body body VerifyRequest true "验证请求参数"
// @Success 200 {object} map[string]interface{} "验证结果"
// @Failure 400 {object} map[string]interface{} "请求参数无效"
// @Failure 404 {object} map[string]interface{} "会话不存在或已过期"
// @Router /api/v1/captcha/verify [post]
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

	if len(req.EnvironmentData) > 0 {
		var envData map[string]interface{}
		if err := json.Unmarshal(req.EnvironmentData, &envData); err == nil {
			envRiskScore := analyzeEnvironmentData(envData)
			if envRiskScore > riskScore {
				riskScore = envRiskScore
			}
			analysisReport += fmt.Sprintf("\n- 环境检测分析:\n")
			analysisReport += fmt.Sprintf("  * 环境风险评分: %.2f\n", envRiskScore)
			if envRiskScore > 50 {
				analysisReport += fmt.Sprintf("  * 环境异常: 检测到高风险环境特征\n")
			}
		}
	}

	if riskScore >= 50 {
		finalSuccess = false
	}

	status := "failed"
	if finalSuccess {
		status = "success"
		sessionMutex.Lock()
		delete(captchaSessions, req.SessionID)
		sessionMutex.Unlock()
	}

	duration := time.Since(startTime).Milliseconds()

	var appID *uint
	if req.ApplicationID > 0 {
		appID = &req.ApplicationID
	}

	verification := &models.Verification{
		SessionID:     req.SessionID,
		CaptchaType:   req.Type,
		ApplicationID: appID,
		UserID:        nil,
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
		IPAddress:      c.ClientIP(),
		UserAgent:      c.GetHeader("User-Agent"),
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

// verifyClickPoints 验证点击验证码的点击点和点击顺序，支持误点容错
// 核心验证逻辑：
// 1. 数量验证：确保点击数量与预期的目标点数量一致
// 2. 位置匹配：使用欧几里得距离计算每个点击位置与目标点的距离
//    - 采用贪心匹配策略，为每个点击找到最近且未被匹配的目标点
//    - 考虑容差值(tolerance)，允许一定的点击偏差
// 3. 顺序验证：将点击顺序转换为对应的目标点顺序
//    - 如果客户端提供了ClickSequence，使用提供的顺序
//    - 否则按照点击的先后顺序进行验证
// 4. 比较验证：确保实际点击顺序与期望顺序一致
// 5. 误点容错：允许少量偏离容差范围的点击
//
// 注意事项：
// - 由于使用了贪心匹配，同一个目标点只能被一个点击匹配
// - 如果存在多个点击匹配到同一个目标点，只有最近的那个会被采用
// - 容差值默认为35像素，可根据图片大小调整
// - 误点容错：允许1次接近容差的点击（35-45像素范围内）
func verifyClickPoints(session *CaptchaSession, req VerifyRequest) (bool, string) {
	if len(req.Points) == 0 {
		return false, "未提供点击坐标"
	}

	clickCount := len(req.Points)
	_ = session.MaxPoints

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
	nearMissTolerance := tolerance + 10
	nearMissCount := 0

	for clickIdx := 0; clickIdx < clickCount; clickIdx++ {
		clickX := req.Points[clickIdx][0]
		clickY := req.Points[clickIdx][1]

		bestMatch := -1
		bestDistance := float64(tolerance) + 1

		for targetIdx := 0; targetIdx < session.MaxPoints; targetIdx++ {
			if usedTargets[targetIdx] {
				continue
			}

			targetX := session.TargetPoints[targetIdx].X
			targetY := session.TargetPoints[targetIdx].Y

			dx := float64(clickX - targetX)
			dy := float64(clickY - targetY)
			distance := math.Sqrt(dx*dx + dy*dy)

			if distance <= float64(tolerance) && distance < bestDistance {
				bestMatch = targetIdx
				bestDistance = distance
			}
		}

		if bestMatch < 0 {
			for targetIdx := 0; targetIdx < session.MaxPoints; targetIdx++ {
				if usedTargets[targetIdx] {
					continue
				}

				targetX := session.TargetPoints[targetIdx].X
				targetY := session.TargetPoints[targetIdx].Y

				dx := float64(clickX - targetX)
				dy := float64(clickY - targetY)
				distance := math.Sqrt(dx*dx + dy*dy)

				if distance <= float64(nearMissTolerance) && distance < bestDistance {
					bestMatch = targetIdx
					bestDistance = distance
					nearMissCount++
				}
			}
		}

		if bestMatch < 0 {
			return false, fmt.Sprintf("点击位置(%d,%d)无法匹配任何目标点，容差范围%d",
				clickX, clickY, tolerance)
		}

		matchedIndices[clickIdx] = bestMatch
		usedTargets[bestMatch] = true
	}

	if nearMissCount > 1 {
		return false, fmt.Sprintf("误点过多: %d次超出容差范围", nearMissCount)
	}

	clickOrder := make([]int, clickCount)
	if len(req.ClickSequence) > 0 {
		if len(req.ClickSequence) != clickCount {
			return false, fmt.Sprintf("点击时序长度不匹配: 提供%d个时序, 实际%d个点击",
				len(req.ClickSequence), clickCount)
		}
		copy(clickOrder, req.ClickSequence)
	} else {
		for i := 0; i < clickCount; i++ {
			clickOrder[i] = i
		}
	}

	clickedTargetsInOrder := make([]int, clickCount)
	for seqIdx, clickIdx := range clickOrder {
		if clickIdx < 0 || clickIdx >= clickCount {
			return false, "点击时序索引无效"
		}
		clickedTargetsInOrder[seqIdx] = matchedIndices[clickIdx]
	}

	for i := 0; i < clickCount; i++ {
		if clickedTargetsInOrder[i] != expectedOrder[i] {
			return false, fmt.Sprintf("点击顺序错误: 第%d个点击应匹配目标%d, 实际匹配目标%d，期望按%s顺序点击",
				i+1, expectedOrder[i]+1, clickedTargetsInOrder[i]+1,
				formatHintOrder(expectedOrder))
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

func analyzeEnvironmentData(envData map[string]interface{}) float64 {
	score := 0.0
	indicators := []string{}

	if riskScoreRaw, ok := envData["risk_score"]; ok {
		if rs, ok := riskScoreRaw.(float64); ok {
			score = rs
		}
	}

	if chainRaw, ok := envData["chain"]; ok {
		switch chain := chainRaw.(type) {
		case []interface{}:
			if len(chain) < 3 {
				score += 15
				indicators = append(indicators, "检测方法链过短")
			}
		case map[string]interface{}:
			if len(chain) < 3 {
				score += 15
				indicators = append(indicators, "检测方法链过短")
			}
		}
	}

	if webdriverRaw, ok := envData["webdriver"]; ok {
		if wd, ok := webdriverRaw.(string); ok {
			if strings.Contains(wd, "wd:true") || strings.Contains(wd, "wd:1") {
				score += 30
				indicators = append(indicators, "检测到WebDriver")
			}
		}
	}

	if webglRaw, ok := envData["webgl"]; ok {
		if wg, ok := webglRaw.(string); ok {
			if wg == "no_webgl" {
				score += 20
				indicators = append(indicators, "WebGL不可用")
			}
			if strings.Contains(wg, "SwiftShader") || strings.Contains(wg, "llvmpipe") || strings.Contains(wg, "Microsoft Basic Render") {
				score += 25
				indicators = append(indicators, "检测到软件渲染器")
			}
		}
	}

	if canvasRaw, ok := envData["canvas"]; ok {
		if cv, ok := canvasRaw.(string); ok {
			if len(cv) < 50 {
				score += 15
				indicators = append(indicators, "Canvas指纹异常")
			}
		}
	}

	if cpuRaw, ok := envData["cpu"]; ok {
		if cpu, ok := cpuRaw.(string); ok {
			if cpu == "unknown" || cpu == "0" || cpu == "1" {
				score += 10
				indicators = append(indicators, "CPU核心数异常")
			}
		}
	}

	if memoryRaw, ok := envData["memory"]; ok {
		if mem, ok := memoryRaw.(string); ok {
			if mem == "unknown" || mem == "0" {
				score += 10
				indicators = append(indicators, "设备内存不可用")
			}
		}
	}

	if touchRaw, ok := envData["touch"]; ok {
		if tc, ok := touchRaw.(string); ok {
			if strings.Contains(tc, "touch:") && !strings.Contains(tc, "touch:0") {
				score += 5
				indicators = append(indicators, "触屏设备")
			}
		}
	}

	if dntRaw, ok := envData["dnt"]; ok {
		if dnt, ok := dntRaw.(string); ok {
			if dnt == "1" || dnt == "yes" {
				score -= 5
			}
		}
	}

	if adblockRaw, ok := envData["adblock"]; ok {
		if ab, ok := adblockRaw.(string); ok {
			if ab == "adblock" {
				score += 10
				indicators = append(indicators, "检测到广告拦截")
			}
		}
	}

	if connectionRaw, ok := envData["connection"]; ok {
		if conn, ok := connectionRaw.(string); ok {
			if strings.Contains(conn, "no_conn") {
				score += 5
				indicators = append(indicators, "网络信息API不可用")
			}
		}
	}

	if len(indicators) > 0 {
		score = math.Min(score, 100)
	}

	return score
}
