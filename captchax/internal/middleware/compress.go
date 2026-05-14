package middleware

import (
	"bufio"
	"compress/gzip"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type GzipConfig struct {
	Level          int
	MinSize        int
	ExcludedPaths  []string
	IncludedTypes  []string
	BufferPoolSize int
}

type GzipMiddleware struct {
	config      *GzipConfig
	bufferPool  *sync.Pool
	compression int
}

func NewGzipMiddleware(cfg *GzipConfig) *GzipMiddleware {
	if cfg == nil {
		cfg = &GzipConfig{
			Level:          gzip.DefaultCompression,
			MinSize:        1024,
			BufferPoolSize: 100,
		}
	}

	pool := &sync.Pool{
		New: func() interface{} {
			return newGzipBuffer(cfg.Level)
		},
	}

	return &GzipMiddleware{
		config:      cfg,
		bufferPool:  pool,
		compression: cfg.Level,
	}
}

type gzipBuffer struct {
	w   *gzip.Writer
	buf *strings.Builder
}

func newGzipBuffer(level int) *gzipBuffer {
	var buf strings.Builder
	w := gzip.NewWriter(&buf)
	return &gzipBuffer{w: w, buf: &buf}
}

func (g *GzipMiddleware) shouldCompress(c *http.Request) bool {
	encoding := c.Header.Get("Accept-Encoding")
	if !strings.Contains(encoding, "gzip") {
		return false
	}

	for _, path := range g.config.ExcludedPaths {
		if strings.HasPrefix(c.URL.Path, path) {
			return false
		}
	}

	return true
}

func (g *GzipMiddleware) compressResponse(w http.ResponseWriter, c *http.Request) (http.ResponseWriter, bool) {
	if !g.shouldCompress(c) {
		return w, false
	}

	contentType := w.Header().Get("Content-Type")
	compressible := false
	for _, t := range g.config.IncludedTypes {
		if strings.Contains(contentType, t) {
			compressible = true
			break
		}
	}

	if !compressible && !strings.HasPrefix(contentType, "text/") && !strings.Contains(contentType, "json") && !strings.Contains(contentType, "xml") {
		return w, false
	}

	gz, err := gzip.NewWriterLevel(w, g.compression)
	if err != nil {
		return w, false
	}

	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")

	return &gzipResponseWriter{
		ResponseWriter: w,
		gz:             gz,
		minSize:       g.config.MinSize,
	}, true
}

type gzipResponseWriter struct {
	http.ResponseWriter
	gz           *gzip.Writer
	wroteHeader bool
	minSize     int
	compressed  bool
	mu          sync.Mutex
}

func (w *gzipResponseWriter) WriteHeader(statusCode int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *gzipResponseWriter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if !w.compressed && len(b) < w.minSize {
		w.gz.Close()
		return w.ResponseWriter.Write(b)
	}

	if !w.compressed {
		w.compressed = true
	}

	return w.gz.Write(b)
}

func (w *gzipResponseWriter) Flush() {
	if w.gz != nil {
		w.gz.Flush()
	}
	if flusher, ok := w.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

func (w *gzipResponseWriter) Close() error {
	if w.gz != nil {
		return w.gz.Close()
	}
	return nil
}

func Gzip(level int) func(http.Handler) http.Handler {
	middleware := NewGzipMiddleware(&GzipConfig{
		Level:         level,
		MinSize:       1024,
		IncludedTypes: []string{"text/", "application/json", "application/xml"},
	})

	return func(next http.Handler) http.Handler {
		return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			if !middleware.shouldCompress(r) {
				next.ServeHTTP(w, r)
				return
			}

			gz, err := gzip.NewWriterLevel(w, level)
			if err != nil {
				next.ServeHTTP(w, r)
				return
			}

			w.Header().Set("Content-Encoding", "gzip")
			w.Header().Set("Vary", "Accept-Encoding")

			defer gz.Close()

			gzWriter := &gzipFlusher{ResponseWriter: w, gz: gz}
			next.ServeHTTP(gzWriter, r)
		})
	}
}

type gzipFlusher struct {
	http.ResponseWriter
	gz *gzip.Writer
}

func (gf *gzipFlusher) Write(b []byte) (int, error) {
	return gf.gz.Write(b)
}

func (gf *gzipFlusher) Flush() {
	gf.gz.Flush()
	if flusher, ok := gf.ResponseWriter.(http.Flusher); ok {
		flusher.Flush()
	}
}

type FlushingResponseWriter interface {
	http.ResponseWriter
	http.Flusher
	io.Closer
}

type responseWriterWithFlush struct {
	http.ResponseWriter
	http.Flusher
}

func DecompressMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		if !strings.Contains(r.Header.Get("Content-Encoding"), "gzip") {
			next.ServeHTTP(w, r)
			return
		}

		reader, err := gzip.NewReader(r.Body)
		if err != nil {
			http.Error(w, "Failed to decompress body", http.StatusBadRequest)
			return
		}
		defer reader.Close()

		r.Body = reader
		r.Header.Set("Content-Length", "")
		r.Header.Del("Content-Encoding")

		next.ServeHTTP(w, r)
	})
}

func NewDecompressReader(r io.Reader) (*gzip.Reader, error) {
	return gzip.NewReader(r)
}

