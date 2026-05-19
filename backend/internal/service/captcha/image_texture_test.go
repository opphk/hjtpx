package captcha

import (
	"image"
	"image/color"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestImageGenerator_DrawDotPatternTexture(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	generator.drawDotPatternTexture(img)

	count := 0
	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			r, _, _, a := img.At(x, y).RGBA()
			if a > 0 && r > 0 {
				count++
			}
		}
	}

	assert.Greater(t, count, 1000, "dot pattern texture should have significant colored pixels")
}

func TestImageGenerator_DrawSpiralTexture(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	generator.drawSpiralTexture(img)

	brightnessValues := make([]float64, 0)
	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 {
				brightness := float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114
				brightnessValues = append(brightnessValues, brightness)
			}
		}
	}

	assert.Greater(t, len(brightnessValues), 0, "spiral texture should generate brightness variations")

	variance := calculateVariance(brightnessValues)
	assert.Greater(t, variance, 50.0, "spiral texture should have significant brightness variance")
}

func TestImageGenerator_DrawGridTexture(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	generator.drawGridTexture(img)

	gridLineCount := 0
	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			r, _, _, a := img.At(x, y).RGBA()
			if a > 0 && r > 150 {
				gridLineCount++
			}
		}
	}

	assert.Greater(t, gridLineCount, 100, "grid texture should have visible grid lines")
}

func TestImageGenerator_DrawRadialTexture(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	generator.drawRadialTexture(img)

	edgeBrightness := make([]float64, 0)
	for y := 0; y < 160; y++ {
		r, g, b, a := img.At(10, y).RGBA()
		if a > 0 {
			brightness := float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114
			edgeBrightness = append(edgeBrightness, brightness)
		}
	}

	centerBrightness := make([]float64, 0)
	for y := 0; y < 160; y++ {
		r, g, b, a := img.At(160, y).RGBA()
		if a > 0 {
			brightness := float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114
			centerBrightness = append(centerBrightness, brightness)
		}
	}

	edgeAvg := calculateAverage(edgeBrightness)
	centerAvg := calculateAverage(centerBrightness)

	assert.NotEqual(t, edgeAvg, centerAvg, "radial texture should have brightness difference between edge and center")
}

func TestImageGenerator_DrawEnhancedNoiseTexture(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	generator.drawEnhancedNoiseTexture(img)

	brightnessValues := make([]float64, 0)
	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			r, g, b, a := img.At(x, y).RGBA()
			if a > 0 {
				brightness := float64(r>>8)*0.299 + float64(g>>8)*0.587 + float64(b>>8)*0.114
				brightnessValues = append(brightnessValues, brightness)
			}
		}
	}

	variance := calculateVariance(brightnessValues)
	assert.Greater(t, variance, 45.0, "enhanced noise texture should have significant variance")

	assert.Greater(t, len(brightnessValues), 50000, "enhanced noise should cover entire image")
}

func TestImageGenerator_ApplyEdgeConsistencyEnhancement(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{R: 100, G: 100, B: 100, A: 255})
		}
	}

	gap := image.Rect(100, 50, 140, 90)
	result := generator.applyEdgeConsistencyEnhancement(img, gap)

	assert.NotNil(t, result, "edge consistency enhancement should return result image")

	for y := gap.Min.Y; y < gap.Max.Y; y++ {
		for x := gap.Min.X - 5; x <= gap.Min.X+5; x++ {
			if x >= 0 && x < 320 {
				_, _, _, a := result.At(x, y).RGBA()
				assert.Equal(t, uint32(0xffff), a, "enhanced pixels should have full alpha")
			}
		}
	}
}

func TestImageGenerator_ApplyMultiScaleEdgeDetection(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{R: 120, G: 120, B: 120, A: 255})
		}
	}

	gap := image.Rect(100, 50, 140, 90)
	result := generator.applyMultiScaleEdgeDetection(img, gap)

	assert.NotNil(t, result, "multi-scale edge detection should return result image")
}

func TestImageGenerator_CalculateEdgeConsistency(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			gray := uint8(100 + (x%20)*2)
			img.Set(x, y, color.RGBA{R: gray, G: gray, B: gray, A: 255})
		}
	}

	gap := image.Rect(100, 50, 140, 90)
	consistency := generator.calculateEdgeConsistency(img, 100, 70, gap, true)

	assert.GreaterOrEqual(t, consistency, 0.0, "consistency score should be non-negative")
	assert.LessOrEqual(t, consistency, 1.0, "consistency score should be at most 1.0")
}

func TestImageGenerator_CalculateEdgeAdjustmentAtScale(t *testing.T) {
	generator := NewImageGenerator()
	img := image.NewRGBA(image.Rect(0, 0, 320, 160))

	for y := 0; y < 160; y++ {
		for x := 0; x < 320; x++ {
			img.Set(x, y, color.RGBA{R: 110, G: 110, B: 110, A: 255})
		}
	}

	gap := image.Rect(100, 50, 140, 90)

	adjustment := generator.calculateEdgeAdjustmentAtScale(img, 100, 70, gap, 1)
	assert.GreaterOrEqual(t, adjustment, -0.3, "edge adjustment should be within expected range")
	assert.LessOrEqual(t, adjustment, 0.3, "edge adjustment should be within expected range")

	adjustment2 := generator.calculateEdgeAdjustmentAtScale(img, 200, 200, gap, 1)
	assert.Equal(t, 0.0, adjustment2, "non-edge pixels should have zero adjustment")
}

func calculateVariance(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	mean := 0.0
	for _, v := range values {
		mean += v
	}
	mean /= float64(len(values))

	variance := 0.0
	for _, v := range values {
		diff := v - mean
		variance += diff * diff
	}
	return variance / float64(len(values))
}

func calculateAverage(values []float64) float64 {
	if len(values) == 0 {
		return 0
	}
	sum := 0.0
	for _, v := range values {
		sum += v
	}
	return sum / float64(len(values))
}
