package gateway

import (
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
)

// ServiceRegistry 服务注册表
type ServiceRegistry struct {
	mu       sync.RWMutex
	services map[string]*ServiceInfo
}

// ServiceInfo 服务信息
type ServiceInfo struct {
	Name      string            `json:"name"`
	Endpoints []string          `json:"endpoints"`
	Healthy   bool              `json:"healthy"`
	LastCheck time.Time        `json:"last_check"`
	Metadata  map[string]string `json:"metadata"`
}

// APIGateway API网关
type APIGateway struct {
	registry *ServiceRegistry
}

// NewAPIGateway 创建API网关
func NewAPIGateway() *APIGateway {
	return &APIGateway{
		registry: &ServiceRegistry{
			services: make(map[string]*ServiceInfo),
		},
	}
}

// RegisterService 注册服务
func (gw *APIGateway) RegisterService(name string, endpoints []string, metadata map[string]string) {
	gw.registry.mu.Lock()
	defer gw.registry.mu.Unlock()

	gw.registry.services[name] = &ServiceInfo{
		Name:       name,
		Endpoints:  endpoints,
		Healthy:    true,
		LastCheck: time.Now(),
		Metadata:   metadata,
	}
}

// GetService 获取服务
func (gw *APIGateway) GetService(name string) (*ServiceInfo, bool) {
	gw.registry.mu.RLock()
	defer gw.registry.mu.RUnlock()

	svc, ok := gw.registry.services[name]
	return svc, ok
}

// GatewayMiddleware API网关中间件
func (gw *APIGateway) GatewayMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		
		// 记录请求
		c.Header("X-Gateway-Version", "v1.0")

		// 继续处理请求
		c.Next()

		// 记录响应时间
		duration := time.Since(start)
		c.Header("X-Response-Time", duration.String())
	}
}

// APIVersionMiddleware API版本中间件
func APIVersionMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		path := c.Request.URL.Path
		if strings.HasPrefix(path, "/api/v1/") {
			c.Set("api_version", "v1")
		} else if strings.HasPrefix(path, "/api/v2/") {
			c.Set("api_version", "v2")
		}
		c.Next()
	}
}

// HealthCheckHandler 健康检查处理
func (gw *APIGateway) HealthCheckHandler(c *gin.Context) {
		status := map[string]interface{}{
		"status": "healthy",
		"services": gw.registry.services,
		"timestamp": time.Now().Format(time.RFC3339),
	}
	c.JSON(http.StatusOK, status)
}
