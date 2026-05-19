package grpc

import (
	"context"
	"crypto/tls"
	"crypto/x509"
	"fmt"
	"log"
	"math/rand"
	"net"
	"os"
	"strings"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/circuitbreaker"
	"github.com/hjtpx/hjtpx/pkg/config"
	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/health"
	"google.golang.org/grpc/health/grpc_health_v1"
	"google.golang.org/grpc/keepalive"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/peer"
	"google.golang.org/grpc/reflection"
)

type Server struct {
	server   *grpc.Server
	config   *config.GRPCConfig
	mu       sync.RWMutex
	services map[string]bool
}

type ServerOption func(*Server)

func NewServer(cfg *config.GRPCConfig, opts ...ServerOption) *Server {
	s := &Server{
		config:   cfg,
		services: make(map[string]bool),
	}

	var serverOpts []grpc.ServerOption

	serverOpts = append(serverOpts, grpc.MaxRecvMsgSize(cfg.MaxMsgSize))
	serverOpts = append(serverOpts, grpc.MaxSendMsgSize(cfg.MaxMsgSize))

	if cfg.TLS.Enabled {
		creds, err := loadTLSCredentials(cfg.TLS.CertFile, cfg.TLS.KeyFile)
		if err != nil {
			log.Printf("Warning: Failed to load TLS credentials: %v, using insecure", err)
			serverOpts = append(serverOpts, grpc.Creds(insecure.NewCredentials()))
		} else {
			serverOpts = append(serverOpts, grpc.Creds(creds))
		}
	} else {
		serverOpts = append(serverOpts, grpc.Creds(insecure.NewCredentials()))
	}

	ka := keepalive.ServerParameters{
		MaxConnectionIdle:     cfg.KeepAlive.MaxConnectionIdle,
		MaxConnectionAge:       cfg.KeepAlive.MaxConnectionAge,
		MaxConnectionAgeGrace:  cfg.KeepAlive.MaxConnectionAgeGrace,
		Time:                  cfg.KeepAlive.Time,
		Timeout:               cfg.KeepAlive.Timeout,
	}
	serverOpts = append(serverOpts, grpc.KeepaliveParams(ka))

	serverOpts = append(serverOpts,
		grpc.ChainUnaryInterceptor(s.unaryInterceptors()...),
		grpc.ChainStreamInterceptor(s.streamInterceptors()...),
	)

	s.server = grpc.NewServer(serverOpts...)

	grpc_health_v1.RegisterHealthServer(s.server, health.NewServer())
	reflection.Register(s.server)

	return s
}

func (s *Server) unaryInterceptors() []grpc.UnaryServerInterceptor {
	interceptors := []grpc.UnaryServerInterceptor{
		s.recoveryInterceptor,
		s.loggingInterceptor,
		s.metadataInterceptor,
	}

	if s.config.Interceptors != nil {
		for _, name := range s.config.Interceptors {
			switch name {
			case "validation":
				interceptors = append(interceptors, s.validationInterceptor)
			case "circuit_breaker":
				interceptors = append(interceptors, s.circuitBreakerInterceptor)
			case "rate_limit":
				interceptors = append(interceptors, s.rateLimitInterceptor)
			}
		}
	}

	return interceptors
}

func (s *Server) streamInterceptors() []grpc.StreamServerInterceptor {
	return []grpc.StreamServerInterceptor{
		s.recoveryStreamInterceptor,
		s.loggingStreamInterceptor,
	}
}

func (s *Server) recoveryInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in unary RPC: %v", r)
		}
	}()
	return handler(ctx, req)
}

func (s *Server) loggingInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	start := time.Now()

	resp, err := handler(ctx, req)

	duration := time.Since(start)

	if p, ok := peer.FromContext(ctx); ok {
		log.Printf("[gRPC] %s %s from %s took %v error=%v",
			info.FullMethod, "unary", p.Addr.String(), duration, err)
	}

	return resp, err
}

