package admin

import (
	"database/sql"
	"net/http"
	"strconv"
	"time"

	"captchax/internal/model"
	"captchax/internal/repository"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
)

type AdminHandlers struct {
	authService    *AuthService
	adminRepo      *repository.AdminRepo
	whitelistRepo  *repository.WhitelistRepo
	blacklistRepo  *repository.BlacklistRepo
	configRepo     *repository.ConfigRepo
	captchaRepo    *repository.CaptchaRepo
}

func NewAdminHandlers(
	authService *AuthService,
	adminRepo *repository.AdminRepo,
	whitelistRepo *repository.WhitelistRepo,
	blacklistRepo *repository.BlacklistRepo,
	configRepo *repository.ConfigRepo,
	captchaRepo *repository.CaptchaRepo,
) *AdminHandlers {
	return &AdminHandlers{
		authService:   authService,
		adminRepo:     adminRepo,
		whitelistRepo: whitelistRepo,
		blacklistRepo: blacklistRepo,
		configRepo:    configRepo,
		captchaRepo:   captchaRepo,
	}
}

func (h *AdminHandlers) ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "CaptchaX Admin Login",
	})
}

func (h *AdminHandlers) ShowDashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "CaptchaX Admin Dashboard",
	})
}

func (h *AdminHandlers) Login(c *gin.Context) {
	var req model.LoginRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	resp, err := h.authService.Login(c, &req)
	if err != nil {
		response.Error(c, http.StatusUnauthorized, err.Error())
		return
	}

	response.Success(c, resp)
}

func (h *AdminHandlers) Logout(c *gin.Context) {
	response.SuccessWithMessage(c, "logged out successfully", nil)
}

func (h *AdminHandlers) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()

	stats := make(map[string]interface{})

	totalCaptchas, err := h.captchaRepo.GetStats(ctx, time.Now().AddDate(0, 0, -7), time.Now())
	if err == nil {
		stats["captcha_stats"] = totalCaptchas
	}

	whitelistCount, err := h.whitelistRepo.Count(ctx)
	if err == nil {
		stats["whitelist_count"] = whitelistCount
	}

	blacklistCount, err := h.blacklistRepo.Count(ctx, false)
	if err == nil {
		stats["blacklist_count"] = blacklistCount
	}

	adminCount, err := h.adminRepo.Count(ctx)
	if err == nil {
		stats["admin_count"] = adminCount
	}

	sysConfig, err := h.configRepo.GetSystemConfig(ctx)
	if err == nil {
		stats["system_config"] = sysConfig
	}

	recentLogs, err := h.captchaRepo.List(ctx, &model.CaptchaLogFilter{
		Page:     1,
		PageSize: 10,
	})
	if err == nil {
		logs := make([]*model.CaptchaLogDTO, 0, len(recentLogs))
		for _, log := range recentLogs {
			logs = append(logs, log.ToDTO())
		}
		stats["recent_logs"] = logs
	}

	stats["admin_id"] = h.authService.GetAdminID(c)
	stats["username"] = h.authService.GetUsername(c)
	stats["role"] = h.authService.GetRole(c)

	response.Success(c, stats)
}

func (h *AdminHandlers) GetStats(c *gin.Context) {
	ctx := c.Request.Context()

	period := c.DefaultQuery("period", "7d")
	var startDate time.Time
	now := time.Now()

	switch period {
	case "24h":
		startDate = now.AddDate(0, 0, -1)
	case "7d":
		startDate = now.AddDate(0, 0, -7)
	case "30d":
		startDate = now.AddDate(0, 0, -30)
	case "90d":
		startDate = now.AddDate(0, 0, -90)
	default:
		startDate = now.AddDate(0, 0, -7)
	}

	stats, err := h.captchaRepo.GetStats(ctx, startDate, now)
	if err != nil {
		response.InternalError(c, "failed to get stats")
		return
	}

	whitelistCount, _ := h.whitelistRepo.Count(ctx)
	blacklistCount, _ := h.blacklistRepo.Count(ctx, true)

	response.Success(c, gin.H{
		"period":           period,
		"start_date":      startDate.Format(time.RFC3339),
		"end_date":        now.Format(time.RFC3339),
		"captcha_stats":   stats,
		"whitelist_count": whitelistCount,
		"blacklist_count": blacklistCount,
	})
}

func (h *AdminHandlers) GetConfig(c *gin.Context) {
	ctx := c.Request.Context()

	configs, err := h.configRepo.List(ctx)
	if err != nil {
		response.InternalError(c, "failed to get config")
		return
	}

	dtos := make([]*model.ConfigDTO, 0, len(configs))
	for _, cfg := range configs {
		dtos = append(dtos, cfg.ToDTO())
	}

	response.Success(c, gin.H{
		"configs": dtos,
	})
}

