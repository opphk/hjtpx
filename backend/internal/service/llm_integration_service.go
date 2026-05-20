package service

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"math"
	"math/rand"
	"regexp"
	"strings"
	"sync"
	"time"
	"unicode"
)

type LLMIntegrationServiceV2 struct {
	mu                     sync.RWMutex
	initialized            bool
	nluVerifier            *NLUVerifier
	captchaGenerator       *LLMDrivenCaptchaGenerator
	dialogueSystem         *ConversationalVerificationSystem
	semanticRiskEvaluator  *SemanticRiskEvaluator
	cache                  *LLMCache
}

type NLUVerifier struct {
	mu          sync.RWMutex
	initialized bool
	models      *NLUModels
	vocabulary  map[string]int
	intentPatterns map[string][]*regexp.Regexp
}

type NLUModels struct {
	IntentClassifier *IntentClassifier
	EntityExtractor  *EntityExtractor
	SentimentAnalyzer *SentimentAnalyzer
	LanguageDetector *LanguageDetector
}

type IntentClassifier struct {
	ModelWeights []float64
	Intents     []string
}

type EntityExtractor struct {
	Patterns map[string]*regexp.Regexp
}

type SentimentAnalyzer struct {
	positiveWords map[string]float64
	negativeWords map[string]float64
	neutralWords  map[string]float64
}

type LanguageDetector struct {
	languageModels map[string]*LanguageModel
}

type LanguageModel struct {
	Language   string
	charFreq   map[rune]float64
	digrams    map[string]float64
	trigrams   map[string]float64
}

type LLMDrivenCaptchaGenerator struct {
	mu              sync.RWMutex
	initialized     bool
	themes          []CaptchaTheme
	questionTypes   []QuestionType
	difficultyLevels map[int]*LLMDifficultyConfig
	templateEngine  *TemplateEngine
	rng             *rand.Rand
}

type CaptchaTheme string

const (
	ThemeNature    CaptchaTheme = "nature"
	ThemeCity      CaptchaTheme = "city"
	ThemeScience   CaptchaTheme = "science"
	ThemeArt       CaptchaTheme = "art"
	ThemeHistory   CaptchaTheme = "history"
	ThemeSports    CaptchaTheme = "sports"
)

type QuestionType string

const (
	TypeMathProblem    QuestionType = "math_problem"
	TypeLogicPuzzle    QuestionType = "logic_puzzle"
	TypeTextCompletion QuestionType = "text_completion"
	TypeImageSelect    QuestionType = "image_select"
	TypePatternMatch   QuestionType = "pattern_match"
	TypeWordPuzzle     QuestionType = "word_puzzle"
)

type LLMDifficultyConfig struct {
	Level              int
	Complexity         float64
	TimeLimit          time.Duration
	MaxOptions         int
	RequiresReasoning  bool
}

type TemplateEngine struct {
	templates map[QuestionType][]*CaptchaTemplate
}

type CaptchaTemplate struct {
	TemplateID   string
	QuestionType QuestionType
	Theme        CaptchaTheme
	Prompt       string
	GenerateFn   func(rng *rand.Rand, difficulty int) (*GeneratedCaptcha, error)
}

type GeneratedCaptcha struct {
	ID              string
	Question        string
	Options         []string
	CorrectAnswer   string
	AnswerHash      string
	Hint            string
	Difficulty      int
	Theme           CaptchaTheme
	Type            QuestionType
	Metadata        map[string]interface{}
	GeneratedAt     time.Time
	ExpiresAt       time.Time
}

type ConversationalVerificationSystem struct {
	mu           sync.RWMutex
	initialized  bool
	sessions     map[string]*ConversationSession
	contexts     map[string]*VerificationContextV2
	llmModel     *ConversationalLLM
	maxTurns     int
}

type ConversationSession struct {
	SessionID      string
	UserID         string
	Messages       []ConversationMessage
	CurrentStep    int
	MaxTurns       int
	Status         string
	StartedAt      time.Time
	LastMessageAt  time.Time
	Context        *VerificationContextV2
}

type ConversationMessage struct {
	Role       string
	Content    string
	Timestamp  time.Time
	Intent     string
	Entities   map[string]string
	Confidence float64
}

type VerificationContextV2 struct {
	TaskType        string
	TaskDescription string
	Requirements    []string
	SuccessCriteria string
	Hints           []string
	Progress        float64
}

type ConversationalLLM struct {
	modelName    string
	temperature  float64
	maxTokens    int
	topP         float64
	presencePenalty float64
	frequencyPenalty float64
}

type SemanticRiskEvaluator struct {
	mu           sync.RWMutex
	initialized  bool
	riskPatterns map[string]*RiskPattern
	models       *RiskModels
	thresholds   *LLMRiskThresholds
}

type RiskPattern struct {
	PatternID    string
	Pattern       *regexp.Regexp
	Severity     float64
	Category     string
	Description  string
	Action       string
}

type RiskModels struct {
	toxicityModel    *ToxicityModel
	spamDetector     *SpamDetector
	phishingDetector *PhishingDetector
	insultDetector   *InsultDetector
}

type ToxicityModel struct {
	weights      []float64
	toxicTokens  map[string]float64
	thresholds   map[string]float64
}

type SpamDetector struct {
	spamPatterns []*regexp.Regexp
	spamWords    map[string]float64
	hamWords     map[string]float64
}

type PhishingDetector struct {
	phishingPatterns []*regexp.Regexp
	suspiciousTLDs   map[string]float64
	legitimateDomains map[string]bool
}

type InsultDetector struct {
	insultPatterns []*regexp.Regexp
	profanityList  map[string]float64
}

