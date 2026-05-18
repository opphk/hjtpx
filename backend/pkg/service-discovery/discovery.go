package servicediscovery

import (
	"sync"
	"time"
)

// ServiceInstance 服务实例
type ServiceInstance struct {
	ID            string            `json:"id"`
	Name          string            `json:"name"`
	Address       string            `json:"address"`
	Port          int               `json:"port"`
	Healthy       bool              `json:"healthy"`
	LastHeartbeat time.Time         `json:"last_heartbeat"`
	Metadata      map[string]string `json:"metadata"`
	RegisteredAt  time.Time         `json:"registered_at"`
	Weight        int               `json:"weight"`
	Region        string            `json:"region"`
	DC            string            `json:"dc"`
	Version       string            `json:"version"`
	Priority      int               `json:"priority"`
	Capacity      int               `json:"capacity"`
	CurrentLoad   int64             `json:"current_load"`
}

// Registry 服务注册器
type Registry struct {
	mu        sync.RWMutex
	instances map[string]*ServiceInstance
}

// NewRegistry 创建服务注册器
func NewRegistry() *Registry {
	return &Registry{
		instances: make(map[string]*ServiceInstance),
	}
}

// Register 注册服务实例
func (r *Registry) Register(instance *ServiceInstance) {
	r.mu.Lock()
	defer r.mu.Unlock()
	instance.LastHeartbeat = time.Now()
	r.instances[instance.ID] = instance
}

// Deregister 注销服务实例
func (r *Registry) Deregister(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.instances, id)
}

// Discover 发现服务实例
func (r *Registry) Discover(name string) []*ServiceInstance {
	r.mu.RLock()
	defer r.mu.RUnlock()
	var result []*ServiceInstance
	for _, inst := range r.instances {
		if inst.Name == name && inst.Healthy {
			result = append(result, inst)
		}
	}
	return result
}

// Heartbeat 心跳
func (r *Registry) Heartbeat(id string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	if inst, ok := r.instances[id]; ok {
		inst.LastHeartbeat = time.Now()
		inst.Healthy = true
	}
}

// HealthCheck 健康检查
func (r *Registry) HealthCheck() {
	r.mu.Lock()
	defer r.mu.Unlock()
	now := time.Now()
	for _, inst := range r.instances {
		if now.Sub(inst.LastHeartbeat) > 30*time.Second {
			inst.Healthy = false
		}
	}
}
