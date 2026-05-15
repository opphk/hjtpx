package admin

import (
	"context"
	"database/sql"
	"fmt"
	"net/http"
	"strconv"
	"time"

	"captchax/internal/model"
	"captchax/internal/monitoring"
	"captchax/internal/repository"
	"captchax/pkg/response"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
)

// AdminHandlers 管理后台处理器
type AdminHandlers struct {
	authService      *AuthService
	adminRepo        *repository.AdminRepo
	whitelistRepo    *repository.WhitelistRepo
	blacklistRepo    *repository.BlacklistRepo
	configRepo       *repository.ConfigRepo
	captchaRepo      *repository.CaptchaRepo
	analyticsService *AnalyticsService
	metrics          *monitoring.Metrics
}

// NewAdminHandlers 创建管理后台处理器
func NewAdminHandlers(
	authService *AuthService,
	adminRepo *repository.AdminRepo,
	whitelistRepo *repository.WhitelistRepo,
	blacklistRepo *repository.BlacklistRepo,
	configRepo *repository.ConfigRepo,
	captchaRepo *repository.CaptchaRepo,
	analyticsService *AnalyticsService,
	metrics *monitoring.Metrics,
) *AdminHandlers {
	handlers := &AdminHandlers{
		authService:      authService,
		adminRepo:        adminRepo,
		whitelistRepo:    whitelistRepo,
		blacklistRepo:    blacklistRepo,
		configRepo:       configRepo,
		captchaRepo:      captchaRepo,
		analyticsService: analyticsService,
		metrics:          metrics,
	}

	hub := GetRealtimeHub()
	hub.SetMetrics(metrics)
	hub.Run()

	return handlers
}

// ShowLoginPage 显示登录页面
func (h *AdminHandlers) ShowLoginPage(c *gin.Context) {
	c.HTML(http.StatusOK, "login.html", gin.H{
		"title": "CaptchaX Admin Login",
	})
}

// ShowDashboardPage 显示仪表盘页面
func (h *AdminHandlers) ShowDashboardPage(c *gin.Context) {
	c.HTML(http.StatusOK, "dashboard.html", gin.H{
		"title": "CaptchaX Admin Dashboard",
	})
}

// ShowStatsPage 显示统计页面
func (h *AdminHandlers) ShowStatsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "stats.html", gin.H{
		"title": "CaptchaX Admin Statistics",
	})
}

// ShowAnalyticsPage 显示高级分析页面
func (h *AdminHandlers) ShowAnalyticsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "analytics.html", gin.H{
		"title": "CaptchaX Admin Analytics",
	})
}

// ShowConfigPage 显示配置页面
func (h *AdminHandlers) ShowConfigPage(c *gin.Context) {
	c.HTML(http.StatusOK, "config.html", gin.H{
		"title": "CaptchaX Admin Configuration",
	})
}

// ShowWhitelistPage 显示白名单页面
func (h *AdminHandlers) ShowWhitelistPage(c *gin.Context) {
	c.HTML(http.StatusOK, "whitelist.html", gin.H{
		"title": "CaptchaX Admin Whitelist",
	})
}

// ShowBlacklistPage 显示黑名单页面
func (h *AdminHandlers) ShowBlacklistPage(c *gin.Context) {
	c.HTML(http.StatusOK, "blacklist.html", gin.H{
		"title": "CaptchaX Admin Blacklist",
	})
}

// ShowRealtimePage 显示实时监控页面
func (h *AdminHandlers) ShowRealtimePage(c *gin.Context) {
	c.HTML(http.StatusOK, "realtime.html", gin.H{
		"title": "CaptchaX Admin Realtime Monitor",
	})
}

// Login 登录
func (h *AdminHandlers) ShowAdminsPage(c *gin.Context) {
	c.HTML(http.StatusOK, "admins.html", gin.H{
		"title": "CaptchaX Admin Management",
	})
}

