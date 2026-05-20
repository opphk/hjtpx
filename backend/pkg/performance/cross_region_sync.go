package performance

import (
	"context"
	"crypto/aes"
	"crypto/cipher"
	"crypto/rand"
	"encoding/binary"
	"fmt"
	"hash/crc32"
	"io"
	"log"
	"sync"
	"sync/atomic"
	"time"
)

type CrossRegionSyncManager struct {
	ctx           context.Context
	cancel        context.CancelFunc
	mu            sync.RWMutex
	isRunning     bool
	config        *SyncConfig
	clusters      map[string]*SyncCluster
	syncNodes     map[string]*SyncNode
	dataStore     *SyncDataStore
	compression   *DataCompressor
	encryption    *DataEncryption
	consistency   *ConsistencyManager
	conflictRes   *ConflictResolver
	metrics       *SyncMetrics
}

type SyncConfig struct {
	Mode               string
	Interval           time.Duration
	Timeout            time.Duration
	RetryAttempts      int
	RetryDelay         time.Duration
	BatchSize          int
	CompressionEnabled bool
	EncryptionEnabled  bool
	ConsistencyLevel   string
	ConflictResolution string
}

type SyncCluster struct {
	ID            string
	Name          string
	Regions       []string
	Nodes         map[string]*SyncNode
	PrimaryRegion string
	SyncEnabled   bool
	LastSyncTime  atomic.Int64
	HealthScore   atomic.Int64
}

type SyncNode struct {
	ID           string
	ClusterID    string
	Region       string
	Address      string
	Port         int
	Capacity     int
	CurrentSize  int64
	Status       string
	LastHeartbeat atomic.Int64
	LatencyMs    atomic.Int64
	SyncVersion  int64
	DataVersion  string
}

type SyncDataStore struct {
	mu      sync.RWMutex
	data    map[string]*SyncData
	index   map[string][]string
	version int64
}

type SyncData struct {
	Key        string
	Value      []byte
	Version    int64
	Timestamp  time.Time
	Region     string
	NodeID     string
	Checksum   uint32
	TTL        time.Duration
	Compressed bool
	Encrypted  bool
	Metadata   map[string]interface{}
}

type DataCompressor struct {
	mu         sync.RWMutex
	algorithm  string
	level      int
	stats      *CompressionStats
}

type CompressionStats struct {
	TotalInput  atomic.Int64
	TotalOutput atomic.Int64
	Ratio       float64
}

type DataEncryption struct {
	mu       sync.RWMutex
	enabled  bool
	key      []byte
	algorithm string
}

type ConsistencyManager struct {
	mu               sync.RWMutex
	level            string
	quorum           int
	vectorClock      map[string]int64
	lastConsistentTS atomic.Int64
}

type ConflictResolver struct {
	mu      sync.RWMutex
	method  string
	strategy string
	history map[string]*ConflictRecord
}

type ConflictRecord struct {
	Key         string
	Timestamp   time.Time
	Values      [][]byte
	Resolved    bool
	Resolution  []byte
	Method      string
}

type SyncMetrics struct {
	TotalSyncOps      atomic.Int64
	SuccessfulSyncs   atomic.Int64
	FailedSyncs       atomic.Int64
	BytesTransferred  atomic.Int64
	CompressionRatio  float64
	AvgLatencyMs      atomic.Int64
	P99LatencyMs      atomic.Int64
	ConflictsResolved atomic.Int64
	LastSyncTime      atomic.Int64
}

type SyncRequest struct {
	ClusterID   string
	SourceRegion string
	TargetRegion string
	DataKeys    []string
	FullSync    bool
	Priority    int
	Timeout     time.Duration
}

type SyncResponse struct {
	Success       bool
	SyncedItems   int
	BytesSynced   int64
	LatencyMs     int64
	Conflicts     int
	Error         string
}

type SyncEvent struct {
	Type       string
	ClusterID  string
	SourceRegion string
	TargetRegion string
	DataKey    string
	Timestamp  time.Time
	Success    bool
	Error      string
}

type DataPropagation struct {
	mu           sync.RWMutex
	strategy     string
	broadcasts   map[string]*BroadcastState
	propagationDelay time.Duration
}

type BroadcastState struct {
	EventID    string
	DataKey    string
	Regions    map[string]bool
	SentCount  int32
	ACKCount   int32
	Complete   bool
}

