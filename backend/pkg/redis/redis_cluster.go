package redis

import (
	"context"
	"fmt"
	"sync"
	"time"

	goredis "github.com/redis/go-redis/v9"
)

type ClusterMode int

const (
	ClusterModeStandalone ClusterMode = iota
	ClusterModeSentinel
	ClusterModeCluster
)

type ClusterConfig struct {
	Mode           ClusterMode
	Addrs          []string
	MasterName     string
	Password       string
	DB             int
	PoolSize       int
	MinIdleConns   int
	MaxIdleConns   int
	DialTimeout    time.Duration
	ReadTimeout    time.Duration
	WriteTimeout   time.Duration
	ReadWriteSplit bool
}

type EnhancedRedisClient struct {
	mu         sync.RWMutex
	config     *ClusterConfig
	standalone *goredis.Client
	cluster    *goredis.ClusterClient
	sentinel   *goredis.SentinelClient
	readClient goredis.Cmdable
	writeClient goredis.Cmdable
}

var (
	globalEnhancedRedisClient *EnhancedRedisClient
	globalEnhancedRedisOnce sync.Once
)

func NewEnhancedRedisClient(config *ClusterConfig) (*EnhancedRedisClient, error) {
	if config == nil {
		config = &ClusterConfig{
			Mode:         ClusterModeStandalone,
			Addrs:        []string{"localhost:6379"},
			PoolSize:     100,
			MinIdleConns: 10,
			MaxIdleConns: 50,
			DialTimeout:  5 * time.Second,
			ReadTimeout:  3 * time.Second,
			WriteTimeout: 3 * time.Second,
		}
	}

	client := &EnhancedRedisClient{
		config: config,
	}

	if err := client.connect(); err != nil {
		return nil, err
	}

	return client, nil
}

func (erc *EnhancedRedisClient) connect() error {
	erc.mu.Lock()
	defer erc.mu.Unlock()

	switch erc.config.Mode {
	case ClusterModeStandalone:
		return erc.connectStandalone()
	case ClusterModeSentinel:
		return erc.connectSentinel()
	case ClusterModeCluster:
		return erc.connectCluster()
	default:
		return fmt.Errorf("unknown cluster mode: %d", erc.config.Mode)
	}
}

func (erc *EnhancedRedisClient) connectStandalone() error {
	if len(erc.config.Addrs) == 0 {
		return fmt.Errorf("no addresses provided for standalone mode")
	}

	erc.standalone = goredis.NewClient(&goredis.Options{
		Addr:         erc.config.Addrs[0],
		Password:     erc.config.Password,
		DB:           erc.config.DB,
		PoolSize:     erc.config.PoolSize,
		MinIdleConns: erc.config.MinIdleConns,
		MaxIdleConns: erc.config.MaxIdleConns,
		DialTimeout:  erc.config.DialTimeout,
		ReadTimeout:  erc.config.ReadTimeout,
		WriteTimeout: erc.config.WriteTimeout,
	})

	erc.readClient = erc.standalone
	erc.writeClient = erc.standalone

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return erc.standalone.Ping(ctx).Err()
}

func (erc *EnhancedRedisClient) connectSentinel() error {
	if erc.config.MasterName == "" {
		return fmt.Errorf("master name required for sentinel mode")
	}

	if len(erc.config.Addrs) == 0 {
		return fmt.Errorf("no sentinel addresses provided")
	}

	failoverOptions := &goredis.FailoverOptions{
		MasterName:       erc.config.MasterName,
		SentinelAddrs:    erc.config.Addrs,
		Password:         erc.config.Password,
		DB:               erc.config.DB,
		PoolSize:         erc.config.PoolSize,
		MinIdleConns:     erc.config.MinIdleConns,
		MaxIdleConns:     erc.config.MaxIdleConns,
		DialTimeout:      erc.config.DialTimeout,
		ReadTimeout:      erc.config.ReadTimeout,
		WriteTimeout:     erc.config.WriteTimeout,
		ReplicaOnly:      erc.config.ReadWriteSplit,
	}

	erc.standalone = goredis.NewFailoverClient(failoverOptions)

	erc.writeClient = erc.standalone

	if erc.config.ReadWriteSplit {
		readOptions := *failoverOptions
		readOptions.ReplicaOnly = true
		erc.readClient = goredis.NewFailoverClient(&readOptions)
	} else {
		erc.readClient = erc.standalone
	}

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return erc.standalone.Ping(ctx).Err()
}

