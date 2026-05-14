package handler

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code      int         `json:"code"`
	Message   string      `json:"message"`
	Data      interface{} `json:"data,omitempty"`
	RequestID string      `json:"request_id,omitempty"`
}

type ListResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Total   int64       `json:"total"`
	Page    int         `json:"page"`
	Size    int         `json:"size"`
}

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: "success",
		Data:    data,
	})
}

func SuccessWithMessage(c *gin.Context, message string, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    0,
		Message: message,
		Data:    data,
	})
}

func Error(c *gin.Context, code int, message string) {
	c.JSON(http.StatusBadRequest, Response{
		Code:    code,
		Message: message,
	})
}

func ErrorWithStatus(c *gin.Context, status int, code int, message string) {
	c.JSON(status, Response{
		Code:    code,
		Message: message,
	})
}

func Paginated(c *gin.Context, data interface{}, total int64, page, size int) {
	c.JSON(http.StatusOK, ListResponse{
		Code:    0,
		Message: "success",
		Data:    data,
		Total:   total,
		Page:    page,
		Size:    size,
	})
}
