package service

import (
	"context"
	"fmt"
	"math/rand"
	"sort"
	"sync"
	"time"
)

// 使用已定义的 RiskLevel 类型 (int)

// CaptchaCapability 验证码能力
type CaptchaCapability struct {
	Type                 CaptchaType
	Name                 string
	SecurityLevel        int     // 安全性级别 1-10
	UserExperienceScore  float64 // 用户体验评分 0-1
	Complexity           int     // 复杂度 1-5
	SupportedDifficulties []DifficultyLevel
	RequiresInteraction  bool
	AccessibilitySupport bool
	PlatformSupport      []string
}

// CaptchaSelectionResult 验证码选择结果
type CaptchaSelectionResult struct {
	CaptchaType        CaptchaType
	Name               string
	Reason             string
	Confidence         float64
	ExpectedSuccessRate float64
	AlternativeTypes   []CaptchaType
}

// SelectionContext 选择上下文
type SelectionContext struct {
	UserID          string
	RiskScore       float64
	DeviceType      string
	Platform        string
	NetworkQuality  float64
	AccessibilityRequired bool
	PreviousAttempts int
	SuccessHistory   []bool
	TimeConstraints  bool
	GeoLocation     string
}

// SmartCaptchaSelector 智能验证码选择器
type SmartCaptchaSelector struct {
	capabilities       map[CaptchaType]*CaptchaCapability
	riskProfiles       map[string]*CaptchaRiskProfile
	selectionHistory   map[string][]*SelectionRecord
	mu                 sync.RWMutex
	learningRate       float64
}

// CaptchaRiskProfile 验证码风险配置文件
type CaptchaRiskProfile struct {
	RiskLevel           RiskLevel
	MinSecurityLevel    int
	RecommendedTypes    []CaptchaType
	BlockedTypes        []CaptchaType
	MaxComplexity       int
}

// SelectionRecord 选择记录
type SelectionRecord struct {
	UserID            string
	Timestamp         time.Time
	CaptchaType       CaptchaType
	RiskScore         float64
	Success           bool
	ResponseTime      time.Duration
	UserFeedback      float64
}

// NewSmartCaptchaSelector 创建智能验证码选择器
func NewSmartCaptchaSelector() *SmartCaptchaSelector {
	selector := &SmartCaptchaSelector{
		capabilities:     make(map[CaptchaType]*CaptchaCapability),
		riskProfiles:     make(map[string]*CaptchaRiskProfile),
		selectionHistory: make(map[string][]*SelectionRecord),
		learningRate:     0.1,
	}
	selector.initializeCapabilities()
	selector.initializeRiskProfiles()
	return selector
}

