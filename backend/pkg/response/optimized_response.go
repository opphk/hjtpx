package response

import (
	"bytes"
	"compress/gzip"
	"compress/zlib"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

var (
	jsonMarshal   = json.Marshal
	jsonMarshalIndent = json.MarshalIndent
	jsonUnmarshal   = json.Unmarshal
)

type OptimizedResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    *ResponseMeta `json:"meta,omitempty"`
}

type ResponseMeta struct {
	Timestamp   int64  `json:"timestamp"`
	RequestID   string `json:"request_id,omitempty"`
	CacheHit    bool   `json:"cache_hit,omitempty"`
	ProcessTime string `json:"process_time,omitempty"`
}

type OptimizedJSONSerializer struct {
	enablePool     bool
	encodePool     sync.Pool
	compressLevel  int
	useFastJSON    bool
}

func NewOptimizedJSONSerializer() *OptimizedJSONSerializer {
	return &OptimizedJSONSerializer{
		enablePool:    true,
		compressLevel: gzip.BestSpeed,
		encodePool: sync.Pool{
			New: func() interface{} {
				return &bytes.Buffer{}
			},
		},
	}
}

func (s *OptimizedJSONSerializer) Marshal(v interface{}) ([]byte, error) {
	if s.enablePool {
		buf := s.encodePool.Get().(*bytes.Buffer)
		defer func() {
			buf.Reset()
			s.encodePool.Put(buf)
		}()
		defer s.encodePool.Put(buf)

		enc := json.NewEncoder(buf)
		enc.SetEscapeHTML(false)

		if err := enc.Encode(v); err != nil {
			return nil, err
		}

		result := make([]byte, buf.Len())
		copy(result, buf.Bytes())
		return result, nil
	}

	return json.Marshal(v)
}

func (s *OptimizedJSONSerializer) MarshalToString(v interface{}) (string, error) {
	data, err := s.Marshal(v)
	if err != nil {
		return "", err
	}
	return string(data), nil
}

func (s *OptimizedJSONSerializer) MarshalIndent(v interface{}, prefix, indent string) ([]byte, error) {
	return json.MarshalIndent(v, prefix, indent)
}

func (s *OptimizedJSONSerializer) Unmarshal(data []byte, v interface{}) error {
	return json.Unmarshal(data, v)
}

var defaultSerializer = NewOptimizedJSONSerializer()

func FastJSONMarshal(v interface{}) ([]byte, error) {
	return defaultSerializer.Marshal(v)
}

func FastJSONMarshalString(v interface{}) (string, error) {
	return defaultSerializer.MarshalToString(v)
}

func SuccessWithMeta(c *gin.Context, data interface{}) {
	requestID := c.GetString("RequestID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	processTime := c.GetString("ProcessTime")

	meta := &ResponseMeta{
		Timestamp:   time.Now().Unix(),
		RequestID:   requestID,
		ProcessTime: processTime,
	}

	resp := OptimizedResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Meta:    meta,
	}

	c.JSON(http.StatusOK, resp)
}

func SuccessWithCompression(c *gin.Context, data interface{}) {
	requestID := c.GetString("RequestID")
	if requestID == "" {
		requestID = generateRequestID()
	}

	processTime := c.GetString("ProcessTime")

	meta := &ResponseMeta{
		Timestamp:   time.Now().Unix(),
		RequestID:   requestID,
		ProcessTime: processTime,
	}

	resp := OptimizedResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Meta:    meta,
	}

	body, err := json.Marshal(resp)
	if err != nil {
		c.JSON(http.StatusOK, resp)
		return
	}

	acceptEncoding := c.GetHeader("Accept-Encoding")
	supportsGzip := contains(acceptEncoding, "gzip")
	supportsDeflate := contains(acceptEncoding, "deflate")

	if supportsGzip {
		c.Header("Content-Encoding", "gzip")
		c.Header("Vary", "Accept-Encoding")

		gz := gzipPool.Get().(*gzip.Writer)
		defer gzipPool.Put(gz)

		gz.Reset(c.Writer)
		defer gz.Close()

		gz.Write(body)
		c.Header("X-Content-Type-Options", "nosniff")
		return
	}

	if supportsDeflate {
		c.Header("Content-Encoding", "deflate")
		c.Header("Vary", "Accept-Encoding")

		zlibWriter := zlibPool.Get().(*zlib.Writer)
		defer zlibPool.Put(zlibWriter)

		zlibWriter.Reset(c.Writer)
		defer zlibWriter.Close()

		zlibWriter.Write(body)
		return
	}

	c.JSON(http.StatusOK, resp)
}

