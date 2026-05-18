package storage

import (
	"bytes"
	"context"
	"crypto/md5"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"image"
	"image/gif"
	"image/jpeg"
	"image/png"
	"io"
	"mime/multipart"
	"net/http"
	"os"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

type StorageType string

const (
	StorageTypeLocal StorageType = "local"
	StorageTypeS3    StorageType = "s3"
	StorageTypeOSS   StorageType = "oss"
	StorageTypeCDN   StorageType = "cdn"
)

type ImageFormat string

const (
	ImageFormatJPEG ImageFormat = "jpeg"
	ImageFormatPNG  ImageFormat = "png"
	ImageFormatGIF  ImageFormat = "gif"
	ImageFormatWEBP ImageFormat = "webp"
)

type ImageOptions struct {
	MaxWidth    int
	MaxHeight   int
	Quality     int
	Format      ImageFormat
	StripMeta   bool
	Progressive bool
}

var defaultImageOptions = &ImageOptions{
	MaxWidth:    1920,
	MaxHeight:   1080,
	Quality:     85,
	Format:      ImageFormatJPEG,
	StripMeta:   true,
	Progressive: true,
}

type StorageConfig struct {
	BasePath       string
	BaseURL        string
	MaxFileSize    int64
	AllowedTypes   []string
	StorageType    StorageType
	CDNBaseURL     string
	EnableCompress bool
	CompressLevel  int
}

var defaultStorageConfig = &StorageConfig{
	BasePath:       "./uploads",
	MaxFileSize:    10 * 1024 * 1024,
	AllowedTypes:   []string{".jpg", ".jpeg", ".png", ".gif", ".webp"},
	StorageType:    StorageTypeLocal,
	EnableCompress: true,
	CompressLevel:  6,
}

type Storage struct {
	config     *StorageConfig
	cdn        *CDNService
	mu         sync.RWMutex
	fileHashes map[string]string
}

type UploadResult struct {
	Path       string `json:"path"`
	URL        string `json:"url"`
	Size       int64  `json:"size"`
	Width      int    `json:"width,omitempty"`
	Height     int    `json:"height,omitempty"`
	Hash       string `json:"hash"`
	Format     string `json:"format"`
	Compressed bool   `json:"compressed"`
}

type ImageInfo struct {
	Width    int
	Height   int
	Format   string
	Size     int64
	HasAlpha bool
}

func NewStorage(cfg *StorageConfig) *Storage {
	if cfg == nil {
		cfg = defaultStorageConfig
	}

	s := &Storage{
		config:     cfg,
		fileHashes: make(map[string]string),
	}

	if cfg.StorageType == StorageTypeCDN || cfg.CDNBaseURL != "" {
		s.cdn = NewCDNService(cfg.CDNBaseURL)
	}

	os.MkdirAll(cfg.BasePath, 0755)

	return s
}

func (s *Storage) Upload(ctx context.Context, file *multipart.FileHeader, opts *ImageOptions) (*UploadResult, error) {
	if opts == nil {
		opts = defaultImageOptions
	}

	if s.config.MaxFileSize > 0 && file.Size > s.config.MaxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed: %d > %d", file.Size, s.config.MaxFileSize)
	}

	ext := strings.ToLower(filepath.Ext(file.Filename))
	if !s.isAllowedType(ext) {
		return nil, fmt.Errorf("file type not allowed: %s", ext)
	}

	src, err := file.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open file: %w", err)
	}
	defer src.Close()

	data, err := io.ReadAll(src)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	hash := md5.Sum(data)
	hashStr := hex.EncodeToString(hash[:])

	s.mu.RLock()
	existingPath, exists := s.fileHashes[hashStr]
	s.mu.RUnlock()

	if exists {
		return &UploadResult{
			Path:   existingPath,
			URL:    s.getURL(existingPath),
			Size:   file.Size,
			Hash:   hashStr,
			Format: ext,
		}, nil
	}

	filename := fmt.Sprintf("%s_%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8], ext)
	relPath := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), filename)
	absPath := filepath.Join(s.config.BasePath, relPath)

	os.MkdirAll(filepath.Dir(absPath), 0755)

	processedData := data
	imgInfo := &ImageInfo{Size: file.Size}

	if s.isImage(ext) && s.config.EnableCompress {
		processedData, imgInfo, err = s.processImage(data, ext, opts)
		if err != nil {
			processedData = data
		}
	}

	if err := os.WriteFile(absPath, processedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	s.mu.Lock()
	s.fileHashes[hashStr] = relPath
	s.mu.Unlock()

	return &UploadResult{
		Path:       relPath,
		URL:        s.getURL(relPath),
		Size:       int64(len(processedData)),
		Width:      imgInfo.Width,
		Height:     imgInfo.Height,
		Hash:       hashStr,
		Format:     ext,
		Compressed: len(processedData) < len(data),
	}, nil
}