// ShowRolesPage 显示角色管理页面
func (h *AdminHandlers) ShowRolesPage(c *gin.Context) {
	c.HTML(http.StatusOK, "roles.html", gin.H{
		"title": "CaptchaX Role Management",
	})
}

// Login 登录
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

// Logout 登出
func (h *AdminHandlers) Logout(c *gin.Context) {
	response.SuccessWithMessage(c, "logged out successfully", nil)
}

// GetDashboard 获取仪表盘数据
func (h *AdminHandlers) GetDashboard(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()

	stats := make(map[string]interface{})

	// 获取今日统计
	todayStart := time.Date(now.Year(), now.Month(), now.Day(), 0, 0, 0, 0, now.Location())
	yesterdayStart := todayStart.AddDate(0, 0, -1)
	todayStats, err := h.captchaRepo.GetStats(ctx, todayStart, now)
	if err == nil {
		stats["today_verifications"] = todayStats.TotalCount
		stats["success_rate"] = calculateSuccessRate(todayStats)
	}

	// 获取昨日统计用于对比
	yesterdayStats, err := h.captchaRepo.GetStats(ctx, yesterdayStart, todayStart)
	if err == nil {
		stats["yesterday_verifications"] = yesterdayStats.TotalCount
		stats["verifications_change"] = calculateChange(todayStats.TotalCount, yesterdayStats.TotalCount)
	}

	// 获取7天统计
	weekStart := now.AddDate(0, 0, -7)
	weekStats, err := h.captchaRepo.GetStats(ctx, weekStart, now)
	if err == nil {
		stats["captcha_stats"] = weekStats
		stats["blocked_attacks"] = weekStats.FailCount
	}

	// 获取白名单、黑名单、管理员数量
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

	// 获取活跃用户数
	activeUsers, err := h.captchaRepo.CountUniqueIPs(ctx, weekStart, now)
	if err == nil {
		stats["active_users"] = activeUsers
	}

	// 获取系统配置
	sysConfig, err := h.configRepo.GetSystemConfig(ctx)
	if err == nil {
		stats["system_config"] = sysConfig
	}

	// 获取最近日志
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

	// 管理员信息
	stats["admin_id"] = h.authService.GetAdminID(c)
	stats["username"] = h.authService.GetUsername(c)
	stats["role"] = h.authService.GetRole(c)

	response.Success(c, stats)
}

// GetStats 获取统计数据
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

	// 获取趋势数据
	trend, err := h.captchaRepo.GetTrend(ctx, startDate, now)
	if err != nil {
		trend = []model.TrendPoint{}
	}

	// 获取验证码类型分布
	distribution, err := h.captchaRepo.GetTypeDistribution(ctx, startDate, now)
	if err != nil {
		distribution = []model.TypeDistribution{}
	}

	// 获取IP排行
	ipRanking, err := h.captchaRepo.GetIPRanking(ctx, startDate, now, 20)
	if err != nil {
		ipRanking = []model.IPRanking{}
	}

	response.Success(c, gin.H{
		"period":             period,
		"start_date":         startDate.Format(time.RFC3339),
		"end_date":           now.Format(time.RFC3339),
		"captcha_stats":      stats,
		"whitelist_count":    whitelistCount,
		"blacklist_count":    blacklistCount,
		"trend":              trend,
		"captcha_distribution": distribution,
		"ip_ranking":         ipRanking,
	})
}

// GetTrend 获取趋势数据
func (h *AdminHandlers) GetTrend(c *gin.Context) {
	ctx := c.Request.Context()

	hoursStr := c.DefaultQuery("hours", "24")
	hours, err := strconv.Atoi(hoursStr)
	if err != nil || hours <= 0 {
		hours = 24
	}

	now := time.Now()
	startDate := now.Add(time.Duration(-hours) * time.Hour)

	trend, err := h.captchaRepo.GetTrend(ctx, startDate, now)
	if err != nil {
		response.InternalError(c, "failed to get trend data")
		return
	}

	response.Success(c, gin.H{
		"trend": trend,
	})
}

