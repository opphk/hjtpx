package api

import (
	"captchax/internal/middleware"
	"captchax/internal/service"
	"compress/gzip"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router         *gin.Engine
	captchaService *service.CaptchaService
	handler        *Handler
}

type ServerConfig struct {
	EnableV1          bool
	EnableV2          bool
	EnableCompression bool
	EnableDeduplication bool
	EnableCache       bool
	GzipLevel         int
}

var defaultServerConfig = &ServerConfig{
	EnableV1:           true,
	EnableV2:           true,
	EnableCompression:  true,
	EnableDeduplication: true,
	EnableCache:        true,
	GzipLevel:          gzip.DefaultCompression,
}

func NewServer(captchaService *service.CaptchaService) *Server {
	return NewServerWithConfig(captchaService, defaultServerConfig)
}

func NewServerWithConfig(captchaService *service.CaptchaService, config *ServerConfig) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	if config == nil {
		config = defaultServerConfig
	}

	server := &Server{
		router:         router,
		captchaService: captchaService,
		handler:        NewHandler(captchaService),
	}

	server.setupMiddleware(config)
	server.setupRoutes(config)

	return server
}

func (s *Server) setupMiddleware(config *ServerConfig) {
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.Recovery())
	s.router.Use(middleware.CORS())
	s.router.Use(middleware.RequestID())
	s.router.Use(middleware.Security())

	if config.EnableCompression {
		s.router.Use(middleware.GzipHandler(&middleware.GzipConfig{
			Level:         config.GzipLevel,
			MinSize:       1024,
			IncludedTypes: []string{"text/", "application/json", "application/xml"},
		}))
	}

	if config.EnableDeduplication {
		s.router.Use(middleware.Deduplication())
	}

	if config.EnableCache {
		s.router.Use(middleware.CacheControl())
	}
}

func (s *Server) setupRoutes(config *ServerConfig) {
	s.router.GET("/health", s.handler.HealthCheck)

	if config.EnableV1 {
		v1 := s.router.Group("/api/v1")
		v1.Use(middleware.RateLimit(s.captchaService))
		{
			s.setupV1Routes(v1)
		}
	}

	if config.EnableV2 {
		v2 := s.router.Group("/api/v2")
		v2.Use(middleware.RateLimit(s.captchaService))
		{
			s.setupV2Routes(v2)
		}
	}
}

func (s *Server) setupV1Routes(api *gin.RouterGroup) {
	captcha := api.Group("/captcha")
	{
		captcha.POST("/slider", s.handler.getSliderCaptcha)
		captcha.POST("/slider/verify", s.handler.verifySliderCaptcha)
		captcha.POST("/click", s.handler.getClickCaptcha)
		captcha.POST("/click/verify", s.handler.verifyClickCaptcha)
		captcha.POST("/puzzle", s.handler.getPuzzleCaptcha)
		captcha.POST("/puzzle/verify", s.handler.verifyPuzzleCaptcha)
		captcha.POST("/rotate", s.handler.getRotateCaptcha)
		captcha.POST("/rotate/verify", s.handler.verifyRotateCaptcha)
		captcha.POST("/text", s.handler.getTextCaptcha)
		captcha.POST("/text/verify", s.handler.verifyTextCaptcha)
		captcha.POST("/icon", s.handler.getIconCaptcha)
		captcha.POST("/icon/verify", s.handler.verifyIconCaptcha)
		captcha.POST("/audio", s.handler.getAudioCaptcha)
		captcha.POST("/audio/verify", s.handler.verifyAudioCaptcha)
	}
}

func (s *Server) setupV2Routes(api *gin.RouterGroup) {
	captcha := api.Group("/captcha")
	{
		captcha.POST("/slider", s.handler.GetSliderCaptchaV2)
		captcha.POST("/slider/verify", s.handler.VerifySliderCaptchaV2)
		captcha.POST("/click", s.handler.GetClickCaptchaV2)
		captcha.POST("/click/verify", s.handler.VerifyClickCaptchaV2)
		captcha.POST("/puzzle", s.handler.GetPuzzleCaptchaV2)
		captcha.POST("/puzzle/verify", s.handler.VerifyPuzzleCaptchaV2)
		captcha.POST("/rotate", s.handler.GetRotateCaptchaV2)
		captcha.POST("/rotate/verify", s.handler.VerifyRotateCaptchaV2)
		captcha.POST("/text", s.handler.GetTextCaptchaV2)
		captcha.POST("/text/verify", s.handler.VerifyTextCaptchaV2)
		captcha.POST("/icon", s.handler.GetIconCaptchaV2)
		captcha.POST("/icon/verify", s.handler.VerifyIconCaptchaV2)
		captcha.POST("/audio", s.handler.GetAudioCaptchaV2)
		captcha.POST("/audio/verify", s.handler.VerifyAudioCaptchaV2)
		captcha.POST("/behavior", s.handler.GetBehaviorCaptcha)
		captcha.POST("/behavior/verify", s.handler.VerifyBehaviorCaptcha)
	}

	batch := api.Group("/batch")
	{
		batch.POST("/verify", s.handler.BatchVerifyCaptcha)
	}

	scenarios := api.Group("/scenarios")
	{
		scenarios.GET("", s.handler.ListScenarios)
		scenarios.POST("", s.handler.CreateScenario)
		scenarios.GET("/:id", s.handler.GetScenario)
		scenarios.PUT("/:id", s.handler.UpdateScenario)
		scenarios.DELETE("/:id", s.handler.DeleteScenario)
	}

	webhook := api.Group("/webhook")
	{
		webhook.POST("/register", s.handler.RegisterWebhook)
		webhook.DELETE("/:id", s.handler.UnregisterWebhook)
		webhook.GET("", s.handler.ListWebhooks)
		webhook.PUT("/:id", s.handler.UpdateWebhook)
	}
}

func (s *Server) Router() *gin.Engine {
	return s.router
}
