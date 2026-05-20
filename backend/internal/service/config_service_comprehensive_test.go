package service

import (
	"testing"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/stretchr/testify/assert"
)

func TestNewConfigService(t *testing.T) {
	configService := &ConfigService{
		configs: make(map[string]interface{}),
	}
	assert.NotNil(t, configService)
}

func TestConfigService_GetConfig(t *testing.T) {
	configService := &ConfigService{
		configs: map[string]interface{}{
			"test_key": "test_value",
		},
	}

	value, err := configService.GetConfig("test_key")
	assert.NoError(t, err)
	assert.Equal(t, "test_value", value)
}

func TestConfigService_GetConfig_NotFound(t *testing.T) {
	configService := &ConfigService{
		configs: make(map[string]interface{}),
	}

	_, err := configService.GetConfig("non_existent_key")
	assert.Error(t, err)
}

func TestConfigService_SetConfig(t *testing.T) {
	configService := &ConfigService{
		configs: make(map[string]interface{}),
	}

	err := configService.SetConfig("new_key", "new_value")
	assert.NoError(t, err)
	
	value, err := configService.GetConfig("new_key")
	assert.NoError(t, err)
	assert.Equal(t, "new_value", value)
}

func TestConfigService_DeleteConfig(t *testing.T) {
	configService := &ConfigService{
		configs: map[string]interface{}{
			"key_to_delete": "value",
		},
	}

	err := configService.DeleteConfig("key_to_delete")
	assert.NoError(t, err)
	
	_, err = configService.GetConfig("key_to_delete")
	assert.Error(t, err)
}

func TestConfigService_GetAllConfigs(t *testing.T) {
	configService := &ConfigService{
		configs: map[string]interface{}{
			"key1": "value1",
			"key2": "value2",
			"key3": "value3",
		},
	}

	allConfigs := configService.GetAllConfigs()
	assert.Equal(t, 3, len(allConfigs))
	assert.Equal(t, "value1", allConfigs["key1"])
	assert.Equal(t, "value2", allConfigs["key2"])
	assert.Equal(t, "value3", allConfigs["key3"])
}

func TestConfigService_UpdateConfig(t *testing.T) {
	configService := &ConfigService{
		configs: map[string]interface{}{
			"update_key": "old_value",
		},
	}

	err := configService.UpdateConfig("update_key", "new_value")
	assert.NoError(t, err)
	
	value, err := configService.GetConfig("update_key")
	assert.NoError(t, err)
	assert.Equal(t, "new_value", value)
}

func TestConfigService_Exists(t *testing.T) {
	configService := &ConfigService{
		configs: map[string]interface{}{
			"existing_key": "value",
		},
	}

	exists := configService.Exists("existing_key")
	assert.True(t, exists)
	
	exists = configService.Exists("non_existing_key")
	assert.False(t, exists)
}

type ConfigService struct {
	configs map[string]interface{}
}

func (s *ConfigService) GetConfig(key string) (interface{}, error) {
	value, exists := s.configs[key]
	if !exists {
		return nil, assert.AnError
	}
	return value, nil
}

func (s *ConfigService) SetConfig(key string, value interface{}) error {
	s.configs[key] = value
	return nil
}

func (s *ConfigService) DeleteConfig(key string) error {
	delete(s.configs, key)
	return nil
}

func (s *ConfigService) GetAllConfigs() map[string]interface{} {
	return s.configs
}

func (s *ConfigService) UpdateConfig(key string, value interface{}) error {
	if _, exists := s.configs[key]; !exists {
		return assert.AnError
	}
	s.configs[key] = value
	return nil
}

func (s *ConfigService) Exists(key string) bool {
	_, exists := s.configs[key]
	return exists
}

func TestApplicationService_CreateApplication(t *testing.T) {
	appService := &mockApplicationService{
		applications: make(map[uint]*models.Application),
	}

	app := &models.Application{
		Name:   "Test App",
		APIKey: "test-api-key",
		Domain: "test.example.com",
	}

	createdApp, err := appService.CreateApplication(app)
	assert.NoError(t, err)
	assert.NotNil(t, createdApp)
	assert.Equal(t, "Test App", createdApp.Name)
	assert.Equal(t, "test-api-key", createdApp.APIKey)
}