func (s *Storage) UploadFromBytes(ctx context.Context, data []byte, filename string, opts *ImageOptions) (*UploadResult, error) {
	if opts == nil {
		opts = defaultImageOptions
	}

	ext := strings.ToLower(filepath.Ext(filename))
	if !s.isAllowedType(ext) {
		return nil, fmt.Errorf("file type not allowed: %s", ext)
	}

	hash := md5.Sum(data)
	hashStr := hex.EncodeToString(hash[:])

	s.mu.RLock()
	existingPath, exists := s.fileHashes[hashStr]
	s.mu.RUnlock()

	if exists {
		return &UploadResult{
			Path:   existingPath,
			URL:    s.getURL(existingPath),
			Size:   int64(len(data)),
			Hash:   hashStr,
			Format: ext,
		}, nil
	}

	processedData := data
	imgInfo := &ImageInfo{Size: int64(len(data))}

	if s.isImage(ext) && s.config.EnableCompress {
		var procErr error
		processedData, imgInfo, procErr = s.processImage(data, ext, opts)
		if procErr != nil {
			processedData = data
		}
	}

	filename = fmt.Sprintf("%s_%s%s", time.Now().Format("20060102150405"), uuid.New().String()[:8], ext)
	relPath := fmt.Sprintf("%s/%s", time.Now().Format("2006/01/02"), filename)
	absPath := filepath.Join(s.config.BasePath, relPath)

	os.MkdirAll(filepath.Dir(absPath), 0755)

	if err := os.WriteFile(absPath, processedData, 0644); err != nil {
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	s.mu.Lock()
	s.fileHashes[hashStr] = relPath
	s.mu.Unlock()

	return &UploadResult{
		Path:       relPath,
		URL:        s.getURL(relPath),
		Size:       int64(len(processedData)),
		Width:      imgInfo.Width,
		Height:     imgInfo.Height,
		Hash:       hashStr,
		Format:     ext,
		Compressed: len(processedData) < len(data),
	}, nil
}

func (s *Storage) processImage(data []byte, format string, opts *ImageOptions) ([]byte, *ImageInfo, error) {
	img, imgFormat, err := image.Decode(bytes.NewReader(data))
	if err != nil {
		return nil, nil, err
	}

	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	resized := img
	if width > opts.MaxWidth || height > opts.MaxHeight {
		resized = resizeImage(img, opts.MaxWidth, opts.MaxHeight)
		bounds = resized.Bounds()
		width = bounds.Dx()
		height = bounds.Dy()
	}

	var buf bytes.Buffer

	switch strings.ToLower(format) {
	case ".jpg", ".jpeg":
		quality := opts.Quality
		if quality <= 0 {
			quality = 85
		}
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: quality})
	case ".png":
		err = png.Encode(&buf, resized)
	case ".gif":
		err = gif.Encode(&buf, resized, nil)
	default:
		err = jpeg.Encode(&buf, resized, &jpeg.Options{Quality: opts.Quality})
		imgFormat = ".jpeg"
	}

	if err != nil {
		return nil, nil, err
	}

	info := &ImageInfo{
		Width:  width,
		Height: height,
		Format: imgFormat,
		Size:   int64(buf.Len()),
	}

	return buf.Bytes(), info, nil
}

func pngCompressionLevel(level int) png.CompressionLevel {
	switch {
	case level <= 1:
		return png.BestSpeed
	case level >= 9:
		return png.BestCompression
	default:
		return png.DefaultCompression
	}
}

func resizeImage(img image.Image, maxWidth, maxHeight int) image.Image {
	bounds := img.Bounds()
	width := bounds.Dx()
	height := bounds.Dy()

	if width <= maxWidth && height <= maxHeight {
		return img
	}

	ratio := float64(width) / float64(height)

	var newWidth, newHeight int
	if width > height {
		newWidth = maxWidth
		newHeight = int(float64(maxWidth) / ratio)
	} else {
		newHeight = maxHeight
		newWidth = int(float64(maxHeight) * ratio)
	}

	if newWidth <= 0 {
		newWidth = 1
	}
	if newHeight <= 0 {
		newHeight = 1
	}

	newImg := image.NewRGBA(image.Rect(0, 0, newWidth, newHeight))

	scaleX := float64(width) / float64(newWidth)
	scaleY := float64(height) / float64(newHeight)

	for y := 0; y < newHeight; y++ {
		for x := 0; x < newWidth; x++ {
			srcX := int(float64(x) * scaleX)
			srcY := int(float64(y) * scaleY)

			if srcX >= width {
				srcX = width - 1
			}
			if srcY >= height {
				srcY = height - 1
			}

			newImg.Set(x, y, img.At(srcX, srcY))
		}
	}

	return newImg
}

func (s *Storage) Delete(ctx context.Context, path string) error {
	absPath := filepath.Join(s.config.BasePath, path)

	if err := os.Remove(absPath); err != nil {
		return fmt.Errorf("failed to delete file: %w", err)
	}

	return nil
}

