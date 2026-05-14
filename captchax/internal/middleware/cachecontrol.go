package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CacheControlConfig struct {
	Enabled        bool
	ETagEnabled    bool
	MaxAge         time.Duration
	SharedMaxAge   time.Duration
	MustRevalidate bool
	Private        bool
	NoCache        bool
	NoStore        bool
	Preload        bool
	Immutable      bool
}

var defaultCacheControlConfig = &CacheControlConfig{
	Enabled:        true,
	ETagEnabled:    true,
	MaxAge:         5 * time.Minute,
	SharedMaxAge:   10 * time.Minute,
	MustRevalidate: true,
	Private:        false,
	NoCache:        false,
	NoStore:        false,
	Preload:        false,
	Immutable:      false,
}

func CacheControl() gin.HandlerFunc {
	return CacheControlWithConfig(defaultCacheControlConfig)
}

func CacheControlWithConfig(config *CacheControlConfig) gin.HandlerFunc {
	if config == nil {
		config = defaultCacheControlConfig
	}

	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		c.Set("cache_config", config)

		c.Next()

		applyCacheHeaders(c, config)
	}
}

func applyCacheHeaders(c *gin.Context, config *CacheControlConfig) {
	if c.Writer.Written() && config.ETagEnabled {
		etag := generateETag(c)
		c.Header("ETag", etag)
	}

	cacheValue := buildCacheControlHeader(config)
	if cacheValue != "" {
		c.Header("Cache-Control", cacheValue)
	}
}

func buildCacheControlHeader(config *CacheControlConfig) string {
	var directives []string

	if config.NoStore {
		directives = append(directives, "no-store")
		return strings.Join(directives, ", ")
	}

	if config.NoCache {
		directives = append(directives, "no-cache")
	}

	if config.Private {
		directives = append(directives, "private")
	} else if config.SharedMaxAge > 0 {
		directives = append(directives, fmt.Sprintf("s-maxage=%d", int(config.SharedMaxAge.Seconds())))
	}

	if config.MaxAge > 0 {
		directives = append(directives, fmt.Sprintf("max-age=%d", int(config.MaxAge.Seconds())))
	}

	if config.MustRevalidate {
		directives = append(directives, "must-revalidate")
	}

	if config.Immutable {
		directives = append(directives, "immutable")
	}

	if config.Preload {
		directives = append(directives, "preload")
	}

	return strings.Join(directives, ", ")
}

func generateETag(c *gin.Context) string {
	if c.Writer.Written() {
		timestamp := time.Now().Unix()
		content := fmt.Sprintf("%s-%d-%s", c.Request.URL.Path, timestamp, c.GetHeader("Accept-Encoding"))
		hash := sha256.Sum256([]byte(content))
		return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:8]))
	}

	timestamp := time.Now().Unix()
	content := fmt.Sprintf("%s-%d-%s", c.Request.URL.Path, timestamp, c.GetHeader("Accept-Encoding"))
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:8]))
}

func ETag() gin.HandlerFunc {
	return ETagWithConfig(defaultCacheControlConfig)
}

func ETagWithConfig(config *CacheControlConfig) gin.HandlerFunc {
	if config == nil {
		config = defaultCacheControlConfig
	}

	return func(c *gin.Context) {
		if !config.ETagEnabled {
			c.Next()
			return
		}

		c.Next()

		if !c.Writer.Written() {
			return
		}

		etag := generateETagFromResponse(c)
		c.Header("ETag", etag)

		ifNoneMatch := c.GetHeader("If-None-Match")
		if ifNoneMatch != "" {
			if matchETag(ifNoneMatch, etag) {
				c.AbortWithStatus(http.StatusNotModified)
				return
			}
		}

		ifModifiedSince := c.GetHeader("If-Modified-Since")
		if ifModifiedSince != "" {
			if isNotModifiedSince(ifModifiedSince) {
				c.AbortWithStatus(http.StatusNotModified)
				return
			}
		}
	}
}

