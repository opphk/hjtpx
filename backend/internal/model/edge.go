package model

import (
	"time"

	"github.com/google/uuid"
)

type EdgeNode struct {
	ID              string          `json:"id" gorm:"primaryKey"`
	NodeID          string          `json:"node_id" gorm:"uniqueIndex"`
	NodeName        string          `json:"node_name"`
	NodeType        string          `json:"node_type"`
	Region          string          `json:"region"`
	Zone            string          `json:"zone"`
	Status          EdgeNodeStatus  `json:"status"`
	Capacity        EdgeCapacity    `json:"capacity"`
	CurrentLoad     EdgeLoadMetrics `json:"current_load"`
	HealthScore     float64         `json:"health_score"`
	LastHeartbeat   time.Time       `json:"last_heartbeat"`
	LastSyncTime    time.Time       `json:"last_sync_time"`
	CloudEndpoint   string          `json:"cloud_endpoint"`
	IPAddress       string          `json:"ip_address"`
	Version         string          `json:"version"`
	CreatedAt       time.Time       `json:"created_at"`
	UpdatedAt       time.Time       `json:"updated_at"`
}

type EdgeNodeStatus string

const (
	EdgeNodeStatusOnline      EdgeNodeStatus = "online"
	EdgeNodeStatusOffline     EdgeNodeStatus = "offline"
	EdgeNodeStatusDegraded    EdgeNodeStatus = "degraded"
	EdgeNodeStatusMaintenance EdgeNodeStatus = "maintenance"
)

type EdgeCapacity struct {
	MaxRequestsPerSecond   int `json:"max_requests_per_second"`
	MaxConcurrentRequests  int `json:"max_concurrent_requests"`
	MemoryLimitMB          int `json:"memory_limit_mb"`
	CPUCores               int `json:"cpu_cores"`
}

type EdgeLoadMetrics struct {
	CurrentRequestsPerSecond   int     `json:"current_rps"`
	CurrentConcurrentRequests  int     `json:"current_concurrent"`
	MemoryUsageMB              int     `json:"memory_usage_mb"`
	CPUUsagePercent            float64 `json:"cpu_usage_percent"`
	QueueLength                int     `json:"queue_length"`
	ActiveConnections          int     `json:"active_connections"`
}

type EdgeVerificationRequest struct {
	ID         string    `json:"id"`
	NodeID     string    `json:"node_id"`
	SessionID  string    `json:"session_id"`
	Request    []byte    `json:"request_data"`
	Response   []byte    `json:"response_data"`
	Status     string    `json:"status"`
	IsSynced   bool      `json:"is_synced"`
	CreatedAt  time.Time `json:"created_at"`
	SyncedAt   time.Time `json:"synced_at"`
}

type EdgeSyncRecord struct {
	ID              string    `json:"id"`
	NodeID          string    `json:"node_id"`
	SyncType        string    `json:"sync_type"`
	Status          string    `json:"status"`
	RecordsCount    int       `json:"records_count"`
	SuccessCount    int       `json:"success_count"`
	FailedCount     int       `json:"failed_count"`
	StartTime       time.Time `json:"start_time"`
	EndTime         time.Time `json:"end_time"`
	ErrorMessage    string    `json:"error_message"`
}

type EdgeHealthCheckResult struct {
	NodeID           string          `json:"node_id"`
	NodeName         string          `json:"node_name"`
	Status           EdgeNodeStatus  `json:"status"`
	HealthScore      float64         `json:"health_score"`
	ResponseTimeMs   int64           `json:"response_time_ms"`
	Error            string          `json:"error"`
	CheckedAt        time.Time       `json:"checked_at"`
	LoadMetrics      EdgeLoadMetrics `json:"load_metrics"`
}

type EdgeNodeListResponse struct {
	Nodes      []EdgeNode `json:"nodes"`
	TotalCount int        `json:"total_count"`
}

func NewEdgeNode(nodeID, nodeName, nodeType, region, zone string) *EdgeNode {
	return &EdgeNode{
		ID:        uuid.New().String(),
		NodeID:    nodeID,
		NodeName:  nodeName,
		NodeType:  nodeType,
		Region:    region,
		Zone:      zone,
		Status:    EdgeNodeStatusOffline,
		HealthScore: 0.0,
		CreatedAt:  time.Now(),
		UpdatedAt:  time.Now(),
	}
}

func NewEdgeVerificationRequest(nodeID, sessionID string, request, response []byte, status string) *EdgeVerificationRequest {
	return &EdgeVerificationRequest{
		ID:        uuid.New().String(),
		NodeID:    nodeID,
		SessionID: sessionID,
		Request:   request,
		Response:  response,
		Status:    status,
		IsSynced:  false,
		CreatedAt: time.Now(),
	}
}