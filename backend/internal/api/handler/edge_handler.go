package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/repository"
	"github.com/hjtpx/hjtpx/internal/service/edge"
	"github.com/hjtpx/hjtpx/pkg/config"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type EdgeHandler struct {
	verificationService edge.EdgeVerificationService
	loadBalancer       edge.EdgeLoadBalancer
	syncService        edge.EdgeSyncService
	healthMonitor      edge.EdgeHealthMonitor
	repo               repository.EdgeRepository
	cfg                *config.Config
}

func NewEdgeHandler(
	verificationService edge.EdgeVerificationService,
	loadBalancer edge.EdgeLoadBalancer,
	syncService edge.EdgeSyncService,
	healthMonitor edge.EdgeHealthMonitor,
	repo repository.EdgeRepository,
	cfg *config.Config,
) *EdgeHandler {
	return &EdgeHandler{
		verificationService: verificationService,
		loadBalancer:        loadBalancer,
		syncService:         syncService,
		healthMonitor:       healthMonitor,
		repo:                repo,
		cfg:                 cfg,
	}
}

func (h *EdgeHandler) RegisterRoutes(api *gin.RouterGroup) {
	edgeGroup := api.Group("/edge")

	edgeGroup.POST("/nodes", h.CreateNode)
	edgeGroup.GET("/nodes", h.ListNodes)
	edgeGroup.GET("/nodes/:nodeID", h.GetNode)
	edgeGroup.PUT("/nodes/:nodeID", h.UpdateNode)
	edgeGroup.DELETE("/nodes/:nodeID", h.DeleteNode)

	edgeGroup.POST("/nodes/:nodeID/heartbeat", h.UpdateHeartbeat)
	edgeGroup.GET("/nodes/:nodeID/health", h.GetNodeHealth)
	edgeGroup.GET("/nodes/:nodeID/status", h.GetNodeStatus)
	edgeGroup.PUT("/nodes/:nodeID/status", h.UpdateNodeStatus)

	edgeGroup.POST("/verify", h.VerifyCaptcha)
	edgeGroup.GET("/verify/:sessionID", h.GetVerificationResult)

	edgeGroup.POST("/sync", h.TriggerSync)
	edgeGroup.GET("/sync/records/:nodeID", h.GetSyncRecords)

	edgeGroup.POST("/load-balance/select", h.SelectNode)
	edgeGroup.GET("/load-balance/nodes", h.GetOnlineNodes)

	edgeGroup.GET("/health/all", h.GetAllNodesHealth)
}

