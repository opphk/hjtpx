package servicediscovery

import (
	"context"
	"encoding/json"
	"fmt"
	"log"
	"math"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

type DistributedServiceRegistry struct {
	mu          sync.RWMutex
	instances   map[string]*ServiceInstance
	localAddr   string
	localPort   int
	region      string
	dc          string
	version     string
	heartbeatInterval time.Duration
	ttl         time.Duration
	healthChecker *ServiceHealthChecker
	redisClient  *redis.DistributedRedisClient
	useRedis     bool
	metrics      *RegistryMetrics
}

type RegistryMetrics struct {
	TotalRegistrations atomic.Int64
	TotalDeregistrations atomic.Int64
	TotalDiscoveries  atomic.Int64
	HeartbeatSuccess  atomic.Int64
	HeartbeatFailure  atomic.Int64
	FailoverEvents    atomic.Int64
	InstanceCount     atomic.Int64
}

type ServiceHealthChecker struct {
	registry *DistributedServiceRegistry
	interval time.Duration
	stopCh   chan struct{}
	wg       sync.WaitGroup
}

type ServiceDiscoveryConfig struct {
	ServiceName         string
	ServiceAddr         string
	ServicePort         int
	Region              string
	DC                  string
	Version             string
	HeartbeatInterval   time.Duration
	TTL                 time.Duration
	HealthCheckInterval time.Duration
	MaxHeartbeatMisses  int
	UseRedis            bool
}

var (
	globalRegistry *DistributedServiceRegistry
	registryOnce   sync.Once
)

func NewDistributedRegistry(config *ServiceDiscoveryConfig) (*DistributedServiceRegistry, error) {
	registry := &DistributedServiceRegistry{
		instances:         make(map[string]*ServiceInstance),
		localAddr:         config.ServiceAddr,
		localPort:         config.ServicePort,
		region:            config.Region,
		dc:                config.DC,
		version:           config.Version,
		heartbeatInterval: config.HeartbeatInterval,
		ttl:               config.TTL,
		useRedis:          config.UseRedis,
		metrics:           &RegistryMetrics{},
	}

	if config.HeartbeatInterval == 0 {
		registry.heartbeatInterval = 10 * time.Second
	}
	if config.TTL == 0 {
		registry.ttl = 30 * time.Second
	}

	if config.UseRedis {
		redisClient := redis.GetDistributedRedisClient()
		if redisClient != nil {
			registry.redisClient = redisClient
		} else {
			log.Printf("[SERVICE_DISCOVERY] Redis not available, using in-memory registry")
			registry.useRedis = false
		}
	}

	registry.healthChecker = newServiceHealthChecker(registry, config.HealthCheckInterval)
	registry.healthChecker.Start()

	go registry.startHeartbeat()

	log.Printf("[SERVICE_DISCOVERY] Registry initialized for %s:%d, region=%s, dc=%s",
		config.ServiceAddr, config.ServicePort, config.Region, config.DC)

	return registry, nil
}

func newServiceHealthChecker(registry *DistributedServiceRegistry, interval time.Duration) *ServiceHealthChecker {
	if interval == 0 {
		interval = 15 * time.Second
	}
	return &ServiceHealthChecker{
		registry: registry,
		interval: interval,
		stopCh:   make(chan struct{}),
	}
}

func (hc *ServiceHealthChecker) Start() {
	hc.wg.Add(1)
	go hc.checkLoop()
}

func (hc *ServiceHealthChecker) Stop() {
	close(hc.stopCh)
	hc.wg.Wait()
}

func (hc *ServiceHealthChecker) checkLoop() {
	defer hc.wg.Done()

	ticker := time.NewTicker(hc.interval)
	defer ticker.Stop()

	for {
		select {
		case <-hc.stopCh:
			return
		case <-ticker.C:
			hc.registry.performHealthCheck()
		}
	}
}

func (r *DistributedServiceRegistry) performHealthCheck() {
	r.mu.Lock()
	defer r.mu.Unlock()

	now := time.Now()
	for id, inst := range r.instances {
		timeSinceHeartbeat := now.Sub(inst.LastHeartbeat)
		if timeSinceHeartbeat > r.ttl {
			log.Printf("[SERVICE_DISCOVERY] Instance %s (%s:%d) heartbeat timeout, marking unhealthy",
				id, inst.Address, inst.Port)
			inst.Healthy = false
		}

		if !inst.Healthy && timeSinceHeartbeat > r.ttl*3 {
			log.Printf("[SERVICE_DISCOVERY] Instance %s expired, removing from registry", id)
			delete(r.instances, id)
			r.metrics.TotalDeregistrations.Add(1)
		}
	}

	r.metrics.InstanceCount.Store(int64(len(r.instances)))
}

func (r *DistributedServiceRegistry) startHeartbeat() {
	ticker := time.NewTicker(r.heartbeatInterval)
	defer ticker.Stop()

	for range ticker.C {
		r.sendHeartbeat()
	}
}

func (r *DistributedServiceRegistry) sendHeartbeat() {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	instanceID := fmt.Sprintf("%s:%d", r.localAddr, r.localPort)

	r.mu.RLock()
	inst, exists := r.instances[instanceID]
	if !exists {
		r.mu.RUnlock()
		return
	}
	inst.LastHeartbeat = time.Now()
	inst.Healthy = true
	r.mu.RUnlock()

	if r.useRedis && r.redisClient != nil {
		key := fmt.Sprintf("service:heartbeat:%s", instanceID)
		err := r.redisClient.Set(ctx, key, time.Now().Unix(), r.ttl)
		if err != nil {
			r.metrics.HeartbeatFailure.Add(1)
			log.Printf("[SERVICE_DISCOVERY] Failed to update heartbeat in Redis: %v", err)
		} else {
			r.metrics.HeartbeatSuccess.Add(1)
		}
	}
}

func (r *DistributedServiceRegistry) Register(instance *ServiceInstance) error {
	if instance == nil {
		return fmt.Errorf("instance is nil")
	}

	if instance.ID == "" {
		instance.ID = fmt.Sprintf("%s:%d", instance.Address, instance.Port)
	}

	instance.LastHeartbeat = time.Now()
	instance.RegisteredAt = time.Now()
	instance.Healthy = true
	instance.Region = r.region
	instance.DC = r.dc
	instance.Version = r.version

	r.mu.Lock()
	r.instances[instance.ID] = instance
	r.mu.Unlock()

	r.metrics.TotalRegistrations.Add(1)
	r.metrics.InstanceCount.Add(1)

	if r.useRedis && r.redisClient != nil {
		go r.registerInRedis(instance)
	}

	log.Printf("[SERVICE_DISCOVERY] Registered instance: %s (%s:%d), region=%s",
		instance.Name, instance.Address, instance.Port, instance.Region)

	return nil
}

func (r *DistributedServiceRegistry) registerInRedis(instance *ServiceInstance) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	key := fmt.Sprintf("service:instance:%s:%s", instance.Name, instance.ID)
	data, err := json.Marshal(instance)
	if err != nil {
		return err
	}

	return r.redisClient.Set(ctx, key, string(data), r.ttl*2)
}

