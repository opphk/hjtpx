package middleware

import (
	"fmt"
	"net"
	"net/url"
	"regexp"
	"strconv"
	"strings"
	"unicode/utf8"

	"github.com/gin-gonic/gin"
)

// ValidationRule 验证规则
type ValidationRule struct {
	Field    string      // 字段名
	Type     string      // 类型：string, number, email, url, phone, ip, custom
	Required bool        // 是否必填
	Min      interface{} // 最小值
	Max      interface{} // 最大值
	Pattern  string      // 正则表达式
	Message  string      // 错误消息
	Custom   func(interface{}) bool // 自定义验证函数
}

// ValidationError 验证错误
type ValidationError struct {
	Field   string
	Message string
}

// Error 实现error接口
func (e *ValidationError) Error() string {
	return fmt.Sprintf("%s: %s", e.Field, e.Message)
}

// Validator 验证器
type Validator struct {
	rules []ValidationRule
}

// NewValidator 创建验证器
func NewValidator(rules []ValidationRule) *Validator {
	return &Validator{
		rules: rules,
	}
}

// Validate 验证数据
func (v *Validator) Validate(data map[string]interface{}) error {
	var errors []*ValidationError

	for _, rule := range v.rules {
		value, exists := data[rule.Field]

		if rule.Required && (!exists || isEmptyValue(value)) {
			errors = append(errors, &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage("此字段为必填项"),
			})
			continue
		}

		if !exists || isEmptyValue(value) {
			continue
		}

		if err := v.validateField(rule, value); err != nil {
			errors = append(errors, err)
		}
	}

	if len(errors) > 0 {
		return errors[0]
	}

	return nil
}

// validateField 验证字段
func (v *Validator) validateField(rule ValidationRule, value interface{}) *ValidationError {
	switch rule.Type {
	case "string":
		return v.validateString(rule, value)
	case "number":
		return v.validateNumber(rule, value)
	case "integer":
		return v.validateInteger(rule, value)
	case "email":
		return v.validateEmail(rule, value)
	case "url":
		return v.validateURL(rule, value)
	case "phone":
		return v.validatePhone(rule, value)
	case "ip":
		return v.validateIP(rule, value)
	case "alpha":
		return v.validateAlpha(rule, value)
	case "alphanum":
		return v.validateAlphanum(rule, value)
	case "uuid":
		return v.validateUUID(rule, value)
	case "custom":
		return v.validateCustom(rule, value)
	default:
		return nil
	}
}

// validateString 验证字符串
func (v *Validator) validateString(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	length := utf8.RuneCountInString(str)

	if rule.Min != nil {
		min := int(rule.Min.(float64))
		if length < min {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage(fmt.Sprintf("长度不能少于%d个字符", min)),
			}
		}
	}

	if rule.Max != nil {
		max := int(rule.Max.(float64))
		if length > max {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage(fmt.Sprintf("长度不能超过%d个字符", max)),
			}
		}
	}

	if rule.Pattern != "" {
		pattern := regexp.MustCompile(rule.Pattern)
		if !pattern.MatchString(str) {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage("格式不正确"),
			}
		}
	}

	return nil
}

// validateNumber 验证数字
func (v *Validator) validateNumber(rule ValidationRule, value interface{}) *ValidationError {
	var num float64

	switch v := value.(type) {
	case float64:
		num = v
	case float32:
		num = float64(v)
	case int:
		num = float64(v)
	case int64:
		num = float64(v)
	case string:
		parsed, err := strconv.ParseFloat(v, 64)
		if err != nil {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage("必须是一个有效的数字"),
			}
		}
		num = parsed
	default:
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("类型错误"),
		}
	}

	if rule.Min != nil {
		min := rule.Min.(float64)
		if num < min {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage(fmt.Sprintf("值不能小于%v", min)),
			}
		}
	}

	if rule.Max != nil {
		max := rule.Max.(float64)
		if num > max {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage(fmt.Sprintf("值不能大于%v", max)),
			}
		}
	}

	return nil
}

