package ha

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
)

var (
	instance *Manager
	once     sync.Once
)

type Manager struct {
	config           *ManagerConfig
	cluster         *ClusterManager
	healthChecker   *HealthChecker
	failover        *FailoverController
	dataSync        *DataSyncService
	haProxy         *HighAvailabilityLoadBalancer
	disasterRecovery *DisasterRecovery
	nodeID          string
	role            NodeRole
	mu              sync.RWMutex
	ctx             context.Context
	cancel          context.CancelFunc
	started         atomic.Bool
}

type ManagerConfig struct {
	ClusterName       string
	NodeID           string
	BindAddress      string
	AdvertiseAddress string
	Port             int
	InitialNodes     []string
	Role             NodeRole
	EnableHA          bool
	EnableSync       bool
	EnableBackup     bool
	HealthCheckInterval time.Duration
	FailoverEnabled  bool
}

type NodeRole string

const (
	RolePrimary   NodeRole = "primary"
	RoleSecondary NodeRole = "secondary"
	RoleStandalone NodeRole = "standalone"
)

func DefaultManagerConfig() *ManagerConfig {
	return &ManagerConfig{
		HealthCheckInterval: 10 * time.Second,
		EnableHA:           true,
		EnableSync:         true,
		EnableBackup:       true,
		FailoverEnabled:    true,
	}
}

func NewManager(cfg *ManagerConfig) (*Manager, error) {
	if cfg == nil {
		cfg = DefaultManagerConfig()
	}

	if cfg.NodeID == "" {
		cfg.NodeID = fmt.Sprintf("node-%s", generateNodeID())
	}

	m := &Manager{
		config:  cfg,
		nodeID:  cfg.NodeID,
		role:    cfg.Role,
	}

	if cfg.EnableHA {
		clusterConfig := DefaultClusterConfig(cfg.NodeID)
		clusterConfig.ClusterName = cfg.ClusterName
		clusterConfig.BindAddress = cfg.BindAddress
		clusterConfig.AdvertiseAddress = cfg.AdvertiseAddress
		clusterConfig.InitialNodes = cfg.InitialNodes

		cluster, err := NewClusterManager(clusterConfig)
		if err != nil {
			return nil, fmt.Errorf("failed to create cluster manager: %w", err)
		}
		m.cluster = cluster

		healthChecker := NewHealthChecker(cfg.HealthCheckInterval, 5*time.Second)
		m.healthChecker = healthChecker

		if cfg.FailoverEnabled {
			failoverConfig := DefaultFailoverConfig()
			failover := NewFailoverController(failoverConfig, healthChecker)
			m.failover = failover
		}

		haProxy := NewHighAvailabilityLoadBalancer(DefaultLoadBalancerConfig())
		m.haProxy = haProxy
	}

	if cfg.EnableSync {
		syncConfig := DefaultDataSyncConfig()
		m.dataSync = NewDataSyncService(cfg.NodeID, syncConfig, nil)
	}

	if cfg.EnableBackup {
		backupConfig := DefaultDisasterRecoveryConfig()
		m.disasterRecovery = NewDisasterRecovery(backupConfig, m.healthChecker, m.cluster, m.dataSync)
	}

	return m, nil
}

func GetManager() *Manager {
	return instance
}

func InitManager(cfg *ManagerConfig) error {
	var err error
	once.Do(func() {
		instance, err = NewManager(cfg)
	})
	return err
}

func (m *Manager) Start(ctx context.Context) error {
	if m.started.Load() {
		return fmt.Errorf("manager already started")
	}

	m.ctx, m.cancel = context.WithCancel(ctx)
	m.started.Store(true)

	if m.cluster != nil {
		if err := m.cluster.Start(m.ctx); err != nil {
			return fmt.Errorf("failed to start cluster: %w", err)
		}
	}

	if m.healthChecker != nil {
		m.healthChecker.Start(m.ctx)
	}

	if m.haProxy != nil {
		m.haProxy.Start(m.ctx)
	}

	if m.dataSync != nil {
		m.dataSync.Start(m.ctx)
	}

	if m.disasterRecovery != nil {
		if err := m.disasterRecovery.Start(m.ctx); err != nil {
			return fmt.Errorf("failed to start disaster recovery: %w", err)
		}
	}

	return nil
}

