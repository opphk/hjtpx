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

	"github.com/hjtpx/hjtpx/internal/repository/cache"
	"github.com/hjtpx/hjtpx/internal/repository/db"
	"github.com/hjtpx/hjtpx/internal/service/captcha"
	"github.com/hjtpx/hjtpx/pkg/circuitbreaker"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/service-discovery"
	"github.com/hjtpx/hjtpx/pb/captcha"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type CaptchaServer struct {
	captcha.UnimplementedCaptchaServiceServer
	generatorService *captcha.GeneratorService
	verifierService  *captcha.VerifierService
	cbManager        *circuitbreaker.CircuitBreakerManager
}

func NewCaptchaServer(generator *captcha.GeneratorService, verifier *captcha.VerifierService) *CaptchaServer {
	return &CaptchaServer{
		generatorService: generator,
		verifierService:  verifier,
		cbManager:        circuitbreaker.NewCircuitBreakerManager(),
	}
}

func (s *CaptchaServer) CreateCaptcha(ctx context.Context, req *captcha.CreateCaptchaRequest) (*captcha.CreateCaptchaResponse, error) {
	request := &captcha.CreateCaptchaRequest{
		Width:        int(req.Width),
		Height:       int(req.Height),
		SliderWidth:  int(req.SliderWidth),
		SliderHeight: int(req.SliderHeight),
		ClientIP:     req.ClientIp,
		UserAgent:    req.UserAgent,
		Fingerprint:  req.Fingerprint,
	}

	response, err := s.generatorService.Create(ctx, request)
	if err != nil {
		return nil, err
	}

	return &captcha.CreateCaptchaResponse{
		SessionId:     response.SessionID,
		BackgroundUrl: response.BackgroundURL,
		SliderUrl:     response.SliderURL,
		GapX:          int32(response.GapX),
		GapY:          int32(response.GapY),
		ExpiresIn:     response.ExpiresIn,
		ExpiresAt:     response.ExpiresAt,
	}, nil
}

func (s *CaptchaServer) VerifyCaptcha(ctx context.Context, req *captcha.VerifyCaptchaRequest) (*captcha.VerifyCaptchaResponse, error) {
	request := &captcha.VerifyRequest{
		SessionID: req.SessionId,
		SlideX:    int(req.SlideX),
		SlideY:    int(req.SlideY),
		Duration:  int(req.Duration),
		ClientIP:  req.ClientIp,
	}

	response, err := s.verifierService.Verify(ctx, request)
	if err != nil {
		return &captcha.VerifyCaptchaResponse{
			Success:    false,
			SessionId:  req.SessionId,
			Status:     "failed",
			Score:      0,
			Message:    err.Error(),
		}, nil
	}

	return &captcha.VerifyCaptchaResponse{
		Success:    response.Success,
		SessionId:  response.SessionID,
		Status:     response.Status,
		Score:      float32(response.Score),
		Message:    response.Message,
	}, nil
}

func (s *CaptchaServer) GetSession(ctx context.Context, req *captcha.GetSessionRequest) (*captcha.GetSessionResponse, error) {
	session, err := s.generatorService.GetSession(ctx, req.SessionId)
	if err != nil {
		return nil, err
	}

	return &captcha.GetSessionResponse{
		SessionId:    session.SessionID,
		Status:       session.Status,
		VerifyCount:  int32(session.VerifyCount),
		MaxAttempts:  int32(session.MaxAttempts),
		RiskScore:    float32(session.RiskScore),
	}, nil
}

func (s *CaptchaServer) DeleteSession(ctx context.Context, req *captcha.DeleteSessionRequest) (*captcha.DeleteSessionResponse, error) {
	err := s.generatorService.DeleteSession(ctx, req.SessionId)
	return &captcha.DeleteSessionResponse{Success: err == nil}, nil
}

func main() {
	cfg := config.LoadConfig()

	sessionCache := cache.NewSessionCache(&cfg.Redis)
	captchaRepo := db.NewCaptchaRepository()

	generatorService := captcha.NewGeneratorService(sessionCache, captchaRepo)
	verifierService := captcha.NewVerifierService(sessionCache, captchaRepo)

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50052"
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

	captcha.RegisterCaptchaServiceServer(server, NewCaptchaServer(generatorService, verifierService))

	reflection.Register(server)

	consulConfig := &servicediscovery.ConsulConfig{
		Address:    cfg.Consul.Address,
		ServiceName: "captcha-service",
		ServiceID:   fmt.Sprintf("captcha-service-%d", time.Now().UnixNano()),
		Host:       "localhost",
		Port:       50052,
		Tags:       []string{"captcha", "grpc"},
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