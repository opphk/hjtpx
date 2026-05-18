package service

import (
	"context"
	"fmt"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/pkg/redis"
)

// QuotaType 配额类型
type QuotaType string

const (
	QuotaTypeDaily    QuotaType = "daily"
	QuotaTypeWeekly   QuotaType = "weekly"
	QuotaTypeMonthly  QuotaType = "monthly"
	QuotaTypeHourly   QuotaType = "hourly"
	QuotaTypePerMinute QuotaType = "per_minute"
)

// QuotaStatus 配额状态
type QuotaStatus struct {
	Used        int64         // 已使用
	Limit       int64         // 限制
	Remaining   int64         // 剩余
	ResetAt     time.Time     // 重置时间
	Type        QuotaType     // 配额类型
	Percentage  float64       // 使用百分比
}

// QuotaConfig 配额配置
type QuotaConfig struct {
	Type         QuotaType    // 配额类型
	Limit        int64        // 配额限制
	WarningThreshold float64  // 警告阈值（百分比）
	HardLimit    bool         // 是否硬限制
}

// Quota 配额结构
type Quota struct {
	Key          string       // 配额键
	Type         QuotaType    // 配额类型
	Limit        int64        // 限制
	Used         int64        // 已使用
	CreatedAt    time.Time    // 创建时间
	UpdatedAt    time.Time    // 更新时间
	ResetAt      time.Time    // 下次重置时间
	WarningThreshold float64  // 警告阈值
	HardLimit    bool         // 是否硬限制
}

// QuotaManagementService 配额管理服务
type QuotaManagementService struct {
	quotas map[string]*Quota
	mu     sync.RWMutex
	redisEnabled bool
}

const (
	quotaPrefix = "quota:"
)

var defaultQuotaConfigs = map[string]QuotaConfig{
	"default_daily": {
		Type:            QuotaTypeDaily,
		Limit:           10000,
		WarningThreshold: 80.0,
		HardLimit:       true,
	},
	"default_hourly": {
		Type:            QuotaTypeHourly,
		Limit:           1000,
		WarningThreshold: 80.0,
		HardLimit:       true,
	},
}

// NewQuotaManagementService 创建配额管理服务
func NewQuotaManagementService() *QuotaManagementService {
	service := &QuotaManagementService{
		quotas:       make(map[string]*Quota),
		redisEnabled: redis.Client != nil,
	}
	go service.resetExpiredQuotas()
	return service
}

// calculateResetTime 计算重置时间
func calculateResetTime(quotaType QuotaType) time.Time {
	now := time.Now()
	switch quotaType {
	case QuotaTypePerMinute:
		return now.Add(time.Minute).Truncate(time.Minute)
	case QuotaTypeHourly:
		return now.Add(time.Hour).Truncate(time.Hour)
	case QuotaTypeDaily:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	case QuotaTypeWeekly:
		weekday := int(now.Weekday())
		if weekday == 0 {
			weekday = 7
		}
		daysUntilMonday := 8 - weekday
		return time.Date(now.Year(), now.Month(), now.Day()+daysUntilMonday, 0, 0, 0, 0, now.Location())
	case QuotaTypeMonthly:
		return time.Date(now.Year(), now.Month()+1, 1, 0, 0, 0, 0, now.Location())
	default:
		return time.Date(now.Year(), now.Month(), now.Day()+1, 0, 0, 0, 0, now.Location())
	}
}

// CreateOrUpdateQuota 创建或更新配额
func (s *QuotaManagementService) CreateOrUpdateQuota(
	ctx context.Context,
	key string,
	config *QuotaConfig,
) error {
	quotaKey := quotaPrefix + key
	now := time.Now()
	resetAt := calculateResetTime(config.Type)

	quota := &Quota{
		Key:              key,
		Type:             config.Type,
		Limit:            config.Limit,
		Used:             0,
		CreatedAt:        now,
		UpdatedAt:        now,
		ResetAt:          resetAt,
		WarningThreshold: config.WarningThreshold,
		HardLimit:        config.HardLimit,
	}

	s.mu.Lock()
	s.quotas[quotaKey] = quota
	s.mu.Unlock()

	if s.redisEnabled {
		pipe := redis.Client.Pipeline()
		pipe.HSet(ctx, quotaKey, "type", config.Type)
		pipe.HSet(ctx, quotaKey, "limit", config.Limit)
		pipe.HSet(ctx, quotaKey, "used", 0)
		pipe.HSet(ctx, quotaKey, "created_at", now.Unix())
		pipe.HSet(ctx, quotaKey, "updated_at", now.Unix())
		pipe.HSet(ctx, quotaKey, "reset_at", resetAt.Unix())
		pipe.HSet(ctx, quotaKey, "warning_threshold", config.WarningThreshold)
		pipe.HSet(ctx, quotaKey, "hard_limit", config.HardLimit)
		pipe.ExpireAt(ctx, quotaKey, resetAt)
		_, err := pipe.Exec(ctx)
		return err
	}
	return nil
}

