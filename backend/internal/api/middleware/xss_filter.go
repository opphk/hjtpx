package middleware

import (
	"html"
	"net/url"
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
)

var xssPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`),
	regexp.MustCompile(`(?i)<script[^>]*>`),
	regexp.MustCompile(`(?i)javascript:`),
	regexp.MustCompile(`(?i)on\w+\s*=`),
	regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`),
	regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`),
	regexp.MustCompile(`(?i)<embed[^>]*>`),
	regexp.MustCompile(`(?i)<applet[^>]*>.*?</applet>`),
	regexp.MustCompile(`(?i)expression\s*\(`),
	regexp.MustCompile(`(?i)<xml[^>]*>.*?</xml>`),
	regexp.MustCompile(`(?i)data:`),
	regexp.MustCompile(`(?i)vbscript:`),
}

var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union|select|insert|update|delete|drop|alter|create|truncate|exec|execute|script)\s+`),
	regexp.MustCompile(`(?i)'|(--)|(/\*)|(\*/)|(@@)|(\bOR\b.*=.*\bOR\b)`),
	regexp.MustCompile(`(?i)\bOR\b\s+\d+\s*=\s*\d+`),
	regexp.MustCompile(`(?i)'\s*(OR|AND)\s+'`),
	regexp.MustCompile(`(?i)(union|select).*(from|where)`),
	regexp.MustCompile(`(?i)(concat|char|ascii|hex|unhex)\s*\(`),
	regexp.MustCompile(`(?i)(sleep|benchmark|waitfor)\s*(`),
	regexp.MustCompile(`(?i)(load_file|into\s+outfile|into\s+dumpfile)`),
}

type XSSConfig struct {
	Enabled        bool
	LogBlocked     bool
	EscapeHTML     bool
	StrictMode     bool
	AllowedTags    []string
	MaxInputLength int
}

var DefaultXSSConfig = XSSConfig{
	Enabled:        true,
	LogBlocked:     true,
	EscapeHTML:     true,
	StrictMode:     false,
	AllowedTags:    []string{"p", "br", "b", "i", "u", "strong", "em"},
	MaxInputLength: 10000,
}

func XSSFilterMiddleware(config ...XSSConfig) gin.HandlerFunc {
	cfg := DefaultXSSConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		if c.Request.Body != nil && c.Request.ContentLength > 0 && c.Request.ContentLength < int64(cfg.MaxInputLength) {
			bodyBytes, _ := readBody(c)
			if len(bodyBytes) > 0 {
				originalBody := string(bodyBytes)
				filtered := FilterXSS(originalBody, cfg)
				if filtered != originalBody && cfg.LogBlocked {
					c.Set("xss_blocked", true)
					c.Set("xss_original", originalBody)
					c.Set("xss_filtered", filtered)
				}
				c.Request.Body = newBodyReader(filtered)
			}
		}

		c.SetQuery("xss_filtered", "true")
		processQueryParams(c, cfg)
		processHeaders(c, cfg)

		c.Next()
	}
}

func FilterXSS(input string, config XSSConfig) string {
	if input == "" {
		return input
	}

	if config.EscapeHTML {
		for _, pattern := range xssPatterns {
			if pattern.MatchString(input) {
				if config.StrictMode {
					return ""
				}
				input = pattern.ReplaceAllString(input, "")
			}
		}
	}

	input = html.EscapeString(input)
	return input
}

func FilterSQL(input string) string {
	if input == "" {
		return input
	}

	for _, pattern := range sqlInjectionPatterns {
		input = pattern.ReplaceAllString(input, " ")
	}

	return strings.TrimSpace(input)
}

func CheckXSS(input string) (bool, string) {
	for _, pattern := range xssPatterns {
		if pattern.MatchString(input) {
			return true, pattern.String()
		}
	}
	return false, ""
}

func CheckSQLInjection(input string) (bool, string) {
	for _, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(input) {
			return true, pattern.String()
		}
	}
	return false, ""
}

func SanitizeAllUserInputs(c *gin.Context) map[string]interface{} {
	sanitized := make(map[string]interface{})

	for key, values := range c.Request.URL.Query() {
		sanitizedValues := make([]string, len(values))
		for i, v := range values {
			sanitizedValues[i] = FilterXSS(v, DefaultXSSConfig)
		}
		sanitized[key] = sanitizedValues
	}

	if err := c.Request.ParseForm(); err == nil {
		for key, values := range c.Request.PostForm {
			sanitizedValues := make([]string, len(values))
			for i, v := range values {
				sanitizedValues[i] = FilterXSS(v, DefaultXSSConfig)
			}
			sanitized[key] = sanitizedValues
		}
	}

	return sanitized
}