func (h *EdgeHandler) CreateNode(c *gin.Context) {
	var req struct {
		NodeID        string            `json:"node_id" binding:"required"`
		NodeName      string            `json:"node_name" binding:"required"`
		NodeType      string            `json:"node_type"`
		Region        string            `json:"region"`
		Zone          string            `json:"zone"`
		CloudEndpoint string            `json:"cloud_endpoint"`
		IPAddress     string            `json:"ip_address"`
		Version       string            `json:"version"`
		Capacity      model.EdgeCapacity `json:"capacity"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	node := model.NewEdgeNode(req.NodeID, req.NodeName, req.NodeType, req.Region, req.Zone)
	node.CloudEndpoint = req.CloudEndpoint
	node.IPAddress = req.IPAddress
	node.Version = req.Version
	node.Capacity = req.Capacity
	node.Status = model.EdgeNodeStatusOnline

	err := h.repo.CreateNode(c.Request.Context(), node)
	if err != nil {
		response.InternalServerError(c, "Failed to create node")
		return
	}

	response.Success(c, node)
}

func (h *EdgeHandler) ListNodes(c *gin.Context) {
	region := c.Query("region")
	zone := c.Query("zone")
	statusStr := c.Query("status")

	var status model.EdgeNodeStatus
	if statusStr != "" {
		status = model.EdgeNodeStatus(statusStr)
	}

	nodes, err := h.repo.ListNodes(c.Request.Context(), region, zone, status)
	if err != nil {
		response.InternalServerError(c, "Failed to list nodes")
		return
	}

	response.Success(c, gin.H{
		"nodes":       nodes,
		"total_count": len(nodes),
	})
}

func (h *EdgeHandler) GetNode(c *gin.Context) {
	nodeID := c.Param("nodeID")

	node, err := h.repo.GetNodeByNodeID(c.Request.Context(), nodeID)
	if err != nil {
		response.NotFound(c, "Node not found")
		return
	}

	response.Success(c, node)
}

func (h *EdgeHandler) UpdateNode(c *gin.Context) {
	nodeID := c.Param("nodeID")

	var req struct {
		NodeName       string            `json:"node_name"`
		NodeType       string            `json:"node_type"`
		Region         string            `json:"region"`
		Zone           string            `json:"zone"`
		CloudEndpoint  string            `json:"cloud_endpoint"`
		IPAddress      string            `json:"ip_address"`
		Version        string            `json:"version"`
		Capacity       model.EdgeCapacity `json:"capacity"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	node, err := h.repo.GetNodeByNodeID(c.Request.Context(), nodeID)
	if err != nil {
		response.NotFound(c, "Node not found")
		return
	}

	if req.NodeName != "" {
		node.NodeName = req.NodeName
	}
	if req.NodeType != "" {
		node.NodeType = req.NodeType
	}
	if req.Region != "" {
		node.Region = req.Region
	}
	if req.Zone != "" {
		node.Zone = req.Zone
	}
	if req.CloudEndpoint != "" {
		node.CloudEndpoint = req.CloudEndpoint
	}
	if req.IPAddress != "" {
		node.IPAddress = req.IPAddress
	}
	if req.Version != "" {
		node.Version = req.Version
	}
	if req.Capacity.MaxRequestsPerSecond > 0 {
		node.Capacity = req.Capacity
	}

	err = h.repo.UpdateNode(c.Request.Context(), node)
	if err != nil {
		response.InternalServerError(c, "Failed to update node")
		return
	}

	response.Success(c, node)
}

func (h *EdgeHandler) DeleteNode(c *gin.Context) {
	nodeID := c.Param("nodeID")

	err := h.repo.DeleteNode(c.Request.Context(), nodeID)
	if err != nil {
		response.InternalServerError(c, "Failed to delete node")
		return
	}

	c.Status(204)
}

func (h *EdgeHandler) UpdateHeartbeat(c *gin.Context) {
	nodeID := c.Param("nodeID")

	var req struct {
		LoadMetrics model.EdgeLoadMetrics `json:"load_metrics"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	err := h.repo.UpdateNodeHeartbeat(c.Request.Context(), nodeID, req.LoadMetrics)
	if err != nil {
		response.InternalServerError(c, "Failed to update heartbeat")
		return
	}

	response.Success(c, gin.H{"message": "Heartbeat updated"})
}

func (h *EdgeHandler) GetNodeHealth(c *gin.Context) {
	nodeID := c.Param("nodeID")

	result, err := h.healthMonitor.GetHealthStatus(c.Request.Context(), nodeID)
	if err != nil {
		response.InternalServerError(c, "Failed to get health status")
		return
	}

	response.Success(c, result)
}

func (h *EdgeHandler) GetNodeStatus(c *gin.Context) {
	nodeID := c.Param("nodeID")

	status, err := h.healthMonitor.GetNodeStatus(c.Request.Context(), nodeID)
	if err != nil {
		response.InternalServerError(c, "Failed to get node status")
		return
	}

	response.Success(c, gin.H{"node_id": nodeID, "status": status})
}

func (h *EdgeHandler) UpdateNodeStatus(c *gin.Context) {
	nodeID := c.Param("nodeID")

	var req struct {
		Status model.EdgeNodeStatus `json:"status" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	err := h.healthMonitor.UpdateNodeStatus(c.Request.Context(), nodeID, req.Status)
	if err != nil {
		response.InternalServerError(c, "Failed to update node status")
		return
	}

	response.Success(c, gin.H{"node_id": nodeID, "status": req.Status})
}

func (h *EdgeHandler) VerifyCaptcha(c *gin.Context) {
	var req struct {
		NodeID      string                 `json:"node_id" binding:"required"`
		SessionID   string                 `json:"session_id" binding:"required"`
		RequestData map[string]interface{} `json:"request_data"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	result, err := h.verificationService.VerifyCaptcha(c.Request.Context(), req.NodeID, req.SessionID, req.RequestData)
	if err != nil {
		response.InternalServerError(c, "Verification failed")
		return
	}

	response.Success(c, result)
}

func (h *EdgeHandler) GetVerificationResult(c *gin.Context) {
	sessionID := c.Param("sessionID")

	result, err := h.verificationService.GetCachedVerificationResult(c.Request.Context(), sessionID)
	if err != nil {
		response.InternalServerError(c, "Failed to get verification result")
		return
	}

	if result == nil {
		response.NotFound(c, "Verification result not found")
		return
	}

	response.Success(c, result)
}

func (h *EdgeHandler) TriggerSync(c *gin.Context) {
	var req struct {
		NodeID string `json:"node_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	if req.NodeID != "" {
		err := h.syncService.TriggerSync(c.Request.Context(), req.NodeID)
		if err != nil {
			response.InternalServerError(c, "Sync failed")
			return
		}
	}

	response.Success(c, gin.H{"message": "Sync triggered"})
}

func (h *EdgeHandler) GetSyncRecords(c *gin.Context) {
	nodeID := c.Param("nodeID")
	limitStr := c.Query("limit")

	limit := 10
	if limitStr != "" {
		var err error
		limit, err = strconv.Atoi(limitStr)
		if err != nil {
			response.BadRequest(c, "Invalid limit")
			return
		}
	}

	records, err := h.syncService.GetSyncRecords(c.Request.Context(), nodeID, limit)
	if err != nil {
		response.InternalServerError(c, "Failed to get sync records")
		return
	}

	response.Success(c, records)
}

func (h *EdgeHandler) SelectNode(c *gin.Context) {
	var req struct {
		Region   string                  `json:"region"`
		Zone     string                  `json:"zone"`
		Strategy edge.LoadBalanceStrategy `json:"strategy"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	var node *model.EdgeNode
	var err error

	if req.Strategy != "" {
		node, err = h.loadBalancer.SelectNodeWithStrategy(c.Request.Context(), req.Region, req.Zone, req.Strategy)
	} else {
		node, err = h.loadBalancer.SelectNode(c.Request.Context(), req.Region, req.Zone)
	}

	if err != nil {
		response.InternalServerError(c, "Failed to select node")
		return
	}

	response.Success(c, node)
}

func (h *EdgeHandler) GetOnlineNodes(c *gin.Context) {
	region := c.Query("region")
	zone := c.Query("zone")

	nodes, err := h.loadBalancer.GetOnlineNodes(c.Request.Context(), region, zone)
	if err != nil {
		response.InternalServerError(c, "Failed to get online nodes")
		return
	}

	response.Success(c, nodes)
}

func (h *EdgeHandler) GetAllNodesHealth(c *gin.Context) {
	results, err := h.healthMonitor.GetAllNodesHealth(c.Request.Context())
	if err != nil {
		response.InternalServerError(c, "Failed to get all nodes health")
		return
	}

	response.Success(c, results)
}

func GetEdgeHandler() *EdgeHandler {
	cfg := config.GetConfig()
	repo := repository.NewEdgeRepository()
	verificationService := edge.NewEdgeVerificationService(repo, cfg)
	loadBalancer := edge.NewEdgeLoadBalancer(repo, cfg)
	syncService := edge.NewEdgeSyncService(repo, cfg)
	healthMonitor := edge.NewEdgeHealthMonitor(repo, cfg, syncService)

	return NewEdgeHandler(verificationService, loadBalancer, syncService, healthMonitor, repo, cfg)
}