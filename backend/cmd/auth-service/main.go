package main

import (
	"context"
	"fmt"
	"log"
	"net"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/circuitbreaker"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/service-discovery"
	"github.com/hjtpx/hjtpx/pb/auth"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type AuthServer struct {
	auth.UnimplementedAuthServiceServer
	authService service.AuthService
	cbManager   *circuitbreaker.CircuitBreakerManager
}

func NewAuthServer(authService service.AuthService) *AuthServer {
	return &AuthServer{
		authService: authService,
		cbManager:   circuitbreaker.NewCircuitBreakerManager(),
	}
}

func (s *AuthServer) GenerateToken(ctx context.Context, req *auth.GenerateTokenRequest) (*auth.GenerateTokenResponse, error) {
	accessToken, refreshToken, err := s.authService.GenerateToken(ctx, uint(req.AdminId), req.Username, req.Role)
	if err != nil {
		return nil, err
	}

	expiry := s.authService.GetTokenExpiry(ctx)

	return &auth.GenerateTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(expiry / time.Second),
	}, nil
}

func (s *AuthServer) ValidateToken(ctx context.Context, req *auth.ValidateTokenRequest) (*auth.ValidateTokenResponse, error) {
	claims, err := s.authService.ValidateToken(ctx, req.Token)
	if err != nil {
		return &auth.ValidateTokenResponse{Valid: false}, nil
	}

	return &auth.ValidateTokenResponse{
		Valid:    true,
		AdminId:  claims.AdminID,
		Username: claims.Username,
		Role:     claims.Role,
	}, nil
}

func (s *AuthServer) RefreshToken(ctx context.Context, req *auth.RefreshTokenRequest) (*auth.RefreshTokenResponse, error) {
	accessToken, refreshToken, err := s.authService.RefreshToken(ctx, req.RefreshToken)
	if err != nil {
		return nil, err
	}

	expiry := s.authService.GetTokenExpiry(ctx)

	return &auth.RefreshTokenResponse{
		AccessToken:  accessToken,
		RefreshToken: refreshToken,
		ExpiresIn:    int64(expiry / time.Second),
	}, nil
}

func (s *AuthServer) InvalidateToken(ctx context.Context, req *auth.InvalidateTokenRequest) (*auth.InvalidateTokenResponse, error) {
	err := s.authService.InvalidateToken(ctx, req.Token)
	return &auth.InvalidateTokenResponse{Success: err == nil}, nil
}

func main() {
	cfg := config.LoadConfig()

	authService, err := service.NewAuthService(service.AuthServiceConfig{
		SecretKey:     cfg.JWT.Secret,
		AccessExpiry:  time.Hour,
		RefreshExpiry: 7 * 24 * time.Hour,
	})
	if err != nil {
		log.Fatalf("Failed to create auth service: %v", err)
	}

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50051"
	}

	lis, err := net.Listen("tcp", ":"+grpcPort)
	if err != nil {
		log.Fatalf("Failed to listen: %v", err)
	}

	cbManager := circuitbreaker.NewCircuitBreakerManager()

	server := grpc.NewServer(
		grpc.UnaryInterceptor(circuitbreaker.UnaryServerInterceptor(cbManager)),
		grpc.StreamInterceptor(circuitbreaker.StreamServerInterceptor(cbManager)),
	)

	auth.RegisterAuthServiceServer(server, NewAuthServer(authService))

	reflection.Register(server)

	consulConfig := &servicediscovery.ConsulConfig{
		Address:    cfg.Consul.Address,
		ServiceName: "auth-service",
		ServiceID:   fmt.Sprintf("auth-service-%d", time.Now().UnixNano()),
		Host:       "localhost",
		Port:       50051,
		Tags:       []string{"auth", "grpc"},
	}

	consulService, err := servicediscovery.NewConsulService(consulConfig)
	if err != nil {
		log.Printf("Warning: Failed to connect to Consul: %v", err)
	} else {
		if err := consulService.Register(consulConfig); err != nil {
			log.Printf("Warning: Failed to register service: %v", err)
		} else {
			defer consulService.Deregister()
		}
	}

	go func() {
		log.Printf("gRPC server starting on port %s...", grpcPort)
		if err := server.Serve(lis); err != nil {
			log.Fatalf("Failed to serve: %v", err)
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	log.Println("Shutting down gRPC server...")
	server.GracefulStop()
	log.Println("Server exited")
}