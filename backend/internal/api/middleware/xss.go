package middleware

import (
	"encoding/json"
	"fmt"
	"html"
	"html/template"
	"net/url"
	"regexp"
	"strings"

	"github.com/gin-gonic/gin"
)

// XSSConfig XSS防护配置
type XSSConfig struct {
	EnableInputFilter  bool     // 启用输入过滤
	EnableOutputFilter bool     // 启用输出过滤
	AllowedTags        []string // 允许的HTML标签
	AllowedAttrs       []string // 允许的HTML属性
	AllowedProtocols   []string // 允许的协议
	MaxInputLength     int      // 最大输入长度
}

// DefaultXSSConfig 默认XSS配置
var DefaultXSSConfig = &XSSConfig{
	EnableInputFilter:  true,
	EnableOutputFilter: true,
	AllowedTags:        []string{"p", "br", "b", "i", "u", "em", "strong", "a", "ul", "ol", "li", "h1", "h2", "h3", "h4", "h5", "h6", "blockquote", "code", "pre"},
	AllowedAttrs:       []string{"href", "title", "alt", "class"},
	AllowedProtocols:   []string{"http", "https", "mailto"},
	MaxInputLength:     10000,
}

// xssPatternStore XSS模式存储
type xssPatternStore struct {
	tagPattern           *regexp.Regexp
	attributePattern     *regexp.Regexp
	eventHandlerPattern  *regexp.Regexp
	javascriptPattern    *regexp.Regexp
	expressionPattern    *regexp.Regexp
	urlPattern          *regexp.Regexp
	cssPattern          *regexp.Regexp
	nullBytePattern      *regexp.Regexp
	encodedTagPattern    *regexp.Regexp
}

// globalXSSPatterns 全局XSS模式
var globalXSSPatterns = &xssPatternStore{
	tagPattern:          regexp.MustCompile(`(?i)<\s*script[^>]*>`),
	attributePattern:    regexp.MustCompile(`(?i)\s(on\w+)\s*=`),
	eventHandlerPattern: regexp.MustCompile(`(?i)<[^>]+(\s(onload|onerror|onclick|onmouseover|onfocus|onblur|onchange|onsubmit|onkeydown|onkeyup|onkeypress|ondblclick|oncontextmenu|onabort|onbeforeunload|onhashchange|onmessage|onoffline|ononline|onpagehide|onpageshow|onpopstate|onresize|onstorage|onunload)\s*=)`),
	javascriptPattern:    regexp.MustCompile(`(?i)javascript\s*:`),
	expressionPattern:   regexp.MustCompile(`(?i)expression\s*\(`),
	urlPattern:          regexp.MustCompile(`(?i)\b(javascript|vbscript)\s*:`),
	cssPattern:          regexp.MustCompile(`(?i)(expression|url|import|@import)\s*\(`),
	nullBytePattern:     regexp.MustCompile(`(?i)%00`),
	encodedTagPattern:   regexp.MustCompile(`(?i)(&lt;|<)(script|img|svg|iframe|object|embed|applet|form|body|frameset|frame|layer|ilayer|meta|base|plaintext|param|link|style|marquee|blink|xss)[\s>]+`),
}

// XSSProtector XSS防护器
type XSSProtector struct {
	config      *XSSConfig
	allowedTags map[string]bool
	allowedAttr map[string]bool
}

// NewXSSProtector 创建XSS防护器
func NewXSSProtector(config *XSSConfig) *XSSProtector {
	if config == nil {
		config = DefaultXSSConfig
	}

	allowedTags := make(map[string]bool)
	for _, tag := range config.AllowedTags {
		allowedTags[strings.ToLower(tag)] = true
	}

	allowedAttr := make(map[string]bool)
	for _, attr := range config.AllowedAttrs {
		allowedAttr[strings.ToLower(attr)] = true
	}

	return &XSSProtector{
		config:      config,
		allowedTags: allowedTags,
		allowedAttr: allowedAttr,
	}
}

// EscapeHTML 转义HTML特殊字符
func (xp *XSSProtector) EscapeHTML(input string) string {
	return html.EscapeString(input)
}