func (r *DistributedServiceRegistry) Deregister(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	if _, exists := r.instances[id]; !exists {
		return fmt.Errorf("instance %s not found", id)
	}

	delete(r.instances, id)
	r.metrics.TotalDeregistrations.Add(1)
	r.metrics.InstanceCount.Add(1)

	if r.useRedis && r.redisClient != nil {
		go r.deregisterFromRedis(id)
	}

	log.Printf("[SERVICE_DISCOVERY] Deregistered instance: %s", id)
	return nil
}

func (r *DistributedServiceRegistry) deregisterFromRedis(id string) error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	pattern := fmt.Sprintf("service:instance:*:%s", id)
	_, err := r.redisClient.Del(ctx, pattern)
	return err
}

func (r *DistributedServiceRegistry) Discover(name string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.metrics.TotalDiscoveries.Add(1)

	var result []*ServiceInstance
	for _, inst := range r.instances {
		if inst.Name == name && inst.Healthy {
			result = append(result, inst)
		}
	}

	return result
}

func (r *DistributedServiceRegistry) DiscoverByRegion(name, region string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.metrics.TotalDiscoveries.Add(1)

	var result []*ServiceInstance
	for _, inst := range r.instances {
		if inst.Name == name && inst.Healthy && inst.Region == region {
			result = append(result, inst)
		}
	}

	return result
}