type LLMRiskThresholds struct {
	Toxicity     float64
	Spam         float64
	Phishing     float64
	Insult       float64
	Overall      float64
}

type LLMCache struct {
	mu           sync.RWMutex
	entries      map[string]*LLMCacheEntry
	maxSize      int
	currentSize  int
	evictionPolicy string
}

type LLMCacheEntry struct {
	Key        string
	Value      interface{}
	CreatedAt  time.Time
	AccessedAt time.Time
	TTL        time.Duration
	Frequency  int
}

type NLUVerificationRequest struct {
	Text      string                 `json:"text"`
	Language  string                 `json:"language,omitempty"`
	IntentRequired string            `json:"intent_required,omitempty"`
	Context   map[string]interface{} `json:"context,omitempty"`
}

type NLUVerificationResponse struct {
	Success         bool                  `json:"success"`
	Text            string                 `json:"text"`
	DetectedLanguage string               `json:"detected_language"`
	Intent          string                 `json:"intent"`
	IntentConfidence float64               `json:"intent_confidence"`
	Entities        map[string]string      `json:"entities"`
	Sentiment       string                 `json:"sentiment"`
	SentimentScore  float64               `json:"sentiment_score"`
	IsNatural       bool                   `json:"is_natural"`
	Details         map[string]interface{} `json:"details"`
}

type CaptchaGenerationRequest struct {
	Scene       string        `json:"scene"`
	Difficulty  int           `json:"difficulty"`
	Theme       CaptchaTheme  `json:"theme,omitempty"`
	QuestionType QuestionType `json:"question_type,omitempty"`
	Count       int           `json:"count"`
}

type CaptchaGenerationResponse struct {
	Success bool               `json:"success"`
	Captchas []*GeneratedCaptcha `json:"captchas"`
	Total   int                `json:"total"`
}

type DialogueRequest struct {
	SessionID string `json:"session_id"`
	UserID    string `json:"user_id"`
	Message   string `json:"message"`
}

type DialogueResponse struct {
	Success      bool        `json:"success"`
	SessionID    string      `json:"session_id"`
	Response     string      `json:"response"`
	Message      *ConversationMessage `json:"message"`
	NextPrompts  []string    `json:"next_prompts,omitempty"`
	Progress     float64     `json:"progress"`
	IsComplete   bool        `json:"is_complete"`
}

type SemanticRiskRequest struct {
	Text      string `json:"text"`
	Context   string `json:"context,omitempty"`
	StrictMode bool  `json:"strict_mode"`
}

type SemanticRiskResponse struct {
	Success      bool                    `json:"success"`
	OverallScore float64                 `json:"overall_score"`
	RiskLevel    string                  `json:"risk_level"`
	Categories   map[string]float64      `json:"categories"`
	DetectedRisks []DetectedRisk         `json:"detected_risks"`
	IsAllowed    bool                    `json:"is_allowed"`
	Details      map[string]interface{}   `json:"details"`
}

type DetectedRisk struct {
	Category    string  `json:"category"`
	Severity    float64 `json:"severity"`
	Description string  `json:"description"`
	MatchedText string  `json:"matched_text"`
	Action      string  `json:"action"`
}

func NewLLMIntegrationServiceV2() *LLMIntegrationServiceV2 {
	return &LLMIntegrationServiceV2{
		nluVerifier:           NewNLUVerifier(),
		captchaGenerator:      NewLLMDrivenCaptchaGenerator(),
		dialogueSystem:        NewConversationalVerificationSystem(),
		semanticRiskEvaluator: NewSemanticRiskEvaluator(),
		cache:                NewLLMCache(1000),
	}
}

func (s *LLMIntegrationServiceV2) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	if s.initialized {
		return nil
	}

	if err := s.nluVerifier.Initialize(ctx); err != nil {
		return err
	}

	if err := s.captchaGenerator.Initialize(ctx); err != nil {
		return err
	}

	if err := s.dialogueSystem.Initialize(ctx); err != nil {
		return err
	}

	if err := s.semanticRiskEvaluator.Initialize(ctx); err != nil {
		return err
	}

	s.initialized = true
	return nil
}

func NewNLUVerifier() *NLUVerifier {
	return &NLUVerifier{
		vocabulary:    make(map[string]int),
		intentPatterns: make(map[string][]*regexp.Regexp),
	}
}

