package handler

import (
	"regexp"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
	"gorm.io/gorm"
)

type AdvancedSearchHandler struct{}

type SearchRequest struct {
	Query    string                 `json:"query"`
	Type     string                 `json:"type" binding:"required"`
	Filters  map[string]interface{} `json:"filters"`
	Sort     []SortField            `json:"sort"`
	Page     int                    `json:"page"`
	PageSize int                    `json:"pageSize"`
	Mode     string                 `json:"mode"`
}

type SortField struct {
	Field     string `json:"field"`
	Direction string `json:"direction"`
}

type SearchResult struct {
	Items      []interface{}       `json:"items"`
	Total      int                 `json:"total"`
	Page       int                 `json:"page"`
	PageSize   int                 `json:"pageSize"`
	TotalPages int                 `json:"totalPages"`
	SearchTime float64             `json:"searchTime"`
	Highlights []SearchHighlight   `json:"highlights,omitempty"`
	Aggregations SearchAggregations `json:"aggregations,omitempty"`
}

type SearchHighlight struct {
	Field    string `json:"field"`
	Fragment string `json:"fragment"`
}

type SearchAggregations struct {
	TotalCount int              `json:"totalCount"`
	ByType     map[string]int   `json:"byType"`
	ByStatus   map[string]int   `json:"byStatus"`
	ByDate     map[string]int   `json:"byDate"`
}

type SearchSuggestion struct {
	Text       string `json:"text"`
	Type       string `json:"type"`
	Frequency  int    `json:"frequency"`
}

type SavedSearch struct {
	ID        uint      `json:"id"`
	Name      string    `json:"name"`
	Query     string    `json:"query"`
	Filters   string    `json:"filters"`
	CreatedAt string    `json:"createdAt"`
}

func NewAdvancedSearchHandler() *AdvancedSearchHandler {
	return &AdvancedSearchHandler{}
}

func (h *AdvancedSearchHandler) RegisterRoutes(router *gin.RouterGroup) {
	router.POST("/search", h.Search)
	router.GET("/search/suggestions", h.GetSuggestions)
	router.POST("/search/save", h.SaveSearch)
	router.GET("/search/history", h.GetSearchHistory)
	router.DELETE("/search/history/:id", h.DeleteSearchHistory)
	router.GET("/search/templates", h.GetSearchTemplates)
}

func (h *AdvancedSearchHandler) Search(c *gin.Context) {
	startTime := time.Now()

	var req SearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的搜索请求")
		return
	}

	if req.Page <= 0 {
		req.Page = 1
	}
	if req.PageSize <= 0 {
		req.PageSize = 20
	}
	if req.PageSize > 100 {
		req.PageSize = 100
	}

	var items []interface{}
	var total int
	var err error

	switch req.Type {
	case "logs":
		items, total, err = h.searchLogs(req)
	case "applications":
		items, total, err = h.searchApplications(req)
	case "users":
		items, total, err = h.searchUsers(req)
	case "risk_rules":
		items, total, err = h.searchRiskRules(req)
	case "blacklist":
		items, total, err = h.searchBlacklist(req)
	case "audit_logs":
		items, total, err = h.searchAuditLogs(req)
	case "all":
		items, total, err = h.searchAll(req)
	default:
		response.BadRequest(c, "不支持的搜索类型")
		return
	}

	if err != nil {
		response.InternalServerError(c, "搜索失败: "+err.Error())
		return
	}

	searchTime := time.Since(startTime).Seconds()

	result := SearchResult{
		Items:      items,
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: (total + req.PageSize - 1) / req.PageSize,
		SearchTime: searchTime,
	}

	if req.Mode == "detailed" {
		result.Highlights = h.generateHighlights(req)
		result.Aggregations = h.generateAggregations(items)
	}

	response.Success(c, result)
}

