package handler

import (
	"strconv"
	"time"

	"github.com/gin-gonic/gin"

"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type EnhancedAdaptiveDifficultyHandler struct {
	enhancedService *service.EnhancedAdaptiveDifficultyService
}

func NewEnhancedAdaptiveDifficultyHandler() *EnhancedAdaptiveDifficultyHandler {
	return &EnhancedAdaptiveDifficultyHandler{
		enhancedService: service.NewEnhancedAdaptiveDifficultyService(),
	}
}

var enhancedAdaptiveHandler = NewEnhancedAdaptiveDifficultyHandler()

func GetEnhancedAdaptiveDifficultyHandler() *EnhancedAdaptiveDifficultyHandler {
	return enhancedAdaptiveHandler
}

type EnhancedDifficultyRequest struct {
	UserID string `json:"user_id" example:"user123"`
	HighRiskContext       bool   `json:"high_risk_context" example:"false"`
	TimeSensitive         bool   `json:"time_sensitive" example:"false"`
	UserRequestedDifficulty string `json:"user_requested_difficulty" example:""`
}

type EnhancedDifficultyResponse struct {
	Difficulty       service.DifficultyLevel                `json:"difficulty"`
	UserID           string                                `json:"user_id"`
	Recommendation   *service.DifficultyRecommendation     `json:"recommendation"`
	Analytics        *service.EnhancedUserAnalyticsReport   `json:"analytics"`
}

func (h *EnhancedAdaptiveDifficultyHandler) GetEnhancedDifficulty(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "anonymous_" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
	}

	highRiskContext := c.Query("high_risk_context") == "true"
	timeSensitive := c.Query("time_sensitive") == "true"
	userRequestedDifficulty := c.Query("user_requested_difficulty")

	context := &service.DifficultyContext{
		HighRiskContext:         highRiskContext,
		TimeSensitive:          timeSensitive,
		UserRequestedDifficulty: userRequestedDifficulty,
	}

	difficulty, recommendation := h.enhancedService.GetEnhancedDifficulty(userID, context)
	analytics := h.enhancedService.GetEnhancedUserAnalytics(userID)

	response.Success(c, gin.H{
		"difficulty":    difficulty,
		"user_id":      userID,
		"recommendation": recommendation,
		"analytics":    analytics,
	})
}

type EnhancedResultRequest struct {
	UserID      string  `json:"user_id" binding:"required" example:"user123"`
	Success     bool    `json:"success" binding:"required" example:"true"`
	Time        int64   `json:"time" example:"1500"`
	Method      string  `json:"method" example:"slider"`
	Difficulty  string  `json:"difficulty" example:"Medium"`
	Context     *service.EnhancedVerificationContext `json:"context,omitempty"`
}

func (h *EnhancedAdaptiveDifficultyHandler) UpdateEnhancedResult(c *gin.Context) {
	var req EnhancedResultRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	difficulty := service.DifficultyLevel(req.Difficulty)
	if difficulty == "" {
		difficulty = service.DifficultyMedium
	}

	h.enhancedService.UpdateEnhancedDifficultyWithContext(
		req.UserID,
		difficulty,
		req.Success,
		time.Duration(req.Time)*time.Millisecond,
		req.Method,
		req.Context,
	)

	newDifficulty, _ := h.enhancedService.GetEnhancedDifficulty(req.UserID, nil)

	response.Success(c, gin.H{
		"new_difficulty": newDifficulty,
		"user_id":        req.UserID,
	})
}

func (h *EnhancedAdaptiveDifficultyHandler) GetEnhancedAnalytics(c *gin.Context) {
	userID := c.Query("user_id")
	if userID == "" {
		userID = "anonymous_" + strconv.FormatInt(time.Now().UnixNano()%1000000, 10)
	}

	analytics := h.enhancedService.GetEnhancedUserAnalytics(userID)

	response.Success(c, analytics)
}

type RecommendedDifficultyRequest struct {
	UserID          string  `json:"user_id" binding:"required" example:"user123"`
	SecurityLevel   float64 `json:"security_level" example:"0.7"`
	ExperienceLevel float64 `json:"experience_level" example:"0.5"`
}

func (h *EnhancedAdaptiveDifficultyHandler) GetRecommendedDifficulty(c *gin.Context) {
	var req RecommendedDifficultyRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	difficulty := h.enhancedService.GetRecommendedDifficulty(
		req.UserID,
		req.SecurityLevel,
		req.ExperienceLevel,
	)

	response.Success(c, gin.H{
		"recommended_difficulty": difficulty,
		"user_id":              req.UserID,
		"security_level":       req.SecurityLevel,
		"experience_level":     req.ExperienceLevel,
	})
}

func (h *EnhancedAdaptiveDifficultyHandler) GetGlobalStats(c *gin.Context) {
	stats := h.enhancedService.GetPersonalizationGlobalStats()

	response.Success(c, gin.H{
		"total_users":              stats.TotalUsers,
		"avg_success_rate":         stats.AvgSuccessRate,
		"most_popular_method":      stats.MostPopularMethod,
		"difficulty_distribution":   stats.DifficultyDistribution,
		"segment_distribution":      stats.SegmentDistribution,
	})
}

type TransitionRequest struct {
	UserID string `json:"user_id" binding:"required" example:"user123"`
	From   string `json:"from" binding:"required" example:"Easy"`
	To     string `json:"to" binding:"required" example:"Medium"`
}

func (h *EnhancedAdaptiveDifficultyHandler) InitiateTransition(c *gin.Context) {
	var req TransitionRequest

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "Invalid request")
		return
	}

	from := service.DifficultyLevel(req.From)
	to := service.DifficultyLevel(req.To)

	h.enhancedService.InitiateTransition(req.UserID, from, to)

	response.Success(c, gin.H{
		"message":     "Transition initiated",
		"user_id":     req.UserID,
		"from":        from,
		"to":          to,
		"transition_steps": h.enhancedService.GetTransitionSteps(),
	})
}

// Top-level functions

func GetEnhancedDifficulty(c *gin.Context) {
	GetEnhancedAdaptiveDifficultyHandler().GetEnhancedDifficulty(c)
}

func UpdateEnhancedResult(c *gin.Context) {
	GetEnhancedAdaptiveDifficultyHandler().UpdateEnhancedResult(c)
}

func GetEnhancedAnalytics(c *gin.Context) {
	GetEnhancedAdaptiveDifficultyHandler().GetEnhancedAnalytics(c)
}

func GetRecommendedDifficulty(c *gin.Context) {
	GetEnhancedAdaptiveDifficultyHandler().GetRecommendedDifficulty(c)
}

func GetEnhancedGlobalStats(c *gin.Context) {
	GetEnhancedAdaptiveDifficultyHandler().GetGlobalStats(c)
}

func InitiateDifficultyTransition(c *gin.Context) {
	GetEnhancedAdaptiveDifficultyHandler().InitiateTransition(c)
}
