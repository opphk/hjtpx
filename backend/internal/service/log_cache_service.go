package service

import (
	"context"
	"encoding/json"
	"fmt"
	"time"

	"github.com/hjtpx/hjtpx/pkg/models"
	"github.com/redis/go-redis/v9"
)

type LogCacheService struct {
	cachePrefix  string
	defaultTTL   time.Duration
	redisClient  *redis.Client
	enabled      bool
}

func NewLogCacheService(redisClient *redis.Client) *LogCacheService {
	return &LogCacheService{
		cachePrefix: "log_cache:",
		defaultTTL:   5 * time.Minute,
		redisClient: redisClient,
		enabled:     redisClient != nil,
	}
}

func (s *LogCacheService) buildCacheKey(params LogQueryParams) string {
	data, _ := json.Marshal(params)
	return fmt.Sprintf("%s%s", s.cachePrefix, string(data))
}

func (s *LogCacheService) Get(ctx context.Context, params LogQueryParams) (*LogListResult, error) {
	if !s.enabled {
		return nil, fmt.Errorf("cache disabled")
	}

	key := s.buildCacheKey(params)
	data, err := s.redisClient.Get(ctx, key).Bytes()
	if err != nil {
		return nil, err
	}

	var result LogListResult
	if err := json.Unmarshal(data, &result); err != nil {
		return nil, err
	}

	return &result, nil
}

func (s *LogCacheService) Set(ctx context.Context, params LogQueryParams, result *LogListResult) error {
	if !s.enabled {
		return nil
	}

	key := s.buildCacheKey(params)
	data, err := json.Marshal(result)
	if err != nil {
		return err
	}

	return s.redisClient.Set(ctx, key, data, s.defaultTTL).Err()
}

func (s *LogCacheService) Invalidate(ctx context.Context, pattern string) error {
	if !s.enabled {
		return nil
	}

	keys, err := s.redisClient.Keys(ctx, s.cachePrefix+pattern).Result()
	if err != nil {
		return err
	}

	if len(keys) > 0 {
		return s.redisClient.Del(ctx, keys...).Err()
	}

	return nil
}

func (s *LogCacheService) InvalidateAll(ctx context.Context) error {
	return s.Invalidate(ctx, "*")
}

type LogQueryOptimizer struct {
	enableIndexHints bool
	preferPartialIdx bool
}

func NewLogQueryOptimizer() *LogQueryOptimizer {
	return &LogQueryOptimizer{
		enableIndexHints: true,
		preferPartialIdx: true,
	}
}

func (o *LogQueryOptimizer) OptimizeParams(params *LogQueryParams) {
	if params.PageSize > 100 {
		params.PageSize = 100
	}

	if params.Page < 1 {
		params.Page = 1
	}
}

func (o *LogQueryOptimizer) GetIndexStrategy(params LogQueryParams) string {
	conditions := 0

	if params.ApplicationID > 0 {
		conditions++
	}
	if params.Status != "" {
		conditions++
	}
	if params.CaptchaType != "" {
		conditions++
	}
	if params.SessionID != "" {
		conditions++
	}
	if !params.StartDate.IsZero() {
		conditions++
	}
	if !params.EndDate.IsZero() {
		conditions++
	}
	if params.IPAddress != "" {
		conditions++
	}

	if conditions >= 3 {
		return "composite_index"
	} else if !params.StartDate.IsZero() && !params.EndDate.IsZero() {
		return "date_range_index"
	} else if params.ApplicationID > 0 {
		return "app_index"
	} else if params.Status != "" {
		return "status_index"
	}

	return "default_index"
}

type LogAggregationService struct{}

func NewLogAggregationService() *LogAggregationService {
	return &LogAggregationService{}
}

type AggregationResult struct {
	GroupBy   string                   `json:"group_by"`
	Results   []map[string]interface{} `json:"results"`
	Total     int64                    `json:"total"`
	StartTime time.Time                `json:"start_time"`
	EndTime   time.Time                `json:"end_time"`
}

