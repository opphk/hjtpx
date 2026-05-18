package handler

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
)

type APIInfo struct {
	Title          string                   `json:"title"`
	Version        string                   `json:"version"`
	Description    string                   `json:"description"`
	BaseURL        string                   `json:"base_url"`
	Endpoints      []EndpointInfo           `json:"endpoints"`
	Authentication AuthenticationInfo       `json:"authentication"`
	RateLimiting   RateLimitingInfo         `json:"rate_limiting"`
	ErrorCodes     map[string]ErrorCodeInfo `json:"error_codes"`
}

type EndpointInfo struct {
	Method      string           `json:"method"`
	Path        string           `json:"path"`
	Summary     string           `json:"summary"`
	Description string           `json:"description"`
	Parameters  []ParameterInfo  `json:"parameters,omitempty"`
	RequestBody *RequestBodyInfo `json:"request_body,omitempty"`
	Responses   []ResponseInfo   `json:"responses"`
	Examples    []ExampleInfo    `json:"examples,omitempty"`
	Security    []string         `json:"security,omitempty"`
	Tags        []string         `json:"tags,omitempty"`
}

type ParameterInfo struct {
	Name        string   `json:"name"`
	In          string   `json:"in"`
	Type        string   `json:"type"`
	Required    bool     `json:"required"`
	Description string   `json:"description"`
	Default     string   `json:"default,omitempty"`
	Enum        []string `json:"enum,omitempty"`
}

type RequestBodyInfo struct {
	Description string                 `json:"description"`
	ContentType string                 `json:"content_type"`
	Schema      map[string]interface{} `json:"schema"`
	Example     interface{}            `json:"example,omitempty"`
}

type ResponseInfo struct {
	StatusCode  int                    `json:"status_code"`
	Description string                 `json:"description"`
	Schema      map[string]interface{} `json:"schema,omitempty"`
	Example     interface{}            `json:"example,omitempty"`
}

type ExampleInfo struct {
	Title       string      `json:"title"`
	Description string      `json:"description"`
	Request     interface{} `json:"request,omitempty"`
	Response    interface{} `json:"response,omitempty"`
}

type AuthenticationInfo struct {
	Type        string        `json:"type"`
	Headers     []string      `json:"required_headers"`
	Description string        `json:"description"`
	Signature   SignatureInfo `json:"signature,omitempty"`
}

type SignatureInfo struct {
	Algorithm   string   `json:"algorithm"`
	Components  []string `json:"components"`
	Tolerance   string   `json:"tolerance"`
	NonceLength string   `json:"nonce_length"`
}

type RateLimitingInfo struct {
	DefaultLimit  string   `json:"default_limit"`
	Window        string   `json:"window"`
	Headers       []string `json:"response_headers"`
	ExcludedPaths []string `json:"excluded_paths"`
}

type ErrorCodeInfo struct {
	Code        string `json:"code"`
	HTTPStatus  int    `json:"http_status"`
	Message     string `json:"message"`
	Description string `json:"description"`
	Resolution  string `json:"resolution,omitempty"`
}