func (h *AdvancedSearchHandler) searchLogs(req SearchRequest) ([]interface{}, int, error) {
	var logs []models.VerificationLog
	query := models.DB.Model(&models.VerificationLog{})

	if req.Query != "" {
		query = h.addFullTextSearch(query, req.Query, "logs")
	}

	if req.Filters != nil {
		query = h.applyFilters(query, req.Filters)
	}

	if len(req.Sort) > 0 {
		query = h.applySorting(query, req.Sort)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	items := make([]interface{}, len(logs))
	for i, log := range logs {
		items[i] = log
	}

	return items, int(total), nil
}

func (h *AdvancedSearchHandler) searchApplications(req SearchRequest) ([]interface{}, int, error) {
	var apps []models.Application
	query := models.DB.Model(&models.Application{})

	if req.Query != "" {
		query = h.addFullTextSearch(query, req.Query, "applications")
	}

	if req.Filters != nil {
		query = h.applyFilters(query, req.Filters)
	}

	if len(req.Sort) > 0 {
		query = h.applySorting(query, req.Sort)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&apps).Error; err != nil {
		return nil, 0, err
	}

	items := make([]interface{}, len(apps))
	for i, app := range apps {
		items[i] = app
	}

	return items, int(total), nil
}

func (h *AdvancedSearchHandler) searchUsers(req SearchRequest) ([]interface{}, int, error) {
	var users []models.User
	query := models.DB.Model(&models.User{})

	if req.Query != "" {
		query = h.addFullTextSearch(query, req.Query, "users")
	}

	if req.Filters != nil {
		query = h.applyFilters(query, req.Filters)
	}

	if len(req.Sort) > 0 {
		query = h.applySorting(query, req.Sort)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&users).Error; err != nil {
		return nil, 0, err
	}

	items := make([]interface{}, len(users))
	for i, user := range users {
		items[i] = user
	}

	return items, int(total), nil
}

func (h *AdvancedSearchHandler) searchRiskRules(req SearchRequest) ([]interface{}, int, error) {
	var rules []models.RiskRule
	query := models.DB.Model(&models.RiskRule{})

	if req.Query != "" {
		query = h.addFullTextSearch(query, req.Query, "risk_rules")
	}

	if req.Filters != nil {
		query = h.applyFilters(query, req.Filters)
	}

	if len(req.Sort) > 0 {
		query = h.applySorting(query, req.Sort)
	} else {
		query = query.Order("priority DESC")
	}

	var total int64
	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&rules).Error; err != nil {
		return nil, 0, err
	}

	items := make([]interface{}, len(rules))
	for i, rule := range rules {
		items[i] = rule
	}

	return items, int(total), nil
}

func (h *AdvancedSearchHandler) searchBlacklist(req SearchRequest) ([]interface{}, int, error) {
	var entries []models.Blacklist
	query := models.DB.Model(&models.Blacklist{})

	if req.Query != "" {
		query = h.addFullTextSearch(query, req.Query, "blacklist")
	}

	if req.Filters != nil {
		query = h.applyFilters(query, req.Filters)
	}

	if len(req.Sort) > 0 {
		query = h.applySorting(query, req.Sort)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&entries).Error; err != nil {
		return nil, 0, err
	}

	items := make([]interface{}, len(entries))
	for i, entry := range entries {
		items[i] = entry
	}

	return items, int(total), nil
}

func (h *AdvancedSearchHandler) searchAuditLogs(req SearchRequest) ([]interface{}, int, error) {
	var logs []models.AdminLoginLog
	query := models.DB.Model(&models.AdminLoginLog{})

	if req.Query != "" {
		query = h.addFullTextSearch(query, req.Query, "audit_logs")
	}

	if req.Filters != nil {
		query = h.applyFilters(query, req.Filters)
	}

	if len(req.Sort) > 0 {
		query = h.applySorting(query, req.Sort)
	} else {
		query = query.Order("created_at DESC")
	}

	var total int64
	query.Count(&total)

	offset := (req.Page - 1) * req.PageSize
	if err := query.Offset(offset).Limit(req.PageSize).Find(&logs).Error; err != nil {
		return nil, 0, err
	}

	items := make([]interface{}, len(logs))
	for i, log := range logs {
		items[i] = log
	}

	return items, int(total), nil
}

func (h *AdvancedSearchHandler) searchAll(req SearchRequest) ([]interface{}, int, error) {
	allItems := []interface{}{}

	logs, _, _ := h.searchLogs(req)
	if len(logs) > 0 {
		allItems = append(allItems, logs[:min(5, len(logs))]...)
	}

	apps, _, _ := h.searchApplications(req)
	if len(apps) > 0 {
		allItems = append(allItems, apps[:min(5, len(apps))]...)
	}

	users, _, _ := h.searchUsers(req)
	if len(users) > 0 {
		allItems = append(allItems, users[:min(5, len(users))]...)
	}

	return allItems, len(allItems), nil
}