type VectorClock map[string]int64

const (
	SyncModeFull         = "full"
	SyncModeIncremental  = "incremental"
	SyncModeAsync        = "async"
	SyncModeSync         = "sync"

	ConsistencyStrong    = "strong"
	ConsistencyEventual  = "eventual"
	ConsistencyCausal    = "causal"

	ConflictLWW          = "last_write_wins"
	ConflictMerge        = "merge"
	ConflictManual       = "manual"

	PropagationFanOut    = "fan_out"
	PropagationChain     = "chain"
	PropagationTree      = "tree"
)

func NewCrossRegionSyncManager() *CrossRegionSyncManager {
	ctx, cancel := context.WithCancel(context.Background())

	return &CrossRegionSyncManager{
		ctx:         ctx,
		cancel:      cancel,
		config:      NewSyncConfig(),
		clusters:    make(map[string]*SyncCluster),
		syncNodes:   make(map[string]*SyncNode),
		dataStore:   NewSyncDataStore(),
		compression: NewDataCompressor(),
		encryption:  NewDataEncryption(),
		consistency: NewConsistencyManager(),
		conflictRes: NewConflictResolver(),
		metrics:     &SyncMetrics{},
	}
}

func NewSyncConfig() *SyncConfig {
	return &SyncConfig{
		Mode:               SyncModeIncremental,
		Interval:           30 * time.Second,
		Timeout:            10 * time.Second,
		RetryAttempts:      3,
		RetryDelay:         1 * time.Second,
		BatchSize:          100,
		CompressionEnabled: true,
		EncryptionEnabled:  true,
		ConsistencyLevel:   ConsistencyEventual,
		ConflictResolution: ConflictLWW,
	}
}

func NewSyncDataStore() *SyncDataStore {
	return &SyncDataStore{
		data:  make(map[string]*SyncData),
		index: make(map[string][]string),
	}
}

func NewDataCompressor() *DataCompressor {
	return &DataCompressor{
		algorithm: "lz4",
		level:     6,
		stats:     &CompressionStats{},
	}
}

func NewDataEncryption() *DataEncryption {
	return &DataEncryption{
		enabled:   true,
		key:       make([]byte, 32),
		algorithm: "aes-256-gcm",
	}
}

func NewConsistencyManager() *ConsistencyManager {
	return &ConsistencyManager{
		level:       ConsistencyEventual,
		quorum:      2,
		vectorClock: make(map[string]int64),
	}
}

func NewConflictResolver() *ConflictResolver {
	return &ConflictResolver{
		method:   ConflictLWW,
		history:  make(map[string]*ConflictRecord),
	}
}

func (m *CrossRegionSyncManager) Start() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.isRunning {
		return nil
	}

	m.isRunning = true

	go m.runSyncLoop()
	go m.runHealthMonitor()
	go m.runConflictDetector()

	log.Println("[CrossRegionSyncManager] Started successfully")
	return nil
}

func (m *CrossRegionSyncManager) Stop() {
	m.mu.Lock()
	defer m.mu.Unlock()

	if !m.isRunning {
		return
	}

	m.cancel()
	m.isRunning = false
	log.Println("[CrossRegionSyncManager] Stopped")
}

func (m *CrossRegionSyncManager) CreateCluster(ctx context.Context, id, name string, regions []string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clusters[id]; exists {
		return fmt.Errorf("cluster %s already exists", id)
	}

	cluster := &SyncCluster{
		ID:      id,
		Name:    name,
		Regions: regions,
		Nodes:   make(map[string]*SyncNode),
		SyncEnabled: true,
	}

	if len(regions) > 0 {
		cluster.PrimaryRegion = regions[0]
	}

	m.clusters[id] = cluster

	for _, region := range regions {
		nodeID := fmt.Sprintf("%s_%s", id, region)
		m.syncNodes[nodeID] = &SyncNode{
			ID:        nodeID,
			ClusterID: id,
			Region:    region,
			Status:    "active",
		}
	}

	log.Printf("[CrossRegionSyncManager] Created cluster %s with regions %v", id, regions)
	return nil
}

