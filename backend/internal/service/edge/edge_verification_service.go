package edge

import (
	"context"
	"encoding/json"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type EdgeVerificationService interface {
	VerifyCaptcha(ctx context.Context, nodeID, sessionID string, requestData map[string]interface{}) (*EdgeVerificationResult, error)
	CacheVerificationResult(ctx context.Context, sessionID string, result *EdgeVerificationResult) error
	GetCachedVerificationResult(ctx context.Context, sessionID string) (*EdgeVerificationResult, error)
	StoreVerificationRequest(ctx context.Context, nodeID, sessionID string, request, response []byte, status string) error
	GetPendingRequests(ctx context.Context, nodeID string, limit int) ([]model.EdgeVerificationRequest, error)
	MarkRequestsSynced(ctx context.Context, requestIDs []string) error
}

type EdgeVerificationResult struct {
	SessionID     string                 `json:"session_id"`
	NodeID        string                 `json:"node_id"`
	Success       bool                   `json:"success"`
	Score         float64                `json:"score"`
	Message       string                 `json:"message"`
	VerificationType string              `json:"verification_type"`
	Timestamp     time.Time              `json:"timestamp"`
	Metadata      map[string]interface{} `json:"metadata"`
}

type edgeVerificationService struct {
	repo        repository.EdgeRepository
	cfg         *config.Config
	mu          sync.RWMutex
	localCache  map[string]*EdgeVerificationResult
	cacheTTL    time.Duration
}

func NewEdgeVerificationService(repo repository.EdgeRepository, cfg *config.Config) EdgeVerificationService {
	return &edgeVerificationService{
		repo:       repo,
		cfg:        cfg,
		localCache: make(map[string]*EdgeVerificationResult),
		cacheTTL:   time.Duration(cfg.Edge.LocalCacheTTLMinutes) * time.Minute,
	}
}

func (s *edgeVerificationService) VerifyCaptcha(ctx context.Context, nodeID, sessionID string, requestData map[string]interface{}) (*EdgeVerificationResult, error) {
	verificationType := requestData["type"].(string)
	success := true
	score := 0.95
	message := "验证成功"

	if verificationType == "" {
		success = false
		score = 0.0
		message = "验证类型不能为空"
	}

	result := &EdgeVerificationResult{
		SessionID:        sessionID,
		NodeID:           nodeID,
		Success:          success,
		Score:            score,
		Message:          message,
		VerificationType: verificationType,
		Timestamp:        time.Now(),
		Metadata:         map[string]interface{}{"node_id": nodeID, "processed_at": time.Now().Unix()},
	}

	if s.cfg.Edge.EnableLocalVerification {
		err := s.CacheVerificationResult(ctx, sessionID, result)
		if err != nil {
			return result, err
		}
	}

	requestBytes, _ := json.Marshal(requestData)
	responseBytes, _ := json.Marshal(result)
	status := "success"
	if !success {
		status = "failed"
	}

	go func() {
		storeCtx := context.Background()
		s.StoreVerificationRequest(storeCtx, nodeID, sessionID, requestBytes, responseBytes, status)
	}()

	return result, nil
}

func (s *edgeVerificationService) CacheVerificationResult(ctx context.Context, sessionID string, result *EdgeVerificationResult) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.localCache[sessionID] = result

	if redis.Client != nil {
		cacheKey := "edge:verification:" + sessionID
		data, err := json.Marshal(result)
		if err != nil {
			return err
		}
		return redis.Client.Set(ctx, cacheKey, data, s.cacheTTL).Err()
	}

	return nil
}

func (s *edgeVerificationService) GetCachedVerificationResult(ctx context.Context, sessionID string) (*EdgeVerificationResult, error) {
	s.mu.RLock()
	if result, ok := s.localCache[sessionID]; ok {
		s.mu.RUnlock()
		return result, nil
	}
	s.mu.RUnlock()

	if redis.Client != nil {
		cacheKey := "edge:verification:" + sessionID
		data, err := redis.Client.Get(ctx, cacheKey).Bytes()
		if err != nil {
			return nil, err
		}
		var result EdgeVerificationResult
		err = json.Unmarshal(data, &result)
		return &result, err
	}

	return nil, nil
}

func (s *edgeVerificationService) StoreVerificationRequest(ctx context.Context, nodeID, sessionID string, request, response []byte, status string) error {
	req := model.NewEdgeVerificationRequest(nodeID, sessionID, request, response, status)
	return s.repo.CreateVerificationRequest(ctx, req)
}

func (s *edgeVerificationService) GetPendingRequests(ctx context.Context, nodeID string, limit int) ([]model.EdgeVerificationRequest, error) {
	return s.repo.GetUnsyncedRequests(ctx, nodeID, limit)
}

func (s *edgeVerificationService) MarkRequestsSynced(ctx context.Context, requestIDs []string) error {
	return s.repo.MarkRequestsAsSynced(ctx, requestIDs)
}