func (v *NLUVerifier) Initialize(ctx context.Context) error {
	v.mu.Lock()
	defer v.mu.Unlock()

	v.models = &NLUModels{
		IntentClassifier: &IntentClassifier{
			ModelWeights: generateModelWeights(100),
			Intents:      []string{"greeting", "query", "command", "statement", "question"},
		},
		EntityExtractor: &EntityExtractor{
			Patterns: map[string]*regexp.Regexp{
				"email": regexp.MustCompile(`[a-zA-Z0-9._%+-]+@[a-zA-Z0-9.-]+\.[a-zA-Z]{2,}`),
				"url":   regexp.MustCompile(`https?://[^\s]+`),
				"phone": regexp.MustCompile(`\d{3}-\d{3}-\d{4}`),
			},
		},
		SentimentAnalyzer: newSentimentAnalyzer(),
		LanguageDetector:  newLanguageDetector(),
	}

	v.intentPatterns["greeting"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^hi|hello|hey|good (morning|afternoon|evening)`),
		regexp.MustCompile(`(?i)^howdy|greetings|sup`),
	}

	v.intentPatterns["query"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(what|who|where|when|why|how)\s`),
		regexp.MustCompile(`(?i)\?(.*)`),
	}

	v.intentPatterns["command"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i)^(please|can you|could you)\s`),
		regexp.MustCompile(`(?i)^(do|start|stop|create|delete)\s`),
	}

	v.intentPatterns["question"] = []*regexp.Regexp{
		regexp.MustCompile(`(?i).*\?$`),
	}

	v.vocabulary["the"] = 1000
	v.vocabulary["is"] = 900
	v.vocabulary["are"] = 850
	v.vocabulary["you"] = 800
	v.vocabulary["what"] = 750
	v.vocabulary["how"] = 700
	v.vocabulary["can"] = 650
	v.vocabulary["help"] = 600

	v.initialized = true
	return nil
}

func generateModelWeights(size int) []float64 {
	weights := make([]float64, size)
	for i := range weights {
		weights[i] = (float64(i%10) - 5.0) * 0.1
	}
	return weights
}

func newSentimentAnalyzer() *SentimentAnalyzer {
	return &SentimentAnalyzer{
		positiveWords: map[string]float64{
			"good": 0.8, "great": 0.9, "excellent": 1.0, "amazing": 0.95,
			"wonderful": 0.95, "fantastic": 0.9, "love": 0.85, "happy": 0.8,
			"beautiful": 0.85, "perfect": 1.0, "nice": 0.7, "awesome": 0.9,
		},
		negativeWords: map[string]float64{
			"bad": -0.8, "terrible": -1.0, "awful": -0.9, "hate": -0.95,
			"worst": -1.0, "horrible": -0.95, "sad": -0.7, "angry": -0.8,
			"ugly": -0.8, "fail": -0.7, "wrong": -0.6, "poor": -0.6,
		},
		neutralWords: map[string]float64{
			"the": 0.0, "is": 0.0, "are": 0.0, "a": 0.0, "an": 0.0,
			"this": 0.0, "that": 0.0, "it": 0.0, "they": 0.0,
		},
	}
}

func newLanguageDetector() *LanguageDetector {
	return &LanguageDetector{
		languageModels: map[string]*LanguageModel{
			"en": {
				Language: "English",
				charFreq: map[rune]float64{'e': 0.12, 't': 0.09, 'a': 0.08, 'o': 0.07, 'i': 0.07},
				digrams:  map[string]float64{"th": 0.03, "he": 0.025, "in": 0.02, "er": 0.02},
			},
			"zh": {
				Language: "Chinese",
				charFreq: map[rune]float64{'的': 0.05, '是': 0.04, '在': 0.03, '了': 0.03},
				digrams:  map[string]float64{"是的": 0.01, "在这": 0.008},
			},
		},
	}
}

func (v *NLUVerifier) Verify(ctx context.Context, request *NLUVerificationRequest) (*NLUVerificationResponse, error) {
	v.mu.RLock()
	defer v.mu.RUnlock()

	if !v.initialized {
		return nil, fmt.Errorf("NLU verifier not initialized")
	}

	text := strings.TrimSpace(request.Text)
	if text == "" {
		return &NLUVerificationResponse{
			Success: false,
			Text:    text,
		}, nil
	}

	detectedLang := v.detectLanguage(text)
	intent := v.classifyIntent(text)
	intentConfidence := v.calculateIntentConfidence(text, intent)
	entities := v.extractEntities(text)
	sentiment, sentimentScore := v.analyzeSentiment(text)
	isNatural := v.checkNaturalness(text)

	return &NLUVerificationResponse{
		Success:           true,
		Text:              text,
		DetectedLanguage:  detectedLang,
		Intent:            intent,
		IntentConfidence:  intentConfidence,
		Entities:          entities,
		Sentiment:         sentiment,
		SentimentScore:    sentimentScore,
		IsNatural:        isNatural,
		Details: map[string]interface{}{
			"word_count":     len(strings.Fields(text)),
			"char_count":      len(text),
			"avg_word_length": v.calculateAvgWordLength(text),
		},
	}, nil
}

func (v *NLUVerifier) detectLanguage(text string) string {
	if v.models == nil || v.models.LanguageDetector == nil {
		return "en"
	}

	chars := []rune(text)
	scores := make(map[string]float64)

	for lang, model := range v.models.LanguageDetector.languageModels {
		score := 0.0
		for _, c := range chars {
			if freq, ok := model.charFreq[c]; ok {
				score += freq
			}
		}
		scores[lang] = score
	}

	bestLang := "en"
	bestScore := 0.0
	for lang, score := range scores {
		if score > bestScore {
			bestScore = score
			bestLang = lang
		}
	}

	return bestLang
}

func (v *NLUVerifier) classifyIntent(text string) string {
	lowerText := strings.ToLower(text)

	for intent, patterns := range v.intentPatterns {
		for _, pattern := range patterns {
			if pattern.MatchString(lowerText) {
				return intent
			}
		}
	}

	return "statement"
}

func (v *NLUVerifier) calculateIntentConfidence(text, intent string) float64 {
	words := strings.Fields(strings.ToLower(text))
	if len(words) == 0 {
		return 0.0
	}

	knownWords := 0
	for _, word := range words {
		if _, ok := v.vocabulary[word]; ok {
			knownWords++
		}
	}

	return float64(knownWords) / float64(len(words))
}

func (v *NLUVerifier) extractEntities(text string) map[string]string {
	entities := make(map[string]string)

	if v.models == nil || v.models.EntityExtractor == nil {
		return entities
	}

	for entityType, pattern := range v.models.EntityExtractor.Patterns {
		matches := pattern.FindStringSubmatch(text)
		if len(matches) > 1 {
			entities[entityType] = matches[1]
		}
	}

	return entities
}

func (v *NLUVerifier) analyzeSentiment(text string) (string, float64) {
	if v.models == nil || v.models.SentimentAnalyzer == nil {
		return "neutral", 0.0
	}

	analyzer := v.models.SentimentAnalyzer
	words := strings.Fields(strings.ToLower(text))

	score := 0.0
	count := 0

	for _, word := range words {
		cleanWord := strings.TrimFunc(word, func(r rune) bool {
			return !unicode.IsLetter(r)
		})

		if val, ok := analyzer.positiveWords[cleanWord]; ok {
			score += val
			count++
		} else if val, ok := analyzer.negativeWords[cleanWord]; ok {
			score += val
			count++
		}
	}

	if count == 0 {
		return "neutral", 0.0
	}

	avgScore := score / float64(count)

	if avgScore > 0.2 {
		return "positive", avgScore
	} else if avgScore < -0.2 {
		return "negative", avgScore
	}

	return "neutral", avgScore
}

func (v *NLUVerifier) checkNaturalness(text string) bool {
	words := strings.Fields(text)

	if len(words) < 2 {
		return false
	}

	avgLen := 0.0
	for _, word := range words {
		avgLen += float64(len(word))
	}
	avgLen /= float64(len(words))

	if avgLen < 2.0 || avgLen > 15.0 {
		return false
	}

	upperCount := 0
	for _, c := range text {
		if unicode.IsUpper(c) {
			upperCount++
		}
	}

	upperRatio := float64(upperCount) / float64(len(text))
	if upperRatio > 0.5 {
		return false
	}

	return true
}

func (v *NLUVerifier) calculateAvgWordLength(text string) float64 {
	words := strings.Fields(text)
	if len(words) == 0 {
		return 0.0
	}

	total := 0
	for _, word := range words {
		total += len(word)
	}

	return float64(total) / float64(len(words))
}

func NewLLMDrivenCaptchaGenerator() *LLMDrivenCaptchaGenerator {
	return &LLMDrivenCaptchaGenerator{
		themes: []CaptchaTheme{
			ThemeNature, ThemeCity, ThemeScience, ThemeArt, ThemeHistory, ThemeSports,
		},
		questionTypes: []QuestionType{
			TypeMathProblem, TypeLogicPuzzle, TypeTextCompletion,
			TypeImageSelect, TypePatternMatch, TypeWordPuzzle,
		},
		difficultyLevels: map[int]*LLMDifficultyConfig{
			1: {Level: 1, Complexity: 0.3, TimeLimit: 60 * time.Second, MaxOptions: 3, RequiresReasoning: false},
			2: {Level: 2, Complexity: 0.5, TimeLimit: 45 * time.Second, MaxOptions: 4, RequiresReasoning: true},
			3: {Level: 3, Complexity: 0.7, TimeLimit: 30 * time.Second, MaxOptions: 4, RequiresReasoning: true},
			4: {Level: 4, Complexity: 0.85, TimeLimit: 20 * time.Second, MaxOptions: 5, RequiresReasoning: true},
			5: {Level: 5, Complexity: 1.0, TimeLimit: 15 * time.Second, MaxOptions: 5, RequiresReasoning: true},
		},
		templateEngine: NewTemplateEngine(),
		rng:            rand.New(rand.NewSource(time.Now().UnixNano())),
	}
}

func NewTemplateEngine() *TemplateEngine {
	return &TemplateEngine{
		templates: map[QuestionType][]*CaptchaTemplate{
			TypeMathProblem: {
				{TemplateID: "math_001", QuestionType: TypeMathProblem, Theme: ThemeScience},
				{TemplateID: "math_002", QuestionType: TypeMathProblem, Theme: ThemeNature},
			},
			TypeLogicPuzzle: {
				{TemplateID: "logic_001", QuestionType: TypeLogicPuzzle, Theme: ThemeScience},
				{TemplateID: "logic_002", QuestionType: TypeLogicPuzzle, Theme: ThemeArt},
			},
			TypeWordPuzzle: {
				{TemplateID: "word_001", QuestionType: TypeWordPuzzle, Theme: ThemeNature},
				{TemplateID: "word_002", QuestionType: TypeWordPuzzle, Theme: ThemeHistory},
			},
		},
	}
}

func (g *LLMDrivenCaptchaGenerator) Initialize(ctx context.Context) error {
	g.mu.Lock()
	defer g.mu.Unlock()

	g.templateEngine.templates[TypeMathProblem] = append(
		g.templateEngine.templates[TypeMathProblem],
		&CaptchaTemplate{TemplateID: "math_003", QuestionType: TypeMathProblem, Theme: ThemeCity},
	)

	g.initialized = true
	return nil
}

func (g *LLMDrivenCaptchaGenerator) GenerateCaptcha(ctx context.Context, request *CaptchaGenerationRequest) (*CaptchaGenerationResponse, error) {
	g.mu.RLock()
	defer g.mu.RUnlock()

	if !g.initialized {
		return nil, fmt.Errorf("captcha generator not initialized")
	}

	count := request.Count
	if count <= 0 {
		count = 1
	}

	captchas := make([]*GeneratedCaptcha, 0, count)

	for i := 0; i < count; i++ {
		captcha, err := g.generateSingleCaptcha(request)
		if err != nil {
			continue
		}
		captchas = append(captchas, captcha)
	}

	return &CaptchaGenerationResponse{
		Success: len(captchas) > 0,
		Captchas: captchas,
		Total:   len(captchas),
	}, nil
}

func (g *LLMDrivenCaptchaGenerator) generateSingleCaptcha(request *CaptchaGenerationRequest) (*GeneratedCaptcha, error) {
	var questionType QuestionType
	if request.QuestionType != "" {
		questionType = request.QuestionType
	} else {
		questionType = g.questionTypes[g.rng.Intn(len(g.questionTypes))]
	}

	var theme CaptchaTheme
	if request.Theme != "" {
		theme = request.Theme
	} else {
		theme = g.themes[g.rng.Intn(len(g.themes))]
	}

	difficulty := request.Difficulty
	if difficulty < 1 || difficulty > 5 {
		difficulty = 2
	}

	var question, answer string
	var options []string

	switch questionType {
	case TypeMathProblem:
		question, options, answer = g.generateMathProblem(difficulty)
	case TypeLogicPuzzle:
		question, options, answer = g.generateLogicPuzzle(difficulty)
	case TypeWordPuzzle:
		question, options, answer = g.generateWordPuzzle(theme, difficulty)
	default:
		question, options, answer = g.generateMathProblem(difficulty)
	}

	captchaID := generateCaptchaID()
	hint := g.generateHint(difficulty)

	return &GeneratedCaptcha{
		ID:            captchaID,
		Question:      question,
		Options:       options,
		CorrectAnswer: answer,
		AnswerHash:    hashAnswer(answer),
		Hint:          hint,
		Difficulty:    difficulty,
		Theme:         theme,
		Type:          questionType,
		Metadata: map[string]interface{}{
			"scene":    request.Scene,
			"template": fmt.Sprintf("%s_%s", questionType, theme),
		},
		GeneratedAt: time.Now(),
		ExpiresAt:    time.Now().Add(5 * time.Minute),
	}, nil
}

func (g *LLMDrivenCaptchaGenerator) generateMathProblem(difficulty int) (string, []string, string) {
	ops := []string{"+", "-", "×", "÷"}
	op := ops[g.rng.Intn(len(ops))]

	maxNum := 10 + difficulty*10
	minNum := 1

	a := minNum + g.rng.Intn(maxNum-minNum)
	b := minNum + g.rng.Intn(maxNum-minNum)

	var result int
	var question string

	switch op {
	case "+":
		result = a + b
		question = fmt.Sprintf("%d + %d = ?", a, b)
	case "-":
		if a < b {
			a, b = b, a
		}
		result = a - b
		question = fmt.Sprintf("%d - %d = ?", a, b)
	case "×":
		a = minNum + g.rng.Intn(12-minNum)
		b = minNum + g.rng.Intn(12-minNum)
		result = a * b
		question = fmt.Sprintf("%d × %d = ?", a, b)
	case "÷":
		b = minNum + g.rng.Intn(10-minNum)
		result = minNum + g.rng.Intn(10-minNum)
		a = b * result
		question = fmt.Sprintf("%d ÷ %d = ?", a, b)
	}

	options := g.generateOptions(result, 4)
	return question, options, fmt.Sprintf("%d", result)
}

func (g *LLMDrivenCaptchaGenerator) generateLogicPuzzle(difficulty int) (string, []string, string) {
	puzzles := []struct {
		q string
		a string
		o []string
	}{
		{
			q: "If all cats are animals, and all animals need water, what can we conclude?",
			a: "All cats need water",
			o: []string{"All cats need water", "Some cats don't need water", "No animals are cats", "All animals are cats"},
		},
		{
			q: "What number comes next: 2, 4, 8, 16, ?",
			a: "32",
			o: []string{"32", "24", "20", "30"},
		},
		{
			q: "Tom is taller than Jim. Jim is taller than Sam. Who is the shortest?",
			a: "Sam",
			o: []string{"Sam", "Tom", "Jim", "All same height"},
		},
	}

	puzzle := puzzles[g.rng.Intn(len(puzzles))]
	return puzzle.q, puzzle.o, puzzle.a
}

func (g *LLMDrivenCaptchaGenerator) generateWordPuzzle(theme CaptchaTheme, difficulty int) (string, []string, string) {
	wordBanks := map[CaptchaTheme][]string{
		ThemeNature:  {"tree", "river", "mountain", "ocean", "forest", "cloud", "flower", "animal"},
		ThemeCity:    {"street", "building", "park", "bridge", "station", "museum", "library", "tower"},
		ThemeHistory: {"king", "queen", "empire", "war", "peace", "nation", "ancient", "dynasty"},
		ThemeSports:  {"football", "tennis", "basketball", "swimming", "running", "cycling", "hockey", "golf"},
		ThemeScience: {"atom", "planet", "energy", "gravity", "oxygen", "molecule", "electron", "nucleus"},
		ThemeArt:     {"painting", "sculpture", "music", "dance", "poetry", "theater", "drawing", "photography"},
	}

	words := wordBanks[theme]
	if len(words) == 0 {
		words = wordBanks[ThemeNature]
	}

	word := words[g.rng.Intn(len(words))]
	chars := []rune(word)
	for i := range chars {
		j := g.rng.Intn(i + 1)
		chars[i], chars[j] = chars[j], chars[i]
	}

	question := fmt.Sprintf("Unscramble: %s", string(chars))

	options := make([]string, 0, 4)
	options = append(options, word)

	otherWords := make([]string, 0)
	for _, w := range words {
		if w != word {
			otherWords = append(otherWords, w)
		}
	}

	for len(options) < 4 && len(otherWords) > 0 {
		idx := g.rng.Intn(len(otherWords))
		options = append(options, otherWords[idx])
		otherWords = append(otherWords[:idx], otherWords[idx+1:]...)
	}

	return question, options, word
}

func (g *LLMDrivenCaptchaGenerator) generateOptions(correct, count int) []string {
	options := make([]string, 0, count)
	options = append(options, fmt.Sprintf("%d", correct))

	used := map[int]bool{correct: true}

	for len(options) < count {
		delta := -10 + g.rng.Intn(21)
		wrong := correct + delta
		if wrong >= 0 && !used[wrong] {
			used[wrong] = true
			options = append(options, fmt.Sprintf("%d", wrong))
		}
	}

	for i := range options {
		j := g.rng.Intn(i + 1)
		options[i], options[j] = options[j], options[i]
	}

	return options
}

func (g *LLMDrivenCaptchaGenerator) generateHint(difficulty int) string {
	hints := map[int][]string{
		1: {"Take your time!", "This is easy!"},
		2: {"Think about it", "Consider all options"},
		3: {"Focus carefully", "Read the question twice"},
		4: {"This requires thought", "Think step by step"},
		5: {"Challenge yourself", "Break it down"},
	}

	hintList := hints[difficulty]
	return hintList[g.rng.Intn(len(hintList))]
}

func NewConversationalVerificationSystem() *ConversationalVerificationSystem {
	return &ConversationalVerificationSystem{
		sessions: make(map[string]*ConversationSession),
		contexts: make(map[string]*VerificationContextV2),
		maxTurns: 10,
	}
}

func (s *ConversationalVerificationSystem) Initialize(ctx context.Context) error {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.llmModel = &ConversationalLLM{
		modelName:        "conversational-v2",
		temperature:      0.7,
		maxTokens:        200,
		topP:             0.9,
		presencePenalty:  0.1,
		frequencyPenalty: 0.1,
	}

	s.initialized = true
	return nil
}

func (s *ConversationalVerificationSystem) ProcessMessage(ctx context.Context, request *DialogueRequest) (*DialogueResponse, error) {
	s.mu.Lock()
	defer s.mu.Unlock()

	if !s.initialized {
		return nil, fmt.Errorf("dialogue system not initialized")
	}

	session, exists := s.sessions[request.SessionID]
	if !exists {
		session = s.createSession(request.SessionID, request.UserID)
		s.sessions[request.SessionID] = session
	}

	session.LastMessageAt = time.Now()
	session.Messages = append(session.Messages, ConversationMessage{
		Role:       "user",
		Content:    request.Message,
		Timestamp:  time.Now(),
	})

	session.CurrentStep++

	response := s.generateResponse(session)

	session.Messages = append(session.Messages, ConversationMessage{
		Role:       "assistant",
		Content:    response,
		Timestamp:  time.Now(),
	})

	isComplete := session.CurrentStep >= session.MaxTurns

	return &DialogueResponse{
		Success:    true,
		SessionID:  request.SessionID,
		Response:   response,
		Message: &ConversationMessage{
			Role:       "assistant",
			Content:    response,
			Timestamp:  time.Now(),
		},
		NextPrompts: s.generateNextPrompts(session),
		Progress:    float64(session.CurrentStep) / float64(session.MaxTurns),
		IsComplete: isComplete,
	}, nil
}

func (s *ConversationalVerificationSystem) createSession(sessionID, userID string) *ConversationSession {
	return &ConversationSession{
		SessionID:   sessionID,
		UserID:     userID,
		Messages:   make([]ConversationMessage, 0),
		CurrentStep: 0,
		MaxTurns:   s.maxTurns,
		Status:     "active",
		StartedAt:  time.Now(),
		Context: &VerificationContextV2{
			TaskType:     "verification",
			Requirements: []string{},
			Progress:     0.0,
		},
	}
}

func (s *ConversationalVerificationSystem) generateResponse(session *ConversationSession) string {
	if session.CurrentStep == 1 {
		return "Hello! I'm here to help you with verification. Can you tell me your name?"
	}

	lastMessage := ""
	if len(session.Messages) > 0 {
		lastMessage = session.Messages[len(session.Messages)-1].Content
	}

	if strings.Contains(strings.ToLower(lastMessage), "name") {
		return "Nice to meet you! What would you like to verify today?"
	}

	questions := []string{
		"Could you please provide your email address?",
		"What is your purpose for using this service?",
		"Do you have any specific requirements?",
		"How did you hear about our service?",
		"Is there anything else you'd like to know?",
	}

	step := session.CurrentStep - 1
	if step < len(questions) {
		return questions[step]
	}

	return "Thank you for your responses. Verification complete!"
}

func (s *ConversationalVerificationSystem) generateNextPrompts(session *ConversationSession) []string {
	remaining := session.MaxTurns - session.CurrentStep
	if remaining <= 0 {
		return []string{}
	}

	return []string{
		"Continue",
		"Skip this step",
		"Get help",
		"Cancel",
	}
}

func NewSemanticRiskEvaluator() *SemanticRiskEvaluator {
	return &SemanticRiskEvaluator{
		riskPatterns: make(map[string]*RiskPattern),
		thresholds: &LLMRiskThresholds{
			Toxicity:   0.7,
			Spam:       0.6,
			Phishing:   0.8,
			Insult:     0.6,
			Overall:    0.7,
		},
	}
}

func (e *SemanticRiskEvaluator) Initialize(ctx context.Context) error {
	e.mu.Lock()
	defer e.mu.Unlock()

	e.models = &RiskModels{
		toxicityModel: &ToxicityModel{
			weights:     generateModelWeights(100),
			toxicTokens:  map[string]float64{"hate": 0.9, "kill": 0.95, "die": 0.8, "attack": 0.85},
			thresholds:  map[string]float64{"high": 0.8, "medium": 0.5, "low": 0.3},
		},
		spamDetector: &SpamDetector{
			spamPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)buy now|limited offer|click here|winner`),
				regexp.MustCompile(`(?i)free money|get rich|act now`),
			},
			spamWords: map[string]float64{
				"viagra": 0.9, "lottery": 0.8, "winner": 0.7, "prize": 0.7,
				"click": 0.5, "buy": 0.4, "offer": 0.5, "discount": 0.4,
			},
			hamWords: map[string]float64{
				"meeting": -0.3, "project": -0.2, "report": -0.2, "update": -0.1,
			},
		},
		phishingDetector: &PhishingDetector{
			phishingPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)urgent|account.*suspended|verify.*password`),
				regexp.MustCompile(`(?i)confirm.*identity|security.*alert`),
			},
			suspiciousTLDs: map[string]float64{
				".xyz": 0.7, ".top": 0.7, ".work": 0.8, ".click": 0.6,
			},
			legitimateDomains: map[string]bool{
				"google.com": true, "microsoft.com": true, "apple.com": true,
			},
		},
		insultDetector: &InsultDetector{
			insultPatterns: []*regexp.Regexp{
				regexp.MustCompile(`(?i)\b(idiot|stupid|dumb|loser|fool)\b`),
			},
			profanityList: map[string]float64{
				"damn": 0.5, "hell": 0.4, "crap": 0.3,
			},
		},
	}

	e.riskPatterns["toxicity"] = &RiskPattern{
		PatternID:   "toxicity_001",
		Pattern:     regexp.MustCompile(`(?i)(hate|kill|attack|threat)`),
		Severity:   0.9,
		Category:   "toxicity",
		Action:     "block",
	}

	e.riskPatterns["spam"] = &RiskPattern{
		PatternID:   "spam_001",
		Pattern:     regexp.MustCompile(`(?i)(buy now|free|winner|prize)`),
		Severity:   0.7,
		Category:   "spam",
		Action:     "review",
	}

	e.riskPatterns["phishing"] = &RiskPattern{
		PatternID:   "phishing_001",
		Pattern:     regexp.MustCompile(`(?i)(verify.*account|reset.*password|suspended)`),
		Severity:   0.85,
		Category:   "phishing",
		Action:     "block",
	}

	e.initialized = true
	return nil
}

func (e *SemanticRiskEvaluator) Evaluate(ctx context.Context, request *SemanticRiskRequest) (*SemanticRiskResponse, error) {
	e.mu.RLock()
	defer e.mu.RUnlock()

	if !e.initialized {
		return nil, fmt.Errorf("semantic risk evaluator not initialized")
	}

	categories := make(map[string]float64)
	detectedRisks := make([]DetectedRisk, 0)

	toxicityScore := e.evaluateToxicity(request.Text)
	categories["toxicity"] = toxicityScore

	spamScore := e.evaluateSpam(request.Text)
	categories["spam"] = spamScore

	phishingScore := e.evaluatePhishing(request.Text)
	categories["phishing"] = phishingScore

	insultScore := e.evaluateInsult(request.Text)
	categories["insult"] = insultScore

	for patternID, pattern := range e.riskPatterns {
		if pattern.Pattern.MatchString(request.Text) {
			detectedRisks = append(detectedRisks, DetectedRisk{
				Category:    pattern.Category,
				Severity:    pattern.Severity,
				Description: fmt.Sprintf("Risk pattern %s detected", patternID),
				MatchedText: pattern.Pattern.FindString(request.Text),
				Action:      pattern.Action,
			})
		}
	}

	overallScore := e.calculateOverallScore(categories)
	riskLevel := e.classifyRiskLevel(overallScore)
	isAllowed := overallScore < e.thresholds.Overall

	if request.StrictMode {
		isAllowed = isAllowed && !hasHighSeverityRisk(detectedRisks)
	}

	return &SemanticRiskResponse{
		Success:       true,
		OverallScore:  overallScore,
		RiskLevel:     riskLevel,
		Categories:    categories,
		DetectedRisks: detectedRisks,
		IsAllowed:     isAllowed,
		Details: map[string]interface{}{
			"text_length":  len(request.Text),
			"word_count":   len(strings.Fields(request.Text)),
			"strict_mode":  request.StrictMode,
		},
	}, nil
}

func (e *SemanticRiskEvaluator) evaluateToxicity(text string) float64 {
	if e.models == nil || e.models.toxicityModel == nil {
		return 0.0
	}

	model := e.models.toxicityModel
	score := 0.0
	count := 0

	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		if val, ok := model.toxicTokens[word]; ok {
			score += val
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return math.Min(score/float64(count), 1.0)
}

func (e *SemanticRiskEvaluator) evaluateSpam(text string) float64 {
	if e.models == nil || e.models.spamDetector == nil {
		return 0.0
	}

	detector := e.models.spamDetector
	score := 0.0

	if detector.spamPatterns != nil {
		for _, pattern := range detector.spamPatterns {
			if pattern.MatchString(text) {
				score += 0.3
			}
		}
	}

	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		if val, ok := detector.spamWords[word]; ok {
			score += val
		}
		if val, ok := detector.hamWords[word]; ok {
			score += val
		}
	}

	return math.Min(math.Max(score, 0.0), 1.0)
}

func (e *SemanticRiskEvaluator) evaluatePhishing(text string) float64 {
	if e.models == nil || e.models.phishingDetector == nil {
		return 0.0
	}

	detector := e.models.phishingDetector
	score := 0.0

	if detector.phishingPatterns != nil {
		for _, pattern := range detector.phishingPatterns {
			if pattern.MatchString(text) {
				score += 0.4
			}
		}
	}

	urlPattern := regexp.MustCompile(`https?://[^\s]+`)
	urls := urlPattern.FindAllString(text, -1)
	for _, url := range urls {
		for tld, val := range detector.suspiciousTLDs {
			if strings.HasSuffix(url, tld) {
				score += val
			}
		}
	}

	return math.Min(math.Max(score, 0.0), 1.0)
}

