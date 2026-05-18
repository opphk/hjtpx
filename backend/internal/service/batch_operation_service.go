package service

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/redis"
)

var (
	ErrBatchOperationNotFound = errors.New("batch operation not found")
	ErrBatchOperationRunning  = errors.New("batch operation is still running")
	ErrCannotRollback         = errors.New("batch operation cannot be rolled back")
	ErrInvalidOperation       = errors.New("invalid batch operation")
)

type BatchOperationStatus string

const (
	BatchStatusPending   BatchOperationStatus = "pending"
	BatchStatusRunning   BatchOperationStatus = "running"
	BatchStatusCompleted BatchOperationStatus = "completed"
	BatchStatusFailed    BatchOperationStatus = "failed"
	BatchStatusCancelled BatchOperationStatus = "cancelled"
)

type BatchOperationService struct {
	progressCallback func(operationID string, progress int)
	mu              sync.RWMutex
}

func NewBatchOperationService() *BatchOperationService {
	return &BatchOperationService{}
}

func (s *BatchOperationService) SetProgressCallback(callback func(operationID string, progress int)) {
	s.progressCallback = callback
}

type BatchOperationInput struct {
	OperationType string   `json:"operation_type" binding:"required"`
	TargetType    string   `json:"target_type" binding:"required"`
	TargetIDs     []string `json:"target_ids" binding:"required,min=1"`
	Data          map[string]interface{} `json:"data,omitempty"`
	CreatedBy     uint     `json:"created_by"`
}

type BatchOperationResult struct {
	OperationID   string                 `json:"operation_id"`
	Total         int                    `json:"total"`
	Processed     int                    `json:"processed"`
	Succeeded     int                    `json:"succeeded"`
	Failed        int                    `json:"failed"`
	Skipped       int                    `json:"skipped"`
	Progress      int                    `json:"progress"`
	Status        BatchOperationStatus   `json:"status"`
	CanRollback   bool                   `json:"can_rollback"`
	RollbackData  map[string]interface{} `json:"rollback_data,omitempty"`
	FailedItems   []BatchFailedItem      `json:"failed_items,omitempty"`
}

type BatchFailedItem struct {
	TargetID   string `json:"target_id"`
	Error      string `json:"error"`
}

func (s *BatchOperationService) CreateOperation(input *BatchOperationInput) (*models.BatchOperation, error) {
	if len(input.TargetIDs) == 0 {
		return nil, ErrInvalidOperation
	}

	targetIDsJSON, err := json.Marshal(input.TargetIDs)
	if err != nil {
		return nil, fmt.Errorf("failed to marshal target IDs: %w", err)
	}

	operation := &models.BatchOperation{
		OperationType: input.OperationType,
		TargetType:    input.TargetType,
		TargetIDs:     string(targetIDsJSON),
		Status:        string(BatchStatusPending),
		Total:         len(input.TargetIDs),
		Processed:     0,
		Succeeded:     0,
		Failed:        0,
		Skipped:       0,
		Progress:      0,
		CanRollback:   false,
		IsRolledBack:  false,
		CreatedBy:     input.CreatedBy,
	}

	if err := database.DB.Create(operation).Error; err != nil {
		return nil, fmt.Errorf("failed to create batch operation: %w", err)
	}

	return operation, nil
}

func (s *BatchOperationService) GetOperation(id uint) (*models.BatchOperation, error) {
	var operation models.BatchOperation
	if err := database.DB.First(&operation, id).Error; err != nil {
		return nil, ErrBatchOperationNotFound
	}
	return &operation, nil
}

func (s *BatchOperationService) ListOperations(filter *ListBatchOperationsFilter) ([]models.BatchOperation, int64, error) {
	query := database.DB.Model(&models.BatchOperation{})

	if filter.TargetType != "" {
		query = query.Where("target_type = ?", filter.TargetType)
	}
	if filter.OperationType != "" {
		query = query.Where("operation_type = ?", filter.OperationType)
	}
	if filter.Status != "" {
		query = query.Where("status = ?", filter.Status)
	}
	if filter.CreatedBy > 0 {
		query = query.Where("created_by = ?", filter.CreatedBy)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		return nil, 0, err
	}

	offset := (filter.Page - 1) * filter.PageSize
	var operations []models.BatchOperation
	if err := query.Order("created_at DESC").Offset(offset).Limit(filter.PageSize).Find(&operations).Error; err != nil {
		return nil, 0, err
	}

	return operations, total, nil
}

type ListBatchOperationsFilter struct {
	Page          int
	PageSize      int
	TargetType    string
	OperationType string
	Status        string
	CreatedBy     uint
}