func GetXSSBlockedRequests() []*XSSBlockedRequest {
	return xssBlockedRequests
}

func readBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	buf := new(strings.Builder)
	_, err := buf.ReadFrom(c.Request.Body)
	if err != nil {
		return nil, err
	}
	return []byte(buf.String()), nil
}

func newBodyReader(content string) *bodyReaderCloser {
	return &bodyReaderCloser{content: content, index: 0}
}

type bodyReaderCloser struct {
	content string
	index  int
}

func (b *bodyReaderCloser) Read(p []byte) (n int, err error) {
	if b.index >= len(b.content) {
		return 0, nil
	}
	n = copy(p, b.content[b.index:])
	b.index += n
	return n, nil
}

func (b *bodyReaderCloser) Close() error {
	return nil
}

func processQueryParams(c *gin.Context, config XSSConfig) {
	query := c.Request.URL.Query()
	sanitized := make(url.Values)

	for key, values := range query {
		sanitizedValues := make([]string, len(values))
		for i, v := range values {
			sanitizedValues[i] = FilterXSS(v, config)
		}
		sanitized[key] = sanitizedValues
	}

	c.Request.URL.RawQuery = sanitized.Encode()
}

func processHeaders(c *gin.Context, config XSSConfig) {
	headerNames := []string{"X-User-Data", "X-Client-Info", "X-Referer", "X-User-Agent"}

	for _, headerName := range headerNames {
		headerValue := c.GetHeader(headerName)
		if headerValue != "" {
			filtered := FilterXSS(headerValue, config)
			if filtered != headerValue {
				c.Request.Header.Set(headerName, filtered)
			}
		}
	}
}

type XSSBlockedRequest struct {
	Path      string
	Method    string
	Original  string
	Filtered  string
	Timestamp string
	ClientIP  string
}

var xssBlockedRequests []*XSSBlockedRequest

func init() {
	xssBlockedRequests = make([]*XSSBlockedRequest, 0)
}

type SQLInjectionConfig struct {
	Enabled        bool
	LogBlocked     bool
	BlockRequests  bool
	CustomPatterns []*regexp.Regexp
}

var DefaultSQLInjectionConfig = SQLInjectionConfig{
	Enabled:       true,
	LogBlocked:    true,
	BlockRequests: false,
	CustomPatterns: nil,
}

func SQLInjectionDetectionMiddleware(config ...SQLInjectionConfig) gin.HandlerFunc {
	cfg := DefaultSQLInjectionConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		if !cfg.Enabled {
			c.Next()
			return
		}

		query := c.Request.URL.RawQuery
		if query != "" {
			if detected, pattern := CheckSQLInjection(query); detected {
				if cfg.LogBlocked {
					c.Set("sql_injection_detected", true)
					c.Set("sql_injection_pattern", pattern)
				}
				if cfg.BlockRequests {
					c.AbortWithStatusJSON(400, gin.H{
						"error":   "potential sql injection detected",
						"message": "Request contains suspicious SQL patterns",
					})
					return
				}
			}
		}

		if c.Request.Body != nil {
			bodyBytes, _ := readBody(c)
			if len(bodyBytes) > 0 {
				bodyStr := string(bodyBytes)
				if detected, pattern := CheckSQLInjection(bodyStr); detected {
					if cfg.LogBlocked {
						c.Set("sql_injection_detected", true)
						c.Set("sql_injection_pattern", pattern)
					}
					if cfg.BlockRequests {
						c.AbortWithStatusJSON(400, gin.H{
							"error":   "potential sql injection detected",
							"message": "Request body contains suspicious SQL patterns",
						})
						return
					}
				}
			}
		}

		for _, headerName := range []string{"X-User-Data", "X-Client-Info"} {
			headerValue := c.GetHeader(headerName)
			if headerValue != "" {
				if detected, _ := CheckSQLInjection(headerValue); detected {
					if cfg.LogBlocked {
						c.Set("sql_injection_in_header", headerName)
					}
				}
			}
		}

		c.Next()
	}
}

type BlacklistConfig struct {
	Enabled             bool
	IPBlacklist         map[string]time.Time
	PathBlacklist       []string
	UserAgentBlacklist  []string
	AutoBlock           bool
	BlockDuration       time.Duration
}

func NewBlacklistConfig() *BlacklistConfig {
	return &BlacklistConfig{
		Enabled:             true,
		IPBlacklist:         make(map[string]time.Time),
		PathBlacklist:       []string{"/.env", "/.git", "/admin/config"},
		UserAgentBlacklist:  []string{"sqlmap", "nikto", "nmap"},
		AutoBlock:           true,
		BlockDuration:       1 * time.Hour,
	}
}