func (s *LogAggregationService) AggregateByTime(logs []models.VerificationLog, interval string) map[string][]models.VerificationLog {
	result := make(map[string][]models.VerificationLog)

	for _, log := range logs {
		var key string
		switch interval {
		case "hour":
			key = log.CreatedAt.Format("2006-01-02 15")
		case "day":
			key = log.CreatedAt.Format("2006-01-02")
		case "week":
			year, week := log.CreatedAt.ISOWeek()
			key = fmt.Sprintf("%d-W%02d", year, week)
		case "month":
			key = log.CreatedAt.Format("2006-01")
		default:
			key = log.CreatedAt.Format("2006-01-02")
		}
		result[key] = append(result[key], log)
	}

	return result
}

func (s *LogAggregationService) AggregateByField(logs []models.VerificationLog, field string) map[string][]models.VerificationLog {
	result := make(map[string][]models.VerificationLog)

	for _, log := range logs {
		var key string
		switch field {
		case "status":
			key = log.Status
		case "captcha_type":
			key = log.CaptchaType
		case "ip_address":
			key = log.IPAddress
		case "application_id":
			key = fmt.Sprintf("%d", log.ApplicationID)
		default:
			key = "unknown"
		}
		result[key] = append(result[key], log)
	}

	return result
}

func (s *LogAggregationService) CalculateStatistics(logs []models.VerificationLog) map[string]interface{} {
	if len(logs) == 0 {
		return map[string]interface{}{
			"total_count":      0,
			"success_count":    0,
			"failed_count":    0,
			"avg_risk_score":   0.0,
			"max_risk_score":  0.0,
			"min_risk_score":  0.0,
			"avg_duration":     0.0,
			"max_duration":    0,
			"min_duration":    0,
			"unique_ips":      0,
			"unique_sessions": 0,
		}
	}

	var successCount, failedCount int64
	var totalRiskScore, maxRiskScore, minRiskScore float64 = 0, 0, 100
	var totalDuration, maxDuration, minDuration int64 = 0, 0, -1
	uniqueIPs := make(map[string]bool)
	uniqueSessions := make(map[string]bool)

	for _, log := range logs {
		if log.Status == "success" {
			successCount++
		} else if log.Status == "failed" {
			failedCount++
		}

		totalRiskScore += log.RiskScore
		if log.RiskScore > maxRiskScore {
			maxRiskScore = log.RiskScore
		}
		if log.RiskScore < minRiskScore {
			minRiskScore = log.RiskScore
		}

		totalDuration += log.Duration
		if log.Duration > maxDuration {
			maxDuration = log.Duration
		}
		if minDuration < 0 || log.Duration < minDuration {
			minDuration = log.Duration
		}

		uniqueIPs[log.IPAddress] = true
		uniqueSessions[log.SessionID] = true
	}

	count := float64(len(logs))

	return map[string]interface{}{
		"total_count":      len(logs),
		"success_count":    successCount,
		"failed_count":     failedCount,
		"success_rate":     float64(successCount) / count * 100,
		"avg_risk_score":   totalRiskScore / count,
		"max_risk_score":   maxRiskScore,
		"min_risk_score":   minRiskScore,
		"avg_duration":     float64(totalDuration) / count,
		"max_duration":     maxDuration,
		"min_duration":     minDuration,
		"unique_ips":       len(uniqueIPs),
		"unique_sessions":  len(uniqueSessions),
	}
}

type LogBatchService struct {
	batchSize    int
	maxBatchSize int
}

func NewLogBatchService() *LogBatchService {
	return &LogBatchService{
		batchSize:    100,
		maxBatchSize: 1000,
	}
}

func (s *LogBatchService) ProcessInBatches(logs []models.VerificationLog, processor func([]models.VerificationLog) error) error {
	total := len(logs)
	for i := 0; i < total; i += s.batchSize {
		end := i + s.batchSize
		if end > total {
			end = total
		}

		batch := logs[i:end]
		if err := processor(batch); err != nil {
			return err
		}
	}
	return nil
}

func (s *LogBatchService) SetBatchSize(size int) {
	if size > 0 && size <= s.maxBatchSize {
		s.batchSize = size
	}
}
