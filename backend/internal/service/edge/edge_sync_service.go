package edge

import (
	"context"
	"fmt"
	"net/http"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/pkg/config"
)

type EdgeSyncService interface {
	StartSyncScheduler(ctx context.Context) error
	StopSyncScheduler()
	TriggerSync(ctx context.Context, nodeID string) error
	GetSyncRecords(ctx context.Context, nodeID string, limit int) ([]model.EdgeSyncRecord, error)
	SyncVerificationRequests(ctx context.Context, nodeID string) error
	HealthCheck(ctx context.Context, nodeID string) (*model.EdgeHealthCheckResult, error)
}

type edgeSyncService struct {
	repo          repository.EdgeRepository
	cfg           *config.Config
	scheduler     *time.Ticker
	schedulerMu   sync.Mutex
	isRunning     bool
	httpClient    *http.Client
}

func NewEdgeSyncService(repo repository.EdgeRepository, cfg *config.Config) EdgeSyncService {
	return &edgeSyncService{
		repo: repo,
		cfg:  cfg,
		httpClient: &http.Client{
			Timeout: 30 * time.Second,
		},
	}
}

func (s *edgeSyncService) StartSyncScheduler(ctx context.Context) error {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()

	if s.isRunning {
		return fmt.Errorf("sync scheduler is already running")
	}

	interval := time.Duration(s.cfg.Edge.SyncIntervalSecs) * time.Second
	s.scheduler = time.NewTicker(interval)
	s.isRunning = true

	go func() {
		for {
			select {
			case <-s.scheduler.C:
				s.syncAllNodes(ctx)
			case <-ctx.Done():
				s.StopSyncScheduler()
				return
			}
		}
	}()

	return nil
}

func (s *edgeSyncService) StopSyncScheduler() {
	s.schedulerMu.Lock()
	defer s.schedulerMu.Unlock()

	if s.scheduler != nil {
		s.scheduler.Stop()
		s.scheduler = nil
	}
	s.isRunning = false
}

func (s *edgeSyncService) TriggerSync(ctx context.Context, nodeID string) error {
	return s.SyncVerificationRequests(ctx, nodeID)
}

func (s *edgeSyncService) GetSyncRecords(ctx context.Context, nodeID string, limit int) ([]model.EdgeSyncRecord, error) {
	return s.repo.ListSyncRecords(ctx, nodeID, limit)
}

func (s *edgeSyncService) SyncVerificationRequests(ctx context.Context, nodeID string) error {
	startTime := time.Now()
	record := &model.EdgeSyncRecord{
		ID:        uuid.New().String(),
		NodeID:    nodeID,
		SyncType:  "verification_requests",
		Status:    "running",
		StartTime: startTime,
	}

	err := s.repo.CreateSyncRecord(ctx, record)
	if err != nil {
		return err
	}

	requests, err := s.repo.GetUnsyncedRequests(ctx, nodeID, s.cfg.Edge.MaxSyncBatchSize)
	if err != nil {
		return err
	}

	record.RecordsCount = len(requests)
	successCount := 0
	failedCount := 0

	for _, req := range requests {
		err := s.sendToCloud(ctx, req)
		if err != nil {
			failedCount++
		} else {
			successCount++
		}
	}

	if successCount > 0 {
		requestIDs := make([]string, 0, len(requests))
		for _, req := range requests {
			requestIDs = append(requestIDs, req.ID)
		}
		err := s.repo.MarkRequestsAsSynced(ctx, requestIDs)
		if err != nil {
			record.ErrorMessage = err.Error()
		}
	}

	endTime := time.Now()
	record.EndTime = endTime
	record.SuccessCount = successCount
	record.FailedCount = failedCount

	if failedCount == 0 {
		record.Status = "success"
	} else if successCount > 0 {
		record.Status = "partial"
	} else {
		record.Status = "failed"
	}

	return s.repo.UpdateNode(ctx, &model.EdgeNode{
		NodeID:       nodeID,
		LastSyncTime: endTime,
	})
}

func (s *edgeSyncService) sendToCloud(ctx context.Context, request model.EdgeVerificationRequest) error {
	cloudEndpoint := s.cfg.Edge.CloudEndpoint
	if cloudEndpoint == "" {
		return fmt.Errorf("cloud endpoint not configured")
	}

	return nil
}

func (s *edgeSyncService) HealthCheck(ctx context.Context, nodeID string) (*model.EdgeHealthCheckResult, error) {
	node, err := s.repo.GetNodeByNodeID(ctx, nodeID)
	if err != nil {
		return nil, err
	}

	start := time.Now()
	err = s.pingNode(ctx, node.CloudEndpoint)
	responseTime := time.Since(start).Milliseconds()

	healthScore := s.calculateHealthScore(node)

	status := model.EdgeNodeStatusOnline
	if err != nil {
		status = model.EdgeNodeStatusOffline
	} else if healthScore < 60 {
		status = model.EdgeNodeStatusDegraded
	}

	return &model.EdgeHealthCheckResult{
		NodeID:         node.NodeID,
		NodeName:       node.NodeName,
		Status:         status,
		HealthScore:    healthScore,
		ResponseTimeMs: responseTime,
		Error:          "",
		CheckedAt:      time.Now(),
		LoadMetrics:    node.CurrentLoad,
	}, nil
}

func (s *edgeSyncService) calculateHealthScore(node *model.EdgeNode) float64 {
	if node == nil {
		return 0
	}

	score := 100.0

	if node.LastHeartbeat.IsZero() {
		return 0
	}

	heartbeatAge := time.Since(node.LastHeartbeat).Seconds()
	if heartbeatAge > float64(s.cfg.Edge.HeartbeatIntervalSecs*3) {
		score -= 50
	} else if heartbeatAge > float64(s.cfg.Edge.HeartbeatIntervalSecs*2) {
		score -= 25
	}

	cpuLoad := float64(node.CurrentLoad.CPUUsagePercent)
	if cpuLoad > 90 {
		score -= 30
	} else if cpuLoad > 70 {
		score -= 15
	}

	memoryLoad := float64(node.CurrentLoad.MemoryUsageMB) / float64(node.Capacity.MemoryLimitMB)
	if memoryLoad > 0.9 {
		score -= 30
	} else if memoryLoad > 0.7 {
		score -= 15
	}

	rpsLoad := float64(node.CurrentLoad.CurrentRequestsPerSecond) / float64(node.Capacity.MaxRequestsPerSecond)
	if rpsLoad > 0.9 {
		score -= 20
	} else if rpsLoad > 0.7 {
		score -= 10
	}

	if score < 0 {
		score = 0
	}

	return score
}

func (s *edgeSyncService) pingNode(ctx context.Context, endpoint string) error {
	if endpoint == "" {
		return nil
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint+"/health", nil)
	if err != nil {
		return err
	}

	resp, err := s.httpClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode != http.StatusOK {
		return fmt.Errorf("node returned status %d", resp.StatusCode)
	}

	return nil
}

func (s *edgeSyncService) syncAllNodes(ctx context.Context) {
	nodes, err := s.repo.ListAllNodes(ctx)
	if err != nil {
		return
	}

	for _, node := range nodes {
		if node.Status == model.EdgeNodeStatusOnline {
			go s.SyncVerificationRequests(ctx, node.NodeID)
		}
	}
}