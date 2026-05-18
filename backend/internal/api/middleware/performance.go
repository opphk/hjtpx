package middleware

import (
	"bytes"
	"compress/gzip"
	"io"
	"net/http"
	"strconv"
	"sync"
	"sync/atomic"
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

type PerformanceStats struct {
	TotalRequests int64
	TotalDuration int64
	SlowRequests  int64
}

var (
	totalRequests  atomic.Int64
	totalDuration  atomic.Int64
	slowRequests   atomic.Int64
	pathCounters   sync.Map
)

func PerformanceMonitoring() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.FullPath()

		c.Next()

		duration := time.Since(start)
		recordRequest(path, duration)

		c.Header("X-Response-Time", strconv.FormatInt(duration.Milliseconds(), 10)+"ms")
	}
}

func recordRequest(path string, duration time.Duration) {
	totalRequests.Add(1)
	totalDuration.Add(duration.Milliseconds())

	if path != "" {
		if counter, ok := pathCounters.LoadOrStore(path, &atomic.Int64{}); ok {
			counter.(*atomic.Int64).Add(1)
		}
	}

	if duration > 100*time.Millisecond {
		slowRequests.Add(1)
	}
}

func GetPerformanceStats() map[string]interface{} {
	reqCount := totalRequests.Load()
	durCount := totalDuration.Load()

	avgDuration := int64(0)
	if reqCount > 0 {
		avgDuration = durCount / reqCount
	}

	requestCount := make(map[string]int64)
	pathCounters.Range(func(key, value interface{}) bool {
		requestCount[key.(string)] = value.(*atomic.Int64).Load()
		return true
	})

	return map[string]interface{}{
		"total_requests":  reqCount,
		"avg_duration_ms": avgDuration,
		"slow_requests":   slowRequests.Load(),
		"request_count":   requestCount,
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