func GetAPIDocumentation(c *gin.Context) {
	doc := APIInfo{
		Title:       "HJTPX Captcha API",
		Version:     "1.0.0",
		Description: "高性能验证码服务API，支持滑动验证码、点击验证码、3D验证码等多种验证方式",
		BaseURL:     "/api/v1",
		Endpoints: []EndpointInfo{
			{
				Method:      "GET",
				Path:        "/captcha/slider",
				Summary:     "获取滑动验证码",
				Description: "生成并返回一个新的滑动验证码图片和滑块图片",
				Parameters: []ParameterInfo{
					{Name: "Authorization", In: "header", Type: "string", Required: true, Description: "签名认证令牌"},
					{Name: "X-Signature", In: "header", Type: "string", Required: true, Description: "请求签名"},
					{Name: "X-Timestamp", In: "header", Type: "int64", Required: true, Description: "Unix时间戳(秒)"},
					{Name: "X-Nonce", In: "header", Type: "string", Required: true, Description: "随机字符串,长度8-64位"},
				},
				Responses: []ResponseInfo{
					{
						StatusCode:  200,
						Description: "成功返回验证码数据",
						Schema: map[string]interface{}{
							"session_id":   "string - 会话ID",
							"image_url":    "string - 验证码图片(base64编码)",
							"puzzle_image": "string - 滑块图片(base64编码)",
							"target_x":     "int - 目标X坐标",
							"target_y":     "int - 目标Y坐标",
							"puzzle_y":     "int - 滑块Y坐标",
							"tolerance":    "int - 容差值",
						},
						Example: map[string]interface{}{
							"session_id":   "sess_1621234567890_1234",
							"image_url":    "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
							"puzzle_image": "data:image/png;base64,iVBORw0KGgoAAAANSUhEUgAA...",
							"target_x":     150,
							"target_y":     100,
							"puzzle_y":     100,
							"tolerance":    10,
						},
					},
					{
						StatusCode:  401,
						Description: "认证失败",
						Example: map[string]interface{}{
							"error":   "missing_signature",
							"message": "X-Signature header is required",
						},
					},
					{
						StatusCode:  429,
						Description: "请求过于频繁",
						Example: map[string]interface{}{
							"error":       "rate_limit_exceeded",
							"message":     "Too many requests",
							"retry_after": 60,
						},
					},
				},
				Examples: []ExampleInfo{
					{
						Title:       "成功获取滑动验证码",
						Description: "使用正确的签名获取滑动验证码",
						Request: map[string]interface{}{
							"headers": map[string]string{
								"Authorization": "Bearer eyJhbGciOiJIUzI1NiIsInR5cCI6IkpXVCJ9...",
								"X-Signature":   "a1b2c3d4e5f6...",
								"X-Timestamp":   "1621234567",
								"X-Nonce":       "random_nonce_string",
							},
						},
						Response: map[string]interface{}{
							"session_id":   "sess_1621234567890_1234",
							"image_url":    "data:image/png;base64,...",
							"puzzle_image": "data:image/png;base64,...",
							"target_x":     150,
							"target_y":     100,
						},
					},
				},
				Security: []string{"signature"},
				Tags:     []string{"captcha"},
			},
			{
				Method:      "GET",
				Path:        "/captcha/click",
				Summary:     "获取点击验证码",
				Description: "生成并返回一个点击式验证码，支持多种模式",
				Parameters: []ParameterInfo{
					{Name: "Authorization", In: "header", Type: "string", Required: true, Description: "签名认证令牌"},
					{Name: "X-Signature", In: "header", Type: "string", Required: true, Description: "请求签名"},
					{Name: "mode", In: "query", Type: "string", Required: false, Description: "验证码模式", Default: "number", Enum: []string{"number", "letter", "chinese", "mixed", "icon"}},
					{Name: "shuffle", In: "query", Type: "string", Required: false, Description: "是否打乱顺序", Default: "true", Enum: []string{"true", "false"}},
					{Name: "points", In: "query", Type: "int", Required: false, Description: "点击点数", Default: "3"},
					{Name: "lang", In: "query", Type: "string", Required: false, Description: "语言", Default: "en-US", Enum: []string{"zh-CN", "en-US"}},
				},
				Responses: []ResponseInfo{
					{
						StatusCode:  200,
						Description: "成功返回验证码数据",
						Example: map[string]interface{}{
							"session_id":    "sess_1621234567890_5678",
							"image_url":     "data:image/png;base64,...",
							"hint":          "Click in order: 3 → 7 → 5",
							"hint_order":    []int{2, 6, 4},
							"max_points":    3,
							"mode":          "number",
							"allow_shuffle": true,
							"language":      "en-US",
						},
					},
				},
				Security: []string{"signature"},
				Tags:     []string{"captcha"},
			},
			{
				Method:      "POST",
				Path:        "/captcha/verify",
				Summary:     "验证验证码",
				Description: "验证用户对验证码的操作是否正确",
				Parameters: []ParameterInfo{
					{Name: "Authorization", In: "header", Type: "string", Required: true, Description: "签名认证令牌"},
					{Name: "X-Signature", In: "header", Type: "string", Required: true, Description: "请求签名"},
					{Name: "Content-Type", In: "header", Type: "string", Required: true, Description: "application/json"},
				},
				RequestBody: &RequestBodyInfo{
					Description: "验证请求参数",
					ContentType: "application/json",
					Schema: map[string]interface{}{
						"session_id":       "string - 会话ID",
						"type":             "string - 验证码类型: slider/click",
						"x":                "int - 滑动X坐标(滑块验证时)",
						"y":                "int - 滑动Y坐标(滑块验证时)",
						"points":           "array - 点击坐标数组(点击验证时)",
						"click_sequence":   "array - 点击顺序",
						"behavior_data":    "array - 行为数据",
						"application_id":   "uint - 应用ID",
						"environment_data": "object - 环境检测数据",
					},
					Example: map[string]interface{}{
						"session_id": "sess_1621234567890_1234",
						"type":       "slider",
						"x":          150,
						"y":          100,
						"behavior_data": []map[string]interface{}{
							{"event": "mousedown", "timestamp": 1621234567001, "x": 10.5, "y": 20.3},
							{"event": "mousemove", "timestamp": 1621234567002, "x": 15.2, "y": 25.1},
						},
						"application_id": 1,
					},
				},
				Responses: []ResponseInfo{
					{
						StatusCode:  200,
						Description: "验证成功",
						Example: map[string]interface{}{
							"success":      true,
							"message":      "verification successful",
							"risk_score":   15.5,
							"captcha_pass": true,
						},
					},
					{
						StatusCode:  200,
						Description: "验证失败",
						Example: map[string]interface{}{
							"success":      false,
							"message":      "verification failed",
							"risk_score":   65.0,
							"captcha_pass": false,
							"fail_reason":  "slider position deviation too large",
						},
					},
					{
						StatusCode:  404,
						Description: "会话不存在",
						Example: map[string]interface{}{
							"success": false,
							"message": "session not found or expired",
						},
					},
				},
				Security: []string{"signature"},
				Tags:     []string{"captcha"},
			},
		},
		Authentication: AuthenticationInfo{
			Type:        "HMAC-SHA256 Signature",
			Headers:     []string{"Authorization", "X-Signature", "X-Timestamp", "X-Nonce"},
			Description: "所有API请求必须包含签名认证信息",
			Signature: SignatureInfo{
				Algorithm:   "HMAC-SHA256",
				Components:  []string{"HTTP_METHOD", "PATH", "QUERY_STRING", "TIMESTAMP", "NONCE", "BODY_HASH"},
				Tolerance:   "5 minutes",
				NonceLength: "8-64 characters",
			},
		},
		RateLimiting: RateLimitingInfo{
			DefaultLimit:  "100 requests",
			Window:        "per minute",
			Headers:       []string{"X-RateLimit-Limit", "X-RateLimit-Remaining", "X-RateLimit-Reset"},
			ExcludedPaths: []string{"/health", "/api/health", "/metrics"},
		},
		ErrorCodes: map[string]ErrorCodeInfo{
			"MISSING_SIGNATURE": {
				Code:        "MISSING_SIGNATURE",
				HTTPStatus:  401,
				Message:     "X-Signature header is required",
				Description: "请求缺少签名信息",
				Resolution:  "在请求头中添加X-Signature字段",
			},
			"SIGNATURE_MISMATCH": {
				Code:        "SIGNATURE_MISMATCH",
				HTTPStatus:  401,
				Message:     "Signature verification failed",
				Description: "签名验证失败",
				Resolution:  "检查签名算法和密钥是否正确",
			},
			"TIMESTAMP_EXPIRED": {
				Code:        "TIMESTAMP_EXPIRED",
				HTTPStatus:  401,
				Message:     "Timestamp out of tolerance",
				Description: "请求时间戳超出允许范围",
				Resolution:  "确保客户端时间与服务器时间同步",
			},
			"NONCE_INVALID": {
				Code:        "NONCE_INVALID",
				HTTPStatus:  429,
				Message:     "Nonce already used",
				Description: "Nonce被重复使用，可能是重放攻击",
				Resolution:  "每个请求使用新的随机Nonce",
			},
			"RATE_LIMIT_EXCEEDED": {
				Code:        "RATE_LIMIT_EXCEEDED",
				HTTPStatus:  429,
				Message:     "Too many requests",
				Description: "请求频率超出限制",
				Resolution:  "降低请求频率或申请提高限额",
			},
			"SESSION_EXPIRED": {
				Code:        "SESSION_EXPIRED",
				HTTPStatus:  404,
				Message:     "Session not found or expired",
				Description: "验证码会话不存在或已过期",
				Resolution:  "重新获取验证码",
			},
			"INVALID_PARAMETERS": {
				Code:        "INVALID_PARAMETERS",
				HTTPStatus:  400,
				Message:     "Invalid request parameters",
				Description: "请求参数无效",
				Resolution:  "检查请求参数格式和类型",
			},
		},
	}

	c.JSON(http.StatusOK, gin.H{
		"code":      0,
		"message":   "success",
		"data":      doc,
		"timestamp": time.Now().Unix(),
	})
}