func (erc *EnhancedRedisClient) connectCluster() error {
	if len(erc.config.Addrs) == 0 {
		return fmt.Errorf("no cluster addresses provided")
	}

	erc.cluster = goredis.NewClusterClient(&goredis.ClusterOptions{
		Addrs:        erc.config.Addrs,
		Password:     erc.config.Password,
		PoolSize:     erc.config.PoolSize,
		MinIdleConns: erc.config.MinIdleConns,
		MaxIdleConns: erc.config.MaxIdleConns,
		DialTimeout:  erc.config.DialTimeout,
		ReadTimeout:  erc.config.ReadTimeout,
		WriteTimeout: erc.config.WriteTimeout,
		ReadOnly:     erc.config.ReadWriteSplit,
		RouteByLatency: erc.config.ReadWriteSplit,
	})

	erc.readClient = erc.cluster
	erc.writeClient = erc.cluster

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	return erc.cluster.Ping(ctx).Err()
}

func (erc *EnhancedRedisClient) Close() error {
	erc.mu.Lock()
	defer erc.mu.Unlock()

	var err error

	if erc.standalone != nil {
		if closeErr := erc.standalone.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if erc.cluster != nil {
		if closeErr := erc.cluster.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	if erc.sentinel != nil {
		if closeErr := erc.sentinel.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	return err
}

func (erc *EnhancedRedisClient) GetClient() goredis.Cmdable {
	erc.mu.RLock()
	defer erc.mu.RUnlock()
	return erc.readClient
}

func (erc *EnhancedRedisClient) GetWriteClient() goredis.Cmdable {
	erc.mu.RLock()
	defer erc.mu.RUnlock()
	return erc.writeClient
}

func (erc *EnhancedRedisClient) Get(ctx context.Context, key string) (string, error) {
	return erc.readClient.Get(ctx, key).Result()
}

func (erc *EnhancedRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	return erc.writeClient.Set(ctx, key, value, expiration).Err()
}

func (erc *EnhancedRedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	return erc.writeClient.Del(ctx, keys...).Result()
}

func (erc *EnhancedRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	return erc.readClient.Exists(ctx, keys...).Result()
}

func (erc *EnhancedRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	return erc.writeClient.Incr(ctx, key).Result()
}

func (erc *EnhancedRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	return erc.writeClient.Expire(ctx, key, expiration).Result()
}

func (erc *EnhancedRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	return erc.readClient.TTL(ctx, key).Result()
}

func (erc *EnhancedRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	return erc.readClient.MGet(ctx, keys...).Result()
}

func (erc *EnhancedRedisClient) Pipeline() goredis.Pipeliner {
	return erc.writeClient.Pipeline()
}

func (erc *EnhancedRedisClient) TxPipelined(ctx context.Context, fn func(goredis.Pipeliner) error) ([]goredis.Cmder, error) {
	return erc.writeClient.TxPipelined(ctx, fn)
}

func (erc *EnhancedRedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	return erc.readClient.HGet(ctx, key, field).Result()
}

func (erc *EnhancedRedisClient) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	return erc.writeClient.HSet(ctx, key, values...).Result()
}

func (erc *EnhancedRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	return erc.readClient.HGetAll(ctx, key).Result()
}

func (erc *EnhancedRedisClient) SAdd(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return erc.writeClient.SAdd(ctx, key, members...).Result()
}

func (erc *EnhancedRedisClient) SMembers(ctx context.Context, key string) ([]string, error) {
	return erc.readClient.SMembers(ctx, key).Result()
}

func (erc *EnhancedRedisClient) SRem(ctx context.Context, key string, members ...interface{}) (int64, error) {
	return erc.writeClient.SRem(ctx, key, members...).Result()
}

func (erc *EnhancedRedisClient) PoolStats() *goredis.PoolStats {
	erc.mu.RLock()
	defer erc.mu.RUnlock()

	if erc.standalone != nil {
		return erc.standalone.PoolStats()
	}
	if erc.cluster != nil {
		return &goredis.PoolStats{}
	}
	return &goredis.PoolStats{}
}

func InitEnhancedRedisClient(config *ClusterConfig) error {
	var err error
	globalEnhancedRedisOnce.Do(func() {
		globalEnhancedRedisClient, err = NewEnhancedRedisClient(config)
	})
	return err
}

func GetEnhancedRedisClient() *EnhancedRedisClient {
	return globalEnhancedRedisClient
}

type PipelineExecutor struct {
	client  goredis.Cmdable
	cmds    []goredis.Cmder
	ctx     context.Context
}

func NewPipelineExecutor(ctx context.Context, client goredis.Cmdable) *PipelineExecutor {
	return &PipelineExecutor{
		client: client,
		ctx:    ctx,
	}
}

func (pe *PipelineExecutor) Get(key string) *goredis.StringCmd {
	cmd := pe.client.Get(pe.ctx, key)
	pe.cmds = append(pe.cmds, cmd)
	return cmd
}

func (pe *PipelineExecutor) Set(key string, value interface{}, expiration time.Duration) *goredis.StatusCmd {
	cmd := pe.client.Set(pe.ctx, key, value, expiration)
	pe.cmds = append(pe.cmds, cmd)
	return cmd
}

func (pe *PipelineExecutor) Exec() ([]goredis.Cmder, error) {
	return pe.cmds, nil
}

type RedisBatchOperator struct {
	client     goredis.Cmdable
	batchSize  int
}

func NewRedisBatchOperator(client goredis.Cmdable, batchSize int) *RedisBatchOperator {
	if batchSize <= 0 {
		batchSize = 100
	}
	return &RedisBatchOperator{
		client:    client,
		batchSize: batchSize,
	}
}

func (bo *RedisBatchOperator) MSet(ctx context.Context, items map[string]interface{}, expiration time.Duration) error {
	if len(items) == 0 {
		return nil
	}

	keys := make([]string, 0, len(items))
	for key := range items {
		keys = append(keys, key)
	}

	for i := 0; i < len(keys); i += bo.batchSize {
		end := i + bo.batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		pipe := bo.client.Pipeline()

		for _, key := range batch {
			pipe.Set(ctx, key, items[key], expiration)
		}

		_, err := pipe.Exec(ctx)
		if err != nil {
			return err
		}
	}

	return nil
}

func (bo *RedisBatchOperator) MGet(ctx context.Context, keys []string) (map[string]string, error) {
	if len(keys) == 0 {
		return make(map[string]string), nil
	}

	result := make(map[string]string)

	for i := 0; i < len(keys); i += bo.batchSize {
		end := i + bo.batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		values, err := bo.client.MGet(ctx, batch...).Result()
		if err != nil {
			return nil, err
		}

		for j, key := range batch {
			if values[j] != nil {
				result[key] = values[j].(string)
			}
		}
	}

	return result, nil
}

func (bo *RedisBatchOperator) MDel(ctx context.Context, keys []string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	var totalDeleted int64

	for i := 0; i < len(keys); i += bo.batchSize {
		end := i + bo.batchSize
		if end > len(keys) {
			end = len(keys)
		}

		batch := keys[i:end]
		deleted, err := bo.client.Del(ctx, batch...).Result()
		if err != nil {
			return totalDeleted, err
		}
		totalDeleted += deleted
	}

	return totalDeleted, nil
}