func (r *DistributedServiceRegistry) DiscoverByDC(name, dc string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.metrics.TotalDiscoveries.Add(1)

	var result []*ServiceInstance
	for _, inst := range r.instances {
		if inst.Name == name && inst.Healthy && inst.DC == dc {
			result = append(result, inst)
		}
	}

	return result
}

func (r *DistributedServiceRegistry) DiscoverOptimal(name string) *ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	r.metrics.TotalDiscoveries.Add(1)

	var optimal *ServiceInstance
	minLoad := int64(math.MaxInt64)

	for _, inst := range r.instances {
		if inst.Name != name || !inst.Healthy {
			continue
		}

		load := inst.CurrentLoad
		if load < minLoad {
			minLoad = load
			optimal = inst
		}
	}

	return optimal
}

func (r *DistributedServiceRegistry) DiscoverAll(name string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	var result []*ServiceInstance
	for _, inst := range r.instances {
		if inst.Name == name {
			result = append(result, inst)
		}
	}

	return result
}

func (r *DistributedServiceRegistry) UpdateLoad(id string, load int64) error {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, exists := r.instances[id]
	if !exists {
		return fmt.Errorf("instance %s not found", id)
	}

	inst.CurrentLoad = load
	return nil
}

func (r *DistributedServiceRegistry) Heartbeat(id string) error {
	r.mu.Lock()
	defer r.mu.Unlock()

	inst, exists := r.instances[id]
	if !exists {
		return fmt.Errorf("instance %s not found", id)
	}

	inst.LastHeartbeat = time.Now()
	inst.Healthy = true
	return nil
}

func (r *DistributedServiceRegistry) GetInstance(id string) (*ServiceInstance, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	inst, exists := r.instances[id]
	return inst, exists
}

func (r *DistributedServiceRegistry) ListServices() []string {
	r.mu.RLock()
	defer r.mu.RUnlock()

	serviceNames := make(map[string]struct{})
	for _, inst := range r.instances {
		serviceNames[inst.Name] = struct{}{}
	}

	result := make([]string, 0, len(serviceNames))
	for name := range serviceNames {
		result = append(result, name)
	}

	return result
}

func (r *DistributedServiceRegistry) GetAllInstances() []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make([]*ServiceInstance, 0, len(r.instances))
	for _, inst := range r.instances {
		result = append(result, inst)
	}

	return result
}

func (r *DistributedServiceRegistry) GetMetrics() *RegistryMetrics {
	return r.metrics
}

func (r *DistributedServiceRegistry) GetStats() map[string]interface{} {
	r.mu.RLock()
	defer r.mu.RUnlock()

	serviceCount := make(map[string]int)
	healthyCount := 0
	unhealthyCount := 0

	for _, inst := range r.instances {
		serviceCount[inst.Name]++
		if inst.Healthy {
			healthyCount++
		} else {
			unhealthyCount++
		}
	}

	return map[string]interface{}{
		"total_instances":  len(r.instances),
		"healthy_count":    healthyCount,
		"unhealthy_count":  unhealthyCount,
		"service_count":    len(serviceCount),
		"services":         serviceCount,
		"total_discoveries": r.metrics.TotalDiscoveries.Load(),
		"total_registrations": r.metrics.TotalRegistrations.Load(),
	}
}

func (r *DistributedServiceRegistry) Close() error {
	if r.healthChecker != nil {
		r.healthChecker.Stop()
	}

	r.mu.Lock()
	defer r.mu.Unlock()

	for id := range r.instances {
		instanceID := fmt.Sprintf("%s:%d", r.localAddr, r.localPort)
		if id == instanceID {
			r.metrics.TotalDeregistrations.Add(1)
		}
		delete(r.instances, id)
	}

	log.Println("[SERVICE_DISCOVERY] Registry closed")
	return nil
}