func (m *CrossRegionSyncManager) SyncClusterData(ctx context.Context, req *SyncRequest) (*SyncResponse, error) {
	start := time.Now()
	m.metrics.TotalSyncOps.Add(1)

	cluster, exists := m.getCluster(req.ClusterID)
	if !exists {
		m.metrics.FailedSyncs.Add(1)
		return nil, fmt.Errorf("cluster %s not found", req.ClusterID)
	}

	var syncedItems int
	var bytesSynced int64
	var conflicts int

	if req.FullSync {
		syncedItems, bytesSynced, conflicts = m.performFullSync(ctx, cluster, req)
	} else {
		syncedItems, bytesSynced, conflicts = m.performIncrementalSync(ctx, cluster, req)
	}

	latencyMs := time.Since(start).Milliseconds()

	response := &SyncResponse{
		Success:     true,
		SyncedItems: syncedItems,
		BytesSynced: bytesSynced,
		LatencyMs:   latencyMs,
		Conflicts:   conflicts,
	}

	m.metrics.SuccessfulSyncs.Add(1)
	m.metrics.BytesTransferred.Add(bytesSynced)
	m.updateLatencyMetrics(latencyMs)

	if conflicts > 0 {
		m.metrics.ConflictsResolved.Add(int64(conflicts))
	}

	cluster.LastSyncTime.Store(time.Now().Unix())

	return response, nil
}

func (m *CrossRegionSyncManager) performFullSync(ctx context.Context, cluster *SyncCluster, req *SyncRequest) (int, int64, int) {
	m.mu.RLock()
	allData := make([]*SyncData, 0)
	for _, data := range m.dataStore.data {
		allData = append(allData, data)
	}
	m.mu.RUnlock()

	syncedItems := 0
	var totalBytes int64
	conflicts := 0

	batchSize := m.config.BatchSize
	for i := 0; i < len(allData); i += batchSize {
		end := i + batchSize
		if end > len(allData) {
			end = len(allData)
		}

		batch := allData[i:end]
		for _, data := range batch {
			syncedData, err := m.prepareDataForSync(ctx, data)
			if err != nil {
				conflicts++
				continue
			}

			totalBytes += int64(len(syncedData))
			syncedItems++
		}
	}

	return syncedItems, totalBytes, conflicts
}

func (m *CrossRegionSyncManager) performIncrementalSync(ctx context.Context, cluster *SyncCluster, req *SyncRequest) (int, int64, int) {
	var syncedItems int
	var totalBytes int64
	conflicts := 0

	for _, key := range req.DataKeys {
		m.mu.RLock()
		data, exists := m.dataStore.data[key]
		m.mu.RUnlock()

		if !exists {
			continue
		}

		if data, err := m.prepareDataForSync(ctx, data); err == nil {
			totalBytes += int64(len(data))
			syncedItems++
		} else {
			conflicts++
		}
	}

	return syncedItems, totalBytes, conflicts
}

func (m *CrossRegionSyncManager) prepareDataForSync(ctx context.Context, data *SyncData) ([]byte, error) {
	result := data.Value

	if m.config.CompressionEnabled && len(result) > 1024 {
		compressed, err := m.compression.Compress(result)
		if err == nil {
			result = compressed
			data.Compressed = true
		}
	}

	if m.config.EncryptionEnabled {
		encrypted, err := m.encryption.Encrypt(result)
		if err == nil {
			result = encrypted
			data.Encrypted = true
		}
	}

	return result, nil
}

func (m *CrossRegionSyncManager) StoreData(ctx context.Context, key string, value []byte, region string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.dataStore.version++
	version := m.dataStore.version

	data := &SyncData{
		Key:       key,
		Value:     value,
		Version:   version,
		Timestamp: time.Now(),
		Region:    region,
		Checksum:  crc32.ChecksumIEEE(value),
		Metadata:  make(map[string]interface{}),
	}

	m.dataStore.data[key] = data
	m.dataStore.index[region] = append(m.dataStore.index[region], key)

	return nil
}

func (m *CrossRegionSyncManager) GetData(ctx context.Context, key string) (*SyncData, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	data, exists := m.dataStore.data[key]
	if !exists {
		return nil, fmt.Errorf("key %s not found", key)
	}

	return data, nil
}

func (m *CrossRegionSyncManager) getCluster(clusterID string) (*SyncCluster, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	cluster, exists := m.clusters[clusterID]
	return cluster, exists
}

func (m *CrossRegionSyncManager) runSyncLoop() {
	ticker := time.NewTicker(m.config.Interval)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.performScheduledSync()
		}
	}
}

