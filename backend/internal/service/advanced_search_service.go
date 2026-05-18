package service

import (
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"gorm.io/gorm"
)

// SearchOperator 搜索操作符类型
type SearchOperator string

const (
	OpEquals      SearchOperator = "eq"
	OpNotEquals   SearchOperator = "ne"
	OpGreaterThan SearchOperator = "gt"
	OpGreaterOrEq SearchOperator = "gte"
	OpLessThan    SearchOperator = "lt"
	OpLessOrEq    SearchOperator = "lte"
	OpContains    SearchOperator = "contains"
	OpStartsWith  SearchOperator = "starts_with"
	OpEndsWith    SearchOperator = "ends_with"
	OpIn          SearchOperator = "in"
	OpNotIn       SearchOperator = "not_in"
	OpIsNull      SearchOperator = "is_null"
	OpIsNotNull   SearchOperator = "is_not_null"
	OpBetween     SearchOperator = "between"
)

// SearchCondition 单个搜索条件
type SearchCondition struct {
	Field    string         `json:"field"`
	Operator SearchOperator `json:"operator"`
	Value    interface{}    `json:"value"`
}

// SortOption 排序选项
type SortOption struct {
	Field string `json:"field"`
	Order string `json:"order"` // asc 或 desc
}

// AdvancedSearchQuery 高级搜索查询
type AdvancedSearchQuery struct {
	Conditions []SearchCondition `json:"conditions"`
	Sort       []SortOption      `json:"sort"`
	Page       int               `json:"page"`
	PageSize   int               `json:"page_size"`
}

// SearchResult 搜索结果
type SearchResult struct {
	Total      int64       `json:"total"`
	Page       int         `json:"page"`
	PageSize   int         `json:"page_size"`
	TotalPages int         `json:"total_pages"`
	Data       interface{} `json:"data"`
}

// SavedSearch 保存的搜索
type SavedSearch struct {
	ID          uint      `gorm:"primaryKey" json:"id"`
	Name        string    `gorm:"size:255;not null" json:"name"`
	EntityType  string    `gorm:"size:50;not null;index" json:"entity_type"` // logs, applications, blacklist
	Query       string    `gorm:"type:text" json:"query"`                    // JSON格式的AdvancedSearchQuery
	Description string    `gorm:"type:text" json:"description"`
	CreatedBy   uint      `gorm:"index" json:"created_by"`
	CreatedAt   time.Time `json:"created_at"`
}

// TableName 指定表名
func (SavedSearch) TableName() string {
	return "saved_searches"
}

// AdvancedSearchService 高级搜索服务
type AdvancedSearchService struct{}

// NewAdvancedSearchService 创建新的高级搜索服务
func NewAdvancedSearchService() *AdvancedSearchService {
	return &AdvancedSearchService{}
}

// BuildQuery 构建GORM查询
func (s *AdvancedSearchService) BuildQuery(db *gorm.DB, conditions []SearchCondition) *gorm.DB {
	query := db
	for _, cond := range conditions {
		query = s.applyCondition(query, cond)
	}
	return query
}

// applyCondition 应用单个查询条件
func (s *AdvancedSearchService) applyCondition(db *gorm.DB, cond SearchCondition) *gorm.DB {
	switch cond.Operator {
	case OpEquals:
		return db.Where(fmt.Sprintf("%s = ?", cond.Field), cond.Value)
	case OpNotEquals:
		return db.Where(fmt.Sprintf("%s != ?", cond.Field), cond.Value)
	case OpGreaterThan:
		return db.Where(fmt.Sprintf("%s > ?", cond.Field), cond.Value)
	case OpGreaterOrEq:
		return db.Where(fmt.Sprintf("%s >= ?", cond.Field), cond.Value)
	case OpLessThan:
		return db.Where(fmt.Sprintf("%s < ?", cond.Field), cond.Value)
	case OpLessOrEq:
		return db.Where(fmt.Sprintf("%s <= ?", cond.Field), cond.Value)
	case OpContains:
		return db.Where(fmt.Sprintf("%s LIKE ?", cond.Field), "%"+fmt.Sprintf("%v", cond.Value)+"%")
	case OpStartsWith:
		return db.Where(fmt.Sprintf("%s LIKE ?", cond.Field), fmt.Sprintf("%v", cond.Value)+"%")
	case OpEndsWith:
		return db.Where(fmt.Sprintf("%s LIKE ?", cond.Field), "%"+fmt.Sprintf("%v", cond.Value))
	case OpIn:
		if values, ok := cond.Value.([]interface{}); ok {
			return db.Where(fmt.Sprintf("%s IN ?", cond.Field), values)
		}
		return db
	case OpNotIn:
		if values, ok := cond.Value.([]interface{}); ok {
			return db.Where(fmt.Sprintf("%s NOT IN ?", cond.Field), values)
		}
		return db
	case OpIsNull:
		return db.Where(fmt.Sprintf("%s IS NULL", cond.Field))
	case OpIsNotNull:
		return db.Where(fmt.Sprintf("%s IS NOT NULL", cond.Field))
	case OpBetween:
		if values, ok := cond.Value.([]interface{}); ok && len(values) == 2 {
			return db.Where(fmt.Sprintf("%s BETWEEN ? AND ?", cond.Field), values[0], values[1])
		}
		return db
	default:
		return db
	}
}

