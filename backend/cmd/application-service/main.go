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
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/service-discovery"
	"github.com/hjtpx/hjtpx/pb/application"

	"google.golang.org/grpc"
	"google.golang.org/grpc/reflection"
)

type ApplicationServer struct {
	application.UnimplementedApplicationServiceServer
	appService *service.ApplicationService
	cbManager  *circuitbreaker.CircuitBreakerManager
}

func NewApplicationServer(appService *service.ApplicationService) *ApplicationServer {
	return &ApplicationServer{
		appService: appService,
		cbManager:  circuitbreaker.NewCircuitBreakerManager(),
	}
}

func (s *ApplicationServer) CreateApplication(ctx context.Context, req *application.CreateApplicationRequest) (*application.CreateApplicationResponse, error) {
	input := &service.CreateApplicationInput{
		Name:        req.Name,
		UserID:      uint(req.UserId),
		Description: req.Description,
		Domain:      req.Domain,
		Website:     req.Website,
	}

	app, err := s.appService.CreateApplication(input)
	if err != nil {
		return nil, err
	}

	return &application.CreateApplicationResponse{
		Id:       app.ID,
		Name:     app.Name,
		UserId:   app.UserID,
		ApiKey:   app.APIKey,
		IsActive: app.IsActive,
	}, nil
}

func (s *ApplicationServer) GetApplication(ctx context.Context, req *application.GetApplicationRequest) (*application.GetApplicationResponse, error) {
	app, err := s.appService.GetApplicationByID(req.Id)
	if err != nil {
		return nil, err
	}

	return &application.GetApplicationResponse{
		Id:          app.ID,
		Name:        app.Name,
		UserId:      app.UserID,
		Description: app.Description,
		ApiKey:      app.APIKey,
		Domain:      app.Domain,
		Website:     app.Website,
		IsActive:    app.IsActive,
	}, nil
}

func (s *ApplicationServer) ListApplications(ctx context.Context, req *application.ListApplicationsRequest) (*application.ListApplicationsResponse, error) {
	filter := &service.ListApplicationsFilter{
		Page:     int(req.Page),
		PageSize: int(req.PageSize),
		Keyword:  req.Keyword,
		UserID:   uint(req.UserId),
	}

	result, err := s.appService.ListApplications(filter)
	if err != nil {
		return nil, err
	}

	var apps []*application.Application
	for _, item := range result.Data.([]interface{}) {
		app := item.(*service.ApplicationResponse)
		apps = append(apps, &application.Application{
			Id:          app.ID,
			Name:        app.Name,
			UserId:      app.UserID,
			Description: app.Description,
			ApiKey:      app.APIKey,
			IsActive:    app.IsActive,
		})
	}

	return &application.ListApplicationsResponse{
		Applications: apps,
		Total:        result.Total,
		Page:         int32(result.Page),
		PageSize:     int32(result.PageSize),
		TotalPages:   int32(result.TotalPages),
	}, nil
}

func (s *ApplicationServer) UpdateApplication(ctx context.Context, req *application.UpdateApplicationRequest) (*application.UpdateApplicationResponse, error) {
	input := &service.UpdateApplicationInput{}

	if req.Name != nil {
		input.Name = req.Name
	}
	if req.Description != nil {
		input.Description = req.Description
	}
	if req.IsActive != nil {
		input.IsActive = req.IsActive
	}

	app, err := s.appService.UpdateApplication(req.Id, input)
	if err != nil {
		return nil, err
	}

	return &application.UpdateApplicationResponse{
		Id:       app.ID,
		Name:     app.Name,
		IsActive: app.IsActive,
	}, nil
}

func (s *ApplicationServer) DeleteApplication(ctx context.Context, req *application.DeleteApplicationRequest) (*application.DeleteApplicationResponse, error) {
	err := s.appService.DeleteApplication(req.Id)
	return &application.DeleteApplicationResponse{Success: err == nil}, nil
}

func (s *ApplicationServer) RegenerateAPIKey(ctx context.Context, req *application.RegenerateAPIKeyRequest) (*application.RegenerateAPIKeyResponse, error) {
	app, oldKey, err := s.appService.RegenerateAPIKey(req.Id)
	if err != nil {
		return nil, err
	}

	return &application.RegenerateAPIKeyResponse{
		Id:        app.ID,
		NewApiKey: app.APIKey,
		OldApiKey: oldKey,
	}, nil
}

func main() {
	cfg := config.LoadConfig()

	if err := database.InitDB(cfg); err != nil {
		log.Printf("Warning: Failed to initialize database: %v", err)
	}

	if err := postgres.Connect(&cfg.Postgres); err != nil {
		log.Printf("Warning: Failed to connect to PostgreSQL: %v", err)
	}

	appService := service.NewApplicationService()

	grpcPort := os.Getenv("GRPC_PORT")
	if grpcPort == "" {
		grpcPort = "50053"
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

	application.RegisterApplicationServiceServer(server, NewApplicationServer(appService))

	reflection.Register(server)

	consulConfig := &servicediscovery.ConsulConfig{
		Address:    cfg.Consul.Address,
		ServiceName: "application-service",
		ServiceID:   fmt.Sprintf("application-service-%d", time.Now().UnixNano()),
		Host:       "localhost",
		Port:       50053,
		Tags:       []string{"application", "grpc"},
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

	if postgres.DB != nil {
		postgres.DB.Close()
	}

	log.Println("Server exited")
}