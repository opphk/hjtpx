package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"log"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

type LogData struct {
	Timestamp  string      `json:"timestamp"`
	Level      string      `json:"level"`
	RequestID  string      `json:"request_id"`
	Method     string      `json:"method"`
	Path       string      `json:"path"`
	StatusCode int         `json:"status_code"`
	Latency    string      `json:"latency"`
	IP         string      `json:"ip"`
	UserAgent  string      `json:"user_agent"`
	Body       interface{} `json:"body,omitempty"`
	Error      string      `json:"error,omitempty"`
}

var logger = log.Default()

func RequestLogger() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		requestID := uuid.New().String()
		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		var bodyBytes []byte
		if c.Request.Body != nil {
			bodyBytes, _ = io.ReadAll(c.Request.Body)
			c.Request.Body = io.NopCloser(bytes.NewBuffer(bodyBytes))
		}

		c.Next()

		latency := time.Since(start)
		level := "INFO"
		errorMsg := ""

		if c.Writer.Status() >= 500 {
			level = "ERROR"
		} else if c.Writer.Status() >= 400 {
			level = "WARN"
		}

		if len(c.Errors) > 0 {
			errorMsg = c.Errors.String()
		}

		logData := LogData{
			Timestamp:  start.Format(time.RFC3339),
			Level:      level,
			RequestID:  requestID,
			Method:     c.Request.Method,
			Path:       c.Request.URL.Path,
			StatusCode: c.Writer.Status(),
			Latency:    latency.String(),
			IP:         c.ClientIP(),
			UserAgent:  c.Request.UserAgent(),
			Error:      errorMsg,
		}

		if len(bodyBytes) > 0 && len(bodyBytes) < 1000 {
			var body interface{}
			json.Unmarshal(bodyBytes, &body)
			logData.Body = body
		}

		logJSON, _ := json.Marshal(logData)
		logger.Printf("%s", logJSON)
	}
}

func Recovery() gin.HandlerFunc {
	return func(c *gin.Context) {
		defer func() {
			if err := recover(); err != nil {
				requestID, _ := c.Get("request_id")
				logger.Printf("[PANIC] request_id=%v error=%v", requestID, err)
				c.JSON(500, gin.H{
					"code":       500,
					"message":    "internal server error",
					"request_id": requestID,
				})
				c.Abort()
			}
		}()
		c.Next()
	}
}

func CORS() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("Access-Control-Allow-Origin", "*")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		c.Header("Access-Control-Allow-Headers", "Origin, Content-Type, Authorization, X-Request-ID")
		c.Header("Access-Control-Expose-Headers", "X-Request-ID")

		if c.Request.Method == "OPTIONS" {
			c.AbortWithStatus(204)
			return
		}

		c.Next()
	}
}

func init() {
	log.SetFlags(log.LstdFlags | log.Lshortfile)
}
