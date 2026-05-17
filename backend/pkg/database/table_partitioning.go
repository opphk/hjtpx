package database

import (
	"context"
	"fmt"
	"log"
	"regexp"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	"gorm.io/gorm"
)

type TablePartitioner struct {
	db              *gorm.DB
	config          *config.Config
	timePartitioner *TimePartitioner
	appPartitioner  *AppPartitioner
	enabled         bool
	mu              sync.RWMutex
}

type TimePartitioner struct {
	db           *gorm.DB
	partitionCfg PartitionConfig
	mu           sync.RWMutex
}

type AppPartitioner struct {
	db       *gorm.DB
	mu       sync.RWMutex
	shards   map[string]*gorm.DB
	strategy string
}

type PartitionConfig struct {
	Strategy         string        `json:"strategy"`
	Unit            string        `json:"unit"`
	RetentionDays   int           `json:"retention_days"`
	PreCreateDays   int           `json:"pre_create_days"`
	AutoArchive     bool          `json:"auto_archive"`
	ArchiveThreshold int64        `json:"archive_threshold"`
	PartitionNaming  string        `json:"partition_naming"`
}

type PartitionInfo struct {
	TableName     string    `json:"table_name"`
	PartitionName string    `json:"partition_name"`
	RangeStart    time.Time `json:"range_start"`
	RangeEnd      time.Time `json:"range_end"`
	RowCount      int64     `json:"row_count"`
	SizeBytes     int64     `json:"size_bytes"`
	IsActive      bool      `json:"is_active"`
	IsArchived    bool      `json:"is_archived"`
}

type AppShardInfo struct {
	AppID      string    `json:"app_id"`
	ShardID    int       `json:"shard_id"`
	DBName     string    `json:"db_name"`
	TableCount int64     `json:"table_count"`
	TotalSize  int64     `json:"total_size"`
	IsActive   bool      `json:"is_active"`
	CreatedAt  time.Time `json:"created_at"`
}

type ShardConfig struct {
	TotalShards    int      `json:"total_shards"`
	ShardingKey    string   `json:"sharding_key"`
	DBPrefix       string   `json:"db_prefix"`
	AppIDs         []string `json:"app_ids"`
	AutoCreate     bool     `json:"auto_create"`
	ReplicationFactor int   `json:"replication_factor"`
}

var globalPartitioner *TablePartitioner

func InitTablePartitioner(db *gorm.DB, cfg *config.Config) error {
	globalPartitioner = &TablePartitioner{
		db:    db,
		config: cfg,
		timePartitioner: &TimePartitioner{
			db: db,
			partitionCfg: PartitionConfig{
				Strategy:        "monthly",
				Unit:            "month",
				RetentionDays:   90,
				PreCreateDays:   7,
				AutoArchive:     true,
				ArchiveThreshold: 1000000,
				PartitionNaming: "p_{table}_{date}",
			},
		},
		appPartitioner: &AppPartitioner{
			db:       db,
			shards:   make(map[string]*gorm.DB),
			strategy: "hash",
		},
		enabled: true,
	}

	if globalPartitioner.enabled {
		go globalPartitioner.startPartitionMaintenance()
		log.Println("Table partitioner initialized")
	}

	return nil
}

func GetTablePartitioner() *TablePartitioner {
	return globalPartitioner
}

func (p *TablePartitioner) CreateTimePartition(ctx context.Context, tableName, dateColumn string, startDate, endDate time.Time) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	partitionName := fmt.Sprintf("%s_%s", tableName, startDate.Format("20060102"))
	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, tableName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	if err := p.db.WithContext(ctx).Exec(createSQL).Error; err != nil {
		return fmt.Errorf("failed to create partition: %w", err)
	}

	log.Printf("Created partition %s for table %s", partitionName, tableName)
	return nil
}

