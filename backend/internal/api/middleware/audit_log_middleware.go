package middleware

import (
	"bytes"
	"encoding/json"
	"io"
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
)

type bodyLogWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w bodyLogWriter) Write(b []byte) (int, error) {
	w.body.Write(b)
	return w.ResponseWriter.Write(b)
}

func AuditLogMiddleware() gin.HandlerFunc {
	auditService := service.NewAuditLogService()

	return func(c *gin.Context) {
		startTime := time.Now()

		var requestBody string
		if c.Request.Body != nil {
			body, err := io.ReadAll(c.Request.Body)
			if err == nil {
				requestBody = string(body)
				c.Request.Body = io.NopCloser(bytes.NewBuffer(body))
			}
		}

		blw := &bodyLogWriter{body: bytes.NewBufferString(""), ResponseWriter: c.Writer}
		c.Writer = blw

		c.Next()

		responseTime := time.Since(startTime)

		userID := uint(0)
		username := ""
		if claims, exists := c.Get("claims"); exists {
			if tokenClaims, ok := claims.(*service.TokenClaims); ok {
				userID = tokenClaims.AdminID
				username = tokenClaims.Username
			}
		}

		entry := &service.AuditLogEntry{
			Timestamp:    time.Now(),
			UserID:       userID,
			Username:     username,
			IPAddress:    c.ClientIP(),
			UserAgent:    c.Request.UserAgent(),
			Endpoint:     c.Request.URL.Path,
			Method:       c.Request.Method,
			StatusCode:   c.Writer.Status(),
			ResponseTime: responseTime.Milliseconds(),
			RequestData:  requestBody,
			ResponseData: blw.body.String(),
			ResourceType: extractResourceType(c.Request.URL.Path),
			ResourceID:   extractResourceID(c.Request.URL.Path, c.Params),
			Action:       extractAction(c.Request.Method),
		}

		if c.Writer.Status() >= 400 {
			entry.Error = blw.body.String()
		}

		go func() {
			auditService.LogRequest(entry)
		}()
	}
}

func extractResourceType(path string) string {
	switch {
	case contains(path, "/admin/"):
		return "admin"
	case contains(path, "/applications/"):
		return "application"
	case contains(path, "/users/"):
		return "user"
	case contains(path, "/tenants/"):
		return "tenant"
	case contains(path, "/oauth2/"):
		return "oauth2"
	case contains(path, "/oidc/"):
		return "oidc"
	case contains(path, "/sso/"):
		return "sso"
	case contains(path, "/scim/"):
		return "scim"
	case contains(path, "/captcha/"):
		return "captcha"
	case contains(path, "/api/"):
		return "api"
	default:
		return "unknown"
	}
}

func extractResourceID(path string, params gin.Params) string {
	for _, param := range params {
		if param.Key == "id" || param.Key == "user_id" || param.Key == "tenant_id" {
			return param.Value
		}
	}
	return ""
}

func extractAction(method string) string {
	switch method {
	case http.MethodGet:
		return "read"
	case http.MethodPost:
		return "create"
	case http.MethodPut:
		return "update"
	case http.MethodPatch:
		return "update"
	case http.MethodDelete:
		return "delete"
	default:
		return method
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s[:len(substr)] == substr || containsHelper(s[1:], substr))
}

func containsHelper(s, substr string) bool {
	if len(s) < len(substr) {
		return false
	}
	if s[:len(substr)] == substr {
		return true
	}
	return containsHelper(s[1:], substr)
}

func SecurityAuditMiddleware() gin.HandlerFunc {
	auditService := service.NewAuditLogService()

	return func(c *gin.Context) {
		c.Next()

		if c.Writer.Status() == http.StatusUnauthorized {
			metadata := map[string]interface{}{
				"endpoint":    c.Request.URL.Path,
				"method":      c.Request.Method,
				"user_agent":  c.Request.UserAgent(),
				"client_ip":   c.ClientIP(),
				"timestamp":   time.Now().Format(time.RFC3339),
			}

			go func() {
				auditService.LogSecurityEvent("unauthorized_access", "Unauthorized access attempt", 0, c.ClientIP(), metadata)
			}()
		}

		if c.Writer.Status() == http.StatusForbidden {
			userID := uint(0)
			if claims, exists := c.Get("claims"); exists {
				if tokenClaims, ok := claims.(*service.TokenClaims); ok {
					userID = tokenClaims.AdminID
				}
			}

			metadata := map[string]interface{}{
				"endpoint":    c.Request.URL.Path,
				"method":      c.Request.Method,
				"user_agent":  c.Request.UserAgent(),
				"client_ip":   c.ClientIP(),
				"timestamp":   time.Now().Format(time.RFC3339),
			}

			go func() {
				auditService.LogSecurityEvent("access_denied", "Access denied to resource", userID, c.ClientIP(), metadata)
			}()
		}
	}
}

func AuthenticationAuditMiddleware() gin.HandlerFunc {
	auditService := service.NewAuditLogService()

	return func(c *gin.Context) {
		c.Next()

		if c.Request.URL.Path == "/api/v1/admin/login" {
			if c.Writer.Status() == http.StatusOK {
				var loginRequest struct {
					Username string `json:"username"`
				}
				if body, err := io.ReadAll(c.Request.Body); err == nil {
					json.Unmarshal(body, &loginRequest)
					c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

					go func() {
						auditService.LogAuthentication(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), true, "")
					}()
				}
			} else if c.Writer.Status() == http.StatusUnauthorized {
				var loginRequest struct {
					Username string `json:"username"`
				}
				if body, err := io.ReadAll(c.Request.Body); err == nil {
					json.Unmarshal(body, &loginRequest)
					c.Request.Body = io.NopCloser(bytes.NewBuffer(body))

					go func() {
						auditService.LogAuthentication(loginRequest.Username, c.ClientIP(), c.Request.UserAgent(), false, "Invalid credentials")
					}()
				}
			}
		}
	}
}