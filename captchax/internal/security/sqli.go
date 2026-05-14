package security

import (
	"errors"
	"regexp"
	"strconv"
	"strings"
	"unicode"
	"unicode/utf8"
)

var (
	ErrDangerousPattern    = errors.New("sqli: dangerous SQL pattern detected")
	ErrUnescapedQuote     = errors.New("sqli: unescaped quote detected")
	ErrCommentInjection   = errors.New("sqli: SQL comment injection detected")
	ErrUnionInjection     = errors.New("sqli: potential UNION-based SQL injection")
	ErrStackedQuery       = errors.New("sqli: potential stacked query injection")
	ErrConditionBypass    = errors.New("sqli: potential condition bypass detected")
	ErrEncodedCharacter   = errors.New("sqli: encoded character injection detected")
)

type SQLInjectionDetector struct {
	patterns       []*regexp.Regexp
	commentPattern *regexp.Regexp
	unionPattern   *regexp.Regexp
	dangerousFuncs []string
	enabled        bool
}

var defaultDetector *SQLInjectionDetector

func init() {
	defaultDetector = NewSQLInjectionDetector()
}

func NewSQLInjectionDetector() *SQLInjectionDetector {
	patterns := []*regexp.Regexp{
		regexp.MustCompile(`(?i)(union\s+select|union\s+all\s+select)`),
		regexp.MustCompile(`(?i)(select\s+.*\s+from)`),
		regexp.MustCompile(`(?i)(insert\s+into)`),
		regexp.MustCompile(`(?i)(update\s+.*\s+set)`),
		regexp.MustCompile(`(?i)(delete\s+from)`),
		regexp.MustCompile(`(?i)(drop\s+(table|database|index))`),
		regexp.MustCompile(`(?i)(alter\s+table)`),
		regexp.MustCompile(`(?i)(create\s+(table|database|index|procedure))`),
		regexp.MustCompile(`(?i)(exec(\s|\()+|execute(\s|\()+|eval\s*\()`),
		regexp.MustCompile(`(?i)(xp_cmdshell|sp_executesql|openrowset|opendatasource)`),
		regexp.MustCompile(`(?i)(sleep\s*\(\s*\d+\s*\))`),
		regexp.MustCompile(`(?i)(benchmark\s*\()`),
		regexp.MustCompile(`(?i)(waitfor\s+delay|waitfor\s+time)`),
		regexp.MustCompile(`(?i)(load_file\s*\()`),
		regexp.MustCompile(`(?i)(into\s+(out|dump)file)`),
		regexp.MustCompile(`(?i)(having\s+\d+\s*=\s*\d+)`),
		regexp.MustCompile(`(?i)(and\s+\d+\s*=\s*\d+)`),
		regexp.MustCompile(`(?i)(or\s+\d+\s*=\s*\d+)`),
		regexp.MustCompile(`--`),
		regexp.MustCompile(`(?i)(;\s*drop)`),
		regexp.MustCompile(`(?i)(;\s*delete)`),
		regexp.MustCompile(`(?i)(;\s*insert)`),
		regexp.MustCompile(`(?i)(;\s*update)`),
	}

	return &SQLInjectionDetector{
		patterns:       patterns,
		commentPattern: regexp.MustCompile(`(?i)(--|\/\*|\*\/|#)`),
		unionPattern:   regexp.MustCompile(`(?i)union\s+(all\s+)?select`),
		dangerousFuncs: []string{
			"exec", "execute", "eval", "system",
			"load_file", "into_outfile", "into_dumpfile",
			"xp_cmdshell", "sp_executesql", "openrowset",
		},
		enabled: true,
	}
}

func (d *SQLInjectionDetector) Enable() {
	d.enabled = true
}

func (d *SQLInjectionDetector) Disable() {
	d.enabled = false
}

func (d *SQLInjectionDetector) IsEnabled() bool {
	return d.enabled
}

func (d *SQLInjectionDetector) Detect(input string) (bool, error) {
	if !d.enabled {
		return false, nil
	}

	if len(input) == 0 {
		return false, nil
	}

	if err := d.checkEncodedChars(input); err != nil {
		return true, err
	}

	for _, pattern := range d.patterns {
		if pattern.MatchString(input) {
			return true, ErrDangerousPattern
		}
	}

	if d.commentPattern.MatchString(input) {
		if err := d.checkCommentInjection(input); err != nil {
			return true, err
		}
	}

	if d.unionPattern.MatchString(input) {
		return true, ErrUnionInjection
	}

	if err := d.checkStackedQueries(input); err != nil {
		return true, err
	}

	if err := d.checkQuoteEscaping(input); err != nil {
		return true, err
	}

	if err := d.checkConditionBypass(input); err != nil {
		return true, err
	}

	return false, nil
}

func (d *SQLInjectionDetector) DetectMultiple(inputs ...string) (bool, []error) {
	if !d.enabled {
		return false, nil
	}

	var errors []error
	for _, input := range inputs {
		detected, err := d.Detect(input)
		if detected {
			errors = append(errors, err)
			if len(errors) >= 10 {
				break
			}
		}
	}

	if len(errors) > 0 {
		return true, errors
	}
	return false, nil
}

