package security

import (
	"regexp"
	"strings"
)

// sqlFilter SQL注入过滤器
type sqlFilter struct {
	dangerousKeywords []string
	patterns         []*regexp.Regexp
}

// NewSQLFilter 创建SQL注入过滤器
func NewSQLFilter() *sqlFilter {
	return &sqlFilter{
		dangerousKeywords: []string{
			"UNION",
			"SELECT",
			"INSERT",
			"UPDATE",
			"DELETE",
			"DROP",
			"CREATE",
			"ALTER",
			"TRUNCATE",
			"EXEC",
			"EXECUTE",
			"DECLARE",
			"CAST",
			"CONVERT",
			"XP_",
			"SP_",
			"--",
			";--",
			"/*",
			"*/",
			"@@",
			"CHAR",
			"NCHAR",
			"VARCHAR",
			"NVARCHAR",
			"ALTER",
			"BEGIN",
			"CAST",
			"CREATE",
			"CURSOR",
			"DECLARE",
			"DELETE",
			"DROP",
			"END",
			"EXEC",
			"EXECUTE",
			"FETCH",
			"INSERT",
			"KILL",
			"LOAD",
			"OPEN",
			"SELECT",
			"SYS",
			"SYSCOLUMNS",
			"SYSOBJECTS",
			"TABLE",
			"UPDATE",
			"WHERE",
			"GRANT",
			"REVOKE",
		},
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bUNION\s+(ALL\s+)?SELECT\b`),
			regexp.MustCompile(`(?i)\bINSERT\s+INTO\b`),
			regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`),
			regexp.MustCompile(`(?i)\bDROP\s+(TABLE|DATABASE)\b`),
			regexp.MustCompile(`(?i)\bUPDATE\s+\w+\s+SET\b`),
			regexp.MustCompile(`(?i)\bEXEC(UTE)?\s*\(?`),
			regexp.MustCompile(`(?i)\bDECLARE\s+@`),
			regexp.MustCompile(`(?i)'[^']*'='[^']*'`),
			regexp.MustCompile(`(?i)\bOR\s+1\s*=\s*1\b`),
			regexp.MustCompile(`(?i)\bAND\s+1\s*=\s*1\b`),
			regexp.MustCompile(`(?i)--\s*$`),
			regexp.MustCompile(`(?i)/\*.*\*/`),
			regexp.MustCompile(`(?i)\bXP_\w+`),
			regexp.MustCompile(`(?i)\bSP_\w+`),
			regexp.MustCompile(`(?i)CHAR\s*\(\s*\d+\s*\)`),
			regexp.MustCompile(`(?i)CONVERT\s*\(`),
		},
	}
}

// FilterSQL 过滤SQL注入危险字符
func FilterSQL(input string) string {
	if input == "" {
		return ""
	}

	filter := NewSQLFilter()
	return filter.Filter(input)
}

// Filter 过滤SQL注入
func (f *sqlFilter) Filter(input string) string {
	result := input

	result = f.escapeQuotes(result)
	result = f.removeComments(result)
	result = f.neutralizeKeywords(result)
	result = f.blockPatterns(result)

	return result
}

// escapeQuotes 转义引号
func (f *sqlFilter) escapeQuotes(input string) string {
	result := strings.ReplaceAll(input, "'", "''")
	result = strings.ReplaceAll(result, "\"", "\\\"")
	return result
}

// removeComments 移除SQL注释
func (f *sqlFilter) removeComments(input string) string {
	result := regexp.MustCompile(`--[^\r\n]*`).ReplaceAllString(input, "")
	result = regexp.MustCompile(`/\*[\s\S]*?\*/`).ReplaceAllString(result, "")
	return result
}

// neutralizeKeywords 中性化危险关键词
func (f *sqlFilter) neutralizeKeywords(input string) string {
	result := input

	for _, keyword := range f.dangerousKeywords {
		pattern := regexp.MustCompile(`(?i)\b` + keyword + `\b`)
		replacement := f.maskKeyword(keyword)
		result = pattern.ReplaceAllString(result, replacement)
	}

	return result
}

