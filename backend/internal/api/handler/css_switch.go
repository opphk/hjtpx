package handler

import (
	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
	"github.com/hjtpx/hjtpx/pkg/response"
)

const cssSourceKey = "config:css_source"

type CSSConfig struct {
	Source string `json:"source"`
}

func GetCSSSource(c *gin.Context) {
	source := "cdn"
	if redis.Client != nil {
		val, err := redis.Client.Get(c, cssSourceKey).Result()
		if err == nil && val != "" {
			source = val
		}
	}
	response.Success(c, gin.H{"source": source})
}

func SetCSSSource(c *gin.Context) {
	var config CSSConfig
	if err := c.ShouldBindJSON(&config); err != nil {
		response.Error(c, 400, "无效的请求参数")
		return
	}
	if config.Source != "cdn" && config.Source != "local" {
		response.Error(c, 400, "CSS来源必须是 cdn 或 local")
		return
	}
	if redis.Client != nil {
		redis.Client.Set(c, cssSourceKey, config.Source, 0)
	}
	response.Success(c, gin.H{
		"source":  config.Source,
		"message": "CSS来源已切换",
	})
}
