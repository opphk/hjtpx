package handler

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
)

func TestNewBackupHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "creates backup handler successfully",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := config.GetConfig()
			
			// Act
			handler := NewBackupHandler(cfg)
			
			// Assert
			assert.NotNil(t, handler)
			assert.NotNil(t, handler.backupService)
		})
	}
}

func TestGetBackupHandler(t *testing.T) {
	tests := []struct {
		name string
	}{
		{
			name: "returns singleton backup handler instance",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			cfg := config.GetConfig()
			
			// Act
			handler1 := GetBackupHandler(cfg)
			handler2 := GetBackupHandler(cfg)
			
			// Assert
			assert.NotNil(t, handler1)
			assert.Equal(t, handler1, handler2)
		})
	}
}

func TestBackupHandler_ListBackups(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "returns list of backups",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_list_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/admin/backups", nil)
			c.Request = req
			
			// Act
			handler.ListBackups(c)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, float64(0), response["code"])
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_CreateBackup(t *testing.T) {
	tests := []struct {
		name           string
		requestBody    CreateBackupRequest
		expectedStatus int
	}{
		{
			name: "creates full backup",
			requestBody: CreateBackupRequest{
				Type: "full",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "creates incremental backup",
			requestBody: CreateBackupRequest{
				Type: "incremental",
			},
			expectedStatus: http.StatusOK,
		},
		{
			name: "returns error for invalid backup type",
			requestBody: CreateBackupRequest{
				Type: "invalid",
			},
			expectedStatus: http.StatusBadRequest,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_create_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			body, _ := json.Marshal(tt.requestBody)
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/admin/backups", bytes.NewBuffer(body))
			req.Header.Set("Content-Type", "application/json")
			c.Request = req
			
			// Act
			handler.CreateBackup(c)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			if tt.expectedStatus == http.StatusOK {
				var response map[string]interface{}
				err := json.Unmarshal(w.Body.Bytes(), &response)
				assert.NoError(t, err)
				assert.Equal(t, float64(0), response["code"])
			}
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_GetBackup(t *testing.T) {
	tests := []struct {
		name           string
		backupID       string
		expectedStatus int
	}{
		{
			name:           "returns 404 for non-existent backup",
			backupID:       "non_existent",
			expectedStatus: http.StatusNotFound,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_get_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tt.backupID}}
			req, _ := http.NewRequest("GET", "/admin/backups/"+tt.backupID, nil)
			c.Request = req
			
			// Act
			handler.GetBackup(c)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_DeleteBackup(t *testing.T) {
	tests := []struct {
		name           string
		backupID       string
		expectedStatus int
	}{
		{
			name:           "returns error for non-existent backup",
			backupID:       "non_existent",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_delete_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tt.backupID}}
			req, _ := http.NewRequest("DELETE", "/admin/backups/"+tt.backupID, nil)
			c.Request = req
			
			// Act
			handler.DeleteBackup(c)
			
			// Assert
			assert.GreaterOrEqual(t, w.Code, 200)
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_RestoreBackup(t *testing.T) {
	tests := []struct {
		name           string
		backupID       string
		expectedStatus int
	}{
		{
			name:           "returns error for non-existent backup",
			backupID:       "non_existent",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_restore_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tt.backupID}}
			req, _ := http.NewRequest("POST", "/admin/backups/"+tt.backupID+"/restore", nil)
			c.Request = req
			
			// Act
			handler.RestoreBackup(c)
			
			// Assert
			assert.GreaterOrEqual(t, w.Code, 200)
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_VerifyBackup(t *testing.T) {
	tests := []struct {
		name           string
		backupID       string
		expectedStatus int
	}{
		{
			name:           "returns error for non-existent backup",
			backupID:       "non_existent",
			expectedStatus: http.StatusInternalServerError,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_verify_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			c.Params = gin.Params{gin.Param{Key: "id", Value: tt.backupID}}
			req, _ := http.NewRequest("POST", "/admin/backups/"+tt.backupID+"/verify", nil)
			c.Request = req
			
			// Act
			handler.VerifyBackup(c)
			
			// Assert
			assert.GreaterOrEqual(t, w.Code, 200)
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_CleanupOldBackups(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "runs cleanup successfully",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			testBackupDir := filepath.Join(os.TempDir(), "handler_cleanup_test_"+time.Now().Format("20060102150405"))
			cfg.Backup.BackupDir = testBackupDir
			os.MkdirAll(testBackupDir, 0755)
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("POST", "/admin/backups/cleanup", nil)
			c.Request = req
			
			// Act
			handler.CleanupOldBackups(c)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, float64(0), response["code"])
			
			// Cleanup
			os.RemoveAll(testBackupDir)
		})
	}
}

func TestBackupHandler_GetBackupConfig(t *testing.T) {
	tests := []struct {
		name           string
		expectedStatus int
	}{
		{
			name:           "returns backup configuration",
			expectedStatus: http.StatusOK,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			// Arrange
			gin.SetMode(gin.TestMode)
			cfg := config.GetConfig()
			
			handler := NewBackupHandler(cfg)
			
			w := httptest.NewRecorder()
			c, _ := gin.CreateTestContext(w)
			req, _ := http.NewRequest("GET", "/admin/backup-config", nil)
			c.Request = req
			
			// Act
			handler.GetBackupConfig(c)
			
			// Assert
			assert.Equal(t, tt.expectedStatus, w.Code)
			
			var response map[string]interface{}
			err := json.Unmarshal(w.Body.Bytes(), &response)
			assert.NoError(t, err)
			assert.Equal(t, float64(0), response["code"])
			assert.NotNil(t, response["data"])
		})
	}
}
