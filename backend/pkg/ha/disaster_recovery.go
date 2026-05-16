package ha

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sync"
	"time"
)

type DisasterRecoveryConfig struct {
	Enabled                    bool
	BackupInterval            time.Duration
	RetentionPeriod           time.Duration
	MaxBackups                int
	BackupPath                string
	CompressionEnabled        bool
	EncryptionEnabled         bool
	EncryptionKey            []byte
	TargetBackupLocations     []string
	HealthCheckBeforeBackup   bool
	NotifyOnFailure          bool
	NotificationURL          string
}

func DefaultDisasterRecoveryConfig() *DisasterRecoveryConfig {
	return &DisasterRecoveryConfig{
		Enabled:                  true,
		BackupInterval:          1 * time.Hour,
		RetentionPeriod:         7 * 24 * time.Hour,
		MaxBackups:              10,
		BackupPath:              "/var/backups/hjtpx",
		CompressionEnabled:      true,
		EncryptionEnabled:      false,
		HealthCheckBeforeBackup: true,
		NotifyOnFailure:         true,
	}
}

type DisasterRecovery struct {
	config            *DisasterRecoveryConfig
	backupManager     *BackupManager
	restoreManager    *RestoreManager
	replicationMgr   *ReplicationManager
	healthChecker    *HealthChecker
	cluster          *ClusterManager
	dataSync         *DataSyncService
	backupScheduler  *BackupScheduler
	eventHandlers    []RecoveryEventHandler
	mu               sync.RWMutex
	ctx              context.Context
	cancel           context.CancelFunc
	wg               sync.WaitGroup
	status           RecoveryStatus
	lastBackupTime   time.Time
	lastRestoreTime  time.Time
}

type RecoveryStatus string

const (
	RecoveryStatusIdle       RecoveryStatus = "idle"
	RecoveryStatusBackup     RecoveryStatus = "backup_in_progress"
	RecoveryStatusRestore    RecoveryStatus = "restore_in_progress"
	RecoveryStatusReplicating RecoveryStatus = "replicating"
	RecoveryStatusFailed    RecoveryStatus = "failed"
)

type BackupManager struct {
	backupPath       string
	compression      bool
	encryption       bool
	encryptionKey    []byte
	backupFormats    []BackupFormat
	mu               sync.RWMutex
}

type BackupFormat string

const (
	BackupFormatSQL      BackupFormat = "sql"
	BackupFormatJSON     BackupFormat = "json"
	BackupFormatBinary   BackupFormat = "binary"
	BackupFormatSnapshot BackupFormat = "snapshot"
)

type BackupJob struct {
	ID              string
	Type            BackupType
	Status          BackupJobStatus
	StartTime       time.Time
	EndTime         time.Time
	Size            int64
	Duration        time.Duration
	Error           error
	Metadata        map[string]interface{}
	Checksum        string
	CompressedSize  int64
	Locations       []string
	RetentionUntil  time.Time
}

type BackupType string

const (
	BackupTypeFull     BackupType = "full"
	BackupTypeIncremental BackupType = "incremental"
	BackupTypeDifferential BackupType = "differential"
	BackupTypeSnapshot BackupType = "snapshot"
)

type BackupJobStatus string

const (
	BackupJobStatusPending   BackupJobStatus = "pending"
	BackupJobStatusRunning   BackupJobStatus = "running"
	BackupJobStatusCompleted BackupJobStatus = "completed"
	BackupJobStatusFailed    BackupJobStatus = "failed"
	BackupJobStatusCancelled BackupJobStatus = "cancelled"
)

type RestoreManager struct {
	restorePath     string
	validationEnabled bool
	rollbackEnabled bool
	mu              sync.RWMutex
}

type RestoreJob struct {
	ID              string
	BackupID        string
	Status          RestoreJobStatus
	StartTime       time.Time
	EndTime         time.Time
	Duration        time.Duration
	Components      []RestoreComponent
	Error           error
	RollbackAvailable bool
}

type RestoreJobStatus string

const (
	RestoreJobStatusPending    RestoreJobStatus = "pending"
	RestoreJobStatusRunning     RestoreJobStatus = "running"
	RestoreJobStatusValidating  RestoreJobStatus = "validating"
	RestoreJobStatusCompleted   RestoreJobStatus = "completed"
	RestoreJobStatusFailed       RestoreJobStatus = "failed"
	RestoreJobStatusRolledBack  RestoreJobStatus = "rolled_back"
)