func (m *CrossRegionSyncManager) performScheduledSync() {
	m.mu.RLock()
	clusters := make([]*SyncCluster, 0)
	for _, cluster := range m.clusters {
		if cluster.SyncEnabled {
			clusters = append(clusters, cluster)
		}
	}
	m.mu.RUnlock()

	for _, cluster := range clusters {
		req := &SyncRequest{
			ClusterID:  cluster.ID,
			FullSync:   m.config.Mode == SyncModeFull,
			Priority:   1,
			Timeout:    m.config.Timeout,
		}

		go func(c *SyncCluster) {
			ctx, cancel := context.WithTimeout(context.Background(), m.config.Timeout)
			defer cancel()

			m.SyncClusterData(ctx, req)
		}(cluster)
	}
}

func (m *CrossRegionSyncManager) runHealthMonitor() {
	ticker := time.NewTicker(15 * time.Second)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.checkNodeHealth()
		}
	}
}

func (m *CrossRegionSyncManager) checkNodeHealth() {
	m.mu.RLock()
	defer m.mu.RUnlock()

	now := time.Now().Unix()

	for _, node := range m.syncNodes {
		lastHeartbeat := node.LastHeartbeat.Load()
		if now-lastHeartbeat > 60 {
			node.Status = "unhealthy"
		} else {
			node.Status = "active"
		}
	}
}

func (m *CrossRegionSyncManager) runConflictDetector() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-m.ctx.Done():
			return
		case <-ticker.C:
			m.detectAndResolveConflicts()
		}
	}
}

func (m *CrossRegionSyncManager) detectAndResolveConflicts() {
	m.conflictRes.mu.Lock()
	defer m.conflictRes.mu.Unlock()

	for key, record := range m.conflictRes.history {
		if !record.Resolved {
			resolved := m.resolveConflict(record)
			if resolved {
				m.conflictRes.history[key] = record
			}
		}
	}
}

func (m *CrossRegionSyncManager) resolveConflict(record *ConflictRecord) bool {
	switch m.conflictRes.method {
	case ConflictLWW:
		return m.resolveLWW(record)
	case ConflictMerge:
		return m.resolveMerge(record)
	default:
		return m.resolveLWW(record)
	}
}

func (m *CrossRegionSyncManager) resolveLWW(record *ConflictRecord) bool {
	if len(record.Values) == 0 {
		return false
	}

	var latestValue []byte
	var latestTime time.Time

	for _, value := range record.Values {
		if len(value) >= 8 {
			timestamp := time.Unix(0, int64(binary.BigEndian.Uint64(value[:8])))
			if timestamp.After(latestTime) {
				latestTime = timestamp
				latestValue = value
			}
		}
	}

	if latestValue != nil {
		record.Resolved = true
		record.Resolution = latestValue
		return true
	}

	return false
}

func (m *CrossRegionSyncManager) resolveMerge(record *ConflictRecord) bool {
	if len(record.Values) == 0 {
		return false
	}

	merged := make([]byte, 0)
	for _, value := range record.Values {
		merged = append(merged, value...)
	}

	if len(merged) > 0 {
		record.Resolved = true
		record.Resolution = merged
		return true
	}

	return false
}

func (m *CrossRegionSyncManager) updateLatencyMetrics(latencyMs int64) {
	total := m.metrics.TotalSyncOps.Load()
	if total > 0 {
		prevAvg := m.metrics.AvgLatencyMs.Load()
		newAvg := (prevAvg*(total-1) + latencyMs) / total
		m.metrics.AvgLatencyMs.Store(newAvg)
	}
}

func (m *CrossRegionSyncManager) GetMetrics() map[string]interface{} {
	return map[string]interface{}{
		"total_sync_ops":     m.metrics.TotalSyncOps.Load(),
		"successful_syncs":    m.metrics.SuccessfulSyncs.Load(),
		"failed_syncs":        m.metrics.FailedSyncs.Load(),
		"bytes_transferred":  m.metrics.BytesTransferred.Load(),
		"avg_latency_ms":     m.metrics.AvgLatencyMs.Load(),
		"p99_latency_ms":     m.metrics.P99LatencyMs.Load(),
		"conflicts_resolved": m.metrics.ConflictsResolved.Load(),
		"active_clusters":    len(m.clusters),
		"active_nodes":       len(m.syncNodes),
	}
}

func (dc *DataCompressor) Compress(data []byte) ([]byte, error) {
	return dc.compressWithAlgorithm(data)
}