func GetAPIQuickStart(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"title":       "快速开始指南",
			"description": "使用HJTPX API的基本流程",
			"steps": []map[string]interface{}{
				{
					"step":        1,
					"title":       "获取签名密钥",
					"description": "联系管理员获取API密钥",
					"note":        "生产环境请使用HTTPS传输",
				},
				{
					"step":        2,
					"title":       "构造签名",
					"description": "按照指定算法生成请求签名",
					"algorithm": gin.H{
						"input":   "METHOD\\nPATH\\nSORTED_QUERY\\nTIMESTAMP\\nNONCE\\nBODY_HASH",
						"example": "POST\\n/api/v1/captcha/verify\\ntimestamp=1621234567\\n1621234567\\nrandom123\\ne3b0c44298fc1c14",
						"sign":    "HMAC-SHA256(secret_key, input)",
					},
				},
				{
					"step":        3,
					"title":       "发送请求",
					"description": "携带签名信息发送API请求",
					"headers": gin.H{
						"Authorization": "Bearer <token>",
						"X-Signature":   "<calculated_signature>",
						"X-Timestamp":   "<unix_timestamp>",
						"X-Nonce":       "<random_string>",
					},
				},
				{
					"step":        4,
					"title":       "验证响应",
					"description": "检查响应状态码和业务状态",
					"success": gin.H{
						"http_code": 200,
						"success":   true,
					},
					"failure": gin.H{
						"http_code": 401,
						"error":     "signature_mismatch",
					},
				},
			},
			"example_code": gin.H{
				"go": `
package main

import (
    "crypto/hmac"
    "crypto/sha256"
    "encoding/hex"
    "fmt"
    "net/http"
    "strconv"
    "time"
)

func main() {
    secretKey := "your-secret-key"
    
    timestamp := strconv.FormatInt(time.Now().Unix(), 10)
    nonce := "random123"
    method := "GET"
    path := "/api/v1/captcha/slider"
    
    stringToSign := method + "\\n" + path + "\\n\\n" + timestamp + "\\n" + nonce + "\\n"
    
    mac := hmac.New(sha256.New, []byte(secretKey))
    mac.Write([]byte(stringToSign))
    signature := hex.EncodeToString(mac.Sum(nil))
    
    req, _ := http.NewRequest(method, path, nil)
    req.Header.Set("X-Signature", signature)
    req.Header.Set("X-Timestamp", timestamp)
    req.Header.Set("X-Nonce", nonce)
    
    client := &http.Client{}
    resp, _ := client.Do(req)
    defer resp.Body.Close()
}
				`,
				"python": `
import hmac
import hashlib
import time
import requests

secret_key = "your-secret-key"

timestamp = str(int(time.time()))
nonce = "random123"
method = "GET"
path = "/api/v1/captcha/slider"

string_to_sign = f"{method}\\n{path}\\n\\n{timestamp}\\n{nonce}\\n"

signature = hmac.new(
    secret_key.encode(),
    string_to_sign.encode(),
    hashlib.sha256
).hexdigest()

headers = {
    "X-Signature": signature,
    "X-Timestamp": timestamp,
    "X-Nonce": nonce,
}

response = requests.get(path, headers=headers)
				`,
			},
			"limits": gin.H{
				"rate_limit":    "100 requests/minute",
				"daily_quota":   "100,000 requests/day",
				"session_ttl":   "10 minutes",
				"max_file_size": "1MB",
			},
		},
		"timestamp": time.Now().Unix(),
	})
}