func generateETagFromResponse(c *gin.Context) string {
	timestamp := time.Now().UnixNano()
	content := fmt.Sprintf("%s-%d-%s-%s", c.Request.URL.Path, timestamp, c.Request.Method, c.GetHeader("Accept-Encoding"))
	hash := sha256.Sum256([]byte(content))
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:8]))
}

func matchETag(ifNoneMatch, etag string) bool {
	etags := strings.Split(ifNoneMatch, ",")
	for _, e := range etags {
		e = strings.TrimSpace(e)
		if e == "*" || e == etag {
			return true
		}
	}
	return false
}

func isNotModifiedSince(ifModifiedSince string) bool {
	t, err := time.Parse(http.TimeFormat, ifModifiedSince)
	if err != nil {
		return false
	}
	return time.Since(t) < 0
}

type LastModifiedConfig struct {
	Enabled bool
}

var defaultLastModifiedConfig = &LastModifiedConfig{
	Enabled: true,
}

func LastModified() gin.HandlerFunc {
	return LastModifiedWithConfig(defaultLastModifiedConfig)
}

func LastModifiedWithConfig(config *LastModifiedConfig) gin.HandlerFunc {
	if config == nil {
		config = defaultLastModifiedConfig
	}

	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		c.Next()

		if c.Writer.Written() {
			c.Header("Last-Modified", time.Now().Format(http.TimeFormat))

			ifModifiedSince := c.GetHeader("If-Modified-Since")
			if ifModifiedSince != "" {
				if isNotModifiedSince(ifModifiedSince) {
					c.AbortWithStatus(http.StatusNotModified)
					return
				}
			}
		}
	}
}

type ConditionalRequestConfig struct {
	Enabled           bool
	UseETag           bool
	UseLastModified   bool
	WeakETagThreshold int
}

var defaultConditionalRequestConfig = &ConditionalRequestConfig{
	Enabled:           true,
	UseETag:           true,
	UseLastModified:   true,
	WeakETagThreshold: 0,
}

func ConditionalRequest() gin.HandlerFunc {
	return ConditionalRequestWithConfig(defaultConditionalRequestConfig)
}

func ConditionalRequestWithConfig(config *ConditionalRequestConfig) gin.HandlerFunc {
	if config == nil {
		config = defaultConditionalRequestConfig
	}

	return func(c *gin.Context) {
		if !config.Enabled {
			c.Next()
			return
		}

		c.Next()

		if !c.Writer.Written() {
			return
		}

		if config.UseETag {
			etag := generateETagFromResponse(c)
			c.Header("ETag", etag)

			ifNoneMatch := c.GetHeader("If-None-Match")
			if ifNoneMatch != "" {
				if matchETag(ifNoneMatch, etag) {
					c.AbortWithStatus(http.StatusNotModified)
					return
				}
			}
		}

		if config.UseLastModified {
			lastModified := time.Now().Format(http.TimeFormat)
			c.Header("Last-Modified", lastModified)

			ifModifiedSince := c.GetHeader("If-Modified-Since")
			if ifModifiedSince != "" {
				if isNotModifiedSince(ifModifiedSince) {
					c.AbortWithStatus(http.StatusNotModified)
					return
				}
			}
		}
	}
}

type CacheableResponse struct {
	Content      []byte
	ContentType  string
	ETag         string
	LastModified time.Time
	StatusCode   int
}

func GenerateStrongETag(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf(`"%s"`, hex.EncodeToString(hash[:]))
}

func GenerateWeakETag(content []byte) string {
	hash := sha256.Sum256(content)
	return fmt.Sprintf(`W/"%s"`, hex.EncodeToString(hash[:]))
}

func ParseETagHeader(header string) []string {
	if header == "" {
		return nil
	}
	return strings.Split(header, ",")
}

func CompareETags(e1, e2 string) int {
	if strings.HasPrefix(e1, "W/") && !strings.HasPrefix(e2, "W/") {
		return -1
	}
	if !strings.HasPrefix(e1, "W/") && strings.HasPrefix(e2, "W/") {
		return 1
	}

	e1Tag := strings.Trim(e1, `"W/"`)
	e2Tag := strings.Trim(e2, `"W/"`)
	if e1Tag < e2Tag {
		return -1
	}
	if e1Tag > e2Tag {
		return 1
	}
	return 0
}

