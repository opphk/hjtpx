package handler

import (
	"encoding/json"
	"strconv"

	"github.com/gin-gonic/gin"
	"github.com/hjtpx/hjtpx/internal/service"
	"github.com/hjtpx/hjtpx/pkg/response"
)

type SCIMHandler struct {
	manager *service.SCIMServiceManager
}

func NewSCIMHandler() *SCIMHandler {
	return &SCIMHandler{
		manager: service.GetSCIMManager(),
	}
}

func (h *SCIMHandler) GetServiceProviderConfig(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	config := scimService.GetServiceProviderConfig()
	response.Success(c, config)
}

func (h *SCIMHandler) GetResourceTypes(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	resourceTypes := scimService.GetResourceTypes()
	response.Success(c, resourceTypes)
}

func (h *SCIMHandler) GetSchemas(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	schemas := scimService.GetSchemas()
	response.Success(c, schemas)
}

func (h *SCIMHandler) ListUsers(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	filter := c.Query("filter")
	sortBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")
	
	startIndex, _ := strconv.Atoi(c.DefaultQuery("startIndex", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))

	result, err := scimService.ListUsers(filter, sortBy, sortOrder, startIndex, count)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *SCIMHandler) GetUser(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	userID := c.Param("id")
	user, err := scimService.GetUser(userID)
	if err != nil {
		response.NotFound(c)
		return
	}

	response.Success(c, user)
}

