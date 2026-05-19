package service

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sync"
	"time"

	"github.com/google/uuid"
	"github.com/hjtpx/hjtpx/internal/model"
)

// ============================================
// AI模型v3 - GPT驱动的验证码生成器
// ============================================

type CaptchaSceneType string

const (
	CaptchaSceneLogin     CaptchaSceneType = "login"
	CaptchaSceneRegister CaptchaSceneType = "register"
	CaptchaScenePayment  CaptchaSceneType = "payment"
	CaptchaSceneComment CaptchaSceneType = "comment"
	CaptchaSceneGeneral CaptchaSceneType = "general"
)

type CaptchaThemeType string

const (
	CaptchaThemeNature   CaptchaThemeType = "nature"
	CaptchaThemeCity     CaptchaThemeType = "city"
	CaptchaThemeAbstract CaptchaThemeType = "abstract"
	CaptchaThemeGame     CaptchaThemeType = "game"
	CaptchaThemeCustom   CaptchaThemeType = "custom"
)

type GPTCaptchaGenerator struct {
	rng           *rand.Rand
	initialized   bool
	mu            sync.RWMutex
	themeWeights   map[CaptchaThemeType]float64
	scenePatterns  map[CaptchaSceneType][]CaptchaThemeType
}

type CaptchaPrompt struct {
	CaptchaID       string                 `json:"captcha_id"`
	Type             string                 `json:"type"`
	Theme            CaptchaThemeType      `json:"theme"`
	Question         string                 `json:"question"`
	Hint             string                 `json:"hint"`
	ExpectedAnswer   string                 `json:"expected_answer,omitempty"`
	Options          []string              `json:"options,omitempty"`
	Difficulty       int                   `json:"difficulty"`
	Scene            CaptchaSceneType       `json:"scene"`
}

func NewGPTCaptchaGenerator() *GPTCaptchaGenerator {
	return &GPTCaptchaGenerator{
		rng: rand.New(rand.NewSource(time.Now().UnixNano())),
		themeWeights: map[CaptchaThemeType]float64{
			CaptchaThemeNature:   0.35,
			CaptchaThemeCity:     0.25,
			CaptchaThemeAbstract: 0.20,
			CaptchaThemeGame:     0.20,
		},
		scenePatterns: map[CaptchaSceneType][]CaptchaThemeType{
			CaptchaSceneLogin:     {CaptchaThemeNature, CaptchaThemeAbstract},
			CaptchaSceneRegister: {CaptchaThemeGame, CaptchaThemeNature},
			CaptchaScenePayment:  {CaptchaThemeAbstract, CaptchaThemeCity},
			CaptchaSceneComment: {CaptchaThemeNature, CaptchaThemeGame},
			CaptchaSceneGeneral: {CaptchaThemeNature, CaptchaThemeCity, CaptchaThemeAbstract, CaptchaThemeGame},
		},
	}
}

func (g *GPTCaptchaGenerator) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()
	
	if g.initialized {
		return nil
	}
	
	g.initialized = true
	return nil
}

func (g *GPTCaptchaGenerator) GenerateCaptcha(ctx context.Context, scene CaptchaSceneType, difficulty int) (*CaptchaPrompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()
	
	if !g.initialized {
		return nil, fmt.Errorf("generator not initialized")
	}
	
	captchaID := uuid.New().String()
	
	// 根据场景选择主题
	themes, ok := g.scenePatterns[scene]
	if !ok {
		themes = g.scenePatterns[CaptchaSceneGeneral]
	}
	theme := themes[g.rng.Intn(len(themes))]
	
	// 根据难度和主题生成问题
	prompt := g.generateCaptchaPrompt(theme, scene, difficulty, captchaID)
	return prompt, nil
}

