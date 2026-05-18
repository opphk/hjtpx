package edge

import (
	"context"
	"testing"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/stretchr/testify/assert"
)

type mockEdgeRepo struct {
	nodes              map[string]*model.EdgeNode
	verificationReqs   []*model.EdgeVerificationRequest
}

func (m *mockEdgeRepo) CreateNode(ctx context.Context, node *model.EdgeNode) error {
	m.nodes[node.NodeID] = node
	return nil
}

func (m *mockEdgeRepo) UpdateNode(ctx context.Context, node *model.EdgeNode) error {
	m.nodes[node.NodeID] = node
	return nil
}

func (m *mockEdgeRepo) DeleteNode(ctx context.Context, nodeID string) error {
	delete(m.nodes, nodeID)
	return nil
}

func (m *mockEdgeRepo) GetNodeByNodeID(ctx context.Context, nodeID string) (*model.EdgeNode, error) {
	node, ok := m.nodes[nodeID]
	if !ok {
		return nil, ErrNodeNotFound
	}
	return node, nil
}

func (m *mockEdgeRepo) GetNodeByID(ctx context.Context, id string) (*model.EdgeNode, error) {
	for _, node := range m.nodes {
		if node.ID == id {
			return node, nil
		}
	}
	return nil, ErrNodeNotFound
}

func (m *mockEdgeRepo) ListNodes(ctx context.Context, region, zone string, status model.EdgeNodeStatus) ([]model.EdgeNode, error) {
	var nodes []model.EdgeNode
	for _, node := range m.nodes {
		if region != "" && node.Region != region {
			continue
		}
		if zone != "" && node.Zone != zone {
			continue
		}
		if status != "" && node.Status != status {
			continue
		}
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

func (m *mockEdgeRepo) ListAllNodes(ctx context.Context) ([]model.EdgeNode, error) {
	var nodes []model.EdgeNode
	for _, node := range m.nodes {
		nodes = append(nodes, *node)
	}
	return nodes, nil
}

func (m *mockEdgeRepo) UpdateNodeStatus(ctx context.Context, nodeID string, status model.EdgeNodeStatus) error {
	node, ok := m.nodes[nodeID]
	if !ok {
		return ErrNodeNotFound
	}
	node.Status = status
	return nil
}

func (m *mockEdgeRepo) UpdateNodeHeartbeat(ctx context.Context, nodeID string, loadMetrics model.EdgeLoadMetrics) error {
	node, ok := m.nodes[nodeID]
	if !ok {
		return ErrNodeNotFound
	}
	node.CurrentLoad = loadMetrics
	node.LastHeartbeat = time.Now()
	return nil
}

func (m *mockEdgeRepo) UpdateNodeHealthScore(ctx context.Context, nodeID string, healthScore float64) error {
	node, ok := m.nodes[nodeID]
	if !ok {
		return ErrNodeNotFound
	}
	node.HealthScore = healthScore
	return nil
}

func (m *mockEdgeRepo) CreateVerificationRequest(ctx context.Context, request *model.EdgeVerificationRequest) error {
	reqCopy := *request
	m.verificationReqs = append(m.verificationReqs, &reqCopy)
	return nil
}

func (m *mockEdgeRepo) GetUnsyncedRequests(ctx context.Context, nodeID string, limit int) ([]model.EdgeVerificationRequest, error) {
	var reqs []model.EdgeVerificationRequest
	for _, req := range m.verificationReqs {
		if req.NodeID == nodeID && !req.IsSynced {
			reqs = append(reqs, *req)
			if limit > 0 && len(reqs) >= limit {
				break
			}
		}
	}
	return reqs, nil
}

func (m *mockEdgeRepo) MarkRequestsAsSynced(ctx context.Context, requestIDs []string) error {
	for _, req := range m.verificationReqs {
		for _, id := range requestIDs {
			if req.ID == id {
				req.IsSynced = true
				req.SyncedAt = time.Now()
			}
		}
	}
	return nil
}

func (m *mockEdgeRepo) CreateSyncRecord(ctx context.Context, record *model.EdgeSyncRecord) error {
	return nil
}

func (m *mockEdgeRepo) ListSyncRecords(ctx context.Context, nodeID string, limit int) ([]model.EdgeSyncRecord, error) {
	return nil, nil
}

func newMockRepo() *mockEdgeRepo {
	return &mockEdgeRepo{
		nodes:            make(map[string]*model.EdgeNode),
		verificationReqs: make([]*model.EdgeVerificationRequest, 0),
	}
}

func TestEdgeVerificationService_VerifyCaptcha(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			EnableLocalVerification: true,
			LocalCacheTTLMinutes:    60,
		},
	}
	repo := newMockRepo()
	service := NewEdgeVerificationService(repo, cfg)

	ctx := context.Background()

	t.Run("verify captcha success", func(t *testing.T) {
		result, err := service.VerifyCaptcha(ctx, "node-001", "session-001", map[string]interface{}{
			"type": "slider",
		})
		assert.NoError(t, err)
		assert.True(t, result.Success)
		assert.Equal(t, float64(0.95), result.Score)
		assert.Equal(t, "验证成功", result.Message)
	})

	t.Run("verify captcha with empty type", func(t *testing.T) {
		result, err := service.VerifyCaptcha(ctx, "node-001", "session-002", map[string]interface{}{
			"type": "",
		})
		assert.NoError(t, err)
		assert.False(t, result.Success)
		assert.Equal(t, float64(0.0), result.Score)
		assert.Equal(t, "验证类型不能为空", result.Message)
	})
}

func TestEdgeVerificationService_CacheVerificationResult(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			EnableLocalVerification: true,
			LocalCacheTTLMinutes:    60,
		},
	}
	repo := newMockRepo()
	service := NewEdgeVerificationService(repo, cfg)

	ctx := context.Background()
	result := &EdgeVerificationResult{
		SessionID: "session-001",
		NodeID:    "node-001",
		Success:   true,
		Score:     0.95,
	}

	err := service.CacheVerificationResult(ctx, "session-001", result)
	assert.NoError(t, err)

	cachedResult, err := service.GetCachedVerificationResult(ctx, "session-001")
	assert.NoError(t, err)
	assert.NotNil(t, cachedResult)
	assert.Equal(t, "session-001", cachedResult.SessionID)
	assert.Equal(t, "node-001", cachedResult.NodeID)
	assert.True(t, cachedResult.Success)
}

func TestEdgeVerificationService_GetCachedVerificationResult(t *testing.T) {
	cfg := &config.Config{
		Edge: config.EdgeConfig{
			EnableLocalVerification: true,
			LocalCacheTTLMinutes:    60,
		},
	}
	repo := newMockRepo()
	service := NewEdgeVerificationService(repo, cfg)

	ctx := context.Background()

	t.Run("get cached result", func(t *testing.T) {
		result := &EdgeVerificationResult{
			SessionID: "session-001",
			NodeID:    "node-001",
			Success:   true,
		}
		err := service.CacheVerificationResult(ctx, "session-001", result)
		assert.NoError(t, err)

		cached, err := service.GetCachedVerificationResult(ctx, "session-001")
		assert.NoError(t, err)
		assert.NotNil(t, cached)
	})

	t.Run("get non-existent result", func(t *testing.T) {
		cached, err := service.GetCachedVerificationResult(ctx, "nonexistent")
		assert.NoError(t, err)
		assert.Nil(t, cached)
	})
}