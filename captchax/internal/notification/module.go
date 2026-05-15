package notification

import (
	"github.com/gin-gonic/gin"
)

type Module struct{}

func New() *Module {
	return &Module{}
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	r.GET("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "notification list endpoint"})
	})
	r.POST("/", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "notification create endpoint"})
	})
}