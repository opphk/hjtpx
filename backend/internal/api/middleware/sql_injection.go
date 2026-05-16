package middleware

import (
	"context"
	"fmt"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

type SQLInjectionConfig struct {
	EnableLog           bool
	BlockMode          bool
	LogOnlySafeQueries bool
	MaxQueryLength     int
	ExcludePaths       []string
	ExcludeParams      []string
	CustomPatterns     []string
	SeverityThreshold  int
}

var defaultSQLInjectionConfig = SQLInjectionConfig{
	EnableLog:          true,
	BlockMode:         true,
	LogOnlySafeQueries: false,
	MaxQueryLength:     10000,
	ExcludePaths:       []string{"/health", "/api/health"},
	ExcludeParams:      []string{},
	CustomPatterns:     []string{},
	SeverityThreshold:  3,
}

var sqlInjectionPatterns = []*regexp.Regexp{
	regexp.MustCompile(`(?i)(union\s+select|union\s+all\s+select)`),
	regexp.MustCompile(`(?i)(select\s+.*\s+from)`),
	regexp.MustCompile(`(?i)(insert\s+into)`),
	regexp.MustCompile(`(?i)(update\s+.*\s+set)`),
	regexp.MustCompile(`(?i)(delete\s+from)`),
	regexp.MustCompile(`(?i)(drop\s+table|drop\s+database)`),
	regexp.MustCompile(`(?i)(alter\s+table|alter\s+database)`),
	regexp.MustCompile(`(?i)(create\s+table|create\s+database)`),
	regexp.MustCompile(`(?i)(exec\s*\(|execute\s*\(|eval\s*\()`),
	regexp.MustCompile(`(?i)(xp_cmdshell|sp_executesql|openrowset|opendatasource)`),
	regexp.MustCompile(`(?i)(information_schema|mysql\.|pg_catalog|pg_user)`),
	regexp.MustCompile(`(?i)(systables|syscolumns|sysusers)`),
	regexp.MustCompile(`(?i)(--|\#|\/\*|\*\/)`),
	regexp.MustCompile(`(?i)(or\s+1\s*=\s*1|and\s+1\s*=\s*1)`),
	regexp.MustCompile(`(?i)(or\s+'.*'\s*=\s*'|or\s+"\s*=\s*")`),
	regexp.MustCompile(`(?i)(sleep\s*\(|benchmark\s*\(|pg_sleep\s*\()`),
	regexp.MustCompile(`(?i)(waitfor\s+delay|delay\s+'\d+:\d+:\d+')`),
	regexp.MustCompile(`(?i)(load_file\s*\(|into\s+outfile|into\s+dumpfile)`),
	regexp.MustCompile(`(?i)(0x[0-9a-f]+)`),
	regexp.MustCompile(`(?i)(char\s*\(|concat\s*\(|cast\s*\(|convert\s*\()`),
	regexp.MustCompile(`(?im)(having\s+\S+\s*[=<>]+\s*\S+)`),
	regexp.MustCompile(`(?i)(group\s+by\s+.*having)`),
	regexp.MustCompile(`(?i)(order\s+by\s+\d+|order\s+by\s+\S+)`),
	regexp.MustCompile(`(?i)(limit\s+\d+\s*,\s*\d+)`),
}

var sqlInjectionCache = &SQLInjectionCache{
	cache: make(map[string]*SQLInjectionCheckResult),
	mu:    sync.RWMutex{},
}

type SQLInjectionCheckResult struct {
	Detected   bool
	Patterns   []string
	Severity   int
	Timestamp  time.Time
}

type SQLInjectionCache struct {
	cache map[string]*SQLInjectionCheckResult
	mu    sync.RWMutex
}

func (c *SQLInjectionCache) Get(key string) (*SQLInjectionCheckResult, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	result, exists := c.cache[key]
	return result, exists
}

func (c *SQLInjectionCache) Set(key string, result *SQLInjectionCheckResult) {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache[key] = result
}

func (c *SQLInjectionCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.cache = make(map[string]*SQLInjectionCheckResult)
}

func init() {
	go sqlInjectionCache.cleanup()
}

func (c *SQLInjectionCache) cleanup() {
	ticker := time.NewTicker(5 * time.Minute)
	defer ticker.Stop()
	for range ticker.C {
		c.mu.Lock()
		now := time.Now()
		for key, result := range c.cache {
			if now.Sub(result.Timestamp) > 10*time.Minute {
				delete(c.cache, key)
			}
		}
		c.mu.Unlock()
	}
}

func checkSQLInjection(input string, config *SQLInjectionConfig) *SQLInjectionCheckResult {
	result := &SQLInjectionCheckResult{
		Detected:  false,
		Patterns:  []string{},
		Severity:  0,
		Timestamp: time.Now(),
	}

	if len(input) > config.MaxQueryLength {
		input = input[:config.MaxQueryLength]
	}

	inputLower := strings.ToLower(input)

	for i, pattern := range sqlInjectionPatterns {
		if pattern.MatchString(inputLower) {
			result.Detected = true
			result.Patterns = append(result.Patterns, fmt.Sprintf("pattern_%d", i))
			result.Severity++

			if i < 10 {
				result.Severity += 2
			} else if i < 15 {
				result.Severity++
			}
		}
	}

	for _, pattern := range config.CustomPatterns {
		re := regexp.MustCompile(pattern)
		if re.MatchString(inputLower) {
			result.Detected = true
			result.Patterns = append(result.Patterns, "custom_pattern")
			result.Severity += 2
		}
	}

	return result
}

func extractParams(c *gin.Context, config *SQLInjectionConfig) map[string]string {
	params := make(map[string]string)

	for key, values := range c.Request.URL.Query() {
		excluded := false
		for _, excludedParam := range config.ExcludeParams {
			if key == excludedParam {
				excluded = true
				break
			}
		}
		if !excluded && len(values) > 0 {
			params[key] = strings.Join(values, " ")
		}
	}

	if c.Request.Body != nil {
		c.Request.ParseForm()
		for key, values := range c.Request.PostForm {
			excluded := false
			for _, excludedParam := range config.ExcludeParams {
				if key == excludedParam {
					excluded = true
					break
				}
			}
			if !excluded && len(values) > 0 {
				params[key] = strings.Join(values, " ")
			}
		}
	}

	return params
}

func SQLInjectionProtection(config ...SQLInjectionConfig) gin.HandlerFunc {
	cfg := defaultSQLInjectionConfig
	if len(config) > 0 {
		cfg = config[0]
	}

	return func(c *gin.Context) {
		path := c.Request.URL.Path

		isExcluded := false
		for _, excluded := range cfg.ExcludePaths {
			if path == excluded || strings.HasPrefix(path, excluded+"/") {
				isExcluded = true
				break
			}
		}
		if isExcluded {
			c.Next()
			return
		}

		params := extractParams(c, &cfg)

		for paramName, paramValue := range params {
			cacheKey := fmt.Sprintf("%s:%s:%s", c.ClientIP(), paramName, paramValue[:min(len(paramValue), 50)])

			var result *SQLInjectionCheckResult
			if cached, exists := sqlInjectionCache.Get(cacheKey); exists {
				result = cached
			} else {
				result = checkSQLInjection(paramValue, &cfg)
				sqlInjectionCache.Set(cacheKey, result)
			}

			if result.Detected && result.Severity >= cfg.SeverityThreshold {
				if cfg.EnableLog {
					securityLog := GetSecurityLog()
					securityLog.Log(SecurityEvent{
						EventType:   EventInjectionAttempt,
						Level:       LevelHigh,
						ClientIP:    c.ClientIP(),
						Path:        path,
						Method:      c.Request.Method,
						UserAgent:   c.GetHeader("User-Agent"),
						Description: fmt.Sprintf("SQL injection attempt detected in parameter: %s", paramName),
						Details:     fmt.Sprintf("Patterns: %v, Severity: %d", result.Patterns, result.Severity),
						IsBlocked:   cfg.BlockMode,
					})

					if redis.Client != nil {
						ctx := context.Background()
						redis.Client.Incr(ctx, "security:sql_injection_count")
					}
				}

				if cfg.BlockMode {
					c.JSON(403, gin.H{
						"code":    403,
						"message": "请求包含非法参数",
						"error":   "sql_injection_detected",
					})
					c.Abort()
					return
				}
			}
		}

		c.Next()
	}
}

func CheckSQLInjection(value string) (bool, []string, int) {
	result := checkSQLInjection(value, &defaultSQLInjectionConfig)
	return result.Detected, result.Patterns, result.Severity
}

func SQLSanitize(input string) string {
	sanitized := input

	dangerousPatterns := []string{
		";", "--", "/*", "*/", "xp_", "sp_", "exec", "execute", "eval",
		"union", "select", "insert", "update", "delete", "drop", "create", "alter",
	}

	for _, pattern := range dangerousPatterns {
		re := regexp.MustCompile("(?i)" + pattern)
		sanitized = re.ReplaceAllString(sanitized, "")
	}

	return sanitized
}

func SQLInjectionStats() map[string]interface{} {
	stats := map[string]interface{}{
		"cached_checks": len(sqlInjectionCache.cache),
	}

	if redis.Client != nil {
		ctx := context.Background()
		count, err := redis.Client.Get(ctx, "security:sql_injection_count").Int()
		if err == nil {
			stats["total_detected"] = count
		}
	}

	return stats
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

type SQLQueryValidator struct {
	allowedTables   map[string]bool
	allowedColumns  map[string]map[string]bool
	maxValueLength  int
}

func NewSQLQueryValidator() *SQLQueryValidator {
	return &SQLQueryValidator{
		allowedTables:   make(map[string]bool),
		allowedColumns:  make(map[string]map[string]bool),
		maxValueLength:  1000,
	}
}

func (v *SQLQueryValidator) AddAllowedTable(table string) {
	v.allowedTables[strings.ToLower(table)] = true
}

func (v *SQLQueryValidator) AddAllowedColumn(table, column string) {
	if v.allowedColumns[table] == nil {
		v.allowedColumns[table] = make(map[string]bool)
	}
	v.allowedColumns[table][strings.ToLower(column)] = true
}

func (v *SQLQueryValidator) ValidateQuery(table, column, value string) error {
	if len(v.allowedTables) > 0 && !v.allowedTables[strings.ToLower(table)] {
		return fmt.Errorf("table '%s' is not in the allowed list", table)
	}

	if len(v.allowedColumns) > 0 {
		if cols, ok := v.allowedColumns[strings.ToLower(table)]; ok {
			if !cols[strings.ToLower(column)] {
				return fmt.Errorf("column '%s' is not in the allowed list for table '%s'", column, table)
			}
		}
	}

	if len(value) > v.maxValueLength {
		return fmt.Errorf("value length exceeds maximum allowed length of %d", v.maxValueLength)
	}

	detected, _, _ := CheckSQLInjection(value)
	if detected {
		return fmt.Errorf("value contains potentially dangerous SQL patterns")
	}

	return nil
}

func SQLQueryValidatorMiddleware(tables []string) gin.HandlerFunc {
	validator := NewSQLQueryValidator()
	for _, table := range tables {
		validator.AddAllowedTable(table)
	}

	return func(c *gin.Context) {
		c.Set("sql_validator", validator)
		c.Next()
	}
}

func GetSQLQueryValidator(c *gin.Context) *SQLQueryValidator {
	if validator, exists := c.Get("sql_validator"); exists {
		if v, ok := validator.(*SQLQueryValidator); ok {
			return v
		}
	}
	return nil
}

type SQLInjectionLog struct {
	ID          string    `json:"id"`
	Timestamp   time.Time `json:"timestamp"`
	ClientIP    string    `json:"client_ip"`
	Path        string    `json:"path"`
	Method      string    `json:"method"`
	Parameter   string    `json:"parameter"`
	Value       string    `json:"value"`
	Severity    int       `json:"severity"`
	Patterns    []string  `json:"patterns"`
	Blocked     bool      `json:"blocked"`
	UserAgent   string    `json:"user_agent"`
}

func LogSQLInjectionAttempt(log *SQLInjectionLog) {
	if redis.Client != nil {
		ctx := context.Background()
		key := fmt.Sprintf("security:sql_log:%s", log.ID)
		
		data := fmt.Sprintf("%s|%s|%s|%s|%s|%d|%v",
			log.Timestamp.Format(time.RFC3339),
			log.ClientIP,
			log.Path,
			log.Method,
			log.Parameter,
			log.Severity,
			log.Patterns,
		)
		
		redis.Client.Set(ctx, key, data, 7*24*time.Hour)
		redis.Client.LPush(ctx, "security:sql_logs", log.ID)
		redis.Client.LTrim(ctx, "security:sql_logs", 0, 999)
	}
}

func GetRecentSQLInjectionLogs(limit int) []*SQLInjectionLog {
	logs := make([]*SQLInjectionLog, 0)

	if redis.Client != nil {
		ctx := context.Background()
		ids, _ := redis.Client.LRange(ctx, "security:sql_logs", 0, int64(limit-1)).Result()

		for _, id := range ids {
			key := fmt.Sprintf("security:sql_log:%s", id)
			data, err := redis.Client.Get(ctx, key).Result()
			if err != nil {
				continue
			}

			log := &SQLInjectionLog{ID: id}
			parts := strings.Split(data, "|")
			if len(parts) >= 6 {
				if t, err := time.Parse(time.RFC3339, parts[0]); err == nil {
					log.Timestamp = t
				}
				log.ClientIP = parts[1]
				log.Path = parts[2]
				log.Method = parts[3]
				log.Parameter = parts[4]
				if sev, err := strconv.Atoi(parts[5]); err == nil {
					log.Severity = sev
				}
			}
			logs = append(logs, log)
		}
	}

	return logs
}