type StreamCompressor struct {
	w       io.Writer
	gz      *gzip.Writer
	mu      sync.Mutex
	enabled bool
}

func NewStreamCompressor(w io.Writer, level int) (*StreamCompressor, error) {
	gz, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil, err
	}
	return &StreamCompressor{
		w:       w,
		gz:      gz,
		enabled: true,
	}, nil
}

func (sc *StreamCompressor) Write(p []byte) (n int, err error) {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if !sc.enabled {
		return sc.w.Write(p)
	}
	return sc.gz.Write(p)
}

func (sc *StreamCompressor) Flush() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.gz != nil {
		return sc.gz.Flush()
	}
	return nil
}

func (sc *StreamCompressor) Close() error {
	sc.mu.Lock()
	defer sc.mu.Unlock()

	if sc.gz != nil {
		err := sc.gz.Close()
		sc.gz = nil
		return err
	}
	return nil
}

func (sc *StreamCompressor) Disable() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.enabled = false
}

func (sc *StreamCompressor) Enable() {
	sc.mu.Lock()
	defer sc.mu.Unlock()
	sc.enabled = true
}

type pooledGzipWriter struct {
	pool  *sync.Pool
	gz    *gzip.Writer
	w     io.Writer
	level int
}

func newPooledGzipWriter(w io.Writer, level int) *pooledGzipWriter {
	return &pooledGzipWriter{
		w:     w,
		level: level,
	}
}

var defaultGzipPool = &sync.Pool{
	New: func() interface{} {
		return &pooledGzipWriter{level: gzip.DefaultCompression}
	},
}

func GetGzipWriter(w io.Writer) *pooledGzipWriter {
	gzw := defaultGzipPool.Get().(*pooledGzipWriter)
	gzw.w = w
	gzw.gz = nil
	return gzw
}

func PutGzipWriter(gzw *pooledGzipWriter) {
	if gzw.gz != nil {
		gzw.gz.Close()
	}
	defaultGzipPool.Put(gzw)
}

func (gzw *pooledGzipWriter) Write(p []byte) (int, error) {
	if gzw.gz == nil {
		var err error
		gzw.gz, err = gzip.NewWriterLevel(gzw.w, gzw.level)
		if err != nil {
			return 0, err
		}
	}
	return gzw.gz.Write(p)
}

func (gzw *pooledGzipWriter) Close() error {
	if gzw.gz != nil {
		return gzw.gz.Close()
	}
	return nil
}

type BufferedFlusher struct {
	w        io.Writer
	buf      *bufio.Writer
	mu       sync.Mutex
	interval time.Duration
	stopCh   chan struct{}
}

func NewBufferedFlusher(w io.Writer, size int, interval time.Duration) *BufferedFlusher {
	if interval == 0 {
		interval = 100 * time.Millisecond
	}

	bf := &BufferedFlusher{
		w:        w,
		buf:      bufio.NewWriterSize(w, size),
		interval: interval,
		stopCh:   make(chan struct{}),
	}

	go bf.autoFlush()
	return bf
}

func (bf *BufferedFlusher) autoFlush() {
	ticker := time.NewTicker(bf.interval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			bf.mu.Lock()
			bf.buf.Flush()
			bf.mu.Unlock()
		case <-bf.stopCh:
			return
		}
	}
}

func (bf *BufferedFlusher) Write(p []byte) (int, error) {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.buf.Write(p)
}

func (bf *BufferedFlusher) Flush() error {
	bf.mu.Lock()
	defer bf.mu.Unlock()
	return bf.buf.Flush()
}

func (bf *BufferedFlusher) Stop() {
	close(bf.stopCh)
	bf.Flush()
}

func (gm *GzipMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !gm.shouldCompress(c.Request) {
			c.Next()
			return
		}

		gz, err := gzip.NewWriterLevel(c.Writer, gm.compression)
		if err != nil {
			c.Next()
			return
		}

		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		c.Writer = &gzipResponseWriterAdapter{
			ResponseWriter: c.Writer,
			gz:            gz,
			wroteHeader:   false,
			minSize:       gm.config.MinSize,
			compressed:    false,
		}

		c.Next()
		gz.Close()
	}
}

type gzipResponseWriterAdapter struct {
	gin.ResponseWriter
	gz           *gzip.Writer
	wroteHeader  bool
	minSize      int
	compressed   bool
	headerWrote  bool
}

func (w *gzipResponseWriterAdapter) WriteHeader(code int) {
	if w.wroteHeader {
		return
	}
	w.wroteHeader = true
	w.Header().Set("Content-Encoding", "gzip")
	w.Header().Set("Vary", "Accept-Encoding")
	w.ResponseWriter.WriteHeader(code)
}

func (w *gzipResponseWriterAdapter) Write(b []byte) (int, error) {
	if !w.wroteHeader {
		w.WriteHeader(http.StatusOK)
	}

	if !w.compressed && len(b) < w.minSize {
		w.gz.Close()
		return w.ResponseWriter.Write(b)
	}

	if !w.compressed {
		w.compressed = true
	}

	return w.gz.Write(b)
}

func (w *gzipResponseWriterAdapter) Close() error {
	return w.gz.Close()
}

func GzipHandler(cfg *GzipConfig) gin.HandlerFunc {
	mw := NewGzipMiddleware(cfg)
	return mw.Handler()
}
