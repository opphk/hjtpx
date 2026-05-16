package middleware

import (
	"context"
	"encoding/json"
	"fmt"
	"net"
	"net/mail"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"sync"
	"time"
	"unicode"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

type ValidationRule interface {
	Validate(value interface{}) error
	Name() string
}

type ValidationResult struct {
	Valid   bool     `json:"valid"`
	Field   string   `json:"field"`
	Rule    string   `json:"rule"`
	Message string   `json:"message"`
	Errors  []string `json:"errors,omitempty"`
}

type Validator struct {
	rules       map[string][]ValidationRule
	customFuncs map[string]func(interface{}) error
	mu          sync.RWMutex
}

func NewValidator() *Validator {
	v := &Validator{
		rules:       make(map[string][]ValidationRule),
		customFuncs: make(map[string]func(interface{}) error),
	}
	v.registerBuiltInRules()
	return v
}

func (v *Validator) registerBuiltInRules() {
	v.RegisterRule("required", &RequiredRule{})
	v.RegisterRule("email", &EmailRule{})
	v.RegisterRule("url", &URLRule{})
	v.RegisterRule("ip", &IPRule{})
	v.RegisterRule("alpha", &AlphaRule{})
	v.RegisterRule("alphanum", &AlphanumRule{})
	v.RegisterRule("numeric", &NumericRule{})
	v.RegisterRule("integer", &IntegerRule{})
	v.RegisterRule("float", &FloatRule{})
	v.RegisterRule("min", &MinRule{})
	v.RegisterRule("max", &MaxRule{})
	v.RegisterRule("minlen", &MinLengthRule{})
	v.RegisterRule("maxlen", &MaxLengthRule{})
	v.RegisterRule("regex", &RegexRule{})
	v.RegisterRule("uuid", &UUIDRule{})
	v.RegisterRule("phone", &PhoneRule{})
	v.RegisterRule("idcard", &IDCardRule{})
	v.RegisterRule("json", &JSONRule{})
	v.RegisterRule("base64", &Base64Rule{})
	v.RegisterRule("hex", &HexRule{})
	v.RegisterRule("contains", &ContainsRule{})
	v.RegisterRule("in", &InRule{})
	v.RegisterRule("notin", &NotInRule{})
	v.RegisterRule("startswith", &StartsWithRule{})
	v.RegisterRule("endswith", &EndsWithRule{})
	v.RegisterRule("date", &DateRule{})
	v.RegisterRule("datetime", &DateTimeRule{})
	v.RegisterRule("timestamp", &TimestampRule{})
	v.RegisterRule("creditcard", &CreditCardRule{})
	v.RegisterRule("iban", &IBANRule{})
	v.RegisterRule("mac", &MACAddressRule{})
	v.RegisterRule("cidr", &CIDRRule{})
	v.RegisterRule("fqdn", &FQDNRule{})
	v.RegisterRule("ascii", &ASCIIRule{})
	v.RegisterRule("printascii", &PrintASCIIRule{})
	v.RegisterRule("utf8", &UTF8Rule{})
	v.RegisterRule("html", &HTMLRule{})
	v.RegisterRule("noscript", &NoScriptTagRule{})
	v.RegisterRule("sql", &SQLInjectionRule{})
	v.RegisterRule("xss", &XSSRule{})
	v.RegisterRule("path", &PathTraversalRule{})
}

func (v *Validator) RegisterRule(name string, rule ValidationRule) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.rules[name] = append(v.rules[name], rule)
}

func (v *Validator) RegisterCustomFunc(name string, fn func(interface{}) error) {
	v.mu.Lock()
	defer v.mu.Unlock()
	v.customFuncs[name] = fn
}

func (v *Validator) ValidateField(field string, value interface{}, ruleNames []string) *ValidationResult {
	result := &ValidationResult{
		Field:  field,
		Errors: []string{},
	}

	v.mu.RLock()
	defer v.mu.RUnlock()

	result.Valid = true
	for _, ruleName := range ruleNames {
		if rules, exists := v.rules[ruleName]; exists {
			for _, rule := range rules {
				if err := rule.Validate(value); err != nil {
					result.Valid = false
					result.Rule = ruleName
					result.Message = err.Error()
					result.Errors = append(result.Errors, err.Error())
				}
			}
		} else if fn, exists := v.customFuncs[ruleName]; exists {
			if err := fn(value); err != nil {
				result.Valid = false
				result.Rule = ruleName
				result.Message = err.Error()
				result.Errors = append(result.Errors, err.Error())
			}
		}
	}

	return result
}

