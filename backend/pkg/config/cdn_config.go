package config

import (
	"fmt"
	"strings"
	"time"
)

type CDNConfig struct {
	Enabled           bool     `yaml:"enabled"`
	Provider          string   `yaml:"provider"`
	BaseURL          string   `yaml:"base_url"`
	OriginURL        string   `yaml:"origin_url"`
	ApiVersion       string   `yaml:"api_version"`
	Timeout          int      `yaml:"timeout"`
	RetryAttempts    int      `yaml:"retry_attempts"`
	RetryDelay       int      `yaml:"retry_delay"`
	CacheMaxAge      int      `yaml:"cache_max_age"`
	EnableCompression bool     `yaml:"enable_compression"`
	CompressionLevel int      `yaml:"compression_level"`
	Domains          []string `yaml:"domains"`
	PathPatterns     []string `yaml:"path_patterns"`
	QueryStringCache bool     `yaml:"query_string_cache"`
	Headers          map[string]string `yaml:"headers"`
	EnableHTTPS      bool     `yaml:"enable_https"`
	HTTP2Enabled     bool     `yaml:"http2_enabled"`
}

type CDNProviderConfig struct {
	Name     string
	BaseURL  string
	Features CDNProviderFeatures
}

type CDNProviderFeatures struct {
	HTTP2         bool
	WebP          bool
	HTTPCompression bool
	EdgeLocations int
	APIVersion    string
}

var CDNProviders = map[string]CDNProviderConfig{
	"cloudflare": {
		Name:    "Cloudflare",
		BaseURL: "https://api.cloudflare.com/client/v4",
		Features: CDNProviderFeatures{
			HTTP2:          true,
			WebP:           true,
			HTTPCompression: true,
			EdgeLocations:  200,
			APIVersion:     "v4",
		},
	},
	"akamai": {
		Name:    "Akamai",
		BaseURL: "https://api.akamai.com",
		Features: CDNProviderFeatures{
			HTTP2:          true,
			WebP:           true,
			HTTPCompression: true,
			EdgeLocations:  300,
			APIVersion:     "v3",
		},
	},
	"aliyun": {
		Name:    "Aliyun CDN",
		BaseURL: "https://cdn.aliyuncs.com",
		Features: CDNProviderFeatures{
			HTTP2:          true,
			WebP:           true,
			HTTPCompression: true,
			EdgeLocations:  280,
			APIVersion:     "v4",
		},
	},
	"qcloud": {
		Name:    "Tencent Cloud CDN",
		BaseURL: "https://cdn.api.qcloud.com/v2",
		Features: CDNProviderFeatures{
			HTTP2:          true,
			WebP:           true,
			HTTPCompression: true,
			EdgeLocations:  250,
			APIVersion:     "v2",
		},
	},
	"custom": {
		Name:    "Custom CDN",
		BaseURL: "",
		Features: CDNProviderFeatures{
			HTTP2:          true,
			WebP:           true,
			HTTPCompression: true,
			EdgeLocations:  0,
			APIVersion:     "v1",
		},
	},
}

func NewDefaultCDNConfig() *CDNConfig {
	return &CDNConfig{
		Enabled:           false,
		Provider:          "custom",
		BaseURL:          "",
		OriginURL:        "",
		ApiVersion:       "v1",
		Timeout:          30,
		RetryAttempts:    3,
		RetryDelay:       1000,
		CacheMaxAge:      31536000,
		EnableCompression: true,
		CompressionLevel:  6,
		Domains:          []string{},
		PathPatterns: []string{
			"/static/*",
			"/dist/*",
			"/assets/*",
			"*.css",
			"*.js",
			"*.png",
			"*.jpg",
			"*.jpeg",
			"*.gif",
			"*.svg",
			"*.ico",
			"*.woff",
			"*.woff2",
			"*.ttf",
			"*.eot",
		},
		QueryStringCache: false,
		Headers: map[string]string{
			"X-CDN-Provider": "hjtpx",
			"X-CDN-Version": "15.0",
		},
		EnableHTTPS: true,
		HTTP2Enabled: true,
	}
}

type CDNResource struct {
	OriginalPath  string `json:"original_path"`
	CDNURL        string `json:"cdn_url"`
	ResourceType  string `json:"resource_type"`
	ContentType   string `json:"content_type"`
	FileSize      int64  `json:"file_size"`
	CacheControl string `json:"cache_control"`
	LastModified  string `json:"last_modified"`
	ETag          string `json:"etag"`
}

type CDNCacheInvalidationRequest struct {
	Paths   []string `json:"paths"`
	Target  string   `json:"target"`
}

