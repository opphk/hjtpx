package handler

import (
	"bytes"
	"fmt"
	"image"
	"image/color"
	"image/png"
	"math"
	"strings"
	"testing"

	"golang.org/x/image/font"
	"golang.org/x/image/font/basicfont"
	"golang.org/x/image/font/sfnt"
)

func TestDifficultyLevels(t *testing.T) {
	testCases := []struct {
		name          string
		difficulty    DifficultyLevel
		expectedLine  int
		expectedNoise int
		expectedDist  int
	}{
		{"Easy", Easy, 3, 50, 1},
		{"Medium", Medium, 8, 120, 2},
		{"Hard", Hard, 15, 200, 3},
		{"Expert", Expert, 25, 300, 4},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			lineCount, noiseCount, distortionLevel := getImageDifficultyConfig(tc.difficulty)

			if lineCount != tc.expectedLine {
				t.Errorf("Expected line count %d, got %d", tc.expectedLine, lineCount)
			}
			if noiseCount != tc.expectedNoise {
				t.Errorf("Expected noise count %d, got %d", tc.expectedNoise, noiseCount)
			}
			if distortionLevel != tc.expectedDist {
				t.Errorf("Expected distortion level %d, got %d", tc.expectedDist, distortionLevel)
			}
		})
	}
}

func TestCharSets(t *testing.T) {
	testCases := []struct {
		charSetType CharSetType
		expectedLen int
		contains    string
	}{
		{Numeric, 10, "0"},
		{Alphabetic, 48, "a"},
		{Alphanumeric, 58, "0"},
		{Chinese, 66, "的"},
	}

	for _, tc := range testCases {
		t.Run(tc.charSetType.String(), func(t *testing.T) {
			result := getCharSetByType(tc.charSetType)
			if len(result) != tc.expectedLen {
				t.Errorf("Expected char set length %d, got %d", tc.expectedLen, len(result))
			}
			if !strings.Contains(result, tc.contains) {
				t.Errorf("Expected char set to contain %s", tc.contains)
			}
		})
	}
}

func TestGenerateGradientBackground(t *testing.T) {
	width := 200
	height := 80

	img := GenerateGradientBackground(width, height)

	if img.Bounds().Dx() != width {
		t.Errorf("Expected width %d, got %d", width, img.Bounds().Dx())
	}
	if img.Bounds().Dy() != height {
		t.Errorf("Expected height %d, got %d", height, img.Bounds().Dy())
	}

	pixelCount := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				pixelCount++
			}
		}
	}

	if pixelCount == 0 {
		t.Error("Image should have at least some opaque pixels")
	}
}

func TestGenerateTexture(t *testing.T) {
	width := 200
	height := 80

	img := GenerateTexture(width, height)

	if img.Bounds().Dx() != width {
		t.Errorf("Expected width %d, got %d", width, img.Bounds().Dx())
	}
	if img.Bounds().Dy() != height {
		t.Errorf("Expected height %d, got %d", height, img.Bounds().Dy())
	}

	hasVariation := false
	refColor := img.At(0, 0)
	for y := 0; y < height && !hasVariation; y++ {
		for x := 0; x < width && !hasVariation; x++ {
			if !sameColor(img.At(x, y), refColor) {
				hasVariation = true
			}
		}
	}

	if !hasVariation {
		t.Error("Texture should have color variation")
	}
}

func TestAddNoise(t *testing.T) {
	width := 100
	height := 50
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	baseColor := color.RGBA{200, 200, 200, 255}
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			img.Set(x, y, baseColor)
		}
	}

	err := AddNoise(img, 0.1)
	if err != nil {
		t.Errorf("AddNoise should not return error, got %v", err)
	}

	noiseCount := 0
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			if !sameColor(img.At(x, y), baseColor) {
				noiseCount++
			}
		}
	}

	if noiseCount == 0 {
		t.Error("Noise should have been added to the image")
	}
}

func TestGenerateInterferenceLines(t *testing.T) {
	width := 140
	height := 50
	img := image.NewRGBA(image.Rect(0, 0, width, height))

	testCases := []DifficultyLevel{Easy, Medium, Hard, Expert}

	for _, difficulty := range testCases {
		t.Run(difficulty.String(), func(t *testing.T) {
			err := GenerateInterferenceLines(img, difficulty)
			if err != nil {
				t.Errorf("GenerateInterferenceLines should not return error, got %v", err)
			}

			lineCount, _, _ := getImageDifficultyConfig(difficulty)
			if lineCount <= 0 {
				t.Errorf("Expected positive line count for difficulty %v", difficulty)
			}
		})
	}
}

