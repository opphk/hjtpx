package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

type SyncStrategy string

const (
	SyncStrategyEager     SyncStrategy = "eager"
	SyncStrategyLazy      SyncStrategy = "lazy"
	SyncStrategyEventual  SyncStrategy = "eventual"
	SyncStrategyQuorum    SyncStrategy = "quorum"
)

type SyncState string

const (
	SyncStatePending   SyncState = "pending"
	SyncStateSyncing   SyncState = "syncing"
	SyncStateSynced    SyncState = "synced"
	SyncStateConflict  SyncState = "conflict"
	SyncStateFailed    SyncState = "failed"
)

type SyncOperation struct {
	ID          string
	Type        SyncOperationType
	Key         string
	Value       interface{}
	Timestamp   time.Time
	Version     int64
	SourceNode  string
	TargetNodes []string
	Status      SyncState
	Retries     int32
	Error       error
}

type SyncOperationType string

const (
	SyncOpSet         SyncOperationType = "set"
	SyncOpDelete      SyncOperationType = "delete"
	SyncOpUpdate      SyncOperationType = "update"
	SyncOpFullSync    SyncOperationType = "full_sync"
	SyncOpDeltaSync   SyncOperationType = "delta_sync"
)

type SyncEvent struct {
	Type       SyncEventType
	Operation  *SyncOperation
	Timestamp  time.Time
	NodeID     string
	Error      error
	Metadata   map[string]interface{}
}

type SyncEventType string

const (
	SyncEventStart          SyncEventType = "sync_start"
	SyncEventComplete       SyncEventType = "sync_complete"
	SyncEventFailed         SyncEventType = "sync_failed"
	SyncEventConflict       SyncEventType = "sync_conflict"
	SyncEventNodeJoined     SyncEventType = "node_joined"
	SyncEventNodeLeft       SyncEventType = "node_left"
	SyncEventPartition      SyncEventType = "partition"
	SyncEventRecovery       SyncEventType = "recovery"
)

type DataSyncConfig struct {
	Strategy           SyncStrategy
	SyncInterval       time.Duration
	SyncTimeout        time.Duration
	RetryAttempts      int32
	RetryDelay         time.Duration
	BatchSize          int32
	EnableCompression  bool
	EnableEncryption   bool
	EncryptionKey      []byte
	QuorumSize         int32
	HeartbeatInterval  time.Duration
	HeartbeatTimeout   time.Duration
}

func DefaultDataSyncConfig() *DataSyncConfig {
	return &DataSyncConfig{
		Strategy:          SyncStrategyEventual,
		SyncInterval:      5 * time.Second,
		SyncTimeout:       30 * time.Second,
		RetryAttempts:     3,
		RetryDelay:        1 * time.Second,
		BatchSize:         100,
		EnableCompression: true,
		EnableEncryption:  false,
		QuorumSize:        2,
		HeartbeatInterval: 3 * time.Second,
		HeartbeatTimeout:  10 * time.Second,
	}
}

type DataSyncService struct {
	config         *DataSyncConfig
	nodeID         string
	nodes          map[string]*SyncNode
	dataStore      SyncDataStore
	operationLog   []*SyncOperation
	pendingOps     chan *SyncOperation
	completedOps   chan *SyncOperation
	failedOps      chan *SyncOperation
	mu             sync.RWMutex
	ctx            context.Context
	cancel         context.CancelFunc
	wg             sync.WaitGroup
	eventHandlers  []SyncEventHandler
	versionVector  map[string]int64
	partitionMap   map[string]string
	leaderNode     atomic.Value
	isLeader       atomic.Bool
	metrics        *SyncMetrics
}

type SyncNode struct {
	NodeID       string
	Address      string
	LastSync     time.Time
	Version      int64
	Status       HealthStatus
	Priority     int
	Region       string
	DataCenter   string
	IsPrimary    bool
	SyncState    SyncState
	pendingOps   int32
	completedOps int64
	failedOps    int64
	mu           sync.RWMutex
}

type SyncDataStore interface {
	Get(ctx context.Context, key string) (interface{}, error)
	Set(ctx context.Context, key string, value interface{}) error
	Delete(ctx context.Context, key string) error
	Keys(ctx context.Context, pattern string) ([]string, error)
	GetAll(ctx context.Context) (map[string]interface{}, error)
	SetWithTTL(ctx context.Context, key string, value interface{}, ttl time.Duration) error
}

type SyncEventHandler func(event *SyncEvent)

