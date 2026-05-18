package middleware

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"net/http"
	"strings"
	"sync"

	"github.com/gin-gonic/gin"
)

type CompressionType int

const (
	CompressionNone CompressionType = iota
	CompressionGzip
	CompressionDeflate
	CompressionBr
)

type CompressionStats struct {
	RequestsCompressed int64
	BytesSaved        int64
	CompressionRatio  float64
	mu                sync.RWMutex
}

var compressionStats = &CompressionStats{}

func GetCompressionStats() *CompressionStats {
	return compressionStats
}

type gzipResponseWriter struct {
	gin.ResponseWriter
	Writer *gzip.Writer
}

func (g *gzipResponseWriter) Write(data []byte) (int, error) {
	return g.Writer.Write(data)
}

func AdvancedCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldSkipCompression(c) {
			c.Next()
			return
		}

		encoding := c.GetHeader("Accept-Encoding")

		var compressionType CompressionType

		if strings.Contains(encoding, "br") {
			compressionType = CompressionBr
		} else if strings.Contains(encoding, "gzip") {
			compressionType = CompressionGzip
		} else if strings.Contains(encoding, "deflate") {
			compressionType = CompressionDeflate
		}

		if compressionType == CompressionNone {
			c.Next()
			return
		}

		c.Header("Content-Encoding", getCompressionHeader(compressionType))
		c.Header("Vary", "Accept-Encoding")

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipResponseWriter{Writer: gz, ResponseWriter: c.Writer}

		c.Next()
	}
}

func shouldSkipCompression(c *gin.Context) bool {
	path := c.Request.URL.Path
	if strings.HasPrefix(path, "/admin/") ||
		strings.HasPrefix(path, "/api/internal/") ||
		strings.HasPrefix(path, "/health") {
		return true
	}

	contentType := c.GetHeader("Content-Type")
	if !isCompressibleContentType(contentType) {
		return true
	}

	if c.Request.Method != http.MethodGet && c.Request.Method != http.MethodPost {
		return true
	}

	return false
}

func isCompressibleContentType(contentType string) bool {
	compressibleTypes := []string{
		"text/html",
		"text/plain",
		"text/css",
		"text/javascript",
		"application/javascript",
		"application/json",
		"application/xml",
		"text/xml",
		"application/xhtml+xml",
	}

	for _, t := range compressibleTypes {
		if strings.Contains(contentType, t) {
			return true
		}
	}

	return false
}

func gzipCompress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func deflateCompress(data []byte, level int) ([]byte, error) {
	var buf bytes.Buffer
	writer, err := zlib.NewWriterLevel(&buf, level)
	if err != nil {
		return nil, err
	}

	_, err = writer.Write(data)
	if err != nil {
		return nil, err
	}

	if err := writer.Close(); err != nil {
		return nil, err
	}

	return buf.Bytes(), nil
}

func getCompressionHeader(ct CompressionType) string {
	switch ct {
	case CompressionGzip:
		return "gzip"
	case CompressionDeflate:
		return "deflate"
	case CompressionBr:
		return "br"
	default:
		return ""
	}
}

type AdaptiveCompression struct {
	quality int
	minSize int
	maxSize int
	stats   *CompressionStats
	mu      sync.RWMutex
}

func NewAdaptiveCompression() *AdaptiveCompression {
	return &AdaptiveCompression{
		quality: 6,
		minSize: 1024,
		maxSize: 10 * 1024 * 1024,
		stats:   &CompressionStats{},
	}
}

func (ac *AdaptiveCompression) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if shouldSkipCompression(c) {
			c.Next()
			return
		}

		encoding := c.GetHeader("Accept-Encoding")

		var compressionType CompressionType

		if strings.Contains(encoding, "br") {
			compressionType = CompressionBr
		} else if strings.Contains(encoding, "gzip") {
			compressionType = CompressionGzip
		} else if strings.Contains(encoding, "deflate") {
			compressionType = CompressionDeflate
		}

		if compressionType == CompressionNone {
			c.Next()
			return
		}

		c.Header("Content-Encoding", getCompressionHeader(compressionType))
		c.Header("Vary", "Accept-Encoding")

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipResponseWriter{Writer: gz, ResponseWriter: c.Writer}

		c.Next()
	}
}

func (ac *AdaptiveCompression) SetQuality(quality int) {
	if quality < 1 {
		quality = 1
	}
	if quality > 9 {
		quality = 9
	}
	ac.quality = quality
}

func (ac *AdaptiveCompression) GetStats() *CompressionStats {
	ac.mu.RLock()
	defer ac.mu.RUnlock()
	return ac.stats
}

func (ac *AdaptiveCompression) ResetStats() {
	ac.mu.Lock()
	defer ac.mu.Unlock()
	ac.stats = &CompressionStats{}
}

type CompressionConfig struct {
	Enabled        bool
	MinSize        int
	MaxSize        int
	Quality        int
	DisableGzip    bool
	DisableBrotli  bool
	DisableDeflate bool
}

var globalCompressionConfig = &CompressionConfig{
	Enabled:        true,
	MinSize:        1024,
	MaxSize:        10 * 1024 * 1024,
	Quality:        6,
	DisableGzip:    false,
	DisableBrotli:  false,
	DisableDeflate: false,
}

func SetCompressionConfig(config *CompressionConfig) {
	if config == nil {
		return
	}
	globalCompressionConfig = config
}

func GetCompressionConfig() *CompressionConfig {
	return globalCompressionConfig
}

type CompressionMiddleware struct {
	config *CompressionConfig
}

func NewCompressionMiddleware() *CompressionMiddleware {
	return &CompressionMiddleware{
		config: globalCompressionConfig,
	}
}

func (cm *CompressionMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !cm.config.Enabled {
			c.Next()
			return
		}

		if shouldSkipCompression(c) {
			c.Next()
			return
		}

		encoding := c.GetHeader("Accept-Encoding")

		var compressionType CompressionType

		if strings.Contains(encoding, "br") && !cm.config.DisableBrotli {
			compressionType = CompressionBr
		} else if strings.Contains(encoding, "gzip") && !cm.config.DisableGzip {
			compressionType = CompressionGzip
		} else if strings.Contains(encoding, "deflate") && !cm.config.DisableDeflate {
			compressionType = CompressionDeflate
		}

		if compressionType == CompressionNone {
			c.Next()
			return
		}

		c.Header("Content-Encoding", getCompressionHeader(compressionType))
		c.Header("Vary", "Accept-Encoding")

		gz := gzip.NewWriter(c.Writer)
		defer gz.Close()

		c.Writer = &gzipResponseWriter{Writer: gz, ResponseWriter: c.Writer}

		c.Next()
	}
}

func CompressionMiddlewareHandler() gin.HandlerFunc {
	cm := NewCompressionMiddleware()
	return cm.Handler()
}
