package handler

import (
	"encoding/json"
	"net/http"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"gorm.io/gorm"

	"github.com/hjtpx/hjtpx/internal/model"
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

// CreateAlertChannelRequest 创建告警通道请求参数
type CreateAlertChannelRequest struct {
	Name        string                 `json:"name" binding:"required,min=1,max=255" example:"Slack通知"`
	Type        string                 `json:"type" binding:"required,oneof=slack webhook email dingtalk" example:"slack"`
	Config      map[string]interface{} `json:"config" binding:"required"`
	Description string                 `json:"description" binding:"max=1000" example:"用于发送告警通知到Slack频道"`
	IsEnabled   bool                   `json:"is_enabled" example:"true"`
}

// UpdateAlertChannelRequest 更新告警通道请求参数
type UpdateAlertChannelRequest struct {
	Name        *string                `json:"name" binding:"omitempty,min=1,max=255"`
	Type        *string                `json:"type" binding:"omitempty,oneof=slack webhook email dingtalk"`
	Config      map[string]interface{} `json:"config"`
	Description *string                `json:"description" binding:"omitempty,max=1000"`
	IsEnabled   *bool                  `json:"is_enabled"`
}

// CreateAlertRuleRequest 创建告警规则请求参数
type CreateAlertRuleRequest struct {
	Name              string `json:"name" binding:"required,min=1,max=255" example:"高频验证失败告警"`
	EventType         string `json:"event_type" binding:"required" example:"verification_failed"`
	Condition         string `json:"condition" example:"risk_score > 80"`
	Severity          string `json:"severity" binding:"required,oneof=info warning error critical" example:"warning"`
	ChannelIDs        []uint `json:"channel_ids" binding:"required"`
	IsEnabled         bool   `json:"is_enabled" example:"true"`
	AggregationWindow int    `json:"aggregation_window" binding:"min=1" example:"300"`
	Threshold         int    `json:"threshold" binding:"min=1" example:"5"`
	Description       string `json:"description" binding:"max=1000" example:"当5分钟内验证失败次数超过5次时触发告警"`
}

// UpdateAlertRuleRequest 更新告警规则请求参数
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

// ResolveAlertRequest 解决告警请求参数
type ResolveAlertRequest struct {
	Note string `json:"note" binding:"max=1000" example:"已处理，问题已修复"`
}

// SendTestAlertRequest 发送测试告警请求参数
type SendTestAlertRequest struct {
	EventType string                 `json:"event_type" binding:"required" example:"test_event"`
	Message   string                 `json:"message" binding:"required" example:"这是一条测试告警消息"`
	Context   map[string]interface{} `json:"context"`
}

// ListAlertsQuery 查询告警列表参数
type ListAlertsQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=20"`
	Status   string `form:"status"`
	Severity string `form:"severity"`
}

// CreateAlertChannel 创建告警通道
// @Summary 创建告警通道
// @Description 创建新的告警通知通道，支持slack、webhook、email、dingtalk等类型
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param body body CreateAlertChannelRequest true "创建告警通道请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/channels [post]
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

// UpdateAlertChannel 更新告警通道
// @Summary 更新告警通道
// @Description 更新指定的告警通知通道信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警通道ID"
// @Param body body UpdateAlertChannelRequest true "更新告警通道请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "通道不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/channels/{id} [put]
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

// DeleteAlertChannel 删除告警通道
// @Summary 删除告警通道
// @Description 删除指定的告警通知通道
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警通道ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/channels/{id} [delete]
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

// ListAlertChannels 获取告警通道列表
// @Summary 获取告警通道列表
// @Description 获取所有告警通知通道
// @Tags 告警管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/channels [get]
func ListAlertChannels(c *gin.Context) {
	channels, err := alertServiceInstance.ListChannels()
	if err != nil {
		response.InternalServerError(c, "failed to list channels: "+err.Error())
		return
	}
	response.Success(c, channels)
}

// GetAlertChannel 获取告警通道详情
// @Summary 获取告警通道详情
// @Description 获取指定告警通知通道的详细信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警通道ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "通道不存在"
// @Router /api/v1/admin/alerts/channels/{id} [get]
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