func (s *BatchOperationService) UpdateProgress(operationID uint, processed, succeeded, failed, skipped int) error {
	progress := 0
	if processed > 0 {
		progress = (processed * 100) / (processed + failed + skipped)
		if progress > 100 {
			progress = 100
		}
	}

	updates := map[string]interface{}{
		"processed": processed,
		"succeeded": succeeded,
		"failed":    failed,
		"skipped":   skipped,
		"progress":  progress,
	}

	if err := database.DB.Model(&models.BatchOperation{}).Where("id = ?", operationID).Updates(updates).Error; err != nil {
		return err
	}

	if s.progressCallback != nil {
		s.progressCallback(fmt.Sprintf("%d", operationID), progress)
	}

	return nil
}

func (s *BatchOperationService) CompleteOperation(operationID uint, status BatchOperationStatus, result string, errorMsg string, canRollback bool, rollbackData string) error {
	updates := map[string]interface{}{
		"status":       string(status),
		"result":       result,
		"error_message": errorMsg,
		"can_rollback": canRollback,
	}

	if status == BatchStatusCompleted || status == BatchStatusFailed {
		now := time.Now()
		updates["completed_at"] = now
	}

	if canRollback && rollbackData != "" {
		updates["rollback_data"] = rollbackData
	}

	return database.DB.Model(&models.BatchOperation{}).Where("id = ?", operationID).Updates(updates).Error
}

func (s *BatchOperationService) BlacklistBatchImport(ctx context.Context, operationID uint, targets []string, blType, reason, action, expiration string, createdBy uint) (*BatchOperationResult, error) {
	var operation models.BatchOperation
	if err := database.DB.First(&operation, operationID).Error; err != nil {
		return nil, ErrBatchOperationNotFound
	}

	if err := s.updateStatus(operationID, BatchStatusRunning); err != nil {
		return nil, err
	}

	result := &BatchOperationResult{
		OperationID: fmt.Sprintf("%d", operationID),
		Total:       len(targets),
		Status:      BatchStatusRunning,
	}

	blacklistService := NewBlacklistService()
	rollbackData := make(map[string]interface{})
	rollbackData["created_ids"] = []uint{}

	processed := 0
	succeeded := 0
	failed := 0
	skipped := 0
	failedItems := []BatchFailedItem{}

	for i, target := range targets {
		select {
		case <-ctx.Done():
			s.CompleteOperation(operationID, BatchStatusCancelled, "", "Operation cancelled by user", false, "")
			result.Status = BatchStatusCancelled
			return result, nil
		default:
		}

		input := &CreateBlacklistInput{
			Target:     target,
			Type:       blType,
			Source:     "import",
			Reason:     reason,
			Action:     action,
			Expiration: expiration,
			CreatedBy:  createdBy,
		}

		item, err := blacklistService.CreateBlacklist(input)
		if err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: target,
				Error:    err.Error(),
			})
		} else {
			succeeded++
			rollbackData["created_ids"] = append(rollbackData["created_ids"].([]uint), item.ID)
		}

		processed++
		if i%10 == 0 || i == len(targets)-1 {
			s.UpdateProgress(operationID, processed, succeeded, failed, skipped)
		}
	}

	rollbackDataJSON, _ := json.Marshal(rollbackData)

	status := BatchStatusCompleted
	if failed == len(targets) {
		status = BatchStatusFailed
	}

	resultData, _ := json.Marshal(map[string]interface{}{
		"succeeded": succeeded,
		"failed":    failed,
		"skipped":   skipped,
	})

	s.CompleteOperation(operationID, status, string(resultData), "", true, string(rollbackDataJSON))

	result.Processed = processed
	result.Succeeded = succeeded
	result.Failed = failed
	result.Skipped = skipped
	result.Progress = 100
	result.Status = status
	result.CanRollback = true
	result.FailedItems = failedItems

	return result, nil
}