func (e *SemanticRiskEvaluator) evaluateInsult(text string) float64 {
	if e.models == nil || e.models.insultDetector == nil {
		return 0.0
	}

	detector := e.models.insultDetector
	score := 0.0
	count := 0

	if detector.insultPatterns != nil {
		for _, pattern := range detector.insultPatterns {
			matches := pattern.FindAllString(text, -1)
			score += float64(len(matches)) * 0.3
			count += len(matches)
		}
	}

	words := strings.Fields(strings.ToLower(text))
	for _, word := range words {
		if val, ok := detector.profanityList[word]; ok {
			score += val
			count++
		}
	}

	if count == 0 {
		return 0.0
	}

	return math.Min(math.Max(score/float64(count+1), 0.0), 1.0)
}

func (e *SemanticRiskEvaluator) calculateOverallScore(categories map[string]float64) float64 {
	if len(categories) == 0 {
		return 0.0
	}

	weights := map[string]float64{
		"toxicity": 0.3,
		"spam":    0.2,
		"phishing": 0.3,
		"insult":   0.2,
	}

	totalWeight := 0.0
	weightedSum := 0.0

	for cat, score := range categories {
		weight := weights[cat]
		if weight == 0 {
			weight = 1.0 / float64(len(categories))
		}
		weightedSum += score * weight
		totalWeight += weight
	}

	if totalWeight == 0 {
		return 0.0
	}

	return weightedSum / totalWeight
}

