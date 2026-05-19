package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

type AuditLogService struct{}

func NewAuditLogService() *AuditLogService {
	return &AuditLogService{}
}

type AuditLogEntry struct {
	ID             uint                 `json:"id"`
	Timestamp      time.Time            `json:"timestamp"`
	UserID         uint                 `json:"user_id"`
	Username       string               `json:"username"`
	IPAddress      string               `json:"ip_address"`
	UserAgent      string               `json:"user_agent"`
	Endpoint       string               `json:"endpoint"`
	Method         string               `json:"method"`
	StatusCode     int                  `json:"status_code"`
	ResponseTime   int64                `json:"response_time_ms"`
	RequestData    string               `json:"request_data,omitempty"`
	ResponseData   string               `json:"response_data,omitempty"`
	Error          string               `json:"error,omitempty"`
	ResourceType   string               `json:"resource_type"`
	ResourceID     string               `json:"resource_id"`
	Action         string               `json:"action"`
	Changes        string               `json:"changes,omitempty"`
	Metadata       map[string]interface{} `json:"metadata,omitempty"`
}

func (s *AuditLogService) LogRequest(entry *AuditLogEntry) error {
	logEntry := &models.AuditLog{
		LogType:      "api_request",
		Level:        "info",
		UserID:       entry.UserID,
		Username:     entry.Username,
		IPAddress:    entry.IPAddress,
		UserAgent:    entry.UserAgent,
		Action:       entry.Action,
		ResourceType: entry.ResourceType,
		ResourceID:   entry.ResourceID,
		Status:       fmt.Sprintf("%d", entry.StatusCode),
		Metadata:     entry.RequestData,
	}

	if entry.Error != "" {
		logEntry.Level = "error"
		logEntry.ErrorMessage = entry.Error
	}

	if entry.Changes != "" {
		logEntry.Changes = entry.Changes
	}

	return database.DB.Create(logEntry).Error
}

func (s *AuditLogService) LogSecurityEvent(eventType, description string, userID uint, ipAddress string, metadata map[string]interface{}) error {
	metadataJSON, _ := json.Marshal(metadata)

	logEntry := &models.AuditLog{
		LogType:      "security_event",
		Level:        "warning",
		UserID:       userID,
		IPAddress:    ipAddress,
		Action:       eventType,
		ResourceType: "security",
		Status:       "triggered",
		Changes:      description,
		Metadata:     string(metadataJSON),
	}

	return database.DB.Create(logEntry).Error
}

func (s *AuditLogService) LogAuthentication(username string, success bool, ipAddress string, userAgent string) error {
	logEntry := &models.AuditLog{
		LogType:   "authentication",
		Level:     "info",
		Username:  username,
		IPAddress: ipAddress,
		UserAgent: userAgent,
		Action:    "login",
		Status:    "success",
	}

	if !success {
		logEntry.Level = "warning"
		logEntry.Status = "failed"
	}

	return database.DB.Create(logEntry).Error
}

func (s *AuditLogService) LogAdminAction(userID uint, username string, action string, resourceType string, resourceID string, changes string) error {
	logEntry := &models.AuditLog{
		LogType:      "admin_action",
		Level:        "info",
		UserID:       userID,
		Username:     username,
		Action:       action,
		ResourceType: resourceType,
		ResourceID:   resourceID,
		Changes:      changes,
		Status:       "completed",
	}

	return database.DB.Create(logEntry).Error
}