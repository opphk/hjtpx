package cdn

import (
	"bytes"
	"context"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"fmt"
	"image"
	"image/jpeg"
	"image/png"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/redis/go-redis/v9"
	"golang.org/x/image/draw"
)

var (
	ErrAssetNotFound      = errors.New("asset not found")
	ErrAssetTooLarge      = errors.New("asset size exceeds limit")
	ErrUnsupportedFormat  = errors.New("unsupported asset format")
	ErrCacheMiss          = errors.New("cache miss")
)

type StaticAssetAccelerator struct {
	redisClient      *redis.Client
	cache            map[string]*CachedAsset
	cacheTTL         time.Duration
	maxCacheSize     int64
	currentCacheSize int64
	originPath       string
	requests         int64
	hits             int64
	mu               sync.RWMutex
}

type CachedAsset struct {
	Path          string    `json:"path"`
	Content       []byte    `json:"content"`
	ContentType   string    `json:"content_type"`
	ETag          string    `json:"etag"`
	LastModified  time.Time `json:"last_modified"`
	CacheTime     time.Time `json:"cache_time"`
	Size          int64     `json:"size"`
	Compressed    bool      `json:"compressed"`
	OptimizeLevel int       `json:"optimize_level"`
}

type AssetResponse struct {
	Content       []byte    `json:"content"`
	ContentType   string    `json:"content_type"`
	ETag          string    `json:"etag"`
	LastModified  time.Time `json:"last_modified"`
	CacheHit      bool      `json:"cache_hit"`
	OptimizeLevel int       `json:"optimize_level"`
	RegionID      string    `json:"region_id"`
}

type CacheStats struct {
	TotalAssets   int     `json:"total_assets"`
	CacheSize     int64   `json:"cache_size_bytes"`
	HitRate       float64 `json:"hit_rate"`
	Requests      int64   `json:"total_requests"`
	Hits          int64   `json:"cache_hits"`
}

type OptimizeOptions struct {
	Compress       bool `json:"compress"`
	Resize         bool `json:"resize"`
	Quality        int  `json:"quality"`
	MaxWidth       int  `json:"max_width"`
	MaxHeight      int  `json:"max_height"`
}

func NewStaticAssetAccelerator(redisClient *redis.Client) *StaticAssetAccelerator {
	return &StaticAssetAccelerator{
		redisClient:  redisClient,
		cache:        make(map[string]*CachedAsset),
		cacheTTL:     24 * time.Hour,
		maxCacheSize: 1024 * 1024 * 1024,
		originPath:   "./static",
	}
}

func (a *StaticAssetAccelerator) recordHit() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests++
	a.hits++
}

func (a *StaticAssetAccelerator) recordMiss() {
	a.mu.Lock()
	defer a.mu.Unlock()
	a.requests++
}

func (a *StaticAssetAccelerator) ServeAsset(ctx context.Context, assetPath string, clientIP string) (*AssetResponse, error) {
	cached, err := a.getFromCache(assetPath)
	if err == nil {
		a.recordHit()
		return &AssetResponse{
			Content:       cached.Content,
			ContentType:   cached.ContentType,
			ETag:          cached.ETag,
			LastModified:  cached.LastModified,
			CacheHit:      true,
			OptimizeLevel: cached.OptimizeLevel,
			RegionID:      "local",
		}, nil
	}

	asset, err := a.loadFromOrigin(assetPath)
	if err != nil {
		return nil, err
	}

	optimized, err := a.optimizeAsset(asset)
	if err != nil {
		return nil, err
	}

	err = a.addToCache(assetPath, optimized)
	if err != nil {
		return nil, err
	}

	a.recordMiss()

	return &AssetResponse{
		Content:       optimized.Content,
		ContentType:   optimized.ContentType,
		ETag:          optimized.ETag,
		LastModified:  optimized.LastModified,
		CacheHit:      false,
		OptimizeLevel: optimized.OptimizeLevel,
		RegionID:      "local",
	}, nil
}

func (a *StaticAssetAccelerator) getFromCache(assetPath string) (*CachedAsset, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	cached, exists := a.cache[assetPath]
	if !exists {
		return nil, ErrCacheMiss
	}

	if time.Since(cached.CacheTime) > a.cacheTTL {
		return nil, ErrCacheMiss
	}

	return cached, nil
}

func (a *StaticAssetAccelerator) addToCache(assetPath string, asset *CachedAsset) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	for a.currentCacheSize+asset.Size > a.maxCacheSize {
		a.evictOldest()
	}

	a.cache[assetPath] = asset
	a.currentCacheSize += asset.Size

	return nil
}

