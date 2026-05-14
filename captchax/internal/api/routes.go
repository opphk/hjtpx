package api

import (
	"captchax/internal/middleware"
	"captchax/internal/service"

	"github.com/gin-gonic/gin"
)

type Server struct {
	router         *gin.Engine
	captchaService *service.CaptchaService
	handler        *Handler
}

func NewServer(captchaService *service.CaptchaService) *Server {
	gin.SetMode(gin.ReleaseMode)
	router := gin.New()

	server := &Server{
		router:         router,
		captchaService: captchaService,
		handler:        NewHandler(captchaService),
	}

	server.setupMiddleware()
	server.setupRoutes()

	return server
}

func (s *Server) setupMiddleware() {
	s.router.Use(middleware.Logger())
	s.router.Use(middleware.Recovery())
	s.router.Use(middleware.CORS())
}

func (s *Server) setupRoutes() {
	s.router.GET("/health", s.handler.healthCheck)

	api := s.router.Group("/api/v1")
	api.Use(middleware.RateLimit(s.captchaService))
	{
		captcha := api.Group("/captcha")
		{
			captcha.POST("/slider", s.handler.getSliderCaptcha)
			captcha.POST("/slider/verify", s.handler.verifySliderCaptcha)
			captcha.POST("/click", s.handler.getClickCaptcha)
			captcha.POST("/click/verify", s.handler.verifyClickCaptcha)
			captcha.POST("/puzzle", s.handler.getPuzzleCaptcha)
			captcha.POST("/puzzle/verify", s.handler.verifyPuzzleCaptcha)
		}
	}
}

func (s *Server) Router() *gin.Engine {
	return s.router
}
