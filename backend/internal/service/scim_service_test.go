package service

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSCIMService_GetServiceProviderConfig(t *testing.T) {
	service := NewSCIMService()
	
	result := service.GetServiceProviderConfig()
	assert.NotNil(t, result)
	assert.Equal(t, "2.0", result["schemas"])
	assert.Contains(t, result, "documentationUri")
	assert.Contains(t, result, "patch")
	assert.Contains(t, result, "bulk")
	assert.Contains(t, result, "filter")
}

func TestSCIMService_CreateUser(t *testing.T) {
	service := NewSCIMService()
	
	user := map[string]interface{}{
		"schemas": []string{"urn:ietf:params:scim:schemas:core:2.0:User"},
		"userName": "testuser@example.com",
		"name": map[string]interface{}{
			"givenName": "Test",
			"familyName": "User",
		},
		"emails": []interface{}{
			map[string]interface{}{
				"value":   "testuser@example.com",
				"primary": true,
			},
		},
	}
	
	result, err := service.CreateUser(user)
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "id")
	assert.Contains(t, result, "schemas")
}

func TestSCIMService_GetUser(t *testing.T) {
	service := NewSCIMService()
	
	result, err := service.GetUser("test-id")
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "schemas")
}

func TestSCIMService_GetUsers(t *testing.T) {
	service := NewSCIMService()
	
	result, err := service.GetUsers(map[string]string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "Resources")
}

func TestSCIMService_DeleteUser(t *testing.T) {
	service := NewSCIMService()
	
	err := service.DeleteUser("test-id")
	assert.NoError(t, err)
}

func TestSCIMService_GetGroups(t *testing.T) {
	service := NewSCIMService()
	
	result, err := service.GetGroups(map[string]string{})
	assert.NoError(t, err)
	assert.NotNil(t, result)
	assert.Contains(t, result, "Resources")
}