type RequiredRule struct{}

func (r *RequiredRule) Validate(value interface{}) error {
	if value == nil {
		return fmt.Errorf("field is required")
	}

	switch v := value.(type) {
	case string:
		if strings.TrimSpace(v) == "" {
			return fmt.Errorf("field is required and cannot be empty")
		}
	case []interface{}:
		if len(v) == 0 {
			return fmt.Errorf("field is required and cannot be empty")
		}
	case map[string]interface{}:
		if len(v) == 0 {
			return fmt.Errorf("field is required and cannot be empty")
		}
	}

	return nil
}

func (r *RequiredRule) Name() string { return "required" }

type EmailRule struct{}

func (r *EmailRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	_, err := mail.ParseAddress(v)
	if err != nil {
		return fmt.Errorf("invalid email format")
	}
	return nil
}

func (r *EmailRule) Name() string { return "email" }

type URLRule struct{}

func (r *URLRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	parsed, err := url.Parse(v)
	if err != nil {
		return fmt.Errorf("invalid URL format")
	}
	if parsed.Scheme == "" || parsed.Host == "" {
		return fmt.Errorf("URL must include scheme and host")
	}
	if parsed.Scheme != "http" && parsed.Scheme != "https" {
		return fmt.Errorf("URL scheme must be http or https")
	}

	return nil
}

func (r *URLRule) Name() string { return "url" }

type IPRule struct{}

var ipRegex = regexp.MustCompile(`^(\d{1,3}\.){3}\d{1,3}$`)

func (r *IPRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if net.ParseIP(v) == nil {
		return fmt.Errorf("invalid IP address format")
	}
	return nil
}

func (r *IPRule) Name() string { return "ip" }

type AlphaRule struct{}

func (r *AlphaRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	for _, r := range v {
		if !unicode.IsLetter(r) {
			return fmt.Errorf("value must contain only letters")
		}
	}
	return nil
}

func (r *AlphaRule) Name() string { return "alpha" }

type AlphanumRule struct{}

func (r *AlphanumRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	for _, r := range v {
		if !unicode.IsLetter(r) && !unicode.IsDigit(r) {
			return fmt.Errorf("value must contain only letters and numbers")
		}
	}
	return nil
}

func (r *AlphanumRule) Name() string { return "alphanum" }

type NumericRule struct{}

func (r *NumericRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		if _, ok := value.(float64); ok {
			return nil
		}
		if _, ok := value.(int); ok {
			return nil
		}
		return fmt.Errorf("value must be a string or number")
	}
	if v == "" {
		return nil
	}

	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return fmt.Errorf("value must be a valid number")
	}
	return nil
}

func (r *NumericRule) Name() string { return "numeric" }

type IntegerRule struct{}

func (r *IntegerRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		if _, ok := value.(int); ok {
			return nil
		}
		if _, ok := value.(int64); ok {
			return nil
		}
		return fmt.Errorf("value must be a string or integer")
	}
	if v == "" {
		return nil
	}

	if _, err := strconv.ParseInt(v, 10, 64); err != nil {
		return fmt.Errorf("value must be a valid integer")
	}
	return nil
}

func (r *IntegerRule) Name() string { return "integer" }

type FloatRule struct{}

func (r *FloatRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		if _, ok := value.(float64); ok {
			return nil
		}
		return fmt.Errorf("value must be a string or float")
	}
	if v == "" {
		return nil
	}

	if _, err := strconv.ParseFloat(v, 64); err != nil {
		return fmt.Errorf("value must be a valid float")
	}
	return nil
}

func (r *FloatRule) Name() string { return "float" }

type MinRule struct {
	Value float64
}

func (r *MinRule) Validate(value interface{}) error {
	var num float64

	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case string:
		if v == "" {
			return nil
		}
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("value must be a number for min validation")
		}
		num = parsed
	default:
		return fmt.Errorf("value must be a number for min validation")
	}

	if num < r.Value {
		return fmt.Errorf("value must be at least %v", r.Value)
	}
	return nil
}

func (r *MinRule) Name() string { return "min" }

type MaxRule struct {
	Value float64
}

func (r *MaxRule) Validate(value interface{}) error {
	var num float64

	switch v := value.(type) {
	case float64:
		num = v
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case string:
		if v == "" {
			return nil
		}
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return fmt.Errorf("value must be a number for max validation")
		}
		num = parsed
	default:
		return fmt.Errorf("value must be a number for max validation")
	}

	if num > r.Value {
		return fmt.Errorf("value must be at most %v", r.Value)
	}
	return nil
}