func (a *StaticAssetAccelerator) evictOldest() {
	var oldestKey string
	var oldestTime time.Time

	for key, asset := range a.cache {
		if oldestKey == "" || asset.CacheTime.Before(oldestTime) {
			oldestKey = key
			oldestTime = asset.CacheTime
		}
	}

	if oldestKey != "" {
		a.currentCacheSize -= a.cache[oldestKey].Size
		delete(a.cache, oldestKey)
	}
}

func (a *StaticAssetAccelerator) loadFromOrigin(assetPath string) (*CachedAsset, error) {
	fullPath := filepath.Join(a.originPath, assetPath)

	fileInfo, err := os.Stat(fullPath)
	if err != nil {
		return nil, ErrAssetNotFound
	}

	if fileInfo.Size() > 50*1024*1024 {
		return nil, ErrAssetTooLarge
	}

	content, err := os.ReadFile(fullPath)
	if err != nil {
		return nil, ErrAssetNotFound
	}

	contentType := a.getContentType(assetPath)
	etag := generateETag(content)

	return &CachedAsset{
		Path:          assetPath,
		Content:       content,
		ContentType:   contentType,
		ETag:          etag,
		LastModified:  fileInfo.ModTime(),
		CacheTime:     time.Now(),
		Size:          fileInfo.Size(),
		Compressed:    false,
		OptimizeLevel: 0,
	}, nil
}

func (a *StaticAssetAccelerator) optimizeAsset(asset *CachedAsset) (*CachedAsset, error) {
	optimized := &CachedAsset{
		Path:          asset.Path,
		ContentType:   asset.ContentType,
		ETag:          asset.ETag,
		LastModified:  asset.LastModified,
		CacheTime:     time.Now(),
	}

	switch {
	case strings.HasSuffix(strings.ToLower(asset.Path), ".jpg") ||
		strings.HasSuffix(strings.ToLower(asset.Path), ".jpeg"):
		optimized.Content, optimized.Size = a.optimizeJPEG(asset.Content)
		optimized.OptimizeLevel = 2
		optimized.Compressed = true
	case strings.HasSuffix(strings.ToLower(asset.Path), ".png"):
		optimized.Content, optimized.Size = a.optimizePNG(asset.Content)
		optimized.OptimizeLevel = 2
		optimized.Compressed = true
	case strings.HasSuffix(strings.ToLower(asset.Path), ".css"):
		optimized.Content, optimized.Size = a.minifyCSS(asset.Content)
		optimized.OptimizeLevel = 1
		optimized.Compressed = true
	case strings.HasSuffix(strings.ToLower(asset.Path), ".js"):
		optimized.Content, optimized.Size = a.minifyJS(asset.Content)
		optimized.OptimizeLevel = 1
		optimized.Compressed = true
	default:
		optimized.Content = asset.Content
		optimized.Size = asset.Size
		optimized.OptimizeLevel = 0
		optimized.Compressed = false
	}

	optimized.ETag = generateETag(optimized.Content)

	return optimized, nil
}

func (a *StaticAssetAccelerator) optimizeJPEG(content []byte) ([]byte, int64) {
	img, err := jpeg.Decode(bytes.NewReader(content))
	if err != nil {
		return content, int64(len(content))
	}

	maxDimension := 1920
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	if width > maxDimension || height > maxDimension {
		scale := float64(maxDimension) / float64(max(width, height))
		newWidth := int(float64(width) * scale)
		newHeight := int(float64(height) * scale)

		newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.ApproxBiLinear.Scale(newImg, newImg.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = newImg
	}

	var buf bytes.Buffer
	err = jpeg.Encode(&buf, img, &jpeg.Options{Quality: 80})
	if err != nil {
		return content, int64(len(content))
	}

	result := buf.Bytes()
	return result, int64(len(result))
}

func (a *StaticAssetAccelerator) optimizePNG(content []byte) ([]byte, int64) {
	img, err := png.Decode(bytes.NewReader(content))
	if err != nil {
		return content, int64(len(content))
	}

	maxDimension := 1920
	bounds := img.Bounds()
	width := bounds.Max.X
	height := bounds.Max.Y

	if width > maxDimension || height > maxDimension {
		scale := float64(maxDimension) / float64(max(width, height))
		newWidth := int(float64(width) * scale)
		newHeight := int(float64(height) * scale)

		newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))
		draw.ApproxBiLinear.Scale(newImg, newImg.Bounds(), img, img.Bounds(), draw.Over, nil)
		img = newImg
	}

	var buf bytes.Buffer
	err = png.Encode(&buf, img)
	if err != nil {
		return content, int64(len(content))
	}

	result := buf.Bytes()
	return result, int64(len(result))
}

