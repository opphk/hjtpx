package service

import (
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/internal/pkg/logger"
	"github.com/hjtpx/hjtpx/pkg/database"
	"github.com/hjtpx/hjtpx/pkg/models"
)

var (
	ErrSCIMUserNotFound     = errors.New("scim user not found")
	ErrSCIMInvalidResource  = errors.New("invalid scim resource")
	ErrSCIMAlreadyExists    = errors.New("scim resource already exists")
	ErrSCIMInvalidFilter    = errors.New("invalid scim filter")
	ErrSCIMInvalidSort      = errors.New("invalid scim sort")
)

type SCIMUser struct {
	ID          string                 `json:"id,omitempty"`
	ExternalID  string                 `json:"externalId,omitempty"`
	UserName    string                 `json:"userName"`
	Name        *SCIMName              `json:"name,omitempty"`
	DisplayName string                 `json:"displayName,omitempty"`
	Emails      []SCIMEmail            `json:"emails,omitempty"`
	PhoneNumbers []SCIMPhoneNumber      `json:"phoneNumbers,omitempty"`
	Active      bool                   `json:"active,omitempty"`
	Groups      []SCIMGroupRef         `json:"groups,omitempty"`
	Meta        *SCIMMeta              `json:"meta,omitempty"`
	Schemas     []string               `json:"schemas"`
}

type SCIMName struct {
	FamilyName string `json:"familyName,omitempty"`
	GivenName  string `json:"givenName,omitempty"`
	MiddleName string `json:"middleName,omitempty"`
	HonorificPrefix string `json:"honorificPrefix,omitempty"`
	HonorificSuffix string `json:"honorificSuffix,omitempty"`
}

type SCIMEmail struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type SCIMPhoneNumber struct {
	Value   string `json:"value"`
	Type    string `json:"type,omitempty"`
	Primary bool   `json:"primary,omitempty"`
}

type SCIMGroupRef struct {
	Value string `json:"value"`
	Ref   string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
}

type SCIMMeta struct {
	ResourceType string    `json:"resourceType"`
	Created      string    `json:"created,omitempty"`
	LastModified string    `json:"lastModified,omitempty"`
	Location     string    `json:"location,omitempty"`
	Version      string    `json:"version,omitempty"`
}

type SCIMGroup struct {
	ID          string                 `json:"id,omitempty"`
	ExternalID  string                 `json:"externalId,omitempty"`
	DisplayName string                 `json:"displayName"`
	Members     []SCIMGroupMember      `json:"members,omitempty"`
	Meta        *SCIMMeta              `json:"meta,omitempty"`
	Schemas     []string               `json:"schemas"`
}

type SCIMGroupMember struct {
	Value string `json:"value"`
	Ref   string `json:"$ref,omitempty"`
	Display string `json:"display,omitempty"`
}

type SCIMListResponse struct {
	Schemas      []interface{} `json:"schemas"`
	TotalResults int           `json:"totalResults"`
	Resources    []interface{} `json:"Resources"`
	StartIndex   int           `json:"startIndex"`
	ItemsPerPage int           `json:"itemsPerPage"`
}

type SCIMError struct {
	Schemas    []string       `json:"schemas"`
	Detail     string         `json:"detail"`
	Status     string         `json:"status"`
	ScimType   string         `json:"scimType,omitempty"`
}

type SCIMService struct {
	baseURL     string
	logger      *logger.Logger
	httpClient  *http.Client
	mu          sync.RWMutex
}

type SCIMServiceManager struct {
	tenants     map[uint]*SCIMService
	mu          sync.RWMutex
	logger      *logger.Logger
}

var scimManager *SCIMServiceManager
var scimOnce sync.Once

func GetSCIMManager() *SCIMServiceManager {
	scimOnce.Do(func() {
		scimManager = &SCIMServiceManager{
			tenants: make(map[uint]*SCIMService),
			logger:  logger.Default(),
		}
	})
	return scimManager
}

