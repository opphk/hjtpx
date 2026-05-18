package redis

import (
	"context"
	"fmt"
	"log"
	"math/rand"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/config"
	goredis "github.com/redis/go-redis/v9"
)

type DistributedRedisClient struct {
	mu            sync.RWMutex
	config       *DistributedRedisConfig
	cluster      *goredis.ClusterClient
	standalone   *goredis.Client
	sentinel     *goredis.SentinelClient
	nodes        []*RedisNode
	nodeMu       sync.RWMutex
	currentNode  uint32
	healthChecker *RedisHealthChecker
	failoverEnabled bool
	region        string
	metrics      *DistributedRedisMetrics
}

type DistributedRedisConfig struct {
	Mode             RedisClusterMode
	ClusterNodes     []string
	SentinelNodes    []string
	StandaloneAddr   string
	MasterName       string
	Password         string
	DB               int
	PoolSize         int
	MinIdleConns     int
	MaxIdleConns     int
	DialTimeout      time.Duration
	ReadTimeout      time.Duration
	WriteTimeout     time.Duration
	PoolTimeout      time.Duration
	MaxRetries       int
	FailoverEnabled  bool
	Region           string
	ReplicaLagLimit  time.Duration
	HeartbeatInterval time.Duration
}

type RedisClusterMode int

const (
	RedisModeStandalone RedisClusterMode = iota
	RedisModeSentinel
	RedisModeCluster
	RedisModeDistributed
)

type RedisNode struct {
	ID           string
	Addr         string
	Role         NodeRole
	Region       string
	Healthy      bool
	Latency      time.Duration
	Priority     int
	MasterID     string
	LastHeartbeat time.Time
	Stats        *NodeStats
	mu           sync.RWMutex
}

type NodeRole string

const (
	RoleMaster NodeRole = "master"
	RoleSlave  NodeRole = "slave"
	RoleSentinel NodeRole = "sentinel"
)

type NodeStats struct {
	TotalRequests  atomic.Int64
	FailedRequests atomic.Int64
	AvgLatency     atomic.Int64
	LastLatency    atomic.Int64
}

type RedisHealthChecker struct {
	client         *DistributedRedisClient
	interval       time.Duration
	stopCh         chan struct{}
	wg             sync.WaitGroup
	checkResults   map[string]*HealthCheckResult
	mu             sync.RWMutex
	maxFailCount   int
	enabled        bool
}

type HealthCheckResult struct {
	NodeID      string
	Healthy     bool
	Latency     time.Duration
	Error       error
	LastCheck   time.Time
	FailCount   int
}

type DistributedRedisMetrics struct {
	TotalRequests   atomic.Int64
	MasterRequests   atomic.Int64
	SlaveRequests    atomic.Int64
	FailedRequests   atomic.Int64
	FailoverCount    atomic.Int64
	NodeSwitches     atomic.Int64
	AvgLatency       atomic.Int64
	HitRate          atomic.Int64
	Hits             atomic.Int64
	Misses           atomic.Int64
}

var (
	globalDistributedRedis *DistributedRedisClient
	distributedRedisOnce   sync.Once
)

func NewDistributedRedisClient(cfg *DistributedRedisConfig) (*DistributedRedisClient, error) {
	if cfg == nil {
		cfg = getDefaultDistributedRedisConfig()
	}

	client := &DistributedRedisClient{
		config:       cfg,
		metrics:      &DistributedRedisMetrics{},
		region:       cfg.Region,
		failoverEnabled: cfg.FailoverEnabled,
	}

	if err := client.initialize(); err != nil {
		return nil, err
	}

	if cfg.FailoverEnabled {
		client.healthChecker = newRedisHealthChecker(client, 10*time.Second)
		client.healthChecker.Start()
	}

	return client, nil
}