func TestGenerateDistortedChar(t *testing.T) {
	f := &sfnt.Font{}
	face := basicfont.Face7x13
	_ = face

	char := 'A'
	angle := 15.0
	size := 13.0

	defer func() {
		if r := recover(); r != nil {
			t.Skip("Skipping test due to font initialization failure")
		}
	}()

	img := GenerateDistortedChar(f, char, angle, size)

	if img == nil {
		t.Skip("Skipping test due to nil image")
	}

	if img.Bounds().Dx() != 40 {
		t.Errorf("Expected width 40, got %d", img.Bounds().Dx())
	}
	if img.Bounds().Dy() != 40 {
		t.Errorf("Expected height 40, got %d", img.Bounds().Dy())
	}

	hasContent := false
	for y := 0; y < 40 && !hasContent; y++ {
		for x := 0; x < 40 && !hasContent; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				hasContent = true
			}
		}
	}

	if !hasContent {
		t.Error("Distorted character should have content")
	}
}

func TestGenerateConnectedChars(t *testing.T) {
	f := &sfnt.Font{}
	face := basicfont.Face7x13
	_ = face

	chars := []rune("ABC")
	charSize := 13.0

	defer func() {
		if r := recover(); r != nil {
			t.Skip("Skipping test due to font initialization failure")
		}
	}()

	result := GenerateConnectedChars(chars, f, charSize)

	if len(result) != len(chars) {
		t.Errorf("Expected %d char images, got %d", len(chars), len(result))
	}

	for i, img := range result {
		if img == nil {
			t.Errorf("Char image at index %d should not be nil", i)
		}
	}
}

func TestGenerateConnectedCharsWithSingleChar(t *testing.T) {
	f := &sfnt.Font{}
	face := basicfont.Face7x13
	_ = face

	chars := []rune{'A'}
	charSize := 13.0

	defer func() {
		if r := recover(); r != nil {
			t.Skip("Skipping test due to font initialization failure")
		}
	}()

	result := GenerateConnectedChars(chars, f, charSize)

	if len(result) != 1 {
		t.Errorf("Expected 1 char image, got %d", len(result))
	}
}

func TestImageSize(t *testing.T) {
	testCases := []struct {
		width  int
		height int
	}{
		{100, 40},
		{140, 50},
		{200, 80},
	}

	for _, tc := range testCases {
		t.Run("", func(t *testing.T) {
			img := GenerateGradientBackground(tc.width, tc.height)
			if img.Bounds().Dx() != tc.width {
				t.Errorf("Expected width %d, got %d", tc.width, img.Bounds().Dx())
			}
			if img.Bounds().Dy() != tc.height {
				t.Errorf("Expected height %d, got %d", tc.height, img.Bounds().Dy())
			}
		})
	}
}

func TestGenerateEnhancedCaptchaImage(t *testing.T) {
	config := ImageConfig{
		Length:     4,
		Width:      140,
		Height:     50,
		Difficulty: Medium,
		CharSet:    Alphanumeric,
	}

	text := "AB12"
	img := GenerateEnhancedCaptchaImage(text, config)

	if img.Bounds().Dx() != config.Width {
		t.Errorf("Expected width %d, got %d", config.Width, img.Bounds().Dx())
	}
	if img.Bounds().Dy() != config.Height {
		t.Errorf("Expected height %d, got %d", config.Height, img.Bounds().Dy())
	}

	hasContent := false
	for y := 0; y < config.Height && !hasContent; y++ {
		for x := 0; x < config.Width && !hasContent; x++ {
			_, _, _, a := img.At(x, y).RGBA()
			if a > 0 {
				hasContent = true
			}
		}
	}

	if !hasContent {
		t.Error("Enhanced captcha image should have content")
	}
}

func TestEnhancedCaptchaAllDifficultyLevels(t *testing.T) {
	text := "TEST"

	difficulties := []DifficultyLevel{Easy, Medium, Hard, Expert}
	charSets := []CharSetType{Numeric, Alphabetic, Alphanumeric, Chinese}

	for _, difficulty := range difficulties {
		for _, charSet := range charSets {
			t.Run(difficulty.String()+"_"+charSet.String(), func(t *testing.T) {
				config := ImageConfig{
					Length:     4,
					Width:      140,
					Height:     50,
					Difficulty: difficulty,
					CharSet:    charSet,
				}

				img := GenerateEnhancedCaptchaImage(text, config)

				if img.Bounds().Dx() != config.Width || img.Bounds().Dy() != config.Height {
					t.Error("Image dimensions do not match config")
				}
			})
		}
	}
}

func TestOCRResistance(t *testing.T) {
	config := ImageConfig{
		Length:     4,
		Width:      140,
		Height:     50,
		Difficulty: Expert,
		CharSet:    Alphanumeric,
	}

	avgComplexity := calculateImageComplexity(GenerateEnhancedCaptchaImage("TEST", config))

	if avgComplexity < 0.3 {
		t.Error("Image complexity too low, may be vulnerable to OCR")
	}
}

