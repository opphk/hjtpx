package middleware

import (
	"crypto/rand"
	"encoding/hex"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
)

func RequestIDMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		requestID := c.GetHeader("X-Request-ID")
		if requestID == "" {
			requestID = generateRequestID()
		}

		c.Set("request_id", requestID)
		c.Header("X-Request-ID", requestID)

		c.Next()
	}
}

func generateRequestID() string {
	id := make([]byte, 16)
	_, err := rand.Read(id)
	if err != nil {
		return uuid.New().String()
	}
	return hex.EncodeToString(id)
}

func RequestLoggingMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		start := time.Now()
		path := c.Request.URL.Path
		query := c.Request.URL.RawQuery

		c.Next()

		latency := time.Since(start)
		status := c.Writer.Status()

		requestID, _ := c.Get("request_id")
		rid, _ := requestID.(string)

		method := c.Request.Method
		clientIP := c.ClientIP()

		if query != "" {
			path = path + "?" + query
		}

		if status >= 500 {
			gin.DefaultErrorWriter.Write([]byte(
				time.Now().Format("2006/01/02 - 15:04:05") +
					" | " + rid +
					" | " + latency.String() +
					" | " + clientIP +
					" | " + method +
					" | " + path +
					" | " + strconv.Itoa(status) + "\n",
			))
		} else if status >= 400 {
			gin.DefaultErrorWriter.Write([]byte(
				time.Now().Format("2006/01/02 - 15:04:05") +
					" | " + rid +
					" | " + latency.String() +
					" | " + clientIP +
					" | " + method +
					" | " + path +
					" | " + strconv.Itoa(status) + "\n",
			))
		}
	}
}