func getDefaultDistributedRedisConfig() *DistributedRedisConfig {
	return &DistributedRedisConfig{
		Mode:              RedisModeStandalone,
		ClusterNodes:      []string{},
		SentinelNodes:     []string{},
		StandaloneAddr:    "localhost:6379",
		Password:          "",
		DB:                0,
		PoolSize:          100,
		MinIdleConns:      10,
		MaxIdleConns:      50,
		DialTimeout:       5 * time.Second,
		ReadTimeout:       3 * time.Second,
		WriteTimeout:      3 * time.Second,
		PoolTimeout:       4 * time.Second,
		MaxRetries:        3,
		FailoverEnabled:   true,
		Region:            "default",
		ReplicaLagLimit:   100 * time.Millisecond,
		HeartbeatInterval: 30 * time.Second,
	}
}

func (c *DistributedRedisClient) initialize() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	switch c.config.Mode {
	case RedisModeStandalone:
		return c.connectStandalone()
	case RedisModeSentinel:
		return c.connectSentinel()
	case RedisModeCluster:
		return c.connectCluster()
	case RedisModeDistributed:
		return c.connectDistributed()
	default:
		return fmt.Errorf("unsupported redis mode: %d", c.config.Mode)
	}
}

func (c *DistributedRedisClient) connectStandalone() error {
	c.standalone = goredis.NewClient(&goredis.Options{
		Addr:         c.config.StandaloneAddr,
		Password:     c.config.Password,
		DB:           c.config.DB,
		PoolSize:     c.config.PoolSize,
		MinIdleConns: c.config.MinIdleConns,
		MaxIdleConns: c.config.MaxIdleConns,
		DialTimeout:  c.config.DialTimeout,
		ReadTimeout:  c.config.ReadTimeout,
		WriteTimeout: c.config.WriteTimeout,
		PoolTimeout:  c.config.PoolTimeout,
		MaxRetries:   c.config.MaxRetries,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.standalone.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to standalone redis: %w", err)
	}

	log.Printf("[REDIS] Connected to standalone redis at %s", c.config.StandaloneAddr)
	return nil
}

func (c *DistributedRedisClient) connectSentinel() error {
	if c.config.MasterName == "" {
		return fmt.Errorf("master name is required for sentinel mode")
	}

	if len(c.config.SentinelNodes) == 0 {
		return fmt.Errorf("sentinel nodes are required for sentinel mode")
	}

	failoverOptions := &goredis.FailoverOptions{
		MasterName:    c.config.MasterName,
		SentinelAddrs: c.config.SentinelNodes,
		Password:      c.config.Password,
		DB:            c.config.DB,
		PoolSize:      c.config.PoolSize,
		MinIdleConns:  c.config.MinIdleConns,
		MaxIdleConns:  c.config.MaxIdleConns,
		DialTimeout:   c.config.DialTimeout,
		ReadTimeout:   c.config.ReadTimeout,
		WriteTimeout:  c.config.WriteTimeout,
	}

	c.standalone = goredis.NewFailoverClient(failoverOptions)

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.standalone.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to sentinel redis: %w", err)
	}

	if len(c.config.SentinelNodes) > 0 {
		c.sentinel = goredis.NewSentinelClient(&goredis.Options{
			Addr:         c.config.SentinelNodes[0],
			Password:     c.config.Password,
			DialTimeout:  c.config.DialTimeout,
			ReadTimeout:  c.config.ReadTimeout,
			WriteTimeout: c.config.WriteTimeout,
		})
	}

	log.Printf("[REDIS] Connected to sentinel redis, master: %s", c.config.MasterName)
	return nil
}