// EscapeHTMLAttribute 转义HTML属性值
func (xp *XSSProtector) EscapeHTMLAttribute(input string) string {
	input = html.EscapeString(input)
	input = strings.ReplaceAll(input, "\"", "&quot;")
	input = strings.ReplaceAll(input, "'", "&#39;")
	input = strings.ReplaceAll(input, " ", "%20")
	return input
}

// EscapeJavaScript 转义JavaScript代码
func (xp *XSSProtector) EscapeJavaScript(input string) string {
	var result strings.Builder
	for _, r := range input {
		switch r {
		case '\\':
			result.WriteString("\\\\")
		case '"':
			result.WriteString("\\\"")
		case '\'':
			result.WriteString("\\'")
		case '\n':
			result.WriteString("\\n")
		case '\r':
			result.WriteString("\\r")
		case '\t':
			result.WriteString("\\t")
		case '<':
			result.WriteString("\\x3c")
		case '>':
			result.WriteString("\\x3e")
		default:
			result.WriteRune(r)
		}
	}
	return result.String()
}

// EscapeURL 转义URL
func (xp *XSSProtector) EscapeURL(input string) string {
	return template.URLQueryEscaper(input)
}

// EscapeCSS 转义CSS
func (xp *XSSProtector) EscapeCSS(input string) string {
	var result strings.Builder
	for _, r := range input {
		if r < 256 {
			result.WriteString(fmt.Sprintf("\\%x", r))
		} else {
			result.WriteRune(r)
		}
	}
	return result.String()
}

// StripScriptTags 移除脚本标签
func (xp *XSSProtector) StripScriptTags(input string) string {
	re := regexp.MustCompile(`(?i)<script[^>]*>[\s\S]*?</script>`)
	result := re.ReplaceAllString(input, "")

	re = regexp.MustCompile(`(?i)<script[^>]*>`)
	result = re.ReplaceAllString(result, "")

	return result
}

// StripEventHandlers 移除事件处理器
func (xp *XSSProtector) StripEventHandlers(input string) string {
	result := input

	eventHandlers := []string{
		"onload", "onerror", "onclick", "onmouseover", "onmouseout",
		"onfocus", "onblur", "onchange", "onsubmit", "onkeydown",
		"onkeyup", "onkeypress", "ondblclick", "oncontextmenu",
		"onabort", "onbeforeunload", "onhashchange", "onmessage",
		"onoffline", "ononline", "onpagehide", "onpageshow",
		"onpopstate", "onresize", "onstorage", "onunload",
	}

	for _, handler := range eventHandlers {
		pattern := fmt.Sprintf(`(?i)\s*%s\s*=\s*["'][^"']*["']`, handler)
		re := regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")

		pattern = fmt.Sprintf(`(?i)\s*%s\s*=\s*[^\s>]+\s*`, handler)
		re = regexp.MustCompile(pattern)
		result = re.ReplaceAllString(result, "")
	}

	return result
}

// StripJavascriptProtocol 移除JavaScript协议
func (xp *XSSProtector) StripJavascriptProtocol(input string) string {
	re := regexp.MustCompile(`(?i)javascript\s*:`)
	result := re.ReplaceAllString(input, "")

	re = regexp.MustCompile(`(?i)vbscript\s*:`)
	result = re.ReplaceAllString(result, "")

	re = regexp.MustCompile(`(?i)data\s*:`)
	result = re.ReplaceAllString(result, "")

	return result
}

// SanitizeHTML 清理HTML内容
func (xp *XSSProtector) SanitizeHTML(input string) string {
	if len(input) > xp.config.MaxInputLength {
		input = input[:xp.config.MaxInputLength]
	}

	result := xp.StripScriptTags(input)
	result = xp.StripEventHandlers(result)
	result = xp.StripJavascriptProtocol(result)

	result = globalXSSPatterns.nullBytePattern.ReplaceAllString(result, "")

	result = xp.processAllowedTags(result)

	return result
}

