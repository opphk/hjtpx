package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/go-redis/redis/v8"
	"github.com/jmoiron/sqlx"
)

type HealthHandler struct {
	db    *sqlx.DB
	redis *redis.Client
}

func NewHealthHandler(db *sqlx.DB, redis *redis.Client) *HealthHandler {
	return &HealthHandler{db: db, redis: redis}
}

type HealthStatus struct {
	Status    string            `json:"status"`
	Timestamp time.Time         `json:"timestamp"`
	Services  map[string]string `json:"services,omitempty"`
}

func (h *HealthHandler) Health(c *gin.Context) {
	status := "healthy"

	if err := h.db.Ping(); err != nil {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: HealthStatus{
			Status:    status,
			Timestamp: time.Now(),
		},
	})
}

func (h *HealthHandler) DetailedHealth(c *gin.Context) {
	services := make(map[string]string)
	overallStatus := "healthy"

	if err := h.db.Ping(); err != nil {
		services["database"] = "unhealthy: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["database"] = "healthy"
	}

	if err := h.redis.Ping(c.Request.Context()).Err(); err != nil {
		services["redis"] = "unhealthy: " + err.Error()
		overallStatus = "unhealthy"
	} else {
		services["redis"] = "healthy"
	}

	status := "healthy"
	if overallStatus == "unhealthy" {
		status = "unhealthy"
	}

	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data: HealthStatus{
			Status:    status,
			Timestamp: time.Now(),
			Services:  services,
		},
	})
}