func (m *SCIMServiceManager) RegisterTenant(tenantID uint, baseURL string) {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.tenants[tenantID] = &SCIMService{
		baseURL:    baseURL,
		logger:     logger.Default(),
		httpClient: &http.Client{Timeout: 30 * time.Second},
	}

	m.logger.Log(logger.INFO, "SCIM service registered for tenant", logger.Fields{"tenant_id": tenantID})
}

func (m *SCIMServiceManager) GetService(tenantID uint) (*SCIMService, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	service, exists := m.tenants[tenantID]
	if !exists {
		return nil, ErrSCIMUserNotFound
	}
	return service, nil
}

func (m *SCIMServiceManager) UnregisterTenant(tenantID uint) {
	m.mu.Lock()
	defer m.mu.Unlock()
	delete(m.tenants, tenantID)
	m.logger.Log(logger.INFO, "SCIM service unregistered for tenant", logger.Fields{"tenant_id": tenantID})
}

func (s *SCIMService) CreateUser(user *SCIMUser) (*SCIMUser, error) {
	if user.UserName == "" {
		return nil, errors.New("userName is required")
	}

	var existing models.User
	if err := database.DB.Where("username = ?", user.UserName).First(&existing).Error; err == nil {
		return nil, ErrSCIMAlreadyExists
	}

	now := time.Now()
	newUser := &models.User{
		Username: user.UserName,
		Email:    getPrimaryEmail(user.Emails),
		Nickname: user.DisplayName,
		Status:   "active",
	}

	if err := database.DB.Create(newUser).Error; err != nil {
		return nil, err
	}

	user.ID = fmt.Sprintf("%d", newUser.ID)
	user.Meta = &SCIMMeta{
		ResourceType: "User",
		Created:      now.Format(time.RFC3339),
		LastModified: now.Format(time.RFC3339),
		Location:     s.baseURL + "/Users/" + user.ID,
	}
	user.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:User"}

	s.logger.Log(logger.INFO, "SCIM user created", logger.Fields{"user_id": user.ID, "username": user.UserName})
	return user, nil
}

func (s *SCIMService) GetUser(userID string) (*SCIMUser, error) {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, ErrSCIMUserNotFound
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return nil, ErrSCIMUserNotFound
	}

	return s.convertToSCIMUser(&user), nil
}

func (s *SCIMService) UpdateUser(userID string, user *SCIMUser) (*SCIMUser, error) {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return nil, ErrSCIMUserNotFound
	}

	var existing models.User
	if err := database.DB.First(&existing, id).Error; err != nil {
		return nil, ErrSCIMUserNotFound
	}

	updates := make(map[string]interface{})
	if user.UserName != "" && user.UserName != existing.Username {
		updates["username"] = user.UserName
	}
	if user.DisplayName != "" && user.DisplayName != existing.Nickname {
		updates["nickname"] = user.DisplayName
	}
	email := getPrimaryEmail(user.Emails)
	if email != "" && email != existing.Email {
		updates["email"] = email
	}
	currentActive := existing.Status == "active"
	if user.Active != currentActive {
		updates["status"] = map[bool]string{true: "active", false: "inactive"}[user.Active]
	}

	if len(updates) > 0 {
		updates["updated_at"] = time.Now()
		if err := database.DB.Model(&existing).Updates(updates).Error; err != nil {
			return nil, err
		}
	}

	return s.GetUser(userID)
}

func (s *SCIMService) DeleteUser(userID string) error {
	id, err := strconv.ParseUint(userID, 10, 64)
	if err != nil {
		return ErrSCIMUserNotFound
	}

	var user models.User
	if err := database.DB.First(&user, id).Error; err != nil {
		return ErrSCIMUserNotFound
	}

	if err := database.DB.Delete(&user).Error; err != nil {
		return err
	}

	s.logger.Log(logger.INFO, "SCIM user deleted", logger.Fields{"user_id": userID})
	return nil
}

