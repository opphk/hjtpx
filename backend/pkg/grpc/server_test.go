package grpc

import (
	"testing"
	"time"
)

func TestServerCreation(t *testing.T) {
	cfg := &MockGRPCConfig{
		MaxMsgSize: 4 * 1024 * 1024,
		KeepAlive:  MockKeepAliveConfig{},
		Interceptors: []string{"validation"},
	}

	server := NewServer(cfg)

	if server == nil {
		t.Fatal("Expected server to be created")
	}

	services := server.GetServices()
	if len(services) != 0 {
		t.Errorf("Expected 0 services initially, got %d", len(services))
	}
}

func TestClientCreation(t *testing.T) {
	cfg := &ClientConfig{
		Address:     "localhost:9090",
		DialTimeout: 5 * time.Second,
	}

	client, err := NewClient(cfg)

	if err != nil {
		t.Fatalf("Failed to create client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client to be created")
	}
}

func TestClientPool(t *testing.T) {
	factory := func(address string) (*Client, error) {
		return &Client{}, nil
	}

	pool := NewClientPool(factory, 3)

	if pool == nil {
		t.Fatal("Expected pool to be created")
	}

	client, err := pool.GetClient("localhost:9091")
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if client == nil {
		t.Fatal("Expected client from pool")
	}

	err = pool.Close()
	if err != nil {
		t.Fatalf("Failed to close pool: %v", err)
	}
}

func TestBalancer(t *testing.T) {
	clients := []*Client{
		{},
		{},
		{},
	}

	balancer := NewBalancer(clients, RoundRobin)

	if balancer == nil {
		t.Fatal("Expected balancer to be created")
	}

	client1, err := balancer.GetClient()
	if err != nil {
		t.Fatalf("Failed to get client: %v", err)
	}

	if client1 == nil {
		t.Fatal("Expected client from balancer")
	}

	client2, _ := balancer.GetClient()
	if client2 == nil {
		t.Fatal("Expected client from balancer")
	}

	client3, _ := balancer.GetClient()
	if client3 == nil {
		t.Fatal("Expected client from balancer")
	}

	client4, _ := balancer.GetClient()
	if client4 == client1 {
		t.Error("Expected different client after round robin")
	}
}

func TestBalancerEmpty(t *testing.T) {
	balancer := NewBalancer([]*Client{}, RoundRobin)

	_, err := balancer.GetClient()
	if err == nil {
		t.Error("Expected error when no clients available")
	}
}

func TestServiceRegistry(t *testing.T) {
	registry := NewServiceRegistry()

	if registry == nil {
		t.Fatal("Expected registry to be created")
	}

	svc := &MockService{name: "test-service"}
	registry.Register("test", svc)

	retrieved, ok := registry.Get("test")
	if !ok {
		t.Error("Expected service to be retrieved")
	}

	if retrieved == nil {
		t.Error("Expected non-nil service")
	}

	all := registry.GetAll()
	if len(all) != 1 {
		t.Errorf("Expected 1 service, got %d", len(all))
	}

	registry.Unregister("test")
	_, ok = registry.Get("test")
	if ok {
		t.Error("Expected service to be unregistered")
	}
}

func TestGetServiceNameFromMethod(t *testing.T) {
	tests := []struct {
		method    string
		expected  string
	}{
		{"/hjtpx.CaptchaService/Generate", "hjtpx.CaptchaService"},
		{"/hjtpx.BehaviorService/Analyze", "hjtpx.BehaviorService"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := GetServiceNameFromMethod(tt.method)
		if result != tt.expected {
			t.Errorf("GetServiceNameFromMethod(%s) = %s, want %s", tt.method, result, tt.expected)
		}
	}
}

func TestGetMethodNameFromMethod(t *testing.T) {
	tests := []struct {
		method    string
		expected  string
	}{
		{"/hjtpx.CaptchaService/Generate", "Generate"},
		{"/hjtpx.BehaviorService/Analyze", "Analyze"},
		{"invalid", ""},
	}

	for _, tt := range tests {
		result := GetMethodNameFromMethod(tt.method)
		if result != tt.expected {
			t.Errorf("GetMethodNameFromMethod(%s) = %s, want %s", tt.method, result, tt.expected)
		}
	}
}

type MockGRPCConfig struct {
	MaxMsgSize   int
	KeepAlive    MockKeepAliveConfig
	TLS          MockTLSConfig
	Interceptors []string
}

type MockKeepAliveConfig struct {
	MaxConnectionIdle     time.Duration
	MaxConnectionAge       time.Duration
	MaxConnectionAgeGrace  time.Duration
	Time                  time.Duration
	Timeout               time.Duration
}

type MockTLSConfig struct {
	Enabled  bool
	CertFile string
	KeyFile  string
}

type MockService struct {
	name string
}

func (m *MockService) Register(server *Server) {
}

func init() {
}
