package handler

import (
	"encoding/json"
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"gorm.io/gorm"

	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
)

var alertServiceInstance *service.AlertService

func InitAlertService(db *gorm.DB) {
	alertServiceInstance = service.NewAlertService(db)
	alertServiceInstance.LoadRules()
	alertServiceInstance.LoadChannels()
}

type CreateAlertChannelRequest struct {
	Name        string                 `json:"name" binding:"required,min=1,max=255"`
	Type        string                 `json:"type" binding:"required,oneof=slack webhook"`
	Config      map[string]interface{} `json:"config" binding:"required"`
	Description string                 `json:"description" binding:"max=1000"`
	IsEnabled   bool                   `json:"is_enabled"`
}

type UpdateAlertChannelRequest struct {
	Name        *string                `json:"name" binding:"omitempty,min=1,max=255"`
	Type        *string                `json:"type" binding:"omitempty,oneof=slack webhook"`
	Config      map[string]interface{} `json:"config"`
	Description *string                `json:"description" binding:"omitempty,max=1000"`
	IsEnabled   *bool                  `json:"is_enabled"`
}

type CreateAlertRuleRequest struct {
	Name              string `json:"name" binding:"required,min=1,max=255"`
	EventType         string `json:"event_type" binding:"required"`
	Condition         string `json:"condition"`
	Severity          string `json:"severity" binding:"required,oneof=info warning error critical"`
	ChannelIDs        []uint `json:"channel_ids" binding:"required"`
	IsEnabled         bool   `json:"is_enabled"`
	AggregationWindow int    `json:"aggregation_window" binding:"min=1"`
	Threshold         int    `json:"threshold" binding:"min=1"`
	Description       string `json:"description" binding:"max=1000"`
}

type UpdateAlertRuleRequest struct {
	Name              *string `json:"name" binding:"omitempty,min=1,max=255"`
	EventType         *string `json:"event_type"`
	Condition         *string `json:"condition"`
	Severity          *string `json:"severity" binding:"omitempty,oneof=info warning error critical"`
	ChannelIDs        []uint  `json:"channel_ids"`
	IsEnabled         *bool   `json:"is_enabled"`
	AggregationWindow *int    `json:"aggregation_window" binding:"omitempty,min=1"`
	Threshold         *int    `json:"threshold" binding:"omitempty,min=1"`
	Description       *string `json:"description" binding:"omitempty,max=1000"`
}

type ResolveAlertRequest struct {
	Note string `json:"note" binding:"max=1000"`
}

type SendTestAlertRequest struct {
	EventType string                 `json:"event_type" binding:"required"`
	Message   string                 `json:"message" binding:"required"`
	Context   map[string]interface{} `json:"context"`
}

type ListAlertsQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Status   string `form:"status"`
	Severity string `form:"severity"`
}

func CreateAlertChannel(c *gin.Context) {
	var req CreateAlertChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	configJSON, err := json.Marshal(req.Config)
	if err != nil {
		response.BadRequest(c, "invalid config: "+err.Error())
		return
	}
	channel := &models.AlertChannel{
		Name:        req.Name,
		Type:        req.Type,
		Config:      string(configJSON),
		Description: req.Description,
		IsEnabled:   req.IsEnabled,
	}
	if err := alertServiceInstance.CreateChannel(channel); err != nil {
		response.InternalServerError(c, "failed to create channel: "+err.Error())
		return
	}
	response.Success(c, channel)
}

func UpdateAlertChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid channel id")
		return
	}
	var req UpdateAlertChannelRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	channel, err := alertServiceInstance.GetChannel(uint(id))
	if err != nil {
		response.NotFound(c, "channel not found")
		return
	}
	if req.Name != nil {
		channel.Name = *req.Name
	}
	if req.Type != nil {
		channel.Type = *req.Type
	}
	if req.Config != nil {
		configJSON, _ := json.Marshal(req.Config)
		channel.Config = string(configJSON)
	}
	if req.Description != nil {
		channel.Description = *req.Description
	}
	if req.IsEnabled != nil {
		channel.IsEnabled = *req.IsEnabled
	}
	if err := alertServiceInstance.UpdateChannel(channel); err != nil {
		response.InternalServerError(c, "failed to update channel: "+err.Error())
		return
	}
	response.Success(c, channel)
}

func DeleteAlertChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid channel id")
		return
	}
	if err := alertServiceInstance.DeleteChannel(uint(id)); err != nil {
		response.InternalServerError(c, "failed to delete channel: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func ListAlertChannels(c *gin.Context) {
	channels, err := alertServiceInstance.ListChannels()
	if err != nil {
		response.InternalServerError(c, "failed to list channels: "+err.Error())
		return
	}
	response.Success(c, channels)
}

func GetAlertChannel(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid channel id")
		return
	}
	channel, err := alertServiceInstance.GetChannel(uint(id))
	if err != nil {
		response.NotFound(c, "channel not found")
		return
	}
	response.Success(c, channel)
}

