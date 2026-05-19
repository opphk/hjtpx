package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/model"
	"github.com/hjtpx/hjtpx/internal/service"
)

var tenantService = service.NewTenantService()
var tenantQuotaService = service.NewTenantQuotaService()
var tenantBillingService = service.NewTenantBillingService()

// CreateTenant 创建租户
func CreateTenant(c *gin.Context) {
	var tenant model.Tenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tenantService.CreateTenant(c.Request.Context(), &tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, tenant)
}

// GetTenant 获取租户信息
func GetTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	tenant, err := tenantService.GetTenant(c.Request.Context(), uint(id))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if tenant == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Tenant not found"})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// UpdateTenant 更新租户信息
func UpdateTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	var tenant model.Tenant
	if err := c.ShouldBindJSON(&tenant); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	tenant.ID = uint(id)
	if err := tenantService.UpdateTenant(c.Request.Context(), &tenant); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, tenant)
}

// DeleteTenant 删除租户
func DeleteTenant(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	if err := tenantService.DeleteTenant(c.Request.Context(), uint(id)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// ListTenants 列出租户
func ListTenants(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	tenants, total, err := tenantService.ListTenants(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  tenants,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// AddTenantUser 添加租户用户
func AddTenantUser(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	var req struct {
		UserID uint   `json:"user_id"`
		Role   string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tenantService.AddTenantUser(c.Request.Context(), uint(tenantID), req.UserID, req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, gin.H{"message": "User added to tenant successfully"})
}

// RemoveTenantUser 移除租户用户
func RemoveTenantUser(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	if err := tenantService.RemoveTenantUser(c.Request.Context(), uint(tenantID), uint(userID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusNoContent, nil)
}

// UpdateTenantUserRole 更新租户用户角色
func UpdateTenantUserRole(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	userID, err := strconv.ParseUint(c.Param("user_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid user ID"})
		return
	}

	var req struct {
		Role string `json:"role"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tenantService.UpdateTenantUserRole(c.Request.Context(), uint(tenantID), uint(userID), req.Role); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Role updated successfully"})
}

// ListTenantUsers 列出租户用户
func ListTenantUsers(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	users, err := tenantService.ListTenantUsers(c.Request.Context(), uint(tenantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, users)
}

// CreateInvitation 创建邀请
func CreateInvitation(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	var req struct {
		Email     string `json:"email"`
		Role      string `json:"role"`
		InvitedBy uint   `json:"invited_by"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	invitation, err := tenantService.CreateInvitation(c.Request.Context(), uint(tenantID), req.Email, req.Role, req.InvitedBy)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, invitation)
}

// AcceptInvitation 接受邀请
func AcceptInvitation(c *gin.Context) {
	token := c.Param("token")

	var req struct {
		UserID uint `json:"user_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tenantService.AcceptInvitation(c.Request.Context(), token, req.UserID); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation accepted successfully"})
}

// RevokeInvitation 撤销邀请
func RevokeInvitation(c *gin.Context) {
	invitationID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid invitation ID"})
		return
	}

	if err := tenantService.RevokeInvitation(c.Request.Context(), uint(invitationID)); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Invitation revoked successfully"})
}

// GetTenantQuota 获取租户配额
func GetTenantQuota(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	resourceType := c.Query("resource_type")
	if resourceType == "" {
		resourceType = "verification"
	}

	quota, err := tenantQuotaService.GetTenantQuota(c.Request.Context(), uint(tenantID), resourceType)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if quota == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Quota not found"})
		return
	}

	c.JSON(http.StatusOK, quota)
}

// UpdateTenantQuota 更新租户配额
func UpdateTenantQuota(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	resourceType := c.Query("resource_type")
	if resourceType == "" {
		c.JSON(http.StatusBadRequest, gin.H{"error": "resource_type is required"})
		return
	}

	var req struct {
		Limit int64 `json:"limit"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tenantQuotaService.UpdateTenantQuota(c.Request.Context(), uint(tenantID), resourceType, req.Limit); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Quota updated successfully"})
}

// ListTenantQuotas 列出租户配额
func ListTenantQuotas(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	quotas, err := tenantQuotaService.ListTenantQuotas(c.Request.Context(), uint(tenantID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, quotas)
}

// GetTenantBill 获取租户账单
func GetTenantBill(c *gin.Context) {
	billID, err := strconv.ParseUint(c.Param("bill_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bill ID"})
		return
	}

	bill, err := tenantBillingService.GetBill(c.Request.Context(), uint(billID))
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	if bill == nil {
		c.JSON(http.StatusNotFound, gin.H{"error": "Bill not found"})
		return
	}

	c.JSON(http.StatusOK, bill)
}

// ListTenantBills 列出租户账单
func ListTenantBills(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	bills, total, err := tenantBillingService.ListTenantBills(c.Request.Context(), uint(tenantID), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  bills,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// GenerateMonthlyBill 生成月度账单
func GenerateMonthlyBill(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	var req struct {
		Year  int `json:"year"`
		Month int `json:"month"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	bill, err := tenantBillingService.GenerateMonthlyBill(c.Request.Context(), uint(tenantID), req.Year, req.Month)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, bill)
}

// CreatePayment 创建支付记录
func CreatePayment(c *gin.Context) {
	billID, err := strconv.ParseUint(c.Param("bill_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid bill ID"})
		return
	}

	var req struct {
		Amount        float64 `json:"amount"`
		PaymentMethod string  `json:"payment_method"`
		TransactionID string  `json:"transaction_id"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	payment, err := tenantBillingService.CreatePaymentRecord(c.Request.Context(), uint(billID), req.Amount, req.PaymentMethod, req.TransactionID)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusCreated, payment)
}

// UpdatePaymentStatus 更新支付状态
func UpdatePaymentStatus(c *gin.Context) {
	paymentID, err := strconv.ParseUint(c.Param("payment_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid payment ID"})
		return
	}

	var req struct {
		Status       string `json:"status"`
		FailedReason string `json:"failed_reason,omitempty"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": err.Error()})
		return
	}

	if err := tenantBillingService.UpdatePaymentStatus(c.Request.Context(), uint(paymentID), req.Status, req.FailedReason); err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{"message": "Payment status updated successfully"})
}

// ListTenantPayments 列出租户支付记录
func ListTenantPayments(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 32)
	if err != nil {
		c.JSON(http.StatusBadRequest, gin.H{"error": "Invalid tenant ID"})
		return
	}

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	payments, total, err := tenantBillingService.ListTenantPayments(c.Request.Context(), uint(tenantID), page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  payments,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}

// ListAllBills 列出所有账单（管理员）
func ListAllBills(c *gin.Context) {
	status := c.Query("status")
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	bills, total, err := tenantBillingService.ListAllBills(c.Request.Context(), status, page, pageSize)
	if err != nil {
		c.JSON(http.StatusInternalServerError, gin.H{"error": err.Error()})
		return
	}

	c.JSON(http.StatusOK, gin.H{
		"data":  bills,
		"total": total,
		"page":  page,
		"page_size": pageSize,
	})
}
