package handler

import (
	"context"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/postgres"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

// HealthCheck 健康检查接口
func HealthCheck(c *gin.Context) {
	status := map[string]interface{}{
		"status": "ok",
		"time":   time.Now().Format(time.RFC3339),
	}

	// 检查PostgreSQL连接
	if postgres.DB != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := postgres.DB.PingContext(ctx); err != nil {
			status["postgres"] = map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}
		} else {
			status["postgres"] = map[string]interface{}{
				"status": "ok",
			}
		}
	} else {
		status["postgres"] = map[string]interface{}{
			"status":  "disconnected",
			"message": "database not initialized",
		}
	}

	// 检查Redis连接
	if redis.Client != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 3*time.Second)
		defer cancel()
		if err := redis.Client.Ping(ctx).Err(); err != nil {
			status["redis"] = map[string]interface{}{
				"status":  "error",
				"message": err.Error(),
			}
		} else {
			status["redis"] = map[string]interface{}{
				"status": "ok",
			}
		}
	} else {
		status["redis"] = map[string]interface{}{
			"status":  "disconnected",
			"message": "redis not initialized",
		}
	}

	response.Success(c, status)
}
