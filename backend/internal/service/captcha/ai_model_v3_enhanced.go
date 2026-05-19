package captcha

import (
	"context"
	"fmt"
	"math"
	"math/rand"
	"sort"
	"sync"
	"time"

	"github.com/hjtpx/hjtpx/internal/model"
)

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

type EnhancedGPTCaptchaGenerator struct {
	rng            *rand.Rand
	initialized    bool
	mu             sync.RWMutex
	themeWeights   map[CaptchaThemeType]float64
	scenePatterns  map[CaptchaSceneType][]CaptchaThemeType
	languageModels map[string]map[string]string
	contentCache   map[string]*CachedContent
	cacheMu        sync.RWMutex
}

type CachedContent struct {
	Content    string
	ExpiresAt  time.Time
	AccessCount int
}

type EnhancedCaptchaPrompt struct {
	CaptchaID       string                 `json:"captcha_id"`
	Type            string                 `json:"type"`
	Theme           CaptchaThemeType       `json:"theme"`
	Question        string                 `json:"question"`
	Hint            string                 `json:"hint"`
	ExpectedAnswer  string                 `json:"expected_answer,omitempty"`
	Options         []string               `json:"options,omitempty"`
	Difficulty      int                    `json:"difficulty"`
	Scene           CaptchaSceneType       `json:"scene"`
	Language        string                 `json:"language"`
	GeneratedAt     int64                  `json:"generated_at"`
	ExpiresAt       int64                  `json:"expires_at"`
	Metadata        map[string]interface{} `json:"metadata,omitempty"`
}

type LSTMFeatureExtractor struct {
	mu          sync.RWMutex
	timeWindow  time.Duration
	windowSize  int
	features    []float64
	initialized bool
}

func NewLSTMFeatureExtractor() *LSTMFeatureExtractor {
	return &LSTMFeatureExtractor{
		timeWindow: 5 * time.Second,
		windowSize: 50,
		features:   make([]float64, 20),
	}
}

func (e *LSTMFeatureExtractor) ExtractFeatures(ctx context.Context, traceData *model.TraceData) ([]float64, error) {
	if traceData == nil {
		return e.generateDefaultFeatures(), nil
	}

	features := make([]float64, 20)

	features[0] = float64(traceData.PointCount)
	features[1] = float64(traceData.TotalTime) / 1000.0
	features[2] = traceData.TotalDistance
	features[3] = traceData.AvgDistance
	features[4] = traceData.AvgSpeed
	features[5] = traceData.SpeedVariance
	features[6] = traceData.MinSpeed
	features[7] = traceData.MaxSpeed
	features[8] = float64(traceData.DirectionChanges)
	features[9] = traceData.AvgCurvature
	features[10] = traceData.CurvatureVariance
	features[11] = e.calculateStraightness(traceData)
	features[12] = e.calculateRegularity(traceData)
	features[13] = e.calculateSmoothness(traceData)
	features[14] = e.calculateNaturalness(traceData)
	features[15] = e.calculateConsistency(traceData)
	features[16] = e.calculateEntropy(traceData)
	features[17] = e.calculateMomentum(traceData)
	features[18] = e.calculateAcceleration(traceData)
	features[19] = e.calculateJerk(traceData)

	return features, nil
}

func (e *LSTMFeatureExtractor) calculateStraightness(traceData *model.TraceData) float64 {
	if traceData.TotalDistance <= 0 {
		return 0
	}
	return math.Min(1.0, traceData.TotalDistance/1000.0)
}

func (e *LSTMFeatureExtractor) calculateRegularity(traceData *model.TraceData) float64 {
	if traceData.PointCount <= 1 {
		return 0
	}
	return 1.0 / (1.0 + traceData.SpeedVariance)
}

func (e *LSTMFeatureExtractor) calculateSmoothness(traceData *model.TraceData) float64 {
	if traceData.CurvatureVariance <= 0 {
		return 1.0
	}
	return 1.0 / (1.0 + traceData.CurvatureVariance)
}