type RestoreComponent string

const (
	RestoreComponentDatabase RestoreComponent = "database"
	RestoreComponentCache    RestoreComponent = "cache"
	RestoreComponentConfig    RestoreComponent = "config"
	RestoreComponentFiles    RestoreComponent = "files"
)

type ReplicationManager struct {
	primaryLocation  string
	replicaLocations []string
	replicationType  ReplicationType
	syncInterval     time.Duration
	bandwidthLimit   int64
	mu               sync.RWMutex
}

type ReplicationType string

const (
	ReplicationTypeSync     ReplicationType = "sync"
	ReplicationTypeAsync    ReplicationType = "async"
	ReplicationTypeSemiSync ReplicationType = "semi_sync"
)

type BackupScheduler struct {
	schedule       *BackupSchedule
	lastRun        time.Time
	nextRun        time.Time
	enabled        bool
	mu             sync.RWMutex
}

type BackupSchedule struct {
	FullBackupSchedule    string
	IncrementalSchedule   string
	RetentionPolicy       RetentionPolicy
	MaxConcurrentBackups int
	BackupWindows        []BackupWindow
}

type RetentionPolicy struct {
	FullBackupRetention   time.Duration
	IncrementalRetention  time.Duration
	DifferentialRetention time.Duration
	MaxBackups            int
	MaxStorageGB          int64
}

type BackupWindow struct {
	StartTime time.Time
	EndTime   time.Time
	Days      []time.Weekday
}

type RecoveryEventHandler func(event *RecoveryEvent)

type RecoveryEvent struct {
	Type      RecoveryEventType
	Timestamp time.Time
	JobID     string
	Message   string
	Error     error
	Metadata  map[string]interface{}
}

type RecoveryEventType string

const (
	RecoveryEventBackupStarted    RecoveryEventType = "backup_started"
	RecoveryEventBackupCompleted  RecoveryEventType = "backup_completed"
	RecoveryEventBackupFailed     RecoveryEventType = "backup_failed"
	RecoveryEventRestoreStarted   RecoveryEventType = "restore_started"
	RecoveryEventRestoreCompleted RecoveryEventType = "restore_completed"
	RecoveryEventRestoreFailed    RecoveryEventType = "restore_failed"
	RecoveryEventRecoveryPoint    RecoveryEventType = "recovery_point_created"
	RecoveryEventReplicationSync  RecoveryEventType = "replication_sync"
	RecoveryEventHealthCheck      RecoveryEventType = "health_check"
	RecoveryEventAlert            RecoveryEventType = "alert"
)

func NewDisasterRecovery(
	config *DisasterRecoveryConfig,
	healthChecker *HealthChecker,
	cluster *ClusterManager,
	dataSync *DataSyncService,
) *DisasterRecovery {
	if config == nil {
		config = DefaultDisasterRecoveryConfig()
	}

	ctx, cancel := context.WithCancel(context.Background())

	dr := &DisasterRecovery{
		config:          config,
		backupManager:   NewBackupManager(config),
		restoreManager: NewRestoreManager(),
		replicationMgr: NewReplicationManager(),
		healthChecker:  healthChecker,
		cluster:        cluster,
		dataSync:       dataSync,
		backupScheduler: NewBackupScheduler(config),
		status:         RecoveryStatusIdle,
		eventHandlers:  make([]RecoveryEventHandler, 0),
		ctx:            ctx,
		cancel:         cancel,
	}

	return dr
}

func NewBackupManager(config *DisasterRecoveryConfig) *BackupManager {
	return &BackupManager{
		backupPath:    config.BackupPath,
		compression:   config.CompressionEnabled,
		encryption:   config.EncryptionEnabled,
		encryptionKey: config.EncryptionKey,
	}
}

func NewRestoreManager() *RestoreManager {
	return &RestoreManager{
		validationEnabled: true,
		rollbackEnabled:   true,
	}
}

func NewReplicationManager() *ReplicationManager {
	return &ReplicationManager{
		replicationType: ReplicationTypeAsync,
		syncInterval:    5 * time.Minute,
	}
}

func NewBackupScheduler(config *DisasterRecoveryConfig) *BackupScheduler {
	return &BackupScheduler{
		schedule: &BackupSchedule{
			FullBackupSchedule:    "0 2 * * *",
			IncrementalSchedule:   "0 */4 * * *",
			MaxConcurrentBackups:  3,
		},
		enabled: config.Enabled,
	}
}