// CreateAlertRule 创建告警规则
// @Summary 创建告警规则
// @Description 创建新的告警规则，定义告警触发条件和通知方式
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param body body CreateAlertRuleRequest true "创建告警规则请求"
// @Success 200 {object} response.Response "创建成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/rules [post]
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

// UpdateAlertRule 更新告警规则
// @Summary 更新告警规则
// @Description 更新指定的告警规则信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警规则ID"
// @Param body body UpdateAlertRuleRequest true "更新告警规则请求"
// @Success 200 {object} response.Response "更新成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "规则不存在"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/rules/{id} [put]
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

// DeleteAlertRule 删除告警规则
// @Summary 删除告警规则
// @Description 删除指定的告警规则
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警规则ID"
// @Success 200 {object} response.Response "删除成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/rules/{id} [delete]
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

// ListAlertRules 获取告警规则列表
// @Summary 获取告警规则列表
// @Description 获取所有告警规则
// @Tags 告警管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/rules [get]
func ListAlertRules(c *gin.Context) {
	rules, err := alertServiceInstance.ListRules()
	if err != nil {
		response.InternalServerError(c, "failed to list rules: "+err.Error())
		return
	}
	response.Success(c, rules)
}

// GetAlertRule 获取告警规则详情
// @Summary 获取告警规则详情
// @Description 获取指定告警规则的详细信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警规则ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "规则不存在"
// @Router /api/v1/admin/alerts/rules/{id} [get]
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

// ListAlerts 获取告警列表
// @Summary 获取告警列表
// @Description 分页获取告警记录列表
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Param status query string false "状态过滤"
// @Param severity query string false "严重等级过滤"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts [get]
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

// GetAlert 获取告警详情
// @Summary 获取告警详情
// @Description 获取指定告警记录的详细信息
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 404 {object} response.Response "告警不存在"
// @Router /api/v1/admin/alerts/{id} [get]
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

// ResolveAlert 解决告警
// @Summary 解决告警
// @Description 将指定告警标记为已解决状态
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警ID"
// @Param body body ResolveAlertRequest true "解决告警请求"
// @Success 200 {object} response.Response "解决成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/{id}/resolve [post]
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

// GetAlertHistory 获取告警历史
// @Summary 获取告警历史
// @Description 获取指定告警的处理历史记录
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param id path uint true "告警ID"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/{id}/history [get]
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

// SendTestAlert 发送测试告警
// @Summary 发送测试告警
// @Description 发送测试告警消息以验证告警通道配置
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param body body SendTestAlertRequest true "发送测试告警请求"
// @Success 200 {object} response.Response "发送成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/test [post]
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

// AlertWebSocketHandler 告警WebSocket连接处理
func AlertWebSocketHandler(c *gin.Context) {
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to upgrade WebSocket connection")
		return
	}

	client := &AlertClient{
		conn:    conn,
		send:    make(chan []byte, 256),
		groups:  make(map[string]bool),
		filters: make(map[string][]string),
	}

	alertClientsMu.Lock()
	alertClients[client] = true
	alertClientsMu.Unlock()

	go client.writePump()
	go client.readPump()
}

// AlertClient 告警WebSocket客户端
type AlertClient struct {
	conn    *websocket.Conn
	send    chan []byte
	groups  map[string]bool
	filters map[string][]string
}

var (
	alertClients   = make(map[*AlertClient]bool)
	alertClientsMu sync.RWMutex
)

func (c *AlertClient) readPump() {
	defer func() {
		alertClientsMu.Lock()
		delete(alertClients, c)
		alertClientsMu.Unlock()
		c.conn.Close()
		close(c.send)
	}()

	c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
	c.conn.SetPongHandler(func(string) error {
		c.conn.SetReadDeadline(time.Now().Add(60 * time.Second))
		return nil
	})

	for {
		_, message, err := c.conn.ReadMessage()
		if err != nil {
			if websocket.IsUnexpectedCloseError(err, websocket.CloseGoingAway, websocket.CloseAbnormalClosure) {
			}
			break
		}

		var msg map[string]interface{}
		if err := json.Unmarshal(message, &msg); err != nil {
			continue
		}

		switch msg["type"] {
		case "subscribe":
			c.handleSubscribe(msg)
		case "filter":
			c.handleFilter(msg)
		case "ping":
			c.handlePing()
		}
	}
}