// processAllowedTags 处理允许的标签
func (xp *XSSProtector) processAllowedTags(input string) string {
	result := input

	tagPattern := regexp.MustCompile(`<(\w+)([^>]*)>`)
	matches := tagPattern.FindAllStringSubmatch(input, -1)

	for _, match := range matches {
		tagName := strings.ToLower(match[1])
		attrs := match[2]

		if !xp.allowedTags[tagName] {
			result = strings.Replace(result, match[0], xp.EscapeHTML(match[0]), 1)
			continue
		}

		if tagName == "a" {
			result = xp.processATag(result, match[0], attrs)
		}
	}

	return result
}

// processATag 处理链接标签
func (xp *XSSProtector) processATag(input, fullTag, attrs string) string {
	hrefPattern := regexp.MustCompile(`(?i)href\s*=\s*["']([^"']+)["']`)
	matches := hrefPattern.FindStringSubmatch(attrs)

	if len(matches) > 1 {
		href := matches[1]

		if !xp.isAllowedProtocol(href) {
			unsafeTag := globalXSSPatterns.urlPattern.FindString(fullTag)
			if unsafeTag != "" {
				return strings.Replace(input, fullTag, xp.EscapeHTML(fullTag), 1)
			}
		}
	}

	return input
}

// isAllowedProtocol 检查协议是否允许
func (xp *XSSProtector) isAllowedProtocol(href string) bool {
	parsedURL, err := url.Parse(href)
	if err != nil {
		return true
	}

	scheme := strings.ToLower(parsedURL.Scheme)
	for _, allowed := range xp.config.AllowedProtocols {
		if scheme == strings.ToLower(allowed) {
			return true
		}
	}

	return scheme == "" || scheme == "http" || scheme == "https"
}

// FilterRequestParams 过滤请求参数
func (xp *XSSProtector) FilterRequestParams(params map[string]interface{}) map[string]interface{} {
	filtered := make(map[string]interface{})

	for key, value := range params {
		switch v := value.(type) {
		case string:
			filtered[key] = xp.SanitizeHTML(v)
		case []string:
			sanitized := make([]string, len(v))
			for i, s := range v {
				sanitized[i] = xp.SanitizeHTML(s)
			}
			filtered[key] = sanitized
		case map[string]interface{}:
			filtered[key] = xp.FilterRequestParams(v)
		default:
			filtered[key] = v
		}
	}

	return filtered
}

// XSSProtection 返回XSS防护中间件
func XSSProtection() gin.HandlerFunc {
	protector := NewXSSProtector(DefaultXSSConfig)

	return func(c *gin.Context) {
		if DefaultXSSConfig.EnableInputFilter {
			protector.filterQueryParams(c)
			protector.filterFormParams(c)
			protector.filterJSONBody(c)
		}

		if DefaultXSSConfig.EnableOutputFilter {
			protector.wrapResponseWriter(c)
		}

		c.Next()
	}
}

// filterQueryParams 过滤查询参数
func (xp *XSSProtector) filterQueryParams(c *gin.Context) {
	query := c.Request.URL.Query()
	filtered := xp.FilterRequestParams(xp.queryToInterface(query))

	newQuery := url.Values{}
	for key, value := range filtered {
		switch v := value.(type) {
		case string:
			newQuery.Add(key, v)
		case []string:
			for _, s := range v {
				newQuery.Add(key, s)
			}
		}
	}

	c.Request.URL.RawQuery = newQuery.Encode()
}

// filterFormParams 过滤表单参数
func (xp *XSSProtector) filterFormParams(c *gin.Context) {
	if c.Request.Form != nil {
		filtered := xp.FilterRequestParams(xp.formToInterface(c.Request.Form))
		xp.writeBackForm(c, filtered)
	}
}

// filterJSONBody 过滤JSON请求体
func (xp *XSSProtector) filterJSONBody(c *gin.Context) {
	if strings.Contains(c.GetHeader("Content-Type"), "application/json") {
		var body map[string]interface{}
		if err := json.NewDecoder(c.Request.Body).Decode(&body); err == nil {
			filtered := xp.FilterRequestParams(body)
			c.Set("filtered_json_body", filtered)

			encoded, _ := json.Marshal(filtered)
			c.Request.Body = xp.newReadCloser(string(encoded))
		}
	}
}