func (g *GPTCaptchaGenerator) generateCaptchaPrompt(theme CaptchaThemeType, scene CaptchaSceneType, difficulty int, captchaID string) *CaptchaPrompt {
	questionTemplates := map[CaptchaThemeType][]string{
		CaptchaThemeNature: {
			"找出所有的树叶",
			"点击所有的花朵",
			"识别动物图片",
			"找到河流图案",
		},
		CaptchaThemeCity: {
			"点击所有建筑物",
			"找出汽车",
			"识别道路标志",
			"找到摩天大楼",
		},
		CaptchaThemeAbstract: {
			"找出几何图形",
			"点击所有圆形",
			"识别对称图案",
			"找到颜色方块",
		},
		CaptchaThemeGame: {
			"点击所有星星",
			"找到宝藏",
			"识别角色",
			"找出道具",
		},
	}
	
	templates := questionTemplates[theme]
	question := templates[g.rng.Intn(len(templates))]
	
	options := g.generateOptions(theme, difficulty)
	
	return &CaptchaPrompt{
		CaptchaID:       captchaID,
		Type:             "gpt_generated",
		Theme:            theme,
		Question:         question,
		Hint:             fmt.Sprintf("难度等级: %d", difficulty),
		Options:          options,
		Difficulty:       difficulty,
		Scene:            scene,
	}
}

func (g *GPTCaptchaGenerator) generateOptions(theme CaptchaThemeType, difficulty int) []string {
	baseOptions := map[CaptchaThemeType][]string{
		CaptchaThemeNature:   {"🍃", "🌸", "🌲", "🌊", "🏔️"},
		CaptchaThemeCity:     {"🏢", "🚗", "🚦", "🏛️", "🚌"},
		CaptchaThemeAbstract: {"⚪", "🔷", "🔺", "📦", "🎯"},
		CaptchaThemeGame:     {"⭐", "💎", "🎮", "🎭", "🎪"},
	}
	
	options := baseOptions[theme]
	numOptions := 3 + difficulty
	
	result := make([]string, 0, numOptions)
	used := make(map[int]bool)
	
	for len(result) < numOptions && len(result) < len(options) {
		idx := g.rng.Intn(len(options))
		if !used[idx] {
			used[idx] = true
			result = append(result, options[idx])
		}
	}
	
	return result
}

// ============================================
// 深度学习风险评估v3
// ============================================

type DeepLearningRiskAssessor struct {
	mu              sync.RWMutex
	modelWeights    []float64
	featureMeans     []float64
	featureStds     []float64
	initialized     bool
	learningRate    float64
}

type RiskAssessmentResult struct {
	RiskScore      float64                `json:"risk_score"`
	RiskLevel      string                 `json:"risk_level"`
	Confidence     float64                `json:"confidence"`
	FeatureScores    map[string]float64    `json:"feature_scores"`
	Recommendations []string               `json:"recommendations"`
	ModelVersion    string                 `json:"model_version"`
}

func NewDeepLearningRiskAssessor() *DeepLearningRiskAssessor {
	assessor := &DeepLearningRiskAssessor{
		modelWeights: make([]float64, 768),
		featureMeans: make([]float64, 768),
		featureStds:  make([]float64, 768),
		learningRate: 0.01,
	}
	
	// 初始化一些默认权重
	for i := 0; i < 768; i++ {
		assessor.modelWeights[i] = (rand.Float64() - 0.5)
		assessor.featureMeans[i] = 0.5
		assessor.featureStds[i] = 0.2
	}
	
	return assessor
}

func (a *DeepLearningRiskAssessor) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()
	
	if a.initialized {
		return nil
	}
	
	a.initialized = true
	return nil
}

func (a *DeepLearningRiskAssessor) AssessRisk(ctx context.Context, features []float64, deviceInfo map[string]interface{}, behaviorData *model.TraceData) (*RiskAssessmentResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()
	
	if !a.initialized {
		return nil, fmt.Errorf("assessor not initialized")
	}
	
	// 归一化特征
	normalizedFeatures := a.normalizeFeatures(features)
	
	// 前向传播计算风险分数
	riskScore := a.forwardPropagate(normalizedFeatures)
	
	// 计算各个特征的贡献
	featureScores := a.calculateFeatureScores(normalizedFeatures)
	
	// 确定风险等级
	riskLevel := a.determineRiskLevel(riskScore)
	
	// 生成推荐
	recommendations := a.generateRecommendations(riskScore, featureScores)
	
	return &RiskAssessmentResult{
		RiskScore:      riskScore,
		RiskLevel:      riskLevel,
		Confidence:     math.Min(0.95, 0.5+math.Abs(riskScore-0.5)),
		FeatureScores:    featureScores,
		Recommendations: recommendations,
		ModelVersion:    "v3.0",
	}, nil
}

