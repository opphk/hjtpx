package models

import (
	"time"

	"gorm.io/gorm"
)

type BatchOperation struct {
	gorm.Model
	OperationType string         `gorm:"size:50;not null;index:idx_batch_type" json:"operation_type"`
	TargetType    string         `gorm:"size:50;not null;index:idx_batch_target" json:"target_type"`
	TargetIDs     string         `gorm:"type:text;not null" json:"target_ids"`
	Status        string         `gorm:"size:50;default:pending;index:idx_batch_status" json:"status"`
	Total         int            `gorm:"default:0" json:"total"`
	Processed     int            `gorm:"default:0" json:"processed"`
	Succeeded     int            `gorm:"default:0" json:"succeeded"`
	Failed        int            `gorm:"default:0" json:"failed"`
	Skipped       int            `gorm:"default:0" json:"skipped"`
	Progress      int            `gorm:"default:0" json:"progress"`
	Result        string         `gorm:"type:text" json:"result,omitempty"`
	ErrorMessage  string         `gorm:"type:text" json:"error_message,omitempty"`
	RollbackData  string         `gorm:"type:text" json:"rollback_data,omitempty"`
	CanRollback   bool           `gorm:"default:false" json:"can_rollback"`
	IsRolledBack  bool           `gorm:"default:false" json:"is_rolled_back"`
	CreatedBy     uint           `gorm:"default:0" json:"created_by"`
	CompletedAt   *time.Time     `json:"completed_at,omitempty"`
	RollbackAt    *time.Time     `json:"rollback_at,omitempty"`
}

func (BatchOperation) TableName() string {
	return "batch_operations"
}

type BatchOperationItem struct {
	ID              uint      `gorm:"primaryKey" json:"id"`
	BatchOperationID uint     `gorm:"not null;index:idx_batch_item_operation" json:"batch_operation_id"`
	TargetID        string    `gorm:"size:255;not null" json:"target_id"`
	TargetType      string    `gorm:"size:50" json:"target_type"`
	Status          string    `gorm:"size:50;default:pending" json:"status"`
	BeforeData      string    `gorm:"type:text" json:"before_data,omitempty"`
	AfterData       string    `gorm:"type:text" json:"after_data,omitempty"`
	ErrorMessage    string    `gorm:"type:text" json:"error_message,omitempty"`
	ProcessedAt     *time.Time `json:"processed_at,omitempty"`
}

func (BatchOperationItem) TableName() string {
	return "batch_operation_items"
}