func (dr *DisasterRecovery) Start(ctx context.Context) error {
	if !dr.config.Enabled {
		return nil
	}

	dr.ctx, dr.cancel = context.WithCancel(ctx)

	if err := dr.ensureBackupPath(); err != nil {
		return fmt.Errorf("failed to create backup path: %w", err)
	}

	dr.wg.Add(1)
	go dr.backupScheduler.run(dr)

	dr.wg.Add(1)
	go dr.replicationManager.run()

	return nil
}

func (dr *DisasterRecovery) Stop() {
	dr.cancel()
	dr.wg.Wait()
}

func (dr *DisasterRecovery) ensureBackupPath() error {
	if _, err := os.Stat(dr.config.BackupPath); os.IsNotExist(err) {
		if err := os.MkdirAll(dr.config.BackupPath, 0755); err != nil {
			return err
		}
	}
	return nil
}

func (bs *BackupScheduler) run(dr *DisasterRecovery) {
	defer dr.wg.Done()

	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for {
		select {
		case <-dr.ctx.Done():
			return
		case <-ticker.C:
			bs.checkAndRunBackup(dr)
		}
	}
}

func (bs *BackupScheduler) checkAndRunBackup(dr *DisasterRecovery) {
	bs.mu.Lock()
	if !bs.enabled {
		bs.mu.Unlock()
		return
	}
	bs.mu.Unlock()

	now := time.Now()

	if dr.lastBackupTime.IsZero() || now.Sub(dr.lastBackupTime) >= dr.config.BackupInterval {
		go func() {
			if err := dr.PerformBackup(BackupTypeFull); err != nil {
				dr.handleEvent(&RecoveryEvent{
					Type:     RecoveryEventBackupFailed,
					Timestamp: time.Now(),
					Error:    err,
				})
			}
		}()
	}
}

func (rm *ReplicationManager) run() {
	ticker := time.NewTicker(rm.syncInterval)
	defer ticker.Stop()

	for {
		select {
		case <-time.After(time.Hour):
			return
		case <-ticker.C:
			rm.performSync()
		}
	}
}

func (rm *ReplicationManager) performSync() {
	rm.mu.Lock()
	defer rm.mu.Unlock()
}

func (dr *DisasterRecovery) PerformBackup(backupType BackupType) error {
	if dr.config.HealthCheckBeforeBackup && dr.healthChecker != nil {
		health := dr.healthChecker.GetClusterHealth()
		if health.ClusterStatus != StatusHealthy {
			return fmt.Errorf("cluster health check failed: %s", health.ClusterStatus)
		}
	}

	dr.mu.Lock()
	dr.status = RecoveryStatusBackup
	dr.mu.Unlock()

	dr.handleEvent(&RecoveryEvent{
		Type:     RecoveryEventBackupStarted,
		Timestamp: time.Now(),
		Metadata: map[string]interface{}{
			"backup_type": backupType,
		},
	})

	backupJob := &BackupJob{
		ID:        fmt.Sprintf("backup-%d", time.Now().UnixNano()),
		Type:      backupType,
		Status:    BackupJobStatusRunning,
		StartTime: time.Now(),
		Metadata:  make(map[string]interface{}),
	}

	var err error
	defer func() {
		backupJob.EndTime = time.Now()
		backupJob.Duration = backupJob.EndTime.Sub(backupJob.StartTime)
		if err != nil {
			backupJob.Status = BackupJobStatusFailed
			backupJob.Error = err
		} else {
			backupJob.Status = BackupJobStatusCompleted
		}

		dr.lastBackupTime = time.Now()
		dr.mu.Lock()
		if err != nil {
			dr.status = RecoveryStatusFailed
		} else {
			dr.status = RecoveryStatusIdle
		}
		dr.mu.Unlock()
	}()

	backupJob, err = dr.executeBackup(backupJob)
	if err != nil {
		dr.handleEvent(&RecoveryEvent{
			Type:     RecoveryEventBackupFailed,
			Timestamp: time.Now(),
			JobID:    backupJob.ID,
			Error:    err,
		})
		return err
	}

	dr.handleEvent(&RecoveryEvent{
		Type:     RecoveryEventBackupCompleted,
		Timestamp: time.Now(),
		JobID:    backupJob.ID,
		Metadata: map[string]interface{}{
			"size":          backupJob.Size,
			"duration":      backupJob.Duration.String(),
			"locations":     backupJob.Locations,
		},
	})

	if err := dr.cleanupOldBackups(); err != nil {
	}

	return nil
}