// GetCaptchaDistribution 获取验证码类型分布
func (h *AdminHandlers) GetCaptchaDistribution(c *gin.Context) {
	ctx := c.Request.Context()

	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	distribution, err := h.captchaRepo.GetTypeDistribution(ctx, startDate, now)
	if err != nil {
		response.InternalError(c, "failed to get distribution data")
		return
	}

	response.Success(c, distribution)
}

// GetIPRanking 获取IP排行
func (h *AdminHandlers) GetIPRanking(c *gin.Context) {
	ctx := c.Request.Context()

	limitStr := c.DefaultQuery("limit", "20")
	limit, err := strconv.Atoi(limitStr)
	if err != nil || limit <= 0 {
		limit = 20
	}

	daysStr := c.DefaultQuery("days", "30")
	days, err := strconv.Atoi(daysStr)
	if err != nil || days <= 0 {
		days = 30
	}

	now := time.Now()
	startDate := now.AddDate(0, 0, -days)

	ranking, err := h.captchaRepo.GetIPRanking(ctx, startDate, now, limit)
	if err != nil {
		response.InternalError(c, "failed to get IP ranking")
		return
	}

	// 添加状态信息
	for i := range ranking {
		ip := ranking[i].IP
		isWhitelisted, _ := h.whitelistRepo.IsWhitelisted(ctx, ip)
		isBlacklisted, _ := h.blacklistRepo.IsBlacklisted(ctx, ip)

		switch {
		case isWhitelisted:
			ranking[i].Status = "whitelisted"
		case isBlacklisted:
			ranking[i].Status = "blocked"
		case ranking[i].SuccessRate < 50:
			ranking[i].Status = "suspicious"
		default:
			ranking[i].Status = "normal"
		}
	}

	response.Success(c, ranking)
}

// GetConfig 获取配置
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

// UpdateConfig 更新配置
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

// GetWhitelist 获取白名单
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
		"items":       dtos,
		"total":       total,
		"page":        page,
		"page_size":   pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// AddWhitelist 添加白名单
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

// DeleteWhitelist 删除白名单
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

// GetBlacklist 获取黑名单
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

// AddBlacklist 添加黑名单
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

// DeleteBlacklist 删除黑名单
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

// 辅助函数：计算成功率
func calculateSuccessRate(stats *model.CaptchaLogStats) float64 {
	if stats.TotalCount == 0 {
		return 0
	}
	return (float64(stats.SuccessCount) / float64(stats.TotalCount)) * 100
}

// 辅助函数：计算变化百分比
func calculateChange(current, previous int64) float64 {
	if previous == 0 {
		if current > 0 {
			return 100.0
		}
		return 0
	}
	return (float64(current-previous) / float64(previous)) * 100
}

// GetAnalyticsOverview 获取数据分析概览
func (h *AdminHandlers) GetAnalyticsOverview(c *gin.Context) {
	ctx := c.Request.Context()

	rangeType := c.DefaultQuery("range", "7d")
	timeRange := ParseTimeRange(rangeType)

	var prevRange TimeRange
	switch rangeType {
	case "today":
		prevRange = TimeRange{
			Start: timeRange.Start.AddDate(0, 0, -1),
			End:   timeRange.Start.Add(-time.Second),
			Label: "yesterday",
		}
	case "yesterday":
		prevRange = TimeRange{
			Start: timeRange.Start.AddDate(0, 0, -1),
			End:   timeRange.Start.Add(-time.Second),
			Label: "yesterday",
		}
	case "7d":
		prevRange = TimeRange{
			Start: timeRange.Start.AddDate(0, 0, -7),
			End:   timeRange.Start.Add(-time.Second),
			Label: "prev_7d",
		}
	case "30d":
		prevRange = TimeRange{
			Start: timeRange.Start.AddDate(0, 0, -30),
			End:   timeRange.Start.Add(-time.Second),
			Label: "prev_30d",
		}
	case "90d":
		prevRange = TimeRange{
			Start: timeRange.Start.AddDate(0, 0, -90),
			End:   timeRange.Start.Add(-time.Second),
			Label: "prev_90d",
		}
	default:
		prevRange = TimeRange{
			Start: timeRange.Start.AddDate(0, 0, -7),
			End:   timeRange.Start.Add(-time.Second),
			Label: "prev",
		}
	}

	stats, err := h.analyticsService.GetOverview(ctx, timeRange, prevRange)
	if err != nil {
		response.InternalError(c, "failed to get analytics overview")
		return
	}

	response.Success(c, stats)
}