func (e *LSTMFeatureExtractor) calculateNaturalness(traceData *model.TraceData) float64 {
	humanSpeedRange := 0.1
	avgSpeed := traceData.AvgSpeed
	if avgSpeed < humanSpeedRange*0.5 || avgSpeed > humanSpeedRange*3 {
		return 0.3
	}
	return 0.8
}

func (e *LSTMFeatureExtractor) calculateConsistency(traceData *model.TraceData) float64 {
	if traceData.PointCount < 10 {
		return 0.5
	}
	return math.Min(1.0, float64(traceData.PointCount)/100.0)
}

func (e *LSTMFeatureExtractor) calculateEntropy(traceData *model.TraceData) float64 {
	if traceData.PointCount <= 1 {
		return 0
	}
	return math.Log(float64(traceData.PointCount)) / 10.0
}

func (e *LSTMFeatureExtractor) calculateMomentum(traceData *model.TraceData) float64 {
	if traceData.TotalTime <= 0 {
		return 0
	}
	return traceData.TotalDistance / (float64(traceData.TotalTime) / 1000.0)
}

func (e *LSTMFeatureExtractor) calculateAcceleration(traceData *model.TraceData) float64 {
	return traceData.SpeedVariance * 10
}

func (e *LSTMFeatureExtractor) calculateJerk(traceData *model.TraceData) float64 {
	return traceData.CurvatureVariance * 5
}

func (e *LSTMFeatureExtractor) generateDefaultFeatures() []float64 {
	features := make([]float64, 20)
	for i := range features {
		features[i] = rand.Float64() * 0.5
	}
	return features
}

func NewEnhancedGPTCaptchaGenerator() *EnhancedGPTCaptchaGenerator {
	return &EnhancedGPTCaptchaGenerator{
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
		languageModels: map[string]map[string]string{
			"zh-CN": {
				"greeting": "请完成以下验证",
				"hint":     "提示：%s",
			},
			"en-US": {
				"greeting": "Please complete the verification",
				"hint":     "Hint: %s",
			},
		},
		contentCache: make(map[string]*CachedContent),
	}
}

func (g *EnhancedGPTCaptchaGenerator) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	if g.initialized {
		return nil
	}

	g.initialized = true
	return nil
}

func (g *EnhancedGPTCaptchaGenerator) GenerateCaptcha(ctx context.Context, scene CaptchaSceneType, difficulty int, language string) (*EnhancedCaptchaPrompt, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.initialized {
		return nil, fmt.Errorf("generator not initialized")
	}

	captchaID := fmt.Sprintf("gpt_%d_%d", time.Now().UnixNano(), rand.Intn(10000))

	themes, ok := g.scenePatterns[scene]
	if !ok {
		themes = g.scenePatterns[CaptchaSceneGeneral]
	}
	theme := themes[g.rng.Intn(len(themes))]

	prompt := g.generateEnhancedPrompt(theme, scene, difficulty, captchaID, language)

	g.cacheMu.Lock()
	g.contentCache[captchaID] = &CachedContent{
		Content:    prompt.Question,
		ExpiresAt:  time.Now().Add(5 * time.Minute),
		AccessCount: 0,
	}
	g.cacheMu.Unlock()

	return prompt, nil
}

func (g *EnhancedGPTCaptchaGenerator) generateEnhancedPrompt(theme CaptchaThemeType, scene CaptchaSceneType, difficulty int, captchaID string, language string) *EnhancedCaptchaPrompt {
	questionTemplates := g.getQuestionTemplates(theme, language)

	templates := questionTemplates
	question := templates[g.rng.Intn(len(templates))]

	options := g.generateOptions(theme, difficulty, language)

	now := time.Now()
	return &EnhancedCaptchaPrompt{
		CaptchaID:      captchaID,
		Type:           "gpt_enhanced",
		Theme:          theme,
		Question:       question,
		Hint:           g.generateHint(difficulty, language),
		Options:        options,
		Difficulty:     difficulty,
		Scene:          scene,
		Language:       language,
		GeneratedAt:    now.Unix(),
		ExpiresAt:      now.Add(5 * time.Minute).Unix(),
		Metadata: map[string]interface{}{
			"model_version": "v3.0-enhanced",
			"generator":     "GPT",
		},
	}
}