func (c *AlertClient) writePump() {
	ticker := time.NewTicker(30 * time.Second)
	defer func() {
		ticker.Stop()
		c.conn.Close()
	}()

	for {
		select {
		case message, ok := <-c.send:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if !ok {
				c.conn.WriteMessage(websocket.CloseMessage, []byte{})
				return
			}

			w, err := c.conn.NextWriter(websocket.TextMessage)
			if err != nil {
				return
			}
			w.Write(message)

			if err := w.Close(); err != nil {
				return
			}

		case <-ticker.C:
			c.conn.SetWriteDeadline(time.Now().Add(10 * time.Second))
			if err := c.conn.WriteMessage(websocket.PingMessage, nil); err != nil {
				return
			}
		}
	}
}

func (c *AlertClient) handleSubscribe(msg map[string]interface{}) {
	if groups, ok := msg["groups"].([]interface{}); ok {
		for _, g := range groups {
			if group, ok := g.(string); ok {
				c.groups[group] = true
			}
		}
	}

	responseMsg := map[string]interface{}{
		"type":    "subscribed",
		"groups":  c.groups,
		"filters": c.filters,
	}
	data, _ := json.Marshal(responseMsg)
	c.send <- data
}

func (c *AlertClient) handleFilter(msg map[string]interface{}) {
	if filters, ok := msg["filters"].(map[string]interface{}); ok {
		for key, value := range filters {
			if values, ok := value.([]interface{}); ok {
				strValues := make([]string, 0)
				for _, v := range values {
					if s, ok := v.(string); ok {
						strValues = append(strValues, s)
					}
				}
				c.filters[key] = strValues
			}
		}
	}

	responseMsg := map[string]interface{}{
		"type":    "filters_updated",
		"filters": c.filters,
	}
	data, _ := json.Marshal(responseMsg)
	c.send <- data
}

func (c *AlertClient) handlePing() {
	responseMsg := map[string]interface{}{
		"type":      "pong",
		"timestamp": time.Now().Unix(),
	}
	data, _ := json.Marshal(responseMsg)
	c.send <- data
}

// BroadcastAlert 通过WebSocket广播告警
func BroadcastAlert(alert *models.AlertRecord) {
	alertClientsMu.RLock()
	defer alertClientsMu.RUnlock()

	for client := range alertClients {
		if !client.matchesFilter(alert) {
			continue
		}

		alertMsg := map[string]interface{}{
			"type":      "alert",
			"id":        alert.ID,
			"rule_id":   alert.RuleID,
			"rule_name": alert.RuleName,
			"event_type": alert.EventType,
			"severity":   alert.Severity,
			"message":    alert.Message,
			"context":    alert.Context,
			"status":     alert.Status,
			"timestamp":  alert.CreatedAt.Unix(),
			"count":      alert.Count,
		}

		data, err := json.Marshal(alertMsg)
		if err != nil {
			continue
		}

		select {
		case client.send <- data:
		default:
			alertClientsMu.Lock()
			delete(alertClients, client)
			alertClientsMu.Unlock()
			close(client.send)
		}
	}
}