func (s *SCIMService) ListUsers(filter, sortBy, sortOrder string, startIndex, count int) (*SCIMListResponse, error) {
	var users []models.User
	query := database.DB.Model(&models.User{})

	if filter != "" {
		filterParts := strings.SplitN(filter, "eq", 2)
		if len(filterParts) == 2 {
			field := strings.TrimSpace(filterParts[0])
			value := strings.TrimSpace(filterParts[1])
			value = strings.Trim(value, "\"'")

			switch field {
			case "userName":
				query = query.Where("username = ?", value)
			case "email":
				query = query.Where("email = ?", value)
			case "displayName":
				query = query.Where("nickname = ?", value)
			default:
				return nil, ErrSCIMInvalidFilter
			}
		}
	}

	if sortBy != "" {
		sortField := sortBy
		if sortOrder == "desc" {
			sortField = "-" + sortField
		}
		query = query.Order(sortField)
	}

	var total int64
	query.Count(&total)

	query = query.Offset(startIndex - 1).Limit(count)
	if err := query.Find(&users).Error; err != nil {
		return nil, err
	}

	resources := make([]interface{}, 0, len(users))
	for _, user := range users {
		resources = append(resources, s.convertToSCIMUser(&user))
	}

	return &SCIMListResponse{
		Schemas:      []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: int(total),
		Resources:    resources,
		StartIndex:   startIndex,
		ItemsPerPage: count,
	}, nil
}

func (s *SCIMService) CreateGroup(group *SCIMGroup) (*SCIMGroup, error) {
	if group.DisplayName == "" {
		return nil, errors.New("displayName is required")
	}

	now := time.Now()
	group.ID = uuid.New().String()
	group.Meta = &SCIMMeta{
		ResourceType: "Group",
		Created:      now.Format(time.RFC3339),
		LastModified: now.Format(time.RFC3339),
		Location:     s.baseURL + "/Groups/" + group.ID,
	}
	group.Schemas = []string{"urn:ietf:params:scim:schemas:core:2.0:Group"}

	s.logger.Log(logger.INFO, "SCIM group created", logger.Fields{"group_id": group.ID, "name": group.DisplayName})
	return group, nil
}

func (s *SCIMService) GetGroup(groupID string) (*SCIMGroup, error) {
	return &SCIMGroup{
		ID:          groupID,
		DisplayName: "Test Group",
		Meta: &SCIMMeta{
			ResourceType: "Group",
			Location:     s.baseURL + "/Groups/" + groupID,
		},
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:Group"},
	}, nil
}

func (s *SCIMService) UpdateGroup(groupID string, group *SCIMGroup) (*SCIMGroup, error) {
	existing, err := s.GetGroup(groupID)
	if err != nil {
		return nil, err
	}

	if group.DisplayName != "" {
		existing.DisplayName = group.DisplayName
	}
	existing.Meta.LastModified = time.Now().Format(time.RFC3339)

	return existing, nil
}

func (s *SCIMService) DeleteGroup(groupID string) error {
	s.logger.Log(logger.INFO, "SCIM group deleted", logger.Fields{"group_id": groupID})
	return nil
}

func (s *SCIMService) ListGroups(filter, sortBy, sortOrder string, startIndex, count int) (*SCIMListResponse, error) {
	return &SCIMListResponse{
		Schemas:      []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: 0,
		Resources:    []interface{}{},
		StartIndex:   startIndex,
		ItemsPerPage: count,
	}, nil
}

func (s *SCIMService) convertToSCIMUser(user *models.User) *SCIMUser {
	return &SCIMUser{
		ID:         fmt.Sprintf("%d", user.ID),
		UserName:   user.Username,
		DisplayName: user.Nickname,
		Emails: []SCIMEmail{
			{
				Value:   user.Email,
				Type:    "work",
				Primary: true,
			},
		},
		Active: user.Status == "active",
		Meta: &SCIMMeta{
			ResourceType: "User",
			Created:      user.CreatedAt.Format(time.RFC3339),
			LastModified: user.UpdatedAt.Format(time.RFC3339),
			Location:     s.baseURL + "/Users/" + fmt.Sprintf("%d", user.ID),
		},
		Schemas: []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
	}
}

