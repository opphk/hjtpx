package handler

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	"strconv"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type BatchOperationHandler struct {
	batchService *service.BatchOperationService
}

func NewBatchOperationHandler() *BatchOperationHandler {
	return &BatchOperationHandler{
		batchService: service.NewBatchOperationService(),
	}
}

type BlacklistImportRequest struct {
	Type       string `json:"type" binding:"required"`
	Reason     string `json:"reason"`
	Action     string `json:"action"`
	Expiration string `json:"expiration"`
}

type BlacklistBatchDeleteRequest struct {
	IDs []uint `json:"ids" binding:"required,min=1"`
}

type ApplicationBatchConfigRequest struct {
	AppIDs              []uint `json:"app_ids" binding:"required,min=1"`
	CaptchaTypes        []string `json:"captcha_types"`
	MaxVerifyPerMinute  int      `json:"max_verify_per_minute"`
	MaxVerifyPerDay     int      `json:"max_verify_per_day"`
	AllowedIPs          []string `json:"allowed_ips"`
	BlockRefusedRequests bool     `json:"block_refused_requests"`
}

type BatchOperationQuery struct {
	Page      int    `form:"page,default=1"`
	Size      int    `form:"size,default=20"`
	TargetType    string `form:"target_type"`
	OperationType string `form:"operation_type"`
	Status    string `form:"status"`
}

func ListBatchOperations(c *gin.Context) {
	handler := NewBatchOperationHandler()
	var query BatchOperationQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.Size < 1 || query.Size > 100 {
		query.Size = 20
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	filter := &service.ListBatchOperationsFilter{
		Page:          query.Page,
		PageSize:      query.Size,
		TargetType:    query.TargetType,
		OperationType: query.OperationType,
		Status:        query.Status,
		CreatedBy:     createdBy,
	}

	operations, total, err := handler.batchService.ListOperations(filter)
	if err != nil {
		response.InternalServerError(c, "获取批量操作记录失败")
		return
	}

	totalPages := int(total) / query.Size
	if int(total)%query.Size > 0 {
		totalPages++
	}

	response.Success(c, gin.H{
		"list":        operations,
		"total":       total,
		"page":        query.Page,
		"page_size":   query.Size,
		"total_pages": totalPages,
	})
}

func GetBatchOperation(c *gin.Context) {
	handler := NewBatchOperationHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的操作ID")
		return
	}

	operation, err := handler.batchService.GetOperation(uint(id))
	if err != nil {
		if err == service.ErrBatchOperationNotFound {
			response.NotFound(c, "批量操作记录不存在")
			return
		}
		response.InternalServerError(c, "获取批量操作详情失败")
		return
	}

	response.Success(c, operation)
}

func BlacklistBatchImport(c *gin.Context) {
	handler := NewBatchOperationHandler()

	file, _, err := c.Request.FormFile("file")
	if err != nil {
		response.BadRequest(c, "请上传文件")
		return
	}
	defer file.Close()

	var req BlacklistImportRequest
	req.Type = c.PostForm("type")
	if req.Type == "" {
		req.Type = "ip"
	}
	req.Reason = c.PostForm("reason")
	req.Action = c.PostForm("action")
	req.Expiration = c.PostForm("expiration")

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	contentType := c.Request.Header.Get("Content-Type")
	var targets []string

	if strings.Contains(contentType, "application/json") {
		var jsonReq struct {
			Type       string   `json:"type"`
			Reason     string   `json:"reason"`
			Action     string   `json:"action"`
			Expiration string   `json:"expiration"`
			Targets    []string `json:"targets" binding:"required,min=1"`
		}
		if err := c.ShouldBindJSON(&jsonReq); err != nil {
			response.BadRequest(c, "无效的请求参数")
			return
		}
		targets = jsonReq.Targets
		req.Type = jsonReq.Type
		req.Reason = jsonReq.Reason
		req.Action = jsonReq.Action
		req.Expiration = jsonReq.Expiration
	} else {
		reader := csv.NewReader(file)
		for {
			record, err := reader.Read()
			if err == io.EOF {
				break
			}
			if err != nil {
				continue
			}
			for _, field := range record {
				field = strings.TrimSpace(field)
				if field != "" {
					targets = append(targets, field)
				}
			}
		}
	}

	if len(targets) == 0 {
		response.BadRequest(c, "没有找到有效的导入数据")
		return
	}

	if len(targets) > 10000 {
		response.BadRequest(c, "单次导入最多支持10000条记录")
		return
	}

	input := &service.BatchOperationInput{
		OperationType: "blacklist_import",
		TargetType:    "blacklist",
		TargetIDs:     targets,
		CreatedBy:     createdBy,
	}

	operation, err := handler.batchService.CreateOperation(input)
	if err != nil {
		response.InternalServerError(c, "创建批量操作失败")
		return
	}

	go func() {
		ctx := context.Background()
		handler.batchService.BlacklistBatchImport(ctx, operation.ID, targets, req.Type, req.Reason, req.Action, req.Expiration, createdBy)
	}()

	response.Success(c, gin.H{
		"operation_id": operation.ID,
		"total":       len(targets),
		"message":     "批量导入任务已创建，正在后台处理",
	})
}

