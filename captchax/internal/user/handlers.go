package user

import (
	"net/http"
	"strconv"

	"captchax/internal/model"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type Handlers struct {
	svc *Service
}

func NewHandlers(svc *Service) *Handlers {
	return &Handlers{svc: svc}
}

func (h *Handlers) Register(c *gin.Context) {
	var req model.UserRegisterRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.svc.Register(&req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *Handlers) Login(c *gin.Context) {
	var req model.UserLoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	tokenResp, err := h.svc.Login(&req)
	if err != nil {
		response.Unauthorized(c, err.Error())
		return
	}

	response.Success(c, tokenResp)
}

func (h *Handlers) GetMe(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "authentication required")
		return
	}

	user, err := h.svc.GetUser(userID.(uint))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *Handlers) ChangePassword(c *gin.Context) {
	userID, exists := c.Get("user_id")
	if !exists {
		response.Unauthorized(c, "authentication required")
		return
	}

	var req model.UserChangePasswordRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.ChangePassword(userID.(uint), &req); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.SuccessWithMessage(c, "password changed successfully", nil)
}

func (h *Handlers) ListUsers(c *gin.Context) {
	var req model.UserListRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	users, total, err := h.svc.ListUsers(req.Page, req.PageSize, req.Search, req.Role, req.Status)
	if err != nil {
		response.InternalError(c, err.Error())
		return
	}

	response.Success(c, &response.PageResult{
		List:     users,
		Total:    total,
		Page:     req.Page,
		PageSize: req.PageSize,
	})
}

func (h *Handlers) GetUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	user, err := h.svc.GetUser(uint(id))
	if err != nil {
		response.NotFound(c, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *Handlers) UpdateUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req model.UserUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	user, err := h.svc.UpdateUser(uint(id), &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, user)
}

func (h *Handlers) DeleteUser(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	if err := h.svc.DeleteUser(uint(id)); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.SuccessWithMessage(c, "user deleted successfully", nil)
}

func (h *Handlers) UpdateUserRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req model.UserUpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.UpdateRole(uint(id), req.Role); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.SuccessWithMessage(c, "user role updated successfully", nil)
}

func (h *Handlers) UpdateUserStatus(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	var req model.UserUpdateStatusRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	if err := h.svc.UpdateStatus(uint(id), req.Status); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.SuccessWithMessage(c, "user status updated successfully", nil)
}