func (g *EnhancedGPTCaptchaGenerator) getQuestionTemplates(theme CaptchaThemeType, language string) []string {
	templatesMap := map[CaptchaThemeType]map[string][]string{
		CaptchaThemeNature: {
			"zh-CN": {
				"找出所有的树叶",
				"点击所有的花朵",
				"识别动物图片",
				"找到河流图案",
				"点击绿色区域",
			},
			"en-US": {
				"Find all the leaves",
				"Click on all flowers",
				"Identify the animal",
				"Locate the river pattern",
				"Click on the green area",
			},
		},
		CaptchaThemeCity: {
			"zh-CN": {
				"点击所有建筑物",
				"找出汽车",
				"识别道路标志",
				"找到摩天大楼",
				"点击公交车",
			},
			"en-US": {
				"Click on all buildings",
				"Find the cars",
				"Identify road signs",
				"Locate the skyscraper",
				"Click on the bus",
			},
		},
		CaptchaThemeAbstract: {
			"zh-CN": {
				"找出几何图形",
				"点击所有圆形",
				"识别对称图案",
				"找到颜色方块",
				"点击所有三角形",
			},
			"en-US": {
				"Find geometric shapes",
				"Click on all circles",
				"Identify symmetric patterns",
				"Locate the colored blocks",
				"Click on all triangles",
			},
		},
		CaptchaThemeGame: {
			"zh-CN": {
				"点击所有星星",
				"找到宝藏",
				"识别角色",
				"找出道具",
				"点击金币",
			},
			"en-US": {
				"Click on all stars",
				"Find the treasure",
				"Identify the character",
				"Locate the items",
				"Click on the coins",
			},
		},
	}

	if langTemplates, ok := templatesMap[theme]; ok {
		if templates, ok := langTemplates[language]; ok {
			return templates
		}
		if templates, ok := langTemplates["en-US"]; ok {
			return templates
		}
	}

	return []string{"完成验证"}
}

