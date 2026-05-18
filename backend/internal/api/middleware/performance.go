package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w *responseWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func (w *responseWriter) WriteString(s string) (int, error) {
	w.body.WriteString(s)
	return w.ResponseWriter.WriteString(s)
}

type MiddlewarePerformanceStats struct {
	TotalRequests int64
	TotalDuration int64
	SlowRequests  int64
	RequestCount  map[string]int64
	mu            sync.RWMutex
}

var stats = &MiddlewarePerformanceStats{
	RequestCount: make(map[string]int64),
}

func PerformanceMonitoring() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		c.Next()

		duration := time.Since(start)
		stats.recordRequest(path, duration)

		c.Header("X-Response-Time", strconv.FormatInt(duration.Milliseconds(), 10)+"ms")
	}
}

func (ps *MiddlewarePerformanceStats) recordRequest(path string, duration time.Duration) {
	ps.mu.Lock()
	defer ps.mu.Unlock()

	ps.TotalRequests++
	ps.TotalDuration += duration.Milliseconds()
	ps.RequestCount[path]++

	if duration > 100*time.Millisecond {
		ps.SlowRequests++
	}
}

func GetMiddlewarePerformanceStats() map[string]interface{} {
	stats.mu.RLock()
	defer stats.mu.RUnlock()

	avgDuration := int64(0)
	if stats.TotalRequests > 0 {
		avgDuration = stats.TotalDuration / stats.TotalRequests
	}

	return map[string]interface{}{
		"total_requests":  stats.TotalRequests,
		"avg_duration_ms": avgDuration,
		"slow_requests":   stats.SlowRequests,
		"request_count":   stats.RequestCount,
	}
}

func GzipCompression() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("Accept-Encoding") == "" ||
			c.GetBool("gzip_disabled") ||
			c.GetHeader("Content-Encoding") != "" {
			c.Next()
			return
		}

		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)
		gz.Reset(c.Writer)

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		gw := &gzipWriter{Writer: gz, ResponseWriter: c.Writer}
		c.Writer = gw
		defer func() {
			gz.Close()
		}()

		c.Next()
	}
}

type gzipWriter struct {
	gin.ResponseWriter
	Writer io.Writer
	body   bytes.Buffer
}

func (g *gzipWriter) Write(b []byte) (int, error) {
	g.body.Write(b)
	return g.Writer.Write(b)
}

func (g *gzipWriter) WriteString(s string) (int, error) {
	g.body.WriteString(s)
	return io.WriteString(g.Writer, s)
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		return gzip.NewWriter(nil)
	},
}

func CacheControl(maxAge time.Duration) gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(maxAge.Seconds())))
		c.Header("Expires", time.Now().Add(maxAge).Format(http.TimeFormat))
		c.Header("Pragma", "cache")
		c.Next()
	}
}

func NoCache() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		c.Header("Pragma", "no-cache")
		c.Header("Expires", "0")
		c.Next()
	}
}

func RequestID() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.GetHeader("X-Request-ID") == "" {
			id := generateID()
			c.Header("X-Request-ID", id)
			c.Set("RequestID", id)
		}
		c.Next()
	}
}

func generateID() string {
	return strconv.FormatInt(time.Now().UnixNano(), 36)
}
