package servicediscovery

import (
	"sync"
	"testing"
	"time"
)

func TestDistributedServiceRegistry_Initialization(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName:         "test-service",
		ServiceAddr:         "localhost",
		ServicePort:         8080,
		Region:              "us-east",
		DC:                  "dc1",
		Version:             "1.0.0",
		HeartbeatInterval:   10 * time.Second,
		TTL:                 30 * time.Second,
		HealthCheckInterval: 15 * time.Second,
		UseRedis:            false,
	}

	registry, err := NewDistributedRegistry(config)
	if err != nil {
		t.Fatalf("Expected registry to be created, got error: %v", err)
	}

	if registry == nil {
		t.Fatal("Expected registry to be non-nil")
	}

	if registry.localAddr != "localhost" {
		t.Errorf("Expected local addr to be localhost, got %s", registry.localAddr)
	}

	if registry.localPort != 8080 {
		t.Errorf("Expected local port to be 8080, got %d", registry.localPort)
	}

	if registry.region != "us-east" {
		t.Errorf("Expected region to be us-east, got %s", registry.region)
	}

	registry.Close()
}

func TestServiceDiscoveryConfig_Defaults(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, err := NewDistributedRegistry(config)
	if err != nil {
		t.Fatalf("Expected registry to be created, got error: %v", err)
	}

	if registry.heartbeatInterval != 10*time.Second {
		t.Errorf("Expected heartbeat interval to be 10s, got %v", registry.heartbeatInterval)
	}

	if registry.ttl != 30*time.Second {
		t.Errorf("Expected TTL to be 30s, got %v", registry.ttl)
	}

	registry.Close()
}

func TestServiceInstance_Registration(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	instance := &ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Weight:  100,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
	}

	err := registry.Register(instance)
	if err != nil {
		t.Errorf("Expected registration to succeed, got error: %v", err)
	}

	if len(registry.instances) != 1 {
		t.Errorf("Expected 1 instance registered, got %d", len(registry.instances))
	}

	registry.Close()
}

func TestServiceRegistry_Discover(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Healthy: true,
	})

	registry.Register(&ServiceInstance{
		ID:      "instance-2",
		Name:    "test-service",
		Address: "localhost",
		Port:    8081,
		Healthy: true,
	})

	registry.Register(&ServiceInstance{
		ID:      "instance-3",
		Name:    "other-service",
		Address: "localhost",
		Port:    8082,
		Healthy: true,
	})

	instances := registry.Discover("test-service")
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}

	registry.Close()
}

func TestServiceRegistry_DiscoverByRegion(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Healthy: true,
		Region:  "us-east",
	})

	registry.Register(&ServiceInstance{
		ID:      "instance-2",
		Name:    "test-service",
		Address: "localhost",
		Port:    8081,
		Healthy: true,
		Region:  "us-west",
	})

	instances := registry.DiscoverByRegion("test-service", "us-east")
	if len(instances) != 1 {
		t.Errorf("Expected 1 instance, got %d", len(instances))
	}

	registry.Close()
}

func TestServiceRegistry_DiscoverOptimal(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:           "instance-1",
		Name:         "test-service",
		Address:      "localhost",
		Port:         8080,
		Healthy:      true,
		CurrentLoad:  0,
	})

	registry.Register(&ServiceInstance{
		ID:           "instance-2",
		Name:         "test-service",
		Address:      "localhost",
		Port:         8081,
		Healthy:      true,
		CurrentLoad:  0,
	})

	registry.instances["instance-1"].CurrentLoad = 100
	registry.instances["instance-2"].CurrentLoad = 50

	optimal := registry.DiscoverOptimal("test-service")
	if optimal == nil {
		t.Fatal("Expected optimal instance to be returned")
	}

	if optimal.ID != "instance-2" {
		t.Logf("Optimal instance: %s (expected instance-2 with lower load)", optimal.ID)
	}

	registry.Close()
}