func BlacklistBatchDelete(c *gin.Context) {
	handler := NewBatchOperationHandler()
	var req BlacklistBatchDeleteRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.IDs) == 0 {
		response.BadRequest(c, "请选择要删除的记录")
		return
	}

	if len(req.IDs) > 1000 {
		response.BadRequest(c, "单次最多删除1000条记录")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	targetIDs := make([]string, len(req.IDs))
	for i, id := range req.IDs {
		targetIDs[i] = fmt.Sprintf("%d", id)
	}

	input := &service.BatchOperationInput{
		OperationType: "blacklist_delete",
		TargetType:    "blacklist",
		TargetIDs:     targetIDs,
		CreatedBy:     createdBy,
	}

	operation, err := handler.batchService.CreateOperation(input)
	if err != nil {
		response.InternalServerError(c, "创建批量操作失败")
		return
	}

	go func() {
		ctx := context.Background()
		handler.batchService.BlacklistBatchDelete(ctx, operation.ID, req.IDs)
	}()

	response.Success(c, gin.H{
		"operation_id": operation.ID,
		"total":       len(req.IDs),
		"message":     "批量删除任务已创建，正在后台处理",
	})
}

func ApplicationBatchConfig(c *gin.Context) {
	handler := NewBatchOperationHandler()
	var req ApplicationBatchConfigRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.AppIDs) == 0 {
		response.BadRequest(c, "请选择要配置的应用")
		return
	}

	if len(req.AppIDs) > 100 {
		response.BadRequest(c, "单次最多配置100个应用")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	targetIDs := make([]string, len(req.AppIDs))
	for i, id := range req.AppIDs {
		targetIDs[i] = fmt.Sprintf("%d", id)
	}

	input := &service.BatchOperationInput{
		OperationType: "application_config",
		TargetType:    "application",
		TargetIDs:     targetIDs,
		CreatedBy:     createdBy,
	}

	operation, err := handler.batchService.CreateOperation(input)
	if err != nil {
		response.InternalServerError(c, "创建批量操作失败")
		return
	}

	config := &service.ApplicationConfig{
		CaptchaTypes:         req.CaptchaTypes,
		MaxVerifyPerMinute:  req.MaxVerifyPerMinute,
		MaxVerifyPerDay:     req.MaxVerifyPerDay,
		AllowedIPs:          req.AllowedIPs,
		BlockRefusedRequests: req.BlockRefusedRequests,
	}

	go func() {
		ctx := context.Background()
		handler.batchService.ApplicationBatchUpdate(ctx, operation.ID, req.AppIDs, config)
	}()

	response.Success(c, gin.H{
		"operation_id": operation.ID,
		"total":        len(req.AppIDs),
		"message":      "批量配置任务已创建，正在后台处理",
	})
}

func GetBatchOperationProgress(c *gin.Context) {
	handler := NewBatchOperationHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的操作ID")
		return
	}

	operation, err := handler.batchService.GetOperation(uint(id))
	if err != nil {
		if err == service.ErrBatchOperationNotFound {
			response.NotFound(c, "批量操作记录不存在")
			return
		}
		response.InternalServerError(c, "获取批量操作详情失败")
		return
	}

	response.Success(c, gin.H{
		"operation_id": operation.ID,
		"status":       operation.Status,
		"progress":     operation.Progress,
		"total":        operation.Total,
		"processed":    operation.Processed,
		"succeeded":    operation.Succeeded,
		"failed":       operation.Failed,
		"skipped":      operation.Skipped,
	})
}

func RollbackBatchOperation(c *gin.Context) {
	handler := NewBatchOperationHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的操作ID")
		return
	}

	operation, err := handler.batchService.GetOperation(uint(id))
	if err != nil {
		if err == service.ErrBatchOperationNotFound {
			response.NotFound(c, "批量操作记录不存在")
			return
		}
		response.InternalServerError(c, "获取批量操作详情失败")
		return
	}

	if !operation.CanRollback {
		response.BadRequest(c, "该操作不支持回滚")
		return
	}

	if operation.IsRolledBack {
		response.BadRequest(c, "该操作已经回滚过")
		return
	}

	var rollbackErr error
	switch operation.OperationType {
	case "blacklist_import":
		rollbackErr = handler.batchService.RollbackBlacklistImport(uint(id))
	case "blacklist_delete":
		rollbackErr = handler.batchService.RollbackBlacklistDelete(uint(id))
	case "application_config":
		rollbackErr = handler.batchService.RollbackApplicationConfig(uint(id))
	default:
		response.BadRequest(c, "不支持的回滚类型")
		return
	}

	if rollbackErr != nil {
		response.InternalServerError(c, fmt.Sprintf("回滚失败: %s", rollbackErr.Error()))
		return
	}

	response.Success(c, gin.H{
		"message": "回滚成功",
	})
}

