package database

import (
	"context"
	"database/sql"
	"fmt"
	"math/rand"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

type DBRole string

const (
	DBRolePrimary DBRole = "primary"
	DBRoleReplica DBRole = "replica"
)

type DBConfig struct {
	Host            string
	Port            int
	User            string
	Password        string
	DBName          string
	SSLMode         string
	MaxOpenConns    int
	MaxIdleConns    int
	ConnMaxLifetime time.Duration
	Role            DBRole
}

type ReadWriteSplitter struct {
	primary       *sql.DB
	replicas      []*sql.DB
	replicaHealth map[int]*HealthStatus
	mu            sync.RWMutex
	rand          *rand.Rand
}

type HealthStatus struct {
	Healthy    bool
	LagBytes    int64
	LagSeconds  float64
	LastChecked time.Time
}

func NewReadWriteSplitter(primaryCfg *DBConfig, replicaCfgs []*DBConfig) (*ReadWriteSplitter, error) {
	rws := &ReadWriteSplitter{
		replicaHealth: make(map[int]*HealthStatus),
		rand:          rand.New(rand.NewSource(time.Now().UnixNano())),
	}

	if primaryCfg != nil {
		primaryDSN := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			primaryCfg.Host, primaryCfg.Port, primaryCfg.User, primaryCfg.Password,
			primaryCfg.DBName, primaryCfg.SSLMode,
		)
		primary, err := sql.Open("postgres", primaryDSN)
		if err != nil {
			return nil, fmt.Errorf("failed to open primary connection: %w", err)
		}
		primary.SetMaxOpenConns(primaryCfg.MaxOpenConns)
		primary.SetMaxIdleConns(primaryCfg.MaxIdleConns)
		primary.SetConnMaxLifetime(primaryCfg.ConnMaxLifetime)
		rws.primary = primary
	}

	for i, replica := range replicaCfgs {
		dsn := fmt.Sprintf(
			"host=%s port=%d user=%s password=%s dbname=%s sslmode=%s",
			replica.Host, replica.Port, replica.User, replica.Password,
			replica.DBName, replica.SSLMode,
		)
		db, err := sql.Open("postgres", dsn)
		if err != nil {
			return nil, fmt.Errorf("failed to open replica connection: %w", err)
		}
		db.SetMaxOpenConns(replica.MaxOpenConns)
		db.SetMaxIdleConns(replica.MaxIdleConns)
		db.SetConnMaxLifetime(replica.ConnMaxLifetime)
		rws.replicas = append(rws.replicas, db)
		rws.replicaHealth[i] = &HealthStatus{Healthy: true}
	}

	return rws, nil
}

func (rws *ReadWriteSplitter) Primary() *sql.DB {
	rws.mu.RLock()
	defer rws.mu.RUnlock()
	return rws.primary
}

func (rws *ReadWriteSplitter) Replica() *sql.DB {
	rws.mu.RLock()
	defer rws.mu.RUnlock()

	if len(rws.replicas) == 0 {
		return rws.primary
	}

	healthyReplicas := make([]int, 0, len(rws.replicas))
	for i, status := range rws.replicaHealth {
		if status.Healthy {
			healthyReplicas = append(healthyReplicas, i)
		}
	}

	if len(healthyReplicas) == 0 {
		return rws.primary
	}

	idx := healthyReplicas[rws.rand.Intn(len(healthyReplicas))]
	return rws.replicas[idx]
}

func (rws *ReadWriteSplitter) ReadQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db := rws.getReadDB()
	return db.QueryContext(ctx, query, args...)
}

func (rws *ReadWriteSplitter) ReadQueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	db := rws.getReadDB()
	return db.QueryRowContext(ctx, query, args...)
}

func (rws *ReadWriteSplitter) WriteExec(ctx context.Context, query string, args ...interface{}) (sql.Result, error) {
	db := rws.getWriteDB()
	return db.ExecContext(ctx, query, args...)
}

func (rws *ReadWriteSplitter) WriteQuery(ctx context.Context, query string, args ...interface{}) (*sql.Rows, error) {
	db := rws.getWriteDB()
	return db.QueryContext(ctx, query, args...)
}

