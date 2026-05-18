package handler

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
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
	"strconv"
	"strings"
	"sync"
	"sync/atomic"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"golang.org/x/image/font/sfnt"
	"golang.org/x/image/math/fixed"
	"golang.org/x/image/vector"
)

type CaptchaResponseCache struct {
	mu        sync.RWMutex
	data      map[string]*CachedResponse
	maxSize   int
	ttl       time.Duration
	hitCount  int64
	missCount int64
}

type CachedResponse struct {
	Data      interface{} `json:"data"`
	ExpiresAt time.Time   `json:"expires_at"`
}

func NewCaptchaResponseCache(maxSize int, ttl time.Duration) *CaptchaResponseCache {
	cache := &CaptchaResponseCache{
		data:    make(map[string]*CachedResponse),
		maxSize: maxSize,
		ttl:     ttl,
	}
	go cache.cleanupLoop()
	return cache
}

func (c *CaptchaResponseCache) cleanupLoop() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.cleanup()
	}
}

func (c *CaptchaResponseCache) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()
	now := time.Now()
	for key, resp := range c.data {
		if now.After(resp.ExpiresAt) {
			delete(c.data, key)
		}
	}
	if len(c.data) > c.maxSize {
		for key := range c.data {
			delete(c.data, key)
			if len(c.data) <= c.maxSize/2 {
				break
			}
		}
	}
}

func (c *CaptchaResponseCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if resp, ok := c.data[key]; ok {
		if time.Now().Before(resp.ExpiresAt) {
			atomic.AddInt64(&c.hitCount, 1)
			return resp.Data, true
		}
	}
	atomic.AddInt64(&c.missCount, 1)
	return nil, false
}

func (c *CaptchaResponseCache) Set(key string, data interface{}) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.data) >= c.maxSize {
		var oldestKey string
		var oldestTime time.Time
		first := true
		for k, v := range c.data {
			if first || v.ExpiresAt.Before(oldestTime) {
				oldestKey = k
				oldestTime = v.ExpiresAt
				first = false
			}
		}
		delete(c.data, oldestKey)
	}
	c.data[key] = &CachedResponse{
		Data:      data,
		ExpiresAt: time.Now().Add(c.ttl),
	}
}

func (c *CaptchaResponseCache) GetStats() map[string]interface{} {
	hitCount := atomic.LoadInt64(&c.hitCount)
	missCount := atomic.LoadInt64(&c.missCount)
	total := hitCount + missCount
	hitRate := float64(0)
	if total > 0 {
		hitRate = float64(hitCount) / float64(total) * 100
	}
	return map[string]interface{}{
		"size":       len(c.data),
		"max_size":   c.maxSize,
		"hit_count":  hitCount,
		"miss_count": missCount,
		"hit_rate":   hitRate,
	}
}

type ImageGeneratorPool struct {
	pool sync.Pool
}

func NewImageGeneratorPool(bufferSize int) *ImageGeneratorPool {
	return &ImageGeneratorPool{
		pool: sync.Pool{
			New: func() interface{} {
				return bytes.NewBuffer(make([]byte, 0, bufferSize))
			},
		},
	}
}

func (p *ImageGeneratorPool) Get() *bytes.Buffer {
	return p.pool.Get().(*bytes.Buffer)
}

func (p *ImageGeneratorPool) Put(buf *bytes.Buffer) {
	buf.Reset()
	p.pool.Put(buf)
}

type OptimizedSliderCache struct {
	mu       sync.RWMutex
	sessions map[string]*SliderCacheEntry
	ttl      time.Duration
	maxSize  int
}

type SliderCacheEntry struct {
	ImageURL    string `json:"image_url"`
	PuzzleImage string `json:"puzzle_image"`
	TargetX     int    `json:"target_x"`
	TargetY     int    `json:"target_y"`
	PuzzleY     int    `json:"puzzle_y"`
	Tolerance   int    `json:"tolerance"`
	ExpiresAt   time.Time
}

func NewOptimizedSliderCache(ttl time.Duration, maxSize int) *OptimizedSliderCache {
	cache := &OptimizedSliderCache{
		sessions: make(map[string]*SliderCacheEntry),
		ttl:      ttl,
		maxSize:  maxSize,
	}
	go cache.cleanup()
	return cache
}