type CacheStats struct {
	Hits          int64
	Misses        int64
	Invalidations int64
	CurrentSize   int64
	MaxSize       int64
	mu            sync.RWMutex
}

var cacheStats = &CacheStats{}

func GetCacheStats() *CacheStats {
	return cacheStats
}

func RecordCacheHit() {
	cacheStats.mu.Lock()
	defer cacheStats.mu.Unlock()
	cacheStats.Hits++
}

func RecordCacheMiss() {
	cacheStats.mu.Lock()
	defer cacheStats.mu.Unlock()
	cacheStats.Misses++
}

func RecordCacheInvalidation() {
	cacheStats.mu.Lock()
	defer cacheStats.mu.Unlock()
	cacheStats.Invalidations++
}

type CacheValidator interface {
	ValidateETag(etag string) bool
	ValidateLastModified(t time.Time) bool
}

type DefaultCacheValidator struct{}

func (v *DefaultCacheValidator) ValidateETag(etag string) bool {
	return etag != "" && len(etag) >= 2
}

func (v *DefaultCacheValidator) ValidateLastModified(t time.Time) bool {
	return !t.IsZero() && t.Before(time.Now().Add(time.Hour))
}

func GetCacheValidator() CacheValidator {
	return &DefaultCacheValidator{}
}

func SetSharedMaxAge(c *gin.Context, seconds int) {
	c.Header("Cache-Control", fmt.Sprintf("s-maxage=%d", seconds))
}

func SetMaxAge(c *gin.Context, seconds int) {
	c.Header("Cache-Control", fmt.Sprintf("max-age=%d", seconds))
}

func SetNoCache(c *gin.Context) {
	c.Header("Cache-Control", "no-cache")
}

func SetNoStore(c *gin.Context) {
	c.Header("Cache-Control", "no-store")
}

func SetPrivate(c *gin.Context) {
	c.Header("Cache-Control", "private")
}

func SetPublic(c *gin.Context) {
	c.Header("Cache-Control", "public")
}

func SetExpires(c *gin.Context, t time.Time) {
	c.Header("Expires", t.Format(http.TimeFormat))
}

func SetVary(c *gin.Context, fields ...string) {
	c.Header("Vary", strings.Join(fields, ", "))
}

func IsConditionalRequest(c *gin.Context) bool {
	return c.GetHeader("If-None-Match") != "" || c.GetHeader("If-Modified-Since") != ""
}

func GetRequestedETag(c *gin.Context) string {
	return c.GetHeader("If-None-Match")
}

func GetRequestedLastModified(c *gin.Context) string {
	return c.GetHeader("If-Modified-Since")
}

func ShouldRespondWith304(c *gin.Context, etag, lastModified string) bool {
	ifNoneMatch := c.GetHeader("If-None-Match")
	ifModifiedSince := c.GetHeader("If-Modified-Since")

	if ifNoneMatch != "" && matchETag(ifNoneMatch, etag) {
		return true
	}

	if ifModifiedSince != "" {
		if lastModified == "" {
			return false
		}
		lm, err := time.Parse(http.TimeFormat, lastModified)
		if err != nil {
			return false
		}
		if !lm.Before(parseHTTPTime(ifModifiedSince)) {
			return true
		}
	}

	return false
}

func parseHTTPTime(s string) time.Time {
	if t, err := time.Parse(http.TimeFormat, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.RFC850, s); err == nil {
		return t
	}
	if t, err := time.Parse(time.ANSIC, s); err == nil {
		return t
	}
	return time.Time{}
}

func RespondWith304(c *gin.Context) {
	c.Header("X-Cache-Status", "HIT")
	c.AbortWithStatus(http.StatusNotModified)
}

func RespondWith200(c *gin.Context) {
	c.Header("X-Cache-Status", "MISS")
}

func RespondWithETag(c *gin.Context, etag string) {
	c.Header("ETag", etag)
	if ShouldRespondWith304(c, etag, "") {
		RespondWith304(c)
	}
}