type SyncMetrics struct {
	TotalSyncOps     atomic.Int64
	SuccessfulSyncs  atomic.Int64
	FailedSyncs      atomic.Int64
	AvgLatency       atomic.Int64
	LastSyncTime     atomic.Int64
	BytesTransferred atomic.Int64
	Conflicts        atomic.Int64
	mu               sync.RWMutex
	latencies        []time.Duration
}

func NewSyncMetrics() *SyncMetrics {
	return &SyncMetrics{
		latencies: make([]time.Duration, 0, 1000),
	}
}

func (sm *SyncMetrics) RecordSync(duration time.Duration, success bool, bytes int64) {
	sm.TotalSyncOps.Add(1)
	if success {
		sm.SuccessfulSyncs.Add(1)
	} else {
		sm.FailedSyncs.Add(1)
	}
	sm.BytesTransferred.Add(bytes)
	sm.LastSyncTime.Store(time.Now().UnixNano())

	sm.mu.Lock()
	sm.latencies = append(sm.latencies, duration)
	if len(sm.latencies) > 1000 {
		sm.latencies = sm.latencies[1:]
	}
	var total int64
	for _, l := range sm.latencies {
		total += l.Nanoseconds()
	}
	sm.AvgLatency.Store(total / int64(len(sm.latencies)))
	sm.mu.Unlock()
}

func (sm *SyncMetrics) RecordConflict() {
	sm.Conflicts.Add(1)
}

func NewDataSyncService(nodeID string, config *DataSyncConfig, store SyncDataStore) *DataSyncService {
	if config == nil {
		config = DefaultDataSyncConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	return &DataSyncService{
		config:        config,
		nodeID:        nodeID,
		nodes:         make(map[string]*SyncNode),
		dataStore:     store,
		operationLog:  make([]*SyncOperation, 0, 10000),
		pendingOps:    make(chan *SyncOperation, 1000),
		completedOps: make(chan *SyncOperation, 1000),
		failedOps:    make(chan *SyncOperation, 1000),
		versionVector: make(map[string]int64),
		partitionMap: make(map[string]string),
		ctx:          ctx,
		cancel:       cancel,
		metrics:      NewSyncMetrics(),
	}
}

func (ds *DataSyncService) Start(ctx context.Context) {
	ds.ctx, ds.cancel = context.WithCancel(ctx)

	ds.wg.Add(1)
	go ds.operationProcessor()

	ds.wg.Add(1)
	go ds.syncScheduler()

	ds.wg.Add(1)
	go ds.heartbeatChecker()

	ds.wg.Add(1)
	go ds.metricsCollector()
}

func (ds *DataSyncService) Stop() {
	ds.cancel()
	ds.wg.Wait()
}

func (ds *DataSyncService) AddNode(nodeID, address string, priority int) {
	ds.mu.Lock()
	defer ds.mu.Unlock()

	ds.nodes[nodeID] = &SyncNode{
		NodeID:   nodeID,
		Address:  address,
		Priority: priority,
		Status:   StatusUnknown,
		SyncState: SyncStatePending,
	}
}

func (ds *DataSyncService) RemoveNode(nodeID string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	delete(ds.nodes, nodeID)
	delete(ds.versionVector, nodeID)

	ds.logEvent(&SyncEvent{
		Type:      SyncEventNodeLeft,
		Timestamp: time.Now(),
		NodeID:    nodeID,
	})
}

func (ds *DataSyncService) SetDataStore(store SyncDataStore) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.dataStore = store
}

func (ds *DataSyncService) Set(key string, value interface{}) error {
	return ds.setWithStrategy(key, value, SyncOpSet)
}

func (ds *DataSyncService) setWithStrategy(key string, value interface{}, opType SyncOperationType) error {
	ds.mu.Lock()
	ds.versionVector[ds.nodeID]++
	localVersion := ds.versionVector[ds.nodeID]
	ds.mu.Unlock()

	operation := &SyncOperation{
		ID:         fmt.Sprintf("%s-%d", ds.nodeID, localVersion),
		Type:       opType,
		Key:        key,
		Value:      value,
		Timestamp: time.Now(),
		Version:    localVersion,
		SourceNode: ds.nodeID,
		Status:     SyncStatePending,
	}

	if ds.config.Strategy == SyncStrategyEager {
		if err := ds.dataStore.Set(context.Background(), key, value); err != nil {
			operation.Status = SyncStateFailed
			operation.Error = err
			ds.failedOps <- operation
			return err
		}
	}

	ds.pendingOps <- operation

	if ds.config.Strategy == SyncStrategyEager {
		return ds.waitForSync(operation)
	}

	return nil
}

