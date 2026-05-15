package file

import (
	"github.com/gin-gonic/gin"
)

type Module struct {
	uploadDir     string
	maxUploadSize int64
}

func New(uploadDir string, maxUploadSize int64) *Module {
	return &Module{
		uploadDir:     uploadDir,
		maxUploadSize: maxUploadSize,
	}
}

func (m *Module) RegisterRoutes(r *gin.RouterGroup) {
	r.POST("/upload", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "file upload endpoint"})
	})
	r.GET("/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "file download endpoint"})
	})
	r.DELETE("/:id", func(c *gin.Context) {
		c.JSON(200, gin.H{"message": "file delete endpoint"})
	})
}