func (h *AdminHandlers) UpdateConfig(c *gin.Context) {
	var req model.UpdateConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := c.Request.Context()

	existingCfg, err := h.configRepo.Get(ctx, req.Key)
	if err != nil {
		response.InternalError(c, "failed to check config")
		return
	}

	if existingCfg == nil {
		response.NotFound(c, "config key not found")
		return
	}

	if err := h.configRepo.Update(ctx, req.Key, req.Value); err != nil {
		response.InternalError(c, "failed to update config")
		return
	}

	response.SuccessWithMessage(c, "config updated successfully", nil)
}

func (h *AdminHandlers) GetWhitelist(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ip := c.Query("ip")
	domain := c.Query("domain")

	filter := &model.WhitelistFilter{
		IP:       ip,
		Domain:   domain,
		Page:     page,
		PageSize: pageSize,
	}

	entries, err := h.whitelistRepo.List(ctx, filter)
	if err != nil {
		response.InternalError(c, "failed to get whitelist")
		return
	}

	total, err := h.whitelistRepo.Count(ctx)
	if err != nil {
		total = 0
	}

	dtos := make([]*model.WhitelistDTO, 0, len(entries))
	for _, entry := range entries {
		dtos = append(dtos, entry.ToDTO())
	}

	response.Success(c, gin.H{
		"items":      dtos,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *AdminHandlers) AddWhitelist(c *gin.Context) {
	var req model.CreateWhitelistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := c.Request.Context()

	entry := &model.Whitelist{
		IP: req.IP,
	}
	if req.Domain != "" {
		entry.Domain = sql.NullString{String: req.Domain, Valid: true}
	}
	if req.Reason != "" {
		entry.Reason = sql.NullString{String: req.Reason, Valid: true}
	}

	id, err := h.whitelistRepo.Create(ctx, entry)
	if err != nil {
		response.Error(c, http.StatusConflict, err.Error())
		return
	}

	response.Success(c, gin.H{"id": id})
}

func (h *AdminHandlers) DeleteWhitelist(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	ctx := c.Request.Context()

	entry, err := h.whitelistRepo.GetByID(ctx, id)
	if err != nil {
		response.InternalError(c, "failed to check whitelist")
		return
	}
	if entry == nil {
		response.NotFound(c, "whitelist entry not found")
		return
	}

	if err := h.whitelistRepo.Delete(ctx, id); err != nil {
		response.InternalError(c, "failed to delete whitelist")
		return
	}

	response.SuccessWithMessage(c, "whitelist entry deleted", nil)
}

func (h *AdminHandlers) GetBlacklist(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	ip := c.Query("ip")
	activeOnly := c.Query("active_only") == "true"

	filter := &model.BlacklistFilter{
		IP:         ip,
		ActiveOnly: activeOnly,
		Page:       page,
		PageSize:   pageSize,
	}

	entries, err := h.blacklistRepo.List(ctx, filter)
	if err != nil {
		response.InternalError(c, "failed to get blacklist")
		return
	}

	total, err := h.blacklistRepo.Count(ctx, false)
	if err != nil {
		total = 0
	}

	dtos := make([]*model.BlacklistDTO, 0, len(entries))
	for _, entry := range entries {
		dtos = append(dtos, entry.ToDTO())
	}

	response.Success(c, gin.H{
		"items":       dtos,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

func (h *AdminHandlers) AddBlacklist(c *gin.Context) {
	var req model.CreateBlacklistRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	ctx := c.Request.Context()

	entry := &model.Blacklist{
		IP: req.IP,
	}
	if req.Reason != "" {
		entry.Reason = sql.NullString{String: req.Reason, Valid: true}
	}
	if req.ExpireAt != nil {
		entry.ExpireAt = sql.NullTime{Time: *req.ExpireAt, Valid: true}
	}

	id, err := h.blacklistRepo.Create(ctx, entry)
	if err != nil {
		response.InternalError(c, "failed to add blacklist")
		return
	}

	response.Success(c, gin.H{"id": id})
}

func (h *AdminHandlers) DeleteBlacklist(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid id")
		return
	}

	ctx := c.Request.Context()

	entry, err := h.blacklistRepo.GetByID(ctx, id)
	if err != nil {
		response.InternalError(c, "failed to check blacklist")
		return
	}
	if entry == nil {
		response.NotFound(c, "blacklist entry not found")
		return
	}

	if err := h.blacklistRepo.Delete(ctx, id); err != nil {
		response.InternalError(c, "failed to delete blacklist")
		return
	}

	response.SuccessWithMessage(c, "blacklist entry deleted", nil)
}
