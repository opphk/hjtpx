package handler

import (
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type ABTestHandler struct {
	abTestService *service.ABTestService
}

func NewABTestHandler() *ABTestHandler {
	return &ABTestHandler{
		abTestService: service.NewABTestService(),
	}
}

func GetABTestHandler() *ABTestHandler {
	return NewABTestHandler()
}

type CreateABTestRequest struct {
	Name          string                 `json:"name" binding:"required,min=1,max=255"`
	Description   string                 `json:"description"`
	ApplicationID uint                   `json:"application_id" binding:"required"`
	Variants      []CreateVariantRequest `json:"variants" binding:"required,min=2"`
	Config        map[string]interface{} `json:"config"`
}

type CreateVariantRequest struct {
	Name           string                 `json:"name" binding:"required"`
	IsControl      bool                   `json:"is_control"`
	TrafficPercent int                    `json:"traffic_percent" binding:"required,min=0,max=100"`
	Config         map[string]interface{} `json:"config"`
	Description    string                 `json:"description"`
}

type UpdateABTestRequest struct {
	Name        *string                 `json:"name" binding:"omitempty,max=255"`
	Description *string                 `json:"description"`
	Variants    *[]CreateVariantRequest `json:"variants"`
	Config      *map[string]interface{} `json:"config"`
}

type ListABTestsQuery struct {
	Page          int    `form:"page,default=1"`
	PageSize      int    `form:"page_size,default=10"`
	Keyword       string `form:"keyword"`
	ApplicationID uint   `form:"application_id"`
	Status        string `form:"status"`
	SortField     string `form:"sort_field"`
	SortOrder     string `form:"sort_order"`
}

func ListABTests(c *gin.Context) {
	var query ListABTestsQuery
	if err := c.ShouldBindQuery(&query); err != nil {
		response.BadRequest(c, "invalid query parameters: "+err.Error())
		return
	}

	filter := &service.ListABTestsFilter{
		Page:          query.Page,
		PageSize:      query.PageSize,
		Keyword:       query.Keyword,
		ApplicationID: query.ApplicationID,
		Status:        query.Status,
		SortField:     query.SortField,
		SortOrder:     query.SortOrder,
	}

	result, err := service.NewABTestService().ListABTests(filter)
	if err != nil {
		response.InternalServerError(c, "failed to list ab tests: "+err.Error())
		return
	}

	response.Success(c, result)
}

func CreateABTest(c *gin.Context) {
	var req CreateABTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.CreateABTestInput{
		Name:          req.Name,
		Description:   req.Description,
		ApplicationID: req.ApplicationID,
		Config:        req.Config,
	}

	input.Variants = make([]service.CreateVariantInput, len(req.Variants))
	for i, v := range req.Variants {
		input.Variants[i] = service.CreateVariantInput{
			Name:           v.Name,
			IsControl:      v.IsControl,
			TrafficPercent: v.TrafficPercent,
			Config:         v.Config,
			Description:    v.Description,
		}
	}

	test, err := service.NewABTestService().CreateABTest(input)
	if err != nil {
		if err == service.ErrInvalidInput || err == service.ErrInvalidTraffic {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "failed to create ab test: "+err.Error())
		return
	}

	response.Success(c, test)
}

func GetABTest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	test, err := service.NewABTestService().GetABTestByID(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		response.InternalServerError(c, "failed to get ab test: "+err.Error())
		return
	}

	response.Success(c, test)
}

func UpdateABTest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	var req UpdateABTestRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	input := &service.UpdateABTestInput{
		Name:        req.Name,
		Description: req.Description,
		Config:      req.Config,
	}

	if req.Variants != nil {
		variants := make([]service.CreateVariantInput, len(*req.Variants))
		for i, v := range *req.Variants {
			variants[i] = service.CreateVariantInput{
				Name:           v.Name,
				IsControl:      v.IsControl,
				TrafficPercent: v.TrafficPercent,
				Config:         v.Config,
				Description:    v.Description,
			}
		}
		input.Variants = &variants
	}

	test, err := service.NewABTestService().UpdateABTest(uint(id), input)
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		if err == service.ErrInvalidTestStatus || err == service.ErrInvalidTraffic {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "failed to update ab test: "+err.Error())
		return
	}

	response.Success(c, test)
}

