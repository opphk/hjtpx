package utils

import (
	"net/http"

	"github.com/gin-gonic/gin"
)

type Response struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Error   string      `json:"error,omitempty"`
}

type PageResponse struct {
	Code    int         `json:"code"`
	Message string      `json:"message"`
	Data    interface{} `json:"data,omitempty"`
	Meta    PageMeta    `json:"meta"`
}

type PageMeta struct {
	Page       int   `json:"page"`
	PageSize   int   `json:"page_size"`
	Total      int64 `json:"total"`
	TotalPages int   `json:"total_pages"`
}

const (
	CodeSuccess         = 200
	CodeCreated         = 201
	CodeBadRequest      = 400
	CodeUnauthorized    = 401
	CodeForbidden       = 403
	CodeNotFound        = 404
	CodeTooManyRequests = 429
	CodeInternalError   = 500
)

var (
	MsgSuccess         = "操作成功"
	MsgCreated         = "创建成功"
	MsgBadRequest      = "请求参数错误"
	MsgUnauthorized    = "未授权"
	MsgForbidden       = "禁止访问"
	MsgNotFound        = "资源不存在"
	MsgTooManyRequests = "请求过于频繁"
	MsgInternalError   = "服务器内部错误"
)

func Success(c *gin.Context, data interface{}) {
	c.JSON(http.StatusOK, Response{
		Code:    CodeSuccess,
		Message: MsgSuccess,
		Data:    data,
	})
}

func Created(c *gin.Context, data interface{}) {
	c.JSON(http.StatusCreated, Response{
		Code:    CodeCreated,
		Message: MsgCreated,
		Data:    data,
	})
}

func RespondError(c *gin.Context, httpStatus int, code int, message string) {
	c.JSON(httpStatus, Response{
		Code:    code,
		Message: message,
	})
}

func BadRequest(c *gin.Context, message string) {
	RespondError(c, http.StatusBadRequest, CodeBadRequest, message)
}

func Unauthorized(c *gin.Context, message string) {
	if message == "" {
		message = MsgUnauthorized
	}
	RespondError(c, http.StatusUnauthorized, CodeUnauthorized, message)
}

func Forbidden(c *gin.Context, message string) {
	if message == "" {
		message = MsgForbidden
	}
	RespondError(c, http.StatusForbidden, CodeForbidden, message)
}

func NotFound(c *gin.Context, message string) {
	if message == "" {
		message = MsgNotFound
	}
	RespondError(c, http.StatusNotFound, CodeNotFound, message)
}

func TooManyRequests(c *gin.Context, message string) {
	if message == "" {
		message = MsgTooManyRequests
	}
	RespondError(c, http.StatusTooManyRequests, CodeTooManyRequests, message)
}

func InternalError(c *gin.Context, message string) {
	if message == "" {
		message = MsgInternalError
	}
	RespondError(c, http.StatusInternalServerError, CodeInternalError, message)
}

func Paginate(c *gin.Context, data interface{}, page, pageSize int, total int64) {
	totalPages := int(total) / pageSize
	if int(total)%pageSize > 0 {
		totalPages++
	}

	c.JSON(http.StatusOK, PageResponse{
		Code:    CodeSuccess,
		Message: MsgSuccess,
		Data:    data,
		Meta: PageMeta{
			Page:       page,
			PageSize:   pageSize,
			Total:      total,
			TotalPages: totalPages,
		},
	})
}

func ValidateParams(c *gin.Context, obj interface{}) bool {
	if err := c.ShouldBindJSON(obj); err != nil {
		BadRequest(c, "参数验证失败: "+err.Error())
		return false
	}
	return true
}