type LoadBalancer struct {
	registry *DistributedServiceRegistry
	strategy LoadBalanceStrategy
}

type LoadBalanceStrategy string

const (
	StrategyRoundRobin   LoadBalanceStrategy = "round_robin"
	StrategyWeighted     LoadBalanceStrategy = "weighted"
	StrategyLeastLoad    LoadBalanceStrategy = "least_load"
	StrategyRandom       LoadBalanceStrategy = "random"
	StrategyIPHash       LoadBalanceStrategy = "ip_hash"
)

var (
	globalLoadBalancer *LoadBalancer
	lbOnce             sync.Once
)

func NewLoadBalancer(strategy LoadBalanceStrategy) *LoadBalancer {
	return &LoadBalancer{
		registry: globalRegistry,
		strategy: strategy,
	}
}

func (lb *LoadBalancer) SetRegistry(registry *DistributedServiceRegistry) {
	lb.registry = registry
}

func (lb *LoadBalancer) Select(serviceName string, clientIP string) *ServiceInstance {
	if lb.registry == nil {
		return nil
	}

	instances := lb.registry.Discover(serviceName)
	if len(instances) == 0 {
		return nil
	}

	switch lb.strategy {
	case StrategyRoundRobin:
		return lb.roundRobin(instances)
	case StrategyWeighted:
		return lb.weighted(instances)
	case StrategyLeastLoad:
		return lb.leastLoad(instances)
	case StrategyRandom:
		return lb.random(instances)
	case StrategyIPHash:
		return lb.ipHash(instances, clientIP)
	default:
		return lb.leastLoad(instances)
	}
}

func (lb *LoadBalancer) roundRobin(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	return instances[time.Now().UnixNano()%int64(len(instances))]
}

func (lb *LoadBalancer) weighted(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	totalWeight := 0
	for _, inst := range instances {
		if inst.Weight <= 0 {
			inst.Weight = 100
		}
		totalWeight += inst.Weight
	}

	if totalWeight <= 0 {
		return instances[0]
	}

	randomVal := time.Now().UnixNano() % int64(totalWeight)
	currentWeight := 0

	for _, inst := range instances {
		currentWeight += inst.Weight
		if randomVal < int64(currentWeight) {
			return inst
		}
	}

	return instances[0]
}

func (lb *LoadBalancer) leastLoad(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	var best *ServiceInstance
	minLoad := int64(math.MaxInt64)

	for _, inst := range instances {
		load := inst.CurrentLoad
		if load < minLoad {
			minLoad = load
			best = inst
		}
	}

	return best
}

func (lb *LoadBalancer) random(instances []*ServiceInstance) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}
	return instances[time.Now().UnixNano()%int64(len(instances))]
}

func (lb *LoadBalancer) ipHash(instances []*ServiceInstance, clientIP string) *ServiceInstance {
	if len(instances) == 0 {
		return nil
	}

	hash := 0
	for _, c := range clientIP {
		hash = 31*hash + int(c)
	}

	return instances[hash%len(instances)]
}

func GetLocalIP() string {
	addrs, err := net.InterfaceAddrs()
	if err != nil {
		return "127.0.0.1"
	}

	for _, addr := range addrs {
		if ipNet, ok := addr.(*net.IPNet); ok && !ipNet.IP.IsLoopback() {
			if ipNet.IP.To4() != nil {
				return ipNet.IP.String()
			}
		}
	}

	return "127.0.0.1"
}

func InitDistributedRegistry(config *ServiceDiscoveryConfig) error {
	var err error
	registryOnce.Do(func() {
		if config.ServiceAddr == "" {
			config.ServiceAddr = GetLocalIP()
		}
		globalRegistry, err = NewDistributedRegistry(config)
	})
	return err
}

func GetDistributedRegistry() *DistributedServiceRegistry {
	return globalRegistry
}

func NewLoadBalancerWithRegistry(strategy LoadBalanceStrategy) *LoadBalancer {
	lb := NewLoadBalancer(strategy)
	lb.SetRegistry(globalRegistry)
	return lb
}