// validateInteger 验证整数
func (v *Validator) validateInteger(rule ValidationRule, value interface{}) *ValidationError {
	var num int64

	switch val := value.(type) {
	case int:
		num = int64(val)
	case int32:
		num = int64(val)
	case int64:
		num = val
	case float64:
		if val != float64(int64(val)) {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage("必须是一个整数"),
			}
		}
		num = int64(val)
	case string:
		parsed, err := strconv.ParseInt(val, 10, 64)
		if err != nil {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage("必须是一个有效的整数"),
			}
		}
		num = parsed
	default:
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("类型错误"),
		}
	}

	if rule.Min != nil {
		min := int64(rule.Min.(float64))
		if num < min {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage(fmt.Sprintf("值不能小于%v", min)),
			}
		}
	}

	if rule.Max != nil {
		max := int64(rule.Max.(float64))
		if num > max {
			return &ValidationError{
				Field:   rule.Field,
				Message: rule.getMessage(fmt.Sprintf("值不能大于%v", max)),
			}
		}
	}

	return nil
}

// validateEmail 验证邮箱
func (v *Validator) validateEmail(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	pattern := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
	if !pattern.MatchString(str) {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("邮箱格式不正确"),
		}
	}

	return nil
}

// validateURL 验证URL
func (v *Validator) validateURL(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	parsed, err := url.Parse(str)
	if err != nil {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("URL格式不正确"),
		}
	}

	if parsed.Scheme == "" || parsed.Host == "" {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("URL必须包含协议和主机"),
		}
	}

	validSchemes := []string{"http", "https", "ftp", "mailto"}
	isValidScheme := false
	for _, scheme := range validSchemes {
		if parsed.Scheme == scheme {
			isValidScheme = true
			break
		}
	}

	if !isValidScheme {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("URL协议不被允许"),
		}
	}

	return nil
}

// validatePhone 验证手机号
func (v *Validator) validatePhone(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	pattern := regexp.MustCompile(`^1[3-9]\d{9}$`)
	if !pattern.MatchString(str) {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("手机号格式不正确"),
		}
	}

	return nil
}

// validateIP 验证IP地址
func (v *Validator) validateIP(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	ip := net.ParseIP(str)
	if ip == nil {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("IP地址格式不正确"),
		}
	}

	return nil
}

// validateAlpha 验证纯字母
func (v *Validator) validateAlpha(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	pattern := regexp.MustCompile(`^[a-zA-Z]+$`)
	if !pattern.MatchString(str) {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("只能包含字母"),
		}
	}

	return nil
}

// validateAlphanum 验证字母数字
func (v *Validator) validateAlphanum(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	pattern := regexp.MustCompile(`^[a-zA-Z0-9]+$`)
	if !pattern.MatchString(str) {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("只能包含字母和数字"),
		}
	}

	return nil
}

// validateUUID 验证UUID
func (v *Validator) validateUUID(rule ValidationRule, value interface{}) *ValidationError {
	str, ok := value.(string)
	if !ok {
		str = fmt.Sprintf("%v", value)
	}

	pattern := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
	if !pattern.MatchString(str) {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("UUID格式不正确"),
		}
	}

	return nil
}

// validateCustom 自定义验证
func (v *Validator) validateCustom(rule ValidationRule, value interface{}) *ValidationError {
	if rule.Custom == nil {
		return nil
	}

	if !rule.Custom(value) {
		return &ValidationError{
			Field:   rule.Field,
			Message: rule.getMessage("验证失败"),
		}
	}

	return nil
}

// getMessage 获取错误消息
func (r *ValidationRule) getMessage(defaultMsg string) string {
	if r.Message != "" {
		return r.Message
	}
	return defaultMsg
}

// isEmptyValue 检查是否为空值
func isEmptyValue(value interface{}) bool {
	if value == nil {
		return true
	}

	switch v := value.(type) {
	case string:
		return strings.TrimSpace(v) == ""
	case []interface{}:
		return len(v) == 0
	case map[string]interface{}:
		return len(v) == 0
	case int, int32, int64:
		return false
	case float64:
		return false
	case bool:
		return false
	default:
		str := fmt.Sprintf("%v", value)
		return strings.TrimSpace(str) == ""
	}
}

