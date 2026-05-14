package image

import (
	"bytes"
	"context"
	"errors"
	"image"
	"image/jpeg"
	"image/png"
	"sync"
	"sync/atomic"
	"time"
)

type Format string

const (
	FormatPNG  Format = "png"
	FormatJPEG Format = "jpeg"
	FormatWebP Format = "webp"
)

type CompressConfig struct {
	Format           Format
	Quality          int
	MaxWidth         int
	MaxHeight        int
	EnableProgressive bool
	StripMetadata    bool
}

type CompressResult struct {
	Data   []byte
	Format Format
	Size   int
}

var (
	ErrUnsupportedFormat = errors.New("unsupported image format")
	ErrImageTooLarge     = errors.New("image too large")
	ErrInvalidImage      = errors.New("invalid image data")
)

type CompressStats struct {
	TotalProcessed int64
	TotalTimeMs    int64
	AvgTimeMs      float64
	BytesSaved     int64
	BytesIn        int64
	BytesOut       int64
}

type Compressor struct {
	config CompressConfig
	stats  struct {
		totalProcessed atomic.Int64
		totalTimeMs    atomic.Int64
		avgTimeMs      atomic.Int64
		bytesSaved     atomic.Int64
		bytesIn        atomic.Int64
		bytesOut       atomic.Int64
	}
}

func NewCompressor(config CompressConfig) *Compressor {
	if config.Quality <= 0 {
		config.Quality = 85
	}
	if config.Quality > 100 {
		config.Quality = 100
	}

	return &Compressor{
		config: config,
	}
}

func (c *Compressor) Compress(img image.Image) (*CompressResult, error) {
	if img == nil {
		return nil, ErrInvalidImage
	}

	startTime := time.Now()

	var buf bytes.Buffer

	switch c.config.Format {
	case FormatPNG:
		encoder := &png.Encoder{
			CompressionLevel: png.DefaultCompression,
		}
		if err := encoder.Encode(&buf, img); err != nil {
			return nil, err
		}
	case FormatJPEG:
		options := &jpeg.Options{Quality: c.config.Quality}
		if err := jpeg.Encode(&buf, img, options); err != nil {
			return nil, err
		}
	case FormatWebP:
		data, err := EncodeWebP(img, c.config.Quality)
		if err != nil {
			data, err = encodeWebPFallback(img, c.config.Quality)
			if err != nil {
				return nil, err
			}
		}
		buf.Write(data)
	default:
		if err := png.Encode(&buf, img); err != nil {
			return nil, err
		}
	}

	data := buf.Bytes()
	originalSize := len(data)

	if c.config.MaxWidth > 0 || c.config.MaxHeight > 0 {
		img = c.resizeIfNeeded(img)
	}

	c.stats.totalProcessed.Add(1)
	c.stats.bytesIn.Add(int64(originalSize))
	c.stats.bytesOut.Add(int64(len(data)))
	c.stats.bytesSaved.Add(int64(originalSize - len(data)))

	elapsed := time.Since(startTime)
	c.stats.totalTimeMs.Add(elapsed.Milliseconds())

	return &CompressResult{
		Data:   data,
		Format: c.config.Format,
		Size:   len(data),
	}, nil
}

func (c *Compressor) CompressWithContext(ctx context.Context, img image.Image) (*CompressResult, error) {
	select {
	case <-ctx.Done():
		return nil, ctx.Err()
	default:
		return c.Compress(img)
	}
}

func (c *Compressor) resizeIfNeeded(img image.Image) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if c.config.MaxWidth > 0 && width > c.config.MaxWidth {
		ratio := float64(c.config.MaxWidth) / float64(width)
		height = int(float64(height) * ratio)
		width = c.config.MaxWidth
	}

	if c.config.MaxHeight > 0 && height > c.config.MaxHeight {
		ratio := float64(c.config.MaxHeight) / float64(height)
		width = int(float64(width) * ratio)
		height = c.config.MaxHeight
	}

	if width == bounds.Dx() && height == bounds.Dy() {
		return img
	}

	result := image.NewRGBA(image.Rect(0, 0, width, height))
	scaleX := float64(bounds.Dx()) / float64(width)
	scaleY := float64(bounds.Dy()) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)
			result.Set(x, y, img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y))
		}
	}

	return result
}

func (c *Compressor) GetStats() CompressStats {
	total := c.stats.totalProcessed.Load()
	totalTime := c.stats.totalTimeMs.Load()
	avgTime := float64(0)
	if total > 0 {
		avgTime = float64(totalTime) / float64(total)
	}
	return CompressStats{
		TotalProcessed: total,
		TotalTimeMs:    totalTime,
		AvgTimeMs:      avgTime,
		BytesSaved:     c.stats.bytesSaved.Load(),
		BytesIn:        c.stats.bytesIn.Load(),
		BytesOut:       c.stats.bytesOut.Load(),
	}
}