// getQuotaFromMemory 从内存获取配额
func (s *QuotaManagementService) getQuotaFromMemory(key string) (*Quota, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	quota, exists := s.quotas[quotaPrefix+key]
	return quota, exists
}

// getQuotaFromRedis 从 Redis 获取配额
func (s *QuotaManagementService) getQuotaFromRedis(ctx context.Context, key string) (*Quota, error) {
	quotaKey := quotaPrefix + key
	result, err := redis.Client.HGetAll(ctx, quotaKey).Result()
	if err != nil {
		return nil, err
	}
	if len(result) == 0 {
		return nil, nil
	}

	quota := &Quota{
		Key: key,
	}
	if val, ok := result["type"]; ok {
		quota.Type = QuotaType(val)
	}
	if val, ok := result["limit"]; ok {
		fmt.Sscanf(val, "%d", &quota.Limit)
	}
	if val, ok := result["used"]; ok {
		fmt.Sscanf(val, "%d", &quota.Used)
	}
	if val, ok := result["created_at"]; ok {
		var ts int64
		fmt.Sscanf(val, "%d", &ts)
		quota.CreatedAt = time.Unix(ts, 0)
	}
	if val, ok := result["updated_at"]; ok {
		var ts int64
		fmt.Sscanf(val, "%d", &ts)
		quota.UpdatedAt = time.Unix(ts, 0)
	}
	if val, ok := result["reset_at"]; ok {
		var ts int64
		fmt.Sscanf(val, "%d", &ts)
		quota.ResetAt = time.Unix(ts, 0)
	}
	if val, ok := result["warning_threshold"]; ok {
		fmt.Sscanf(val, "%f", &quota.WarningThreshold)
	}
	if val, ok := result["hard_limit"]; ok {
		quota.HardLimit = val == "1" || val == "true"
	}

	return quota, nil
}

// GetQuota 获取配额
func (s *QuotaManagementService) GetQuota(ctx context.Context, key string) (*Quota, error) {
	if s.redisEnabled {
		return s.getQuotaFromRedis(ctx, key)
	}
	quota, _ := s.getQuotaFromMemory(key)
	return quota, nil
}

// GetQuotaStatus 获取配额状态
func (s *QuotaManagementService) GetQuotaStatus(ctx context.Context, key string) (*QuotaStatus, error) {
	quota, err := s.GetQuota(ctx, key)
	if err != nil {
		return nil, err
	}
	if quota == nil {
		config := defaultQuotaConfigs["default_daily"]
		return &QuotaStatus{
			Used:       0,
			Limit:      config.Limit,
			Remaining:  config.Limit,
			ResetAt:    calculateResetTime(config.Type),
			Type:       config.Type,
			Percentage: 0,
		}, nil
	}

	remaining := quota.Limit - quota.Used
	if remaining < 0 {
		remaining = 0
	}
	percentage := 0.0
	if quota.Limit > 0 {
		percentage = (float64(quota.Used) / float64(quota.Limit)) * 100
	}

	return &QuotaStatus{
		Used:       quota.Used,
		Limit:      quota.Limit,
		Remaining:  remaining,
		ResetAt:    quota.ResetAt,
		Type:       quota.Type,
		Percentage: percentage,
	}, nil
}

// ConsumeQuota 消费配额
func (s *QuotaManagementService) ConsumeQuota(
	ctx context.Context,
	key string,
	amount int64,
) (*QuotaStatus, bool, error) {
	if s.redisEnabled {
		return s.consumeQuotaRedis(ctx, key, amount)
	}
	return s.consumeQuotaMemory(ctx, key, amount)
}