func (d *SQLInjectionDetector) Sanitize(input string) (string, error) {
	if !d.enabled {
		return input, nil
	}

	sanitized := input

	hexPattern := regexp.MustCompile(`(?i)(0x[0-9a-f]+)`)
	sanitized = hexPattern.ReplaceAllStringFunc(sanitized, func(match string) string {
		return "HEX_ENCODED"
	})

	charPattern := regexp.MustCompile(`(?i)(char\s*\(\s*\d+\s*\))`)
	sanitized = charPattern.ReplaceAllStringFunc(sanitized, func(match string) string {
		return "CHAR_ENCODED"
	})

	sanitized = strings.ReplaceAll(sanitized, "--", " ")
	sanitized = strings.ReplaceAll(sanitized, "/*", " ")
	sanitized = strings.ReplaceAll(sanitized, "*/", " ")
	sanitized = strings.ReplaceAll(sanitized, "#", " ")

	detected, err := d.Detect(sanitized)
	if detected {
		return "", err
	}

	return sanitized, nil
}

func (d *SQLInjectionDetector) Validate(input string) error {
	detected, err := d.Detect(input)
	if detected {
		return err
	}
	return nil
}

func (d *SQLInjectionDetector) ValidateAll(inputs ...string) error {
	for _, input := range inputs {
		if err := d.Validate(input); err != nil {
			return err
		}
	}
	return nil
}

func (d *SQLInjectionDetector) checkEncodedChars(input string) error {
	hexEncoded := regexp.MustCompile(`(?i)0x[0-9a-f]+`)
	if hexEncoded.MatchString(input) {
		return ErrEncodedCharacter
	}

	charEncoded := regexp.MustCompile(`(?i)(char\s*\(\s*\d+\s*\)|chr\s*\(\s*\d+\s*\))`)
	if charEncoded.MatchString(input) {
		return ErrEncodedCharacter
	}

	doubleEncoded := regexp.MustCompile(`%25[0-9a-f]{2}`)
	if doubleEncoded.MatchString(input) {
		return ErrEncodedCharacter
	}

	return nil
}

func (d *SQLInjectionDetector) checkCommentInjection(input string) error {
	lines := strings.Split(input, "\n")
	for _, line := range lines {
		line = strings.TrimSpace(line)
		if strings.HasPrefix(line, "--") {
			return ErrCommentInjection
		}
		if strings.HasPrefix(line, "#") && !strings.HasPrefix(line, "#") {
			return ErrCommentInjection
		}
	}

	if strings.Contains(input, "/*") && !strings.Contains(input, "*/") {
		return ErrCommentInjection
	}
	if strings.Contains(input, "*/") && !strings.Contains(input, "/*") {
		return ErrCommentInjection
	}

	return nil
}

func (d *SQLInjectionDetector) checkStackedQueries(input string) error {
	trimmed := strings.TrimSpace(input)

	stmtStarters := []string{
		";", "AND", "OR", "UNION", "SELECT", "INSERT", "UPDATE", "DELETE",
		"DROP", "CREATE", "ALTER", "EXEC", "EXECUTE", "CALL",
	}

	for _, starter := range stmtStarters {
		pattern := regexp.MustCompile(`(?i)\s*;\s*` + starter + `\s`)
		if pattern.MatchString(trimmed) {
			return ErrStackedQuery
		}
	}

	return nil
}

func (d *SQLInjectionDetector) checkQuoteEscaping(input string) error {
	singleQuotes := 0
	for _, r := range input {
		if r == '\'' {
			singleQuotes++
		}
	}
	if singleQuotes%2 != 0 {
		return ErrUnescapedQuote
	}

	doubledQuotes := regexp.MustCompile(`(?i)''`)
	if doubledQuotes.MatchString(input) {
		unescaped := regexp.MustCompile(`(?i)'(?!\s*')`)
		matches := unescaped.FindAllString(input, -1)
		if len(matches)%2 != 0 {
			return ErrUnescapedQuote
		}
	}

	return nil
}

func (d *SQLInjectionDetector) checkConditionBypass(input string) error {
	orPattern := regexp.MustCompile(`(?i)\bor\b\s*\d+\s*=\s*\d+`)
	if orPattern.MatchString(input) {
		orTruePattern := regexp.MustCompile(`(?i)\bor\b\s*('[^']*'\s*=\s*'[^']*'|\d+\s*=\s*\d+|\btrue\b|\b1\b\s*=\s*\d+)`)
		if orTruePattern.MatchString(input) {
			return ErrConditionBypass
		}
	}

	andPattern := regexp.MustCompile(`(?i)\band\b\s*\d+\s*=\s*\d+`)
	if andPattern.MatchString(input) {
		andFalsePattern := regexp.MustCompile(`(?i)\band\b\s*('[^']*'\s*!=\s*'[^']*'|\bfalse\b|\b0\b\s*=\s*\d+)`)
		if andFalsePattern.MatchString(input) {
			return ErrConditionBypass
		}
	}

	return nil
}