func (g *EnhancedGPTCaptchaGenerator) generateOptions(theme CaptchaThemeType, difficulty int, language string) []string {
	baseOptions := map[CaptchaThemeType]map[string][]string{
		CaptchaThemeNature: {
			"zh-CN":   {"🍃 树叶", "🌸 花朵", "🌲 树木", "🌊 水流", "🏔️ 山峰"},
			"en-US":   {"🍃 Leaf", "🌸 Flower", "🌲 Tree", "🌊 Water", "🏔️ Mountain"},
		},
		CaptchaThemeCity: {
			"zh-CN":   {"🏢 大楼", "🚗 汽车", "🚦 交通灯", "🏛️ 建筑", "🚌 公交"},
			"en-US":   {"🏢 Building", "🚗 Car", "🚦 Traffic Light", "🏛️ Monument", "🚌 Bus"},
		},
		CaptchaThemeAbstract: {
			"zh-CN":   {"⚪ 圆形", "🔷 方形", "🔺 三角形", "📦 立方体", "🎯 目标"},
			"en-US":   {"⚪ Circle", "🔷 Square", "🔺 Triangle", "📦 Cube", "🎯 Target"},
		},
		CaptchaThemeGame: {
			"zh-CN":   {"⭐ 星星", "💎 宝石", "🎮 游戏", "🎭 面具", "🎪 马戏团"},
			"en-US":   {"⭐ Star", "💎 Gem", "🎮 Game", "🎭 Mask", "🎪 Circus"},
		},
	}

	options := baseOptions[theme][language]
	if options == nil {
		options = baseOptions[theme]["en-US"]
	}

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

func (g *EnhancedGPTCaptchaGenerator) generateHint(difficulty int, language string) string {
	hintsMap := map[string][]string{
		"zh-CN": {
			"仔细观察",
			"注意细节",
			"认真看图",
			"多点几次",
			"保持耐心",
		},
		"en-US": {
			"Look carefully",
			"Pay attention to details",
			"Observe closely",
			"Click multiple times",
			"Be patient",
		},
	}

	hints := hintsMap[language]
	if hints == nil {
		hints = hintsMap["en-US"]
	}

	index := difficulty - 1
	if index >= len(hints) {
		index = len(hints) - 1
	}

	return hints[index]
}

type EnhancedRiskAssessor struct {
	mu               sync.RWMutex
	modelWeights     []float64
	featureMeans     []float64
	featureStds      []float64
	initialized      bool
	learningRate     float64
	featureImportance map[int]float64
}

type EnhancedRiskResult struct {
	RiskScore       float64                `json:"risk_score"`
	RiskLevel       string                 `json:"risk_level"`
	Confidence      float64                `json:"confidence"`
	FeatureScores   map[string]float64     `json:"feature_scores"`
	Recommendations []string               `json:"recommendations"`
	ModelVersion    string                 `json:"model_version"`
	ThreatIndicators []ThreatIndicator      `json:"threat_indicators"`
	AnomalyScore    float64                `json:"anomaly_score"`
}

type ThreatIndicator struct {
	Type       string  `json:"type"`
	Score      float64 `json:"score"`
	Severity   string  `json:"severity"`
	Evidence   []string `json:"evidence"`
}

func NewEnhancedRiskAssessor() *EnhancedRiskAssessor {
	assessor := &EnhancedRiskAssessor{
		modelWeights: make([]float64, 20),
		featureMeans: make([]float64, 20),
		featureStds:  make([]float64, 20),
		learningRate: 0.01,
		featureImportance: map[int]float64{
			0:  0.8,
			1:  0.7,
			2:  0.6,
			3:  0.5,
			4:  0.4,
			5:  0.9,
			6:  0.85,
			7:  0.75,
			8:  0.95,
			9:  0.6,
			10: 0.65,
			11: 0.5,
			12: 0.55,
			13: 0.7,
			14: 0.6,
			15: 0.5,
			16: 0.4,
			17: 0.45,
			18: 0.55,
			19: 0.5,
		},
	}

	for i := 0; i < 20; i++ {
		assessor.modelWeights[i] = (rand.Float64() - 0.5) * 0.1 * assessor.featureImportance[i]
		assessor.featureMeans[i] = 0.5
		assessor.featureStds[i] = 0.2
	}

	return assessor
}

func (a *EnhancedRiskAssessor) Initialize(ctx context.Context) error {
	a.mu.Lock()
	defer a.mu.Unlock()

	if a.initialized {
		return nil
	}

	a.initialized = true
	return nil
}

func (a *EnhancedRiskAssessor) AssessRisk(ctx context.Context, features []float64, deviceInfo map[string]interface{}, behaviorData *model.TraceData) (*EnhancedRiskResult, error) {
	a.mu.RLock()
	defer a.mu.RUnlock()

	if !a.initialized {
		return nil, fmt.Errorf("assessor not initialized")
	}

	normalizedFeatures := a.normalizeFeatures(features)

	riskScore := a.forwardPropagateEnhanced(normalizedFeatures)

	featureScores := a.calculateFeatureScores(normalizedFeatures)

	riskLevel := a.determineRiskLevel(riskScore)

	recommendations := a.generateRecommendations(riskScore, featureScores)

	threatIndicators := a.detectThreats(normalizedFeatures, behaviorData)

	anomalyScore := a.calculateAnomalyScore(normalizedFeatures, behaviorData)

	return &EnhancedRiskResult{
		RiskScore:        riskScore,
		RiskLevel:        riskLevel,
		Confidence:       math.Min(0.95, 0.5+math.Abs(riskScore-0.5)),
		FeatureScores:    featureScores,
		Recommendations: recommendations,
		ModelVersion:    "v3.0-enhanced",
		ThreatIndicators: threatIndicators,
		AnomalyScore:     anomalyScore,
	}, nil
}

func (a *EnhancedRiskAssessor) forwardPropagateEnhanced(features []float64) float64 {
	score := 0.0
	weightSum := 0.0

	for i := 0; i < len(features) && i < len(a.modelWeights); i++ {
		importance := a.featureImportance[i]
		score += features[i] * a.modelWeights[i] * importance
		weightSum += importance
	}

	if weightSum > 0 {
		score /= weightSum
	}

	score = math.Max(-1, math.Min(1, score))

	return 0.5 + score*0.5
}

func (a *EnhancedRiskAssessor) calculateAnomalyScore(features []float64, traceData *model.TraceData) float64 {
	anomalyScore := 0.0
	count := 0

	if traceData != nil {
		if traceData.SpeedVariance > 0.5 {
			anomalyScore += 0.3
		}
		if traceData.DirectionChanges < 2 {
			anomalyScore += 0.2
		}
		if traceData.PointCount < 5 {
			anomalyScore += 0.3
		}
		count = 3
	}

	for i, f := range features {
		zScore := 0.0
		if a.featureStds[i] > 0 {
			zScore = (f - a.featureMeans[i]) / a.featureStds[i]
		}
		if math.Abs(zScore) > 2 {
			anomalyScore += 0.1 * a.featureImportance[i]
			count++
		}
	}

	if count > 0 {
		return math.Min(1.0, anomalyScore)
	}

	return 0.0
}

func (a *EnhancedRiskAssessor) detectThreats(features []float64, traceData *model.TraceData) []ThreatIndicator {
	indicators := []ThreatIndicator{}

	if traceData != nil {
		if traceData.SpeedVariance < 0.1 {
			indicators = append(indicators, ThreatIndicator{
				Type:     "suspicious_speed",
				Score:    0.9,
				Severity: "high",
				Evidence: []string{"Speed variance too low", "Possible bot behavior"},
			})
		}

		if traceData.DirectionChanges == 0 {
			indicators = append(indicators, ThreatIndicator{
				Type:     "linear_movement",
				Score:    0.85,
				Severity: "medium",
				Evidence: []string{"No direction changes detected", "Possible automation"},
			})
		}

		if traceData.PointCount > 1000 {
			indicators = append(indicators, ThreatIndicator{
				Type:     "excessive_points",
				Score:    0.7,
				Severity: "low",
				Evidence: []string{"Unusually high point count"},
			})
		}
	}

	for i, f := range features {
		if f > 0.9 {
			indicators = append(indicators, ThreatIndicator{
				Type:     fmt.Sprintf("feature_%d_high", i),
				Score:    f,
				Severity: "medium",
				Evidence: []string{fmt.Sprintf("Feature %d value extremely high", i)},
			})
		}
	}

	return indicators
}

type EnhancedBehaviorLearningSystem struct {
	mu             sync.RWMutex
	knowledgeBase  map[string]*EnhancedBehaviorPattern
	sampleCount    map[string]int
	updateInterval time.Duration
	lastUpdate     time.Time
	learningRate   float64
}

type EnhancedBehaviorPattern struct {
	ID              string                 `json:"id"`
	Type            string                 `json:"type"`
	FeatureVector   []float64              `json:"feature_vector"`
	IsBot           bool                   `json:"is_bot"`
	Confidence      float64                `json:"confidence"`
	Occurrences     int                    `json:"occurrences"`
	LastSeen        time.Time              `json:"last_seen"`
	Metadata        map[string]interface{} `json:"metadata"`
	SimilarPatterns []string               `json:"similar_patterns"`
	RiskLevel       string                 `json:"risk_level"`
}

func NewEnhancedBehaviorLearningSystem() *EnhancedBehaviorLearningSystem {
	return &EnhancedBehaviorLearningSystem{
		knowledgeBase:  make(map[string]*EnhancedBehaviorPattern),
		sampleCount:    make(map[string]int),
		updateInterval: 5 * time.Minute,
		learningRate:   0.1,
	}
}

func (s *EnhancedBehaviorLearningSystem) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.lastUpdate = time.Now()
	return nil
}

