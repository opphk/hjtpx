package service

import (
	"math"
	"sync"
	"time"
)

// DifficultyLevel 难度级别
type DifficultyLevel string

const (
	DifficultyEasy   DifficultyLevel = "Easy"
	DifficultyMedium DifficultyLevel = "Medium"
	DifficultyHard   DifficultyLevel = "Hard"
	DifficultyExpert DifficultyLevel = "Expert"
)

// UserRiskProfile 用户风险档案
type UserRiskProfile struct {
	UserID        string
	RiskScore     float64 // 0-100，越高风险越大
	SuccessRate   float64 // 验证成功率
	AvgTime       float64 // 平均验证时间（秒）
	FailureCount  int     // 连续失败次数
	BehaviorFlags []string
	LastUpdated   time.Time
}

// DifficultyConfig 难度配置
type DifficultyConfig struct {
	EasyThreshold   float64
	MediumThreshold float64
	HardThreshold   float64
	ExpertThreshold float64
	FailureWeight   float64
	SuccessWeight   float64
	TimePenalty     float64
}

// AdaptiveDifficultyService 自适应难度服务
type AdaptiveDifficultyService struct {
	profiles map[string]*UserRiskProfile
	config   *DifficultyConfig
	mu       sync.RWMutex
}

// NewAdaptiveDifficultyService 创建自适应难度服务
func NewAdaptiveDifficultyService() *AdaptiveDifficultyService {
	return &AdaptiveDifficultyService{
		profiles: make(map[string]*UserRiskProfile),
		config: &DifficultyConfig{
			EasyThreshold:   20.0,
			MediumThreshold: 40.0,
			HardThreshold:   60.0,
			ExpertThreshold: 80.0,
			FailureWeight:   15.0,
			SuccessWeight:   -5.0,
			TimePenalty:     2.0,
		},
	}
}

// GetOrCreateProfile 获取或创建用户风险档案
func (s *AdaptiveDifficultyService) GetOrCreateProfile(userID string) *UserRiskProfile {
	s.mu.Lock()
	defer s.mu.Unlock()

	if profile, exists := s.profiles[userID]; exists {
		return profile
	}

	profile := &UserRiskProfile{
		UserID:      userID,
		RiskScore:   50.0, // 初始中等风险
		SuccessRate: 0.8,
		AvgTime:     5.0,
		LastUpdated: time.Now(),
	}
	s.profiles[userID] = profile
	return profile
}

// UpdateProfile 更新用户风险档案
func (s *AdaptiveDifficultyService) UpdateProfile(userID string, success bool, duration time.Duration) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.GetOrCreateProfile(userID)
	durationSec := duration.Seconds()

	// 更新平均时间
	profile.AvgTime = profile.AvgTime*0.9 + durationSec*0.1

	// 更新成功率
	if success {
		profile.SuccessRate = profile.SuccessRate*0.95 + 1.0*0.05
		profile.FailureCount = 0
		profile.RiskScore = math.Max(0, profile.RiskScore+s.config.SuccessWeight)
	} else {
		profile.SuccessRate = profile.SuccessRate*0.95 + 0.0*0.05
		profile.FailureCount++
		profile.RiskScore = math.Min(100, profile.RiskScore+s.config.FailureWeight*float64(profile.FailureCount))
	}

	// 时间惩罚（验证太快或太慢都可能可疑）
	if durationSec < 1.0 || durationSec > 30.0 {
		profile.RiskScore = math.Min(100, profile.RiskScore+s.config.TimePenalty)
	}

	profile.LastUpdated = time.Now()
}

// GetDifficulty 获取适合用户的难度级别
func (s *AdaptiveDifficultyService) GetDifficulty(userID string) DifficultyLevel {
	profile := s.GetOrCreateProfile(userID)
	score := profile.RiskScore

	switch {
	case score < s.config.EasyThreshold:
		return DifficultyEasy
	case score < s.config.MediumThreshold:
		return DifficultyMedium
	case score < s.config.HardThreshold:
		return DifficultyHard
	default:
		return DifficultyExpert
	}
}

// GetDifficultyForCaptcha 获取验证码的难度（考虑A/B测试）
func (s *AdaptiveDifficultyService) GetDifficultyForCaptcha(userID string, abTestEnabled bool) DifficultyLevel {
	baseDifficulty := s.GetDifficulty(userID)

	if abTestEnabled {
		// A/B测试：10%概率随机提升或降低一级难度
		if time.Now().UnixNano()%10 == 0 {
			switch baseDifficulty {
			case DifficultyEasy:
				return DifficultyMedium
			case DifficultyMedium:
				if time.Now().UnixNano()%2 == 0 {
					return DifficultyEasy
				}
				return DifficultyHard
			case DifficultyHard:
				if time.Now().UnixNano()%2 == 0 {
					return DifficultyMedium
				}
				return DifficultyExpert
			case DifficultyExpert:
				return DifficultyHard
			}
		}
	}

	return baseDifficulty
}

// UpdateConfig 更新配置
func (s *AdaptiveDifficultyService) UpdateConfig(config *DifficultyConfig) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.config = config
}

// GetConfig 获取当前配置
func (s *AdaptiveDifficultyService) GetConfig() *DifficultyConfig {
	s.mu.RLock()
	defer s.mu.RUnlock()
	return s.config
}

// GetAllProfiles 获取所有用户风险档案（用于管理端）
func (s *AdaptiveDifficultyService) GetAllProfiles() []*UserRiskProfile {
	s.mu.RLock()
	defer s.mu.RUnlock()

	profiles := make([]*UserRiskProfile, 0, len(s.profiles))
	for _, p := range s.profiles {
		profiles = append(profiles, p)
	}
	return profiles
}

// AddBehaviorFlag 添加行为标记
func (s *AdaptiveDifficultyService) AddBehaviorFlag(userID string, flag string) {
	s.mu.Lock()
	defer s.mu.Unlock()

	profile := s.GetOrCreateProfile(userID)
	for _, f := range profile.BehaviorFlags {
		if f == flag {
			return
		}
	}
	profile.BehaviorFlags = append(profile.BehaviorFlags, flag)
	profile.RiskScore = math.Min(100, profile.RiskScore+5.0)
}