func GetAPISecurityGuide(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"title":        "安全使用指南",
			"version":      "1.0",
			"last_updated": "2024-01-01",
			"sections": []map[string]interface{}{
				{
					"title":       "1. 签名验证",
					"description": "所有API请求必须包含有效的HMAC-SHA256签名",
					"key_points": []string{
						"使用安全的随机数生成器创建Nonce",
						"Nonce长度建议32字符以上",
						"时间戳必须与服务器时间误差在5分钟内",
						"签名计算包含: 方法、路径、查询参数、时间戳、Nonce、请求体哈希",
					},
					"example": gin.H{
						"string_to_sign": "POST\\n/api/v1/captcha/verify\\n\\n1621234567\\nrandom_nonce\\ne3b0c44298fc1c14",
						"signature":      "hmac_sha256(secret_key, string_to_sign)",
					},
				},
				{
					"title":       "2. 防重放机制",
					"description": "防止请求被恶意重复发送",
					"key_points": []string{
						"Nonce在24小时内不能重复使用",
						"时间戳超出容忍范围将被拒绝",
						"使用Bloom Filter快速检测重复Nonce",
						"可配合Redis实现分布式Nonce存储",
					},
				},
				{
					"title":       "3. 请求加密",
					"description": "敏感请求建议使用加密传输",
					"key_points": []string{
						"生产环境强制使用HTTPS",
						"可选启用请求体AES-256-GCM加密",
						"支持密钥轮换",
						"加密请求使用X-Encrypted头标识",
					},
				},
				{
					"title":       "4. CSRF防护",
					"description": "防止跨站请求伪造攻击",
					"key_points": []string{
						"GET请求自动生成CSRF Token",
						"POST/PUT/DELETE请求必须携带有效Token",
						"Token存储在Cookie和服务器端",
						"Token有效期为1小时",
					},
				},
				{
					"title":       "5. 最佳实践",
					"description": "安全使用API的建议",
					"recommendations": []string{
						"密钥定期轮换(建议每月)",
						"监控异常请求模式",
						"记录所有API访问日志",
						"使用IP白名单限制访问",
						"实施最小权限原则",
						"启用详细审计日志",
					},
					"warnings": []string{
						"禁止在客户端代码中硬编码密钥",
						"禁止在URL中传递敏感参数",
						"禁止记录完整签名信息",
						"禁止忽略SSL证书验证",
					},
				},
			},
		},
		"timestamp": time.Now().Unix(),
	})
}

func GetAPIChangelog(c *gin.Context) {
	c.JSON(http.StatusOK, gin.H{
		"code":    0,
		"message": "success",
		"data": gin.H{
			"current_version": "1.0.0",
			"changelog": []map[string]interface{}{
				{
					"version": "1.0.0",
					"date":    "2024-01-01",
					"type":    "major",
					"changes": []string{
						"初始版本发布",
						"支持滑动验证码",
						"支持点击验证码",
						"支持3D验证码",
						"实现HMAC-SHA256签名验证",
						"实现防重放机制",
						"实现限流保护",
					},
				},
				{
					"version": "1.1.0",
					"date":    "2024-02-01",
					"type":    "minor",
					"changes": []string{
						"新增语音验证码",
						"优化图片生成性能",
						"增强行为分析算法",
						"添加更多安全头部",
					},
				},
			},
		},
		"timestamp": time.Now().Unix(),
	})
}
