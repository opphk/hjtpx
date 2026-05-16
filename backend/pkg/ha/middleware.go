package ha

import (
	"bytes"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type HAHealthCheckMiddleware struct {
	healthChecker *HealthChecker
	failover     *FailoverController
	cluster      *ClusterManager
	loadBalancer *HighAvailabilityLoadBalancer
	config       *HAHealthCheckConfig
}

type HAHealthCheckConfig struct {
	Enabled           bool
	HealthEndpoint    string
	LivenessEndpoint  string
	ReadinessEndpoint string
	ClusterEndpoint   string
	CheckPostgres     bool
	CheckRedis        bool
	CheckAllNodes     bool
}

func DefaultHAHealthCheckConfig() *HAHealthCheckConfig {
	return &HAHealthCheckConfig{
		Enabled:           true,
		HealthEndpoint:    "/health/ha",
		LivenessEndpoint:  "/health/live",
		ReadinessEndpoint: "/health/ready",
		ClusterEndpoint:   "/health/cluster",
		CheckPostgres:     true,
		CheckRedis:        true,
		CheckAllNodes:     true,
	}
}

func NewHAHealthCheckMiddleware(
	healthChecker *HealthChecker,
	failover *FailoverController,
	cluster *ClusterManager,
	lb *HighAvailabilityLoadBalancer,
) *HAHealthCheckMiddleware {
	return &HAHealthCheckMiddleware{
		healthChecker: healthChecker,
		failover:     failover,
		cluster:      cluster,
		loadBalancer: lb,
		config:       DefaultHAHealthCheckConfig(),
	}
}

func (m *HAHealthCheckMiddleware) RegisterRoutes(r *gin.Engine) {
	health := r.Group("/health")
	{
		health.GET("/ha", m.HAHealthCheck)
		health.GET("/live", m.LivenessCheck)
		health.GET("/ready", m.ReadinessCheck)
		health.GET("/cluster", m.ClusterStatusCheck)
		health.GET("/backends", m.BackendsStatusCheck)
	}
}

func (m *HAHealthCheckMiddleware) HAHealthCheck(c *gin.Context) {
	health := m.healthChecker.GetClusterHealth()

	status := gin.H{
		"status":       string(health.ClusterStatus),
		"timestamp":    time.Now().Format(time.RFC3339),
		"total_nodes":  health.TotalNodes,
		"healthy":      health.HealthyNodes,
		"unhealthy":    health.UnhealthyNodes,
		"degraded":     health.DegradedNodes,
		"avg_latency":  health.AvgLatency.String(),
	}

	statusCode := http.StatusOK
	if health.ClusterStatus != StatusHealthy {
		statusCode = http.StatusServiceUnavailable
	}

	if m.loadBalancer != nil {
		summary := m.loadBalancer.GetHealthSummary()
		status["circuit_breakers"] = summary["circuit_breakers"]
		status["backends"] = summary["backends"]
	}

	if m.cluster != nil {
		status["cluster_state"] = string(m.cluster.GetState())
		status["is_leader"] = m.cluster.IsLeader()
		status["role"] = string(m.cluster.GetCurrentRole())
	}

	c.JSON(statusCode, status)
}

func (m *HAHealthCheckMiddleware) LivenessCheck(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"status":    "alive",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (m *HAHealthCheckMiddleware) ReadinessCheck(c *gin.Context) {
	if m.healthChecker != nil {
		healthyNodes := m.healthChecker.GetHealthyNodes()
		if len(healthyNodes) == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":  "not_ready",
				"reason":  "no healthy nodes available",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}
	}

	if m.loadBalancer != nil {
		backends := m.loadBalancer.HAProxy.loadBalancer.GetStats()
		healthyCount := 0
		for _, backend := range backends {
			if backend.Healthy {
				healthyCount++
			}
		}

		if healthyCount == 0 {
			c.JSON(http.StatusServiceUnavailable, gin.H{
				"status":    "not_ready",
				"reason":    "no healthy backends",
				"timestamp": time.Now().Format(time.RFC3339),
			})
			return
		}
	}

	c.JSON(http.StatusOK, gin.H{
		"status":    "ready",
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (m *HAHealthCheckMiddleware) ClusterStatusCheck(c *gin.Context) {
	if m.cluster == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "cluster_not_configured",
		})
		return
	}

	status := m.cluster.GetStatus()

	c.JSON(http.StatusOK, gin.H{
		"status":        string(status.State),
		"node_id":       status.NodeID,
		"role":          string(status.Role),
		"is_leader":     status.IsLeader,
		"leader_id":     status.LeaderID,
		"term":          status.Term,
		"commit_index":  status.CommitIndex,
		"members":       len(status.Members),
		"member_details": status.Members,
	})
}

func (m *HAHealthCheckMiddleware) BackendsStatusCheck(c *gin.Context) {
	if m.loadBalancer == nil {
		c.JSON(http.StatusServiceUnavailable, gin.H{
			"status": "load_balancer_not_configured",
		})
		return
	}

	backends := m.loadBalancer.HAProxy.loadBalancer.GetStats()
	circuitBreakers := m.loadBalancer.HAProxy.GetAllCircuitBreakerStatuses()

	type BackendInfo struct {
		URL             string `json:"url"`
		Weight          int    `json:"weight"`
		Healthy         bool   `json:"healthy"`
		ActiveConns     int64  `json:"active_connections"`
		TotalConns      uint64 `json:"total_connections"`
		Failures        int    `json:"failures"`
		Latency         string `json:"latency"`
		CircuitBreaker  string `json:"circuit_breaker_state"`
	}

	backendInfos := make([]BackendInfo, 0, len(backends))
	for _, backend := range backends {
		backendInfos = append(backendInfos, BackendInfo{
			URL:            backend.URL,
			Weight:         backend.Weight,
			Healthy:        backend.Healthy,
			ActiveConns:    backend.ActiveConn,
			TotalConns:     backend.TotalConn,
			Failures:       backend.Failures,
			Latency:        backend.Latency,
			CircuitBreaker: circuitBreakers[backend.URL],
		})
	}

	metrics := m.loadBalancer.HAProxy.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"backends":      backendInfos,
		"metrics":       metrics,
		"timestamp":     time.Now().Format(time.RFC3339),
	})
}