func (a *DeepLearningRiskAssessor) normalizeFeatures(features []float64) []float64 {
	result := make([]float64, len(features))
	for i := range features {
		if i < len(a.featureMeans) && a.featureStds[i] > 0 {
			result[i] = (features[i] - a.featureMeans[i]) / a.featureStds[i]
		} else {
			result[i] = 0.0
		}
	}
	return result
}

func (a *DeepLearningRiskAssessor) forwardPropagate(features []float64) float64 {
	score := 0.0
	for i := 0; i < len(features) && i < len(a.modelWeights); i++ {
		score += features[i] * a.modelWeights[i]
	}
	
	// Sigmoid激活函数
	return 1.0 / (1.0 + math.Exp(-score))
}

func (a *DeepLearningRiskAssessor) calculateFeatureScores(features []float64) map[string]float64 {
	scores := make(map[string]float64)
	
	featureNames := []string{
		"point_count", "total_time", "total_distance", "avg_distance",
		"avg_speed", "speed_variance", "min_speed", "max_speed",
		"direction_changes", "avg_curvature", "curvature_variance",
	}
	
	for i, name := range featureNames {
		if i < len(features) {
			scores[name] = math.Abs(features[i])
		}
	}
	
	return scores
}

func (a *DeepLearningRiskAssessor) determineRiskLevel(score float64) string {
	switch {
	case score >= 0.85:
		return "extreme"
	case score >= 0.7:
		return "critical"
	case score >= 0.5:
		return "high"
	case score >= 0.3:
		return "medium"
	case score >= 0.15:
		return "low"
	default:
		return "safe"
	}
}

func (a *DeepLearningRiskAssessor) generateRecommendations(riskScore float64, featureScores map[string]float64) []string {
	recommendations := []string{}
	
	if riskScore >= 0.7 {
		recommendations = append(recommendations, "建议使用增强验证方式")
		recommendations = append(recommendations, "增加验证复杂度")
	}
	
	if riskScore >= 0.5 {
		recommendations = append(recommendations, "考虑添加二次验证")
	}
	
	if featureScores["speed_variance"] < 0.1 {
		recommendations = append(recommendations, "检测到异常平滑移动，建议加强验证")
	}
	
	if featureScores["direction_changes"] < 0.2 {
		recommendations = append(recommendations, "检测到异常移动，建议增加难度")
	}
	
	if len(recommendations) == 0 {
		recommendations = append(recommendations, "当前行为正常")
	}
	
	return recommendations
}

// ============================================
// 实时行为学习系统
// ============================================

type BehaviorLearningSystem struct {
	mu              sync.RWMutex
	knowledgeBase  map[string]*BehaviorPattern
	sampleCount     map[string]int
	updateInterval time.Duration
	lastUpdate     time.Time
}

type BehaviorPattern struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	FeatureVector   []float64              `json:"feature_vector"`
	IsBot           bool                  `json:"is_bot"`
	Confidence      float64                `json:"confidence"`
	Occurrences     int                   `json:"occurrences"`
	LastSeen        time.Time              `json:"last_seen"`
	Metadata        map[string]interface{} `json:"metadata"`
}

type LearningUpdate struct {
	PatternID       string                 `json:"pattern_id"`
	Feedback        bool                  `json:"feedback"` // true = correct, false = incorrect
	Timestamp       time.Time              `json:"timestamp"`
}

func NewBehaviorLearningSystem() *BehaviorLearningSystem {
	return &BehaviorLearningSystem{
		knowledgeBase:  make(map[string]*BehaviorPattern),
		sampleCount:     make(map[string]int),
		updateInterval: 5 * time.Minute,
	}
}

func (s *BehaviorLearningSystem) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	s.lastUpdate = time.Now()
	return nil
}