// GetAnalyticsTrends 获取趋势分析数据
func (h *AdminHandlers) GetAnalyticsTrends(c *gin.Context) {
	ctx := c.Request.Context()

	rangeType := c.DefaultQuery("range", "7d")
	interval := c.DefaultQuery("interval", "")

	timeRange := ParseTimeRange(rangeType)

	if customStart := c.Query("start_date"); customStart != "" {
		if customEnd := c.Query("end_date"); customEnd != "" {
			customRange, err := ParseCustomTimeRange(customStart, customEnd)
			if err == nil {
				timeRange = customRange
			}
		}
	}

	trends, err := h.analyticsService.GetTrends(ctx, timeRange, interval)
	if err != nil {
		response.InternalError(c, "failed to get trends data")
		return
	}

	response.Success(c, gin.H{
		"time_range": timeRange.Label,
		"start_date": timeRange.Start.Format("2006-01-02"),
		"end_date":   timeRange.End.Format("2006-01-02"),
		"trends":     trends,
	})
}

// GetAnalyticsDistribution 获取分布分析数据
func (h *AdminHandlers) GetAnalyticsDistribution(c *gin.Context) {
	ctx := c.Request.Context()

	rangeType := c.DefaultQuery("range", "7d")
	groupBy := c.DefaultQuery("group_by", "type")

	timeRange := ParseTimeRange(rangeType)

	distribution, err := h.analyticsService.GetDistribution(ctx, timeRange, groupBy)
	if err != nil {
		response.InternalError(c, "failed to get distribution data")
		return
	}

	response.Success(c, gin.H{
		"time_range": timeRange.Label,
		"group_by":   groupBy,
		"data":       distribution,
	})
}

// GetAnalyticsGeo 获取地域分布数据
func (h *AdminHandlers) GetAnalyticsGeo(c *gin.Context) {
	ctx := c.Request.Context()

	rangeType := c.DefaultQuery("range", "7d")
	limitStr := c.DefaultQuery("limit", "20")
	limit, _ := strconv.Atoi(limitStr)
	if limit <= 0 || limit > 100 {
		limit = 20
	}

	timeRange := ParseTimeRange(rangeType)

	geoData, err := h.analyticsService.GetGeoDistribution(ctx, timeRange, limit)
	if err != nil {
		response.InternalError(c, "failed to get geo data")
		return
	}

	response.Success(c, gin.H{
		"time_range": timeRange.Label,
		"data":       geoData,
	})
}

// GetAnalyticsDevices 获取设备分析数据
func (h *AdminHandlers) GetAnalyticsDevices(c *gin.Context) {
	ctx := c.Request.Context()

	rangeType := c.DefaultQuery("range", "7d")
	timeRange := ParseTimeRange(rangeType)

	deviceStats, err := h.analyticsService.GetDeviceDistribution(ctx, timeRange)
	if err != nil {
		response.InternalError(c, "failed to get device data")
		return
	}

	response.Success(c, gin.H{
		"time_range": timeRange.Label,
		"data":       deviceStats,
	})
}

// GetAnalyticsRisk 获取风险分布数据
func (h *AdminHandlers) GetAnalyticsRisk(c *gin.Context) {
	ctx := c.Request.Context()

	rangeType := c.DefaultQuery("range", "7d")
	timeRange := ParseTimeRange(rangeType)

	riskStats, err := h.analyticsService.GetRiskDistribution(ctx, timeRange)
	if err != nil {
		response.InternalError(c, "failed to get risk data")
		return
	}

	response.Success(c, gin.H{
		"time_range": timeRange.Label,
		"data":       riskStats,
	})
}

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin: func(r *http.Request) bool {
		return true
	},
}