type FailoverHandler struct {
	failover *FailoverController
	lb       *HighAvailabilityLoadBalancer
}

func NewFailoverHandler(failover *FailoverController, lb *HighAvailabilityLoadBalancer) *FailoverHandler {
	return &FailoverHandler{
		failover: failover,
		lb:       lb,
	}
}

func (h *FailoverHandler) RegisterRoutes(r *gin.Engine) {
	failover := r.Group("/failover")
	{
		failover.GET("/status", h.GetStatus)
		failover.POST("/manual", h.ManualFailover)
		failover.GET("/metrics", h.GetMetrics)
		failover.GET("/events", h.GetEvents)
	}
}

func (h *FailoverHandler) GetStatus(c *gin.Context) {
	if h.failover == nil {
		response.InternalServerError(c, "failover not configured")
		return
	}

	status := h.failover.GetClusterStatus()

	c.JSON(http.StatusOK, gin.H{
		"primary_node":     status.PrimaryNode,
		"failover_active":  status.FailoverActive,
		"node_states":      status.NodeStates,
		"healthy_nodes":    status.HealthyNodes,
		"cluster_health":   status.ClusterHealth,
	})
}

func (h *FailoverHandler) ManualFailover(c *gin.Context) {
	if h.failover == nil {
		response.InternalServerError(c, "failover not configured")
		return
	}

	var req struct {
		FromNode string `json:"from_node" binding:"required"`
		ToNode   string `json:"to_node" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	if err := h.failover.ManualFailover(req.FromNode, req.ToNode); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("failover from %s to %s completed", req.FromNode, req.ToNode),
	})
}

func (h *FailoverHandler) GetMetrics(c *gin.Context) {
	if h.failover == nil {
		response.InternalServerError(c, "failover not configured")
		return
	}

	metrics := h.failover.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"metrics":    metrics,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *FailoverHandler) GetEvents(c *gin.Context) {
	if h.failover == nil {
		response.InternalServerError(c, "failover not configured")
		return
	}

	events := h.failover.GetEventLog(100)

	type EventInfo struct {
		Type      string                 `json:"type"`
		NodeID    string                 `json:"node_id"`
		Timestamp string                 `json:"timestamp"`
		Message   string                 `json:"message"`
		Metadata  map[string]interface{} `json:"metadata"`
	}

	eventInfos := make([]EventInfo, 0, len(events))
	for _, event := range events {
		eventInfos = append(eventInfos, EventInfo{
			Type:      string(event.Type),
			NodeID:    event.NodeID,
			Timestamp: event.Timestamp.Format(time.RFC3339),
			Message:   event.Message,
			Metadata:  event.Metadata,
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"events":    eventInfos,
		"count":     len(eventInfos),
	})
}

type HAProxyMiddleware struct {
	lb          *HighAvailabilityLoadBalancer
	timeout     time.Duration
	maxRetries  int
	enableRetry bool
}

func NewHAProxyMiddleware(lb *HighAvailabilityLoadBalancer) *HAProxyMiddleware {
	return &HAProxyMiddleware{
		lb:          lb,
		timeout:     30 * time.Second,
		maxRetries:  3,
		enableRetry: true,
	}
}

func (m *HAProxyMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.URL.Path == "/health/ha" ||
			c.Request.URL.Path == "/health/live" ||
			c.Request.URL.Path == "/health/ready" ||
			c.Request.URL.Path == "/health/cluster" {
			c.Next()
			return
		}

		backend, err := m.lb.SelectBackend(c.ClientIP())
		if err != nil {
			response.ServiceUnavailable(c, "no available backend")
			c.Abort()
			return
		}

		start := time.Now()

		var lastErr error
		for attempt := 0; attempt <= m.maxRetries; attempt++ {
			if attempt > 0 && m.enableRetry {
				time.Sleep(time.Duration(attempt*100) * time.Millisecond)
			}

			req, err := m.createProxyRequest(c, backend.URL)
			if err != nil {
				lastErr = err
				continue
			}

			resp, err := m.executeRequest(req)
			if err != nil {
				m.lb.HAProxy.RecordFailure(backend.URL)
				lastErr = err
				continue
			}

			m.lb.RecordRequest(backend.URL, true, time.Since(start))
			m.copyResponse(c, resp)
			return
		}

		m.lb.RecordRequest(backend.URL, false, time.Since(start))
		m.lb.HAProxy.RecordFailure(backend.URL)
		response.ServiceUnavailable(c, "failed to process request")
		c.Abort()
	}
}

func (m *HAProxyMiddleware) createProxyRequest(c *gin.Context, backendURL string) (*http.Request, error) {
	fullURL := backendURL + c.Request.URL.Path
	if c.Request.URL.RawQuery != "" {
		fullURL += "?" + c.Request.URL.RawQuery
	}

	var body io.Reader
	if c.Request.Body != nil {
		body = c.Request.Body
	}

	req, err := http.NewRequest(c.Request.Method, fullURL, body)
	if err != nil {
		return nil, err
	}

	for key, values := range c.Request.Header {
		for _, value := range values {
			req.Header.Add(key, value)
		}
	}

	req.Header.Set("X-Forwarded-For", c.ClientIP())
	req.Header.Set("X-Real-IP", c.ClientIP())
	req.Header.Set("X-Original-URL", c.Request.URL.String())

	return req, nil
}

func (m *HAProxyMiddleware) executeRequest(req *http.Request) (*http.Response, error) {
	client := &http.Client{
		Timeout: m.timeout,
		Transport: &http.Transport{
			MaxIdleConns:        100,
			MaxIdleConnsPerHost: 10,
		},
	}

	return client.Do(req)
}

func (m *HAProxyMiddleware) copyResponse(c *gin.Context, resp *http.Response) {
	defer resp.Body.Close()

	for key, values := range resp.Header {
		for _, value := range values {
			c.Header(key, value)
		}
	}

	c.Header("X-Proxy-Backend", resp.Request.URL.String())

	c.DataFromReader(resp.StatusCode, resp.ContentLength, resp.Header.Get("Content-Type"), resp.Body, nil)
}

type SyncHandler struct {
	syncService *DataSyncService
}

func NewSyncHandler(syncService *DataSyncService) *SyncHandler {
	return &SyncHandler{
		syncService: syncService,
	}
}

func (h *SyncHandler) RegisterRoutes(r *gin.Engine) {
	sync := r.Group("/sync")
	{
		sync.GET("/status", h.GetStatus)
		sync.GET("/nodes", h.GetNodes)
		sync.GET("/metrics", h.GetMetrics)
		sync.POST("/full", h.FullSync)
	}
}

func (h *SyncHandler) GetStatus(c *gin.Context) {
	if h.syncService == nil {
		response.InternalServerError(c, "sync service not configured")
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"node_id": h.syncService.nodeID,
		"strategy": string(h.syncService.config.Strategy),
	})
}

func (h *SyncHandler) GetNodes(c *gin.Context) {
	if h.syncService == nil {
		response.InternalServerError(c, "sync service not configured")
		return
	}

	nodes := h.syncService.GetAllNodeStatuses()

	type NodeInfo struct {
		NodeID    string `json:"node_id"`
		Address   string `json:"address"`
		Status    string `json:"status"`
		LastSync  string `json:"last_sync"`
	}

	nodeInfos := make([]NodeInfo, 0, len(nodes))
	for nodeID, node := range nodes {
		nodeInfos = append(nodeInfos, NodeInfo{
			NodeID:   nodeID,
			Address:  node.Address,
			Status:   string(node.Status),
			LastSync: node.LastSync.Format(time.RFC3339),
		})
	}

	c.JSON(http.StatusOK, gin.H{
		"nodes": nodeInfos,
	})
}

func (h *SyncHandler) GetMetrics(c *gin.Context) {
	if h.syncService == nil {
		response.InternalServerError(c, "sync service not configured")
		return
	}

	metrics := h.syncService.GetMetrics()

	c.JSON(http.StatusOK, gin.H{
		"metrics":   metrics,
		"timestamp": time.Now().Format(time.RFC3339),
	})
}

func (h *SyncHandler) FullSync(c *gin.Context) {
	if h.syncService == nil {
		response.InternalServerError(c, "sync service not configured")
		return
	}

	var req struct {
		TargetNode string `json:"target_node" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request")
		return
	}

	if err := h.syncService.fullSync(req.TargetNode); err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"status":  "success",
		"message": fmt.Sprintf("full sync to %s completed", req.TargetNode),
	})
}