func (c *OptimizedSliderCache) cleanup() {
	ticker := time.NewTicker(30 * time.Second)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for id, entry := range c.sessions {
			if now.After(entry.ExpiresAt) {
				delete(c.sessions, id)
			}
		}
		c.mu.Unlock()
	}
}

func (c *OptimizedSliderCache) Set(id string, entry *SliderCacheEntry) {
	c.mu.Lock()
	defer c.mu.Unlock()
	if len(c.sessions) >= c.maxSize {
		for k := range c.sessions {
			delete(c.sessions, k)
			break
		}
	}
	entry.ExpiresAt = time.Now().Add(c.ttl)
	c.sessions[id] = entry
}

func (c *OptimizedSliderCache) Get(id string) (*SliderCacheEntry, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	if entry, ok := c.sessions[id]; ok {
		if time.Now().Before(entry.ExpiresAt) {
			return entry, true
		}
	}
	return nil, false
}

func (c *OptimizedSliderCache) Delete(id string) {
	c.mu.Lock()
	defer c.mu.Unlock()
	delete(c.sessions, id)
}

var (
	optCaptchaSessions = make(map[string]*CaptchaSession)
	optSessionMutex    sync.RWMutex
	optBehaviorService = service.NewBehaviorAnalysisService()

	captchaResponseCache   *CaptchaResponseCache
	sliderImageCache      *OptimizedSliderCache
	imageGeneratorPool    *ImageGeneratorPool
	imageGenerationCounter atomic.Int64
	imageGenerationTime    atomic.Int64
)

func init() {
	captchaResponseCache = NewCaptchaResponseCache(1000, 5*time.Minute)
	sliderImageCache = NewOptimizedSliderCache(10*time.Minute, 5000)
	imageGeneratorPool = NewImageGeneratorPool(64 * 1024)

	go func() {
		ticker := time.NewTicker(5 * time.Minute)
		defer ticker.Stop()
		for range ticker.C {
			cleanupExpiredSessions()
		}
	}()
}

func optGenerateSessionID() string {
	return fmt.Sprintf("opt_sess_%d_%d", time.Now().UnixNano(), rand.Intn(10000))
}