func (c *DistributedRedisClient) connectCluster() error {
	if len(c.config.ClusterNodes) == 0 {
		return fmt.Errorf("cluster nodes are required for cluster mode")
	}

	c.cluster = goredis.NewClusterClient(&goredis.ClusterOptions{
		Addrs:          c.config.ClusterNodes,
		Password:       c.config.Password,
		PoolSize:       c.config.PoolSize,
		MinIdleConns:   c.config.MinIdleConns,
		MaxIdleConns:   c.config.MaxIdleConns,
		DialTimeout:    c.config.DialTimeout,
		ReadTimeout:    c.config.ReadTimeout,
		WriteTimeout:   c.config.WriteTimeout,
		PoolTimeout:    c.config.PoolTimeout,
		MaxRetries:     c.config.MaxRetries,
		ReadOnly:       true,
		RouteByLatency: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.cluster.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to redis cluster: %w", err)
	}

	log.Printf("[REDIS] Connected to redis cluster with %d nodes", len(c.config.ClusterNodes))
	return nil
}

func (c *DistributedRedisClient) connectDistributed() error {
	if len(c.config.ClusterNodes) == 0 {
		return fmt.Errorf("cluster nodes are required for distributed mode")
	}

	c.nodes = make([]*RedisNode, 0, len(c.config.ClusterNodes))
	for i, addr := range c.config.ClusterNodes {
		node := &RedisNode{
			ID:             fmt.Sprintf("node-%d", i),
			Addr:           addr,
			Role:           RoleMaster,
			Region:         c.config.Region,
			Healthy:        true,
			Priority:       100 - i,
			LastHeartbeat:  time.Now(),
			Stats:          &NodeStats{},
		}
		c.nodes = append(c.nodes, node)
	}

	c.cluster = goredis.NewClusterClient(&goredis.ClusterOptions{
		Addrs:          c.config.ClusterNodes,
		Password:       c.config.Password,
		PoolSize:       c.config.PoolSize,
		MinIdleConns:   c.config.MinIdleConns,
		MaxIdleConns:   c.config.MaxIdleConns,
		DialTimeout:    c.config.DialTimeout,
		ReadTimeout:    c.config.ReadTimeout,
		WriteTimeout:   c.config.WriteTimeout,
		PoolTimeout:    c.config.PoolTimeout,
		MaxRetries:     c.config.MaxRetries,
		ReadOnly:       true,
		RouteByLatency: true,
	})

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	if err := c.cluster.Ping(ctx).Err(); err != nil {
		return fmt.Errorf("failed to connect to distributed redis: %w", err)
	}

	log.Printf("[REDIS] Connected to distributed redis cluster with %d nodes", len(c.nodes))
	return nil
}

func newRedisHealthChecker(client *DistributedRedisClient, interval time.Duration) *RedisHealthChecker {
	return &RedisHealthChecker{
		client:       client,
		interval:     interval,
		stopCh:       make(chan struct{}),
		checkResults: make(map[string]*HealthCheckResult),
		maxFailCount: 3,
		enabled:      true,
	}
}

func (hc *RedisHealthChecker) Start() {
	hc.wg.Add(1)
	go hc.checkLoop()
}

func (hc *RedisHealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *RedisHealthChecker) SetMaxFailCount(count int) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.maxFailCount = count
}

func (hc *RedisHealthChecker) SetEnabled(enabled bool) {
	hc.mu.Lock()
	defer hc.mu.Unlock()
	hc.enabled = enabled
}

func (hc *RedisHealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.performHealthCheck()
			if hc.client.failoverEnabled {
				hc.evaluateFailover()
			}
		}
	}
}

func (hc *RedisHealthChecker) performHealthCheck() {
	hc.mu.RLock()
	client := hc.client
	hc.mu.RUnlock()

	var nodes []*RedisNode
	client.nodeMu.RLock()
	if len(client.nodes) > 0 {
		nodes = make([]*RedisNode, len(client.nodes))
		copy(nodes, client.nodes)
	}
	client.nodeMu.RUnlock()

	if len(nodes) == 0 {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()

		var pingErr error
		start := time.Now()

		if client.cluster != nil {
			if err := client.cluster.Ping(ctx).Err(); err != nil {
				pingErr = err
			}
		} else if client.standalone != nil {
			if err := client.standalone.Ping(ctx).Err(); err != nil {
				pingErr = err
			}
		} else {
			return
		}

		latency := time.Since(start)

		hc.mu.Lock()
		hc.checkResults["default"] = &HealthCheckResult{
			Healthy:   pingErr == nil,
			Latency:   latency,
			Error:     pingErr,
			LastCheck: time.Now(),
		}
		hc.mu.Unlock()
		return
	}

	for _, node := range nodes {
		go hc.checkNode(node)
	}
}