func (d *SQLInjectionDetector) AddPattern(pattern *regexp.Regexp) {
	d.patterns = append(d.patterns, pattern)
}

func (d *SQLInjectionDetector) RemovePattern(pattern *regexp.Regexp) {
	newPatterns := make([]*regexp.Regexp, 0, len(d.patterns))
	for _, p := range d.patterns {
		if p != pattern {
			newPatterns = append(newPatterns, p)
		}
	}
	d.patterns = newPatterns
}

type QueryBuilder struct {
	table     string
	columns   []string
	conditions []string
	orderBy   string
	limit     int
	offset    int
	detector  *SQLInjectionDetector
}

func NewQueryBuilder(table string) *QueryBuilder {
	return &QueryBuilder{
		table:    table,
		detector: defaultDetector,
	}
}

func (qb *QueryBuilder) Select(columns ...string) *QueryBuilder {
	qb.columns = columns
	return qb
}

func (qb *QueryBuilder) Where(condition string, args ...interface{}) *QueryBuilder {
	if err := qb.detector.Validate(condition); err != nil {
		return qb
	}
	for _, arg := range args {
		if str, ok := arg.(string); ok {
			if err := qb.detector.Validate(str); err != nil {
				return qb
			}
		}
	}
	qb.conditions = append(qb.conditions, condition)
	return qb
}

func (qb *QueryBuilder) OrderBy(column string, direction string) *QueryBuilder {
	qb.orderBy = column + " " + strings.ToUpper(direction)
	return qb
}

func (qb *QueryBuilder) Limit(n int) *QueryBuilder {
	qb.limit = n
	return qb
}

func (qb *QueryBuilder) Offset(n int) *QueryBuilder {
	qb.offset = n
	return qb
}

func (qb *QueryBuilder) Build() (string, []interface{}) {
	var query strings.Builder

	query.WriteString("SELECT ")
	if len(qb.columns) == 0 {
		query.WriteString("*")
	} else {
		query.WriteString(strings.Join(qb.columns, ", "))
	}

	query.WriteString(" FROM ")
	query.WriteString(qb.table)

	if len(qb.conditions) > 0 {
		query.WriteString(" WHERE ")
		query.WriteString(strings.Join(qb.conditions, " AND "))
	}

	if qb.orderBy != "" {
		query.WriteString(" ORDER BY ")
		query.WriteString(qb.orderBy)
	}

	if qb.limit > 0 {
		query.WriteString(" LIMIT ")
		query.WriteString(strconv.Itoa(qb.limit))
	}

	if qb.offset > 0 {
		query.WriteString(" OFFSET ")
		query.WriteString(strconv.Itoa(qb.offset))
	}

	return query.String(), nil
}

func DetectSQLInjection(input string) (bool, error) {
	return defaultDetector.Detect(input)
}

func SanitizeSQLInput(input string) (string, error) {
	return defaultDetector.Sanitize(input)
}

func ValidateSQLInput(input string) error {
	return defaultDetector.Validate(input)
}

type SQLValidator struct {
	detector   *SQLInjectionDetector
	maxLength  int
	minLength  int
	allowEmpty bool
}

func NewSQLValidator(maxLength, minLength int, allowEmpty bool) *SQLValidator {
	return &SQLValidator{
		detector:   defaultDetector,
		maxLength:  maxLength,
		minLength:  minLength,
		allowEmpty: allowEmpty,
	}
}

func (v *SQLValidator) Validate(input string) error {
	if len(input) == 0 {
		if v.allowEmpty {
			return nil
		}
		return errors.New("sqli: input cannot be empty")
	}

	if v.maxLength > 0 && len(input) > v.maxLength {
		return errors.New("sqli: input exceeds maximum length")
	}

	if v.minLength > 0 && len(input) < v.minLength {
		return errors.New("sqli: input below minimum length")
	}

	if !utf8.ValidString(input) {
		return errors.New("sqli: invalid UTF-8 encoding")
	}

	for _, r := range input {
		if !unicode.IsPrint(r) && !unicode.IsSpace(r) {
			return errors.New("sqli: contains non-printable characters")
		}
	}

	return v.detector.Validate(input)
}

func (v *SQLValidator) ValidateMultiple(inputs ...string) error {
	for _, input := range inputs {
		if err := v.Validate(input); err != nil {
			return err
		}
	}
	return nil
}

func IsSQLKeyword(input string) bool {
	keywords := []string{
		"SELECT", "INSERT", "UPDATE", "DELETE", "DROP", "CREATE",
		"ALTER", "TRUNCATE", "EXEC", "EXECUTE", "UNION", "JOIN",
		"WHERE", "FROM", "TABLE", "DATABASE", "INDEX", "PROCEDURE",
		"GRANT", "REVOKE", "COMMIT", "ROLLBACK", "SAVEPOINT",
	}

	upper := strings.ToUpper(input)
	for _, keyword := range keywords {
		if upper == keyword {
			return true
		}
	}
	return false
}