func (r *MaxRule) Name() string { return "max" }

type MinLengthRule struct {
	Value int
}

func (r *MinLengthRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if utf8.RuneCountInString(v) < r.Value {
		return fmt.Errorf("value must be at least %d characters", r.Value)
	}
	return nil
}

func (r *MinLengthRule) Name() string { return "minlen" }

type MaxLengthRule struct {
	Value int
}

func (r *MaxLengthRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if utf8.RuneCountInString(v) > r.Value {
		return fmt.Errorf("value must be at most %d characters", r.Value)
	}
	return nil
}

func (r *MaxLengthRule) Name() string { return "maxlen" }

type RegexRule struct {
	Pattern string
}

func (r *RegexRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	pattern, err := regexp.Compile(r.Pattern)
	if err != nil {
		return fmt.Errorf("invalid regex pattern: %w", err)
	}

	if !pattern.MatchString(v) {
		return fmt.Errorf("value does not match required pattern")
	}
	return nil
}

func (r *RegexRule) Name() string { return "regex" }

type UUIDRule struct{}

var uuidRegex = regexp.MustCompile(`^[0-9a-f]{8}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{4}-[0-9a-f]{12}$`)

func (r *UUIDRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if !uuidRegex.MatchString(strings.ToLower(v)) {
		return fmt.Errorf("invalid UUID format")
	}
	return nil
}

func (r *UUIDRule) Name() string { return "uuid" }

type PhoneRule struct{}

var phoneRegex = regexp.MustCompile(`^\+?[1-9]\d{1,14}$`)

func (r *PhoneRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	cleaned := strings.ReplaceAll(strings.ReplaceAll(v, " ", ""), "-", "")
	if !phoneRegex.MatchString(cleaned) {
		return fmt.Errorf("invalid phone number format")
	}
	return nil
}

func (r *PhoneRule) Name() string { return "phone" }

type IDCardRule struct{}

func (r *IDCardRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if len(v) != 18 {
		return fmt.Errorf("ID card number must be 18 characters")
	}

	if !regexp.MustCompile(`^\d{17}[\dXx]$`).MatchString(v) {
		return fmt.Errorf("invalid ID card number format")
	}

	return nil
}

func (r *IDCardRule) Name() string { return "idcard" }

type JSONRule struct{}

func (r *JSONRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	var js json.RawMessage
	if err := json.Unmarshal([]byte(v), &js); err != nil {
		return fmt.Errorf("invalid JSON format: %w", err)
	}
	return nil
}

func (r *JSONRule) Name() string { return "json" }

type Base64Rule struct{}

func (r *Base64Rule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if _, err := regexp.MatchString(`^[A-Za-z0-9+/]*={0,2}$`, v); err != nil {
		return fmt.Errorf("invalid base64 format")
	}
	return nil
}

func (r *Base64Rule) Name() string { return "base64" }

type HexRule struct{}

func (r *HexRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if _, err := regexp.MatchString(`^[0-9a-fA-F]+$`, v); err != nil {
		return fmt.Errorf("invalid hex format")
	}
	return nil
}

func (r *HexRule) Name() string { return "hex" }

type ContainsRule struct {
	Substring string
}

func (r *ContainsRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if !strings.Contains(v, r.Substring) {
		return fmt.Errorf("value must contain '%s'", r.Substring)
	}
	return nil
}

func (r *ContainsRule) Name() string { return "contains" }

type InRule struct {
	Values []string
}

func (r *InRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	for _, allowed := range r.Values {
		if v == allowed {
			return nil
		}
	}
	return fmt.Errorf("value must be one of: %v", r.Values)
}

func (r *InRule) Name() string { return "in" }

type NotInRule struct {
	Values []string
}

func (r *NotInRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	for _, disallowed := range r.Values {
		if v == disallowed {
			return fmt.Errorf("value must not be one of: %v", r.Values)
		}
	}
	return nil
}

func (r *NotInRule) Name() string { return "notin" }

type StartsWithRule struct {
	Prefix string
}

func (r *StartsWithRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if !strings.HasPrefix(v, r.Prefix) {
		return fmt.Errorf("value must start with '%s'", r.Prefix)
	}
	return nil
}

func (r *StartsWithRule) Name() string { return "startswith" }

type EndsWithRule struct {
	Suffix string
}

