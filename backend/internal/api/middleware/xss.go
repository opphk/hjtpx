package middleware

import (
	"bytes"
	"fmt"
	"html"
	"html/template"
	"io"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

type XSSConfig struct {
	EnableLog       bool
	AllowedTags     []string
	BlockAttributes bool
	MaxLength       int
}

var defaultXSSConfig = XSSConfig{
	EnableLog:       true,
	AllowedTags:     []string{"p", "br", "strong", "em", "u", "h1", "h2", "h3", "h4", "h5", "h6", "ul", "ol", "li", "a", "img"},
	BlockAttributes: false,
	MaxLength:       10000,
}

var (
	scriptPattern     = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
	stylePattern      = regexp.MustCompile(`(?i)<style[^>]*>.*?</style>`)
	iframePattern     = regexp.MustCompile(`(?i)<iframe[^>]*>.*?</iframe>`)
	objectPattern     = regexp.MustCompile(`(?i)<object[^>]*>.*?</object>`)
	embedPattern      = regexp.MustCompile(`(?i)<embed[^>]*>`)
	appletPattern     = regexp.MustCompile(`(?i)<applet[^>]*>.*?</applet>`)
	formPattern       = regexp.MustCompile(`(?i)<form[^>]*>.*?</form>`)
	inputPattern      = regexp.MustCompile(`(?i)<input[^>]*>`)
	textareaPattern   = regexp.MustCompile(`(?i)<textarea[^>]*>.*?</textarea>`)
	selectPattern     = regexp.MustCompile(`(?i)<select[^>]*>.*?</select>`)
	eventAttrPattern  = regexp.MustCompile(`(?i)\s+on\w+\s*=\s*(?:"[^"]*"|'[^']*'|[^\s>]*)`)
	javascriptPattern = regexp.MustCompile(`(?i)javascript\s*:`)
	dataAttrPattern   = regexp.MustCompile(`(?i)\s+data-[\w-]+\s*=\s*["'][^"']*["']`)
	expressionPattern = regexp.MustCompile(`(?i)expression\s*\(`)
	urlPattern        = regexp.MustCompile(`(?i)\b(javascript|vbscript|data)\s*:`)
	styleExprPattern  = regexp.MustCompile(`(?i)expression\s*\([^)]*\)`)
	xmlPattern        = regexp.MustCompile(`(?i)<\?xml[^>]*\?>`)
)

type sanitizedValue struct {
	Value    string
	IsUnsafe bool
}

func sanitizeHTML(input string, cfg XSSConfig) sanitizedValue {
	if len(input) > cfg.MaxLength {
		input = input[:cfg.MaxLength]
	}

	result := input

	result = scriptPattern.ReplaceAllString(result, "")
	result = stylePattern.ReplaceAllString(result, "")
	result = iframePattern.ReplaceAllString(result, "")
	result = objectPattern.ReplaceAllString(result, "")
	result = appletPattern.ReplaceAllString(result, "")
	result = formPattern.ReplaceAllString(result, "")
	result = inputPattern.ReplaceAllString(result, "")
	result = textareaPattern.ReplaceAllString(result, "")
	result = selectPattern.ReplaceAllString(result, "")
	result = embedPattern.ReplaceAllString(result, "")

	result = eventAttrPattern.ReplaceAllString(result, "")
	result = javascriptPattern.ReplaceAllString(result, "")
	result = urlPattern.ReplaceAllString(result, "")
	result = expressionPattern.ReplaceAllString(result, "")
	result = styleExprPattern.ReplaceAllString(result, "")
	result = xmlPattern.ReplaceAllString(result, "")

	if cfg.BlockAttributes {
		result = dataAttrPattern.ReplaceAllString(result, "")
	}

	isUnsafe := result != input

	return sanitizedValue{
		Value:    result,
		IsUnsafe: isUnsafe,
	}
}

func sanitizeHTMLWithAllowList(input string, cfg XSSConfig) string {
	if len(input) > cfg.MaxLength {
		input = input[:cfg.MaxLength]
	}

	result := input

	result = scriptPattern.ReplaceAllString(result, "")
	result = stylePattern.ReplaceAllString(result, "")
	result = iframePattern.ReplaceAllString(result, "")
	result = objectPattern.ReplaceAllString(result, "")
	result = appletPattern.ReplaceAllString(result, "")
	result = formPattern.ReplaceAllString(result, "")
	result = inputPattern.ReplaceAllString(result, "")
	result = textareaPattern.ReplaceAllString(result, "")
	result = selectPattern.ReplaceAllString(result, "")
	result = embedPattern.ReplaceAllString(result, "")

	result = eventAttrPattern.ReplaceAllString(result, "")
	result = javascriptPattern.ReplaceAllString(result, "")
	result = urlPattern.ReplaceAllString(result, "")
	result = expressionPattern.ReplaceAllString(result, "")
	result = styleExprPattern.ReplaceAllString(result, "")
	result = xmlPattern.ReplaceAllString(result, "")

	if cfg.BlockAttributes {
		result = dataAttrPattern.ReplaceAllString(result, "")
	}

	allowedTagsMap := make(map[string]bool)
	for _, tag := range cfg.AllowedTags {
		allowedTagsMap[strings.ToLower(tag)] = true
	}

	var finalResult strings.Builder
	var inTag bool
	var tagBuffer strings.Builder
	var tagNameBuffer strings.Builder

	for i := 0; i < len(result); i++ {
		c := result[i]

		if c == '<' && !inTag {
			inTag = true
			tagBuffer.Reset()
			tagNameBuffer.Reset()
			continue
		}

		if c == '>' && inTag {
			inTag = false
			rawTag := tagBuffer.String()
			tagBuffer.Reset()

			rawTagLower := strings.ToLower(rawTag)
			isClosingTag := strings.HasPrefix(rawTagLower, "/")

			var cleanTagName string
			if isClosingTag {
				cleanTagName = strings.TrimPrefix(rawTagLower, "/")
				cleanTagName = strings.Split(cleanTagName, " ")[0]
				cleanTagName = strings.Split(cleanTagName, ">")[0]
				cleanTagName = strings.TrimSpace(cleanTagName)
			} else {
				parts := strings.SplitN(rawTagLower, " ", 2)
				cleanTagName = strings.Split(parts[0], ">")[0]
				cleanTagName = strings.TrimSpace(cleanTagName)
			}

			if allowedTagsMap[cleanTagName] {
				if !isClosingTag {
					finalResult.WriteByte('<')
					finalResult.WriteString(rawTag)
					finalResult.WriteByte('>')
				} else {
					finalResult.WriteString("</")
					finalResult.WriteString(tagNameBuffer.String())
					finalResult.WriteByte('>')
				}
			} else {
				finalResult.WriteString("&lt;")
				finalResult.WriteString(html.EscapeString(rawTag))
				finalResult.WriteString("&gt;")
			}
			continue
		}

		if inTag {
			tagBuffer.WriteByte(c)
			if tagNameBuffer.Len() == 0 && c != '/' && c != ' ' {
				tagNameBuffer.WriteByte(c)
			} else if tagNameBuffer.Len() > 0 && c == ' ' {
			} else if tagNameBuffer.Len() > 0 && c != '/' {
				tagNameBuffer.WriteByte(c)
			}
		} else {
			finalResult.WriteByte(c)
		}
	}

	if inTag {
		finalResult.WriteString("&lt;")
		finalResult.WriteString(html.EscapeString(tagBuffer.String()))
	}

	return finalResult.String()
}

type XSSFilteredBody struct {
	gin.ResponseWriter
	body       *bytes.Buffer
	cfg        XSSConfig
	path       string
	method     string
}

func (w *XSSFilteredBody) Write(b []byte) (int, error) {
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		sanitized := sanitizeHTMLWithAllowList(string(b), w.cfg)
		return w.ResponseWriter.Write([]byte(sanitized))
	}
	return w.ResponseWriter.Write(b)
}

