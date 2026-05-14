package click

import (
	"bytes"
	"crypto/rand"
	"encoding/base64"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math/big"
	"os"
	"strings"
	"time"

	"github.com/golang/freetype"
	"github.com/golang/freetype/truetype"
)

var chineseChars = []string{
	"的一是不了在人有我他这个们中来上大为和国地到以说时要就出可也你对生能而会子那得于着下自之年过发后作里用道行所然家种事成方多经么去法学如都同现当没动面起看定天分还小好小身",
	"万与比较场从公此工活意开业用思品物应前队样自维器果距社常平总清它设无机械目标联内相结今天义各三指定直两间只干关命条点前线没更并需存目三低却强配足跟追敢悉耗损材量",
	"单据官运希验促枝惊语态列纷核步聚燕适届满临宜析敏彭拓印封锐溢迫沟昌盟贴措辞扣巧伐逊粗俗咸胡巫盼孔贾赤鲁软惨券漏壳晨饰屡辽忆挨酷弥妖遮婆帐丹忧魂厚泥缘衫绑匪徒俊猫晶兽笼甫葬吐溜裤疯瘫蛮腾贼媒讽镜",
	"姓碧滑闹郭储窃颂彬鸽蛇齿拦钓钙飘饰趴隐隔麦芒苍肤贞贝卜卦侍钉铃闲阔洒酷蹲颤裤扮逗罚恰棋葛芯竖柳咏缸枢枪飘泛潭笛卵溢沃炸峡沃岛甫逝裕朴匀昂妨郁株咸宜齿眠吐逆阵亡惨纽吻凑耶",
	"诞戚墓兆凶庙币废奔孝宅凶巾帆帆帐幅币布幼带奥荒诞孝岛帝瞬眉亭岸融岭脊岭崩帐帘帖帅帆席带帮年干幼帆布饰庄弄塔婆媚媳琅怖性怕急脉箱帖岭岭带带帮庄布帝帝岛帮帖岭帖幼帮帐带帮帝带帮帝帝岛岛",
	"带带帝岛帮帖帐带帮帖幼帮带带帝岛帮帖帝带带帝带帝带帮帖岛带带帮帖帐带带帝岛帮帖幼帝岛带带帮帝带帮帝岛帮帖帝岛帮帖帝带岛帮性急事同而于每从会时可下变过因",
}

type CaptchaGenerator struct {
	charLib []string
	font    *truetype.Font
}

func NewCaptchaGenerator() (*CaptchaGenerator, error) {
	font, err := loadSystemFont()
	if err != nil {
		return nil, fmt.Errorf("failed to load font: %w", err)
	}

	allChars := strings.Join(chineseChars, "")
	uniqueChars := removeDuplicateChars(allChars)

	return &CaptchaGenerator{
		charLib: uniqueChars,
		font:    font,
	}, nil
}

func removeDuplicateChars(s string) []string {
	seen := make(map[rune]bool)
	var result []string
	for _, ch := range s {
		if !seen[ch] {
			seen[ch] = true
			result = append(result, string(ch))
		}
	}
	return result
}

func loadSystemFont() (*truetype.Font, error) {
	fontPaths := []string{
		"/usr/share/fonts/truetype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/opentype/noto/NotoSansCJK-Regular.ttc",
		"/usr/share/fonts/truetype/wqy/wqy-zenhei.ttc",
		"/usr/share/fonts/truetype/dejavu/DejaVuSans.ttf",
		"/usr/share/fonts/dejavu/DejaVuSans.ttf",
		"/System/Library/Fonts/PingFang.ttc",
		"/System/Library/Fonts/STHeiti Light.ttc",
		"C:\\Windows\\Fonts\\simsun.ttc",
		"C:\\Windows\\Fonts\\msyh.ttc",
	}

	for _, path := range fontPaths {
		data, err := os.ReadFile(path)
		if err == nil {
			font, err := truetype.Parse(data)
			if err == nil {
				return font, nil
			}
		}
	}

	return nil, fmt.Errorf("no suitable font found")
}

func (cg *CaptchaGenerator) GenerateCaptcha(charCount int) (*CaptchaData, error) {
	if charCount < 3 {
		charCount = 3
	}
	if charCount > 5 {
		charCount = 5
	}

	targetChars, err := cg.selectRandomChars(charCount)
	if err != nil {
		return nil, err
	}

	captchaID, err := generateID()
	if err != nil {
		return nil, err
	}

	img, charPositions, err := cg.renderChars(targetChars)
	if err != nil {
		return nil, err
	}

	imgBase64, err := encodeToBase64(img)
	if err != nil {
		return nil, err
	}

	return &CaptchaData{
		ID:            captchaID,
		Image:         imgBase64,
		TargetChars:   targetChars,
		CharPositions: charPositions,
		CreatedAt:     time.Now(),
	}, nil
}