func (r *EndsWithRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if !strings.HasSuffix(v, r.Suffix) {
		return fmt.Errorf("value must end with '%s'", r.Suffix)
	}
	return nil
}

func (r *EndsWithRule) Name() string { return "endswith" }

type DateRule struct {
	Format string
}

func (r *DateRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	format := r.Format
	if format == "" {
		format = "2006-01-02"
	}

	if _, err := time.Parse(format, v); err != nil {
		return fmt.Errorf("invalid date format, expected: %s", format)
	}
	return nil
}

func (r *DateRule) Name() string { return "date" }

type DateTimeRule struct{}

func (r *DateTimeRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	formats := []string{
		time.RFC3339,
		"2006-01-02T15:04:05",
		"2006-01-02 15:04:05",
	}

	for _, format := range formats {
		if _, err := time.Parse(format, v); err == nil {
			return nil
		}
	}
	return fmt.Errorf("invalid datetime format")
}

func (r *DateTimeRule) Name() string { return "datetime" }

type TimestampRule struct{}

func (r *TimestampRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	_, err := strconv.ParseInt(v, 10, 64)
	if err != nil {
		return fmt.Errorf("invalid timestamp")
	}
	return nil
}

func (r *TimestampRule) Name() string { return "timestamp" }

type CreditCardRule struct{}

var ccRegex = regexp.MustCompile(`^\d{13,19}$`)

func (r *CreditCardRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	cleaned := strings.ReplaceAll(strings.ReplaceAll(v, " ", ""), "-", "")
	if !ccRegex.MatchString(cleaned) {
		return fmt.Errorf("invalid credit card number")
	}

	if !validateLuhn(cleaned) {
		return fmt.Errorf("invalid credit card number (failed checksum)")
	}
	return nil
}

func validateLuhn(number string) bool {
	var sum int
	alternate := false

	for i := len(number) - 1; i >= 0; i-- {
		n := int(number[i] - '0')
		if alternate {
			n *= 2
			if n > 9 {
				n -= 9
			}
		}
		sum += n
		alternate = !alternate
	}
	return sum%10 == 0
}

func (r *CreditCardRule) Name() string { return "creditcard" }

type IBANRule struct{}

func (r *IBANRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	cleaned := strings.ToUpper(strings.ReplaceAll(strings.ReplaceAll(v, " ", ""), "-", ""))
	if len(cleaned) < 15 || len(cleaned) > 34 {
		return fmt.Errorf("invalid IBAN length")
	}

	if !regexp.MustCompile(`^[A-Z]{2}\d{2}[A-Z0-9]+$`).MatchString(cleaned) {
		return fmt.Errorf("invalid IBAN format")
	}

	return nil
}

func (r *IBANRule) Name() string { return "iban" }

type MACAddressRule struct{}

func (r *MACAddressRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	_, err := net.ParseMAC(v)
	if err != nil {
		return fmt.Errorf("invalid MAC address format")
	}
	return nil
}

func (r *MACAddressRule) Name() string { return "mac" }

type CIDRRule struct{}

func (r *CIDRRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	_, _, err := net.ParseCIDR(v)
	if err != nil {
		return fmt.Errorf("invalid CIDR format")
	}
	return nil
}

func (r *CIDRRule) Name() string { return "cidr" }

type FQDNRule struct{}

func (r *FQDNRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if len(v) > 253 {
		return fmt.Errorf("FQDN too long")
	}

	labels := strings.Split(v, ".")
	if len(labels) < 2 {
		return fmt.Errorf("invalid FQDN format")
	}

	for _, label := range labels {
		if len(label) > 63 {
			return fmt.Errorf("FQDN label too long")
		}
		if !regexp.MustCompile(`^[a-zA-Z0-9]([a-zA-Z0-9-]*[a-zA-Z0-9])?$`).MatchString(label) {
			return fmt.Errorf("invalid FQDN label format")
		}
	}
	return nil
}

func (r *FQDNRule) Name() string { return "fqdn" }

type ASCIIRule struct{}

func (r *ASCIIRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	for _, c := range v {
		if c > 127 {
			return fmt.Errorf("value must contain only ASCII characters")
		}
	}
	return nil
}

func (r *ASCIIRule) Name() string { return "ascii" }

type PrintASCIIRule struct{}

func (r *PrintASCIIRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	for _, c := range v {
		if c < 32 || c > 126 {
			return fmt.Errorf("value must contain only printable ASCII characters")
		}
	}
	return nil
}