func (e *SemanticRiskEvaluator) classifyRiskLevel(score float64) string {
	if score < 0.3 {
		return "low"
	} else if score < 0.6 {
		return "medium"
	} else if score < 0.8 {
		return "high"
	}
	return "critical"
}

func NewLLMCache(maxSize int) *LLMCache {
	return &LLMCache{
		entries:       make(map[string]*LLMCacheEntry),
		maxSize:       maxSize,
		evictionPolicy: "lru",
	}
}

func (c *LLMCache) Get(key string) (interface{}, bool) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	entry, exists := c.entries[key]
	if exists {
		if time.Since(entry.CreatedAt) > entry.TTL {
			return nil, false
		}
		entry.AccessedAt = time.Now()
		entry.Frequency++
		return entry.Value, true
	}

	return nil, false
}

func (c *LLMCache) Set(key string, value interface{}, ttl time.Duration) {
	c.mu.Lock()
	defer c.mu.Unlock()

	entry := &LLMCacheEntry{
		Key:       key,
		Value:     value,
		CreatedAt: time.Now(),
		AccessedAt: time.Now(),
		TTL:       ttl,
		Frequency: 0,
	}

	c.entries[key] = entry
	c.currentSize++

	if c.currentSize > c.maxSize {
		c.evictLRU()
	}
}

