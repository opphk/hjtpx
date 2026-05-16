package middleware

import (
	"context"
	"fmt"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

type CDNConfig struct {
	BaseURL        string
	Version        string
	EnableMinify   bool
	EnableCompress bool
	CacheMaxAge    time.Duration
	AllowedPaths   []string
	ExcludedPaths  []string
}

var defaultCDNConfig = &CDNConfig{
	BaseURL:        "",
	Version:        "v1",
	EnableMinify:   true,
	EnableCompress: true,
	CacheMaxAge:    7 * 24 * time.Hour,
	AllowedPaths:   []string{"/static/", "/assets/", "/public/"},
	ExcludedPaths:  []string{},
}

type CDNMiddleware struct {
	config  *CDNConfig
	cdnURLs map[string]string
	mu      sync.RWMutex
}

func NewCDNMiddleware(cfg *CDNConfig) *CDNMiddleware {
	if cfg == nil {
		cfg = defaultCDNConfig
	}

	return &CDNMiddleware{
		config:  cfg,
		cdnURLs: make(map[string]string),
	}
}

func (cm *CDNMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cm.config.BaseURL == "" {
			c.Next()
			return
		}

		path := c.Request.URL.Path

		if !cm.shouldServeFromCDN(path) {
			c.Next()
			return
		}

		cdnURL := cm.getCDNURL(path)
		if cdnURL != "" {
			c.Header("X-CDN-Cache", "HIT")
			c.Header("X-CDN-URL", cdnURL)
			c.Header("Cache-Control", fmt.Sprintf("public, max-age=%d", int(cm.config.CacheMaxAge.Seconds())))
			c.Header("CDN-URL", cdnURL)

			c.Redirect(http.StatusMovedPermanently, cdnURL)
			c.Abort()
			return
		}

		c.Next()
	}
}

func (cm *CDNMiddleware) shouldServeFromCDN(path string) bool {
	if len(cm.config.AllowedPaths) > 0 {
		allowed := false
		for _, prefix := range cm.config.AllowedPaths {
			if strings.HasPrefix(path, prefix) {
				allowed = true
				break
			}
		}
		if !allowed {
			return false
		}
	}

	for _, excluded := range cm.config.ExcludedPaths {
		if strings.HasPrefix(path, excluded) {
			return false
		}
	}

	return true
}

func (cm *CDNMiddleware) getCDNURL(path string) string {
	cm.mu.RLock()
	url, ok := cm.cdnURLs[path]
	cm.mu.RUnlock()

	if ok {
		return url
	}

	url = cm.buildCDNURL(path)

	cm.mu.Lock()
	cm.cdnURLs[path] = url
	cm.mu.Unlock()

	return url
}

func (cm *CDNMiddleware) buildCDNURL(path string) string {
	if cm.config.BaseURL == "" {
		return path
	}

	baseURL := strings.TrimSuffix(cm.config.BaseURL, "/")
	versionedPath := cm.addVersion(path)

	return baseURL + versionedPath
}

func (cm *CDNMiddleware) addVersion(path string) string {
	if cm.config.Version == "" {
		return path
	}

	parts := strings.Split(path, "?")
	pathPart := parts[0]

	if len(parts) > 1 {
		return pathPart + "?v=" + cm.config.Version + "&" + parts[1]
	}

	return pathPart + "?v=" + cm.config.Version
}

func (cm *CDNMiddleware) InvalidateCache(paths []string) error {
	cm.mu.Lock()
	defer cm.mu.Unlock()

	for _, path := range paths {
		delete(cm.cdnURLs, path)
	}

	return nil
}

func (cm *CDNMiddleware) GetStats() *CDNStats {
	cm.mu.RLock()
	defer cm.mu.RUnlock()

	return &CDNStats{
		CachedURLs: len(cm.cdnURLs),
		Config:     cm.config,
	}
}

type CDNStats struct {
	CachedURLs int         `json:"cached_urls"`
	Config     *CDNConfig `json:"config"`
}

type StaticFileHandler struct {
	basePath     string
	cdnMiddleware *CDNMiddleware
	enableETag   bool
	enableGzip   bool
}

func NewStaticFileHandler(basePath string, cdn *CDNMiddleware) *StaticFileHandler {
	return &StaticFileHandler{
		basePath:     basePath,
		cdnMiddleware: cdn,
		enableETag:   true,
		enableGzip:   true,
	}
}

