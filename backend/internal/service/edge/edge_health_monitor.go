package edge

import (
	"context"
	"fmt"
	"log"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/pkg/config"
)

type EdgeHealthMonitor interface {
	StartMonitoring(ctx context.Context) error
	StopMonitoring()
	GetHealthStatus(ctx context.Context, nodeID string) (*model.EdgeHealthCheckResult, error)
	GetAllNodesHealth(ctx context.Context) ([]model.EdgeHealthCheckResult, error)
	GetNodeStatus(ctx context.Context, nodeID string) (model.EdgeNodeStatus, error)
	UpdateNodeStatus(ctx context.Context, nodeID string, status model.EdgeNodeStatus) error
	RegisterHealthCallback(callback HealthCallback)
}

type HealthCallback func(nodeID string, status model.EdgeNodeStatus, healthScore float64)

type edgeHealthMonitor struct {
	repo         repository.EdgeRepository
	cfg          *config.Config
	syncService  EdgeSyncService
	scheduler    *time.Ticker
	schedulerMu  sync.Mutex
	isRunning    bool
	callbacks    []HealthCallback
	callbacksMu  sync.RWMutex
}

func NewEdgeHealthMonitor(repo repository.EdgeRepository, cfg *config.Config, syncService EdgeSyncService) EdgeHealthMonitor {
	return &edgeHealthMonitor{
		repo:        repo,
		cfg:         cfg,
		syncService: syncService,
		callbacks:   make([]HealthCallback, 0),
	}
}

func (m *edgeHealthMonitor) StartMonitoring(ctx context.Context) error {
	m.schedulerMu.Lock()
	defer m.schedulerMu.Unlock()

	if m.isRunning {
		return fmt.Errorf("health monitor is already running")
	}

	interval := time.Duration(m.cfg.Edge.HealthCheckIntervalSecs) * time.Second
	m.scheduler = time.NewTicker(interval)
	m.isRunning = true

	go func() {
		for {
			select {
			case <-m.scheduler.C:
				m.checkAllNodesHealth(ctx)
			case <-ctx.Done():
				m.StopMonitoring()
				return
			}
		}
	}()

	return nil
}

func (m *edgeHealthMonitor) StopMonitoring() {
	m.schedulerMu.Lock()
	defer m.schedulerMu.Unlock()

	if m.scheduler != nil {
		m.scheduler.Stop()
		m.scheduler = nil
	}
	m.isRunning = false
}

func (m *edgeHealthMonitor) GetHealthStatus(ctx context.Context, nodeID string) (*model.EdgeHealthCheckResult, error) {
	return m.syncService.HealthCheck(ctx, nodeID)
}

func (m *edgeHealthMonitor) GetAllNodesHealth(ctx context.Context) ([]model.EdgeHealthCheckResult, error) {
	nodes, err := m.repo.ListAllNodes(ctx)
	if err != nil {
		return nil, err
	}

	results := make([]model.EdgeHealthCheckResult, 0, len(nodes))
	for _, node := range nodes {
		result, err := m.syncService.HealthCheck(ctx, node.NodeID)
		if err != nil {
			log.Printf("Failed to check health for node %s: %v", node.NodeID, err)
			continue
		}
		results = append(results, *result)
	}

	return results, nil
}

func (m *edgeHealthMonitor) GetNodeStatus(ctx context.Context, nodeID string) (model.EdgeNodeStatus, error) {
	node, err := m.repo.GetNodeByNodeID(ctx, nodeID)
	if err != nil {
		return "", err
	}
	return node.Status, nil
}

func (m *edgeHealthMonitor) UpdateNodeStatus(ctx context.Context, nodeID string, status model.EdgeNodeStatus) error {
	return m.repo.UpdateNodeStatus(ctx, nodeID, status)
}

func (m *edgeHealthMonitor) RegisterHealthCallback(callback HealthCallback) {
	m.callbacksMu.Lock()
	defer m.callbacksMu.Unlock()
	m.callbacks = append(m.callbacks, callback)
}

func (m *edgeHealthMonitor) checkAllNodesHealth(ctx context.Context) {
	nodes, err := m.repo.ListAllNodes(ctx)
	if err != nil {
		log.Printf("Failed to list nodes for health check: %v", err)
		return
	}

	for _, node := range nodes {
		result, err := m.syncService.HealthCheck(ctx, node.NodeID)
		if err != nil {
			log.Printf("Health check failed for node %s: %v", node.NodeID, err)
			continue
		}

		if result.Status != node.Status {
			err := m.repo.UpdateNodeStatus(ctx, node.NodeID, result.Status)
			if err != nil {
				log.Printf("Failed to update status for node %s: %v", node.NodeID, err)
			} else {
				log.Printf("Node %s status changed from %s to %s", node.NodeID, node.Status, result.Status)
				m.notifyCallbacks(node.NodeID, result.Status, result.HealthScore)
			}
		}

		err = m.repo.UpdateNodeHealthScore(ctx, node.NodeID, result.HealthScore)
		if err != nil {
			log.Printf("Failed to update health score for node %s: %v", node.NodeID, err)
		}
	}
}

func (m *edgeHealthMonitor) notifyCallbacks(nodeID string, status model.EdgeNodeStatus, healthScore float64) {
	m.callbacksMu.RLock()
	defer m.callbacksMu.RUnlock()

	for _, callback := range m.callbacks {
		go callback(nodeID, status, healthScore)
	}
}