// ValidateRequest 返回验证中间件
func ValidateRequest(rules []ValidationRule) gin.HandlerFunc {
	validator := NewValidator(rules)

	return func(c *gin.Context) {
		var data map[string]interface{}

		contentType := c.GetHeader("Content-Type")
		if strings.Contains(contentType, "application/json") {
			if err := c.ShouldBindJSON(&data); err != nil {
				c.AbortWithStatusJSON(400, gin.H{
					"code":    400,
					"message": "请求体格式错误",
					"error":   err.Error(),
				})
				return
			}
		} else if strings.Contains(contentType, "application/x-www-form-urlencoded") {
			data = make(map[string]interface{})
			for key, values := range c.Request.PostForm {
				if len(values) > 0 {
					data[key] = values[0]
				}
			}
		} else {
			data = make(map[string]interface{})
			for key, values := range c.Request.URL.Query() {
				if len(values) > 0 {
					data[key] = values[0]
				}
			}
		}

		if err := validator.Validate(data); err != nil {
			if validationErr, ok := err.(*ValidationError); ok {
				c.AbortWithStatusJSON(400, gin.H{
					"code":    400,
					"message": "验证失败",
					"error":   validationErr.Error(),
					"field":   validationErr.Field,
				})
				return
			}
		}

		c.Next()
	}
}

// ValidateJSON 返回JSON验证中间件
func ValidateJSON(rules []ValidationRule) gin.HandlerFunc {
	validator := NewValidator(rules)

	return func(c *gin.Context) {
		var data map[string]interface{}

		if err := c.ShouldBindJSON(&data); err != nil {
			c.AbortWithStatusJSON(400, gin.H{
				"code":    400,
				"message": "请求体格式错误",
				"error":   err.Error(),
			})
			return
		}

		if err := validator.Validate(data); err != nil {
			if validationErr, ok := err.(*ValidationError); ok {
				c.AbortWithStatusJSON(400, gin.H{
					"code":    400,
					"message": "验证失败",
					"error":   validationErr.Error(),
					"field":   validationErr.Field,
				})
				return
			}
		}

		c.Set("validated_data", data)
		c.Next()
	}
}

// CommonValidationRules 常用验证规则
var CommonValidationRules = map[string][]ValidationRule{
	"login": {
		{Field: "username", Type: "string", Required: true, Min: 3, Max: 50, Message: "用户名长度为3-50个字符"},
		{Field: "password", Type: "string", Required: true, Min: 6, Max: 100, Message: "密码长度至少6个字符"},
	},
	"register": {
		{Field: "username", Type: "string", Required: true, Min: 3, Max: 50, Message: "用户名长度为3-50个字符"},
		{Field: "email", Type: "email", Required: true, Message: "请输入有效的邮箱地址"},
		{Field: "password", Type: "string", Required: true, Min: 6, Max: 100, Message: "密码长度至少6个字符"},
		{Field: "phone", Type: "phone", Required: false, Message: "请输入有效的手机号"},
	},
	"changePassword": {
		{Field: "old_password", Type: "string", Required: true, Min: 6, Message: "旧密码长度至少6个字符"},
		{Field: "new_password", Type: "string", Required: true, Min: 6, Max: 100, Message: "新密码长度至少6个字符"},
		{Field: "confirm_password", Type: "string", Required: true, Message: "请确认新密码"},
	},
	"createApplication": {
		{Field: "name", Type: "string", Required: true, Min: 1, Max: 100, Message: "应用名称不能为空"},
		{Field: "url", Type: "url", Required: true, Message: "请输入有效的应用URL"},
		{Field: "description", Type: "string", Required: false, Max: 500, Message: "描述不能超过500个字符"},
	},
}

// GetValidationRules 获取验证规则
func GetValidationRules(name string) []ValidationRule {
	if rules, ok := CommonValidationRules[name]; ok {
		return rules
	}
	return []ValidationRule{}
}