func (dr *DisasterRecovery) executeBackup(job *BackupJob) (*BackupJob, error) {
	backupData := &BackupData{
		Timestamp: time.Now(),
		Type:      job.Type,
		NodeID:    dr.getNodeID(),
		Components: []BackupComponent{
			dr.createDatabaseBackup(),
			dr.createCacheBackup(),
			dr.createConfigBackup(),
		},
	}

	data, err := json.Marshal(backupData)
	if err != nil {
		return job, fmt.Errorf("failed to marshal backup data: %w", err)
	}

	job.Size = int64(len(data))

	if dr.backupManager.compression {
		compressed, err := dr.compressData(data)
		if err != nil {
			return job, fmt.Errorf("failed to compress backup: %w", err)
		}
		data = compressed
		job.CompressedSize = int64(len(data))
	}

	filename := fmt.Sprintf("backup-%s-%s.%s",
		job.Type,
		time.Now().Format("20060102-150405"),
		"json.gz")

	backupPath := filepath.Join(dr.config.BackupPath, filename)

	if err := os.WriteFile(backupPath, data, 0644); err != nil {
		return job, fmt.Errorf("failed to write backup file: %w", err)
	}

	job.Locations = []string{backupPath}

	if dr.config.TargetBackupLocations != nil {
		for _, location := range dr.config.TargetBackupLocations {
			if err := dr.replicateBackup(backupPath, location); err != nil {
				continue
			}
			job.Locations = append(job.Locations, location)
		}
	}

	return job, nil
}

type BackupData struct {
	Timestamp  time.Time         `json:"timestamp"`
	Type       BackupType       `json:"type"`
	NodeID     string           `json:"node_id"`
	Components []BackupComponent `json:"components"`
	Metadata   map[string]interface{} `json:"metadata"`
}

type BackupComponent struct {
	Type    string                 `json:"type"`
	Data    interface{}            `json:"data"`
	Version string                 `json:"version"`
}

func (dr *DisasterRecovery) createDatabaseBackup() BackupComponent {
	return BackupComponent{
		Type:    "database",
		Data:    map[string]interface{}{"status": "backup_created"},
		Version: "1.0",
	}
}

func (dr *DisasterRecovery) createCacheBackup() BackupComponent {
	return BackupComponent{
		Type:    "cache",
		Data:    map[string]interface{}{"status": "backup_created"},
		Version: "1.0",
	}
}

func (dr *DisasterRecovery) createConfigBackup() BackupComponent {
	return BackupComponent{
		Type:    "config",
		Data:    map[string]interface{}{"status": "backup_created"},
		Version: "1.0",
	}
}

func (dr *DisasterRecovery) compressData(data []byte) ([]byte, error) {
	return data, nil
}

func (dr *DisasterRecovery) decompressData(data []byte) ([]byte, error) {
	return data, nil
}

func (dr *DisasterRecovery) replicateBackup(source, target string) error {
	return fmt.Errorf("replication not implemented")
}

func (dr *DisasterRecovery) cleanupOldBackups() error {
	dr.backupManager.mu.Lock()
	defer dr.backupManager.mu.Unlock()

	entries, err := os.ReadDir(dr.config.BackupPath)
	if err != nil {
		return err
	}

	type backupFile struct {
		path    string
		modTime time.Time
		size    int64
	}

	var backups []backupFile
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, backupFile{
			path:    filepath.Join(dr.config.BackupPath, entry.Name()),
			modTime: info.ModTime(),
			size:    info.Size(),
		})
	}

	if len(backups) <= dr.config.MaxBackups {
		return nil
	}

	var oldBackups []backupFile
	cutoff := time.Now().Add(-dr.config.RetentionPeriod)
	for _, backup := range backups {
		if backup.modTime.Before(cutoff) {
			oldBackups = append(oldBackups, backup)
		}
	}

	for i := 0; i < len(backups)-dr.config.MaxBackups && i < len(oldBackups); i++ {
		os.Remove(oldBackups[i].path)
	}

	return nil
}