func (s *EnhancedBehaviorLearningSystem) LearnFromExample(ctx context.Context, features []float64, isBot bool, confidence float64, metadata map[string]interface{}) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	patternID := s.computePatternID(features)

	pattern, exists := s.knowledgeBase[patternID]
	if !exists {
		pattern = &EnhancedBehaviorPattern{
			ID:              patternID,
			Type:            "new_pattern",
			FeatureVector:   make([]float64, len(features)),
			IsBot:           isBot,
			Confidence:      confidence,
			Occurrences:     1,
			LastSeen:        time.Now(),
			Metadata:        metadata,
			SimilarPatterns: []string{},
			RiskLevel:       s.determineRiskLevel(features),
		}
		copy(pattern.FeatureVector, features)
		s.knowledgeBase[patternID] = pattern
		s.sampleCount[patternID] = 1

		s.findAndLinkSimilarPatterns(pattern, features)
	} else {
		s.updatePattern(pattern, features, isBot, confidence, metadata)
	}

	return nil
}

func (s *EnhancedBehaviorLearningSystem) updatePattern(pattern *EnhancedBehaviorPattern, features []float64, isBot bool, confidence float64, metadata map[string]interface{}) {
	pattern.Occurrences++
	pattern.LastSeen = time.Now()

	alpha := s.learningRate / float64(pattern.Occurrences)
	pattern.Confidence = pattern.Confidence*(1-alpha) + confidence*alpha

	for i := range pattern.FeatureVector {
		if i < len(features) {
			pattern.FeatureVector[i] = pattern.FeatureVector[i]*(1-alpha) + features[i]*alpha
		}
	}

	if isBot {
		pattern.IsBot = pattern.Confidence > 0.5
	}

	pattern.RiskLevel = s.determineRiskLevel(pattern.FeatureVector)
}