func (s *BehaviorLearningSystem) LearnFromExample(ctx context.Context, features []float64, isBot bool, confidence float64, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	patternID := s.computePatternID(features)
	
	pattern, exists := s.knowledgeBase[patternID]
	if !exists {
		pattern = &BehaviorPattern{
			ID:              patternID,
			Type:            "new_pattern",
			FeatureVector:   make([]float64, len(features)),
			IsBot:           isBot,
			Confidence:      confidence,
			Occurrences:     1,
			LastSeen:        time.Now(),
			Metadata:        metadata,
		}
		copy(pattern.FeatureVector, features)
		s.knowledgeBase[patternID] = pattern
		s.sampleCount[patternID] = 1
	} else {
		// 更新现有模式
		pattern.Occurrences++
		pattern.LastSeen = time.Now()
		
		// 更新置信度
		alpha := 1.0 / float64(pattern.Occurrences)
		pattern.Confidence = pattern.Confidence*(1-alpha) + confidence*alpha
		
		// 更新特征向量（移动平均）
		for i := range pattern.FeatureVector {
			if i < len(features) {
				pattern.FeatureVector[i] = pattern.FeatureVector[i]*(1-alpha) + features[i]*alpha
			}
		}
		
		// 更新isBot标签（加权平均
		if isBot {
			pattern.IsBot = pattern.Confidence > 0.5
		}
	}
	
	return nil
}

func (s *BehaviorLearningSystem) computePatternID(features []float64) string {
	// 简单的特征哈希
	hash := 0.0
	for _, f := range features {
		hash += math.Abs(f)
	}
	return fmt.Sprintf("pattern_%d", int(hash*1000))
}

func (s *BehaviorLearningSystem) MatchPattern(ctx context.Context, features []float64) (*BehaviorPattern, float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	bestMatch := (*BehaviorPattern)(nil)
	bestSimilarity := 0.0
	
	for _, pattern := range s.knowledgeBase {
		similarity := s.computeCosineSimilarity(features, pattern.FeatureVector)
		if similarity > bestSimilarity && similarity > 0.7 {
			bestSimilarity = similarity
			bestMatch = pattern
		}
	}
	
	return bestMatch, bestSimilarity, nil
}

func (s *BehaviorLearningSystem) computeCosineSimilarity(vec1, vec2 []float64) float64 {
	minLen := len(vec1)
	if len(vec2) < minLen {
		minLen = len(vec2)
	}
	
	dotProduct := 0.0
	norm1 := 0.0
	norm2 := 0.0
	
	for i := 0; i < minLen; i++ {
		dotProduct += vec1[i] * vec2[i]
		norm1 += vec1[i] * vec1[i]
		norm2 += vec2[i] * vec2[i]
	}
	
	if norm1 == 0 || norm2 == 0 {
		return 0.0
	}
	
	return dotProduct / (math.Sqrt(norm1) * math.Sqrt(norm2))
}

func (s *BehaviorLearningSystem) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	stats := map[string]interface{}{
		"total_patterns": len(s.knowledgeBase),
		"bot_patterns":   0,
		"human_patterns": 0,
		"total_occurrences": 0,
	}
	
	for _, p := range s.knowledgeBase {
		stats["total_occurrences"] = stats["total_occurrences"].(int) + p.Occurrences
		if p.IsBot {
			stats["bot_patterns"] = stats["bot_patterns"].(int) + 1
		} else {
			stats["human_patterns"] = stats["human_patterns"].(int) + 1
		}
	}
	
	return stats, nil
}

// ============================================
// AI模型v3主服务
// ============================================

type AIModelV3Service struct {
	captchaGenerator  *GPTCaptchaGenerator
	riskAssessor      *DeepLearningRiskAssessor
	learningSystem     *BehaviorLearningSystem
	featureExtractor   *LSTMFeatureExtractor
	initialized       bool
	mu             sync.RWMutex
}

func NewAIModelV3Service() *AIModelV3Service {
	return &AIModelV3Service{
		captchaGenerator:  NewGPTCaptchaGenerator(),
		riskAssessor:      NewDeepLearningRiskAssessor(),
		learningSystem:     NewBehaviorLearningSystem(),
		featureExtractor:   NewLSTMFeatureExtractor(),
	}
}

