package redis

import (
	"context"
	"database/sql"
	"fmt"
	"sync"
	"time"

	_ "github.com/lib/pq"
)

const (
	DefaultL3TTL = 24 * time.Hour
	L3Table      = "cache_store"
)

type L3CacheConfig struct {
	Enabled      bool
	DSN          string
	TTL          time.Duration
	MaxOpenConns int
	MaxIdleConns int
	ConnMaxLifetime time.Duration
}

var DefaultL3CacheConfig = &L3CacheConfig{
	Enabled:         false,
	TTL:             DefaultL3TTL,
	MaxOpenConns:    10,
	MaxIdleConns:    5,
	ConnMaxLifetime: 1 * time.Hour,
}

type L3Cache struct {
	config     *L3CacheConfig
	db         *sql.DB
	mu         sync.RWMutex
	initialized bool
}

type L3CacheEntry struct {
	Key        string
	Value      []byte
	TTL        time.Duration
	Version    int64
	Tags       string
	Compressed bool
	CreatedAt  time.Time
	UpdatedAt  time.Time
}

var globalL3Cache *L3Cache
var globalL3CacheOnce sync.Once

func NewL3Cache(config *L3CacheConfig) *L3Cache {
	if config == nil {
		config = DefaultL3CacheConfig
	}

	return &L3Cache{
		config: config,
	}
}

func (l3 *L3Cache) Initialize(ctx context.Context) error {
	l3.mu.Lock()
	defer l3.mu.Unlock()

	if l3.initialized {
		return nil
	}

	if !l3.config.Enabled {
		return nil
	}

	var err error
	l3.db, err = sql.Open("postgres", l3.config.DSN)
	if err != nil {
		return err
	}

	l3.db.SetMaxOpenConns(l3.config.MaxOpenConns)
	l3.db.SetMaxIdleConns(l3.config.MaxIdleConns)
	l3.db.SetConnMaxLifetime(l3.config.ConnMaxLifetime)

	if err := l3.db.PingContext(ctx); err != nil {
		return err
	}

	if err := l3.ensureTable(ctx); err != nil {
		return err
	}

	l3.initialized = true
	return nil
}

func (l3 *L3Cache) ensureTable(ctx context.Context) error {
	query := fmt.Sprintf(`
		CREATE TABLE IF NOT EXISTS %s (
			key TEXT PRIMARY KEY,
			value BYTEA NOT NULL,
			ttl BIGINT NOT NULL,
			version BIGINT NOT NULL DEFAULT 0,
			tags TEXT,
			compressed BOOLEAN NOT NULL DEFAULT false,
			created_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP,
			updated_at TIMESTAMP NOT NULL DEFAULT CURRENT_TIMESTAMP
		);
		CREATE INDEX IF NOT EXISTS idx_%s_expiry ON %s (created_at);
		CREATE INDEX IF NOT EXISTS idx_%s_tags ON %s (tags);
	`, L3Table, L3Table, L3Table, L3Table, L3Table)

	_, err := l3.db.ExecContext(ctx, query)
	return err
}

func (l3 *L3Cache) Get(ctx context.Context, key string) ([]byte, error) {
	if !l3.config.Enabled || !l3.initialized {
		return nil, ErrCacheMiss
	}

	l3.mu.RLock()
	defer l3.mu.RUnlock()

	var value []byte
	var createdAt time.Time
	var ttl int64

	err := l3.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT value, created_at, ttl FROM %s WHERE key = $1`, L3Table),
		key).Scan(&value, &createdAt, &ttl)

	if err == sql.ErrNoRows {
		return nil, ErrCacheMiss
	}

	if err != nil {
		return nil, err
	}

	if time.Since(createdAt) > time.Duration(ttl)*time.Second {
		l3.delete(ctx, key)
		return nil, ErrCacheMiss
	}

	return value, nil
}

func (l3 *L3Cache) Set(ctx context.Context, key string, value []byte, ttl time.Duration, version int64, tags []string, compressed bool) error {
	if !l3.config.Enabled || !l3.initialized {
		return nil
	}

	l3.mu.Lock()
	defer l3.mu.Unlock()

	tagsStr := ""
	if len(tags) > 0 {
		tagsStr = fmt.Sprintf("[%s]", joinStrings(tags, ","))
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (key, value, ttl, version, tags, compressed, updated_at)
		VALUES ($1, $2, $3, $4, $5, $6, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			ttl = EXCLUDED.ttl,
			version = EXCLUDED.version,
			tags = EXCLUDED.tags,
			compressed = EXCLUDED.compressed,
			updated_at = CURRENT_TIMESTAMP
	`, L3Table)

	_, err := l3.db.ExecContext(ctx, query, key, value, int64(ttl.Seconds()), version, tagsStr, compressed)
	return err
}

func (l3 *L3Cache) Delete(ctx context.Context, key string) error {
	if !l3.config.Enabled || !l3.initialized {
		return nil
	}

	return l3.delete(ctx, key)
}

