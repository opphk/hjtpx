package benchmark

import (
	"image"
	"image/color"
	"testing"
	"time"

	"captchax/internal/optimization"
)

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for x := 0; x < width; x++ {
		for y := 0; y < height; y++ {
			img.Set(x, y, color.RGBA{
				R: uint8(x % 256),
				G: uint8(y % 256),
				B: uint8((x + y) % 256),
				A: 255,
			})
		}
	}
	return img
}

func BenchmarkImageGeneration(b *testing.B) {
	compressor := optimization.NewImageCompressor()
	img := createTestImage(300, 150)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = compressor.CompressJPEG(img, 80)
	}
}

func BenchmarkCacheGet(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)
	cache.Set("test-key", []byte("test-data"))

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("test-key")
	}
}

func BenchmarkCacheSet(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		cache.Set("test-key", []byte("test-data"))
	}
}

func BenchmarkCacheGetMiss(b *testing.B) {
	cache := optimization.NewImageCache(1000, 10*time.Minute)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		_, _ = cache.Get("non-existent-key")
	}
}