func (sfh *StaticFileHandler) ServeFile(c *gin.Context, filePath string) {
	if sfh.enableETag {
		sfh.serveWithETag(c, filePath)
		return
	}
	c.File(filePath)
}

func (sfh *StaticFileHandler) serveWithETag(c *gin.Context, filePath string) {
	etag := sfh.generateETag(filePath)
	if etag == "" {
		c.File(filePath)
		return
	}

	if c.GetHeader("If-None-Match") == etag {
		c.AbortWithStatus(http.StatusNotModified)
		return
	}

	c.Header("ETag", etag)
	c.File(filePath)
}

func (sfh *StaticFileHandler) generateETag(filePath string) string {
	return ""
}

type AssetManifest struct {
	mu       sync.RWMutex
	manifest map[string]string
}

func NewAssetManifest() *AssetManifest {
	return &AssetManifest{
		manifest: make(map[string]string),
	}
}

func (am *AssetManifest) Set(key, value string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.manifest[key] = value
}

func (am *AssetManifest) Get(key string) (string, bool) {
	am.mu.RLock()
	defer am.mu.RUnlock()
	val, ok := am.manifest[key]
	return val, ok
}

func (am *AssetManifest) GetCDNURL(key string) string {
	am.mu.RLock()
	defer am.mu.RUnlock()

	val, ok := am.manifest[key]
	if !ok {
		return key
	}
	return val
}

func (am *AssetManifest) SetManifest(manifest map[string]string) {
	am.mu.Lock()
	defer am.mu.Unlock()
	am.manifest = manifest
}

type CacheBuster struct {
	cdnURL   string
	version  string
	mu       sync.RWMutex
	urlCache map[string]string
}

func NewCacheBuster(cdnURL, version string) *CacheBuster {
	return &CacheBuster{
		cdnURL:   cdnURL,
		version:  version,
		urlCache: make(map[string]string),
	}
}

func (cb *CacheBuster) GetURL(path string) string {
	cb.mu.RLock()
	if url, ok := cb.urlCache[path]; ok {
		cb.mu.RUnlock()
		return url
	}
	cb.mu.RUnlock()

	url := cb.buildURL(path)

	cb.mu.Lock()
	cb.urlCache[path] = url
	cb.mu.Unlock()

	return url
}

func (cb *CacheBuster) buildURL(path string) string {
	if cb.cdnURL == "" {
		return path
	}

	baseURL := strings.TrimSuffix(cb.cdnURL, "/")
	versionedPath := path

	if !strings.Contains(path, "?") {
		versionedPath = path + "?v=" + cb.version
	} else {
		versionedPath = path + "&v=" + cb.version
	}

	return baseURL + versionedPath
}

func (cb *CacheBuster) Invalidate(paths []string) {
	cb.mu.Lock()
	defer cb.mu.Unlock()

	for _, path := range paths {
		delete(cb.urlCache, path)
	}
}

type ResourcePreloader struct {
	resources []Resource
	mu        sync.RWMutex
}

type Resource struct {
	URL      string `json:"url"`
	Type     string `json:"type"`
	CrossOrigin string `json:"crossorigin,omitempty"`
	As       string `json:"as,omitempty"`
}

func NewResourcePreloader() *ResourcePreloader {
	return &ResourcePreloader{
		resources: make([]Resource, 0),
	}
}

func (rp *ResourcePreloader) Add(url, resourceType string) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.resources = append(rp.resources, Resource{
		URL:  url,
		Type: resourceType,
	})
}

func (rp *ResourcePreloader) AddScript(url string, crossOrigin string) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.resources = append(rp.resources, Resource{
		URL:          url,
		Type:         "script",
		CrossOrigin: crossOrigin,
	})
}

func (rp *ResourcePreloader) AddStyle(url string) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.resources = append(rp.resources, Resource{
		URL:  url,
		Type: "style",
	})
}

func (rp *ResourcePreloader) AddImage(url, as string) {
	rp.mu.Lock()
	defer rp.mu.Unlock()

	rp.resources = append(rp.resources, Resource{
		URL: url,
		Type: "image",
		As:  as,
	})
}

