package file

import (
	"context"
	"fmt"
	"io"
	"mime/multipart"
	"os"
	"path/filepath"
	"strings"

	"captchax/internal/model"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

const (
	maxFileSize    = 10 * 1024 * 1024
	uploadDir      = "uploads"
)

var allowedMimeTypes = map[string]bool{
	"image/jpeg":      true,
	"image/png":       true,
	"image/gif":       true,
	"image/webp":      true,
	"image/svg+xml":   true,
	"application/pdf": true,
	"text/plain":      true,
	"text/csv":        true,
	"application/json": true,
	"application/zip": true,
	"application/x-gzip": true,
}

type Service struct {
	db        *gorm.DB
	uploadDir string
}

func NewService(db *gorm.DB) *Service {
	if err := os.MkdirAll(uploadDir, 0755); err != nil {
		panic(fmt.Sprintf("failed to create upload directory: %v", err))
	}
	return &Service{
		db:        db,
		uploadDir: uploadDir,
	}
}

func (s *Service) Upload(ctx context.Context, userID uint, fileHeader *multipart.FileHeader) (*model.File, error) {
	if fileHeader.Size > maxFileSize {
		return nil, fmt.Errorf("file size exceeds maximum allowed size of 10MB")
	}

	ext := strings.ToLower(filepath.Ext(fileHeader.Filename))
	mimeType := fileHeader.Header.Get("Content-Type")
	if mimeType == "" {
		mimeType = detectMimeTypeByExt(ext)
	}

	if !allowedMimeTypes[mimeType] {
		return nil, fmt.Errorf("file type %s is not allowed", mimeType)
	}

	src, err := fileHeader.Open()
	if err != nil {
		return nil, fmt.Errorf("failed to open uploaded file: %w", err)
	}
	defer src.Close()

	uniqueName := uuid.New().String() + ext
	filePath := filepath.Join(s.uploadDir, uniqueName)

	dst, err := os.Create(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to create destination file: %w", err)
	}
	defer dst.Close()

	if _, err := io.Copy(dst, src); err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to write file: %w", err)
	}

	file := &model.File{
		UserID:       userID,
		Filename:     uniqueName,
		OriginalName: fileHeader.Filename,
		MimeType:     mimeType,
		Size:         fileHeader.Size,
		Path:         filePath,
	}

	if err := s.db.WithContext(ctx).Create(file).Error; err != nil {
		os.Remove(filePath)
		return nil, fmt.Errorf("failed to save file record: %w", err)
	}

	return file, nil
}

func (s *Service) Download(ctx context.Context, id uint) (*model.File, error) {
	var file model.File
	if err := s.db.WithContext(ctx).First(&file, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return nil, fmt.Errorf("file not found")
		}
		return nil, fmt.Errorf("failed to query file: %w", err)
	}
	return &file, nil
}

func (s *Service) Delete(ctx context.Context, id uint) error {
	var file model.File
	if err := s.db.WithContext(ctx).First(&file, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			return fmt.Errorf("file not found")
		}
		return fmt.Errorf("failed to query file: %w", err)
	}

	if err := s.db.WithContext(ctx).Delete(&file).Error; err != nil {
		return fmt.Errorf("failed to delete file record: %w", err)
	}

	if err := os.Remove(file.Path); err != nil && !os.IsNotExist(err) {
		return fmt.Errorf("failed to remove file from disk: %w", err)
	}

	return nil
}

func (s *Service) List(ctx context.Context, userID uint, page, pageSize int) ([]model.File, int64, error) {
	if page <= 0 {
		page = 1
	}
	if pageSize <= 0 || pageSize > 100 {
		pageSize = 20
	}

	var files []model.File
	var total int64

	query := s.db.WithContext(ctx).Model(&model.File{}).Where("user_id = ?", userID)

	if err := query.Count(&total).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to count files: %w", err)
	}

	offset := (page - 1) * pageSize
	if err := query.Order("created_at DESC").Offset(offset).Limit(pageSize).Find(&files).Error; err != nil {
		return nil, 0, fmt.Errorf("failed to list files: %w", err)
	}

	return files, total, nil
}

func (s *Service) GetUploadDir() string {
	return s.uploadDir
}

func detectMimeTypeByExt(ext string) string {
	switch ext {
	case ".jpg", ".jpeg":
		return "image/jpeg"
	case ".png":
		return "image/png"
	case ".gif":
		return "image/gif"
	case ".webp":
		return "image/webp"
	case ".svg":
		return "image/svg+xml"
	case ".pdf":
		return "application/pdf"
	case ".txt":
		return "text/plain"
	case ".csv":
		return "text/csv"
	case ".json":
		return "application/json"
	case ".zip":
		return "application/zip"
	case ".gz":
		return "application/x-gzip"
	default:
		return "application/octet-stream"
	}
}