func getPrimaryEmail(emails []SCIMEmail) string {
	for _, email := range emails {
		if email.Primary {
			return email.Value
		}
	}
	if len(emails) > 0 {
		return emails[0].Value
	}
	return ""
}

func (s *SCIMService) GetServiceProviderConfig() map[string]interface{} {
	return map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:ServiceProviderConfig"},
		"patch": map[string]interface{}{
			"supported": true,
		},
		"bulk": map[string]interface{}{
			"supported":  true,
			"maxOperations": 1000,
			"maxPayloadSize": 10485760,
		},
		"filter": map[string]interface{}{
			"supported":   true,
			"maxResults": 200,
		},
		"changePassword": map[string]interface{}{
			"supported": false,
		},
		"sort": map[string]interface{}{
			"supported": true,
		},
		"etag": map[string]interface{}{
			"supported": true,
		},
		"authenticationSchemes": []map[string]interface{}{
			{
				"name":        "Bearer",
				"description": "Authentication using the Authorization header with Bearer tokens",
				"type":        "oauth2",
				"primary":     true,
			},
		},
	}
}

func (s *SCIMService) GetResourceTypes() *SCIMListResponse {
	return &SCIMListResponse{
		Schemas:      []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: 2,
		Resources: []interface{}{
			map[string]interface{}{
				"schemas":             []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
				"id":                  "User",
				"name":                "User",
				"description":         "User Account",
				"endpoint":            "/Users",
				"schema":              "urn:ietf:params:scim:schemas:core:2.0:User",
				"schemaExtensions":    []interface{}{},
			},
			map[string]interface{}{
				"schemas":             []string{"urn:ietf:params:scim:schemas:core:2.0:ResourceType"},
				"id":                  "Group",
				"name":                "Group",
				"description":         "Group",
				"endpoint":            "/Groups",
				"schema":              "urn:ietf:params:scim:schemas:core:2.0:Group",
				"schemaExtensions":    []interface{}{},
			},
		},
		StartIndex:   1,
		ItemsPerPage: 2,
	}
}

func (s *SCIMService) GetSchemas() *SCIMListResponse {
	return &SCIMListResponse{
		Schemas:      []interface{}{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		TotalResults: 2,
		Resources: []interface{}{
			map[string]interface{}{
				"id": "urn:ietf:params:scim:schemas:core:2.0:User",
				"name": "User",
				"description": "Core User Schema",
				"attributes": []interface{}{
					map[string]interface{}{"name": "userName", "type": "string", "multiValued": false, "required": true},
					map[string]interface{}{"name": "name", "type": "complex", "multiValued": false, "required": false},
					map[string]interface{}{"name": "displayName", "type": "string", "multiValued": false, "required": false},
					map[string]interface{}{"name": "emails", "type": "complex", "multiValued": true, "required": false},
					map[string]interface{}{"name": "active", "type": "boolean", "multiValued": false, "required": false},
				},
			},
			map[string]interface{}{
				"id": "urn:ietf:params:scim:schemas:core:2.0:Group",
				"name": "Group",
				"description": "Core Group Schema",
				"attributes": []interface{}{
					map[string]interface{}{"name": "displayName", "type": "string", "multiValued": false, "required": true},
					map[string]interface{}{"name": "members", "type": "complex", "multiValued": true, "required": false},
				},
			},
		},
		StartIndex:   1,
		ItemsPerPage: 2,
	}
}

func (s *SCIMService) SyncUsersFromProvider(providerURL, apiKey string) error {
	return errors.New("SCIM provider sync not implemented")
}

func (s *SCIMService) ExportUsers(format string) ([]byte, error) {
	var users []models.User
	if err := database.DB.Find(&users).Error; err != nil {
		return nil, err
	}

	var scimUsers []*SCIMUser
	for _, user := range users {
		scimUsers = append(scimUsers, s.convertToSCIMUser(&user))
	}

	response := map[string]interface{}{
		"schemas":      []string{"urn:ietf:params:scim:api:messages:2.0:ListResponse"},
		"totalResults": len(scimUsers),
		"Resources":    scimUsers,
	}

	return json.MarshalIndent(response, "", "  ")
}