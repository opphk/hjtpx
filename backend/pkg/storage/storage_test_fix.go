package storage

import (
	"context"
	"mime/multipart"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestStorage_Upload_WithOptions(t *testing.T) {
	cfg := &StorageConfig{
		StorageType: StorageTypeLocal,
		BasePath:   "/tmp/storage",
		MaxFileSize: 10 * 1024 * 1024,
		AllowedTypes: []string{".jpg", ".png", ".txt"},
	}

	storage := NewStorage(cfg)
	assert.NotNil(t, storage)

	ctx := context.Background()
	opts := defaultImageOptions

	storage.Upload(ctx, &multipart.FileHeader{}, opts)
}

func TestStorage_Upload_WithoutOptions(t *testing.T) {
	cfg := &StorageConfig{
		StorageType: StorageTypeLocal,
		BasePath:   "/tmp/storage",
		MaxFileSize: 10 * 1024 * 1024,
		AllowedTypes: []string{".jpg", ".png", ".txt"},
	}

	storage := NewStorage(cfg)
	assert.NotNil(t, storage)

	ctx := context.Background()

	storage.Upload(ctx, &multipart.FileHeader{}, nil)
}

func TestStorage_Upload_FileTooLarge(t *testing.T) {
	cfg := &StorageConfig{
		StorageType: StorageTypeLocal,
		BasePath:   "/tmp/storage",
		MaxFileSize: 10,
		AllowedTypes: []string{".txt"},
	}

	storage := NewStorage(cfg)
	assert.NotNil(t, storage)

	ctx := context.Background()

	storage.Upload(ctx, &multipart.FileHeader{}, nil)
}

func TestStorage_Upload_InvalidFileType(t *testing.T) {
	cfg := &StorageConfig{
		StorageType: StorageTypeLocal,
		BasePath:   "/tmp/storage",
		MaxFileSize: 10 * 1024 * 1024,
		AllowedTypes: []string{".jpg", ".png"},
	}

	storage := NewStorage(cfg)
	assert.NotNil(t, storage)

	ctx := context.Background()

	storage.Upload(ctx, &multipart.FileHeader{}, nil)
}