func (ds *DataSyncService) Delete(key string) error {
	return ds.setWithStrategy(key, nil, SyncOpDelete)
}

func (ds *DataSyncService) Get(key string) (interface{}, error) {
	return ds.dataStore.Get(context.Background(), key)
}

func (ds *DataSyncService) SyncToNode(targetNodeID string, operation *SyncOperation) error {
	ds.mu.RLock()
	targetNode, exists := ds.nodes[targetNodeID]
	ds.mu.RUnlock()

	if !exists {
		return fmt.Errorf("target node not found: %s", targetNodeID)
	}

	ctx, cancel := context.WithTimeout(context.Background(), ds.config.SyncTimeout)
	defer cancel()

	operation.TargetNodes = append(operation.TargetNodes, targetNodeID)
	operation.Status = SyncStateSyncing

	startTime := time.Now()
	
	err := ds.sendSyncData(ctx, targetNode, operation)

	duration := time.Since(startTime)
	if err != nil {
		operation.Status = SyncStateFailed
		operation.Error = err
		operation.Retries++
		ds.failedOps <- operation
		ds.metrics.RecordSync(duration, false, 0)
		return err
	}

	operation.Status = SyncStateSynced
	ds.completedOps <- operation
	ds.metrics.RecordSync(duration, true, 0)

	return nil
}

func (ds *DataSyncService) sendSyncData(ctx context.Context, node *SyncNode, op *SyncOperation) error {
	node.mu.Lock()
	defer node.mu.Unlock()

	node.LastSync = time.Now()
	node.SyncState = SyncStateSyncing

	data, err := json.Marshal(op)
	if err != nil {
		return fmt.Errorf("failed to marshal operation: %w", err)
	}

	if ds.config.EnableCompression {
		data, err = ds.compressData(data)
		if err != nil {
			return fmt.Errorf("failed to compress data: %w", err)
		}
	}

	_, err = ds.sendToNode(ctx, node.Address, data)
	if err != nil {
		return fmt.Errorf("failed to send to node %s: %w", node.NodeID, err)
	}

	return nil
}

func (ds *DataSyncService) sendToNode(ctx context.Context, address string, data []byte) ([]byte, error) {
	return data, nil
}

func (ds *DataSyncService) compressData(data []byte) ([]byte, error) {
	return data, nil
}

func (ds *DataSyncService) decompressData(data []byte) ([]byte, error) {
	return data, nil
}

func (ds *DataSyncService) operationProcessor() {
	defer ds.wg.Done()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case op := <-ds.pendingOps:
			ds.processOperation(op)
		}
	}
}

func (ds *DataSyncService) processOperation(op *SyncOperation) {
	ds.mu.RLock()
	targetNodes := make([]string, 0)
	for nodeID := range ds.nodes {
		if nodeID != ds.nodeID {
			targetNodes = append(targetNodes, nodeID)
		}
	}
	ds.mu.RUnlock()

	if len(targetNodes) == 0 {
		op.Status = SyncStateSynced
		ds.completedOps <- op
		return
	}

	var wg sync.WaitGroup
	var successCount int32

	for _, targetNodeID := range targetNodes {
		wg.Add(1)
		go func(nodeID string) {
			defer wg.Done()
			if err := ds.SyncToNode(nodeID, op); err == nil {
				atomic.AddInt32(&successCount, 1)
			}
		}(targetNodeID)
	}

	wg.Wait()

	if ds.config.Strategy == SyncStrategyQuorum {
		quorum := ds.config.QuorumSize
		if atomic.LoadInt32(&successCount) >= quorum {
			op.Status = SyncStateSynced
			ds.completedOps <- op
		} else {
			op.Status = SyncStateFailed
			ds.failedOps <- op
		}
	} else if atomic.LoadInt32(&successCount) == int32(len(targetNodes)) {
		op.Status = SyncStateSynced
		ds.completedOps <- op
	} else if atomic.LoadInt32(&successCount) > 0 {
		op.Status = SyncStateSynced
		ds.completedOps <- op
	} else {
		op.Status = SyncStateFailed
		ds.failedOps <- op
	}
}

func (ds *DataSyncService) syncScheduler() {
	defer ds.wg.Done()

	ticker := time.NewTicker(ds.config.SyncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.performScheduledSync()
		}
	}
}

func (ds *DataSyncService) performScheduledSync() {
	ds.mu.RLock()
	var nodes []*SyncNode
	for _, node := range ds.nodes {
		if node.NodeID != ds.nodeID {
			nodes = append(nodes, node)
		}
	}
	ds.mu.RUnlock()

	for _, node := range nodes {
		go ds.fullSync(node.NodeID)
	}
}