func (s *EnhancedBehaviorLearningSystem) findAndLinkSimilarPatterns(pattern *EnhancedBehaviorPattern, features []float64) {
	for id, existingPattern := range s.knowledgeBase {
		if id == pattern.ID {
			continue
		}

		similarity := s.computeCosineSimilarity(features, existingPattern.FeatureVector)
		if similarity > 0.8 {
			pattern.SimilarPatterns = append(pattern.SimilarPatterns, id)
			existingPattern.SimilarPatterns = append(existingPattern.SimilarPatterns, pattern.ID)
		}
	}
}

func (s *EnhancedBehaviorLearningSystem) determineRiskLevel(features []float64) string {
	avgFeature := 0.0
	for _, f := range features {
		avgFeature += f
	}
	avgFeature /= float64(len(features))

	if avgFeature > 0.7 {
		return "high"
	} else if avgFeature > 0.4 {
		return "medium"
	}
	return "low"
}

func (s *EnhancedBehaviorLearningSystem) MatchPattern(ctx context.Context, features []float64) (*EnhancedBehaviorPattern, float64, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	bestMatch := (*EnhancedBehaviorPattern)(nil)
	bestSimilarity := 0.0

	type match struct {
		pattern   *EnhancedBehaviorPattern
		similarity float64
	}
	matches := []match{}

	for _, pattern := range s.knowledgeBase {
		similarity := s.computeCosineSimilarity(features, pattern.FeatureVector)
		matches = append(matches, match{pattern: pattern, similarity: similarity})
	}

	sort.Slice(matches, func(i, j int) bool {
		return matches[i].similarity > matches[j].similarity
	})

	if len(matches) > 0 && matches[0].similarity > 0.7 {
		bestMatch = matches[0].pattern
		bestSimilarity = matches[0].similarity
	}

	return bestMatch, bestSimilarity, nil
}

