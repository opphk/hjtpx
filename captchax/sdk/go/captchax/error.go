package captchax

import "fmt"

type Error struct {
	Code       int
	StatusCode int
	Message    string
	Details    interface{}
}

func (e *Error) Error() string {
	return fmt.Sprintf("CaptchaXError(%d): %s", e.Code, e.Message)
}

func NewError(message string) *Error {
	return &Error{
		Code:       500,
		StatusCode: 500,
		Message:    message,
	}
}

func NewErrorWithCode(message string, code int, statusCode int) *Error {
	return &Error{
		Code:       code,
		StatusCode: statusCode,
		Message:    message,
	}
}

func NewErrorWithDetails(message string, code int, statusCode int, details interface{}) *Error {
	return &Error{
		Code:       code,
		StatusCode: statusCode,
		Message:    message,
		Details:    details,
	}
}
