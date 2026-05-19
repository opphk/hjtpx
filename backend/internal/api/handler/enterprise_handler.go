package handler

import (
	"net/http"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type EnterpriseHandler struct {
	enterpriseService *service.EnterpriseService
	tenantService     *service.TenantService
}

func NewEnterpriseHandler(enterpriseService *service.EnterpriseService, tenantService *service.TenantService) *EnterpriseHandler {
	return &EnterpriseHandler{
		enterpriseService: enterpriseService,
		tenantService:     tenantService,
	}
}

func (h *EnterpriseHandler) GetSSOConfig(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	config, err := h.enterpriseService.GetSSOConfig(tenantID.(uint))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "SSO config not found", err.Error())
		return
	}

	response.Success(c, config)
}

func (h *EnterpriseHandler) CreateOrUpdateSSOConfig(c *gin.Context) {
	var req struct {
		Provider         string `json:"provider"`
		EntityID         string `json:"entity_id"`
		SSOURL           string `json:"sso_url"`
		Certificate      string `json:"certificate"`
		ClientID         string `json:"client_id"`
		ClientSecret     string `json:"client_secret"`
		AuthorizationURL string `json:"authorization_url"`
		TokenURL         string `json:"token_url"`
		UserinfoURL      string `json:"userinfo_url"`
		Scopes           string `json:"scopes"`
		Attributes       string `json:"attributes"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	config := &models.SSOConfig{
		Provider:         req.Provider,
		EntityID:         req.EntityID,
		SSOURL:           req.SSOURL,
		Certificate:      req.Certificate,
		ClientID:         req.ClientID,
		ClientSecret:     req.ClientSecret,
		AuthorizationURL: req.AuthorizationURL,
		TokenURL:         req.TokenURL,
		UserinfoURL:      req.UserinfoURL,
		Scopes:           req.Scopes,
		Attributes:      req.Attributes,
	}

	if err := h.enterpriseService.CreateOrUpdateSSOConfig(tenantID.(uint), config); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to save SSO config", err.Error())
		return
	}

	response.Success(c, gin.H{"message": "SSO config saved successfully"})
}

func (h *EnterpriseHandler) EnableSSO(c *gin.Context) {
	var req struct {
		Provider string `json:"provider" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	if err := h.enterpriseService.EnableSSO(tenantID.(uint), req.Provider); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to enable SSO", err.Error())
		return
	}

	response.Success(c, gin.H{"message": "SSO enabled successfully"})
}

func (h *EnterpriseHandler) DisableSSO(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	if err := h.enterpriseService.DisableSSO(tenantID.(uint)); err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to disable SSO", err.Error())
		return
	}

	response.Success(c, gin.H{"message": "SSO disabled successfully"})
}

func (h *EnterpriseHandler) InitiateSSO(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	config, err := h.enterpriseService.GetSSOConfig(tenantID.(uint))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "SSO config not found", err.Error())
		return
	}

	var provider service.SSOProvider
	switch config.Provider {
	case "saml":
		provider = h.enterpriseService.SAMLProvider(config)
	case "oauth2":
		provider = h.enterpriseService.OAuth2Provider(config)
	case "oidc":
		provider = h.enterpriseService.OIDCProvider(config)
	default:
		response.Fail(c, http.StatusBadRequest, "unsupported SSO provider", nil)
		return
	}

	authURL, err := provider.InitiateAuth()
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to initiate SSO", err.Error())
		return
	}

	response.Success(c, gin.H{"redirect_url": authURL})
}

func (h *EnterpriseHandler) HandleSSOCallback(c *gin.Context) {
	code := c.Query("code")
	if code == "" {
		response.Fail(c, http.StatusBadRequest, "authorization code required", nil)
		return
	}

	tenantID, _ := c.Get("tenant_id")

	config, err := h.enterpriseService.GetSSOConfig(tenantID.(uint))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "SSO config not found", err.Error())
		return
	}

	var provider service.SSOProvider
	switch config.Provider {
	case "oauth2":
		provider = h.enterpriseService.OAuth2Provider(config)
	case "oidc":
		provider = h.enterpriseService.OIDCProvider(config)
	default:
		response.Fail(c, http.StatusBadRequest, "unsupported SSO provider", nil)
		return
	}

	user, err := provider.HandleCallback(code)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to handle callback", err.Error())
		return
	}

	scimService := service.NewSCIMService(nil)
	scimUser, err := scimService.CreateSCIMUser(tenantID.(uint), user)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to create user", err.Error())
		return
	}

	response.Success(c, gin.H{
		"user":      user,
		"scim_user": scimUser,
	})
}

