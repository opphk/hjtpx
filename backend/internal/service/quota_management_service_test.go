package service

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
)

func TestNewQuotaManagementService(t *testing.T) {
	service := NewQuotaManagementService()
	assert.NotNil(t, service)
	assert.NotNil(t, service.quotas)
}

func TestQuotaManagementService_CreateOrUpdateQuota(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-quota-key"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            1000,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	err := service.CreateOrUpdateQuota(ctx, key, config)
	assert.NoError(t, err)

	// 检查配额是否创建
	quota, ok := service.quotas[quotaPrefix+key]
	assert.True(t, ok)
	assert.Equal(t, config.Limit, quota.Limit)
	assert.Equal(t, config.Type, quota.Type)
}

func TestQuotaManagementService_GetQuota(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-get-quota"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            500,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	_ = service.CreateOrUpdateQuota(ctx, key, config)

	quota, err := service.GetQuota(ctx, key)
	assert.NoError(t, err)
	assert.NotNil(t, quota)
	assert.Equal(t, config.Limit, quota.Limit)
}

func TestQuotaManagementService_GetQuotaStatus(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-quota-status"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            1000,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	_ = service.CreateOrUpdateQuota(ctx, key, config)

	status, err := service.GetQuotaStatus(ctx, key)
	assert.NoError(t, err)
	assert.Equal(t, int64(0), status.Used)
	assert.Equal(t, config.Limit, status.Limit)
}

func TestQuotaManagementService_ConsumeQuota(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-consume-quota"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            10,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	_ = service.CreateOrUpdateQuota(ctx, key, config)

	// 消费配额
	for i := 0; i < 10; i++ {
		status, allowed, err := service.ConsumeQuota(ctx, key, 1)
		assert.NoError(t, err)
		assert.True(t, allowed)
		assert.Equal(t, int64(i+1), status.Used)
	}

	// 超过配额限制
	status, allowed, _ := service.ConsumeQuota(ctx, key, 1)
	assert.False(t, allowed)
	assert.Equal(t, int64(10), status.Used)
}

func TestQuotaManagementService_ResetQuota(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-reset-quota"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            100,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	_ = service.CreateOrUpdateQuota(ctx, key, config)

	// 消费一部分
	_, _, _ = service.ConsumeQuota(ctx, key, 50)

	// 重置配额
	err := service.ResetQuota(ctx, key)
	assert.NoError(t, err)

	// 检查是否重置
	status, _ := service.GetQuotaStatus(ctx, key)
	assert.Equal(t, int64(0), status.Used)
}

func TestQuotaManagementService_DeleteQuota(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-delete-quota"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            100,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	_ = service.CreateOrUpdateQuota(ctx, key, config)

	// 删除配额
	err := service.DeleteQuota(ctx, key)
	assert.NoError(t, err)

	// 检查是否删除
	_, ok := service.quotas[quotaPrefix+key]
	assert.False(t, ok)
}

func TestQuotaManagementService_ListQuotas(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()

	// 创建多个配额
	for i := 0; i < 5; i++ {
		key := "test-list-quota-" + string(rune(i))
		config := &QuotaConfig{
			Type:             QuotaTypeDaily,
			Limit:            int64(100 * (i + 1)),
			WarningThreshold: 80.0,
			HardLimit:        true,
		}
		_ = service.CreateOrUpdateQuota(ctx, key, config)
	}

	quotas, err := service.ListQuotas(ctx)
	assert.NoError(t, err)
	assert.GreaterOrEqual(t, len(quotas), 5)
}

func TestQuotaManagementService_CheckQuotaWarning(t *testing.T) {
	service := NewQuotaManagementService()
	ctx := context.Background()
	key := "test-warning-quota"

	config := &QuotaConfig{
		Type:             QuotaTypeDaily,
		Limit:            100,
		WarningThreshold: 80.0,
		HardLimit:        true,
	}

	_ = service.CreateOrUpdateQuota(ctx, key, config)

	// 消费到警告阈值以下
	for i := 0; i < 79; i++ {
		_, _, _ = service.ConsumeQuota(ctx, key, 1)
	}
	needsWarning, status, err := service.CheckQuotaWarning(ctx, key)
	assert.NoError(t, err)
	assert.False(t, needsWarning)
	assert.Less(t, status.Percentage, 80.0)

	// 消费到警告阈值以上
	_, _, _ = service.ConsumeQuota(ctx, key, 2)
	needsWarning, status, err = service.CheckQuotaWarning(ctx, key)
	assert.NoError(t, err)
	assert.True(t, needsWarning)
	assert.Greater(t, status.Percentage, 80.0)
}

func TestCalculateResetTime(t *testing.T) {
	// 测试各种配额类型的重置时间计算
	testCases := []struct {
		name      string
		quotaType QuotaType
	}{
		{"per-minute", QuotaTypePerMinute},
		{"hourly", QuotaTypeHourly},
		{"daily", QuotaTypeDaily},
		{"weekly", QuotaTypeWeekly},
		{"monthly", QuotaTypeMonthly},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			resetTime := calculateResetTime(tc.quotaType)
			assert.True(t, resetTime.After(time.Now()))
		})
	}
}

func TestUserQuotaKey(t *testing.T) {
	userID := uint(12345)
	resource := "api"
	quotaType := QuotaTypeDaily

	key := UserQuotaKey(userID, resource, quotaType)
	assert.Contains(t, key, "user:12345")
	assert.Contains(t, key, "api")
	assert.Contains(t, key, "daily")
}

func TestAppQuotaKey(t *testing.T) {
	appID := uint(67890)
	resource := "api"
	quotaType := QuotaTypeHourly

	key := AppQuotaKey(appID, resource, quotaType)
	assert.Contains(t, key, "app:67890")
	assert.Contains(t, key, "api")
	assert.Contains(t, key, "hourly")
}