func (h *SCIMHandler) CreateUser(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	var user service.SCIMUser
	if err := c.ShouldBindJSON(&user); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	createdUser, err := scimService.CreateUser(&user)
	if err != nil {
		if err == service.ErrSCIMAlreadyExists {
			response.BadRequest(c, "user already exists")
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(201, createdUser)
}

func (h *SCIMHandler) UpdateUser(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	userID := c.Param("id")

	var user service.SCIMUser
	if err := c.ShouldBindJSON(&user); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updatedUser, err := scimService.UpdateUser(userID, &user)
	if err != nil {
		if err == service.ErrSCIMUserNotFound {
			response.NotFound(c)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, updatedUser)
}

func (h *SCIMHandler) DeleteUser(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	userID := c.Param("id")
	err = scimService.DeleteUser(userID)
	if err != nil {
		if err == service.ErrSCIMUserNotFound {
			response.NotFound(c)
			return
		}
		response.InternalServerError(c, err.Error())
		return
	}

	c.Status(204)
}

func (h *SCIMHandler) ListGroups(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	filter := c.Query("filter")
	sortBy := c.Query("sortBy")
	sortOrder := c.Query("sortOrder")
	
	startIndex, _ := strconv.Atoi(c.DefaultQuery("startIndex", "1"))
	count, _ := strconv.Atoi(c.DefaultQuery("count", "10"))

	result, err := scimService.ListGroups(filter, sortBy, sortOrder, startIndex, count)
	if err != nil {
		response.BadRequest(c, err.Error())
		return
	}

	response.Success(c, result)
}

func (h *SCIMHandler) GetGroup(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	groupID := c.Param("id")
	group, err := scimService.GetGroup(groupID)
	if err != nil {
		response.NotFound(c)
		return
	}

	response.Success(c, group)
}

func (h *SCIMHandler) CreateGroup(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	var group service.SCIMGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	createdGroup, err := scimService.CreateGroup(&group)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.JSON(201, createdGroup)
}

func (h *SCIMHandler) UpdateGroup(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	groupID := c.Param("id")

	var group service.SCIMGroup
	if err := c.ShouldBindJSON(&group); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	updatedGroup, err := scimService.UpdateGroup(groupID, &group)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, updatedGroup)
}

func (h *SCIMHandler) DeleteGroup(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	groupID := c.Param("id")
	err = scimService.DeleteGroup(groupID)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.Status(204)
}

func (h *SCIMHandler) PatchUser(c *gin.Context) {
	userID := c.Param("id")
	
	var patchRequest struct {
		Schemas    []string                 `json:"schemas"`
		Operations []map[string]interface{} `json:"Operations"`
	}

	if err := c.ShouldBindJSON(&patchRequest); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	user := &service.SCIMUser{}
	for _, op := range patchRequest.Operations {
		if op["op"] == "replace" {
			path := op["path"].(string)
			value := op["value"]

			switch path {
			case "userName":
				user.UserName = value.(string)
			case "displayName":
				user.DisplayName = value.(string)
			case "active":
				user.Active = value.(bool)
			case "emails":
				if emails, ok := value.([]interface{}); ok {
					var scimEmails []service.SCIMEmail
					for _, e := range emails {
						if emailMap, ok := e.(map[string]interface{}); ok {
							scimEmails = append(scimEmails, service.SCIMEmail{
								Value: emailMap["value"].(string),
								Type:  emailMap["type"].(string),
							})
						}
					}
					user.Emails = scimEmails
				}
			}
		}
	}

	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	updatedUser, err := scimService.UpdateUser(userID, user)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	response.Success(c, updatedUser)
}

func (h *SCIMHandler) RegisterTenant(c *gin.Context) {
	var req struct {
		TenantID uint   `json:"tenant_id" binding:"required"`
		BaseURL  string `json:"base_url" binding:"required"`
	}

	if err := c.ShouldBindJSON(&req); err != nil {
		response.BadRequest(c, "invalid request body")
		return
	}

	h.manager.RegisterTenant(req.TenantID, req.BaseURL)
	response.Success(c, "SCIM service registered for tenant")
}

func (h *SCIMHandler) UnregisterTenant(c *gin.Context) {
	tenantID, err := strconv.ParseUint(c.Param("tenant_id"), 10, 64)
	if err != nil {
		response.BadRequest(c, "invalid tenant ID")
		return
	}

	h.manager.UnregisterTenant(uint(tenantID))
	response.Success(c, "SCIM service unregistered for tenant")
}

func (h *SCIMHandler) ExportUsers(c *gin.Context) {
	tenantID := getTenantID(c)
	scimService, err := h.manager.GetService(tenantID)
	if err != nil {
		scimService = &service.SCIMService{
			baseURL: "https://api.hjtpx.com/scim/v2",
		}
	}

	format := c.DefaultQuery("format", "json")
	data, err := scimService.ExportUsers(format)
	if err != nil {
		response.InternalServerError(c, err.Error())
		return
	}

	c.Header("Content-Type", "application/json")
	c.Header("Content-Disposition", "attachment; filename=users.json")
	c.Data(200, "application/json", data)
}

func getTenantID(c *gin.Context) uint {
	tenantIDStr := c.GetHeader("X-Tenant-ID")
	if tenantIDStr == "" {
		tenantIDStr = c.Query("tenant_id")
	}
	if tenantIDStr == "" {
		return 1
	}
	tenantID, _ := strconv.ParseUint(tenantIDStr, 10, 64)
	return uint(tenantID)
}

func RegisterSCIMRoutes(r *gin.RouterGroup) {
	handler := NewSCIMHandler()

	scim := r.Group("/scim/v2")
	{
		scim.GET("/ServiceProviderConfig", handler.GetServiceProviderConfig)
		scim.GET("/ResourceTypes", handler.GetResourceTypes)
		scim.GET("/Schemas", handler.GetSchemas)

		users := scim.Group("/Users")
		{
			users.GET("", handler.ListUsers)
			users.POST("", handler.CreateUser)
			users.GET("/:id", handler.GetUser)
			users.PUT("/:id", handler.UpdateUser)
			users.PATCH("/:id", handler.PatchUser)
			users.DELETE("/:id", handler.DeleteUser)
		}

		groups := scim.Group("/Groups")
		{
			groups.GET("", handler.ListGroups)
			groups.POST("", handler.CreateGroup)
			groups.GET("/:id", handler.GetGroup)
			groups.PUT("/:id", handler.UpdateGroup)
			groups.PATCH("/:id", handler.UpdateGroup)
			groups.DELETE("/:id", handler.DeleteGroup)
		}

		scim.POST("/tenant/register", handler.RegisterTenant)
		scim.DELETE("/tenant/:tenant_id", handler.UnregisterTenant)
		scim.GET("/export/users", handler.ExportUsers)
	}
}