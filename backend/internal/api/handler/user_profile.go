package handler

import (
	"net/http"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type UserProfileHandler struct {
	userProfileService *service.UserProfileService
}

func NewUserProfileHandler() *UserProfileHandler {
	return &UserProfileHandler{
		userProfileService: service.NewUserProfileService(),
	}
}

func GetUserProfileHandler() *UserProfileHandler {
	return NewUserProfileHandler()
}

func GetUserProfile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	profile, err := service.NewUserProfileService().GenerateUserProfile(uint(id))
	if err != nil {
		response.InternalServerError(c, "failed to generate user profile: "+err.Error())
		return
	}

	response.Success(c, profile)
}

func GetUserProfileSummary(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	summary, err := service.NewUserProfileService().GetUserProfileSummary(uint(id))
	if err != nil {
		response.InternalServerError(c, "failed to get user profile summary: "+err.Error())
		return
	}

	response.Success(c, summary)
}

func ExportUserProfile(c *gin.Context) {
	idStr := c.Param("id")
	id, err := strconv.ParseUint(idStr, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid user id")
		return
	}

	format := c.DefaultQuery("format", "json")
	if format != "json" && format != "csv" {
		response.BadRequest(c, "unsupported format, use 'json' or 'csv'")
		return
	}

	data, err := service.NewUserProfileService().ExportUserProfile(uint(id), format)
	if err != nil {
		response.InternalServerError(c, "failed to export user profile: "+err.Error())
		return
	}

	filename := "user_profile_" + idStr
	if format == "json" {
		c.Header("Content-Type", "application/json")
		filename += ".json"
	} else {
		c.Header("Content-Type", "text/csv")
		filename += ".csv"
	}

	c.Header("Content-Disposition", "attachment; filename="+filename)
	c.Data(http.StatusOK, "application/octet-stream", data)
}

func ListUserProfiles(c *gin.Context) {
	page, _ := strconv.Atoi(c.DefaultQuery("page", "1"))
	pageSize, _ := strconv.Atoi(c.DefaultQuery("page_size", "10"))
	trustLevel := c.Query("trust_level")
	riskLevel := c.Query("risk_level")

	profiles, err := service.NewUserProfileService().ListUserProfiles(page, pageSize, trustLevel, riskLevel)
	if err != nil {
		response.InternalServerError(c, "failed to list user profiles: "+err.Error())
		return
	}

	response.Success(c, profiles)
}

func CompareUserProfiles(c *gin.Context) {
	id1Str := c.Param("id1")
	id2Str := c.Param("id2")

	id1, err := strconv.ParseUint(id1Str, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid first user id")
		return
	}

	id2, err := strconv.ParseUint(id2Str, 10, 32)
	if err != nil {
		response.BadRequest(c, "invalid second user id")
		return
	}

	comparison, err := service.NewUserProfileService().CompareUserProfiles(uint(id1), uint(id2))
	if err != nil {
		response.InternalServerError(c, "failed to compare user profiles: "+err.Error())
		return
	}

	response.Success(c, comparison)
}
