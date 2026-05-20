package api

import (
	"github.com/gin-gonic/gin"
)

type APIVersion struct {
	Version   string `json:"version"`
	APIVersion string `json:"api_version"`
	BuildDate string `json:"build_date"`
	GoVersion string `json:"go_version"`
}

const (
	APIVersionV1 = "v1"
	APIVersionV2 = "v2"
	APIVersionV3 = "v3"
)

var BuildInfo = APIVersion{
	Version:   "20.0.0",
	APIVersion: APIVersionV3,
	BuildDate:  "2026-05-20",
	GoVersion:  "1.21",
}

func RegisterVersionEndpoints(r *gin.Engine) {
	r.GET("/api/version", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"version":     BuildInfo.Version,
			"api_version": BuildInfo.APIVersion,
			"build_date":  BuildInfo.BuildDate,
			"go_version":  BuildInfo.GoVersion,
		})
	})

	r.GET("/api/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": BuildInfo.Version,
		})
	})

	r.GET("/api/v1/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": BuildInfo.Version,
		})
	})

	r.GET("/api/v2/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": BuildInfo.Version,
		})
	})

	r.GET("/api/v3/health", func(c *gin.Context) {
		c.JSON(200, gin.H{
			"status":  "healthy",
			"version": BuildInfo.Version,
		})
	})
}