func optGetHintPrefix(mode CaptchaMode, language string) string {
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

func optGetArrowSeparator(language string) string {
	if language == "zh" || language == "zh-CN" {
		return " → "
	}
	return " → "
}

type OptIconType string

const (
	OptIconCircle   OptIconType = "circle"
	OptIconSquare   OptIconType = "square"
	OptIconTriangle OptIconType = "triangle"
	OptIconStar     OptIconType = "star"
	OptIconDiamond  OptIconType = "diamond"
	OptIconHeart    OptIconType = "heart"
	OptIconArrow    OptIconType = "arrow"
	OptIconCross    OptIconType = "cross"
	OptIconMoon     OptIconType = "moon"
	OptIconRing     OptIconType = "ring"
)

var optClickIcons = []OptIconType{
	OptIconCircle, OptIconSquare, OptIconTriangle, OptIconStar, OptIconDiamond,
	OptIconHeart, OptIconArrow, OptIconCross, OptIconMoon, OptIconRing,
}

var optIconNames = map[OptIconType]string{
	OptIconCircle:   "圆形",
	OptIconSquare:   "方形",
	OptIconTriangle: "三角形",
	OptIconStar:     "星形",
	OptIconDiamond:  "菱形",
	OptIconHeart:    "心形",
	OptIconArrow:    "箭头",
	OptIconCross:    "十字",
	OptIconMoon:     "月牙",
	OptIconRing:     "圆环",
}

var optClickChineseChars = []string{
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

var optClickLetterChars = []string{
	"A", "B", "C", "D", "E", "F", "G", "H", "J", "K",
	"L", "M", "N", "P", "Q", "R", "S", "T", "U", "V",
	"W", "X", "Y", "Z",
}

var optClickNumberChars = []string{
	"0", "1", "2", "3", "4", "5", "6", "7", "8", "9",
}

func optGetIconNameLocalized(icon OptIconType, language string) string {
	if language == "zh" || language == "zh-CN" {
		if name, ok := optIconNames[icon]; ok {
			return name
		}
	}
	iconENNames := map[OptIconType]string{
		OptIconCircle:   "Circle",
		OptIconSquare:   "Square",
		OptIconTriangle: "Triangle",
		OptIconStar:     "Star",
		OptIconDiamond:  "Diamond",
		OptIconHeart:    "Heart",
		OptIconArrow:    "Arrow",
		OptIconCross:    "Cross",
		OptIconMoon:     "Moon",
		OptIconRing:     "Ring",
	}
	if name, ok := iconENNames[icon]; ok {
		return name
	}
	return string(icon)
}

func optGenerateSmartHint(session *CaptchaSession, displayChars []string, language string) string {
	prefix := optGetHintPrefix(session.Mode, language)
	arrow := optGetArrowSeparator(language)

	parts := make([]string, len(session.HintOrder))
	for i, idx := range session.HintOrder {
		if session.Mode == ModeIcon {
			parts[i] = optGetIconNameLocalized(OptIconType(displayChars[idx]), language)
		} else {
			parts[i] = displayChars[idx]
		}
	}

	return prefix + strings.Join(parts, arrow)
}

func optShuffleInts(arr []int) []int {
	result := make([]int, len(arr))
	perm := rand.Perm(len(arr))
	for i := 0; i < len(arr); i++ {
		result[i] = arr[perm[i]]
	}
	return result
}

var (
	optChineseFont     *sfnt.Font
	optChineseFontData []byte
	optFontLoadOnce    sync.Once
	optFontLoadError   error
)

func optLoadChineseFont() error {
	optFontLoadOnce.Do(func() {
		optChineseFontData, optFontLoadError = os.ReadFile("/usr/share/fonts/truetype/wqy/wqy-microhei.ttc")
		if optFontLoadError != nil {
			return
		}
		var coll *sfnt.Collection
		coll, optFontLoadError = sfnt.ParseCollection(optChineseFontData)
		if optFontLoadError != nil {
			return
		}
		optChineseFont, optFontLoadError = coll.Font(0)
	})
	return optFontLoadError
}

func optRenderCharToMask(char rune, size int) *image.Alpha {
	if err := optLoadChineseFont(); err != nil {
		return nil
	}
	var buf sfnt.Buffer
	idx, err := optChineseFont.GlyphIndex(&buf, char)
	if err != nil || idx == 0 {
		return nil
	}
	ppem := fixed.I(size)
	segs, err := optChineseFont.LoadGlyph(&buf, idx, ppem, nil)
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

func optRenderCharToRGBA(char rune, size int, textColor color.RGBA) *image.RGBA {
	alpha := optRenderCharToMask(char, size)
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

func optRotateImageRGBA(src *image.RGBA, angleDeg float64) *image.RGBA {
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

func optRandomVibrantColor() color.RGBA {
	hue := rand.Intn(360)
	saturation := 0.6 + rand.Float64()*0.35
	value := 0.5 + rand.Float64()*0.4
	r, g, b := optHsvToRGB(hue, saturation, value)
	return color.RGBA{
		R: uint8(r * 255),
		G: uint8(g * 255),
		B: uint8(b * 255),
		A: 200 + uint8(rand.Intn(56)),
	}
}

func optHsvToRGB(h int, s, v float64) (float64, float64, float64) {
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

func optDrawGradientBackground(img *image.RGBA) {
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

func optAddNoiseDots(img *image.RGBA, count int) {
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

func optAddInterferenceLines(img *image.RGBA, count int) {
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

func optAddGridLines(img *image.RGBA) {
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

func optIsOverlapping(x, y, size int, placed []image.Rectangle) bool {
	candidate := image.Rect(x-size/2, y-size/2, x+size/2, y+size/2)
	for _, r := range placed {
		overlap := candidate.Intersect(r)
		if overlap.Dx() > 10 && overlap.Dy() > 10 {
			return true
		}
	}
	return false
}

func optGenerateClickImageWithBackground(session *CaptchaSession) (string, []ClickPoint, []int, string) {
	startTime := time.Now()
	defer func() {
		imageGenerationCounter.Add(1)
		imageGenerationTime.Add(time.Since(startTime).Milliseconds())
	}()

	session.ImageWidth = 300
	session.ImageHeight = 300
	session.Tolerance = 25

	img := image.NewRGBA(image.Rect(0, 0, session.ImageWidth, session.ImageHeight))

	optDrawGradientBackground(img)
	optAddNoiseDots(img, 200)
	optAddInterferenceLines(img, 4)
	optAddGridLines(img)

	maxPoints := session.MaxPoints
	totalChars := 6 + rand.Intn(3)
	if maxPoints > totalChars {
		maxPoints = totalChars
	}

	targetChars := make([]string, maxPoints)
	for i := 0; i < maxPoints; i++ {
		targetChars[i] = optGetCharForIndex(i, session.Mode)
	}

	decoyCount := totalChars - maxPoints
	allChars := make([]string, totalChars)
	for i := 0; i < maxPoints; i++ {
		allChars[i] = targetChars[i]
	}
	for i := 0; i < decoyCount; i++ {
		allChars[maxPoints+i] = optGetCharForIndex(i+maxPoints+100, session.Mode)
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
			if !optIsOverlapping(cx, cy, halfSize, placedRects) {
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
		textColor := optRandomVibrantColor()

		if session.Mode == ModeIcon {
			iconType := OptIconType(shuffledChars[i])
			rendered := optRenderIcon(iconType, charSize, textColor)
			rotated := optRotateImageRGBA(rendered, rotation)
			draw.Draw(img, image.Rect(
				charCenters[i].X-rotated.Bounds().Dx()/2,
				charCenters[i].Y-rotated.Bounds().Dy()/2,
				charCenters[i].X-rotated.Bounds().Dx()/2+rotated.Bounds().Dx(),
				charCenters[i].Y-rotated.Bounds().Dy()/2+rotated.Bounds().Dy(),
			), rotated, image.Point{}, draw.Over)
		} else {
			rendered := optRenderCharToRGBA(char, charSize, textColor)
			if rendered == nil {
				rendered = image.NewRGBA(image.Rect(0, 0, 10, 10))
				draw.Draw(rendered, rendered.Bounds(), &image.Uniform{textColor}, image.Point{}, draw.Src)
			}
			rotated := optRotateImageRGBA(rendered, rotation)
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
		hintOrder = optShuffleInts(hintOrder)
	}

	session.HintOrder = hintOrder

	session.Hint = optGenerateSmartHint(session, displayChars, session.Language)

	session.Points = make([][2]int, maxPoints)
	for i, pt := range targetPoints {
		session.Points[i] = [2]int{pt.X, pt.Y}
	}

	base64Data := optImageToBase64(img)
	return "data:image/png;base64," + base64Data, targetPoints, hintOrder, session.Hint
}

func optGetCharForIndex(index int, mode CaptchaMode) string {
	switch mode {
	case ModeNumber:
		return optClickNumberChars[rand.Intn(len(optClickNumberChars))]
	case ModeLetter:
		return optClickLetterChars[rand.Intn(len(optClickLetterChars))]
	case ModeChinese:
		return optClickChineseChars[rand.Intn(len(optClickChineseChars))]
	case ModeIcon:
		return string(optClickIcons[rand.Intn(len(optClickIcons))])
	case ModeMixed:
		switch rand.Intn(3) {
		case 0:
			return optClickNumberChars[rand.Intn(len(optClickNumberChars))]
		case 1:
			return optClickLetterChars[rand.Intn(len(optClickLetterChars))]
		default:
			return optClickChineseChars[rand.Intn(len(optClickChineseChars))]
		}
	default:
		return optClickNumberChars[rand.Intn(len(optClickNumberChars))]
	}
}

func optRenderIcon(iconType OptIconType, size int, iconColor color.RGBA) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, size, size))
	centerX, centerY := size/2, size/2
	halfSize := size / 3

	switch iconType {
	case OptIconCircle:
		optDrawFilledCircle(img, centerX, centerY, halfSize, iconColor)
	case OptIconSquare:
		optDrawFilledRect(img, centerX-halfSize, centerY-halfSize, halfSize*2, halfSize*2, iconColor)
	case OptIconTriangle:
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				relY := float64(y-centerY) / float64(halfSize)
				relX := float64(x-centerX) / float64(halfSize)
				if relY >= -1 && relY <= 0 && math.Abs(relX) <= -relY {
					img.Set(x, y, iconColor)
				}
			}
		}
	case OptIconStar:
		optDrawStar(img, centerX, centerY, halfSize, iconColor)
	case OptIconDiamond:
		for y := 0; y < size; y++ {
			for x := 0; x < size; x++ {
				relX := math.Abs(float64(x - centerX))
				relY := math.Abs(float64(y - centerY))
				if relX+relY <= float64(halfSize) {
					img.Set(x, y, iconColor)
				}
			}
		}
	case OptIconHeart:
		optDrawHeart(img, centerX, centerY, halfSize, iconColor)
	case OptIconArrow:
		optDrawArrow(img, centerX, centerY, halfSize, iconColor)
	case OptIconCross:
		thickness := halfSize / 2
		if thickness < 2 {
			thickness = 2
		}
		optDrawFilledRect(img, centerX-thickness, centerY-halfSize, thickness*2, halfSize*2, iconColor)
		optDrawFilledRect(img, centerX-halfSize, centerY-thickness, halfSize*2, thickness*2, iconColor)
	case OptIconMoon:
		optDrawFilledCircle(img, centerX, centerY, halfSize, iconColor)
		coverColor := color.RGBA{
			R: 220, G: 220, B: 225, A: 255,
		}
		optDrawFilledCircle(img, centerX+halfSize/3, centerY-halfSize/4, halfSize*3/4, coverColor)
	case OptIconRing:
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

func optDrawStar(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
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
	optFillPolygon(img, points, c)
}

func optDrawHeart(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
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

func optDrawArrow(img *image.RGBA, cx, cy, radius int, c color.RGBA) {
	shaftLen := radius * 3 / 4
	shaftThick := radius / 4
	if shaftThick < 2 {
		shaftThick = 2
	}
	headSize := radius / 2

	optDrawFilledRect(img, cx-shaftLen, cy-shaftThick/2, shaftLen, shaftThick, c)

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

func optFillPolygon(img *image.RGBA, points [][2]float64, c color.RGBA) {
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

func optDrawFilledCircle(img *image.RGBA, cx, cy, radius int, c color.Color) {
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

func optDrawFilledRect(img *image.RGBA, x, y, w, h int, c color.Color) {
	for dy := 0; dy < h; dy++ {
		for dx := 0; dx < w; dx++ {
			px, py := x+dx, y+dy
			if px >= 0 && px < img.Bounds().Dx() && py >= 0 && py < img.Bounds().Dy() {
				img.Set(px, py, c)
			}
		}
	}
}

func optDrawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.Color) {
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

func optDrawBezier(img *image.RGBA, x0, y0, x1, y1, x2, y2 int, c color.Color) {
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

func optClampU8(v int) uint8 {
	if v < 0 {
		return 0
	}
	if v > 255 {
		return 255
	}
	return uint8(v)
}

func optIsInPuzzlePiece(x, y, pieceSize, radius int) bool {
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

func optAddPuzzleShadow(pieceImg *image.RGBA, pieceSize, radius int) {
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

func optAddCutoutBorder(img *image.RGBA, targetX, targetY, pieceSize, radius int) {
	borderColor := color.RGBA{255, 255, 255, 200}

	for y := 0; y < pieceSize; y++ {
		for x := 0; x < pieceSize+radius; x++ {
			if !optIsInPuzzlePiece(x, y, pieceSize, radius) {
				continue
			}
			isBorder := false
			for dy := -1; dy <= 1 && !isBorder; dy++ {
				for dx := -1; dx <= 1 && !isBorder; dx++ {
					if dx == 0 && dy == 0 {
						continue
					}
					if !optIsInPuzzlePiece(x+dx, y+dy, pieceSize, radius) {
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
							optClampU8(int(orig.R) * factor / 100),
							optClampU8(int(orig.G) * factor / 100),
							optClampU8(int(orig.B) * factor / 100),
							255,
						})
					}
				}
			}
		}
	}
}

func optImageToBase64(img image.Image) string {
	buf := imageGeneratorPool.Get()
	defer imageGeneratorPool.Put(buf)
	buf.Reset()

	encoder := png.Encoder{CompressionLevel: png.BestSpeed}
	_ = encoder.Encode(buf, img)
	return hex.EncodeToString(buf.Bytes())
}

func optGenerateSliderCaptchaImages() (string, string, int, int) {
	startTime := time.Now()
	defer func() {
		imageGenerationCounter.Add(1)
		imageGenerationTime.Add(time.Since(startTime).Milliseconds())
	}()

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
			optDrawFilledCircle(img, rand.Intn(width), rand.Intn(height), 8+rand.Intn(35), c)
		case 1:
			optDrawFilledRect(img, rand.Intn(width), rand.Intn(height), 15+rand.Intn(50), 8+rand.Intn(30), c)
		case 2:
			optDrawLine(img, rand.Intn(width), rand.Intn(height), rand.Intn(width), rand.Intn(height), c)
		case 3:
			optDrawBezier(img, rand.Intn(width), rand.Intn(height), rand.Intn(width), rand.Intn(height), rand.Intn(width), rand.Intn(height), c)
		}
	}

	for i := 0; i < 1000; i++ {
		x := rand.Intn(width)
		y := rand.Intn(height)
		noise := rand.Intn(50) - 25
		p := img.RGBAAt(x, y)
		img.Set(x, y, color.RGBA{
			optClampU8(int(p.R) + noise),
			optClampU8(int(p.G) + noise),
			optClampU8(int(p.B) + noise),
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
				if optIsInPuzzlePiece(px, py, pieceSize, bumpRadius) {
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

	optAddPuzzleShadow(pieceImg, pieceSize, bumpRadius)
	optAddCutoutBorder(img, targetX, targetY, pieceSize, bumpRadius)

	imageURL := "data:image/png;base64," + optImageToBase64(img)
	puzzleImage := "data:image/png;base64," + optImageToBase64(pieceImg)

	return imageURL, puzzleImage, targetX, targetY
}

func optGetSliderCaptcha(c *gin.Context) {
	startTime := time.Now()
	ctx := c.Request.Context()

	select {
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{
			"success": false,
			"message": "request timeout",
		})
		return
	default:
	}

	sessionID := optGenerateSessionID()
	imageURL, puzzleImage, targetX, targetY := optGenerateSliderCaptchaImages()

	session := &CaptchaSession{
		ID:        sessionID,
		Type:      "slider",
		TargetX:   targetX,
		TargetY:   targetY,
		CreatedAt: time.Now(),
	}

	optSessionMutex.Lock()
	optCaptchaSessions[sessionID] = session
	optSessionMutex.Unlock()

	cacheKey := fmt.Sprintf("slider:%s:%s", sessionID, c.ClientIP())
	captchaResponseCache.Set(cacheKey, map[string]interface{}{
		"image_url":    imageURL,
		"puzzle_image": puzzleImage,
		"target_x":     targetX,
		"target_y":     targetY,
	})

	sliderImageCache.Set(sessionID, &SliderCacheEntry{
		ImageURL:    imageURL,
		PuzzleImage: puzzleImage,
		TargetX:     targetX,
		TargetY:     targetY,
		PuzzleY:     targetY,
		Tolerance:   10,
	})

	c.Header("X-Response-Time", strconv.FormatInt(time.Since(startTime).Milliseconds(), 10)+"ms")

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

func optGetClickCaptcha(c *gin.Context) {
	startTime := time.Now()
	ctx := c.Request.Context()

	select {
	case <-ctx.Done():
		c.JSON(http.StatusRequestTimeout, gin.H{
			"success": false,
			"message": "request timeout",
		})
		return
	default:
	}

	sessionID := optGenerateSessionID()
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

	imageURL, _, hintOrder, hint := optGenerateClickImageWithBackground(session)

	optSessionMutex.Lock()
	optCaptchaSessions[sessionID] = session
	optSessionMutex.Unlock()

	c.Header("X-Response-Time", strconv.FormatInt(time.Since(startTime).Milliseconds(), 10)+"ms")

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

type optBehaviorDataPoint struct {
	Event     string  `json:"event"`
	Timestamp int64   `json:"timestamp"`
	X         float64 `json:"x"`
	Y         float64 `json:"y"`
}

type OptVerifyRequest struct {
	SessionID       string              `json:"session_id" binding:"required"`
	Type            string              `json:"type" binding:"required"`
	X               int                 `json:"x"`
	Y               int                 `json:"y"`
	Points          [][2]int            `json:"points"`
	ClickSequence   []int               `json:"click_sequence"`
	BehaviorData    []optBehaviorDataPoint `json:"behavior_data"`
	SpeedData       json.RawMessage     `json:"speed_data,omitempty" swaggerignore:"true"`
	ApplicationID   uint                `json:"application_id"`
	EnvironmentData json.RawMessage     `json:"environment_data,omitempty" swaggerignore:"true"`
}

func optVerifyCaptcha(c *gin.Context) {
	startTime := time.Now()
	ctx := c.Request.Context()

	var req OptVerifyRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "invalid request parameters",
		})
		return
	}

	optSessionMutex.RLock()
	session, exists := optCaptchaSessions[req.SessionID]
	optSessionMutex.RUnlock()

	if !exists {
		c.JSON(http.StatusNotFound, gin.H{
			"success": false,
			"message": "session not found or expired",
		})
		return
	}

	if session.Type != req.Type {
		c.JSON(http.StatusBadRequest, gin.H{
			"success": false,
			"message": "verification type mismatch",
		})
		return
	}

	var captchaSuccess bool
	var failReason string

	if req.Type == "slider" {
		tolerance := 10
		captchaSuccess = intAbs(req.X-session.TargetX) <= tolerance && intAbs(req.Y-session.TargetY) <= tolerance
		if !captchaSuccess {
			failReason = fmt.Sprintf("slider position deviation too large: expected(%d,%d), actual(%d,%d), tolerance(%d)",
				session.TargetX, session.TargetY, req.X, req.Y, tolerance)
		}
	} else if req.Type == "click" {
		captchaSuccess, failReason = optVerifyClickPoints(session, req)
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

	finalSuccess, riskScore, analysisReport := optBehaviorService.VerifyWithBehaviorAnalysis(
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
			analysisReport += fmt.Sprintf("\n- environment detection analysis:\n")
			analysisReport += fmt.Sprintf("  * environment risk score: %.2f\n", envRiskScore)
			if envRiskScore > 50 {
				analysisReport += fmt.Sprintf("  * environment anomaly: high-risk environment features detected\n")
			}
		}
	}

	if riskScore >= 50 {
		finalSuccess = false
	}

	status := "failed"
	if finalSuccess {
		status = "success"
		optSessionMutex.Lock()
		delete(optCaptchaSessions, req.SessionID)
		optSessionMutex.Unlock()
		sliderImageCache.Delete(req.SessionID)
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

	if db != nil {
		if err := db.WithContext(ctx).Create(verification).Error; err != nil {
			fmt.Printf("Failed to save verification: %v\n", err)
		}
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

	if db != nil {
		if err := db.WithContext(ctx).Create(logEntry).Error; err != nil {
			fmt.Printf("Failed to save verification log: %v\n", err)
		}
	}

	message := "verification failed"
	if finalSuccess {
		message = "verification successful"
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

	c.Header("X-Response-Time", strconv.FormatInt(duration, 10)+"ms")

	c.JSON(http.StatusOK, response)
}

func optVerifyClickPoints(session *CaptchaSession, req OptVerifyRequest) (bool, string) {
	if len(req.Points) == 0 {
		return false, "no click coordinates provided"
	}

	clickCount := len(req.Points)

	if clickCount != session.MaxPoints {
		return false, "click count mismatch"
	}

	if req.ClickSequence != nil && len(req.ClickSequence) != clickCount {
		return false, "click sequence length mismatch"
	}

	tolerance := session.Tolerance
	if tolerance <= 0 {
		tolerance = 35
	}

	if session.TargetPoints == nil || len(session.TargetPoints) == 0 {
		return false, "target points empty"
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
			return false, fmt.Sprintf("click position(%d,%d) cannot match any target point, tolerance range %d",
				clickX, clickY, tolerance)
		}

		matchedIndices[clickIdx] = bestMatch
		usedTargets[bestMatch] = true
	}

	if nearMissCount > 1 {
		return false, fmt.Sprintf("too many missed clicks: %d out of tolerance range", nearMissCount)
	}

	clickOrder := make([]int, clickCount)
	if len(req.ClickSequence) > 0 {
		if len(req.ClickSequence) != clickCount {
			return false, fmt.Sprintf("click sequence length mismatch: provided %d sequences, actual %d clicks",
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
			return false, "click sequence index invalid"
		}
		clickedTargetsInOrder[seqIdx] = matchedIndices[clickIdx]
	}

	for i := 0; i < clickCount; i++ {
		if clickedTargetsInOrder[i] != expectedOrder[i] {
			return false, fmt.Sprintf("click order error: click %d should match target %d, actually matched target %d, expected to click in %s order",
				i+1, expectedOrder[i]+1, clickedTargetsInOrder[i]+1,
				formatHintOrder(expectedOrder))
		}
	}

	return true, ""
}

func optFormatHintOrder(order []int) string {
	if len(order) == 0 {
		return ""
	}
	parts := make([]string, len(order))
	for i, idx := range order {
		parts[i] = fmt.Sprintf("%d", idx+1)
	}
	return strings.Join(parts, "→")
}

func optIntAbs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func optCleanupExpiredSessions() {
	optSessionMutex.Lock()
	defer optSessionMutex.Unlock()
	now := time.Now()
	for id, session := range optCaptchaSessions {
		if now.Sub(session.CreatedAt) > 10*time.Minute {
			delete(optCaptchaSessions, id)
			sliderImageCache.Delete(id)
		}
	}
}

func optAnalyzeEnvironmentData(envData map[string]interface{}) float64 {
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
				indicators = append(indicators, "detection method chain too short")
			}
		case map[string]interface{}:
			if len(chain) < 3 {
				score += 15
				indicators = append(indicators, "detection method chain too short")
			}
		}
	}

	if webdriverRaw, ok := envData["webdriver"]; ok {
		if wd, ok := webdriverRaw.(string); ok {
			if strings.Contains(wd, "wd:true") || strings.Contains(wd, "wd:1") {
				score += 30
				indicators = append(indicators, "webdriver detected")
			}
		}
	}

	if webglRaw, ok := envData["webgl"]; ok {
		if wg, ok := webglRaw.(string); ok {
			if wg == "no_webgl" {
				score += 20
				indicators = append(indicators, "webgl not available")
			}
			if strings.Contains(wg, "SwiftShader") || strings.Contains(wg, "llvmpipe") || strings.Contains(wg, "Microsoft Basic Render") {
				score += 25
				indicators = append(indicators, "software renderer detected")
			}
		}
	}

	if canvasRaw, ok := envData["canvas"]; ok {
		if cv, ok := canvasRaw.(string); ok {
			if len(cv) < 50 {
				score += 15
				indicators = append(indicators, "canvas fingerprint abnormal")
			}
		}
	}

	if cpuRaw, ok := envData["cpu"]; ok {
		if cpu, ok := cpuRaw.(string); ok {
			if cpu == "unknown" || cpu == "0" || cpu == "1" {
				score += 10
				indicators = append(indicators, "cpu core count abnormal")
			}
		}
	}

	if memoryRaw, ok := envData["memory"]; ok {
		if mem, ok := memoryRaw.(string); ok {
			if mem == "unknown" || mem == "0" {
				score += 10
				indicators = append(indicators, "device memory not available")
			}
		}
	}

	if touchRaw, ok := envData["touch"]; ok {
		if tc, ok := touchRaw.(string); ok {
			if strings.Contains(tc, "touch:") && !strings.Contains(tc, "touch:0") {
				score += 5
				indicators = append(indicators, "touch device")
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
				indicators = append(indicators, "ad blocker detected")
			}
		}
	}

	if connectionRaw, ok := envData["connection"]; ok {
		if conn, ok := connectionRaw.(string); ok {
			if strings.Contains(conn, "no_conn") {
				score += 5
				indicators = append(indicators, "network information api not available")
			}
		}
	}

	if len(indicators) > 0 {
		score = math.Min(score, 100)
	}

	return score
}

func GetCaptchaStats(c *gin.Context) {
	total := imageGenerationCounter.Load()
	totalTime := imageGenerationTime.Load()
	avgTime := int64(0)
	if total > 0 {
		avgTime = totalTime / total
	}

	c.JSON(http.StatusOK, gin.H{
		"image_generation_count":  total,
		"image_generation_avg_ms": avgTime,
		"cache_stats":             captchaResponseCache.GetStats(),
		"session_count": func() int {
			optSessionMutex.RLock()
			defer optSessionMutex.RUnlock()
			return len(optCaptchaSessions)
		}(),
	})
}

func GenerateRequestSignature(secretKey, method, path string, timestamp int64, nonce string) string {
	data := fmt.Sprintf("%s:%s:%d:%s", method, path, timestamp, nonce)
	h := sha256.Sum256([]byte(data))
	return hex.EncodeToString(h[:])
}