func (rws *ReadWriteSplitter) WriteQueryRow(ctx context.Context, query string, args ...interface{}) *sql.Row {
	db := rws.getWriteDB()
	return db.QueryRowContext(ctx, query, args...)
}

func (rws *ReadWriteSplitter) getReadDB() *sql.DB {
	if len(rws.replicas) > 0 && rws.hasHealthyReplicas() {
		return rws.Replica()
	}
	return rws.Primary()
}

func (rws *ReadWriteSplitter) getWriteDB() *sql.DB {
	return rws.Primary()
}

func (rws *ReadWriteSplitter) hasHealthyReplicas() bool {
	rws.mu.RLock()
	defer rws.mu.RUnlock()

	for _, status := range rws.replicaHealth {
		if status.Healthy {
			return true
		}
	}
	return false
}

func (rws *ReadWriteSplitter) CheckReplicaHealth(ctx context.Context) {
	rws.mu.Lock()
	defer rws.mu.Unlock()

	for i, db := range rws.replicas {
		if db == nil {
			continue
		}

		err := db.PingContext(ctx)
		status := rws.replicaHealth[i]
		if status == nil {
			status = &HealthStatus{}
			rws.replicaHealth[i] = status
		}

		status.Healthy = err == nil
		status.LastChecked = time.Now()
	}
}

func (rws *ReadWriteSplitter) GetPoolStats() (primary, replicas []ConnectionPoolStats) {
	rws.mu.RLock()
	defer rws.mu.RUnlock()

	if rws.primary != nil {
		stats := rws.primary.Stats()
		primary = append(primary, ConnectionPoolStats{
			MaxOpenConns:      stats.MaxOpenConnections,
			OpenConns:         stats.OpenConnections,
			IdleConns:         stats.Idle,
			WaitCount:         stats.WaitCount,
			WaitDurationMs:    stats.WaitDuration.Milliseconds(),
			MaxIdleClosed:     stats.MaxIdleClosed,
			MaxLifetimeClosed: stats.MaxLifetimeClosed,
		})
	}

	for _, db := range rws.replicas {
		if db != nil {
			stats := db.Stats()
			replicas = append(replicas, ConnectionPoolStats{
				MaxOpenConns:      stats.MaxOpenConnections,
				OpenConns:         stats.OpenConnections,
				IdleConns:         stats.Idle,
				WaitCount:         stats.WaitCount,
				WaitDurationMs:    stats.WaitDuration.Milliseconds(),
				MaxIdleClosed:     stats.MaxIdleClosed,
				MaxLifetimeClosed: stats.MaxLifetimeClosed,
			})
		}
	}

	return primary, replicas
}

func (rws *ReadWriteSplitter) Close() error {
	rws.mu.Lock()
	defer rws.mu.Unlock()

	var errs []error

	if rws.primary != nil {
		if err := rws.primary.Close(); err != nil {
			errs = append(errs, err)
		}
	}

	for _, replica := range rws.replicas {
		if replica != nil {
			if err := replica.Close(); err != nil {
				errs = append(errs, err)
			}
		}
	}

	if len(errs) > 0 {
		return fmt.Errorf("errors closing connections: %v", errs)
	}
	return nil
}

func (rws *ReadWriteSplitter) Ping(ctx context.Context) error {
	if err := rws.Primary().PingContext(ctx); err != nil {
		return fmt.Errorf("primary ping failed: %w", err)
	}
	return nil
}

type ConnectionPoolStats struct {
	MaxOpenConns      int   `json:"max_open_conns"`
	OpenConns         int   `json:"open_conns"`
	IdleConns         int   `json:"idle_conns"`
	WaitCount         int64 `json:"wait_count"`
	WaitDurationMs    int64 `json:"wait_duration_ms"`
	MaxIdleClosed     int64 `json:"max_idle_closed"`
	MaxLifetimeClosed int64 `json:"max_lifetime_closed"`
}

type RoutingHint struct {
	ForcePrimary       bool
	ForceReplica       bool
	PreferredReplica   int
}
