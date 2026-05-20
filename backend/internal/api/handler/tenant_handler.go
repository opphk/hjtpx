package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	github.com/hjtpx/hjtpx/internal/service"
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

func (h *TenantHandler) ActivateTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	adminID, _ := c.Get("admin_id")

	updates := map[string]interface{}{"status": "active"}
	_, err = h.tenantService.UpdateTenant(uint(id), updates, adminID.(uint))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to activate tenant")
		return
	}

	response.Success(c, gin.H{"message": "tenant activated successfully"})
}

func (h *TenantHandler) DeactivateTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	adminID, _ := c.Get("admin_id")

	updates := map[string]interface{}{"status": "inactive"}
	_, err = h.tenantService.UpdateTenant(uint(id), updates, adminID.(uint))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to deactivate tenant")
		return
	}

	response.Success(c, gin.H{"message": "tenant deactivated successfully"})
}

func (h *TenantHandler) GetTenantStats(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid tenant id")
		return
	}

	stats, err := h.tenantService.GetTenantUsageStats(uint(id))
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to get tenant stats")
		return
	}

	response.Success(c, stats)
}