func (p *TablePartitioner) CreateRangePartition(ctx context.Context, tableName string, bounds []PartitionBound) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	for i := 0; i < len(bounds)-1; i++ {
		partitionName := fmt.Sprintf("%s_%d", tableName, i)
		createSQL := fmt.Sprintf(`
			CREATE TABLE IF NOT EXISTS %s PARTITION OF %s
			FOR VALUES FROM (%d) TO (%d)
		`, partitionName, tableName, bounds[i].Value, bounds[i+1].Value)

		if err := p.db.WithContext(ctx).Exec(createSQL).Error; err != nil {
			return fmt.Errorf("failed to create range partition: %w", err)
		}

		log.Printf("Created range partition %s", partitionName)
	}

	return nil
}

type PartitionBound struct {
	Value     int64
	BoundType string
}

func (p *TablePartitioner) ListPartitions(ctx context.Context, tableName string) ([]PartitionInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT 
			child.relname AS partition_name,
			pg_size_pretty(pg_relation_size(child.oid)) AS size,
			pg_relation_size(child.oid) AS size_bytes
		FROM pg_inherits
		JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
		JOIN pg_class child ON pg_inherits.inhrelid = child.oid
		WHERE parent.relname = $1
		ORDER BY child.relname
	`

	type partitionRow struct {
		PartitionName string `gorm:"column:partition_name"`
		Size          string `gorm:"column:size"`
		SizeBytes     int64 `gorm:"column:size_bytes"`
	}

	var rows []partitionRow
	if err := p.db.WithContext(ctx).Raw(query, tableName).Scan(&rows).Error; err != nil {
		return nil, fmt.Errorf("failed to list partitions: %w", err)
	}

	partitions := make([]PartitionInfo, 0, len(rows))
	for _, row := range rows {
		partitions = append(partitions, PartitionInfo{
			TableName:     tableName,
			PartitionName: row.PartitionName,
			SizeBytes:     row.SizeBytes,
			IsActive:      true,
		})
	}

	return partitions, nil
}

func (p *TablePartitioner) DropPartition(ctx context.Context, partitionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", partitionName)
	if err := p.db.WithContext(ctx).Exec(dropSQL).Error; err != nil {
		return fmt.Errorf("failed to drop partition: %w", err)
	}

	log.Printf("Dropped partition %s", partitionName)
	return nil
}

func (p *TablePartitioner) ArchivePartition(ctx context.Context, partitionName, archivePath string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	log.Printf("Archiving partition %s to %s", partitionName, archivePath)
	return nil
}

func (p *TablePartitioner) startPartitionMaintenance() {
	ticker := time.NewTicker(1 * time.Hour)
	defer ticker.Stop()

	for range ticker.C {
		p.performPartitionMaintenance()
	}
}

func (p *TablePartitioner) performPartitionMaintenance() {
	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Minute)
	defer cancel()

	if p.timePartitioner != nil {
		p.timePartitioner.autoCreatePartitions(ctx)
		p.timePartitioner.cleanupOldPartitions(ctx)
	}
}

func (t *TimePartitioner) autoCreatePartitions(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()

	tables := []string{"verifications", "verification_logs", "behavior_data"}

	now := time.Now()
	for _, table := range tables {
		for i := 0; i < t.partitionCfg.PreCreateDays; i++ {
			date := now.AddDate(0, 0, i)
			partitionName := fmt.Sprintf("%s_%s", table, date.Format("20060102"))
			startDate := date
			endDate := date.AddDate(0, 0, 1)

			if err := t.createPartitionIfNotExists(ctx, table, partitionName, startDate, endDate); err != nil {
				log.Printf("Failed to create partition %s: %v", partitionName, err)
			}
		}
	}
}

func (t *TimePartitioner) createPartitionIfNotExists(ctx context.Context, tableName, partitionName string, startDate, endDate time.Time) error {
	if t.db == nil {
		return fmt.Errorf("database not available")
	}

	checkSQL := `
		SELECT EXISTS (
			SELECT FROM pg_tables 
			WHERE schemaname = 'public' AND tablename = $1
		)
	`
	var exists bool
	if err := t.db.WithContext(ctx).Raw(checkSQL, partitionName).Scan(&exists).Error; err != nil {
		return err
	}

	if exists {
		return nil
	}

	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			LIKE %s INCLUDING ALL
		) PARTITION OF %s
		FOR VALUES FROM ('%s') TO ('%s')
	`, partitionName, tableName, tableName, startDate.Format("2006-01-02"), endDate.Format("2006-01-02"))

	return t.db.WithContext(ctx).Exec(createSQL).Error
}