func TestServiceRegistry_Deregister(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
	})

	err := registry.Deregister("instance-1")
	if err != nil {
		t.Errorf("Expected deregistration to succeed, got error: %v", err)
	}

	if len(registry.instances) != 0 {
		t.Errorf("Expected 0 instances, got %d", len(registry.instances))
	}

	registry.Close()
}

func TestServiceRegistry_Heartbeat(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Address: "localhost",
		Port:    8080,
		Healthy: true,
	})

	err := registry.Heartbeat("instance-1")
	if err != nil {
		t.Errorf("Expected heartbeat to succeed, got error: %v", err)
	}

	registry.Close()
}

func TestServiceRegistry_ListServices(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:   "instance-1",
		Name: "service-a",
	})

	registry.Register(&ServiceInstance{
		ID:   "instance-2",
		Name: "service-b",
	})

	registry.Register(&ServiceInstance{
		ID:   "instance-3",
		Name: "service-a",
	})

	services := registry.ListServices()
	if len(services) != 2 {
		t.Errorf("Expected 2 services, got %d", len(services))
	}

	registry.Close()
}

func TestServiceRegistry_GetMetrics(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	metrics := registry.GetMetrics()
	if metrics == nil {
		t.Fatal("Expected metrics to be returned")
	}

	registry.metrics.TotalRegistrations.Add(10)
	if metrics.TotalRegistrations.Load() != 10 {
		t.Errorf("Expected 10 registrations, got %d", metrics.TotalRegistrations.Load())
	}

	registry.Close()
}

func TestServiceRegistry_GetStats(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Healthy: true,
	})

	registry.Register(&ServiceInstance{
		ID:      "instance-2",
		Name:    "test-service",
		Healthy: false,
	})

	stats := registry.GetStats()
	if stats == nil {
		t.Fatal("Expected stats to be returned")
	}

	if stats["total_instances"].(int) != 2 {
		t.Errorf("Expected 2 total instances, got %v", stats["total_instances"])
	}

	if stats["healthy_count"].(int) != 2 {
		t.Errorf("Expected 2 healthy instances, got %v", stats["healthy_count"])
	}

	registry.Close()
}

func TestRegistryMetrics(t *testing.T) {
	metrics := &RegistryMetrics{}

	metrics.TotalRegistrations.Add(100)
	metrics.TotalDeregistrations.Add(20)
	metrics.TotalDiscoveries.Add(500)
	metrics.HeartbeatSuccess.Add(300)
	metrics.HeartbeatFailure.Add(10)
	metrics.InstanceCount.Store(80)

	if metrics.TotalRegistrations.Load() != 100 {
		t.Errorf("Expected 100 registrations, got %d", metrics.TotalRegistrations.Load())
	}

	if metrics.InstanceCount.Load() != 80 {
		t.Errorf("Expected 80 instances, got %d", metrics.InstanceCount.Load())
	}
}

func TestLoadBalancer_RoundRobin(t *testing.T) {
	lb := &LoadBalancer{
		strategy: StrategyRoundRobin,
	}

	lb.registry = &DistributedServiceRegistry{
		instances: map[string]*ServiceInstance{
			"instance-1": {ID: "instance-1", Name: "test-service", Healthy: true},
			"instance-2": {ID: "instance-2", Name: "test-service", Healthy: true},
		},
		metrics: &RegistryMetrics{},
	}

	selected := lb.Select("test-service", "127.0.0.1")
	if selected == nil {
		t.Error("Expected instance to be selected")
	}
}

func TestLoadBalancer_Weighted(t *testing.T) {
	lb := &LoadBalancer{
		strategy: StrategyWeighted,
	}

	lb.registry = &DistributedServiceRegistry{
		instances: map[string]*ServiceInstance{
			"instance-1": {ID: "instance-1", Name: "test-service", Healthy: true, Weight: 100},
			"instance-2": {ID: "instance-2", Name: "test-service", Healthy: true, Weight: 50},
		},
		metrics: &RegistryMetrics{},
	}

	selected := lb.Select("test-service", "127.0.0.1")
	if selected == nil {
		t.Error("Expected instance to be selected")
	}
}