func (hc *RedisHealthChecker) checkNode(node *RedisNode) {
	ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
	defer cancel()

	start := time.Now()
	healthy := true
	var checkErr error

	hc.mu.RLock()
	client := hc.client
	hc.mu.RUnlock()

	if client.cluster != nil {
		if err := client.cluster.Ping(ctx).Err(); err != nil {
			checkErr = err
		}
	} else if client.standalone != nil {
		if err := client.standalone.Ping(ctx).Err(); err != nil {
			checkErr = err
		}
	}

	latency := time.Since(start)
	if checkErr != nil {
		healthy = false
	}

	hc.mu.Lock()
	result, exists := hc.checkResults[node.ID]
	if !exists {
		result = &HealthCheckResult{NodeID: node.ID}
		hc.checkResults[node.ID] = result
	}

	result.Healthy = healthy
	result.Latency = latency
	result.Error = checkErr
	result.LastCheck = time.Now()

	if !healthy {
		result.FailCount++
	} else {
		result.FailCount = 0
	}

	node.mu.Lock()
	node.Healthy = healthy
	node.Latency = latency
	node.LastHeartbeat = time.Now()
	node.mu.Unlock()

	hc.mu.Unlock()
}

func (hc *RedisHealthChecker) evaluateFailover() {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	client := hc.client
	needFailover := false
	var failedNodeID string

	for nodeID, result := range hc.checkResults {
		if !result.Healthy && result.FailCount >= hc.maxFailCount {
			needFailover = true
			failedNodeID = nodeID
			log.Printf("[REDIS_HEALTH] Node %s failed health check, fail count: %d", nodeID, result.FailCount)
			break
		}
	}

	if needFailover && failedNodeID != "" {
		client.nodeMu.Lock()
		for _, node := range client.nodes {
			if node.ID == failedNodeID {
				node.mu.Lock()
				node.Healthy = false
				node.mu.Unlock()
				log.Printf("[REDIS_FAILOVER] Marking node %s as unhealthy", failedNodeID)
				break
			}
		}
		client.nodeMu.Unlock()

		client.metrics.FailoverCount.Add(1)
	}
}

func (hc *RedisHealthChecker) GetHealthyNodes() []string {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	var healthy []string
	for nodeID, result := range hc.checkResults {
		if result.Healthy {
			healthy = append(healthy, nodeID)
		}
	}
	return healthy
}

func (hc *RedisHealthChecker) GetHealthStatus() map[string]*HealthCheckResult {
	hc.mu.RLock()
	defer hc.mu.RUnlock()

	status := make(map[string]*HealthCheckResult)
	for k, v := range hc.checkResults {
		status[k] = v
	}
	return status
}

func (c *DistributedRedisClient) Get(ctx context.Context, key string) (string, error) {
	c.metrics.TotalRequests.Add(1)
	start := time.Now()
	defer func() {
		c.recordLatency(time.Since(start), false)
	}()

	var result string
	var err error

	if c.cluster != nil {
		result, err = c.cluster.Get(ctx, key).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.Get(ctx, key).Result()
	} else {
		return "", fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
		c.metrics.Misses.Add(1)
	} else {
		c.metrics.Hits.Add(1)
	}

	return result, err
}

func (c *DistributedRedisClient) Set(ctx context.Context, key string, value interface{}, expiration time.Duration) error {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)
	start := time.Now()
	defer func() {
		c.recordLatency(time.Since(start), true)
	}()

	var err error
	if c.cluster != nil {
		err = c.cluster.Set(ctx, key, value, expiration).Err()
	} else if c.standalone != nil {
		err = c.standalone.Set(ctx, key, value, expiration).Err()
	} else {
		return fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
	}

	return err
}

func (c *DistributedRedisClient) SetNX(ctx context.Context, key string, value interface{}, expiration time.Duration) (bool, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result bool
	var err error

	if c.cluster != nil {
		result, err = c.cluster.SetNX(ctx, key, value, expiration).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.SetNX(ctx, key, value, expiration).Result()
	} else {
		return false, fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
	}

	return result, err
}

func (c *DistributedRedisClient) Del(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result int64
	var err error

	if c.cluster != nil {
		result, err = c.cluster.Del(ctx, keys...).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.Del(ctx, keys...).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
	}

	return result, err
}