func (r *PrintASCIIRule) Name() string { return "printascii" }

type UTF8Rule struct{}

func (r *UTF8Rule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if !utf8.ValidString(v) {
		return fmt.Errorf("value must be valid UTF-8")
	}
	return nil
}

func (r *UTF8Rule) Name() string { return "utf8" }

type HTMLRule struct{}

func (r *HTMLRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if regexp.MustCompile(`<[^>]*>`).MatchString(v) {
		return fmt.Errorf("value contains HTML tags which are not allowed")
	}
	return nil
}

func (r *HTMLRule) Name() string { return "html" }

type NoScriptTagRule struct{}

var scriptTagRegex = regexp.MustCompile(`(?i)<script[^>]*>.*?</script>`)
var eventHandlerRegex = regexp.MustCompile(`(?i)\bon\w+\s*=`)

func (r *NoScriptTagRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	if scriptTagRegex.MatchString(v) {
		return fmt.Errorf("value contains script tags")
	}
	if eventHandlerRegex.MatchString(v) {
		return fmt.Errorf("value contains potentially dangerous event handlers")
	}
	return nil
}

func (r *NoScriptTagRule) Name() string { return "noscript" }

type SQLInjectionRule struct{}

var sqlKeywords = []string{
	"select", "insert", "update", "delete", "drop", "create", "alter",
	"exec", "execute", "union", "where", "from", "table", "database",
	"grant", "revoke", "shutdown", "script", "javascript",
}

var sqlSpecialChars = []string{"'", "\"", ";", "--", "/*", "*/", "@@", "@"}

func (r *SQLInjectionRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	lower := strings.ToLower(v)
	for _, keyword := range sqlKeywords {
		if strings.Contains(lower, keyword) {
			return fmt.Errorf("value contains potentially dangerous SQL keyword '%s'", keyword)
		}
	}

	for _, char := range sqlSpecialChars {
		if strings.Contains(v, char) {
			return fmt.Errorf("value contains potentially dangerous character '%s'", char)
		}
	}
	return nil
}

func (r *SQLInjectionRule) Name() string { return "sql" }

type XSSRule struct{}

var xssPatterns = []string{
	"<script", "javascript:", "onerror=", "onload=", "onclick=",
	"onmouseover=", "onfocus=", "onblur=", "onchange=", "onsubmit=",
	"onreset=", "onselect=", "onunload=", "onabort=",
}

func (r *XSSRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	lower := strings.ToLower(v)
	for _, pattern := range xssPatterns {
		if strings.Contains(lower, pattern) {
			return fmt.Errorf("value contains potentially dangerous XSS pattern")
		}
	}
	return nil
}

func (r *XSSRule) Name() string { return "xss" }

type PathTraversalRule struct{}

var pathTraversalPatterns = []string{
	"../", "..\\", "%2e%2e", "%252e", "....//", "....///",
}

func (r *PathTraversalRule) Validate(value interface{}) error {
	v, ok := value.(string)
	if !ok {
		return fmt.Errorf("value must be a string")
	}
	if v == "" {
		return nil
	}

	lower := strings.ToLower(v)
	for _, pattern := range pathTraversalPatterns {
		if strings.Contains(lower, pattern) {
			return fmt.Errorf("value contains potentially dangerous path traversal pattern")
		}
	}
	return nil
}

func (r *PathTraversalRule) Name() string { return "path" }

type ValidationSchema struct {
	Fields map[string][]string
}

type ValidationMiddleware struct {
	validator       *Validator
	schema          *ValidationSchema
	sanitize        bool
	logViolations   bool
	strictMode      bool
	rateLimit       int
	rateLimitWindow time.Duration
	blockedIPs      map[string]time.Time
	mu              sync.RWMutex
}

func NewValidationMiddleware(schema *ValidationSchema) *ValidationMiddleware {
	return &ValidationMiddleware{
		validator:       NewValidator(),
		schema:          schema,
		sanitize:        true,
		logViolations:   true,
		strictMode:      true,
		rateLimit:       100,
		rateLimitWindow: 1 * time.Minute,
		blockedIPs:      make(map[string]time.Time),
	}
}

