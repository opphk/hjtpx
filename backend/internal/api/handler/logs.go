package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/hjtpx/hjtpx/pkg/response"
	"gorm.io/gorm"
)

// GetVerificationLogsRequest 验证日志查询请求
type GetVerificationLogsRequest struct {
	Page          int     `form:"page,default=1"`
	PageSize      int     `form:"page_size,default=20"`
	ApplicationID uint    `form:"application_id"`
	Status        string  `form:"status"`
	CaptchaType   string  `form:"captcha_type"`
	SessionID     string  `form:"session_id"`
	StartDate     string  `form:"start_date"`
	EndDate       string  `form:"end_date"`
	MinRiskScore  float64 `form:"min_risk_score"`
	MaxRiskScore  float64 `form:"max_risk_score"`
}

// LogListResponse 日志列表响应
type LogListResponse struct {
	Total      int64                     `json:"total"`
	Page       int                       `json:"page"`
	PageSize   int                       `json:"page_size"`
	TotalPages int                       `json:"total_pages"`
	Logs       []models.VerificationLog `json:"logs"`
}

// GetVerificationLogs 获取验证日志列表
func GetVerificationLogs(c *gin.Context) {
	var req GetVerificationLogsRequest
	if err := c.ShouldBindQuery(&req); err != nil {
		response.BadRequest(c, "无效的查询参数")
		return
	}

	if req.Page < 1 {
		req.Page = 1
	}
	if req.PageSize < 1 || req.PageSize > 100 {
		req.PageSize = 20
	}

	db := database.GetDB()
	query := db.Model(&models.VerificationLog{})

	if req.ApplicationID > 0 {
		query = query.Where("application_id = ?", req.ApplicationID)
	}
	if req.Status != "" {
		query = query.Where("status = ?", req.Status)
	}
	if req.CaptchaType != "" {
		query = query.Where("captcha_type = ?", req.CaptchaType)
	}
	if req.SessionID != "" {
		query = query.Where("session_id LIKE ?", "%"+req.SessionID+"%")
	}
	if req.StartDate != "" {
		if startDate, err := time.Parse("2006-01-02", req.StartDate); err == nil {
			query = query.Where("created_at >= ?", startDate)
		}
	}
	if req.EndDate != "" {
		if endDate, err := time.Parse("2006-01-02", req.EndDate); err == nil {
			endDate = endDate.Add(24 * time.Hour)
			query = query.Where("created_at < ?", endDate)
		}
	}
	if req.MinRiskScore > 0 {
		query = query.Where("risk_score >= ?", req.MinRiskScore)
	}
	if req.MaxRiskScore > 0 {
		query = query.Where("risk_score <= ?", req.MaxRiskScore)
	}

	var total int64
	if err := query.Count(&total).Error; err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	var logs []models.VerificationLog
	offset := (req.Page - 1) * req.PageSize
	if err := query.Preload("Application").
		Order("created_at DESC").
		Offset(offset).
		Limit(req.PageSize).
		Find(&logs).Error; err != nil {
		response.InternalServerError(c, "查询失败")
		return
	}

	totalPages := int((total + int64(req.PageSize) - 1) / int64(req.PageSize))

	response.Success(c, LogListResponse{
		Total:      total,
		Page:       req.Page,
		PageSize:   req.PageSize,
		TotalPages: totalPages,
		Logs:       logs,
	})
}

// GetLogDetail 获取日志详情
func GetLogDetail(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "无效的日志ID")
		return
	}

	db := database.GetDB()
	var log models.VerificationLog
	if err := db.Preload("Verification").
		Preload("Verification.BehaviorData").
		Preload("Application").
		First(&log, id).Error; err != nil {
		if err == gorm.ErrRecordNotFound {
			response.NotFound(c, "日志不存在")
		} else {
			response.InternalServerError(c, "查询失败")
		}
		return
	}

	response.Success(c, log)
}

// GetLogStatistics 获取日志统计信息
func GetLogStatistics(c *gin.Context) {
	db := database.GetDB()

	var totalCount int64
	var successCount int64
	var failedCount int64
	var avgRiskScore float64

	db.Model(&models.VerificationLog{}).Count(&totalCount)
	db.Model(&models.VerificationLog{}).Where("status = ?", "success").Count(&successCount)
	db.Model(&models.VerificationLog{}).Where("status = ?", "failed").Count(&failedCount)

	rows, _ := db.Model(&models.VerificationLog{}).Select("AVG(risk_score) as avg_risk").Rows()
	if rows.Next() {
		rows.Scan(&avgRiskScore)
	}

	type CaptchaStats struct {
		CaptchaType string  `json:"captcha_type"`
		Count       int64   `json:"count"`
		SuccessRate float64 `json:"success_rate"`
	}

	var captchaStats []CaptchaStats
	db.Model(&models.VerificationLog{}).
		Select("captcha_type, COUNT(*) as count").
		Group("captcha_type").
		Scan(&captchaStats)

	for i := range captchaStats {
		var success int64
		db.Model(&models.VerificationLog{}).
			Where("captcha_type = ? AND status = ?", captchaStats[i].CaptchaType, "success").
			Count(&success)
		if captchaStats[i].Count > 0 {
			captchaStats[i].SuccessRate = float64(success) / float64(captchaStats[i].Count)
		}
	}

	response.Success(c, gin.H{
		"total_count":     totalCount,
		"success_count":   successCount,
		"failed_count":    failedCount,
		"success_rate":    func() float64 { if totalCount > 0 { return float64(successCount) / float64(totalCount) }; return 0 }(),
		"avg_risk_score":  avgRiskScore,
		"captcha_stats":   captchaStats,
	})
}