func (w *XSSFilteredBody) WriteString(s string) (int, error) {
	contentType := w.Header().Get("Content-Type")
	if strings.Contains(contentType, "text/html") || strings.Contains(contentType, "application/xhtml") {
		sanitized := sanitizeHTMLWithAllowList(s, w.cfg)
		return w.ResponseWriter.Write([]byte(sanitized))
	}
	return w.ResponseWriter.Write([]byte(s))
}

func sanitizeRequestBody(c *gin.Context, cfg XSSConfig) {
	if c.Request.Body == nil {
		return
	}

	contentType := c.GetHeader("Content-Type")
	if contentType == "" {
		return
	}

	if !strings.Contains(contentType, "application/json") &&
		!strings.Contains(contentType, "application/x-www-form-urlencoded") &&
		!strings.Contains(contentType, "multipart/form-data") {
		return
	}

	if strings.Contains(contentType, "application/json") {
		sanitizeJSONBody(c, cfg)
	}
}

func sanitizeJSONBody(c *gin.Context, cfg XSSConfig) {
	bodyBytes, err := readBody(c)
	if err != nil {
		return
	}

	bodyStr := string(bodyBytes)
	sanitized := sanitizeHTML(bodyStr, cfg)

	if sanitized.IsUnsafe && cfg.EnableLog {
		logXSSAttempt(c, "json_body", sanitized.Value)
	}
}

func readBody(c *gin.Context) ([]byte, error) {
	if c.Request.Body == nil {
		return nil, nil
	}
	body, err := c.GetRawData()
	if err != nil {
		return nil, err
	}
	c.Request.Body = createBodyReader(body)
	return body, nil
}