// maskKeyword 遮蔽关键词
func (f *sqlFilter) maskKeyword(keyword string) string {
	if len(keyword) <= 2 {
		return strings.Repeat("*", len(keyword))
	}
	return keyword[:2] + strings.Repeat("*", len(keyword)-2)
}

// blockPatterns 阻止危险模式
func (f *sqlFilter) blockPatterns(input string) string {
	result := input

	for _, pattern := range f.patterns {
		result = pattern.ReplaceAllString(result, " ")
	}

	return result
}

// ContainsSQLKeywords 检查是否包含SQL关键词
func (f *sqlFilter) ContainsSQLKeywords(input string) bool {
	upperInput := strings.ToUpper(input)

	for _, keyword := range f.dangerousKeywords {
		pattern := regexp.MustCompile(`\b` + keyword + `\b`)
		if pattern.MatchString(upperInput) {
			return true
		}
	}

	for _, pattern := range f.patterns {
		if pattern.MatchString(input) {
			return true
		}
	}

	return false
}

// ContainsSQLKeywords 导出函数
func ContainsSQLKeywords(input string) bool {
	filter := NewSQLFilter()
	return filter.ContainsSQLKeywords(input)
}

// ValidateSQLInput 验证SQL输入
func ValidateSQLInput(input string) (bool, string) {
	if input == "" {
		return true, ""
	}

	filter := NewSQLFilter()

	if filter.ContainsSQLKeywords(input) {
		return false, "输入包含潜在的SQL注入风险"
	}

	if len(input) > 10000 {
		return false, "输入长度超过限制"
	}

	return true, ""
}

// EscapeSQLString 转义SQL字符串
func EscapeSQLString(input string) string {
	if input == "" {
		return ""
	}

	result := strings.ReplaceAll(input, "'", "''")
	return result
}

// SanitizeIdentifier 清理SQL标识符（表名、列名等）
func SanitizeIdentifier(identifier string) string {
	if identifier == "" {
		return ""
	}

	result := strings.TrimSpace(identifier)

	if matched, _ := regexp.MatchString(`^[a-zA-Z_][a-zA-Z0-9_]*$`, result); !matched {
		return ""
	}

	result = strings.ToLower(result)

	unsafe := []string{"drop", "delete", "insert", "update", "select", "exec", "execute"}
	for _, kw := range unsafe {
		if result == kw {
			return ""
		}
	}

	return result
}

// SafeQueryBuilder 安全查询构建器
type SafeQueryBuilder struct {
	tableName   string
	conditions  []string
	orderBy     string
	limit       int
	offset      int
}

// NewSafeQueryBuilder 创建安全查询构建器
func NewSafeQueryBuilder(table string) *SafeQueryBuilder {
	return &SafeQueryBuilder{
		tableName: SanitizeIdentifier(table),
	}
}

// Where 添加WHERE条件
func (qb *SafeQueryBuilder) Where(condition string) *SafeQueryBuilder {
	if qb.tableName == "" {
		return qb
	}

	qb.conditions = append(qb.conditions, condition)
	return qb
}

// WhereEqual 添加等于条件
func (qb *SafeQueryBuilder) WhereEqual(column string, value interface{}) *SafeQueryBuilder {
	if qb.tableName == "" {
		return qb
	}

	sanitizedColumn := SanitizeIdentifier(column)
	if sanitizedColumn == "" {
		return qb
	}

	qb.conditions = append(qb.conditions, sanitizedColumn+" = ?")
	return qb
}

// OrderBy 添加排序
func (qb *SafeQueryBuilder) OrderBy(column string, direction string) *SafeQueryBuilder {
	if qb.tableName == "" {
		return qb
	}

	sanitizedColumn := SanitizeIdentifier(column)
	if sanitizedColumn == "" {
		return qb
	}

	direction = strings.ToUpper(direction)
	if direction != "ASC" && direction != "DESC" {
		direction = "ASC"
	}

	qb.orderBy = sanitizedColumn + " " + direction
	return qb
}

// Limit 添加限制
func (qb *SafeQueryBuilder) Limit(limit int) *SafeQueryBuilder {
	if limit > 0 && limit <= 1000 {
		qb.limit = limit
	}
	return qb
}

// Offset 添加偏移
func (qb *SafeQueryBuilder) Offset(offset int) *SafeQueryBuilder {
	if offset >= 0 {
		qb.offset = offset
	}
	return qb
}

