package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
	"gorm.io/gorm"
)

type CreateApplicationRequest struct {
	Name        string `json:"name" binding:"required"`
	UserID      uint   `json:"user_id" binding:"required"`
	Description string `json:"description"`
}

type UpdateApplicationRequest struct {
	Name        string `json:"name"`
	Description string `json:"description"`
	IsActive    *bool  `json:"is_active"`
}

type ListApplicationsQuery struct {
	Page     int    `form:"page,default=1"`
	PageSize int    `form:"page_size,default=10"`
	Keyword  string `form:"keyword"`
}

type PaginatedApplications struct {
	Data  []models.Application `json:"data"`
	Total int64                `json:"total"`
	Page  int                  `json:"page"`
	PageSize int               `json:"page_size"`
}

// ListApplications 获取应用列表
func ListApplications(c *gin.Context) {
	var query ListApplicationsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query parameters")
		return
	}

	if query.Page < 1 {
		query.Page = 1
	}
	if query.PageSize < 1 || query.PageSize > 100 {
		query.PageSize = 10
	}

	db := database.DB.Model(&models.Application{})

	if query.Keyword != "" {
		db = db.Where("name LIKE ? OR description LIKE ?", "%"+query.Keyword+"%", "%"+query.Keyword+"%")
	}

	var total int64
	db.Count(&total)

	var applications []models.Application
	offset := (query.Page - 1) * query.PageSize
	db.Preload("User").Offset(offset).Limit(query.PageSize).Order("created_at DESC").Find(&applications)

	response.Success(c, PaginatedApplications{
		Data:      applications,
		Total:     total,
		Page:      query.Page,
		PageSize:  query.PageSize,
	})
}

// CreateApplication 创建应用
func CreateApplication(c *gin.Context) {
	var req CreateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	var user models.User
	if err := database.DB.First(&user, req.UserID).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "user not found")
		} else {
			response.InternalServerError(c, "")
		}
		return
	}

	apiKey := uuid.New().String()

	application := models.Application{
		Name:        req.Name,
		UserID:      req.UserID,
		Description: req.Description,
		APIKey:      apiKey,
		IsActive:    true,
	}

	if err := database.DB.Create(&application).Error; err != nil {
		response.InternalServerError(c, "")
		return
	}

	response.Success(c, application)
}

// UpdateApplication 更新应用
func UpdateApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	var req UpdateApplicationRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters")
		return
	}

	var application models.Application
	if err := database.DB.First(&application, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "application not found")
		} else {
			response.InternalServerError(c, "")
		}
		return
	}

	updates := make(map[string]interface{})
	if req.Name != "" {
		updates["name"] = req.Name
	}
	if req.Description != "" {
		updates["description"] = req.Description
	}
	if req.IsActive != nil {
		updates["is_active"] = *req.IsActive
	}

	if len(updates) > 0 {
		if err := database.DB.Model(&application).Updates(updates).Error; err != nil {
			response.InternalServerError(c, "")
			return
		}
	}

	database.DB.Preload("User").First(&application, id)
	response.Success(c, application)
}

// DeleteApplication 删除应用
func DeleteApplication(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid application id")
		return
	}

	var application models.Application
	if err := database.DB.First(&application, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "application not found")
		} else {
			response.InternalServerError(c, "")
		}
		return
	}

	if err := database.DB.Delete(&application).Error; err != nil {
		response.InternalServerError(c, "")
		return
	}

	response.Success(c, nil)
}