func CancelBatchOperation(c *gin.Context) {
	handler := NewBatchOperationHandler()
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的操作ID")
		return
	}

	if err := handler.batchService.CancelOperation(uint(id)); err != nil {
		response.InternalServerError(c, fmt.Sprintf("取消操作失败: %s", err.Error()))
		return
	}

	response.Success(c, gin.H{
		"message": "操作已取消",
	})
}

type RuleBatchUpdateRequest struct {
	RuleIDs   []uint `json:"rule_ids" binding:"required,min=1"`
	IsEnabled bool   `json:"is_enabled"`
}

func RuleBatchUpdate(c *gin.Context) {
	handler := NewBatchOperationHandler()
	var req RuleBatchUpdateRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if len(req.RuleIDs) == 0 {
		response.BadRequest(c, "请选择要更新的规则")
		return
	}

	if len(req.RuleIDs) > 500 {
		response.BadRequest(c, "单次最多更新500条规则")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	targetIDs := make([]string, len(req.RuleIDs))
	for i, id := range req.RuleIDs {
		targetIDs[i] = fmt.Sprintf("%d", id)
	}

	input := &service.BatchOperationInput{
		OperationType: "rule_update",
		TargetType:    "rule",
		TargetIDs:     targetIDs,
		Data: map[string]interface{}{
			"is_enabled": req.IsEnabled,
		},
		CreatedBy: createdBy,
	}

	operation, err := handler.batchService.CreateOperation(input)
	if err != nil {
		response.InternalServerError(c, "创建批量操作失败")
		return
	}

	go func() {
		ctx := context.Background()
		handler.batchService.RuleBatchUpdate(ctx, operation.ID, req.RuleIDs, req.IsEnabled)
	}()

	action := "启用"
	if !req.IsEnabled {
		action = "禁用"
	}

	response.Success(c, gin.H{
		"operation_id": operation.ID,
		"total":       len(req.RuleIDs),
		"message":     fmt.Sprintf("批量%s任务已创建，正在后台处理", action),
	})
}

func DownloadBlacklistTemplate(c *gin.Context) {
	blType := c.DefaultQuery("type", "ip")

	c.Header("Content-Type", "text/csv")
	c.Header("Content-Disposition", fmt.Sprintf("attachment; filename=blacklist_template_%s.csv", blType))

	writer := csv.NewWriter(c.Writer)

	switch blType {
	case "ip":
		writer.Write([]string{"192.168.1.1", "恶意IP"})
		writer.Write([]string{"10.0.0.0/24", "IP段"})
	case "user_id":
		writer.Write([]string{"user_malicious_001", "违规用户"})
		writer.Write([]string{"user_spam_002", "垃圾用户"})
	case "device_id":
		writer.Write([]string{"device_fp_abc123", "异常设备"})
		writer.Write([]string{"device_fp_def456", "可疑设备"})
	case "phone":
		writer.Write([]string{"13800138000", "营销电话"})
		writer.Write([]string{"13900139000", "骚扰电话"})
	case "email":
		writer.Write([]string{"spam@example.com", "垃圾邮件"})
		writer.Write([]string{"phishing@example.com", "钓鱼邮件"})
	default:
		writer.Write([]string{"target_value", "reason"})
	}

	writer.Flush()
}

func BlacklistImportSync(c *gin.Context) {
	handler := NewBatchOperationHandler()

	var jsonReq struct {
		Type       string   `json:"type" binding:"required"`
		Targets    []string `json:"targets" binding:"required,min=1"`
		Reason     string   `json:"reason"`
		Action     string   `json:"action"`
		Expiration string   `json:"expiration"`
	}

	if err := c.ShouldBindJSON(&jsonReq); err != nil {
		response.BadRequest(c, "无效的请求参数")
		return
	}

	if len(jsonReq.Targets) > 100 {
		response.BadRequest(c, "同步导入最多支持100条记录")
		return
	}

	adminID, _ := c.Get("admin_id")
	var createdBy uint
	if id, ok := adminID.(uint); ok {
		createdBy = id
	}

	input := &service.BatchOperationInput{
		OperationType: "blacklist_import",
		TargetType:    "blacklist",
		TargetIDs:     jsonReq.Targets,
		CreatedBy:     createdBy,
	}

	operation, err := handler.batchService.CreateOperation(input)
	if err != nil {
		response.InternalServerError(c, "创建批量操作失败")
		return
	}

	ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
	defer cancel()

	result, err := handler.batchService.BlacklistBatchImport(ctx, operation.ID, jsonReq.Targets, jsonReq.Type, jsonReq.Reason, jsonReq.Action, jsonReq.Expiration, createdBy)
	if err != nil {
		response.InternalServerError(c, fmt.Sprintf("导入失败: %s", err.Error()))
		return
	}

	response.Success(c, result)
}