// 初始化验证码能力配置
func (s *SmartCaptchaSelector) initializeCapabilities() {
	s.capabilities[CaptchaTypeSlider] = &CaptchaCapability{
		Type:                 CaptchaTypeSlider,
		Name:                 "滑动验证码",
		SecurityLevel:        3,
		UserExperienceScore:  0.9,
		Complexity:           1,
		SupportedDifficulties: []DifficultyLevel{DifficultyEasy, DifficultyMedium, DifficultyHard},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypeClick] = &CaptchaCapability{
		Type:                 CaptchaTypeClick,
		Name:                 "点击验证码",
		SecurityLevel:        5,
		UserExperienceScore:  0.75,
		Complexity:           2,
		SupportedDifficulties: []DifficultyLevel{DifficultyEasy, DifficultyMedium, DifficultyHard, DifficultyExpert},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypeIcon] = &CaptchaCapability{
		Type:                 CaptchaTypeIcon,
		Name:                 "图标验证码",
		SecurityLevel:        4,
		UserExperienceScore:  0.85,
		Complexity:           2,
		SupportedDifficulties: []DifficultyLevel{DifficultyEasy, DifficultyMedium},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaType3D] = &CaptchaCapability{
		Type:                 CaptchaType3D,
		Name:                 "3D验证码",
		SecurityLevel:        9,
		UserExperienceScore:  0.6,
		Complexity:           5,
		SupportedDifficulties: []DifficultyLevel{DifficultyHard, DifficultyExpert},
		RequiresInteraction:  true,
		AccessibilitySupport: false,
		PlatformSupport:      []string{"web", "desktop"},
	}

	s.capabilities[CaptchaTypeSemantic] = &CaptchaCapability{
		Type:                 CaptchaTypeSemantic,
		Name:                 "语义验证码",
		SecurityLevel:        8,
		UserExperienceScore:  0.55,
		Complexity:           5,
		SupportedDifficulties: []DifficultyLevel{DifficultyHard, DifficultyExpert},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypeEmoji] = &CaptchaCapability{
		Type:                 CaptchaTypeEmoji,
		Name:                 "表情验证码",
		SecurityLevel:        4,
		UserExperienceScore:  0.88,
		Complexity:           2,
		SupportedDifficulties: []DifficultyLevel{DifficultyEasy, DifficultyMedium},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypeVoice] = &CaptchaCapability{
		Type:                 CaptchaTypeVoice,
		Name:                 "语音验证码",
		SecurityLevel:        7,
		UserExperienceScore:  0.7,
		Complexity:           3,
		SupportedDifficulties: []DifficultyLevel{DifficultyMedium, DifficultyHard},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypeMath] = &CaptchaCapability{
		Type:                 CaptchaTypeMath,
		Name:                 "数学验证码",
		SecurityLevel:        6,
		UserExperienceScore:  0.65,
		Complexity:           3,
		SupportedDifficulties: []DifficultyLevel{DifficultyMedium, DifficultyHard, DifficultyExpert},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypeColor] = &CaptchaCapability{
		Type:                 CaptchaTypeColor,
		Name:                 "颜色验证码",
		SecurityLevel:        3,
		UserExperienceScore:  0.82,
		Complexity:           1,
		SupportedDifficulties: []DifficultyLevel{DifficultyEasy, DifficultyMedium},
		RequiresInteraction:  true,
		AccessibilitySupport: false,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}

	s.capabilities[CaptchaTypePhrase] = &CaptchaCapability{
		Type:                 CaptchaTypePhrase,
		Name:                 "词组验证码",
		SecurityLevel:        5,
		UserExperienceScore:  0.72,
		Complexity:           3,
		SupportedDifficulties: []DifficultyLevel{DifficultyMedium, DifficultyHard},
		RequiresInteraction:  true,
		AccessibilitySupport: true,
		PlatformSupport:      []string{"web", "mobile", "desktop"},
	}
}

// 初始化风险配置文件
func (s *SmartCaptchaSelector) initializeRiskProfiles() {
	s.riskProfiles[string(RiskLevelCritical)] = &CaptchaRiskProfile{
		RiskLevel:        RiskLevelCritical,
		MinSecurityLevel: 8,
		RecommendedTypes: []CaptchaType{
			CaptchaType3D,
			CaptchaTypeSemantic,
			CaptchaTypeVoice,
			CaptchaTypeMath,
		},
		BlockedTypes:  []CaptchaType{CaptchaTypeSlider, CaptchaTypeColor},
		MaxComplexity: 5,
	}

	s.riskProfiles[string(RiskLevelHigh)] = &CaptchaRiskProfile{
		RiskLevel:        RiskLevelHigh,
		MinSecurityLevel: 6,
		RecommendedTypes: []CaptchaType{
			CaptchaType3D,
			CaptchaTypeSemantic,
			CaptchaTypeMath,
			CaptchaTypeVoice,
			CaptchaTypeClick,
		},
		BlockedTypes:  []CaptchaType{CaptchaTypeSlider},
		MaxComplexity: 4,
	}

	s.riskProfiles[string(RiskLevelMedium)] = &CaptchaRiskProfile{
		RiskLevel:        RiskLevelMedium,
		MinSecurityLevel: 4,
		RecommendedTypes: []CaptchaType{
			CaptchaTypeClick,
			CaptchaTypeIcon,
			CaptchaTypeEmoji,
			CaptchaTypeVoice,
			CaptchaTypeMath,
			CaptchaTypePhrase,
		},
		BlockedTypes:  []CaptchaType{},
		MaxComplexity: 3,
	}

	s.riskProfiles[string(RiskLevelLow)] = &CaptchaRiskProfile{
		RiskLevel:        RiskLevelLow,
		MinSecurityLevel: 2,
		RecommendedTypes: []CaptchaType{
			CaptchaTypeSlider,
			CaptchaTypeIcon,
			CaptchaTypeEmoji,
			CaptchaTypeColor,
		},
		BlockedTypes:  []CaptchaType{CaptchaType3D, CaptchaTypeSemantic},
		MaxComplexity: 2,
	}
}