func (rp *ResourcePreloader) GetPreloadLinks() string {
	rp.mu.RLock()
	defer rp.mu.RUnlock()

	var links []string
	for _, r := range rp.resources {
		switch r.Type {
		case "script":
			link := fmt.Sprintf(`<link rel="preload" href="%s" as="script"`, r.URL)
			if r.CrossOrigin != "" {
				link += fmt.Sprintf(` crossorigin="%s"`, r.CrossOrigin)
			}
			link += ">"
			links = append(links, link)
		case "style":
			links = append(links, fmt.Sprintf(`<link rel="preload" href="%s" as="style">`, r.URL))
		case "image":
			as := "image"
			if r.As != "" {
				as = r.As
			}
			links = append(links, fmt.Sprintf(`<link rel="preload" href="%s" as="%s">`, r.URL, as))
		}
	}

	return strings.Join(links, "\n")
}

type DNSPrefetch struct {
	domains []string
	mu      sync.RWMutex
}

func NewDNSPrefetch() *DNSPrefetch {
	return &DNSPrefetch{
		domains: make([]string, 0),
	}
}

func (dp *DNSPrefetch) Add(domain string) {
	dp.mu.Lock()
	defer dp.mu.Unlock()

	for _, d := range dp.domains {
		if d == domain {
			return
		}
	}
	dp.domains = append(dp.domains, domain)
}

func (dp *DNSPrefetch) GetPrefetchLinks() string {
	dp.mu.RLock()
	defer dp.mu.RUnlock()

	var links []string
	for _, domain := range dp.domains {
		links = append(links, fmt.Sprintf(`<link rel="dns-prefetch" href="%s">`, domain))
	}

	return strings.Join(links, "\n")
}

type HealthChecker struct {
	mu          sync.RWMutex
	backends    map[string]*BackendHealth
	checkPeriod time.Duration
	timeout     time.Duration
	stopCh      chan struct{}
}

type BackendHealth struct {
	URL       string
	Healthy   bool
	LastCheck time.Time
	Latency   time.Duration
	Failures  int
	Weight    int
}

func NewHealthChecker(checkPeriod, timeout time.Duration) *HealthChecker {
	return &HealthChecker{
		backends:  make(map[string]*BackendHealth),
		checkPeriod: checkPeriod,
		timeout:   timeout,
		stopCh:    make(chan struct{}),
	}
}

func (hc *HealthChecker) AddBackend(url string, weight int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	hc.backends[url] = &BackendHealth{
		URL:    url,
		Weight: weight,
	}
}

func (hc *HealthChecker) Start(ctx context.Context) {
	ticker := time.NewTicker(hc.checkPeriod)
	defer ticker.Stop()

	hc.checkAll()

	for {
		select {
		case <-ctx.Done():
			return
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.checkAll()
		}
	}
}

func (hc *HealthChecker) Stop() {
	close(hc.stopCh)
}

func (hc *HealthChecker) checkAll() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	for _, backend := range hc.backends {
		go hc.checkBackend(backend)
	}
}

func (hc *HealthChecker) checkBackend(backend *BackendHealth) {
	start := time.Now()

	req, err := http.NewRequest("GET", backend.URL, nil)
	if err != nil {
		hc.recordFailure(backend)
		return
	}

	req = req.WithContext(context.Background())

	client := &http.Client{
		Timeout: hc.timeout,
	}

	resp, err := client.Do(req)
	if err != nil {
		hc.recordFailure(backend)
		return
	}
	defer resp.Body.Close()

	latency := time.Since(start)

	hc.mu.Lock()
	backend.Healthy = resp.StatusCode >= 200 && resp.StatusCode < 400
	backend.LastCheck = time.Now()
	backend.Latency = latency
	if backend.Healthy {
		backend.Failures = 0
	}
	hc.mu.Unlock()
}

func (hc *HealthChecker) recordFailure(backend *BackendHealth) {
	hc.mu.Lock()
	defer hc.mu.Unlock()

	backend.Failures++
	if backend.Failures >= 3 {
		backend.Healthy = false
	}
	backend.LastCheck = time.Now()
}

func (hc *HealthChecker) GetHealthyBackends() []*BackendHealth {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy []*BackendHealth
	for _, backend := range hc.backends {
		if backend.Healthy {
			healthy = append(healthy, backend)
		}
	}

	return healthy
}

func (hc *HealthChecker) GetStats() map[string]interface{} {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	stats := make(map[string]interface{})
	healthy := 0
	unhealthy := 0

	for _, backend := range hc.backends {
		if backend.Healthy {
			healthy++
		} else {
			unhealthy++
		}
	}

	stats["total"] = len(hc.backends)
	stats["healthy"] = healthy
	stats["unhealthy"] = unhealthy

	return stats
}