func (l3 *L3Cache) delete(ctx context.Context, key string) error {
	_, err := l3.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE key = $1`, L3Table), key)
	return err
}

func (l3 *L3Cache) DeleteByTag(ctx context.Context, tag string) error {
	if !l3.config.Enabled || !l3.initialized {
		return nil
	}

	query := fmt.Sprintf(`DELETE FROM %s WHERE tags LIKE $1`, L3Table)
	_, err := l3.db.ExecContext(ctx, query, fmt.Sprintf("%%%s%%", tag))
	return err
}

func (l3 *L3Cache) DeleteExpired(ctx context.Context) (int64, error) {
	if !l3.config.Enabled || !l3.initialized {
		return 0, nil
	}

	result, err := l3.db.ExecContext(ctx,
		fmt.Sprintf(`DELETE FROM %s WHERE created_at + INTERVAL '1 second' * ttl < CURRENT_TIMESTAMP`, L3Table))
	if err != nil {
		return 0, err
	}

	return result.RowsAffected()
}

func (l3 *L3Cache) GetVersion(ctx context.Context, key string) (int64, error) {
	if !l3.config.Enabled || !l3.initialized {
		return 0, nil
	}

	var version int64
	err := l3.db.QueryRowContext(ctx,
		fmt.Sprintf(`SELECT version FROM %s WHERE key = $1`, L3Table), key).Scan(&version)

	if err == sql.ErrNoRows {
		return 0, nil
	}

	return version, err
}

func (l3 *L3Cache) MGet(ctx context.Context, keys []string) (map[string][]byte, error) {
	if !l3.config.Enabled || !l3.initialized {
		return nil, nil
	}

	if len(keys) == 0 {
		return make(map[string][]byte), nil
	}

	placeholders := make([]string, len(keys))
	args := make([]interface{}, len(keys))
	for i, key := range keys {
		placeholders[i] = fmt.Sprintf("$%d", i+1)
		args[i] = key
	}

	query := fmt.Sprintf(`SELECT key, value, created_at, ttl FROM %s WHERE key IN (%s)`,
		L3Table, joinStrings(placeholders, ","))

	rows, err := l3.db.QueryContext(ctx, query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	result := make(map[string][]byte)
	for rows.Next() {
		var key string
		var value []byte
		var createdAt time.Time
		var ttl int64

		if err := rows.Scan(&key, &value, &createdAt, &ttl); err != nil {
			return nil, err
		}

		if time.Since(createdAt) <= time.Duration(ttl)*time.Second {
			result[key] = value
		}
	}

	return result, rows.Err()
}

func (l3 *L3Cache) MSet(ctx context.Context, items map[string][]byte, ttl time.Duration) error {
	if !l3.config.Enabled || !l3.initialized {
		return nil
	}

	if len(items) == 0 {
		return nil
	}

	tx, err := l3.db.BeginTx(ctx, nil)
	if err != nil {
		return err
	}

	query := fmt.Sprintf(`
		INSERT INTO %s (key, value, ttl, version, updated_at)
		VALUES ($1, $2, $3, 0, CURRENT_TIMESTAMP)
		ON CONFLICT (key) DO UPDATE SET
			value = EXCLUDED.value,
			ttl = EXCLUDED.ttl,
			updated_at = CURRENT_TIMESTAMP
	`, L3Table)

	ttlSeconds := int64(ttl.Seconds())
	for key, value := range items {
		if _, err := tx.ExecContext(ctx, query, key, value, ttlSeconds); err != nil {
			tx.Rollback()
			return err
		}
	}

	return tx.Commit()
}

func (l3 *L3Cache) GetStats(ctx context.Context) (map[string]interface{}, error) {
	if !l3.config.Enabled || !l3.initialized {
		return nil, nil
	}

	var count int64
	err := l3.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT COUNT(*) FROM %s`, L3Table)).Scan(&count)
	if err != nil {
		return nil, err
	}

	var size int64
	err = l3.db.QueryRowContext(ctx, fmt.Sprintf(`SELECT SUM(OCTET_LENGTH(value)) FROM %s`, L3Table)).Scan(&size)
	if err != nil && err != sql.ErrNoRows {
		return nil, err
	}

	return map[string]interface{}{
		"entry_count": count,
		"total_size":  size,
	}, nil
}

func (l3 *L3Cache) Close() {
	if l3.db != nil {
		l3.db.Close()
	}
}

func joinStrings(strs []string, sep string) string {
	result := ""
	for i, s := range strs {
		if i > 0 {
			result += sep
		}
		result += s
	}
	return result
}

func InitL3Cache(config *L3CacheConfig) {
	globalL3CacheOnce.Do(func() {
		globalL3Cache = NewL3Cache(config)
	})
}

func GetL3Cache() *L3Cache {
	if globalL3Cache == nil {
		InitL3Cache(nil)
	}
	return globalL3Cache
}