func (h *EnterpriseHandler) SyncSCIMUsers(c *gin.Context) {
	var req struct {
		Provider    string `json:"provider" binding:"required"`
		BaseURL     string `json:"base_url" binding:"required"`
		BearerToken string `json:"bearer_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	scimService := service.NewSCIMService(nil)
	result, err := scimService.SyncUsers(tenantID.(uint), req.Provider, req.BaseURL, req.BearerToken)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to sync users", err.Error())
		return
	}

	response.Success(c, result)
}

func (h *EnterpriseHandler) SyncSCIMGroups(c *gin.Context) {
	var req struct {
		Provider    string `json:"provider" binding:"required"`
		BaseURL     string `json:"base_url" binding:"required"`
		BearerToken string `json:"bearer_token" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	scimService := service.NewSCIMService(nil)
	result, err := scimService.SyncGroups(tenantID.(uint), req.BaseURL, req.BearerToken)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to sync groups", err.Error())
		return
	}

	response.Success(c, result)
}

func (h *EnterpriseHandler) GetAuditLogs(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filters := make(map[string]interface{})
	if method := c.Query("method"); method != "" {
		filters["method"] = method
	}
	if endpoint := c.Query("endpoint"); endpoint != "" {
		filters["endpoint"] = endpoint
	}
	if status := c.Query("status"); status != "" {
		filters["status"], _ = strconv.Atoi(status)
	}
	if ip := c.Query("ip_address"); ip != "" {
		filters["ip_address"] = ip
	}
	if startDate := c.Query("start_date"); startDate != "" {
		filters["start_date"] = startDate
	}
	if endDate := c.Query("end_date"); endDate != "" {
		filters["end_date"] = endDate
	}

	auditService := service.NewAPIAuditService(nil)
	logs, total, err := auditService.GetAuditLogs(tenantID.(uint), page, pageSize, filters)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to get audit logs", err.Error())
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

func (h *EnterpriseHandler) GetAuditStats(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	startDate := c.DefaultQuery("start_date", "")
	endDate := c.DefaultQuery("end_date", "")

	auditService := service.NewAPIAuditService(nil)
	stats, err := auditService.GetAuditStats(tenantID.(uint), startDate, endDate)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to get audit stats", err.Error())
		return
	}

	response.Success(c, stats)
}

func (h *EnterpriseHandler) CreateComplianceReport(c *gin.Context) {
	var req struct {
		ReportType   string `json:"report_type" binding:"required"`
		PeriodStart  string `json:"period_start" binding:"required"`
		PeriodEnd    string `json:"period_end" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid request", err.Error())
		return
	}

	startDate, err := time.Parse("2006-01-02", req.PeriodStart)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid start date format", err.Error())
		return
	}

	endDate, err := time.Parse("2006-01-02", req.PeriodEnd)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid end date format", err.Error())
		return
	}

	tenantID, _ := c.Get("tenant_id")

	complianceService := service.NewComplianceService(nil)
	report, err := complianceService.CreateReport(tenantID.(uint), req.ReportType, startDate, endDate)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to create report", err.Error())
		return
	}

	go complianceService.GenerateReport(report.ID)

	response.Success(c, gin.H{
		"report": report,
		"message": "Report generation started",
	})
}

func (h *EnterpriseHandler) GetComplianceReports(c *gin.Context) {
	tenantID, _ := c.Get("tenant_id")

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	complianceService := service.NewComplianceService(nil)
	reports, total, err := complianceService.GetReports(tenantID.(uint), page, pageSize)
	if err != nil {
		response.Fail(c, http.StatusInternalServerError, "failed to get reports", err.Error())
		return
	}

	response.Success(c, gin.H{
		"items":       reports,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *EnterpriseHandler) DownloadComplianceReport(c *gin.Context) {
	reportID, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.Fail(c, http.StatusBadRequest, "invalid report ID", err.Error())
		return
	}

	complianceService := service.NewComplianceService(nil)
	filePath, err := complianceService.DownloadReport(uint(reportID))
	if err != nil {
		response.Fail(c, http.StatusNotFound, "report not found or not ready", err.Error())
		return
	}

	c.File(filePath)
}