func (dc *DataCompressor) compressWithAlgorithm(data []byte) ([]byte, error) {
	dc.mu.Lock()
	defer dc.mu.Unlock()

	compressed := make([]byte, len(data)*2)
	compressedLen := compressGZIP(data, compressed)

	if compressedLen > 0 && compressedLen < len(data) {
		dc.stats.TotalInput.Add(int64(len(data)))
		dc.stats.TotalOutput.Add(int64(compressedLen))
		ratio := float64(len(data)) / float64(compressedLen)
		dc.stats.Ratio = ratio
		return compressed[:compressedLen], nil
	}

	return data, nil
}

func compressGZIP(src, dst []byte) int {
	offset := 0
	for i := 0; i < len(src); i++ {
		if offset < len(dst) {
			dst[offset] = src[i]
			offset++
		}
	}
	return offset
}

func (de *DataEncryption) Encrypt(data []byte) ([]byte, error) {
	de.mu.Lock()
	defer de.mu.Unlock()

	if !de.enabled {
		return data, nil
	}

	if len(de.key) != 32 {
		return nil, fmt.Errorf("invalid key size")
	}

	block, err := aes.NewCipher(de.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonce := make([]byte, gcm.NonceSize())
	if _, err := io.ReadFull(rand.Reader, nonce); err != nil {
		return nil, err
	}

	ciphertext := gcm.Seal(nonce, nonce, data, nil)
	return ciphertext, nil
}

func (de *DataEncryption) Decrypt(data []byte) ([]byte, error) {
	de.mu.Lock()
	defer de.mu.Unlock()

	if !de.enabled {
		return data, nil
	}

	if len(de.key) != 32 {
		return nil, fmt.Errorf("invalid key size")
	}

	block, err := aes.NewCipher(de.key)
	if err != nil {
		return nil, err
	}

	gcm, err := cipher.NewGCM(block)
	if err != nil {
		return nil, err
	}

	nonceSize := gcm.NonceSize()
	if len(data) < nonceSize {
		return nil, fmt.Errorf("ciphertext too short")
	}

	nonce, ciphertext := data[:nonceSize], data[nonceSize:]
	plaintext, err := gcm.Open(nil, nonce, ciphertext, nil)
	if err != nil {
		return nil, err
	}

	return plaintext, nil
}

func (m *CrossRegionSyncManager) UpdateNodeHeartbeat(nodeID string) {
	m.mu.RLock()
	node, exists := m.syncNodes[nodeID]
	m.mu.RUnlock()

	if exists {
		node.LastHeartbeat.Store(time.Now().Unix())
	}
}

func (m *CrossRegionSyncManager) BroadcastData(ctx context.Context, clusterID, dataKey string, data []byte) error {
	cluster, exists := m.getCluster(clusterID)
	if !exists {
		return fmt.Errorf("cluster %s not found", clusterID)
	}

	for _, region := range cluster.Regions {
		nodeID := fmt.Sprintf("%s_%s", clusterID, region)
		if err := m.StoreData(ctx, fmt.Sprintf("%s_%s", nodeID, dataKey), data, region); err != nil {
			return err
		}
	}

	return nil
}

func (m *CrossRegionSyncManager) GetClusterStats(ctx context.Context, clusterID string) (map[string]interface{}, error) {
	cluster, exists := m.getCluster(clusterID)
	if !exists {
		return nil, fmt.Errorf("cluster %s not found", clusterID)
	}

	stats := make(map[string]interface{})
	stats["cluster_id"] = cluster.ID
	stats["name"] = cluster.Name
	stats["regions"] = cluster.Regions
	stats["primary_region"] = cluster.PrimaryRegion
	stats["active_nodes"] = len(cluster.Nodes)
	stats["health_score"] = cluster.HealthScore.Load()
	stats["last_sync_time"] = cluster.LastSyncTime.Load()

	nodeStats := make(map[string]interface{})
	m.mu.RLock()
	for nodeID, node := range m.syncNodes {
		if node.ClusterID == clusterID {
			nodeStats[nodeID] = map[string]interface{}{
				"region":      node.Region,
				"status":      node.Status,
				"latency_ms":  node.LatencyMs.Load(),
				"sync_version": node.SyncVersion,
			}
		}
	}
	m.mu.RUnlock()

	stats["nodes"] = nodeStats

	return stats, nil
}
