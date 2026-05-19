package handler

import (
	"context"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type AIModelV3Handler struct {
	aiService *service.AIModelV3Service
}

func NewAIModelV3Handler() *AIModelV3Handler {
	return &AIModelV3Handler{
		aiService: service.NewAIModelV3Service(),
	}
}

func (h *AIModelV3Handler) Initialize(ctx context.Context) error {
	return h.aiService.Initialize(ctx)
}

func (h *AIModelV3Handler) GenerateSmartCaptcha(c *gin.Context) {
	var req service.GenerateSmartCaptchaRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	if req.Difficulty == 0 {
		req.Difficulty = 2
	}

	captcha, err := h.aiService.GenerateSmartCaptcha(c, req.Scene, req.Difficulty, req.RiskContext)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to generate captcha")
		return
	}

	response.Success(c, service.GenerateSmartCaptchaResponse{
		Success: true,
		Captcha: captcha,
	})
}

func (h *AIModelV3Handler) ComprehensiveRiskAssessment(c *gin.Context) {
	var req service.ComprehensiveRiskAssessmentRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	result, err := h.aiService.ComprehensiveRiskAssessment(c, req.TraceData, req.DeviceInfo)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to assess risk")
		return
	}

	response.Success(c, service.ComprehensiveRiskAssessmentResponse{
		Success: true,
		Result:  result,
	})
}

func (h *AIModelV3Handler) RecordFeedback(c *gin.Context) {
	var req service.RecordFeedbackRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.Error(c, http.StatusBadRequest, "Invalid request")
		return
	}

	err := h.aiService.RecordFeedback(c, req.TraceData, req.IsCorrect, req.Metadata)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to record feedback")
		return
	}

	response.Success(c, service.RecordFeedbackResponse{
		Success: true,
		Message: "Feedback recorded successfully",
	})
}

func (h *AIModelV3Handler) GetLearningStats(c *gin.Context) {
	stats, err := h.aiService.GetLearningStats(c)
	if err != nil {
		response.Error(c, http.StatusInternalServerError, "Failed to get learning stats")
		return
	}

	response.Success(c, service.GetLearningStatsResponse{
		Success: true,
		Stats:   stats,
	})
}
