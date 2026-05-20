package servicediscovery

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewRegistry(t *testing.T) {
	registry := NewRegistry()
	assert.NotNil(t, registry)
	assert.NotNil(t, registry.instances)
	assert.Equal(t, 0, len(registry.instances))
}

func TestRegistry_Register(t *testing.T) {
	registry := NewRegistry()

	instance := &ServiceInstance{
		ID:      "test-instance-1",
		Name:    "test-service",
		Address: "192.168.1.1",
		Port:    8080,
		Healthy: true,
		Metadata: map[string]string{
			"version": "1.0.0",
		},
	}

	registry.Register(instance)

	assert.Equal(t, 1, len(registry.instances))
	assert.NotNil(t, registry.instances["test-instance-1"])
	assert.NotZero(t, registry.instances["test-instance-1"].LastHeartbeat)
}

func TestRegistry_Register_MultipleInstances(t *testing.T) {
	registry := NewRegistry()

	instances := []*ServiceInstance{
		{
			ID:      "instance-1",
			Name:    "service-a",
			Address: "10.0.0.1",
			Port:    8080,
		},
		{
			ID:      "instance-2",
			Name:    "service-a",
			Address: "10.0.0.2",
			Port:    8080,
		},
		{
			ID:      "instance-3",
			Name:    "service-b",
			Address: "10.0.0.3",
			Port:    9090,
		},
	}

	for _, inst := range instances {
		registry.Register(inst)
	}

	assert.Equal(t, 3, len(registry.instances))
}

func TestRegistry_Deregister(t *testing.T) {
	registry := NewRegistry()

	instance := &ServiceInstance{
		ID:      "test-instance",
		Name:    "test-service",
		Address: "192.168.1.1",
		Port:    8080,
	}

	registry.Register(instance)
	assert.Equal(t, 1, len(registry.instances))

	registry.Deregister("test-instance")
	assert.Equal(t, 0, len(registry.instances))
}

func TestRegistry_Deregister_NonExistent(t *testing.T) {
	registry := NewRegistry()
	registry.Deregister("non-existent")
	assert.Equal(t, 0, len(registry.instances))
}

func TestRegistry_Discover(t *testing.T) {
	registry := NewRegistry()

	instances := []*ServiceInstance{
		{
			ID:      "healthy-1",
			Name:    "test-service",
			Address: "10.0.0.1",
			Port:    8080,
			Healthy: true,
		},
		{
			ID:      "healthy-2",
			Name:    "test-service",
			Address: "10.0.0.2",
			Port:    8080,
			Healthy: true,
		},
		{
			ID:      "unhealthy",
			Name:    "test-service",
			Address: "10.0.0.3",
			Port:    8080,
			Healthy: false,
		},
		{
			ID:      "other-healthy",
			Name:    "other-service",
			Address: "10.0.0.4",
			Port:    9090,
			Healthy: true,
		},
	}

	for _, inst := range instances {
		registry.Register(inst)
	}

	results := registry.Discover("test-service")
	assert.Equal(t, 2, len(results))
	for _, inst := range results {
		assert.Equal(t, "test-service", inst.Name)
		assert.True(t, inst.Healthy)
	}
}

func TestRegistry_Discover_NoHealthyInstances(t *testing.T) {
	registry := NewRegistry()

	instance := &ServiceInstance{
		ID:      "unhealthy-instance",
		Name:    "test-service",
		Healthy: false,
	}

	registry.Register(instance)
	results := registry.Discover("test-service")
	assert.Equal(t, 0, len(results))
}

func TestRegistry_Discover_NonExistentService(t *testing.T) {
	registry := NewRegistry()

	instance := &ServiceInstance{
		ID:   "instance-1",
		Name: "existing-service",
	}
	registry.Register(instance)

	results := registry.Discover("non-existent-service")
	assert.Equal(t, 0, len(results))
}

func TestRegistry_Heartbeat(t *testing.T) {
	registry := NewRegistry()

	instance := &ServiceInstance{
		ID:      "test-instance",
		Name:    "test-service",
		Healthy: true,
	}
	registry.Register(instance)

	time.Sleep(10 * time.Millisecond)
	oldHeartbeat := instance.LastHeartbeat

	registry.Heartbeat("test-instance")
	assert.True(t, instance.Healthy)
	assert.True(t, instance.LastHeartbeat.After(oldHeartbeat) || instance.LastHeartbeat.Equal(oldHeartbeat))
}

func TestRegistry_Heartbeat_NonExistent(t *testing.T) {
	registry := NewRegistry()
	registry.Heartbeat("non-existent")
}

func TestRegistry_ConcurrentAccess(t *testing.T) {
	registry := NewRegistry()

	instances := make([]*ServiceInstance, 100)
	for i := 0; i < 100; i++ {
		instances[i] = &ServiceInstance{
			ID:      string(rune(i)),
			Name:    "test-service",
			Address: "10.0.0.1",
			Port:    8080,
		}
	}

	for _, inst := range instances {
		registry.Register(inst)
	}

	assert.Equal(t, 100, len(registry.instances))

	for i := 0; i < 100; i++ {
		registry.Deregister(string(rune(i)))
	}

	assert.Equal(t, 0, len(registry.instances))
}

func TestServiceInstance_Structure(t *testing.T) {
	now := time.Now()
	instance := &ServiceInstance{
		ID:      "test-id",
		Name:    "test-service",
		Address: "192.168.1.1",
		Port:    8080,
		Healthy: true,
		LastHeartbeat: now,
		Metadata: map[string]string{
			"version": "1.0.0",
			"env":     "production",
		},
	}

	assert.Equal(t, "test-id", instance.ID)
	assert.Equal(t, "test-service", instance.Name)
	assert.Equal(t, "192.168.1.1", instance.Address)
	assert.Equal(t, 8080, instance.Port)
	assert.True(t, instance.Healthy)
	assert.Equal(t, now, instance.LastHeartbeat)
	assert.Equal(t, "1.0.0", instance.Metadata["version"])
	assert.Equal(t, "production", instance.Metadata["env"])
}

func TestRegistry_Discover_EmptyRegistry(t *testing.T) {
	registry := NewRegistry()
	results := registry.Discover("any-service")
	assert.Equal(t, 0, len(results))
}
