package middleware

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"fmt"
	"io"
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type CacheOptions struct {
	Prefix       string
	TTL          time.Duration
	Methods      []string
	ExcludePaths []string
	KeyFunc      func(*gin.Context) string
}

type cacheResponse struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

var defaultCacheOptions = &CacheOptions{
	Prefix:  "http_cache:",
	TTL:     5 * time.Minute,
	Methods: []string{"GET", "POST"},
	ExcludePaths: []string{
		"/api/admin",
		"/api/auth/login",
		"/health",
	},
}

func CacheMiddleware(options *CacheOptions) gin.HandlerFunc {
	if options == nil {
		options = defaultCacheOptions
	}
	if options.Prefix == "" {
		options.Prefix = "http_cache:"
	}
	if options.TTL == 0 {
		options.TTL = 5 * time.Minute
	}
	if len(options.Methods) == 0 {
		options.Methods = []string{"GET", "POST"}
	}

	return func(c *gin.Context) {
		if !shouldCache(c, options) {
			c.Next()
			return
		}

		cacheKey := generateCacheKey(c, options)

		if redis.Client == nil {
			c.Next()
			return
		}

		ctx := c.Request.Context()

		cached, err := redis.Client.Get(ctx, cacheKey).Bytes()
		if err == nil && len(cached) > 0 {
			cr, err := deserializeCacheResponse(cached)
			if err == nil {
				for key, values := range cr.Header {
					for _, value := range values {
						c.Header(key, value)
					}
				}
				c.Header("X-Cache-Hit", "true")
				c.Data(cr.StatusCode, "application/json", cr.Body)
				c.Abort()
				return
			}
		}

		writer := &cacheResponseWriter{
			ResponseWriter: c.Writer,
			body:          &bytes.Buffer{},
		}
		c.Writer = writer

		c.Next()

		if writer.StatusCode() >= 200 && writer.StatusCode() < 300 {
			cr := &cacheResponse{
				StatusCode: writer.StatusCode(),
				Body:       writer.body.Bytes(),
				Header:     make(http.Header),
			}
			writer.Header().Clone()
			for key, values := range writer.Header() {
				cr.Header[key] = values
			}

			cachedData, err := serializeCacheResponse(cr)
			if err == nil {
				redis.Client.Set(ctx, cacheKey, cachedData, options.TTL)
			}
		}
	}
}

func shouldCache(c *gin.Context, options *CacheOptions) bool {
	methodAllowed := false
	for _, m := range options.Methods {
		if c.Request.Method == m {
			methodAllowed = true
			break
		}
	}
	if !methodAllowed {
		return false
	}

	path := c.Request.URL.Path
	for _, excluded := range options.ExcludePaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}

	return true
}

func generateCacheKey(c *gin.Context, options *CacheOptions) string {
	if options.KeyFunc != nil {
		return options.Prefix + options.KeyFunc(c)
	}

	url := c.Request.URL.String()
	hash := sha256.Sum256([]byte(url))
	urlHash := hex.EncodeToString(hash[:8])

	return fmt.Sprintf("%s%s:%s", options.Prefix, c.Request.Method, urlHash)
}

type cacheResponseWriter struct {
	gin.ResponseWriter
	body        *bytes.Buffer
	statusCode int
}

func (w *cacheResponseWriter) Write(data []byte) (int, error) {
	w.body.Write(data)
	return w.ResponseWriter.Write(data)
}

func (w *cacheResponseWriter) WriteHeader(statusCode int) {
	w.statusCode = statusCode
	w.ResponseWriter.WriteHeader(statusCode)
}

func (w *cacheResponseWriter) StatusCode() int {
	if w.statusCode == 0 {
		return http.StatusOK
	}
	return w.statusCode
}