func (m *Manager) Stop() error {
	if !m.started.Load() {
		return nil
	}

	m.cancel()

	if m.disasterRecovery != nil {
		m.disasterRecovery.Stop()
	}

	if m.dataSync != nil {
		m.dataSync.Stop()
	}

	if m.haProxy != nil {
		m.haProxy.Stop()
	}

	if m.healthChecker != nil {
		m.healthChecker.Stop()
	}

	if m.cluster != nil {
		m.cluster.Stop()
	}

	m.started.Store(false)
	return nil
}

func (m *Manager) GetNodeID() string {
	return m.nodeID
}

func (m *Manager) GetRole() NodeRole {
	return m.role
}

func (m *Manager) SetRole(role NodeRole) {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.role = role
}

func (m *Manager) IsLeader() bool {
	if m.cluster != nil {
		return m.cluster.IsLeader()
	}
	return false
}

func (m *Manager) GetClusterManager() *ClusterManager {
	return m.cluster
}

func (m *Manager) GetHealthChecker() *HealthChecker {
	return m.healthChecker
}

func (m *Manager) GetFailoverController() *FailoverController {
	return m.failover
}

func (m *Manager) GetDataSync() *DataSyncService {
	return m.dataSync
}

func (m *Manager) GetHAProxy() *HighAvailabilityLoadBalancer {
	return m.haProxy
}

func (m *Manager) GetDisasterRecovery() *DisasterRecovery {
	return m.disasterRecovery
}

func (m *Manager) GetClusterStatus() map[string]interface{} {
	status := make(map[string]interface{})

	status["node_id"] = m.nodeID
	status["role"] = m.role
	status["started"] = m.started.Load()

	if m.cluster != nil {
		clusterStatus := m.cluster.GetStatus()
		status["cluster"] = map[string]interface{}{
			"state":        clusterStatus.State,
			"is_leader":    clusterStatus.IsLeader,
			"role":         clusterStatus.Role,
			"member_count": len(clusterStatus.Members),
			"term":         clusterStatus.Term,
		}
	}

	if m.healthChecker != nil {
		health := m.healthChecker.GetClusterHealth()
		status["health"] = map[string]interface{}{
			"cluster_status": health.ClusterStatus,
			"healthy_nodes":  health.HealthyNodes,
			"total_nodes":    health.TotalNodes,
			"avg_latency":    health.AvgLatency.String(),
		}
	}

	if m.failover != nil {
		failoverStatus := m.failover.GetClusterStatus()
		status["failover"] = map[string]interface{}{
			"primary_node":    failoverStatus.PrimaryNode,
			"failover_active": failoverStatus.FailoverActive,
			"metrics":        failoverStatus.FailoverMetrics,
		}
	}

	if m.dataSync != nil {
		syncMetrics := m.dataSync.GetMetrics()
		status["sync"] = syncMetrics
	}

	if m.disasterRecovery != nil {
		drStatus := m.disasterRecovery.GetStatus()
		status["disaster_recovery"] = map[string]interface{}{
			"status":           drStatus.Status,
			"last_backup_time": drStatus.LastBackupTime,
			"total_backups":    drStatus.TotalBackups,
		}
	}

	return status
}

func (m *Manager) AddBackend(url string, weight int) {
	if m.haProxy != nil {
		m.haProxy.AddBackend(url, weight)
	}
}

func (m *Manager) RemoveBackend(url string) {
	if m.haProxy != nil {
		m.haProxy.RemoveBackend(url)
	}
}

func (m *Manager) GetBackend(clientIP string) (string, error) {
	if m.haProxy == nil {
		return "", fmt.Errorf("HA proxy not configured")
	}

	backend, err := m.haProxy.SelectBackend(clientIP)
	if err != nil {
		return "", err
	}

	return backend.URL, nil
}