func contains(s, substr string) bool {
	if s == "" || substr == "" {
		return false
	}
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}

var gzipPool = sync.Pool{
	New: func() interface{} {
		w, _ := gzip.NewWriterLevel(nil, gzip.BestSpeed)
		return w
	},
}

var zlibPool = sync.Pool{
	New: func() interface{} {
		w, _ := zlib.NewWriterLevel(nil, zlib.BestSpeed)
		return w
	},
}

type CompressionWriter struct {
	Writer  io.Writer
	gz     *gzip.Writer
	buf    *bytes.Buffer
}

func NewCompressionWriter(w io.Writer, level int) (*CompressionWriter, error) {
	if level == 0 {
		level = gzip.BestSpeed
	}

	gz, err := gzip.NewWriterLevel(w, level)
	if err != nil {
		return nil, err
	}

	return &CompressionWriter{
		Writer: w,
		gz:    gz,
		buf:   bytes.NewBuffer(nil),
	}, nil
}

func (cw *CompressionWriter) Write(p []byte) (n int, err error) {
	return cw.gz.Write(p)
}

func (cw *CompressionWriter) Close() error {
	return cw.gz.Close()
}

type CachedResponse struct {
	StatusCode int
	Headers    http.Header
	Body       []byte
	Compressed bool
	CompressEncoding string
	CreatedAt  time.Time
	TTL        time.Duration
}

func NewCachedResponse(c *gin.Context, body []byte) *CachedResponse {
	return &CachedResponse{
		StatusCode: c.Writer.Status(),
		Headers:    c.Writer.Header().Clone(),
		Body:       body,
		CreatedAt:  time.Now(),
		TTL:        5 * time.Minute,
	}
}

func (cr *CachedResponse) IsExpired() bool {
	return time.Since(cr.CreatedAt) > cr.TTL
}

func (cr *CachedResponse) SetTTL(ttl time.Duration) {
	cr.TTL = ttl
}

func (cr *CachedResponse) Compress(level int) error {
	if level == 0 {
		level = gzip.BestSpeed
	}

	var buf bytes.Buffer
	gz, err := gzip.NewWriterLevel(&buf, level)
	if err != nil {
		return err
	}

	if _, err := gz.Write(cr.Body); err != nil {
		return err
	}

	if err := gz.Close(); err != nil {
		return err
	}

	cr.Body = buf.Bytes()
	cr.Compressed = true
	cr.CompressEncoding = "gzip"
	cr.Headers.Set("Content-Encoding", "gzip")

	return nil
}

func (cr *CachedResponse) WriteTo(w http.ResponseWriter) {
	for key, values := range cr.Headers {
		for _, value := range values {
			w.Header().Add(key, value)
		}
	}
	w.WriteHeader(cr.StatusCode)
	w.Write(cr.Body)
}

type ResponseCache struct {
	items     map[string]*CachedResponse
	mu        sync.RWMutex
	maxSize   int
	evictions int64
}

func NewResponseCache(maxSize int) *ResponseCache {
	if maxSize <= 0 {
		maxSize = 10000
	}

	cache := &ResponseCache{
		items:   make(map[string]*CachedResponse),
		maxSize: maxSize,
	}

	go cache.cleanup()

	return cache
}

func (rc *ResponseCache) Get(key string) (*CachedResponse, bool) {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	resp, exists := rc.items[key]
	if exists && !resp.IsExpired() {
		return resp, true
	}

	return nil, false
}