func serializeCacheResponse(cr *cacheResponse) ([]byte, error) {
	var buf bytes.Buffer
	buf.WriteString(fmt.Sprintf("STATUS:%d\n", cr.StatusCode))

	for key, values := range cr.Header {
		for _, value := range values {
			buf.WriteString(fmt.Sprintf("HEADER:%s:%s\n", key, value))
		}
	}
	buf.WriteString("BODY:\n")
	buf.Write(cr.Body)

	return buf.Bytes(), nil
}

func deserializeCacheResponse(data []byte) (*cacheResponse, error) {
	cr := &cacheResponse{
		Header: make(http.Header),
	}

	lines := strings.Split(string(data), "\n")
	bodyStarted := false
	var bodyLines []string

	for _, line := range lines {
		if bodyStarted {
			bodyLines = append(bodyLines, line)
			continue
		}

		if line == "BODY:" {
			bodyStarted = true
			continue
		}

		if strings.HasPrefix(line, "STATUS:") {
			fmt.Sscanf(line[7:], "%d", &cr.StatusCode)
			continue
		}

		if strings.HasPrefix(line, "HEADER:") {
			headerParts := strings.SplitN(line[7:], ":", 2)
			if len(headerParts) == 2 {
				cr.Header.Add(headerParts[0], headerParts[1])
			}
			continue
		}
	}

	cr.Body = []byte(strings.Join(bodyLines, "\n"))
	return cr, nil
}

func InvalidateCachePrefix(prefix string) error {
	if redis.Client == nil {
		return nil
	}

	ctx := redis.GetContext()
	pattern := redis.Client.Publish(ctx, "cache_invalidate", prefix)
	return pattern.Err()
}

func InvalidateCacheKey(key string) error {
	if redis.Client == nil {
		return nil
	}

	ctx := redis.GetContext()
	return redis.Client.Del(ctx, key).Err()
}

func ClearCache() error {
	if redis.Client == nil {
		return nil
	}

	ctx := redis.GetContext()
	iter := redis.Client.Scan(ctx, 0, "http_cache:*", 100).Iterator()
	for iter.Next(ctx) {
		redis.Client.Del(ctx, iter.Val())
	}
	return iter.Err()
}

type CacheStats struct {
	Hits   int64
	Misses int64
	Keys   int64
}

func GetCacheStats() (*CacheStats, error) {
	if redis.Client == nil {
		return &CacheStats{}, nil
	}

	ctx := redis.GetContext()

	hits, err := redis.Client.Get(ctx, "cache:stats:hits").Int64()
	if err != nil {
		hits = 0
	}

	misses, err := redis.Client.Get(ctx, "cache:stats:misses").Int64()
	if err != nil {
		misses = 0
	}

	var keys int64
	iter := redis.Client.Scan(ctx, 0, "http_cache:*", 0).Iterator()
	for iter.Next(ctx) {
		keys++
	}
	if err := iter.Err(); err != nil {
		keys = 0
	}

	return &CacheStats{
		Hits:   hits,
		Misses: misses,
		Keys:   keys,
	}, nil
}

func (w *cacheResponseWriter) Status() int {
	return w.statusCode
}

func CopyRequestBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	body, err := io.ReadAll(c.Request.Body)
	if err != nil {
		return nil, err
	}
	c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
	return body, nil
}

type ResponseCacheEntry struct {
	Key       string
	TTL       time.Duration
	CreatedAt time.Time
}

func ListCachedResponses(prefix string, limit int64) ([]ResponseCacheEntry, error) {
	if redis.Client == nil {
		return nil, nil
	}

	ctx := redis.GetContext()
	pattern := fmt.Sprintf("%s*", prefix)

	var entries []ResponseCacheEntry
	iter := redis.Client.Scan(ctx, 0, pattern, limit).Iterator()
	for iter.Next(ctx) {
		ttl, err := redis.Client.TTL(ctx, iter.Val()).Result()
		if err != nil {
			continue
		}

		entries = append(entries, ResponseCacheEntry{
			Key: iter.Val(),
			TTL: ttl,
		})
	}

	return entries, iter.Err()
}
