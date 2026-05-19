package microservice

import (
	"fmt"
	"log"
	"sync"

	"github.com/hjtpx/hjtpx/pkg/circuitbreaker"
	"github.com/hjtpx/hjtpx/pkg/service-discovery"
	"github.com/hjtpx/hjtpx/pb/application"
	"github.com/hjtpx/hjtpx/pb/auth"
	"github.com/hjtpx/hjtpx/pb/captcha"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type Client struct {
	authClient        auth.AuthServiceClient
	captchaClient     captcha.CaptchaServiceClient
	applicationClient application.ApplicationServiceClient
	cbManager         *circuitbreaker.CircuitBreakerManager
	consulService     *servicediscovery.ConsulService
	mu                sync.RWMutex
}

type ClientConfig struct {
	ConsulAddress string
}

func NewClient(config *ClientConfig) (*Client, error) {
	consulConfig := &servicediscovery.ConsulConfig{
		Address: config.ConsulAddress,
	}

	consulService, err := servicediscovery.NewConsulService(consulConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to Consul: %v", err)
	}

	return &Client{
		cbManager:     circuitbreaker.NewCircuitBreakerManager(),
		consulService: consulService,
	}, nil
}

func (c *Client) GetAuthClient() (auth.AuthServiceClient, error) {
	c.mu.RLock()
	if c.authClient != nil {
		c.mu.RUnlock()
		return c.authClient, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.authClient != nil {
		return c.authClient, nil
	}

	conn, err := c.dialService("auth-service")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to auth-service: %w", err)
	}

	c.authClient = auth.NewAuthServiceClient(conn)
	return c.authClient, nil
}

func (c *Client) GetCaptchaClient() (captcha.CaptchaServiceClient, error) {
	c.mu.RLock()
	if c.captchaClient != nil {
		c.mu.RUnlock()
		return c.captchaClient, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.captchaClient != nil {
		return c.captchaClient, nil
	}

	conn, err := c.dialService("captcha-service")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to captcha-service: %w", err)
	}

	c.captchaClient = captcha.NewCaptchaServiceClient(conn)
	return c.captchaClient, nil
}

func (c *Client) GetApplicationClient() (application.ApplicationServiceClient, error) {
	c.mu.RLock()
	if c.applicationClient != nil {
		c.mu.RUnlock()
		return c.applicationClient, nil
	}
	c.mu.RUnlock()

	c.mu.Lock()
	defer c.mu.Unlock()

	if c.applicationClient != nil {
		return c.applicationClient, nil
	}

	conn, err := c.dialService("application-service")
	if err != nil {
		return nil, fmt.Errorf("failed to connect to application-service: %w", err)
	}

	c.applicationClient = application.NewApplicationServiceClient(conn)
	return c.applicationClient, nil
}

func (c *Client) dialService(serviceName string) (*grpc.ClientConn, error) {
	if c.consulService != nil {
		instances, err := c.consulService.Discover(serviceName)
		if err != nil {
			log.Printf("Warning: Failed to discover service %s via Consul: %v", serviceName, err)
		} else if len(instances) > 0 {
			instance := servicediscovery.RandomInstance(instances)
			if instance != nil {
				target := fmt.Sprintf("%s:%d", instance.Address, instance.Port)
				log.Printf("Connecting to %s at %s", serviceName, target)
				return grpc.NewClient(
					target,
					grpc.WithTransportCredentials(insecure.NewCredentials()),
					grpc.WithUnaryInterceptor(circuitbreaker.UnaryClientInterceptor(c.cbManager, serviceName)),
				)
			}
		}
	}

	defaultPorts := map[string]string{
		"auth-service":        "localhost:50051",
		"captcha-service":     "localhost:50052",
		"application-service": "localhost:50053",
	}

	target := defaultPorts[serviceName]
	if target == "" {
		target = fmt.Sprintf("localhost:50050")
	}

	log.Printf("Using default address %s for %s (Consul not available)", target, serviceName)
	return grpc.NewClient(
		target,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithUnaryInterceptor(circuitbreaker.UnaryClientInterceptor(c.cbManager, serviceName)),
	)
}

func (c *Client) Close() {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Note: gRPC connections are managed internally and will be closed when the client is garbage collected
	// For proper cleanup, we'd need to store the connections
	log.Println("Microservice client closed")
}

var (
	client     *Client
	clientOnce sync.Once
)

func GetClient(config *ClientConfig) (*Client, error) {
	var err error
	clientOnce.Do(func() {
		client, err = NewClient(config)
	})
	if err != nil {
		return nil, err
	}
	return client, nil
}