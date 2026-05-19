package middleware

import (
	"net/http"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/cdn"
)

var cdnService *cdn.CDNService

func InitCDNMiddleware(service *cdn.CDNService) {
	cdnService = service
}

func CDNAcceleration() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cdnService == nil {
			c.Next()
			return
		}

		path := c.Request.URL.Path
		if !isStaticAsset(path) {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		asset, err := cdnService.AccelerateStaticAsset(c.Request.Context(), path, clientIP)
		if err != nil {
			c.Next()
			return
		}

		c.Header("Content-Type", asset.ContentType)
		c.Header("ETag", asset.ETag)
		c.Header("Last-Modified", asset.LastModified.Format(time.RFC1123))
		c.Header("X-Cache-Hit", boolToString(asset.CacheHit))
		c.Header("X-Optimize-Level", string(rune(asset.OptimizeLevel+'0')))
		c.Header("X-Region", asset.RegionID)

		c.Data(http.StatusOK, asset.ContentType, asset.Content)
		c.Abort()
	}
}

func SmartRouting() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cdnService == nil {
			c.Next()
			return
		}

		clientIP := c.ClientIP()

		node, err := cdnService.RouteRequest(c.Request.Context(), clientIP)
		if err != nil {
			c.Next()
			return
		}

		c.Set("cdn_node_id", node.ID)
		c.Set("cdn_region_id", node.RegionID)
		c.Set("cdn_node_ip", node.IPAddress)

		c.Header("X-CDN-Node", node.ID)
		c.Header("X-CDN-Region", node.RegionID)

		c.Next()
	}
}

func EdgeComputing() gin.HandlerFunc {
	return func(c *gin.Context) {
		if cdnService == nil {
			c.Next()
			return
		}

		functionName := c.GetHeader("X-Edge-Function")
		if functionName == "" {
			c.Next()
			return
		}

		var params map[string]interface{}
		if err := c.ShouldBindJSON(&params); err != nil {
			c.Next()
			return
		}

		result, err := cdnService.ExecuteEdgeFunction(c.Request.Context(), functionName, params)
		if err != nil {
			c.Next()
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"code":    0,
			"message": "success",
			"data":    result,
		})
		c.Abort()
	}
}

func CDNCacheControl() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path

		if isStaticAsset(path) {
			c.Header("Cache-Control", "public, max-age=86400")
		} else {
			c.Header("Cache-Control", "no-cache, no-store, must-revalidate")
		}

		c.Next()
	}
}

func CDNLogging() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()

		c.Next()

		duration := time.Since(start)
		clientIP := c.ClientIP()
		path := c.Request.URL.Path
		method := c.Request.Method
		statusCode := c.Writer.Status()

		nodeID, _ := c.Get("cdn_node_id")
		regionID, _ := c.Get("cdn_region_id")

		_ = map[string]interface{}{
			"timestamp":   time.Now().UTC(),
			"client_ip":   clientIP,
			"method":      method,
			"path":        path,
			"status_code": statusCode,
			"duration_ms": duration.Milliseconds(),
			"cdn_node":    nodeID,
			"cdn_region":  regionID,
		}
	}
}

func isStaticAsset(path string) bool {
	staticExtensions := []string{
		".js", ".css", ".html", ".htm",
		".jpg", ".jpeg", ".png", ".gif", ".svg", ".ico",
		".json", ".xml", ".txt",
		".woff", ".woff2", ".ttf", ".eot",
		".mp4", ".webm", ".ogg",
	}

	for _, ext := range staticExtensions {
		if strings.HasSuffix(strings.ToLower(path), ext) {
			return true
		}
	}
	return false
}

func boolToString(b bool) string {
	if b {
		return "true"
	}
	return "false"
}