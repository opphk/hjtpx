package handler

import (
	"bytes"
	"context"
	"encoding/base64"
	"image"
	"image/color"
	"image/draw"
	"image/png"
	"math/rand"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/math/fixed"
)

type CaptchaType string

const (
	CaptchaTypeNumber CaptchaType = "number"
	CaptchaTypeLetter CaptchaType = "letter"
	CaptchaTypeMixed  CaptchaType = "mixed"
)

type GenerateImageCaptchaRequest struct {
	Type  CaptchaType `form:"type" json:"type"`
	Count int         `form:"count" json:"count"`
}

type GenerateImageCaptchaResponse struct {
	ChallengeID string `json:"challenge_id"`
	Image       string `json:"image"`
}

type VerifyImageCaptchaRequest struct {
	ChallengeID string `json:"challenge_id" binding:"required"`
	Answer      string `json:"answer" binding:"required"`
}

type VerifyImageCaptchaResponse struct {
	Success bool `json:"success"`
}

const (
	captchaWidth  = 120
	captchaHeight = 40
	captchaTTL    = 5 * time.Minute
)

var (
	digitChars    = "0123456789"
	letterChars   = "abcdefghijkmnpqrstuvwxyzABCDEFGHJKLMNPQRSTUVWXYZ"
	allChars      = digitChars + letterChars
	r             *rand.Rand
)

func init() {
	r = rand.New(rand.NewSource(time.Now().UnixNano()))
}

func GenerateImageCaptcha(c *gin.Context) {
	var req GenerateImageCaptchaRequest
	if err := c.ShouldBind(&req); err != nil {
		req.Type = CaptchaTypeMixed
		req.Count = 4
	}

	if req.Count <= 0 || req.Count > 8 {
		req.Count = 4
	}

	var chars string
	switch req.Type {
	case CaptchaTypeNumber:
		chars = digitChars
	case CaptchaTypeLetter:
		chars = letterChars
	default:
		chars = allChars
	}

	answer := generateRandomString(chars, req.Count)
	challengeID := uuid.New().String()

	img := generateCaptchaImage(answer)

	var buf bytes.Buffer
	if err := png.Encode(&buf, img); err != nil {
		response.InternalServerError(c, "failed to generate captcha image")
		return
	}

	imageBase64 := "data:image/png;base64," + base64.StdEncoding.EncodeToString(buf.Bytes())

	// 使用我们的辅助函数存储答案
	setCaptchaAnswer(challengeID, answer)

	response.Success(c, GenerateImageCaptchaResponse{
		ChallengeID: challengeID,
		Image:       imageBase64,
	})
}

// 为了测试，我们保存一个内存存储作为Redis的后备方案
var fallbackCaptchaStore = make(map[string]string)

// setCaptchaAnswer 存储验证码答案
func setCaptchaAnswer(challengeID, answer string) {
	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.Set(ctx, "captcha:"+challengeID, strings.ToLower(answer), captchaTTL)
	} else {
		fallbackCaptchaStore[challengeID] = strings.ToLower(answer)
	}
}

// getCaptchaAnswer 获取验证码答案
func getCaptchaAnswer(challengeID string) (string, bool) {
	if redis.Client != nil {
		ctx := context.Background()
		answer, err := redis.Client.Get(ctx, "captcha:"+challengeID).Result()
		if err == nil {
			return answer, true
		}
		return "", false
	}
	answer, ok := fallbackCaptchaStore[challengeID]
	return answer, ok
}

// deleteCaptchaAnswer 删除验证码答案
func deleteCaptchaAnswer(challengeID string) {
	if redis.Client != nil {
		ctx := context.Background()
		redis.Client.Del(ctx, "captcha:"+challengeID)
	} else {
		delete(fallbackCaptchaStore, challengeID)
	}
}

func VerifyImageCaptcha(c *gin.Context) {
	var req VerifyImageCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	// 使用我们的辅助函数获取答案
	storedAnswer, found := getCaptchaAnswer(req.ChallengeID)
	if !found {
		response.NotFound(c, "captcha expired or not found")
		return
	}

	success := strings.ToLower(req.Answer) == storedAnswer

	if success {
		deleteCaptchaAnswer(req.ChallengeID)
	}

	response.Success(c, VerifyImageCaptchaResponse{
		Success: success,
	})
}

func generateRandomString(chars string, length int) string {
	result := make([]byte, length)
	for i := range result {
		result[i] = chars[r.Intn(len(chars))]
	}
	return string(result)
}

func generateCaptchaImage(text string) *image.RGBA {
	img := image.NewRGBA(image.Rect(0, 0, captchaWidth, captchaHeight))

	bgColor := randomLightColor()
	draw.Draw(img, img.Bounds(), &image.Uniform{C: bgColor}, image.Point{}, draw.Src)

	addNoiseLines(img)
	addNoiseDots(img)

	drawText(img, text)

	return img
}

func randomLightColor() color.RGBA {
	return color.RGBA{
		R: uint8(200 + r.Intn(55)),
		G: uint8(200 + r.Intn(55)),
		B: uint8(200 + r.Intn(55)),
		A: 255,
	}
}

func randomDarkColor() color.RGBA {
	return color.RGBA{
		R: uint8(r.Intn(100)),
		G: uint8(r.Intn(100)),
		B: uint8(r.Intn(100)),
		A: 255,
	}
}

func addNoiseLines(img *image.RGBA) {
	for i := 0; i < 4; i++ {
		x1 := r.Intn(captchaWidth)
		y1 := r.Intn(captchaHeight)
		x2 := r.Intn(captchaWidth)
		y2 := r.Intn(captchaHeight)
		drawLine(img, x1, y1, x2, y2, randomDarkColor())
	}
}

func addNoiseDots(img *image.RGBA) {
	for i := 0; i < 80; i++ {
		x := r.Intn(captchaWidth)
		y := r.Intn(captchaHeight)
		img.Set(x, y, randomDarkColor())
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, col color.Color) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)
	sx := -1
	if x1 < x2 {
		sx = 1
	}
	sy := -1
	if y1 < y2 {
		sy = 1
	}
	err := dx - dy

	for {
		img.Set(x1, y1, col)
		if x1 == x2 && y1 == y2 {
			break
		}
		e2 := 2 * err
		if e2 > -dy {
			err -= dy
			x1 += sx
		}
		if e2 < dx {
			err += dx
			y1 += sy
		}
	}
}

func drawText(img *image.RGBA, text string) {
	f := basicfont.Face7x13
	charWidth := captchaWidth / len(text)

	for i, char := range text {
		x := i*charWidth + (charWidth-7)/2
		y := captchaHeight/2 + 5

		offset := r.Intn(6) - 3
		y += offset

		d := &font.Drawer{
			Dst:  img,
			Src:  image.NewUniform(randomDarkColor()),
			Face: f,
			Dot: fixed.Point26_6{
				X: fixed.I(x),
				Y: fixed.I(y),
			},
		}
		d.DrawString(string(char))
	}
}
