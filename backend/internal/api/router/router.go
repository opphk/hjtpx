package router

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/api/middleware"
)

func SetupRoutes(r *gin.Engine) {
	r.Use(middleware.Logger())
	r.Use(middleware.Recovery())
	r.Use(middleware.CORS())

	api := r.Group("/api/v1")
	{
		api.GET("/health", func(c *gin.Context) {
			c.JSON(200, gin.H{"status": "ok"})
		})
	}
}