func (c *AlertClient) matchesFilter(alert *models.AlertRecord) bool {
	if len(c.filters) == 0 {
		return true
	}

	if severities, ok := c.filters["severity"]; ok && len(severities) > 0 {
		found := false
		for _, s := range severities {
			if s == alert.Severity {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	if eventTypes, ok := c.filters["event_type"]; ok && len(eventTypes) > 0 {
		found := false
		for _, t := range eventTypes {
			if t == alert.EventType {
				found = true
				break
			}
		}
		if !found {
			return false
		}
	}

	return true
}

// GetAlertsBySeverity 按风险等级获取告警
// @Summary 按风险等级获取告警
// @Description 根据严重等级过滤告警记录
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param severity query string true "严重等级(info/warning/error/critical)"
// @Param page query int false "页码，默认1"
// @Param page_size query int false "每页数量，默认20"
// @Success 200 {object} response.Response "获取成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/severity/{severity} [get]
func GetAlertsBySeverity(c *gin.Context) {
	severity := c.Param("severity")
	
	var query ListAlertsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query: "+err.Error())
		return
	}

	if query.Page == 0 {
		query.Page = 1
	}
	if query.PageSize == 0 {
		query.PageSize = 20
	}

	alerts, total, err := alertServiceInstance.ListAlertsBySeverity(severity, query.Page, query.PageSize)
	if err != nil {
		response.InternalServerError(c, "failed to list alerts: "+err.Error())
		return
	}

	response.Success(c, gin.H{
		"items":     alerts,
		"total":     total,
		"page":      query.Page,
		"page_size": query.PageSize,
		"severity":  severity,
	})
}

// GetAlertStatistics 获取告警统计
// @Summary 获取告警统计
// @Description 获取告警统计信息，包括各等级告警数量
// @Tags 告警管理
// @Accept json
// @Produce json
// @Success 200 {object} response.Response "获取成功"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/statistics [get]
func GetAlertStatistics(c *gin.Context) {
	stats, err := alertServiceInstance.GetAlertStatistics()
	if err != nil {
		response.InternalServerError(c, "failed to get alert statistics: "+err.Error())
		return
	}
	response.Success(c, stats)
}

// TriggerRiskAlert 触发风险告警
// @Summary 触发风险告警
// @Description 根据风险评估结果触发告警
// @Tags 告警管理
// @Accept json
// @Produce json
// @Param body body RiskAlertRequest true "风险告警请求"
// @Success 200 {object} response.Response "触发成功"
// @Failure 400 {object} response.Response "请求参数错误"
// @Failure 500 {object} response.Response "服务器内部错误"
// @Router /api/v1/admin/alerts/risk [post]
func TriggerRiskAlert(c *gin.Context) {
	var req RiskAlertRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request: "+err.Error())
		return
	}

	riskLevel := model.DetermineRiskLevel(req.RiskScore)
	
	event := service.AlertEvent{
		EventType: req.EventType,
		Message:   req.Message,
		Context: map[string]interface{}{
			"risk_score":   req.RiskScore,
			"risk_level":   riskLevel,
			"session_id":   req.SessionID,
			"ip_address":   req.IPAddress,
			"user_id":      req.UserID,
			"risk_factors": req.RiskFactors,
		},
		Timestamp: time.Now(),
	}

	if err := alertServiceInstance.ProcessEvent(event); err != nil {
		response.InternalServerError(c, "failed to trigger risk alert: "+err.Error())
		return
	}

	TriggerRiskEvent(req.EventType, riskLevel, req.RiskScore, req.Message, req.RiskFactors, map[string]string{
		"session_id": req.SessionID,
		"ip_address": req.IPAddress,
		"user_id":    req.UserID,
	})

	response.Success(c, gin.H{
		"message":     "risk alert triggered",
		"risk_level":  riskLevel,
		"risk_score":  req.RiskScore,
	})
}

// RiskAlertRequest 风险告警请求
type RiskAlertRequest struct {
	EventType   string   `json:"event_type" binding:"required"`
	Message     string   `json:"message" binding:"required"`
	RiskScore   float64  `json:"risk_score" binding:"required"`
	SessionID   string   `json:"session_id"`
	IPAddress   string   `json:"ip_address"`
	UserID      string   `json:"user_id"`
	RiskFactors []string `json:"risk_factors"`
}

// AlertStatistics 告警统计
type AlertStatistics struct {
	TotalCount     int64                  `json:"total_count"`
	ActiveCount    int64                  `json:"active_count"`
	ResolvedCount  int64                  `json:"resolved_count"`
	SeverityStats  map[string]int64       `json:"severity_stats"`
	EventTypeStats map[string]int64       `json:"event_type_stats"`
	TrendData      []HourlyAlertStat      `json:"trend_data"`
}

type HourlyAlertStat struct {
	Hour      string `json:"hour"`
	Count     int64  `json:"count"`
	Severity  string `json:"severity,omitempty"`
}