type CDNStats struct {
	TotalRequests    int64     `json:"total_requests"`
	CacheHits       int64     `json:"cache_hits"`
	CacheMisses     int64     `json:"cache_misses"`
	Bandwidth       int64     `json:"bandwidth_bytes"`
	HitRate         float64   `json:"hit_rate"`
	LastUpdated     time.Time `json:"last_updated"`
}

func (c *CDNConfig) Validate() error {
	if c.Enabled {
		if c.BaseURL == "" && c.Provider != "custom" {
			if provider, ok := CDNProviders[c.Provider]; ok {
				c.BaseURL = provider.BaseURL
			}
		}

		if c.OriginURL == "" {
			return fmt.Errorf("origin_url is required when CDN is enabled")
		}

		if c.Timeout <= 0 {
			c.Timeout = 30
		}

		if c.RetryAttempts <= 0 {
			c.RetryAttempts = 3
		}

		if c.RetryDelay <= 0 {
			c.RetryDelay = 1000
		}

		if c.CacheMaxAge <= 0 {
			c.CacheMaxAge = 31536000
		}
	}

	return nil
}

func (c *CDNConfig) GetCacheMaxAgeDuration() time.Duration {
	return time.Duration(c.CacheMaxAge) * time.Second
}

func (c *CDNConfig) ShouldCachePath(path string) bool {
	for _, pattern := range c.PathPatterns {
		if matchPathPattern(path, pattern) {
			return true
		}
	}
	return false
}

func matchPathPattern(path, pattern string) bool {
	if pattern == "*" {
		return true
	}

	if strings.HasPrefix(pattern, "*.") {
		ext := pattern[1:]
		return strings.HasSuffix(path, ext)
	}

	if strings.HasSuffix(pattern, "*") {
		prefix := strings.TrimSuffix(pattern, "*")
		return strings.HasPrefix(path, prefix)
	}

	return path == pattern
}

func (c *CDNConfig) GetCDNURL(originalPath string) string {
	if !c.Enabled {
		return originalPath
	}

	if !c.ShouldCachePath(originalPath) {
		return originalPath
	}

	if c.BaseURL == "" {
		return originalPath
	}

	return strings.TrimSuffix(c.BaseURL, "/") + originalPath
}

func (c *CDNConfig) GetProviderConfig() *CDNProviderConfig {
	if provider, ok := CDNProviders[c.Provider]; ok {
		return &CDNProviderConfig{
			Name:     provider.Name,
			BaseURL:  provider.BaseURL,
			Features: provider.Features,
		}
	}
	custom := CDNProviders["custom"]
	return &CDNProviderConfig{
		Name:     custom.Name,
		BaseURL:  custom.BaseURL,
		Features: custom.Features,
	}
}

type CDNMiddlewareConfig struct {
	EnableStaticCDN  bool     `yaml:"enable_static_cdn"`
	EnableAPICache   bool     `yaml:"enable_api_cache"`
	StaticCDNURL    string   `yaml:"static_cdn_url"`
	ApiCacheTTL     int      `yaml:"api_cache_ttl"`
	ApiCachePrefix  string   `yaml:"api_cache_prefix"`
	IgnorePaths     []string `yaml:"ignore_paths"`
	CacheStatusCodes []int   `yaml:"cache_status_codes"`
}

func NewDefaultCDNMiddlewareConfig() *CDNMiddlewareConfig {
	return &CDNMiddlewareConfig{
		EnableStaticCDN:  true,
		EnableAPICache:   false,
		StaticCDNURL:    "",
		ApiCacheTTL:     300,
		ApiCachePrefix:  "cdn:api:",
		IgnorePaths: []string{
			"/api/v1/captcha/verify",
			"/api/v1/admin/*",
			"/health",
			"/metrics",
		},
		CacheStatusCodes: []int{200, 201, 204, 301, 302},
	}
}

type CDNPerformanceMetrics struct {
	AverageResponseTime time.Duration `json:"average_response_time"`
	P95ResponseTime    time.Duration `json:"p95_response_time"`
	P99ResponseTime    time.Duration `json:"p99_response_time"`
	ThroughputMBps     float64       `json:"throughput_mbps"`
	ConcurrentRequests  int64         `json:"concurrent_requests"`
	ErrorRate          float64       `json:"error_rate"`
}

func GetCDNConfig() *CDNConfig {
	if cfg := GetConfig(); cfg != nil {
		return cfg.CDN
	}
	return nil
}

func IsCDNEnabled() bool {
	if cfg := GetConfig(); cfg != nil && cfg.CDN != nil {
		return cfg.CDN.Enabled
	}
	return false
}

type CDNResourceType string

const (
	CDNResourceTypeStatic   CDNResourceType = "static"
	CDNResourceTypeDynamic  CDNResourceType = "dynamic"
	CDNResourceTypeMedia    CDNResourceType = "media"
	CDNResourceTypeDocument CDNResourceType = "document"
	CDNResourceTypeAPI      CDNResourceType = "api"
)