func (rc *ResponseCache) Set(key string, resp *CachedResponse) {
	rc.mu.Lock()
	defer rc.mu.Unlock()

	if len(rc.items) >= rc.maxSize {
		rc.evictOldest()
	}

	rc.items[key] = resp
}

func (rc *ResponseCache) Delete(key string) {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	delete(rc.items, key)
}

func (rc *ResponseCache) Clear() {
	rc.mu.Lock()
	defer rc.mu.Unlock()
	rc.items = make(map[string]*CachedResponse)
}

func (rc *ResponseCache) evictOldest() {
	if len(rc.items) == 0 {
		return
	}

	var oldestKey string
	var oldestTime time.Time

	for key, resp := range rc.items {
		if oldestTime.IsZero() || resp.CreatedAt.Before(oldestTime) {
			oldestKey = key
			oldestTime = resp.CreatedAt
		}
	}

	if oldestKey != "" {
		delete(rc.items, oldestKey)
		rc.evictions++
	}
}

func (rc *ResponseCache) cleanup() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		rc.mu.Lock()
		now := time.Now()
		keysToDelete := make([]string, 0)

		for key, resp := range rc.items {
			if now.Sub(resp.CreatedAt) > resp.TTL {
				keysToDelete = append(keysToDelete, key)
			}
		}

		for _, key := range keysToDelete {
			delete(rc.items, key)
			rc.evictions++
		}
		rc.mu.Unlock()
	}
}

func (rc *ResponseCache) GetStats() map[string]interface{} {
	rc.mu.RLock()
	defer rc.mu.RUnlock()

	stats := map[string]interface{}{
		"size":      len(rc.items),
		"max_size":  rc.maxSize,
		"evictions": rc.evictions,
	}

	if len(rc.items) > 0 {
		var totalSize int
		for _, resp := range rc.items {
			totalSize += len(resp.Body)
		}
		stats["avg_size"] = totalSize / len(rc.items)
		stats["total_size"] = totalSize
	}

	return stats
}

func generateRequestID() string {
	return fmt.Sprintf("%d-%d", time.Now().UnixNano(), time.Now().UnixNano()%10000)
}

type ResponseTimeRecorder struct {
	mu         sync.RWMutex
	requests   []time.Duration
	maxRecords int
}

func NewResponseTimeRecorder(maxRecords int) *ResponseTimeRecorder {
	if maxRecords <= 0 {
		maxRecords = 1000
	}

	return &ResponseTimeRecorder{
		requests:   make([]time.Duration, 0, maxRecords),
		maxRecords: maxRecords,
	}
}

func (r *ResponseTimeRecorder) Record(d time.Duration) {
	r.mu.Lock()
	defer r.mu.Unlock()

	r.requests = append(r.requests, d)
	if len(r.requests) > r.maxRecords {
		r.requests = r.requests[1:]
	}
}

func (r *ResponseTimeRecorder) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.requests) == 0 {
		return map[string]interface{}{
			"count": 0,
		}
	}

	var total time.Duration
	min := r.requests[0]
	max := r.requests[0]

	for _, d := range r.requests {
		total += d
		if d < min {
			min = d
		}
		if d > max {
			max = d
		}
	}

	avg := total / time.Duration(len(r.requests))

	return map[string]interface{}{
		"count":    len(r.requests),
		"avg_ms":   avg.Milliseconds(),
		"min_ms":   min.Milliseconds(),
		"max_ms":   max.Milliseconds(),
		"p50_ms":   r.percentile(50),
		"p95_ms":   r.percentile(95),
		"p99_ms":   r.percentile(99),
	}
}

func (r *ResponseTimeRecorder) percentile(p int) int64 {
	r.mu.RLock()
	defer r.mu.RUnlock()

	if len(r.requests) == 0 {
		return 0
	}

	sorted := make([]time.Duration, len(r.requests))
	copy(sorted, r.requests)

	i := len(sorted) * p / 100
	if i >= len(sorted) {
		i = len(sorted) - 1
	}

	return sorted[i].Milliseconds()
}