func (a *StaticAssetAccelerator) minifyCSS(content []byte) ([]byte, int64) {
	result := strings.ReplaceAll(string(content), "\n", "")
	result = strings.ReplaceAll(result, "\t", "")
	result = strings.ReplaceAll(result, "  ", " ")

	var builder strings.Builder
	inComment := false

	for i := 0; i < len(result); i++ {
		if i+1 < len(result) && result[i] == '/' && result[i+1] == '*' {
			inComment = true
			i++
			continue
		}
		if i+1 < len(result) && result[i] == '*' && result[i+1] == '/' {
			inComment = false
			i++
			continue
		}
		if !inComment {
			builder.WriteByte(result[i])
		}
	}

	minified := builder.String()
	return []byte(minified), int64(len(minified))
}

func (a *StaticAssetAccelerator) minifyJS(content []byte) ([]byte, int64) {
	result := strings.ReplaceAll(string(content), "\n", " ")
	result = strings.ReplaceAll(result, "\t", " ")
	result = strings.ReplaceAll(result, "  ", " ")

	return []byte(result), int64(len(result))
}

func (a *StaticAssetAccelerator) getContentType(assetPath string) string {
	lowerPath := strings.ToLower(assetPath)

	switch {
	case strings.HasSuffix(lowerPath, ".html"):
		return "text/html"
	case strings.HasSuffix(lowerPath, ".css"):
		return "text/css"
	case strings.HasSuffix(lowerPath, ".js"):
		return "application/javascript"
	case strings.HasSuffix(lowerPath, ".json"):
		return "application/json"
	case strings.HasSuffix(lowerPath, ".jpg") || strings.HasSuffix(lowerPath, ".jpeg"):
		return "image/jpeg"
	case strings.HasSuffix(lowerPath, ".png"):
		return "image/png"
	case strings.HasSuffix(lowerPath, ".gif"):
		return "image/gif"
	case strings.HasSuffix(lowerPath, ".svg"):
		return "image/svg+xml"
	case strings.HasSuffix(lowerPath, ".ico"):
		return "image/x-icon"
	case strings.HasSuffix(lowerPath, ".txt"):
		return "text/plain"
	default:
		return "application/octet-stream"
	}
}

func generateETag(content []byte) string {
	hash := sha256.Sum256(content)
	return hex.EncodeToString(hash[:])
}

func (a *StaticAssetAccelerator) GetStats() *CacheStats {
	a.mu.RLock()
	defer a.mu.RUnlock()

	return &CacheStats{
		TotalAssets: len(a.cache),
		CacheSize:   a.currentCacheSize,
		HitRate:     float64(a.hits) / float64(a.requests),
		Requests:    a.requests,
		Hits:        a.hits,
	}
}

func (a *StaticAssetAccelerator) Clear() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cache = make(map[string]*CachedAsset)
	a.currentCacheSize = 0
	a.requests = 0
	a.hits = 0
}

func (a *StaticAssetAccelerator) Purge(assetPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if _, exists := a.cache[assetPath]; !exists {
		return ErrCacheMiss
	}
	a.currentCacheSize -= a.cache[assetPath].Size
	delete(a.cache, assetPath)
	return nil
}

func (a *StaticAssetAccelerator) Warmup(paths []string) error {
	for _, path := range paths {
		_, err := a.loadFromOrigin(path)
		if err != nil {
			return err
		}
	}
	return nil
}

func (a *StaticAssetAccelerator) PurgeCache(assetPath string) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if asset, exists := a.cache[assetPath]; exists {
		a.currentCacheSize -= asset.Size
		delete(a.cache, assetPath)
		return nil
	}

	return ErrCacheMiss
}

func (a *StaticAssetAccelerator) ClearCache() {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cache = make(map[string]*CachedAsset)
	a.currentCacheSize = 0
}

func (a *StaticAssetAccelerator) SetCacheTTL(ttl time.Duration) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.cacheTTL = ttl
}

func (a *StaticAssetAccelerator) SetMaxCacheSize(size int64) {
	a.mu.Lock()
	defer a.mu.Unlock()

	a.maxCacheSize = size
}

func (a *StaticAssetAccelerator) WarmupCache(assetPaths []string) error {
	for _, path := range assetPaths {
		_, err := a.ServeAsset(context.Background(), path, "127.0.0.1")
		if err != nil && err != ErrAssetNotFound {
			return err
		}
	}
	return nil
}

func (a *StaticAssetAccelerator) String() string {
	return fmt.Sprintf("StaticAssetAccelerator{cacheSize=%d, assets=%d}",
		a.currentCacheSize, len(a.cache))
}