// SelectCaptcha 选择验证码
func (s *SmartCaptchaSelector) SelectCaptcha(ctx context.Context, context *SelectionContext) (*CaptchaSelectionResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	riskLevel := s.determineRiskLevel(context.RiskScore)
	riskProfile, exists := s.riskProfiles[string(riskLevel)]
	if !exists {
		return nil, fmt.Errorf("unknown risk level: %s", riskLevel)
	}

	candidates := s.filterCapabilities(riskProfile, context)
	
	if len(candidates) == 0 {
		return nil, fmt.Errorf("no captcha type available for the given constraints")
	}

	scores := s.scoreCandidates(candidates, context, riskProfile)
	
	bestType := s.selectBestCandidate(scores, context)
	
	capability := s.capabilities[bestType]
	confidence := scores[bestType]
	
	alternatives := s.getAlternatives(candidates, bestType, scores)
	
	return &CaptchaSelectionResult{
		CaptchaType:        bestType,
		Name:               capability.Name,
		Reason:             s.generateReason(bestType, riskLevel, context),
		Confidence:         confidence,
		ExpectedSuccessRate: s.calculateExpectedSuccessRate(bestType, context),
		AlternativeTypes:   alternatives,
	}, nil
}

// 确定风险等级
func (s *SmartCaptchaSelector) determineRiskLevel(riskScore float64) RiskLevel {
	switch {
	case riskScore >= 80:
		return RiskLevelCritical
	case riskScore >= 60:
		return RiskLevelHigh
	case riskScore >= 40:
		return RiskLevelMedium
	default:
		return RiskLevelLow
	}
}

// 过滤符合条件的验证码类型
func (s *SmartCaptchaSelector) filterCapabilities(riskProfile *CaptchaRiskProfile, context *SelectionContext) []*CaptchaCapability {
	var candidates []*CaptchaCapability
	
	for _, capability := range s.capabilities {
		if s.isBlocked(riskProfile, capability.Type) {
			continue
		}
		
		if capability.SecurityLevel < riskProfile.MinSecurityLevel {
			continue
		}
		
		if capability.Complexity > riskProfile.MaxComplexity {
			continue
		}
		
		if context.AccessibilityRequired && !capability.AccessibilitySupport {
			continue
		}
		
		if !s.isPlatformSupported(capability, context.Platform) {
			continue
		}
		
		candidates = append(candidates, capability)
	}
	
	return candidates
}

// 检查是否被阻止
func (s *SmartCaptchaSelector) isBlocked(riskProfile *CaptchaRiskProfile, captchaType CaptchaType) bool {
	for _, blocked := range riskProfile.BlockedTypes {
		if blocked == captchaType {
			return true
		}
	}
	return false
}

// 检查平台支持
func (s *SmartCaptchaSelector) isPlatformSupported(capability *CaptchaCapability, platform string) bool {
	if platform == "" {
		return true
	}
	for _, supported := range capability.PlatformSupport {
		if supported == platform {
			return true
		}
	}
	return false
}

// 评分候选验证码
func (s *SmartCaptchaSelector) scoreCandidates(candidates []*CaptchaCapability, context *SelectionContext, riskProfile *CaptchaRiskProfile) map[CaptchaType]float64 {
	scores := make(map[CaptchaType]float64)
	
	for _, capability := range candidates {
		score := float64(capability.SecurityLevel) * 0.4
		score += capability.UserExperienceScore * 30
		score += float64(6-capability.Complexity) * 2
		
		if context.PreviousAttempts > 2 {
			successRate := s.calculateSuccessRate(context.SuccessHistory)
			if successRate < 0.5 {
				score += float64(6-capability.Complexity) * 3
			}
		}
		
		if context.NetworkQuality < 0.5 {
			if capability.Complexity <= 2 {
				score += 10
			}
		}
		
		if context.TimeConstraints {
			score += float64(6-capability.Complexity) * 2
		}
		
		historyScore := s.getHistoricalScore(capability.Type, context.UserID)
		score += historyScore * 10
		
		scores[capability.Type] = score
	}
	
	return scores
}

// 计算成功率
func (s *SmartCaptchaSelector) calculateSuccessRate(history []bool) float64 {
	if len(history) == 0 {
		return 0.5
	}
	count := 0
	for _, success := range history {
		if success {
			count++
		}
	}
	return float64(count) / float64(len(history))
}

// 获取历史评分
func (s *SmartCaptchaSelector) getHistoricalScore(captchaType CaptchaType, userID string) float64 {
	if userID == "" {
		return 0.5
	}
	
	records, exists := s.selectionHistory[userID]
	if !exists || len(records) == 0 {
		return 0.5
	}
	
	count := 0
	totalScore := 0.0
	for _, record := range records {
		if record.CaptchaType == captchaType {
			count++
			if record.Success {
				totalScore += 1.0
			} else {
				totalScore += 0.3
			}
		}
	}
	
	if count == 0 {
		return 0.5
	}
	
	return totalScore / float64(count)
}