// consumeQuotaMemory 内存版配额消费
func (s *QuotaManagementService) consumeQuotaMemory(
	ctx context.Context,
	key string,
	amount int64,
) (*QuotaStatus, bool, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	quotaKey := quotaPrefix + key
	quota, exists := s.quotas[quotaKey]
	if !exists {
		config := defaultQuotaConfigs["default_daily"]
		quota = &Quota{
			Key:              key,
			Type:             config.Type,
			Limit:            config.Limit,
			Used:             0,
			CreatedAt:        time.Now(),
			UpdatedAt:        time.Now(),
			ResetAt:          calculateResetTime(config.Type),
			WarningThreshold: config.WarningThreshold,
			HardLimit:        config.HardLimit,
		}
		s.quotas[quotaKey] = quota
	}

	// 检查是否需要重置
	if time.Now().After(quota.ResetAt) {
		quota.Used = 0
		quota.ResetAt = calculateResetTime(quota.Type)
		quota.UpdatedAt = time.Now()
	}

	quota.Used += amount
	quota.UpdatedAt = time.Now()

	allowed := true
	if quota.HardLimit && quota.Used > quota.Limit {
		allowed = false
	}

	status := &QuotaStatus{
		Used:       quota.Used,
		Limit:      quota.Limit,
		Remaining:  quota.Limit - quota.Used,
		ResetAt:    quota.ResetAt,
		Type:       quota.Type,
		Percentage: (float64(quota.Used) / float64(quota.Limit)) * 100,
	}
	if status.Remaining < 0 {
		status.Remaining = 0
	}

	return status, allowed, nil
}

// consumeQuotaRedis Redis 版配额消费
func (s *QuotaManagementService) consumeQuotaRedis(
	ctx context.Context,
	key string,
	amount int64,
) (*QuotaStatus, bool, error) {
	quotaKey := quotaPrefix + key
	now := time.Now().Unix()

	script := `
	local key = KEYS[1]
	local amount = tonumber(ARGV[1])
	local now = tonumber(ARGV[2])
	local defaultLimit = tonumber(ARGV[3])
	local defaultType = ARGV[4]
	local warningThreshold = tonumber(ARGV[5])
	local hardLimit = ARGV[6] == 'true'

	local data = redis.call('HGETALL', key)
	local exists = next(data) ~= nil

	local limit, used, resetAt, quotaType, warnThresh, isHardLimit
	
	if exists then
		for i = 1, #data, 2 do
			if data[i] == 'limit' then limit = tonumber(data[i+1]) end
			if data[i] == 'used' then used = tonumber(data[i+1]) end
			if data[i] == 'reset_at' then resetAt = tonumber(data[i+1]) end
			if data[i] == 'type' then quotaType = data[i+1] end
			if data[i] == 'warning_threshold' then warnThresh = tonumber(data[i+1]) end
			if data[i] == 'hard_limit' then isHardLimit = data[i+1] == '1' or data[i+1] == 'true' end
		end
	else
		limit = defaultLimit
		used = 0
		quotaType = defaultType
		warnThresh = warningThreshold
		isHardLimit = hardLimit
	end

	local newResetAt = resetAt
	if now > newResetAt or not exists then
		used = 0
		if quotaType == 'hourly' then
			newResetAt = math.floor(now / 3600) * 3600 + 3600
		elseif quotaType == 'daily' then
			local t = os.date('*t', now)
			newResetAt = os.time({year = t.year, month = t.month, day = t.day + 1, hour = 0, min = 0, sec = 0})
		else
			newResetAt = math.floor(now / 3600) * 3600 + 3600
		end
	end

	used = used + amount
	local allowed = true
	if isHardLimit and used > limit then
		allowed = false
	end

	local pipe = redis.pipeline()
	pipe.HSET(key, 'limit', limit)
	pipe.HSET(key, 'used', used)
	pipe.HSET(key, 'reset_at', newResetAt)
	pipe.HSET(key, 'type', quotaType)
	pipe.HSET(key, 'warning_threshold', warnThresh)
	pipe.HSET(key, 'hard_limit', isHardLimit and 1 or 0)
	pipe.EXPIREAT(key, newResetAt)
	pipe.exec()

	local remaining = limit - used
	if remaining < 0 then remaining = 0 end
	local percentage = 0
	if limit > 0 then percentage = (used / limit) * 100 end

	return {allowed, used, limit, remaining, newResetAt, quotaType, percentage}
`

	result, err := redis.Client.Eval(ctx, script, []string{quotaKey},
		amount, now, 10000, "daily", 80.0, true).Result()

	if err != nil {
		return nil, true, nil
	}

	values := result.([]interface{})
	allowed := values[0].(int64) == 1
	used := values[1].(int64)
	limit := values[2].(int64)
	remaining := values[3].(int64)
	resetAt := time.Unix(values[4].(int64), 0)
	quotaType := QuotaType(values[5].(string))
	percentage := values[6].(float64)

	return &QuotaStatus{
		Used:       used,
		Limit:      limit,
		Remaining:  remaining,
		ResetAt:    resetAt,
		Type:       quotaType,
		Percentage: percentage,
	}, allowed, nil
}