// BuildSelect 构建SELECT查询
func (qb *SafeQueryBuilder) BuildSelect() (string, []interface{}) {
	if qb.tableName == "" {
		return "", nil
	}

	query := "SELECT * FROM " + qb.tableName

	args := []interface{}{}

	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}

	if qb.orderBy != "" {
		query += " ORDER BY " + qb.orderBy
	}

	if qb.limit > 0 {
		query += " LIMIT ?"
		args = append(args, qb.limit)
	}

	if qb.offset > 0 {
		query += " OFFSET ?"
		args = append(args, qb.offset)
	}

	return query, args
}

// BuildCount 构建COUNT查询
func (qb *SafeQueryBuilder) BuildCount() (string, []interface{}) {
	if qb.tableName == "" {
		return "", nil
	}

	query := "SELECT COUNT(*) FROM " + qb.tableName

	args := []interface{}{}

	if len(qb.conditions) > 0 {
		query += " WHERE " + strings.Join(qb.conditions, " AND ")
	}

	return query, args
}

// SQLInjectionDetector SQL注入检测器
type SQLInjectionDetector struct {
	patterns []*regexp.Regexp
	weight   map[*regexp.Regexp]int
}

// NewSQLInjectionDetector 创建SQL注入检测器
func NewSQLInjectionDetector() *SQLInjectionDetector {
	detector := &SQLInjectionDetector{
		patterns: []*regexp.Regexp{
			regexp.MustCompile(`(?i)\bUNION\s+(ALL\s+)?SELECT\b`),
			regexp.MustCompile(`(?i)\bSELECT\s+\*\s+FROM\b`),
			regexp.MustCompile(`(?i)\bINSERT\s+INTO\b`),
			regexp.MustCompile(`(?i)\bDELETE\s+FROM\b`),
			regexp.MustCompile(`(?i)\bDROP\s+(TABLE|DATABASE)\b`),
			regexp.MustCompile(`(?i)\bUPDATE\b.*\bSET\b`),
			regexp.MustCompile(`(?i)\bEXEC(UTE)?\b`),
			regexp.MustCompile(`(?i)\bDECLARE\b`),
			regexp.MustCompile(`(?i)'[^']*'\s*=\s*'[^']*'`),
			regexp.MustCompile(`(?i)\bOR\s+1\s*=\s*1\b`),
			regexp.MustCompile(`(?i)\bAND\s+1\s*=\s*1\b`),
			regexp.MustCompile(`--\s*$`),
			regexp.MustCompile(`/\*.*\*/`),
			regexp.MustCompile(`(?i)\bXP_\w+`),
			regexp.MustCompile(`(?i)\bBENCHMARK\s*\(.*,.*\)`),
			regexp.MustCompile(`(?i)\bSLEEP\s*\(`),
			regexp.MustCompile(`(?i)\bWAITFOR\s+DELAY\b`),
			regexp.MustCompile(`(?i)\bLOAD_FILE\s*\(`),
			regexp.MustCompile(`(?i)\bINTO\s+OUTFILE\b`),
			regexp.MustCompile(`(?i)\bINTO\s+DUMPFILE\b`),
		},
		weight: make(map[*regexp.Regexp]int),
	}

	detector.weight[detector.patterns[0]] = 10
	detector.weight[detector.patterns[4]] = 10
	detector.weight[detector.patterns[6]] = 8
	detector.weight[detector.patterns[7]] = 8
	detector.weight[detector.patterns[14]] = 10
	detector.weight[detector.patterns[15]] = 10
	detector.weight[detector.patterns[16]] = 10
	detector.weight[detector.patterns[17]] = 10
	detector.weight[detector.patterns[18]] = 10

	return detector
}

// Detect 检测SQL注入
func (d *SQLInjectionDetector) Detect(input string) (bool, int) {
	score := 0

	for _, pattern := range d.patterns {
		if pattern.MatchString(input) {
			score += d.weight[pattern]
		}
	}

	return score >= 5, score
}

// DetectSQLInjection 导出函数
func DetectSQLInjection(input string) (bool, int) {
	detector := NewSQLInjectionDetector()
	return detector.Detect(input)
}