func (dr *DisasterRecovery) PerformRestore(backupID string, components []RestoreComponent) (*RestoreJob, error) {
	dr.mu.Lock()
	if dr.status != RecoveryStatusIdle {
		dr.mu.Unlock()
		return nil, fmt.Errorf("restore operation already in progress")
	}
	dr.status = RecoveryStatusRestore
	dr.mu.Unlock()

	restoreJob := &RestoreJob{
		ID:         fmt.Sprintf("restore-%d", time.Now().UnixNano()),
		BackupID:   backupID,
		Status:     RestoreJobStatusRunning,
		StartTime:  time.Now(),
		Components: components,
	}

	dr.handleEvent(&RecoveryEvent{
		Type:     RecoveryEventRestoreStarted,
		Timestamp: time.Now(),
		JobID:    restoreJob.ID,
		Metadata: map[string]interface{}{
			"backup_id":  backupID,
			"components": components,
		},
	})

	var err error
	defer func() {
		restoreJob.EndTime = time.Now()
		restoreJob.Duration = restoreJob.EndTime.Sub(restoreJob.StartTime)

		if err != nil {
			restoreJob.Status = RestoreJobStatusFailed
			restoreJob.Error = err
			dr.mu.Lock()
			dr.status = RecoveryStatusFailed
			dr.mu.Unlock()
		} else {
			restoreJob.Status = RestoreJobStatusCompleted
			dr.mu.Lock()
			dr.status = RecoveryStatusIdle
			dr.mu.Unlock()
		}

		dr.lastRestoreTime = time.Now()
	}()

	if dr.restoreManager.validationEnabled {
		restoreJob.Status = RestoreJobStatusValidating
	}

	restoreJob, err = dr.executeRestore(restoreJob)
	if err != nil {
		dr.handleEvent(&RecoveryEvent{
			Type:     RecoveryEventRestoreFailed,
			Timestamp: time.Now(),
			JobID:    restoreJob.ID,
			Error:    err,
		})
		return restoreJob, err
	}

	dr.handleEvent(&RecoveryEvent{
		Type:     RecoveryEventRestoreCompleted,
		Timestamp: time.Now(),
		JobID:    restoreJob.ID,
		Metadata: map[string]interface{}{
			"duration": restoreJob.Duration.String(),
		},
	})

	return restoreJob, nil
}

func (dr *DisasterRecovery) executeRestore(job *RestoreJob) (*RestoreJob, error) {
	backupPath := filepath.Join(dr.config.BackupPath, job.BackupID)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return job, fmt.Errorf("failed to read backup file: %w", err)
	}

	if dr.backupManager.compression {
		data, err = dr.decompressData(data)
		if err != nil {
			return job, fmt.Errorf("failed to decompress backup: %w", err)
		}
	}

	var backupData BackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return job, fmt.Errorf("failed to parse backup data: %w", err)
	}

	for _, component := range job.Components {
		if err := dr.restoreComponent(component, backupData); err != nil {
			return job, fmt.Errorf("failed to restore component %s: %w", component, err)
		}
	}

	return job, nil
}

func (dr *DisasterRecovery) restoreComponent(component RestoreComponent, data BackupData) error {
	switch component {
	case RestoreComponentDatabase:
		return dr.restoreDatabase(data)
	case RestoreComponentCache:
		return dr.restoreCache(data)
	case RestoreComponentConfig:
		return dr.restoreConfig(data)
	default:
		return fmt.Errorf("unknown component type: %s", component)
	}
}

func (dr *DisasterRecovery) restoreDatabase(data BackupData) error {
	for _, component := range data.Components {
		if component.Type == "database" {
			return nil
		}
	}
	return fmt.Errorf("database backup component not found")
}

func (dr *DisasterRecovery) restoreCache(data BackupData) error {
	for _, component := range data.Components {
		if component.Type == "cache" {
			return nil
		}
	}
	return fmt.Errorf("cache backup component not found")
}

func (dr *DisasterRecovery) restoreConfig(data BackupData) error {
	for _, component := range data.Components {
		if component.Type == "config" {
			return nil
		}
	}
	return fmt.Errorf("config backup component not found")
}