func CreateAlertRule(c *gin.Context) {
	var req CreateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	channelIDsJSON, _ := json.Marshal(req.ChannelIDs)
	if req.AggregationWindow == 0 {
		req.AggregationWindow = 300
	}
	if req.Threshold == 0 {
		req.Threshold = 1
	}
	rule := &models.AlertRule{
		Name:              req.Name,
		EventType:         req.EventType,
		Condition:         req.Condition,
		Severity:          req.Severity,
		ChannelIDs:        string(channelIDsJSON),
		IsEnabled:         req.IsEnabled,
		AggregationWindow: req.AggregationWindow,
		Threshold:         req.Threshold,
		Description:       req.Description,
	}
	if err := alertServiceInstance.CreateRule(rule); err != nil {
		response.InternalServerError(c, "failed to create rule: "+err.Error())
		return
	}
	response.Success(c, rule)
}

func UpdateAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid rule id")
		return
	}
	var req UpdateAlertRuleRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	rule, err := alertServiceInstance.GetRule(uint(id))
	if err != nil {
		response.NotFound(c, "rule not found")
		return
	}
	if req.Name != nil {
		rule.Name = *req.Name
	}
	if req.EventType != nil {
		rule.EventType = *req.EventType
	}
	if req.Condition != nil {
		rule.Condition = *req.Condition
	}
	if req.Severity != nil {
		rule.Severity = *req.Severity
	}
	if req.ChannelIDs != nil {
		channelIDsJSON, _ := json.Marshal(req.ChannelIDs)
		rule.ChannelIDs = string(channelIDsJSON)
	}
	if req.IsEnabled != nil {
		rule.IsEnabled = *req.IsEnabled
	}
	if req.AggregationWindow != nil {
		rule.AggregationWindow = *req.AggregationWindow
	}
	if req.Threshold != nil {
		rule.Threshold = *req.Threshold
	}
	if req.Description != nil {
		rule.Description = *req.Description
	}
	if err := alertServiceInstance.UpdateRule(rule); err != nil {
		response.InternalServerError(c, "failed to update rule: "+err.Error())
		return
	}
	response.Success(c, rule)
}

func DeleteAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid rule id")
		return
	}
	if err := alertServiceInstance.DeleteRule(uint(id)); err != nil {
		response.InternalServerError(c, "failed to delete rule: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func ListAlertRules(c *gin.Context) {
	rules, err := alertServiceInstance.ListRules()
	if err != nil {
		response.InternalServerError(c, "failed to list rules: "+err.Error())
		return
	}
	response.Success(c, rules)
}

func GetAlertRule(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid rule id")
		return
	}
	rule, err := alertServiceInstance.GetRule(uint(id))
	if err != nil {
		response.NotFound(c, "rule not found")
		return
	}
	response.Success(c, rule)
}

func ListAlerts(c *gin.Context) {
	var query ListAlertsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query: "+err.Error())
		return
	}
	alerts, total, err := alertServiceInstance.ListAlerts(query.Page, query.PageSize)
	if err != nil {
		response.InternalServerError(c, "failed to list alerts: "+err.Error())
		return
	}
	response.Success(c, gin.H{
		"items":     alerts,
		"total":     total,
		"page":      query.Page,
		"page_size": query.PageSize,
	})
}

func GetAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid alert id")
		return
	}
	alert, err := alertServiceInstance.GetAlert(uint(id))
	if err != nil {
		response.NotFound(c, "alert not found")
		return
	}
	response.Success(c, alert)
}

func ResolveAlert(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid alert id")
		return
	}
	var req ResolveAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	performedBy := uint(0)
	if userID, exists := c.Get("user_id"); exists {
		performedBy = userID.(uint)
	}
	if err := alertServiceInstance.ResolveAlert(uint(id), req.Note, performedBy); err != nil {
		response.InternalServerError(c, "failed to resolve alert: "+err.Error())
		return
	}
	response.Success(c, nil)
}

func GetAlertHistory(c *gin.Context) {
	id, err := strconv.ParseUint(c.Param("id"), 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid alert id")
		return
	}
	history, err := alertServiceInstance.GetAlertHistory(uint(id))
	if err != nil {
		response.InternalServerError(c, "failed to get alert history: "+err.Error())
		return
	}
	response.Success(c, history)
}

func SendTestAlert(c *gin.Context) {
	var req SendTestAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}
	event := service.AlertEvent{
		EventType: req.EventType,
		Message:   req.Message,
		Context:   req.Context,
		Timestamp: time.Now(),
	}
	if err := alertServiceInstance.ProcessEvent(event); err != nil {
		response.InternalServerError(c, "failed to send test alert: "+err.Error())
		return
	}
	response.Success(c, gin.H{"message": "test alert processed"})
}
