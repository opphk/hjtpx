package repository

import (
	"context"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/database"
	"gorm.io/gorm"
)

type EdgeRepository interface {
	CreateNode(ctx context.Context, node *model.EdgeNode) error
	UpdateNode(ctx context.Context, node *model.EdgeNode) error
	DeleteNode(ctx context.Context, nodeID string) error
	GetNodeByNodeID(ctx context.Context, nodeID string) (*model.EdgeNode, error)
	GetNodeByID(ctx context.Context, id string) (*model.EdgeNode, error)
	ListNodes(ctx context.Context, region, zone string, status model.EdgeNodeStatus) ([]model.EdgeNode, error)
	ListAllNodes(ctx context.Context) ([]model.EdgeNode, error)
	UpdateNodeStatus(ctx context.Context, nodeID string, status model.EdgeNodeStatus) error
	UpdateNodeHeartbeat(ctx context.Context, nodeID string, loadMetrics model.EdgeLoadMetrics) error
	UpdateNodeHealthScore(ctx context.Context, nodeID string, healthScore float64) error
	CreateVerificationRequest(ctx context.Context, request *model.EdgeVerificationRequest) error
	GetUnsyncedRequests(ctx context.Context, nodeID string, limit int) ([]model.EdgeVerificationRequest, error)
	MarkRequestsAsSynced(ctx context.Context, requestIDs []string) error
	CreateSyncRecord(ctx context.Context, record *model.EdgeSyncRecord) error
	ListSyncRecords(ctx context.Context, nodeID string, limit int) ([]model.EdgeSyncRecord, error)
}

type edgeRepository struct {
	db *gorm.DB
}

func NewEdgeRepository() EdgeRepository {
	return &edgeRepository{
		db: database.DB,
	}
}

func (r *edgeRepository) CreateNode(ctx context.Context, node *model.EdgeNode) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Create(node).Error
}

func (r *edgeRepository) UpdateNode(ctx context.Context, node *model.EdgeNode) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Save(node).Error
}

func (r *edgeRepository) DeleteNode(ctx context.Context, nodeID string) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Where("node_id = ?", nodeID).Delete(&model.EdgeNode{}).Error
}

func (r *edgeRepository) GetNodeByNodeID(ctx context.Context, nodeID string) (*model.EdgeNode, error) {
	if r.db == nil {
		return nil, gorm.ErrInvalidData
	}
	var node model.EdgeNode
	err := r.db.WithContext(ctx).Where("node_id = ?", nodeID).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *edgeRepository) GetNodeByID(ctx context.Context, id string) (*model.EdgeNode, error) {
	if r.db == nil {
		return nil, gorm.ErrInvalidData
	}
	var node model.EdgeNode
	err := r.db.WithContext(ctx).Where("id = ?", id).First(&node).Error
	if err != nil {
		return nil, err
	}
	return &node, nil
}

func (r *edgeRepository) ListNodes(ctx context.Context, region, zone string, status model.EdgeNodeStatus) ([]model.EdgeNode, error) {
	if r.db == nil {
		return nil, gorm.ErrInvalidData
	}
	var nodes []model.EdgeNode
	query := r.db.WithContext(ctx)
	if region != "" {
		query = query.Where("region = ?", region)
	}
	if zone != "" {
		query = query.Where("zone = ?", zone)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	err := query.Find(&nodes).Error
	return nodes, err
}

func (r *edgeRepository) ListAllNodes(ctx context.Context) ([]model.EdgeNode, error) {
	return r.ListNodes(ctx, "", "", "")
}

func (r *edgeRepository) UpdateNodeStatus(ctx context.Context, nodeID string, status model.EdgeNodeStatus) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("node_id = ?", nodeID).
		Update("status", status).Error
}

func (r *edgeRepository) UpdateNodeHeartbeat(ctx context.Context, nodeID string, loadMetrics model.EdgeLoadMetrics) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("node_id = ?", nodeID).
		Updates(map[string]interface{}{
			"last_heartbeat": time.Now(),
			"current_load":   loadMetrics,
			"updated_at":     time.Now(),
		}).Error
}

func (r *edgeRepository) UpdateNodeHealthScore(ctx context.Context, nodeID string, healthScore float64) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Model(&model.EdgeNode{}).
		Where("node_id = ?", nodeID).
		Update("health_score", healthScore).Error
}

func (r *edgeRepository) CreateVerificationRequest(ctx context.Context, request *model.EdgeVerificationRequest) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Create(request).Error
}

func (r *edgeRepository) GetUnsyncedRequests(ctx context.Context, nodeID string, limit int) ([]model.EdgeVerificationRequest, error) {
	if r.db == nil {
		return nil, gorm.ErrInvalidData
	}
	var requests []model.EdgeVerificationRequest
	query := r.db.WithContext(ctx).
		Where("node_id = ? AND is_synced = ?", nodeID, false).
		Order("created_at ASC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&requests).Error
	return requests, err
}

func (r *edgeRepository) MarkRequestsAsSynced(ctx context.Context, requestIDs []string) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Model(&model.EdgeVerificationRequest{}).
		Where("id IN ?", requestIDs).
		Updates(map[string]interface{}{
			"is_synced": true,
			"synced_at": time.Now(),
		}).Error
}

func (r *edgeRepository) CreateSyncRecord(ctx context.Context, record *model.EdgeSyncRecord) error {
	if r.db == nil {
		return gorm.ErrInvalidData
	}
	return r.db.WithContext(ctx).Create(record).Error
}

func (r *edgeRepository) ListSyncRecords(ctx context.Context, nodeID string, limit int) ([]model.EdgeSyncRecord, error) {
	if r.db == nil {
		return nil, gorm.ErrInvalidData
	}
	var records []model.EdgeSyncRecord
	query := r.db.WithContext(ctx).
		Where("node_id = ?", nodeID).
		Order("start_time DESC")
	if limit > 0 {
		query = query.Limit(limit)
	}
	err := query.Find(&records).Error
	return records, err
}