func (s *BatchOperationService) BlacklistBatchDelete(ctx context.Context, operationID uint, ids []uint) (*BatchOperationResult, error) {
	var operation models.BatchOperation
	if err := database.DB.First(&operation, operationID).Error; err != nil {
		return nil, ErrBatchOperationNotFound
	}

	if err := s.updateStatus(operationID, BatchStatusRunning); err != nil {
		return nil, err
	}

	result := &BatchOperationResult{
		OperationID: fmt.Sprintf("%d", operationID),
		Total:       len(ids),
		Status:      BatchStatusRunning,
	}

	blacklistService := NewBlacklistService()
	rollbackData := make(map[string]interface{})
	rollbackData["deleted_items"] = []map[string]interface{}{}

	processed := 0
	succeeded := 0
	failed := 0
	failedItems := []BatchFailedItem{}

	for i, id := range ids {
		select {
		case <-ctx.Done():
			s.CompleteOperation(operationID, BatchStatusCancelled, "", "Operation cancelled by user", false, "")
			result.Status = BatchStatusCancelled
			return result, nil
		default:
		}

		item, err := blacklistService.GetBlacklistByID(id)
		if err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: fmt.Sprintf("%d", id),
				Error:    err.Error(),
			})
			processed++
			continue
		}

		deletedItem := map[string]interface{}{
			"target":        item.Target,
			"type":          item.Type,
			"source":        item.Source,
			"reason":        item.Reason,
			"action":        item.Action,
			"status":        item.Status,
			"note":          item.Note,
			"application_ids": item.ApplicationIDs,
			"expiration":    item.Expiration,
			"created_by":    item.CreatedBy,
		}
		rollbackData["deleted_items"] = append(rollbackData["deleted_items"].([]map[string]interface{}), deletedItem)

		if err := blacklistService.DeleteBlacklist(id); err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: fmt.Sprintf("%d", id),
				Error:    err.Error(),
			})
		} else {
			succeeded++
		}

		processed++
		if i%10 == 0 || i == len(ids)-1 {
			s.UpdateProgress(operationID, processed, succeeded, failed, 0)
		}
	}

	rollbackDataJSON, _ := json.Marshal(rollbackData)

	status := BatchStatusCompleted
	if failed == len(ids) {
		status = BatchStatusFailed
	}

	resultData, _ := json.Marshal(map[string]interface{}{
		"succeeded": succeeded,
		"failed":    failed,
	})

	s.CompleteOperation(operationID, status, string(resultData), "", true, string(rollbackDataJSON))

	result.Processed = processed
	result.Succeeded = succeeded
	result.Failed = failed
	result.Progress = 100
	result.Status = status
	result.CanRollback = true
	result.FailedItems = failedItems

	return result, nil
}

func (s *BatchOperationService) ApplicationBatchUpdate(ctx context.Context, operationID uint, ids []uint, config *ApplicationConfig) (*BatchOperationResult, error) {
	var operation models.BatchOperation
	if err := database.DB.First(&operation, operationID).Error; err != nil {
		return nil, ErrBatchOperationNotFound
	}

	if err := s.updateStatus(operationID, BatchStatusRunning); err != nil {
		return nil, err
	}

	result := &BatchOperationResult{
		OperationID: fmt.Sprintf("%d", operationID),
		Total:       len(ids),
		Status:      BatchStatusRunning,
	}

	appService := NewApplicationService()
	rollbackData := make(map[string]interface{})
	rollbackData["configs"] = []map[string]interface{}{}

	processed := 0
	succeeded := 0
	failed := 0
	failedItems := []BatchFailedItem{}

	for i, id := range ids {
		select {
		case <-ctx.Done():
			s.CompleteOperation(operationID, BatchStatusCancelled, "", "Operation cancelled by user", false, "")
			result.Status = BatchStatusCancelled
			return result, nil
		default:
		}

		oldConfig, err := appService.GetApplicationConfig(id)
		if err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: fmt.Sprintf("%d", id),
				Error:    err.Error(),
			})
			processed++
			continue
		}

		oldConfigMap := map[string]interface{}{}
		oldConfigJSON, _ := json.Marshal(oldConfig)
		json.Unmarshal(oldConfigJSON, &oldConfigMap)
		rollbackData["configs"] = append(rollbackData["configs"].([]map[string]interface{}), map[string]interface{}{
			"app_id": id,
			"config": oldConfigMap,
		})

		if err := appService.UpdateApplicationConfigWithMerge(id, config); err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: fmt.Sprintf("%d", id),
				Error:    err.Error(),
			})
		} else {
			succeeded++
		}

		processed++
		if i%10 == 0 || i == len(ids)-1 {
			s.UpdateProgress(operationID, processed, succeeded, failed, 0)
		}
	}

	rollbackDataJSON, _ := json.Marshal(rollbackData)

	status := BatchStatusCompleted
	if failed == len(ids) {
		status = BatchStatusFailed
	}

	resultData, _ := json.Marshal(map[string]interface{}{
		"succeeded": succeeded,
		"failed":    failed,
	})

	s.CompleteOperation(operationID, status, string(resultData), "", true, string(rollbackDataJSON))

	result.Processed = processed
	result.Succeeded = succeeded
	result.Failed = failed
	result.Progress = 100
	result.Status = status
	result.CanRollback = true
	result.FailedItems = failedItems

	return result, nil
}

