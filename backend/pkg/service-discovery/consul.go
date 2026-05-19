package servicediscovery

import (
	"fmt"
	"log"
	"sync"
	"time"

	consul "github.com/hashicorp/consul/api"
)

type ConsulService struct {
	client    *consul.Client
	serviceID string
	config    *consul.Config
	mu        sync.RWMutex
}

type ConsulConfig struct {
	Address    string
	ServiceName string
	ServiceID   string
	Host        string
	Port        int
	Tags        []string
}

func NewConsulService(config *ConsulConfig) (*ConsulService, error) {
	consulConfig := consul.DefaultConfig()
	if config.Address != "" {
		consulConfig.Address = config.Address
	}

	client, err := consul.NewClient(consulConfig)
	if err != nil {
		return nil, fmt.Errorf("failed to create consul client: %w", err)
	}

	return &ConsulService{
		client:    client,
		serviceID: config.ServiceID,
		config:    consulConfig,
	}, nil
}

func (c *ConsulService) Register(config *ConsulConfig) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	registration := &consul.AgentServiceRegistration{
		ID:      config.ServiceID,
		Name:    config.ServiceName,
		Address: config.Host,
		Port:    config.Port,
		Tags:    config.Tags,
		Check: &consul.AgentServiceCheck{
			HTTP:                           fmt.Sprintf("http://%s:%d/health", config.Host, config.Port),
			Interval:                       "10s",
			Timeout:                        "5s",
			DeregisterCriticalServiceAfter: "30s",
		},
	}

	if err := c.client.Agent().ServiceRegister(registration); err != nil {
		return fmt.Errorf("failed to register service: %w", err)
	}

	c.serviceID = config.ServiceID
	log.Printf("Service %s registered with ID %s at %s:%d", config.ServiceName, config.ServiceID, config.Host, config.Port)
	return nil
}

func (c *ConsulService) Deregister() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.serviceID == "" {
		return nil
	}

	if err := c.client.Agent().ServiceDeregister(c.serviceID); err != nil {
		return fmt.Errorf("failed to deregister service: %w", err)
	}

	log.Printf("Service %s deregistered", c.serviceID)
	c.serviceID = ""
	return nil
}

func (c *ConsulService) Discover(serviceName string) ([]*ServiceInstance, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services, _, err := c.client.Catalog().Service(serviceName, "", nil)
	if err != nil {
		return nil, fmt.Errorf("failed to discover service: %w", err)
	}

	var instances []*ServiceInstance
	for _, service := range services {
		health, _, err := c.client.Health().Service(service.ServiceName, service.ServiceID, true, nil)
		if err != nil {
			log.Printf("Warning: failed to check health for service %s: %v", service.ServiceID, err)
			continue
		}

		isHealthy := len(health) > 0 && health[0].Checks.AggregatedStatus() == consul.HealthPassing

		instances = append(instances, &ServiceInstance{
			ID:            service.ServiceID,
			Name:          service.ServiceName,
			Address:       service.Address,
			Port:          service.ServicePort,
			Healthy:       isHealthy,
			LastHeartbeat: time.Now(),
			Metadata:      make(map[string]string),
		})
	}

	return instances, nil
}

func (c *ConsulService) GetAllServices() (map[string][]string, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	services, err := c.client.Agent().Services()
	if err != nil {
		return nil, fmt.Errorf("failed to get services: %w", err)
	}

	result := make(map[string][]string)
	for id, service := range services {
		if _, ok := result[service.Service]; !ok {
			result[service.Service] = []string{}
		}
		result[service.Service] = append(result[service.Service], id)
	}

	return result, nil
}

func (c *ConsulService) WatchService(serviceName string, callback func([]*ServiceInstance)) error {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		instances, err := c.Discover(serviceName)
		if err != nil {
			log.Printf("Warning: failed to watch service %s: %v", serviceName, err)
			continue
		}
		callback(instances)
		<-ticker.C
	}
}

func (c *ConsulService) GetHealthChecks(serviceID string) ([]*consul.HealthCheck, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	checks, _, err := c.client.Agent().Checks()
	if err != nil {
		return nil, fmt.Errorf("failed to get health checks: %w", err)
	}

	var result []*consul.HealthCheck
	for _, check := range checks {
		if check.ServiceID == serviceID {
			result = append(result, check)
		}
	}

	return result, nil
}