func (s *Server) metadataInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	md, ok := metadata.FromIncomingContext(ctx)
	if ok {
		traceID := md.Get("x-trace-id")
		if len(traceID) > 0 {
			ctx = context.WithValue(ctx, "trace_id", traceID[0])
		}

		requestID := md.Get("x-request-id")
		if len(requestID) > 0 {
			ctx = context.WithValue(ctx, "request_id", requestID[0])
		}
	}
	return handler(ctx, req)
}

func (s *Server) validationInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func (s *Server) circuitBreakerInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func (s *Server) rateLimitInterceptor(ctx context.Context, req interface{}, info *grpc.UnaryServerInfo, handler grpc.UnaryHandler) (interface{}, error) {
	return handler(ctx, req)
}

func (s *Server) recoveryStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	defer func() {
		if r := recover(); r != nil {
			log.Printf("Panic recovered in stream RPC: %v", r)
		}
	}()
	return handler(srv, ss)
}

func (s *Server) loggingStreamInterceptor(srv interface{}, ss grpc.ServerStream, info *grpc.StreamServerInfo, handler grpc.StreamHandler) error {
	start := time.Now()
	err := handler(srv, ss)
	duration := time.Since(start)
	log.Printf("[gRPC] %s stream took %v error=%v", info.FullMethod, duration, err)
	return err
}

func (s *Server) RegisterService(desc *grpc.ServiceDesc, srv interface{}) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.server.RegisterService(desc, srv)
	s.services[desc.ServiceName] = true
}

func (s *Server) Serve(lis net.Listener) error {
	log.Printf("gRPC server starting on %s", lis.Addr().String())
	return s.server.Serve(lis)
}

func (s *Server) GracefulStop() {
	log.Println("gRPC server graceful stop")
	s.server.GracefulStop()
}

func (s *Server) Stop() {
	log.Println("gRPC server stopping")
	s.server.Stop()
}

func (s *Server) GetServices() []string {
	s.mu.RLock()
	defer s.mu.RUnlock()

	services := make([]string, 0, len(s.services))
	for name := range s.services {
		services = append(services, name)
	}
	return services
}

func loadTLSCredentials(certFile, keyFile string) (credentials.TransportCredentials, error) {
	certificate, err := tls.LoadX509KeyPair(certFile, keyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	ca, err := os.ReadFile(certFile)
	if err != nil {
		return nil, fmt.Errorf("failed to read CA certificate: %w", err)
	}

	if ok := certPool.AppendCertsFromPEM(ca); !ok {
		return nil, fmt.Errorf("failed to append CA certificate")
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{certificate},
		ClientAuth:   tls.NoClientCert,
		MinVersion:   tls.VersionTLS12,
		ServerName:   "",
	}), nil
}

type Client struct {
	conn   *grpc.ClientConn
	config *ClientConfig
	mu     sync.RWMutex
}

type ClientConfig struct {
	Address        string
	DialTimeout    time.Duration
	MaxRetries     int
	RetryInterval  time.Duration
	TLS            *TLSConfig
	KeepAlive      *KeepAliveConfig
	CircuitBreaker *circuitbreaker.Options
}

type TLSConfig struct {
	CertFile   string
	KeyFile    string
	CAFile     string
	ServerName string
}

type KeepAliveConfig struct {
	Time    time.Duration
	Timeout time.Duration
}