// 选择最佳候选
func (s *SmartCaptchaSelector) selectBestCandidate(scores map[CaptchaType]float64, context *SelectionContext) CaptchaType {
	type scoredType struct {
		captchaType CaptchaType
		score       float64
	}
	
	var scoredTypes []scoredType
	for ct, score := range scores {
		scoredTypes = append(scoredTypes, scoredType{ct, score})
	}
	
	sort.Slice(scoredTypes, func(i, j int) bool {
		return scoredTypes[i].score > scoredTypes[j].score
	})
	
	if len(scoredTypes) == 0 {
		return CaptchaTypeSlider
	}
	
	if context.RiskScore < 20 && rand.Float64() < 0.2 {
		return scoredTypes[min(rand.Intn(len(scoredTypes)), len(scoredTypes)-1)].captchaType
	}
	
	return scoredTypes[0].captchaType
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

// 获取备选类型
func (s *SmartCaptchaSelector) getAlternatives(candidates []*CaptchaCapability, bestType CaptchaType, scores map[CaptchaType]float64) []CaptchaType {
	var alternatives []CaptchaType
	
	type scoredType struct {
		captchaType CaptchaType
		score       float64
	}
	
	var scoredTypes []scoredType
	for _, cap := range candidates {
		if cap.Type != bestType {
			scoredTypes = append(scoredTypes, scoredType{cap.Type, scores[cap.Type]})
		}
	}
	
	sort.Slice(scoredTypes, func(i, j int) bool {
		return scoredTypes[i].score > scoredTypes[j].score
	})
	
	for i, st := range scoredTypes {
		if i >= 3 {
			break
		}
		alternatives = append(alternatives, st.captchaType)
	}
	
	return alternatives
}

// 生成选择理由
func (s *SmartCaptchaSelector) generateReason(captchaType CaptchaType, riskLevel RiskLevel, context *SelectionContext) string {
	capability := s.capabilities[captchaType]
	
	reason := fmt.Sprintf("选择%s作为验证方式。", capability.Name)
	
	switch riskLevel {
	case RiskLevelCritical:
		reason += " 检测到高风险环境，需要最高级别的安全保护。"
	case RiskLevelHigh:
		reason += " 检测到较高风险，需要增强安全验证。"
	case RiskLevelMedium:
		reason += " 中等风险评估，平衡安全性与用户体验。"
	default:
		reason += " 低风险环境，优先考虑用户体验。"
	}
	
	if context.PreviousAttempts > 2 {
		successRate := s.calculateSuccessRate(context.SuccessHistory)
		if successRate < 0.5 {
			reason += fmt.Sprintf(" 用户之前验证失败较多(%d次失败)，选择较简单的验证方式。", context.PreviousAttempts)
		}
	}
	
	return reason
}

// 计算预期成功率
func (s *SmartCaptchaSelector) calculateExpectedSuccessRate(captchaType CaptchaType, context *SelectionContext) float64 {
	capability := s.capabilities[captchaType]
	
	baseRate := 0.9
	
	switch capability.Complexity {
	case 1:
		baseRate = 0.95
	case 2:
		baseRate = 0.92
	case 3:
		baseRate = 0.85
	case 4:
		baseRate = 0.75
	case 5:
		baseRate = 0.65
	}
	
	if context.NetworkQuality < 0.5 {
		baseRate *= 0.9
	}
	
	if context.PreviousAttempts > 3 {
		successRate := s.calculateSuccessRate(context.SuccessHistory)
		baseRate = baseRate*0.7 + successRate*0.3
	}
	
	return baseRate
}

// RecordSelection 记录选择结果
func (s *SmartCaptchaSelector) RecordSelection(ctx context.Context, record *SelectionRecord) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if record.UserID != "" {
		s.selectionHistory[record.UserID] = append(s.selectionHistory[record.UserID], record)
		
		if len(s.selectionHistory[record.UserID]) > 100 {
			s.selectionHistory[record.UserID] = s.selectionHistory[record.UserID][len(s.selectionHistory[record.UserID])-100:]
		}
	}
	
	s.updateCapabilityPerformance(record)
}