// ApplySort 应用排序
func (s *AdvancedSearchService) ApplySort(db *gorm.DB, sortOptions []SortOption) *gorm.DB {
	query := db
	for _, sort := range sortOptions {
		if sort.Order == "desc" {
			query = query.Order(fmt.Sprintf("%s DESC", sort.Field))
		} else {
			query = query.Order(fmt.Sprintf("%s ASC", sort.Field))
		}
	}
	return query
}

// SearchLogs 搜索日志
func (s *AdvancedSearchService) SearchLogs(query AdvancedSearchQuery) (*SearchResult, error) {
	var logs []models.VerificationLog
	var total int64

	db := database.DB.Model(&models.VerificationLog{}).Preload("Application")
	db = s.BuildQuery(db, query.Conditions)

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	db = s.ApplySort(db, query.Sort)
	if len(query.Sort) == 0 {
		db = db.Order("created_at DESC")
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Find(&logs).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))

	return &SearchResult{
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
		Data:       logs,
	}, nil
}

// SearchApplications 搜索应用
func (s *AdvancedSearchService) SearchApplications(query AdvancedSearchQuery) (*SearchResult, error) {
	var apps []models.Application
	var total int64

	db := database.DB.Model(&models.Application{}).Preload("User")
	db = s.BuildQuery(db, query.Conditions)

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	db = s.ApplySort(db, query.Sort)
	if len(query.Sort) == 0 {
		db = db.Order("created_at DESC")
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Find(&apps).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))

	return &SearchResult{
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
		Data:       apps,
	}, nil
}

// SearchBlacklist 搜索黑名单
func (s *AdvancedSearchService) SearchBlacklist(query AdvancedSearchQuery) (*SearchResult, error) {
	var items []models.Blacklist
	var total int64

	db := database.DB.Model(&models.Blacklist{})
	db = s.BuildQuery(db, query.Conditions)

	if err := db.Count(&total).Error; err != nil {
		return nil, err
	}

	db = s.ApplySort(db, query.Sort)
	if len(query.Sort) == 0 {
		db = db.Order("created_at DESC")
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 20
	}

	offset := (query.Page - 1) * query.PageSize
	if err := db.Offset(offset).Limit(query.PageSize).Find(&items).Error; err != nil {
		return nil, err
	}

	totalPages := int((total + int64(query.PageSize) - 1) / int64(query.PageSize))

	return &SearchResult{
		Total:      total,
		Page:       query.Page,
		PageSize:   query.PageSize,
		TotalPages: totalPages,
		Data:       items,
	}, nil
}

// SaveSearch 保存搜索
func (s *AdvancedSearchService) SaveSearch(name, entityType string, query AdvancedSearchQuery, description string, createdBy uint) (*SavedSearch, error) {
	queryJSON, err := json.Marshal(query)
	if err != nil {
		return nil, err
	}

	savedSearch := &SavedSearch{
		Name:        name,
		EntityType:  entityType,
		Query:       string(queryJSON),
		Description: description,
		CreatedBy:   createdBy,
		CreatedAt:   time.Now(),
	}

	if err := database.DB.Create(savedSearch).Error; err != nil {
		return nil, err
	}

	return savedSearch, nil
}

// GetSavedSearches 获取保存的搜索
func (s *AdvancedSearchService) GetSavedSearches(entityType string, createdBy uint) ([]SavedSearch, error) {
	var searches []SavedSearch
	query := database.DB.Where("entity_type = ?", entityType)
	if createdBy > 0 {
		query = query.Where("created_by = ?", createdBy)
	}
	err := query.Order("created_at DESC").Find(&searches).Error
	return searches, err
}

// GetSavedSearch 获取单个保存的搜索
func (s *AdvancedSearchService) GetSavedSearch(id uint) (*SavedSearch, error) {
	var search SavedSearch
	err := database.DB.First(&search, id).Error
	if err != nil {
		return nil, err
	}
	return &search, nil
}

// DeleteSavedSearch 删除保存的搜索
func (s *AdvancedSearchService) DeleteSavedSearch(id uint) error {
	return database.DB.Delete(&SavedSearch{}, id).Error
}

// ParseQuery 解析保存的查询
func (s *AdvancedSearchService) ParseQuery(savedSearch *SavedSearch) (*AdvancedSearchQuery, error) {
	var query AdvancedSearchQuery
	err := json.Unmarshal([]byte(savedSearch.Query), &query)
	return &query, err
}