func NewClient(cfg *ClientConfig) (*Client, error) {
	var opts []grpc.DialOption

	opts = append(opts, grpc.WithDefaultCallOptions(
		grpc.MaxCallRecvMsgSize(4*1024*1024),
		grpc.MaxCallSendMsgSize(4*1024*1024),
	))

	if cfg.DialTimeout > 0 {
		opts = append(opts, grpc.WithTimeout(cfg.DialTimeout))
	}

	if cfg.TLS != nil && cfg.TLS.CertFile != "" {
		creds, err := loadClientTLSCredentials(cfg.TLS)
		if err != nil {
			log.Printf("Warning: Failed to load client TLS credentials: %v", err)
		} else {
			opts = append(opts, grpc.WithTransportCredentials(creds))
		}
	} else {
		opts = append(opts, grpc.WithTransportCredentials(insecure.NewCredentials()))
	}

	if cfg.KeepAlive != nil {
		ka := keepalive.ClientParameters{
			Time:    cfg.KeepAlive.Time,
			Timeout: cfg.KeepAlive.Timeout,
		}
		opts = append(opts, grpc.WithKeepaliveParams(ka))
	}

	opts = append(opts, grpc.WithUnaryInterceptor(clientUnaryInterceptor()))
	opts = append(opts, grpc.WithStreamInterceptor(clientStreamInterceptor()))

	ctx, cancel := context.WithTimeout(context.Background(), cfg.DialTimeout)
	defer cancel()

	conn, err := grpc.DialContext(ctx, cfg.Address, opts...)
	if err != nil {
		return nil, fmt.Errorf("failed to dial gRPC server: %w", err)
	}

	return &Client{
		conn:   conn,
		config: cfg,
	}, nil
}

func (c *Client) GetConn() *grpc.ClientConn {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.conn
}

func (c *Client) Close() error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.conn != nil {
		return c.conn.Close()
	}
	return nil
}

func (c *Client) Invoke(ctx context.Context, method string, args, reply interface{}, opts ...grpc.CallOption) error {
	return c.conn.Invoke(ctx, method, args, reply, opts...)
}

func clientUnaryInterceptor() grpc.UnaryClientInterceptor {
	return func(ctx context.Context, method string, req, reply interface{}, cc *grpc.ClientConn, invoker grpc.UnaryInvoker, opts ...grpc.CallOption) error {
		start := time.Now()

		startMD, _ := metadata.FromOutgoingContext(ctx)
		ctx = metadata.NewOutgoingContext(ctx, startMD)

		err := invoker(ctx, method, req, reply, cc, opts...)

		duration := time.Since(start)
		log.Printf("[gRPC Client] %s took %v error=%v", method, duration, err)

		return err
	}
}

func clientStreamInterceptor() grpc.StreamClientInterceptor {
	return func(ctx context.Context, desc *grpc.StreamDesc, cc *grpc.ClientConn, method string, streamer grpc.Streamer, opts ...grpc.CallOption) (grpc.ClientStream, error) {
		start := time.Now()

		startMD, _ := metadata.FromOutgoingContext(ctx)
		ctx = metadata.NewOutgoingContext(ctx, startMD)

		stream, err := streamer(ctx, desc, cc, method, opts...)

		duration := time.Since(start)
		log.Printf("[gRPC Client] %s stream took %v error=%v", method, duration, err)

		return stream, err
	}
}

func loadClientTLSCredentials(cfg *TLSConfig) (credentials.TransportCredentials, error) {
	certificate, err := tls.LoadX509KeyPair(cfg.CertFile, cfg.KeyFile)
	if err != nil {
		return nil, fmt.Errorf("failed to load client certificate: %w", err)
	}

	certPool := x509.NewCertPool()
	if cfg.CAFile != "" {
		ca, err := os.ReadFile(cfg.CAFile)
		if err != nil {
			return nil, fmt.Errorf("failed to read CA certificate: %w", err)
		}

		if ok := certPool.AppendCertsFromPEM(ca); !ok {
			return nil, fmt.Errorf("failed to append CA certificate")
		}
	}

	return credentials.NewTLS(&tls.Config{
		Certificates: []tls.Certificate{certificate},
		RootCAs:      certPool,
		ServerName:   cfg.ServerName,
		MinVersion:   tls.VersionTLS12,
	}), nil
}

type ClientPool struct {
	clients  map[string]*Client
	factory  func(string) (*Client, error)
	mu       sync.RWMutex
	maxConns int
}

