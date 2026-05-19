package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service/cdn"
)

var cdnService *cdn.CDNService

func InitCDNService(service *cdn.CDNService) {
	cdnService = service
}

func GetRegions(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regions := cdnService.ListRegions()
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    regions,
	})
}

func GetRegion(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regionID := c.Param("region_id")
	region, err := cdnService.GetRegion(regionID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    region,
	})
}

func CreateRegion(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	var region cdn.Region
	if err := c.ShouldBindJSON(&region); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := cdnService.AddRegion(&region)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "region created",
		"data":    region,
	})
}

func UpdateRegion(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regionID := c.Param("region_id")
	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := cdnService.UpdateRegion(regionID, updates)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "region updated",
	})
}

func DeleteRegion(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regionID := c.Param("region_id")
	err := cdnService.DeleteRegion(regionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "region deleted",
	})
}

func GetRegionStats(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regionID := c.Param("region_id")
	stats, err := cdnService.GetRegionStats(regionID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

func GetGlobalStats(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	stats, err := cdnService.GetGlobalStats()
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

func GetNodes(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regionID := c.Query("region_id")
	nodes := cdnService.ListNodes(regionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nodes,
	})
}

func GetNode(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	nodeID := c.Param("node_id")
	node, err := cdnService.GetNode(nodeID)
	if err != nil {
		c.JSON(http.StatusNotFound, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    node,
	})
}

func RegisterNode(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	var node cdn.EdgeNode
	if err := c.ShouldBindJSON(&node); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := cdnService.RegisterNode(&node)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{
		"code":    0,
		"message": "node registered",
		"data":    node,
	})
}

func RemoveNode(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	nodeID := c.Param("node_id")
	err := cdnService.RemoveNode(nodeID)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "node removed",
	})
}

func GetHealthyNodes(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	regionID := c.Query("region_id")
	nodes := cdnService.GetHealthyNodes(regionID)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    nodes,
	})
}

func UpdateNodeHealth(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	nodeID := c.Param("node_id")

	var healthUpdate struct {
		IsHealthy bool    `json:"is_healthy"`
		LatencyMs float64 `json:"latency_ms"`
	}

	if err := c.ShouldBindJSON(&healthUpdate); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := cdnService.UpdateNodeHealth(nodeID, healthUpdate.IsHealthy, healthUpdate.LatencyMs)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "node health updated",
	})
}

func ExecuteEdgeFunction(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	functionName := c.Param("function_name")
	var params map[string]interface{}
	if err := c.ShouldBindJSON(&params); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	result, err := cdnService.ExecuteEdgeFunction(c.Request.Context(), functionName, params)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "function executed",
		"data":    result,
	})
}

func GetCacheStats(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	stats := cdnService.GetCacheStats()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    stats,
	})
}

func ClearCache(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	cdnService.ClearCache()

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "cache cleared",
	})
}

func PurgeCache(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	assetPath := c.Query("path")
	if assetPath == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "path is required"})
		return
	}

	err := cdnService.PurgeCache(assetPath)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "cache purged",
	})
}

func WarmupCache(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	var request struct {
		Paths []string `json:"paths"`
	}

	if err := c.ShouldBindJSON(&request); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	err := cdnService.WarmupCache(request.Paths)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "cache warmed up",
	})
}

func GetClientLocation(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	clientIP := c.ClientIP()
	location := cdnService.GetClientLocation(clientIP)

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    location,
	})
}

func GetRoutingDecision(c *gin.Context) {
	if cdnService == nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": "CDN service not initialized"})
		return
	}

	clientIP := c.ClientIP()
	result, err := cdnService.GetRoutingDecision(clientIP)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data":    result,
	})
}