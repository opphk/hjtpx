package middleware

import (
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

type CDNConfig struct {
	Enabled          bool
	BaseURL          string
	Version          string
	EnableMinify     bool
	EnableBrotli     bool
	CacheMaxAge      time.Duration
	IncludeQuery     bool
	ExcludePaths     []string
	Headers          map[string]string
}

var defaultCDNConfig = &CDNConfig{
	Enabled:      false,
	Version:      "v1",
	EnableMinify: true,
	EnableBrotli: true,
	CacheMaxAge:  365 * 24 * time.Hour,
	IncludeQuery: false,
	ExcludePaths: []string{
		"/api/",
		"/admin/",
		"/health",
		"/metrics",
	},
	Headers: map[string]string{
		"X-CDN-Enabled":        "true",
		"X-Content-Type-Options": "nosniff",
		"X-XSS-Protection":     "1; mode=block",
	},
}

var globalCDNConfig = defaultCDNConfig

func SetCDNConfig(config *CDNConfig) {
	if config == nil {
		return
	}
	globalCDNConfig = config
}

func GetCDNConfig() *CDNConfig {
	return globalCDNConfig
}

func CDNHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !globalCDNConfig.Enabled {
			c.Next()
			return
		}

		for key, value := range globalCDNConfig.Headers {
			c.Header(key, value)
		}

		if isCDNEnabled(c) {
			c.Header("X-Served-From", "CDN")
			if globalCDNConfig.CacheMaxAge > 0 {
				c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(globalCDNConfig.CacheMaxAge.Seconds())))
			}
		}

		c.Next()
	}
}

func isCDNEnabled(c *gin.Context) bool {
	path := c.Request.URL.Path

	for _, exclude := range globalCDNConfig.ExcludePaths {
		if strings.HasPrefix(path, exclude) {
			return false
		}
	}

	if c.Request.URL.RawQuery != "" && !globalCDNConfig.IncludeQuery {
		return false
	}

	return true
}

func RewriteStaticURL(cdnURL, version, originalPath string) string {
	if !globalCDNConfig.Enabled || cdnURL == "" {
		return originalPath
	}

	parts := strings.Split(originalPath, "?")
	path := parts[0]

	if !strings.HasPrefix(path, "/static/") && !strings.HasPrefix(path, "/assets/") {
		return originalPath
	}

	url := cdnURL
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	url += version
	if !strings.HasSuffix(url, "/") {
		url += "/"
	}

	url += strings.TrimPrefix(path, "/")

	if len(parts) > 1 && globalCDNConfig.IncludeQuery {
		url += "?" + parts[1]
	}

	return url
}

func CDNAssetMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		if !globalCDNConfig.Enabled {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		if !strings.HasPrefix(path, "/static/") && !strings.HasPrefix(path, "/assets/") {
			c.Next()
			return
		}

		for _, exclude := range globalCDNConfig.ExcludePaths {
			if strings.HasPrefix(path, exclude) {
				c.Next()
				return
			}
		}

		if globalCDNConfig.BaseURL != "" {
			newPath := RewriteStaticURL(globalCDNConfig.BaseURL, globalCDNConfig.Version, path)
			c.Request.URL.Path = newPath
		}

		if globalCDNConfig.EnableBrotli {
			acceptEncoding := c.GetHeader("Accept-Encoding")
			if strings.Contains(acceptEncoding, "br") {
				c.Header("Content-Encoding", "br")
			}
		}

		c.Next()
	}
}

type CacheControlConfig struct {
	IsPublic        bool
	MaxAge          time.Duration
	IsPrivate       bool
	NoStore         bool
	NoCache         bool
	MustRevalidate  bool
}

func NewCacheControlConfig() *CacheControlConfig {
	return &CacheControlConfig{
		IsPublic:  true,
		MaxAge:   1 * time.Hour,
	}
}

func (cc *CacheControlConfig) EnablePublic(maxAge time.Duration) *CacheControlConfig {
	cc.IsPublic = true
	cc.MaxAge = maxAge
	cc.IsPrivate = false
	return cc
}

func (cc *CacheControlConfig) EnablePrivate() *CacheControlConfig {
	cc.IsPrivate = true
	cc.IsPublic = false
	return cc
}

func (cc *CacheControlConfig) SetNoCache() *CacheControlConfig {
	cc.NoCache = true
	return cc
}

func (cc *CacheControlConfig) SetNoStore() *CacheControlConfig {
	cc.NoStore = true
	return cc
}

func (cc *CacheControlConfig) SetMustRevalidate() *CacheControlConfig {
	cc.MustRevalidate = true
	return cc
}

func (cc *CacheControlConfig) String() string {
	var parts []string

	if cc.NoStore {
		return "no-store, no-cache, must-revalidate"
	}

	if cc.NoCache {
		parts = append(parts, "no-cache")
	}

	if cc.IsPrivate {
		parts = append(parts, "private")
	} else if cc.IsPublic {
		parts = append(parts, "public")
	}

	if cc.MaxAge > 0 {
		parts = append(parts, "max-age="+strconv.Itoa(int(cc.MaxAge.Seconds())))
	}

	if cc.MustRevalidate {
		parts = append(parts, "must-revalidate")
	}

	return strings.Join(parts, ", ")
}

func (cc *CacheControlConfig) Apply(c *gin.Context) {
	c.Header("Cache-Control", cc.String())
}

func CacheControlMiddleware(cc *CacheControlConfig) gin.HandlerFunc {
	return func(c *gin.Context) {
		cc.Apply(c)
		c.Next()
	}
}

type StaticAssetCache struct {
	MaxAge time.Duration
}

func NewStaticAssetCache() *StaticAssetCache {
	return &StaticAssetCache{
		MaxAge: 7 * 24 * time.Hour,
	}
}

func (s *StaticAssetCache) Middleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if strings.HasSuffix(path, ".html") {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
			c.Header("Pragma", "no-cache")
			c.Header("Expires", "0")
			c.Next()
			return
		}

		if strings.HasSuffix(path, ".css") || strings.HasSuffix(path, ".js") {
			c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(s.MaxAge.Seconds()))+", immutable")
			c.Header("Vary", "Accept-Encoding")
			c.Next()
			return
		}

		if strings.HasSuffix(path, ".png") || strings.HasSuffix(path, ".jpg") ||
			strings.HasSuffix(path, ".jpeg") || strings.HasSuffix(path, ".gif") ||
			strings.HasSuffix(path, ".svg") || strings.HasSuffix(path, ".ico") ||
			strings.HasSuffix(path, ".webp") || strings.HasSuffix(path, ".avif") {
			c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(s.MaxAge.Seconds())))
			c.Header("Vary", "Accept-Encoding")
			c.Next()
			return
		}

		if strings.HasSuffix(path, ".woff2") || strings.HasSuffix(path, ".woff") ||
			strings.HasSuffix(path, ".ttf") || strings.HasSuffix(path, ".eot") {
			c.Header("Cache-Control", "public, max-age="+strconv.Itoa(int(365*24*time.Hour.Seconds()))+", immutable")
			c.Header("Vary", "Accept-Encoding")
			c.Next()
			return
		}

		c.Next()
	}
}

func ServeStaticAssetsWithCache() gin.HandlerFunc {
	cache := NewStaticAssetCache()
	return cache.Middleware()
}