type CacheManager struct {
	config     *CacheControlConfig
	maxEntries int
	entries    map[string]*CacheEntry
	mu         sync.RWMutex
}

type CacheEntry struct {
	Key             string
	Value           interface{}
	ETag            string
	LastModified    time.Time
	ExpiresAt       time.Time
	AccessCount     int64
	LastAccessedAt  time.Time
	Headers         map[string]string
}

func NewCacheManager(maxEntries int) *CacheManager {
	return &CacheManager{
		config:     defaultCacheControlConfig,
		maxEntries: maxEntries,
		entries:    make(map[string]*CacheEntry),
	}
}

func (cm *CacheManager) Get(key string) (*CacheEntry, bool) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	entry, exists := cm.entries[key]
	if !exists {
		RecordCacheMiss()
		return nil, false
	}

	if time.Now().After(entry.ExpiresAt) {
		delete(cm.entries, key)
		RecordCacheMiss()
		return nil, false
	}

	entry.AccessCount++
	entry.LastAccessedAt = time.Now()
	RecordCacheHit()

	return entry, true
}

func (cm *CacheManager) Set(key string, entry *CacheEntry) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if len(cm.entries) >= cm.maxEntries {
		cm.evictOldest()
	}

	entry.LastAccessedAt = time.Now()
	cm.entries[key] = entry
}

func (cm *CacheManager) Delete(key string) {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	if _, exists := cm.entries[key]; exists {
		delete(cm.entries, key)
		RecordCacheInvalidation()
	}
}

func (cm *CacheManager) Clear() {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	cm.entries = make(map[string]*CacheEntry)
	RecordCacheInvalidation()
}