func (s *BatchOperationService) updateStatus(operationID uint, status BatchOperationStatus) error {
	return database.DB.Model(&models.BatchOperation{}).Where("id = ?", operationID).Update("status", string(status)).Error
}

func (s *BatchOperationService) RollbackBlacklistImport(operationID uint) error {
	operation, err := s.GetOperation(operationID)
	if err != nil {
		return err
	}

	if operation.OperationType != "blacklist_import" {
		return ErrCannotRollback
	}

	if !operation.CanRollback || operation.IsRolledBack {
		return ErrCannotRollback
	}

	var rollbackData struct {
		CreatedIDs []uint `json:"created_ids"`
	}
	if err := json.Unmarshal([]byte(operation.RollbackData), &rollbackData); err != nil {
		return fmt.Errorf("failed to parse rollback data: %w", err)
	}

	blacklistService := NewBlacklistService()
	deletedCount := 0
	for _, id := range rollbackData.CreatedIDs {
		if err := blacklistService.DeleteBlacklist(id); err == nil {
			deletedCount++
		}
	}

	now := time.Now()
	database.DB.Model(operation).Updates(map[string]interface{}{
		"is_rolled_back": true,
		"rollback_at":    now,
	})

	if redis.Client != nil {
		redis.Client.Del(context.Background(), fmt.Sprintf("batch_progress:%d", operationID))
	}

	return nil
}

func (s *BatchOperationService) RollbackBlacklistDelete(operationID uint) error {
	operation, err := s.GetOperation(operationID)
	if err != nil {
		return err
	}

	if operation.OperationType != "blacklist_delete" {
		return ErrCannotRollback
	}

	if !operation.CanRollback || operation.IsRolledBack {
		return ErrCannotRollback
	}

	var rollbackData struct {
		DeletedItems []map[string]interface{} `json:"deleted_items"`
	}
	if err := json.Unmarshal([]byte(operation.RollbackData), &rollbackData); err != nil {
		return fmt.Errorf("failed to parse rollback data: %w", err)
	}

	blacklistService := NewBlacklistService()
	restoredCount := 0
	for _, item := range rollbackData.DeletedItems {
		input := &CreateBlacklistInput{
			Target:         getStringValueFromMap(item, "target"),
			Type:           getStringValueFromMap(item, "type"),
			Source:         getStringValueFromMap(item, "source"),
			Reason:         getStringValueFromMap(item, "reason"),
			Action:         getStringValueFromMap(item, "action"),
			Expiration:     getStringValueFromMap(item, "expiration"),
			ApplicationIDs: parseApplicationIDsFromMap(item),
		}
		if _, err := blacklistService.CreateBlacklist(input); err == nil {
			restoredCount++
		}
	}

	now := time.Now()
	database.DB.Model(operation).Updates(map[string]interface{}{
		"is_rolled_back": true,
		"rollback_at":    now,
	})

	return nil
}

func getStringValueFromMap(m map[string]interface{}, key string) string {
	if v, ok := m[key].(string); ok {
		return v
	}
	return ""
}

func parseApplicationIDsFromMap(m map[string]interface{}) []string {
	if v, ok := m["application_ids"].(string); ok && v != "" {
		var ids []string
		json.Unmarshal([]byte(v), &ids)
		return ids
	}
	return nil
}

func (s *BatchOperationService) RollbackApplicationConfig(operationID uint) error {
	operation, err := s.GetOperation(operationID)
	if err != nil {
		return err
	}

	if operation.OperationType != "application_config" {
		return ErrCannotRollback
	}

	if !operation.CanRollback || operation.IsRolledBack {
		return ErrCannotRollback
	}

	var rollbackData struct {
		Configs []struct {
			AppID  uint               `json:"app_id"`
			Config map[string]interface{} `json:"config"`
		} `json:"configs"`
	}
	if err := json.Unmarshal([]byte(operation.RollbackData), &rollbackData); err != nil {
		return fmt.Errorf("failed to parse rollback data: %w", err)
	}

	appService := NewApplicationService()
	restoredCount := 0
	for _, item := range rollbackData.Configs {
		config := &ApplicationConfig{}
		configJSON, _ := json.Marshal(item.Config)
		json.Unmarshal(configJSON, config)
		if _, err := appService.UpdateApplicationConfig(item.AppID, config); err == nil {
			restoredCount++
		}
	}

	now := time.Now()
	database.DB.Model(operation).Updates(map[string]interface{}{
		"is_rolled_back": true,
		"rollback_at":    now,
	})

	return nil
}