func (c *DistributedRedisClient) Exists(ctx context.Context, keys ...string) (int64, error) {
	if len(keys) == 0 {
		return 0, nil
	}

	c.metrics.TotalRequests.Add(1)
	c.metrics.SlaveRequests.Add(1)

	var result int64
	var err error

	if c.cluster != nil {
		result, err = c.cluster.Exists(ctx, keys...).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.Exists(ctx, keys...).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) Expire(ctx context.Context, key string, expiration time.Duration) (bool, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result bool
	var err error

	if c.cluster != nil {
		result, err = c.cluster.Expire(ctx, key, expiration).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.Expire(ctx, key, expiration).Result()
	} else {
		return false, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) TTL(ctx context.Context, key string) (time.Duration, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.SlaveRequests.Add(1)

	var result time.Duration
	var err error

	if c.cluster != nil {
		result, err = c.cluster.TTL(ctx, key).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.TTL(ctx, key).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) Incr(ctx context.Context, key string) (int64, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result int64
	var err error

	if c.cluster != nil {
		result, err = c.cluster.Incr(ctx, key).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.Incr(ctx, key).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
	}

	return result, err
}

func (c *DistributedRedisClient) IncrBy(ctx context.Context, key string, value int64) (int64, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result int64
	var err error

	if c.cluster != nil {
		result, err = c.cluster.IncrBy(ctx, key, value).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.IncrBy(ctx, key, value).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
	}

	return result, err
}

func (c *DistributedRedisClient) Decr(ctx context.Context, key string) (int64, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result int64
	var err error

	if c.cluster != nil {
		result, err = c.cluster.Decr(ctx, key).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.Decr(ctx, key).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) HSet(ctx context.Context, key string, values ...interface{}) (int64, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var result int64
	var err error

	if c.cluster != nil {
		result, err = c.cluster.HSet(ctx, key, values...).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.HSet(ctx, key, values...).Result()
	} else {
		return 0, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) HGet(ctx context.Context, key, field string) (string, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.SlaveRequests.Add(1)

	var result string
	var err error

	if c.cluster != nil {
		result, err = c.cluster.HGet(ctx, key, field).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.HGet(ctx, key, field).Result()
	} else {
		return "", fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) HGetAll(ctx context.Context, key string) (map[string]string, error) {
	c.metrics.TotalRequests.Add(1)
	c.metrics.SlaveRequests.Add(1)

	var result map[string]string
	var err error

	if c.cluster != nil {
		result, err = c.cluster.HGetAll(ctx, key).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.HGetAll(ctx, key).Result()
	} else {
		return nil, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) MGet(ctx context.Context, keys ...string) ([]interface{}, error) {
	if len(keys) == 0 {
		return []interface{}{}, nil
	}

	c.metrics.TotalRequests.Add(1)
	c.metrics.SlaveRequests.Add(1)

	var result []interface{}
	var err error

	if c.cluster != nil {
		result, err = c.cluster.MGet(ctx, keys...).Result()
	} else if c.standalone != nil {
		result, err = c.standalone.MGet(ctx, keys...).Result()
	} else {
		return nil, fmt.Errorf("no redis client available")
	}

	return result, err
}

func (c *DistributedRedisClient) MSet(ctx context.Context, values ...interface{}) error {
	if len(values) == 0 {
		return nil
	}

	c.metrics.TotalRequests.Add(1)
	c.metrics.MasterRequests.Add(1)

	var err error

	if c.cluster != nil {
		err = c.cluster.MSet(ctx, values...).Err()
	} else if c.standalone != nil {
		err = c.standalone.MSet(ctx, values...).Err()
	} else {
		return fmt.Errorf("no redis client available")
	}

	if err != nil {
		c.metrics.FailedRequests.Add(1)
	}

	return err
}

func (c *DistributedRedisClient) Pipeline() goredis.Pipeliner {
	if c.cluster != nil {
		return c.cluster.Pipeline()
	} else if c.standalone != nil {
		return c.standalone.Pipeline()
	}
	return nil
}

func (c *DistributedRedisClient) TxPipeline() goredis.Pipeliner {
	if c.cluster != nil {
		return c.cluster.TxPipeline()
	} else if c.standalone != nil {
		return c.standalone.TxPipeline()
	}
	return nil
}

func (c *DistributedRedisClient) recordLatency(latency time.Duration, isWrite bool) {
	total := c.metrics.TotalRequests.Load()
	if total > 0 {
		avgLatency := (c.metrics.AvgLatency.Load()*(total-1) + latency.Nanoseconds()) / total
		c.metrics.AvgLatency.Store(avgLatency)
	}

	hits := c.metrics.Hits.Load()
	misses := c.metrics.Misses.Load()
	totalCache := hits + misses
	if totalCache > 0 {
		hitRateValue := int64(float64(hits) / float64(totalCache) * 100)
		c.metrics.HitRate.Store(hitRateValue)
	}
}

func (c *DistributedRedisClient) GetMetrics() *DistributedRedisMetrics {
	return c.metrics
}

func (c *DistributedRedisClient) GetNodeStats() []*RedisNode {
	c.nodeMu.RLock()
	defer c.nodeMu.RUnlock()

	stats := make([]*RedisNode, len(c.nodes))
	copy(stats, c.nodes)
	return stats
}

func (c *DistributedRedisClient) GetPoolStats() *goredis.PoolStats {
	if c.cluster != nil {
		return &goredis.PoolStats{}
	} else if c.standalone != nil {
		return c.standalone.PoolStats()
	}
	return &goredis.PoolStats{}
}

func (c *DistributedRedisClient) Ping(ctx context.Context) error {
	var err error
	if c.cluster != nil {
		err = c.cluster.Ping(ctx).Err()
	} else if c.standalone != nil {
		err = c.standalone.Ping(ctx).Err()
	}
	return err
}

func (c *DistributedRedisClient) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.healthChecker != nil {
		c.healthChecker.Stop()
	}

	var err error
	if c.cluster != nil {
		if closeErr := c.cluster.Close(); closeErr != nil {
			err = closeErr
		}
	}

	if c.standalone != nil {
		if closeErr := c.standalone.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	if c.sentinel != nil {
		if closeErr := c.sentinel.Close(); closeErr != nil && err == nil {
			err = closeErr
		}
	}

	return err
}

func (c *DistributedRedisClient) AddNode(addr string, role NodeRole) error {
	c.nodeMu.Lock()
	defer c.nodeMu.Unlock()

	for _, node := range c.nodes {
		if node.Addr == addr {
			return fmt.Errorf("node %s already exists", addr)
		}
	}

	node := &RedisNode{
		ID:            fmt.Sprintf("node-%d", len(c.nodes)),
		Addr:          addr,
		Role:          role,
		Region:        c.region,
		Healthy:       true,
		Priority:      100,
		LastHeartbeat: time.Now(),
		Stats:         &NodeStats{},
	}

	c.nodes = append(c.nodes, node)
	c.metrics.NodeSwitches.Add(1)

	return nil
}

func (c *DistributedRedisClient) RemoveNode(addr string) error {
	c.nodeMu.Lock()
	defer c.nodeMu.Unlock()

	for i, node := range c.nodes {
		if node.Addr == addr {
			c.nodes = append(c.nodes[:i], c.nodes[i+1:]...)
			c.metrics.NodeSwitches.Add(1)
			return nil
		}
	}

	return fmt.Errorf("node %s not found", addr)
}

func (c *DistributedRedisClient) GetOptimalNode() *RedisNode {
	c.nodeMu.RLock()
	defer c.nodeMu.RUnlock()

	var optimal *RedisNode
	var minLatency time.Duration = time.Hour

	for _, node := range c.nodes {
		if !node.Healthy {
			continue
		}

		node.mu.RLock()
		if node.Latency < minLatency {
			minLatency = node.Latency
			optimal = node
		}
		node.mu.RUnlock()
	}

	if optimal == nil && len(c.nodes) > 0 {
		optimal = c.nodes[rand.Intn(len(c.nodes))]
	}

	return optimal
}

func (c *DistributedRedisClient) SetNodeHealthy(addr string, healthy bool) error {
	c.nodeMu.Lock()
	defer c.nodeMu.Unlock()

	for _, node := range c.nodes {
		if node.Addr == addr {
			node.mu.Lock()
			node.Healthy = healthy
			node.mu.Unlock()
			return nil
		}
	}

	return fmt.Errorf("node %s not found", addr)
}

func InitDistributedRedisClient(cfg *config.DistributedRedisConfig) error {
	var err error
	distributedRedisOnce.Do(func() {
		redisCfg := &DistributedRedisConfig{
			Mode:              RedisClusterMode(cfg.Mode),
			ClusterNodes:      cfg.ClusterNodes,
			SentinelNodes:     cfg.SentinelNodes,
			StandaloneAddr:    cfg.StandaloneAddr,
			MasterName:        cfg.MasterName,
			Password:          cfg.Password,
			DB:                cfg.DB,
			PoolSize:          cfg.PoolSize,
			MinIdleConns:      cfg.MinIdleConns,
			MaxIdleConns:      cfg.MaxIdleConns,
			DialTimeout:       time.Duration(cfg.DialTimeoutSecs) * time.Second,
			ReadTimeout:       time.Duration(cfg.ReadTimeoutSecs) * time.Second,
			WriteTimeout:      time.Duration(cfg.WriteTimeoutSecs) * time.Second,
			PoolTimeout:       time.Duration(cfg.PoolTimeoutSecs) * time.Second,
			MaxRetries:       cfg.MaxRetries,
			FailoverEnabled:  cfg.FailoverEnabled,
			Region:            cfg.Region,
			ReplicaLagLimit:   time.Duration(cfg.ReplicaLagLimitSecs) * time.Second,
			HeartbeatInterval: time.Duration(cfg.HeartbeatIntervalSecs) * time.Second,
		}
		globalDistributedRedis, err = NewDistributedRedisClient(redisCfg)
	})
	return err
}

func GetDistributedRedisClient() *DistributedRedisClient {
	return globalDistributedRedis
}

type ClusterDistributedLock struct {
	client *DistributedRedisClient
	key    string
	value  string
	ttl    time.Duration
}

func NewClusterDistributedLock(client *DistributedRedisClient, key, value string, ttl time.Duration) *ClusterDistributedLock {
	return &ClusterDistributedLock{
		client: client,
		key:    key,
		value:  value,
		ttl:    ttl,
	}
}

func (l *ClusterDistributedLock) Acquire(ctx context.Context) (bool, error) {
	if l.client == nil {
		return false, fmt.Errorf("redis client is nil")
	}

	result, err := l.client.SetNX(ctx, l.key, l.value, l.ttl)
	if err != nil {
		return false, err
	}

	return result, nil
}

func (l *ClusterDistributedLock) Release(ctx context.Context) error {
	if l.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("del", KEYS[1])
		else
			return 0
		end
	`

	var err error
	if l.client.cluster != nil {
		_, err = l.client.cluster.Eval(ctx, script, []string{l.key}, l.value).Result()
	} else if l.client.standalone != nil {
		_, err = l.client.standalone.Eval(ctx, script, []string{l.key}, l.value).Result()
	}

	return err
}

func (l *ClusterDistributedLock) Extend(ctx context.Context, ttl time.Duration) error {
	if l.client == nil {
		return fmt.Errorf("redis client is nil")
	}

	script := `
		if redis.call("get", KEYS[1]) == ARGV[1] then
			return redis.call("pexpire", KEYS[1], ARGV[2])
		else
			return 0
		end
	`

	var err error
	if l.client.cluster != nil {
		_, err = l.client.cluster.Eval(ctx, script, []string{l.key}, l.value, ttl.Milliseconds()).Result()
	} else if l.client.standalone != nil {
		_, err = l.client.standalone.Eval(ctx, script, []string{l.key}, l.value, ttl.Milliseconds()).Result()
	}

	return err
}

func (l *ClusterDistributedLock) TryWithLock(ctx context.Context, fn func() error) error {
	acquired, err := l.Acquire(ctx)
	if err != nil {
		return err
	}
	if !acquired {
		return fmt.Errorf("failed to acquire lock for key: %s", l.key)
	}

	defer func() {
		if releaseErr := l.Release(ctx); releaseErr != nil {
			fmt.Printf("[CLUSTER_DISTRIBUTED_LOCK] Failed to release lock for key %s: %v\n", l.key, releaseErr)
		}
	}()

	return fn()
}