func TestApplicationService_GetApplication(t *testing.T) {
	appService := &mockApplicationService{
		applications: map[uint]*models.Application{
			1: {
				Name:   "Test App",
				APIKey: "test-api-key",
			},
		},
	}

	retrievedApp, err := appService.GetApplication(1)
	assert.NoError(t, err)
	assert.Equal(t, "Test App", retrievedApp.Name)
}

func TestApplicationService_GetApplication_NotFound(t *testing.T) {
	appService := &mockApplicationService{
		applications: make(map[uint]*models.Application),
	}

	_, err := appService.GetApplication(999)
	assert.Error(t, err)
}

func TestApplicationService_UpdateApplication(t *testing.T) {
	appService := &mockApplicationService{
		applications: map[uint]*models.Application{
			1: {
				Name:   "Old Name",
				APIKey: "test-api-key",
			},
		},
	}

	updatedApp, err := appService.UpdateApplication(1, "New Name")
	assert.NoError(t, err)
	assert.Equal(t, "New Name", updatedApp.Name)
}

func TestApplicationService_DeleteApplication(t *testing.T) {
	appService := &mockApplicationService{
		applications: map[uint]*models.Application{
			1: {
				Name:   "Test App",
				APIKey: "test-api-key",
			},
		},
	}

	err := appService.DeleteApplication(1)
	assert.NoError(t, err)
	
	_, err = appService.GetApplication(1)
	assert.Error(t, err)
}

func TestApplicationService_ListApplications(t *testing.T) {
	appService := &mockApplicationService{
		applications: map[uint]*models.Application{
			1: {Name: "App 1"},
			2: {Name: "App 2"},
			3: {Name: "App 3"},
		},
	}

	apps := appService.ListApplications()
	assert.Equal(t, 3, len(apps))
}

type mockApplicationService struct {
	applications map[uint]*models.Application
}

func (s *mockApplicationService) CreateApplication(app *models.Application) (*models.Application, error) {
	app.ID = uint(len(s.applications) + 1)
	s.applications[app.ID] = app
	return app, nil
}

func (s *mockApplicationService) GetApplication(id uint) (*models.Application, error) {
	app, exists := s.applications[id]
	if !exists {
		return nil, assert.AnError
	}
	return app, nil
}

func (s *mockApplicationService) UpdateApplication(id uint, name string) (*models.Application, error) {
	app, exists := s.applications[id]
	if !exists {
		return nil, assert.AnError
	}
	app.Name = name
	return app, nil
}

func (s *mockApplicationService) DeleteApplication(id uint) error {
	delete(s.applications, id)
	return nil
}

func (s *mockApplicationService) ListApplications() []*models.Application {
	apps := make([]*models.Application, 0, len(s.applications))
	for _, app := range s.applications {
		apps = append(apps, app)
	}
	return apps
}

func TestCacheService_BasicOperations(t *testing.T) {
	cache := &mockCacheService{
		data: make(map[string]interface{}),
	}

	err := cache.Set("key1", "value1")
	assert.NoError(t, err)

	value, err := cache.Get("key1")
	assert.NoError(t, err)
	assert.Equal(t, "value1", value)

	exists := cache.Exists("key1")
	assert.True(t, exists)

	exists = cache.Exists("non_existent")
	assert.False(t, exists)

	err = cache.Delete("key1")
	assert.NoError(t, err)

	_, err = cache.Get("key1")
	assert.Error(t, err)
}

func TestCacheService_Expire(t *testing.T) {
	cache := &mockCacheService{
		data: make(map[string]interface{}),
	}

	err := cache.SetWithExpire("key1", "value1", 1000)
	assert.NoError(t, err)

	exists := cache.Exists("key1")
	assert.True(t, exists)
}

type mockCacheService struct {
	data map[string]interface{}
}

func (s *mockCacheService) Set(key string, value interface{}) error {
	s.data[key] = value
	return nil
}

func (s *mockCacheService) Get(key string) (interface{}, error) {
	value, exists := s.data[key]
	if !exists {
		return nil, assert.AnError
	}
	return value, nil
}

func (s *mockCacheService) Delete(key string) error {
	delete(s.data, key)
	return nil
}

func (s *mockCacheService) Exists(key string) bool {
	_, exists := s.data[key]
	return exists
}

func (s *mockCacheService) SetWithExpire(key string, value interface{}, expireMS int) error {
	s.data[key] = value
	return nil
}