func (s *Storage) GetURL(path string) string {
	return s.getURL(path)
}

func (s *Storage) getURL(path string) string {
	if s.config.CDNBaseURL != "" {
		return strings.TrimSuffix(s.config.CDNBaseURL, "/") + "/" + path
	}
	return s.config.BaseURL + "/" + path
}

func (s *Storage) Exists(ctx context.Context, path string) bool {
	absPath := filepath.Join(s.config.BasePath, path)
	_, err := os.Stat(absPath)
	return err == nil
}

func (s *Storage) GetFileSize(ctx context.Context, path string) (int64, error) {
	absPath := filepath.Join(s.config.BasePath, path)
	info, err := os.Stat(absPath)
	if err != nil {
		return 0, fmt.Errorf("failed to get file info: %w", err)
	}
	return info.Size(), nil
}

func (s *Storage) isAllowedType(ext string) bool {
	for _, allowed := range s.config.AllowedTypes {
		if strings.EqualFold(allowed, ext) {
			return true
		}
	}
	return false
}

func (s *Storage) isImage(ext string) bool {
	imageExts := []string{".jpg", ".jpeg", ".png", ".gif", ".webp", ".bmp"}
	for _, imgExt := range imageExts {
		if strings.EqualFold(imgExt, ext) {
			return true
		}
	}
	return false
}

func (s *Storage) GetImageInfo(ctx context.Context, path string) (*ImageInfo, error) {
	absPath := filepath.Join(s.config.BasePath, path)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to read file: %w", err)
	}

	cfg, format, err := image.DecodeConfig(bytes.NewReader(data))
	if err != nil {
		return nil, fmt.Errorf("failed to decode image config: %w", err)
	}

	info, err := os.Stat(absPath)
	if err != nil {
		return nil, fmt.Errorf("failed to get file info: %w", err)
	}

	return &ImageInfo{
		Width:  cfg.Width,
		Height: cfg.Height,
		Format: format,
		Size:   info.Size(),
	}, nil
}

func (s *Storage) CompressImage(ctx context.Context, path string, opts *ImageOptions) error {
	if opts == nil {
		opts = defaultImageOptions
	}

	absPath := filepath.Join(s.config.BasePath, path)

	data, err := os.ReadFile(absPath)
	if err != nil {
		return fmt.Errorf("failed to read file: %w", err)
	}

	ext := strings.ToLower(filepath.Ext(path))

	processedData, _, err := s.processImage(data, ext, opts)
	if err != nil {
		return fmt.Errorf("failed to compress image: %w", err)
	}

	if err := os.WriteFile(absPath, processedData, 0644); err != nil {
		return fmt.Errorf("failed to write compressed file: %w", err)
	}

	return nil
}

type CDNService struct {
	baseURL    string
	httpClient *http.Client
	mu         sync.RWMutex
	cache      map[string]string
}

func NewCDNService(baseURL string) *CDNService {
	return &CDNService{
		baseURL: strings.TrimSuffix(baseURL, "/"),
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
		cache: make(map[string]string),
	}
}

func (c *CDNService) GetURL(path string) string {
	return c.baseURL + "/" + path
}

func (c *CDNService) Upload(ctx context.Context, path string, data []byte) error {
	url := c.baseURL + "/upload"

	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(data))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/octet-stream")
	req.Header.Set("X-Path", path)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to upload to CDN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CDN upload failed with status: %d", resp.StatusCode)
	}

	c.mu.Lock()
	c.cache[path] = c.GetURL(path)
	c.mu.Unlock()

	return nil
}

func (c *CDNService) Delete(ctx context.Context, path string) error {
	url := c.baseURL + "/delete/" + path

	req, err := http.NewRequestWithContext(ctx, "DELETE", url, nil)
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to delete from CDN: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CDN delete failed with status: %d", resp.StatusCode)
	}

	c.mu.Lock()
	delete(c.cache, path)
	c.mu.Unlock()

	return nil
}

func (c *CDNService) Invalidate(ctx context.Context, paths []string) error {
	url := c.baseURL + "/invalidate"

	body, _ := json.Marshal(map[string]interface{}{"paths": paths})
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(body))
	if err != nil {
		return fmt.Errorf("failed to create request: %w", err)
	}

	req.Header.Set("Content-Type", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return fmt.Errorf("failed to invalidate CDN cache: %w", err)
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("CDN invalidation failed with status: %d", resp.StatusCode)
	}

	c.mu.Lock()
	for _, path := range paths {
		delete(c.cache, path)
	}
	c.mu.Unlock()

	return nil
}

func (c *CDNService) GetCachedURL(path string) string {
	c.mu.RLock()
	url, ok := c.cache[path]
	c.mu.RUnlock()

	if ok {
		return url
	}
	return c.GetURL(path)
}