func (t *TimePartitioner) cleanupOldPartitions(ctx context.Context) {
	t.mu.Lock()
	defer t.mu.Unlock()

	if t.db == nil {
		return
	}

	cutoffDate := time.Now().AddDate(0, 0, -t.partitionCfg.RetentionDays)

	query := `
		SELECT child.relname AS partition_name
		FROM pg_inherits
		JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
		JOIN pg_class child ON pg_inherits.inhrelid = child.oid
		WHERE parent.relname LIKE 'verification%'
	`

	var partitions []string
	if err := t.db.WithContext(ctx).Raw(query).Scan(&partitions).Error; err != nil {
		log.Printf("Failed to query old partitions: %v", err)
		return
	}

	partitionDateRegex := regexp.MustCompile(`(\d{8})$`)
	for _, partition := range partitions {
		matches := partitionDateRegex.FindStringSubmatch(partition)
		if len(matches) < 2 {
			continue
		}

		dateStr := matches[1]
		partitionDate, err := time.Parse("20060102", dateStr)
		if err != nil {
			continue
		}

		if partitionDate.Before(cutoffDate) {
			if t.partitionCfg.AutoArchive {
				log.Printf("Would archive partition: %s (date: %s)", partition, partitionDate.Format("2006-01-02"))
			} else {
				dropSQL := fmt.Sprintf("DROP TABLE IF EXISTS %s", partition)
				if err := t.db.WithContext(ctx).Exec(dropSQL).Error; err != nil {
					log.Printf("Failed to drop partition %s: %v", partition, err)
				} else {
					log.Printf("Dropped old partition: %s", partition)
				}
			}
		}
	}
}