type readerCloser struct {
	*bytes.Reader
}

func (rc *readerCloser) Close() error {
	return nil
}

func createBodyReader(data []byte) io.ReadCloser {
	return &readerCloser{Reader: bytes.NewReader(data)}
}

func logXSSAttempt(c *gin.Context, field string, value string) {
	clientIP := c.ClientIP()
	method := c.Request.Method
	path := c.Request.URL.Path
	userAgent := c.GetHeader("User-Agent")

	fmt.Printf("[XSS_BLOCKED] %s | %s %s | IP: %s | Field: %s | UA: %s\n",
		method,
		path,
		c.Request.URL.RawQuery,
		clientIP,
		field,
		userAgent,
	)

	if value != "" {
		displayValue := value
		if len(displayValue) > 200 {
			displayValue = displayValue[:200] + "..."
		}
		fmt.Printf("[XSS_BLOCKED] Sanitized value: %s\n", displayValue)
	}
}

func XSSFilter(config ...XSSConfig) gin.HandlerFunc {
	cfg := defaultXSSConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		safePaths := map[string]bool{
			"/health":         true,
			"/api/health":     true,
			"/metrics":        true,
			"/api/metrics":    true,
		}

		if safePaths[path] {
			c.Next()
			return
		}

		sanitizeRequestBody(c, cfg)

		bodyBuffer := bytes.NewBuffer(nil)
		filteredWriter := &XSSFilteredBody{
			ResponseWriter: c.Writer,
			body:           bodyBuffer,
			cfg:            cfg,
			path:           path,
			method:         c.Request.Method,
		}
		c.Writer = filteredWriter

		c.Next()

		if filteredWriter.body.Len() > 0 {
			c.Header("X-XSS-Protection", "1; mode=block")
			c.Header("X-Content-Type-Options", "nosniff")
		}
	}
}

func SanitizeString(input string) string {
	result := sanitizeHTML(input, defaultXSSConfig)
	return html.EscapeString(result.Value)
}

func SanitizeStringWithConfig(input string, cfg XSSConfig) string {
	result := sanitizeHTML(input, cfg)
	return html.EscapeString(result.Value)
}

func SanitizeJSONResponse(data interface{}) interface{} {
	switch v := data.(type) {
	case string:
		return sanitizeHTMLWithAllowList(v, defaultXSSConfig)
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = SanitizeJSONResponse(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = SanitizeJSONResponse(item)
		}
		return result
	default:
		return v
	}
}

func EscapeHTML(input string) string {
	return template.HTMLEscapeString(input)
}

func UnescapeHTML(input string) string {
	return html.UnescapeString(input)
}

func isXSSAttempt(input string) bool {
	sanitized := sanitizeHTML(input, defaultXSSConfig)
	return sanitized.IsUnsafe
}

type XSSReport struct {
	Timestamp     string `json:"timestamp"`
	ClientIP      string `json:"client_ip"`
	Path          string `json:"path"`
	Method        string `json:"method"`
	Field         string `json:"field"`
	OriginalValue string `json:"original_value"`
	Blocked       bool   `json:"blocked"`
}

func GetXSSReport(c *gin.Context, field string, value string) XSSReport {
	return XSSReport{
		Timestamp:     c.GetHeader("X-Request-ID"),
		ClientIP:      c.ClientIP(),
		Path:          c.Request.URL.Path,
		Method:        c.Request.Method,
		Field:         field,
		OriginalValue: value,
		Blocked:       isXSSAttempt(value),
	}
}

func AddSecurityHeaders() gin.HandlerFunc {
	return func(c *gin.Context) {
		c.Header("X-Frame-Options", "SAMEORIGIN")
		c.Header("X-XSS-Protection", "1; mode=block")
		c.Header("X-Content-Type-Options", "nosniff")
		c.Header("Referrer-Policy", "strict-origin-when-cross-origin")
		c.Header("Strict-Transport-Security", "max-age=31536000; includeSubDomains")
		c.Header("Content-Security-Policy", "default-src 'self'; script-src 'self' https://cdn.bootcdn.net; style-src 'self' 'unsafe-inline' https://cdn.bootcdn.net; font-src 'self' https://cdn.bootcdn.net; img-src 'self' data: https:; connect-src 'self'; frame-ancestors 'self'; form-action 'self'; base-uri 'self'")
		c.Header("Permissions-Policy", "geolocation=(), microphone=(), camera=(), payment=(), usb=()")
		c.Header("Cross-Origin-Resource-Policy", "same-origin")
		c.Header("Cross-Origin-Opener-Policy", "same-origin")

		c.Next()
	}
}

func SecurityHeadersMiddleware() gin.HandlerFunc {
	return AddSecurityHeaders()
}
