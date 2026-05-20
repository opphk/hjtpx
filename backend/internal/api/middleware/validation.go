package middleware

import (
	"regexp"
	"strconv"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ValidationRule struct {
	Field    string
	Type     string
	Required bool
	Min      int
	Max      int
	Pattern  string
	Message  string
}

func ValidateRequestMiddleware(rules []ValidationRule) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.Request.Method == "GET" {
			c.Next()
			return
		}

		var errors []response.FieldError

		for _, rule := range rules {
			value := getFieldValue(c, rule.Field)

			if rule.Required && (value == "" || value == nil) {
				msg := rule.Message
				if msg == "" {
					msg = rule.Field + " is required"
				}
				errors = append(errors, response.FieldError{
					Field:   rule.Field,
					Message: msg,
				})
				continue
			}

			if value != nil && value != "" {
				switch rule.Type {
				case "string":
					if rule.Min > 0 && len(value.(string)) < rule.Min {
						errors = append(errors, response.FieldError{
							Field:   rule.Field,
							Message: rule.Field + " must be at least " + strconv.Itoa(rule.Min) + " characters",
						})
					}
					if rule.Max > 0 && len(value.(string)) > rule.Max {
						errors = append(errors, response.FieldError{
							Field:   rule.Field,
							Message: rule.Field + " must be at most " + strconv.Itoa(rule.Max) + " characters",
						})
					}
					if rule.Pattern != "" {
						matched, _ := regexp.MatchString(rule.Pattern, value.(string))
						if !matched {
							msg := rule.Message
							if msg == "" {
								msg = rule.Field + " has invalid format"
							}
							errors = append(errors, response.FieldError{
								Field:   rule.Field,
								Message: msg,
							})
						}
					}

				case "int":
					val, err := strconv.Atoi(value.(string))
					if err != nil {
						errors = append(errors, response.FieldError{
							Field:   rule.Field,
							Message: rule.Field + " must be an integer",
						})
					} else {
						if rule.Min > 0 && val < rule.Min {
							errors = append(errors, response.FieldError{
								Field:   rule.Field,
								Message: rule.Field + " must be at least " + strconv.Itoa(rule.Min),
							})
						}
						if rule.Max > 0 && val > rule.Max {
							errors = append(errors, response.FieldError{
								Field:   rule.Field,
								Message: rule.Field + " must be at most " + strconv.Itoa(rule.Max),
							})
						}
					}

				case "email":
					emailRegex := regexp.MustCompile(`^[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}$`)
					if !emailRegex.MatchString(value.(string)) {
						msg := rule.Message
						if msg == "" {
							msg = rule.Field + " must be a valid email address"
						}
						errors = append(errors, response.FieldError{
							Field:   rule.Field,
							Message: msg,
						})
					}

				case "url":
					urlRegex := regexp.MustCompile(`^https?://[^\s/$.?#].[^\s]*$`)
					if !urlRegex.MatchString(value.(string)) {
						msg := rule.Message
						if msg == "" {
							msg = rule.Field + " must be a valid URL"
						}
						errors = append(errors, response.FieldError{
							Field:   rule.Field,
							Message: msg,
						})
					}

				case "uuid":
					uuidRegex := regexp.MustCompile(`^[0-9a-fA-F]{8}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{4}-[0-9a-fA-F]{12}$`)
					if !uuidRegex.MatchString(value.(string)) {
						msg := rule.Message
						if msg == "" {
							msg = rule.Field + " must be a valid UUID"
						}
						errors = append(errors, response.FieldError{
							Field:   rule.Field,
							Message: msg,
						})
					}
				}
			}
		}

		if len(errors) > 0 {
			response.BadRequestWithFields(c, "validation failed", errors)
			c.Abort()
			return
		}

		c.Next()
	}
}

func getFieldValue(c *gin.Context, field string) interface{} {
	switch c.Request.Method {
	case "GET":
		return c.Query(field)
	default:
		if c.ContentType() == "application/json" {
			var jsonData map[string]interface{}
			if err := c.ShouldBindJSON(&jsonData); err == nil {
				if val, exists := jsonData[field]; exists {
					return val
				}
			}
		}

		if val := c.PostForm(field); val != "" {
			return val
		}

		return c.Query(field)
	}
}

func ValidateJSONSchemaMiddleware(requiredFields []string) gin.HandlerFunc {
	return func(c *gin.Context) {
		if c.ContentType() != "application/json" {
			response.BadRequest(c, "Content-Type must be application/json")
			c.Abort()
			return
		}

		var jsonData map[string]interface{}
		if err := c.ShouldBindJSON(&jsonData); err != nil {
			response.BadRequest(c, "Invalid JSON body: "+err.Error())
			c.Abort()
			return
		}

		var errors []response.FieldError
		for _, field := range requiredFields {
			if _, exists := jsonData[field]; !exists {
				errors = append(errors, response.FieldError{
					Field:   field,
					Message: field + " is required",
				})
			}
		}

		if len(errors) > 0 {
			response.BadRequestWithFields(c, "validation failed", errors)
			c.Abort()
			return
		}

		c.Next()
	}
}

func SanitizeInputMiddleware() gin.HandlerFunc {
	return func(c *gin.Context) {
		sanitizeQueryParams(c)
		sanitizeFormParams(c)
		c.Next()
	}
}

func sanitizeQueryParams(c *gin.Context) {
	for key, values := range c.Request.URL.Query() {
		for i, val := range values {
			values[i] = sanitizeString(val)
		}
		c.Request.URL.Query()[key] = values
	}
}

func sanitizeFormParams(c *gin.Context) {
	c.Request.Form = make(map[string][]string)
	c.Request.PostForm = make(map[string][]string)

	for key, values := range c.Request.PostForm {
		for i, val := range values {
			values[i] = sanitizeString(val)
		}
		c.Request.PostForm[key] = values
		c.Request.Form[key] = values
	}
}

func sanitizeString(s string) string {
	s = strings.TrimSpace(s)

	dangerousPatterns := []string{
		"<script",
		"javascript:",
		"onerror=",
		"onclick=",
		"onload=",
	}

	for _, pattern := range dangerousPatterns {
		s = strings.ReplaceAll(s, pattern, "")
	}

	return s
}