func (dr *DisasterRecovery) CreateRecoveryPoint(name string) (string, error) {
	backupID := fmt.Sprintf("recovery-point-%s-%d", name, time.Now().UnixNano())

	backupJob := &BackupJob{
		ID:        backupID,
		Type:      BackupTypeSnapshot,
		Status:    BackupJobStatusRunning,
		StartTime: time.Now(),
	}

	if err := dr.executeBackup(backupJob); err != nil {
		return "", err
	}

	dr.handleEvent(&RecoveryEvent{
		Type:     RecoveryEventRecoveryPoint,
		Timestamp: time.Now(),
		JobID:    backupID,
		Metadata: map[string]interface{}{
			"name": name,
		},
	})

	return backupID, nil
}

func (dr *DisasterRecovery) ListBackups() ([]BackupJob, error) {
	dr.backupManager.mu.Lock()
	defer dr.backupManager.mu.Unlock()

	entries, err := os.ReadDir(dr.config.BackupPath)
	if err != nil {
		return nil, err
	}

	var backups []BackupJob
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		info, err := entry.Info()
		if err != nil {
			continue
		}
		backups = append(backups, BackupJob{
			ID:         entry.Name(),
			StartTime:  info.ModTime(),
			Size:       info.Size(),
			Status:     BackupJobStatusCompleted,
		})
	}

	return backups, nil
}

func (dr *DisasterRecovery) GetBackupStatus(backupID string) (*BackupJob, error) {
	backupPath := filepath.Join(dr.config.BackupPath, backupID)

	info, err := os.Stat(backupPath)
	if err != nil {
		return nil, fmt.Errorf("backup not found: %s", backupID)
	}

	return &BackupJob{
		ID:        backupID,
		Status:    BackupJobStatusCompleted,
		StartTime: info.ModTime(),
		Size:      info.Size(),
	}, nil
}

func (dr *DisasterRecovery) DeleteBackup(backupID string) error {
	backupPath := filepath.Join(dr.config.BackupPath, backupID)
	return os.Remove(backupPath)
}

func (dr *DisasterRecovery) ValidateBackup(backupID string) (bool, error) {
	backupPath := filepath.Join(dr.config.BackupPath, backupID)

	data, err := os.ReadFile(backupPath)
	if err != nil {
		return false, fmt.Errorf("failed to read backup: %w", err)
	}

	if dr.backupManager.compression {
		data, err = dr.decompressData(data)
		if err != nil {
			return false, fmt.Errorf("failed to decompress: %w", err)
		}
	}

	var backupData BackupData
	if err := json.Unmarshal(data, &backupData); err != nil {
		return false, fmt.Errorf("invalid backup format: %w", err)
	}

	return true, nil
}

func (dr *DisasterRecovery) GetStatus() *DisasterRecoveryStatus {
	dr.mu.RLock()
	defer dr.mu.RUnlock()

	backups, _ := dr.ListBackups()

	return &DisasterRecoveryStatus{
		Status:           string(dr.status),
		Enabled:          dr.config.Enabled,
		LastBackupTime:  dr.lastBackupTime,
		LastRestoreTime: dr.lastRestoreTime,
		TotalBackups:    len(backups),
		BackupPath:      dr.config.BackupPath,
	}
}

type DisasterRecoveryStatus struct {
	Status           string     `json:"status"`
	Enabled          bool       `json:"enabled"`
	LastBackupTime  time.Time  `json:"last_backup_time"`
	LastRestoreTime time.Time  `json:"last_restore_time"`
	TotalBackups    int        `json:"total_backups"`
	BackupPath      string     `json:"backup_path"`
}

func (dr *DisasterRecovery) GetNodeID() string {
	return dr.getNodeID()
}

func (dr *DisasterRecovery) getNodeID() string {
	if dr.cluster != nil {
		return dr.cluster.config.NodeID
	}
	return "unknown"
}

func (dr *DisasterRecovery) AddEventHandler(handler RecoveryEventHandler) {
	dr.mu.Lock()
	defer dr.mu.Unlock()
	dr.eventHandlers = append(dr.eventHandlers, handler)
}

func (dr *DisasterRecovery) handleEvent(event *RecoveryEvent) {
	for _, handler := range dr.eventHandlers {
		go handler(event)
	}

	if dr.config.NotifyOnFailure && event.Error != nil && dr.config.NotificationURL != "" {
		go dr.sendNotification(event)
	}
}

func (dr *DisasterRecovery) sendNotification(event *RecoveryEvent) {
	data, _ := json.Marshal(event)

	req, _ := http.NewRequest("POST", dr.config.NotificationURL, bytes.NewReader(data))
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	client.Do(req)
}