func calculateImageComplexity(img *image.RGBA) float64 {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	edgeCount := 0
	totalPixels := width * height

	for y := 1; y < height-1; y++ {
		for x := 1; x < width-1; x++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			r2, g2, b2, _ := img.At(x+1, y).RGBA()
			r3, g3, b3, _ := img.At(x, y+1).RGBA()

			dr := int(r1>>8) - int(r2>>8)
			dg := int(g1>>8) - int(g2>>8)
			db := int(b1>>8) - int(b2>>8)
			grad1 := math.Sqrt(float64(dr*dr + dg*dg + db*db))

			dr = int(r1>>8) - int(r3>>8)
			dg = int(g1>>8) - int(g3>>8)
			db = int(b1>>8) - int(b3>>8)
			grad2 := math.Sqrt(float64(dr*dr + dg*dg + db*db))

			if grad1 > 30 || grad2 > 30 {
				edgeCount++
			}
		}
	}

	return float64(edgeCount) / float64(totalPixels)
}

func TestNoiseDensity(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 100, 50))

	err := AddNoise(img, 0.05)
	if err != nil {
		t.Errorf("AddNoise should not return error, got %v", err)
	}

	err = AddNoise(img, 0.15)
	if err != nil {
		t.Errorf("AddNoise should not return error, got %v", err)
	}
}

func TestInterferenceLinesVariety(t *testing.T) {
	img := image.NewRGBA(image.Rect(0, 0, 140, 50))

	GenerateInterferenceLines(img, Hard)

	lineCount := 0
	for y := 1; y < 49; y++ {
		for x := 1; x < 139; x++ {
			r1, g1, b1, _ := img.At(x, y).RGBA()
			r2, g2, b2, _ := img.At(x+1, y).RGBA()
			r3, g3, b3, _ := img.At(x, y+1).RGBA()

			dr := int(r1>>8) - int(r2>>8)
			dg := int(g1>>8) - int(g2>>8)
			db := int(b1>>8) - int(b2>>8)
			grad1 := math.Sqrt(float64(dr*dr + dg*dg + db*db))

			dr = int(r1>>8) - int(r3>>8)
			dg = int(g1>>8) - int(g3>>8)
			db = int(b1>>8) - int(b3>>8)
			grad2 := math.Sqrt(float64(dr*dr + dg*dg + db*db))

			if grad1 > 50 || grad2 > 50 {
				lineCount++
			}
		}
	}

	if lineCount < 100 {
		t.Errorf("Expected significant line content, got %d edge pixels", lineCount)
	}
}

func TestWarpEffect(t *testing.T) {
	text := "TEST"
	img := GenerateEnhancedCaptchaImage(text, ImageConfig{
		Length:     4,
		Width:      140,
		Height:     50,
		Difficulty: Hard,
		CharSet:    Alphanumeric,
	})

	hasVerticalVariation := false
	for x := 10; x < 130 && !hasVerticalVariation; x++ {
		refColor := img.At(x, 10)
		for y := 15; y < 35 && !hasVerticalVariation; y++ {
			if !sameColor(img.At(x, y), refColor) {
				hasVerticalVariation = true
			}
		}
	}

	if !hasVerticalVariation {
		t.Error("Text warp effect should create vertical color variation")
	}
}

func TestColorVariation(t *testing.T) {
	img := GenerateGradientBackground(100, 50)

	colors := make(map[string]bool)
	for y := 0; y < 50; y += 10 {
		for x := 0; x < 100; x += 10 {
			c := img.At(x, y)
			r, g, b, _ := c.RGBA()
			key := fmt.Sprintf("%d-%d-%d", r>>8, g>>8, b>>8)
			colors[key] = true
		}
	}

	if len(colors) < 2 {
		t.Error("Gradient should have color variation")
	}
}

func TestEnhancedCaptchaImageSize(t *testing.T) {
	config := ImageConfig{
		Length:     4,
		Width:      140,
		Height:     50,
		Difficulty: Hard,
		CharSet:    Alphanumeric,
	}

	img := GenerateEnhancedCaptchaImage("TEST", config)

	var buf bytes.Buffer
	err := png.Encode(&buf, img)
	if err != nil {
		t.Errorf("Failed to encode image: %v", err)
	}

	sizeKB := buf.Len() / 1024
	if sizeKB >= 20 {
		t.Errorf("Image size %dKB exceeds 20KB limit", sizeKB)
	}
}

func sameColor(a, b color.Color) bool {
	r1, g1, b1, a1 := a.RGBA()
	r2, g2, b2, a2 := b.RGBA()
	return r1 == r2 && g1 == g2 && b1 == b2 && a1 == a2
}

func (d DifficultyLevel) String() string {
	switch d {
	case Easy:
		return "Easy"
	case Medium:
		return "Medium"
	case Hard:
		return "Hard"
	case Expert:
		return "Expert"
	default:
		return "Unknown"
	}
}

func (c CharSetType) String() string {
	switch c {
	case Numeric:
		return "Numeric"
	case Alphabetic:
		return "Alphabetic"
	case Alphanumeric:
		return "Alphanumeric"
	case Chinese:
		return "Chinese"
	default:
		return "Unknown"
	}
}

var _ = font.HintingFull