func (cm *CacheManager) evictOldest() {
	var oldest *CacheEntry
	var oldestKey string

	for key, entry := range cm.entries {
		if oldest == nil || entry.LastAccessedAt.Before(oldest.LastAccessedAt) {
			oldest = entry
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(cm.entries, oldestKey)
	}
}

type compressionStats struct {
	RequestsAccepted  int64
	RequestsRejected  int64
	BytesIn           int64
	BytesOut          int64
	CompressionRatio  float64
	mu                sync.RWMutex
}

var brotliStats = &compressionStats{}

func GetBrotliStats() *compressionStats {
	return brotliStats
}

func recordBrotliRequest(accepted bool, bytesIn, bytesOut int64) {
	brotliStats.mu.Lock()
	defer brotliStats.mu.Unlock()

	if accepted {
		brotliStats.RequestsAccepted++
		brotliStats.BytesIn += bytesIn
		brotliStats.BytesOut += bytesOut
		if bytesIn > 0 {
			brotliStats.CompressionRatio = float64(bytesOut) / float64(bytesIn)
		}
	} else {
		brotliStats.RequestsRejected++
	}
}

type BrotliConfig struct {
	Level          int
	MinSize        int
	ExcludedPaths  []string
	IncludedTypes  []string
	Quality        int
}

var defaultBrotliConfig = &BrotliConfig{
	Level:         6,
	MinSize:       1024,
	IncludedTypes: []string{"text/", "application/json", "application/xml"},
	Quality:       6,
}

func Brotli(level int) gin.HandlerFunc {
	return BrotliWithConfig(&BrotliConfig{
		Level:         level,
		MinSize:       1024,
		IncludedTypes: []string{"text/", "application/json", "application/xml"},
		Quality:       level,
	})
}

func BrotliWithConfig(config *BrotliConfig) gin.HandlerFunc {
	if config == nil {
		config = defaultBrotliConfig
	}

	level := config.Level
	if level <= 0 {
		level = 6
	}
	quality := config.Quality
	if quality <= 0 {
		quality = 6
	}

	return func(c *gin.Context) {
		encoding := c.GetHeader("Accept-Encoding")
		if !strings.Contains(strings.ToLower(encoding), "br") {
			c.Next()
			return
		}

		for _, path := range config.ExcludedPaths {
			if strings.HasPrefix(c.Request.URL.Path, path) {
				c.Next()
				return
			}
		}

		c.Next()

		if !c.Writer.Written() {
			return
		}

		contentType := c.Writer.Header().Get("Content-Type")
		shouldCompress := false

		for _, t := range config.IncludedTypes {
			if strings.Contains(contentType, t) {
				shouldCompress = true
				break
			}
		}

		if !shouldCompress && !strings.HasPrefix(contentType, "text/") && !strings.Contains(contentType, "json") && !strings.Contains(contentType, "xml") {
			return
		}

		if c.Writer.Size() < config.MinSize {
			return
		}

		content := c.Writer.Header().Get("Content-Length")
		if content == "" {
			return
		}

		compressedData := []byte(content)
		compressed := compressWithBrotli(compressedData, level, quality)
		if compressed == nil {
			return
		}

		recordBrotliRequest(true, int64(len(compressedData)), int64(len(compressed)))

		c.Header("Content-Encoding", "br")
		c.Header("Content-Length", strconv.Itoa(len(compressed)))
		c.Header("Vary", "Accept-Encoding")
		c.Header("X-Content-Type-Options", "nosniff")

		c.Data(http.StatusOK, contentType, compressed)
	}
}

func compressWithBrotli(data []byte, level, quality int) []byte {
	var buf bytes.Buffer
	br := getBrotliWriter(&buf, level, quality)
	if br == nil {
		return nil
	}

	_, err := br.Write(data)
	if err != nil {
		return nil
	}

	err = br.Close()
	if err != nil {
		return nil
	}

	putBrotliWriter(br)
	return buf.Bytes()
}

type brotliWriter struct {
	w   io.Writer
	buf *bytes.Buffer
}

func getBrotliWriter(w io.Writer, level, quality int) io.WriteCloser {
	return &simpleBrotliWriter{w: w}
}

func putBrotliWriter(bw io.WriteCloser) {}

type simpleBrotliWriter struct {
	w io.Writer
}

func (bw *simpleBrotliWriter) Write(p []byte) (int, error) {
	return bw.w.Write(p)
}

func (bw *simpleBrotliWriter) Close() error {
	return nil
}

func DecompressBrotliBody(next gin.HandlerFunc) gin.HandlerFunc {
	return func(c *gin.Context) {
		if !strings.Contains(c.GetHeader("Content-Encoding"), "br") {
			next(c)
			return
		}

		bodyBytes, err := io.ReadAll(c.Request.Body)
		if err != nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to read body"})
			return
		}

		decompressed := decompressBrotli(bodyBytes)
		if decompressed == nil {
			c.JSON(http.StatusBadRequest, gin.H{"error": "failed to decompress body"})
			return
		}

		c.Request.Body = io.NopCloser(bytes.NewReader(decompressed))
		c.Request.ContentLength = int64(len(decompressed))
		c.Request.Header.Set("Content-Length", strconv.Itoa(len(decompressed)))
		c.Request.Header.Del("Content-Encoding")

		next(c)
	}
}

func decompressBrotli(data []byte) []byte {
	return data
}

type BrotliResponseWriter struct {
	gin.ResponseWriter
	buf          *bytes.Buffer
	level        int
	headerSent   bool
	written      bool
}

func NewBrotliResponseWriter(w gin.ResponseWriter, level int) *BrotliResponseWriter {
	return &BrotliResponseWriter{
		ResponseWriter: w,
		buf:           new(bytes.Buffer),
		level:         level,
	}
}

func (bw *BrotliResponseWriter) Write(data []byte) (int, error) {
	if !bw.headerSent {
		bw.headerSent = true
	}

	bw.written = true
	return bw.buf.Write(data)
}

func (bw *BrotliResponseWriter) WriteHeader(statusCode int) {
	if !bw.headerSent {
		bw.headerSent = true
		bw.ResponseWriter.WriteHeader(statusCode)
	}
}

func (bw *BrotliResponseWriter) GetCompressedBody() []byte {
	if !bw.written {
		return nil
	}
	return bw.buf.Bytes()
}

func (bw *BrotliResponseWriter) Flush() {
	if f, ok := bw.ResponseWriter.(http.Flusher); ok {
		f.Flush()
	}
}

func (bw *BrotliResponseWriter) Close() error {
	return nil
}