func (m *ValidationMiddleware) Handler() gin.HandlerFunc {
	return func(c *gin.Context) {
		if m.rateLimit > 0 {
			ip := c.ClientIP()
			if m.isRateLimited(ip) {
				c.AbortWithStatusJSON(429, gin.H{
					"error":   "rate_limit_exceeded",
					"message": "Too many validation requests",
				})
				return
			}
		}

		if m.schema == nil || len(m.schema.Fields) == 0 {
			c.Next()
			return
		}

		var validationErrors []string

		for field, rules := range m.schema.Fields {
			value := m.getFieldValue(c, field)

			for _, ruleName := range rules {
				result := m.validator.ValidateField(field, value, []string{ruleName})
				if !result.Valid {
					validationErrors = append(validationErrors, fmt.Sprintf("%s: %s", field, result.Message))
					
					if m.logViolations {
						fmt.Printf("[Validation] Field: %s, Rule: %s, Error: %s, IP: %s, Path: %s\n",
							field, ruleName, result.Message, c.ClientIP(), c.Request.URL.Path)
					}
				}
			}
		}

		if len(validationErrors) > 0 {
			c.AbortWithStatusJSON(400, gin.H{
				"error":   "validation_failed",
				"message": "Request validation failed",
				"errors":  validationErrors,
			})
			return
		}

		c.Next()
	}
}

func (m *ValidationMiddleware) getFieldValue(c *gin.Context, field string) interface{} {
	if strings.HasPrefix(field, "body.") {
		fieldName := strings.TrimPrefix(field, "body.")
		if m, ok := c.Get(gin.ContextKey); ok {
			if ctx, ok := m.(context.Context); ok {
				body, _ := ctx.Value("requestBody")
				if bodyMap, ok := body.(map[string]interface{}); ok {
					return bodyMap[fieldName]
				}
			}
		}
		
		c.Request.ParseForm()
		return c.PostForm(fieldName)
	}

	if strings.HasPrefix(field, "query.") {
		fieldName := strings.TrimPrefix(field, "query.")
		return c.Query(fieldName)
	}

	if strings.HasPrefix(field, "header.") {
		fieldName := strings.TrimPrefix(field, "header.")
		return c.GetHeader(fieldName)
	}

	if strings.HasPrefix(field, "param.") {
		fieldName := strings.TrimPrefix(field, "param.")
		return c.Param(fieldName)
	}

	return c.Query(field)
}

func (m *ValidationMiddleware) isRateLimited(ip string) bool {
	m.mu.Lock()
	defer m.mu.Unlock()

	if blockedUntil, exists := m.blockedIPs[ip]; exists {
		if time.Now().Before(blockedUntil) {
			return true
		}
		delete(m.blockedIPs, ip)
	}

	m.blockedIPs[ip] = time.Now().Add(m.rateLimitWindow)
	return false
}

func (m *ValidationMiddleware) AddRule(field, ruleName string, rule ValidationRule) {
	m.validator.RegisterRule(field+":"+ruleName, rule)
}

func (m *ValidationMiddleware) SetSchema(schema *ValidationSchema) {
	m.schema = schema
}

func (m *ValidationMiddleware) EnableSanitization(enable bool) {
	m.sanitize = enable
}

func (m *ValidationMiddleware) EnableLogging(enable bool) {
	m.logViolations = enable
}

func (m *ValidationMiddleware) SetRateLimit(limit int, window time.Duration) {
	m.rateLimit = limit
	m.rateLimitWindow = window
}

type OWASPComplianceValidator struct {
	*Validator
	enableSQLInjectionCheck bool
	enableXSSCheck         bool
	enableCSRFCheck        bool
	enablePathTraversalCheck bool
	enableCommandInjection   bool
}

func NewOWASPComplianceValidator() *OWASPComplianceValidator {
	return &OWASPComplianceValidator{
		Validator: NewValidator(),
		enableSQLInjectionCheck:  true,
		enableXSSCheck:          true,
		enablePathTraversalCheck: true,
		enableCommandInjection:   true,
	}
}

func (v *OWASPComplianceValidator) ValidateAllParams(params map[string]interface{}) []*ValidationResult {
	var results []*ValidationResult

	for field, value := range params {
		result := v.Validator.ValidateField(field, value, []string{"required", "sql", "xss", "noscript", "path"})
		results = append(results, result)
	}

	return results
}

func SanitizeInput(input string) string {
	replacements := map[string]string{
		"<":  "&lt;",
		">":  "&gt;",
		"\"": "&quot;",
		"'":  "&#x27;",
		"/":  "&#x2F;",
		"`":  "&#96;",
	}

	result := input
	for old, new := range replacements {
		result = strings.ReplaceAll(result, old, new)
	}

	return result
}