func (h *AdminHandlers) HandleWebSocket(c *gin.Context) {
	hub := GetRealtimeHub()
	hub.SetContext(c.Request.Context())
	if h.captchaRepo != nil {
		hub.SetCaptchaRepo(h.captchaRepo)
	}

	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "failed to upgrade connection")
		return
	}

	client := &WebSocketClient{
		hub:  hub,
		conn: conn,
		send: make(chan []byte, 256),
		quit: make(chan struct{}),
	}

	hub.Register(client)

	go client.writePump()
	go client.readPump()
}

func (h *AdminHandlers) GetRealtimeStats(c *gin.Context) {
	ctx := c.Request.Context()
	now := time.Now()
	windowStart := now.Add(-1 * time.Minute)

	stats, err := h.captchaRepo.GetStats(ctx, windowStart, now)
	if err != nil {
		response.InternalError(c, "failed to get realtime stats")
		return
	}

	successRate := 0.0
	if stats.TotalCount > 0 {
		successRate = float64(stats.SuccessCount) / float64(stats.TotalCount) * 100
	}

	trend, err := h.captchaRepo.GetTrend(ctx, windowStart, now)
	if err != nil {
		trend = []model.TrendPoint{}
	}

	hub := GetRealtimeHub()

	response.Success(c, gin.H{
		"requests_per_second":  0,
		"success_rate":        successRate,
		"total_verifications": stats.TotalCount,
		"verified":            stats.SuccessCount,
		"rejected":            stats.FailCount,
		"active_connections":  hub.getClientCount(),
		"chart_data":          trend,
		"timestamp":           now.Unix(),
	})
}

func (h *AdminHandlers) GetRealtimeCharts(c *gin.Context) {
	ctx := c.Request.Context()
	period := c.DefaultQuery("period", "5m")

	var duration time.Duration
	switch period {
	case "1m":
		duration = 1 * time.Minute
	case "5m":
		duration = 5 * time.Minute
	case "15m":
		duration = 15 * time.Minute
	case "1h":
		duration = 1 * time.Hour
	default:
		duration = 5 * time.Minute
	}

	now := time.Now()
	startTime := now.Add(-duration)

	trend, err := h.captchaRepo.GetTrend(ctx, startTime, now)
	if err != nil {
		trend = []model.TrendPoint{}
	}

	distribution, err := h.captchaRepo.GetTypeDistribution(ctx, startTime, now)
	if err != nil {
		distribution = []model.TypeDistribution{}
	}

	stats, err := h.captchaRepo.GetStats(ctx, startTime, now)
	if err != nil {
		stats = &model.CaptchaLogStats{}
	}

	response.Success(c, gin.H{
		"period":       period,
		"start_time":   startTime.Format(time.RFC3339),
		"end_time":     now.Format(time.RFC3339),
		"trend":        trend,
		"distribution": distribution,
		"summary": gin.H{
			"total":     stats.TotalCount,
			"verified":  stats.SuccessCount,
			"rejected":  stats.FailCount,
			"success_rate": 0.0,
		},
	})
}

// RBACHandlers RBAC 权限管理处理器
type RBACHandlers struct {
	rbacService *RBACService
}

// NewRBACHandlers 创建 RBAC 处理器
func NewRBACHandlers(rbacService *RBACService) *RBACHandlers {
	return &RBACHandlers{
		rbacService: rbacService,
	}
}

