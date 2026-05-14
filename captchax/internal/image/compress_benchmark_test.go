package image

import (
	"bytes"
	"image"
	"image/color"
	"image/png"
	"testing"
)

func createTestImage(width, height int) image.Image {
	img := image.NewRGBA(image.Rect(0, 0, width, height))
	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
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

func BenchmarkCompressPNG(b *testing.B) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatPNG,
		Quality: 6,
	})

	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compressor.Compress(img)
		}
	})
}

func BenchmarkCompressJPEG(b *testing.B) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatJPEG,
		Quality: 85,
	})

	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compressor.Compress(img)
		}
	})
}

func BenchmarkCompressWebP(b *testing.B) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatWebP,
		Quality: 80,
	})

	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compressor.Compress(img)
		}
	})
}

func BenchmarkEncodeWebP(b *testing.B) {
	img := createTestImage(300, 200)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		EncodeWebP(img, 80)
	}
}

func BenchmarkAdaptiveCompressor(b *testing.B) {
	compressor := NewAdaptiveCompressor()
	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compressor.Compress(img, FormatWebP)
		}
	})
}

func TestCompressorStats(t *testing.T) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatPNG,
		Quality: 6,
	})

	img := createTestImage(100, 100)

	for i := 0; i < 10; i++ {
		compressor.Compress(img)
	}

	stats := compressor.GetStats()
	if stats.TotalProcessed != 10 {
		t.Errorf("expected 10 processed, got %d", stats.TotalProcessed)
	}
}

func TestCompressWithResize(t *testing.T) {
	compressor := NewCompressor(CompressConfig{
		Format:    FormatPNG,
		MaxWidth:  100,
		MaxHeight: 100,
	})

	img := createTestImage(400, 300)

	result, err := compressor.Compress(img)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if result == nil {
		t.Error("expected result to not be nil")
	}
}

func TestEncodeWebPValidation(t *testing.T) {
	_, err := EncodeWebP(nil, 80)
	if err != ErrInvalidImage {
		t.Errorf("expected ErrInvalidImage, got %v", err)
	}

	img := image.NewRGBA(image.Rect(0, 0, 0, 0))
	_, err = EncodeWebP(img, 80)
	if err != ErrInvalidImage {
		t.Errorf("expected ErrInvalidImage for zero-size image, got %v", err)
	}
}

func TestDecodeWebP(t *testing.T) {
	img := createTestImage(100, 100)
	data, err := EncodeWebP(img, 80)
	if err != nil {
		t.Fatalf("failed to encode WebP: %v", err)
	}

	decoded, err := DecodeWebP(data)
	if err != nil {
		t.Fatalf("failed to decode WebP: %v", err)
	}

	if decoded == nil {
		t.Error("expected decoded image to not be nil")
	}
}

func TestDecodeWebPInvalid(t *testing.T) {
	_, err := DecodeWebP([]byte("short"))
	if err != ErrInvalidImage {
		t.Errorf("expected ErrInvalidImage, got %v", err)
	}

	invalidData := []byte("RIFFxxxxWEBPxxxx")
	_, err = DecodeWebP(invalidData)
	if err != ErrInvalidImage {
		t.Errorf("expected ErrInvalidImage for invalid WebP, got %v", err)
	}
}

func BenchmarkResizeImage(b *testing.B) {
	img := createTestImage(800, 600)

	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resizeImage(img, 400, 300)
	}
}

func TestResizeImage(t *testing.T) {
	img := createTestImage(400, 300)

	resized := resizeImage(img, 200, 150)
	if resized.Bounds().Dx() != 200 {
		t.Errorf("expected width 200, got %d", resized.Bounds().Dx())
	}
	if resized.Bounds().Dy() != 150 {
		t.Errorf("expected height 150, got %d", resized.Bounds().Dy())
	}
}

func TestResizeImageNoOp(t *testing.T) {
	img := createTestImage(200, 150)

	resized := resizeImage(img, 200, 150)
	if resized != img {
		t.Error("expected same image when dimensions match")
	}
}

func BenchmarkBytesAllocation(b *testing.B) {
	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf bytes.Buffer
			png.Encode(&buf, img)
		}
	})
}

func TestCompressResult(t *testing.T) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatPNG,
		Quality: 6,
	})

	img := createTestImage(100, 100)

	result, err := compressor.Compress(img)
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}

	if result.Format != FormatPNG {
		t.Errorf("expected format PNG, got %s", result.Format)
	}
	if len(result.Data) == 0 {
		t.Error("expected non-empty data")
	}
	if result.Size != len(result.Data) {
		t.Errorf("expected size %d, got %d", len(result.Data), result.Size)
	}
}

func BenchmarkConcurrentCompress(b *testing.B) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatJPEG,
		Quality: 85,
	})

	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			compressor.Compress(img)
		}
	})
}

func TestAdaptiveCompressorStats(t *testing.T) {
	compressor := NewAdaptiveCompressor()
	img := createTestImage(100, 100)

	compressor.Compress(img, FormatPNG)
	compressor.Compress(img, FormatJPEG)
	compressor.Compress(img, FormatWebP)

	stats := compressor.GetStats()
	if len(stats) != 3 {
		t.Errorf("expected 3 formats in stats, got %d", len(stats))
	}
}

func BenchmarkEncodeWebPVaryingQuality(b *testing.B) {
	img := createTestImage(200, 150)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			EncodeWebP(img, 50)
		}
	})
}

func TestCompressStatsBytes(t *testing.T) {
	compressor := NewCompressor(CompressConfig{
		Format:  FormatJPEG,
		Quality: 85,
	})

	img := createTestImage(200, 150)
	compressor.Compress(img)

	stats := compressor.GetStats()
	if stats.BytesIn == 0 {
		t.Error("expected BytesIn to be set")
	}
	if stats.BytesOut == 0 {
		t.Error("expected BytesOut to be set")
	}
}

func BenchmarkImageCreation(b *testing.B) {
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		createTestImage(300, 200)
	}
}

func TestCompressQualityBounds(t *testing.T) {
	img := createTestImage(100, 100)

	compressor := NewCompressor(CompressConfig{
		Format:  FormatJPEG,
		Quality: 0,
	})
	result, _ := compressor.Compress(img)
	if result.Size == 0 {
		t.Error("expected non-empty result with quality 0")
	}

	compressor = NewCompressor(CompressConfig{
		Format:  FormatJPEG,
		Quality: 200,
	})
	result, _ = compressor.Compress(img)
	if result.Size == 0 {
		t.Error("expected non-empty result with quality 200")
	}
}

func BenchmarkPNGEncoder(b *testing.B) {
	img := createTestImage(300, 200)

	b.ResetTimer()
	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			var buf bytes.Buffer
			encoder := &png.Encoder{CompressionLevel: png.DefaultCompression}
			encoder.Encode(&buf, img)
		}
	})
}