// ResetQuota 重置配额
func (s *QuotaManagementService) ResetQuota(ctx context.Context, key string) error {
	quotaKey := quotaPrefix + key

	s.mu.Lock()
	if quota, exists := s.quotas[quotaKey]; exists {
		quota.Used = 0
		quota.UpdatedAt = time.Now()
	}
	s.mu.Unlock()

	if s.redisEnabled {
		return redis.Client.HSet(ctx, quotaKey, "used", 0).Err()
	}
	return nil
}

// DeleteQuota 删除配额
func (s *QuotaManagementService) DeleteQuota(ctx context.Context, key string) error {
	quotaKey := quotaPrefix + key

	s.mu.Lock()
	delete(s.quotas, quotaKey)
	s.mu.Unlock()

	if s.redisEnabled {
		return redis.Client.Del(ctx, quotaKey).Err()
	}
	return nil
}

// ListQuotas 列出所有配额
func (s *QuotaManagementService) ListQuotas(ctx context.Context) ([]*Quota, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	quotas := make([]*Quota, 0, len(s.quotas))
	for _, q := range s.quotas {
		quotas = append(quotas, q)
	}
	return quotas, nil
}

// resetExpiredQuotas 重置过期的配额
func (s *QuotaManagementService) resetExpiredQuotas() {
	ticker := time.NewTicker(1 * time.Minute)
	defer ticker.Stop()

	for range ticker.C {
		s.mu.Lock()
		now := time.Now()
		for _, quota := range s.quotas {
			if now.After(quota.ResetAt) {
				quota.Used = 0
				quota.ResetAt = calculateResetTime(quota.Type)
				quota.UpdatedAt = now
			}
		}
		s.mu.Unlock()
	}
}

// CheckQuotaWarning 检查配额警告
func (s *QuotaManagementService) CheckQuotaWarning(ctx context.Context, key string) (bool, *QuotaStatus, error) {
	status, err := s.GetQuotaStatus(ctx, key)
	if err != nil {
		return false, nil, err
	}
	return status.Percentage >= 80, status, nil
}

// BatchConsumeQuota 批量消费配额
func (s *QuotaManagementService) BatchConsumeQuota(
	ctx context.Context,
	keys []string,
	amounts []int64,
) (map[string]*QuotaStatus, map[string]bool, error) {
	results := make(map[string]*QuotaStatus)
	allowedMap := make(map[string]bool)

	for i, key := range keys {
		amount := int64(1)
		if i < len(amounts) {
			amount = amounts[i]
		}
		status, allowed, err := s.ConsumeQuota(ctx, key, amount)
		if err != nil {
			results[key] = nil
			allowedMap[key] = true
		} else {
			results[key] = status
			allowedMap[key] = allowed
		}
	}

	return results, allowedMap, nil
}

// UserQuotaKey 生成用户配额键
func UserQuotaKey(userID uint, resource string, quotaType QuotaType) string {
	return fmt.Sprintf("user:%d:%s:%s", userID, resource, quotaType)
}

// AppQuotaKey 生成应用配额键
func AppQuotaKey(appID uint, resource string, quotaType QuotaType) string {
	return fmt.Sprintf("app:%d:%s:%s", appID, resource, quotaType)
}