func (c *Compressor) avgTimeMs() *atomic.Int64 {
	return &c.stats.totalTimeMs
}

func EncodeWebP(img image.Image, quality int) ([]byte, error) {
	if img == nil {
		return nil, ErrInvalidImage
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= 0 || height <= 0 {
		return nil, ErrInvalidImage
	}

	rgbaImg, ok := img.(*image.RGBA)
	if !ok {
		rgba := image.NewRGBA(bounds)
		for y := bounds.Min.Y; y < bounds.Max.Y; y++ {
			for x := bounds.Min.X; x < bounds.Max.X; x++ {
				rgba.Set(x, y, img.At(x, y))
			}
		}
		rgbaImg = rgba
	}

	return encodeWebPInternal(rgbaImg, quality)
}

func encodeWebPInternal(img *image.RGBA, quality int) ([]byte, error) {
	var buf bytes.Buffer

	webpData, err := encodeSimpleWebP(img, quality)
	if err != nil {
		return nil, err
	}

	chunkSize := uint32(len(webpData) - 4)
	buf.WriteString("RIFF")
	buf.WriteByte(byte(chunkSize & 0xFF))
	buf.WriteByte(byte((chunkSize >> 8) & 0xFF))
	buf.WriteByte(byte((chunkSize >> 16) & 0xFF))
	buf.WriteByte(byte((chunkSize >> 24) & 0xFF))
	buf.Write(webpData)

	return buf.Bytes(), nil
}

func encodeSimpleWebP(img *image.RGBA, quality int) ([]byte, error) {
	_ = img.Bounds()

	var buf bytes.Buffer
	buf.WriteString("WEBP")
	buf.WriteString("VP8 ")

	vp8Data := encodeVP8Simple(img)
	vp8Len := uint32(len(vp8Data))
	buf.WriteByte(byte(vp8Len & 0xFF))
	buf.WriteByte(byte((vp8Len >> 8) & 0xFF))
	buf.WriteByte(byte((vp8Len >> 16) & 0xFF))
	buf.WriteByte(byte((vp8Len >> 24) & 0xFF))
	buf.Write(vp8Data)

	return buf.Bytes(), nil
}

func encodeVP8Simple(img *image.RGBA) []byte {
	width := img.Bounds().Dx()
	height := img.Bounds().Dy()

	frameHeader := make([]byte, 10)
	frameHeader[0] = 0x9D
	frameHeader[1] = 0x01
	frameHeader[2] = 0x2A

	frameHeader[3] = byte(width & 0xFF)
	frameHeader[4] = byte((width >> 8) & 0xFF)
	frameHeader[5] = byte((width >> 16) & 0x3F)
	frameHeader[6] = byte(height & 0xFF)
	frameHeader[7] = byte((height >> 8) & 0xFF)
	frameHeader[8] = byte((height >> 16) & 0x3F)
	frameHeader[9] = 0x01

	pix := img.Pix
	result := make([]byte, 10+width*height*3/2)

	copy(result, frameHeader)
	offset := 10

	for y := 0; y < height; y += 2 {
		result[offset] = 0x00
		offset++

		for x := 0; x < width; x += 2 {
			idx := y*img.Stride + x*4
			if idx+3 >= len(pix) {
				offset += 3
				continue
			}

			r := int(pix[idx])
			g := int(pix[idx+1])
			b := int(pix[idx+2])

			y1 := ((66*r + 129*g + 25*b + 128) >> 8) + 16
			u := ((-38*r - 74*g + 112*b + 128) >> 8) + 128
			_ = ((112*r - 94*g - 18*b + 128) >> 8) + 128

			result[offset] = byte(y1)
			offset++
			result[offset] = byte(y1)
			offset++
			result[offset] = byte(u)
			offset++
		}
	}

	return result
}

func encodeWebPFallback(img image.Image, quality int) ([]byte, error) {
	var buf bytes.Buffer
	if err := jpeg.Encode(&buf, img, &jpeg.Options{Quality: quality}); err != nil {
		return nil, err
	}
	return buf.Bytes(), nil
}

type AdaptiveCompressor struct {
	compressors map[Format]*Compressor
	mu          sync.RWMutex
	stats       map[Format]CompressStats
}

func NewAdaptiveCompressor() *AdaptiveCompressor {
	ac := &AdaptiveCompressor{
		compressors: make(map[Format]*Compressor),
		stats:       make(map[Format]CompressStats),
	}

	ac.compressors[FormatWebP] = NewCompressor(CompressConfig{
		Format:  FormatWebP,
		Quality: 80,
	})

	ac.compressors[FormatPNG] = NewCompressor(CompressConfig{
		Format:  FormatPNG,
		Quality: 6,
	})

	ac.compressors[FormatJPEG] = NewCompressor(CompressConfig{
		Format:  FormatJPEG,
		Quality: 85,
	})

	return ac
}

func (ac *AdaptiveCompressor) Compress(img image.Image, format Format) (*CompressResult, error) {
	ac.mu.RLock()
	compressor, ok := ac.compressors[format]
	ac.mu.RUnlock()

	if !ok {
		ac.mu.RLock()
		compressor = ac.compressors[FormatWebP]
		ac.mu.RUnlock()
	}

	return compressor.Compress(img)
}

func (ac *AdaptiveCompressor) GetStats() map[Format]CompressStats {
	ac.mu.RLock()
	defer ac.mu.RUnlock()

	result := make(map[Format]CompressStats)
	for f, c := range ac.compressors {
		result[f] = c.GetStats()
	}
	return result
}

func Decode(data []byte) (image.Image, error) {
	return png.Decode(bytes.NewReader(data))
}

func DecodeConfig(data []byte) (image.Config, error) {
	return png.DecodeConfig(bytes.NewReader(data))
}

func DecodeWebP(data []byte) (image.Image, error) {
	if len(data) < 30 {
		return nil, ErrInvalidImage
	}

	if string(data[0:4]) != "RIFF" || string(data[8:12]) != "WEBP" {
		return nil, ErrInvalidImage
	}

	if string(data[12:16]) == "VP8 " {
		return decodeVP8Frame(data[16:])
	}

	return nil, ErrUnsupportedFormat
}

func decodeVP8Frame(data []byte) (image.Image, error) {
	if len(data) < 10 {
		return nil, ErrInvalidImage
	}

	width := int(data[7]) | ((int(data[8]) & 0x3F) << 8)
	height := int(data[9]) | ((int(data[10]) & 0x3F) << 8)

	if width <= 0 || height <= 0 || width > 16383 || height > 16383 {
		return nil, ErrInvalidImage
	}

	img := image.NewRGBA(image.Rect(0, 0, width, height))
	pix := img.Pix

	offset := 10
	pixelIndex := 0

	for y := 0; y < height && offset < len(data); y += 2 {
		offset++
		for x := 0; x < width && offset+2 < len(data); x += 2 {
			if offset+2 >= len(data) {
				break
			}

			y1 := int(data[offset])
			y2 := int(data[offset+1])
			u := int(data[offset+2])
			offset += 3

			r1 := int(float64(y1) + 1.402*(float64(u)-128))
			g1 := int(float64(y1) - 0.344136*(float64(u)-128) - 0.714136*(float64(u)-128))
			b1 := int(float64(y1) + 1.772*(float64(u)-128))

			r2 := int(float64(y2) + 1.402*(float64(u)-128))
			g2 := int(float64(y2) - 0.344136*(float64(u)-128) - 0.714136*(float64(u)-128))
			b2 := int(float64(y2) + 1.772*(float64(u)-128))

			clamp := func(v int) uint8 {
				if v < 0 {
					return 0
				}
				if v > 255 {
					return 255
				}
				return uint8(v)
			}

			pix[pixelIndex] = clamp(r1)
			pix[pixelIndex+1] = clamp(g1)
			pix[pixelIndex+2] = clamp(b1)
			pix[pixelIndex+3] = 255
			pixelIndex += 4

			if x+1 < width {
				pix[pixelIndex] = clamp(r2)
				pix[pixelIndex+1] = clamp(g2)
				pix[pixelIndex+2] = clamp(b2)
				pix[pixelIndex+3] = 255
				pixelIndex += 4
			}
		}
	}

	return img, nil
}

func resizeImage(img image.Image, width, height int) image.Image {
	if width <= 0 || height <= 0 {
		return img
	}

	bounds := img.Bounds()
	srcWidth := bounds.Dx()
	srcHeight := bounds.Dy()

	if srcWidth == width && srcHeight == height {
		return img
	}

	result := image.NewRGBA(image.Rect(0, 0, width, height))
	scaleX := float64(srcWidth) / float64(width)
	scaleY := float64(srcHeight) / float64(height)

	for y := 0; y < height; y++ {
		for x := 0; x < width; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)
			if srcX >= srcWidth {
				srcX = srcWidth - 1
			}
			if srcY >= srcHeight {
				srcY = srcHeight - 1
			}
			result.Set(x, y, img.At(srcX+bounds.Min.X, srcY+bounds.Min.Y))
		}
	}

	return result
}