func GetResourceType(path string) CDNResourceType {
	ext := getFileExtension(path)

	switch ext {
	case ".css", ".js", ".json":
		return CDNResourceTypeStatic
	case ".png", ".jpg", ".jpeg", ".gif", ".svg", ".ico", ".webp", ".avif":
		return CDNResourceTypeMedia
	case ".pdf", ".doc", ".docx", ".xls", ".xlsx", ".ppt", ".pptx":
		return CDNResourceTypeDocument
	case ".html", ".htm":
		return CDNResourceTypeDynamic
	default:
		if strings.HasPrefix(path, "/api/") {
			return CDNResourceTypeAPI
		}
		return CDNResourceTypeStatic
	}
}

func getFileExtension(path string) string {
	if idx := strings.LastIndex(path, "."); idx != -1 {
		return path[idx:]
	}
	return ""
}

type CDNOriginConfig struct {
	Address      string            `yaml:"address"`
	Port         int               `yaml:"port"`
	Protocol     string            `yaml:"protocol"`
	HostHeader   string            `yaml:"host_header"`
	BackupOrigin string            `yaml:"backup_origin"`
	Weight       int               `yaml:"weight"`
	HealthCheck  CDNHealthCheck    `yaml:"health_check"`
}

type CDNHealthCheck struct {
	Enabled    bool   `yaml:"enabled"`
	Path       string `yaml:"path"`
	Interval   int    `yaml:"interval"`
	Timeout    int    `yaml:"timeout"`
	Threshold  int    `yaml:"threshold"`
}

func NewDefaultOriginConfig() *CDNOriginConfig {
	return &CDNOriginConfig{
		Port:      80,
		Protocol:  "http",
		Weight:    100,
		HealthCheck: CDNHealthCheck{
			Enabled:   true,
			Path:      "/health",
			Interval:  30,
			Timeout:   5,
			Threshold: 3,
		},
	}
}

type CDNBandwidthLimit struct {
	MaxBandwidthMBps  float64 `yaml:"max_bandwidth_mbps"`
	WarningThreshold  float64 `yaml:"warning_threshold"`
	BurstMultiplier   float64 `yaml:"burst_multiplier"`
}

func NewDefaultBandwidthLimit() *CDNBandwidthLimit {
	return &CDNBandwidthLimit{
		MaxBandwidthMBps: 1000,
		WarningThreshold: 0.8,
		BurstMultiplier:  1.5,
	}
}

func (c *CDNConfig) GetHeaders() map[string]string {
	headers := make(map[string]string)

	for k, v := range c.Headers {
		headers[k] = v
	}

	headers["Cache-Control"] = fmt.Sprintf("public, max-age=%d", c.CacheMaxAge)

	if c.EnableCompression {
		headers["Accept-Encoding"] = "gzip, deflate, br"
	}

	return headers
}

type CDNRouteRule struct {
	Pattern     string            `yaml:"pattern"`
	Destination string            `yaml:"destination"`
	Cache       CDNCachePolicy   `yaml:"cache"`
	Headers     map[string]string `yaml:"headers"`
	RedirectCode int              `yaml:"redirect_code"`
}

type CDNCachePolicy struct {
	Enabled      bool   `yaml:"enabled"`
	TTL          int    `yaml:"ttl"`
	CacheKey    string `yaml:"cache_key"`
	StaleWhileRevalidate int `yaml:"stale_while_revalidate"`
	StaleIfError        int `yaml:"stale_if_error"`
}

func (c *CDNConfig) GetRouteRules() []CDNRouteRule {
	return []CDNRouteRule{
		{
			Pattern:      "/static/*",
			Destination:  "",
			Cache: CDNCachePolicy{
				Enabled:      true,
				TTL:          86400,
				CacheKey:    "${uri}",
				StaleWhileRevalidate: 3600,
				StaleIfError: 86400,
			},
		},
		{
			Pattern:      "/dist/*",
			Destination:  "",
			Cache: CDNCachePolicy{
				Enabled:      true,
				TTL:          604800,
				CacheKey:    "${uri}",
			},
		},
		{
			Pattern:      "*.{css,js}",
			Destination:  "",
			Cache: CDNCachePolicy{
				Enabled:      true,
				TTL:          2592000,
				CacheKey:    "${uri}",
			},
		},
		{
			Pattern:      "*.{png,jpg,jpeg,gif,svg,ico,webp}",
			Destination:  "",
			Cache: CDNCachePolicy{
				Enabled:      true,
				TTL:          2592000,
				CacheKey:    "${uri}",
			},
		},
	}
}