func (ds *DataSyncService) fullSync(targetNodeID string) error {
	ds.logEvent(&SyncEvent{
		Type:      SyncEventStart,
		Timestamp: time.Now(),
		NodeID:    targetNodeID,
		Metadata:  map[string]interface{}{"type": "full_sync"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), ds.config.SyncTimeout)
	defer cancel()

	allData, err := ds.dataStore.GetAll(ctx)
	if err != nil {
		ds.logEvent(&SyncEvent{
			Type:      SyncEventFailed,
			Timestamp: time.Now(),
			NodeID:    targetNodeID,
			Error:     err,
		})
		return err
	}

	for key, value := range allData {
		op := &SyncOperation{
			ID:        fmt.Sprintf("%s-full-%s-%d", ds.nodeID, key, time.Now().UnixNano()),
			Type:      SyncOpFullSync,
			Key:       key,
			Value:     value,
			Timestamp: time.Now(),
			SourceNode: ds.nodeID,
		}
		if err := ds.SyncToNode(targetNodeID, op); err != nil {
			continue
		}
	}

	ds.logEvent(&SyncEvent{
		Type:      SyncEventComplete,
		Timestamp: time.Now(),
		NodeID:    targetNodeID,
		Metadata:  map[string]interface{}{"type": "full_sync", "keys_synced": len(allData)},
	})

	return nil
}

func (ds *DataSyncService) deltaSync(targetNodeID string) error {
	ds.logEvent(&SyncEvent{
		Type:      SyncEventStart,
		Timestamp: time.Now(),
		NodeID:    targetNodeID,
		Metadata:  map[string]interface{}{"type": "delta_sync"},
	})

	ctx, cancel := context.WithTimeout(context.Background(), ds.config.SyncTimeout)
	defer cancel()

	ds.mu.RLock()
	localVersion := ds.versionVector[ds.nodeID]
	ds.mu.RUnlock()
	_ = localVersion

	allData, err := ds.dataStore.GetAll(ctx)
	if err != nil {
		return err
	}

	count := 0
	for key, value := range allData {
		ds.mu.RLock()
		partitionOwner, partitioned := ds.partitionMap[key]
		ds.mu.RUnlock()

		if partitioned && partitionOwner != targetNodeID {
			continue
		}

		op := &SyncOperation{
			ID:        fmt.Sprintf("%s-delta-%s-%d", ds.nodeID, key, time.Now().UnixNano()),
			Type:      SyncOpDeltaSync,
			Key:       key,
			Value:     value,
			Timestamp: time.Now(),
			SourceNode: ds.nodeID,
		}
		if err := ds.SyncToNode(targetNodeID, op); err == nil {
			count++
		}
	}

	ds.logEvent(&SyncEvent{
		Type:      SyncEventComplete,
		Timestamp: time.Now(),
		NodeID:    targetNodeID,
		Metadata:  map[string]interface{}{"type": "delta_sync", "keys_synced": count},
	})

	return nil
}

func (ds *DataSyncService) heartbeatChecker() {
	defer ds.wg.Done()

	ticker := time.NewTicker(ds.config.HeartbeatInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.checkNodeHealth()
		}
	}
}

func (ds *DataSyncService) checkNodeHealth() {
	ds.mu.RLock()
	for _, node := range ds.nodes {
		if node.NodeID == ds.nodeID {
			continue
		}

		timeSinceLastSync := time.Since(node.LastSync)
		if timeSinceLastSync > ds.config.HeartbeatTimeout {
			node.mu.Lock()
			node.Status = StatusUnhealthy
			node.SyncState = SyncStateFailed
			node.mu.Unlock()

			ds.logEvent(&SyncEvent{
				Type:      SyncEventPartition,
				Timestamp: time.Now(),
				NodeID:    node.NodeID,
				Metadata:  map[string]interface{}{"last_sync": node.LastSync},
			})
		}
	}
	ds.mu.RUnlock()
}

func (ds *DataSyncService) metricsCollector() {
	defer ds.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-ds.ctx.Done():
			return
		case <-ticker.C:
			ds.collectMetrics()
		case op := <-ds.completedOps:
			ds.recordCompletedOp(op)
		case op := <-ds.failedOps:
			ds.recordFailedOp(op)
		}
	}
}