// GetAdmins 获取管理员列表
func (h *RBACHandlers) GetAdmins(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))
	username := c.Query("username")
	email := c.Query("email")
	role := c.Query("role")

	filter := &model.AdminFilter{
		Username: username,
		Email:    email,
		Role:     role,
		Page:     page,
		PageSize: pageSize,
	}

	admins, total, err := h.rbacService.ListAdmins(ctx, filter)
	if err != nil {
		response.InternalError(c, "failed to get admins")
		return
	}

	dtos := make([]*model.AdminDTO, 0, len(admins))
	for _, admin := range admins {
		dtos = append(dtos, admin.ToDTO())
	}

	response.Success(c, gin.H{
		"items":       dtos,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetAdmin 获取单个管理员
func (h *RBACHandlers) GetAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid admin id")
		return
	}

	ctx := c.Request.Context()
	admin, err := h.rbacService.GetAdmin(ctx, id)
	if err != nil {
		response.NotFound(c, "admin not found")
		return
	}

	response.Success(c, admin.ToDTO())
}

// CreateAdmin 创建管理员
func (h *RBACHandlers) CreateAdmin(c *gin.Context) {
	var req model.CreateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	admin, err := h.rbacService.CreateAdmin(ctx, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, admin.ToDTO())
}

// UpdateAdmin 更新管理员
func (h *RBACHandlers) UpdateAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid admin id")
		return
	}

	var req model.UpdateAdminRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	admin, err := h.rbacService.UpdateAdmin(ctx, id, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, admin.ToDTO())
}

// DeleteAdmin 删除管理员
func (h *RBACHandlers) DeleteAdmin(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid admin id")
		return
	}

	ctx := c.Request.Context()
	if err := h.rbacService.DeleteAdmin(ctx, id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.SuccessWithMessage(c, "admin deleted successfully", nil)
}

// GetRoles 获取角色列表
func (h *RBACHandlers) GetRoles(c *gin.Context) {
	ctx := c.Request.Context()

	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "20"))

	filter := &model.RoleFilter{
		Page:    page,
		PageSize: pageSize,
	}

	roles, total, err := h.rbacService.ListRoles(ctx, filter)
	if err != nil {
		response.InternalError(c, "failed to get roles")
		return
	}

	dtos := make([]*model.RoleDTO, 0, len(roles))
	for _, role := range roles {
		dtos = append(dtos, role.ToDTO())
	}

	response.Success(c, gin.H{
		"items":       dtos,
		"total":      total,
		"page":       page,
		"page_size":  pageSize,
		"total_pages": (total + int64(pageSize) - 1) / int64(pageSize),
	})
}

// GetRole 获取单个角色
func (h *RBACHandlers) GetRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}

	ctx := c.Request.Context()
	role, err := h.rbacService.GetRole(ctx, id)
	if err != nil {
		response.NotFound(c, "role not found")
		return
	}

	response.Success(c, role.ToDTO())
}

// CreateRole 创建角色
func (h *RBACHandlers) CreateRole(c *gin.Context) {
	var req model.CreateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	role, err := h.rbacService.CreateRole(ctx, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, role.ToDTO())
}

// UpdateRole 更新角色
func (h *RBACHandlers) UpdateRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}

	var req model.UpdateRoleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	role, err := h.rbacService.UpdateRole(ctx, id, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, role.ToDTO())
}

// DeleteRole 删除角色
func (h *RBACHandlers) DeleteRole(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid role id")
		return
	}

	ctx := c.Request.Context()
	if err := h.rbacService.DeleteRole(ctx, id); err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.SuccessWithMessage(c, "role deleted successfully", nil)
}

// GetPermissions 获取权限列表
func (h *RBACHandlers) GetPermissions(c *gin.Context) {
	ctx := c.Request.Context()

	permissions, err := h.rbacService.ListPermissions(ctx)
	if err != nil {
		response.InternalError(c, "failed to get permissions")
		return
	}

	dtos := make([]*model.PermissionDTO, 0, len(permissions))
	for _, perm := range permissions {
		dtos = append(dtos, perm.ToDTO())
	}

	response.Success(c, dtos)
}

// ExportHandlers 数据导出处理器
type ExportHandlers struct {
	exportService *ExportService
}