func (s *BatchOperationService) CancelOperation(operationID uint) error {
	operation, err := s.GetOperation(operationID)
	if err != nil {
		return err
	}

	if operation.Status == string(BatchStatusCompleted) || operation.Status == string(BatchStatusFailed) {
		return fmt.Errorf("cannot cancel completed or failed operation")
	}

	return s.updateStatus(operationID, BatchStatusCancelled)
}

func (s *BatchOperationService) RuleBatchUpdate(ctx context.Context, operationID uint, ruleIDs []uint, isEnabled bool) (*BatchOperationResult, error) {
	var operation models.BatchOperation
	if err := database.DB.First(&operation, operationID).Error; err != nil {
		return nil, ErrBatchOperationNotFound
	}

	if err := s.updateStatus(operationID, BatchStatusRunning); err != nil {
		return nil, err
	}

	result := &BatchOperationResult{
		OperationID: fmt.Sprintf("%d", operationID),
		Total:       len(ruleIDs),
		Status:      BatchStatusRunning,
	}

	rollbackData := make(map[string]interface{})
	rollbackData["updates"] = []map[string]interface{}{}

	processed := 0
	succeeded := 0
	failed := 0
	failedItems := []BatchFailedItem{}

	for i, ruleID := range ruleIDs {
		select {
		case <-ctx.Done():
			s.CompleteOperation(operationID, BatchStatusCancelled, "", "Operation cancelled by user", false, "")
			result.Status = BatchStatusCancelled
			return result, nil
		default:
		}

		var rule struct {
			ID        uint   `gorm:"primaryKey"`
			IsEnabled bool   `gorm:"default:true"`
			Name      string `gorm:"size:255"`
		}

		if err := database.DB.Table("risk_rules").First(&rule, ruleID).Error; err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: fmt.Sprintf("%d", ruleID),
				Error:    err.Error(),
			})
			processed++
			continue
		}

		oldEnabled := rule.IsEnabled
		rollbackData["updates"] = append(rollbackData["updates"].([]map[string]interface{}), map[string]interface{}{
			"rule_id":    ruleID,
			"old_enabled": oldEnabled,
		})

		if err := database.DB.Table("risk_rules").Where("id = ?", ruleID).Update("is_enabled", isEnabled).Error; err != nil {
			failed++
			failedItems = append(failedItems, BatchFailedItem{
				TargetID: fmt.Sprintf("%d", ruleID),
				Error:    err.Error(),
			})
		} else {
			succeeded++
		}

		processed++
		if i%10 == 0 || i == len(ruleIDs)-1 {
			s.UpdateProgress(operationID, processed, succeeded, failed, 0)
		}
	}

	rollbackDataJSON, _ := json.Marshal(rollbackData)

	status := BatchStatusCompleted
	if failed == len(ruleIDs) {
		status = BatchStatusFailed
	}

	resultData, _ := json.Marshal(map[string]interface{}{
		"succeeded": succeeded,
		"failed":    failed,
	})

	s.CompleteOperation(operationID, status, string(resultData), "", true, string(rollbackDataJSON))

	result.Processed = processed
	result.Succeeded = succeeded
	result.Failed = failed
	result.Progress = 100
	result.Status = status
	result.CanRollback = true
	result.FailedItems = failedItems

	return result, nil
}

func (s *BatchOperationService) RollbackRuleUpdate(operationID uint) error {
	operation, err := s.GetOperation(operationID)
	if err != nil {
		return err
	}

	if operation.OperationType != "rule_update" {
		return ErrCannotRollback
	}

	if !operation.CanRollback || operation.IsRolledBack {
		return ErrCannotRollback
	}

	var rollbackData struct {
		Updates []struct {
			RuleID     uint `json:"rule_id"`
			OldEnabled bool `json:"old_enabled"`
		} `json:"updates"`
	}
	if err := json.Unmarshal([]byte(operation.RollbackData), &rollbackData); err != nil {
		return fmt.Errorf("failed to parse rollback data: %w", err)
	}

	restoredCount := 0
	for _, update := range rollbackData.Updates {
		if err := database.DB.Table("risk_rules").Where("id = ?", update.RuleID).Update("is_enabled", update.OldEnabled).Error; err == nil {
			restoredCount++
		}
	}

	now := time.Now()
	database.DB.Model(operation).Updates(map[string]interface{}{
		"is_rolled_back": true,
		"rollback_at":    now,
	})

	return nil
}
