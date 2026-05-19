package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type TenantHandler struct {
	tenantService *service.TenantService
}

func NewTenantHandler(tenantService *service.TenantService) *TenantHandler {
	return &TenantHandler{
		tenantService: tenantService,
	}
}

func (h *TenantHandler) CreateTenant(c *gin.Context) {
	var req struct {
		Name         string `json:"name" binding:"required"`
		Code         string `json:"code" binding:"required"`
		Plan         string `json:"plan"`
		ContactEmail string `json:"contact_email"`
		ContactPhone string `json:"contact_phone"`
		Domain       string `json:"domain"`
		Description  string `json:"description"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	adminID, _ := c.Get("admin_id")

	tenant := &models.Tenant{
		Name:         req.Name,
		Code:         req.Code,
		Plan:         req.Plan,
		ContactEmail: req.ContactEmail,
		ContactPhone: req.ContactPhone,
		Domain:       req.Domain,
		Description:  req.Description,
		Status:       "active",
		CreatedBy:    adminID.(uint),
	}

	result, err := h.tenantService.CreateTenant(tenant, adminID.(uint))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to create tenant")
		return
	}

	response.Success(c, result)
}

func (h *TenantHandler) GetTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	tenant, err := h.tenantService.GetTenant(uint(id))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "tenant not found")
		return
	}

	response.Success(c, tenant)
}

func (h *TenantHandler) GetTenantByCode(c *gin.Context) {
	code := c.Param("code")

	tenant, err := h.tenantService.GetTenantByCode(code)
	if err != nil {
		response.Fail(c, http.StatusNotFound, "tenant not found")
		return
	}

	response.Success(c, tenant)
}

func (h *TenantHandler) ListTenants(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	status := c.Query("status")
	plan := c.Query("plan")
	search := c.Query("search")

	tenants, total, err := h.tenantService.ListTenants(page, pageSize, status, plan, search)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to list tenants")
		return
	}

	response.Success(c, gin.H{
		"items":      tenants,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *TenantHandler) UpdateTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var updates map[string]interface{}
	if err := c.ShouldBindJSON(&updates); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	adminID, _ := c.Get("admin_id")

	tenant, err := h.tenantService.UpdateTenant(uint(id), updates, adminID.(uint))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to update tenant")
		return
	}

	response.Success(c, tenant)
}

func (h *TenantHandler) DeleteTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	adminID, _ := c.Get("admin_id")

	if err := h.tenantService.DeleteTenant(uint(id), adminID.(uint)); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to delete tenant")
		return
	}

	response.Success(c, gin.H{"message": "tenant deleted successfully"})
}

func (h *TenantHandler) AddTenantUser(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	adminID, _ := c.Get("admin_id")
	if req.Role == "" {
		req.Role = "member"
	}

	user, err := h.tenantService.AddTenantUser(uint(tenantID), req.UserID, req.Role, adminID.(uint))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "failed to add user")
		return
	}

	response.Success(c, user)
}

func (h *TenantHandler) RemoveTenantUser(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	userID, err := strconv.ParseUint(c.Query("user_id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid user id")
		return
	}

	if err := h.tenantService.RemoveTenantUser(uint(tenantID), uint(userID)); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to remove user")
		return
	}

	response.Success(c, gin.H{"message": "user removed successfully"})
}

func (h *TenantHandler) UpdateTenantUserRole(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var req struct {
		UserID uint   `json:"user_id" binding:"required"`
		Role   string `json:"role" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	adminID, _ := c.Get("admin_id")

	if err := h.tenantService.UpdateTenantUserRole(uint(tenantID), req.UserID, req.Role, adminID.(uint)); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to update role")
		return
	}

	response.Success(c, gin.H{"message": "role updated successfully"})
}

func (h *TenantHandler) ListTenantUsers(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	users, total, err := h.tenantService.ListTenantUsers(uint(tenantID), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to list users")
		return
	}

	response.Success(c, gin.H{
		"items":       users,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *TenantHandler) CreateInvitation(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var req struct {
		Email string `json:"email" binding:"required,email"`
		Role  string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	adminID, _ := c.Get("admin_id")
	if req.Role == "" {
		req.Role = "member"
	}

	invitation, err := h.tenantService.CreateInvitation(uint(tenantID), req.Email, req.Role, adminID.(uint))
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "failed to create invitation")
		return
	}

	response.Success(c, invitation)
}

func (h *TenantHandler) AcceptInvitation(c *gin.Context) {
	var req struct {
		Token  string `json:"token" binding:"required"`
		UserID uint   `json:"user_id" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	user, err := h.tenantService.AcceptInvitation(req.Token, req.UserID)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "failed to accept invitation")
		return
	}

	response.Success(c, user)
}

func (h *TenantHandler) CheckQuota(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	resourceType := c.Query("resource_type")
	if resourceType == "" {
		response.Fail(c, http.StatusBadRequest, "resource_type is required")
		return
	}

	allowed, message := h.tenantService.CheckQuota(uint(tenantID), resourceType)

	response.Success(c, gin.H{
		"allowed": allowed,
		"message": message,
	})
}

func (h *TenantHandler) UpdateQuota(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var req struct {
		ResourceType string `json:"resource_type" binding:"required"`
		Increment    int    `json:"increment" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	if err := h.tenantService.UpdateQuotaUsage(uint(tenantID), req.ResourceType, req.Increment); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to update quota")
		return
	}

	response.Success(c, gin.H{"message": "quota updated successfully"})
}

func (h *TenantHandler) UpdateBillingPlan(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	var req struct {
		Plan  string  `json:"plan" binding:"required"`
		Price float64 `json:"price"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request")
		return
	}

	adminID, _ := c.Get("admin_id")

	if err := h.tenantService.UpdateBillingPlan(uint(tenantID), req.Plan, req.Price, adminID.(uint)); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to update billing plan")
		return
	}

	response.Success(c, gin.H{"message": "billing plan updated successfully"})
}

func (h *TenantHandler) GetTenantUsage(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	stats, err := h.tenantService.GetTenantUsageStats(uint(tenantID))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to get usage stats")
		return
	}

	response.Success(c, stats)
}

func (h *TenantHandler) GetTenantAuditLogs(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	action := c.Query("action")
	resourceType := c.Query("resource_type")

	logs, total, err := h.tenantService.GetTenantAuditLogs(uint(tenantID), page, pageSize, action, resourceType)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to get audit logs")
		return
	}

	response.Success(c, gin.H{
		"items":       logs,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}