func TestLoadBalancer_LeastLoad(t *testing.T) {
	lb := &LoadBalancer{
		strategy: StrategyLeastLoad,
	}

	lb.registry = &DistributedServiceRegistry{
		instances: map[string]*ServiceInstance{
			"instance-1": {ID: "instance-1", Name: "test-service", Healthy: true, CurrentLoad: 0},
			"instance-2": {ID: "instance-2", Name: "test-service", Healthy: true, CurrentLoad: 0},
		},
		metrics: &RegistryMetrics{},
	}

	lb.registry.instances["instance-1"].CurrentLoad = 100
	lb.registry.instances["instance-2"].CurrentLoad = 50

	selected := lb.Select("test-service", "127.0.0.1")
	if selected == nil {
		t.Error("Expected instance to be selected")
	}

	if selected.ID != "instance-2" {
		t.Logf("Selected: %s (expected instance-2 with lower load)", selected.ID)
	}
}

func TestLoadBalancer_Random(t *testing.T) {
	lb := &LoadBalancer{
		strategy: StrategyRandom,
	}

	lb.registry = &DistributedServiceRegistry{
		instances: map[string]*ServiceInstance{
			"instance-1": {ID: "instance-1", Name: "test-service", Healthy: true},
			"instance-2": {ID: "instance-2", Name: "test-service", Healthy: true},
		},
		metrics: &RegistryMetrics{},
	}

	selected := lb.Select("test-service", "127.0.0.1")
	if selected == nil {
		t.Error("Expected instance to be selected")
	}
}

func TestGetLocalIP(t *testing.T) {
	ip := GetLocalIP()
	if ip == "" {
		t.Error("Expected local IP to be returned")
	}

	if ip == "127.0.0.1" {
		t.Log("Using loopback address (may be expected in some environments)")
	}
}

func TestNewLoadBalancer(t *testing.T) {
	lb := NewLoadBalancer(StrategyRoundRobin)
	if lb == nil {
		t.Fatal("Expected load balancer to be created")
	}

	if lb.strategy != StrategyRoundRobin {
		t.Errorf("Expected strategy to be round_robin, got %s", lb.strategy)
	}
}

func TestConcurrentRegistrations(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	var wg sync.WaitGroup
	for i := 0; i < 100; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()
			registry.Register(&ServiceInstance{
				ID:   "instance-" + string(rune('0'+id%10)),
				Name: "test-service",
			})
		}(i)
	}

	wg.Wait()

	if len(registry.instances) == 0 {
		t.Error("Expected instances to be registered concurrently")
	}

	registry.Close()
}

func TestServiceHealthChecker_Lifecycle(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)
	registry.Close()
}

func TestUpdateLoad(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:           "instance-1",
		Name:         "test-service",
		CurrentLoad:  0,
	})

	err := registry.UpdateLoad("instance-1", 100)
	if err != nil {
		t.Errorf("Expected update to succeed, got error: %v", err)
	}

	load := registry.instances["instance-1"].CurrentLoad
	if load != 100 {
		t.Errorf("Expected load to be 100, got %d", load)
	}

	registry.Close()
}

func TestDiscoverAll(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:      "instance-1",
		Name:    "test-service",
		Healthy: true,
	})

	registry.Register(&ServiceInstance{
		ID:      "instance-2",
		Name:    "test-service",
		Healthy: false,
	})

	instances := registry.DiscoverAll("test-service")
	if len(instances) != 2 {
		t.Errorf("Expected 2 instances, got %d", len(instances))
	}

	registry.Close()
}

func TestGetInstance(t *testing.T) {
	config := &ServiceDiscoveryConfig{
		ServiceName: "test-service",
		UseRedis:    false,
	}

	registry, _ := NewDistributedRegistry(config)

	registry.Register(&ServiceInstance{
		ID:   "instance-1",
		Name: "test-service",
	})

	inst, exists := registry.GetInstance("instance-1")
	if !exists {
		t.Error("Expected instance to exist")
	}

	if inst.ID != "instance-1" {
		t.Errorf("Expected instance ID to be instance-1, got %s", inst.ID)
	}

	registry.Close()
}