func (c *LLMCache) evictLRU() {
	if len(c.entries) == 0 {
		return
	}

	var oldestKey string
	var oldestTime = time.Now()

	for key, entry := range c.entries {
		if entry.AccessedAt.Before(oldestTime) {
			oldestTime = entry.AccessedAt
			oldestKey = key
		}
	}

	if oldestKey != "" {
		delete(c.entries, oldestKey)
		c.currentSize--
	}
}

func generateCaptchaID() string {
	hash := sha256.Sum256([]byte(fmt.Sprintf("%d_%d", time.Now().UnixNano(), rand.Int63())))
	return hex.EncodeToString(hash[:])[:16]
}

func hashAnswer(answer string) string {
	hash := sha256.Sum256([]byte(answer))
	return hex.EncodeToString(hash[:])
}

func hasHighSeverityRisk(risks []DetectedRisk) bool {
	for _, risk := range risks {
		if risk.Severity >= 0.8 {
			return true
		}
	}
	return false
}

func (s *LLMIntegrationServiceV2) VerifyNaturalLanguage(ctx context.Context, request *NLUVerificationRequest) (*NLUVerificationResponse, error) {
	return s.nluVerifier.Verify(ctx, request)
}

func (s *LLMIntegrationServiceV2) GenerateLLMCaptcha(ctx context.Context, request *CaptchaGenerationRequest) (*CaptchaGenerationResponse, error) {
	return s.captchaGenerator.GenerateCaptcha(ctx, request)
}

func (s *LLMIntegrationServiceV2) ProcessDialogue(ctx context.Context, request *DialogueRequest) (*DialogueResponse, error) {
	return s.dialogueSystem.ProcessMessage(ctx, request)
}

func (s *LLMIntegrationServiceV2) EvaluateSemanticRisk(ctx context.Context, request *SemanticRiskRequest) (*SemanticRiskResponse, error) {
	return s.semanticRiskEvaluator.Evaluate(ctx, request)
}

func ParseNLUVerificationRequest(data string) (*NLUVerificationRequest, error) {
	var req NLUVerificationRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}

func ParseCaptchaGenerationRequest(data string) (*CaptchaGenerationRequest, error) {
	var req CaptchaGenerationRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}

func ParseDialogueRequest(data string) (*DialogueRequest, error) {
	var req DialogueRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}

func ParseSemanticRiskRequest(data string) (*SemanticRiskRequest, error) {
	var req SemanticRiskRequest
	err := json.Unmarshal([]byte(data), &req)
	return &req, err
}