// 更新能力性能
func (s *SmartCaptchaSelector) updateCapabilityPerformance(record *SelectionRecord) {
	capability := s.capabilities[record.CaptchaType]
	if capability == nil {
		return
	}
	
	adjustment := s.learningRate
	if record.Success {
		capability.UserExperienceScore = capability.UserExperienceScore*(1-adjustment) + 1.0*adjustment
	} else {
		capability.UserExperienceScore = capability.UserExperienceScore*(1-adjustment) + 0.3*adjustment
	}
	
	capability.UserExperienceScore = max(0.1, minFloat(1.0, capability.UserExperienceScore))
}

func max(a, b float64) float64 {
	if a > b {
		return a
	}
	return b
}

func minFloat(a, b float64) float64 {
	if a < b {
		return a
	}
	return b
}

// GetCapability 获取验证码能力信息
func (s *SmartCaptchaSelector) GetCapability(captchaType CaptchaType) (*CaptchaCapability, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	capability, exists := s.capabilities[captchaType]
	if !exists {
		return nil, fmt.Errorf("unknown captcha type: %s", captchaType)
	}
	return capability, nil
}

// GetAllCapabilities 获取所有验证码能力信息
func (s *SmartCaptchaSelector) GetAllCapabilities() []*CaptchaCapability {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	var capabilities []*CaptchaCapability
	for _, cap := range s.capabilities {
		capabilities = append(capabilities, cap)
	}
	return capabilities
}

// UpdateRiskProfile 更新风险配置
func (s *SmartCaptchaSelector) UpdateRiskProfile(riskLevel RiskLevel, profile *CaptchaRiskProfile) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.riskProfiles[string(riskLevel)] = profile
	return nil
}

// GetRiskProfile 获取风险配置
func (s *SmartCaptchaSelector) GetRiskProfile(riskLevel RiskLevel) (*CaptchaRiskProfile, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	profile, exists := s.riskProfiles[string(riskLevel)]
	if !exists {
		return nil, fmt.Errorf("unknown risk level: %s", riskLevel)
	}
	return profile, nil
}

// AnalyzeUserHistory 分析用户历史记录
func (s *SmartCaptchaSelector) AnalyzeUserHistory(userID string) *UserAnalysisResult {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	records, exists := s.selectionHistory[userID]
	if !exists || len(records) == 0 {
		return nil
	}
	
	analysis := &UserAnalysisResult{
		UserID:         userID,
		TotalAttempts:  len(records),
		SuccessRate:    0,
		PreferredTypes: make(map[CaptchaType]int),
		BestPerforming: CaptchaTypeSlider,
		WorstPerforming: CaptchaTypeSlider,
	}
	
	successCount := 0
	typePerformance := make(map[CaptchaType]*struct{ success, total int })
	
	for _, record := range records {
		if _, exists := typePerformance[record.CaptchaType]; !exists {
			typePerformance[record.CaptchaType] = &struct{ success, total int }{0, 0}
		}
		typePerformance[record.CaptchaType].total++
		analysis.PreferredTypes[record.CaptchaType]++
		
		if record.Success {
			successCount++
			typePerformance[record.CaptchaType].success++
		}
	}
	
	analysis.SuccessRate = float64(successCount) / float64(len(records))
	
	bestRate := 0.0
	worstRate := 1.0
	
	for ct, perf := range typePerformance {
		rate := float64(perf.success) / float64(perf.total)
		if rate > bestRate {
			bestRate = rate
			analysis.BestPerforming = ct
		}
		if rate < worstRate {
			worstRate = rate
			analysis.WorstPerforming = ct
		}
	}
	
	return analysis
}

// UserAnalysisResult 用户分析结果
type UserAnalysisResult struct {
	UserID          string
	TotalAttempts   int
	SuccessRate     float64
	PreferredTypes  map[CaptchaType]int
	BestPerforming  CaptchaType
	WorstPerforming CaptchaType
}

// SelectMultipleCaptchas 选择多个验证码用于组合验证
func (s *SmartCaptchaSelector) SelectMultipleCaptchas(ctx context.Context, context *SelectionContext, count int) ([]*CaptchaSelectionResult, error) {
	if count <= 0 {
		return nil, fmt.Errorf("count must be positive")
	}
	
	results := make([]*CaptchaSelectionResult, 0, count)
	
	for i := 0; i < count; i++ {
		selection, err := s.SelectCaptcha(ctx, context)
		if err != nil {
			return results, err
		}
		
		results = append(results, selection)
		
		context.PreviousAttempts++
	}
	
	return results, nil
}