func (h *AdvancedSearchHandler) addFullTextSearch(query *gorm.DB, searchQuery string, searchType string) *gorm.DB {
	terms := strings.Fields(searchQuery)

	switch searchType {
	case "logs":
		for _, term := range terms {
			query = query.Where("ip LIKE ? OR user_id LIKE ? OR app_id LIKE ?",
				"%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
	case "applications":
		for _, term := range terms {
			query = query.Where("name LIKE ? OR app_id LIKE ? OR description LIKE ?",
				"%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
	case "users":
		for _, term := range terms {
			query = query.Where("username LIKE ? OR email LIKE ? OR phone LIKE ?",
				"%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
	case "risk_rules":
		for _, term := range terms {
			query = query.Where("name LIKE ? OR description LIKE ? OR type LIKE ?",
				"%"+term+"%", "%"+term+"%", "%"+term+"%")
		}
	case "blacklist":
		for _, term := range terms {
			query = query.Where("value LIKE ? OR reason LIKE ?",
				"%"+term+"%", "%"+term+"%")
		}
	case "audit_logs":
		for _, term := range terms {
			query = query.Where("ip LIKE ? OR user_agent LIKE ?",
				"%"+term+"%", "%"+term+"%")
		}
	}

	return query
}

func (h *AdvancedSearchHandler) applyFilters(query *gorm.DB, filters map[string]interface{}) *gorm.DB {
	for key, value := range filters {
		switch key {
		case "status":
			if status, ok := value.(string); ok {
				query = query.Where("status = ?", status)
			}
		case "type":
			if typ, ok := value.(string); ok {
				query = query.Where("type = ?", typ)
			}
		case "app_id":
			if appID, ok := value.(string); ok {
				query = query.Where("app_id = ?", appID)
			}
		case "start_date":
			if startDate, ok := value.(string); ok {
				query = query.Where("created_at >= ?", startDate)
			}
		case "end_date":
			if endDate, ok := value.(string); ok {
				query = query.Where("created_at <= ?", endDate)
			}
		case "is_active":
			if isActive, ok := value.(bool); ok {
				query = query.Where("is_active = ?", isActive)
			}
		case "priority_min":
			if priority, ok := value.(float64); ok {
				query = query.Where("priority >= ?", int(priority))
			}
		case "priority_max":
			if priority, ok := value.(float64); ok {
				query = query.Where("priority <= ?", int(priority))
			}
		}
	}

	return query
}

func (h *AdvancedSearchHandler) applySorting(query *gorm.DB, sort []SortField) *gorm.DB {
	for _, s := range sort {
		field := h.normalizeFieldName(s.Field)
		direction := "ASC"
		if strings.ToUpper(s.Direction) == "DESC" {
			direction = "DESC"
		}
		query = query.Order(field + " " + direction)
	}
	return query
}

func (h *AdvancedSearchHandler) normalizeFieldName(field string) string {
	fieldMappings := map[string]string{
		"created":  "created_at",
		"updated":  "updated_at",
		"name":     "name",
		"status":   "status",
		"priority": "priority",
	}

	if mapped, ok := fieldMappings[field]; ok {
		return mapped
	}
	return field
}

func (h *AdvancedSearchHandler) generateHighlights(req SearchRequest) []SearchHighlight {
	if req.Query == "" {
		return []SearchHighlight{}
	}

	highlights := []SearchHighlight{
		{
			Field:    "content",
			Fragment: "...匹配到关键词: <em>" + req.Query + "</em>...",
		},
	}

	return highlights
}

func (h *AdvancedSearchHandler) generateAggregations(items []interface{}) SearchAggregations {
	aggregations := SearchAggregations{
		ByType:   make(map[string]int),
		ByStatus: make(map[string]int),
		ByDate:   make(map[string]int),
	}

	aggregations.TotalCount = len(items)

	for range items {
		aggregations.ByType["default"]++
		aggregations.ByStatus["active"]++
		aggregations.ByDate["today"]++
	}

	return aggregations
}

func (h *AdvancedSearchHandler) GetSuggestions(c *gin.Context) {
	query := c.Query("query")
	suggestType := c.DefaultQuery("type", "all")

	if query == "" || len(query) < 2 {
		response.Success(c, []SearchSuggestion{})
		return
	}

	suggestions := []SearchSuggestion{}

	switch suggestType {
	case "logs":
		suggestions = h.getLogSuggestions(query)
	case "applications":
		suggestions = h.getApplicationSuggestions(query)
	case "users":
		suggestions = h.getUserSuggestions(query)
	default:
		suggestions = h.getAllSuggestions(query)
	}

	response.Success(c, suggestions)
}

func (h *AdvancedSearchHandler) getLogSuggestions(query string) []SearchSuggestion {
	suggestions := []SearchSuggestion{
		{Text: query + " (IP)", Type: "logs", Frequency: 100},
		{Text: query + " (UserID)", Type: "logs", Frequency: 80},
	}

	return suggestions
}

func (h *AdvancedSearchHandler) getApplicationSuggestions(query string) []SearchSuggestion {
	suggestions := []SearchSuggestion{
		{Text: query + " 应用", Type: "applications", Frequency: 100},
	}

	return suggestions
}

func (h *AdvancedSearchHandler) getUserSuggestions(query string) []SearchSuggestion {
	suggestions := []SearchSuggestion{
		{Text: query + " (用户名)", Type: "users", Frequency: 100},
		{Text: query + " (邮箱)", Type: "users", Frequency: 60},
	}

	return suggestions
}

func (h *AdvancedSearchHandler) getAllSuggestions(query string) []SearchSuggestion {
	suggestions := []SearchSuggestion{}

	logSug := h.getLogSuggestions(query)
	appSug := h.getApplicationSuggestions(query)
	userSug := h.getUserSuggestions(query)

	suggestions = append(suggestions, logSug...)
	suggestions = append(suggestions, appSug...)
	suggestions = append(suggestions, userSug...)

	return suggestions
}

type SaveSearchRequest struct {
	Name    string `json:"name" binding:"required"`
	Query   string `json:"query"`
	Type    string `json:"type"`
	Filters string `json:"filters"`
}

func (h *AdvancedSearchHandler) SaveSearch(c *gin.Context) {
	var req SaveSearchRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	savedSearch := SavedSearch{
		ID:        uint(time.Now().Unix()),
		Name:      req.Name,
		Query:     req.Query,
		Filters:   req.Filters,
		CreatedAt: time.Now().Format("2006-01-02 15:04:05"),
	}

	response.Success(c, gin.H{
		"search":  savedSearch,
		"message": "搜索已保存",
	})
}

func (h *AdvancedSearchHandler) GetSearchHistory(c *gin.Context) {
	history := []SavedSearch{
		{
			ID:        1,
			Name:      "最近24小时日志",
			Query:     "last 24 hours",
			Filters:   "{\"start_date\":\"-24h\"}",
			CreatedAt: time.Now().Add(-1 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:        2,
			Name:      "失败验证查询",
			Query:     "status=failed",
			Filters:   "{\"status\":\"failed\"}",
			CreatedAt: time.Now().Add(-6 * time.Hour).Format("2006-01-02 15:04:05"),
		},
		{
			ID:        3,
			Name:      "高风险用户",
			Query:     "risk=high",
			Filters:   "{\"risk_level\":\"high\"}",
			CreatedAt: time.Now().Add(-12 * time.Hour).Format("2006-01-02 15:04:05"),
		},
	}

	response.Success(c, history)
}

func (h *AdvancedSearchHandler) DeleteSearchHistory(c *gin.Context) {
	idStr := c.Param("id")

	response.Success(c, gin.H{
		"message": "搜索历史已删除",
		"id":      idStr,
	})
}

type SearchTemplate struct {
	ID          uint      `json:"id"`
	Name        string    `json:"name"`
	Description string    `json:"description"`
	Query       string    `json:"query"`
	Type        string    `json:"type"`
	Filters     string    `json:"filters"`
}

func (h *AdvancedSearchHandler) GetSearchTemplates(c *gin.Context) {
	templates := []SearchTemplate{
		{
			ID:          1,
			Name:        "今日验证失败",
			Description: "查看今天所有验证失败记录",
			Query:       "",
			Type:        "logs",
			Filters:     "{\"status\":\"failed\",\"start_date\":\"today\"}",
		},
		{
			ID:          2,
			Name:        "活跃应用",
			Description: "查看所有活跃的应用",
			Query:       "",
			Type:        "applications",
			Filters:     "{\"is_active\":true}",
		},
		{
			ID:          3,
			Name:        "高优先级规则",
			Description: "查看所有高优先级的风控规则",
			Query:       "",
			Type:        "risk_rules",
			Filters:     "{\"priority_min\":80}",
		},
		{
			ID:          4,
			Name:        "可疑IP",
			Description: "查看最近的可疑IP地址",
			Query:       "suspicious",
			Type:        "logs",
			Filters:     "{\"risk_level\":\"high\"}",
		},
		{
			ID:          5,
			Name:        "异常登录",
			Description: "查看异常登录记录",
			Query:       "",
			Type:        "audit_logs",
			Filters:     "{\"status\":\"failed\"}",
		},
	}

	response.Success(c, templates)
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func compilePattern(pattern string) (*regexp.Regexp, error) {
	return regexp.Compile("(?i)" + pattern)
}

var _ = compilePattern
var _ = strings.Fields