func NewExportHandlers(exportService *ExportService) *ExportHandlers {
	return &ExportHandlers{
		exportService: exportService,
	}
}

// ExportCaptchas 导出验证码数据
func (h *ExportHandlers) ExportCaptchas(c *gin.Context) {
	ctx := c.Request.Context()
	req := ParseExportRequest(c)
	req.Type = "captchas"

	result, err := h.exportService.ExportCaptchas(ctx, req)
	if err != nil {
		response.InternalError(c, "failed to export captchas: "+err.Error())
		return
	}

	data, err := SerializeToFormat(result.Data, req.Format)
	if err != nil {
		response.InternalError(c, "failed to serialize data: "+err.Error())
		return
	}

	filename := fmt.Sprintf("%s.%s", result.FileName, req.Format)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, result.MimeType, data)
}

// ExportStats 导出统计数据
func (h *ExportHandlers) ExportStats(c *gin.Context) {
	ctx := c.Request.Context()
	req := ParseExportRequest(c)
	req.Type = "stats"

	result, err := h.exportService.ExportStats(ctx, req)
	if err != nil {
		response.InternalError(c, "failed to export stats: "+err.Error())
		return
	}

	data, err := SerializeToFormat(result.Data, req.Format)
	if err != nil {
		response.InternalError(c, "failed to serialize data: "+err.Error())
		return
	}

	filename := fmt.Sprintf("%s.%s", result.FileName, req.Format)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, result.MimeType, data)
}

// ExportLogs 导出日志数据
func (h *ExportHandlers) ExportLogs(c *gin.Context) {
	ctx := c.Request.Context()
	req := ParseExportRequest(c)
	req.Type = "logs"

	result, err := h.exportService.ExportLogs(ctx, req)
	if err != nil {
		response.InternalError(c, "failed to export logs: "+err.Error())
		return
	}

	data, err := SerializeToFormat(result.Data, req.Format)
	if err != nil {
		response.InternalError(c, "failed to serialize data: "+err.Error())
		return
	}

	filename := fmt.Sprintf("%s.%s", result.FileName, req.Format)
	c.Header("Content-Description", "File Transfer")
	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, result.MimeType, data)
}

// GetExportCount 获取导出数据量
func (h *ExportHandlers) GetExportCount(c *gin.Context) {
	ctx := c.Request.Context()
	req := ParseExportRequest(c)

	count, err := h.exportService.GetExportCount(ctx, req.Type, req)
	if err != nil {
		response.InternalError(c, "failed to get export count: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"count":       count,
		"estimated_size": count * 200,
	})
}

// AssignRoles 给管理员分配角色
func (h *RBACHandlers) AssignRoles(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseInt(idStr, 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid admin id")
		return
	}

	var req model.AssignRolesRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body: "+err.Error())
		return
	}

	ctx := c.Request.Context()
	admin, err := h.rbacService.UpdateAdminRoles(ctx, id, &req)
	if err != nil {
		response.Error(c, http.StatusBadRequest, err.Error())
		return
	}

	response.Success(c, admin.ToDTO())
}

// GetMyPermissions 获取当前管理员的权限
func (h *RBACHandlers) GetMyPermissions(c *gin.Context) {
	adminID, exists := c.Get("admin_id")
	if !exists {
		response.Unauthorized(c, "authentication required")
		return
	}

	ctx := c.Request.Context()
	var adminIDInt64 int64
	switch v := adminID.(type) {
	case uint:
		adminIDInt64 = int64(v)
	case int64:
		adminIDInt64 = v
	case int:
		adminIDInt64 = int64(v)
	default:
		adminIDInt64 = 0
	}
	permissions, err := h.rbacService.GetAdminPermissions(ctx, adminIDInt64)
	if err != nil {
		response.InternalError(c, "failed to get permissions")
		return
	}

	dtos := make([]*model.PermissionDTO, 0, len(permissions))
	for _, perm := range permissions {
		dtos = append(dtos, perm.ToDTO())
	}

	response.Success(c, dtos)
}