func NewClientPool(factory func(string) (*Client, error), maxConns int) *ClientPool {
	return &ClientPool{
		clients:  make(map[string]*Client),
		factory:  factory,
		maxConns: maxConns,
	}
}

func (p *ClientPool) GetClient(address string) (*Client, error) {
	p.mu.RLock()
	client, exists := p.clients[address]
	p.mu.RUnlock()

	if exists {
		return client, nil
	}

	p.mu.Lock()
	defer p.mu.Unlock()

	if client, exists = p.clients[address]; exists {
		return client, nil
	}

	if len(p.clients) >= p.maxConns {
		oldest := ""
		for addr := range p.clients {
			oldest = addr
			break
		}
		if oldest != "" {
			p.clients[oldest].Close()
			delete(p.clients, oldest)
		}
	}

	newClient, err := p.factory(address)
	if err != nil {
		return nil, err
	}

	p.clients[address] = newClient
	return newClient, nil
}

func (p *ClientPool) Close() error {
	p.mu.Lock()
	defer p.mu.Unlock()

	var lastErr error
	for address, client := range p.clients {
		if err := client.Close(); err != nil {
			lastErr = err
			log.Printf("Error closing client for %s: %v", address, err)
		}
	}

	p.clients = make(map[string]*Client)
	return lastErr
}

type Balancer struct {
	clients  []*Client
	mu       sync.RWMutex
	strategy BalanceStrategy
	current  int
}

type BalanceStrategy string

const (
	RoundRobin  BalanceStrategy = "round_robin"
	Random      BalanceStrategy = "random"
	LeastConns  BalanceStrategy = "least_conns"
)

func NewBalancer(clients []*Client, strategy BalanceStrategy) *Balancer {
	return &Balancer{
		clients:  clients,
		strategy: strategy,
		current:  0,
	}
}

func (b *Balancer) GetClient() (*Client, error) {
	b.mu.Lock()
	defer b.mu.Unlock()

	if len(b.clients) == 0 {
		return nil, fmt.Errorf("no clients available")
	}

	switch b.strategy {
	case RoundRobin:
		client := b.clients[b.current]
		b.current = (b.current + 1) % len(b.clients)
		return client, nil
	case Random:
		return b.clients[rand.Intn(len(b.clients))], nil
	default:
		return b.clients[0], nil
	}
}

func GetServiceNameFromMethod(method string) string {
	parts := strings.Split(strings.TrimPrefix(method, "/"), "/")
	if len(parts) > 0 {
		return parts[0]
	}
	return ""
}

func GetMethodNameFromMethod(method string) string {
	parts := strings.Split(strings.TrimPrefix(method, "/"), "/")
	if len(parts) > 1 {
		return parts[1]
	}
	return ""
}

type Service interface {
	Register(server *Server)
}

type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]Service
}

func NewServiceRegistry() *ServiceRegistry {
	return &ServiceRegistry{
		services: make(map[string]Service),
	}
}

func (r *ServiceRegistry) Register(name string, svc Service) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.services[name] = svc
}

func (r *ServiceRegistry) Get(name string) (Service, bool) {
	r.mu.RLock()
	defer r.mu.RUnlock()
	svc, ok := r.services[name]
	return svc, ok
}

func (r *ServiceRegistry) GetAll() map[string]Service {
	r.mu.RLock()
	defer r.mu.RUnlock()

	result := make(map[string]Service, len(r.services))
	for name, svc := range r.services {
		result[name] = svc
	}
	return result
}

func (r *ServiceRegistry) Unregister(name string) {
	r.mu.Lock()
	defer r.mu.Unlock()
	delete(r.services, name)
}

func (r *ServiceRegistry) RegisterAll(server *Server) {
	r.mu.RLock()
	defer r.mu.RUnlock()

	for _, svc := range r.services {
		svc.Register(server)
	}
}