func (p *TablePartitioner) GetPartitionStats(ctx context.Context, tableName string) (map[string]interface{}, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT 
			COUNT(*) as total_partitions,
			SUM(pg_relation_size(child.oid)) as total_size,
			MAX(pg_relation_size(child.oid)) as max_partition_size,
			AVG(pg_relation_size(child.oid)) as avg_partition_size
		FROM pg_inherits
		JOIN pg_class parent ON pg_inherits.inhparent = parent.oid
		JOIN pg_class child ON pg_inherits.inhrelid = child.oid
		WHERE parent.relname = $1
	`

	type statsResult struct {
		TotalPartitions   int64   `gorm:"column:total_partitions"`
		TotalSize         int64   `gorm:"column:total_size"`
		MaxPartitionSize  int64   `gorm:"column:max_partition_size"`
		AvgPartitionSize  float64 `gorm:"column:avg_partition_size"`
	}

	var stats statsResult
	if err := p.db.WithContext(ctx).Raw(query, tableName).Scan(&stats).Error; err != nil {
		return nil, err
	}

	return map[string]interface{}{
		"table_name":          tableName,
		"total_partitions":    stats.TotalPartitions,
		"total_size_bytes":    stats.TotalSize,
		"max_partition_bytes": stats.MaxPartitionSize,
		"avg_partition_bytes":  stats.AvgPartitionSize,
	}, nil
}

func (p *TablePartitioner) CreateAppShard(ctx context.Context, appID string, shardConfig *ShardConfig) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	shardID := hashAppID(appID, shardConfig.TotalShards)
	shardDBName := fmt.Sprintf("%s_%d", shardConfig.DBPrefix, shardID)

	createDBSQL := fmt.Sprintf("CREATE DATABASE %s", shardDBName)
	if err := p.db.WithContext(ctx).Exec(createDBSQL).Error; err != nil {
		if !strings.Contains(err.Error(), "already exists") {
			return fmt.Errorf("failed to create shard database: %w", err)
		}
	}

	log.Printf("Created app shard: %s (shard_id: %d) for app: %s", shardDBName, shardID, appID)
	return nil
}

func hashAppID(appID string, totalShards int) int {
	hash := 0
	for _, c := range appID {
		hash = int(c) + (hash << 6) + (hash << 16) - hash
	}
	return hash % totalShards
}

func (p *TablePartitioner) GetShardForApp(ctx context.Context, appID string) (*AppShardInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.appPartitioner.shards == nil {
		return nil, fmt.Errorf("shards not initialized")
	}

	shardID := hashAppID(appID, 8)
	shardDBName := fmt.Sprintf("app_shard_%d", shardID)

	info := &AppShardInfo{
		AppID:     appID,
		ShardID:   shardID,
		DBName:    shardDBName,
		IsActive:  true,
		CreatedAt:  time.Now(),
	}

	return info, nil
}

func (p *TablePartitioner) ListAppShards(ctx context.Context) ([]AppShardInfo, error) {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.db == nil {
		return nil, fmt.Errorf("database connection not available")
	}

	query := `
		SELECT datname AS db_name
		FROM pg_database
		WHERE datname LIKE 'app_shard_%'
		ORDER BY datname
	`

	var dbNames []string
	if err := p.db.WithContext(ctx).Raw(query).Scan(&dbNames).Error; err != nil {
		return nil, err
	}

	shards := make([]AppShardInfo, 0, len(dbNames))
	for i, dbName := range dbNames {
		shards = append(shards, AppShardInfo{
			ShardID:   i,
			DBName:    dbName,
			IsActive:  true,
			CreatedAt: time.Now(),
		})
	}

	return shards, nil
}

func (p *TablePartitioner) RebalanceShards(ctx context.Context, newTotalShards int) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	log.Printf("Rebalancing shards to %d total shards", newTotalShards)
	return nil
}

func (p *TablePartitioner) SetPartitionConfig(cfg PartitionConfig) {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.timePartitioner != nil {
		p.timePartitioner.partitionCfg = cfg
	}
}

func (p *TablePartitioner) GetPartitionConfig() PartitionConfig {
	p.mu.RLock()
	defer p.mu.RUnlock()

	if p.timePartitioner != nil {
		return p.timePartitioner.partitionCfg
	}
	return PartitionConfig{}
}

func (p *TablePartitioner) AutoCreatePartitionedTable(ctx context.Context, tableName string, dateColumn string, partitionType string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	createSQL := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			LIKE %s INCLUDING ALL
		) PARTITION BY RANGE (%s)
	`, tableName, tableName, dateColumn)

	if err := p.db.WithContext(ctx).Exec(createSQL).Error; err != nil {
		if strings.Contains(err.Error(), "already exists") {
			return nil
		}
		return fmt.Errorf("failed to create partitioned table: %w", err)
	}

	log.Printf("Created partitioned table: %s partitioned by %s", tableName, dateColumn)
	return nil
}

func (p *TablePartitioner) DetachPartition(ctx context.Context, partitionName string) error {
	p.mu.Lock()
	defer p.mu.Unlock()

	if p.db == nil {
		return fmt.Errorf("database connection not available")
	}

	detachSQL := fmt.Sprintf("ALTER TABLE %s DETACH PARTITION %s", getParentTable(partitionName), partitionName)
	if err := p.db.WithContext(ctx).Exec(detachSQL).Error; err != nil {
		return fmt.Errorf("failed to detach partition: %w", err)
	}

	log.Printf("Detached partition: %s", partitionName)
	return nil
}

func getParentTable(partitionName string) string {
	parts := strings.Split(partitionName, "_")
	if len(parts) >= 1 {
		return parts[0]
	}
	return partitionName
}