// wrapResponseWriter 包装响应写入器
func (xp *XSSProtector) wrapResponseWriter(c *gin.Context) {
	writer := &xssResponseWriter{
		ResponseWriter: c.Writer,
		protector:      xp,
	}
	c.Writer = writer
}

// queryToInterface 将Query转换为interface
func (xp *XSSProtector) queryToInterface(query url.Values) map[string]interface{} {
	result := make(map[string]interface{})
	for key, values := range query {
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			result[key] = values
		}
	}
	return result
}

// formToInterface 将Form转换为interface
func (xp *XSSProtector) formToInterface(form map[string][]string) map[string]interface{} {
	result := make(map[string]interface{})
	for key, values := range form {
		if len(values) == 1 {
			result[key] = values[0]
		} else {
			result[key] = values
		}
	}
	return result
}

// writeBackForm 写回表单数据
func (xp *XSSProtector) writeBackForm(c *gin.Context, data map[string]interface{}) {
	for key, value := range data {
		switch v := value.(type) {
		case string:
			c.Request.Form.Set(key, v)
		case []string:
			c.Request.Form.Del(key)
			for _, s := range v {
				c.Request.Form.Add(key, s)
			}
		}
	}
}

// newReadCloser 创建新的ReadCloser
func (xp *XSSProtector) newReadCloser(content string) *stringReadCloser {
	return &stringReadCloser{data: content, index: 0}
}

// stringReadCloser 字符串ReadCloser
type stringReadCloser struct {
	data  string
	index int
}

// Read 读取数据
func (src *stringReadCloser) Read(b []byte) (n int, err error) {
	if src.index >= len(src.data) {
		return 0, fmt.Errorf("EOF")
	}
	n = copy(b, src.data[src.index:])
	src.index += n
	return n, nil
}

// Close 关闭
func (src *stringReadCloser) Close() error {
	return nil
}

// xssResponseWriter XSS响应写入器
type xssResponseWriter struct {
	gin.ResponseWriter
	protector *XSSProtector
}

// Write 写入响应
func (w *xssResponseWriter) Write(data []byte) (n int, err error) {
	contentType := w.Header().Get("Content-Type")

	if strings.Contains(contentType, "text/html") {
		escaped := w.protector.SanitizeHTML(string(data))
		return w.ResponseWriter.Write([]byte(escaped))
	}

	if strings.Contains(contentType, "application/json") {
		var jsonData interface{}
		if err := json.Unmarshal(data, &jsonData); err == nil {
			if jsonMap, ok := jsonData.(map[string]interface{}); ok {
				filtered := w.protector.FilterRequestParams(jsonMap)
				encoded, err := json.Marshal(filtered)
				if err == nil {
					return w.ResponseWriter.Write(encoded)
				}
			}
		}
	}

	return w.ResponseWriter.Write(data)
}

// jsonToInterface 将JSON转换为interface
func jsonToInterface(data interface{}) interface{} {
	switch v := data.(type) {
	case map[string]interface{}:
		result := make(map[string]interface{})
		for key, value := range v {
			result[key] = jsonToInterface(value)
		}
		return result
	case []interface{}:
		result := make([]interface{}, len(v))
		for i, item := range v {
			result[i] = jsonToInterface(item)
		}
		return result
	case string:
		return html.EscapeString(v)
	default:
		return v
	}
}

// SanitizeHTMLString 导出函数用于外部调用
func SanitizeHTMLString(input string) string {
	protector := NewXSSProtector(DefaultXSSConfig)
	return protector.SanitizeHTML(input)
}

// EscapeHTMLString 导出函数用于外部调用
func EscapeHTMLString(input string) string {
	return html.EscapeString(input)
}

// EscapeForJavaScript 导出函数用于JavaScript转义
func EscapeForJavaScript(input string) string {
	protector := NewXSSProtector(DefaultXSSConfig)
	return protector.EscapeJavaScript(input)
}

// EscapeForURL 导出函数用于URL转义
func EscapeForURL(input string) string {
	return url.QueryEscape(input)
}