func DeleteABTest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	err = service.NewABTestService().DeleteABTest(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		response.InternalServerError(c, "failed to delete ab test: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "ab test deleted successfully"})
}

func StartABTest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	test, err := service.NewABTestService().StartABTest(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		if err == service.ErrInvalidTestStatus {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "failed to start ab test: "+err.Error())
		return
	}

	response.Success(c, test)
}

func StopABTest(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	test, err := service.NewABTestService().StopABTest(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		if err == service.ErrInvalidTestStatus {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "failed to stop ab test: "+err.Error())
		return
	}

	response.Success(c, test)
}

func GetActiveTests(c *gin.Context) {
	var applicationID uint
	appIDStr := c.Query("application_id")
	if appIDStr != "" {
		id, err := strconv.ParseUint(appIDStr, 10, 32)
		if err == nil {
			applicationID = uint(id)
		}
	}

	tests, err := service.NewABTestService().GetActiveTests(applicationID)
	if err != nil {
		response.InternalServerError(c, "failed to get active tests: "+err.Error())
		return
	}

	response.Success(c, tests)
}

func GetABTestSummary(c *gin.Context) {
	var applicationID uint
	appIDStr := c.Query("application_id")
	if appIDStr != "" {
		id, err := strconv.ParseUint(appIDStr, 10, 32)
		if err == nil {
			applicationID = uint(id)
		}
	}

	summary, err := service.NewABTestService().GetABTestSummary(applicationID)
	if err != nil {
		response.InternalServerError(c, "failed to get ab test summary: "+err.Error())
		return
	}

	response.Success(c, summary)
}

func GetTestReport(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	report, err := service.NewABTestService().GetTestReport(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		response.InternalServerError(c, "failed to get test report: "+err.Error())
		return
	}

	response.Success(c, report)
}

func AssignVariant(c *gin.Context) {
	var req service.AssignVariantRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	variant, err := service.NewABTestService().AssignVariant(&req)
	if err != nil {
		if err == service.ErrABTestNotFound || err == service.ErrVariantNotFound {
			response.NotFound(c, err.Error())
			return
		}
		if err == service.ErrInvalidTestStatus {
			response.BadRequest(c, err.Error())
			return
		}
		response.InternalServerError(c, "failed to assign variant: "+err.Error())
		return
	}

	response.Success(c, variant)
}

func TrackEvent(c *gin.Context) {
	var req service.TrackEventRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request parameters: "+err.Error())
		return
	}

	err := service.NewABTestService().TrackEvent(&req)
	if err != nil {
		response.InternalServerError(c, "failed to track event: "+err.Error())
		return
	}

	response.Success(c, gin.H{"message": "event tracked successfully"})
}

func CompareVariants(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	report, err := service.NewABTestService().CompareVariants(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		response.InternalServerError(c, "failed to compare variants: "+err.Error())
		return
	}

	response.Success(c, report)
}

func GetVariantAnalytics(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	variantIDStr := c.Param("variantId")
	variantID, err := strconv.ParseUint(variantIDStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid variant id")
		return
	}

	period := c.DefaultQuery("period", "7d")
	analytics, err := service.NewABTestService().GetVariantAnalytics(uint(id), uint(variantID), period)
	if err != nil {
		response.InternalServerError(c, "failed to get variant analytics: "+err.Error())
		return
	}

	response.Success(c, analytics)
}

func GetTestRecommendations(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid ab test id")
		return
	}

	recommendations, err := service.NewABTestService().GetTestRecommendations(uint(id))
	if err != nil {
		if err == service.ErrABTestNotFound {
			response.NotFound(c, "ab test not found")
			return
		}
		response.InternalServerError(c, "failed to get recommendations: "+err.Error())
		return
	}

	response.Success(c, recommendations)
}