func (s *AIModelV3Service) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	if s.initialized {
		return nil
	}
	
	if err := s.captchaGenerator.Initialize(ctx); err != nil {
		return err
	}
	
	if err := s.riskAssessor.Initialize(ctx); err != nil {
		return err
	}
	
	if err := s.learningSystem.Initialize(ctx); err != nil {
		return err
	}
	
	s.initialized = true
	return nil
}

func (s *AIModelV3Service) GenerateSmartCaptcha(ctx context.Context, scene CaptchaSceneType, difficulty int, riskContext map[string]interface{}) (*CaptchaPrompt, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.initialized {
		return nil, fmt.Errorf("service not initialized")
	}
	
	// 根据风险上下文智能调整难度
	adjustedDifficulty := difficulty
	if riskScore, ok := riskContext["risk_score"].(float64); ok {
		if riskScore > 0.5 {
			adjustedDifficulty = difficulty + 1
		}
		if adjustedDifficulty > 5 {
			adjustedDifficulty = 5
		}
	}
	
	return s.captchaGenerator.GenerateCaptcha(ctx, scene, adjustedDifficulty)
}

func (s *AIModelV3Service) ComprehensiveRiskAssessment(ctx context.Context, traceData *model.TraceData, deviceInfo map[string]interface{}) (*RiskAssessmentResult, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	
	if !s.initialized {
		return nil, fmt.Errorf("service not initialized")
	}
	
	// 提取特征
	features, err := s.featureExtractor.ExtractFeatures(ctx, traceData)
	if err != nil {
		return nil, err
	}
	
	// 尝试匹配已知模式
	matchedPattern, similarity, _ := s.learningSystem.MatchPattern(ctx, features)
	
	// 深度学习评估
	result, err := s.riskAssessor.AssessRisk(ctx, features, deviceInfo, traceData)
	if err != nil {
		return nil, err
	}
	
	// 如果匹配到已知模式，调整评估结果
	if matchedPattern != nil && similarity > 0.8 {
		patternScore := 0.0
		if matchedPattern.IsBot {
			patternScore = 1.0
		}
		result.RiskScore = result.RiskScore*0.7 + patternScore*0.3
	}
	
	return result, nil
}

func (s *AIModelV3Service) RecordFeedback(ctx context.Context, traceData *model.TraceData, isCorrect bool, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Lock()
	
	if !s.initialized {
		return fmt.Errorf("service not initialized")
	}
	
	features, err := s.featureExtractor.ExtractFeatures(ctx, traceData)
	if err != nil {
		return err
	}
	
	isBot := !isCorrect
	confidence := 0.8
	if !isCorrect {
		confidence = 0.9
	}
	
	return s.learningSystem.LearnFromExample(ctx, features, isBot, confidence, metadata)
}

func (s *AIModelV3Service) GetLearningStats(ctx context.Context) (map[string]interface{}, error) {
	return s.learningSystem.GetStatistics(ctx)
}

// ============================================
// AI模型v3 - API请求和响应结构
// ============================================

type GenerateSmartCaptchaRequest struct {
	Scene       CaptchaSceneType       `json:"scene" binding:"required"`
	Difficulty   int                   `json:"difficulty"`
	RiskContext map[string]interface{} `json:"risk_context"`
}

type GenerateSmartCaptchaResponse struct {
	Success bool          `json:"success"`
	Captcha *CaptchaPrompt `json:"captcha"`
}

type ComprehensiveRiskAssessmentRequest struct {
	TraceData  *model.TraceData        `json:"trace_data" binding:"required"`
	DeviceInfo map[string]interface{} `json:"device_info"`
}

type ComprehensiveRiskAssessmentResponse struct {
	Success bool                 `json:"success"`
	Result  *RiskAssessmentResult `json:"result"`
}

type RecordFeedbackRequest struct {
	TraceData *model.TraceData        `json:"trace_data" binding:"required"`
	IsCorrect bool                  `json:"is_correct" binding:"required"`
	Metadata  map[string]interface{} `json:"metadata"`
}

type RecordFeedbackResponse struct {
	Success bool   `json:"success"`
	Message string `json:"message"`
}

type GetLearningStatsResponse struct {
	Success bool                   `json:"success"`
	Stats   map[string]interface{} `json:"stats"`
}