func (m *Manager) RegisterRoutes(r interface{}) {
	if m.healthChecker != nil && m.haProxy != nil {
		haMiddleware := NewHAHealthCheckMiddleware(
			m.healthChecker,
			m.failover,
			m.cluster,
			m.haProxy,
		)

		if gin, ok := r.(interface {
			Group(string) *gin.RouterGroup
		}); ok {
			haMiddleware.RegisterRoutes(gin.Group("").Engine.(*gin.Engine))
		}
	}

	if m.failover != nil && m.haProxy != nil {
		failoverHandler := NewFailoverHandler(m.failover, m.haProxy)
	}

	if m.dataSync != nil {
		syncHandler := NewSyncHandler(m.dataSync)
	}
}

type ClusterDeploymentConfig struct {
	Nodes []NodeConfig `json:"nodes"`
}

type NodeConfig struct {
	NodeID          string `json:"node_id"`
	Address         string `json:"address"`
	AdvertiseAddress string `json:"advertise_address"`
	Port            int    `json:"port"`
	Role            NodeRole `json:"role"`
	Weight          int    `json:"weight"`
	Region          string `json:"region"`
	DataCenter      string `json:"data_center"`
}

func GenerateDeploymentConfig(nodeCount int, basePort int) *ClusterDeploymentConfig {
	config := &ClusterDeploymentConfig{
		Nodes: make([]NodeConfig, nodeCount),
	}

	for i := 0; i < nodeCount; i++ {
		role := RoleSecondary
		if i == 0 {
			role = RolePrimary
		}

		config.Nodes[i] = NodeConfig{
			NodeID:          fmt.Sprintf("node-%d", i+1),
			Address:         fmt.Sprintf("10.0.0.%d", i+10),
			AdvertiseAddress: fmt.Sprintf("10.0.0.%d", i+10),
			Port:            basePort + i,
			Role:            role,
			Weight:          100 - (i * 10),
			Region:          "region-1",
			DataCenter:      fmt.Sprintf("dc-%d", (i%2)+1),
		}
	}

	return config
}

func LoadConfigFromFile(path string) (*ClusterDeploymentConfig, error) {
	return &ClusterDeploymentConfig{}, nil
}

func ValidateDeploymentConfig(config *ClusterDeploymentConfig) error {
	if len(config.Nodes) == 0 {
		return fmt.Errorf("no nodes configured")
	}

	var hasPrimary bool
	for _, node := range config.Nodes {
		if node.NodeID == "" {
			return fmt.Errorf("node ID cannot be empty")
		}
		if node.Address == "" {
			return fmt.Errorf("node address cannot be empty")
		}
		if node.Role == RolePrimary {
			if hasPrimary {
				return fmt.Errorf("multiple primary nodes configured")
			}
			hasPrimary = true
		}
	}

	if !hasPrimary {
		return fmt.Errorf("no primary node configured")
	}

	return nil
}

func (m *Manager) PerformHealthCheck() *ClusterHealth {
	if m.healthChecker == nil {
		return nil
	}
	return m.healthChecker.GetClusterHealth()
}

func (m *Manager) InitiateFailover(targetNode string) error {
	if m.failover == nil {
		return fmt.Errorf("failover not configured")
	}

	currentPrimary := m.failover.GetPrimaryNode()
	if currentPrimary == "" {
		return fmt.Errorf("no primary node")
	}

	return m.failover.ManualFailover(currentPrimary, targetNode)
}

func (m *Manager) PerformBackup() error {
	if m.disasterRecovery == nil {
		return fmt.Errorf("disaster recovery not configured")
	}

	return m.disasterRecovery.PerformBackup(BackupTypeFull)
}

func (m *Manager) PerformRestore(backupID string, components []RestoreComponent) error {
	if m.disasterRecovery == nil {
		return fmt.Errorf("disaster recovery not configured")
	}

	_, err := m.disasterRecovery.PerformRestore(backupID, components)
	return err
}

func (m *Manager) GetMetrics() map[string]interface{} {
	return m.GetClusterStatus()
}