func (ds *DataSyncService) recordCompletedOp(op *SyncOperation) {
	ds.mu.Lock()
	ds.operationLog = append(ds.operationLog, op)
	if len(ds.operationLog) > 10000 {
		ds.operationLog = ds.operationLog[1:]
	}
	ds.mu.Unlock()

	if err := ds.dataStore.Set(context.Background(), op.Key, op.Value); err == nil {
		if op.Type == SyncOpSet || op.Type == SyncOpUpdate {
			ds.mu.Lock()
			ds.versionVector[ds.nodeID] = op.Version
			ds.mu.Unlock()
		}
	}
}

func (ds *DataSyncService) recordFailedOp(op *SyncOperation) {
	ds.mu.Lock()
	op.Retries++
	if op.Retries < ds.config.RetryAttempts {
		ds.mu.Unlock()

		time.Sleep(ds.config.RetryDelay)
		ds.pendingOps <- op
		return
	}
	ds.operationLog = append(ds.operationLog, op)
	ds.mu.Unlock()
}

func (ds *DataSyncService) collectMetrics() {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	var totalPending int32
	var totalCompleted, totalFailed int64

	for _, node := range ds.nodes {
		node.mu.RLock()
		totalPending += node.pendingOps
		totalCompleted += node.completedOps
		totalFailed += node.failedOps
		node.mu.RUnlock()
	}
}

func (ds *DataSyncService) waitForSync(op *SyncOperation) error {
	timeout := time.After(ds.config.SyncTimeout)
	for {
		select {
		case <-ds.ctx.Done():
			return context.Canceled
		case <-timeout:
			return fmt.Errorf("sync timeout for operation %s", op.ID)
		case completedOp := <-ds.completedOps:
			if completedOp.ID == op.ID {
				return nil
			}
		case failedOp := <-ds.failedOps:
			if failedOp.ID == op.ID {
				return failedOp.Error
			}
		}
	}
}

func (ds *DataSyncService) SetPartition(key, ownerNodeID string) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.partitionMap[key] = ownerNodeID
}

func (ds *DataSyncService) GetPartitionOwner(key string) (string, bool) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()
	owner, exists := ds.partitionMap[key]
	return owner, exists
}

func (ds *DataSyncService) logEvent(event *SyncEvent) {
	for _, handler := range ds.eventHandlers {
		go handler(event)
	}
}

func (ds *DataSyncService) AddEventHandler(handler SyncEventHandler) {
	ds.mu.Lock()
	defer ds.mu.Unlock()
	ds.eventHandlers = append(ds.eventHandlers, handler)
}

func (ds *DataSyncService) GetMetrics() map[string]interface{} {
	m := ds.metrics
	return map[string]interface{}{
		"total_sync_ops":     m.TotalSyncOps.Load(),
		"successful_syncs":    m.SuccessfulSyncs.Load(),
		"failed_syncs":        m.FailedSyncs.Load(),
		"avg_latency_ms":      m.AvgLatency.Load() / 1e6,
		"bytes_transferred":   m.BytesTransferred.Load(),
		"conflicts":           m.Conflicts.Load(),
	}
}

func (ds *DataSyncService) GetOperationLog(limit int) []*SyncOperation {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	if limit <= 0 || limit > len(ds.operationLog) {
		limit = len(ds.operationLog)
	}

	log := make([]*SyncOperation, limit)
	copy(log, ds.operationLog[len(ds.operationLog)-limit:])
	return log
}

func (ds *DataSyncService) GetNodeStatus(nodeID string) (*SyncNode, error) {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	node, exists := ds.nodes[nodeID]
	if !exists {
		return nil, fmt.Errorf("node not found: %s", nodeID)
	}

	node.mu.RLock()
	defer node.mu.RUnlock()

	return &SyncNode{
		NodeID:       node.NodeID,
		Address:      node.Address,
		LastSync:    node.LastSync,
		Version:     node.Version,
		Status:      node.Status,
		Priority:    node.Priority,
		IsPrimary:   node.IsPrimary,
		SyncState:   node.SyncState,
	}, nil
}

func (ds *DataSyncService) GetAllNodeStatuses() map[string]*SyncNode {
	ds.mu.RLock()
	defer ds.mu.RUnlock()

	statuses := make(map[string]*SyncNode)
	for nodeID, node := range ds.nodes {
		node.mu.RLock()
		statuses[nodeID] = &SyncNode{
			NodeID:     node.NodeID,
			Address:    node.Address,
			LastSync:   node.LastSync,
			Version:    node.Version,
			Status:     node.Status,
			Priority:   node.Priority,
			IsPrimary:  node.IsPrimary,
			SyncState:  node.SyncState,
		}
		node.mu.RUnlock()
	}
	return statuses
}