func (cg *CaptchaGenerator) selectRandomChars(count int) ([]string, error) {
	var chars []string
	used := make(map[int]bool)

	for len(chars) < count {
		idx, err := rand.Int(rand.Reader, big.NewInt(int64(len(cg.charLib))))
		if err != nil {
			return nil, err
		}
		i := int(idx.Int64())
		if !used[i] {
			used[i] = true
			chars = append(chars, cg.charLib[i])
		}
	}

	return chars, nil
}

func (cg *CaptchaGenerator) renderChars(chars []string) (image.Image, []CharPosition, error) {
	imgWidth := DefaultImageWidth
	imgHeight := DefaultImageHeight

	img := image.NewRGBA(image.Rect(0, 0, imgWidth, imgHeight))

	for y := 0; y < imgHeight; y++ {
		for x := 0; x < imgWidth; x++ {
			img.Set(x, y, color.RGBA{245, 247, 250, 255})
		}
	}

	addNoise(img, imgWidth, imgHeight)

	dpi := float64(72)
	fontSize := float64(40)

	var positions []CharPosition

	spacing := imgWidth / (len(chars) + 1)
	startX := spacing / 2

	for i, ch := range chars {
		x := startX + i*spacing + 25
		y := imgHeight/2 + 15

		fgColor := color.RGBA{
			R: uint8(randomInt(30, 80)),
			G: uint8(randomInt(30, 80)),
			B: uint8(randomInt(30, 80)),
			A: 255,
		}

		ctx := freetype.NewContext()
		ctx.SetDPI(dpi)
		ctx.SetFont(cg.font)
		ctx.SetFontSize(fontSize)
		ctx.SetClip(img.Bounds())
		ctx.SetDst(img)
		ctx.SetSrc(image.NewUniform(fgColor))

		pt := freetype.Pt(x, y)
		_, err := ctx.DrawString(ch, pt)
		if err != nil {
			continue
		}

		charWidth := 40
		charHeight := 50

		positions = append(positions, CharPosition{
			Char:   ch,
			X:      x - 5,
			Y:      y - charHeight + 10,
			Width:  charWidth,
			Height: charHeight,
		})
	}

	return img, positions, nil
}

func addNoise(img *image.RGBA, width, height int) {
	noiseCount := 800
	for i := 0; i < noiseCount; i++ {
		x, _ := rand.Int(rand.Reader, big.NewInt(int64(width)))
		y, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		gray := uint8(randomInt(200, 240))
		img.Set(int(x.Int64()), int(y.Int64()), color.RGBA{gray, gray, gray, 255})
	}

	lineCount := 5
	for i := 0; i < lineCount; i++ {
		x1, _ := rand.Int(rand.Reader, big.NewInt(int64(width)))
		y1, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))
		x2, _ := rand.Int(rand.Reader, big.NewInt(int64(width)))
		y2, _ := rand.Int(rand.Reader, big.NewInt(int64(height)))

		drawLine(img, int(x1.Int64()), int(y1.Int64()), int(x2.Int64()), int(y2.Int64()), color.RGBA{200, 200, 200, 255})
	}
}

func drawLine(img *image.RGBA, x1, y1, x2, y2 int, c color.RGBA) {
	dx := abs(x2 - x1)
	dy := abs(y2 - y1)

	if dx > dy {
		if x1 > x2 {
			x1, x2 = x2, x1
			y1, y2 = y2, y1
		}
		for x := x1; x <= x2; x++ {
			t := float64(x-x1) / float64(x2-x1)
			y := int(float64(y1) + t*float64(y2-y1))
			if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.Set(x, y, c)
			}
		}
	} else {
		if y1 > y2 {
			x1, x2 = x2, x1
			y1, y2 = y2, y1
		}
		for y := y1; y <= y2; y++ {
			t := float64(y-y1) / float64(y2-y1)
			x := int(float64(x1) + t*float64(x2-x1))
			if x >= 0 && x < img.Bounds().Dx() && y >= 0 && y < img.Bounds().Dy() {
				img.Set(x, y, c)
			}
		}
	}
}

func abs(x int) int {
	if x < 0 {
		return -x
	}
	return x
}

func randomInt(min, max int) int {
	n, _ := rand.Int(rand.Reader, big.NewInt(int64(max-min+1)))
	return int(n.Int64()) + min
}

func encodeToBase64(img image.Image) (string, error) {
	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		return "", err
	}
	return base64.StdEncoding.EncodeToString(buf.Bytes()), nil
}

func generateID() (string, error) {
	b := make([]byte, 16)
	_, err := rand.Read(b)
	if err != nil {
		return "", err
	}
	return fmt.Sprintf("click_%x", b), nil
}