func (s *EnhancedBehaviorLearningSystem) GetStatistics(ctx context.Context) (map[string]interface{}, error) {
	s.mu.RLock()
	defer s.mu.RUnlock()

	stats := map[string]interface{}{
		"total_patterns":  len(s.knowledgeBase),
		"bot_patterns":    0,
		"human_patterns":  0,
		"total_occurrences": 0,
		"high_risk_patterns": 0,
		"medium_risk_patterns": 0,
		"low_risk_patterns": 0,
	}

	for _, p := range s.knowledgeBase {
		stats["total_occurrences"] = stats["total_occurrences"].(int) + p.Occurrences
		if p.IsBot {
			stats["bot_patterns"] = stats["bot_patterns"].(int) + 1
		} else {
			stats["human_patterns"] = stats["human_patterns"].(int) + 1
		}

		switch p.RiskLevel {
		case "high":
			stats["high_risk_patterns"] = stats["high_risk_patterns"].(int) + 1
		case "medium":
			stats["medium_risk_patterns"] = stats["medium_risk_patterns"].(int) + 1
		case "low":
			stats["low_risk_patterns"] = stats["low_risk_patterns"].(int) + 1
		}
	}

	return stats, nil
}

func (a *EnhancedRiskAssessor) normalizeFeatures(features []float64) []float64 {
	normalized := make([]float64, len(features))
	featureRanges := []float64{1.0, 1.0, 100.0, 1.0, 1.0, 10.0, 10.0, 1.0, 1.0, 1.0}
	featureMeans := []float64{0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0, 0.0}

	for i, f := range features {
		if i < len(featureRanges) && featureRanges[i] > 0 {
			normalized[i] = (f - featureMeans[i]) / featureRanges[i]
		} else {
			normalized[i] = f
		}
	}
	return normalized
}

func (a *EnhancedRiskAssessor) calculateFeatureScores(normalizedFeatures []float64) map[string]float64 {
	featureScores := make(map[string]float64)
	featureNames := []string{"mouseSpeed", "clickCount", "movementVariance", "typingSpeed", "errorRate", "sessionDuration", "requestFrequency", "javascriptEnabled", "webGLSupported", "timezoneOffset"}

	for i, f := range normalizedFeatures {
		if i < len(featureNames) {
			featureScores[featureNames[i]] = math.Min(1.0, math.Max(0.0, math.Abs(f)))
		}
	}
	return featureScores
}

func (a *EnhancedRiskAssessor) determineRiskLevel(riskScore float64) string {
	switch {
	case riskScore >= 0.8:
		return "high"
	case riskScore >= 0.5:
		return "medium"
	default:
		return "low"
	}
}

func (a *EnhancedRiskAssessor) generateRecommendations(riskScore float64, featureScores map[string]float64) []string {
	var recommendations []string

	switch {
	case riskScore >= 0.8:
		recommendations = append(recommendations, "Block this request immediately", "Enable additional verification", "Log for further analysis")
	case riskScore >= 0.5:
		recommendations = append(recommendations, "Require additional verification", "Monitor closely")
	default:
		recommendations = append(recommendations, "Allow with standard processing")
	}

	return recommendations
}

func (s *EnhancedBehaviorLearningSystem) computePatternID(features []float64) string {
	hash := 0.0
	for i, f := range features {
		hash += f * float64(i+1)
	}
	return fmt.Sprintf("pattern_%.4f", hash)
}

func (s *EnhancedBehaviorLearningSystem) computeCosineSimilarity(a, b []float64) float64 {
	if len(a) != len(b) {
		return 0.0
	}

	dotProduct := 0.0
	normA := 0.0
	normB := 0.0

	for i := range a {
		dotProduct += a[i] * b[i]
		normA += a[i] * a[i]
		normB += b[i] * b[i]
	}

	if normA == 0 || normB == 0 {
		return 0.0
	}

	return dotProduct / (math.Sqrt(normA) * math.Sqrt(normB))
}
